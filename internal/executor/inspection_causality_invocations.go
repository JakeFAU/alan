package executor

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

// All secondary references here are backed by an original source operation.
// The table indexes are transient, and taking each required row also detects
// orphan duplicates and absent reverse links in the same cut.
func (c *causalCut) readInvocations() error {
	for id, src := range c.intents {
		if src.Kind == "invocation" {
			v := &InvocationView{OperationID: id, Caller: src.Run, Sequence: src.Sequence, SourceRevision: src.revision, TemplateID: src.Target, Inputs: src.Payload, State: InvocationCommitted}
			if auth, ok := c.auth[id]; ok {
				v.Authorization = &auth
				v.State = InvocationSourceRefused
				if auth.Allowed {
					v.State = InvocationAuthorized
				}
			}
			c.invocations[id] = v
		}
	}
	admitted, err := indexCausalRows(c.tables["composition_invocations"], "operation_id")
	if err != nil {
		return err
	}
	bindings, err := indexCausalRows(c.tables["composition_bindings"], "invocation_id")
	if err != nil {
		return err
	}
	requests, err := indexCausalRows(c.tables["composition_requests"], "invocation_id")
	if err != nil {
		return err
	}
	charges := causalIndex{}
	for _, row := range c.tables["composition_charges"] {
		if row["kind"] == "invocation" {
			id, _ := row["operation_id"].(string)
			charges[id] = row
		}
	}
	for _, row := range c.tables["composition_templates"] {
		t, err := inspectionDecode[compositionTemplate](row, "descriptor")
		if err != nil {
			return err
		}
		if t.ID != row["id"] {
			return corrupt("template row identity")
		}
		c.templates[t.ID] = t
	}
	for _, row := range c.tables["composition_admission_decisions"] {
		if len(inspectText(row, "descriptor")) > c.limits.MaxOutputBytes {
			return ErrLimit
		}
		a, err := inspectionDecode[compositionAdmission](row, "descriptor")
		if err != nil {
			return err
		}
		i := a.Invocation
		v := c.invocations[i.OperationID]
		if v == nil || v.Admission != nil || row["operation_id"] != i.OperationID || row["root_id"] != i.RootID ||
			i.ID != compositionInvocationID(c.snap.Namespace, i.OperationID) || i.TemplateID != v.TemplateID || i.Caller != v.Caller ||
			i.RootID == "" || i.RootID != c.membership[v.Caller] || i.SourceSequence != v.Sequence ||
			i.SourceRevision != c.intents[v.OperationID].revision || !reflect.DeepEqual(i.Inputs, v.Inputs) || a.Accepted != (a.Reason == "accepted") {
			return corrupt("invocation admission does not match original operation")
		}
		order, err := c.protocolOrder(compositionAdmissionFact, i.OperationID, i.RootID, i.Caller, i.SourceSequence, row)
		if err != nil {
			return err
		}
		auth, ok := c.auth[i.OperationID]
		authOrder := c.factOrder(compositionAuthorizationFact, i.OperationID)
		if !ok || authOrder == 0 || authOrder >= order || !auth.Allowed && a.Reason != "source_denied:"+auth.Reason || auth.Allowed && strings.HasPrefix(a.Reason, "source_denied:") {
			return corrupt("invocation admission bypassed source authorization")
		}
		v.Admission = &a
		c.byID[i.ID] = v
		if !a.Accepted {
			if a.BindingID != "" || len(a.Members) > 0 {
				return corrupt("refused invocation created a binding")
			}
			v.State = InvocationRefused
			continue
		}
		v.State = InvocationAdmitted
		full := admitted.take(i.OperationID)
		if full == nil || full["id"] != i.ID || full["template_id"] != i.TemplateID || full["root_id"] != i.RootID ||
			full["descriptor"] != compositionJSON(i) || full["admission"] != row["descriptor"] {
			return corrupt("accepted invocation lacks original member admission")
		}
		t, ok := c.templates[i.TemplateID]
		if !ok {
			return corrupt("accepted invocation template missing")
		}
		binding, err := inspectionDecode[compositionBinding](bindings.take(i.ID), "descriptor")
		if err != nil {
			return err
		}
		if binding.Protocol != compositionProtocolVersion || binding.ID == "" || binding.ID != a.BindingID || binding.OperationID != v.OperationID || binding.Caller != v.Caller || binding.Depth != i.Depth {
			return corrupt("admission and binding identity differ")
		}
		// buildDeployment authenticated every original binding and resolved its members.

		members := make([]RunID, 0, len(t.Members))
		for _, name := range sortedKeys(t.Members) {
			run := binding.Members[name]
			rv := c.runs[run]
			if rv == nil || rv.OwningInvocationID != i.ID || rv.ProgramID != t.Members[name].ProgramID || !reflect.DeepEqual(rv.Initial, t.Members[name].Initial) {
				return corrupt("invoked member does not match retained origin")
			}
			members = append(members, run)
		}
		if !slices.Equal(members, a.Members) {
			return corrupt("admitted members differ from original template and binding")
		}
		if err = c.checkInvocationRequest(v, t, binding, requests.take(i.ID), order); err != nil {
			return err
		}
		chargeRow := charges.take(i.OperationID)
		debit, err := inspectionDecode[compositionCharge](chargeRow, "descriptor")
		if err != nil {
			return err
		}
		want := compositionCharge{RootID: i.RootID, OperationID: i.OperationID, Kind: "invocation", Invocations: 1, CreatedRuns: int64(len(members)), DecisionOrder: order}
		if debit != want || chargeRow["root_id"] != want.RootID || chargeRow["decision_order"] != order {
			return corrupt("accepted invocation lacks original resource charge")
		}
	}
	if len(admitted)+len(bindings)+len(requests)+len(charges) != 0 {
		return corrupt("invocation protocol has an orphan membership, request, or charge")
	}
	for _, v := range c.invocations {
		if v.Admission == nil {
			continue
		}
		i := v.Admission.Invocation
		parent := c.runs[i.Caller].OwningInvocationID
		if i.ParentInvocationID != parent {
			return corrupt("invocation parent differs from caller membership")
		}
		depth := int64(1)
		if parent != "" {
			pv := c.byID[parent]
			if pv == nil || !pv.Admission.Accepted {
				return corrupt("missing original parent invocation")
			}
			depth += pv.Admission.Invocation.Depth
		}
		if i.Depth != depth {
			return corrupt("invocation depth or parent cycle")
		}
	}
	if err := c.readReplies(); err != nil {
		return err
	}
	if err := c.readFreezes(); err != nil {
		return err
	}
	// A refused built-in external operation has no remote receipt. Its
	// reservation decision is already a known local disposition. Adapters
	// instead persist a separate result for this case.
	c.deniedExternal = map[string]int64{}
	for _, row := range c.tables["external_attempts"] {
		a, err := inspectionDecode[ExternalAttempt](row, "descriptor")
		if err != nil {
			return err
		}
		if !a.Allowed {
			c.deniedExternal[a.Intent.ID] = c.factOrder(compositionReservationFact, a.Intent.ID)
		}
	}
	return c.readPublications()
}

func (c *causalCut) knownExternalOrder(id string) int64 {
	if denied := c.deniedExternal[id]; denied != 0 {
		return denied
	}
	return c.factOrder(compositionExternalResultFact, id)
}

func (c *causalCut) factOrder(kind compositionFactKind, operation string) int64 {
	return c.facts[compositionIdentity("alan/fact/"+string(kind)+"/v1", operation)].Order
}

func (c *causalCut) protocolOrder(kind compositionFactKind, op, root string, run RunID, seq int64, row map[string]any) (int64, error) {
	id := compositionIdentity("alan/fact/"+string(kind)+"/v1", op)
	f, ok := c.facts[id]
	column := "descriptor"
	if kind == compositionAuthorizationFact {
		column = "result"
	}
	raw := inspectText(row, column)
	if !ok || f.Kind != kind || f.OperationID != op || f.Run != run || f.Sequence != seq || !slices.Equal(f.RootIDs, []string{root}) ||
		f.RecordDigest != compositionIdentity("alan/"+string(kind)+"-row/v1", json.RawMessage(raw)) {
		return 0, corrupt("missing or mismatched %s fact", kind)
	}
	return f.Order, nil
}

func (c *causalCut) checkInvocationRequest(v *InvocationView, t compositionTemplate, binding compositionBinding, row map[string]any, order int64) error {
	entry := binding.Members[t.EntryMember]
	p := c.programs[t.Members[t.EntryMember].ProgramID]
	if p.Author == nil {
		return corrupt("invocation entry program lacks port metadata")
	}
	port, ok := p.Author.Ports[t.EntryPort]
	if !ok || port.Receive.Record.Validate(v.Inputs) != nil {
		return corrupt("invocation entry does not accept original inputs")
	}
	expected := Observation{ID: compositionIdentity("alan/invocation-request/v1", binding.InvocationID), Run: entry, Kind: "message", Provenance: "runtime-invocation-request", Source: v.Caller.Subject, SourceGeneration: v.Caller.Generation, Obligation: port.Obligation, Operation: v.OperationID, Payload: v.Inputs}
	o := c.observations[expected.ID]
	if row == nil || row["observation_id"] != expected.ID || row["descriptor"] != compositionJSON(expected) || o == nil || !reflect.DeepEqual(o.Observation, expected) {
		return corrupt("invocation request does not match original admission")
	}
	if err := c.selectionAfter(o, order); err != nil {
		return err
	}
	v.RequestObservationID = o.ID
	o.Origin = "invocation_request"
	o.SourceOperationID = v.OperationID
	o.ProducerReduction = &ReductionRef{v.Caller, v.Sequence}
	source := v.Caller
	o.ProducerRun = &source
	return nil
}

func (c *causalCut) selectionAfter(o *ObservationView, order int64) error {
	if o.SelectedAt != nil {
		ref := *o.SelectedAt
		f := c.facts[compositionIdentity("alan/fact/execution/v1", []any{ref.Run, ref.Sequence})]
		if f.Order <= order {
			return corrupt("invocation observation selected before creation")
		}
	}
	return nil
}

func (c *causalCut) isInvocationReply(v *InvocationView, src Intent) bool {
	t := c.templates[v.TemplateID]
	p := c.programs[t.Members[t.EntryMember].ProgramID]
	return p.Author != nil && src.Kind == "message" && src.Run == compositionMemberID(v.Admission.Invocation.ID, t.EntryMember) &&
		src.Target == v.Caller.Subject && src.Generation == v.Caller.Generation && src.Obligation == p.Author.Ports[t.EntryPort].Obligation
}

func (c *causalCut) readReplies() error {
	for _, row := range c.tables["composition_replies"] {
		if len(inspectText(row, "descriptor")) > c.limits.MaxObservationBytes {
			return ErrLimit
		}
		r, err := inspectionDecode[compositionReply](row, "descriptor")
		if err != nil {
			return err
		}
		v := c.byID[r.InvocationID]
		src, ok := c.intents[r.Intent.ID]
		auth, hasAuth := c.auth[r.Intent.ID]
		if v == nil || !v.Admission.Accepted || row["operation_id"] != r.Intent.ID || row["invocation_id"] != r.InvocationID ||
			!ok || r.RootID != v.Admission.Invocation.RootID || !reflect.DeepEqual(src.Intent, r.Intent) || r.Authorization != auth || !hasAuth ||
			!c.isInvocationReply(v, r.Intent) {
			return corrupt("invocation reply provenance")
		}
		t := c.templates[v.TemplateID]
		port := c.programs[t.Members[t.EntryMember].ProgramID].Author.Ports[t.EntryPort]
		reason := "recorded_reply"
		if !auth.Allowed {
			reason = "source_denied:" + auth.Reason
		} else if port.Send.Record.Validate(src.Payload) != nil {
			reason = "schema_invalid"
		}
		if r.Reason != reason || r.Accepted != (reason == "recorded_reply") {
			return corrupt("invocation reply decision")
		}
		order, err := c.protocolOrder(compositionReplyFact, r.Intent.ID, r.RootID, r.Intent.Run, r.Intent.Sequence, row)
		if err != nil {
			return err
		}
		if authOrder := c.factOrder(compositionAuthorizationFact, r.Intent.ID); authOrder == 0 || authOrder >= order {
			return corrupt("reply precedes authorization")
		}
		c.replies[r.Intent.ID] = r
		v.Replies = append(v.Replies, r)
	}
	for _, v := range c.byID {
		slices.SortFunc(v.Replies, func(a, b InvocationReply) int { return strings.Compare(a.Intent.ID, b.Intent.ID) })
	}
	return nil
}

func (c *causalCut) readFreezes() error {
	// Index source operations by the owning invocation once. Each operation can
	// contribute to one terminal freeze and its optional retirement closure.
	byScope := map[string][]Intent{}
	for _, src := range c.intents {
		if run := c.runs[src.Run]; run != nil && run.OwningInvocationID != "" {
			byScope[run.OwningInvocationID] = append(byScope[run.OwningInvocationID], src.Intent)
		}
	}
	for _, row := range c.tables["composition_freezes"] {
		if len(inspectText(row, "descriptor")) > c.limits.MaxDataBytes {
			return ErrLimit
		}
		f, err := inspectionDecode[compositionFreeze](row, "descriptor")
		if err != nil {
			return err
		}
		if strings.HasPrefix(f.InvocationID, "root:") {
			continue
		}
		v := c.byID[f.InvocationID]
		if v == nil || !v.Admission.Accepted || row["scope_id"] != f.InvocationID || row["root_id"] != v.Admission.Invocation.RootID {
			return corrupt("orphan invocation freeze")
		}
		order, err := c.protocolOrder(compositionFreezeFact, f.InvocationID, v.Admission.Invocation.RootID, f.Run, f.Sequence, row)
		if err != nil {
			return err
		}
		if order != f.DecisionOrder || !c.freezeBelongs(v.Admission.Members, v, f, byScope[f.InvocationID], false) {
			return corrupt("invocation freeze has an invalid causal boundary")
		}
		v.Freeze = &f
		v.State = InvocationFrozen
	}
	for _, row := range c.tables["composition_closures"] {
		if len(inspectText(row, "descriptor")) > c.limits.MaxDataBytes {
			return ErrLimit
		}
		close, err := inspectionDecode[compositionClosure](row, "descriptor")
		if err != nil {
			return err
		}
		f := close.Freeze
		if strings.HasPrefix(f.InvocationID, "root:") {
			continue
		}
		v := c.byID[f.InvocationID]
		trigger := c.facts[f.Trigger]
		if v == nil || !v.Admission.Accepted || row["scope_id"] != f.InvocationID || row["root_id"] != close.RootID || close.RootID != v.Admission.Invocation.RootID ||
			row["decision_order"] != f.DecisionOrder || close.Reason != "subject_retired" || f.Status != "cancelled" || trigger.Kind != compositionLifecycleFact || trigger.OperationID != close.LifecycleOperationID ||
			trigger.Order != f.DecisionOrder || !c.freezeBelongs(v.Admission.Members, v, f, byScope[f.InvocationID], true) {
			return corrupt("invalid invocation retirement closure")
		}
		if v.Freeze != nil && (v.Freeze.DecisionOrder >= f.DecisionOrder || v.Freeze.Status != "succeeded") {
			return corrupt("retirement contradicts an earlier invocation decision")
		}
		v.Closure = &InvocationClosure{OperationID: close.LifecycleOperationID, Reason: close.Reason, Freeze: f}
		v.State = InvocationClosed
	}
	return nil
}

func (c *causalCut) freezeBelongs(members []RunID, v *InvocationView, f compositionFreeze, sources []Intent, closure bool) bool {
	run := c.runs[f.Run]
	trigger := c.facts[f.Trigger]
	if run == nil || !slices.Contains(members, f.Run) || trigger.ID == "" || trigger.Run != f.Run || trigger.Sequence != f.Sequence ||
		(trigger.Kind != compositionExecutionFact && trigger.Kind != compositionLifecycleFact) || trigger.Order > f.DecisionOrder {
		return false
	}
	if !closure {
		if trigger.Order+1 != f.DecisionOrder || f.Status == "open" {
			return false
		}
		if trigger.Kind == compositionExecutionFact {
			after, ok := c.after[ReductionRef{Run: f.Run, Sequence: f.Sequence}]
			if !ok || f.Status == "open" || f.Status != after.Status {
				return false
			}
		} else {
			d := c.lifecycle[trigger.OperationID]
			if !d.Accepted || d.Before == nil || d.After == nil || d.Before.Projection.Status != "open" || d.After.Projection.Status != f.Status || d.Before.Run != f.Run {
				return false
			}
		}
		if f.Status == "succeeded" {
			for _, member := range members {
				run := c.runs[member]
				order := c.executionOrder(member, run.Sequence)
				if run.Projection.Status != "succeeded" || order == 0 || order >= f.DecisionOrder {
					return false
				}
			}
		}
	} else {
		d := c.lifecycle[trigger.OperationID]
		if !d.Accepted || d.Request.Operation != "RetireSubject" || d.Before == nil || d.After == nil || d.Before.Retired || !d.After.Retired || d.Before.Run != f.Run {
			return false
		}
	}
	want := compositionFreeze{ReplyIntents: []string{}, InvocationIntents: []string{}, ExternalIntents: []string{}}
	for _, i := range sources {
		fact := c.facts[compositionIdentity("alan/fact/execution/v1", []any{i.Run, i.Sequence})]
		if fact.Order == 0 {
			return false
		}
		if fact.Order > f.DecisionOrder {
			continue
		}
		switch i.Kind {
		case "invocation":
			want.InvocationIntents = append(want.InvocationIntents, i.ID)
		case "adapter", "external":
			want.ExternalIntents = append(want.ExternalIntents, i.ID)
		case "message":
			if v != nil && c.isInvocationReply(v, i) {
				want.ReplyIntents = append(want.ReplyIntents, i.ID)
			}
		}
	}
	slices.Sort(want.ReplyIntents)
	slices.Sort(want.InvocationIntents)
	slices.Sort(want.ExternalIntents)
	return slices.Equal(want.ReplyIntents, f.ReplyIntents) && slices.Equal(want.InvocationIntents, f.InvocationIntents) && slices.Equal(want.ExternalIntents, f.ExternalIntents)
}

func (c *causalCut) readPublications() error {
	results, err := indexCausalRows(c.tables["composition_results"], "invocation_id")
	if err != nil {
		return err
	}
	deliveries, err := indexCausalRows(c.tables["composition_outcome_deliveries"], "operation_id")
	if err != nil {
		return err
	}
	for _, row := range c.tables["composition_publications"] {
		out, err := inspectionDecode[compositionOutcome](row, "descriptor")
		if err != nil {
			return err
		}
		v := c.invocations[out.OperationID]
		if v == nil || v.Admission == nil || row["operation_id"] != out.OperationID || out.InvocationID != v.Admission.Invocation.ID || out.Caller != v.Caller || out.TemplateID != v.TemplateID || out.ID != compositionIdentity("alan/invocation-outcome/v1", out.InvocationID) {
			return corrupt("outcome does not belong to original invocation")
		}
		order, err := c.protocolOrder(compositionOutcomeFact, v.OperationID, v.Admission.Invocation.RootID, v.Caller, v.Sequence, row)
		if err != nil {
			return err
		}
		if out.DecisionOrder != order || order <= c.factOrder(compositionAdmissionFact, v.OperationID) {
			return corrupt("publication before admission")
		}
		if v.Admission.Accepted {
			result := results.take(out.InvocationID)
			if result == nil || result["decision_order"] != order || result["descriptor"] != row["descriptor"] {
				return corrupt("publication and original child result differ")
			}
		}
		if err := c.checkOutcome(v, out); err != nil {
			return err
		}
		want := invocationObservation(out, c.intents[v.OperationID].Intent)
		delivered := deliveries.take(v.OperationID)
		o := c.observations[want.ID]
		if delivered == nil || delivered["observation_id"] != want.ID || o == nil || !reflect.DeepEqual(o.Observation, want) {
			return corrupt("publication missing its original outcome observation")
		}
		if err := c.selectionAfter(o, order); err != nil {
			return err
		}
		v.Outcome = &out
		v.OutcomeObservationID = o.ID
		o.Origin = "invocation_outcome"
		o.SourceOperationID = v.OperationID
		v.SelectedAt = o.SelectedAt
		v.State = InvocationPublished
		if v.SelectedAt != nil {
			v.State = InvocationConsumed
		}
	}
	if len(results)+len(deliveries) != 0 {
		return corrupt("original result or delivery missing its invocation publication")
	}
	// Each fact must still have its source row, even when all foreign keys in
	// the protocol tables were removed together.
	for _, f := range c.facts {
		switch f.Kind {
		case compositionAdmissionFact:
			if v := c.invocations[f.OperationID]; v == nil || v.Admission == nil {
				return corrupt("orphan invocation admission fact")
			}
		case compositionReplyFact:
			if _, ok := c.replies[f.OperationID]; !ok {
				return corrupt("orphan invocation reply fact")
			}
		case compositionFreezeFact:
			if !strings.HasPrefix(f.OperationID, "root:") {
				if v := c.byID[f.OperationID]; v == nil || v.Freeze == nil {
					return corrupt("orphan invocation freeze fact")
				}
			}
		case compositionOutcomeFact:
			if !strings.HasPrefix(f.OperationID, "root:") {
				if v := c.invocations[f.OperationID]; v == nil || v.Outcome == nil {
					return corrupt("orphan invocation publication fact")
				}
			}
		}
	}
	return nil
}

func (c *causalCut) checkOutcome(v *InvocationView, out compositionOutcome) error {
	status, reason := "rejected", v.Admission.Reason
	var response map[string]Value
	if v.Admission.Accepted {
		f := v.Freeze
		reason = ""
		if close := v.Closure; close != nil && (f == nil || f.Status == "succeeded" || close.Freeze.DecisionOrder < f.DecisionOrder) {
			f = &close.Freeze
			reason = close.Reason
		}
		if f == nil || f.DecisionOrder >= out.DecisionOrder {
			return corrupt("outcome before scope freeze")
		}
		status = f.Status
		if reason == "" {
			reason = "member_" + status
		}
		hasReply := false
		for _, id := range f.ReplyIntents {
			reply, ok := c.replies[id]
			if !ok || c.factOrder(compositionReplyFact, id) >= out.DecisionOrder {
				return corrupt("publication before original reply admission")
			}
			if reply.Accepted {
				hasReply = true
				if len(f.ReplyIntents) == 1 {
					response = reply.Intent.Payload
					v.ResponseOperationID = id
				}
			}
		}
		if len(f.ReplyIntents) > 1 && reason != "subject_retired" {
			status, reason = "failed", "conflicting_replies"
		} else if status == "succeeded" && !hasReply {
			status, reason = "failed", "missing_reply"
		}
		if f.Status == "succeeded" {
			for _, id := range f.ExternalIntents {
				order := c.knownExternalOrder(id)
				if order == 0 || order >= out.DecisionOrder {
					return corrupt("publication before known external result")
				}
			}
			for _, id := range f.InvocationIntents {
				child := c.invocations[id]
				if child == nil || child.Admission == nil || c.factOrder(compositionAdmissionFact, id) >= out.DecisionOrder {
					return corrupt("publication before child admission")
				}
				if child.Admission.Accepted {
					issued := c.factOrder(compositionOutcomeFact, id)
					if issued == 0 || issued >= out.DecisionOrder {
						return corrupt("publication before child result")
					}
				}
			}
		}
	}
	if out.Status != status || out.Reason != reason || out.ResponsePresent != (response != nil) || !reflect.DeepEqual(out.Response, response) {
		return corrupt("published child outcome differs from original reply or terminal decision")
	}
	return nil
}
