# Independent Phase 1G feasibility assessment

Assessment date: 2026-10-26. Inspected checkout HEAD: `80dead19846d9b034b4457a9436e43acb81da3df`. This is an independent, read-only architecture assessment of the supplied Phase 1G request, before implementation or contract review. No tests, live provider calls, credentials, or product edits were used. The report does not claim qualification.

## Recommendation

Choose **B: bounded durable invocation of retained component templates**, using the existing Store, pure reducer, ordinary-message protocol, single Run owner, adapter dispatch and read-only replay. A template must represent an ordinary composite, including its internal programs and wiring, rather than just a single Program. Keep traversal, item selection, fan-out, collection and result policy in authored programs.

Physical concurrency remains an executor limit; logical child invocation is separately bounded by durable total-work and nesting limits. Waiting parent executions consume no physical dispatch permit. This is bounded dynamic creation of logical executions from provisioned definitions, not unrestricted recursion or runtime code generation.

A fixed participant pool is feasible in principle, but is not the smallest complete implementation against *this* baseline and all acceptance cases. It needs either reentrant invocation frames, parked continuations and per-invocation routes, or explicitly partitioned capacity for every nesting level. Once it correctly reuses the two-execution evidence helper and handles saturated nested calls, it has recreated most of the child-execution mechanism while additionally managing worker-to-invocation binding.

## Baseline evidence and missing capabilities

- `schema/schema.go` defines closed scalar-only Value and Record. There is no native list, record-valued expression or collection bound. Scalar Value is directly comparable; extending it with slices affects equality, typed defaults, validation, copying and canonical serialization, not merely one enum.
- `author/author.go` has named Set/Add/IfEqual, forward-only Goto, author-time Repeat, ports and correlated adapter Request/Await. `author/compile.go:124` expands Repeat into instructions and its validation assumes a forward-only control-flow graph.
- `author/application.go` provisions a fixed map of subjects and generation-one runs. `author/compile.go:218` emits literal peer/obligation/generation for ports. `internal/executor/author_metadata.go:153` rejects ordinary author messages with nonempty Operation or source generation other than one. New invocation correlation must therefore be a versioned change, not an assumption that ordinary ports already correlate successive calls.
- `examples/evidence/evidence.go:61` builds an assistant and retrieval run. Both are one-shot programs; the assistant sends a public result and ends. Its corpus/model/lookup logic is already ordinary policy and should remain so.
- `internal/executor/lifecycle.go:146` Retry only applies to failed/abandoned/eligible-cancelled runs and resets to the retained root initial state. It is not a successful next invocation with new inputs.
- `internal/executor/store.go:634` already atomically commits selected observation consumption, successor, history and intents. Admission, adapter authorization/attempt/result/settlement and ordinary consumption are already distinct durable stages.
- `internal/executor/runner.go:49` interleaves internal delivery, ready ordinary execution and bounded external dispatch. MaxInFlight limits active external workers, not open logical runs. It already provides the essential scheduling shape for nonblocking parents.
- `internal/executor/replay.go:79` reconstructs each retained run and validates cross-ledger evidence. Child origins, scoped grants and shared budgets need explicit reconciliation there. Open/Inspect must validate the corresponding durable relationships too.
- Phase 1E contract preserves known results, conservative unknown provider outcomes and cancellation-independent accounting of received completions. Phase 1F contract explicitly uses static helper expansion and per-participant ceilings, not a shared hierarchy-wide budget. `phase1f/authoring-obstacles.md` records e1–e4 fields and separately provisioned helpers as the earlier bounded solution.

## Concrete old-basis counterexamples

These are source-derived counterexamples to retain as executable public-consumer probes during implementation; they were not executed in this assessment.

1. **Collection:** after initialization deliver `[{id:"a",value:2},{id:"b",value:2},{id:"c",value:9}]` and later, in a separate store using the identical retained application, deliver seven differently valued items. No existing Field kind represents this collection. Seven named scalar slots change the public shape and retain fixed-slot authoring; JSON in a string violates the requested typed contract.
2. **Iteration:** one retained program must visit each member of that received list. Repeat requires an author-time Go int and expands its body; Goto cannot implement the runtime loop. Repeating the maximum with manual slot selection only conceals finite expansion.
3. **Fresh invocation:** with capacity one, execute two evidence requests sequentially. The helper and retrieval worker end after the first. Retry cannot restart success or install the second request as fresh invocation input; resetting projection or reusing effect IDs would silently change history.
4. **Scope and correlation:** instantiate the same helper definition twice with identical questions. Existing literal peers point to the same statically named retrieval run. Existing ordinary Receive selects source/generation/obligation with empty Operation, so a reused participant needs a new authenticated invocation binding; payload equality is not an identity.
5. **Saturated nesting:** one active parent invokes a child of the same component while a one-slot pool is full. Holding the slot until parent completion prevents the child from running. Increasing capacity in response conceals the failure. Parking the parent requires retained invocation state separate from the worker.
6. **Shared budget:** a root allows three external requests; two nested helpers each start with their own default model/lookup counters. Existing local counters can admit more than three root-wide requests, and RunOptions limits reset on another Run call. Neither is a durable root budget.

The first two require generic collection/iteration semantics. The third and fourth require fresh invocation and scoped endpoint semantics. The last two require resource accounting and ownership enforcement, not application-specific workflow instructions.

## Approach comparison

| Required property | A: reusable participant pool | B: retained child executions |
| --- | --- | --- |
| Runtime work count and typed values | Needs collection/iteration addition | Same addition |
| Existing two-run evidence helper | Needs reset/reentrancy of both runs, fresh invocation state and routes | Instantiate its retained two-run template for each admitted item |
| Identical inputs and stale results | Needs durable worker lease plus invocation identity and binding history | Invocation identity owns stable child subjects and result obligation |
| Capacity-one nested call | Needs parked frames or separate depth capacity; naive pool deadlocks | Waiting parent is an ordinary blocked run; descendant uses existing dispatch slot |
| Unknown item plus other work | Unknown must release worker without losing parked obligation | Unknown child stays open; unrelated runs remain eligible |
| Recovery | Recover queue, worker assignment, invocation/frame states and evidence | Recover invocation origin, ordinary runs and existing evidence |
| Provisioning cost | Fixed pool per definition/depth, with bounded parked work | Fixed program/template catalogue; runs proportional to admitted work |
| Scope risk | New reusable-execution abstraction in addition to queueing | New durable origin/binding with existing run lifecycle |

B is a smaller *semantic* extension here; this is not a claim that its implementation is trivial. In particular, a “spawn one program” implementation omitting composite scope binding does not satisfy helper reuse.

## Suggested narrow public primitives

Names are illustrative contract candidates, not declarations that these APIs already exist.

1. **Bounded record list:** a schema field describing a list of a declared record schema with MaxItems and total encoded-byte limits. Initially allow scalar record elements; a deterministic work tree can use scalar depth/path fields and construct bounded child lists. Do not introduce arbitrary maps, dynamic fields or unbounded recursive schema types. Supply typed list construction, length, indexed record binding and append. Retain canonical native data, deep-copy mutable values, and keep empty lists distinct and valid. Counterexample: item list in case 1.
2. **Structured ForEach:** iterate a retained list with a persisted cursor, typed element bindings and compile-time finite maximum. It yields through ordinary durable steps, with definite-assignment validation appropriate to a bounded loop. Its body can send, invoke, receive, append or branch. It must not itself create children, select a worker, aggregate results or implement Map policy. Counterexample: case 2. Prefer this over exposing numeric program counters or unrestricted backward Goto.
3. **Retained ComponentTemplate:** immutable catalogue of ordinary artifacts, initial constants, internal relative endpoints and one declared public request/result boundary. Compile once. Invocation supplies only typed input and an authorized caller/result binding; it cannot supply executable code, grants or arbitrary artifact bytes. Scoped endpoints distinguish internal-relative peers from the caller boundary. The compiler/runtime must make this distinction explicit in new metadata; do not perform string-prefix rewriting of arbitrary user values.
4. **Invoke and correlated Receive:** an ordinary authored Invoke emits a generic invocation intent naming an allowlisted template and item identity/input; it returns a typed invocation handle. The existing owner processes that intent into one durable child origin, scoped runs/grants and request delivery. Receive uses the handle and public result schema, validating originating invocation, source execution, result obligation and result identity. No private child-state read. Generic durable admission rejection should be observable as a correlated non-success outcome. Counterexamples: cases 3–4.
5. **Root resource scope:** an immutable root budget plus durable shared accounting attached to each child origin. Invocation admission charges root total work and checks depth. External intent commitment charges the same root external-effect ledger atomically; participant counters may remain stricter application limits. No new model/tool workflow opcode is needed. Counterexample: case 6.

The public collection component is an ordinary program: receive list, initialize an empty output list, invoke each item, then consume each correlated handle in input order and append its outcome. Sending all bounded child requests before waiting enables independence. Physical execution remains bounded by the existing runner. This satisfies the acceptance capacity requirement if capacity is explicitly the simultaneous physical adapter/external work limit: the reducer itself is already serialized. It does not satisfy a different promise that only C child invocations may be logically open. Do not advertise that latter promise. Report admitted, pending, active-physical and completed counts independently, and show queued external intents at capacity one. There is no need for an AwaitAny primitive merely to prove out-of-order child completion: later results can be durably recorded while the parent awaits an earlier input position.

Existing evidence policy should be factored through its shared authoring definitions into a scoped template, with the legacy New/NewExcluding factories retained unchanged. Do not rewrite its planning/retrieval/synthesis logic or wrap it in an adapter-owned loop. Direct and nested collection calls use exactly the same public schema and template.

## Admission, ownership and bounds to freeze

**Work-set acceptance:** define it as the ordinary Receive commit consuming the validated request and retaining its complete bounded list, item identifiers and input ordering. Delivery admission before that commit is pending input, not accepted work. The request observation and resulting history contain the complete canonical list. No planner is consulted again after acceptance.

Assign item identity from the parent invocation plus accepted-list index, optionally retaining an application key as data. Equal payloads are distinct. If application keys are accepted as identities, reject duplicate keys atomically rather than silently deduplicating. Preserve the immutable accepted list separately from mutable cursors/output accumulation.

**Child admission:** Invoke's committed intent is a pending child-admission request. Its idempotent handling must atomically record the admission decision, deterministic invocation ID, template/artifact refs, root scope, depth, result correlation, child subjects/runs/grants and initial request. Unique origin/intent constraints make a repeated delivery reuse that decision. A process death before the transaction produces none; after it produces all. A work item whose child admission exceeds remaining root budget receives a defined rejected/budget outcome and is not silently dropped. Thus accepting a list promises a correlated outcome for every item, not unlimited execution capacity.

**Completion:** child execution results are delivered through the declared public port. A success result must not become a successful parent result while required descendants remain pending/unknown. Enforce this for the new invocation scope, in addition to ordinary program collection policy, and define when a sent result becomes publishable. Suggested rule: retain the result candidate but publish successful scope completion only after required descendants terminate successfully or are explicitly handled under the declared result policy. A failed/invalid leaf result remains a correlated outcome for the caller to classify.

**Budgets:** immutable per-list MaxItems; root total admitted items/invocations; root maximum nesting depth; bounded total retained bytes; global physical MaxInFlight; durable root external-intent count. Define whether template-internal runs charge total work separately; report both item count and run count. All descendant admissions share one root ledger and atomic debit. Do not reset counts on Open/Run. A blocked parent and an unknown logical item consume retained-work allowance but no physical worker permit. No speculative slot reservation may cause a waiting parent to retain capacity required by its child.

**Unknown:** once an adapter attempt has no known result, preserve that exact unresolved identity and spent reservation. Do not repeat or declare success. The runner can schedule unrelated eligible work after the physical attempt has ended. At physical capacity one, a currently held external operation necessarily prevents another external launch; demonstrate independent ordinary progress in that case and held-plus-independent external progress at capacity two.

**Cancellation:** distinguish physical runner cancellation from logical invocation cancellation. The smallest explicit policy is **close admission and drain already admitted descendants**, with no automatic recursive CancelRun. Serialize logical cancellation and child admission in the owner transaction order. If cancellation wins, reject later admission; if admission wins, that child remains admitted. Record a root/scope closure that blocks future child admissions throughout that scope. Already admitted descendants may continue ordinary steps and external effects under the same remaining durable budgets; a later attempted nested admission receives a correlated cancelled/rejected outcome. The cancelled parent cannot publish success, and descendant replies are retained late evidence rather than reopening it. Already authorized physical work and recorded/late results retain their original correlations; cancellation never refunds uncertain spent work or rebinds a result to a later invocation. This policy can leave an unknown descendant unresolved indefinitely, which must remain visible. If the desired product policy instead stops all admitted descendants, that requires an explicit scoped cancellation closure and dispatch/admission fencing; do not imply that stronger behavior without implementing and qualifying it.

**Replay:** extend existing ledger validation to reconstruct child admissions, origin bindings, scope closure and root budget debits, including reverse links and orphan detection. Read-only replay validates retained runs and results; it must not materialize a missing child, call the template builder/planner, dispatch an adapter or read the original corpus. Use current credential-free replay and existing process boundary/independent evidence tools.

## Versioning and compatibility

Current versions are store schema 3, external key 2, base semantics `alan/1d/v1`, author semantics `alan/1e/v1`, compiler `alan/author/v1`.

Allocate store schema 4 (the baseline is schema 3, not schema 1) for invocation origins/budget/scope ledgers and a new author semantic/compiler version for typed lists, iteration and scoped ports (for example `alan/1g/v1` and `alan/author/v2`). Retain exact legacy program and schema canonical encodings: new zero-valued fields need omission/version-aware encoding so scalar artifact IDs do not drift accidentally. New list equality/default rules must not silently change legacy scalar reduction.

Either explicitly support schema-3 replay/Open through the unchanged path, or reject schema 3 with ErrVersion before activation and document reproduction using its retained source. The request permits clear rejection; do not build a general migration framework. Preserve old author semantics in new stores where feasible, and reject unsupported semantic/compiler combinations precisely. Keep namespace/external key version 2 unless its actual encoding must change; new invocation subjects already yield fresh local intent identities.

Qualification must include legacy artifact hashes, an original historical store under its stated support/rejection rule, unknown future schema/semantic rejection, and dispatch-disabled replay of child histories. Historical recordings remain facts under their original source, including prior adapter classifications.

## Contract review focus and acceptance mapping

- A: one artifact identity, delayed list inputs for 0/1/3/7/max; max+1 rejected before accepted work; empty returns typed empty success.
- B/D: physical capacity 1 and 2, more logical items than slots, nested request with saturated active capacity, no parent-held permit.
- C/G: identical payloads with distinct item IDs, reverse completions, duplicate transport/result and a result from a prior invocation while another is pending; independent expected ID-to-value relation.
- E/H: acknowledged held effect and independent progress; unknown held identity stays unresolved/spent and receives no automatic retry.
- F: actual process death at accepted-list commit, partial completion, and result-recorded-before-consume; compare accepted inputs, origins, artifacts, results and budget before/after.
- I: cancellation before and after child admission, completion concurrent with cancellation, known-unsettled and late evidence after scope closure.
- J: offline replay over the exact retained store with disabled dispatch; immutable table comparison and independently logged zero new contacts.

Retain original failing probes and inject negative controls for payload-based deduplication, wrong result handle, budget reset, parent-held capacity, missing child admission, replay-triggered creation and late-result reopening. Reuse `internal/boundary`, `internal/evidencetest`, `internal/investigationtest` and the existing phase evidence tooling. Independent deterministic tree expected outputs and correlation checks are necessary because Run and Replay can agree on the same wrong interpreter.

Primary architectural risks are scoped composite wiring, ownership closure during concurrent admission, and budget debit/replay agreement. Freeze those details in the independent contract review before adding semantics. No new scheduler, provider catalogue, migration framework or privileged application agent is justified.
