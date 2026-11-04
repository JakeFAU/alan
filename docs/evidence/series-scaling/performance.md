# Paired performance results

These are the complete predeclared six-pair results, not the pilot. The baseline
`af6943a5805a9a6c0e8fe77f804264f4765d07ec` and production candidate
`e32d0e2f020638f301dcd0f0687c5c444180225c` were exported into independent
directories and built from the same frozen external driver. All 98 processes
passed: 14 excluded warmups and 84 measured processes. No measured row was
discarded, replaced or added. Both complete source manifests matched before
and after collection. The operating tools and binaries matched their frozen
hashes for every sample.

## Ordinary execution

Execute encloses all Run calls and fresh protected snapshot/basis/approval
calls. Ranges are observed minimum–maximum among six samples per source.
The reduction is one minus the ratio of source medians; it is distinct from
the median of within-pair ratios. All times are seconds.

| Positions | Baseline median [range] | Candidate median [range] | Median reduction |
| ---: | ---: | ---: | ---: |
| 1 | 0.004349 [0.003570–0.004719] | 0.004179 [0.003386–0.004466] | 3.9% |
| 8 | 0.103591 [0.097590–0.123983] | 0.090748 [0.085107–0.096926] | 12.4% |
| 16 | 0.366924 [0.347896–0.397138] | 0.294358 [0.274378–0.314432] | 19.8% |
| 32 | 1.632605 [1.554139–2.350832] | 1.081954 [1.005171–1.301357] | 33.7% |
| 64 | 8.250941 [8.204076–8.703398] | 4.211180 [4.108499–5.013851] | 49.0% |
| 128 | 51.130406 [49.648982–53.217608] | 19.771476 [19.464378–21.453907] | 61.3% |
| 256 | 347.985978 [340.830602–383.190931] | 105.522120 [103.383122–108.476672] | 69.7% |

The practical 25% target at 256 was exceeded in this experiment. At 128 the
observed median reduction is 61.3%. All six 256 within-pair candidate/baseline
ratios are below one, ranging 0.269795–0.312398; their median is 0.303618.
The predeclared exact two-sided sign test gives p=0.03125 under independent
paired signs. Six pairs and a shared host provide limited precision; this is
not a universal latency claim or independent-machine replication.

No small-bound guard at 1/8/16/32 exceeded max(5% of baseline median, 0.5ms).
No phase guard at any bound exceeded max(10%, 0.5ms). These empirical guards
are not equivalence proofs. Individual small samples vary in either direction.
The slower baseline 256 samples at 383.190931s and 372.683004s remain included.

## Allocation, CPU and resident memory

Each cell is baseline → candidate median. GB is decimal cumulative Go
allocation; M means million allocation events. CPU is process user+system
time within Execute. RSS is the complete process high-water mark, in MiB
(2^20 bytes), including final readers/recovery/serialization. It is not an
Execute-only or phase-local peak. Go allocation traffic excludes C/SQLite
allocations; resident memory includes them.

| Positions | Allocated GB | Allocation count (M) | CPU seconds | Peak RSS MiB |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 0.001601 → 0.001548 | 0.020 → 0.019 | 0.006 → 0.005 | 23.13 → 23.23 |
| 8 | 0.070451 → 0.055851 | 0.694 → 0.656 | 0.138 → 0.122 | 26.49 → 26.73 |
| 16 | 0.307707 → 0.207780 | 2.619 → 2.388 | 0.488 → 0.402 | 27.89 → 27.94 |
| 32 | 1.533055 → 0.798794 | 10.709 → 9.150 | 2.173 → 1.532 | 31.59 → 31.33 |
| 64 | 8.917112 → 3.201711 | 48.432 → 37.179 | 10.853 → 6.019 | 39.73 → 39.48 |
| 128 | 57.676655 → 13.698727 | 251.553 → 163.885 | 68.383 → 29.262 | 52.86 → 52.71 |
| 256 | 408.168152 → 62.158738 | 1487.651 → 784.258 | 478.565 → 161.676 | 78.38 → 78.80 |

At 256, allocation volume decreases 84.77% and allocation
events decrease 47.28%. CPU time decreases 66.22%.
Median peak RSS increases by 0.430MiB (0.55%).
The observed RSS ranges overlap: baseline 76.48–81.06MiB, candidate
75.36–79.72MiB. This is not a resident-memory reduction result. Raw JSON retains
all six values and ranges for every metric at every bound.

## Separate phase costs at 256

These are independently calculated phase medians; do not add phase medians
to reconstruct the whole-window median. Repeated body phases are summed
within a sample before their cross-sample median is calculated.

| Phase | Baseline seconds | Candidate seconds |
| --- | ---: | ---: |
| `approval_establishment` | 0.206610 | 0.201013 |
| `approval_snapshot_basis` | 15.159015 | 15.880579 |
| `close` | 0.000401 | 0.000294 |
| `close_reopen_resume` | 1.552452 | 1.381568 |
| `construct_provision` | 0.010841 | 0.011149 |
| `create` | 0.246733 | 0.246587 |
| `final_inspection` | 0.105116 | 0.112476 |
| `final_series_view` | 0.000821 | 0.000854 |
| `replay` | 0.857137 | 0.852766 |
| `resume_inspection_view` | 0.113506 | 0.114928 |
| `run` | 332.626496 | 89.623044 |

The fresh approval snapshot/basis total increases from 15.159s to 15.881s,
and final inspection from 0.105116s to 0.112476s; neither crosses its phase
guard. Reader work was not optimized. Completed Close/Open/Run decreases
from 1.552s to 1.382s; Replay remains approximately 0.85s. Independent retained
validation still runs on reopen and Replay. No reader-cost saving is inferred
from the large Execute improvement.

The separate one-sample interval-reopen diagnostics are deliberately excluded
from ordinary medians: at 8, reopening after every selected position takes
0.249665s baseline and 0.233068s candidate; at 32, reopening every 8 positions
takes 2.025801s and 1.426231s. All identities, charges and contacts survive.
These four descriptive samples do not estimate a recovery speedup distribution.

## Work preserved and work removed

Every 256 sample has 256 distinct original effects, local adapter entries, paid
opportunities, retained results and selections; 255 protected successor approvals;
and 1,023 committed reductions. The same evidence survives completed reopening
with no extra contact. These are deterministic local contacts, not live provider
calls or a claim of remote exactly-once application. Exact same-namespace
cross-source comparisons are separately qualified at 8 and on 21 actual crash
cuts; independently created timing stores naturally have distinct namespaces.

The separately instrumented Execute maps at all seven bounds differ only in
full descriptor-decode count and its input-byte counter. At 256, decodes fall
from 399,868 to 1,535 and input traffic from 54.111GB to 0.208GB. Root sweeps remain
1,535, PrepareRoot calls 392,960, root/intent reporting pairs 8,421,376, and
measured SQL executions 11,770,629. The original operations and checks are still
performed. [Attribution](attribution.md) documents probe coverage, overhead,
allocation profiles and the frozen invariant.

## Finite growth comparisons and limits

Constant+linear/quadratic/cubic fits have baseline wall-time RMSE 42.319/12.312/
2.572s and candidate RMSE 11.456/2.277/2.220s over the seven medians. The linear
and quadratic fits predict nonpositive values at small bounds; cubic fits have
large positive offsets that also describe small bounds poorly. Global log-power
fits give exponents 2.037 and 1.821 with systematic residuals. Adjacent 128→256
log-slopes are 2.767 and 2.416. The raw analysis preserves every coefficient and
residual. These finite comparisons do not establish an asymptotic law.

The original host is Apple M4, 10 CPUs, 16GiB, macOS 27.2, Go 1.27.3 darwin/arm64,
CGO 1, SQLite module v1.14.52, `GOMAXPROCS=4`, `GOGC=100` and no Go memory limit. The
controller for this paired campaign is Python 3.15.2; the earlier environment
record separately identifies system Python 3.9.6. Both records are preserved.

Task-launched builds, tests, profilers and other experiments were suspended
during serial timing. Background host load, OS caches, thermals and CPU
placement were uncontrolled. Recorded 1-minute load spans 1.405–16.586, so this
was not an isolated host. Adjacent pairs, balanced AB/BA order and randomized
bound order reduce some ordering confounders; they do not eliminate them.
Source/build/schedule hashes, exact commands, full raw values and before/after
source manifests are retained. The result was obtained outside the development
working directory, but has not been replicated on another machine.

The optimization adds constant storage for one private boolean and no recovery
state. Remaining reporting scans, contextual SQL work, snapshot construction
and other descriptor copies remain. The measured candidate still takes about
105.5s at 256. Root-report membership lookup is a supported next profiling
hypothesis, not a second optimization or a promised improvement in this change.
