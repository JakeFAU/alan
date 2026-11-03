package runtime

import (
	"context"

	"github.com/JakeFAU/alan/internal/executor"
)

// InspectExecution reads one coherent cut of the retained computation without
// taking ownership, executing the reducer or contacting external services.
func InspectExecution(ctx context.Context, path string, limits Limits) (*ExecutionSnapshot, error) {
	return executor.InspectExecution(ctx, path, limits)
}

// ExecutionSnapshot contains detached facts and indexed accessors for one retained cut.
type ExecutionSnapshot = executor.ExecutionSnapshot

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
