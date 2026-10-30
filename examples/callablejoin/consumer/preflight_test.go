package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func decode[T any](t *testing.T, value any) T {
	t.Helper()
	var out T
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		t.Fatalf("unexpected SQL JSON %T", value)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// These four construction samples exercise both answer bindings and the two
// diagnosed failures. They are not a runtime schedule qualification campaign.
func TestPublicPreflight(t *testing.T) {
	for _, tc := range []struct {
		candidate, mode, tag, status string
		charges                      int
	}{
		{"a", "answer", "answer", "succeeded", 3},
		{"b", "answer", "answer", "succeeded", 4},
		{"a", "fail", "candidate_failed", "failed", 3},
		{"b", "absent", "candidate_missing_reply", "failed", 4},
	} {
		t.Run(tc.candidate+"_"+tc.mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "preflight.db")
			args := []string{"-db", path, "-candidate", tc.candidate, "-mode", tc.mode}
			outputs := map[string]json.RawMessage{}
			for _, command := range []string{"init", "run", "replay", "inspect", "artifacts"} {
				var out bytes.Buffer
				if err := execute(t.Context(), append([]string{command}, args...), &out); err != nil {
					t.Fatal(command, err)
				}
				outputs[command] = append([]byte(nil), out.Bytes()...)
			}
			// Re-inspection retains native row values for the JSON decoder.
			s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			var outer, join runtime.Projection
			var outerProgram any
			for _, row := range s.Tables["runs"] {
				p := decode[runtime.Projection](t, row["projection"])
				if row["subject"] == "root" {
					outer = p
					outerProgram = row["program_id"]
				}
				if _, ok := p.Data["candidate_operation"]; ok {
					join = p
				}
			}
			if outer.Status != tc.status || outer.Data["decision"].Text != tc.tag || !outer.Data["join_present"].Flag || outer.Data["report_classification"].Text != tc.tag || outer.Data["report_id"].Number != 1 {
				t.Fatal("outer public policy", outer)
			}
			if join.Status != tc.status || join.Data["classification"].Text != tc.tag || join.Data["s_status"].Text != "succeeded" || !join.Data["s_present"].Flag {
				t.Fatal("joined actual classification", join)
			}
			wantReason := "member_failed"
			if tc.status == "succeeded" {
				wantReason = "member_succeeded"
			}
			if outer.Data["join_status"].Text != tc.status || outer.Data["join_reason"].Text != wantReason {
				t.Fatal("failure diagnostic changed envelope", outer)
			}
			answers, err := outer.Data["report_answers"].Records()
			if err != nil {
				t.Fatal("missing bounded answers")
			}
			wantAnswers := 0
			if tc.tag == "answer" {
				wantAnswers = 1
			}
			if len(answers) != wantAnswers {
				t.Fatal("incorrect answer cardinality", answers)
			}
			if wantAnswers == 1 && (answers[0]["id"].Number != 1 || answers[0]["value"].Number != 1) {
				t.Fatal("wrong answer", answers)
			}
			if len(s.Roots) != 1 || s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != tc.status || s.Roots[0].Outcome.Reason != wantReason {
				t.Fatal("root publication", s.Roots)
			}
			if len(s.Tables["composition_admission_decisions"]) != tc.charges || len(s.Tables["composition_publications"]) != tc.charges {
				t.Fatal("original admissions/consumptions", s.Tables["composition_admission_decisions"], s.Tables["composition_publications"])
			}
			// Tie the public verdict to an actual committed set instruction. Host
			// reconstruction alone could not satisfy this evidence check.
			var outerCode []struct {
				Op, Dst string
				A       struct{ Literal *runtime.Value }
			}
			for _, row := range s.Tables["programs"] {
				var program struct {
					Name string
					Code json.RawMessage
				}
				program = decode[struct {
					Name string
					Code json.RawMessage
				}](t, row["bytes"])
				if row["id"] == outerProgram {
					if err := json.Unmarshal(program.Code, &outerCode); err != nil {
						t.Fatal(err)
					}
				}
			}

			handle := outer.Data["join_operation"].Text
			publications, selected := 0, 0
			for _, row := range s.Tables["composition_publications"] {
				if row["operation_id"] == handle {
					out := decode[runtime.InvocationOutcome](t, row["descriptor"])
					if out.Status != tc.status || out.Reason != wantReason || !out.ResponsePresent || out.Response["classification"].Text != tc.tag {
						t.Fatal("original J publication", out)
					}
					publications++
				}
			}
			for _, row := range s.Tables["observations"] {
				obs := decode[struct {
					Kind, Operation string
					Run             runtime.RunID
				}](t, row["descriptor"])
				if obs.Run.Subject == "root" && obs.Kind == "invocation" && obs.Operation == handle && row["consumed_sequence"] != nil {
					for _, h := range s.Tables["history"] {
						if h["subject"] == "root" && h["sequence"] == row["consumed_sequence"] && h["selected_observation"] == row["id"] {
							selected++
						}
					}
				}
			}
			if publications != 1 || selected != 1 {
				t.Fatal("original handle publication/selected consumption", publications, selected)
			}
			if s.Roots[0].Charges.Invocations != int64(tc.charges) || s.Roots[0].Charges.CreatedRuns != int64(tc.charges) || s.Roots[0].Charges.Effects != 0 {
				t.Fatal("actual charges", s.Roots)
			}
			history := append([]map[string]any(nil), s.Tables["history"]...)
			sort.Slice(history, func(i, j int) bool { return history[i]["sequence"].(int64) < history[j]["sequence"].(int64) })
			previous := runtime.Projection{PC: 0}
			writes := 0
			for _, row := range history {
				if row["subject"] == "root" {
					next := decode[runtime.Projection](t, row["successor"])
					if len(outerCode) <= previous.PC {
						t.Fatal("missing actual O code")
					}
					in := outerCode[previous.PC]
					if in.Op == "set" && in.Dst == "decision" {
						writes++
						if in.A.Literal == nil || in.A.Literal.Text != tc.tag || next.Data["decision"].Text != tc.tag {
							t.Fatal("wrong committed policy", row)
						}
					}
					previous = next
				}
			}
			if writes != 1 {
				t.Fatal("outer decision not committed exactly once", writes)
			}
			if dir := os.Getenv("ALAN_CALLABLE_PREFLIGHT_DIR"); dir != "" {
				f, err := os.Create(filepath.Join(dir, "sample-"+tc.candidate+"-"+tc.mode+".json.gz"))
				if err != nil {
					t.Fatal(err)
				}
				z := gzip.NewWriter(f)
				if err := json.NewEncoder(z).Encode(outputs); err != nil {
					t.Fatal(err)
				}
				if err := z.Close(); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
