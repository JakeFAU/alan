package executor

import (
	"context"
	"database/sql"
	"errors"

	resource "github.com/JakeFAU/alan/internal/fixture"
)

// ReceiptEvidence summarizes an original with at least one additional retained
// receipt. Known names its one validated receipt. Conflict has two or more.
// The first result remains the source of any settlement and observation.
type ReceiptEvidence struct {
	IntentID      string
	State         string
	ReceiptDigest string `json:",omitempty"`
}

func readReceiptEvidence(ctx context.Context, q queryer, limits Limits) ([]ReceiptEvidence, error) {
	version, err := storedSchemaNumber(ctx, q)
	if err != nil {
		return nil, err
	}
	var object string
	err = q.QueryRowContext(ctx, `SELECT coalesce((SELECT type FROM sqlite_schema
		WHERE name='receipt_conflicts' COLLATE NOCASE AND type IN ('table','view')), '')`).Scan(&object)
	if err != nil {
		return nil, err
	}
	if !hasReceiptConflictSchema(version) {
		if object != "" {
			return nil, ErrVersion
		}
		return nil, nil
	}
	if object != "table" {
		return nil, ErrVersion
	}
	ids, err := stringList(ctx, q, "SELECT DISTINCT intent_id FROM receipt_conflicts ORDER BY intent_id")
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, corrupt("receipt evidence format without a retained offer")
	}
	var evidence []ReceiptEvidence
	for _, id := range ids {
		entry, err := replayReceiptOffers(ctx, q, id, limits)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, corrupt("missing original for receipt offer %s", id)
		}
		if err != nil {
			return nil, err
		}
		evidence = append(evidence, entry)
	}
	return evidence, nil
}

// The ordinary ledger checks the original attempt, grant and first result.
// Recheck each candidate against that same attempt without invoking a recorder
// or adapter. An allowed adapter's unsuccessful first result contributes no receipt.
func replayReceiptOffers(ctx context.Context, q queryer, id string, limits Limits) (ReceiptEvidence, error) {
	entry := ReceiptEvidence{IntentID: id}
	intent, err := loadIntent(ctx, q, id)
	if err != nil {
		return entry, err
	}
	var firstQuery string
	var check func([]byte) (bool, error)
	switch intent.Kind {
	case "external":
		a, err := loadExternalAttempt(ctx, q, id)
		if err != nil {
			return entry, err
		}
		if !a.Allowed {
			return entry, corrupt("receipt offer for denied raw attempt")
		}
		firstQuery = "SELECT descriptor FROM external_receipts WHERE intent_id=?"
		check = func(raw []byte) (bool, error) {
			var r resource.Receipt
			if len(raw) > limits.MaxObservationBytes || decode(raw, &r) != nil || !sameJSON(raw, r) || !validReceipt(a, r) {
				return false, corrupt("invalid raw receipt evidence")
			}
			return true, nil
		}
	case "adapter":
		a, err := loadAdapterAttempt(ctx, q, id)
		if err != nil {
			return entry, err
		}
		if !a.Allowed || intent.Target == receiptReadCapability {
			return entry, corrupt("receipt offer for denied attempt or read capability")
		}
		firstQuery = "SELECT descriptor FROM adapter_results WHERE intent_id=?"
		check = func(raw []byte) (bool, error) {
			var r AdapterResult
			if len(raw) > limits.MaxObservationBytes || decode(raw, &r) != nil || !sameJSON(raw, r) || r.Read != nil || r.Reconciled != nil || r.RefusalWitness != "" {
				return false, corrupt("invalid baseline adapter receipt evidence")
			}
			if err := validateAdapterResult(a, r, limits); err != nil {
				return false, err
			}
			return r.Disposition == "validated", nil
		}
	default:
		return entry, corrupt("receipt offer for nonmutating intent")
	}
	var first []byte
	if err = q.QueryRowContext(ctx, firstQuery, id).Scan(&first); err != nil {
		return entry, err
	}
	known, err := check(first)
	if err != nil {
		return entry, err
	}
	receipts := 0
	if known {
		receipts++
	}
	winnerDigest := programHash(first)
	rows, err := q.QueryContext(ctx, `SELECT winner_digest,offer_digest,descriptor FROM receipt_conflicts
		WHERE intent_id=? ORDER BY offer_digest`, id)
	if err != nil {
		return entry, err
	}
	defer rows.Close()
	var previous string
	for rows.Next() {
		var winner, offer string
		var raw []byte
		if err := rows.Scan(&winner, &offer, &raw); err != nil {
			return entry, err
		}
		if winner != winnerDigest || offer == winner || offer == previous || programHash(raw) != offer {
			return entry, corrupt("receipt offer digest, first result or duplicate")
		}
		known, err := check(raw)
		if err != nil {
			return entry, err
		}
		if !known {
			return entry, corrupt("additional offer has no validated receipt")
		}
		previous = offer
		receipts++
	}
	if err := rows.Err(); err != nil {
		return entry, err
	}
	if previous == "" {
		return entry, corrupt("receipt evidence has no corresponding offer")
	}
	if receipts == 1 {
		entry.State, entry.ReceiptDigest = "known", previous
	} else {
		entry.State = "conflict"
	}
	return entry, nil
}
