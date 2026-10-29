package substitution

import "fmt"

// JoinObservation keeps both original outcomes and the enclosing result. Fixed
// provisioning is separate from cumulative dynamic admission spending.
type JoinObservation struct {
	Candidate, S                 Observation
	CandidateConsumed, SConsumed bool
	ParentStage                  int
	ParentStatus, RootStatus     string
	I, R, E, FixedRuns           int
}

func ObserveJoin(s JoinState) JoinObservation {
	result := JoinObservation{ParentStage: s.ParentStage, ParentStatus: s.ParentStatus, I: s.Component.I, R: s.Component.R, E: s.Component.E, FixedRuns: 1}
	if result.ParentStatus == "" {
		result.ParentStatus = "open"
	}
	if s.RootPublished {
		result.RootStatus = s.ParentStatus
	}
	result.Candidate = observable(s.Component, entry, "outcome")
	result.S = observable(s.Component, sibling, "outcome")
	result.CandidateConsumed, result.SConsumed = s.Component.Ops[entry].Consumed, s.Component.Ops[sibling].Consumed
	return result
}

type JoinCounterexample struct {
	Code, Detail string
	Trace        []JoinEvent
}

type JoinMaximum struct {
	Kind        string
	State       JoinState
	Observation JoinObservation
	Guarantee   []Violation
	Trace       []JoinEvent
}

// States counts model x independent-prefix-monitor states. ModelStates reports
// distinct model states separately; no representative-path monitor assumption
// is needed. Cycles is zero only when every edge passes strict intrinsic rank.
type JoinCensus struct {
	Config                                                         JoinConfig
	Variant                                                        Variant
	Entitled                                                       bool
	States, ModelStates, Edges, Revisits, Rejections               int
	Terminal, Pending, Stuck, Cycles, Cutoffs                      int
	RankMinimum, RankMaximum, RankCap                              int
	RankFailures, StateFailures, PrefixFailures, ComponentFailures int
	GuaranteeFailures                                              int
	Maxima                                                         []JoinMaximum
	Counterexamples                                                []JoinCounterexample
}

type joinSearchKey struct {
	State  JoinState
	Prefix joinPrefixState
}

type joinSearchNode struct {
	Key    joinSearchKey
	Parent int
	Event  JoinEvent
}

func joinSearchPath(nodes []joinSearchNode, index int, last *JoinEvent) []JoinEvent {
	path := []JoinEvent{}
	if last != nil {
		path = append(path, *last)
	}
	for index > 0 {
		path = append(path, nodes[index].Event)
		index = nodes[index].Parent
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func joinMaximalKind(s JoinState) string {
	if s.ParentStage == 6 && s.RootPublished {
		return "terminal"
	}
	if s.ParentStatus == "" && s.Component.Ops[entry].PC == pending && !s.Component.Ops[entry].Published && s.Component.Ops[sibling].Published && s.Component.Ops[sibling].Delivered {
		return "pending"
	}
	return "stuck"
}

// ExploreJoin has no scheduler cutoff and does not prune failed states. Call only
// after the contract/reference/bounds freeze is independently reviewed. Every
// generated edge is checked before deduplication, including revisited targets.
// Idle, idempotent retry and physical restart are stutters of this retained state;
// eventual eligible service excludes an infinite suffix containing only stutters.
func ExploreJoin(c JoinConfig, variant Variant) (JoinCensus, error) {
	result := JoinCensus{Config: c, Variant: variant, Entitled: JoinEntitled(c), RankCap: JoinRankCap, Maxima: []JoinMaximum{}, Counterexamples: []JoinCounterexample{}}
	if err := ValidateJoinConfig(c); err != nil {
		return result, err
	}
	if variant != Reference && variant != Local && variant != Helper {
		return result, fmt.Errorf("unknown variant %q", variant)
	}
	initial := JoinInitial(c)
	nodes := []joinSearchNode{{Key: joinSearchKey{State: initial}, Parent: -1}}
	seen := map[joinSearchKey]bool{nodes[0].Key: true}
	models := map[JoinState]bool{initial: true}
	failures := map[string]bool{}
	result.RankMinimum, result.RankMaximum = JoinRank(initial), JoinRank(initial)
	report := func(code, detail string, index int, last *JoinEvent) {
		if !failures[code] {
			failures[code] = true
			result.Counterexamples = append(result.Counterexamples, JoinCounterexample{Code: code, Detail: detail, Trace: joinSearchPath(nodes, index, last)})
		}
	}
	if err := CheckJoinState(c, initial); err != nil {
		result.StateFailures++
		report("state", err.Error(), 0, nil)
	}
	for index := 0; index < len(nodes); index++ {
		node := nodes[index]
		state := node.Key.State
		edges := JoinSuccessors(c, variant, state)
		if len(edges) == 0 {
			kind := joinMaximalKind(state)
			switch kind {
			case "terminal":
				result.Terminal++
			case "pending":
				result.Pending++
			default:
				result.Stuck++
				report("stuck", "maximal state is neither completed nor declared pending", index, nil)
			}
			trace := joinSearchPath(nodes, index, nil)
			guarantee := CheckJoinGuarantee(c, trace)
			if result.Entitled && len(guarantee) != 0 {
				result.GuaranteeFailures++
				report("guarantee", fmt.Sprint(guarantee), index, nil)
			}
			result.Maxima = append(result.Maxima, JoinMaximum{Kind: kind, State: state, Observation: ObserveJoin(state), Guarantee: guarantee, Trace: trace})
		}
		for _, edge := range edges {
			result.Edges++
			if edge.Event.Fact.Kind == "admit" && edge.Event.Fact.Status == "rejected" {
				result.Rejections++
			}
			rank := JoinRank(edge.Next)
			if rank <= JoinRank(state) || rank > JoinRankCap {
				result.RankFailures++
				report("rank", "intrinsic rank did not increase within cap", index, &edge.Event)
			}
			if rank < result.RankMinimum {
				result.RankMinimum = rank
			}
			if rank > result.RankMaximum {
				result.RankMaximum = rank
			}
			if err := CheckJoinState(c, edge.Next); err != nil {
				result.StateFailures++
				report("state", err.Error(), index, &edge.Event)
			}
			trace := joinSearchPath(nodes, index, &edge.Event)
			prefix, violations := advanceJoinPrefix(c, node.Key.Prefix, edge.Event, len(trace)-1)
			for _, violation := range violations {
				result.PrefixFailures++
				report("prefix:"+violation.Code, violation.Detail, index, &edge.Event)
			}
			// All CheckTrace memory (commits, admissions, replies, publications,
			// consumptions) plus the mode/helper discipline is retained in Prefix.
			// Thus histories merged by Key cannot differ on a future component
			// verdict. Past violations were checked before that merge and retained.
			for _, violation := range CheckJoinComponent(c, trace) {
				if violation.Index != len(trace)-1 {
					continue
				}
				result.ComponentFailures++
				report("component:"+violation.Code, violation.Detail, index, &edge.Event)
			}
			key := joinSearchKey{State: edge.Next, Prefix: prefix}
			if seen[key] {
				result.Revisits++
				continue
			}
			seen[key], models[edge.Next] = true, true
			nodes = append(nodes, joinSearchNode{Key: key, Parent: index, Event: edge.Event})
		}
	}
	result.States, result.ModelStates = len(nodes), len(models)
	if result.RankFailures != 0 {
		result.Cycles = -1
	} // Unresolved, never called acyclic.
	return result, nil
}
