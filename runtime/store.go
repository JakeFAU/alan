package runtime

import (
	"context"

	"github.com/JakeFAU/alan/internal/executor"
)

// Run advances eligible local work, dispatches registered capabilities, joins
// in-flight calls and returns a full report. Steps can commit before an error.
// A quiescent run may await input or a receipt. Use InspectExecution to examine
// the waiting operation, then call Run again after its prerequisite is available.
func (s *Store) Run(ctx context.Context, options RunOptions) (RunReport, error) {
	report, err := s.owner.Run(ctx, executor.RunOptions(options))
	return RunReport(report), err
}

// RunProgress advances under the same budgets as Run and returns per-call
// counts and quiescence. Retained state remains available via InspectExecution.
// Counts include work committed before an error.
func (s *Store) RunProgress(ctx context.Context, options RunOptions) (RunProgress, error) {
	progress, err := s.owner.RunProgress(ctx, executor.RunOptions(options))
	return RunProgress(progress), err
}

// AdmitInput delivers an application event to a declared local input port.
// Reuse the same transport ID and content for redelivery. Check both the error
// and the returned admission decision. Call Run to consume an accepted event.
func (s *Store) AdmitInput(ctx context.Context, input Input) (AdmissionResult, error) {
	return s.owner.AdmitInput(ctx, executor.Input(input))
}

// Snapshot reads the current state and authority of one logical generation.
func (s *Store) Snapshot(ctx context.Context, id RunID) (RunState, error) {
	return s.owner.Snapshot(ctx, id)
}

// Namespace returns the persistent store identity. It is preserved on reopen.
func (s *Store) Namespace() string {
	if s.owner == nil {
		return ""
	}
	return s.owner.Namespace()
}

// Close cancels an active runner, waits for local calls and their accounting,
// and releases the database and owner lock. Repeated calls are safe.
func (s *Store) Close() error {
	if s.owner == nil {
		return nil
	}
	return s.owner.Close()
}

// ContinuationBasis reads the evidence needed to approve a protected
// successor. ErrIneligible means no eligible basis exists at this cut.
func (s *Store) ContinuationBasis(ctx context.Context, target RunID) (ContinuationBasis, error) {
	return s.owner.ContinuationBasis(ctx, target)
}

// ApproveContinuation admits one protected local input under the supplied
// predecessor basis. The operator chooses the reason. A stale basis requires
// inspection before another approval decision.
func (s *Store) ApproveContinuation(ctx context.Context, basis ContinuationBasis, reason string) (AdmissionResult, error) {
	return s.owner.ApproveContinuation(ctx, basis, reason)
}

// AwaitDispatchQuiescence waits for covered local dispatches and their
// accounting after revocation. Previously transmitted effects can remain unknown.
func (s *Store) AwaitDispatchQuiescence(ctx context.Context, target DispatchAuthority) error {
	return s.owner.AwaitDispatchQuiescence(ctx, target)
}

// Commit performs one reduction under the supplied activation authority,
// selecting the observation ID when required. Run schedules these automatically.
func (s *Store) Commit(ctx context.Context, authority Authority, selected string) (RunState, error) {
	return s.owner.Commit(ctx, authority, selected)
}

// AuthorizeIntent records or returns the original committed intent's source decision.
func (s *Store) AuthorizeIntent(ctx context.Context, id string) (Authorization, error) {
	return s.owner.AuthorizeIntent(ctx, id)
}

// DeliverMessage attempts delivery of a committed message under the given transport ID.
func (s *Store) DeliverMessage(ctx context.Context, intentID, transportID string) (AdmissionResult, error) {
	return s.owner.DeliverMessage(ctx, intentID, transportID)
}

// AcknowledgeMessage records the source acknowledgment for an admitted message delivery.
func (s *Store) AcknowledgeMessage(ctx context.Context, intentID, transportID string) error {
	return s.owner.AcknowledgeMessage(ctx, intentID, transportID)
}

// TerminalDestination identifies a retained terminal delivery route.
type TerminalDestination = executor.TerminalDestination

// DeliverTerminal attempts delivery along a previously retained terminal route.
func (s *Store) DeliverTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) (AdmissionResult, error) {
	return s.owner.DeliverTerminal(ctx, source, destination, transportID)
}

// AcknowledgeTerminal records the source acknowledgment for a terminal delivery.
func (s *Store) AcknowledgeTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) error {
	return s.owner.AcknowledgeTerminal(ctx, source, destination, transportID)
}

// DispatchExternal services the legacy synchronous external protocol. Composition
// stores require Run and return ErrDispatchMode for direct dispatch.
func (s *Store) DispatchExternal(ctx context.Context, id string) (bool, error) {
	return s.owner.DispatchExternal(ctx, id)
}

// LoadLifecycleRequest reads the original request for a committed lifecycle operation.
func (s *Store) LoadLifecycleRequest(ctx context.Context, id string) (LifecycleRequest, error) {
	return s.owner.LoadLifecycleRequest(ctx, id)
}

// DecideLifecycle records or returns the decision for an authorized lifecycle request.
func (s *Store) DecideLifecycle(ctx context.Context, request LifecycleRequest) (LifecycleResult, error) {
	return s.owner.DecideLifecycle(ctx, request)
}

// SettleLifecycle delivers the lifecycle decision to its original execution.
func (s *Store) SettleLifecycle(ctx context.Context, id, transport string) (AdmissionResult, error) {
	return s.owner.SettleLifecycle(ctx, id, transport)
}

// AdmitInvocation authorizes the committed source and records its original admission
// together with the child generations, bindings and resource charge.
func (s *Store) AdmitInvocation(ctx context.Context, operation string) (InvocationAdmission, error) {
	return s.owner.AdmitInvocation(ctx, operation)
}

// PrepareInvocation checks whether the original invocation is ready for publication.
func (s *Store) PrepareInvocation(ctx context.Context, operation string) (InvocationPreparation, error) {
	return s.owner.PrepareInvocation(ctx, operation)
}

// SettleInvocation settles an invocation's available original outcome.
func (s *Store) SettleInvocation(ctx context.Context, operation string) (AdmissionResult, error) {
	return s.owner.SettleInvocation(ctx, operation)
}

// PrepareRoot reports the structured root outcome or its outstanding prerequisite.
func (s *Store) PrepareRoot(ctx context.Context, rootID string) (InvocationPreparation, error) {
	return s.owner.PrepareRoot(ctx, rootID)
}

// SettleRoot publishes the root outcome when its composition join is known.
func (s *Store) SettleRoot(ctx context.Context, rootID string) (InvocationPreparation, error) {
	return s.owner.SettleRoot(ctx, rootID)
}
