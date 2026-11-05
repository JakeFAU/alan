package executor

import (
	"context"
	"database/sql"
	"errors"
	"sort"
)

// p3BaselineSelectReady is the unchanged selector body from production baseline
// 926a899b651fccc48be52a054612a92afcf1ddfd, internal/executor/runner.go.
// Original function SHA-256: e542cad01b791114ed6f53c4fa94df05a6e1387cb8395c93a54f1ccc3caca3b3.
// Only its name changes. It must never call a candidate reuse helper.
func (s *Store) p3BaselineSelectReady(ctx context.Context, last *RunID) (*readyRun, error) {
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
