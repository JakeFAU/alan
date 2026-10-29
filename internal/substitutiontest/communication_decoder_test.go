//go:build integration

package substitutiontest

import (
	"encoding/json"
	"slices"
	"testing"

	joinapp "github.com/JakeFAU/alan/examples/join"
	app "github.com/JakeFAU/alan/examples/substitution"
	communication "github.com/JakeFAU/alan/research/communication"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestCommunicationDecoder(t *testing.T) {
	q := model.Request{ID: 1, Value: 0, Mode: "answer"}
	originals := map[string]runtime.Inspection{}
	// Every selected case can obtain its genuine source directly. The cache
	// shares immutable snapshots without requiring a sibling subtest to run.
	original := func(t *testing.T, source string) runtime.Inspection {
		t.Helper()
		if s, ok := originals[source]; ok {
			return s
		}
		path := communicationFixture(t)
		var s runtime.Inspection
		switch source {
		case "late", "early":
			s = driveCommunication(t, path, communication.J1, source)
		case "admitted":
			d := newCommunicationDriver(t, path)
			s = d.drive("report-authorized-peer-terminal")
			d.deliver(communicationReport(s).ID)
			s = inspect(t, path)
		default:
			t.Fatalf("unknown decoder source %q", source)
		}
		originals[source] = s
		return s
	}
	for _, order := range []string{"late", "early"} {
		t.Run("source-"+order, func(t *testing.T) {
			_ = original(t, order)
		})
	}
	t.Run("admitted-unconsumed", func(t *testing.T) {
		s := original(t, "admitted")
		tr, e := decodeCommunication(s, q, communication.J1)
		must(t, e)
		if !tr.Messages[0].Admission.Accepted || tr.Messages[0].Consumption != nil || projection(s, "root").Status != "open" {
			t.Fatal("admission incorrectly became consumption")
		}
		retain(t, "positive-prefix", map[string]any{"inspection": s, "trace": tr})
	})
	cases := []struct {
		name, source string
		edit         func(*testing.T, *runtime.Inspection)
	}{
		{"denial-ack-is-not-delivery", "late", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			r := communicationRow(*s, "admissions", "transport_id", communicationIdentity("message-delivery", in.ID))
			editEvidenceJSON(t, r, "result", func(v map[string]any) {
				v["Accepted"] = true
				v["Reason"] = "accepted"
				v["ObservationID"] = "invented"
			})
		}},
		{"admission-is-not-consumption", "admitted", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "observations", "id", communicationIdentity("observation", "message", in.ID))
			row["consumed_sequence"] = int64(3)
		}},
		{"swapped-original-intent", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "admissions", "transport_id", communicationIdentity("message-delivery", in.ID))
			editEvidenceJSON(t, row, "descriptor", func(v map[string]any) { v["IntentID"] = "foreign-original" })
		}},
		{"swapped-authorization", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "attempts", "intent_id", in.ID)
			editEvidenceJSON(t, row, "result", func(v map[string]any) { v["IntentID"] = "foreign-original" })
		}},
		{"changed-auth-sequence", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "attempts", "intent_id", in.ID)
			editEvidenceJSON(t, row, "result", func(v map[string]any) { v["Source"].(map[string]any)["Sequence"] = float64(1) })
		}},
		{"changed-auth-epoch", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "attempts", "intent_id", in.ID)
			editEvidenceJSON(t, row, "result", func(v map[string]any) { v["Source"].(map[string]any)["Epoch"] = float64(99) })
		}},
		{"changed-auth-descriptor-revision", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "attempts", "intent_id", in.ID)
			editEvidenceJSON(t, row, "descriptor", func(v map[string]any) { v["Revision"] = float64(2) })
		}},
		{"wrong-dispatch-link", "late", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			communicationRow(*s, "message_acknowledgments", "intent_id", in.ID)["admission_id"] = "foreign-admission"
		}},
		{"wrong-report-correlation", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			row := communicationRow(*s, "observations", "id", communicationIdentity("observation", "message", in.ID))
			editEvidenceJSON(t, row, "descriptor", func(v map[string]any) { v["Payload"].(map[string]any)["request_id"] = schema.Int(2) })
		}},
		{"wrong-selected-observation", "early", func(t *testing.T, s *runtime.Inspection) {
			in := communicationReport(*s)
			obs := communicationIdentity("observation", "message", in.ID)
			for _, h := range s.Tables["history"] {
				if h["selected_observation"] == obs {
					h["selected_observation"] = "foreign-observation"
				}
			}
		}},
		{"false-aggregate", "late", func(t *testing.T, s *runtime.Inspection) { s.Roots[0].LocalStatus = "succeeded" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source := original(t, c.source)
			raw, e := json.Marshal(source)
			must(t, e)
			var s runtime.Inspection
			must(t, json.Unmarshal(raw, &s))
			before := hash(s)
			c.edit(t, &s)
			if before == hash(s) {
				t.Fatal("mutation did not change raw evidence")
			}
			tr, e := decodeCommunication(s, q, communication.J1)
			if e == nil {
				t.Fatal("raw mutation accepted before normalization", tr)
			}
			retain(t, "rejection", map[string]any{"mutation": c.name, "source": c.source, "source_hash": hash(source), "mutated_hash": hash(s), "error": e.Error()})
		})
	}
	t.Run("unrelated-order", func(t *testing.T) {
		s := original(t, "early")
		raw, _ := json.Marshal(s)
		must(t, json.Unmarshal(raw, &s))
		for _, rows := range s.Tables {
			slices.Reverse(rows)
		}
		tr, e := decodeCommunication(s, q, communication.J1)
		must(t, e)
		if v := communication.CheckTrace(tr); len(v) > 0 {
			t.Fatal(v)
		}
		retain(t, "reordered", map[string]any{"trace": tr, "inspection_hash": hash(s)})
	})
	t.Run("J2-unavailable", func(t *testing.T) {
		if _, e := decodeCommunication(original(t, "early"), q, communication.J2); e == nil {
			t.Fatal("J1 relabeled as absent J2 artifact")
		}
	})
	for _, c := range []struct {
		name    string
		options joinapp.Options
	}{{"original-premature", joinapp.Options{PrematureReport: true}}, {"original-wrong-request", joinapp.Options{WrongRequest: true}}} {
		t.Run(c.name, func(t *testing.T) {
			d, e := joinapp.New(candidate(t, "a"), app.Config{Shape: "parallel", Request: app.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: limits(3, 3, 2)}, c.options)
			must(t, e)
			path := t.TempDir() + "/control.db"
			create(t, d, path)
			run(t, path)
			s := inspect(t, path)
			if _, e := decodeCommunication(s, q, communication.J1); e == nil {
				t.Fatal("unchanged J1 truth control accepted")
			}
			retain(t, "original-control", map[string]any{"source": "unchanged examples/join and research/join truth checker", "inspection": s, "programs": d.Provision.Programs, "options": c.options})
		})
	}
}

// A lookup miss is a valid prefix only when the independent truth/mapping
// checks also find no malformed committed report. These are copied-inspection
// decoder attacks, never authoritative database edits or new protocol histories.
func TestCommunicationDecoderReportIdentity(t *testing.T) {
	q := model.Request{ID: 1, Value: 0, Mode: "answer"}
	path := communicationFixture(t)
	d := newCommunicationDriver(t, path)
	if !d.step(runtime.RunID{Subject: "root", Generation: 1}) {
		t.Fatal("genuine pre-report Invoke was not enabled")
	}
	pre := inspect(t, path)
	t.Run("genuine-pre-report", func(t *testing.T) {
		if communicationReport(pre).ID != "" {
			t.Fatal("prefix already contains report")
		}
		tr, e := decodeCommunication(pre, q, communication.J1)
		must(t, e)
		if len(tr.Messages) != 0 || len(communication.CheckTrace(tr)) != 0 {
			t.Fatal("genuine pre-report prefix rejected or invented message", tr)
		}
		retain(t, "genuine-prefix", map[string]any{"inspection": pre, "trace": tr})
	})
	d.toReport()
	original := inspect(t, path)
	if !communicationCut(original, "report-no-authorization") {
		t.Fatal("genuine report prefix missing")
	}
	retain(t, "original-no-authorization", original)
	raw, e := json.Marshal(original)
	must(t, e)
	for _, field := range []string{"Target", "Obligation"} {
		t.Run("wrong-"+field, func(t *testing.T) {
			var s runtime.Inspection
			must(t, json.Unmarshal(raw, &s))
			in := communicationReport(s)
			editEvidenceJSON(t, communicationRow(s, "intents", "id", in.ID), "descriptor", func(v map[string]any) { v[field] = "foreign-report-context" })
			for _, h := range s.Tables["history"] {
				var intents []map[string]any
				must(t, json.Unmarshal([]byte(h["intents"].(string)), &intents))
				for _, v := range intents {
					if v["ID"] == in.ID {
						v[field] = "foreign-report-context"
						b, e := json.Marshal(intents)
						must(t, e)
						h["intents"] = string(b)
					}
				}
			}
			if hash(s) == hash(original) || communicationReport(s).ID != "" {
				t.Fatal("mutation did not exercise strict lookup miss")
			}
			tr, e := decodeCommunication(s, q, communication.J1)
			diagnostic := ""
			if e != nil {
				diagnostic = e.Error()
			}
			retain(t, "lookup-result", map[string]any{"field": field, "source_hash": hash(original), "mutated_inspection": s, "decoder_error": diagnostic, "returned_trace": tr, "independent_violations": communication.CheckTrace(tr)})
			if e == nil {
				t.Fatalf("accepted mismatched original report %s as empty prefix; messages=%d; skipped oracle violations=%v", field, len(tr.Messages), communication.CheckTrace(tr))
			}
		})
	}
}
