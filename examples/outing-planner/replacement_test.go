package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlannerCLIReplacementWitness(t *testing.T) {
	dir := t.TempDir()
	output := plannerOutput(t, "check-replacement", "--config", "testdata/configuration.json",
		"--fixture", "testdata/fixture.json", "--weather", confirmFirst, "--output-dir", dir)
	plannerOutcome(t, output, "circular_dependency")
	for _, want := range []string{
		"The planner needs a forecast; the forecast service requires the planner’s final choice.",
		"first missing message", "outing-assessment", "outing-decision",
		`"pc"`, `"history"`, `"reopened"`, `"source_config_sha256"`, `"derived_config_sha256"`, `"fixture_sha256"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("replacement report lacks %s", want)
		}
	}
	for name, outcome := range map[string]string{"baseline.db": recommended, "candidate.db": "pending"} {
		path := filepath.Join(dir, name)
		plannerOutcome(t, plannerOutput(t, "replay", "--db", path, "--json"), outcome)
	}
	persisted, err := os.ReadFile(filepath.Join(dir, "replacement.json"))
	if err != nil || string(persisted) != output {
		t.Fatalf("persisted report differs: %v", err)
	}
	before, _, _, err := inspectPlanner(t.Context(), filepath.Join(dir, "candidate.db"))
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"check-replacement", "--config", "testdata/configuration.json",
		"--fixture", "testdata/fixture.json", "--weather", confirmFirst, "--output-dir", dir}
	if err := runCLI(t.Context(), args, new(strings.Builder)); err == nil {
		t.Fatal("checker overwrote retained Stores")
	}
	after, _, _, err := inspectPlanner(t.Context(), filepath.Join(dir, "candidate.db"))
	if err != nil || after.Namespace != before.Namespace {
		t.Fatalf("checker changed existing evidence: %v", err)
	}
	for label, file := range map[string]string{"source_config_sha256": "source-configuration.json", "derived_config_sha256": "derived-configuration.json", "fixture_sha256": "fixture.json"} {
		raw, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		var report map[string]json.RawMessage
		if err := json.Unmarshal(persisted, &report); err != nil {
			t.Fatal(err)
		}
		if string(report[label]) != `"`+digestText(string(raw))+`"` {
			t.Fatalf("%s does not bind retained file", label)
		}
	}
}

func TestPlannerCLIReplacementRequiresFixtureTupleCoverage(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version       int               `json:"version"`
		ReferenceTime string            `json:"reference_time"`
		Entries       []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, want string
		entries    []json.RawMessage
	}{
		{"missing", "missing fixture tuple", fixture.Entries[:1]},
		{"duplicate", "duplicate fixture tuple", []json.RawMessage{fixture.Entries[0], fixture.Entries[0]}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture.Entries = tc.entries
			path := filepath.Join(t.TempDir(), "fixture.json")
			payload, err := json.Marshal(fixture)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, payload, 0600); err != nil {
				t.Fatal(err)
			}
			err = runCLI(t.Context(), []string{"check-replacement", "--config", "testdata/configuration.json",
				"--fixture", path, "--weather", confirmFirst}, new(strings.Builder))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q error, got %v", tc.want, err)
			}
		})
	}
}
