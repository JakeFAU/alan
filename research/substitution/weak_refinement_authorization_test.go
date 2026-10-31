package substitution

import (
	"testing"
)

// Authority numbers are declared finite clause premises. They are not a raw
// execution extraction: the historical finite model has no executor sequence.
func weakAuthFixture() (weakAuthContext, weakAuthorization) {
	intent := weakAuthIntent{ID: "r", Subject: "j", Kind: "message",
		Target: "caller", Obligation: "reply", Payload: "original diagnostic",
		SourceGeneration: 1, Sequence: 8, Generation: 1, Revision: 1}
	c := weakAuthContext{Intent: intent, Committed: &intent,
		Source:            weakAuthority{Subject: "j", Generation: 1, Epoch: 1, Sequence: 8, Revision: 1},
		SubjectGeneration: 1, Status: "open", SchemaValid: true,
		Binding: weakReplyBinding{Protocol: "alan/composition/v1", Member: "join", EntryMember: "join",
			CallerSubject: "caller", CallerGeneration: 1, Obligation: "reply"}}
	a := weakAuthorization{ID: `["authorization","r"]`, IntentID: "r", Allowed: true, Reason: "authorized",
		Source: c.Source}
	return c, a
}
func TestWeakRefinementF5InvalidAuthorization(t *testing.T) {
	c, s, prefix := weakOriginalAnchor(t, true)
	for _, kind := range []string{"choose_local", "reply_commit", "terminal_freeze"} {
		e := weakOriginalEdge(t, c, s, "e", kind)
		prefix = append(prefix, e)
		s = e.Next
	}
	if issues := checkWeakOriginalPrefix(c, prefix); len(issues) != 0 {
		t.Fatal(issues)
	}
	correctReply := s.Ops[entry].ReplyID == 1 && s.Ops[entry].ReplyValue == 1
	if s.Ops[entry].Freeze != success || s.Ops[entry].Reply != 1 || !correctReply {
		t.Fatal("e answer/committed reply anchor not reached")
	}
	context, claim := weakAuthFixture()
	context.Intent.Subject = "e"
	intent := context.Intent
	context.Committed = &intent
	context.Source.Subject = "e"
	context.Status = "succeeded"
	context.Source.Sequence = 9
	context.Closure = "actual queried closure"
	claim.Source = context.Source
	claim.ClosureOperation = context.Closure
	claim.Allowed = false
	claim.Reason = "source_closed"
	if issues := checkWeakAuthorization(context, claim, nil); !hasWeakIssue(issues, "first_authorization") {
		t.Fatalf("fabricated late denial: %+v", issues)
	}
	claim.Allowed = true
	claim.Reason = "authorized"
	if issues := checkWeakAuthorization(context, claim, nil); len(issues) != 0 {
		t.Fatal(issues)
	}
	saved := weakSavedAuthorization{Intent: context.Intent, Value: claim}
	context.Source.Sequence++ // later lookup must retain the first snapshot
	if issues := checkWeakAuthorization(context, claim, &saved); len(issues) != 0 {
		t.Fatal(issues)
	}
	// A disconnected denied node is harmless until a reachable disposition is claimed.
	p, cert := weakF1()
	p.X.Nodes["unreachable-denied-sidecar"] = weakNode{Projection: "denied"}
	requireWeak(t, p, cert)
}
func weakJoinDecisionAnchor(t *testing.T, mode string) (CallableJoinConfig, CallableJoinState) {
	t.Helper()
	c := CallableJoinConfig{Request: Request{ID: 1, Value: 0, Mode: mode}, I: 4, R: 4, Depth: 3}
	s := CallableJoinInitial(c)
	step := func(op, kind string) {
		t.Helper()
		matches := []CallableJoinEdge{}
		for _, e := range CallableJoinSuccessors(c, Reference, s) {
			if e.Event.Fact.Operation == op && e.Event.Fact.Kind == kind {
				matches = append(matches, e)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("original J anchor %s/%s successors=%d", op, kind, len(matches))
		}
		s = matches[0].Next
	}
	step("j", "invoke_commit")
	step("j", "invoke_authorize")
	step("j", "admit")
	step("j", "request_consume")
	step("e", "invoke_commit")
	step("s", "invoke_commit")
	for _, kind := range []string{
		"invoke_authorize", "admit", "request_consume", "reply_commit", "terminal_freeze",
		"reply_authorize", "reply_admit", "publish", "outcome_deliver", "outcome_consume",
	} {
		step("s", kind)
	}
	for _, kind := range []string{"invoke_authorize", "admit", "request_consume", "choose_local"} {
		step("e", kind)
	}
	if mode == "answer" {
		step("e", "reply_commit")
	}
	for _, kind := range []string{"terminal_freeze"} {
		step("e", kind)
	}
	if mode == "answer" {
		step("e", "reply_authorize")
		step("e", "reply_admit")
	}
	for _, kind := range []string{"publish", "outcome_deliver", "outcome_consume"} {
		step("e", kind)
	}
	step("j", "join_classify")
	step("j", "join_decide")
	replyUnresolved := !s.ReportCommitted && !s.ReportAuthorized && !s.ReportAdmitted
	if s.Inner.ParentStage != 5 || s.Inner.ParentStatus != "" || !replyUnresolved {
		t.Fatal("J post-decision boundary not original open state")
	}
	return c, s
}
func TestWeakRefinementSixAuthorizationAdmissionSchedules(t *testing.T) {
	schedules := []struct {
		name, mode, status string
		steps              []string
	}{
		{name: "c-a-d-t-succeeded", mode: "answer", status: "succeeded", steps: []string{"c", "a", "d", "t"}},
		{name: "c-a-t-succeeded-d", mode: "answer", status: "succeeded", steps: []string{"c", "a", "t", "d"}},
		{name: "c-t-succeeded-a-d", mode: "answer", status: "succeeded", steps: []string{"c", "t", "a", "d"}},
		{name: "c-a-d-t-failed", mode: "fail", status: "failed", steps: []string{"c", "a", "d", "t"}},
		{name: "c-a-t-failed-d", mode: "fail", status: "failed", steps: []string{"c", "a", "t", "d"}},
		{name: "c-t-failed-a-d", mode: "fail", status: "failed", steps: []string{"c", "t", "a", "d"}},
	}
	for _, tc := range schedules {
		t.Run(tc.name, func(t *testing.T) {
			c, s := weakJoinDecisionAnchor(t, tc.mode)
			context, claim := weakAuthFixture()
			var saved *weakSavedAuthorization
			history := []weakAuthorizationLookup{}
			committed, admitted, ended := false, false, false
			context.Committed = nil
			for _, action := range tc.steps {
				kinds := map[string]string{"c": "reply_commit", "a": "reply_authorize", "d": "reply_admit", "t": "terminal_freeze"}
				kind := kinds[action]
				matches := []CallableJoinEdge{}
				for _, edge := range CallableJoinSuccessors(c, Reference, s) {
					if edge.Event.Fact.Operation == "j" && edge.Event.Fact.Kind == kind {
						matches = append(matches, edge)
					}
				}
				if len(matches) != 1 {
					t.Fatalf("schedule has no original J %s: %d", kind, len(matches))
				}
				edge := matches[0]
				s = edge.Next
				switch action {
				case "c":
					committed = true
					intent := context.Intent
					context.Committed = &intent
				case "t":
					ended = true
					context.Status = tc.status
					context.Source.Sequence = 9
					if edge.Event.Fact.Status != tc.status || !s.ReportCommitted || s.Inner.ParentStatus != tc.status {
						t.Fatal("End failed to capture committed original diagnostic")
					}
				case "a":
					claim.Source = context.Source
					claim.ClosureOperation = context.Closure
					if issues := checkWeakAuthorization(context, claim, nil); len(issues) != 0 {
						t.Fatal(issues)
					}
					saved = &weakSavedAuthorization{Intent: context.Intent, Value: claim}
					history = append(history, weakAuthorizationLookup{Context: context, Claimed: claim})
					if edge.Event.Fact.Status != "allowed" || edge.Event.Fact.Reason != "authorized" || !edge.Event.ReplyBound ||
						edge.Event.SourceOpen != (context.Status == "open") {
						t.Fatal("original J authorization switch disagrees")
					}
				case "d":
					admitted = true
					if saved == nil {
						t.Fatal("unresolved authorization admitted")
					}
					row := weakReplyAdmission{Intent: context.Intent, Authorization: saved.Value,
						Accepted: true, Reason: "recorded_reply"}
					if issues := checkWeakReplyAdmission(context, *saved, row); len(issues) != 0 {
						t.Fatal(issues)
					}
					if edge.Event.Fact.Status != "accepted" || !edge.Event.Fact.Present || edge.Event.Diagnostic != s.Report {
						t.Fatal("original reply admission changed diagnostic")
					}
				}
				if committed && ended && !admitted {
					if s.Published || s.ReportAdmitted {
						t.Fatal("pending admission invented final diagnostic")
					}
					pending := weakCapturedReplyPending(s.Inner.ParentStatus != "", s.ReportCommitted, s.ReportAdmitted)
					if pending != "reply_admission" {
						t.Fatalf("original captured reply preparation: %s", pending)
					}
					for _, e := range CallableJoinSuccessors(c, Reference, s) {
						if e.Event.Fact.Operation == "j" && e.Event.Fact.Kind == "publish" {
							t.Fatal("published with unresolved reply_admission")
						}
					}
				}
			}
			if saved == nil {
				t.Fatal("missing first result")
			}
			history = append(history, weakAuthorizationLookup{Context: context, Claimed: saved.Value})
			if issues := checkWeakAuthorizationHistory(history); len(issues) != 0 {
				t.Fatal(issues)
			}
			if issues := checkWeakAuthorization(context, saved.Value, saved); len(issues) != 0 {
				t.Fatal(issues)
			}
			publication := []CallableJoinEdge{}
			for _, e := range CallableJoinSuccessors(c, Reference, s) {
				if e.Event.Fact.Operation == "j" && e.Event.Fact.Kind == "publish" {
					publication = append(publication, e)
				}
			}
			if len(publication) != 1 {
				t.Fatalf("publication successors=%d", len(publication))
			}
			o := publication[0].Next.Outcome
			if o.Status != tc.status || o.Reason != "member_"+tc.status || !o.Present || o.Diagnostic != s.Report {
				t.Fatalf("final diagnostic: %+v", o)
			}
			t.Logf("first source=%+v repeat exact; publication=%s/%s present=%t",
				saved.Value.Source, o.Status, o.Reason, o.Present)
		})
	}
}
func TestWeakRefinementAuthorizationInvalidControls(t *testing.T) {
	cases := []struct {
		name   string
		change func(*weakAuthContext, *weakAuthorization)
	}{
		{name: "late-source-closed-denial", change: func(c *weakAuthContext, a *weakAuthorization) {
			c.Status = "failed"
			a.Allowed = false
			a.Reason = "source_closed"
		}},
		{name: "uncommitted", change: func(c *weakAuthContext, a *weakAuthorization) { c.Committed = nil }},
		{name: "changed-descriptor", change: func(c *weakAuthContext, a *weakAuthorization) {
			c.Intent.Payload = "changed"
		}},
		{name: "changed-first-source", change: func(c *weakAuthContext, a *weakAuthorization) { a.Source.Sequence++ }},
		{name: "changed-first-closure", change: func(c *weakAuthContext, a *weakAuthorization) {
			a.ClosureOperation = "invented"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, a := weakAuthFixture()
			tc.change(&c, &a)
			if issues := checkWeakAuthorization(c, a, nil); len(issues) == 0 {
				t.Fatal("invalid first provenance accepted")
			}
		})
	}
	for _, name := range []string{"allowed", "source", "closure", "descriptor"} {
		t.Run("repeat-"+name, func(t *testing.T) {
			c, a := weakAuthFixture()
			saved := weakSavedAuthorization{Intent: c.Intent, Value: a}
			c.Status = "succeeded"
			c.Source.Sequence = 9
			switch name {
			case "allowed":
				a.Allowed = false
				a.Reason = "source_closed"
			case "source":
				a.Source = c.Source
			case "closure":
				a.ClosureOperation = "fresh"
			case "descriptor":
				c.Intent.Obligation = "changed"
				intent := c.Intent
				c.Committed = &intent
			}
			if issues := checkWeakAuthorization(c, a, &saved); len(issues) == 0 {
				t.Fatal("changed repeated first record accepted")
			}
		})
	}
}
func TestWeakRefinementAuthorizationOutsideProfile(t *testing.T) {
	for _, tc := range []struct {
		name, reason         string
		revision, generation int64
	}{
		{name: "revision-priority", reason: "source_revision_changed", revision: 2, generation: 2},
		{name: "generation", reason: "source_generation_changed", revision: 1, generation: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, a := weakAuthFixture()
			c.Source.Revision = tc.revision
			c.SubjectGeneration = tc.generation
			a.Source = c.Source
			a.Allowed = false
			a.Reason = tc.reason
			if issues := checkWeakAuthorization(c, a, nil); len(issues) != 0 {
				t.Fatal(issues)
			}
		})
	}
	t.Run("closed-nonreply", func(t *testing.T) {
		c, a := weakAuthFixture()
		c.Intent.Kind = "invocation"
		intent := c.Intent
		c.Committed = &intent
		c.Status = "failed"
		a.Allowed = false
		a.Reason = "source_closed"
		if issues := checkWeakAuthorization(c, a, nil); len(issues) != 0 {
			t.Fatal(issues)
		}
		c.Status = "open"
		a.Allowed = true
		a.Reason = "authorized"
		saved := weakSavedAuthorization{Intent: c.Intent, Value: a}
		c.Status = "failed"
		c.Source.Sequence++
		if issues := checkWeakAuthorization(c, a, &saved); len(issues) != 0 {
			t.Fatal(issues)
		}
		if reason := weakInvocationAdmission(c, a, false); reason != "source_closed" {
			t.Fatal(reason)
		}
		c.Status = "open"
		if reason := weakInvocationAdmission(c, a, true); reason != "scope_closed" {
			t.Fatal(reason)
		}
	})
	for _, field := range []string{"protocol", "member", "target", "generation", "obligation"} {
		t.Run("binding-"+field, func(t *testing.T) {
			c, a := weakAuthFixture()
			c.Status = "failed"
			switch field {
			case "protocol":
				c.Binding.Protocol = "wrong"
			case "member":
				c.Binding.Member = "wrong"
			case "target":
				c.Binding.CallerSubject = "wrong"
			case "generation":
				c.Binding.CallerGeneration = 2
			case "obligation":
				c.Binding.Obligation = "wrong"
			}
			a.Allowed = false
			a.Reason = "source_closed"
			if issues := checkWeakAuthorization(c, a, nil); len(issues) != 0 {
				t.Fatal(issues)
			}
		})
	}
	t.Run("admission-retains-payload-and-auth", func(t *testing.T) {
		c, a := weakAuthFixture()
		saved := weakSavedAuthorization{Intent: c.Intent, Value: a}
		row := weakReplyAdmission{Intent: c.Intent, Authorization: a, Accepted: true, Reason: "recorded_reply"}
		row.Intent.Payload = "changed"
		if issues := checkWeakReplyAdmission(c, saved, row); len(issues) == 0 {
			t.Fatal("changed diagnostic admitted")
		}
		row.Intent = c.Intent
		row.Authorization.Source.Sequence++
		if issues := checkWeakReplyAdmission(c, saved, row); len(issues) == 0 {
			t.Fatal("changed saved authorization admitted")
		}
	})
	t.Log("outside stable profile: revision/generation denials and nonreply admission remain separate")
}

func TestWeakRefinementAuthorizationHistoryAndPending(t *testing.T) {
	c, a := weakAuthFixture()
	late := c
	late.Status = "failed"
	late.Source.Sequence = 9
	history := []weakAuthorizationLookup{{Context: c, Claimed: a}, {Context: late, Claimed: a}}
	if issues := checkWeakAuthorizationHistory(history); len(issues) != 0 {
		t.Fatal(issues)
	}
	history[0].Claimed.Allowed = false
	history[0].Claimed.Reason = "source_closed"
	history[1].Claimed = history[0].Claimed
	if issues := checkWeakAuthorizationHistory(history); !hasWeakIssue(issues, "first_authorization") {
		t.Fatalf("unvalidated forged first disposition reused: %+v", issues)
	}
	if pending := weakCapturedReplyPending(false, true, false); pending != "members" {
		t.Fatal(pending)
	}
	if pending := weakCapturedReplyPending(true, true, true); pending != "" {
		t.Fatal(pending)
	}
}
