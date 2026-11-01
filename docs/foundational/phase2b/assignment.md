ALAN Phase 2B: Revocation, Local Quiescence, and Recovery Authority
For a fresh Codex conversation: Execute this complete research and implementation assignment through independent review, manuscript reconciliation, and a draft PR. Begin with the short specification gate in Task 1, then continue autonomously. Do not stop after returning a plan or finding the first easy counterexample. Use independent workers for separable model, implementation, fixture and audit work; use superpowers:subagent-driven-development when available. No merge or release.

Goal: Implement and qualify a precise two-stage local authority contract: closure prevents new dispatch admissions, and a separate quiescence acknowledgment accounts for previously admitted work, without inventing knowledge about the external world or sacrificing unrelated local progress.

Architecture: A short source-and-scope admission gate coordinated with the single Store owner; separately tracked lifetimes for each physical dispatch opportunity through completion accounting; and an additive public host/runtime quiescence observer for an exact revoked authority. Compare an independent finite model with genuine public execution, persistent endpoint histories, and process-loss experiments.

Tech stack: Existing Go/CGO/SQLite, public author, runtime, effects and schema, the Phase 2A controlled endpoint/process apparatus, and independent standard-library Python audits. No live providers or new dependencies by default.

Specification: This assignment, archived unchanged as docs/foundational/phase2b/assignment.md, plus the reviewed contract frozen in Task 1. Read AGENTS.md. The two-stage candidate is the selected direction; precise public signatures, revocation predicates and version treatment must be source-grounded and frozen before implementation. Routine scoped decisions do not require another user confirmation.
1. Continuity and baseline
At preparation on 2026-11-01, PR #32 is merged. The integration reference is 72e406b73b05a3ed4a6e8b37801ab6e0f0a73209; its Phase 2A delivery head is 24485a2b4f5bb3af45027995c6e6d8f34e0acb2d, and its experimental base was 18d35a2b312cf7180926d4b4cc5c7c2094cd68e2. Fetch and verify current main, ancestry, origin and worktree state before work. Preserve unrelated newer changes. Do not rename or split the repository.

Series 1 remains concluded as an ESTABLISHED FRAGMENT for the exact finite pure owning-tree domain. Do not reopen its general congruence question as startup work. Phase 2 asks whether retained computation remains a truthful and useful basis for external action and recovery.

Read these Phase 2A sources first:
- docs/foundational/phase2a/02-protocol.md, 03-correspondence.md, 04-experiments.md, 04-process-experiments.md, 05-parent-boundary.md, 06-qualification.md, 08-review.md, and 09-decisions-and-next.md.
- The final review and input index under docs/foundational/phase2a/evidence/; use the runbook for exact historic commands. The final index SHA-256 is c5fc2c55087cdc206fc4af578a90654b82f027714ef02985d0e40081d8ce6855.
- Manuscript Pillars IV and V in both Markdown and TeX. The former physical-launch clause was withdrawn. Do not restore it by changing the definition of physical launch.
What Phase 2A established within its scope
Finding	Consequence for this run
A held reserved raw/adapter worker first called after actual source cancellation; raw idempotent work could issue another POST after closure.	Original permission and reservation are historical facts, not a fresh current-authority fence.
U0/U1 had exactly equal surviving local records and different independent application histories.	Refusing another dispatch cannot reveal whether the original operation applied.
Raw idempotent recovery completed both branches with one application; non-idempotent raw and adapter recovery withheld repetition.	Preserve useful supported recovery and distinguish a route restriction from an endpoint information obstruction.
A retained transport error could coexist with an applied mutation.	Result availability and world knowledge remain different propositions.
A child could privately fail, its parent could select failure, and an outstanding worker could subsequently mutate.	Local termination and publication do not by themselves settle world obligations.
An ordinary peer performed useful later work at physical capacity one.	A stronger authority mechanism must be tested for progress costs.


The prior packet reports 32 finite profiles, 1,300 states and 1,981 edges; 132 normal/race leaves per regime include one inert helper entry. It reports 12 genuine executor kills/recoveries and 28 replay comparisons per regime. These are prior qualified results, not fresh runs by this assignment. The original paired cuts excluded in-flight packets; the new campaign must include one deliberately delayed external application across process loss.

Phase 2A corrected an overbroad manuscript claim. That alone does not prove the old runtime had an implementation defect. This phase proposes and tests a stronger, explicit operational contract with costs and compatibility consequences.
2. Selected design and the alternatives it must distinguish
Implement one two-stage candidate, with small adverse controls for the rejected shortcuts. Do not build three general cancellation frameworks.
Approach	Meaning and tradeoff
Historical reservation remains executable after closure	Describes the tested baseline in the relevant cases; it offers no new revocation of exposed work. Preserve that evidence.
A short current-authority admission gate	Revocation and admission have a definite local order. A worker admitted first may physically enter later. This is the first stage of the selected contract.
A lock spanning the entire HTTP/Execute call	Can delay a cancellation acknowledgment until the call ends. Its responsiveness and scheduler cost must be exposed; it is not a free physical-launch guarantee.
Short gate plus separate local quiescence acknowledgment	Revocation completes without waiting for the endpoint. A separate observer waits for admitted work and available evidence handling. This is the selected complete candidate.


The research question is:
Which precise authority and quiescence guarantees can the public executor implement while preserving useful execution, truthful uncertainty, and a justified rule for future recovery actions?

An admission gate is useful only if it actually rejects a newly considered dispatch after the relevant revocation, including an exposure of an old idempotent reservation. It must not reuse the first authorization as its current decision. It does not establish that every physical call occurs before logical closure.

The separate quiescence observation makes the remaining cost visible. It may depend on previously admitted work completing or withdrawing, cooperative context cancellation, and accounting service. A timeout is an unsuccessful observation attempt, not a quiescence certificate.
3. Required contract distinctions
3.1 Events, identity and protected boundary
Freeze separate events for original permission, reservation/debit, admission of a particular worker opportunity, physical call entry, endpoint ingress/application, response acquisition, durable recording, settlement, consumption, effective revocation, and successful quiescence acknowledgment.

Use these names in the argument, regardless of implementation names:
- R: effective revocation for an exact original generation or scope under the chosen policy.
- A(w): current-authority admission of one physical worker opportunity w.
- L(w): the protected runtime entry: raw http.Client.Do, or the registered adapter's Execute call.
- H(w): completion of the existing handling of that opportunity and any available returned evidence, including required recording/settlement or an explicitly surfaced error.
- Q: successful local-quiescence acknowledgment for the exact already-revoked authority.
Execute entry is not proof of an HTTP request, and HTTP call entry is not proof of ingress or application. A general adapter's internal behavior is not made transparent by wrapping its entry. Use the controlled adapter's separately audited single-POST behavior for endpoint correspondence; do not promote that fixture property to every adapter.

For the selected contract:
1. R precedes A(w): deny admission; this opportunity makes no protected call.
2. A(w) precedes R: the opportunity is already admitted. R < L(w) may still occur. Retain a genuine ordered witness; it remains a counterexample to the stronger physical-launch statement.
3. Q succeeds: no covered runtime-managed opportunity can subsequently enter the protected call boundary, and available completions have finished their required handling. A queued authentic result cannot be discarded to make Q succeed.
4. Remote application remains separate: previously transmitted work may apply after R, Q, or executor death. Neither acknowledgment establishes remote nonapplication, rollback, business completion, or a right to retry.
Track each admitted opportunity separately. One logical operation can have multiple permitted idempotent exposures, including direct calls. A single Boolean or map entry per operation must not collapse two live workers into one. Every new exposure requires its own current gate decision; old admission tokens are not transferable or reusable after recovery.

The admission decision and registration of its covered lifetime must be atomic with respect to R and Q under the synchronization protocol. There must be no interval in which work has been admitted but a revoker or observer can see an empty set and acknowledge Q before that work becomes registered. Include this race in the model and concrete controls.
3.2 Revocation policy is a semantic decision
The starting candidate requires the effect source's original generation/revision to remain current, the source not to be retired, current source eligibility, and the applicable owning scope to remain eligible. Freeze the exact predicate and refusal reasons before implementation.

Actual CancelRun, AbandonRun, retirement and supersession of the original generation must prevent a new admission covered by their authority. An accepted already_terminal no-op is not a new closure event. An old-generation quiescence acknowledgment makes no promise about an authorized later generation.

Owner shutdown is separate from durable source/scope revocation. Store.Close prevents new admissions by that owner and drains its covered work; it does not itself cancel an open logical source. A subsequent legitimate Open may admit work for that still-eligible source. Neither an owner-closing flag, an inactive Run, nor an empty worker set satisfies the durable revocation premise for source/scope Q. Include an open-source Close/Open comparator that preserves this distinction.

For ordinary successful End and private Failure, explicitly decide whether current terminal source status revokes previously reserved but not newly admitted work. A current-open-source gate is the proposed conservative candidate, but it is a behavioral change, not an implementation-neutral interpretation of Phase 2A. Evaluate captured obligations and report any newly pending aggregate. If preserving a stated contract requires a different predicate, resolve and record that choice at the specification gate. Do not silently change the policy after seeing tests.

Preserve the existing distinction between the top-level resource root and a successful nested invocation freeze. At the pinned base, compositionScopeClosed rejects a non-open top-level root. Along nested invocation ancestry it checks non-success freezes, retirement closures and retained composition_results; a successful unpublished nested freeze alone does not close it. That frame may still own captured child admissions, descendant outcomes or external evidence. Do not replace this predicate with “every terminal ancestor revokes.” Require a positive nested example in which an eligible descendant completes under a successfully frozen but unpublished invocation ancestor.

Internal bound replies and authentic late effect evidence have their own existing rules. Revoking new world action must not revoke evidence merely because it arrives later.
3.3 Quiescence is an explicit public observation
Add one minimal public host/runtime observation or wait surface. Its identity must bind an exact generation/scope using existing genuine identities; a bare subject name that silently follows the latest generation is insufficient. It observes authority already closed through legitimate ordinary lifecycle or control transitions. It does not itself grant cancellation, retry, reconciliation or new effects.

Freeze its precise signature, target-construction rules, result meaning, errors and execution requirements in Task 1, then exercise it from a separate module importing only public packages. Distinguish:
State	Required interpretation
Authority remains eligible	Temporary idleness is not a lasting no-future-entry acknowledgment.
Authority revoked; admitted worker held before call	R has completed; Q remains pending.
Worker returned; completion waiting for accounting	Q cannot succeed merely because a physical slot was released.
Available result recorded and settled	Handling can finish without selection by a closed continuation.
No receipt, or adapter returns unknown	Handling may finish as unresolved under the existing protocol; do not fabricate a result.
Accounting/settlement error	Surface the error; do not silently convert an empty worker count into successful Q.
Wait context expires	No successful Q; original revocation and evidence obligations remain.
Process lost volatile evidence	Reopen cannot reconstruct missing bytes. Preserve uncertainty and handle only genuine surviving evidence.


Waiting must not hold Store.mu, a SQLite transaction, or the synchronous ordinary lifecycle turn inside Run. The scheduler must remain able to make unrelated ordinary commits, account results and settle evidence. Existing physical capacity is still a real constraint: one occupied effect slot does not promise a second simultaneous external call.

Do not call this an ordinary program-visible uncertainty feature. Host/runtime observation and an authored parent's selected invocation envelope remain different interfaces. Any new ordinary-agent observation would be a separately specified extension, not an implicit part of this task.
3.4 Evidence, resources and version boundaries
Keep the first authorization, original reservation, request/preparation, operation key and semantic effect debit immutable. A later gate refusal is a different fact. It must neither rewrite historical permission nor refund a spent reservation or charge it a second time.

In particular, denying an idempotent recovery exposure does not show that the original operation never applied. Do not synthesize an operation-level source_denied result from that later refusal and thereby erase U1 uncertainty. If new diagnostic records are needed, specify their proposition and binding separately.

Preserve exact original request bytes on same-operation raw recovery. The historical request/permission epoch and the current owner's activation epoch are distinct facts. Do not change the request to a fresh epoch merely to satisfy a gate or create a fresh business key.

Freeze compatibility/version treatment. Explain how historical replay remains valid and which future dispatch policy applies when opening an older retained store. Use an explicit qualified compatibility rule or a meaningful version refusal; do not silently reinterpret evidence, rewrite old captures, or bump unrelated compiler/schema versions without need. Any necessary metadata/schema change must be minimal, reviewed and tested against exact prior stores.
4. Source map and implementation authority
Read the actual current definitions before naming new interfaces:
- internal/executor/runner.go: worker exposure, physical slots, cleanup, synchronous lifecycle delivery, and releaseSlot preceding accountReceived.
- external.go: prepareExternalWork, exchangeExternal, accountExternal, retry eligibility and the public synchronous DispatchExternal path.
- adapters.go: pure preparation, transportAdapter, recordAdapterResult, settlement and existing no-result withholding.
- lifecycle.go, store.go: all relevant lifecycle/Commit closure producers, owner locking, Close/Open and activation changes.
- composition_reservation.go, invocation.go, invocation_outcome.go, invocation_closure.go: root and nested eligibility, freezes, publication and charges.
- effects/effects.go, runtime/runtime.go, existing public-method consumers and examples/effectboundary.
- Phase 2A authority/process tests, internal/boundary, the persistent endpoint and its instrumentation audit.
Focused production changes are authorized in this run: the short gate, admission/completion ownership, a minimal additive public quiescence surface, necessary Close/direct-dispatch integration, corresponding tests and explicitly justified version handling. Preserve the pure reducer/compiler, established typed artifacts, dependencies and unrelated scheduling policy unless an exact necessity is demonstrated. Do not hide all changes in a fixture wrapper and credit them to the runtime.

The legacy DispatchExternal method is a real public physical route for non-composition stores. At the base it is not registered in Run's worker lifetime, while Close waits for Run. Include it in the ownership protocol or explicitly refuse an unsupported mixed mode under a reviewed compatibility rule. Do not leave an unadvertised route around the gate or Q.

The current single-host owner uses an exclusive nonblocking OS flock. Qualify that actual mechanism. Do not invent distributed leases, a multi-owner theorem, endpoint fencing, or a general takeover protocol.

The adapter contract already requires no Store access and respect for context cancellation; implementations are trusted deployment code. Conforming delayed adapters belong in the positive domain. A reentrant adapter or one that ignores cancellation is an out-of-contract boundary control, not a conforming counterexample. Do not quietly add a new adapter-entry callback/ABI to rescue a stronger theorem.
5. Work packages
Suggested new areas are docs/foundational/phase2b/, research/dispatchauthority/, examples/revocation/ and its separate public consumer, additive phase-specific tests, and focused executor files for gate/lifetime coordination. Verify naming and package conventions. Keep old models and evidence bound to the old semantics.
Task 1 — Freeze the revocation and acknowledgment contract
Outputs: charter, 01-contract.md, source map, case matrix, exact public/internal interfaces, compatibility and historical-test disposition, independent specification review.
- [ ] Verify the integration base, clean isolation, actual toolchain, dependencies and current relevant interfaces. Archive this assignment and identify the protected historical inputs.
- [ ] Trace every way the chosen authority can cease to be eligible, including ordinary terminal Commit, real lifecycle changes, root and nested scope changes, generation supersession, and Store closing. Freeze the source-terminal policy described in §3.2.
- [ ] Specify R/A/L/H/Q, target identity, ownership and lock order, all refused/error outcomes, and which transitions remain possible after each acknowledgment. State endpoint/adapter trust, local service assumptions, finite bounds and the exact stronger claim the candidate does not establish.
- [ ] Define a minimal public Q interface and consumer behavior. Decide whether the wait requires an active Run or can safely drive existing settlement; document this openly. It must not wait for a missing receipt that may never exist or initiate a new mutating recovery action.
- [ ] Freeze the path/closure/cut matrix and the version/compatibility decision. Mark source-proven unreachable combinations; do not fabricate private state to realize them. Missing feasible fixture work is not an unreachable case.
- [ ] Preserve small counterexamples to last-check physical fencing and whole-call locking that blocks lifecycle/scheduler service. These are focused controls, not three full implementations. Obtain an independent conceptual review, then freeze and proceed.
Task 2 — Independent model and conditional arguments
Outputs: independent finite transition system, complete outgoing-action checker/exports, 02-argument-and-correspondence.md, adverse controls and independent review.
- [ ] Separate durable authority/reservation/evidence, volatile admitted opportunities and completion queues, and external packets/application. Keep A and L separate transitions. A crash kills local volatile work but need not remove a request already held by the endpoint.
- [ ] Model both race orders, scope/identity distinctions, at least one independent peer, known and unknown completion handling, Q, and one crash/reopen. State each profile's bound before enumeration; cover all its outgoing alternatives before deduplicating states.
- [ ] Give separate invariants for no admission after R, no covered call after successful Q, no lost available evidence at Q, original identity/charge conservation, and truthful remaining world uncertainty. Give conditional progress statements for revocation, accounting, Q and the peer; do not hide withheld eligible work behind “pending.”
- [ ] Derive the implementation argument from the real owner/lock/transaction protocol. Include all public transport entry points and natural closure producers. State where the finite checker, source argument and concrete automatic scheduler witnesses differ.
- [ ] Reject checker mutations that merge A with L, reuse an old permit on recovery, drop a live exposure with the same operation ID, certify Q before accounting, confuse successful nested freeze with revocation, erase a packet at crash, or infer nonapplication from gate refusal/Q.
- [ ] Obtain independent model and correspondence review. The checker must be capable of rejecting an implementation trace. Reviewer agreement and replay agreement are not independent empirical corroboration.
Task 3 — Implement the short gate and complete lifetime protocol
Outputs: focused runtime changes, public Q surface, direct-dispatch/Close integration, public-import construction tests, targeted RED/GREEN evidence and implementation review.
- [ ] Build the failing tests for the frozen new contract on the pinned baseline before changing behavior. Retain a separate baseline witness of the old physical-launch claim; a compile failure is not its scientific RED.
- [ ] Implement short admission synchronized with relevant durable authority changes. Validate actual owner/generation/revision/scope eligibility without holding a Store/database lock across HTTP or Execute. Each exposure gets a distinct lifetime, including an idempotent resubmission.
- [ ] Track that lifetime through protected entry or withdrawal, transport return, and required completion handling. Release each acquired reference exactly once on every success, refusal, timeout, preparation, transport, accounting and cleanup path. Track accounting independently of the runner's physical-slot count.
- [ ] Implement the separate Q observer on exact revoked authority. Wait outside scheduler/Store locks; surface timeout and accounting errors; permit existing truthful unknown handling; never require a closed continuation to consume a result.
- [ ] Bring direct DispatchExternal into the same ownership discipline. Close must prevent new admissions and must not release the owner lock/connection while covered old dispatch/accounting can still use them. Resolve concurrent direct/automatic dispatch explicitly; track multiplicity or refuse a mode before exposure.
- [ ] Demonstrate use through a standalone consumer importing public packages. Keep ordinary lifecycle authority distinct from the observer. Do not modify the adapter ABI or introduce automatic adapter retry, result injection, compensation, a new uncertainty event, or endpoint reconciliation.
- [ ] Review lock order, cleanup, same-operation concurrency, durable compatibility and progress before the full experiment campaign. Keep the source identity of each candidate and rejected correction.
Task 4 — Adversarial live orders, composition and progress
Outputs: 03-live-experiments.md, exact histories and an independently audited case-to-source mapping. Use acknowledged causal barriers, not sleeps or timestamps as the ordering proof.

Complete these distinct families without a gratuitous Cartesian product:
Family	Required evidence
Open success	Both routes perform one real effect; the original consumer actually selects its genuine result.
R wins	Hold before the new admission gate; commit/acknowledge actual revocation; release; observe refusal and zero protected entry/ingress for this opportunity.
A wins	Hold after actual admission but before physical call; let revocation return; verify Q pending; release and observe the permitted post-R call and later Q. This surviving negative must remain explicit.
Revocation while effect is slow	A conforming delayed effect does not prevent R or an unrelated ordinary peer from making progress. Q waits separately. Distinguish ordinary progress from an external peer blocked by capacity one.
Completion accounting	Hold a genuine returned result before durable recording, then before settlement where distinct; Q must not succeed early. Release service; inspect exact record, settlement and Q order.
Source and scope	Cover actual source CancelRun, AbandonRun, retirement, selected ordinary End/private Failure policies, top-level closure with source still open, and one nested non-success/retirement closure. Compare the successful-unpublished nested ancestor positive case.
Failure publication	Test Request/private Failure on both sides of A. Preserve the fact that a failed child envelope is not a nonexecution certificate. Any changed pre-admission behavior is explicitly attributed to the new policy.
Late evidence	Authentic evidence acquired after revocation is recorded and settled under existing rules without reopening source control. Returned unknown/error and actual postcommit connection loss remain distinct.
Identity and accounting	Old-generation admission after genuine eligible Retry; original retained key/wire on recovery; exact single semantic reservation charge; later gate refusal with no refund or second debit.
Public ownership	Direct dispatch, supported or refused mixed direct/automatic mode, Close racing an admitted opportunity, and a live second Open refusal with no epoch mutation.
Wait failure	Cancel/timeout a Q wait without undoing R, losing an authentic completion or manufacturing success. Exercise actual accounting error handling.


At least one peer-progress witness must arise through the unchanged automatic Run scheduling categories, not solely a manually driven Store. The observer must not perform hidden ordinary commits to manufacture the peer result. An independent effectful peer should also complete once capacity is legitimately available.

The gate hook, protected call evidence, endpoint ingress and endpoint application have different meanings. Audit every new hook and lock held at it. Do not call back into Store while holding its mutex. If a proposed event is not separately observable, narrow the claim and supply the appropriate source argument; do not invent a fictitious atomic cut.
Task 5 — Recovery, external delay and the ordinary consumer boundary
Outputs: 04-recovery-and-world.md, 05-public-observations.md, genuine normal/race subprocess histories and independent local/world decoding.
- [ ] Requalify a bounded open-source raw-idempotent U0/U1 pair under the new gate: preserve the exact original operation/request; recover both to one application and actual consumer completion. Continue withholding unresolved non-idempotent raw and adapter attempts.
- [ ] Establish a closed-authority U0/U1 comparison: same relevant retained local evidence, different application histories, no newly admitted mutating resubmission. Both may become locally quiescent while their world histories differ. State exactly when/where local equality is checked; no row editing or erased identity fields.
- [ ] Kill at a new admitted-before-call boundary and after genuine result recording before settlement. Reopen in a fresh process on the same retained store. A lost volatile admission token is not a reusable permission; surviving durable evidence still receives ordinary handling. Tie Q to the exact old authority and report any unsupported adapter continuation honestly.
- [ ] Mandatory new delayed-application witness: let the independent endpoint acknowledge receipt of a real request and hold it before Apply; actually revoke the source/scope; SIGKILL the executor; reopen with a new owner; obtain Q for that revoked authority; then release the endpoint and observe its later application. This demonstrates the limit of local quiescence beyond Phase 2A's selected no-in-flight cuts. Use an endpoint profile that can commit the accepted request despite the client's disappearance, and state that contract explicitly.
- [ ] Check that a simultaneously live second owner is refused by the existing flock. Show fresh Open only after actual owner release or death. Direct-dispatch ownership must not be bypassed by Close followed by Open while an old goroutine can still call.
- [ ] Drive the public Q surface from the standalone consumer. Compare host observations with an ordinary parent's original selected invocation envelope. No endpoint oracle, Inspect value or Q result silently enters authored state. Record whether the ordinary parent remains waiting, receives truthful failure or completes with genuine application evidence.
- [ ] Independently audit replay against retained state and endpoint ingress bytes. No replay contact/application may occur. Qualify replay as historical consistency, not proof of new launch or recovery equivalence.
For every crash retain acknowledgment/PID/token, exact signal/exit, new PID and activation, executable/source hashes, before/after local and endpoint stores, request/receipt identity and actual selections. Keep each branch's endpoint state isolated before execution and alive through its executor crash. Do not reset the world during recovery. A timeout, graceful Close or cancelled context is not a crash witness.
Task 6 — Qualification, manuscript and reviewed delivery
Outputs: qualification, executed runbook, review/disposition record, decisions and one next task, source/evidence index, synchronized manuscript/PDF and draft PR.
- [ ] Have a separate auditor decode raw authority/attempt/result/settlement rows, physical order and endpoint journals. Check the R/A/L/H/Q relationships, multiplicity, owner lifetime, completed accounting, old-generation boundaries and uncertainty. Disclose shared formats/primitive checks and any auditor who also authored the apparatus.
- [ ] Run final focused normal/race campaigns for new model, gate/lifetime, public consumer, live/process and affected inherited core behavior. Race qualification must instrument the actual executor and endpoint subprocess binaries. Census actual leaves/skips and bind every result to its executed source.
- [ ] Because this run changes core lifetime/lifecycle paths, run the ordinary root suite once with an explicit suitable timeout, plus affected integration/ownership/authorization/replay gates. Finish interrupted or unexecuted required selections; do not claim a green full suite from a partial run. Broaden reruns only for concrete changed-code risks.
- [ ] Use the actual pinned toolchain and cached readonly modules; prior evidence uses Go 1.27.3, CGO and sqlite3 v1.14.52. Record executable, argv, cwd, environment, build inputs, outputs, exit status and hashes. Discover current selectors/build tags. No live providers. Run gofmt, focused vet, root/separate-consumer module verification, independent audits, scope checks and git diff --check.
- [ ] Keep a displaced-assertion ledger. Old Phase 2A tests intentionally describe old behavior; preserve their original sources and evidence at the pinned identity, then change only assertions legitimately displaced by the new contract. The historical model/archive must continue describing its baseline. Do not call old counterexamples “fixed” merely because a changed hook selects another order.
- [ ] Validate the Phase 2A source/evidence index in its pinned historical checkout: it binds all then-current Go/C/module inputs. Do not regenerate it against Phase 2B production changes, or describe its expected mismatch against a changed tree as invalidation of the original evidence. The new phase receives its own source/evidence index.
- [ ] Resolve each contract claim as a proved conditional argument, bounded check, tested behavior, falsification or exact open obligation. Obtain independent conceptual/model, implementation/lifetime, process/evidence and final whole-phase reviews. Agreement is review, not experimental corroboration.
- [ ] Synchronize Markdown and TeX Pillars IV/V and affected summary claims. Retain the withdrawn physical-launch clause and its historical counterexample. State the new R/A/Q contract, exact revocation domain, progress assumptions, compatibility and delayed-world counterexample. Do not promote it to general recovery equivalence or effectful-tree substitution. Change bibliography only for primary references actually used.
- [ ] After source/manuscript review, invoke tectonic alan-dissertation.tex once from paper/; retain warnings/status and visually inspect changed pages and continuations. Record any failure or necessary additional invocation honestly. Verify final source/evidence identities and open a draft PR against the verified integration base. No merge or release.
6. Guardrails and stop conditions
The selected campaign is deliberately larger than a last-minute predicate change. Complete the admission mechanism, separate acknowledgment, progress controls, all public dispatch ownership coverage, real recovery, and external-delay counterexample. A plain check added before Execute is not the complete deliverable.

Stop after the frozen obligations have supported dispositions. A substantive negative can conclude an obligation, but an invalid fixture, missing feasible integration or unrun required gate is unfinished work. If a contract premise fails, preserve the frozen version, the counterexample and the exact amendment. Do not keep changing assumptions until all checks turn green.

Keep these limits visible:
- Single-host trusted ownership and controlled endpoints; no distributed, Byzantine, or endpoint-fencing theorem.
- No universal terminal-unknown policy, automatic adapter resubmission, result injection, compensation, live provider, repository split, dependency upgrade or unrelated refactor.
- No universal no-application-after-cancellation guarantee. A remote system can complete already accepted work after local Q.
- No hidden oracle information, invented receipt, refunded historical debit or reinterpretation of a new-generation request as original-operation recovery.
- No new privileged agent class. The observer is a public host/runtime capability with its exact authority and visibility boundaries stated.
- No promise that an out-of-contract adapter terminates. Record the limit without using it to excuse deadlock or starvation for conforming adapters.
The final report must lead with what changed in the justified action contract. Give both race orders, what R and Q promise, the cost and assumptions of obtaining Q, useful-progress evidence, the exact source/scope terminal policy, preserved uncertainty, current public observations, source versions, reviews and manuscript/PDF/PR status. Distinguish implementation changes from research conclusions.

Finish by proposing one next substantial step selected by this evidence. The likely question is how a legitimately authorized actor can obtain or reconcile evidence for an original unresolved operation without silently creating a new business attempt. Do not implement that new policy in this phase. If the authority/quiescence work exposes a more fundamental obstacle, explain why it takes priority.

Phase 2 advances when a caller can tell which authority has ended, which local work has actually drained, what external facts remain unknown, and what additional observation or permission would justify the next action.