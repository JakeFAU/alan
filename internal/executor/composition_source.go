package executor

import (
	"context"
	"database/sql"
	"encoding/json"
)

func compositionSourceEvidence(ctx context.Context, q queryer, kind compositionFactKind, operation string) (compositionFact, error) {
	i, err := loadIntent(ctx, q, operation)
	if err != nil {
		return compositionFact{}, err
	}
	root, err := compositionMembershipRoot(ctx, q, i.Run)
	if err != nil || root == "" {
		return compositionFact{}, err
	}
	var raw []byte
	if kind == compositionAuthorizationFact {
		err = q.QueryRowContext(ctx, "SELECT result FROM attempts WHERE intent_id=?", operation).Scan(&raw)
	} else if i.Kind == "adapter" {
		err = q.QueryRowContext(ctx, "SELECT descriptor FROM adapter_results WHERE intent_id=?", operation).Scan(&raw)
	} else {
		err = q.QueryRowContext(ctx, "SELECT descriptor FROM external_receipts WHERE intent_id=?", operation).Scan(&raw)
	}
	if err != nil {
		return compositionFact{}, err
	}
	return compositionFact{Version: compositionFactVersion, ID: compositionIdentity("alan/fact/"+string(kind)+"/v1", operation), Kind: kind, RootIDs: []string{root}, Run: i.Run, Sequence: i.Sequence, OperationID: operation, RecordDigest: compositionIdentity("alan/"+string(kind)+"-row/v1", json.RawMessage(raw))}, nil
}
func (s *Store) recordCompositionSource(ctx context.Context, tx *sql.Tx, kind compositionFactKind, operation string) error {
	if !s.invocationEnabled() {
		return nil
	}
	fact, err := compositionSourceEvidence(ctx, tx, kind, operation)
	if err != nil || fact.ID == "" {
		return err
	}
	_, err = appendCompositionFact(ctx, tx, fact)
	return err
}
