package join

import "github.com/JakeFAU/alan/research/substitution"

const (
	C0         = "C0"
	CJoin      = "C_join"
	Obligation = "phase1j_s_join"
)
const (
	entry    = 0
	helper   = 1
	sibling  = 3
	awaiting = 4
	terminal = 7
)

type Config struct {
	Request     substitution.Request
	I, R, Depth int
	Context     string
}
type State struct {
	Base                           substitution.State
	Delay                          int
	Report                         Report
	ReportStage                    int // 0 absent, 1 committed, 2 authorized/refused, 3 admitted/refused, 4 consumed
	ReportAllowed, ReportDelivered bool
	RootSequence, PeerSequence     int64
	Aggregate                      string
}
type Edge struct {
	Event Event
	Next  State
}

func baseConfig(c Config) substitution.Config {
	return substitution.Config{Shape: "parallel", Request: c.Request, I: c.I, R: c.R, Depth: c.Depth}
}
func Initial(c Config) State { return State{Base: substitution.Initial(baseConfig(c))} }
func Configurations() []Config {
	var all []Config
	for id := 1; id <= 2; id++ {
		for v := 0; v <= 1; v++ {
			for d := 2; d <= 3; d++ {
				all = append(all, Config{Request: substitution.Request{ID: id, Value: v, Mode: "answer"}, I: 3, R: 3, Depth: d, Context: CJoin})
			}
		}
	}
	return all
}
func outcome(o substitution.Operation) Outcome {
	return Outcome{Status: o.Status, Reason: o.Reason, Present: o.Present, ID: o.ID, Value: o.Value}
}
func reportEvent(kind string, s State) Event {
	return Event{Kind: kind, Actor: "peer", Operation: "report", Source: "peer", Target: "root", Generation: 1, Obligation: Obligation, Report: s.Report, Sequence: s.PeerSequence}
}

// Successors changes only context control and report stages. The unchanged
// reviewed K1/environment graph and A/B instruction graphs provide primitives.
func Successors(c Config, v substitution.Variant, s State) []Edge {
	graph := substitution.CandidateGraph(v)
	if v == substitution.Reference {
		graph = func(c substitution.Config, s substitution.State) []substitution.Edge {
			return substitution.Successors(c, substitution.Reference, s)
		}
	}
	graph = substitution.RuntimeBoundaryGraph(graph)
	var edges []Edge
	for _, e := range graph(baseConfig(c), s.Base) {
		if e.Event.Operation == "s" && e.Event.Kind == "invoke_commit" && s.Delay < 32 {
			continue
		}
		n := s
		n.Base = e.Next
		ev := Event{Kind: e.Event.Kind, Operation: e.Event.Operation, Parent: e.Event.Parent, Role: e.Event.Role, Status: e.Event.Status, Reason: e.Event.Reason, I: e.Event.I, R: e.Event.R, E: e.Event.E, Outcome: Outcome{Status: e.Event.Status, Reason: e.Event.Reason, Present: e.Event.Present, ID: e.Event.ID, Value: e.Event.Value}}
		if ev.Kind == "outcome_consume" && ev.Operation == "e" {
			n.Base.RootPC = awaiting
			n.Base.RootStatus = ""
			n.RootSequence++
			ev.Actor = "root"
			ev.Sequence = n.RootSequence
		}
		if ev.Kind == "outcome_consume" && ev.Operation == "s" {
			n.Base.PeerPC = awaiting
			n.Base.PeerStatus = ""
			n.PeerSequence++
			ev.Actor = "peer"
			ev.Sequence = n.PeerSequence
		}
		edges = append(edges, Edge{ev, n})
	}
	add := func(n State, e Event) { edges = append(edges, Edge{e, n}) }
	if s.Delay < 32 {
		n := s
		n.Delay++
		n.PeerSequence++
		add(n, Event{Kind: "delay_set", Actor: "peer", Sequence: n.PeerSequence})
	}
	if c.Context == CJoin && s.Base.Ops[sibling].Consumed && s.ReportStage == 0 {
		n := s
		n.ReportStage = 1
		n.PeerSequence++
		n.Report = Report{RequestID: c.Request.ID, SOperation: "s", Outcome: outcome(s.Base.Ops[sibling])}
		add(n, reportEvent("report_commit", n))
	}
	if s.ReportStage == 1 {
		n := s
		n.ReportStage = 2
		n.ReportAllowed = s.Base.PeerStatus == ""
		e := reportEvent("report_authorize", n)
		e.Status = "allowed"
		e.Reason = "authorized"
		if !n.ReportAllowed {
			e.Status = "refused"
			e.Reason = "source_closed"
		}
		add(n, e)
	}
	if s.ReportStage == 2 {
		n := s
		n.ReportStage = 3
		n.ReportDelivered = s.ReportAllowed && s.Base.RootStatus == ""
		e := reportEvent("report_deliver", n)
		e.Status = "accepted"
		if !n.ReportDelivered {
			e.Status = "refused"
			e.Reason = "source_denied:source_closed"
			if s.ReportAllowed {
				e.Reason = "target_closed"
			}
		}
		add(n, e)
	}
	if s.ReportStage == 3 && s.ReportDelivered && s.Base.Ops[entry].Consumed && s.Base.RootStatus == "" {
		n := s
		n.ReportStage = 4
		n.RootSequence++
		e := reportEvent("report_consume", n)
		e.Actor = "root"
		e.Sequence = n.RootSequence
		add(n, e)
	}
	rootReady := s.Base.Ops[entry].Consumed && (c.Context == C0 || s.ReportStage == 4)
	if rootReady && s.Base.RootStatus == "" {
		n := s
		n.Base.RootPC = terminal
		n.RootSequence++
		n.Base.RootStatus = "failed"
		ok := validCandidate(c.Request, outcome(s.Base.Ops[entry]))
		if c.Context == CJoin {
			ok = ok && validS(c.Request, s.Report.Outcome)
		}
		if ok {
			n.Base.RootStatus = "succeeded"
		}
		add(n, Event{Kind: "root_terminal", Actor: "root", Sequence: n.RootSequence, Status: n.Base.RootStatus})
	}
	peerReady := s.Base.Ops[sibling].Consumed && (c.Context == C0 || s.ReportStage > 0)
	if peerReady && s.Base.PeerStatus == "" {
		n := s
		n.Base.PeerPC = terminal
		n.PeerSequence++
		n.Base.PeerStatus = "succeeded"
		if c.Context == C0 && !validS(c.Request, outcome(s.Base.Ops[sibling])) {
			n.Base.PeerStatus = "failed"
		}
		add(n, Event{Kind: "peer_terminal", Actor: "peer", Sequence: n.PeerSequence, Status: n.Base.PeerStatus})
	}
	if s.Aggregate == "" && s.Base.RootStatus != "" && s.Base.PeerStatus != "" {
		n := s
		n.Aggregate = "succeeded"
		if s.Base.RootStatus != "succeeded" || s.Base.PeerStatus != "succeeded" {
			n.Aggregate = "failed"
		}
		add(n, Event{Kind: "aggregate_publish", Status: n.Aggregate})
	}
	return edges
}

// Rank is bounded by finite fields and increases on every modeled transition.
// Sequence fields are not used as a surrogate execution-history/path rank.
func Rank(s State) int {
	r := 100*substitution.Rank(s.Base) + s.Base.PrivateRegisters[entry][7]
	for _, pc := range s.Base.PrivatePC {
		r += pc
	}
	r += s.Delay + s.ReportStage
	if s.Aggregate != "" {
		r++
	}
	return r
}
func demand(s State) int {
	n := 0
	for _, o := range s.Base.Ops {
		if o.Ticket < 2 {
			n++
		}
	}
	return n
}
