//go:build integration

package adaptive_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/schema"
)

// Exercise the unchanged helper through the real bounded Responses adapter.
// The provider returns only helper observations; retained steps derive work.
func TestControlledEvidenceAdaptiveTransport(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Input []struct{ Content string } }
		if e := json.NewDecoder(r.Body).Decode(&q); e != nil || len(q.Input) != 1 {
			http.Error(w, "invalid request", 400)
			return
		}
		var payload map[string]any
		if e := json.Unmarshal([]byte(q.Input[0].Content), &payload); e != nil {
			http.Error(w, "invalid payload", 400)
			return
		}
		count++
		result := map[string]any{"decision": "clarification", "tool": "", "query": "", "message": "Clarify WAL"}
		if count > 1 {
			result = map[string]any{"decision": "lookup", "tool": "lookup", "query": "WAL", "message": ""}
		}
		if _, synthesis := payload["snippet"]; synthesis {
			result = map[string]any{"answer": "WAL records changes."}
			for _, k := range []string{"document", "version", "snippet", "start", "end"} {
				result[k] = payload[k]
			}
		}
		raw, _ := json.Marshal(result)
		json.NewEncoder(w).Encode(map[string]any{"status": "completed", "model": "controlled", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(raw)}}}}})
	}))
	defer server.Close()
	model, e := openai.New(openai.Config{APIKey: "phase1h-controlled-only", Model: "controlled", Endpoint: server.URL, MaxOutputTokens: 1024, MaxResponseBytes: 65536})
	if e != nil {
		t.Fatal(e)
	}
	lookup := adapter{func(effects.Request) effects.Candidate {
		return effects.Candidate{Disposition: "validated", Payload: map[string]schema.Value{"document": schema.Text("durability"), "version": schema.Text("v1"), "snippet": schema.Text("WAL records changes."), "start": schema.Int(0), "end": schema.Int(20)}}
	}}
	x := execute(t, helper(t), true, 2, questions(1, "How does the retained WAL record relate to database changes?"), map[string]effects.Adapter{evidence.ModelCapability: model, evidence.LookupCapability: lookup})
	assertStop(t, x, "exhausted", 2, 2, []int64{})
	if x.report.Roots[0].Charges.Effects != 4 || count != 3 {
		t.Fatal("transport reservations", x.report.Roots, count)
	}
}
