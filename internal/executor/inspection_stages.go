package executor

// InvocationStage records the latest retained stage of a call and its delivery.
type InvocationStage string

const (
	InvocationCommitted     InvocationStage = "committed"
	InvocationAuthorized    InvocationStage = "authorized"
	InvocationSourceRefused InvocationStage = "source_refused"
	InvocationAdmitted      InvocationStage = "admitted"
	InvocationRefused       InvocationStage = "refused"
	InvocationFrozen        InvocationStage = "frozen"
	InvocationClosed        InvocationStage = "closed"
	InvocationPublished     InvocationStage = "published"
	InvocationConsumed      InvocationStage = "consumed"
)

// EffectStage records the latest retained stage of a local effect.
type EffectStage string

const (
	EffectCommitted            EffectStage = "committed"
	EffectAuthorized           EffectStage = "authorized"
	EffectAuthorizationRefused EffectStage = "authorization_refused"
	EffectReservationRefused   EffectStage = "reservation_refused"
	EffectReservedUnknown      EffectStage = "reserved_unknown"
	EffectResultUnsettled      EffectStage = "result_unsettled"
	EffectSettled              EffectStage = "settled"
)

// ExternalConsequence summarizes retained evidence concerning the original external operation.
type ExternalConsequence string

const (
	ConsequenceNoDispatch          ExternalConsequence = "no_dispatch"
	ConsequenceUnresolved          ExternalConsequence = "unresolved"
	ConsequenceReceiptRetained     ExternalConsequence = "receipt_retained"
	ConsequenceConflictingReceipts ExternalConsequence = "conflicting_receipts"
	ConsequenceReadOnly            ExternalConsequence = "read_only"
)

// RunLocalState describes the outstanding local work for a run.
type RunLocalState string

const (
	RunOpen                            RunLocalState = "open"
	RunTerminalWithObligations         RunLocalState = "terminal_with_obligations"
	RunTerminalWithUntrackedOperations RunLocalState = "terminal_with_untracked_operations"
	RunTerminalSettled                 RunLocalState = "terminal_settled"
)
