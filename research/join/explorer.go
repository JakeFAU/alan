package join

import (
	"fmt"
	"github.com/JakeFAU/alan/research/substitution"
	"math/big"
	"sort"
)

type Witness struct {
	Code   string
	Events []Event
	Final  State
}
type Census struct {
	Name                                                                                  string
	InitialStates, States, Transitions, MaximalStates, Successful, Pending, Failed, Stuck int
	Cycles, RankFailures, Cutoffs, ResourceFailures, LifetimeFailures, ReportFailures     int
	DecisiveStates, EarlyReportStates, LateReportStates                                   int
	MaximalExecutions                                                                     string
	Witnesses                                                                             []Witness
}
type node struct {
	state  State
	parent int
	event  Event
	next   []int
}

func trace(nodes []node, i int) []Event {
	var events []Event
	for i > 0 {
		events = append(events, nodes[i].event)
		i = nodes[i].parent
	}
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events
}
func Explore(c Config, v substitution.Variant) Census {
	result := Census{Name: fmt.Sprintf("%s/%s/id%d/value%d/I%d/R%d/D%d", c.Context, v, c.Request.ID, c.Request.Value, c.I, c.R, c.Depth), InitialStates: 1}
	initial := Initial(c)
	nodes := []node{{state: initial, parent: -1}}
	seen := map[State]int{initial: 0}
	reported := map[string]bool{}
	witness := func(code string, i int) {
		if !reported[code] {
			reported[code] = true
			result.Witnesses = append(result.Witnesses, Witness{code, trace(nodes, i), nodes[i].state})
		}
	}
	for i := 0; i < len(nodes); i++ {
		s := nodes[i].state
		accepted := 0
		for _, op := range s.Base.Ops {
			if op.Admission == 1 {
				accepted++
				if op.Depth > c.Depth {
					result.ResourceFailures++
					witness("depth", i)
				}
			}
		}
		if s.Base.I != accepted || s.Base.R != accepted {
			result.ResourceFailures++
			witness("ledger", i)
		}
		if s.Base.I+demand(s) > c.I || s.Base.R+demand(s) > c.R || s.Base.E != 0 {
			result.ResourceFailures++
			witness("resource", i)
		}
		if s.Base.RootStatus != "" && demand(s) > 0 {
			result.LifetimeFailures++
			witness("lifetime", i)
		}
		if s.ReportStage > 0 && (!s.Base.Ops[sibling].Consumed || s.Report.Outcome != outcome(s.Base.Ops[sibling])) {
			result.ReportFailures++
			witness("report_causality", i)
		}
		if s.Base.Ops[entry].Consumed && s.Base.RootStatus == "" && s.Base.Ops[sibling].Admission == 0 {
			result.DecisiveStates++
			witness("candidate_consumed_before_s_admission", i)
		}
		if s.ReportDelivered && !s.Base.Ops[entry].Consumed {
			result.EarlyReportStates++
		}
		if s.ReportStage == 0 && s.Base.Ops[entry].Consumed {
			result.LateReportStates++
		}
		edges := Successors(c, v, s)
		if len(edges) == 0 {
			result.MaximalStates++
			switch s.Aggregate {
			case "succeeded":
				result.Successful++
				witness("aggregate_succeeded", i)
			case "failed":
				result.Failed++
				witness("aggregate_failed", i)
			default:
				if s.ReportStage == 3 && !s.ReportAllowed && s.Base.PeerStatus == "succeeded" && s.Base.RootStatus == "" && s.Base.Ops[entry].Consumed && s.Base.Ops[sibling].Consumed {
					result.Pending++
					witness("report_source_closed_pending", i)
				} else {
					result.Stuck++
					witness("unexpected_stuck", i)
				}
			}
		}
		for _, e := range edges {
			result.Transitions++
			if Rank(e.Next) <= Rank(s) {
				result.RankFailures++
			}
			j, ok := seen[e.Next]
			if !ok {
				j = len(nodes)
				seen[e.Next] = j
				nodes = append(nodes, node{state: e.Next, parent: i, event: e.Event})
			}
			nodes[i].next = append(nodes[i].next, j)
		}
	}
	result.States = len(nodes)
	if result.RankFailures > 0 {
		result.Cycles = -1
		result.MaximalExecutions = "unresolved"
		return result
	}
	// Exact path count is separate from maximal-state count. Strict intrinsic rank
	// is a topological order, including multiple incoming interleaving histories.
	order := make([]int, len(nodes))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return Rank(nodes[order[i]].state) < Rank(nodes[order[j]].state) })
	counts := make([]big.Int, len(nodes))
	counts[0].SetInt64(1)
	var total big.Int
	for _, i := range order {
		if len(nodes[i].next) == 0 {
			total.Add(&total, &counts[i])
		}
		for _, j := range nodes[i].next {
			counts[j].Add(&counts[j], &counts[i])
		}
	}
	result.MaximalExecutions = total.String()
	return result
}
