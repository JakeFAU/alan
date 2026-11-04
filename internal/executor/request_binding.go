package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"reflect"
)

// selectedOriginalRequest starts with the immutable grant's exact original ID.
// Admission has no chronological ordinal. The writer requires admission before
// selection; the selection's execution fact and its source authorization anchor
// that causal dependency in the ordered prefix. Duplicate transports may follow.
func selectedOriginalRequest(ctx context.Context, q queryer, x *authorityPrefix, service retainedIntent, claim DelegationClaim) (string, int64, error) {
	original, err := loadIntent(ctx, q, claim.OriginalRequestID)
	if err != nil {
		return "", 0, err
	}
	if !x.committed(original.Intent) || original.Kind != "message" || original.Run != claim.Root.Run || original.revision != claim.Root.Permission.Revision || original.Target != service.Run.Subject || original.Generation != service.Run.Generation {
		return "", 0, corrupt("binding original request differs from grant")
	}
	for _, intent := range []retainedIntent{original, service} {
		if err := validateBindingCommit(ctx, q, intent); err != nil {
			return "", 0, err
		}
	}
	observationID := identity("observation", "message", original.ID)
	var run RunID
	var raw []byte
	var consumed sql.NullInt64
	err = q.QueryRowContext(ctx, "SELECT subject,generation,descriptor,consumed_sequence FROM observations WHERE id=?", observationID).Scan(&run.Subject, &run.Generation, &raw, &consumed)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}
	var observation Observation
	if err = decode(raw, &observation); err != nil {
		return "", 0, err
	}
	expected := observationFor(messageDescriptor(original.Intent, "unused transport"), service.Run)
	if run != service.Run || !sameJSON(raw, observation) || !reflect.DeepEqual(expected, observation) {
		return "", 0, corrupt("binding observation differs from original")
	}
	// An absent or future consumption cannot populate the historical witness.
	if !consumed.Valid || consumed.Int64 >= service.Sequence {
		return "", 0, nil
	}
	if consumed.Int64 <= 0 {
		return "", 0, corrupt("invalid binding consumption")
	}
	var selected sql.NullString
	var revision int64
	var successor, intents []byte
	if err = q.QueryRowContext(ctx, "SELECT selected_observation,source_revision,successor,intents FROM history WHERE subject=? AND generation=? AND sequence=?", service.Run.Subject, service.Run.Generation, consumed.Int64).Scan(&selected, &revision, &successor, &intents); err != nil {
		return "", 0, err
	}
	if !selected.Valid || selected.String != observationID || revision != claim.RecipientRevision {
		return "", 0, corrupt("binding history/consumption mismatch")
	}
	var selectionOrder, serviceOrder, authorizationOrder, originalOrder int64
	for _, f := range x.facts {
		if f.Kind == compositionExecutionFact {
			if f.Run == service.Run && f.Sequence == consumed.Int64 {
				selectionOrder = f.Order
			}
			if f.Run == service.Run && f.Sequence == service.Sequence {
				serviceOrder = f.Order
			}
			if f.Run == original.Run && f.Sequence == original.Sequence {
				originalOrder = f.Order
			}
		}
		if f.Kind == compositionAuthorizationFact && f.OperationID == original.ID {
			authorizationOrder = f.Order
		}
	}
	if selectionOrder == 0 || serviceOrder <= selectionOrder || originalOrder == 0 || authorizationOrder <= originalOrder || authorizationOrder >= selectionOrder {
		return "", 0, corrupt("binding causal prefix mismatch")
	}
	auth, err := validatedAuthorization(ctx, q, identity("authorization", original.ID))
	if err != nil {
		return "", 0, err
	}
	if !auth.Allowed {
		return "", 0, corrupt("selected original lacks source authorization")
	}
	rows, err := q.QueryContext(ctx, "SELECT id FROM admissions WHERE observation_id=? ORDER BY id", observationID)
	if err != nil {
		return "", 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return "", 0, err
		}
		ids = append(ids, id)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return "", 0, err
	}
	if len(ids) == 0 {
		return "", 0, corrupt("selected original lacks admission")
	}
	for _, id := range ids {
		d, a, err := loadAdmissionKey(ctx, q, id)
		if err != nil {
			return "", 0, err
		}
		if !validAdmissionHeader(d) || !reflect.DeepEqual(d, messageDescriptor(original.Intent, d.TransportID)) || !a.Accepted || a.Bound == nil || *a.Bound != service.Run || a.ObservationID != observationID || !reflect.DeepEqual(observationFor(d, service.Run), observation) {
			return "", 0, corrupt("binding admission differs from original")
		}
		permitted, err := admissionPermitted(ctx, q, d)
		if err != nil {
			return "", 0, err
		}
		if !permitted {
			return "", 0, corrupt("binding admission lacks route permission")
		}
	}
	// Check the actual receive transition, independent of its retained successor.
	// Byte ceilings are enforced by Commit and whole-ledger validation with caller
	// Limits. This semantic recheck must not introduce a smaller default ceiling.
	limits := Limits{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt}
	runState, err := loadRun(ctx, q, service.Run)
	if err != nil {
		return "", 0, err
	}
	program, err := loadProgram(ctx, q, runState.ProgramID, limits)
	if err != nil {
		return "", 0, err
	}
	before, err := projectionAt(ctx, q, service.Run, consumed.Int64-1)
	if err != nil {
		return "", 0, err
	}
	// Size the canonical typed inputs and retained expected outputs, rather
	// than raw JSON whose omitted zero fields may marshal to a larger value.
	var retainedSuccessor Projection
	if err = decode(successor, &retainedSuccessor); err != nil {
		return "", 0, err
	}
	size := func(value any) (int, error) { raw, err := json.Marshal(value); return len(raw), err }
	if limits.MaxProgramBytes, err = size(program); err != nil {
		return "", 0, err
	}
	if limits.MaxObservationBytes, err = size(observation); err != nil {
		return "", 0, err
	}
	beforeSize, err := size(before)
	if err != nil {
		return "", 0, err
	}
	afterSize, err := size(retainedSuccessor)
	if err != nil {
		return "", 0, err
	}
	limits.MaxDataBytes = max(beforeSize, afterSize)
	// Receive emits no intents. Its exact expected output is the empty array.
	limits.MaxOutputBytes = len("[]")
	reduction, err := Reduce(program, before, &observation, service.Run, consumed.Int64, limits)
	if err != nil {
		return "", 0, corrupt("binding selection reduction: %v", err)
	}
	if !sameJSON(successor, reduction.Projection) || !sameJSON(intents, reduction.Intents) {
		return "", 0, corrupt("binding selection reduction differs from history")
	}
	return observationID, consumed.Int64, nil
}

func validateRequestBindingIndex(ctx context.Context, q queryer, version int) error {
	rows, err := q.QueryContext(ctx, "SELECT name,\"unique\",origin,partial FROM pragma_index_list('admissions') WHERE name='admissions_observation_id'")
	if err != nil {
		return err
	}
	var count int
	var unique, partial int
	var name, origin string
	for rows.Next() {
		count++
		if err = rows.Scan(&name, &unique, &origin, &partial); err != nil {
			rows.Close()
			return err
		}
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	binding := requestAuthorityVersionForSchema(version) == RequestAuthorityBindingVersion
	if !binding {
		if count != 0 {
			return ErrVersion
		}
		return nil
	}
	if count != 1 || unique != 0 || partial != 0 || origin != "c" {
		return ErrVersion
	}
	// The exact observation_id equality uses BINARY collation. index_info
	// omits collation and would accept a NOCASE replacement that forces a scan.
	// Require the single ascending BINARY key emitted by Create; SQLite's
	// auxiliary rowid entry is not an indexed key and is excluded here.
	var columns, matching int
	if err = q.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(
		seqno=0 AND cid>=0 AND name='observation_id' AND coll='BINARY' AND "desc"=0
		),0) FROM pragma_index_xinfo('admissions_observation_id') WHERE "key"=1`).Scan(&columns, &matching); err != nil {
		return err
	}
	if columns != 1 || matching != 1 {
		return ErrVersion
	}
	return nil
}

// validateBindingCommit checks the exact primary-key/index and emitted-intent
// mirror. Whole-ledger validation additionally replays every producing reduction.
func validateBindingCommit(ctx context.Context, q queryer, intent retainedIntent) error {
	var run RunID
	var sequence, revision int64
	var ordinal int
	if err := q.QueryRowContext(ctx, "SELECT subject,generation,sequence,ordinal,source_revision FROM intents WHERE id=?", intent.ID).Scan(&run.Subject, &run.Generation, &sequence, &ordinal, &revision); err != nil {
		return err
	}
	if run != intent.Run || sequence != intent.Sequence || ordinal != intent.Ordinal || revision != intent.revision {
		return corrupt("binding intent index mismatch")
	}
	var raw []byte
	var historyRevision int64
	if err := q.QueryRowContext(ctx, "SELECT source_revision,intents FROM history WHERE subject=? AND generation=? AND sequence=?", run.Subject, run.Generation, sequence).Scan(&historyRevision, &raw); err != nil {
		return err
	}
	var emitted []Intent
	if err := decode(raw, &emitted); err != nil {
		return err
	}
	if historyRevision != revision || len(emitted) != 1 || !reflect.DeepEqual(emitted[0], intent.Intent) {
		return corrupt("binding intent differs from committing reduction")
	}
	return nil
}
