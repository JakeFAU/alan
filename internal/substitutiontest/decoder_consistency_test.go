//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func editEvidenceJSON(t *testing.T, row map[string]any, key string, edit func(map[string]any)) {
	t.Helper()
	raw, ok := row[key].(string)
	if !ok {
		t.Fatal("original JSON field missing", key)
	}
	var value map[string]any
	must(t, json.Unmarshal([]byte(raw), &value))
	edit(value)
	b, e := json.Marshal(value)
	must(t, e)
	row[key] = string(b)
}
func TestSubstitutionDecoderConsistency(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, path := provision(t, "b", "bare", q, limits(3, 3, 3))
	create(t, d, path)
	run(t, path)
	original := inspect(t, path)
	valid := normalize(original, d, q)
	if len(valid.Violations) > 0 || !valid.EvidenceChecked {
		t.Fatal(valid.Violations)
	}
	raw, e := json.Marshal(original)
	must(t, e)
	candidateSubject, candidateScope, rootProgram := "", "", ""
	for _, o := range valid.Operations {
		if o.Role == "candidate" {
			candidateSubject = o.Members[0].Subject
			candidateScope = o.Invocation
		}
	}
	for _, r := range original.Tables["runs"] {
		if r["subject"] == "root" {
			rootProgram = r["program_id"].(string)
		}
	}
	mutations := []string{"missing_observation", "wrong_source", "wrong_operation", "wrong_obligation", "wrong_caller", "wrong_consumed_sequence", "missing_selected_pointer", "wrong_pending_handle", "wrong_receive_source", "wrong_receive_handle_register", "wrong_parent", "wrong_root", "wrong_admission_caller", "missing_binding", "missing_member", "missing_publication", "missing_delivery", "missing_history", "missing_execution_fact", "wrong_freeze_status", "wrong_freeze_scope", "wrong_freeze_run", "wrong_freeze_sequence", "missing_freeze_trigger", "missing_root_summary", "missing_freeze_capture", "missing_payload_status", "wrong_presence_type", "wrong_bound_successor", "wrong_terminal_history", "missing_root_publication", "missing_publication_fact", "missing_freeze_fact"}
	for _, mutation := range mutations {
		t.Run(mutation, func(t *testing.T) {
			var s runtime.Inspection
			must(t, json.Unmarshal(raw, &s))
			changed := false
			change := func(table, key string, selectRow func(map[string]any) bool, edit func(map[string]any)) {
				for _, r := range s.Tables[table] {
					if selectRow(r) {
						editEvidenceJSON(t, r, key, edit)
						changed = true
						return
					}
				}
			}
			remove := func(table string, selectRow func(map[string]any) bool) {
				for i, r := range s.Tables[table] {
					if selectRow(r) {
						s.Tables[table] = append(s.Tables[table][:i], s.Tables[table][i+1:]...)
						changed = true
						return
					}
				}
			}
			rootObservation := func(r map[string]any) bool {
				o := decode[observation](r, "descriptor")
				return o.Kind == "invocation" && o.Run.Subject == "root"
			}
			rootSelected := func(r map[string]any) bool { return r["subject"] == "root" && r["selected_observation"] != nil }
			candidateFreeze := func(r map[string]any) bool { return r["scope_id"] == candidateScope }
			rootAdmission := func(r map[string]any) bool {
				return decode[runtime.InvocationAdmission](r, "descriptor").Invocation.Caller.Subject == "root"
			}
			switch mutation {
			case "missing_observation":
				remove("observations", rootObservation)
			case "wrong_source", "wrong_operation", "wrong_obligation", "wrong_caller", "missing_payload_status", "wrong_presence_type":
				change("observations", "descriptor", rootObservation, func(v map[string]any) {
					switch mutation {
					case "wrong_source":
						v["Source"] = "another-template"
					case "wrong_operation":
						v["Operation"] = "another-operation"
					case "wrong_obligation":
						v["Obligation"] = "other-invocation"
					case "wrong_caller":
						v["Run"].(map[string]any)["Subject"] = "peer"
					case "missing_payload_status":
						delete(v["Payload"].(map[string]any), "status")
					case "wrong_presence_type":
						v["Payload"].(map[string]any)["response_present"] = schema.Text("true")
					}
				})
			case "wrong_consumed_sequence":
				for _, r := range s.Tables["observations"] {
					if rootObservation(r) {
						r["consumed_sequence"] = 99
						changed = true
						break
					}
				}
			case "missing_selected_pointer":
				for _, r := range s.Tables["history"] {
					if rootSelected(r) {
						r["selected_observation"] = nil
						changed = true
						break
					}
				}
			case "wrong_pending_handle":
				change("history", "successor", func(r map[string]any) bool { return r["subject"] == "root" && r["sequence"] == float64(1) }, func(v map[string]any) { v["Data"].(map[string]any)["pending"] = schema.Text("wrong-original-handle") })
			case "wrong_receive_source", "wrong_receive_handle_register":
				change("programs", "bytes", func(r map[string]any) bool { return r["id"] == rootProgram }, func(v map[string]any) {
					code := v["Code"].([]any)
					for _, step := range code {
						c := step.(map[string]any)
						if c["Op"] != "receive" {
							continue
						}
						m := c["Match"].(map[string]any)
						if mutation == "wrong_receive_source" {
							m["Source"].(map[string]any)["Literal"] = schema.Text("wrong-template")
						} else {
							m["Operation"].(map[string]any)["Ref"] = "wrong-handle-register"
						}
						return
					}
				})
			case "wrong_parent", "wrong_root", "wrong_admission_caller":
				change("composition_admission_decisions", "descriptor", rootAdmission, func(v map[string]any) {
					i := v["Invocation"].(map[string]any)
					switch mutation {
					case "wrong_parent":
						i["ParentInvocationID"] = "wrong-parent"
					case "wrong_root":
						i["RootID"] = "wrong-root"
					case "wrong_admission_caller":
						i["Caller"].(map[string]any)["Generation"] = 99
					}
				})
			case "missing_binding":
				remove("composition_bindings", func(r map[string]any) bool {
					return decode[evidenceBinding](r, "descriptor").InvocationID == candidateScope
				})
			case "missing_member":
				remove("composition_members", func(r map[string]any) bool { return r["subject"] == candidateSubject })
			case "missing_publication":
				remove("composition_publications", func(r map[string]any) bool {
					return decode[runtime.InvocationOutcome](r, "descriptor").Caller.Subject == "root"
				})
			case "missing_delivery":
				remove("composition_outcome_deliveries", func(r map[string]any) bool {
					return r["operation_id"] == valid.Operations[nextCandidateOperation(valid)].Operation
				})
			case "missing_history":
				remove("history", rootSelected)
			case "missing_execution_fact":
				remove("composition_facts", func(r map[string]any) bool {
					f := decode[fact](r, "descriptor")
					return f.Kind == "execution" && f.Run.Subject == "root" && f.Sequence == 2
				})
			case "wrong_freeze_status", "wrong_freeze_scope", "wrong_freeze_run", "wrong_freeze_sequence", "missing_freeze_trigger", "missing_freeze_capture":
				change("composition_freezes", "descriptor", candidateFreeze, func(v map[string]any) {
					switch mutation {
					case "wrong_freeze_status":
						v["Status"] = "failed"
					case "wrong_freeze_scope":
						v["InvocationID"] = "wrong-invocation"
					case "wrong_freeze_run":
						v["Run"].(map[string]any)["Generation"] = 99
					case "wrong_freeze_sequence":
						v["Sequence"] = 99
					case "missing_freeze_trigger":
						v["Trigger"] = "missing-trigger"
					case "missing_freeze_capture":
						v["InvocationIntents"] = []any{}
					}
				})
			case "missing_root_summary":
				s.Roots = nil
				changed = true
			case "wrong_bound_successor":
				change("history", "successor", rootSelected, func(v map[string]any) { v["Data"].(map[string]any)["answer_id"] = schema.Int(2) })
			case "wrong_terminal_history":
				change("history", "successor", func(r map[string]any) bool {
					return r["subject"] == candidateSubject && decode[runtime.Projection](r, "successor").Status == "succeeded"
				}, func(v map[string]any) { v["Status"] = "failed" })
			case "missing_root_publication":
				s.Tables["composition_root_publications"] = nil
				changed = true
			case "missing_publication_fact":
				remove("composition_facts", func(r map[string]any) bool {
					f := decode[fact](r, "descriptor")
					return f.Kind == "outcome" && f.OperationID == nextCandidateOperation(valid)
				})
			case "missing_freeze_fact":
				remove("composition_facts", func(r map[string]any) bool {
					f := decode[fact](r, "descriptor")
					return f.Kind == "freeze" && f.OperationID == candidateScope
				})
			}
			if !changed {
				t.Fatal("control did not mutate actual original evidence")
			}
			n := normalize(s, d, q)
			retain(t, "isolated-control", map[string]any{"mutation": mutation, "original": original, "mutated": s, "normalized": n})
			if n.EvidenceChecked || len(n.Violations) == 0 || n.Violations[0].Code != "retained_evidence" {
				t.Fatalf("missing explicit raw-evidence rejection for %s: %+v", mutation, n)
			}
		})
	}
}
func nextCandidateOperation(n normalized) string {
	for op, o := range n.Operations {
		if o.Role == "candidate" {
			return op
		}
	}
	panic("candidate operation missing")
}
func TestSubstitutionDecoderCancellationEvidence(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, _ := provision(t, "b", "bare", q, limits(3, 3, 3))
	original := originalRecord(t, "normal-v1", "TestSubstitutionClosure/b/helper-commit", "after")
	for _, mutation := range []string{"missing_accepted_lifecycle", "mismatched_cancelled_projection", "mismatched_lifecycle_target"} {
		t.Run(mutation, func(t *testing.T) {
			b, e := json.Marshal(original)
			must(t, e)
			var s runtime.Inspection
			must(t, json.Unmarshal(b, &s))
			switch mutation {
			case "missing_accepted_lifecycle":
				s.Tables["lifecycle_results"] = nil
			case "mismatched_cancelled_projection":
				editEvidenceJSON(t, s.Tables["lifecycle_results"][0], "result", func(v map[string]any) {
					v["After"].(map[string]any)["Projection"].(map[string]any)["Status"] = "succeeded"
				})
			case "mismatched_lifecycle_target":
				editEvidenceJSON(t, s.Tables["lifecycle_results"][0], "result", func(v map[string]any) { v["Request"].(map[string]any)["Intent"].(map[string]any)["Target"] = "peer" })
			}
			n := normalize(s, d, q)
			retain(t, "cancellation-control", map[string]any{"original": original, "mutated": s, "normalized": n})
			if n.EvidenceChecked || len(n.Violations) == 0 {
				t.Fatal(fmt.Sprintf("cancellation evidence control accepted: %s", mutation))
			}
		})
	}
}
