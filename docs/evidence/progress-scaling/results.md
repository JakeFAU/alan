# Measured execution progress

At B256, median execution-window time on the frozen candidate was
**4.585 s with full Run and 1.298 s with RunProgress**, a 71.7% reduction.
All six B256 block contrasts favored progress by 3.254–3.355 s. The
measured lifecycle still included a final full report, typed inspection,
completed reopen and independent Replay. Its median fell from **6.424 s
to 3.156 s**. Baseline full and candidate full medians were 4.543 s
and 4.585 s for the execution window and 6.405 s and 6.424 s for the
lifecycle. [Frozen qualification](qualification.md)

## Measurement

The harness ran on one Apple M4 host using a new Store and separate child
process for each row. Each bound had one excluded warmup per arm and six
measured blocks, one for each of the six permutations of the three arms.
All 147 rows passed, including 126 measured rows. All builds completed before
the primary children ran serially. Every arm used the same public bounded
series, deterministic direct adapter, targeted approval and common progress
and contact checks. The final report and typed checks ran after the window.
Every B256 sample proved 256 original completed, selected, spent and
contacted operations, 255 protected approvals and 1,023 reductions. The
same-seed weather route independently checks exact retained identity across
the two candidate methods.

The execution window includes calls to Run or RunProgress, targeted
continuation basis reads, approval writes, and the common checks. The
per-operation clocks measure elapsed time only. Lifecycle includes
construction, Create, execution, final inspection and proof, close and
reopen, final verification and Replay. Process CPU is user plus system
time. Allocation bytes and count measure cumulative allocation activity.
Execute-end RSS is the cumulative process high-water mark at that
boundary. Post-exit peak RSS covers the entire reaped child, including
final JSON serialization. The observed peak RSS ranges overlap.

[Ordered rows](qualification-data/timing-rows.jsonl) ·
[All stage medians, observed ranges and block contrasts](qualification-data/timing-summary.json)

Median elapsed milliseconds [observed minimum–maximum].

| Bound | Baseline full | Candidate full | Candidate progress | Paired full minus progress |
|---:|---:|---:|---:|---:|
| 1 | 4.436 [4.045–4.610] | 4.299 [4.069–4.493] | 3.840 [3.710–4.221] | 0.352 [0.128–0.767] |
| 8 | 45.188 [44.666–46.437] | 45.419 [44.802–46.441] | 41.169 [40.695–41.537] | 4.283 [3.778–4.905] |
| 16 | 95.681 [95.265–96.665] | 96.473 [95.907–98.552] | 80.884 [80.393–81.412] | 15.399 [15.087–18.158] |
| 32 | 211.784 [210.114–219.231] | 213.468 [211.321–229.341] | 158.295 [157.322–159.629] | 55.369 [52.099–71.711] |
| 64 | 518.348 [514.501–529.679] | 520.996 [517.402–522.552] | 317.290 [312.625–332.090] | 201.808 [189.800–209.927] |
| 128 | 1426.780 [1415.448–1462.803] | 1439.197 [1430.001–1544.986] | 641.570 [630.705–677.572] | 791.644 [766.710–914.281] |
| 256 | 4543.353 [4533.997–4798.407] | 4585.003 [4563.804–4654.659] | 1298.190 [1292.643–1309.453] | 3291.106 [3254.350–3355.416] |

| B256 metric | Baseline full | Candidate full | Candidate progress |
|---|---:|---:|---:|
| Construction and provision, ms | 10.493 [10.270–11.778] | 10.348 [10.233–10.425] | 10.378 [10.152–10.721] |
| Create, ms | 275.635 [268.855–277.178] | 276.965 [275.618–278.664] | 276.499 [274.571–277.168] |
| Basis reads, ms | 154.900 [153.757–163.798] | 154.552 [152.756–156.480] | 125.110 [124.312–128.401] |
| Approval writes, ms | 115.073 [112.177–125.416] | 116.983 [113.190–121.811] | 100.828 [98.571–102.172] |
| Run calls, ms | 4276.035 [4263.743–4509.078] | 4312.116 [4294.512–4380.511] | 1073.079 [1065.910–1080.544] |
| Execute allocated MiB | 3227.979 [3227.436–3228.562] | 3228.845 [3227.899–3229.242] | 637.357 [636.959–638.040] |
| Execute allocations, millions | 43.984 [43.981–43.989] | 43.993 [43.990–43.995] | 8.710 [8.708–8.715] |
| Execute-end RSS, MiB | 32.789 [32.547–33.219] | 32.977 [32.016–33.656] | 32.109 [31.828–32.547] |
| Exit peak RSS, MiB | 79.016 [78.000–80.438] | 79.781 [77.547–80.906] | 78.633 [76.797–80.125] |
| Final full report, ms | 23.866 [23.338–23.993] | 24.120 [23.656–24.525] | 22.275 [22.113–22.531] |
| Final typed inspection, ms | 109.964 [108.374–111.684] | 109.452 [108.198–115.186] | 108.927 [108.279–112.448] |
| Fresh Open, ms | 698.794 [690.383–708.819] | 696.973 [693.441–698.381] | 701.251 [695.840–711.318] |
| Full Run after reopen, ms | 37.875 [35.644–38.524] | 37.912 [37.315–38.530] | 37.771 [37.371–38.237] |
| Reopen inspection and view, ms | 109.962 [108.825–116.422] | 110.511 [109.862–117.379] | 111.046 [109.950–119.273] |
| Replay, ms | 566.234 [562.956–570.065] | 563.708 [533.996–565.327] | 568.874 [564.048–579.470] |
| Lifecycle, ms | 6405.379 [6380.444–6639.783] | 6424.068 [6403.236–6510.420] | 3156.178 [3148.391–3177.312] |
| Lifecycle allocated MiB | 4745.324 [4744.872–4746.159] | 4746.300 [4745.555–4746.941] | 2155.217 [2154.197–2155.993] |

| B256 CPU | Baseline full | Candidate full | Candidate progress |
|---|---:|---:|---:|
| Execute, s | 6.655 [6.631–7.022] | 6.723 [6.702–6.813] | 1.853 [1.849–1.861] |
| Lifecycle, s | 9.258 [9.206–9.606] | 9.305 [9.279–9.404] | 4.454 [4.425–4.486] |

Full control and baseline B256 execution ranges overlap and their paired
differences have both signs. The slowest baseline row (4.798 s) and the
slowest candidate-full row (4.655 s) remain included. The B1 lifecycle
contrasts between the two candidate methods also have both signs. These
observations cover this bounded program, host, toolchain and transport.
Weather itself still takes typed snapshots around execution, and these
timings do not measure weather end-to-end latency. Approval stage times
were lower in the progress arm even though their code is shared, so
those times include the process context left by prior work.

## Diagnostic lanes

The separate counter binaries verified the same workload and final evidence.
The probes partitioned Execute SQL inside and outside `finishRun`.
The progress arm emitted zero roots during Execute. Both full arms emitted
64 at B8 and 65,536 at B256.

| Bound and arm | Execute SQL | In finishRun | SQLite row returns | VM steps | Full-scan steps |
|---|---:|---:|---:|---:|---:|
| 8 baseline full | 5,312 | 348 | 4,728 | 76,770 | 602 |
| 8 candidate full | 5,312 | 348 | 4,728 | 76,770 | 602 |
| 8 candidate progress | 4,964 | 0 | 3,960 | 62,346 | 182 |
| 256 baseline full | 397,884 | 233,344 | 854,240 | 15,835,407 | 717,570 |
| 256 candidate full | 397,804 | 233,344 | 854,160 | 15,834,967 | 717,570 |
| 256 candidate progress | 164,428 | 0 | 131,184 | 2,739,111 | 227,970 |

At B256, the report partition accounted for 722,944 row returns,
13,095,680 VM steps and 489,600 full-scan steps. The 32 extra
non-report SQL executions in candidate full are eight additional calls
to each of the four reads in the unchanged `readExecutionStamp`.
The counter data identifies those reads and does not isolate why the
helper ran more often. The counter lane overlapped integration race
testing and supplies work counts only.

| Remaining full-scan steps by SQL template | B8 | B256 |
|---|---:|---:|
| `INSERT INTO history VALUES(?,?,?,?,?,?,?,?)` | 105 | 130,305 |
| `SELECT result FROM attempts WHERE intent_id=?` | 56 | 65,280 |
| `SELECT count(*) FROM composition_charges WHERE root_id=? AND kind=?` | 21 | 32,385 |

These are the complete non-report full-scan totals in all three arms
at the two diagnostic bounds. The latter two queries repeatedly scan
growing tables without a matching declared index. They retain
quadratic work on this series, with counts matching `n(n-1)` and
`(n-1)(n-2)/2` at B8 and B256. The history insert counts match
`(2n-1)(n-1)`. That statement may scan the observations child table
while checking its deferred foreign key. This attribution follows
the source and counter pattern and remains unisolated experimentally.
The bounded observations do not establish an asymptotic or latency
bound for other programs. Create, Open, inspection and Replay retain
their global validation work.

[Counter rows](qualification-data/counter-rows.jsonl) ·
[Residual registry extraction](qualification-data/residual-counters.json)

B256 CPU and sampled allocation profiles ran in separate processes after
the original qualification controller exited. Both profile children
passed their checks. The first render setup compared a metadata object
to its digest and exited before rendering. The subsequent render bound
the same formatter, producer binaries and profile bytes and passed. The whole-lifecycle allocation profile
attributed 2,599.18 MiB (55.16% of sampled bytes) cumulatively to
`finishRun` in candidate full and 31.54 MiB (1.50%) in candidate
progress. The progress profile still includes the two explicit full
reports after Execute. `validateCompositionStore` attributed 923.54 MiB
in full and 935.10 MiB in progress. These sampled cumulative stacks
include descendants and can overlap. They measure allocation activity
across the lifecycle. [Profiles and render identities](qualification-data/profile-evidence.json)
