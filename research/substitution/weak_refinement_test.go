package substitution

import (
	"slices"
	"testing"
)

func weakToyRules() map[string]weakRule {
	return map[string]weakRule{
		"a":   {Owner: "component", Semantic: "a", Word: []Observation{{Kind: "a"}}},
		"b":   {Owner: "component", Semantic: "b", Word: []Observation{{Kind: "b"}}},
		"d":   {Owner: "environment", Semantic: "d", Word: []Observation{{Kind: "d"}}},
		"tau": {Owner: "component", Private: true},
	}
}
func weakToyNode(p string, rank int, boundary string) weakNode {
	return weakNode{Projection: p, Guards: "0", Rank: rank, Boundary: boundary}
}
func weakF1() (weakProblem, weakCertificate) {
	p := weakProblem{Rules: weakToyRules(), Demand: true, Guards: true,
		X: weakFiniteGraph{Initial: []string{"x0"}, Complete: true, Boundaries: map[string]string{"done": "1"},
			Nodes: map[string]weakNode{"x0": weakToyNode("0", 1, ""), "x1": weakToyNode("1", 0, "done")},
			Edges: []weakEdge{{ID: "A", From: "x0", To: "x1", Rule: "a"}}},
		K: weakFiniteGraph{Initial: []string{"k0"}, Complete: true, Boundaries: map[string]string{"done": "1"},
			Nodes: map[string]weakNode{"k0": weakToyNode("0", 2, ""), "kl": weakToyNode("0", 1, ""),
				"kr": weakToyNode("0", 1, ""), "k1": weakToyNode("1", 0, "done")},
			Edges: []weakEdge{{ID: "L", From: "k0", To: "kl", Rule: "tau"}, {ID: "R", From: "k0", To: "kr", Rule: "tau"},
				{ID: "AL", From: "kl", To: "k1", Rule: "a"}, {ID: "AR", From: "kr", To: "k1", Rule: "a"}}}}
	c := weakCertificate{Pairs: []weakPair{{X: "x0", K: "k0"}, {X: "x0", K: "kl"}, {X: "x0", K: "kr"}, {X: "x1", K: "k1"}},
		Component: []weakMatch{
			{Pair: weakPair{X: "x0", K: "k0"}, Edge: "A", Path: []string{"L", "AL"}, Direct: "AL"},
			{Pair: weakPair{X: "x0", K: "kl"}, Edge: "A", Path: []string{"AL"}, Direct: "AL"},
			{Pair: weakPair{X: "x0", K: "kr"}, Edge: "A", Path: []string{"AR"}, Direct: "AR"}}}
	return p, c
}
func requireWeak(t *testing.T, p weakProblem, c weakCertificate) weakResult {
	t.Helper()
	r := checkWeakCertificate(p, c)
	if !r.accepted() {
		t.Fatalf("supplied weak witness: %+v", r)
	}
	return r
}
func hasWeakIssue(issues []weakIssue, clause string) bool {
	return slices.ContainsFunc(issues, func(i weakIssue) bool { return i.Clause == clause })
}
func TestWeakRefinementF1Noncanonical(t *testing.T) {
	p, c := weakF1()
	for _, name := range []string{"left", "right", "smaller"} {
		t.Run(name, func(t *testing.T) {
			cc := c
			cc.Component = slices.Clone(c.Component)
			if name == "right" {
				cc.Component[0].Path = []string{"R", "AR"}
				cc.Component[0].Direct = "AR"
			}
			if name == "smaller" {
				cc.Pairs = []weakPair{c.Pairs[0], c.Pairs[3]}
				cc.Component = cc.Component[:1]
			}
			r := requireWeak(t, p, cc)
			if len(r.Strong) != 0 {
				t.Fatalf("constant strong clauses: %+v", r.Strong)
			}
			for _, g := range []weakFiniteGraph{p.X, p.K} {
				if issues := checkWeakRank(g); len(issues) != 0 {
					t.Fatal(issues)
				}
			}
		})
	}
	t.Run("noncontiguous", func(t *testing.T) {
		c.Component[1].Path = []string{"AR"}
		c.Component[1].Direct = "AR"
		r := checkWeakCertificate(p, c)
		if !hasWeakIssue(r.Input, "path") {
			t.Fatalf("want certificate path error: %+v", r)
		}
	})
}
func TestWeakRefinementF1RejectedPathsRemainOpen(t *testing.T) {
	p, c := weakF1()
	c.Component[0].Path = []string{}
	c.Component[0].Direct = ""
	r := checkWeakCertificate(p, c)
	if len(r.Input) != 0 || !hasWeakIssue(r.Transfer, "word") || !hasWeakIssue(r.Transfer, "endpoint") {
		t.Fatalf("empty path must reject this witness: %+v", r)
	}
	_, valid := weakF1()
	requireWeak(t, p, valid)
}
func weakF2() (weakProblem, weakCertificate) {
	p := weakProblem{Rules: weakToyRules(), X: weakFiniteGraph{Initial: []string{"x00"}, Complete: true,
		Boundaries: map[string]string{"done": "11"}, Nodes: map[string]weakNode{
			"x00": weakToyNode("00", 3, ""), "x10": weakToyNode("00", 2, ""), "x11": weakToyNode("01", 1, ""),
			"xd0": weakToyNode("10", 1, ""), "xd1": weakToyNode("11", 0, "done")},
		Edges: []weakEdge{{ID: "T", From: "x00", To: "x10", Rule: "tau"}, {ID: "A00", From: "x00", To: "xd0", Rule: "a"},
			{ID: "A10", From: "x10", To: "xd0", Rule: "a"}, {ID: "A11", From: "x11", To: "xd1", Rule: "a"},
			{ID: "D10", From: "x10", To: "x11", Rule: "d"}, {ID: "DD", From: "xd0", To: "xd1", Rule: "d"}}},
		K: weakFiniteGraph{Initial: []string{"k0"}, Complete: true, Boundaries: map[string]string{"done": "11"},
			Nodes: map[string]weakNode{"k0": weakToyNode("00", 2, ""), "k1": weakToyNode("01", 1, ""),
				"kd0": weakToyNode("10", 1, ""), "kd1": weakToyNode("11", 0, "done")},
			Edges: []weakEdge{{ID: "D0", From: "k0", To: "k1", Rule: "d"}, {ID: "DK", From: "kd0", To: "kd1", Rule: "d"},
				{ID: "AK0", From: "k0", To: "kd0", Rule: "a"}, {ID: "AK1", From: "k1", To: "kd1", Rule: "a"}}}}
	pairs := []weakPair{
		{X: "x00", K: "k0"}, {X: "x10", K: "k0"}, {X: "x11", K: "k1"}, {X: "xd0", K: "kd0"}, {X: "xd1", K: "kd1"},
	}
	c := weakCertificate{Pairs: pairs, Component: []weakMatch{
		{Pair: pairs[0], Edge: "T", Path: []string{}}, {Pair: pairs[0], Edge: "A00", Path: []string{"AK0"}},
		{Pair: pairs[1], Edge: "A10", Path: []string{"AK0"}}, {Pair: pairs[2], Edge: "A11", Path: []string{"AK1"}}}}
	for _, v := range []struct {
		pair weakPair
		x, k string
	}{{pair: pairs[1], x: "D10", k: "D0"}, {pair: pairs[3], x: "DD", k: "DK"}} {
		c.Environment = append(c.Environment, weakEnvironment{Pair: v.pair, Side: "X", Edge: v.x, Other: v.k},
			weakEnvironment{Pair: v.pair, Side: "K", Edge: v.k, Other: v.x})
	}
	return p, c
}
func TestWeakRefinementF2ImmediateReadiness(t *testing.T) {
	p, c := weakF2()
	r := checkWeakCertificate(p, c)
	if len(r.Input) != 0 || len(r.Transfer) != 0 || len(r.Readiness) != 1 ||
		r.Readiness[0].Pair != (weakPair{X: "x00", K: "k0"}) || r.Readiness[0].Edge != "D0" {
		t.Fatalf("exact missing initial bilateral readiness: %+v", r)
	}
	for _, g := range []weakFiniteGraph{p.X, p.K} {
		if issues := checkWeakRank(g); len(issues) != 0 {
			t.Fatal(issues)
		}
	}
	// A counterpart is one edge ID, never the delayed two-edge service path.
	c.Environment = append(c.Environment, weakEnvironment{Pair: c.Pairs[0], Side: "K", Edge: "D0", Other: "D10"})
	if r = checkWeakCertificate(p, c); !hasWeakIssue(r.Input, "environment") {
		t.Fatalf("nonimmediate certificate: %+v", r)
	}
}
func TestWeakRefinementF3DivergenceSeparateFromSafety(t *testing.T) {
	p := weakProblem{Rules: weakToyRules(), Demand: true, Guards: true,
		X: weakFiniteGraph{Initial: []string{"x0"}, Complete: true, Nodes: map[string]weakNode{"x0": weakToyNode("0", 1, "")},
			Edges: []weakEdge{{ID: "T", From: "x0", To: "x0", Rule: "tau"}}},
		K: weakFiniteGraph{Initial: []string{"k0"}, Complete: true, Boundaries: map[string]string{"done": "1"},
			Nodes: map[string]weakNode{"k0": weakToyNode("0", 1, ""), "k1": weakToyNode("1", 0, "done")},
			Edges: []weakEdge{{ID: "A", From: "k0", To: "k1", Rule: "a"}}}}
	c := weakCertificate{Pairs: []weakPair{{X: "x0", K: "k0"}},
		Component: []weakMatch{{Pair: weakPair{X: "x0", K: "k0"}, Edge: "T", Path: []string{}}}}
	r := requireWeak(t, p, c)
	if len(r.Strong) != 0 {
		t.Fatal(r.Strong)
	}
	if !hasWeakIssue(checkWeakRank(p.X), "rank") {
		t.Fatal("cycle rank accepted")
	}
	if issues := checkWeakCycle(p.X, p.Rules, []string{"T"}); len(issues) != 0 {
		t.Fatal(issues)
	}
	if issues := checkWeakCycle(p.X, p.Rules, []string{}); len(issues) == 0 {
		t.Fatal("empty cycle certificate accepted")
	}
}
func TestWeakRefinementF4VisibleOccurrence(t *testing.T) {
	p, _ := weakF1()
	p.X.Nodes = map[string]weakNode{
		"x0": weakToyNode("0", 2, ""), "x1": weakToyNode("1", 1, ""), "x2": weakToyNode("1", 0, "done"),
	}
	p.X.Edges = append(p.X.Edges, weakEdge{ID: "B", From: "x1", To: "x2", Rule: "b"})
	p.K.Nodes = map[string]weakNode{"k0": weakToyNode("0", 1, ""), "k1": weakToyNode("1", 0, "done")}
	p.K.Edges = []weakEdge{{ID: "AK", From: "k0", To: "k1", Rule: "a"}}
	c := weakCertificate{Pairs: []weakPair{{X: "x0", K: "k0"}, {X: "x1", K: "k1"}, {X: "x2", K: "k1"}},
		Component: []weakMatch{
			{Pair: weakPair{X: "x0", K: "k0"}, Edge: "A", Path: []string{"AK"}, Direct: "AK"},
			{Pair: weakPair{X: "x1", K: "k1"}, Edge: "B", Path: []string{}}}}
	r := checkWeakCertificate(p, c)
	if len(r.Input) != 0 || !hasWeakIssue(r.Transfer, "word") || len(r.Readiness) != 0 {
		t.Fatalf("extra b: %+v", r)
	}
	for _, g := range []weakFiniteGraph{p.X, p.K} {
		if issues := checkWeakRank(g); len(issues) != 0 {
			t.Fatal(issues)
		}
	}
}
func TestWeakRefinementEveryPairAndMove(t *testing.T) {
	p, c := weakF1()
	c.Component = c.Component[:2]
	if r := checkWeakCertificate(p, c); !hasWeakIssue(r.Transfer, "missing_component") {
		t.Fatalf("pair skipped: %+v", r)
	}
	p, c = weakF1()
	p.X.Edges = append(p.X.Edges, weakEdge{ID: "A2", From: "x0", To: "x1", Rule: "a"})
	if r := checkWeakCertificate(p, c); !hasWeakIssue(r.Transfer, "missing_component") {
		t.Fatalf("duplicate-label edge skipped: %+v", r)
	}
}
func TestWeakRefinementOrderedFactsAndGrammar(t *testing.T) {
	p, c := weakF1()
	r := p.Rules["a"]
	r.Word = []Observation{{Kind: "a"}, {Kind: "b"}, {Kind: "a"}}
	p.Rules["aba"] = r
	p.X.Edges[0].Rule = "aba"
	for i := range c.Component {
		c.Component[i].Direct = ""
	}
	if result := checkWeakCertificate(p, c); !hasWeakIssue(result.Transfer, "word") {
		t.Fatalf("multiplicity lost: %+v", result)
	}
	p, c = weakF1()
	p.K.Edges[0].Rule = "d"
	if result := checkWeakCertificate(p, c); !hasWeakIssue(result.Transfer, "component_owner") {
		t.Fatalf("environment substituted: %+v", result)
	}
}
func TestWeakRefinementUnreachableAndMalformed(t *testing.T) {
	p, c := weakF1()
	p.X.Nodes["unused"] = weakToyNode("denied", 0, "")
	requireWeak(t, p, c)
	p.X.Edges = append(p.X.Edges, weakEdge{ID: "dangling", From: "unused", To: "missing", Rule: "tau"})
	if r := checkWeakCertificate(p, c); len(r.Input) == 0 {
		t.Fatal("dangling unreachable edge accepted")
	}
}

func TestWeakRefinementStrongGuardsAndDirectIntegrity(t *testing.T) {
	t.Run("silent-current-guard", func(t *testing.T) {
		p, c := weakF1()
		n := p.K.Nodes["kl"]
		n.Guards = "changed"
		p.K.Nodes["kl"] = n
		r := requireWeak(t, p, c)
		if !hasWeakIssue(r.Strong, "guard") || !hasWeakIssue(r.Strong, "silent_guard") {
			t.Fatalf("guard crossing: %+v", r)
		}
	})
	t.Run("direct-must-occur-once", func(t *testing.T) {
		p, c := weakF1()
		p.K.Edges = append(p.K.Edges, weakEdge{ID: "back", From: "k1", To: "kl", Rule: "tau"})
		c.Component[0].Path = []string{"L", "AL", "back", "AL"}
		r := checkWeakCertificate(p, c)
		if !hasWeakIssue(r.Input, "direct_match") {
			t.Fatalf("repeated direct exemption: %+v", r)
		}
	})
}
func TestWeakRefinementProgressBoundaries(t *testing.T) {
	p, _ := weakF1()
	cases := []struct {
		name, clause string
		change       func(*weakFiniteGraph)
	}{
		{name: "stuck", clause: "stuck", change: func(g *weakFiniteGraph) { g.Edges = []weakEdge{} }},
		{name: "negative-rank", clause: "rank", change: func(g *weakFiniteGraph) {
			n := g.Nodes["x0"]
			n.Rank = -1
			g.Nodes["x0"] = n
		}},
		{name: "boundary-facts", clause: "boundary", change: func(g *weakFiniteGraph) { g.Boundaries["done"] = "wrong" }},
		{name: "open-ports", clause: "open_ports", change: func(g *weakFiniteGraph) { g.Complete = false }},
		{name: "dangling", clause: "edge", change: func(g *weakFiniteGraph) { g.Edges[0].To = "missing" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pp, _ := weakF1()
			g := pp.X
			tc.change(&g)
			if issues := checkWeakRank(g); !hasWeakIssue(issues, tc.clause) {
				t.Fatalf("rank/boundary certificate accepted: %+v", issues)
			}
		})
	}
	// A rank rejection is only a rejected positive certificate; do not call it
	// divergence. The explicit cycle checker also refuses competing work.
	p.X.Nodes["x0"] = weakToyNode("0", 0, "")
	if !hasWeakIssue(checkWeakRank(p.X), "rank") {
		t.Fatal("bad rank accepted")
	}
	if len(checkWeakCycle(p.X, p.Rules, []string{"A"})) == 0 {
		t.Fatal("visible path called private cycle")
	}
}
func TestWeakRefinementGrammarCannotEraseFacts(t *testing.T) {
	p, c := weakF1()
	r := p.Rules["a"]
	r.Private = true
	p.Rules["a"] = r
	if result := checkWeakCertificate(p, c); !hasWeakIssue(result.Input, "grammar") {
		t.Fatalf("visible occurrence declared private: %+v", result)
	}
	p, c = weakF1()
	r = p.Rules["d"]
	r.Owner = "candidate-choice"
	p.Rules["d"] = r
	if result := checkWeakCertificate(p, c); !hasWeakIssue(result.Input, "grammar") {
		t.Fatalf("undeclared owner: %+v", result)
	}
}

func TestWeakRefinementSilentInteriorsPreservePublicFacts(t *testing.T) {
	p, c := weakF1()
	c.Pairs = []weakPair{c.Pairs[0], c.Pairs[3]}
	c.Component = c.Component[:1]
	n := p.K.Nodes["kl"]
	n.Projection = "hidden public mutation"
	p.K.Nodes["kl"] = n
	r := checkWeakCertificate(p, c)
	if len(r.Input) != 0 || !hasWeakIssue(r.Transfer, "private_projection") {
		t.Fatalf("private interior changed public facts: %+v", r)
	}
}

// Review R1/P1: an absent alphabet must not silently classify every edge as
// neither component nor environment. The rank-only path still needs no grammar.
func TestWeakRefinementRequiredGrammar(t *testing.T) {
	for _, side := range []string{"X", "K"} {
		for _, owner := range []string{"component", "environment"} {
			for _, grammar := range []string{"nil", "empty", "missing"} {
				t.Run(side+"/"+owner+"/"+grammar, func(t *testing.T) {
					p := weakProblem{
						X: weakFiniteGraph{Initial: []string{"x0"}, Nodes: map[string]weakNode{
							"x0": weakToyNode("0", 1, ""), "x1": weakToyNode("1", 0, "")}},
						K: weakFiniteGraph{Initial: []string{"k0"}, Nodes: map[string]weakNode{
							"k0": weakToyNode("0", 1, ""), "k1": weakToyNode("1", 0, "")}},
					}
					edge := weakEdge{ID: "move", From: "x0", To: "x1", Rule: "required"}
					if side == "X" {
						p.X.Edges = []weakEdge{edge}
					} else {
						edge.From, edge.To = "k0", "k1"
						p.K.Edges = []weakEdge{edge}
					}
					// First bind the edge to its intended fixed grammar and
					// demonstrate the component/environment obligation.
					p.Rules = map[string]weakRule{
						"required": {Owner: owner, Word: []Observation{{Kind: "a"}}},
					}
					c := weakCertificate{Pairs: []weakPair{{X: "x0", K: "k0"}}}
					control := checkWeakCertificate(p, c)
					if len(control.Input) != 0 {
						t.Fatalf("valid declared grammar: %+v", control)
					}
					if owner == "environment" && !hasWeakIssue(control.Readiness, "missing_environment") {
						t.Fatalf("environment obligation not exercised: %+v", control)
					}
					if owner == "component" && side == "X" && !hasWeakIssue(control.Transfer, "missing_component") {
						t.Fatalf("component obligation not exercised: %+v", control)
					}
					switch grammar {
					case "nil":
						p.Rules = nil
					case "empty":
						p.Rules = map[string]weakRule{}
					case "missing":
						p.Rules = map[string]weakRule{
							"unrelated": {Owner: owner, Word: []Observation{{Kind: "a"}}},
						}
					}
					result := checkWeakCertificate(p, c)
					if result.accepted() || len(result.Input) == 0 || len(result.Transfer) != 0 || len(result.Readiness) != 0 {
						t.Fatalf("malformed %s grammar reached semantic assessment: %+v", grammar, result)
					}
				})
			}
		}
	}
	t.Run("explicit-empty-alphabet", func(t *testing.T) {
		p := weakProblem{Rules: map[string]weakRule{},
			X: weakFiniteGraph{Initial: []string{"x"}, Nodes: map[string]weakNode{"x": weakToyNode("0", 0, "")}},
			K: weakFiniteGraph{Initial: []string{"k"}, Nodes: map[string]weakNode{"k": weakToyNode("0", 0, "")}},
		}
		c := weakCertificate{Pairs: []weakPair{{X: "x", K: "k"}}}
		requireWeak(t, p, c)
		p.Rules = nil
		if result := checkWeakCertificate(p, c); len(result.Input) == 0 {
			t.Fatalf("absent grammar is not an explicit empty alphabet: %+v", result)
		}
	})
	t.Run("cycle-requires-grammar", func(t *testing.T) {
		g := weakFiniteGraph{Complete: true, Initial: []string{"x"},
			Nodes: map[string]weakNode{"x": weakToyNode("0", 1, "")},
			Edges: []weakEdge{{ID: "T", From: "x", To: "x", Rule: "tau"}},
		}
		if issues := checkWeakCycle(g, nil, []string{"T"}); !hasWeakIssue(issues, "grammar") {
			t.Fatalf("cycle must reject absent grammar before classification: %+v", issues)
		}
	})
	p, _ := weakF1()
	if issues := checkWeakRank(p.X); len(issues) != 0 {
		t.Fatalf("structural rank path changed: %+v", issues)
	}
}

// Review R1/P2: future demand equality is an endpoint clause. A lawful silent
// semantic choice may update demand while every selected current guard is stable.
func TestWeakRefinementSilentSemanticChoice(t *testing.T) {
	for _, changeGuard := range []bool{false, true} {
		name := "stable-guard"
		if changeGuard {
			name = "changed-guard"
		}
		t.Run(name, func(t *testing.T) {
			node := func(projection string, demand int, guard string) weakNode {
				return weakNode{Projection: projection, Demand: [3]int{demand, demand, 0}, Guards: guard}
			}
			p := weakProblem{Demand: true, Guards: true, Rules: map[string]weakRule{
				"local": {Owner: "component", Semantic: "choose_local"},
				"reply": {Owner: "component", Semantic: "reply", Word: []Observation{{Kind: "a"}}},
			}, X: weakFiniteGraph{Initial: []string{"x0"}, Nodes: map[string]weakNode{
				"x0": node("0", 2, "g"), "x1": node("1", 1, "g")},
				Edges: []weakEdge{{ID: "A", From: "x0", To: "x1", Rule: "reply"}}},
				K: weakFiniteGraph{Initial: []string{"k0"}, Nodes: map[string]weakNode{
					"k0": node("0", 2, "g"), "kL": node("0", 1, "g"), "k1": node("1", 1, "g")},
					Edges: []weakEdge{{ID: "L", From: "k0", To: "kL", Rule: "local"},
						{ID: "KA", From: "kL", To: "k1", Rule: "reply"}}},
			}
			if changeGuard {
				n := p.K.Nodes["kL"]
				n.Guards = "different-current-guard"
				p.K.Nodes["kL"] = n
			}
			c := weakCertificate{Pairs: []weakPair{{X: "x0", K: "k0"}, {X: "x1", K: "k1"}},
				Component: []weakMatch{{Pair: weakPair{X: "x0", K: "k0"}, Edge: "A", Path: []string{"L", "KA"}, Direct: "KA"}},
			}
			result := requireWeak(t, p, c)
			if changeGuard {
				if len(result.Strong) != 1 || result.Strong[0].Clause != "silent_guard" || result.Strong[0].Edge != "L" {
					t.Fatalf("only the changed current guard rejects this optional subset: %+v", result.Strong)
				}
			} else if len(result.Strong) != 0 {
				t.Fatalf("lawful semantic choice rejected despite equal endpoint demand/stable guards: %+v", result.Strong)
			}
		})
	}
}

// Review R1/P2: genuine private work cannot change selected public accounting
// or guards, even when equal endpoints and Direct would hide both mutations.
func TestWeakRefinementPrivateSelectedFields(t *testing.T) {
	node := func(demand int, guard string) weakNode {
		return weakNode{Projection: "0", Demand: [3]int{demand, demand, 0}, Guards: guard}
	}
	for _, field := range []string{"guard", "demand"} {
		for _, location := range []string{"candidate", "both-direct", "reference-interior"} {
			t.Run(field+"/"+location, func(t *testing.T) {
				before, after := node(2, "before"), node(2, "before")
				if field == "guard" {
					after.Guards = "after"
				} else {
					after.Demand = [3]int{1, 1, 0}
				}
				p := weakProblem{Demand: true, Guards: true, Rules: map[string]weakRule{
					"tau": {Owner: "component", Semantic: "private-calc", Private: true},
					"a":   {Owner: "component", Semantic: "reply", Word: []Observation{{Kind: "a"}}},
				}, X: weakFiniteGraph{Initial: []string{"x0"}, Nodes: map[string]weakNode{"x0": before, "x1": after},
					Edges: []weakEdge{{ID: "T", From: "x0", To: "x1", Rule: "tau"}}},
					K: weakFiniteGraph{Initial: []string{"k0"}, Nodes: map[string]weakNode{"k0": before}},
				}
				pair := weakPair{X: "x0", K: "k0"}
				c := weakCertificate{Pairs: []weakPair{pair, {X: "x1", K: "k0"}},
					Component: []weakMatch{{Pair: pair, Edge: "T", Path: []string{}}},
				}
				edges := []string{"T"}
				if location == "both-direct" {
					p.K.Nodes["k1"] = after
					p.K.Edges = []weakEdge{{ID: "KT", From: "k0", To: "k1", Rule: "tau"}}
					c.Pairs[1].K = "k1"
					c.Component[0].Path = []string{"KT"}
					c.Component[0].Direct = "KT"
					edges = append(edges, "KT")
				}
				if location == "reference-interior" {
					terminal := before
					terminal.Projection = "1"
					p.X.Nodes["x1"] = terminal
					p.X.Edges[0].Rule = "a"
					p.K.Nodes["ki"] = after
					p.K.Nodes["kr"] = before
					p.K.Nodes["k1"] = terminal
					p.K.Edges = []weakEdge{{ID: "P", From: "k0", To: "ki", Rule: "tau"},
						{ID: "Q", From: "ki", To: "kr", Rule: "tau"}, {ID: "KA", From: "kr", To: "k1", Rule: "a"}}
					c.Pairs[1].K = "k1"
					c.Component[0].Path = []string{"P", "Q", "KA"}
					c.Component[0].Direct = "KA"
					edges = []string{"P", "Q"}
				}
				result := requireWeak(t, p, c)
				for _, edge := range edges {
					found := false
					for _, issue := range result.Strong {
						if issue.Clause == "private_"+field && issue.Edge == edge {
							found = true
						}
					}
					if !found {
						t.Fatalf("genuine private %s mutation at %s was not checked: %+v", field, edge, result.Strong)
					}
				}
				if location != "candidate" && (hasWeakIssue(result.Strong, "demand") || hasWeakIssue(result.Strong, "guard")) {
					t.Fatalf("fixture must isolate private mutation despite equal related endpoints: %+v", result.Strong)
				}
			})
		}
	}
	for _, private := range []bool{true, false} {
		name := "actual-semantic-choice"
		if private {
			name = "unchanged-private-direct"
		}
		t.Run(name, func(t *testing.T) {
			before, after := node(2, "g"), node(2, "g")
			semantic := "private-calc"
			if !private {
				semantic = "choose_local"
				after.Demand = [3]int{1, 1, 0}
			}
			p := weakProblem{Demand: true, Guards: true, Rules: map[string]weakRule{
				"step": {Owner: "component", Semantic: semantic, Private: private},
			}, X: weakFiniteGraph{Initial: []string{"x0"}, Nodes: map[string]weakNode{"x0": before, "x1": after},
				Edges: []weakEdge{{ID: "T", From: "x0", To: "x1", Rule: "step"}}},
				K: weakFiniteGraph{Initial: []string{"k0"}, Nodes: map[string]weakNode{"k0": before, "k1": after},
					Edges: []weakEdge{{ID: "KT", From: "k0", To: "k1", Rule: "step"}}},
			}
			c := weakCertificate{Pairs: []weakPair{{X: "x0", K: "k0"}, {X: "x1", K: "k1"}},
				Component: []weakMatch{{Pair: weakPair{X: "x0", K: "k0"}, Edge: "T", Path: []string{"KT"}, Direct: "KT"}},
			}
			if result := requireWeak(t, p, c); len(result.Strong) != 0 {
				t.Fatalf("valid matched action: %+v", result.Strong)
			}
		})
	}
}
