package effectprotocol

import (
	"fmt"
	"strings"
)

// Profiles declares all choices before the initial state is constructed.
// A5's first Run loses a response after commit; the new Run is conditional on
// closure. A terminal prefix which never meets that condition is not continued.
func Profiles() []Profile {
	var rows []Profile
	push := func(p Profile, name string) {
		p.Name = name
		rows = append(rows, p)
	}
	for _, route := range []string{"raw", "adapter"} {
		for _, member := range []bool{false, true} {
			where := "unscoped"
			if member {
				where = "member"
			}
			for _, target := range []string{"source", "retire-source", "ancestor", "retire-ancestor"} {
				if !member && strings.Contains(target, "ancestor") {
					continue
				}
				push(Profile{Route: route, Member: member, Closure: target}, route+"-"+where+"-"+target)
			}
		}
		for _, idem := range []bool{false, true} {
			name := route + "-recovery-plain"
			if idem {
				name = route + "-recovery-keyed"
			}
			push(Profile{Route: route, Idempotent: idem, Crash: true}, name)
		}
		push(Profile{Route: route, NoGrant: true, Member: true}, route+"-no-grant")
		push(Profile{Route: route, ZeroLimit: true, Member: true}, route+"-zero-limit")
		push(Profile{Route: route, Idempotent: true, Peer: true}, route+"-peer")
	}
	push(Profile{Route: "raw", Idempotent: true, SecondRun: true, Closure: "source"}, "raw-second-run")
	for _, mode := range []string{"ok", "loss", "early-error", "chosen-error", "unknown"} {
		push(Profile{Route: "adapter", Parent: true, Member: true, Peer: true, Reply: mode}, "parent-"+mode)
	}
	push(Profile{Route: "adapter", Member: true, Parent: true, Peer: true, PrivateFault: true, Reply: "unknown"}, "parent-private-fault")
	push(Profile{Route: "adapter", Member: true, Parent: true, Peer: true, PrivateFault: true, Idempotent: true, Reply: "ok"}, "parent-private-fault-receipt")
	push(Profile{Route: "adapter", Member: true, Parent: true, Peer: true, Crash: true}, "parent-recovery")
	return rows
}

type Target struct {
	Action Action
	ID     int
}
type Row struct {
	ID       int
	State    State
	Outgoing []Target
}
type Graph struct {
	Profile Profile
	Rows    []Row
}

func Explore(p Profile) Graph {
	g := Graph{Profile: p}
	initial := Initial()
	ids := map[State]int{initial: 0}
	states := []State{initial}
	for i := 0; i < len(states); i++ {
		s := states[i]
		next := Outgoing(p, s)
		row := Row{ID: i, State: s, Outgoing: []Target{}}
		for _, edge := range next {
			id, found := ids[edge.To]
			if !found {
				id = len(states)
				ids[edge.To] = id
				states = append(states, edge.To)
			}
			row.Outgoing = append(row.Outgoing, Target{edge.Action, id})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}

type HostView struct {
	Local   Local
	Missing bool
}

func Host(s State) HostView {
	l := s.Local
	return HostView{l, l.Reservation == Allowed && l.Result == ""}
}

// The parent knows it has invoked the fixed child and may wait on its handle.
// Only the selected envelope can supply new application-specific information.
type ParentView struct{ Issued, Selected, Decision string }

func Parent(s State) ParentView {
	return ParentView{Invocation, s.Local.ParentSelected, s.Local.ParentDecision}
}

func parentDecision(v ParentView) string {
	if v.Selected == "succeeded/member_succeeded/confirmed" {
		return "historical_application_receipt"
	}
	return "local_failure_world_unresolved"
}

// Knowledge is an assertion concerning the particular historical mutation.
// The declared honest endpoint binds a positive receipt to its commit.
func Knowledge(s State) string {
	switch {
	case s.Local.Reservation != "" && s.Local.Reservation != Allowed:
		return "suppressed"
	case s.Local.ResultDetail == "validated-raw-receipt":
		return "applied"
	default:
		return "unresolved"
	}
}

func Frontier(p Profile, s State) string {
	if s.Local.Reservation == Allowed && s.Local.Result == "" {
		if s.Local.Program == "failed" {
			return "failed-source-unresolved"
		}
		if !s.Local.open() {
			return "closed-source-unresolved"
		}
		return "unresolved"
	}
	if !s.Local.open() && !s.Local.Selected {
		return "closed-source-unselected"
	}
	return "completed"
}

// Check the universal state invariants and serviced boundaries. The absence
// of cycles is verified separately rather than assuming fairness in a deadlock.
func Check(g Graph) error {
	p := g.Profile
	for _, r := range g.Rows {
		s := r.State
		l, w, o := s.Local, s.World, s.Oracle
		if o.Launches > 2 || w.Applications > w.ValidRequests || w.ValidRequests > w.Ingress || w.Ingress > o.HTTPCalls || o.HTTPCalls > o.Launches || o.Launches > o.WorkerEntries {
			return fmt.Errorf("state %d: physical count order", r.ID)
		}
		if o.Launches > 1 && !(p.Route == "raw" && p.Idempotent) || p.Idempotent && w.Applications > 1 {
			return fmt.Errorf("state %d: unauthorized repetition", r.ID)
		}
		expectedCharge := 0
		if p.Member && l.Reservation == Allowed {
			expectedCharge = 1
		}
		if l.Charge != expectedCharge || o.Launches > 0 && l.Reservation != Allowed || o.Records > 1 {
			return fmt.Errorf("state %d: resource or reservation causality", r.ID)
		}
		if Knowledge(s) == "applied" && w.Applications == 0 || Knowledge(s) == "suppressed" && o.Launches != 0 {
			return fmt.Errorf("state %d: invented world knowledge", r.ID)
		}
		if l.Selected && !l.Settled || l.Settled && l.Result == "" && !(p.Route == "raw" && l.Reservation != "" && l.Reservation != Allowed) {
			return fmt.Errorf("state %d: observation causality", r.ID)
		}
		seen := map[Action]bool{}
		for _, edge := range r.Outgoing {
			if edge.ID < 0 || edge.ID >= len(g.Rows) || seen[edge.Action] {
				return fmt.Errorf("bad edge at %d", r.ID)
			}
			seen[edge.Action] = true
		}
		if len(r.Outgoing) == 0 {
			if p.Peer && !l.PeerDone || l.Reservation == "" || l.Result != "" && !l.Settled || l.Settled && l.open() {
				return fmt.Errorf("state %d: withheld eligible work", r.ID)
			}
			if p.Parent && !l.open() && l.ParentDecision == "" {
				return fmt.Errorf("state %d: stranded parent", r.ID)
			}
			if p.Crash && p.Route == "raw" && p.Idempotent && !l.Selected {
				return fmt.Errorf("state %d: responsive same-key recovery made no progress", r.ID)
			}
		}
	}
	// Kahn's algorithm: every infinite interleaving would revisit a vertex.
	indegree := make([]int, len(g.Rows))
	for _, row := range g.Rows {
		for _, e := range row.Outgoing {
			indegree[e.ID]++
		}
	}
	queue := []int{}
	for id, d := range indegree {
		if d == 0 {
			queue = append(queue, id)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, e := range g.Rows[queue[i]].Outgoing {
			indegree[e.ID]--
			if indegree[e.ID] == 0 {
				queue = append(queue, e.ID)
			}
		}
	}
	if len(queue) != len(g.Rows) {
		return fmt.Errorf("reachable cycle")
	}
	return nil
}
