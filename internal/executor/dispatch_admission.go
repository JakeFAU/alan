package executor

import (
	"context"
	"iter"
)

// DispatchAdmission records a scoped adapter passing its owner's final local
// gate after the identified composition fact. The transport may never have run.
type DispatchAdmission struct {
	OperationID, PreparedDigest string
	AfterFact                   int64
}

const dispatchAdmissionDDL = `CREATE TABLE adapter_dispatch_admissions (
	intent_id TEXT PRIMARY KEY NOT NULL REFERENCES adapter_attempts(intent_id),
	prepared_digest TEXT NOT NULL,
	after_fact INTEGER NOT NULL REFERENCES composition_facts(decision_order));`

// The caller holds mu and ownerMu from the final authority check through
// registration. A failed write cannot release a transport or refund its debit.
func (s *Store) recordDispatchAdmission(ctx context.Context, intent Intent) error {
	root, err := compositionMembershipRoot(ctx, s.conn, intent.Run)
	if err != nil || root == "" {
		return err
	}
	tx, err := s.beginOwnedTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	a, err := loadAdapterAttempt(ctx, tx, intent.ID)
	if err != nil {
		return err
	}
	if !a.Allowed {
		return corrupt("dispatch without allowed adapter reservation")
	}
	var prefix int64
	if err := tx.QueryRowContext(ctx, "SELECT max(decision_order) FROM composition_facts").Scan(&prefix); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO adapter_dispatch_admissions VALUES(?,?,?)", intent.ID, a.Prepared.Digest, prefix)
	if err != nil {
		return err
	}
	var stored DispatchAdmission
	if err = tx.QueryRowContext(ctx, "SELECT intent_id,prepared_digest,after_fact FROM adapter_dispatch_admissions WHERE intent_id=?", intent.ID).Scan(&stored.OperationID, &stored.PreparedDigest, &stored.AfterFact); err != nil {
		return err
	}
	if stored != (DispatchAdmission{intent.ID, a.Prepared.Digest, prefix}) {
		return corrupt("dispatch admission was not retained")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// commitOwnedTx also takes ownerMu, which guards this entire interval.
	return s.commitExecutionTx(ctx, tx, &executionDelta{run: intent.Run})
}

// Several admissions may observe the same prefix. Their own insertion order
// must nevertheless agree with the monotone composition journal.
func indexDispatchAdmissions(rows []map[string]any, last int) (map[int64][]DispatchAdmission, error) {
	byPrefix := make(map[int64][]DispatchAdmission)
	seen := make(map[string]bool)
	var previous int64
	for _, row := range rows {
		d := DispatchAdmission{inspectText(row, "intent_id"), inspectText(row, "prepared_digest"), inspectInt(row, "after_fact")}
		if d.OperationID == "" || seen[d.OperationID] || !digestIdentity(d.PreparedDigest) || d.AfterFact < 1 || d.AfterFact > int64(last) || d.AfterFact < previous {
			return nil, corrupt("invalid dispatch admission prefix")
		}
		byPrefix[d.AfterFact] = append(byPrefix[d.AfterFact], d)
		seen[d.OperationID] = true
		previous = d.AfterFact
	}
	return byPrefix, nil
}

func replayDispatchAdmissions(ctx context.Context, q queryer, enabled bool, last int) (map[int64][]DispatchAdmission, error) {
	if !enabled {
		return nil, nil
	}
	rows, err := scanInspectionTable(ctx, q, "adapter_dispatch_admissions", true)
	if err != nil {
		return nil, err
	}
	return indexDispatchAdmissions(rows, last)
}

func (x *compositionPrefix) checkDispatchAdmission(ctx context.Context, q queryer, d DispatchAdmission) error {
	op := d.OperationID
	i, ok := x.intents[op]
	if !ok || i.Kind != "adapter" || !x.reservations[op] || x.known[op] || x.scopes[i.Run].RootID == "" || !x.active[i.Run] || x.retired[i.Run] || x.states[i.Run].Status != "open" || x.closed(i.Run) {
		return corrupt("dispatch admission outside original authority")
	}
	a, err := loadAdapterAttempt(ctx, q, op)
	if err != nil {
		return err
	}
	if !a.Allowed || d.PreparedDigest != a.Prepared.Digest {
		return corrupt("dispatch admission differs from reservation")
	}
	return nil
}

func requireDispatchForResults(ctx context.Context, q queryer) error {
	var missing bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM adapter_results r
		JOIN composition_charges c ON c.operation_id=r.intent_id AND c.kind='effect'
		LEFT JOIN adapter_dispatch_admissions a ON a.intent_id=r.intent_id
		WHERE a.intent_id IS NULL)`).Scan(&missing)
	if err == nil && missing {
		return corrupt("result without its dispatch admission")
	}
	return err
}

// The single runner retains a returned result before launching another worker
// in that slot. A lost response has no retained finish time, so its admission
// proves occupancy only at that instant.
func checkDispatchCapacity(gates map[int64][]DispatchAdmission, last int, roots []compositionRoot, facts iter.Seq[compositionFact]) error {
	if len(gates) == 0 {
		return nil
	}
	limit := map[string]int64{}
	for _, root := range roots {
		limit[root.ID] = root.Limits.MaxInFlight
	}
	rootOf := map[string]string{}
	resultAt := map[string]int64{}
	for f := range facts {
		switch f.Kind {
		case compositionReservationFact:
			if len(f.RootIDs) == 1 {
				rootOf[f.OperationID] = f.RootIDs[0]
			}
		case compositionExternalResultFact:
			resultAt[f.OperationID] = f.Order
		}
	}
	active := map[string]int64{}
	release := map[int64][]string{}
	for prefix := int64(1); prefix <= int64(last); prefix++ {
		for _, root := range release[prefix] {
			active[root]--
		}
		for _, gate := range gates[prefix] {
			root := rootOf[gate.OperationID]
			if active[root] >= limit[root] {
				return corrupt("dispatch admission exceeds in-flight allowance")
			}
			if end := resultAt[gate.OperationID]; end > prefix {
				active[root]++
				release[end] = append(release[end], root)
			}
		}
	}
	return nil
}
