# Reproducing the integrated performance comparison

Use a clean local macOS/arm64 filesystem with SQLite locks, Git, Go 1.27.4, CGO, Apple developer tools and Python 3. Follow the historical procedure in the candidate's `docs/qualification/v0.1.0/benchmarks/README.md` using detached checkouts of the public baseline `64a692b6f82ae10353f50c8741377166bc6cf854` and integrated release `57c230f8d1ac1a0bac4cc3b36f7b8e6b46775530`. Verify each module through `go mod download` and `go mod verify` before offline builds. Record the host load and run the two timing jobs serially. The qualification used a shared Apple M4 and excluded concurrent release tests from the timed period.

The baseline root selects go-sqlite3 v1.14.52. The integrated root selects v1.14.53. The historical driver module also listed an indirect dependency. Apply the included patch to a temporary copy of the historical measuring script so the driver uses the exact binding version and checksums of each measured source. The patch changes only setup in the disposable driver modules. The Go driver code, timed windows, ordering and outcome assertions are identical across like-for-like arms.

Set `base`, `candidate`, `evidence` and `scratch` to absolute paths. The evidence directory contains this file and `driver-binding.patch`. Use a fresh scratch directory. Both dependency versions must already be available in the module cache.

```sh
export GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=1 GOENV=off GOPROXY=off GOFLAGS=
unset ALAN_RECOVERY_BENCH_STORE GOGC GOMAXPROCS GOMEMLIMIT
benchmark_tools="$candidate/docs/qualification/v0.1.0/benchmarks"
cp "$benchmark_tools/compare_bounded_series.py" "$scratch/compare_bounded_series.py"
(cd "$scratch" && patch -p1 < "$evidence/driver-binding.patch")
python3 -B "$scratch/compare_bounded_series.py" prepare --baseline "$base" --candidate "$candidate" --out "$scratch/series"
python3 -B "$scratch/compare_bounded_series.py" attest --baseline "$base" --candidate "$candidate" --out "$scratch/series" --revision 57c230f8d1ac1a0bac4cc3b36f7b8e6b46775530
python3 -B "$scratch/compare_bounded_series.py" time --baseline "$base" --candidate "$candidate" --out "$scratch/series"
python3 -B "$benchmark_tools/compare_creation_recovery.py" --baseline "$base" --candidate "$candidate" --fixture "$benchmark_tools/creation_test.go.txt" --output "$scratch/creation-recovery.json"
```

Inspect `go version -m` on each driver and match the SQLite version and h1 sum to the measured root. The build receipt identifies the compiler, environment, source inventories and driver hashes. Each bounded-series workload is a fresh process and exercises 256 selections, completions, contacts and spent attempts, 255 approvals and 1,023 history entries. It verifies reopening and offline replay. Four excluded warmups precede four balanced timing blocks. The paired percentages compare corresponding samples within each block. The microbenchmark uses three blocks, both source revisions, eleven cases, five iterations and one logical processor per benchmark. Results are specific to these workloads and host conditions.
