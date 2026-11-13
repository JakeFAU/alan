package executor

import (
	"context"
	"database/sql"
)

// reportReader belongs to one report under its caller's owner lock or read
// transaction. It batches SQL reads but leaves validation at the point of use.
type reportReader struct {
	q queryer

	rootsLoaded bool
	roots       map[string]reportRootRow
	runsLoaded  bool
	runs        map[RunID]reportRunRow
}

type reportRootRow struct {
	descriptor  []byte
	publication []byte
	published   bool
	hasFreeze   bool
	hasClosure  bool
}

type reportRunRow struct {
	state      RunState
	projection []byte
	hasSubject bool
}

// The presence gates reference the same columns as the point queries, so a
// column-level read denial triggers fallback even if no matching row exists.
// Only the publication descriptor is needed by report preparation.
const reportRootsSQL = `
	SELECT r.id, r.descriptor, p.scope_id, p.descriptor,
		(f.scope_id IS NOT NULL) + (f.descriptor IS NOT NULL),
		(c.scope_id IS NOT NULL) + (c.root_id IS NOT NULL)
			+ (c.decision_order IS NOT NULL) + (c.descriptor IS NOT NULL)
	FROM composition_roots r
	LEFT JOIN composition_root_publications p ON p.scope_id = 'root:' || r.id
	LEFT JOIN composition_freezes f ON f.scope_id = 'root:' || r.id
	LEFT JOIN composition_closures c ON c.scope_id = 'root:' || r.id`

const reportRunsSQL = `
	SELECT r.subject, r.generation, r.program_id, r.epoch, r.sequence, s.id IS NOT NULL, s.revision, r.projection
	FROM runs r LEFT JOIN subjects s ON s.id = r.subject`

// reportBytes accepts the storage classes that Scan into []byte reads
// directly. Other classes use the original Scan conversion via fallback.
func reportBytes(value any) ([]byte, bool) {
	switch value := value.(type) {
	case nil:
		return nil, true
	case []byte:
		return value, true
	case string:
		return []byte(value), true
	default:
		return nil, false
	}
}

// A nil map disables the family. Any failed speculative read or unsupported
// row defers to point queries, which own both Scan errors and their ordering.
func readReportRootRows(ctx context.Context, q queryer) map[string]reportRootRow {
	rows, err := q.QueryContext(ctx, reportRootsSQL)
	if err != nil {
		return nil
	}
	defer rows.Close()
	roots := make(map[string]reportRootRow)
	for rows.Next() {
		var key, descriptor, publicationScope, publication any
		var freeze, closure int64
		if err := rows.Scan(&key, &descriptor, &publicationScope, &publication, &freeze, &closure); err != nil {
			return nil
		}
		id, keyOK := key.(string)
		rootRaw, rootOK := reportBytes(descriptor)
		publicationRaw, publicationOK := reportBytes(publication)
		scope, scopeOK := publicationScope.(string)
		if !keyOK || !rootOK || !publicationOK || (publicationScope != nil && (!scopeOK || scope != "root:"+id)) {
			return nil
		}
		if _, duplicate := roots[id]; duplicate {
			return nil
		}
		roots[id] = reportRootRow{
			descriptor:  rootRaw,
			publication: publicationRaw,
			published:   publicationScope != nil,
			hasFreeze:   freeze > 0,
			hasClosure:  closure > 0,
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return roots
}

func readReportRunRows(ctx context.Context, q queryer) map[RunID]reportRunRow {
	rows, err := q.QueryContext(ctx, reportRunsSQL)
	if err != nil {
		return nil
	}
	defer rows.Close()
	runs := make(map[RunID]reportRunRow)
	for rows.Next() {
		var subject, generation, program, epoch, sequence, revision, projection any
		var hasSubject bool
		if err := rows.Scan(&subject, &generation, &program, &epoch, &sequence, &hasSubject, &revision, &projection); err != nil {
			return nil
		}
		name, subjectOK := subject.(string)
		gen, generationOK := generation.(int64)
		programID, programOK := program.(string)
		epochNumber, epochOK := epoch.(int64)
		sequenceNumber, sequenceOK := sequence.(int64)
		raw, projectionOK := reportBytes(projection)
		if !subjectOK || !generationOK || !programOK || !epochOK || !sequenceOK || !projectionOK {
			return nil
		}
		rev, revisionOK := revision.(int64)
		if hasSubject && !revisionOK {
			return nil
		}
		id := RunID{Subject: name, Generation: gen}
		if _, duplicate := runs[id]; duplicate {
			return nil
		}
		runs[id] = reportRunRow{
			state:      RunState{ID: id, ProgramID: programID, Epoch: epochNumber, Sequence: sequenceNumber, Revision: rev},
			projection: raw,
			hasSubject: hasSubject,
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return runs
}

func (r *reportReader) root(ctx context.Context, id string) (ResourceRoot, error) {
	var root ResourceRoot
	if !r.rootsLoaded {
		r.rootsLoaded = true
		r.roots = readReportRootRows(ctx, r.q)
	}
	var raw []byte
	if r.roots == nil {
		if err := r.q.QueryRowContext(ctx, "SELECT descriptor FROM composition_roots WHERE id=?", id).Scan(&raw); err != nil {
			return root, err
		}
	} else {
		if err := ctx.Err(); err != nil {
			return root, err
		}
		row, ok := r.roots[id]
		if !ok {
			return root, sql.ErrNoRows
		}
		raw = row.descriptor
	}
	err := decode(raw, &root)
	return root, err
}

func (r *reportReader) prepareRoot(ctx context.Context, id string) (InvocationPreparation, error) {
	row, ok := r.roots[id]
	if !ok || (!row.published && (row.hasFreeze || row.hasClosure)) {
		return prepareRoot(ctx, r.q, id)
	}
	root, err := r.root(ctx, id)
	if err != nil {
		return InvocationPreparation{}, err
	}
	// Same initial metadata and publication decoding as prepareRoot. With
	// neither a freeze nor a closure, prepareCompositionScope stops at members.
	scope := "root:" + id
	p := InvocationPreparation{Outcome: compositionOutcome{InvocationID: scope, OperationID: scope, Caller: root.Run}}
	if !row.published {
		p.Pending = "members"
		return p, nil
	}
	if err := decode(row.publication, &p.Outcome); err != nil {
		return p, err
	}
	p.Ready = true
	return p, nil
}

func (r *reportReader) run(ctx context.Context, id RunID) (RunState, error) {
	if !r.runsLoaded {
		r.runsLoaded = true
		r.runs = readReportRunRows(ctx, r.q)
	}
	if r.runs == nil {
		return loadRun(ctx, r.q, id)
	}
	if err := ctx.Err(); err != nil {
		return RunState{}, err
	}
	row, ok := r.runs[id]
	if !ok || !row.hasSubject {
		return RunState{}, sql.ErrNoRows
	}
	state := row.state
	if err := decode(row.projection, &state.Projection); err != nil {
		return RunState{}, err
	}
	return state, nil
}
