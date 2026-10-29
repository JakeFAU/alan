package communication

import (
	"fmt"
	"github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/research/substitution"
	"math/big"
	"sort"
)

type Witness struct {
	Code   string
	Events []join.Event
	Final  State
}
type Census struct {
	Name                                                                                                string
	States, Transitions, MaximalStates, Successful, Pending, Failed, Stuck                              int
	RankFailures, DomainFailures, ResourceFailures, LifetimeFailures, ProtocolFailures, Cycles, Cutoffs int
	MinRank, MaxRank                                                                                    int
	MaximalExecutions                                                                                   string
	Maxima                                                                                              []State
	Witnesses                                                                                           []Witness
}

type node struct {
	state  State
	parent int
	event  join.Event
	next   []int
}

func path(nodes []node, i int) []join.Event {
	var result []join.Event
	for i > 0 {
		result = append(result, nodes[i].event)
		i = nodes[i].parent
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// Explore exhausts one declared product, retaining every maximal state. The
// supported domain is deliberately fixed; other requests need a new contract.
func Explore(c Config, v substitution.Variant) Census {
	if c != PrimaryConfig() || (v != substitution.Reference && v != substitution.Local && v != substitution.Helper) {
		panic("communication: outside frozen Phase1K domain")
	}
	initial := Initial(c)
	r := Census{Name: fmt.Sprintf("J2/%s/id1/value0/answer/I3/R3/D2/E0", v), MinRank: Rank(initial)}
	nodes := []node{{state: initial, parent: -1}}
	seen := map[State]int{initial: 0}
	reported := map[string]bool{}
	witness := func(code string, i int) {
		if !reported[code] {
			reported[code] = true
			r.Witnesses = append(r.Witnesses, Witness{code, path(nodes, i), nodes[i].state})
		}
	}
	for i := 0; i < len(nodes); i++ {
		s := nodes[i].state
		b := s.Join.Base
		if !InDomain(s) {
			r.DomainFailures++
			witness("domain", i)
		}
		if Rank(s) > r.MaxRank {
			r.MaxRank = Rank(s)
		}
		demand, accepted := 0, 0
		for _, o := range b.Ops {
			if o.Ticket < 2 {
				demand++
			}
			if o.Admission == 1 {
				accepted++
				if o.Depth > c.Depth {
					r.ResourceFailures++
				}
			}
		}
		if b.I != accepted || b.R != accepted || b.I+demand > c.I || b.R+demand > c.R || b.E != 0 {
			r.ResourceFailures++
			witness("resource", i)
		}
		if b.RootStatus != "" && demand > 0 {
			r.LifetimeFailures++
			witness("lifetime", i)
		}
		if (s.AckStage > 0 && s.Join.ReportStage != 4) || (b.RootStatus != "" && s.AckStage == 0) || (b.PeerStatus != "" && s.AckStage != 4) || (s.Join.ReportStage >= 2 && !s.Join.ReportAllowed) || (s.AckDelivered && !s.AckAllowed) {
			r.ProtocolFailures++
			witness("protocol", i)
		}
		// Preserve the allowed-first, source-terminal, not-admitted prefix and finish
		// its exact trace; a generic shortest success trace need not contain this cut.
		if !reported["authorized_then_terminal_then_admission"] && s.AckStage == 2 && s.AckAllowed && b.RootStatus == "succeeded" {
			end := s
			events := path(nodes, i)
			for {
				es := Successors(c, v, end)
				if len(es) == 0 {
					break
				}
				e := es[0]
				if Rank(e.Next) <= Rank(end) {
					break
				}
				events = append(events, e.Event)
				end = e.Next
			}
			reported["authorized_then_terminal_then_admission"] = true
			r.Witnesses = append(r.Witnesses, Witness{"authorized_then_terminal_then_admission", events, end})
		}
		edges := Successors(c, v, s)
		if len(edges) == 0 {
			r.MaximalStates++
			r.Maxima = append(r.Maxima, s)
			switch {
			case s.Join.Aggregate == "succeeded":
				r.Successful++
				witness("aggregate_succeeded", i)
			case s.Join.Aggregate == "failed":
				r.Failed++
				witness("aggregate_failed", i)
			case s.AckStage == 3 && !s.AckAllowed && !s.AckDelivered && b.RootStatus == "succeeded" && b.PeerStatus == "" && s.Join.ReportStage == 4:
				r.Pending++
				witness("ack_source_closed_pending", i)
			default:
				r.Stuck++
				witness("unexpected_stuck", i)
			}
		}
		for _, e := range edges {
			r.Transitions++
			if Rank(e.Next) <= Rank(s) {
				r.RankFailures++
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
	r.States = len(nodes)
	if r.RankFailures > 0 {
		r.Cycles = -1
		r.MaximalExecutions = "unresolved"
		return r
	}
	// Strict bounded intrinsic rank is a topological order, checked on every edge.
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
	r.MaximalExecutions = total.String()
	return r
}
