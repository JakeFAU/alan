# Measurement-only aggregate instrumentation

These files generate a separate ALAN source copy, SQLite source copy and public
measurement-driver copy. No counter is compiled into the production checkout.
The committed Go payload uses `.go.txt`; generated `.go` files exist only in the
scratch copies. This instrumentation adds no module dependency or production API.
The generated public `measurecounter` package and SQLite `MeasurementSnapshot`
exist solely so the copied external driver can collect counters without unsafe
access, `linkname`, a live-owner cache, or database shortcuts.

The normal driver is built unchanged by `../driver/build.sh`. Diagnostic build:

```sh
GOMAXPROCS=4 GOCACHE=/private/tmp/alan-series-scaling.yOkr1FA6/probe-build-cache \
  docs/evidence/series-scaling/instrument/build.sh \
  /private/tmp/alan-series-scaling.yOkr1FA6/baseline \
  /private/tmp/alan-runtime-health.18e1DyxD/modcache/github.com/mattn/go-sqlite3@v1.14.52 \
  docs/evidence/series-scaling/driver \
  /private/tmp/alan-series-scaling.yOkr1FA6/probe-baseline-v3
GOMAXPROCS=4 GOGC=100 GOMEMLIMIT=off \
  /private/tmp/alan-series-scaling.yOkr1FA6/probe-baseline-v3/driver.bin \
  -bound 8 -dir /private/tmp/alan-series-scaling.yOkr1FA6/probe-smoke8 \
  -sample probe-smoke8 -source-label baseline-instrumented
```

The output path and sample directory must be new for a rerun. The input ALAN
export is `af6943a5805a9a6c0e8fe77f804264f4765d07ec`; SQLite is pinned
`github.com/mattn/go-sqlite3 v1.14.52`. Input paths are copied before edits, including
the original amalgamation and its SQLite version markers. `generation.json`
records the generating scripts' hashes and input paths. Each of the three inputs
receives complete original, modified and post-gofmt manifests. The generated
`transformations.json` and `measurement.patch` retain original/modified hashes
and exact text changes. Generated additions are preserved in their source copies
and hashed in the manifests. `build.sha256` retains the executable and modfile
hashes. The build uses local module replacements, `GOWORK=off`, local toolchain,
`GOPROXY=off`, `GOSUMDB=off`, and `-mod=readonly`; it does not mutate the input
modfiles. The generator aborts on a missing or ambiguous expected signature or
text anchor. A changed source needs an explicit audit of those anchors.

`baseline-hashes.json`, `baseline-measurement.patch` and `smoke-summary.json`
preserve the first successful instrumented build and bounded N=1/N=8 sanity
checks. Both checks completed with `instrumented=true` and `invariants_ok=true`.
They used the same final driver source as the ordinary driver
(`ee03ad6650027b50d6dae6793d4dd78910b1395175fed09cfc08b0a708fcbe1e`).
No large instrumented campaign or overhead estimate is represented by these
smokes. Earlier scratch generation attempts failed closed on an ambiguous
marshal anchor and a read-only copied directory; neither produced a measured
binary. The successful generation is `probe-baseline-v3`.

## Function and byte counter definitions

All counters are cumulative, unsigned integer aggregates. The existing copied
driver hook samples before/after each phase and emits `counter_deltas`; the hook
also makes the report's `instrumented` flag true. No counter is reset between
phases. No function timing, goroutine labeling or per-SQL event log is added.
Snapshots can include concurrently finishing work within the driver's existing
phase boundaries; counters do not establish new causal boundaries.

| Key | Exact increment |
| --- | --- |
| `fn.<name>.calls` | Entry, including an entry that later returns an error. Names include descriptor, invocation enablement, root preparation/settlement, scope preparation, readiness, delivery enumeration, finishRun, root completion reads, ledger/ordinary/composition/protocol validation, profile reads/retention, InspectExecution, table scanner and Reduce. Owner and reader pendingDeliveries entries are separate. |
| `fn.compositionProfile.descriptor.input_bytes` | `len(p.raw)` immediately before that descriptor's JSON decode. Nil-profile errors contribute an entry but no bytes. |
| `fn.retainCompositionProfile.initial_marshal_bytes` | Length of the successfully marshaled input provision before size/semantic validation and canonical retention. |
| `work.selectReady.runs_considered` | One readiness run-loop iteration, including skipped retired/closed runs. |
| `work.selectReady.observations_considered` | One inner observation-loop iteration, before matching run or consumption state. |
| `work.readRootCompletions.roots` | One root loop iteration. |
| `work.readRootCompletions.root_intent_pairs` | One root × sorted-intent loop iteration, before membership resolution. |
| `materialized.<table>.rows` | One successfully scanned and validated row appended by scanInspectionTable. |
| `materialized.<table>.text_blob_bytes` | Sum of lengths of text/blob values in those rows. Excludes integer values, Go headers/maps, column names and SQLite storage overhead. |
| `reduce.selectReady`, `reduce.Commit`, `reduce.validateOrdinaryLedger` | Entry at each respective reducer call site, including a reducer error. The first category includes both no-observation and selected-observation paths. |

`fn.Reduce.calls` counts all executor Reduce entries. There is no function named
`normalReduce` in this frozen source. Ordinary readiness and Commit calls are
identified separately from validation replay calls. Replay, Open and raw Inspect
may all reach validateOrdinaryLedger; this counter therefore names the call site,
not exclusively the public Replay API. Public phase labels provide that second
dimension. Byte totals quantify parser input traffic, not CPU cost or allocation
size. Function entry counts do not by themselves prove that an optimization is
safe or that the function dominates time.

## SQL counter definitions and coverage

Every histogram key starts `sql.template.<normalized SQL>.`; the same metrics
are also summed under `sql.total.`. Metrics cover the copied driver's normal
SQLiteStmt query/exec/preparation paths and the no-argument Exec fast path.
Connection initialization directly calls C `sqlite3_exec` for PRAGMAs and is
**excluded**. Internal trigger programs and implicit engine actions contribute
to the measured parent statement's VM work rather than being separate Go-level
SQL executions. These totals must not be called all SQL executed by SQLite.

| Metric | Exact definition |
| --- | --- |
| `completed_prepares` | Successful explicit prepare producing a non-null statement. Empty statements and failed prepares are excluded. Cached reuse is not a prepare. Internal SQLite automatic reprepare is not included. The no-argument fast path records its successful prepare after its actual step, before returning to Go; a step error still counts the successful prepare. |
| `executions` | Sum of SQLite `SQLITE_STMTSTATUS_RUN` sampled and reset after actual steps: the number of times statement execution starts. A multi-row result is one execution. |
| `step_calls` | Number of actual C step operations observed, including SQLITE_DONE or an error. Reading an already buffered eager first-step result is not another step. |
| `sqlite_row_returns` | Actual observed SQLITE_ROW returns from those step operations, including an eager first step even if the caller never consumes that row, and Exec used with a row-producing statement. This is neither consumer-delivered rows nor rows examined. |
| `fullscan_steps` | Sum of SQLite `SQLITE_STMTSTATUS_FULLSCAN_STEP`, sampled and reset after actual steps: forward movements of table cursors during full table scans. This is not all visited rows, indexed rows examined, or the number of complete scans. |
| `vm_steps` | Sum of SQLite `SQLITE_STMTSTATUS_VM_STEP`, sampled and reset after actual steps: virtual-machine instructions executed. |

The C helper reads/resets all three status values in one extra C call per actual
step. The no-argument C fast path reads them before finalize. Both direct and
cancellable/eager paths are covered, and no status reading is added to the
cached pending-row consumption path. There is consequently no deliberate
query/exec wrapper double-counting and no first-row double-counting. Existing
statement-reset/cancellation/transaction behavior is retained. Failed steps may
contribute work even though they do not return a row.

Templates come from `sqlite3_sql` after prepare, or the consumed statement prefix
of a no-argument Exec. Whitespace is collapsed; single-quoted strings, blob
literals and numeric tokens become `?`; comments are removed. Bound values are
never expanded, queried or retained. Identifiers remain visible. The map retains
at most 256 templates plus one explicit `<template-cap-overflow>` bucket; overflow
aggregates work without retaining additional keys. N=1 and N=8 smokes observed
247 and 252 templates respectively. A merged overflow bucket cannot establish
which individual query dominated. These are aggregate template histograms,
not EXPLAIN output or samples of bound parameters.

Exact total rows examined, especially indexed cursor visits, are unavailable in
this build. Neither returned rows, full-scan steps nor VM instructions can be
substituted for that missing quantity. No SQLITE_ENABLE_STMT_SCANSTATUS build
option is introduced; changing the engine's compile options is outside this
probe's scope.

## Interpretation and overhead

All counter runs are diagnostic and excluded from headline timing. Function
counters add synchronization and allocations; SQL adds template normalization,
bounded mutex-protected aggregation and a C status call; snapshots allocate maps.
Table-byte counting adds a pass over scanned cell values. Counter snapshots
occur outside each phase's measured wall interval, but instrumentation inside
functions and SQL is inside that interval. The full counter process also retains
the counter maps, so its allocation and RSS values describe instrumented work.
The parent campaign separately compares instrumented/plain overhead at
representative bounds. Those measured ratios, rather than an assumed negligible
cost, must qualify the counts/profiles. No speedup claim uses this binary.

## Independent review finding and direct proof

The independent source review initially returned REJECT I1, alleging double
backslashes in the raw-string normalizer. That original rejection is preserved
in the campaign's `instrument-review.md`. A direct raw-source check found six
single backslashes and zero doubled backslashes in both payload and generated
v3. The reviewer acknowledged that JSON-escaped tool output had been mistaken
for literal source and withdrew I1 as a review error. No normalization source
fix or v4 binary exists; the original v3 source and binary remain unchanged.
This is a resolved review disagreement, not a failing implementation fixed after
RED.

Focused validation on actual v3 passed nine normalization cases (integer and
decimal literals, block and line comments, quoted strings with comment markers,
escaped single quotes, blobs, identifiers and placeholders) and actual copied
SQLite accounting. Two cached three-row queries produced exactly one explicit
prepare, two executions, eight step calls and six SQLITE_ROW returns, with
nonzero VM work and zero full-scan movement. An unconsumed cached eager row
added exactly one execution, one step and one SQLITE_ROW return; no-argument
Exec added exactly one explicit prepare, execution and step. A deliberately
doubled-escape synthetic mutant failed normalization, establishing fault
detection by those cases. The synthetic RED is not evidence of a defect in v3.
`proof-green.log` and `proof-mutant-red.log` retain actual output.

To reproduce these narrow checks in fresh scratch copies, outside campaign
measurement windows:

```sh
GOCACHE=/private/tmp/alan-series-scaling.yOkr1FA6/probe-build-cache \
  python3 docs/evidence/series-scaling/instrument/verify.py \
  --sqlite /private/tmp/alan-series-scaling.yOkr1FA6/probe-baseline-v3/sqlite \
  --out /private/tmp/alan-series-scaling.yOkr1FA6/probe-proof-rerun
```

The proof file remains `.go.txt` in the repository and cannot be discovered as
a production Go package. Initial verification temporarily installed it in the
generated v3 SQLite copy, then removed it; the generated source bytes and binary
were not changed. The reproduction helper uses a fresh copy from the outset.
These finite lexical cases do not claim a complete general-purpose SQL parser;
this normalizer is scoped to the frozen ALAN driver's statement vocabulary.

## Separate coarse wall-timing diagnostic

`timing.py` prepares an independent source/driver copy with **no SQL, descriptor,
root-loop, or row probes**, using the original pinned SQLite dependency. It does
not reuse the detailed counter overlay or change v3. Only `--build` invokes Go
compilation. Run it outside other measurement windows:

```sh
GOMAXPROCS=4 GOCACHE=/private/tmp/alan-series-scaling.yOkr1FA6/probe-build-cache \
  python3 docs/evidence/series-scaling/instrument/timing.py \
  --alan /private/tmp/alan-series-scaling.yOkr1FA6/baseline \
  --driver docs/evidence/series-scaling/driver \
  --sqlite /private/tmp/alan-runtime-health.18e1DyxD/modcache/github.com/mattn/go-sqlite3@v1.14.52 \
  --out /private/tmp/alan-series-scaling.yOkr1FA6/probe-timing-baseline --build
```

A fixed array holds two counters per static boundary. Each invocation captures
`time.Now` at entry and its deferred completion records `time.Since` plus one
completed call. Existing deferred cleanup executes before that completion.
Elapsed time ends before the aggregate mutex is acquired; one mutex acquisition
updates both counters. Snapshots copy the finite array to the existing driver's
`counterSnapshot` hook, so the resulting report is `instrumented=true` and uses
ordinary `counter_deltas`. Calls returning errors are included; in-progress calls
have not yet contributed. No nested timer is subtracted from another timer.

Keys are `coarse.<boundary>.calls` and
`coarse.<boundary>.inclusive_wall_ns`. Boundaries cover Store.Run, settleRoots,
selectReady, the pendingDeliveries reader, finishRun, readRootCompletions, ledger/
ordinary/composition/protocol validation, InspectExecution, AuthorizeIntent,
prepareAdapterWork, transportAdapter, accountAdapter, recordAdapterResult and
settleAdapter. The function names/signatures are asserted unambiguous before
injection. No individual PrepareRoot, prepareCompositionScope or descriptor
invocation receives a timer; those can occur once per root per runner turn.

These values are **inclusive elapsed wall time**, including scheduling delays,
SQLite work and lock waits inside the boundary. They are not CPU proportions or
exclusive costs. For example, finishRun includes readRootCompletions;
validateCompositionStore includes ordinary/protocol validation; Run includes
settleRoots, readiness, delivery work and final reporting. accountAdapter includes
recordAdapterResult and settleAdapter. transportAdapter runs on a worker and
can overlap the owner. Do not sum nested or concurrent values. Counter deltas
attribute completed spans to their completion phase; timer calls should remain
wholly inside the ordinary driver phases for this workload, but the diagnostics
do not establish a stronger causal cut.

The generator emits exact post-gofmt `timing.patch` and
`timing-transformations.json` (including added files), original/modified source
manifests, and `timing-generation.json` with boundary definitions, input paths,
generator/driver/original-SQLite hashes, and, after build, executable/modfile
hashes. Input trees and original SQLite are never edited. Timer diagnostics are
excluded from headline timing; closure, clock, aggregation and snapshot overhead
must be acknowledged. Coarse time attribution can distinguish root settlement
from final report construction but does not itself select a production change.
