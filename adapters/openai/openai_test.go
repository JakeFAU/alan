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
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

func request(t *testing.T) effects.Request {
	t.Helper()
	in, _ := (schema.Record{"instructions": {Kind: "string", MaxBytes: 1000}, "input": {Kind: "string", MaxBytes: 1000}}).Retain()
	out, _ := (schema.Record{"kind": {Kind: "string", MaxBytes: 30, Enum: []string{"lookup", "clarify"}}, "query": {Kind: "string", MaxBytes: 100}}).Retain()
	return effects.Request{ID: "local", Key: "global-key", Capability: "model", Effect: "plan", Input: in, Output: out, Payload: map[string]schema.Value{"instructions": schema.Text("retained instructions"), "input": schema.Text("complete question")}}
}
func TestResponsesProtocol(t *testing.T) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Error(err)
		}
		if body["model"] != "explicit-model" || body["instructions"] != "retained instructions" || body["store"] != false || body["truncation"] != "disabled" || body["max_output_tokens"] != float64(512) {
			t.Errorf("wire %s", raw)
		}
		input := body["input"].([]any)
		if len(input) != 1 || input[0].(map[string]any)["role"] != "user" || input[0].(map[string]any)["content"] != "complete question" {
			t.Errorf("incomplete input: %s", raw)
		}
		for _, field := range []string{"tools", "previous_response_id", "conversation", "background"} {
			if _, ok := body[field]; ok {
				t.Errorf("unexpected %s", field)
			}
		}
		if r.Header.Get("Authorization") != "Bearer runtime-secret" || r.Header.Get("X-Client-Request-Id") == "" {
			t.Error("missing runtime authorization/correlation")
		}
		f := body["text"].(map[string]any)["format"].(map[string]any)
		outputSchema := f["schema"].(map[string]any)
		properties := outputSchema["properties"].(map[string]any)
		if outputSchema["additionalProperties"] != false || len(outputSchema["required"].([]any)) != 2 || len(properties) != 2 || properties["valid"] != nil || properties["disposition"] != nil {
			t.Error("incorrect closed payload schema")
		}
		if f["strict"] != true || f["type"] != "json_schema" {
			t.Errorf("format %v", f)
		}
		w.Header().Set("x-request-id", "http-id")
		io.WriteString(w, `{"id":"response-id","model":"returned-model","status":"completed","output":[{"type":"reasoning"},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"lookup\",\"query\":\"durability\"}"}]}],"usage":{"input_tokens":12,"output_tokens":8,"total_tokens":20}}`)
	}))
	defer server.Close()
	a, err := openai.New(openai.Config{APIKey: "runtime-secret", Model: "explicit-model", Endpoint: server.URL, MaxOutputTokens: 512, MaxResponseBytes: 8192})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Prepare(request(t))
	if err != nil {
		t.Fatal(err)
	}
	c := a.Execute(context.Background(), p)
	if c.Disposition != "validated" || c.Payload["kind"].Text != "lookup" || c.ResponseID != "response-id" || c.RequestID != "http-id" || c.ReturnedModel != "returned-model" || c.Usage == nil || c.Usage.TotalTokens == nil || *c.Usage.TotalTokens != 20 {
		t.Fatalf("candidate %+v", c)
	}
	b, _ := json.Marshal(struct {
		P effects.Prepared
		C effects.Candidate
	}{p, c})
	if strings.Contains(string(b), "runtime-secret") {
		t.Fatal("credential retained")
	}
	if contacts.Load() != 1 {
		t.Fatal("hidden retry")
	}
}
func TestResponsesDispositionsAndCredentialExclusion(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"refusal", `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"refusal","refusal":"no"}]}]}`, "refusal", 200},
		{"incomplete", `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`, "incomplete", 200},
		{"invalid schema", `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"shell\",\"query\":\"x\"}"}]}]}`, "schema_invalid", 200},
		{"tool call", `{"status":"completed","output":[{"type":"function_call","name":"shell","arguments":"{}"}]}`, "protocol_invalid", 200},
		{"failed", `{"status":"failed","error":{"message":"runtime-secret"}}`, "provider_failed", 200},
		{"http rejection", `{"error":{"message":"runtime-secret"}}`, "http_rejected", 401},
		{"http bad request", `{"error":{"message":"bad schema"}}`, "http_rejected", 400},
		{"http rate limit", `{"error":{"message":"rate limited"}}`, "http_rejected", 429},
		{"http server error", `{"error":{"message":"server error"}}`, "http_rejected", 500},
		{"content filter", `{"status":"incomplete","incomplete_details":{"reason":"content_filter"}}`, "incomplete", 200},
		{"ambiguous output", `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{}"},{"type":"output_text","text":"{}"}]}]}`, "protocol_invalid", 200},
		{"redirect", `runtime-secret`, "http_rejected", 302},
		{"malformed", `{`, "protocol_invalid", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var n atomic.Int64
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n.Add(1)
				w.Header().Set("x-request-id", "runtime-secret")
				w.Header().Set("Location", "/again")
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer s.Close()
			a, e := openai.New(openai.Config{APIKey: "runtime-secret", Model: "chosen", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
			if e != nil {
				t.Fatal(e)
			}
			p, e := a.Prepare(request(t))
			if e != nil {
				t.Fatal(e)
			}
			c := a.Execute(context.Background(), p)
			if c.Disposition != tc.want {
				t.Fatalf("got %s want %s", c.Disposition, tc.want)
			}
			raw, _ := json.Marshal(c)
			if strings.Contains(string(raw), "runtime-secret") {
				t.Fatal("credential leaked")
			}
			if n.Load() != 1 {
				t.Fatal("retry/redirect")
			}
		})
	}
}
func TestMissingConfigurationBeforePrepare(t *testing.T) {
	if _, e := openai.New(openai.Config{Model: "explicit"}); e == nil {
		t.Fatal("missing credential accepted")
	}
}
func TestEscapedCredentialNeverRetained(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"runtime-\u0073ecret","model":"runtime-\u0073ecret","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"clarify\",\"query\":\"runtime-\\u0073ecret\"}"}]}]}`)
	}))
	defer s.Close()
	a, _ := openai.New(openai.Config{APIKey: "runtime-secret", Model: "chosen", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
	p, e := a.Prepare(request(t))
	if e != nil {
		t.Fatal(e)
	}
	c := a.Execute(context.Background(), p)
	raw, _ := json.Marshal(c)
	if strings.Contains(string(raw), "runtime-secret") || c.Disposition == "validated" {
		t.Fatalf("escaped credential accepted: %s", raw)
	}
}
func TestDuplicateOutputKeysRejected(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"shell\",\"kind\":\"lookup\",\"query\":\"x\"}"}]}]}`)
	}))
	defer s.Close()
	a, _ := openai.New(openai.Config{APIKey: "runtime-secret", Model: "chosen", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
	p, _ := a.Prepare(request(t))
	if c := a.Execute(context.Background(), p); c.Disposition != "schema_invalid" {
		t.Fatalf("duplicate output keys accepted: %+v", c)
	}
}
func TestEscapedCredentialRemovedFromRawEvidence(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		io.WriteString(w, `{"error":"runtime-\u0073ecret"}`)
	}))
	defer s.Close()
	a, _ := openai.New(openai.Config{APIKey: "runtime-secret", Model: "chosen", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
	p, _ := a.Prepare(request(t))
	c := a.Execute(context.Background(), p)
	var evidence map[string]string
	if e := json.Unmarshal([]byte(c.Response), &evidence); e != nil {
		t.Fatal(e)
	}
	if evidence["error"] == "runtime-secret" {
		t.Fatal("reversible escaped credential retained in raw evidence")
	}
}
func TestUsageAndMalformedEscapedCredentialExclusion(t *testing.T) {
	for _, body := range []string{`{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"clarify\",\"query\":\"q\"}"}]}],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3,"extra":"runtime-\u0073ecret"}}`, `{"error":"runtime-\u0073ecret",`} {
		t.Run(body[:10], func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
			defer s.Close()
			a, _ := openai.New(openai.Config{APIKey: "runtime-secret", Model: "chosen", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
			p, _ := a.Prepare(request(t))
			c := a.Execute(context.Background(), p)
			raw, _ := json.Marshal(c)
			if strings.Contains(string(raw), "runtime-secret") || strings.Contains(c.Response, `runtime-\u0073ecret`) {
				t.Fatalf("credential evidence retained %s", raw)
			}
		})
	}
}
func TestOversizeMetadataIsKnownInvalid(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": strings.Repeat("r", 600), "model": "actual", "status": "completed", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]string{"type": "output_text", "text": `{"kind":"clarify","query":"q"}`}}}}})
	}))
	defer s.Close()
	a, _ := openai.New(openai.Config{APIKey: "runtime-secret", Model: "chosen", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
	p, _ := a.Prepare(request(t))
	c := a.Execute(context.Background(), p)
	if c.Disposition != "protocol_invalid" || len(c.ResponseID) > 512 {
		t.Fatal("metadata bound not classified", c.Disposition, len(c.ResponseID))
	}
}
func TestResponsesSerializesCompleteNativeRecord(t *testing.T) {
	var contacts atomic.Int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		var body struct {
			Instructions string                           `json:"instructions"`
			Input        []struct{ Role, Content string } `json:"input"`
		}
		if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
			t.Error(e)
		}
		want := `{"document":"doc","end":38,"question":"Where are durable records?","snippet":"SQLite retains durable records.","start":7,"verified":true,"version":"retained-version"}`
		if body.Instructions != "Use this retained record" || len(body.Input) != 1 || body.Input[0].Role != "user" || body.Input[0].Content != want {
			t.Errorf("model input does not retain exact native record: %+v", body)
		}
		io.WriteString(w, `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"kind\":\"clarify\",\"query\":\"q\"}"}]}]}`)
	}))
	defer s.Close()
	a, _ := openai.New(openai.Config{APIKey: "runtime-secret", Model: "explicit", Endpoint: s.URL, MaxOutputTokens: 20, MaxResponseBytes: 4096})
	r := request(t)
	r.Payload = map[string]schema.Value{"instructions": schema.Text("Use this retained record"), "question": schema.Text("Where are durable records?"), "document": schema.Text("doc"), "version": schema.Text("retained-version"), "snippet": schema.Text("SQLite retains durable records."), "start": schema.Int(7), "end": schema.Int(38), "verified": schema.Bool(true)}
	fields := schema.Record{}
	for k, v := range r.Payload {
		f := schema.Field{Kind: v.Kind}
		if v.Kind == "string" {
			f.MaxBytes = 1000
		}
		fields[k] = f
	}
	r.Input, _ = fields.Retain()
	p, e := a.Prepare(r)
	if e != nil {
		t.Fatal("record preparation rejected", e)
	}
	if c := a.Execute(context.Background(), p); c.Disposition != "validated" || contacts.Load() != 1 {
		t.Fatal(c, contacts.Load())
	}
}
