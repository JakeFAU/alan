package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

type compositionReservation struct {
	OperationID, RootID, Kind string
	Run                       RunID
	Sequence                  int64
	Allowed                   bool
	Reason                    string
	AttemptDigest             string
}

func compositionProtocol(ctx context.Context, q queryer) (bool, error) {
	n, err := schemaNumber(ctx, q)
	if err != nil || !isCompositionSchema(n) {
		return false, err
	}
	var version string
	err = q.QueryRowContext(ctx, "SELECT version FROM composition_profile WHERE singleton=1").Scan(&version)
	return version == compositionProtocolVersion, err
}
func compositionReservationDecision(ctx context.Context, q queryer, i Intent, allowed bool, reason string) (bool, string, error) {
	enabled, err := compositionProtocol(ctx, q)
	if err != nil || !enabled {
		return allowed, reason, err
	}
	id, err := compositionMembershipRoot(ctx, q, i.Run)
	if err != nil || id == "" {
		return allowed, reason, err
	}
	if !allowed {
		return allowed, reason, nil
	}
	root, err := compositionRootFor(ctx, q, i.Run)
	if err != nil {
		return false, "", err
	}
	closed, err := compositionScopeClosed(ctx, q, i.Run)
	if err != nil {
		return false, "", err
	}
	if closed {
		return false, "scope_closed", nil
	}
	var spent int64
	if err = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_charges WHERE root_id=? AND kind='effect'", id).Scan(&spent); err != nil {
		return false, "", err
	}
	if spent >= root.Limits.MaxEffects {
		return false, "effect_limit", nil
	}
	return allowed, reason, nil
}
func recordCompositionReservation(ctx context.Context, tx *sql.Tx, i Intent, allowed bool, reason string, attempt any) error {
	enabled, err := compositionProtocol(ctx, tx)
	if err != nil || !enabled {
		return err
	}
	root, err := compositionMembershipRoot(ctx, tx, i.Run)
	if err != nil || root == "" {
		return err
	}
	r := compositionReservation{OperationID: i.ID, RootID: root, Kind: i.Kind, Run: i.Run, Sequence: i.Sequence, Allowed: allowed, Reason: reason, AttemptDigest: compositionIdentity("alan/attempt-row/v1", attempt)}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO composition_reservations VALUES(?,?,?)", i.ID, root, raw); err != nil {
		return err
	}
	fact, err := compositionProtocolEvidence(ctx, tx, compositionReservationFact, i.ID)
	if err != nil {
		return err
	}
	order, err := appendCompositionFact(ctx, tx, fact)
	if err != nil {
		return err
	}
	if !allowed {
		return nil
	}
	charge := compositionCharge{RootID: root, OperationID: i.ID, Kind: "effect", Effects: 1, DecisionOrder: order}
	raw, err = json.Marshal(charge)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO composition_charges VALUES(?,?,?,?,?)", i.ID, "effect", root, order, raw)
	return err
}
func retainedCompositionReservation(ctx context.Context, q queryer, i Intent, allowed bool, reason string) (bool, string, error) {
	enabled, err := compositionProtocol(ctx, q)
	if err != nil || !enabled {
		return allowed, reason, err
	}
	root, err := compositionMembershipRoot(ctx, q, i.Run)
	if err != nil || root == "" {
		return allowed, reason, err
	}
	var raw []byte
	var r compositionReservation
	if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_reservations WHERE operation_id=?", i.ID).Scan(&raw); err != nil {
		return false, "", err
	}
	if err = decode(raw, &r); err != nil {
		return false, "", err
	}
	if !sameJSON(raw, r) || r.OperationID != i.ID || r.RootID != root || r.Kind != i.Kind || r.Run != i.Run || r.Sequence != i.Sequence {
		return false, "", ErrCorrupt
	}
	if !allowed && (r.Allowed || r.Reason != reason) {
		return false, "", ErrCorrupt
	}
	if allowed && (!r.Allowed && r.Reason != "scope_closed" && r.Reason != "effect_limit") {
		return false, "", ErrCorrupt
	}
	return r.Allowed, r.Reason, nil
}
func compositionExternalKnown(ctx context.Context, q queryer, id string) (bool, error) {
	i, err := loadIntent(ctx, q, id)
	if err != nil {
		return false, err
	}
	if i.Kind == "adapter" {
		_, err = loadAdapterResult(ctx, q, id)
	} else {
		_, err = externalOutcome(ctx, q, id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
