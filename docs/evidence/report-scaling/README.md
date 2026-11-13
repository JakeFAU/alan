# Full report scaling evidence

The [contract](contract.md) and [implementation plan](plan.md) govern the work.
[Baseline results](baseline.md) describe the production source before report
batching.

## Reproduce the excluded baseline

Use Go 1.27.4 on darwin/arm64, /usr/bin/python3, the native macOS SDK and the
cached go-sqlite3 v1.14.52 module. The controller records the source archive,
per-file inventories, tool and binary hashes, module checks, command arguments,
environment, and actual exits. Use an otherwise idle host and a new raw output
directory under /private/tmp or the ignored report-scaling plan workspace.

```sh
/usr/bin/python3 -B docs/evidence/report-scaling/oracle.py --check
/usr/bin/python3 -B docs/evidence/report-scaling/measure.py \
  --root /private/tmp/alan-report-scaling-baseline
```

The baseline route exports and tests the pinned Git commit, compiles the frozen
oracle through a Go test overlay, builds the ordinary driver, runs one excluded
sample at each of 1, 8, 16, 32, 64, 128 and 256, then builds and runs a separate
B256 SQL diagnostic. It records failures in fresh directories. The optional
`--gocache` selects an existing local build cache. `--mode smoke` runs B1 and
B8 plus a B8 counter row. The `timing` and `counters` modes can run the
corresponding excluded rows on a specified clean `--revision`. The final
paired campaign has its own schedule.

The driver uses the existing public bounded-series program and protected
`Store.ContinuationBasis` approvals. It checks each complete `RunReport`
inside the execution window, including open-run and root ordering, original
identity, charges, outcomes and nil versus empty evidence fields. It then
verifies the independent typed evidence, completed reopen with a further
zero-progress Run, and Replay. Proofs and raw diagnostics remain under the
chosen raw root. The JSON proof digest covers prefix cuts, original root
membership, final typed evidence and physical contacts. Independent Create calls
produce different namespaces, so their proof digests are not cross-arm equality
tests. Use same-seed differential histories for exact cross-arm comparisons.

The `execute_window` captures the entire repeated Run/approval window,
including returned-cut verification and resource snapshots. Per-Run resource
sums isolate the time inside ordinary calls. CPU is process user plus system
time. RSS is the cumulative process high-water mark in bytes on this host.
Construction, Create, final inspection, reopen and Replay have separate
measurements.

The counter build scopes SQL issued through `finishRun`, with every other
statement recorded outside that scope. Counts are SQLite executions, SQLite row
returns, VM steps and full-scan steps. The counter binary never supplies timing
rows. It reuses the prior SQLite measurement code and generates a disposable
runner patch that counts actually returned roots. The full registry is retained
and checked against the compact stage counters. There is no CPU or allocation
profiling in this baseline route.

## Frozen report oracle

`oracle.py` regenerates `report_frozen_test.go.txt` from the pinned commit.
Install the fixture as `internal/executor/report_frozen_test.go` for integration
differentials. It freezes finishRun, pendingDeliveries, grouped root traversal
and charges, and root/invocation preparation through composition scope
preparation. The oracle and candidate share the authoritative leaf readers
such as `readRunIDs`, `loadRun`, `readIntents`, `compositionMembershipRoot`,
`compositionExternalKnown`, `loadCompositionClosure`, `loadCompositionAdmission`,
`compositionDescendantsPending`, the adapter/external attempt and result
readers, and the decode and correlation rules. A change to a shared validator
requires separate differential coverage or freezing its original implementation.
