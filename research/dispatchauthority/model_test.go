package dispatchauthority

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func profile(t *testing.T, name string) Profile {
	t.Helper()
	for _, p := range Profiles() {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("no profile %q", name)
	return Profile{}
}

// walk accepts event projections supplied by the caller; it must not extrapolate
// an absent transition just because the desired outcome is locally plausible.
func walk(p Profile, s State, actions ...string) (State, error) {
	for _, a := range actions {
		found := false
		for _, step := range Outgoing(p, s) {
			if step.Action == a {
				s = step.State
				found = true
				break
			}
		}
		if !found {
			return s, fmt.Errorf("%s not enabled: %+v", a, s)
		}
	}
	return s, nil
}
func follow(t *testing.T, p Profile, s State, actions ...string) State {
	t.Helper()
	n, err := walk(p, s, actions...)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func reject(t *testing.T, p Profile, s State, actions ...string) {
	t.Helper()
	if _, err := walk(p, s, actions...); err == nil {
		t.Fatalf("accepted invalid trace %q", actions)
	}
}

func TestAllEdges(t *testing.T) {
	for _, p := range Profiles() {
		t.Run(p.Name, func(t *testing.T) {
			g := Explore(p)
			initial := g.Rows[0].State
			edges, sinks := 0, 0
			for _, row := range g.Rows {
				s := row.State
				l := s.Local
				if l.Key != initial.Local.Key || l.Grant != initial.Local.Grant || l.Request != initial.Local.Request || l.Debit != initial.Local.Debit || l.Reservations != 1 {
					t.Fatalf("changed original binding at %d", row.ID)
				}
				if l.Record != "missing" && s.Applications != 1 {
					t.Fatalf("evidence without correlated application at %d", row.ID)
				}
				if s.Applications > 1 {
					t.Fatalf("duplicate application at %d", row.ID)
				}
				if row.Wait == "quiescent" && (s.Sticky || occupied(s) || l.Record == "recorded") {
					t.Fatalf("premature quiescence at %d", row.ID)
				}
				if len(row.Outgoing) == 0 {
					sinks++
				}
				seen := map[string]bool{}
				for _, e := range row.Outgoing {
					edges++
					n := g.Rows[e.To].State
					if seen[e.Action] {
						t.Fatalf("duplicated action %s", e.Action)
					}
					seen[e.Action] = true
					if s.Ack && strings.HasPrefix(e.Action, "worker.enter.") {
						t.Fatalf("L after Q at %d", row.ID)
					}
					if strings.HasPrefix(e.Action, "gate.admit.") && (s.Owner != "open" || sourceRefusal(s) != "") {
						t.Fatalf("admission after revocation at %d", row.ID)
					}
					if (strings.HasPrefix(e.Action, "gate.refuse-") || e.Action == "observe.Q") && (s.Local != n.Local || s.Applications != n.Applications || s.Wire != n.Wire) {
						t.Fatalf("revocation rewrote history at %d", row.ID)
					}
					if e.Action == "owner.kill" && (s.Local != n.Local || s.Wire != n.Wire || s.Applications != n.Applications) {
						t.Fatalf("kill rewrote durable records/world at %d", row.ID)
					}
				}
			}
			t.Logf("states=%d edges=%d sinks=%d", len(g.Rows), edges, sinks)
		})
	}
}

func TestRaceAndDuplicates(t *testing.T) {
	for _, route := range []string{"raw-keyed", "adapter"} {
		p := profile(t, route+"-cancel")
		s := Initial(p)
		reject(t, p, s, "operator.cancel", "gate.admit.0")
		r := follow(t, p, s, "operator.cancel", "observe.Q", "gate.refuse-source_closed.0")
		if r.Applications != 0 || r.Worker[0] != "refused" {
			t.Fatal(r)
		}
		a := follow(t, p, s, "gate.admit.0", "operator.cancel", "peer.commit")
		reject(t, p, a, "observe.Q")
		a = follow(t, p, a, "worker.enter.0", "worker.send.0", "endpoint.ingress.0", "endpoint.serve.0", "worker.return.0")
		reject(t, p, a, "observe.Q")
		a = follow(t, p, a, "account.record.0")
		reject(t, p, a, "observe.Q")
		a = follow(t, p, a, "account.settle.0", "observe.Q")
		if a.Local.Selected || a.Local.Record != "settled" {
			t.Fatal(a)
		}
	}
	p := profile(t, "two-prepared-direct")
	s := follow(t, p, Initial(p), "gate.admit.0", "gate.admit.1", "worker.enter.0", "operator.cancel", "worker.send.0", "endpoint.ingress.0", "endpoint.serve.0", "worker.return.0", "account.record.0", "account.settle.0")
	reject(t, p, s, "observe.Q")
	s = follow(t, p, s, "worker.enter.1", "worker.send.1", "endpoint.ingress.1", "endpoint.serve.1", "worker.return.1", "account.record.1", "account.settle.1", "observe.Q")
	if s.Applications != 1 || s.Worker != [2]string{"handled", "handled"} {
		t.Fatal(s)
	}
}

func TestScopesAndAccounting(t *testing.T) {
	p := profile(t, "nested-publish")
	s := follow(t, p, Initial(p), "operator.freeze-success")
	if Wait(p, s) != "authority-open" {
		t.Fatal(s)
	}
	reject(t, p, s, "gate.refuse-scope_closed.0")
	s = follow(t, p, s, "gate.admit.0", "worker.enter.0", "worker.send.0", "endpoint.ingress.0", "endpoint.serve.0", "worker.return.0", "account.record.0", "account.settle.0", "source.select", "operator.nested-publish", "observe.Q")
	if s.Applications != 1 {
		t.Fatal(s)
	}
	p = profile(t, "alias-source-ended-scope-open")
	s = follow(t, p, Initial(p), "operator.end", "gate.refuse-source_closed.0")
	reject(t, p, s, "observe.Q")
	p = profile(t, "raw-retry")
	s = follow(t, p, Initial(p), "operator.failure", "operator.retry", "gate.refuse-source_generation_changed.0", "observe.Q")
	if s.Local.Generation != 2 {
		t.Fatal(s)
	}
	p = profile(t, "account-error")
	s = follow(t, p, Initial(p), "gate.admit.0", "worker.enter.0", "operator.root-close", "worker.send.0", "endpoint.ingress.0", "endpoint.serve.0", "worker.return.0", "account.record.0", "account.error.0")
	if Wait(p, s) != "accounting-error" {
		t.Fatal(s)
	}
	reject(t, p, s, "observe.Q")
	p = profile(t, "owner-close-open")
	s = follow(t, p, Initial(p), "owner.begin-close", "gate.refuse-owner_closing.0", "owner.release", "owner.open")
	if Wait(p, s) != "authority-open" {
		t.Fatal(s)
	}
	follow(t, p, s, "gate.admit.1")
}

func TestRecoveryCutsAndAllServicedMaxima(t *testing.T) {
	for _, name := range []string{"raw-keyed-recover-open", "raw-plain-recover-open", "adapter-recover-open", "raw-keyed-recover-revoked"} {
		t.Run(name, func(t *testing.T) {
			p := profile(t, name)
			prefix := follow(t, p, Initial(p), "gate.admit.0")
			closed := p.Closure == "cancel"
			if closed {
				prefix = follow(t, p, prefix, "operator.cancel")
			}
			u0 := follow(t, p, prefix, "owner.kill")
			u1 := follow(t, p, prefix, "worker.enter.0", "worker.send.0", "endpoint.ingress.0", "endpoint.serve.0", "worker.return.0", "owner.kill")
			projected := u1
			projected.Wire = u0.Wire
			projected.Applications = u0.Applications
			if projected != u0 || u0.Applications != 0 || u1.Applications != 1 {
				t.Fatalf("cuts not locally equal/world distinct: %+v %+v", u0, u1)
			}
			for _, cut := range []State{u0, u1} {
				reopened := follow(t, p, cut, "owner.open")
				front := []State{reopened}
				seen := map[State]bool{reopened: true}
				maxima := 0
				for i := 0; i < len(front); i++ {
					s := front[i]
					edges := Outgoing(p, s)
					for _, e := range edges {
						if !seen[e.State] {
							seen[e.State] = true
							front = append(front, e.State)
						}
					}
					if len(edges) != 0 {
						continue
					}
					maxima++
					resolved := p.Route == "raw-keyed" && !closed
					if s.Local.Selected != resolved || (s.Local.Record == "settled") != resolved || (!resolved && s.Applications != cut.Applications) || (resolved && s.Applications != 1) || s.Ack != closed {
						t.Fatalf("wrong serviced maximal extension %+v", s)
					}
				}
				if maxima == 0 {
					t.Fatal("vacuous recovery")
				}
				t.Logf("initial applications=%d serviced states=%d maximal=%d", cut.Applications, len(front), maxima)
			}
		})
	}
}

func TestPacketAfterFreshOwnerQAndSurvivingRecord(t *testing.T) {
	p := profile(t, "raw-keyed-recover-revoked")
	held := follow(t, p, Initial(p), "gate.admit.0", "worker.enter.0", "worker.send.0", "endpoint.ingress.0", "operator.cancel")
	cut := follow(t, p, held, "owner.kill", "owner.open", "observe.Q", "gate.refuse-source_closed.1")
	reject(t, p, cut, "worker.enter.1")
	after := follow(t, p, cut, "endpoint.serve.0")
	if cut.Applications != 0 || after.Applications != 1 || after.Local != cut.Local {
		t.Fatal(cut, after)
	}
	cut = follow(t, p, held, "endpoint.serve.0", "worker.return.0", "account.record.0", "owner.kill", "owner.open")
	if Wait(p, cut) != "needs-settlement" {
		t.Fatal(cut)
	}
	reject(t, p, cut, "observe.Q")
	follow(t, p, cut, "recovery.settle", "observe.Q")
}

// These are executable adverse protocols, deliberately separate from the Go
// reference and the production Store. Channel handshakes impose the order.
func TestShortcutControls(t *testing.T) {
	t.Run("last-check", func(t *testing.T) {
		var mu sync.Mutex
		open := true
		checked := make(chan bool)
		release := make(chan struct{})
		call := make(chan bool)
		go func() {
			mu.Lock()
			allow := open
			mu.Unlock()
			checked <- allow
			<-release
			call <- allow
		}()
		if !<-checked {
			t.Fatal("initial refusal")
		}
		mu.Lock()
		open = false
		mu.Unlock()
		close(release)
		if !<-call {
			t.Fatal("missing later call")
		}
		t.Log("check(open) < release owner lock < R < L(mock call)")
	})
	t.Run("whole-call-owner-lock", func(t *testing.T) {
		var mu sync.Mutex
		holding := make(chan struct{})
		release := make(chan struct{})
		done := make(chan struct{})
		go func() { mu.Lock(); close(holding); <-release; mu.Unlock(); close(done) }()
		<-holding
		for _, who := range []string{"R", "peer.commit"} {
			if mu.TryLock() {
				mu.Unlock()
				t.Fatalf("%s acquired owner while call held", who)
			}
			t.Logf("%s cannot acquire owner while synthetic transport is held", who)
		}
		close(release)
		<-done
		mu.Lock()
		mu.Unlock()
		t.Log("owner becomes available only after call releases")
	})
	t.Run("split-check-register", func(t *testing.T) {
		var mu sync.Mutex
		open, live := true, 0
		checked := make(chan struct{})
		release := make(chan struct{})
		entered := make(chan struct{})
		go func() {
			mu.Lock()
			allowed := open
			mu.Unlock()
			close(checked)
			<-release
			if allowed {
				mu.Lock()
				live++
				mu.Unlock()
				close(entered)
			}
		}()
		<-checked
		mu.Lock()
		open = false
		q := live == 0
		mu.Unlock()
		if !q {
			t.Fatal("no premature Q")
		}
		close(release)
		<-entered
		t.Log("check(open) < R < empty registry Q < register < L(mock call)")
	})
}

func TestMemberCancelClosesItsScope(t *testing.T) {
	for _, p := range Profiles() {
		if p.Name != "no-receipt" {
			continue
		}
		for _, edge := range Outgoing(p, Initial(p)) {
			if edge.Action == "operator.cancel" {
				if edge.State.Local.Source != "cancelled" || edge.State.Local.Scope != "root-close" {
					t.Fatal("actual cancellation freezes the member's owning scope in the same transaction", edge.State.Local)
				}
				return
			}
		}
	}
	t.Fatal("missing registered cancellation case")
}
