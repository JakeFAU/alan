package executor

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	recordschema "github.com/JakeFAU/alan/schema"
)

func bindingProgram(t *testing.T, peer, obligation string) (Program, Artifact) {
	t.Helper()
	empty, _ := recordschema.Record{}.Retain()
	record, _ := recordschema.Record{"literal": {Kind: "string", MaxBytes: 100}}.Retain()
	p := Program{Semantics: AuthorSemanticVersion, Code: []Instruction{{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text(peer)), Generation: Lit(Int(1)), Obligation: Lit(Text(obligation)), Payload: map[string]Operand{"literal": Lit(Text(peer))}}}, {Op: "end", Outcome: "succeeded"}}, Author: &ProgramMetadata{Compiler: AuthorCompilerVersion, Inputs: empty, Outputs: empty, Positions: []string{"send", "end"}, Ports: map[string]PortDeclaration{"public": {Peer: peer, Obligation: obligation, Send: record, Receive: empty, SendUsed: true}}, Receives: map[int]recordschema.Retained{}}}
	raw, id, err := EncodeProgram(p)
	if err != nil {
		t.Fatal(err)
	}
	return p, Artifact{ID: id, Semantics: p.Semantics, Bytes: raw}
}
func TestCompositionBindingRetainsArtifactAndPayload(t *testing.T) {
	p, a := bindingProgram(t, "caller", "reply")
	provision, c := compositionFixture(t)
	provision.Programs = append(provision.Programs, a)
	c.Templates = []compositionTemplate{{Members: map[string]compositionTemplateMember{"entry": {ProgramID: a.ID, Initial: Projection{Status: "open", Data: map[string]Value{}}}}, EntryMember: "entry", EntryPort: "public", CallerPeer: "caller"}}
	path := t.TempDir() + "/binding.sqlite"
	s, err := createCompositionStore(t.Context(), path, provision, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	retained, err := s.composition.descriptor()
	if err != nil {
		t.Fatal(err)
	}
	template := retained.Templates[0]
	root := retained.Roots[0]
	operation := identity("A", int64(1), int64(1), 0)
	invocation := compositionInvocationID(s.namespace, operation)
	binding := compositionBinding{Version: compositionBindingVersion, Namespace: s.namespace, InvocationID: invocation, OperationID: operation, TemplateID: template.ID, RootID: root.ID, Caller: RunID{"A", 1}, Depth: 1, Members: map[string]RunID{"entry": compositionMemberID(invocation, "entry")}, Programs: map[string]string{"entry": a.ID}}
	env, err := retainCompositionBinding(s.namespace, root.ID, template, binding)
	if err != nil {
		t.Fatal(err)
	}
	binding.Members["entry"] = RunID{"mutated", 1}
	bound, err := resolveCompositionProgram(p, env, "entry")
	if err != nil {
		t.Fatal(err)
	}
	if bound.Code[0].Effect.Target.Literal.Text != "A" || bound.Code[0].Effect.Payload["literal"].Literal.Text != "caller" {
		t.Fatal("binding rewrote payload or failed endpoint resolution")
	}
	raw, id, err := EncodeProgram(p)
	if err != nil || id != a.ID || string(raw) != string(a.Bytes) {
		t.Fatal("binding mutated source artifact")
	}
	reduced, err := Reduce(bound, Projection{Status: "open", Data: map[string]Value{}}, nil, compositionMemberID(invocation, "entry"), 1, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if reduced.Intents[0].Target != "A" || reduced.Intents[0].Payload["literal"] != Text("caller") {
		t.Fatal(reduced)
	}
	if _, err := resolveCompositionProgram(p, env, "missing"); err == nil {
		t.Fatal("missing member accepted")
	}
	var copyBinding compositionBinding
	if err = json.Unmarshal([]byte(env.binding), &copyBinding); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*compositionBinding)
	}{
		{"cross invocation", func(b *compositionBinding) {
			b.Members["entry"] = compositionMemberID(strings.Repeat("f", 64), "entry")
		}},
		{"wrong program", func(b *compositionBinding) { b.Programs["entry"] = strings.Repeat("f", 64) }},
		{"root", func(b *compositionBinding) { b.RootID = strings.Repeat("f", 64) }},
		{"caller member", func(b *compositionBinding) { b.Caller = b.Members["entry"] }},
		{"version", func(b *compositionBinding) { b.Version = "future" }},
		{"invalid caller encoding", func(b *compositionBinding) { b.Caller.Subject = string([]byte{0xff}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b compositionBinding
			raw, _ := json.Marshal(copyBinding)
			json.Unmarshal(raw, &b)
			b.ID = ""
			tc.change(&b)
			if _, err := retainCompositionBinding(s.namespace, root.ID, template, b); err == nil {
				t.Fatal("invalid binding accepted")
			}
		})
	}
	stored, err := loadProgram(t.Context(), s.conn, a.ID, DefaultLimits())
	if err != nil || !reflect.DeepEqual(stored, p) {
		t.Fatalf("retained artifact changed: %v", err)
	}
}

func TestCompositionBindingTwoMemberClosure(t *testing.T) {
	entry, entryArtifact := bindingProgram(t, "caller", "reply")
	worker, workerArtifact := bindingProgram(t, "entry", "internal")
	record := entry.Author.Ports["public"].Send
	entry.Author.Ports["work"] = PortDeclaration{Peer: "worker", Obligation: "internal", Send: record, Receive: record, SendUsed: true, ReceiveUsed: true}
	send := entry.Code[0]
	send.Effect = &Effect{Kind: "message", Target: Lit(Text("worker")), Generation: Lit(Int(1)), Obligation: Lit(Text("internal")), Payload: map[string]Operand{"literal": Lit(Text("worker"))}}
	receive := Instruction{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("worker")), Generation: Lit(Int(1)), Obligation: Lit(Text("internal")), Operation: Lit(Text(""))}, Bind: map[string]string{"literal": "answer"}}
	entry.Code = append([]Instruction{send, receive}, entry.Code...)
	entry.Author.Positions = []string{"send work", "receive work", "send reply", "end"}
	entry.Author.Receives[1] = record
	wp := worker.Author.Ports["public"]
	wp.Receive = record
	wp.ReceiveUsed = true
	worker.Author.Ports["public"] = wp
	receive.Match = &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("entry")), Generation: Lit(Int(1)), Obligation: Lit(Text("internal")), Operation: Lit(Text(""))}
	worker.Code = append([]Instruction{receive}, worker.Code...)
	worker.Author.Positions = []string{"receive", "send", "end"}
	worker.Author.Receives[0] = record
	var err error
	entryArtifact.Bytes, entryArtifact.ID, err = EncodeProgram(entry)
	if err != nil {
		t.Fatal(err)
	}
	workerArtifact.Bytes, workerArtifact.ID, err = EncodeProgram(worker)
	if err != nil {
		t.Fatal(err)
	}
	p, c := compositionFixture(t)
	p.Programs = append(p.Programs, entryArtifact, workerArtifact)
	c.Templates = []compositionTemplate{{EntryMember: "entry", EntryPort: "public", CallerPeer: "caller", Members: map[string]compositionTemplateMember{"entry": {ProgramID: entryArtifact.ID, Initial: Projection{Status: "open", Data: map[string]Value{}}}, "worker": {ProgramID: workerArtifact.ID, Initial: Projection{Status: "open", Data: map[string]Value{}}}}}}
	s, err := createCompositionStore(t.Context(), t.TempDir()+"/two-member.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	retained, _ := s.composition.descriptor()
	template := retained.Templates[0]
	root := retained.Roots[0]
	invocation := compositionInvocationID(s.namespace, "committed-operation")
	b := compositionBinding{Version: compositionBindingVersion, Namespace: s.namespace, RootID: root.ID, TemplateID: template.ID, OperationID: "committed-operation", InvocationID: invocation, Caller: RunID{"A", 1}, Depth: 1, Members: map[string]RunID{}, Programs: map[string]string{}}
	for name, m := range template.Members {
		b.Members[name] = compositionMemberID(invocation, name)
		b.Programs[name] = m.ProgramID
	}
	env, err := retainCompositionBinding(s.namespace, root.ID, template, b)
	if err != nil {
		t.Fatal(err)
	}
	resolvedEntry, err := resolveCompositionProgram(entry, env, "entry")
	if err != nil {
		t.Fatal(err)
	}
	resolvedWorker, err := resolveCompositionProgram(worker, env, "worker")
	if err != nil {
		t.Fatal(err)
	}
	if resolvedEntry.Code[0].Effect.Target.Literal.Text != b.Members["worker"].Subject || resolvedWorker.Code[0].Match.Source.Literal.Text != b.Members["entry"].Subject || resolvedEntry.Code[2].Effect.Target.Literal.Text != "A" {
		t.Fatal("two-member/caller endpoints cross-wired")
	}
	if resolvedEntry.Code[0].Effect.Payload["literal"].Literal.Text != "worker" {
		t.Fatal("relative name payload substituted")
	}
}
func TestCompositionTemplateRejectsUndeclaredEndpoint(t *testing.T) {
	program, artifact := bindingProgram(t, "caller", "reply")
	program.Code[0].Effect.Target = Lit(Text("unapproved"))
	var err error
	artifact.Bytes, artifact.ID, err = EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p, c := compositionFixture(t)
	p.Programs = append(p.Programs, artifact)
	c.Templates = []compositionTemplate{{EntryMember: "entry", EntryPort: "public", CallerPeer: "caller", Members: map[string]compositionTemplateMember{"entry": {ProgramID: artifact.ID, Initial: Projection{Status: "open", Data: map[string]Value{}}}}}}
	s, err := createCompositionStore(t.Context(), t.TempDir()+"/invalid.sqlite", p, c, DefaultLimits())
	if s != nil {
		s.Close()
	}
	if err == nil {
		t.Fatal("code endpoint outside immutable declarations admitted to catalogue")
	}
}
