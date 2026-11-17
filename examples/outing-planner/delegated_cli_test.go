package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeDelegatedConfig(t *testing.T, c configuration) string {
	t.Helper()
	raw, err := c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "configuration.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDelegatedCLIStartOptions(t *testing.T) {
	server, calls := fixtureServer(t, "1", "3")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + fixturePath
	config := writeDelegatedConfig(t, c)
	for _, first := range []string{"omitted", "0", "1"} {
		t.Run(first, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "outing.db")
			base := []string{"start", "--db", path, "--config", config, "--json"}
			withWeather := func(allowance string) []string {
				args := append(slices.Clone(base), "--weather", delegatedWeather)
				if allowance == "omitted" {
					return args
				}
				return append(args, "--helper-admission", allowance)
			}
			before := plannerOutput(t, withWeather(first)...)
			want := "1"
			if first == "0" {
				want = "0"
			}
			if !strings.Contains(before, `"helper_admission": `+want) {
				t.Fatalf("missing retained helper admission %s", want)
			}
			dbBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, attempt := range []string{"omitted", "0", "1", "2", "-1", "legacy", "legacy with 0", "legacy with 1"} {
				t.Run(attempt, func(t *testing.T) {
					args := withWeather(attempt)
					switch attempt {
					case "legacy":
						args = slices.Clone(base)
					case "legacy with 0", "legacy with 1":
						args = append(slices.Clone(base), "--helper-admission", attempt[len(attempt)-1:])
					}
					var output strings.Builder
					err := runCLI(t.Context(), args, &output)
					matches := attempt == want || attempt == "omitted" && want == "1"
					if matches && (err != nil || output.String() != before) || !matches && err == nil {
						t.Fatalf("repeat start: equal report %v, error %v", output.String() == before, err)
					}
					after, readErr := os.ReadFile(path)
					if readErr != nil || !bytes.Equal(dbBytes, after) || plannerOutput(t, "inspect", "--db", path) != before {
						t.Fatalf("repeat start changed Store: %v", readErr)
					}
				})
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("starting contacted the forecast endpoint %d times", calls.Load())
	}
}

func TestDelegatedCLIRejectsUnsupportedFreshStarts(t *testing.T) {
	server, calls := fixtureServer(t, "1", "3")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + fixturePath
	c.Slots = []string{"10:00", "11:00"}
	config := writeDelegatedConfig(t, c)
	for _, test := range []struct {
		name, want string
		flags      []string
	}{
		{"too many eligible", "at most two eligible candidates", []string{"--weather", delegatedWeather}},
		{"unknown weather", "unsupported Weather", []string{"--weather", "other"}},
		{"invalid allowance", "must be 0 or 1", []string{"--weather", delegatedWeather, "--helper-admission", "2"}},
		{"legacy explicit zero", "requires delegated Weather", []string{"--helper-admission", "0"}},
		{"legacy explicit one", "requires delegated Weather", []string{"--weather", confirmFirst, "--helper-admission", "1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			err := runCLI(t.Context(), append([]string{"start", "--config", config, "--db", filepath.Join(dir, "outing.db")}, test.flags...), new(strings.Builder))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("want %q, got %v", test.want, err)
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != 0 {
				t.Fatalf("unsupported start created evidence: %v, %v", files, err)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected starts contacted the endpoint %d times", calls.Load())
	}
}

func TestDelegatedCLIResumeRetainsRefusal(t *testing.T) {
	server, calls := fixtureServer(t, "1", "3")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + fixturePath
	config := writeDelegatedConfig(t, c)
	path := filepath.Join(t.TempDir(), "outing.db")
	plannerOutcome(t, plannerOutput(t, "start", "--db", path, "--config", config, "--weather", delegatedWeather,
		"--helper-admission", "0", "--json"), pendingOutcome)
	plannerOutcome(t, plannerOutput(t, "resume", "--db", path, "--json"), helperRefused)
	inspection := plannerOutput(t, "inspect", "--db", path)
	for _, command := range []string{"run", "resume", "status", "inspect", "replay"} {
		plannerOutcome(t, plannerOutput(t, command, "--db", path, "--json"), helperRefused)
		if err := runCLI(t.Context(), []string{command, "--db", path, "--helper-admission", "1"}, new(strings.Builder)); err == nil {
			t.Errorf("%s accepted an override to retained helper allowance", command)
		}
	}
	if calls.Load() != 0 || !strings.Contains(inspection, "invocation_limit") || !strings.Contains(inspection, `"Kind": "waiting_message"`) {
		t.Fatalf("refusal did not retain the Outing wait and zero contacts (%d)", calls.Load())
	}
}
