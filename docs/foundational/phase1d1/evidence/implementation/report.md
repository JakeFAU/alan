# Task 1 implementation report — DONE / code frozen

Observed decisive sequence: endpoint holding request -> independent peer committed -> peer result consumed by ordinary caller -> endpoint response released -> original result settled and its ordinary execution completed.

The actual CLI and independent resource-process cases record that sequence for both endpoint profiles. Each peer/caller commit acknowledgment includes the exact still-active authorization from worker lifecycle observations; a separate request-context live probe confirms the same held endpoint request. The controller only releases real boundaries and supplies an ordinary input in the separate wakeup test. It never calls Commit to advance the decisive peer.

## Source and scope

Workspace: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`; branch `codex/phase1d1-async-dispatch`; baseline HEAD `6effdc02259af5335633bfc41e9786386430fa7c`. No commits or subagents were created by this implementer. Parent owns README, new phase documentation, final archive/publication and independent review.

Frozen implementation source aggregate: `84b629a219226f45636f904379bcd9a5d387b36f0c0b81e24ee007060a1c1436` over 57 files (go.mod, go.sum and every cmd/internal/research Go file). Algorithm and per-file hashes: `/tmp/phase1d1-implementation-evidence.json`. No dependency/schema/migration/key namespace changes. Research sources are unchanged. The historical baseline and its immutable full test reproduction remain separate parent-owned evidence.

Runtime edits:

- `internal/executor/runner.go`: one Run owner, bounded worker/candidate slots, per-invocation intent claims, category opportunities, idle wake/select, cancellation/drain/join, precise budgets.
- `internal/executor/external.go`: owner-side preparation and completion accounting; immutable transport work; network-only workers; retained descriptor/authorization revalidation; late old-epoch evidence; skip retained non-idempotent Unknown without a new dispatch charge; exact worker finished/settled boundaries.
- `internal/executor/store.go`: Close/Run coordination; owned write-transaction lifetime; coalesced wake hints; committed-state return on post-commit observer error so budget counts the durable commit.
- `internal/executor/messages.go`, `lifecycle.go`: active-store write transactions use the same owner rollback/commit/wake helper. No policy/identity logic changes.
- `cmd/alan/main.go`: `--max-in-flight`, `--external-timeout`, explicit nonpositive-option rejection.
- `internal/boundary/boundary.go`: exact multi-filter support without holding the filter mutex across a selected pause; cancellable inherited-pipe reads; request-live probes; worker-active authorization snapshots. `cmd/alan-resource/main.go` passes request cancellation into its observer.
- Existing `external_test.go`, `fault_process_test.go`: only mechanical keyed RunOptions conversion. Existing `durabletest/crash_test.go`: only add the optional InFlight field to the evidence envelope; all old assertions retained.
- New tests: executor `async_test.go`, `async_limits_test.go`; durabletest `async_test.go`; CLI `async_test.go`.

## Actual contract

Go RunOptions adds `MaxInFlight int` and `ExternalTimeout time.Duration`. API zero chooses defaults 2 and 2 seconds; negative values fail. CLI explicitly supplied zero/negative values fail before opening the store. Existing positive MaxSteps/MaxDispatch validation remains.

One owner holds `runDone/runCancel/runWake` under a separate owner mutex. Concurrent Run returns ErrRunActive. Close marks closing, cancels Run, waits for its joined cleanup/report, then closes the SQLite connection and releases the OS lock. New Run is rejected after closing begins.

At most MaxInFlight transport workers plus unaccounted candidates exist in total. Completion capacity equals that bound; every launch owns exactly one send, so joining workers cannot block behind a full completion queue. The same intent is claimed once per invocation, including uncertain outcomes. Extra work stays in the durable ledger. Workers receive copied descriptors/events/byte bounds/timeout and no Store pointer. They perform HTTP and return a candidate; all authoritative writes stay with the owner.

Each scheduler pass offers one completed candidate, one internal delivery/lifecycle operation, one eligible external launch, and one round-robin ordinary commit an opportunity. Saturated external slots are skipped. There is no perpetual delivery-priority loop. When no immediate work exists and owned transport remains, select waits for completion, cancellation, or a coalesced authoritative-store-commit wake hint. The hint has capacity one and is never the source of truth. Concurrent AdmitInput is supported and tested while the owner is idle and an external request remains held. Quiescent is never true with locally owned transport; Quiescent plus retained Unknown only means no eligible action in this invocation.

MaxSteps bounds committed ordinary transitions, including a transition whose post-commit observer returns an error. MaxDispatch reserves one unit before each internal-delivery attempt or fresh/recovery external work preparation/launch. A preparation that settles a retained receipt or denial counts as supported delivery work. A retained non-idempotent uncertain attempt is skipped with no charge. Candidate handling, scanning and wake hints spend no extra unit. A limit error occurs when additional eligible work would exceed its bound; merely reaching a bound with no further eligible action need not be an error. Saturation does not suppress local steps. No logical CancelRun/RetireSubject is synthesized by stopping the executor.

Exit policy: stop launches, cancel the Run transport context, join all owned workers, then account exactly the remaining candidates and build the report using a separate 5-second cleanup context derived with cancellation removed (observer values retained). Received valid outcomes are persisted when possible; a failed persistence/settlement/report remains an explicit joined error, with durable unresolved records retained for recovery. An already dequeued candidate whose persistence fails is not falsely reported settled. Non-idempotent uncertainty is never blindly repeated. The test observer honors context cancellation and polls its release pipe at 50ms intervals.

The five seconds is a cleanup context budget, not a hard wall-clock promise for all of Close: joining is mandatory, OS scheduling is external, and SQLite's existing 5000ms busy timeout / an in-progress non-context Commit can outlast a context deadline. No worker is abandoned to obtain a faster return. The standard local HTTP transport is cancellable and also has its positive configured total request/body timeout. There is no store mutex or SQLite transaction held during network/body reads, capacity/completion waits, or worker joins. Selected test-only pre-commit pauses deliberately hold the real transaction to test crash/progress boundaries.

Owned write transactions use context.WithoutCancel only for BeginTx's transaction lifetime, after an initial ctx.Err check. Statement contexts remain cancellable; every caller retains deferred Rollback; commit checks ctx.Err before committing. Cancellation before that final check aborts. Cancellation racing after it may leave a whole committed transition, never a partial one. This avoids database/sql's cancellation auto-rollback discarding the pinned SQLite connection. Create/Open and read-only inspection/replay transactions retain their original behavior. This helper is used by active-store commit/admission/authorization/delivery/lifecycle/external receipt/settlement transactions.

The explicit public DispatchExternal API remains synchronous and performs at most one bounded request per call; Run uses the bounded path. In-flight limits/budgets refer to automatic Run-owned work. Arbitrary user-installed observer callbacks must honor their context; the supplied observer does.

## Tests and unchanged legacy coverage

A: `TestAsyncHeldRequestPeerProgress`; actual CLI `TestAutomaticAsyncHeldProgressAndConcurrentCrash` settle cases. Exact endpoint request, peer staged/committed, caller consumption, active worker authorization and live endpoint probe all precede response release; final result is durably correlated/consumed.

B: `TestAsyncSaturationLeavesBacklogAndDeliversPeer` capacities 1/2/3, all slots held, extra ledger work, 12 ordinary peer steps, terminal internal delivery, one-slot release admits backlog, endpoint peak bound, no duplicate intent contacts.

C: `TestAsyncReorderedReceiptsConsumeWithoutHeadOfLineAndReplay`: second operation consumes first, distinct results remain correctly bound, first released only afterward, replay makes no contact.

D: `TestAsyncOrdinarySupervisorAdvancesWhileExternalHeld` uses real NestedProvision policy and accepted lifecycle protocol during hold; `TestAsyncAuthorizedTransportFinishesAfterOrdinarySourceClosure`; existing authorization-before/after-closure test unchanged; old-epoch completion validation plus forged candidate rejection.

E: received-candidate plus held-request cancellation, Close/overlapping-Run, both step/dispatch budget exits with active and received outcomes, reopen consumption/Unknown, failure-injected SQLite receipt persistence, post-commit counter accuracy, and concurrent ordinary input wake. Repeated race cancellation count20 and queued-budget count5 passed.

F: timeout and HTTP503 unavailable cases require peer commit before transport failure, leave application open and Unknown; existing timeout/Unknown tests retained. Storage failure remains explicit and recoverable.

Crash: both endpoint profiles x held-before-receipt SIGKILL and worker-received-before-record SIGKILL, plus both successful settle paths. Resource process/store survive; peer/caller history retained; idempotent recovery reuses original key with one mutation and two contacts; non-idempotent recovery retains Unknown with one contact. Replay before/after recovery does not dispatch. Existing accepted Retry/rejection/second-recovery-crash/terminal-notification/replay checks remain unchanged and pass.

Oracle controls reject peer-before-hold, wrong intent/attempt/process, controller commit substitution, missing active transport, timeout-first, and final-state-only sequences. Raw event order is retained; no race ordering is sorted into a passing sequence.

## RED, discovered defects and retained artifacts

1. Original synchronous defect: `/tmp/phase1d1-held-original-test.go` and `/tmp/phase1d1-held-red.log`; original uninstrumented fixture SHA256 `0d09aab90953d472c5f7ae8265b5cf76c83a3c4ea02b50bcd25811cc6f387993`. Behavioral failure is `held request ended before peer commit`. This was observed before production edits and after parent baseline PASS. The first attempted fixture had missing Match.Operation and wrong terminal provenance; that setup-only failure (`operand must have exactly one reference or literal`) was corrected before the behavioral RED and is not claimed as runtime evidence.
2. Same oracle with telemetry only (assertions/fixture behavior unchanged): `/tmp/phase1d1-held-traced-oracle.go` SHA256 `5584d82893c0fb2663289dc5bbaaa019560486d80e00731007daff364503e88a`. Byte-identical test runs on synchronous baseline and corrected source. `/tmp/phase1d1-held-traced-red.log` FAIL; `/tmp/phase1d1-held-traced-green.log` PASS. Includes initial provision, exact request/boundary sequence, raw record snapshot, errors/report. Reproducible synchronous full code archive `/tmp/phase1d1-sync-negative-control.tar.gz` and per-file manifest.
3. Inherited-pipe observer defect: exact saved `/tmp/phase1d1-pipe-before-fix.go`, original `/tmp/phase1d1-pipe-deadline-failure.log`; `os.NewFile` inherited blocking fd lacked deadlines. SetNonblock before NewFile fixes poll registration. Complete reconstructed compilable pre-fix archive `/tmp/phase1d1-pipe-negative-control.tar.gz`, manifest, command metadata and `/tmp/phase1d1-pipe-negative-control.log` reproduce the missing endpoint acknowledgment; child log reports `file type does not support deadline`. Reconstruction includes later caller binding correction; the original pipe failure happens before that binding executes. This is failed harness evidence, not progress evidence.
4. Cancellation auto-rollback discards pinned connection: original full-race failure `/tmp/phase1d1-full-race.jsonl`, complete `/tmp/phase1d1-cancel-before-fix.tar.gz`, manifest/command metadata, repeated reproduction `/tmp/phase1d1-cancel-before-fix.log` FAIL with `sql: connection is already closed`. Fixed unchanged oracle race count20: `/tmp/phase1d1-cancel-after-fix.log` PASS.
5. Post-commit budget undercount: `/tmp/phase1d1-postcommit-count-red.log` shows durable succeeded peer but Steps0; complete `/tmp/phase1d1-postcommit-before-fix.tar.gz` plus manifest/metadata. Fixed unchanged test race count5: `/tmp/phase1d1-postcommit-count-green.log` PASS.
6. Test-only fixture corrections: caller Bind map direction corrected before CLI acceptance; held HTTP test handlers now consume request bodies so server cancellation is observable. Original stalled fixture log/goroutine dump retained in `/tmp/phase1d1-focused.log`, timeout retry in `/tmp/phase1d1-focused2.log`. Step-budget fixture initially had no further eligible step and correctly quiesced after timeout; added eligible peer instead of changing budget semantics. `/tmp/phase1d1-focused3.log` retains that failed fixture expectation. These are not runtime qualification successes.

All four complete source-control archives include go.mod/go.sum and full cmd/internal/research sources and have per-file + archive digests in `/tmp/phase1d1-implementation-evidence.json`. Parent should package the exact final evidence, not overwrite historical Phase1D records.

## Final commands/results

Environment for all Go commands: `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d1-go-build GOPROXY=off GOSUMDB=off`. Local HTTP/subprocess tests ran with authorized worktree/network sandbox escalation. No dependency fetch.

- `go test -json -count=1 -timeout=180s ./...` — exit0; `/tmp/phase1d1-qualified-normal.jsonl`; 353 test/subtest passes, 0 failures, 0 skips; all six test packages pass (resource command has no tests).
- `go test -race -json -count=1 -timeout=240s ./...` — exit0; `/tmp/phase1d1-qualified-race.jsonl`; same 353 passes; actual `alan` and `alan-resource` children race-built by the existing harness. Durable subprocess package 83.857s, research 86.932s.
- `go vet ./...` — exit0; `/tmp/phase1d1-qualified-vet.log` empty.
- `gofmt -l cmd internal research` — empty; `/tmp/phase1d1-qualified-gofmt.log`.
- `git diff --check` — exit0; `/tmp/phase1d1-qualified-diffcheck.log` empty.
- Unchanged traced held regression against baseline — exit1 expected; command in `/tmp/phase1d1-sync-negative-control.json` (original pretelemetry digest retained there; final traced digest is authoritative in implementation-evidence.json).
- Focused race new suite `/tmp/phase1d1-final-focused-race.log` PASS; specific additional repeated controls as above. Earlier intermediate full normal/race logs are retained separately, including the failed race source.

`/tmp/phase1d1-qualified-async-traces-normal.jsonl` and `...-race.jsonl` each contain 27 extracted ASYNC records in original output order, including six actual CLI profile/cut records with provisions, pre-release/stopped/recovered raw records, endpoint ledgers, exact process tokens and binary hashes. Full go-test JSON logs remain canonical; extraction script `/tmp/phase1d1-package-implementation-evidence.py` does not sort events. `/tmp/phase1d1-implementation-evidence.json` binds frozen code, gates, hashes, archive controls and oracle byte equality.

No known required runtime work remains. Limits remain the declared single-host/static-subject profile, caller-selected finite budgets/capacity, no hard real-time wall-clock shutdown theorem, and the existing cross-store key namespace limitation. Independent reviewer should focus on owned transaction cancellation, lock order, candidate accounting after error, observer state snapshots, and coalesced wake correctness. Code is frozen pending independent review findings.
