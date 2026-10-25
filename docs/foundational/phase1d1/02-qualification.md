# Phase 1D.1 qualification

**Observed:** endpoint holding request -> independent peer committed -> peer result delivered -> endpoint response released -> original result settled.

The actual `alan` process produced that ordering against a separate `alan-resource` process for both endpoint profiles. The caller consumed the peer's terminal outcome before response release. The original client later consumed its own correlated receipt. This qualifies independent progress between separate logical executions while one transport waits.

## Exact source and baseline

Qualified source: **`9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63`**, 58 Go/module files. [source-manifest.json](source-manifest.json) gives every file hash and the length-framed SHA-256 algorithm. The publication commit is separate from the source identity. Branch: `codex/phase1d1-async-dispatch`; verified base: `6effdc02259af5335633bfc41e9786386430fa7c` (merged Phase 1D tree).

Go 1.27.3, darwin/arm64, CGO with Apple clang 21; unchanged `github.com/mattn/go-sqlite3 v1.14.52`, SQLite 3.53.4. The manifest records binary/compiler identities and every pinned module file. No dependency, schema, versioned language, operation-key namespace or research-model change.

The immutable base reproduced 321 passing test/subtest events in both normal and race runs. Its 53-file digest is `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`. The original Phase 1/1B/1C archives reproduce their own 15/20/32 qualification records exactly. The original Phase 1D archive also passed from an empty module cache using its bundled dependency. See [baseline report](evidence/baseline/report.md); these records keep their original identities.

On current source, all 15/20/32 research payloads equal the reproduced base payloads, including their model-source manifests. All six existing model/runtime checkpoint comparisons pass. Four comparison payloads match exactly; two external fixtures contain fresh endpoint URLs and derived program identities. Their raw records are retained and their independent checkpoint assertions pass; no byte-equality claim is made for those two. The Phase 1C oracle regression about different executions and wrongly ordered interruption/reactivation remains unchanged.

## Decisive causal evidence

`TestAutomaticAsyncHeldProgressAndConcurrentCrash` uses the real automatic CLI. A real peer transaction pauses before commit until the endpoint acknowledges durable application and its held response. The controller releases this pause; it does not call Commit. The peer terminal commit and caller receive commit each acknowledge the exact authorization still in transport. A request-context live probe confirms the same endpoint request remains held. Only then does the controller release its response; `external.settled` acknowledges durable receipt admission, and final records show ordinary consumption.

The transcript records scenario, process-start token/PID, store, subject, generation, epoch, sequence, intent and authorization. The release entry records the controller's pipe write using the last authenticated live request identity; it is not a second endpoint acknowledgment. Worker `external.finished` telemetry and the request-context probe exclude a timeout/cancellation completing before peer progress. Watchdog deadlines only fail stuck tests. No cross-process clock comparison establishes order.

One normal-run witness: scenario `TestAutomaticAsyncHeldProgressAndConcurrentCrash/idempotent/settle`, endpoint PID 42679, executor PID 42683, client generation 1/epoch 2/sequence 1, intent `["client",1,1,0]`, authorization `["authorization","[\"client\",1,1,0]"]`. Peer and caller each commit sequence 1 with that authorization in their active transport snapshot. Machine records preserve both process tokens, complete provisioning, before-release inspection, final inspection, endpoint ledger and binary hashes.

[qualification.jsonl](qualification.jsonl) indexes the complete compressed records and carries the causal transcripts. Each pointer identifies mode, record index and content hash. [Normal raw output](evidence/final-normal.go-test.jsonl.gz) and [race raw output](evidence/final-race.go-test.jsonl.gz) preserve original event order. No concurrent trace is sorted into a passing order.

## Evidence to claim

| Contract | Executable check and result | Scope |
| --- | --- | --- |
| A: peer and caller progress during the exact held request | `TestAsyncHeldRequestPeerProgress`; CLI settle cases, both profiles: PASS | Successful controlled executions, not all scheduling interleavings |
| B: actual capacity, backlog and useful independent work | `TestAsyncSaturationLeavesBacklogAndDeliversPeer`, capacities 1/2/3, all slots held, 12 peer steps, internal terminal delivery, one-slot release: PASS | Real active transports bounded; durable backlog, no per-intent goroutine fanout |
| C: second response can be consumed first | `TestAsyncReorderedReceiptsConsumeWithoutHeadOfLineAndReplay`: PASS | Exact two-operation correlation; replay makes no endpoint contact |
| D: ordinary lifecycle progress while transport waits | `TestAsyncOrdinarySupervisorAdvancesWhileExternalHeld`: PASS | Existing nested policy and real accepted Retry; environmental hold is not supervisory policy |
| D: closure around authorization and late evidence | Existing source-closed authorization case; `TestAsyncAuthorizedTransportFinishesAfterOrdinarySourceClosure`; `TestAsyncLateReceiptValidatesRetainedEpochAndRejectsForgery`: PASS | Deny before authorization; retain authentic authorized old evidence after closure without reopening |
| E: cancellation, limits, queued completions, Close and overlap | `TestAsyncCancelAccountsReceivedCandidateAndPreservesUnknown`, `TestAsyncLimitsWithHeldWork`, `TestAsyncBudgetAccountsReceivedOutcomeWhileAnotherRequestIsHeld`, `TestAsyncOverlappingRunRejectedAndCloseJoins`: PASS | Received evidence accounted or explicit error/uncertainty; join before close; both budget kinds |
| E: storage failure and post-commit counts | `TestAsyncReceiptPersistenceFailureIsExplicitAndRecoverable`; `TestAsyncBudgetCountsCommitEvenIfPostCommitObserverFails`; `TestAsyncKnownReceiptSurvivesSettlementFailure`; malformed-receipt report checks: PASS | Persistence failure is visible; durable commit still consumes a step when post-commit observer fails |
| F: transport failure does not hold peer progress | `TestAsyncTimeoutAndUnavailableAllowProgressBeforeFailure`, timeout and HTTP 503: PASS | Peer acknowledgment precedes transport ending; run stays logically open with Unknown |
| Supported idle wakeup | `TestAsyncAdmittedInputWakesIdleOwnerWhileTransportHeld`: PASS | Concurrent in-process ordinary admission wakes the sole owner; no input daemon introduced |
| Concurrent process death | CLI both profiles × held-kill/received-kill: PASS | Four actual SIGKILL cases after peer/caller progress; endpoint process/store survives |
| Existing durable semantics | All earlier Phase 1D crash tests and six conformance fixtures: PASS | Accepted Retry, immutable rejection, second crash in recovery, terminal notification and no-dispatch replay preserved |
| Oracle rejects false positives | `TestAsyncOracleRejectsFalseOrderAndIdentity`, synchronous source control: PASS / expected FAIL | Early peer, wrong request/process, substituted controller commit, missing active transport, timeout-first and final-state-only traces rejected |

For concurrent crash cases, no runtime transaction spans endpoint application. At `held-kill`, the endpoint has committed application and runtime receipt storage has not begun. At `received-kill`, a worker has a valid response in memory but has not recorded it. Both require the already acknowledged peer and caller commits to survive. Idempotent reopen contacts the endpoint under the original key and retains one mutation (two contacts). Non-idempotent reopen keeps Unknown and one contact, with the client still open. The volatile completion is never recovery evidence. Replay before and after live recovery changes neither endpoint state nor lifecycle state.

These are process-crash tests under the declared local-storage assumptions, not power-loss, disk-failure or distributed-partition tests. Repeated race runs supplement specific shutdown cases; they are not exhaustive concurrency searches.

## Actual commands and results

Environment: `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d1-go-build GOPROXY=off GOSUMDB=off`; final collector also sets `GOFLAGS=-mod=readonly`. Local loopback/process permissions were enabled. Commands and raw output are retained in [normal metadata](evidence/final-normal.command.json), [race metadata](evidence/final-race.command.json), [static checks](evidence/final-checks.json) and [implementation report](evidence/implementation/report.md).

| Command | Actual result |
| --- | --- |
| `go test -json -count=1 ./...` | PASS, 357 test/subtest passes, zero failures, 19.801s wall time |
| `go test -json -count=1 -race ./...` | PASS, 357 test/subtest passes, zero failures, 102.571s wall time; actual child binaries race-built |
| Focused new async suite under race | PASS; retained implementer log |
| Cancellation regression `-race -count=20` | PASS after pinned-connection correction |
| Queued-candidate budget regression `-race -count=5` | PASS for both budgets |
| Post-commit count regression `-race -count=5` | PASS after count correction |
| `go vet ./...` | Exit 0, no output |
| `gofmt -l cmd internal research` | No output |
| `git diff --check` | Exit 0, no output |
| `go mod verify` | All modules verified |
| Complete source archive, empty module cache | 95 members verified; full suite39.266s, module verification and vet PASS using bundled file proxy |
| README smoke commands | 18 commands PASS; nested completion, exhaustion, reopen/replay and independent external endpoint |

Each final mode retains 15 QUALIFICATION, 20 COMPOSITION, 32 SUPERVISION, 6 CONFORMANCE and 119 DURABLE records plus 22 directly emitted ASYNC records. Six of the DURABLE records also have `type=ASYNC`, giving 27 structured async traces plus one human-readable held trace. There are 85 subprocess records, including 20 acknowledged SIGKILL exits, per mode. `cmd/alan-resource` has no package tests; its executable is exercised by integration tests. Test/subtest counts are reporting counts, not independent property counts.

## Retained defects and negative controls

[evidence/implementation/inventory.json](evidence/implementation/inventory.json) maps original paths to retained bytes and hashes. Five complete pre-fix source archives have per-file manifests, commands and original failures. The pinned dependency is supplied by the final source archive's local proxy.

1. **Original synchronous dispatch:** the unchanged traced held test fails with `held request ended before peer commit`. Its exact bytes pass on corrected source. The original pre-telemetry RED and fixture remain separately retained. Shorter timeouts are not the repair.
2. **Inherited-pipe observer deadline:** the new observer initially exited because a blocking inherited FD was not registered for deadlines. The original source/failure and a complete reconstructed failing source are retained. Nonblocking registration before `os.NewFile` fixes this harness defect. The reconstruction includes a later caller-binding correction; it is labeled reconstruction, not an exact historical whole-tree capture.
3. **Cancellation and SQLite connection ownership:** database/sql cancellation-triggered rollback discarded the pinned connection, so cleanup could not persist a received candidate or inspect uncertainty (`sql: connection is already closed`). Complete pre-fix source, actual failed full race output, repeated reproduction and unchanged passing regression remain. Active write transactions now own rollback, retain cancellable statements and check cancellation before commit. No lost acknowledged commit was established by this failure.
4. **Post-commit budget undercount:** a canceled post-commit observer could return an empty state and Steps=0 after a durable successful commit. Return the known committed state with the error and charge that commit. Complete pre-fix source and RED/GREEN remain.

5. **Known receipt incorrectly reported Unknown:** independent final review persisted a valid receipt then rejected its separate settlement. Recovery was correct, but `UnknownExternal` still included the operation and malformed receipt evidence was not validated in this report path. The fix uses existing durable outcome validation: known evidence remains pending/unsettled with the explicit error, and missing evidence alone is Unknown. Permanent regression plus unchanged reviewer assertion pass, including non-idempotent reopen without redispatch. [Exact pre-fix input, complete source, RED/GREEN and fix report](evidence/final-fix/inventory.json) remain; [the pre-final full qualification](evidence/pre-final-review/02-qualification.md) retains the former source identity and gates.

Fixture construction errors are disclosed separately: initial missing receive correlation fields; reversed caller binding; handlers waiting for request cancellation without draining the body; and a step-budget fixture with no further eligible work. They were corrected rather than counted as behavioral evidence. Their saved failures are not passing qualification.

Packaging corrections did not change runtime code: the collector initially assumed every ASYNC line was JSON, then learned to retain the human-readable line and direct JSON records; copied Go excerpts initially looked like a package until renamed `.go.txt`; an attempted exact comparison of all conformance payloads was narrowed to separate deterministic model payload equality from the two ephemeral external fixtures. The retained unified patch was stored as gzip after the staged whitespace check interpreted its context prefix as Go indentation; decompressed bytes and original hash are unchanged. Original test outputs remain authoritative.

## Limits and review

The [contract](01-dispatch-contract.md) defines default capacity 2, request timeout 2s, separate commit/dispatch budgets and five-second cleanup context. Cleanup is a context budget, not a hard wall-clock shutdown guarantee: workers must be joined, SQLite's 5000ms busy timeout and an in-progress non-context commit can outlast a context deadline. Arbitrary installed observers must honor cancellation; the supplied observer does.

The explicit `DispatchExternal` method remains synchronous; bounded ownership refers to automatic `Run`. No support is claimed for independently calling that method concurrently with Run to create another dispatcher. One host, one active executor, static grants, single PC per run, no business-key namespace correction, no SDK/provider integration, no migration/compaction and no distributed fencing remain the scope. Unknown is not proof of no mutation. Bounded category opportunities are not an unbounded fairness or eventual-recovery theorem.

Independent scoped and whole-branch review findings, probes and dispositions are retained in [03-review.md](03-review.md). Review approval is evidence of examination, not proof of general correctness. [04-runbook.md](04-runbook.md) supplies repeatable commands and archive verification.
