package substitution

import (
	"fmt"
	"slices"
)

// These records validate finite supplied certificates only. They provide no
// existence procedure, candidate-domain check, or full ARG-1 verdict.
type weakRule struct {
	Owner, Semantic string
	Word            []Observation
	Private         bool
}
type weakNode struct {
	Projection string
	Demand     [3]int
	Guards     string
	Rank       int
	Boundary   string
}
type weakEdge struct{ ID, From, To, Rule string }
type weakFiniteGraph struct {
	Initial    []string
	Nodes      map[string]weakNode
	Edges      []weakEdge
	Complete   bool
	Boundaries map[string]string // boundary name -> required endpoint projection
}
type weakPair struct{ X, K string }
type weakMatch struct {
	Pair weakPair
	Edge string
	Path []string
	// Direct identifies the original matched semantic action. Genuine private
	// work must preserve selected public fields even when marked Direct.
	Direct string
}
type weakEnvironment struct {
	Pair              weakPair
	Side, Edge, Other string
}
type weakCertificate struct {
	Pairs       []weakPair
	Component   []weakMatch
	Environment []weakEnvironment
}
type weakProblem struct {
	X, K           weakFiniteGraph
	Rules          map[string]weakRule
	Demand, Guards bool
}
type weakIssue struct {
	Clause, Detail string
	Pair           weakPair
	Edge           string
}
type weakResult struct {
	Input, Transfer, Readiness, Strong []weakIssue
	Pairs, Components, Environments    int
}

func (r weakResult) accepted() bool {
	return len(r.Input) == 0 && len(r.Transfer) == 0 && len(r.Readiness) == 0
}

// Structural errors stop semantic assessment. Missing/wrong legal obligations
// reject only this supplied witness. There is deliberately no search fallback.
func checkWeakCertificate(p weakProblem, c weakCertificate) weakResult {
	r := weakResult{}
	x, xi := indexWeakGraph(p.X)
	k, ki := indexWeakGraph(p.K)
	r.Input = append(xi, ki...)
	r.Input = append(r.Input, checkWeakGrammar(p.X, p.Rules)...)
	r.Input = append(r.Input, checkWeakGrammar(p.K, p.Rules)...)
	if len(r.Input) != 0 {
		return r
	}
	pairs := map[weakPair]bool{}
	for _, pair := range c.Pairs {
		_, xok := p.X.Nodes[pair.X]
		_, kok := p.K.Nodes[pair.K]
		if !xok || !kok || pairs[pair] {
			r.Input = append(r.Input, weakIssue{Clause: "relation", Pair: pair})
		}
		pairs[pair] = true
	}
	type key struct {
		Pair       weakPair
		Side, Edge string
	}
	matches := map[key]weakMatch{}
	environments := map[key]weakEnvironment{}
	for _, m := range c.Component {
		mk := key{Pair: m.Pair, Edge: m.Edge}
		edge, ok := x.edges[m.Edge]
		if !pairs[m.Pair] || !ok || edge.From != m.Pair.X || matches[mk].Edge != "" {
			r.Input = append(r.Input, weakIssue{Clause: "component", Pair: m.Pair, Edge: m.Edge})
			continue
		}
		matches[mk] = m
		at := m.Pair.K
		direct := m.Direct == ""
		directCount := 0
		for _, id := range m.Path {
			e, exists := k.edges[id]
			if !exists || e.From != at {
				r.Input = append(r.Input, weakIssue{Clause: "path", Pair: m.Pair, Edge: id})
				break
			}
			at = e.To
			if id == m.Direct {
				directCount++
				xr, kr := p.Rules[edge.Rule], p.Rules[e.Rule]
				direct = xr.Semantic != "" && xr.Semantic == kr.Semantic && slices.Equal(xr.Word, kr.Word)
			}
		}
		if !direct || (m.Direct != "" && directCount != 1) {
			r.Input = append(r.Input, weakIssue{Clause: "direct_match", Pair: m.Pair, Edge: m.Direct})
		}
	}
	for _, m := range c.Environment {
		mk := key{Pair: m.Pair, Side: m.Side, Edge: m.Edge}
		a, b := x, k
		from, other := m.Pair.X, m.Pair.K
		if m.Side == "K" {
			a, b = k, x
			from, other = m.Pair.K, m.Pair.X
		}
		e, ok := a.edges[m.Edge]
		f, fok := b.edges[m.Other]
		legalSide := m.Side == "X" || m.Side == "K"
		if !legalSide || !pairs[m.Pair] || !ok || !fok || e.From != from || f.From != other ||
			p.Rules[e.Rule].Owner != "environment" || p.Rules[f.Rule].Owner != "environment" || environments[mk].Edge != "" {
			r.Input = append(r.Input, weakIssue{Clause: "environment", Pair: m.Pair, Edge: m.Edge})
			continue
		}
		environments[mk] = m
	}
	if len(r.Input) != 0 {
		return r
	}
	xr, kr := weakReachable(p.X, x), weakReachable(p.K, k)
	for _, initial := range p.X.Initial {
		covered := false
		for _, ref := range p.K.Initial {
			covered = covered || pairs[weakPair{X: initial, K: ref}]
		}
		if !covered {
			r.Transfer = append(r.Transfer, weakIssue{Clause: "initial", Detail: initial})
		}
	}
	for _, pair := range c.Pairs {
		if !xr[pair.X] || !kr[pair.K] {
			continue
		}
		r.Pairs++
		a, b := p.X.Nodes[pair.X], p.K.Nodes[pair.K]
		if a.Projection != b.Projection {
			r.Transfer = append(r.Transfer, weakIssue{Clause: "projection", Pair: pair})
		}
		if p.Demand && a.Demand != b.Demand {
			r.Strong = append(r.Strong, weakIssue{Clause: "demand", Pair: pair})
		}
		if p.Guards && a.Guards != b.Guards {
			r.Strong = append(r.Strong, weakIssue{Clause: "guard", Pair: pair})
		}
		for _, e := range x.out[pair.X] {
			rule := p.Rules[e.Rule]
			if rule.Owner != "component" {
				continue
			}
			r.Components++
			if rule.Private {
				after := p.X.Nodes[e.To]
				if a.Projection != after.Projection {
					r.Transfer = append(r.Transfer, weakIssue{Clause: "private_projection", Pair: pair, Edge: e.ID})
				}
				if p.Guards && a.Guards != after.Guards {
					r.Strong = append(r.Strong, weakIssue{Clause: "private_guard", Detail: "candidate", Pair: pair, Edge: e.ID})
				}
				if p.Demand && a.Demand != after.Demand {
					r.Strong = append(r.Strong, weakIssue{Clause: "private_demand", Detail: "candidate", Pair: pair, Edge: e.ID})
				}
			}
			m, ok := matches[key{Pair: pair, Edge: e.ID}]
			if !ok {
				r.Transfer = append(r.Transfer, weakIssue{Clause: "missing_component", Pair: pair, Edge: e.ID})
				continue
			}
			at := pair.K
			word := []Observation{}
			for _, id := range m.Path {
				f := k.edges[id]
				rr := p.Rules[f.Rule]
				if rr.Owner != "component" {
					r.Transfer = append(r.Transfer, weakIssue{Clause: "component_owner", Pair: pair, Edge: id})
				}
				word = append(word, rr.Word...)
				before, after := p.K.Nodes[f.From], p.K.Nodes[f.To]
				// Genuine private work has no accounting or current-guard update.
				// Direct is not an exemption from this preservation requirement.
				if rr.Private {
					if before.Projection != after.Projection {
						r.Transfer = append(r.Transfer, weakIssue{Clause: "private_projection", Pair: pair, Edge: id})
					}
					if p.Guards && before.Guards != after.Guards {
						r.Strong = append(r.Strong, weakIssue{Clause: "private_guard", Detail: "reference", Pair: pair, Edge: id})
					}
					if p.Demand && before.Demand != after.Demand {
						r.Strong = append(r.Strong, weakIssue{Clause: "private_demand", Detail: "reference", Pair: pair, Edge: id})
					}
				}
				// Unmatched silent semantic actions preserve current guards; endpoint
				// demand equality does not require their future demand to stay constant.
				if len(rr.Word) == 0 && id != m.Direct {
					if p.Guards && before.Guards != after.Guards {
						r.Strong = append(r.Strong, weakIssue{Clause: "silent_guard", Pair: pair, Edge: id})
					}
				}
				at = f.To
			}
			if !slices.Equal(rule.Word, word) {
				r.Transfer = append(r.Transfer, weakIssue{Clause: "word", Pair: pair, Edge: e.ID,
					Detail: fmt.Sprintf("candidate %v reference %v", rule.Word, word)})
			}
			if !pairs[weakPair{X: e.To, K: at}] {
				r.Transfer = append(r.Transfer, weakIssue{Clause: "endpoint", Pair: pair, Edge: e.ID})
			}
			if p.X.Nodes[e.To].Projection != p.K.Nodes[at].Projection {
				r.Transfer = append(r.Transfer, weakIssue{Clause: "endpoint_projection", Pair: pair, Edge: e.ID})
			}
		}
		for _, side := range []string{"X", "K"} {
			g, h := x, k
			from := pair.X
			if side == "K" {
				g, h = k, x
				from = pair.K
			}
			for _, e := range g.out[from] {
				if p.Rules[e.Rule].Owner != "environment" {
					continue
				}
				r.Environments++
				m, ok := environments[key{Pair: pair, Side: side, Edge: e.ID}]
				if !ok {
					r.Readiness = append(r.Readiness, weakIssue{Clause: "missing_environment", Pair: pair, Edge: e.ID})
					continue
				}
				f := h.edges[m.Other]
				end := weakPair{X: e.To, K: f.To}
				if side == "K" {
					end = weakPair{X: f.To, K: e.To}
				}
				same := slices.Equal(p.Rules[e.Rule].Word, p.Rules[f.Rule].Word)
				if !same || !pairs[end] {
					r.Readiness = append(r.Readiness, weakIssue{Clause: "environment_match", Pair: pair, Edge: e.ID})
				}
			}
		}
	}
	return r
}

type weakIndex struct {
	edges map[string]weakEdge
	out   map[string][]weakEdge
}

func indexWeakGraph(g weakFiniteGraph) (weakIndex, []weakIssue) {
	index := weakIndex{edges: map[string]weakEdge{}, out: map[string][]weakEdge{}}
	issues := []weakIssue{}
	if len(g.Initial) == 0 {
		issues = append(issues, weakIssue{Clause: "initials"})
	}
	for id, n := range g.Nodes {
		if id == "" || n.Projection == "" {
			issues = append(issues, weakIssue{Clause: "node", Detail: id})
		}
	}
	for _, id := range g.Initial {
		if _, ok := g.Nodes[id]; !ok {
			issues = append(issues, weakIssue{Clause: "initial_id", Detail: id})
		}
	}
	for _, e := range g.Edges {
		_, source := g.Nodes[e.From]
		_, target := g.Nodes[e.To]
		if _, duplicate := index.edges[e.ID]; e.ID == "" || duplicate || !source || !target {
			issues = append(issues, weakIssue{Clause: "edge", Edge: e.ID})
		}
		index.edges[e.ID] = e
		index.out[e.From] = append(index.out[e.From], e)
	}
	return index, issues
}

// The structural-only rank check intentionally has no alphabet. Certificate
// and cycle checks must call this validator; nil is not a declared empty table.
func checkWeakGrammar(g weakFiniteGraph, rules map[string]weakRule) []weakIssue {
	if rules == nil {
		return []weakIssue{{Clause: "grammar", Detail: "missing declared alphabet"}}
	}
	issues := []weakIssue{}
	for name, rule := range rules {
		owner := rule.Owner == "component" || rule.Owner == "environment"
		malformedPrivate := rule.Private && (rule.Owner != "component" || len(rule.Word) != 0)
		if name == "" || !owner || malformedPrivate {
			issues = append(issues, weakIssue{Clause: "grammar", Detail: name})
		}
	}
	for _, edge := range g.Edges {
		if _, present := rules[edge.Rule]; !present {
			issues = append(issues, weakIssue{Clause: "grammar", Detail: "missing rule " + edge.Rule, Edge: edge.ID})
		}
	}
	return issues
}

func weakReachable(g weakFiniteGraph, index weakIndex) map[string]bool {
	seen := map[string]bool{}
	queue := slices.Clone(g.Initial)
	for _, id := range queue {
		seen[id] = true
	}
	for i := 0; i < len(queue); i++ {
		for _, e := range index.out[queue[i]] {
			if !seen[e.To] {
				seen[e.To] = true
				queue = append(queue, e.To)
			}
		}
	}
	return seen
}

// Rank is a sufficient complete-graph certificate, not a complete progress test.
// Boundary facts and sink checks are independent of relation acceptance.
func checkWeakRank(g weakFiniteGraph) []weakIssue {
	issues := []weakIssue{}
	if !g.Complete {
		return []weakIssue{{Clause: "open_ports"}}
	}
	index, structural := indexWeakGraph(g)
	if len(structural) != 0 {
		return structural
	}
	reached := weakReachable(g, index)
	for id, n := range g.Nodes {
		if !reached[id] {
			continue
		}
		if n.Rank < 0 {
			issues = append(issues, weakIssue{Clause: "rank", Detail: id})
		}
		if n.Boundary != "" {
			expected, ok := g.Boundaries[n.Boundary]
			if !ok || n.Projection != expected || len(index.out[id]) != 0 {
				issues = append(issues, weakIssue{Clause: "boundary", Detail: id})
			}
			continue
		}
		if len(index.out[id]) == 0 {
			issues = append(issues, weakIssue{Clause: "stuck", Detail: id})
		}
		for _, e := range index.out[id] {
			if g.Nodes[e.To].Rank >= n.Rank {
				issues = append(issues, weakIssue{Clause: "rank", Edge: e.ID})
			}
		}
	}
	return issues
}

// A supplied reachable private cycle is a concrete negative nondivergence
// witness in a complete toy grammar with no environment or other enabled work.
// No SCC search, fairness inference, or actual K1 Progress_N claim is made.
func checkWeakCycle(g weakFiniteGraph, rules map[string]weakRule, cycle []string) []weakIssue {
	if !g.Complete || len(cycle) == 0 {
		return []weakIssue{{Clause: "cycle"}}
	}
	index, structural := indexWeakGraph(g)
	structural = append(structural, checkWeakGrammar(g, rules)...)
	if len(structural) != 0 {
		return structural
	}
	reached := weakReachable(g, index)
	first, ok := index.edges[cycle[0]]
	if !ok || !reached[first.From] {
		return []weakIssue{{Clause: "cycle"}}
	}
	at := first.From
	for _, id := range cycle {
		e, ok := index.edges[id]
		private := rules[e.Rule].Private && rules[e.Rule].Owner == "component"
		onlyEnabled := len(index.out[at]) == 1 && g.Nodes[at].Boundary == ""
		if !ok || e.From != at || !private || !onlyEnabled {
			return []weakIssue{{Clause: "cycle", Edge: id}}
		}
		at = e.To
	}
	if at != first.From {
		return []weakIssue{{Clause: "cycle"}}
	}
	return []weakIssue{}
}
