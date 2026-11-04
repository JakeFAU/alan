package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
)

type rootIdentity struct {
	Position int            `json:"position"`
	Run      aruntime.RunID `json:"run"`
	RootID   string         `json:"root_id"`
}
type rootCut struct {
	RootID      string                      `json:"root_id"`
	Run         aruntime.RunID              `json:"run"`
	LocalStatus string                      `json:"local_status"`
	Outcome     *aruntime.InvocationOutcome `json:"outcome"`
	Effects     int64                       `json:"effect_charges"`
	Invocations int64                       `json:"invocation_charges"`
	CreatedRuns int64                       `json:"created_run_charges"`
}
type cutProof struct {
	RootMembershipSHA256 string  `json:"root_membership_sha256"`
	RootCount            int     `json:"root_count"`
	Completed            int     `json:"completed"`
	SelectedPositions    int     `json:"selected_positions"`
	PriorCompleted       int     `json:"prior_completed"`
	FutureOpen           int     `json:"future_open"`
	Quiescent            bool    `json:"quiescent"`
	Steps                int     `json:"steps"`
	Dispatches           int     `json:"dispatches"`
	OpenRuns             int     `json:"open_runs"`
	UnresolvedIntents    int     `json:"unresolved_intents"`
	PendingDeliveries    int     `json:"pending_deliveries"`
	UnknownExternal      int     `json:"unknown_external"`
	UnknownAdapters      int     `json:"unknown_adapters"`
	RejectedAdmissions   int     `json:"rejected_admissions"`
	NewRoot              rootCut `json:"new_root"`
}
type cutVerifier struct {
	membership    []rootIdentity
	membershipSHA string
	completed     map[aruntime.RunID]aruntime.RootCompletion
}

func membershipDigest(roots []rootIdentity) string {
	raw, _ := json.Marshal(roots)
	return digestBytes(raw)
}
func newCutVerifier(roots []aruntime.RootCompletion, bound int) (*cutVerifier, error) {
	if len(roots) != bound {
		return nil, errors.New("initial root membership count mismatch")
	}
	byRun := make(map[aruntime.RunID]aruntime.RootCompletion, bound)
	ids := make(map[string]bool, bound)
	for _, root := range roots {
		if root.RootID == "" || ids[root.RootID] {
			return nil, errors.New("initial duplicate/missing root identity")
		}
		if _, exists := byRun[root.Run]; exists {
			return nil, errors.New("initial duplicate run identity")
		}
		ids[root.RootID], byRun[root.Run] = true, root
	}
	v := &cutVerifier{membership: make([]rootIdentity, 0, bound), completed: make(map[aruntime.RunID]aruntime.RootCompletion, bound)}
	for i := 1; i <= bound; i++ {
		run := aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1}
		root, ok := byRun[run]
		if !ok {
			return nil, errors.New("initial foreign/missing/wrong-generation root")
		}
		if err := verifyRootState(root, false); err != nil {
			return nil, err
		}
		v.membership = append(v.membership, rootIdentity{Position: i, Run: run, RootID: root.RootID})
	}
	v.membershipSHA = membershipDigest(v.membership)
	return v, nil
}
func verifyRootState(root aruntime.RootCompletion, complete bool) error {
	limits := aruntime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}
	if root.Limits != limits || root.Charges.RootID != root.RootID || root.Charges.Invocations != 0 || root.Charges.CreatedRuns != 0 ||
		len(root.PendingInvocations) != 0 || len(root.UnresolvedExternal) != 0 || len(root.UnknownExternal) != 0 {
		return errors.New("root allowance/charge/uncertainty mismatch")
	}
	if !complete {
		if root.LocalStatus != "open" || root.Pending != "members" || root.Outcome != nil || root.Charges.Effects != 0 {
			return errors.New("future root is not unstarted/open")
		}
		return nil
	}
	outcome := root.Outcome
	if root.LocalStatus != "succeeded" || root.Pending != "" || root.Charges.Effects != 1 || outcome == nil {
		return errors.New("completed root lacks success/outcome/debit")
	}
	scope := "root:" + root.RootID
	if outcome.ID == "" || outcome.Caller != root.Run || outcome.InvocationID != scope || outcome.OperationID != scope ||
		outcome.TemplateID != "" || outcome.Status != "succeeded" || outcome.Reason != "member_succeeded" ||
		outcome.ResponsePresent || len(outcome.Response) != 0 || outcome.DecisionOrder <= 0 {
		return errors.New("completed root original outcome mismatch")
	}
	return nil
}

// Check the already returned report only: no new inspection, Replay or SQL.
// For this frozen request/await/success program, a succeeded original root
// entails consumption/selection of its sole adapter result. This implication
// does not generalize to other programs or to Quiescent alone.
func (v *cutVerifier) verify(rr aruntime.RunReport, k int, ordinary bool) (cutProof, error) {
	bound := len(v.membership)
	if k < 1 || k > bound || len(rr.Roots) != bound || !rr.Quiescent {
		return cutProof{}, errors.New("invalid/quiescent cut cardinality")
	}
	if ordinary && len(v.completed) != k-1 || !ordinary && len(v.completed) != k {
		return cutProof{}, errors.New("cut does not advance exactly one original position")
	}
	wantSteps, wantDispatches := 0, 0
	if ordinary {
		wantSteps, wantDispatches = 4, 1
		if k == 1 {
			wantSteps = 3
		}
	}
	if rr.Steps != wantSteps || rr.Dispatches != wantDispatches || rr.RejectedAdmissions != 0 ||
		len(rr.UnresolvedIntents) != 0 || len(rr.PendingDeliveries) != 0 || len(rr.UnknownExternal) != 0 || len(rr.UnknownAdapters) != 0 {
		return cutProof{}, errors.New("finite ordinary report work/uncertainty mismatch")
	}
	byRun := make(map[aruntime.RunID]aruntime.RootCompletion, bound)
	roots := make(map[string]bool, bound)
	for _, root := range rr.Roots {
		if _, exists := byRun[root.Run]; exists || root.RootID == "" || roots[root.RootID] {
			return cutProof{}, errors.New("duplicate/missing report root")
		}
		byRun[root.Run], roots[root.RootID] = root, true
	}
	complete := 0
	originals := make(map[string]bool, k)
	for _, identity := range v.membership {
		root, ok := byRun[identity.Run]
		if !ok || root.RootID != identity.RootID {
			return cutProof{}, errors.New("foreign/missing/wrong-generation report root")
		}
		succeeded := root.LocalStatus == "succeeded"
		if succeeded {
			complete++
		}
		if err := verifyRootState(root, identity.Position <= k); err != nil {
			return cutProof{}, fmt.Errorf("position %d: %w", identity.Position, err)
		}
		if original, exists := v.completed[identity.Run]; exists && !reflect.DeepEqual(root, original) {
			return cutProof{}, errors.New("prior root original evidence changed")
		}
		if identity.Position <= k {
			if originals[root.Outcome.ID] {
				return cutProof{}, errors.New("duplicate original root outcome")
			}
			originals[root.Outcome.ID] = true
		}
	}
	if complete != k {
		return cutProof{}, errors.New("completed count not established by returned roots")
	}
	open := make(map[aruntime.RunID]bool, bound-k)
	for _, run := range rr.OpenRuns {
		if open[run] {
			return cutProof{}, errors.New("duplicate open run")
		}
		open[run] = true
	}
	if len(open) != bound-k {
		return cutProof{}, errors.New("open run cardinality mismatch")
	}
	for _, identity := range v.membership {
		if open[identity.Run] != (identity.Position > k) {
			return cutProof{}, errors.New("open run membership mismatch")
		}
	}
	for _, identity := range v.membership {
		if identity.Position <= k {
			if _, exists := v.completed[identity.Run]; !exists {
				v.completed[identity.Run] = effects.Copy(byRun[identity.Run])
			}
		}
	}
	root := byRun[v.membership[k-1].Run]
	return cutProof{RootMembershipSHA256: v.membershipSHA, RootCount: bound, Completed: complete, SelectedPositions: complete,
		PriorCompleted: complete - 1, FutureOpen: bound - complete, Quiescent: rr.Quiescent, Steps: rr.Steps, Dispatches: rr.Dispatches, OpenRuns: len(rr.OpenRuns),
		UnresolvedIntents: len(rr.UnresolvedIntents), PendingDeliveries: len(rr.PendingDeliveries), UnknownExternal: len(rr.UnknownExternal), UnknownAdapters: len(rr.UnknownAdapters), RejectedAdmissions: rr.RejectedAdmissions,
		NewRoot: rootCut{RootID: root.RootID, Run: root.Run, LocalStatus: root.LocalStatus, Outcome: effects.Copy(root.Outcome), Effects: root.Charges.Effects, Invocations: root.Charges.Invocations, CreatedRuns: root.Charges.CreatedRuns}}, nil
}
