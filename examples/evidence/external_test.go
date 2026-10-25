//go:build integration

package evidence_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestExternalConsumerUsesTwoPublicHelperContracts(t *testing.T) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		_, _ = w.Write([]byte(`{"status":"completed","model":"controlled-returned","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"decision\":\"clarification\",\"tool\":\"\",\"query\":\"\",\"message\":\"Which durability guarantee do you mean?\"}"}]}]}`))
	}))
	defer server.Close()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("testdata/consumer.go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err = os.WriteFile(filepath.Join(dir, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	mod := "module external.example/evidenceconsumer\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + root + "\n"
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "consumer")
	args := []string{"build", "-mod=mod", "-o", binary}
	if raceChildren {
		args = append(args, "-race")
	}
	args = append(args, ".")
	t.Logf("external child: go %v", args)
	build := exec.CommandContext(t.Context(), "go", args...)
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	path := filepath.Join(dir, "external.sqlite")
	for _, mode := range []string{"create", "resume", "replay"} {
		if mode == "replay" {
			server.Close()
		}
		cmd := exec.CommandContext(t.Context(), binary, mode, path, server.URL)
		for _, v := range os.Environ() {
			if !strings.HasPrefix(v, "OPENAI_API_KEY=") {
				cmd.Env = append(cmd.Env, v)
			}
		}
		if mode == "resume" {
			cmd.Env = append(cmd.Env, "OPENAI_API_KEY=external-controlled-secret")
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v %s", mode, err, out)
		}
		t.Logf("%s: %s", mode, out)
	}
	if contacts.Load() != 2 {
		t.Fatalf("two helpers should each plan once: contacts=%d", contacts.Load())
	}
}
