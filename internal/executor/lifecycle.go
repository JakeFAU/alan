package executor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"math"
	"reflect"
)

var ErrSourceDenied = errors.New("source authorization denied")

type LifecycleRequest struct {
	Intent           Intent
	SourceRevision   int64
	Operation        string
	ExpectedRevision int64
}
type TargetState struct {
	Run                       RunID
	ProgramID, Owner          string
	Revision, Epoch, Sequence int64
	Retired, RetryCancelled   bool
	Projection                Projection
}
type LifecycleResult struct {
	Request          LifecycleRequest
	DecisionSequence int64
	Accepted         bool
	Reason           string
	Successor        int64
	Before, After    *TargetState
}
type LifecycleOutcome struct {
	Request     LifecycleRequest
	Disposition string
	Accepted    bool
	Reason      string
	Successor   int64
}
type rootOrigin struct {
	Subject   Subject
	ProgramID string
	Initial   Projection
}

func loadSubject(ctx context.Context, q queryer, id string) (Subject, error) {
	s := Subject{ID: id}
	err := q.QueryRowContext(ctx, "SELECT generation,owner,revision,retired,retry_cancelled FROM subjects WHERE id=?", id).Scan(&s.Generation, &s.Owner, &s.Revision, &s.Retired, &s.RetryCancelled)
	return s, err
}
func loadLifecycleRequest(ctx context.Context, q queryer, id string) (LifecycleRequest, error) {
	retained, err := loadIntent(ctx, q, id)
	if err != nil {
		return LifecycleRequest{}, err
	}
	intent := retained.Intent
	if intent.Kind != "lifecycle" || intent.Generation <= 0 || intent.Target == "" || intent.Obligation == "" {
		return LifecycleRequest{}, fmt.Errorf("invalid lifecycle intent")
	}
	operation, ok := intent.Payload["operation"]
	if !ok || operation.Kind != "string" {
		return LifecycleRequest{}, fmt.Errorf("missing lifecycle operation")
	}
	switch operation.Text {
	case "Retry", "CancelRun", "AbandonRun", "RetireSubject":
	default:
		return LifecycleRequest{}, fmt.Errorf("unknown lifecycle operation")
	}
	revision, ok := intent.Payload["revision"]
	if !ok || revision.Kind != "int" || revision.Number <= 0 {
		return LifecycleRequest{}, fmt.Errorf("invalid expected target revision")
	}
	return LifecycleRequest{Intent: intent, SourceRevision: retained.revision, Operation: operation.Text, ExpectedRevision: revision.Number}, nil
}
func (s *Store) LoadLifecycleRequest(ctx context.Context, id string) (LifecycleRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return LifecycleRequest{}, sql.ErrConnDone
	}
	return loadLifecycleRequest(ctx, s.conn, id)
}
func loadAuthorization(ctx context.Context, q queryer, id string) (Authorization, error) {
	var raw []byte
	var auth Authorization
	if err := q.QueryRowContext(ctx, "SELECT result FROM attempts WHERE id=?", identity("authorization", id)).Scan(&raw); err != nil {
		return auth, err
	}
	if err := decode(raw, &auth); err != nil {
		return auth, err
	}
	if auth.IntentID != id || auth.ID != identity("authorization", id) || !sameJSON(raw, auth) {
		return auth, ErrCorrupt
	}
	return auth, nil
}
func loadLifecycleResult(ctx context.Context, q queryer, id string) (LifecycleResult, error) {
	var descriptor, raw []byte
	var order int64
	var result LifecycleResult
	if err := q.QueryRowContext(ctx, "SELECT decision_sequence,descriptor,result FROM lifecycle_results WHERE id=?", id).Scan(&order, &descriptor, &raw); err != nil {
		return result, err
	}
	if err := decode(raw, &result); err != nil {
		return result, err
	}
	if result.Request.Intent.ID != id || result.DecisionSequence != order || !sameJSON(raw, result) || !sameJSON(descriptor, result.Request) {
		return result, ErrCorrupt
	}
	return result, nil
}
func targetState(ctx context.Context, q queryer, subject Subject) (TargetState, error) {
	run, err := loadRun(ctx, q, RunID{subject.ID, subject.Generation})
	if err != nil {
		return TargetState{}, err
	}
	return TargetState{Run: run.ID, ProgramID: run.ProgramID, Owner: subject.Owner, Revision: subject.Revision, Epoch: run.Epoch, Sequence: run.Sequence, Retired: subject.Retired, RetryCancelled: subject.RetryCancelled, Projection: run.Projection}, nil
}
func loadRoot(ctx context.Context, q queryer, subject string) (rootOrigin, []byte, error) {
	var raw []byte
	var root rootOrigin
	err := q.QueryRowContext(ctx, "SELECT descriptor FROM run_origins WHERE subject=? AND kind='root'", subject).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		enabled, e := compositionProtocol(ctx, q)
		if e != nil {
			return root, nil, e
		}
		if enabled {
			var m compositionMemberOrigin
			if e = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_members WHERE subject=?", subject).Scan(&raw); e != nil {
				return root, nil, e
			}
			if e = decode(raw, &m); e != nil {
				return root, nil, e
			}
			root = rootOrigin{Subject: Subject{ID: m.Run.Subject, Generation: 1, Owner: m.Owner, Revision: 1}, ProgramID: m.ProgramID, Initial: m.Initial}
			raw, e = json.Marshal(root)
			return root, raw, e
		}
	}
	if err != nil {
		return root, nil, err
	}
	if err := decode(raw, &root); err != nil {
		return root, nil, err
	}
	if root.Subject.ID != subject || !sameJSON(raw, root) {
		return root, nil, ErrCorrupt
	}
	return root, raw, nil
}
func lifecycleGrant(ctx context.Context, q queryer, request LifecycleRequest) (bool, error) {
	var count int
	err := q.QueryRowContext(ctx, "SELECT count(*) FROM permissions WHERE kind='lifecycle' AND source=? AND target=? AND operation=? AND revision=?", request.Intent.Run.Subject, request.Intent.Target, request.Operation, request.ExpectedRevision).Scan(&count)
	return count > 0, err
}

// decideTarget is authority policy, not application retry policy. It only evaluates
// the requested operation; ordinary programs decide whether to issue that intent.
func decideTarget(request LifecycleRequest, order int64, before *TargetState, root rootOrigin, granted bool) (LifecycleResult, error) {
	result := LifecycleResult{Request: request, DecisionSequence: order, Before: before}
	if before == nil {
		result.Reason = "target_missing"
		return result, nil
	}
	after := *before
	result.After = &after
	switch {
	case !granted:
		result.Reason = "grant_denied"
	case before.Run.Generation != request.Intent.Generation:
		result.Reason = "generation_changed"
	case before.Revision != request.ExpectedRevision:
		result.Reason = "revision_changed"
	case before.Retired:
		result.Reason = "retired"
	default:
		switch request.Operation {
		case "Retry":
			status := before.Projection.Status
			if status == "cancelled" && !before.RetryCancelled {
				result.Reason = "retry_cancelled_disabled"
				break
			}
			if status != "failed" && status != "abandoned" && status != "cancelled" {
				result.Reason = "not_retryable"
				break
			}
			if before.Run.Generation == math.MaxInt64 {
				return result, ErrLimit
			}
			result.Accepted = true
			result.Reason = "retried"
			result.Successor = before.Run.Generation + 1
			after.Run.Generation = result.Successor
			after.ProgramID = root.ProgramID
			after.Epoch = 1
			after.Sequence = 0
			after.Projection = root.Initial
		case "CancelRun", "AbandonRun", "RetireSubject":
			result.Accepted = true
			result.Reason = "already_terminal"
			if request.Operation == "RetireSubject" {
				after.Retired = true
				result.Reason = "retired"
			}
			if before.Projection.Status == "open" {
				after.Projection.Status = "cancelled"
				if request.Operation == "AbandonRun" {
					after.Projection.Status = "abandoned"
				}
				if request.Operation != "RetireSubject" {
					result.Reason = "closed"
				}
			}
		default:
			return result, fmt.Errorf("unknown lifecycle operation")
		}
	}
	return result, nil
}
func (s *Store) DecideLifecycle(ctx context.Context, request LifecycleRequest) (LifecycleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return LifecycleResult{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return LifecycleResult{}, err
	}
	defer tx.Rollback()
	existing, err := loadLifecycleResult(ctx, tx, request.Intent.ID)
	if err == nil {
		if !reflect.DeepEqual(existing.Request, request) {
			return existing, ErrConflict
		}
		return existing, s.lifecycleBoundary(ctx, tx, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return existing, err
	}
	committed, err := loadLifecycleRequest(ctx, tx, request.Intent.ID)
	if err != nil {
		return LifecycleResult{}, err
	}
	if !reflect.DeepEqual(committed, request) {
		return LifecycleResult{}, ErrConflict
	}
	auth, err := loadAuthorization(ctx, tx, request.Intent.ID)
	if err != nil {
		return LifecycleResult{}, err
	}
	if !auth.Allowed {
		return LifecycleResult{}, ErrSourceDenied
	}
	var order int64
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(decision_sequence),0) FROM lifecycle_results").Scan(&order); err != nil {
		return LifecycleResult{}, err
	}
	if order == math.MaxInt64 {
		return LifecycleResult{}, ErrLimit
	}
	order++
	subject, err := loadSubject(ctx, tx, request.Intent.Target)
	var before *TargetState
	var root rootOrigin
	var rootRaw []byte
	if err == nil {
		state, e := targetState(ctx, tx, subject)
		if e != nil {
			return LifecycleResult{}, e
		}
		before = &state
		root, rootRaw, err = loadRoot(ctx, tx, subject.ID)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return LifecycleResult{}, err
	}
	granted, err := lifecycleGrant(ctx, tx, request)
	if err != nil {
		return LifecycleResult{}, err
	}
	result, err := decideTarget(request, order, before, root, granted)
	if err == nil && s.invocationEnabled() && before != nil && request.Operation == "Retry" {
		rootID, e := compositionMembershipRoot(ctx, tx, before.Run)
		if e != nil {
			return result, e
		}
		if rootID != "" {
			after := *before
			result.Accepted = false
			result.Successor = 0
			result.Reason = "invocation_retry_unsupported"
			result.After = &after
		}
	}

	if err != nil {
		return result, err
	}
	if result.Accepted {
		if result.Successor != 0 {
			initial, e := json.Marshal(root.Initial)
			if e != nil {
				return result, e
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO runs VALUES(?,?,?,?,1,0,?)", subject.ID, result.Successor, root.ProgramID, initial, initial); err != nil {
				return result, err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO run_origins VALUES(?,?,'retry',?,?,?)", subject.ID, result.Successor, request.Intent.ID, before.Run.Generation, rootRaw); err != nil {
				return result, err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE subjects SET generation=? WHERE id=?", result.Successor, subject.ID); err != nil {
				return result, err
			}
		} else {
			if result.After.Retired != before.Retired {
				if _, err = tx.ExecContext(ctx, "UPDATE subjects SET retired=? WHERE id=?", result.After.Retired, subject.ID); err != nil {
					return result, err
				}
			}
			if before.Projection.Status == "open" && result.After.Projection.Status != "open" {
				projection, e := json.Marshal(result.After.Projection)
				if e != nil {
					return result, e
				}
				if _, err = tx.ExecContext(ctx, "UPDATE runs SET projection=? WHERE subject=? AND generation=?", projection, before.Run.Subject, before.Run.Generation); err != nil {
					return result, err
				}
				destinations, e := terminalDestinations(ctx, tx, subject.ID, subject.Revision)
				if e != nil {
					return result, e
				}
				routes, e := json.Marshal(destinations)
				if e != nil {
					return result, e
				}
				if _, err = tx.ExecContext(ctx, "INSERT INTO terminal_obligations VALUES(?,?,?,?,?,?,?,?)", subject.ID, before.Run.Generation, before.Sequence, projection, routes, []byte("{}"), request.Intent.ID, subject.Revision); err != nil {
					return result, err
				}
			}
		}
	}
	descriptor, err := json.Marshal(request)
	if err != nil {
		return result, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO lifecycle_results VALUES(?,?,?,?)", request.Intent.ID, order, descriptor, raw); err != nil {
		return result, err
	}
	if err = s.recordCompositionLifecycle(ctx, tx, result); err != nil {
		return result, err
	}
	return result, s.lifecycleBoundary(ctx, tx, result)
}
func lifecycleOutcome(ctx context.Context, q queryer, id string) (LifecycleOutcome, error) {
	request, err := loadLifecycleRequest(ctx, q, id)
	if err != nil {
		return LifecycleOutcome{}, err
	}
	auth, err := loadAuthorization(ctx, q, id)
	if err != nil {
		return LifecycleOutcome{}, err
	}
	if !auth.Allowed {
		return LifecycleOutcome{Request: request, Disposition: "source_denied", Reason: auth.Reason}, nil
	}
	result, err := loadLifecycleResult(ctx, q, id)
	if err != nil {
		return LifecycleOutcome{}, err
	}
	return LifecycleOutcome{Request: result.Request, Disposition: "target_decision", Accepted: result.Accepted, Reason: result.Reason, Successor: result.Successor}, nil
}
func lifecycleDescriptor(outcome LifecycleOutcome, transport string) AdmissionDescriptor {
	request := outcome.Request
	return AdmissionDescriptor{TransportID: transport, Kind: "lifecycle", LogicalID: request.Intent.ID, Source: RunID{request.Intent.Target, request.Intent.Generation}, Target: request.Intent.Run.Subject, Generation: request.Intent.Run.Generation, Obligation: request.Intent.Obligation, Operation: request.Intent.ID, IntentID: request.Intent.ID, Payload: map[string]Value{"accepted": {Kind: "bool", Flag: outcome.Accepted}, "reason": Text(outcome.Reason), "successor": Int(outcome.Successor), "disposition": Text(outcome.Disposition)}}
}

// SettleLifecycle retains the authenticated outcome and admits it only to its
// original execution. Late evidence is retained even if that execution is closed.
func (s *Store) SettleLifecycle(ctx context.Context, id, transport string) (AdmissionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return AdmissionResult{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer tx.Rollback()
	if transport == "" {
		return AdmissionResult{}, fmt.Errorf("empty lifecycle transport")
	}
	outcome, err := lifecycleOutcome(ctx, tx, id)
	if err != nil {
		return AdmissionResult{}, err
	}
	descriptor := lifecycleDescriptor(outcome, transport)
	if err = checkSize(descriptor, s.limits.MaxObservationBytes); err != nil {
		return AdmissionResult{}, err
	}
	raw, err := json.Marshal(descriptor)
	if err != nil {
		return AdmissionResult{}, err
	}
	existing, result, err := loadAdmission(ctx, tx, "lifecycle", transport)
	if err == nil {
		if !sameJSON(raw, existing) {
			return result, ErrConflict
		}
		return result, s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AdmissionResult{}, err
	}
	bound := outcome.Request.Intent.Run
	observation := observationFor(descriptor, bound)
	if err = insertObservation(ctx, tx, observation, s.limits); err != nil {
		return AdmissionResult{}, err
	}
	result = AdmissionResult{TransportID: transport, Accepted: true, Reason: "settled", Bound: &bound, ObservationID: observation.ID}
	resultRaw, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO admissions VALUES(?,?,?,?,?,?)", admissionKey("lifecycle", transport), "lifecycle", transport, raw, resultRaw, observation.ID); err != nil {
		return result, err
	}
	outcomeRaw, err := json.Marshal(outcome)
	if err != nil {
		return result, err
	}
	settlementID := identity("lifecycle-outcome", id)
	var prior []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM settlements WHERE id=?", settlementID).Scan(&prior)
	if err == nil {
		if !bytes.Equal(prior, outcomeRaw) {
			return result, ErrConflict
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if _, err = tx.ExecContext(ctx, "INSERT INTO settlements VALUES(?,?,?,?)", settlementID, id, outcomeRaw, observation.ID); err != nil {
			return result, err
		}
	} else {
		return result, err
	}
	return result, s.commitOwnedTx(ctx, tx)
}

func (s *Store) lifecycleBoundary(ctx context.Context, tx *sql.Tx, result LifecycleResult) error {
	source, err := loadRun(ctx, tx, result.Request.Intent.Run)
	if err != nil {
		return err
	}
	if err = s.commitOwnedTx(ctx, tx); err != nil {
		return err
	}
	phase := "rejected"
	if result.Accepted {
		phase = "accepted"
	}
	return boundary.Reached(ctx, boundary.Event{Boundary: "lifecycle.decided", Phase: phase, Subject: source.ID.Subject, Generation: source.ID.Generation, Epoch: source.Epoch, Sequence: source.Sequence, Intent: result.Request.Intent.ID, Attempt: identity("authorization", result.Request.Intent.ID)})
}
