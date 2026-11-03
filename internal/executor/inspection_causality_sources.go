package executor

import (
	"encoding/json"
	"reflect"
	"slices"
)

func (c *causalCut) executionOrder(run RunID, sequence int64) int64 {
	return c.facts[compositionIdentity("alan/fact/execution/v1", []any{run, sequence})].Order
}

// A scoped authorization names the source sequence that was current when the
// decision was made. Later closure, reactivation, or retirement must not
// reclassify an earlier authorized operation.
func (c *causalCut) checkAuthorizations() error {
	terminals := map[RunID]map[string]any{}
	for _, row := range c.tables["terminal_obligations"] {
		terminals[inspectionRun(row)] = row
	}
	for _, row := range c.tables["attempts"] {
		a := c.auth[inspectText(row, "intent_id")]
		i := c.intents[a.IntentID]
		decision := c.factOrder(compositionAuthorizationFact, i.ID)
		root := c.membership[i.Run]
		if root != "" {
			if _, err := c.protocolOrder(compositionAuthorizationFact, i.ID, root, i.Run, i.Sequence, row); err != nil {
				return err
			}
			before, next := c.executionOrder(i.Run, a.Source.Sequence), c.executionOrder(i.Run, a.Source.Sequence+1)
			if before == 0 || before >= decision || next != 0 && decision >= next {
				return corrupt("authorization outside retained source prefix")
			}
		} else if decision != 0 {
			return corrupt("unexpected unscoped authorization fact")
		}
		proj, ok := c.after[ReductionRef{i.Run, a.Source.Sequence}]
		if !ok || !c.epochAt(i.Run, a.Source.Sequence, a.Source.Epoch) || a.Source.Revision != c.runs[i.Run].Revision {
			return corrupt("authorization outside original source activation or revision")
		}
		if a.ClosureOperation != "" {
			t := terminals[i.Run]
			if t["operation_id"] != a.ClosureOperation || t["sequence"] != a.Source.Sequence || c.runs[i.Run].Projection.Status == "open" {
				return corrupt("authorization cited an unrelated closure")
			}
			if decision != 0 && c.factOrder(compositionLifecycleFact, a.ClosureOperation) >= decision {
				return corrupt("authorization precedes source closure")
			}
			proj = c.runs[i.Run].Projection
		}
		if proj.Status != "open" && a.Source.Epoch != c.runs[i.Run].Epoch {
			return corrupt("authorization epoch differs from closed source")
		}
		member := c.byID[c.runs[i.Run].OwningInvocationID]
		reply := member != nil && c.isInvocationReply(member, i.Intent)
		switch a.Reason {
		case "authorized":
			if a.Source.Revision != i.revision || (!reply && proj.Status != "open") {
				return corrupt("unsupported authorization")
			}
		case "source_closed":
			if proj.Status == "open" || reply {
				return corrupt("unsupported closure refusal")
			}
		case "source_revision_changed":
			if a.Source.Revision == i.revision {
				return corrupt("unsupported revision refusal")
			}
		case "source_generation_changed":
			if c.runs[i.Run].Subject.Generation == i.Run.Generation {
				return corrupt("unsupported generation refusal")
			}
		case "source_retired":
			if !c.runs[i.Run].Subject.Retired || reply {
				return corrupt("unsupported retirement refusal")
			}
		default:
			return corrupt("unknown authorization reason")
		}
	}
	for _, f := range c.facts {
		if f.Kind == compositionAuthorizationFact {
			if _, ok := c.auth[f.OperationID]; !ok {
				return corrupt("authorization fact lacks its original decision")
			}
		}
	}
	return nil
}

// readLifecycleSources indexes the original requests and their target decisions,
// including decisions that have not yet been settled to an observation.
func (c *causalCut) readLifecycleSources() error {
	ordered := make([]LifecycleResult, len(c.tables["lifecycle_results"]))
	for _, row := range c.tables["lifecycle_results"] {
		r, err := inspectionDecode[LifecycleResult](row, "result")
		if err != nil {
			return err
		}
		i, ok := c.intents[r.Request.Intent.ID]
		rec := LifecycleRequest{Intent: i.Intent, SourceRevision: i.revision, Operation: i.Payload["operation"].Text, ExpectedRevision: i.Payload["revision"].Number}
		if !ok || i.Kind != "lifecycle" || !c.auth[i.ID].Allowed || row["id"] != i.ID || row["decision_sequence"] != r.DecisionSequence || r.DecisionSequence <= 0 || r.DecisionSequence > int64(len(ordered)) ||
			!reflect.DeepEqual(rec, r.Request) || row["descriptor"] != compositionJSON(rec) || rec.ExpectedRevision <= 0 ||
			!slices.Contains([]string{"CancelRun", "AbandonRun", "RetireSubject", "Retry"}, rec.Operation) {
			return corrupt("lifecycle decision lacks its original authorized request")
		}
		target := RunID{i.Target, i.Generation}
		var sequence int64
		if r.Before != nil {
			target, sequence = r.Before.Run, r.Before.Sequence
			if run := c.runs[target]; target.Subject != i.Target || run == nil || r.After == nil || r.Before.ProgramID != run.ProgramID || sequence < 0 || sequence > run.Sequence || r.Before.Epoch < 1 || r.Before.Epoch > run.Epoch {
				return corrupt("lifecycle decision names an invalid target witness")
			}
		} else if r.Accepted || r.After != nil || r.Reason != "target_missing" {
			return corrupt("lifecycle decision missing target witness")
		}
		sourceRoot := c.membership[i.Run]
		targetRoot := ""
		if r.Before != nil {
			// A later invocation can create a once-missing target. Only its
			// original target witness can put this decision in the target root.
			targetRoot = c.membership[target]
		}
		roots := []string{}
		if sourceRoot != "" {
			roots = append(roots, sourceRoot)
		}
		if targetRoot != "" && targetRoot != sourceRoot {
			roots = append(roots, targetRoot)
		}
		slices.Sort(roots)
		id := compositionIdentity("alan/fact/lifecycle/v1", i.ID)
		f := c.facts[id]
		if len(roots) != 0 {
			parts := []json.RawMessage{json.RawMessage(inspectText(row, "descriptor")), json.RawMessage(inspectText(row, "result"))}
			if f.Kind != compositionLifecycleFact || f.OperationID != i.ID || f.Run != target || f.Sequence != sequence ||
				!slices.Equal(f.RootIDs, roots) || f.RecordDigest != compositionIdentity("alan/lifecycle-row/v1", parts) ||
				f.Order <= c.factOrder(compositionAuthorizationFact, i.ID) {
				return corrupt("lifecycle decision has a foreign source fact")
			}
		} else if f.ID != "" {
			return corrupt("unexpected unscoped lifecycle fact")
		}
		c.lifecycle[i.ID] = r
		ordered[r.DecisionSequence-1] = r
	}
	for _, f := range c.facts {
		if f.Kind == compositionLifecycleFact {
			if _, ok := c.lifecycle[f.OperationID]; !ok {
				return corrupt("lifecycle fact without original decision")
			}
		}
	}
	if err := c.checkLifecycleAuthority(ordered); err != nil {
		return err
	}
	c.snap.LifecycleDecisions = ordered
	c.snap.lifecycleIndex = c.lifecycle
	return nil
}

func (c *causalCut) checkTransportOrder(o *ObservationView) error {
	if o.SelectedAt == nil || c.membership[o.Run] == "" {
		return nil
	}
	selected := c.executionOrder(o.Run, o.SelectedAt.Sequence)
	if r := o.ProducerReduction; r != nil && c.executionOrder(r.Run, r.Sequence) >= selected {
		return corrupt("observation selected before emitting reduction")
	}
	if o.Origin == "message" || o.Origin == "lifecycle" {
		if c.factOrder(compositionAuthorizationFact, o.SourceOperationID) >= selected {
			return corrupt("observation selected before source authorization")
		}
	}
	if (o.Origin == "terminal" || o.Origin == "lifecycle") && o.SourceOperationID != "" {
		if c.factOrder(compositionLifecycleFact, o.SourceOperationID) >= selected {
			return corrupt("observation selected before lifecycle decision")
		}
	}
	return nil
}
