package executor

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"slices"
	"strings"
)

func (x *compositionPrefix) closed(run RunID) bool {
	scope := x.scopes[run]
	root := x.roots[scope.RootID]
	if x.states[root.Run].Status != "open" {
		return true
	}
	closed := func(id string) bool {
		f, has := x.freezes[id]
		_, retired := x.closures[id]
		_, published := x.outcomes[id]
		return has && f.Status != "succeeded" || retired || published
	}
	if closed("root:" + root.ID) {
		return true
	}
	for b, ok := x.bindings[run]; ok; b, ok = x.bindings[b.Caller] {
		if closed(b.InvocationID) {
			return true
		}
	}
	return false
}
func (x *compositionPrefix) capture(scope compositionScope, run RunID, trigger string, order int64, status string) compositionFreeze {
	f := compositionFreeze{InvocationID: scope.ID, Trigger: trigger, Run: run, Sequence: x.sequences[run], DecisionOrder: order, Status: status, ReplyIntents: []string{}, InvocationIntents: []string{}, ExternalIntents: []string{}}
	for id, i := range x.intents {
		if !slices.Contains(scope.Members, i.Run) {
			continue
		}
		switch i.Kind {
		case "invocation":
			f.InvocationIntents = append(f.InvocationIntents, id)
		case "adapter", "external":
			f.ExternalIntents = append(f.ExternalIntents, id)
		case "message":
			if x.public[id] {
				f.ReplyIntents = append(f.ReplyIntents, id)
			}
		}
	}
	slices.Sort(f.ReplyIntents)
	slices.Sort(f.InvocationIntents)
	slices.Sort(f.ExternalIntents)
	return f
}
func (x *compositionPrefix) freeze(run RunID, trigger string, order int64) (compositionFreeze, bool) {
	scope := x.scopes[run]
	if _, ok := x.freezes[scope.ID]; ok {
		return compositionFreeze{}, false
	}
	if _, ok := x.closures[scope.ID]; ok {
		return compositionFreeze{}, false
	}
	status := x.states[run].Status
	if status == "open" {
		return compositionFreeze{}, false
	}
	if status == "succeeded" {
		for _, m := range scope.Members {
			if x.states[m].Status != "succeeded" {
				return compositionFreeze{}, false
			}
		}
	}
	return x.capture(scope, run, trigger, order, status), true
}
func (x *compositionPrefix) validateAdmission(ctx context.Context, q queryer, l Limits, a compositionAdmission) error {
	id := a.Invocation.OperationID
	i, ok := x.intents[id]
	if !ok || i.Kind != "invocation" || !x.authorized[id] {
		return corrupt("admission precedes authorized Invoke")
	}
	source, err := loadIntent(ctx, q, id)
	if err != nil {
		return err
	}
	scope := x.scopes[i.Run]
	root := x.roots[scope.RootID]
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return err
	}
	expected := compositionInvocation{ID: compositionInvocationID(namespace, id), OperationID: id, TemplateID: i.Target, RootID: root.ID, Caller: i.Run, SourceRevision: source.revision, SourceSequence: i.Sequence, Depth: 1, Inputs: i.Payload}
	parent, hasParent := x.bindings[i.Run]
	if hasParent {
		expected.ParentInvocationID = parent.InvocationID
		expected.Depth = parent.Depth + 1
	}
	if !reflect.DeepEqual(a.Invocation, expected) {
		return corrupt("invocation immutable descriptor")
	}
	auth, err := loadAuthorization(ctx, q, id)
	if err != nil {
		return err
	}
	state, err := loadRun(ctx, q, i.Run)
	if err != nil {
		return err
	}
	p, err := loadProgram(ctx, q, state.ProgramID, l)
	if err != nil {
		return err
	}
	var declaration InvocationDeclaration
	declared := false
	if p.Author != nil {
		declaration, declared = p.Author.Invocations[i.Obligation]
	}
	t, templateErr := loadCompositionTemplate(ctx, q, i.Target)
	if templateErr != nil && !errors.Is(templateErr, sql.ErrNoRows) {
		return templateErr
	}
	reason := "accepted"
	switch {
	case !auth.Allowed:
		reason = "source_denied:" + auth.Reason
	case x.states[i.Run].Status != "open" || state.Revision != source.revision:
		reason = "source_closed"
	case x.closed(i.Run):
		reason = "scope_closed"
	case !declared || declaration.TemplateID != i.Target || i.Generation != 1:
		reason = "declaration_denied"
	case templateErr != nil:
		reason = "template_missing"
	case !slices.Contains(root.Templates, i.Target):
		reason = "template_denied"
	case expected.Depth > root.Limits.MaxDepth:
		reason = "depth_limit"
	case declaration.Input.Record.Validate(i.Payload) != nil:
		reason = "schema_invalid"
	}
	if reason == "accepted" && hasParent {
		pt, e := loadCompositionTemplate(ctx, q, parent.TemplateID)
		if e != nil {
			return e
		}
		if !slices.Contains(pt.Templates, t.ID) {
			reason = "template_denied"
		}
	}
	if reason == "accepted" {
		for _, capability := range t.Capabilities {
			if !slices.Contains(root.Capabilities, capability) {
				reason = "capability_denied"
			}
		}
	}
	if reason == "accepted" {
		spent := x.charges[root.ID]
		if spent.Invocations >= root.Limits.MaxInvocations {
			reason = "invocation_limit"
		} else if int64(len(t.Members)) > root.Limits.MaxCreatedRuns-spent.CreatedRuns {
			reason = "created_run_limit"
		}
	}
	if a.Reason != reason || a.Accepted != (reason == "accepted") {
		return corrupt("admission prefix decision")
	}
	return nil
}
func (x *compositionPrefix) expectedOutcome(operation string) (compositionOutcome, error) {
	var o compositionOutcome
	requireReply := true
	if strings.HasPrefix(operation, "root:") {
		root, ok := x.roots[strings.TrimPrefix(operation, "root:")]
		if !ok {
			return o, ErrCorrupt
		}
		o = compositionOutcome{InvocationID: operation, OperationID: operation, Caller: root.Run}
		requireReply = false
	} else {
		a, ok := x.admitted[operation]
		if !ok {
			return o, corrupt("outcome before admission")
		}
		o = compositionOutcome{InvocationID: a.Invocation.ID, OperationID: operation, TemplateID: a.Invocation.TemplateID, Caller: a.Invocation.Caller}
		if !a.Accepted {
			o.Status = "rejected"
			o.Reason = a.Reason
			return o, nil
		}
	}
	f, has := x.freezes[o.InvocationID]
	closure, closed := x.closures[o.InvocationID]
	closureReason := ""
	if closed && (!has || f.Status == "succeeded" || closure.Freeze.DecisionOrder < f.DecisionOrder) {
		f = closure.Freeze
		has = true
		closureReason = closure.Reason
	}
	if !has {
		return o, corrupt("publication before scope freeze")
	}
	responses := []map[string]Value{}
	for _, id := range f.ReplyIntents {
		r, ok := x.replies[id]
		if !ok {
			return o, corrupt("publication before reply admission")
		}
		if r.Accepted {
			responses = append(responses, r.Intent.Payload)
		}
	}
	o.Status = f.Status
	o.Reason = "member_" + f.Status
	if closureReason != "" {
		o.Reason = closureReason
	}
	if f.Status == "succeeded" {
		for _, id := range f.InvocationIntents {
			a, ok := x.admitted[id]
			if !ok {
				return o, corrupt("publication before invocation admission")
			}
			if a.Accepted {
				if err := x.joinDescendants(a, map[string]bool{}); err != nil {
					return o, err
				}
				if _, ok = x.outcomes[a.Invocation.ID]; !ok {
					return o, corrupt("publication before descendant outcome")
				}
			}
		}
		for _, id := range f.ExternalIntents {
			if !x.known[id] {
				return o, corrupt("publication before known external result")
			}
		}
		if requireReply && len(responses) == 0 {
			o.Status = "failed"
			o.Reason = "missing_reply"
		}
	}
	if len(f.ReplyIntents) > 1 {
		if closureReason == "" {
			o.Status = "failed"
			o.Reason = "conflicting_replies"
		}
	} else if len(responses) == 1 {
		o.ResponsePresent = true
		o.Response = responses[0]
	}
	return o, nil
}
func (x *compositionPrefix) validateConsumption(ctx context.Context, q queryer, f compositionFact, seen map[string]bool) error {
	var selected sql.NullString
	if err := q.QueryRowContext(ctx, "SELECT selected_observation FROM history WHERE subject=? AND generation=? AND sequence=?", f.Run.Subject, f.Run.Generation, f.Sequence).Scan(&selected); err != nil {
		return err
	}
	if !selected.Valid {
		return nil
	}
	var raw []byte
	var o Observation
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM observations WHERE id=?", selected.String).Scan(&raw); err != nil {
		return err
	}
	if err := decode(raw, &o); err != nil {
		return err
	}
	switch o.Provenance {
	case "runtime-invocation-request":
		a, ok := x.admitted[o.Operation]
		if !ok || !a.Accepted {
			return corrupt("request consumption before creation")
		}
	case "runtime-invocation":
		a, ok := x.admitted[o.Operation]
		if !ok {
			return corrupt("outcome consumption before admission")
		}
		if _, ok = x.outcomes[a.Invocation.ID]; !ok {
			return corrupt("consumption before outcome publication")
		}
	case "runtime-adapter", "resource-receipt":
		i, err := loadIntent(ctx, q, o.Operation)
		if err != nil {
			return err
		}
		root, err := compositionMembershipRoot(ctx, q, i.Run)
		if err != nil {
			return err
		}
		if root != "" && !x.known[o.Operation] {
			return corrupt("consumption before external result")
		}
	default:
		p, err := compositionObservationPredecessor(ctx, q, f.Run, f.Sequence)
		if err != nil {
			return err
		}
		if p.ID != "" && !seen[p.ID] {
			return corrupt("observation consumed before source fact")
		}
	}
	return nil
}

func (x *compositionPrefix) joinDescendants(a compositionAdmission, seen map[string]bool) error {
	if seen[a.Invocation.ID] {
		return ErrCorrupt
	}
	seen[a.Invocation.ID] = true
	if _, ok := x.outcomes[a.Invocation.ID]; !ok {
		return corrupt("success before transitive descendant result")
	}
	for id, i := range x.intents {
		if !slices.Contains(a.Members, i.Run) {
			continue
		}
		switch i.Kind {
		case "invocation":
			child, ok := x.admitted[id]
			if !ok {
				return corrupt("success before transitive invocation admission")
			}
			if child.Accepted {
				if err := x.joinDescendants(child, seen); err != nil {
					return err
				}
			}
		case "external", "adapter":
			if !x.known[id] {
				return corrupt("success before transitive external result")
			}
		}
	}
	return nil
}
