package executor

import (
	"errors"
	"io"
	"testing"
)

// This genuine public-execution history is also the positive control for the
// report-bypass overlay retained with the compact-result assessment. It checks
// execution facts independently of the optional shape of returned report lists.
func TestRunCompactAssessmentGenuineHistory(t *testing.T) {
	s, path := staticWaitingStore(t, 2)
	compactAssessmentComplete(t, s, "r00", "first", 11)
	compactAssessmentComplete(t, s, "r01", "second", 22)
	if count(t, s, "admissions") != 2 || count(t, s, "observations") != 2 {
		t.Fatal("genuine admissions/observations missing")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	t.Log("two ordinary inputs completed; independent Inspect, Open, Replay succeeded")
}

// The unsupported private UPDATE is a deliberately corrupt disposable store,
// not a public history or evidence of unauthorized dispatch. The second root's
// valid execution does not revisit the first root's historical admission result;
// full reporting still detects it. Skipping finishRun changes that coverage.
func TestRunCompactAssessmentRetainedAdmission(t *testing.T) {
	s, _ := staticWaitingStore(t, 2)
	compactAssessmentComplete(t, s, "r00", "first", 11)
	result, err := s.conn.ExecContext(t.Context(),
		"UPDATE admissions SET result=? WHERE domain='input' AND transport_id=?", []byte("{"), "first")
	if err != nil {
		t.Fatal(err)
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		t.Fatalf("private corruption affected %d rows: %v", n, err)
	}
	// This legitimate later write must not erase the report's integrity coverage.
	p3Admit(t, s, "r01", "second", 22)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	state := current(t, s, "r01")
	if state.Projection.Status != "succeeded" || state.Projection.Data["number"] != Int(22) ||
		report.Steps != 3 || report.Dispatches != 0 {
		t.Fatalf("unrelated authorized execution changed: state=%+v report=%+v error=%v", state, report, err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("retained admission diagnostic lost after valid unrelated execution: report=%+v error=%v", report, err)
	}
	// Existing Quiescent assignment precedes reporting; a reporting error does
	// not retroactively change it. This test does not redefine that contract.
	if !report.Quiescent {
		t.Fatalf("report error changed existing quiescence semantics: %+v", report)
	}
	t.Logf("unrelated root completed in %d steps with %d dispatches; retained admission rejected: %v",
		report.Steps, report.Dispatches, err)
}

func compactAssessmentComplete(t *testing.T, s *Store, subject, transport string, value int64) {
	t.Helper()
	p3Admit(t, s, subject, transport, value)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	state := current(t, s, subject)
	if err != nil || !report.Quiescent || report.Steps != 3 || report.Dispatches != 0 ||
		state.Projection.Status != "succeeded" || state.Projection.Data["number"] != Int(value) {
		t.Fatalf("genuine execution: state=%+v report=%+v error=%v", state, report, err)
	}
}
