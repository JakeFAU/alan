package executor

import "context"

// Ordinary messaging has already validated admission bytes and selections.
// Index protected routes once, then place their source records in Replay's
// independent ordered execution and resource reconstruction.
func replayContinuations(ctx context.Context, q queryer, c compositionProvision, expected map[string]compositionFact, x *compositionPrefix) (*continuationIndex, error) {
	if len(c.Continuations) == 0 {
		return nil, nil
	}
	idx := indexContinuations(c.Continuations)
	rows, err := scanInspectionTable(ctx, q, "admissions", true)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		d, err := inspectionDecode[AdmissionDescriptor](row, "descriptor")
		if err != nil {
			return nil, err
		}
		r, err := inspectionDecode[AdmissionResult](row, "result")
		if err != nil {
			return nil, err
		}
		if err := idx.admission(d, r); err != nil {
			return nil, err
		}
	}
	for _, slot := range c.Continuations {
		w := idx.byID[slot.ApprovalID]
		if w.view != nil {
			f := continuationFact(slot, w.d, w.r, x.scopes[slot.Target].RootID)
			expected[f.ID] = f
		}
	}
	return &idx, nil
}

func (x *compositionPrefix) continuationDecision(ctx context.Context, q queryer, idx *continuationIndex, f compositionFact, gate map[string]bool) error {
	if idx == nil {
		return corrupt("continuation outside opted-in protocol")
	}
	w := idx.byID[f.OperationID]
	if w == nil || w.view == nil || w.slot.Target != f.Run || !x.active[f.Run] || x.states[f.Run].Status != "open" || x.retired[f.Run] || x.closed(f.Run) {
		return corrupt("continuation outside target authority")
	}
	id := w.view.Approval.PredecessorID
	if !x.reservations[id] {
		return corrupt("continuation before predecessor reservation")
	}
	a, err := loadAdapterAttempt(ctx, q, id)
	if err != nil {
		return err
	}
	run, err := loadRun(ctx, q, f.Run)
	if err != nil {
		return err
	}
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return err
	}
	var result string
	if x.known[id] {
		r, err := loadAdapterResult(ctx, q, id)
		if err != nil {
			return err
		}
		result = r.Digest
	}
	return w.decision(f, namespace, x.scopes[f.Run].RootID, run.Revision, &a, gate[id], result)
}
