package substitution

import "fmt"

// CallableJoinConfig is a fixed constructor, not a context generator. Smaller
// limits are representable for directed refusal checks; entitlement is 4/4/3.
type CallableJoinConfig struct {
	Request     Request
	I, R, Depth int
}

// JoinDiagnostic is semantic data, independent of a compiled collection layout.
// Count=0 has canonical zero answer fields; Count=1 retains one original answer.
type JoinDiagnostic struct {
	ID, Count, AnswerID, AnswerValue int
	Classification                   string
}

type CallableJoinOutcome struct {
	Status, Reason string
	Present        bool
	Diagnostic     JoinDiagnostic
}

type CallableJoinEvent struct {
	JoinEvent
	Classification string
	Diagnostic     JoinDiagnostic
	Decision       string
}

// CallableJoinState stores physical inner depths 2/3. The historical inner
// reducer is used through an exact relative-depth/residual-budget bijection.
// O is represented only in this full product, not in K_join's definition.
type CallableJoinState struct {
	Inner                                                                          ConsumerReasonState
	InvocationCommitted, InvocationAuthorized, InvocationAdmitted, RequestConsumed bool
	ReportCommitted, ReportAuthorized, ReportAdmitted                              bool
	Published, Delivered, Consumed                                                 bool
	Report                                                                         JoinDiagnostic
	Outcome                                                                        CallableJoinOutcome
	// OuterStage: 0 Invoke, 1 await, 2 decide, 3 End, 4 terminal.
	OuterStage                 int
	OuterDecision, OuterStatus string
	RootPublished              bool
}

type CallableJoinEdge struct {
	Event       CallableJoinEvent
	Next        CallableJoinState
	Environment bool
}

func ValidateCallableJoinConfig(c CallableJoinConfig) error {
	if c.I < 1 || c.I > 4 || c.R < 1 || c.R > 4 || c.Depth < 2 || c.Depth > 3 {
		return fmt.Errorf("outside callable I/R1..4,depth2..3 domain")
	}
	return ValidateJoinConfig(callableRelativeConfig(c))
}

func CallableJoinConfigurations() []CallableJoinConfig {
	out := []CallableJoinConfig{}
	for _, c := range ConsumerReasonConfigurations() {
		out = append(out, CallableJoinConfig{Request: c.Request, I: 4, R: 4, Depth: 3})
	}
	return out
}

func CallableJoinEntitled(c CallableJoinConfig) bool {
	return ValidateCallableJoinConfig(c) == nil && c.I == 4 && c.R == 4 && c.Depth == 3
}

func callableRelativeConfig(c CallableJoinConfig) JoinConfig {
	return JoinConfig{Request: c.Request, I: c.I - 1, R: c.R - 1, Depth: c.Depth - 1}
}

func callableShift(s ConsumerReasonState, delta int) ConsumerReasonState {
	for _, i := range []int{entry, helper, sibling} {
		s.Component.Ops[i].Depth += delta
	}
	return s
}

func CallableJoinInitial(c CallableJoinConfig) CallableJoinState {
	return CallableJoinState{Inner: callableShift(ConsumerReasonInitial(callableRelativeConfig(c)), 1)}
}

func callableMeta(c CallableJoinConfig, kind string) Observation {
	return Observation{Kind: kind, Operation: "j", Parent: "root", Role: "join", Request: c.Request}
}

func callableReport(c CallableJoinConfig, s ConsumerReasonState) JoinDiagnostic {
	d := JoinDiagnostic{ID: c.Request.ID, Classification: s.Classification}
	if s.Classification == ReasonAnswer {
		e := s.Component.Ops[entry]
		d.Count, d.AnswerID, d.AnswerValue = 1, e.ID, e.Value
	}
	return d
}

// callableOuterPolicy uses only public outcome/input. No inner state or mode.
func callableOuterPolicy(request Request, o CallableJoinOutcome) (string, string) {
	d := o.Diagnostic
	validID := o.Present && d.ID == request.ID
	if validID && d.Classification == ReasonAnswer && o.Status == "succeeded" && o.Reason == "member_succeeded" {
		if d.Count == 1 && d.AnswerID == request.ID && d.AnswerValue == 1-request.Value {
			return ReasonAnswer, "succeeded"
		}
	}
	validFailure := validID && o.Status == "failed" && o.Reason == "member_failed" &&
		d.Count == 0 && d.AnswerID == 0 && d.AnswerValue == 0
	if validFailure && joinFiniteString(d.Classification, ReasonFailed, ReasonMissingReply, ReasonInvalid) {
		return d.Classification, "failed"
	}
	return "invalid_join_outcome", "failed"
}

// CallableJoinSuccessors has no search or qualification side effects. The old
// inner root End/publication is replaced by J's reply-before-End boundary.
func CallableJoinSuccessors(c CallableJoinConfig, v Variant, s CallableJoinState) []CallableJoinEdge {
	out := []CallableJoinEdge{}
	add := func(n CallableJoinState, e CallableJoinEvent, environment bool) {
		out = append(out, CallableJoinEdge{Event: e, Next: n, Environment: environment})
	}
	if s.RequestConsumed {
		for _, edge := range ConsumerReasonSuccessors(callableRelativeConfig(c), v, callableShift(s.Inner, -1)) {
			f := edge.Event.Fact
			if f.Operation == "root" && (f.Kind == "parent_terminal" || f.Kind == "root_publish") {
				continue
			}
			n := s
			n.Inner = callableShift(edge.Next, 1)
			e := CallableJoinEvent{JoinEvent: edge.Event.JoinEvent, Classification: edge.Event.Classification}
			if f.Operation == "root" {
				e.Fact = callableMeta(c, "join_classify")
				if f.Kind == "parent_decide" {
					e.Fact.Kind, e.Fact.Status = "join_decide", f.Status
				}
			} else if f.Parent == "root" {
				e.Fact.Parent = "j"
			}
			add(n, e, edge.Environment)
		}
	}
	if !s.InvocationCommitted {
		n := s
		n.InvocationCommitted, n.OuterStage = true, 1
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: callableMeta(c, "invoke_commit")}}, false)
	}
	if s.InvocationCommitted && !s.InvocationAuthorized {
		n := s
		n.InvocationAuthorized = true
		f := callableMeta(c, "invoke_authorize")
		f.Status, f.Reason = "allowed", "authorized"
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{
			Fact: f, SourceOpen: true, RevisionMatches: true, GenerationMatches: true,
		}}, true)
	}
	if s.InvocationAuthorized && !s.InvocationAdmitted {
		n := s
		n.InvocationAdmitted = true
		f := callableMeta(c, "admit")
		f.Status, f.I, f.R = "accepted", 1, 1
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}}, true)
	}
	if s.InvocationAdmitted && !s.RequestConsumed {
		n := s
		n.RequestConsumed = true
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: callableMeta(c, "request_consume")}}, false)
	}
	if s.Inner.ParentStage == 5 && !s.ReportCommitted {
		n := s
		n.ReportCommitted, n.Report = true, callableReport(c, s.Inner)
		f := callableMeta(c, "reply_commit")
		f.Status = "committed"
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}, Diagnostic: n.Report}, false)
	}
	if s.ReportCommitted && !s.ReportAuthorized {
		n := s
		n.ReportAuthorized = true
		f := callableMeta(c, "reply_authorize")
		f.Status, f.Reason = "allowed", "authorized"
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{
			Fact: f, SourceOpen: s.Inner.ParentStatus == "", ReplyBound: true,
			RevisionMatches: true, GenerationMatches: true,
		}}, true)
	}
	if s.ReportAuthorized && !s.ReportAdmitted {
		n := s
		n.ReportAdmitted = true
		f := callableMeta(c, "reply_admit")
		f.Status, f.Present = "accepted", true
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}, Diagnostic: s.Report}, true)
	}
	if s.Inner.ParentStage == 5 && s.ReportCommitted {
		n := s
		n.Inner.ParentStage, n.Inner.ParentStatus = 6, s.Inner.Decision
		f := callableMeta(c, "terminal_freeze")
		f.Status = n.Inner.ParentStatus
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}}, false)
	}
	if s.Inner.ParentStatus != "" && s.ReportAdmitted && !s.Published {
		n := s
		n.Published = true
		n.Outcome = CallableJoinOutcome{
			Status: s.Inner.ParentStatus, Reason: "member_" + s.Inner.ParentStatus,
			Present: true, Diagnostic: s.Report,
		}
		add(n, callableEnvelopeEvent(c, "publish", n.Outcome), true)
	}
	if s.Published && !s.Delivered {
		n := s
		n.Delivered = true
		add(n, callableEnvelopeEvent(c, "outcome_deliver", s.Outcome), true)
	}
	if s.Delivered && !s.Consumed && s.OuterStage == 1 {
		n := s
		n.Consumed, n.OuterStage = true, 2
		add(n, callableEnvelopeEvent(c, "outcome_consume", s.Outcome), false)
	}
	f := Observation{Operation: "root", Role: "caller", Request: c.Request}
	switch s.OuterStage {
	case 2:
		n := s
		n.OuterStage = 3
		n.OuterDecision, f.Status = callableOuterPolicy(c.Request, s.Outcome)
		f.Kind = "outer_decide"
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}, Decision: n.OuterDecision}, false)
	case 3:
		n := s
		n.OuterStage = 4
		_, n.OuterStatus = callableOuterPolicy(c.Request, s.Outcome)
		f.Kind, f.Status = "outer_terminal", n.OuterStatus
		add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}}, false)
	case 4:
		if !s.RootPublished {
			n := s
			n.RootPublished = true
			f.Kind, f.Status, f.Reason = "root_publish", s.OuterStatus, "member_"+s.OuterStatus
			add(n, CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}}, true)
		}
	}
	return out
}

func callableEnvelopeEvent(c CallableJoinConfig, kind string, o CallableJoinOutcome) CallableJoinEvent {
	f := callableMeta(c, kind)
	f.Status, f.Reason, f.Present = o.Status, o.Reason, o.Present
	return CallableJoinEvent{JoinEvent: JoinEvent{Fact: f}, Diagnostic: o.Diagnostic}
}

// Ten once-only J milestones, four O stages and root publication augment the
// old intrinsic rank. Payload values/history length are not rank components.
const CallableJoinRankCap = ConsumerReasonRankCap + 100*(10+4+1)

func CallableJoinRank(s CallableJoinState) int {
	rank := ConsumerReasonRank(s.Inner) + 100*s.OuterStage
	for _, flag := range []bool{
		s.InvocationCommitted, s.InvocationAuthorized, s.InvocationAdmitted, s.RequestConsumed,
		s.ReportCommitted, s.ReportAuthorized, s.ReportAdmitted, s.Published, s.Delivered, s.Consumed,
		s.RootPublished,
	} {
		if flag {
			rank += 100
		}
	}
	return rank
}

func CheckCallableJoinState(c CallableJoinConfig, s CallableJoinState) error {
	if err := ValidateCallableJoinConfig(c); err != nil {
		return err
	}
	if err := CheckConsumerReasonState(callableRelativeConfig(c), callableShift(s.Inner, -1)); err != nil {
		return fmt.Errorf("inner: %w", err)
	}
	if s.Inner.RootPublished || s.OuterStage < 0 || s.OuterStage > 4 {
		return fmt.Errorf("unused inner root or outer control changed")
	}
	if !s.RequestConsumed && s.Inner != CallableJoinInitial(c).Inner {
		return fmt.Errorf("inner work precedes original J request")
	}
	for _, pair := range [][2]bool{
		{s.InvocationAuthorized, s.InvocationCommitted}, {s.InvocationAdmitted, s.InvocationAuthorized},
		{s.RequestConsumed, s.InvocationAdmitted}, {s.ReportCommitted, s.RequestConsumed},
		{s.ReportAuthorized, s.ReportCommitted}, {s.ReportAdmitted, s.ReportAuthorized},
		{s.Published, s.ReportAdmitted}, {s.Delivered, s.Published}, {s.Consumed, s.Delivered},
	} {
		if pair[0] && !pair[1] {
			return fmt.Errorf("J protocol predecessor missing")
		}
	}
	if s.ReportCommitted {
		if s.Inner.ParentStage < 5 || s.Report != callableReport(c, s.Inner) {
			return fmt.Errorf("report differs from original consumed outcomes")
		}
	} else if s.Report != (JoinDiagnostic{}) {
		return fmt.Errorf("uncommitted report payload")
	}
	if s.Inner.ParentStage == 6 && !s.ReportCommitted {
		return fmt.Errorf("J End precedes diagnostic Send")
	}
	if s.Published {
		want := CallableJoinOutcome{Status: s.Inner.ParentStatus, Reason: "member_" + s.Inner.ParentStatus,
			Present: true, Diagnostic: s.Report}
		if s.Inner.ParentStage != 6 || s.Outcome != want {
			return fmt.Errorf("J publication differs from End and admitted diagnostic")
		}
	} else if s.Outcome != (CallableJoinOutcome{}) {
		return fmt.Errorf("unpublished J outcome")
	}
	if s.InvocationCommitted != (s.OuterStage >= 1) || s.Consumed != (s.OuterStage >= 2) {
		return fmt.Errorf("outer original-handle control mismatch")
	}
	decision, status := callableOuterPolicy(c.Request, s.Outcome)
	if s.OuterStage < 3 && s.OuterDecision != "" || s.OuterStage >= 3 && s.OuterDecision != decision {
		return fmt.Errorf("outer decision differs from public evidence")
	}
	if s.OuterStage < 4 && s.OuterStatus != "" || s.OuterStage == 4 && s.OuterStatus != status ||
		s.RootPublished && s.OuterStage != 4 {
		return fmt.Errorf("outer terminal/publication mismatch")
	}
	if CallableJoinRank(s) > CallableJoinRankCap {
		return fmt.Errorf("callable rank exceeds finite cap")
	}
	return nil
}
