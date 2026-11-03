package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JakeFAU/alan/adapters/receiptquery"
)

const reconciliationSchemaVersion = 5

func isCompositionSchema(n int) bool {
	return n == compositionSchemaVersion || n == reconciliationSchemaVersion
}

// ReceiptGrant authorizes one resolver to query one original operation on one
// ledger. Invocation and Member identify a dynamic origin; Original a static one.
type ReceiptGrant struct {
	Resolver, Original RunID
	Invocation, Member string
	Sequence           int64
	Endpoint           string
	Adopt              bool
}

type ReconciliationProfile struct{ Grants []ReceiptGrant }

const reconciliationSchema = `
CREATE TABLE reconciliation_profile(singleton INTEGER PRIMARY KEY CHECK(singleton=1), descriptor BLOB NOT NULL);
CREATE TABLE receipt_read_ordinals(intent_id TEXT NOT NULL REFERENCES adapter_attempts(intent_id),ordinal INTEGER NOT NULL CHECK(ordinal BETWEEN 1 AND 2), descriptor BLOB NOT NULL, PRIMARY KEY(intent_id,ordinal));`

func (g ReceiptGrant) target(namespace string) (RunID, string) {
	run := g.Original
	if g.Invocation != "" {
		run = compositionMemberID(compositionInvocationID(namespace, g.Invocation), g.Member)
	}
	id, _ := json.Marshal([]any{run.Subject, run.Generation, g.Sequence, 0})
	return run, string(id)
}

func (p ReconciliationProfile) check() error {
	if len(p.Grants) > 64 {
		return ErrLimit
	}
	type locator struct {
		Resolver RunID
		Target   string
	}
	seen := map[locator]bool{}
	for _, g := range p.Grants {
		static := g.Original.Subject != "" && g.Original.Generation > 0 && g.Invocation == "" && g.Member == ""
		dynamic := g.Original == (RunID{}) && g.Invocation != "" && g.Member != ""
		if (!static && !dynamic) || g.Sequence <= 0 || g.Resolver.Subject == "" || g.Resolver.Generation <= 0 {
			return fmt.Errorf("invalid receipt grant")
		}
		if _, err := receiptquery.EndpointID(g.Endpoint, "apply"); err != nil {
			return err
		}
		_, id := g.target("profile")
		k := locator{g.Resolver, id}
		if seen[k] {
			return fmt.Errorf("duplicate receipt grant")
		}
		seen[k] = true
	}
	return nil
}
func createReconciliationProfile(ctx context.Context, tx *sql.Tx, p ReconciliationProfile) error {
	if err := p.check(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, reconciliationSchema); err != nil {
		return err
	}
	raw, err := json.Marshal(p)
	if err == nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO reconciliation_profile VALUES(1,?)", raw)
	}
	return err
}
func readReconciliationProfile(ctx context.Context, q queryer) (ReconciliationProfile, error) {
	var p ReconciliationProfile
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM reconciliation_profile WHERE singleton=1").Scan(&raw); err != nil {
		return p, err
	}
	if err := decode(raw, &p); err != nil || !sameJSON(raw, p) || p.check() != nil {
		return p, corrupt("receipt grant profile")
	}
	return p, nil
}

func receiptGrantFor(ctx context.Context, q queryer, namespace string, a AdapterAttempt) (ReceiptGrant, RunID, error) {
	p, err := readReconciliationProfile(ctx, q)
	if err != nil {
		return ReceiptGrant{}, RunID{}, err
	}
	target := a.Prepared.Request.Payload["target"].Text
	for _, g := range p.Grants {
		run, id := g.target(namespace)
		if g.Resolver != a.Intent.Run || id != target {
			continue
		}
		if g.Invocation != "" {
			b, m, e := loadCompositionBinding(ctx, q, run)
			if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return g, run, e
			}
			if e != nil || b.OperationID != g.Invocation || b.Namespace != namespace || m != g.Member {
				break
			}
		}
		return g, run, nil
	}
	return ReceiptGrant{}, RunID{}, sql.ErrNoRows
}
