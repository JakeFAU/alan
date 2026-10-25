# Phase 1D.1 bounded asynchronous dispatch implementation plan

> Use delegated development with one owner for coupled scheduling, transport, completion and shutdown. The user authorizes implementation after this plan.

**Goal:** Observe exact endpoint hold -> independent ordinary peer commit -> peer result delivery -> response release -> original result settlement.

**Architecture:** One authoritative scheduler and a bounded set of network-only workers. Immutable correlated completions return to authoritative validation/persistence. Undispatched work remains in the existing durable ledger.

**Stack:** Go 1.27.3, existing go-sqlite3 v1.14.52. **Spec:** [complete request](requirements.md). **Contract:** [dispatch contract](01-dispatch-contract.md).

## Baseline

Original checkout dc119e00f6b8a6629011bca36a7f0b9f20456f28 has only an untracked user-extracted docs/foundational/phase1d/source directory; preserve it. PR #4 is merged. Fresh fetch verifies research base `6effdc02259af5335633bfc41e9786386430fa7c`, with identical tracked tree. The original 53-file source digest is `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`. Dedicated branch codex/phase1d1-async-dispatch. Run currently waits synchronously in DispatchExternal before selectReady; its timeout test observes only final state.

## Global Constraints

- One host, one active executor per store, static subjects/grants, current single-PC-per-run representation.
- Preserve intent commit -> authorization -> separate application -> receipt persistence -> ordinary consumption; network workers cannot mutate policy or durable state.
- One Run owner; no store mutex/transaction/scheduler wait spans network I/O, slot waiting or completion backpressure.
- Keep source-versus-target authority, late old evidence, replay and existing idempotent/non-idempotent recovery behavior.
- No unbounded goroutine fanout, busy-spin or application cancellation inferred from physical shutdown.
- Preserve prior source/evidence/counterexamples. No business-key namespace repair, providers, SDK, UI, multi-host, dynamic creation, transfer, compaction or migration.

## Review Focus

1. Cancellation/budget exit racing with a received completion: retain evidence or explicit uncertainty/error; no post-close access.
2. Forged/mismatched completion correlation versus authentic late evidence from obsolete epochs.
3. Saturation/duplicate scans/backpressure cannot block local or internal lifecycle work.
4. Held oracle rejects early peer commits, timeout/cancel first, wrong request identity, manual Commit and final-state-only proof.
5. Give local commits/internal delivery/completions bounded opportunities rather than perpetual delivery priority.

## Task 1: Coupled dispatch repair and qualification tests

Files: internal/executor/{runner,external,store}.go and focused tests; cmd/alan; narrow boundary/resource/controller additions as required. One implementer owns these coupled changes.

- [x] Reproduce baseline from immutable source before behavior changes; separately record historical versus current comparisons.
- [x] Add unchanged held-request acceptance checks first; retain behavioral RED on synchronous source with complete input/trace and a reproducible negative control.
- [x] Bound active transports (default 2); immutable candidates return to authoritative validation. One active Run, nonblocking slot reservation, bounded completion capacity, regular category opportunities, explicit in-flight wait/quiescence.
- [x] Reserve MaxDispatch before launch; MaxSteps counts ordinary commits. Cancel/drain with bounded cleanup, join before return/Close/releasing exclusivity; physical stop is not logical cancellation.
- [x] Tests A–F from requirements: actual CLI held progress; saturation including capacity 1; reordered completion; ordinary supervision while held; closure before/after authorization; shutdown/budgets/queued outcomes; timeout/unavailable endpoint.
- [x] Concurrent crash cases for both endpoint profiles, plus received-but-unrecorded completion. Preserve all Phase 1D crash assertions.
- [x] Focused/full/race including child binaries/vet/gofmt/diff; freeze source and report complete causal evidence for independent scoped review.

## Task 2: README, evidence and publication

Files: README.md, docs/foundational/phase1d1/*, append-only foundational decisions.

- [ ] Create README with project purpose/status, prerequisites, quick start, nested and external examples, inspect/replay, tests/research map and honest limits.
- [ ] Publish concrete dispatch/shutdown contract, updated runbook and evidence-to-claim report; preserve raw causal order and legitimate defects.
- [ ] Exact source/dependency manifest and complete archive; reproduce from empty module cache. Preserve earlier qualifications unchanged.
- [ ] Independent whole-branch review, necessary correction/re-review, commit/push dedicated branch and draft PR against freshly verified research base. No merge/release.

## Alternatives and stopping point

Per-intent goroutines are rejected because blocked producers are still unbounded. Shorter synchronous timeout does not meet the contract. The selected bounded transport set leaves SQLite as short-operation authority and needs no framework or new calculus. Stop at reviewed implementation, README, reproducible evidence and draft PR.
