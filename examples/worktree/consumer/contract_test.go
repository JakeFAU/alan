package consumer_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/compose"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Literal expectations are frozen before the implementation; no compiler,
// reducer, replay or collection helper calculates expected item correspondence.
var ids = []string{"item-a", "item-b", "item-c", "item-d", "item-e", "item-f", "item-g", "item-h", "item-i"}
var inputs = []int64{2, -3, 2, 0, 11, -1, 9, 4, 100}
var outputs = []int64{4, -6, 4, 0, 22, -2, 18, 8}

func work(n int) schema.Value {
	rows := make([]map[string]schema.Value, n)
	for i := range rows {
		rows[i] = map[string]schema.Value{"item_id": schema.Text(ids[i]), "number": schema.Int(inputs[i])}
	}
	return schema.List(rows...)
}

// This fails if the component serializes/rebinds handles, coalesces equal
// payloads, changes retained topology, accepts invalid work or loses ordering.
func TestRetainedCollectionContract(t *testing.T) {
	leaf, err := worktree.Deterministic()
	if err != nil {
		t.Fatal(err)
	}
	collection, err := compose.New(leaf, compose.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	in, out, err := collection.Schemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, nested := range []bool{false, true} {
		deployment, err := worktree.New(leaf, nested, worktree.DefaultLimits(2))
		if err != nil {
			t.Fatal(err)
		}
		if deployment.Collection.ID() != collection.ID() {
			t.Fatal("different collection template")
		}
		di, do, err := deployment.Collection.Schemas()
		if err != nil || di.ID != in.ID || do.ID != out.ID {
			t.Fatal("schema drift", err)
		}
		before := deployment.Collection.Bytes()
		t.Logf("identity collection=%s input=%s output=%s catalogue_templates=%d artifact_count=%d initial_runs=%d nested=%v collection_bundle_bytes=%d", collection.ID(), in.ID, out.ID, len(deployment.Provision.Composition.Templates), len(deployment.Provision.Programs), len(deployment.Provision.Runs), nested, len(collection.Bytes()))
		for _, n := range []int{0, 1, 3, 7, 8, 9} {
			t.Run(fmt.Sprintf("nested=%v/n=%d", nested, n), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "work.sqlite")
				s, err := runtime.Create(t.Context(), path, deployment.Provision, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				initial, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				if len(initial.Tables["runs"]) != 1 {
					t.Fatal("work expanded at initialization")
				}
				admitted, err := worktree.Submit(t.Context(), s, "work", work(n))
				if err != nil {
					t.Fatal(err)
				}
				if n == 9 {
					if admitted.Accepted {
						t.Fatal("ninth item accepted")
					}
					rejected, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
					if err != nil {
						t.Fatal(err)
					}
					if len(rejected.Tables["composition_invocations"]) != 0 {
						t.Fatal("children before input")
					}
					return
				}
				if !admitted.Accepted {
					t.Fatal(admitted)
				}
				started := time.Now()
				report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 2000, MaxDispatch: 500, MaxInFlight: 2})
				if err != nil {
					t.Fatal(err)
				}
				if len(report.Roots) != 1 || report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != "succeeded" {
					t.Fatalf("root incomplete: %+v", report)
				}
				state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
				if err != nil {
					t.Fatal(err)
				}
				rows, err := state.Projection.Data["results"].Records()
				if err != nil || len(rows) != n {
					t.Fatal(rows, err)
				}
				if err := verifyRows(rows, n); err != nil {
					t.Fatal(err)
				}
				enclosing := int64(1)
				if nested {
					enclosing = 2
				}
				if report.Roots[0].Charges.Invocations != int64(n)+enclosing || report.Roots[0].Charges.CreatedRuns != int64(n)+enclosing || report.Roots[0].Charges.Effects != 0 {
					t.Fatal("shared charge mismatch", report.Roots)
				}
				final, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				if len(final.Tables["programs"]) != len(initial.Tables["programs"]) || len(final.Tables["composition_invocations"]) != n+int(enclosing) {
					t.Fatal("topology or invocation count mismatch")
				}
				t.Logf("n=%d enclosing_calls=%d item_calls=%d created_runs=%d effects=%d results=%d intents=%d history=%d latency=%s", n, enclosing, n, report.Roots[0].Charges.CreatedRuns, report.Roots[0].Charges.Effects, len(final.Tables["composition_results"]), len(final.Tables["intents"]), len(final.Tables["history"]), time.Since(started))
				if !bytes.Equal(before, deployment.Collection.Bytes()) {
					t.Fatal("artifact changed after input")
				}
				if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestRejectDuplicateIDsBeforeAnyChild(t *testing.T) {
	leaf, err := worktree.Deterministic()
	if err != nil {
		t.Fatal(err)
	}
	d, err := worktree.New(leaf, false, worktree.DefaultLimits(1))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "work.sqlite")
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v := schema.List(map[string]schema.Value{"item_id": schema.Text("item-a"), "number": schema.Int(2)}, map[string]schema.Value{"item_id": schema.Text("item-a"), "number": schema.Int(-3)})
	a, err := worktree.Submit(t.Context(), s, "duplicate", v)
	if err != nil {
		t.Fatal(err)
	}
	if a.Accepted {
		t.Fatal("duplicate IDs accepted")
	}
	snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tables["composition_invocations"]) != 0 {
		t.Fatal("rejected work created children")
	}
}

func TestLowerBudgetPreservesAcceptedWorkAndOrderedRejections(t *testing.T) {
	for _, field := range []string{"invocations", "runs", "depth"} {
		t.Run(field, func(t *testing.T) {
			leaf, err := worktree.Deterministic()
			if err != nil {
				t.Fatal(err)
			}
			limits := worktree.DefaultLimits(1)
			switch field {
			case "invocations":
				limits.MaxInvocations = 2
			case "runs":
				limits.MaxCreatedRuns = 2
			case "depth":
				limits.MaxDepth = 1
			}
			d, err := worktree.New(leaf, false, limits)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "work.sqlite")
			s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			a, err := worktree.Submit(t.Context(), s, "work", work(3))
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
			report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 2000, MaxDispatch: 500, MaxInFlight: 1})
			if err != nil {
				t.Fatal(err)
			}
			if report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != "failed" {
				t.Fatal("budget rejection reported success", report)
			}
			state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			rows, err := state.Projection.Data["results"].Records()
			if err != nil || len(rows) != 3 {
				t.Fatal("lost accepted work", rows, err)
			}
			for i, row := range rows {
				if row["item_id"].Text != ids[i] {
					t.Fatal("lost ordering", row)
				}
				want := "rejected"
				if i == 0 && field != "depth" {
					want = "succeeded"
				}
				if row["status"].Text != want {
					t.Fatal(field, row)
				}
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublicSchemaCopiesAndReservedField(t *testing.T) {
	if _, _, err := (author.Template{}).Schemas(); err == nil {
		t.Fatal("zero template schemas")
	}
	leaf, err := worktree.Deterministic()
	if err != nil {
		t.Fatal(err)
	}
	in, out, err := leaf.Schemas()
	if err != nil {
		t.Fatal(err)
	}
	in.Record["number"] = schema.Field{Kind: "bool"}
	out.Record["doubled"] = schema.Field{Kind: "bool"}
	again, result, err := leaf.Schemas()
	if err != nil || again.Record["number"].Kind != "int" || result.Record["doubled"].Kind != "int" {
		t.Fatal("mutable retained schemas", err)
	}
	a, err := author.Compile(author.Definition{Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: schema.Record{"item_id": {Kind: "string", MaxBytes: 20}}}}, Steps: []author.Step{author.Receive("public", map[string]string{"item_id": "id"}), author.Send("public", nil), author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	conflict, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = compose.New(conflict, compose.DefaultOptions()); err == nil {
		t.Fatal("reserved item_id conflict accepted")
	}
}

func verifyRows(rows []map[string]schema.Value, n int) error {
	if len(rows) != n {
		return fmt.Errorf("result count=%d want %d", len(rows), n)
	}
	operations := map[string]bool{}
	for i, row := range rows {
		response, err := row["responses"].Records()
		if err != nil || len(response) != 1 {
			return fmt.Errorf("item %d response missing", i)
		}
		operation := row["operation"].Text
		if row["item_id"].Text != ids[i] || row["status"].Text != "succeeded" || response[0]["doubled"].Number != outputs[i] || operation == "" || operations[operation] {
			return fmt.Errorf("incorrect item mapping at %d: got %s=%d want %s=%d", i, row["item_id"].Text, response[0]["doubled"].Number, ids[i], outputs[i])
		}
		operations[operation] = true
	}
	return nil
}

// Deliberately faulty ordinary collection: every authenticated leaf receives 0.
// Runtime correlation and replay remain valid; the SAME oracle used above must
// reject the public application mapping rather than a runtime schema guard.
func TestFaultyOrdinaryCollectorReachesIndependentMappingMonitor(t *testing.T) {
	leaf, err := worktree.Deterministic()
	if err != nil {
		t.Fatal(err)
	}
	collection, err := compose.New(leaf, compose.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	input, output, err := collection.Schemas()
	if err != nil {
		t.Fatal(err)
	}
	resultField := output.Record["results"]
	responseField := resultField.Element["responses"]
	handles := schema.Field{Kind: "list", MaxItems: 8, MaxBytes: 262144, Element: schema.Record{"item_id": {Kind: "string", MaxBytes: 128}, "operation": {Kind: "string", MaxBytes: 4096}}}
	faulty, err := author.Compile(author.Definition{Name: "faulty ordinary mapping", Invocations: map[string]author.Template{"item": leaf}, Collections: map[string]schema.Field{"handles": handles, "results": resultField, "responses": responseField}, Ports: map[string]author.Port{"intake": {Local: true, Peer: "local", Obligation: "work", Receive: input.Record}}, Steps: []author.Step{
		author.Receive("intake", map[string]string{"items": "items"}),
		author.ForEach(author.Var("items"), map[string]string{"item_id": "item_id", "number": "number"}, author.Invoke("pending", "item", map[string]author.Expr{"number": author.Integer(0)}, nil), author.Append("handles", map[string]author.Expr{"item_id": author.Var("item_id"), "operation": author.Var("pending")})),
		author.ForEach(author.Var("handles"), map[string]string{"item_id": "item_id", "operation": "saved"}, author.AwaitInvocation("item", "saved", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"doubled": "doubled"}}), author.Set("responses", author.Value(schema.List())), author.IfEqual(author.Var("present"), author.Boolean(true), []author.Step{author.Append("responses", map[string]author.Expr{"doubled": author.Var("doubled")})}, nil), author.Append("results", map[string]author.Expr{"item_id": author.Var("item_id"), "operation": author.Var("saved"), "status": author.Var("status"), "reason": author.Var("reason"), "responses": author.Var("responses")})), author.Success(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"root": {Artifact: faulty}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	p.Composition.Roots = []runtime.ResourceRoot{{Run: runtime.RunID{Subject: "root", Generation: 1}, Members: []runtime.RunID{{Subject: "root", Generation: 1}}, Templates: []string{leaf.ID()}, Limits: worktree.DefaultLimits(1)}}
	path := filepath.Join(t.TempDir(), "faulty.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	admitted, err := worktree.Submit(t.Context(), s, "faulty", work(2))
	if err != nil || !admitted.Accepted {
		t.Fatal(admitted, err)
	}
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 500, MaxDispatch: 200, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	if report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != "succeeded" || report.Roots[0].Charges.Invocations != 2 {
		t.Fatal("fault did not reach mapping boundary", report)
	}
	state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := state.Projection.Data["results"].Records()
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyRows(rows, 2); err == nil {
		t.Fatal("independent mapping monitor missed faulty ordinary code")
	}
	t.Log("unchanged mapping monitor detected:", err)
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal("runtime identity must remain valid for this monitor", err)
	}
}
