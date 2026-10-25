//go:build integration

package main

import (
	"encoding/json"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// These children must work after the selected source disappears and credentials
// are removed. Rebuilding application artifacts on run would break this contract.
func TestCLIProcessesRetainCorpusAndReplayOffline(t *testing.T) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := contacts.Add(1)
		var request struct{ Input []struct{ Content string } }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		payload := map[string]any{"decision": "lookup", "tool": "lookup", "query": "WAL", "message": ""}
		if n == 2 {
			var in map[string]any
			if err := json.Unmarshal([]byte(request.Input[0].Content), &in); err != nil {
				t.Error(err)
			}
			if in["snippet"] != "WAL retains a durable log." {
				t.Errorf("snapshot lost: %+v", in)
			}
			payload = map[string]any{"answer": "WAL retains a durable log.", "document": in["document"], "version": in["version"], "snippet": in["snippet"], "start": in["start"], "end": in["end"]}
		}
		text, _ := json.Marshal(payload)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "controlled-cli-response", "model": "controlled-cli-returned", "status": "completed", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(text)}}}}})
	}))
	defer server.Close()
	dir := t.TempDir()
	binary := filepath.Join(dir, "alan-evidence")
	path := filepath.Join(dir, "app.sqlite")
	corpus := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(corpus, []byte("WAL retains a durable log.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"build", "-buildvcs=false", "-o", binary}
	if raceChildren {
		args = append(args, "-race")
	}
	args = append(args, ".")
	t.Logf("child build: go %v", args)
	if out, err := exec.CommandContext(t.Context(), "go", args...).CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	invoke := func(secret string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), binary, args...)
		for _, v := range os.Environ() {
			if !strings.HasPrefix(v, "OPENAI_API_KEY=") {
				cmd.Env = append(cmd.Env, v)
			}
		}
		if secret != "" {
			cmd.Env = append(cmd.Env, "OPENAI_API_KEY="+secret)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v %s", args, err, out)
		}
		if strings.Contains(string(out), "cli-secret") {
			t.Fatal("credential leaked")
		}
		return out
	}
	invoke("", "init", "--db", path, "--question", "What does WAL retain?", "--model", "controlled-cli", "--endpoint", server.URL, "--corpus", corpus, "--request-timeout", "3s")
	if contacts.Load() != 0 {
		t.Fatal("init contacted provider")
	}
	if err := os.Remove(corpus); err != nil {
		t.Fatal(err)
	}
	invoke("cli-secret", "run", "--db", path)
	var view evidence.View
	if err := json.Unmarshal(invoke("", "inspect", "--db", path), &view); err != nil {
		t.Fatal(err)
	}
	if view.Result["answer"].Text != "WAL retains a durable log." || view.ModelRequests != 2 || view.LookupRequests != 1 || view.Attempts != 3 {
		t.Fatalf("result/accounting: %+v", view)
	}
	if view.Config.Model != "controlled-cli" || len(view.Results) != 3 {
		t.Fatal("config/results missing")
	}
	for _, r := range view.Results {
		if r.Candidate.Usage != nil {
			t.Fatal("fabricated usage")
		}
	}
	if contacts.Load() != 2 {
		t.Fatalf("contacts=%d", contacts.Load())
	}
	invoke("cli-secret", "recover", "--db", path)
	if contacts.Load() != 2 {
		t.Fatal("recovery repeated recorded effect")
	}
	server.Close()
	var replay runtime.ReplayReport
	if err := json.Unmarshal(invoke("", "replay", "--db", path), &replay); err != nil {
		t.Fatal(err)
	}
	if len(replay.Runs) != 3 {
		t.Fatal("missing ordinary composition")
	}
	for _, r := range replay.Runs {
		if r.Projection.Status != "succeeded" {
			t.Fatalf("unfinished run %s", r.ID.Subject)
		}
	}
	t.Logf("controlled CLI answer=%q contacts=%d logical model=%d lookup=%d attempt reservations=%d; offline replay succeeded", view.Result["answer"].Text, contacts.Load(), view.ModelRequests, view.LookupRequests, view.Attempts)
}
