# Structural correction contract — 2026-11-13

This is a prospective amendment to the [original contract](contract.md), frozen
after the user's correction and before the structural implementation. The
original contract and its measurements remain attributable to commits 85bcdee
and 2484251. The interrupted, unqualified bulk-read prototype is preserved in
the working evidence; it is not the implementation qualified by this contract.

## Diagnosis

The measured executor does quadratic bookkeeping. At 128 and 256 observations,
SQL executions are 836,827 and 3,246,427 (3.88 times), while executions per
observation nearly double. The source explains this: `(6B-1)*B` root checks and
`B*B` full-report root visits. Repeated root-by-intent matching adds
`B*B*(B+1)/2` comparisons in this particular schedule, a cubic term. These are
algorithmic traversal counts, not a statistically established wall-time law.

Replacing individual queries with bulk queries does not eliminate repeated
global traversal. Acceptance must count visited identities, SQLite returned
rows, full-scan and VM steps, and validation/reduction work as well as statements.
The objective is to stop revisiting unaffected work, not to hide those visits
inside fewer SQL calls.

## Source and measurement reset

The integrated baseline is `ef38ca47179d86e7544885471ae43e86e1d6351e`, tree
`4b8f76b5fd84df99e13c4a8381dde65d1b70746b`. It normally merges verified main
`c72c396df09c55b0be1a91b2557dcec2ef6457a8` into the continuation of P3
`d59eaf2976849e450635d252c7bca62eeccbef2e`; reviewed commits were not rewritten.
GitHub still reports PR14 open and unmerged. Main's request-binding work remains
in scope for compatibility, not for optimization or redesign.

The resumed host has Go 1.27.4, darwin/arm64, with the unchanged go-sqlite3
v1.14.52 dependency. Rebuild both measurement sources with this toolchain; do
not compare new timings to Go 1.27.3 as a controlled pair. The original temporary
Task1 raw directory did not survive the interruption. Its committed summaries
and hashes remain historical evidence, not currently available raw captures.
Preserve new source exports, logs and captures until delivery and publication.

## First structural implementation

Permit one disposable private execution index within a single Store owner. It
may retain candidate identities and dependencies across ordinary Run calls;
it may not replace any authority, reservation, admission, reduction, result,
settlement or publication check in the existing transactions. No persistent
cache, new schema/table/index/dependency, background worker, public execution
abstraction, or change to BoundedSeries is permitted.

Initially optimize only independently validated static composition: no dynamic
invocation templates or bindings, handoff, request-authority profile, or
reconciliation; each root owns its generation-one root run; only ordinary
adapter operations and empty terminal destinations. Check actual validated
profile/program shape and retained state, never application names or benchmark
identifiers. Unsupported shapes and mutations keep the original execution path.
The complete eligibility and mutation inventory is in the retained independent
[source design review](reviews/structural-design.md).

Initialize only after successful Create/Open validation and activation. Retain
all observation-validation coverage, including consumed observations. Preserve
run rotation, absent-last behavior, lazy first-reached errors, observation-ID
ordering, per-Run attempted-delivery state, and root settlement order. Only
unchanged negative readiness results may be skipped. Reload a selected run's
authority; Commit still validates it independently. Root publication remains
separate from external uncertainty.

Publish explicit changed identities only after successful commits in Commit,
ApproveContinuation, AuthorizeIntent, adapter reservation/result/settlement,
dispatch admission, and SettleRoot. Every other successful mutating commit
disables reuse for that owner. Preserve existing lock order: direct commits
while ownerMu is held must not call the waking helper that reacquires ownerMu.
Post-commit notification errors do not undo committed deltas. Rollbacks cannot
publish them. Bookkeeping failure after commit disables reuse without claiming
that the durable operation rolled back.

Before a trusted transaction and before reused reads, check the pinned
connection's total_changes, schema context and private limits/profile premises.
Unexpected writes, changed main/temp schema, triggers or mismatched context
disable reuse and restore the original traversal. A post-commit watermark alone
is insufficient: it could swallow corruption preceding that transaction.
Do not eagerly replace the fallback with full validation, which changes lazy
error order. Fresh Open independently validates again; cached state is never
transferred across owners or Stores. This remains the existing cooperative
single-host ownership contract, not hostile-writer or distributed fencing.

An SQLite update hook alone is rejected: it omits some REPLACE and truncate
deletions and is not a committed semantic log. The documented
[update-hook](https://www.sqlite.org/c3ref/update_hook.html) and
[total-changes](https://www.sqlite.org/c3ref/total_changes.html) boundaries must
remain explicit in the implementation and attacks.

Keep full default RunReport output. Group root/intent membership and charges
once per report instead of rebuilding cross products or rescanning all charges
for each root. Full output still costs at least its size; that is a caller/API
choice, not a durability requirement. Do not claim generally linear end-to-end
execution. A compact report option is deferred until the first structural
change is measured, and would require a separate explicit contract. The planned
targeted owner-only ContinuationBasis reader remains permitted; the independent
full snapshot and Replay contracts remain unchanged.

## Discriminating qualification

In addition to the original qualification, require RED/GREEN structural tests:
after initialization, repeated unchanged scheduling and a change to one of many
independent roots must not reload unrelated run/observation/settled-intent
history or prepare unrelated roots. Count initial work separately. Count live
frontier maintenance and full-report work separately; a large live frontier or
one large root is a control, not a claimed constant-cost case. The ordinary
bounded-series path must not continually fall back to global traversal.

Compare original helpers and optimized execution on matching deterministic
schedules: IDs, selected observations, approvals, charges, contacts, unknown
results, exact report contents, Replay and inspection. Include duplicate/no-op
writes, cancellation/rollback, failed commit, post-commit errors, direct
corruption followed by a legitimate write, DELETE-all, REPLACE, triggers, limits
and schema changes, foreign Store state, unsupported lifecycle/authority changes,
and fresh-owner recovery. Retain all existing process cuts and final gates.

The original seven-bound matrix and six balanced pairs remain the timing
protocol, using the integrated baseline and matched toolchain. Report unchanged
full-snapshot/full-report behavior before separately labeling targeted-basis
savings. Preserve failed/slow rows and independent reviews. No speedup target
permits weaker semantics. If complete mutation coverage cannot be established
within this narrow static path, stop with the counterexample and measured
partial result rather than creating a general incremental execution framework.
