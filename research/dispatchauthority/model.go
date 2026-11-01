package dispatchauthority

import "fmt"

// Local is the durable projection. Source always names the original source
// projection, even if the subject now has a newer current Generation.
type Local struct {
	Source, Scope, Record   string
	Generation              int
	Retired, Selected, Peer bool
	Key, Request, Grant     string
	Reservations, Debit     int
}

type State struct {
	Local        Local
	Owner        string
	Epoch        int
	Worker       [2]string
	Sticky, Ack  bool
	Wire         [2]string
	Applications int
}

type Transition struct {
	Action string
	State  State
}

func Initial(p Profile) State {
	s := State{
		Local: Local{Source: "open", Scope: "open", Record: "missing", Generation: 1,
			Key: "N/O", Request: "O@epoch1/delta1/" + p.Route, Grant: "allowed@revision1/epoch1", Reservations: 1},
		Owner: "open", Epoch: 1, Worker: [2]string{"ready", "unused"}, Wire: [2]string{"none", "none"},
	}
	if p.Member != "none" {
		s.Local.Debit = 1
	}
	if p.Duplicate {
		s.Worker[1] = "ready"
	}
	return s
}

func scopeClosed(s State) bool {
	return s.Local.Scope != "open" && s.Local.Scope != "success-unpublished"
}

func sourceRefusal(s State) string {
	switch {
	case s.Local.Generation != 1:
		return "source_generation_changed"
	case s.Local.Retired:
		return "source_retired"
	case s.Local.Source != "open":
		return "source_closed"
	case scopeClosed(s):
		return "scope_closed"
	default:
		return ""
	}
}

func isLive(w string) bool {
	switch w {
	case "admitted", "entered", "known", "empty", "recorded":
		return true
	default:
		return false
	}
}

func occupied(s State) bool { return isLive(s.Worker[0]) || isLive(s.Worker[1]) }

// Wait describes a Q attempt at this vertex. Service of an unsettled record
// requires another step and is never an immediate affirmative observation.
func Wait(p Profile, s State) string {
	switch {
	case s.Owner != "open":
		return "owner-unavailable"
	case p.Target == "scope" && !scopeClosed(s), p.Target == "source" && sourceRefusal(s) == "":
		return "authority-open"
	case s.Sticky:
		return "accounting-error"
	case occupied(s):
		return "pending-worker"
	case s.Local.Record == "recorded":
		return "needs-settlement"
	default:
		return "quiescent"
	}
}

func Outgoing(p Profile, s State) []Transition {
	edges := []Transition{}
	add := func(action string, next State) { edges = append(edges, Transition{Action: action, State: next}) }
	ownerChoices(p, s, add)
	if s.Owner == "open" {
		opChoices(p, s, add)
		if !s.Ack && Wait(p, s) == "quiescent" {
			n := s
			n.Ack = true
			add("observe.Q", n)
		}
		if p.Peer && !s.Local.Peer {
			n := s
			n.Local.Peer = true
			add("peer.commit", n)
		}
		if !s.Sticky && !occupied(s) && s.Local.Record == "recorded" {
			n := s
			n.Local.Record = "settled"
			add("recovery.settle", n)
		}
		awaits := p.Closure != "end" && p.Closure != "member-end" && p.Closure != "failure" && p.Closure != "retry-after-failure"
		if awaits && s.Local.Source == "open" && !s.Local.Selected && s.Local.Record == "settled" {
			n := s
			n.Local.Selected = true
			add("source.select", n)
		}
	}
	for i := range s.Worker {
		workerChoices(p, s, i, add)
	}
	return edges
}

func ownerChoices(p Profile, s State, add func(string, State)) {
	if s.Epoch != 1 {
		return
	}
	if p.Restart == "close" && s.Owner == "open" {
		n := s
		n.Owner = "closing"
		add("owner.begin-close", n)
	}
	if s.Owner == "closing" && !occupied(s) && s.Worker[0] != "ready" && s.Worker[1] != "ready" {
		n := s
		n.Owner = "released"
		n.Worker = [2]string{"unused", "unused"}
		add("owner.release", n)
	}
	cut := s.Worker[0] == "admitted" || s.Worker[0] == "known" || s.Worker[0] == "recorded" || (s.Worker[0] == "entered" && s.Wire[0] == "accepted")
	if p.Restart == "kill" && s.Owner == "open" && cut {
		n := s
		n.Owner = "dead"
		n.Worker = [2]string{"unused", "unused"}
		n.Sticky = false
		add("owner.kill", n)
	}
	if s.Owner == "dead" || s.Owner == "released" {
		n := s
		n.Owner = "open"
		n.Epoch++
		n.Sticky = false
		if p.Route == "raw-keyed" && s.Local.Record == "missing" {
			n.Worker[1] = "ready"
		}
		add("owner.open", n)
	}
}

func opChoices(p Profile, s State, add func(string, State)) {
	n := s
	if s.Local.Source == "open" {
		switch p.Closure {
		case "cancel", "retire":
			n.Local.Source = "cancelled"
			n.Local.Retired = p.Closure == "retire"
			if p.Member == "root" {
				n.Local.Scope = "root-close"
			}
			add("operator."+p.Closure, n)
		case "abandon":
			n.Local.Source = "abandoned"
			add("operator.abandon", n)
		case "failure", "retry-after-failure":
			n.Local.Source = "failed"
			if p.Member == "nested" {
				n.Local.Scope = "nested-fail"
			}
			add("operator.failure", n)
		case "end", "member-end":
			n.Local.Source = "succeeded"
			if p.Closure == "end" {
				n.Local.Scope = "root-close"
			}
			add("operator.end", n)
		}
	}
	if p.Closure == "retry-after-failure" && s.Local.Source == "failed" && s.Local.Generation == 1 {
		n.Local.Generation = 2
		add("operator.retry", n)
	}
	if s.Local.Scope == "open" {
		switch p.Closure {
		case "root-close", "nested-fail":
			n.Local.Scope = p.Closure
			add("operator."+p.Closure, n)
		case "nested-retire", "nested-publish":
			n.Local.Scope = "success-unpublished"
			add("operator.freeze-success", n)
		}
	}
	if s.Local.Scope == "success-unpublished" && (p.Closure == "nested-retire" || p.Closure == "nested-publish" && s.Local.Selected) {
		n.Local.Scope = p.Closure
		if p.Closure == "nested-publish" {
			n.Local.Source = "succeeded"
		}
		add("operator."+p.Closure, n)
	}
}

func workerChoices(p Profile, s State, i int, add func(string, State)) {
	emit := func(action string, next State) { add(fmt.Sprintf("%s.%d", action, i), next) }
	move := func(action, phase string) { n := s; n.Worker[i] = phase; emit(action, n) }
	// The endpoint can continue servicing transmitted work without a living
	// runtime or a worker still waiting for its response.
	switch s.Wire[i] {
	case "sent":
		n := s
		n.Wire[i] = "accepted"
		emit("endpoint.ingress", n)
	case "accepted":
		n := s
		n.Wire[i] = "served"
		if p.Route == "raw-plain" || s.Applications == 0 {
			n.Applications++
		}
		emit("endpoint.serve", n)
	}
	switch s.Worker[i] {
	case "ready":
		reason := sourceRefusal(s)
		if s.Owner != "open" {
			reason = "owner_closing"
		}
		if reason != "" {
			move("gate.refuse-"+reason, "refused")
			return
		}
		move("gate.admit", "admitted")
	case "admitted":
		move("worker.enter", "entered")
		if s.Owner == "closing" {
			move("worker.withdraw", "empty")
		}
	case "entered":
		if s.Wire[i] == "none" {
			n := s
			n.Wire[i] = "sent"
			emit("worker.send", n)
		}
		if p.Reply == "loss" {
			move("worker.loss", "empty")
		} else if s.Wire[i] == "served" {
			move("worker.return", "known")
		}
	case "known":
		n := s
		n.Worker[i] = "recorded"
		if n.Local.Record == "missing" {
			n.Local.Record = "recorded"
		}
		emit("account.record", n)
	case "recorded":
		n := s
		n.Worker[i] = "handled"
		if p.Reply == "account-error" {
			n.Sticky = true
			emit("account.error", n)
		} else {
			n.Local.Record = "settled"
			emit("account.settle", n)
		}
	case "empty":
		move("account.unknown", "handled")
	}
}
