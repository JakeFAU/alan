package executor

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func reportedProgress(report RunReport) RunProgress {
	return RunProgress{Steps: report.Steps, Dispatches: report.Dispatches, Quiescent: report.Quiescent}
}

// Fork a checkpointed, genuinely provisioned store so namespace-bound IDs are
// comparable. Both branches go through the same fresh-owner validation.
func forkProgressStore(t *testing.T, path string) (*Store, string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fork := filepath.Join(t.TempDir(), "fork.sqlite")
	if err := os.WriteFile(fork, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(t.Context(), fork, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, fork
}

func assertProgressEvidenceParity(t *testing.T, fullPath, progressPath string) *ExecutionSnapshot {
	t.Helper()
	fullCut, err := InspectExecution(t.Context(), fullPath, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	progressCut, err := InspectExecution(t.Context(), progressPath, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fullCut, progressCut) {
		t.Fatal("typed cuts differ after equivalent execution")
	}
	fullReplay, err := Replay(t.Context(), fullPath, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	progressReplay, err := Replay(t.Context(), progressPath, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fullReplay, progressReplay) {
		t.Fatal("independent Replay results differ")
	}
	return progressCut
}

func TestRunProgressDynamicCompositionParity(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	p.Composition = &c
	seed, path := messageStore(t, p)
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	full, fullPath := forkProgressStore(t, path)
	progress, progressPath := forkProgressStore(t, path)
	if full.execution != nil || progress.execution != nil {
		t.Fatal("dynamic template fixture entered static discovery")
	}

	options := RunOptions{MaxSteps: 40, MaxDispatch: 40}
	want, fullErr := full.Run(t.Context(), options)
	got, progressErr := progress.RunProgress(t.Context(), options)
	if fullErr != nil || progressErr != nil || got != reportedProgress(want) ||
		!got.Quiescent || got.Steps == 0 || got.Dispatches == 0 {
		t.Fatalf("full=%+v/%v progress=%+v/%v", want, fullErr, got, progressErr)
	}
	cut := assertProgressEvidenceParity(t, fullPath, progressPath)
	root, ok := cut.Run(RunID{Subject: "root", Generation: 1})
	if !ok || root.Projection.Status != "succeeded" || len(cut.Invocations) != 2 || len(cut.Roots) != 1 {
		t.Fatalf("expected two dynamic invocations and one root: %+v", root)
	}
	completion := cut.Roots[0]
	if completion.Outcome == nil || completion.Charges.Invocations != 2 || completion.Charges.CreatedRuns != 4 {
		t.Fatalf("dynamic root did not retain completion and admission charges: %+v", completion)
	}
	for _, invocation := range cut.Invocations {
		if invocation.State != InvocationPublished {
			t.Fatalf("invocation did not publish its result: %+v", invocation)
		}
	}
}

func TestRunProgressUnrelatedAdmissionDiagnostic(t *testing.T) {
	s, path := staticWaitingStore(t, 2)
	compactAssessmentComplete(t, s, "r00", "first", 11)

	// Deliberately corrupt an admission that execution of r01 does not use.
	result, err := s.conn.ExecContext(t.Context(),
		"UPDATE admissions SET result=? WHERE domain='input' AND transport_id=?", []byte("{"), "first")
	if err != nil {
		t.Fatal(err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		t.Fatalf("corrupt admission rows=%d error=%v", rows, err)
	}
	p3Admit(t, s, "r01", "second", 22)
	options := RunOptions{MaxSteps: 20, MaxDispatch: 20}
	got, err := s.RunProgress(t.Context(), options)
	state := current(t, s, "r01")
	if err != nil || got != (RunProgress{Steps: 3, Quiescent: true}) ||
		state.Projection.Status != "succeeded" || state.Projection.Data["number"] != Int(22) {
		t.Fatalf("unrelated execution progress=%+v error=%v state=%+v", got, err, state)
	}
	report, err := s.Run(t.Context(), options)
	if !errors.Is(err, io.ErrUnexpectedEOF) || reportedProgress(report) != (RunProgress{Quiescent: true}) {
		t.Fatalf("full Run lost its retained-admission diagnostic: report=%+v error=%v", report, err)
	}
	if _, err := InspectExecution(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("inspection accepted malformed admission")
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("Replay accepted malformed admission")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(t.Context(), path, DefaultLimits()); err == nil {
		reopened.Close()
		t.Fatal("fresh Open accepted malformed admission")
	}
}

func TestRunProgressEntryErrors(t *testing.T) {
	s, _ := messageStore(t, inputFixture(t))
	for _, tc := range []struct {
		name    string
		options RunOptions
	}{
		{"steps", RunOptions{MaxDispatch: 1}},
		{"dispatch", RunOptions{MaxSteps: 1}},
		{"in_flight", RunOptions{MaxSteps: 1, MaxDispatch: 1, MaxInFlight: -1}},
		{"timeout", RunOptions{MaxSteps: 1, MaxDispatch: 1, ExternalTimeout: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.RunProgress(t.Context(), tc.options)
			if !errors.Is(err, ErrLimit) || got != (RunProgress{}) {
				t.Fatalf("invalid limits progress=%+v error=%v", got, err)
			}
		})
	}
	options := RunOptions{MaxSteps: 1, MaxDispatch: 1}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := s.RunProgress(ctx, options)
	if !errors.Is(err, context.Canceled) || got != (RunProgress{}) {
		t.Fatalf("canceled progress=%+v error=%v", got, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	got, err = s.RunProgress(t.Context(), options)
	if !errors.Is(err, sql.ErrConnDone) || got != (RunProgress{}) {
		t.Fatalf("closed progress=%+v error=%v", got, err)
	}
}

func TestRunProgressPerCallStepLimit(t *testing.T) {
	s, path := messageStore(t, inputFixture(t))
	p3Admit(t, s, "receiver", "input", 8)
	options := RunOptions{MaxSteps: 1, MaxDispatch: 1}
	first, err := s.RunProgress(t.Context(), options)
	if !errors.Is(err, ErrLimit) || first != (RunProgress{Steps: 1}) {
		t.Fatalf("step limit progress=%+v error=%v", first, err)
	}
	second, err := s.RunProgress(t.Context(), options)
	if err != nil || second != (RunProgress{Steps: 1, Quiescent: true}) {
		t.Fatalf("next call progress=%+v error=%v", second, err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestRunProgressPerCallDispatchLimit(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	p.Composition = &c
	s, path := messageStore(t, p)

	options := RunOptions{MaxSteps: 40, MaxDispatch: 1}
	first, err := s.RunProgress(t.Context(), options)
	if !errors.Is(err, ErrLimit) || first.Dispatches != 1 || first.Quiescent {
		t.Fatalf("dispatch limit progress=%+v error=%v", first, err)
	}
	second, err := s.RunProgress(t.Context(), options)
	if !errors.Is(err, ErrLimit) || second.Dispatches != 1 || second.Quiescent {
		t.Fatalf("next dispatch budget progress=%+v error=%v", second, err)
	}
	options.MaxDispatch = 40
	last, err := s.RunProgress(t.Context(), options)
	if err != nil || !last.Quiescent || current(t, s, "root").Projection.Status != "succeeded" {
		t.Fatalf("finish after dispatch limits progress=%+v error=%v", last, err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
