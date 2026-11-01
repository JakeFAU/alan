package effectprotocol

import "testing"

func TestPrelaunchIsNotLaunch(t *testing.T) {
	p := Profile{Route: "raw"}
	s := Initial()
	for _, a := range []string{"source.emit", "store.permission", "store.reserve", "worker.enter"} {
		var ok bool
		s, ok = Step(p, s, Action{a, Operation})
		if !ok {
			t.Fatalf("refused %s", a)
		}
	}
	if s.Oracle.Launches != 0 || s.Local.Reservation != Allowed || s.World.Applications != 0 {
		t.Fatalf("held worker: %+v", s)
	}
	if _, ok := Step(p, s, Action{"endpoint.apply", Operation}); ok {
		t.Fatal("apply before launch")
	}
}

func TestAllReachable(t *testing.T) {
	for _, p := range Profiles() {
		t.Run(p.Name, func(t *testing.T) {
			g := Explore(p)
			if err := Check(g); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func take(t *testing.T, p Profile, s State, name string) State {
	t.Helper()
	for _, e := range Outgoing(p, s) {
		if e.Action.Name == name {
			return e.To
		}
	}
	t.Fatalf("refused %s at %+v", name, s)
	return s
}
func findCut(t *testing.T, g Graph, work string) State {
	t.Helper()
	for _, r := range g.Rows {
		s := r.State
		if s.Volatile.Work == work && s.Volatile.Process == 1 && !s.Local.PeerDone && s.Local.Program == "after-emit" {
			return s
		}
	}
	t.Fatalf("missing %s", work)
	return State{}
}
func complete(t *testing.T, p Profile, s State) State {
	t.Helper()
	for n := 0; n < 100; n++ {
		next := Outgoing(p, s)
		if len(next) == 0 {
			return s
		}
		s = next[0].To
	}
	t.Fatal("did not reach a frontier")
	return s
}
func TestPairedCuts(t *testing.T) {
	for _, p := range Profiles() {
		if !p.Crash {
			continue
		}
		t.Run(p.Name, func(t *testing.T) {
			g := Explore(p)
			a := take(t, p, findCut(t, g, "entered"), "environment.kill")
			b := take(t, p, findCut(t, g, "good"), "environment.kill")
			if Host(a) != Host(b) || Parent(a) != Parent(b) || a.Volatile != b.Volatile || a.World.Applications != 0 || b.World.Applications != 1 {
				t.Fatal("invalid comparison")
			}
			a = complete(t, p, a)
			b = complete(t, p, b)
			retry := p.Route == "raw" && p.Idempotent
			if a.Local.Selected != retry || b.Local.Selected != retry {
				t.Fatal("wrong progress")
			}
			if retry && (a.World.Applications != 1 || b.World.Applications != 1 || a.Oracle.Launches != 1 || b.Oracle.Launches != 2 || a.Local.Result != b.Local.Result) {
				t.Fatal("lost recovered receipt or duplicated mutation")
			}
			if !retry && (a.Oracle.Launches != 0 || b.Oracle.Launches != 1 || Knowledge(a) != "unresolved" || Knowledge(b) != "unresolved") {
				t.Fatal("blind repetition or invention")
			}
		})
	}
}

func TestGuardsAndOwners(t *testing.T) {
	for _, route := range []string{"raw", "adapter"} {
		t.Run(route, func(t *testing.T) {
			p := Profile{Route: route, Closure: "source"}
			g := Explore(p)
			held := findCut(t, g, "entered")
			for _, bad := range []Action{{"worker.http", "other/1"}, {"operator.source", Operation}, {"parent.select", Invocation}, {"endpoint.apply", Operation}, {"store.record", Operation}} {
				if _, ok := Step(p, held, bad); ok {
					t.Fatalf("accepted %+v", bad)
				}
			}
			closed := take(t, p, held, "operator.source")
			launch := "worker.http"
			if route == "adapter" {
				launch = "worker.execute"
			}
			after := take(t, p, closed, launch)
			if after.Oracle.Launches != 1 || closed.Oracle.Launches != 0 {
				t.Fatal("false physical launch count")
			}
			last := complete(t, p, after)
			if last.Local.Program != "cancelled" || last.Local.Selected || !last.Local.Settled || last.World.Applications != 1 {
				t.Fatalf("late authentic evidence: %+v", last)
			}
			first := take(t, p, take(t, p, Initial(), "source.emit"), "operator.source")
			last = complete(t, p, first)
			if last.Local.Reservation != "source_closed" || last.Oracle.Launches != 0 || Knowledge(last) != "suppressed" {
				t.Fatal("failure to suppress")
			}
		})
	}
}

func TestWithheldServiceMutant(t *testing.T) {
	p := Profile{Route: "raw", Idempotent: true, Crash: true}
	g := Explore(p)
	cut := take(t, p, take(t, p, findCut(t, g, "entered"), "environment.kill"), "host.reopen")
	for i, r := range g.Rows {
		if r.State == cut {
			if len(r.Outgoing) != 1 || r.Outgoing[0].Action.Name != "runner.reexpose" {
				t.Fatal("missing eligible work before mutation")
			}
			g.Rows[i].Outgoing = nil
			if Check(g) == nil {
				t.Fatal("accepted safety by withholding raw recovery")
			}
			return
		}
	}
	t.Fatal("missing reopened state")
}

func TestParentNoninterferenceAndErrors(t *testing.T) {
	finish := func(mode string) State {
		p := Profile{Route: "adapter", Parent: true, Member: true, Reply: mode}
		return complete(t, p, Initial())
	}
	applied, empty := finish("loss"), finish("early-error")
	if Host(applied) != Host(empty) || Parent(applied) != Parent(empty) || Parent(applied).Selected != "failed/member_failed/no-response" {
		t.Fatal("wrong projections")
	}
	if applied.World.Applications != 1 || empty.World.Applications != 0 || Host(applied).Missing || applied.Local.Result != "transport_error" {
		t.Fatal("invalid negative histories")
	}
	for _, s := range []State{applied, empty} {
		if Knowledge(s) != "unresolved" {
			t.Fatal("unsuccessful candidate settled mutation knowledge")
		}
	}
	// A policy is implementable on this pair only if it is constant on a
	// parent-observation class. This mutant illegally inspects the world.
	leaking := func(s State) bool { return s.World.Applications > 0 }
	if Parent(applied) != Parent(empty) || leaking(applied) == leaking(empty) {
		t.Fatal("missed oracle leak")
	}
	// Interpreting the failure default as proof of zero applications is
	// refuted by the real model history with a committed increment.
	if applied.Local.Result == "transport_error" && applied.World.Applications == 0 {
		t.Fatal("missing false-nonexecution counterexample")
	}
	p := Profile{Route: "adapter", Parent: true, Member: true, PrivateFault: true, Reply: "unknown"}
	g := Explore(p)
	for _, r := range g.Rows {
		s := r.State
		if Parent(s).Selected == "failed/member_failed/no-response" && Host(s).Missing && s.World.Applications == 1 && !s.Local.Selected {
			return
		}
	}
	t.Fatal("omitted independent fault/publication while effect unresolved")
}

func TestKeyAndEvidenceComparators(t *testing.T) {
	q := Request{"key1", Operation, "one"}
	w, first, _ := (Ledger{}).Submit(q, true)
	// A new endpoint process can be initialized solely from the durable ledger.
	w, dup, d := w.Submit(q, true)
	if first != dup || d != "duplicate" || w.Applications != 1 {
		t.Fatal("lost deduplication")
	}
	changed := q
	changed.Payload = "two"
	w, _, d = w.Submit(changed, true)
	if d != "key-conflict" || w.Contacts != 2 || w.Ingress != 3 {
		t.Fatal("altered request accepted")
	}
	w, _, d = w.Submit(q, false)
	if d != "key-conflict" || w.Contacts != 2 || w.Ingress != 4 || w.Applications != 1 {
		t.Fatal("changed endpoint profile reused a bound key", d, w)
	}
	retry := Request{"key2", "N/source/2/emit/0", "one"}
	w, _, d = w.Submit(retry, true)
	if d != "application" || w.Applications != 2 || first.Total != 1 || w.Total != 2 {
		t.Fatal("generation confused with business identity")
	}
	for _, tc := range []struct {
		name, on  string
		candidate Attestation
		decision  string
	}{
		{"wrong-generation", retry.Identity, first, "wrong-binding"},
		{"wrong-key", q.Identity, Attestation{"other", "one", 1, 1}, "wrong-binding"},
		{"conflict", q.Identity, Attestation{"key1", "one", 2, 2}, "conflicting-result"},
		{"equal", q.Identity, first, "equal-result"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, d := Retain(q, tc.on, first, tc.candidate)
			if got != first || d != tc.decision {
				t.Fatal("lost original evidence")
			}
		})
	}
	w = Ledger{}
	w, _, _ = w.Submit(q, false)
	w, r, _ := w.Submit(q, false)
	if w.Applications != 2 || r.Number != 2 {
		t.Fatal("fictional non-idempotent deduplication")
	}
}

func TestSingleSlotEffectfulPeer(t *testing.T) {
	orders := map[[2]int]bool{}
	for _, row := range PeerGraph() {
		a, b := row.State.Actors[0], row.State.Actors[1]
		if occupies(a) && occupies(b) {
			t.Fatal("overbooked slot")
		}
		if len(row.Outgoing) > 0 {
			continue
		}
		if !Host(a).Missing || a.Local.Selected || b.Local.Program != "ended" || !b.Local.Selected || row.State.Ordinal[0]+row.State.Ordinal[1] != 3 || a.World.Applications+b.World.Applications != 2 {
			t.Fatal("stranded peer or invented source result")
		}
		orders[row.State.Ordinal] = true
	}
	if len(orders) != 2 {
		t.Fatal("omitted an eligible first worker")
	}
}

func TestSelectedErrorCarriesHostEvidence(t *testing.T) {
	p := Profile{Route: "adapter", Parent: true, Member: true, Reply: "chosen-error"}
	s := complete(t, p, Initial())
	if s.Local.Result != "selected_error" || Knowledge(s) != "applied" || Parent(s).Decision != "local_failure_world_unresolved" {
		t.Fatal("failed to distinguish a retained authentic receipt from public typed defaults")
	}
}

// A parent policy may depend on the projection only. Compare all members of
// each supplied class and reject the first policy whose decisions separate them.
func realizable(states []State, policy func(State) string) bool {
	classes := map[ParentView]string{}
	for _, s := range states {
		p := Parent(s)
		decision := policy(s)
		if prior, ok := classes[p]; ok && decision != prior {
			return false
		}
		classes[p] = decision
	}
	return true
}
func TestRejectOraclePolicy(t *testing.T) {
	p := Profile{Route: "adapter", Parent: true, Member: true, Crash: true}
	g := Explore(p)
	var states []State
	for _, r := range g.Rows {
		states = append(states, r.State)
	}
	if !realizable(states, func(s State) string { return Parent(s).Decision }) {
		t.Fatal("rejected a parent-only policy")
	}
	if realizable(states, func(s State) string {
		if s.World.Applications != 0 {
			return "applied"
		}
		return "absent"
	}) {
		t.Fatal("accepted a policy that sees the oracle")
	}
}

// The ordinary parent may consume member_failed before the held adapter makes
// its POST. A responsive endpoint still returns authentic evidence afterward.
func TestPrivateFailureThenLateReceipt(t *testing.T) {
	for _, p := range Profiles() {
		if p.Name != "parent-private-fault-receipt" {
			continue
		}
		s := Initial()
		for _, act := range []string{"source.emit", "store.permission", "store.reserve", "worker.enter", "worker.execute", "source.fault", "store.publish-deliver", "parent.select", "parent.finish"} {
			s = take(t, p, s, act)
		}
		if s.World.Ingress != 0 || s.Local.Result != "" || s.Local.ParentDecision != "local_failure_world_unresolved" {
			t.Fatal("invalid held prefix", s)
		}
		final := complete(t, p, s)
		if final.World.Applications != 1 || final.Local.Result != Receipt || !final.Local.Settled || final.Local.Selected || final.Local.Program != "failed" || final.Local.ParentDecision != s.Local.ParentDecision {
			t.Fatal("missing late evidence", final)
		}
		return
	}
	t.Fatal("no responsive private-fault profile in exported family")
}
