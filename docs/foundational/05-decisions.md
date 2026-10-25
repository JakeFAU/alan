# ALAN foundational decisions and open questions

Status: research candidate, 24 October 2026. The recommended foundation is a versioned interactive transition machine with explicit effects and a finite frontier of live continuations, embedded in a runtime that supplies independent scheduling, durable decisions, and checked lifecycle authority. OneShot, Conditional, Control, and supervision can be programming constructions; none needs to be a privileged agent type.

This recommendation is provisional. The documents establish a concrete candidate to examine, not a proof of minimality, general compositional correctness, or a production-ready design. The user's requested scope ends with reviewed design documents.

That scope statement records the original foundational investigation. The separately authorized Phase 1 bounded research model and its evidence are recorded below; they do not authorize a production runtime or convert this baseline into an approved production specification.

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

## Phase 1 executable semantics findings

The [Phase 1 specification](phase1/01-operational-spec.md) instantiates selected decisions in an isolated Go test-only model. The [qualification report](phase1/02-qualification.md) records the bounds, exact commands, review findings, and counterexamples; [machine-readable evidence](phase1/qualification.jsonl) retains complete alphabets and traces. This is evidence about a finite model, not a promotion into production code.

Nine bounded positive searches checked 331 scenario states and 681 accepted transitions without a reported safety violation. All three deliberately removed protections were detected by the unchanged monitors. Two further traces disprove the interpretation of run cancellation as a permanent retry barrier. The positive counts are sums over fixed-alphabet graphs, with 58 unexpanded depth-boundary states; they are not an unbounded proof.

| Candidate decision | What survived the bounded examination | What remains unresolved |
| --- | --- | --- |
| D1–D3 Finite deterministic steps and frontier | One closed instruction per turn, separate prepare/commit, independent ready-continuation witness, fixed-version replay projection | Adequacy for arbitrary programs, optimal primitive count, general fairness and compositional closure |
| D4 Communication and D9 replay | Stable service binding after lost acknowledgment/retry; selected observations and complete immutable intents participate in replay | General capability security, retention expiry, full distributed snapshot/replay and migrations |
| D5–D7 Identity and authority separation | Generation/epoch/sequence and ownership revision checks remain distinct; the epoch mutant is detected | Ownership transfer, remote consent validation, delegation/revocation and stale-dispatcher incarnation fencing |
| D8 Commit/attempt/application/receipt boundary | External application can remain absent from journal knowledge; late evidence does not reopen closed computation; closure mutant is detected | Distributed atomicity, storage durability, endpoint fencing, full error/unknown reconciliation and transactional effects |
| D10 Cancellation scope | Baseline permanent cancellation is explicitly named **subject retirement**; run-only cancellation with both automatic-retry policies has different observable outcomes | Whether the public API should offer retirement, run cancellation, a separate stop-retry latch, or a combination |
| D11 Ordinary composition | A static three-subject Control program completes in 65 events using only ordinary instructions/effects | Dynamic creation, group cleanup, custodians, failure coupling and substitution proofs |
| D12–D13 Uncertainty and failure | Interruption preserves a recoverable run; deterministic fault and authorized abandonment end it; fresh retry creates a distinct generation | Failure detectors, available distributed authority and unbounded progress |

### Findings that changed the executable proposal

The initial model allowed tag-only receipt matching. Independent review supplied a trace where an ordinary message impersonated a receipt before any external attempt. The corrected language carries bounded effect references and requires exact intent correlation and observation kind; an independent monitor detects a faulty reducer that bypasses the guard.

Review also found two identity collisions: a receipt could overwrite an admitted ordinary message, and environmental ingress could shadow a generated Send's admission. The corrected model tags ingress/Send transport identity separately, then separates message/receipt inbox identity. Send receipts validate the full descriptor. The offending raw input IDs remain legal, with regression traces retained in the report. These findings strengthen the explicit interaction contract; they do not justify an extra privileged agent type.

Scope review fixed the minimum replay projection before qualification and clarified that every terminal status closes ordinary attempt authorization in this model. Its simplified terminal policy suppresses never-authorized mutations; custodial transfer and compensation are excluded. Modeling a trusted dispatcher with owner/revision checks does not establish the wider stale-dispatcher guarantee of the foundational candidate.

### Alternatives preserved after testing

`CancelRun(active g1)` permits a fresh run when retry of cancelled outcomes is enabled. Even when that policy is disabled, `Fault(g1) → CancelRun(g1) → Retry(g1)` permits a fresh run because the existing failed outcome is preserved. These are retained policy counterexamples, not defects removed by changing assumptions. Subject retirement supplies a permanent barrier; a separate run-only stop-retry latch requires its own semantics and tests.

Weak fairness versus stronger progress contracts, coordinated transfer validation versus acceptance of unavailable owners, and alternative observation/equivalence models remain open. No success count resolves them. The next authorized research question should name a specific missing contract; no runtime implementation follows automatically from this model.

Independent adversarial review returned **APPROVE for Phase 1 research-deliverable readiness** after the correlation/identity corrections and fresh verification of all 15 evidence records. The earlier findings remain in the qualification report. This verdict approves the research package, not a production specification or unbounded theorem.

## Phase 1B recursive composition findings

The [Phase 1B public contract](phase1b/01-composition-contract.md) defines admission, request/run correlation, distinct logical contributions, terminal outcomes, and externally visible unit resource effects before testing leaf and nested implementations. The [qualification report](phase1b/02-qualification.md) links complete source identity, fixtures, setup traces, alphabets, counts, negative controls, and independent review history. Phase 1's fifteen evidence records and regression inputs remain intact and reproduce exactly.

Eleven positive multi-subject searches checked 192 scenario states and 473 accepted transitions. Five guard-removal controls detect generation-correlation, logical-duplicate, and provenance failures in leaf/nested cases. A separate faulty-program control detects duplicate nested resource work for one public request. These are focused bounded graphs: setup prefixes reach 108 events, the searched suffix limit is ten, and two fresh-start boundary states remain unexpanded.

| Hypothesis or decision | Evidence from this increment | Remaining limit |
| --- | --- | --- |
| Ordinary compositions can use a leaf's interaction boundary | Outer → provider → worker uses the same interpreter and ordinary request/reply/effect operations; leaf and nested programs face the same public contract monitor | Conformance for the selected finite programs/windows; no general recursive-closure or contextual-equivalence theorem |
| Correlation belongs in explicit ordinary protocol state | Saved upstream envelope, pending downstream references, source generation/provenance, and accepted logical keys survive commit/replay | Adequacy for richer protocols, migrations, revocation and arbitrary outstanding-request sets remains open |
| Transport deduplication is sufficient for aggregation | Falsified: distinct Send identities can carry one logical result; an accepted-key check is independently necessary in the selected protocol | No universal join/quorum API is selected |
| A local resource count composes across nesting | Falsified by independent review: different downstream request IDs concealed two unit mutations for one upstream root | Resource obligations must refer to the full public request path; arbitrary resource/compensation contracts are untested |
| Closure and retry preserve isolation from old work | Combined receiver lifecycle/acknowledgment search, real successor computation, late old replies against eligible new receives, and leaf/nested stale-consumption controls | Recovery actions are injected by the environment, not evidence of ordinary supervisor policy or group cleanup |
| Independent work can proceed during an inner wait | A searched witness commits peer work while the nested provider awaits its worker | Possibility, not fair-scheduling or eventual-response proof |

Independent review initially returned **REJECT pending correction and re-review** for R1/P1, an insufficient resource-cardinality oracle. Its exact 37-event faulty-program trace remains executable. The corrected monitor follows full root request identity, including generation, and detects the second unit application without excluding the input or increasing bounds. The reviewer independently verified that correction and reproduced the final twenty composition evidence records. After final report/decision review, the independent reviewer returned **APPROVE for Phase 1B research-deliverable readiness**, with no blocking findings remaining. The original rejection and correction remain in the report.

The same-boundary hypothesis survives these selected tests only with an explicit public contract and explicit protocol state. Equal success scalars, syntactic role uniformity, or unchanged endpoint names are insufficient evidence by themselves. Ownership transfer, dynamic creation, cleanup custodians, production persistence/providers, dispatcher fencing, general behavioral equivalence, and unbounded progress remain separate research. This increment authorizes no runtime implementation.

## Phase 1C ordinary supervision findings

The [Phase 1C contract](phase1c/01-supervision-contract.md) adds ordinary representations for frozen child outcomes, physical suspicion, and immutable lifecycle requests/results. The [qualification package](phase1c/02-qualification.md) retains full fixtures, explored windows, counterexamples, exact source identity, and review history. Earlier phase documents, regression inputs, and evidence remain unchanged. Exact historical reproduction and current-source behavioral regression comparison are distinguished because Phase 1B's manifest hashes all current test sources.

| Decision or hypothesis | Evidence from this increment | Remaining limit |
| --- | --- | --- |
| Supervision can be an ordinary program | A finite Set/Receive/Branch/Emit program observes committed child failure, reserves its own budget, requests Retry, consumes its recorded outcome, and waits or fails; outer and inner use the same interpreter | Specific bounded policies and programs, not arbitrary recursive recovery or a privileged supervisor type |
| The baseline already exposed enough ordinary interaction | Falsified by the retained terminal-inbox regression: DeliverTerminal originally updated an external delivery ledger only | Typed ordinary outcome admission and lifecycle effect/result representation are necessary for this model; no claim of a minimal universal opcode set |
| Observing failure grants lifecycle authority | Falsified: static observation permission can coexist with denied lifecycle grants; the ordinary program consumes the retained rejection and escalates | Static trusted grants only; no general delegation, transfer, revocation, or distributed authority protocol |
| One lifecycle operation has one retained decision | Accepted and rejected results survive retransmission; the deduplication mutant turns a prior rejection into acceptance after target state changes and is detected | Atomic decision/result retention is assumed; implementation durability and distributed serialization remain open |
| Retry budget follows the worker generation | Rejected for the selected protocol: one explicit supervisory obligation preserves its spent count across worker generations and supervisor epochs | A fresh supervisor explicitly begins a new local obligation; hierarchy-wide/root business budgets need another ordinary protocol |
| A runtime should enforce every supervisor's policy | Rejected: a faulty ordinary branch exceeds its contract and is detected by the oracle, but the unchanged lifecycle runtime can authorize generation 3 | Contract-correct composition depends on correct ordinary programs as well as runtime enforcement |
| Physical unavailability is application failure | Rejected: suspicion is a distinct consumed observation; ordinary policy returns to waiting while same-generation work can continue | No failure detector, eventual recovery, or fairness guarantee |
| Retrying computation safely repeats business work | Falsified by the retained `business:order-7` witness: distinct execution intent IDs can mutate twice while the first receipt was unknown | Business identity remains explicit; endpoint-specific reconciliation or permission to repeat is still required |
| Restarting a nested supervisor repairs its subtree | Not established: outer can recover/escalate through public outcomes; a fresh inner naming historical worker generation 1 receives stale-generation rejection when generation 2 already exists | Current-child discovery/handoff and budgets across fresh inner obligations remain unresolved |

The new lifecycle effect protocol checks source dispatch authority separately from target rights. Target generation/revision and lifecycle eligibility are validated when the operation decision commits. A program cannot mutate another run directly, and runtime code does not inspect private supervisory state to select policy. Cancellation remains explicit: CancelRun preserves an already committed failure and is not a permanent barrier; RetireSubject prevents future generations. Both run-cancellation retry-eligibility settings remain modeled alternatives.

Independent review initially withheld approval for a missing nested Interrupt event and a witness-oracle defect: one global interruption flag could combine different executions. The exact false-positive suffix is retained. The correction requires an actual interruption and subsequent reactivation of the same execution with the same pending operation, and adds the missing H event without increasing bounds. Independent re-review confirmed both findings addressed, reproduced all 32 final records byte-for-byte, and returned **specification PASS, quality PASS, APPROVE for Phase 1C bounded research-deliverable readiness**. The original findings and exact regression remain in the report. This approval is not an unbounded correctness result.

Ordinary recovery policy now has an executable research boundary to challenge. Atomicity, provenance, finite retention, and static authority remain assumptions. Dynamic creation, ownership transfer, cleanup custodians, distributed consensus, provider integration, production persistence/runtime/SDK, unbounded liveness, and arbitrary-depth equivalence remain excluded. Stop at this research package and its draft PR.

## Phase 1D single-host durable executor findings

Phase 1D explicitly authorizes a first implementation profile. The [durability contract](phase1d/01-durability-contract.md) and [qualification package](phase1d/02-qualification.md) scope its evidence to one host, one active executor per store, static subjects/grants, immutable program artifacts, and controlled process crashes. Earlier research source, counterexamples and saved evidence remain intact. This is an internal runnable executor and CLI, with no public SDK or general production-readiness commitment.

| Decision | Implemented evidence | Remaining boundary |
| --- | --- | --- |
| Local persistence authority | SQLite WAL/FULL, verified connection settings, separate transactional ledgers, local file-lock exclusion and activation renewal | Local filesystem/storage assumptions; no power-loss, disk-failure or distributed-fencing qualification |
| One recovery decision survives process death | An ordinary inner program retains its committed Retry, budget reservation and pending operation across acknowledged SIGKILL after the target decision; the successor executes real instructions | Same supervisory execution; fresh-supervisor child reassociation and hierarchy-wide budgets remain separate work |
| Source and target authority are distinct | Retained source authorization precedes lifecycle target grant/generation/revision checks; observing failure does not confer Retry rights | Static grants and trusted local authority only; no ownership transfer or general delegation |
| Rejection is a durable operation result | A valid Retry rejected while the target is active remains rejected after actual target failure; a new operation can accept | Operational capacity errors without a committed decision remain errors, not invented semantic rejections |
| Recovery and replay have different authority | Live open validates records and renews active epochs; read-only replay reconstructs without lifecycle or external dispatch | Replay shares semantic evaluation with execution; it is consistency checking, not an independent oracle |
| External application remains a separate fact | Independent process/store, pre-transport authorization, separate receipt recording/settlement, explicit unknown outcomes and late evidence | Idempotence is per retained request key; a fresh execution-scoped intent can repeat business work |
| Supervision remains ordinary computation | Versioned finite instructions, explicit inputs, ordinary correlation/budget state, nested completion/escalation, no runtime role dispatch | One program counter per run in this profile; no arbitrary-depth equivalence, dynamic provisioning or unbounded progress claim |
| Cancellation alternatives remain explicit | CancelRun preserves an already terminal result; cancelled retry eligibility is configured; accepted RetireSubject prevents future generations | No universal cancellation policy, cleanup custodian, group cancellation or new stop-retry latch |
| Independent checks can expose implementation faults | Unchanged record checks detect lost rejection, reset recovered policy state and speculative external dispatch | These concrete controls do not prove that every possible defect is detected |

Review retained three implementation/qualification defects and their original inputs: generated-output amplification, unqualified transport-identity collision, and insufficient crash-acknowledgment identity checking. Their corrections add an explicit operational output bound, trusted admission domains, and exact process/run/operation correlation. None requires a privileged supervisor type. Pre-fix sources and actual failing probes remain in the Phase 1D package.

The runtime's ordinary receive and subsequent clearing of a local pending field can be separate commits, whereas the research program clears its pending field in its receive turn. Selected model/runtime checkpoint comparisons must disclose that intermediate state and its recovery behavior. Equal final values and replaying one reducer twice are not general conformance evidence.

The pinned SQLite driver, effective engine identity, exact contributing source identities, complete source/dependency snapshot, commands, actual process exits and inspected records are part of qualification. Final whole-phase review and publication status are recorded in the Phase 1D report; task-scoped approvals alone do not establish completion.

## Phase 1D.1 independent progress correction

The [dispatch contract](phase1d1/01-dispatch-contract.md) and [qualification](phase1d1/02-qualification.md) narrow the earlier progress claim: the original automatic loop waited for HTTP before selecting a peer. Its retained unchanged held-request regression fails. A timeout followed by peer success did not establish progress while the request was held.

One automatic Run now owns a bounded transport set (default 2), while one authoritative loop offers local computation, internal/lifecycle delivery, completions and launches regular opportunities. Workers carry immutable correlation and do no storage writes. Saturation leaves durable backlog and permits peer progress even at capacity one. The actual CLI peer and its caller commit before the exact held response is released; reordered responses remain correctly correlated.

MaxSteps, MaxDispatch and transport capacity remain separate operational bounds. Run cancellation cancels/join workers and accounts candidates with a separate cleanup context; it does not cancel logical executions. Active Store write transactions own rollback to preserve the pinned connection through cancellation. Authentic old-epoch evidence is validated against retained authorization and may settle after source closure. No new language primitive, role-based policy, lifecycle authority, schema or dependency is introduced.

Retained counterexamples include synchronous scheduling, inherited-pipe observer setup, cancellation discarding a pinned connection and post-commit step undercounting. The latter two were exposed by the new shutdown qualification. Independent review and probes are retained with the full source/dependency archive. Earlier phase records and regressions remain unchanged.

This does not establish unbounded fairness, a hard real-time Close deadline, arbitrary simultaneous direct dispatch outside Run, or exactly-once business effects. The existing cross-store operation-key namespace limitation, single-host/static-subject profile, one PC per run, distributed fencing, migration/compaction and fresh-supervisor child reassociation remain separate work.

Final review additionally exposed a reporting defect: a durable valid receipt was still labeled Unknown when its separate local settlement failed. The correction validates retained outcome evidence before reporting Unknown, preserves pending delivery/error state, and verifies non-idempotent reopen without redispatch. The pre-fix source/input and unchanged independent regression are retained. This changes reporting, not business idempotence or the separate transaction boundaries.


## Phase 1E authoring and effect boundaries (increment in progress)

The [Phase 1E contract](phase1e/01-authoring-effects-contract.md) keeps the existing interpreter and single bounded dispatch owner. Public authoring is intended to compile named finite operations into retained artifacts; model and tool work must remain separate intents selected by ordinary computation. Later Phase 1E claims remain pending until its qualification package records their evidence.

The cross-store operation-key limitation is corrected in the reviewed namespace slice. A newly provisioned schema-version-2 store atomically retains a cryptographically random 256-bit namespace. External keys encode key version, namespace and local intent identity; message, journal and lifecycle identities stay local. The unchanged two-store input exposed one mutation under the old key and two independent mutations under the corrected key. The [original input/source and review](phase1e/evidence/task1/manifest.json) remain available.

Reopen and rename preserve operation identity. A copy retains that identity; concurrent independent use of copies remains unsupported. This increment explicitly rejects version-1 runtime stores before activation or dispatch rather than migrating or re-keying committed work. Historical source archives remain the reproduction path for their original evidence. Namespace separation does not supply business deduplication across fresh application decisions.


Independent compiler review supplied two accepted definitions whose Run and Replay agreed on the wrong result: a reachable trailing label changed the selected branch, and invalid UTF-8 variable names collided after JSON encoding. Their [retained probes](phase1e/evidence/task2/review-round1.md) reinforce the existing distinction: shared-reducer replay checks historical consistency, while an independently specified public behavior is needed to test compilation semantics. Reachability-based lowering and consistent UTF-8 identifier rejection correct both cases; scoped independent re-review accepted the fixes. The counterexamples remain part of the evidence.


The reviewed adapter slice uses schema 3 with external key version 2; the intermediate schema-2 layout is also rejected before activation. Corpus bytes, prepared requests and results are retained in separate durable records. Source authorization, physical reservation, result recording, settlement and ordinary consumption remain separate commits. A reserved adapter attempt with no recorded result stays Unknown and is never automatically repeated; known results settle without a registry. Fixture and adapter transport share the existing bounded Run owner.

The supplied OpenAI adapter performs one Responses request. A generic record serializer carries question/evidence fields selected by ordinary computation; it introduces no formatting or orchestration opcode. Credentials remain runtime-only, and response evidence may be conservatively omitted when sanitization cannot safely inspect it. Independent review retained credential-encoding, protocol-key alias, and Inspect reverse-edge counterexamples; their [fixes passed scoped re-review](phase1e/evidence/task3/rereview-round1.md). Adapter Prepare/Execute implementations remain trusted Go infrastructure, with purity/cancellation obligations rather than sandbox enforcement. Application-level and process-kill qualification remain pending at this checkpoint.
