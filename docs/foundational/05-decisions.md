# ALAN foundational decisions and open questions

Status: research candidate, 24 October 2026. The recommended foundation is a versioned interactive transition machine with explicit effects and a finite frontier of live continuations, embedded in a runtime that supplies independent scheduling, durable decisions, and checked lifecycle authority. OneShot, Conditional, Control, and supervision can be programming constructions; none needs to be a privileged agent type.

This recommendation is provisional. The documents establish a concrete candidate to examine, not a proof of minimality, general compositional correctness, or a production-ready design. The user's requested scope ends with reviewed design documents.

## Conclusions supported by established work

| Conclusion | Basis | Limit on the conclusion |
| --- | --- | --- |
| Small machine models can express arbitrary partial recursive computation under suitable encodings | Church, Turing, and Minsky; sources in [Foundations](01-foundations.md) | No result fixes a minimum number of agent-framework primitives or supplies runtime guarantees |
| Communication and composition need explicit semantics beyond local evaluation | Actor configurations, CSP, π-calculus, and I/O automata | Their theorems have different assumptions; ALAN does not inherit them by analogy |
| A transition system can be specified with local operational rules | Plotkin's SOS | Determinism, adequacy, and congruence still need arguments for the specific rules |
| Requests and their interpretation can be separated compositionally | Xia et al.'s Interaction Trees | Response alternatives are not a theorem about concurrent live branches |
| “Derived” depends on the permitted translation and observations | Felleisen's expressibility criteria | Whole-program simulation is weaker than a local encoding preserving interaction behavior |
| Restart policy, ordered deterministic state, and effect execution solve different problems | OTP, durable workflow documentation, and state-machine replication | Restart and replay do not imply exactly-once external execution |

The ALAN-specific inferences are to expose interactions in the semantics, keep control roles ordinary, and specify runtime obligations separately. They are architectural judgments informed by these results.

## Candidate decisions and their costs

These are proposed choices used consistently in the traces, not decisions already accepted by the project owner.

| ID | Candidate choice | Reason and cost |
| --- | --- | --- |
| D1 | Use an open transition system with deterministic finite local reduction | Makes external uncertainty visible; requires a language/evaluator that can actually enforce finite deterministic steps |
| D2 | Store residual control in execution state; expose a finite frontier when useful | Supports several live branches; explicit routing, wait, and completion rules remain necessary |
| D3 | Treat OneShot and Conditional as derived programming constructs under a defined host basis | Avoids defining architecture by names that conceal host-language power; exact surface syntax stays open |
| D4 | Use asynchronous capability-addressed communication with explicit durable admission | Supports dynamic topology and independent progress; ordering, deduplication, capacity, and revocation become explicit contracts |
| D5 | Separate logical identity, execution generation, physical activation, and ownership revision | Makes recovery and competing authorities intelligible; adds metadata and validation obligations |
| D6 | Serialize one current run and lifecycle authority per logical agent | Gives a clear retry/cancel/transfer boundary; parallel independent runs need separate logical subjects or a more complex model |
| D7 | Separate provenance, lifecycle authority, and communication rights | Permits peers outside ownership and independent supervision; no automatic tree semantics |
| D8 | Commit state and effect intents atomically; separately serialize attempt authorization against cancellation | Prevents local dual-write loss and defines suppression; external ambiguity still requires endpoint contracts |
| D9 | Bind replay to immutable semantics and recorded nondeterministic observations | Reproduces decisions; creates retention and upgrade obligations |
| D10 | In the candidate, cancellation permanently closes a logical subject to work and automatic retries | Gives an unambiguous retry barrier; is stronger than run-only cancellation and may be unsuitable for long-lived services |
| D11 | Make composite membership, failure policy, and group cancellation ordinary protocols | Preserves recursive uniformity; group cleanup after facade retirement needs a surviving principal with explicit ledger access and member authority |
| D12 | Permit loss of availability when authority or effect outcome is uncertain | Preserves stated safety; liveness requires additional recovery/connectivity assumptions |
| D13 | Distinguish activation interruption, deterministic program failure, and authorized abandonment | Resume retains the run; a new retry generation requires a committed ended run, preventing suspicion from silently repeating business operations |

No choice fixes a global total order, storage engine, broker, consensus algorithm, programming language, provider, or deployment topology. Atomicity is specified at an authority boundary, and any implementation must justify that boundary.

## Alternative designs and why they remain relevant

| Design | Strength | Limitation or reason not to prefer it as the current foundation |
| --- | --- | --- |
| Pure lambda calculus plus an external driver | Very small computational core; familiar denotational reasoning | The driver carries interaction, concurrency, authority, and durability; omitting it from the model hides the main research problem |
| Single callback with arbitrary host power | Compact API and easy prototyping | Minimality is vacuous if the callback/interpreter contains the entire architecture; unbounded callbacks compromise scheduling |
| Actor system as the whole foundation | Natural asynchronous creation and communication | Requires explicit durable identity, authority, replay, and capability contracts beyond ordinary process semantics |
| CSP-style rendezvous processes | Explicit synchronization and algebraic reasoning | The target behavior includes independently suspended asynchronous work; buffering and durability still need definition |
| π-calculus as the core language | Direct account of dynamic connectivity and scoped names | More formal machinery than needed for the first candidate; security and durability remain separate |
| Sequential Interaction Tree with effect handlers | Clean representation of request/response and residual computation | Concurrency must be represented and given scheduling semantics; it is not obtained by treating all response continuations as live |
| One globally serialized machine | Simple global ordering and atomic cross-component reasoning | Couples availability and throughput; changes failure independence and can hide costs of distributed composition |
| Privileged Control or Supervisor runtime types | Can package useful defaults | Type proliferation is unnecessary for policy; special types do not solve missing resource protocols |
| Nondeterministic local transitions with recorded decisions | Permits a richer evaluator and local search | Requires recording every relevant choice and defining replay at a finer level; deterministic local reduction is easier to audit |

The candidate is closest to an interactive machine with actor-like deployment, but that comparison is descriptive. Its guarantees are the explicit rules in these documents, not whichever behavior a reader associates with an existing actor or workflow framework.

## Minimality as an elimination exercise

No primitive count is claimed. The right question is what behavior disappears if an obligation is removed, under an explicitly fixed host language and observation model.

| Remove or encode | Result |
| --- | --- |
| Separate `K` field | Can be absorbed into successor state without losing behavior; it is representational |
| Conditional constructor | Can be encoded by a bounded pure reducer with selection; not by every restricted terminal-action basis |
| Control constructor | Can be encoded by ordinary state, messages, creation, and effect handling as shown in [Composition](02-composition.md) |
| Local frontier | Can be represented as ordinary state plus a dispatcher; whether the resulting encoding preserves scheduling/progress observations must be shown |
| Dynamic runtime creation | Can be simulated by a universal machine's data structures for some observations; independent failure/isolation and external identities do not follow from that simulation |
| External interaction | A closed program can simulate a world model but cannot receive an actual new observation or mutate an external resource without an interface |
| Lifecycle authority enforcement | Ordinary policy can request exclusivity but cannot guarantee rejection of stale or unauthorized requests by itself |
| Durable intent/observation record | Crash recovery loses the evidence needed to distinguish pending, completed, and ambiguous work |

This suggests minimizing distinct semantic obligations and making derivations visible, rather than minimizing constructor names. Alternative kernels should be compared on encodings, guarantees, and implementation obligations under the same observations.

## Architectural choices still open

| Question | Candidate position | Alternatives and evidence needed |
| --- | --- | --- |
| What is observable? | Inputs/outputs, effects, terminal outcomes, cancellation and declared progress; timing only when promised | Define a trace/failures/divergence or bisimulation model and prove its congruence properties |
| What does cancellation target? | Logical subject with a permanent tombstone | Run-only cancellation plus an explicit stop-retry operation; assess long-lived service use cases and case 8 |
| How does a composite stop? | Ordinary durable member policy with a surviving authorized custodian when the facade is retired | Without that custodian only local stop is promised; a shared cancellation barrier could provide stronger guarantees |
| What fairness is promised? | Weak fairness for continuously enabled work under available resources | Strong fairness, priorities, or bounded response guarantees; specify admission and hostile-workload assumptions |
| How are finite turns enforced? | Required semantic property | Restricted language, resumable interpreter, or deterministic fuel; unrestricted host callbacks do not suffice |
| How much ordering is exposed? | Recorded per-execution order, no implicit FIFO across delivery | Per-sender FIFO or causal ordering; quantify added state and recovery obligations |
| Is ownership a forest? | No implicit topology restriction | Enforce a forest with transactional cycle prevention, or allow cycles with an administrative recovery policy |
| How is transfer acceptance validated? | Acceptance valid at transfer commit, or explicit unavailable-owner recovery | Prove a coordinated cross-record protocol; do not assume local CAS validates remote retirement |
| How are capability rights revoked? | Explicit validation; transfer alone does not revoke send rights | Revocable indirection, expiry, or bounded capabilities; define already-admitted message treatment |
| What replay horizon is offered? | No duration selected | Set data/code retention, deduplication expiry, compaction, and backup consistency as one contract |
| How do code and state evolve? | Reject incompatible replay | Retained executables, recorded compatibility branches, or explicit migrations with pending-work mapping |
| Which effects permit automatic retry? | Adapter-specific; Unknown is preserved | Require idempotency for some categories, use query/reconciliation for others, or explicitly accept duplicate/omission risk |
| Is one current run per agent sufficient? | Yes for the candidate, with ordinary composition for concurrency | Separate service identity from multiple writable executions; specify shared state and cancellation scope |

These questions are not silently delegated to a future implementation. Changing any answer can change observable behavior and invalidate one or more candidate invariants.

## Research obligations before an implementation specification

| Obligation | Concrete next research artifact | Related adversarial cases |
| --- | --- | --- |
| P1 Scheduling and closure | Small-step rules for a frontier and network; nesting/fairness preservation argument | 1 |
| P2 Capability authority separation | Capability transition rules including delegation, revocation, and external scope | 2, 9 |
| P3 Ownership transfer | State machine for consent, commit, retry, target retirement, and recovery principal | 3 |
| P4 Activation fencing | Resource-specific safety argument naming every validation/linearization point | 4 |
| P5 External uncertainty | Effect protocol specification with retention bounds and Unknown transitions | 5 |
| P6 Replay and migration | Induction for fixed-version replay; separate simulation for each migration strategy | 6 |
| P7 Composition progress | Assume-guarantee obligations and counterexamples for feedback and resource bounds | 7 |
| P8 Lifecycle races | Exhaustive paper or formal transition exploration for cancel/retry/complete/transfer | 8 |
| P9 Substitution | Selected observation relation with hiding and replacement congruence conditions | 9 |
| P10 Control adequacy | Expand every advertised operation into ordinary transitions or state the missing protocol | 10 |

Further source-level work should compare the candidate formally against a small actor language and an interactive-tree encoding. It should distinguish what can be encoded globally from what can be eliminated locally while preserving contextual behavior. That work is a proposed next phase, not authorization to start implementation.

## Independent adversarial review

A fresh reviewer, separate from the two source-research contributors and the document author, read the original brief and all five documents. The first review returned **REJECT for deliverable readiness**. The rejection and the substantive findings are preserved below; they were not waived as implementation details.

| Finding | Initial severity | Correction and remaining limit |
| --- | --- | --- |
| R1 Effect authorization was undefined between intent commit and cancellation | P1 | Runtime boundary now defines durable per-attempt authorization serialized against cancellation, known suppression before any authorization, uncertain outcomes afterward, and retained settlement authority. No new mutating attempts are authorized after cancellation; already authorized calls may still execute. Additional traces cover both orders and terminal reply delivery. |
| R2 Group cleanup had no guaranteed surviving executor after its owning facade was permanently cancelled | P2 | Composition now constructs an ordinary custodian with explicit member authority and a durable creation/member ledger. Pending creations must be resolved before declaring group quiescence. Without those premises, only local cancellation is promised. No rights are inherited from parentage. |
| R3 Physical interruption, program failure, and abandonment had no distinct transition rules | P2 | Runtime boundary now defines recoverable pre-commit interruption, versioned deterministic terminal faults, explicit authorized abandonment, and durable failure notifications. Suspicion alone does not create a fresh execution generation. |
| S1 The restricted OneShot non-expressibility argument omitted totality assumptions on leaves | Source-audit precision finding | Foundations now requires complete fixed traces, normal termination, and no input-dependent failure/divergence/early exit for the induction. The conclusion remains relative to that restricted basis. |

The review also preserved three substantive design alternatives: permanent logical-subject cancellation versus run-only cancellation; coordinated transfer validation versus accepting unavailable owners; and weak fairness versus stronger progress guarantees. The candidate choices are retained for analysis, while the alternatives remain open in the decision ledger above. The reviewer found no additional blocking source-accuracy issue in sampled primary sources; the separate foundations audit produced S1.

Follow-up consistency checks aligned timeout wording with activation failure, made case 10's cooperative CancelRequest distinct from runtime cancellation, and explicitly applied intent disposition rules to authorized abandonment as well as success/failure.

The independent re-review returned **APPROVE for design-deliverable readiness**, with no blocking findings remaining, after rechecking all five documents and the follow-up corrections. The reviewer confirmed R1, R2, and R3 resolved. This verdict concerns the readiness of the investigation, not mathematical proof, implementation approval, or selection of the open alternatives.

## Recommended stopping point

Retain this candidate as the baseline for further formal investigation. Its strongest result is an explicit separation of deterministic computation, compositional policy, and runtime enforcement, with concrete cases that can falsify overstated guarantees.

Do not begin runtime, SDK, persistence, or provider implementation from this phase. Resolve the observation model, finite-step enforcement, cancellation scope, authority serialization, and replay/version strategy before an implementation specification can promise behavior with confidence.
