//go:build integration

package substitutiontest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	app "github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/research/nominal"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

const (
	nominalB = "2e10975004e0351d07b237e0360a2b96b50d5884b651cfa5bfc8e88af62da266"
	nominalH = "1806cf3a9e1e67ec0ed25a665b7a8a9be2dd7cf37585c5b1cd3e274b12595e3d"
	nominalJ = "8f537a4abec7a501f6b6875db112da8bd2d0bcc8ecbd7d16d2c97d7e8424a942"
	nominalR = "9f0841b4bc9a056d4be13e33d53042305dc3e7e36c6905799783f657fcc2c907"
)

type nominalPath struct {
	HelperOperation string
	LeafPCs         []int
	HelperPCs       []int
	Decision        schema.Value
}

func TestNominalCertifiedRelocation(t *testing.T) {
	leaf := candidate(t, "b")
	cert, err := nominal.CheckTemplate(leaf)
	must(t, err)
	for _, mode := range []string{"answer", "fail", "absent", "pending"} {
		t.Run(mode, func(t *testing.T) {
			var canonical nominalPath
			for k := 0; k <= 1; k++ {
				t.Run(fmt.Sprintf("k%d", k), func(t *testing.T) {
					cfg := substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: mode}, Limits: limits(4+k, 4+k, 3+k)}
					d, err := app.New(leaf, cfg)
					must(t, err)
					if !bytes.Equal(d.Candidate.Bytes(), leaf.Bytes()) || leaf.ID() != nominalB || d.Join.ID() != nominalJ ||
						d.Outer.ID() != "5e8fee04fd8855c501c1c20b87fdc4c2a1f96d6f4152baba3a7b5a48acb45621" {
						t.Fatal("original certified leaf or canonical J changed")
					}
					if k == 1 {
						relay, err := app.NewRelay(d.Join)
						must(t, err)
						if relay.ID() != nominalR {
							t.Fatal("relay differs from measured construction", relay.ID())
						}
						d.Outer, err = app.NewOuter(relay)
						must(t, err)
						d.Provision, err = (author.Application{"root": {Artifact: d.Outer, Inputs: map[string]schema.Value{
							"request_id": schema.Int(1), "request_value": schema.Int(0), "request_mode": schema.Text(mode),
						}}}).Provision()
						must(t, err)
						var catalogue []string
						for _, desc := range d.Provision.Composition.Templates {
							catalogue = append(catalogue, desc.ID)
						}
						root := runtime.RunID{Subject: "root", Generation: 1}
						d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: catalogue, Limits: cfg.Limits}}
					}
					path := filepath.Join(t.TempDir(), "relocation.db")
					store, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
					must(t, err)
					steps, dispatch := 149+71*k, 12+3*k
					report, err := store.Run(t.Context(), runtime.RunOptions{MaxSteps: steps + 1, MaxDispatch: dispatch + 1, MaxInFlight: 1})
					must(t, store.Close())
					must(t, err)
					s := inspect(t, path)
					if !report.Quiescent || report.RejectedAdmissions != 0 || report.Steps > steps || len(s.Tables["history"]) > steps || report.Dispatches > dispatch {
						t.Fatal("frozen bound or quiescence", report)
					}
					actual := nominalCheckFrame(t, s, d, k, mode, report)
					if k == 0 {
						canonical = actual
					} else if actual.HelperOperation == canonical.HelperOperation || actual.Decision != canonical.Decision ||
						!reflect.DeepEqual(actual.LeafPCs, canonical.LeafPCs) || !reflect.DeepEqual(actual.HelperPCs, canonical.HelperPCs) {
						t.Fatal("required change of binding or private control path", canonical, actual)
					}
					replayUnchanged(t, path)
					retain(t, "relocation", map[string]any{"k": k, "mode": mode, "certificate": cert, "path": actual,
						"commit_bound": steps, "dispatch_bound": dispatch, "capacity": 1, "report": report, "inspection": s})
				})
			}
		})
	}
}

func nominalCheckFrame(t *testing.T, s runtime.Inspection, d app.Deployment, k int, mode string, report runtime.RunReport) nominalPath {
	t.Helper()
	closed, err := validateRetained(s)
	must(t, err)
	if closed || len(s.Roots) != 1 || s.Roots[0].Limits != limits(4+k, 4+k, 3+k) ||
		s.Roots[0].Charges.Invocations != int64(4+k) || s.Roots[0].Charges.CreatedRuns != int64(4+k) || s.Roots[0].Charges.Effects != 0 {
		t.Fatal("root, charges or depth")
	}
	programs := make(map[string][]byte)
	for _, row := range s.Tables["programs"] {
		programs[row["id"].(string)] = []byte(row["bytes"].(string))
	}
	for _, original := range d.Provision.Programs {
		if !bytes.Equal(programs[original.ID], original.Bytes) {
			t.Fatal("deployed program bytes differ", original.ID)
		}
	}
	// validateRetained has already checked the originating history, binding,
	// actual selected observation, exact Await and frozen outward reply.
	byTemplate := make(map[string]runtime.InvocationAdmission)
	for _, row := range s.Tables["composition_admission_decisions"] {
		a := decode[runtime.InvocationAdmission](row, "descriptor")
		if _, dup := byTemplate[a.Invocation.TemplateID]; dup || !a.Accepted || len(a.Members) != 1 {
			t.Fatal("expected once-only singleton", a)
		}
		byTemplate[a.Invocation.TemplateID] = a
	}
	if len(byTemplate) != 4+k {
		t.Fatal("unexpected invocation topology")
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	owner := root
	if k == 1 {
		owner = nominalMember(t, byTemplate[nominalR], root, 1)
	}
	j := nominalMember(t, byTemplate[nominalJ], owner, 1+k)
	b := nominalMember(t, byTemplate[nominalB], j, 2+k)
	succ := nominalMember(t, byTemplate[d.Successor.ID()], j, 2+k)
	h := nominalMember(t, byTemplate[nominalH], b, 3+k)

	id, zero, one := schema.Int(1), schema.Int(0), schema.Int(1)
	response := map[string]schema.Value{"id": id, "value": one}
	request := map[string]schema.Value{"id": id, "value": zero, "mode": schema.Text(mode)}
	for _, template := range []string{nominalJ, nominalB, nominalR} {
		if a, ok := byTemplate[template]; ok && !sameValues(a.Invocation.Inputs, request) {
			t.Fatal("relocated request differs at entry", template, a.Invocation)
		}
	}
	check := func(template, status, reason string, payload map[string]schema.Value) {
		t.Helper()
		a := byTemplate[template]
		op := a.Invocation.OperationID
		row := communicationRow(s, "composition_publications", "operation_id", op)
		delivery := communicationRow(s, "composition_outcome_deliveries", "operation_id", op)
		replies := 0
		for _, rr := range s.Tables["composition_replies"] {
			bound := decode[reply](rr, "descriptor")
			if bound.InvocationID != a.Invocation.ID {
				continue
			}
			replies++
			if !bound.Accepted || bound.Intent.Run != a.Members[0] || bound.Intent.Target != a.Invocation.Caller.Subject ||
				bound.Intent.Generation != a.Invocation.Caller.Generation || !sameValues(bound.Intent.Payload, payload) {
				t.Fatal("reply used a different original binding or report", template, bound)
			}
		}
		wantReplies := 0
		if payload != nil {
			wantReplies = 1
		}
		if replies != wantReplies {
			t.Fatal("bound reply count", template, replies, wantReplies)
		}
		if status == "open" {
			if row != nil || delivery != nil || projection(s, a.Members[0].Subject).Status != "open" {
				t.Fatal("unpublished wait has outcome", template)
			}
			return
		}
		if row == nil || delivery == nil {
			t.Fatal("missing original publication or delivery", template)
		}
		outcome := decode[runtime.InvocationOutcome](row, "descriptor")
		if outcome.Status != status || outcome.Reason != reason || outcome.ResponsePresent != (payload != nil) || !sameValues(outcome.Response, payload) {
			t.Fatal("selected child envelope", template, outcome)
		}
		selected := communicationRow(s, "history", "selected_observation", delivery["observation_id"])
		if selected == nil || selected["subject"] != a.Invocation.Caller.Subject || selected["generation"] != a.Invocation.Caller.Generation {
			t.Fatal("original outcome not selected by its caller", op)
		}
	}
	check(nominalH, "succeeded", "member_succeeded", response)
	check(d.Successor.ID(), "succeeded", "member_succeeded", map[string]schema.Value{"id": id, "value": zero})
	bdata := projection(s, b.Subject).Data
	if bdata["requested_mode"] != schema.Text(mode) || bdata["helper_operation"] != schema.Text(byTemplate[nominalH].Invocation.OperationID) ||
		bdata["helper_value"] != one || bdata["helper_id"] != id || projection(s, succ.Subject).Status != "succeeded" {
		t.Fatal("leaf did not actually consume its request and chosen helper")
	}
	outer := projection(s, root.Subject)
	if mode == "pending" {
		check(nominalB, "open", "", nil)
		check(nominalJ, "open", "", nil)
		if k == 1 {
			check(nominalR, "open", "", nil)
		}
		if report.Roots[0].Outcome != nil || len(report.OpenRuns) != 3+k || projection(s, j.Subject).Data["s_present"] != schema.Bool(true) {
			t.Fatal("unsettled work outside the intended pending chain")
		}
		entry := nominalJ
		if k == 1 {
			entry = nominalR
		}
		nominalAwait(t, s, programs, root, "join_operation", byTemplate[entry].Invocation.OperationID)
		if k == 1 {
			nominalAwait(t, s, programs, owner, "child_operation", byTemplate[nominalJ].Invocation.OperationID)
		}
		nominalAwait(t, s, programs, j, "candidate_operation", byTemplate[nominalB].Invocation.OperationID)
		nominalAwait(t, s, programs, b, "", "")
		if _, ok := outer.Data["decision"]; ok {
			t.Fatal("premature outer decision")
		}
	} else {
		tag := map[string]string{"answer": "answer", "fail": "candidate_failed", "absent": "candidate_missing_reply"}[mode]
		status, answers := "succeeded", schema.List(response)
		if mode == "answer" {
			check(nominalB, status, "member_succeeded", response)
		} else {
			status, answers = "failed", schema.List()
			reason := map[string]string{"fail": "member_failed", "absent": "missing_reply"}[mode]
			check(nominalB, status, reason, nil)
		}
		diag := map[string]schema.Value{"id": id, "classification": schema.Text(tag), "answers": answers}
		check(nominalJ, status, "member_"+status, diag)
		if k == 1 {
			check(nominalR, status, "member_"+status, diag)
		}
		if len(report.OpenRuns) != 0 || outer.Data["decision"] != schema.Text(tag) || s.Roots[0].Outcome == nil ||
			s.Roots[0].Outcome.Status != status || s.Roots[0].Outcome.Reason != "member_"+status {
			t.Fatal("final outer decision/End", outer)
		}
	}
	pcs := func(id runtime.RunID) []int {
		var pcs []int
		for _, row := range s.Tables["history"] {
			if row["subject"] == id.Subject && row["generation"] == id.Generation {
				pcs = append(pcs, decode[runtime.Projection](row, "successor").PC)
			}
		}
		return pcs
	}
	return nominalPath{byTemplate[nominalH].Invocation.OperationID, pcs(b), pcs(h), outer.Data["decision"]}
}

func nominalMember(t *testing.T, a runtime.InvocationAdmission, caller runtime.RunID, depth int) runtime.RunID {
	t.Helper()
	if !a.Accepted || len(a.Members) != 1 || a.Invocation.Caller != caller || a.Invocation.Depth != int64(depth) {
		t.Fatal("actual caller, member or depth", a, caller, depth)
	}
	return a.Members[0]
}

// Identify the retained instruction where a pending run is stopped. An empty
// register denotes B's unmatched second request on its bound public entry port.
func nominalAwait(t *testing.T, s runtime.Inspection, programs map[string][]byte, id runtime.RunID, register, operation string) {
	t.Helper()
	row := communicationRow(s, "runs", "subject", id.Subject)
	proj := projection(s, id.Subject)
	var code evidenceProgram
	must(t, json.Unmarshal(programs[row["program_id"].(string)], &code))
	in := code.Code[proj.PC]
	if proj.Status != "open" || in.Op != "receive" || in.Match == nil ||
		(register != "" && (in.Match.Kind != "invocation" || in.Match.Operation.Ref != register || proj.Data[register] != schema.Text(operation))) ||
		(register == "" && in.Match.Kind != "message") {
		t.Fatal("wrong pending instruction", id, proj.PC, in.Match)
	}
}
