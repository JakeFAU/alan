//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/research/nominal"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Both roots execute these same acyclic bytes. Freeze the two comparison edges
// and reverse consumption order independently of actual retained records.
func TestNominalAliasFreshExecution(t *testing.T) {
	port := author.Port{Peer: "caller", Obligation: "work", Receive: schema.Record{"value": {Kind: "int"}}, Send: schema.Record{"value": {Kind: "int"}}}
	helperProgram, err := author.Compile(author.Definition{Name: "nominal-echo", Ports: map[string]author.Port{"public": port}, Steps: []author.Step{
		author.Receive("public", map[string]string{"value": "received"}),
		author.Send("public", map[string]author.Expr{"value": author.Var("received")}), author.Success(),
	}})
	must(t, err)
	helper, err := author.NewTemplate(author.Application{"entry": {Artifact: helperProgram, Inputs: map[string]schema.Value{}}}, "entry", "public")
	must(t, err)
	_, err = nominal.CheckTemplate(helper)
	must(t, err)
	commit := func(dst string, value int64) author.Step {
		return author.Invoke(dst, "echo", map[string]author.Expr{"value": author.Integer(value)}, nil)
	}
	marker := func(slot, s string) []author.Step { return []author.Step{author.Set(slot, author.String(s))} }
	consume := func(h, dst string) author.Step {
		return author.AwaitInvocation("echo", h, author.InvocationOutcomeBindings{
			Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"value": dst},
		})
	}
	parent, err := author.Compile(author.Definition{Name: "nominal-alias-and-fresh", Invocations: map[string]author.Template{"echo": helper}, Steps: []author.Step{
		commit("first", 7), author.Set("saved", author.Var("first")), commit("second", 11),
		author.IfEqual(author.Var("first"), author.Var("saved"), marker("alias", "equal"), marker("alias", "unequal")),
		author.IfEqual(author.Var("first"), author.Var("second"), marker("fresh", "equal"), marker("fresh", "distinct")),
		consume("second", "new_value"), consume("saved", "old_value"), author.Success(),
	}})
	must(t, err)
	cert, err := nominal.CheckArtifact(parent)
	must(t, err)
	var program struct {
		Code []struct {
			Op                string
			Target, Otherwise int
		}
	}
	must(t, json.Unmarshal(parent.Bytes(), &program))
	// One admission, reply and outcome per child; each PC visits at most once.
	stepBound := len(program.Code) + 2*len(helperProgram.Metadata().Positions)
	for _, root := range []string{"nominalA", "longerNominalB"} {
		t.Run(root, func(t *testing.T) {
			runID := runtime.RunID{Subject: root, Generation: 1}
			p, err := (author.Application{root: {Artifact: parent, Inputs: map[string]schema.Value{}}}).Provision()
			must(t, err)
			p.Composition.Roots = []runtime.ResourceRoot{{Run: runID, Members: []runtime.RunID{runID}, Templates: []string{helper.ID()}, Limits: limits(2, 2, 1)}}
			path := filepath.Join(t.TempDir(), "control.sqlite")
			store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			must(t, err)
			report, err := store.Run(t.Context(), runtime.RunOptions{MaxSteps: stepBound + 1, MaxDispatch: 7, MaxInFlight: 1})
			must(t, err)
			must(t, store.Close())
			s := inspect(t, path)
			if !report.Quiescent || len(report.OpenRuns) != 0 || report.RejectedAdmissions != 0 || report.Steps > stepBound || report.Dispatches != 6 {
				t.Fatalf("watchdog or service bound: %+v", report)
			}
			retainedPrograms := map[string]string{}
			for _, row := range s.Tables["programs"] {
				retainedPrograms[row["id"].(string)] = row["bytes"].(string)
			}
			if retainedPrograms[parent.ID()] != string(parent.Bytes()) || retainedPrograms[helperProgram.ID()] != string(helperProgram.Bytes()) {
				t.Fatal("retained programs differ from the checked artifacts")
			}
			state := projection(s, root)
			if state.Status != "succeeded" || state.Data["alias"] != schema.Text("equal") || state.Data["fresh"] != schema.Text("distinct") ||
				state.Data["new_value"] != schema.Int(11) || state.Data["old_value"] != schema.Int(7) {
				t.Fatal("terminal published values/markers", state)
			}
			first := fmt.Sprintf(`["%s",1,1,0]`, root)
			second := fmt.Sprintf(`["%s",1,3,0]`, root)
			if state.Data["first"] != schema.Text(first) || state.Data["saved"] != schema.Text(first) || state.Data["second"] != schema.Text(second) {
				t.Fatal("original commitments", state)
			}
			// Look up unmodified runtime rows before attributing an observed branch
			// or selection. Branch targets are read from this very artifact's bytes.
			results := map[string]runtime.InvocationOutcome{}
			for _, row := range s.Tables["composition_publications"] {
				o := decode[runtime.InvocationOutcome](row, "descriptor")
				results[o.OperationID] = o
			}
			observations := map[string]observation{}
			for _, row := range s.Tables["observations"] {
				observations[row["id"].(string)] = decode[observation](row, "descriptor")
			}
			delivered := map[string]string{}
			for _, row := range s.Tables["composition_outcome_deliveries"] {
				delivered[row["observation_id"].(string)] = row["operation_id"].(string)
			}
			var commits, choices, selections []string
			var pc int
			for _, row := range s.Tables["history"] {
				if row["subject"] != root || row["generation"] != runID.Generation {
					continue
				}
				next := decode[runtime.Projection](row, "successor")
				in := program.Code[pc]
				for _, it := range decode[[]intent](row, "intents") {
					if it.Run == runID && it.Kind == "invocation" && it.Target == helper.ID() && it.Obligation == "echo" && it.Sequence == row["sequence"] {
						commits = append(commits, it.ID)
					}
				}
				if in.Op == "branch" {
					choices = append(choices, fmt.Sprintf("%d->%d", pc, next.PC))
					want := in.Target
					if len(choices) == 2 {
						want = in.Otherwise
					}
					if next.PC != want {
						t.Fatalf("branch %d went to %d, want %d", pc, next.PC, want)
					}
				}
				if oid, ok := row["selected_observation"].(string); ok && oid != "" {
					index := len(selections)
					if index > 1 {
						t.Fatal("extra selection", oid)
					}
					operation := []string{second, first}[index]
					value := []int64{11, 7}[index]
					result := results[operation]
					obs := observations[oid]
					if in.Op != "receive" || delivered[oid] != operation || obs.Operation != operation || obs.Provenance != "runtime-invocation" ||
						obs.Kind != "invocation" || obs.ID != oid || obs.Run != runID || obs.Source != helper.ID() || obs.SourceGeneration != 1 || obs.Obligation != "echo" ||
						obs.Payload["response.value"] != schema.Int(value) || obs.Payload["response_present"] != schema.Bool(true) ||
						result.OperationID != operation || result.Caller != runID || result.TemplateID != helper.ID() || result.Status != "succeeded" ||
						!result.ResponsePresent || result.Response["value"] != schema.Int(value) {
						t.Fatalf("selection %d not the predicted original outcome: %s %+v %+v", index, oid, obs, result)
					}
					selections = append(selections, operation)
				}
				pc = next.PC
			}
			if !reflect.DeepEqual(commits, []string{first, second}) || len(choices) != 2 || !reflect.DeepEqual(selections, []string{second, first}) {
				t.Fatalf("actual commits=%v branches=%v selections=%v", commits, choices, selections)
			}
			t.Logf("parent=%s helper=%s cert=%s branches=%v selected=%v payloads=[11 7] bound=%d steps=%d dispatches=%d", parent.ID(), helper.ID(), cert.Digest, choices, selections, stepBound, report.Steps, report.Dispatches)
			retain(t, "alias-fresh", map[string]any{"inspection": s, "parent": parent.ID(), "helper": helper.ID(), "certificate": cert, "report": report})
		})
	}
}

func TestNominalWrongAnswerCertifies(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, path := provision(t, "weak-a", "bare", q, limits(3, 3, 3))
	cert, err := nominal.CheckTemplate(d.Candidate)
	must(t, err)
	create(t, d, path)
	run(t, path)
	s := inspect(t, path)
	n := normalize(s, d, q)
	requireEvidence(t, n)
	var badAnswer, mismatch bool
	for _, e := range n.Trace {
		if e.Role == "candidate" && e.Kind == "reply_admit" && e.Present && e.ID == 1 && e.Value == 0 {
			badAnswer = true
		}
	}
	for _, v := range n.Violations {
		if v.Code == "complement" {
			mismatch = true
		}
	}
	if !badAnswer || !mismatch {
		t.Fatalf("no admitted wrong literal or no K1 mismatch: %+v", n)
	}
	t.Logf("weak-a certificate=%s observed=(1,0) K1_required=(1,1) violations=%+v", cert.Digest, n.Violations)
}
