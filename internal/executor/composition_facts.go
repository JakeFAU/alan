package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
)

type compositionFactKind string

const (
	compositionExecutionFact   compositionFactKind = "execution"
	compositionLifecycleFact   compositionFactKind = "lifecycle"
	compositionAdmissionFact   compositionFactKind = "admission"
	compositionReservationFact compositionFactKind = "reservation"
	compositionReplyFact       compositionFactKind = "reply"
	compositionFreezeFact      compositionFactKind = "freeze"
	compositionOutcomeFact     compositionFactKind = "outcome"
)

// A fact orders a durable row, not a callback. Digest covers its complete source
// row evidence. RootIDs includes both sides of a cross-root lifecycle decision.
type compositionFact struct {
	Version, ID               string
	Kind                      compositionFactKind
	Order                     int64
	RootIDs                   []string
	Run                       RunID
	Sequence                  int64
	OperationID, RecordDigest string
}
type compositionHistoryEvidence struct {
	Run                       RunID
	Sequence, Epoch, Revision int64
	Selected                  string
	Successor, Intents        json.RawMessage
}

func compositionMembershipRoot(ctx context.Context, q queryer, run RunID) (string, error) {
	var root string
	err := q.QueryRowContext(ctx, "SELECT root_id FROM composition_static_members WHERE subject=? AND generation=?", run.Subject, run.Generation).Scan(&root)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return root, err
}
func compositionExecutionEvidence(ctx context.Context, q queryer, run RunID, seq int64) (compositionFact, error) {
	root, err := compositionMembershipRoot(ctx, q, run)
	if err != nil {
		return compositionFact{}, err
	}
	if root == "" {
		return compositionFact{}, nil
	}
	row := compositionHistoryEvidence{Run: run, Sequence: seq}
	var selected sql.NullString
	err = q.QueryRowContext(ctx, "SELECT epoch,source_revision,selected_observation,successor,intents FROM history WHERE subject=? AND generation=? AND sequence=?", run.Subject, run.Generation, seq).Scan(&row.Epoch, &row.Revision, &selected, &row.Successor, &row.Intents)
	if err != nil {
		return compositionFact{}, err
	}
	row.Selected = selected.String
	return compositionFact{Version: compositionFactVersion, ID: compositionIdentity("alan/fact/execution/v1", []any{run, seq}), Kind: compositionExecutionFact, RootIDs: []string{root}, Run: run, Sequence: seq, RecordDigest: compositionIdentity("alan/history-row/v1", row)}, nil
}
func compositionLifecycleEvidence(ctx context.Context, q queryer, result LifecycleResult) (compositionFact, error) {
	roots := []string{}
	target := RunID{Subject: result.Request.Intent.Target, Generation: result.Request.Intent.Generation}
	if result.Before != nil {
		target = result.Before.Run
	}
	for _, run := range []RunID{result.Request.Intent.Run, target} {
		root, err := compositionMembershipRoot(ctx, q, run)
		if err != nil {
			return compositionFact{}, err
		}
		if root != "" && !slices.Contains(roots, root) {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return compositionFact{}, nil
	}
	slices.Sort(roots)
	var descriptor, raw json.RawMessage
	if err := q.QueryRowContext(ctx, "SELECT descriptor,result FROM lifecycle_results WHERE id=?", result.Request.Intent.ID).Scan(&descriptor, &raw); err != nil {
		return compositionFact{}, err
	}
	sequence := int64(0)
	if result.Before != nil {
		sequence = result.Before.Sequence
	}
	return compositionFact{Version: compositionFactVersion, ID: compositionIdentity("alan/fact/lifecycle/v1", result.Request.Intent.ID), Kind: compositionLifecycleFact, RootIDs: roots, Run: target, Sequence: sequence, OperationID: result.Request.Intent.ID, RecordDigest: compositionIdentity("alan/lifecycle-row/v1", []json.RawMessage{descriptor, raw})}, nil
}

// appendCompositionFact is transaction-local. It neither starts nor commits a
// transaction, so callers cannot separate the ordered fact from its source row.
// The finite protocol kinds are reserved but rejected until their row validators
// are supplied by Task2b/2c; opaque or synthetic protocol facts cannot be appended.
func appendCompositionFact(ctx context.Context, tx *sql.Tx, f compositionFact) (int64, error) {
	if f.ID == "" {
		return 0, fmt.Errorf("empty composition fact")
	}
	if f.Version != compositionFactVersion || f.Order != 0 || len(f.RootIDs) == 0 || !digestIdentity(f.ID) || !digestIdentity(f.RecordDigest) {
		return 0, fmt.Errorf("invalid composition fact")
	}
	roots, err := canonicalStrings(f.RootIDs)
	if err != nil || !reflect.DeepEqual(roots, f.RootIDs) {
		return 0, fmt.Errorf("invalid fact roots")
	}
	var expected compositionFact
	switch f.Kind {
	case compositionExecutionFact:
		expected, err = compositionExecutionEvidence(ctx, tx, f.Run, f.Sequence)
	case compositionLifecycleFact:
		var r LifecycleResult
		r, err = loadLifecycleResult(ctx, tx, f.OperationID)
		if err == nil {
			expected, err = compositionLifecycleEvidence(ctx, tx, r)
		}
	default:
		return 0, ErrVersion
	}
	if err != nil {
		return 0, err
	}
	if !reflect.DeepEqual(expected, f) {
		return 0, corrupt("fact lacks its exact committed row")
	}
	var old []byte
	var order int64
	err = tx.QueryRowContext(ctx, "SELECT decision_order,descriptor FROM composition_facts WHERE id=?", f.ID).Scan(&order, &old)
	if err == nil {
		f.Order = order
		if !sameJSON(old, f) {
			return 0, ErrConflict
		}
		return order, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(decision_order),0) FROM composition_facts").Scan(&order); err != nil {
		return 0, err
	}
	if order == math.MaxInt64 {
		return 0, ErrLimit
	}
	f.Order = order + 1
	raw, err := json.Marshal(f)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO composition_facts VALUES(?,?,?,?)", f.Order, f.ID, f.Kind, raw)
	return f.Order, err
}
func (s *Store) recordCompositionExecution(ctx context.Context, tx *sql.Tx, run RunID, seq int64) error {
	if s.composition == nil {
		return nil
	}
	f, err := compositionExecutionEvidence(ctx, tx, run, seq)
	if err != nil {
		return err
	}
	if f.ID == "" {
		return nil
	}
	_, err = appendCompositionFact(ctx, tx, f)
	return err
}
func (s *Store) recordCompositionLifecycle(ctx context.Context, tx *sql.Tx, result LifecycleResult) error {
	if s.composition == nil {
		return nil
	}
	// Scoped Retry's correlated denial policy belongs to Task2b. Until then,
	// never commit a generation that escapes the foundation's fixed membership.
	if result.Successor != 0 {
		root, err := compositionMembershipRoot(ctx, tx, result.Before.Run)
		if err != nil {
			return err
		}
		if root != "" {
			return ErrVersion
		}
	}
	f, err := compositionLifecycleEvidence(ctx, tx, result)
	if err != nil {
		return err
	}
	if f.ID == "" {
		return nil
	}
	_, err = appendCompositionFact(ctx, tx, f)
	return err
}
func readCompositionFacts(ctx context.Context, q queryer) ([]compositionFact, error) {
	rows, err := q.QueryContext(ctx, "SELECT decision_order,id,kind,descriptor FROM composition_facts ORDER BY decision_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	facts := []compositionFact{}
	for rows.Next() {
		var order int64
		var id string
		var kind compositionFactKind
		var raw []byte
		var f compositionFact
		if err = rows.Scan(&order, &id, &kind, &raw); err != nil {
			return nil, err
		}
		if err = decode(raw, &f); err != nil {
			return nil, err
		}
		if f.Order != order || f.ID != id || f.Kind != kind || f.Version != compositionFactVersion || !sameJSON(raw, f) {
			return nil, corrupt("composition fact row mismatch")
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}
func validateCompositionFacts(ctx context.Context, q queryer) error {
	expected := map[string]compositionFact{}
	runs, err := readRunIDs(ctx, q)
	if err != nil {
		return err
	}
	for _, run := range runs {
		root, err := compositionMembershipRoot(ctx, q, run)
		if err != nil {
			return err
		}
		if root == "" {
			continue
		}
		history, err := readHistory(ctx, q, run)
		if err != nil {
			return err
		}
		for _, h := range history {
			f, err := compositionExecutionEvidence(ctx, q, run, h.seq)
			if err != nil {
				return err
			}
			expected[f.ID] = f
		}
	}
	ids, err := stringList(ctx, q, "SELECT id FROM lifecycle_results ORDER BY decision_sequence")
	if err != nil {
		return err
	}
	for _, id := range ids {
		r, err := loadLifecycleResult(ctx, q, id)
		if err != nil {
			return err
		}
		f, err := compositionLifecycleEvidence(ctx, q, r)
		if err != nil {
			return err
		}
		if f.ID != "" {
			expected[f.ID] = f
		}
	}
	facts, err := readCompositionFacts(ctx, q)
	if err != nil {
		return err
	}
	if len(facts) != len(expected) {
		return corrupt("missing or invented composition facts")
	}
	sequences := map[RunID]int64{}
	closed := map[RunID]bool{}
	lastLifecycle := int64(0)
	seen := map[string]bool{}
	for i, f := range facts {
		want, ok := expected[f.ID]
		if !ok || f.Order != int64(i+1) {
			return corrupt("invalid fact identity/order")
		}
		want.Order = f.Order
		if !reflect.DeepEqual(f, want) {
			return corrupt("fact source row changed")
		}
		delete(expected, f.ID)
		switch f.Kind {
		case compositionExecutionFact:
			if closed[f.Run] || f.Sequence != sequences[f.Run]+1 {
				return corrupt("execution fact violates prefix order")
			}
			predecessor, err := compositionObservationPredecessor(ctx, q, f.Run, f.Sequence)
			if err != nil {
				return err
			}
			if predecessor.ID != "" && !seen[predecessor.ID] {
				return corrupt("selected observation consumed before scoped source fact %s", predecessor.ID)
			}
			sequences[f.Run] = f.Sequence
		case compositionLifecycleFact:
			r, err := loadLifecycleResult(ctx, q, f.OperationID)
			if err != nil {
				return err
			}
			if r.DecisionSequence <= lastLifecycle {
				return corrupt("lifecycle fact order")
			}
			lastLifecycle = r.DecisionSequence
			for _, run := range []RunID{r.Request.Intent.Run, f.Run} {
				root, err := compositionMembershipRoot(ctx, q, run)
				if err != nil {
					return err
				}
				if root == "" {
					continue
				}
				if run == r.Request.Intent.Run && sequences[run] < r.Request.Intent.Sequence {
					return corrupt("lifecycle fact precedes its source intent")
				}
				if run == f.Run && r.Before != nil && sequences[run] != r.Before.Sequence {
					return corrupt("lifecycle target prefix mismatch")
				}
			}
			if r.Accepted && r.Before != nil && r.After != nil && r.Before.Projection.Status == "open" && r.After.Projection.Status != "open" {
				closed[r.Before.Run] = true
			}
		default:
			return ErrVersion
		}
		seen[f.ID] = true
	}
	return nil
}

// Resolve consumption through retained admission evidence, not the observation's
// source name alone. An execution fact already containing a terminal sequence is
// insufficient when a later lifecycle decision produced the terminal obligation.
// Only predecessor classes represented by this foundation are enforced here.
func compositionObservationPredecessor(ctx context.Context, q queryer, run RunID, sequence int64) (compositionFact, error) {
	var empty compositionFact
	var selected sql.NullString
	if err := q.QueryRowContext(ctx, "SELECT selected_observation FROM history WHERE subject=? AND generation=? AND sequence=?", run.Subject, run.Generation, sequence).Scan(&selected); err != nil {
		return empty, err
	}
	if !selected.Valid {
		return empty, nil
	}
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM observations WHERE id=?", selected.String).Scan(&raw); err != nil {
		return empty, err
	}
	var observation Observation
	if err := decode(raw, &observation); err != nil {
		return empty, err
	}
	switch observation.Provenance {
	case "ordinary-message", "runtime-terminal", "runtime-lifecycle":
	default:
		// Local input has no scoped source fact. Adapter/receipt and future invocation
		// predecessors require the later protocol facts; they are not inferred here.
		return empty, nil
	}
	var admissionID string
	if err := q.QueryRowContext(ctx, "SELECT id FROM admissions WHERE observation_id=? ORDER BY id LIMIT 1", selected.String).Scan(&admissionID); err != nil {
		return empty, err
	}
	d, result, err := loadAdmissionKey(ctx, q, admissionID)
	if err != nil {
		return empty, err
	}
	if !result.Accepted || result.Bound == nil || *result.Bound != run || result.ObservationID != selected.String || !reflect.DeepEqual(observation, observationFor(d, *result.Bound)) {
		return empty, corrupt("selected observation lacks its admission binding")
	}
	switch d.Kind {
	case "message":
		intent, err := loadIntent(ctx, q, d.IntentID)
		if err != nil {
			return empty, err
		}
		if intent.Kind != "message" || !reflect.DeepEqual(d, messageDescriptor(intent.Intent, d.TransportID)) {
			return empty, corrupt("selected message admission source")
		}
		return compositionExecutionEvidence(ctx, q, intent.Run, intent.Sequence)
	case "terminal":
		expected, err := terminalDescriptor(ctx, q, d.Source, TerminalDestination{d.Target, d.Obligation}, d.TransportID)
		if err != nil {
			return empty, err
		}
		if !reflect.DeepEqual(d, expected) {
			return empty, corrupt("selected terminal admission source")
		}
		var operation sql.NullString
		var terminalSequence int64
		if err := q.QueryRowContext(ctx, "SELECT sequence,operation_id FROM terminal_obligations WHERE subject=? AND generation=?", d.Source.Subject, d.Source.Generation).Scan(&terminalSequence, &operation); err != nil {
			return empty, err
		}
		if operation.Valid {
			decision, err := loadLifecycleResult(ctx, q, operation.String)
			if err != nil {
				return empty, err
			}
			return compositionLifecycleEvidence(ctx, q, decision)
		}
		return compositionExecutionEvidence(ctx, q, d.Source, terminalSequence)
	case "lifecycle":
		outcome, err := lifecycleOutcome(ctx, q, d.IntentID)
		if err != nil {
			return empty, err
		}
		if !reflect.DeepEqual(d, lifecycleDescriptor(outcome, d.TransportID)) {
			return empty, corrupt("selected lifecycle admission source")
		}
		if outcome.Disposition != "target_decision" {
			// A source-denied settlement has no target decision/fact in this profile.
			return empty, nil
		}
		decision, err := loadLifecycleResult(ctx, q, d.IntentID)
		if err != nil {
			return empty, err
		}
		return compositionLifecycleEvidence(ctx, q, decision)
	default:
		return empty, corrupt("selected observation admission kind")
	}
}
