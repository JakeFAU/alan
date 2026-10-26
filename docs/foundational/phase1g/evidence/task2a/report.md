# Phase1G Task2a — gated ordered facts and immutable binding foundation

Task2a is implemented and tested against base `bd27abe05d6014694cc1102d345ca8b1762e1b13`. Public schema4 activation remains deliberately rejected. This is the agreed storage/binding foundation, not completion of Task2 or qualification of invocation outcomes. No author Invoke, admission protocol, scope outcome, effect-budget enforcement, runner change, provider calls, credentials, dependencies or migrations were added. Independent scoped review is still required.

## Internal construction and validation

`createCompositionStore(ctx,path,Provision,compositionProvision,Limits)` and `openCompositionStore(ctx,path,Limits)` call the same `createStore`/`openStore` implementations used by public Create/Open. There is one existing execution transaction and one existing lifecycle decision transaction; no copied scheduler or commit engine exists. `Store.composition` retains only canonical encoded profile bytes, so mutating caller maps/slices or decoded accessors cannot change it.

Public Create always creates the unchanged schema3 layout. Public Open/Inspect/Replay require schema3 before activation or ledger work. A real schema4 store returns ErrVersion and cannot advance epochs through public Open. Internal namespace reading recognizes schema3/4 with matching PRAGMA user_version and external key version2. The private foundation path checks the profile before advancing epochs. Unknown versions fail closed.

`validateCompositionStore(ctx,q,limits)` requires schema4, checks/retains the exact profile, runs the existing ordinary ledger validator, then reconciles all foundation tables and ordered facts. It can run on the existing read-only query/transaction interface; tests verify query_only, write refusal, unchanged epochs/projections/facts and no registry/dispatch inputs. It is not a public schema4 replay claim.

Versions are independent: `alan/composition-foundation/v1`, `alan/resource-profile/v1`, `alan/template/v1`, `alan/binding/v1`, `alan/composition-fact/v1`. Existing program semantics and artifact bytes/hashes are unchanged.

## Typed descriptors and identities

All new types are private to internal/executor. `compositionProvision` contains `Roots []compositionRoot` and `Templates []compositionTemplate`. `compositionRoot` holds immutable ID/version, generation1 root Run, explicitly assigned static Members, permitted Templates/Capabilities and `compositionLimits{MaxInvocations,MaxCreatedRuns,MaxDepth,MaxEffects,MaxInFlight int64}`. Zero logical limits are representable exhaustion; physical capacity must be positive. Each static member belongs to one root, the root belongs to its own membership list, every member has a provisioned root origin and generation1, and metadata-declared static capabilities must be permitted. These limits are retained declarations only in Task2a; charging/attempt enforcement is Task2c.

`compositionTemplate` holds immutable ID/version, map of relative member names to ProgramID/Initial Projection, EntryMember/EntryPort/CallerPeer, and declared Templates/Capabilities. Retention verifies artifact presence, typed initial projections, one designated caller boundary, closed compatible internal wiring, capability declarations, deployed template dependencies and actual message/adapter code endpoints against metadata. An undeclared dynamic endpoint cannot enter the catalogue. Mutable inputs are copied, and invalid UTF-8 is rejected before encoding/json can normalize identities.

`compositionBinding` retains ID/version/namespace, InvocationID, OperationID, TemplateID, RootID, Caller, Depth, relative Members and original Programs. `retainCompositionBinding(namespace,rootID,template,binding)` checks complete exact member/program maps, version, original caller, depth, canonical IDs and cross-scope consistency, then returns an immutable `compositionEnvironment`. It accepts no grants or replacement code from payloads. It does not itself prove that an Invoke has committed or been admitted; Task2b must obtain the template/root/operation from authoritative retained records before using it.

IDs are domain-separated SHA-256 of canonical JSON. Root ID includes resource-profile version, namespace and root Run. Invocation ID includes namespace and original operation ID. Member RunID includes invocation ID and relative member name, generation1. Template/binding IDs include the complete descriptor with the ID field empty during hashing.

`resolveCompositionProgram(program,environment,member)` creates a fresh ephemeral interpretation. It resolves only exact ordinary message source/target/generation operands justified by used port metadata, and matching metadata peers. Original Program bytes/ID stay untouched; payload strings, local values and unrelated operands remain literal. The returned interpretation must never be stored under the original artifact ID.2b must use this same pure path for live reduction and replay. Authenticated invocation correlation, caller reply admission and Operation matching are still Task2b work; this helper does not confer them.

Typed future envelopes `compositionInvocation`, `compositionAdmission`, `compositionOutcome`, `compositionCharge`, and `compositionFreeze` reserve identity/result fields, including Status/Reason/ResponsePresent and exact frozen reply/Invoke/external operation lists. No success is inferred or published by Task2a.

## Tables and ordered transaction facts

Schema3's DDL is unchanged. Schema4 additionally creates:

- `composition_profile`, `composition_roots`, `composition_static_members`, `composition_templates`: immutable deployment descriptors plus relational identity columns.
- `composition_facts`: gap-free monotonically increasing decision_order, unique deterministic fact ID, finite kind and canonical descriptor.
- Reserved `composition_invocations`, `composition_bindings`, `composition_members`, `composition_results`, `composition_charges`: foreign keys and unique operation/member/result identities for later protocol work. The foundation validator requires these tables to be empty. `composition_members` is the intended new invocation-origin domain; Task2b must reconcile it explicitly, not forge static `run_origins` entries to satisfy legacy validation.

`compositionExecutionEvidence` derives a typed fact from the exact history row, covering Run/Sequence/Epoch/Revision/selected observation/successor/intents. `recordCompositionExecution` appends it inside Store.Commit, after staging ordinary rows and before the existing execution.staged hook and transaction commit. Unscoped execution retains the legacy path without a fact.

`compositionLifecycleEvidence` covers the exact retained lifecycle descriptor/result and every involved scoped root. It binds to the actual Before.Run, including a stale-generation request inspecting a scoped generation1 target. `recordCompositionLifecycle` runs after inserting the first lifecycle result but before the existing lifecycle commit. Source authorization remains separate. Retransmitted original decisions do not enter the new hook. A new operation whose original result is already_terminal gets one decision fact; retransmitting that no-op adds none.

`appendCompositionFact(ctx,*sql.Tx,fact)` neither starts nor commits a transaction. It validates exact source-row linkage, version, deterministic ID, roots and digest, returns an existing order for an identical duplicate, rejects conflicting/unlinked/empty descriptors, and allocates the next order with overflow protection. Supported kinds are execution/lifecycle. Admission/reservation/reply/freeze/outcome kinds are explicitly reserved but return ErrVersion until Task2b/Task2c add their source-row validators. This is the narrow extension point for later actual transaction facts; it cannot currently fabricate unfinished outcomes.

The validator checks both directions: every scoped history/lifecycle row needs exactly its linked fact, every fact needs its original row, no missing/invented facts, exact canonical descriptors/digests/root membership, per-run sequence prefixes, lifecycle decision order, source intent precedes target decision, and target Before.Sequence matches the ordered prefix. A lifecycle closure cannot be followed by another execution fact for that run. Profile/root/member/template rows must match in both directions, and legacy origin validation still applies.

Scoped Retry was found to escape fixed membership during self-review. The incomplete private profile now rolls back a would-be scoped successor with ErrVersion and rejects a scoped subject that has changed generation on validation. This is deliberately not the final durable `invocation_retry_unsupported` result policy; Task2b must implement that correlated policy before removing the public gate. Other existing lifecycle decisions use the normal production path.

## Executed ordering witness and regressions

`TestCompositionOrderedFailurePrefixes` executes the real Store.Commit sequences A.Send1/B.Failure/A.Send2 and A.Send1/A.Send2/B.Failure. It compares exact run rows, terminal rows, per-run history rows and immutable intents; they are identical. It independently checks logical fact subject order and computes the IDs committed before B's failure: one reply versus two. Namespace-derived root IDs are irrelevant to that assertion. `Frozen` in the retained witness is the test monitor's expected prefix, not a runtime freeze/result implementation. Inputs and actual rows are retained in `counterexample-input.json`, `executed-counterexample.json` and original log 17.

Transaction tests inject failure at execution/lifecycle fact insertion. No ordinary projection/history/intent/terminal/lifecycle-result transition survives without its fact. Stale Commit and identical fact/lifecycle retransmission do not add facts. Other tests cover a valid two-member template, payload literal preservation, original artifact hash/byte identity, cross-invocation/member/program/root/invalid-encoding binding refusal, immutable profile copies, schema3 table isolation, private reopen and public gates, read-only validation, and forward/reverse corruption.

Original failures are preserved, not relabeled as success:

- 01 is missing-API compilation RED.02 is a schema import-name build failure.
- 03 failed because the missing-root corruption injection was prevented by SQLite's foreign key constraint before reaching validation. This is setup failure, not evidence that the intended monitor fired. The test now disables FK only while injecting corruption and restores it before validation.11 retains exact failures: missing/invented facts, static membership mismatch/missing member, row mismatch, prefix-order violation, and separate FK-origin failures.
- 04/05: arbitrary empty fact accepted, then rejected.
- 06/07: metadata/code endpoint mismatch entered the catalogue, then rejected.
- 09/10: private scoped Retry escaped generation1, then transactionally gated.
- 12/13: missing scoped stale-lifecycle fact and invalid UTF-8 caller binding, then fixed.
- 14/15: invalid root spelling normalized into another valid root identity during JSON copying, then rejected before copying.

`commands.json` records all exact Go argv/environment/output files. `08` and final `16` passed `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build GOTOOLCHAIN=local go test ./... -count=1` across all 15 tested packages (runtime and alan-resource have no tests). The final full run includes all production fixes. Only stronger test-only X/Y row-equality assertions followed; targeted 17 passed them. No production changes followed final 16. Changed Go files are gofmt'd; whitespace checks are clean. No race or integration-tagged suite is claimed.

## Task2b/2c handoff

Keep the public gate until admission/outcome/closure/budget/replay integration is complete. Add actual protocol source-row validation to the reserved fact branches, in each operation's existing owner transaction. Dynamic origin validation must be explicit in both directions. R2's failure freeze must attach to the failing execution/lifecycle transaction; callbacks cannot choose its candidate set. Successful publication must wait for committed-but-unadmitted Invoke obligations and unknown descendants/external obligations. Extend the pure binding environment without mutating retained artifacts or user strings. Persist reason/status/response-presence for ordinary AwaitInvocation through an explicit outcome-bindings struct as directed by the controller.

No whole-protocol success, external conservation, per-root physical capacity, live recovery dispatch, or public compatibility with schema4 is established by this foundation. Those remain the next agreed slices; the gate makes their absence explicit.


## Independent review fix round 1 — I1 causal observation predecessors

The original report above records the baseline implementation and its verification. This subsequent patch addresses Important I1 from the retained, unchanged `review.md`; Minor M1 remains deferred as directed. Review base is `23d6ee4fa1fa03fa05bcdae16f998ea73788e91d`.

> Execution-prefix validation permits consuming a future scoped message.

The unchanged independent overlay probe reproduced the defect before any production fix (`review-fix1/01-original-probe-red.log`, exit1). The original source and overlay are retained byte-for-byte as `original-probe.go.txt` and `original-overlay.json`. The reviewer report remains unchanged (SHA-256 `26d2d4aa31b317401e4ab4225111fa573e29ba22fc408dd2fa450b48d642b422`).

`compositionObservationPredecessor` now resolves each selected ordinary-message/runtime-terminal/runtime-lifecycle observation through its immutable admission binding and original source ledger. It returns the exact represented execution/lifecycle fact. The global fact validator requires that identity in the already-validated prefix before accepting the consuming execution:

- Ordinary messages require their original scoped source intent's execution commit.
- Ordinary terminal evidence requires the source terminal execution. If `terminal_obligations.operation_id` identifies a lifecycle closure, it instead requires that lifecycle decision; the same terminal sequence's earlier ordinary execution is insufficient, including closure at sequence0.
- Runtime lifecycle results require the retained target-decision fact when represented. Merely seeing the source request execution is insufficient. Source-denied settlements have no target decision to invent.

The resolver verifies the selected admission's original target/run, canonical observation, and source descriptor. Local inputs and genuinely unrepresented source transactions have no invented predecessor. Adapter/receipt and future invocation facts remain deferred; public schema4 gates and all protocol scope limits remain unchanged.

New tests use actual Commit/DeliverMessage/DeliverTerminal/DecideLifecycle/SettleLifecycle paths and first pass full foundation validation. They preserve every ordinary inspection table and every fact identity/source digest while changing only decision_order and the descriptor Order. Five cases failed before the fix (`02-causal-variants-red.log`) and now reach the specific selected-observation prefix rejection (`03-causal-variants-green.log`): ordinary message, ordinary terminal, lifecycle terminal at sequence0, lifecycle terminal after an ordinary source step, and lifecycle-result consumption after its source request but before its target decision. Four positive controls cover unscoped message/terminal/closure sources and local input.

Verification: unchanged independent probe GREEN in `04-original-probe-green.log`; the complete executor default package passed `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build GOTOOLCHAIN=local go test ./internal/executor -count=1` in `05-executor-suite.log`. All exact argv/environment/output/exit statuses are appended to `commands.json`. Only composition_facts.go and its focused test file changed in production/test source. They are gofmt'd and whitespace-clean. No new full-repository, race or integration-tagged run is claimed for this scoped fix. No providers, credentials, dependencies or subagents were used. Ready for independent scoped rereview.
