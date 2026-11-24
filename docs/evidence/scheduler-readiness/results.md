# Fallback observation scans and guarded candidate copies

On November 24, 2026, the fallback selector validated 64 retained observation
rows, visited 64 candidate runs and checked 4,096 observation IDs to select
one ready receiver. On a different workload with 64 ready receivers, the
guarded selector visited one candidate per receive and materialized 64 ordered
ranks on each of those 64 selections. These are work counts for runtime
revision `64a692b6f82ae10353f50c8741377166bc6cf854`. Their contribution to
elapsed time remains unmeasured.

## Workloads and controls

Each run has the same three instructions. It receives one local input with
obligation `work`, sets a tag, then succeeds. Each store starts with one
resource root per run and the static execution index armed. The first ordinary
`AdmitInput` invalidates the index. The fixture checks that this transition
happened exactly once. One synchronous `Store.RunProgress` call then advances
the runs to quiescence.

- **Fallback.** A single receiver has one matching input and all remaining
  inputs carry the permitted obligation `parked`. Every admission succeeded.
  The matching observation sorts last. Unrelated receivers must inspect the
  same retained IDs before their turns end. The control moves the eligible
  receiver from last to first in run order, keeping both dimensions fixed.
- **Guarded.** Every receiver has its own matching input. Closing and reopening
  the store after admission arms the index from retained state. In the first
  round each candidate is immediately eligible. The candidate list retains
  its full width as the round-robin cursor advances.

The driver runs each fixture twice, once with stock selectors and once with
counters inserted by a temporary Go build overlay. Both runs check the full
ordered sequence of commit events against the expected subjects and sequence
numbers, the selected admission IDs in history, successful payloads, waiting
peers, quiescence, zero dispatches and replayed history. Each instrumented
call also checks which selector ran. [Workload and checks](fixture_test.go.txt)

## Observed counts

The fallback window covers the first selection. The guarded window covers
the first receive of every run. **Observation rows** counts successful
retained-row validations inside selection, including guarded point reads.
**Run visits** counts entries into the actual candidate traversal.
**ID checks** counts receive-loop iterations, including IDs for other runs.
**Ordered ranks** counts `len(ordered)` immediately after the guarded selector
clones and rotates its candidate list. It measures logical entries materialized.
Allocator growth, bytes copied and the later compaction pass have separate costs.

| Eligible receivers | Runs | Retained observations | Selections | Observation rows | Run visits | ID checks | Ordered ranks |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Last, fallback | 1 | 1 | 1 | 1 | 1 | 1 | 0 |
| Last, fallback | 8 | 8 | 1 | 8 | 8 | 64 | 0 |
| Last, fallback | 8 | 32 | 1 | 32 | 8 | 256 | 0 |
| Last, fallback | 32 | 8 | 1 | 8 | 32 | 256 | 0 |
| Last, fallback | 32 | 32 | 1 | 32 | 32 | 1,024 | 0 |
| Last, fallback | 64 | 64 | 1 | 64 | 64 | 4,096 | 0 |
| First, fallback | 32 | 32 | 1 | 32 | 1 | 32 | 0 |
| All, guarded | 1 | 1 | 1 | 1 | 1 | 1 | 1 |
| All, guarded | 8 | 8 | 8 | 15 | 8 | 8 | 64 |
| All, guarded | 32 | 32 | 32 | 63 | 32 | 32 | 1,024 |
| All, guarded | 64 | 64 | 64 | 127 | 64 | 64 | 4,096 |

At 32 runs and 32 rows the fallback loaded all 32 observations in both
positions. Moving the eligible receiver to the front reduced ID checks from
1,024 to 32. Across the entire 64-by-64 last-ready call, including the final
quiescence scan, four selections validated 256 rows and made 16,192 ID checks.

In the first guarded round each receiver needs one point validation for its
input. After the first receive, the selector also revalidates the preceding
consumption. This accounts for the 127 rows at 64 receivers. The full call
committed 192 reductions and made 193 selections. It visited 256 candidates
and materialized 12,352 ordered ranks. All bounds include both the measured
window and full-call totals in the [recorded rows](rows.jsonl).

The earlier P7 direct-adapter Execute window had zero executions and zero
returned rows for the global observation query in both its baseline and
order-overlay arms at bounds 8 and 256. Its point query with `WHERE id=?`
returned 45 and 1,533 rows respectively. The archived stdout for all four
counter runs matched the digests in the P7 counter records. P7's timing pilot
addressed retained-history insertion. [P7 results](../retained-query-scaling/results.md)
and [counter records](../retained-query-scaling/data/counters-rows.jsonl)

## Reproduction

The recorded host ran Go 1.27.4, darwin/arm64, macOS 27.3 and Python 3.15.2.
The stock module pin was `github.com/mattn/go-sqlite3 v1.14.52` and Replay
reported SQLite 3.53.4. To reproduce the archived rows from the runtime
repository root, export the measured commit to a fresh directory and copy the
measurement harness there. This fixes the executor, fixture helpers and module
pin even when the checkout has moved forward.

```sh
set -eu
scratch=$(mktemp -d)
base=64a692b6f82ae10353f50c8741377166bc6cf854
evidence=docs/evidence/scheduler-readiness
git archive --format=tar -o "$scratch/base.tar" "$base"
mkdir "$scratch/runtime"
tar -xf "$scratch/base.tar" -C "$scratch/runtime"
cp -R "$evidence" "$scratch/runtime/docs/evidence/"
CGO_ENABLED=1 GOTOOLCHAIN=local python3 -B "$scratch/runtime/$evidence/measure.py" > "$scratch/observed.jsonl"
diff -u "$evidence/rows.jsonl" "$scratch/observed.jsonl"
```

For a fresh measurement against the current checkout, run
`CGO_ENABLED=1 GOTOOLCHAIN=local python3 -B docs/evidence/scheduler-readiness/measure.py`.
Each row includes the SQLite version reported by that run.

The driver uses `go test -mod=readonly -count=1`, requires unique insertion
points in the source and excludes setup, index initialization, commits and
Replay from the counters. The bounds concern a single owner, fixed-size
programs and valid local inputs. Work at other frontier widths, the frequency
of fallback in applications and any benefit from changing the selector remain
open. [Counter locations](measure.py)
