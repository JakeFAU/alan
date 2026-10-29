package substitution

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CheckCandidateContext checks the independently transcribed graph, not the
// ideal local/helper branch selector. Every returned membership is bounded.
func CheckCandidateContext(c Config, v Variant, observed ContextObservation) error {
	result := ExploreGraph(c, CandidateGraph(v))
	if len(result.Counterexamples) > 0 {
		return fmt.Errorf("candidate graph violation: %+v", result.Counterexamples[0])
	}
	for _, o := range result.Outcomes {
		if o == observed {
			return nil
		}
	}
	return fmt.Errorf("observation is not a maximal transcribed %s product result: %+v", v, observed)
}

// weakGraph selects the actual W_A/W_B retained-artifact transcription.
func weakGraph(v Variant) Graph {
	if v == Local {
		return actualProbeGraph("weak-a")
	}
	return actualProbeGraph("weak-b")
}

// equalGraph selects actual P/Q, including Q's single scratch=0 Set.
func equalGraph(extraSet bool) Graph {
	if extraSet {
		return actualProbeGraph("q")
	}
	return actualProbeGraph("p")
}

type ProbeResult struct {
	Name                                                                           string
	States, Transitions, Revisits, Terminal, Pending, Stuck, RankFailures, Cutoffs int
	Outcomes                                                                       []ContextObservation
	Traces                                                                         [][]Observation
	Counterexamples                                                                []Counterexample
}

// Probe uses the profile-specific output monitor, retaining K1's protocol and
// ledger assumptions; K1's complement monitor is not applied to a weaker profile.
func Probe(c Config, g Graph, zeroResult bool) ProbeResult {
	r := ProbeResult{Name: ConfigName(c)}
	start := Initial(c)
	nodes := []node{{s: start, parent: -1}}
	seen := map[State]bool{start: true}
	results := map[ContextObservation]bool{}
	for index := 0; index < len(nodes); index++ {
		s := nodes[index].s
		edges := g(c, s)
		if len(edges) == 0 {
			switch terminalKind(c, s) {
			case "terminal":
				r.Terminal++
			case "pending":
				r.Pending++
			default:
				r.Stuck++
			}
			o := Observe(s)
			if !results[o] {
				results[o] = true
				r.Outcomes = append(r.Outcomes, o)
				r.Traces = append(r.Traces, path(nodes, index, nil))
			}
		}
		for _, e := range edges {
			r.Transitions++
			if graphRank(e.Next) <= graphRank(s) {
				r.RankFailures++
			}
			if e.Next.I > c.I || e.Next.R > c.R {
				r.Counterexamples = append(r.Counterexamples, Counterexample{"ledger", "profile exceeded capacity", path(nodes, index, &e.Event)})
			}
			// This explicit profile monitor is independent of the K1 oracle.
			// Weak candidate AND helper send0; P/Q and their helper complement.
			if e.Event.Kind == "reply_admit" && (e.Event.Role == "candidate" || e.Event.Role == "helper") && e.Event.Present {
				expected := 1 - c.Request.Value
				if zeroResult {
					expected = 0
				}
				if e.Event.ID != c.Request.ID || e.Event.Value != expected {
					r.Counterexamples = append(r.Counterexamples, Counterexample{"probe_output", fmt.Sprintf("expected id%d/value%d", c.Request.ID, expected), path(nodes, index, &e.Event)})
				}
			}

			if seen[e.Next] {
				r.Revisits++
			} else {
				seen[e.Next] = true
				nodes = append(nodes, node{e.Next, index, e.Event})
			}
		}
	}
	r.States = len(nodes)
	return r
}

type WeakRelationResult struct {
	Name                              string
	Pairs, Checks, Stutters, Failures int
	Counterexample                    []Observation
}

func equalProjection(s State) State { s.PrivateRegisters[entry][7] = 0; return s }

// CheckEqualLocal checks the proposed weak relation BEFORE any contextual trace
// language check. Environment matches are direct at every instruction boundary.
func CheckEqualLocal(c Config) WeakRelationResult {
	type pair struct{ a, b State }
	r := WeakRelationResult{Name: ConfigName(c)}
	start := pair{Initial(c), Initial(c)}
	queue := []pair{start}
	seen := map[pair]bool{start: true}
	graphs := [2]Graph{equalGraph(false), equalGraph(true)}
	for index := 0; index < len(queue); index++ {
		p := queue[index]
		r.Pairs++
		for side := 0; side < 2; side++ {
			a, b := p.a, p.b
			if side == 1 {
				a, b = b, a
			}
			for _, edge := range graphs[side](c, a) {
				r.Checks++
				matched := false
				if edge.Event.Kind == "private_step" && equalProjection(edge.Next) == equalProjection(b) {
					matched = true
					r.Stutters++
					q := pair{edge.Next, b}
					if side == 1 {
						q = pair{b, edge.Next}
					}
					if !seen[q] {
						seen[q] = true
						queue = append(queue, q)
					}
				}
				candidates := []State{b}
				if !edge.Environment {
					for j := 0; j < len(candidates); j++ {
						for _, e := range graphs[1-side](c, candidates[j]) {
							if e.Event.Kind == "private_step" && equalProjection(e.Next) == equalProjection(candidates[j]) {
								candidates = append(candidates, e.Next)
							}
						}
					}
				}
				for _, b0 := range candidates {
					for _, other := range graphs[1-side](c, b0) {
						if edge.Event == other.Event && equalProjection(edge.Next) == equalProjection(other.Next) {
							matched = true
							q := pair{edge.Next, other.Next}
							if side == 1 {
								q = pair{other.Next, edge.Next}
							}
							if !seen[q] {
								seen[q] = true
								queue = append(queue, q)
							}
						}
					}
				}
				if !matched {
					r.Failures++
					if r.Counterexample == nil {
						r.Counterexample = []Observation{edge.Event}
					}
				}
			}
		}
	}
	return r
}

type automaton struct {
	states []State
	edges  [][]Edge
	ids    map[State]int
}

func buildAutomaton(c Config, g Graph) automaton {
	a := automaton{states: []State{Initial(c)}, ids: map[State]int{Initial(c): 0}}
	for i := 0; i < len(a.states); i++ {
		edges := g(c, a.states[i])
		a.edges = append(a.edges, edges)
		for _, e := range edges {
			if _, ok := a.ids[e.Next]; !ok {
				a.ids[e.Next] = len(a.states)
				a.states = append(a.states, e.Next)
			}
		}
	}
	return a
}
func silent(e Edge) bool {
	return e.Event.Kind == "private_step" || e.Event.Kind == "choose_local" || e.Event.Kind == "choose_helper"
}
func closure(a automaton, ids []int) []int {
	seen := map[int]bool{}
	var result []int
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	for i := 0; i < len(result); i++ {
		for _, e := range a.edges[result[i]] {
			if silent(e) {
				id := a.ids[e.Next]
				if !seen[id] {
					seen[id] = true
					result = append(result, id)
				}
			}
		}
	}
	sort.Ints(result)
	return result
}
func subsetKey(ids []int) string { return fmt.Sprint(ids) }
func abstractMoves(a automaton, ids []int) map[string][]int {
	moves := map[string][]int{}
	for _, id := range ids {
		if len(a.edges[id]) == 0 {
			b, _ := json.Marshal(Observe(a.states[id]))
			moves["FINAL:"+string(b)] = nil
		}
		for _, e := range a.edges[id] {
			if silent(e) {
				continue
			}
			b, _ := json.Marshal(e.Event)
			key := string(b)
			moves[key] = append(moves[key], a.ids[e.Next])
		}
	}
	return moves
}

type LanguageResult struct {
	Name                                              string
	LeftStates, RightStates, SubsetPairs, Transitions int
	Equal                                             bool
	ShortestDifference                                []string
}

// CompareLanguages checks exact equality of finite observable trace languages by
// epsilon-closure/subset exploration. It does not compare concrete scheduler runs.
func CompareLanguages(c Config, left, right Graph) LanguageResult {
	a, b := buildAutomaton(c, left), buildAutomaton(c, right)
	r := LanguageResult{Name: ConfigName(c), LeftStates: len(a.states), RightStates: len(b.states), Equal: true}
	type item struct {
		a, b  []int
		trace []string
	}
	start := item{a: closure(a, []int{0}), b: closure(b, []int{0})}
	queue := []item{start}
	seen := map[string]bool{}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		key := subsetKey(p.a) + "/" + subsetKey(p.b)
		if seen[key] {
			continue
		}
		seen[key] = true
		r.SubsetPairs++
		ma, mb := abstractMoves(a, p.a), abstractMoves(b, p.b)
		keys := map[string]bool{}
		for k := range ma {
			keys[k] = true
		}
		for k := range mb {
			keys[k] = true
		}
		var ordered []string
		for k := range keys {
			ordered = append(ordered, k)
		}
		sort.Strings(ordered)
		for _, k := range ordered {
			r.Transitions++
			ta, oka := ma[k]
			tb, okb := mb[k]
			if !oka || !okb {
				r.Equal = false
				r.ShortestDifference = append(append([]string{}, p.trace...), k)
				return r
			}
			if strings.HasPrefix(k, "FINAL:") {
				continue
			}
			queue = append(queue, item{closure(a, ta), closure(b, tb), append(append([]string{}, p.trace...), k)})
		}
	}
	return r
}
