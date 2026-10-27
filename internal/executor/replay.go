package executor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// ReplayReport is evidence of consistency, not an independent policy proof.
// Replay has no dispatcher, endpoint or lifecycle mutation capability.
type ReplayReport struct {
	Namespace                 string
	SQLiteVersion             string
	SQLiteSourceID            string
	HistoryCount, IntentCount int
	Runs                      []RunState
	Roots                     []RootCompletion
}

func Replay(ctx context.Context, path string, limits Limits) (report ReplayReport, err error) {
	real, err := canonicalPath(path, false)
	if err != nil {
		return report, err
	}
	s, err := connect(ctx, real, true)
	if err != nil {
		return report, err
	}
	defer func() { err = joinClose(err, s.Close()) }()
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	report, err = validateLedger(ctx, tx, limits)
	if err != nil {
		return report, err
	}
	return report, tx.Commit()
}

func joinClose(err, closeErr error) error {
	if err != nil {
		if closeErr != nil {
			return fmt.Errorf("%w; closing: %v", err, closeErr)
		}
		return err
	}
	return closeErr
}
func corrupt(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrCorrupt, fmt.Sprintf(format, args...))
}
func sameJSON(raw []byte, v any) bool {
	canonical, err := json.Marshal(v)
	return err == nil && bytes.Equal(raw, canonical)
}

type retainedObservation struct {
	Observation
	consumed sql.NullInt64
}
type retainedIntent struct {
	Intent
	revision int64
	raw      []byte
}
type historyRow struct {
	seq, epoch, revision int64
	selected             sql.NullString
	successor, intents   []byte
}

// validateLedger reconstructs ordinary computation. Future lifecycle operations
// must separately reconcile authority-caused closure and successor creation here;
// a projection mismatch is never treated as an acceptable lifecycle shortcut.
func validateLedger(ctx context.Context, q queryer, limits Limits) (ReplayReport, error) {
	n, err := schemaNumber(ctx, q)
	if err != nil {
		return ReplayReport{}, err
	}
	switch n {
	case SchemaVersion:
		return validateOrdinaryLedger(ctx, q, limits)
	case compositionSchemaVersion:
		enabled, err := compositionProtocol(ctx, q)
		if err != nil {
			return ReplayReport{}, err
		}
		if !enabled {
			return ReplayReport{}, ErrVersion
		}
		return validateCompositionStore(ctx, q, limits)
	default:
		return ReplayReport{}, ErrVersion
	}
}

func validateOrdinaryLedger(ctx context.Context, q queryer, limits Limits) (ReplayReport, error) {
	report := ReplayReport{Runs: []RunState{}}
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return report, err
	}
	report.Namespace = namespace
	if err := validateAdapters(ctx, q, limits); err != nil {
		return report, err
	}
	if err := q.QueryRowContext(ctx, "SELECT sqlite_version(),sqlite_source_id()").Scan(&report.SQLiteVersion, &report.SQLiteSourceID); err != nil {
		return report, err
	}
	var integrity string
	if err := q.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		return report, err
	}
	if integrity != "ok" {
		return report, corrupt("SQLite integrity: %s", integrity)
	}
	fk, err := q.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return report, err
	}
	hasViolation := fk.Next()
	fkErr := fk.Err()
	closeErr := fk.Close()
	if fkErr != nil {
		return report, fkErr
	}
	if closeErr != nil {
		return report, closeErr
	}
	if hasViolation {
		return report, corrupt("foreign key violation")
	}
	ids, err := stringList(ctx, q, "SELECT id FROM programs ORDER BY id")
	if err != nil {
		return report, err
	}
	programs := make(map[string]Program, len(ids))
	for _, id := range ids {
		p, e := loadProgram(ctx, q, id, limits)
		if e != nil {
			return report, e
		}
		programs[id] = p
	}
	observations, err := readObservations(ctx, q, limits)
	if err != nil {
		return report, err
	}
	intents, err := readIntents(ctx, q)
	if err != nil {
		return report, err
	}
	runIDs, err := readRunIDs(ctx, q)
	if err != nil {
		return report, err
	}
	usedObs := map[string]bool{}
	usedIntents := map[string]bool{}
	for _, id := range runIDs {
		run, e := loadRun(ctx, q, id)
		if e != nil {
			return report, e
		}
		p, ok := programs[run.ProgramID]
		if !ok {
			return report, corrupt("missing run artifact")
		}
		p, e = boundCompositionProgram(ctx, q, p, id, limits)
		if e != nil {
			return report, e
		}
		var initialRaw []byte
		if e = q.QueryRowContext(ctx, "SELECT initial FROM runs WHERE subject=? AND generation=?", id.Subject, id.Generation).Scan(&initialRaw); e != nil {
			return report, e
		}
		var projection Projection
		if e = decode(initialRaw, &projection); e != nil {
			return report, corrupt("invalid initial: %v", e)
		}
		if e = validateAuthorInitial(p, projection); e != nil {
			return report, e
		}
		if e = validateProjection(projection, p, limits); e != nil {
			return report, e
		}
		if projection.Status != "open" {
			return report, corrupt("initial run is closed")
		}
		history, e := readHistory(ctx, q, id)
		if e != nil {
			return report, e
		}
		var previousEpoch int64
		for i, h := range history {
			if h.seq != int64(i+1) || h.epoch < 1 || h.epoch < previousEpoch || h.epoch > run.Epoch || h.revision < 1 || h.revision > run.Revision {
				return report, corrupt("history identity/authority for %+v sequence %d", id, h.seq)
			}
			previousEpoch = h.epoch
			var selected *Observation
			if h.selected.Valid {
				ob, ok := observations[h.selected.String]
				if !ok || usedObs[h.selected.String] || ob.Run != id || !ob.consumed.Valid || ob.consumed.Int64 != h.seq {
					return report, corrupt("observation selection %s", h.selected.String)
				}
				selected = &ob.Observation
				usedObs[h.selected.String] = true
			}
			result, e := Reduce(p, projection, selected, id, h.seq, limits)
			if e != nil {
				return report, corrupt("reduction for %+v sequence %d: %v", id, h.seq, e)
			}
			if !sameJSON(h.successor, result.Projection) || !sameJSON(h.intents, result.Intents) {
				return report, corrupt("history mismatch for %+v sequence %d", id, h.seq)
			}
			for _, intent := range result.Intents {
				stored, ok := intents[intent.ID]
				if !ok || usedIntents[intent.ID] || stored.revision != h.revision || !sameJSON(stored.raw, intent) {
					return report, corrupt("intent mismatch %s", intent.ID)
				}
				usedIntents[intent.ID] = true
			}
			projection = result.Projection
			report.HistoryCount++
		}
		if run.Epoch < 1 || run.Sequence != int64(len(history)) {
			return report, corrupt("current projection/sequence for %+v", id)
		}
		if e = reconcileProjection(ctx, q, run, projection); e != nil {
			return report, e
		}
		if e = validateTerminal(ctx, q, run); e != nil {
			return report, e
		}
		report.Runs = append(report.Runs, run)
	}
	for id, o := range observations {
		if o.consumed.Valid != usedObs[id] {
			return report, corrupt("unexplained consumption %s", id)
		}
	}
	if len(usedIntents) != len(intents) {
		return report, corrupt("unexplained intent")
	}
	report.IntentCount = len(intents)
	// Every current subject points to an actual run. Fresh-run linkage from Retry is
	// added with its lifecycle ledger rather than inferred from numeric generations.
	var invalidSubjects int
	if err = q.QueryRowContext(ctx, `SELECT count(*) FROM subjects s LEFT JOIN runs r ON r.subject=s.id AND r.generation=s.generation WHERE r.subject IS NULL OR s.owner='' OR s.id=''`).Scan(&invalidSubjects); err != nil {
		return report, err
	}
	if invalidSubjects != 0 {
		return report, corrupt("invalid current subject")
	}
	if err = validateLifecycle(ctx, q); err != nil {
		return report, err
	}
	if err = validateMessaging(ctx, q, limits); err != nil {
		return report, err
	}
	return report, nil
}
func stringList(ctx context.Context, q queryer, query string) ([]string, error) {
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func readRunIDs(ctx context.Context, q queryer) ([]RunID, error) {
	rows, err := q.QueryContext(ctx, "SELECT subject,generation FROM runs ORDER BY subject,generation")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RunID{}
	for rows.Next() {
		var id RunID
		if err = rows.Scan(&id.Subject, &id.Generation); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func readObservations(ctx context.Context, q queryer, l Limits) (map[string]retainedObservation, error) {
	rows, err := q.QueryContext(ctx, "SELECT id,subject,generation,descriptor,consumed_sequence FROM observations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]retainedObservation{}
	for rows.Next() {
		var o retainedObservation
		var id, subject string
		var generation int64
		var raw []byte
		if err = rows.Scan(&id, &subject, &generation, &raw, &o.consumed); err != nil {
			return nil, err
		}
		if err = decode(raw, &o.Observation); err != nil {
			return nil, corrupt("observation JSON: %v", err)
		}
		if o.ID != id || o.Run != (RunID{subject, generation}) || id == "" || o.Kind == "" || o.Provenance == "" || o.Source == "" || o.SourceGeneration <= 0 || o.Payload == nil || !sameJSON(raw, o.Observation) {
			return nil, corrupt("observation identity")
		}
		for k, v := range o.Payload {
			if k == "" || !validValue(v) {
				return nil, corrupt("observation payload")
			}
		}
		if err = checkSize(o.Observation, l.MaxObservationBytes); err != nil {
			return nil, err
		}
		out[id] = o
	}
	return out, rows.Err()
}
func readIntents(ctx context.Context, q queryer) (map[string]retainedIntent, error) {
	rows, err := q.QueryContext(ctx, "SELECT id,subject,generation,sequence,ordinal,source_revision,descriptor FROM intents")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]retainedIntent{}
	for rows.Next() {
		var r retainedIntent
		var id, subject string
		var generation, seq int64
		var ordinal int
		if err = rows.Scan(&id, &subject, &generation, &seq, &ordinal, &r.revision, &r.raw); err != nil {
			return nil, err
		}
		if err = decode(r.raw, &r.Intent); err != nil {
			return nil, corrupt("intent JSON: %v", err)
		}
		if r.ID != id || r.Run != (RunID{subject, generation}) || r.Sequence != seq || r.Ordinal != ordinal {
			return nil, corrupt("intent identity")
		}
		out[id] = r
	}
	return out, rows.Err()
}
func readHistory(ctx context.Context, q queryer, id RunID) ([]historyRow, error) {
	rows, err := q.QueryContext(ctx, "SELECT sequence,epoch,source_revision,selected_observation,successor,intents FROM history WHERE subject=? AND generation=? ORDER BY sequence", id.Subject, id.Generation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []historyRow{}
	for rows.Next() {
		var h historyRow
		if err = rows.Scan(&h.seq, &h.epoch, &h.revision, &h.selected, &h.successor, &h.intents); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
func validateTerminal(ctx context.Context, q queryer, r RunState) error {
	var seq int64
	var outcome, destinations, status []byte
	err := q.QueryRowContext(ctx, "SELECT sequence,outcome,destinations,delivery_status FROM terminal_obligations WHERE subject=? AND generation=?", r.ID.Subject, r.ID.Generation).Scan(&seq, &outcome, &destinations, &status)
	if r.Projection.Status == "open" {
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		return corrupt("open run has terminal outcome")
	}
	if err != nil {
		return err
	}
	if seq != r.Sequence || !sameJSON(outcome, r.Projection) {
		return corrupt("terminal outcome mismatch")
	}
	var revision int64
	var operation sql.NullString
	if err = q.QueryRowContext(ctx, "SELECT source_revision,operation_id FROM terminal_obligations WHERE subject=? AND generation=?", r.ID.Subject, r.ID.Generation).Scan(&revision, &operation); err != nil {
		return err
	}
	if !operation.Valid {
		var historical int64
		if err = q.QueryRowContext(ctx, "SELECT source_revision FROM history WHERE subject=? AND generation=? AND sequence=?", r.ID.Subject, r.ID.Generation, r.Sequence).Scan(&historical); err != nil {
			return err
		}
		if historical != revision {
			return corrupt("terminal source revision")
		}
	}
	expected, err := terminalDestinations(ctx, q, r.ID.Subject, revision)
	if err != nil {
		return err
	}
	if !sameJSON(destinations, expected) {
		return corrupt("terminal destinations mismatch")
	}
	var deliveries map[string]bool
	if err = decode(status, &deliveries); err != nil || deliveries == nil {
		return corrupt("invalid terminal delivery status")
	}
	for key := range deliveries {
		found := false
		for _, destination := range expected {
			if key == destination.key() {
				found = true
			}
		}
		if !found {
			return corrupt("unknown terminal delivery destination")
		}
	}
	return nil
}
