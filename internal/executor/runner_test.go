package executor

import (
	"errors"
	"testing"
)

func TestRunnerMakesIndependentProgressAndReportsQuiescence(t *testing.T) {
	p := inputFixture(t)
	addProgram(t, &p, "independent", []Instruction{{Op: "set", Dst: "value", A: Lit(Int(999))}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, _ := messageStore(t, p)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || report.Steps != 2 || len(report.OpenRuns) != 1 || report.OpenRuns[0] != (RunID{"receiver", 1}) || current(t, s, "independent").Projection.Status != "succeeded" {
		t.Fatalf("waiting blocked independent execution %+v", report)
	}
}
func TestRunnerLimitAndUnsupportedEffectRemainVisible(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "looper", []Instruction{{Op: "jump", Target: 0}}, map[string]Value{})
	addProgram(t, &p, "opaque", []Instruction{{Op: "emit", Effect: &Effect{Kind: "future-operation", Target: Lit(Text("endpoint")), Generation: Lit(Int(0)), Obligation: Lit(Text(""))}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, _ := messageStore(t, p)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 5, MaxDispatch: 10})
	if !errors.Is(err, ErrLimit) || report.Quiescent || report.Steps != 5 || len(report.UnresolvedIntents) != 1 || current(t, s, "opaque").Projection.Status != "succeeded" {
		t.Fatalf("limit/fairness/unresolved %+v %v", report, err)
	}
}
func TestRunnerAutomaticallyDispatchesMessages(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "sender", []Instruction{{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(42))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	receiver := inputProgram()
	receiver[0].Match.Kind = "message"
	receiver[0].Match.Provenance = "ordinary-message"
	receiver[0].Match.Source = Lit(Text("sender"))
	addProgram(t, &p, "receiver", receiver, map[string]Value{})
	p.Permissions = []Permission{{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "work"}}
	s, _ := messageStore(t, p)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || report.Steps != 4 || report.Dispatches != 1 || len(report.UnresolvedIntents) != 0 || current(t, s, "receiver").Projection.Data["number"] != Int(42) {
		t.Fatalf("automatic message execution %+v", report)
	}
}

func TestRunnerAutomaticallyDeliversFrozenTerminalOutcome(t *testing.T) {
	for _, names := range [][2]string{{"child", "observer"}, {"renamed-17", "renamed-99"}} {
		t.Run(names[0], func(t *testing.T) {
			p := Provision{}
			addProgram(t, &p, names[0], []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
			addProgram(t, &p, names[1], []Instruction{{Op: "receive", Match: &Match{Kind: "terminal", Provenance: "runtime-terminal", Source: Lit(Text(names[0])), Generation: Lit(Int(1)), Obligation: Lit(Text("public")), Operation: Lit(Text(""))}, Bind: map[string]string{"outcome": "public-outcome"}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			p.Permissions = []Permission{{Kind: "observation", Source: names[0], Target: names[1], Operation: "terminal", Revision: 1, Obligation: "public"}}
			s, path := messageStore(t, p)
			report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
			if err != nil {
				t.Fatal(err)
			}
			if !report.Quiescent || report.Steps != 3 || report.Dispatches != 1 || len(report.OpenRuns) != 0 || current(t, s, names[1]).Projection.Data["public-outcome"] != Text("failed") {
				t.Fatalf("terminal propagation %+v", report)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestRunnerDispatchBudgetRetainsUnfinishedWork(t *testing.T) {
	p := Provision{}
	effect := &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(1))}}
	addProgram(t, &p, "sender", []Instruction{{Op: "emit", Effect: effect}, {Op: "emit", Effect: effect}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	addProgram(t, &p, "receiver", inputProgram(), map[string]Value{})
	p.Permissions = []Permission{{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "work"}}
	s, _ := messageStore(t, p)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 1})
	if !errors.Is(err, ErrLimit) || report.Quiescent || report.Dispatches != 1 || len(report.PendingDeliveries) != 1 || len(report.UnresolvedIntents) != 1 {
		t.Fatalf("dispatch limit hid work %+v %v", report, err)
	}
}
