package executor

import (
	"reflect"
	"testing"
)

func TestInspectionExposesOneReadOnlyRecordSnapshot(t *testing.T) {
	s, path := messageStore(t, inputFixture(t))
	if _, err := s.AdmitInput(t.Context(), Input{TransportID: "transport", LogicalID: "logical", Target: "receiver", Obligation: "work", Payload: map[string]Value{"number": Int(12)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10}); err != nil {
		t.Fatal(err)
	}
	before := current(t, s, "receiver")
	snapshot, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"metadata", "programs", "subjects", "permissions", "runs", "history", "observations", "intents", "attempts", "admissions", "lifecycle_results", "settlements", "terminal_obligations", "message_acknowledgments"} {
		if _, ok := snapshot.Tables[table]; !ok {
			t.Fatalf("missing %s", table)
		}
	}
	if snapshot.Settings.JournalMode != "wal" || snapshot.Settings.Synchronous != 2 || snapshot.Settings.QueryOnly != 1 || snapshot.Settings.SQLiteSourceID == "" {
		t.Fatalf("settings %+v", snapshot.Settings)
	}
	if len(snapshot.Tables["observations"]) != 1 || snapshot.Tables["observations"][0]["consumed_sequence"] != int64(1) || len(snapshot.Tables["history"]) != 2 || len(snapshot.Tables["admissions"]) != 1 {
		t.Fatalf("record snapshot %+v", snapshot.Tables)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	after, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot, after) || !reflect.DeepEqual(before, current(t, s, "receiver")) {
		t.Fatal("inspection/replay changed durable records")
	}
}
func TestInspectionDoesNotReduceOrdinaryPolicy(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "not-ready", []Instruction{{Op: "set", Dst: "x", A: Ref("missing")}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	_, path := messageStore(t, p)
	if _, err := Inspect(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatalf("inspection tried to execute policy: %v", err)
	}
}
