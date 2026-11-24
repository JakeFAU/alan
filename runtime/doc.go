// Package runtime owns execution, persistence, recovery, inspection, and
// replay for a single-host ALAN application on a local Unix filesystem.
//
// # Running an agent
//
// Compile a definition with author.Compile, bind its named instances using
// author.Application.Provision, and pass the resulting deployment and
// DefaultLimits() to Create. Each database has one execution owner. Open it
// with Open after a process restart. Both operations validate retained state.
// Close the owner when finished. Independent readers use InspectExecution,
// Inspect and Replay by path and may coexist with the owner.
//
// Run takes positive MaxSteps and MaxDispatch budgets and an optional map from
// capability name to effects.Adapter. It returns when execution is quiescent,
// a budget is exhausted, the context is cancelled, or an error occurs.
// RunProgress skips the full report. Both methods may have committed steps
// before returning an error. Quiescent executions can still wait for local input,
// protected continuation approval, or a missing receipt. Use AdmitInput for an
// application request. For protected continuation slots, read ContinuationBasis,
// make the application's approval decision, then call ApproveContinuation.
//
// InspectExecution returns a detached view of a single durable cut. For example,
// cut.Run(RunID{Subject: "reader", Generation: 1}) returns the run and a found
// flag. Inspect its Projection.Status, Projection.Data, Wait, and EffectIDs.
// cut.Effect(id) returns the committed request's stage, reservation, result and
// settlement. Replay verifies recorded decisions offline with no adapter registry.
//
// # Handling outcomes
//
// Use errors.Is for wrapped errors. A malformed local input envelope matches
// fs.ErrInvalid. A well-formed input that violates the declared port schema
// produces an AdmissionResult with Accepted=false and Reason="schema_invalid".
// Program compilation and provisioning also have their own validation errors.
//
// ErrCapabilityUnavailable leaves the committed request available. Supply the
// capability on a later Run. ErrLimit covers exhausted budgets and configured
// size limits. An ineligible continuation basis matches ErrIneligible. ErrStale
// identifies a changed basis and ErrClosed identifies a closed target.
//
// An adapter's known rejection is returned to the program through Await, with
// valid=false, its disposition and typed default payload fields. A success
// candidate with a schema mismatch settles as "schema_invalid". The program
// should branch on valid before consuming application output.
//
// ErrAdapterOutput means the adapter supplied response evidence that could not
// be recorded, for example invalid usage JSON or the wrong correlation key.
// An allowed ordinary attempt without a recorded result remains unknown across
// reopen, including an explicit "unknown" candidate. RunReport.UnknownAdapters
// and InspectExecution identify these operations. The runner will not dispatch
// that ordinary effect again. External application remains uncertain.
//
// ErrCorrupt indicates a persistent integrity violation. ErrVersion indicates
// an unsupported format or protocol. Filesystem, database and cancellation
// errors retain their underlying error identities where supplied by the OS,
// database driver or context. Inspect the returned error, the operation and
// the retained state before deciding whether to continue.
package runtime
