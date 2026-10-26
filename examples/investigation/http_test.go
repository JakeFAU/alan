package investigation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// The controlled HTTP service sees the real public schemas and all retained
// evidence through the actual OpenAI adapter, including the revised candidate.
func TestHTTPReviewRevision(t *testing.T) {
	provider := &controlled{mode: "revise"}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request struct {
			Input []struct{ Content string }
			Text  struct {
				Format struct {
					Schema struct{ Properties map[string]json.RawMessage }
				}
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if len(request.Input) != 1 {
			t.Error("missing structured input")
			w.WriteHeader(400)
			return
		}
		var native map[string]any
		if err := json.Unmarshal([]byte(request.Input[0].Content), &native); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		payload := map[string]schema.Value{}
		for k, v := range native {
			switch value := v.(type) {
			case string:
				payload[k] = schema.Text(value)
			case float64:
				payload[k] = schema.Int(int64(value))
			case bool:
				payload[k] = schema.Bool(value)
			}
		}
		properties := request.Text.Format.Schema.Properties
		effect := ""
		switch {
		case properties["query"] != nil:
			effect = "plan"
		case properties["document"] != nil:
			effect = "synthesize"
		case properties["candidate"] != nil:
			effect = "compose"
		case properties["reason"] != nil:
			effect = "review"
		case properties["subquestion1"] != nil:
			effect = "decompose"
		default:
			t.Error("unexpected output contract")
			w.WriteHeader(400)
			return
		}
		result := provider.Execute(r.Context(), effects.Prepared{Request: effects.Request{Effect: effect, Payload: payload}})
		output := map[string]any{}
		for k, v := range result.Payload {
			switch v.Kind {
			case "string":
				output[k] = v.Text
			case "int":
				output[k] = v.Number
			case "bool":
				output[k] = v.Flag
			}
		}
		text, _ := json.Marshal(output)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "completed", "model": "controlled-returned", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]string{"type": "output_text", "text": string(text)}}}}})
	}))
	defer server.Close()
	c := investigation.DefaultConfig()
	c.Model = "controlled"
	c.Endpoint = server.URL
	c.Nested = true
	a, _ := effects.NewAsset("packet", "fixture", []byte("alpha release supports Linux.\nbeta rollback requires version 2.\ngamma approval is recorded.\ndelta window is seven minutes.\n"))
	p, err := investigation.Provision("question", c, []effects.Asset{a})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "http.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = investigation.Run(t.Context(), path, "controlled-secret"); err != nil {
		t.Fatal(err)
	}
	v, err := investigation.Inspect(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 13 || v.ModelRequests != 13 || v.LookupRequests != 4 || v.Result["revision_count"].Number != 1 || v.Result["answer"].Text != "Linux supported; rollback version 2; approval recorded; seven minutes." {
		t.Fatalf("HTTP workflow failed: calls=%d result=%v", calls.Load(), v.Result)
	}
	for _, result := range v.Results {
		if !result.Values["valid"].Flag {
			t.Fatalf("invalid result: %+v", result)
		}
	}
	server.Close()
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	t.Logf("INVESTIGATION {\"case\":\"http-review-revision\",\"contacts\":%d,\"model_requests\":%d,\"lookup_requests\":%d}", calls.Load(), v.ModelRequests, v.LookupRequests)
}
