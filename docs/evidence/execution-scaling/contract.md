# Bounded execution work contract

Frozen before production changes. This follows the user's approval to address
the structural amplification behind P3's 3,246,427 SQL executions for 256
observations. The goal is a defensible amount of work per new observation,
preserving the same independently auditable computation. No speedup factor is
promised.

## Source and scope

Implementation baseline: `d59eaf2976849e450635d252c7bca62eeccbef2e`, tree
`dc082ea07e4c24d8f36e56d756771f7c1154b0ec`. On verification, PR #14 is still an
open draft and `origin/main` is `926a899b651fccc48be52a054612a92afcf1ddfd`.
The new branch is `codex/p4-bounded-execution-work`; P3 and its frozen evidence
remain untouched. Delivery will be a separate draft with the dependency stated
unless P3 has merged before delivery. No merge or release.

Preserve original operation IDs, independent charges, protected approvals,
compiler-derived grants, causal order, source/dispatch admission, revocation,
receipt conflicts, unknown remote outcomes, selection and settlement semantics.
Preserve ordinary owners and competing readers, historical formats, independent
Replay/inspection and fresh-owner validation. No new dependency, table, index,
schema migration, global scheduler, persistent scheduling cache or execution
abstraction. Do not change `author.BoundedSeries` or `RunReport` contracts.

One narrow read API is permitted: `Store.ContinuationBasis(ctx, target)`, if the
existing owner cannot obtain the required evidence without full inspection.
This is a read of the validated owned Store, analogous to `Snapshot`; it is not
a replacement for the independent full-history audit. Approval remains a
separate transaction that rejects changed evidence. It does not approve, spend,
settle, dispatch, or advance epochs. Existing snapshot-based code stays valid.

## Established cost and output limits

P3's retained counter evidence attributes repeated work to all-root settlement,
per-run readiness probes, per-intent delivery probes, root-by-intent report
construction and full inspection at every approval. Its global observations
loop performs 29,358,102 candidate considerations at bound 256. These are Go
loop iterations, not SQL statements. The 65,536 report roots are cumulative
root visits, not distinct retained roots.

The unchanged public workload returns B reports, each containing B roots, and
requests B-1 complete snapshots. Therefore its total output/materialization has
an unavoidable quadratic lower bound. This milestone must not call that whole
path asymptotically linear or remove output silently. Instead distinguish:

1. SQL executions and avoidable nested matching/probing in ordinary execution.
2. Rows examined and copied by coherent call-local bulk reads.
3. Required full report output and optional full-history audits.
4. End-to-end execution using full-snapshot approval versus targeted approval.

The first implementation uses bulk reads with semantic indexes whose lifetime
ends with the locked call/transaction. Reduce SQL-per-record loops and unrelated
cross products before considering persistent state. No process-local summary
may become authority. If this leaves material quadratic costs, report them;
do not rename them linear or broaden into a new scheduler.

## Semantic boundaries and falsifiers

Readiness preserves the same run rotation, first eligible observation by its
original ID, program/binding validation, reducer calls and first-reached errors.
Bulk reads must not hide missing related rows via an inner join or eagerly
decode an otherwise unvisited corrupt projection. All observations previously
validated at the call boundary remain validated, including consumed records.
Bound programs cannot alias reused unbound programs; handoff exclusion remains.

Delivery status comes from the same original IDs and retained acknowledgments,
settlements and publications. It is not inferred from equal payloads or final
counts. Invocation readiness still uses its original preparation rules.

Root preparation uses the same freeze/closure/publication precedence and
descendant/remote-uncertainty rules. Only a retained publication, or absence of
both freeze and closure under the same cut, can justify avoiding deeper work.
Preserve root ordering and recheck inside the existing settlement transaction.
Reports retain all roots, charges, pending dependencies and uncertainty.

Targeted basis returns the first retained approval's historical basis if one
exists. Otherwise it identifies exactly one allowed paid adapter operation for
the declared predecessor/effect, with its authentic prepared digest, dispatch
admission and optional retained result digest. Multiple candidates fail closed.
Result absence remains absence, not remote nonapplication. The existing
`ApproveContinuation` comparison stays authoritative after the read.

Tests must distinguish baseline/candidate on work counts, not merely final
values. Impossible identities, binding/correlation corruption, stale authority,
wrong generation, unrelated Stores, conflicting receipts, post-read evidence
changes and actual SIGKILL/reopen must not gain authority. Failures and rejected
candidate rules remain in evidence. A narrow optimization that requires weakened
validation, new durable state or a new execution architecture is withdrawn.

## Measurement protocol

Toolchain is Go 1.27.3, darwin/arm64, existing go-sqlite3 v1.14.52. Record OS,
CPU, native compiler/SQLite identity, dependency and source/binary hashes. Use
`GOMAXPROCS=4 GOGC=100 GOMEMLIMIT=off GOWORK=off GOTOOLCHAIN=local` for both
sources, identical durability and single-in-flight deterministic local adapter.
No live provider requests. Heavy measurements run serially.

Reuse the P2 public workload and proof through an explicit, hash-bound driver
overlay; do not overwrite earlier campaign tools. The workload still performs
B ordinary Run calls, B independent contacts/reservations/results/selections,
B-1 separately committed approvals and 4B-1 reductions. Prefix checks and full
returned reports remain inside Execute. Construction/Create, initial audit,
final audit, completed reopen/resume and Replay remain separately measured.

Two comparisons are required: unchanged snapshot approval on both sources, and
candidate targeted approval against candidate snapshot approval with identical
retained semantic evidence. An end-to-end old/new recommendation must identify
both changes; it cannot attribute API-use savings solely to internal SQL work.

Before confirmatory execution, freeze actual binaries, driver/controller hashes
and the exact serial schedule: bounds 1,8,16,32,64,128,256; one excluded warmup
per implementation/mode/bound; six adjacent measured pairs per comparison/bound,
alternating order AB/BA. Do not discard, replace or relabel failed or slow rows.
Targeted bound 1 is a no-approval control. Report medians, ranges and every pair,
CPU, cumulative allocations/count, GC and lifetime peak RSS. No universal latency
or memory bound. Small-bound guard: slowdown above both 10% and 1ms at 1,8,32.
The practical B256 target is at least 25% median improvement, with all six
baseline-minus-candidate differences positive; failure is a valid negative
finding, not permission to weaken semantics or tune the sample.

Diagnostic counters and profiles are separate processes, never timing inputs.
Count SQL executions, returned rows, fullscan steps and VM steps separately;
count matching visits, root checks, bindings/reductions and contacts. Aggregate
bounded counter keys; do not serialize cumulative per-statement maps at every
prefix. Retain raw JSON/logs, meaningful CPU/allocation profiles and the
instrumentation patch. Confirm outside the original working directory.

## Qualification and review

Freeze source before final ordinary, integration and integration-race suites;
focused race/causal/authority/corruption/compatibility tests; real acknowledged
SIGKILL/reap and fresh-owner cases; external eight-operation consumer; build,
vet, module verification/tidy-diff, formatting/whitespace and ordinary/integration
Staticcheck (including race configuration). Record commands, working directories,
exit codes, skips, exact input hashes before/after. A blocked sandbox baseline
attempt is recorded as blocked, not a runtime failure or successful test.

Obtain independent semantic/durability, public-use, measurement and whole-change
review of actual source and decisive evidence. Fix substantive findings and
requalify affected code. Publish a draft PR with compact reproducible evidence,
raw-artifact hashes/locations, actual results and remaining limitations.
