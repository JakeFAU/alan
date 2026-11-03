package executor

import "fmt"

// NestedConfig changes only ordinary fixture data and participant names. The
// interpreter/runner do not inspect this type or assign runtime supervisory roles.
type NestedConfig struct {
	Outer, Inner, Worker string
	RetryBudget          int64
	SuccessorInput       string
}

func NestedProvision(config NestedConfig) (Provision, error) {
	if config.Outer == "" {
		config.Outer = "outer"
	}
	if config.Inner == "" {
		config.Inner = "inner"
	}
	if config.Worker == "" {
		config.Worker = "worker"
	}
	if config.SuccessorInput == "" {
		config.SuccessorInput = "success"
	}
	if config.RetryBudget < 0 || config.Outer == config.Inner || config.Inner == config.Worker || config.Outer == config.Worker {
		return Provision{}, fmt.Errorf("invalid nested fixture configuration")
	}
	p := Provision{}
	add := func(subject string, code []Instruction, data map[string]Value) error {
		encoded, id, err := EncodeProgram(Program{Semantics: SemanticVersion, Code: code})
		if err != nil {
			return err
		}
		p.Programs = append(p.Programs, Artifact{ID: id, Semantics: SemanticVersion, Bytes: encoded})
		p.Subjects = append(p.Subjects, Subject{ID: subject, Generation: 1, Owner: "owner:" + subject, Revision: 1})
		p.Runs = append(p.Runs, InitialRun{ID: RunID{subject, 1}, ProgramID: id, Projection: Projection{Status: "open", Data: data}})
		return nil
	}
	receive := func(kind, provenance, source, obligation string, generation, operation Operand, bind map[string]string) Instruction {
		return Instruction{Op: "receive", Match: &Match{Kind: kind, Provenance: provenance, Source: Lit(Text(source)), Generation: generation, Obligation: Lit(Text(obligation)), Operation: operation}, Bind: bind}
	}
	message := func(target, obligation string, generation Operand, payload map[string]Operand) Instruction {
		return Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text(target)), Generation: generation, Obligation: Lit(Text(obligation)), Payload: payload}}
	}
	if err := add(config.Worker, []Instruction{
		receive("message", "ordinary-message", config.Inner, "worker-work", Lit(Int(1)), Lit(Text("")), map[string]string{"mode": "mode"}),
		{Op: "branch", A: Ref("mode"), B: Lit(Text("success")), Target: 2, Otherwise: 3},
		{Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}, map[string]Value{}); err != nil {
		return p, err
	}
	delta := int64(1)
	if err := add(config.Inner, []Instruction{
		receive("message", "ordinary-message", config.Outer, "inner-start", Lit(Int(1)), Lit(Text("")), map[string]string{"start": "started"}),
		message(config.Worker, "worker-work", Ref("expected"), map[string]Operand{"mode": Lit(Text("fail"))}),
		receive("terminal", "runtime-terminal", config.Worker, "worker-obligation", Ref("expected"), Lit(Text("")), map[string]string{"outcome": "outcome"}),
		{Op: "branch", A: Ref("outcome"), B: Lit(Text("succeeded")), Target: 13, Otherwise: 4},
		{Op: "branch", A: Ref("outcome"), B: Lit(Text("failed")), Target: 5, Otherwise: 14},
		{Op: "branch", A: Ref("spent"), B: Ref("limit"), Target: 14, Otherwise: 6},
		{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text(config.Worker)), Generation: Ref("expected"), Obligation: Lit(Text("worker-obligation")), Payload: map[string]Operand{"operation": Lit(Text("Retry")), "revision": Ref("revision")}}, Updates: map[string]Operand{"spent": {Ref: "spent", Offset: &delta}}},
		receive("lifecycle", "runtime-lifecycle", config.Worker, "worker-obligation", Ref("expected"), Ref("pending"), map[string]string{"accepted": "accepted", "reason": "reason", "successor": "successor", "disposition": "disposition"}),
		{Op: "branch", A: Ref("accepted"), B: Lit(Value{Kind: "bool", Flag: true}), Target: 9, Otherwise: 14},
		{Op: "set", Dst: "expected", A: Ref("successor")},
		{Op: "set", Dst: "pending", A: Lit(Text(""))},
		message(config.Worker, "worker-work", Ref("expected"), map[string]Operand{"mode": Ref("successor-input")}),
		{Op: "jump", Target: 2},
		{Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}, map[string]Value{"spent": Int(0), "limit": Int(config.RetryBudget), "expected": Int(1), "revision": Int(1), "pending": Text(""), "successor-input": Text(config.SuccessorInput)}); err != nil {
		return p, err
	}
	// This outer policy has zero retries: any public inner failure escalates.
	if err := add(config.Outer, []Instruction{
		message(config.Inner, "inner-start", Lit(Int(1)), map[string]Operand{"start": Lit(Value{Kind: "bool", Flag: true})}),
		receive("terminal", "runtime-terminal", config.Inner, "outer-obligation", Lit(Int(1)), Lit(Text("")), map[string]string{"outcome": "inner-outcome"}),
		{Op: "branch", A: Ref("inner-outcome"), B: Lit(Text("succeeded")), Target: 3, Otherwise: 4},
		{Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}, map[string]Value{}); err != nil {
		return p, err
	}
	p.Permissions = []Permission{
		{Kind: "message", Source: config.Outer, Target: config.Inner, Operation: "message", Revision: 1, Obligation: "inner-start"},
		{Kind: "message", Source: config.Inner, Target: config.Worker, Operation: "message", Revision: 1, Obligation: "worker-work"},
		{Kind: "observation", Source: config.Worker, Target: config.Inner, Operation: "terminal", Revision: 1, Obligation: "worker-obligation"},
		{Kind: "observation", Source: config.Inner, Target: config.Outer, Operation: "terminal", Revision: 1, Obligation: "outer-obligation"},
		{Kind: "lifecycle", Source: config.Inner, Target: config.Worker, Operation: "Retry", Revision: 1},
	}
	return p, nil
}
