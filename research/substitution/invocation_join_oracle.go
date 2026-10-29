package substitution

import "fmt"

type joinPrefixOperation struct {
	Committed, Authorized, Allowed, Requested                                  bool
	Admitted, Rejected                                                         bool
	Rejection                                                                  string
	ReplyCommitted, ReplyAuthorized, ReplyAllowed, ReplyDecided, ReplyAccepted bool
	ReplyID, ReplyValue                                                        int
	Terminal                                                                   string
	Published, Delivered, Consumed                                             bool
	Outcome                                                                    Observation
}

// Included in exploration identity: a representative path is not assumed to
// determine every monitor state at a merge. All fields have finite domains.
type joinPrefixState struct {
	Ops                    [slots]joinPrefixOperation
	I, R                   int
	Decision, ParentStatus string
	RootPublished          bool
}

func joinIndex(operation string) int {
	switch operation {
	case "e":
		return entry
	case "h":
		return helper
	case "s":
		return sibling
	default:
		return -1
	}
}

func joinExpectedMeta(c JoinConfig, i int) (string, string, Request) {
	switch i {
	case entry:
		return "root", "candidate", c.Request
	case helper:
		return "e", "helper", Request{ID: c.Request.ID, Value: c.Request.Value, Mode: "answer"}
	default:
		return "root", "s", Request{ID: c.Request.ID, Value: c.Request.Value}
	}
}

func joinSameEnvelope(a, b Observation) bool {
	return a.Status == b.Status && a.Reason == b.Reason && a.Present == b.Present && a.ID == b.ID && a.Value == b.Value
}

func joinPrefixCorrect(c JoinConfig, s joinPrefixState) bool {
	e, sibling := s.Ops[entry], s.Ops[sibling]
	return e.Consumed && sibling.Consumed && e.Outcome.Status == "succeeded" && e.Outcome.Present &&
		e.Outcome.ID == c.Request.ID && e.Outcome.Value == 1-c.Request.Value &&
		sibling.Outcome.Status == "succeeded" && sibling.Outcome.Present &&
		sibling.Outcome.ID == c.Request.ID && sibling.Outcome.Value == 0
}

func advanceJoinPrefix(c JoinConfig, s joinPrefixState, event JoinEvent, index int) (joinPrefixState, []Violation) {
	f := event.Fact
	bad := func(code, detail string) (joinPrefixState, []Violation) {
		return s, []Violation{{Index: index, Code: code, Detail: detail}}
	}
	n := s
	if f.Kind != "admit" && (f.I != 0 || f.R != 0 || f.E != 0) {
		return bad("unexpected_debit", f.Kind)
	}
	if f.Operation == "root" {
		if f.Parent != "" || f.Role != "parent" || f.Request != c.Request || f.Present || f.ID != 0 || f.Value != 0 {
			return bad("parent_identity", "root fact/request/presence mismatch")
		}
		switch f.Kind {
		case "parent_decide":
			if s.Decision != "" || s.ParentStatus != "" || !s.Ops[entry].Consumed || !s.Ops[sibling].Consumed {
				return bad("required_joins", "decision requires both original selected outcomes")
			}
			want := "failed"
			if joinPrefixCorrect(c, s) {
				want = "succeeded"
			}
			if f.Status != want {
				return bad("parent_decision", "decision differs from guarded actual outcomes")
			}
			n.Decision = f.Status
		case "parent_terminal":
			if s.Decision == "" || s.ParentStatus != "" || f.Status != s.Decision {
				return bad("parent_terminal", "End lacks original matching decision")
			}
			n.ParentStatus = f.Status
		case "root_publish":
			if s.ParentStatus == "" || s.RootPublished || f.Status != s.ParentStatus || f.Reason != "member_"+s.ParentStatus {
				return bad("root_publication", "publication lacks original terminal outcome")
			}
			n.RootPublished = true
		default:
			return bad("parent_event", f.Kind)
		}
		return n, nil
	}

	i := joinIndex(f.Operation)
	if i < 0 {
		return bad("operation_domain", f.Operation)
	}
	parent, role, request := joinExpectedMeta(c, i)
	if f.Parent != parent || f.Role != role || f.Request != request {
		return bad("operation_binding", "original role/parent/request envelope mismatch: "+f.Operation)
	}
	o := s.Ops[i]
	if f.Kind != "invoke_commit" && !o.Committed {
		return bad("original_commit", f.Operation)
	}
	openSource := s.ParentStatus == ""
	if i == helper {
		openSource = s.Ops[entry].Terminal == ""
	}
	openMember := o.Terminal == ""
	switch f.Kind {
	case "invoke_commit":
		if o.Committed || !openSource {
			return bad("invocation_commit", "duplicate or closed source")
		}
		if i == sibling && !s.Ops[entry].Committed {
			return bad("commit_order", "S precedes candidate")
		}
		if i == helper && (!s.Ops[entry].Admitted || !s.Ops[entry].Requested) {
			return bad("helper_parent", "helper precedes original candidate request")
		}
		n.Ops[i].Committed = true
	case "invoke_authorize", "reply_authorize":
		reply := f.Kind == "reply_authorize"
		if !event.RevisionMatches || !event.GenerationMatches || event.SourceRetired {
			return bad("environment_profile", "source generation/revision/retirement outside stable profile")
		}
		if reply {
			if !o.ReplyCommitted || o.ReplyAuthorized {
				return bad("reply_authorization", "missing commit or repeated first disposition")
			}
			if !event.ReplyBound {
				return bad("reply_binding", "not an original invocation-bound reply")
			}
			if event.SourceOpen != openMember {
				return bad("authorization_snapshot", "reply source snapshot differs from prior terminal fact")
			}
			if f.Status != "allowed" || f.Reason != "authorized" {
				return bad("reply_authorization", "stable bound reply denied, including terminal source")
			}
			n.Ops[i].ReplyAuthorized, n.Ops[i].ReplyAllowed = true, true
		} else {
			if o.Authorized {
				return bad("invocation_authorization", "repeated first disposition")
			}
			if event.SourceOpen != openSource {
				return bad("authorization_snapshot", "invocation source snapshot differs from prior terminal fact")
			}
			status, reason := "allowed", "authorized"
			if !openSource {
				status, reason = "refused", "source_closed"
			}
			if f.Status != status || f.Reason != reason {
				return bad("invocation_authorization", "source disposition mismatch")
			}
			n.Ops[i].Authorized, n.Ops[i].Allowed = true, openSource
		}
	case "admit":
		if !o.Authorized || o.Admitted || o.Rejected {
			return bad("admission_order", "admission lacks unique first authorization")
		}
		reason := ""
		depth := 1
		if i == helper {
			depth = 2
		}
		switch {
		case !o.Allowed:
			reason = "source_denied:source_closed"
		case !openSource:
			reason = "source_closed"
		case s.ParentStatus != "":
			reason = "scope_closed"
		case depth > c.Depth:
			reason = "depth_limit"
		case s.I >= c.I:
			reason = "invocation_limit"
		case s.R >= c.R:
			reason = "created_run_limit"
		}
		if reason == "" {
			if f.Status != "accepted" || f.Reason != "" || f.I != 1 || f.R != 1 || f.E != 0 {
				return bad("admission_debit", "accepted singleton must debit (1,1,0)")
			}
			n.Ops[i].Admitted = true
			n.I++
			n.R++
		} else {
			if f.Status != "rejected" || f.Reason != reason || f.I != 0 || f.R != 0 || f.E != 0 {
				return bad("admission_refusal", "wrong refusal or refused debit")
			}
			n.Ops[i].Rejected, n.Ops[i].Rejection = true, reason
		}
	case "request_consume":
		if !o.Admitted || o.Requested || !openMember {
			return bad("request_consume", "missing admission, duplicate or terminal member")
		}
		n.Ops[i].Requested = true
	case "choose_local", "choose_helper", "private_step", "pending_receive":
		// Ghost/private control is checked by artifact correspondence and local
		// refinement. It cannot create a protocol fact in this context monitor.
		if !o.Admitted || !o.Requested || !openMember {
			return bad("local_control", "control outside an open requested member")
		}
	case "reply_commit":
		if !o.Admitted || !o.Requested || !openMember || o.ReplyCommitted || f.Status != "committed" || f.Present {
			return bad("reply_commit", "reply lacks original open request or repeats")
		}
		if f.ID < 1 || f.ID > 2 || f.Value < 0 || f.Value > 1 {
			return bad("response_domain", "committed response outside selected finite domain")
		}
		n.Ops[i].ReplyCommitted, n.Ops[i].ReplyID, n.Ops[i].ReplyValue = true, f.ID, f.Value
	case "terminal_freeze":
		if !o.Admitted || !openMember || f.Status != "succeeded" && f.Status != "failed" {
			return bad("member_terminal", "invalid or repeated terminal freeze")
		}
		n.Ops[i].Terminal = f.Status
	case "reply_admit":
		if !o.ReplyAuthorized || !o.ReplyAllowed || o.ReplyDecided || f.Status != "accepted" || f.Reason != "" || !f.Present || f.ID != o.ReplyID || f.Value != o.ReplyValue {
			return bad("reply_admission", "reply does not preserve original authorized payload")
		}
		// id1/2,value0/1 are valid K1 response-domain values. Their relationship
		// to the request is deliberately checked by component conformance/G.
		if f.ID < 1 || f.ID > 2 || f.Value < 0 || f.Value > 1 {
			return bad("response_domain", "response outside selected finite domain")
		}
		n.Ops[i].ReplyDecided, n.Ops[i].ReplyAccepted = true, true
	case "publish":
		if o.Published || !o.Rejected && o.Terminal == "" || o.ReplyCommitted && !o.ReplyDecided {
			return bad("publication_order", "publication precedes admission/terminal/reply admission")
		}
		want := Observation{Status: "failed", Reason: "member_failed"}
		if o.Rejected {
			want.Status, want.Reason = "rejected", o.Rejection
		} else {
			if o.Terminal == "succeeded" {
				if i == entry && s.Ops[helper].Committed && (!s.Ops[helper].Admitted && !s.Ops[helper].Rejected || s.Ops[helper].Admitted && !s.Ops[helper].Published) {
					return bad("descendant_publication", "success precedes original descendant resolution")
				}
				want.Status, want.Reason = "succeeded", "member_succeeded"
				if !o.ReplyAccepted {
					want.Status, want.Reason = "failed", "missing_reply"
				}
			}
			if o.ReplyAccepted {
				want.Present, want.ID, want.Value = true, o.ReplyID, o.ReplyValue
			}
		}
		if !joinSameEnvelope(f, want) {
			return bad("publication_envelope", "outcome differs from original terminal/reply/refusal")
		}
		n.Ops[i].Published, n.Ops[i].Outcome = true, f
	case "outcome_deliver":
		if !o.Published || o.Delivered || !joinSameEnvelope(f, o.Outcome) {
			return bad("outcome_delivery", "delivery differs from unique original publication")
		}
		n.Ops[i].Delivered = true
	case "outcome_consume":
		if !o.Delivered || o.Consumed || !openSource || !joinSameEnvelope(f, o.Outcome) {
			return bad("outcome_consumption", "selection lacks original delivered unconsumed envelope")
		}
		if i == entry && !s.Ops[sibling].Committed {
			return bad("both_committed", "first await precedes S commit")
		}
		if i == sibling && !s.Ops[entry].Consumed {
			return bad("consume_order", "S selected before candidate")
		}
		n.Ops[i].Consumed = true
	default:
		return bad("unknown_event", f.Kind)
	}
	return n, nil
}

func checkJoinPrefix(c JoinConfig, events []JoinEvent) (joinPrefixState, []Violation) {
	state := joinPrefixState{}
	if err := ValidateJoinConfig(c); err != nil {
		return state, []Violation{{Index: -1, Code: "config", Detail: err.Error()}}
	}
	violations := []Violation{}
	for index, event := range events {
		next, found := advanceJoinPrefix(c, state, event, index)
		violations = append(violations, found...)
		state = next
	}
	return state, violations
}

// CheckJoinPrefix accepts honest incomplete/failed context histories; it does
// not assert K1 application conformance, G, or authentication of raw records.
func CheckJoinPrefix(c JoinConfig, events []JoinEvent) []Violation {
	_, violations := checkJoinPrefix(c, events)
	return violations
}

// CheckJoinComponent preserves the old K1 monitor as an independent result,
// and checks the helper's fixed answer/Success behavior and the candidate's
// declared mode/required-helper discipline at reply and terminal prefixes.
// Passing prefixes alone do not establish local progress or the local relation.
func CheckJoinComponent(c JoinConfig, events []JoinEvent) []Violation {
	facts := []Observation{}
	indices := []int{}
	for index, event := range events {
		if event.Fact.Role == "candidate" || event.Fact.Role == "helper" {
			facts = append(facts, event.Fact)
			indices = append(indices, index)
		}
	}
	violations := CheckTrace(c.Request, facts)
	for index := range violations {
		violations[index].Index = indices[violations[index].Index]
	}
	helperCommitted, helperConsumed, helperOK, replied := false, false, false, false
	// These flags are derived from keyed Prefix.Ops[helper].Admitted and
	// ReplyCommitted/ReplyID/ReplyValue, so they add no exploration memory.
	helperAdmitted, helperReplyCorrect := false, false
	for index, event := range events {
		f := event.Fact
		if f.Role == "helper" {
			switch f.Kind {
			case "invoke_commit":
				helperCommitted = true
			case "admit":
				helperAdmitted = f.Status == "accepted"
			case "reply_commit":
				helperReplyCorrect = f.ID == c.Request.ID && f.Value == 1-c.Request.Value
				if !helperReplyCorrect {
					violations = append(violations, Violation{Index: index, Code: "helper_reply", Detail: "helper committed a reply that differs from the K1 answer"})
				}
			case "terminal_freeze":
				if helperAdmitted && (f.Status != "succeeded" || !helperReplyCorrect) {
					violations = append(violations, Violation{Index: index, Code: "helper_terminal", Detail: "admitted helper must commit its correct reply before Success"})
				}
			case "outcome_consume":
				helperConsumed, helperOK = true, f.Status == "succeeded" && f.Present
			}
		}
		if f.Role != "candidate" {
			continue
		}
		if f.Kind == "reply_commit" {
			replied = true
			if c.Request.Mode != "answer" {
				violations = append(violations, Violation{Index: index, Code: "mode_reply", Detail: c.Request.Mode})
			}
			if helperCommitted && !helperConsumed {
				violations = append(violations, Violation{Index: index, Code: "helper_join", Detail: "candidate replied before original helper consumption"})
			}
			if helperCommitted && helperConsumed && !helperOK {
				violations = append(violations, Violation{Index: index, Code: "helper_result", Detail: "candidate replied after an unsuccessful original helper outcome"})
			}
		}
		if f.Kind == "terminal_freeze" {
			want := "succeeded"
			if c.Request.Mode == "fail" || helperCommitted && helperConsumed && !helperOK {
				want = "failed"
			}
			if helperCommitted && !helperConsumed {
				violations = append(violations, Violation{Index: index, Code: "helper_join", Detail: "candidate terminated before original helper consumption"})
			}
			if c.Request.Mode == "pending" && !(helperCommitted && helperConsumed && !helperOK) || f.Status != want || c.Request.Mode == "answer" && want == "succeeded" && !replied {
				violations = append(violations, Violation{Index: index, Code: "mode_terminal", Detail: "terminal behavior differs from K1 mode/helper outcome"})
			}
		}
	}
	return violations
}

// CheckJoinGuarantee requires completed evidence. It does not silently include
// component conformance in G: callers report CheckJoinComponent independently.
func CheckJoinGuarantee(c JoinConfig, events []JoinEvent) []Violation {
	s, violations := checkJoinPrefix(c, events)
	if !JoinEntitled(c) {
		violations = append(violations, Violation{Index: len(events), Code: "positive_profile", Detail: "G requires answer,I=R3,depth2"})
	}
	if !joinPrefixCorrect(c, s) {
		violations = append(violations, Violation{Index: len(events), Code: "required_results", Detail: "both original correct successful present results must be consumed"})
	}
	if s.ParentStatus != "succeeded" || !s.RootPublished {
		violations = append(violations, Violation{Index: len(events), Code: "completion", Detail: "parent success and matching root publication are required"})
	}
	if s.I > c.I || s.R > c.R {
		violations = append(violations, Violation{Index: len(events), Code: "resource", Detail: fmt.Sprintf("spending (%d,%d,0) exceeds limits", s.I, s.R)})
	}
	return violations
}
