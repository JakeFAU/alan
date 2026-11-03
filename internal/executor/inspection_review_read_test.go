package executor

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

// Stop the owner between each protocol action. The root's publication can be
// ready even while its two rejected calls have no published child outcomes.
func TestInspectionMatchesReplayBeforeRootPublication(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "root.sqlite")
	provision, composition := invocationFixture(t, "succeeded", 1)
	s, err := createInvocationStore(ctx, path, provision, composition, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for range 3 {
		if _, err := s.Commit(ctx, current(t, s, "root").Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}

	check := func(wantPending string) {
		t.Helper()
		replayed, err := Replay(ctx, path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if len(replayed.Roots) != 1 || replayed.Roots[0].Outcome != nil || replayed.Roots[0].Pending != wantPending {
			t.Fatalf("unexpected publication cut for %q: %+v", wantPending, replayed.Roots)
		}
		cut, err := InspectExecution(ctx, path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cut.Roots, replayed.Roots) {
			t.Errorf("root blocked on %q: inspection %+v, replay %+v", wantPending, cut.Roots, replayed.Roots)
		}
	}
	check("invocation_admission")

	for _, sequence := range []int64{1, 2} {
		decision, err := s.AdmitInvocation(ctx, identity("root", 1, sequence, 0))
		if err != nil || decision.Accepted {
			t.Fatalf("expected a closed admission, got %+v, %v", decision, err)
		}
	}
	check("")
	if report, err := s.Run(ctx, RunOptions{MaxSteps: 50, MaxDispatch: 50, MaxInFlight: 1}); err != nil || !report.Quiescent {
		t.Fatal(report, err)
	}
	cut, err := InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	parent, ok := cut.Run(RunID{"root", 1})
	if !ok || parent.LocalState != RunTerminalSettled || len(cut.Pending) != 0 || cut.Roots[0].Outcome == nil {
		t.Fatal("closed parent incorrectly owes a selection", parent, cut.Pending)
	}
	for _, v := range cut.Invocations {
		if v.Outcome == nil || v.SelectedAt != nil || v.ParentSelection != "consumer_closed" {
			t.Fatal("lost unconsumed publication", v)
		}
	}
}

func TestInspectionHonorsStoredOutputAndObservationLimits(t *testing.T) {
	ctx := t.Context()
	s, path := newStore(t, false)
	r := state(t, s)
	if _, err := s.Commit(ctx, r.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	o := Observation{ID: "observed", Run: r.ID, Kind: "input", Provenance: "fixture", Source: "sender", SourceGeneration: 1, Payload: map[string]Value{}}
	if err := s.recordObservation(ctx, o); err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits()
	if _, err := InspectExecution(ctx, path, limits); err != nil {
		t.Fatal(err)
	}
	output, observation := limits, limits
	output.MaxOutputBytes = 1
	observation.MaxObservationBytes = 1
	for _, l := range []Limits{output, observation} {
		if cut, err := InspectExecution(ctx, path, l); cut != nil || !errors.Is(err, ErrLimit) {
			t.Fatalf("limits %+v: %v, %v", l, cut, err)
		}
	}
}

func TestInspectionRejectsPartiallyDowngradedSchema(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "partial.sqlite")
	provision, composition := invocationFixture(t, "succeeded", 1)
	s, err := createInvocationStore(ctx, path, provision, composition, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// The retained root, permissions and invocation tables still belong to
	// the composition protocol even after its primary profile disappears.
	if _, err := s.conn.ExecContext(ctx, "DROP TABLE composition_profile; UPDATE metadata SET schema_version=3; PRAGMA user_version=3"); err != nil {
		t.Fatal(err)
	}
	if cut, err := InspectExecution(ctx, path, DefaultLimits()); cut != nil || !errors.Is(err, ErrVersion) {
		t.Fatalf("partial downgrade: %v, %v", cut, err)
	}
}
