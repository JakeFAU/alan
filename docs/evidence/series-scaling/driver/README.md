# Public bounded-series scaling driver

This nested external Go module uses `author.BoundedSeries`, `Series.Provision`,
`runtime.Create`, fresh `InspectExecution`/`ContinuationBasis`,
`Store.ApproveContinuation`, and ordinary `Store.Run`. It performs one fresh
sample per process against one fresh SQLite store; there is no trusted
application shortcut or retry loop. It adds no dependencies beyond ALAN's
existing pinned SQLite driver.

## Build and run

`build.sh ALAN_SOURCE OUTPUT_BINARY` builds against any selected ALAN source
checkout. It creates a temporary modfile containing the local `replace` and
removes that modfile on exit. No machine-specific replacement is committed.
Use the same toolchain, cache settings, `GOMAXPROCS`, source, and workload for all
samples. Example (replace the source and output paths):

```sh
export GOWORK=off GOTOOLCHAIN=local GOPROXY=off
export GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache
export GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache
./build.sh "$ALAN_SOURCE" /tmp/series-scaling
GOMAXPROCS=4 /tmp/series-scaling -bound 8 -sample baseline-01 \
  -source-label "$SOURCE_COMMIT_AND_TREE" -dir /tmp/series-sample-baseline-01 \
  > /tmp/series-sample-baseline-01.json \
  2> /tmp/series-sample-baseline-01.progress
```

Bounds are 1, 8, 16, 32, 64, 128, and 256. Run samples serially, each in a new
process and directory. Existing stores are rejected. `-timeout` sets the sample
deadline (default 30m); errors stop the sample and still emit partial stage JSON.
No sample is automatically retried. `source-label` is a caller-supplied label,
not verification of source provenance; the campaign must retain actual source
commit/tree/file hashes and exact build commands separately.

The ordinary schedule calls `Run` once for each position, with a fresh
inspection/basis and protected approval before every position after the first.
`-reopen-every K` is a **separate recovery schedule** that closes, opens, and
resumes after each Kth selected position before the final one. For example,
`-reopen-every 1` exercises every eligible intermediate position. No interval
schedule is silently included in ordinary timing. Every schedule additionally
closes/reopens/resumes the completed series, rechecks original evidence, closes
ownership, and calls public Replay.

## Measurements

Each named phase records monotonic wall nanoseconds, process user/system CPU
nanoseconds from `getrusage(RUSAGE_SELF)`, and deltas of Go `TotalAlloc` and
`Mallocs`. No forced GC is performed. Allocation counters cover all goroutines
in the process; non-Go/SQLite allocations are not Go TotalAlloc. Measurement
calls and stage bookkeeping introduce overhead. The wall interval encloses the
public phase body only. CPU snapshots bracket that wall interval; Go memory
snapshots surround CPU snapshots. Very short phases should not be interpreted
beyond that resolution/overhead.

`process_maxrss_cumulative` is the **process lifetime high-water mark**, not a
phase-local peak or current RSS. `maxrss_unit` records bytes on Darwin, KiB on
Linux, and platform-dependent elsewhere. The last recorded high-water mark is
not guaranteed to cover profile serialization or final JSON serialization.

Phases are:

- `execute_window`: primary whole ordinary execution interval, from before the
  first Run through the final Run and per-position contact check. It encloses all
  N Runs and N-1 fresh snapshot/basis/approval operations, including stage
  snapshots, progress logging, and inter-phase bookkeeping. It excludes
  construction, Create, final readers, completed recovery, and Replay.
- `execution_window_including_reopen`: the separately named whole interval for
  an interval recovery schedule; it additionally includes intermediate resumes.

- `construct_provision`: BoundedSeries plus Provision.
- `create`: public runtime Create.
- `approval_snapshot_basis`: fresh public InspectExecution plus ContinuationBasis.
- `approval_establishment`: public ApproveContinuation.
- `run`: one ordinary Run per position.
- `final_inspection`, `final_series_view`: separate completed-cut readers.
- `close_reopen_resume`: completed Close/Open/Run, with no extra adapter contact.
- `resume_inspection_view`: detached public cut and series view after resume.
- `close`, `replay`: final owner close and read-only Replay.

The `execute_totals` object (`name=execute_sum`) remains the sum of the
individual `run`, `approval_snapshot_basis`, and `approval_establishment`
measurements. It excludes logging and inter-phase bookkeeping, so it is a
diagnostic decomposition, not the primary end-to-end execution elapsed time.
The whole `execute_window` is the primary ordinary execution measurement.
Construction, Create, final readers, and completed recovery remain separately
reported. Interval recovery has its own `interval_close_reopen_resume` phases
and whole `execution_window_including_reopen`. Do not sum nested whole-window
and per-phase records together.

Use a separate campaign/process for `-cpu-profile FILE`, `-alloc-profile FILE`,
and/or `-trace FILE`; profiled results are marked `profiled=true`. Allocation
profiling writes the `allocs` profile. `-mem-profile-rate BYTES` defaults to the
runtime sampling rate (normally 524288 bytes), recorded as `mem_profile_rate`
in every JSON report. Only an explicit `-mem-profile-rate 1` requests sampling
every allocation; use that expensive setting solely for separate diagnostic
runs. Sampling rate changes allocation profiling overhead and granularity. The
profile records cumulative allocation samples subject to runtime profile-update
lag. It does not claim phase-specific retained heap. Profiling covers all
workload stages, including final verification and readers; CPU stage labels
identify each measured public phase. Final proof checks outside those phases
remain unlabeled. Never mix profiled
measurements with headline timing samples.

## Identity and causality checks

The deterministic in-process adapter makes one local contact per operation and
returns the same semantic sample at every position. It records each actual
Execute entry's original request ID/key, prepared digest, protocol, configured
endpoint, and distinct response ID. These are **local adapter contacts**, with
no HTTP or remote finality claim. Prepare is pure and encodes the same wire
payload for the same-action continuation contract; response provenance carries
identity separately.

The driver fails unless all N original generation-one subjects have distinct
operation IDs, request keys, response-result digests, selected observation IDs,
and roots, exactly N adapter entries/results/selections/completions/one-effect
root charges, and N-1 protected approvals. Every contact must match its retained
original prepared request and original candidate. Every selected observation
must identify that original operation and its actual selected reduction.
Each protected approval must reference the prior position's exact original
operation/prepared/result digests, be admitted with reason `operator`, and be
selected in the successor before its distinct effect commit. Equality of
sample payloads is never used as identity evidence.

The same identity/causality checks run after completed recovery; the resulting
original evidence must remain equal, and contact count must remain N. Replay
must independently validate the store and return the same namespace with N
successful runs and N roots. This is consistency evidence, not an independent
policy proof or guarantee about an uncontrolled adapter.

## Optional instrumentation integration

Normal builds contain no internal counter API. A measurement-only generated
copy of this driver may include a small `package main` hook assigning
`counterSnapshot func() map[string]uint64` to a generated instrumented ALAN
snapshot function. The driver then marks `instrumented=true` and records
per-phase counter deltas. Counter snapshots for a measured body are outside
that body's timer; nested per-phase snapshots are included in the whole window
as inter-phase overhead in instrumented diagnostics. Ordinary headline builds
have no hook. Counter hooks belong in scratch copies, not production
source. Counter labels/semantics, original and generated source hashes, exact
transformation, SQL statement/row/VM/full-scan definitions, and overhead must
be retained by the instrumentation campaign. Returned rows do not imply rows
examined. Nested validator counters can overlap; they must not be summed as
independent work. Instrumented runs are separate from headline timing.
