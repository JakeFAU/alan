# Reproduce the release performance comparisons

These comparisons measure the public runtime before and after the v0.1.0 changes on the same host. The baseline is `64a692b6f82ae10353f50c8741377166bc6cf854`. The candidate is `bea0acc86d9a7302ccc3511327455922503c4a57`.

## Set up the source and tools

Start in the directory containing this README and both Python scripts. Use a full ALAN Git clone. The bounded-series driver reads its original measuring code and workload from the baseline's `docs/evidence` directory. Recorded runs used Go 1.27.4, Darwin/arm64, Apple clang 21, CGO, and `github.com/mattn/go-sqlite3` v1.14.52. Use Python 3, Git, tar, a writable local filesystem with working SQLite WAL and file locks, and sufficient space for the per-sample databases. The bounded-series preparation records the macOS SDK using `xcrun`, which requires the Xcode command-line tools.

```sh
benchmark_tools=$(pwd -P)
scratch=$(mktemp -d)
repo="$scratch/repo"
base="$scratch/base"
candidate="$scratch/candidate"
base_sha=64a692b6f82ae10353f50c8741377166bc6cf854
candidate_sha=bea0acc86d9a7302ccc3511327455922503c4a57

git clone --no-checkout https://github.com/JakeFAU/alan.git "$repo"
git -C "$repo" worktree add --detach "$base" "$base_sha"
git -C "$repo" worktree add --detach "$candidate" "$candidate_sha"

export CGO_ENABLED=1 GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOENV=off
export GOCACHE="$scratch/go-build" GOMODCACHE="$scratch/go-mod"
go -C "$base" mod download
go -C "$candidate" mod download
go -C "$base" mod verify
go -C "$candidate" mod verify
export GOPROXY=off
unset ALAN_RECOVERY_BENCH_STORE
```

Use the same Go executable, C compiler, SDK, power settings and local volume for both revisions. Run the comparisons sequentially while the host is otherwise idle. Keep each output directory and use a fresh one for another trial.

## Time store creation and causal recovery

```sh
python3 -B "$benchmark_tools/compare_creation_recovery.py" \
  --baseline "$base" --candidate "$candidate" \
  --fixture "$benchmark_tools/creation_test.go.txt" \
  --output "$scratch/creation-recovery.json"
```

The script overlays the same Go benchmark onto each checkout. It times Create and Close at a fresh database path for a one-step program with either zero asset bytes or one 128 KiB retained asset. Both revisions receive the same public provision. The asset-bearing case exercises the candidate's additional scoped-format metadata and validation even though this agent declares no effect. Compilation, provisioning the input values and constructing the asset precede the timer. The nine existing recovery cases measure InspectExecution, Replay and Open against each of three retained histories, covering a missing target, a present target and a forged history. Each Open uses a new private copy. Making that copy and closing the returned owner take place outside the Open timer. The forged case must return `ErrCorrupt`.

Each of three blocks runs both revisions, ordered base/candidate, candidate/base, then base/candidate. Each invocation uses `-cpu=1 -benchtime=5x -count=1 -benchmem`. The JSON records all 66 benchmark rows and the per-revision median of three observations for each operation in ns/op, B/op and allocs/op. Compare matching case names and units. B/op measures Go allocations and excludes memory allocated inside SQLite's C code.

## Time a bounded series of 256 observations

```sh
series="$scratch/bounded-series"
python3 -B "$benchmark_tools/compare_bounded_series.py" prepare \
  --baseline "$base" --candidate "$candidate" --out "$series"
python3 -B "$benchmark_tools/compare_bounded_series.py" attest \
  --baseline "$base" --candidate "$candidate" --out "$series" \
  --revision "$candidate_sha"
python3 -B "$benchmark_tools/compare_bounded_series.py" time \
  --baseline "$base" --candidate "$candidate" --out "$series"
```

Preparation snapshots the local build inputs, checks the baseline against Git, and builds four drivers from the baseline harness. Each revision gets a full-report driver using Run and a progress driver using RunProgress. The driver modules use private local replacements pointing to the measured source snapshots. Attestation independently enumerates build inputs from the committed candidate and checks them against the prepared source. `build/manifest.json` records the inputs, binary hashes and toolchain. `commit.json` records the attested candidate revision, tree and source manifest hash.

Each fresh driver process runs the same ordinary schedule and direct adapter workload with `GOMAXPROCS=4`, `GOGC=100` and `GOMEMLIMIT=off`. Validation requires 256 contacts, selections, completions, spent attempts and replayed intents, 255 protected approvals, and 1,023 reductions and replayed history rows. The harness also validates stage order, the final inspection and the reopened computation. It checks each binary's digest before every sample. One warm-up per arm precedes four measured blocks. The order balances each arm across the four positions and across immediate predecessors.

`samples/` holds each store, raw report, process log and validated checks. `rows.json` keeps the 16 measured samples. `summary.json` reports wall-time medians and ranges in milliseconds for provision construction, creation, the execution window, reopening, replay and the full lifecycle. Compare candidate-full with base-full and candidate-progress with base-progress. For each mode, the paired percentage is computed within each block as `100 × (candidate/base − 1)`, then summarized by its median. A positive value means the candidate took longer. The lifecycle includes work outside the execution window. The measured cases establish costs for this finite workload, host and toolchain. Different host loads, filesystems or corpus sizes can produce different timings.
