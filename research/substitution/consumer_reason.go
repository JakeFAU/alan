package substitution

import "fmt"

const (
	ReasonAnswer       = "answer"
	ReasonFailed       = "candidate_failed"
	ReasonMissingReply = "candidate_missing_reply"
	ReasonInvalid      = "invalid_outcome"
)

// ConsumerReasonState adds one retained classification to the unchanged C_se
// protocol state. Empty is neutral, never a completed classification. Stage4
// first classifies, then decides; the old stages and public types do not change.
type ConsumerReasonState struct {
	JoinState
	Classification string
}

// ConsumerReasonEvent keeps the classification separate from original envelope
// reasons. Only parent_classify carries this field. Raw authentication and the
// classification-write/decision instruction linkage are external obligations.
type ConsumerReasonEvent struct {
	JoinEvent
	Classification string
}

type ConsumerReasonEdge struct {
	Event       ConsumerReasonEvent
	Next        ConsumerReasonState
	Environment bool
}

// ConsumerReasonConfigurations declares the 16-input G_outcome domain before
// enumeration. The S template remains the original answer template throughout.
func ConsumerReasonConfigurations() []JoinConfig {
	configs := []JoinConfig{}
	for _, c := range JoinConfigurations() {
		for _, mode := range []string{"answer", "fail", "absent", "pending"} {
			c.Request.Mode = mode
			configs = append(configs, c)
		}
	}
	return configs
}

// ConsumerReasonEntitled deliberately includes all four modes. The inherited
// answer-only predicates must not disable non-answer progress or G_outcome.
func ConsumerReasonEntitled(c JoinConfig) bool {
	return ValidateJoinConfig(c) == nil && c.I == 3 && c.R == 3 && c.Depth == 2
}

func ConsumerReasonInitial(c JoinConfig) ConsumerReasonState {
	return ConsumerReasonState{JoinState: ConsumerPolicyInitial(c)}
}

// reasonClassification is the model's guarded predicate on actual observations.
// It does not use requested mode. The independent monitor restates this policy.
func reasonClassification(c JoinConfig, e, s Operation) string {
	if !e.Consumed || !s.Consumed {
		return ""
	}
	if s.Status != "succeeded" || s.Reason != "member_succeeded" || !s.Present {
		return ReasonInvalid
	}
	if s.ID != c.Request.ID || s.Value != 0 {
		return ReasonInvalid
	}
	if e.Status == "succeeded" && e.Reason == "member_succeeded" && e.Present {
		if e.ID == c.Request.ID && e.Value == 1-c.Request.Value {
			return ReasonAnswer
		}
		return ReasonInvalid
	}
	if e.Status == "failed" && !e.Present {
		switch e.Reason {
		case "member_failed":
			return ReasonFailed
		case "missing_reply":
			return ReasonMissingReply
		}
	}
	return ReasonInvalid
}

// ConsumerReasonSuccessors reuses only the original non-root C_se transitions.
// The parent's actual classification, decision, End and root publication are
// explicit below. K1 still has both independent local/helper choices in all modes.
func ConsumerReasonSuccessors(c JoinConfig, v Variant, s ConsumerReasonState) []ConsumerReasonEdge {
	out := []ConsumerReasonEdge{}
	for _, edge := range ConsumerPolicySuccessors(c, v, s.JoinState) {
		if edge.Event.Fact.Operation == "root" {
			continue
		}
		out = append(out, ConsumerReasonEdge{
			Event:       ConsumerReasonEvent{JoinEvent: edge.Event},
			Next:        ConsumerReasonState{JoinState: edge.Next, Classification: s.Classification},
			Environment: edge.Environment,
		})
	}
	n := s
	f := Observation{Operation: "root", Role: "parent", Request: c.Request}
	event := ConsumerReasonEvent{}
	environment := false
	switch {
	case s.ParentStage == 4 && s.Classification == "":
		n.Classification = reasonClassification(c, s.Component.Ops[entry], s.Component.Ops[sibling])
		if n.Classification == "" {
			return out
		}
		f.Kind, event.Classification = "parent_classify", n.Classification
	case s.ParentStage == 4 && s.Classification != "":
		n.ParentStage, n.Decision = 5, "failed"
		if s.Classification == ReasonAnswer {
			n.Decision = "succeeded"
		}
		f.Kind, f.Status = "parent_decide", n.Decision
	case s.ParentStage == 5:
		n.ParentStage, n.ParentStatus = 6, s.Decision
		f.Kind, f.Status = "parent_terminal", n.ParentStatus
	case s.ParentStage == 6 && !s.RootPublished:
		n.RootPublished, environment = true, true
		f.Kind, f.Status, f.Reason = "root_publish", s.ParentStatus, "member_"+s.ParentStatus
	default:
		return out
	}
	event.Fact = f
	return append(out, ConsumerReasonEdge{Event: event, Next: n, Environment: environment})
}

// ConsumerReasonRank adds one monotone classification milestone. Tag ordinals,
// payloads, history length and the requested mode do not contribute to rank.
func ConsumerReasonRank(s ConsumerReasonState) int {
	rank := JoinRank(s.JoinState)
	if s.Classification != "" {
		rank += 100
	}
	return rank
}

const ConsumerReasonRankCap = JoinRankCap + 100

func CheckConsumerReasonState(c JoinConfig, s ConsumerReasonState) error {
	if err := CheckConsumerPolicyState(c, s.JoinState); err != nil {
		return err
	}
	if !joinFiniteString(s.Classification, "", ReasonAnswer, ReasonFailed, ReasonMissingReply, ReasonInvalid) {
		return fmt.Errorf("classification outside finite domain")
	}
	if s.Classification != "" {
		if s.ParentStage < 4 || s.Classification != reasonClassification(c, s.Component.Ops[entry], s.Component.Ops[sibling]) {
			return fmt.Errorf("classification differs from both original consumed outcomes")
		}
	}
	if s.ParentStage >= 5 {
		want := "failed"
		if s.Classification == ReasonAnswer {
			want = "succeeded"
		}
		if s.Classification == "" || s.Decision != want {
			return fmt.Errorf("decision lacks matching completed classification")
		}
	}
	if ConsumerReasonEntitled(c) {
		demand := 0
		for _, o := range s.Component.Ops {
			if o.Ticket == possible || o.Ticket == committed {
				demand++
			}
		}
		if s.Component.I+demand > c.I || s.Component.R+demand > c.R {
			return fmt.Errorf("all-mode spent plus remaining demand exceeds entitlement")
		}
	}
	if ConsumerReasonRank(s) > ConsumerReasonRankCap {
		return fmt.Errorf("classification rank exceeds declared cap")
	}
	return nil
}
