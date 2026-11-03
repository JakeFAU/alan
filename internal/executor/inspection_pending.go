package executor

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

func (b *inspectionBuilder) finishInspection() error {
	if err := b.checkSelections(); err != nil {
		return err
	}
	s := b.snap
	s.byInvocationID = map[string]*InvocationView{}
	s.admissionIndex = map[string]*AdmissionView{}
	s.operationIndex = map[string]*Intent{}
	s.reductionIndex = b.steps
	for i := range s.Operations {
		op := &s.Operations[i]
		s.operationIndex[op.ID] = op
	}
	slices.SortFunc(s.Admissions, func(a, c *AdmissionView) int { return strings.Compare(a.ID, c.ID) })
	for _, a := range s.Admissions {
		s.admissionIndex[a.ID] = a
	}
	s.runIndex = b.runs
	s.invocationIndex = b.invocations
	s.effectIndex = b.effects
	s.observationIndex = b.observations
	for _, id := range sortedKeys(b.observations) {
		o := b.observations[id]
		s.Observations = append(s.Observations, o)
		r := b.runs[o.Run]
		if r == nil {
			return corrupt("observation target missing")
		}
		if w := r.Wait; w != nil && o.SelectedAt == nil && w.matches(o.Observation) {
			w.MatchingObservationIDs = append(w.MatchingObservationIDs, id)
		}
	}
	for _, r := range s.Runs {
		if w := r.Wait; w != nil {
			kind := "waiting_" + w.Kind
			switch {
			case !w.Resolved:
				kind = "unresolved_receive"
			case len(w.MatchingObservationIDs) > 0:
				kind = "matching_observation"
			}
			p := PendingObligation{Kind: kind, Run: r.ID, OperationID: w.OperationID, DependsOn: w.OperationID}
			if len(w.MatchingObservationIDs) > 0 {
				p.ObservationID = w.MatchingObservationIDs[0]
			}
			s.Pending = append(s.Pending, p)
		}
	}
	roots := make(map[string]*RootCompletion, len(s.Roots))
	for i := range s.Roots {
		r := &s.Roots[i]
		roots[r.RootID] = r
	}
	for _, id := range sortedKeys(b.invocations) {
		v := b.invocations[id]
		s.Invocations = append(s.Invocations, v)
		if v.Admission != nil {
			s.byInvocationID[v.Admission.Invocation.ID] = v
		}
		b.runs[v.Caller].InvocationIDs = append(b.runs[v.Caller].InvocationIDs, id)
		root := roots[b.membership[v.Caller]]
		if v.Outcome == nil {
			v.ParentSelection = "unpublished"
			if root != nil {
				root.PendingInvocations = append(root.PendingInvocations, id)
			}
			kind := "invocation_awaiting_outcome"
			if v.Admission == nil {
				kind = "invocation_awaiting_admission"
			}
			s.Pending = append(s.Pending, PendingObligation{Kind: kind, Run: v.Caller, OperationID: id, DependsOn: id})
		} else if v.SelectedAt != nil {
			v.ParentSelection = "selected"
		} else if !canConsume(b.runs[v.Caller]) {
			v.ParentSelection = "consumer_closed"
		} else {
			v.ParentSelection = "unconsumed"
			s.Pending = append(s.Pending, PendingObligation{Kind: "published_unconsumed", Run: v.Caller, OperationID: id, ObservationID: v.OutcomeObservationID, DependsOn: v.OutcomeObservationID})
		}
	}
	for _, id := range sortedKeys(b.effects) {
		e := b.effects[id]
		s.Effects = append(s.Effects, e)
		r := b.runs[e.Source]
		r.EffectIDs = append(r.EffectIDs, id)
		if e.Consequence == "unresolved" || e.Consequence == "conflicting_receipts" {
			r.UnresolvedConsequences = append(r.UnresolvedConsequences, id)
		}
		if e.State != "settled" {
			s.Pending = append(s.Pending, PendingObligation{Kind: "effect_" + string(e.State), Run: e.Source, OperationID: id, DependsOn: id})
		}
		if root := roots[b.membership[e.Source]]; root != nil && e.ResultDisposition == "" {
			root.UnresolvedExternal = append(root.UnresolvedExternal, id)
			if e.State == "reserved_unknown" {
				root.UnknownExternal = append(root.UnknownExternal, id)
			}
		}
	}
	type terminalRoute struct {
		source      RunID
		destination TerminalDestination
	}
	deliveries := map[terminalRoute][]string{}
	for _, a := range s.Admissions {
		if a.Domain == "terminal" {
			route := terminalRoute{a.Source, TerminalDestination{a.Target, a.Obligation}}
			deliveries[route] = append(deliveries[route], a.ID)
		}
	}
	for _, row := range b.tables["terminal_obligations"] {
		run := inspectionRun(row)
		if b.runs[run] == nil {
			return corrupt("terminal source missing")
		}
		dests, err := inspectionDecode[[]TerminalDestination](row, "destinations")
		if err != nil {
			return err
		}
		done, err := inspectionDecode[map[string]bool](row, "delivery_status")
		if err != nil {
			return err
		}
		for _, d := range dests {
			if !done[d.key()] {
				ob := PendingObligation{Kind: "terminal_delivery", Run: run, DependsOn: d.key()}
				if ids := deliveries[terminalRoute{run, d}]; len(ids) > 0 {
					ob.Kind = "terminal_awaiting_acknowledgment"
					ob.AdmissionIDs = ids
				}
				s.Pending = append(s.Pending, ob)
			}
		}
	}
	if err := b.finishRoots(roots); err != nil {
		return err
	}
	slices.SortFunc(s.Roots, func(a, c RootCompletion) int { return strings.Compare(a.RootID, c.RootID) })
	s.rootIndex = map[string]*RootCompletion{}
	for i := range s.Roots {
		r := &s.Roots[i]
		s.rootIndex[r.RootID] = r
	}
	b.otherObligations()
	slices.SortFunc(s.Pending, func(a, c PendingObligation) int {
		if n := runLess(a.Run, c.Run); n != 0 {
			return n
		}
		if n := strings.Compare(a.Kind, c.Kind); n != 0 {
			return n
		}
		if n := strings.Compare(a.OperationID, c.OperationID); n != 0 {
			return n
		}
		if n := strings.Compare(a.DependsOn, c.DependsOn); n != 0 {
			return n
		}
		if a.DependsOnRun != nil && c.DependsOnRun != nil {
			return runLess(*a.DependsOnRun, *c.DependsOnRun)
		}
		return strings.Compare(a.ObservationID, c.ObservationID)
	})
	return nil
}

func (b *inspectionBuilder) finishRoots(roots map[string]*RootCompletion) error {
	for _, row := range b.tables["composition_charges"] {
		c, err := inspectionDecode[compositionCharge](row, "descriptor")
		if err != nil {
			return err
		}
		i, ok := b.intents[c.OperationID]
		r := roots[c.RootID]
		if !ok || r == nil || c.OperationID != row["operation_id"] || c.RootID != row["root_id"] || c.Kind != row["kind"] || c.DecisionOrder != row["decision_order"] || b.membership[i.Run] != c.RootID {
			return corrupt("resource charge source")
		}
		switch c.Kind {
		case "invocation":
			v := b.invocations[c.OperationID]
			if v == nil || v.Admission == nil || !v.Admission.Accepted || c.Invocations != 1 || c.CreatedRuns != int64(len(v.Admission.Members)) || c.Effects != 0 {
				return corrupt("invalid invocation debit")
			}
		case "effect":
			e := b.effects[c.OperationID]
			if e == nil || c.Invocations != 0 || c.CreatedRuns != 0 || c.Effects != 1 || !(e.AdapterAttempt != nil && e.AdapterAttempt.Allowed || e.ExternalAttempt != nil && e.ExternalAttempt.Allowed) {
				return corrupt("invalid effect debit")
			}
		default:
			return corrupt("unknown resource charge")
		}
		r.Charges.Invocations += c.Invocations
		r.Charges.CreatedRuns += c.CreatedRuns
		r.Charges.Effects += c.Effects
		if r.Charges.Invocations > r.Limits.MaxInvocations || r.Charges.CreatedRuns > r.Limits.MaxCreatedRuns || r.Charges.Effects > r.Limits.MaxEffects {
			return corrupt("resource envelope exceeded")
		}
	}
	if b.snap.Composition == nil {
		return nil
	}
	static := make(map[string][]RunID)
	sources := make(map[string][]Intent)
	for _, r := range b.snap.Composition.Roots {
		static[r.ID] = r.Members
	}
	for _, op := range b.snap.Operations {
		r := b.runs[op.Run]
		if r.RootID != "" && r.OwningInvocationID == "" {
			sources[r.RootID] = append(sources[r.RootID], op)
		}
	}
	cut := causalCut{inspectionBuilder: b, lifecycle: b.lifecycle}
	recordedFreezes := map[string]bool{}
	freezes := map[string]compositionFreeze{}
	closures := map[string]bool{}
	for _, row := range b.tables["composition_freezes"] {
		scope := inspectText(row, "scope_id")
		if !strings.HasPrefix(scope, "root:") {
			continue
		}
		r := roots[strings.TrimPrefix(scope, "root:")]
		f, e := inspectionDecode[compositionFreeze](row, "descriptor")
		if e != nil || r == nil || scope != f.InvocationID || r.RootID != row["root_id"] || b.membership[f.Run] != r.RootID || !b.rootRowFact(compositionFreezeFact, scope, r.RootID, f.Run, f.Sequence, f.DecisionOrder, row) {
			return corrupt("root freeze identity")
		}
		if !cut.freezeBelongs(static[r.RootID], nil, f, sources[r.RootID], false) {
			return corrupt("root freeze source")
		}
		recordedFreezes[scope] = true
		freezes[scope] = f
	}
	for _, row := range b.tables["composition_closures"] {
		scope := inspectText(row, "scope_id")
		if !strings.HasPrefix(scope, "root:") {
			continue
		}
		c, e := inspectionDecode[compositionClosure](row, "descriptor")
		r := roots[strings.TrimPrefix(scope, "root:")]
		if e != nil || r == nil || c.RootID != r.RootID || c.Freeze.InvocationID != scope || row["root_id"] != c.RootID || row["decision_order"] != c.Freeze.DecisionOrder || c.Reason != "subject_retired" || c.Freeze.Status != "cancelled" || !cut.freezeBelongs(static[r.RootID], nil, c.Freeze, sources[r.RootID], true) {
			return corrupt("root closure source")
		}
		trigger := b.facts[c.Freeze.Trigger]
		if trigger.Order != c.Freeze.DecisionOrder || trigger.Kind != compositionLifecycleFact || trigger.OperationID != c.LifecycleOperationID {
			return corrupt("root closure")
		}
		f, ok := freezes[scope]
		if ok && (f.DecisionOrder >= c.Freeze.DecisionOrder || f.Status != "succeeded") {
			return corrupt("retirement replaced a prior failing root")
		}
		freezes[scope] = c.Freeze
		closures[scope] = true
	}
	for _, row := range b.tables["composition_root_publications"] {
		r := roots[inspectText(row, "root_id")]
		out, e := inspectionDecode[InvocationOutcome](row, "descriptor")
		scope := inspectText(row, "scope_id")
		if e != nil || r == nil || r.Outcome != nil || scope != "root:"+r.RootID || out.OperationID != scope || out.InvocationID != scope || out.Caller != r.Run || out.ID != compositionIdentity("alan/root-outcome/v1", r.RootID) || out.DecisionOrder != row["decision_order"] || !b.rootRowFact(compositionOutcomeFact, scope, r.RootID, r.Run, 0, out.DecisionOrder, row) {
			return corrupt("root publication identity")
		}
		f, ok := freezes[scope]
		reason := "member_" + f.Status
		if closures[scope] {
			reason = "subject_retired"
		}
		if !ok || f.DecisionOrder >= out.DecisionOrder || out.Status != f.Status || out.Reason != reason || out.TemplateID != "" || out.ResponsePresent || out.Response != nil {
			return corrupt("root publication differs from terminal freeze")
		}
		r.Outcome = &out
	}
	if err := b.checkPublicationPrefixes(roots, freezes, recordedFreezes); err != nil {
		return err
	}
	// prepareCompositionScope orders direct children, then their transitive
	// obligations, then the root's own effects. Reuse that precedence when a
	// cut has no publication yet, keeping the exact first missing operation.
	members := map[string][]string{}
	for _, id := range sortedKeys(b.intents) {
		r := b.runs[b.intents[id].Run]
		if r.OwningInvocationID != "" {
			members[r.OwningInvocationID] = append(members[r.OwningInvocationID], id)
		}
	}
	type blocker struct{ kind, id string }
	checked := map[string]blocker{}
	var dependency func(string) blocker
	dependency = func(id string) blocker {
		if d, ok := checked[id]; ok {
			return d
		}
		var d blocker
		if v := b.invocations[id]; v != nil {
			switch {
			case v.Admission == nil:
				d = blocker{"invocation_admission", id}
			case !v.Admission.Accepted:
			case v.Outcome == nil:
				d = blocker{"descendant_outcome", id}
			default:
				for _, op := range members[v.Admission.Invocation.ID] {
					if d = dependency(op); d.kind != "" {
						break
					}
				}
			}
		} else if e := b.effects[id]; e != nil && e.ResultDisposition == "" {
			d = blocker{"external_outcome", id}
		}
		checked[id] = d
		return d
	}
	for i := range b.snap.Roots {
		r := &b.snap.Roots[i]
		if r.Outcome != nil {
			continue
		}
		scope := "root:" + r.RootID
		ob := PendingObligation{Kind: "root_awaiting_publication", Run: r.Run, DependsOn: r.RootID}
		f, frozen := freezes[scope]
		switch {
		case !frozen:
			r.Pending = "members"
			open := false
			for _, id := range static[r.RootID] {
				switch b.runs[id].Projection.Status {
				case "open":
					open = true
					member := PendingObligation{Kind: "root_member", Run: r.Run, DependsOnRun: &id}
					b.snap.Pending = append(b.snap.Pending, member)
				case "succeeded":
				default:
					return corrupt("terminal static member without its scope freeze")
				}
			}
			if !open {
				return corrupt("successful static members without their scope freeze")
			}
		case f.Status == "succeeded":
			find := func(ops []string) {
				for _, op := range ops {
					if d := dependency(op); d.kind != "" {
						r.Pending, ob.DependsOn = d.kind, d.id
						return
					}
				}
			}
			find(f.InvocationIntents)
			if r.Pending == "" {
				find(f.ExternalIntents)
			}
		}
		b.snap.Pending = append(b.snap.Pending, ob)
	}
	return nil
}

func (b *inspectionBuilder) rootRowFact(kind compositionFactKind, scope, root string, run RunID, seq, order int64, row map[string]any) bool {
	f := b.facts[compositionIdentity("alan/fact/"+string(kind)+"/v1", scope)]
	return f.Kind == kind && f.Order == order && f.Run == run && f.Sequence == seq && f.OperationID == scope && reflect.DeepEqual(f.RootIDs, []string{root}) && f.RecordDigest == compositionIdentity("alan/"+string(kind)+"-row/v1", json.RawMessage(inspectText(row, "descriptor")))
}

func canConsume(r *ExecutionRun) bool {
	return r.Projection.Status == "open" && !r.Subject.Retired && r.ID.Generation == r.Subject.Generation
}

func (b *inspectionBuilder) otherObligations() {
	deliveries := map[string][]string{}
	for _, a := range b.snap.Admissions {
		if a.Domain == "message" {
			deliveries[a.SourceOperationID] = append(deliveries[a.SourceOperationID], a.ID)
		}
	}
	acknowledged := map[string]bool{}
	settled := map[string]bool{}
	for _, row := range b.tables["message_acknowledgments"] {
		acknowledged[inspectText(row, "intent_id")] = true
	}
	for _, row := range b.tables["composition_replies"] {
		acknowledged[inspectText(row, "operation_id")] = true
	}
	for _, row := range b.tables["settlements"] {
		settled[inspectText(row, "intent_id")] = true
	}
	opaque := map[RunID]bool{}
	for _, op := range b.snap.Operations {
		kind := ""
		switch op.Kind {
		case "message":
			if !acknowledged[op.ID] {
				kind = "message_awaiting_delivery"
				if len(deliveries[op.ID]) != 0 {
					kind = "message_awaiting_acknowledgment"
				}
			}
		case "lifecycle":
			if !settled[op.ID] {
				kind = "lifecycle_awaiting_settlement"
			}
		case "adapter", "external", "invocation":
		default:
			opaque[op.Run] = true
		}
		if kind != "" {
			b.snap.Pending = append(b.snap.Pending, PendingObligation{Kind: kind, Run: op.Run, OperationID: op.ID, DependsOn: op.ID, AdmissionIDs: deliveries[op.ID]})
		}
	}
	local := map[RunID]bool{}
	for _, p := range b.snap.Pending {
		local[p.Run] = true
	}
	for _, r := range b.snap.Runs {
		switch {
		case r.Projection.Status == "open":
			r.LocalState = RunOpen
		case local[r.ID]:
			r.LocalState = RunTerminalWithObligations
		case opaque[r.ID]:
			r.LocalState = RunTerminalWithUntrackedOperations
		default:
			r.LocalState = RunTerminalSettled
		}
	}
}
