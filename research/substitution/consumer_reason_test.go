package substitution

import (
	"fmt"
	"testing"
)

// Directed K1 construction paths only: no product or replacement census.
func consumerReasonDirectedTrace(t *testing.T, c JoinConfig, withHelper bool) (ConsumerReasonState, []ConsumerReasonEvent) {
	t.Helper()
	s := ConsumerReasonInitial(c)
	trace := []ConsumerReasonEvent{}
	step := func(op, kind string) {
		t.Helper()
		matches := []ConsumerReasonEdge{}
		for _, edge := range ConsumerReasonSuccessors(c, Reference, s) {
			if edge.Event.Fact.Operation == op && edge.Event.Fact.Kind == kind {
				matches = append(matches, edge)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("specified %s/%s edge count=%d at stage%d", op, kind, len(matches), s.ParentStage)
		}
		next := matches[0].Next
		if ConsumerReasonRank(next) <= ConsumerReasonRank(s) || ConsumerReasonRank(next) > ConsumerReasonRankCap {
			t.Fatalf("nonprogressing/out-of-bound edge %s/%s", op, kind)
		}
		if err := CheckConsumerReasonState(c, next); err != nil {
			t.Fatalf("invalid successor %s/%s: %v", op, kind, err)
		}
		trace = append(trace, matches[0].Event)
		if failures := CheckConsumerReasonPrefix(c, trace); len(failures) != 0 {
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
	for _, edge := range ConsumerReasonSuccessors(c, Reference, s) {
		if edge.Event.Fact.Kind == "outcome_consume" {
			t.Fatal("first original consumption enabled before S commitment")
		}
	}
	step("s", "invoke_commit")
	for _, kind := range []string{"invoke_authorize", "admit", "request_consume"} {
		step("e", kind)
	}
	choices := map[string]bool{}
	for _, edge := range ConsumerReasonSuccessors(c, Reference, s) {
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
	for _, edge := range ConsumerReasonSuccessors(c, Reference, s) {
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
	for _, kind := range []string{"parent_classify", "parent_decide", "parent_terminal", "root_publish"} {
		step("root", kind)
	}
	return s, trace
}

// This specification-authored fixture adds an actual classification write before
// the original decision. It is neither a runtime trace nor artifact correspondence.
func reasonFixture(mode string) (JoinConfig, []ConsumerReasonEvent) {
	c, old := consumerPolicyFixture(Request{ID: 1, Value: 1, Mode: mode}, false)
	trace := []ConsumerReasonEvent{}
	classification := map[string]string{
		"answer": ReasonAnswer, "fail": ReasonFailed, "absent": ReasonMissingReply,
	}[mode]
	for _, event := range old {
		if event.Fact.Kind == "parent_decide" {
			trace = append(trace, ConsumerReasonEvent{
				JoinEvent: JoinEvent{Fact: Observation{
					Kind: "parent_classify", Operation: "root", Role: "parent", Request: c.Request,
				}}, Classification: classification,
			})
		}
		trace = append(trace, ConsumerReasonEvent{JoinEvent: event})
	}
	return c, trace
}

func TestConsumerReasonDomain(t *testing.T) {
	configs := ConsumerReasonConfigurations()
	if len(configs) != 16 {
		t.Fatalf("declared domain has %d configurations", len(configs))
	}
	seen := map[Request]bool{}
	for _, c := range configs {
		if seen[c.Request] || !ConsumerReasonEntitled(c) {
			t.Fatalf("duplicate or excluded mode: %+v", c)
		}
		seen[c.Request] = true
		if err := CheckConsumerReasonState(c, ConsumerReasonInitial(c)); err != nil {
			t.Fatal(err)
		}
		local := ConsumerReasonLocalConfig(c)
		if !local.Local || !local.Peer || !local.Closure || local.EntryDepth != 1 || local.Request != c.Request {
			t.Fatalf("wrong original local environment: %+v", local)
		}
	}
	c := configs[0]
	c.I = 2
	if ConsumerReasonEntitled(c) {
		t.Fatal("scarcity silently included in G_outcome profile")
	}
	c.Request.Mode = "invalid"
	// Invalid requests return before exploring; these calls execute no census.
	if _, err := ExploreConsumerReason(c, Reference); err == nil {
		t.Fatal("invalid product input accepted")
	}
	if _, err := QualifyConsumerReasonLocal(c, Local); err == nil {
		t.Fatal("invalid local qualification input accepted")
	}
	if _, err := ExploreConsumerReason(configs[0], Variant("unknown")); err == nil {
		t.Fatal("invalid product variant accepted")
	}
}

func TestConsumerReasonDirectedReference(t *testing.T) {
	for _, mode := range []string{"answer", "fail", "absent", "pending"} {
		for _, helper := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/helper%t", mode, helper), func(t *testing.T) {
				c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: mode}, I: 3, R: 3, Depth: 2}
				s, trace := consumerReasonDirectedTrace(t, c, helper)
				if failures := CheckConsumerReasonComponent(c, trace); len(failures) != 0 {
					t.Fatalf("K1 component path: %+v", failures)
				}
				if failures := CheckConsumerReasonGuarantee(c, trace); len(failures) != 0 {
					t.Fatalf("G_outcome rejects specified path: %+v", failures)
				}
				if len(ConsumerReasonSuccessors(c, Reference, s)) != 0 {
					t.Fatal("designated completed/pending path has more progressing actions")
				}
				kind := "terminal"
				if mode == "pending" {
					kind = "pending"
				}
				if reasonMaximalKind(s) != kind {
					t.Fatalf("wrong maximum kind: %s", reasonMaximalKind(s))
				}
				if mode == "pending" {
					s.Component.Ops[sibling].Consumed = false
					s.ParentStage = 2
					if reasonMaximalKind(s) != "stuck" {
						t.Fatal("mere S delivery accepted as intended pending maximum")
					}
				}
			})
		}
	}
}

func TestConsumerReasonClassificationTable(t *testing.T) {
	cases := []struct {
		name string
		e, s Operation
		want string
	}{
		{name: "answer", e: Operation{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 1}, want: ReasonAnswer},
		{name: "failed", e: Operation{Status: "failed", Reason: "member_failed"}, want: ReasonFailed},
		{name: "absent", e: Operation{Status: "failed", Reason: "missing_reply"}, want: ReasonMissingReply},
		{name: "failed-present", e: Operation{Status: "failed", Reason: "member_failed", Present: true}, want: ReasonInvalid},
		{name: "success-absent", e: Operation{Status: "succeeded", Reason: "member_succeeded"}, want: ReasonInvalid},
		{name: "success-wrong-reason", e: Operation{Status: "succeeded", Reason: "missing_reply", Present: true, ID: 1}, want: ReasonInvalid},
		{name: "wrong-id", e: Operation{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 2}, want: ReasonInvalid},
		{name: "wrong-complement", e: Operation{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 1, Value: 1}, want: ReasonInvalid},
		{name: "rejected", e: Operation{Status: "rejected", Reason: "invocation_limit"}, want: ReasonInvalid},
		{name: "s-wrong-reason", e: Operation{Status: "failed", Reason: "member_failed"}, s: Operation{Status: "succeeded", Reason: "member_failed", Present: true, ID: 1}, want: ReasonInvalid},
		{name: "s-absent", e: Operation{Status: "failed", Reason: "missing_reply"}, s: Operation{Status: "succeeded", Reason: "member_succeeded", ID: 1}, want: ReasonInvalid},
		{name: "s-nonzero", e: Operation{Status: "failed", Reason: "member_failed"}, s: Operation{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 1, Value: 1}, want: ReasonInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := JoinConfig{Request: Request{ID: 1, Value: 1, Mode: "pending"}, I: 3, R: 3, Depth: 2}
			s := ConsumerReasonInitial(c)
			s.ParentStage = 4
			e, other := tc.e, tc.s
			if other.Status == "" {
				other = Operation{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 1}
			}
			e.Consumed, other.Consumed = true, true
			s.Component.Ops[entry], s.Component.Ops[sibling] = e, other
			seen := false
			for _, edge := range ConsumerReasonSuccessors(c, Reference, s) {
				if edge.Event.Fact.Operation != "root" {
					continue
				}
				seen = true
				if edge.Event.Fact.Kind != "parent_classify" || edge.Event.Classification != tc.want {
					t.Fatalf("classify actual evidence: %+v, want %s", edge.Event, tc.want)
				}
				if edge.Next.Decision != "" || ConsumerReasonRank(edge.Next) <= ConsumerReasonRank(s) {
					t.Fatal("classification did not remain distinct from progressing decision")
				}
			}
			if !seen {
				t.Fatal("both observations did not enable classification")
			}
			prefix := reasonPrefixState{}
			prefix.Ops[entry] = joinPrefixOperation{Consumed: true, Outcome: Observation{
				Status: e.Status, Reason: e.Reason, Present: e.Present, ID: e.ID, Value: e.Value,
			}}
			prefix.Ops[sibling] = joinPrefixOperation{Consumed: true, Outcome: Observation{
				Status: other.Status, Reason: other.Reason, Present: other.Present, ID: other.ID, Value: other.Value,
			}}
			event := ConsumerReasonEvent{JoinEvent: JoinEvent{Fact: Observation{
				Kind: "parent_classify", Operation: "root", Role: "parent", Request: c.Request,
			}}, Classification: tc.want}
			if _, failures := advanceReasonPrefix(c, prefix, event, 0); len(failures) != 0 {
				t.Fatalf("independent table disagrees: %+v", failures)
			}
		})
	}
}

func TestConsumerReasonOracleAdversaries(t *testing.T) {
	cases := []struct {
		name   string
		mutate func([]ConsumerReasonEvent) []ConsumerReasonEvent
	}{
		{name: "missing-classification", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			out := []ConsumerReasonEvent{}
			for _, e := range in {
				if e.Fact.Kind != "parent_classify" {
					out = append(out, e)
				}
			}
			return out
		}},
		{name: "wrong-tag", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			for i := range in {
				if in[i].Fact.Kind == "parent_classify" {
					in[i].Classification = ReasonFailed
				}
			}
			return in
		}},
		{name: "duplicate-classification", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			out := []ConsumerReasonEvent{}
			for _, e := range in {
				out = append(out, e)
				if e.Fact.Kind == "parent_classify" {
					out = append(out, e)
				}
			}
			return out
		}},
		{name: "missing-candidate-consume", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			out := []ConsumerReasonEvent{}
			for _, e := range in {
				if e.Fact.Operation != "e" || e.Fact.Kind != "outcome_consume" {
					out = append(out, e)
				}
			}
			return out
		}},
		{name: "wrong-terminal", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			for i := range in {
				if in[i].Fact.Kind == "parent_terminal" {
					in[i].Fact.Status = "succeeded"
				}
			}
			return in
		}},
		{name: "root-child-reason-conflation", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			in[len(in)-1].Fact.Reason = "missing_reply"
			return in
		}},
		{name: "tag-without-write", mutate: func(in []ConsumerReasonEvent) []ConsumerReasonEvent {
			for i := range in {
				if in[i].Fact.Kind == "parent_decide" {
					in[i].Classification = ReasonMissingReply
				}
			}
			return in
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, trace := reasonFixture("absent")
			if failures := CheckConsumerReasonGuarantee(c, trace); len(failures) != 0 {
				t.Fatalf("good fixture: %+v", failures)
			}
			if failures := CheckConsumerReasonPrefix(c, tc.mutate(trace)); len(failures) == 0 {
				t.Fatal("adversarial evidence accepted")
			}
		})
	}
	for _, kind := range []string{"parent_classify", "parent_decide", "parent_terminal", "root_publish"} {
		t.Run("early-"+kind, func(t *testing.T) {
			c, original := reasonFixture("fail")
			trace := []ConsumerReasonEvent{}
			var inserted ConsumerReasonEvent
			for _, e := range original {
				if e.Fact.Kind == kind {
					inserted = e
				}
			}
			for _, e := range original {
				trace = append(trace, e)
				if e.Fact.Operation == "s" && e.Fact.Kind == "outcome_consume" {
					break
				}
			}
			if failures := CheckConsumerReasonPrefix(c, trace); len(failures) != 0 {
				t.Fatalf("honest one-consumed prefix: %+v", failures)
			}
			trace = append(trace, inserted)
			if failures := CheckConsumerReasonPrefix(c, trace); len(failures) == 0 {
				t.Fatal("early parent action accepted")
			}
		})
	}
}

func TestConsumerReasonObservationsOverrideRequestedMode(t *testing.T) {
	c, trace := reasonFixture("absent")
	for i := range trace {
		f := &trace[i].Fact
		if f.Operation == "e" {
			if f.Kind == "terminal_freeze" {
				f.Status = "failed"
			}
			if f.Reason == "missing_reply" {
				f.Reason = "member_failed"
			}
		}
		if f.Kind == "parent_classify" {
			trace[i].Classification = ReasonFailed
		}
	}
	if failures := CheckConsumerReasonPrefix(c, trace); len(failures) != 0 {
		t.Fatalf("truthful deliberate failure: %+v", failures)
	}
	if failures := CheckConsumerReasonComponent(c, trace); len(failures) == 0 {
		t.Fatal("absent-request failure wrongly conforms to K1")
	}
	if failures := CheckConsumerReasonGuarantee(c, trace); len(failures) == 0 {
		t.Fatal("request-mode departure passed G_outcome")
	}
}

func TestConsumerReasonPendingEvidence(t *testing.T) {
	c, original := reasonFixture("pending")
	if failures := CheckConsumerReasonGuarantee(c, original); len(failures) != 0 {
		t.Fatalf("specified pending: %+v", failures)
	}
	if failures := CheckConsumerReasonGuarantee(c, original[:len(original)-1]); len(failures) == 0 {
		t.Fatal("finite pre-pending prefix proved progress")
	}
	for _, kind := range []string{"pending_receive", "terminal_freeze", "reply_commit"} {
		t.Run("after-pending-"+kind, func(t *testing.T) {
			trace := append([]ConsumerReasonEvent{}, original...)
			e := trace[len(trace)-1]
			e.Fact.Kind = kind
			trace = append(trace, e)
			if failures := CheckConsumerReasonPrefix(c, trace); len(failures) == 0 {
				t.Fatal("unmatched receive resumed without an input")
			}
		})
	}
}

func TestConsumerReasonStateGuards(t *testing.T) {
	c := ConsumerReasonConfigurations()[0]
	s := ConsumerReasonInitial(c)
	s.Classification = ReasonAnswer
	if err := CheckConsumerReasonState(c, s); err == nil {
		t.Fatal("classification before actual consumptions accepted")
	}
	s, _ = consumerReasonDirectedTrace(t, c, false)
	for _, classification := range []string{"", ReasonFailed, "invented"} {
		t.Run("terminal-tag-"+classification, func(t *testing.T) {
			bad := s
			bad.Classification = classification
			if err := CheckConsumerReasonState(c, bad); err == nil {
				t.Fatal("terminal state without actual matching classification accepted")
			}
		})
	}
}

func TestConsumerReasonInvalidSControl(t *testing.T) {
	c, trace := reasonFixture("answer")
	for i := range trace {
		f := &trace[i].Fact
		if f.Operation == "s" && (f.Kind == "reply_commit" || f.Present) {
			f.ID = 2
		}
		if f.Kind == "parent_classify" {
			trace[i].Classification = ReasonInvalid
		}
		if f.Operation == "root" && f.Status != "" {
			f.Status = "failed"
			if f.Kind == "root_publish" {
				f.Reason = "member_failed"
			}
		}
	}
	if failures := CheckConsumerReasonPrefix(c, trace); len(failures) != 0 {
		t.Fatalf("honest invalid-S defensive path: %+v", failures)
	}
	if failures := CheckConsumerReasonGuarantee(c, trace); len(failures) == 0 {
		t.Fatal("invalid_outcome silently admitted as conforming G_outcome label")
	}
	// Wrong S must still join e before classifying the failure.
	withoutCandidate := []ConsumerReasonEvent{}
	for _, event := range trace {
		if event.Fact.Operation != "e" || event.Fact.Kind != "outcome_consume" {
			withoutCandidate = append(withoutCandidate, event)
		}
	}
	if failures := CheckConsumerReasonPrefix(c, withoutCandidate); len(failures) == 0 {
		t.Fatal("invalid S bypassed original candidate join")
	}
}
