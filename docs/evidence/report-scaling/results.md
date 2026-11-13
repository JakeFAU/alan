# Full report scaling results

Batching report-local root descriptors and run projections reduced median
Execute elapsed at B256 from **7.031 s to 4.613 s (34.4%)** on the frozen
bounded observation series. All six B256 pairs favored the candidate, by
2.297–2.545 s. The candidate still produced every full report and kept the
existing report validation. Execute allocation fell while both measured
RSS medians rose. [Qualification and reproduction](qualification.md) ·
[Contract](contract.md)

## Paired timing

Both arms use the protected targeted continuation basis, deterministic
direct adapter, no interval reopen and a new Store per process. The
uninstrumented timing run retained 14 excluded warmups and 84 measured
rows across seven bounds. Within each bound the six adjacent pairs
alternate AB and BA, with A as baseline and B as candidate. The Execute
window includes all Run calls, complete reports and prefix proofs, and
the basis read and approval write at each continuation. Each repeated
stage is summed within the original row before taking medians.
[Ordered timing rows](qualification-data/timing-rows.jsonl) ·
[All stage/resource ranges and paired differences](qualification-data/timing-analysis.json)

Execute elapsed in milliseconds, median [observed min–max] and the six pair differences.

| B | Baseline | Candidate | Paired A minus B |
|---:|---:|---:|---|
| 1 | 4.494 [4.106–4.817] | 4.376 [3.992–4.452] | -0.283, 0.282, 0.260, 0.333, 0.664, -0.224 |
| 8 | 48.453 [47.569–65.882] | 46.311 [45.768–46.574] | 1.598, 1.181, 2.250, 19.444, 3.371, 2.389 |
| 16 | 107.372 [105.614–107.852] | 98.790 [97.736–100.935] | 7.991, 8.633, 6.785, 8.740, 6.917, 8.289 |
| 32 | 249.598 [247.220–266.969] | 217.339 [216.174–220.208] | 31.849, 49.456, 28.556, 33.461, 34.007, 30.819 |
| 64 | 669.071 [660.192–725.922] | 530.531 [527.335–542.278] | 195.610, 131.249, 126.421, 133.865, 147.635, 129.980 |
| 128 | 2,019.164 [1,995.050–2,029.506] | 1,468.330 [1,455.877–1,476.575] | 573.628, 555.342, 530.372, 544.313, 528.473, 542.063 |
| 256 | 7,030.927 [6,966.338–7,135.436] | 4,613.167 [4,588.235–4,669.765] | 2,409.975, 2,544.621, 2,391.077, 2,439.421, 2,425.546, 2,296.573 |

| B256 metric | Baseline median [min–max] | Candidate median [min–max] |
|---|---:|---:|
| Execute CPU, s | 10.864 [10.750–10.964] | 6.789 [6.742–6.860] |
| Execute allocation, MiB | 3,726.687 [3,726.080–3,727.809] | 3,277.785 [3,277.333–3,278.173] |
| Execute allocation count | 54,502,782.000 [54,497,021.000–54,510,657.000] | 44,156,733.500 [44,153,965.000–44,158,458.000] |
| Run sum elapsed, s | 6.657 [6.591–6.753] | 4.262 [4.238–4.316] |
| Run sum CPU, s | 10.369 [10.248–10.453] | 6.314 [6.262–6.376] |
| Run sum allocation, MiB | 3,550.336 [3,549.824–3,551.581] | 3,101.294 [3,101.102–3,101.808] |
| Run sum allocation count | 52,620,016.000 [52,615,057.000–52,628,622.000] | 42,273,927.500 [42,271,685.000–42,276,157.000] |
| Process RSS at Execute end, MiB | 33.133 [32.734–33.703] | 33.742 [32.906–34.562] |
| Post-exit process peak RSS, MiB | 81.750 [81.094–85.109] | 83.125 [80.703–85.828] |
| Final inspection elapsed, ms | 109.250 [108.855–110.337] | 109.264 [108.848–118.539] |
| Completed close/reopen/Run elapsed, ms | 760.842 [755.661–762.044] | 738.834 [733.766–759.495] |
| Inspection/view after reopen elapsed, ms | 110.801 [110.356–111.746] | 110.563 [110.037–110.874] |
| Replay elapsed, ms | 576.078 [573.556–577.311] | 566.866 [565.449–568.076] |

B1 has mixed pair signs. B8 through B256 each have six positive differences.
The B8 baseline row at 65.882 ms and all other slow rows remain in the
comparison. The finite observations share one host, OS, filesystem and
measurement configuration. Adjacent pairs and alternating order limit
drift. Host activity, caches and thermal effects may still influence
elapsed time.

Process CPU is user plus system and can exceed elapsed. Allocation bytes
and counts measure cumulative churn. Execute-end RSS is the cumulative
process high-water sampled at that phase boundary. The post-exit
high-water is the operating system's single-child result after termination.
It includes the independent checks and final JSON serialization. The
observed ranges for both RSS measurements overlap. Inspection, completed
reopen with another Run, and Replay have their own measured costs.

Every timing and counter row checked the complete returned reports against
the original root membership, operation IDs, selections, charges and
outcomes. At B256 each arm retained 256 physical direct adapter contacts,
256 original selected, spent and completed operations, 255 protected
approvals, and 1,023 committed reductions. Independent final inspection,
completed reopen and Replay verified the same original evidence; reopen
added zero contacts. The locally scoped direct calls use a deterministic
adapter. The weather recovery runs have separate loopback provider
journals. Independently created timing Stores have different namespaces.
The exact cross-arm comparisons use matched same-seed Stores.

## SQL diagnostics

Four separate instrumented rows measure B8 and B256 for each arm. Each
validated the SQL registry and its report-scope partition without overflow
or incomplete events. The probes provide statement, row-return, VM-step
and full-scan counts. [Diagnostic rows and original IDs](qualification-data/counters-rows.jsonl) ·
[SQL family counts](qualification-data/counters-analysis.json)

| B | Scope | Executions A → B | Returned rows A → B | VM steps A → B | Full-scan steps A → B |
|---:|---|---:|---:|---:|---:|
| 8 | Execute | 5,672 → 5,312 | 4,892 → 4,728 | 76,954 → 76,770 | 490 → 602 |
| 8 | finishRun | 708 → 348 | 932 → 768 | 14,608 → 14,424 | 308 → 420 |
| 256 | Execute | 790,220 → 397,772 | 1,018,096 → 854,128 | 16,221,607 → 15,834,791 | 587,010 → 717,570 |
| 256 | finishRun | 625,792 → 233,344 | 886,912 → 722,944 | 13,482,496 → 13,095,680 | 359,040 → 489,600 |

The five replaced point-query families executed 376 statements at B8 and
392,960 at B256. One root batch and one run batch per report replaced them
with 16 and 512 statements, respectively. Other report executions
totaled 332 and 233,088 in both arms. At B256 the root batch joined
publication, freeze and closure presence and used 4,524,032 VM steps.
The run batch used 1,181,696 VM steps. Each added 65,280 full-scan steps.
The combined net change was 392,448 fewer report executions, 163,968 fewer
SQLite returned rows, 386,816 fewer VM steps and 130,560 more full-scan steps.
The timing figures come from the separately built uninstrumented binaries.

Across B reports of B roots, full output still includes B² root records.
The diagnostic emitted 64 at B8 and 65,536 at B256 in each arm. The reader
still scans the root and run projections for each report, and remaining
per-intent validation and lookups accumulate over the growing prefix.
The candidate therefore retains quadratic work on this schedule. Complex
unpublished roots with retained freeze or closure evidence and unsupported
batch shapes can reach original point reads. The two measured counter bounds
and seven timing bounds support the observed comparison on this series.

## Report contract

Each call-local reader lives inside the existing ownership or read
transaction boundary. It reads raw rows and applies the original decoding
and error semantics at the original use sites. Failed speculative reads
fall back to the original point path. The frozen oracle and selected runtime
histories exercise report fields and ordering, partial errors, original
identities and independent inspection and Replay. The qualification exercises successful and refusal
histories, unknown outcomes, retained corruption and real process recovery.
[Implementation](../../../internal/executor/report_read.go) ·
[Qualification](qualification.md) · [Rejected variants](rejected-variants.md)

The compact result remains deferred. The executable witness completes an
unrelated legitimate run and detects a corrupt older admission while building
the full report. Omitting the deferred report validation loses that error.
A smaller result that retains the validation could save output allocation,
whose effect remains unmeasured here. [Decision and reproducer](compact-result-decision.md)

The reader holds raw root/publication descriptors and retained run
projections until the call returns. Its transient memory therefore scales
with the total bytes loaded, including projections for runs whose decoded
values do not appear in the final report. The RSS observations measure the
small projections in this deterministic series. Large projections and
different retained histories have a separate memory cost.
