package executor

import "strings"

// Check success joins at the original publication cut. A failed child can
// publish while its own descendants or external operations remain unresolved.
// A later receipt or admission must never justify an earlier ancestor success.
func (b *inspectionBuilder) checkPublicationPrefixes(roots map[string]*RootCompletion, freezes map[string]compositionFreeze, recordedFreezes map[string]bool) error {
	outstanding := map[string]int{}
	change := func(run RunID, delta int) {
		r := b.runs[run]
		for r.OwningInvocationID != "" {
			outstanding[r.OwningInvocationID] += delta
			r = b.runs[b.invocations[r.OriginOperationID].Caller]
		}
		outstanding["root:"+r.RootID] += delta
	}
	for _, row := range b.tables["composition_facts"] {
		f := b.facts[inspectText(row, "id")]
		op := f.OperationID
		key := compositionIdentity("alan/fact/"+string(f.Kind)+"/v1", op)
		switch f.Kind {
		case compositionAuthorityAdmissionFact:
			if !b.authorityValidated || f.AuthorityAdmission == nil {
				return ErrCorrupt
			}
			key = compositionIdentity("alan/fact/authority-admission/v1", []any{b.snap.Namespace, op, f.AuthorityAdmission.Ordinal})
		case compositionExecutionFact:
			key = compositionIdentity("alan/fact/execution/v1", []any{f.Run, f.Sequence})
			step := b.steps[ReductionRef{f.Run, f.Sequence}]
			if step == nil || op != "" {
				return corrupt("ordered reduction lacks its original")
			}
			for _, id := range step.OperationIDs {
				if b.invocations[id] != nil || b.effects[id] != nil {
					change(f.Run, 1)
				}
			}
		case compositionReadFact:
			key = compositionIdentity("alan/fact/receipt-read/v1", []any{op, f.Sequence})
		case compositionAdmissionFact:
			if !b.invocations[op].Admission.Accepted {
				change(f.Run, -1)
			}
		case compositionReservationFact:
			if e := b.effects[op]; e.ExternalAttempt != nil && !e.ExternalAttempt.Allowed {
				change(f.Run, -1)
			}
		case compositionExternalResultFact:
			change(f.Run, -1)
		case compositionFreezeFact:
			if strings.HasPrefix(op, "root:") && !recordedFreezes[op] {
				return corrupt("root freeze fact without retained source")
			}
		case compositionOutcomeFact:
			scope := op
			var successJoin bool
			if rootID, ok := strings.CutPrefix(op, "root:"); ok {
				root := roots[rootID]
				if root == nil || root.Outcome == nil || root.Outcome.DecisionOrder != f.Order {
					return corrupt("root outcome fact without publication")
				}
				successJoin = freezes[scope].Status == "succeeded"
			} else {
				v := b.invocations[op]
				if v == nil || v.Outcome == nil || v.Outcome.DecisionOrder != f.Order {
					return corrupt("outcome fact without publication")
				}
				if v.Admission.Accepted {
					scope = v.Admission.Invocation.ID
					successJoin = v.Freeze != nil && v.Freeze.Status == "succeeded" && v.Closure == nil
					change(v.Caller, -1)
				}
			}
			if successJoin && outstanding[scope] != 0 {
				return corrupt("successful publication with unresolved original dependencies")
			}
		case compositionAuthorizationFact, compositionLifecycleFact, compositionReplyFact, compositionContinuationFact:
			// These facts and their exact source bytes were checked with their owning protocol.
		default:
			return corrupt("unknown ordered fact kind")
		}
		if key != f.ID {
			return corrupt("ordered fact identity")
		}
	}
	return nil
}
