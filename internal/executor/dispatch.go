package executor

import (
	"context"
	"database/sql"
	"errors"

	"github.com/JakeFAU/alan/internal/boundary"
)

// DispatchAuthority identifies one retained source generation in a Store. With
// OwningScope set it selects the source's immediate scope and its descendants.
type DispatchAuthority struct {
	Namespace   string
	Run         RunID
	OwningScope bool
}

var ErrDispatchAuthorityOpen = errors.New("dispatch authority is still open")

type dispatchOpportunity struct {
	source RunID
	done   chan struct{}
}

// Add and Close's prohibition on further Add calls serialize via ownerMu.
// The caller releases its pin on every exit, after its last Store operation.
func (s *Store) pinDispatchOwner() (<-chan struct{}, error) {
	s.ownerMu.Lock()
	defer s.ownerMu.Unlock()
	if s.closing {
		return nil, sql.ErrConnDone
	}
	if s.ownerClosing == nil {
		s.ownerClosing = make(chan struct{})
	}
	s.ownerUsers.Add(1)
	return s.ownerClosing, nil
}

func (s *Store) dispatchOwnerOpen() bool {
	s.ownerMu.Lock()
	defer s.ownerMu.Unlock()
	return !s.closing
}

func dispatchEvent(ctx context.Context, event boundary.Event, name, phase string) error {
	event.Boundary = name
	event.Phase = phase
	return boundary.Reached(ctx, event)
}

// An allowed reservation reaches this gate for each distinct physical exposure.
func (s *Store) admitDispatch(ctx context.Context, event boundary.Event) (*dispatchOpportunity, error) {
	if err := dispatchEvent(ctx, event, "dispatch.consider", "pre-admission"); err != nil {
		return nil, err
	}
	op, reason, err := s.registerDispatch(ctx, event.Intent)
	if err == nil && reason != "" {
		err = dispatchEvent(ctx, event, "dispatch.refused", reason)
	}
	return op, err
}

func (s *Store) registerDispatch(ctx context.Context, id string) (*dispatchOpportunity, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.dispatchOwnerOpen() {
		return nil, "", sql.ErrConnDone
	}
	intent, err := loadIntent(ctx, s.conn, id)
	if err != nil {
		return nil, "", err
	}
	reason, err := s.dispatchSourceReason(ctx, intent.Run, intent.revision)
	if err != nil || reason != "" {
		return nil, reason, err
	}
	// Keep Close and durable R outside the final check-and-register interval.
	s.ownerMu.Lock()
	defer s.ownerMu.Unlock()
	if s.closing {
		return nil, "", sql.ErrConnDone
	}
	if err = ctx.Err(); err != nil {
		return nil, "", err
	}
	op := &dispatchOpportunity{source: intent.Run, done: make(chan struct{})}
	if s.liveDispatch == nil {
		s.liveDispatch = make(map[*dispatchOpportunity]struct{})
	}
	s.liveDispatch[op] = struct{}{}
	return op, "", nil
}

// Caller holds mu. A zero original revision selects the entire RunID for Q.
func (s *Store) dispatchSourceReason(ctx context.Context, id RunID, originalRevision int64) (string, error) {
	run, err := loadRun(ctx, s.conn, id)
	if err != nil {
		return "", err
	}
	var generation int64
	var retired bool
	if err = s.conn.QueryRowContext(ctx, "SELECT generation,retired FROM subjects WHERE id=?", id.Subject).Scan(&generation, &retired); err != nil {
		return "", err
	}
	switch {
	case generation != id.Generation:
		return "source_generation_changed", nil
	case originalRevision != 0 && originalRevision != run.Revision:
		return "source_revision_changed", nil
	case retired:
		return "source_retired", nil
	case run.Projection.Status != "open":
		return "source_closed", nil
	}
	if s.invocationEnabled() {
		root, err := compositionMembershipRoot(ctx, s.conn, id)
		if err != nil {
			return "", err
		}
		if root != "" {
			closed, err := compositionScopeClosed(ctx, s.conn, id)
			if err != nil {
				return "", err
			}
			if closed {
				return "scope_closed", nil
			}
		}
	}
	return "", nil
}

func (s *Store) finishDispatch(op *dispatchOpportunity, handlingErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rememberDispatchFailure(op.source, handlingErr)
	delete(s.liveDispatch, op)
	close(op.done)
}

// Caller holds mu. Keep the first failure for the original source, even if
// service later succeeds or another caller is concurrently observing Q.
func (s *Store) rememberDispatchFailure(source RunID, handlingErr error) {
	if handlingErr != nil && s.dispatchFailures[source] == nil {
		if s.dispatchFailures == nil {
			s.dispatchFailures = make(map[RunID]error)
		}
		s.dispatchFailures[source] = handlingErr
	}
}

type revokedDispatch struct {
	source RunID
	scope  string
}

// Caller holds mu. Scope revocation is independent of the alias member's
// terminal status. Its immutable binding identifies the actual scope.
func (s *Store) revokedDispatch(ctx context.Context, target DispatchAuthority) (revokedDispatch, error) {
	match := revokedDispatch{source: target.Run}
	if target.Namespace != s.namespace || target.Run.Subject == "" || target.Run.Generation <= 0 {
		return match, ErrStale
	}
	if !target.OwningScope {
		reason, err := s.dispatchSourceReason(ctx, target.Run, 0)
		if err == nil && reason == "" {
			err = ErrDispatchAuthorityOpen
		}
		return match, err
	}
	if _, err := loadRun(ctx, s.conn, target.Run); err != nil {
		return match, err
	}
	if !s.invocationEnabled() {
		return match, ErrStale
	}
	root, err := compositionMembershipRoot(ctx, s.conn, target.Run)
	if err != nil {
		return match, err
	}
	if root == "" {
		return match, ErrStale
	}
	scope, err := compositionScopeFor(ctx, s.conn, target.Run)
	if err != nil {
		return match, err
	}
	match.scope = scope.ID
	closed, err := compositionScopeClosed(ctx, s.conn, target.Run)
	if err == nil && !closed {
		err = ErrDispatchAuthorityOpen
	}
	return match, err
}

func (m revokedDispatch) covers(ctx context.Context, q queryer, source RunID) (bool, error) {
	if m.scope == "" {
		return source == m.source, nil
	}
	root, err := compositionMembershipRoot(ctx, q, source)
	if err != nil || root == "" {
		return false, err
	}
	if m.scope == "root:"+root {
		return true, nil
	}
	seen := map[RunID]bool{}
	for !seen[source] {
		seen[source] = true
		b, _, err := loadCompositionBinding(ctx, q, source)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if b.InvocationID == m.scope {
			return true, nil
		}
		source = b.Caller
	}
	return false, ErrCorrupt
}

type recordedDispatch struct {
	id, kind string
	source   RunID
}

func (s *Store) dispatchDrainState(ctx context.Context, target DispatchAuthority) (<-chan struct{}, []recordedDispatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dispatchOwnerOpen() {
		return nil, nil, sql.ErrConnDone
	}
	match, err := s.revokedDispatch(ctx, target)
	if err != nil {
		return nil, nil, err
	}
	for source, failure := range s.dispatchFailures {
		ok, err := match.covers(ctx, s.conn, source)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return nil, nil, failure
		}
	}
	for op := range s.liveDispatch {
		ok, err := match.covers(ctx, s.conn, op.source)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return op.done, nil, nil
		}
	}
	// Only original, already-recorded results are serviceable here. Neither a
	// prior reservation nor an absent remote receipt permits any recovery call.
	rows, err := s.conn.QueryContext(ctx, `SELECT i.id,i.subject,i.generation,e.kind
FROM (SELECT intent_id,'external' AS kind FROM external_receipts UNION ALL
      SELECT intent_id,'adapter' AS kind FROM adapter_results) e JOIN intents i ON i.id=e.intent_id
WHERE NOT EXISTS (SELECT 1 FROM settlements s WHERE s.intent_id=i.id)
AND (? OR (i.subject=? AND i.generation=?)) ORDER BY i.id`, match.scope != "", match.source.Subject, match.source.Generation)
	if err != nil {
		return nil, nil, err
	}
	var recorded []recordedDispatch
	for rows.Next() {
		var r recordedDispatch
		if err = rows.Scan(&r.id, &r.source.Subject, &r.source.Generation, &r.kind); err != nil {
			break
		}
		recorded = append(recorded, r)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return nil, nil, err
	}
	matched := recorded[:0]
	for _, r := range recorded {
		ok, err := match.covers(ctx, s.conn, r.source)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			matched = append(matched, r)
		}
	}
	return nil, matched, nil
}

// Recovery has no volatile worker to hold Q off. Bind the durable result,
// attempt settlement, and remember any genuine accounting failure in the same
// critical section as Q's inspection. A missing result or an interrupted
// caller may leave the original obligation pending for another observer.
func (s *Store) settleRecordedDispatch(ctx context.Context, r recordedDispatch) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false, sql.ErrConnDone
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var handlingErr error
	switch r.kind {
	case "external":
		outcome, err := externalOutcome(ctx, s.conn, r.id)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if outcome.Attempt.Intent.Run != r.source {
			return false, ErrConflict
		}
		handlingErr = s.settleExternalLocked(ctx, outcome)
		// A synthesized first-permission denial has no returned receipt.
		if outcome.Receipt == nil {
			return true, handlingErr
		}
	case "adapter":
		attempt, err := loadAdapterAttempt(ctx, s.conn, r.id)
		if err != nil {
			return false, err
		}
		if attempt.Intent.Run != r.source {
			return false, ErrConflict
		}
		result, err := loadAdapterResult(ctx, s.conn, r.id)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if err = validateAdapterResult(attempt, result, s.limits); err != nil {
			return false, err
		}
		handlingErr = s.settleAdapterLocked(ctx, attempt, result)
	default:
		return false, ErrCorrupt
	}
	if handlingErr != nil && !errors.Is(handlingErr, ctx.Err()) {
		s.rememberDispatchFailure(r.source, handlingErr)
	}
	return true, handlingErr
}

// AwaitDispatchQuiescence observes an already revoked source or scope. It waits
// for covered local calls and their accounting, and settles surviving recorded
// results. Success makes no assertion about a previously transmitted request.
func (s *Store) AwaitDispatchQuiescence(ctx context.Context, target DispatchAuthority) error {
	closing, err := s.pinDispatchOwner()
	if err != nil {
		return err
	}
	defer s.ownerUsers.Done()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done, recorded, err := s.dispatchDrainState(ctx, target)
		if err != nil {
			return err
		}
		if done != nil {
			event := boundary.Event{Subject: target.Run.Subject, Generation: target.Run.Generation}
			if err := dispatchEvent(ctx, event, "dispatch.waiting", "in-flight"); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-closing:
				return sql.ErrConnDone
			case <-done:
			}
			continue
		}
		if len(recorded) == 0 {
			return ctx.Err()
		}
		for _, r := range recorded {
			present, err := s.settleRecordedDispatch(ctx, r)
			if err != nil {
				return err
			}
			if !present {
				return corrupt("recorded dispatch result disappeared")
			}
		}
	}
}
