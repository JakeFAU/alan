# Bounded execution scaling results

On the qualified static series at B256, guarded owner discovery and grouped full reporting reduced median Execute elapsed from **46.813259 s to 22.351172 s** with snapshot approval in both arms. A separate candidate-only experiment reduced it from **22.316851 s to 6.860208 s** by reading the continuation basis directly from the owner. Both experiments retained full RunReports. Each met the frozen B256 target of at least 25% median improvement and six positive paired differences. These measurements apply to the series, builds and host below. [Contract](structural-contract.md) · [Qualification and reproduction](qualification.md) · [Delivery decisions and reviews](delivery-decisions.md)

## Measured elapsed and resources

The schedule ran serially at bounds 1, 8, 16, 32, 64, 128 and 256. It retained 21 excluded warmups and 168 measured rows with six adjacent pairs per bound and comparison, ordered AB/BA/AB/BA/AB/BA. Structural A/B means integrated baseline snapshot / candidate snapshot. Targeted A/B means separately measured candidate snapshot / candidate targeted. Both use direct transport, no interval reopen and ordinary binaries with profiling and diagnostic probes absent. Execute includes all Run calls, their complete reports and prefix proofs, and the basis read and approval write at each continuation. [Frozen apparatus](task5/README.md) · [Ordered rows with source report hashes](timing-rows.jsonl)

Median and observed range in milliseconds:

| B | Structural A | Structural B | Targeted A | Targeted B |
|---:|---:|---:|---:|---:|
| 1 | 3.888 [3.809–4.133] | 4.287 [4.077–4.465] | 4.307 [3.961–4.712] | 4.295 [3.946–4.516] |
| 8 | 78.072 [77.408–78.347] | 62.953 [62.672–63.987] | 62.085 [61.428–63.779] | 47.638 [46.293–47.762] |
| 16 | 236.411 [235.456–248.306] | 167.359 [165.124–168.347] | 165.856 [165.172–166.927] | 104.317 [103.411–108.731] |
| 32 | 798.118 [794.538–800.759] | 499.789 [494.068–503.275] | 497.723 [493.779–501.349] | 246.874 [244.547–249.710] |
| 64 | 2,889.986 [2,877.940–2,906.132] | 1,646.966 [1,635.914–1,673.945] | 1,646.016 [1,637.128–1,647.976] | 654.239 [648.769–663.573] |
| 128 | 11,309.742 [11,269.787–11,530.701] | 5,864.356 [5,824.577–6,062.476] | 5,843.428 [5,831.049–5,893.367] | 1,972.116 [1,966.736–1,976.173] |
| 256 | 46,813.259 [46,568.209–46,860.246] | 22,351.172 [22,270.330–22,384.730] | 22,316.851 [22,274.949–22,430.646] | 6,860.208 [6,852.840–6,876.402] |

All Execute differences in pair order, A minus B in milliseconds:

| B | Structural | Targeted |
|---:|---|---|
| 1 | −0.252, −0.161, −0.332, −0.387, −0.517, −0.591 | 0.711, 0.147, −0.177, 0.140, −0.331, 0.063 |
| 8 | 15.388, 15.675, 14.519, 14.215, 14.081, 14.934 | 13.666, 17.486, 14.028, 14.651, 14.637, 14.461 |
| 16 | 80.527, 67.524, 69.977, 71.692, 68.574, 68.326 | 62.372, 61.795, 61.690, 60.844, 58.196, 60.143 |
| 32 | 294.811, 300.469, 299.215, 300.600, 298.983, 294.558 | 256.466, 252.165, 244.128, 248.854, 252.600, 249.400 |
| 64 | 1,255.612, 1,266.469, 1,245.796, 1,241.708, 1,230.712, 1,214.902 | 983.540, 987.350, 984.403, 996.885, 991.488, 990.232 |
| 128 | 5,477.763, 5,410.183, 5,457.423, 5,556.008, 5,407.989, 5,468.225 | 3,866.675, 3,882.419, 3,918.778, 3,859.991, 3,870.834, 3,866.833 |
| 256 | 24,451.624, 24,429.777, 24,505.307, 24,254.117, 24,589.916, 24,235.194 | 15,398.851, 15,460.761, 15,454.646, 15,439.361, 15,474.742, 15,554.245 |

Structural B1 was slower in all six pairs. Its median rose **0.398874 ms (10.260%)**. The frozen guard requires a slowdown above both 10% and 1 ms at B1/B8/B32, so neither comparison crossed it. Targeted B1 has zero approvals and mixed pair signs. B256 median elapsed fell 52.255% and 69.260% in the respective experiments. Every original row completed. The observed ranges retain slower rows, including the structural B128 candidate at 6,062.476 ms and the targeted B256 snapshot at 22,430.646 ms.

| B256 resource median | Structural A | Structural B | Targeted A | Targeted B |
|---|---:|---:|---:|---:|
| Execute user + system process CPU, s | 68.932257 | 30.332277 | 30.285162 | 10.589256 |
| Execute cumulative allocated bytes | 42,192,060,724 | 23,370,762,724 | 23,370,914,128 | 3,906,393,764 |
| Execute allocation count | 459,905,443.5 | 246,155,724 | 246,152,064.5 | 54,491,279 |
| Execute GC cycles | 14,948 | 4,476 | 4,465.5 | 2,176.5 |
| Execute GC pause, ms | 384.295 | 116.835 | 116.169 | 57.320 |
| High-water process RSS at Execute end, bytes | 74,629,120 | 74,956,800 | 74,964,992 | 34,611,200 |
| Latest sampled high-water process RSS, bytes | 84,664,320 | 86,786,048 | 85,434,368 | 84,754,432 |

Process CPU sums user and system time and can exceed elapsed. Allocation bytes/counts measure churn. No GC cycles were forced in Execute. RSS is a cumulative process gauge. The latest sampled value is the maximum over recorded stages, through Replay and before final JSON serialization. The targeted arms' latest sampled peaks are close after the independent audits, despite the large difference at Execute end. The summary gives exact medians, ranges and all six paired differences for every phase and resource. [Timing rows](timing-rows.jsonl) · [Resource and phase summaries](timing-summary.jsonl) · [Calculation commands](qualification.md#reproduction)

| B256 phase median elapsed, s | Structural A | Structural B | Targeted A | Targeted B |
|---|---:|---:|---:|---:|
| Construct provision | 0.010495 | 0.010546 | 0.010763 | 0.010542 |
| Create | 0.262726 | 0.282110 | 0.282194 | 0.283257 |
| Initial snapshot proof | 0.011049 | 0.011243 | 0.011187 | 0.011105 |
| Approval basis reads | 15.366677 | 15.315061 | 15.314596 | 0.159007 |
| Approval establishment | 0.193198 | 0.193679 | 0.189336 | 0.123691 |
| Run, including full reports | 31.122440 | 6.727569 | 6.714502 | 6.494233 |
| Final inspection | 0.109042 | 0.109177 | 0.109250 | 0.109572 |
| Completed close/reopen/resume | 0.745569 | 0.753658 | 0.751526 | 0.749417 |
| Inspection and view after reopen | 0.110513 | 0.111265 | 0.110856 | 0.110568 |
| Replay | 0.582223 | 0.574648 | 0.573439 | 0.570025 |

Repeated stages are summed per row before taking medians. The Execute window is measured directly. These are finite observations from one Apple M4 workstation. Recorded host load, fixed order and adjacent balancing constrain drift, with shared-host activity, thermal conditions and OS/filesystem caches still possible influences. The bounds provide neither a fitted wall-time law nor population confidence bounds.

## Reached work

The baseline repeated quadratic walks. For this schedule there are `(6B−1)B` settlement root checks and `B²` report root visits. Each successive full report matches B roots against the growing intent prefix, giving the source-supported cubic term `B²(B+1)/2` for that traversal. These are algorithmic counts. The original 3,246,427 statement result came from the earlier baseline. The new integrated baseline executes 3,246,682 at B256. [Original baseline](baseline.md) · [Correction contract](structural-contract.md)

Execute diagnostic counts, in baseline snapshot / candidate snapshot / candidate targeted order:

| B | SQL executions | SQLite returned rows | SQLite fullscan steps | SQLite VM steps |
|---:|---|---|---|---|
| 1 | 332 / 483 / 483 | 231 / 389 / 389 | 0 / 0 / 0 | 5,085 / 5,894 / 5,894 |
| 8 | 6,204 / 5,651 / 5,672 | 6,950 / 6,089 / 4,892 | 3,626 / 2,499 / 490 | 120,301 / 95,462 / 76,954 |
| 16 | 18,720 / 12,695 / 12,740 | 24,158 / 16,801 / 11,656 | 14,970 / 9,015 / 2,130 | 386,676 / 255,081 / 183,987 |
| 32 | 61,992 / 30,431 / 30,524 | 88,910 / 51,665 / 30,368 | 66,650 / 33,759 / 8,866 | 1,364,676 / 757,681 / 480,811 |
| 64 | 221,342 / 80,495 / 80,684 | 339,681 / 175,153 / 88,528 | 328,986 / 130,095 / 36,162 | 5,228,998 / 2,496,321 / 1,405,467 |
| 128 | 836,954 / 238,991 / 239,372 | 1,329,135 / 637,169 / 287,792 | 1,833,626 / 510,159 / 146,050 | 21,685,680 / 8,907,351 / 4,578,791 |
| 256 | 3,246,682 / 789,455 / 790,220 | 5,255,119 / 2,421,361 / 1,018,096 | 11,515,290 / 2,019,855 / 587,010 | 97,668,208 / 33,464,471 / 16,221,607 |

Each B256 diagnostic retains 256 distinct original operation IDs with ordered matching contacts, 256 spent, completed and selected originals, 255 protected approvals and 1,023 committed reductions. The compact data preserves the actual operation IDs, root IDs, prepared/result digests, observation IDs and selection positions for each independently created namespace. The direct and HTTP differential uses same-seed copies for exact identity equality. [Diagnostic rows and identity projections](diagnostic-data.jsonl) · [Differential qualification](qualification.md#semantic-and-recovery-evidence)

- **Owner discovery.** At B256 the baseline considers 360,320 runs and 29,358,102 observation candidates, globally validates 392,448 observation rows and checks 392,960 settlement roots. The candidate considers 1,790 ready candidates and prepares 1,790 dirty roots, validates 1,022 changed observation IDs and performs 1,533 total observation point validations. It copies 2,045 ranks, visits 767 entries during compaction, handles 256 pending original IDs, and records 10,744 successful guards and 13,558 stamp reads. All ordinary candidate series use the optimized scheduler. Create initializes 256 roots, 256 runs and 1,023 instructions at B256, increasing Create SQL from 16,054 to 16,586. Ready/dirty visits scale as 5, 54, 110, 222, 446, 894 and 1,790 across the measured bounds. Large live frontiers, large roots and unsupported operations have separate costs. [Owner index](../../../internal/executor/static_execution.go)
- **Full reports.** One grouping pass per report replaces 8,421,376 root-by-intent comparisons at B256 with 32,896 grouping visits. The baseline exposes two counter names for the same matching loop. Grouped member appends depend on which root sorts first and have a different meaning from total membership checks. Charge preparation goes from 65,536 scalar queries, 8,355,840 fullscan steps and 26,968,832 VM steps to 256 grouped queries returning 32,896 groups, with 32,640 fullscan steps and 2,771,200 VM steps. Groups returned are distinct from source rows scanned. The aggregate preserves the original error route when required. [Report implementation](../../../internal/executor/composition_inspect.go)
- **Targeted basis.** Snapshot Execute performs 255 `InspectExecution` calls, materializes and validates 1,403,520 inspection rows, and visits 32,640 snapshot continuation slots. The Execute snapshot path does not replay the ledger or invoke its reducer. The targeted path reads 255 predecessor ranges returning 255 originals, checks attempt, dispatch admission and result for each, and performs 2,040 fact and 510 observation point reads. For basis reads alone, SQL goes from 14,790 to 15,555 statements, from 1,417,035 to 13,770 returned rows, from 1,432,845 to zero fullscan steps and from 17,455,260 to 212,396 VM steps. The overall Execute statement count increases by 765 while returned rows fall by 1,403,265 and VM steps fall by 17,242,864. The separate approval transactions remain. [Snapshot materialization](../../../internal/executor/inspection_execution.go) · [Targeted read](../../../internal/executor/continuation_basis.go)
- **Remaining cost.** At B256 all modes emit 65,536 full report root records. The candidate still prepares and visits 65,536 report roots, visits 65,536 runs and admission rows, performs 32,896 member checks and retains per-root and per-intent lookups. Candidate `finishRun` alone records 625,792 statements, 886,912 returned rows, 359,040 fullscan steps and 13,482,496 VM steps. The existing reservation charge-count query adds 256 executions with 32,385 fullscan steps. Initial/final inspection, completed reopen and Replay are separately measured. B256 reopen and Replay each reach independent ledger validation and 1,023 history reductions. The default B full B-root reports impose a quadratic emission floor, and measured Execute SQL remains quadratic in this schedule. The output requirement still leaves room to test whether batching the remaining report descriptor and status lookups can reduce SQL and allocation churn while preserving the complete report and errors. [Reservation](../../../internal/executor/composition_reservation.go) · [Diagnostic stages](diagnostic-data.jsonl)

The 31 diagnostics comprise 21 ordinary matrix counts, three B256 interval-reopen counts, three B256 whole-process profile runs and four same-source B1/B8 original/compact calibrations. The independent review checked SQL family/scope partitions for all 28 instrumented reports, with zero attribution overflow or incompleteness. Original/compact comparisons agree on shared counters with 3,907 and 13,854 checked equalities. The absent probe keys indicate no reached event or a source-specific vocabulary gap. Candidate-only counters have no baseline equivalent. The interval-reopen rows are single diagnostics with seven in-window reopens at B256, kept apart from timing.

The unfiltered B256 CPU profiles contain 61.24, 28.29 and 11.59 sampled CPU seconds in the diagnostic order. The allocation-space profiles report 41,624.87, 23,829.85 and 5,252.26 pprof-displayed MB. Baseline `selectReady` accounts for 12,407.42 MB cumulative allocation, candidate snapshot `scanInspectionTable` for 2,731.97 MB and candidate targeted `readRootCompletions` for 2,054.70 MB. Candidate targeted `finishRun` is 3.41 s cumulative sampled CPU and 2,979.37 MB cumulative allocation. Stacks overlap and the profiles cover the whole process, including construction, audits and reopen. The profiles provide sampled attribution and the primary experiment supplies the phase timings. [Profile report and raw hash index](diagnostic-data.jsonl)
