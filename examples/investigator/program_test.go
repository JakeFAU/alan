package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
)

// The endpoint sees only HTTP. Its proposed citation, opinion and failures go
// through the real Responses adapter and the ordinary authored parent.
func TestInvestigatorDecisions(t *testing.T) {
	cases := []struct {
		name, primary, counter, completion, assessment string
		calls, findings                                int
	}{
		{"agreement", "supports", "supports", "completed", "validated", 2, 2},
		{"opposition", "opposes", "opposes", "completed", "validated", 2, 2},
		{"disagreement", "supports", "opposes", "disagreement", "validated", 2, 2},
		{"other worker's exact quote", "supports", "other_quote", "partial", "citation_mismatch", 2, 1},
		{"blank quote", "supports", "blank_quote", "partial", "citation_mismatch", 2, 1},
		{"missing citation", "supports", "omit_quote", "partial", "model_failed", 2, 1},
		{"empty interpretation", "supports", "empty_interpretation", "partial", "invalid_interpretation", 2, 1},
		{"whitespace interpretation", "supports", "space_interpretation", "partial", "invalid_interpretation", 2, 1},
		{"punctuation interpretation", "supports", "punctuation_interpretation", "partial", "invalid_interpretation", 2, 1},
		{"punctuation followup", "supports", "punctuation_question", "partial", "invalid_question", 2, 1},
		{"insufficient response", "supports", "insufficient", "partial", "insufficient", 2, 1},
		{"known refusal", "supports", "refusal", "partial", "model_failed", 2, 1},
		{"provider incomplete", "supports", "incomplete", "partial", "model_failed", 2, 1},
		{"invalid provider JSON", "supports", "malformed", "partial", "model_failed", 2, 1},
		{"HTTP rejection", "supports", "http_error", "partial", "model_failed", 2, 1},
		{"neither resolved", "refusal", "insufficient", "incomplete", "insufficient", 2, 0},
		{"no lookup match", "supports", "no_match", "partial", "lookup_failed", 1, 1},
		{"second invocation denied", "supports", "reject_invocation", "partial", "worker_failed", 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			replies := map[string]string{"primary": tc.primary, "counter": tc.counter}
			var mu sync.Mutex
			var contacts []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct{ Input []struct{ Content string } }
				var input struct{ Role, Snippet, Document, Question string }
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Input) != 1 {
					t.Error("invalid provider input")
					return
				}
				if err := json.Unmarshal([]byte(request.Input[0].Content), &input); err != nil || input.Document != "notes" || input.Question != "Does the plan hold?" {
					t.Error("invalid retained model evidence", input)
					return
				}
				mu.Lock()
				contacts = append(contacts, input.Role)
				mu.Unlock()
				result := replies[input.Role]
				output := map[string]string{"verdict": "supports", "quote": input.Snippet, "interpretation": "One interpretation", "unresolved": ""}
				part := map[string]string{"type": "output_text"}
				status := "completed"
				switch result {
				case "supports", "opposes", "insufficient":
					output["verdict"] = result
				case "other_quote":
					output["quote"] = "alpha — first source"
				case "blank_quote":
					output["quote"] = " "
				case "omit_quote":
					delete(output, "quote")
				case "empty_interpretation":
					output["interpretation"] = ""
				case "space_interpretation":
					output["interpretation"] = " \t\u2003"
				case "punctuation_interpretation":
					output["interpretation"] = "?!…"
				case "punctuation_question":
					output["unresolved"] = "!!!"
				case "refusal":
					part = map[string]string{"type": "refusal", "refusal": "declined"}
				case "incomplete":
					status = "incomplete"
				case "malformed":
					_, _ = w.Write([]byte("{"))
					return
				case "http_error":
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				default:
					t.Errorf("unexpected provider call for %q", result)
					return
				}
				if result == "insufficient" {
					output["unresolved"] = "Is there independent confirmation?"
				}
				if part["type"] == "output_text" {
					encoded, _ := json.Marshal(output)
					part["text"] = string(encoded)
				}
				err := json.NewEncoder(w).Encode(map[string]any{
					"id": "resp-" + input.Role, "model": "controlled", "status": status,
					"usage":  map[string]any{"input_tokens": 17, "output_tokens": 7, "total_tokens": 24},
					"output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{part}}},
				})
				if err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			queries := [2]string{"alpha", "beta"}
			bounds := runtime.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 1, MaxEffects: 4, MaxInFlight: 2}
			if tc.counter == "no_match" {
				queries[1] = "absent"
			}
			if tc.counter == "reject_invocation" {
				bounds.MaxInvocations = 1
			}
			p, err := investigatorProvision("Does the plan hold?", queries, "{}", bounds)
			if err != nil {
				t.Fatal(err)
			}
			asset, err := effects.NewAsset("notes", "test", []byte("alpha — first source\nbeta — second source"))
			if err != nil {
				t.Fatal(err)
			}
			p.Assets = []effects.Asset{asset}
			path := filepath.Join(t.TempDir(), "investigation.sqlite")
			store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			model, err := openai.New(openai.Config{Endpoint: server.URL, APIKey: "controlled-secret", Model: "controlled", MaxOutputTokens: 100, MaxResponseBytes: 8192})
			if err != nil {
				t.Fatal(err)
			}
			r, err := store.Run(t.Context(), runtime.RunOptions{Adapters: map[string]effects.Adapter{"corpus": lookup.New(), "model": proposalAdapter{model}}, MaxSteps: 300, MaxDispatch: 100, MaxInFlight: 2})
			if err != nil {
				t.Fatal(err)
			}
			state, err := store.Snapshot(t.Context(), runtime.RunID{Subject: "investigator", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			data := state.Projection.Data
			if len(r.OpenRuns) != 0 || data["completion"].Text != tc.completion || data["counter_assessment"].Text != tc.assessment {
				t.Fatalf("parent %s, counter assessment %s, open %v; counter outcome %v", data["completion"].Text, data["counter_assessment"].Text, r.OpenRuns, data["counter_status"])
			}
			cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			answer, err := reportSnapshot(cut)
			if err != nil || answer.Status != tc.completion || len(answer.Findings) != tc.findings {
				t.Fatalf("typed answer %q, %d findings, %v", answer.Status, len(answer.Findings), err)
			}
			if tc.counter == "refusal" && (data["counter_status"].Text != "failed" || data["counter_model_disposition"].Text != "refusal") {
				t.Fatalf("worker did not retain failed outcome and provider refusal: %v", data)
			}
			if tc.counter == "reject_invocation" && (data["counter_status"].Text != "rejected" || data["counter_present"].Flag) {
				t.Fatal("rejected invocation gained a response")
			}
			if tc.counter == "punctuation_question" {
				view, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				var rejected runtime.AdapterResult
				for _, row := range view.Tables["adapter_results"] {
					if row["intent_id"] == data["counter_model_id"].Text {
						if err := json.Unmarshal([]byte(row["descriptor"].(string)), &rejected); err != nil {
							t.Fatal(err)
						}
					}
				}
				c := rejected.Candidate
				if rejected.Disposition != invalidQuestion || rejected.Values["valid"].Flag || c.ResponseID != "resp-counter" ||
					c.Payload["unresolved"].Text != "!!!" || !strings.Contains(c.Response, "!!!") || c.Usage == nil || c.Usage.TotalTokens == nil || *c.Usage.TotalTokens != 24 {
					t.Fatalf("lost original provider evidence on local rejection: %+v", rejected)
				}
			}
			findings, err := data["findings"].Records()
			if err != nil || len(findings) != tc.findings {
				t.Fatalf("findings %v, %v", findings, err)
			}
			for _, f := range findings {
				role := f["role"].Text
				start, end := f["start"].Number, f["end"].Number
				if !data[role+"_accepted"].Flag || f["invocation_operation_id"].Text != data[role+"_call"].Text ||
					f["version"].Text != asset.Version || start < 0 || end <= start || end > int64(len(asset.Content)) || f["quote"].Text != asset.Content[start:end] {
					t.Errorf("unsupported finding %v", f)
				}
			}
			gaps, err := data["gaps"].Records()
			if err != nil || (len(gaps) == 0) != (tc.completion == "completed") {
				t.Fatalf("gaps %v %v", gaps, err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(contacts) != tc.calls || !slices.Contains(contacts, "primary") || slices.Contains(contacts, "counter") != (tc.calls == 2) {
				t.Fatalf("physical provider contacts: %v", contacts)
			}
		})
	}
}
