# Phase 1C ordinary supervision contract

Status: provisional bounded research contract, frozen before implementation on 25 October 2026. It extends only the ordinary representations missing from the [Phase 1](../phase1/01-operational-spec.md) and [Phase 1B](../phase1b/01-composition-contract.md) baselines. It does not grant supervision a privileged execution type.

## Public boundary and identities

Any ordinary program may observe a child if its configured observation permission allows it. That observation does not grant permission to mutate the child's lifecycle. Static lifecycle grants separately identify requester, target, allowed operation, and ownership revision. An ordinary program can commit a request without a valid grant; authority rejection is a recorded protocol result that the program can consume and escalate.

Keep these identities separate: logical subject `a`; execution `(a,g)`; physical activation `(a,g,e)`; ownership principal/revision `(owner,w)`; logical supervisory obligation; and immutable lifecycle operation ID derived from a committed requester intent. The obligation includes a stable declared key and its explicit ordinary initialization context. Worker retry preserves that obligation and its budget. Physical reactivation preserves it too. A new obligation requires an explicit ordinary initialization decision; a new child generation is not such a decision.

A request includes requester subject/generation, obligation identity, operation ID, target subject, expected target generation, expected ownership revision, and operation. Retransmission preserves every field. Changing fields under an existing identity is a conflict, not a second request. A subsequent new-generation operation needs a separately committed decision with a new identity.

## Authenticated observations

Terminal child observations retain subject, exact generation, frozen outcome identity, classification, and authorized observation route. Supported classifications distinguish committed deterministic failure, successful completion, authorized abandonment, and run cancellation. Retirement remains a permanent authority state, distinct from a run's cancelled terminal outcome.

Suspicion is an explicitly environmental observation with subject/generation/activation provenance; it is not a terminal outcome or proof of failure. Physical Interrupt only discards speculative evaluation. Reactivate changes an epoch and preserves the committed logical run. Neither produces application failure. A program may wait, report suspicion, or request a permitted reactivation under an explicitly declared policy; the selected retry policy does not request a fresh generation solely on suspicion.

Lifecycle operation results retain operation and obligation identities, requester execution, target expected generation/revision, accepted/rejected classification, reason, and committed successor generation if any. Authentication comes from modeled runtime provenance and retained decisions, not matching tags or claimed sender strings. Ordinary message, external receipt, terminal outcome, suspicion, and lifecycle result are distinguishable.

An old observation may remain in an inbox or arrive late. It cannot satisfy a new generation's pending obligation, change a committed decision, or reopen a closed execution. A result admitted before closure may remain unconsumed afterward. Delivery and consumption are distinct, and neither is promised eventually.

## Lifecycle request protocol

1. **Local intent commitment:** ordinary bounded reduction commits the immutable request and its pending reference with the local projection. An interrupted uncommitted proposal has no request. A committed intent survives physical interruption.
2. **Runtime receipt/authorization:** a separate event validates the committed source and dispatch permission and records receipt/authorization status. It cannot select a recovery action or advance the supervisor program. An unauthorized lifecycle request must obtain a retained rejection outcome rather than silently acquire authority.
3. **Lifecycle decision:** a separate atomic authority operation validates request identity, authority, expected generation/revision, and operation eligibility against current state. It atomically commits the lifecycle transition and immutable operation result, or an immutable rejection. A cancellation may interleave between authorization and decision; no stale permit bypasses decision-time validity.
4. **Outcome admission:** a separate event exposes the retained result to its requester under stable binding and provenance. Loss/delay of this acknowledgment does not erase the decision. Resending the identical operation returns the same original result, including rejection, without re-execution.
5. **Ordinary consumption:** a selected Receive matches the pending operation, requester generation, obligation and expected target; the next local commit records consumption and the program's state change. The program then branches to continue, retry, report, or escalate.

The retained operation ledger is separate from the execution journal and external-resource world. The lifecycle decision and target authority update are modeled atomic; authorizing a request, recording/admitting its result, and consuming it are not one combined helper. No distributed atomicity implementation follows from this assumption. Already authorized requests may reach a decision after requester closure, subject to target validation; the closed requester does not resume.

The existing Retry eligibility remains: failed or abandoned runs may retry; cancelled runs depend on the explicit `RetryCancelled` configuration; retired subjects cannot retry. CancelRun addresses its specified run. Late cancellation of an already failed run preserves failure and does not become a permanent stop-retry latch. RetireSubject prevents successor creation. Explore both cancellation policies rather than asserting a universal choice.

## Selected ordinary recovery policy

An ordinary program initializes a declared obligation with worker identity/expected generation and a retry limit. It retains classification, consumed terminal identity, attempts used, pending lifecycle operation, correlated result, and final outcome as bounded ordinary data. Conditions and finite updates, not a runtime policy handler, choose these steps:

- Success from the expected child generation permits the declared success outcome.
- A committed failure, or authorized abandonment in the explicitly selected abandonment variant, permits a new Retry intent only when the obligation has remaining budget and no unresolved operation for the same decision.
- One committed recovery decision consumes one budget unit, whether its later lifecycle outcome accepts or rejects. A retransmission consumes none. A worker successor and a physical supervisor reactivation do not reset the count.
- An accepted Retry result updates expected child generation to the recorded successor. The supervisor waits for that generation's outcome before making another recovery decision.
- Rejected Retry reports/escalates failure in the selected policy. More elaborate policies that inspect reason and choose another allowed action are alternatives, not silently implied behavior.
- Exhaustion reports terminal failure/escalation. A duplicate or stale terminal notification cannot reset the counter, count as a new failure, or produce another operation for the same obligation/child-generation decision.
- Suspicion alone never satisfies the failure branch. Run cancellation is classified separately and follows the fixture's declared policy. Missing external receipts never imply that business work failed.

The limit is 0–2, chosen in fixture data. Faulty ordinary programs may omit a budget branch or misclassify an observation; the runtime must not secretly enforce the policy contract. Independent monitors must detect the fault.

## Nested supervision

The worker, inner supervisor, and outer supervisor use the same interpreter, ordinary projection, commit protocol, and lifecycle effects. The inner observes/retries the worker and can exhaust its limit and fail. The outer observes only the inner's authenticated public terminal outcome, classifies it, and applies its own declared limit/action. It cannot read the inner's retry counter, program state, or worker journal.

At least one complete trace must show worker failure, an inner recovery request with an accepted outcome, another worker failure, inner exhaustion, and the outer's ordinary recovery or escalation decision. Search the outer/inner boundary with duplicate outcomes, lost lifecycle acknowledgment, and interruption. Static provisioned participants do not imply ancestry-derived rights. Public names can be renamed without changing runtime behavior.

## External effects and recovery

Retain the existing separate external world and application/receipt events. An authorized worker mutation may apply before or after worker failure, with no recorded receipt. Computation retry does not undo it, prove it failed, or safely repeat it. New execution-scoped intent IDs do not silently inherit a business identity. The selected ambiguity fixture issues distinct explicit effect identities across generations and exposes any repeated mutation; it claims no business-level exactly-once guarantee. A late receipt settles its original effect only.

In that fixture, the ordinary program explicitly labels the operation `business:order-7` in its immutable effect Tag. This opaque business reference stays unchanged across generations while execution intent IDs differ. The runtime neither replaces that business reference with a generation ID nor deduplicates by it. The program's repeated Emit can therefore produce two mutations for the same declared business operation. That behavior is a retained limit/counterexample to safe automatic business retry, not a successful business-recovery contract. A policy that promises business safety must reconcile or explicitly authorize repetition under an endpoint-specific protocol; this phase does not supply one.

## Minimality classification

| Candidate capability | Classification and reason |
| --- | --- |
| Terminal/suspicion metadata admitted to ordinary Receive | Existing runtime outcome/observation with new ordinary representation; baseline `Delivered` was not an inbox |
| Lifecycle Emit descriptor and retained decision/result protocol | Existing lifecycle operations with new ordinary effect representation; necessary to keep programs from direct mutation |
| Finite field load/store, arithmetic, comparison, correlation slots | Computational representation needed if existing two-register operands cannot express the declared policy; add only bounded generic operations, not a supervision policy instruction |
| Retry loop, duplicate tracking, escalation, nested supervision | Derived ordinary programs/protocols; never role-specific runtime behavior |
| Static observation permissions and lifecycle grants | Explicit fixture authority assumptions; no general delegation or transfer |
| Contract facts, history-derived counts, witness flags, mutants | Test-only verification; unreadable by reducer/lifecycle handlers |

Concrete added operands and state fields must be documented after implementation. The selected transition language remains closed and finite. No arbitrary callback is used to assume determinism or finite turns.

The selected additive local representation is eight bounded integer fields, two name fields (child and explicit obligation identity), a pending immutable operation reference, last observed metadata, and at most eight consumed logical keys. It is optional; legacy projections and fixtures retain their representation. Set/Branch gain finite local field access and arithmetic, and an effect can carry a declared local scalar update in the same atomic intent commit. That update provides a generic budget reservation mechanism; the program chooses the field, delta, branch, and limit. It does not ask the runtime to apply a retry policy. Final concrete operands and scalar encodings are recorded in the operational appendix with the executable fixture definitions.

## Concrete operational representation

`projection.Local` is optional `localData`: `Values[8]`, `Names[2]`, `Pending`, `Last`, and bounded `Seen`. The supplied programs assign the value slots to expected child generation, classification, spent count, limit, accepted-result flag, successor generation, ownership revision, and scratch. Name slots hold child and obligation. These names describe the fixture's data convention; ordinary Set/Branch operations address finite slots rather than agent roles.

| Instruction form | One finite local step |
| --- | --- |
| Set/local, Set/copy, Set/add | Set a bounded scalar, copy another scalar, or add a declared bounded value |
| Set/name | Store a static name operand |
| Set/identity | Explicitly initialize an obligation identity from this run, declared key, and committed initialization position |
| Branch/local | Compare a scalar with a literal or another scalar and select the next program counter |
| Emit/lifecycle | Form the immutable request from local binding fields; save its intent ID as Pending; apply the explicitly declared `Slot`/`Delta` update atomically with the intent |
| Receive/observation | Consume an authenticated terminal or suspicion observation for the expected child generation, reject an already consumed logical key, and project its classification |
| Receive/terminal | The terminal-only form of that observation projection |
| Receive/lifecycle | Match the full pending operation context; project acceptance/successor, clear Pending, and retain full result metadata in Last |

The classification convention is 1 success, 2 application failure, 3 abandonment, 4 run cancellation, 5 suspicion. The lifecycle acceptance flag is separate. The supplied program branches back to waiting on classification 5. It does not derive failure from missing delivery. A local field update cannot inspect another subject, authority table, external world, or oracle facts.

`lifecycleRequest` retains ID, Requester, RequesterG, Obligation, Target, ExpectedG, W, and Operation. The ID is the ordinary execution intent ID; the obligation and physical epoch are distinct. `lifecycleResult` retains the complete request, Accepted, Reason, and Successor. The runtime's `lifecycleLedger` retains Authorized, Results, and Notices maps. Ordinary code can access only admitted observations, not these mutable maps.

`AuthorizeLifecycle` records source dispatch authorization under the requester's current owner/revision and closure rules. `DecideLifecycle` first returns any retained result, then validates static target grants, current target revision/generation, and the existing lifecycle rule. Semantic rejection is a committed `Accepted=false` result, not a failed model step that loses its state. `AdmitLifecycleResult` separately admits the recorded result. `DeliverTerminal` exposes a frozen outcome to its configured observer when that observer is a modeled subject; existing external audit routes retain their earlier behavior. `Suspect` is an explicitly environmental stimulus with generation/epoch metadata, not a failure detector implementation.

Notification delivery IDs and logical notification keys differ. Service admission retains each delivery's first run binding; a different delayed delivery ID may reach the current run with old metadata. Correlation must therefore hold even when routing has admitted an old result to an eligible current receiver. Both rejected and accepted admissions remain retained. Closed programs do not consume late observations.

The supplied policy explicitly initializes a new obligation when a fresh supervisor run executes Set/identity. It does not carry a tree-wide business budget through a supervisor restart. Its local budget survives worker generations and its own physical reactivation. A root-wide budget spanning fresh inner obligations would need an explicit public protocol; no such guarantee is inferred here.

## Adversarial windows and expected behavior

| ID | Window | Expected contract behavior |
| --- | --- | --- |
| A | Duplicate committed worker failure | One logical failure decision and at most one new recovery operation for that child generation; transport duplication cannot spend budget twice |
| B | Retry commits; acknowledgment lost; identical request resent | One retained original result and at most its one authorized successor; include rejection retransmission too |
| C | Supervisor interrupted after intent commit | Reactivation retains pending request/count; replay dispatches nothing; retransmission uses the same operation |
| D | Retry versus external CancelRun/RetireSubject | Search both serialization orders and both run-cancellation retry policies; no new generation after retirement |
| E | Worker evaluation interrupted or suspected | Run stays active; no fresh retry solely from suspicion; later same-generation computation may commit |
| F | Repeated failures exhaust limit | Count persists across worker generations and supervisor epochs; duplicates cannot multiply/reset it; explicit escalation/terminal result |
| G | Old terminal/result/receipt after successor starts | Actual successor Prepare/Commit; old evidence cannot satisfy the current wait or rewrite a committed decision |
| H | Nested failure propagation with boundary faults | Outer consumes public inner outcome and executes its own policy; duplicate/delayed delivery and reactivation preserve identities/budgets |
| I | Observation allowed, lifecycle authority denied | Immutable authority rejection, no target mutation, ordinary rejection consumption/escalation |
| J | Authorized mutation with unknown receipt before recovery | Application remains visible in world; no false failure inference or automatic business identity reuse; late settlement only for old effect |

Each searched window discloses its exact fixture/program, real setup prefix, finite event alphabet, bounds and statistics. Some expected behaviors are reached witnesses within a search; report them separately from safety assertions. Unsupported combinations remain explicitly untested.

## Qualification and stopping point

The same independent monitors must detect removal of operation deduplication, failure correlation, budget persistence, lifecycle authority, and old-generation result isolation. At least one faulty ordinary supervisor must be caught with all runtime guards intact. Counterexamples are shortest within their disclosed reachable-seed suffix search, not globally minimal across all ALAN programs.

Assumptions include atomic local commit/authority decision, trusted observation provenance, static rights, immutable versions, complete bounded retention, and the existing external endpoint profile. Bounded safety and successful recursive traces do not prove unbounded recovery, liveness, equivalence, durability, dispatcher fencing, or distributed authority enforcement. Stop after reviewed research evidence and draft PR.
