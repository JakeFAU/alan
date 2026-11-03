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

// A reopen can advance an epoch without writing a reduction. The adjacent
// reductions still bound the epoch of a witness at this sequence.
func validateWitnessEpoch(ctx context.Context, q queryer, run RunState, seq, epoch int64) error {
	if seq < 0 || seq > run.Sequence || epoch < 1 || epoch > run.Epoch {
		return corrupt("authority outside run history")
	}
	if run.Epoch == 1 || seq == run.Sequence && epoch == run.Epoch {
		return nil
	}
	var prev, next int64
	err := q.QueryRowContext(ctx, `SELECT
		coalesce((SELECT epoch FROM history WHERE subject=?1 AND generation=?2 AND sequence=?3),1),
		coalesce((SELECT epoch FROM history WHERE subject=?1 AND generation=?2 AND sequence=?3+1),?4)`,
		run.ID.Subject, run.ID.Generation, seq, run.Epoch).Scan(&prev, &next)
	if err != nil {
		return err
	}
	if epoch < prev || epoch > next {
		return corrupt("authority epoch outside its reduction prefix")
	}
	return nil
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
	last := map[RunID]Authority{}
	for index, id := range ids {
		result, e := loadLifecycleResult(ctx, q, id)
		if e != nil {
			return e
		}
		request, e := loadLifecycleRequest(ctx, q, id)
		if e != nil {
			return e
		}
		if result.DecisionSequence != int64(index+1) || !reflect.DeepEqual(request, result.Request) {
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
		if exists && result.Before == nil {
			// Only an admitted invocation can introduce a new subject name.
			// Its creation order is checked after validating the composition protocol.
			var dynamic bool
			enabled, err := compositionProtocol(ctx, q)
			if err != nil {
				return err
			}
			if enabled {
				err = q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM composition_members WHERE subject=?)", subject.ID).Scan(&dynamic)
				if err != nil {
					return err
				}
			}
			exists = !dynamic
		}
		if (result.Before != nil) != exists {
			return corrupt("lifecycle target existence")
		}
		if exists {
			before := result.Before
			if before.Run != (RunID{subject.ID, subject.Generation}) {
				return corrupt("lifecycle target generation")
			}
			run, e := loadRun(ctx, q, before.Run)
			if e != nil {
				return e
			}
			if before.Owner != subject.Owner || before.Revision != subject.Revision || before.Retired != subject.Retired || before.RetryCancelled != subject.RetryCancelled || before.ProgramID != roots[subject.ID].ProgramID || before.Sequence < last[before.Run].Sequence || before.Epoch < last[before.Run].Epoch {
				return corrupt("lifecycle before-state authority")
			}
			if e := validateWitnessEpoch(ctx, q, run, before.Sequence, before.Epoch); e != nil {
				return e
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
			if !reflect.DeepEqual(before.Projection, projected) || projected.Status != "open" && before.Epoch != run.Epoch {
				return corrupt("lifecycle before-state projection")
			}
			last[before.Run] = Authority{Sequence: before.Sequence, Epoch: before.Epoch}
		}
		grant, e := lifecycleGrant(ctx, q, request)
		if e != nil {
			return e
		}
		expected, e := decideTarget(request, result.DecisionSequence, result.Before, roots[request.Intent.Target], grant)
		if e != nil {
			return e
		}
		enabled, e := compositionProtocol(ctx, q)
		if e != nil {
			return e
		}
		if enabled && result.Before != nil && request.Operation == "Retry" {
			root, e := compositionMembershipRoot(ctx, q, result.Before.Run)
			if e != nil {
				return e
			}
			if root != "" {
				after := *result.Before
				expected.Accepted = false
				expected.Successor = 0
				expected.Reason = "invocation_retry_unsupported"
				expected.After = &after
			}
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
		err = q.QueryRowContext(ctx, "SELECT kind,operation_id,predecessor_generation,descriptor FROM run_origins WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&kind, &operation, &predecessor, &raw)
		if errors.Is(err, sql.ErrNoRows) {
			enabled, e := compositionProtocol(ctx, q)
			if e != nil {
				return e
			}
			if enabled {
				var n int
				if e = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_members WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&n); e != nil {
					return e
				}
				if n == 1 {
					kind = "invocation"
					raw = rootBytes[id.Subject]
					err = nil
				}
			}
		}
		if err != nil {
			return err
		}
		if err = q.QueryRowContext(ctx, "SELECT initial FROM runs WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&initial); err != nil {
			return err
		}
		if string(raw) != string(rootBytes[id.Subject]) || run.ProgramID != root.ProgramID || !sameJSON(initial, root.Initial) {
			return corrupt("run changed immutable original artifact/initial state")
		}
		switch kind {
		case "invocation":
			if id.Generation != 1 {
				return ErrCorrupt
			}
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
		if intent.Kind == "adapter" {
			expected, e := loadAdapterResult(ctx, q, r.intent)
			if e != nil {
				return e
			}
			if r.id != identity("adapter-outcome", r.intent) || !sameJSON(r.raw, expected) || r.observation != identity("observation", "adapter", r.intent) {
				return corrupt("adapter settlement binding")
			}
			var n int
			if e = q.QueryRowContext(ctx, "SELECT count(*) FROM admissions WHERE domain='adapter' AND observation_id=?", r.observation).Scan(&n); e != nil {
				return e
			}
			if n != 1 {
				return corrupt("adapter settlement admission")
			}
			continue
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
