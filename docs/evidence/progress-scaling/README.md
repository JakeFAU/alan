# Execution progress evidence

[Contract](contract.md) · [Implementation plan](plan.md) ·
[Excluded baseline](baseline.md) · [Measured results](results.md) ·
[Qualification](qualification.md)

The measured candidate is commit `f8ecf081867b29012bbf1688eddfa067abbc1e43`,
tree `8fcc74b9d3e94230459d402c9492a5f9c96d50de`. Use a clean checkout at this
revision for the recorded measurement. The corrected reader checker is
`fe5baf86ca285fc2496e2ed679c00d35bf119924`. Keep the integrated baseline
and historical reader commits available in the local Git object database.

## Reproduce

The campaign ran on an Apple M4 Mac with Go 1.27.4, native macOS 27.2 SDK,
Staticcheck 2026.2.2, Python 3.9.6 and cached go-sqlite3 v1.14.52. It resolves
Go modules offline and uses CGO, `GOMAXPROCS=4` and `GOGC=100`. The
qualification needs loopback for its controlled HTTP endpoint and a host
that supports the Go race detector and Unix process signals. Run from the
checkout root, with fresh output directories and no competing builds or
tests during primary timing.

```sh
revision=f8ecf081867b29012bbf1688eddfa067abbc1e43
workspace=.superpowers/sdd/plan-progress-scaling
mkdir -p "$workspace"

python3 -B docs/evidence/progress-scaling/measure.py \
  --revision "$revision" --lane timing --out "$workspace/reproduce-timing"
python3 -B docs/evidence/progress-scaling/summarize.py \
  "$workspace/reproduce-timing" "$workspace/reproduce-timing-summary.json"

for mode in full progress; do
  for lane in counters profile; do
    python3 -B docs/evidence/progress-scaling/measure.py \
      --revision "$revision" --mode "$mode" --lane "$lane" \
      --out "$workspace/reproduce-$lane-$mode"
  done
done
```

Run the qualification from a clean checkout of the corrected revision,
with a new output directory and outside path. The original two-stage
run retained its failed comparison and separate reader recheck. The
exact commands for that run are recorded in the custody data.

```sh
revision=fe5baf86ca285fc2496e2ed679c00d35bf119924
workspace=.superpowers/sdd/plan-progress-scaling
mkdir -p "$workspace"
python3 -B docs/evidence/progress-scaling/qualify.py \
  --revision "$revision" --out "$workspace/reproduce-qualification" \
  --outside /private/tmp/alan-p6-reproduce-b8
```

The optional `--gocache` selects a writable existing build cache. The
profile lane saves CPU and sampled allocation profiles. For the installed
Go distribution, `go tool pprof` was unavailable. The baseline profile
rendering built the bundled `cmd/pprof` from the selected Go distribution
and retained its input hashes. If your Go distribution includes the tool,
`go tool pprof -top BINARY PROFILE` can render those profiles.

The campaign records commands, actual exits, environments, source and
tool identities, and before/after input inventories. The raw stores,
profiles, executables and detailed logs remain in the ignored local
workspace. The tracked records summarize and hash those artifacts.
