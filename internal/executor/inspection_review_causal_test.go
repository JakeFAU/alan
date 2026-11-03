package executor

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestInspectRejectsRehashedInvocationRefusal(t *testing.T) {
	ctx := t.Context()
	p, profile := invocationFixture(t, "succeeded", 1)
	profile.Roots[0].Limits.MaxInvocations = 0
	path := filepath.Join(t.TempDir(), "refused.sqlite")
	s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := invokeOnce(t, s)
	if a.Accepted || a.Reason != "invocation_limit" {
		t.Fatal(a)
	}
	if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}

	// Replace one valid refusal with another and rebuild its exact fact digest.
	// Only the original decision prefix distinguishes the two explanations.
	a.Reason = "depth_limit"
	id := a.Invocation.OperationID
	if _, err := s.conn.ExecContext(ctx, "UPDATE composition_admission_decisions SET descriptor=CAST(? AS BLOB) WHERE operation_id=?", compositionJSON(a), id); err != nil {
		t.Fatal(err)
	}
	fact, err := compositionProtocolEvidence(ctx, s.conn, compositionAdmissionFact, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.conn.QueryRowContext(ctx, "SELECT decision_order FROM composition_facts WHERE id=?", fact.ID).Scan(&fact.Order); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE composition_facts SET descriptor=CAST(? AS BLOB) WHERE id=?", compositionJSON(fact), fact.ID); err != nil {
		t.Fatal(err)
	}
	if cut, err := InspectExecution(ctx, path, DefaultLimits()); !errors.Is(err, ErrCorrupt) || cut != nil {
		t.Fatalf("accepted an invocation refusal without its original cause: %v", err)
	}
}
