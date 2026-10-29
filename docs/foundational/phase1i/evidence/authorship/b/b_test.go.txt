//go:build integration

package b_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	candidate "github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// These cases catch missing replies, changed application identity/value, confused
// local/invocation outcomes, skipped helper work, and extra subtree admissions.
func TestNewContractRequests(t *testing.T) {
	for _, id := range []int64{1, 2} {
		for _, value := range []int64{0, 1} {
			for _, mode := range []string{"answer", "fail", "absent", "pending"} {
				name := fmt.Sprintf("id%d-value%d-%s", id, value, mode)
				t.Run(name, func(t *testing.T) {
					qualify(t, scenario{name: name, id: id, value: value, mode: mode,
						limits: runtime.ResourceLimits{
							MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 2, MaxInFlight: 1,
						}})
				})
			}
		}
	}
}

// Helper refusal must override every requested mode, including pending/absent.
func TestNewHelperRefusal(t *testing.T) {
	for _, denial := range []struct {
		name, reason string
		limits       runtime.ResourceLimits
	}{
		{name: "invocations", reason: "invocation_limit", limits: runtime.ResourceLimits{
			MaxInvocations: 1, MaxCreatedRuns: 2, MaxDepth: 2, MaxInFlight: 1,
		}},
		{name: "runs", reason: "created_run_limit", limits: runtime.ResourceLimits{
			MaxInvocations: 2, MaxCreatedRuns: 1, MaxDepth: 2, MaxInFlight: 1,
		}},
		{name: "depth", reason: "depth_limit", limits: runtime.ResourceLimits{
			MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 1, MaxInFlight: 1,
		}},
	} {
		for _, mode := range []string{"answer", "fail", "absent", "pending"} {
			name := denial.name + "-" + mode
			t.Run(name, func(t *testing.T) {
				qualify(t, scenario{name: name, id: 1, value: 0, mode: mode,
					limits: denial.limits, refusal: denial.reason})
			})
		}
	}
}

type scenario struct {
	name, mode, refusal string
	id, value           int64
	limits              runtime.ResourceLimits
}

func qualify(t *testing.T, sc scenario) {
	t.Helper()
	template, err := candidate.New()
	if err != nil {
		t.Fatal(err)
	}
	if template.ID() == "" {
		t.Fatal("New returned empty retained template; candidate B is not implemented")
	}
	input, output, err := template.Schemas()
	if err != nil {
		t.Fatal(err)
	}
	wantInput, _ := protocol.Request().Retain()
	wantOutput, _ := protocol.Response().Retain()
	if input.ID != wantInput.ID || output.ID != wantOutput.ID {
		t.Fatal("candidate changed the public protocol schemas")
	}
	caller, err := author.Compile(author.Definition{
		Name:        "phase1i-b-public-qualification",
		Invocations: map[string]author.Template{"candidate": template},
		Steps: []author.Step{
			author.Invoke("original", "candidate", map[string]author.Expr{
				"id": author.Integer(sc.id), "value": author.Integer(sc.value), "mode": author.String(sc.mode),
			}, nil),
			author.AwaitInvocation("candidate", "original", author.InvocationOutcomeBindings{
				Status: "status", Reason: "reason", ResponsePresent: "present",
				Response: map[string]string{"id": "id", "value": "value"},
			}),
			author.IfEqual(author.Var("status"), author.String("succeeded"), []author.Step{
				author.IfEqual(author.Var("present"), author.Boolean(true),
					[]author.Step{author.Success()}, []author.Step{author.Failure()}),
			}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"root": {Artifact: caller, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	allowed := make([]string, 0, len(p.Composition.Templates))
	for _, descriptor := range p.Composition.Templates {
		allowed = append(allowed, descriptor.ID)
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	p.Composition.Roots = []runtime.ResourceRoot{{
		Run: root, Members: []runtime.RunID{root}, Templates: allowed, Limits: sc.limits,
	}}
	path := filepath.Join(t.TempDir(), "candidate.sqlite")
	store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	// Acyclic candidate+helper+caller have fewer than 100 instructions total;
	// each protocol message/outcome has one admission and one delivery. These
	// caps exceed the finite graph's step/dispatch demands without adding work.
	report, err := store.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || report.Steps >= 200 || report.Dispatches >= 100 {
		t.Fatalf("incomplete bounded run: %+v", report)
	}
	inspection, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	admissions := []runtime.InvocationAdmission{}
	var entry, helper runtime.InvocationAdmission
	for _, row := range inspection.Tables["composition_admission_decisions"] {
		var admission runtime.InvocationAdmission
		decodeDescriptor(t, row, &admission)
		admissions = append(admissions, admission)
		if admission.Invocation.TemplateID == template.ID() {
			entry = admission
		} else {
			helper = admission
		}
	}
	if len(admissions) != 2 || !entry.Accepted || len(entry.Members) != 1 {
		t.Fatalf("want entry plus exactly one helper decision: %+v", admissions)
	}
	if helper.Invocation.ParentInvocationID != entry.Invocation.ID || helper.Invocation.Depth != 2 {
		t.Fatalf("helper parent/depth mismatch: %+v", helper)
	}
	if helper.Invocation.Inputs["id"] != schema.Int(sc.id) ||
		helper.Invocation.Inputs["value"] != schema.Int(sc.value) ||
		helper.Invocation.Inputs["mode"] != schema.Text("answer") {
		t.Fatalf("helper request must preserve payload with answer mode: %+v", helper.Invocation.Inputs)
	}
	wantCharge := int64(2)
	if sc.refusal != "" {
		wantCharge = 1
		if helper.Accepted || helper.Reason != sc.refusal || len(helper.Members) != 0 {
			t.Fatalf("helper refusal: %+v; want %s", helper, sc.refusal)
		}
	} else if !helper.Accepted || len(helper.Members) != 1 {
		t.Fatalf("helper not admitted: %+v", helper)
	}
	if len(inspection.Roots) != 1 {
		t.Fatalf("roots: %+v", inspection.Roots)
	}
	charge := inspection.Roots[0].Charges
	if charge.Invocations != wantCharge || charge.CreatedRuns != wantCharge || charge.Effects != 0 {
		t.Fatalf("charges: %+v; want (%d,%d,0)", charge, wantCharge, wantCharge)
	}
	if len(inspection.Tables["composition_charges"]) != int(wantCharge) ||
		len(replay.Runs) != int(1+wantCharge) || len(inspection.Tables["external_attempts"]) != 0 ||
		len(inspection.Tables["adapter_attempts"]) != 0 {
		t.Fatalf("unexpected extra work: charges=%d runs=%d", len(inspection.Tables["composition_charges"]), len(replay.Runs))
	}
	outcomes := map[string]runtime.InvocationOutcome{}
	for _, row := range inspection.Tables["composition_publications"] {
		var outcome runtime.InvocationOutcome
		decodeDescriptor(t, row, &outcome)
		outcomes[outcome.OperationID] = outcome
	}
	helperOutcome, ok := outcomes[helper.Invocation.OperationID]
	if !ok {
		t.Fatal("missing helper outcome")
	}
	if sc.refusal != "" {
		if helperOutcome.Status != "rejected" || helperOutcome.Reason != sc.refusal || helperOutcome.ResponsePresent {
			t.Fatalf("helper rejected envelope: %+v", helperOutcome)
		}
	} else {
		if helperOutcome.Status != "succeeded" || helperOutcome.Reason != "member_succeeded" || !helperOutcome.ResponsePresent {
			t.Fatalf("helper success envelope: %+v", helperOutcome)
		}
		checkResponse(t, helperOutcome.Response, sc.id, 1-sc.value)
		helperState := findRun(t, replay.Runs, helper.Members[0])
		if helperState.Projection.Status != "succeeded" {
			t.Fatalf("helper local status: %+v", helperState.Projection)
		}
	}
	entryState := findRun(t, replay.Runs, entry.Members[0])
	callerState := findRun(t, replay.Runs, root)
	entryOutcome, published := outcomes[entry.Invocation.OperationID]
	pending := sc.mode == "pending" && sc.refusal == ""
	if pending {
		if published || entryState.Projection.Status != "open" || callerState.Projection.Status != "open" {
			t.Fatalf("pending published or closed: entry=%+v caller=%+v outcome=%+v", entryState, callerState, entryOutcome)
		}
		if _, consumed := callerState.Projection.Data["present"]; consumed {
			t.Fatal("pending caller acquired a response-presence binding")
		}
	} else {
		wantLocal, wantStatus, wantReason, wantPresent := "failed", "failed", "member_failed", false
		if sc.refusal == "" {
			switch sc.mode {
			case "answer":
				wantLocal, wantStatus, wantReason, wantPresent = "succeeded", "succeeded", "member_succeeded", true
			case "absent":
				wantLocal, wantReason = "succeeded", "missing_reply"
			}
		}
		if entryState.Projection.Status != wantLocal || !published ||
			entryOutcome.Status != wantStatus || entryOutcome.Reason != wantReason || entryOutcome.ResponsePresent != wantPresent {
			t.Fatalf("candidate local/envelope mismatch: local=%s outcome=%+v", entryState.Projection.Status, entryOutcome)
		}
		data := callerState.Projection.Data
		if data["status"] != schema.Text(wantStatus) || data["reason"] != schema.Text(wantReason) || data["present"] != schema.Bool(wantPresent) {
			t.Fatalf("caller consumed wrong envelope: %+v", data)
		}
		wantCaller := "failed"
		if wantPresent {
			wantCaller = "succeeded"
			checkResponse(t, entryOutcome.Response, sc.id, 1-sc.value)
			checkResponse(t, map[string]schema.Value{"id": data["id"], "value": data["value"]}, sc.id, 1-sc.value)
		}
		if callerState.Projection.Status != wantCaller {
			t.Fatalf("caller local=%s; want %s", callerState.Projection.Status, wantCaller)
		}
	}
	wantReplies := 1
	if sc.mode == "answer" {
		wantReplies = 2
	}
	if sc.refusal != "" {
		wantReplies = 0
	}
	if len(inspection.Tables["composition_replies"]) != wantReplies {
		t.Fatalf("reply count=%d; want %d", len(inspection.Tables["composition_replies"]), wantReplies)
	}
	if dir := os.Getenv("ALAN_PHASE1I_B_EVIDENCE"); dir != "" {
		writeJSON(t, filepath.Join(dir, sc.name+".json"), struct {
			Scenario   map[string]any
			Report     runtime.RunReport
			Inspection runtime.Inspection
			Replay     runtime.ReplayReport
		}{Scenario: map[string]any{
			"name": sc.name, "id": sc.id, "value": sc.value, "mode": sc.mode,
			"limits": sc.limits, "refusal": sc.refusal,
		}, Report: report, Inspection: inspection, Replay: replay})
	}
	t.Logf("mode=%s refusal=%s candidate-local=%s charges=(%d,%d,%d) steps=%d dispatch=%d replay-runs=%d",
		sc.mode, sc.refusal, entryState.Projection.Status, charge.Invocations,
		charge.CreatedRuns, charge.Effects, report.Steps, report.Dispatches, len(replay.Runs))
}

func decodeDescriptor(t *testing.T, row map[string]any, target any) {
	t.Helper()
	raw, ok := row["descriptor"].(string)
	if !ok {
		t.Fatalf("missing JSON descriptor: %+v", row)
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		t.Fatal(err)
	}
}

func findRun(t *testing.T, runs []runtime.RunState, id runtime.RunID) runtime.RunState {
	t.Helper()
	for _, run := range runs {
		if run.ID == id {
			return run
		}
	}
	t.Fatalf("missing run %+v", id)
	return runtime.RunState{}
}

func checkResponse(t *testing.T, response map[string]schema.Value, id, value int64) {
	t.Helper()
	if len(response) != 2 || response["id"] != schema.Int(id) || response["value"] != schema.Int(value) {
		t.Fatalf("response=%+v; want id=%d value=%d", response, id, value)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFreezeFirstVersion is an explicit authoring-evidence utility, not a
// semantic test or oracle. It preserves the exact bytes returned by New.
func TestFreezeFirstVersion(t *testing.T) {
	dir := os.Getenv("ALAN_PHASE1I_B_FREEZE")
	if dir == "" {
		t.Skip("set ALAN_PHASE1I_B_FREEZE to retain first-version bytes")
	}
	template, err := candidate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template.json"), template.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template-id.txt"), []byte(template.ID()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var bundle struct {
		Programs []runtime.Artifact
	}
	if err := json.Unmarshal(template.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if len(bundle.Programs) != 2 {
		t.Fatalf("want candidate and helper programs, got %d", len(bundle.Programs))
	}
	for _, program := range bundle.Programs {
		path := filepath.Join(dir, "program-"+program.ID+".json")
		if err := os.WriteFile(path, program.Bytes, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("retained template=%s programs=%d bytes=%d", template.ID(), len(bundle.Programs), len(template.Bytes()))
}
