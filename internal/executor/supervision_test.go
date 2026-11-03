package executor

import (
	"errors"
	"testing"
)

func TestNestedOrdinaryProgramsExecuteRealSuccessor(t *testing.T) {
	for _, config := range []NestedConfig{{RetryBudget: 1, SuccessorInput: "success"}, {Outer: "alpha", Inner: "beta", Worker: "gamma", RetryBudget: 1, SuccessorInput: "success"}} {
		t.Run(config.Outer, func(t *testing.T) {
			p, err := NestedProvision(config)
			if err != nil {
				t.Fatal(err)
			}
			s, path := messageStore(t, p)
			report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100})
			if err != nil {
				t.Fatal(err)
			}
			if !report.Quiescent || len(report.OpenRuns) != 0 || len(report.UnresolvedIntents) != 0 || count(t, s, "lifecycle_results") != 1 {
				t.Fatalf("nested computation incomplete %+v", report)
			}
			worker := config.Worker
			if worker == "" {
				worker = "worker"
			}
			inner := config.Inner
			if inner == "" {
				inner = "inner"
			}
			outer := config.Outer
			if outer == "" {
				outer = "outer"
			}
			successor, err := s.Snapshot(t.Context(), RunID{worker, 2})
			if err != nil {
				t.Fatal(err)
			}
			if successor.Sequence != 3 || successor.Projection.Status != "succeeded" || current(t, s, inner).Projection.Data["spent"] != Int(1) || current(t, s, inner).Projection.Data["expected"] != Int(2) || current(t, s, outer).Projection.Status != "succeeded" {
				t.Fatalf("ordinary successor/policy state %+v", successor)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestNestedBudgetExhaustionIsOrdinaryPublicFailure(t *testing.T) {
	for _, budget := range []int64{0, 1} {
		t.Run(string(rune('0'+budget)), func(t *testing.T) {
			p, err := NestedProvision(NestedConfig{RetryBudget: budget, SuccessorInput: "fail"})
			if err != nil {
				t.Fatal(err)
			}
			s, path := messageStore(t, p)
			if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100}); err != nil {
				t.Fatal(err)
			}
			if current(t, s, "inner").Projection.Status != "failed" || current(t, s, "outer").Projection.Status != "failed" || current(t, s, "inner").Projection.Data["spent"] != Int(budget) || count(t, s, "lifecycle_results") != int(budget) {
				t.Fatal("budget exhaustion not ordinary escalation")
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestNestedLostAcceptedResultPreservesSameObligation(t *testing.T) {
	p, err := NestedProvision(NestedConfig{RetryBudget: 1, SuccessorInput: "success"})
	if err != nil {
		t.Fatal(err)
	}
	s, path := messageStore(t, p)
	for attempts := 0; current(t, s, "inner").Projection.PC != 6; attempts++ {
		if attempts >= 30 {
			t.Fatal("fixture did not reach ordinary Retry instruction")
		}
		if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 1, MaxDispatch: 100}); err != nil && !errors.Is(err, ErrLimit) {
			t.Fatal(err)
		}
	}
	committed, err := s.Commit(t.Context(), current(t, s, "inner").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	pending := committed.Projection.Data["pending"].Text
	request, err := s.LoadLifecycleRequest(t.Context(), pending)
	if err != nil {
		t.Fatal(err)
	}
	decision := authorizedDecision(t, s, request)
	if !decision.Accepted {
		t.Fatal(decision)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resumed := current(t, s, "inner")
	if resumed.ID != committed.ID || resumed.Epoch != committed.Epoch+1 || resumed.Projection.Data["pending"] != Text(pending) || resumed.Projection.Data["spent"] != Int(1) {
		t.Fatal("same-execution pending/budget lost")
	}
	repeated, err := s.DecideLifecycle(t.Context(), request)
	if err != nil || repeated.Successor != decision.Successor {
		t.Fatal("decision retransmission changed")
	}
	if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100}); err != nil {
		t.Fatal(err)
	}
	if count(t, s, "lifecycle_results") != 1 || current(t, s, "inner").Projection.Data["spent"] != Int(1) || current(t, s, "outer").Projection.Status != "succeeded" {
		t.Fatal("recovery duplicated ordinary decision")
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestRetryProbeDistinguishesSilentChildWithoutTerminalDelivery(t *testing.T) {
	worker := Lit(Text("worker"))
	generation := Lit(Int(1))
	obligation := Lit(Text("work"))
	parentCode := []Instruction{
		{Op: "emit", Dst: "pending", Effect: &Effect{
			Kind: "lifecycle", Target: worker, Generation: generation, Obligation: obligation,
			Payload: map[string]Operand{"operation": Lit(Text("Retry")), "revision": Lit(Int(1))},
		}},
		{Op: "receive", Match: &Match{
			Kind: "lifecycle", Provenance: "runtime-lifecycle", Source: worker,
			Generation: generation, Obligation: obligation, Operation: Ref("pending"),
		}, Bind: map[string]string{"accepted": "accepted", "reason": "reason", "successor": "successor"}},
		{Op: "branch", A: Ref("accepted"), B: Lit(Value{Kind: "bool", Flag: true}), Target: 4, Otherwise: 3},
		{Op: "end", Outcome: "succeeded"},
		{Op: "end", Outcome: "failed"},
	}
	workerCode := []Instruction{
		{Op: "branch", A: Ref("bit"), B: Lit(Int(0)), Target: 1, Otherwise: 2},
		{Op: "end", Outcome: "succeeded"},
		{Op: "end", Outcome: "failed"},
	}
	for _, tc := range []struct {
		outcome   string
		bit       int64
		accepted  bool
		reason    string
		successor int64
	}{
		{"succeeded", 0, false, "not_retryable", 0},
		{"failed", 1, true, "retried", 2},
	} {
		runProbe := func(t *testing.T, terminalGranted bool) {
			p := Provision{Permissions: []Permission{
				{Kind: "lifecycle", Source: "parent", Target: "worker", Operation: "Retry", Revision: 1},
			}}
			if terminalGranted {
				p.Permissions = append(p.Permissions, Permission{
					Kind: "observation", Source: "worker", Target: "parent",
					Operation: "terminal", Obligation: "work", Revision: 1,
				})
			}
			addProgram(t, &p, "parent", parentCode, map[string]Value{})
			addProgram(t, &p, "worker", workerCode, map[string]Value{"bit": Int(tc.bit)})
			s, path := messageStore(t, p)
			commit := func(subject, selected string) RunState {
				t.Helper()
				next, err := s.Commit(t.Context(), current(t, s, subject).Authority(), selected)
				if err != nil {
					t.Fatal(err)
				}
				return next
			}

			initialParent := current(t, s, "parent")
			commit("worker", "")
			ended := commit("worker", "")
			if ended.Projection.Status != tc.outcome || count(t, s, "terminal_obligations") != 1 || count(t, s, "intents") != 0 {
				t.Fatalf("silent child End = %+v", ended)
			}
			if !terminalGranted {
				destination := TerminalDestination{Target: "parent", Obligation: "work"}
				_, err := s.DeliverTerminal(t.Context(), ended.ID, destination, "child-end")
				if err == nil || err.Error() != "terminal destination not retained" {
					t.Fatalf("unprovisioned terminal delivery = %v, want missing-route error", err)
				}
			}
			if parent := current(t, s, "parent"); parent.Authority() != initialParent.Authority() || parent.ProgramID != initialParent.ProgramID ||
				parent.Projection.PC != 0 || parent.Projection.Status != "open" || len(parent.Projection.Data) != 0 {
				t.Fatalf("child End changed the parent before its Retry: %+v", parent)
			}
			request := committedRequest(t, s, "parent")
			authorizedDecision(t, s, request)
			if count(t, s, "observations") != 0 {
				t.Fatal("parent received an observation before lifecycle settlement")
			}
			if _, err := s.Commit(t.Context(), current(t, s, "parent").Authority(), ""); !errors.Is(err, ErrIneligible) {
				t.Fatalf("unserviced receive = %v, want ErrIneligible", err)
			}
			settlement, err := s.SettleLifecycle(t.Context(), request.Intent.ID, "retry-result")
			if err != nil || !settlement.Accepted {
				t.Fatalf("settle original Retry = %+v, %v", settlement, err)
			}
			received := commit("parent", settlement.ObservationID)
			data := received.Projection.Data
			if data["accepted"] != (Value{Kind: "bool", Flag: tc.accepted}) || data["reason"] != Text(tc.reason) || data["successor"] != Int(tc.successor) {
				t.Fatalf("consumed lifecycle fields = %+v", data)
			}
			target, err := loadSubject(t.Context(), s.conn, "worker")
			if err != nil || target.Generation != max(1, tc.successor) {
				t.Fatalf("worker generation after Retry = %d, %v", target.Generation, err)
			}
			commit("parent", "")
			parent := commit("parent", "")
			if parent.Projection.Status != tc.outcome || count(t, s, "observations") != 1 || count(t, s, "admissions") != 1 {
				t.Fatalf("parent End after only the original settlement = %+v", parent)
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		}
		t.Run(tc.outcome, func(t *testing.T) { runProbe(t, true) })
		t.Run(tc.outcome+"_without_terminal_grant", func(t *testing.T) { runProbe(t, false) })
	}
}
