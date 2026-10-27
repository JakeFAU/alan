# Task2c independent scoped integration review

Reviewed 2026-10-26. Base `a66760fb23c93e333d2244508eab4eeb3f1f7c82`; head `168f4d74b51a970def5996d23f01e9d01865ae07` (implementation plus controller contract). Exact Go/go.mod/go.sum identity: `32a2aa026b7be64c082d9672331daf338384ee118294338afdc12a03ec709a8b`.

**Spec verdict: Needs fixes. Quality verdict: Needs fixes; qualityApproved=false.** Most integration requirements are implemented coherently, but complete public schema4 causal-prefix validation has a confirmed gap. Public Replay accepts a scoped ordinary message consumed before its retained source authorization. This blocks the Task2c complete-validator approval, independently of the passing default/integration/scoped-race suites. This is a scoped Task2c review, not final whole-branch approval.

## Scope and method

Read the Task2c brief first, normative contract R1–R8, Task2a report/fix handoff, Task2b report/retirement handoff, and Task2c report. Reviewed the 203,475-byte source+contract companion in focused passes: reservation/runner integration; outcomes/closure/transitive joins; relational and prefix reconstruction; version activation; public APIs/inspection; hooks; tests and evidence. Used the full package/report for evidence context rather than rereading its repeated source manifests. Full review package is 1,375,191 bytes. HEAD and the exact current source manifest match the requested revision and all three final reported suite manifests.

Applied Go database/concurrency/error-handling guidance and the code-reviewer checklist. No subagents, live calls, credentials, dependencies, source edits, index/HEAD/branch changes, or reruns of reported suites. The sole new execution was a focused local overlay probe of a concrete unanswered ordering risk. Its original input, overlay, output, numeric subprocess status, and source manifest are retained beside this report.

## Strengths and requirement assessment

- **One owner and bounded physical work (R1/R5/R8):** `runner.go` extends the existing owner loop and worker accounting with persisted per-root limits while retaining global `MaxInFlight`. Saturated roots are skipped. Logical waiting parents do not claim slots. `external.go:247` refuses synchronous dispatch for all complete-profile stores before authorization/reservation/contact, including unscoped runs. The same original two-call counterexample remains byte-for-byte present in the current test; its retained red trace shows peak2 under capacity1, and the final test checks zero refusal-side mutations then normal Run peak1. Transport call sites in the three integration files are the existing Run workers and the refused synchronous path; no second scheduler was added.
- **Separate protocol units and shared debit (R1/R3):** source authorization remains distinct from the admission/reservation transaction. `composition_reservation.go:28` applies closure and shared cumulative effect limits; `:60` atomically retains the exact attempt digest, ordered decision and one allowed charge. Denials retain correlated known negatives. Adapter/legacy paths share this accounting. Unknown adapter candidates still produce no result (`adapters.go:268`, `:313`), do not refund charges, and existing attempts are not automatically rerun. Root inspection preserves unresolved evidence alongside non-success publication.
- **Structured outcomes (R2/R3):** roots have separate local status and immutable public publication. Frozen reply/Invoke/external obligations are joined before success. `invocation_outcome.go:627` traverses admitted descendants even below a known failed child; `composition_prefix.go:309` reconstructs the analogous prefix obligation independently of writer preparation. The retained original transitive counterexample excerpt is unchanged in the current test, and red16/green17 are preserved. No additional premature-success issue was found in these paths.
- **Retirement and no-op behavior (R7):** reconstruction distinguishes execution terminal freezes, first retirement closure, success freeze, and publication. Applicable retirement preserves its own frozen prefix and `cancelled/subject_retired` precedence; publication remains immutable; no-op Cancel/Abandon adds no closure. Root publication is included in the retirement publication check. Supported static ancestor/reopen tests now validate the complete ledger; the earlier seeded dynamic-grant probes remain explicitly internal, not a public authority claim.
- **Reconstruction (R4/R6, with I1 below):** static deployment origins and exact initial states, dynamic member/binding/request rows, derived permissions, charges, result/delivery observations, fact source digests and reverse links are reconciled. The prefix state tracks active members, source sequences, authorization, admissions, reservations, results, freezes, closures and outcomes. It checks creation before child execution/request consumption and result/publication before their consumption. Missing complete-profile tables fail rather than being created during replay/open.
- **Versions and compatibility:** public schema4 accepts complete `alan/composition/v1`; the private foundation profile remains explicitly rejected publicly and is not promoted. Schema3 remains a separate validation path, with no schema conversion or dependency change. The three changed temporary private-gate expectations are documented with original inputs retained; they do not replace foundation/unknown-version corruption rejection.
- **Task2a M1:** `composition_integration_test.go:426` inserts a foundation reserved invocation with actual intent/template/root foreign keys, checks FK cleanliness, and requires both ErrVersion and the precise `unfinished composition protocol in composition_invocations` diagnostic. This reaches the intended reserved-table guard, rather than relying on an earlier FK failure. It covers that invocation-table guard; it is not evidence that every reserved table was individually exercised.
- **Hooks and API handoff:** admission and result hooks follow `commitOwnedTx`; the result hook follows publication plus observation/delivery retention, before later parent Commit consumption (`invocation.go:523`, `invocation_outcome.go:558`). `TestCompositionPostCommitBarriers` inspects committed rows and an unconsumed observation. Public author/runtime end-to-end coverage uses the omitted profile version and real automatic Run. These are valid protocol/hook tests, not actual process-kill qualification.

## Issues

### Critical

None found in this scoped review.

### Important — I1: Replay accepts ordinary-message consumption before source authorization

**Location:** `internal/executor/composition_prefix.go:297` (ordinary/default consumption branch), with the inherited resolver at `internal/executor/composition_facts.go:397` and new authorization processing at `internal/executor/composition_replay.go:494`.

The complete profile now has an ordered authorization fact, but `validateConsumption` only asks `compositionObservationPredecessor` for the ordinary message's source execution fact. It never requires the message's scoped authorization fact to have occurred before the receiver consumes the observation. Final ordinary admission/authorization rows remain valid, so the ordinary ledger validator cannot recover the omitted causal relationship.

**Executed counterexample:** public Create of one complete-profile root containing sender and receiver; sender Commit emits a normal permitted message; real DeliverMessage authorizes and admits it; receiver Commit consumes its authenticated observation. Baseline public Replay succeeds. The only corruption is exchanging facts2 and3, updating relational `decision_order` and descriptor `Order` consistently:

1. sender execution/intent commit;
2. receiver execution consuming the message;
3. sender source authorization.

The original valid order was sender execution → sender authorization → receiver consumption. No ordinary row, admission/acknowledgment, observation, intent, program, source digest, fact ID, projection or authorization descriptor is edited. Foreign keys remain enabled and `PRAGMA foreign_key_check` reports no violation. Public Replay still returns nil. The unchanged assertion expecting rejection therefore fails with numeric Go exit1.

**Impact:** schema4 Replay certifies a causally impossible authority/consumption prefix. This violates Task2c's complete ordered predecessor validation and invalidates the claim that final-row agreement is sufficient for every consumption relationship. It is a validator defect demonstrated through public Replay; the probe does not claim that the live writer itself produces this ordering.

**Fix direction:** for scoped ordinary-message sources, require the exact retained authorization predecessor as well as the source execution/admission linkage before consumption. Preserve genuine unrepresented/unscoped-source compatibility and foundation behavior. Audit the closely related source-denied lifecycle-observation path for represented authorization predecessors without inventing a target-decision fact. Retain this unchanged order-only probe as a regression; do not weaken its expected rejection or rewrite its original red evidence.

Exact input: `review-message-authorization-input.go.txt`; original overlay: `review-message-authorization-overlay.json`; output: `review-message-authorization.log`; command/status: `review-command.json`; source: `review-source.json`. Input SHA256: `577a4fc817fcedad612d6092d2f1f144248db2e2000bc04befaed81413e4997c`.

### Minor

None recorded. Small stylistic issues were not elevated into correctness findings.

## Verification actually performed

One focused Go execution, no reported-suite reruns:

```text
go test -overlay /tmp/alan-task2c-independent-review/overlay.json ./internal/executor -run ^TestIndependentCompositionMessageAuthorizationPrefix$ -count=1 -v -timeout=30s
```

Environment: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`. Numeric subprocess exit **1**, with `Replay error = <nil>` after the order-only corruption. The baseline Replay and FK control pass in the same execution. Overlay replaces a virtual extra test path only; production checkout files are untouched. Original local inputs remain in `/tmp/alan-task2c-independent-review/` as well as the retained copies. Reproduction elsewhere needs an overlay path remap; the original overlay is preserved unchanged.

Read and cross-checked `commands.json`, final suite logs39/40/41, all three source manifests, and final formatting records. Each final suite manifest exactly equals the independently computed current manifest, not merely its identity string. Recorded statuses are default exit0 (15 tested packages), integration exit0 (18 tested packages), and scoped worker-ownership race exit0. These are inspected prior-run evidence, not executions by this reviewer. Formatting records report clean changed Go files/diff, with the disclosed staged evidence-only blank-EOF warning preserved.

Read original red16/green17 transitive evidence and red35 dispatch evidence; verified both retained original executable test excerpts are exact substrings of current test source. Original fixture/compiler failures and temporary-gate input changes remain separately disclosed. No green suite is treated as proof against I1.

## Named outside-diff integration inspections

Only the following existing code was opened beyond changed hunks for concrete risks:

- `composition_facts.go:357` and the focused causal test/helper ranges: determine which source predecessor ordinary consumption validates and how to construct a real message schedule for I1.
- `composition_store_test.go:13`: minimal retained profile fixture shape for the same probe.
- `adapters.go:128`, `:249`, `:268`, `:291`: verify uncertain-attempt recovery, no known result for unknown, and authenticated late-result retention.
- `runner.go:145`: confirm the unchanged global capacity condition encloses the new per-root launch/accounting path.
- `invocation.go:158`: compare live last-boundary ancestor closure with prefix reconstruction, including root local terminal.
- `invocation_outcome.go:357`: verify the shared outcome preparation's reply-admission wait and status handling around the changed descendant join.

No broad repository crawl or redundant whole-file reread was used.

## Cannot verify / declined to judge

- Task3's reusable ordinary collection application, deterministic expected work tree, ordered aggregation and complete item-count acceptance: future task, not this protocol integration slice.
- Actual SIGKILL at acceptance/partial completion/recorded-response boundaries: Task4; the hook/unit observations do not establish process recovery.
- Task4 faulty-monitor variants, whole-source race/vet/archive and whole-branch acceptance: future qualification, not inferred from scoped race or finite successful traces.
- Public dynamic lifecycle grant authoring: deliberately absent; seeded internal probes are not treated as supported deployment evidence.
- Standalone adapter/HTTP use outside Store.Run, remote activity after an unknown local return, multi-host coordination, migration, business deduplication/compensation and live-provider answer quality: expressly outside the contract and no claim made.
- Unbounded liveness/safety or exhaustive corruption coverage: finite source review and targeted evidence only.

## Assessment

Task2c is **not yet approved**. The single-owner integration, immutable protocol accounting, transitive join correction, R7/R8 handling and evidence discipline are substantial strengths. Fix I1 and independently rerun its unchanged regression plus focused causal controls before granting complete schema4 validation approval. Broader Phase1G acceptance remains later.
