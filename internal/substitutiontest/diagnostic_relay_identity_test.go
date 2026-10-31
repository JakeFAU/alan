//go:build integration

package substitutiontest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This is a source-level observation probe at one physical depth. It is neither
// an original A_op candidate nor a relay or a test of the D_N transport theorem.
// The literal and opposite branch predictions were frozen before execution;
// see the Phase 1Q qualification packet and its retained probe plan.
func TestDiagnosticRelayHandleObservation(t *testing.T) {
	helperArtifact, err := author.Compile(author.Definition{
		Name: "phase1q-handle-observation-helper",
		Ports: map[string]author.Port{"public": {
			Peer: "caller", Obligation: "work",
			Receive: schema.Record{"value": {Kind: "int"}},
			Send:    schema.Record{"value": {Kind: "int"}},
		}},
		Steps: []author.Step{
			author.Receive("public", map[string]string{"value": "input_value"}),
			author.Send("public", map[string]author.Expr{"value": author.Var("input_value")}),
			author.Success(),
		},
	})
	must(t, err)
	helper, err := author.NewTemplate(author.Application{
		"entry": {Artifact: helperArtifact, Inputs: map[string]schema.Value{}},
	}, "entry", "public")
	must(t, err)
	handles := schema.Field{Kind: "list", MaxItems: 1, MaxBytes: 8192,
		Element: schema.Record{"operation": {Kind: "string", MaxBytes: 4096}}}
	const frozenLiteral = `["probeA",1,2,0]`
	artifact, err := author.Compile(author.Definition{
		Name:        "phase1q-handle-observation-parent",
		Invocations: map[string]author.Template{"helper": helper},
		Collections: map[string]schema.Field{"handles": handles},
		Steps: []author.Step{
			author.Invoke("original", "helper", map[string]author.Expr{"value": author.Integer(7)}, nil),
			author.Append("handles", map[string]author.Expr{"operation": author.Var("original")}),
			author.ForEach(author.Var("handles"), map[string]string{"operation": "saved"},
				author.AwaitInvocation("helper", "saved", author.InvocationOutcomeBindings{
					Status: "child_status", Reason: "child_reason", ResponsePresent: "child_present",
					Response: map[string]string{"value": "child_value"},
				}),
				author.IfEqual(author.Var("saved"), author.String(frozenLiteral),
					[]author.Step{author.Set("branch", author.String("literal_equal"))},
					[]author.Step{author.Set("branch", author.String("literal_different"))})),
			author.Success(),
		},
	})
	must(t, err)
	originalBytes := artifact.Bytes()
	artifactHash := sha256.Sum256(originalBytes)
	var parentCode, helperCode struct{ Code []json.RawMessage }
	must(t, json.Unmarshal(originalBytes, &parentCode))
	must(t, json.Unmarshal(helperArtifact.Bytes(), &helperCode))
	// Both programs are acyclic except the one-element foreach. Its final next
	// test is the sole possible repeated instruction. One admission, one reply,
	// and one outcome delivery give three protocol dispatches, at capacity one.
	commitBound := len(parentCode.Code) + len(helperCode.Code) + 1
	for _, tc := range []struct{ root, operation, branch string }{
		{root: "probeA", operation: frozenLiteral, branch: "literal_equal"},
		{root: "probeB", operation: `["probeB",1,2,0]`, branch: "literal_different"},
	} {
		t.Run(tc.root, func(t *testing.T) {
			p, err := (author.Application{tc.root: {
				Artifact: artifact, Inputs: map[string]schema.Value{},
			}}).Provision()
			must(t, err)
			root := runtime.RunID{Subject: tc.root, Generation: 1}
			if len(p.Runs) != 1 || p.Runs[0].ID != root || len(p.Runs[0].Projection.Data) != 0 {
				t.Fatal("probe is not a sole root with empty application input", p.Runs)
			}
			p.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root},
				Templates: []string{helper.ID()}, Limits: runtime.ResourceLimits{
					MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxEffects: 0, MaxInFlight: 1,
				}}}
			path := filepath.Join(t.TempDir(), "handle-observation.sqlite")
			store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			must(t, err)
			defer store.Close()
			report, err := store.Run(t.Context(), runtime.RunOptions{
				MaxSteps: commitBound + 1, MaxDispatch: 4, MaxInFlight: 1,
			})
			must(t, err)
			state, err := store.Snapshot(t.Context(), root)
			must(t, err)
			if state.ProgramID != artifact.ID() || !bytes.Equal(originalBytes, artifact.Bytes()) {
				t.Fatal("parent artifact changed between root provisions")
			}
			data := state.Projection.Data
			if state.Projection.Status != "succeeded" || data["branch"].Text != tc.branch {
				t.Fatal("frozen branch prediction failed", state)
			}
			if data["original"].Text != tc.operation || data["saved"].Text != tc.operation {
				t.Fatal("collection round-trip changed original operation", data)
			}
			saved, err := data["handles"].Records()
			must(t, err)
			if len(saved) != 1 || saved[0]["operation"] != schema.Text(tc.operation) {
				t.Fatal("retained collection does not contain original operation", saved)
			}
			if data["child_status"].Text != "succeeded" || data["child_reason"].Text != "member_succeeded" ||
				!data["child_present"].Flag || data["child_value"] != schema.Int(7) {
				t.Fatal("actual child payload/status was not retained", data)
			}
			snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			must(t, err)
			diagnosticRelayCheckHandleEvidence(t, snapshot, root, artifact, helper, tc.operation, tc.branch)
			if !report.Quiescent || len(report.OpenRuns) != 0 || report.RejectedAdmissions != 0 ||
				report.Steps > commitBound || report.Dispatches != 3 || len(snapshot.Tables["history"]) > commitBound {
				t.Fatal("finite operational envelope", report, commitBound)
			}
			replay, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
			must(t, err)
			after, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			must(t, err)
			if !reflect.DeepEqual(snapshot, after) {
				t.Fatal("read-only replay changed retained evidence")
			}
			// The recorder keeps verbose output, including the complete genuine
			// retained rows, after Go has cleaned up its temporary store.
			evidence, err := json.Marshal(map[string]any{
				"parent_artifact": artifact.ID(), "parent_sha256": hex.EncodeToString(artifactHash[:]),
				"helper_template": helper.ID(), "frozen_literal": frozenLiteral,
				"root": root, "operation": tc.operation, "branch": tc.branch,
				"commit_bound": commitBound, "dispatch_bound": 3,
				"report": report, "state": state, "inspection": snapshot, "replay": replay,
			})
			must(t, err)
			t.Logf("handle_observation_evidence=%s", evidence)
		})
	}
}

func diagnosticRelayCheckHandleEvidence(
	t *testing.T,
	s runtime.Inspection,
	root runtime.RunID,
	artifact author.Artifact,
	helper author.Template,
	operation, branch string,
) {
	t.Helper()
	for _, table := range []string{"composition_invocations", "composition_admission_decisions",
		"composition_charges", "composition_results", "composition_publications", "composition_outcome_deliveries"} {
		if len(s.Tables[table]) != 1 {
			t.Fatal("expected exactly one original protocol record", table, len(s.Tables[table]))
		}
	}
	if len(s.Roots) != 1 || s.Roots[0].Run != root || s.Roots[0].Charges.Invocations != 1 ||
		s.Roots[0].Charges.CreatedRuns != 1 || s.Roots[0].Charges.Effects != 0 ||
		s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != "succeeded" {
		t.Fatal("root status or once-only charges", s.Roots)
	}
	admission := decode[runtime.InvocationAdmission](s.Tables["composition_admission_decisions"][0], "descriptor")
	if !admission.Accepted || len(admission.Members) != 1 || admission.Invocation.OperationID != operation ||
		admission.Invocation.Caller != root || admission.Invocation.TemplateID != helper.ID() ||
		admission.Invocation.Depth != 1 || admission.Invocation.SourceSequence != 2 ||
		admission.Invocation.ParentInvocationID != "" {
		t.Fatal("original invocation admission does not match handle and frame", admission)
	}
	outcome := decode[runtime.InvocationOutcome](s.Tables["composition_publications"][0], "descriptor")
	result := decode[runtime.InvocationOutcome](s.Tables["composition_results"][0], "descriptor")
	if !reflect.DeepEqual(outcome, result) || outcome.OperationID != operation || outcome.Caller != root ||
		outcome.TemplateID != helper.ID() || outcome.InvocationID != admission.Invocation.ID ||
		outcome.Status != "succeeded" || outcome.Reason != "member_succeeded" ||
		!outcome.ResponsePresent || !reflect.DeepEqual(outcome.Response, map[string]schema.Value{"value": schema.Int(7)}) {
		t.Fatal("original child publication/result correlation or payload", outcome, result)
	}
	child := projection(s, admission.Members[0].Subject)
	if child.Status != "succeeded" || child.Data["input_value"] != schema.Int(7) {
		t.Fatal("actual selected original child did not complete", child)
	}
	selected := ""
	selectedSequence := int64(0)
	branchRetained := false
	originalCommitted := false
	previous := runtime.Projection{}
	for _, row := range s.Tables["history"] {
		if row["subject"] != root.Subject || row["generation"] != root.Generation {
			continue
		}
		next := decode[runtime.Projection](row, "successor")
		for _, in := range decode[[]intent](row, "intents") {
			if in.Kind == "invocation" && in.ID == operation && in.Run == root && in.Sequence == 2 {
				originalCommitted = true
			}
		}
		if id, ok := row["selected_observation"].(string); ok && id != "" {
			if selected != "" || previous.Data["saved"].Text != operation {
				t.Fatal("selection did not use retained original handle", row, previous)
			}
			if _, receive := artifact.Metadata().Receives[previous.PC]; !receive {
				t.Fatal("selected history was not actual compiled Await", previous.PC)
			}
			selected, selectedSequence = id, row["sequence"].(int64)
		}
		if next.Data["branch"].Text == branch {
			branchRetained = true
		}
		previous = next
	}
	if selected == "" || !branchRetained || !originalCommitted {
		t.Fatal("missing original commit, selected observation or retained branch", selected, branchRetained, originalCommitted)
	}
	delivery := s.Tables["composition_outcome_deliveries"][0]
	if delivery["operation_id"] != operation || delivery["observation_id"] != selected {
		t.Fatal("actual selected observation is not original outcome delivery", delivery, selected)
	}
	found := false
	for _, row := range s.Tables["observations"] {
		if row["id"] != selected {
			continue
		}
		observed := decode[observation](row, "descriptor")
		expectedPayload := map[string]schema.Value{"status": schema.Text("succeeded"),
			"reason": schema.Text("member_succeeded"), "response_present": schema.Bool(true),
			"response.value": schema.Int(7)}
		if observed.Run != root || observed.Operation != operation || observed.Source != helper.ID() ||
			observed.SourceGeneration != 1 || observed.Provenance != "runtime-invocation" ||
			observed.Kind != "invocation" || observed.Obligation != "helper" ||
			row["consumed_sequence"] != selectedSequence || !reflect.DeepEqual(observed.Payload, expectedPayload) {
			t.Fatal("original selected observation provenance/correlation/payload", row)
		}
		found = true
	}
	if !found {
		t.Fatal("actual selected observation missing", selected)
	}
	programFound := false
	for _, row := range s.Tables["programs"] {
		if row["id"] == artifact.ID() {
			programFound = row["bytes"] == string(artifact.Bytes())
		}
	}
	if !programFound {
		t.Fatal("retained parent artifact bytes differ")
	}
}
