# Bounded series scaling contract

Frozen before production changes, 2026-11-04. This experiment asks whether one
demonstrably repeated computation can be avoided without changing what an ALAN
execution may do. A negative result is acceptable. Only one production
optimization may be selected; profiling is not permission for a wider rewrite.

## Source and workload

Verified main is `af6943a5805a9a6c0e8fe77f804264f4765d07ec`, tree
`5d1f64029c7fd764ddd1941bc77b831beb3bdcb3`. Both PRs 7 and 8 are merged. This
tree differs from reviewed series tree `ea8f8b185116d152dd09e21d2a65623451d18c5b`
only by the four-file Staticcheck cleanup; those four files match its reviewed
delivery. The immutable baseline has 389 files and filename/content manifest
SHA256 `4f91d07c23a31f433f795ebdffc5bc81394ed829c11f07abd64905b9c214f3d8`.

Use one ordinary public `author.BoundedSeries` declaration with a deterministic
in-process adapter, one opportunity per position, explicit first approval and
one protected approval per successor. Bounds are 1, 8, 16, 32, 64, 128 and 256.
Every successor obtains a fresh public inspection snapshot and continuation
basis. No direct SQL shortcut supplies authority. All N operation identities,
paid opportunities, adapter entries, selected results and completion facts must
be checked independently; N-1 successor approvals must retain their original
predecessor, target, selection and evidence relationships. No live requests.

Ordinary Execute includes the first through last Run and all intervening public
snapshot/basis/approval calls. Construction, provisioning, Create, final readers,
Close/Open and recovered execution are reported separately. A separate declared
reopen schedule measures recovery, rather than folding it into ordinary timing.
The historical 364-second sample used a different configuration and supplies
motivation only, not the denominator for a new speedup claim.

## Measurement and selection gates

1. Before optimizing, run a diagnostic baseline at every bound. Collect public
   phase wall/CPU time, Go cumulative allocations/counts and process peak RSS.
   Preserve CPU, allocation and representative execution-trace profiles.
2. Collect aggregate function/SQL/scan/reduction counters using a separately
   identified instrumented source/dependency copy. Keep original hashes and
   reproduction patches. Detailed counters and profiling runs are excluded from
   headline timings; compare their overhead at representative bounds. Returned
   rows, SQLite full-scan steps and VM instructions are distinct measurements.
   If exact indexed rows examined are unavailable, say so explicitly.
3. Classify expensive work as necessary, unchanged-input repetition, or a fresh
   check across a temporal boundary. Freeze the chosen function, its premises,
   invalidating mutations, reuse condition and falsifiers in a separate written
   analysis before editing production. Require profile and growing work-count
   evidence; source suspicion or a helper microbenchmark alone is insufficient.
4. Implement only that change. Keep all authority/admission/settlement checks
   whose inputs can change. A retained in-memory summary is never durable
   authority and must be reconstructed through normal validation on reopen.

The measured environment is Go 1.27.3 darwin/arm64, CGO1, Apple clang21,
`github.com/mattn/go-sqlite3 v1.14.52`, Apple M4 (10 CPUs, 16 GiB), macOS27.2.
Use GOMAXPROCS4, default GOGC100 and unlimited GOMEMLIMIT, identical build flags
and local filesystem, and pinned offline modules for both sources. Record actual
effective settings, host load, executable/source/harness hashes and commands.
No other experiment, build or broad test suite runs concurrently with timing.

Confirmatory measurement uses six serial fresh-process pairs at every bound,
after one excluded fresh-process warmup per source/bound. Alternate AB/BA within
pairs and predeclare the complete schedule before starting. Each process uses a
fresh database; OS caches remain warm and uncontrolled. Do not add samples until
a desired p-value or target is reached. Retain slow valid samples; externally
invalidated samples require an independently recorded reason and replacement of
the entire pair. Report all raw values, medians, spread, paired ratios, CPU,
allocation traffic/count and process peak RSS. Allocation traffic excludes C
allocations and is not resident memory; peak RSS is a process high-water mark.

The primary target is a 25% decrease in median ordinary Execute at 256. It is
aspirational. At 1/8/16/32, flag median increases above max(5%, 0.5ms), and phase
increases above max(10%, 0.5ms); these are empirical guardrails, not equivalence
proofs. Any inference treats 256 Execute as primary and other endpoints as
descriptive. Six pairs offer limited precision. Compare finite growth models and
adjacent-size slopes with residuals; no universal asymptotic law is inferred.

## Semantic and delivery gates

Preserve original operation IDs, grants, approval targeting, paid-predecessor
eligibility, resource debits, causally significant order, revocation, dispatch
admission, immutable results/conflicts and unknown remote consequences. Compare
baseline and candidate on matched deterministic schedules and actual ordinary
writer process histories, using exact retained bytes where promised and explicit
normalization only for identified permissible differences. Final weather values
alone are not an equivalence criterion.

Retain actual acknowledged SIGKILL/reap coverage of approval commitment/selection,
source authorization, reservation/gating, lost versus retained response, result
selection, terminal settlement, unknown third plus approved fourth, and repeated
reopen. Exercise corrupt stores, coherent foreign deployments, stale evidence,
same-store owner exclusion, independent owners and concurrent readers. Attack
every new reuse premise independently. A process death is not logical completion.
Inspection and Replay remain independent, read-only and effect-free; failed Open
must not partly renew authority or mutate durable state.

No new dependency, table, migration, public execution abstraction or BoundedSeries
API change; no global scheduler, distributed ownership or provider subsystem.
Existing format and historical-reader contracts remain intact. Run ordinary and
integration suites, integration race, relevant fuzz/regression/compatibility and
public-consumer tests, build, vet, formatting, module verification and Staticcheck
on the frozen delivery source. Record actual exits/skips and source hashes.

Obtain independent semantic, recovery, methodology and compatibility/scope
reviews of actual source and evidence. Preserve rejected findings/corrections.
Deliver source, profiles/data, histories, commands, invariant analysis, reviews
and a bounded research summary in a draft PR. Do not merge or release.

Stop the optimization rather than expand it if correctness cannot be repaired
within this one change, evidence does not support the selected bottleneck,
improvement is within noise, a format/architecture change is needed, an existing
guarantee must be weakened, or unbounded memory/unrecoverable local state appears.
In that case deliver the negative evidence and the precise next hypothesis.
