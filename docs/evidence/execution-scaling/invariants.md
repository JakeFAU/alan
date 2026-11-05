# Frozen invariants and edit boundaries

Implementation source is `d59eaf2976849e450635d252c7bca62eeccbef2e`.
These are requirements for subsequent implementation, not claims that an
optimization has passed. The [contract](contract.md) governs any conflict.

## Readiness and delivery

`runner.go:320–405` first obtains sorted original RunIDs and rotates after an
exact retained `last`; an absent `last` restarts at the first ID. It reads and
validates **all** observations before visiting a run, including consumed and
unrelated observations (`replay.go:280–327`). Only then may a call-local map group
unconsumed observations by full RunID. Each receive tries its own IDs in the
original order; ErrIneligible skips that observation, while other errors stop.
Do not reduce observations eagerly or retain ineligibility across calls.

Run projection/program/binding validation stays lazy in that rotated order.
Bulk raw inputs must preserve missing related rows rather than hide them with
an inner join; unvisited corrupt projections must not fail an earlier selected
run. Current generation, retirement, program binding and handoff exclusion stay
exact. Selection grants no authority: Commit's original reread and reduction
remain mandatory (`store.go:806–950`).

`pendingDeliveries` validates all intent descriptors before completion lookups
and all terminal obligations before return (`runner.go:425–518`,
`replay.go:329–361`). Bulk completion sets use exact ACK operation IDs, exact
computed settlement IDs and exact invocation-delivery IDs. Equal payloads,
result presence and counts cannot substitute for delivery or settlement.
Invocation preparation runs only for unresolved invocations. Historical stores
without composition tables remain readable where those tables were previously
unneeded. Category opportunities, attempted-set/active-root limits and internal
before external delivery ordering remain unchanged (`runner.go:131–260`).

## Roots, reports and error boundaries

Settlement traverses retained profile order; reports use root-ID order; each
operation list uses original operation-ID order (`composition_root.go:87–108`,
`composition_inspect.go:26–125`). Decode the root before publication; a retained
publication can stop preparation. Otherwise apply the original freeze/closure
precedence (`invocation_outcome.go:340–349`). Only absence of both freeze and
closure at the same cut permits the early members-pending path. Failed child
publication never erases unresolved descendants or remote consequences.

Bulk basis inputs live under one owner lock/cut. Public SettleRoot still
rechecks in its write transaction. A newly ready concurrent root may require a
subsequent turn; progress must come from the actual settlement result, not the
stale candidate list. Keep cancellation checks in long in-memory traversals.
No error may fabricate a successful or partial report.

Batch root-to-intent membership and charge relationships using full identities;
retain original SQL charge numeric/overflow semantics. Do not eagerly validate
unreached malformed membership/result rows. Existing literal per-query injected
failure tests (`composition_inspect_test.go:307+`) must be translated to the
actual bulk source boundaries: removed SQL calls cannot preserve arbitrary I/O
fault timing, but missing/corrupt evidence and cancellation must still fail.
All roots, charges, pending dependencies, uncertainty, published outcomes and
nil/empty distinctions remain in reports. Returned nested objects are detached.

## Continuation evidence and authority

The sole permitted new API is an owner read, not approval or independent audit.
Return the original accepted approval's historical basis first, even after a
late result (`continuation_history.go:30–35`). Otherwise find exactly one
allowed paid adapter operation for the declared predecessor/effect and original
generation; ambiguity fails closed (`continuation_history.go:45–65`). Read the
authentic prepared digest, exact dispatch admission and optional retained result
digest; absence never proves remote nonapplication. A malformed referenced row
is an error. Use a coherent read transaction and return detached scalar values.

ApproveContinuation's existing write comparison remains authoritative after the
read (`continuation.go:218–261`); a result arriving between read and approval
must produce ErrStale. Do not advance epochs, spend, authorize or dispatch in
the reader. Inspect/Replay/Open retain their independent full-history validation.

## Why call-local batching

The selected first slice changes redundant SQL-per-record probes and unrelated
matching; every semantic index expires before unlocking or mutation. Owner
lifetime caches would additionally need complete invalidations for input,
ordinary Commit, Retry/retirement, invocation, ACK, reservation, admission,
result/receipt/conflict and settlement, plus rollback/cancellation/reopen.
Several retention paths commit directly instead of through commitOwnedTx
(`request_authority_admission.go:256–260`, `dispatch_admission.go:57–58`,
`receipt_read.go:217–228,285–301`). A revision-triggered full rebuild preserves
the global sweep; a rowid watermark misses updates. No persistent cache or new
scheduler is authorized for this slice.

The same call-local inputs may still scan growing retained history. B complete
reports containing B roots, plus B−1 requested full snapshots and the strict
returned-cut proof, impose quadratic output/materialization. A SQL count
reduction does not establish linear row/VM/Go work or whole-path latency.

## Falsifiers to retain

Compare choices, errors, complete returned reports, contacts and durable evidence
at mutation cuts: absent last, wrong/retired generation, consumed/wrong-run input,
first-reached corrupt projection/program/binding, missing rows, optional old
tables, late evidence/conflicts, failed descendants, closure after success
freeze, publication before late result, rollback/cancellation, foreign Stores,
wrong namespace/generation and changed basis before approval. Fresh-owner,
actual SIGKILL/reap and independent old-reader/Replay/inspection controls remain
required. No new dependency/table/index/schema or BoundedSeries/RunReport change.

The retained composition profile already owns immutable canonical root order
(`composition_store.go:204`). A private immutable ordered-root-ID projection
constructed with that validated profile and rebuilt by Open is a permissible
provisioning index if needed; it must expose no mutable aliases and must not
cache readiness, authorization or a changing durable prefix.
