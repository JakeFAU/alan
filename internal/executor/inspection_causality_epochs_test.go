package executor

import (
	"errors"
	"math"
	"path/filepath"
	"testing"
)

func TestInspectMissingMemberAcrossSharedReopen(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		existed, closeController bool
	}{
		{"member shared controller's activation", true, false},
		{"controller closed after its second activation", true, true},
		{"member created after the missing decision", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			p, profile := invocationFixture(t, "succeeded", 1)
			addRetirementController(t, &p, []string{"CancelRun"})
			path := filepath.Join(t.TempDir(), "shared-epoch.sqlite")
			s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if s != nil {
					_ = s.Close()
				}
			}()
			future := compositionMemberID(compositionInvocationID(s.Namespace(), identity("root", 1, 1, 0)), "entry")
			const source = "retirement-controller"
			a, err := s.AdmitInput(ctx, Input{TransportID: "t", LogicalID: "t", Target: source, Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(future.Subject)}})
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
			if _, err = s.Commit(ctx, current(t, s, source).Authority(), a.ObservationID); err != nil {
				t.Fatal(err)
			}
			req := committedRequest(t, s, source)
			if tc.existed && !invokeOnce(t, s).Accepted {
				t.Fatal("member not admitted")
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if s, err = Open(ctx, path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			d := authorizedDecision(t, s, req)
			if tc.closeController {
				if _, err = s.Commit(ctx, current(t, s, source).Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			if !tc.existed {
				if d.Reason != "target_missing" || !invokeOnce(t, s).Accepted || mustSnapshot(t, s, future).Epoch != 1 {
					t.Fatal(d)
				}
				if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
					t.Fatal("genuine absence across reopen", err)
				}
				// Simulate many later opens of these still-open generations. The
				// member remains one epoch younger than the original source.
				if _, err := s.conn.ExecContext(ctx, "UPDATE runs SET epoch=epoch+?", int64(math.MaxInt64-2)); err != nil {
					t.Fatal(err)
				}
				if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
					t.Fatal("sparse feasible maximum epoch", err)
				}
				if _, err := s.conn.ExecContext(ctx, "UPDATE runs SET epoch=? WHERE subject=?", int64(math.MaxInt64), future.Subject); err != nil {
					t.Fatal(err)
				}
				if cut, err := InspectExecution(ctx, path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
					t.Fatal("maximum epoch hid a shared-activation contradiction", err)
				}
				return
			}
			if d.Before == nil || d.Before.Epoch != 2 || mustSnapshot(t, s, future).Epoch != 2 {
				t.Fatal(d)
			}
			if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
				t.Fatal("genuine witnessed decision", err)
			}
			d.Before, d.After, d.Reason = nil, nil, "target_missing"
			if _, err := s.conn.ExecContext(ctx, "UPDATE lifecycle_results SET result=CAST(? AS BLOB) WHERE id=?", compositionJSON(d), req.Intent.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(ctx, "DELETE FROM composition_facts WHERE id=?", compositionIdentity("alan/fact/lifecycle/v1", req.Intent.ID)); err != nil {
				t.Fatal(err)
			}
			if snapshot, err := InspectExecution(ctx, path, DefaultLimits()); snapshot != nil || !errors.Is(err, ErrCorrupt) {
				t.Fatal("missing decision after a shared reactivation", err)
			}
		})
	}
}

// Two dynamic siblings are born together and remain at epoch one through
// closure. Their entire interval excludes any intervening successful Open.
func TestInspectClosedDynamicLifetimeConstrainsTargetAbsence(t *testing.T) {
	ctx := t.Context()
	p, profile := invocationFixture(t, "succeeded", 1)
	const sender = "sender"
	addProgram(t, &p, sender, []Instruction{
		{Op: "emit", Dst: "sent", Effect: &Effect{Kind: "message", Target: Lit(Text("never-present")), Generation: Lit(Int(1)), Obligation: Lit(Text("work"))}},
		{Op: "end", Outcome: "succeeded"},
	}, map[string]Value{})
	path := filepath.Join(t.TempDir(), "closed-interval.sqlite")
	s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			_ = s.Close()
		}
	}()
	commit := func(id RunID, observation string) RunState {
		t.Helper()
		r, err := s.Commit(ctx, mustSnapshot(t, s, id).Authority(), observation)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	sent := commit(RunID{sender, 1}, "").Projection.Data["sent"].Text
	first := invokeOnce(t, s)
	entry := compositionMemberID(first.Invocation.ID, "entry")
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if s, err = Open(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	bridge, err := s.DeliverMessage(ctx, sent, "epoch-two")
	if err != nil || bridge.Reason != "target_missing" {
		t.Fatal(bridge, err)
	}
	visit, err := s.AdmitInput(ctx, Input{TransportID: "visit", LogicalID: "visit", Target: entry.Subject, Generation: 1, Payload: map[string]Value{}})
	if err != nil || visit.Bound == nil || *visit.Bound != entry || visit.Reason != "route_denied" {
		t.Fatal(visit, err)
	}
	commit(RunID{sender, 1}, "")
	second := invokeOnce(t, s)
	if !first.Accepted || !second.Accepted {
		t.Fatal(first, second)
	}
	commit(RunID{"root", 1}, "")
	commit(entry, firstRequestID(first))
	for _, member := range []string{"entry", "helper"} {
		id := compositionMemberID(second.Invocation.ID, member)
		if member == "entry" {
			commit(id, firstRequestID(second))
			commit(id, "")
		}
		r := commit(id, "")
		if r.Epoch != 1 || r.Projection.Status != "succeeded" {
			t.Fatal(r)
		}
	}
	if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	visit.Bound, visit.Reason = nil, "target_missing"
	if _, err = s.conn.ExecContext(ctx, "UPDATE admissions SET result=CAST(? AS BLOB) WHERE id=?", compositionJSON(visit), admissionKey("input", "visit")); err != nil {
		t.Fatal(err)
	}
	if cut, err := InspectExecution(ctx, path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
		t.Fatalf("accepted missing member inside a known shared activation: %v", err)
	}
}

func firstRequestID(a compositionAdmission) string {
	return compositionIdentity("alan/invocation-request/v1", a.Invocation.ID)
}
