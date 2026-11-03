package executor

import (
	"maps"
	"reflect"
	"slices"
)

// ContinuationView links one operator decision to its selected input and the
// distinct adapter request it actually unlocked. Both links can be pending.
type ContinuationView struct {
	Approval             ContinuationApproval
	DecisionOrder        int64
	ObservationID        string
	SelectedAt           *ReductionRef
	SuccessorOperationID string
}

// Continuation finds a committed approval for the original target generation.
func (s *ExecutionSnapshot) Continuation(target RunID) (*ContinuationView, bool) {
	i, ok := slices.BinarySearchFunc(s.Continuations, target, func(v *ContinuationView, id RunID) int {
		return runLess(v.Approval.Target, id)
	})
	if !ok {
		return nil, false
	}
	return s.Continuations[i], true
}

// ContinuationBasis returns the original paid operation's evidence at this cut.
// An existing decision always returns its historical basis, including after a
// later result. This read-only method does not approve or reserve an operation.
func (s *ExecutionSnapshot) ContinuationBasis(target RunID) (ContinuationBasis, error) {
	if v, ok := s.Continuation(target); ok {
		return v.Approval.ContinuationBasis, nil
	}
	var basis ContinuationBasis
	if s.Composition == nil {
		return basis, ErrVersion
	}
	for _, slot := range s.Composition.Continuations {
		if slot.Target != target {
			continue
		}
		r, ok := s.Run(slot.Predecessor)
		if !ok {
			break
		}
		for _, id := range r.EffectIDs {
			e, ok := s.Effect(id)
			if !ok || e.Name != slot.Effect || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed {
				continue
			}
			if basis.PredecessorID != "" {
				return ContinuationBasis{}, ErrConflict
			}
			basis = ContinuationBasis{Namespace: s.Namespace, Target: target, PredecessorID: id,
				PreparedDigest: e.AdapterAttempt.Prepared.Digest, Admitted: e.DispatchAdmission != nil}
			if e.AdapterResult != nil {
				basis.ResultDigest = e.AdapterResult.Digest
			}
		}
		if basis.PredecessorID != "" {
			return basis, nil
		}
	}
	return basis, ErrIneligible
}

type continuationWitness struct {
	slot  ContinuationSlot
	d     AdmissionDescriptor
	r     AdmissionResult
	view  *ContinuationView
	prior *AdapterAttempt
}

// The reserved input name, delivery name and retained decision must agree in
// both directions. A generic local input cannot stand in for an approval.
type continuationIndex struct {
	bySubject map[string]*continuationWitness
	byID      map[string]*continuationWitness
}

func indexContinuations(slots []ContinuationSlot) continuationIndex {
	x := continuationIndex{map[string]*continuationWitness{}, map[string]*continuationWitness{}}
	for _, slot := range slots {
		w := &continuationWitness{slot: slot}
		x.bySubject[slot.Target.Subject], x.byID[slot.ApprovalID] = w, w
	}
	return x
}

func (x continuationIndex) admission(d AdmissionDescriptor, r AdmissionResult) error {
	w := x.bySubject[d.Target]
	if d.Kind != "input" || (w == nil || w.slot.Obligation != d.Obligation) && x.byID[d.TransportID] == nil && x.byID[d.LogicalID] == nil {
		if d.Continuation != nil {
			return corrupt("unexpected continuation provenance")
		}
		return nil
	}
	a := d.Continuation
	if a == nil || w == nil || w.view != nil || d.Source != (RunID{"local", 1}) || d.TransportID != w.slot.ApprovalID ||
		d.LogicalID != d.TransportID || d.Obligation != w.slot.Obligation || d.Generation != w.slot.Target.Generation ||
		d.Operation != "" || d.IntentID != "" || !validContinuationReason(a.Reason) || a.Target != w.slot.Target ||
		!maps.Equal(d.Payload, continuationPayload(*a)) || !r.Accepted || r.Bound == nil || *r.Bound != w.slot.Target ||
		r.ObservationID != observationFor(d, w.slot.Target).ID {
		return corrupt("continuation approval and local input disagree")
	}
	w.d, w.r = d, r
	w.view = &ContinuationView{Approval: *a, ObservationID: r.ObservationID}
	return nil
}

// Independently reconstructed journal prefixes supply admitted and result.
// A negative historical claim never borrows a fact that was retained later.
func (w *continuationWitness) decision(f compositionFact, namespace, root string, revision int64, prior *AdapterAttempt, admitted bool, resultDigest string) error {
	if w.view == nil || w.view.DecisionOrder != 0 || prior == nil || !prior.Allowed {
		return corrupt("continuation lacks its original paid reservation or local approval")
	}
	a := w.view.Approval
	want := continuationFact(w.slot, w.d, w.r, root)
	want.Order = f.Order
	if !reflect.DeepEqual(want, f) || a.Namespace != namespace || a.TargetRevision != revision ||
		a.PredecessorID != prior.Intent.ID || prior.Intent.Run != w.slot.Predecessor || prior.Intent.Obligation != w.slot.Effect ||
		a.PreparedDigest != prior.Prepared.Digest || a.Admitted != admitted || a.ResultDigest != resultDigest ||
		a.ScopeDigest != continuationScope(prior.Prepared) {
		return corrupt("continuation disagrees with its historical authority or predecessor")
	}
	w.view.DecisionOrder, w.prior = f.Order, prior
	return nil
}

func (w *continuationWitness) reduction(ref ReductionRef, selected string, intents []Intent, revision int64) error {
	v := w.view
	if v != nil && selected == v.ObservationID {
		if v.DecisionOrder == 0 || v.SelectedAt != nil || ref.Run != v.Approval.Target || revision != v.Approval.TargetRevision {
			return corrupt("continuation selected outside its approved generation or decision")
		}
		v.SelectedAt = &ref
	}
	for _, i := range intents {
		if i.Kind == "message" {
			continue
		}
		if i.Kind != "adapter" || v == nil || w.prior == nil || v.SelectedAt == nil || v.SelectedAt.Sequence >= ref.Sequence ||
			v.SuccessorOperationID != "" || revision != v.Approval.TargetRevision || i.Run != w.slot.Target ||
			i.Obligation != w.slot.Effect || i.Target != w.prior.Intent.Target || i.Generation != w.prior.Intent.Generation || !reflect.DeepEqual(i.Payload, w.prior.Intent.Payload) {
			return corrupt("operation outside single selected continuation scope")
		}
		v.SuccessorOperationID = i.ID
	}
	return nil
}

func (w *continuationWitness) reservation(a *AdapterAttempt) error {
	if a == nil || w.view == nil || w.view.SuccessorOperationID != a.Intent.ID || w.view.Approval.ScopeDigest != continuationScope(a.Prepared) {
		return corrupt("continuation reservation differs from approved action")
	}
	return nil
}

func (p *effectPrefix) continuationDecision(f compositionFact) error {
	if p.continuations == nil {
		return corrupt("continuation without its protected deployment")
	}
	w := p.continuations.byID[f.OperationID]
	target := p.runs[f.Run]
	if w == nil || w.view == nil || target == nil || w.slot.Target != f.Run || p.closed[f.Run] || p.retired[f.Run] || p.scopeClosed(f.Run, f.Order) {
		return corrupt("continuation outside open target authority")
	}
	id := w.view.Approval.PredecessorID
	e := p.effects[id]
	paid := p.order(compositionReservationFact, id)
	if e == nil || paid == 0 || paid >= f.Order {
		return corrupt("continuation precedes its reservation")
	}
	var result string
	if end := p.resultOrder[id]; end > 0 && end < f.Order {
		if e.AdapterResult == nil {
			return corrupt("continuation result has no retained source")
		}
		result = e.AdapterResult.Digest
	}
	return w.decision(f, p.snap.Namespace, p.membership[f.Run], target.Revision, e.AdapterAttempt, e.DispatchAdmission != nil, result)
}

func (b *inspectionBuilder) finishContinuations() error {
	if b.continuations == nil {
		return nil
	}
	for _, slot := range b.snap.Composition.Continuations {
		w := b.continuations.byID[slot.ApprovalID]
		if w.view != nil {
			if w.view.DecisionOrder == 0 {
				return corrupt("accepted continuation has no decision")
			}
			b.snap.Continuations = append(b.snap.Continuations, w.view)
		}
	}
	slices.SortFunc(b.snap.Continuations, func(a, b *ContinuationView) int { return runLess(a.Approval.Target, b.Approval.Target) })
	return nil
}
