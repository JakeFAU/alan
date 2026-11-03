//go:build integration

package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/schema"
)

// These exact bodies are the original FR1 counterexamples, not reconstructed JSON.
const finalReviewIncomplete = `{"id":"probe-id","model":"fixture","status":"completed","output":[{"type":"message","role":"assistant","status":"incomplete","content":[{"type":"output_text","text":"{\"answer\":\"candidate\"}"}]}]}`
const finalReviewInProgress = `{"id":"probe-id","model":"fixture","status":"completed","output":[{"type":"message","role":"assistant","status":"in_progress","content":[{"type":"output_text","text":"{\"answer\":\"candidate\"}"}]}]}`

// Ignoring present output-item status must fail through the actual HTTP parser.
func TestFinalReviewOutputItemStatus(t *testing.T) {
	completed := strings.Replace(finalReviewIncomplete, `"status":"incomplete"`, `"status":"completed"`, 1)
	absent := strings.Replace(finalReviewIncomplete, `"status":"incomplete",`, "", 1)
	cases := []struct{ name, body, want string }{
		{"original_incomplete", finalReviewIncomplete, "protocol_invalid"},
		{"original_in_progress", finalReviewInProgress, "protocol_invalid"},
		{"completed_message", completed, "validated"},
		{"absent_message", absent, "validated"},
	}
	statuses := []struct{ name, field, want string }{
		{"absent", "", "validated"},
		{"completed", `,"status":"completed"`, "validated"},
		{"incomplete", `,"status":"incomplete"`, "protocol_invalid"},
		{"in_progress", `,"status":"in_progress"`, "protocol_invalid"},
		{"null", `,"status":null`, "protocol_invalid"},
		{"empty", `,"status":""`, "protocol_invalid"},
		{"unknown", `,"status":"future"`, "protocol_invalid"},
		{"number", `,"status":1`, "protocol_invalid"},
		{"boolean", `,"status":true`, "protocol_invalid"},
		{"array", `,"status":[]`, "protocol_invalid"},
		{"object", `,"status":{}`, "protocol_invalid"},
		{"case_value", `,"status":"Completed"`, "protocol_invalid"},
		{"case_alias", `,"Status":"completed"`, "protocol_invalid"},
		{"unicode_alias", `,"ſtatus":"completed"`, "protocol_invalid"},
		{"case_conflict", `,"status":"incomplete","Status":"completed"`, "protocol_invalid"},
		{"unicode_conflict", `,"status":"incomplete","ſtatus":"completed"`, "protocol_invalid"},
		{"reverse_conflict", `,"Status":"completed","status":"incomplete"`, "protocol_invalid"},
		{"duplicate", `,"status":"incomplete","status":"completed"`, "protocol_invalid"},
	}
	for _, itemType := range []string{"message", "reasoning"} {
		for _, status := range statuses {
			body := strings.Replace(absent, `"role":"assistant"`, `"role":"assistant"`+status.field, 1)
			if itemType == "reasoning" {
				body = strings.Replace(completed, `"output":[`, `"output":[{"type":"reasoning"`+status.field+`},`, 1)
			}
			cases = append(cases, struct{ name, body, want string }{itemType + "_" + status.name, body, status.want})
		}
	}
	cases = append(cases, struct{ name, body, want string }{
		"both_absent", strings.Replace(absent, `"output":[`, `"output":[{"type":"reasoning"},`, 1), "validated",
	})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var contacts atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contacts.Add(1)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			adapter, err := openai.New(openai.Config{
				APIKey: "synthetic-final-review-credential", Model: "fixture", Endpoint: server.URL,
				MaxOutputTokens: 128, MaxResponseBytes: 8192,
			})
			if err != nil {
				t.Fatal(err)
			}
			req := request(t)
			req.Output, err = (schema.Record{"answer": {Kind: "string", MaxBytes: 100}}).Retain()
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := adapter.Prepare(req)
			if err != nil {
				t.Fatal(err)
			}
			candidate := adapter.Execute(context.Background(), prepared)
			encoded, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("input=%s candidate=%s contacts=%d", tc.body, encoded, contacts.Load())
			if candidate.Disposition != tc.want {
				t.Errorf("disposition=%s want %s", candidate.Disposition, tc.want)
			}
			wantResponse := tc.body
			if strings.HasSuffix(tc.name, "_duplicate") {
				// The existing sanitizer omits ambiguous duplicate-key evidence.
				wantResponse = ""
			}
			if candidate.Response != wantResponse {
				t.Error("bounded evidence differs from the sanitizer contract")
			}
			if contacts.Load() != 1 {
				t.Errorf("contacts=%d want 1", contacts.Load())
			}
			if tc.want == "validated" {
				if candidate.Payload["answer"].Text != "candidate" {
					t.Errorf("lost complete payload: %+v", candidate.Payload)
				}
			} else if candidate.Payload != nil {
				t.Errorf("invalid item published payload: %+v", candidate.Payload)
			}
		})
	}
}
