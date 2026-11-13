//go:build integration

package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/JakeFAU/alan/effects"
	recordschema "github.com/JakeFAU/alan/schema"
	"github.com/mattn/go-sqlite3"
)

// Simulate a fully masked profile before its first approval. The persisted
// composition_facts constraint is independent of both mutable version headers
// and the profile bytes. A format-9 database can never have written this DDL.
func TestContinuationMaskedBeforeFirstApproval(t *testing.T) {
	s, path, _ := gateStore(t, true)
	ctx := t.Context()
	if _, err := s.conn.ExecContext(ctx, `PRAGMA writable_schema=ON;
		UPDATE sqlite_schema SET sql=replace(sql,'''external_result''','''external_result'',''continuation''')
		WHERE type='table' AND name='composition_facts';
		PRAGMA writable_schema=OFF`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	l := DefaultLimits()
	_, raw := Inspect(ctx, path, l)
	_, typed := InspectExecution(ctx, path, l)
	_, replay := Replay(ctx, path, l)
	owner, open := Open(ctx, path, l)
	if owner != nil {
		owner.Close()
	}
	for _, err := range []error{raw, typed, replay, open} {
		if !errors.Is(err, ErrVersion) {
			t.Fatalf("masked format accepted: %v", err)
		}
	}
}

// Upgrade an actual protected Store by submitting a conflicting result to an
// existing paid operation. Both reads and their original decision must survive.
func TestContinuationConflictPromotion(t *testing.T) {
	p := Provision{}
	addAdapterProgram(t, &p, "first", true)
	addAdapterProgram(t, &p, "second", true)
	var program Program
	if err := decode(p.Programs[1].Bytes, &program); err != nil {
		t.Fatal(err)
	}
	empty, _ := recordschema.Record{}.Retain()
	field := recordschema.Field{Kind: "string", MaxBytes: 4096}
	input, err := (recordschema.Record{"previous_operation_id": field, "purpose": field, "prepared_digest": field, "result_digest": field}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	program.Author.Ports = map[string]PortDeclaration{"approval": {Local: true, Peer: "local", Obligation: "approve", Receive: input, Send: empty, ReceiveUsed: true}}
	program.Author.Receives = map[int]recordschema.Retained{0: input, 2: program.Author.Receives[1]}
	program.Author.Positions = append([]string{"approval"}, program.Author.Positions...)
	program.Code = append([]Instruction{{Op: "receive", Match: &Match{Kind: "input", Provenance: "local-input", Source: Lit(Text("local")), Generation: Lit(Int(1)), Obligation: Lit(Text("approve")), Operation: Lit(Text(""))}}}, program.Code...)
	raw, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[1] = Artifact{ID: id, Semantics: program.Semantics, Bytes: raw}
	p.Runs[1].ProgramID = id
	p.Permissions = append(p.Permissions, Permission{Kind: "input", Source: "local", Target: "second", Operation: "input", Obligation: "approve"})
	slot := ContinuationSlot{Predecessor: p.Runs[0].ID, Target: p.Runs[1].ID, Effect: "call", Obligation: "approve", ApprovalID: "second-approval"}
	p.Composition = &CompositionProvision{RecordDispatchAdmissions: true, Continuations: []ContinuationSlot{slot}}
	for _, r := range p.Runs {
		p.Composition.Roots = append(p.Composition.Roots, ResourceRoot{Run: r.ID, Members: []RunID{r.ID}, Capabilities: []string{"model"}, Limits: ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
	}
	s, path := messageStore(t, p)
	ctx, l := t.Context(), DefaultLimits()
	options := RunOptions{MaxSteps: 30, MaxDispatch: 30, Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}}
	if _, err := s.Run(ctx, options); err != nil {
		t.Fatal(err)
	}
	cut, err := InspectExecution(ctx, path, l)
	if err != nil {
		t.Fatal(err)
	}
	basis, err := cut.ContinuationBasis(slot.Target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ApproveContinuation(ctx, basis, "fresh"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(ctx, options); err != nil {
		t.Fatal(err)
	}
	attempt, err := loadAdapterAttempt(ctx, s.conn, basis.PredecessorID)
	if err != nil {
		t.Fatal(err)
	}
	loser := effects.Candidate{ID: basis.PredecessorID, Key: attempt.Prepared.Request.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text("other")}}
	if _, err = s.recordAdapterResult(ctx, attempt, loser); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	cut, err = InspectExecution(ctx, path, l)
	if err != nil {
		t.Fatal(err)
	}
	old, found := cut.Effect(basis.PredecessorID)
	v, ok := cut.Continuation(slot.Target)
	if cut.StorageVersion != 12 || !cut.Features.ConflictJournal || !found || len(old.AdditionalReceipts) != 1 || old.AdapterResult.Digest != basis.ResultDigest || !ok || v.Approval.ContinuationBasis != basis || v.SelectedAt == nil || v.SuccessorOperationID == "" {
		t.Fatalf("lost approval or original result during promotion: %+v", v)
	}
	if _, err := Inspect(ctx, path, l); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(ctx, path, l); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(ctx, path, l)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	got, err := owner.ContinuationBasis(ctx, slot.Target)
	if err != nil || got != basis {
		t.Fatalf("promoted owner's basis = %+v, %v", got, err)
	}
	t.Run("canceled inside read", func(t *testing.T) {
		canceled, cancel := context.WithCancel(t.Context())
		defer cancel()
		if err := owner.conn.Raw(func(driver any) error {
			driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, _ string, _ string) int {
				if op == sqlite3.SQLITE_READ && table == "composition_facts" {
					cancel()
				}
				return sqlite3.SQLITE_OK
			})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		got, err := owner.ContinuationBasis(canceled, slot.Target)
		if reset := owner.conn.Raw(func(driver any) error {
			driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil)
			return nil
		}); reset != nil {
			t.Fatal(reset)
		}
		if got != (ContinuationBasis{}) || !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled read = %+v, %v", got, err)
		}
		got, err = owner.ContinuationBasis(t.Context(), slot.Target)
		if err != nil || got != basis {
			t.Fatalf("next owner read = %+v, %v", got, err)
		}
	})
}
