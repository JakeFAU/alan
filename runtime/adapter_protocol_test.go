//go:build integration

package runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	runtime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func modelProvision(t *testing.T) runtime.Provision {
	t.Helper()
	a, e := author.Compile(author.Definition{Effects: map[string]author.Effect{"model": {Capability: "provider", Input: schema.Record{"instructions": {Kind: "string", MaxBytes: 100}, "input": {Kind: "string", MaxBytes: 100}}, Output: schema.Record{"answer": {Kind: "string", MaxBytes: 100}}}}, Steps: []author.Step{author.Request("request", "model", map[string]author.Expr{"instructions": author.String("Retained instructions"), "input": author.String("Full retained input")}, nil), author.Await("model", "request", map[string]string{"answer": "answer", "valid": "valid"}), author.IfEqual(author.Var("valid"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()})}})
	if e != nil {
		t.Fatal(e)
	}
	p, e := (author.Application{"agent": {Artifact: a}}).Provision()
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestRealResponsesRuntimeRetainsThenResumesOffline(t *testing.T) {
	for _, body := range []string{`{"id":"response","model":"actual","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"answer\":\"recorded answer\"}"}]}],"usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8}}`, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`} {
		t.Run(body[:12], func(t *testing.T) {
			ctx := context.Background()
			var contacts atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contacts.Add(1)
				w.Header().Set("x-request-id", "request")
				io.WriteString(w, body)
			}))
			a, e := openai.New(openai.Config{Model: "explicit", APIKey: "fixture-runtime-secret", Endpoint: server.URL, MaxOutputTokens: 20, MaxResponseBytes: 8192})
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "store.db")
			s, e := runtime.Create(ctx, path, modelProvision(t), runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			stop := errors.New("post-record")
			observed := false
			runctx := boundary.WithObserver(ctx, func(_ context.Context, event boundary.Event) error {
				if event.Boundary == "adapter.result-recorded" {
					observed = true
					return stop
				}
				return nil
			})
			r, e := s.Run(runctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"provider": a}})
			if !errors.Is(e, stop) || !observed || len(r.UnknownAdapters) != 0 {
				t.Fatal(r, e)
			}
			s.Close()
			server.Close()
			s, e = runtime.Open(ctx, path, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			r, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20})
			if e != nil || len(r.OpenRuns) != 0 {
				t.Fatal(r, e)
			}
			s.Close()
			inspection, e := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := json.Marshal(inspection)
			if strings.Contains(string(raw), "fixture-runtime-secret") {
				t.Fatal("runtime credential persisted")
			}
			if !strings.Contains(string(raw), "Full retained input") || !strings.Contains(string(raw), "Retained instructions") {
				t.Fatal("input not retained")
			}
			replay, e := runtime.Replay(ctx, path, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			want := "succeeded"
			if strings.Contains(body, "incomplete") {
				want = "failed"
			}
			if replay.Runs[0].Projection.Status != want || contacts.Load() != 1 {
				t.Fatalf("offline replay %+v contacts=%d", replay, contacts.Load())
			}
		})
	}
}
func TestRealResponsesUnknownDoesNotRepeat(t *testing.T) {
	ctx := context.Background()
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		contacts.Add(1)
		c, _, e := w.(http.Hijacker).Hijack()
		if e != nil {
			t.Error(e)
			return
		}
		c.Close()
	}))
	defer server.Close()
	a, _ := openai.New(openai.Config{Model: "explicit", APIKey: "fixture-runtime-secret", Endpoint: server.URL, MaxOutputTokens: 20, MaxResponseBytes: 8192})
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := runtime.Create(ctx, path, modelProvision(t), runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"provider": a}})
	if e != nil || len(r.UnknownAdapters) != 1 || contacts.Load() != 1 {
		t.Fatal(r, e, contacts.Load())
	}
	s.Close()
	s, e = runtime.Open(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	r, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"provider": a}})
	s.Close()
	if e != nil || len(r.UnknownAdapters) != 1 || contacts.Load() != 1 {
		t.Fatal("unknown repeated", r, e, contacts.Load())
	}
	if _, e = runtime.Replay(ctx, path, runtime.DefaultLimits()); e != nil {
		t.Fatal(e)
	}
}
func TestLookupSnapshotSurvivesMissingSource(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	source := filepath.Join(dir, "selected.txt")
	if e := os.WriteFile(source, []byte("prefix\nSQLite retains durable records.\nend"), 0600); e != nil {
		t.Fatal(e)
	}
	asset, e := effects.Snapshot("document", source)
	if e != nil {
		t.Fatal(e)
	}
	a, e := author.Compile(author.Definition{Effects: map[string]author.Effect{"lookup": {Capability: "corpus", Input: lookup.InputSchema(), Output: lookup.OutputSchema()}}, Steps: []author.Step{author.Request("request", "lookup", map[string]author.Expr{"query": author.String("durable")}, nil), author.Await("lookup", "request", map[string]string{"snippet": "snippet", "start": "start", "end": "end", "version": "version", "valid": "valid"}), author.Success()}})
	if e != nil {
		t.Fatal(e)
	}
	p, e := (author.Application{"agent": {Artifact: a}}).Provision()
	if e != nil {
		t.Fatal(e)
	}
	p.Assets = []effects.Asset{asset}
	path := filepath.Join(dir, "store.db")
	s, e := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(source); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"corpus": lookup.New()}}); e != nil {
		t.Fatal(e)
	}
	s.Close()
	rr, e := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	data := rr.Runs[0].Projection.Data
	if data["snippet"].Text != "SQLite retains durable records." || data["start"].Number != 7 || data["end"].Number != 38 || data["version"].Text != asset.Version {
		t.Fatalf("wrong retained bytes %+v", data)
	}
}
func TestLookupFieldsReachModelWithoutStringOpcode(t *testing.T) {
	ctx := context.Background()
	asset, e := effects.NewAsset("document", "fixture", []byte("prefix\nSQLite retains durable records.\nend"))
	if e != nil {
		t.Fatal(e)
	}
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		var body struct {
			Input []struct{ Content string } `json:"input"`
		}
		if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
			t.Error(e)
		}
		var input map[string]any
		if len(body.Input) != 1 {
			t.Error("missing model input")
		} else if e := json.Unmarshal([]byte(body.Input[0].Content), &input); e != nil {
			t.Error(e)
		}
		if input["question"] != "Where are durable records?" || input["document"] != "document" || input["version"] != asset.Version || input["snippet"] != "SQLite retains durable records." || input["start"] != float64(7) || input["end"] != float64(38) {
			t.Errorf("incomplete synthesis input: %+v", input)
		}
		io.WriteString(w, `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"answer\":\"recorded\"}"}]}]}`)
	}))
	defer server.Close()
	model, _ := openai.New(openai.Config{APIKey: "fixture-runtime-secret", Model: "explicit", Endpoint: server.URL, MaxOutputTokens: 20, MaxResponseBytes: 8192})
	modelInput := lookup.OutputSchema()
	modelInput["instructions"] = schema.Field{Kind: "string", MaxBytes: 100}
	modelInput["question"] = schema.Field{Kind: "string", MaxBytes: 100}
	a, e := author.Compile(author.Definition{Inputs: schema.Record{"question": {Kind: "string", MaxBytes: 100}}, Effects: map[string]author.Effect{"lookup": {Capability: "corpus", Input: lookup.InputSchema(), Output: lookup.OutputSchema()}, "model": {Capability: "provider", Input: modelInput, Output: schema.Record{"answer": {Kind: "string", MaxBytes: 100}}}}, Steps: []author.Step{author.Request("lookup-request", "lookup", map[string]author.Expr{"query": author.Var("question")}, nil), author.Await("lookup", "lookup-request", map[string]string{"document": "document", "version": "version", "snippet": "snippet", "start": "start", "end": "end"}), author.Request("model-request", "model", map[string]author.Expr{"instructions": author.String("Answer from this complete retained evidence record"), "question": author.Var("question"), "document": author.Var("document"), "version": author.Var("version"), "snippet": author.Var("snippet"), "start": author.Var("start"), "end": author.Var("end")}, nil), author.Await("model", "model-request", map[string]string{"answer": "answer", "valid": "valid"}), author.Success()}})
	if e != nil {
		t.Fatal(e)
	}
	p, e := (author.Application{"agent": {Artifact: a, Inputs: map[string]schema.Value{"question": schema.Text("Where are durable records?")}}}).Provision()
	if e != nil {
		t.Fatal(e)
	}
	p.Assets = []effects.Asset{asset}
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"provider": model, "corpus": lookup.New()}})
	if e != nil || len(r.OpenRuns) != 0 || contacts.Load() != 1 {
		t.Fatal(r, e, contacts.Load())
	}
	s.Close()
	replay, e := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if replay.Runs[0].Projection.Data["answer"].Text != "recorded" {
		t.Fatal("model result missing")
	}
}
