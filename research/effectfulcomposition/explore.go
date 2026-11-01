package effectfulcomposition

import (
	"fmt"
	"strings"
)

type Edge struct {
	Action string
	To     int
}
type Row struct {
	ID    int
	State State
	Class string
	Edges []Edge
}
type Graph struct {
	Profile Profile
	Tree    Pure
	Rows    []Row
}

func Explore(p Profile) Graph {
	g := Graph{Profile: p, Tree: Tree(p)}
	init := Initial(p)
	states := []State{init}
	seen := map[State]int{init: 0}
	for i := 0; i < len(states); i++ {
		s := states[i]
		r := Row{ID: i, State: s, Class: Class(p, s), Edges: []Edge{}}
		for _, step := range Next(p, s) {
			to, ok := seen[step.To]
			if !ok {
				to = len(states)
				seen[step.To] = to
				states = append(states, step.To)
			}
			r.Edges = append(r.Edges, Edge{Action: step.Action, To: to})
		}
		g.Rows = append(g.Rows, r)
	}
	return g
}
func (s Spend) coordinates() [8]int {
	return [8]int{s.I, s.R, s.Mut, s.Read, s.Ord, s.Post, s.Get, s.Apply}
}
func Check(g Graph) error {
	p := g.Profile
	for _, r := range g.Rows {
		s := r.State
		if s.Decision != "" && s.Joined != 5 {
			return fmt.Errorf("premature join %d", r.ID)
		}
		if s.Serial > len(s.Log) {
			return fmt.Errorf("log overflow")
		}
		if s.Cost[0].Apply+s.Cost[1].Apply > 2 {
			return fmt.Errorf("extra business application")
		}
		for i, v := range s.S {
			if v.Witness != 0 && v.Proof.Role != i+1 {
				return fmt.Errorf("cross-role proof")
			}
			if v.ResolverEnd && !v.AckSelected {
				return fmt.Errorf("resolver ended without selection")
			}
			for _, q := range v.Query {
				if q.Ord > 2 {
					return fmt.Errorf("overspent ordinal")
				}
				if q.Phase == "selected" && q.Adoption == "adopted" && !v.Settled {
					return fmt.Errorf("unsettled selected importer")
				}
			}
		}
		if len(r.Edges) == 0 && r.Class == "prefix" {
			return fmt.Errorf("unclassified sink %s/%d %+v", p.Name, r.ID, s)
		}
		if len(r.Edges) == 0 && p.Slice != "launch" && p.Case != "manual-ack" && !strings.HasPrefix(r.Class, "pending/") && (!s.ParentRoot || !s.S[0].Root || !s.S[1].Root) {
			return fmt.Errorf("unserviced maximum %s/%d", p.Name, r.ID)
		}
		labels := map[string]bool{}
		for _, e := range r.Edges {
			if labels[e.Action] {
				return fmt.Errorf("duplicate edge %s", e.Action)
			}
			labels[e.Action] = true
			n := g.Rows[e.To].State
			for i, c := range s.Cost {
				next := n.Cost[i].coordinates()
				for k, x := range c.coordinates() {
					if next[k] < x {
						return fmt.Errorf("refunded %s/%d", e.Action, k)
					}
				}
			}
			if s.World != n.World && !strings.HasSuffix(e.Action, ".apply") && !strings.HasSuffix(e.Action, ".apply-held-packet") {
				return fmt.Errorf("foreign mutation by %s", e.Action)
			}
		}
	}
	// Reverse Kahn elimination checks all vertices have finite longest suffix rank;
	// terminal prefixes, intrinsically pending maxima and full completion differ.
	inbound := make([][]int, len(g.Rows))
	degs := make([]int, len(g.Rows))
	queue := []int{}
	for _, r := range g.Rows {
		degs[r.ID] = len(r.Edges)
		if len(r.Edges) == 0 {
			queue = append(queue, r.ID)
		}
		for _, e := range r.Edges {
			inbound[e.To] = append(inbound[e.To], r.ID)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, from := range inbound[queue[i]] {
			degs[from]--
			if degs[from] == 0 {
				queue = append(queue, from)
			}
		}
	}
	if len(queue) != len(g.Rows) {
		return fmt.Errorf("reachable cycle")
	}
	return nil
}
