# P7 retained-query scaling

**Decision: stop before the production change.** The history-first order
removed 130,305 SQLite scan steps at B256, but the stock paired pilot found
a median Execute reduction of **−0.326%**, with only **3 of 6** pairs faster.
The pilot did not meet any of the three frozen advance conditions. No
production source was changed and no Task 2 work began. The order-only
counterfactual exists in disposable copies of the exact baseline. See the
[qualification record](qualification.md) for the source bridge and reproduction
checks.

## Inputs and ordinary baseline

- Baseline commit `9db473bae28e5b806c3ccdc7e2e134c1d0b38f27`, tree
  `56669e93b11a35ab8a67d09918b3a5e055944015`, archive SHA-256
  `62336d8ca4f2f5328ec22ea51ebe97402c0ce2a48aec507fc005212ccac0e5eb`.
- Contract `7a4a46c`; harness initially frozen at
  `481de68a98f50efc0c79918a822444afb4464074` and corrected before any
  counter or primary samples at
  `ca65abb857a75f3677ce88a74cbeb94a2251450b` (tree
  `bb51e53578fda278c7b02a08c50c6efc9b56f7d8`).
- Overlay parent is the baseline above; the retained patch SHA-256 is
  `b343f561fe77970a062f79dc3428d719227308f392fcbefcd35525b5dd7cd388`.
  Only `internal/executor/store.go` differs: SHA-256
  `8a746b7526b0e2a9eec7f1d12a64046dd29b67375afe098e462d75a3f252d882`
  before and `694aa7816fdedc56d054c25cd9e9dc24cad899b231247a43780d198e45106ef6`
  after. Both complete exports were compared before the unique adjacent
  statement swap; their inventories were checked again after each lane.
- Apple M4 (Mac16,10, 10 logical CPUs, 16 GB), macOS 27.2/26B123
  (Darwin arm64 27.2.0), Go 1.27.4, Apple clang 21.0.0
  (2100.4.1.1), Python 3.9.6, bundled SQLite 3.53.4
  (`2026-07-24 19:02:57 bf7c7f30031888f4e796e429ab3978879485813aaca6f641c7b33e4e09459bcc`).
  The module graph is ALAN and `github.com/mattn/go-sqlite3 v1.14.52`;
  stock builds carry its recorded module checksum, diagnostic builds
  bind exact replacements. Module verification returned 0.
- Explicit process configuration included `GOMAXPROCS=4 GOGC=100
  GOMEMLIMIT=off CGO_ENABLED=1 GOTOOLCHAIN=local GOWORK=off GOPROXY=off
  GOSUMDB=off GOENV=off GOFLAGS=-buildvcs=false`. The gate ledger retains
  exact argv, cwd, environment including compiler/SDK and cache paths,
  exit, start/end, and raw output and input inventory hashes.
- Fresh ordinary baseline `go test -count=1 -json ./...` from a new
  export, with loopback permitted: exit 0, 1,420 test pass events,
  18 passing packages, 3 with no test files, 4 skipped subtests, 0
  failures. The `TestRequestAuthorityFormatsConflictPromotionAndOldReader/{13,14}/frozen_old_reader`
  subtests require `ALAN_PHASE3B3_OLD_READER`; the
  `TestRequestBindingConflictAndOldReader/{false,true}/frozen_reader`
  subtests require `ALAN_PHASE3B4_OLD_READER`. This is ordinary baseline
  coverage, not a historical-reader qualification.
- First ordinary attempt retained: exit 1, 411 test pass events, 27
  failed test events, 9 failed packages, no skips. Multiple fixture
  servers hit sandbox loopback bind errors (e.g. `httptest: failed to
  listen on a port: listen tcp [::1]:0: bind: operation not permitted`);
  the unchanged exact-source suite passed outside that sandbox.

## Method and evidence

Both arms ran the same reused P6 public direct-adapter workload and
`RunProgress` in Execute. Per position it checked progress and exactly
one new contact; final full `Run`, typed inspection/series proof,
close, reopen, another full `Run`, resumed inspection and independent
`Replay` checked retained original IDs, selections, approval links,
result digests, one charge per position and unchanged completion and
contact counts within each sample. Every B256 row in both pilot arms reported
256 completed positions, 256 original observations selected, 256 original
results retained, 256 independently charged effects and 256 direct adapter
contacts. Replay found 256 intents and 1,023 history entries. The driver
checked 255 protected approvals, 256 distinct roots, one effect charge on
each root and zero pending, failed or unresolved series positions. Each fresh
Create used its own namespace. The driver compared identities and evidence
within that Store through reopen and Replay.

Fresh SQLite controls ran against the extracted baseline DDL using
the pinned bundled driver. At X=1,8,64,256 seeded observations (mixed
subjects/generations except the one-row case), the original insert
counted X−1 full-scan steps and 62, 97, 377, 1,337 VM steps; the
reordered insert counted zero full-scan steps and 52 VM steps at each
bound. NULL selection without consumption counted zero / 48. The
observations table rootpage was 13; the insert VDBE's `FkIfZero`
at 51 jumps to 62 past the observation cursor's 53–61 loop, which
contains the deferred decrement at 60. The consumption update used
the history PK index (rootpage 12); its deferred increment path is
present when the parent lookup fails. FK enforcement was on and
`defer_foreign_keys` was off. Both orders produced the same ordered
observation/history rows and empty `foreign_key_check` from a fresh
connection. A subsequent deliberately failed nonnull update followed
by rollback retained neither staged fact. These controls do not
constitute crash or application recovery qualification.

| Bound | Baseline → overlay SQL executions | Row returns (both) | VM steps | Full-scan steps |
|---:|---:|---:|---:|---:|
| 8 | 4,964 → 4,964 | 3,960 | 62,346 → 61,838 | 182 → 77 |
| 256 | 164,428 → 164,428 | 131,184 | 2,739,111 → 2,341,555 | 227,970 → 97,665 |

All four fresh pinned counter samples passed. The complete Execute
template registry differed only in history scans (105/130,305 → 0),
history VM steps (2,041/448,193 → 1,548/51,148), and 15/511 fewer VM
steps for consumption updates. Each arm executed 31/1,023 history
inserts and 15/511 consumption updates at B8/B256. There were no
consumption scans, no new scan family, no migration to another
lifecycle stage, and unchanged result-lookup (56/65,280) and
charge-count (21/32,385) scans. The retained-position sums for those two
families on the tested fixture are B(B−1) and (B−1)(B−2)/2. This accounts
for all 97,665 remaining Execute full-scan steps in the B256 overlay. The
observed quadratic scan work survives the write-order change. These counts
describe this workload and pinned SQLite access paths.

The derivative binds three exact prepared SQL strings at prepare.
Its fixed aggregate clock wraps only the pinned C step calls: row
exec, eager query first step, later cancellable or noncancellable
steps. It excludes buffered returns, preparation, marshaling and
commit; it includes cgo call overhead and any in-bracket preemption.
Clock-off keeps the same driver paths without timing or increments;
empty reads both clocks before C with the same out-of-bracket
aggregation. Timing starts after the initial Execute counter snapshot
and stops before the terminal snapshot. Pinned and new controls
passed, including eager/next/done step accounting, exclusion of
lookalike statements and calls outside Execute, and no timed calls
while disabled. All timed counts reconciled to pinned step counters.

| Bound | Family | Actual steps per row | On median [range], ms | Empty median, ms |
|---:|---|---:|---:|---:|
| 8 | History | 31 | 0.067748 [0.066498–0.073086] | 0.000878 |
| 8 | Authorization result | 16 | 0.008542 [0.008418–0.009502] | 0.000460 |
| 8 | Root charge count | 8 | 0.005332 [0.005085–0.005791] | 0.000126 |
| 256 | History | 1,023 | 5.854341 [5.819599–5.911789] | 0.024032 |
| 256 | Authorization result | 512 | 2.266195 [2.262958–2.275697] | 0.011344 |
| 256 | Root charge count | 256 | 1.261961 [1.249369–1.265495] | 0.005415 |

At B256 instrumented Execute wall ranges were 1,902.643–1,910.613 ms
(on), 1,912.849–1,933.166 ms (off), and 1,907.417–1,996.305 ms (empty).
Off recorded zero clock totals. At B8 the corresponding ranges were
60.677–65.307, 60.779–61.566 and 61.120–73.232 ms. In this small
diagnostic comparison the differences in whole-window elapsed time
do not isolate clock overhead; the empty bracket shows the aggregate
timing floor. These values include required step work and cannot be
read as savings. The stock pilot supplies the causal elapsed-time
contrast.

The separate stock baseline profile row passed. The CPU profile sampled
4.05 CPU seconds across a 3.31 s whole lifecycle, with 0.61 s flat in
`runtime.cgocall`; its scope and sample resolution cannot isolate the
three suspect SQL statements. Local CPU and sampled allocation profiles
and their four flat/cumulative renders are independently hash-bound.

## Uninstrumented paired pilot

Each child created a fresh Store. Both stock binaries were built before
serial samples began. The recorded campaign builds, tests, profiles and
counter runs did not overlap primary timing. The ledger does not observe
unrelated host background work. Per bound there was one excluded warmup per
arm and six adjacent pairs, alternating AB and BA. All 98 rows passed (14
warmups, 84 primary). Positive paired percentages favor the overlay;
each is the median of the per-pair normalized differences. The time
columns are the separate medians for each arm.

| Bound | Baseline Execute, ms | Overlay Execute, ms | Paired Execute | Faster pairs | Savings MAD, ms | AB | BA | Paired lifecycle |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 3.992 | 3.997 | −0.152% | 3/6 | 0.259 | −0.706% | +0.402% | +0.469% |
| 8 | 40.743 | 40.719 | +0.152% | 3/6 | 0.227 | −0.317% | +0.621% | −0.029% |
| 16 | 81.639 | 81.519 | −0.194% | 2/6 | 0.786 | −0.204% | −0.184% | −0.428% |
| 32 | 159.236 | 157.925 | +0.309% | 4/6 | 0.931 | +0.085% | +1.675% | +0.408% |
| 64 | 314.590 | 312.810 | +0.900% | 5/6 | 2.267 | +0.709% | +1.842% | +0.563% |
| 128 | 628.283 | 628.322 | +0.004% | 3/6 | 1.288 | −0.294% | +0.150% | +0.085% |
| 256 | 1,290.647 | 1,294.152 | −0.326% | 3/6 | 17.415 | +1.828% | −1.132% | −0.157% |

B256 paired Execute differences in order (baseline minus overlay, ms)
were +35.268250, +1.711749, +23.848917, −14.597666, −10.144458 and
−28.664668. Median saving was −4.2163545 ms, below its 17.4148125 ms
median absolute deviation (MAD).
The frozen screen required at least +1%, at least five faster pairs
and savings strictly above MAD. The opposing order strata and near
zero B128 contrast limit the interpretation; the small bounds show no
clear elapsed-time regression pattern or general benefit. The entire
inclusive history-step aggregate was about 5.85 ms under diagnostics,
whereas 1% of the stock baseline median is about 12.91 ms. The
different builds and timing boundaries make this a scale comparison,
not a strict bound or a substitute for the measured pilot.

| B256 medians | Baseline | Order overlay |
|---|---:|---:|
| Summed RunProgress calls, ms | 1,065.852 | 1,068.429 |
| Execute CPU (user + system), ms | 1,841.091 | 1,838.619 |
| Execute allocation volume, MiB | 637.704 | 637.519 |
| Execute allocation count, millions | 8.7110 | 8.7101 |
| Execute end peak RSS, MiB | 31.953 | 32.477 |
| Post-exit peak RSS, MiB | 77.969 | 78.648 |
| Final full report, ms | 22.123 | 22.101 |
| Final typed inspection, ms | 108.177 | 107.994 |
| Final series view, ms | 0.768 | 0.748 |
| Final series proof, ms | 0.884 | 0.878 |
| Fresh Open, ms | 695.102 | 695.465 |
| Full Run after Open, ms | 37.455 | 37.280 |
| Resumed inspection and series view, ms | 109.786 | 109.695 |
| Replay, ms | 564.696 | 564.977 |
| Full lifecycle, ms | 3,133.527 | 3,136.067 |
| Lifecycle CPU (user + system), ms | 4,407.033 | 4,403.279 |
| Lifecycle allocation volume, MiB | 2,155.149 | 2,154.914 |
| Lifecycle allocation count, millions | 25.4826 | 25.4808 |

The small CPU and allocation differences do not establish meaningful
savings; the high-water RSS ranges overlap (31.31–32.63 versus
31.81–32.72 MiB at Execute end; 76.98–79.45 versus 77.45–79.95 MiB
at process exit). Peak gauges are cumulative and allocations are volume,
not live bytes. The lifecycle evidence does not rescue the failed Execute
screen. This is a single-host bounded engineering decision, without an
inference of statistical significance or a claim of zero possible gain.
The scan reduction is reproducible on the pinned diagnostic builds. The
stock latency contrast misses the threshold in this workload, so the frozen
decision is to stop here. The two remaining query families were measured and
are unchanged.

The durable evidence artifacts are under
`docs/evidence/retained-query-scaling/data/`. The row files retain every
sample, the full stage vectors and per-row command/output digests;
the gate ledger and environment map retain exact commands, cwd, exits,
skips and hashes. The evidence bridge identifies every changed file
after the corrected harness freeze and checks against a byte-for-byte
unchanged baseline application runtime.

To rerun a lane, use a clean checkout of the corrected harness commit and
exclude the local plan workspace. Choose a fresh output directory for each
invocation. Add `--gocache PATH` to reuse a writable Go build cache.

```sh
git checkout --detach ca65abb857a75f3677ce88a74cbeb94a2251450b
P7=.superpowers/sdd/plan-retained-query-scaling
printf '\n/.superpowers/\n' >> "$(git rev-parse --git-path info/exclude)"
mkdir -p "$P7"
python3 -B -m unittest discover -s docs/evidence/retained-query-scaling -p test_measure.py
python3 -B docs/evidence/retained-query-scaling/measure.py --lane clock --preflight --out "$P7/repro-controls"
python3 -B docs/evidence/retained-query-scaling/measure.py --lane smoke --out "$P7/repro-smoke"
python3 -B docs/evidence/retained-query-scaling/measure.py --lane counters --out "$P7/repro-counters"
python3 -B docs/evidence/retained-query-scaling/measure.py --lane clock --out "$P7/repro-clock"
python3 -B docs/evidence/retained-query-scaling/measure.py --lane profile --out "$P7/repro-profile"
# Run serially without overlapping campaign builds, tests or diagnostics.
python3 -B docs/evidence/retained-query-scaling/measure.py --lane timing --out "$P7/repro-timing"
```

Run ordinary tests from a fresh Git archive of the baseline above with
`go test -count=1 -json ./...` and with permission for local loopback
servers. This baseline route is independent of the lane controller.

## Attempt history

- Initial Python red controls failed on missing apparatus; corrected a
  generator search that matched both input and output literals, and a
  control string-quoting typo before running any diagnostic samples.
- Preflight clock run 1 stopped before generation because the recorder
  refused a duplicate `baseline-tree` gate name. The next preflight used
  a new root and passed 35 test events, including independent fixed-control
  checks; the final harness checks the P6-returned tree without a second
  gate. The finalized Go controls reran in the fresh clock lane.
- First frozen smoke passed all four stock rows at `481de68`. The
  first counters attempt built both diagnostic binaries but rejected
  build metadata before samples because the locally replaced module
  does not have the stock checksum/tab suffix. The corrected parser
  uses dependency fields and exact replacement path; fresh counters,
  clock, profile and pilot used the corrected clean harness `ca65abb`.
- Raw roots are kept under `.superpowers/sdd/plan-retained-query-scaling`,
  including failures. A packaging self-check initially read the wrapper path
  instead of the binary argument; correcting the argument index produced
  the complete compact evidence from the unchanged raw rows. The evidence
  commit retains compact rows and
  custody records. The raw archives remain local.
- Early controller and packaging corrections have narrative and partial
  artifacts. Separate terminal-outcome logs were unavailable for the
  initial Python red controls, duplicate-gate exception and packaging
  correction. The ordinary baseline failure and final frozen control logs
  are retained in full.
