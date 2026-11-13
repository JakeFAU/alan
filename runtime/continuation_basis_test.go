//go:build integration

package runtime_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

type continuationBasisReader interface {
	ContinuationBasis(context.Context, runtime.RunID) (runtime.ContinuationBasis, error)
}

func readContinuationBasis(t *testing.T, ctx context.Context, owner *runtime.Store, target runtime.RunID) (runtime.ContinuationBasis, error) {
	t.Helper()
	reader, ok := any(owner).(continuationBasisReader)
	if !ok {
		t.Fatal("Store.ContinuationBasis is unavailable")
	}
	return reader.ContinuationBasis(ctx, target)
}

func compareContinuationBasis(t *testing.T, f *meterFixture, target runtime.RunID) (runtime.ContinuationBasis, error) {
	t.Helper()
	want, wantErr := meterCut(t, f.path).ContinuationBasis(target)
	before := meterFingerprint(t, f.path)
	got, err := readContinuationBasis(t, t.Context(), f.owner, target)
	if got != want || !errors.Is(err, wantErr) {
		t.Fatalf("owner basis = (%+v, %v), snapshot = (%+v, %v)", got, err, want, wantErr)
	}
	if after := meterFingerprint(t, f.path); after != before {
		t.Fatal("owner basis read changed database or WAL bytes")
	}
	return got, err
}

type refusingBasisMeter struct{ continuationMeter }

func (m *refusingBasisMeter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.calls.Add(1)
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "transport_error"}
}

func TestOwnerContinuationBasisAtRetainedCuts(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		name := "success"
		if refuse {
			name = "refusal"
		}
		t.Run(name, func(t *testing.T) {
			f := newMeterFixture(t, "north")
			if _, err := compareContinuationBasis(t, f, meterNext); err == nil {
				t.Fatal("unspent predecessor yielded a basis")
			}
			if _, err := compareContinuationBasis(t, f, runtime.RunID{Subject: meterNext.Subject, Generation: 2}); err == nil {
				t.Fatal("unknown target generation yielded a basis")
			}
			cuts := map[string]runtime.ContinuationBasis{}
			ctx := boundary.WithObserver(t.Context(), func(_ context.Context, e boundary.Event) error {
				if e.Subject != meterOriginal.Subject {
					return nil
				}
				switch e.Boundary {
				case "dispatch.consider", "adapter.response", "adapter.result-recorded", "adapter.settled":
					basis, err := compareContinuationBasis(t, f, meterNext)
					cuts[e.Boundary] = basis
					return err
				default:
					return nil
				}
			})
			var a effects.Adapter = f.meter
			if refuse {
				a = &refusingBasisMeter{continuationMeter{endpoint: "local-meter"}}
			}
			_, err := f.owner.Run(ctx, runtime.RunOptions{MaxSteps: 32, MaxDispatch: 16, MaxInFlight: 1,
				Adapters: map[string]effects.Adapter{"meter": a}})
			if err != nil {
				t.Fatal(err)
			}
			paid, admitted := cuts["dispatch.consider"], cuts["adapter.response"]
			result, settled := cuts["adapter.result-recorded"], cuts["adapter.settled"]
			selected, err := compareContinuationBasis(t, f, meterNext)
			if err != nil || paid.Namespace != f.owner.Namespace() || paid.Target != meterNext ||
				paid.PredecessorID == "" || paid.PreparedDigest == "" || paid.Admitted || paid.ResultDigest != "" {
				t.Fatalf("missing paid but unadmitted basis %+v, %v", paid, err)
			}
			paid.Admitted = true
			if admitted != paid || result.ResultDigest == "" || !result.Admitted || result != settled || settled != selected {
				t.Fatalf("retained cuts = admitted %+v, result %+v, settled %+v, selected %+v", admitted, result, settled, selected)
			}
			result.ResultDigest = ""
			if admitted != result {
				t.Fatal("result changed unrelated basis fields")
			}
			cut := meterCut(t, f.path)
			effect, ok := cut.Effect(selected.PredecessorID)
			want := "validated"
			if refuse {
				want = "transport_error"
			}
			if !ok || effect.ResultDisposition != want || effect.Settlement == nil {
				t.Fatalf("missing %s settlement: %+v", want, effect)
			}
			if observation, ok := cut.Observation(effect.Settlement.ObservationID); !ok || observation.SelectedAt == nil {
				t.Fatal("original result was not selected")
			}
		})
	}
}

func TestOwnerContinuationBasisReopenAndHistoricalDecision(t *testing.T) {
	f, historical := lateResultFixture(t)
	check := func() {
		t.Helper()
		got, err := compareContinuationBasis(t, f, meterNext)
		if err != nil || got != historical || got.ResultDigest != "" {
			t.Fatalf("approval basis after later result = %+v, %v", got, err)
		}
	}
	check()
	cut := meterCut(t, f.path)
	v, ok := cut.Continuation(meterNext)
	if !ok || v.SelectedAt == nil || v.SuccessorOperationID == "" || v.SuccessorOperationID == historical.PredecessorID ||
		f.meter.calls.Load() != 2 || meterSpent(cut) != 2 {
		t.Fatal("missing independently spent successor")
	}
	if err := f.owner.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readContinuationBasis(t, t.Context(), f.owner, meterNext); !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("closed owner basis = %v", err)
	}
	fresh, err := runtime.Open(t.Context(), f.path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	f.owner = fresh
	check() // Open has no configured adapter registry.
	first, err := f.owner.ApproveContinuation(t.Context(), historical, "new observation while original outstanding")
	if err != nil || !first.Accepted || first.ObservationID != v.ObservationID {
		t.Fatalf("duplicate approval = %+v, %v", first, err)
	}
	if f.meter.calls.Load() != 2 {
		t.Fatal("read or duplicate approval dispatched a provider")
	}
}

func TestOwnerContinuationBasisAfterTargetRetirement(t *testing.T) {
	f := newMeterFixture(t, "north", func(p *runtime.Provision) { meterRetirementController(t, p) })
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	original := meterBasis(t, f)
	decision, err := f.owner.ApproveContinuation(t.Context(), original, "new sample")
	if err != nil || !decision.Accepted {
		t.Fatalf("approval = %+v, %v", decision, err)
	}
	retireMeterTarget(t, f)
	got, err := compareContinuationBasis(t, f, meterNext)
	if err != nil || got != original || f.meter.calls.Load() != 1 {
		t.Fatalf("basis after target retirement = %+v, %v", got, err)
	}
}

func TestOwnerContinuationBasisBeforeLaterDispatchAdmission(t *testing.T) {
	f := newMeterFixture(t, "north")
	var original runtime.ContinuationBasis
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary != "dispatch.consider" || e.Subject != meterOriginal.Subject {
			return nil
		}
		basis, err := readContinuationBasis(t, ctx, f.owner, meterNext)
		if err != nil {
			return err
		}
		if basis.Admitted || basis.ResultDigest != "" {
			return errors.New("original was already admitted or had a result")
		}
		original = basis
		_, err = f.owner.ApproveContinuation(ctx, basis, "new sample before original admission")
		return err
	})
	if err := f.run(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := compareContinuationBasis(t, f, meterNext)
	if err != nil || got != original || got.Admitted || got.ResultDigest != "" {
		t.Fatalf("historical basis after later gate = %+v, %v", got, err)
	}
	e, ok := meterCut(t, f.path).Effect(original.PredecessorID)
	if !ok || e.DispatchAdmission == nil || e.AdapterResult == nil {
		t.Fatal("later original dispatch and result were not retained")
	}
}

func TestOwnerContinuationBasisConcurrentCompletionMakesPriorCutStale(t *testing.T) {
	f := newMeterFixture(t, "north")
	response := make(chan struct{})
	release := make(chan struct{})
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "adapter.response" && e.Subject == meterOriginal.Subject {
			close(response)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	done := make(chan error, 1)
	go func() { done <- f.run(ctx) }()
	select {
	case <-response:
	case err := <-done:
		t.Fatalf("Run returned before its response boundary: %v", err)
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	prior, err := compareContinuationBasis(t, f, meterNext)
	if err != nil || !prior.Admitted || prior.ResultDigest != "" {
		t.Fatalf("basis during completion = %+v, %v", prior, err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	before := meterFingerprint(t, f.path)
	if _, err := readContinuationBasis(t, canceled, f.owner, meterNext); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled basis = %v", err)
	}
	if meterFingerprint(t, f.path) != before {
		t.Fatal("canceled owner basis read changed durable bytes")
	}
	close(release)
	released = true
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	fresh, err := compareContinuationBasis(t, f, meterNext)
	if err != nil || fresh.PredecessorID != prior.PredecessorID || fresh.ResultDigest == "" {
		t.Fatalf("basis after completion = %+v, %v", fresh, err)
	}
	if d, err := f.owner.ApproveContinuation(t.Context(), prior, "result was outstanding"); !errors.Is(err, runtime.ErrStale) || d.Accepted {
		t.Fatalf("approval of stale basis = %+v, %v", d, err)
	}
	if f.meter.calls.Load() != 1 || meterSpent(meterCut(t, f.path)) != 1 {
		t.Fatal("stale approval spent the successor opportunity")
	}
}

func TestOwnerContinuationBasisRejectsAmbiguousPaidOriginals(t *testing.T) {
	f := newMeterFixture(t, "north", func(p *runtime.Provision) {
		twice := meterProgram(t, false, "north", 2)
		for i := range p.Runs {
			if p.Runs[i].ID == meterOriginal {
				p.Runs[i].ProgramID = twice.ID()
			}
		}
		p.Programs = append(p.Programs, runtime.Artifact{ID: twice.ID(), Bytes: twice.Bytes(), Semantics: author.SemanticVersion})
		for i := range p.Composition.Roots {
			if p.Composition.Roots[i].Run == meterOriginal {
				p.Composition.Roots[i].Limits.MaxEffects = 2
			}
		}
	})
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.meter.calls.Load() != 2 {
		t.Fatal("expected two paid matching originals")
	}
	if _, err := compareContinuationBasis(t, f, meterNext); !errors.Is(err, runtime.ErrConflict) {
		t.Fatalf("ambiguous basis = %v", err)
	}
	// The write API accepts an explicit exact basis when the operator chooses.
	// An established decision therefore has precedence over ambiguity.
	cut := meterCut(t, f.path)
	e := cut.Effects[0]
	if e.Source != meterOriginal || e.AdapterAttempt == nil || e.AdapterResult == nil {
		t.Fatal("missing explicitly chosen original")
	}
	explicit := runtime.ContinuationBasis{Namespace: cut.Namespace, Target: meterNext, PredecessorID: e.OperationID,
		PreparedDigest: e.AdapterAttempt.Prepared.Digest, Admitted: true, ResultDigest: e.AdapterResult.Digest}
	if decision, err := f.owner.ApproveContinuation(t.Context(), explicit, "explicit original"); err != nil || !decision.Accepted {
		t.Fatalf("explicit approval = %+v, %v", decision, err)
	}
	if got, err := compareContinuationBasis(t, f, meterNext); got != explicit || err != nil {
		t.Fatalf("explicit approved basis = %+v, %v", got, err)
	}
}

func TestOwnerContinuationBasisRequiresComposition(t *testing.T) {
	p, err := (author.Application{meterOriginal.Subject: {Artifact: meterProgram(t, false, "north", 1)}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ordinary.sqlite")
	owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close() })
	if _, err := compareContinuationBasis(t, &meterFixture{path: path, owner: owner}, meterNext); !errors.Is(err, runtime.ErrVersion) {
		t.Fatalf("basis without composition = %v", err)
	}
}

// Each fixture is a disposable ledger. Damage occurs after owner validation
// so Open cannot intercept it before the targeted reader runs.
func TestOwnerContinuationBasisRejectsMalformedReferencedEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		approved  bool
	}{
		{"original_generation", "UPDATE intents SET descriptor=CAST(json_set(descriptor,'$.Run.Generation',2) AS BLOB) WHERE id=?", false},
		{"wrong_effect", "UPDATE intents SET descriptor=CAST(json_set(descriptor,'$.Obligation','different') AS BLOB) WHERE id=?", false},
		{"authorization", "UPDATE attempts SET descriptor=CAST('{}' AS BLOB) WHERE intent_id=?", false},
		{"reservation", "UPDATE composition_reservations SET descriptor=CAST(json_set(descriptor,'$.AttemptDigest','different') AS BLOB) WHERE operation_id=?", false},
		{"charge", "DELETE FROM composition_charges WHERE operation_id=?", false},
		{"gate", "UPDATE adapter_dispatch_admissions SET prepared_digest='different' WHERE intent_id=?", false},
		{"gate_after_terminal", "UPDATE adapter_dispatch_admissions SET after_fact=(SELECT max(decision_order) FROM composition_facts) WHERE intent_id=?", false},
		{"missing_gate", "DELETE FROM adapter_dispatch_admissions WHERE intent_id=?", false},
		{"result", "UPDATE adapter_results SET descriptor=CAST(json_set(descriptor,'$.Candidate.Key','different') AS BLOB) WHERE intent_id=?", false},
		{"missing_result", "DELETE FROM adapter_results WHERE intent_id=?", false},
		{"missing_result_and_fact", "DELETE FROM composition_facts WHERE id IN (SELECT id FROM composition_facts WHERE kind='external_result' AND json_extract(descriptor,'$.OperationID')=?)", false},
		{"approval_admission", "UPDATE admissions SET observation_id=NULL WHERE transport_id=?", true},
		{"missing_approval_admission", "DELETE FROM admissions WHERE transport_id=?", true},
		{"approval_namespace", "UPDATE admissions SET descriptor=CAST(json_set(descriptor,'$.Continuation.Namespace','foreign') AS BLOB) WHERE transport_id=?", true},
		{"approval_fact", "UPDATE composition_facts SET descriptor=CAST(json_set(descriptor,'$.RecordDigest','different') AS BLOB) WHERE kind='continuation' AND json_extract(descriptor,'$.OperationID')=?", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMeterFixture(t, "north")
			if err := f.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			original := meterBasis(t, f)
			id := original.PredecessorID
			if tc.approved {
				if _, err := f.owner.ApproveContinuation(t.Context(), original, "new sample"); err != nil {
					t.Fatal(err)
				}
				id = meterApproval
			}
			db, err := sql.Open("sqlite3", f.path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			tx, err := db.BeginTx(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if tc.name == "missing_result_and_fact" {
				meterExec(t, tx, "DELETE FROM adapter_results WHERE intent_id=?", id)
			}
			meterExec(t, tx, tc.sql, id)
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			before := meterFingerprint(t, f.path)
			if got, err := readContinuationBasis(t, t.Context(), f.owner, meterNext); got != (runtime.ContinuationBasis{}) || !errors.Is(err, runtime.ErrCorrupt) {
				t.Fatalf("targeted read accepted damaged %s: %+v, %v", tc.name, got, err)
			}
			if after := meterFingerprint(t, f.path); after != before {
				t.Fatal("rejecting owner read changed durable bytes")
			}
		})
	}
}
