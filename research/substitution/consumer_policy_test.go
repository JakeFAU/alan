package substitution

import (
	"fmt"
	"testing"
)

// consumerPolicyFixture is a specification-authored history. It completes and
// consumes S before running the candidate; it is not an observed/reordered trace.
func consumerPolicyFixture(request Request, wrongID bool) (JoinConfig, []JoinEvent) {
	c := JoinConfig{Request: request, I: 3, R: 3, Depth: 2}
	events := []JoinEvent{}
	fact := func(op, kind string) Observation {
		input, role := request, "candidate"
		if op == "s" {
			input.Mode, role = "", "s"
		}
		return Observation{Kind: kind, Operation: op, Parent: "root", Role: role, Request: input}
	}
	for _, op := range []string{"e", "s"} {
		events = append(events, JoinEvent{Fact: fact(op, "invoke_commit")})
		f := fact(op, "invoke_authorize")
		f.Status, f.Reason = "allowed", "authorized"
		events = append(events, JoinEvent{
			Fact: f, SourceOpen: true, RevisionMatches: true, GenerationMatches: true,
		})
	}
	for _, op := range []string{"s", "e"} {
		f := fact(op, "admit")
		f.Status, f.I, f.R = "accepted", 1, 1
		events = append(events, JoinEvent{Fact: f}, JoinEvent{Fact: fact(op, "request_consume")})
		if op == "e" && request.Mode == "pending" {
			events = append(events, JoinEvent{Fact: fact(op, "pending_receive")})
			return c, events
		}
		present := op == "s" || request.Mode == "answer"
		id, value := request.ID, 0
		if op == "e" {
			value = 1 - request.Value
			if wrongID {
				id = 3 - request.ID
			}
		}
		if present {
			f = fact(op, "reply_commit")
			f.Status, f.ID, f.Value = "committed", id, value
			events = append(events, JoinEvent{Fact: f})
		}
		f = fact(op, "terminal_freeze")
		f.Status = "succeeded"
		if op == "e" && request.Mode == "fail" {
			f.Status = "failed"
		}
		events = append(events, JoinEvent{Fact: f})
		if present {
			f = fact(op, "reply_authorize")
			f.Status, f.Reason = "allowed", "authorized"
			events = append(events, JoinEvent{
				Fact: f, RevisionMatches: true, GenerationMatches: true, ReplyBound: true,
			})
			f = fact(op, "reply_admit")
			f.Status, f.Present, f.ID, f.Value = "accepted", true, id, value
			events = append(events, JoinEvent{Fact: f})
		}
		for _, kind := range []string{"publish", "outcome_deliver", "outcome_consume"} {
			f = fact(op, kind)
			f.Status, f.Reason = "succeeded", "member_succeeded"
			if present {
				f.Present, f.ID, f.Value = true, id, value
			} else {
				f.Status, f.Reason = "failed", "member_failed"
				if request.Mode == "absent" {
					f.Reason = "missing_reply"
				}
			}
			events = append(events, JoinEvent{Fact: f})
		}
	}
	status := "succeeded"
	if wrongID || request.Mode != "answer" {
		status = "failed"
	}
	for _, kind := range []string{"parent_decide", "parent_terminal", "root_publish"} {
		f := Observation{Kind: kind, Operation: "root", Role: "parent", Request: request, Status: status}
		if kind == "root_publish" {
			f.Reason = "member_" + status
		}
		events = append(events, JoinEvent{Fact: f})
	}
	return c, events
}

func TestConsumerPolicySFirstOracle(t *testing.T) {
	for id := 1; id <= 2; id++ {
		for value := 0; value <= 1; value++ {
			t.Run(fmt.Sprintf("id%d/value%d", id, value), func(t *testing.T) {
				c, events := consumerPolicyFixture(Request{ID: id, Value: value, Mode: "answer"}, false)
				if failures := CheckConsumerPolicyGuarantee(c, events); len(failures) != 0 {
					t.Fatalf("S-first successful specified history rejected: %+v", failures)
				}
				if failures := CheckJoinComponent(c, events); len(failures) != 0 {
					t.Fatalf("independent component verdict: %+v", failures)
				}
			})
		}
	}
}

func TestConsumerPolicyRejectsCandidateFirst(t *testing.T) {
	c, candidateFirst := joinOracleFixture(false, "answer")
	if failures := CheckConsumerPolicyPrefix(c, candidateFirst); len(failures) == 0 {
		t.Fatal("C_se must reject actual candidate-before-S selection")
	}
}

func TestConsumerPolicyFirstSelectionTransition(t *testing.T) {
	c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
	s := JoinInitial(c)
	// Isolated receive-boundary fixture: both original envelopes are delivered.
	// This checks the order gate, not reachability or the census premise.
	s.ParentStage = 2
	for _, i := range []int{entry, sibling} {
		s.Component.Ops[i].Delivered = true
	}
	seen := false
	for _, edge := range ConsumerPolicySuccessors(c, Reference, s) {
		if edge.Event.Fact.Kind != "outcome_consume" {
			continue
		}
		if edge.Event.Fact.Operation != "s" || edge.Next.ParentStage != 3 {
			t.Fatalf("first original receive must select S: %+v", edge.Event)
		}
		seen = true
	}
	if !seen {
		t.Fatal("delivered S did not enable first selection")
	}
}

func TestConsumerPolicyPendingPrefix(t *testing.T) {
	c, events := consumerPolicyFixture(Request{ID: 1, Value: 0, Mode: "pending"}, false)
	if failures := CheckConsumerPolicyPrefix(c, events); len(failures) != 0 {
		t.Fatalf("S consumed while original candidate pending: %+v", failures)
	}
	if failures := CheckConsumerPolicyGuarantee(c, events); len(failures) == 0 {
		t.Fatal("pending prefix cannot establish G_answer")
	}
}

func TestConsumerPolicyOracleRequiresOriginalJoins(t *testing.T) {
	c, original := consumerPolicyFixture(Request{ID: 1, Value: 0, Mode: "answer"}, false)
	for _, remove := range []string{"e", "s"} {
		t.Run("missing-consume-"+remove, func(t *testing.T) {
			events := []JoinEvent{}
			for _, event := range original {
				if event.Fact.Kind != "outcome_consume" || event.Fact.Operation != remove {
					events = append(events, event)
				}
			}
			if failures := CheckConsumerPolicyPrefix(c, events); len(failures) == 0 {
				t.Fatal("decision accepted without both actual original selections")
			}
		})
	}
	for _, kind := range []string{"parent_decide", "parent_terminal", "root_publish"} {
		t.Run("early-"+kind, func(t *testing.T) {
			events := []JoinEvent{}
			for _, event := range original {
				events = append(events, event)
				if event.Fact.Kind == "outcome_consume" && event.Fact.Operation == "s" {
					break
				}
			}
			if failures := CheckConsumerPolicyPrefix(c, events); len(failures) != 0 {
				t.Fatalf("honest S-only consumed prefix rejected: %+v", failures)
			}
			events = append(events, JoinEvent{Fact: Observation{
				Kind: kind, Operation: "root", Role: "parent", Request: c.Request, Status: "failed",
			}})
			if failures := CheckConsumerPolicyPrefix(c, events); len(failures) == 0 {
				t.Fatal("error decision/terminal accepted before required candidate consumption")
			}
		})
	}
	t.Run("missing-root-publication", func(t *testing.T) {
		if failures := CheckConsumerPolicyGuarantee(c, original[:len(original)-1]); len(failures) == 0 {
			t.Fatal("parent success alone cannot establish G_answer")
		}
	})
}

func TestConsumerPolicyHonestFailureIsNotConformance(t *testing.T) {
	c, events := consumerPolicyFixture(Request{ID: 1, Value: 0, Mode: "answer"}, true)
	if failures := CheckConsumerPolicyPrefix(c, events); len(failures) != 0 {
		t.Fatalf("truthful consumed wrong-id reply and parent failure: %+v", failures)
	}
	if failures := CheckJoinComponent(c, events); len(failures) == 0 {
		t.Fatal("wrong id must independently violate K1")
	}
	if failures := CheckConsumerPolicyGuarantee(c, events); len(failures) == 0 {
		t.Fatal("truthful failed parent cannot establish G_answer")
	}
}

func TestConsumerPolicyAbsentPayloadAndFailedS(t *testing.T) {
	for _, mode := range []string{"fail", "absent"} {
		t.Run(mode, func(t *testing.T) {
			c, events := consumerPolicyFixture(Request{ID: 1, Value: 1, Mode: mode}, false)
			if failures := CheckConsumerPolicyPrefix(c, events); len(failures) != 0 {
				t.Fatalf("truthful absent payload rejected: %+v", failures)
			}
			if failures := CheckJoinComponent(c, events); len(failures) != 0 {
				t.Fatalf("K1 non-answer discipline rejected: %+v", failures)
			}
			// A detached oracle mutation: default zero happens to equal the
			// requested complement but cannot turn absence into success.
			for i := range events {
				if events[i].Fact.Kind == "parent_decide" {
					events[i].Fact.Status = "succeeded"
				}
			}
			if failures := CheckConsumerPolicyPrefix(c, events); len(failures) == 0 {
				t.Fatal("absent default payload was accepted as an answer")
			}
		})
	}
	t.Run("failed-s-still-joins-e", func(t *testing.T) {
		c, original := consumerPolicyFixture(Request{ID: 1, Value: 0, Mode: "answer"}, false)
		events := []JoinEvent{}
		// A specification control with S locally failing without a reply.
		// This is outside the original S-answer product, not runtime evidence.
		for _, event := range original {
			f := &event.Fact
			if f.Operation == "s" {
				switch f.Kind {
				case "reply_commit", "reply_authorize", "reply_admit":
					continue
				case "terminal_freeze":
					f.Status = "failed"
				case "publish", "outcome_deliver", "outcome_consume":
					f.Status, f.Reason = "failed", "member_failed"
					f.Present, f.ID, f.Value = false, 0, 0
				}
			}
			if f.Operation == "root" {
				f.Status = "failed"
				if f.Kind == "root_publish" {
					f.Reason = "member_failed"
				}
			}
			events = append(events, event)
		}
		if failures := CheckConsumerPolicyPrefix(c, events); len(failures) != 0 {
			t.Fatalf("S failure after both required joins: %+v", failures)
		}
		for i, event := range events {
			if event.Fact.Kind == "outcome_consume" && event.Fact.Operation == "s" {
				bad := append([]JoinEvent{}, events[:i+1]...)
				bad = append(bad, events[len(events)-3])
				if failures := CheckConsumerPolicyPrefix(c, bad); len(failures) == 0 {
					t.Fatal("S failure permitted a decision before e consumption")
				}
				break
			}
		}
	})
}

// consumerPolicyDirectedTrace follows named specification actions on one K1
// branch. It checks each emitted edge, but does not enumerate a product graph.
func consumerPolicyDirectedTrace(t *testing.T, c JoinConfig, withHelper bool) (JoinState, []JoinEvent) {
	t.Helper()
	s := ConsumerPolicyInitial(c)
	trace := []JoinEvent{}
	step := func(op, kind string) {
		t.Helper()
		matches := []JoinEdge{}
		for _, edge := range ConsumerPolicySuccessors(c, Reference, s) {
			if edge.Event.Fact.Operation == op && edge.Event.Fact.Kind == kind {
				matches = append(matches, edge)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("specified %s/%s edge count=%d at stage%d", op, kind, len(matches), s.ParentStage)
		}
		next := matches[0].Next
		if JoinRank(next) <= JoinRank(s) || JoinRank(next) > JoinRankCap {
			t.Fatalf("nonprogressing/out-of-bound edge %s/%s", op, kind)
		}
		if err := CheckConsumerPolicyState(c, next); err != nil {
			t.Fatalf("invalid successor %s/%s: %v", op, kind, err)
		}
		trace = append(trace, matches[0].Event)
		if failures := CheckConsumerPolicyPrefix(c, trace); len(failures) != 0 {
			t.Fatalf("model/oracle disagreement at %s/%s: %+v", op, kind, failures)
		}
		s = next
	}
	finishAnswer := func(op string) {
		for _, kind := range []string{
			"reply_commit", "terminal_freeze", "reply_authorize", "reply_admit",
			"publish", "outcome_deliver", "outcome_consume",
		} {
			step(op, kind)
		}
	}
	step("e", "invoke_commit")
	for _, edge := range ConsumerPolicySuccessors(c, Reference, s) {
		if edge.Event.Fact.Kind == "outcome_consume" {
			t.Fatal("first original consumption enabled before S commitment")
		}
	}
	step("s", "invoke_commit")
	for _, kind := range []string{"invoke_authorize", "admit", "request_consume"} {
		step("e", kind)
	}
	choices := map[string]bool{}
	for _, edge := range ConsumerPolicySuccessors(c, Reference, s) {
		choices[edge.Event.Fact.Kind] = true
	}
	if !choices["choose_local"] || !choices["choose_helper"] {
		t.Fatal("K1 did not retain both independent choices")
	}
	choice := "choose_local"
	if withHelper {
		choice = "choose_helper"
	}
	step("e", choice)
	for _, kind := range []string{"invoke_authorize", "admit", "request_consume"} {
		step("s", kind)
	}
	finishAnswer("s")
	if !s.Component.Ops[sibling].Consumed || s.Component.Ops[entry].Published || s.ParentStage != 3 {
		t.Fatal("S-consumed / candidate-unproduced boundary was not retained")
	}
	if s.Decision != "" || s.ParentStatus != "" || s.RootPublished {
		t.Fatal("parent closed or decided with the candidate still required")
	}
	for _, edge := range ConsumerPolicySuccessors(c, Reference, s) {
		if edge.Event.Fact.Operation == "root" {
			t.Fatal("parent decision enabled before original candidate result")
		}
	}
	if withHelper {
		for _, kind := range []string{"invoke_commit", "invoke_authorize", "admit", "request_consume"} {
			step("h", kind)
		}
		finishAnswer("h")
	}
	switch c.Request.Mode {
	case "answer":
		finishAnswer("e")
	case "pending":
		step("e", "pending_receive")
		return s, trace
	default:
		for _, kind := range []string{"terminal_freeze", "publish", "outcome_deliver", "outcome_consume"} {
			step("e", kind)
		}
	}
	for _, kind := range []string{"parent_decide", "parent_terminal", "root_publish"} {
		step("root", kind)
	}
	return s, trace
}

func TestConsumerPolicyDirectedReferenceBranches(t *testing.T) {
	for _, mode := range []string{"answer", "fail", "absent", "pending"} {
		for _, withHelper := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/helper%t", mode, withHelper), func(t *testing.T) {
				c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: mode}, I: 3, R: 3, Depth: 2}
				s, trace := consumerPolicyDirectedTrace(t, c, withHelper)
				if failures := CheckJoinComponent(c, trace); len(failures) != 0 {
					t.Fatalf("specified component path: %+v", failures)
				}
				if mode == "pending" {
					if joinPolicyMaximalKind(s, consumeSFirst) != "pending" {
						t.Fatal("declared pending boundary misclassified")
					}
					if len(ConsumerPolicySuccessors(c, Reference, s)) != 0 {
						t.Fatal("pending boundary has unexpected progressing action")
					}
					// S delivered alone is the old parent's maximum, not C_se's.
					s.Component.Ops[sibling].Consumed = false
					s.ParentStage = 2
					if joinPolicyMaximalKind(s, consumeSFirst) != "stuck" {
						t.Fatal("unconsumed S accepted as a C_se pending maximum")
					}
					return
				}
				if joinPolicyMaximalKind(s, consumeSFirst) != "terminal" {
					t.Fatal("specified completed path did not reach root publication")
				}
				failures := CheckConsumerPolicyGuarantee(c, trace)
				if (len(failures) == 0) != (mode == "answer") {
					t.Fatalf("G_answer scope mismatch: %+v", failures)
				}
			})
		}
	}
}

func TestConsumerPolicyDomainAndStateGuards(t *testing.T) {
	configs := ConsumerPolicyConfigurations()
	if len(configs) != 4 {
		t.Fatalf("declared input domain: %+v", configs)
	}
	seen := map[Request]bool{}
	for _, c := range configs {
		if seen[c.Request] || !JoinEntitled(c) {
			t.Fatalf("repeated or non-entitled positive input: %+v", c)
		}
		seen[c.Request] = true
	}
	c := configs[0]
	s := ConsumerPolicyInitial(c)
	if err := CheckConsumerPolicyState(c, s); err != nil {
		t.Fatalf("initial state: %v", err)
	}
	s.ParentStage = 3
	if err := CheckConsumerPolicyState(c, s); err == nil {
		t.Fatal("stage3 without required S consumption accepted")
	}
	s = ConsumerPolicyInitial(c)
	s.Component.PrivatePC[entry] = 16
	if err := CheckConsumerPolicyState(c, s); err == nil {
		t.Fatal("private instruction state exceeds frozen finite bound")
	}
	c.Request.ID = 0
	// Invalid inputs return before exploration; these are not census calls.
	if _, err := ExploreConsumerPolicy(c, Reference); err == nil {
		t.Fatal("out-of-domain product input accepted")
	}
	if _, err := ExploreConsumerPolicy(configs[0], Variant("unknown")); err == nil {
		t.Fatal("unknown product variant accepted")
	}
}
