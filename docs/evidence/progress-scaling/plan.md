# Execution progress separation implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Execute task-by-task with independent review.

**Goal:** Advance execution without implicitly constructing a full historical report.

**Architecture:** One private runner, two explicit result contracts. Full Run unchanged; RunProgress skips only its final report traversal. No validity cache.

**Tech Stack:** Go 1.27.4, github.com/mattn/go-sqlite3 v1.14.52, existing SQLite formats and single-host owner.

**Spec:** [contract.md](contract.md).

## Global Constraints

- Baseline e43e4f6be4500d177b50caf29181608d09b99c62; preserve existing Run, Open, Replay and InspectExecution contracts.
- Preserve all execution, authorization, approval, reservation, admission, accounting, settlement and cleanup checks.
- No dependencies, migration, new cache, scheduler, ownership, automatic retry or BoundedSeries API change.
- Preserve research evidence and rejected findings; draft PR only, no merge/release.
- Progress is not a retained-history validity certificate, completion proof or remote-finality claim.

## Review Focus

- Cross-method exclusion and Close/cancellation still wait for result accounting (Task 2).
- Unrelated corrupted history has explicitly different diagnostics; reached corrupt results still fail (Task 2).
- Static discovery fallback, changed authority and historical stores gain no new shortcut (Tasks 2–3).
- Benchmark arms do identical semantic work and equal checks; final audit costs remain visible (Tasks 1,3).
- Quiescence with unresolved provider operation differs from completion, including after SIGKILL (Tasks 2–3).

### Task 1: Reproducible measurement and baseline

**Files:** create `docs/evidence/progress-scaling/measure.py`, `test_measure.py`, minimal driver templates if needed. Reuse tracked P2/P5/P4 helpers unchanged. Raw outputs only in this plan's ignored workspace.

**Interfaces:** CLI accepting frozen source revision, output directory and full/progress mode; serial balanced sample design; JSON rows/manifests and exact command logs. Baseline full builds without new API. Candidate progress uses generated typed helper, no reflection. Capture audits from `.superpowers/sdd/plan-progress-scaling/{semantic,cost}-audit.md` as design inputs.

- [ ] Test deterministic ordering, output/source binding, failed-row preservation and unique-anchor driver transforms.
- [ ] Reuse public deterministic P2 workload, targeted owner basis, final rich verifier/reopen/Replay. Remove per-prefix report proof equally in all arms; same cheap progress/contact checks. Final full report separately timed, roots compared with typed cut. Completed reopen asserts zero work/no contact; Replay root equality retained.
- [ ] Export frozen sources and build exact inputs with commands/exits/hashes and post-exit RSS. Primary timing uninstrumented; profiles/counters separate. Reuse report-scoped probes instead of new tracing architecture.
- [ ] Run apparatus tests and fresh baseline excluded smoke/profile B256 before production changes; retain baseline ordinary result. No candidate performance claim yet.
- [ ] Commit apparatus/baseline metadata after complete diff review; independent task review.

### Task 2: Shared runner and explicit progress result

**Files:** modify `internal/executor/runner.go`, `runtime/runtime.go`, `examples/weather-agent/main.go`; add `runner_progress_test.go`, needed integration tests, public runtime example/test; README and `docs/execution-progress.md`.

**Interfaces:** `type RunProgress struct { Steps, Dispatches int; Quiescent bool }`; `func (s *Store) RunProgress(ctx context.Context, options RunOptions) (RunProgress, error)`; runtime alias. RunOptions/RunReport unchanged. One private engine with explicit full-report choice.

- [ ] Add failing tests for API, genuine full/progress equivalence, documented unrelated-history diagnostic difference. Original compact-assessment tests unchanged.
- [ ] Mechanically move Run body to shared private engine. Full Run forwards unchanged. Progress projects three counters; same cleanup/ownership and quiescence predicate. Only full mode invokes unchanged finishRun.
- [ ] Test invalid limits, cancellation, limits, cross-method ErrRunActive, Close awaiting adapter, received-result retention/joined errors, reached corrupt result/no extra contact, unknown effects and static/fallback paths. Genuine positives use runtime operations, not fabricated SQL.
- [ ] Change only weather operate's discarded-report call; retain before/after inspection and ingress. Public example explains per-call counts and explicit inspection.
- [ ] Run focused ordinary/integration/race and old report oracle. Commit after complete diff review; independent semantics/recovery/API task review.

### Task 3: Frozen qualification and delivery

**Files:** extend new apparatus only as required; add compact results/qualification/README/reviews/evidence records under `docs/evidence/progress-scaling`. Raw logs/stores/profiles retained in plan workspace.

**Interfaces:** reviewed Task 2 API and Task 1 CLI → exact frozen revision qualification, three-arm comparison and draft PR evidence.

- [ ] Freeze clean executable/tests/tools revision. Bounds1,8,16,32,64,128,256; one warmup plus six balanced serial measured samples per arm. All six permutations of three arms per bound. Retain profiles/counters separately. Verify256 charges/contacts/results and255approvals.
- [ ] Same-seed full/progress typed evidence comparison includes refusal, unknown third then separately approved fourth, duplicates and repeated reopen. Retain real weather barriers/SIGKILL/reap evidence, original IDs and contacts; opposite-method recovery where useful.
- [ ] Full ordinary/integration/race-integration, focused race, build, vet, module/tidy, formatting, Staticcheck, nested consumer/historical readers. Record skips/diagnostics and source before/after hashes.
- [ ] Independent methodology/recovery review; substantive fixes require affected requalification under a new freeze.
- [ ] Document bounded results, remaining global work, diagnostic/trust limits and reproduction. Commit evidence only with source bridge; whole-branch review then push/open draft PR against verified main. No merge/release.
