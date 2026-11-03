package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
	publicschema "github.com/JakeFAU/alan/schema"
)

// Interpret the controlled transport without interpreting an arbitrary adapter.
func readSnapshot(o receiptReadOrdinal, c effects.Candidate) (receiptquery.Snapshot, error) {
	var s receiptquery.Snapshot
	if c.ID != o.Query.QueryID || c.Key != o.Query.QueryKey || c.RequestID != "" || c.ResponseID != "" || c.ReturnedModel != "" || c.Usage != nil {
		return s, corrupt("read candidate correlation")
	}
	switch c.Disposition {
	case "transport_error", "protocol_invalid":
		if c.Payload != nil {
			return s, corrupt("failed read carries evidence")
		}
		if c.Disposition == "protocol_invalid" && c.HTTPStatus == 200 {
			if _, err := receiptquery.DecodeSnapshot(o.Query, c.Response); err == nil {
				return s, corrupt("valid snapshot classified invalid")
			}
		}
		return s, nil
	case "validated":
		if c.HTTPStatus != 200 {
			return s, corrupt("snapshot HTTP status")
		}
	default:
		return s, corrupt("unsupported read candidate")
	}
	s, err := receiptquery.DecodeSnapshot(o.Query, c.Response)
	if err != nil {
		return s, corrupt("read snapshot")
	}
	mutation, total := int64(0), int64(0)
	if s.Receipt != nil {
		mutation, total = s.Receipt.Mutation, s.Receipt.Total
	}
	expected := map[string]Value{"target": Text(o.Query.TargetID), "status": Text(s.Status), "mutation": Int(mutation), "total": Int(total), "adoption": Text("unassessed"), "adopted": publicschema.Bool(false)}
	if !reflect.DeepEqual(c.Payload, expected) {
		return s, corrupt("snapshot/candidate payload mismatch")
	}
	return s, nil
}

func decideRead(o receiptReadOrdinal, s receiptquery.Snapshot, prior *AdapterResult) string {
	switch {
	case s.Status == "conflict":
		return "conflict"
	case s.Status != "found":
		return "none"
	case !o.Adopt:
		return "denied"
	case o.OriginalMode == "raw":
		return "evidence_only"
	case prior != nil && prior.Reconciled == nil:
		return "existing"
	case o.OriginalMode != "success":
		return "unsupported_policy"
	case prior == nil:
		return "adopted"
	case prior.Reconciled.Receipt == *s.Receipt:
		return "duplicate"
	default:
		return "conflict"
	}
}
func reconciledResult(o receiptReadOrdinal, receipt receiptquery.Receipt) AdapterResult {
	r := AdapterResult{Reconciled: &receiptReconstruction{Target: o.Query.TargetID, QueryID: o.Query.QueryID, Ordinal: o.Query.Ordinal, Receipt: receipt}, Disposition: "validated", Values: map[string]Value{"mutation": Int(receipt.Mutation), "total": Int(receipt.Total), "valid": publicschema.Bool(true), "disposition": Text("validated")}}
	r.Digest = adapterResultDigest(r)
	return r
}
func readResult(a AdapterAttempt, c effects.Candidate, ordinal int, decision string) AdapterResult {
	values, d := effects.Result(a.Prepared.Request.Output.Record, c)
	if d == "validated" {
		values["adoption"] = Text(decision)
		values["adopted"] = publicschema.Bool(decision == "adopted")
	}
	r := AdapterResult{Candidate: c, Values: values, Disposition: d, Read: &receiptReadDecision{Ordinal: ordinal, Adoption: decision}}
	r.Digest = adapterResultDigest(r)
	return r
}
func (s *Store) insertReadResult(ctx context.Context, tx *sql.Tx, id string, r AdapterResult) error {
	if err := checkSize(r, s.limits.MaxObservationBytes); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err == nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO adapter_results VALUES(?,?)", id, raw)
	}
	if err != nil {
		return err
	}
	return s.recordCompositionSource(ctx, tx, compositionExternalResultFact, id)
}

func (s *Store) recordReadResult(ctx context.Context, a AdapterAttempt, ordinal receiptReadOrdinal, c effects.Candidate) (AdapterResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var r AdapterResult
	if s.closed {
		return r, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return r, err
	}
	defer tx.Rollback()
	retained, err := loadAdapterAttempt(ctx, tx, a.Intent.ID)
	if err != nil {
		return r, err
	}
	if !reflect.DeepEqual(retained, a) || !a.Allowed {
		return r, ErrConflict
	}
	o, err := readOrdinal(ctx, tx, a.Intent.ID, ordinal.Query.Ordinal)
	if err != nil {
		return r, err
	}
	if !reflect.DeepEqual(o, ordinal) {
		return r, ErrConflict
	}
	if err = validateOrdinalBinding(ctx, tx, s.namespace, a, o); err != nil {
		return r, err
	}
	snapshot, err := readSnapshot(o, c)
	if err != nil {
		return r, err
	}
	existing, err := loadAdapterResult(ctx, tx, a.Intent.ID)
	if err == nil {
		if existing.Read == nil || existing.Read.Ordinal != o.Query.Ordinal || !reflect.DeepEqual(existing.Candidate, c) {
			return r, ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return r, err
	}
	original, err := loadAdapterResult(ctx, tx, o.Query.TargetID)
	var prior *AdapterResult
	if err == nil {
		prior = &original
	} else if !errors.Is(err, sql.ErrNoRows) {
		return r, err
	}
	decision := decideRead(o, snapshot, prior)
	r = readResult(a, c, o.Query.Ordinal, decision)
	if err = s.insertReadResult(ctx, tx, a.Intent.ID, r); err != nil {
		return r, err
	}
	if decision == "adopted" {
		if err = s.insertReadResult(ctx, tx, o.Query.TargetID, reconciledResult(o, *snapshot.Receipt)); err != nil {
			return r, err
		}
	}
	return r, s.commitOwnedTx(ctx, tx)
}

// Both settlements belong to the resolver's H. Settle its own result last so
// that a crash leaves a durable, discoverable outstanding resolver obligation.
func (s *Store) settleReadOriginalLocked(ctx context.Context, a AdapterAttempt, r AdapterResult) error {
	if r.Read == nil || r.Read.Adoption != "adopted" {
		return nil
	}
	id := a.Prepared.Request.Payload["target"].Text
	old, err := loadAdapterAttempt(ctx, s.conn, id)
	if err != nil {
		return err
	}
	original, err := loadAdapterResult(ctx, s.conn, id)
	if err != nil {
		return err
	}
	if original.Reconciled == nil || original.Reconciled.QueryID != a.Intent.ID {
		return ErrConflict
	}
	return s.settleAdapterLocked(ctx, old, original)
}
