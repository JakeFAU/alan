package main

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/runtime"
)

func (c *delegatedCut) checkRootsAndPending(info retainedDeploymentInfo, call *runtime.InvocationView) error {
	pending := slices.Clone(c.Pending)
	for _, run := range c.Runs {
		if w := run.Wait; w != nil {
			p := runtime.PendingObligation{Kind: "waiting_" + w.Kind, Run: run.ID, OperationID: w.OperationID, DependsOn: w.OperationID}
			if !w.Resolved {
				p.Kind = "unresolved_receive"
			} else if len(w.MatchingObservationIDs) != 0 {
				p.Kind, p.ObservationID = "matching_observation", w.MatchingObservationIDs[0]
			}
			if err := consumePending(&pending, p); err != nil {
				return err
			}
		}
	}
	var invocations, effects int64
	var descendants, unresolved, unknown []string
	if call != nil {
		if call.Admission != nil && call.Admission.Accepted {
			invocations = 1
		}
		id := call.OperationID
		p := runtime.PendingObligation{Run: weatherRun, OperationID: id, DependsOn: id}
		wantSelection := "unpublished"
		switch {
		case call.Admission == nil:
			p.Kind = "invocation_awaiting_admission"
			descendants = append(descendants, id)
		case call.Outcome == nil:
			p.Kind = "invocation_awaiting_outcome"
			if call.Admission.Accepted {
				descendants = append(descendants, id)
			}
		case call.SelectedAt == nil:
			wantSelection = "unconsumed"
			p.Kind, p.ObservationID, p.DependsOn = "published_unconsumed", call.OutcomeObservationID, call.OutcomeObservationID
		default:
			wantSelection = "selected"
		}
		if call.ParentSelection != wantSelection {
			return fmt.Errorf("delegated invocation parent selection differs from its original evidence")
		}
		if p.Kind != "" {
			if err := consumePending(&pending, p); err != nil {
				return err
			}
		}
	}
	if err := c.consumeInvocationDependencies(&pending, call, info.HelperRun); err != nil {
		return err
	}

	for _, e := range c.Effects {
		if e.AdapterAttempt != nil && e.AdapterAttempt.Allowed {
			effects++
		}
		if e.ResultDisposition == "" {
			unresolved = append(unresolved, e.OperationID)
			if e.State == runtime.EffectReservedUnknown {
				unknown = append(unknown, e.OperationID)
			}
		}
		if e.State != runtime.EffectSettled {
			p := runtime.PendingObligation{Kind: "effect_" + string(e.State), Run: e.Source, OperationID: e.OperationID, DependsOn: e.OperationID}
			if err := consumePending(&pending, p); err != nil {
				return err
			}
		}
	}

	if len(c.Roots) != 2 {
		return fmt.Errorf("expected the two original delegated root views")
	}
	slices.Sort(unresolved)
	slices.Sort(unknown)
	for _, id := range []runtime.RunID{outingRun, weatherRun} {
		run, _ := c.Run(id)
		pos := slices.IndexFunc(c.Roots, func(r runtime.RootCompletion) bool { return r.Run == id })
		if pos < 0 {
			return fmt.Errorf("missing root account for %s", id.Subject)
		}
		root := c.Roots[pos]
		spec := slices.IndexFunc(c.Composition.Roots, func(r runtime.ResourceRoot) bool { return r.Run == id })
		if spec < 0 || root.RootID == "" || root.RootID != run.RootID || root.LocalStatus != run.Projection.Status ||
			root.Limits != c.Composition.Roots[spec].Limits || root.Charges.RootID != root.RootID {
			return fmt.Errorf("delegated root %s differs from the original account", id.Subject)
		}
		var wantInvocations, wantEffects int64
		var wantDescendants, wantUnresolved, wantUnknown []string
		if id == weatherRun {
			wantInvocations, wantEffects = invocations, effects
			wantDescendants, wantUnresolved, wantUnknown = descendants, unresolved, unknown
			if (info.HelperRun != nil) != (invocations == 1) {
				return fmt.Errorf("delegated creation charge differs from its admitted member")
			}
		}
		if root.Charges.Invocations != wantInvocations || root.Charges.CreatedRuns != wantInvocations ||
			root.Charges.Effects != wantEffects || !slices.Equal(root.PendingInvocations, wantDescendants) ||
			!slices.Equal(root.UnresolvedExternal, wantUnresolved) || !slices.Equal(root.UnknownExternal, wantUnknown) {
			return fmt.Errorf("delegated root %s hides original charges or outstanding work", id.Subject)
		}
		if out := root.Outcome; out != nil {
			scope := "root:" + root.RootID
			if run.Projection.Status == "open" || out.Status != root.LocalStatus || out.Caller != id || out.InvocationID != scope ||
				out.OperationID != scope || out.TemplateID != "" || out.ID == "" || out.DecisionOrder <= 0 ||
				out.ResponsePresent || len(out.Response) != 0 || root.Pending != "" {
				return fmt.Errorf("delegated root %s publication differs from its terminal run", id.Subject)
			}
			continue
		}
		if run.Projection.Status == "open" {
			p := runtime.PendingObligation{Kind: "root_member", Run: id, DependsOnRun: &id}
			if root.Pending != "members" {
				return fmt.Errorf("delegated root %s omits its open member", id.Subject)
			}
			if err := consumePending(&pending, p); err != nil {
				return err
			}
		}
		// In this application a root can finish locally only after consuming
		// its required evidence. A terminal root waiting only to publish is
		// keyed by the root ID, even when an ordinary send still needs an ack.
		p := runtime.PendingObligation{Kind: "root_awaiting_publication", Run: id, DependsOn: root.RootID}
		if err := consumePending(&pending, p); err != nil {
			return err
		}
	}

	for _, op := range c.messages {
		var admission *runtime.AdmissionView
		for _, a := range c.Admissions {
			if a.SourceOperationID == op.ID {
				admission = a
				break
			}
		}
		if admission == nil {
			p := runtime.PendingObligation{Kind: "message_awaiting_delivery", Run: op.Run, OperationID: op.ID, DependsOn: op.ID}
			if err := consumePending(&pending, p); err != nil {
				return err
			}
		} else if op.Obligation != helperObligation {
			// Public inspection exposes acknowledged versus unacknowledged
			// ordinary delivery as debt. Validate that an outstanding ack
			// names its exact original admission.
			p := runtime.PendingObligation{Kind: "message_awaiting_acknowledgment", Run: op.Run, OperationID: op.ID, DependsOn: op.ID, AdmissionIDs: []string{admission.ID}}
			if i := slices.IndexFunc(pending, func(v runtime.PendingObligation) bool { return samePending(v, p) }); i >= 0 {
				pending = slices.Delete(pending, i, i+1)
			}
		}
	}
	if len(pending) != 0 {
		return fmt.Errorf("unexpected delegated pending obligation %q", pending[0].Kind)
	}
	for _, run := range c.Runs {
		if run.Projection.Status == "open" {
			continue
		}
		hasDebt := slices.ContainsFunc(c.Pending, func(p runtime.PendingObligation) bool { return p.Run == run.ID })
		if hasDebt != (run.LocalState == runtime.RunTerminalWithObligations) {
			return fmt.Errorf("delegated pending obligations disagree with terminal state of %s", run.ID.Subject)
		}
	}
	return nil
}

func (c *delegatedCut) consumeInvocationDependencies(pending *[]runtime.PendingObligation, call *runtime.InvocationView, helper *runtime.RunID) error {
	if call == nil {
		return nil
	}
	var expected []runtime.PendingObligation
	depend := func(kind, id string, member *runtime.RunID) {
		expected = append(expected, runtime.PendingObligation{
			Kind: kind, Run: weatherRun, OperationID: call.OperationID, DependsOn: id, DependsOnRun: member,
		})
	}
	if call.Admission != nil && call.Outcome == nil {
		switch {
		case helper != nil && call.Freeze == nil:
			depend("invocation_member", "", helper)
		case call.Freeze != nil:
			if len(call.Freeze.ReplyIntents) != 0 && len(call.Replies) == 0 {
				depend("invocation_reply_admission", call.Freeze.ReplyIntents[0], nil)
			}
			if call.Freeze.Status == "succeeded" {
				for _, e := range c.Effects {
					if e.ResultDisposition == "" {
						depend("invocation_external_outcome", e.OperationID, nil)
					}
				}
			}
		}
		if len(expected) == 0 {
			depend("invocation_publication", call.OperationID, nil)
		}
	}
	if !slices.EqualFunc(expected, call.Dependencies, samePending) {
		return fmt.Errorf("delegated invocation has different original dependencies")
	}
	for _, p := range expected {
		if err := consumePending(pending, p); err != nil {
			return err
		}
	}
	return nil
}

func samePending(a, b runtime.PendingObligation) bool {
	return a.Kind == b.Kind && a.Run == b.Run && a.OperationID == b.OperationID && a.ObservationID == b.ObservationID &&
		a.DependsOn == b.DependsOn && reflect.DeepEqual(a.DependsOnRun, b.DependsOnRun) && slices.Equal(a.AdmissionIDs, b.AdmissionIDs)
}

func consumePending(pending *[]runtime.PendingObligation, want runtime.PendingObligation) error {
	i := slices.IndexFunc(*pending, func(p runtime.PendingObligation) bool { return samePending(p, want) })
	if i < 0 {
		return fmt.Errorf("missing delegated pending obligation %q for %q", want.Kind, want.OperationID)
	}
	*pending = slices.Delete(*pending, i, i+1)
	return nil
}

func rootsPublished(roots []runtime.RootCompletion) bool {
	if len(roots) != 2 {
		return false
	}
	for _, r := range roots {
		if r.Outcome == nil || r.Pending != "" || len(r.PendingInvocations)+len(r.UnresolvedExternal)+len(r.UnknownExternal) != 0 {
			return false
		}
	}
	return true
}
