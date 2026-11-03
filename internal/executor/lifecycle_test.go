package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func lifecycleFixture(t *testing.T, operations []string, retryCancelled bool) Provision {
	t.Helper()
	p := Provision{}
	code := []Instruction{}
	delta := int64(1)
	seen := map[string]bool{}
	for _, operation := range operations {
		code = append(code, Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("worker")), Generation: Lit(Int(1)), Obligation: Lit(Text("obligation")), Payload: map[string]Operand{"operation": Lit(Text(operation)), "revision": Lit(Int(1))}}, Updates: map[string]Operand{"spent": {Ref: "spent", Offset: &delta}}})
		if !seen[operation] {
			p.Permissions = append(p.Permissions, Permission{Kind: "lifecycle", Source: "requester", Target: "worker", Operation: operation, Revision: 1})
			seen[operation] = true
		}
	}
	code = append(code, Instruction{Op: "end", Outcome: "succeeded"})
	addProgram(t, &p, "requester", code, map[string]Value{"spent": Int(0)})
	addProgram(t, &p, "worker", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{"seed": Int(700)})
	p.Subjects[1].RetryCancelled = retryCancelled
	return p
}
func committedRequest(t *testing.T, s *Store, source string) LifecycleRequest {
	t.Helper()
	after, err := s.Commit(t.Context(), current(t, s, source).Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	request, err := s.LoadLifecycleRequest(t.Context(), after.Projection.Data["pending"].Text)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
func authorizedDecision(t *testing.T, s *Store, request LifecycleRequest) LifecycleResult {
	t.Helper()
	auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
	if err != nil || !auth.Allowed {
		t.Fatalf("authorization %+v %v", auth, err)
	}
	result, err := s.DecideLifecycle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestLifecycleAcceptedRetryReopensBeforeSettlement(t *testing.T) {
	s, path := messageStore(t, lifecycleFixture(t, []string{"Retry"}, false))
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	request := committedRequest(t, s, "requester")
	before := current(t, s, "requester")
	result := authorizedDecision(t, s, request)
	if !result.Accepted || result.Successor != 2 || count(t, s, "observations") != 0 {
		t.Fatalf("decision not separate %+v", result)
	}
	successor, err := s.Snapshot(t.Context(), RunID{"worker", 2})
	if err != nil {
		t.Fatal(err)
	}
	if successor.Sequence != 0 || successor.Projection.Data["seed"] != Int(700) || successor.ProgramID != current(t, s, "worker").ProgramID {
		t.Fatal("fresh execution did not clone immutable initial artifact")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	resumed := current(t, reopened, "requester")
	if resumed.Epoch != before.Epoch+1 || !reflect.DeepEqual(resumed.Projection, before.Projection) || resumed.Projection.Data["spent"] != Int(1) {
		t.Fatal("physical recovery reset ordinary policy")
	}
	duplicate, err := reopened.DecideLifecycle(t.Context(), request)
	if err != nil || !reflect.DeepEqual(result, duplicate) {
		t.Fatalf("accepted retransmission %+v %v", duplicate, err)
	}
	changed := request
	changed.ExpectedRevision++
	if _, err = reopened.DecideLifecycle(t.Context(), changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed request accepted: %v", err)
	}
	first, err := reopened.SettleLifecycle(t.Context(), request.Intent.ID, "result")
	if err != nil || !first.Accepted {
		t.Fatalf("settlement %+v %v", first, err)
	}
	if _, err = reopened.SettleLifecycle(t.Context(), request.Intent.ID, "result-again"); err != nil {
		t.Fatal(err)
	}
	if count(t, reopened, "settlements") != 1 || count(t, reopened, "observations") != 1 {
		t.Fatal("duplicated outcome observation")
	}
	if _, err = reopened.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); err != nil {
		t.Fatal(err)
	}
	successor, err = reopened.Snapshot(t.Context(), RunID{"worker", 2})
	if err != nil || successor.Sequence != 1 || successor.Projection.Status != "failed" {
		t.Fatalf("successor did not execute ordinary code %+v %v", successor, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleOriginalRejectionSurvivesChangedEligibility(t *testing.T) {
	s, path := messageStore(t, lifecycleFixture(t, []string{"Retry", "Retry"}, false))
	first := committedRequest(t, s, "requester")
	rejected := authorizedDecision(t, s, first)
	if rejected.Accepted || rejected.Reason != "not_retryable" {
		t.Fatalf("wrong initial rejection %+v", rejected)
	}
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	again, err := s.DecideLifecycle(t.Context(), first)
	if err != nil || !reflect.DeepEqual(rejected, again) {
		t.Fatalf("lost original rejection %+v %v", again, err)
	}
	second := committedRequest(t, s, "requester")
	accepted := authorizedDecision(t, s, second)
	if !accepted.Accepted || accepted.Successor != 2 {
		t.Fatalf("new operation did not see real eligibility %+v", accepted)
	}
	again, err = s.DecideLifecycle(t.Context(), first)
	if err != nil || !reflect.DeepEqual(rejected, again) {
		t.Fatal("old rejection changed after actual successor")
	}
	if _, err = s.SettleLifecycle(t.Context(), first.Intent.ID, "old-rejection"); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleCancellationAbandonmentAndRetirement(t *testing.T) {
	for _, test := range []struct {
		name           string
		operations     []string
		retryCancelled bool
		wantRetry      bool
		status         string
	}{
		{"cancel-no-retry", []string{"CancelRun", "Retry"}, false, false, "cancelled"},
		{"cancel-retry", []string{"CancelRun", "Retry"}, true, true, "cancelled"},
		{"abandon-retry", []string{"AbandonRun", "Retry"}, false, true, "abandoned"},
		{"retire", []string{"RetireSubject", "Retry"}, true, false, "cancelled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, path := messageStore(t, lifecycleFixture(t, test.operations, test.retryCancelled))
			first := authorizedDecision(t, s, committedRequest(t, s, "requester"))
			if !first.Accepted {
				t.Fatal(first)
			}
			target := current(t, s, "worker")
			if target.Sequence != 0 || target.Projection.Status != test.status {
				t.Fatalf("authority closure %+v", target)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			retry := authorizedDecision(t, reopened, committedRequest(t, reopened, "requester"))
			if retry.Accepted != test.wantRetry {
				t.Fatal(retry)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLifecycleObserverWithoutGrantGetsRetainedRejection(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry"}, false)
	p.Permissions = []Permission{{Kind: "observation", Source: "worker", Target: "requester", Operation: "terminal", Revision: 1, Obligation: "obligation"}}
	s, path := messageStore(t, p)
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	result := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if result.Accepted || result.Reason != "grant_denied" {
		t.Fatalf("observation conferred lifecycle authority %+v", result)
	}
	if _, err := s.SettleLifecycle(t.Context(), result.Request.Intent.ID, "denial"); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestOriginalServiceBindingSurvivesRealRetry(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry"}, false)
	p.Permissions = append(p.Permissions, Permission{Kind: "input", Source: "local", Target: "worker", Operation: "input"})
	s, path := messageStore(t, p)
	input := Input{TransportID: "original-service", LogicalID: "original-logical", Target: "worker", Payload: map[string]Value{"explicit": Text("old")}}
	original, err := s.AdmitInput(t.Context(), input)
	if err != nil || !original.Accepted {
		t.Fatalf("input %+v %v", original, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	accepted := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if !accepted.Accepted {
		t.Fatal(accepted)
	}
	again, err := s.AdmitInput(t.Context(), input)
	if err != nil || !reflect.DeepEqual(original, again) || again.Bound.Generation != 1 {
		t.Fatalf("old service rebound %+v %v", again, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleAuthorizationSurvivesClosureButNewAuthorizationIsDenied(t *testing.T) {
	for _, authorizeFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(authorizeFirst), func(t *testing.T) {
			p := lifecycleFixture(t, []string{"Retry"}, false)
			addProgram(t, &p, "closer", []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("requester")), Generation: Lit(Int(1)), Obligation: Lit(Text("close")), Payload: map[string]Operand{"operation": Lit(Text("CancelRun")), "revision": Lit(Int(1))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			p.Permissions = append(p.Permissions, Permission{Kind: "lifecycle", Source: "closer", Target: "requester", Operation: "CancelRun", Revision: 1})
			s, path := messageStore(t, p)
			if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			request := committedRequest(t, s, "requester")
			if authorizeFirst {
				auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
				if err != nil || !auth.Allowed {
					t.Fatalf("auth %+v %v", auth, err)
				}
			}
			closure := authorizedDecision(t, s, committedRequest(t, s, "closer"))
			if !closure.Accepted {
				t.Fatal(closure)
			}
			auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
			if err != nil || auth.Allowed != authorizeFirst {
				t.Fatalf("source boundary %+v %v", auth, err)
			}
			if authorizeFirst {
				result, err := s.DecideLifecycle(t.Context(), request)
				if err != nil || !result.Accepted {
					t.Fatalf("prior authorization erased %+v %v", result, err)
				}
			} else {
				if _, err = s.DecideLifecycle(t.Context(), request); !errors.Is(err, ErrSourceDenied) {
					t.Fatalf("new authorization bypassed closure: %v", err)
				}
				if auth.ClosureOperation != closure.Request.Intent.ID {
					t.Fatal("closure denial lost authority evidence")
				}
			}
			settlement, err := s.SettleLifecycle(t.Context(), request.Intent.ID, "late-result")
			if err != nil || !settlement.Accepted {
				t.Fatalf("late settlement %+v %v", settlement, err)
			}
			if current(t, s, "requester").Projection.Status != "cancelled" || current(t, s, "requester").Sequence != 1 {
				t.Fatal("late evidence reopened requester")
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
		})
	}
}

func TestLifecycleCancellationAfterFailurePreservesRetryEligibility(t *testing.T) {
	s, path := messageStore(t, lifecycleFixture(t, []string{"CancelRun", "Retry"}, false))
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	cancel := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if !cancel.Accepted || cancel.Reason != "already_terminal" || current(t, s, "worker").Projection.Status != "failed" {
		t.Fatal("late cancellation replaced failure")
	}
	retry := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if !retry.Accepted {
		t.Fatal(retry)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func appendRequesterReceive(t *testing.T, p *Provision) {
	t.Helper()
	var program Program
	if err := json.Unmarshal(p.Programs[0].Bytes, &program); err != nil {
		t.Fatal(err)
	}
	end := program.Code[len(program.Code)-1]
	program.Code = append(program.Code[:len(program.Code)-1], Instruction{Op: "receive", Match: &Match{Kind: "lifecycle", Provenance: "runtime-lifecycle", Source: Lit(Text("worker")), Generation: Lit(Int(1)), Obligation: Lit(Text("obligation")), Operation: Ref("pending")}, Bind: map[string]string{"accepted": "accepted", "reason": "reason", "successor": "successor"}}, end)
	encoded, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[0] = Artifact{ID: id, Semantics: SemanticVersion, Bytes: encoded}
	p.Runs[0].ProgramID = id
}
func TestOldLifecycleResultCannotConsumeNewPendingOperation(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry", "Retry"}, false)
	appendRequesterReceive(t, &p)
	s, path := messageStore(t, p)
	old := committedRequest(t, s, "requester")
	if result := authorizedDecision(t, s, old); result.Accepted {
		t.Fatal("active worker retried")
	}
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	fresh := committedRequest(t, s, "requester")
	if result := authorizedDecision(t, s, fresh); !result.Accepted {
		t.Fatal(result)
	}
	oldAdmission, err := s.SettleLifecycle(t.Context(), old.Intent.ID, "old")
	if err != nil {
		t.Fatal(err)
	}
	before := current(t, s, "requester")
	if _, err = s.Commit(t.Context(), before.Authority(), oldAdmission.ObservationID); !errors.Is(err, ErrIneligible) {
		t.Fatalf("old result matched fresh pending: %v", err)
	}
	if !reflect.DeepEqual(before, current(t, s, "requester")) {
		t.Fatal("old result changed projection")
	}
	newAdmission, err := s.SettleLifecycle(t.Context(), fresh.Intent.ID, "new")
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.Commit(t.Context(), before.Authority(), newAdmission.ObservationID)
	if err != nil || !after.Projection.Data["accepted"].Flag || after.Projection.Data["spent"] != Int(2) {
		t.Fatalf("new correlated result %+v %v", after, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestStaleGenerationRejectionCorrelatesToOriginalExpectedGeneration(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry"}, false)
	appendRequesterReceive(t, &p)
	addProgram(t, &p, "advance", []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("worker")), Generation: Lit(Int(1)), Obligation: Lit(Text("advance")), Payload: map[string]Operand{"operation": Lit(Text("Retry")), "revision": Lit(Int(1))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = append(p.Permissions, Permission{Kind: "lifecycle", Source: "advance", Target: "worker", Operation: "Retry", Revision: 1})
	s, path := messageStore(t, p)
	old := committedRequest(t, s, "requester")
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if result := authorizedDecision(t, s, committedRequest(t, s, "advance")); !result.Accepted {
		t.Fatal(result)
	}
	rejection := authorizedDecision(t, s, old)
	if rejection.Accepted || rejection.Reason != "generation_changed" {
		t.Fatal(rejection)
	}
	admitted, err := s.SettleLifecycle(t.Context(), old.Intent.ID, "stale-result")
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.Commit(t.Context(), current(t, s, "requester").Authority(), admitted.ObservationID)
	if err != nil || after.Projection.Data["reason"] != Text("generation_changed") {
		t.Fatalf("original wait could not consume valid rejection %+v %v", after, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleCorruptCreationAndClosureRejectBeforeEpochRenewal(t *testing.T) {
	for _, query := range []string{
		`UPDATE run_origins SET predecessor_generation=99 WHERE kind='retry'`,
		`UPDATE run_origins SET operation_id=NULL WHERE kind='retry'`,
		`DELETE FROM lifecycle_results`,
	} {
		t.Run(query, func(t *testing.T) {
			s, path := messageStore(t, lifecycleFixture(t, []string{"Retry"}, false))
			if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if result := authorizedDecision(t, s, committedRequest(t, s, "requester")); !result.Accepted {
				t.Fatal(result)
			}
			before := current(t, s, "requester")
			// This is deliberate corruption injection, never lifecycle setup/eligibility.
			if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if reopened, err := Open(t.Context(), path, DefaultLimits()); err == nil {
				reopened.Close()
				t.Fatal("corrupt creation accepted")
			}
			inspector, err := connect(t.Context(), path, true)
			if err != nil {
				t.Fatal(err)
			}
			defer inspector.Close()
			var epoch int64
			if err = inspector.conn.QueryRowContext(t.Context(), "SELECT epoch FROM runs WHERE subject='requester'").Scan(&epoch); err != nil {
				t.Fatal(err)
			}
			if epoch != before.Epoch {
				t.Fatal("corruption renewed epoch")
			}
		})
	}
	s, path := messageStore(t, lifecycleFixture(t, []string{"CancelRun"}, false))
	authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if _, err := s.conn.ExecContext(t.Context(), `UPDATE terminal_obligations SET operation_id=NULL WHERE subject='worker'`); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("unexplained lifecycle closure accepted")
	}
}

func TestRetryRacesWithCancelAndRetire(t *testing.T) {
	for _, other := range []string{"CancelRun", "RetireSubject"} {
		for _, retryFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/retryFirst=%t", other, retryFirst), func(t *testing.T) {
				operations := []string{other, "Retry"}
				if retryFirst {
					operations = []string{"Retry", other}
				}
				s, path := messageStore(t, lifecycleFixture(t, operations, false))
				if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
					t.Fatal(err)
				}
				first := authorizedDecision(t, s, committedRequest(t, s, "requester"))
				if !first.Accepted {
					t.Fatal(first)
				}
				second := authorizedDecision(t, s, committedRequest(t, s, "requester"))
				if retryFirst {
					if second.Accepted || second.Reason != "generation_changed" {
						t.Fatal(second)
					}
				} else if other == "RetireSubject" {
					if second.Accepted || second.Reason != "retired" {
						t.Fatal(second)
					}
				} else if !second.Accepted {
					t.Fatal(second)
				}
				if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestRetryPreservesNonOneStaticRootGeneration(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry"}, false)
	var program Program
	if err := json.Unmarshal(p.Programs[0].Bytes, &program); err != nil {
		t.Fatal(err)
	}
	program.Code[0].Effect.Generation = Lit(Int(42))
	encoded, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[0] = Artifact{ID: id, Semantics: SemanticVersion, Bytes: encoded}
	p.Runs[0].ProgramID = id
	p.Subjects[1].Generation = 42
	p.Runs[1].ID.Generation = 42
	s, path := messageStore(t, p)
	worker, err := s.Snapshot(t.Context(), RunID{"worker", 42})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), worker.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	result := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if !result.Accepted || result.Successor != 43 {
		t.Fatal(result)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	successor, err := reopened.Snapshot(t.Context(), RunID{"worker", 43})
	if err != nil || successor.Sequence != 0 || successor.Projection.Data["seed"] != Int(700) {
		t.Fatalf("non-1 root successor %+v %v", successor, err)
	}
}
func TestLifecycleExpectedRevisionIsCheckedWithMatchingGrant(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry"}, false)
	var program Program
	if err := json.Unmarshal(p.Programs[0].Bytes, &program); err != nil {
		t.Fatal(err)
	}
	program.Code[0].Effect.Payload["revision"] = Lit(Int(2))
	encoded, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[0] = Artifact{ID: id, Semantics: SemanticVersion, Bytes: encoded}
	p.Runs[0].ProgramID = id
	p.Permissions[0].Revision = 2
	s, _ := messageStore(t, p)
	if _, err = s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	result := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if result.Accepted || result.Reason != "revision_changed" {
		t.Fatalf("revision check masked by grant/eligibility %+v", result)
	}
}
func TestLifecycleDecisionAndSettlementRollbackRemainSeparate(t *testing.T) {
	for _, operation := range []string{"Retry", "CancelRun"} {
		t.Run(operation, func(t *testing.T) {
			s, path := messageStore(t, lifecycleFixture(t, []string{operation}, false))
			if operation == "Retry" {
				if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			request := committedRequest(t, s, "requester")
			auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
			if err != nil || !auth.Allowed {
				t.Fatalf("auth %+v %v", auth, err)
			}
			before := current(t, s, "worker")
			terminals := count(t, s, "terminal_obligations")
			if _, err = s.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_lifecycle BEFORE INSERT ON lifecycle_results BEGIN SELECT RAISE(ABORT,'test decision rollback'); END`); err != nil {
				t.Fatal(err)
			}
			if _, err = s.DecideLifecycle(t.Context(), request); err == nil {
				t.Fatal("expected target transaction failure")
			}
			if !reflect.DeepEqual(before, current(t, s, "worker")) || count(t, s, "runs") != 2 || count(t, s, "run_origins") != 2 || count(t, s, "lifecycle_results") != 0 || count(t, s, "terminal_obligations") != terminals || count(t, s, "attempts") != 1 {
				t.Fatal("partial target decision or lost separate authorization")
			}
			if _, err = s.conn.ExecContext(t.Context(), "DROP TRIGGER fail_lifecycle"); err != nil {
				t.Fatal(err)
			}
			if result, err := s.DecideLifecycle(t.Context(), request); err != nil || !result.Accepted {
				t.Fatalf("retry transaction %+v %v", result, err)
			}
			if _, err = s.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_settlement BEFORE INSERT ON settlements BEGIN SELECT RAISE(ABORT,'test settlement rollback'); END`); err != nil {
				t.Fatal(err)
			}
			if _, err = s.SettleLifecycle(t.Context(), request.Intent.ID, "result"); err == nil {
				t.Fatal("expected settlement transaction failure")
			}
			if count(t, s, "lifecycle_results") != 1 || count(t, s, "settlements") != 0 || count(t, s, "admissions") != 0 || count(t, s, "observations") != 0 {
				t.Fatal("settlement was partial or erased target decision")
			}
			if _, err = s.conn.ExecContext(t.Context(), "DROP TRIGGER fail_settlement"); err != nil {
				t.Fatal(err)
			}
			if _, err = s.SettleLifecycle(t.Context(), request.Intent.ID, "result"); err != nil {
				t.Fatal(err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRetryAtGenerationCapacityIsExplicitOperationalError(t *testing.T) {
	const maximum int64 = 9223372036854775807
	p := lifecycleFixture(t, []string{"Retry"}, false)
	var program Program
	if err := json.Unmarshal(p.Programs[0].Bytes, &program); err != nil {
		t.Fatal(err)
	}
	program.Code[0].Effect.Generation = Lit(Int(maximum))
	encoded, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[0] = Artifact{ID: id, Semantics: SemanticVersion, Bytes: encoded}
	p.Runs[0].ProgramID = id
	p.Subjects[1].Generation = maximum
	p.Runs[1].ID.Generation = maximum
	s, path := messageStore(t, p)
	target, err := s.Snapshot(t.Context(), RunID{"worker", maximum})
	if err != nil {
		t.Fatal(err)
	}
	target, err = s.Commit(t.Context(), target.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	request := committedRequest(t, s, "requester")
	auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
	if err != nil || !auth.Allowed {
		t.Fatalf("auth %+v %v", auth, err)
	}
	if _, err = s.DecideLifecycle(t.Context(), request); !errors.Is(err, ErrLimit) {
		t.Fatalf("generation capacity did not fail explicitly: %v", err)
	}
	after, err := s.Snapshot(t.Context(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(target, after) || count(t, s, "runs") != 2 || count(t, s, "run_origins") != 2 || count(t, s, "lifecycle_results") != 0 || count(t, s, "attempts") != 1 {
		t.Fatal("overflow partially mutated target or fabricated rejection")
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
