# Independent bounded-series performance and compatibility evidence

This package contains an unrelated public consumer module, a controlled internal
writer test overlay, source/command manifests and raw source-scoped results. It
contains no ALAN source export, compiled binary, cache or database duplicate.
The deterministic adapter runs in process; there is no HTTP/socket implementation.

Read `manifests/sources.json` and `manifests/commands.json` before attributing any
result to a source. Candidate and corrected-task1 results are preserved under their
actual source identities. Final producer results are retained separately as `raw/final-*`. Source
manifests identify the exact immutable exports; original benchmark logs identify
CPU, GOOS/GOARCH, allocation counts and sample count.

## Reproduction

Go 1.27.3 and a C toolchain for github.com/mattn/go-sqlite3 v1.14.52 are required.
Use an existing populated GOMODCACHE/GOCACHE for offline execution. The module has
no machine-specific replacement. In a **copy** of `module/`, bind a verified source:

```
go mod edit -replace=github.com/JakeFAU/alan=/absolute/path/to/immutable/export
go test -mod=mod -tags=series -run='^TestCanonicalLoweringEquality$' -v
```

`reproduce.sh` makes its own module copy in a fresh output directory and records
Go/toolchain/OS context. Its final mode also builds the generic reader from the
verified base source and produces a genuine conflicting-receipt format 12 store via the
existing retained-result writer. All test databases are disposable output files.
Neither source export is modified. Run benchmarks serially on a quiet machine:

```
./reproduce.sh baseline /path/to/base-export /path/to/new-baseline-output
./reproduce.sh final /path/to/final-producer-export /path/to/new-final-output /path/to/base-export
```

Each full 256-position stateful execution uses exactly one benchmark operation and can take
about 6 minutes. There are 256 in-process adapter entries, never 256 live requests.
Create, execute (including per-successor basis inspection and approval), final
snapshot inspection, Replay and author-view work are measured separately.
Construction and view use five 100 ms samples; full execution uses one empirical
sample. Cumulative allocated bytes are **allocation traffic**, not peak memory.
No statistical latency improvement is claimed from these shared-host samples.

`BenchmarkManualConstruction` is the two-cached-artifact manual comparator;
`BenchmarkManualLoopConstruction` mirrors the historical compile-per-position
pattern on the controlled definition. The helper additionally validates and owns
its declaration and returns an independent Provision copy. The separate canonical
manual expansion equality test proves complete Provision JSON equality using the
same generated names. Do not attribute shorter historical approval-name payload
costs to new runtime overhead.

The overlay imports an internal package only to call the normal retained-result
writer and submit a valid losing candidate to an actual paid operation. It does
not manufacture SQL history or edit a storage version. All other module source
uses only public ALAN packages. The old reader intentionally uses only generic
runtime APIs; this does not promise an old weather CLI accepts unfamiliar programs.

## Suggested committed copy

Copy this directory as one evidence package after final logs/source manifests and
report are sealed. Include `README.md`, `reproduce.sh`, `module/`, `overlay/`,
`raw/`, `manifests/`, `REPORT.md` and `REVIEW.md`. Exclude scratch databases,
binaries, source exports and caches. The controller owns final source confirmation
and the actual commit.

The overlay payload is named `overlay/promote_series_test.go.txt` so root
`go test ./...` cannot discover it as a standalone package. Reproduction maps
those unchanged Go-source bytes to a virtual `_test.go` file inside the verified
base executor package. This packaging suffix does not change measured writer code.
