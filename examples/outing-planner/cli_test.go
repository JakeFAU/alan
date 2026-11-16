package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/runtime"
)

func plannerOutput(t *testing.T, args ...string) string {
	t.Helper()
	var out strings.Builder
	if err := runCLI(t.Context(), args, &out); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	return out.String()
}

func plannerOutcome(t *testing.T, output, want string) {
	t.Helper()
	var result struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Outcome != want {
		t.Fatalf("expected outcome %q, got %q", want, result.Outcome)
	}
}

func requireReportedEffect(t *testing.T, output, operationID string) {
	t.Helper()
	var result struct {
		Effects []struct{ OperationID string } `json:"effects"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(result.Effects, func(effect struct{ OperationID string }) bool {
		return effect.OperationID == operationID
	}) {
		t.Fatalf("report lost original forecast operation %s", operationID)
	}
}

func TestPlannerCLIRecommendation(t *testing.T) {
	server, calls := fixtureServer(t, "999", "2")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + "/v1/forecast"
	_, path := executeFixture(t, c, assessFirst)
	human := plannerOutput(t, "status", "--db", path)
	for _, want := range []string{
		"Hemlock Park", "2026-10-11", "America/New_York", "historical", "12 minutes",
		"Open-Meteo", "17.2", "18.3", "Cedar Park", "unavailable",
	} {
		if !strings.Contains(human, want) {
			t.Errorf("human recommendation lacks %q", want)
		}
	}
	detailed := plannerOutput(t, "status", "--db", path, "--json")
	plannerOutcome(t, detailed, recommended)
	for _, want := range []string{`"proposal"`, `"assessment"`, `"decision"`, `"recommendation"`, `"runs"`, `"effects"`, `"observations"`, `"pending"`, `"SelectedAt"`, `"historical": true`} {
		if !strings.Contains(detailed, want) {
			t.Errorf("detailed recommendation lacks %s", want)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("unexpected contact count %d", calls.Load())
	}
}

func TestPlannerCLISeparatesOutcomes(t *testing.T) {
	for _, tc := range []struct{ name, cedar, hemlock, outcome string }{
		{"both unsuitable", "61", "45", noMatch},
		{"unavailable evidence", "999", "45", unavailable},
		{"provisioned", "0", "0", "pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := fixtureServer(t, tc.cedar, tc.hemlock)
			c := fixtureConfiguration()
			c.Endpoint = server.URL + "/v1/forecast"
			var path string
			if tc.outcome == "pending" {
				path = filepath.Join(t.TempDir(), "pending.db")
				if err := createPlanner(t.Context(), path, c, assessFirst); err != nil {
					t.Fatal(err)
				}
			} else {
				_, path = executeFixture(t, c, assessFirst)
			}
			plannerOutcome(t, plannerOutput(t, "status", "--db", path, "--json"), tc.outcome)
			if tc.outcome == "pending" && calls.Load() != 0 {
				t.Fatal("status contacted the fixture")
			}
		})
	}
}

func TestPlannerCLIOfflineReaders(t *testing.T) {
	server, calls := fixtureServer(t, "0", "2")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + "/v1/forecast"
	before, path := executeFixture(t, c, assessFirst)
	for _, action := range []string{"status", "inspect", "replay"} {
		report := plannerOutput(t, action, "--db", path, "--json")
		plannerOutcome(t, report, recommended)
		requireReportedEffect(t, report, before.Effects[0].OperationID)
	}
	after, _, _, err := inspectPlanner(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range after.Runs {
		prior, _ := before.Run(run.ID)
		if run.Epoch != prior.Epoch || run.Sequence != prior.Sequence {
			t.Fatalf("reader changed %s", run.ID.Subject)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("readers contacted fixture, %d calls", calls.Load())
	}
}

func TestPlannerCLIUnknownForecast(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	c := fixtureConfiguration()
	c.Endpoint = server.URL + "/v1/forecast"
	before, path := executeFixture(t, c, assessFirst)
	if cycleAt(before, runtime.RunProgress{Quiescent: true}) != nil {
		t.Fatal("unknown forecast was misdiagnosed as a cycle")
	}
	for _, command := range []string{"status", "inspect", "replay", "resume"} {
		output := plannerOutput(t, command, "--db", path, "--json")
		plannerOutcome(t, output, "unknown_forecast")
		requireReportedEffect(t, output, before.Effects[0].OperationID)
	}
	if calls.Load() != 1 {
		t.Fatalf("unknown operation was resent, %d calls", calls.Load())
	}
}

func TestPlannerCLIArguments(t *testing.T) {
	for _, args := range [][]string{
		nil, {"bad"}, {"start"}, {"status", "--new"}, {"status", "--db", "bad", "extra"},
		{"check-replacement", "--config", "source.json", "--weather", confirmFirst},
	} {
		if err := runCLI(t.Context(), args, new(strings.Builder)); err == nil {
			t.Errorf("accepted arguments %v", args)
		}
	}
	for _, args := range [][]string{{"--help"}, {"inspect", "--help"}, {"check-replacement", "--help"}} {
		if text := plannerOutput(t, args...); text == "" {
			t.Errorf("no help for arguments %v", args)
		}
	}
}

func TestPlannerCLIStartRecognizesRetainedDeployment(t *testing.T) {
	server, calls := fixtureServer(t, "1", "0")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + fixturePath
	c.ReferenceTime = ""
	raw, err := c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	configPath, path := filepath.Join(dir, "configuration.json"), filepath.Join(dir, "outing.db")
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	// Two starts on opposite sides of local midnight use the same retained
	// date and never contact the configured provider.
	beforeMidnight := time.Date(2026, time.October, 11, 3, 59, 0, 0, time.UTC)
	before, _, err := startPlanner(t.Context(), path, configPath, assessFirst, beforeMidnight)
	if err != nil {
		t.Fatal(err)
	}
	after, saved, err := startPlanner(t.Context(), path, configPath, assessFirst, beforeMidnight.Add(2*time.Minute))
	if err != nil || after.Namespace != before.Namespace || saved.ReferenceTime != "2026-10-11T03:59:00Z" {
		t.Fatalf("repeated start changed the retained session or reference instant: %+v %v", saved, err)
	}
	args := []string{"start", "--db", path, "--config", configPath, "--json"}
	output := plannerOutput(t, args...)
	plannerOutcome(t, output, "pending")
	if !strings.Contains(output, before.Namespace) || !strings.Contains(output, "2026-10-11") {
		t.Fatal("repeated start lost the reference date or namespace")
	}
	if err := runCLI(t.Context(), append(args, "--weather", confirmFirst), new(strings.Builder)); err == nil {
		t.Fatal("accepted conflicting Weather variant")
	}
	c.ReferenceTime = "2026-10-11T00:01:00-04:00"
	raw, err = c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCLI(t.Context(), args, new(strings.Builder)); err == nil {
		t.Fatal("accepted conflicting explicit reference")
	}
	if calls.Load() != 0 {
		t.Fatalf("provisioning contacted forecast endpoint %d times", calls.Load())
	}
}
