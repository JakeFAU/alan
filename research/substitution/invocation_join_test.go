package substitution

import "testing"

func TestJoinCommitOrder(t *testing.T) {
	c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
	s := JoinInitial(c)
	edges := JoinSuccessors(c, Reference, s)
	if len(edges) != 1 || edges[0].Event.Fact.Kind != "invoke_commit" || edges[0].Event.Fact.Operation != "e" {
		t.Fatalf("initial context must commit candidate first: %+v", edges)
	}
	s = edges[0].Next
	foundS := false
	for _, e := range JoinSuccessors(c, Reference, s) {
		if e.Event.Fact.Kind == "outcome_consume" {
			t.Fatal("consume before second commit")
		}
		if e.Event.Fact.Kind == "invoke_commit" && e.Event.Fact.Operation == "s" {
			foundS = true
		}
	}
	if !foundS {
		t.Fatal("S commitment must be enabled without waiting for candidate")
	}
}

func TestJoinReplyAuthorizationAfterTerminal(t *testing.T) {
	c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
	s := JoinInitial(c)
	s.Component.Ops[entry].Reply = 1
	s.Component.Ops[entry].Freeze = success
	for _, e := range JoinSuccessors(c, Reference, s) {
		if e.Event.Fact.Kind == "reply_authorize" && e.Event.Fact.Operation == "e" {
			if e.Event.Fact.Status != "allowed" || e.Event.SourceOpen || !e.Event.ReplyBound {
				t.Fatalf("bound original reply must survive terminal source: %+v", e.Event)
			}
			return
		}
	}
	t.Fatal("missing separate reply authorization")
}

// This is a hand-written protocol fixture from K1/C_inv, not model exploration
// or a retained-runtime sample. It deliberately terminates members before auth.
func joinOracleFixture(wrongID bool, mode string) (JoinConfig, []JoinEvent) {
	c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: mode}, I: 3, R: 3, Depth: 2}
	events := []JoinEvent{}
	fact := func(op, kind string) Observation {
		r, role := c.Request, "candidate"
		if op == "s" {
			r.Mode, role = "", "s"
		}
		return Observation{Kind: kind, Operation: op, Parent: "root", Role: role, Request: r}
	}
	for _, op := range []string{"e", "s"} {
		events = append(events, JoinEvent{Fact: fact(op, "invoke_commit")})
		f := fact(op, "invoke_authorize")
		f.Status, f.Reason = "allowed", "authorized"
		events = append(events, JoinEvent{Fact: f, SourceOpen: true, RevisionMatches: true, GenerationMatches: true})
	}
	for _, op := range []string{"e", "s"} {
		f := fact(op, "admit")
		f.Status, f.I, f.R = "accepted", 1, 1
		events = append(events, JoinEvent{Fact: f}, JoinEvent{Fact: fact(op, "request_consume")})
		if op == "e" && mode == "pending" {
			continue
		}
		hasReply := op == "s" || mode == "answer"
		id, value := 1, 0
		if op == "e" {
			value = 1
			if wrongID {
				id = 2
			}
		}
		if hasReply {
			f = fact(op, "reply_commit")
			f.Status, f.ID, f.Value = "committed", id, value
			events = append(events, JoinEvent{Fact: f})
		}
		f = fact(op, "terminal_freeze")
		f.Status = "succeeded"
		if op == "e" && mode == "fail" {
			f.Status = "failed"
		}
		events = append(events, JoinEvent{Fact: f})
		if hasReply {
			f = fact(op, "reply_authorize")
			f.Status, f.Reason = "allowed", "authorized"
			events = append(events, JoinEvent{Fact: f, RevisionMatches: true, GenerationMatches: true, ReplyBound: true})
			f = fact(op, "reply_admit")
			f.Status, f.ID, f.Value, f.Present = "accepted", id, value, true
			events = append(events, JoinEvent{Fact: f})
		}
		for _, kind := range []string{"publish", "outcome_deliver"} {
			f = fact(op, kind)
			f.Status, f.Reason = "succeeded", "member_succeeded"
			if hasReply {
				f.ID, f.Value, f.Present = id, value, true
			} else {
				f.Status, f.Reason = "failed", "member_failed"
				if mode == "absent" {
					f.Reason = "missing_reply"
				}
			}
			events = append(events, JoinEvent{Fact: f})
		}
	}
	if mode == "pending" {
		return c, events
	}
	for _, op := range []string{"e", "s"} {
		for _, e := range events {
			if e.Fact.Operation == op && e.Fact.Kind == "publish" {
				f := e.Fact
				f.Kind = "outcome_consume"
				events = append(events, JoinEvent{Fact: f})
				break
			}
		}
	}
	status := "succeeded"
	if wrongID || mode != "answer" {
		status = "failed"
	}
	for _, kind := range []string{"parent_decide", "parent_terminal", "root_publish"} {
		f := Observation{Kind: kind, Operation: "root", Role: "parent", Request: c.Request, Status: status}
		if kind == "root_publish" {
			f.Reason = "member_" + status
		}
		events = append(events, JoinEvent{Fact: f})
	}
	return c, events
}

func TestJoinOracleSeparatesConformanceAndHonestFailure(t *testing.T) {
	c, events := joinOracleFixture(true, "answer")
	if v := CheckJoinPrefix(c, events); len(v) != 0 {
		t.Fatalf("honest wrong-id history: %+v", v)
	}
	if v := CheckJoinComponent(c, events); len(v) == 0 {
		t.Fatal("wrong id must violate K1 independently of parent failure")
	}
	if v := CheckJoinGuarantee(c, events); len(v) == 0 {
		t.Fatal("honest failure must not establish G")
	}
}

func TestJoinOracleRequiresBothOriginalConsumes(t *testing.T) {
	c, events := joinOracleFixture(false, "answer")
	if v := CheckJoinGuarantee(c, events); len(v) != 0 {
		t.Fatalf("specified successful fixture: %+v", v)
	}
	for _, remove := range []string{"e", "s"} {
		t.Run(remove, func(t *testing.T) {
			bad := []JoinEvent{}
			for _, e := range events {
				if e.Fact.Kind != "outcome_consume" || e.Fact.Operation != remove {
					bad = append(bad, e)
				}
			}
			if v := CheckJoinPrefix(c, bad); len(v) == 0 {
				t.Fatal("normal decision skipped required original consumption")
			}
		})
	}
	if v := CheckJoinGuarantee(c, events[:len(events)-1]); len(v) == 0 {
		t.Fatal("missing root publication cannot establish G")
	}
}

func TestJoinOracleHonestNonAnswerPrefixes(t *testing.T) {
	for _, mode := range []string{"fail", "absent", "pending"} {
		t.Run(mode, func(t *testing.T) {
			c, events := joinOracleFixture(false, mode)
			if v := CheckJoinPrefix(c, events); len(v) != 0 {
				t.Fatalf("honest prefix: %+v", v)
			}
			if v := CheckJoinComponent(c, events); len(v) != 0 {
				t.Fatalf("K1 mode conformance: %+v", v)
			}
			if v := CheckJoinGuarantee(c, events); len(v) == 0 {
				t.Fatal("non-answer does not establish positive G")
			}
		})
	}
}

func TestJoinOracleRejectsBindingAndRequestSubstitution(t *testing.T) {
	c, events := joinOracleFixture(false, "answer")
	for _, attack := range []string{"reply-binding", "request-envelope", "role", "root-before-terminal"} {
		t.Run(attack, func(t *testing.T) {
			bad := append([]JoinEvent{}, events...)
			for i := range bad {
				f := &bad[i].Fact
				if attack == "reply-binding" && f.Kind == "reply_authorize" {
					bad[i].ReplyBound = false
					break
				}
				if attack == "request-envelope" && f.Kind == "invoke_commit" && f.Operation == "s" {
					f.Request.Value = 1
					break
				}
				if attack == "role" && f.Kind == "admit" {
					f.Role = "s"
					break
				}
				if attack == "root-before-terminal" && f.Kind == "parent_terminal" {
					bad[i], bad[i+1] = bad[i+1], bad[i]
					break
				}
			}
			if v := CheckJoinPrefix(c, bad); len(v) == 0 {
				t.Fatal("altered protocol fact accepted")
			}
		})
	}
}

func TestJoinReferenceRetainsBothChoices(t *testing.T) {
	c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
	s := JoinInitial(c)
	s.ParentStage, s.Component.I, s.Component.R = 1, 1, 1
	s.InvocationAuthorization[entry] = 1
	s.Component.Ops[sibling].Enabled = true
	o := &s.Component.Ops[entry]
	o.Ticket, o.Admission, o.RequestStage, o.PC = spent, 1, 2, choose
	s.Component.PrivatePC[entry] = 1
	if err := CheckJoinState(c, s); err != nil {
		t.Fatal(err)
	}
	choices := map[string]bool{}
	for _, edge := range JoinSuccessors(c, Reference, s) {
		choices[edge.Event.Fact.Kind] = true
		if JoinRank(edge.Next) <= JoinRank(s) || JoinRank(edge.Next) > JoinRankCap {
			t.Fatal("nonincreasing/out-of-bound rank")
		}
		if err := CheckJoinState(c, edge.Next); err != nil {
			t.Fatal(err)
		}
	}
	if !choices["choose_local"] || !choices["choose_helper"] {
		t.Fatal("reference discarded a K1 choice")
	}
}

func TestJoinScarceAdmissionKeepsBothPossibleLosers(t *testing.T) {
	// Hand-built boundary seeds test guards, not reachable-schedule qualification.
	for _, loser := range []int{helper, sibling} {
		t.Run(name(loser), func(t *testing.T) {
			c := JoinConfig{Request: Request{ID: 1, Value: 0, Mode: "answer"}, I: 2, R: 2, Depth: 2}
			s := JoinInitial(c)
			s.ParentStage, s.Component.I, s.Component.R = 2, 2, 2
			for _, i := range []int{entry, helper, sibling} {
				o := &s.Component.Ops[i]
				o.Enabled, o.Ticket, o.Admission, o.RequestStage, o.PC = true, spent, 1, 1, receive
				s.InvocationAuthorization[i] = 1
			}
			s.Component.Ops[entry].RequestStage, s.Component.Ops[entry].PC, s.Component.Ops[entry].Choice = 2, await, 2
			s.Component.Ops[loser].Ticket, s.Component.Ops[loser].Admission = committed, 0
			s.Component.Ops[loser].RequestStage, s.Component.Ops[loser].PC = 0, uncreated
			if err := CheckJoinState(c, s); err != nil {
				t.Fatal(err)
			}
			for _, edge := range JoinSuccessors(c, Reference, s) {
				if edge.Event.Fact.Kind != "admit" || edge.Event.Fact.Operation != name(loser) {
					continue
				}
				f := edge.Event.Fact
				if f.Status != "rejected" || f.Reason != "invocation_limit" || f.I != 0 || f.R != 0 || edge.Next.Component.I != 2 {
					t.Fatalf("lost real refusal/charge: %+v", edge)
				}
				if err := CheckJoinState(c, edge.Next); err != nil {
					t.Fatal(err)
				}
				return
			}
			t.Fatal("capacity exhaustion silently disabled an actual admission")
		})
	}
}

func TestJoinRankDomainGuard(t *testing.T) {
	c := JoinConfigurations()[0]
	s := JoinInitial(c)
	if JoinRankCap != 10621 || CheckJoinState(c, s) != nil {
		t.Fatal("initial rank/domain declaration")
	}
	s.Component.PrivatePC[helper] = 7
	if err := CheckJoinState(c, s); err == nil {
		t.Fatal("private PC escaped declared bound")
	}
	s = JoinInitial(c)
	s.Component.Ops[sibling].Input.Mode = "answer"
	if err := CheckJoinState(c, s); err == nil {
		t.Fatal("S request envelope acquired undeclared mode")
	}
}

// joinHelperOracleFixture inserts a specification-authored helper history into
// the existing manual protocol fixture. It is not a graph or runtime execution.
func joinHelperOracleFixture(helperBehavior string, candidateReply bool) (JoinConfig, []JoinEvent) {
	c, original := joinOracleFixture(false, "answer")
	if helperBehavior == "refused" {
		c.I, c.R = 2, 2
	}
	candidateFailed := helperBehavior != "success" && helperBehavior != "wrong_reply"
	var sAdmission JoinEvent
	for _, event := range original {
		if event.Fact.Operation == "s" && event.Fact.Kind == "admit" {
			sAdmission = event
		}
	}
	helperFact := func(kind string) Observation {
		return Observation{Kind: kind, Operation: "h", Parent: "e", Role: "helper", Request: c.Request}
	}
	events := []JoinEvent{}
	for _, event := range original {
		f := &event.Fact
		if f.Operation == "s" && f.Kind == "admit" {
			continue
		}
		if f.Operation == "e" {
			if !candidateReply && (f.Kind == "reply_commit" || f.Kind == "reply_authorize" || f.Kind == "reply_admit") {
				continue
			}
			if candidateFailed && (f.Kind == "terminal_freeze" || f.Kind == "publish" || f.Kind == "outcome_deliver" || f.Kind == "outcome_consume") {
				f.Status = "failed"
				if f.Kind != "terminal_freeze" {
					f.Reason = "member_failed"
					if !candidateReply {
						f.Present, f.ID, f.Value = false, 0, 0
					}
				}
			}
		}
		if f.Operation == "root" && candidateFailed {
			f.Status = "failed"
			if f.Kind == "root_publish" {
				f.Reason = "member_failed"
			}
		}
		events = append(events, event)
		if f.Operation != "e" || f.Kind != "request_consume" {
			continue
		}
		events = append(events, sAdmission, JoinEvent{Fact: helperFact("invoke_commit")})
		h := helperFact("invoke_authorize")
		h.Status, h.Reason = "allowed", "authorized"
		events = append(events, JoinEvent{Fact: h, SourceOpen: true, RevisionMatches: true, GenerationMatches: true})
		h = helperFact("admit")
		present := helperBehavior == "success" || helperBehavior == "reply_then_fail" || helperBehavior == "wrong_reply"
		id := c.Request.ID
		if helperBehavior == "wrong_reply" {
			id = 2
		}
		status, reason := "succeeded", "member_succeeded"
		if helperBehavior == "refused" {
			h.Status, h.Reason = "rejected", "invocation_limit"
			events = append(events, JoinEvent{Fact: h})
			status, reason = "rejected", "invocation_limit"
		} else {
			h.Status, h.I, h.R = "accepted", 1, 1
			events = append(events, JoinEvent{Fact: h}, JoinEvent{Fact: helperFact("request_consume")})
			if present {
				h = helperFact("reply_commit")
				h.Status, h.ID, h.Value = "committed", id, 1-c.Request.Value
				events = append(events, JoinEvent{Fact: h})
			}
			h = helperFact("terminal_freeze")
			h.Status = "succeeded"
			if helperBehavior == "failed" || helperBehavior == "reply_then_fail" {
				h.Status, status, reason = "failed", "failed", "member_failed"
			} else if !present {
				status, reason = "failed", "missing_reply"
			}
			events = append(events, JoinEvent{Fact: h})
			if present {
				h = helperFact("reply_authorize")
				h.Status, h.Reason = "allowed", "authorized"
				events = append(events, JoinEvent{Fact: h, RevisionMatches: true, GenerationMatches: true, ReplyBound: true})
				h = helperFact("reply_admit")
				h.Status, h.Present, h.ID, h.Value = "accepted", true, id, 1-c.Request.Value
				events = append(events, JoinEvent{Fact: h})
			}
		}
		for _, kind := range []string{"publish", "outcome_deliver", "outcome_consume"} {
			h = helperFact(kind)
			h.Status, h.Reason = status, reason
			if present {
				h.Present, h.ID, h.Value = true, id, 1-c.Request.Value
			}
			events = append(events, JoinEvent{Fact: h})
		}
	}
	return c, events
}

func TestJoinComponentHelperDiscipline(t *testing.T) {
	for _, test := range []struct {
		name, behavior, wantCode                    string
		candidateReply, wantG, stopAfterHelperReply bool
	}{
		{name: "refused_no_reply_control", behavior: "refused"},
		{name: "successful_helper_control", behavior: "success", candidateReply: true, wantG: true},
		{name: "candidate_reply_after_refusal", behavior: "refused", candidateReply: true, wantCode: "helper_result"},
		{name: "candidate_reply_after_failed_helper", behavior: "failed", candidateReply: true, wantCode: "helper_result"},
		{name: "helper_failure_without_reply", behavior: "failed", wantCode: "helper_terminal"},
		{name: "helper_success_without_reply", behavior: "absent", wantCode: "helper_terminal"},
		{name: "helper_correct_reply_then_failure", behavior: "reply_then_fail", wantCode: "helper_terminal"},
		{name: "helper_wrong_reply_before_terminal", behavior: "wrong_reply", candidateReply: true, wantCode: "helper_reply", stopAfterHelperReply: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, events := joinHelperOracleFixture(test.behavior, test.candidateReply)
			if test.stopAfterHelperReply {
				for i, event := range events {
					if event.Fact.Operation == "h" && event.Fact.Kind == "reply_commit" {
						events = events[:i+1]
						break
					}
				}
			}
			if violations := CheckJoinPrefix(c, events); len(violations) != 0 {
				t.Fatalf("truthful protocol prefix rejected: %+v", violations)
			}
			violations := CheckJoinComponent(c, events)
			if test.wantCode == "" {
				if len(violations) != 0 {
					t.Fatalf("conforming control rejected: %+v", violations)
				}
			} else {
				found := false
				for _, violation := range violations {
					if violation.Code == test.wantCode {
						found = true
					}
				}
				if !found {
					t.Fatalf("expected %s, got %+v", test.wantCode, violations)
				}
			}
			if violations := CheckJoinGuarantee(c, events); (len(violations) == 0) != test.wantG {
				t.Fatalf("independent G verdict: %+v", violations)
			}
		})
	}
}
