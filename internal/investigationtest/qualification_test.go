//go:build integration

package investigationtest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
)

// compact retains public projections and the exact causal tables without copying
// immutable compiled program bytes into every event. CLI init and build hashes
// bind the source; replay independently verifies the complete retained history.
func compact(v investigation.View) any {
	return map[string]any{"namespace": v.Snapshot.Namespace, "config": v.Config, "result": v.Result, "runs": v.Runs, "unknown": v.Unknown, "models": v.ModelRequests, "lookups": v.LookupRequests, "attempts": v.Attempts, "results": v.Results, "causal_tables": causalSnapshot(v.Snapshot).Tables}
}
func causalSnapshot(s runtime.Inspection) runtime.Inspection {
	out := s
	out.Tables = map[string][]map[string]any{}
	for _, k := range []string{"attempts", "adapter_attempts", "intents", "settlements", "observations", "admissions", "message_acknowledgments"} {
		out.Tables[k] = s.Tables[k]
	}
	return out
}
func run(t *testing.T, v investigation.View, subject string) runtime.RunState {
	t.Helper()
	for _, r := range v.Runs {
		if r.ID.Subject == subject {
			return r
		}
	}
	t.Fatal("missing run " + subject)
	return runtime.RunState{}
}
func assertBoundary(t *testing.T, p *process, v investigation.View, e boundary.Event) {
	t.Helper()
	if err := validateActiveBoundary(v, e); err != nil {
		t.Fatal(err)
	}
	a := attempt(t, v, e.Intent)
	emit(t, map[string]any{"type": "durable-boundary", "ack": e, "intent": a.Intent, "authorization": a.AuthorizationID, "request": a.Prepared.Request, "filter": p.expected, "view": compact(v)})
}
func validateActiveBoundary(v investigation.View, e boundary.Event) error {
	if err := validateDurableBoundary(v.Snapshot, e); err != nil {
		return err
	}
	found := false
	for _, r := range v.Runs {
		if r.ID.Subject == e.Subject && r.ID.Generation == e.Generation {
			found = true
			if r.Epoch != e.Epoch || r.Sequence < e.Sequence {
				return fmt.Errorf("event differs from active owner epoch/sequence")
			}
		}
	}
	if !found {
		return fmt.Errorf("event active source absent")
	}
	if e.Boundary == "adapter.result-recorded" {
		for _, row := range v.Snapshot.Tables["settlements"] {
			if row["intent_id"] == e.Intent {
				return fmt.Errorf("earlier settled result is not active pre-settlement result")
			}
		}
	}
	return nil
}
func assertUnconsumed(t *testing.T, v investigation.View, id string) {
	t.Helper()
	found := false
	for _, r := range v.Results {
		if r.Candidate.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatal("recorded result absent")
	}
	for _, row := range v.Snapshot.Tables["settlements"] {
		if row["intent_id"] == id {
			t.Fatal("selected result already settled")
		}
	}
}
func monitorPreserved(before, after investigation.View) error {
	for _, old := range before.Runs {
		var next *runtime.RunState
		for i := range after.Runs {
			if after.Runs[i].ID == old.ID {
				next = &after.Runs[i]
				break
			}
		}
		if next == nil || next.Epoch < old.Epoch || next.ProgramID != old.ProgramID {
			return fmt.Errorf("run identity replaced: %s", old.ID.Subject)
		}
		for k, want := range old.Projection.Data {
			if strings.HasPrefix(k, "pending_") && want.Text != "" && next.Projection.Data[k] != want {
				return fmt.Errorf("pending identity replaced: %s/%s", old.ID.Subject, k)
			}
			if strings.HasSuffix(k, "_spent") && next.Projection.Data[k].Number < want.Number {
				return fmt.Errorf("budget reset: %s/%s", old.ID.Subject, k)
			}
			if (strings.HasPrefix(k, "e1_") || strings.HasPrefix(k, "e2_")) && want.Kind != "" && next.Projection.Data[k] != want {
				return fmt.Errorf("earlier evidence replaced: %s/%s", old.ID.Subject, k)
			}
		}
	}
	for _, r := range after.Runs {
		if r.ID.Subject == "investigation" && r.Projection.Data["revision_count"].Number > 1 {
			return fmt.Errorf("duplicate revision")
		}
	}
	return nil
}
func cloneView(t *testing.T, v investigation.View) investigation.View {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out investigation.View
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func controls(t *testing.T, p *process, v investigation.View, e boundary.Event) {
	t.Helper()
	filter := p.expected[0]
	// The actual filter wildcards epoch and sequence; those mutants pass its
	// matcher and must fail the durable monitor. Source substitution must fail
	// both the real source constraint and the independent durable correlation.
	if err := validateAcknowledgment(e, []boundary.Event{filter}, e.PID, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if err := validateDurableBoundary(v.Snapshot, e); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"epoch", "sequence", "source"} {
		changed := e
		switch field {
		case "epoch":
			changed.Epoch++
		case "sequence":
			changed.Sequence++
		case "source":
			changed.Subject = "investigation/helper1"
		}
		matchErr := validateAcknowledgment(changed, []boundary.Event{filter}, e.PID, map[string]bool{})
		if field == "source" {
			if matchErr == nil {
				t.Fatal("actual source filter accepted changed source")
			}
		} else if matchErr != nil {
			t.Fatalf("irrelevant wildcard guard: %v", matchErr)
		}
		err := validateDurableBoundary(v.Snapshot, changed)
		if err == nil {
			t.Fatal("durable monitor accepted " + field)
		}
		emit(t, map[string]any{"type": "counterexample", "monitor": "validateDurableBoundary", "field": field, "filter": filter, "original": e, "mutant": changed, "snapshot": causalSnapshot(v.Snapshot), "rejection": err.Error(), "actual_matcher_result": fmt.Sprint(matchErr)})
	}
	if err := monitorPreserved(v, v); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"duplicate-revision", "budget-reset", "pending-replacement"} {
		changed := cloneView(t, v)
		for i := range changed.Runs {
			r := &changed.Runs[i]
			if r.ID.Subject != "investigation/reviewer" && field != "duplicate-revision" {
				continue
			}
			switch field {
			case "duplicate-revision":
				if r.ID.Subject == "investigation" {
					x := r.Projection.Data["revision_count"]
					x.Number = 2
					r.Projection.Data["revision_count"] = x
				}
			case "budget-reset":
				x := r.Projection.Data["model_spent"]
				x.Number = 0
				r.Projection.Data["model_spent"] = x
			case "pending-replacement":
				x := r.Projection.Data["pending_review2"]
				x.Text = "replacement"
				r.Projection.Data["pending_review2"] = x
			}
		}
		err := monitorPreserved(v, changed)
		if err == nil {
			t.Fatal("state monitor accepted " + field)
		}
		emit(t, map[string]any{"type": "counterexample", "monitor": "monitorPreserved", "field": field, "original_runs": v.Runs, "mutant_runs": changed.Runs, "rejection": err.Error()})
	}
}

func TestReviewAndRevisionInterruption(t *testing.T) {
	p := startProvider(t, false)
	db, corpus := initialize(t, p)
	first := paused(t, db, boundary.Event{Boundary: "adapter.result-recorded", Phase: "pre-settlement", Subject: "investigation/reviewer"})
	ack1 := ackBoundary(t, first, "adapter.result-recorded", "investigation/reviewer")
	before := inspect(t, db)
	assertBoundary(t, first, before, ack1)
	assertUnconsumed(t, before, ack1.Intent)
	reviewer := run(t, before, "investigation/reviewer")
	if reviewer.Projection.Data["pending_review1"].Text != ack1.Intent || reviewer.Projection.Data["model_spent"].Number != 1 || run(t, before, "investigation").Projection.Data["revision_count"].Number != 0 {
		t.Fatal("wrong initial review obligation or revision charged early")
	}
	first.kill(t, ack1)
	if err := os.RemoveAll(corpus); err != nil {
		t.Fatal(err)
	}
	// Recovery consumes retained review1 and proceeds normally through revision.
	// The next result-recorded event for this subject is review2; review1 cannot
	// reappear here unless a model request was wrongly repeated.
	second := paused(t, db, boundary.Event{Boundary: "adapter.result-recorded", Phase: "pre-settlement", Subject: "investigation/reviewer"})
	ack2 := ackBoundary(t, second, "adapter.result-recorded", "investigation/reviewer")
	active := inspect(t, db)
	assertBoundary(t, second, active, ack2)
	assertUnconsumed(t, active, ack2.Intent)
	assertConsumed(t, active, ack1.Intent)
	if ack2.Intent == ack1.Intent || ack2.Sequence <= ack1.Sequence || run(t, active, "investigation").Projection.Data["revision_count"].Number != 1 || run(t, active, "investigation/reviewer").Projection.Data["pending_review2"].Text != ack2.Intent {
		t.Fatal("wrong concrete revision boundary")
	}
	if err := monitorPreserved(before, active); err != nil {
		t.Fatal(err)
	}
	if active.ModelRequests != 13 || contacts(p.events(t)) != 13 {
		t.Fatal("review recovery repeated model or omitted successor")
	}
	controls(t, second, active, ack2)
	swapped := ack1
	swapped.PID = ack2.PID
	swapped.Token = ack2.Token
	if err := validateAcknowledgment(swapped, second.expected, ack2.PID, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if err := validateDurableBoundary(active.Snapshot, swapped); err != nil {
		t.Fatal("original retained attempt invalid", err)
	}
	if err := validateActiveBoundary(active, ack2); err != nil {
		t.Fatal(err)
	}
	swapErr := validateActiveBoundary(active, swapped)
	if swapErr == nil {
		t.Fatal("stale earlier boundary substituted for active revision")
	}
	emit(t, map[string]any{"type": "counterexample", "monitor": "validateActiveBoundary", "field": "earlier-review-ack-substituted-for-review2", "original": ack2, "mutant": swapped, "snapshot": causalSnapshot(active.Snapshot), "runs": active.Runs, "rejection": swapErr.Error()})
	second.kill(t, ack2)
	redeliverEarlier(t, db, active)
	invoke(t, true, "recover", "--db", db)
	after := inspect(t, db)
	assertSuccess(t, after)
	assertConsumed(t, after, ack2.Intent)
	assertContacts(t, after, p.events(t))
	if err := monitorPreserved(active, after); err != nil {
		t.Fatal(err)
	}
	if contacts(p.events(t)) != 13 || !reflect.DeepEqual(attempt(t, before, ack1.Intent), attempt(t, after, ack1.Intent)) || !reflect.DeepEqual(attempt(t, active, ack2.Intent), attempt(t, after, ack2.Intent)) {
		t.Fatal("retained model request changed or repeated")
	}
	emit(t, map[string]any{"type": "recovered", "scenario": t.Name(), "initial_review_ack": ack1, "revision_review_ack": ack2, "after": compact(after), "provider_ledger": p.events(t), "sources_deleted": true, "schedule": "two SIGKILLs at recorded pre-settlement review1 and review2"})
	assertOffline(t, p, db, after)
}

// monitorLate checks the original consumed identity and all participant views;
// it cannot be satisfied by attributing retained earlier evidence to review2.
func monitorLate(before, after investigation.View) error {
	if len(before.Runs) != len(after.Runs) {
		return fmt.Errorf("late earlier message changed participant set")
	}
	for i, old := range before.Runs {
		next := after.Runs[i]
		if old.ID != next.ID || old.Sequence != next.Sequence || !reflect.DeepEqual(old.Projection, next.Projection) {
			return fmt.Errorf("late earlier message changed active projection")
		}
	}
	if !reflect.DeepEqual(before.Snapshot.Tables["observations"], after.Snapshot.Tables["observations"]) {
		return fmt.Errorf("late earlier message replaced or reconsumed observation")
	}
	return monitorPreserved(before, after)
}
func redeliverEarlier(t *testing.T, db string, before investigation.View) {
	t.Helper()
	s, err := executor.Open(t.Context(), db, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	count := 0
	for _, row := range before.Snapshot.Tables["intents"] {
		var intent executor.Intent
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &intent); err != nil {
			t.Fatal(err)
		}
		if intent.Kind != "message" || intent.Target != "investigation" || (intent.Run.Subject != "investigation/reviewer" && intent.Run.Subject != "investigation/helper1") {
			continue
		}
		result, err := s.DeliverMessage(t.Context(), intent.ID, "phase1f-redelivery:"+intent.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Accepted {
			t.Fatalf("retained message was not admitted: %+v", result)
		}
		originalObservation := false
		for _, row := range before.Snapshot.Tables["observations"] {
			if row["id"] == result.ObservationID && row["consumed_sequence"] != nil {
				originalObservation = true
			}
		}
		if !originalObservation {
			t.Fatal("redelivery did not resolve to original consumed observation")
		}
		count++
		emit(t, map[string]any{"type": "retained-message-redelivery", "intent": intent, "admission": result, "active_review": run(t, before, "investigation/reviewer").Projection.Data["pending_review2"], "scope": "existing executor settlement; no fabricated late external candidate"})
	}
	if count != 2 {
		t.Fatalf("expected reviewer1 and helper1 retained replies, got %d", count)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	after := inspect(t, db)
	if err := monitorLate(before, after); err != nil {
		t.Fatal(err)
	}
	mutant := cloneView(t, after)
	for i := range mutant.Runs {
		if mutant.Runs[i].ID.Subject == "investigation/reviewer" {
			x := mutant.Runs[i].Projection.Data["pending_review2"]
			x.Text = mutant.Runs[i].Projection.Data["pending_review1"].Text
			mutant.Runs[i].Projection.Data["pending_review2"] = x
		}
	}
	err = monitorLate(before, mutant)
	if err == nil {
		t.Fatal("late attribution mutant accepted")
	}
	emit(t, map[string]any{"type": "counterexample", "monitor": "monitorLate", "field": "late-review1-attributed-to-review2", "original_runs": before.Runs, "mutant_runs": mutant.Runs, "rejection": err.Error(), "after": compact(after)})
}

func TestPartialSiblingUnknownRecovery(t *testing.T) {
	p := startProvider(t, true)
	db, corpus := initialize(t, p)
	child := paused(t, db, boundary.Event{Boundary: "execution.staged", Phase: "terminal", Subject: "investigation/helper2"}, boundary.Event{Boundary: "execution.committed", Phase: "terminal", Subject: "investigation/helper2"})
	staged := ackBoundary(t, child, "execution.staged", "investigation/helper2")
	var durableAccepted providerEvent
	for {
		var event providerEvent
		p.p.next(t, &event)
		if event.Kind == "accepted" && strings.Contains(event.Wire, `\"question\":\"alpha\"`) {
			durableAccepted = event
		}
		if durableAccepted.Ordinal != 0 && event.Ordinal == durableAccepted.Ordinal && event.Kind == "applied" {
			break
		}
	}
	child.send(t, "c")
	ack := ackBoundary(t, child, "execution.committed", "investigation/helper2")
	held := inspect(t, db)
	sibling := run(t, held, "investigation/helper2")
	if sibling.Projection.Status != "succeeded" || sibling.Epoch != ack.Epoch || sibling.Sequence != ack.Sequence || sibling.ID.Generation != ack.Generation {
		t.Fatal("sibling commit ack differs from durable completed projection")
	}
	if err := child.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal(err)
	}
	if err := p.p.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal(err)
	}
	var original runtime.AdapterAttempt
	var accepted providerEvent
	applied := false
	for _, row := range held.Snapshot.Tables["adapter_attempts"] {
		var a runtime.AdapterAttempt
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &a); err != nil {
			t.Fatal(err)
		}
		if a.Intent.Run.Subject == "investigation/helper1" {
			original = a
		}
	}
	for _, e := range p.events(t) {
		if e.Kind == "accepted" && strings.Contains(e.Wire, `\"question\":\"alpha\"`) {
			accepted = e
		}
		if accepted.Ordinal != 0 && e.Ordinal == accepted.Ordinal && e.Kind == "applied" {
			applied = true
		}
		if accepted.Ordinal != 0 && e.Ordinal == accepted.Ordinal && e.Kind == "responding" {
			t.Fatal("held provider responded before kill")
		}
	}
	if !applied || original.AuthorizationID == "" {
		t.Fatal("missing durable provider application")
	}
	assertContact(t, original, accepted)
	if !reflect.DeepEqual(accepted, durableAccepted) {
		t.Fatal("provider ack not equal to fsynced ledger")
	}
	inFlight := false
	for _, id := range ack.InFlight {
		if id == original.AuthorizationID {
			inFlight = true
		}
	}
	if !inFlight {
		t.Fatal("held authorization absent at sibling commit")
	}
	assertConsumed(t, held, sibling.Projection.Data["pending_synthesis"].Text)
	if len(held.Unknown) != 1 || held.Unknown[0] != original.Intent.ID {
		t.Fatalf("not exact unknown: %v", held.Unknown)
	}
	emit(t, map[string]any{"type": "partial-sibling-proof", "staged": staged, "causal_order": []string{"helper2 terminal staged", "provider alpha applied and fsynced before response", "release helper2 commit", "helper2 terminal committed while alpha held"}, "ack": ack, "held_attempt": original, "accepted": accepted, "cli_alive": true, "provider_alive": true, "view": compact(held), "provider_ledger": p.events(t)})
	child.kill(t, ack)
	if err := os.RemoveAll(corpus); err != nil {
		t.Fatal(err)
	}
	old := p.events(t)
	invoke(t, true, "recover", "--db", db)
	after := inspect(t, db)
	if after.Result != nil || !reflect.DeepEqual(held.Unknown, after.Unknown) || !reflect.DeepEqual(original, attempt(t, after, original.Intent.ID)) || !reflect.DeepEqual(old, p.events(t)) || run(t, after, "investigation").Projection.Data["revision_count"].Number != 0 || !reflect.DeepEqual(sibling.Projection, run(t, after, "investigation/helper2").Projection) {
		t.Fatal("unknown reset, repeated, or completed sibling lost")
	}
	if err := monitorPreserved(held, after); err != nil {
		t.Fatal(err)
	}
	emit(t, map[string]any{"type": "unknown-recovery", "disposition": "blocked completion; original helper1 outcome remains Unknown", "after": compact(after), "ledger": p.events(t)})
	assertOffline(t, p, db, after)
}

func monitorReplay(before, after investigation.View, old, next []providerEvent) error {
	if !reflect.DeepEqual(old, next) {
		return fmt.Errorf("replay dispatched provider: %d -> %d", contacts(old), contacts(next))
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("replay mutated store")
	}
	return nil
}
func TestCompleteAndFaultyReplayDispatch(t *testing.T) {
	p := startProvider(t, false)
	db, corpus := initialize(t, p)
	if err := os.RemoveAll(corpus); err != nil {
		t.Fatal(err)
	}
	invoke(t, true, "run", "--db", db)
	before := inspect(t, db)
	assertSuccess(t, before)
	assertContacts(t, before, p.events(t))
	old := p.events(t)
	invoke(t, false, "replay", "--db", db)
	if err := monitorReplay(before, inspect(t, db), old, p.events(t)); err != nil {
		t.Fatal(err)
	}
	var original runtime.AdapterAttempt
	for _, r := range before.Results {
		a := attempt(t, before, r.Candidate.ID)
		if a.Prepared.Request.Capability == investigation.ModelCapability {
			original = a
			break
		}
	}
	adapter, err := openai.New(openai.Config{APIKey: "synthetic-process-fixture", Model: before.Config.Model, Endpoint: p.endpoint, MaxOutputTokens: before.Config.MaxOutputTokens, MaxResponseBytes: before.Config.MaxResponseBytes})
	if err != nil {
		t.Fatal(err)
	}
	// Real extra HTTP dispatch deliberately bypasses owner/source guards. This is
	// a faulty replay wrapper, not a mutation of the actual read-only replay API.
	candidate := adapter.Execute(t.Context(), original.Prepared)
	if candidate.Disposition != "validated" {
		t.Fatal("faulty dispatch did not succeed")
	}
	next := p.events(t)
	err = monitorReplay(before, inspect(t, db), old, next)
	if err == nil || contacts(next) != contacts(old)+1 {
		t.Fatal("actual extra dispatch escaped replay monitor")
	}
	emit(t, map[string]any{"type": "counterexample", "monitor": "monitorReplay", "field": "actual-extra-dispatch", "original_attempt": original, "candidate": candidate, "ledger_before": old, "ledger_after": next, "rejection": err.Error()})
	assertOffline(t, p, db, before)
}
