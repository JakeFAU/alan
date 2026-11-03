package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This construction stands alone as a regression witness for the public API.
// The driver only provisions, runs, and reads; both joins live in the parent.
func TestPublicConstructionGate(t *testing.T) {
	ctx := t.Context()
	snippet := schema.Record{"snippet": lookup.OutputSchema()["snippet"]}
	worker, err := author.Compile(author.Definition{
		Ports:   map[string]author.Port{"work": {Peer: "caller", Obligation: "look-up", Receive: lookup.InputSchema(), Send: snippet}},
		Effects: map[string]author.Effect{"search": {Capability: "corpus", Input: lookup.InputSchema(), Output: lookup.OutputSchema()}},
		Steps: []author.Step{
			author.Receive("work", map[string]string{"query": "query"}),
			author.Request("lookup", "search", map[string]author.Expr{"query": author.Var("query")}, nil),
			author.Await("search", "lookup", map[string]string{"valid": "found", "snippet": "snippet"}),
			author.IfEqual(author.Var("found"), author.Boolean(true), []author.Step{
				author.Send("work", map[string]author.Expr{"snippet": author.Var("snippet")}), author.Success(),
			}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	template, err := author.NewTemplate(author.Application{"worker": {Artifact: worker, Inputs: map[string]schema.Value{}}}, "worker", "work")
	if err != nil {
		t.Fatal(err)
	}
	join := func(role string) author.Step {
		return author.AwaitInvocation("research", role, author.InvocationOutcomeBindings{
			Status: role + "_status", Reason: role + "_reason", ResponsePresent: role + "_present",
			Response: map[string]string{"snippet": role + "_snippet"},
		})
	}
	parent, err := author.Compile(author.Definition{
		Invocations: map[string]author.Template{"research": template},
		Steps: []author.Step{
			author.Invoke("one", "research", map[string]author.Expr{"query": author.String("alpha")}, nil),
			author.Invoke("two", "research", map[string]author.Expr{"query": author.String("beta")}, nil),
			join("one"), join("two"),
			author.IfEqual(author.Var("one_snippet"), author.String("alpha source"), nil, []author.Step{author.Failure()}),
			author.IfEqual(author.Var("two_snippet"), author.String("beta source"), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"parent": {Artifact: parent}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	asset, err := effects.NewAsset("source", "gate", []byte("alpha source\nbeta source"))
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = []effects.Asset{asset}
	root := runtime.RunID{Subject: "parent", Generation: 1}
	p.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: []string{template.ID()}, Capabilities: []string{"corpus"},
		Limits: runtime.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 1, MaxEffects: 2, MaxInFlight: 2}}}
	path := filepath.Join(t.TempDir(), "gate.sqlite")
	limits := runtime.DefaultLimits()
	store, err := runtime.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	report, err := store.Run(ctx, runtime.RunOptions{MaxSteps: 64, MaxDispatch: 64, MaxInFlight: 2, Adapters: map[string]effects.Adapter{"corpus": lookup.New()}})
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Snapshot(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	data := state.Projection.Data
	if len(report.OpenRuns) != 0 || state.Projection.Status != "succeeded" || data["one"].Text == data["two"].Text {
		t.Fatalf("public construction failed: parent %+v, report %+v", state, report)
	}
	inspection, err := runtime.Inspect(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Tables["runs"]) != 3 || len(inspection.Tables["adapter_attempts"]) != 2 || len(inspection.Tables["adapter_results"]) != 2 {
		t.Fatal("expected two independently retained workers with their own effect attempts and results")
	}
	if len(inspection.Tables["composition_results"]) != 2 {
		t.Fatal("expected two original outcomes")
	}
	original := map[string]string{data["one"].Text: "alpha source", data["two"].Text: "beta source"}
	for _, row := range inspection.Tables["composition_results"] {
		var outcome runtime.InvocationOutcome
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &outcome); err != nil {
			t.Fatal(err)
		}
		want, ok := original[outcome.OperationID]
		if !ok || outcome.Caller != root || outcome.Status != "succeeded" || !outcome.ResponsePresent || outcome.Response["snippet"].Text != want {
			t.Fatalf("unrelated outcome %+v", outcome)
		}
		delete(original, outcome.OperationID)
	}
	if len(original) != 0 {
		t.Fatal("missing original outcome")
	}
	t.Logf("original operations %s and %s, replies %q and %q; %d ordinary steps", data["one"].Text, data["two"].Text, data["one_snippet"].Text, data["two_snippet"].Text, report.Steps)
}
