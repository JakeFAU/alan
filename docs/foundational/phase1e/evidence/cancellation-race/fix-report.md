# Cancellation ownership correction

Status: DONE; runtime/test source frozen, no commits made. Independent review and whole-source final qualification remain the parent's gates.

## Diagnosis

The exact failed source is `026321f39dce9332d24b4891a2e9164f0d14aaacdbecec69058989341e8196a8`. Its source archive, source manifest, full default/normal/race outputs and commands were present under `docs/foundational/phase1e/evidence/cancellation-race` before any edit. The original `TestAdapterCancellationAccountsReceivedCandidate` remains byte-for-byte unchanged.

Both live receive sites in `Run` removed a completion from the buffered channel and decremented `active`, then called `accountExternal` with the cancelable runner context. Cancellation before receipt persistence makes the SQLite operation return canceled. Deferred shutdown can drain only the remaining active slots; the already-owned completion is no longer queued or active. The same issue affects fixture receipts and adapter candidates. Cancellation during the separate result-recording / settlement commits can similarly interrupt owned accounting.

## Change

- `internal/executor/runner.go`: both live dequeue sites call one `accountReceived` helper. It reports the observational `runner.received` / `pre-account` boundary using the original request lifetime and the original event identity. It then accounts the owned result with `WithTimeout(WithoutCancel(ctx), runnerCleanupTimeout)` (existing five-second budget), preserving observer error, accounting error and original context error with `errors.Join`. This detached context is never used to prepare, authorize, dispatch or run ordinary program work. Deferred draining retains its existing shared bounded cleanup context. There is no retry of a completion and no change to durable result/settlement transactions.
- `internal/executor/cancellation_regression_test.go`: deterministic cancellation after dequeue for both adapter and real loopback fixture; cancellation between adapter result recording and settlement; a failpoint returning an accounting error while cancellation occurs. Assertions cover one result/settlement/observation, preserved cancellation, no ordinary advancement, original contact count, offline replay, recovery without repeated transport, and known-pending versus Unknown distinction. The mid-settlement check also observes the bounded accounting deadline.
- `internal/evidencetest/process_test.go`: a separate actual CLI regression retains the original SIGTERM-at-adapter-response test and adds SIGTERM after owner dequeue. The controller checks existing full boundary correlation, snapshots the durable store before/after, and verifies the original attempt/result plus settlement, no new history or attempts, no lookup or additional provider contact after cancellation, then recovery without repeating the first model call. Full snapshots are emitted before assertions as well as at success.

## Red evidence and reproduction

`deterministic-red.log` shows both dequeue subcases failing with "dequeued evidence lost during cancellation", the mid-settlement case failing, and the actual CLI losing its received candidate. `deterministic-red-snapshots.log` repeats the same red command with full pre/post durable snapshots emitted before the failing assertion. Both exited 1. No test assertion was relaxed.

Recreate the red instrumentation from `before-source.tar.gz`, apply `red-observer-and-cli.patch`, then copy `red-cancellation-regression-test.go.txt` to `internal/executor/cancellation_regression_test.go`. `red-runner.go.txt` is also retained explicitly. The patch adds only an observational dequeue seam while preserving the bad canceled-context accounting. The pre-snapshot and snapshot variants differ only by additional evidence emission.

All exact commands, exit codes and log digests are in `correction-verification.json`; commands use the explicit local/offline Go environment requested in the task. The final bounded source delta including the new test file is `correction.patch`. The final changed-file hashes are in the verification JSON.

## Verification

- Red command, twice: `go test -tags integration ./internal/executor ./internal/evidencetest -run 'TestCancellationAfterDequeue|TestCancellationDuringAdapterSettlement' -count=1 -v` — expected FAIL (exit 1), preserved above.
- Focused green command including original cancellation, both actual CLI windows, receipt persistence and settlement failure controls, plus new regressions — PASS.
- Complete relevant normal packages: `go test -tags integration ./internal/executor ./internal/evidencetest -count=1` — PASS, executor 3.763s / evidencetest 4.067s.
- Original plus deterministic cancellation race stress: `go test -race -tags integration ./internal/executor -run 'TestCancellationAfterDequeueAccountsCompletion|TestCancellationDuringAdapterSettlement|TestCancellationAccountingError|TestAdapterCancellationAccountsReceivedCandidate' -count=100` — PASS, 12.073s.
- Complete relevant race packages: `go test -race -tags integration ./internal/executor ./internal/evidencetest -count=1 -v` — PASS, including race-instrumented actual CLI/provider subprocesses and full EVIDENCE snapshots.
- `git diff --check` and gofmt inspection of all three files — clean.

## Boundaries and limits

The deterministic shared helper seam covers the cancellation window for both completion kinds; both live dequeue sites call precisely that helper. Tests do not introduce a sleep or poll to establish ordering. The original asynchronous cancellation test is also stressed unchanged. No result is re-enqueued or recounted, so active-slot retirement remains one per receive.

The five-second context is an accounting budget, not a claim that all shutdown completes within five seconds. Existing worker join semantics still require cooperative trusted adapters; observer callbacks likewise remain trusted. Deferred cleanup/reporting has its existing separate bounded context. Deadline or persistence failures still return explicitly and may leave Unknown if recording failed, or known-pending if a result was retained but settlement failed. This correction does not guarantee persistence despite storage failures or process death. The explicit synchronous DispatchExternal API is unchanged; this correction is scoped to the automatic Run owner identified by the failed qualification.

The separately approved FR1 OpenAI fix, parent documents/tools, original failure evidence and all existing assertions were left intact. Parent should run the whole source-matched qualification and independent review before publication.


## Fix round 1 — CR1 acknowledgment correlation

Status: DONE; source frozen for parent qualification and scoped independent re-review. No commits. Runtime ownership/spec was independently PASS; this round corrects only the P2 monitor finding.

Read the complete cancellation review and verified the precise omission: the new actual CLI dequeue test used a filter with wildcard epoch and sequence and never called the existing durable checker. Positive epoch/sequence, intent ID and authorization ID did not establish equality with the retained source authority and committed intent.

Changed only:
- `internal/evidencetest/process_test.go`: immediately after `before := inspect(t, db)` in `TestCancellationAfterDequeueRetainsRealModelCandidate`, require `validateDurableBoundary(before.Snapshot, ack)` to succeed. The original wildcard filter and every existing assertion remain unchanged.
- New `internal/evidencetest/cancellation_control_test.go`: `TestCancellationWildcardBoundaryRejectsDurableSubstitutions` accepts the exact original stopped input, then separately changes epoch by +7 and sequence by +9. All three pass the original wildcard acknowledgment matcher; the durable check accepts the original and rejects each substitution. The actual wildcard filter is explicitly retained, not fully specified to manufacture rejection. Each case emits the exact filter, acknowledgment, durable tables and verdict as EVIDENCE.
- New `internal/evidencetest/testdata/cancellation-dequeue-boundary.json`: byte-identical copy of the reviewer's `strengthened-original-cli-stopped-input.json`, retaining PID 60461, epoch 2, sequence 15, the full original snapshots and contacts. No fixture fields were rewritten.

New round evidence is isolated under `.superpowers/sdd/00-plan/evidence/cancellation-fix-round1/`; the prior correction records are untouched. `fix-round1.patch` contains only this round's source delta. `verification.json` records exact commands, outcomes, log digests, changed-file hashes, unchanged runner hash and fixture identity.

Red/green verification:
- Added substitution test before fixing the monitor: `go test -tags integration ./internal/evidencetest -run '^TestCancellationWildcardBoundaryRejectsDurableSubstitutions$' -count=1 -v` — expected exit 1; original accepted, separate epoch and sequence cases fail with `durable rejection=false, want true`. Exact red test source retained as `control-red.go.txt`, output `control-red.log`.
- Re-ran the actual CLI with the reviewer's observation-only runner event overlay (`Epoch += 7; Sequence += 9`) before the fix: the entire cancellation test incorrectly PASSed, exit 0; `mutant-before.log`. The exact before monitor and runner source plus overlay are retained. This reproduces the real monitor counterexample, not a runtime accounting failure.
- After the immediate durable assertion, the same actual CLI mutant fails at `boundary not bound to durable source authority`, exit 1; `mutant-rejected.log`. Production runner source was not edited.
- `go test -tags integration ./internal/evidencetest -run 'TestCancellationAfterDequeueRetainsRealModelCandidate|TestCancellationWildcardBoundaryRejectsDurableSubstitutions' -count=1 -v` — PASS, 2.893s (`normal-green.log`).
- Same covering command with `-race` — PASS, 7.714s; six `race_children:true` records establish actual child instrumentation (`race-green.log`).
- `gofmt -l` for both Go files and `git diff --check` — clean.

All commands retain `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off PYTHONDONTWRITEBYTECODE=1`. Mutant commands use GOFLAGS overlay so the actual compiled CLI inherits the mutated event instrumentation; runtime bytes remain unchanged. The parent owns whole-source qualification, published evidence packaging and independent re-review. There are no new runtime semantics or provider changes in this round.
