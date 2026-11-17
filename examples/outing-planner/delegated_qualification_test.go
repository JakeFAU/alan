//go:build integration

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// The raw checker independently validates both cuts. This additional comparison
// binds their immutable rows, including paid charge/fact prefixes, across reopen.
func requireDelegatedRawPrefix(t *testing.T, before, after string) {
	t.Helper()
	program := `import sqlite3,sys,json
from pathlib import Path
def read(p):
 with sqlite3.connect(Path(p).resolve().as_uri()+'?mode=ro',uri=True) as db:
  return {t:[tuple(r) for r in db.execute('SELECT * FROM "'+t+'"')] for t, in db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
a,b=map(read,sys.argv[1:])
fixed=('metadata','programs','composition_deployment','composition_profile','composition_roots','composition_static_members','composition_templates','run_origins')
prefix=('history','intents','attempts','adapter_attempts','adapter_dispatch_admissions','adapter_results','settlements','admissions','message_acknowledgments','composition_charges','composition_facts','composition_admission_decisions','composition_invocations','composition_bindings','composition_members','composition_requests','composition_replies','composition_reservations','composition_freezes','composition_publications','composition_results','composition_outcome_deliveries','composition_root_publications')
for t in fixed: assert sorted(a[t])==sorted(b[t]),('frozen inventory changed',t)
for t in prefix: assert all(r in b[t] for r in a[t]),('retained row prefix changed',t)
print(json.dumps({'frozen_tables':list(fixed),'prefix_tables':list(prefix),'retained_charges':len(a['composition_charges']),'final_charges':len(b['composition_charges']),'preserved':True}))`
	output := rawModelCommand(t, "-c", program, before, after)
	if err := os.WriteFile(strings.TrimSuffix(after, ".db")+"-prefix.json", output, 0600); err != nil {
		t.Fatal(err)
	}
}

// This uses the real compiled CLI's replacement export, including its exact
// original IDs and separate costs, as an optional cross-check of both raw Stores.
func TestDelegatedQualificationCLIReplacement(t *testing.T) {
	binary := publicOutingBinary(t)
	for _, tc := range []struct {
		name      string
		allowance int64
		positive  bool
		outcome   string
	}{
		{"accepted", 1, true, recommended}, {"refused", 0, false, helperRefused},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if root := os.Getenv("ALAN_RAW_CAPTURE_DIR"); root != "" {
				if err := os.MkdirAll(root, 0700); err != nil {
					t.Fatal(err)
				}
				var err error
				dir, err = os.MkdirTemp(root, "cli-replacement-"+tc.name+"-")
				if err != nil {
					t.Fatal(err)
				}
			}
			pair := filepath.Join(dir, "pair")
			command := exec.CommandContext(t.Context(), binary, "check-replacement", "--config", "testdata/configuration.json", "--fixture", "testdata/fixture.json", "--weather", delegatedWeather, "--helper-admission", strconv.FormatInt(tc.allowance, 10), "--output-dir", pair)
			output, err := command.CombinedOutput()
			recordOutingCommand(t, command.Args, output, err)
			if err != nil {
				t.Fatalf("actual CLI replacement: %v\n%s", err, output)
			}
			var report replacementReport
			if err := json.Unmarshal(output, &report); err != nil {
				t.Fatal(err)
			}
			if report.Delegated == nil || report.Delegated.PositiveReplacement != tc.positive || report.Candidate.Outcome != tc.outcome || report.Baseline.Outcome != recommended || report.Baseline.FixtureRequests != 2 {
				t.Fatalf("wrong actual replacement outcome: %s", output)
			}
			candidate := report.Candidate.Report
			if !tc.positive {
				if candidate.Delegated == nil || candidate.Delegated.HelperRun != nil || len(candidate.Delegated.Invocations) != 1 || candidate.Delegated.Invocations[0].Admission == nil || candidate.Delegated.Invocations[0].Admission.Accepted ||
					len(candidate.Runs) != 2 || len(candidate.Effects) != 0 || report.Candidate.FixtureRequests != 0 || candidate.Assessment != nil || candidate.Decision != nil || candidate.Recommendation != nil ||
					report.Delegated.CandidateCost.Invocations != 0 || report.Delegated.CandidateCost.CreatedRuns != 0 || report.Delegated.CandidateCost.Effects != 0 {
					t.Fatal("refusal invented a helper, forecast contact, debit or public answer")
				}
			} else if len(candidate.Runs) != 3 || len(candidate.Effects) != 2 || report.Candidate.FixtureRequests != 2 {
				t.Fatal("accepted CLI pair lacks one helper and two original GETs")
			}
			for _, name := range []string{"baseline", "candidate"} {
				original := filepath.Join(pair, name+".db")
				before, err := os.ReadFile(original)
				if err != nil {
					t.Fatal(err)
				}
				// The owner exited. The existing public read-only reader creates
				// empty WAL/read-side SHM without opening an owner or changing epochs.
				inspected := outingProcessJSON(t, binary, "inspect", original)
				var fresh plannerReport
				if err := json.Unmarshal(inspected, &fresh); err != nil {
					t.Fatal(err)
				}
				expected := report.Baseline.Report
				if name == "candidate" {
					expected = report.Candidate.Report
				}
				if fresh.Namespace != expected.Namespace || !reflect.DeepEqual(fresh.Runs, expected.Runs) {
					t.Fatal("actual export differs from final read-only runs/epochs")
				}
				sidecars := map[string]int64{}
				for _, suffix := range []string{"-wal", "-shm"} {
					info, err := os.Stat(original + suffix)
					if err == nil {
						sidecars[suffix] = info.Size()
					} else if !os.IsNotExist(err) {
						t.Fatal(err)
					}
				}
				rawModelCommand(t, "model/test_evidence.py", "capture", original, filepath.Join(pair, name+"-captured.db"))
				after, err := os.ReadFile(original)
				if err != nil {
					t.Fatal(err)
				}
				digest := sha256.Sum256(before)
				if digest != sha256.Sum256(after) || string(inspected) != string(outingProcessJSON(t, binary, "inspect", original)) {
					t.Fatal("read-only preparation/capture changed original bytes or logical report")
				}
				raw, _ := json.MarshalIndent(map[string]any{"original_main_sha256": fmt.Sprintf("%x", digest), "original_main_unchanged": true, "runs_epochs_and_projections_unchanged": true, "namespace": fresh.Namespace, "reader_sidecar_bytes": sidecars, "method": "CLI owner exited; public read-only inspect supplies reader sidecars; mode=ro transactional backup verifies full schema/row equality; no owner reopen"}, "", "  ")
				if err := os.WriteFile(filepath.Join(pair, name+"-read-provenance.json"), append(raw, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			}
			rawModelCommand(t, "model/run.py", "compare", "--baseline", filepath.Join(pair, "baseline-captured.db"), "--candidate", filepath.Join(pair, "candidate-captured.db"), "--configuration", filepath.Join(pair, "derived-configuration.json"), "--fixture", filepath.Join(pair, "fixture.json"), "--manifest", filepath.Join(pair, "deployment-manifest.json"), "--replacement", filepath.Join(pair, "replacement.json"), "--output", filepath.Join(pair, "independent-comparison.json"))
			t.Logf("actual CLI %s replacement capture: %s", tc.name, pair)
		})
	}
}

// Retain argv, exit and bytes for fresh public commands during qualification.
func recordOutingCommand(t *testing.T, args []string, output []byte, runErr error) {
	t.Helper()
	root := os.Getenv("ALAN_RAW_CAPTURE_DIR")
	if root == "" {
		return
	}
	dir, err := os.MkdirTemp(root, "command-")
	if err != nil {
		t.Fatal(err)
	}
	code := 0
	if runErr != nil {
		code = -1
		if e, ok := runErr.(*exec.ExitError); ok {
			code = e.ExitCode()
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"argv": args, "exit": code}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "command.json"), append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "output.json"), output, 0600); err != nil {
		t.Fatal(err)
	}
}
