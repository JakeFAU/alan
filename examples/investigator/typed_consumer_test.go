package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Build the example under an unrelated module path. An internal ALAN import in
// the consumer would fail here even though it could compile inside this repo.
func externalInspector(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("../inspect-execution/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), src, 0600); err != nil {
		t.Fatal(err)
	}
	module := "module example.org/independent-inspector\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + strconv.Quote(root) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	// Pin third-party sums in the consumer just as they are pinned in ALAN.
	sums, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "inspect-execution")
	ctx, cancel := context.WithTimeout(t.Context(), processDeadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-mod=mod", "-o", binary, ".")
	cmd.Dir, cmd.Env = dir, append(os.Environ(), "GOWORK=off", "GOPROXY=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build independent consumer: %v\n%s", err, output)
	}
	return binary
}

func TestExternalExecutionInspector(t *testing.T) {
	binary := externalInspector(t)
	inspect := func(t *testing.T, path string, p *responsesFixture) string {
		t.Helper()
		before, contacts := retainedBytes(t, path), len(p.contacts())
		ctx, cancel := context.WithTimeout(t.Context(), processDeadline)
		defer cancel()
		output, err := exec.CommandContext(ctx, binary, path).CombinedOutput()
		if err != nil {
			t.Fatalf("read execution summary: %v\n%s", err, output)
		}
		if len(p.contacts()) != contacts || !reflect.DeepEqual(before, retainedBytes(t, path)) {
			t.Fatal("external inspector changed a durable byte or contacted the fixture")
		}
		return string(output)
	}
	t.Run("completed", func(t *testing.T) {
		p := responsesServer(t, "opposes")
		db := initializeCase(t, p, 4)
		cliOutput(t, "run", db)
		text := inspect(t, db, p)
		if len(p.contacts()) != 2 || !strings.Contains(text, `authored_completion="disagreement"`) ||
			!strings.Contains(text, `worker_role="primary"`) || !strings.Contains(text, `worker_role="counter"`) ||
			strings.Count(text, " parent_selected_sequence=") != 2 || strings.Count(text, "matches_source=true") != 2 ||
			strings.Count(text, "reported_tokens[input=17 output=3 total=20]") != 2 ||
			!strings.Contains(text, "effect_reservations=4/4") || strings.Contains(text, "remote_outcome=unknown") {
			t.Fatal("incomplete or misleading completed summary", text)
		}
		t.Log("completed disagreement: two selected workers, two byte-verified source spans, two provider contacts")
	})
	t.Run("known worker refusal", func(t *testing.T) {
		p := responsesServer(t, "refusal")
		db := initializeCase(t, p, 4)
		cliOutput(t, "run", db)
		text := inspect(t, db, p)
		if len(p.contacts()) != 2 || !strings.Contains(text, `authored_completion="partial"`) ||
			!strings.Contains(text, `retained_disposition="refusal"`) || !strings.Contains(text, "outcome=failed") ||
			strings.Count(text, " parent_selected_sequence=") != 2 {
			t.Fatal("known provider refusal confused with a missing result", text)
		}
		t.Log("known refusal: failed worker publication selected, authored completion partial, two provider contacts")
	})
	t.Run("live and killed owner", func(t *testing.T) {
		p := responsesServer(t, "supports", "counter")
		db := initializeCase(t, p, 4)
		c := startInvestigator(t, []string{"run", db}, repeatedCut(publicationCut, "pre-consumption", 1)...)
		p.waitBoth(t)
		event := c.next(t)
		witness := inspectCase(t, db)
		witness.requirePublication(t, event, "primary")
		operation := witness.model("counter").Intent.ID
		if operation == "" || len(p.contacts()) != 2 {
			t.Fatal("lost the independently witnessed original model call")
		}
		check := func(text string) {
			want := fmt.Sprintf("effect operation=%q capability=%q", operation, "model")
			found := false
			for line := range strings.SplitSeq(text, "\n") {
				if strings.Contains(line, want) && strings.Contains(line, "remote_outcome=unknown retry=not-automatically-authorized") &&
					strings.Contains(line, "reported_usage=unknown") && strings.Contains(line, "physical_executions=0..1") {
					found = true
				}
			}
			if !found || !strings.Contains(text, " depends_on_run=") || !strings.Contains(text, "parent_selection=unconsumed") || strings.Contains(text, "authored_completion=") ||
				!strings.Contains(text, "effect_reservations=4/4") {
				t.Fatal("inaccurate interrupted summary", text)
			}
		}
		live := inspect(t, db, p)
		check(live)
		c.kill(t, "external server accepted counter operation "+operation+"; primary publication "+event.Attempt)
		p.allow("counter")
		killed := inspect(t, db, p)
		check(killed)
		t.Logf("after SIGKILL: unknown provider operation %s, one published worker outcome still unselected, %d POSTs", operation, len(p.contacts()))
		if killed != live {
			t.Fatal("process death alone changed the retained causal summary")
		}
		cliOutput(t, "resume", db)
		reopened := inspect(t, db, p)
		if !strings.Contains(reopened, fmt.Sprintf("effect operation=%q capability=%q", operation, "model")) ||
			!strings.Contains(reopened, "remote_outcome=unknown retry=not-automatically-authorized") || len(p.contacts()) != 2 {
			t.Fatal("recovery changed original identity or spent another provider attempt", reopened)
		}
	})
}
