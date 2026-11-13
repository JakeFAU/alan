// Frozen from ef38ca47179d86e7544885471ae43e86e1d6351e. Independent traversal oracles; do not optimize.
package executor

import (
	"context"
	"database/sql"
	"errors"
	"sort"
)

func (s *Store) staticBaselineSelectReady(ctx context.Context, last *RunID) (*readyRun, error) {
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
	// Retained programs and validation context are fixed under the supported
	// Store ownership model during this call. Keep only unbound programs here;
	// handoff binding mutates nested program data and must retain its own load.
	var programs map[string]Program
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
		p, ok := programs[run.ProgramID]
		if !ok {
			p, err = loadProgram(ctx, s.conn, run.ProgramID, s.limits)
			if err != nil {
				return nil, err
			}
			if !hasHandoffPeerPort(p) {
				if programs == nil {
					programs = make(map[string]Program)
				}
				programs[run.ProgramID] = p
			}
		}
		p, err = boundCompositionProgram(ctx, s.conn, p, id, s.limits)
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

func staticBaselinePendingDeliveries(ctx context.Context, q queryer) ([]pendingDelivery, error) {
	pending := []pendingDelivery{}
	intents, err := readIntents(ctx, q)
	if err != nil {
		return nil, err
	}
	for id, intent := range intents {
		if intent.Kind == "invocation" {
			var n int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_outcome_deliveries WHERE operation_id=?", id).Scan(&n); err != nil {
				return nil, err
			}
			if n == 0 {
				prepared, e := prepareInvocation(ctx, q, id)
				if e != nil && !errors.Is(e, sql.ErrNoRows) {
					return nil, e
				}
				if errors.Is(e, sql.ErrNoRows) || prepared.Ready {
					pending = append(pending, pendingDelivery{key: identity("invocation-delivery", id), intent: id, kind: "invocation"})
				}
			}
			continue
		}
		if intent.Kind == "adapter" {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("adapter-outcome", id)).Scan(&count); err != nil {
				return nil, err
			}
			if count == 0 {
				pending = append(pending, pendingDelivery{key: identity("adapter-delivery", id), intent: id, kind: "adapter"})
			}
			continue
		}
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

func (s *Store) staticBaselineSettleRoots(ctx context.Context) (bool, error) {
	if !s.invocationEnabled() {
		return false, nil
	}
	c, err := s.composition.descriptor()
	if err != nil {
		return false, err
	}
	progressed := false
	for _, root := range c.Roots {
		before, err := s.PrepareRoot(ctx, root.ID)
		if err != nil {
			return false, err
		}
		if before.Ready && before.Outcome.ID == "" {
			if _, err = s.SettleRoot(ctx, root.ID); err != nil {
				return false, err
			}
			progressed = true
		}
	}
	return progressed, nil
}
