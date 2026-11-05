# P3 validated readiness program reuse implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Execute each task with an independent specification and quality review. Preserve all evidence and rejected findings.

**Goal:** Establish whether repeated validation of one immutable unbound program within one readiness scan is material; implement at most one justified optimization and deliver a qualified draft PR.

**Architecture:** Preserve the ordinary executor and public weather workload. The sole candidate is lazy, successful-only, exact-program-ID reuse within one `Store.selectReady` call. Per-run projection, generation, binding, observation and reduction checks remain independent. A no-change finding is a valid outcome.

**Tech Stack:** Existing Go 1.27.3, CGO SQLite, Python standard-library qualification tools. No new dependencies.

**Spec:** User's complete P3 assignment preserved beside this plan as `request.txt`; original attachment `/Users/jacob/.codex/attachments/de9bde1e-9c8e-4660-a7fb-27584a069f6e/Pasted text.txt`. The user assignment governs any discrepancy. The repository contract produced in Task 1 supplies portable source identities and constraints.

## Global Constraints

- Work only in github.com/JakeFAU/alan. Do not edit alan-research or execute the separate Phase 3B.4 request-binding assignment.
- Complete reconnaissance, contract freeze, implementation if justified, differential and process qualification, independent review, and a new draft PR. Do not stop after a plan. Do not merge, release, or modify another task's branches.
- Do not change public APIs, SQLite schemas/formats/indexes, dependency pins, scheduler policy, approval semantics, physical dispatch rules, or ownership.
- No Store-lifetime/global cache, background indexer, alternate database, or general caching framework. No second optimization of observation scanning, root traversal, approval snapshots, or binding.
- Preserve P1/P2 and the integrated request-authority implementation. Formats 13/14 retain their existing feature restrictions. Do not invent authority-plus-bounded-series support or change the known immediate-selection model mismatch.
- Do not reduce durability settings, combine semantic commits, skip checks, preapprove later observations, omit reporting, or replace public approval with a cheaper benchmark-only route.
- No paid services or live providers. Primary timings retain the existing deterministic in-process adapter. Controlled HTTP is used separately where actual transport/recovery evidence is needed.
- Do not capture the entire inherited environment. Record exact commands, source/binary identities, whitelisted settings, exits, skips, failed attempts and before/after hashes.
- Preserve every original operation, authority, approval, expenditure, uncertainty, result and selection. Never normalize away a semantic difference.
- One implementation writer at a time; controller owns reviews and publication. No nested subagents. Do not run heavy qualification concurrently with timed campaigns.

## Review Focus

1. ProgramID equality must not hide different feature-context or limit inputs; test independent Stores and changed context on a later call (Tasks 2–3).
2. Binding or Reduce must not mutate nested reused data; analyze every branch and actively probe aliases (Tasks 2–3).
3. Early return and lazy first-use failures are observable; compare actual baseline selection and error boundaries (Task 3).
4. Reopen and revocation cannot acquire authority from previous validation; test fresh-owner and authority cases separately from series (Task 4).
5. Tool/counter changes must not move approval/reporting out of Execute or silently drop nonzero runs (Tasks 1, 5–6).

## Verified starting state

Base `926a899b651fccc48be52a054612a92afcf1ddfd`, tree `a58fe3b0ef0b9a7525c46c4f0a662774e6d4221d`. GitHub merge method changed commit identities: check ancestry and tree equivalence explicitly, not assumed original SHA ancestry. New branch `codex/p3-readiness-program-reuse` in the existing clean linked worktree `/Users/jacob/.codex/worktrees/recoverable-investigator/alan`. Original P2/convergence branches and other worktrees are preserved. Raw campaign root `/private/tmp/alan-p3-j216kxoz`; initial source/toolchain metadata already retained there.

### Task 1: Establish the current baseline and freeze attribution

**Files:** Create `docs/evidence/performance-p3/contract.md`, `baseline-attribution.md`, `source-map.md`, compact raw-data index and narrowly needed scripts under `docs/evidence/performance-p3/`. Reuse unchanged P2 tools by explicit path/source hash rather than copying them wholesale. No production edits.

**Interfaces:** Consume exact base above and existing P2 driver/instrumentation/comparison interfaces. Produce a frozen contract with pilot evidence, reproducible instrumentation, exact confirmatory schedule/guards, and the proposed invariant's source premises. Controller records the task commit and obtains independent attribution/invariant challenge before production code changes.

- [ ] Read every starting-point document named in the user spec, existing driver/recorder sources and the actual selector, loader, binding, validation/copy and Reduce paths.
- [ ] Record module/native/compiler identities and a tracked-file baseline manifest. Verify current PR states, no existing P3, ordinary baseline tests and controlled loopback/acknowledged subprocess capability. Preserve every nonzero result; no until-green loop.
- [ ] Build frozen ordinary and separate lightweight diagnostic binaries. Retain instrumentation source/patch/hash and verify anchors; do not contaminate primary timing. Record corrections to a recorder as separate tooling changes with tests proving failed/outcome preservation and credential exclusion.
- [ ] Run one declared baseline pilot at each of 1, 8, 128, 256, with separate useful counter/profile scopes. Execute includes N Run calls and N-1 fresh snapshot/basis/approval cycles; provisioning/Create/final reports/reopen/Replay stay separately measured.
- [ ] Count actual selectReady calls, eligible visits, encountered and distinct ProgramIDs per call, successful/failed loads, SQL/decode/validation work, binding calls, observation attempts and selections. Export visit/reuse distributions, including early returns. Capture B128 allocation/CPU attribution and B256 coarse scopes when useful without treating overlapping scopes as additive.
- [ ] Conclude whether actual repeated-load work is material; freeze exact predictions, negative controls, sample schedule and regression guards before confirmatory runs. Default schedule is six adjacent fresh-process pairs per bound 1,8,16,32,64,128,256, balanced AB/BA, plus one excluded warmup per source/bound. Any smaller design requires a prior explicit evidence-backed ruling.
- [ ] Freeze small-bound and other-phase guards in the contract (start from max(10%,1ms), refine only with pilot evidence before candidate results). Record shared-host confounders. If no material mechanism exists, retain no-change finding and route directly to relevant Task 6 delivery.
- [ ] Self-review full diff, run tooling checks, commit narrowly and report exact paths/hashes/commands. Do not implement the candidate.

### Task 2: Freeze the reuse invariant and RED controls

**Files:** Create `docs/evidence/performance-p3/invariants.md`; new focused selector test files under `internal/executor/`; test-only baseline selector/reference and instrumentation with source provenance. No optimized production path yet.

**Interfaces:** Consume approved Task 1 contract/counters. Produce source-backed proof obligations and executable RED load-count regression plus semantic fixtures. Preserve baseline function independently of any candidate helper.

- [ ] Enumerate all loadProgram validation inputs: persisted artifact hash/bytes, semantics, limits, author metadata/schema validation and request-authority feature context. Explain why each is fixed within a legitimate call and why arbitrary unsynchronized SQL mutation is unsupported.
- [ ] Trace immutable-program insertion, owner exclusion and Store mutex, binding resolution, handoff, reducers, nested slices/maps/schema/expressions. Document actual mutation and copy ownership, including any required minimal isolation and cost.
- [ ] Freeze lazy first-reach, successful-only exact-ID reuse, per-call lifetime ending before commit, no negative entries, unchanged per-run binding/generation/observation/reduction checks and logical cancellation boundaries. Preserve error order and distinguish wall-clock I/O fault opportunities.
- [ ] Add a deterministic performance RED test showing baseline successful load/validation repeats for shared IDs, with distinct-program and no-visit controls. Counters must be test/diagnostic-only, not a production telemetry abstraction.
- [ ] Construct positive fixtures and explicitly labeled disposable corruptions for every semantic matrix row in the user assignment. Compare full ready tuple (RunID/generation/epoch/sequence/observation) and errors against the independent baseline reference; include alias mutation probes, later-call changes, independent Stores/reopen and cancellation/closing.
- [ ] Run and retain RED failure at the exact intended redundant-work assertion, plus existing semantic controls that already pass. Commit tests/invariant separately; clearly label expected RED status. Controller independently reviews before Task 3.

### Task 3: Implement one call-local reuse and readiness equivalence

**Files:** Modify only `internal/executor/runner.go` production selector unless a narrowly justified, reviewed alias isolation change is essential. Complete focused tests from Task 2 and compact mechanism results.

**Interfaces:** Keep `func (s *Store) selectReady(ctx context.Context, last *RunID) (*readyRun, error)` unchanged. Reuse unbound successful `Program` by exact `run.ProgramID` only in that invocation. Keep `boundCompositionProgram` and `Reduce` calls per-run/per-observation exactly where required.

- [ ] Implement the approved minimal lazy reuse, with no public API or Store field. Do not retain bound values or ready decisions. Add a concise supported-scope comment.
- [ ] Make deterministic load-count RED green; keep binding/reduction counts unchanged. Run baseline-reference matrix and adversarial controls ordinary/race. Verify invalid first-use, unvisited invalid program, bad membership/binding, independent contexts, generations, later calls, cancellation and closing.
- [ ] Run focused existing execution/authority/composition checks appropriate to touched semantics. Retain exact production diff and source hashes. If any essential invariant cannot be preserved narrowly, withdraw candidate and report the no-change finding.
- [ ] Commit implementation and report; controller reviews immutability/aliasing/per-run semantics before recovery qualification.

### Task 4: Qualify exact evidence, process recovery and external use

**Files:** Reuse corrected P2 differential, weather process, external eight-read, historical reader and authority tooling. Add only needed P3 wrappers/compact summaries under `docs/evidence/performance-p3/`; preserve prior files/archives.

**Interfaces:** Use independently built baseline/candidate binaries and complete isolated durable copies. Compare full typed inspection, Replay, selected history and retained causal records at decisive prefixes; use original IDs and exact receipts.

- [ ] Freeze a compact route/case manifest before execution. Preserve the same Execute/public approval boundary; no authority-plus-series fixture. Challenge hardened comparison fields for contact capability/effect/protocol/endpoint and semantic identity, recording intended negative outcomes separately from apparatus errors.
- [ ] Run direct and controlled HTTP identical-input comparisons and genuine acknowledged SIGKILL/reap/fresh-owner cuts for retained request, paid reservation, accepted/lost response, retained result, settlement before selection, and completion before the next approval.
- [ ] Demonstrate no reissue of retained responses, unresolved spent missing non-idempotent results, independently approved successors, no refunded expenditure and effect-free readers. Cross-build resume uses isolated copies and noncontaminating endpoint contracts.
- [ ] Run own/delegated request authority, wrong request/generation, revoked-before-gate, valid grant across reopen and immediate-selection refusal separately. Keep the hardened competing-reader test unchanged; diagnose any new failure against actual base.
- [ ] Run external eight-operation consumer and relevant genuine historical reader/format/protocol cases. Record exact supported versions and any opt-in skips honestly.
- [ ] Commit wrappers/compact evidence summaries and report for authority/recovery/compatibility review. No broad timing concurrently.

### Task 5: Execute the frozen performance comparison

**Files:** Frozen schedule and hash-bound source/binary manifests, raw external measurements/profiles; compact analysis/results under `docs/evidence/performance-p3/`.

**Interfaces:** Consume Task 1's predeclared schedule/guards and independently reviewed production inputs. Use the same driver bytes for both sources, with exact module replacement provenance. Instrumented runs never substitute for uninstrumented primary timings.

- [ ] Before any confirmatory process, implement and adversarially qualify the minimum P3 direct-mode strict paired-validator/runner adaptation. Preserve original P2 HTTP tooling unchanged; diagnostic validation is insufficient. Freeze and commit the adapted tools, then pause this task at a controller-owned independent tooling review checkpoint. Only an approved exact tool revision may run the declared campaign.
- [ ] Freeze timed source/binary hashes and execute the declared schedule serially with all warmups/outcomes retained. Record Execute, other phases, process CPU, Go allocations/counts/GC, peak RSS and shared-host conditions.
- [ ] Run compact mostly-shared and mostly-distinct program controls; record actual visited/distinct ID distributions. Diagnose predicted program load/SQL/decode/validation changes separately from preserved per-run binding and reduction work.
- [ ] Verify logical work at B256: 256 effects/direct adapter entries/charges/results/selections, 255 later approvals and 1023 reductions. Do not call direct entries HTTP contacts.
- [ ] Refresh candidate CPU/allocation profile. Report medians, variation, paired outcomes and limited inference; quantify small-bound/other-phase guards and peak memory trade-offs. Retain observations/root/snapshot costs without a second optimization.
- [ ] If immaterial or materially regressive, report the frozen result and withdraw the production change through a separate reviewed commit; never tune samples or weakened requirements. Commit compact analysis and report for independent attribution/method/tool review.

### Task 6: Freeze final source, independently review and deliver

**Files:** `docs/evidence/performance-p3/results.md`, `reproduce.md`, review/evidence index, README link as appropriate; external immutable full qualification records.

**Interfaces:** Exact final source tree must match qualified executable inputs. Canonical full-gate outcomes can live outside the frozen tree to avoid self-reference. Evidence-only changes require explicit byte bridges, not relabeled runs.

- [ ] Write the concise eight-question decision and scope limits, exact contract/invariant/implementation/timing identities, and reproduction. Preserve rejected findings/corrections. Freeze final source before full checks.
- [ ] Run focused new tests; full ordinary, integration, integration-race; public consumers; actual historical readers/protocol regressions; nested benchmark/comparator module checks; build; ordinary/integration vet; module verification/tidy-diff; formatting/whitespace; ordinary/integration/race/integration-race root Staticcheck. Use documented timeouts, whitelist environment, exact exits/skips and before/after hashes.
- [ ] Obtain independent final whole-change review on actual source and decisive retained data. Resolve substantive findings with affected requalification. Preserve any baseline diagnostic separately; no prior-source run relabeling.
- [ ] Controller verifies current target/main and integration without altering other tasks. If main advanced, report and make a separately reviewed integration decision; do not silently rebase or include another task.
- [ ] Controller publishes only reviewed/scanned evidence, verifies actual URLs, pushes the new branch and opens a new draft PR. Preserve raw/workspace/review artifacts. Do not merge or release.
