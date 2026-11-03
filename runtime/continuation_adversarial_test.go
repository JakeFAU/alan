//go:build integration

package runtime_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Independently authored client. The adapter counts physical entries and
// preserves the application-selected wire data across namespace-bound keys.
type continuationMeter struct {
	endpoint string
	calls    atomic.Int64
}

func (m *continuationMeter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "sample/v1", Endpoint: m.endpoint,
		Wire: r.Payload["station"].Text, MaxResponseBytes: 256}, nil
}

func (m *continuationMeter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.calls.Add(1)
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated",
		Payload: map[string]schema.Value{"sample": schema.Text("present")}}
}

var (
	meterOriginal = runtime.RunID{Subject: "original", Generation: 1}
	meterNext     = runtime.RunID{Subject: "follower", Generation: 1}
)

const meterApproval = "approve-sample-2"

func meterProgram(t *testing.T, protected bool, station string, reads int) author.Artifact {
	t.Helper()
	d := author.Definition{Name: "sample", Effects: map[string]author.Effect{"read": {
		Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}},
		Output: schema.Record{"sample": {Kind: "string", MaxBytes: 16}},
	}}}
	if protected {
		record := schema.Record{
			"previous_operation_id": {Kind: "string", MaxBytes: 4096},
			"purpose":               {Kind: "string", MaxBytes: 64}, "prepared_digest": {Kind: "string", MaxBytes: 64},
			"result_digest": {Kind: "string", MaxBytes: 64},
		}
		d.Ports = map[string]author.Port{"consent": {Local: true, Peer: "local", Obligation: "approve", Receive: record}}
		d.Steps = []author.Step{author.Receive("consent", nil)}
	}
	for range reads {
		d.Steps = append(d.Steps, author.Request("handle", "read", map[string]author.Expr{"station": author.String(station)}, nil),
			author.Await("read", "handle", map[string]string{"sample": "sample"}))
	}
	d.Steps = append(d.Steps, author.Success())
	program, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

type meterFixture struct {
	path  string
	owner *runtime.Store
	meter *continuationMeter
}

func newMeterFixture(t *testing.T, nextStation string, configure ...func(*runtime.Provision)) *meterFixture {
	t.Helper()
	app := author.Application{
		meterOriginal.Subject: {Artifact: meterProgram(t, false, "north", 1)},
		meterNext.Subject:     {Artifact: meterProgram(t, true, nextStation, 1)},
	}
	p, err := app.Provision()
	if err != nil {
		t.Fatal(err)
	}
	p.Composition = &runtime.CompositionProvision{RecordDispatchAdmissions: true,
		Continuations: []runtime.ContinuationSlot{{Predecessor: meterOriginal, Target: meterNext,
			Effect: "read", Obligation: "approve", ApprovalID: meterApproval}}}
	for _, r := range []runtime.RunID{meterOriginal, meterNext} {
		p.Composition.Roots = append(p.Composition.Roots, runtime.ResourceRoot{Run: r, Members: []runtime.RunID{r},
			Capabilities: []string{"meter"}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
	}
	for _, edit := range configure {
		edit(&p)
	}
	f := &meterFixture{path: filepath.Join(t.TempDir(), "meter.sqlite"), meter: &continuationMeter{endpoint: "local-meter"}}
	f.owner, err = runtime.Create(t.Context(), f.path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.owner.Close(); err != nil {
			t.Error(err)
		}
	})
	return f
}

func (f *meterFixture) run(ctx context.Context) error {
	_, err := f.owner.Run(ctx, runtime.RunOptions{MaxSteps: 32, MaxDispatch: 16, MaxInFlight: 1,
		Adapters: map[string]effects.Adapter{"meter": f.meter}})
	return err
}

func meterCut(t *testing.T, path string) *runtime.ExecutionSnapshot {
	t.Helper()
	cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return cut
}

func meterBasis(t *testing.T, f *meterFixture) runtime.ContinuationBasis {
	t.Helper()
	basis, err := meterCut(t, f.path).ContinuationBasis(meterNext)
	if err != nil {
		t.Fatal(err)
	}
	return basis
}

func meterSpent(cut *runtime.ExecutionSnapshot) int64 {
	var spent int64
	for _, root := range cut.Roots {
		spent += root.Charges.Effects
	}
	return spent
}

func TestContinuationApprovalBindsOriginalStoreAndAction(t *testing.T) {
	f, other := newMeterFixture(t, "north"), newMeterFixture(t, "north")
	for _, store := range []*meterFixture{f, other} {
		if err := store.run(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	original, foreign := meterBasis(t, f), meterBasis(t, other)
	if original.Namespace == foreign.Namespace || original.PreparedDigest == foreign.PreparedDigest ||
		original.PredecessorID != foreign.PredecessorID || original.ResultDigest == "" || !original.Admitted {
		t.Fatal("independent stores failed to produce a discriminating pair")
	}
	for _, tc := range []struct {
		name   string
		change func(*runtime.ContinuationBasis)
	}{
		{"authentic_foreign_basis", func(b *runtime.ContinuationBasis) { *b = foreign }},
		{"wrong_predecessor", func(b *runtime.ContinuationBasis) { b.PredecessorID += "1" }},
		{"wrong_target", func(b *runtime.ContinuationBasis) { b.Target.Subject = meterOriginal.Subject }},
		{"wrong_generation", func(b *runtime.ContinuationBasis) { b.Target.Generation++ }},
		{"foreign_prepared_request", func(b *runtime.ContinuationBasis) { b.PreparedDigest = foreign.PreparedDigest }},
		{"foreign_result", func(b *runtime.ContinuationBasis) { b.ResultDigest = foreign.ResultDigest }},
		{"hidden_completion", func(b *runtime.ContinuationBasis) { b.ResultDigest = "" }},
		{"hidden_dispatch", func(b *runtime.ContinuationBasis) { b.Admitted = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := original
			tc.change(&b)
			a, err := f.owner.ApproveContinuation(t.Context(), b, "render another sample")
			if err == nil && a.Accepted {
				t.Fatal("accepted unrelated original evidence")
			}
		})
	}
	// Even a completely accurate copy of the public input fields is ordinary
	// ingress and cannot stand in for an actual decision on a protected slot.
	manual := runtime.Input{TransportID: meterApproval, LogicalID: meterApproval, Target: meterNext.Subject,
		Generation: 1, Obligation: "approve", Payload: map[string]schema.Value{
			"previous_operation_id": schema.Text(original.PredecessorID), "purpose": schema.Text("render another sample"),
			"prepared_digest": schema.Text(original.PreparedDigest), "result_digest": schema.Text(original.ResultDigest),
		}}
	a, err := f.owner.AdmitInput(t.Context(), manual)
	if err == nil && a.Accepted {
		t.Fatal("raw ingress authorized a protected effect")
	}
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.meter.calls.Load() != 1 || meterSpent(meterCut(t, f.path)) != 1 {
		t.Fatal("invalid approvals caused expenditure")
	}
	decision, err := f.owner.ApproveContinuation(t.Context(), original, "render another sample")
	if err != nil || !decision.Accepted {
		t.Fatalf("valid continuation after refused inputs: %v, %v", decision, err)
	}
	if err = f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	duplicate, err := f.owner.ApproveContinuation(t.Context(), original, "render another sample")
	if err != nil || duplicate.ObservationID != decision.ObservationID {
		t.Fatalf("duplicate: %v, %v", duplicate, err)
	}
	different, err := f.owner.ApproveContinuation(t.Context(), original, "a different decision")
	if err == nil && different.Accepted {
		t.Fatal("conflicting decision accepted")
	}
	if err = f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	cut := meterCut(t, f.path)
	consent, ok := cut.Observation(decision.ObservationID)
	if !ok || consent.SelectedAt == nil || consent.SelectedAt.Run != meterNext ||
		len(cut.Effects) != 2 || f.meter.calls.Load() != 2 || meterSpent(cut) != 2 {
		t.Fatal("expected one selected approval, two distinct observations and two paid contacts")
	}
}

func TestContinuationChangesCannotSpendAnotherOpportunity(t *testing.T) {
	for _, tc := range []struct{ name, station, endpoint string }{
		{"authorized_parameters_changed", "south", "local-meter"},
		{"registered_provider_changed", "north", "other-meter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMeterFixture(t, tc.station)
			if err := f.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			decision, err := f.owner.ApproveContinuation(t.Context(), meterBasis(t, f), "second sample")
			if err != nil || !decision.Accepted {
				t.Fatalf("approval %v %v", decision, err)
			}
			f.meter.endpoint = tc.endpoint
			if err = f.run(t.Context()); !errors.Is(err, runtime.ErrConflict) {
				t.Fatalf("substituted action: %v", err)
			}
			cut := meterCut(t, f.path)
			if f.meter.calls.Load() != 1 || meterSpent(cut) != 1 {
				t.Fatal("substituted action spent the successor opportunity")
			}
			if _, err := runtime.Replay(t.Context(), f.path, runtime.DefaultLimits()); err != nil {
				t.Fatalf("refusal damaged original history: %v", err)
			}
		})
	}
}

// No result exists at this acknowledged boundary. The same owner subsequently
// retains the original result. The immutable approval states what was true at
// its own transaction even after two successful observations.
func lateResultFixture(t *testing.T) (*meterFixture, runtime.ContinuationBasis) {
	t.Helper()
	f := newMeterFixture(t, "north")
	var basis runtime.ContinuationBasis
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary != "adapter.response" || e.Subject != meterOriginal.Subject {
			return nil
		}
		cut, err := runtime.InspectExecution(ctx, f.path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		basis, err = cut.ContinuationBasis(meterNext)
		if err != nil {
			return err
		}
		if !basis.Admitted || basis.ResultDigest != "" {
			return errors.New("wrong original boundary")
		}
		d, err := f.owner.ApproveContinuation(ctx, basis, "new observation while original outstanding")
		if err == nil && !d.Accepted {
			return errors.New("approval refused")
		}
		return err
	})
	if err := f.run(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	return f, basis
}

func TestContinuationApprovalPrecedesLaterOriginalResult(t *testing.T) {
	f, basis := lateResultFixture(t)
	cut := meterCut(t, f.path)
	v, ok := cut.Continuation(meterNext)
	if !ok || v.Approval.ContinuationBasis != basis || v.SelectedAt == nil || v.SuccessorOperationID == "" {
		t.Fatal("typed original-to-approval-to-successor link is incomplete")
	}
	original, ok := cut.Effect(basis.PredecessorID)
	if !ok || original.AdapterResult == nil || basis.ResultDigest != "" || f.meter.calls.Load() != 2 || meterSpent(cut) != 2 {
		t.Fatal("late original or independent successor lost")
	}
	prior, err := f.owner.ApproveContinuation(t.Context(), basis, "new observation while original outstanding")
	if err != nil || !prior.Accepted {
		t.Fatalf("same historical approval after a later result: %v %v", prior, err)
	}
	if _, err = runtime.Replay(t.Context(), f.path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if err = f.owner.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		s, err := runtime.Open(t.Context(), f.path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContinuationSelectionCanAuthorizeOnlyOneOperation(t *testing.T) {
	f := newMeterFixture(t, "north", func(p *runtime.Provision) {
		twice := meterProgram(t, true, "north", 2)
		p.Programs = append(p.Programs, runtime.Artifact{ID: twice.ID(), Bytes: twice.Bytes(), Semantics: author.SemanticVersion})
		for n := range p.Runs {
			if p.Runs[n].ID == meterNext {
				p.Runs[n].ProgramID = twice.ID()
			}
		}
	})
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	approved, err := f.owner.ApproveContinuation(t.Context(), meterBasis(t, f), "one more")
	if err != nil || !approved.Accepted {
		t.Fatalf("approval: %v %v", approved, err)
	}
	for range 2 {
		if err := f.run(t.Context()); !errors.Is(err, runtime.ErrConflict) {
			t.Fatalf("second request by the same program: %v", err)
		}
		cut := meterCut(t, f.path)
		v, ok := cut.Continuation(meterNext)
		if !ok || v.SuccessorOperationID == "" || len(cut.Effects) != 2 || meterSpent(cut) != 2 || f.meter.calls.Load() != 2 {
			t.Fatal("another operation used the selected approval")
		}
		if err := f.owner.Close(); err != nil {
			t.Fatal(err)
		}
		f.owner, err = runtime.Open(t.Context(), f.path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
	}
}

// The public author language has no lifecycle statement. This test-only
// controller provisions the existing executor instruction used for revocation.
func meterRetirementController(t *testing.T, p *runtime.Provision) {
	t.Helper()
	ctl := runtime.RunID{Subject: "controller", Generation: 1}
	lit := func(s string) executor.Operand { return executor.Lit(schema.Text(s)) }
	one := executor.Lit(schema.Int(1))
	code := []executor.Instruction{
		{Op: "receive", Match: &executor.Match{Kind: "input", Provenance: "local-input", Source: lit("local"), Generation: one, Obligation: lit("stop"), Operation: lit("")}},
		{Op: "emit", Dst: "request", Effect: &executor.Effect{Kind: "lifecycle", Target: lit(meterNext.Subject), Generation: one,
			Obligation: lit("stop"), Payload: map[string]executor.Operand{"operation": lit("RetireSubject"), "revision": one}}},
		{Op: "end", Outcome: "succeeded"},
	}
	b, id, err := executor.EncodeProgram(executor.Program{Semantics: executor.SemanticVersion, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: executor.SemanticVersion, Bytes: b})
	p.Subjects = append(p.Subjects, runtime.Subject{ID: ctl.Subject, Generation: 1, Revision: 1, Owner: "owner:controller"})
	p.Runs = append(p.Runs, runtime.InitialRun{ID: ctl, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: ctl.Subject, Operation: "input", Obligation: "stop"},
		runtime.Permission{Kind: "lifecycle", Source: ctl.Subject, Target: meterNext.Subject, Operation: "RetireSubject", Revision: 1})
}

func retireMeterTarget(t *testing.T, f *meterFixture) {
	t.Helper()
	ctx := t.Context()
	input, err := f.owner.AdmitInput(ctx, runtime.Input{TransportID: "retire-once", LogicalID: "retire-once", Target: "controller", Generation: 1, Obligation: "stop", Payload: map[string]schema.Value{}})
	if err != nil || !input.Accepted {
		t.Fatalf("controller ingress: %v %v", input, err)
	}
	state, err := f.owner.Snapshot(ctx, *input.Bound)
	if err != nil {
		t.Fatal(err)
	}
	state, err = f.owner.Commit(ctx, state.Authority(), input.ObservationID)
	if err != nil {
		t.Fatal(err)
	}
	state, err = f.owner.Commit(ctx, state.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	op := state.Projection.Data["request"].Text
	a, err := f.owner.AuthorizeIntent(ctx, op)
	if err != nil || !a.Allowed {
		t.Fatalf("lifecycle auth: %v %v", a, err)
	}
	req, err := f.owner.LoadLifecycleRequest(ctx, op)
	if err != nil {
		t.Fatal(err)
	}
	r, err := f.owner.DecideLifecycle(ctx, req)
	if err != nil || !r.Accepted || !r.After.Retired {
		t.Fatalf("retirement: %v %v", r, err)
	}
}

func TestContinuationCannotReviveRetiredAuthority(t *testing.T) {
	for _, name := range []string{"retire_before_approval", "retire_after_selection"} {
		t.Run(name, func(t *testing.T) {
			f := newMeterFixture(t, "north", func(p *runtime.Provision) { meterRetirementController(t, p) })
			if err := f.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			b := meterBasis(t, f)
			if name == "retire_after_selection" {
				a, err := f.owner.ApproveContinuation(t.Context(), b, "second sample")
				if err != nil || !a.Accepted {
					t.Fatalf("approval: %v %v", a, err)
				}
				before, err := f.owner.Snapshot(t.Context(), meterNext)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = f.owner.Commit(t.Context(), before.Authority(), a.ObservationID); err != nil {
					t.Fatal(err)
				}
			}
			retireMeterTarget(t, f)
			if name == "retire_before_approval" {
				d, err := f.owner.ApproveContinuation(t.Context(), b, "second sample")
				if err == nil && d.Accepted {
					t.Fatal("approved a retired successor")
				}
			}
			if err := f.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			cut := meterCut(t, f.path)
			if len(cut.Effects) != 1 || meterSpent(cut) != 1 || f.meter.calls.Load() != 1 {
				t.Fatal("retired target dispatched")
			}
			if _, err := runtime.Replay(t.Context(), f.path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if err := f.owner.Close(); err != nil {
				t.Fatal(err)
			}
			owner, err := runtime.Open(t.Context(), f.path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = owner.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestContinuationRejectsAnotherPaidActionInSameStore(t *testing.T) {
	other := runtime.RunID{Subject: "other-origin", Generation: 1}
	otherNext := runtime.RunID{Subject: "other-follower", Generation: 1}
	f := newMeterFixture(t, "north", func(p *runtime.Provision) {
		remap := map[string]runtime.RunID{meterOriginal.Subject: other, meterNext.Subject: otherNext}
		for _, s := range p.Subjects {
			s.ID = remap[s.ID].Subject
			p.Subjects = append(p.Subjects, s)
		}
		for _, r := range p.Runs {
			r.ID = remap[r.ID.Subject]
			p.Runs = append(p.Runs, r)
		}
		for _, g := range p.Permissions {
			if id, ok := remap[g.Source]; ok {
				g.Source = id.Subject
			}
			if id, ok := remap[g.Target]; ok {
				g.Target = id.Subject
			}
			p.Permissions = append(p.Permissions, g)
		}
		for _, root := range p.Composition.Roots {
			root.Run = remap[root.Run.Subject]
			root.Members = []runtime.RunID{root.Run}
			p.Composition.Roots = append(p.Composition.Roots, root)
		}
		slot := p.Composition.Continuations[0]
		slot.Predecessor, slot.Target, slot.ApprovalID = other, otherNext, "other-approval"
		p.Composition.Continuations = append(p.Composition.Continuations, slot)
	})
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	cut := meterCut(t, f.path)
	otherBasis, err := cut.ContinuationBasis(otherNext)
	if err != nil {
		t.Fatal(err)
	}
	original := meterBasis(t, f)
	if otherBasis.Namespace != original.Namespace || otherBasis.PredecessorID == original.PredecessorID || meterSpent(cut) != 2 {
		t.Fatal("expected two independent paid origins")
	}
	copied := otherBasis
	copied.Target = meterNext
	decision, err := f.owner.ApproveContinuation(t.Context(), copied, "reuse paid action")
	if err == nil && decision.Accepted {
		t.Fatal("one paid origin approved an unrelated successor")
	}
	valid, err := f.owner.ApproveContinuation(t.Context(), original, "sample again")
	if err != nil || !valid.Accepted {
		t.Fatalf("legitimate successor failed: %v %v", valid, err)
	}
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	cut = meterCut(t, f.path)
	if _, ok := cut.Continuation(otherNext); ok || meterSpent(cut) != 3 || f.meter.calls.Load() != 3 {
		t.Fatal("one approval amplified to the unrelated protected operation")
	}
}

func TestContinuationRejectsBasisThatBecameStaleBeforeApproval(t *testing.T) {
	f := newMeterFixture(t, "north")
	var stale runtime.ContinuationBasis
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary != "adapter.response" || e.Subject != meterOriginal.Subject {
			return nil
		}
		cut, err := runtime.InspectExecution(ctx, f.path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		stale, err = cut.ContinuationBasis(meterNext)
		return err
	})
	if err := f.run(ctx); err != nil {
		t.Fatal(err)
	}
	fresh := meterBasis(t, f)
	if stale.PredecessorID != fresh.PredecessorID || !stale.Admitted || stale.ResultDigest != "" || fresh.ResultDigest == "" {
		t.Fatal("observation did not become stale")
	}
	d, err := f.owner.ApproveContinuation(t.Context(), stale, "it had no result")
	if err == nil && d.Accepted {
		t.Fatal("stale view of original authorized a new operation")
	}
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if meterSpent(meterCut(t, f.path)) != 1 || f.meter.calls.Load() != 1 {
		t.Fatal("stale approval caused expenditure")
	}
}
