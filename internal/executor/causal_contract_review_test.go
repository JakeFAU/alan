package executor

import (
	"errors"
	"testing"
)

// A locally plausible source epoch need not precede the target lookup. Both
// subjects exist from provision and stay open through the same owner renewal.
func TestLifecycleLookupSharesEpochWithProvisionedSource(t *testing.T) {
	ctx := t.Context()
	s, path := messageStore(t, lifecycleFixture(t, []string{"Retry"}, false))
	request := committedRequest(t, s, "requester")
	d := authorizedDecision(t, s, request)
	if d.Accepted || d.Before == nil || d.Before.Epoch != 1 || d.Reason != "not_retryable" {
		t.Fatal(d)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	a, err := owner.AuthorizeIntent(ctx, request.Intent.ID)
	if err != nil || a.Source.Epoch != 1 || !a.Allowed {
		t.Fatal(a, err)
	}
	if current(t, owner, "worker").Epoch != 2 || current(t, owner, "requester").Epoch != 2 {
		t.Fatal("both open runs must share their second activation")
	}
	a.Source.Epoch = 2
	rejectAlteredAuthorization(t, owner, path, a)
}

func TestLifecycleGenerationRefusalRequiresEarlierRetry(t *testing.T) {
	ctx := t.Context()
	p := Provision{}
	emit := func(target, operation string) Instruction {
		return Instruction{Op: "emit", Dst: "pending", Effect: &Effect{
			Kind: "lifecycle", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("obligation")),
			Payload: map[string]Operand{"operation": Lit(Text(operation)), "revision": Lit(Int(1))},
		}}
	}
	addProgram(t, &p, "worker", []Instruction{emit("requester", "CancelRun"), {Op: "end", Outcome: "failed"}}, map[string]Value{})
	addProgram(t, &p, "requester", []Instruction{emit("worker", "Retry"), {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = []Permission{{Kind: "lifecycle", Source: "requester", Target: "worker", Operation: "Retry", Revision: 1}}
	s, path := messageStore(t, p)
	request := committedRequest(t, s, "worker")
	if _, err := s.Commit(ctx, current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	a, err := s.AuthorizeIntent(ctx, request.Intent.ID)
	if err != nil || a.Allowed || a.Reason != "source_closed" {
		t.Fatal(a, err)
	}
	retry := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if !retry.Accepted || retry.Successor != 2 {
		t.Fatal(retry)
	}
	// The attempts journal places this refusal before the authorization of
	// the only Retry. The eventual successor cannot explain this refusal.
	a.Reason = "source_generation_changed"
	rejectAlteredAuthorization(t, s, path, a)
}

func TestLifecycleCannotClaimSourceRevisionBeforeProvision(t *testing.T) {
	ctx := t.Context()
	p := lifecycleFixture(t, []string{"CancelRun"}, false)
	for i := range p.Subjects {
		if p.Subjects[i].ID == "requester" {
			p.Subjects[i].Revision = 2
		}
	}
	s, path := messageStore(t, p)
	request := committedRequest(t, s, "requester")
	a, err := s.AuthorizeIntent(ctx, request.Intent.ID)
	if err != nil || !a.Allowed || a.Source.Revision != 2 {
		t.Fatal(a, err)
	}
	if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	// Replace the emission revision and source response together. Their local
	// agreement cannot establish a revision preceding the genuine origin.
	a.Allowed, a.Reason = false, "source_revision_changed"
	if _, err := s.conn.ExecContext(ctx, `UPDATE history SET source_revision=1 WHERE subject='requester';
		UPDATE intents SET source_revision=1 WHERE subject='requester';
		UPDATE attempts SET descriptor=CAST(json_set(descriptor,'$.Revision',1) AS BLOB)`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE attempts SET result=? WHERE id=?", []byte(compositionJSON(a)), a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	checkInvalidAuthority(t, path)
}

func rejectAlteredAuthorization(t *testing.T, s *Store, path string, a Authorization) {
	t.Helper()
	ctx, limits := t.Context(), DefaultLimits()
	if _, err := InspectExecution(ctx, path, limits); err != nil {
		t.Fatal("original inspection", err)
	}
	if _, err := Replay(ctx, path, limits); err != nil {
		t.Fatal("original replay", err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE attempts SET result=? WHERE id=?", []byte(compositionJSON(a)), a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	checkInvalidAuthority(t, path)
}

func checkInvalidAuthority(t *testing.T, path string) {
	t.Helper()
	ctx, limits := t.Context(), DefaultLimits()
	before := inspectionFingerprint(t, path)
	if cut, err := InspectExecution(ctx, path, limits); cut != nil || !errors.Is(err, ErrCorrupt) {
		t.Errorf("inspection accepted impossible authority: %v", err)
	}
	if _, err := Replay(ctx, path, limits); !errors.Is(err, ErrCorrupt) {
		t.Errorf("replay accepted impossible authority: %v", err)
	}
	owner, err := Open(ctx, path, limits)
	if owner != nil {
		owner.Close()
	}
	if owner != nil || !errors.Is(err, ErrCorrupt) || inspectionFingerprint(t, path) != before {
		t.Fatalf("owner failed to reject impossible authority without durable change: %v", err)
	}
}
