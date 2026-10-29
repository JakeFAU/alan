package a_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	candidate "github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestCandidateARequests(t *testing.T) {
	for id := int64(1); id <= 2; id++ {
		for value := int64(0); value <= 1; value++ {
			for _, mode := range []string{"answer", "fail", "absent", "pending"} {
				t.Run(fmt.Sprintf("id%d-value%d-%s", id, value, mode), func(t *testing.T) {
					qualify(t, id, value, mode)
				})
			}
		}
	}
}

func qualify(t *testing.T, id, value int64, mode string) {
	t.Helper()
	template, err := candidate.New()
	if err != nil {
		t.Fatal(err)
	}
	caller, err := author.Compile(author.Definition{
		Name:        "candidate-a-qualification-caller",
		Invocations: map[string]author.Template{"candidate": template},
		Steps: []author.Step{
			author.Invoke("handle", "candidate", map[string]author.Expr{
				"id": author.Integer(id), "value": author.Integer(value), "mode": author.String(mode),
			}, nil),
			author.AwaitInvocation("candidate", "handle", author.InvocationOutcomeBindings{
				Status: "status", Reason: "reason", ResponsePresent: "present",
				Response: map[string]string{"id": "response_id", "value": "response_value"},
			}),
			author.IfEqual(author.Var("status"), author.String("succeeded"),
				[]author.Step{author.IfEqual(author.Var("present"), author.Boolean(true),
					[]author.Step{author.Success()}, []author.Step{author.Failure()})},
				[]author.Step{author.Failure()}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	provision, err := (author.Application{
		"root": {Artifact: caller, Inputs: map[string]schema.Value{}},
	}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	rootID := runtime.RunID{Subject: "root", Generation: 1}
	provision.Composition.Roots = []runtime.ResourceRoot{{
		Run: rootID, Members: []runtime.RunID{rootID}, Templates: []string{template.ID()},
		Limits: runtime.ResourceLimits{
			MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 3, MaxEffects: 0, MaxInFlight: 1,
		},
	}}
	path := filepath.Join(t.TempDir(), "candidate.sqlite")
	store, err := runtime.Create(t.Context(), path, provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	// The finite artifact has no loop. 128 exceeds its instruction/dispatch
	// path bound; reaching it is a failure, never pending-mode evidence.
	report, err := store.Run(t.Context(), runtime.RunOptions{MaxSteps: 128, MaxDispatch: 128, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || report.Steps >= 128 || report.Dispatches >= 128 {
		t.Fatalf("cutoff or nonquiescence: %+v", report)
	}
	inspection, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Snapshot(t.Context(), rootID)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Roots) != 1 || len(replay.Roots) != 1 {
		t.Fatalf("root count: inspect=%d replay=%d", len(inspection.Roots), len(replay.Roots))
	}
	charge := inspection.Roots[0].Charges
	if charge.Invocations != 1 || charge.CreatedRuns != 1 || charge.Effects != 0 {
		t.Fatalf("charges: %+v", charge)
	}
	if len(inspection.Tables["composition_admission_decisions"]) != 1 ||
		len(inspection.Tables["composition_members"]) != 1 || len(inspection.Tables["runs"]) != 2 {
		t.Fatal("local candidate created unexpected subtree work")
	}
	member := inspection.Tables["composition_members"][0]["subject"].(string)
	child, err := store.Snapshot(t.Context(), runtime.RunID{Subject: member, Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	if child.Projection.Data["request_id"].Number != id || child.Projection.Data["request_value"].Number != value {
		t.Fatalf("request not retained/consumed: %+v", child.Projection)
	}
	publications := inspection.Tables["composition_publications"]
	if mode == "pending" {
		if state.Projection.Status != "open" || child.Projection.Status != "open" || len(publications) != 0 ||
			len(inspection.Tables["composition_replies"]) != 0 || inspection.Roots[0].Outcome != nil {
			t.Fatal("pending published, replied, or terminated")
		}
		if _, exists := state.Projection.Data["present"]; exists {
			t.Fatal("pending caller consumed an outcome")
		}
	} else {
		wantLocal, wantStatus, wantReason := "succeeded", "succeeded", "member_succeeded"
		if mode == "fail" {
			wantLocal, wantStatus, wantReason = "failed", "failed", "member_failed"
		}
		if mode == "absent" {
			wantStatus, wantReason = "failed", "missing_reply"
		}
		if child.Projection.Status != wantLocal || len(publications) != 1 {
			t.Fatalf("local=%s want=%s publications=%d", child.Projection.Status, wantLocal, len(publications))
		}
		var outcome runtime.InvocationOutcome
		decodeDescriptor(t, publications[0], &outcome)
		present := mode == "answer"
		if outcome.Status != wantStatus || outcome.Reason != wantReason || outcome.ResponsePresent != present {
			t.Fatalf("child envelope: %+v", outcome)
		}
		data := state.Projection.Data
		if data["status"].Text != wantStatus || data["reason"].Text != wantReason || data["present"].Flag != present {
			t.Fatalf("consumed outcome: %+v", data)
		}
		wantCaller := "failed"
		if present {
			wantCaller = "succeeded"
			if outcome.Response["id"].Number != id || outcome.Response["value"].Number != 1-value ||
				data["response_id"].Number != id || data["response_value"].Number != 1-value {
				t.Fatalf("wrong response: %+v %+v", outcome, data)
			}
			if len(inspection.Tables["composition_replies"]) != 1 {
				t.Fatal("answer needs exactly one reply")
			}
		} else if len(outcome.Response) != 0 || len(inspection.Tables["composition_replies"]) != 0 {
			t.Fatal("non-answer gained response presence")
		}
		if state.Projection.Status != wantCaller || inspection.Roots[0].Outcome == nil ||
			inspection.Roots[0].Outcome.Status != wantCaller {
			t.Fatalf("caller/root completion: %+v %+v", state, inspection.Roots[0])
		}
	}
	t.Logf("local=%s caller=%s steps=%d dispatch=%d charges=(1,1,0) replay-history=%d",
		child.Projection.Status, state.Projection.Status, report.Steps, report.Dispatches, replay.HistoryCount)
	if dir := os.Getenv("ALAN_PHASE1I_A_OUTPUT"); dir != "" {
		writeEvidence(t, dir, fmt.Sprintf("id%d-value%d-%s.json", id, value, mode), struct {
			ID, Value     int64
			Mode          string
			Report        runtime.RunReport
			Caller, Child runtime.RunState
			Inspection    runtime.Inspection
			Replay        runtime.ReplayReport
		}{ID: id, Value: value, Mode: mode, Report: report, Caller: state, Child: child, Inspection: inspection, Replay: replay})
	}
}

func decodeDescriptor(t *testing.T, row map[string]any, dst any) {
	t.Helper()
	raw, ok := row["descriptor"].(string)
	if !ok {
		t.Fatal("missing JSON descriptor")
	}
	if err := json.Unmarshal([]byte(raw), dst); err != nil {
		t.Fatal(err)
	}
}

func writeEvidence(t *testing.T, dir, name string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateAExport(t *testing.T) {
	dir := os.Getenv("ALAN_PHASE1I_A_OUTPUT")
	if dir == "" {
		t.Skip("set ALAN_PHASE1I_A_OUTPUT to export first retained bytes")
	}
	template, err := candidate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template.json"), template.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template-id.txt"), []byte(template.ID()+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var bundle struct{ Programs []runtime.Artifact }
	if err := json.Unmarshal(template.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if len(bundle.Programs) != 1 {
		t.Fatalf("local candidate programs=%d want=1", len(bundle.Programs))
	}
	for _, program := range bundle.Programs {
		if err := os.WriteFile(filepath.Join(dir, program.ID+".program.json"), program.Bytes, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
