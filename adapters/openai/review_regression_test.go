//go:build integration

package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/schema"
)

const reviewCredential = "review-fake-secret"
const reviewCompleted = `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"clarify\",\"query\":\"q\"}"}]}]}`

func encodedReviewCredential(layers int) string {
	s := `"review-fake-\u0073ecret"`
	for i := 1; i < layers; i++ {
		b, _ := json.Marshal(s)
		s = string(b)
	}
	return s
}

// This oracle decodes retained data instead of checking only its raw spelling.
func credentialRecoverable(v any, depth int) bool {
	if depth > 100 {
		return false
	}
	switch x := v.(type) {
	case string:
		if strings.Contains(x, reviewCredential) {
			return true
		}
		var decoded any
		if json.Unmarshal([]byte(x), &decoded) == nil {
			if s, ok := decoded.(string); !ok || s != x {
				if credentialRecoverable(decoded, depth+1) {
					return true
				}
			}
		}
		var fragment string
		if strings.Contains(x, `\`) && json.Unmarshal([]byte(`"`+x+`"`), &fragment) == nil && fragment != x {
			return credentialRecoverable(fragment, depth+1)
		}
	case map[string]any:
		for k, value := range x {
			if credentialRecoverable(k, depth+1) || credentialRecoverable(value, depth+1) {
				return true
			}
		}
	case []any:
		for _, value := range x {
			if credentialRecoverable(value, depth+1) {
				return true
			}
		}
	}
	return false
}
func TestReviewR1NoReversibleCredentialEvidence(t *testing.T) {
	nestedKey, _ := json.Marshal(encodedReviewCredential(8))
	for _, tc := range []struct {
		name, body, header string
		status             int
	}{
		{"escaped_header", reviewCompleted, `review-fake-\u0073ecret`, 200},
		{"eight_nested_error_strings", `{"error":` + encodedReviewCredential(8) + `}`, "normal", 401},
		{"nested_object_key", `{` + string(nestedKey) + `:"error"}`, "normal", 401},
		{"escaped_response_metadata", strings.Replace(reviewCompleted, `"status":`, `"id":"review-fake-\\u0073ecret","model":"review-fake-\\u0073ecret","status":`, 1), "normal", 200},
		{"nested_usage_detail", strings.TrimSuffix(reviewCompleted, "}") + `,"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3,"extra":` + encodedReviewCredential(8) + `}}`, "normal", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("x-request-id", tc.header)
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			a, e := openai.New(openai.Config{APIKey: reviewCredential, Model: "explicit", Endpoint: server.URL, MaxOutputTokens: 20, MaxResponseBytes: 65536})
			if e != nil {
				t.Fatal(e)
			}
			p, e := a.Prepare(request(t))
			if e != nil {
				t.Fatal(e)
			}
			candidate := a.Execute(context.Background(), p)
			raw, e := json.Marshal(candidate)
			if e != nil {
				t.Fatal(e)
			}
			var retained any
			if e = json.Unmarshal(raw, &retained); e != nil {
				t.Fatal(e)
			}
			if credentialRecoverable(retained, 0) {
				t.Fatalf("credential recoverable after JSON decoding retained candidate: %s", raw)
			}
		})
	}
}
func TestReviewR2RejectsProtocolCaseAliases(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"unicode_status_alias", strings.Replace(reviewCompleted, `"status":"completed"`, `"status":"incomplete","ſtatus":"completed"`, 1)},
		{"status", strings.Replace(reviewCompleted, `"status":"completed"`, `"status":"incomplete","Status":"completed"`, 1)},
		{"error", strings.TrimSuffix(reviewCompleted, "}") + `,"error":{"message":"failed"},"Error":null}`},
		{"output", strings.Replace(reviewCompleted, `"output":`, `"output":[],"Output":`, 1)},
		{"role", strings.Replace(reviewCompleted, `"role":"assistant"`, `"role":"user","Role":"assistant"`, 1)},
		{"content", strings.Replace(reviewCompleted, `"content":`, `"content":[],"Content":`, 1)},
		{"content_type", strings.Replace(reviewCompleted, `"type":"output_text"`, `"type":"refusal","Type":"output_text"`, 1)},
		{"text", strings.Replace(reviewCompleted, `"text":`, `"text":"invalid","Text":`, 1)},
		{"usage", strings.TrimSuffix(reviewCompleted, "}") + `,"usage":{"input_tokens":1,"Input_Tokens":999,"output_tokens":2,"total_tokens":3}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, tc.body) }))
			defer s.Close()
			a, _ := openai.New(openai.Config{APIKey: reviewCredential, Model: "explicit", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 65536})
			p, e := a.Prepare(request(t))
			if e != nil {
				t.Fatal(e)
			}
			c := a.Execute(context.Background(), p)
			if c.Disposition != "protocol_invalid" {
				t.Fatalf("noncanonical alias accepted: disposition=%s response=%s", c.Disposition, c.Response)
			}
		})
	}
}
func TestReviewR2KeepsPayloadKeysCaseSensitive(t *testing.T) {
	output := `{"answer":"lower","Answer":"upper","é":"composed","e\u0301":"decomposed"}`
	body, _ := json.Marshal(map[string]any{"status": "completed", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]string{"type": "output_text", "text": output}}}}})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	a, _ := openai.New(openai.Config{APIKey: reviewCredential, Model: "explicit", Endpoint: server.URL, MaxOutputTokens: 20, MaxResponseBytes: 65536})
	r := request(t)
	r.Output, _ = (schema.Record{"answer": {Kind: "string", MaxBytes: 100}, "Answer": {Kind: "string", MaxBytes: 100}, "é": {Kind: "string", MaxBytes: 100}, "e\u0301": {Kind: "string", MaxBytes: 100}}).Retain()
	p, e := a.Prepare(r)
	if e != nil {
		t.Fatal(e)
	}
	c := a.Execute(context.Background(), p)
	if c.Disposition != "validated" || c.Payload["answer"].Text != "lower" || c.Payload["Answer"].Text != "upper" || c.Payload["é"].Text != "composed" || c.Payload["e\u0301"].Text != "decomposed" {
		t.Fatalf("distinct payload keys lost: %+v", c)
	}
}
