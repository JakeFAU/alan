package substitution

// retainedBSuccessors specializes the historical instruction transcription to
// one successful retained instruction per private edge. N's choose_helper and
// pending_receive bookkeeping were separate edges without an instruction.
// Here each destination already describes its actual next instruction.
func retainedBSuccessors(c Config, s State) []Edge {
	edges := CandidateGraph(Helper)(c, s)
	for i := range edges {
		edge := &edges[i]
		actor := entry
		switch {
		case edge.Environment:
			continue
		case edge.Event.Kind == "invoke_commit" && edge.Event.Operation == "e",
			edge.Event.Kind == "outcome_consume" && edge.Event.Operation == "e":
			continue // The fixed caller owns these instructions.
		case edge.Event.Kind == "request_consume" && edge.Event.Operation == "e":
			edge.Next.Ops[entry].PC = invoke
			edge.Next.Ops[entry].Choice = 2
			edge.Next.Ops[helper].Enabled = true
		case edge.Event.Kind == "private_step" && edge.Event.Operation == "e" &&
			s.PrivatePC[entry] == 10 && edge.Next.PrivatePC[entry] == 12:
			edge.Next.Ops[entry].PC = pending
			edge.Event = observable(edge.Next, entry, "pending_receive")
		case edge.Event.Operation == "h" && edge.Event.Kind != "invoke_commit" && edge.Event.Kind != "outcome_consume":
			actor = helper
		}
		// Both PCs can jump, and End stays at its source PC. Retain the
		// successful sequence separately, including that final instruction.
		edge.Next.PrivateRegisters[actor][7]++
	}
	return edges
}

// At the post-Receive PC1 the fixed code can only emit its original helper
// Invoke. K1 can still choose either branch. h is uncommitted in both states.
// Its retained identity, ticket and all already visible protocol fields agree.
func retainedBReferencePair(s State) State {
	k := publicState(s)
	if s.Ops[entry].Admission == 1 && s.PrivatePC[entry] == 1 {
		k.Ops[entry].PC = choose
		k.Ops[entry].Choice = 0
		k.Ops[helper].Enabled = false
	}
	return k
}
