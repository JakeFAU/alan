package substitution

import "testing"

func TestActualProbeInstructionRegression(t *testing.T) {
	c := Config{Shape: "bare", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 2}
	t.Run("weak-helper-zero", func(t *testing.T) {
		a := buildAutomaton(c, weakGraph(Helper))
		found := false
		for _, edges := range a.edges {
			for _, e := range edges {
				if e.Event.Kind == "reply_commit" && e.Event.Role == "helper" {
					found = true
					if e.Event.Value != 0 {
						t.Fatalf("actual probeHelper(true) emits0, got%d", e.Event.Value)
					}
				}
			}
		}
		if !found {
			t.Fatal("no helper send checked")
		}
	})
	t.Run("p-status-branch-join", func(t *testing.T) {
		g := equalGraph(false)
		a := buildAutomaton(c, g)
		found := false
		for _, s := range a.states {
			if !s.Ops[helper].Consumed || s.PrivatePC[entry] != 3 || s.PrivateRegisters[entry][3] != 1 {
				continue
			}
			found = true
			var branch State
			ok := false
			for _, e := range g(c, s) {
				if e.Event.Kind == "private_step" && e.Event.Operation == "e" {
					branch = e.Next
					ok = true
				}
			}
			if !ok || branch.PrivatePC[entry] != 4 {
				t.Fatal("actual P success branch must targetPC4")
			}
			ok = false
			for _, e := range g(c, branch) {
				if e.Event.Kind == "private_step" && e.Event.Operation == "e" {
					ok = true
					if e.Next.PrivatePC[entry] != 6 {
						t.Fatalf("actual P PC4 jumps directly toPC6 Send; gotPC%d", e.Next.PrivatePC[entry])
					}
				}
			}
			if !ok {
				t.Fatal("missing actual P jump")
			}
		}
		if !found {
			t.Fatal("no success branch checked")
		}
	})
}

func TestProbeProfileMonitorRejectsWrongHelperPayload(t *testing.T) {
	c := Config{Shape: "sequential", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 2}
	faulty := func(c Config, s State) []Edge {
		edges := actualProbeGraph("weak-b")(c, s)
		for i := range edges {
			if edges[i].Event.Kind == "reply_commit" && edges[i].Event.Role == "helper" {
				edges[i].Next.Ops[helper].ReplyValue = 1
				edges[i].Event.Value = 1
			}
		}
		return edges
	}
	result := Probe(c, faulty, true)
	found := false
	for _, f := range result.Counterexamples {
		if f.Code == "probe_output" {
			found = true
		}
	}
	if !found {
		t.Fatal("explicit weak profile oracle accepted helper1")
	}
	emit(t, "actual-weak-helper-negative", result)
}

func TestActualProbeContextMembership(t *testing.T) {
	for _, profile := range []string{"weak-a", "weak-b", "p", "q"} {
		t.Run(profile, func(t *testing.T) {
			weak := profile == "weak-a" || profile == "weak-b"
			shape := "parallel"
			if weak {
				shape = "sequential"
			}
			for limit := 1; limit <= 3; limit++ {
				c := Config{Shape: shape, Request: Request{1, 0, "answer"}, I: limit, R: limit, Depth: 2}
				for _, boundary := range []bool{false, true} {
					g := actualProbeGraph(profile)
					if boundary {
						g = RuntimeBoundaryGraph(g)
					}
					r := Probe(c, g, weak)
					if r.Stuck != 0 || r.RankFailures != 0 || len(r.Counterexamples) > 0 {
						t.Fatalf("%s: %+v", profile, r)
					}
					for _, o := range r.Outcomes {
						if _, err := CheckProbeContext(c, profile, o, boundary); err != nil {
							t.Fatal(err)
						}
					}
					impossible := r.Outcomes[0]
					impossible.E = 1
					if _, err := CheckProbeContext(c, profile, impossible, boundary); err == nil {
						t.Fatal("pure probe accepted an effect charge")
					}
				}
			}
		})
	}
}
