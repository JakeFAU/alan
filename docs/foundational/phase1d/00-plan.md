# Phase 1D first durable executor implementation plan

> **For agentic workers:** Use delegated development, tests first, and independent review. The user authorizes implementation after this plan; no further design approval gate applies.

**Goal:** A new process reopens a local store and continues the same ordinary supervision obligation without losing or duplicating its committed Retry decision.

**Architecture:** A small internal Go executor interprets versioned finite data-driven programs and serializes individual protocol boundaries in SQLite. A generic runner advances independently eligible runs and committed obligations. A separate loopback fixture owns external mutation state. Research code remains an independent test-only reference.

**Stack:** Go 1.27.3; standard library plus pinned `github.com/mattn/go-sqlite3 v1.14.52` (CGO and a C compiler required). No ORM, broker, storage abstraction, or migration framework.

**Contract:** [01-durability-contract.md](01-durability-contract.md). The Phase 1D user request governs any conflict. This is a provisional single-host implementation, not a public SDK or distributed runtime.

## Baseline and source preservation

Initial checkout: `3e003c91d35af7055f5be99aa9c79c74f3ea44db`, clean. Remote verification found PR #3 merged and its head branch removed. Implementation base is `a0d337c7b4f7340a55dea11b619cf9b22c25e2b0`, the current `codex/phase1b-recursive-composition` tip, whose tree is identical to the initial checkout. `origin/main` is `97fd14eb1d1d261d4274de04f38a2d9432ad0397` and lacks the later research phases. Branch: `codex/phase1d-durable-executor`.

The baseline Phase 1C source digest is `08a107be1b4eff35986ab605dd1ef77db660baa6fb8e920f2d88a8f3031b859b`. Reproduce prior saved evidence from original source archives before behavior changes, then compare current-source records separately. Preserve all previous source, negative controls, counterexamples, and the same-execution/ordered-interruption oracle regression. Adding a dependency changes current research manifest identity through `go.mod`; it does not retroactively change historical evidence.

## Selected implementation choices and alternatives

- New runtime code under `internal/executor`, with focused source files for programs, store, protocol, runner, replay, and inspection. `cmd/alan` supplies init/run/inspect/replay/input operations. `internal/fixture` and `cmd/alan-resource` provide the independent endpoint. Integration controllers live outside runtime policy code.
- One pinned database connection per executor, checked WAL/FULL settings, local file lock for exclusivity. A connection pool or several execution writers would add connection-setting and concurrency obligations without helping this first profile.
- Normalized durable ledgers with JSON for immutable descriptors and serializable projections. Reject a single serialized world blob: it obscures the independently crashable protocol boundaries. No helper spans an external application and runtime settlement transaction.
- Closed generic instruction set: finite set/copy/add, conditional jump, receive, send/effect, and terminal result. Operands and correlation refer to ordinary data. No supervisor opcode, role dispatch, unrestricted callback, or generation-based worker-success shortcut.
- One existing intent may be reattempted with its stable endpoint key under the explicitly idempotent profile. Non-idempotent uncertainty remains visible and is separately demonstrated. No business-level deduplication is inferred from intent IDs.
- Use a new implementation rather than promoting the research interpreter. Conformance checks compare selected public projections and transition obligations; replay consistency using the runtime reducer alone is not independent conformance.

## Global constraints

One host, local filesystem, one active executor per store; static subjects and grants; immutable locally retained program artifacts; source authorization separate from target authority. No dynamic creation, ownership transfer, general capability delegation, multi-host fencing, distributed consensus, public SDK commitment, provider integration, migration, hierarchy-wide budget, cleanup custodian, or fresh-supervisor child reassociation.

Policy choices and budgets remain ordinary computation. Process restart advances activation epochs without changing generation, obligation, budget, pending operation, or consumed logical IDs. Both accepted and rejected lifecycle results are immutable. Replay has no dispatch authority. Crash tests kill subprocesses after acknowledged barriers, never claim a power-loss test.

## Review focus

1. Closure between source intent commitment and authorization; already authorized uncertainty survives closure.
2. Identical rejected operation retransmitted after target eligibility changes; unrelated guards must not mask the retained-rejection contract.
3. Recovered ordinary state and selected observations disagreeing with history or a retained artifact; fail explicitly.
4. Duplicate/late observations, stale epoch/sequence, and a service-message retry after generation change.
5. Failpoint acknowledgment occurring before the named persistence boundary, or a controller reconstructing state instead of recovering it.

## Task 1: Durable execution commit and reopen

Files: `internal/executor/{program,store,replay}.go`, corresponding contract tests, `go.mod`, `go.sum`.

- [x] Specify schema and transaction boundaries in the contract before code.
- [x] Write failing tests for atomic execution commit, selected-observation consumption, immutable artifacts/intents, stale authority, reopen, and corruption/version rejection; retain RED results.
- [x] Implement the pure finite reducer and store, connection verification, exclusive activation and same-run reopen.
- [x] Reconstruct histories without effects and compare projection/intent descriptors. Test corrupted records, conflicting identities, second-process exclusion, and runtime limits distinct from research bounds.
- [x] Run focused tests and review before extending the protocol.

## Task 2: Automatic runner and ordinary messaging

Files: `internal/executor/{runner,messages,inspect}.go`, `cmd/alan/main.go`, corresponding tests.

- [x] Test independent progress, automatic finite execution, retained message admission/binding, terminal notification persistence, and explicit inputs.
- [x] Implement deterministic eligible selection and serialized commits, with bounded operational dispatch and visible unresolved work. No harness-issued Prepare/Commit is needed for application progress.
- [x] Add inspection and CLI starting/reopening/input commands. Keep permission checks and origin authentication at admission.
- [x] Review ordinary selection, fairness limitations, and message acknowledgment loss.

## Task 3: Persisted ordinary lifecycle protocol

Files: `internal/executor/lifecycle.go`, ordinary program fixtures, lifecycle and supervision tests.

- [x] Test ordinary failure → Retry intent/budget reservation → source authorization → target decision → settlement → consumption.
- [x] Implement retained accepted/rejected results, expected-generation/revision/grant checks, CancelRun versus RetireSubject, source closure rules, and immutable retransmission.
- [x] Execute real successor instructions. Test observer-without-authority, duplicates, old results, physical interruption, and bounded nested supervision.
- [x] Review the central accepted-result recovery case and changed-eligibility rejection case.

## Task 4: Independent endpoint and crash recovery

Files: `internal/fixture/*`, `cmd/alan-resource/main.go`, subprocess integration controllers and failpoint support.

- [x] Test separate durable application, retained idempotent key/parameter matching, and explicit non-idempotent ambiguity through the public endpoint protocol.
- [x] Add failpoints that only report/pause at real boundaries. The controller waits for identity-bearing acknowledgment then kills without graceful shutdown.
- [x] Cover uncommitted local transaction, post-intent/pre-dispatch, post-authorization, application/missing receipt, accepted/rejected lifecycle results, terminal notification, and a second crash during recovery.
- [x] Retain exits, acknowledgments, post-restart inspection and independent checks. Add three test-only faults: rejection loss, recovered policy-state reset, and uncommitted dispatch; unchanged checks must detect them non-vacuously.
- [x] Review the harness independently of runtime tests.

## Task 5: Runnable nested demonstration, qualification, review

Files: demo program definitions/CLI, `docs/foundational/phase1d/*`, append-only decisions update, reproducibility tooling and complete source archive.

- [x] Demonstrate normal completion, same-run recovery, retained rejection, missing receipt, and replay without dispatch using documented commands.
- [x] Record platform, exact source/dependency identities, commands, results, source snapshot and machine-readable evidence. Preserve every valid discovered defect and original regression input.
- [x] Run focused tests, full suite, race checks, vet, gofmt, historical reproduction and current-source comparison.
- [x] Obtain independent adversarial review of complete source, contract, evidence, transaction boundaries, and controller. Correct defects and retain findings.
- [x] Commit the reviewed increment, push, open a draft PR against the freshly verified current research base. Do not merge or release.

## Operational bounds and stopping point

Program instruction count, per-step work, serialized payload size, database growth and per-runner invocation work limits are explicit implementation limits, not architectural maxima of 0–8 values or three generations. Exact chosen defaults and qualification fixtures are recorded alongside evidence. Automatic execution may return quiescent with unresolved inputs/effects; this is not successful completion or eventual-recovery evidence.

Stop with the runnable reviewed Phase 1D increment, real process-kill qualification, full source/dependency snapshot, and draft PR. Any implementation adjustment to this plan is recorded with its reason and cost.

Publication: implementation commit `121b504591509e1879b9cdd7eea0eb1b6280d613`; [draft PR #4](https://github.com/JakeFAU/alan/pull/4) targets verified base `a0d337c7b4f7340a55dea11b619cf9b22c25e2b0` on `codex/phase1b-recursive-composition`. No merge or release was performed. A documentation-only follow-up records this publication; the qualified Go source digest is unchanged.
