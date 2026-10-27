package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

type RunOptions struct{ MaxSteps, MaxDispatch int }
type RunReport struct {
	Steps, Dispatches  int
	Quiescent          bool
	OpenRuns           []RunID
	UnresolvedIntents  []Intent
	PendingDeliveries  []string
	UnknownExternal    []string
	RejectedAdmissions int
}
type readyRun struct {
	authority   Authority
	observation string
}
type pendingDelivery struct {
	key, intent string
	kind        string
	source      RunID
	destination TerminalDestination
}

// Run automatically schedules finite ordinary commits and committed deliveries.
// Quiescence means no supported eligible work now, not completion or liveness.
func (s *Store) Run(ctx context.Context, options RunOptions) (RunReport, error) {
	report := RunReport{}
	if options.MaxSteps <= 0 || options.MaxDispatch <= 0 {
		return report, fmt.Errorf("runner budgets must be positive: %w", ErrLimit)
	}
	var last *RunID
	attemptedExternal := map[string]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return s.finishRun(ctx, report, false, err)
		}
		pending, err := s.pendingDeliveries(ctx)
		if err != nil {
			return report, err
		}
		filtered := pending[:0]
		for _, d := range pending {
			if d.kind != "external" || !attemptedExternal[d.intent] {
				filtered = append(filtered, d)
			}
		}
		pending = filtered
		if len(pending) > 0 {
			if report.Dispatches >= options.MaxDispatch {
				return s.finishRun(ctx, report, false, fmt.Errorf("dispatch budget exhausted: %w", ErrLimit))
			}
			delivery := pending[0]
			if delivery.kind == "external" {
				attemptedExternal[delivery.intent] = true
				_, err = s.DispatchExternal(ctx, delivery.intent)
			} else if delivery.kind == "lifecycle" {
				var auth Authorization
				auth, err = s.AuthorizeIntent(ctx, delivery.intent)
				if err == nil && auth.Allowed {
					var request LifecycleRequest
					request, err = s.LoadLifecycleRequest(ctx, delivery.intent)
					if err == nil {
						_, err = s.DecideLifecycle(ctx, request)
					}
				}
				if err == nil {
					_, err = s.SettleLifecycle(ctx, delivery.intent, delivery.key)
				}
			} else if delivery.intent != "" {
				if _, err = s.DeliverMessage(ctx, delivery.intent, delivery.key); err == nil {
					err = s.AcknowledgeMessage(ctx, delivery.intent, delivery.key)
				}
			} else {
				if _, err = s.DeliverTerminal(ctx, delivery.source, delivery.destination, delivery.key); err == nil {
					err = s.AcknowledgeTerminal(ctx, delivery.source, delivery.destination, delivery.key)
				}
			}
			if err != nil {
				return s.finishRun(ctx, report, false, err)
			}
			report.Dispatches++
			continue
		}
		ready, err := s.selectReady(ctx, last)
		if err != nil {
			return s.finishRun(ctx, report, false, err)
		}
		if ready == nil {
			return s.finishRun(ctx, report, true, nil)
		}
		if report.Steps >= options.MaxSteps {
			return s.finishRun(ctx, report, false, fmt.Errorf("execution budget exhausted: %w", ErrLimit))
		}
		next, err := s.Commit(ctx, ready.authority, ready.observation)
		if err != nil {
			return s.finishRun(ctx, report, false, err)
		}
		report.Steps++
		id := next.ID
		last = &id
	}
}
func (s *Store) selectReady(ctx context.Context, last *RunID) (*readyRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, sql.ErrConnDone
	}
	ids, err := readRunIDs(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	start := 0
	if last != nil {
		for i, id := range ids {
			if id == *last {
				start = i + 1
				break
			}
		}
	}
	observations, err := readObservations(ctx, s.conn, s.limits)
	if err != nil {
		return nil, err
	}
	observationIDs := make([]string, 0, len(observations))
	for id := range observations {
		observationIDs = append(observationIDs, id)
	}
	sort.Strings(observationIDs)
	for i := 0; i < len(ids); i++ {
		id := ids[(start+i)%len(ids)]
		run, err := loadRun(ctx, s.conn, id)
		if err != nil {
			return nil, err
		}
		var generation int64
		var retired bool
		if err = s.conn.QueryRowContext(ctx, "SELECT generation,retired FROM subjects WHERE id=?", id.Subject).Scan(&generation, &retired); err != nil {
			return nil, err
		}
		if run.Projection.Status != "open" || retired || generation != id.Generation {
			continue
		}
		p, err := loadProgram(ctx, s.conn, run.ProgramID, s.limits)
		if err != nil {
			return nil, err
		}
		if p.Code[run.Projection.PC].Op != "receive" {
			if _, err = Reduce(p, run.Projection, nil, id, run.Sequence+1, s.limits); err != nil {
				return nil, err
			}
			return &readyRun{authority: run.Authority()}, nil
		}
		for _, observationID := range observationIDs {
			observation := observations[observationID]
			if observation.Run != id || observation.consumed.Valid {
				continue
			}
			_, err = Reduce(p, run.Projection, &observation.Observation, id, run.Sequence+1, s.limits)
			if errors.Is(err, ErrIneligible) {
				continue
			}
			if err != nil {
				return nil, err
			}
			return &readyRun{authority: run.Authority(), observation: observationID}, nil
		}
	}
	return nil, nil
}
func (s *Store) pendingDeliveries(ctx context.Context) ([]pendingDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, sql.ErrConnDone
	}
	return pendingDeliveries(ctx, s.conn)
}
func pendingDeliveries(ctx context.Context, q queryer) ([]pendingDelivery, error) {
	pending := []pendingDelivery{}
	intents, err := readIntents(ctx, q)
	if err != nil {
		return nil, err
	}
	for id, intent := range intents {
		if intent.Kind == "external" {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("external-outcome", id)).Scan(&count); err != nil {
				return nil, err
			}
			if count == 0 {
				pending = append(pending, pendingDelivery{key: identity("external-delivery", id), intent: id, kind: "external"})
			}
			continue
		}
		if intent.Kind == "lifecycle" {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("lifecycle-outcome", id)).Scan(&count); err != nil {
				return nil, err
			}
			if count == 0 {
				pending = append(pending, pendingDelivery{key: identity("lifecycle-delivery", id), intent: id, kind: "lifecycle"})
			}
			continue
		}
		if intent.Kind != "message" {
			continue
		}
		var count int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM message_acknowledgments WHERE intent_id=?", id).Scan(&count); err != nil {
			return nil, err
		}
		if count == 0 {
			pending = append(pending, pendingDelivery{key: identity("message-delivery", id), intent: id})
		}
	}
	rows, err := q.QueryContext(ctx, "SELECT subject,generation,sequence,destinations,delivery_status FROM terminal_obligations ORDER BY subject,generation")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var source RunID
		var sequence int64
		var raw, status []byte
		if err = rows.Scan(&source.Subject, &source.Generation, &sequence, &raw, &status); err != nil {
			return nil, err
		}
		var destinations []TerminalDestination
		var delivered map[string]bool
		if err = decode(raw, &destinations); err != nil {
			return nil, err
		}
		if err = decode(status, &delivered); err != nil {
			return nil, err
		}
		for _, destination := range destinations {
			if !delivered[destination.key()] {
				pending = append(pending, pendingDelivery{key: identity("terminal-delivery", source.Subject, source.Generation, sequence, destination.Target, destination.Obligation), source: source, destination: destination})
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].key < pending[j].key })
	return pending, nil
}
func (s *Store) finishRun(ctx context.Context, report RunReport, quiescent bool, runErr error) (RunReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report.Quiescent = quiescent
	report.OpenRuns = []RunID{}
	report.UnresolvedIntents = []Intent{}
	report.PendingDeliveries = []string{}
	report.UnknownExternal = []string{}
	if s.closed {
		return report, errors.Join(runErr, sql.ErrConnDone)
	}
	ids, err := readRunIDs(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	for _, id := range ids {
		run, err := loadRun(ctx, s.conn, id)
		if err != nil {
			return report, errors.Join(runErr, err)
		}
		if run.Projection.Status == "open" {
			report.OpenRuns = append(report.OpenRuns, id)
		}
	}
	intents, err := readIntents(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	for id, intent := range intents {
		var count int
		if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM message_acknowledgments WHERE intent_id=?", id).Scan(&count); err != nil {
			return report, errors.Join(runErr, err)
		}
		if count == 0 && intent.Kind == "lifecycle" {
			if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("lifecycle-outcome", id)).Scan(&count); err != nil {
				return report, errors.Join(runErr, err)
			}
		}
		if count == 0 && intent.Kind == "external" {
			if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("external-outcome", id)).Scan(&count); err != nil {
				return report, errors.Join(runErr, err)
			}
		}
		if count == 0 && intent.Kind == "external" {
			a, e := loadExternalAttempt(ctx, s.conn, id)
			if e == nil && a.Allowed {
				report.UnknownExternal = append(report.UnknownExternal, id)
			} else if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return report, errors.Join(runErr, e)
			}
		}
		if count == 0 {
			report.UnresolvedIntents = append(report.UnresolvedIntents, intent.Intent)
		}
	}
	sort.Slice(report.UnresolvedIntents, func(i, j int) bool { return report.UnresolvedIntents[i].ID < report.UnresolvedIntents[j].ID })
	sort.Strings(report.UnknownExternal)
	pending, err := pendingDeliveries(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	for _, d := range pending {
		report.PendingDeliveries = append(report.PendingDeliveries, d.key)
	}
	rows, err := s.conn.QueryContext(ctx, "SELECT result FROM admissions")
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return report, errors.Join(runErr, err)
		}
		var result AdmissionResult
		if err = decode(raw, &result); err != nil {
			return report, errors.Join(runErr, err)
		}
		if !result.Accepted {
			report.RejectedAdmissions++
		}
	}
	return report, errors.Join(runErr, rows.Err())
}
