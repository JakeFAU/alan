package executor

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
)

func validateOrdinalBinding(ctx context.Context, q queryer, namespace string, a AdapterAttempt, o receiptReadOrdinal) error {
	want, reason, err := bindReceiptRead(ctx, q, namespace, a, o.Query.Ordinal)
	if err != nil {
		return err
	}
	if reason != "" || !a.Allowed || o.Query.Check() != nil || o.Query.Ordinal < 1 || o.Query.Ordinal > 2 {
		return corrupt("unauthorized receipt read ordinal")
	}
	current, err := loadRun(ctx, q, a.Intent.Run)
	if err != nil {
		return err
	}
	auth, err := loadAuthorization(ctx, q, a.Intent.ID)
	if err != nil {
		return err
	}
	if o.Resolver.Run != a.Intent.Run || o.Resolver.Revision != auth.Source.Revision || o.Resolver.Epoch < auth.Source.Epoch || o.Resolver.Epoch > current.Epoch || o.Resolver.Sequence < auth.Source.Sequence || o.Resolver.Sequence > current.Sequence || o.Prefix <= 0 {
		return corrupt("receipt read source authority")
	}
	want.Resolver = o.Resolver
	want.Prefix = o.Prefix
	want.OriginalTerminalSequence = o.OriginalTerminalSequence
	if !reflect.DeepEqual(o, want) {
		return corrupt("receipt read target binding")
	}
	return nil
}

const compositionReadFact compositionFactKind = "receipt_read"

func receiptReadFact(ctx context.Context, q queryer, id string, n int) (compositionFact, error) {
	o, err := readOrdinal(ctx, q, id, n)
	if err != nil {
		return compositionFact{}, err
	}
	root, err := compositionMembershipRoot(ctx, q, o.Resolver.Run)
	if err != nil {
		return compositionFact{}, err
	}
	if root == "" {
		return compositionFact{}, ErrCorrupt
	}
	return compositionFact{Version: compositionFactVersion, Kind: compositionReadFact, ID: compositionIdentity("alan/fact/receipt-read/v1", []any{id, n}), RootIDs: []string{root}, Run: o.Resolver.Run, Sequence: int64(n), OperationID: id, RecordDigest: compositionIdentity("alan/receipt-read-ordinal/v1", o)}, nil
}

func resultOrder(ctx context.Context, q queryer, id string) (int64, error) {
	var seq int64
	err := q.QueryRowContext(ctx, "SELECT decision_order FROM composition_facts WHERE id=?", compositionIdentity("alan/fact/external_result/v1", id)).Scan(&seq)
	return seq, err
}

func verifyQueryResult(ctx context.Context, q queryer, a AdapterAttempt, r AdapterResult, ordinals []receiptReadOrdinal) error {
	if r.Reconciled != nil {
		return corrupt("reconstructed query")
	}
	if r.Read == nil {
		// A local refusal contains no transport metadata or assertion of world state.
		allowed := map[string]bool{"grant_denied": true, "source_denied": true, "read_denied": true, "read_exhausted": true, "target_missing": true, "target_mismatch": true, "attempt_missing": true, "original_denied": true, "unsupported_original": true, "endpoint_mismatch": true, "original_open": true, "original_busy": true, "original_unsettled": true, "unscoped": true}
		if !allowed[r.Disposition] || !reflect.DeepEqual(r.Candidate, effects.Candidate{ID: a.Intent.ID, Key: a.Prepared.Request.Key, Disposition: r.Disposition}) || r.Disposition == "read_exhausted" && len(ordinals) != 2 {
			return corrupt("read without physical ordinal or valid refusal")
		}
		return nil
	}
	if r.Read.Ordinal != len(ordinals) || len(ordinals) == 0 {
		return corrupt("read not from last spent ordinal")
	}
	o := ordinals[len(ordinals)-1]
	snapshot, err := readSnapshot(o, r.Candidate)
	if err != nil {
		return err
	}
	seq, err := resultOrder(ctx, q, a.Intent.ID)
	if err != nil {
		return err
	}
	if o.Prefix >= seq {
		return corrupt("read ordinal after result")
	}
	var prior *AdapterResult
	original, err := loadAdapterResult(ctx, q, o.Query.TargetID)
	if err == nil {
		order, e := resultOrder(ctx, q, o.Query.TargetID)
		if e != nil {
			return e
		}
		if order < seq {
			prior = &original
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	decision := decideRead(o, snapshot, prior)
	if !reflect.DeepEqual(r, readResult(a, r.Candidate, o.Query.Ordinal, decision)) {
		return corrupt("read adoption decision")
	}
	if decision == "adopted" {
		want := reconciledResult(o, *snapshot.Receipt)
		if !reflect.DeepEqual(original, want) {
			return corrupt("missing original result for imported receipt")
		}
		order, e := resultOrder(ctx, q, o.Query.TargetID)
		if e != nil || order != seq+1 {
			return corrupt("atomic receipt import order")
		}
		var outOfOrder bool
		err = q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM settlements WHERE intent_id=?) AND NOT EXISTS(SELECT 1 FROM settlements WHERE intent_id=?)", a.Intent.ID, o.Query.TargetID).Scan(&outOfOrder)
		if err != nil {
			return err
		}
		if outOfOrder {
			return corrupt("resolver settled before imported original")
		}

	}
	return nil
}

func validateReceiptLedger(ctx context.Context, q queryer, namespace string, ids []string) error {
	version, err := schemaNumber(ctx, q)
	if err != nil {
		return err
	}
	var tables int
	if err = q.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE type='table' AND name IN ('reconciliation_profile','receipt_read_ordinals')").Scan(&tables); err != nil {
		return err
	}
	if version != reconciliationSchemaVersion {
		if tables != 0 {
			return ErrVersion
		}
		return nil
	}
	if tables != 2 {
		return ErrVersion
	}
	if _, err = readReconciliationProfile(ctx, q); err != nil {
		return err
	}
	for _, id := range ids {
		a, e := loadAdapterAttempt(ctx, q, id)
		if e != nil {
			return e
		}
		r, resultErr := loadAdapterResult(ctx, q, id)
		if resultErr != nil && !errors.Is(resultErr, sql.ErrNoRows) {
			return resultErr
		}
		if a.Intent.Target != receiptReadCapability {
			if r.Read != nil {
				return corrupt("read proof on non-query")
			}
			if a.Intent.Target == receiptCounterCapability && receiptquery.CounterPrepared(a.Prepared) != nil {
				return corrupt("invalid opt-in counter preparation")
			}
			if r.Reconciled != nil {
				proof, e := loadAdapterResult(ctx, q, r.Reconciled.QueryID)
				if e != nil {
					return e
				}
				if proof.Read == nil || proof.Read.Adoption != "adopted" || proof.Read.Ordinal != r.Reconciled.Ordinal || proof.Candidate.Payload["target"].Text != id {
					return corrupt("unexplained reconciled original")
				}
			}
			var n int
			if e = q.QueryRowContext(ctx, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=?", id).Scan(&n); e != nil {
				return e
			}
			if n != 0 {
				return corrupt("mutation has receipt read allowance")
			}
			continue
		}
		p, e := (receiptquery.Reader{Endpoint: a.Prepared.Endpoint}).Prepare(a.Prepared.Request)
		if e != nil || !reflect.DeepEqual(effects.Freeze(p), a.Prepared) {
			return corrupt("invalid receipt read preparation")
		}
		var n int
		if e = q.QueryRowContext(ctx, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=?", id).Scan(&n); e != nil {
			return e
		}
		if n > 2 {
			return corrupt("read allowance exceeded")
		}
		var ordinals []receiptReadOrdinal
		for j := 1; j <= n; j++ {
			o, e := readOrdinal(ctx, q, id, j)
			if e != nil {
				return e
			}
			if e = validateOrdinalBinding(ctx, q, namespace, a, o); e != nil {
				return e
			}
			if j > 1 && (o.Resolver.Epoch <= ordinals[j-2].Resolver.Epoch || o.Prefix < ordinals[j-2].Prefix) {
				return corrupt("read retried without fresh authority")
			}
			ordinals = append(ordinals, o)
		}
		if resultErr == nil {
			if e = verifyQueryResult(ctx, q, a, r, ordinals); e != nil {
				return e
			}
		}
	}
	return nil
}

// Validate each admission against the ordered composition prefix, rather than
// assuming an eligible resolver at reopen implies eligibility when A occurred.
func receiptPrefixChecks(ctx context.Context, q queryer) (map[int64][]receiptReadOrdinal, error) {
	n, err := schemaNumber(ctx, q)
	if err != nil || n != reconciliationSchemaVersion {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, "SELECT descriptor FROM receipt_read_ordinals")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[int64][]receiptReadOrdinal{}
	for rows.Next() {
		var raw []byte
		var o receiptReadOrdinal
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = decode(raw, &o); err != nil {
			return nil, err
		}
		m[o.Prefix] = append(m[o.Prefix], o)
	}
	return m, rows.Err()
}
func verifyReadPrefix(x *compositionPrefix, o receiptReadOrdinal) error {
	id := o.Resolver.Run
	if !x.active[id] || x.retired[id] || x.states[id].Status != "open" || x.closed(id) || x.sequences[id] != o.Resolver.Sequence || !x.reservations[o.Query.QueryID] || x.known[o.Query.QueryID] || !x.reservations[o.Query.TargetID] || !x.active[o.OriginalRun] || x.states[o.OriginalRun].Status == "open" || x.sequences[o.OriginalRun] != o.OriginalTerminalSequence {
		return corrupt("receipt read at ineligible historical prefix")
	}
	return nil
}
