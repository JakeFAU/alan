package substitution

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

const auditBundleSHA = "8693f675a4ec4be6d14a859157fbbf9d21b3608e4485d32b900421c0d7d7b5aa"

var auditPrograms = [2]string{
	"ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67",
	"698256075f67b360cacb0b20e16c5aeafc561c76255aeb3fd1ca7481ef9e9a6b",
}

const (
	auditEntryKey       = `["root",1,1,0]`
	auditHelperKey      = `["e",1,2,0]`
	auditHelperTemplate = "1806cf3a9e1e67ec0ed25a665b7a8a9be2dd7cf37585c5b1cd3e274b12595e3d"
)

type auditMachine struct{ programs [2]executor.Program }

func auditLoad(raw []byte) (auditMachine, error) {
	m := auditMachine{}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != auditBundleSHA {
		return m, fmt.Errorf("stale B bundle (program, descriptor or initializer)")
	}
	var bundle struct {
		Programs []struct {
			ID    string
			Bytes []byte
		}
	}
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return m, err
	}
	for i, id := range auditPrograms {
		for _, artifact := range bundle.Programs {
			if artifact.ID != id {
				continue
			}
			if fmt.Sprintf("%x", sha256.Sum256(artifact.Bytes)) != id {
				return m, fmt.Errorf("program %d identity", i)
			}
			p := &m.programs[i]
			if err := json.Unmarshal(artifact.Bytes, p); err != nil {
				return m, err
			}
			// Fixed representative of the original port binding, following
			// resolveCompositionProgram. Invoke/Await descriptors stay literal.
			caller, operation := "root", auditEntryKey
			if i == helper {
				caller, operation = "e", auditHelperKey
			}
			port := p.Author.Ports["public"]
			port.Peer = caller
			p.Author.Ports["public"] = port
			for j, in := range p.Code {
				if in.Op == "receive" && in.Match.Kind == "message" {
					p.Code[j].Match.Source = executor.Lit(schema.Text(caller))
					p.Code[j].Match.Operation = executor.Lit(schema.Text(operation))
					p.Code[j].Match.Provenance = "runtime-invocation-request"
				}
				if in.Op == "emit" && in.Effect.Kind == "message" {
					p.Code[j].Effect.Target = executor.Lit(schema.Text(caller))
				}
			}
		}
		if len(m.programs[i].Code) == 0 {
			return m, fmt.Errorf("missing %s", id)
		}
	}
	return m, nil
}

func auditBundle(t *testing.T) ([]byte, auditMachine) {
	t.Helper()
	raw, err := os.ReadFile("../../docs/foundational/phase1k/evidence/context/frozen/b-candidate-template.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := auditLoad(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, m
}

// Decode the fixed instance's register representation into the actual map domain.
// Uncreated members have no projection. Callers inspect this only after admission.
func auditProjection(s State, i int) executor.Projection {
	pc, r, o := s.PrivatePC[i], s.PrivateRegisters[i], s.Ops[i]
	p := executor.Projection{PC: pc, Status: "open", Data: map[string]schema.Value{}}
	if o.Freeze != none {
		p.Status = map[int]string{success: "succeeded", failure: "failed"}[o.Freeze]
	}
	if o.RequestStage != 2 {
		return p
	}
	if i == helper {
		p.Data["input_id"] = schema.Int(int64(r[0]))
		p.Data["input_value"] = schema.Int(int64(r[1]))
		p.Data["input_mode"] = schema.Text(o.Input.Mode)
		if pc == 3 || pc == 5 || pc == 6 {
			p.Data["complement_value"] = schema.Int(int64(r[2]))
		}
		return p
	}
	p.Data["requested_id"] = schema.Int(int64(r[0]))
	p.Data["requested_value"] = schema.Int(int64(r[1]))
	p.Data["requested_mode"] = schema.Text(o.Input.Mode)
	h := s.Ops[helper]
	if pc >= 2 {
		p.Data["helper_operation"] = schema.Text(auditHelperKey)
	}
	if h.Consumed {
		p.Data["helper_status"] = schema.Text(h.Status)
		p.Data["helper_reason"] = schema.Text(h.Reason)
		p.Data["helper_present"] = schema.Bool(h.Present)
		if h.Present {
			p.Data["helper_id"] = schema.Int(int64(r[5]))
			p.Data["helper_value"] = schema.Int(int64(r[6]))
		}
	}
	return p
}

type auditAction struct{ kind, op string }

func auditActionOf(e Edge) auditAction { return auditAction{e.Event.Kind, e.Event.Operation} }

// Enumerate enabled kernel actions from the written fixed local guards, before
// looking at submitted edges. The immutable K1 source supplies the updates of
// these declared actions. The independent guard list can detect a missing rule
// even if every surviving edge is legal.
func auditKernel(c Config, s State) map[auditAction]bool {
	out := map[auditAction]bool{}
	if c.Closure && !s.Closed {
		out[auditAction{"close_root", ""}] = true
	}
	if c.Peer && !s.Closed && s.PeerTicket == possible && s.I < c.I && s.R < c.R {
		out[auditAction{"peer_debit", "q"}] = true
	}
	e := s.Ops[entry]
	if e.Ticket == possible && !s.Closed {
		out[auditAction{"invoke_commit", "e"}] = false
	}
	if e.Delivered && !e.Consumed && !s.Closed && s.RootPC == await {
		out[auditAction{"outcome_consume", "e"}] = false
	}
	for i, role := range []string{"e", "h"} {
		o := s.Ops[i]
		if o.Ticket == committed && o.Admission == 0 {
			out[auditAction{"admit", role}] = true
		}
		if o.Reply == 1 {
			out[auditAction{"reply_admit", role}] = true
		}
		childSettled := i == helper || s.Ops[helper].Admission != 1 || s.Ops[helper].Published
		if !o.Published && (o.Admission == 2 || (o.Freeze != none && o.Reply != 1 && (o.Freeze != success || childSettled))) {
			out[auditAction{"publish", role}] = true
		}
		if o.Published && !o.Delivered {
			out[auditAction{"outcome_deliver", role}] = true
		}
	}
	return out
}

func auditReference(c Config, s State) map[auditAction]bool {
	out := auditKernel(c, s)
	e, h := s.Ops[entry], s.Ops[helper]
	if h.Enabled && h.Ticket == possible && e.Freeze == none {
		out[auditAction{"invoke_commit", "h"}] = false
	}
	if h.Delivered && !h.Consumed && e.PC == await && e.Freeze == none {
		out[auditAction{"outcome_consume", "h"}] = false
	}
	if e.PC == choose {
		out[auditAction{"choose_local", "e"}] = false
		out[auditAction{"choose_helper", "e"}] = false
	}
	for i, role := range []string{"e", "h"} {
		o := s.Ops[i]
		switch o.PC {
		case receive:
			if o.Admission == 1 && o.RequestStage == 1 {
				out[auditAction{"request_consume", role}] = false
			}
		case mode:
			switch {
			case o.FailureReady:
				out[auditAction{"terminal_freeze", role}] = false
			case i == helper || o.Input.Mode == "answer":
				out[auditAction{"reply_commit", role}] = false
			case o.Input.Mode == "pending":
				out[auditAction{"pending_receive", role}] = false
			default:
				out[auditAction{"terminal_freeze", role}] = false
			}
		case sent:
			out[auditAction{"terminal_freeze", role}] = false
		}
	}
	return out
}

func auditInput(s State, i int) *executor.Observation {
	run := executor.RunID{Subject: name(i), Generation: 1}
	o := s.Ops[i]
	pc := s.PrivatePC[i]
	if pc == 0 && o.RequestStage == 1 {
		caller, key := "root", auditEntryKey
		if i == helper {
			caller, key = "e", auditHelperKey
		}
		r := o.Input
		return &executor.Observation{ID: "request/" + name(i), Run: run, Kind: "message", Provenance: "runtime-invocation-request", Source: caller, SourceGeneration: 1, Obligation: "substitution", Operation: key, Payload: map[string]schema.Value{"id": schema.Int(int64(r.ID)), "value": schema.Int(int64(r.Value)), "mode": schema.Text(r.Mode)}}
	}
	h := s.Ops[helper]
	if i == entry && pc == 2 && h.Delivered && !h.Consumed {
		data := map[string]schema.Value{"status": schema.Text(h.Status), "reason": schema.Text(h.Reason), "response_present": schema.Bool(h.Present)}
		if h.Present {
			data["response.id"] = schema.Int(int64(h.ID))
			data["response.value"] = schema.Int(int64(h.Value))
		}
		return &executor.Observation{ID: "outcome/h", Run: run, Kind: "invocation", Provenance: "runtime-invocation", Source: auditHelperTemplate, SourceGeneration: 1, Obligation: "complement", Operation: auditHelperKey, Payload: data}
	}
	return nil
}

// auditReduction attempts the actual next instruction. A missing operand is an
// error, including when all supplied edges omit that instruction. Only a Receive
// with no eligible input can have a nonterminal, nonproductive attempt here.
func (m auditMachine) auditReduction(s State, i int) (executor.Reduction, auditAction, error) {
	p := m.programs[i]
	pre := auditProjection(s, i)
	seq := int64(s.PrivateRegisters[i][7] + 1)
	ob := auditInput(s, i)
	red, err := executor.Reduce(p, pre, ob, executor.RunID{Subject: name(i), Generation: 1}, seq, executor.DefaultLimits())
	if errors.Is(err, executor.ErrIneligible) && ob == nil && p.Code[pre.PC].Op == "receive" {
		return red, auditAction{}, nil
	}
	if err != nil {
		return red, auditAction{}, fmt.Errorf("%s PC%d reduction: %w", name(i), pre.PC, err)
	}
	a := auditAction{"private_step", name(i)}
	switch p.Code[pre.PC].Op {
	case "receive":
		a.kind = "request_consume"
		if ob.Kind == "invocation" {
			a = auditAction{"outcome_consume", "h"}
		}
	case "emit":
		a.kind = "reply_commit"
		if red.Intents[0].Kind == "invocation" {
			a = auditAction{"invoke_commit", "h"}
		}
	case "end":
		a.kind = "terminal_freeze"
	case "branch":
		if i == entry && red.Projection.PC == 12 {
			a.kind = "pending_receive"
		}
	}
	return red, a, nil
}

func auditPrivateTarget(s State, i int, e Edge, red executor.Reduction) error {
	n := e.Next
	if n.PrivateRegisters[i][7] != s.PrivateRegisters[i][7]+1 || !reflect.DeepEqual(auditProjection(n, i), red.Projection) {
		return fmt.Errorf("reducer target %s PC%d: got %+v seq%d want %+v seq%d", name(i), s.PrivatePC[i], auditProjection(n, i), n.PrivateRegisters[i][7], red.Projection, s.PrivateRegisters[i][7]+1)
	}
	// Also reject extra writes to slots absent from the reachable map domain.
	data := red.Projection.Data
	var want [8]int
	want[7] = s.PrivateRegisters[i][7] + 1
	if i == helper {
		want[0], want[1], want[2] = int(data["input_id"].Number), int(data["input_value"].Number), int(data["complement_value"].Number)
	} else {
		want[0], want[1] = int(data["requested_id"].Number), int(data["requested_value"].Number)
		if data["helper_status"].Text == "succeeded" {
			want[3] = 1
		}
		if data["helper_present"].Flag {
			want[4] = 1
		}
		want[5], want[6] = int(data["helper_id"].Number), int(data["helper_value"].Number)
	}
	if n.PrivateRegisters[i] != want {
		return fmt.Errorf("private register encoding %s PC%d: %v, want %v", name(i), s.PrivatePC[i], n.PrivateRegisters[i], want)
	}
	h := n.Ops[helper]
	if len(red.Intents) == 0 {
		return nil
	}
	d := red.Intents[0]
	switch d.Kind {
	case "invocation":
		if d.ID != auditHelperKey || d.Target != auditHelperTemplate || d.Generation != 1 || d.Obligation != "complement" ||
			!maps.Equal(d.Payload, map[string]schema.Value{"id": schema.Int(int64(h.Input.ID)), "value": schema.Int(int64(h.Input.Value)), "mode": schema.Text(h.Input.Mode)}) {
			return fmt.Errorf("unmatched helper occurrence/payload: %+v", d)
		}
	case "message":
		caller := "root"
		if i == helper {
			caller = "e"
		}
		if d.Target != caller || d.Generation != 1 || d.Obligation != "substitution" || !maps.Equal(d.Payload, map[string]schema.Value{"id": schema.Int(int64(e.Event.ID)), "value": schema.Int(int64(e.Event.Value))}) {
			return fmt.Errorf("unmatched reply intent %+v", d)
		}
	default:
		return fmt.Errorf("unexpected effect %q", d.Kind)
	}
	return nil
}

func auditOutgoing(c Config, s State, edges []Edge, m *auditMachine) error {
	expected := auditReference(c, s)
	type attempt struct {
		i   int
		red executor.Reduction
	}
	attempts := map[auditAction]attempt{}
	if m != nil {
		expected = auditKernel(c, s)
		for i := entry; i <= helper; i++ {
			if s.Ops[i].Admission != 1 || s.Ops[i].Freeze != none {
				continue
			}
			red, action, err := m.auditReduction(s, i)
			if err != nil {
				return err
			}
			if action.kind != "" {
				expected[action] = false
				attempts[action] = attempt{i, red}
			}
		}
	}
	// For declared actions the fixed rule source is the authority on labels
	// and next protocol state. Independently derive eligibility above. Do not
	// project private PCs away for the actual reducer checks.
	kernel := map[auditAction]Edge{}
	for _, ke := range Successors(c, Reference, s) {
		kernel[auditActionOf(ke)] = ke
	}
	for _, e := range edges {
		a := auditActionOf(e)
		env, enabled := expected[a]
		if !enabled {
			return fmt.Errorf("unexpected or duplicate action %+v", a)
		}
		if e.Environment != env {
			return fmt.Errorf("owner %+v", a)
		}
		delete(expected, a)
		r, private := attempts[a]
		if !private {
			if ke, ok := kernel[a]; !ok || e != ke {
				return fmt.Errorf("kernel edge %+v", a)
			}
			continue
		}
		if err := auditPrivateTarget(s, r.i, e, r.red); err != nil {
			return err
		}
		ke, ok := kernel[a]
		if a.kind == "private_step" {
			ke = Edge{Event: Observation{Kind: "private_step", Operation: a.op, Role: s.Ops[r.i].Role}, Next: s}
			ok = true
		}
		if !ok {
			return fmt.Errorf("no declared boundary fact %+v", a)
		}
		if a == (auditAction{"request_consume", "e"}) {
			// Inside B0's target: unique next instruction is Invoke(h); no
			// new occurrence or cut. These three fields are bookkeeping.
			ke.Next.Ops[entry].PC = invoke
			ke.Next.Ops[entry].Choice = 2
			ke.Next.Ops[helper].Enabled = true
		}
		// The actor's own private data and sequence were checked above.
		ke.Next.PrivatePC[r.i] = e.Next.PrivatePC[r.i]
		ke.Next.PrivateRegisters[r.i] = e.Next.PrivateRegisters[r.i]
		if e != ke {
			return fmt.Errorf("private edge public effect %+v actual=%+v expected=%+v", a, e, ke)
		}
	}
	if len(expected) > 0 {
		return fmt.Errorf("missing successors %v", expected)
	}
	return nil
}

func TestRetainedBAuditOutgoing(t *testing.T) {
	_, m := auditBundle(t)
	for _, input := range CallableJoinConfigurations() {
		c := CallableJoinLocalConfig(input)
		for _, closure := range []bool{true, false} {
			c.Closure = closure
			for _, candidate := range []bool{true, false} {
				first := Initial(c)
				queue := []State{first}
				seen := map[State]bool{first: true}
				count := 0
				for pos := 0; pos < len(queue); pos++ {
					s := queue[pos]
					var machine *auditMachine
					var edges []Edge
					if candidate {
						machine = &m
						edges = retainedBSuccessors(c, s)
					} else {
						edges = Successors(c, Reference, s)
					}
					if err := auditOutgoing(c, s, edges, machine); err != nil {
						t.Fatalf("%s closure=%v B=%v state=%+v: %v", ConfigName(c), closure, candidate, s, err)
					}
					count += len(edges)
					for _, e := range edges {
						if !seen[e.Next] {
							seen[e.Next] = true
							queue = append(queue, e.Next)
						}
					}
				}
				t.Logf("%s closure=%v B=%v states=%d edges=%d", ConfigName(c), closure, candidate, len(queue), count)
			}
		}
	}
}

func TestRetainedBAuditMissingSuccessors(t *testing.T) {
	_, m := auditBundle(t)
	c := CallableJoinLocalConfig(CallableJoinConfigurations()[0])
	for _, test := range []struct {
		name      string
		candidate bool
		remove    auditAction
	}{
		{"candidate q", true, auditAction{"peer_debit", "q"}},
		{"candidate closure", true, auditAction{"close_root", ""}},
		{"reference local branch", false, auditAction{"choose_local", "e"}},
		{"reference helper branch", false, auditAction{"choose_helper", "e"}},
		{"reference service", false, auditAction{"admit", "h"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			queue := []State{Initial(c)}
			seen := map[State]bool{queue[0]: true}
			for pos := 0; pos < len(queue); pos++ {
				s := queue[pos]
				var mptr *auditMachine
				var edges []Edge
				if test.candidate {
					mptr = &m
					edges = retainedBSuccessors(c, s)
				} else {
					edges = Successors(c, Reference, s)
				}
				for j, e := range edges {
					if auditActionOf(e) == test.remove && (!test.candidate || s.PrivatePC[entry] == 1) {
						mutant := append(append([]Edge{}, edges[:j]...), edges[j+1:]...)
						err := auditOutgoing(c, s, mutant, mptr)
						if err == nil || !strings.Contains(err.Error(), "missing successors") {
							t.Fatalf("deletion returned %v", err)
						}
						return
					}
					if !seen[e.Next] {
						seen[e.Next] = true
						queue = append(queue, e.Next)
					}
				}
			}
			t.Fatal("target not reachable")
		})
	}
}

// Audit the graph actually passed to O. A misspelled owner, omitted edge or
// wrong successor in that table need not alter the separately built Go graph.
// Recover a state-to-node bijection by following edges from the supplied root.
func auditTable(c Config, g weakFiniteGraph, rules map[string]weakRule, m *auditMachine, fact bool) (map[State]string, error) {
	idx, issues := indexWeakGraph(g)
	if len(issues) > 0 || len(g.Initial) != 1 {
		return nil, fmt.Errorf("graph structure: %v", issues)
	}
	start := Initial(c)
	ids := map[State]string{start: g.Initial[0]}
	inverse := map[string]State{g.Initial[0]: start}
	queue := []State{start}
	for j := 0; j < len(queue); j++ {
		s := queue[j]
		from := ids[s]
		var edges []Edge
		if m != nil {
			edges = retainedBSuccessors(c, s)
		} else {
			edges = Successors(c, Reference, s)
		}
		if err := auditOutgoing(c, s, edges, m); err != nil {
			return nil, err
		}
		expected := map[string]Edge{}
		for _, e := range edges {
			expected[from+"/"+e.Event.Operation+"/"+e.Event.Kind] = e
		}
		for _, we := range idx.out[from] {
			e, ok := expected[we.ID]
			if !ok {
				return nil, fmt.Errorf("unexpected supplied edge %s", we.ID)
			}
			delete(expected, we.ID)
			r := rules[we.Rule]
			owner := "component"
			if e.Environment {
				owner = "environment"
			}
			word := []Observation(nil)
			label := e.Event
			if label.Kind == "terminal_freeze" {
				actor := entry
				if label.Operation == "h" {
					actor = helper
				}
				label.Status = map[int]string{success: "succeeded", failure: "failed"}[e.Next.Ops[actor].Freeze]
			}
			if label.Kind != "private_step" && !(fact && (label.Kind == "choose_helper" || label.Kind == "choose_local")) {
				word = []Observation{label}
			}
			if r.Owner != owner {
				return nil, fmt.Errorf("wrong supplied owner %s", we.ID)
			}
			if r.Semantic != e.Event.Kind || r.Private != (e.Event.Kind == "private_step") || !reflect.DeepEqual(word, r.Word) {
				return nil, fmt.Errorf("wrong supplied word %s", we.ID)
			}
			if id, ok := ids[e.Next]; ok && id != we.To {
				return nil, fmt.Errorf("split successor at %s", we.ID)
			}
			if n, ok := inverse[we.To]; ok && n != e.Next {
				return nil, fmt.Errorf("merged successor at %s", we.ID)
			}
			if _, ok := ids[e.Next]; !ok {
				ids[e.Next] = we.To
				inverse[we.To] = e.Next
				queue = append(queue, e.Next)
			}
		}
		for id := range expected {
			return nil, fmt.Errorf("missing supplied edge %s", id)
		}
	}
	if len(ids) != len(g.Nodes) {
		return nil, fmt.Errorf("unreachable supplied states")
	}
	return ids, nil
}

// One unit per actual productive local action. In the open funded profile e
// commits exactly once, both decisions accept, h commitment and h selection
// are private instructions, and only the original caller owns e selection.
func auditWork(s State) int {
	n := s.PrivateRegisters[entry][7] + s.PrivateRegisters[helper][7]
	if s.Ops[entry].Ticket != possible {
		n++
	}
	if s.Ops[entry].Consumed {
		n++
	}
	if s.PeerTicket == spent {
		n++
	}
	for _, o := range s.Ops[:2] {
		for _, done := range []bool{o.Admission == 1, o.Reply == 2, o.Published, o.Delivered} {
			if done {
				n++
			}
		}
	}
	return n
}

// Recognize the actual boundary independently of the graph sink marker.
func auditMaximum(c Config, s State) bool {
	e, h := s.Ops[entry], s.Ops[helper]
	if s.Closed || s.PeerTicket != spent || s.I != 3 || s.R != 3 || s.E != 0 || e.Input != c.Request ||
		e.Depth != 2 || h.Depth != 3 || e.Choice != 2 || e.Ticket != spent || h.Ticket != spent ||
		e.Admission != 1 || h.Admission != 1 || e.RequestStage != 2 || h.RequestStage != 2 ||
		h.Input != (Request{c.Request.ID, c.Request.Value, "answer"}) || h.Freeze != success ||
		h.Reply != 2 || !h.Published || !h.Delivered || !h.Consumed || !h.Present || h.ID != c.Request.ID ||
		h.Value != 1-c.Request.Value || h.Status != "succeeded" || h.Reason != "member_succeeded" {
		return false
	}
	if c.Request.Mode == "pending" {
		return e.PC == pending && s.PrivatePC[entry] == 12 && e.Freeze == none && e.Reply == 0 && !e.Published && !e.Delivered && !e.Consumed &&
			!e.Present && e.Status == "" && e.Reason == "" && s.RootPC == await && s.RootStatus == ""
	}
	if e.PC != terminal || !e.Published || !e.Delivered || !e.Consumed || s.RootPC != terminal ||
		e.Status != s.RootStatus || e.Present != s.RootPresent || e.ID != s.RootID || e.Value != s.RootValue {
		return false
	}
	switch c.Request.Mode {
	case "answer":
		return s.PrivatePC[entry] == 7 && e.Freeze == success && e.Reply == 2 && e.Present && e.ID == c.Request.ID && e.Value == 1-c.Request.Value && e.Status == "succeeded" && e.Reason == "member_succeeded"
	case "fail":
		return s.PrivatePC[entry] == 9 && e.Freeze == failure && e.Reply == 0 && !e.Present && e.Status == "failed" && e.Reason == "member_failed"
	case "absent":
		return s.PrivatePC[entry] == 11 && e.Freeze == success && e.Reply == 0 && !e.Present && e.Status == "failed" && e.Reason == "missing_reply"
	}
	return false
}

func TestRetainedBAuditTables(t *testing.T) {
	_, m := auditBundle(t)
	for _, c := range retainedBConfigs() {
		for _, fact := range []bool{false, true} {
			p, cert := retainedBWitness(t, c, fact)
			x, err := auditTable(c, p.X, p.Rules, &m, fact)
			if err != nil {
				t.Fatal(err)
			}
			k, err := auditTable(c, p.K, p.Rules, nil, fact)
			if err != nil {
				t.Fatal(err)
			}
			// Cert carries exactly the promised pairs. Check public-boundary
			// equality also at helper-Invoke readiness before reference choice.
			seen := map[weakPair]bool{}
			for _, pair := range cert.Pairs {
				seen[pair] = true
			}
			if len(seen) != len(x) {
				t.Fatalf("unexpected relation size")
			}
			// P_boundary permits the precommit helper-enabled K control to
			// observe as prechoice. All other differences here are retained.
			values := map[string]State{}
			names := map[State]string{}
			for ref, id := range k {
				pid := p.K.Nodes[id].Projection
				if ref.Ops[entry].Choice == 2 && ref.Ops[helper].Ticket == possible {
					ref.Ops[entry].PC = choose
					ref.Ops[entry].Choice = 0
					ref.Ops[helper].Enabled = false
				}
				if val, ok := values[pid]; ok && val != ref {
					t.Fatal("unequal public facts share a projection")
				}
				if name, ok := names[ref]; ok && name != pid {
					t.Fatal("same public facts split across projections")
				}
				values[pid] = ref
				names[ref] = pid
			}
			for s, xid := range x {
				ref := s
				ref.PrivatePC = [slots]int{}
				ref.PrivateRegisters = [slots][8]int{}
				if s.Ops[entry].RequestStage == 2 && s.PrivatePC[entry] == 1 {
					ref.Ops[entry].PC = choose
					ref.Ops[entry].Choice = 0
					ref.Ops[helper].Enabled = false
				}
				kid, ok := k[ref]
				if !ok || !seen[weakPair{xid, kid}] || p.X.Nodes[xid].Projection != p.K.Nodes[kid].Projection {
					t.Fatalf("missing pair from %s", xid)
				}
			}
		}
	}
}

func TestRetainedBAuditCertificateControls(t *testing.T) {
	_, m := auditBundle(t)
	c := retainedBConfigs()[0]
	for _, test := range []struct{ name, side, action string }{
		{"dropped q", "X", "/q/peer_debit"},
		{"dropped closure", "X", "//close_root"},
		{"dropped reference branch", "K", "/e/choose_local"},
		{"wrong owner", "X", "/e/admit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, _ := retainedBWitness(t, c, true)
			g, mptr := p.X, &m
			if test.side == "K" {
				g, mptr = p.K, nil
			}
			for i, e := range g.Edges {
				if !strings.HasSuffix(e.ID, test.action) {
					continue
				}
				want := "missing supplied edge"
				if test.name == "wrong owner" {
					r := p.Rules[e.Rule]
					r.Owner = "component"
					p.Rules[e.Rule] = r
					want = "wrong supplied owner"
				} else {
					g.Edges = append(g.Edges[:i], g.Edges[i+1:]...)
				}
				if _, err := auditTable(c, g, p.Rules, mptr, true); err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("want %s, got %v", want, err)
				}
				return
			}
			t.Fatal("missing control target")
		})
	}
	for _, what := range []string{"path word", "path endpoint", "bilateral readiness"} {
		t.Run(what, func(t *testing.T) {
			p, cert := retainedBWitness(t, c, true)
			if what == "bilateral readiness" {
				cert.Environment = cert.Environment[1:]
				r := checkWeakCertificate(p, cert)
				if !hasWeakIssue(r.Readiness, "missing_environment") {
					t.Fatal("missing readiness counterexample")
				}
				return
			}
			for i, move := range cert.Component {
				if what == "path word" {
					if !strings.HasSuffix(move.Edge, "/h/invoke_commit") || len(move.Path) != 2 {
						continue
					}
					move.Direct = ""
					move.Path = move.Path[:1]
				} else {
					if !strings.HasSuffix(move.Edge, "/e/request_consume") || len(move.Path) != 1 {
						continue
					}
					// Same ordered word: append the silent no-helper choice
					// after the matched Receive. This lands at the wrong K state.
					idx, _ := indexWeakGraph(p.K)
					move.Path = append(move.Path, idx.edges[move.Path[0]].To+"/e/choose_local")
				}
				cert.Component[i] = move
				r := checkWeakCertificate(p, cert)
				clause := map[string]string{"path word": "word", "path endpoint": "endpoint"}[what]
				if len(r.Input) != 0 || !hasWeakIssue(r.Transfer, clause) || what == "path endpoint" && hasWeakIssue(r.Transfer, "word") {
					t.Fatalf("want %s, got %+v", clause, r)
				}
				return
			}
			t.Fatal("no helper-Invoke match")
		})
	}
}

func TestRetainedBAuditOpenProgress(t *testing.T) {
	_, m := auditBundle(t)
	for _, c := range retainedBConfigs() {
		c.Closure = false
		p, _ := retainedBWitness(t, c, true)
		ids, err := auditTable(c, p.X, p.Rules, &m, true)
		if err != nil {
			t.Fatal(err)
		}
		for s := range ids {
			edges := retainedBSuccessors(c, s)
			for _, e := range edges {
				if auditWork(e.Next) != auditWork(s)+1 || auditWork(e.Next) > 64 {
					t.Fatalf("open work rank %s", e.Event.Kind)
				}
			}
			if (len(edges) == 0) != auditMaximum(c, s) {
				t.Fatalf("bad maximal classification: %+v", s)
			}
			if auditMaximum(c, s) && c.Request.Mode == "pending" {
				s.PrivatePC[entry] = 10
				if auditMaximum(c, s) {
					t.Fatal("accepted pending before branch into actual unmatched receive")
				}
			}
		}
	}
}

func TestRetainedBAuditServicedControls(t *testing.T) {
	_, m := auditBundle(t)
	for _, kind := range []string{"spin", "stuck"} {
		t.Run(kind, func(t *testing.T) {
			c := retainedBConfigs()[0]
			c.Closure = false
			p, cert := retainedBWitness(t, c, true)
			ids, err := auditTable(c, p.X, p.Rules, &m, true)
			if err != nil {
				t.Fatal(err)
			}
			p.X.Boundaries = map[string]string{}
			id := ""
			for s, nid := range ids {
				n := p.X.Nodes[nid]
				n.Rank = 64 - auditWork(s)
				if auditMaximum(c, s) {
					n.Boundary = nid
					p.X.Boundaries[nid] = n.Projection
				}
				p.X.Nodes[nid] = n
				// Helper was consumed, q is spent, caller awaits e. Only e's
				// PC3 branch is due; all earlier protocol work was serviced.
				if s.PrivatePC[entry] == 3 && s.PeerTicket == spent && s.Ops[helper].Consumed && s.RootPC == await {
					id = nid
				}
			}
			if id == "" || len(checkWeakRank(p.X)) != 0 {
				t.Fatal("original open graph/rank or serviced prefix")
			}
			ix, _ := indexWeakGraph(p.X)
			outs := ix.out[id]
			if len(outs) != 1 || !p.Rules[outs[0].Rule].Private {
				t.Fatalf("unserviced alternative at %s: %+v", id, outs)
			}
			edge := outs[0].ID
			for i, e := range p.X.Edges {
				if e.ID == edge {
					if kind == "spin" {
						p.X.Edges[i].To = id
					} else {
						p.X.Edges = append(p.X.Edges[:i], p.X.Edges[i+1:]...)
					}
					break
				}
			}
			if kind == "stuck" {
				for i, m := range cert.Component {
					if m.Edge == edge {
						cert.Component = append(cert.Component[:i], cert.Component[i+1:]...)
						break
					}
				}
			}
			if !checkWeakCertificate(p, cert).accepted() {
				t.Fatal("finite-prefix weak transfer rejected")
			}
			if kind == "spin" {
				if !hasWeakIssue(checkWeakRank(p.X), "rank") || len(checkWeakCycle(p.X, p.Rules, []string{edge})) != 0 {
					t.Fatal("unconfirmed serviced private divergence")
				}
			} else if !hasWeakIssue(checkWeakRank(p.X), "stuck") {
				t.Fatal("missing unexpected finite maximum")
			}
		})
	}
}

func TestRetainedBAuditStaleIdentity(t *testing.T) {
	raw, _ := auditBundle(t)
	// Mutate well-formed JSON separately in the closure and initializer.
	for _, which := range []string{"helper", "initializer"} {
		t.Run(which, func(t *testing.T) {
			var bundle map[string]json.RawMessage
			if err := json.Unmarshal(raw, &bundle); err != nil {
				t.Fatal(err)
			}
			var changed []byte
			switch which {
			case "helper":
				var artifacts []struct {
					ID, Semantics string
					Bytes         []byte
				}
				if err := json.Unmarshal(bundle["Programs"], &artifacts); err != nil {
					t.Fatal(err)
				}
				found := false
				for i, a := range artifacts {
					if a.ID != auditPrograms[helper] {
						continue
					}
					var p executor.Program
					if err := json.Unmarshal(a.Bytes, &p); err != nil {
						t.Fatal(err)
					}
					p.Code[4].A = executor.Lit(schema.Int(0))
					newBytes, newID, err := executor.EncodeProgram(p)
					if err != nil || newID == a.ID {
						t.Fatalf("mutant identity %s %v", newID, err)
					}
					artifacts[i].Bytes = newBytes
					artifacts[i].ID = newID
					found = true
				}
				if !found {
					t.Fatal("helper absent")
				}
				changed, _ = json.Marshal(artifacts)
				bundle["Programs"] = changed
			case "initializer":
				var d struct {
					Members map[string]struct{ Initial executor.Projection }
				}
				if err := json.Unmarshal(bundle["Descriptor"], &d); err != nil {
					t.Fatal(err)
				}
				init := d.Members["entry"].Initial
				if len(init.Data) != 0 {
					t.Fatal("initial data not empty")
				}
				dst := strings.Replace(string(bundle["Descriptor"]), `"Data":{}`, `"Data":{"extra":{"Kind":"int","Number":1,"Text":"","Flag":false}}`, 1)
				if dst == string(bundle["Descriptor"]) {
					t.Fatal("initializer not changed")
				}
				bundle["Descriptor"] = json.RawMessage(dst)
			}
			// Preserve all unchanged bytes; reserializing the outer map changes
			// field ordering, which would test the wrong reason for a mismatch.
			var start map[string]json.RawMessage
			if err := json.Unmarshal(raw, &start); err != nil {
				t.Fatal(err)
			}
			field := "Descriptor"
			if which == "helper" {
				field = "Programs"
			}
			mutant := strings.Replace(string(raw), string(start[field]), string(bundle[field]), 1)
			if mutant == string(raw) || !json.Valid([]byte(mutant)) {
				t.Fatal("invalid detached mutation")
			}
			if _, err := auditLoad([]byte(mutant)); err == nil || !strings.Contains(err.Error(), "stale B bundle") {
				t.Fatalf("old correspondence accepted %s: %v", which, err)
			}
		})
	}
}
