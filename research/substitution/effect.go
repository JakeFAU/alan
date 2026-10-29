package substitution

// EffectState is a separate K1E finite companion. This models neither A nor B.
// Source authorization and final reservation are distinct. Unknown outcomes do
// not enable reply or successful publication, including after retained reopen.
type EffectState struct {
	Entry, Effect                                                    int
	Authorized, Reserved, Launched, Known, Unknown, Closed, Reopened bool
	I, R, E                                                          int
	Reply, Frozen, Published                                         bool
}
type EffectEdge struct {
	Kind string
	Next EffectState
}
type EffectCensus struct {
	Scenario                                                                string
	States, Transitions, Revisits, Terminal, Pending, RankFailures, Cutoffs int
	Outcomes                                                                []EffectState
	ShortestTraces                                                          [][]string
	Violations                                                              []string
}

func effectRank(s EffectState) int {
	r := s.Entry + s.Effect + s.I + s.R + s.E
	for _, b := range []bool{s.Authorized, s.Reserved, s.Launched, s.Known, s.Unknown, s.Closed, s.Reopened, s.Reply, s.Frozen, s.Published} {
		if b {
			r++
		}
	}
	return r
}
func effectNext(s EffectState, scenario string) []EffectEdge {
	add := func(kind string, n EffectState) []EffectEdge { return []EffectEdge{{kind, n}} }
	if s.Entry < 3 {
		n := s
		n.Entry++
		kind := "entry_commit"
		if n.Entry == 2 {
			kind = "entry_admit"
			n.I = 1
			n.R = 1
		}
		if n.Entry == 3 {
			kind = "request_consume"
		}
		return add(kind, n)
	}
	if s.Effect == 0 {
		n := s
		n.Effect = 1
		return add("effect_commit", n)
	}
	if s.Effect == 1 {
		n := s
		n.Effect = 2
		n.Authorized = true
		return add("source_authorize", n)
	}
	if s.Effect == 2 {
		if scenario == "closure-before-reservation" && !s.Closed {
			n := s
			n.Closed = true
			return add("close_root", n)
		}
		n := s
		n.Effect = 3
		if !s.Closed {
			n.Reserved = true
			n.E = 1
		}
		return add("final_reservation", n)
	}
	if s.Effect == 3 && s.Reserved {
		if scenario == "reserved-then-closure" && !s.Closed {
			n := s
			n.Closed = true
			return add("close_root", n)
		}
		n := s
		n.Launched = true
		n.Effect = 4
		return add("original_launch", n)
	}
	if s.Effect == 4 {
		if scenario == "late-live" && !s.Closed {
			n := s
			n.Closed = true
			return add("close_root", n)
		}
		n := s
		n.Effect = 5
		if scenario == "unknown-reopen" {
			n.Unknown = true
			return add("unknown_attempt", n)
		}
		n.Known = true
		return add("authentic_receipt", n)
	}
	if s.Unknown && !s.Reopened {
		n := s
		n.Reopened = true
		return add("reopen_same_operation", n)
	}
	if s.Known && !s.Reply {
		n := s
		n.Reply = true
		return add("reply_commit", n)
	}
	if s.Reply && !s.Frozen {
		n := s
		n.Frozen = true
		return add("terminal_freeze", n)
	}
	if s.Frozen && !s.Published {
		n := s
		n.Published = true
		return add("publish_known_outcome", n)
	}
	return nil
}
func ExploreEffect(scenario string) EffectCensus {
	r := EffectCensus{Scenario: scenario}
	type node struct {
		s     EffectState
		trace []string
	}
	queue := []node{{}}
	seen := map[EffectState]bool{{}: true}
	for i := 0; i < len(queue); i++ {
		s := queue[i].s
		edges := effectNext(s, scenario)
		if len(edges) == 0 {
			if s.Unknown {
				r.Pending++
			} else {
				r.Terminal++
			}
			r.Outcomes = append(r.Outcomes, s)
			r.ShortestTraces = append(r.ShortestTraces, queue[i].trace)
		}
		for _, e := range edges {
			r.Transitions++
			n := e.Next
			if effectRank(n) <= effectRank(s) {
				r.RankFailures++
			}
			if n.E > 1 || (n.Launched && !n.Reserved) {
				r.Violations = append(r.Violations, "launch without unique final reservation")
			}
			if n.Reply && !n.Known {
				r.Violations = append(r.Violations, "reply without matching known result")
			}
			if n.Unknown && (n.Known || n.Reply || n.Frozen || n.Published) {
				r.Violations = append(r.Violations, "unknown inferred terminal outcome")
			}
			if seen[n] {
				r.Revisits++
			} else {
				seen[n] = true
				queue = append(queue, node{n, append(append([]string{}, queue[i].trace...), e.Kind)})
			}
		}
	}
	r.States = len(queue)
	return r
}
