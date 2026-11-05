# Bounded execution work implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Work task by task, one implementation writer at a time, with independent reviews.

**Goal:** Remove redundant SQL-per-record probes and unrelated matching from the ordinary bounded-series execution path, while preserving causal evidence and recovery.

**Architecture:** Bulk retained reads and semantic indexes confined to one locked call; targeted owner-only continuation evidence reading. No persistent cache or scheduling policy change. Full reporting/audits remain explicitly costed.

**Tech Stack:** Existing Go 1.27.3, database/sql, go-sqlite3 v1.14.52; Python standard library for evidence.

**Spec:** [contract.md](contract.md), frozen before production edits following the user's instruction “Make it so.”

## Global Constraints

- Preserve original identities, authority, approval, ordering, independent expenditure, uncertainty, receipt conflicts and recovery.
- No new dependency, table, index, schema migration, persistent scheduling cache, global scheduler or public execution abstraction.
- Keep BoundedSeries and RunReport contracts; the sole permitted API addition is read-only `Store.ContinuationBasis(context.Context, RunID) (ContinuationBasis, error)`.
- No live providers, merge or release. Preserve prior evidence and all failures/rejected findings.
- One implementation writer; no nested subagents. Serial heavy measurements. No relabeling tests after source changes.
- Call-local reuse ends before mutations; existing fresh-owner/full-reader validation stays independent.

## Review Focus

1. Lazy readiness corruption errors and rotation with an absent last run (Task 2).
2. Closed/revoked roots with unresolved descendants/results and early non-success publication (Task 3).
3. Historical approval basis after later result, multiple paid predecessors and stale read-to-approve evidence (Task 4).
4. Old-format optional tables and active competing owner/readers (Tasks 2–5).
5. Workload output lower bounds and deceptive SQL-count savings that merely move work into scans (Tasks 1,5).

### Task 1: Freeze baseline, apparatus and invariants

**Files:** `docs/evidence/execution-scaling/{contract,plan,baseline,invariants}.md`, compact driver/counter overlay and baseline metadata; external raw logs.

**Interfaces:** Baseline d59eaf2976849e450635d252c7bca62eeccbef2e; existing P2 public driver/proof and P3 immutable evidence. Produce frozen source-backed invalidation analysis and compact reproducible baseline apparatus. No production changes.

- [ ] Record source/PR/clean worktree, environment and baseline tests, including blocked attempts.
- [ ] Finish source investigation of readiness, delivery, root/report and continuation evidence; record why bulk call-local indexing is chosen over owner-lifetime caches and a new scheduler.
- [ ] Build the existing public driver from a clean frozen export; retain 1,32,128,256 pilots and separate compact SQL/profile diagnostics where prior evidence is insufficient.
- [ ] Implement minimal diagnostic/driver overlay with input-anchor/hash checks and tests rejecting invalid modes, empty/failed rows and changed inputs. Preserve original driver/proof functionality.
- [ ] Freeze exact invariants and planned edits before production changes; commit, report and obtain independent specification/quality review.

### Task 2: Batch readiness and delivery reads

**Files:** `internal/executor/runner.go`, new focused retained-read helper/test files if needed. Do not modify authority, Commit or reducer semantics.

**Interfaces:** Keep `selectReady`, `pendingDeliveries`, `finishRun` signatures and public RunReport unchanged. Preserve original selector reference tests. Produce bulk run/status reads and grouped observation candidates, plus bulk retained delivery status reusable within a report call.

- [ ] Add differential tests against the frozen original selector/delivery paths: rotation, missing last, generations/retirement, closed runs, consumed/wrong-run observations, first-reached corrupt program/binding/projection, old-format absence, all effect kinds and unknown outcomes.
- [ ] Establish a diagnostic RED count for SQL-per-run/status probes and unrelated observation visits.
- [ ] Replace per-record queries with bulk inputs under the same Store lock; decode projections lazily in the original visited order, validate all observations as before, group only after validation. Preserve per-run binding and reducer checks while allowing bulk membership lookups at the same cut.
- [ ] Use bulk ack/settlement/delivery identity sets without assuming payload identity or hiding unexpected table errors. Prepare only still-unresolved invocations using existing semantics.
- [ ] Run focused ordinary/race/reference tests and verify exact choices/errors/reports plus lower work counts. Commit and independently review.

### Task 3: Batch root settlement and report relationships

**Files:** `internal/executor/composition_root.go`, `composition_inspect.go`, focused helper/tests. Touch descendant scope traversal only if measured cost requires it.

**Interfaces:** Keep PrepareRoot/SettleRoot/RootCompletion contracts. Consume retained root/freeze/closure/publication rows at one cut, same precedence as existing prepare helpers. Reports include all roots and exact charges.

- [ ] Add baseline/candidate comparisons for no freeze, known result awaiting settlement, successful root, early failed child, retired/revoked scope, pending descendant and unresolved external consequences; deliberately malformed rows must not become completion.
- [ ] Establish RED SQL-per-root and root×intent matching counts.
- [ ] Bulk root preparatory inputs and root completion data. Group sorted original intent IDs by authentic membership once; preserve deterministic root/operation ordering and independent unknown/result distinctions.
- [ ] Avoid deeper settlement preparation only from retained publication or absence of freeze/closure at the same cut; final settlement still uses existing transaction checks. No cache survives the call.
- [ ] Run focused composition, receipt/reconciliation, lifecycle and report tests ordinary/race and compare retained outcomes. Commit and independently review.

### Task 4: Read targeted continuation evidence

**Files:** `internal/executor/continuation.go` or a focused reader file, `runtime` public-package tests, `docs/continuations.md` (existing documentation path verified before edit), and new evidence driver overlay.

**Interfaces:** `func (s *Store) ContinuationBasis(ctx context.Context, target RunID) (ContinuationBasis, error)`. Existing snapshot method and ApproveContinuation unchanged. No new public type.

- [ ] Write public tests comparing against full snapshot for unspent, paid-but-unadmitted, unresolved, refused/successful result, settled/selected and already-approved predecessor states.
- [ ] Return original committed approval basis first; otherwise find exactly one allowed original paid operation for the declared predecessor/effect via retained identities. Reject ambiguity, wrong generation/namespace/correlation and malformed evidence; never infer approval.
- [ ] Use owner lock/coherent read scope, no writes/activation/dispatch. Test read-only durable-state equality, no adapter registration, changed result after basis, duplicate approval, new owner and concurrent completion.
- [ ] Exercise the targeted method in the same independent public workload; preserve final inspection/Replay/proof and report validation, with an explicit snapshot/targeted mode field.
- [ ] Document targeted owner read versus independent full audit and the read-to-approve stale check. Commit and independently review.

### Task 5: Freeze, qualify, measure and deliver

**Files:** compact results, reproducible campaign/qualification runner, review reports, README/reference links under `docs/evidence/execution-scaling`; external immutable raw logs/profiles/databases.

**Interfaces:** Reviewed frozen production/test/driver inputs; exact source and binary hashes; contract's unchanged and targeted-mode comparisons.

- [ ] Differential deterministic direct/controlled HTTP histories against baseline: operation IDs, reports, approvals, paid/contact counts, exact results/observations, Replay and inspection. Reuse acknowledged genuine SIGKILL/reap fixtures at approval, commitment, reservation/admission, lost/retained response, result selection, terminal and repeated reopen; copied corrupt/foreign cases remain separate.
- [ ] Run historical compatibility/external eight-operation consumer and competing owner/reader/authority suites. Preserve failures and old known flaky evidence separately.
- [ ] Freeze measurement schedule/tool hashes, execute serial fixed samples, separately collect compact counters/profiles. Analyze SQL, rows/VM, matching work, elapsed/CPU/allocations/RSS and per-observation trends; explicitly retain full-output quadratic floor.
- [ ] Freeze source; run full ordinary/integration/integration-race plus focused race, build/vet/module/tidy/format/Staticcheck gates. Record exact exits/skips and before/after source hashes.
- [ ] Obtain independent measurement and final whole-change reviews; resolve substantive findings with affected requalification. Freeze final delivery identity with evidence-only hash bridges if needed.
- [ ] Verify current PR ancestry; push the new branch and create a separate draft PR with clear dependency/base/head and reproducible evidence. Do not merge or release.
