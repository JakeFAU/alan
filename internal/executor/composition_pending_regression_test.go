package executor

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
)

func TestRootPendingInvocationsDistinguishesRejectedAndUnadmitted(t *testing.T) {
	ctx := t.Context()
	limits := DefaultLimits()
	p, c := invocationFixture(t, "succeeded", 1)
	c.Roots[0].Limits.MaxCreatedRuns = 1 // The child template needs two created runs.
	path := filepath.Join(t.TempDir(), "pending.sqlite")
	s, err := createInvocationStore(ctx, path, p, c, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	first := invokeOnce(t, s)
	if first.Accepted || first.Reason != "created_run_limit" {
		t.Fatalf("expected a created-run refusal, got %+v", first)
	}
	next, err := s.Commit(ctx, current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	second := next.Projection.Data["pending"].Text
	roots, err := readRootCompletions(ctx, s.conn)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "live mixed cut", roots, second)
	snapshot, err := InspectExecution(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "snapshot mixed cut", snapshot.Roots, second)

	decision, err := s.AdmitInvocation(ctx, second)
	if err != nil || decision.Accepted || decision.Reason != "created_run_limit" {
		t.Fatalf("expected a second created-run refusal, got %+v, %v", decision, err)
	}
	if _, err = s.Commit(ctx, current(t, s, "root").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	result, err := s.SettleRoot(ctx, first.Invocation.RootID)
	if err != nil || !result.Ready || result.Outcome.ID == "" || result.Outcome.Status != "succeeded" {
		t.Fatalf("expected a published root success before child publications, got %+v, %v", result, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	replay, err := Replay(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "replay after settlement", replay.Roots)
	raw, err := Inspect(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "raw inspection after settlement", raw.Roots)
	reopened, err := Open(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	roots, err = readRootCompletions(ctx, reopened.conn)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "reopened after settlement", roots)
}

func TestRootPendingInvocationsIncludesAcceptedUnpublished(t *testing.T) {
	ctx := t.Context()
	limits := DefaultLimits()
	p, c := invocationFixture(t, "succeeded", 1)
	path := filepath.Join(t.TempDir(), "accepted.sqlite")
	s, err := createInvocationStore(ctx, path, p, c, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	accepted := invokeOnce(t, s)
	if !accepted.Accepted {
		t.Fatalf("expected an accepted admission, got %+v", accepted)
	}
	roots, err := readRootCompletions(ctx, s.conn)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "accepted and unpublished", roots, accepted.Invocation.OperationID)
	snapshot, err := InspectExecution(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	checkRootPendingInvocations(t, "snapshot accepted and unpublished", snapshot.Roots, accepted.Invocation.OperationID)
}

func TestRootPendingInvocationsPreservesRejectedCallerEvidence(t *testing.T) {
	ctx := t.Context()
	limits := DefaultLimits()
	p, c := invocationFixture(t, "succeeded", 1)
	awaitingInvocationRoot(t, &p)
	c.Roots[0].Limits.MaxCreatedRuns = 1
	path := filepath.Join(t.TempDir(), "await.sqlite")
	s, err := createInvocationStore(ctx, path, p, c, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	decision := invokeOnce(t, s)
	op := decision.Invocation.OperationID
	caller := RunID{Subject: "root", Generation: 1}
	if decision.Accepted || decision.Reason != "created_run_limit" {
		t.Fatalf("expected a created-run refusal, got %+v", decision)
	}
	inspect := func(cut string) (*ExecutionSnapshot, *InvocationView) {
		t.Helper()
		snapshot, err := InspectExecution(ctx, path, limits)
		if err != nil {
			t.Fatal(err)
		}
		checkRootPendingInvocations(t, cut, snapshot.Roots)
		intent, ok := snapshot.Operation(op)
		if !ok || intent.Kind != "invocation" || intent.Run != caller {
			t.Fatalf("%s: missing original invocation intent, got %+v", cut, intent)
		}
		view, ok := snapshot.Invocation(op)
		if !ok || view.Admission == nil || view.Admission.Accepted ||
			view.Admission.Reason != decision.Reason || view.Admission.Invocation.ID != decision.Invocation.ID {
			t.Fatalf("%s: missing original refusal, got %+v", cut, view)
		}
		return snapshot, view
	}
	hasPending := func(snapshot *ExecutionSnapshot, kind, observation string) bool {
		return slices.ContainsFunc(snapshot.Pending, func(p PendingObligation) bool {
			return p.Kind == kind && p.Run == caller && p.OperationID == op && p.ObservationID == observation
		})
	}

	before, refused := inspect("caller waiting before refusal delivery")
	run, ok := before.Run(caller)
	if !ok || run.Wait == nil || !run.Wait.Resolved || run.Wait.OperationID != op ||
		len(run.Wait.MatchingObservationIDs) != 0 || refused.Outcome != nil ||
		!hasPending(before, "waiting_invocation", "") || !hasPending(before, "invocation_awaiting_outcome", "") {
		t.Fatalf("expected a caller wait and outstanding refusal delivery, got %+v, %+v, %+v", run, refused, before.Pending)
	}

	delivery, err := s.SettleInvocation(ctx, op)
	if err != nil || !delivery.Accepted || delivery.ObservationID == "" {
		t.Fatalf("expected a delivered rejection outcome, got %+v, %v", delivery, err)
	}
	after, published := inspect("caller waiting after refusal delivery")
	observation, ok := after.Observation(delivery.ObservationID)
	if !ok || observation.Operation != op || observation.SelectedAt != nil ||
		observation.Provenance != "runtime-invocation" || observation.Payload["status"].Text != "rejected" ||
		published.Outcome == nil || published.Outcome.Status != "rejected" || published.Outcome.Reason != decision.Reason ||
		published.OutcomeObservationID != delivery.ObservationID || published.ParentSelection != "unconsumed" ||
		!hasPending(after, "matching_observation", delivery.ObservationID) ||
		!hasPending(after, "published_unconsumed", delivery.ObservationID) {
		t.Fatalf("expected the original unconsumed rejection outcome and caller wait, got %+v, %+v, %+v",
			observation, published, after.Pending)
	}

	selected, err := s.Commit(ctx, current(t, s, "root").Authority(), delivery.ObservationID)
	if err != nil {
		t.Fatal(err)
	}
	finished, consumed := inspect("caller selected the refusal")
	observation, ok = finished.Observation(delivery.ObservationID)
	wantSelection := ReductionRef{Run: caller, Sequence: selected.Sequence}
	if !ok || observation.SelectedAt == nil || *observation.SelectedAt != wantSelection ||
		consumed.SelectedAt == nil || *consumed.SelectedAt != wantSelection || consumed.ParentSelection != "selected" ||
		slices.ContainsFunc(finished.Pending, func(p PendingObligation) bool { return p.OperationID == op }) {
		t.Fatalf("expected the recorded selection and no remaining caller obligation, got %+v, %+v, %+v",
			observation, consumed, finished.Pending)
	}
}

func TestRootPendingInvocationsPropagatesMalformedAdmission(t *testing.T) {
	ctx := t.Context()
	s := invocationStore(t, "succeeded", 1, nil)
	accepted := invokeOnce(t, s)
	// Keep the root open so root preparation does not read this admission.
	if _, err := s.conn.ExecContext(ctx,
		"UPDATE composition_admission_decisions SET descriptor=? WHERE operation_id=?",
		[]byte("{}"),
		accepted.Invocation.OperationID,
	); err != nil {
		t.Fatal(err)
	}
	roots, err := readRootCompletions(ctx, s.conn)
	if roots != nil || !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected a corrupt admission error, got roots %+v, %v", roots, err)
	}
}

func checkRootPendingInvocations(t *testing.T, cut string, roots []RootCompletion, want ...string) {
	t.Helper()
	if len(roots) != 1 {
		t.Fatalf("%s: got %d roots, want 1", cut, len(roots))
	}
	if got := roots[0].PendingInvocations; !slices.Equal(got, want) {
		t.Errorf("%s: pending invocations = %q, want %q", cut, got, want)
	}
}
