# Task2a scoped rereview — round 1

## Verdicts

**I1: ADDRESSED.** **Spec compliance: Approved for the gated Task2a foundation.** **Code quality: Approved.** No new issue found in the fix diff. M1 remains a deferred, nonblocking Minor finding and was not reopened.

Scope: exact `23d6ee4fa1fa03fa05bcdae16f998ea73788e91d..64d650fe39b9df1003b19bc416f108fb4e39632d` review package, original task brief, appended fix report, retained commands and test evidence. This approval does not activate public schema4 or qualify the complete invocation protocol.

## I1 verification

The new prefix check resolves the consuming execution's selected observation to its represented source fact and requires that fact's ID in `seen` before accepting consumption (`internal/executor/composition_facts.go:294`). An ID enters `seen` only after the current fact has passed its existing row-link, order, and kind-specific validation (`composition_facts.go:332`). The check therefore requires an earlier validated source transaction, rather than merely finding a source row in the final database.

`compositionObservationPredecessor` follows retained observation and admission evidence, verifies the admitted target/run and exact canonical observation, and verifies the original source descriptor (`composition_facts.go:341`, `composition_facts.go:369`). Its bounded source handling matches the finding:

- Ordinary message: requires the original intent's scoped execution fact (`composition_facts.go:377`).
- Runtime terminal: requires the terminal source execution, except that a terminal obligation linked to a lifecycle operation requires that operation's decision fact (`composition_facts.go:386`, `composition_facts.go:399`). The earlier execution at the same sequence cannot satisfy a later lifecycle closure.
- Runtime lifecycle result: requires the retained target-decision fact (`composition_facts.go:407`). Source-denied settlement does not invent a target decision (`composition_facts.go:415`).

The implementation is confined to read-only validation. It adds no scheduling, admission, commit, source-authorization, artifact-binding, or public-version behavior. Local input and source transactions without represented foundation facts retain their permitted treatment.

## Evidence and test quality

The five corruption cases use real production Commit, delivery and lifecycle/settlement paths (`composition_facts_test.go:225`). The shared assertion first requires successful full foundation validation (`composition_facts_test.go:348`), changes only fact order, checks ordinary inspection tables remain equal (`composition_facts_test.go:389`), checks each fact retains its identity and digest (`composition_facts_test.go:400`), and requires `ErrCorrupt` with the specific selected-observation predecessor rejection (`composition_facts_test.go:404`). This closes the earlier vacuity concern for I1.

Retained RED and GREEN logs establish the same five boundaries: message consumption, execution-produced terminal, lifecycle-produced terminal at sequence zero, lifecycle-produced terminal after an ordinary step, and lifecycle-result consumption after the source request but before the target decision. The last two tests specifically prevent substituting the earlier ordinary execution for the actual lifecycle predecessor. See `review-fix1/02-causal-variants-red.log:1` and `review-fix1/03-causal-variants-green.log:1`.

Four positive controls retain acceptance of unscoped message, unscoped execution terminal, unscoped lifecycle terminal and local input (`composition_facts_test.go:237`; `review-fix1/03-causal-variants-green.log:23`). The unchanged independent original probe is RED before the fix and GREEN afterward (`review-fix1/01-original-probe-red.log:1`, `review-fix1/04-original-probe-green.log:1`). The complete executor package passes (`review-fix1/05-executor-suite.log:1`). Exact command arguments and statuses are appended to `commands.json:221`; final successful outputs contain no unexplained warnings.

I independently compared the retained original probe and overlay to my original `/tmp/alan-task2a-review-probe` copies: both match byte-for-byte. The original `review.md` hash remains `26d2d4aa31b317401e4ab4225111fa573e29ba22fc408dd2fa450b48d642b422`, matching the retained fix report/source manifest. Original failure evidence was preserved.

## New issues and boundaries

**New fix-diff issues: None.** The helper has one bounded responsibility, propagates query/decoding failures, and reuses the existing source-evidence builders. The test additions exercise source distinctions and unchanged valid cases without implementing a second protocol.

**Out of scope:** M1 remains deferred. Adapter/receipt and future invocation/result predecessors require their future transaction facts; complete public schema4 compatibility, dynamic origins, outcomes, budgets, closure and runner behavior remain Task2b/2c work. The complete public profile must retain the controller's distinct version rather than silently promote foundation/v1. No new out-of-scope finding was raised.

**Review checks:** Inspected only the fix diff for code changes and its covering evidence, with no unchanged-code re-review or new source inspection outside the package. No test rerun was warranted: the retained original probe and all requested variants answer the original doubt. No broad suite rerun, live call, credentials, subagent, git/index operation, or source mutation occurred. Only this rereview report is added.
