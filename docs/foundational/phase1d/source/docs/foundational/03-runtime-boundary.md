# ALAN runtime boundary

Status: research candidate, 24 October 2026. The runtime supplies mechanisms whose guarantees ordinary agent policy cannot create merely by requesting them. This document proposes contracts for those mechanisms without selecting storage, transport, infrastructure, or an implementation language.

## Computation and enforcement

The deterministic transition in [Foundations](01-foundations.md) returns successor state and a finite collection of effect intents. It performs no network I/O, reads no ambient clock, and does not inspect physical worker identity. An effect result is a later observation. A language model invocation is an external effect even when a provider offers a seed or temperature zero; replay consumes the recorded result.

| Agent computation decides | Runtime enforces |
| --- | --- |
| Which work to request and how to interpret a reply | Validated intent admission, dispatch, and correlated completion |
| Which messages to send and which capability to pass | Capability scope, inbox admission, durable message identity, and routing |
| Whether to retry, replace, cancel, or escalate | Current lifecycle authority, generation checks, and serialized operation outcomes |
| How to combine results or react to failure | Independent scheduling and durable observation ordering |
| Whether application state is compatible with an upgrade | Version binding, safe rejection of incompatible replay, and recorded migration boundaries |

No runtime operation is named `Plan`, `Control`, `Worker`, or `Supervisor`. Those are program roles. Runtime authority still exists: eliminating privileged orchestrator types does not eliminate the trusted substrate that validates capabilities and commits state.

## Candidate runtime state

For each logical agent `a`, keep a conceptually authoritative record:

```text
R[a] = (owner, ownershipRevision w, currentGeneration g,
        activationEpoch e, admissionStatus, cancellationTombstone)
```

For each retained execution `x = (a, g)`, keep a version binding `v`, sequence `n`, state `X`, run status, admitted inbox, pending intent ledger, and committed observations. Run status distinguishes active, succeeded, failed, abandoned, and cancelled. Ending one run does not itself retire the logical subject; the cancellation tombstone does. Old-run evidence and settlement obligations remain available within the declared retention horizon. These are semantic records; they need not be separate database tables.

The candidate allows one current run per logical agent and one serialized state history per run. Multiple physical workers may exist, but only authorized commits are accepted. Independently advancing concurrent work uses distinct logical agents; multiple local continuations share one serialized execution state. A stable service spanning several independently writable runs needs a routing facade or a different authority model.

Provenance records who created `a`; it is historical metadata. `owner` identifies the current lifecycle authority. Message capabilities identify permitted communication. None is inferred from either of the others. An owner can explicitly delegate selected operations, but a delegate's authority remains bound to the current ownership revision.

There is no assumed supervision tree. If a later design requires acyclic ownership, it must enforce that property during transfers. Even without a tree, recovery needs an explicit administrative principal for an unavailable owner; mutual ownership does not by itself guarantee recovery.

## Conceptual operations

The following operation families describe obligations, not a final API or a claim of minimal primitive count:

| Intent | Required parameters and result | Critical semantic point |
| --- | --- | --- |
| Send | Target capability, message ID, payload, correlation; admission or rejection | Durable inbox acceptance differs from handling and reply |
| Create | Program/version, initial input, requested owner, operation ID; new identity and scoped capabilities | Retrying the same creation operation returns the same logical creation or recorded rejection |
| Perform | Effect kind, immutable parameters, effect ID, adapter contract; result/error/unknown | Dispatch may be retried physically; external outcomes require a protocol |
| Observe | Subject and observation capability; correlated status/failure notification | Failure suspicion is not proof of physical death |
| Lifecycle | Subject, expected generation/revision, authorized action, operation ID; committed outcome | All conflicting operations are ordered at the subject's authority boundary |

Timer requests, random values, environmental reads, and model calls can be typed `Perform` effects. A durable timer stores its identity and deadline; replay observes a recorded firing rather than waiting again. Requesting a deadline does not imply punctual delivery during outages.

A capability is an unforgeable or authenticated authorization reference, implemented by an unspecified trusted mechanism. It includes target scope, permitted operations, delegation constraints, and any generation or revocation binding. Passing a capability can change communication topology. A send-only capability does not confer cancellation, transfer, state inspection, or arbitrary capability minting.

Revocation requires a checked authority reference or an equally explicit protocol. An immutable bearer token cannot be claimed to disappear from all peers merely because an owner changed. Messages already durably admitted under valid authority need a defined disposition after revocation; the candidate retains them unless cancellation or a protocol-specific rule rejects their handling.

## Messages and observation order

The baseline transport may delay, duplicate, or reorder deliveries. The runtime admits a message ID at most once to a target run's inbox within its declared deduplication retention scope. It must atomically retain the deduplication record and the accepted message, or an acknowledgment could refer to data that was lost. A sender retries an uncertain admission using the same ID.

Service-addressed messages resolve to a run at durable admission. That binding is recorded, so a retransmission cannot silently land in a newer run. Run-addressed messages always target their specified run. Handling admitted messages after a retry or replacement requires an explicit transfer/reconciliation policy; they do not automatically move to `g + 1`.

The candidate promises no ordering between independent senders and no implicit FIFO even for one sender. Protocols needing FIFO carry sequence numbers and specify gap handling. Each execution records the actual order of admitted observations that affect its state. Race winners, selective receive choices, and ordering among timer, message, and completion events must be recorded or derived deterministically from recorded data.

An admitted event can wait in the inbox while another continuation runs. A deterministic dispatcher or a recorded scheduling choice selects its consumer. Application-level broadcast is an explicit construction; one ordinary message is not implicitly consumed by every waiting continuation.

For liveness, eventual delivery requires durable retry, available storage, an accepting destination, and eventual connectivity. Queue quotas and backpressure can prevent admission. Deduplication records cannot be garbage-collected while delayed retries are still considered valid; finite retention requires expiry or an acknowledgment/epoch protocol that makes old retries rejectable. Transport admission deduplication does not imply external effect deduplication.

## Commit and dispatch protocol

A logical execution step has the following abstract protocol:

1. Select an admissible observation `o`, including any necessary scheduling choice, and read `(v, X_n)`.
2. Evaluate the bounded pure transition to obtain `(X_{n+1}, E_n)`.
3. At the subject's serialization point, validate current run, activation epoch, sequence `n`, and admission status. Atomically commit the consumed observation, successor state or reconstructible state delta, version, intent records, and sequence advance.
4. Only committed intents are eligible for attempt authorization. A dispatcher must durably authorize each attempt under the rules below before interacting; permitted retries retain the original intent ID.
5. Record each admitted completion and its correlation durably before the computation consumes it in a later committed step.

“Atomically” is a required property, not a selected storage product. A log-based design can derive state from the committed event and deterministic transition; a snapshot-based design can materialize it. Either must prevent a state advance without its intended effects, and effects from an uncommitted state advance. An inbox is retained until consumption and successor state are committed together.

Per-run sequence checks and current authority validation must share a serialization protocol with lifecycle operations. Checking an epoch and later writing without guarding the commit creates a time-of-check race. A write acknowledged as durable must survive the stated failure model; ordinary process memory is insufficient.

Use an intent identity such as `effectID = (x, committedSequence, ordinal)`. The parameters are immutable under that identity. A separate dispatch attempt ID may change. An attempt ID is not an idempotency key. The same pattern covers create and lifecycle request retries; their receivers retain the original operation outcome.

During historical replay, the runtime reconstructs state and compares regenerated intents against recorded identities, kinds, and parameters. It does not redispatch historical effects. At the live frontier, a recovery dispatcher may reconcile unresolved committed intents; that is recovery activity, not re-execution of the replayed program's historical I/O.

### Attempt authorization and closure

An intent commit records a request; it does not yet authorize an external attempt. Each attempt has a separate durable `AuthorizeAttempt(k, attemptID)` decision, serialized against cancellation at the source subject. It validates the intent's disposition and current dispatch authority. A fresh dispatcher may take over committed work under current authority without changing `k`; a stale dispatcher cannot authorize another attempt.

| Intent state at cancellation | Required disposition |
| --- | --- |
| Committed, with no attempt ever authorized | Atomically mark suppressed; no dispatcher may start it; non-execution is known under the trusted-dispatch assumption |
| An attempt was authorized, but no authoritative outcome is recorded | Retain unresolved evidence and mark Unknown when appropriate; prohibit new mutating attempt authorizations after cancellation |
| Outcome durably known | Retain it without changing the cancelled program's terminal outcome |
| Frozen terminal/status notification to a previously authorized observer | Retain delivery as a settlement obligation; it cannot request arbitrary new application work |

Authorization before cancellation allows that particular attempt to proceed even if its physical RPC starts or finishes later. A crash after authorization but before dispatch can be indistinguishable from a crash after external application. Authorization is therefore not proof of execution; suppression is known only when no attempt was authorized. Authorizing and suppressing the same intent cannot both win the subject's serialization order.

A terminal transition for success, failure, or authorized abandonment closes program turns, but does not indiscriminately erase delivery obligations. Its terminal commit must assign every outstanding intent a disposition: settle it, suppress it if never authorized, or explicitly transfer responsibility to an authorized custodian. Cancellation uses the stricter suppression and no-new-mutation rules in the table above. The frozen terminal result and its notification route are retained independently of the program frontier. Their delivery can continue under that settlement record even after closure or a later generation. The runtime checks this narrow notification contract; a program cannot label a payment or child creation a status notification to bypass cancellation.

Late receipts are authenticated against retained intent/attempt identity and recorded even when their source activation or run is obsolete. They do not require the obsolete activation to acquire current execution authority, and they cannot resume its normal computation. Authoritative evidence can resolve a provisional Unknown classification. Duplicate receipts preserve a known outcome; contradictory receipts are retained as a reconciliation fault rather than overwriting an accepted known outcome.

After cancellation, the retained intent ledger may authorize read-only outcome queries and delivery of frozen receipts/status. It does not authorize another potentially mutating attempt, even with an idempotency key, because that attempt might apply an operation that had not happened before cancellation. A separate surviving owner or custodian can choose compensation or a new business action only using its own live execution and explicit resource authority. The cancelled subject cannot execute its cleanup callback. This deliberately permits Unknown to remain unresolved when the endpoint has no authoritative query or late receipt.

## Ownership transfer and failure recovery

A proposed transfer protocol is:

1. The prospective owner records acceptance of responsibility for subject `a`, referencing a transfer ID and its authority identity. Acceptance need not imply present availability.
2. An authorized caller submits `Transfer(a, expectedOwner, w, newOwner, acceptanceID, opID)`.
3. One serialized update validates the current revision and acceptance, assigns `newOwner`, increments `w`, and records the transfer result. There is no intermediate ownerless state.
4. Notifications can be retried. After a crash, callers query or retry the same `opID`; the committed authority record settles who owns `a`.

The authority service must define whether target retirement invalidates its acceptance before transfer. The candidate requires acceptance valid at the transfer's serialization point; coordinating that validation across different authority partitions is an unresolved implementation obligation. Without that property, the protocol must permit transfer to an unavailable owner and rely on an explicit recovery principal. It must not claim both atomic validation and independent partitions without a protocol.

A transfer can preserve activation epoch `e`. Then the worker continues, while the former owner loses lifecycle control. If transfer is intended also to stop or move execution, an epoch change is an additional serialized action. Ownership revision `w` and worker epoch `e` serve different purposes and are not interchangeable.

Erlang/OTP demonstrates practical supervision policies and restart intensity limits. Its process restart behavior is evidence for separating policy from mechanism, not evidence that application state, effects, and distributed ownership are durably recovered. [Erlang supervision principles](https://www.erlang.org/doc/system/sup_princ.html).

## Program failure and physical interruption

An activation crash, evaluation interruption, or timeout before transition commit leaves the prior committed state and observation disposition intact. The run remains recoverable; a new activation may resume the same `x` after obtaining authority. A timeout is suspicion and is recorded as such if policy observes it. It does not establish that the former worker died or that a remote effect failed.

A program-level failure is a defined outcome of the versioned evaluator, such as a typed failure return or an exception whose interpretation is specified by `v`. It atomically commits a terminal failure with the consumed observation, fault information, and the intent dispositions above. No partial state or uncommitted intents escape. A deterministic fault is reproduced by replay; repeatedly restarting physical evaluation must not be used to turn it into an unrecorded new attempt.

An interpreter/host crash whose cause is not a defined program outcome is an activation failure, not proof of an application failure. A lifecycle-authorized owner may explicitly commit `Abandon(x, expectedGeneration, reason)` when recovery is impossible or policy stops trying. That closes/fences the run and records an abandoned outcome without claiming that old processes or effects stopped. Only an ended run can receive a fresh retry generation under the candidate policy; a suspicion alone does not create one. Current authority and the cancellation barrier still apply.

Failure notifications to registered observers are durable settlement obligations committed with the run's terminal status; their IDs support duplicate admission. An owner crash between terminal commit and notification delivery cannot erase the failure. Observers distinguish program failure, abandonment, cancellation, and suspicion so ordinary supervision policy can choose resume, retry, or escalation deliberately.

## Lifecycle meanings and races

| Action | Candidate meaning | What it does not establish |
| --- | --- | --- |
| Resume | Reactivate the same `x`, preserving committed state and intents; obtain a fresh epoch | No new application attempt and no repeated historical external I/O |
| Replay | Recompute a recorded prefix under its bound semantics | No rollback of the outside world |
| Retry or restart from initial state | Create generation `g + 1` under explicit policy after the prior run ends | Existing effects and admitted messages are not automatically safe to repeat |
| Replace | Change implementation or internal member through a recorded compatibility/routing procedure | Public identity alone does not prove substitutability |
| Cancel | Close admission for logical subject `a` and record a tombstone; reconcile already authorized work | No erasure of prior results or rollback of completed effects |
| Transfer | Change lifecycle owner and ownership revision | No implicit change in communication topology |
| Escalate | Notify a designated authority of a failure and requested policy action | No automatic authorization to mutate that authority's subjects |

Cancellation of the logical subject is a deliberately strong provisional choice: automatic retries cannot revive it. A future intentional task uses a new identity. Cancelling only a run while permitting later runs is a viable alternative in [Decisions](05-decisions.md); its API must distinguish cancelling `x` from stopping retries for `a`.

Cancellation, retry, completion, and ownership changes serialize against the same subject record. If cancellation commits first, retry fails. If retry commits first, cancellation of `a` covers the new current run. A completed result remains immutable if completion commits before cancellation; cancellation closes future admission and reports that work already completed. If cancellation commits first, a later ordinary success transition cannot overwrite the cancelled outcome. Cancellation has separate status for unresolved external intents.

Enforcement cannot depend on a cooperative cancellation callback. The runtime rejects new state commits and ordinary attempt authorizations, while an active, explicitly authorized custodian can handle cleanup in its own execution. Previously authorized calls may still finish. Frozen outcome delivery and read-only reconciliation remain possible under their retained settlement authority. Scope-wide cancellation and its required surviving principal are described in [Composition](02-composition.md).

## Stale activations and external fencing

An old activation can keep running after a lease expires or a network partitions. Preventing two processes from existing is unnecessary for the candidate safety property: at most one authorized history extends a run at any sequence, and obsolete epochs cannot authorize new commits.

State storage must reject a stale `(x, e, n)` at commit. Dispatch must validate current authorization before starting a new attempt, but a check made earlier cannot revoke an RPC already in flight. A cooperative effect target can check an authoritative fencing token at the mutation's serialization point. Otherwise a cancellation or failover cannot promise that all old physical calls have stopped.

Chubby's sequencers illustrate recipient-side validation of lock authority. ALAN adopts the obligation to make stale authority ineffective at the protected resource; it does not adopt Chubby as infrastructure. [Burrows, The Chubby lock service, section 2.4](https://www.usenix.org/legacy/event/osdi06/tech/full_papers/burrows/burrows_html/).

A target that only remembers the largest epoch it has seen offers a narrower guarantee: an old request is rejected after a newer epoch reaches that target. An old request arriving before the new epoch may still execute. Stronger “no old mutations after failover” semantics require authoritative validation at the target or a barrier acknowledged there before failover is declared complete. Effects at non-cooperating endpoints retain explicitly weaker guarantees.

## Interrupted effects and delivery guarantees

Suppose a payment succeeds, then the dispatcher crashes before recording success. The history contains a committed intent but no completion. It cannot distinguish success from non-execution merely by replaying local state. Mark the outcome `Unknown`; do not transform absence of a completion into proof of failure.

| Endpoint protocol | Recovery behavior | Defensible guarantee |
| --- | --- | --- |
| Durable idempotency key, immutable parameters, atomic deduplication with mutation, recoverable result | Retry/query using the original key | At most one logical mutation for that key within its stated scope; eventual completion only with availability and retry assumptions |
| Transactional participation in the same commit protocol | Recover the common transaction decision | Atomicity only for participating resources and under that protocol's assumptions |
| Queryable operation identity without safe retry | Query and reconcile; do not blindly repeat | Outcome knowledge when the query is authoritative and complete |
| No idempotency or authoritative query | Stop automatic retry, expose unknown, apply explicit domain policy | Duplicate-versus-loss ambiguity remains |

Idempotency needs a retention horizon covering recovery, parameter conflict detection, and result recovery or a documented weaker response. A provider that forgets a key after a day cannot protect a replayed request a week later. Compensation is a new effect that can fail; it is not historical erasure.

A retry in generation `g + 1` creates new execution-scoped IDs. That alone does not deduplicate an operation already performed in `g`. The supervisor must choose whether the task is a new business operation, retain a stable business operation key across generations, or reconcile the prior intent before retry. Unknown effects cannot safely disappear behind a fresh run ID.

Exactly-once external execution is not claimed. Even with an idempotent endpoint, multiple physical invocations can occur while only one logical mutation takes effect. Internal atomic intent recording closes a local failure window; it does not make an arbitrary remote endpoint transactional.

## Replay evidence and version changes

The durable record must contain, or permit deterministic reconstruction of:

- Initial input, execution identity, program artifact identity, interpreter/runtime semantic version, and state schema.
- Every consumed message and external result, including error/unknown classifications, model outputs, timer firings, random draws, environmental reads, and their selected order.
- Continuation selection or a deterministic selection rule, correlation bindings, intent IDs and immutable parameters, and resulting state or verifiable checkpoints.
- Applied lifecycle decisions, generations, migration boundaries, admission rejections that influence behavior, and external outcomes used by policy.
- Content needed to reconstruct referenced artifacts. A hash without retrievable bytes cannot reproduce the result. Access control and retention must preserve the declared replay horizon.

Pure computation also depends on stable arithmetic, serialization, iteration order, locale, and library behavior. Those dependencies belong in `v` or in a constrained deterministic language. A snapshot is useful only with its exact version binding, history position, integrity checks, and the inbox/intent records needed at that position.

For a fixed `v`, initial state, and observation sequence, induction over deterministic transitions gives equal reconstructed states and emitted intents. This is a conditional local replay argument. It does not prove that the recorded history is complete, that a distributed snapshot is consistent, or that the external world will reproduce its past behavior. Replicated-state-machine literature similarly requires agreed ordering and deterministic processing; replication does not remove external effect ambiguity. [Schneider, Implementing Fault-Tolerant Services Using the State Machine Approach](https://www.cs.cornell.edu/fbs/publications/SMSurvey.pdf).

Three upgrade strategies remain viable: retain and route to the old executable; introduce an explicitly recorded compatibility branch; or migrate state at a declared boundary after resolving or carrying forward pending work. Blindly replaying old observations through changed code is rejected. A new model version affects future model calls; an old model result remains data during replay.

Migration needs a mapping for continuation labels, schema, pending effects, capability scopes, and in-flight replies, plus a refinement or compatibility argument. Rejecting incompatible replay is safer than silently accepting divergent intents, but cannot itself recover a missing old executable. Availability depends on artifact retention and a chosen migration policy.

Per-execution replay is the baseline. Debugging or restoring an entire composite additionally requires a causally consistent cut of component histories, admitted messages, and in-flight sends. A global total order is unnecessary for local replay, but arbitrary snapshots from independent logs are not automatically a valid global state. [Lamport, Time, Clocks, and the Ordering of Events in a Distributed System](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/12/Time-Clocks-and-the-Ordering-of-Events-in-a-Distributed-System.pdf).

## Safety and progress assumptions

The design assumes trusted authority enforcement, durable storage within a declared failure model, atomic per-subject serialization, finite turns, and immutable versioned inputs to replay. It does not assume a perfect failure detector or bounded message delays.

Progress additionally needs eventual recovery/connectivity, fair scheduling of continuously enabled work, sufficient resource admission, and external dependencies that eventually respond or allow a declared timeout/unknown outcome. A distributed authority implementation cannot promise deterministic termination under every fully asynchronous execution with even one crash; it must state its extra progress assumptions. [Fischer, Lynch, and Paterson, Impossibility of Distributed Consensus with One Faulty Process](https://www.cs.cornell.edu/courses/cs5414/2017fa/papers/fischer.pdf).

No storage engine, consensus implementation, queue, SDK, or provider integration is selected. Their required properties are the subject of the next design phase only if that work is authorized.
