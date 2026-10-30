package substitution

import "fmt"

type ConsumerReasonObservation struct {
	JoinObservation
	Classification string
}

func ObserveConsumerReason(s ConsumerReasonState) ConsumerReasonObservation {
	return ConsumerReasonObservation{JoinObservation: ObserveJoin(s.JoinState), Classification: s.Classification}
}

type ConsumerReasonCounterexample struct {
	Code, Detail string
	Trace        []ConsumerReasonEvent
}

type ConsumerReasonMaximum struct {
	Kind        string
	State       ConsumerReasonState
	Observation ConsumerReasonObservation
	Guarantee   []Violation
	Trace       []ConsumerReasonEvent
}

// ConsumerReasonCensus is distinct from historical censuses. All-mode entitlement
// is explicit; pending maxima are checked against G_outcome rather than rejected.
type ConsumerReasonCensus struct {
	Config                                                         JoinConfig
	Variant                                                        Variant
	Entitled                                                       bool
	States, ModelStates, Edges, Revisits, Rejections               int
	Terminal, Pending, Stuck, Cycles, Cutoffs                      int
	RankMinimum, RankMaximum, RankCap                              int
	RankFailures, StateFailures, PrefixFailures, ComponentFailures int
	GuaranteeFailures                                              int
	Maxima                                                         []ConsumerReasonMaximum
	Counterexamples                                                []ConsumerReasonCounterexample
}

type reasonSearchKey struct {
	State  ConsumerReasonState
	Prefix reasonPrefixState
}

type reasonSearchNode struct {
	Key    reasonSearchKey
	Parent int
	Event  ConsumerReasonEvent
}

func reasonSearchPath(nodes []reasonSearchNode, index int, last *ConsumerReasonEvent) []ConsumerReasonEvent {
	path := []ConsumerReasonEvent{}
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

func reasonMaximalKind(s ConsumerReasonState) string {
	if s.ParentStage == 6 && s.RootPublished {
		return "terminal"
	}
	if s.Classification == "" && s.Decision == "" && !s.RootPublished &&
		joinPolicyMaximalKind(s.JoinState, consumeSFirst) == "pending" {
		return "pending"
	}
	return "stuck"
}

// ExploreConsumerReason must run only after independent reference review. It
// explores all model x monitor states without a cutoff or failure pruning,
// checks every generated edge before deduplication, and retains every maximum.
// A failed rank leaves cycles unresolved. Eventual eligible service excludes
// infinite idle/retry/restart-only suffixes; no pending release is supplied.
func ExploreConsumerReason(c JoinConfig, variant Variant) (ConsumerReasonCensus, error) {
	result := ConsumerReasonCensus{
		Config: c, Variant: variant, Entitled: ConsumerReasonEntitled(c), RankCap: ConsumerReasonRankCap,
		Maxima: []ConsumerReasonMaximum{}, Counterexamples: []ConsumerReasonCounterexample{},
	}
	if err := ValidateJoinConfig(c); err != nil {
		return result, err
	}
	if variant != Reference && variant != Local && variant != Helper {
		return result, fmt.Errorf("unknown variant %q", variant)
	}
	initial := ConsumerReasonInitial(c)
	nodes := []reasonSearchNode{{Key: reasonSearchKey{State: initial}, Parent: -1}}
	seen := map[reasonSearchKey]bool{nodes[0].Key: true}
	models := map[ConsumerReasonState]bool{initial: true}
	failures := map[string]bool{}
	result.RankMinimum, result.RankMaximum = ConsumerReasonRank(initial), ConsumerReasonRank(initial)
	report := func(code, detail string, index int, last *ConsumerReasonEvent) {
		if !failures[code] {
			failures[code] = true
			result.Counterexamples = append(result.Counterexamples, ConsumerReasonCounterexample{
				Code: code, Detail: detail, Trace: reasonSearchPath(nodes, index, last),
			})
		}
	}
	if err := CheckConsumerReasonState(c, initial); err != nil {
		result.StateFailures++
		report("state", err.Error(), 0, nil)
	}
	for index := 0; index < len(nodes); index++ {
		node := nodes[index]
		state := node.Key.State
		edges := ConsumerReasonSuccessors(c, variant, state)
		if len(edges) == 0 {
			kind := reasonMaximalKind(state)
			switch kind {
			case "terminal":
				result.Terminal++
			case "pending":
				result.Pending++
			default:
				result.Stuck++
				report("stuck", "maximum is neither completed nor designated pending", index, nil)
			}
			trace := reasonSearchPath(nodes, index, nil)
			guarantee := CheckConsumerReasonGuarantee(c, trace)
			if result.Entitled && len(guarantee) != 0 {
				result.GuaranteeFailures++
				report("guarantee", fmt.Sprint(guarantee), index, nil)
			}
			result.Maxima = append(result.Maxima, ConsumerReasonMaximum{
				Kind: kind, State: state, Observation: ObserveConsumerReason(state), Guarantee: guarantee, Trace: trace,
			})
		}
		for _, edge := range edges {
			result.Edges++
			if edge.Event.Fact.Kind == "admit" && edge.Event.Fact.Status == "rejected" {
				result.Rejections++
			}
			rank := ConsumerReasonRank(edge.Next)
			if rank <= ConsumerReasonRank(state) || rank > ConsumerReasonRankCap {
				result.RankFailures++
				report("rank", "intrinsic rank did not increase within cap", index, &edge.Event)
			}
			result.RankMinimum = min(result.RankMinimum, rank)
			result.RankMaximum = max(result.RankMaximum, rank)
			if err := CheckConsumerReasonState(c, edge.Next); err != nil {
				result.StateFailures++
				report("state", err.Error(), index, &edge.Event)
			}
			trace := reasonSearchPath(nodes, index, &edge.Event)
			prefix, violations := advanceReasonPrefix(c, node.Key.Prefix, edge.Event, len(trace)-1)
			for _, violation := range violations {
				result.PrefixFailures++
				report("prefix:"+violation.Code, violation.Detail, index, &edge.Event)
			}
			// The old component monitor's memory is in Prefix.Ops; new parent
			// events do not enter it. Pending/classification memory is keyed too.
			for _, violation := range CheckConsumerReasonComponent(c, trace) {
				if violation.Index != len(trace)-1 {
					continue
				}
				result.ComponentFailures++
				report("component:"+violation.Code, violation.Detail, index, &edge.Event)
			}
			key := reasonSearchKey{State: edge.Next, Prefix: prefix}
			if seen[key] {
				result.Revisits++
				continue
			}
			seen[key], models[edge.Next] = true, true
			nodes = append(nodes, reasonSearchNode{Key: key, Parent: index, Event: edge.Event})
		}
	}
	result.States, result.ModelStates = len(nodes), len(models)
	if result.RankFailures != 0 {
		result.Cycles = -1
	}
	return result, nil
}
