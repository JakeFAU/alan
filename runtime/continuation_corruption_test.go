//go:build integration

package runtime_test

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// All damage is applied after closing a private copy. Main and durable WAL
// bytes must survive every rejecting reader, including unsuccessful renewal.
func copyMeterLedger(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "corrupt.sqlite")
	for _, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(source + suffix)
		if suffix == "-wal" && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path+suffix, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func meterFingerprint(t *testing.T, path string) [2][32]byte {
	t.Helper()
	var digest [2][32]byte
	for i, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(path + suffix)
		if err != nil && !(suffix == "-wal" && errors.Is(err, os.ErrNotExist)) {
			t.Fatal(err)
		}
		digest[i] = sha256.Sum256(data)
	}
	return digest
}

func rejectMeterLedger(t *testing.T, path string) {
	t.Helper()
	before := meterFingerprint(t, path)
	ctx, l := t.Context(), runtime.DefaultLimits()
	cut, inspect := runtime.InspectExecution(ctx, path, l)
	_, raw := runtime.Inspect(ctx, path, l)
	_, replay := runtime.Replay(ctx, path, l)
	fresh, open := runtime.Open(ctx, path, l)
	if fresh != nil {
		fresh.Close()
	}
	for name, err := range map[string]error{"typed": inspect, "raw": raw, "Replay": replay, "Open": open} {
		if !errors.Is(err, runtime.ErrCorrupt) && !errors.Is(err, runtime.ErrVersion) {
			t.Errorf("%s accepted inconsistent evidence or returned an unrelated error: %v", name, err)
		}
	}
	if cut != nil || fresh != nil || before != meterFingerprint(t, path) {
		t.Fatal("rejected history returned a graph, new owner, or changed durable bytes")
	}
}

func meterExec(t *testing.T, tx *sql.Tx, query string, args ...any) {
	t.Helper()
	if _, err := tx.ExecContext(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func changeMeterLedger(t *testing.T, path string, change func(*sql.Tx)) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	change(tx)
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// Set the same false claim everywhere it occurs, then independently rehash its
// composition fact. A stale hash or an inconsistent copy in the local input
// must not be the only reason the readers detect these forgeries.
func forgeMeterApproval(t *testing.T, tx *sql.Tx, field, payload string, value any) {
	t.Helper()
	text, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	meterExec(t, tx, "UPDATE admissions SET descriptor=CAST(json_set(descriptor,?,json(?)) AS BLOB) WHERE transport_id=? AND domain='input'",
		"$.Continuation."+field, string(text), meterApproval)
	if payload != "" {
		path := "$.Payload." + payload + ".Text"
		meterExec(t, tx, "UPDATE admissions SET descriptor=CAST(json_set(descriptor,?,json(?)) AS BLOB) WHERE transport_id=? AND domain='input'", path, string(text), meterApproval)
		meterExec(t, tx, "UPDATE observations SET descriptor=CAST(json_set(descriptor,?,json(?)) AS BLOB) WHERE id=(SELECT observation_id FROM admissions WHERE transport_id=? AND domain='input')", path, string(text), meterApproval)
	}
	var d, r json.RawMessage
	if err := tx.QueryRowContext(t.Context(), "SELECT descriptor,result FROM admissions WHERE transport_id=? AND domain='input'", meterApproval).Scan(&d, &r); err != nil {
		t.Fatal(err)
	}
	evidence, err := json.Marshal([]any{"alan/continuation-admission/v1", []json.RawMessage{d, r}})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(evidence)
	meterExec(t, tx, "UPDATE composition_facts SET descriptor=CAST(json_set(descriptor,'$.RecordDigest',?) AS BLOB) WHERE kind='continuation'", hex.EncodeToString(digest[:]))
}

// These facts have no subordinate charge or admission-order foreign keys. An
// adversary moves their complete descriptors and fixes both copies of Order.
func swapMeterFacts(t *testing.T, tx *sql.Tx, first, second int64) {
	t.Helper()
	if first == second || first < 1 || second < 1 {
		t.Fatal("invalid independent fact pair")
	}
	move := func(from, to int64) {
		meterExec(t, tx, "UPDATE composition_facts SET decision_order=?,descriptor=CAST(json_set(descriptor,'$.Order',?) AS BLOB) WHERE decision_order=?", to, to, from)
	}
	move(first, 1<<50)
	move(second, first)
	move(1<<50, second)
}

func TestContinuationLedgerRejectsRehashedCorruption(t *testing.T) {
	f := newMeterFixture(t, "north")
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	b := meterBasis(t, f)
	a, err := f.owner.ApproveContinuation(t.Context(), b, "second sample")
	if err != nil || !a.Accepted {
		t.Fatalf("setup approval: %v %v", a, err)
	}
	if err = f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	cut := meterCut(t, f.path)
	view, ok := cut.Continuation(meterNext)
	if !ok || view.SelectedAt == nil || view.SuccessorOperationID == "" || meterSpent(cut) != 2 {
		t.Fatal("positive history incomplete")
	}
	if err = f.owner.Close(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*testing.T, *sql.Tx)
	}{
		{"namespace", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "Namespace", "", "unrelated-namespace") }},
		{"source_operation", func(t *testing.T, tx *sql.Tx) {
			forgeMeterApproval(t, tx, "PredecessorID", "previous_operation_id", view.SuccessorOperationID)
		}},
		{"absent_source", func(t *testing.T, tx *sql.Tx) {
			forgeMeterApproval(t, tx, "PredecessorID", "previous_operation_id", "uncommitted")
		}},
		{"target_subject", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "Target.Subject", "", meterOriginal.Subject) }},
		{"target_generation", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "Target.Generation", "", 2) }},
		{"target_revision", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "TargetRevision", "", 2) }},
		{"prepared_digest", func(t *testing.T, tx *sql.Tx) {
			forgeMeterApproval(t, tx, "PreparedDigest", "prepared_digest", b.ResultDigest)
		}},
		{"different_result", func(t *testing.T, tx *sql.Tx) {
			forgeMeterApproval(t, tx, "ResultDigest", "result_digest", b.PreparedDigest)
		}},
		{"hidden_known_result", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "ResultDigest", "result_digest", "") }},
		{"hidden_original_gate", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "Admitted", "", false) }},
		{"different_prepared_action", func(t *testing.T, tx *sql.Tx) { forgeMeterApproval(t, tx, "ScopeDigest", "", b.PreparedDigest) }},
		{"approval_after_new_effect", func(t *testing.T, tx *sql.Tx) {
			e, _ := cut.Effect(view.SuccessorOperationID)
			var n int64
			if err := tx.QueryRowContext(t.Context(), `SELECT decision_order FROM composition_facts WHERE kind='execution' AND json_extract(descriptor,'$.Run.Subject')=? AND json_extract(descriptor,'$.Sequence')=?`, meterNext.Subject, e.Sequence).Scan(&n); err != nil {
				t.Fatal(err)
			}
			swapMeterFacts(t, tx, view.DecisionOrder, n)
		}},
		{"fabricated_effect_result_correlation", func(t *testing.T, tx *sql.Tx) {
			meterExec(t, tx, "UPDATE adapter_results SET descriptor=CAST(json_set(descriptor,'$.Candidate.ID',?) AS BLOB) WHERE intent_id=?", b.PredecessorID, view.SuccessorOperationID)
		}},
		{"unpaid_new_opportunity", func(t *testing.T, tx *sql.Tx) {
			meterExec(t, tx, "DELETE FROM composition_charges WHERE operation_id=? AND kind='effect'", view.SuccessorOperationID)
		}},
		{"missing_original_input", func(t *testing.T, tx *sql.Tx) {
			meterExec(t, tx, "DELETE FROM observations WHERE id=?", view.ObservationID)
		}},
		{"missing_decision", func(t *testing.T, tx *sql.Tx) {
			meterExec(t, tx, "DELETE FROM composition_facts WHERE kind='continuation'")
		}},
		{"masked_older_format", func(t *testing.T, tx *sql.Tx) {
			meterExec(t, tx, "UPDATE metadata SET schema_version=9")
			meterExec(t, tx, "PRAGMA user_version=9")
		}},
		{"duplicate_new_transport", func(t *testing.T, tx *sql.Tx) {
			meterExec(t, tx, `ALTER TABLE adapter_dispatch_admissions RENAME TO old_gate;
				CREATE TABLE adapter_dispatch_admissions(intent_id TEXT,prepared_digest TEXT,after_fact INTEGER);
				INSERT INTO adapter_dispatch_admissions SELECT * FROM old_gate`)
			meterExec(t, tx, "INSERT INTO adapter_dispatch_admissions SELECT * FROM old_gate WHERE intent_id=?", view.SuccessorOperationID)
			meterExec(t, tx, "DROP TABLE old_gate")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := copyMeterLedger(t, f.path)
			changeMeterLedger(t, path, func(tx *sql.Tx) { test.change(t, tx) })
			rejectMeterLedger(t, path)
		})
	}
}

func TestContinuationLedgerDistinguishesOppositeDecisionOrders(t *testing.T) {
	f, b := lateResultFixture(t)
	cut := meterCut(t, f.path)
	v, ok := cut.Continuation(meterNext)
	if !ok || v.Approval.ResultDigest != "" {
		t.Fatal("expected approval with original outstanding")
	}
	if err := f.owner.Close(); err != nil {
		t.Fatal(err)
	}
	// The original did eventually return this exact result. Rehashing the
	// input to claim that the response already existed at approval is a
	// forged completion even though the final result and all counters match.
	original, ok := cut.Effect(b.PredecessorID)
	if !ok || original.AdapterResult == nil {
		t.Fatal("missing real late result")
	}
	fake := copyMeterLedger(t, f.path)
	changeMeterLedger(t, fake, func(tx *sql.Tx) {
		forgeMeterApproval(t, tx, "ResultDigest", "result_digest", original.AdapterResult.Digest)
	})
	rejectMeterLedger(t, fake)
	path := copyMeterLedger(t, f.path)
	changeMeterLedger(t, path, func(tx *sql.Tx) {
		var result int64
		if err := tx.QueryRowContext(t.Context(), `SELECT decision_order FROM composition_facts WHERE kind='external_result' AND json_extract(descriptor,'$.OperationID')=?`, b.PredecessorID).Scan(&result); err != nil {
			t.Fatal(err)
		}
		if result <= v.DecisionOrder {
			t.Fatal("original did not finish after issuance")
		}
		swapMeterFacts(t, tx, result, v.DecisionOrder)
		var charges, gates int
		if err := tx.QueryRowContext(t.Context(), `SELECT (SELECT count(*) FROM composition_charges WHERE kind='effect'),(SELECT count(*) FROM adapter_dispatch_admissions)`).Scan(&charges, &gates); err != nil || charges != 2 || gates != 2 {
			t.Fatalf("counterexample lost matching counters: %d %d %v", charges, gates, err)
		}
	})
	rejectMeterLedger(t, path)
}
