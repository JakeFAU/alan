package semantics

import (
	"fmt"
	"slices"
)

func supervisorProgram(child string, budget int, abandoned bool) []instruction {
	abandonmentPC := 20
	if abandoned {
		abandonmentPC = 10
	}
	return []instruction{
		{Op: set, Mode: "name", Slot: 0, Tag: child},
		{Op: set, Mode: "identity", Slot: 1, Key: "declared-obligation"},
		{Op: set, Mode: "local", Slot: childGeneration, Value: 1},
		{Op: set, Mode: "local", Slot: revision, Value: 7},
		{Op: set, Mode: "local", Slot: limit, Value: budget},
		{Op: receive, Mode: "observation", Tag: "child"},
		{Op: branch, Mode: "local", Slot: classification, Value: 1, PC: 17},
		{Op: branch, Mode: "local", Slot: classification, Value: 2, PC: 10},
		{Op: branch, Mode: "local", Slot: classification, Value: 5, PC: 5},
		{Op: branch, Mode: "local", Slot: scratch, Value: 0, PC: 19},
		{Op: branch, Mode: "local", Slot: spent, FromReg: true, Source: limit, PC: 18},
		{Op: emit, Mode: "lifecycle", Tag: string(retry), Slot: spent, Delta: 1},
		{Op: receive, Mode: "lifecycle", Tag: "lifecycle"},
		{Op: branch, Mode: "local", Slot: acceptedResult, Value: 1, PC: 15},
		{Op: fault},
		{Op: set, Mode: "copy", Slot: childGeneration, Source: successor},
		{Op: branch, Mode: "local", Slot: scratch, Value: 0, PC: 5},
		{Op: success},
		{Op: fault},
		{Op: branch, Mode: "local", Slot: classification, Value: 3, PC: abandonmentPC},
		{Op: fault},
	}
}
func supervisionFixture(nested bool) config {
	c := config{Version: "v1c", Programs: map[string][]instruction{"worker": {{Op: fault}}, "inner": supervisorProgram("worker", 1, false)}, Routes: map[string]string{"worker": "inner", "inner": "observer"}, Edges: map[string]bool{"worker>inner": true}, Inputs: map[string]message{}, Bounds: bounds{Generations: 3, Epochs: 3, Commits: 48, Continuations: 2, Proposals: 2, Intents: 8, Attempts: 2, Inbox: 16, Value: 8}, Idempotent: true, Supervision: &supervisionContract{Grants: []lifecycleGrant{{Requester: "inner", Target: "worker", Operation: retry, W: 7}}, Policies: map[string]policyContract{"inner": {Child: "worker", Limit: 1}}}}
	if nested {
		c.Programs["outer"] = supervisorProgram("inner", 1, false)
		c.Routes["inner"] = "outer"
		c.Edges["inner>outer"] = true
		c.Supervision.Grants = append(c.Supervision.Grants, lifecycleGrant{Requester: "outer", Target: "inner", Operation: retry, W: 7})
		c.Supervision.Policies["outer"] = policyContract{Child: "inner", Limit: 1}
	}
	return c
}
func terminalID(a string, g int, variant string) string {
	return "terminal:" + runID(a, g) + ":" + variant
}
func resultID(id, variant string) string { return "lifecycle:" + id + ":" + variant }
func supervisorInit(b *traceBuilder, a string, g int) {
	for range 5 {
		b.local(a, g, "")
	}
}
func terminalDecision(b *traceBuilder, a, child string, g, childG int, variant string) {
	b.add(event{Kind: deliver, A: child, G: childG, ID: variant})
	b.local(a, g, terminalID(child, childG, variant))
	for range 4 {
		b.local(a, g, "")
	}
}
func lifecycleDispatch(b *traceBuilder, id string, admitResult bool) {
	b.add(event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7}, event{Kind: decideLifecycle, ID: id})
	if admitResult {
		b.add(event{Kind: admitLifecycle, ID: id})
	}
}
func acceptLifecycle(b *traceBuilder, a string, g int, id string) {
	b.local(a, g, resultID(id, ""))
	for range 3 {
		b.local(a, g, "")
	}
}
func recoveredSeed(nested bool) traceBuilder {
	b := traceBuilder{Config: supervisionFixture(nested)}
	supervisorInit(&b, "inner", 1)
	if nested {
		supervisorInit(&b, "outer", 1)
	}
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	lifecycleDispatch(&b, intentID("inner", 1, 10), true)
	acceptLifecycle(&b, "inner", 1, intentID("inner", 1, 10))
	return b
}

// The same queue, full-state identity, carried oracle and inspect-before-dedup
// discipline as Phase 1B. No fixture-specific schedule is hidden in step.
func exploreSupervision(spec compositionSpec) (compositionStats, error) {
	st := compositionStats{Name: spec.Name, Category: "bounded-safety", Config: spec.Config, ConfigID: digest(spec.Config), ScenarioID: digest(spec), Seed: spec.Seed, SeedLength: len(spec.Seed), Alphabet: spec.Alphabet, DepthLimit: spec.Depth, StateCap: spec.StateCap, Rejections: map[string]int{}, Witnesses: map[string][]event{}}
	if spec.Category != "" {
		st.Category = spec.Category
	}
	initial, err := replayCompositionSeed(spec)
	if err != nil {
		return st, err
	}
	queue := []compositionNode{initial}
	seen := map[string]bool{compositionKey(initial): true}
	st.States = 1
	for head := 0; head < len(queue); head++ {
		n := queue[head]
		depth := len(n.Trace)
		st.MaxDepth = max(st.MaxDepth, depth)
		if depth == spec.Depth {
			st.Boundary++
			continue
		}
		for _, e := range spec.Alphabet {
			after, why := step(n.Machine, e)
			mon, bad := inspect(n.Machine, after, e, n.Monitor)
			trace := append(slices.Clone(n.Trace), e)
			if why != "" {
				st.Rejected++
				st.Rejections[why]++
			} else {
				st.Transitions++
			}
			if bad != "" {
				st.Violation = bad
				st.Trace = trace
				st.MaxDepth = max(st.MaxDepth, len(trace))
				return st, nil
			}
			for _, w := range supervisionWitness(n.Machine, after, e, why, &mon) {
				if _, ok := st.Witnesses[w]; !ok {
					st.Witnesses[w] = trace
				}
			}
			candidate := compositionNode{Machine: after, Monitor: mon, Trace: trace}
			key := compositionKey(candidate)
			if seen[key] {
				st.Revisits++
				continue
			}
			if len(seen) >= spec.StateCap {
				return st, fmt.Errorf("hard state cap %d exhausted in %s", spec.StateCap, spec.Name)
			}
			seen[key] = true
			st.States++
			queue = append(queue, candidate)
		}
	}
	return st, nil
}
func supervisionWitness(before, after machine, e event, why string, mon *monitor) []string {
	if mon.Supervision == nil {
		mon.Supervision = &supervisionFacts{Seen: map[string]bool{}}
	}
	f := mon.Supervision.Seen
	if why == "" && e.Kind == interrupt {
		// Interrupt names a proposal slot, not an execution. Attribute the
		// event to the proposal's committed run and its exact pending intent.
		p := before.Proposals[e.Slot]
		if p != nil {
			key := pendingWitnessKey(before.Runs[runID(p.A, p.G)])
			if key != "" {
				f["interrupted:"+key] = true
			}
		}
	}
	if why == "" && (e.Kind == cancel || e.Kind == retire) {
		if before.Lifecycle != nil {
			for _, r := range before.Lifecycle.Results {
				q := requestValue(r.Request)
				if q.Target == e.A {
					outcome := "rejected"
					if r.Accepted {
						outcome = "accepted"
					}
					f["decision-"+outcome+"-before-"+string(e.Kind)] = true
				}
			}
		}
		f["environment:"+string(e.Kind)] = true
	}
	var out []string
	for k := range f {
		if k == "decision-accepted-before-CancelRun" || k == "decision-accepted-before-RetireSubject" {
			out = append(out, k)
		}
	}
	if why == "" && e.Kind == decideLifecycle && after.Lifecycle != nil {
		r := after.Lifecycle.Results[e.ID]
		if f["environment:CancelRun"] {
			out = append(out, "CancelRun-before-decision")
		}
		if f["environment:RetireSubject"] && !r.Accepted {
			out = append(out, "RetireSubject-before-rejected-decision")
		}
	}
	if why == "terminal correlation" {
		f["old-terminal-blocked"] = true
	}
	if why == "lifecycle correlation" {
		f["old-result-blocked"] = true
	}
	if why == "observation kind" {
		f["suspicion-blocked"] = true
	}
	if why == "closed" && e.Kind == commit {
		f["stale-commit-blocked"] = true
	}
	if why == "epoch" {
		out = append(out, "stale-epoch-blocked")
	}
	if why == "" && e.Kind == reactivate {
		rid := runID(e.A, e.G)
		key := pendingWitnessKey(before.Runs[rid])
		if key != "" && key == pendingWitnessKey(after.Runs[rid]) {
			f["pending-reactivated:"+key] = true
			// This fact is created only at the later Reactivate event. An
			// earlier reactivation cannot combine with a future Interrupt.
			if f["interrupted:"+key] {
				f["interrupted-reactivated:"+key] = true
			}
		}
	}
	if after.Lifecycle != nil {
		for id, v := range after.Lifecycle.Results {
			q := requestValue(v.Request)
			if !v.Accepted && v.Reason == "authority" {
				out = append(out, "authority-rejection-retained")
			}
			if before.Lifecycle != nil {
				if old, ok := before.Lifecycle.Results[id]; ok && e.Kind == decideLifecycle && e.ID == id && old == v {
					f["resent:"+id] = true
					if v.Accepted {
						out = append(out, "accepted-result-retransmitted")
					} else {
						out = append(out, "rejected-result-retransmitted")
					}
				}
			}
			if v.Accepted && v.Successor > 0 {
				rr := after.Runs[runID(q.Target, v.Successor)]
				if rr != nil && rr.N > 0 {
					out = append(out, "successor-committed")
					if f["resent:"+id] {
						out = append(out, "lost-ack-retransmission-successor-commit")
					}
				}
			}
		}
	}
	for a, s := range after.Subjects {
		r := after.Runs[runID(a, s.Current)]
		if s.Retired {
			out = append(out, "retirement-observed")
		}
		if r.Projection.Local != nil {
			d := r.Projection.Local
			if key := pendingWitnessKey(r); key != "" && f["pending-reactivated:"+key] {
				out = append(out, "pending-survives-reactivation")
				if f["interrupted-reactivated:"+key] {
					out = append(out, "committed-intent-interrupted-reactivated")
				}
			}
			if r.Projection.Status == "failed" && d.Values[spent] == d.Values[limit] && d.Values[spent] > 0 {
				out = append(out, "ordinary-exhaustion")
			}
			if r.Projection.Status == "failed" && d.Last != "" && noticeValue(d.Last).Kind == "lifecycle" && !noticeValue(d.Last).Result.Accepted {
				out = append(out, "ordinary-rejection-escalation")
			}
			if len(d.Seen) > 1 && f["old-terminal-blocked"] {
				out = append(out, "stale-terminal-blocked-current-consumed")
			}
			if d.Last != "" && noticeValue(d.Last).Kind == "lifecycle" && f["old-result-blocked"] {
				q := requestValue(noticeValue(d.Last).Result.Request)
				if q.RequesterG == r.G {
					out = append(out, "stale-result-blocked-current-consumed")
				}
			}
		}
	}
	if why == "" && e.Kind == commit {
		p := before.Proposals[e.Slot]
		if p != nil {
			if p.Observed != nil && p.Observed.Notice != "" {
				if noticeValue(p.Observed.Notice).Kind == "suspicion" {
					f["suspicion-consumed"] = true
				}
				out = append(out, "ordinary-observation-consumed")
			}
			if p.Intent != nil && p.Intent.Lifecycle != "" {
				out = append(out, "ordinary-lifecycle-intent")
			}
			if p.A == "worker" && p.G == 1 && f["suspicion-consumed"] {
				out = append(out, "suspicion-consumed-same-generation-commit")
			}
		}
	}
	if after.World.Mutations >= 2 {
		out = append(out, "ambiguous-repeat-business-mutation")
	}
	for id, in := range after.Intents {
		if in.Descriptor.Lifecycle != "" || in.Descriptor.Kind != emit {
			continue
		}
		d := in.Descriptor
		old := after.Runs[runID(d.A, d.G)]
		if _, ok := after.World.Keys[id]; ok && !in.Settled && old.Outcome != nil {
			f["unknown:"+id] = true
		}
		curr := after.Runs[runID(d.A, after.Subjects[d.A].Current)]
		if f["unknown:"+id] && curr.G > d.G && curr.N > 0 && in.Settled {
			out = append(out, "unknown-application-recovery-late-receipt")
		}
	}
	slices.Sort(out)
	return out
}

func rawRequestFixture() config {
	c := supervisionFixture(false)
	c.Supervision.Policies = map[string]policyContract{}
	c.Programs["inner"] = append(supervisorProgram("worker", 1, false)[:5], instruction{Op: emit, Mode: "lifecycle", Tag: string(retry)}, instruction{Op: receive, Mode: "lifecycle", Tag: "lifecycle"}, instruction{Op: fault})
	return c
}
func rawRequestSeed() traceBuilder {
	b := traceBuilder{Config: rawRequestFixture()}
	supervisorInit(&b, "inner", 1)
	b.local("inner", 1, "")
	return b
}
func supervisionScenarios() []compositionSpec {
	prep := func(a string, g int, obs string) event {
		return event{Kind: prepare, A: a, G: g, C: 1, Observation: obs}
	}
	cm := event{Kind: commit}
	makeSpec := func(name string, b traceBuilder, alpha []event) compositionSpec {
		return compositionSpec{Name: name, Config: b.Config, Seed: b.Events, Alphabet: alpha, Depth: 10, StateCap: 5000}
	}
	var cases []compositionSpec
	b := traceBuilder{Config: supervisionFixture(false)}
	fresh := makeSpec("fresh_ordinary_initialization", b, []event{prep("inner", 1, ""), cm, prep("worker", 1, ""), {Kind: deliver, A: "worker", G: 1}, {Kind: interrupt}})
	fresh.Depth = 8
	cases = append(cases, fresh)
	b = recoveredSeed(false)
	a := makeSpec("A_duplicate_failure_current_generation", b, []event{{Kind: deliver, A: "worker", G: 1, ID: "duplicate"}, prep("inner", 1, terminalID("worker", 1, "duplicate")), prep("worker", 2, ""), cm, {Kind: deliver, A: "worker", G: 2}, prep("inner", 1, terminalID("worker", 2, ""))})
	a.WantWitnesses = []string{"stale-terminal-blocked-current-consumed"}
	a.WantRejects = []string{"terminal correlation"}
	cases = append(cases, a)
	b = traceBuilder{Config: supervisionFixture(false)}
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	id := intentID("inner", 1, 10)
	lifecycleDispatch(&b, id, false)
	lost := makeSpec("B_accepted_lost_ack_and_successor", b, []event{{Kind: decideLifecycle, ID: id}, prep("worker", 2, ""), cm, {Kind: admitLifecycle, ID: id}, prep("inner", 1, resultID(id, "")), life(reactivate, "inner", 1)})
	lost.WantWitnesses = []string{"lost-ack-retransmission-successor-commit", "pending-survives-reactivation"}
	cases = append(cases, lost)
	b = rawRequestSeed()
	rawID := intentID("inner", 1, 6)
	lifecycleDispatch(&b, rawID, false)
	rejected := makeSpec("B_rejected_then_failed_retransmission", b, []event{prep("worker", 1, ""), cm, {Kind: decideLifecycle, ID: rawID}, {Kind: admitLifecycle, ID: rawID}, prep("inner", 1, resultID(rawID, ""))})
	rejected.WantWitnesses = []string{"rejected-result-retransmitted"}
	cases = append(cases, rejected)
	b = traceBuilder{Config: supervisionFixture(false)}
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	interrupted := makeSpec("C_committed_intent_interrupted", b, []event{prep("inner", 1, ""), life(reactivate, "inner", 1), {Kind: interrupt}, {Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7}, {Kind: decideLifecycle, ID: id}, {Kind: admitLifecycle, ID: id}, prep("inner", 1, resultID(id, "")), cm})
	interrupted.WantWitnesses = []string{"pending-survives-reactivation", "committed-intent-interrupted-reactivated"}
	cases = append(cases, interrupted)
	for _, retryCancelled := range []bool{false, true} {
		for _, operation := range []kind{cancel, retire} {
			b = rawRequestSeed()
			b.Config.RetryCancelled = retryCancelled
			b.add(event{Kind: authorizeLifecycle, ID: rawID, Owner: "owner", W: 7})
			cases = append(cases, makeSpec(fmt.Sprintf("D_%s_retry_cancelled_%t", operation, retryCancelled), b, []event{life(operation, "worker", 1), {Kind: decideLifecycle, ID: rawID}, prep("worker", 1, ""), cm, prep("worker", 2, "")}))
		}
	}
	b = traceBuilder{Config: supervisionFixture(false)}
	b.Config.Programs["worker"] = []instruction{{Op: set, Value: 1}, {Op: fault}}
	supervisorInit(&b, "inner", 1)
	b.add(prep("worker", 1, ""))
	physical := makeSpec("E_suspicion_interrupt_reactivation", b, []event{{Kind: suspect, A: "worker", G: 1}, prep("inner", 1, "suspicion:worker/g1/e1/suspicion:"), life(reactivate, "worker", 1), cm, {Kind: interrupt}, prep("worker", 1, "")})
	physical.WantWitnesses = []string{"suspicion-consumed-same-generation-commit", "stale-epoch-blocked"}
	cases = append(cases, physical)
	b = recoveredSeed(false)
	b.local("worker", 2, "")
	b.add(event{Kind: deliver, A: "worker", G: 2})
	exhaustion := makeSpec("F_budget_one_exhaustion_capacity_three", b, []event{prep("inner", 1, terminalID("worker", 2, "")), prep("inner", 1, ""), cm, life(reactivate, "inner", 1)})
	exhaustion.WantWitnesses = []string{"ordinary-exhaustion"}
	cases = append(cases, exhaustion)
	b = rawRequestSeed()
	lifecycleDispatch(&b, rawID, false)
	b.add(life(abandon, "inner", 1), life(retry, "inner", 1))
	supervisorInit(&b, "inner", 2)
	b.local("inner", 2, "")
	currentID := intentID("inner", 2, 6)
	lifecycleDispatch(&b, currentID, false)
	old := makeSpec("G_old_lifecycle_result_eligible_current_requester", b, []event{{Kind: admitLifecycle, ID: rawID}, {Kind: admitLifecycle, ID: currentID}, prep("inner", 2, resultID(rawID, "")), prep("inner", 2, resultID(currentID, "")), cm})
	old.WantWitnesses = []string{"stale-result-blocked-current-consumed"}
	old.WantRejects = []string{"lifecycle correlation"}
	cases = append(cases, old)
	b = recoveredSeed(true)
	b.local("worker", 2, "")
	terminalDecision(&b, "inner", "worker", 1, 2, "")
	terminalDecision(&b, "outer", "inner", 1, 1, "")
	outerID := intentID("outer", 1, 10)
	nested := makeSpec("H_nested_boundary_lost_ack_interrupt", b, []event{{Kind: authorizeLifecycle, ID: outerID, Owner: "owner", W: 7}, {Kind: decideLifecycle, ID: outerID}, {Kind: admitLifecycle, ID: outerID}, prep("outer", 1, resultID(outerID, "")), cm, life(reactivate, "outer", 1), {Kind: deliver, A: "inner", G: 1, ID: "duplicate"}, prep("inner", 2, "")})
	nested.Alphabet = append(nested.Alphabet, event{Kind: interrupt})
	nested.WantWitnesses = []string{"pending-survives-reactivation", "accepted-result-retransmitted", "successor-committed", "committed-intent-interrupted-reactivated"}
	cases = append(cases, nested)
	b = traceBuilder{Config: supervisionFixture(false)}
	b.Config.Supervision.Grants = nil
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	denied := makeSpec("I_observation_without_lifecycle_authority", b, []event{{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7}, {Kind: decideLifecycle, ID: id}, {Kind: admitLifecycle, ID: id}, prep("inner", 1, resultID(id, "")), prep("inner", 1, ""), cm})
	denied.WantWitnesses = []string{"authority-rejection-retained", "ordinary-rejection-escalation"}
	cases = append(cases, denied)
	b = traceBuilder{Config: supervisionFixture(false)}
	b.Config.Programs["worker"] = []instruction{{Op: emit, Tag: "business:order-7", Value: 1}, {Op: fault}}
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	b.dispatch(intentID("worker", 1, 1), false, false)
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	lifecycleDispatch(&b, id, true)
	acceptLifecycle(&b, "inner", 1, id)
	external := makeSpec("J_unknown_effect_recovery_late_settlement", b, []event{prep("worker", 2, ""), cm, {Kind: authorize, ID: intentID("worker", 2, 1), Attempt: 1, Owner: "owner", W: 7}, {Kind: applyExternal, ID: intentID("worker", 2, 1), Attempt: 1}, {Kind: recordReceipt, ID: intentID("worker", 1, 1), Attempt: 1}})
	external.WantWitnesses = []string{"unknown-application-recovery-late-receipt", "ambiguous-repeat-business-mutation"}
	cases = append(cases, external)
	b = traceBuilder{Config: supervisionFixture(false)}
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	lifecycleDispatch(&b, id, true)
	budget := makeSpec("broken_budget_persistence", b, []event{prep("inner", 1, resultID(id, "")), cm})
	budget.Config.Mutant = "budget-persistence"
	budget.WantViolation = "supervision-budget-persistence"
	budget.Category = "guard-negative-control"
	cases = append(cases, budget)
	for _, x := range []struct {
		spec              compositionSpec
		mutant, violation string
	}{{rejected, "operation-dedup", "supervision-operation-immutable"}, {a, "failure-correlation", "supervision-failure-correlation"}, {denied, "lifecycle-authority", "supervision-lifecycle-authority"}, {old, "old-lifecycle-result", "supervision-old-lifecycle-result"}} {
		v := copyOf(x.spec)
		v.Name = "broken_" + x.mutant
		v.Config.Mutant = x.mutant
		v.WantViolation = x.violation
		v.WantWitnesses = nil
		v.WantRejects = nil
		v.Category = "guard-negative-control"
		cases = append(cases, v)
	}
	faulty := copyOf(exhaustion)
	faulty.Name = "faulty_ordinary_budget_branch"
	faulty.Config.Programs["inner"][10] = instruction{Op: branch, Mode: "local", Slot: scratch, Value: 1, PC: 18}
	faulty.WantViolation = "supervision-retry-budget"
	faulty.WantWitnesses = nil
	faulty.Category = "program-fault-negative-control"
	cases = append(cases, faulty)

	// Genuine selected-policy requests race external target authority stimuli.
	for _, operation := range []kind{cancel, retire} {
		b = traceBuilder{Config: supervisionFixture(false)}
		supervisorInit(&b, "inner", 1)
		b.local("worker", 1, "")
		terminalDecision(&b, "inner", "worker", 1, 1, "")
		b.add(event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7})
		v := makeSpec("D_policy_failure_"+string(operation), b, []event{life(operation, "worker", 1), {Kind: decideLifecycle, ID: id}, prep("worker", 2, ""), cm})
		v.WantWitnesses = []string{"decision-accepted-before-" + string(operation)}
		if operation == cancel {
			v.WantWitnesses = append(v.WantWitnesses, "CancelRun-before-decision")
		} else {
			v.WantWitnesses = append(v.WantWitnesses, "RetireSubject-before-rejected-decision")
		}
		cases = append(cases, v)
	}
	// After outer accepts the inner successor, both old and current public
	// outcomes compete at its otherwise eligible ordinary terminal receive.
	b = recoveredSeed(true)
	b.local("worker", 2, "")
	terminalDecision(&b, "inner", "worker", 1, 2, "")
	terminalDecision(&b, "outer", "inner", 1, 1, "")
	lifecycleDispatch(&b, outerID, true)
	acceptLifecycle(&b, "outer", 1, outerID)
	supervisorInit(&b, "inner", 2)
	// The new inner initialization declares child generation 1 explicitly; its
	// old failed child outcome remains valid evidence for this new obligation.
	terminalDecision(&b, "inner", "worker", 2, 1, "new-obligation")
	newInnerID := intentID("inner", 2, 10)
	lifecycleDispatch(&b, newInnerID, true)
	b.local("inner", 2, resultID(newInnerID, ""))
	b.local("inner", 2, "")
	b.local("inner", 2, "")
	nestedOld := makeSpec("H_nested_late_duplicate_current_outcome", b, []event{{Kind: deliver, A: "inner", G: 1, ID: "late"}, {Kind: deliver, A: "inner", G: 2}, prep("outer", 1, terminalID("inner", 1, "late")), prep("outer", 1, terminalID("inner", 2, "")), cm})
	nestedOld.WantWitnesses = []string{"stale-terminal-blocked-current-consumed"}
	nestedOld.WantRejects = []string{"terminal correlation"}
	cases = append(cases, nestedOld)

	return cases
}

// Only witness bookkeeping uses this identity; runtime/reducer code cannot
// consult it. Canonical tuple encoding keeps run and operation unambiguous.
func pendingWitnessKey(r *run) string {
	if r == nil || r.Projection.Local == nil || r.Projection.Local.Pending == "" {
		return ""
	}
	return canonical([2]string{runID(r.A, r.G), r.Projection.Local.Pending})
}
