# Task 3 independent specification and quality review

Reviewed 2026-10-25 against `task-3-review-package.diff`, the Task 3 brief/report, current binding durability contract, and prior task review findings. All 22 frozen file identities and byte counts match the source. Read the new lifecycle execution/reconstruction/program/test files and all supporting store, replay, messaging, runner, inspection and CLI changes. No repository source edits or subagents; no duplicate full-suite run.

**Specification verdict: PASS for Task 3.**

**Quality verdict: PASS for Task 3.** No blocking scoped defect was found. This approves the lifecycle slice for continuation, not the complete Phase 1D durability qualification.

## Protocol and transaction findings

- The five required boundaries remain distinct public operations: ordinary Commit, source AuthorizeIntent, target DecideLifecycle, SettleLifecycle, and ordinary consumption Commit. Target mutation/result retention shares one transaction; outcome admission/observation/settlement is another transaction. Target rollback and settlement rollback tests exercise separate failures rather than a combined helper.
- Decision lookup compares the complete retained request and returns the original accepted or rejected result before reevaluating current target circumstances. The changed-eligibility test genuinely rejects while active, then commits failure, retains the old rejection, and allows a separately committed operation to accept. Expected-generation guards do not mask this test.
- Source authorization is distinct from lifecycle target grants. An already allowed authorization remains usable after requester closure. New authorization following lifecycle closure retains the exact closure operation as evidence; source denial does not create a target decision. Settlement distinguishes source denial from target rejection.
- Target checks cover explicit grants, expected generation/revision, retirement and operation eligibility. Late CancelRun preserves committed failure; RetireSubject blocks successors; cancelled retry eligibility follows explicit subject configuration. MaxInt64 generation exhaustion returns an operational error atomically rather than overflowing or inventing an immutable semantic rejection.
- Outcome observation correlation uses the original request's target generation, requester execution, obligation and operation. This permits ordinary consumption of a legitimate stale-generation rejection. Settlement can record late evidence for a closed requester without reopening its execution.
- Lifecycle admission extends the trusted domain namespace; raw caller IDs remain data, and the prior transport-domain fix is preserved.

## Reconstruction findings

Root origins preserve the explicitly provisioned generation, original artifact and initial projection. There is no implicit assumption that the initial generation is 1. Retry-created runs require their exact accepted operation and predecessor link; accepted Retry results must have the corresponding successor, including at zero committed instructions.

Ordinary replay is still strict. Authority-induced closure is reconciled only through an exact accepted result, before/after projection, unchanged execution sequence, and terminal obligation linkage. Lifecycle reconstruction checks decision-order subject generation/retirement, target before-state against retained ordinary history/earlier closure, static grants, recomputed decision and successor provenance. The tests reject inconsistent creation/result/closure links before epoch renewal. Sequence-zero CancelRun/AbandonRun/RetireSubject closure is represented directly rather than fabricated as an ordinary instruction.

The report accurately describes this as consistency reconstruction sharing authority evaluation with the runtime. It does not claim arbitrary coherent tampering detection, and it explicitly retains the need for an independent before/after oracle to detect loss of an otherwise unreferenced rejection result. I found no new permissive projection-mismatch bypass.

## Ordinary nested-policy findings

The worker artifact receives explicit input from the inner execution, branches on its mode, and ends ordinarily; it does not branch on worker generation. Inner policy branches on authenticated failure and its own ordinary limit/spent fields, reserves spent plus pending ID in Emit, consumes the exact result, advances expected generation from the result, sends explicit successor input, and waits again. The same inner execution retains the obligation and budget across worker generations and physical reopen. Outer sees only inner's authenticated public outcome and applies its explicitly chosen zero-retry policy.

No runtime branch uses the fixture names, ancestry, private policy slots or a supervisor opcode. Renamed-program and budget-exhaustion tests supplement source inspection. The historical research model remains independent from these runtime fixtures and their reducer.

## Independent focused probes

Copied the frozen source to `/tmp/alan-task3-review-probe` and added reviewer-only probes there. Exact source: `/tmp/alan-task3-review-probe/internal/executor/review_probe_test.go`. Exact output: `/tmp/alan-task3-review-probe.log`.

Command: `GOCACHE=/tmp/alan-phase1d-go-build go test -v ./internal/executor -run '^TestReview'` in the isolated copy. Result: PASS, 1.230s.

1. **Three retries with reopen after each bounded invocation.** Renamed nested subjects o/i/w, RetryBudget=3, explicit successor input=fail, MaxSteps=1. Repeated actual Close/Open and Replay while work progresses. Final inner spent=3 and expected worker generation=4; worker g4 has three actual committed instructions and failed; outer escalates failed. The inner remains generation1 through epoch53. Exactly three lifecycle results exist. This exercises budget/continuation preservation beyond the supplied one-retry success fixture.
2. **Varied lifecycle interleavings.** Five fixed pseudorandom seeds, 100 actions each, with three ordinary participants issuing Retry/CancelRun/AbandonRun/RetireSubject, including self-targets. Actions use public Commit/Authorize/Decide/Settle/Open operations. Replay is checked after every attempted action. All resulting states reconstruct and reopen. This is a bounded consistency probe, not an independent policy oracle or exhaustive search; unsuccessful operations are permitted and must leave reconstructible state.
3. **Actual service-message binding across Retry and closure.** Deliver a generation-0 service message to worker/g1, close its sender after authorization, commit worker failure, accept real Retry to g2, Close/Open, and retransmit the original message transport. It retains the exact g1 observation/binding and ACK succeeds. In the same setup, a valid ordinary input whose raw ID is the future automatic lifecycle-delivery string coexists with the authenticated lifecycle settlement under its distinct domain. Replay validates both. No SQL manufactures generations or protocol setup.

## Evidence and remaining gates

The task report retains feature RED logs, disclosed postimplementation boundary coverage, full normal tests, focused race/vet results, and compiled CLI smoke evidence. The original service-binding cross-slice requirement is now exercised with real Retry in the implementation tests and the additional message-specific reviewer probe.

Expanded acknowledged subprocess kills at the lifecycle boundaries, the second crash during same-obligation recovery, independent policy/research conformance, external endpoint uncertainty/idempotency, fault controls, historical reproduction, and final source/dependency archive verification remain later gates. No claim of unbounded liveness, fresh-supervisor reassociation, arbitrary-depth equivalence, or power-loss qualification follows from this Task 3 approval.
