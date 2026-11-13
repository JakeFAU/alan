# Report materialization contract

Frozen before production changes, 2026-11-13.

Base: `5f5d382c8d1e9c7369e7624e3a962746c9bf2229` (merged PR #15),
tree `cb6dc03624010d13f8af8d045af44e490110bfaa`. Work begins on the clean
`codex/p5-report-scaling` branch in the existing isolated worktree. The prior
campaign and its archives remain unchanged.

## Question and scope

Can full Run reports read retained facts in batches instead of issuing point
queries for every root, run and operation? After that separately measured
change, can an optional smaller execution result avoid materializing the
global report while preserving necessary validation?

The previous B256 targeted experiment recorded 790,220 SQL statements,
including 625,792 in finishRun, and emitted 65,536 root records. These are
historical measurements, not new results. Full reports after each of n
observations have quadratic output size. Batching can reduce redundant SQL
and decoding; it cannot remove that output requirement.

## Binding invariants

- Existing RunOptions, RunReport, Run and inspection output remain compatible.
  Full reports retain every field, deterministic ordering, nil/empty distinctions,
  pending/unknown classifications, and retained-data validation/error behavior.
- Scheduling, cleanup, worker joins, result retention, authority, reservations,
  dispatch admission, settlement and publication remain authoritative and unchanged.
- Original operation IDs, approval targeting, independent charges, selected
  observations, uncertainty and conflicting receipt evidence remain unchanged.
- Reuse is call-local to the existing owner lock or read transaction. No new
  persistent/process-lifetime cache, schema, tables, migration or dependency.
- Fresh Open, Replay and typed inspection remain independent. No local report
  summary becomes authority. Existing historical formats and owner exclusion stay.
- A compact result, if added, is explicitly opt-in and separately measured.
  It must not suppress checks needed for existing safety or integrity guarantees.
  If that requires a new validation certificate/protocol, defer it and retain a
  concrete counterexample instead of widening this optimization.

## Approach and falsifiers

Prefer small typed batch readers over a generic query cache. Read original
identities, retain source descriptors, and apply existing decoding/correlation
rules when traversing the original semantic relationships. Read-ahead must not
turn unused corrupt evidence into an earlier error or turn used corrupt
evidence into success. A conservative fallback to the original read path is
acceptable for unsupported shapes or SQL value domains. It performs no writes.

Compare against an independent frozen report implementation and same-seed
genuine execution histories, including pending, failed, unknown and completed
effects, invocation publication, revocation, historical formats and corruption.
Attack missing references, swapped identities, malformed descriptors, receipt
correlation, scan errors and competing readers. Reopen and actual acknowledged
SIGKILL histories must preserve evidence and contact counts.

Stop or narrow the change if an invariant cannot be preserved without a new
architecture, or if measured work/time is indistinguishable from noise. No
weakened assertions or altered expected semantics to manufacture improvement.

## Measurement and delivery

Use deterministic local adapters, ordinary public execution and protected
targeted approval paths at bounds 1, 8, 16, 32, 64, 128, 256. Measure full
reports separately from any compact result. Use separate comparable processes,
one excluded warmup per arm/bound and six balanced adjacent AB/BA pairs.
Record wall/CPU time, allocated bytes/counts, cumulative peak RSS, original
contacts/charges/results and explicit final inspection/reopen/Replay costs.
Collect diagnostic SQL/profile evidence separately from primary timing.
Report medians and spread; preserve failed rows and every command's actual exit.

Freeze source before final ordinary, integration, integration-race, build, vet,
module, formatting and Staticcheck gates. Preserve exact source hashes and
review findings, including rejected candidates. Obtain independent semantics,
recovery, performance and scope review. Commit a reproducible compact evidence
package and open a draft PR; do not merge or release. Do not repeat the prior
large artifact publication campaign without a demonstrated need.
