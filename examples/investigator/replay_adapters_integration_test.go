//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A new process starts its Go coverage counters at zero. The run establishes
// that the instrumentation sees each real adapter. Replay must leave every
// statement in those same Prepare and Execute implementations untouched.
func TestInvestigatorReplayDoesNotExecuteAdapters(t *testing.T) {
	p := responsesServer(t, "supports")
	db := initializeCase(t, p, 4)
	for _, action := range []string{"run", "replay"} {
		profile := filepath.Join(t.TempDir(), "adapter.cover")
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-run=^TestInvestigatorProcess$", "-coverpkg=github.com/JakeFAU/alan/adapters/lookup,github.com/JakeFAU/alan/adapters/openai", "-coverprofile="+profile, ".")
		args, err := json.Marshal([]string{action, db})
		if err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "OPENAI_API_KEY=", "ALAN_INVESTIGATOR_TEST_ARGS="+string(args))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("instrumented %s: %v: %s", action, err, output)
		}
		output, err := exec.CommandContext(ctx, "go", "tool", "cover", "-func="+profile).CombinedOutput()
		if err != nil {
			t.Fatalf("read %s coverage: %v: %s", action, err, output)
		}
		seen := map[string]bool{}
		for line := range strings.SplitSeq(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 3 || (fields[1] != "Execute" && fields[1] != "Prepare") {
				continue
			}
			path, _, _ := strings.Cut(fields[0], ":")
			if !strings.HasSuffix(path, "/adapters/lookup/lookup.go") && !strings.HasSuffix(path, "/adapters/openai/openai.go") {
				continue
			}
			if (fields[2] == "0.0%") != (action == "replay") {
				t.Fatalf("%s touched unexpected adapter statements: %s", action, line)
			}
			seen[path+"."+fields[1]] = true
		}
		if len(seen) != 4 || len(p.contacts()) != 2 {
			t.Fatalf("%s: incomplete adapter evidence, funcs=%v contacts=%d", action, seen, len(p.contacts()))
		}
		t.Logf("%s: %d checked adapter functions; cumulative independent POSTs=%d", action, len(seen), len(p.contacts()))
	}
}
