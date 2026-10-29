package substitution

import "fmt"

// Config names one of the frozen context configurations. Local is a separate
// open-environment profile, never part of the 3,840 context census.
type Config struct {
	Shape                string
	Request              Request
	I, R, Depth          int
	Local, Peer, Closure bool
	EntryDepth           int
}
type Variant string

const Reference Variant = "K1"
const Local Variant = "local"
const Helper Variant = "helper"
const (
	entry   = 0
	helper  = 1
	wrapper = 2
	sibling = 3
	slots   = 4
)
const (
	possible  = 0
	committed = 1
	spent     = 2
	excluded  = 3
)
const (
	uncreated = 0
	receive   = 1
	choose    = 2
	invoke    = 3
	await     = 4
	mode      = 5
	sent      = 6
	terminal  = 7
	pending   = 8
)
const (
	none    = 0
	success = 1
	failure = 2
)

type Operation struct {
	ReturnedID, ReturnedValue                                  int
	ReturnedPresent                                            bool
	FailureReady                                               bool
	Enabled                                                    bool
	Role                                                       string
	Source, Depth                                              int
	Input                                                      Request
	Ticket, Admission, RequestStage, PC, Choice, Reply, Freeze int
	ReplyID, ReplyValue                                        int
	AdmissionReason, ReplyReason                               string
	Published, Delivered, Consumed                             bool
	Status, Reason                                             string
	Present                                                    bool
	ID, Value                                                  int
}

// State is comparable: every semantically live field participates in identity.
// There is no path-length/history field that could conceal revisits or cycles.
type State struct {
	// Private control/registers are interpreted only by separately transcribed graphs.
	PrivatePC        [slots]int
	PrivateRegisters [slots][8]int

	Ops                    [slots]Operation
	I, R, E                int
	Closed                 bool
	RootPC, PeerPC         int
	RootStatus, PeerStatus string
	RootID, RootValue      int
	RootPresent            bool
	PeerTicket             int
}
type Edge struct {
	Event       Observation
	Next        State
	Environment bool
}

func Entitled(c Config) bool {
	n, d := 2, 2
	if c.Local {
		if c.Peer {
			n++
		}
		d = c.EntryDepth + 1
	} else {
		switch c.Shape {
		case "sequential", "parallel":
			n = 3
		case "wrapped":
			n = 3
			d = 3
		}
	}
	return c.Request.Mode == "answer" && c.I >= n && c.R >= n && c.Depth >= d
}
func Initial(c Config) State {
	s := State{PeerPC: terminal, PeerTicket: excluded}
	d := 1
	if c.Local {
		d = c.EntryDepth
	}
	for i := range s.Ops {
		s.Ops[i].Ticket = excluded
	}
	s.Ops[entry] = Operation{Enabled: true, Role: "candidate", Source: -1, Depth: d, Input: c.Request}
	s.Ops[helper] = Operation{Role: "helper", Source: entry, Depth: d + 1, Input: Request{c.Request.ID, c.Request.Value, "answer"}}
	if c.Shape == "wrapped" && !c.Local {
		s.Ops[entry].Enabled = false
		s.Ops[entry].Source = wrapper
		s.Ops[entry].Depth = 2
		s.Ops[helper].Depth = 3
		s.Ops[wrapper] = Operation{Enabled: true, Role: "wrapper", Source: -1, Depth: 1, Input: c.Request}
	}
	if (c.Shape == "parallel" || c.Shape == "sequential") && !c.Local {
		src := -1
		if c.Shape == "parallel" {
			src = -2
			s.PeerPC = uncreated
		}
		s.Ops[sibling] = Operation{Enabled: c.Shape == "parallel", Role: "s", Source: src, Depth: 1, Input: c.Request}
	}
	if c.Local && c.Peer {
		s.PeerTicket = possible
	}
	return s
}
func name(i int) string { return [slots]string{"e", "h", "w", "s"}[i] }
func sourceOpen(s State, src int) bool {
	if src < 0 {
		return !s.Closed
	}
	return s.Ops[src].Freeze == none
}
func observable(s State, i int, kind string) Observation {
	o := s.Ops[i]
	p := "root"
	if o.Source == -2 {
		p = "peer"
	} else if o.Source >= 0 {
		p = name(o.Source)
	}
	return Observation{Kind: kind, Operation: name(i), Parent: p, Role: o.Role, Request: o.Input, ID: o.ID, Value: o.Value, Status: o.Status, Reason: o.Reason, Present: o.Present}
}
func exclude(s *State, i int) {
	if s.Ops[i].Ticket < spent {
		s.Ops[i].Ticket = excluded
	}
	if i == entry {
		exclude(s, helper)
	}
	if i == wrapper {
		exclude(s, entry)
	}
}
func rejection(c Config, s State, o Operation) string {
	switch {
	case !sourceOpen(s, o.Source):
		return "source_closed"
	case s.Closed:
		return "scope_closed"
	case o.Depth > c.Depth:
		return "depth_limit"
	case s.I >= c.I:
		return "invocation_limit"
	case s.R >= c.R:
		return "created_run_limit"
	}
	return ""
}

// Successors keeps intent, admission, request consumption, reply commitment,
// reply admission, terminal/freeze, publication, delivery and consumption distinct.
func Successors(c Config, v Variant, s State) []Edge {
	var out []Edge
	add := func(n State, i int, kind string, env bool) {
		e := observable(n, i, kind)
		if kind == "admit" {
			e.Status = "rejected"
			e.Reason = n.Ops[i].AdmissionReason
			if n.Ops[i].Admission == 1 {
				e.Status = "accepted"
				e.I = 1
				e.R = 1
			}
		}
		if kind == "reply_commit" || kind == "reply_admit" {
			e.ID = n.Ops[i].ReplyID
			e.Value = n.Ops[i].ReplyValue
			e.Present = n.Ops[i].Reply == 2
			e.Status = "committed"
			if n.Ops[i].Reply == 2 {
				e.Status = "accepted"
			}
			if n.Ops[i].Reply == 3 {
				e.Status = "refused"
				e.Reason = n.Ops[i].ReplyReason
			}
		}
		out = append(out, Edge{e, n, env})
	}
	if c.Closure && !s.Closed {
		n := s
		n.Closed = true
		n.RootStatus = "cancelled"
		n.RootPC = terminal
		if c.Shape == "parallel" {
			n.PeerStatus = "cancelled"
			n.PeerPC = terminal
		}
		out = append(out, Edge{Observation{Kind: "close_root"}, n, true})
	}
	if c.Local && s.PeerTicket == possible && !s.Closed && s.I < c.I && s.R < c.R {
		n := s
		n.PeerTicket = spent
		n.I++
		n.R++
		out = append(out, Edge{Observation{Kind: "peer_debit", Operation: "q", I: 1, R: 1}, n, true})
	}
	for i, o := range s.Ops {
		if o.Enabled && o.Ticket == possible && sourceOpen(s, o.Source) {
			n := s
			n.Ops[i].Ticket = committed
			if o.Source == -1 {
				n.RootPC = await
			}
			if o.Source == -2 {
				n.PeerPC = await
			}
			if o.Source >= 0 {
				n.Ops[o.Source].PC = await
			}
			add(n, i, "invoke_commit", false)
		}
		if o.Ticket == committed && o.Admission == 0 {
			n := s
			r := rejection(c, s, o)
			if r != "" {
				n.Ops[i].Admission = 2
				n.Ops[i].AdmissionReason = r
				exclude(&n, i)
			} else {
				n.Ops[i].Admission = 1
				n.Ops[i].Ticket = spent
				n.I++
				n.R++
				n.Ops[i].RequestStage = 1
				n.Ops[i].PC = receive
			}
			add(n, i, "admit", true)
		}
		if o.Admission == 1 && o.PC == receive && o.RequestStage == 1 {
			n := s
			n.Ops[i].RequestStage = 2
			n.Ops[i].PC = mode
			if i == entry {
				n.Ops[i].PC = choose
			}
			if i == wrapper {
				n.Ops[i].PC = invoke
				n.Ops[entry].Enabled = true
			}
			add(n, i, "request_consume", false)
		}
		if i == entry && o.PC == choose {
			if v == Reference || v == Local {
				n := s
				n.Ops[i].Choice = 1
				n.Ops[i].PC = mode
				exclude(&n, helper)
				add(n, i, "choose_local", false)
			}
			if v == Reference || v == Helper {
				n := s
				n.Ops[i].Choice = 2
				n.Ops[i].PC = invoke
				n.Ops[helper].Enabled = true
				add(n, i, "choose_helper", false)
			}
		}
		if o.PC == mode && o.FailureReady {
			n := s
			n.Ops[i].PC = terminal
			n.Ops[i].Freeze = failure
			add(n, i, "terminal_freeze", false)
		}
		if o.PC == mode && !o.FailureReady {
			if (i == entry && o.Input.Mode == "answer") || i == helper || i == sibling || i == wrapper {
				n := s
				n.Ops[i].Reply = 1
				n.Ops[i].PC = sent
				n.Ops[i].ReplyID = o.Input.ID
				n.Ops[i].ReplyValue = 1 - o.Input.Value
				if i == sibling {
					n.Ops[i].ReplyValue = 0
				}
				if i == wrapper {
					n.Ops[i].ReplyID = o.ReturnedID
					n.Ops[i].ReplyValue = o.ReturnedValue
				}
				add(n, i, "reply_commit", false)
			} else if i == entry && o.Input.Mode == "pending" {
				n := s
				n.Ops[i].PC = pending
				add(n, i, "pending_receive", false)
			} else {
				n := s
				n.Ops[i].PC = terminal
				n.Ops[i].Freeze = success
				if o.Input.Mode == "fail" {
					n.Ops[i].Freeze = failure
				}
				add(n, i, "terminal_freeze", false)
			}
		}
		if o.PC == sent {
			n := s
			n.Ops[i].Freeze = success
			n.Ops[i].PC = terminal
			add(n, i, "terminal_freeze", false)
		}
		if o.Reply == 1 {
			n := s
			n.Ops[i].Reply = 2
			if s.Closed {
				n.Ops[i].Reply = 3
				n.Ops[i].ReplyReason = "scope_closed"
			}
			add(n, i, "reply_admit", true)
		}
		if !o.Published && (o.Admission == 2 || (o.Freeze != none && o.Reply != 1)) {
			ready := true
			if o.Freeze == success {
				for j, child := range s.Ops {
					if j != i && child.Source == i && child.Admission == 1 && !child.Published {
						ready = false
					}
				}
			}
			if ready {
				n := s
				n.Ops[i].Published = true
				n.Ops[i].Status = "failed"
				n.Ops[i].Reason = "member_failed"
				if o.Admission == 2 {
					n.Ops[i].Status = "rejected"
					n.Ops[i].Reason = o.AdmissionReason
				} else if o.Freeze == success {
					n.Ops[i].Reason = "missing_reply"
					if o.Reply == 2 {
						n.Ops[i].Status = "succeeded"
						n.Ops[i].Reason = "member_succeeded"
						n.Ops[i].Present = true
						n.Ops[i].ID = o.ReplyID
						n.Ops[i].Value = o.ReplyValue
					}
				}
				add(n, i, "publish", true)
			}
		}
		if o.Published && !o.Delivered {
			n := s
			n.Ops[i].Delivered = true
			add(n, i, "outcome_deliver", true)
		}
		if o.Delivered && !o.Consumed && sourceOpen(s, o.Source) {
			n := s
			ready := false
			ok := o.Status == "succeeded" && o.Present
			if o.Source >= 0 {
				p := o.Source
				if s.Ops[p].PC == await {
					ready = true
					n.Ops[p].ReturnedID = o.ID
					n.Ops[p].ReturnedValue = o.Value
					n.Ops[p].ReturnedPresent = o.Present
					if ok {
						n.Ops[p].PC = mode
					} else {
						n.Ops[p].PC = mode
						n.Ops[p].FailureReady = true
					}
				}
			} else if o.Source == -1 && s.RootPC == await {
				ready = true
				n.RootID = o.ID
				n.RootValue = o.Value
				n.RootPresent = o.Present
				if !c.Local && c.Shape == "sequential" && i == entry && ok {
					n.RootPC = invoke
					n.Ops[sibling].Enabled = true
					n.Ops[sibling].Input = Request{o.ID, o.Value, "answer"}
				} else {
					n.RootPC = terminal
					n.RootStatus = "failed"
					if ok {
						n.RootStatus = "succeeded"
					}
					if i == entry && !ok && c.Shape == "sequential" {
						exclude(&n, sibling)
					}
				}
			} else if o.Source == -2 && s.PeerPC == await {
				ready = true
				n.PeerPC = terminal
				n.PeerStatus = "failed"
				if ok {
					n.PeerStatus = "succeeded"
				}
			}
			if ready {
				n.Ops[i].Consumed = true
				add(n, i, "outcome_consume", false)
			}
		}
	}
	return out
}

// Rank is intrinsic and strictly increases along all reference edges. It does
// not use history length. The explorer checks it on revisited successors too.
func Rank(s State) int {
	r := s.I + s.R + s.E + s.RootPC + s.PeerPC + s.PeerTicket
	if s.Closed {
		r += 100
	}
	for _, o := range s.Ops {
		if o.FailureReady {
			r++
		}
		r += o.Ticket + o.Admission + o.RequestStage + o.PC + o.Choice + o.Reply + o.Freeze
		if o.Enabled {
			r++
		}
		if o.Published {
			r++
		}
		if o.Delivered {
			r++
		}
		if o.Consumed {
			r++
		}
	}
	return r
}
func ConfigName(c Config) string {
	return fmt.Sprintf("%s/id%d/value%d/%s/I%d/R%d/D%d", c.Shape, c.Request.ID, c.Request.Value, c.Request.Mode, c.I, c.R, c.Depth)
}
