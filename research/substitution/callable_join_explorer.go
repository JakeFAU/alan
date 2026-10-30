package substitution

import "fmt"

type CallableJoinCounterexample struct {
	Code, Detail string
	Trace        []CallableJoinEvent
}

type CallableJoinMaximum struct {
	Kind            string
	State           CallableJoinState
	Contract, Outer []Violation
	Trace           []CallableJoinEvent
}

type CallableJoinCensus struct {
	Config                                                         CallableJoinConfig
	Variant                                                        Variant
	States, ModelStates, Edges, Revisits, Rejections               int
	Terminal, Pending, Stuck, Cutoffs, Cycles                      int
	StateFailures, RankFailures, PrefixFailures, ComponentFailures int
	ProjectionFailures, ContractFailures, OuterFailures            int
	RankMinimum, RankMaximum, RankCap                              int
	Maxima                                                         []CallableJoinMaximum
	Counterexamples                                                []CallableJoinCounterexample
}

type callableSearchKey struct {
	State  CallableJoinState
	Prefix callablePrefixState
}

type callableSearchNode struct {
	Key    callableSearchKey
	Parent int
	Event  CallableJoinEvent
}

func callableSearchPath(nodes []callableSearchNode, index int, last *CallableJoinEvent) []CallableJoinEvent {
	path := []CallableJoinEvent{}
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

func callableMaximalKind(s CallableJoinState) string {
	if s.RootPublished && s.OuterStage == 4 {
		return "terminal"
	}
	e := s.Inner.Component.Ops[entry]
	pending := s.Inner.ParentStage == 3 && s.Inner.Component.Ops[sibling].Consumed && e.PC == pending &&
		s.Inner.Classification == "" && s.Inner.Decision == "" && s.Inner.ParentStatus == "" &&
		!s.ReportCommitted && !s.Published && s.OuterStage == 1 && s.OuterDecision == "" && !s.RootPublished
	if pending {
		return "pending"
	}
	return "stuck"
}

// ExploreCallableJoin is gated until independent reference review. It checks all
// edges before deduplication and retains every maximum without failure pruning
// or a cutoff. One product supplies two separately checked guarantee projections.
func ExploreCallableJoin(c CallableJoinConfig, v Variant) (CallableJoinCensus, error) {
	r := CallableJoinCensus{Config: c, Variant: v, RankCap: CallableJoinRankCap,
		Maxima: []CallableJoinMaximum{}, Counterexamples: []CallableJoinCounterexample{}}
	if err := ValidateCallableJoinConfig(c); err != nil {
		return r, err
	}
	if v != Reference && v != Local && v != Helper {
		return r, fmt.Errorf("unknown callable variant %q", v)
	}
	initial := CallableJoinInitial(c)
	nodes := []callableSearchNode{{Key: callableSearchKey{State: initial}, Parent: -1}}
	seen := map[callableSearchKey]bool{nodes[0].Key: true}
	models := map[CallableJoinState]bool{initial: true}
	failures := map[string]bool{}
	r.RankMinimum, r.RankMaximum = CallableJoinRank(initial), CallableJoinRank(initial)
	report := func(code, detail string, index int, last *CallableJoinEvent) {
		if !failures[code] {
			failures[code] = true
			r.Counterexamples = append(r.Counterexamples, CallableJoinCounterexample{
				Code: code, Detail: detail, Trace: callableSearchPath(nodes, index, last),
			})
		}
	}
	if err := CheckCallableJoinState(c, initial); err != nil {
		r.StateFailures++
		report("state", err.Error(), 0, nil)
	}
	for index := 0; index < len(nodes); index++ {
		node := nodes[index]
		s := node.Key.State
		edges := CallableJoinSuccessors(c, v, s)
		if len(edges) == 0 {
			kind := callableMaximalKind(s)
			switch kind {
			case "terminal":
				r.Terminal++
			case "pending":
				r.Pending++
			default:
				r.Stuck++
				report("stuck", "maximum is neither terminal nor designated pending", index, nil)
			}
			trace := callableSearchPath(nodes, index, nil)
			contract := CheckCallableJoinContract(c, trace)
			outer := CheckCallableJoinOuterGuarantee(c, trace)
			if CallableJoinEntitled(c) && len(contract) != 0 {
				r.ContractFailures++
				report("contract", fmt.Sprint(contract), index, nil)
			}
			if CallableJoinEntitled(c) && len(outer) != 0 {
				r.OuterFailures++
				report("outer", fmt.Sprint(outer), index, nil)
			}
			r.Maxima = append(r.Maxima, CallableJoinMaximum{
				Kind: kind, State: s, Contract: contract, Outer: outer, Trace: trace,
			})
		}
		for _, edge := range edges {
			r.Edges++
			if edge.Event.Fact.Kind == "admit" && edge.Event.Fact.Status == "rejected" {
				r.Rejections++
			}
			rank := CallableJoinRank(edge.Next)
			r.RankMinimum, r.RankMaximum = min(r.RankMinimum, rank), max(r.RankMaximum, rank)
			if rank <= CallableJoinRank(s) || rank > CallableJoinRankCap {
				r.RankFailures++
				report("rank", "nonincreasing or out-of-bound intrinsic rank", index, &edge.Event)
			}
			if err := CheckCallableJoinState(c, edge.Next); err != nil {
				r.StateFailures++
				report("state", err.Error(), index, &edge.Event)
			}
			if CallableJoinEntitled(c) {
				if err := CheckCallableJoinProjectionEdge(c, v, s, edge); err != nil {
					r.ProjectionFailures++
					report("projection", err.Error(), index, &edge.Event)
				}
			}
			trace := callableSearchPath(nodes, index, &edge.Event)
			prefix, violations := advanceCallablePrefix(c, node.Key.Prefix, edge.Event, len(trace)-1)
			for _, violation := range violations {
				r.PrefixFailures++
				report("prefix:"+violation.Code, violation.Detail, index, &edge.Event)
			}
			for _, violation := range CheckCallableJoinComponent(c, trace) {
				if violation.Index == len(trace)-1 {
					r.ComponentFailures++
					report("component:"+violation.Code, violation.Detail, index, &edge.Event)
				}
			}
			key := callableSearchKey{State: edge.Next, Prefix: prefix}
			if seen[key] {
				r.Revisits++
				continue
			}
			seen[key], models[edge.Next] = true, true
			nodes = append(nodes, callableSearchNode{Key: key, Parent: index, Event: edge.Event})
		}
	}
	r.States, r.ModelStates = len(nodes), len(models)
	if r.RankFailures != 0 {
		r.Cycles = -1 // No unperformed SCC claim after a failed rank.
	}
	return r, nil
}
