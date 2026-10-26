# Measured application usefulness and cost

The one-shot heldout study has **two of four strict quality successes for investigation and one of four for the simple helper**. These are descriptive results on four synthetic cases, not a statistical superiority claim. The larger composition used more model/lookup work. An additional investigation answer supplied all required facts but failed the strict stopping rule because it did not explicitly ask for the missing signed decision; the adjudicator retains the permissive interpretation separately. The contradiction investigation stopped with an empty delivered answer, which receives zero answer credit.

The runtime, model prompts, corpus, run order and operational limits were fixed before [custodian unsealing](evaluation/unseal-report.md). The [source-extension record](evidence/heldout-source-extension.json) proves all185 pre-unseal source files stayed byte-identical when four previously committed data artifacts were published. Gold never enters application input. No post-heldout tuning or outcome-dependent repeat occurred.

## Heldout cases

Support and covered columns count complete frozen fact groups, which differ in semantic size. Coverage is covered required groups, not merely valid bytes. Stop is the independent semantic stopping grade, not the application disposition.

| Case | System | Support | Covered | Stop | Strict quality | Revision | Model/lookup | Seconds | Estimated USD |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| held-01 | baseline | 4/4 | 4/4 | pass | pass | 0 | 2/1 | 11.496 | $0.005148 |
| held-01 | investigation | 4/4 | 4/4 | pass | pass | 1 | 13/4 | 48.658 | $0.041951 |
| held-02 | baseline | 3/5 | 3/5 | fail | fail | 0 | 2/1 | 9.500 | $0.004732 |
| held-02 | investigation | 5/5 | 5/5 | pass | pass | 0 | 6/2 | 21.205 | $0.013230 |
| held-03 | baseline | 1/3 | 1/3 | fail | fail | 0 | 2/1 | 11.235 | $0.005122 |
| held-03 | investigation | 3/3 | 3/3 | fail | fail | 1 | 13/4 | 43.286 | $0.039205 |
| held-04 | baseline | 1/5 | 1/5 | fail | fail | 0 | 2/1 | 7.823 | $0.004326 |
| held-04 | investigation | 0/5 | 0/5 | fail | fail | 1 | 13/4 | 60.073 | $0.044433 |

All30 initial/final heldout citation instances are byte-valid. Validity does not imply relevance: one investigation reference is an irrelevant Birch span for a Maple question and supplies no Maple claim coverage. Precision is1.0 only for the explicitly enumerated, independently supported claim sets in heldout responses01/05/06/08; other denominators are unavailable or empty. Raw grades retain every claim, reason, exact excerpt and source span. No precision value is pooled or inferred for the unavailable cases.

From initial candidate to final, held-01 investigation improved stopping/applicability from false to true with required-fact support unchanged at 4/4; held-03 investigation improved supported required groups from 1/3 to 3/3 while strict stopping remained false; held-02 was unchanged; held-04 regressed to empty delivery despite a retained diagnostic candidate. Diagnostic citation association is unavailable and is not borrowed from the final evidence slots. Revision is an ordinary new authorized round, not reattempt of unknown work.

## All development versions remain in the denominator

| Study | System | Strict successes | Model/lookup requests | Input/output tokens | Median seconds | Estimated USD |
| --- | --- | --- | --- | --- | --- | --- |
| live-development | baseline | 0/4 | 7/4 | 1644/1194 | 9.814 | $0.015228 |
| live-development | investigation | 1/4 | 38/12 | 14430/6505 | 43.773 | $0.097168 |
| live-development-v2 | baseline | 0/4 | 8/4 | 2420/1410 | 10.043 | $0.018940 |
| live-development-v2 | investigation | 1/4 | 45/14 | 21574/7663 | 57.131 | $0.139806 |
| live-development-v3 | baseline | 0/4 | 8/4 | 2420/1536 | 9.933 | $0.020200 |
| live-development-v3 | investigation | 1/4 | 46/14 | 22080/7523 | 70.800 | $0.138797 |
| live-heldout | baseline | 1/4 | 8/4 | 2449/1443 | 10.367 | $0.019328 |
| live-heldout | investigation | 2/4 | 45/14 | 21767/7505 | 45.972 | $0.138819 |

Each development version has one fully satisfactory delivered response out of eight paired outputs. Revision1 exposed an invalid overlong lexical query and incomplete answers; revision2 fixed the query instruction but stopped three investigations with no delivered answer; revision3 retrieved additional evidence and improved a contradiction summary to4/5 groups, but omitted explicit authority/reconciliation context and stopped two other investigations. These are development-informed measurements on the same questions, not three independent heldout experiments. Both systems share the revised helper prompt; only investigation revisions get the optional exclusion profile. The protocol is not an isolated or equal-compute ablation.

The synthetic heldout templates are related to development templates, all corpus bytes were public, and only four cases were sealed. Model alias drift, nondeterminism, order effects, host/network variability, retrieval layout and coarse fact grouping limit inference. One sample per case/system cannot establish stable performance, variance, broad domain usefulness or statistical significance. Latency includes the existing controller barriers and complete local executor work; it is not a provider-only benchmark. No expensive full-suite run overlapped the paid heldout comparison.

## Accounting and evidence

All33 live runs including the old gate used 207 model requests and 71 lookups, with 89,185 input and 35,044 output tokens. Estimated token charges total **$0.591738**, against the shared **$10** authorization. The controller's conservative accounting total is $0.993089; no live unknown-cost reservations remain. These are available-usage/rate estimates, not invoices. All33 keyless replay checks passed. Controlled unknown-outcome process tests remain a separate blocked-progress result.

[Machine-readable joined results](evidence/evaluation-results.json) retain per-run source identities, usage, costs, raw paths and adjudication mappings. Each raw study directory retains configuration, immutable store, observations/attempts, provider IDs, raw outcomes, citation evidence, replay and spending-gate acknowledgments. [The shared ledger](evidence/live-budget.json) accounts for every released live operation.

| Study | Blind grading | Independent adjudication |
| --- | --- | --- |
| live-development | [grades](evidence/live-development/grading-report.md) | [adjudication](evidence/live-development/adjudication.md) |
| live-development-v2 | [grades](evidence/live-development-v2/grading-report.md) | [adjudication](evidence/live-development-v2/adjudication.md) |
| live-development-v3 | [grades](evidence/live-development-v3/grading-report.md) | [adjudication](evidence/live-development-v3/adjudication.md) |
| live-heldout | [grades](evidence/live-heldout/grading-report.md) | [adjudication](evidence/live-heldout/adjudication.md) |

The graders did not author the application or use its reviewer as an oracle. Opaque IDs conceal explicit system labels, but answer/citation structure can suggest the system. Revision2 adjudication reuses an agent with prior-study context and discloses that limit; final development and heldout adjudicators are fresh. These are independent agent reviews, not external human evaluation.
