//go:build integration

package substitutiontest

import (
	"encoding/json"
	app "github.com/JakeFAU/alan/examples/substitution"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"reflect"
	"testing"
)

func TestReviewConsumedPayloadIdentityPreserved(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, p := provision(t, "b", "bare", q, limits(3, 3, 3))
	create(t, d, p)
	run(t, p)
	original := inspect(t, p)
	before := normalize(original, d, q)
	if len(before.Violations) != 0 {
		t.Fatal(before.Violations)
	}
	bytes, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"consumed_payload", "consumed_provenance", "consumed_generation", "candidate_local_status", "missing_member_freezes"} {
		t.Run(mutation, func(t *testing.T) {
			var s runtime.Inspection
			if err := json.Unmarshal(bytes, &s); err != nil {
				t.Fatal(err)
			}
			changed := 0
			if mutation == "missing_member_freezes" {
				changed = len(s.Tables["composition_freezes"])
				s.Tables["composition_freezes"] = nil
			} else if mutation == "candidate_local_status" {
				subject := ""
				for _, o := range before.Operations {
					if o.Role == "candidate" && len(o.Members) == 1 {
						subject = o.Members[0].Subject
					}
				}
				if subject == "" {
					t.Fatal("candidate member missing")
				}
				for _, r := range s.Tables["runs"] {
					if r["subject"] == subject {
						var p map[string]any
						if err := json.Unmarshal([]byte(r["projection"].(string)), &p); err != nil {
							t.Fatal(err)
						}
						if p["Status"] != "succeeded" {
							t.Fatal(p)
						}
						p["Status"] = "failed"
						b, e := json.Marshal(p)
						if e != nil {
							t.Fatal(e)
						}
						r["projection"] = string(b)
						changed++
					}
				}
			} else {
				for _, r := range s.Tables["observations"] {
					var o map[string]any
					if err := json.Unmarshal([]byte(r["descriptor"].(string)), &o); err != nil {
						t.Fatal(err)
					}
					if o["Kind"] != "invocation" {
						continue
					}
					switch mutation {
					case "consumed_payload":
						payload := o["Payload"].(map[string]any)
						if _, ok := payload["response.id"]; !ok {
							t.Fatal("response.id absent")
						}
						payload["response.id"] = schema.Int(2)
					case "consumed_provenance":
						if _, ok := o["Provenance"]; !ok {
							t.Fatal("Provenance absent")
						}
						o["Provenance"] = "ordinary-message"
					case "consumed_generation":
						if _, ok := o["SourceGeneration"]; !ok {
							t.Fatal("SourceGeneration absent")
						}
						o["SourceGeneration"] = 99
					}
					b, e := json.Marshal(o)
					if e != nil {
						t.Fatal(e)
					}
					r["descriptor"] = string(b)
					changed++
				}
			}
			if changed == 0 {
				t.Fatal("no actual retained records selected")
			}
			after := normalize(s, d, q)
			err := model.CheckCandidateContext(model.Config{Shape: "bare", Request: model.Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 3}, model.Helper, after.Context)
			retain(t, "input", map[string]any{"mutation": mutation, "original": original, "mutated": s, "before_normalized": before, "after_normalized": after})
			t.Logf("mutated records=%d; trace identical=%v; context identical=%v; violations=%v; membership=%v", changed, reflect.DeepEqual(before.Trace, after.Trace), before.Context == after.Context, after.Violations, err)
			if len(after.Violations) == 0 || after.Violations[0].Code != "retained_evidence" || after.EvidenceChecked {
				t.Fatal("normalizer accepted mismatched retained facts")
			}
		})
	}
}
