# P2: marginal cost of retained execution

This contract precedes new measurements and production changes. The user requests
cost attribution, at most one justified optimization, complete semantic/recovery
qualification and a draft PR. A well-supported negative result is acceptable.

## Source and environment

Verified default branch: `main`, `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315`.
PR #9 is merged. GitHub rebased its commits: the old delivery `9bd13a2` is not
an ancestor of this base, but both have exactly tree
`eda6f49da75bcb81d70eaaa244536d7258a9a312`. Its 427-file export manifest is
`a4bb3c7a7241c8f5d73698867a0860e2a1c7e151bf467b8394cf649cb9ba3859`.
The immutable profile-mode projection is preserved. Earlier evidence is read-only.

Host: Apple M4, Mac16,10, 10 CPUs, 16 GiB RAM, macOS 27.2. Toolchain: Go
1.27.3 darwin/arm64, CGO 1, Apple clang 21, go-sqlite3 v1.14.52, Staticcheck
2026.2.2. Exact commands, environment, hashes and exits accompany each run.
Python controllers record their actual interpreter; no cross-version relabeling.
Timing processes use `GOMAXPROCS=4`, `GOGC=100`, unlimited GOMEMLIMIT,
`GOWORK=off`, local toolchain and offline pinned modules. Local fixture servers
and real process termination run with host permissions; denied-bind attempts
are failed setup evidence, never qualified runs.

## Authoritative workload and boundaries

One public-package driver defines the workload for both revisions. It preserves
the existing author.BoundedSeries request/await/success definition, input,
first approval at provisioning, one selected operation per Run, a fresh public
ContinuationBasis and protected approval for each subsequent position, and
single in-flight dispatch. Bounds B are 1, 8, 16, 32, 64, 128, 256.

The main P2 campaign uses a deterministic loopback HTTP endpoint with finite
responses and a server-side contact journal independent of the adapter's
Execute-entry record. Both journals must match actual retained Prepared and
Candidate identities, fields, digests, response IDs and selection evidence.
No live provider requests, retry, automatic continuation or additional authority.
HTTP startup/shutdown are outside ordinary Execute. Retained endpoints are not
silently normalized in differential comparisons; identical schedules use the
same controlled endpoint and genuinely provisioned seed where required.
The endpoint runs in the driver's process with a separate handler-side journal:
its HTTP allocation, CPU and GC are included in process measurements. This is
independent contact evidence, not independent process-resource accounting.

A direct-adapter mode is retained only as a bridge to P1: three fresh ordinary
runs at B=32 and B=256 on the current baseline, separately labeled. A material
difference from the old 1.082s/105.522s figures is investigated, not discarded.
The loopback campaign establishes its own comparable baseline; it is not claimed
to be the historical P1 workload.

Record construction/provisioning, Create and original-state verification,
ordinary Execute, approval snapshot/basis, approval commitment, Run, final
inspection, SeriesView/proof, Close/Open/resume, Replay and total process time
separately. Ordinary Execute covers its complete enclosing window, including
measurement bookkeeping. Nested scopes are inclusive and must not be summed as
disjoint costs. Successful runs retain B operations/contacts/charges/results/
selections, B-1 approvals and 4B-1 reductions, with unchanged IDs and provenance.

## Marginal and attribution protocol

Let T(B,k), Q(B,k), A(B,k) describe the measured prefix after k selected
observations of a declaration that provisioned B roots at the start. Measure
adjacent increments within the same process, with approval for operation k
included in its interval. Report both direct iteration intervals and cumulative
prefixes; explain measurement overhead and any residual against the outer window.
Do not call T(256,8) a standalone eight-position execution, or subtract unrelated
samples to invent an exact 256th-operation cost. Standalone T(B,B) differences
are interval comparisons across separately sampled workloads.

Use per-stage counters and process resource snapshots, without extra inspection
or Replay at prefix boundaries. Record wall/user/system CPU, TotalAlloc,
Mallocs, GC count/pause and available GC CPU metrics, peak RSS, declaration size,
retained positions and actual contact counts. Process-wide GC/CPU measurements
are not exact allocation or CPU charges to one runtime operation. All 98 confirmatory
rows collect prefixes. Separately, run two baseline-only prefix-on/off pairs at
each B=1,8,32,256, in on/off then off/on order, without additional warmups. These
16 fresh processes provide a descriptive overhead calibration, excluded from the
confirmatory comparison; two pairs cannot establish negligible overhead. Off
disables only the new per-prefix resource snapshots/records, preserving all
ordinary operations, existing stage measurements and final identity proofs. Do
not subtract estimated overhead from measured latencies. Freeze endpoint payload,
response bounds, timeouts, connection reuse and no-retry policy in the reviewed
harness before collecting authoritative data. Never force GC at prefix boundaries.

Diagnose all seven bounds with aggregated function/work/SQL counters in disposable
source/dependency copies. Reuse and test the prior SQLite prepare/execute/step/row/
VM/fullscan instrumentation; do not log millions of SQL statements. Normalize
literals while preserving identifiers. Report overflow explicitly and fail
attribution if its contents could conceal a top-ten family. Classify top families
by table/resource, read/write, point/range, phase and actual calling operation.
Use exact scope counters or declared sampled caller stacks, with sample counts
and limitations. Record transaction begin/commit/rollback and connection use;
an unresolved transaction/caller attribution stays unresolved.

Keep CPU/allocation profiles separate from counter-heavy diagnostics. Collect
normal sampled CPU/alloc profiles at B=32 and B=128, a trace at B=16, and coarse
inclusive timing at B=128 and B=256. Diagnose reducer/commit, authority, grants/
approval, reservation, prepare/admission, retention/settlement, observation
selection, root reports, immutable profile, SQL, transaction/connection,
serialization and GC costs. Report unavailable per-category allocation/time
rather than fabricate an attribution. Do not sum inclusive CPU or wall samples.

Compare linear/quadratic/cubic work models and fixed-B adjacent increments.
Source-loop recurrences are conditional on the measured schedule; fitted growth
is not an asymptotic proof. SQL count alone is not latency evidence. Record both
queries and their sampled/coarse cost, input sizes and per-call rates where valid.

## Selection and semantic gates

Before any production edit, commit a separate selection/invariant record with
the measured bottleneck, proposed single mechanism, validity boundary, evidence
that can invalidate it, prediction, negative controls and regressions. An
independent reviewer must challenge why the repeated work is unnecessary.
Options include leaving implementation unchanged, reducing repeated reads within
one existing stable operation, or batching a proven equivalent read. A persistent
mutable account/report cache is not presumed safe.

Preserve every original identity, generation/epoch, authority decision, grant,
reservation/debit, dispatch bound, receipt, settlement, observation selection,
pending obligation and historical causal relationship. No public API, dependency,
schema/format, global cache, scheduler, background worker or ownership change.
No required durable fact is removed. Unknown application remains unknown.

Compare baseline/candidate against the same genuine seed and action schedules:
complete public inspection/History/Replay, requests/results and typed SQL schema/
cells, with byte equality where promised. Only source labels, paths and physical
row order may be normalized, with every exception enumerated. IDs, authority,
epochs, accounting and request parameters are never normalized. Use disposable
corruption copies and independent endpoint logs. Exercise failed/revoked admission,
lost and retained responses, later receipts/conflicts, repeated reopen and real
ACK/SIGKILL/reap cuts after authorization, payment, admission, acceptance, retention,
settlement, selection and before subsequent approval. Preserve historical formats,
opposite-build readers and the independent external consumer.
Hash and clone archived main+WAL files after the writer is reaped and before any
Open/resume; original archives are never opened by qualification readers. The
endpoint proof must reject wrong operation/key, prepared digest, request scope,
response identity, missing/duplicate contacts and missing/reordered prefixes.

## Confirmatory comparison and delivery

After freezing the candidate, use separate fresh processes and one frozen schedule:
one excluded warmup per build/bound, then six adjacent baseline/candidate pairs
per bound with balanced AB/BA order and predeclared shuffled bounds (seed 20261104).
All 98 rows are mandatory. Bind executable/source/tool hashes and environment;
preserve every outcome. No replacement or opportunistic exclusion. Counter/profile
runs never enter timing medians. No other task-launched tests, builds, profiling
or compression during timing. Disclose shared-host load, caches, thermals and
CPU-placement uncertainty. Report per-size median/range/paired ratios, CPU,
allocations, RSS and GC; no universal performance claim.

Regression guards: at B=1,8,32 a candidate median slowdown above both 10% and 1ms
requires investigation; no retained-evidence/contact/debit mismatch is acceptable.
A candidate must reduce the directly attributed work and show a repeatable latency
effect beyond observed paired variation at large bounds. The primary endpoint is
the ordinary Execute-window wall time at B=256. Predeclare a one-sided exact sign
check on its six paired baseline-minus-candidate differences: all six must be
positive (p=1/64 under independent fair null signs; ties do not count as wins).
Report the ratio of source medians, median paired ratios and every difference;
the sign check concerns paired differences, not the ratio of medians. Other bounds, phases and
prefixes are descriptive, dependent measurements, not additional significance
tests. The sign-model assumptions are conditional and shared-host drift limits
the inference. No forced speedup target.
If a material regression or semantic counterexample cannot be repaired within the
one change, retain diagnosis and reject the candidate. Do not weaken tests or
expand architecture to rescue it. Repeat attribution after the change, then stop.

Run baseline relevant suites and final immutable-export focused, ordinary, full
integration, full integration-race, process/replay/inspection/compatibility,
external-consumer and tool tests; build/vet, formatting/whitespace, module verify/
tidy-diff and applicable Staticcheck. Preserve all failures and skips, including
the known competing-read ordering issue. Final before/after source hashes must
match the published commit. Independent reviews cover attribution/statistics,
invariant/authority, recovery/compatibility and exact differential integrity.

Deliver compact SQL/marginal/cost-model reports, one qualified optimization if
justified, reproducible tools, checksummed raw evidence, all review corrections,
the ten requested answers and exact base/contract/implementation/tested/delivery
identities in a draft PR. Do not merge, release or implement the next hypothesis.
