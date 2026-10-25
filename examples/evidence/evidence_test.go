//go:build integration

package evidence_test

import (
	"encoding/json"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// Removing any ordinary validation branch must change the public result or contact count.
func TestControlledApplication(t *testing.T) {
	for _, tc := range []struct {
		name, decision, tool, query, special, want string
		model, lookup, calls                       int
	}{
		{"answer", "lookup", "lookup", "WAL", "", "answer", 3, 2, 2},
		{"clarify", "clarification", "", "", "", "clarification", 3, 2, 1},
		{"insufficient", "insufficient_evidence", "", "", "", "insufficient_evidence", 3, 2, 1},
		{"wrong-tool", "lookup", "shell", "WAL", "", "invalid_result", 3, 2, 1},
		{"endpoint", "lookup", "http://127.0.0.1/evil", "WAL", "", "invalid_result", 3, 2, 1},
		{"no-match", "lookup", "lookup", "zzzzunknown", "", "no_match", 3, 2, 1},
		{"wrong-document", "lookup", "lookup", "WAL", "document", "invalid_result", 3, 2, 2},
		{"wrong-start", "lookup", "lookup", "WAL", "start", "invalid_result", 3, 2, 2},
		{"wrong-end", "lookup", "lookup", "WAL", "end", "invalid_result", 3, 2, 2},
		{"wrong-version", "lookup", "lookup", "WAL", "version", "invalid_result", 3, 2, 2},
		{"wrong-snippet", "lookup", "lookup", "WAL", "snippet", "invalid_result", 3, 2, 2},
		{"empty-answer", "lookup", "lookup", "WAL", "answer", "invalid_result", 3, 2, 2},
		{"invalid", "lookup", "lookup", "WAL", "invalid", "invalid_result", 3, 2, 1},
		{"refusal", "lookup", "lookup", "WAL", "refusal", "invalid_result", 3, 2, 1},
		{"incomplete", "lookup", "lookup", "WAL", "incomplete", "invalid_result", 3, 2, 1},
		{"model-budget", "lookup", "lookup", "WAL", "", "budget_exhausted", 1, 2, 1},
		{"zero-model-budget", "lookup", "lookup", "WAL", "", "budget_exhausted", 0, 2, 0},
		{"lookup-budget", "lookup", "lookup", "WAL", "", "budget_exhausted", 3, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var contacts atomic.Int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := contacts.Add(1)
				var body struct {
					Model string
					Input []struct{ Content string }
					Text  json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Model != "controlled-model" || len(body.Input) != 1 {
					t.Error("missing explicit model/input")
				}
				payload := map[string]any{"decision": tc.decision, "tool": tc.tool, "query": tc.query, "message": "Please clarify the question."}
				if n == 2 {
					var in map[string]any
					if err := json.Unmarshal([]byte(body.Input[0].Content), &in); err != nil {
						t.Error(err)
					}
					if in["question"] != "How does WAL work?" || in["snippet"] == nil {
						t.Error("synthesis omitted question/evidence")
					}
					payload = map[string]any{"answer": "WAL records changes before updating the main database.", "document": in["document"], "version": in["version"], "snippet": in["snippet"], "start": in["start"], "end": in["end"]}
					if tc.special == "start" || tc.special == "end" {
						payload[tc.special] = -1
					} else if tc.special == "answer" {
						payload["answer"] = ""
					} else if tc.special != "" {
						payload[tc.special] = "wrong"
					}
				}
				text, _ := json.Marshal(payload)
				if tc.special == "invalid" {
					text = []byte(`{"decision":3}`)
				}
				status := "completed"
				content := map[string]any{"type": "output_text", "text": string(text)}
				if tc.special == "incomplete" {
					status = "incomplete"
				}
				if tc.special == "refusal" {
					content = map[string]any{"type": "refusal", "refusal": "controlled refusal"}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": "controlled-response", "model": "controlled-returned", "status": status, "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{content}}}, "usage": map[string]any{"input_tokens": 20, "output_tokens": 10, "total_tokens": 30}})
			}))
			defer srv.Close()
			c := evidence.DefaultConfig()
			c.Model = "controlled-model"
			c.Endpoint = srv.URL
			c.MaxModelRequests = tc.model
			c.MaxLookupRequests = tc.lookup
			p, err := evidence.Provision("How does WAL work?", c, nil)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "app.sqlite")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			report, err := evidence.Run(t.Context(), path, "controlled-secret")
			if err != nil {
				t.Fatal(err)
			}
			if len(report.UnknownAdapters) != 0 {
				t.Fatalf("unknown: %+v", report)
			}
			view, err := evidence.Inspect(t.Context(), path)
			if err != nil {
				t.Fatal(err)
			}
			if view.Result["disposition"].Text != tc.want {
				t.Fatalf("result=%+v want %s", view.Result, tc.want)
			}
			if contacts.Load() != int64(tc.calls) {
				t.Fatalf("contacts=%d want%d", contacts.Load(), tc.calls)
			}
			if tc.want == "answer" && (view.Result["answer"].Text != "WAL records changes before updating the main database." || view.ModelRequests != 2 || view.LookupRequests != 1) {
				t.Fatalf("answer/accounting: %+v", view)
			}
			expectedLookup := 1
			if tc.calls == 0 || tc.calls == 1 && tc.name != "no-match" && tc.name != "model-budget" {
				expectedLookup = 0
			}
			if view.ModelRequests != tc.calls || view.LookupRequests != expectedLookup || view.Attempts != tc.calls+expectedLookup {
				t.Fatalf("logical/attempt accounting: model=%d lookup=%d attempts=%d", view.ModelRequests, view.LookupRequests, view.Attempts)
			}
			for _, result := range view.Results {
				if result.Candidate.Usage != nil {
					usage := result.Candidate.Usage
					if usage.InputTokens == nil || *usage.InputTokens != 20 || usage.OutputTokens == nil || *usage.OutputTokens != 10 || usage.TotalTokens == nil || *usage.TotalTokens != 30 {
						t.Fatalf("usage changed: %+v", usage)
					}
				}
			}
			if tc.want == "answer" {
				if view.Result["start"].Kind != "int" || view.Result["start"].Number != 0 || view.Result["end"].Number != 82 {
					t.Fatalf("wrong span: %+v", view.Result)
				}
				encoded, _ := json.Marshal(view.Result)
				t.Logf("controlled public result: %s", encoded)
			}
			before := contacts.Load()
			srv.Close()
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if contacts.Load() != before {
				t.Fatal("replay contacted provider")
			}
			t.Logf("controlled result=%s answer=%q model=%d lookup=%d attempts=%d", view.Result["disposition"].Text, view.Result["answer"].Text, view.ModelRequests, view.LookupRequests, view.Attempts)
		})
	}
}

// A dropped response remains unknown across a physical close/open; running again
// must not reset spent counters or authorize a second possibly billable call.
func TestUnknownRetainsPendingIdentityAndBudget(t *testing.T) {
	var contacts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	}))
	defer srv.Close()
	c := evidence.DefaultConfig()
	c.Model = "controlled-model"
	c.Endpoint = srv.URL
	p, err := evidence.Provision("How does WAL work?", c, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "unknown.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		report, err := evidence.Run(t.Context(), path, "controlled-secret")
		if err != nil {
			t.Fatal(err)
		}
		if len(report.UnknownAdapters) != 1 {
			t.Fatalf("unknown=%+v", report)
		}
		v, err := evidence.Inspect(t.Context(), path)
		if err != nil {
			t.Fatal(err)
		}
		if v.ModelRequests != 1 || v.Attempts != 1 || len(v.Unknown) != 1 || v.Result != nil || len(v.Results) != 0 {
			t.Fatalf("unknown accounting=%+v", v)
		}
		found := false
		for _, run := range v.Runs {
			if spent, ok := run.Projection.Data["model_spent"]; ok {
				found = true
				if spent.Number != 1 || run.Projection.Data["pending_plan"].Text != v.Unknown[0] {
					t.Fatalf("lost spent/pending state: %+v", run)
				}
			}
		}
		if !found {
			t.Fatal("missing ordinary budget")
		}
	}
	if contacts.Load() != 1 {
		t.Fatalf("repeated unknown model request: %d", contacts.Load())
	}
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestDeniedCapabilityIsNotCountedAsTransportAttempt(t *testing.T) {
	var contacts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := evidence.DefaultConfig()
	c.Model = "controlled-model"
	c.Endpoint = srv.URL
	p, err := evidence.Provision("How does WAL work?", c, nil)
	if err != nil {
		t.Fatal(err)
	}
	permissions := p.Permissions[:0]
	for _, permission := range p.Permissions {
		if permission.Target != evidence.ModelCapability {
			permissions = append(permissions, permission)
		}
	}
	p.Permissions = permissions
	path := filepath.Join(t.TempDir(), "denied.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = evidence.Run(t.Context(), path, "controlled-secret"); err != nil {
		t.Fatal(err)
	}
	v, err := evidence.Inspect(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if v.ModelRequests != 1 || v.Attempts != 0 || v.DeniedAttempts != 1 || contacts.Load() != 0 || v.Result["disposition"].Text != "invalid_result" {
		t.Fatalf("denied request counted as executed: logical=%d attempts=%d contacts=%d result=%v", v.ModelRequests, v.Attempts, contacts.Load(), v.Result)
	}
}
