// Package effectprotocol is a finite, independently authored relation for one
// effect operation. It imports no runtime, adapter, or endpoint implementation.
package effectprotocol

const (
	Operation  = "N/source/1/emit/0"
	Invocation = "N/parent/1/invoke/0"
	ReplyOp    = "N/source/1/reply/0"
	Allowed    = "authorized"
	Receipt    = "receipt:first-application"
)

type Profile struct {
	Name, Route, Closure, Reply                  string
	Idempotent, Member, NoGrant, ZeroLimit       bool
	Crash, SecondRun, Peer, Parent, PrivateFault bool
}

// Local contains retained fields. Epoch is the source's physical activation.
// Program is the source control quotient. Reply progresses independently of
// source termination once its intent is committed.
type Local struct {
	ResultDetail, ParentDecision                       string
	Program, Ancestor, Permission, Reservation, Result string
	Epoch, PermissionEpoch, Charge                     int
	Settled, Selected, PeerDone                        bool
	Reply, Published, ParentSelected                   string
}
type Volatile struct {
	Process, Run int
	Work         string
}
type World struct {
	Ingress, ValidRequests, Applications int
}
type Oracle struct {
	WorkerEntries, Executes, Launches, HTTPCalls, Records, Kills int
}
type State struct {
	Local    Local
	Volatile Volatile
	World    World
	Oracle   Oracle
}
type Action struct{ Name, On string }
type Edge struct {
	Action Action
	To     State
}

func Initial() State {
	return State{
		Local:    Local{Program: "emit", Ancestor: "open", Epoch: 1},
		Volatile: Volatile{Process: 1, Run: 1},
	}
}
func (l Local) open() bool {
	switch l.Program {
	case "cancelled", "retired", "failed", "ended":
		return false
	}
	return true
}

func reservation(p Profile, l Local) string {
	switch {
	case l.Permission != Allowed:
		return l.Permission
	case p.NoGrant:
		return "grant_denied"
	case p.Member && (l.Ancestor != "open" || !l.open()):
		return "scope_closed"
	case p.Member && p.ZeroLimit:
		return "effect_limit"
	default:
		return Allowed
	}
}

func denial(reason string) string {
	if reason == "grant_denied" {
		return reason
	}
	return "source_denied"
}

// Outgoing enumerates owners independently. Each add starts from the original
// state, including when two distinct enabled actions lead to equivalent states.
func Outgoing(p Profile, s State) []Edge {
	var edges []Edge
	add := func(name, on string, change func(*State)) {
		t := s
		change(&t)
		edges = append(edges, Edge{Action{name, on}, t})
	}
	l, v := s.Local, s.Volatile
	if v.Process == 0 {
		add("host.reopen", "executor", func(t *State) {
			t.Volatile = Volatile{Process: 2, Run: 2}
			if l.open() {
				t.Local.Epoch++
			}
		})
		return edges
	}
	if p.Peer && !l.PeerDone {
		add("peer.end", "peer/1", func(t *State) { t.Local.PeerDone = true })
	}
	if l.Program == "emit" {
		add("source.emit", Operation, func(t *State) { t.Local.Program = "after-emit" })
		return edges
	}
	if p.Closure != "" && l.Ancestor == "open" {
		switch p.Closure {
		case "source", "retire-source":
			if l.open() {
				add("operator."+p.Closure, "source/1", func(t *State) {
					t.Local.Program = "cancelled"
					if p.Closure == "retire-source" {
						t.Local.Program = "retired"
					}
				})
			}
		case "ancestor", "retire-ancestor":
			add("operator."+p.Closure, "ancestor/1", func(t *State) {
				t.Local.Ancestor = p.Closure
			})
		}
	}
	if l.Permission == "" {
		add("store.permission", Operation, func(t *State) {
			t.Local.Permission = Allowed
			t.Local.PermissionEpoch = l.Epoch
			if l.Program == "retired" {
				t.Local.Permission = "source_retired"
			} else if !l.open() {
				t.Local.Permission = "source_closed"
			}
		})
	}
	if l.Permission != "" && l.Reservation == "" {
		add("store.reserve", Operation, func(t *State) {
			t.Local.Reservation = reservation(p, l)
			if t.Local.Reservation == Allowed {
				if p.Member {
					t.Local.Charge++
				}
				t.Volatile.Work = "reserved"
			}
		})
	}
	if l.Reservation != "" && l.Reservation != Allowed && p.Route == "adapter" && l.Result == "" {
		add("store.record-denial", Operation, func(t *State) {
			t.Local.Result = denial(l.Reservation)
			t.Oracle.Records++
		})
	}
	if p.Crash && v.Process == 1 && (v.Work == "entered" || v.Work == "good" || l.Result != "" && !l.Selected) {
		add("environment.kill", "executor", func(t *State) {
			t.Volatile = Volatile{}
			t.Oracle.Kills++
		})
	}
	if p.SecondRun && v.Run == 1 && v.Work == "unanswered" && !l.open() {
		add("host.new-run", "executor", func(t *State) { t.Volatile = Volatile{Process: 1, Run: 2} })
	}
	if v.Run == 2 && v.Work == "" && l.Reservation == Allowed && l.Result == "" && p.Route == "raw" && p.Idempotent {
		add("runner.reexpose", Operation, func(t *State) { t.Volatile.Work = "reserved" })
	}
	workSteps(p, s, add)
	rawDenial := p.Route == "raw" && l.Reservation != "" && l.Reservation != Allowed
	if (l.Result != "" || rawDenial) && !l.Settled {
		add("store.settle", Operation, func(t *State) { t.Local.Settled = true })
	}
	if l.Program == "after-emit" {
		if p.PrivateFault {
			add("source.fault", "source/1", func(t *State) { t.Local.Program = "failed" })
		} else if l.Settled {
			add("source.select", Operation, func(t *State) {
				t.Local.Selected = true
				t.Local.Program = "guard"
				if p.Route == "raw" {
					t.Local.Program = "ending"
				}
			})
		}
	}
	if l.Program == "guard" {
		add("source.guard", Operation, func(t *State) {
			t.Local.Program = "ending"
			if l.Result != Receipt {
				t.Local.Program = "failing"
			} else if p.Parent {
				t.Local.Program = "emit-reply"
			}
		})
	}
	if l.Program == "failing" {
		add("source.fail", "source/1", func(t *State) { t.Local.Program = "failed" })
	}
	if l.Program == "emit-reply" {
		add("source.reply", ReplyOp, func(t *State) {
			t.Local.Reply = "committed"
			t.Local.Program = "ending"
		})
	}
	if l.Program == "ending" {
		add("source.end", "source/1", func(t *State) { t.Local.Program = "ended" })
	}
	if l.Reply == "committed" {
		add("store.reply-permission", ReplyOp, func(t *State) { t.Local.Reply = "allowed" })
	}
	if l.Reply == "allowed" {
		add("store.admit-reply", ReplyOp, func(t *State) { t.Local.Reply = "admitted" })
	}
	if p.Parent && l.Published == "" && !l.open() && (l.Reply == "" || l.Reply == "admitted") {
		add("store.publish-deliver", Invocation, func(t *State) { t.Local.Published = envelope(l) })
	}
	if p.Parent && l.Published != "" && l.ParentSelected == "" && l.Ancestor == "open" {
		add("parent.select", Invocation, func(t *State) { t.Local.ParentSelected = l.Published })
	}
	if l.ParentSelected != "" && l.ParentDecision == "" {
		add("parent.finish", "parent/1", func(t *State) { t.Local.ParentDecision = parentDecision(Parent(s)) })
	}
	return edges
}

func envelope(l Local) string {
	if l.Program == "failed" {
		return "failed/member_failed/no-response"
	}
	if l.Program != "ended" {
		return "cancelled/member_cancelled"
	}
	if l.Reply != "admitted" {
		return "failed/missing_reply/no-response"
	}
	if l.Result == Receipt {
		return "succeeded/member_succeeded/confirmed"
	}
	return "succeeded/member_succeeded/unconfirmed"
}

func workSteps(p Profile, s State, add func(string, string, func(*State))) {
	next := func(name, stage string) {
		add(name, Operation, func(t *State) { t.Volatile.Work = stage })
	}
	switch s.Volatile.Work {
	case "reserved":
		add("worker.enter", Operation, func(t *State) {
			t.Volatile.Work = "entered"
			t.Oracle.WorkerEntries++
		})
	case "entered":
		if p.Route == "adapter" {
			add("worker.execute", Operation, func(t *State) {
				t.Volatile.Work = "executing"
				t.Oracle.Executes++
				t.Oracle.Launches++
			})
		} else {
			add("worker.http", Operation, func(t *State) {
				t.Volatile.Work = "sent"
				t.Oracle.Launches++
				t.Oracle.HTTPCalls++
			})
		}
	case "executing":
		add("adapter.http", Operation, func(t *State) {
			t.Volatile.Work = "sent"
			t.Oracle.HTTPCalls++
		})
	case "sent":
		if p.Reply == "early-error" {
			next("transport.no-response", "error")
		} else {
			add("endpoint.ingress", Operation, func(t *State) {
				t.Volatile.Work = "arrived"
				t.World.Ingress++
			})
		}
	case "arrived":
		add("endpoint.apply", Operation, func(t *State) {
			t.Volatile.Work = "applied"
			t.World.ValidRequests++
			if !p.Idempotent || t.World.Applications == 0 {
				t.World.Applications++
			}
		})
	case "applied":
		switch {
		case p.Reply == "loss" || p.SecondRun && s.Volatile.Run == 1:
			if p.Route == "raw" {
				next("transport.no-response", "unanswered")
			} else {
				next("transport.no-response", "error")
			}
		case p.Reply == "chosen-error":
			next("adapter.choose-error", "chosen-error")
		case p.Reply == "unknown":
			next("adapter.return-unknown", "unknown")
		default:
			next("transport.obtain-receipt", "good")
		}
	case "unknown":
		next("store.leave-unresolved", "unanswered")
	case "good", "error", "chosen-error":
		add("store.record", Operation, func(t *State) {
			t.Local.Result = Receipt
			t.Local.ResultDetail = "validated-raw-receipt"
			switch s.Volatile.Work {
			case "error":
				t.Local.Result = "transport_error"
				t.Local.ResultDetail = "no-authentic-receipt"
			case "chosen-error":
				t.Local.Result = "selected_error"
			}
			t.Oracle.Records++
			t.Volatile.Work = "accounted"
		})
	}
}

func Step(p Profile, s State, a Action) (State, bool) {
	for _, e := range Outgoing(p, s) {
		if e.Action == a {
			return e.To, true
		}
	}
	return s, false
}
