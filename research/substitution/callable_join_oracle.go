package substitution

// The public monitor sees no child states. An authenticated false diagnostic can
// pass this monitor and O's policy while failing the separate K_join monitor.
type callablePublicState struct {
	Committed, Authorized, Admitted, Requested     bool
	ReplyCommitted, ReplyAuthorized, ReplyAdmitted bool
	Report                                         JoinDiagnostic
	Terminal                                       string
	Published, Delivered, Consumed                 bool
	Outcome                                        CallableJoinOutcome
	Decision, DecisionStatus, OuterTerminal        string
	RootPublished                                  bool
}

// All future-relevant memory, including pending/selected evidence, is keyed.
type callablePrefixState struct {
	Inner  reasonPrefixState
	Public callablePublicState
}

func callablePublicDecision(r Request, o CallableJoinOutcome) (string, string) {
	d := o.Diagnostic
	if !o.Present || d.ID != r.ID {
		return "invalid_join_outcome", "failed"
	}
	switch d.Classification {
	case ReasonAnswer:
		valid := o.Status == "succeeded" && o.Reason == "member_succeeded" &&
			d.Count == 1 && d.AnswerID == r.ID && d.AnswerValue+r.Value == 1
		if valid {
			return ReasonAnswer, "succeeded"
		}
	case ReasonFailed, ReasonMissingReply, ReasonInvalid:
		valid := o.Status == "failed" && o.Reason == "member_failed" &&
			d.Count == 0 && d.AnswerID == 0 && d.AnswerValue == 0
		if valid {
			return d.Classification, "failed"
		}
	}
	return "invalid_join_outcome", "failed"
}

func callableEventOutcome(event CallableJoinEvent) CallableJoinOutcome {
	f := event.Fact
	return CallableJoinOutcome{Status: f.Status, Reason: f.Reason, Present: f.Present,
		Diagnostic: event.Diagnostic}
}

func callableDiagnosticFinite(d JoinDiagnostic) bool {
	return d.ID >= 0 && d.ID <= 2 && d.Count >= 0 && d.Count <= 1 &&
		d.AnswerID >= 0 && d.AnswerID <= 2 && d.AnswerValue >= 0 && d.AnswerValue <= 1 &&
		joinFiniteString(d.Classification, "", ReasonAnswer, ReasonFailed, ReasonMissingReply, ReasonInvalid)
}

func advanceCallablePublic(
	c CallableJoinConfig,
	s callablePublicState,
	event CallableJoinEvent,
	index int,
) (callablePublicState, []Violation) {
	f := event.Fact
	n := s
	bad := func(code, detail string) (callablePublicState, []Violation) {
		return s, []Violation{{Index: index, Code: code, Detail: detail}}
	}
	if f.Operation != "j" && f.Operation != "root" {
		return n, nil // Projection discards only declared private inner events.
	}
	if f.Operation == "j" && (f.Kind == "join_classify" || f.Kind == "join_decide") {
		return n, nil // J's private policy is not an input to O's policy checker.
	}
	identity := f.Request == c.Request && f.ID == 0 && f.Value == 0 && f.E == 0
	if !identity || f.Kind != "admit" && (f.I != 0 || f.R != 0) {
		return bad("public_identity", "public request/payload/accounting metadata changed")
	}
	if !callableDiagnosticFinite(event.Diagnostic) {
		return bad("diagnostic_domain", "diagnostic outside the declared finite semantic domain")
	}
	if event.Classification != "" {
		return bad("classification_event", "classification is a separate committed event")
	}
	if f.Operation == "root" {
		if f.Parent != "" || f.Role != "caller" || f.Present || event.Diagnostic != (JoinDiagnostic{}) {
			return bad("outer_identity", "outer event is not the original caller")
		}
		switch f.Kind {
		case "outer_decide":
			want, status := callablePublicDecision(c.Request, s.Outcome)
			if !s.Consumed || s.Decision != "" || s.OuterTerminal != "" ||
				event.Decision != want || f.Status != status || f.Reason != "" {
				return bad("outer_decision", "decision must use original selected public evidence")
			}
			n.Decision, n.DecisionStatus = want, status
		case "outer_terminal":
			if s.Decision == "" || s.OuterTerminal != "" || f.Status != s.DecisionStatus || f.Reason != "" {
				return bad("outer_terminal", "End lacks original committed public decision")
			}
			n.OuterTerminal = f.Status
		case "root_publish":
			if s.OuterTerminal == "" || s.RootPublished || f.Status != s.OuterTerminal ||
				f.Reason != "member_"+s.OuterTerminal {
				return bad("outer_publication", "root publication lacks original End")
			}
			n.RootPublished = true
		default:
			return bad("outer_event", f.Kind)
		}
		if f.Kind != "outer_decide" && event.Decision != "" {
			return bad("decision_event", "decision tag without committed decision")
		}
		return n, nil
	}
	if f.Parent != "root" || f.Role != "join" || event.Decision != "" {
		return bad("join_identity", "J must be the original caller-owned invocation")
	}
	if f.Kind != "invoke_commit" && !s.Committed {
		return bad("join_commit", "event lacks original J commitment")
	}
	payloadEvent := f.Kind == "reply_commit" || f.Kind == "reply_admit" ||
		f.Kind == "publish" || f.Kind == "outcome_deliver" || f.Kind == "outcome_consume"
	if !payloadEvent && (f.Present || event.Diagnostic != (JoinDiagnostic{})) {
		return bad("diagnostic_event", "diagnostic occurs outside report/public outcome")
	}
	switch f.Kind {
	case "invoke_commit":
		if s.Committed || s.OuterTerminal != "" || f.Status != "" || f.Reason != "" {
			return bad("join_commit", "duplicate or closed-caller J commit")
		}
		n.Committed = true
	case "invoke_authorize", "reply_authorize":
		stable := event.RevisionMatches && event.GenerationMatches && !event.SourceRetired
		if !stable || f.Status != "allowed" || f.Reason != "authorized" {
			return bad("public_authorization", "outside stable original binding authority")
		}
		if f.Kind == "invoke_authorize" {
			if s.Authorized || !event.SourceOpen || s.OuterTerminal != "" || event.ReplyBound {
				return bad("join_authorization", "invocation requires the open original caller")
			}
			n.Authorized = true
		} else {
			if !s.ReplyCommitted || s.ReplyAuthorized || !event.ReplyBound ||
				event.SourceOpen != (s.Terminal == "") {
				return bad("join_reply_authorization", "original bound reply/snapshot required, even after End")
			}
			n.ReplyAuthorized = true
		}
	case "admit":
		if !s.Authorized || s.Admitted || s.OuterTerminal != "" ||
			f.Status != "accepted" || f.Reason != "" || f.I != 1 || f.R != 1 {
			return bad("join_admission", "fresh entitled J admission charges exactly once")
		}
		n.Admitted = true
	case "request_consume":
		if !s.Admitted || s.Requested || s.Terminal != "" || f.Status != "" || f.Reason != "" {
			return bad("join_request", "original J request must be consumed once")
		}
		n.Requested = true
	case "reply_commit":
		if !s.Requested || s.Terminal != "" || s.ReplyCommitted ||
			f.Status != "committed" || f.Reason != "" || f.Present {
			return bad("join_reply", "report needs original open request, once-only Send")
		}
		n.ReplyCommitted, n.Report = true, event.Diagnostic
	case "reply_admit":
		if !s.ReplyAuthorized || s.ReplyAdmitted || f.Status != "accepted" ||
			f.Reason != "" || !f.Present || event.Diagnostic != s.Report {
			return bad("join_reply_admission", "report must preserve original authorized payload")
		}
		n.ReplyAdmitted = true
	case "terminal_freeze":
		if !s.Requested || s.Terminal != "" || !joinFiniteString(f.Status, "succeeded", "failed") || f.Reason != "" {
			return bad("join_terminal", "J terminal must follow its original request once")
		}
		n.Terminal = f.Status
	case "publish":
		if s.Terminal == "" || s.Published || s.ReplyCommitted && !s.ReplyAdmitted {
			return bad("join_publication_order", "publication waits for End and captured reply disposition")
		}
		want := CallableJoinOutcome{Status: s.Terminal, Reason: "member_" + s.Terminal}
		if s.ReplyAdmitted {
			want.Present, want.Diagnostic = true, s.Report
		} else if s.Terminal == "succeeded" {
			want.Status, want.Reason = "failed", "missing_reply"
		}
		if callableEventOutcome(event) != want {
			return bad("join_publication", "J envelope must retain admitted report, including failed End")
		}
		n.Published, n.Outcome = true, want
	case "outcome_deliver":
		if !s.Published || s.Delivered || callableEventOutcome(event) != s.Outcome {
			return bad("join_delivery", "delivery changed or duplicated original publication")
		}
		n.Delivered = true
	case "outcome_consume":
		if !s.Delivered || s.Consumed || s.OuterTerminal != "" || callableEventOutcome(event) != s.Outcome {
			return bad("join_consumption", "caller must select its original delivered outcome once")
		}
		n.Consumed = true
	default:
		return bad("join_event", f.Kind)
	}
	return n, nil
}

func advanceCallablePrefix(
	c CallableJoinConfig,
	s callablePrefixState,
	event CallableJoinEvent,
	index int,
) (callablePrefixState, []Violation) {
	n := s
	f := event.Fact
	bad := func(code, detail string) (callablePrefixState, []Violation) {
		return s, []Violation{{Index: index, Code: code, Detail: detail}}
	}
	public, failures := advanceCallablePublic(c, s.Public, event, index)
	if len(failures) != 0 {
		return s, failures
	}
	n.Public = public
	if f.Operation == "root" {
		return n, nil
	}
	if f.Operation != "j" {
		if joinIndex(f.Operation) < 0 || !s.Public.Requested || s.Public.Terminal != "" {
			return bad("inner_lifetime", "inner event outside original open requested J")
		}
		if event.Classification != "" || event.Diagnostic != (JoinDiagnostic{}) || event.Decision != "" {
			return bad("inner_metadata", "inner protocol event carries new public policy metadata")
		}
		old := ConsumerReasonEvent{JoinEvent: event.JoinEvent}
		if f.Operation == "e" || f.Operation == "s" {
			if f.Parent != "j" {
				return bad("inner_owner", "original e/S are owned by J")
			}
			old.Fact.Parent = "root"
		}
		inner, found := advanceReasonPrefix(callableRelativeConfig(c), s.Inner, old, index)
		n.Inner = inner
		return n, found
	}
	if f.Kind == "join_classify" || f.Kind == "join_decide" || f.Kind == "terminal_freeze" {
		if !s.Public.Requested || f.Parent != "root" || f.Role != "join" {
			return bad("join_policy_identity", "classification/decision must belong to original requested J")
		}
		if event.Diagnostic != (JoinDiagnostic{}) || event.Decision != "" {
			return bad("join_policy_metadata", "classification/decision is not a public diagnostic event")
		}
		kind := "parent_classify"
		switch f.Kind {
		case "join_decide":
			kind = "parent_decide"
		case "terminal_freeze":
			kind = "parent_terminal"
			if !s.Public.ReplyCommitted {
				return bad("required_diagnostic", "K_join End requires its earlier diagnostic Send")
			}
		}
		old := ConsumerReasonEvent{JoinEvent: event.JoinEvent, Classification: event.Classification}
		old.Fact.Kind, old.Fact.Operation, old.Fact.Parent, old.Fact.Role = kind, "root", "", "parent"
		inner, found := advanceReasonPrefix(callableRelativeConfig(c), s.Inner, old, index)
		n.Inner = inner
		return n, found
	}
	if f.Kind == "reply_commit" {
		e := s.Inner.Ops[entry]
		want := JoinDiagnostic{ID: c.Request.ID, Classification: s.Inner.Classification}
		if s.Inner.Classification == ReasonAnswer {
			want.Count, want.AnswerID, want.AnswerValue = 1, e.Outcome.ID, e.Outcome.Value
		}
		if s.Inner.Classification == "" || s.Inner.Decision == "" || event.Diagnostic != want {
			return bad("diagnostic_truth", "report must describe actual original selected inner outcomes")
		}
	}
	return n, nil
}

func checkCallablePrefix(
	c CallableJoinConfig,
	events []CallableJoinEvent,
	outer bool,
) (callablePrefixState, []Violation) {
	state := callablePrefixState{}
	if err := ValidateCallableJoinConfig(c); err != nil {
		return state, []Violation{{Index: -1, Code: "config", Detail: err.Error()}}
	}
	failures := []Violation{}
	for index, event := range events {
		if !outer && event.Fact.Operation == "root" {
			continue // K_join projection has no O private decision/End/publication.
		}
		next, found := advanceCallablePrefix(c, state, event, index)
		state = next
		failures = append(failures, found...)
	}
	return state, failures
}

func CheckCallableJoinPrefix(c CallableJoinConfig, events []CallableJoinEvent) []Violation {
	_, failures := checkCallablePrefix(c, events, true)
	return failures
}

// CheckCallableJoinPublic checks only J's public protocol and O's public policy.
// It does not assert inner truth, K1 conformance, K_join or progress/maximality.
func CheckCallableJoinPublic(c CallableJoinConfig, events []CallableJoinEvent) []Violation {
	_, failures := checkCallablePublic(c, events)
	return failures
}

func checkCallablePublic(c CallableJoinConfig, events []CallableJoinEvent) (callablePublicState, []Violation) {
	state := callablePublicState{}
	failures := []Violation{}
	if err := ValidateCallableJoinConfig(c); err != nil {
		return state, []Violation{{Index: -1, Code: "config", Detail: err.Error()}}
	}
	for index, event := range events {
		next, found := advanceCallablePublic(c, state, event, index)
		state = next
		failures = append(failures, found...)
	}
	return state, failures
}

func CheckCallableJoinComponent(c CallableJoinConfig, events []CallableJoinEvent) []Violation {
	facts := []JoinEvent{}
	for _, event := range events {
		facts = append(facts, event.JoinEvent)
	}
	return CheckJoinComponent(callableRelativeConfig(c), facts)
}

// CheckCallableJoinContract is K_join under its public caller environment, with
// no dependence on O's private PCs, decision, End or root publication. A trace
// verdict does not by itself establish maximality or fairness.
func CheckCallableJoinContract(c CallableJoinConfig, events []CallableJoinEvent) []Violation {
	s, failures := checkCallablePrefix(c, events, false)
	fail := func(code, detail string) {
		failures = append(failures, Violation{Index: len(events), Code: code, Detail: detail})
	}
	if !CallableJoinEntitled(c) {
		fail("join_profile", "K_join requires all-mode I=R4/depth3 entitlement")
	}
	e, h, other := s.Inner.Ops[entry], s.Inner.Ops[helper], s.Inner.Ops[sibling]
	validS := other.Consumed && other.Outcome.Status == "succeeded" &&
		other.Outcome.Reason == "member_succeeded" && other.Outcome.Present &&
		other.Outcome.ID == c.Request.ID && other.Outcome.Value == 0
	if !validS {
		fail("join_required_s", "original S must actually be consumed with correct id/zero")
	}
	if h.Committed {
		valid := h.Consumed && h.Admitted && h.Outcome.Status == "succeeded" &&
			h.Outcome.Reason == "member_succeeded" && h.Outcome.Present &&
			h.Outcome.ID == c.Request.ID && h.Outcome.Value == 1-c.Request.Value
		if !valid {
			fail("join_required_helper", "chosen original helper must complete and be consumed")
		}
	}
	charge := 2
	if h.Committed {
		charge++
	}
	if !s.Public.Admitted || !s.Public.Requested || !e.Admitted || !e.Requested ||
		!other.Admitted || s.Inner.I != charge || s.Inner.R != charge {
		fail("join_resources", "original requested admissions and prior J + exact inner charges required")
	}
	if c.Request.Mode == "pending" {
		pending := s.Inner.CandidatePending && !e.ReplyCommitted && e.Terminal == "" &&
			!e.Published && !e.Delivered && !e.Consumed
		open := s.Inner.Classification == "" && s.Inner.Decision == "" &&
			!s.Public.ReplyCommitted && s.Public.Terminal == "" && !s.Public.Published
		if !pending || !open {
			fail("join_pending", "unmatched original candidate and S consumed, with open unclassified J required")
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
		fail("join_candidate_mode", "actual candidate selected outcome differs from K1 mode")
	}
	if c.Request.Mode == "answer" {
		if !e.Outcome.Present || e.Outcome.ID != c.Request.ID || e.Outcome.Value != 1-c.Request.Value {
			fail("join_candidate_payload", "original correct present complement required")
		}
	} else if e.Outcome.Present {
		fail("join_candidate_presence", "fail/absent candidate response must be absent")
	}
	complete := s.Inner.Classification == wantClass && s.Inner.Decision == wantStatus &&
		s.Public.Terminal == wantStatus && s.Public.Published && s.Public.Outcome.Present &&
		s.Public.Outcome.Reason == "member_"+wantStatus && s.Public.Report.Classification == wantClass
	if !complete {
		fail("join_completion", "truthful diagnostic, matching End and original J publication required")
	}
	return failures
}

// CheckCallableJoinOuterGuarantee checks G_outer solely from public events and
// original input. J truth remains the separate K_join premise.
func CheckCallableJoinOuterGuarantee(c CallableJoinConfig, events []CallableJoinEvent) []Violation {
	s, failures := checkCallablePublic(c, events)
	fail := func(code, detail string) {
		failures = append(failures, Violation{Index: len(events), Code: code, Detail: detail})
	}
	if !CallableJoinEntitled(c) {
		fail("outer_profile", "G_outer requires the declared entitlement")
	}
	if c.Request.Mode == "pending" {
		if !s.Requested || s.Published || s.Consumed || s.Decision != "" || s.OuterTerminal != "" || s.RootPublished {
			fail("outer_pending", "public original J await remains open without a fabricated outcome")
		}
		return failures
	}
	want, status := ReasonAnswer, "succeeded"
	switch c.Request.Mode {
	case "fail":
		want, status = ReasonFailed, "failed"
	case "absent":
		want, status = ReasonMissingReply, "failed"
	}
	if !s.Consumed || s.Decision != want || s.OuterTerminal != status || !s.RootPublished {
		fail("outer_completion", "matching original public selection, committed decision, End and root publication required")
	}
	return failures
}
