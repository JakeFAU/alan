# Phase 1 bounded operational specification

Status: bounded research profile. Initial scope was frozen before implementation; later review clarifications are recorded below. This instantiates a finite part of the [foundational candidate](../01-foundations.md); it does not approve that candidate for production. Exact fixture bounds and disclosed scope amendments appear in [Qualification](02-qualification.md).

## Choices and alternatives

The model uses a finite frontier, one serialized current run per logical subject, asynchronous message admission without FIFO, a fixed immutable program version per run, explicit attempt authorization, and complete retention within a test. These adopt D1, D2, D4–D9, D12, and D13 for bounded examination. A closed instruction language is the finite-step strategy for this model, not a future SDK choice.

The baseline D10 operation is named `RetireSubject(a, w)`: permanently close logical subject `a`, end its active run, suppress unattempted mutating intents, and prevent new generations. `CancelRun(a, g, w)` is an alternative operation: close only named run `g`, preserve the subject, and never redirect cancellation to a later generation. Automatic retry after a cancelled run is tested with both `retryCancelled=false` and `retryCancelled=true`. Retry otherwise applies only to failed or abandoned runs. Success is not automatically retried. These profiles are compared; none is designated the accepted production policy.

In this profile, a late `CancelRun` preserves an already committed failed/abandoned outcome, so automatic retry remains eligible even with `retryCancelled=false`. That flag governs runs whose terminal outcome actually becomes cancelled. A separate stop-retry latch could provide a stronger contract without retiring the subject, but is not modeled; omitting it is an explicit limitation to expose through a witness, not an assertion that users should prefer this behavior.

Ownership transfer is not modeled. The current principal and ownership revision are supplied in fixtures, and stale or unauthorized lifecycle requests must be rejected. This tests validation of a serialized authority record, not cross-partition consent, retirement validation, ownership cycles, or distributed transfer. General delegation/revocation and adversarial runtime/storage corruption are excluded.

Attempt authorization is issued by a trusted dispatcher using the current owner/revision grant. Dispatcher incarnations, dispatch takeover, and stale-dispatcher fencing are not modeled. Reactivating a reducer changes its commit epoch, not the dispatcher's authority. Thus stale-activation checks qualify reducer commits only; they do not establish the baseline's wider stale-dispatcher or external-fencing guarantees.

The Control example uses three statically provisioned subjects. It exercises ordinary messaging, waiting, effects, and result aggregation using the same instruction interpreter for all three. Dynamic creation, group cancellation, and a surviving cleanup custodian remain outside the tested claims.

## State

The model configuration contains distinct components:

- **Subject authority:** logical identity `a`, owner, ownership revision `w`, current generation `g`, permanent retirement flag.
- **Run journal:** execution identity `(a,g)`, immutable program version, activation epoch `e`, committed sequence `n`, active/succeeded/failed/abandoned/cancelled status, finite data, frontier, inbox, history, and committed intent ledger. Ended runs are retained.
- **Worker proposals:** version, generation, epoch, sequence, selected continuation/observation, and an evaluated successor proposal. Proposals are not committed state and can be discarded or become stale.
- **Message admission ledger:** tagged transport origin plus stable identity and immutable payload/target description, original run binding, and acceptance result. Environmental ingress and generated Send intents have disjoint identities even when their raw text IDs match. Acknowledgment delivery can be omitted or delayed independently of admission.
- **Attempt ledger:** intent identity, attempt identity, and durable authorization evidence. Intent commitment is not attempt authorization.
- **External resource:** applied operation keys/parameters and results or mutation counts, held separately from the journal. This is oracle-visible world state; ordinary reduction and lifecycle policy cannot inspect it.
- **Receipts and frozen outcomes:** retained correlated evidence, duplicate disposition, terminal result and its fixed notification route/delivery state.
- **Checker state:** immutable facts needed to detect stale commits, changed service bindings, post-closure authorization, or terminal rewrites. Checker facts must not serve as hidden implementation guards.

Logical identity, generation, activation epoch, and ownership revision are distinct values even if small integers represent them. Effect identity is derived from run/commit/ordinal; attempt identity is separate. A new generation has new execution-scoped effect identities and therefore does not automatically deduplicate the same business task across runs.

Internal inbox identity is a tagged namespace: message observations derive from `(message, transportIdentity)` and effect receipts from `(receipt, intentID)`. External IDs remain ordinary data, even when their text resembles an internal receipt key. The namespaces must be injective and disjoint; receipt recording cannot overwrite an admitted message. Admission validation retains and checks message kind, label, payload, and correlation as well as its original run binding.

For transport, the message identity in that construction is itself tagged as `(ingress, rawID)` or `(send, intentID)`. Admit/Acknowledge select the origin explicitly. A Send receipt may use only its own send-origin admission and must match the complete immutable message descriptor. Environmental ingress cannot supply acceptance evidence for a generated Send with a coincidentally equal raw ID.

## Finite local language

Programs are immutable finite instruction arrays. One selected instruction executes per local turn. The instruction families are literal assignment, conditional branch/jump, fork of a continuation, correlated receive/wait, effect/message emission, continuation completion, terminal success, and defined program fault. Data operands, target PCs, message labels, and effect arguments range over explicitly bounded fixture values. There are no host-language callbacks in agent programs.

Two bounded effect-reference slots accompany the data registers. Emit/Send store their newly derived intent identity in the instruction's selected reference slot. A receipt receive specifies that slot and requires both receipt origin and exact correlation to the stored identity, in addition to its label. An ordinary receive requires message origin. A label such as `receipt:charge` is not authority and cannot turn an ordinary message into receipt evidence. These reference slots are part of committed and replayed state.

| Instruction | One-turn result |
| --- | --- |
| Set(register, literal) | Assign the literal and advance the selected PC. |
| Branch(register, literal, target) | Jump to target when equal, otherwise advance. A known constant register can express an unconditional jump. |
| Fork(target) | Advance the selected PC and add a fresh continuation at target. |
| Receive(tag, kind, reference, register) | Consume the selected matching observation, add its bounded integer payload to the register, and advance. Receipt kind additionally requires exact reference-slot correlation. |
| Emit(tag, literal, reference) | Commit one external mutation intent and place its identity in the reference slot. |
| Send(target, generation-or-service, tag, literal, reference) | Commit one message intent and place its identity in the reference slot. |
| Done | Remove only the selected continuation. An empty frontier alone is not success. |
| Success / Fault | Set the explicit run outcome, clear the frontier, and freeze the outcome/route for later notification. |

A continuation has a unique identity and program counter. A receive instruction is enabled only for an unconsumed matching observation; other continuations remain independently selectable. A ready instruction consumes a logical tick. Selection is an explicit event recorded with the resulting commit, so completion order and receive choice are not recovered by iterating a host map.

The interpreter checks program bounds and the defined instruction cases. Model capacity limits stop or reject exploration extensions with an explicit count/reason; they are not success or semantic proof of termination. A defined `Fault` instruction creates a deterministic application failure. Invalid host memory access or an interpreter bug is a failed test, not an application outcome.

## Transition rules and atomic boundaries

Each row is one model event unless explicitly described as pure evaluation. The Go model enumerates these events serially to explore concurrent interleavings; Go goroutines are unnecessary.

| Rule | Preconditions | State change and atomicity |
| --- | --- | --- |
| Prepare | Active run, matching selected continuation and available observation | Purely evaluate one instruction from a snapshot; retain a proposal with `(v,g,e,n)`. No inbox consumption, journal mutation, or dispatch. |
| Interrupt | A worker proposal exists | Discard only the proposal. The observation remains unconsumed and the run recoverable. |
| Commit | Current generation/epoch/sequence match, run active, subject not retired, selected observation still available | Atomically consume the observation, install successor data/frontier, append history, record immutable intents, and advance sequence. Terminal instructions also freeze outcome and notification obligations. |
| Admit | Valid send scope and fresh message ID, target open and capacity available | Atomically record payload identity, run binding, inbox insertion, and admission result. Service targets bind once to the current run; run targets name it directly. |
| Duplicate admission | Message ID already known with identical descriptor | Return the original admission/binding even if generation changed or the original run closed. No second inbox insertion. Conflicting descriptors are rejected. |
| Acknowledge | An admission result for environmental ingress exists | Mark that the external sender observed it; loss is represented by omission/delay. Admission does not depend on acknowledgment. For a modeled Send intent, recording its admission receipt instead performs the sender-visible acknowledgment step. |
| Authorize attempt | Committed unresolved intent, current dispatch authority, source run active and current, subject not retired, intent not suppressed | Atomically retain a unique attempt authorization. Every terminal run status forbids a new ordinary authorization. The closure check and authorization have one serialization point. A negative control deliberately splits or omits this protection. |
| Apply externally | A specific attempt was authorized and has not yet physically applied | Mutate only the external resource, retaining world-side result. Application can occur after source closure if authorization preceded it. No journal receipt is implied. |
| Record receipt | Authentic retained intent/attempt correlation with an available result | Retain evidence independently of current worker epoch or run status. For Send, delivery of the admission result marks acknowledgment observed. A late receipt cannot resume a closed program. Duplicate receipts do not re-consume or rewrite known outcomes. |
| Reactivate | Current active run, lifecycle authority, epoch bound permits | Grant a new epoch in the same generation, retaining the prior journal. Old proposals remain available to attempt stale commits. |
| Abandon | Current run active and caller has current lifecycle authority | Atomically end/fence the run and freeze abandoned outcome; suppress never-authorized mutating intents. Authorized attempts may still apply. |
| CancelRun | Caller has current lifecycle authority over the named subject and retained run exists | Close that run if active; preserve an existing terminal result; suppress its unattempted mutations. No subject tombstone and no cancellation of a later run. |
| RetireSubject | Caller has current lifecycle authority | Monotonically retire the subject; close the current active run and suppress unattempted mutations, retaining all old outcomes/evidence. Reject later retries. |
| Automatic retry | Subject not retired, expected current generation ended in an eligible status, caller authorized, generation bound permits | Create fresh `g+1` with initial program state; retain old journal, message bindings, and pending authorized outcomes. Cancelled runs are eligible only in the explicitly selected policy variant. |
| Deliver terminal result | A frozen notification exists | Deliver its recorded value/route after closure or a later generation without reopening computation or authorizing arbitrary work. |

A physical interruption is not application failure, and suspicion alone neither ends a run nor starts a generation. Receipt loss is omission of `Record receipt`, not a rollback of `Apply externally`. No scheduler transition may consult external application state to infer a failed effect. The journal remains unresolved after authorization until evidence is recorded; no receipt means unknown, not known failure.

For this bounded profile, every terminal transition suppresses never-authorized ordinary mutations and freezes a terminal notification. Already authorized effects remain unresolved until receipts; the profile does not implement transfer of their responsibility or compensation. This is a disclosed simplification of the baseline's wider terminal-disposition alternatives. A notification is generated from a committed terminal outcome and fixed observer route, not from an unrestricted `EmitNotification` instruction.

An idempotent resource retains immutable parameters/results per operation key for the entire search. A separate non-idempotent resource profile allows repeated physical attempts to repeat mutations. Journal safety does not imply that the latter mutates once. No authoritative external fencing, query protocol, or cross-resource transaction is assumed.

The mutation-resource profile models successful application and withholding its receipt; it does not model a known rejection/error return or conflicting authenticated receipts. Send can observe acceptance or rejection of admission. Unknown is represented by an authorized unresolved intent, not a complete adapter outcome lattice. Those limitations prevent generalization to every external-outcome protocol.

## Replay

Replay operates on a fresh shadow of a retained run and the immutable bound program. Its minimum reconstructed projection is program version, committed sequence, data and frontier, activation epoch, run status, consumed observation identities, and complete immutable emitted intent descriptors including frozen terminal notifications. It rebuilds these from the initial run and recorded selected observations/lifecycle events; copying final fields into the shadow is not replay. It must reject a different program version.

Pending inbox contents, service-admission bindings, mutable attempt/receipt evidence, suppression, and notification delivery are validated separately against their retained records and transition invariants. They are not included in a claim of replaying an entire distributed configuration. The qualification report must distinguish this projection from settlement validation.

Replay never authorizes, sends, or applies historical effects. The test compares the external resource and dispatch/attempt counts before and after replay. Replaying state is distinct from the live recovery events that can later reconcile an unresolved intent. Full composite snapshots, migration, external provider reproducibility, and archival retention are excluded.

## Invariants and progress claims

**Structural validity:** IDs reference existing subjects/runs; continuation IDs are unique; PCs and finite domains are valid; intents reference committed history; receipts reference authorized attempts. Structural checks do not establish safety by themselves.

**Safety:** every accepted commit uses current generation, epoch, sequence, and an open run; no pre-commit effect escapes; each admitted message has one stable run binding; no ordinary authorization occurs after the applicable closure; suppression never asserts non-execution for an authorized attempt; terminal outcomes and retirement are monotone; duplicate and late evidence cannot advance unrelated work; replay reconstructs the claimed projection without dispatch. These are checked over actual event effects, not merely by asking whether the guard function returned true.

**Progress evidence:** exhibit a commit by a ready continuation while another waits, a bounded complete Control execution, and terminal-notification delivery after closure. These are reachability/enabledness witnesses. No fairness or eventual-delivery theorem follows from bounded safety exploration. Stuttering or an environment that withholds a receipt forever is permitted by the abstract system.

## Exploration and falsification

Use deterministic event ordering and scenario-specific breadth-first or exhaustive bounded searches. Include delayed/duplicate/reordered admissions, loss of acknowledgments, interruption, reactivation, stale proposals, retirement/cancellation/retry races, endpoint application without receipt, and late settlement. Search state identity must include all future-relevant machine and monitor state; do not merge histories on final payload alone.

For every explored successor, check invariants before any state deduplication. Report depth limits, event alphabet/fixtures, explored states and transitions, rejected actions, merged revisits, and boundary cutoffs. A state-count resource cap must fail qualification rather than silently declare success. Search scenarios may use distinct small domains; their bounds must be explicit and actual counts reproducible.

Mandatory negative controls remove epoch validation, closure/authorization serialization, and stable service-to-run binding. The same safety monitors must find a counterexample for each mutant, and tests fail if they do not. Retain the shortest found trace with the violated invariant. Wrong-policy expectations about run cancellation must be reported as policy counterexamples, separately from broken-guard counterexamples.

Additional focused tests cover stale ownership revision, competing proposals at the same sequence, deterministic fault versus interruption, wrong-version replay, idempotent versus non-idempotent effects, and exact notification correlation. No counterexample may be eliminated by strengthening an assumption without a recorded scope amendment and its cost.

## Scope clarification record

Before executable qualification, independent scope review identified two ambiguities in the first specification draft. Ordinary attempt authorization now explicitly requires an active source run, so success/failure/abandonment cannot accidentally permit new attempts after the simplified terminal disposition. The cost is that continued mutating recovery needs a separate custodian protocol, which remains excluded. Replay's minimum projection is now fixed above; settlement/admission validation is explicitly separate. These tighten the stated profile and its evidence requirements; they do not remove a discovered execution trace or assert a stronger external guarantee.

Early code review found an actual counterexample to correlated receive: a tag-only matcher accepted an ordinary message as an effect receipt before any attempt. The correction adds explicit origin and expected-intent operands plus independent consumption checks; it does not exclude the offending message from the model. Review also made dispatcher-incarnation fencing an explicit exclusion, rather than treating reducer epoch validation as proof of every dispatch guarantee. Both findings and their costs are retained in the qualification report.

A second code-review counterexample admitted an external message whose text ID matched a generated receipt inbox key. Receipt recording overwrote the accepted payload. The corrected specification requires disjoint internal namespaces and retention checks; it does not ban that external ID. Send receipt recording also explicitly marks the acknowledgment observed, making sender knowledge consistent with recorded evidence.

The same review traced a second layer of that identity defect: environmental ingress could shadow a generated Send with the same raw transport ID, and the wrong admission could satisfy its receipt. Transport/admission identities are now origin-tagged and Send receipts require descriptor agreement. The colliding external ID remains legal in the regression test.
