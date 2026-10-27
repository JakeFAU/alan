package compose_test

import (
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/compose"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestCollectionEmptyAndAbsentConstrainedResponses(t *testing.T) {
	for _, empty := range []bool{true, false} {
		t.Run(map[bool]string{true: "empty-record", false: "optional-enum"}[empty], func(t *testing.T) {
			output := schema.Record{}
			response := map[string]author.Expr{}
			if !empty {
				output["disposition"] = schema.Field{Kind: "string", MaxBytes: 8, Enum: []string{"ok"}}
				response["disposition"] = author.String("ok")
			}
			a, err := author.Compile(author.Definition{Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: schema.Record{"number": {Kind: "int"}}, Send: output}}, Steps: []author.Step{
				author.Receive("public", map[string]string{"number": "number"}), author.IfEqual(author.Var("number"), author.Integer(0), []author.Step{author.Send("public", response), author.Success()}, []author.Step{author.Failure()}),
			}})
			if err != nil {
				t.Fatal(err)
			}
			leaf, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
			if err != nil {
				t.Fatal(err)
			}
			d, err := worktree.New(leaf, false, worktree.DefaultLimits(1))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "optional.sqlite")
			s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			items := schema.List(map[string]schema.Value{"item_id": schema.Text("present"), "number": schema.Int(0)}, map[string]schema.Value{"item_id": schema.Text("absent"), "number": schema.Int(1)})
			admitted, err := worktree.Submit(t.Context(), s, "optional", items)
			if err != nil || !admitted.Accepted {
				t.Fatal(admitted, err)
			}
			report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 500, MaxDispatch: 200, MaxInFlight: 1})
			if err != nil {
				t.Fatal(err)
			}
			state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			rows, err := state.Projection.Data["results"].Records()
			if err != nil || len(rows) != 2 {
				t.Fatal(rows, err)
			}
			present, err := rows[0]["responses"].Records()
			if err != nil || len(present) != 1 {
				t.Fatal(rows, err)
			}
			absent, err := rows[1]["responses"].Records()
			if err != nil || len(absent) != 0 || rows[1]["status"].Text != "failed" || rows[1]["reason"].Text == "" {
				t.Fatal("absent response fabricated", rows, err)
			}
			if empty && len(present[0]) != 0 {
				t.Fatal("empty response lost", rows)
			}
			if !empty && present[0]["disposition"].Text != "ok" {
				t.Fatal(rows)
			}
			if report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != "failed" || state.Projection.Data["all_succeeded"].Flag {
				t.Fatal("failed aggregate succeeded", report)
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestCollectionRejectsInvalidBounds(t *testing.T) {
	leaf, err := worktree.Deterministic()
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range []compose.Options{{}, {MaxItems: 9, MaxBytes: -1, MaxIDBytes: 1}, {MaxItems: 1, MaxBytes: 100, MaxIDBytes: 0}, {MaxItems: schema.MaxCollectionItems + 1, MaxBytes: 8192, MaxIDBytes: 10}} {
		if _, err = compose.New(leaf, options); err == nil {
			t.Fatal("invalid bounds accepted", options)
		}
	}
}

func TestDeterministicOverflowIsExplicitAndDoesNotInventResult(t *testing.T) {
	leaf, err := worktree.Deterministic()
	if err != nil {
		t.Fatal(err)
	}
	d, err := worktree.New(leaf, false, worktree.DefaultLimits(1))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "overflow.sqlite")
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	admitted, err := worktree.Submit(t.Context(), s, "overflow", schema.List(map[string]schema.Value{"item_id": schema.Text("overflow"), "number": schema.Int(9223372036854775807)}))
	if err != nil || !admitted.Accepted {
		t.Fatal(admitted, err)
	}
	if _, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err == nil {
		t.Fatal("integer overflow silently succeeded")
	}
	snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Roots[0].Outcome != nil || len(snapshot.Tables["composition_results"]) != 0 {
		t.Fatal("invented overflow result", snapshot.Roots)
	}
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
