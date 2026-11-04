# Reproduce the scaling experiment

Use fresh directories and serial processes. The measurement driver and
differential consumer are separate public-package modules. They are not shipped
as runtime packages, imported by ALAN, or used as execution authority.

## Source and environment

The baseline is `af6943a5805a9a6c0e8fe77f804264f4765d07ec`; the production
candidate is `e32d0e2f020638f301dcd0f0687c5c444180225c`. The two exported
trees, rather than the development working directory, supplied all comparable
builds. Copy the driver to a third directory as well, so its build does not
accidentally incorporate the development checkout's VCS metadata.

The recorded comparison uses Go 1.27.3, darwin/arm64, CGO 1, Apple clang21,
go-sqlite3 v1.14.52, `GOMAXPROCS=4`, `GOGC=100`, unlimited GOMEMLIMIT, and the local
filesystem of an Apple M4 with 16GiB RAM. Exact versions and build commands are
retained with the data. Go module caches are pinned and network module access
is disabled during execution. No live weather/model requests occur.

Illustrative preparation, from this repository (adjust cache paths for the
machine; do not overwrite the original evidence):

```sh
ALAN_REPO=$(pwd)
experiment=$(mktemp -d)
mkdir "$experiment/baseline" "$experiment/candidate" "$experiment/driver"
git archive af6943a5805a9a6c0e8fe77f804264f4765d07ec |
  tar -xf - -C "$experiment/baseline"
git archive e32d0e2f020638f301dcd0f0687c5c444180225c |
  tar -xf - -C "$experiment/candidate"
cp -R docs/evidence/series-scaling/driver/. "$experiment/driver/"
export GOWORK=off GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=1
"$experiment/driver/build.sh" "$experiment/baseline" "$experiment/baseline-driver"
"$experiment/driver/build.sh" "$experiment/candidate" "$experiment/candidate-driver"
```

The build helper changes only a temporary modfile; it adds no repository
replacement or dependency. Preserve the exact source filename/content manifests,
compiler/module versions, commands and executable SHA-256s before measuring.
Source labels passed to a binary are not themselves proof of its build origin.

## Diagnostic attribution

Run the ordinary driver once at each bound 1/8/16/32/64/128/256 with a new output
directory and explicit sample/source labels. Collect separate CPU/allocation
profiles at 128 and an execution trace at 16, as described in
[driver/README.md](driver/README.md). Do not turn allocation sampling to 1 for
headline or representative sampled profiling.

Generate aggregate-count and coarse-timer copies separately using
[instrument/README.md](instrument/README.md). Preserve the exact generator,
transformation patch, source/module manifests and binary hashes. Measure their
overhead against ordinary builds; do not subtract an assumed overhead from
results. `pprof` can inspect the retained raw CPU and allocation profiles, and
`go tool trace` the trace. On the original installation the two tools were built
offline from its bundled Go source because the packaged `go tool` entries were
absent; their exact build commands are in the verification records.

## Frozen ordinary paired campaign

The committed paired-v2 validator intentionally binds this experiment's exact
Go/platform configuration. A different host/toolchain needs a separately
declared campaign; do not edit an old schedule after seeing outcomes.

```sh
python3 docs/evidence/series-scaling/make_schedule.py \
  "$experiment/baseline-driver" "$experiment/candidate-driver" \
  af6943a5805a9a6c0e8fe77f804264f4765d07ec \
  e32d0e2f020638f301dcd0f0687c5c444180225c \
  "$experiment/schedule.json"
cd "$experiment"
python3 "$ALAN_REPO/docs/evidence/series-scaling/run_schedule.py" \
  "$experiment/schedule.json" "$experiment/paired" > "$experiment/runner.log" 2>&1
python3 "$ALAN_REPO/docs/evidence/series-scaling/analyze_pairs.py" \
  "$experiment/schedule.json" "$experiment/paired" "$experiment/analysis.json"
```

`ALAN_REPO` above records the repository's absolute path before changing directory.
The schedule contains14 excluded warmups and 84 measured processes: six adjacent
pairs at each bound. It fixes the AB/BA order, randomized bound order, executable
hashes, four operating-tool hashes and runtime/environment fields before running.
The runner refuses existing samples and checks executable hashes before and
after each process. Analysis requires all 98 rows, complete expected phases,
raw report/progress hashes and exact source/configuration identities. It does
not substitute zero for a missing phase or accept a partial campaign.

Do not run builds, tests, profilers, compression or competing experiments during
timing. OS caches, host load and thermal state are not controlled; load and
wall/CPU/RSS are recorded per process. Every valid sample remains in analysis.
The full Execute interval includes measurement/logging bookkeeping, rather than
silently summing only selected body timers. Body phases overlap that interval.

The separate declared interval-reopen diagnostics use `-reopen-every 1` at 8 and
`-reopen-every 8` at 32, one fresh process per source/configuration. They are
descriptive single samples and never enter the six-pair ordinary medians. Every
ordinary sample already measures completed Close/Open/Run and Replay separately,
including at 256.

## Semantic and recovery qualification

Follow [differential/README.md](differential/README.md) to run four same-seed
execution pairs and reader pairs on the real archived main/WAL cuts. Build both
external consumers from one frozen consumer copy. Preserve the seed, all copied
stores, complete reports, comparator outputs, exact commands and hashes.

The original process fixtures come from the unchanged real weather-series suite:

```sh
ALAN_SERIES_EVIDENCE_DIR="$experiment/crash-archives" \
  go test -json -count=1 -timeout=15m -run '^TestWeatherSeriesProcess$' \
  ./examples/weather-agent > "$experiment/process.log"
```

Run from each exported ALAN source, outside timing. It uses acknowledged process
barriers and actual SIGKILL/reap. Copy main plus retained WAL before using SQLite
on disposable clones; never open the immutable original archive as an owner.
The generic reader does not Run or contact an adapter. Full process tests and
the existing `tools/compare_stores.py` supply actual resume execution coverage.

The previous external eight-operation consumer is preserved at
`docs/evidence/bounded-series/performance/module`; use its unchanged tests,
including the ordinary/race series tags and historical fixture inputs. The
recorded qualification includes genuine format 11→12 conflict promotion and a
baseline reader of both fixtures, not edited SQLite version markers.

Run the [verification commands](qualification.md) from an immutable delivery
export and match filename/content manifests before and after. Nested driver,
differential and Python tooling tests need their separate commands; `go test
./...` at the repository root does not discover nested Go modules. Preserve
failure logs and original negative findings alongside corrections. All exact
commands used for this delivery are retained with the raw evidence.
