# Phase 1G bounded composition contract — review draft

Baseline: `80dead19846d9b034b4457a9436e43acb81da3df` (merged main, inspected 2026-10-26). The 189 Phase 1F contributing source files retain source identity `2dacb402e50ee938d6184c14b40ba469fdcb8d91d9c30eac178b0dcb2a0fd3f8`. Historical records remain untouched.

## Decision and alternatives

A fixed reusable participant pool would retain constant subjects, but today's evidence helper is a one-shot composition of two programs. Safely reusing both requires invocation-scoped input, residual state, reply correlation, and either suspended frames or separate depth pools when parents wait. Repeating fixed helper graphs or unrolling one participant per possible item fails the request. This alternative remains appropriate for future long-lived services; it is not implemented here.

Implement bounded durable invocation of a retained **component template**, which may contain ordinary programs and internal ports. A template is immutable program artifacts, static relative wiring, a public request/result port, input defaults, and explicit permitted child templates/capabilities. Invocation supplies typed request data and a fresh identity. Scope binding resolves relative peers to the created executions and caller; it cannot grant arbitrary targets or effects. Artifact bytes and binding semantics remain locally available. The program and topology are not recompiled by Go or reprovisioned by the author after work becomes known.

This is genuinely data-dependent creation of logical executions within explicit limits. It is not unrestricted spawning, recursion, distributed ownership, or a new scheduler. The same `Store.Run` schedules all ready ordinary executions; its existing bounded adapter dispatcher controls physical external concurrency. Suspended parents own no dispatcher slots. Queued logical work may exceed physical capacity. Local reductions remain serialized finite commits.

## Ordinary authoring basis

Add bounded typed record lists (including bounded nesting where needed), value access/append and snapshot iteration. A runtime loop carries its finite cursor and collection snapshot durably. Each reduction is finite; iteration may span ordinary Receive/Request boundaries. Preserve old scalar representations and old artifact/history meaning.

Add an ordinary invocation effect and correlated invocation-result receive. Invocation intention commits through the same execution/intent protocol; separately authorized admission instantiates a permitted retained template and records its original accepted/rejected outcome. The caller receives the component's public response or a defined terminal failure/rejection. Retransmission of an existing operation cannot create another invocation. Source identity, generation, revision, operation and selected template are checked. Child policy is ordinary code, with no dispatch on component/role names.

A public reusable collection component derives scatter/collect behavior from these steps. Its author defines the item component once; the builder does not unroll items into helpers or fixed result slots. Input is an ordered typed list of records containing an explicit item ID and typed item input. Result is an ordered typed list containing the same IDs and outcomes. Empty input returns an empty successful collection. Duplicate payloads with different IDs remain distinct. Duplicate item IDs reject the whole work set. Over-limit input rejects before admitting any child work. Invalid UTF-8, nested bounds and arithmetic overflow reject explicitly.

The component's request/result schema is identical when direct or nested. Namespaces, invocation IDs, item IDs and worker/run identities are distinct. A new invocation begins fresh local state under a newly authorized operation; reopening an existing invocation preserves its state, budget, operations and observations.

## Durable boundaries

1. Transport admission retains the exact request and original run binding. Application acceptance is the ordinary commit consuming that validated request into the work-set projection. This is the durable work-set point: IDs, values, ordering and bounds are retained, and no planner reruns on recovery.
2. Ordinary iteration commits each invocation intent and pending correlation into the parent projection/history. No child can appear before its source intent commits.
3. Source authorization is retained separately. Invocation admission atomically retains its descriptor/result, child origins and initial executions, resolved bindings, permissions and root-budget charges. Repeating an operation returns this same result, including rejection. Cancellation before admission is revalidated against the scope closure rule below.
4. Children run ordinary programs and interact through their bound public ports. A public reply is retained and authenticated against the invocation endpoint/operation before the caller can consume it. Response recording and parent consumption are separate commits. Terminal failure cannot be reported as a successful item. No success is inferred from receipt absence.
5. Parent collection consumes each original operation result once, retaining item correspondence. Result order follows accepted input order, independent of completion order. Waiting for the next ordered result does not prevent siblings executing or recording their own results. Unknown required work leaves the aggregate pending.

All accepted invocation artifacts, inputs, bindings, results and budget charges are reconstructible. Replay validates creation origins and descriptors without creating runs, admitting work, calling adapters or planning. It has no dispatch registry/credentials. Live recovery may continue already eligible work and is labeled separately.

## Root budgets, closure and uncertainty

Each admitted top-level work obligation has one immutable resource envelope, inherited by all descendant invocations: total invocation admissions, total created program runs, maximum nesting depth, total external-effect authorizations and physical adapter concurrency. Component input list bounds additionally constrain each request. Root charges never reset on child invocation, nested calls, process death or reactivation. New effect authorization is charged once per existing intent, not per retry of the authorization API. Recovery configuration may not silently raise a persisted bound. Inspection exposes limits, charges, original IDs and unresolved evidence.

Qualification defaults: maximum input items 8; counts 0, 1, 3, 7, 8 and rejected 9; physical adapter capacity 1 and 2; maximum invocation depth 3 (root at 0); shared maximum 32 invocation admissions, 64 created program runs and 96 external effect authorizations. These are fixture/deployment bounds, not universal architecture limits. Test lower exhausted bounds separately. No provider spending is authorized by these work ceilings.

Cancellation is scoped closure, not group cancellation or compensation. CancelRun closes the addressed run; for a root obligation it also prevents further descendant invocation admissions and not-yet-authorized effects under that root. Already admitted descendants retain their execution records and may finish pure computation and deliver late evidence; already authorized external operations retain uncertainty and may settle. A closed parent never reopens or reports successful completion. Siblings and descendants do not silently become failed merely because their parent closes. RetireSubject remains distinct and prevents future generations. This profile implements no automatic descendant cancellation, reassociation or cleanup custodian.

Cancellation/admission and cancellation/completion races have serialized outcomes: admission first retains children/charges; closure first retains rejection and no children. Completion recorded first remains evidence even if its consumption is then prevented by closure; closure first may still retain authenticated late evidence bound to the old operation, without satisfying newer work. Scope checks must not erase existing source authorization. Tests distinguish authorized external work from newly requested work.

Unknown adapter outcomes occupy the original unresolved operation. They are neither automatically repeated nor treated as completed; independent eligible work continues within remaining dispatcher capacity. A fresh invocation is different work and must not reuse or replace an uncertain business operation automatically. There is no new business deduplication/compensation claim.

## Compatibility and exclusions

New artifact/binding/collection semantics and any persistence schema change are explicitly versioned. Existing scalar programs and histories retain their IDs and interpretation. Existing version-3 stores remain supported unchanged, or new functionality is rejected explicitly for them; no silent conversion or general migration. Unsupported versions and inconsistent descriptors fail closed. The implementation must document the exact version matrix before claiming compatibility.

No live provider calls in this increment without separately applicable model/credential/spend authorization. Controlled providers qualify execution, not answer quality. Do not alter Phase 1F prompts, corpus, evaluation grades or study ledger. No multi-host execution, ownership transfer, general capability delegation, unrestricted recursion, public SDK stability or UI.

## Acceptance and independent checks

Use an independently calculated deterministic work tree before evidence leaves. Exercise the same artifacts with different post-initialization work sets, more work than physical capacity, out-of-order completion, identical payload/different IDs, direct and nested composition under saturation. Reuse the existing two-program evidence helper as a leaf template through public APIs.

Actual subprocess SIGKILL tests synchronize at work acceptance, partial completion and recorded response before consumption. Add causal provider barriers for held work and independent progress; do not use sleeps to prove ordering. Retain late/stale/duplicate result attempts, unknown outcomes, closure races and boundary evidence. Independent monitors check identity, ordering, budget conservation, no uncommitted dispatch/creation, no premature success, and replay with no dispatch. Faulty variants must reach the relevant boundary and be detected by unchanged monitors.

Measure author code, constant template/provisioning size, admitted work/run/record growth, physical active count, latency and recovery outcomes. Retain original failures and exact input/source identities. Successful traces are reachability evidence; finite tests are not unbounded safety/liveness proofs.
