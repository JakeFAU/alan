package executor

import (
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	recordschema "github.com/JakeFAU/alan/schema"
)

func TestRunnerIsolatesReductionFault(t *testing.T) {
	input, err := (recordschema.Record{"short": {Kind: "string", MaxBytes: 8}}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	output, err := (recordschema.Record{"short": {Kind: "string", MaxBytes: 1}}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	// Retained code can reach an invalid output even if the current compiler rejects it.
	program := Program{
		Semantics: AuthorSemanticVersion,
		Code:      []Instruction{{Op: "end", Outcome: "succeeded"}},
		Author: &ProgramMetadata{
			Compiler: AuthorCompilerVersion, Positions: []string{"end"},
			Inputs: input, Outputs: output,
		},
	}
	encoded, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	fault := RunID{Subject: "a-fault", Generation: 1}
	provision := Provision{
		Programs: []Artifact{{ID: id, Semantics: program.Semantics, Bytes: encoded}},
		Subjects: []Subject{{ID: fault.Subject, Generation: 1, Revision: 1, Owner: "owner"}},
		Runs: []InitialRun{{
			ID: fault, ProgramID: id,
			Projection: Projection{Status: "open", Data: map[string]Value{"short": Text("oversize")}},
		}},
	}
	addProgram(t, &provision, "z-healthy", []Instruction{
		{Op: "set", Dst: "ran", A: Lit(Int(1))},
		{Op: "end", Outcome: "succeeded"},
	}, map[string]Value{})
	for _, static := range []bool{false, true} {
		t.Run(faultIsolationProfile(static), func(t *testing.T) {
			limits := DefaultLimits()
			s, path := faultIsolationStore(t, provision, limits, static)
			blocked := current(t, s, fault.Subject)
			for _, wantSteps := range []int{2, 0} {
				report, err := s.Run(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8})
				healthy := current(t, s, "z-healthy")
				if report.Steps != wantSteps || healthy.Sequence != 2 || healthy.Projection.Status != "succeeded" {
					t.Fatalf("healthy peer after Run: steps %d, want %d, state %+v, error %v", report.Steps, wantSteps, healthy, err)
				}
				if err == nil || !strings.Contains(err.Error(), "a-fault") || !strings.Contains(err.Error(), "output schema") ||
					report.Quiescent || !slices.Equal(report.OpenRuns, []RunID{fault}) {
					t.Fatalf("blocked peer after Run: %+v, %v", report, err)
				}
				if got := current(t, s, fault.Subject); !reflect.DeepEqual(got, blocked) {
					t.Fatalf("fault changed run: got %+v, want %+v", got, blocked)
				}
				if static && s.execution == nil {
					t.Fatal("lost static execution index while isolating fault")
				}
			}
			replay, err := Replay(t.Context(), path, limits)
			if err != nil || replay.HistoryCount != 2 {
				t.Fatalf("Replay: %+v, %v", replay, err)
			}
		})
	}
}

func TestRunnerSkipsObservationWhoseBindingExceedsProjectionLimit(t *testing.T) {
	for _, static := range []bool{false, true} {
		t.Run(faultIsolationProfile(static), func(t *testing.T) {
			s, path, limits := faultIsolationReceiver(t, static)
			larger := admitFaultIsolationInput(t, s, "1-oversize", strings.Repeat("a", 1024))
			smaller := admitFaultIsolationInput(t, s, "2-fits", "fits")
			if static {
				s = reopenFaultIsolationStaticStore(t, s, path, limits)
			}
			if _, err := s.Commit(t.Context(), current(t, s, "receiver").Authority(), larger); !errors.Is(err, ErrLimit) {
				t.Fatalf("oversize binding: %v", err)
			}

			report, err := s.Run(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8})
			receiver := current(t, s, "receiver")
			if err != nil || !report.Quiescent || report.Steps != 2 ||
				receiver.Projection.Status != "succeeded" || receiver.Projection.Data["number"] != Text("fits") {
				t.Fatalf("Run with valid observation behind oversized one: %+v, %v, receiver %+v", report, err, receiver)
			}
			if static && s.execution == nil {
				t.Fatal("lost static execution index while skipping candidate")
			}
			inbox, err := readObservations(t.Context(), s.conn, limits)
			if err != nil || len(inbox) != 2 || inbox[larger].consumed.Valid || !inbox[smaller].consumed.Valid ||
				inbox[smaller].consumed.Int64 != 1 {
				t.Fatalf("observations: %+v, %v", inbox, err)
			}
			replay, err := Replay(t.Context(), path, limits)
			if err != nil || replay.HistoryCount != 2 {
				t.Fatalf("Replay: %+v, %v", replay, err)
			}
		})
	}
}

func TestRunnerReportsFirstInvalidObservation(t *testing.T) {
	for _, static := range []bool{false, true} {
		t.Run(faultIsolationProfile(static), func(t *testing.T) {
			s, path, limits := faultIsolationReceiver(t, static)
			first := admitFaultIsolationInput(t, s, "1-oversize", strings.Repeat("a", 1024))
			second := admitFaultIsolationInput(t, s, "2-oversize", strings.Repeat("b", 1024))
			if static {
				s = reopenFaultIsolationStaticStore(t, s, path, limits)
			}
			before := current(t, s, "receiver")
			for range 2 {
				report, err := s.RunProgress(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8})
				if !errors.Is(err, ErrLimit) || !strings.Contains(err.Error(), "1-oversize") ||
					strings.Contains(err.Error(), "2-oversize") || report.Quiescent || report.Steps != 0 {
					t.Fatalf("all-invalid RunProgress: %+v, %v", report, err)
				}
				if got := current(t, s, "receiver"); !reflect.DeepEqual(got, before) {
					t.Fatalf("receive fault changed run: got %+v, want %+v", got, before)
				}
				if static && s.execution == nil {
					t.Fatal("lost static execution index for blocked receive")
				}
			}
			inbox, err := readObservations(t.Context(), s.conn, limits)
			if err != nil || len(inbox) != 2 || inbox[first].consumed.Valid || inbox[second].consumed.Valid {
				t.Fatalf("observations: %+v, %v", inbox, err)
			}
		})
	}
}

func TestRunnerSkipsMalformedReceiveCandidates(t *testing.T) {
	empty, err := (recordschema.Record{}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	number, err := (recordschema.Record{"number": {Kind: "string", MaxBytes: 2}}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		name, diagnosis string
		program         Program
		payload         map[string]Value
	}{
		{
			name: "missing binding", diagnosis: "invalid receive binding",
			program: Program{Semantics: SemanticVersion, Code: inputProgram()},
			payload: map[string]Value{},
		},
		{
			name: "schema mismatch", diagnosis: "receive schema",
			program: Program{Semantics: AuthorSemanticVersion, Code: inputProgram(), Author: &ProgramMetadata{
				Compiler: AuthorCompilerVersion, Positions: []string{"receive", "end"},
				Inputs: empty, Outputs: empty, Receives: map[int]recordschema.Retained{0: number},
			}},
			payload: map[string]Value{"number": Text("too wide")},
		},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			encoded, id, err := EncodeProgram(fixture.program)
			if err != nil {
				t.Fatal(err)
			}
			run := RunID{Subject: "receiver", Generation: 1}
			s, _ := messageStore(t, Provision{
				Programs: []Artifact{{ID: id, Semantics: fixture.program.Semantics, Bytes: encoded}},
				Subjects: []Subject{{ID: run.Subject, Generation: 1, Revision: 1, Owner: "owner"}},
				Runs: []InitialRun{{
					ID: run, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{}},
				}},
			})
			bad := Observation{ID: "1-invalid", Run: run, Kind: "input", Provenance: "local-input",
				Source: "local", SourceGeneration: 1, Obligation: "work", Payload: fixture.payload}
			good := bad
			good.ID = "2-valid"
			good.Payload = map[string]Value{"number": Text("ok")}
			for _, o := range []Observation{bad, good} {
				if err = s.recordObservation(t.Context(), o); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.Commit(t.Context(), current(t, s, run.Subject).Authority(), bad.ID); err == nil ||
				!strings.Contains(err.Error(), fixture.diagnosis) {
				t.Fatalf("direct commit of malformed candidate: %v", err)
			}
			report, err := s.RunProgress(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8})
			if err != nil || !report.Quiescent || report.Steps != 2 || current(t, s, run.Subject).Projection.Data["number"] != Text("ok") {
				t.Fatalf("RunProgress with valid candidate behind malformed one: %+v, %v", report, err)
			}
			inbox, err := readObservations(t.Context(), s.conn, DefaultLimits())
			if err != nil || inbox[bad.ID].consumed.Valid || !inbox[good.ID].consumed.Valid ||
				inbox[good.ID].consumed.Int64 != 1 {
				t.Fatalf("observations: %+v, %v", inbox, err)
			}
		})
	}
}

func TestRunnerFailsFastOnArtifactCorruption(t *testing.T) {
	provision := Provision{}
	addProgram(t, &provision, "a-corrupt", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	addProgram(t, &provision, "z-healthy", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	for _, static := range []bool{false, true} {
		t.Run(faultIsolationProfile(static), func(t *testing.T) {
			s, _ := faultIsolationStore(t, provision, DefaultLimits(), static)
			before := current(t, s, "z-healthy")
			if _, err := s.conn.ExecContext(t.Context(), "UPDATE programs SET bytes=? WHERE id=?", []byte("{}"), provision.Runs[0].ProgramID); err != nil {
				t.Fatal(err)
			}
			report, err := s.Run(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8})
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "artifact hash") || report.Quiescent || report.Steps != 0 {
				t.Fatalf("corrupt artifact Run: %+v, %v", report, err)
			}
			if got := current(t, s, "z-healthy"); !reflect.DeepEqual(got, before) {
				t.Fatalf("healthy run advanced past corrupt artifact: %+v", got)
			}
		})
	}
}

func faultIsolationProfile(static bool) string {
	if static {
		return "static"
	}
	return "baseline"
}

func faultIsolationStore(t *testing.T, provision Provision, limits Limits, static bool) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fault-isolation.sqlite")
	var s *Store
	var err error
	if static {
		composition := CompositionProvision{Version: compositionProtocolVersion}
		for _, run := range provision.Runs {
			composition.Roots = append(composition.Roots, ResourceRoot{
				Run: run.ID, Members: []RunID{run.ID},
				Limits: ResourceLimits{MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxEffects: 2, MaxInFlight: 1},
			})
		}
		s, err = createInvocationStore(t.Context(), path, provision, composition, limits)
	} else {
		s, err = Create(t.Context(), path, provision, limits)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	if (s.execution != nil) != static {
		t.Fatalf("static execution index enabled = %t, want %t", s.execution != nil, static)
	}
	return s, path
}

func reopenFaultIsolationStaticStore(t *testing.T, s *Store, path string, limits Limits) *Store {
	t.Helper()
	// Ordinary input admissions disable the index. Reopen validates the
	// retained inputs and seeds the real index from the validated ledger.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Error(err)
		}
	})
	if reopened.execution == nil {
		t.Fatal("reopen did not enable static execution index")
	}
	return reopened
}

func faultIsolationReceiver(t *testing.T, static bool) (*Store, string, Limits) {
	t.Helper()
	provision := Provision{Permissions: []Permission{
		{Kind: "input", Source: "local", Target: "receiver", Operation: "input", Obligation: "work"},
	}}
	addProgram(t, &provision, "receiver", inputProgram(), map[string]Value{})
	limits := DefaultLimits()
	// Leave room for the composition freeze at End while the 1024-byte
	// payload still cannot fit the receive's successor projection.
	limits.MaxDataBytes = 512
	s, path := faultIsolationStore(t, provision, limits, static)
	return s, path, limits
}

func admitFaultIsolationInput(t *testing.T, s *Store, id, value string) string {
	t.Helper()
	a, err := s.AdmitInput(t.Context(), Input{TransportID: id, LogicalID: id, Target: "receiver",
		Obligation: "work", Payload: map[string]Value{"number": Text(value)}})
	if err != nil || !a.Accepted {
		t.Fatalf("admit %s: %+v, %v", id, a, err)
	}
	return a.ObservationID
}
