package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

const publicationCut = "invocation.result-recorded"

func TestInvestigatorCompletionOrders(t *testing.T) {
	for _, first := range []string{"primary", "counter"} {
		t.Run(first+"_first", func(t *testing.T) {
			p := responsesServer(t, "opposes", "primary", "counter")
			db := initializeCase(t, p, 4)
			c := startInvestigator(t, []string{"run", db}, repeatedCut(publicationCut, "pre-consumption", 1)...)
			p.waitBoth(t)
			p.allow(first)
			e := c.next(t)
			live := inspectCase(t, db)
			live.requirePublication(t, e, first)
			if len(live.outcomes) != 1 {
				t.Fatal("second outcome published before its provider response")
			}
			other := "counter"
			if first == "counter" {
				other = "primary"
			}
			c.proceed(t)
			p.allow(other)
			c.wait(t)
			final := inspectCase(t, db)
			checkCompletedCase(t, final, p, "disagreement")
			if final.outcomes[final.invocation(first)].DecisionOrder >= final.outcomes[final.invocation(other)].DecisionOrder {
				t.Fatal("incorrect original worker publication order")
			}
			for _, role := range []string{"primary", "counter"} {
				if !final.parent.Data[role+"_accepted"].Flag || final.parent.Data[role+"_snippet"].Text != caseFacts[role].quote {
					t.Fatal("lost or exchanged conflicting evidence")
				}
			}
			replayCase(t, db, p)
		})
	}
}

func TestInvestigatorCrashHistories(t *testing.T) {
	t.Run("before_provider_dispatch", func(t *testing.T) {
		p := responsesServer(t, "supports")
		db := initializeCase(t, p, 4)
		// Select the first real model request, whose namespace and worker name
		// are minted by this store. The single runner has not reserved it yet.
		c := startInvestigator(t, []string{"run", db}, repeatedCut("execution.committed", "post-commit", 64)...)
		var id string
		for id == "" {
			e := c.next(t)
			if e.Boundary != "execution.committed" || e.Phase != "post-commit" {
				t.Fatal("wrong commit cut")
			}
			if e.Intent != "" && e.Subject != "investigator" {
				l := inspectCase(t, db)
				for _, r := range l.Tables["intents"] {
					if r["id"] != e.Intent {
						continue
					}
					i := decodeRow[struct{ Kind, Target string }](t, r, "descriptor")
					if i.Kind == "adapter" && i.Target == "model" {
						if e.Sequence != r["sequence"] || e.Subject != r["subject"] || e.Epoch < 1 || l.attempts[e.Intent].Intent.ID != "" || len(p.contacts()) != 0 {
							t.Fatal("model dispatch preceded durable request cut")
						}
						id = e.Intent
					}
				}
			}
			if id == "" {
				c.proceed(t)
			}
		}
		c.kill(t, "committed, unreserved model operation "+id)
		before := replayStoppedCase(t, db, p)
		after := resumeCase(t, db, c)
		requirePreserved(t, before, after)
		if !after.attempts[id].Allowed || after.results[id].Disposition != "validated" {
			t.Fatal("fresh owner did not execute the original unreserved operation")
		}
		checkCompletedCase(t, after, p, "completed")
		replayCase(t, db, p)
	})
	// A durable reservation already spends its allowance. Even a test observer
	// that knows the POST did not happen cannot confer new authority on reopen.
	t.Run("after_reservation_before_provider_call", func(t *testing.T) {
		p := responsesServer(t, "supports")
		db := initializeCase(t, p, 4)
		c := startInvestigator(t, []string{"run", db}, repeatedCut("dispatch.consider", "pre-admission", 4)...)
		var operation, role string
		for operation == "" {
			e := c.next(t)
			l := inspectCase(t, db)
			a := l.attempts[e.Intent]
			if e.Boundary != "dispatch.consider" || e.Phase != "pre-admission" || !a.Allowed || e.Subject != a.Intent.Run.Subject || e.Sequence != a.Intent.Sequence || e.Attempt != a.AuthorizationID {
				t.Fatalf("uncorrelated reservation cut %+v", e)
			}
			if a.Intent.Target == "model" {
				operation = e.Intent
				role = a.Prepared.Request.Payload["role"].Text
			} else {
				c.proceed(t)
			}
		}
		if len(p.contacts()) != 0 {
			t.Fatal("unexpected provider contact before the original gate")
		}
		c.kill(t, "durable model reservation before admission "+operation)
		before := replayStoppedCase(t, db, p)
		after := resumeCase(t, db, c)
		requirePreserved(t, before, after)
		if !slices.Contains(after.Roots[0].UnknownExternal, operation) || !slices.Contains(after.Roots[0].PendingInvocations, after.invocation(role)) || after.results[operation].Candidate.ID != "" || after.Roots[0].Outcome != nil {
			t.Fatal("fresh owner fabricated a result or closed the unobserved provider operation")
		}
		for _, contact := range p.contacts() {
			if contact.role == role {
				t.Fatal("reopening repeated the original reservation")
			}
		}
		after.compareContacts(t, p)
		replayCase(t, db, p)
	})
	t.Run("after_provider_acceptance", func(t *testing.T) {
		p := responsesServer(t, "supports", "counter")
		db := initializeCase(t, p, 4)
		c := startInvestigator(t, []string{"run", db}, repeatedCut(publicationCut, "pre-consumption", 1)...)
		p.waitBoth(t)
		e := c.next(t)
		live := inspectCase(t, db)
		live.requirePublication(t, e, "primary")
		a := live.model("counter")
		if !a.Allowed || live.results[a.Intent.ID].Candidate.ID != "" {
			t.Fatal("missing unresolved accepted counter call")
		}
		live.compareContacts(t, p)
		c.kill(t, "held accepted counter request "+a.Intent.ID+"; primary outcome "+e.Attempt)
		p.allow("counter")
		before := replayStoppedCase(t, db, p)
		after := resumeCase(t, db, c)
		requirePreserved(t, before, after)
		assertPending(t, after, p, a.Intent.ID)
		var answer struct {
			Workers []struct {
				Role  string
				Model struct {
					ClientID   string `json:"client_request_id"`
					ReturnedID string `json:"provider_request_id"`
				}
			}
		}
		if err := json.Unmarshal(cliOutput(t, "answer", db), &answer); err != nil {
			t.Fatal(err)
		}
		var observed, reported, returned string
		for _, contact := range p.contacts() {
			if contact.role == "counter" {
				observed = contact.clientID
			}
		}
		for _, worker := range answer.Workers {
			if worker.Role == "counter" {
				reported = worker.Model.ClientID
				returned = worker.Model.ReturnedID
			}
		}
		if observed == "" || reported != observed || returned != "" {
			t.Fatalf("lost response: independent client ID %q, reported %q, returned provider ID %q", observed, reported, returned)
		}
		afterAgain := resumeCase(t, db, c)
		requirePreserved(t, after, afterAgain)
		assertPending(t, afterAgain, p, a.Intent.ID)
		replayCase(t, db, p)
	})
	t.Run("after_retained_provider_result", func(t *testing.T) {
		p := responsesServer(t, "supports")
		db := initializeCase(t, p, 4)
		c := startInvestigator(t, []string{"run", db}, repeatedCut("adapter.result-recorded", "pre-settlement", 4)...)
		count := 0
		var target string
		for target == "" {
			e := c.next(t)
			l := inspectCase(t, db)
			a := l.requireResult(t, e)
			if e.Boundary != "adapter.result-recorded" || e.Phase != "pre-settlement" {
				t.Fatal("wrong result cut")
			}
			if a.Intent.Target == "model" {
				count++
				if count == 2 {
					target = e.Intent
				}
			}
			if target == "" {
				c.proceed(t)
			}
		}
		c.kill(t, "second model result committed before settlement "+target)
		before := replayStoppedCase(t, db, p)
		for _, r := range before.Tables["settlements"] {
			if r["intent_id"] == target {
				t.Fatal("target was already settled at crash")
			}
		}
		after := resumeCase(t, db, c)
		requirePreserved(t, before, after)
		checkCompletedCase(t, after, p, "completed")
		replayCase(t, db, p)
	})
	t.Run("after_one_worker_completes", func(t *testing.T) {
		p := responsesServer(t, "supports", "counter")
		db := initializeCase(t, p, 4)
		filters := append(repeatedCut(publicationCut, "pre-consumption", 1), repeatedCut("adapter.settled", "pre-consumption", 4)...)
		c := startInvestigator(t, []string{"run", db}, filters...)
		var original, other string
		for other == "" {
			e := c.next(t)
			l := inspectCase(t, db)
			switch e.Boundary {
			case publicationCut:
				l.requirePublication(t, e, "primary")
				original = e.Attempt
				// The counter result cannot arrive until the primary has completed.
				p.allow("counter")
			case "adapter.settled":
				a := l.requireResult(t, e)
				if a.Intent.Target == "model" && a.Prepared.Request.Payload["role"].Text == "counter" {
					if original == "" || len(l.outcomes) != 1 || l.outcomes[l.invocation("primary")].ID != original {
						t.Fatal("counterwork not available after original publication")
					}
					other = e.Intent
				}
			default:
				t.Fatal("unexpected boundary", e)
			}
			if other == "" {
				c.proceed(t)
			}
		}
		c.kill(t, "only primary published; counter's original result eligible for local processing "+other)
		before := replayStoppedCase(t, db, p)
		after := resumeCase(t, db, c)
		requirePreserved(t, before, after)
		checkCompletedCase(t, after, p, "completed")
		if after.outcomes[after.invocation("primary")].ID != original {
			t.Fatal("first publication replaced")
		}
		replayCase(t, db, p)
	})
	t.Run("before_parent_selection", func(t *testing.T) {
		p := responsesServer(t, "opposes", "primary", "counter")
		db := initializeCase(t, p, 4)
		c := startInvestigator(t, []string{"run", db}, repeatedCut(publicationCut, "pre-consumption", 2)...)
		p.waitBoth(t)
		p.allow("counter")
		e := c.next(t)
		inspectCase(t, db).requirePublication(t, e, "counter")
		c.proceed(t)
		p.allow("primary")
		e = c.next(t)
		l := inspectCase(t, db)
		l.requirePublication(t, e, "primary")
		l.requireSelection(t, "primary", 0)
		l.requireSelection(t, "counter", 0)
		c.kill(t, "both original outcomes published before the parent's first selection")
		before := replayStoppedCase(t, db, p)
		after := resumeCase(t, db, c)
		requirePreserved(t, before, after)
		checkCompletedCase(t, after, p, "disagreement")
		replayCase(t, db, p)
	})
}

func resumeCase(t *testing.T, db string, killed *investigatorChild) caseLedger {
	t.Helper()
	fresh := startInvestigator(t, []string{"resume", db})
	if fresh.pid == killed.pid {
		t.Fatal("same process claimed as fresh owner")
	}
	var r runtime.RunReport
	if err := json.Unmarshal(fresh.wait(t), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Quiescent {
		t.Fatal("fresh owner did not reach quiescence")
	}
	t.Logf("fresh owner pid=%d quiescent open=%d unknown_adapters=%v", fresh.pid, len(r.OpenRuns), r.UnknownAdapters)
	return inspectCase(t, db)
}

func assertPending(t *testing.T, l caseLedger, p *responsesFixture, id string) {
	t.Helper()
	if !l.attempts[id].Allowed || l.results[id].Candidate.ID != "" || l.parent.Status != "open" || l.parent.Data["completion"].Text != "" || l.Roots[0].Outcome != nil || !slices.Contains(l.Roots[0].PendingInvocations, l.invocation("counter")) || !slices.Contains(l.Roots[0].UnknownExternal, id) {
		t.Fatalf("missing original pending obligation: %+v", l.Roots[0])
	}
	if len(p.contacts()) != 2 || l.Roots[0].Charges.Effects != 4 {
		t.Fatal("uncertain call retried or its allowance refunded")
	}
	l.compareContacts(t, p)
}

func TestInvestigatorProviderHistories(t *testing.T) {
	for _, mode := range []string{"refusal", "incomplete", "provider_failed", "malformed", "schema_invalid", "http_rejected", "fabricated_quote", "no_usage", "disconnect"} {
		t.Run(mode, func(t *testing.T) {
			p := responsesServer(t, mode)
			db := initializeCase(t, p, 4)
			cliOutput(t, "run", db)
			l := inspectCase(t, db)
			if len(p.contacts()) != 2 || len(l.attempts) != 4 {
				t.Fatal("incomplete independent contact or attempt ledger")
			}
			goal := "partial"
			if mode == "no_usage" {
				goal = "completed"
			}
			if mode == "disconnect" {
				assertPending(t, l, p, l.model("counter").Intent.ID)
			} else {
				checkCompletedCase(t, l, p, goal)
				counter := l.outcomes[l.invocation("counter")]
				accepted := l.parent.Data["counter_accepted"].Flag
				if accepted != (mode == "no_usage") || (mode == "fabricated_quote" && (counter.Status != "succeeded" || l.parent.Data["counter_assessment"].Text != "citation_mismatch")) || (mode != "fabricated_quote" && mode != "no_usage" && counter.Status != "failed") {
					t.Fatalf("worker or parent erased provider/application distinction: %s %s", counter.Status, l.parent.Data["counter_assessment"].Text)
				}
			}
			replayCase(t, db, p)
		})
	}
}

func TestInvestigatorDurableEffectCeiling(t *testing.T) {
	p := responsesServer(t, "supports")
	db := initializeCase(t, p, 3)
	cliOutput(t, "run", db)
	l := inspectCase(t, db)
	checkCompletedCase(t, l, p, "partial")
	if len(l.attempts) != 4 || len(p.contacts()) != 1 || l.Roots[0].Charges.Effects != 3 {
		t.Fatalf("expected four decisions, exactly three debits, one provider contact; %+v", l.Roots[0])
	}
	denied := 0
	for _, a := range l.attempts {
		if !a.Allowed {
			r := l.results[a.Intent.ID]
			denied++
			if a.Intent.Target != "model" || a.Reason != "effect_limit" || r.Disposition != "source_denied" || r.Candidate.Usage != nil || r.Candidate.HTTPStatus != 0 {
				t.Fatal("budget denial resembled a provider result")
			}
		}
	}
	if denied != 1 {
		t.Fatal("missing uncharged denial")
	}
	before := p.contacts()
	cliOutput(t, "resume", db)
	after := inspectCase(t, db)
	requirePreserved(t, l, after)
	checkCompletedCase(t, after, p, "partial")
	if !reflect.DeepEqual(before, p.contacts()) {
		t.Fatal("reopening restored provider allowance")
	}
	replayCase(t, db, p)
}
