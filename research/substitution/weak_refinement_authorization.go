package substitution

import (
	"encoding/json"
	"fmt"
)

// Finite clause records mirror the original immutable intent/binding and saved
// first result. They are not candidate-readable state and do not extend R.
// Raw authority snapshots are declared clause inputs, not recovered run data.
type weakAuthority struct {
	Subject                               string
	Generation, Epoch, Sequence, Revision int64
}
type weakAuthIntent struct {
	ID, Subject, Kind, Target, Obligation, Payload   string
	SourceGeneration, Sequence, Generation, Revision int64
	Ordinal                                          int
}
type weakReplyBinding struct {
	Protocol, Member, EntryMember, CallerSubject, Obligation string
	CallerGeneration                                         int64
}
type weakAuthorization struct {
	ID, IntentID     string
	Allowed          bool
	Reason           string
	Source           weakAuthority
	ClosureOperation string
}
type weakAuthContext struct {
	Intent            weakAuthIntent
	Committed         *weakAuthIntent
	Source            weakAuthority
	SubjectGeneration int64
	Status            string
	Retired           bool
	Closure           string
	Binding           weakReplyBinding
	SchemaValid       bool
}
type weakSavedAuthorization struct {
	Intent weakAuthIntent
	Value  weakAuthorization
}

func checkWeakAuthorization(c weakAuthContext, claimed weakAuthorization, saved *weakSavedAuthorization) []weakIssue {
	if c.Committed == nil || *c.Committed != c.Intent {
		return []weakIssue{{Clause: "committed_descriptor"}}
	}
	if c.Source.Subject != c.Intent.Subject || c.Source.Generation != c.Intent.SourceGeneration {
		return []weakIssue{{Clause: "source_binding"}}
	}
	if saved != nil {
		if saved.Intent != c.Intent || claimed != saved.Value {
			return []weakIssue{{Clause: "repeated_authorization"}}
		}
		return []weakIssue{}
	}
	// Same switch order as original messages.go: revision/generation precede
	// the bound-reply exception. Source closure alone cannot deny such a reply.
	id, _ := json.Marshal([]string{"authorization", c.Intent.ID})
	expected := weakAuthorization{ID: string(id), IntentID: c.Intent.ID, Source: c.Source, ClosureOperation: c.Closure}
	bound := weakIsBoundReply(c)
	switch {
	case c.Source.Revision != c.Intent.Revision:
		expected.Reason = "source_revision_changed"
	case c.SubjectGeneration != c.Intent.SourceGeneration:
		expected.Reason = "source_generation_changed"
	case c.Retired && !bound:
		expected.Reason = "source_retired"
	case c.Status != "open" && !bound:
		expected.Reason = "source_closed"
	default:
		expected.Allowed = true
		expected.Reason = "authorized"
	}
	if claimed != expected {
		return []weakIssue{{Clause: "first_authorization"}}
	}
	return []weakIssue{}
}

type weakReplyAdmission struct {
	Intent        weakAuthIntent
	Authorization weakAuthorization
	Accepted      bool
	Reason        string
}

func checkWeakReplyAdmission(c weakAuthContext, saved weakSavedAuthorization, claimed weakReplyAdmission) []weakIssue {
	if issues := checkWeakAuthorization(c, saved.Value, &saved); len(issues) != 0 {
		return issues
	}
	if !weakIsBoundReply(c) {
		return []weakIssue{{Clause: "reply_binding"}}
	}
	expected := weakReplyAdmission{Intent: c.Intent, Authorization: saved.Value, Accepted: true, Reason: "recorded_reply"}
	if !saved.Value.Allowed {
		expected.Accepted = false
		expected.Reason = "source_denied:" + saved.Value.Reason
	} else if !c.SchemaValid {
		expected.Accepted = false
		expected.Reason = "schema_invalid"
	}
	if claimed != expected {
		return []weakIssue{{Clause: "reply_admission"}}
	}
	return []weakIssue{}
}
func weakInvocationAdmission(c weakAuthContext, auth weakAuthorization, scopeClosed bool) string {
	switch {
	case !auth.Allowed:
		return "source_denied:" + auth.Reason
	case c.Status != "open" || c.Source.Revision != c.Intent.Revision:
		return "source_closed"
	case scopeClosed:
		return "scope_closed"
	default:
		return "accepted"
	}
}

func weakIsBoundReply(c weakAuthContext) bool {
	b, i := c.Binding, c.Intent
	return b.Protocol == "alan/composition/v1" && i.Kind == "message" && b.Member == b.EntryMember &&
		i.Target == b.CallerSubject && i.Generation == b.CallerGeneration && i.Obligation == b.Obligation
}

type weakAuthorizationLookup struct {
	Context weakAuthContext
	Claimed weakAuthorization
}

func checkWeakAuthorizationHistory(history []weakAuthorizationLookup) []weakIssue {
	saved := map[string]weakSavedAuthorization{}
	for index, lookup := range history {
		var prior *weakSavedAuthorization
		if old, ok := saved[lookup.Context.Intent.ID]; ok {
			prior = &old
		}
		if issues := checkWeakAuthorization(lookup.Context, lookup.Claimed, prior); len(issues) != 0 {
			for i := range issues {
				issues[i].Detail = fmt.Sprintf("lookup %d", index)
			}
			return issues
		}
		if prior == nil {
			saved[lookup.Context.Intent.ID] = weakSavedAuthorization{Intent: lookup.Context.Intent, Value: lookup.Claimed}
		}
	}
	return []weakIssue{}
}

// Preparation is only the captured-reply admission clause, assuming the stated
// member/descendant obligations are already resolved. It is not publication.
func weakCapturedReplyPending(frozen, captured, admitted bool) string {
	if !frozen {
		return "members"
	}
	if captured && !admitted {
		return "reply_admission"
	}
	return ""
}
