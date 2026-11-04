package main

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
)

type evidence struct {
	RootOutcome    *aruntime.InvocationOutcome `json:"root_outcome"`
	Prepared       effects.Prepared            `json:"prepared"`
	Candidate      effects.Candidate           `json:"candidate"`
	Index          int                         `json:"index"`
	Run            aruntime.RunID              `json:"run"`
	RootID         string                      `json:"root_id"`
	OperationID    string                      `json:"operation_id"`
	PreparedDigest string                      `json:"prepared_digest"`
	ResultDigest   string                      `json:"result_digest"`
	ResponseID     string                      `json:"response_id"`
	ObservationID  string                      `json:"observation_id"`
	SelectedAt     *aruntime.ReductionRef      `json:"selected_at"`
	Approval       *aruntime.ContinuationView  `json:"approval,omitempty"`
}

func verifySelected(cut *aruntime.ExecutionSnapshot, v author.SeriesView, m *meter, n int) ([]evidence, error) {
	fail := func(f string, a ...any) ([]evidence, error) { return nil, fmt.Errorf(f, a...) }
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(cut.Effects) != n || len(cut.Continuations) != n-1 || len(m.contacts) != n || (m.mode == "http" && m.serverCount() != n) {
		return fail("counts runs=%d roots=%d effects=%d approvals=%d contacts=%d want=%d", len(cut.Runs), len(cut.Roots), len(cut.Effects), len(cut.Continuations), len(m.contacts), n)
	}
	if v.Bound != n || len(v.Positions) != n || v.Completed != n || v.Selected != n || v.Spent != n || v.Failed != 0 || v.Pending != 0 || v.Unresolved != 0 || !v.BoundExhausted {
		return fail("series aggregate mismatch: %+v", v)
	}
	ids, keys, observations, results, roots := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	out := make([]evidence, 0, n)
	for j, p := range v.Positions {
		want := aruntime.RunID{Subject: fmt.Sprintf("series-%d", j+1), Generation: 1}
		if p.Index != j+1 || p.Run == nil || p.Run.ID != want || p.Run.Projection.Status != "succeeded" || p.State != author.SeriesCompleted || p.Root == nil || p.Root.Run != want || p.Root.Charges.Effects != 1 || p.Root.Charges.Invocations != 0 || p.Root.Charges.CreatedRuns != 0 || p.Root.Outcome == nil {
			return fail("position %d run/root/charge/state mismatch", j+1)
		}
		e, o := p.Effect, p.Observation
		if e == nil || e.Source != want || e.OperationID == "" || ids[e.OperationID] || len(p.Run.EffectIDs) != 1 || p.Run.EffectIDs[0] != e.OperationID || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.AdapterResult == nil || e.ResultDisposition != "validated" || e.State != aruntime.EffectSettled || e.Settlement == nil || e.DispatchAdmission == nil || e.Dispatch.Minimum != 1 || e.Dispatch.Maximum == nil || *e.Dispatch.Maximum != 1 {
			return fail("position %d effect identity/reservation/result mismatch", j+1)
		}
		a, res, c := e.AdapterAttempt, e.AdapterResult, m.contacts[j]
		if a.Prepared.Request.ID != e.OperationID || a.Prepared.Request.Key == "" || keys[a.Prepared.Request.Key] || res.Candidate.ID != e.OperationID || res.Candidate.Key != a.Prepared.Request.Key || res.Digest == "" || results[res.Digest] || c.Ordinal != j+1 || c.OperationID != e.OperationID || c.Key != a.Prepared.Request.Key || c.PreparedDigest != a.Prepared.Digest || c.ResponseID != res.Candidate.ResponseID {
			return fail("position %d adapter contact provenance mismatch", j+1)
		}
		if c.Capability != a.Prepared.Request.Capability || c.Effect != a.Prepared.Request.Effect || c.Protocol != a.Prepared.Protocol || c.Endpoint != a.Prepared.Endpoint {
			return fail("position %d adapter contact request mismatch", j+1)
		}
		if err := verifyTransport(a.Prepared, res.Candidate, c, m, j); err != nil {
			return fail("position %d transport proof: %v", j+1, err)
		}
		if o == nil || o.ID == "" || observations[o.ID] || o.SourceOperationID != e.OperationID || o.SelectedAt == nil || o.SelectedAt.Run != want || e.Settlement.ObservationID != o.ID {
			return fail("position %d selection provenance mismatch", j+1)
		}
		reduction, ok := cut.Reduction(*o.SelectedAt)
		if !ok || reduction.SelectedObservationID != o.ID {
			return fail("position %d missing selected reduction", j+1)
		}
		if j == 0 {
			if p.Approval != nil {
				return fail("first position unexpectedly has protected approval")
			}
		} else {
			prev := out[j-1]
			if p.Approval == nil || p.Approval.Approval.Target != want || p.Approval.Approval.PredecessorID != prev.OperationID || p.Approval.Approval.PreparedDigest != prev.PreparedDigest || p.Approval.Approval.ResultDigest != prev.ResultDigest || !p.Approval.Approval.Admitted || p.Approval.Approval.Reason != "operator" || p.Approval.Approval.ScopeDigest == "" || p.Approval.SelectedAt == nil || p.Approval.SelectedAt.Run != want || p.Approval.SuccessorOperationID != e.OperationID || p.Approval.SelectedAt.Sequence >= e.Sequence {
				return fail("position %d protected predecessor/selection causality mismatch", j+1)
			}
			approvalObs, ok := cut.Observation(p.Approval.ObservationID)
			if !ok || approvalObs.SelectedAt == nil || *approvalObs.SelectedAt != *p.Approval.SelectedAt {
				return fail("position %d approval selection mismatch", j+1)
			}
		}
		if roots[p.Root.RootID] {
			return fail("duplicate root")
		}
		ids[e.OperationID], keys[a.Prepared.Request.Key], observations[o.ID], results[res.Digest], roots[p.Root.RootID] = true, true, true, true, true
		out = append(out, evidence{RootOutcome: p.Root.Outcome, Prepared: a.Prepared, Candidate: res.Candidate, Index: j + 1, Run: want, RootID: p.Root.RootID, OperationID: e.OperationID, PreparedDigest: a.Prepared.Digest, ResultDigest: res.Digest, ResponseID: res.Candidate.ResponseID, ObservationID: o.ID, SelectedAt: o.SelectedAt, Approval: p.Approval})
	}
	return out, nil
}

func verifyPrefix(c cut, m *meter, selected int) error {
	v := c.View
	if v.Completed != selected || v.Selected != selected || v.Spent != selected || v.Failed != 0 || v.Unresolved != 0 || len(c.Snapshot.Effects) != selected || len(c.Snapshot.Continuations) != max(0, selected-1) {
		return fmt.Errorf("prefix accounting mismatch at %d", selected)
	}
	// The complete view remains unchanged in the report. Restrict this existing
	// final-position assertion to the already selected prefix only.
	v.Positions = v.Positions[:selected]
	v.Bound, v.Pending, v.BoundExhausted = selected, 0, true
	_, err := verifySelected(c.Snapshot, v, m, selected)
	return err
}
