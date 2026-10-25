# Phase 1D.1 independent scoped code and test review

**Specification verdict: PASS. Quality verdict: PASS. No actionable defects found.**

The inspected execution establishes: endpoint holding the exact authorized request -> independent peer committed -> ordinary caller consumed the peer result -> controller released the response -> original result settled and was consumed. The synchronous negative control still fails the unchanged oracle with `held request ended before peer commit`.

This is an independent review of the coupled Go implementation and test/controller changes. It is not final approval of the parent-owned documentation, evidence packaging, archive, README, or publication; whole-branch review remains separate. No implementation changes, commits, or subagents were made.

## Reviewed identity and material

- Workspace: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.
- Base: `6effdc02259af5335633bfc41e9786386430fa7c`.
- Frozen source SHA256: `ee144b08e6abac10c704c990e83a80314326a1eac3956bc5ecf9489f8f0afd5a`.
- Independently checked all 57 per-file hashes/lengths and recomputed the sorted uint64BE path/content length-framed aggregate from `docs/foundational/phase1d1/source-manifest.json`, both before and after probes. They match.
- The implementer’s alternate path+NUL+file-hash aggregate is a different representation; it is not substituted for the Phase1D source identity.
- Read the task review brief, implementation brief/report, frozen review package, binding requirements, dispatch contract, changed runtime paths, new async tests, and the existing subprocess acknowledgment/resource/record-validation helpers they use.
- Existing complete normal/race JSON logs each contain 353 test/subtest passes and no failures/skips across six passing test packages. These were inspected, not rerun wholesale. Actual child binary build code adds `-race` under the race build tag; all six retained async CLI cases in the race log identify race-built children.

## Findings and implementation reasoning

There are no ranked revision requests: no reproducible correctness, specification, or quality defect was found in this scope.

1. **Network isolation and ownership.** `externalWork`/`externalCompletion` (`internal/executor/external.go:113`) carry copied descriptors and scalar limits. Transport code at lines 152–209 has no Store reference. HTTP `Do` and full response-body decoding execute outside the store mutex/transaction. Durable receipt validation and settlement return to the owner at lines 210–224. A new review probe flushed only the first JSON byte, then required the capacity-one runner to commit the peer before allowing the remainder; it passed five race-enabled runs.
2. **Bounded workers and completion backpressure.** `internal/executor/runner.go:68–98` owns the completion buffer and worker join. A launch reserves `active`, a WaitGroup entry, and one dispatch charge before creating its worker. Each launch sends exactly one result. At most MaxInFlight launched-but-unaccounted workers/results exist, so the MaxInFlight-sized buffer can receive every outstanding result before `workers.Wait` returns. The scheduler does not launch while accounting a dequeued result; decrementing `active` before accounting therefore does not allow a transient extra worker. Saturation skips launches and leaves unclaimed work in the ledger.
3. **Duplicate claims and category opportunities.** The invocation-local `attempted` map is set before preparation/launch, including uncertain outcomes; no repeated scan can launch that intent again during Run. Each loop offers one completion, one internal operation, one external preparation, and one round-robin ordinary transition. Saturation does not block ordinary transitions. Retained non-idempotent uncertainty is skipped without a new dispatch charge. Coalesced commit hints wake idle owners, and all state is rescanned from durable records.
4. **Budgets.** Dispatch is charged before internal delivery or fresh/recovery preparation; completion accounting spends no additional unit. MaxSteps counts durable ordinary commits, including errors from the post-commit observer (`store.go:724–729`). Exhaustion is checked when additional eligible work is present. No extra worker can independently overspend the owner’s counter. The no-more-eligible-work case may quiesce after exactly reaching a bound, consistent with the stated contract.
5. **Cancellation, Close, and lock order.** Run/Close coordinate through `ownerMu`; Close releases it before joining or taking `mu` (`store.go:486–505`). Active writes take `mu` then briefly `ownerMu` for the wake signal, with no opposite nested acquisition in Close/Run. Run cancels transport, joins workers, accounts remaining candidates under its separate cleanup context, builds the report, and only then closes `runDone`. Close cannot release the database/OS lock underneath that owner. The new three-candidate Close probe verifies received outcomes persist and all are consumed on reopen without redispatch.
6. **Owned transactions.** Active-store write paths consistently use `beginOwnedTx`, retain deferred rollback, use cancellable SQL statements, and pass the commit cancellation gate. Detaching only the transaction lifetime avoids database/sql’s cancellation rollback discarding the pinned connection. Cancellation after the final gate can leave a complete durable transition; it cannot make it partial. The post-commit counter fix correctly distinguishes this from a failed commit.
7. **Late receipts and authority.** `recordExternalReceipt` (`external.go:247–302`) compares the completion’s attempt, retained intent, reconstructed request, retained authorization, and receipt binding before persistence. It intentionally validates the authorization’s original epoch instead of the current source activation. The source’s later closure does not grant a new generation or reopen the old run. Receipt persistence and observation settlement remain separate transactions. Runtime role/policy dispatch was not added; lifecycle policy remains ordinary program execution.
8. **Explicit failure/uncertainty.** Transport timeout/unavailability leaves retained authorized uncertainty instead of inventing non-application. A dequeued candidate whose persistence fails returns an explicit error and remains unresolved in durable records. Shutdown candidates are accounted after join. A volatile receipt is never described as durable merely because the worker received it.

## Acceptance tests and oracle examination

| Requirement | Actual fixtures and checked behavior |
| --- | --- |
| A: held independent progress | `TestAsyncHeldRequestPeerProgress`; `TestAutomaticAsyncHeldProgressAndConcurrentCrash` settle paths for both profiles. Real automatic Run/CLI; peer pre-commit gate releases only after exact endpoint application/hold acknowledgment; caller commits consumption before release. |
| B: saturation | `TestAsyncSaturationLeavesBacklogAndDeliversPeer`, capacities 1/2/3, capacity+1 external clients, 12 peer steps, ordinary terminal delivery, one-slot release allowing backlog, peak/contact checks. |
| C: reordered completion | `TestAsyncReorderedReceiptsConsumeWithoutHeadOfLineAndReplay`, b consumed before a is released, distinct totals 9/5, exactly two dispatches, no replay contact. |
| D: ordinary supervision and closure | `TestAsyncOrdinarySupervisorAdvancesWhileExternalHeld` uses NestedProvision; `TestAsyncAuthorizedTransportFinishesAfterOrdinarySourceClosure`; unchanged before/after-authorization closure coverage; old-epoch/forged-candidate test. |
| E: shutdown and budgets | Cancellation with a received candidate plus held request; overlap rejection/Close join; step and dispatch exhaustion with active/received outcomes; injected receipt-write failure; input wake; post-commit step accounting; reviewer’s all-three-received Close probe. |
| F: timeout/unavailable | `TestAsyncTimeoutAndUnavailableAllowProgressBeforeFailure`, peer commits before failure; Unknown/open source remains; storage-error injection stays explicit. |
| Concurrent crash/recovery | Both profiles × held-kill/received-kill, with independent endpoint process/store surviving. Idempotent original-key recovery has one application/two contacts; uncertain non-idempotent recovery has one application/one contact. Successful settle paths have one application/one contact. |

The subprocess test does not merely assert final projections. `startAsync` runs the actual `alan run` binary, and the resource starts in its own process/store. `child.wait` verifies the exact configured boundary fields, PID, scenario/start token, store, generation, epoch, sequence, intent, and authorization and checks durable boundary facts. The peer is paused at `execution.staged`; the controller releases that real transaction only after `resource.applied`. It never calls Commit. `execution.committed` acknowledgments for peer and caller carry the still-active authorization; a subsequent endpoint `holding` probe checks request-context liveness before response release. The post-record boundary plus final receipt consumption establishes the original result’s continuation.

`response-released` in the summarized transcript is a controller-command marker using the already-verified endpoint identity, not a newly emitted endpoint acknowledgment. Its causal force comes from the successful release write in the test followed by the verified settlement, and the endpoint code cannot send its response until that release. This is not a claimed additional physical observation.

For received-kill, `external.response/pre-record` is observed and the runtime receipt table is checked empty before SIGKILL. This qualifies received-but-unrecorded worker state; it does not claim that a volatile queue is durable. Existing accepted/rejected Retry, second-recovery-crash, terminal-delivery, and replay checks are retained; the old durable crash source differs only by the optional InFlight evidence field. Existing external/fault tests have mechanical keyed-options changes.

The false-order/identity controls reject early peer work, wrong operation/attempt/process, controller-commit substitution, absent active transport, timeout-first, and final-state-only evidence. I independently reconstructed raw multiline Go output in original arrival order and checked all 12 retained normal/race CLI traces for required causal order, active authorization, live hold identity, empty pre-release receipt table, expected crash uncertainty, endpoint applications/contacts, and child race mode. No event sorting was used to satisfy order.

## Independent verification and retained probe artifacts

The probes are overlays only; no Go source was changed in the worktree.

1. `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d1-go-build GOPROXY=off GOSUMDB=off go test -race -overlay=/tmp/phase1d1-review-probes/overlay.json -count=5 -timeout=70s -run '^TestReview' -v ./internal/executor`
   - PASS, ten executions total, no race report.
   - `TestReviewStalledBodyAllowsPeer`: partial response body at capacity1 -> peer commit -> response completion -> non-idempotent receipt consumption; Steps4, Dispatches1, no unresolved outcomes.
   - `TestReviewCloseAccountsAllReceivedCandidates`: three valid responses held before recording, concurrent Close, joined cleanup persists all three, reopen consumes all with Dispatches0.
   - Source: `/tmp/phase1d1-review-probes/probe_test.go`; overlay: `overlay.json`; full output: `result.log` in that directory.
2. From `/tmp/phase1d1-sync-negative-control`, with the same environment: `go test -count=1 -timeout=20s -run '^TestAsyncHeldRequestPeerProgress$' -v ./internal/executor`.
   - Expected FAIL, exit1, `held request ended before peer commit` (2.04-second test execution).
   - Full new output: `/tmp/phase1d1-review-probes/synchronous-red.log`.
   - Independently verified the negative control runner is byte-identical to the baseline runner and its held test is byte-identical to the current held test (SHA256 `5584d82893c0fb2663289dc5bbaaa019560486d80e00731007daff364503e88a`).
3. Verified each retained archive digest and every archived file against its control manifest for synchronous, inherited-pipe, cancellation-before-fix, and post-commit-before-fix controls. Results: `/tmp/phase1d1-review-probes/control-audit.json`.
4. Independent trace/gate/source audit and artifact digests: `/tmp/phase1d1-review-probes/audit-summary.json`. Canonical existing gate logs remain `/tmp/phase1d1-qualified-normal.jsonl` and `/tmp/phase1d1-qualified-race.jsonl`; the reviewer did not rerun the complete suites.

## Considered limits, not revision requests

- The cleanup context is five seconds after worker join; it is not a hard five-second wall-clock Close guarantee. Configured HTTP timeout/cancellation, OS scheduling, SQLite’s existing busy timeout, and context-respecting observer behavior remain relevant. The implementation report explicitly states this limit.
- The explicit synchronous DispatchExternal API is not part of Run’s automatic worker/budget bound. Concurrent external use of that separate API is outside the automatic-owner claim being qualified.
- Cancellation may race after a candidate is dequeued and before its durable write; explicit error plus retained uncertainty is permitted by the requirements. The implementation does not falsely report settlement in that case.
- Generic oracle utility checks are supplemented by exact `child.wait` checks and durable facts; assessing only `checkAsyncTrace` in isolation would omit those protections.
- The tests establish finite executions and declared durable invariants, not unbounded fairness or production/distributed exactly-once guarantees. Existing static-subject, single-host, endpoint-profile, and cross-store operation-key limitations remain.

No remaining required code/test work was identified. Parent should preserve these probe artifacts with the final evidence and obtain the separate whole-branch review after packaging.
