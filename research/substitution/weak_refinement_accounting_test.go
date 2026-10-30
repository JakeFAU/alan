package substitution

import (
	"testing"
)

func weakOriginalConfig() Config {
	return Config{Shape: "local", Request: Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 3,
		Local: true, Peer: true, Closure: true, EntryDepth: 2}
}
func weakOriginalEdge(t *testing.T, c Config, s State, op, kind string) Edge {
	t.Helper()
	matches := []Edge{}
	for _, e := range Successors(c, Reference, s) {
		if e.Event.Operation == op && e.Event.Kind == kind {
			matches = append(matches, e)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("%s/%s successors=%d", op, kind, len(matches))
	}
	return matches[0]
}
func weakOriginalAnchor(t *testing.T, consume bool) (Config, State, []Edge) {
	t.Helper()
	c := weakOriginalConfig()
	s := Initial(c)
	steps := []Edge{}
	for _, kind := range []string{"invoke_commit", "admit"} {
		e := weakOriginalEdge(t, c, s, "e", kind)
		steps = append(steps, e)
		s = e.Next
	}
	if consume {
		e := weakOriginalEdge(t, c, s, "e", "request_consume")
		steps = append(steps, e)
		s = e.Next
	}
	return c, s, steps
}
func weakTicketDemand(s State) [3]int {
	n := 0
	for _, o := range s.Ops {
		if o.Ticket == possible || o.Ticket == committed {
			n++
		}
	}
	if s.PeerTicket == possible || s.PeerTicket == committed {
		n++
	}
	return [3]int{n, n, 0}
}
func TestWeakRefinementOriginalBranchAccounting(t *testing.T) {
	c, b, prefix := weakOriginalAnchor(t, true)
	if issues := checkWeakOriginalPrefix(c, prefix); len(issues) != 0 {
		t.Fatal(issues)
	}
	if b.I != 1 || b.R != 1 || b.E != 0 || b.Ops[entry].PC != choose || b.Ops[entry].RequestStage != 2 ||
		b.Ops[helper].Ticket != possible || b.Ops[helper].Enabled || b.RootPC != await || b.Closed {
		t.Fatalf("original B anchor: %+v", b)
	}
	cases := []struct {
		name          string
		path          []string
		spent, demand int
	}{
		{name: "B", spent: 1, demand: 2}, {name: "L", path: []string{"e/choose_local"}, spent: 1, demand: 1},
		{name: "H", path: []string{"e/choose_helper"}, spent: 1, demand: 2},
		{name: "Bq", path: []string{"q/peer_debit"}, spent: 2, demand: 1},
		{name: "Lq", path: []string{"e/choose_local", "q/peer_debit"}, spent: 2, demand: 0},
		{name: "Hq", path: []string{"e/choose_helper", "q/peer_debit"}, spent: 2, demand: 1},
		{name: "Hcommit", path: []string{"e/choose_helper", "h/invoke_commit"}, spent: 1, demand: 2},
		{name: "Haccepted", path: []string{"e/choose_helper", "h/invoke_commit", "h/admit"}, spent: 2, demand: 1},
		{name: "HqAccepted", path: []string{"e/choose_helper", "q/peer_debit", "h/invoke_commit", "h/admit"},
			spent: 3, demand: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := b
			steps := append([]Edge{}, prefix...)
			for _, action := range tc.path {
				var op, kind string
				for i, v := range action {
					if v == '/' {
						op, kind = action[:i], action[i+1:]
						break
					}
				}
				e := weakOriginalEdge(t, c, s, op, kind)
				s = e.Next
				steps = append(steps, e)
			}
			if issues := checkWeakOriginalPrefix(c, steps); len(issues) != 0 {
				t.Fatal(issues)
			}
			if s.I != tc.spent || s.R != tc.spent || s.E != 0 || weakTicketDemand(s) != ([3]int{tc.demand, tc.demand, 0}) {
				t.Fatalf("spent/ticket demand: %+v %v", s, weakTicketDemand(s))
			}
		})
	}
	// Explore exactly the monotone branch/commit/admission/q/closure fragment;
	// omitted mode and later protocol actions are named ports, not disabled facts.
	queue := []State{b}
	seen := map[State]bool{b: true}
	edgeCount := 0
	for i := 0; i < len(queue); i++ {
		s := queue[i]
		qCount, closeCount := 0, 0
		for _, e := range Successors(c, Reference, s) {
			kind := e.Event.Kind
			if kind == "peer_debit" {
				qCount++
			}
			if kind == "close_root" {
				closeCount++
			}
			allowed := kind == "choose_local" || kind == "choose_helper" || kind == "peer_debit" || kind == "close_root" ||
				(e.Event.Operation == "h" && (kind == "invoke_commit" || kind == "admit"))
			if !allowed {
				continue
			}
			edgeCount++
			n := e.Next
			if n.I < s.I || n.R < s.R || n.E != s.E {
				t.Fatal("ledger refund")
			}
			closurePreserved := n.I == s.I && n.R == s.R &&
				weakTicketDemand(n) == weakTicketDemand(s) && n.Ops[entry].Freeze == none
			if kind == "close_root" && !closurePreserved {
				t.Fatal("closure changed member/demand")
			}
			if kind == "admit" {
				if s.Closed {
					if n.Ops[helper].AdmissionReason != "scope_closed" || n.I != s.I || n.Ops[helper].Ticket != excluded {
						t.Fatal("wrong closed helper admission")
					}
				} else {
					if n.I != s.I+1 || n.R != s.R+1 || n.Ops[helper].Ticket != spent || n.Ops[helper].Admission != 1 {
						t.Fatal("wrong open helper debit")
					}
				}
			}
			if !seen[n] {
				seen[n] = true
				queue = append(queue, n)
			}
		}
		wantQ := 0
		if !s.Closed && s.PeerTicket == possible && s.I < c.I && s.R < c.R {
			wantQ = 1
		}
		wantClose := 0
		if !s.Closed {
			wantClose = 1
		}
		if qCount != wantQ || closeCount != wantClose {
			t.Fatal("q/closure immediate readiness")
		}
	}
	t.Logf("original branch fragment: states=%d edges=%d; local/helper work and later protocol are ports",
		len(queue), edgeCount)
}
func TestWeakRefinementIllegalMergedHelperPrefix(t *testing.T) {
	c, b, prefix := weakOriginalAnchor(t, true)
	local := weakOriginalEdge(t, c, b, "e", "choose_local")
	helper := weakOriginalEdge(t, c, b, "e", "choose_helper")
	imported := weakOriginalEdge(t, c, helper.Next, "h", "invoke_commit")
	bad := append(append([]Edge{}, prefix...), local, imported)
	if issues := checkWeakOriginalPrefix(c, bad); !hasWeakIssue(issues, "original_transition") {
		t.Fatalf("illegal imported h: %+v", issues)
	}
	good := append(append([]Edge{}, prefix...), helper, imported)
	if issues := checkWeakOriginalPrefix(c, good); len(issues) != 0 {
		t.Fatal(issues)
	}
}
func TestWeakRefinementRawDemandMap(t *testing.T) {
	c, b, prefix := weakOriginalAnchor(t, true)
	l := weakOriginalEdge(t, c, b, "e", "choose_local")
	if issues := checkWeakOriginalPrefix(c, append(prefix, l)); len(issues) != 0 {
		t.Fatal(issues)
	}
	claims := []weakDemandClaim{{Raw: "u", Demand: [3]int{2, 2, 0}},
		{Raw: "m", Demand: [3]int{2, 2, 0}}, {Raw: "m", Demand: [3]int{1, 1, 0}}, {Raw: "z", Demand: [3]int{1, 1, 0}}}
	issues := checkWeakDemandMap(claims)
	if len(issues) != 1 || issues[0].Clause != "raw_demand_function" || issues[0].Detail != "m: [2 2 0] versus [1 1 0]" {
		t.Fatalf("raw-state function conflict: %+v", issues)
	}
	same := []weakDemandClaim{{Raw: "m", Demand: [3]int{1, 1, 0}}, {Raw: "m", Demand: [3]int{1, 1, 0}}}
	if issues = checkWeakDemandMap(same); len(issues) != 0 {
		t.Fatal(issues)
	}
	// Conditional F_fact/P_boundary alignments, with no environment/progress claim.
	reply := weakOriginalEdge(t, c, l.Next, "e", "reply_commit")
	p := weakProblem{Rules: map[string]weakRule{
		"keep": {Owner: "component", Private: true}, "local": {Owner: "component", Semantic: "choose_local"},
		"reply": {Owner: "component", Semantic: "reply_commit", Word: []Observation{reply.Event}}},
		X: weakFiniteGraph{Initial: []string{"u"},
			Nodes: map[string]weakNode{"u": {Projection: "consumed"},
				"m": {Projection: "consumed"},
				"z": {Projection: "reply"}},
			Edges: []weakEdge{{ID: "keep", From: "u", To: "m", Rule: "keep"},
				{ID: "local", From: "u", To: "m", Rule: "local"},
				{ID: "a", From: "m", To: "z", Rule: "reply"}}},
		K: weakFiniteGraph{Initial: []string{"B"},
			Nodes: map[string]weakNode{"B": {Projection: "consumed"},
				"L": {Projection: "consumed"},
				"S": {Projection: "reply"}},
			Edges: []weakEdge{{ID: "choose", From: "B", To: "L", Rule: "local"},
				{ID: "reply", From: "L", To: "S", Rule: "reply"}}}}
	ccert := weakCertificate{Pairs: []weakPair{{X: "u", K: "B"}, {X: "m", K: "B"}, {X: "m", K: "L"}, {X: "z", K: "S"}},
		Component: []weakMatch{
			{Pair: weakPair{X: "u", K: "B"}, Edge: "keep", Path: []string{}},
			{Pair: weakPair{X: "u", K: "B"}, Edge: "local", Path: []string{"choose"}, Direct: "choose"},
			{Pair: weakPair{X: "m", K: "B"}, Edge: "a", Path: []string{"choose", "reply"}, Direct: "reply"},
			{Pair: weakPair{X: "m", K: "L"}, Edge: "a", Path: []string{"reply"}, Direct: "reply"}}}
	requireWeak(t, p, ccert)
	if issues := checkWeakRank(p.X); !hasWeakIssue(issues, "open_ports") {
		t.Fatal("fragment progress asserted")
	}
	t.Logf("raw m annotations: %v versus %v; q/closure and later outcomes are ports",
		weakTicketDemand(b), weakTicketDemand(l.Next))
}
