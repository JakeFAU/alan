package executor

import (
	"maps"

	"github.com/JakeFAU/alan/adapters/receiptquery"
)

// Read the ordered facts once. A final source projection cannot say whether
// authority, budget, or a drained original was available at an earlier cut.
type effectPrefix struct {
	*effectCut
	seq        map[RunID]int64
	closed     map[RunID]bool
	retired    map[RunID]bool
	scopeEnd   map[RunID]int64
	limits     map[string]int64
	spent      map[string]int64
	latestRead map[RunID]int64
}

func (c *effectCut) checkEffectOrder() error {
	gates, err := indexDispatchAdmissions(c.tables["adapter_dispatch_admissions"], len(c.facts))
	if err != nil {
		return err
	}
	if len(c.facts) == 0 {
		return nil
	}
	if err := checkDispatchCapacity(gates, len(c.facts), c.snap.Composition.Roots, maps.Values(c.facts)); err != nil {
		return err
	}
	p := effectPrefix{effectCut: c, seq: map[RunID]int64{}, closed: map[RunID]bool{}, retired: map[RunID]bool{},
		scopeEnd: map[RunID]int64{}, limits: map[string]int64{}, spent: map[string]int64{}, latestRead: map[RunID]int64{}}
	for _, s := range c.snap.Deployment.Subjects {
		id := RunID{s.ID, s.Generation}
		p.seq[id] = 0
		p.closed[id] = c.runs[id].Initial.Status != "open"
		p.retired[id] = s.Retired
	}
	if err := p.indexScopeEnds(); err != nil {
		return err
	}
	roots := map[string]compositionRoot{}
	templates := map[string]*compositionTemplate{}
	spent := map[string]compositionCharge{}
	for _, r := range c.snap.Composition.Roots {
		roots[r.ID] = r
	}
	for i := range c.snap.Composition.Templates {
		t := &c.snap.Composition.Templates[i]
		templates[t.ID] = t
	}
	for _, row := range c.tables["composition_facts"] {
		f := c.facts[inspectText(row, "id")]
		id := f.OperationID
		e := c.effects[id]
		switch f.Kind {
		case compositionExecutionFact:
			step := c.steps[ReductionRef{f.Run, f.Sequence}]
			if seq, active := p.seq[f.Run]; !active || step == nil || seq+1 != f.Sequence || p.closed[f.Run] || p.retired[f.Run] {
				return corrupt("misordered source reduction")
			}
			p.seq[f.Run], p.closed[f.Run] = f.Sequence, step.Status != "open"
		case compositionLifecycleFact:
			r, ok := c.lifecycle[id]
			if !ok {
				return corrupt("ordered lifecycle decision without its original")
			}
			if r.Accepted && r.After != nil && c.membership[f.Run] != "" {
				p.retired[f.Run] = r.After.Retired
				p.closed[f.Run] = r.After.Projection.Status != "open"
			}
		case compositionAdmissionFact:
			v := c.invocations[id]
			if v == nil || v.Admission == nil {
				return corrupt("ordered admission without its original")
			}
			i := c.intents[id]
			r := c.runs[i.Run]
			root := roots[r.RootID]
			t := templates[v.TemplateID]
			reason := admissionEligibility(i.Intent, v.Admission.Invocation.Depth, c.programs[r.ProgramID], *v.Authorization, p.closed[i.Run] || r.Revision != i.revision, p.scopeClosed(i.Run, f.Order), root, t != nil)
			balance := spent[root.ID]
			if reason == "accepted" {
				var parent *compositionTemplate
				if r.OwningInvocationID != "" {
					parent = templates[c.invocations[r.OriginOperationID].TemplateID]
				}
				reason = admissionAllowance(root, *t, parent, balance)
			}
			if reason != v.Admission.Reason {
				return corrupt("invocation admission contradicts its original authority or resource balance")
			}
			if v.Admission.Accepted {
				balance.Invocations++
				balance.CreatedRuns += int64(len(v.Admission.Members))
				spent[root.ID] = balance
			}
			for _, m := range v.Admission.Members {
				p.seq[m] = 0
				p.closed[m] = c.runs[m].Initial.Status != "open"
			}
		case compositionAuthorizationFact:
			if e == nil {
				break
			}
			a := e.Authorization
			if a == nil || f.Order != p.order(compositionAuthorizationFact, id) || p.seq[e.Source] < e.Sequence ||
				a.Source.Sequence != p.seq[e.Source] {
				return corrupt("effect authorized before original source reduction")
			}
			reason := "authorized"
			switch {
			case a.Source.Revision != e.SourceRevision:
				reason = "source_revision_changed"
			case p.retired[e.Source]:
				reason = "source_retired"
			case p.closed[e.Source]:
				reason = "source_closed"
			}
			if a.Reason != reason || a.Allowed != (reason == "authorized") {
				return corrupt("effect authorization contradicts historical authority")
			}
		case compositionReservationFact:
			r, ok := c.reservations[id]
			auth := p.order(compositionAuthorizationFact, id)
			if !ok || e == nil || p.order(compositionReservationFact, id) != f.Order || auth == 0 || auth >= f.Order {
				return corrupt("reservation before original authorization")
			}
			// The immutable permission check already established base refusals.
			if e.Authorization.Allowed && r.Reason != "grant_denied" {
				reason := "authorized"
				switch {
				case p.scopeClosed(e.Source, f.Order):
					reason = "scope_closed"
				case p.spent[r.RootID] >= p.limits[r.RootID]:
					reason = "effect_limit"
				}
				if r.Reason != reason || r.Allowed != (reason == "authorized") {
					return corrupt("reservation contradicts historical scope or budget")
				}
			}
			if r.Allowed {
				p.spent[r.RootID]++
			}
		case compositionReadFact:
			ord := c.ordinals[id]
			if e == nil || f.Sequence < 1 || f.Sequence > int64(len(ord)) {
				return corrupt("receipt read without its original ordinal")
			}
			o := ord[f.Sequence-1]
			if f.Order != o.Prefix+1 || p.seq[e.Source] != o.Resolver.Sequence || p.closed[e.Source] || p.retired[e.Source] ||
				p.scopeClosed(e.Source, f.Order) || !p.closed[o.OriginalRun] || p.seq[o.OriginalRun] != o.OriginalTerminalSequence ||
				c.resultOrder[id] != 0 && c.resultOrder[id] < f.Order {
				return corrupt("receipt read outside its historical authority")
			}
			p.latestRead[o.OriginalRun] = o.Prefix
		case compositionExternalResultFact:
			res := p.order(compositionReservationFact, id)
			if e == nil || c.resultOrder[id] != f.Order || res == 0 || res >= f.Order {
				return corrupt("result before original reservation")
			}
			if e.Capability == receiptReadCapability && protocolSchemaNumber(c.snap.StorageVersion) == reconciliationSchemaVersion &&
				e.AdapterAttempt != nil && e.AdapterAttempt.Allowed && e.AdapterResult.Read == nil {
				if err := p.checkReadRefusal(e, f.Order); err != nil {
					return err
				}
			}
		}
		for _, gate := range gates[f.Order] {
			e := c.effects[gate.OperationID]
			if e == nil || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.DispatchAdmission != nil || gate.PreparedDigest != e.AdapterAttempt.Prepared.Digest || c.membership[e.Source] == "" {
				return corrupt("dispatch gate differs from original reservation")
			}
			res := p.order(compositionReservationFact, gate.OperationID)
			if res == 0 || res > f.Order || p.closed[e.Source] || p.retired[e.Source] || p.scopeEnd[e.Source] != 0 && p.scopeEnd[e.Source] <= f.Order || c.resultOrder[e.OperationID] != 0 && c.resultOrder[e.OperationID] <= f.Order {
				return corrupt("dispatch gate outside original authority")
			}
			e.DispatchAdmission = &gate
			e.Dispatch.Maximum = new(1)
		}
	}
	if c.snap.Features.DispatchAdmissions {
		for _, e := range c.effects {
			if e.AdapterResult != nil && e.AdapterAttempt.Allowed && c.membership[e.Source] != "" && e.DispatchAdmission == nil {
				return corrupt("result without its dispatch admission")
			}
		}
	}
	return nil
}

func (p *effectPrefix) order(kind compositionFactKind, id string) int64 {
	f := p.facts[compositionIdentity("alan/fact/"+string(kind)+"/v1", id)]
	if f.Kind == kind && f.OperationID == id {
		return f.Order
	}
	return 0
}

func (p *effectPrefix) scopeClosed(id RunID, order int64) bool {
	end := p.scopeEnd[id]
	return end != 0 && end < order
}

// Compute the earliest closing decision on each ownership path once. Even a
// deeply nested tree does not require a full ancestry walk for every effect.
func (p *effectPrefix) indexScopeEnds() error {
	ends := map[string]int64{}
	set := func(scope string, n int64) {
		if n > 0 && (ends[scope] == 0 || n < ends[scope]) {
			ends[scope] = n
		}
	}
	for _, r := range p.snap.Roots {
		p.limits[r.RootID] = r.Limits.MaxEffects
		t := p.terminals[r.Run]
		if t == nil {
			continue
		}
		order := p.facts[compositionIdentity("alan/fact/execution/v1", []any{r.Run, inspectInt(t, "sequence")})].Order
		if op := inspectText(t, "operation_id"); op != "" {
			order = p.order(compositionLifecycleFact, op)
		}
		set("root:"+r.RootID, order)
	}
	for _, row := range p.tables["composition_freezes"] {
		f, err := inspectionDecode[compositionFreeze](row, "descriptor")
		if err != nil {
			return err
		}
		if f.Status != "succeeded" {
			set(f.InvocationID, f.DecisionOrder)
		}
	}
	for _, row := range p.tables["composition_closures"] {
		set(inspectText(row, "scope_id"), inspectInt(row, "decision_order"))
	}
	for _, v := range p.invocations {
		if v.Outcome != nil {
			set(v.Outcome.InvocationID, v.Outcome.DecisionOrder)
		}
	}
	for _, row := range p.tables["composition_root_publications"] {
		set(inspectText(row, "scope_id"), inspectInt(row, "decision_order"))
	}
	var resolve func(*ExecutionRun) int64
	resolve = func(r *ExecutionRun) int64 {
		if end, ok := p.scopeEnd[r.ID]; ok {
			return end
		}
		scope := "root:" + r.RootID
		if r.OwningInvocationID != "" {
			v := p.invocations[r.OriginOperationID] // parent depth and original membership are already checked
			scope = r.OwningInvocationID
			set(scope, resolve(p.runs[v.Caller]))
		}
		p.scopeEnd[r.ID] = ends[scope]
		return ends[scope]
	}
	for _, r := range p.snap.Runs {
		resolve(r)
	}
	return nil
}

// Match a refusal against what was present when the query finished. A read
// ordinal additionally proves the target's local dispatch drain at that prefix.
func (p *effectPrefix) checkReadRefusal(e *ExecutionEffect, order int64) error {
	r := e.AdapterResult
	if p.closed[e.Source] || p.retired[e.Source] || p.scopeClosed(e.Source, order) {
		return corrupt("receipt refusal after resolver closure")
	}
	if r.Disposition == "read_exhausted" {
		return nil // the two separate spent ordinals and successor epoch were checked above
	}
	a := e.AdapterAttempt
	target := a.Prepared.Request.Payload["target"].Text
	g, grant := p.readGrants[readGrantKey{e.Source, target}]
	run, _ := g.target(p.snap.Namespace)
	wantEndpoint, _ := receiptquery.LookupEndpoint(g.Endpoint)
	i := p.intents[target]
	original := p.effects[target]
	auth, res := p.order(compositionAuthorizationFact, target), p.order(compositionReservationFact, target)
	_, active := p.seq[run]
	reason := ""
	switch {
	case !grant || g.Invocation != "" && !active:
		reason = "read_denied"
	case wantEndpoint != a.Prepared.Endpoint:
		reason = "endpoint_mismatch"
	case p.membership[run] == "":
		reason = "unscoped"
	case i.ID == "" || p.seq[run] < i.Sequence:
		reason = "target_missing"
	case i.Run != run || i.Sequence != g.Sequence || i.Ordinal != 0:
		reason = "target_mismatch"
	case auth == 0 || auth >= order || original != nil && original.Authorization == nil:
		reason = "attempt_missing"
	case original == nil:
		if !p.authorizations[target].Allowed {
			reason = "original_denied"
			break
		}
		reason = "unsupported_original"
	case !original.Authorization.Allowed:
		reason = "original_denied"
	case res == 0 || res >= order:
		reason = "attempt_missing"
	case !p.reservations[target].Allowed:
		reason = "original_denied"
	case original.AdapterAttempt != nil && original.Capability != receiptCounterCapability:
		reason = "unsupported_original"
	case original.AdapterAttempt != nil && original.AdapterAttempt.Prepared.Endpoint != g.Endpoint || original.ExternalAttempt != nil && original.Capability != g.Endpoint:
		reason = "endpoint_mismatch"
	case !p.closed[run]:
		reason = "original_open"
	}
	if reason != "" {
		if r.Disposition != reason || r.RefusalWitness != "" {
			return corrupt("receipt refusal differs from its historical prerequisite")
		}
		return nil
	}
	w := p.effects[r.RefusalWitness]
	wRes := p.order(compositionReservationFact, r.RefusalWitness)
	if w == nil || w.Source != run || wRes == 0 || wRes >= order {
		return corrupt("receipt refusal lacks original dispatch witness")
	}
	switch r.Disposition {
	case "original_busy":
		if !p.reservations[w.OperationID].Allowed || p.latestRead[run] != 0 {
			return corrupt("active witness follows original drain")
		}
	case "original_unsettled":
		result := p.resultOrder[w.OperationID]
		if result == 0 || result >= order || p.latestRead[run] >= result {
			return corrupt("unsettled witness before original result or after drain")
		}
		if w.Settlement != nil {
			o := p.observations[w.Settlement.ObservationID]
			if o.SelectedAt != nil && o.SelectedAt.Sequence <= p.seq[run] {
				return corrupt("unsettled witness after consumption")
			}
		}
	default:
		return corrupt("unexplained local receipt refusal")
	}
	return nil
}
