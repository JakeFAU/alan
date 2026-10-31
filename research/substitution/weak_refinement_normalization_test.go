package substitution

import (
	"fmt"
	"testing"
)

// This independently represented finite component has no A/B PCs or registers.
// Node accounting comes from these fields, rather than a history annotation.
type weakPrivateState struct {
	Control                                                                         string
	QSpent, Closed, Consumed, Computed, LocalChoice, HelperEnabled, HelperCommitted bool
	Result                                                                          int
}

func weakPrivateBoundary(control string, q, closed bool) weakPrivateState {
	s := weakPrivateState{Control: control, QSpent: q, Closed: closed, Consumed: control != "R"}
	s.Computed = control == "P1" || control == "L"
	if s.Computed {
		s.Result = 1
	}
	s.LocalChoice = control == "L"
	s.HelperEnabled = control == "H" || control == "C"
	s.HelperCommitted = control == "C"
	return s
}
func weakPrivateNode(s weakPrivateState) weakNode {
	q := 0
	if s.QSpent {
		q = 1
	}
	h := 1
	if s.LocalChoice {
		h = 0
	}
	return weakNode{
		Projection: fmt.Sprintf(
			"id1/value0/answer/spent%d,0/closed%t/consumed%t/hcommit%t",
			1+q, s.Closed, s.Consumed, s.HelperCommitted,
		),
		Demand: [3]int{1 - q + h, 1 - q + h, 0},
		Guards: fmt.Sprintf("q%t/close%t/source-e-open/caller-closed%t", !s.QSpent && !s.Closed, !s.Closed, s.Closed),
	}
}
func weakNormID(control string, q, closed bool) string {
	return fmt.Sprintf("%s/q%t/c%t", control, q, closed)
}
func weakNormOriginal(t *testing.T, control string, q, closed bool) State {
	t.Helper()
	c, s, _ := weakOriginalAnchor(t, false)
	if control != "R" {
		s = weakOriginalEdge(t, c, s, "e", "request_consume").Next
	}
	if control == "L" {
		s = weakOriginalEdge(t, c, s, "e", "choose_local").Next
	}
	if control == "H" || control == "C" {
		s = weakOriginalEdge(t, c, s, "e", "choose_helper").Next
	}
	if control == "C" {
		s = weakOriginalEdge(t, c, s, "h", "invoke_commit").Next
	}
	if q {
		s = weakOriginalEdge(t, c, s, "q", "peer_debit").Next
	}
	if closed {
		s = weakOriginalEdge(t, c, s, "", "close_root").Next
	}
	return s
}
func weakNormFixture(t *testing.T, anticipate, fork bool) (weakProblem, weakCertificate) {
	t.Helper()
	c, anchor, prefix := weakOriginalAnchor(t, false)
	if issues := checkWeakOriginalPrefix(c, prefix); len(issues) != 0 {
		t.Fatal(issues)
	}
	if anchor.Ops[entry].RequestStage != 1 || anchor.Ops[entry].PC != receive || anchor.Ops[entry].Freeze != none ||
		anchor.Ops[helper].Ticket != possible || anchor.Ops[helper].Enabled || anchor.I != 1 || anchor.R != 1 {
		t.Fatal("normalization anchor not original admitted/unconsumed state")
	}
	recv := weakOriginalEdge(t, c, anchor, "e", "request_consume")
	helper := weakOriginalEdge(t, c, recv.Next, "e", "choose_helper")
	commit := weakOriginalEdge(t, c, helper.Next, "h", "invoke_commit")
	p := weakProblem{Demand: true, Guards: true, Rules: map[string]weakRule{
		"recv":     {Owner: "component", Semantic: "request_consume", Word: []Observation{recv.Event}},
		"calc":     {Owner: "component", Private: true},
		"local":    {Owner: "component", Semantic: "choose_local"},
		"helper":   {Owner: "component", Semantic: "choose_helper"},
		"commit_h": {Owner: "component", Semantic: "invoke_commit_h", Word: []Observation{commit.Event}},
		"q": {Owner: "environment", Semantic: "peer_debit",
			Word: []Observation{{Kind: "peer_debit", Operation: "q", I: 1, R: 1}}},
		"close": {Owner: "environment", Semantic: "close_root", Word: []Observation{{Kind: "close_root"}}},
	}}
	p.X = weakFiniteGraph{Initial: []string{weakNormID("R", false, false)}, Nodes: map[string]weakNode{}}
	p.K = weakFiniteGraph{Initial: []string{weakNormID("R", false, false)}, Nodes: map[string]weakNode{}}
	xs := []string{"R", "P0", "P1", "L"}
	ks := []string{"R", "B", "L"}
	if fork {
		xs = append(xs, "H", "C")
		ks = append(ks, "H", "C")
	}
	makeGraph := func(g *weakFiniteGraph, controls []string, reference bool) {
		for _, q := range []bool{false, true} {
			for _, closed := range []bool{false, true} {
				for _, v := range controls {
					id := weakNormID(v, q, closed)
					s := weakPrivateBoundary(v, q, closed)
					g.Nodes[id] = weakPrivateNode(s)
					add := func(name, to, rule string) {
						g.Edges = append(g.Edges, weakEdge{ID: name + "/" + id, From: id, To: weakNormID(to, q, closed), Rule: rule})
					}
					if !q && !closed {
						g.Edges = append(g.Edges, weakEdge{ID: "q/" + id, From: id, To: weakNormID(v, true, false), Rule: "q"})
					}
					if !closed {
						g.Edges = append(g.Edges, weakEdge{ID: "close/" + id, From: id, To: weakNormID(v, q, true), Rule: "close"})
					}
					if reference {
						if v == "R" {
							add("receive", "B", "recv")
						}
						if v == "B" {
							add("choose", "L", "local")
							if fork {
								add("choose_helper", "H", "helper")
							}
						}
						if v == "H" {
							add("invoke_commit_h", "C", "commit_h")
						}
					} else {
						if v == "R" {
							add("recv", "P0", "recv")
						}
						if v == "P0" {
							add("calc", "P1", "calc")
						}
						if v == "P1" {
							add("local", "L", "local")
							if fork {
								add("helper", "H", "helper")
							}
						}
						if v == "H" {
							add("commit_h", "C", "commit_h")
						}
					}
				}
			}
		}
	}
	makeGraph(&p.X, xs, false)
	makeGraph(&p.K, ks, true)
	// Check the represented K state/edge account against actual original rules.
	original := map[string]State{}
	for _, q := range []bool{false, true} {
		for _, closed := range []bool{false, true} {
			for _, v := range ks {
				id := weakNormID(v, q, closed)
				s := weakNormOriginal(t, v, q, closed)
				original[id] = s
				if p.K.Nodes[id].Demand != weakTicketDemand(s) {
					t.Fatalf("ticket abstraction %s", id)
				}
				wantSpent := 1
				if q {
					wantSpent++
				}
				ledgerEqual := s.I == wantSpent && s.R == s.I && s.E == 0
				if !ledgerEqual || s.Closed != closed || s.Ops[entry].Freeze != none {
					t.Fatalf("original account %s", id)
				}
			}
		}
	}
	kind := map[string]string{"recv": "request_consume", "local": "choose_local", "helper": "choose_helper",
		"commit_h": "invoke_commit", "q": "peer_debit", "close": "close_root"}
	for _, edge := range p.K.Edges {
		op := "e"
		if edge.Rule == "commit_h" {
			op = "h"
		}
		if edge.Rule == "q" {
			op = "q"
		}
		if edge.Rule == "close" {
			op = ""
		}
		e := weakOriginalEdge(t, c, original[edge.From], op, kind[edge.Rule])
		if e.Next != original[edge.To] {
			t.Fatalf("original edge transcription %s", edge.ID)
		}
		word := p.Rules[edge.Rule].Word
		if edge.Rule != "local" && edge.Rule != "helper" && (len(word) != 1 || word[0] != e.Event) {
			t.Fatalf("original event transcription %s", edge.ID)
		}
	}
	cert := weakCertificate{}
	for _, q := range []bool{false, true} {
		for _, closed := range []bool{false, true} {
			for _, v := range xs {
				ref := v
				if v == "P0" || v == "P1" {
					ref = "B"
					if anticipate {
						ref = "L"
					}
				}
				pair := weakPair{X: weakNormID(v, q, closed), K: weakNormID(ref, q, closed)}
				cert.Pairs = append(cert.Pairs, pair)
				for _, e := range p.X.Edges {
					if e.From != pair.X {
						continue
					}
					if p.Rules[e.Rule].Owner == "environment" {
						other := "q/" + pair.K
						if e.Rule == "close" {
							other = "close/" + pair.K
						}
						cert.Environment = append(cert.Environment, weakEnvironment{Pair: pair, Side: "X", Edge: e.ID, Other: other},
							weakEnvironment{Pair: pair, Side: "K", Edge: other, Other: e.ID})
						continue
					}
					path := []string{}
					direct := ""
					switch e.Rule {
					case "recv":
						path = []string{"receive/" + pair.K}
						direct = path[0]
						if anticipate {
							path = append(path, "choose/"+weakNormID("B", q, closed))
						}
					case "local":
						if !anticipate {
							path = []string{"choose/" + pair.K}
							direct = path[0]
						}
					case "helper":
						if !anticipate {
							path = []string{"choose_helper/" + pair.K}
							direct = path[0]
						}
					case "commit_h":
						path = []string{"invoke_commit_h/" + pair.K}
						direct = path[0]
					}
					cert.Component = append(cert.Component, weakMatch{Pair: pair, Edge: e.ID, Path: path, Direct: direct})
				}
			}
		}
	}
	return p, cert
}
func TestWeakRefinementNormalizationRelations(t *testing.T) {
	for _, anticipate := range []bool{true, false} {
		t.Run(fmt.Sprintf("anticipate-%t", anticipate), func(t *testing.T) {
			p, c := weakNormFixture(t, anticipate, false)
			r := requireWeak(t, p, c)
			if anticipate {
				demandFailures := 0
				for _, i := range r.Strong {
					if i.Clause == "demand" {
						demandFailures++
					}
				}
				if demandFailures != 8 || len(r.Strong) != 8 {
					t.Fatalf("only eight endpoint-demand inequalities are frozen-clause failures: %+v", r.Strong)
				}
			} else if len(r.Strong) != 0 {
				t.Fatalf("direct local match incorrectly treated as unmatched stutter: %+v", r.Strong)
			}
			if r.Pairs != 16 || r.Components != 12 || r.Environments != 24 {
				t.Fatalf("finite fixture obligations: %+v", r)
			}
			if issues := checkWeakRank(p.X); !hasWeakIssue(issues, "open_ports") {
				t.Fatal("fragment progress asserted")
			}
			t.Logf("pairs=%d component=%d bilateral environment=%d strong failures=%d",
				r.Pairs, r.Components, r.Environments, len(r.Strong))
		})
	}
}
func TestWeakRefinementNormalizationHelperFork(t *testing.T) {
	p, c := weakNormFixture(t, false, true)
	r := requireWeak(t, p, c)
	if len(r.Strong) != 0 {
		t.Fatal(r.Strong)
	}
	p, c = weakNormFixture(t, true, true)
	r = checkWeakCertificate(p, c)
	if len(r.Input) != 0 || !hasWeakIssue(r.Transfer, "endpoint") {
		t.Fatalf("premature local pair survived helper fork: %+v", r)
	}
	t.Logf("fork rejected supplied premature strategy: %+v", r.Transfer)
}
func TestWeakRefinementNormalizationReadinessMutation(t *testing.T) {
	p, c := weakNormFixture(t, false, false)
	// Actual q remains immediately eligible during the independent calculation.
	id := "q/" + weakNormID("P0", false, false)
	for i, e := range p.X.Edges {
		if e.ID == id {
			p.X.Edges = append(p.X.Edges[:i], p.X.Edges[i+1:]...)
			break
		}
	}
	env := []weakEnvironment{}
	for _, e := range c.Environment {
		if e.Edge != id && e.Other != id {
			env = append(env, e)
		}
	}
	c.Environment = env
	r := checkWeakCertificate(p, c)
	if len(r.Input) != 0 || !hasWeakIssue(r.Readiness, "missing_environment") {
		t.Fatalf("private-PC q delay accepted: %+v", r)
	}
}
