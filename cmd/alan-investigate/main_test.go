package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/runtime"
)

// This catches dropping manifest identity, reading sources on recovery, losing
// nested selection, and accidentally requiring credentials for read-only tools.
func TestCLIManifestInitRecoverInspectReplay(t *testing.T) {
	for _, baseline := range []bool{false, true} {
		t.Run(strconv.FormatBool(baseline), func(t *testing.T) {
			dir := t.TempDir()
			corpus := filepath.Join(dir, "corpus")
			if err := os.Mkdir(corpus, 0700); err != nil {
				t.Fatal(err)
			}
			data := []byte("alpha is a fact.\n")
			asset, _ := effects.NewAsset("source-a", "fixture", data)
			file := filepath.Join(corpus, "a.txt")
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			manifest := filepath.Join(corpus, "manifest.json")
			raw, _ := json.Marshal(map[string]any{"schema": "phase1f-corpus/v1", "documents": []any{map[string]any{"id": "source-a", "path": "corpus/a.txt", "version": "1", "sha256": asset.Version, "bytes": len(data)}}})
			if err := os.WriteFile(manifest, raw, 0600); err != nil {
				t.Fatal(err)
			}
			var contacts atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contacts.Add(1)
				payload := map[string]string{"decision": "clarification", "subquestion1": "", "subquestion2": "", "message": "Which release?"}
				if baseline {
					payload = map[string]string{"decision": "clarification", "tool": "", "query": "", "message": "Which release?"}
				}
				text, _ := json.Marshal(payload)
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "completed", "model": "fixture", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]string{"type": "output_text", "text": string(text)}}}}})
			}))
			defer server.Close()
			path := filepath.Join(dir, "run.sqlite")
			args := []string{"init", "--db", path, "--question", "question", "--model", "fixture", "--endpoint", server.URL, "--corpus-manifest", manifest}
			if baseline {
				args = append(args, "--baseline")
			} else {
				args = append(args, "--nested", "--max-revisions", "0")
			}
			var out bytes.Buffer
			noSecret := func(string) string { t.Fatal("read-only command requested credentials"); return "" }
			if err := runCLI(t.Context(), args, &out, noSecret); err != nil {
				t.Fatal(err)
			}
			if contacts.Load() != 0 {
				t.Fatal("init contacted provider")
			}
			snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(snapshot)
			if !bytes.Contains(encoded, []byte(asset.Version)) {
				t.Fatal("manifest content hash not retained")
			}
			if !baseline {
				v, err := investigation.Inspect(t.Context(), path)
				if err != nil {
					t.Fatal(err)
				}
				if !v.Config.Nested || v.Config.MaxRevisions != 0 {
					t.Fatal("config not retained")
				}
			}
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(manifest); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			if err := runCLI(t.Context(), []string{"recover", "--db", path}, &out, func(string) string { return "controlled-secret" }); err != nil {
				t.Fatal(err)
			}
			for _, command := range []string{"inspect", "replay"} {
				out.Reset()
				if err := runCLI(t.Context(), []string{command, "--db", path}, &out, noSecret); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(out.String(), "controlled-secret") {
					t.Fatal("secret in output")
				}
			}
			if contacts.Load() != 1 {
				t.Fatalf("contacts=%d", contacts.Load())
			}
		})
	}
}
func TestCLIRejectsInvalidInputs(t *testing.T) {
	for _, args := range [][]string{{}, {"unknown"}, {"init", "--db", "unused", "--model", "fixture", "--question", "q", "--max-revisions", "2"}, {"init", "--db", "unused", "--model", "fixture", "--question", "q"}, {"inspect", "--db", "x", "extra"}} {
		if err := runCLI(t.Context(), args, &bytes.Buffer{}, func(string) string { return "" }); err == nil {
			t.Fatal("accepted invalid CLI", args)
		}
	}
}

func TestManifestRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	corpus := filepath.Join(dir, "corpus")
	if err := os.Mkdir(corpus, 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte("exact bytes\n")
	a, _ := effects.NewAsset("a", "fixture", data)
	if err := os.WriteFile(filepath.Join(corpus, "a.txt"), data, 0600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(corpus, "manifest.json")
	for _, tc := range []struct {
		name, path, hash string
		size             int
	}{
		{"hash", "corpus/a.txt", strings.Repeat("0", 64), len(data)},
		{"size", "corpus/a.txt", a.Version, len(data) - 1},
		{"traversal", "../a.txt", a.Version, len(data)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]any{"schema": "phase1f-corpus/v1", "documents": []any{map[string]any{"id": "a", "path": tc.path, "sha256": tc.hash, "bytes": tc.size}}})
			if err := os.WriteFile(manifest, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadManifest(manifest); err == nil {
				t.Fatal("manifest mismatch accepted")
			}
		})
	}
}
