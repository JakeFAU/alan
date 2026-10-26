# Phase 1G bounded data-dependent composition implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accept a typed work set after initialization, apply one retained component definition within durable bounds, and recover its direct or nested composition.

**Architecture:** Bounded durable invocations of retained component templates, ordinary structured collection iteration and scatter/collect programs. Reuse the single existing Store/Run, SQLite protocol boundaries, adapters and read-only replay; physical effect slots are never held by waiting parents.

**Tech Stack:** Existing Go 1.27.3, standard library and pinned SQLite driver; no new infrastructure or dependencies.

**Spec:** [01-composition-contract.md](01-composition-contract.md), governed by the user's Phase 1G attachment. Independent contract review R1–R6 is complete; original findings, two completion-race counterexamples and scoped PASS addendum are retained under evidence/contract-*.md. Baseline `80dead19846d9b034b4457a9436e43acb81da3df` on branch `codex/phase1g-data-composition`.

## Global Constraints

- One retained application artifact handles data supplied after initialization. No tailored Go compilation/provisioning or one helper per possible item.
- No privileged Map, Investigator, Reviewer or Control runtime type, no adapter workflow, no second scheduler.
- Authors use typed public Go APIs, not SQL, private child state, numeric program counters or opaque collection strings.
- Existing research/evaluation evidence, original failures and counterexamples remain unchanged. Historical reproduction and current-source regression are separate.
- Version new collection, binding and persistence semantics explicitly; preserve old scalar artifact identities and meaning.
- Root work/depth/effect bounds are durable and shared by descendants. Unknown work stays bound and incomplete; replay has no dispatch authority.
- Scope remains single host, no ownership transfer, general delegation/migration, UI, provider catalogue, stable SDK or quality retuning.
- No live requests under the previous phase's spending authorization. Controlled providers only for this qualification.
- Independent task reviews plus final adversarial review; publish a draft PR, never merge.

## Review Focus

1. Mutable aliases/cyclic or oversized collection schemas cannot corrupt retained identity or bypass bounds (Task 1).
2. Runtime loop mutation, zero iterations and nested loops preserve snapshots and definite assignment (Task 1).
3. Duplicate invocations and source/root closure cannot leak creations or multiply budgets; rejected results remain immutable (Task 2).
4. Reply/terminal races and incomplete descendants cannot publish successful aggregate output; result authentication cannot be supplied in user payload (Tasks 2–3).
5. Replay must detect missing/orphan child origins, incorrect budgets and rebound results rather than repairing them (Tasks 2–4).

## Tasks

### Task 1: Typed bounded collections and ordinary durable iteration

**Files:** Extend `schema/schema.go` with focused collection helpers/tests; `author/author.go`, `compile.go`, `application.go`, `identifiers.go` and focused collection compiler tests; `internal/executor/program.go`, `author_metadata.go` and focused reducer tests. Keep new algorithms in small adjacent files when necessary; no unrelated refactor.

**Interfaces:** Native bounded list-of-record values and field schemas; public construction/access with deep-copy semantics. Add `Definition.Collections map[string]schema.Field` for declared bounded list locals, `author.Append(name string, values map[string]Expr) Step`, and `author.ForEach(collection Expr, bindings map[string]string, body ...Step) Step`. A list field carries element record, MaxItems, MaxBytes, optional unique key, and explicit nesting validation. The compiler picks `alan/1g/v1` / `alan/author/v2` only for new semantics; legacy compilation remains byte-identical. Use omission for new zero fields. Public scalar APIs remain source-compatible where feasible. Exact collection constructor names are a mechanical implementer choice, documented in the report.

- [x] Write failing tests for post-initialization counts 0/1/3/7/8, rejected 9, distinct IDs with duplicate payload, duplicate key rejection, nested collection/total-byte bounds, UTF-8, aliases, malformed values and schemas.
- [x] Write reducer/compiler tests for a single loop body independent of MaxItems, input snapshot mutation, zero-iteration variable initialization, nested cursor recovery and mid-loop reopen/replay. Preserve old helper artifact IDs and reject new collections under legacy semantics explicitly.
- [x] Implement typed collection validation/canonicalization/equality/copying, finite cursor iteration and append, with one finite durable reduction per instruction. Structured loops alone permit backward control; public Goto remains forward only. Do not hide aggregation/invocation policy in an opcode.
- [x] Run focused schema/author/executor tests and default suite, gofmt; retain original RED and GREEN outputs under phase1g evidence/task1.
- [x] Commit this testable slice and report exact APIs, changed files, compatibility and test outcomes. No child invocation, application or runner implementation yet.

### Task 2: Retained template invocation, shared resource scope and recovery

**Files:** New focused template APIs in `author/`, invocation/binding/budget code in `internal/executor/`, public aliases in `runtime/`; narrowly extend existing store/messages/runner/lifecycle/replay/inspection/metadata paths and tests.

**Interfaces:** `author.Template` retains an ordinary Application plus one public Port and immutable binding metadata; public constructor validates internal closure. Definition gains named invocation declarations referencing retained templates. `author.Invoke(pending, template string, values, updates map[string]Expr) Step` commits one operation and stores its handle; `author.AwaitInvocation` consumes its authenticated correlated outcome and exposes explicit status, reason and response-presence variable bindings alongside public payload bindings. Use a small outcome-bindings struct or equivalent explicit parameters; freeze the exact signature in the Task 2 report. `Invoke` and `AwaitInvocation` are generic request/result operations, not a Map policy. Provision gains immutable template catalogue and explicit per-root work/depth/effect/concurrency envelope. Inspection/replay include invocation origins, bindings, results and charged resource scope. Exact envelope field names are documented before Task 3 consumes them.

- [ ] Write failing tests for immutable accepted/rejected operation deduplication, no creation from uncommitted intent, valid scoped two-member template, fresh invocation vs reopen, authority/budget rejection and stale/cross-invocation replies.
- [ ] Implement reviewed admission/completion/version rules from the contract, preserving source intent/authorization/admission/result/consumption boundaries. Bind only metadata-declared peers, never payload strings. All initial child state and roots are durable. Existing Run schedules it; no child-owned goroutine/scheduler.
- [ ] Add cancellation/receipt/root closure serialization and root-budget conservation tests, including nested and unknown work. Explicitly handle lifecycle Retry under scoped invocation policy; no accidental budget reset or reassociation.
- [ ] Add ledger reconstruction/orphan/corruption tests, historical store/version matrix and keyless read-only replay no-dispatch checks. Preserve base model regressions.
- [ ] Run focused tests/default suite/gofmt and commit; retain evidence and report APIs for public consumers.

### Task 2a: Ordered durable facts and immutable binding foundation

**Reason for subdivision:** Task 2's first implementer demonstrated that identical per-run final histories can have different failure-prefix reply sets. The ordered fact/binding/origin foundation must be testable before wiring the entire protocol. This changes sequencing only; R1–R6 and acceptance remain unchanged. All 2a–2c work must finish before Task 3 or publication.

**Files:** Focused `internal/executor/composition_types.go`, `composition_store.go`, `composition_facts.go`, `composition_binding.go` and tests; minimal hooks in store commit/lifecycle/read metadata paths. Do not yet add public author Invoke or a new scheduling loop.

**Interfaces:** Internal typed template/root/binding descriptors with immutable canonical identities, metadata-only pure binding resolution, and schema-4 ordered facts linking actual scoped execution/lifecycle transactions. A private composition-store constructor may reuse the current Create implementation with an explicit profile parameter; public Create remains legacy and public Open/Inspect/Replay must reject not-yet-enabled schema4 before activation. This temporary activation gate is removed only after 2c's complete validator/protocol is ready. Keep one shared existing transaction path, never copied alternate commit logic.

- [x] Execute and retain the two three-commit schedules in evidence/task2/report.md: A.Send1/B.Failure/A.Send2 versus A.Send1/A.Send2/B.Failure; show identical ordinary per-run rows and different ordered fact prefixes. This is concrete evidence for the missing ordering, not a second explorer.
- [x] Define/add schema4 tables needed for immutable roots/static memberships, template catalogue, invocation/membership/binding descriptors, ordered facts, result/charge ledgers. Reserve exact typed result structures for 2b/2c without enabling unfinished outcomes. Preserve schema3 DDL/IDs/behavior.
- [x] Add an internal immutable composition profile on Store and metadata/profile validation. Root IDs include namespace and declared root run; initial members share root. Pure binding resolves only metadata-declared endpoints while preserving original bytes/IDs and literal payload strings. Reject inconsistent/missing/cross-scope binding descriptors. No trusted authority is inferred from payload data.
- [x] Append scoped execution and lifecycle facts in their existing transactions, with monotonic order and immutable descriptor/row links. Rollback leaves neither transition nor fact; duplicate/no-op paths do not invent another fact. Keep schema3 path untouched. Add narrow fact append APIs for later admission/attempt/result transactions, with explicit finite kinds and descriptor validation.
- [x] Verify both-direction fact/history/origin/static-membership integrity for this foundation, plus version rejection and source/binding byte identity. Reopen/read validation may use a private internal path while public activation is gated; do not claim complete schema4 replay or invocation behavior yet.
- [x] Run focused transaction/rollback/binding/version tests and default suite; retain RED/GREEN and original counterexample inputs. Commit and report exact internal APIs/table invariants for Task2b. No code generation, provider calls, migration or fixture state mutation standing in for a production path.

### Task 2b: Public templates and atomic invocation admission

**Files/interfaces:** Consume 2a descriptors/fact transaction helpers. Add the Task2 public Template, invocation declarations, Invoke and explicit outcome bindings API; immutable closed deployment catalogue and root allowlists; committed intention, separate source authorization, atomic child admission/result and initial request binding. Integrate dynamic generation-1 origins and scope replies. Keep public schema4 activation gated until 2c completes all enforcement/validation.

- [x] Test then implement fresh child initialization from retained artifacts, all-member atomic admission, permanent accepted/rejected result deduplication, denied authority/depth/work counts, no uncommitted creation, public request correlation and source/root closure races.
- [x] Test authenticated reply candidate recording and the reviewed reply-lifetime exception, including evidence's Send(invalid_result) then Failure, duplicate/conflicting sends and original acknowledgment loss. No external or arbitrary-message authority inherits the reply exception.
- [x] Implement ordinary AwaitInvocation outcome bindings with explicit Status/Reason/ResponsePresent. Rejection with an enum-constrained output schema must be a known absent response, not invalid successful payload or store failure. No opcode contains scatter/aggregation policy.
- [x] Use ordered facts to capture failure/success freeze sets; expose internal pending versus ready publication without prematurely claiming completed invocation. Consume current public schema/loop APIs and preserve all older records.
- [x] Run focused tests/default suite, commit, document APIs and remaining 2c gate. No Task3 implementation yet.

### Task 2c: Shared limits, completion, runner integration and full schema4 replay

**Files/interfaces:** Complete 2a/2b via the existing runner, external/adapter reservation paths, lifecycle, inspect and replay. No new standalone scheduler. Remove schema4 activation gate only after complete validation/enforcement tests pass.

- [ ] Implement root-and-global physical capacity, persistent attempt reservation/debit/denial, ancestor closure serialization, scoped Retry rejection and late authorized evidence retention. Skip saturated roots while another root has eligible work.
- [ ] Complete immutable scope outcome publication using ordered frozen facts, committed-but-unadmitted Invoke/reply obligations, required-member/descendant/external joins and failure propagation. Keep application aggregation/policy in ordinary code.
- [ ] Implement exhaustive cross-ledger reconstruction for declared schema4 invariants: origins/bindings/grants, all ordered fact links and prefix decisions, root membership/debit conservation, outcome/candidate admission and consumption. Replay is read-only and rejects corruption rather than repairing it.
- [ ] Test capacity1 nested progress, unknown plus independent work, root/member closure orders, accepted and rejected reopen, late evidence, source authorization versus attempt cutoff, all-member failures and missing/delayed reply schedules. Actual process SIGKILL tests remain Task4.
- [ ] Enable the complete schema4 public profile; preserve schema3 support and explicit unknown-version rejection. Run focused/default/integration tests and present Task2 public APIs to Task3 after independent scoped review.

### Task 3: Ordinary collection component and separate-module consumer

**Files:** New `compose/` public derived collection builder and tests, `examples/worktree/` public-consumer application, a separate-module consumer fixture, narrow optional template factory refactor in `examples/evidence/`, and `cmd/alan-compose/` CLI with focused tests.

**Interfaces:** A reusable ordinary collection component receives a typed ordered list of explicit item IDs plus item input, invokes one template per item through Task 2, retains handles, and consumes outcomes in input order into a typed result list. Derived builder uses public steps only. Evidence template reuses existing definitions/policy and preserves legacy factories/artifacts. CLI supports init (no work list), submit/admit (post-init data), run/recover, inspect and replay; no planner or provider live configuration is needed.

- [ ] Write failing external-module contract tests using one retained artifact for 0/1/3/7/8 and rejected 9; no recompilation/topology changes after submission.
- [ ] Implement deterministic work-tree component with independently calculated expected mapping, direct and nested same port contract, empty output, duplicate payload/different IDs, reverse completion and lower exhausted budgets. Report code/artifact/template/provision/run counts and result ordering separately.
- [ ] Reuse evidence helper as effectful template with controlled existing adapter; no prompt changes, evidence-specific opcode or helper-per-item factory expansion. Define required descendant terminal/result policy through ordinary programs.
- [ ] Demonstrate capacity 1 and 2, nested saturation, constant initial provisioning vs proportional admitted work, unknown operation with independent progress and no premature success. Use causal acknowledgments for physical events.
- [ ] Run focused public consumer/CLI tests/default suite/gofmt; commit and report reproduction commands/public API with example input.

### Task 4: Process boundaries, independent monitors and qualification

**Files:** Focused `internal/compositiontest/` reusing existing boundary/provider/process helpers; new phase1g evidence collector through existing `docs/foundational/phase1e/tools/` and `docs/foundational/phase1f/tools/` machinery; phase1g docs/source package and root README/decisions updates.

**Interfaces:** Existing owner-only observational failpoints, subprocess SIGKILL controller, independent provider contact ledger, immutable runtime inspection, credential-free Replay. Structured evidence records complete fixture, source, commands, boundary acknowledgments, invocation/item IDs, budgets and outcomes. No second qualification framework.

- [ ] Add actual process death after accepted work, partial completion, and result recording before consumption; recover in a new process from retained store/artifacts. Include interruption during recovery where feasible without widening claims.
- [ ] Add stale/duplicate result attempts while newer work is pending, admission/completion cancellation serialization, unknown provider results and held-effect/independent-progress barriers. Retain original inputs and any observed defect before fixes.
- [ ] Add test-only faulty variants detected by unchanged independent monitors for cross-item/result binding, budget reset, premature success, uncommitted creation and replay dispatch/creation. Ensure each input reaches the relevant fault.
- [ ] Run focused checks, full default/integration/race suites, both vet modes, gofmt, dependency verification and prior research regressions. Capture raw commands/platform/outcomes; compare historical records separately from current-source changes.
- [ ] Retain exact source manifest/archive sufficient to reproduce, runbook, evidence-to-claim table, authoring/provisioning/record/concurrency/latency/recovery measurements and limitations. Obtain independent whole-branch review and preserve findings/fixes.
- [ ] Commit reviewed increment, push implementation branch and open/attach draft PR against verified current main; do not merge/release.
