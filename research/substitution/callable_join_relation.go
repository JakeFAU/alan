package substitution

import "fmt"

// callablePublicStateOf is the finite A/B witness of the supplied local R. The
// generic argument quantifies over an R, not over this particular erasure.
func callablePublicStateOf(s CallableJoinState, outer bool) CallableJoinState {
	s.Inner.Component = publicState(s.Inner.Component)
	if !outer {
		s.OuterDecision, s.OuterStatus, s.RootPublished = "", "", false
		s.OuterStage = 0
		if s.InvocationCommitted {
			s.OuterStage = 1
		}
		if s.Consumed {
			s.OuterStage = 2
		}
	}
	return s
}

// CallableJoinRelated separates the constructor relation under a common public
// caller environment from the stronger fixed-O corollary (outer=true).
func CallableJoinRelated(x, k CallableJoinState, outer bool) bool {
	return callablePublicStateOf(x, outer) == callablePublicStateOf(k, outer)
}

// CheckCallableJoinLiftStep checks the finite A/B single-event instance at one
// reachable state. It does not establish the generic weak-path/progress theorem.
func CheckCallableJoinLiftStep(c CallableJoinConfig, v Variant, s CallableJoinState) RelationResult {
	r := RelationResult{Pairs: 1, Counterexamples: []Counterexample{}}
	k := callablePublicStateOf(s, true)
	actual := CallableJoinSuccessors(c, v, s)
	reference := CallableJoinSuccessors(c, Reference, k)
	bad := func(code string, event Observation) {
		r.Failures++
		r.Counterexamples = append(r.Counterexamples, Counterexample{
			Code: code, Detail: "callable constructor event/readiness lacks related K1 successor",
			Trace: []Observation{event},
		})
	}
	for _, edge := range actual {
		if edge.Environment {
			r.EnvironmentChecks++
		} else {
			r.ComponentChecks++
		}
		matched := false
		if edge.Event.Fact.Kind == "private_step" && CallableJoinRelated(edge.Next, k, true) {
			matched = true
			r.Stutters++
		}
		for _, candidate := range reference {
			if edge.Event == candidate.Event && edge.Environment == candidate.Environment &&
				CallableJoinRelated(edge.Next, candidate.Next, true) {
				matched = true
			}
		}
		if !matched {
			bad("callable_simulation", edge.Event.Fact)
		}
	}
	for _, edge := range reference {
		if !edge.Environment {
			continue
		}
		r.EnvironmentChecks++
		matched := false
		for _, candidate := range actual {
			if edge.Event == candidate.Event && candidate.Environment &&
				CallableJoinRelated(candidate.Next, edge.Next, true) {
				matched = true
			}
		}
		if !matched {
			bad("callable_environment_readiness", edge.Event.Fact)
		}
	}
	return r
}

// CheckCallableJoinLifting is exhaustive finite-instance machinery, gated with
// Task3b replacement qualification. Progress is separately checked by the full
// model x monitor explorer and aligned local qualifier.
func CheckCallableJoinLifting(c CallableJoinConfig, v Variant) (RelationResult, error) {
	r := RelationResult{Counterexamples: []Counterexample{}}
	if !CallableJoinEntitled(c) || v != Local && v != Helper {
		return r, fmt.Errorf("outside finite callable replacement lifting domain")
	}
	initial := CallableJoinInitial(c)
	r.Initials = 1
	nodes := []callableSearchNode{{Key: callableSearchKey{State: initial}, Parent: -1}}
	seen := map[CallableJoinState]bool{initial: true}
	for index := 0; index < len(nodes); index++ {
		s := nodes[index].Key.State
		step := CheckCallableJoinLiftStep(c, v, s)
		r.Pairs += step.Pairs
		r.ComponentChecks += step.ComponentChecks
		r.EnvironmentChecks += step.EnvironmentChecks
		r.Stutters += step.Stutters
		r.Failures += step.Failures
		for _, failure := range step.Counterexamples {
			path := callableSearchPath(nodes, index, nil)
			facts := []Observation{}
			for _, event := range path {
				facts = append(facts, event.Fact)
			}
			failure.Trace = append(facts, failure.Trace...)
			r.Counterexamples = append(r.Counterexamples, failure)
		}
		for _, edge := range CallableJoinSuccessors(c, v, s) {
			if !seen[edge.Next] {
				seen[edge.Next] = true
				nodes = append(nodes, callableSearchNode{
					Key: callableSearchKey{State: edge.Next}, Parent: index, Event: edge.Event,
				})
			}
		}
	}
	return r, nil
}
