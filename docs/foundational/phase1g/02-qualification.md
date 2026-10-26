# Phase 1G qualification — in progress

The starting baseline and contract review are complete. Implementation qualification is not complete; this document deliberately makes no invocation, crash-recovery or acceptance claim yet.

## Completed starting evidence

Merged main `80dead19846d9b034b4457a9436e43acb81da3df` has the same 189 contributing files as the Phase 1F frozen source `2dacb402e50ee938d6184c14b40ba469fdcb8d91d9c30eac178b0dcb2a0fd3f8`. Historical source was extracted and run independently of subsequent edits. Default 550 and integration 756 pass events, both vet modes, formatting, module verification and 30 Python controls passed. All 1,120 prior foundational artifact files remained unchanged. [Baseline report and exact commands](evidence/baseline/baseline.md) distinguish historical reproduction from future current-source comparison.

The independent [feasibility assessment](evidence/feasibility.md) compared reusable participants with retained template invocation. The [initial contract review](evidence/contract-review.md) required six operational refinements. Its [rereview](evidence/contract-rereview-r1.md) retained two completion counterexamples: a committed reply still awaiting admission and a committed child request still awaiting admission. The [final addendum](evidence/contract-rereview-r1-final.md) passes the refined contract only. Exact reviewed text versions are retained and hash-verified in [contract-source-retention.json](evidence/contract-source-retention.json).

The [independent fixture expectations](evidence/fixtures/acceptance-plan.md) were written before the implementation. They specify fixed inputs and direct expected integer outputs, ID correspondence, boundary acknowledgments and monitor duties. They are planned fixtures, not executed evidence.

## Reviewed implementation slices

Typed collections and ordinary finite snapshot iteration are implemented in `a1477e30fa172a44f7ce932458a2847ff88d83d8`, with the independently discovered scalar-append panic fixed in `4180ba540d9b35ff92b4e1273a6365906cb18d9a`. The [task report](evidence/task1/report.md) gives public APIs, exact scalar compatibility, version selection and commands. Focused checks and the default suite passed; the final guard repair received fresh focused/schema-package verification. [Independent review](evidence/task1/review.md) and [scoped approval](evidence/task1/rereview-round1.md) are retained. This qualifies the collection/iteration slice only, not child creation or the final work-set application.

The first invocation implementation assignment returned a concrete coupling counterexample before changing product code: per-run final histories do not retain enough cross-run order to verify failure-prefix reply freezes. Its [unchanged checkpoint](evidence/task2/report.md) records that incomplete attempt. The plan was split into ordering/binding foundation, admission/outcomes, then resource enforcement/runner/full replay. All three must complete before exposing schema4 publicly; there is no partial invocation qualification claim.

The gated ordering/binding foundation is implemented at `23d6ee4fa1fa03fa05bcdae16f998ea73788e91d`. Its [report](evidence/task2a/report.md) retains the executed same-final-history/different-order witness and transaction tests. [Independent review](evidence/task2a/review.md) found that order-only corruption could move a message consumption before its scoped source commit without validation rejecting it. The original probe and output are retained. Fix `64d650fe39b9df1003b19bc416f108fb4e39632d` requires the exact source commit or lifecycle decision in the preceding order. Five order-only corruption regressions, four positive controls, the unchanged reviewer probe and the executor package passed; [scoped rereview](evidence/task2a/rereview-round1.md) approved the fix. Public schema4 activation remains rejected, and this slice establishes no invocation admission, completion or resource-conservation claim.

## Evidence-to-claim status

| Claim | Evidence type | Status |
| --- | --- | --- |
| Starting main reproduces retained behavior | Historical source execution and source-byte comparison | Passed as scoped above |
| New invocation contract is operationally specified | Independent specification review | Passed; not implementation proof |
| One artifact handles variable typed work | Public API and independent result checks | Pending reviewed implementation |
| Logical work exceeds physical capacity without parent deadlock | Causal controlled-provider and nested schedules | Pending |
| Accepted work, partial results and pending operation survive process death | Acknowledged boundaries, SIGKILL, new-process inspections | Pending |
| Closure/late/unknown results preserve identities and budgets | Selected schedules and independent monitors | Pending |
| Historical replay cannot create or dispatch work | Credential-free/no-network replay and immutable ledger/contact checks | Pending |
| Evidence helper is an ordinary effectful template consumer | Separate-module application and controlled provider | Pending |
| General dynamic spawning, arbitrary recursion, liveness, remote cancellation or business exactly-once | None | Excluded |
| New live provider smoke or answer-quality improvement | None | Not run/not claimed; previous study's authorization and ledger not reused |

Final qualification will replace pending rows with exact fixtures, source identities, raw commands/results and limitations. A failed or excluded case will remain explicit. Initial root provisioning/template catalogue size, work-proportional records, local active physical work, latency and recovery are separate measurements.
