package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const caseQuestion = "Did the bridge open on schedule?"
const permitText = "Résumé of the hearing.\nAlpha permit records the bridge opened in May.\n"
const auditText = "Audit index π.\nBeta inspection reports the bridge remained closed in May.\n"

type sourceFact struct{ document, content, query, quote string }

var caseFacts = map[string]sourceFact{
	"primary": {"permit", permitText, "Alpha", "Alpha permit records the bridge opened in May."},
	"counter": {"audit", auditText, "Beta", "Beta inspection reports the bridge remained closed in May."},
}

type providerContact struct {
	role, clientID, wire, mode, response, requestID string
	status                                          int
}

// The endpoint owns its contact log and its gates. It has neither an ALAN
// adapter nor a store handle. Only real, authenticated HTTP can enter the log.
type responsesFixture struct {
	server  *httptest.Server
	mu      sync.Mutex
	log     []providerContact
	arrived chan providerContact
	gates   map[string]chan struct{}
}

func responsesServer(t *testing.T, counterMode string, held ...string) *responsesFixture {
	t.Helper()
	p := &responsesFixture{arrived: make(chan providerContact, 16), gates: map[string]chan struct{}{}}
	for _, role := range held {
		p.gates[role] = make(chan struct{})
	}
	p.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.RequestURI() != "/v1/responses" || r.Header.Get("Authorization") != "Bearer alan-controlled-credential" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("invalid method, route or controlled authorization")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<18))
		if err != nil {
			t.Error(err)
			return
		}
		var req struct {
			Model, Instructions, Truncation string
			Store                           *bool
			MaxOutputTokens                 int `json:"max_output_tokens"`
			Input                           []struct{ Role, Content string }
			Text                            struct {
				Format struct {
					Type   string
					Strict bool
					Schema struct {
						Properties           map[string]any
						Required             []string
						AdditionalProperties *bool
					}
				}
			}
		}
		if err := json.Unmarshal(raw, &req); err != nil || len(req.Input) != 1 || req.Input[0].Role != "user" || req.Model != "test-model" || req.Instructions == "" || req.Store == nil || *req.Store || req.Truncation != "disabled" || req.MaxOutputTokens != 96 {
			t.Errorf("unexpected bounded Responses request: %s", raw)
			return
		}
		format := req.Text.Format
		fields := []string{"interpretation", "quote", "unresolved", "verdict"}
		if format.Type != "json_schema" || !format.Strict || format.Schema.AdditionalProperties == nil || *format.Schema.AdditionalProperties || !slices.Equal(slices.Sorted(maps.Keys(format.Schema.Properties)), fields) || !slices.Equal(format.Schema.Required, fields) {
			t.Errorf("incorrect provider schema: %+v", format)
			return
		}
		var in struct {
			Question, Role, Query, Document, Version, Snippet string
			Start, End                                        int
		}
		if err := json.Unmarshal([]byte(req.Input[0].Content), &in); err != nil {
			t.Error(err)
			return
		}
		f, ok := caseFacts[in.Role]
		start := strings.Index(f.content, f.quote)
		if !ok || in.Question != caseQuestion || in.Query != f.query || in.Document != f.document || in.Version != fmt.Sprintf("%x", sha256.Sum256([]byte(f.content))) || in.Start != start || in.End != start+len(f.quote) || in.Snippet != f.quote {
			t.Errorf("model received different retained evidence: %+v", in)
			return
		}
		mode := "supports"
		if in.Role == "counter" {
			mode = counterMode
		}
		status, body := fixtureResponse(in.Role, mode, in.Snippet)
		c := providerContact{role: in.Role, clientID: r.Header.Get("X-Client-Request-Id"), wire: string(raw), mode: mode, response: body, requestID: "req-" + in.Role, status: status}
		p.mu.Lock()
		p.log = append(p.log, c)
		p.mu.Unlock()
		select {
		case p.arrived <- c:
		case <-r.Context().Done():
			return
		}
		if gate := p.gates[c.role]; gate != nil {
			select {
			case <-gate:
			case <-r.Context().Done():
				return
			}
		}
		if mode == "disconnect" {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		w.Header().Set("x-request-id", c.requestID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(func() { p.server.CloseClientConnections(); p.server.Close() })
	return p
}

func fixtureResponse(role, mode, snippet string) (int, string) {
	if mode == "malformed" {
		return 200, "{invalid envelope"
	}
	if mode == "http_rejected" {
		return 429, `{"error":{"type":"rate_limit","message":"controlled refusal"}}`
	}
	proposal := map[string]string{"verdict": "supports", "quote": snippet, "interpretation": "The " + role + " source records a dated observation.", "unresolved": ""}
	if mode == "opposes" {
		proposal["verdict"] = "opposes"
	}
	if mode == "fabricated_quote" {
		proposal["quote"] = caseFacts["primary"].quote
	}
	if mode == "schema_invalid" {
		delete(proposal, "quote")
	}
	payload, _ := json.Marshal(proposal)
	part := map[string]string{"type": "output_text", "text": string(payload)}
	status := "completed"
	if mode == "refusal" {
		part = map[string]string{"type": "refusal", "refusal": "controlled refusal"}
	}
	if mode == "incomplete" {
		status = "incomplete"
	}
	if mode == "provider_failed" {
		status = "failed"
	}
	envelope := map[string]any{"id": "resp-" + role, "model": "served-test-model", "status": status, "output": []any{map[string]any{"type": "message", "role": "assistant", "status": "completed", "content": []any{part}}}}
	if mode != "no_usage" {
		envelope["usage"] = map[string]any{"input_tokens": 17, "output_tokens": 3, "total_tokens": 20, "output_tokens_details": map[string]int{"reasoning_tokens": 1}}
	}
	raw, _ := json.Marshal(envelope)
	return 200, string(raw)
}

func (p *responsesFixture) contacts() []providerContact {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.log)
}

func (p *responsesFixture) waitBoth(t *testing.T) {
	t.Helper()
	roles := map[string]bool{}
	timer := time.NewTimer(processDeadline / 2)
	defer timer.Stop()
	for len(roles) < 2 {
		select {
		case c := <-p.arrived:
			if roles[c.role] {
				t.Fatalf("repeated contact for %s", c.role)
			}
			roles[c.role] = true
		case <-timer.C:
			t.Fatalf("expected both provider contacts, got %v", roles)
		}
	}
}
func (p *responsesFixture) allow(role string) { close(p.gates[role]) }

func initializeCase(t *testing.T, p *responsesFixture, maxEffects int) string {
	t.Helper()
	dir := t.TempDir()
	db := filepath.Join(dir, "case.sqlite")
	spec := filepath.Join(dir, "case.json")
	for _, f := range caseFacts {
		if err := os.WriteFile(filepath.Join(dir, f.document+".txt"), []byte(f.content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// The external input and protocol oracle use their JSON wire formats.
	raw := fmt.Appendf(nil, `{"question":%q,"queries":["Alpha","Beta"],"documents":[{"id":"permit","path":"permit.txt"},{"id":"audit","path":"audit.txt"}],"provider":{"model":"test-model","endpoint":%q,"max_output_tokens":96,"max_response_bytes":16384},"limits":{"max_effects":%d,"max_total_output_tokens":192,"max_in_flight":2}}`, caseQuestion, p.server.URL+"/v1/responses", maxEffects)
	if err := os.WriteFile(spec, raw, 0600); err != nil {
		t.Fatal(err)
	}
	cliOutput(t, "init", db, spec)
	// Resume and citation verification must use retained bytes.
	for _, f := range caseFacts {
		if err := os.WriteFile(filepath.Join(dir, f.document+".txt"), []byte("unrelated replacement"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func checkDiagnostic(t *testing.T, provider providerContact, got string) {
	t.Helper()
	if got != provider.response {
		t.Fatalf("retained %s HTTP body differs from the independent response, got %q", provider.role, got)
	}
}
