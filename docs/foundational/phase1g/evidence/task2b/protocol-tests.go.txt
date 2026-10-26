package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	recordschema "github.com/JakeFAU/alan/schema"
	"reflect"
	"strings"
	"testing"
)

func invocationFixture(t *testing.T, outcome string, replies int) (Provision, compositionProvision) {
	t.Helper()
	empty, _ := recordschema.Record{}.Retain()
	output, _ := (recordschema.Record{"result": {Kind: "string", MaxBytes: 30, Enum: []string{"ok", "invalid_result"}}}).Retain()
	child := Program{Semantics: AuthorSemanticVersion,
		Author: &ProgramMetadata{Compiler: AuthorCompilerVersion,
			Inputs:  empty,
			Outputs: empty,
			Ports: map[string]PortDeclaration{"public": {Peer: "caller",
				Obligation:  "work",
				Receive:     empty,
				Send:        output,
				ReceiveUsed: true,
				SendUsed:    true}},
			Receives: map[int]recordschema.Retained{0: empty}},
		Code: []Instruction{{Op: "receive",
			Match: &Match{Kind: "message",
				Provenance: "ordinary-message",
				Source:     Lit(Text("caller")),
				Generation: Lit(Int(1)),
				Obligation: Lit(Text("work")),
				Operation:  Lit(Text(""))}}}}
	value := "ok"
	if outcome == "failed" {
		value = "invalid_result"
	}
	for range replies {
		child.Code = append(child.Code,
			Instruction{Op: "emit",
				Effect: &Effect{Kind: "message",
					Target:     Lit(Text("caller")),
					Generation: Lit(Int(1)),
					Obligation: Lit(Text("work")),
					Payload:    map[string]Operand{"result": Lit(Text(value))}}})
	}
	child.Code = append(child.Code, Instruction{Op: "end", Outcome: outcome})
	child.Author.Positions = make([]string, len(child.Code))
	raw, id, err := EncodeProgram(child)
	if err != nil {
		t.Fatal(err)
	}
	artifact := Artifact{ID: id, Semantics: child.Semantics, Bytes: raw}
	helper := Program{Semantics: AuthorSemanticVersion,
		Author: &ProgramMetadata{Compiler: AuthorCompilerVersion,
			Inputs:    empty,
			Outputs:   empty,
			Positions: []string{"end"}},
		Code: []Instruction{{Op: "end",
			Outcome: "succeeded"}}}
	hraw, hid, err := EncodeProgram(helper)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := []Artifact{artifact, {ID: hid, Semantics: helper.Semantics, Bytes: hraw}}
	template, err := RetainTemplate(compositionTemplate{Members: map[string]compositionTemplateMember{"entry": {ProgramID: id,
		Initial: Projection{Status: "open",
			Data: map[string]Value{}}},
		"helper": {ProgramID: hid,
			Initial: Projection{Status: "open",
				Data: map[string]Value{}}}},
		EntryMember: "entry",
		EntryPort:   "public",
		CallerPeer:  "caller"},
		artifacts)
	if err != nil {
		t.Fatal(err)
	}
	invoke := Instruction{Op: "emit",
		Dst: "pending",
		Effect: &Effect{Kind: "invocation",
			Target:     Lit(Text(template.ID)),
			Generation: Lit(Int(1)),
			Obligation: Lit(Text("child")),
			Payload:    map[string]Operand{}}}
	root := Program{Semantics: CollectionSemanticVersion,
		Author: &ProgramMetadata{Compiler: CollectionCompilerVersion,
			Inputs:  empty,
			Outputs: empty,
			Positions: []string{"invoke",
				"invoke",
				"end"},
			Invocations: map[string]InvocationDeclaration{"child": {TemplateID: template.ID,
				Input:  empty,
				Output: output}}},
		Code: []Instruction{invoke,
			invoke,
			{Op: "end",
				Outcome: "succeeded"}}}
	rraw, rid, err := EncodeProgram(root)
	if err != nil {
		t.Fatal(err)
	}
	p := Provision{Programs: append(artifacts,
		Artifact{ID: rid,
			Semantics: root.Semantics,
			Bytes:     rraw}),
		Subjects: []Subject{{ID: "root",
			Generation: 1,
			Owner:      "owner",
			Revision:   1}},
		Runs: []InitialRun{{ID: RunID{"root",
			1},
			ProgramID: rid,
			Projection: Projection{Status: "open",
				Data: map[string]Value{}}}}}
	c := compositionProvision{Version: compositionProtocolVersion,
		Roots: []compositionRoot{{Run: RunID{"root",
			1},
			Members: []RunID{{"root",
				1}},
			Templates: []string{template.ID},
			Limits: compositionLimits{MaxInvocations: 4,
				MaxCreatedRuns: 8,
				MaxDepth:       3,
				MaxEffects:     8,
				MaxInFlight:    1}}},
		Templates: []compositionTemplate{template}}
	return p, c
}
func invocationStore(t *testing.T, outcome string, replies int, change func(*Provision, *compositionProvision)) *Store {
	t.Helper()
	p, c := invocationFixture(t, outcome, replies)
	if change != nil {
		change(&p, &c)
	}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/invocation.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func invokeOnce(t *testing.T, s *Store) compositionAdmission {
	t.Helper()
	r, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AdmitInvocation(t.Context(), r.Projection.Data["pending"].Text)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func TestInvocationAdmissionAtomicFreshAndPermanent(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	if _, err := s.AdmitInvocation(t.Context(), identity("root", 1, 1, 0)); err == nil {
		t.Fatal("uncommitted creation")
	}
	a := invokeOnce(t, s)
	if !a.Accepted || len(a.Members) != 2 {
		t.Fatal(a)
	}
	again, err := s.AdmitInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || !reflect.DeepEqual(a, again) {
		t.Fatal(again, err)
	}
	b := invokeOnce(t, s)
	if !b.Accepted || a.Invocation.ID == b.Invocation.ID {
		t.Fatal(b)
	}
	for _, run := range b.Members {
		r, err := s.Snapshot(t.Context(), run)
		if err != nil || r.Sequence != 0 || r.ID.Generation != 1 || r.Projection.Status != "open" {
			t.Fatal(r, err)
		}
	}
	var n int
	if err = s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM composition_members").Scan(&n); err != nil || n != 4 {
		t.Fatal(n, err)
	}
	var raw []byte
	if err = s.conn.QueryRowContext(t.Context(),
		"SELECT descriptor FROM composition_requests WHERE invocation_id=?",
		a.Invocation.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var observation Observation
	if err = json.Unmarshal(raw, &observation); err != nil {
		t.Fatal(err)
	}
	if observation.Operation != a.Invocation.OperationID || observation.Source != "root" || observation.Run != compositionMemberID(a.Invocation.ID,
		"entry") {
		t.Fatal(observation)
	}
}
func TestInvocationAdmissionDenials(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		change       func(*Provision, *compositionProvision)
	}{
		{"authority", "template_denied", func(_ *Provision, c *compositionProvision) { c.Roots[0].Templates = nil }},
		{"capability", "capability_denied", func(p *Provision, c *compositionProvision) {
			template := c.Templates[0]
			template.Capabilities = []string{"restricted"}
			replaceInvocationTemplate(t, p, c, template)
		}},
		{"depth", "depth_limit", func(_ *Provision, c *compositionProvision) { c.Roots[0].Limits.MaxDepth = 0 }},
		{"count", "invocation_limit", func(_ *Provision, c *compositionProvision) { c.Roots[0].Limits.MaxInvocations = 0 }},
		{"runs", "created_run_limit", func(_ *Provision, c *compositionProvision) { c.Roots[0].Limits.MaxCreatedRuns = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := invocationStore(t, "succeeded", 1, tc.change)
			a := invokeOnce(t, s)
			if a.Accepted || a.Reason != tc.reason || len(a.Members) != 0 {
				t.Fatal(a)
			}
			again, err := s.AdmitInvocation(t.Context(), a.Invocation.OperationID)
			if err != nil || !reflect.DeepEqual(a, again) {
				t.Fatal(again, err)
			}
			var n int
			s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM composition_members").Scan(&n)
			if n != 0 {
				t.Fatal(n)
			}
			prepared, err := s.PrepareInvocation(t.Context(), a.Invocation.OperationID)
			if err != nil || !prepared.Ready || prepared.Outcome.Status != "rejected" || prepared.Outcome.ResponsePresent {
				t.Fatal(prepared, err)
			}
		})
	}
}
func TestInvocationAdmissionRollback(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	_, err := s.conn.ExecContext(t.Context(),
		"CREATE TRIGGER fail_members BEFORE INSERT ON composition_members WHEN NEW.member='helper' BEGIN SELECT RAISE(ABORT,'injected'); END")
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AdmitInvocation(t.Context(), r.Projection.Data["pending"].Text); err == nil {
		t.Fatal("missing injected failure")
	}
	for _, table := range []string{"composition_members",
		"composition_invocations",
		"composition_admission_decisions",
		"composition_charges",
		"composition_requests"} {
		var n int
		if err = s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil || n != 0 {
			t.Fatal(table, n, err)
		}
	}
	var n int
	s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM runs").Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestInvocationReplySurvivesFailureAndLostAcknowledgment(t *testing.T) {
	s := invocationStore(t, "failed", 1, nil)
	a := invokeOnce(t, s)
	entry := compositionMemberID(a.Invocation.ID, "entry")
	request := compositionIdentity("alan/invocation-request/v1", a.Invocation.ID)
	if _, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	reply := identity(entry.Subject, 1, 2, 0)
	if _, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PrepareInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || pending.Ready || len(pending.Freeze.ReplyIntents) != 1 {
		t.Fatal(pending, err)
	}
	delivered, err := s.DeliverMessage(t.Context(), reply, "first")
	if err != nil || !delivered.Accepted {
		t.Fatal(delivered, err)
	}
	if _, err = s.DeliverMessage(t.Context(), reply, strings.Repeat("x", DefaultLimits().MaxObservationBytes)); !errors.Is(err, ErrLimit) {
		t.Fatal("oversized reply transport accepted", err)
	}
	duplicate, err := s.DeliverMessage(t.Context(), reply, "lost-ack-retry")
	if err != nil || !duplicate.Accepted {
		t.Fatal(duplicate, err)
	}
	ready, err := s.PrepareInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || !ready.Ready || ready.Outcome.Status != "failed" || !ready.Outcome.ResponsePresent || ready.Outcome.Response["result"].Text != "invalid_result" {
		t.Fatal(ready, err)
	}
	if err = s.AcknowledgeMessage(t.Context(), reply, "lost-ack-retry"); err != nil {
		t.Fatal(err)
	}
}
func mustSnapshot(t *testing.T, s *Store, id RunID) RunState {
	t.Helper()
	r, err := s.Snapshot(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestInvocationConflictingRepliesAreProtocolFailure(t *testing.T) {
	s := invocationStore(t, "succeeded", 2, nil)
	a := invokeOnce(t, s)
	entry := compositionMemberID(a.Invocation.ID, "entry")
	helper := compositionMemberID(a.Invocation.ID, "helper")
	if _, err := s.Commit(t.Context(),
		mustSnapshot(t,
			s,
			entry).Authority(),
		compositionIdentity("alan/invocation-request/v1",
			a.Invocation.ID)); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Commit(t.Context(), mustSnapshot(t, s, helper).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	for _, seq := range []int{2, 3} {
		if _, err := s.DeliverMessage(t.Context(), identity(entry.Subject, 1, seq, 0), identity("delivery", seq)); err != nil {
			t.Fatal(err)
		}
	}
	prepared, err := s.PrepareInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || !prepared.Ready || prepared.Outcome.Status != "failed" || prepared.Outcome.Reason != "conflicting_replies" {
		t.Fatal(prepared, err)
	}
}

func addInvocationController(t *testing.T, p *Provision, operations ...string) {
	code := []Instruction{}
	seen := map[string]bool{}
	for _, op := range operations {
		code = append(code,
			Instruction{Op: "emit",
				Dst: "pending",
				Effect: &Effect{Kind: "lifecycle",
					Target:     Lit(Text("root")),
					Generation: Lit(Int(1)),
					Obligation: Lit(Text("control")),
					Payload: map[string]Operand{"operation": Lit(Text(op)),
						"revision": Lit(Int(1))}}})
		if !seen[op] {
			p.Permissions = append(p.Permissions, Permission{Kind: "lifecycle", Source: "controller", Target: "root", Operation: op, Revision: 1})
			seen[op] = true
		}
	}
	code = append(code, Instruction{Op: "end", Outcome: "succeeded"})
	addProgram(t, p, "controller", code, map[string]Value{})
}
func TestInvocationRootClosureSerializesWithAdmission(t *testing.T) {
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			s := invocationStore(t, "succeeded", 1, func(p *Provision, _ *compositionProvision) { addInvocationController(t, p, "CancelRun") })
			r, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			id := r.Projection.Data["pending"].Text
			auth, err := s.AuthorizeIntent(t.Context(), id)
			if err != nil || !auth.Allowed {
				t.Fatal(auth, err)
			}
			var admitted compositionAdmission
			if first {
				admitted, err = s.AdmitInvocation(t.Context(), id)
				if err != nil || !admitted.Accepted {
					t.Fatal(admitted, err)
				}
			}
			closed := authorizedDecision(t, s, committedRequest(t, s, "controller"))
			if !closed.Accepted {
				t.Fatal(closed)
			}
			a, err := s.AdmitInvocation(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			if first {
				if !reflect.DeepEqual(a, admitted) {
					t.Fatal(a)
				}
			} else if a.Accepted || a.Reason != "source_closed" {
				t.Fatal(a)
			}
			again, err := s.AuthorizeIntent(t.Context(), id)
			if err != nil || !reflect.DeepEqual(auth, again) {
				t.Fatal("closure rewrote source authorization", again, err)
			}
		})
	}
}
func TestInvocationRetryIsDurableUnsupported(t *testing.T) {
	s := invocationStore(t,
		"succeeded",
		1,
		func(p *Provision,
			_ *compositionProvision) {
			addInvocationController(t,
				p,
				"CancelRun",
				"Retry")
		})
	closed := authorizedDecision(t, s, committedRequest(t, s, "controller"))
	if !closed.Accepted {
		t.Fatal(closed)
	}
	request := committedRequest(t, s, "controller")
	result := authorizedDecision(t, s, request)
	if result.Accepted || result.Reason != "invocation_retry_unsupported" {
		t.Fatal(result)
	}
	again, err := s.DecideLifecycle(t.Context(), request)
	if err != nil || !reflect.DeepEqual(result, again) {
		t.Fatal(again, err)
	}
}
func TestInvocationPublishesAuthenticatedAbsentResponse(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	c.Roots[0].Limits.MaxDepth = 0
	var root Program
	json.Unmarshal(p.Programs[len(p.Programs)-1].Bytes, &root)
	declaration := root.Author.Invocations["child"]
	record, _ := InvocationOutcomeRecord(declaration.Output.Record).Retain()
	root.Code[1] = Instruction{Op: "receive",
		Match: &Match{Kind: "invocation",
			Provenance: "runtime-invocation",
			Source:     Lit(Text(declaration.TemplateID)),
			Generation: Lit(Int(1)),
			Obligation: Lit(Text("child")),
			Operation:  Ref("pending")},
		Bind: map[string]string{"status": "status",
			"reason":           "reason",
			"response_present": "present",
			"response.result":  "result"}}
	root.Author.Receives = map[int]recordschema.Retained{1: record}
	raw, id, err := EncodeProgram(root)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[len(p.Programs)-1] = Artifact{ID: id, Semantics: root.Semantics, Bytes: raw}
	p.Runs[0].ProgramID = id
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/absent.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := invokeOnce(t, s)
	delivered, err := s.SettleInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || !delivered.Accepted {
		t.Fatal(delivered, err)
	}
	duplicate, err := s.SettleInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || duplicate.ObservationID != delivered.ObservationID {
		t.Fatal(duplicate, err)
	}
	r, err := s.Commit(t.Context(), current(t, s, "root").Authority(), delivered.ObservationID)
	if err != nil || r.Projection.Data["status"].Text != "rejected" || r.Projection.Data["present"].Flag {
		t.Fatal(r, err)
	}
	if _, exists := r.Projection.Data["result"]; exists {
		t.Fatal("fabricated enum response")
	}
}
func TestInvocationProfileCannotLeakIntoLegacyStore(t *testing.T) {
	p, _ := invocationFixture(t, "succeeded", 1)
	if _, err := Create(t.Context(), t.TempDir()+"/legacy.sqlite", p, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatal("schema3 accepted invocation", err)
	}
}
func TestInvocationCatalogueRejectsSchemaMismatch(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	var root Program
	json.Unmarshal(p.Programs[len(p.Programs)-1].Bytes, &root)
	d := root.Author.Invocations["child"]
	d.Output, _ = (recordschema.Record{"different": {Kind: "int"}}).Retain()
	root.Author.Invocations["child"] = d
	raw, id, err := EncodeProgram(root)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[len(p.Programs)-1] = Artifact{ID: id, Semantics: root.Semantics, Bytes: raw}
	p.Runs[0].ProgramID = id
	if _, err = createInvocationStore(t.Context(), t.TempDir()+"/invalid.sqlite", p, c, DefaultLimits()); err == nil {
		t.Fatal("mismatched public response declaration accepted")
	}
}

func nestedInvocationFixture(t *testing.T, helperFails bool) (Provision, compositionProvision) {
	t.Helper()
	p, c := invocationFixture(t, "succeeded", 1)
	leaf := c.Templates[0]
	outer := leaf
	outer.ID = ""
	outer.Members = map[string]compositionTemplateMember{}
	for name, m := range leaf.Members {
		outer.Members[name] = m
	}
	outer.Templates = []string{leaf.ID}
	var entry Program
	for _, a := range p.Programs {
		if a.ID == leaf.Members["entry"].ProgramID {
			json.Unmarshal(a.Bytes, &entry)
		}
	}
	empty, _ := recordschema.Record{}.Retain()
	output := entry.Author.Ports["public"].Send
	entry.Semantics = CollectionSemanticVersion
	entry.Author.Compiler = CollectionCompilerVersion
	entry.Author.Invocations = map[string]InvocationDeclaration{"nested": {TemplateID: leaf.ID, Input: empty, Output: output}}
	invoke := Instruction{Op: "emit",
		Dst: "pending",
		Effect: &Effect{Kind: "invocation",
			Target:     Lit(Text(leaf.ID)),
			Generation: Lit(Int(1)),
			Obligation: Lit(Text("nested")),
			Payload:    map[string]Operand{}}}
	entry.Code = append(entry.Code[:1], append([]Instruction{invoke}, entry.Code[1:]...)...)
	entry.Author.Positions = make([]string, len(entry.Code))
	raw, id, err := EncodeProgram(entry)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: entry.Semantics, Bytes: raw})
	outer.Members["entry"] = compositionTemplateMember{ProgramID: id, Initial: Projection{Status: "open", Data: map[string]Value{}}}
	if helperFails {
		var helper Program
		for _, a := range p.Programs {
			if a.ID == leaf.Members["helper"].ProgramID {
				json.Unmarshal(a.Bytes, &helper)
			}
		}
		helper.Code[0].Outcome = "failed"
		raw, id, err = EncodeProgram(helper)
		if err != nil {
			t.Fatal(err)
		}
		p.Programs = append(p.Programs, Artifact{ID: id, Semantics: helper.Semantics, Bytes: raw})
		outer.Members["helper"] = compositionTemplateMember{ProgramID: id, Initial: Projection{Status: "open", Data: map[string]Value{}}}
	}
	outer, err = RetainTemplate(outer, p.Programs)
	if err != nil {
		t.Fatal(err)
	}
	c.Templates = append(c.Templates, outer)
	c.Roots[0].Templates = append(c.Roots[0].Templates, outer.ID)
	var root Program
	for _, a := range p.Programs {
		if a.ID == p.Runs[0].ProgramID {
			json.Unmarshal(a.Bytes, &root)
		}
	}
	d := root.Author.Invocations["child"]
	d.TemplateID = outer.ID
	root.Author.Invocations["child"] = d
	for i := range 2 {
		root.Code[i].Effect.Target = Lit(Text(outer.ID))
	}
	raw, id, err = EncodeProgram(root)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: root.Semantics, Bytes: raw})
	p.Runs[0].ProgramID = id
	return p, c
}
func TestInvocationAncestorClosurePreservesEarlierAuthorization(t *testing.T) {
	for _, admitFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(admitFirst), func(t *testing.T) {
			p, c := nestedInvocationFixture(t, true)
			s, err := createInvocationStore(t.Context(), t.TempDir()+"/nested.sqlite", p, c, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			outer := invokeOnce(t, s)
			entry := compositionMemberID(outer.Invocation.ID, "entry")
			helper := compositionMemberID(outer.Invocation.ID, "helper")
			if _, err = s.Commit(t.Context(),
				mustSnapshot(t,
					s,
					entry).Authority(),
				compositionIdentity("alan/invocation-request/v1",
					outer.Invocation.ID)); err != nil {
				t.Fatal(err)
			}
			r, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			id := r.Projection.Data["pending"].Text
			auth, err := s.AuthorizeIntent(t.Context(), id)
			if err != nil || !auth.Allowed {
				t.Fatal(auth, err)
			}
			if admitFirst {
				a, e := s.AdmitInvocation(t.Context(), id)
				if e != nil || !a.Accepted || a.Invocation.Depth != 2 {
					t.Fatal(a, e)
				}
			}
			if _, err = s.Commit(t.Context(), mustSnapshot(t, s, helper).Authority(), ""); err != nil {
				t.Fatal(err)
			}
			a, err := s.AdmitInvocation(t.Context(), id)
			if err != nil || a.Accepted != admitFirst || !admitFirst && a.Reason != "scope_closed" {
				t.Fatal(a, err)
			}
			again, err := s.AuthorizeIntent(t.Context(), id)
			if err != nil || !reflect.DeepEqual(auth, again) {
				t.Fatal(again, err)
			}
			prepared, err := s.PrepareInvocation(t.Context(), outer.Invocation.OperationID)
			if err != nil || !prepared.Ready || prepared.Outcome.Status != "failed" || prepared.Outcome.ResponsePresent {
				t.Fatal(prepared, err)
			}
		})
	}
}
func TestInvocationSuccessWaitsForCommittedUnadmittedChild(t *testing.T) {
	p, c := nestedInvocationFixture(t, false)
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/join.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	outer := invokeOnce(t, s)
	entry := compositionMemberID(outer.Invocation.ID, "entry")
	helper := compositionMemberID(outer.Invocation.ID, "helper")
	if _, err = s.Commit(t.Context(),
		mustSnapshot(t,
			s,
			entry).Authority(),
		compositionIdentity("alan/invocation-request/v1",
			outer.Invocation.ID)); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, helper).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DeliverMessage(t.Context(), identity(entry.Subject, 1, 3, 0), "reply"); err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareInvocation(t.Context(), outer.Invocation.OperationID)
	if err != nil || prepared.Ready || prepared.Pending != "invocation_admission" || len(prepared.Freeze.InvocationIntents) != 1 {
		t.Fatal(prepared, err)
	}
	if result, err := s.SettleInvocation(t.Context(), outer.Invocation.OperationID); err != nil || result.Accepted {
		t.Fatal(result, err)
	}
	denied, err := s.AdmitInvocation(t.Context(), identity(entry.Subject, 1, 2, 0))
	if err != nil || denied.Accepted {
		t.Fatal(denied, err)
	}
	prepared, err = s.PrepareInvocation(t.Context(), outer.Invocation.OperationID)
	if err != nil || !prepared.Ready || prepared.Outcome.Status != "succeeded" {
		t.Fatal(prepared, err)
	}
}
func TestInvocationPrivateReopenFailsClosedWithValidDynamicRows(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	path := t.TempDir() + "/reopen.sqlite"
	s, err := createInvocationStore(t.Context(), path, p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	a := invokeOnce(t, s)
	if !a.Accepted {
		t.Fatal(a)
	}
	// Real admitted rows satisfy FK constraints; the protocol gate is reached
	// before unsupported ordinary dynamic-origin replay. No fabricated FK failure.
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatal(err)
	}
	if _, err = openCompositionStore(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatal(err)
	}
}

func replaceInvocationTemplate(t *testing.T, p *Provision, c *compositionProvision, template compositionTemplate) {
	t.Helper()
	template.ID = ""
	retained, err := RetainTemplate(template, p.Programs)
	if err != nil {
		t.Fatal(err)
	}
	c.Templates = []compositionTemplate{retained}
	c.Roots[0].Templates = []string{retained.ID}
	var root Program
	for _, a := range p.Programs {
		if a.ID == p.Runs[0].ProgramID {
			json.Unmarshal(a.Bytes, &root)
		}
	}
	d := root.Author.Invocations["child"]
	d.TemplateID = retained.ID
	root.Author.Invocations["child"] = d
	for i := range 2 {
		root.Code[i].Effect.Target = Lit(Text(retained.ID))
	}
	raw, id, err := EncodeProgram(root)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: root.Semantics, Bytes: raw})
	p.Runs[0].ProgramID = id
}
func TestInvocationReplyExceptionDoesNotAuthorizeOtherEffects(t *testing.T) {
	p, c := invocationFixture(t, "failed", 1)
	template := c.Templates[0]
	empty, _ := recordschema.Record{}.Retain()
	var entry Program
	for _, a := range p.Programs {
		if a.ID == template.Members["entry"].ProgramID {
			json.Unmarshal(a.Bytes, &entry)
		}
	}
	entry.Author.Effects = map[string]EffectDeclaration{"work": {Capability: "cap", Input: empty, Output: empty}}
	entry.Author.Ports["internal"] = PortDeclaration{Peer: "helper", Obligation: "internal", Send: empty, Receive: empty, SendUsed: true}
	entry.Code = append(entry.Code[:2],
		Instruction{Op: "emit",
			Effect: &Effect{Kind: "adapter",
				Target:     Lit(Text("cap")),
				Generation: Lit(Int(1)),
				Obligation: Lit(Text("work")),
				Payload:    map[string]Operand{}}},
		Instruction{Op: "emit",
			Effect: &Effect{Kind: "message",
				Target:     Lit(Text("helper")),
				Generation: Lit(Int(1)),
				Obligation: Lit(Text("internal")),
				Payload:    map[string]Operand{}}},
		Instruction{Op: "end",
			Outcome: "failed"})
	entry.Author.Positions = make([]string, len(entry.Code))
	raw, id, err := EncodeProgram(entry)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: entry.Semantics, Bytes: raw})
	template.Members["entry"] = compositionTemplateMember{ProgramID: id, Initial: Projection{Status: "open", Data: map[string]Value{}}}
	helper := Program{Semantics: AuthorSemanticVersion,
		Code: []Instruction{{Op: "receive",
			Match: &Match{Kind: "message",
				Provenance: "ordinary-message",
				Source:     Lit(Text("entry")),
				Generation: Lit(Int(1)),
				Obligation: Lit(Text("internal")),
				Operation:  Lit(Text(""))}},
			{Op: "end",
				Outcome: "succeeded"}},
		Author: &ProgramMetadata{Compiler: AuthorCompilerVersion,
			Inputs:  empty,
			Outputs: empty,
			Positions: []string{"receive",
				"end"},
			Receives: map[int]recordschema.Retained{0: empty},
			Ports: map[string]PortDeclaration{"internal": {Peer: "entry",
				Obligation:  "internal",
				Send:        empty,
				Receive:     empty,
				ReceiveUsed: true}}}}
	raw, id, err = EncodeProgram(helper)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: helper.Semantics, Bytes: raw})
	template.Members["helper"] = compositionTemplateMember{ProgramID: id, Initial: Projection{Status: "open", Data: map[string]Value{}}}
	template.Capabilities = []string{"cap"}
	c.Roots[0].Capabilities = []string{"cap"}
	replaceInvocationTemplate(t, &p, &c, template)
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/exception.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := invokeOnce(t, s)
	run := compositionMemberID(a.Invocation.ID, "entry")
	if _, err = s.Commit(t.Context(),
		mustSnapshot(t,
			s,
			run).Authority(),
		compositionIdentity("alan/invocation-request/v1",
			a.Invocation.ID)); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if _, err = s.Commit(t.Context(), mustSnapshot(t, s, run).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	for _, seq := range []int{2, 3, 4} {
		auth, e := s.AuthorizeIntent(t.Context(), identity(run.Subject, 1, seq, 0))
		if e != nil || auth.Allowed != (seq == 2) {
			t.Fatal(seq, auth, e)
		}
	}
	prepared, err := s.PrepareInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || len(prepared.Freeze.ExternalIntents) != 1 || prepared.Freeze.ExternalIntents[0] != identity(run.Subject, 1, 3, 0) {
		t.Fatal(prepared, err)
	}
	delivered, err := s.DeliverMessage(t.Context(), identity(run.Subject, 1, 4, 0), "internal")
	if err != nil || delivered.Accepted || delivered.Reason != "source_denied:source_closed" {
		t.Fatal(delivered, err)
	}
}
func TestInvocationFailureFreezeIsAtomicAndIgnoresLateReply(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 2)
	template := c.Templates[0]
	var helper Program
	for _, a := range p.Programs {
		if a.ID == template.Members["helper"].ProgramID {
			json.Unmarshal(a.Bytes, &helper)
		}
	}
	helper.Code[0].Outcome = "failed"
	raw, id, err := EncodeProgram(helper)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: helper.Semantics, Bytes: raw})
	template.Members["helper"] = compositionTemplateMember{ProgramID: id, Initial: Projection{Status: "open", Data: map[string]Value{}}}
	replaceInvocationTemplate(t, &p, &c, template)
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/freeze.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := invokeOnce(t, s)
	entry := compositionMemberID(a.Invocation.ID, "entry")
	helperRun := compositionMemberID(a.Invocation.ID, "helper")
	if _, err = s.Commit(t.Context(),
		mustSnapshot(t,
			s,
			entry).Authority(),
		compositionIdentity("alan/invocation-request/v1",
			a.Invocation.ID)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(),
		"CREATE TRIGGER freeze_failure BEFORE INSERT ON composition_freezes BEGIN SELECT RAISE(ABORT,'freeze injection'); END"); err != nil {
		t.Fatal(err)
	}
	before := mustSnapshot(t, s, helperRun)
	if _, err = s.Commit(t.Context(), before.Authority(), ""); err == nil {
		t.Fatal("freeze injection missed")
	}
	if after := mustSnapshot(t, s, helperRun); !reflect.DeepEqual(before, after) {
		t.Fatal("partial terminal commit")
	}
	if _, err = s.conn.ExecContext(t.Context(), "DROP TRIGGER freeze_failure"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), before.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	// The later reply is delivered first, but cannot enter the frozen failure set.
	for _, seq := range []int{3, 2} {
		if _, err = s.DeliverMessage(t.Context(), identity(entry.Subject, 1, seq, 0), identity("reply", seq)); err != nil {
			t.Fatal(err)
		}
	}
	prepared, err := s.PrepareInvocation(t.Context(), a.Invocation.OperationID)
	if err != nil || !prepared.Ready || prepared.Outcome.Status != "failed" || !prepared.Outcome.ResponsePresent || !reflect.DeepEqual(prepared.Freeze.ReplyIntents,
		[]string{identity(entry.Subject,
			1,
			2,
			0)}) {
		t.Fatal(prepared, err)
	}
}

func TestInvocationInstructionsRequireVersionedDeclarations(t *testing.T) {
	for _, op := range []string{"emit", "receive"} {
		p := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: op}, {Op: "end", Outcome: "succeeded"}}}
		if op == "emit" {
			p.Code[0].Effect = &Effect{Kind: "invocation",
				Target:     Lit(Text("template")),
				Generation: Lit(Int(1)),
				Obligation: Lit(Text("child")),
				Payload:    map[string]Operand{}}
		} else {
			p.Code[0].Match = &Match{Kind: "invocation",
				Provenance: "runtime-invocation",
				Source:     Lit(Text("template")),
				Generation: Lit(Int(1)),
				Obligation: Lit(Text("child")),
				Operation:  Lit(Text("operation"))}
		}
		if _, _, err := EncodeProgram(p); !errors.Is(err, ErrVersion) {
			t.Fatal("legacy instruction accepted", op, err)
		}
	}
}
func TestInvocationPendingIntentPrivateReopenGate(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	path := t.TempDir() + "/pending.sqlite"
	s, err := createInvocationStore(t.Context(), path, p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "root").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if opened, err := openCompositionStore(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		if opened != nil {
			opened.Close()
		}
		t.Fatal("unfinished protocol reopened", err)
	}
}

func TestInvocationAdmissionChargesDoNotResetAtNestedBoundary(t *testing.T) {
	p, c := nestedInvocationFixture(t, false)
	c.Roots[0].Limits.MaxInvocations = 2
	c.Roots[0].Limits.MaxCreatedRuns = 4
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/charges.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	outer := invokeOnce(t, s)
	entry := compositionMemberID(outer.Invocation.ID, "entry")
	if _, err = s.Commit(t.Context(),
		mustSnapshot(t,
			s,
			entry).Authority(),
		compositionIdentity("alan/invocation-request/v1",
			outer.Invocation.ID)); err != nil {
		t.Fatal(err)
	}
	r, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	inner, err := s.AdmitInvocation(t.Context(), r.Projection.Data["pending"].Text)
	if err != nil || !inner.Accepted || inner.Invocation.RootID != outer.Invocation.RootID || inner.Invocation.Depth != 2 {
		t.Fatal(inner, err)
	}
	rejected := invokeOnce(t, s)
	if rejected.Accepted || rejected.Reason != "invocation_limit" {
		t.Fatal(rejected)
	}
	var admissions, runs int64
	if err = s.conn.QueryRowContext(t.Context(),
		"SELECT sum(json_extract(descriptor,'$.Invocations')),sum(json_extract(descriptor,'$.CreatedRuns')) FROM composition_charges").Scan(&admissions,
		&runs); err != nil || admissions != 2 || runs != 4 {
		t.Fatal(admissions, runs, err)
	}
	if _, err = s.AdmitInvocation(t.Context(), inner.Invocation.OperationID); err != nil {
		t.Fatal(err)
	}
	if count(t, s, "composition_charges") != 2 || count(t, s, "runs") != 5 {
		t.Fatal("duplicate charge or fixed cost included")
	}
}
