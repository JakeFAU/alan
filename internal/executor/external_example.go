package executor

// ExternalProvision declares a counter increment, a receipt wait, and a peer
// that can finish even when the external outcome remains unknown. It is ordinary
// program data, separate from the nested supervision example.
func ExternalProvision(endpoint, profile string) (Provision, error) {
	if _, err := externalRequest(Intent{Kind: "external", Target: endpoint, Generation: 1, Obligation: "counter-receipt", Payload: map[string]Value{"profile": Text(profile), "delta": Int(5)}}); err != nil {
		return Provision{}, err
	}
	p := Provision{Permissions: []Permission{{Kind: "external", Source: "client", Target: endpoint, Operation: "apply", Revision: 1}}}
	for _, entry := range []struct {
		name string
		code []Instruction
	}{
		{"client", []Instruction{
			{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(endpoint)), Generation: Lit(Int(1)), Obligation: Lit(Text("counter-receipt")), Payload: map[string]Operand{"profile": Lit(Text(profile)), "delta": Lit(Int(5))}}},
			{Op: "receive", Match: &Match{Kind: "receipt", Provenance: "resource-receipt", Source: Lit(Text(endpoint)), Generation: Lit(Int(1)), Obligation: Lit(Text("counter-receipt")), Operation: Ref("pending")}, Bind: map[string]string{"total": "total"}},
			{Op: "end", Outcome: "succeeded"},
		}},
		{"peer", []Instruction{{Op: "end", Outcome: "succeeded"}}},
	} {
		raw, id, err := EncodeProgram(Program{Semantics: SemanticVersion, Code: entry.code})
		if err != nil {
			return Provision{}, err
		}
		p.Programs = append(p.Programs, Artifact{ID: id, Semantics: SemanticVersion, Bytes: raw})
		p.Subjects = append(p.Subjects, Subject{ID: entry.name, Generation: 1, Owner: "owner:" + entry.name, Revision: 1})
		p.Runs = append(p.Runs, InitialRun{ID: RunID{entry.name, 1}, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{}}})
	}
	return p, nil
}
