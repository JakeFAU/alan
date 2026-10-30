package substitution

// reasonPrefixState is part of exploration identity. CandidatePending is new
// monitor memory: the old protocol monitor deliberately ignores local PC labels.
type reasonPrefixState struct {
	joinPrefixState
	Classification   string
	CandidatePending bool
}

// reasonPrefixClassification restates the table on retained envelopes, without
// calling the model predicate or consulting the request mode.
func reasonPrefixClassification(c JoinConfig, s reasonPrefixState) string {
	e, other := s.Ops[entry], s.Ops[sibling]
	if !e.Consumed || !other.Consumed {
		return ""
	}
	o := other.Outcome
	if o.Status != "succeeded" || o.Reason != "member_succeeded" || !o.Present {
		return ReasonInvalid
	}
	if o.ID != c.Request.ID || o.Value != 0 {
		return ReasonInvalid
	}
	o = e.Outcome
	switch o.Status {
	case "succeeded":
		if o.Reason != "member_succeeded" || !o.Present {
			return ReasonInvalid
		}
		if o.ID == c.Request.ID && o.Value == 1-c.Request.Value {
			return ReasonAnswer
		}
	case "failed":
		if o.Present {
			return ReasonInvalid
		}
		switch o.Reason {
		case "member_failed":
			return ReasonFailed
		case "missing_reply":
			return ReasonMissingReply
		}
	}
	return ReasonInvalid
}

func advanceReasonPrefix(
	c JoinConfig,
	s reasonPrefixState,
	event ConsumerReasonEvent,
	index int,
) (reasonPrefixState, []Violation) {
	f := event.Fact
	bad := func(code, detail string) (reasonPrefixState, []Violation) {
		return s, []Violation{{Index: index, Code: code, Detail: detail}}
	}
	n := s
	if f.Kind != "parent_classify" && event.Classification != "" {
		return bad("classification_event", "only a committed classification event can carry the tag")
	}
	if f.Operation != "root" {
		if s.CandidatePending && (f.Operation == "e" || f.Operation == "h" && f.Kind == "invoke_commit") {
			return bad("pending_activity", "designated unmatched receive cannot resume or create helper work")
		}
		next, failures := advanceJoinPolicyPrefix(c, s.joinPrefixState, event.JoinEvent, index, consumeSFirst)
		if len(failures) != 0 {
			return s, failures
		}
		n.joinPrefixState = next
		if f.Kind == "pending_receive" {
			if f.Operation != "e" || s.Ops[entry].ReplyCommitted || s.Ops[entry].Published {
				return bad("pending_receive", "pending must be the original candidate without reply or publication")
			}
			if s.Ops[helper].Committed && !s.Ops[helper].Consumed {
				return bad("pending_helper", "pending candidate must first consume its original helper")
			}
			n.CandidatePending = true
		}
		return n, nil
	}
	identityOK := f.Parent == "" && f.Role == "parent" && f.Request == c.Request
	payloadFree := !f.Present && f.ID == 0 && f.Value == 0
	if !identityOK || !payloadFree || f.I != 0 || f.R != 0 || f.E != 0 {
		return bad("parent_identity", "parent event changed original identity/payload/accounting")
	}
	if f.Kind != "root_publish" && f.Reason != "" {
		return bad("parent_reason", "child reason cannot be rewritten as a parent decision reason")
	}
	switch f.Kind {
	case "parent_classify":
		if s.Classification != "" || s.Decision != "" || s.ParentStatus != "" || f.Status != "" {
			return bad("classification_order", "classification must be unique, before decision and End")
		}
		want := reasonPrefixClassification(c, s)
		if want == "" || event.Classification != want {
			return bad("classification", "tag must follow both actual consumed envelopes")
		}
		n.Classification = event.Classification
	case "parent_decide":
		joined := s.Ops[entry].Consumed && s.Ops[sibling].Consumed
		if !joined || s.Classification == "" || s.Decision != "" || s.ParentStatus != "" {
			return bad("required_classification", "decision requires both joins and the original classification write")
		}
		want := "failed"
		if s.Classification == ReasonAnswer {
			want = "succeeded"
		}
		if f.Status != want {
			return bad("parent_decision", "decision differs from the actual classified branch")
		}
		n.Decision = f.Status
	case "parent_terminal", "root_publish":
		if s.Classification == "" {
			return bad("required_classification", "End/publication requires the original classification")
		}
		next, failures := advanceJoinPolicyPrefix(c, s.joinPrefixState, event.JoinEvent, index, consumeSFirst)
		n.joinPrefixState = next
		return n, failures
	default:
		return bad("parent_event", f.Kind)
	}
	return n, nil
}

func checkReasonPrefix(c JoinConfig, events []ConsumerReasonEvent) (reasonPrefixState, []Violation) {
	state := reasonPrefixState{}
	if err := ValidateJoinConfig(c); err != nil {
		return state, []Violation{{Index: -1, Code: "config", Detail: err.Error()}}
	}
	violations := []Violation{}
	for index, event := range events {
		next, found := advanceReasonPrefix(c, state, event, index)
		violations = append(violations, found...)
		state = next
	}
	return state, violations
}

// CheckConsumerReasonPrefix checks truthful protocol/classification prefixes;
// it does not equate prefix safety with component conformance or progress.
func CheckConsumerReasonPrefix(c JoinConfig, events []ConsumerReasonEvent) []Violation {
	_, failures := checkReasonPrefix(c, events)
	return failures
}

func reasonJoinEvents(events []ConsumerReasonEvent) []JoinEvent {
	out := make([]JoinEvent, 0, len(events))
	for _, event := range events {
		out = append(out, event.JoinEvent)
	}
	return out
}

func CheckConsumerReasonComponent(c JoinConfig, events []ConsumerReasonEvent) []Violation {
	return CheckJoinComponent(c, reasonJoinEvents(events))
}

// CheckConsumerReasonGuarantee checks G_outcome, including the designated pending
// boundary. It derives expectations from mode but checks actual retained facts.
// Census maximality and independent local progress are separate obligations.
func CheckConsumerReasonGuarantee(c JoinConfig, events []ConsumerReasonEvent) []Violation {
	s, failures := checkReasonPrefix(c, events)
	fail := func(code, detail string) {
		failures = append(failures, Violation{Index: len(events), Code: code, Detail: detail})
	}
	if !ConsumerReasonEntitled(c) {
		fail("outcome_profile", "G_outcome requires the four-mode I=R3/depth2 profile")
	}
	e, h, other := s.Ops[entry], s.Ops[helper], s.Ops[sibling]
	validS := other.Consumed && other.Outcome.Status == "succeeded" &&
		other.Outcome.Reason == "member_succeeded" && other.Outcome.Present &&
		other.Outcome.ID == c.Request.ID && other.Outcome.Value == 0
	if !validS {
		fail("required_s", "original S must be consumed with its authentic successful id/zero envelope")
	}
	if h.Committed {
		validHelper := h.Consumed && h.Admitted && h.Outcome.Status == "succeeded" &&
			h.Outcome.Reason == "member_succeeded" && h.Outcome.Present &&
			h.Outcome.ID == c.Request.ID && h.Outcome.Value == 1-c.Request.Value
		if !validHelper {
			fail("required_helper", "chosen helper must finish and be consumed before candidate mode action")
		}
	}
	wantCharge := 2
	if h.Committed {
		wantCharge++
	}
	if !e.Admitted || !e.Requested || !other.Admitted || s.I != wantCharge || s.R != wantCharge {
		fail("outcome_resource", "original admissions/request and one-charge spending are required")
	}
	if c.Request.Mode == "pending" {
		quiescent := s.CandidatePending && !e.ReplyCommitted && e.Terminal == "" &&
			!e.Published && !e.Delivered && !e.Consumed
		open := s.Classification == "" && s.Decision == "" && s.ParentStatus == "" && !s.RootPublished
		if !quiescent || !open {
			fail("pending_boundary", "S consumed, original candidate unmatched, and unclassified open parent required")
		}
		return failures
	}
	wantClass, wantStatus, wantReason := ReasonAnswer, "succeeded", "member_succeeded"
	switch c.Request.Mode {
	case "fail":
		wantClass, wantStatus, wantReason = ReasonFailed, "failed", "member_failed"
	case "absent":
		wantClass, wantStatus, wantReason = ReasonMissingReply, "failed", "missing_reply"
	}
	if !e.Consumed || e.Outcome.Status != wantStatus || e.Outcome.Reason != wantReason {
		fail("candidate_outcome", "candidate status/reason differs from the requested K1 mode")
	}
	if c.Request.Mode == "answer" {
		if !e.Outcome.Present || e.Outcome.ID != c.Request.ID || e.Outcome.Value != 1-c.Request.Value {
			fail("answer_payload", "answer requires original id/complement and presence")
		}
	} else if e.Outcome.Present {
		fail("absent_payload", "fail/absent candidate response must be absent")
	}
	completed := s.Classification == wantClass && s.Decision == wantStatus &&
		s.ParentStatus == wantStatus && s.RootPublished
	if !completed {
		fail("outcome_completion", "matching classification, decision, End and root publication required")
	}
	return failures
}
