# Selection: reuse membership only within one root report

This record must be committed before production edits. It selects one narrow
change, conditional on the completed baseline attribution and independent
invariant review. It does not approve an implementation or claim a speedup.
The [experiment contract](contract.md) governs qualification and stopping.

## Evidence and hypothesis

Production base is `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315`.
The [baseline report](baseline-attribution.md) records 41 separate diagnostic
and preliminary processes. At B=256, Execute performs 11,770,629 SQL executions;
8,421,376 are repeated membership queries in `finishRun`'s root × intent loop.
Every measured prefix reconciles those SQL executions with the independent
source-loop counter. Separate coarse timing puts root reconstruction at 52.959s
of 104.087s Execute. Sampled CPU stacks locate its membership work under SQLite;
statement frequency alone is not the timing argument.

This is not the largest allocation component in every interpretation. At B128,
selection has the largest sampled cumulative allocation among selection, typed
inspection, root reporting and descriptor reconstruction. The enclosing Run and
cross-cutting decoder totals are larger and overlap. Readiness and mandatory
approval snapshots remain substantial; no second optimization is authorized.

## Mechanism and governing invariant

Inside `readRootCompletions`, lazily retain each successful
`compositionMembershipRoot` result in a local `map[RunID]string`. Key by the
complete subject and generation. The first encounter performs the original
query at its original traversal position. A later encounter in the same call
reuses only that root string. Map presence distinguishes an unseen key from a
successful empty result. Propagate errors; never cache a partial result.

**Invariant:** every reuse belongs to the same queryer and supported stable
read interval as the first lookup. No supported writer can change membership
within that interval. The map neither escapes nor survives return, error,
process death, a new call or owner reopening. It is not durable authority.

The exact query predicate is subject plus generation. Its result does not
contain epoch, revision, authority, resource balance, closure, result, receipt
or selection state. Those facts remain checked in their original places.
Namespace/queryer identity is implicit only because one call owns the map;
there is no Store field, shared helper cache or persistent summary.

All roots and intents still traverse in their existing deterministic order.
Keep the root × intent Go loop and repeated sorting unchanged. Do not batch
queries, eagerly prefetch memberships, cache complete reports or combine
settlement/authority boundaries. Memory is O(U) for unique source RunIDs among
already materialized intents and is released with the call.

## Why the existing caller cuts suffice

The complete production caller search finds `finishRun` and
`validateCompositionStore`; the latter is reached through these existing paths:

| Caller | Existing stable boundary |
| --- | --- |
| `finishRun` | Workers have been canceled/joined; `Store.mu` is held for the complete helper call on the pinned owner connection. Other Store mutations need that mutex. The process owner lock remains held. |
| Create | Provisioning and validation share one transaction before the Store is published. |
| Open | Validation uses the renewal transaction while the owner lock is held; activation renewal follows successful validation. |
| Replay and raw Inspect | Validation reads one read-only transaction snapshot, even if another supported owner progresses. |

Close cancels and waits for Run, waits for owner users, then takes `Store.mu`
before closing the connection and releasing ownership. It cannot release the
owner lock during `finishRun`. The report helper and its descendants only read;
`prepareRoot` is distinct from the separately invoked settlement writer.
The independent review checked actual callees and membership write sites,
including atomic dynamic admission, rather than assuming purity from names.

Public typed `InspectExecution` uses a different coherent table snapshot and
in-memory validator. It does not call this helper and remains unchanged.
The contract remains cooperative single-host ownership. An arbitrary SQLite
writer bypassing the owner lock is outside the supported failure model;
this change does not add distributed fencing or a new corruption guarantee.
Any future caller or helper mutation must re-establish the stable-cut premise.

## Required work, reusable work and temporal boundaries

| Work | Treatment and reason |
| --- | --- |
| Original operation, authorization, protected approval, reservation/debit, admission, result/uncertainty, settlement, selection and durable continuation | Necessary existing evidence and decisions; all preserved. |
| Repeated membership predicate for the same full RunID in this stable call | Same input and same cut; reuse only this derived string. |
| New report after dynamic admission, authority change, result/receipt retention or settlement | Rebuild the map and every mutable fact; the previous call supplies no reusable authority. |
| Fresh-owner validation, Replay and inspection | Keep independent reconstruction and existing version checks; no restored map or trusted previous report. |

A missing membership is legitimate under existing lookup semantics: empty
means the operation is unscoped and skipped by this root comparison. It is
not corruption, a synthetic root or an unresolved lookup. General dynamic
admission may introduce a new child on a later call. An existing unscoped
member is not thereby entitled to arbitrary reparenting.

Root external-known lists currently depend on retained result presence;
typed effect consequences additionally describe remote uncertainty and
conflicting receipts. Preserve both exact interpretations. An unsuccessful
local result or local quiescence must not become remote nonapplication/finality.

## Error and cancellation boundary

The existing protocol/version gate runs before membership access. First-use
lookup errors remain at the same sorted position, with unchanged propagation;
no failure becomes absence and no partial graph becomes success. Existing
root descriptor, status, charge, preparation and per-kind reads remain.

Eliminating a redundant read also eliminates that read's opportunity to observe
an injected I/O failure or elapsed-time cancellation. Identical outcomes for
such races are not claimed. Context is neither detached nor replaced. Entry
and uncached SQL queries still receive it; subsequent root SQL, finishRun's
following run query, and transaction parents retain their cancellation/commit
gates. No new per-map-hit context check is required for this reporting-only
invariant. Authority/admission checks are never elided.

## Positive witnesses and negative controls

Qualification must preserve the following independent obligations:

- Genuine ordinary histories with multiple roots and repeated source intents:
  complete report order, original IDs, charges, outcomes and uncertainty match;
  the reduced membership-query assertion fails on baseline and passes candidate.
- Full-generation keys, repeated empty membership and first lookup errors.
  Narrow synthetic queryer fixtures are labeled helper tests, not fabricated
  positive runtime executions. A same-key absent-to-present helper fixture
  tests lifetime only; lawful dynamic-child admission is tested separately.
- Independent Stores with coincident RunIDs and distinct namespaces/root IDs;
  successive calls cannot share cached roots. Later membership, authenticated
  results, settlement and conflicting receipt evidence remain visible.
- Cancellation before entry/first lookup and an error at a retained later SQL
  boundary; corrupted original references still reject independently before
  owner renewal. Do not require faults at a deliberately removed nth query.
- Actual acknowledged SIGKILL/reap, repeated reopening, concurrent readers,
  Close/owner exclusion, revoked/stale authority, and opposite-build readers.
  No state survives solely in the map and no spent opportunity is refunded.
- Exact same-seed SQL cells and public inspection/History/Replay/request/result
  evidence under the contract's narrow normalization. Keep original lost versus
  retained response distinctions and separately approved later observations.
- Historical formats and the independent eight-operation external consumer.

## Falsifiable predictions and regression criteria

For the frozen successful fixed-B schedule, the helper currently performs B*k
membership queries at position k. The candidate should perform k: cumulative
B*k*(k+1)/2 becomes k*(k+1)/2. At full B=256 this predicts
8,421,376 → 32,896 finishRun membership executions, 8,388,480 fewer. The unchanged
root × intent work counter should still report 8,421,376. If other source loops
are unchanged, total Execute SQL should become 3,382,149 from 11,770,629.
Explain deviations rather than hiding them in aggregate latency.

Conditional performance forecast: at least 30% lower B256 median Execute wall
and at least 10% lower cumulative Go allocation in the predeclared paired
campaign. Coarse root timing and sampled membership allocation motivate these
empirical forecasts; neither is a guarantee. No peak-RSS reduction is promised.
The contract's six positive B256 paired differences and small-bound regression
guard remain the acceptance criteria. A smaller supported improvement may
falsify the forecast and still be reported honestly. Do not change the schedule,
select samples, weaken assertions or add another optimization to meet it.

## Review and stopping

Independent source challenge `reviews/task4-proposed-invariant-review.md`
returned **APPROVE for this proposed one-call invariant only**. It inspected
all caller/callee ownership boundaries and decisive retained lost/retained/
settled result exports; it did not run or approve an unwritten implementation.
The raw report, source references, falsifiers and historical evidence limits
will accompany delivery. Implementation, exact differential evidence, recovery,
compatibility and measured performance still require independent review.

Reject a Store/global/persistent cache, subject-only key, empty-as-unseen logic,
eager query reordering, error-as-absence, stale authority fallback, omitted
validation, fabricated positive histories or changed format/owner assumptions.
If this one-call proof fails or qualification exposes an unrepairable semantic
counterexample, deliver the diagnostic negative result. No public API,
dependency, schema/table/index, migration, scheduler or dispatch change is in
scope. Stop production optimization after this one change.
