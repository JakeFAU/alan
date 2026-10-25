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
