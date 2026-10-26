# Phase 1G qualification — in progress

The starting baseline and contract review are complete. Implementation qualification is not complete; this document deliberately makes no invocation, crash-recovery or acceptance claim yet.

## Completed starting evidence

Merged main `80dead19846d9b034b4457a9436e43acb81da3df` has the same 189 contributing files as the Phase 1F frozen source `2dacb402e50ee938d6184c14b40ba469fdcb8d91d9c30eac178b0dcb2a0fd3f8`. Historical source was extracted and run independently of subsequent edits. Default 550 and integration 756 pass events, both vet modes, formatting, module verification and 30 Python controls passed. All 1,120 prior foundational artifact files remained unchanged. [Baseline report and exact commands](evidence/baseline/baseline.md) distinguish historical reproduction from future current-source comparison.

The independent [feasibility assessment](evidence/feasibility.md) compared reusable participants with retained template invocation. The [initial contract review](evidence/contract-review.md) required six operational refinements. Its [rereview](evidence/contract-rereview-r1.md) retained two completion counterexamples: a committed reply still awaiting admission and a committed child request still awaiting admission. The [final addendum](evidence/contract-rereview-r1-final.md) passes the refined contract only. Exact reviewed text versions are retained and hash-verified in [contract-source-retention.json](evidence/contract-source-retention.json).

The [independent fixture expectations](evidence/fixtures/acceptance-plan.md) were written before the implementation. They specify fixed inputs and direct expected integer outputs, ID correspondence, boundary acknowledgments and monitor duties. They are planned fixtures, not executed evidence.

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
