//go:build integration

package executor

import (
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/effects"
)

func TestDeniedAdapterCannotHaveProviderResponse(t *testing.T) {
	ctx, limits := t.Context(), DefaultLimits()
	p := Provision{}
	addAdapterProgram(t, &p, "worker", true)
	id := RunID{Subject: "worker", Generation: 1}
	p.Composition = &compositionProvision{RecordDispatchAdmissions: true, Roots: []compositionRoot{{
		Run: id, Members: []RunID{id}, Capabilities: []string{"model"},
		Limits: compositionLimits{MaxEffects: 0, MaxInFlight: 1},
	}}}
	path := filepath.Join(t.TempDir(), "denied.sqlite")
	s, err := Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if _, err := s.Run(ctx, RunOptions{
		Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}, MaxSteps: 30, MaxDispatch: 30,
	}); err != nil {
		t.Fatal(err)
	}
	cut, err := InspectExecution(ctx, path, limits)
	if err != nil || len(cut.Effects) != 1 {
		t.Fatalf("expected one inspected effect: %v", err)
	}
	e := cut.Effects[0]
	if e.AdapterAttempt == nil || e.AdapterAttempt.Allowed || e.AdapterAttempt.Reason != "effect_limit" ||
		e.AdapterResult == nil || e.AdapterResult.Disposition != "source_denied" || e.Settlement == nil ||
		e.DispatchAdmission != nil || e.Dispatch.Maximum == nil || *e.Dispatch.Maximum != 0 || cut.Roots[0].Charges.Effects != 0 {
		t.Fatalf("expected a genuine zero-opportunity refusal: %+v", e)
	}

	// Keep the refusal, identity, selection and every digest consistent. A
	// provider status alone contradicts the original zero-opportunity decision.
	r := *e.AdapterResult
	r.Candidate.HTTPStatus = http.StatusTooManyRequests
	r.Digest = adapterResultDigest(r)
	for _, stmt := range []string{
		"UPDATE adapter_results SET descriptor=CAST(? AS BLOB) WHERE intent_id=?",
		"UPDATE settlements SET descriptor=CAST(? AS BLOB) WHERE intent_id=?",
	} {
		if _, err := s.conn.ExecContext(ctx, stmt, compositionJSON(r), e.OperationID); err != nil {
			t.Fatal(err)
		}
	}
	fact, err := compositionProtocolEvidence(ctx, s.conn, compositionExternalResultFact, e.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.conn.QueryRowContext(ctx, "SELECT decision_order FROM composition_facts WHERE id=?", fact.ID).Scan(&fact.Order); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE composition_facts SET descriptor=CAST(? AS BLOB) WHERE id=?", compositionJSON(fact), fact.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(ctx, path, limits); !errors.Is(err, ErrCorrupt) {
		t.Errorf("raw inspection accepted provider status on a local refusal: %v", err)
	}
	rejectDispatchHistory(t, s, path) // Typed inspection, Replay and fresh Open.
}
