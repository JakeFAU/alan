package semantics

import (
	"crypto/sha256"
	"fmt"
	"slices"
)

type compositionSpec struct {
	Category                   string `json:",omitempty"`
	Name                       string
	Config                     config
	Seed, Alphabet             []event
	Depth, StateCap            int
	WantViolation              string
	WantWitnesses, WantRejects []string
}
type compositionStats struct {
	Name, Category, ConfigID, ScenarioID, SourceID              string
	Config                                                      config
	Seed                                                        []event
	SeedLength                                                  int
	Alphabet                                                    []event
	DepthLimit, StateCap                                        int
	States, Transitions, Rejected, Revisits, Boundary, MaxDepth int
	Rejections                                                  map[string]int
	Witnesses                                                   map[string][]event
	Violation                                                   string
	Trace                                                       []event
}
type compositionNode struct {
	Machine machine
	Monitor monitor
	Trace   []event
}

func digest(v any) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(canonical(v)))) }
func compositionKey(n compositionNode) string {
	return digest(struct {
		Machine machine
		Monitor monitor
	}{n.Machine, n.Monitor})
}
func compositionFixture(nested bool) config {
	leaf := []instruction{
		{Op: receive, Tag: "request", Mode: "capture", Reg: 1},
		{Op: emit, Tag: "unit", Value: 1, Mode: "effect"},
		{Op: receive, Tag: "receipt:unit", Receipt: true},
		{Op: send, Tag: "result", Mode: "reply", FromReg: true},
		{Op: send, Tag: "result", Mode: "reply", FromReg: true},
		{Op: success},
	}
	c := config{Version: "v1b", Programs: map[string][]instruction{
		"outer":    {{Op: send, Tag: "request", Target: "provider", Mode: "request", Key: "left"}, {Op: send, Tag: "request", Target: "peer", Mode: "request", Key: "right", Slot: 1, Ref: 1}, {Op: receive, Tag: "result", Mode: "result"}, {Op: receive, Tag: "result", Mode: "result"}, {Op: branch, Value: 2, PC: 6}, {Op: fault}, {Op: success}},
		"provider": copyOf(leaf), "peer": copyOf(leaf),
	}, Routes: map[string]string{"outer": "caller", "provider": "audit", "peer": "audit", "worker": "audit"}, Edges: map[string]bool{"outer>provider": true, "outer>peer": true, "provider>outer": true, "peer>outer": true, "provider>worker": true, "worker>provider": true, "env>outer": true}, Inputs: map[string]message{}, Bounds: bounds{Generations: 2, Epochs: 2, Commits: 12, Continuations: 2, Proposals: 2, Intents: 3, Attempts: 2, Inbox: 8, Value: 8}, Idempotent: true,
		Composition: &compositionContract{Outer: "outer", BranchLeaves: map[string]string{"left": "provider", "right": "peer"}, UnitTag: "unit", UnitValue: 1}}
	if nested {
		c.Programs["worker"] = copyOf(leaf)
		c.Composition.BranchLeaves["left"] = "worker"
		c.Programs["provider"] = []instruction{{Op: receive, Tag: "request", Mode: "capture", Reg: 1}, {Op: send, Tag: "request", Target: "worker", Mode: "request", Key: "inner"}, {Op: receive, Tag: "result", Mode: "result"}, {Op: send, Tag: "result", Mode: "reply", FromReg: true}, {Op: send, Tag: "result", Mode: "reply", FromReg: true}, {Op: success}}
	}
	return c
}

type traceBuilder struct {
	Config config
	Events []event
}

func (b *traceBuilder) add(es ...event) { b.Events = append(b.Events, es...) }
func (b *traceBuilder) local(a string, g int, obs string) {
	b.add(event{Kind: prepare, A: a, G: g, C: 1, Observation: obs}, event{Kind: commit})
}
func transportEvents(id string, isSend, receipt bool) []event {
	es := []event{{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7}, {Kind: applyExternal, ID: id, Attempt: 1}}
	if isSend {
		es = append(es, event{Kind: admit, Origin: "send", ID: id})
	}
	if receipt {
		es = append(es, event{Kind: recordReceipt, ID: id, Attempt: 1})
	}
	return es
}
func (b *traceBuilder) dispatch(id string, isSend, receipt bool) {
	b.add(transportEvents(id, isSend, receipt)...)
}
func (b *traceBuilder) requests(g int) {
	b.local("outer", g, "")
	b.local("outer", g, "")
	b.dispatch(intentID("outer", g, 1), true, false)
	b.dispatch(intentID("outer", g, 2), true, false)
}
func (b *traceBuilder) leaf(a string, g int, request string, admitReply bool) {
	b.local(a, g, sendObservationID(request))
	b.local(a, g, "")
	b.dispatch(intentID(a, g, 2), false, true)
	b.local(a, g, "receipt:"+intentID(a, g, 2))
	b.local(a, g, "")
	if admitReply {
		b.dispatch(intentID(a, g, 4), true, false)
	} else {
		b.add(transportEvents(intentID(a, g, 4), true, false)[:2]...)
	}
}
func (b *traceBuilder) providers(g int, admitReply bool) {
	if _, ok := b.Config.Programs["worker"]; ok {
		b.local("provider", g, sendObservationID(intentID("outer", g, 1)))
		b.local("provider", g, "")
		b.dispatch(intentID("provider", g, 2), true, false)
		b.leaf("worker", g, intentID("provider", g, 2), true)
		b.local("provider", g, sendObservationID(intentID("worker", g, 4)))
		b.local("provider", g, "")
		if admitReply {
			b.dispatch(intentID("provider", g, 4), true, false)
		} else {
			b.add(transportEvents(intentID("provider", g, 4), true, false)[:2]...)
		}
	} else {
		b.leaf("provider", g, intentID("outer", g, 1), admitReply)
	}
	b.leaf("peer", g, intentID("outer", g, 2), admitReply)
}
func replayCompositionSeed(spec compositionSpec) (compositionNode, error) {
	n := compositionNode{Machine: newMachine(spec.Config), Monitor: freshMonitor()}
	for i, e := range spec.Seed {
		after, why := step(n.Machine, e)
		if why != "" {
			return n, fmt.Errorf("seed[%d] %s rejected %s", i, e.String(), why)
		}
		mon, bad := inspect(n.Machine, after, e, n.Monitor)
		if bad != "" {
			return n, fmt.Errorf("seed[%d] violated %s", i, bad)
		}
		n.Machine = after
		n.Monitor = mon
	}
	return n, nil
}
func compositionWitness(before, after machine, e event, why string, mon *monitor) []string {
	if mon.Composition == nil {
		mon.Composition = freshCompositionFacts()
	}
	f := mon.Composition
	var found []string
	outer := after.Config.Composition.Outer
	rid := runID(outer, after.Subjects[outer].Current)
	r := after.Runs[rid]
	if why == "request correlation" && e.A == outer && e.G == 2 {
		o := before.Runs[rid].Inbox[e.Observation]
		if decodeEnvelope(o.Envelope).Generation == 1 && len(r.Projection.Pending) == 2 && len(r.Projection.Accepted) == 0 && r.Projection.Status == "active" {
			f.RejectedOld[rid] = true
		}
	}
	if why == "logical duplicate" {
		f.RejectedDuplicate[runID(e.A, e.G)] = true
	}
	if r.G == 2 && len(r.Projection.Accepted) > 0 && f.RejectedOld[rid] {
		found = append(found, "old-result-rejected-then-current-contributed")
	}
	if len(r.Projection.Accepted) == 2 && f.RejectedDuplicate[rid] {
		found = append(found, "distinct-transport-duplicate-blocked-and-other-branch-contributed")
	}
	for aid, ad := range after.Admissions {
		rr := after.Runs[runID(ad.Message.Target, after.Subjects[ad.Message.Target].Current)]
		if e.Kind == admit && aid == transportID(e.Origin, e.ID) && !ad.Ack && ad.Bound != runID(rr.A, rr.G) && rr.G == 2 {
			f.LostAckRetry[rr.A] = true
		}
		if rr.G == 2 && rr.N > 0 && f.LostAckRetry[rr.A] {
			found = append(found, "lost-ack-retransmission-and-successor-commit")
			if old := after.Intents[ad.RawID]; ad.Origin == "send" && old != nil && old.Settled && ad.Ack {
				found = append(found, "lost-ack-retry-successor-and-old-send-settlement")
			}
		}
	}
	for id, in := range after.Intents {
		d := in.Descriptor
		rr := after.Runs[runID(d.A, d.G)]
		if d.Kind == emit && !in.Settled && rr.Projection.Status != "active" && after.World.Keys[id] == d.Value {
			f.UnresolvedClosure[id] = true
		}
		current := after.Runs[runID(d.A, after.Subjects[d.A].Current)]
		if f.UnresolvedClosure[id] && current.G > d.G && current.N > 0 && in.Settled && after.World.Keys[id] == d.Value {
			found = append(found, "applied-unresolved-closure-retry-successor-and-late-receipt")
		}
	}
	if e.Kind == commit && why == "" {
		p := before.Proposals[e.Slot]
		if p != nil && p.A == "peer" && p.Intent != nil && p.Intent.Kind == emit {
			inner := after.Runs[runID("provider", 1)]
			for _, ct := range inner.Projection.Frontier {
				if after.Config.Programs["provider"][ct.PC].Mode == "result" && len(inner.Projection.Accepted) == 0 {
					found = append(found, "inner-waiting-peer-commit")
				}
			}
		}
		if p != nil && p.Observed != nil && len(after.Runs[runID(p.A, p.G)].Projection.Saved) > 0 {
			found = append(found, "request-captured")
		}
		if p != nil && p.Observed != nil && p.A == outer {
			found = append(found, "result-consumed-before-closure")
		}
	}
	if e.Kind == commit && why == "closed" {
		p := before.Proposals[e.Slot]
		if p != nil && p.Observed != nil && !before.Runs[runID(p.A, p.G)].Inbox[p.Observed.ID].Consumed {
			found = append(found, "prepared-result-closed-stale-commit-pending")
		}
	}
	if e.Kind == admit && why == "" && e.Origin == "send" {
		found = append(found, "generated-admission")
	}
	return found
}
func exploreComposition(spec compositionSpec) (compositionStats, error) {
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
		if depth > st.MaxDepth {
			st.MaxDepth = depth
		}
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
			for _, w := range compositionWitness(n.Machine, after, e, why, &mon) {
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
func compositionScenarios() []compositionSpec {
	c := compositionFixture(true)
	makeSpec := func(name string, b traceBuilder, es []event) compositionSpec {
		return compositionSpec{Name: name, Config: b.Config, Seed: b.Events, Alphabet: es, Depth: 10, StateCap: 5000}
	}
	prep := func(a string, g int, obs string) event {
		return event{Kind: prepare, A: a, G: g, C: 1, Observation: obs}
	}
	commit0 := event{Kind: commit}
	b := traceBuilder{Config: c}
	fresh := makeSpec("fresh_sender_receiver", b, append([]event{prep("outer", 1, ""), commit0}, transportEvents(intentID("outer", 1, 1), true, false)...))
	fresh.Alphabet = append(fresh.Alphabet, prep("provider", 1, sendObservationID(intentID("outer", 1, 1))))
	fresh.Depth = 8
	fresh.WantWitnesses = []string{"generated-admission", "request-captured"}
	b.requests(1)
	b.local("provider", 1, sendObservationID(intentID("outer", 1, 1)))
	b.local("provider", 1, "")
	wait := makeSpec("inner_wait_peer_ready", b, []event{prep("provider", 1, ""), prep("peer", 1, sendObservationID(intentID("outer", 1, 2))), prep("peer", 1, ""), commit0, {Kind: interrupt}})
	wait.WantWitnesses = []string{"inner-waiting-peer-commit"}
	b = traceBuilder{Config: compositionFixture(false)}
	b.requests(1)
	b.providers(1, true)
	closed := makeSpec("admitted_result_closure", b, []event{prep("outer", 1, sendObservationID(intentID("provider", 1, 4))), commit0, life(abandon, "outer", 1)})
	closed.WantWitnesses = []string{"result-consumed-before-closure", "prepared-result-closed-stale-commit-pending"}
	closed.WantRejects = []string{"closed"}
	b.local("provider", 1, "")
	b.dispatch(intentID("provider", 1, 5), true, false)
	dup := makeSpec("distinct_transport_duplicate", b, []event{prep("outer", 1, sendObservationID(intentID("provider", 1, 4))), prep("outer", 1, sendObservationID(intentID("provider", 1, 5))), prep("outer", 1, sendObservationID(intentID("peer", 1, 4))), commit0})
	dup.WantWitnesses = []string{"distinct-transport-duplicate-blocked-and-other-branch-contributed"}
	dup.WantRejects = []string{"logical duplicate"}
	b = traceBuilder{Config: compositionFixture(false)}
	b.requests(1)
	b.providers(1, false)
	b.add(life(abandon, "outer", 1), life(retry, "outer", 1), life(abandon, "provider", 1), life(retry, "provider", 1), life(abandon, "peer", 1), life(retry, "peer", 1))
	b.requests(2)
	b.providers(2, false)
	old := makeSpec("old_result_eligible_generation2", b, []event{{Kind: admit, Origin: "send", ID: intentID("provider", 1, 4)}, {Kind: admit, Origin: "send", ID: intentID("provider", 2, 4)}, prep("outer", 2, sendObservationID(intentID("provider", 1, 4))), prep("outer", 2, sendObservationID(intentID("provider", 2, 4))), commit0})
	old.WantWitnesses = []string{"old-result-rejected-then-current-contributed"}
	old.WantRejects = []string{"request correlation"}
	b = traceBuilder{Config: compositionFixture(false)}
	b.requests(1)
	b.add(life(abandon, "outer", 1), life(retry, "outer", 1))
	b.local("outer", 2, "")
	b.add(transportEvents(intentID("outer", 2, 1), true, false)[:2]...)
	ack := makeSpec("lost_ack_retry_fresh_work", b, []event{
		life(abandon, "provider", 1), life(retry, "provider", 1),
		{Kind: admit, Origin: "send", ID: intentID("outer", 1, 1)},
		{Kind: admit, Origin: "send", ID: intentID("outer", 2, 1)},
		prep("provider", 2, sendObservationID(intentID("outer", 2, 1))), commit0, {Kind: interrupt},
		{Kind: recordReceipt, ID: intentID("outer", 1, 1), Attempt: 1},
		{Kind: acknowledge, Origin: "send", ID: intentID("outer", 1, 1)},
	})
	ack.WantWitnesses = []string{"lost-ack-retransmission-and-successor-commit", "lost-ack-retry-successor-and-old-send-settlement"}
	b = traceBuilder{Config: compositionFixture(false)}
	b.requests(1)
	b.local("provider", 1, sendObservationID(intentID("outer", 1, 1)))
	b.local("provider", 1, "")
	b.dispatch(intentID("provider", 1, 2), false, false)
	b.add(life(abandon, "outer", 1), life(retry, "outer", 1))
	b.local("outer", 2, "")
	b.add(transportEvents(intentID("outer", 2, 1), true, false)[:2]...)
	unresolved := makeSpec("applied_unresolved_close_retry", b, []event{life(abandon, "provider", 1), life(retry, "provider", 1), {Kind: admit, Origin: "send", ID: intentID("outer", 2, 1)}, prep("provider", 2, sendObservationID(intentID("outer", 2, 1))), commit0, {Kind: recordReceipt, ID: intentID("provider", 1, 2), Attempt: 1}})
	unresolved.WantWitnesses = []string{"applied-unresolved-closure-retry-successor-and-late-receipt"}
	origin := copyOf(dup)
	origin.Name = "environment_origin"
	origin.Seed = origin.Seed[:len(origin.Seed)-5]
	env := envelope{Caller: "outer", Generation: 1, Request: 1, Key: "left", ReplyTo: "outer"}
	origin.Config.Inputs["spoof"] = message{Source: "provider", Target: "outer", Tag: "result", Value: 1, Envelope: canonical(env), SenderG: 1}
	origin.Config.Edges["provider>outer"] = true
	origin.Alphabet = []event{{Kind: admit, ID: "spoof"}, prep("outer", 1, messageObservationID("spoof")), commit0}
	origin.WantWitnesses = nil
	origin.WantRejects = []string{"origin"}
	cases := []compositionSpec{fresh, wait, ack, closed, old, unresolved, dup, origin}
	for _, entry := range []struct {
		spec              compositionSpec
		mutant, violation string
	}{{old, "generation-correlation", "composition-generation-correlation"}, {dup, "logical-duplicate", "composition-logical-duplicate"}, {origin, "origin", "composition-origin"}} {
		s := copyOf(entry.spec)
		s.Name = "broken_" + entry.mutant
		s.Config.Mutant = entry.mutant
		s.WantViolation = entry.violation
		s.WantWitnesses = nil
		s.WantRejects = nil
		cases = append(cases, s)
	}

	// Repeat the safety windows through the recursive implementation, retaining
	// the identical public contract and monitor rather than only a completion run.
	b = traceBuilder{Config: compositionFixture(true)}
	b.requests(1)
	b.providers(1, true)
	nestedClosed := copyOf(closed)
	nestedClosed.Name = "nested_admitted_result_closure"
	nestedClosed.Config = b.Config
	nestedClosed.Seed = slices.Clone(b.Events)
	b.local("provider", 1, "")
	b.dispatch(intentID("provider", 1, 5), true, false)
	nestedDup := copyOf(dup)
	nestedDup.Name = "nested_distinct_transport_duplicate"
	nestedDup.Config = b.Config
	nestedDup.Seed = slices.Clone(b.Events)
	b = traceBuilder{Config: compositionFixture(true)}
	b.requests(1)
	b.providers(1, false)
	for _, a := range []string{"outer", "provider", "peer", "worker"} {
		b.add(life(abandon, a, 1), life(retry, a, 1))
	}
	b.requests(2)
	b.providers(2, false)
	nestedOld := copyOf(old)
	nestedOld.Name = "nested_old_result_eligible_generation2"
	nestedOld.Config = b.Config
	nestedOld.Seed = slices.Clone(b.Events)
	cases = append(cases, nestedClosed, nestedDup, nestedOld)
	for _, entry := range []struct {
		spec              compositionSpec
		mutant, violation string
	}{{nestedDup, "logical-duplicate", "composition-logical-duplicate"}, {nestedOld, "generation-correlation", "composition-generation-correlation"}} {
		bad := copyOf(entry.spec)
		bad.Name = "broken_" + bad.Name
		bad.Config.Mutant = entry.mutant
		bad.WantViolation = entry.violation
		bad.WantWitnesses = nil
		bad.WantRejects = nil
		cases = append(cases, bad)
	}
	faultConfig, faultTrace := compositionFanoutCounterexample()
	cases = append(cases, compositionSpec{Name: "faulty_program_nested_resource_fanout", Category: "program-fault-negative-control", Config: faultConfig, Seed: faultTrace[:len(faultTrace)-1], Alphabet: []event{{Kind: commit}, faultTrace[len(faultTrace)-1]}, Depth: 10, StateCap: 5000, WantViolation: "composition-resource-cardinality"})
	return cases
}

// Retains the independent reviewer's exact bounded faulty-program configuration
// and 37-event trace: two downstream requests cause two effects for one root.
func compositionFanoutCounterexample() (config, []event) {
	c := compositionFixture(true)
	c.Programs["provider"] = []instruction{
		{Op: receive, Tag: "request", Mode: "capture", Reg: 1},
		{Op: send, Tag: "request", Target: "worker", Mode: "request", Key: "inner"},
		{Op: send, Tag: "request", Target: "worker", Mode: "request", Key: "inner", Slot: 1, Ref: 1},
		{Op: receive, Tag: "result", Mode: "result"},
		{Op: send, Tag: "result", Mode: "reply", FromReg: true},
		{Op: success},
	}
	c.Programs["worker"] = []instruction{
		{Op: receive, Tag: "request", Mode: "capture", Reg: 1},
		{Op: emit, Tag: "unit", Value: 1, Mode: "effect"},
		{Op: receive, Tag: "receipt:unit", Receipt: true},
		{Op: receive, Tag: "request", Mode: "capture", Slot: 1, Reg: 1},
		{Op: emit, Tag: "unit", Value: 1, Mode: "effect", Slot: 1},
		{Op: receive, Tag: "receipt:unit", Receipt: true},
		{Op: success},
	}
	b := traceBuilder{Config: c}
	b.requests(1)
	b.local("provider", 1, sendObservationID(intentID("outer", 1, 1)))
	b.local("provider", 1, "")
	b.dispatch(intentID("provider", 1, 2), true, false)
	b.local("worker", 1, sendObservationID(intentID("provider", 1, 2)))
	b.local("worker", 1, "")
	b.dispatch(intentID("worker", 1, 2), false, true)
	b.local("worker", 1, "receipt:"+intentID("worker", 1, 2))
	b.local("provider", 1, "")
	b.dispatch(intentID("provider", 1, 3), true, false)
	b.local("worker", 1, sendObservationID(intentID("provider", 1, 3)))
	b.local("worker", 1, "")
	b.dispatch(intentID("worker", 1, 5), false, false)
	return c, b.Events
}
