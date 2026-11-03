package main

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"

	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveConsentPrecedesOwnership(t *testing.T) {
	dir := t.TempDir()
	specPath, store := filepath.Join(dir, "spec.json"), filepath.Join(dir, "case.sqlite")
	spec := investigationSpec{
		Question: "Does the trial include follow-up?", Queries: []string{"trial", "follow-up"},
		Documents: []documentSpec{{ID: "trial", Path: "trial.txt"}},
		deployment: deployment{
			Provider: providerConfig{Model: "explicit-model", MaxOutputTokens: 96, MaxResponseBytes: 16384},
			Limits:   spendingLimits{MaxEffects: 4, MaxInFlight: 2, MaxTotalOutputTokens: 192},
		},
	}
	body, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trial.txt"), []byte("The trial includes follow-up."), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(t.Context(), []string{"init", store, specPath}, io.Discard); err != nil {
		t.Fatal(err)
	}
	var before, after bytes.Buffer
	if err := run(t.Context(), []string{"inspect", store}, &before); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "synthetic-do-not-transmit")
	if err := run(t.Context(), []string{"resume", store}, io.Discard); err == nil || !strings.Contains(err.Error(), "--live") {
		t.Fatalf("expected opt-in refusal, got %v", err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	if err := run(t.Context(), []string{"run", store, "--live"}, io.Discard); err == nil {
		t.Fatal("live run accepted without credentials")
	}
	if err := run(t.Context(), []string{"init", store, specPath}, io.Discard); err == nil {
		t.Fatal("duplicate initialization replaced the investigation")
	}
	if err := run(t.Context(), []string{"inspect", store}, &after); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatal("rejected operation changed retained state")
	}
}

func TestCommandUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"init", "store"}, {"run", "store", "--force"}, {"answer", "store", "--live"}, {"other", "store"}} {
		if err := run(t.Context(), args, io.Discard); err == nil || err.Error() != usage {
			t.Errorf("%v: %v", args, err)
		}
	}
}

// These are valid public-API stores. A deployment that advertises a smaller
// budget, substitutes a different parent, or adds another static run must
// be rejected before the CLI opens an execution owner or calls an adapter.
func TestRejectsForeignInvestigatorDeployments(t *testing.T) {
	d := deployment{Provider: providerConfig{Model: "fixture", Endpoint: "http://127.0.0.1:1/v1/responses", MaxOutputTokens: 64, MaxResponseBytes: 4096}, Limits: spendingLimits{MaxEffects: 4, MaxInFlight: 2, MaxTotalOutputTokens: 128}}
	settings, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	other, err := author.Compile(author.Definition{Name: "other", Steps: []author.Step{author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"limits", "extra-run", "parent"} {
		t.Run(scenario, func(t *testing.T) {
			p, err := investigatorProvision("Question?", [2]string{"alpha", "beta"}, string(settings), d.resources())
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "limits":
				p.Composition.Roots[0].Limits.MaxEffects++
			case "extra-run":
				addendum, err := (author.Application{"unrelated": {Artifact: other}}).Provision()
				if err != nil {
					t.Fatal(err)
				}
				p.Subjects = append(p.Subjects, addendum.Subjects...)
				p.Runs = append(p.Runs, addendum.Runs...)
				p.Programs = append(p.Programs, addendum.Programs...)
			case "parent":
				fake, err := author.Compile(author.Definition{Name: "unrelated", Inputs: schema.Record{"question": textField(4096), "primary_query": textField(512), "counter_query": textField(512), "settings": textField(4096)}, Steps: []author.Step{author.Success()}})
				if err != nil {
					t.Fatal(err)
				}
				bad, err := (author.Application{"investigator": {Artifact: fake, Inputs: p.Runs[0].Projection.Data}}).Provision()
				if err != nil {
					t.Fatal(err)
				}
				p.Programs = append(p.Programs, bad.Programs...)
				p.Runs = bad.Runs
			}
			path := filepath.Join(t.TempDir(), "foreign.sqlite")
			store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			var before, after bytes.Buffer
			if err := run(t.Context(), []string{"inspect", path}, &before); err != nil {
				t.Fatal(err)
			}
			if err := run(t.Context(), []string{"run", path}, io.Discard); err == nil || !strings.Contains(err.Error(), "investigator deployment") {
				t.Fatalf("did not reject foreign %s before dispatch: %v", scenario, err)
			}
			if err := run(t.Context(), []string{"inspect", path}, &after); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before.Bytes(), after.Bytes()) {
				t.Fatal("rejected store advanced")
			}
		})
	}
}
