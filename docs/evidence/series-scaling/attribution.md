# Cost attribution and the selected change

Baseline: `af6943a5805a9a6c0e8fe77f804264f4765d07ec`.
Implementation: `e32d0e2f020638f301dcd0f0687c5c444180225c`.
The [experiment contract](contract.md) preceded profiling and implementation;
the [invariant analysis](invariants.md) was committed after profiling and before
the production change. This report describes diagnostic evidence. Confirmatory
performance results are reported separately.

## Workload and measurements

The external [driver](driver/README.md) declares an ordinary BoundedSeries with
N positions. It provisions one initial approval, then obtains a fresh public
snapshot and protected continuation approval before each successor. Each
position has a distinct original effect, independently charged opportunity,
retained response and selected observation. The deterministic adapter returns
equal business values with distinct original request and response identities.
No network or provider is involved in these timings.

Execute encloses all N ordinary Run calls and N-1 snapshot/basis/approval calls.
Construction/provisioning, Create, final inspection, completed Close/Open/Run,
and Replay are measured separately. It is not a benchmark of just the adapter
or just a private predicate. Every sample checks all N original causal chains,
charges, contacts, results and selections, then checks them again after reopen.

Three independent diagnostic configurations separate effects of measurement:

1. Unmodified source with phase wall/CPU, Go allocation and process RSS counters.
2. A generated copy with aggregate function, parser-input, reducer, SQL execution,
   row-return, full-scan-step and SQLite VM counters.
3. Another generated copy with coarse inclusive wall timers, without the SQL or
   per-root instrumentation. Nested and concurrent times cannot be added.

Separate ordinary-source CPU and allocation profiles use the default allocation
sampling rate; a 16-position execution trace is retained. Detailed probe runs
and profiles never enter the confirmatory timing samples. The instrumentation
is reproducible from [its generator and definitions](instrument/README.md).

## Baseline diagnostics

These are one diagnostic sample per bound, not medians or an inferential result.
GB here means 1,000,000,000 bytes of cumulative Go allocation, not resident memory.

| Positions | Plain Execute (s) | Allocated GB | Descriptor decodes | Decoder input bytes |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 0.004800 | 0.001701 | 28 | 12,096 |
| 8 | 0.117573 | 0.070549 | 588 | 2,409,036 |
| 16 | 0.395654 | 0.308128 | 1,948 | 16,166,452 |
| 32 | 1.633120 | 1.530007 | 6,972 | 116,536,980 |
| 64 | 8.383641 | 8.931945 | 26,236 | 880,139,092 |
| 128 | 53.399672 | 57.698383 | 101,628 | 6,845,153,940 |
| 256 | 359.390405 | 408.262630 | 399,868 | 54,111,337,364 |

The initial hypothesis included repeated full-store validation. The measured
ordinary Execute interval calls the ledger/protocol replay validators **zero**
times. This does not mean there is no validation: fresh typed inspection occurs
N-1 times, and authority, readiness, reservation and dispatch checks remain.
Open, Replay and final readers have their own measured work outside Execute.

The coarse wall measurements locate the largest scope:

| Positions | Execute (s) | Root settlement sweeps (s) | Root reporting (s) | Fresh typed inspection (s) |
| ---: | ---: | ---: | ---: | ---: |
| 8 | 0.096586 | 0.016606 | 0.005544 | 0.017191 |
| 16 | 0.354256 | 0.093161 | 0.026869 | 0.063043 |
| 32 | 1.540423 | 0.599929 | 0.146817 | 0.244255 |
| 64 | 8.246916 | 4.194489 | 1.074971 | 0.948623 |
| 128 | 50.704232 | 31.349937 | 7.524552 | 3.645160 |
| 256 | 337.680399 | 235.700172 | 54.144934 | 14.620408 |

At 256, root settlement sweeps occupy 69.80% of that diagnostic Execute.
There are 1,535 sweeps and 392,960 PrepareRoot entries. Each preparation asks
whether the retained profile uses the invocation protocol. The old predicate
decodes the complete, growing profile to read its immutable Version field.
398,333 predicate calls account for all but 1,535 of the descriptor decodes.

At 128, the sampled allocation profile attributes 75.29% of allocated bytes to
paths beneath invocationEnabled. This is an allocation attribution, **not** a
claim that 75.29% of wall time can be removed. The predicate path's cumulative
CPU share in that profile is only 2.43%; garbage collection, scheduling and
C/SQLite work appear elsewhere.
The CPU profile contains substantial system-stack and pthread condition work.
Those samples alone cannot assign all elapsed time to one caller.

Root reporting also performs real repeated work: at 256 there are 8,421,376
root/intent pairs and 8,429,051 membership-query executions across Execute.
The measured reporting-pair count follows N²(N+1)/2 for this controlled schedule.
The independent initial advice favored this SQL repetition conditionally; the
coarse timers changed the selection because reporting occupied only 16.03% of
256 Execute, versus 69.80% in settlement sweeps. The unused membership-summary
proposal and the change in advice are retained as research evidence.

## One production optimization

The existing profile-retention constructor already validates and privately owns
the canonical provision. It now stores the boolean expression
`out.Version == compositionProtocolVersion` in that same object. The predicate
reads this boolean. The descriptor decoder remains unchanged for callers that
need an independent complete descriptor.

The production diff is four insertions and six deletions in two private files.
The flag consumes constant space per retained profile. It is neither a ledger
cache nor authority. Create and Open construct a new validated profile; no
owner borrows another owner's summary. Dynamic approval, source revision,
admission, closure, result and expenditure facts retain their existing checks.
The [invariant analysis](invariants.md) identifies invalidators and falsifiers.

The candidate diagnostic counter maps at every bound differ only in descriptor
decode count and input bytes; the other measured function, SQL and reducer
counts are exactly equal. In particular, it does not remove root sweeps, SQL
queries, readiness checks, snapshots, settlements or paid contacts.

| Positions | Baseline decodes | Candidate decodes | Candidate decoder input bytes |
| ---: | ---: | ---: | ---: |
| 1 | 28 | 5 | 2,160 |
| 8 | 588 | 47 | 192,559 |
| 16 | 1,948 | 95 | 788,405 |
| 32 | 6,972 | 191 | 3,192,565 |
| 64 | 26,236 | 383 | 12,848,501 |
| 128 | 101,628 | 767 | 51,661,285 |
| 256 | 399,868 | 1,535 | 207,720,805 |

The independent predicate regression changes from 27 allocations/query on the
baseline to zero on the candidate. That regression identifies the mechanism;
the full paired ordinary workload determines the performance claim.

The separate candidate coarse-timer diagnostic at 256 observes 7.085s in root
settlement sweeps and 53.717s in root reporting within 104.798s of Execute.
Fresh typed inspection remains 14.553s and readiness 24.811s. These single
diagnostic observations support the identified mechanism and show the remaining
reporting cost; they do not replace the paired timing estimate. No further
optimization was added after that assessment.

## Measurement limits and work left intact

Detailed instrumentation materially perturbs timing. At 256, the baseline
counter run took 416.462s versus the separate plain diagnostic's 359.390s; the
candidate counter run took 194.907s versus its plain pilot's 105.700s. The
overheads differ substantially. These one-sample calibrations are not exact
causal overhead corrections, and instrumented times are never compared as a
speedup. Counter equality and growing decode traffic support attribution;
unmodified paired executions test its practical effect.

SQL totals cover the instrumented driver paths, excluding connection-initializing
direct sqlite3_exec PRAGMAs. SQLite full-scan steps are cursor movements, not a
count of complete scans or all rows examined. Exact indexed rows examined are
unavailable in this build. Returned rows and VM steps are retained separately;
none is relabeled as that missing quantity. Histograms have a bounded overflow
bucket. Allocation traffic omits SQLite/C allocations; RSS includes them and
is a whole-process high-water mark.

No root-membership cache, incremental settlement index, profile-wide shared
descriptor, snapshot cache, scheduling rewrite, table, migration, dependency
or public API change was added. The remaining SQL/scanning work is a possible
future hypothesis requiring its own invariant and qualification. This result
does not establish a universal complexity or latency bound.
