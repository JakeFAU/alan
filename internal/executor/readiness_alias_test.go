package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	recordschema "github.com/JakeFAU/alan/schema"
	"github.com/mattn/go-sqlite3"
)

func p3ProgramBytes(t *testing.T, p Program) []byte {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestP3ReadinessSharedInputsAndPorts(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, func(p *Provision, c *compositionProvision) { c.Roots[0].Limits.MaxInFlight = 4 })
	first := invokeOnce(t, s)
	second := invokeOnce(t, s)
	if !first.Accepted || !second.Accepted {
		t.Fatalf("admissions: %+v %+v", first, second)
	}
	var programs []Program
	var originals [][]byte
	var bound []Program
	for _, admission := range []compositionAdmission{first, second} {
		for _, id := range admission.Members {
			run, err := s.Snapshot(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			p, err := loadProgram(t.Context(), s.conn, run.ProgramID, s.limits)
			if err != nil {
				t.Fatal(err)
			}
			if p.Code[0].Op != "receive" {
				continue
			}
			original := p3ProgramBytes(t, p)
			b, err := boundCompositionProgram(t.Context(), s.conn, p, id, s.limits)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(original, p3ProgramBytes(t, p)) {
				t.Fatal("binding changed unbound original")
			}
			programs = append(programs, p)
			originals = append(originals, original)
			bound = append(bound, b)
			ids, err := readRunIDs(t.Context(), s.conn)
			if err != nil {
				t.Fatal(err)
			}
			var last RunID
			for i, candidate := range ids {
				if candidate == id {
					last = ids[(i+len(ids)-1)%len(ids)]
					break
				}
			}
			r, err := p3Compare(t, t.Context(), s, &last)
			if err != nil || r == nil || r.authority != run.Authority() || r.observation == "" {
				t.Fatalf("member readiness: %+v %v", r, err)
			}
			observations, err := readObservations(t.Context(), s.conn, s.limits)
			if err != nil {
				t.Fatal(err)
			}
			selected := observations[r.observation]
			if selected.Run != id || selected.Operation != b.Code[0].Match.Operation.Literal.Text {
				t.Fatal("another invocation's receive selector reused")
			}
		}
	}
	if len(bound) != 2 || !bytes.Equal(originals[0], originals[1]) || reflect.DeepEqual(bound[0].Code[0].Match.Operation, bound[1].Code[0].Match.Operation) {
		t.Fatal("fixture lacks shared ID and distinct bound selectors")
	}
	// Returned bound values deliberately mutated only after real binding. Every
	// nested owner must be independent of the original and the other binding.
	secondBefore := p3ProgramBytes(t, bound[1])
	bound[0].Code[0].Match.Source.Literal.Text = "mutated"
	bound[0].Code[1].Effect.Payload["result"].Literal.Text = "mutated"
	bound[0].Author.Positions[0] = "mutated"
	port := bound[0].Author.Ports["public"]
	field := port.Send.Record["result"]
	field.Enum[0] = "mutated"
	port.Send.Record["result"] = field
	bound[0].Author.Receives[0].Record["extra"] = recordschema.Field{Kind: "int"}
	for i, p := range programs {
		if !bytes.Equal(originals[i], p3ProgramBytes(t, p)) {
			t.Fatal("nested mutation escaped bound copy")
		}
	}
	if !bytes.Equal(secondBefore, p3ProgramBytes(t, bound[1])) {
		t.Fatal("mutation reached another binding")
	}
	// Plain retained programs can also evaluate different initial input operands.
	p := p3WaitingProvision(t, 2, 1)
	for i := range p.Runs {
		p.Runs[i].Projection.Data["peer"] = Text([]string{"local", "different"}[i])
	}
	var program Program
	if err := decode(p.Programs[0].Bytes, &program); err != nil {
		t.Fatal(err)
	}
	program.Code[0].Match.Source = Ref("peer")
	raw, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = []Artifact{{ID: id, Semantics: program.Semantics, Bytes: raw}}
	for i := range p.Runs {
		p.Runs[i].ProgramID = id
	}
	inputs, _ := messageStore(t, p)
	p3Admit(t, inputs, "r00", "match", 10)
	p3Admit(t, inputs, "r01", "wrong-peer", 20)
	last := RunID{"r00", 1}
	r, err := p3Compare(t, t.Context(), inputs, &last)
	if err != nil || r == nil || r.authority.Run != last {
		t.Fatalf("per-run inputs ignored: %+v %v", r, err)
	}
}

// This is a source-level alias witness, using an encoded, validated program.
// The template is deliberately a unit binding descriptor, not claimed as a
// provisioned handoff history. The public handoff suite covers retained setup.
func TestP3ReadinessHandoffShallowAliasWitness(t *testing.T) {
	p, _ := bindingProgram(t, HandoffPeer, "reply")
	before := p3ProgramBytes(t, p)
	shallow := p
	template := compositionTemplate{EntryMember: "entry", Handoff: &InvocationHandoff{Caller: RunID{"caller", 1}, Sequence: 1, CallerPort: "public"}}
	bindHandoffPort(&shallow, "test-namespace", template)
	if bytes.Equal(before, p3ProgramBytes(t, p)) || p.Author.Ports["public"].Peer == HandoffPeer || p.Code[0].Effect.Target.Literal.Text == HandoffPeer {
		t.Fatal("fixture no longer demonstrates the nested handoff alias hazard")
	}
	// A second load is the existing isolation boundary. Eligibility must have
	// been decided before binding removes the marker from the first value.
	fresh, _ := bindingProgram(t, HandoffPeer, "reply")
	if !bytes.Equal(before, p3ProgramBytes(t, fresh)) {
		t.Fatal("independent decode did not restore unbound source")
	}
}

func TestP3ReadinessReduceNestedIsolation(t *testing.T) {
	empty, _ := recordschema.Record{}.Retain()
	field := recordschema.Field{Kind: "list", MaxItems: 2, MaxBytes: 512, Element: recordschema.Record{"value": {Kind: "string", MaxBytes: 20, Enum: []string{"two", "one"}}}}
	literal := recordschema.List(map[string]Value{"value": Text("one")})
	p := Program{Semantics: CollectionSemanticVersion, Author: &ProgramMetadata{Compiler: CollectionCompilerVersion, Inputs: empty, Outputs: empty, Positions: []string{"set", "end"}, Collections: map[string]recordschema.Field{"items": field}}, Code: []Instruction{{Op: "set", Dst: "items", A: Lit(literal), Collection: &field}, {Op: "end", Outcome: "succeeded"}}}
	raw, id, err := EncodeProgram(p)
	if err != nil {
		t.Fatal(err)
	}
	provision := Provision{Programs: []Artifact{{ID: id, Semantics: p.Semantics, Bytes: raw}}, Subjects: []Subject{{ID: "list", Generation: 1, Owner: "owner", Revision: 1}}, Runs: []InitialRun{{ID: RunID{"list", 1}, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{}}}}}
	s, _ := messageStore(t, provision)
	p, err = loadProgram(t.Context(), s.conn, id, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	before := p3ProgramBytes(t, p)
	initial := current(t, s, "list").Projection
	result, err := Reduce(p, initial, nil, RunID{"list", 1}, 1, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	// ListData's mutating decoder is deliberately invoked on the output pointer.
	if err = json.Unmarshal([]byte(`[{"value":{"Kind":"string","Number":0,"Text":"two","Flag":false}}]`), result.Projection.Data["items"].List); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, p3ProgramBytes(t, p)) {
		t.Fatal("reducer output aliases literal/schema")
	}
	if len(initial.Data) != 0 {
		t.Fatal("reducer changed initial projection")
	}
	if _, err = p3Compare(t, t.Context(), s, nil); err != nil {
		t.Fatal(err)
	}
	// AuthorityUse contains only scalar/value fields; its output pointer is copied.
	ap := requestAuthorityFixture(t, nil)
	var authority Program
	if err = decode(ap.Programs[1].Bytes, &authority); err != nil {
		t.Fatal(err)
	}
	authorityBefore := p3ProgramBytes(t, authority)
	reduction, err := Reduce(authority, Projection{PC: 1, Status: "open", Data: map[string]Value{}}, nil, RunID{"A", 1}, 2, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	reduction.Intents[0].Authority.Path = "mutated"
	if !bytes.Equal(authorityBefore, p3ProgramBytes(t, authority)) {
		t.Fatal("output authority aliases program")
	}
}

func TestP3ReadinessCancellationAtRunBoundary(t *testing.T) {
	// Cancel at preparation of the second run's projection query, a remaining
	// logical boundary. Do not compare removed load-number/wall-clock faults.
	for _, reference := range []bool{true, false} {
		s, _ := messageStore(t, p3WaitingProvision(t, 4, 1))
		ctx, cancel := context.WithCancel(t.Context())
		reached := 0
		err := s.conn.Raw(func(driver any) error {
			driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, column, db string) int {
				if op == sqlite3.SQLITE_READ && table == "runs" && column == "projection" {
					reached++
					if reached == 2 {
						cancel()
					}
				}
				return sqlite3.SQLITE_OK
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		var r *readyRun
		if reference {
			r, err = s.p3BaselineSelectReady(ctx, nil)
		} else {
			r, err = s.selectReady(ctx, nil)
		}
		cancel()
		if e := s.conn.Raw(func(driver any) error { driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil }); e != nil {
			t.Fatal(e)
		}
		if reached != 2 || r != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("logical cancellation reference=%t reached=%d ready=%+v error=%v", reference, reached, r, err)
		}
	}
}

// Build through the same retained-template and invocation-store validators as
// ordinary applications. Two static callers share the program, but only root
// owns the fixed handoff descriptor. The other caller must retain HandoffPeer.
func p3HandoffStore(t *testing.T) *Store {
	t.Helper()
	provision, c := invocationFixture(t, "succeeded", 1)
	empty, _ := recordschema.Record{}.Retain()
	target, _ := recordschema.Record{"target": {Kind: "string", MaxBytes: 1024}}.Retain()
	var child Program
	if err := decode(provision.Programs[0].Bytes, &child); err != nil {
		t.Fatal(err)
	}
	child.Author.Ports["handoff"] = PortDeclaration{Peer: "caller", Obligation: "identity", Send: target, Receive: target, SendUsed: true, ReceiveUsed: true}
	child.Code = append(child.Code[:1], Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("caller")), Generation: Lit(Int(1)), Obligation: Lit(Text("identity")), Payload: map[string]Operand{"target": Lit(Text("original"))}}}, Instruction{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("caller")), Generation: Lit(Int(1)), Obligation: Lit(Text("identity")), Operation: Lit(Text(""))}}, child.Code[1], child.Code[2])
	child.Author.Positions = make([]string, len(child.Code))
	child.Author.Receives[2] = target
	raw, id, err := EncodeProgram(child)
	if err != nil {
		t.Fatal(err)
	}
	childArtifact := Artifact{ID: id, Semantics: child.Semantics, Bytes: raw}
	template := c.Templates[0]
	template.ID = ""
	template.Version = ""
	delete(template.Members, "helper")
	member := template.Members["entry"]
	member.ProgramID = id
	template.Members["entry"] = member
	template.Handoff = &InvocationHandoff{Caller: RunID{"root", 1}, Sequence: 1, CallerPort: "handoff", EntryPort: "handoff"}
	template, err = RetainTemplate(template, []Artifact{childArtifact})
	if err != nil {
		t.Fatal(err)
	}
	var parent Program
	if err = decode(provision.Programs[len(provision.Programs)-1].Bytes, &parent); err != nil {
		t.Fatal(err)
	}
	parent.Author.Ports = map[string]PortDeclaration{"handoff": {Peer: HandoffPeer, Obligation: "identity", Send: target, Receive: target, SendUsed: true, ReceiveUsed: true}}
	parent.Author.Invocations["child"] = InvocationDeclaration{TemplateID: template.ID, Input: empty, Output: child.Author.Ports["public"].Send}
	invoke := parent.Code[0]
	invoke.Effect.Target = Lit(Text(template.ID))
	parent.Code = []Instruction{invoke, {Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text(HandoffPeer)), Generation: Lit(Int(1)), Obligation: Lit(Text("identity")), Operation: Lit(Text(""))}}, {Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text(HandoffPeer)), Generation: Lit(Int(1)), Obligation: Lit(Text("identity")), Payload: map[string]Operand{"target": Lit(Text("ack"))}}}, {Op: "end", Outcome: "succeeded"}}
	parent.Author.Positions = make([]string, len(parent.Code))
	parent.Author.Receives = map[int]recordschema.Retained{1: target}
	raw, id, err = EncodeProgram(parent)
	if err != nil {
		t.Fatal(err)
	}
	provision.Programs = []Artifact{childArtifact, {ID: id, Semantics: parent.Semantics, Bytes: raw}}
	provision.Runs[0].ProgramID = id
	provision.Subjects = append(provision.Subjects, Subject{ID: "z-clone", Generation: 1, Owner: "owner", Revision: 1})
	provision.Runs = append(provision.Runs, InitialRun{ID: RunID{"z-clone", 1}, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{}}})
	c.Templates = []compositionTemplate{template}
	c.Roots[0].Templates = []string{template.ID}
	c.Roots[0].Members = append(c.Roots[0].Members, RunID{"z-clone", 1})
	c.Roots[0].Limits.MaxInFlight = 4
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/handoff.sqlite", provision, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for _, name := range []string{"root", "z-clone"} {
		if _, err = s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestP3ReadinessHandoffExcludedBeforeBinding(t *testing.T) {
	s := p3HandoffStore(t)
	run := current(t, s, "root")
	p, err := loadProgram(t.Context(), s.conn, run.ProgramID, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	before := p3ProgramBytes(t, p)
	bound, err := boundCompositionProgram(t.Context(), s.conn, p, run.ID, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	if bound.Author.Ports["handoff"].Peer == HandoffPeer || bytes.Equal(before, p3ProgramBytes(t, p)) {
		t.Fatal("actual handoff path failed to expose shallow mutation")
	}
	fresh, err := loadProgram(t.Context(), s.conn, run.ProgramID, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	other, err := boundCompositionProgram(t.Context(), s.conn, fresh, RunID{"z-clone", 1}, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	if other.Author.Ports["handoff"].Peer != HandoffPeer {
		t.Fatal("another caller inherited handoff binding")
	}
	r, err, loads := p3CountLoads(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), nil) })
	if err != nil || r != nil || loads != 2 {
		t.Fatalf("handoff exclusion must precede binding: ready=%+v error=%v loads=%d want=2", r, err, loads)
	}
	if _, err = p3Compare(t, t.Context(), s, nil); err != nil {
		t.Fatal(err)
	}
	// Even a shared, previously loaded program must reach each caller binding.
	// Disposable malformed handoff descriptor, injected before a later call.
	if _, err = s.conn.ExecContext(t.Context(), "UPDATE composition_templates SET descriptor=?", []byte(`{"Handoff":{"Caller":{"Subject":"root","Generation":1}},"unknown":true}`)); err != nil {
		t.Fatal(err)
	}
	last := RunID{"root", 1} // z-clone validates this same ID before root reaches the corrupted binding.
	if r, err = p3Compare(t, t.Context(), s, &last); err == nil || r != nil {
		t.Fatalf("handoff error suppressed: %+v %v", r, err)
	}
}

func TestP3ReadinessCloseJoinsSelector(t *testing.T) {
	for _, reference := range []bool{true, false} {
		s, _ := messageStore(t, p3WaitingProvision(t, 2, 1))
		reached, release := make(chan struct{}), make(chan struct{})
		once := false
		if err := s.conn.Raw(func(driver any) error {
			driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, column, db string) int {
				if !once && op == sqlite3.SQLITE_READ && table == "programs" && column == "bytes" {
					once = true
					close(reached)
					<-release
				}
				return sqlite3.SQLITE_OK
			})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		selected := make(chan error, 1)
		go func() {
			var err error
			if reference {
				_, err = s.p3BaselineSelectReady(t.Context(), nil)
			} else {
				_, err = s.selectReady(t.Context(), nil)
			}
			selected <- err
		}()
		<-reached
		closing := make(chan error, 1)
		go func() { closing <- s.Close() }()
		close(release)
		if err := <-selected; err != nil {
			t.Fatal(err)
		}
		if err := <-closing; err != nil {
			t.Fatal(err)
		}
		if r, err := p3Compare(t, t.Context(), s, nil); r != nil || err == nil {
			t.Fatalf("closed owner: %+v %v", r, err)
		}
	}
}

// p3SameCallBindingCut creates two admitted instances of one retained two-receive
// child. The lexically first member consumes its only request and waits at its
// second receive; the later member retains its initial request. No SQL mutation
// is used to construct this positive cut.
func p3SameCallBindingCut(t *testing.T) (*Store, RunID, *readyRun) {
	t.Helper()
	provision, c := invocationFixture(t, "succeeded", 1)
	var child, parent Program
	if err := decode(provision.Programs[0].Bytes, &child); err != nil {
		t.Fatal(err)
	}
	child.Code = append([]Instruction{child.Code[0]}, child.Code...)
	child.Author.Positions = make([]string, len(child.Code))
	child.Author.Receives[1] = child.Author.Receives[0]
	raw, id, err := EncodeProgram(child)
	if err != nil {
		t.Fatal(err)
	}
	childArtifact := Artifact{ID: id, Semantics: child.Semantics, Bytes: raw}
	template := c.Templates[0]
	template.ID = ""
	delete(template.Members, "helper")
	member := template.Members["entry"]
	member.ProgramID = id
	template.Members["entry"] = member
	template, err = RetainTemplate(template, []Artifact{childArtifact})
	if err != nil {
		t.Fatal(err)
	}
	if err = decode(provision.Programs[len(provision.Programs)-1].Bytes, &parent); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		parent.Code[i].Effect.Target = Lit(Text(template.ID))
	}
	declaration := parent.Author.Invocations["child"]
	declaration.TemplateID = template.ID
	parent.Author.Invocations["child"] = declaration
	raw, id, err = EncodeProgram(parent)
	if err != nil {
		t.Fatal(err)
	}
	provision.Programs = []Artifact{childArtifact, {ID: id, Semantics: parent.Semantics, Bytes: raw}}
	provision.Runs[0].ProgramID = id
	c.Templates = []compositionTemplate{template}
	c.Roots[0].Templates = []string{template.ID}
	c.Roots[0].Limits.MaxInFlight = 4
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/same-call.sqlite", provision, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for range 2 {
		admission := invokeOnce(t, s)
		if !admission.Accepted || len(admission.Members) != 1 {
			t.Fatalf("admission: %+v", admission)
		}
	}
	ids, err := readRunIDs(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	var members []RunID
	for _, run := range ids {
		if run != (RunID{"root", 1}) {
			members = append(members, run)
		}
	}
	if len(members) != 2 {
		t.Fatalf("members: %+v", members)
	}
	states := make([]RunState, 2)
	observations, err := readObservations(t.Context(), s.conn, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	selected := make([]string, 2)
	operations := make([]string, 2)
	for i, run := range members {
		states[i], err = s.Snapshot(t.Context(), run)
		if err != nil {
			t.Fatal(err)
		}
		for oid, o := range observations {
			if o.Run == run && !o.consumed.Valid {
				if selected[i] != "" {
					t.Fatal("multiple initial requests")
				}
				selected[i] = oid
				operations[i] = o.Operation
			}
		}
		if selected[i] == "" {
			t.Fatal("missing initial request")
		}
	}
	if states[0].ProgramID != states[1].ProgramID || operations[0] == operations[1] {
		t.Fatal("fixture lacks shared ID and distinct invocation selectors")
	}
	waiting, err := s.Commit(t.Context(), states[0].Authority(), selected[0])
	if err != nil {
		t.Fatal(err)
	}
	if waiting.Projection.PC != 1 || waiting.Sequence != 1 {
		t.Fatalf("first member failed to enter second receive: %+v", waiting)
	}
	// Choose the actual predecessor, so the first visited run is the waiting
	// member. Verify the ready member follows directly before any root traversal.
	var last RunID
	for i, run := range ids {
		if run == members[0] {
			last = ids[(i+len(ids)-1)%len(ids)]
			if ids[(i+1)%len(ids)] != members[1] {
				t.Fatal("members not adjacent in traversal")
			}
		}
	}
	return s, last, &readyRun{authority: states[1].Authority(), observation: selected[1]}
}

type p3CompositionVisits struct{ runs, loads, bindings int }

// Counts actual queries in these composition fixtures: projection reads are
// per-run visits, programs.bytes reads are loads, and the membership join's
// member column is read exactly once by each per-run binding lookup.
func p3CountCompositionVisits(t *testing.T, s *Store, fn func() (*readyRun, error)) (*readyRun, error, p3CompositionVisits) {
	t.Helper()
	var counts p3CompositionVisits
	if err := s.conn.Raw(func(driver any) error {
		driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, column, db string) int {
			if op == sqlite3.SQLITE_READ {
				switch {
				case table == "runs" && column == "projection":
					counts.runs++
				case table == "programs" && column == "bytes":
					counts.loads++
				case table == "composition_members" && column == "member":
					counts.bindings++
				}
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.conn.Raw(func(driver any) error { driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil }); err != nil {
			t.Error(err)
		}
	}()
	r, err := fn()
	return r, err, counts
}

func TestP3ReadinessSameCallBoundPrograms(t *testing.T) {
	s, last, want := p3SameCallBindingCut(t)
	baseline, err, bc := p3CountCompositionVisits(t, s, func() (*readyRun, error) { return s.p3BaselineSelectReady(t.Context(), &last) })
	if err != nil || !reflect.DeepEqual(baseline, want) || bc != (p3CompositionVisits{2, 2, 2}) {
		t.Fatalf("baseline same-call cut: ready=%+v want=%+v counts=%+v error=%v", baseline, want, bc, err)
	}
	got, err, counts := p3CountCompositionVisits(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), &last) })
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("same-call bound-program reuse: ready=%+v want=%+v error=%v", got, want, err)
	}
	// This semantic control runs before and after optimization. The existing RED
	// owns required load reduction; both versions must actually bind both runs.
	if counts.runs != 2 || counts.bindings != 2 || (counts.loads != 1 && counts.loads != 2) {
		t.Fatalf("same-call work: %+v", counts)
	}
	t.Logf("baseline=%+v current=%+v", bc, counts)
}
