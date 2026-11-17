package author_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
)

// Rebinding a serialized operation from a different declaration must not consume
// an authentic but unrelated result, even when both declarations share a template.
func TestSerializedInvocationHandleCannotRebindOutcome(t *testing.T) {
	// This provision was compiled before the provenance check. Keeping its bytes
	// exercises runtime compatibility and the historical wrong-declaration attack.
	raw, err := os.ReadFile("testdata/prechange-wrong-invocation.json")
	if err != nil {
		t.Fatal(err)
	}
	var p runtime.Provision
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	var root executor.Program
	for _, artifact := range p.Programs {
		if artifact.ID == p.Runs[0].ProgramID {
			if err := json.Unmarshal(artifact.Bytes, &root); err != nil {
				t.Fatal(err)
			}
		}
	}
	leafID := root.Author.Invocations["first"].TemplateID
	path := filepath.Join(t.TempDir(), "handles.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	if state.Projection.Status != "open" || state.Projection.Data["saved"].Text != state.Projection.Data["two"].Text || state.Projection.Data["status"].Text != "" || report.Roots[0].Outcome != nil {
		t.Fatal("unrelated result consumed", state, report)
	}
	snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tables["composition_results"]) != 2 {
		t.Fatal("did not reach both recorded outcomes")
	}
	expectedOperations := map[string]bool{state.Projection.Data["one"].Text: false, state.Projection.Data["two"].Text: false}
	for _, row := range snapshot.Tables["composition_results"] {
		var outcome runtime.InvocationOutcome
		if err = json.Unmarshal([]byte(row["descriptor"].(string)), &outcome); err != nil {
			t.Fatal(err)
		}
		if _, ok := expectedOperations[outcome.OperationID]; !ok || outcome.Caller != state.ID || outcome.TemplateID != leafID || outcome.Status != "succeeded" || !outcome.ResponsePresent {
			t.Fatal("unrelated or missing original outcome", outcome)
		}
		expectedOperations[outcome.OperationID] = true
	}
	t.Logf("retained first=%s second=%s serialized_selected=%s pending_pc=%d", state.Projection.Data["one"].Text, state.Projection.Data["two"].Text, state.Projection.Data["saved"].Text, state.Projection.PC)
	for operation, seen := range expectedOperations {
		if !seen {
			t.Fatal("missing original operation result", operation)
		}
	}
	if _, atReceive := root.Author.Receives[state.Projection.PC]; !atReceive {
		t.Fatal("did not reach actual Await receive", state.Projection.PC)
	}

	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
