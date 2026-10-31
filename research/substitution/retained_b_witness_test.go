package substitution

import (
	"fmt"
	"strconv"
	"testing"
)

func retainedBConfigs() []Config {
	var configs []Config
	for _, c := range LocalConfigurations() {
		if c.EntryDepth == 2 && c.Depth == 3 && c.I == 3 && c.R == 3 && c.Peer {
			configs = append(configs, c)
		}
	}
	return configs
}

func retainedBReference(c Config, s State) []Edge { return Successors(c, Reference, s) }

// Extract ordered semantic facts from one actual edge. Only reference choice
// bookkeeping changes visibility across the two readings. End always includes
// the committed source status even though the historical coarse label is empty.
func retainedBRule(e Edge, fact bool) weakRule {
	owner := "component" // O's forward-check partition also includes fixed caller edges.
	if e.Environment {
		owner = "environment"
	}
	event := e.Event
	if event.Kind == "terminal_freeze" {
		actor := entry
		if event.Operation == "h" {
			actor = helper
		}
		event.Status = map[int]string{success: "succeeded", failure: "failed"}[e.Next.Ops[actor].Freeze]
	}
	private := event.Kind == "private_step"
	r := weakRule{Owner: owner, Semantic: event.Kind, Private: private}
	if !private && !(fact && (event.Kind == "choose_local" || event.Kind == "choose_helper")) {
		r.Word = []Observation{event}
	}
	return r
}

// Stable edge IDs are local to a configuration and side. Addresses below name
// a supplied path by current source and the unique original operation/action.
// No alternative endpoint, closure or relation search is performed.
type retainedBTable struct {
	graph weakFiniteGraph
	ids   map[State]string
	steps map[string]map[string]weakEdge
}

func retainedBAction(e Observation) string { return e.Operation + "/" + e.Kind }

func retainedBTableFor(a automaton, fact bool, projections map[State]string, rules map[string]weakRule) retainedBTable {
	t := retainedBTable{
		graph: weakFiniteGraph{Initial: []string{"0"}, Nodes: map[string]weakNode{}, Complete: true},
		ids:   map[State]string{}, steps: map[string]map[string]weakEdge{},
	}
	for i, s := range a.states {
		id := strconv.Itoa(i)
		t.ids[s] = id
		p := publicState(s)
		// One observation function on both graphs. Enablement of the
		// branch is control bookkeeping. Preserve the possible ticket,
		// commitment, all later stages and the actual selected outcome.
		if p.Ops[entry].PC == invoke && p.Ops[helper].Ticket == possible {
			p.Ops[entry].PC = choose
			p.Ops[entry].Choice = 0
			p.Ops[helper].Enabled = false
		}
		pid, exists := projections[p]
		if !exists {
			pid = "p" + strconv.Itoa(len(projections))
			projections[p] = pid
		}
		t.graph.Nodes[id] = weakNode{Projection: pid}
		t.steps[id] = map[string]weakEdge{}
	}
	for i, edges := range a.edges {
		id := strconv.Itoa(i)
		for _, e := range edges {
			action := retainedBAction(e.Event)
			if _, exists := t.steps[id][action]; exists {
				panic("duplicate fixed original action " + id + action)
			}
			rule := retainedBRule(e, fact)
			// Include the entire observable payload in the alphabet key; event
			// names alone do not determine admission, reason or response values.
			label := fmt.Sprintf("%#v/%#v", rule.Owner, rule.Word)
			label += "/" + rule.Semantic
			rules[label] = rule
			edge := weakEdge{ID: id + "/" + action, From: id, To: t.ids[e.Next], Rule: label}
			t.graph.Edges = append(t.graph.Edges, edge)
			t.steps[id][action] = edge
		}
	}
	return t
}

func retainedBWitness(t *testing.T, c Config, fact bool) (weakProblem, weakCertificate) {
	t.Helper()
	xa := buildAutomaton(c, retainedBSuccessors)
	ka := buildAutomaton(c, retainedBReference)
	rules := map[string]weakRule{}
	projections := map[State]string{}
	x := retainedBTableFor(xa, fact, projections, rules)
	k := retainedBTableFor(ka, fact, projections, rules)
	cert := weakCertificate{}
	for i, s := range xa.states {
		xid := strconv.Itoa(i)
		kid, exists := k.ids[retainedBReferencePair(s)]
		if !exists {
			t.Fatalf("missing supplied reference endpoint at %v", s)
		}
		pair := weakPair{xid, kid}
		cert.Pairs = append(cert.Pairs, pair)
		for _, e := range xa.edges[i] {
			action := retainedBAction(e.Event)
			xe := x.steps[xid][action]
			if e.Environment {
				cert.Environment = append(cert.Environment, weakEnvironment{pair, "X", xe.ID, k.steps[kid][action].ID})
				continue
			}
			m := weakMatch{Pair: pair, Edge: xe.ID}
			at := kid
			if e.Event.Kind == "invoke_commit" && e.Event.Operation == "h" {
				choice := k.steps[at]["e/choose_helper"]
				m.Path = append(m.Path, choice.ID)
				at = choice.To
			}
			if e.Event.Kind != "private_step" {
				direct := k.steps[at][action]
				m.Path = append(m.Path, direct.ID)
				m.Direct = direct.ID
			}
			cert.Component = append(cert.Component, m)
		}
		for _, e := range ka.edges[ka.ids[retainedBReferencePair(s)]] {
			if e.Environment {
				action := retainedBAction(e.Event)
				cert.Environment = append(cert.Environment, weakEnvironment{pair, "K", k.steps[kid][action].ID, x.steps[xid][action].ID})
			}
		}
	}
	return weakProblem{X: x.graph, K: k.graph, Rules: rules}, cert
}

func TestRetainedBOriginalWitness(t *testing.T) {
	configs := retainedBConfigs()
	if len(configs) != 16 {
		t.Fatal("incomplete original domain", len(configs))
	}
	for _, c := range configs {
		t.Run(ConfigName(c), func(t *testing.T) {
			for _, fact := range []bool{true, false} {
				p, cert := retainedBWitness(t, c, fact)
				result := checkWeakCertificate(p, cert)
				t.Logf("fact=%v X=%d/%d K=%d/%d pairs=%d component=%d bilateral=%d word defects=%d",
					fact, len(p.X.Nodes), len(p.X.Edges), len(p.K.Nodes), len(p.K.Edges), result.Pairs,
					result.Components, result.Environments, len(result.Transfer))
				if fact {
					if !result.accepted() {
						t.Fatalf("F_fact rejected: %+v", result)
					}
					continue
				}
				if len(result.Input) != 0 || len(result.Readiness) != 0 || len(result.Transfer) == 0 {
					t.Fatalf("unexpected F_event disposition: %+v", result)
				}
				for _, issue := range result.Transfer {
					if issue.Clause != "word" || issue.Edge != issue.Pair.X+"/h/invoke_commit" {
						t.Fatalf("unexpected F_event rejection: %+v", issue)
					}
				}
			}
		})
	}
}

func TestRetainedBOpenProgress(t *testing.T) {
	for _, c := range retainedBConfigs() {
		c.Closure = false
		t.Run(ConfigName(c), func(t *testing.T) {
			a := buildAutomaton(c, retainedBSuccessors)
			edges, maxima, highest := 0, 0, 0
			for i, s := range a.states {
				highest = max(highest, graphRank(s))
				for _, edge := range a.edges[i] {
					edges++
					if graphRank(edge.Next) <= graphRank(s) {
						t.Fatalf("nonincreasing progress rank on %v", edge.Event)
					}
				}
				if len(a.edges[i]) != 0 {
					continue
				}
				maxima++
				if err := CheckCallableJoinLocalMaximum(c, Helper, s); err != nil {
					t.Fatal(err)
				}
				if c.Request.Mode == "pending" && s.PrivatePC[entry] != 12 {
					t.Fatalf("pending at wrong retained PC %d", s.PrivatePC[entry])
				}
			}
			if maxima == 0 {
				t.Fatal("no reachable maxima")
			}
			t.Logf("states=%d edges=%d maxima=%d highest increasing rank=%d", len(a.states), edges, maxima, highest)
		})
	}
}
