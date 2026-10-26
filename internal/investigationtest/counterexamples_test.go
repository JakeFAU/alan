//go:build integration

package investigationtest

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

// Original live-process counterexample inputs are immutable regression fixtures.
// These are monitor tests, not additional execution/crash schedules.
func TestRetainedCounterexamples(t *testing.T) {
	raw, err := os.ReadFile("testdata/retained-counterexamples.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Schema string
		Cases  []struct {
			Monitor, Field   string
			Filter           boundary.Event
			Original, Mutant boundary.Event
			Snapshot         runtime.Inspection
			Runs             []runtime.RunState
			OriginalRuns     []runtime.RunState `json:"original_runs"`
			MutantRuns       []runtime.RunState `json:"mutant_runs"`
			LedgerBefore     []providerEvent    `json:"ledger_before"`
			LedgerAfter      []providerEvent    `json:"ledger_after"`
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != "phase1f-process-counterexamples/v1" || len(fixture.Cases) != 9 {
		t.Fatal("wrong retained counterexample fixture")
	}
	for _, c := range fixture.Cases {
		t.Run(c.Field, func(t *testing.T) {
			var good, bad error
			switch c.Monitor {
			case "validateDurableBoundary":
				if err := validateAcknowledgment(c.Original, []boundary.Event{c.Filter}, c.Original.PID, map[string]bool{}); err != nil {
					t.Fatal(err)
				}
				matchErr := validateAcknowledgment(c.Mutant, []boundary.Event{c.Filter}, c.Original.PID, map[string]bool{})
				if c.Field == "source" {
					if matchErr == nil {
						t.Fatal("source filter accepted mutant")
					}
				} else if matchErr != nil {
					t.Fatal("mutant rejected by unrelated matcher guard", matchErr)
				}
				good = validateDurableBoundary(c.Snapshot, c.Original)
				bad = validateDurableBoundary(c.Snapshot, c.Mutant)
			case "validateActiveBoundary":
				v := investigation.View{Snapshot: c.Snapshot, Runs: c.Runs}
				if err := validateDurableBoundary(c.Snapshot, c.Mutant); err != nil {
					t.Fatal("mutant must remain genuine historical attempt", err)
				}
				good = validateActiveBoundary(v, c.Original)
				bad = validateActiveBoundary(v, c.Mutant)
			case "monitorPreserved":
				before := investigation.View{Runs: c.OriginalRuns}
				good = monitorPreserved(before, before)
				bad = monitorPreserved(before, investigation.View{Runs: c.MutantRuns})
			case "monitorLate":
				before := investigation.View{Runs: c.OriginalRuns}
				good = monitorLate(before, before)
				bad = monitorLate(before, investigation.View{Runs: c.MutantRuns})
			case "monitorReplay":
				good = monitorReplay(investigation.View{}, investigation.View{}, c.LedgerBefore, c.LedgerBefore)
				bad = monitorReplay(investigation.View{}, investigation.View{}, c.LedgerBefore, c.LedgerAfter)
			default:
				t.Fatal("unknown monitor", c.Monitor)
			}
			if good != nil {
				t.Fatal("positive original rejected", good)
			}
			if bad == nil {
				t.Fatal("retained mutation escaped unchanged monitor")
			}
			emit(t, map[string]any{"type": "retained-counterexample", "field": c.Field, "monitor": c.Monitor, "fixture_sha256": fileHash("testdata/retained-counterexamples.json"), "rejection": bad.Error()})
		})
	}
}
