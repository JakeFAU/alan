# Repeated root membership reconstruction is a measured baseline cost

The baseline generates 11,770,629 Execute SQL executions at B=256. Of these,
8,429,051 (71.61%) use one root-membership family; 8,421,376 are independently
attributed to the root × intent loop inside `finishRun`. Separate coarse timing
places root reconstruction at 52.959s of a 104.087s Execute window. This supports
investigating that repeated reconstruction in the next invariant review. It
does **not** select a production mechanism or predict its speedup. Public
approval snapshots and readiness remain substantial costs; root reporting is
not the largest sampled allocation caller.

## Evidence boundary

| Identity | Frozen revision |
| --- | --- |
| Production base | `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315` |
| Experiment contract | `34409c56a37a5f5917913ac56a4807479f145375` |
| Public workload | `9d1209a8069a52fbcf71f2176061eef66d6b9fe8` |
| SQL/probe tools | `14ea2effa91ad0a1f101dcf59fd400717508dc4f` |
| Reviewed summarizer | `0fd0f2fd78e23f86320b25befb3b650db840e305` |

All 427 baseline source-file identities agree before/after diagnostics. Ordinary,
SQL-counter and coarse binaries have separate retained build records and SHA-256
identities (`47d5165e…ab721`, `13aeb7ac…55b16`, `4c5a1c68…5ce86` respectively;
full hashes are in [baseline-data.json](baseline-data.json)). The coarse generator
was unchanged across the raw-tail SQL correction; its earlier generation/build
manifest remains its actual provenance, rather than being relabeled as rebuilt.

The host was Apple M4, Mac16,10, 10 CPUs, 16 GiB RAM; macOS 27.2/26B123,
Go 1.27.3 darwin/arm64, Apple clang 21, CGO=1, go-sqlite3 v1.14.52 and
Staticcheck 2026.2.2. The controller recorded Python 3.9.6. Processes used
`GOMAXPROCS=4`, `GOGC=100`, `GOMEMLIMIT=off`, `GOWORK=off`, local toolchain,
offline modules and the frozen environment in [contract.md](contract.md).
All 41 rows were collected serially: seven single ordinary samples, six direct
bridge samples, sixteen prefix-calibration samples, seven SQL-counter rows,
two coarse rows, two CPU/allocation-profile rows and one trace row. These are
preliminary diagnostics, **not confirmatory paired medians**. No live provider
was contacted. HTTP handler and adapter journals independently reconcile retained
identities, while their resources are measured in the same process.

## Ordinary resources and descriptive calibration

Each ordinary row is `ordinary-bB`; Q comes from the **separate** `counters-bB`
process. GB below means 10⁹ bytes. RSS is the whole-process high-water from
`wait4`, including phases after Execute; GC pause is process pause accumulation.

| B | Execute wall s | User / system CPU s | Alloc GB | Mallocs | GC count / pause ms | Process RSS MiB | Q, separate counter row |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 0.005020 | 0.004366 / 0.002819 | 0.001898 | 22,034 | 1 / 0.021 | 24.44 | 331 |
| 8 | 0.085748 | 0.090694 / 0.027080 | 0.057667 | 661,665 | 27 / 0.756 | 27.83 | 6,274 |
| 16 | 0.284590 | 0.315138 / 0.082239 | 0.212425 | 2,402,864 | 110 / 3.224 | 30.52 | 20,706 |
| 32 | 1.027954 | 1.200887 / 0.286330 | 0.812294 | 9,181,416 | 433 / 11.495 | 35.36 | 79,330 |
| 64 | 4.192326 | 5.047737 / 1.179674 | 3.245988 | 37,267,599 | 1,688 / 42.535 | 46.98 | 358,395 |
| 128 | 19.826720 | 23.920802 / 6.004338 | 13.858319 | 164,189,227 | 6,105 / 157.100 | 64.39 | 1,916,389 |
| 256 | 105.268848 | 125.863856 / 34.191562 | 62.594949 | 784,590,239 | 16,333 / 487.690 | 105.11 | 11,770,629 |

Every B256 row retains 256 original contacts/charges/results/selections, 255
protected approvals and 1,023 reductions; HTTP rows also retain 256 matching
handler contacts. No forced GC occurred. B256 Execute GC CPU-class accumulation
was 13.803s, an asynchronous runtime metric not additive to user/system CPU. Its Execute-end
RSS gauge was 79.89 MiB; the process later peaked at 105.11 MiB. Neither gauge is
62.595 GB of cumulative allocation.

The excluded direct bridge has no independent endpoint journal and disables
prefix collection. Its three fresh runs span 1.014102–1.021833s at B32
(median 1.021798s), and 105.259094–112.434058s at B256 (median 106.023046s).
Against the historical rounded P1 medians 1.082s/105.522s, these medians differ
by −5.56%/+0.47%. They reproduce the same approximate scale; these are separate
campaigns with new harness bookkeeping and uncontrolled host conditions, not a
paired change estimate or evidence of an HTTP transport saving.

Prefix calibration used on/off then off/on order at each B, without extra
warmups. Actual Execute on/off observations are:

| B | Pair 1 on / off s | Pair 2 on / off s | On−off allocation bytes, pair 1 / 2 |
| ---: | ---: | ---: | ---: |
| 1 | 0.004524 / 0.004868 | 0.004619 / 0.004416 | +12,640 / +27,832 |
| 8 | 0.088993 / 0.088984 | 0.089944 / 0.089704 | −93,824 / +78,536 |
| 32 | 1.093029 / 1.096503 | 1.100195 / 1.080924 | −169,112 / −132,336 |
| 256 | 104.592931 / 104.615407 | 105.120513 / 105.221769 | +19,689,888 / −8,300,008 |

Two pairs cannot establish negligible overhead. The differing signs, allocation
and GC histories are preserved; no overhead estimate is subtracted. Heavy SQL
probe resource/timing values do not enter any ordinary resource claim.

## SQL origin, scope and accounting limits

At B256, completed prepares, invocation attempts and VM executions each equal
11,770,629; actual C step calls are 14,691,675, `SQLITE_ROW` returns 13,779,321,
fullscan steps 11,502,540 and VM steps 217,020,394. Bind/step errors are zero.
Executions are SQLite statement-status VM starts, not API success counts;
buffered row consumption is not an extra C step. Row returns/fullscan steps do
not count exact rows examined. Automatic reprepare and direct connection-setup
VM work are unavailable.

For all seven counter rows the reviewed tool validates all nine total/family/
scope-class-post-state partitions, including per-family attribution. There is
no family/cell overflow or invalid scope. Known semantic scope is nevertheless
incomplete: B256 has 3,069 unknown executions, all the `COMMIT` family, and
9,207 unknown VM steps. The tagged `finishRun` owner accounts for 9,046,656
executions (76.86%), `selectReady` 1,259,513, `PrepareRoot` 1,180,160 and
`InspectExecution` 14,025. A snapshot can read many rows with few statements.

The following top ten comprise 11,360,572 executions (96.52%). All are reads;
“point” describes SQL predicates, not an independently measured query plan.
Full normalized SQL, all nine metrics, exact per-scope counts and connection/
post-state cells are retained in the JSON. Scope aliases name the actual tagged
operations: F=`finishRun`, P=`PrepareRoot`, S=`selectReady`, R=`Run`, C=`Commit`,
A=`ApproveContinuation`, U=`AuthorizeIntent`, W=`prepareAdapterWork`,
T=`recordAdapterResult`, D=`recordDispatchAdmission`, Z=`SettleRoot`,
I=`InspectExecution`.

| Read family / resource | Executions | Share | Shape | Semantic scopes |
| --- | ---: | ---: | --- | --- |
| Membership: static/dynamic members | 8,429,051 | 71.61% | Union of two subject+generation points | F=8,421,376; C,R,W,T,U,A,D |
| Root descriptor: composition_roots | 526,846 | 4.48% | ID point | P,F,C,R,W,A,Z |
| Run/program/authority projection: runs+subjects | 496,507 | 4.22% | Subject+generation join | S,F,C,W,A,U,R,T |
| Original root publication | 459,264 | 3.90% | Scope point | P,F,Z |
| Subject generation/retirement | 362,110 | 3.08% | ID point | S,C,R,U,A |
| Settlement presence | 262,656 | 2.23% | ID count | R,F |
| Composition closure | 230,783 | 1.96% | Scope point | P,F,C,Z |
| Composition freeze | 230,272 | 1.96% | Scope point | P,F,C,Z |
| Member/binding descriptor | 181,925 | 1.55% | Subject+generation join | S,C,A,U,R,W |
| Schema version: metadata | 181,158 | 1.54% | Singleton point | S,C,W,F,T,A,I |

There are 3,069 successful transaction begins and commits, with 255 requested
read-only and 2,814 requested read-write BeginTx calls. BeginTx/common-begin
counters are nested API levels and must not be added. Execute opens/closes 255
connections and records 1,275 successful direct setup calls, whose internal SQL
work is excluded from the families above. Connection class is configuration,
not proof of per-statement write activity or enforced read-only transactions.
`post_autocommit` is observed after stepping, not a transaction/snapshot ID;
prepare/attempt events have unknown post-state. Contextless Commit retains
unknown scope rather than borrowing its nearest Go caller.

## Fixed-declaration recurrence and actual marginals

[Source](../../../internal/executor/composition_inspect.go) reads the roots and
intents, then re-sorts intent keys and resolves membership for each root ×
intent pair. Under this successful schedule, R=B and I=k at the kth finish.
Thus direct loop membership is Bk, cumulative membership is Bk(k+1)/2, and the
complete series contributes B²(B+1)/2. This is a measured schedule-specific
source recurrence, not an unrestricted runtime complexity or protocol lower
bound. Legitimate absent membership still returns empty; it must not become an
error. Other callers remain separate: B256 has 7,675 membership executions
outside `finishRun`.

The extraction checked **every direct and cumulative prefix at all seven
bounds**: 505 prefix positions, each reconciled independently between source-loop
counters and finishRun-scoped SQL executions. Full-series pair counts are
1, 288, 2,176, 16,896, 133,120, 1,056,768 and 8,421,376. At B256 that membership
subtree has zero fullscan steps and 117,899,264 VM steps. Its dominant count is
repeated point work, rather than evidence of a slow membership range scan.

Selected actual adjacent intervals for **fixed B=256** follow. T/A come from
`ordinary-b256`; Q comes from `counters-b256`, a different process. Each interval
includes its approval when required. [baseline-marginals.csv](baseline-marginals.csv)
retains all 505 positions, both sample IDs and cumulative/direct quantities.

| Retained k | Adjacent T ms | Adjacent alloc MB (10⁶ bytes) | Adjacent Q | Membership Q |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 134.005 | 91.842 | 12,798 | 256 |
| 8 | 208.260 | 153.172 | 18,035 | 2,048 |
| 32 | 252.908 | 175.465 | 23,433 | 8,192 |
| 128 | 403.573 | 248.522 | 45,510 | 32,768 |
| 256 | 628.484 | 317.170 | 76,258 | 65,536 |

T(256,8)=1.523648s, distinct from standalone T(8,8)=0.085748s because all 256
roots were declared initially. The final cumulative prefix leaves a measured
24,292ns/3,120-byte outer Execute residual. No unrelated sample subtraction is
called an exact marginal.

Descriptive normalizations distinguish the visible growth from a simple linear
explanation without pretending seven single samples establish an asymptote:

| B | T/B ms | T/B² ms | T/B³ µs | Q/B² | Q/B³ |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 32 | 32.124 | 1.004 | 31.371 | 77.471 | 2.421 |
| 64 | 65.505 | 1.024 | 15.992 | 87.499 | 1.367 |
| 128 | 154.896 | 1.210 | 9.454 | 116.967 | 0.914 |
| 256 | 411.206 | 1.606 | 6.275 | 179.606 | 0.702 |

T/B rises; T/B² also rises at the larger bounds, while T/B³ falls. Total work
mixes the exact cubic-leading membership count with other repeated root,
readiness and growing snapshot work. Allocation/B² rises from approximately
0.793MB at B32 to 0.955MB at B256. These ratios do not identify an exclusive
latency law, per-query cost or significance level.

## Time, allocation and the fifteen requested categories

Coarse timers are separate inclusive wall diagnostics. At B256 the root helper
is 50.88% of that row's Execute and 96.91% of its enclosing `finishRun`; these
fractions are neither uninstrumented fractions nor predicted savings.

| Inclusive coarse boundary | B128 s | B256 s | B256 calls / mean ms |
| --- | ---: | ---: | ---: |
| Outer Execute | 19.648 | 104.087 | 1 / 104,086.892 |
| finishRun | 7.493 | 54.651 | 256 / 213.480 |
| readRootCompletions, inside finishRun | 7.066 | 52.959 | 256 / 206.873 |
| selectReady | 5.679 | 24.347 | 1,535 / 15.861 |
| Approval InspectExecution | 3.804 | 15.095 | 255 / 59.197 |
| settleRoots | 1.697 | 7.047 | 1,535 / 4.591 |

The B128 CPU profile has 27.87s sampled CPU over 21.55s collection duration.
`runtime.cgocall` is the largest flat frame, 9.76s (35.02%). Root reconstruction
has 5.34s cumulative, including 4.52s at its membership call; focused samples
mostly descend through SQLite stepping/finalization. Native/runtime frames,
asynchronous SQL workers and unlabelled HTTP/runtime work prevent a complete
application-category or SQL-family CPU partition. Stage labels cover 19.22s
(68.96%); nested labels replace the parent label. CPU, trace delay and wall
totals cannot be added.

Actual `alloc_space` output displays **MB**: total 13,938.09MB, selectReady
5,106.66MB (36.64%), all InspectExecution 4,617.48MB, root helper 1,773.12MB and
profile descriptor 319.16MB. Root-focused membership contributes 1,247.61MB
inside that helper. These are sampled cumulative Go-heap estimates with
MemProfileRate=524288, broader collection scope and publication lag; they are
not exact Execute MemStats bytes, C allocations, retained storage or RSS.
Among the compared selection, typed-inspection, root-reporting and descriptor
components, selectReady has the largest sampled cumulative allocation. Larger
overlapping totals include parent Store.Run at 8,559.47MB and the cross-cutting
executor.decode at 6,469.47MB. These parent/decoder/component values are never
added as disjoint allocations. No B256
allocation profile exists, so B128 fractions are not applied to B256 bytes.

The following coverage map uses B256 counter counts/Q and coarse wall unless
another sample is named. “Unavailable” means no exclusive measurement; it does
not mean zero. Categories and their nested helpers overlap.

| Requested category | Supported attribution and remaining limit |
| --- | --- |
| 1. Reducer/state transition | 2,046 Reduce calls: 1,023 Commit and 1,023 selectReady calls. Exclusive time/alloc/Q unavailable. |
| 2. Operation construction/commit | 1,023 Commit calls, 0.480s inclusive, 27,621 directly scoped SQL executions; 0.469ms/call. Exclusive construction allocations unavailable. |
| 3. Original authority | 256 AuthorizeIntent calls, 0.061649s, 4,864 scoped Q. Final rechecks also occur in other scopes; exclusive authority allocation unavailable. |
| 4. Grants/approval | 255 commitments, 0.214157s and 6,885 scoped Q; 255 fresh public snapshots, 15.095s and 14,025 Q. Ordinary approval-basis interval allocates 18.915GB; commitment 0.049880GB. Grant-only split unavailable. |
| 5. Reservation/accounting | 256 accountAdapter calls, 0.162976s inclusive. Reservation SQL crosses tagged preparation/Commit/root operations; exclusive Q/alloc unavailable. |
| 6. Preparation/admission | 256 prepareAdapterWork calls, 0.204421s/9,470 Q; dispatch recording 256, 0.021857s/1,536 Q. Transport 0.065594s. Exclusive allocation unavailable. |
| 7. Retention/settlement | Each occurs 256 times: result retention 0.106300s/6,144 Q; settleAdapter 0.056453s, locked settlement scope 2,048 Q. Exclusive allocation unavailable. |
| 8. Observation admission/selection | 1,535 selectReady calls, 24.347s/1,259,513 Q; 360,320 runs and 29,358,102 observation-loop entries considered. B128 sampled inclusive allocation 5,106.66MB. Admission-only split unavailable. |
| 9. Root/resource reports | 256 report reads, 65,536 root-loop entries, 8,421,376 root×intent pairs. Timing/profile detail above; finishRun's SQL scope includes other report work. |
| 10. Immutable profile/validation | 1,535 descriptor decodes/207,720,805 input bytes; invocation-mode predicate called 398,333 times. B128 descriptor allocation 319.16MB. Ordinary full-ledger validation counters are zero. |
| 11. SQLite prepare/execute | Nine distinct metrics above; root-focused sampled C calls localize a path, not per-family CPU. Native allocation and exclusive prepare-versus-step time unavailable. |
| 12. Transactions/connections | 3,069 begin/commit pairs; 255 opens/closes and 1,275 setup calls. No per-handle distribution, full setup VM count or exclusive CPU/alloc split. |
| 13. Serialization/decoding | Typed snapshot scans materialize 1,403,520 rows/699,555,435 text/blob bytes. B128 generic decode is 6,469.47MB inclusive across callers. Neither is total serialized output or a separate additive category. |
| 14. Allocation/GC | Ordinary B256: 62.595GB, 784,590,239 mallocs, 16,333 GC cycles, 0.487690s accumulated pause and 13.803s GC CPU-class delta; asynchronous process accounting. |
| 15. Owner renewal | No open/create/renewal inside ordinary Execute. Separately, B256 Close/Open/resume is 1.441751s/0.728740GB, including validation and resume; renewal-only split unavailable. |

Creation, initial verification, final inspection/view/proof, reopen/resume,
Replay and Close remain separate in the JSON. For example, ordinary B256 Create
is 0.257601s and Replay 0.871994s, outside its 105.268848s Execute. The hypothesis
that repeated full-ledger validation dominates ordinary Execute is contradicted
by zero validator entries in counter/coarse Execute diagnostics. Public typed snapshot validation is
different, measured work and remains necessary for every subsequent approval.

## Semantic work and the next review boundary

An additional independently approved observation retains a distinct original
operation/reduction, source authorization, preparation/reservation and any debit,
final dispatch admission, original result or unresolved uncertainty, settlement,
selected observation and durable successor/continuation evidence. The first
position has three reductions; later positions add protected approval consumption.
Current authority, generation/epoch, grants, remaining scope and causal
relationships must be checked at their owning cuts. These are necessary fact
classes under the existing contract, not proof of a minimum SQL/write count or
an asymptotic lower bound. Read-back checks and reconstruction of derived reports
are different from creating the durable facts they verify.

The measured root × intent reconstruction is justified for **separate invariant
review**: establish which repeated membership resolution is unnecessary within
one supported stable operation while preserving generation keys, absent
membership, ordering, errors and every derived result. `finishRun` holds the
owner mutex on its pinned connection; Create/Open/Replay/Inspect use their own
transaction/snapshot cuts. A previously correct root report cannot be cached
across mutable membership, authority, charges, settlement, selection or receipt
evidence. Typed approval snapshots are a different implementation path and
cannot be removed by a root-helper change. No mechanism is approved here.

One host, no CPU affinity/thermal isolation, uncontrolled caches and observed
host load 1.127–4.358 limit inference. B16 trace delay sums overlap concurrent
goroutines and do not prove connection-pool contention or provider latency.
Unmeasured native/runtime ownership and instrumentation overhead remain open.
The next gate must supply an invariant, falsifiable prediction and adversarial
challenge before production edits; later paired timing and exact recovery/
compatibility comparisons determine whether any chosen change is accepted.

## Reproduction and artifact map

Use [run_schedule.py](run_schedule.py), the frozen external
`baseline-{timing,diagnostic}-schedule.json` and
[summarize_diagnostics.py](summarize_diagnostics.py). The latter's `--schedule`,
`--results` and `--out NEW_DIR` CLI validates immutable rows without needing their
original executable paths. External `baseline-{timing,diagnostic}-results/`
retain raw/meta/progress files; `baseline-{timing,diagnostic}-summary-initial/`
retain reviewed summaries/indexes. The analysis-source review and completed-data
review both returned APPROVE; independent raw re-extraction sampled B8, while
root's reviewed-tool run validated the complete matrix.

`baseline-profile-analysis/commands.json` preserves 32 successful pinned pprof/
trace interpretations and exact commands; `reviews/profile-attribution-supplement/`
adds three focused interpretations. `profile-tool-identities.json` records the
actual tool hashes. `reviews/source-cost-model.md` and
`reviews/profile-attribution-initial.md` preserve source/temporal and profile
reviews. Earlier probe RED/rejections and corrections remain in the separate
Task 2 records and were not erased by successful extraction.

[baseline-data.json](baseline-data.json) records every input summary hash, its
raw/meta/progress/schedule hashes, source/analysis hashes, complete Execute SQL
families, all ordinary phase resources, diagnostic modes and exact recurrence
checks. External `task3-report-extraction/` preserves the **initial** extraction
scripts, commands, schema-reading correction and output hashes. Its two-argument
extractor is historical evidence: the subsequently documented replay failed
because it attempted to recreate auxiliary files inside the immutable inputs.
That actual failure remains in `task3-reproduction-red/`; the initial report
source and records remain in `task3-report-fix1/initial/`.

The current versioned extractor reproduces the published LF CSV using four
explicit paths:

```sh
python3 /path/to/task3-report-fix2/extract-v3.py \
  --inputs /path/to/immutable-p2-inputs \
  --source /path/to/reviewed-source-export \
  --out /path/to/NEW-data-directory \
  --aux-out /path/to/NEW-auxiliary-directory
```

Both output directories must be absent, distinct, non-nested and outside the
input/source directories. It checks these conditions before writing and places
preview/hash records only in `--aux-out`. Round-1 v2 reproduction in
`task3-report-fix1/` remains tied to the original CRLF CSV bytes. A later packaging
check flagged those 506 carriage-return record terminators as trailing whitespace;
the actual failure remains in `task3-staged-whitespace-red/`.

The published [CSV](baseline-marginals.csv) now uses LF record terminators. The
original CRLF artifact and v2 results remain immutable. Version 3 specifies
`lineterminator='\n'`; `task3-report-fix2/` retains its successful fresh-output
command, parsed equality of every header/cell, and byte equality between v3 and
the published LF CSV. All JSON data and input hashes agree excluding only the
declared extraction command, script identifier and script hash. The repository
JSON remains unmodified with its original extraction provenance. This is a
record-terminator formatting correction, with no semantic normalization or new
measurement. Existing reviews remain bound to their earlier hashes pending
scoped re-review. No raw large counter report or database was opened. Delivery
archive links will be added when sealed artifacts exist.
