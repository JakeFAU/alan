package author

import (
	"fmt"
	"reflect"

	"github.com/JakeFAU/alan/runtime"
)

// SeriesState describes retained local evidence for one original observation.
type SeriesState string

const (
	SeriesNotEligible         SeriesState = "not_eligible"
	SeriesAwaitingRequest     SeriesState = "awaiting_request"
	SeriesAwaitingApproval    SeriesState = "awaiting_approval"
	SeriesApproved            SeriesState = "approved"
	SeriesCommitted           SeriesState = "committed"
	SeriesAuthorized          SeriesState = "authorized"
	SeriesSpentBeforeDispatch SeriesState = "spent_before_dispatch"
	SeriesOutcomeUnknown      SeriesState = "outcome_unknown"
	SeriesResultUnselected    SeriesState = "result_unselected"
	SeriesSelected            SeriesState = "selected"
	SeriesCompleted           SeriesState = "completed"
	SeriesFailed              SeriesState = "failed"
	SeriesRefused             SeriesState = "refused"
)

// SeriesPosition references the original typed evidence in the supplied snapshot.
// Effect is nil before commitment, Approval is nil before protected approval,
// and Observation is nil until the effect result is actually selected. A result
// or settlement alone does not establish selection. Root exposes original
// limits and charges; Effect.Usage preserves unknown provider usage as nil.
type SeriesPosition struct {
	Index       int
	Run         *runtime.ExecutionRun
	Effect      *runtime.ExecutionEffect
	Approval    *runtime.ContinuationView
	Observation *runtime.ObservationView
	Root        *runtime.RootCompletion
	State       SeriesState
}

// SeriesView is an ordered view of a complete finite deployment, not a parent
// invocation or aggregate root outcome. Completed counts selected validated
// observations whose successful run has settled its local obligations and whose
// root has an outcome. Failed counts known failed/refused positions. Pending
// counts all remaining positions, including selected results with local work
// pending. Unresolved independently counts original effects whose runtime
// consequence remains unresolved or has conflicting receipts; a local failure
// can remain externally unresolved. BoundExhausted means all independent
// opportunities were spent, even
// if some original operations remain unknown. None of these claims establishes
// remote finality. Selected is independent of terminal success.
type SeriesView struct {
	Namespace, ID                                           string
	Bound                                                   int
	Positions                                               []SeriesPosition
	Selected, Completed, Failed, Pending, Unresolved, Spent int
	BoundExhausted                                          bool
}

// Inspect checks the exact original deployment against this declaration and
// joins indexed runtime evidence without running policy or contacting adapters.
// Pass a snapshot returned by runtime.InspectExecution, whose retained history
// has already been validated. The result borrows that detached snapshot's data;
// callers must treat both as immutable while reading. This is not an adversarial
// validator for arbitrarily rewritten in-memory snapshots.
func (s Series) Inspect(cut *runtime.ExecutionSnapshot) (SeriesView, error) {
	fail := func(reason string) (SeriesView, error) { return SeriesView{}, fmt.Errorf("bounded series: %s", reason) }
	if s.declaration == nil {
		return fail("zero declaration")
	}
	if cut == nil {
		return fail("nil execution snapshot")
	}
	d := s.declaration
	p := d.provision
	if cut.Namespace == "" || cut.Deployment == nil || cut.Composition == nil || cut.Reconciliation != nil || len(cut.Assets) != 0 || len(cut.Invocations) != 0 || len(cut.Runs) != d.count || len(cut.Roots) != d.count || !cut.Features.DispatchAdmissions {
		return fail("deployment shape differs")
	}
	deployment := cut.Deployment
	if !seriesEqualSet(deployment.Subjects, p.Subjects, func(v runtime.Subject) string { return v.ID }) ||
		!seriesEqualSet(deployment.Runs, p.Runs, func(v runtime.InitialRun) runtime.RunID { return v.ID }) ||
		!seriesEqualSet(deployment.Permissions, p.Permissions, func(v runtime.Permission) runtime.Permission { return v }) ||
		!seriesEqualSet(cut.Permissions, p.Permissions, func(v runtime.Permission) runtime.Permission { return v }) {
		return fail("original subjects, runs, inputs or grants differ")
	}
	profile := cut.Composition
	if profile.Version != seriesProtocol || !profile.RecordDispatchAdmissions || len(profile.Templates) != 0 || len(profile.Roots) != d.count ||
		!seriesEqualSet(profile.Continuations, p.Composition.Continuations, func(v runtime.ContinuationSlot) runtime.RunID { return v.Target }) {
		return fail("resource profile or continuation slots differ")
	}
	roots := make(map[runtime.RunID]runtime.ResourceRoot, d.count)
	for _, r := range profile.Roots {
		if _, duplicate := roots[r.Run]; duplicate || r.ID == "" || r.Version != seriesResourceVersion {
			return fail("invalid root identity")
		}
		roots[r.Run] = r
	}
	view := SeriesView{Namespace: cut.Namespace, ID: d.id, Bound: d.count, Positions: make([]SeriesPosition, 0, d.count)}
	// Original initial runs were retained once, in deterministic application order.
	initial := make(map[runtime.RunID]runtime.InitialRun, d.count)
	for _, r := range p.Runs {
		initial[r.ID] = r
	}
	effects := 0
	for index := 1; index <= d.count; index++ {
		id, _ := s.Run(index)
		run, ok := cut.Run(id)
		if !ok || run == nil {
			return fail("missing original run")
		}
		root, ok := roots[id]
		if !ok {
			return fail("missing original root")
		}
		expected := seriesRoot(id, d.capability)
		expected.ID = root.ID
		expected.Version = seriesResourceVersion
		// Runtime normalizes empty root template sets to empty non-nil slices.
		if root.Run != expected.Run || len(root.Members) != 1 || root.Members[0] != id || len(root.Templates) != 0 || len(root.Capabilities) != 1 || root.Capabilities[0] != d.capability || root.Limits != expected.Limits {
			return fail("root membership, capability or limits differ")
		}
		accounting, ok := cut.Root(root.ID)
		if !ok || accounting == nil || accounting.Run != id || accounting.Limits != expected.Limits || accounting.Charges.Effects < 0 || accounting.Charges.Effects > 1 || accounting.Charges.Invocations != 0 || accounting.Charges.CreatedRuns != 0 {
			return fail("root accounting differs")
		}
		if run.ID != id || run.ProgramID != initial[id].ProgramID || !reflect.DeepEqual(run.Initial, initial[id].Projection) || run.RootID != root.ID || run.Origin != "root" || run.Predecessor != nil || run.OwningInvocationID != "" || len(run.InvocationIDs) != 0 || len(run.EffectIDs) > 1 {
			return fail("original run differs")
		}
		if run.Sequence > 0 {
			step, ok := cut.Reduction(runtime.ReductionRef{Run: id, Sequence: run.Sequence})
			if !ok || step.AfterPC != run.Projection.PC || run.Projection.Status == "succeeded" && step.Status != "succeeded" {
				return fail("current projection differs from retained reduction")
			}
		}
		position := SeriesPosition{Index: index, Run: run, Root: accounting}
		position.Approval, _ = cut.Continuation(id)
		if index == 1 && position.Approval != nil {
			return fail("unexpected first approval")
		}
		if len(run.EffectIDs) == 1 {
			position.Effect, ok = cut.Effect(run.EffectIDs[0])
			e := position.Effect
			if !ok || e == nil || e.OperationID != run.EffectIDs[0] || e.Source != id || e.Kind != "adapter" || e.Name != d.effect || e.Capability != d.capability {
				return fail("original effect differs")
			}
			effects++
			if index > 1 && (position.Approval == nil || position.Approval.SuccessorOperationID != e.OperationID) {
				return fail("effect lacks its protected approval")
			}
			spent := e.AdapterAttempt != nil && e.AdapterAttempt.Allowed
			if spent != (accounting.Charges.Effects == 1) {
				return fail("effect expenditure differs")
			}
			if e.Settlement != nil {
				observation, ok := cut.Observation(e.Settlement.ObservationID)
				if !ok || observation == nil || observation.Run != id || observation.SourceOperationID != e.OperationID || observation.Operation != e.OperationID || e.AdapterResult == nil || observation.Kind != "adapter" || observation.Provenance != "runtime-adapter" || observation.Source != d.capability || observation.SourceGeneration != 1 || observation.Obligation != d.effect || !reflect.DeepEqual(observation.Payload, e.AdapterResult.Values) {
					return fail("result settlement observation differs")
				}
				if observation.SelectedAt != nil {
					selected := observation.SelectedAt
					step, ok := cut.Reduction(*selected)
					if selected.Run != id || selected.Sequence > run.Sequence || !ok || step.SelectedObservationID != observation.ID {
						return fail("result selection differs")
					}
					position.Observation = observation
					view.Selected++
				}
			}
		} else if accounting.Charges.Effects != 0 {
			return fail("charge without original effect")
		}
		if run.Projection.Status == "succeeded" && position.Observation == nil {
			return fail("successful observation lacks selection")
		}
		previousSpent := index > 1 && view.Positions[index-2].Root.Charges.Effects == 1
		position.State = seriesPositionState(position, previousSpent)
		view.Spent += int(accounting.Charges.Effects)
		switch position.State {
		case SeriesCompleted:
			view.Completed++
		case SeriesFailed, SeriesRefused:
			view.Failed++
		default:
			view.Pending++
		}
		if e := position.Effect; e != nil && (e.Consequence == runtime.ConsequenceUnresolved || e.Consequence == runtime.ConsequenceConflictingReceipts) {
			view.Unresolved++
		}
		view.Positions = append(view.Positions, position)
	}
	if effects != len(cut.Effects) {
		return fail("unexpected effect")
	}
	view.BoundExhausted = view.Spent == view.Bound
	return view, nil
}

func seriesPositionState(p SeriesPosition, previousSpent bool) SeriesState {
	// A terminal local failure remains failed even when its original operation
	// is unresolved or its result was never selected. Effect evidence stays intact.
	if p.Run.Projection.Status != "open" && p.Run.Projection.Status != "succeeded" {
		return SeriesFailed
	}
	e := p.Effect
	if e == nil {
		if p.Index == 1 {
			return SeriesAwaitingRequest
		}
		if p.Approval != nil {
			return SeriesApproved
		}
		if previousSpent {
			return SeriesAwaitingApproval
		}
		return SeriesNotEligible
	}
	if e.AdapterResult != nil {
		if p.Observation == nil {
			return SeriesResultUnselected
		}
		if e.AdapterResult.Disposition != "validated" {
			if e.AdapterAttempt != nil && !e.AdapterAttempt.Allowed || e.Authorization != nil && !e.Authorization.Allowed {
				return SeriesRefused
			}
			return SeriesFailed
		}
		if p.Run.Projection.Status == "succeeded" && p.Run.LocalState == runtime.RunTerminalSettled && p.Root.Outcome != nil {
			return SeriesCompleted
		}
		return SeriesSelected
	}
	if e.AdapterAttempt != nil && e.AdapterAttempt.Allowed {
		if e.DispatchAdmission == nil {
			return SeriesSpentBeforeDispatch
		}
		return SeriesOutcomeUnknown
	}
	if e.State == runtime.EffectAuthorizationRefused || e.State == runtime.EffectReservationRefused {
		return SeriesRefused
	}
	if e.Authorization != nil && e.Authorization.Allowed {
		return SeriesAuthorized
	}
	return SeriesCommitted
}

func seriesEqualSet[T any, K comparable](a, b []T, key func(T) K) bool {
	if len(a) != len(b) {
		return false
	}
	expected := make(map[K]T, len(b))
	for _, v := range b {
		expected[key(v)] = v
	}
	for _, v := range a {
		k := key(v)
		want, ok := expected[k]
		if !ok || !reflect.DeepEqual(v, want) {
			return false
		}
		delete(expected, k)
	}
	return len(expected) == 0
}
