package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
)

type retainedClosure struct {
	projection Projection
	sequence   int64
	operation  string
	revision   int64
}

func projectionAt(ctx context.Context, q queryer, id RunID, sequence int64) (Projection, error) {
	var raw []byte
	var err error
	if sequence == 0 {
		err = q.QueryRowContext(ctx, "SELECT initial FROM runs WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&raw)
	} else {
		err = q.QueryRowContext(ctx, "SELECT successor FROM history WHERE subject=? AND generation=? AND sequence=?", id.Subject, id.Generation, sequence).Scan(&raw)
	}
	if err != nil {
		return Projection{}, err
	}
	var projection Projection
	err = decode(raw, &projection)
	return projection, err
}
func reconcileProjection(ctx context.Context, q queryer, run RunState, ordinary Projection) error {
	var operation sql.NullString
	var sequence int64
	var raw []byte
	err := q.QueryRowContext(ctx, "SELECT operation_id,sequence,outcome FROM terminal_obligations WHERE subject=? AND generation=?", run.ID.Subject, run.ID.Generation).Scan(&operation, &sequence, &raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if !operation.Valid {
		if !reflect.DeepEqual(run.Projection, ordinary) {
			return corrupt("unexplained current projection for %+v", run.ID)
		}
		return nil
	}
	result, err := loadLifecycleResult(ctx, q, operation.String)
	if err != nil {
		return err
	}
	if !result.Accepted || result.Before == nil || result.After == nil || result.Successor != 0 || result.Before.Run != run.ID || result.After.Run != run.ID || ordinary.Status != "open" || result.Before.Sequence != run.Sequence || sequence != run.Sequence || !reflect.DeepEqual(result.Before.Projection, ordinary) || !reflect.DeepEqual(result.After.Projection, run.Projection) || !sameJSON(raw, run.Projection) {
		return corrupt("inconsistent authority closure for %+v", run.ID)
	}
	if run.Projection.Status != "cancelled" && run.Projection.Status != "abandoned" {
		return corrupt("invalid authority outcome")
	}
	return nil
}
func validateLifecycle(ctx context.Context, q queryer) error {
	subjects, err := stringList(ctx, q, "SELECT id FROM subjects ORDER BY id")
	if err != nil {
		return err
	}
	states := map[string]Subject{}
	roots := map[string]rootOrigin{}
	rootBytes := map[string][]byte{}
	for _, id := range subjects {
		root, raw, e := loadRoot(ctx, q, id)
		if e != nil {
			return e
		}
		if root.Subject.Retired || root.Subject.Generation <= 0 || root.Subject.Revision <= 0 || root.Subject.Owner == "" || root.Initial.Status != "open" {
			return corrupt("invalid static root")
		}
		states[id] = root.Subject
		roots[id] = root
		rootBytes[id] = raw
	}
	ids, err := stringList(ctx, q, "SELECT id FROM lifecycle_results ORDER BY decision_sequence")
	if err != nil {
		return err
	}
	results := map[string]LifecycleResult{}
	closures := map[RunID]retainedClosure{}
	lastSequence := map[RunID]int64{}
	for _, id := range ids {
		result, e := loadLifecycleResult(ctx, q, id)
		if e != nil {
			return e
		}
		request, e := loadLifecycleRequest(ctx, q, id)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(request, result.Request) {
			return corrupt("lifecycle request changed")
		}
		auth, e := loadAuthorization(ctx, q, id)
		if e != nil {
			return e
		}
		if !auth.Allowed {
			return corrupt("target decision without source authorization")
		}
		subject, exists := states[request.Intent.Target]
		if (result.Before != nil) != exists {
			return corrupt("lifecycle target existence")
		}
		if exists {
			before := result.Before
			run, e := loadRun(ctx, q, before.Run)
			if e != nil {
				return e
			}
			if before.Run != (RunID{subject.ID, subject.Generation}) || before.Owner != subject.Owner || before.Revision != subject.Revision || before.Retired != subject.Retired || before.RetryCancelled != subject.RetryCancelled || before.ProgramID != roots[subject.ID].ProgramID || before.Sequence < lastSequence[before.Run] || before.Sequence > run.Sequence || before.Epoch < 1 || before.Epoch > run.Epoch {
				return corrupt("lifecycle before-state authority")
			}
			projected, e := projectionAt(ctx, q, before.Run, before.Sequence)
			if e != nil {
				return e
			}
			if closure, ok := closures[before.Run]; ok {
				if before.Sequence != closure.sequence {
					return corrupt("execution advanced after authority closure")
				}
				projected = closure.projection
			}
			if !reflect.DeepEqual(before.Projection, projected) {
				return corrupt("lifecycle before-state projection")
			}
			lastSequence[before.Run] = before.Sequence
		}
		grant, e := lifecycleGrant(ctx, q, request)
		if e != nil {
			return e
		}
		expected, e := decideTarget(request, result.DecisionSequence, result.Before, roots[request.Intent.Target], grant)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(expected, result) {
			return corrupt("lifecycle decision mismatch")
		}
		if result.Accepted {
			after := result.After
			subject.Generation = after.Run.Generation
			subject.Retired = after.Retired
			states[subject.ID] = subject
			if result.Before.Projection.Status == "open" && after.Run == result.Before.Run && after.Projection.Status != "open" {
				closures[after.Run] = retainedClosure{after.Projection, after.Sequence, id, after.Revision}
			}
		}
		results[id] = result
	}
	for _, id := range subjects {
		subject, e := loadSubject(ctx, q, id)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(subject, states[id]) {
			return corrupt("subject authority lacks lifecycle evidence")
		}
	}
	runs, err := readRunIDs(ctx, q)
	if err != nil {
		return err
	}
	for _, id := range runs {
		run, e := loadRun(ctx, q, id)
		if e != nil {
			return e
		}
		root, ok := roots[id.Subject]
		if !ok {
			return corrupt("run without static subject")
		}
		var kind string
		var operation sql.NullString
		var predecessor sql.NullInt64
		var raw, initial []byte
		if err = q.QueryRowContext(ctx, "SELECT kind,operation_id,predecessor_generation,descriptor FROM run_origins WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&kind, &operation, &predecessor, &raw); err != nil {
			return err
		}
		if err = q.QueryRowContext(ctx, "SELECT initial FROM runs WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&initial); err != nil {
			return err
		}
		if string(raw) != string(rootBytes[id.Subject]) || run.ProgramID != root.ProgramID || !sameJSON(initial, root.Initial) {
			return corrupt("run changed immutable original artifact/initial state")
		}
		switch kind {
		case "root":
			if operation.Valid || predecessor.Valid || id.Generation != root.Subject.Generation {
				return corrupt("invalid static root link")
			}
		case "retry":
			result, ok := results[operation.String]
			if !operation.Valid || !predecessor.Valid || !ok || !result.Accepted || result.Request.Operation != "Retry" || result.Before == nil || result.Successor != id.Generation || result.Request.Intent.Target != id.Subject || predecessor.Int64 != result.Before.Run.Generation || id.Generation != predecessor.Int64+1 {
				return corrupt("invalid retry creation link")
			}
		default:
			return corrupt("unknown run origin")
		}
		if closure, ok := closures[id]; ok {
			var op sql.NullString
			var seq, revision int64
			var outcome []byte
			if err = q.QueryRowContext(ctx, "SELECT operation_id,sequence,source_revision,outcome FROM terminal_obligations WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&op, &seq, &revision, &outcome); err != nil {
				return err
			}
			if !op.Valid || op.String != closure.operation || seq != closure.sequence || run.Sequence != seq || revision != closure.revision || !sameJSON(outcome, closure.projection) {
				return corrupt("missing/mismatched lifecycle terminal obligation")
			}
		}
	}
	// Every accepted Retry must have exactly its linked successor, even at zero steps.
	for id, result := range results {
		if result.Accepted && result.Successor != 0 {
			var operation sql.NullString
			if err = q.QueryRowContext(ctx, "SELECT operation_id FROM run_origins WHERE subject=? AND generation=?", result.Request.Intent.Target, result.Successor).Scan(&operation); err != nil {
				return err
			}
			if !operation.Valid || operation.String != id {
				return corrupt("accepted retry successor link missing")
			}
		}
	}
	if err := validateExternal(ctx, q); err != nil {
		return err
	}
	return validateLifecycleSettlements(ctx, q)
}
func validateLifecycleSettlements(ctx context.Context, q queryer) error {
	rows, err := q.QueryContext(ctx, "SELECT id,intent_id,descriptor,observation_id FROM settlements")
	if err != nil {
		return err
	}
	type row struct {
		id, intent, observation string
		raw                     []byte
	}
	all := []row{}
	for rows.Next() {
		var r row
		if err = rows.Scan(&r.id, &r.intent, &r.raw, &r.observation); err != nil {
			rows.Close()
			return err
		}
		all = append(all, r)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, r := range all {
		intent, e := loadIntent(ctx, q, r.intent)
		if e != nil {
			return e
		}
		if intent.Kind == "external" {
			expected, e := externalOutcome(ctx, q, r.intent)
			if e != nil {
				return e
			}
			if r.id != identity("external-outcome", r.intent) || !sameJSON(r.raw, expected) || r.observation != identity("observation", "receipt", r.intent) {
				return corrupt("external settlement binding")
			}
			var count int
			if e = q.QueryRowContext(ctx, "SELECT count(*) FROM admissions WHERE domain='receipt' AND observation_id=?", r.observation).Scan(&count); e != nil {
				return e
			}
			if count != 1 {
				return corrupt("external settlement admission")
			}
			continue
		}
		expected, e := lifecycleOutcome(ctx, q, r.intent)
		if e != nil {
			return e
		}
		if r.id != identity("lifecycle-outcome", r.intent) || !sameJSON(r.raw, expected) || r.observation != identity("observation", "lifecycle", r.intent) {
			return corrupt("lifecycle settlement binding")
		}
		var count int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM admissions WHERE domain='lifecycle' AND observation_id=?", r.observation).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return corrupt("settlement lacks admission")
		}
	}
	return nil
}
func authorizationProjection(ctx context.Context, q queryer, auth Authorization) (Projection, error) {
	projection, err := projectionAt(ctx, q, auth.Source.Run, auth.Source.Sequence)
	if err != nil {
		return projection, err
	}
	if auth.ClosureOperation == "" {
		return projection, nil
	}
	var operation sql.NullString
	var sequence int64
	var raw []byte
	if err = q.QueryRowContext(ctx, "SELECT operation_id,sequence,outcome FROM terminal_obligations WHERE subject=? AND generation=?", auth.Source.Run.Subject, auth.Source.Run.Generation).Scan(&operation, &sequence, &raw); err != nil {
		return projection, err
	}
	if !operation.Valid || operation.String != auth.ClosureOperation || sequence != auth.Source.Sequence {
		return projection, corrupt("authorization closure evidence")
	}
	var closed Projection
	if err = json.Unmarshal(raw, &closed); err != nil {
		return closed, err
	}
	return closed, nil
}
