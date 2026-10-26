package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

type compositionScope struct {
	ID, RootID string
	Members    []RunID
}

// Retirement is separate from a terminal freeze: a prior successful freeze is
// immutable evidence, but does not authorize publication after a later closure.
// DecisionOrder references the lifecycle fact in the same owner transaction.
type compositionClosure struct {
	RootID, LifecycleOperationID, Reason string
	Freeze                               compositionFreeze
}

func compositionScopeFor(ctx context.Context, q queryer, run RunID) (compositionScope, error) {
	root, err := compositionRootFor(ctx, q, run)
	if err != nil {
		return compositionScope{}, err
	}
	scope := compositionScope{ID: "root:" + root.ID, RootID: root.ID, Members: root.Members}
	binding, _, err := loadCompositionBinding(ctx, q, run)
	if errors.Is(err, sql.ErrNoRows) {
		return scope, nil
	}
	if err != nil {
		return scope, err
	}
	scope.ID = binding.InvocationID
	scope.Members = make([]RunID, 0, len(binding.Members))
	for _, name := range sortedKeys(binding.Members) {
		scope.Members = append(scope.Members, binding.Members[name])
	}
	return scope, nil
}

func loadCompositionClosure(ctx context.Context, q queryer, scope string) (compositionClosure, error) {
	var closure compositionClosure
	var root string
	var order int64
	var raw []byte
	err := q.QueryRowContext(ctx,
		"SELECT root_id,decision_order,descriptor FROM composition_closures WHERE scope_id=?",
		scope).Scan(&root, &order, &raw)
	if err != nil {
		return closure, err
	}
	if err = decode(raw, &closure); err != nil {
		return closure, err
	}
	identityMatches := closure.RootID == root && closure.Freeze.InvocationID == scope
	decisionMatches := closure.Freeze.DecisionOrder == order && closure.Freeze.Status == "cancelled"
	if !sameJSON(raw, closure) || !identityMatches || !decisionMatches || closure.Reason != "subject_retired" {
		return closure, ErrCorrupt
	}
	return closure, nil
}

func recordCompositionLifecycleClosure(ctx context.Context, tx *sql.Tx, result LifecycleResult, fact compositionFact, order int64, limits Limits) error {
	if !result.Accepted || result.Before == nil || result.After == nil {
		return nil
	}
	newlyRetired := result.Request.Operation == "RetireSubject" && !result.Before.Retired && result.After.Retired
	if newlyRetired {
		return recordCompositionRetirement(ctx, tx, result, fact, order, limits)
	}
	// Accepted already_terminal CancelRun/AbandonRun does not confer scope
	// authority. Only an actual open-to-terminal transition adds a freeze.
	if result.Before.Projection.Status == "open" && result.After.Projection.Status != "open" {
		return freezeCompositionScope(ctx, tx, fact.Run, fact.ID, limits)
	}
	return nil
}

func recordCompositionRetirement(ctx context.Context, tx *sql.Tx, result LifecycleResult, fact compositionFact, order int64, limits Limits) error {
	scope, err := compositionScopeFor(ctx, tx, fact.Run)
	if err != nil {
		return err
	}
	if _, err = loadCompositionClosure(ctx, tx, scope.ID); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	// A previous non-success cause wins. The earlier successful freeze is kept
	// intact and coexists with the later retirement closure when unpublished.
	var raw []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM composition_freezes WHERE scope_id=?", scope.ID).Scan(&raw)
	if err == nil {
		var earlier compositionFreeze
		if err = decode(raw, &earlier); err != nil {
			return err
		}
		if !sameJSON(raw, earlier) || earlier.DecisionOrder >= order {
			return ErrCorrupt
		}
		if earlier.Status != "succeeded" {
			return nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var published int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM composition_results WHERE invocation_id=?", scope.ID).Scan(&published); err != nil {
		return err
	}
	if published != 0 {
		return nil
	} // Publication already closed the scope permanently.
	closure := compositionClosure{
		RootID:               scope.RootID,
		LifecycleOperationID: result.Request.Intent.ID,
		Reason:               "subject_retired",
		Freeze: compositionFreeze{
			InvocationID:  scope.ID,
			Trigger:       fact.ID,
			Run:           fact.Run,
			Sequence:      result.Before.Sequence,
			DecisionOrder: order,
			Status:        "cancelled",
			ReplyIntents:  []string{}, InvocationIntents: []string{}, ExternalIntents: []string{},
		},
	}
	if err = captureCompositionIntents(ctx, tx, &closure.Freeze, scope.Members); err != nil {
		return err
	}
	if err = checkSize(closure, limits.MaxDataBytes); err != nil {
		return err
	}
	raw, err = json.Marshal(closure)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO composition_closures VALUES(?,?,?,?)", scope.ID, scope.RootID, order, raw)
	return err
}
