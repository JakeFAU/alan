package author_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Rebinding a serialized operation from a different declaration must not consume
// an authentic but unrelated result, even when both declarations share a template.
func TestSerializedInvocationHandleCannotRebindOutcome(t *testing.T) {
	leafArtifact, err := author.Compile(author.Definition{Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work"}}, Steps: []author.Step{author.Receive("public", nil), author.Send("public", nil), author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := author.NewTemplate(author.Application{"entry": {Artifact: leafArtifact, Inputs: map[string]schema.Value{}}}, "entry", "public")
	if err != nil {
		t.Fatal(err)
	}
	handles := schema.Field{Kind: "list", MaxItems: 1, MaxBytes: 8192, Element: schema.Record{"operation": {Kind: "string", MaxBytes: 4096}}}
	root, err := author.Compile(author.Definition{Invocations: map[string]author.Template{"first": leaf, "second": leaf}, Collections: map[string]schema.Field{"handles": handles}, Steps: []author.Step{
		author.Invoke("one", "first", nil, nil), author.Invoke("two", "second", nil, nil), author.Append("handles", map[string]author.Expr{"operation": author.Var("two")}),
		author.ForEach(author.Var("handles"), map[string]string{"operation": "saved"}, author.AwaitInvocation("first", "saved", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"})), author.Success(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"root": {Artifact: root}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	p.Composition.Roots = []runtime.ResourceRoot{{Run: runtime.RunID{Subject: "root", Generation: 1}, Members: []runtime.RunID{{Subject: "root", Generation: 1}}, Templates: []string{leaf.ID()}, Limits: runtime.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 1, MaxEffects: 0, MaxInFlight: 1}}}
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
		if _, ok := expectedOperations[outcome.OperationID]; !ok || outcome.Caller != state.ID || outcome.TemplateID != leaf.ID() || outcome.Status != "succeeded" || !outcome.ResponsePresent {
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
	if _, atReceive := root.Metadata().Receives[state.Projection.PC]; !atReceive {
		t.Fatal("did not reach actual Await receive", state.Projection.PC)
	}

	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
