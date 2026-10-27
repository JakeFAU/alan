# Phase 1H closure companion fixture design review

**Disposition: APPROVE DESIGN WITH REQUIRED CONDITIONS below.** No new runtime API or application orchestration is needed. This is a source-based design review only: no test, build, implementation, or repository edit was performed. It does not release Task 3 or close the running Task 2 evidence gate.

## What this fixture can establish

The preparation correctly separates the schedules. A committed complete round decision cannot simultaneously have an unknown required outcome in that round. Keep the primary `decision ACK -> SIGKILL -> reopen -> operator cancel -> close -> fresh Run` schedule unchanged. The late-transport companion is a separate, explicitly named schedule using a live owner. It must not be described as a late required adaptive leaf at the primary decision cut or as fresh-process recovery of an interrupted transport.

A statically provisioned ordinary peer under the same ResourceRoot is sound. The runtime looks up root membership by the effect's source run, including static members (`internal/executor/composition_facts.go:48`), then checks root closure at the reservation boundary (`composition_reservation.go:28-58`, `invocation.go:165`). Cancelling root changes root's local status, not the peer's local projection (`lifecycle.go:205-220`). The peer can therefore retain/consume a preauthorized late result and continue ordinary local instructions while new scoped work is denied.

## Precise public ordinary peer construction

Author one test-owned peer with `author.Compile` and provision it through `author.Application{peer: {Artifact: artifact, Inputs: emptyMap}}.Provision()`. Use a local start port with an explicit bounded request schema and one declared controlled capability/effect. Suggested authored steps:

1. Receive an explicit fixture start input.
2. Request the held effect, retaining its ordinary request ID.
3. Await that exact effect ID, binding the typed response and validity.
4. Request a second effect with a distinct literal fixture payload, retaining its own ID.
5. Await the second ID; retain its actual denial/validity evidence, then terminate locally.

Do not insert Success before the second reservation attempt, because that would cause source closure instead of the intended root-scope reservation denial. Do not derive application work IDs/frontiers in this peer. The two payload labels identify transport fixtures only; map them to actual Request.ID/Key and original committed intents.

Merge the peer's public Provision Programs/Subjects/Runs/Permissions into the adaptive deployment before Create, with unique names and program deduplication where necessary. Add `RunID{Subject: peer, Generation:1}` to the existing root's Members and its capability to that root's fixed allowlist. Preserve root32/64/depth3/effects96 and capacity1 or2. The capability grant alone is insufficient without membership: an unscoped peer would bypass the reservation fence being tested. Existing provisioning validates static origins and permitted effect capabilities (`composition_store.go:132-166`). No SQL writes or private adapter preparation/settlement calls are appropriate.

Use the existing deployment-granted operator as the expressly labeled test-environment lifecycle policy. Its low-level lifecycle instruction fixture is not a newly invented public author lifecycle API. Keep that operator outside the tested root's Members, with the exact permission to cancel root generation1/revision1. The ordinary peer itself uses only the public author/effects/runtime surface.

## Required ordering and root-open condition

A deterministic minimal companion leaves the adaptive root intake awaiting input (no adaptive work request submitted in this companion). The primary schedule separately proves actual adaptive decision closure. This avoids the adaptive root succeeding locally before operator cancellation. If the implementer instead submits adaptive work in the companion, it must provide an acknowledged, source-backed way to keep root open until cancellation; racing a fast application against a held peer is invalid. A root's local completion can close reservation scope even while a separate peer remains open. Require actual lifecycle `Before.Status=open`, `After.Status=cancelled`, accepted reason `closed`, not `already_terminal`.

Use one Store owner and one automatic Run for the companion:

1. Admit the peer start input; start Run with the controlled loopback adapter.
2. Wait for the held provider contact and verify its committed source intent, allowed authorization/adapter attempt, allowed reservation, and exactly one corresponding effect charge. The HTTP fixture blocks before returning a response. Verify no result for that operation yet and peer remains open at Await.
3. Admit the operator's cancel command through the same Store; let the existing Run execute the operator and lifecycle delivery. `runner.go:273-285` already loads, authorizes, decides and settles lifecycle work. Do not race the existing `cancelAsOperator` helper's manual Commit sequence against Run's automatic execution. A separate stopped-owner/manual operator sequence remains suitable for the primary test, not this live transport schedule.
4. Establish the actual committed cancellation/freeze and preferably the published cancelled root outcome before releasing the held response. Use retained history/facts, not sleep or mere command admission. Keep the Store and Run context alive and the adapter timeout long enough for this bounded handshake.
5. Release the HTTP response. Automatic Run must record its exact original candidate/result, settle it to the original peer generation, and consume it through the peer's Await. The second Request then follows by ordinary durable program execution, so its emission and reservation attempt are necessarily after closure.
6. Require the second source authorization to remain allowed, the exact grant to exist, and the adapter attempt/reservation decision to be denied with `scope_closed`. Require no second Execute/provider contact and no second effect charge. Allow its ordinary retained denial result/settlement and local peer termination. Finish Run and inspect/replay; the original root cancelled outcome/freeze remains unchanged.

No SIGKILL/Store.Close/context cancellation belongs between held contact and late settlement in this companion. Those actions can turn an authorized request into unknown work or cancel its transport; they do not prove actual late response settlement. Avoid Snapshot calls inside execution-commit observers because of Store.mu. The approved read-only cut probe may select a boundary, but full independent Inspect/history correlation remains required. Environmental input admission can wake the live Run; no second scheduler is necessary.

## Source, target and evidence distinctions

For both Requests the source is the peer and the target is the capability. For adapter settlement the retained result is bound back to the original peer run, not to root (`adapters.go:345-347,385-397`). Root is the enclosing scope and lifecycle target. Cancelling root must not be reported as directly cancelling the peer source. Keep the peer's generation/revision and local open status unchanged until its second attempt, so `scope_closed` exercises the intended reservation gate rather than `source_closed`/`source_revision_changed` or missing permission.

Late authenticated evidence is retained from the original attempt rather than being reauthorized as a new effect. `recordAdapterResult` validates the original attempt/result identity and records the result (`adapters.go:291-343`); `settleAdapter` uses that retained original operation. Source authorization and reservation are separate facts. The first allowed reservation can remain valid after closure; a provider's later response does not grant a second reservation or reopen root.

Correct the phrase “no new reservation” to **no new allowed reservation or charge**. The denied second operation should create a `composition_reservations` decision row with Allowed=false; `composition_reservation.go:60-93` intentionally records denied decisions without charges. The resulting adapter candidate may use disposition `source_denied`; the more precise root-scope reason is on the attempt/reservation record. Do not conflate those fields.

Capture actual evidence of ordering: allowed reservation fact before lifecycle closure; provider entry before release; root cancellation before release/result recording; late external-result fact plus settlement/consumed observation; second intent then allowed source authorization plus denied reservation. Compare the immutable root outcome/freeze before and after late evidence, not the entire inspection: legitimate result, observation, settlement and local drain rows will be added. UnknownExternal may cease to list the first operation once its result becomes known; this is resolution of evidence, not reopening.

## Charges and scope of claim

For the minimal idle-adaptive companion, expected charged counts are invocation0, created-run0, effect1. The second denied operation adds no charge. Fixed provisioned runs are root1 + peer1 + operator1 (operator outside the scoped root), reported separately; the static peer adds no invocation depth. If the companion includes adaptive work, report its actual charges separately plus the peer's one effect, never reuse the primary seven-leaf charge oracle as an unchanged total. Verify capacity1/2 against the physical transport ledger; sequential Await makes peak1 sufficient here.

No impossibility remains with this construction. The conditions above are essential to avoid three misleading outcomes: cancellation of an already-terminal root, a source-closed denial mistaken for root reservation fencing, or an unknown/replayed attempt mistaken for actual late HTTP settlement. Exact implementation and resulting evidence still need review after Task 3 is authorized.

## Assignment scope: combining the two schedules

I also checked `docs/foundational/phase1h/assignment.md:99` (“Closure between decision and admission, with late evidence unable to reopen the closed computation”), Task 3's named closure item, and the reviewed contract's closure rules. **The two explicitly labeled schedules support the narrow combined claim; real late transport in the exact primary killed-owner schedule is not additionally required.** This is a transparent decomposition of application-boundary preservation and existing scope-level late-evidence/reservation behavior, not a claim that both were witnessed in one execution.

The primary must retain the acknowledged adaptive decision, actually cancel before successor admission, exercise actual attempted successor admission and prove no new descendant or charge. It may then re-deliver/re-settle the original known invocation outcomes and verify that the same closed computation and decision remain unchanged. Describe these operations precisely as duplicate known evidence/idempotent settlement. If the API returns an already-retained settlement without creating an observation, do not describe that as a new late result arrival.

The companion provides the stronger, non-vacuous post-closure transport/result recording and scope_closed reservation check through a locally open static peer. It uses the same unchanged root membership/closure/reservation machinery but adds a fixture member and, in the minimal form, no adaptive work request. State those differences, separate the evidence IDs/charges/claims, and retain this decomposition as an explicit plan clarification. Neither fixture alone should be advertised as proving the other's property.

Requiring a new authenticated late *required round result* at the primary complete-barrier decision would contradict the barrier. Requiring recovery of that same killed owner's in-flight HTTP response would instead require unsupported result injection/retry or a different transport model. Do not add either to manufacture a unified narrative. A future stronger same-execution companion-peer schedule is optional and would need its own ordering review; it is not a prerequisite for this scoped coverage conclusion.
