# Excluded P6 baseline

The full arm built and ran from the archived merged P5 source
`e43e4f6be4500d177b50caf29181608d09b99c62` (tree
`6fbde0ba908a80fc45733c4daf69e473e3cff47a`). The ordinary suite had
1,410 test passes and 18 package passes. Four optional historical reader
subtests skipped because their external reader binaries were not supplied.
[Baseline metadata](baseline.json) binds the raw logs, exports, build inputs,
binaries, per-row command outcomes, and profiles. The raw artifacts are
retained locally in the ignored P6 plan workspace; they are not public
downloads.

The common driver uses the P2 deterministic direct meter and P2 final identity,
result, selection, approval and charge proof. Each Execute position uses the
owner's targeted continuation basis (after position one), then checks only
the expected progress tuple and number of adapter contacts. The three build
variants differ at the typed invocation helper. Baseline full and candidate
full project the tuple from ordinary Run; candidate progress projects it
from RunProgress. Each arm separately calls full Run at the completed cut,
compares its roots with typed inspection, then closes and reopens the store.
A full Run on the reopened store must again return zero work, quiescence, and
no extra contact. A fresh typed cut and independent Replay must preserve the
original roots. The complete driver lifecycle includes final validation.
Execute and lifecycle resource snapshots record elapsed time, CPU and
allocation deltas; per-operation timings inside Execute use only wall clocks.
The child exit RSS is captured after reaping the driver and is distinct from
the Execute-end cumulative high-water reading.

Run from the repository root (the output directory must be new):

```sh
python3 -B -m unittest discover -s docs/evidence/progress-scaling -p 'test_measure.py'
baseline=e43e4f6be4500d177b50caf29181608d09b99c62
for lane in smoke profile counters; do
  python3 -B docs/evidence/progress-scaling/measure.py \
    --revision "$baseline" --mode full --lane "$lane" \
    --out ".superpowers/sdd/plan-progress-scaling/reproduce-$lane"
done
```

The Go module cache must already contain SQLite v1.14.52; dependency
downloads are disabled. The default build cache is under each new raw root.
A writable existing cache may be supplied with `--gocache`. The B1/B8 smoke
and B256 CPU/allocation profile are excluded. The independently built
counter lane uses the existing P2 SQLite probes and P5 `finishRun` scope
to verify the complete SQL partition. At B256 it measured 397,884 Execute SQL
executions, 233,344 under `finishRun`, with 65,536 root records emitted
during Execute. The B256 profile and counter lanes each verified 256
completed, selected and spent originals, 256 adapter contacts, 255 protected
approvals and 1,023 reductions, followed by matching Replay evidence.

For the later primary comparison, give `--lane timing` the exact candidate
revision and omit `--mode`. It builds all three arms before measurement,
then runs one excluded warmup per arm/bound and six adjacent blocks per bound,
each in a different arm order. Keep other builds/tests off the measurement
host during that run. These excluded baseline rows support no speedup
estimate and are not compared to the previous P5 apparatus.
