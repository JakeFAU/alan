package executor

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
)

func silentChildProvision(t *testing.T, bit int64, terminalRoute bool) Provision {
	t.Helper()
	receive := func(kind, provenance, source, obligation string, bind map[string]string) Instruction {
		return Instruction{Op: "receive", Match: &Match{Kind: kind, Provenance: provenance, Source: Lit(Text(source)),
			Generation: Lit(Int(1)), Obligation: Lit(Text(obligation)), Operation: Lit(Text(""))}, Bind: bind}
	}
	send := Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("child")),
		Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Ref("number")}}}
	input := inputProgram()[0] // Local input on obligation "work" binds number.
	p := Provision{Permissions: []Permission{
		{Kind: "message", Source: "p", Target: "child", Operation: "message", Obligation: "work", Revision: 1},
		{Kind: "input", Source: "local", Target: "p", Operation: "input", Obligation: "work"},
		{Kind: "input", Source: "local", Target: "p", Operation: "input", Obligation: "noise"},
	}}
	if terminalRoute {
		p.Permissions = append(p.Permissions, Permission{Kind: "observation", Source: "child", Target: "p", Operation: "terminal", Obligation: "done", Revision: 1})
	}
	addProgram(t, &p, "p", []Instruction{send, send, input, input, send,
		receive("terminal", "runtime-terminal", "child", "done", map[string]string{"outcome": "outcome"}),
		{Op: "branch", A: Ref("outcome"), B: Lit(Text("succeeded")), Target: 7, Otherwise: 8},
		{Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}, map[string]Value{"number": Int(0)})
	addProgram(t, &p, "child", []Instruction{receive("message", "ordinary-message", "p", "work", nil),
		{Op: "branch", A: Ref("private"), B: Lit(Int(0)), Target: 2, Otherwise: 3},
		{Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}, map[string]Value{"private": Int(bit)})
	return p
}

func admitParentNumber(t *testing.T, s *Store, id, obligation string, number int64) string {
	t.Helper()
	r, err := s.AdmitInput(t.Context(), Input{TransportID: id, LogicalID: id, Target: "p",
		Obligation: obligation, Payload: map[string]Value{"number": Int(number)}})
	if err != nil || !r.Accepted {
		t.Fatalf("admit %s: %+v, %v", id, r, err)
	}
	return r.ObservationID
}

type parentSchedulerCut struct {
	run     RunState
	inbox   map[string]retainedObservation
	intents map[string]retainedIntent
	enabled []string
}

// Probe Commit through its last precommit boundary, rolling back each successful
// probe. Try the empty selection and every retained ID, including foreign,
// consumed and guard-mismatched rows. Unexpected faults are distinct from waits.
func inspectParentSchedulerCut(t *testing.T, s *Store) parentSchedulerCut {
	t.Helper()
	cut := parentSchedulerCut{run: current(t, s, "p"), inbox: map[string]retainedObservation{}}
	observations, err := readObservations(t.Context(), s.conn, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	cut.intents, err = readIntents(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []string{""}
	for id, o := range observations {
		candidates = append(candidates, id)
		if o.Run == cut.run.ID {
			cut.inbox[id] = o
		}
	}
	staged := errors.New("parent commit staged")
	probe := boundary.WithObserver(t.Context(), func(_ context.Context, event boundary.Event) error {
		if event.Boundary == "execution.staged" {
			return staged
		}
		return nil
	})
	for _, id := range candidates {
		_, err := s.Commit(probe, cut.run.Authority(), id)
		switch {
		case errors.Is(err, staged):
			cut.enabled = append(cut.enabled, id)
		case errors.Is(err, ErrIneligible):
		default:
			t.Fatalf("probe at parent PC %d with %q: %v", cut.run.Projection.PC, id, err)
		}
	}
	slices.Sort(cut.enabled)
	return cut
}

func TestSilentChildParentSchedulerCuts(t *testing.T) {
	a, z := identity("observation", "input", "a"), identity("observation", "input", "z")
	local := []string{""}
	// At the first two competing cuts the cursor picks the child and then p.
	// Rejecting z only after child failure while a remains unconsumed preserves
	// every actual pick. The two-candidate check below detects that mutant.
	cuts := []struct {
		pick    string
		enabled []string // All p selections, including the one the cursor passes.
	}{{"p", local}, {"child", local}, {"p", local}, {"child", nil}, {"child", nil}, {"", nil},
		{"p", []string{a, z}}, {"p", []string{z}}, {"p", local}, {"", nil}}
	for _, profile := range []struct {
		name  string
		route bool
	}{{"no_terminal_route", false}, {"withheld_then_serviced_terminal", true}} {
		t.Run(profile.name, func(t *testing.T) {
			var success []parentSchedulerCut
			for bit, outcome := range []string{"succeeded", "failed"} {
				s, _ := messageStore(t, silentChildProvision(t, int64(bit), profile.route))
				admitParentNumber(t, s, "0", "noise", 99)
				var last *RunID
				for i, want := range cuts {
					if i == 6 { // After both the branch and End with a blocked parent.
						admitParentNumber(t, s, "z", "work", 22)
						admitParentNumber(t, s, "a", "work", 11)
					}
					cut := inspectParentSchedulerCut(t, s)
					if !slices.Equal(cut.enabled, want.enabled) {
						t.Fatalf("%s cut %d: enabled parent selections %q, want %q", outcome, i, cut.enabled, want.enabled)
					}
					if bit == 0 {
						success = append(success, cut)
					} else if !reflect.DeepEqual(success[i], cut) {
						t.Fatalf("cut %d: parent differs across child End states\nsuccess: %+v\nfailure: %+v", i, success[i], cut)
					}
					pick, err := s.selectReady(t.Context(), last)
					if err != nil || (pick == nil) != (want.pick == "") {
						t.Fatalf("%s cut %d: selector %+v, %v, want %q", outcome, i, pick, err, want.pick)
					}
					if pick == nil {
						continue
					}
					if pick.authority.Run.Subject != want.pick || want.pick == "p" && pick.observation != want.enabled[0] {
						t.Fatalf("%s cut %d: selector %+v, want %+v", outcome, i, pick, want)
					}
					next, err := s.Commit(t.Context(), pick.authority, pick.observation)
					if err != nil {
						t.Fatal(err)
					}
					last = &next.ID
					if next.ID.Subject == "p" && (next.Sequence <= 2 || next.Sequence == 5) {
						id := identity("p", 1, next.Sequence, 0)
						transport := identity("message-delivery", id)
						r, err := s.DeliverMessage(t.Context(), id, transport)
						if err != nil || r.Accepted != (next.Sequence <= 2) || !r.Accepted && r.Reason != "target_closed" {
							t.Fatalf("%s parent send %d: %+v, %v", outcome, next.Sequence, r, err)
						}
						if err := s.AcknowledgeMessage(t.Context(), id, transport); err != nil {
							t.Fatal(err)
						}
					}
				}
				if child := current(t, s, "child"); child.Sequence != 3 || child.Projection.Status != outcome {
					t.Fatalf("child End: %+v", child)
				}
				pending, err := s.pendingDeliveries(t.Context())
				if err != nil || (!profile.route && len(pending) != 0) ||
					(profile.route && (len(pending) != 1 || pending[0].source.Subject != "child")) {
					t.Fatalf("deliveries at blocked parent: %+v, %v", pending, err)
				}
				report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
				parent := current(t, s, "p")
				if err != nil || !report.Quiescent || parent.Projection.Data["number"] != Int(22) {
					t.Fatalf("final runner: %+v, %+v, %v", report, parent, err)
				}
				if profile.route {
					if report.Dispatches != 1 || report.Steps != 3 || parent.Projection.Data["outcome"] != Text(outcome) || parent.Projection.Status != outcome {
						t.Fatalf("serviced terminal: %+v, %+v", report, parent)
					}
				} else if report.Dispatches != 0 || report.Steps != 0 || parent.Projection.Status != "open" || parent.Projection.PC != 5 {
					t.Fatalf("no terminal route: %+v, %+v", report, parent)
				}
			}
		})
	}
}

func TestSilentChildEndProjectionLimit(t *testing.T) {
	for bit, outcome := range []string{"succeeded", "failed"} {
		p := silentChildProvision(t, int64(bit), false)
		// Both private digits have one JSON byte. The failed End's full
		// Projection fits exactly, while the success status costs three more.
		failed, err := json.Marshal(Projection{PC: 3, Status: "failed", Data: p.Runs[1].Projection.Data})
		if err != nil {
			t.Fatal(err)
		}
		limits := DefaultLimits()
		limits.MaxDataBytes = len(failed)
		s, err := Create(t.Context(), filepath.Join(t.TempDir(), "bounded.sqlite"), p, limits)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		admitParentNumber(t, s, "a", "work", 11)
		z := admitParentNumber(t, s, "z", "work", 22)
		report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
		child := current(t, s, "child")
		parent := inspectParentSchedulerCut(t, s)
		t.Logf("child target %s, projection cap %d: %d commits, child status %s, parent PC %d, enabled %q, Run error %v",
			outcome, limits.MaxDataBytes, report.Steps, child.Projection.Status, parent.run.Projection.PC, parent.enabled, err)
		if outcome == "succeeded" {
			if !errors.Is(err, ErrLimit) || report.Quiescent || report.Steps != 5 || child.Projection.Status != "open" ||
				child.Sequence != 2 || parent.run.Projection.PC != 3 || !slices.Equal(parent.enabled, []string{z}) {
				t.Fatalf("oversize success End: %+v, %v, child %+v, parent %+v", report, err, child, parent)
			}
		} else if err != nil || !report.Quiescent || report.Steps != 8 || child.Projection.Status != "failed" ||
			parent.run.Projection.PC != 5 || parent.inbox[z].consumed.Int64 != 4 || len(parent.enabled) != 0 {
			t.Fatalf("in-budget failed End: %+v, %v, child %+v, parent %+v", report, err, child, parent)
		}
	}
}
