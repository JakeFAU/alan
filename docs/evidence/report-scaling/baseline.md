# Excluded full-report baseline

Production commit `5f5d382c8d1e9c7369e7624e3a962746c9bf2229`, tree
`cb6dc03624010d13f8af8d045af44e490110bfaa`. Measurements use its Git archive.
At the start the live checkout was at `ce731c59d99e3dc77af296ab021f652f02d234fb`
with only the three untracked, hash-bound apparatus and fixture files shown in
the [baseline record](baseline.json). The production source and dependency
inventory remained fixed. The host was an Apple M4 with ten logical cores,
16 GiB memory, macOS 27.2 (26B123), Go 1.27.4 darwin/arm64, Go scheduler
width 4, Python 3.9.6, Apple Clang 21.0.0 and macOS SDK 27.2. The only
dependency was go-sqlite3 v1.14.52. `go mod verify` passed.

## Excluded samples

Each row is one fresh process with a separate, freshly provisioned Store.
The ordinary rows have profiling and SQL probes disabled. These samples
establish feasibility and baseline shape. The final comparison requires
its own balanced, repeated pairs.

| B | Execute wall ms | Execute CPU ms | Allocated MiB | Allocations | Peak RSS MiB | Sum of Run wall ms | Final inspection ms | Close/reopen/Run ms | Replay ms |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 5.134 | 6.844 | 1.61 | 22,632 | 20.86 | 5.018 | 1.584 | 5.483 | 3.701 |
| 8 | 48.498 | 66.500 | 22.25 | 309,661 | 24.39 | 40.404 | 4.124 | 24.725 | 17.524 |
| 16 | 106.517 | 148.276 | 51.46 | 721,642 | 25.45 | 90.050 | 7.903 | 48.935 | 35.800 |
| 32 | 247.694 | 353.209 | 127.97 | 1,812,687 | 26.64 | 213.340 | 14.662 | 97.431 | 71.219 |
| 64 | 655.741 | 960.368 | 353.36 | 5,065,121 | 28.00 | 586.083 | 27.563 | 189.260 | 143.236 |
| 128 | 2,015.828 | 3,069.475 | 1,094.63 | 15,861,116 | 28.98 | 1,857.370 | 54.025 | 368.815 | 280.980 |
| 256 | 6,848.429 | 10,601.304 | 3,725.31 | 54,485,909 | 33.81 | 6,478.293 | 108.454 | 744.669 | 572.856 |

Execute covers the original ordinary Run calls, protected basis/approval,
and returned-cut proof. Peak RSS is the process high-water mark at the end
of that window. The sums and final stages have separate measurements.
All rows proved B physical contacts, B original operations selected, spent
and completed, B−1 protected approvals, and 4B−1 committed reductions. Reopen
and Replay preserved the final original roots and did not repeat contacts.

## B256 SQL diagnostic

| Scope | Executions | SQLite row returns | VM steps | Fullscan steps |
|---|---:|---:|---:|---:|
| Execute window | 790,268 | 1,018,144 | 16,221,871 | 587,010 |
| finishRun | 625,792 | 886,912 | 13,482,496 | 359,040 |

The report emitted 65,536 root records in 256 calls inside the window.
The full registry partition passed, with zero family/cell overflow or invalid
scopes. These five statement families account for 392,960 report executions.

| Report SQL family | Executions |
|---|---:|
| Root descriptor by ID | 131,072 |
| Root publication by scope | 65,536 |
| Freeze by scope | 32,640 |
| Closure by scope | 32,640 |
| Run state and subject revision by RunID | 131,072 |

The [prior diagnostic](../execution-scaling/results.md) reported the same
finishRun totals, including rows, VM steps and fullscan steps. This window
records 48 extra executions and rows and 264 extra VM steps. All of that
difference is in the ordinary Run stages outside finishRun. The approval and
basis stages agree, and the prefix residual is zero. The resumed report
verification follows the window. The scheduler can perform different numbers
of polling turns before receiving the asynchronous adapter result. The prior
compact artifact does not retain all non-report SQL family counts needed to
attribute the 48 additional executions to individual statements.

On this profile, two batch statements per call would replace the 392,960
point statements with 512 executions. This is an accounting model for a
candidate, not a measured gain. Root output is still quadratic across the
series. Rows, VM steps, decoding work and elapsed time need separate evidence.

## Report validation boundaries

The frozen traversal first builds root results, decodes the run projections,
validates retained intents, classifies unresolved and unknown operations,
computes pending deliveries, and decodes admission results. It may return an
error together with partial report fields. Root results are visited in
sorted order and assigned only on success. A published root short circuits
freeze and closure reads. An unpublished root with a retained freeze or
closure uses their original canonical validation, precedence and descendant
checks. Missing subjects and projections retain their original access
boundaries. Known adapter or external outcomes require the original validated
intent and result/receipt path. In particular, the global intent scan cannot
replace the canonical encoding and request-authority checks in a point intent
load. This baseline supports testing report-local batching of raw root
descriptors, publication records, freeze/closure presence and raw run state
under the existing ownership cut. Existing charge, status, settlement and
receipt reads remain part of the report.

## Verification

The ordinary suite recorded 1,408 test pass events, 18 passing packages and
zero failures. Four historical-reader subtests skipped because their optional
frozen executable fixtures were unset.

- `TestRequestAuthorityFormatsConflictPromotionAndOldReader/13/frozen_old_reader`
  and `TestRequestAuthorityFormatsConflictPromotionAndOldReader/14/frozen_old_reader`
  require `ALAN_PHASE3B3_OLD_READER`.
- `TestRequestBindingConflictAndOldReader/false/frozen_reader` and
  `TestRequestBindingConflictAndOldReader/true/frozen_reader` require
  `ALAN_PHASE3B4_OLD_READER`.

The integration-tagged frozen oracle compiled via a Go overlay, and regeneration
matched the tracked fixture. The separate B1/B8 smoke validated two uninstrumented
rows and a counter row. The [machine-readable record](baseline.json) carries
the exact source identity, reproducibility inputs, validated row resources,
diagnostic family totals and command/output hashes. The raw trees keep the
actual driver proofs, manifests, commands, statuses and logs locally.
