//go:build integration

package substitutiontest

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	joinapp "github.com/JakeFAU/alan/examples/join"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/internal/executor"
	joinmodel "github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestJoinDecoder(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, e := joinapp.New(candidate(t, "a"), app.Config{Shape: "parallel", Request: q, Limits: limits(3, 3, 2)}, joinapp.Options{})
	must(t, e)
	path := filepath.Join(t.TempDir(), "retained.db")
	create(t, d, path)
	run(t, path)
	original := inspect(t, path)
	tr, e := decodeJoin(original, q)
	must(t, e)
	if v := joinmodel.CheckTrace(tr); len(v) > 0 {
		t.Fatal(v)
	}
	raw, e := json.Marshal(original)
	must(t, e)
	for _, mutation := range []string{"report_payload", "report_generation", "report_selected_identity", "report_bound_payload", "authorization_snapshot", "delivery_identity", "ack_identity", "duplicate_observation"} {
		t.Run(mutation, func(t *testing.T) {
			var s runtime.Inspection
			must(t, json.Unmarshal(raw, &s))
			changed := false
			switch mutation {
			case "report_payload", "report_generation", "duplicate_observation":
				for _, row := range s.Tables["observations"] {
					o := decode[observation](row, "descriptor")
					if o.Provenance != "ordinary-message" || o.Source != "peer" {
						continue
					}
					changed = true
					switch mutation {
					case "report_payload":
						o.Payload["response_value"] = schema.Int(9)
					case "report_generation":
						o.SourceGeneration = 2
					case "duplicate_observation":
						copy := map[string]any{}
						for k, v := range row {
							copy[k] = v
						}
						s.Tables["observations"] = append(s.Tables["observations"], copy)
					}
					if mutation != "duplicate_observation" {
						b, e := json.Marshal(o)
						must(t, e)
						row["descriptor"] = string(b)
					}
					break
				}
			case "report_selected_identity", "report_bound_payload":
				for _, row := range s.Tables["history"] {
					if row["subject"] != "root" {
						continue
					}
					selected, _ := row["selected_observation"].(string)
					for _, oRow := range s.Tables["observations"] {
						o := decode[observation](oRow, "descriptor")
						if o.ID == selected && o.Provenance == "ordinary-message" {
							changed = true
							if mutation == "report_selected_identity" {
								row["selected_observation"] = "wrong-original-report"
							} else {
								p := decode[runtime.Projection](row, "successor")
								p.Data["join_value"] = schema.Int(8)
								b, e := json.Marshal(p)
								must(t, e)
								row["successor"] = string(b)
							}
						}
					}
				}
			case "authorization_snapshot":
				for _, row := range s.Tables["attempts"] {
					a := decode[executor.Authorization](row, "result")
					if a.Source.Run.Subject != "peer" {
						continue
					}
					a.Source.Sequence = 1
					raw, e := json.Marshal(a)
					must(t, e)
					row["result"] = string(raw)
					changed = true
				}
			case "delivery_identity":
				for _, row := range s.Tables["admissions"] {
					a := decode[executor.AdmissionDescriptor](row, "descriptor")
					if a.Kind != "message" || a.Source.Subject != "peer" {
						continue
					}
					a.LogicalID = "wrong-original-message"
					b, e := json.Marshal(a)
					must(t, e)
					row["descriptor"] = string(b)
					changed = true
				}
			case "ack_identity":
				for _, row := range s.Tables["message_acknowledgments"] {
					for _, inRow := range s.Tables["intents"] {
						i := decode[intent](inRow, "descriptor")
						if i.ID == row["intent_id"] && i.Run.Subject == "peer" {
							row["admission_id"] = "wrong-admission"
							changed = true
						}
					}
				}
			}
			if !changed {
				t.Fatal("mutation target absent")
			}
			_, e := decodeJoin(s, q)
			if e == nil {
				t.Fatal("decoder accepted mutation", mutation)
			}
			retain(t, "rejection", map[string]any{"mutation": mutation, "error": e.Error()})
		})
	}
	// Coherent report facts still cannot make an unconsumed/wrong S operation true.
	for _, mutation := range []string{"causal-order", "wrong-s-operation", "wrong-request", "extra-report", "report-before-candidate"} {
		t.Run(mutation, func(t *testing.T) {
			altered := tr
			altered.Events = append([]joinmodel.Event(nil), tr.Events...)
			for i := range altered.Events {
				v := &altered.Events[i]
				if v.Kind == "report_commit" {
					switch mutation {
					case "causal-order":
						v.Sequence = 1
					case "wrong-s-operation":
						v.Report.SOperation = "different-S-operation"
					case "wrong-request":
						v.Report.RequestID = 2
					}
				}
			}

			if mutation == "extra-report" {
				for _, event := range tr.Events {
					if event.Kind == "report_commit" {
						event.Operation = "distinct-extra-peer-send"
						event.Sequence++
						altered.Events = append(altered.Events, event)
						break
					}
				}
			}
			if mutation == "report-before-candidate" {
				for i := range altered.Events {
					if altered.Events[i].Kind == "report_consume" {
						altered.Events[i].Sequence = 1
					}
				}
			}
			v := joinmodel.CheckTrace(altered)
			if len(v) == 0 {
				t.Fatal("independent oracle accepted mutation")
			}
			retain(t, "rejection", map[string]any{"mutation": mutation, "violations": v})
		})
	}
}
func TestJoinDuplicateDelivery(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, e := joinapp.New(candidate(t, "a"), app.Config{Shape: "parallel", Request: q, Limits: limits(3, 3, 2)}, joinapp.Options{})
	must(t, e)
	path := filepath.Join(t.TempDir(), "retained.db")
	create(t, d, path)
	run(t, path)
	before := inspect(t, path)
	var descriptor executor.AdmissionDescriptor
	var result runtime.AdmissionResult
	for _, row := range before.Tables["admissions"] {
		d := decode[executor.AdmissionDescriptor](row, "descriptor")
		if d.Kind == "message" && d.Source.Subject == "peer" {
			descriptor = d
			result = decode[runtime.AdmissionResult](row, "result")
		}
	}
	if descriptor.IntentID == "" || !result.Accepted {
		t.Fatal("original accepted report absent")
	}
	store, e := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, e)
	for range 2 {
		again, e := store.DeliverMessage(t.Context(), descriptor.IntentID, descriptor.TransportID)
		must(t, e)
		if !reflect.DeepEqual(again, result) {
			t.Fatal("duplicate changed retained delivery", again, result)
		}
	}
	must(t, store.Close())
	after := inspect(t, path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("duplicate delivery changed retained ledger")
	}
	retain(t, "result", map[string]any{"original": descriptor, "retained_result": result, "duplicate_attempts": 2, "ledger_unchanged": true})
	replayUnchanged(t, path)
}
