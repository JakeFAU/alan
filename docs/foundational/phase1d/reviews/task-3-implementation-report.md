# Task 3 report: ordinary lifecycle protocol and nested supervision

Status: implemented and source frozen for independent review. No commits, subagents, dependency changes, or historical research changes. Task 4 external endpoints and Task 5 comprehensive crash/controller/oracle work have not been implemented. Parent owns public documentation. The reviewed Task 1 output limits and Task 2 typed transport namespaces remain intact; `program.go` has no changes and no supervisory instruction.

## Changed files

New implementation: `internal/executor/lifecycle.go`, `lifecycle_replay.go`, `examples.go`.
New tests: `internal/executor/lifecycle_test.go`, `supervision_test.go`.
Supporting changes: `store.go`, `replay.go`, `messages.go`, `runner.go`, `inspect.go`; `cmd/alan/main.go` and its test. The existing Go/TDD workflow was continued; no additional library was introduced.

## Interfaces and durable boundaries

```go
var ErrSourceDenied = errors.New("source authorization denied")
type LifecycleRequest struct {
    Intent Intent
    SourceRevision int64
    Operation string
    ExpectedRevision int64
}
type TargetState struct {
    Run RunID
    ProgramID, Owner string
    Revision, Epoch, Sequence int64
    Retired, RetryCancelled bool
    Projection Projection
}
type LifecycleResult struct {
    Request LifecycleRequest
    DecisionSequence int64
    Accepted bool
    Reason string
    Successor int64
    Before, After *TargetState
}
type LifecycleOutcome struct {
    Request LifecycleRequest
    Disposition string
    Accepted bool
    Reason string
    Successor int64
}
func (*Store) LoadLifecycleRequest(context.Context, string) (LifecycleRequest, error)
func (*Store) DecideLifecycle(context.Context, LifecycleRequest) (LifecycleResult, error)
func (*Store) SettleLifecycle(context.Context, string, string) (AdmissionResult, error)
type NestedConfig struct {
    Outer, Inner, Worker string
    RetryBudget int64
    SuccessorInput string
}
func NestedProvision(NestedConfig) (Provision, error)
```

The five independently callable boundaries are existing `Commit` for ordinary Emit (including pending ID and budget reservation), existing `AuthorizeIntent`, new `DecideLifecycle`, new `SettleLifecycle`, and existing ordinary receive/`Commit`. No transaction combines these boundaries. `LoadLifecycleRequest` only reads the exact committed descriptor. Source authorization denial remains an authorization attempt, not a fabricated target rejection.

A lifecycle intent has Kind `lifecycle`, target subject, positive expected target generation, nonempty obligation, and payload `operation` string and positive `revision` integer. Supported operations are `Retry`, `CancelRun`, `AbandonRun`, `RetireSubject`. The complete original intent, including other payload fields, is retained and compared. SourceRevision comes from committed execution evidence, separately from expected target revision.

`DecideLifecycle` first checks the retained result by original intent identity. An identical full request returns the first result before reconsidering circumstances; changed descriptor conflicts. A new result requires the exact committed request and an existing allowed source authorization. It does not silently perform authorization. Denied source authority returns ErrSourceDenied without a target result. Static target permission is `(lifecycle, source subject, target subject, operation, expected target revision)`.

Decision checks, in order: target existence, matching grant, expected generation, expected revision, retirement, operation eligibility. Rejection reasons are `target_missing`, `grant_denied`, `generation_changed`, `revision_changed`, `retired`, `retry_cancelled_disabled`, `not_retryable`. Accepted Retry has reason `retried`, creates exactly generation+1 with original root artifact and initial projection, and records its creation link. Failed and abandoned are retryable; cancelled requires explicit RetryCancelled policy. Open/succeeded are not retryable.

CancelRun/AbandonRun close only open targets, with reason `closed`; on already terminal targets they accept as `already_terminal` without rewriting committed failure/success. RetireSubject records permanent retirement, reason `retired`, closes an open target as cancelled, and preserves an existing terminal classification. Ordinary End still cannot declare privileged cancelled/abandoned status.

At `math.MaxInt64`, eligible Retry returns **ErrLimit as an operational error**, not a retained target rejection. Its transaction leaves no result or successor and does not overflow. Previously committed intent/reservation and source authorization remain retained. This is a representation capacity, not an architectural retry-count bound.

## Schema and inspection changes

The unreleased prototype remains SchemaVersion1; no migration is supplied for databases created by earlier slices. Older incompatible schemas fail validation rather than being silently upgraded.

- `lifecycle_results`: `id TEXT PRIMARY KEY`, `decision_sequence INTEGER NOT NULL UNIQUE CHECK(decision_sequence>0)`, `descriptor BLOB NOT NULL`, `result BLOB NOT NULL`. Descriptor is the complete request; result retains before/after target state, acceptance/reason, successor and decision order.
- New `run_origins`: `(subject,generation)` primary key and FK to runs; `kind` root/retry; nullable `operation_id` FK to lifecycle_results (deferred); nullable predecessor_generation; full descriptor BLOB. Partial unique index `one_static_root` enforces one root per subject. Root descriptor retains original Subject, ProgramID and Initial projection; root has no lifecycle operation/predecessor. Retry retains the same root descriptor and links the exact accepted operation and predecessor generation. Explicit initial generation42 works without assuming gen1.
- `terminal_obligations` adds nullable deferred lifecycle `operation_id` and positive `source_revision`. Its run FK replaces the old history-sequence FK so authoritative closure at sequence0 is representable. Ordinary terminal has no operation link; lifecycle closure records exact authority operation and revision.
- Admissions' trusted domain CHECK includes `lifecycle`, alongside input/message/terminal. Canonical key remains `["admission", domain, rawTransportID]`; caller-supplied raw IDs do not determine the domain.
- Existing settlements now store ID `["lifecycle-outcome", intentID]`, original intent FK, complete LifecycleOutcome descriptor and observation ID.
- Authorization adds `ClosureOperation string`. New authorization after lifecycle closure records the exact closure proof; an already allowed authorization remains unchanged and usable after closure.
- Inspect includes `run_origins` and exposes full existing result/authorization/settlement/admission/terminal tables. Independent reviewers can follow creation and closure links rather than infer them from mutable projections.

Settlement disposition is `target_decision` or `source_denied`; these remain distinct. Authenticated observation Kind is `lifecycle`, Provenance `runtime-lifecycle`, source target subject and **original request's expected generation**, destination original requester execution, exact obligation and operation intentID. Payload contains accepted, reason, successor, disposition. Stale-generation rejection therefore matches the original receive, without claiming the target still occupies that generation.

SettleLifecycle atomically retains admission, observation and settlement. It accepts late evidence for the original requester even if that execution closed; it does not reopen or resume it. AdmissionResult Accepted/Reason `settled` describes evidence admission, not target acceptance. Repeated same-domain raw transport compares immutable descriptors; different transports reuse the same logical observation and settlement. No additional lifecycle ACK boundary is required. Automatic transport is `["lifecycle-delivery", intentID]` in the trusted lifecycle domain.

Runner dispatch now recognizes lifecycle intents generically: authorize; decide if authorized; settle. Each remains a separate transaction. Settled lifecycle intents are resolved; unsettled ones remain inspectable pending work. MaxDispatch counts a protocol delivery as before. Unsupported external intent kinds remain unresolved for Task 4.

## Reconstruction and immutable evidence

Ordinary per-run replay is unchanged. Its final projection must match current state unless an exact accepted authority closure proves the same-sequence transition from the ordinary projection to cancelled/abandoned. There is no general mismatch exemption.

Lifecycle replay reconstructs subject generation/retirement from explicit roots in decision_sequence order, verifies complete requests, allowed authorization, historical before state, monotonic observed per-run sequence, epoch bounds, and recomputed authority decision/result. It validates each successor against its exact accepted Retry, checks that each accepted Retry has its successor, validates root/program/initial identity and final subject authority, and reconciles closure obligations and settlement/admission linkage. Zero-step successor and sequence0 closure are supported.

Messaging replay reconstructs source authorization from ordinary history plus the exact retained lifecycle closure proof where required. This accepts prior allowed authorization after closure while validating newly denied source authorization correctly. Startup validates these records before epoch renewal or dispatch. Corrupt creation link, missing linked result and inconsistent closure tests assert epochs did not change on rejection.

This is consistency validation, not an independent conformance oracle: it shares target-policy evaluation with execution. Arbitrary coherent database tampering is not claimed detectable. Deleting an otherwise unreferenced rejected result can resemble an operation not yet decided; retained before/after evidence in the later independent oracle is required for that negative control. There is no newly assumed tamper-proof global journal.

## Ordinary nested fixture and CLI

NestedProvision defaults to outer/inner/worker and supports distinct renamed IDs, nonnegative RetryBudget and SuccessorInput (default success). Worker generation1 and generation2 run the **same program artifact**: receive explicit mode from inner and branch to succeeded/failed. There is no generation-dependent success behavior.

Inner receives outer's start, sends explicit fail to worker g1, consumes authenticated worker terminal, checks its own ordinary budget, and emits Retry with spent increment and pending operation ID atomically. It waits for exact correlated lifecycle outcome, updates expected generation to returned successor, clears pending, sends configured successor input, then observes the successor terminal. Budget exhaustion produces ordinary public failure. Outer starts inner and consumes only inner's public terminal; it has zero retries and never reads inner's private data. Rights and terminal obligations are explicit static provision data, not ancestry privileges.

Inner PCs: 0 start receive; 1 fail input; 2 worker terminal receive; 3 success branch; 4 failure branch; 5 budget branch; 6 Retry Emit/reservation; 7 lifecycle receive; 8 accepted branch; 9 expected-generation update; 10 pending clear; 11 successor input; 12 jump2; 13 success End; 14 failure End.

CLI adds `alan init --db PATH --demo nested [--retry-budget 1] [--successor-input success|fail]`. Exactly one provision file/demo is required; only nested is recognized. Existing run/inspect/replay/input remain. Standard-library flag only.

Compiled CLI smoke at `/tmp/alan-task3-cli-smoke-auidvbgo` saved init/run/inspect/replay JSON plus summary. Actual result: 26 steps, 7 dispatches, quiescent, no open runs/unresolved intents/pending deliveries/rejected admissions. Retry ID `["inner",1,7,0]`, successor2; inner sequence16/spent1/expected2; worker g2 sequence3; unchanged worker artifact; 26 history rows, 7 admissions, 1 settlement, 4 origins. Inspect before and after read-only replay was identical.

## Tests and RED evidence

There are now 63 top-level executor+CLI tests (19 added this task), plus unchanged historical semantics tests. Initial lifecycle tests were written against missing implementation stubs and failed as retained in `/tmp/alan-task3-lifecycle-red.log`: accepted Retry reopen, original rejection after eligibility changes, cancellation policy/abandonment/retirement, observer without grant, original service binding across real Retry. Each failure was `not implemented`, including four cancellation-policy subcases. Their first GREEN is `/tmp/alan-task3-first-green.log` (executor0.335s).

Nested/CLI tests likewise preceded implementation: `/tmp/alan-task3-nested-red.log` retains `not implemented` for nested default/renamed programs, budget0/1 exhaustion and lost accepted result; CLI failed `flag provided but not defined: -demo`. GREEN `/tmp/alan-task3-nested-green.log`: executor0.935s, CLI0.340s. These logs retain exact observed RED inputs/test names and failure lines; they are not rewritten to suggest later boundary tests independently failed first.

Additional coverage added against the implementation passed first run:

- `/tmp/alan-task3-boundaries.log` (0.271s): prior source authorization survives lifecycle closure; new source authorization denied with closure proof; late CancelRun preserves failed classification; old lifecycle result cannot consume new pending operation; stale-generation rejection consumes original wait; inconsistent creation/result/closure rejected before epoch renewal.
- `/tmp/alan-task3-atomicity-races.log` (0.221s): Retry versus CancelRun/RetireSubject in both actual serialized orders; static root42 to successor43; expected revision rejection with a matching grant so missing rights cannot mask the check; injected target-transaction rollback and separate settlement rollback.
- `/tmp/alan-task3-generation-capacity.log` (0.187s): eligible Retry at MaxInt64 returns ErrLimit without result, successor or partial mutation.
- `/tmp/alan-task3-interim-regression.log`: executor0.464s, CLI0.146s.

The changed-eligibility rejection case uses real APIs: original Retry against active worker rejects; worker later commits failure; identical operation remains rejected; a new authorized operation accepts. Accepted-result recovery closes/reopens after decision and before settlement, preserves same requester obligation/pending/spent, retransmits original decision and executes the successor. These are durable close/reopen and transaction-failure tests; the expanded two-physical-crash/controller matrices remain Task5. Existing Task1 process-kill transaction tests remain in the full suite.

Final exact commands, after all source/test edits:

```sh
GOCACHE=/tmp/alan-phase1d-go-build go test ./...
GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor ./cmd/alan
GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
git diff --check
GOCACHE=/tmp/alan-phase1d-go-build go build -buildvcs=false -o /tmp/alan-task3-cli ./cmd/alan
```

All exit0. Full suite (`/tmp/alan-task3-full.log`): CLI0.166s, executor0.670s, historical research cached PASS. Focused race (`/tmp/alan-task3-race.log`): executor5.282s, CLI1.498s. Vet (`/tmp/alan-task3-vet.log`) and diff check had no diagnostics. Compiled CLI init/run/inspect/replay/inspect smoke used subprocess checked exits and asserted identical inspection, with actual summary above. No source modifications followed these checks; this report is documentation only.

## Remaining scope and costs

No external endpoint, receipt/fencing or effect reconciliation is implemented here. Task4 must extend settlement validation for its own retained outcome types; current settlement validation is lifecycle-specific. No owner transfer/dynamic grants, cross-execution fresh-supervisor reassociation, migration, garbage collection, or performance claim is added. Reconstruction scans retained history/evidence; results and root descriptors intentionally duplicate data for inspectable exact linkage. The API boundaries are reusable for real process failpoints/controller work, but this task does not claim the final independent conformance/crash matrix is complete.
