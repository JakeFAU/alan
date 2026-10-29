package communication

import (
	"github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/research/substitution"
)

const RankBound = 23202

type Config struct {
	Request     substitution.Request
	I, R, Depth int
}

func PrimaryConfig() Config {
	return Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
}

type State struct {
	Join                     join.State
	AckStage                 int
	AckAllowed, AckDelivered bool
}
type Edge struct {
	Event join.Event
	Next  State
}

func baseConfig(c Config) join.Config {
	return join.Config{Request: c.Request, I: c.I, R: c.R, Depth: c.Depth, Context: join.CJoin}
}
func Initial(c Config) State { return State{Join: join.Initial(baseConfig(c))} }
func Rank(s State) int       { return join.Rank(s.Join) + s.AckStage }

// Successors retains all J1 primitive transitions except the two changed local
// terminal guards. In particular root Success remains enabled before ACK auth.
func Successors(c Config, v substitution.Variant, s State) []Edge {
	var edges []Edge
	for _, e := range join.Successors(baseConfig(c), v, s.Join) {
		if e.Event.Kind == "peer_terminal" && s.AckStage != 4 {
			continue
		}
		if e.Event.Kind == "root_terminal" && s.AckStage == 0 {
			continue
		}
		n := s
		n.Join = e.Next
		event := e.Event
		if event.Obligation == join.Obligation {
			event.Obligation = ReportObligation
		}
		edges = append(edges, Edge{Event: event, Next: n})
	}
	event := func(kind string, n State) join.Event {
		return join.Event{Kind: kind, Actor: "root", Operation: "ack", Source: "root", Target: "peer", Generation: 1, Sequence: n.Join.RootSequence, Obligation: AckObligation, Report: join.Report{RequestID: c.Request.ID, SOperation: "s"}}
	}
	if s.Join.ReportStage == 4 && s.AckStage == 0 && s.Join.Base.RootStatus == "" {
		n := s
		n.AckStage = 1
		n.Join.RootSequence++
		edges = append(edges, Edge{event("ack_commit", n), n})
	}
	if s.AckStage == 1 {
		n := s
		n.AckStage = 2
		n.AckAllowed = s.Join.Base.RootStatus == ""
		e := event("ack_authorize", n)
		e.Status = "allowed"
		e.Reason = "authorized"
		if !n.AckAllowed {
			e.Status = "refused"
			e.Reason = "source_closed"
		}
		edges = append(edges, Edge{e, n})
	}
	if s.AckStage == 2 {
		n := s
		n.AckStage = 3
		n.AckDelivered = s.AckAllowed && s.Join.Base.PeerStatus == ""
		e := event("ack_deliver", n)
		e.Status = "accepted"
		if !n.AckDelivered {
			e.Status = "refused"
			e.Reason = "source_denied:source_closed"
			if s.AckAllowed {
				e.Reason = "target_closed"
			}
		}
		edges = append(edges, Edge{e, n})
	}
	if s.AckStage == 3 && s.AckDelivered && s.Join.Base.PeerStatus == "" {
		n := s
		n.AckStage = 4
		n.Join.PeerSequence++
		e := event("ack_consume", n)
		e.Actor = "peer"
		e.Sequence = n.Join.PeerSequence
		edges = append(edges, Edge{e, n})
	}
	return edges
}

// InDomain checks the conservative finite coordinates frozen before enumeration.
func InDomain(s State) bool {
	between := func(x, lo, hi int) bool { return x >= lo && x <= hi }
	b := s.Join.Base
	if !between(b.I, 0, 3) || !between(b.R, 0, 3) || b.E != 0 || b.Closed || !between(b.RootPC, 0, 7) || !between(b.PeerPC, 0, 7) || !between(b.PeerTicket, 0, 3) {
		return false
	}
	if (b.RootStatus != "" && b.RootStatus != "succeeded") || (b.PeerStatus != "" && b.PeerStatus != "succeeded") || (s.Join.Aggregate != "" && s.Join.Aggregate != "succeeded") {
		return false
	}
	if !between(s.Join.Delay, 0, 32) || !between(s.Join.ReportStage, 0, 4) || !between(s.AckStage, 0, 4) || s.Join.RootSequence < 0 || s.Join.RootSequence > 4 || s.Join.PeerSequence < 0 || s.Join.PeerSequence > 36 {
		return false
	}
	for i, o := range b.Ops {
		if !between(o.Ticket, 0, 3) || !between(o.Admission, 0, 2) || !between(o.RequestStage, 0, 2) || !between(o.PC, 0, 8) || !between(o.Choice, 0, 2) || !between(o.Reply, 0, 3) || !between(o.Freeze, 0, 2) || !between(o.Depth, 0, 2) || !between(b.PrivatePC[i], 0, 15) {
			return false
		}
		for _, r := range b.PrivateRegisters[i] {
			if !between(r, 0, 1) {
				return false
			}
		}
	}
	return Rank(s) >= 0 && Rank(s) <= RankBound
}

// LifetimeState is the focused H-auth abstraction: one already committed
// ordinary message, an always-open destination and stable route/schema/grants.
// First 0/1/2 means absent/allowed/source_closed; admission 0/1/2 absent/accepted/refused.
// Retry returns the same state: stuttering reattempts cannot create new permission.
type LifetimeState struct {
	SourceClosed     bool
	First, Admission int
}

func (s LifetimeState) Reauthorize() LifetimeState { return s }
func LifetimeSuccessors(s LifetimeState) []LifetimeState {
	var out []LifetimeState
	if !s.SourceClosed {
		n := s
		n.SourceClosed = true
		out = append(out, n)
	}
	if s.First == 0 {
		n := s
		n.First = 1
		if s.SourceClosed {
			n.First = 2
		}
		out = append(out, n)
	}
	if s.First != 0 && s.Admission == 0 {
		n := s
		n.Admission = s.First
		out = append(out, n)
	}
	return out
}
