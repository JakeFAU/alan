package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// PrepareRoot distinguishes the public structured outcome from the root run's
// local terminal projection. Unknown descendants and effects keep success pending.
func (s *Store) PrepareRoot(ctx context.Context, rootID string) (InvocationPreparation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return InvocationPreparation{}, sql.ErrConnDone
	}
	if !s.invocationEnabled() {
		return InvocationPreparation{}, ErrVersion
	}
	return prepareRoot(ctx, s.conn, rootID)
}
func prepareRoot(ctx context.Context, q queryer, rootID string) (InvocationPreparation, error) {
	var root compositionRoot
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_roots WHERE id=?", rootID).Scan(&raw); err != nil {
		return InvocationPreparation{}, err
	}
	if err := decode(raw, &root); err != nil {
		return InvocationPreparation{}, err
	}
	scope := "root:" + rootID
	p := InvocationPreparation{Outcome: compositionOutcome{InvocationID: scope, OperationID: scope, Caller: root.Run}}
	err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_root_publications WHERE scope_id=?", scope).Scan(&raw)
	if err == nil {
		if err = decode(raw, &p.Outcome); err != nil {
			return p, err
		}
		p.Ready = true
		return p, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	return prepareCompositionScope(ctx, q, p, scope, false)
}

// SettleRoot durably publishes the original root outcome when its join is known.
func (s *Store) SettleRoot(ctx context.Context, rootID string) (InvocationPreparation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return InvocationPreparation{}, sql.ErrConnDone
	}
	if !s.invocationEnabled() {
		return InvocationPreparation{}, ErrVersion
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return InvocationPreparation{}, err
	}
	defer tx.Rollback()
	p, err := prepareRoot(ctx, tx, rootID)
	if err != nil || !p.Ready || p.Outcome.ID != "" {
		return p, err
	}
	p.Outcome.ID = compositionIdentity("alan/root-outcome/v1", rootID)
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(decision_order),0)+1 FROM composition_facts").Scan(&p.Outcome.DecisionOrder); err != nil {
		return p, err
	}
	raw, err := json.Marshal(p.Outcome)
	if err != nil {
		return p, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO composition_root_publications VALUES(?,?,?,?)", p.Outcome.InvocationID, rootID, p.Outcome.DecisionOrder, raw); err != nil {
		return p, err
	}
	fact, err := compositionProtocolEvidence(ctx, tx, compositionOutcomeFact, p.Outcome.OperationID)
	if err != nil {
		return p, err
	}
	if _, err = appendCompositionFact(ctx, tx, fact); err != nil {
		return p, err
	}
	return p, s.commitOwnedTx(ctx, tx)
}
func (s *Store) settleRoots(ctx context.Context) (bool, error) {
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
