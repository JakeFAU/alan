package substitution

import (
	"fmt"
	"testing"
)

// Directed construction tests only. Product/local qualification is separately gated.
func TestCallableJoinDomain(t *testing.T) {
	configs := CallableJoinConfigurations()
	if len(configs) != 16 {
		t.Fatalf("declared request domain: %d", len(configs))
	}
	for _, c := range configs {
		if !CallableJoinEntitled(c) {
			t.Fatalf("mode excluded from entitlement: %+v", c)
		}
		local := CallableJoinLocalConfig(c)
		if local.EntryDepth != 2 || local.Depth != 3 || local.I != 3 || local.R != 3 || !local.Peer {
			t.Fatalf("unaligned local environment: %+v", local)
		}
		if err := CheckCallableJoinState(c, CallableJoinInitial(c)); err != nil {
			t.Fatal(err)
		}
	}
}

func callableDirected(
	t *testing.T,
	mode string,
	helperChoice, sFirst bool,
) (CallableJoinConfig, CallableJoinState, []CallableJoinEvent) {
	t.Helper()
	c := CallableJoinConfig{Request: Request{ID: 1, Value: 0, Mode: mode}, I: 4, R: 4, Depth: 3}
	s := CallableJoinInitial(c)
	trace := []CallableJoinEvent{}
	step := func(op, kind string) {
		t.Helper()
		matches := []CallableJoinEdge{}
		for _, edge := range CallableJoinSuccessors(c, Reference, s) {
			if edge.Event.Fact.Operation == op && edge.Event.Fact.Kind == kind {
				matches = append(matches, edge)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("directed %s/%s successors=%d", op, kind, len(matches))
		}
		edge := matches[0]
		if err := CheckCallableJoinState(c, edge.Next); err != nil {
			t.Fatalf("%s/%s state: %v", op, kind, err)
		}
		if err := CheckCallableJoinProjectionEdge(c, Reference, s, edge); err != nil {
			t.Fatalf("%s/%s projection: %v", op, kind, err)
		}
		if CallableJoinRank(edge.Next) <= CallableJoinRank(s) {
			t.Fatalf("%s/%s does not progress rank", op, kind)
		}
		trace = append(trace, edge.Event)
		if failures := CheckCallableJoinPrefix(c, trace); len(failures) != 0 {
			t.Fatalf("%s/%s prefix: %+v", op, kind, failures)
		}
		if failures := CheckCallableJoinComponent(c, trace); len(failures) != 0 {
			t.Fatalf("%s/%s component: %+v", op, kind, failures)
		}
		s = edge.Next
	}
	admit := func(op string) {
		for _, kind := range []string{"invoke_authorize", "admit", "request_consume"} {
			step(op, kind)
		}
	}
	answer := func(op string, consume bool) {
		for _, kind := range []string{
			"reply_commit", "terminal_freeze", "reply_authorize", "reply_admit", "publish", "outcome_deliver",
		} {
			step(op, kind)
		}
		if consume {
			step(op, "outcome_consume")
		}
	}
	step("j", "invoke_commit")
	admit("j")
	step("e", "invoke_commit")
	step("s", "invoke_commit")
	if sFirst {
		admit("s")
		answer("s", true)
	}
	admit("e")
	choices := map[string]bool{}
	for _, edge := range CallableJoinSuccessors(c, Reference, s) {
		choices[edge.Event.Fact.Kind] = true
	}
	if !choices["choose_local"] || !choices["choose_helper"] {
		t.Fatal("reference does not retain both independent K1 choices")
	}
	choice := "choose_local"
	if helperChoice {
		choice = "choose_helper"
	}
	step("e", choice)
	if helperChoice {
		step("h", "invoke_commit")
		admit("h")
		answer("h", true)
	}
	switch mode {
	case "answer":
		answer("e", false)
	case "fail", "absent":
		for _, kind := range []string{"terminal_freeze", "publish", "outcome_deliver"} {
			step("e", kind)
		}
	case "pending":
		step("e", "pending_receive")
	}
	if !sFirst {
		for _, edge := range CallableJoinSuccessors(c, Reference, s) {
			if edge.Event.Fact.Operation == "e" && edge.Event.Fact.Kind == "outcome_consume" {
				t.Fatal("delivered e was selectable before S consumption")
			}
		}
		admit("s")
		answer("s", true)
	}
	if mode == "pending" {
		return c, s, trace
	}
	step("e", "outcome_consume")
	for _, kind := range []string{"join_classify", "join_decide", "reply_commit", "terminal_freeze"} {
		step("j", kind)
	}
	if s.ReportAuthorized || s.Published || s.Consumed || s.OuterStage != 1 {
		t.Fatal("late-first-authorization boundary was collapsed")
	}
	for _, kind := range []string{"reply_authorize", "reply_admit", "publish", "outcome_deliver", "outcome_consume"} {
		step("j", kind)
	}
	for _, kind := range []string{"outer_decide", "outer_terminal", "root_publish"} {
		step("root", kind)
	}
	return c, s, trace
}

func TestCallableJoinDirectedReference(t *testing.T) {
	for _, mode := range []string{"answer", "fail", "absent", "pending"} {
		for _, helperChoice := range []bool{false, true} {
			for _, sFirst := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/helper%t/s-first%t", mode, helperChoice, sFirst), func(t *testing.T) {
					c, s, trace := callableDirected(t, mode, helperChoice, sFirst)
					if len(CallableJoinSuccessors(c, Reference, s)) != 0 {
						t.Fatal("directed maximum still has a progressing transition")
					}
					if failures := CheckCallableJoinContract(c, trace); len(failures) != 0 {
						t.Fatalf("K_join: %+v", failures)
					}
					if failures := CheckCallableJoinOuterGuarantee(c, trace); len(failures) != 0 {
						t.Fatalf("G_outer: %+v", failures)
					}
					kind := "terminal"
					if mode == "pending" {
						kind = "pending"
					}
					if callableMaximalKind(s) != kind {
						t.Fatalf("wrong directed maximum: %s", callableMaximalKind(s))
					}
				})
			}
		}
	}
}

func callableMutate(trace []CallableJoinEvent, op, kind string, mutate func(*CallableJoinEvent)) []CallableJoinEvent {
	out := append([]CallableJoinEvent{}, trace...)
	for i := range out {
		if out[i].Fact.Operation == op && out[i].Fact.Kind == kind {
			mutate(&out[i])
		}
	}
	return out
}

func TestCallableJoinOracleAdversaries(t *testing.T) {
	c, _, original := callableDirected(t, "absent", true, false)
	cases := []struct {
		name, op, kind string
		mutate         func(*CallableJoinEvent)
	}{
		{name: "wrong-classification", op: "j", kind: "join_classify",
			mutate: func(e *CallableJoinEvent) { e.Classification = ReasonFailed }},
		{name: "wrong-report", op: "j", kind: "reply_commit",
			mutate: func(e *CallableJoinEvent) { e.Diagnostic.Classification = ReasonFailed }},
		{name: "report-id", op: "j", kind: "reply_commit", mutate: func(e *CallableJoinEvent) { e.Diagnostic.ID = 2 }},
		{name: "unbound-reply", op: "j", kind: "reply_authorize",
			mutate: func(e *CallableJoinEvent) { e.ReplyBound = false }},
		{name: "changed-revision", op: "j", kind: "reply_authorize",
			mutate: func(e *CallableJoinEvent) { e.RevisionMatches = false }},
		{name: "changed-generation", op: "j", kind: "reply_authorize",
			mutate: func(e *CallableJoinEvent) { e.GenerationMatches = false }},
		{name: "wrong-source-snapshot", op: "j", kind: "reply_authorize",
			mutate: func(e *CallableJoinEvent) { e.SourceOpen = true }},
		{name: "wrong-caller", op: "j", kind: "reply_commit", mutate: func(e *CallableJoinEvent) { e.Fact.Parent = "peer" }},
		{name: "failed-report-erased", op: "j", kind: "publish", mutate: func(e *CallableJoinEvent) {
			e.Fact.Present, e.Diagnostic = false, JoinDiagnostic{}
		}},
		{name: "child-reason-propagated", op: "j", kind: "publish",
			mutate: func(e *CallableJoinEvent) { e.Fact.Reason = "missing_reply" }},
		{name: "invented-debit", op: "j", kind: "reply_admit", mutate: func(e *CallableJoinEvent) { e.Fact.I = 1 }},
		{name: "outer-decision", op: "root", kind: "outer_decide",
			mutate: func(e *CallableJoinEvent) { e.Decision = ReasonFailed }},
		{name: "early-root-success", op: "root", kind: "outer_terminal",
			mutate: func(e *CallableJoinEvent) { e.Fact.Status = "succeeded" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trace := callableMutate(original, tc.op, tc.kind, tc.mutate)
			if failures := CheckCallableJoinPrefix(c, trace); len(failures) == 0 {
				t.Fatal("adversarial trace accepted")
			}
		})
	}
	for _, target := range []struct{ op, kind string }{
		{op: "s", kind: "outcome_consume"}, {op: "e", kind: "outcome_consume"},
		{op: "j", kind: "join_classify"}, {op: "j", kind: "reply_commit"},
		{op: "j", kind: "reply_authorize"}, {op: "j", kind: "reply_admit"},
		{op: "j", kind: "terminal_freeze"}, {op: "j", kind: "outcome_consume"},
		{op: "root", kind: "outer_decide"},
	} {
		t.Run("missing-"+target.op+"-"+target.kind, func(t *testing.T) {
			trace := []CallableJoinEvent{}
			for _, e := range original {
				if e.Fact.Operation != target.op || e.Fact.Kind != target.kind {
					trace = append(trace, e)
				}
			}
			if failures := CheckCallableJoinPrefix(c, trace); len(failures) == 0 {
				t.Fatal("missing original commitment/selection accepted")
			}
		})
		t.Run("duplicate-"+target.op+"-"+target.kind, func(t *testing.T) {
			trace := []CallableJoinEvent{}
			for _, e := range original {
				trace = append(trace, e)
				if e.Fact.Operation == target.op && e.Fact.Kind == target.kind {
					trace = append(trace, e)
				}
			}
			if failures := CheckCallableJoinPrefix(c, trace); len(failures) == 0 {
				t.Fatal("duplicate original fact accepted")
			}
		})
	}
}

func TestCallableJoinPublicPolicyAndTruthSeparation(t *testing.T) {
	c, _, trace := callableDirected(t, "absent", false, true)
	for i := range trace {
		e := &trace[i]
		if e.Fact.Operation == "j" && e.Diagnostic.Classification != "" {
			e.Diagnostic.Classification = ReasonFailed
		}
		if e.Fact.Kind == "outer_decide" {
			e.Decision = ReasonFailed
		}
	}
	if failures := CheckCallableJoinPublic(c, trace); len(failures) != 0 {
		t.Fatalf("faithful public policy over authentic lie: %+v", failures)
	}
	if failures := CheckCallableJoinContract(c, trace); len(failures) == 0 {
		t.Fatal("public authenticity certified inner truth")
	}
	if failures := CheckCallableJoinOuterGuarantee(c, trace); len(failures) == 0 {
		t.Fatal("mode-inconsistent public decision passed G_outer")
	}
	// K_join's verdict ignores O's private decisions, not J's public boundary.
	c, _, trace = callableDirected(t, "answer", false, true)
	trace = callableMutate(trace, "root", "outer_decide", func(e *CallableJoinEvent) { e.Decision = ReasonFailed })
	if failures := CheckCallableJoinContract(c, trace); len(failures) != 0 {
		t.Fatalf("K_join improperly depends on O's private policy: %+v", failures)
	}
	if failures := CheckCallableJoinOuterGuarantee(c, trace); len(failures) == 0 {
		t.Fatal("O's faulty decision passed public guarantee")
	}
}

func TestCallableJoinPendingEvidence(t *testing.T) {
	c, s, trace := callableDirected(t, "pending", true, true)
	if failures := CheckCallableJoinContract(c, trace[:len(trace)-1]); len(failures) == 0 {
		t.Fatal("prefix before unmatched receive certified pending progress")
	}
	for _, kind := range []string{"pending_receive", "reply_commit", "terminal_freeze"} {
		t.Run("after-pending-"+kind, func(t *testing.T) {
			extra := trace[len(trace)-1]
			extra.Fact.Kind = kind
			bad := append(append([]CallableJoinEvent{}, trace...), extra)
			if failures := CheckCallableJoinPrefix(c, bad); len(failures) == 0 {
				t.Fatal("unmatched candidate resumed without release input")
			}
		})
	}
	s.Inner.Component.Ops[sibling].Consumed = false
	s.Inner.ParentStage = 2
	if callableMaximalKind(s) != "stuck" {
		t.Fatal("mere S delivery accepted as pending maximum")
	}
	// Monitor memory affects search identity even with an identical model state.
	a := callableSearchKey{State: s}
	b := a
	b.Prefix.Inner.CandidatePending = true
	if a == b {
		t.Fatal("pending evidence omitted from exploration identity")
	}
}

func TestCallableJoinNoReportPublicBoundary(t *testing.T) {
	for _, terminal := range []string{"succeeded", "failed"} {
		t.Run(terminal, func(t *testing.T) {
			c, _, full := callableDirected(t, "fail", false, true)
			trace := []CallableJoinEvent{}
			for _, e := range full {
				if e.Fact.Operation == "j" && (e.Fact.Kind == "invoke_commit" || e.Fact.Kind == "invoke_authorize" ||
					e.Fact.Kind == "admit" || e.Fact.Kind == "request_consume") {
					trace = append(trace, e)
				}
			}
			f := callableMeta(c, "terminal_freeze")
			f.Status = terminal
			trace = append(trace, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}})
			o := CallableJoinOutcome{Status: "failed", Reason: "member_failed"}
			if terminal == "succeeded" {
				o.Reason = "missing_reply"
			}
			for _, kind := range []string{"publish", "outcome_deliver", "outcome_consume"} {
				trace = append(trace, callableEnvelopeEvent(c, kind, o))
			}
			f = Observation{Kind: "outer_decide", Operation: "root", Role: "caller", Request: c.Request, Status: "failed"}
			trace = append(trace, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}, Decision: "invalid_join_outcome"})
			if failures := CheckCallableJoinPublic(c, trace); len(failures) != 0 {
				t.Fatalf("honest missing J report/public invalid policy: %+v", failures)
			}
			if failures := CheckCallableJoinContract(c, trace); len(failures) == 0 {
				t.Fatal("missing diagnostic satisfied K_join")
			}
		})
	}
}

func TestCallableJoinGuardedPublicPolicy(t *testing.T) {
	c := CallableJoinConfigurations()[0]
	good := CallableJoinOutcome{Status: "succeeded", Reason: "member_succeeded", Present: true,
		Diagnostic: JoinDiagnostic{ID: 1, Classification: ReasonAnswer, Count: 1, AnswerID: 1, AnswerValue: 1}}
	cases := []struct {
		name   string
		mutate func(*CallableJoinOutcome)
		want   string
	}{
		{name: "answer", mutate: func(*CallableJoinOutcome) {}, want: ReasonAnswer},
		{name: "absent", mutate: func(o *CallableJoinOutcome) { o.Present = false }, want: "invalid_join_outcome"},
		{name: "empty-answer", mutate: func(o *CallableJoinOutcome) { o.Diagnostic.Count = 0 }, want: "invalid_join_outcome"},
		{name: "wrong-id", mutate: func(o *CallableJoinOutcome) { o.Diagnostic.ID = 2 }, want: "invalid_join_outcome"},
		{name: "wrong-answer-id", mutate: func(o *CallableJoinOutcome) { o.Diagnostic.AnswerID = 2 },
			want: "invalid_join_outcome"},
		{name: "wrong-value", mutate: func(o *CallableJoinOutcome) { o.Diagnostic.AnswerValue = 0 },
			want: "invalid_join_outcome"},
		{name: "wrong-reason", mutate: func(o *CallableJoinOutcome) { o.Reason = "missing_reply" },
			want: "invalid_join_outcome"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := good
			tc.mutate(&o)
			model, ms := callableOuterPolicy(c.Request, o)
			oracle, os := callablePublicDecision(c.Request, o)
			if model != tc.want || oracle != tc.want || ms != os {
				t.Fatalf("public policy got %s/%s and %s/%s, want %s", model, ms, oracle, os, tc.want)
			}
		})
	}
}

func TestCallableJoinInvalidInputDoesNotExplore(t *testing.T) {
	c := CallableJoinConfigurations()[0]
	if _, err := ExploreCallableJoin(c, Variant("unknown")); err == nil {
		t.Fatal("unknown variant accepted")
	}
	if _, err := QualifyCallableJoinLocal(c, Reference); err == nil {
		t.Fatal("reference accepted as replacement qualifier")
	}
	if _, err := CheckCallableJoinLifting(c, Reference); err == nil {
		t.Fatal("reference accepted as replacement lifting qualifier")
	}
	c.Request.Mode = "unknown"
	if _, err := ExploreCallableJoin(c, Reference); err == nil {
		t.Fatal("invalid request entered search")
	}
}

func TestCallableJoinObservationsOverrideMode(t *testing.T) {
	c, _, trace := callableDirected(t, "absent", false, true)
	for i := range trace {
		e := &trace[i]
		if e.Fact.Operation == "e" {
			if e.Fact.Kind == "terminal_freeze" {
				e.Fact.Status = "failed"
			}
			if e.Fact.Reason == "missing_reply" {
				e.Fact.Reason = "member_failed"
			}
		}
		if e.Fact.Kind == "join_classify" {
			e.Classification = ReasonFailed
		}
		if e.Diagnostic.Classification != "" {
			e.Diagnostic.Classification = ReasonFailed
		}
		if e.Fact.Kind == "outer_decide" {
			e.Decision = ReasonFailed
		}
	}
	if failures := CheckCallableJoinPrefix(c, trace); len(failures) != 0 {
		t.Fatalf("actual member_failed policy: %+v", failures)
	}
	if failures := CheckCallableJoinComponent(c, trace); len(failures) == 0 {
		t.Fatal("mode-departing candidate passed K1")
	}
	if failures := CheckCallableJoinContract(c, trace); len(failures) == 0 {
		t.Fatal("truthful mode departure passed K_join")
	}
}

func TestCallableJoinInvalidSStillJoinsCandidate(t *testing.T) {
	c, _, trace := callableDirected(t, "answer", false, true)
	for i := range trace {
		e := &trace[i]
		if e.Fact.Operation == "s" && (e.Fact.Kind == "reply_commit" || e.Fact.Present) {
			e.Fact.ID = 2
		}
		if e.Fact.Kind == "join_classify" {
			e.Classification = ReasonInvalid
		}
		if e.Diagnostic.Classification != "" {
			e.Diagnostic = JoinDiagnostic{ID: 1, Classification: ReasonInvalid}
		}
		if e.Fact.Operation == "j" || e.Fact.Operation == "root" {
			if e.Fact.Status == "succeeded" {
				e.Fact.Status = "failed"
			}
			if e.Fact.Reason == "member_succeeded" {
				e.Fact.Reason = "member_failed"
			}
		}
		if e.Fact.Kind == "outer_decide" {
			e.Decision = ReasonInvalid
		}
	}
	if failures := CheckCallableJoinPrefix(c, trace); len(failures) != 0 {
		t.Fatalf("truthful invalid-S defensive execution: %+v", failures)
	}
	if failures := CheckCallableJoinContract(c, trace); len(failures) == 0 {
		t.Fatal("invalid S admitted to conforming K_join maxima")
	}
	without := []CallableJoinEvent{}
	for _, e := range trace {
		if e.Fact.Operation != "e" || e.Fact.Kind != "outcome_consume" {
			without = append(without, e)
		}
	}
	if failures := CheckCallableJoinPrefix(c, without); len(failures) == 0 {
		t.Fatal("invalid S bypassed original candidate join")
	}
}

func TestCallableJoinRefusalProjectionIsNotDebit(t *testing.T) {
	c, s, _ := callableDirected(t, "pending", false, true)
	s.Inner.Component.Ops[sibling].Admission = 2
	if _, err := CallableJoinLocalProjection(c, s); err == nil {
		t.Fatal("refused S silently projected to accepted q debit")
	}
	c = CallableJoinConfigurations()[0]
	old := CallableJoinLocalConfig(c)
	old.EntryDepth, old.Depth, old.Closure = 1, 2, false
	if err := CheckCallableJoinLocalMaximum(old, Local, State{}); err == nil {
		t.Fatal("historical depth profile passed the new maximum checker")
	}
}

func TestCallableJoinRelationSeparatesOuterPolicy(t *testing.T) {
	c, a, _ := callableDirected(t, "answer", false, true)
	b := a
	b.Inner.Component.PrivatePC[entry] = 3
	if !CallableJoinRelated(a, b, true) {
		t.Fatal("pure private candidate control was not abstracted")
	}
	b.OuterDecision = ReasonFailed
	if !CallableJoinRelated(a, b, false) || CallableJoinRelated(a, b, true) {
		t.Fatal("constructor and fixed-O relations were conflated")
	}
	b = a
	b.Inner.Component.I++
	if CallableJoinRelated(a, b, false) {
		t.Fatal("constructor relation erased an actual debit")
	}
	// A directed reference-state check validates machinery only, not A/B acceptance.
	r := CheckCallableJoinLiftStep(c, Reference, CallableJoinInitial(c))
	if r.Failures != 0 || r.ComponentChecks == 0 {
		t.Fatalf("initial reference relation machinery: %+v", r)
	}
}
