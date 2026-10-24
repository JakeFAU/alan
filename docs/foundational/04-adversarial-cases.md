# ALAN adversarial cases

Status: research candidate, 24 October 2026. These are paper execution traces against the semantics in [Foundations](01-foundations.md) and [Runtime boundary](03-runtime-boundary.md). They are counterexample searches and proposed invariant arguments, not model-checking results or executed tests.

## Notation and target invariants

`a` is a logical agent, `x = (a, g)` its current execution, `e` an activation epoch, `w` an ownership revision, and `n` a committed execution sequence. `commit(...)` denotes the relevant authority/history serialization point. `emit(k)` means an intent was committed; `apply(k)` means the external resource performed its mutation; `receipt(k)` means the outcome was durably recorded. Those are three distinct events.

| ID | Safety invariant |
| --- | --- |
| I1 | One authorized history extends each execution sequence; stale epochs cannot append transitions |
| I2 | Lifecycle mutations require current authority and serialize with conflicting lifecycle/commit operations |
| I3 | Communication permission, creation provenance, and lifecycle ownership do not imply one another |
| I4 | Replay of a fixed history under its bound semantics reconstructs the same state and intents without historical external I/O |
| I5 | External ambiguity remains explicit; no absent receipt is interpreted as proof of non-execution |
| I6 | A cancellation tombstone closes the subject to new work and automatic retry; committed outcomes remain immutable |
| I7 | Correlation and generation checks prevent duplicate or late events from advancing unrelated work |
| I8 | Hiding and replacement preserve the selected external contract, including externally visible effects |
| I9 | Admission, consumption, and resulting intents are durably consistent across crashes |

Progress is conditional. L1: continuously enabled work eventually gets a turn under weak fairness. L2: accepted communication eventually reaches its declared disposition if recovery, capacity, and connectivity premises hold. L3: lifecycle decisions eventually resolve if the authority boundary and responsible policy recover. None requires an unresponsive external service to succeed or an arbitrary recursive program to terminate.

## Case 1 A suspended member beside an active receiver

**Setup.** Composite `C` contains `A` and `B`. `A` waits on tool intent `k`; `B` accepts messages from peer `P`.

```text
t0  A commits emit(k), records Wait(k)
t1  B has a ready receive continuation
t2  P -> B: message m is durably admitted
t3  scheduler selects B; B commits consume(m), emits reply r
t4  A remains suspended because no receipt(k) exists
```

**Expected invariant.** I9 holds, and the wait in `A` does not disable `B`. L1 permits `B` to progress while `A` waits. The composite's external contract can report partial progress without declaring total completion.

**Counterexample.** A single composite continuation is “await A, then run B.” `B`'s admitted message never gets processed if `A`'s tool never returns. Another failure is a scheduler that repeatedly chooses an internal busy loop and starves `B` despite available work.

**Semantic choice still open.** Weak fairness for continuously enabled actions is the candidate; stronger service guarantees, priorities, and starvation bounds remain undecided. Serializing a local frontier and deploying separate child executions provide different isolation and resource behavior.

**Proof obligation P1.** Define readiness and fair scheduling so nesting/hiding preserves progress of independently enabled components. Show the implementation of bounded local steps meets that premise.

## Case 2 A child contacts a peer outside ownership

**Setup.** `S` owns child `A`. `A` holds a send-only capability to independent peer `P` and passes a restricted reply capability to it.

```text
t0  authority: owner(A) = S; owner(P) = U
t1  A -> P: request q with reply capability for A
t2  P -> A: reply(q) admitted under that capability
t3  P requests Cancel(A) using its send capability
t4  runtime rejects the lifecycle request; ownership is unchanged
```

**Expected invariant.** I3 holds. Direct peer communication is permitted without forwarding through `S` and without transferring ownership to `P`. I8 counts the peer interaction as externally observable when evaluating the composite.

**Counterexample.** The runtime treats any bidirectional communication edge as a parent-child relationship, or derives lifecycle permission from knowing `A`'s identifier. Then a peer can kill `A`, or contact with `P` unexpectedly changes failure propagation.

**Semantic choice still open.** Capability attenuation, revocation, expiry, and the handling of messages admitted before revocation need a complete model. The candidate does not automatically revoke communication rights on ownership transfer.

**Proof obligation P2.** Show that send, delegation, and name extrusion preserve authority separation and that exported child capabilities are included in the observable interface.

## Case 3 The supervisor crashes during ownership transfer

**Setup.** `S` owns `A` at revision `w = 7`; `T` durably accepts transfer ID `h`.

```text
t0  T records Accept(h, A)
t1  S requests Transfer(A, S, 7, T, h, op42)
t2  authority commits owner(A)=T, w=8, result(op42)=success
t3  S crashes before receiving success
t4  recovered S retries op42; it receives the original result
t5  stale request by S at w=7 is rejected
```

**Expected invariant.** I2 holds: there is exactly one committed owner at every point. Crash before `t2` leaves `S` owner; crash after `t2` leaves `T` owner. L3 requires recovery of the authority boundary; notifications are not the authority record.

**Counterexample.** `S` removes its ownership entry and then asks `T` to create a new entry. A crash between those writes leaves `A` ownerless. Reversing the order can create two owners. Retrying an unremembered operation can create a second transfer with a new meaning.

**Semantic choice still open.** A target can retire after accepting but before transfer. Validating target liveness/eligibility atomically may cross authority partitions. Alternatives are coordinated validation or acceptance of unavailable owners with a designated recovery principal. If ownership must be acyclic, cycle prevention adds another cross-record obligation.

**Proof obligation P3.** Specify the transfer serialization and duplicate-operation protocol, including crashes at every boundary and target retirement. Consent is not a liveness guarantee.

## Case 4 Two physical incarnations survive a partition

**Setup.** Worker `Wold` has `(x, e=4)`. A partition leads to authority granting `(x, e=5)` to `Wnew`.

```text
t0  Wold reads X_n under e=4
t1  authority grants e=5; Wnew resumes the same execution x
t2  Wold attempts commit(n+1, e=4); rejected
t3  Wnew commits(n+1, e=5); accepted
t4  an RPC authorized before t1 from Wold reaches an external resource
```

**Expected invariant.** I1 holds at the history store. Strong external exclusion holds only if the resource validates current authority atomically with mutation or participates in a completed fencing barrier. The candidate does not promise that `Wold` ceases to exist.

**Counterexample.** A lease expires locally, but the store accepts a write checked against authority before `t1`. Or the store rejects the stale write while an unfenced external API accepts `t4`, causing an effect absent from the new worker's assumptions. A “largest epoch seen” target can still accept epoch 4 if epoch 5 has not reached it yet.

**Semantic choice still open.** Which effects require strong fencing, which can use idempotency, and which admit unknown/late outcomes? Blocking for an external fencing barrier sacrifices availability at that resource.

**Proof obligation P4.** Identify every protected mutation boundary and prove stale-authority rejection there. A proof about the history store alone does not cover the external resource.

## Case 5 The effect succeeds before the completion is recorded

**Setup.** A run requests a payment under stable intent ID `k`.

```text
t0  commit(X_next, emit(k))
t1  dispatcher sends payment(k)
t2  payment service applies k
t3  dispatcher crashes before receipt(k) is durable
t4  recovery sees k pending; its outcome is Unknown
```

**Expected invariant.** I5 holds. Replay can reconstruct `emit(k)` but must not assert non-execution. If the service supports atomic idempotency and result retrieval over the recovery horizon, recovery can query/retry the same key. L2 depends on that endpoint and the chosen reconciliation policy.

**Counterexample.** Recovery generates a new key and pays twice. Disabling retries instead can lose a payment in the indistinguishable history where the dispatcher crashed before `t2`. Calling the outbox “exactly once” does not distinguish these histories.

**Semantic choice still open.** The adapter contract determines whether to retry, query, compensate, or await explicit intervention. If a new run `g+1` repeats the task, it must explicitly retain the business operation key or accept a new mutation; a new run ID is not a recovery protocol.

**Proof obligation P5.** Prove the chosen endpoint's deduplication/mutation atomicity, parameter consistency, and retention scope, or preserve Unknown as an externally meaningful outcome. Do not claim exactly-once physical execution.

## Case 6 A durable execution encounters modified code or model versions

**Setup.** Version `v1` emits `Send(A)` after observing model output `r`. Version `v2` would emit `Send(B)` for the same `r`.

```text
t0  x binds v1; model result r is recorded
t1  v1 consumes r and commits intent Send(A)
t2  process crashes; deployment now defaults to v2
t3  recovery attempts replay under v2
t4  version validation or regenerated-intent comparison rejects divergence
```

**Expected invariant.** I4 holds by selecting the bound semantics or refusing incompatible recovery. Recorded model result `r` is reused; replay does not call a new model to recreate it. Refusal preserves safety but may leave the execution unavailable.

**Counterexample.** The runtime loads “latest,” regenerates `Send(B)`, and continues as though history had always contained that command. Another failure reinvokes the same model alias and accepts a changed answer as historical fact. Matching intent labels alone also misses changed arguments or divergent internal state.

**Semantic choice still open.** Preserve old executables, use recorded version branches, or permit explicit state migrations. Migration must account for continuation labels, pending intents, input schemas, and late replies. The retention horizon for code and observations is unresolved.

**Proof obligation P6.** Define replay compatibility and migration simulation relations. Verify full intent identity/parameters and versioned state interpretation, not just the number of commands.

## Case 7 Two valid agents create an infinite feedback loop

**Setup.** `A` replies `ping` to each `pong`; `B` replies `pong` to each `ping`. Each independently handles any admitted input in one finite step.

```text
t0  external seed -> A: pong
t1  A -> B: ping(1)
t2  B -> A: pong(1)
t3  A -> B: ping(2)
... every step is finite; the network never quiesces
```

**Expected invariant.** Local safety can hold throughout. Fair delivery can also hold throughout. Neither implies global termination, useful progress, or bounded cumulative resource use. A composite promising quiescence would violate its contract.

**Counterexample.** The architecture claims that composing terminating handlers yields a terminating agent, or treats an increasing history of messages as proof that the goal is being achieved. The handlers terminate; the reactive composition does not.

**Semantic choice still open.** Protocol budgets, hop counts, explicit stopping predicates, or decreasing measures can constrain selected compositions. Whether to enforce runtime-wide quotas or require per-contract budgets remains open. Quotas end work by exhaustion, which is distinct from successful completion.

**Proof obligation P7.** Establish an assume-guarantee progress argument or a ranking measure for each claimed terminating composition. Do not seek a general termination guarantee for arbitrary universal programs.

## Case 8 Retry races with cancellation from another supervisor

**Setup.** Two delegates hold currently valid but different lifecycle rights over `A`: `S` may retry; `T` may cancel. The current run `g=3` failed. Both requests refer to ownership revision `w=9`.

```text
Order A:
  commit Cancel(A, w=9): cancellation tombstone set
  Retry(A, expected g=3, w=9) -> rejected

Order B:
  commit Retry(A, expected g=3, w=9): current g=4
  commit Cancel(A, w=9): closes admission for A, including g=4
```

**Expected invariant.** I2 and I6 hold. No automatic retry survives a committed cancellation of the logical subject. A request at an obsolete ownership revision fails regardless of which supervisor sent it. A success already committed before cancellation remains historical success. The failed run in this setup must have a committed program-failure or abandonment outcome; physical worker suspicion alone does not authorize a fresh generation.

**Counterexample.** Each supervisor writes an independent flag, producing both “cancelled” and “running.” Or cancellation names only old run `g=3` while users believe it stops all retries, allowing `g=4` to continue unnoticed. An unfenced external call authorized before cancellation may still finish even when the internal rule is correct.

**Semantic choice still open.** The candidate permanently closes logical `A`. A run-only cancellation model is also coherent, but needs distinct operations and a clear retry barrier. Group cancellation and reconciliation of already dispatched effects require separate policies.

**Proof obligation P8.** Model all orderings of retry, cancel, completion, transfer, and late receipts at the same authority boundary. State exactly when “cancelled” means no more local commits and when external quiescence is known.

## Case 9 An internal worker is replaced behind a stable interface

**Setup.** Facade `F` exposes `Answer(requestID)`. It uses internal worker `Aold`; request `q` and external effect `k` are outstanding.

```text
t0  F records routing(q) = Aold, generation 2
t1  F closes new admission to Aold and records replacement procedure
t2  reconcile k; assign each pending request to one authoritative worker
t3  F switches new routing to Anew, generation 3
t4  late reply(q, generation 2) arrives
t5  F applies the recorded old-request disposition; it does not satisfy a new q'
```

**Expected invariant.** I7 and I8 hold. New and old replies are not confused; externally visible effects are neither forgotten nor accidentally repeated. A request whose effects remain unknown may block migration or be explicitly carried forward as unknown.

**Counterexample.** The facade reruns `q` on `Anew` with a fresh payment key and publishes both results. Or it changes the old worker's run capability to point at `Anew`, breaking an external peer's identity and protocol assumptions. An unchanged payload schema hides neither problem.

**Semantic choice still open.** Drain, migrate, forward, or invalidate old work and capabilities. A direct external peer holding `Aold`'s capability enlarges the composite boundary. Perfectly transparent replacement is unavailable unless that peer-facing contract supports it.

**Proof obligation P9.** Prove a refinement or contextual equivalence for the replacement under the selected observations, including ordering, failure, divergence, and effects. A schema check is not enough.

## Case 10 Control demands an operation the primitives do not provide

**Setup.** Ordinary Control `C` starts `W1` and `W2`. A proposed requirement says “either both external writes occur or neither occurs, and cancellation instantly rolls both back.” Both endpoints expose only independent, nontransactional mutation APIs.

```text
t0  C commits intent k1 for W1 and k2 for W2
t1  endpoint 1 applies k1
t2  endpoint 2 becomes unavailable before applying k2
t3  C receives an ordinary CancelRequest message; C remains live
t4  C attempts compensation for k1; compensation also fails
```

**Expected invariant.** I5 and I8 require the partial external outcome to remain visible. Control can coordinate retries and compensations as ordinary computation, but cannot truthfully report atomic rollback under these endpoint contracts. The trace uses cooperative cancellation as a protocol input; after runtime cancellation closes `C`, compensation would require a surviving authorized custodian in its own execution.

**Counterexample.** A privileged Control primitive promises the atomic result without a common transaction protocol, or the design calls compensation rollback and reports success before compensation is confirmed. More continuation branches do not solve this problem.

**Semantic choice still open.** Weaken the public contract to expose partial outcomes and compensation, require participating transactional endpoints, or reject this use case. The candidate chooses honest partial-outcome reporting; adding atomic multi-resource commitment would be a separate protocol decision.

**Proof obligation P10.** Show a constructive encoding for every advertised Control operation using the documented rules. For an unencodable requirement, identify the missing environmental guarantee rather than silently adding a privileged agent type.

## Additional boundary probes

**Deduplication expiry.** Message `m` is admitted and processed; its deduplication record is deleted; a delayed retry of `m` arrives and is treated as new. This violates I7/I9 unless expiry makes the old retry invalid. Retention and acceptance horizons must be specified together.

**Crash before intent commit.** A transition computes a tool intent but crashes before commit. If the tool has already been dispatched, the outside world changes without committed history. Commit-before-dispatch is therefore part of the semantic boundary, not merely an optimization.

**Late receipt after cancellation.** An already authorized effect completes after the subject is closed. Discarding its receipt destroys reconciliation evidence; treating it as a normal reply revives cancelled work. Record the receipt in the effect ledger while rejecting ordinary continuation advancement.

**Cancellation before dispatch authorization.** `emit(k)` commits; cancellation wins before `AuthorizeAttempt(k)`; the intent becomes suppressed and any later authorization fails. Reverse the last two events and execution is potentially Unknown: the authorized attempt may still reach the endpoint. Read-only reconciliation remains possible, but cancellation prohibits authorizing a new mutating retry. This distinction is observable even when an idempotency key exists.

**Terminal reply delivery.** A worker commits success and its frozen result notification in one step, then crashes. Its frontier is closed, but the settlement record permits recovery to deliver that result. A cancellation arriving later preserves the committed outcome. Arbitrary new tool or creation requests cannot use the status-notification exception.

**The owner is cancelled before a creation reply.** Facade `C` closes while a creation is already authorized. Without an independently authorized cleanup agent, the new member can outlive `C` and no group-stop guarantee follows. With ordinary custodian `G` holding the creation ledger and member authority, `G` resolves the retained creation operation, learns the member identity, and cancels it. A missing receipt leaves group quiescence unproven.

**Failure classification.** A worker crashes while evaluating observation `o` before commit: `o` remains eligible and a new activation resumes the same run. A defined deterministic fault while evaluating `o` instead commits terminal failure and a durable notification. A mere timeout does neither; an authorized abandonment decision is required to end an otherwise recoverable run before retrying it as a new generation.

**Unbounded host callback.** A OneShot callback loops without yielding. Logical cancellation can close admission at the authority service, but cannot make this callback produce a resumable checkpoint. The future evaluator must enforce the finite-step contract or classify the activation as failed.

## Review criteria

Each case must retain its counterexample even if a particular protocol prevents it. A proposed resolution is acceptable only when its assumptions match the guarantee: local atomicity cannot justify external atomicity; history fencing cannot justify process death; trace safety cannot justify progress.

The independent review record in [Decisions](05-decisions.md) distinguishes corrected contradictions from remaining research obligations. No case is labeled “proved” merely because the intended invariant has been written down.
