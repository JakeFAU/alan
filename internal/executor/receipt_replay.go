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
	want, reason, err := bindReceiptRead(ctx, q, namespace, a, o.Query.Ordinal, nil)
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
		if !reflect.DeepEqual(r.Candidate, effects.Candidate{ID: a.Intent.ID, Key: a.Prepared.Request.Key, Disposition: r.Disposition}) {
			return corrupt("read refusal with transport metadata")
		}
		if !a.Allowed {
			want := "source_denied"
			if a.Reason == "grant_denied" {
				want = "grant_denied"
			}
			if len(ordinals) != 0 || r.Disposition != want {
				return corrupt("read first-permission denial")
			}
			return nil
		}
		root, err := compositionMembershipRoot(ctx, q, a.Intent.Run)
		if err != nil {
			return err
		}
		if root == "" && (r.Disposition != "unscoped" || len(ordinals) != 0) {
			return corrupt("unscoped read refusal")
		}
		if (len(ordinals) == 2) != (r.Disposition == "read_exhausted") {
			return corrupt("read exhaustion disagrees with allowance")
		}
		if len(ordinals) == 2 {
			current, e := loadRun(ctx, q, a.Intent.Run)
			if e != nil {
				return e
			}
			if current.Epoch <= ordinals[1].Resolver.Epoch {
				return corrupt("exhaustion without a successor activation")
			}
		}
		if len(ordinals) == 1 && r.Disposition != "original_unsettled" {
			return corrupt("refusal contradicts already spent read")
		}
		// Scoped refusals are justified at their ordered result fact below.

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

// All scoped local refusals are committed under the same exclusion that
// observed the cause. Reconstruct the durable prerequisites at that prefix.
func verifyLocalReadAtPrefix(ctx context.Context, q queryer, x *compositionPrefix, f compositionFact) error {
	version, err := schemaNumber(ctx, q)
	if err != nil || version != reconciliationSchemaVersion {
		return err
	}
	a, err := loadAdapterAttempt(ctx, q, f.OperationID)
	if err != nil {
		return err
	}
	r, err := loadAdapterResult(ctx, q, a.Intent.ID)
	if err != nil || !a.Allowed || r.Read != nil {
		return err
	}
	if x.known[a.Intent.ID] || !x.active[a.Intent.Run] || x.retired[a.Intent.Run] || x.states[a.Intent.Run].Status != "open" || x.closed(a.Intent.Run) {
		return corrupt("ineligible local receipt refusal")
	}
	if r.Disposition == "read_exhausted" {
		return nil
	} // allowance equality checked by verifyQueryResult
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return err
	}
	o, reason, err := bindReceiptRead(ctx, q, namespace, a, 1, x)
	if err != nil {
		return err
	}
	if reason == "" && x.states[o.OriginalRun].Status == "open" {
		reason = "original_open"
	}
	if reason != "" {
		if r.Disposition != reason {
			return corrupt("read refusal disagrees with historical binding: %s", reason)
		}
		return nil
	}
	i, ok := x.intents[r.RefusalWitness]
	if !ok || i.Run != o.OriginalRun || !x.reservations[i.ID] || (i.Kind != "external" && i.Kind != "adapter") {
		return corrupt("read refusal lacks original dispatch witness")
	}
	switch r.Disposition {
	case "original_busy":
		var reservation compositionReservation
		var raw []byte
		if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_reservations WHERE operation_id=?", i.ID).Scan(&raw); err != nil {
			return err
		}
		if decode(raw, &reservation) != nil || !reservation.Allowed {
			return corrupt("busy witness has no allowed reservation")
		}
		// An earlier successful read of this terminal generation witnessed
		// its full drain. It cannot later get another original dispatch.
		var spent int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM receipt_read_ordinals WHERE json_extract(descriptor,'$.OriginalRun.Subject')=? AND json_extract(descriptor,'$.OriginalRun.Generation')=? AND json_extract(descriptor,'$.Prefix')<?", o.OriginalRun.Subject, o.OriginalRun.Generation, f.Order-1).Scan(&spent); err != nil {
			return err
		}
		if spent != 0 {
			return corrupt("busy witness follows earlier quiescence")
		}
	case "original_unsettled":
		if !x.known[i.ID] {
			return corrupt("unsettled witness before original result")
		}
		// Selection of the authenticated observation requires its settlement.
		// A later refusal cannot truthfully cite it as pending.
		var selected bool
		if err = q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM settlements st JOIN observations o ON o.id=st.observation_id
          WHERE st.intent_id=? AND o.consumed_sequence<=?)`, i.ID, x.sequences[i.Run]).Scan(&selected); err != nil {
			return err
		}
		if selected {
			return corrupt("unsettled witness was already selected")
		}
		order, err := resultOrder(ctx, q, i.ID)
		if err != nil {
			return err
		}
		// A prior proven drain covered every result then available to this
		// original source. Only a newer result could still await settlement.
		var laterDrain bool
		if err = q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM receipt_read_ordinals WHERE json_extract(descriptor,'$.OriginalRun.Subject')=? AND json_extract(descriptor,'$.OriginalRun.Generation')=? AND json_extract(descriptor,'$.Prefix')>=? AND json_extract(descriptor,'$.Prefix')<?)", o.OriginalRun.Subject, o.OriginalRun.Generation, order, f.Order-1).Scan(&laterDrain); err != nil {
			return err
		}
		if laterDrain {
			return corrupt("unsettled witness already drained")
		}
	default:
		return corrupt("read refusal has no cause at its historical prefix")
	}
	return nil
}
