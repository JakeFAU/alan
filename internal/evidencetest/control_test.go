//go:build integration

package evidencetest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

// monitorReplay is unchanged for both the correct replay and deliberately faulty
// wrapper. It observes the independently retained provider contacts and complete
// durable runtime snapshot, not a claimed invocation counter from the wrapper.
func monitorReplay(before, after evidence.View, old, new []providerEvent) error {
	if !reflect.DeepEqual(old, new) {
		return fmt.Errorf("replay dispatched provider contact: before=%d after=%d", contacts(old), contacts(new))
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("replay mutated runtime evidence")
	}
	return nil
}
func TestFaultyReplayDispatchIsDetected(t *testing.T) {
	p := startProvider(t, false)
	db, _ := initialize(t, p)
	invoke(t, true, "run", "--db", db)
	before := inspect(t, db)
	old := p.events(t)
	invoke(t, false, "replay", "--db", db)
	if err := monitorReplay(before, inspect(t, db), old, p.events(t)); err != nil {
		t.Fatal(err)
	}
	var original runtime.AdapterAttempt
	for _, r := range before.Results {
		a := attempt(t, before, r.Candidate.ID)
		if a.Prepared.Request.Capability == evidence.ModelCapability {
			original = a
			break
		}
	}
	// This fault is a real dispatch after read-only replay. It intentionally calls
	// the real HTTP adapter with the original recorded prepared request, bypassing
	// runtime closed-source guards. The independent monitor must catch the contact.
	if _, err := runtime.Replay(t.Context(), db, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	adapter, err := openai.New(openai.Config{APIKey: "synthetic-process-fixture", Model: before.Config.Model, Endpoint: p.endpoint, MaxOutputTokens: before.Config.MaxOutputTokens, MaxResponseBytes: before.Config.MaxResponseBytes})
	if err != nil {
		t.Fatal(err)
	}
	candidate := adapter.Execute(t.Context(), original.Prepared)
	if candidate.Disposition != "validated" {
		t.Fatalf("fault did not dispatch successfully: %+v", candidate)
	}
	newEvents := p.events(t)
	err = monitorReplay(before, inspect(t, db), old, newEvents)
	if err == nil || contacts(newEvents) != contacts(old)+1 {
		t.Fatal("monitor failed to detect actual replay contact")
	}
	assertContact(t, original, newEvents[len(newEvents)-2])
	emit(t, map[string]any{"scenario": t.Name(), "type": "negative-control", "fault": "test-only replay wrapper directly executes original real-adapter request", "monitor": "monitorReplay unchanged", "detected": err.Error(), "before": before, "after": inspect(t, db), "contacts_before": old, "contacts_after": newEvents, "original_attempt": original, "candidate": candidate, "actual_extra_contacts": 1})
}

func TestAcknowledgmentRejectsIdentitySubstitution(t *testing.T) {
	want := boundary.Event{Scenario: "case", Token: "token", Store: "/store", PID: 17, Boundary: "adapter.result-recorded", Subject: "assistant", Phase: "pre-settlement", Generation: 1, Epoch: 2, Sequence: 15, Intent: "intent", Attempt: "attempt"}
	if err := validateAcknowledgment(want, []boundary.Event{want}, 17, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*boundary.Event){"PID": func(e *boundary.Event) { e.PID++ }, "store": func(e *boundary.Event) { e.Store += "other" }, "epoch": func(e *boundary.Event) { e.Epoch++ }, "generation": func(e *boundary.Event) { e.Generation++ }, "sequence": func(e *boundary.Event) { e.Sequence++ }, "intent": func(e *boundary.Event) { e.Intent += "other" }, "attempt": func(e *boundary.Event) { e.Attempt += "other" }, "phase": func(e *boundary.Event) { e.Phase = "post-consume" }, "subject": func(e *boundary.Event) { e.Subject = "other" }, "token": func(e *boundary.Event) { e.Token = "other" }, "scenario": func(e *boundary.Event) { e.Scenario = "other" }}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			got := want
			mutate(&got)
			err := validateAcknowledgment(got, []boundary.Event{want}, 17, map[string]bool{})
			if err == nil {
				t.Fatal("identity substitution accepted")
			}
			emit(t, map[string]any{"type": "identity-control", "field": name, "original": want, "mutated": got, "detected": err.Error()})
		})
	}
	used := map[string]bool{}
	if err := validateAcknowledgment(want, []boundary.Event{want}, 17, used); err != nil {
		t.Fatal(err)
	}
	if err := validateAcknowledgment(want, []boundary.Event{want}, 17, used); err == nil {
		t.Fatal("duplicate acknowledgment accepted")
	}
}

// The fixture is the actual stopped C-model evidence from the independent
// review. Its filter deliberately leaves Sequence unspecified, just as the
// live crash test does. Both paths invoke the same durable correlation helper.
func TestWildcardAcknowledgmentRejectsRetainedSequenceSubstitution(t *testing.T) {
	raw, err := os.ReadFile("testdata/actual-c-model-boundary.json")
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		Filter       boundary.Event     `json:"actual_filter"`
		Original     boundary.Event     `json:"original_acknowledgment"`
		Substituted  boundary.Event     `json:"substituted_acknowledgment"`
		Snapshot     runtime.Inspection `json:"snapshot"`
		SourceSHA256 string             `json:"source_sha256"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if input.Filter.Sequence != 0 {
		t.Fatal("control must use actual wildcard sequence filter")
	}
	for _, tc := range []struct {
		name   string
		event  boundary.Event
		reject bool
	}{{"original", input.Original, false}, {"substituted", input.Substituted, true}} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateAcknowledgment(tc.event, []boundary.Event{input.Filter}, input.Original.PID, map[string]bool{}); err != nil {
				t.Fatalf("wildcard matcher unexpectedly rejected: %v", err)
			}
			err := validateDurableBoundary(input.Snapshot, tc.event)
			if tc.reject && err == nil {
				t.Fatal("actual wildcard path accepted sequence substitution against retained operation")
			}
			if !tc.reject && err != nil {
				t.Fatal(err)
			}
			emit(t, map[string]any{"type": "durable-sequence-control", "case": tc.name, "filter": input.Filter, "acknowledgment": tc.event, "source_sha256": input.SourceSHA256, "durable_tables": input.Snapshot.Tables, "rejected": err != nil, "result": fmt.Sprint(err)})
		})
	}
}
