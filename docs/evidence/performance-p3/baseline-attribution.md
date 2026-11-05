# Current-baseline attribution

Repeated same-program loading remains material as allocation/validation work on
integrated baseline `926a899b651fccc48be52a054612a92afcf1ddfd`. At B256, the
selector performs **178,601 successful loads for only 1,537 summed distinct
ProgramIDs per call**. A separate B128 sampled allocation profile attributes
2.50GB to the readiness callsite of loadProgram. This supports independent
review of a single-call reuse hypothesis; it does not establish a candidate
speedup. Naive shallow reuse is unsafe for handoff callers, as documented in the
[contract](contract.md) and [source map](source-map.md).

## Fresh pilot, without historical relabeling

The ten-row declared pilot ran serially after the baseline ordinary suite.
Four ordinary fresh-process samples, four separate SQL/P3 counter samples, one
coarse B256 sample and one ordinary-source B128 profile all exited zero with the
unchanged public driver's complete workload invariants satisfied. Primary
samples use `-mode=direct -prefix=true`. Historical P2 HTTP rows and their
57.7476s median are different source/conditions and are not current measurements.

| Bound | Ordinary Execute s | Execute CPU s | Allocated bytes | Mallocs | GC cycles / pause ms | Whole-process peak RSS MiB | Separate Execute SQL |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | .011623375 | .008580 | 1,578,128 | 20,120 | 0 / 0 | 23.5625 | 332 |
| 8 | .172410625 | .199943 | 56,881,432 | 663,229 | 23 / 1.177 | 26.7344 | 6,443 |
| 128 | 21.133361 | 34.949328 | 12,714,023,784 | 137,029,951 | 5,743 / 305.753 | 55.7656 | 923,883 |
| 256 | 76.035488625 | 128.670410 | 52,937,999,232 | 557,742,947 | 20,830 / 1,097.970 | 81.6563 | 3,600,555 |

These are single observations, not medians or significance estimates. CPU sums
user+system within Execute. Go allocation traffic excludes C allocation; RSS is
the whole-process lifetime peak, including later readers. No GC was forced.
Shared-host one-minute load was approximately 7.24–7.68 during ordinary rows;
thermal state, caches and CPU placement were uncontrolled. The separate coarse
B256 Execute was 59.783s and profiled B128 Execute 16.374s; differing conditions
and observer overhead preclude treating these as timing replications or exact
overhead corrections. No heavy task-launched tests/builds overlapped the pilot.

At B256 every positive row preserves 256 original effects, adapter contacts,
charges, results and effect selections; 255 fresh public later approvals; and
1,023 committed reductions. Execute includes all 256 Run calls and 255 fresh
snapshot/basis/approval cycles. Provisioning, Create, final reports, completed
reopen/Run and Replay remain separately measured in [baseline-data.json](baseline-data.json).
Direct contacts are not HTTP contacts or independent remote applications.

## Actual traversal and reuse distribution

| Bound | selectReady calls | Eligible visits / successful loads / bindings | Sum distinct IDs per call | Repeated visits | Observation attempts / selections | Ready selections | Visit min / median / p90 / max |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | 5 | 4 | 4 | 0 | 1 | 3 | 0 / 1 / 1 / 1 |
| 8 | 47 | 172 | 49 | 123 | 15 | 31 | 0 / 3 / 7 / 8 |
| 128 | 767 | 44,297 | 769 | 43,528 | 255 | 511 | 0 / 53 / 113 / 128 |
| 256 | 1,535 | 178,601 | 1,537 | 177,064 | 511 | 1,023 | 0 / 120 / 226 / 256 |

Median is the middle sorted call; p90 is index floor(.9*(calls−1)). Histograms
retain every selected/waiting/error return with its actual visit/distinct count,
including the one empty terminal scan per sample. They do not substitute total
provisioned runs for actually visited runs. Selected-call counts differ from
observation selections because emit/end can be ready without an observation.
At B256 the 511 selected observations include 255 local approval observations
and 256 effect results. Readiness Reduce calls remain 1,023; every successful
binding still executes, including waiting runs.

The program registry has two genuine identities: first program
`654c2f541d8da3440e1cfa4dd462bd65ba569a7180356389b3c4aa67ae8d4b1c`
(2,493 bytes, three instructions, four visits at each measured bound), and
later-approval program
`d743c2967842407012a0bd0e982240e4700b03fa36b276665e317a0525733e07`
(4,083 bytes, four instructions, 178,597 B256 visits). The latter has a local
approval port; neither has a HandoffPeer port. Read-only retained-program
inspection and before/after database hashes are in raw `program-summary.json`.
B1 has no repeated loads and is an actual no-benefit control.

Successful load, decode and unbound validation counts each equal eligible
visits in these positives; failed loads and readiness errors are zero. Decoder
input bytes are 9,972; 695,916; 180,858,291; and 729,221,523 respectively.
B256 potential reusable visits are 99.139% of eligible loads, but this is **not**
a predicted percentage reduction in total runtime. The exact load and two-query
predictions are frozen in the contract; all per-run binding, Reduce validation,
observation/root loops and approval work remain.

## Separate timing, SQL and sampled attribution

The separate B256 coarse Execute is 59.783221959s. Inclusive scopes are:

| Scope | Calls | Inclusive wall seconds |
| --- | ---: | ---: |
| selectReady | 1,535 | 26.068090998 |
| Fresh approval InspectExecution | 255 | 16.897814209 |
| settleRoots | 1,535 | 6.935186704 |
| finishRun | 256 | 6.819227953 |
| readRootCompletions, nested in finishRun | 256 | 5.229464131 |
| pendingDeliveries | 1,791 | 2.064294665 |
| Commit | 1,023 | .506894302 |

These scopes overlap and are not additive exclusive categories. Execute makes
zero full-ledger validation/Create/Open calls; ordinary execution does not
repeatedly run full Replay. Fresh typed approval inspection is distinct work.

B128 allocation sampling reports 12,837.03MB total and 5,129.55MB inclusive in
selectReady. Its loadProgram callsite is 2.50GB; loadProgram across all callers
is 2,849.66MB (22.20% of this profile), including 417.90MB at the program SQL row
fetch, 1.63GB at decode, 35MB at the request-authority format query, 715.30MB at
validateProgram and 4.50MB at hashing. These are literal pprof display units,
sampled cumulative allocation with default 524288-byte sampling, full-process
scope/publication lag and overlapping parents. They are not exact Execute
MemStats, native allocations or peak memory, and are never summed with their
parents. Per-run binding at the readiness callsite still allocates 93.01MB;
readObservations allocates 1.13GB. Typed inspection is 4,767.79MB inclusive.

The corresponding CPU profile sampled 24.12 CPU-seconds over 17.46 wall seconds.
Readiness has 2.84s inclusive; its loadProgram callsite .44s, binding .42s and
readObservations .59s. Across all loadProgram callers, .51s is sampled (.27s
program SQL row fetch, .15s feature query, .06s decoding, .03s validation).
Large system-stack/native/runtime components include cgocall 5.88s flat,
pthread_cond_signal 4.84s flat and madvise 3.68s flat. Caller samples do not
capture exclusive waiting or all GC/scheduler work caused by allocations. CPU
attribution therefore does not independently promise a large latency gain.
The allocation mechanism and required paired timings must establish usefulness.

The reused SQL overlay reports B256 3,600,555 VM executions/prepares/invocation
attempts, 6,521,601 actual step calls, 5,609,247 SQLITE_ROW returns, 11,515,290
fullscan steps and 101,382,472 VM steps. Bind/step errors and attribution overflow
are zero. Actual rows examined and connection-setup internal VM work remain
unavailable. Unknown semantic scope remains explicit in raw maps; overflow-free
is not complete semantic attribution. The integrated source adds authority
format checks and differs from historical P2, so neither the old total nor a
cross-campaign ratio is substituted for these counts.

## Verification and limitations

Baseline ordinary suite: exit0 in 62.422s, 1,295 pass events, zero fail events.
Five skip events comprise three packages with no tests and two frozen-old-reader
format13/14 subtests because the external old binary was unset. These are explicit
later qualification obligations, not compatibility passes. All tracked-source
before/after inventories agree. No production candidate exists in this task.

Unchanged P2 campaign tests: 22 pass, no skips, including post-exit removed and
unreadable executable controls. P3 tooling: four pass, including exit7 plus
self-deletion with preserved outcome/usage and synthetic credential exclusion,
distribution-accounting mutants, fail-closed anchors and schedule balance.
P2 generator anchors: three pass with the actual pinned SQLite source. Reused
SQLite overlay ordinary/race proofs pass, and the deliberately double-escaped
normalizer mutant fails for its intended reason. Ten pprof interpretations pass.

Preserved setup failures: two malformed quoting attempts in the short probe;
then a corrected sandbox loopback bind denial followed by separately named
authorized success; one sandbox Go build-cache denial followed by an authorized
build; one P3 generator anchor ambiguity corrected by limiting the anchor to
loadProgram; and two read-only zsh navigation globs that matched no files.
These are not failed workloads erased by retry. All declared pilot rows passed
once. Failure directories and source-generation remnants remain indexed.

Gate recommendation: proceed only to independent attribution/invariant review.
Conservative exclusion of mutation-sensitive handoff programs appears to preserve
the measured opportunity without changing binding ownership. A nested-alias
counterexample or failed stable-read premise defeats unconditional reuse. The
next tasks must test those claims, qualify direct paired validation, and measure
the single change under the frozen guards. Root reconstruction, observation scans,
public snapshots and per-run binding remain visible and are not optimized here.
