package runtime

import (
	"context"

	"github.com/JakeFAU/alan/internal/executor"
)

// InspectExecution reads one coherent cut of the retained computation without
// taking ownership, executing the reducer or contacting external services.
func InspectExecution(ctx context.Context, path string, limits Limits) (*ExecutionSnapshot, error) {
	cut, err := executor.InspectExecution(ctx, path, executor.Limits(limits))
	return (*ExecutionSnapshot)(cut), err
}

// ExecutionSnapshot contains detached facts and indexed accessors for one retained cut.
// Runs, Effects, Invocations, Continuations and Pending enumerate current evidence.
// Namespace and StorageVersion identify the store and its format. Obtain a cut
// from InspectExecution. Treat it as immutable when sharing it between readers.
type ExecutionSnapshot executor.ExecutionSnapshot

// Run looks up a logical generation, its current projection and receive wait.
func (s *ExecutionSnapshot) Run(id RunID) (*ExecutionRun, bool) {
	return (*executor.ExecutionSnapshot)(s).Run(id)
}

// Invocation looks up a child invocation by the original handle returned to the caller.
func (s *ExecutionSnapshot) Invocation(operationID string) (*InvocationView, bool) {
	return (*executor.ExecutionSnapshot)(s).Invocation(operationID)
}

// InvocationByID looks up a child by its namespace-bound admission identity.
func (s *ExecutionSnapshot) InvocationByID(id string) (*InvocationView, bool) {
	return (*executor.ExecutionSnapshot)(s).InvocationByID(id)
}

// Effect looks up the authority, reservation, original result and settlement of an effect.
func (s *ExecutionSnapshot) Effect(operationID string) (*ExecutionEffect, bool) {
	return (*executor.ExecutionSnapshot)(s).Effect(operationID)
}

// Observation looks up an admitted logical observation and its selection, if any.
func (s *ExecutionSnapshot) Observation(id string) (*ObservationView, bool) {
	return (*executor.ExecutionSnapshot)(s).Observation(id)
}

// Asset returns a retained document with its content-bound version.
func (s *ExecutionSnapshot) Asset(id string) (*Asset, bool) {
	return (*executor.ExecutionSnapshot)(s).Asset(id)
}

// Operation looks up an original operation emitted by a committed reduction.
func (s *ExecutionSnapshot) Operation(id string) (*Intent, bool) {
	return (*executor.ExecutionSnapshot)(s).Operation(id)
}

// Reduction looks up a committed step by logical generation and sequence.
func (s *ExecutionSnapshot) Reduction(ref ReductionRef) (*ExecutionStep, bool) {
	return (*executor.ExecutionSnapshot)(s).Reduction(ref)
}

// Admission looks up a single delivery attempt by its admission identity.
func (s *ExecutionSnapshot) Admission(id string) (*AdmissionView, bool) {
	return (*executor.ExecutionSnapshot)(s).Admission(id)
}

// Root looks up the structured publication and resource accounting of a composition root.
func (s *ExecutionSnapshot) Root(id string) (*RootCompletion, bool) {
	return (*executor.ExecutionSnapshot)(s).Root(id)
}

// LifecycleDecision returns the original lifecycle decision at its recorded boundary.
func (s *ExecutionSnapshot) LifecycleDecision(operationID string) (LifecycleResult, bool) {
	return (*executor.ExecutionSnapshot)(s).LifecycleDecision(operationID)
}

// Continuation finds a committed approval for the target generation.
func (s *ExecutionSnapshot) Continuation(target RunID) (*ContinuationView, bool) {
	return (*executor.ExecutionSnapshot)(s).Continuation(target)
}

// ContinuationBasis returns the predecessor's paid evidence at this cut, or the historical
// basis of an existing approval. ErrIneligible means there is no eligible basis.
func (s *ExecutionSnapshot) ContinuationBasis(target RunID) (ContinuationBasis, error) {
	return (*executor.ExecutionSnapshot)(s).ContinuationBasis(target)
}

// History returns up to limit committed steps in append order. Start at offset zero
// and advance by HistoryPage.Next while More is true. Limit must be between one and MaxHistoryPage inclusive.
func (s *ExecutionSnapshot) History(offset, limit int) (HistoryPage, error) {
	return (*executor.ExecutionSnapshot)(s).History(offset, limit)
}

// InspectionFeatures identifies the optional protocols present in this format.
type InspectionFeatures = executor.InspectionFeatures

// ExecutionDeployment records original static subjects, runs and grants.
type ExecutionDeployment = executor.ExecutionDeployment
type ExecutionRun = executor.ExecutionRun
type ExecutionStep = executor.ExecutionStep
type ReductionRef = executor.ReductionRef
type HistoryPage = executor.HistoryPage
type ReceiveWait = executor.ReceiveWait
type PendingObligation = executor.PendingObligation
type InvocationView = executor.InvocationView
type InvocationClosure = executor.InvocationClosure
type InvocationReply = executor.InvocationReply
type ObservationView = executor.ObservationView
type Observation = executor.Observation
type AdmissionView = executor.AdmissionView
type Intent = executor.Intent
type ExecutionEffect = executor.ExecutionEffect
type EffectDispatch = executor.EffectDispatch
type DispatchAdmission = executor.DispatchAdmission

type ContinuationSlot = executor.ContinuationSlot
type ContinuationBasis = executor.ContinuationBasis
type ContinuationApproval = executor.ContinuationApproval
type ContinuationView = executor.ContinuationView
type EffectSettlement = executor.EffectSettlement
type EffectRead = executor.EffectRead
type EffectReceipt = executor.EffectReceipt
type ExternalRequest = executor.ExternalRequest
type ExternalReceipt = executor.ExternalReceipt
type ExternalAttempt = executor.ExternalAttempt
type Authorization = executor.Authorization
type LifecycleResult = executor.LifecycleResult
type LifecycleRequest = executor.LifecycleRequest
type TargetState = executor.TargetState

const MaxHistoryPage = executor.MaxHistoryPage

// InvocationStage records the latest retained stage of a call and its delivery.
type InvocationStage = executor.InvocationStage

const (
	InvocationCommitted     = executor.InvocationCommitted
	InvocationAuthorized    = executor.InvocationAuthorized
	InvocationSourceRefused = executor.InvocationSourceRefused
	InvocationAdmitted      = executor.InvocationAdmitted
	InvocationRefused       = executor.InvocationRefused
	InvocationFrozen        = executor.InvocationFrozen
	InvocationClosed        = executor.InvocationClosed
	InvocationPublished     = executor.InvocationPublished
	InvocationConsumed      = executor.InvocationConsumed
)

// EffectStage records the latest retained stage of a local effect.
type EffectStage = executor.EffectStage

const (
	EffectCommitted            = executor.EffectCommitted
	EffectAuthorized           = executor.EffectAuthorized
	EffectAuthorizationRefused = executor.EffectAuthorizationRefused
	EffectReservationRefused   = executor.EffectReservationRefused
	EffectReservedUnknown      = executor.EffectReservedUnknown
	EffectResultUnsettled      = executor.EffectResultUnsettled
	EffectSettled              = executor.EffectSettled
)

// ExternalConsequence summarizes retained evidence concerning the original external operation.
type ExternalConsequence = executor.ExternalConsequence

const (
	ConsequenceNoDispatch          = executor.ConsequenceNoDispatch
	ConsequenceUnresolved          = executor.ConsequenceUnresolved
	ConsequenceReceiptRetained     = executor.ConsequenceReceiptRetained
	ConsequenceConflictingReceipts = executor.ConsequenceConflictingReceipts
	ConsequenceReadOnly            = executor.ConsequenceReadOnly
)

// RunLocalState describes the outstanding local work for a run.
type RunLocalState = executor.RunLocalState

const (
	RunOpen                            = executor.RunOpen
	RunTerminalWithObligations         = executor.RunTerminalWithObligations
	RunTerminalWithUntrackedOperations = executor.RunTerminalWithUntrackedOperations
	RunTerminalSettled                 = executor.RunTerminalSettled
)
