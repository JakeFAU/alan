# Bounded series execution scaling

Retaining one immutable protocol-mode flag reduces the observed median execution
time of a 256-position BoundedSeries from **347.99s to 105.52s** on the measured
host. Cumulative Go allocation decreases from **408.17GB to 62.16GB**. Every
original operation, protected approval, paid opportunity and retained result
still passes the same runtime checks. The public API and SQLite formats are
unchanged.

This is a bounded implementation-efficiency result. It does not establish a
universal latency or complexity bound, or a new execution capability.

## What profiling found

The initial hypotheses included full-store validation, root settlement and
fresh approval snapshots. Full ledger/protocol replay validation was called
zero times during ordinary Execute in this workload. Fresh typed snapshots and
dynamic authority checks did occur and remain necessary.

The largest measured repeated work was inside root settlement. Its protocol-mode
predicate decoded the whole privately retained composition profile each time it
asked whether the invocation protocol was enabled. At 256 positions, there were
398,333 such calls and 54.111GB of total descriptor input traffic. The profile's
version does not change during that private object's lifetime.

The sole validated constructor now stores that version comparison as a private
boolean. `invocationEnabled` reads it; `descriptor()` is unchanged. Create and
Open independently validate and reconstruct the profile. The boolean neither
summarizes mutable ledger evidence nor supplies authority. Root readiness,
approval, revision, reservation, settlement and final dispatch checks are intact.

The production change is four insertions and six deletions in two private files.
There is no new dependency, table, migration, public API, persistent cache or
ownership mechanism. The [contract](evidence/series-scaling/contract.md) and
[invariant analysis](evidence/series-scaling/invariants.md) were frozen before
the production change.

## Controlled paired result

Six fresh-process pairs at each bound follow one excluded warmup per source and
bound. Adjacent pairs alternate execution order; bound order is predeclared.
The deterministic local adapter uses the ordinary public authoring, Store,
approval and execution paths. Execute includes all runs and intervening fresh
snapshot/basis/approval calls. No instrumented or profiled sample enters this
table. All 98 processes passed; no measured sample was discarded or replaced.

| Positions | Baseline median | Candidate median | Reduction |
| ---: | ---: | ---: | ---: |
| 1 | 4.349ms | 4.179ms | 3.9% |
| 8 | 103.591ms | 90.748ms | 12.4% |
| 16 | 366.924ms | 294.358ms | 19.8% |
| 32 | 1.633s | 1.082s | 33.7% |
| 64 | 8.251s | 4.211s | 49.0% |
| 128 | 51.130s | 19.771s | 61.3% |
| 256 | 347.986s | 105.522s | 69.7% |

At 256, all six candidate runs are faster than their paired baseline. The
observed ranges are 340.83–383.19s and 103.38–108.48s. Go allocation volume falls
84.77%, allocation events 47.28%, and process CPU time 66.22%. Median peak RSS
increases slightly, from 78.38MiB to 78.80MiB (+0.55%); this is not a resident
memory saving. No predeclared small-bound or phase regression guard is crossed.
These guards and six samples do not prove performance equivalence everywhere.

The host is an Apple M4 with 16GiB RAM, Go 1.27.3 darwin/arm64, CGO 1,
go-sqlite3 v1.14.52, `GOMAXPROCS=4` and `GOGC=100`. Background host load, caches,
thermals and CPU placement were uncontrolled; recorded one-minute load reached
16.586. No other task-launched test, build or experiment ran during measurement.
Both sources and the driver were built and run outside the development working
directory. No independent-machine replication is claimed.

At 256, the measured SQL count remains 11,770,629 and root/intent reporting
pairs remain 8,421,376. Descriptor decodes fall from 399,868 to 1,535. This
explains a specific reduction in repeated work without claiming a global
validation cache or less required computation. [Full results](evidence/series-scaling/performance.md)
include all ranges, allocations, CPU/RSS, phase costs, separate recovery samples,
growth fits and limitations. [Attribution](evidence/series-scaling/attribution.md)
records the profiles, probe coverage and instrumentation overhead.

## Semantic and recovery evidence

Four exact same-seed eight-position schedules cover success, retained refusal
and repeated reopening. Twenty-one process-archive comparisons, including an
ordinary reference, cover
approval commitment/selection, authorization, reservation, dispatch admission,
lost versus retained response, settlement and termination. Complete public
snapshots, History, Replay, requests/results and typed SQL cells match across
builds without identity, epoch, grant or expenditure normalization. A separate
reviewer independently reproduced all 25 comparisons.

Candidate process histories are also read by the baseline implementation.
Genuine conflict promotion preserves formats 11→12 and immutable originals.
The existing external eight-operation consumer and historical-format suites
pass. Independent reviews cover semantics, chronology, recovery, compatibility,
scope, attribution and measurement; rejected tooling findings and corrections
are preserved. [Qualification](evidence/series-scaling/qualification.md) and
[review history](evidence/series-scaling/reviews.md) identify the tested cuts,
negative cases, exact command families and limits.

## Source identities, evidence and remaining work

- Verified baseline: `af6943a5805a9a6c0e8fe77f804264f4765d07ec`, tree
  `5d1f64029c7fd764ddd1941bc77b831beb3bdcb3`. Relative to the reviewed series tree
  `ea8f8b185116d152dd09e21d2a65623451d18c5b`, only the previously merged
  four-file Staticcheck cleanup differs.
- Contract freeze: `5067bcf`; invariant freeze:
  `ddcac16c0d5fed3d252ab8cfb8b5e3687a15cb7a`.
- Timed and initially qualified implementation:
  `e32d0e2f020638f301dcd0f0687c5c444180225c`, tree
  `e19d165ef5353223f34710dcd166a325bcd5fdbd`. The delivery adds this report and
  reproducible qualification tools without changing that production source.

Raw runs, profiles, traces, schedules, commands, source/build hashes, actual
crash stores and full review reports are retained separately from library
source. The draft PR's delivery record binds its exact head and post-commit
qualification. [Reproduction instructions](evidence/series-scaling/reproduce.md)
explain both external modules and the measurement-only probes.

The measured candidate still takes about 105.5s at 256 positions. Root reporting
is now the largest measured coarse scope in the candidate diagnostic. A future
experiment could investigate repeated membership queries within their actual
read boundary. That needs its own invariant and qualification; it is not part
of this change. This experiment supports reuse of one immutable profile fact,
not arbitrary validated history prefixes. Existing single-host ownership,
external-effect uncertainty and resource guarantees remain unchanged.
