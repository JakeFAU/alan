// This package is a finite research transition system, deliberately test-only.
package semantics

import (
	"encoding/json"
	"fmt"
	"slices"
)

type opcode string

const (
	set     opcode = "Set"
	branch  opcode = "Branch"
	fork    opcode = "Fork"
	receive opcode = "Receive"
	emit    opcode = "Emit"
	send    opcode = "Send"
	done    opcode = "Done"
	success opcode = "Success"
	fault   opcode = "Fault"
)

type instruction struct {
	Mode           string `json:",omitempty"`
	Slot           int    `json:",omitempty"`
	Key            string `json:",omitempty"`
	FromReg        bool   `json:",omitempty"`
	Op             opcode
	Reg, Value, PC int
	Receipt        bool
	Ref            int
	Tag, Target    string
	TargetG        int // zero is a service target; positive names an exact run
}
type bounds struct{ Generations, Epochs, Commits, Continuations, Proposals, Intents, Attempts, Inbox, Value int }
type config struct {
	Composition                *compositionContract `json:",omitempty"`
	Programs                   map[string][]instruction
	Version                    string
	Routes                     map[string]string
	Edges                      map[string]bool
	Inputs                     map[string]message
	Bounds                     bounds
	RetryCancelled, Idempotent bool
	Mutant                     string
}
type authority struct {
	Owner      string
	W, Current int
	Retired    bool
}
type continuation struct{ ID, PC int }
type observation struct {
	Envelope                   string `json:",omitempty"`
	Sender                     string `json:",omitempty"`
	SenderG                    int    `json:",omitempty"`
	Origin                     string `json:",omitempty"`
	ID, Tag, Correlation, Kind string
	Value                      int
	Consumed                   bool
}
type projection struct {
	Saved    []envelope       `json:",omitempty"`
	Pending  []pendingRequest `json:",omitempty"`
	Accepted []string         `json:",omitempty"`
	Data     [2]int
	Refs     [2]string
	Frontier []continuation
	NextC    int
	Status   string
}
type outcome struct {
	Status string
	Value  int
	Route  string
}
type selection struct {
	C           int
	Observation string
}
type record struct {
	Kind      string
	Version   string
	E, N      int
	Selection selection
	Observed  *observation
	After     projection
	Intent    *descriptor
	Status    string
}
type run struct {
	A          string
	G, E, N    int
	Version    string
	Projection projection
	Inbox      map[string]observation
	History    []record
	Outcome    *outcome
}
type descriptor struct {
	Envelope string `json:",omitempty"`
	Kind     opcode
	A        string
	G, N     int
	Tag      string
	Value    int
	Target   string
	TargetG  int
}
type intent struct {
	Descriptor          descriptor
	Suppressed, Settled bool
}
type proposal struct {
	A         string
	G, E, N   int
	Version   string
	Selection selection
	Observed  *observation
	After     projection
	Intent    *descriptor
}
type message struct {
	Envelope       string `json:",omitempty"`
	SenderG        int    `json:",omitempty"`
	Source, Target string
	G              int
	Tag            string
	Value          int
}
type admission struct {
	Origin, RawID string
	Message       message
	Bound         string
	Accepted, Ack bool
}
type attempt struct {
	Intent  string
	Ordinal int
	Owner   string
	W       int
}
type physicalResult struct {
	Intent string
	Value  int
}
type resource struct {
	Mutations int
	Keys      map[string]int
	Applied   map[string]physicalResult
	Outbox    map[string]message
}
type receipt struct {
	Intent, Attempt string
	Value           int
}
type machine struct {
	Config     config
	Subjects   map[string]*authority
	Runs       map[string]*run
	Proposals  map[int]*proposal
	Intents    map[string]*intent
	Admissions map[string]*admission
	Attempts   map[string]attempt
	Receipts   map[string]receipt
	World      resource
	Delivered  map[string]outcome
}

type kind string

const (
	prepare       kind = "Prepare"
	interrupt     kind = "Interrupt"
	commit        kind = "Commit"
	admit         kind = "Admit"
	acknowledge   kind = "Acknowledge"
	authorize     kind = "AuthorizeAttempt"
	applyExternal kind = "ApplyExternal"
	recordReceipt kind = "RecordReceipt"
	reactivate    kind = "Reactivate"
	abandon       kind = "Abandon"
	cancel        kind = "CancelRun"
	retire        kind = "RetireSubject"
	retry         kind = "Retry"
	deliver       kind = "DeliverTerminal"
)

type event struct {
	Kind            kind
	A               string
	G, C, Slot      int
	Observation, ID string
	Origin          string
	Attempt         int
	Owner           string
	W               int
}

func (e event) String() string           { return canonical(e) }
func runID(a string, g int) string       { return fmt.Sprintf("%s/g%d", a, g) }
func intentID(a string, g, n int) string { return fmt.Sprintf("%s/n%d/i0", runID(a, g), n) }
func attemptID(id string, n int) string  { return fmt.Sprintf("%s/t%d", id, n) }
func canonical(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
func copyOf[T any](v T) T {
	var out T
	if err := json.Unmarshal([]byte(canonical(v)), &out); err != nil {
		panic(err)
	}
	return out
}
func freshRun(a string, g int, version string) *run {
	return &run{A: a, G: g, E: 1, Version: version, Projection: projection{Frontier: []continuation{{ID: 1}}, NextC: 2, Status: "active"}, Inbox: map[string]observation{}}
}
func newMachine(c config) machine {
	m := machine{Config: copyOf(c), Subjects: map[string]*authority{}, Runs: map[string]*run{}, Proposals: map[int]*proposal{}, Intents: map[string]*intent{}, Admissions: map[string]*admission{}, Attempts: map[string]attempt{}, Receipts: map[string]receipt{}, World: resource{Keys: map[string]int{}, Applied: map[string]physicalResult{}, Outbox: map[string]message{}}, Delivered: map[string]outcome{}}
	for a := range c.Programs {
		m.Subjects[a] = &authority{Owner: "owner", W: 7, Current: 1}
		m.Runs[runID(a, 1)] = freshRun(a, 1, c.Version)
	}
	return m
}
func fixture(code []instruction) machine {
	return newMachine(config{Programs: map[string][]instruction{"a": code}, Version: "v1", Routes: map[string]string{"a": "observer"}, Edges: map[string]bool{"env>a": true}, Inputs: map[string]message{}, Bounds: bounds{Generations: 2, Epochs: 2, Commits: 8, Continuations: 2, Proposals: 2, Intents: 2, Attempts: 2, Inbox: 4, Value: 8}, Idempotent: true})
}
func life(k kind, a string, g int) event { return event{Kind: k, A: a, G: g, Owner: "owner", W: 7} }
func (m machine) hasAttempt(id string) bool {
	for _, a := range m.Attempts {
		if a.Intent == id {
			return true
		}
	}
	return false
}
func (m machine) owned(e event, a string) bool {
	s := m.Subjects[a]
	return s != nil && e.Owner == s.Owner && e.W == s.W
}
func (m *machine) freeze(r *run) {
	if r.Outcome != nil {
		return
	}
	r.Outcome = &outcome{Status: r.Projection.Status, Value: r.Projection.Data[0], Route: m.Config.Routes[r.A]}
	for id, in := range m.Intents {
		if in.Descriptor.A == r.A && in.Descriptor.G == r.G && !m.hasAttempt(id) {
			in.Suppressed = true
		}
	}
}
func (m *machine) close(r *run, status string) {
	if r.Projection.Status != "active" {
		return
	}
	r.Projection.Status = status
	r.Projection.Frontier = nil
	r.History = append(r.History, record{Kind: "close", Status: status, E: r.E, N: r.N})
	m.freeze(r)
}

// step has exactly one serialization point. Rejections return the untouched input.
// In particular no lifecycle/authorization/reducer branch reads World.
func step(before machine, e event) (machine, string) {
	m := copyOf(before)
	reject := func(why string) (machine, string) { return before, why }
	r := m.Runs[runID(e.A, e.G)]
	s := m.Subjects[e.A]
	switch e.Kind {
	case prepare:
		if r == nil || s.Current != e.G || s.Retired || r.Projection.Status != "active" {
			return reject("closed")
		}
		if e.Slot < 0 || e.Slot >= m.Config.Bounds.Proposals {
			return reject("capacity:proposals")
		}
		if m.Proposals[e.Slot] != nil {
			return reject("occupied")
		}
		if r.N >= m.Config.Bounds.Commits {
			return reject("capacity:commits")
		}
		p, why := reduce(m.Config, r, selection{C: e.C, Observation: e.Observation})
		if why != "" {
			return reject(why)
		}
		if p.Intent != nil {
			count := 0
			for _, in := range m.Intents {
				if in.Descriptor.A == r.A && in.Descriptor.G == r.G {
					count++
				}
			}
			if count >= m.Config.Bounds.Intents {
				return reject("capacity:intents")
			}
		}
		m.Proposals[e.Slot] = p
	case interrupt:
		if m.Proposals[e.Slot] == nil {
			return reject("missing proposal")
		}
		delete(m.Proposals, e.Slot)
	case commit:
		p := m.Proposals[e.Slot]
		if p == nil {
			return reject("missing proposal")
		}
		r = m.Runs[runID(p.A, p.G)]
		s = m.Subjects[p.A]
		if s.Current != p.G {
			return reject("generation")
		}
		if s.Retired || r.Projection.Status != "active" {
			return reject("closed")
		}
		if r.E != p.E && m.Config.Mutant != "epoch" {
			return reject("epoch")
		}
		if r.N != p.N {
			return reject("sequence")
		}
		if r.Version != p.Version {
			return reject("version")
		}
		if p.Observed != nil {
			o, ok := r.Inbox[p.Observed.ID]
			if !ok || o.Consumed || canonical(o) != canonical(*p.Observed) {
				return reject("observation")
			}
			o.Consumed = true
			r.Inbox[o.ID] = o
		}
		r.Projection = copyOf(p.After)
		r.N++
		r.History = append(r.History, record{Kind: "commit", Version: p.Version, E: p.E, N: p.N, Selection: p.Selection, Observed: copyOf(p.Observed), After: copyOf(p.After), Intent: copyOf(p.Intent)})
		if p.Intent != nil {
			m.Intents[intentID(p.A, p.G, r.N)] = &intent{Descriptor: *p.Intent}
		}
		if r.Projection.Status != "active" {
			m.freeze(r)
		}
		delete(m.Proposals, e.Slot)
	case admit:
		var msg message
		var ok bool
		switch e.Origin {
		case "":
			msg, ok = m.Config.Inputs[e.ID]
		case "send":
			msg, ok = m.World.Outbox[e.ID]
		default:
			return reject("unknown transport origin")
		}
		if !ok {
			return reject("no transport message")
		}
		transport := transportID(e.Origin, e.ID)
		if !m.Config.Edges[msg.Source+">"+msg.Target] {
			return reject("scope")
		}
		old := m.Admissions[transport]
		if old != nil {
			if old.Message != msg {
				return reject("descriptor conflict")
			}
			if m.Config.Mutant != "binding" || msg.G != 0 || old.Bound == runID(msg.Target, m.Subjects[msg.Target].Current) {
				return m, ""
			}
		}
		target := m.Subjects[msg.Target]
		if target == nil {
			return reject("unknown target")
		}
		g := msg.G
		if g == 0 {
			g = target.Current
		}
		bound := runID(msg.Target, g)
		targetRun := m.Runs[bound]
		accepted := targetRun != nil && !target.Retired && targetRun.Projection.Status == "active" && target.Current == g
		if accepted && len(targetRun.Inbox) >= m.Config.Bounds.Inbox {
			return reject("capacity:inbox")
		}
		m.Admissions[transport] = &admission{Origin: e.Origin, RawID: e.ID, Message: msg, Bound: bound, Accepted: accepted}
		if accepted {
			targetRun.Inbox[admissionObservationID(transport)] = admittedObservation(transport, e.Origin, msg)
		}
	case acknowledge:
		a := m.Admissions[transportID(e.Origin, e.ID)]
		if a == nil {
			return reject("no admission")
		}
		a.Ack = true
	case authorize:
		in := m.Intents[e.ID]
		if in == nil {
			return reject("uncommitted intent")
		}
		r = m.Runs[runID(in.Descriptor.A, in.Descriptor.G)]
		s = m.Subjects[in.Descriptor.A]
		if !m.owned(e, in.Descriptor.A) {
			return reject("ownership")
		}
		// Removing the closure decision also bypasses its suppression consequence;
		// retaining either would conceal this deliberately faulty authorization.
		if m.Config.Mutant != "closure" && (s.Retired || s.Current != r.G || r.Projection.Status != "active" || in.Suppressed) {
			return reject("closed")
		}
		if in.Settled {
			return reject("settled")
		}
		if e.Attempt < 1 || e.Attempt > m.Config.Bounds.Attempts {
			return reject("capacity:attempts")
		}
		id := attemptID(e.ID, e.Attempt)
		if _, ok := m.Attempts[id]; ok {
			return reject("duplicate attempt")
		}
		m.Attempts[id] = attempt{Intent: e.ID, Ordinal: e.Attempt, Owner: e.Owner, W: e.W}
	case applyExternal:
		id := attemptID(e.ID, e.Attempt)
		a, ok := m.Attempts[id]
		if !ok {
			return reject("unauthorized attempt")
		}
		if _, ok := m.World.Applied[id]; ok {
			return reject("already applied")
		}
		d := m.Intents[a.Intent].Descriptor
		value := d.Value
		if d.Kind == send {
			m.World.Outbox[e.ID] = sentMessage(d)
		} else if previous, exists := m.World.Keys[e.ID]; m.Config.Idempotent && exists {
			value = previous
		} else {
			m.World.Mutations++
			m.World.Keys[e.ID] = value
		}
		m.World.Applied[id] = physicalResult{Intent: e.ID, Value: value}
	case recordReceipt:
		id := attemptID(e.ID, e.Attempt)
		a, ok := m.Attempts[id]
		if !ok || a.Intent != e.ID {
			return reject("receipt correlation")
		}
		result, ok := m.World.Applied[id]
		if !ok {
			return reject("no result")
		}
		in := m.Intents[e.ID]
		value := result.Value
		if in.Descriptor.Kind == send {
			ad := m.Admissions[transportID("send", e.ID)]
			if ad == nil {
				return reject("no admission result")
			}
			if ad.Origin != "send" || ad.RawID != e.ID || ad.Message != sentMessage(in.Descriptor) {
				return reject("admission correlation")
			}
			value = 0
			if ad.Accepted {
				value = 1
			}
		}
		if _, ok := m.Receipts[id]; ok {
			return m, ""
		}
		r = m.Runs[runID(in.Descriptor.A, in.Descriptor.G)]
		first := !in.Settled
		if first && r.Projection.Status == "active" && len(r.Inbox) >= m.Config.Bounds.Inbox {
			return reject("capacity:inbox")
		}
		m.Receipts[id] = receipt{Intent: e.ID, Attempt: id, Value: value}
		if in.Descriptor.Kind == send {
			m.Admissions[transportID("send", e.ID)].Ack = true
		}
		in.Settled = true
		if first && r.Projection.Status == "active" {
			r.Inbox["receipt:"+e.ID] = observation{ID: "receipt:" + e.ID, Tag: "receipt:" + in.Descriptor.Tag, Correlation: e.ID, Value: value, Kind: "receipt"}
		}
	case reactivate, abandon, cancel, retire, retry:
		if !m.owned(e, e.A) {
			return reject("ownership")
		}
		if r == nil {
			return reject("unknown run")
		}
		if e.Kind != cancel && e.Kind != retire && s.Current != e.G {
			return reject("generation")
		}
		switch e.Kind {
		case reactivate:
			if s.Retired || r.Projection.Status != "active" {
				return reject("closed")
			}
			if r.E >= m.Config.Bounds.Epochs {
				return reject("capacity:epochs")
			}
			r.E++
			r.History = append(r.History, record{Kind: "reactivate", E: r.E, N: r.N})
		case abandon:
			if s.Retired || r.Projection.Status != "active" {
				return reject("closed")
			}
			m.close(r, "abandoned")
		case cancel:
			m.close(r, "cancelled")
		case retire:
			s.Retired = true
			m.close(m.Runs[runID(e.A, s.Current)], "cancelled")
		case retry:
			status := r.Projection.Status
			if s.Retired || !(status == "failed" || status == "abandoned" || status == "cancelled" && m.Config.RetryCancelled) {
				return reject("retry policy")
			}
			if s.Current >= m.Config.Bounds.Generations {
				return reject("capacity:generations")
			}
			s.Current++
			m.Runs[runID(e.A, s.Current)] = freshRun(e.A, s.Current, r.Version)
		}
	case deliver:
		if r == nil || r.Outcome == nil {
			return reject("no frozen notification")
		}
		m.Delivered[runID(e.A, e.G)] = *r.Outcome
	default:
		panic("unknown event: " + e.Kind)
	}
	return m, ""
}

// reduce evaluates exactly one closed instruction and cannot inspect the machine,
// authority, transport, attempts, or external world.
func reduce(c config, r *run, selected selection) (*proposal, string) {
	p := &proposal{A: r.A, G: r.G, E: r.E, N: r.N, Version: r.Version, Selection: selected, After: copyOf(r.Projection)}
	index := slices.IndexFunc(p.After.Frontier, func(v continuation) bool { return v.ID == selected.C })
	if index < 0 {
		return nil, "no continuation"
	}
	pc := p.After.Frontier[index].PC
	code := c.Programs[r.A]
	if pc < 0 || pc >= len(code) {
		panic("interpreter PC outside immutable program")
	}
	in := code[pc]
	if in.Ref < 0 || in.Ref >= len(p.After.Refs) {
		panic("invalid effect reference slot")
	}
	if in.Reg < 0 || in.Reg >= len(p.After.Data) {
		panic("invalid register")
	}
	p.After.Frontier[index].PC++
	switch in.Op {
	case set:
		p.After.Data[in.Reg] = in.Value
	case branch:
		if p.After.Data[in.Reg] == in.Value {
			p.After.Frontier[index].PC = in.PC
		}
	case fork:
		if len(p.After.Frontier) >= c.Bounds.Continuations {
			return nil, "capacity:continuations"
		}
		p.After.Frontier = append(p.After.Frontier, continuation{ID: p.After.NextC, PC: in.PC})
		p.After.NextC++
	case receive:
		o, ok := r.Inbox[selected.Observation]
		if !ok || o.Consumed || o.Tag != in.Tag {
			return nil, "waiting"
		}
		if in.Receipt {
			if o.Kind != "receipt" || r.Projection.Refs[in.Ref] == "" || o.Correlation != r.Projection.Refs[in.Ref] {
				return nil, "correlation"
			}
		} else if o.Kind != "message" {
			return nil, "correlation"
		}
		if why := compositionReceive(c, r, p, in, o); why != "" {
			return nil, why
		}
		p.Observed = &o
		p.After.Data[in.Reg] += o.Value
	case emit, send:
		target := in.Target
		if in.Mode == "reply" {
			if in.Slot < 0 || in.Slot >= len(r.Projection.Saved) {
				return nil, "missing upstream"
			}
			target = r.Projection.Saved[in.Slot].ReplyTo
		}
		if in.Op == send && !c.Edges[r.A+">"+target] {
			return nil, "scope"
		}
		p.After.Refs[in.Ref] = intentID(r.A, r.G, r.N+1)
		p.Intent = &descriptor{Kind: in.Op, A: r.A, G: r.G, N: r.N + 1, Tag: in.Tag, Value: in.Value, Target: target, TargetG: in.TargetG}
		if why := compositionSend(r, p, in); why != "" {
			return nil, why
		}
	case done:
		p.After.Frontier = slices.Delete(p.After.Frontier, index, index+1)
	case success, fault:
		p.After.Status = "succeeded"
		if in.Op == fault {
			p.After.Status = "failed"
		}
		p.After.Frontier = nil
	default:
		panic("unknown instruction: " + in.Op)
	}
	if in.Op != receive && selected.Observation != "" {
		return nil, "unexpected observation"
	}
	for _, value := range p.After.Data {
		if value < 0 || value > c.Bounds.Value {
			return nil, "capacity:data"
		}
	}
	for _, continuation := range p.After.Frontier {
		if continuation.PC < 0 || continuation.PC >= len(code) {
			panic("instruction produced invalid PC")
		}
	}
	return p, ""
}

// Ingress IDs are opaque, unrestricted strings. Prefix each layer's identity
// injectively rather than reserving user-visible spellings.
func transportID(origin, raw string) string {
	if origin == "send" {
		return "send:" + raw
	}
	return "ingress:" + raw
}
func admissionObservationID(id string) string { return "message:" + id }
func messageObservationID(raw string) string  { return admissionObservationID(transportID("", raw)) }
func sendObservationID(id string) string      { return admissionObservationID(transportID("send", id)) }
func sentMessage(d descriptor) message {
	msg := message{Source: d.A, Target: d.Target, G: d.TargetG, Tag: d.Tag, Value: d.Value, Envelope: d.Envelope}
	if d.Envelope != "" {
		msg.SenderG = d.G
	}
	return msg
}
