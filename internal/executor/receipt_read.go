package executor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
)

// The ordinal is spent before entry, even if the owning worker never transmits.
type receiptReadOrdinal struct {
	Query                                                                              receiptquery.Query
	OriginalRun                                                                        RunID
	OriginalAuthorization, OriginalAttemptDigest, OriginalPreparedDigest, OriginalMode string
	Resolver                                                                           Authority
	OriginalTerminalSequence, Prefix                                                   int64
	Adopt                                                                              bool
}

// Read is on the query result. Reconciled is exclusively on an original result.
// Neither can confer permission: both are verified against the admitted ordinal.
type receiptReadDecision struct {
	Ordinal  int
	Adoption string
}
type receiptReconstruction struct {
	Target, QueryID string
	Ordinal         int
	Receipt         receiptquery.Receipt
}

func controlledReader(adapter effects.Adapter) (receiptquery.Reader, bool) {
	switch r := adapter.(type) {
	case receiptquery.Reader:
		return r, true
	case *receiptquery.Reader:
		if r != nil {
			return *r, true
		}
	}
	return receiptquery.Reader{}, false
}
func controlledCounter(adapter effects.Adapter) bool {
	switch a := adapter.(type) {
	case receiptquery.Counter:
		return true
	case *receiptquery.Counter:
		return a != nil
	}
	return false
}

func bindReceiptRead(ctx context.Context, q queryer, namespace string, a AdapterAttempt, ordinal int) (receiptReadOrdinal, string, error) {
	var r receiptReadOrdinal
	g, run, err := receiptGrantFor(ctx, q, namespace, a)
	if errors.Is(err, sql.ErrNoRows) {
		return r, "read_denied", nil
	}
	if err != nil {
		return r, "", err
	}
	target := a.Prepared.Request.Payload["target"].Text
	expected, err := receiptquery.LookupEndpoint(g.Endpoint)
	if err != nil || expected != a.Prepared.Endpoint {
		return r, "endpoint_mismatch", nil
	}
	intent, err := loadIntent(ctx, q, target)
	if errors.Is(err, sql.ErrNoRows) {
		return r, "target_missing", nil
	}
	if err != nil {
		return r, "", err
	}
	if intent.Run != run || intent.Sequence != g.Sequence || intent.Ordinal != 0 {
		return r, "target_mismatch", nil
	}
	auth, err := loadAuthorization(ctx, q, target)
	if errors.Is(err, sql.ErrNoRows) {
		return r, "attempt_missing", nil
	}
	if err != nil {
		return r, "", err
	}
	if !auth.Allowed {
		return r, "original_denied", nil
	}
	r.OriginalRun = run
	r.OriginalAuthorization = auth.ID
	r.Adopt = g.Adopt
	var endpoint, wire, key string
	switch intent.Kind {
	case "adapter":
		old, e := loadAdapterAttempt(ctx, q, target)
		if errors.Is(e, sql.ErrNoRows) {
			return r, "attempt_missing", nil
		}
		if e != nil {
			return r, "", e
		}
		if !old.Allowed {
			return r, "original_denied", nil
		}
		if old.AuthorizationID != auth.ID || !reflect.DeepEqual(old.Intent, intent.Intent) {
			return r, "", ErrCorrupt
		}
		if receiptquery.CounterPrepared(old.Prepared) != nil {
			return r, "unsupported_original", nil
		}
		endpoint, wire, key = old.Prepared.Endpoint, old.Prepared.Wire, old.Prepared.Request.Key
		r.OriginalMode = old.Prepared.Model
		r.OriginalPreparedDigest = old.Prepared.Digest
		r.OriginalAttemptDigest = compositionIdentity("alan/original-attempt/v1", old)
	case "external":
		old, e := loadExternalAttempt(ctx, q, target)
		if errors.Is(e, sql.ErrNoRows) {
			return r, "attempt_missing", nil
		}
		if e != nil {
			return r, "", e
		}
		if !old.Allowed {
			return r, "original_denied", nil
		}
		raw, e := json.Marshal(old.Request)
		if e != nil {
			return r, "", e
		}
		if old.AuthorizationID != auth.ID || !reflect.DeepEqual(old.Intent, intent.Intent) {
			return r, "", ErrCorrupt
		}
		endpoint, wire, key = old.Intent.Target, string(raw), old.Request.Key
		r.OriginalMode = "raw"
		r.OriginalAttemptDigest = compositionIdentity("alan/original-attempt/v1", old)
	default:
		return r, "unsupported_original", nil
	}
	if endpoint != g.Endpoint || key != ExternalKey(namespace, target) {
		return r, "endpoint_mismatch", nil
	}
	ledger, err := receiptquery.EndpointID(endpoint, "apply")
	if err != nil {
		return r, "unsupported_original", nil
	}
	digest := sha256.Sum256([]byte(wire))
	r.Query = receiptquery.Query{QueryID: a.Intent.ID, QueryKey: a.Prepared.Request.Key, TargetID: target, TargetKey: key, LedgerID: ledger, Wire: wire, WireSHA256: hex.EncodeToString(digest[:]), Ordinal: ordinal}
	if r.Query.Check() != nil {
		return r, "unsupported_original", nil
	}
	return r, "", nil
}

func readOrdinal(ctx context.Context, q queryer, id string, n int) (receiptReadOrdinal, error) {
	var raw []byte
	var r receiptReadOrdinal
	err := q.QueryRowContext(ctx, "SELECT descriptor FROM receipt_read_ordinals WHERE intent_id=? AND ordinal=?", id, n).Scan(&raw)
	if err != nil {
		return r, err
	}
	if decode(raw, &r) != nil || !sameJSON(raw, r) || r.Query.QueryID != id || r.Query.Ordinal != n {
		return r, corrupt("read ordinal encoding")
	}
	return r, nil
}

// Called by the gate under both owner exclusions. No endpoint operation occurs
// here. The short transaction is committed without reacquiring ownerMu.
func (s *Store) reserveReadLocked(ctx context.Context, id string) (*receiptReadOrdinal, string, error) {
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()
	a, err := loadAdapterAttempt(ctx, tx, id)
	if err != nil {
		return nil, "", err
	}
	if !a.Allowed {
		return nil, "", ErrCorrupt
	}
	if _, err = loadAdapterResult(ctx, tx, id); err == nil {
		return nil, "", ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", err
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=?", id).Scan(&n); err != nil {
		return nil, "", err
	}
	if n >= 2 {
		return nil, "read_exhausted", nil
	}
	r, refusal, err := bindReceiptRead(ctx, tx, s.namespace, a, n+1)
	if err != nil || refusal != "" {
		return nil, refusal, err
	}
	for _, run := range []RunID{r.OriginalRun, a.Intent.Run} {
		root, e := compositionMembershipRoot(ctx, tx, run)
		if e != nil {
			return nil, "", e
		}
		if root == "" {
			return nil, "unscoped", nil
		}
	}
	state, err := loadRun(ctx, tx, r.OriginalRun)
	if err != nil {
		return nil, "", err
	}
	if state.Projection.Status == "open" {
		return nil, "original_open", nil
	}
	if failure := s.dispatchFailures[r.OriginalRun]; failure != nil {
		return nil, "", failure
	}
	for o := range s.liveDispatch {
		if o.source == r.OriginalRun {
			return nil, "original_busy", nil
		}
	}
	var pending int
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM intents i WHERE i.subject=? AND i.generation=?
AND (EXISTS (SELECT 1 FROM adapter_results a WHERE a.intent_id=i.id) OR EXISTS (SELECT 1 FROM external_receipts e WHERE e.intent_id=i.id))
AND NOT EXISTS(SELECT 1 FROM settlements st WHERE st.intent_id=i.id)`, r.OriginalRun.Subject, r.OriginalRun.Generation).Scan(&pending)
	if err != nil {
		return nil, "", err
	}
	if pending != 0 {
		return nil, "original_unsettled", nil
	}
	resolver, err := loadRun(ctx, tx, a.Intent.Run)
	if err != nil {
		return nil, "", err
	}
	r.Resolver = resolver.Authority()
	r.OriginalTerminalSequence = state.Sequence
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(decision_order),0) FROM composition_facts").Scan(&r.Prefix); err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(r)
	if err == nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO receipt_read_ordinals VALUES(?,?,?)", id, n+1, raw)
	}
	if err == nil {
		fact, e := receiptReadFact(ctx, tx, id, n+1)
		if e != nil {
			return nil, "", e
		}
		_, err = appendCompositionFact(ctx, tx, fact)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = tx.Commit()
	}
	return &r, "", err
}

const receiptReadCapability = receiptquery.ReadCapability
const receiptCounterCapability = receiptquery.CounterCapability
