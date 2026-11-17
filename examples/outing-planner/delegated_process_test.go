//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

var delegatedProcessCuts = []string{"proposal", "invocation", "request", "result", "settled", "second_forecast", "helper_terminal", "published", "assessment", "decision", "decision_authorized", "decision_admitted"}

func delegatedCutMatches(name string, event boundary.Event, cut *runtime.ExecutionSnapshot) bool {
	op, _ := cut.Operation(event.Intent)
	switch name {
	case "proposal":
		return event.Boundary == "execution.committed" && op != nil && op.Obligation == proposalPort
	case "invocation":
		return event.Boundary == "execution.committed" && op != nil && op.Kind == "invocation"
	case "request":
		return event.Boundary == "invocation.admitted"
	case "result":
		return event.Boundary == "adapter.result-recorded"
	case "settled":
		return event.Boundary == "adapter.settled"
	case "second_forecast":
		return event.Boundary == "execution.committed" && op != nil && op.Kind == "adapter" && len(cut.Effects) == 2
	case "helper_terminal":
		return event.Boundary == "execution.committed" && event.Phase == "terminal" && event.Subject != outingSubject && event.Subject != weatherSubject
	case "published":
		return event.Boundary == "invocation.result-recorded"
	case "assessment":
		return event.Boundary == "execution.committed" && op != nil && op.Obligation == assessmentPort
	case "decision":
		return event.Boundary == "execution.committed" && op != nil && op.Obligation == decisionPort
	case "decision_authorized":
		return event.Boundary == "intent.authorized" && op != nil && op.Obligation == decisionPort
	case "decision_admitted":
		weather, _ := cut.Run(weatherRun)
		return event.Boundary == "execution.staged" && event.Subject == weatherSubject && weather.Wait != nil && weather.Wait.Obligation == decisionPort && len(weather.Wait.MatchingObservationIDs) == 1
	}
	return false
}

// Baseline supplies only static program coordinates, never target Store IDs or
// a verdict about the actual interrupted process. Every target is checked below.
func delegatedCutCoordinates(t *testing.T) map[string]boundary.Event {
	t.Helper()
	f := newRawModelFixture(t, 2, defaultDeploymentOptions(delegatedWeather), []int{1, 3}, false)
	owner, path := f.create(t, true)
	events := map[string]boundary.Event{}
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, event boundary.Event) error {
		if len(events) == len(delegatedProcessCuts) {
			return nil
		}
		relevant := false
		switch event.Boundary {
		case "invocation.admitted", "invocation.result-recorded", "adapter.result-recorded", "adapter.settled":
			relevant = true
		case "execution.committed":
			relevant = event.Intent != "" || event.Phase == "terminal"
		case "execution.staged":
			_, decisionCommitted := events["decision"]
			relevant = event.Subject == weatherSubject && decisionCommitted
		case "intent.authorized":
			_, decisionCommitted := events["decision"]
			relevant = event.Subject == outingSubject && decisionCommitted
		}
		if !relevant {
			return nil
		}
		cut, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		for _, name := range delegatedProcessCuts {
			if _, exists := events[name]; !exists && delegatedCutMatches(name, event, cut) {
				events[name] = event
			}
		}
		return nil
	})
	if err := f.run(ctx, owner); err != nil {
		t.Fatal(err)
	}
	for _, name := range delegatedProcessCuts {
		event, ok := events[name]
		if !ok {
			t.Fatalf("baseline missing %s", name)
		}
		// Only newly created helper runs retain epoch 1. Existing root runs reopen.
		if event.Subject == outingSubject || event.Subject == weatherSubject {
			if event.Epoch != 0 {
				event.Epoch++
			}
		} else {
			event.Subject = ""
		}
		event.Intent, event.Attempt, event.Root = "", "", ""
		events[name] = event
	}
	return events
}

func requireDelegatedCut(t *testing.T, name string, event boundary.Event, s *runtime.ExecutionSnapshot) {
	t.Helper()
	if !delegatedCutMatches(name, event, s) {
		t.Fatalf("acknowledged event is not the requested %s cut: %+v", name, event)
	}
	run, ok := s.Run(runtime.RunID{Subject: event.Subject, Generation: event.Generation})
	if !ok {
		t.Fatal("acknowledgment names a foreign logical run")
	}
	if event.Boundary == "invocation.admitted" || event.Boundary == "invocation.result-recorded" {
		if len(s.Invocations) != 1 {
			t.Fatal("missing original invocation")
		}
		call := s.Invocations[0]
		if call.Caller != run.ID || event.Sequence != call.Sequence || event.Epoch != 0 || event.Intent != call.OperationID {
			t.Fatalf("wrong invocation coordinates: %+v %+v", event, call)
		}
		if call.Admission == nil || event.Root != call.Admission.Invocation.RootID || name == "request" && event.Attempt != call.Admission.Invocation.ID {
			t.Fatal("wrong invocation resource root")
		}
		if name == "published" {
			obs, found := s.Observation(call.OutcomeObservationID)
			if call.Outcome == nil || event.Attempt != call.Outcome.ID || !found || obs.SelectedAt != nil || call.SelectedAt != nil || call.Freeze == nil || len(call.Replies) != 1 || call.ResponseOperationID != call.Replies[0].Intent.ID {
				t.Fatal("publication did not precede exact parent selection")
			}
		}
	} else if event.Boundary == "adapter.result-recorded" || event.Boundary == "adapter.settled" {
		effect, found := s.Effect(event.Intent)
		if !found || effect.Source != run.ID || effect.Sequence != event.Sequence || effect.AdapterAttempt == nil || effect.Authorization == nil || event.Root != "" || event.Attempt != effect.AdapterAttempt.AuthorizationID || event.Epoch != effect.Authorization.Source.Epoch || effect.AdapterResult == nil {
			t.Fatalf("wrong adapter coordinates: %+v %+v", event, effect)
		}
		if name == "result" && effect.Settlement != nil {
			t.Fatal("result cut already settled")
		}
		if name == "settled" {
			if effect.Settlement == nil {
				t.Fatal("missing settlement")
			}
			obs, _ := s.Observation(effect.Settlement.ObservationID)
			if obs == nil || obs.SelectedAt != nil {
				t.Fatal("settled result already selected")
			}
		}
	} else {
		if event.Epoch != run.Epoch {
			t.Fatal("ordinary boundary names wrong epoch")
		}
		if event.Boundary == "execution.committed" && event.Phase == "post-commit" {
			op, _ := s.Operation(event.Intent)
			if op == nil {
				t.Fatal("missing acknowledged operation")
			}
			requireOutingCommit(t, s, event, *op)
		}
		if name == "decision_admitted" {
			weather, _ := s.Run(weatherRun)
			obs, _ := s.Observation(weather.Wait.MatchingObservationIDs[0])
			if obs == nil || obs.SelectedAt != nil || event.Sequence != weather.Sequence+1 {
				t.Fatal("decision receive was already committed")
			}
		}
	}
	switch name {
	case "proposal", "assessment", "decision":
		op, _ := s.Operation(event.Intent)
		for _, admission := range s.Admissions {
			if admission.SourceOperationID == op.ID {
				t.Fatal("original message was already admitted")
			}
		}
		for _, obs := range s.Observations {
			if obs.SourceOperationID == op.ID {
				t.Fatal("original message was already observed")
			}
		}
		if name == "assessment" && s.Invocations[0].SelectedAt == nil {
			t.Fatal("parent assessed before selecting helper result")
		}
	case "invocation":
		call := s.Invocations[0]
		if call.Authorization != nil || call.Admission != nil || call.Outcome != nil || len(s.Runs) != 2 {
			t.Fatal("invocation already admitted")
		}
	case "request":
		call := s.Invocations[0]
		helper, _ := s.Run(call.Admission.Members[0])
		obs, _ := s.Observation(call.RequestObservationID)
		if !call.Admission.Accepted || helper.Sequence != 0 || obs == nil || obs.SelectedAt != nil || len(s.Effects) != 0 {
			t.Fatal("request already selected or helper advanced")
		}
	case "second_forecast":
		effect, _ := s.Effect(event.Intent)
		if effect == nil || effect.Authorization != nil || effect.AdapterAttempt != nil || effect.DispatchAdmission != nil || effect.AdapterResult != nil || effect.Settlement != nil {
			t.Fatal("second original request already dispatched")
		}
		for _, first := range s.Effects {
			if first.OperationID != effect.OperationID {
				requireOutingSelection(t, s, first.OperationID, effect.Source)
			}
		}
	case "helper_terminal":
		call := s.Invocations[0]
		if run.Projection.Status != "succeeded" || event.Sequence != run.Sequence || call.Freeze == nil || call.Freeze.Sequence != event.Sequence || call.Outcome != nil || call.OutcomeObservationID != "" || len(call.Replies) != 1 {
			t.Fatal("terminal cut lacks exact frozen reply or already published")
		}
	case "decision_authorized":
		effectID := event.Intent
		for _, admission := range s.Admissions {
			if admission.SourceOperationID == effectID {
				t.Fatal("decision already admitted")
			}
		}
		canonical, _ := json.Marshal([]any{"authorization", event.Intent})
		if event.Attempt != string(canonical) || event.Sequence != run.Sequence || run.Projection.Status != "open" {
			t.Fatal("decision first authorization not open")
		}
	}
	if name == "decision" {
		for _, id := range []runtime.RunID{outingRun, weatherRun} {
			r, _ := s.Run(id)
			if r.Projection.Status != "open" {
				t.Fatal("decision endpoint closed at commit")
			}
		}
	}
	t.Logf("actual cut=%s event=%+v namespace=%s", name, event, s.Namespace)
}

func requireDelegatedRecovery(t *testing.T, before, after *runtime.ExecutionSnapshot) {
	t.Helper()
	if before.Namespace != after.Namespace || !reflect.DeepEqual(before.Deployment, after.Deployment) || !reflect.DeepEqual(before.Composition, after.Composition) {
		t.Fatal("recovery changed frozen deployment")
	}
	for _, a := range before.Runs {
		b, ok := after.Run(a.ID)
		epoch := a.Epoch
		if a.Projection.Status == "open" {
			epoch++
		}
		if !ok || b.Epoch != epoch || b.ProgramID != a.ProgramID || b.Revision != a.Revision || !reflect.DeepEqual(a.Initial, b.Initial) || b.Sequence < a.Sequence {
			t.Fatalf("identity/artifact/input/epoch changed for %v", a.ID)
		}
	}
	for _, p := range before.Permissions {
		if !slices.ContainsFunc(after.Permissions, func(q runtime.Permission) bool { return reflect.DeepEqual(p, q) }) {
			t.Fatal("recovery lost permission")
		}
	}
	a, b := outingHistory(t, before), outingHistory(t, after)
	if len(b) < len(a) || !reflect.DeepEqual(a, b[:len(a)]) {
		t.Fatal("recovery rewrote history prefix")
	}
	for _, op := range before.Operations {
		next, ok := after.Operation(op.ID)
		if !ok || !reflect.DeepEqual(op, *next) {
			t.Fatal("recovery replaced original operation")
		}
	}
	for _, call := range before.Invocations {
		next, ok := after.Invocation(call.OperationID)
		if !ok || call.Admission != nil && !reflect.DeepEqual(call.Admission, next.Admission) || !reflect.DeepEqual(call.Inputs, next.Inputs) || call.Outcome != nil && !reflect.DeepEqual(call.Outcome, next.Outcome) {
			t.Fatal("recovery replaced original helper admission/result")
		}
	}
	for _, root := range before.Roots {
		next, ok := after.Root(root.RootID)
		if !ok || next.Limits != root.Limits || next.Charges.Invocations < root.Charges.Invocations || next.Charges.CreatedRuns < root.Charges.CreatedRuns || next.Charges.Effects < root.Charges.Effects {
			t.Fatal("recovery lost original debit")
		}
	}
	for _, o := range before.Observations {
		next, ok := after.Observation(o.ID)
		if !ok {
			t.Fatal("recovery lost observation")
		}
		copy := *next
		if o.SelectedAt == nil {
			copy.SelectedAt = nil
		}
		if !reflect.DeepEqual(*o, copy) {
			t.Fatal("recovery changed original observation")
		}
	}
}

func delegatedReadOnly(t *testing.T, binary, path string, s *runtime.ExecutionSnapshot, endpoint *outingEndpoint, contacts ...string) {
	t.Helper()
	inspect := outingProcessJSON(t, binary, "inspect", path)
	if status := outingProcessJSON(t, binary, "status", path); string(status) != string(inspect) {
		t.Fatal("status differs from inspect")
	}
	var a, b plannerReport
	if err := json.Unmarshal(inspect, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(outingProcessJSON(t, binary, "replay", path), &b); err != nil {
		t.Fatal(err)
	}
	if b.Replay == nil || len(b.Replay.Runs) != len(s.Runs) {
		t.Fatal("replay omitted original runs")
	}
	b.Replay = nil
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(a.Runs, s.Runs) || !reflect.DeepEqual(a.History, outingHistory(t, s)) ||
		!slices.EqualFunc(a.Operations, s.Operations, func(a, b runtime.Intent) bool { return reflect.DeepEqual(a, b) }) ||
		!slices.EqualFunc(a.Effects, s.Effects, func(a, b *runtime.ExecutionEffect) bool { return reflect.DeepEqual(a, b) }) ||
		!slices.EqualFunc(a.Observations, s.Observations, func(a, b *runtime.ObservationView) bool { return reflect.DeepEqual(a, b) }) ||
		a.Delegated == nil || !slices.EqualFunc(a.Delegated.Invocations, s.Invocations, func(a, b *runtime.InvocationView) bool { return reflect.DeepEqual(a, b) }) || !reflect.DeepEqual(a.Delegated.Roots, s.Roots) {
		t.Fatal("offline readers disagree with complete snapshot")
	}
	if !reflect.DeepEqual(s, outingSnapshot(t, path)) {
		t.Fatal("offline readers mutated retained cut")
	}
	endpoint.requireContacts(t, s, contacts...)
}

func TestDelegatedProcessRecovery(t *testing.T) {
	binary := publicOutingBinary(t)
	coordinates := delegatedCutCoordinates(t)
	for _, name := range delegatedProcessCuts {
		t.Run(name, func(t *testing.T) {
			endpoint := newOutingEndpoint(t, false)
			f := newRawModelFixtureEndpoint(t, 2, defaultDeploymentOptions(delegatedWeather), []int{1, 3}, false, endpoint.url)
			owner, path := f.create(t, true)
			if err := owner.Close(); err != nil {
				t.Fatal(err)
			}
			filter := coordinates[name]
			process, event := launchOutingOwner(t, binary, "run", path, &filter)
			// Accounting barriers have a five-second context: kill immediately on ack.
			process.kill(t)
			cut := outingSnapshot(t, path)
			requireDelegatedCut(t, name, event, cut)
			n := len(cut.Effects)
			if name == "second_forecast" {
				n = 1
			}
			contacts := []string{"40.8", "40.7"}[:n]
			endpoint.requireContacts(t, cut, contacts...)
			cutPath := f.capture(t, path, "cut", "pending", true)
			if name == "decision" {
				raw, err := os.ReadFile(filepath.Join(f.dir, "cut-check.json"))
				if err != nil {
					t.Fatal(err)
				}
				var checked struct {
					DecisionCut struct {
						Verified bool `json:"verified"`
					} `json:"decision_commit_cut"`
				}
				if err := json.Unmarshal(raw, &checked); err != nil || !checked.DecisionCut.Verified {
					t.Fatalf("raw sole-pending/open-endpoint premise absent: %v", err)
				}
			}
			delegatedReadOnly(t, binary, path, cut, endpoint, contacts...)
			outingProcessJSON(t, binary, "resume", path)
			final := outingSnapshot(t, path)
			requireDelegatedRecovery(t, cut, final)
			if len(final.Runs) != 3 || len(final.Invocations) != 1 || len(final.Effects) != 2 {
				t.Fatal("fresh resume replaced helper or forecasts")
			}
			call := final.Invocations[0]
			requireOutingSelection(t, final, call.OperationID, weatherRun)
			requireOutingSelection(t, final, outingOperation(t, final, decisionPort).ID, weatherRun)
			resumedPath := f.capture(t, path, "resumed", "complete", true)
			requireDelegatedRawPrefix(t, cutPath, resumedPath)
			delegatedReadOnly(t, binary, path, final, endpoint, "40.8", "40.7")
			record := map[string]any{"cut": name, "event": event, "signal": "SIGKILL", "owner_argv": process.cmd.Args, "cut_namespace": cut.Namespace, "cut_contacts": contacts, "resumed_contacts": []string{"40.8", "40.7"}, "same_original_invocation": call.OperationID, "helper_run": call.Admission.Members[0]}
			raw, _ := json.MarshalIndent(record, "", "  ")
			if err := os.WriteFile(filepath.Join(f.dir, "process-witness.json"), append(raw, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDelegatedProcessUnknown(t *testing.T) {
	binary := publicOutingBinary(t)
	endpoint := newOutingEndpoint(t, true)
	f := newRawModelFixtureEndpoint(t, 2, defaultDeploymentOptions(delegatedWeather), []int{1, 3}, false, endpoint.url)
	owner, path := f.create(t, true)
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	process, _ := launchOutingOwner(t, binary, "run", path, nil)
	var contact outingContact
	select {
	case contact = <-endpoint.accepted:
	case <-time.After(outingProcessTimeout):
		t.Fatal("endpoint did not acknowledge original GET")
	}
	process.kill(t)
	first := outingSnapshot(t, path)
	for i := 0; i < 3; i++ {
		cut := outingSnapshot(t, path)
		if len(cut.Invocations) != 1 || len(cut.Effects) != 1 || len(cut.Runs) != 3 {
			t.Fatal("unknown cut lost original helper or started later work")
		}
		call, effect := cut.Invocations[0], cut.Effects[0]
		if call.Admission == nil || !call.Admission.Accepted || call.Outcome != nil || effect.OperationID != contact.operation || effect.Source == weatherRun || effect.DispatchAdmission == nil || effect.AdapterResult != nil || effect.Settlement != nil || call.SelectedAt != nil {
			t.Fatal("withheld original result was invented or lost")
		}
		endpoint.requireContacts(t, cut, "40.8")
		f.capture(t, path, []string{"killed", "resume1", "resume2"}[i], "unknown_forecast", true)
		delegatedReadOnly(t, binary, path, cut, endpoint, "40.8")
		requireUnadvancedOuting(t, first, cut)
		if !reflect.DeepEqual(first.Invocations, cut.Invocations) || !reflect.DeepEqual(first.Roots, cut.Roots) {
			t.Fatal("unknown resume changed invocation or ancestor debt")
		}
		if i < 2 {
			outingProcessJSON(t, binary, "resume", path)
			requireDelegatedRecovery(t, cut, outingSnapshot(t, path))
		}
	}
	raw, _ := json.MarshalIndent(map[string]any{"signal": "SIGKILL", "pid": process.cmd.Process.Pid, "endpoint_acknowledged_operation": contact.operation, "latitude": contact.latitude, "client_request_id": contact.clientID, "contacts_after_two_fresh_resumes": 1, "invocation": first.Invocations[0].OperationID}, "", "  ")
	if err := os.WriteFile(filepath.Join(f.dir, "process-witness.json"), append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
