# Phase 1H Task 2 application fix-round1 rereview

Reviewed range: `52392fa42628b6094c7e15c898cf6e7493a70d93..76d5ccd6cca80586de3efddd8f82d4ea01cf31e1`. Checkout: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.

**Original P2 whole-frontier count finding: ADDRESSED. Scoped code-fix approval: APPROVED. New breakage in the fix diff: none found.** This does not close the full Task 2 root/byte evidence gate or approve Task 3/final qualification.

## Finding disposition

The old per-item `increment("work_count")` is removed from `examples/adaptive/adaptive.go:93`. `:124` now performs one ordinary `Set("work_count", Var("trial"))` before entering the item-emission loop. `trial` is initialized at `:39` and the existing successful preflight at `:86` computes the accepted cumulative total. This implements the original required distinction between whole-frontier application acceptance and individual runtime operation admission.

I traced both entrances to the round body. Initial seed preflight runs only after valid profile/domain input; duplicate, oversized and empty frontiers deactivate before the round body. Later candidate preflight is followed by the unchanged stop-precedence selection and decision commit; inactive later iterations skip the body. Thus failed/oversized/round-stopped frontiers cannot assign their speculative `trial` value to the accepted count. Accepted totals stay stable during item emission and result collection, and the next preflight starts from the correct cumulative count. The fix does not fabricate runtime invocation/run/effect charges.

The private history prefix is now explicitly documented at `adaptive.go:90-92` and `README.md:24-28`. It records emission-loop progress, while `current` plus `work_count` retain full acceptance. History completes before the next collision scan; retaining only IDs and avoiding another full frontier copy remains appropriate. This resolves the original review's request to document that distinction rather than incorrectly equate the prefix with the acceptance total.

## Regression and evidence verification

`examples/adaptive/acceptance_test.go:16-59` executes a real direct/nested application, then inspects immutable history for the first round-two invocation transition with current `[7,10,11]`, one consumed round and no current-round handles yet. It requires an actual invocation intent, logs its original operation and separate transition address, and checks accepted total6/trial6. It does not manufacture state or claim process interruption qualification.

- RED `fix-r1-counter-red` has exit1 and shows both direct/nested actual history sequence502, successor PC170, accepted4/trial6.
- GREEN `fix-r1-counter-green` has exit0 and shows both actual history sequence501, successor PC171, accepted6/trial6. The full current frontier and documented history prefix `[1,2,3,7]` are preserved.
- `TestAdaptiveAcceptedCountDenialAndEarlyStops` also passes direct/nested. It asserts application count3 despite only two accepted runtime leaf operations, and checks initial oversized frontier count0 plus unperformed next-frontier work/round stops retaining count3. Final pure-scalar primary execution still reports exhaustion7/rounds3.

I independently verified the RED/GREEN, identity, and selected-default-input command log hashes and source-manifest digests. The GREEN manifest matches current files exactly. The retained pre-fix application bytes match `FIX_BASE` exactly; the retained red test/application source hashes match the RED manifest. Both original counter witness rows are exact members of the original direct/nested CLI inspection histories, and inspection/successor hashes match. The original application review is retained byte-for-byte unchanged. The gzip packaging of the older test delta decompresses to the exact prior committed diff.

The source delta is limited to the application counter code, explanatory README and new acceptance test; the associated collector change merely adds a separate output directory for fresh audit artifacts. No runtime/author/schema/leaf/helper/consumer production source, public schema or resource ceiling changed. The identity evidence reports fresh compiled adaptive artifacts (scalar114112 bytes/evidence117295 bytes, +204 each); those are new identities and must not inherit old source's qualification claims.

## Gates still open

The refreshed root run's frozen manifest is `fd7ab336ee82ccc27e846e994496a288256c63daa705927e672563d256e06a20`, and every manifested file matches current contents. At this rereview's final evidence check, `fix-r1-root-default.command.json` was absent. **Full corrected default-suite and byte-matrix acceptance remains pending actual completion and inspection of its result.** The focused counter regression is sufficient to close this P2 code finding, not to substitute for that gate.

The obsolete source11c run was intentionally terminated and retained as exit1; it is neither a successful qualification result nor process-crash/recovery evidence. Earlier source-bound successes remain historical. Refreshed consumer/integration/CLI evidence and final artifact/accounting summaries should be evaluated by the controller under their actual source identities before closing Task 2.

## Scope limits

This was a targeted rereview of the original P2 and possible breakage introduced by its fix, not a second broad application audit. I did not rerun a broad suite or competing byte matrix. No new out-of-scope defect was identified. The original honestly documented tests-first chronology limitation remains; no attempt was made to relabel it. Process kill/recovery, unknown sibling behavior, operator closure and causal negative controls remain Task 3 responsibilities.

**Disposition:** P2 addressed; minimal code fix approved within this scope. Overall Task 2 completion is still gated on the pending refreshed root/byte evidence and controller validation of the source-bound supporting runs.
