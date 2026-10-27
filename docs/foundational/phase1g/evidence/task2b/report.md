# Phase1G Task2b — templates and atomic invocation admission

Implemented against `56d25e0ab4b614afdec791f8ff090852d65627bc`, using the Task2b brief and normative contract. The final default suite passed. This is the gated Task2b slice, not full Phase1G qualification. Independent controller review is pending. No Task3 builder, second scheduler, provider calls, credentials, dependencies, migrations or historical-evidence edits were added.

## Frozen public author API

- `author.NewTemplate(application author.Application, entryMember, entryPort string) (author.Template, error)` retains ordinary member artifacts, exact initial input defaults, their declared capabilities and child templates, and one designated public Port from the entry artifact. It validates internal closure through the same pure `executor.RetainTemplate` used by provisioning. `Template.ID()` and `Template.Bytes()` expose identity and independent encoded bytes. Template/artifact/catalogue state is privately encoded and cannot be mutated by changing caller maps after construction.
- `author.Definition.Invocations map[string]author.Template` declares named permitted targets. The existing `Definition.Templates map[string]string` keeps its older meaning and encoding. Invocation metadata retains target template ID plus public input/output schema IDs.
- `author.Invoke(pending, template string, values, updates map[string]author.Expr) author.Step` lowers to ordinary `emit` with effect kind `invocation`, immutable target template ID, generation1, named declaration obligation, and typed request values. It commits the operation ID into `pending` and updates local state in the same existing reduction/commit.
- `author.AwaitInvocation(template, pending string, bindings author.InvocationOutcomeBindings) author.Step` lowers to ordinary correlated `receive`. The bindings struct is `{Status, Reason, ResponsePresent string; Response map[string]string}`. All three envelope destinations must be explicit valid variable names. Payload mapping keys are public response field names; generated observation keys are `response.<field>`, avoiding envelope collisions. No collection/scatter/aggregation/retry policy is in an opcode.
- Await's authenticated boundary is kind `invocation`, provenance `runtime-invocation`, source template ID, generation1, original declaration name, original Invoke operation ID. `status`, `reason`, `response_present` are separate from optional response values. Absent payload bindings are removed from the projection, including stale previous values; no enum-invalid defaults are fabricated. Programs must branch on presence before using payload variables. This slice does not add static optional-type flow refinement.
- Invocation compilation selects `alan/1g/v1` / `alan/author/v2`. Absent metadata remains omitted. Legacy scalar and collection-only artifact meaning is unchanged.

`Application.Provision()` now includes the complete immutable template/artifact dependency catalogue in `runtime.Provision.Composition` when invocations are declared. The deployer must explicitly assign `p.Composition.Roots`. Runtime aliases are `CompositionProvision`, `ResourceRoot`, `ResourceLimits`; a root has generation1 `Run`, explicit static `Members`, `Templates` and `Capabilities` allowlists and `Limits{MaxInvocations, MaxCreatedRuns, MaxDepth, MaxEffects, MaxInFlight int64}`. IDs/version are canonicalized at retention. Merely appearing in the catalogue does not grant root admission authority.

Example of the Task3 boundary, once Task2c activates it:

```go
leaf, err := author.NewTemplate(leafApplication, "entry", "public")
// Check err; compile the caller with Invocations["leaf"] = leaf.
// Its steps use Invoke("pending", "leaf", values, updates), then
// AwaitInvocation("leaf", "pending", author.InvocationOutcomeBindings{
//     Status: "status", Reason: "reason", ResponsePresent: "present",
//     Response: map[string]string{"result": "result"},
// }).
provision, err := callerApplication.Provision()
// Check err; explicitly supply the approved catalogue IDs and static members.
provision.Composition.Roots = []runtime.ResourceRoot{{
    Run: runtime.RunID{Subject: "root", Generation: 1},
    Members: []runtime.RunID{{Subject: "root", Generation: 1}},
    Templates: approvedTemplateIDs, Capabilities: approvedCapabilities,
    Limits: runtime.ResourceLimits{MaxInvocations: 32, MaxCreatedRuns: 64,
        MaxDepth: 3, MaxEffects: 96, MaxInFlight: 1},
}}
// Public runtime.Create still returns ErrVersion for this provision in Task2b.
```

## Version/create/open matrix

| Input/profile | Creation and reopening in this slice |
| --- | --- |
| Legacy scalar or collection-only artifact, no Composition | Public Create uses unchanged schema3; existing Open/Inspect/Replay remain supported. Default suite preserves prior artifact identity tests. |
| Actual invocation declaration/effect/outcome receive with no Composition | Public Create rejects ErrVersion; schema3 cannot silently acquire invocation work. Raw legacy semantics cannot encode invocation emit/receive instructions. |
| Any nonnil public Composition | Public Create rejects ErrVersion before creating a store. |
| `alan/composition-foundation/v1` | Explicit private foundation path retains original foundation descriptors/layout and validation. Reserved protocol facts return ErrVersion. No promotion of retained foundation stores occurs. |
| `alan/composition/v1` | Constructed only by private `createInvocationStore`, using the existing createStore and transaction engine. Schema4 plus invocation tables. Initial empty protocol may validate internally; once an Invoke intent, admission or new protocol fact exists, private validation/open explicitly returns ErrVersion pending Task2c. |
| Any schema4 public Open/Inspect/Replay | Still rejected before activation. No schema4 replay/recovery claim. |
| Unknown persisted profile, binding or program version | Fail closed through explicit version checks. |

The new profile is separate from the foundation. `compositionProvision.Version` is omitted for foundation records and explicitly retained for protocol records. Invocation bindings retain optional `Protocol: alan/composition/v1` under the separate binding descriptor/version, so legacy scalar member bytes are never patched under an old artifact hash. No additional intermediate version or migration was introduced.

## Admission and dynamic-origin implementation

`Store.AdmitInvocation(ctx, operationID)` first uses the existing, separately committed `AuthorizeIntent`. Under the existing owner mutex/SQLite transaction it loads the committed original operation, returns an existing permanent accepted/rejected decision, and checks source run/revision/generation, declaration/schema, deployed template, root/template allowlists, capabilities, source/root/ancestor closure, depth and total root admissions/created runs. It accepts no caller-provided code, binding, root or grants.

One accepted transaction retains:

1. `composition_admission_decisions`: every original accepted or rejected operation result, including decisions for missing/disallowed targets without pretending they are admitted invocations.
2. `composition_invocations` and `composition_bindings`: accepted invocation descriptor, original caller, template, source sequence/revision, root/depth, exact member/program maps and binding ID.
3. All generation1 subjects/runs and immutable `composition_members` origins. Each origin retains original program ID, exact initial projection, relative member, invocation/root and owner. No forged static `run_origins` rows are used.
4. Internal ordinary message and declared adapter permissions derived from retained artifacts. The public reply route is authenticated by the binding, not an arbitrary message grant.
5. `composition_requests`: an initial authenticated request observation for the designated entry, bound to the original caller and Invoke operation. Request provenance is `runtime-invocation-request`. The shared pure `resolveCompositionProgram` adds only this immutable correlation and exact metadata endpoint substitutions; payload strings and retained artifact bytes remain literal.
6. An ordered admission fact and one root charge (`Invocations:1`, `CreatedRuns:len(all members)`, `Effects:0`). Initial static members remain fixed provisioning cost. Denial spends neither count; retransmission never creates or debits again.

`boundCompositionProgram` is the shared typed resolution entry point consumed by live Commit and ordinary message admission. Task2c must use it for runner selection and replay too. `compositionScopeClosed(ctx,q,run)` is the transaction-local last-boundary helper for inherited root/ancestor closure; Task2c must use it at external attempt reservation. Existing allowed source authorization is retained historically even when later admission rejects. Scoped Retry is a durable `invocation_retry_unsupported` result before any successor run is staged; foundation's older transactional safety gate remains distinct.

Admission reasons implemented here: `accepted`, `source_denied:<original reason>`, `source_closed`, `scope_closed`, `declaration_denied`, `template_missing`, `template_denied`, `capability_denied`, `depth_limit`, `schema_invalid`, `invocation_limit`, `created_run_limit`. Unknown/uncommitted operation IDs are errors, not invented rejection records. Unscoped sources have no resource root and are errors. Malformed/corrupt rows remain errors.

Runtime aliases `InvocationAdmission`, `InvocationDescriptor`, `InvocationOutcome`, `InvocationFreeze`, `InvocationPreparation` provide named typed handoff views. These are provisional, not an SDK stability claim.

## Reply lifetime, freezes and known outcomes

For the invocation profile only, `AuthorizeIntent` recognizes the designated entry member's already committed, immutable caller-directed public Send as a retained reply notification. Its source generation/revision still must match, but terminal/retired source status does not suppress this notification. Adapter effects and arbitrary/internal messages do not inherit this exception. Source authorization remains a separate record/transaction.

`DeliverMessage` routes that exact bound public reply into `composition_replies`. It validates/authenticates the original source, route and public response schema, records one candidate per original Send operation, and retains an ordered reply fact. Ordinary transport admission/acknowledgment records still account for delivery and lost acknowledgments; public candidates do not become raw ordinary-message observations in the caller. Different transports retry the same candidate; a second distinct Send is a second protocol obligation even with equal payload. Candidate admission retains the ordinary descriptor byte bound. `AcknowledgeMessage` works on its original admission.

Inside the existing execution/lifecycle owner transaction, `freezeCompositionScope` runs after its ordered source fact and before commit. First non-success freezes status/trigger/member sequence and exact committed reply IDs. All-member success freezes reply, Invoke and adapter/legacy-external IDs. The trigger is an ordered source fact ID, and the freeze has its own ordered decision identity. Failure and freeze roll back together on an injected failure. Later member activity/delivery cannot alter the frozen sets.

Scopes are keyed by invocation ID. Static resource-root scopes use `root:<root.ID>` and include all explicitly provisioned static members. Root local terminal closes future descendant admission. **Root public completion/publication is not implemented:** Task2c must consume these root freeze sets, account for pending Invoke/external obligations, and expose root join/completion separately from local terminal. It must never infer root public success from the local root projection alone.

`Store.PrepareInvocation(ctx, operationID)` is read-only preparation, not publication. It exposes `Ready`, `Pending`, `Freeze`, `Outcome`. Pending reasons are `members`, `reply_admission`, `invocation_admission`, `descendant_join_requires_task2c`, `external_join_requires_task2c`. A committed-but-unadmitted Invoke is required work, not an absent child. Accepted descendant joins and committed external obligations remain explicitly pending in this intermediate slice even if a later reader could infer more. Denied child admissions are known evidence; ordinary program policy may handle them.

`Store.SettleInvocation(ctx, operationID)` publishes only when preparation is ready: immutable rejection, frozen non-success after all frozen reply admissions, or simple success with known admission decisions and no unfinished descendant/external joins. It retains `composition_publications` for all outcomes, `composition_results` for accepted invocations, an ordered outcome fact, and one authenticated correlated result observation plus `composition_outcome_deliveries`. Repeated settlement returns the original result/observation identity. Parent consumption remains a later ordinary Commit. A closed original caller can retain late evidence without reopening or satisfying any other run/operation.

Outcome statuses are `succeeded`, `failed`, `cancelled`, `abandoned`, `rejected`. Non-success defaults to `member_<status>`; rejection uses its original admission reason. A completed successful scope without one valid response becomes `failed/missing_reply`; distinct frozen reply intents become `failed/conflicting_replies`. A sole valid response remains present alongside the failure status, including evidence-style Send(`invalid_result`) then Failure. Later replies are retained late evidence outside the immutable failure outcome. No absent or unknown response is fabricated success.

## Verification and original observed failures

All Go invocations used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`. `commands.json` records exact argv/environment/output files and status evidence; original logs are preserved without overwriting. Logs01–18 were redirected by shells ending with `cat`, so their Go status is evidenced by original PASS/FAIL output, not a separately captured numeric shell status. Log19 separately captures the actual subprocess exit code0 and final source identity.

- 01/02/03/04 retain author missing-API/compiler scaffolding failures, an import collision, and the omitted status MaxBytes error, then GREEN. Log03's original filename says green but its content is explicitly FAILED; it is not relabeled.
- 05/06 retain missing protocol API scaffolding (including a corrected test-only nonexistent store-path reference), then atomic/fresh/permanent admission, bounds, second-member rollback, public request correlation, diagnostic reply after failure, ack-loss replay and conflicting reply GREEN.
- 07 retains a missing settlement API and invalid test field-assignment syntax. 08 is the behavioral RED: scoped Retry produced an old reason, the settlement stub returned ErrVersion, schema3 accepted invocation artifacts, and a caller's mismatched public schema entered the catalogue. 09 passes the fixes; root closure ordering was already green in08.
- 10 exercises both nested admission/ancestor-failure orderings, immutable earlier authorization, failure with blocked entry/no committed reply, and success waiting for committed-but-unadmitted Invoke. A known rejected child subsequently allows otherwise valid ordinary policy success. Valid dynamic rows reach the invocation profile's private reopen gate.
- 11 is a real regression: `TestCompositionFactAppendRejectsUnlinkedAndConflictingFacts` reached a missing protocol table when foundation required ErrVersion. 13 fixes it with explicit profile dispatch before protocol-row reads. **This is not Task2a M1.** M1's specific FK-valid foundation reserved-table guard coverage remains deferred. The valid dynamic reopen fixture separately tests the invocation gate, not M1.
- 12 verifies the reply lifetime exception does not grant adapter/internal-message authority, freezes exact external intent IDs, injects failure-freeze rollback, and delivers a later reply first without changing the frozen response set.
- 14/15 retain and fix self-review REDs: raw legacy invocation instructions encoded successfully, and a private store with a committed unadmitted Invoke reopened. Both now fail closed at the intended version boundaries.
- 16 verifies capability denial and nested shared conservation: outer2 runs + inner2 runs spend2 admissions/4 created runs; a third operation rejects; repeat admission does not charge; the one fixed root is reported as a fifth run, not charged.
- 17/18 retain and fix oversized reply transport admission. The final focused tests passed after that guard, encoding-error propagation and formatting.
- **19:** `go test ./... -count=1` passed all15 tested packages; runtime and alan-resource have no tests. Exit0. No production/test source changes followed this run. No race or integration-tagged run is claimed. No subprocess SIGKILL, automatic scheduling, physical saturation, provider behavior, full recovery or schema4 replay qualification is claimed by these manual protocol transition tests.

Final test fixture sources are retained as `protocol-tests.go.txt` and `author-tests.go.txt`; `input-fixtures.json` records concrete default and exhausted limits, member/schema/input values and the byte-bound probe. Earlier logs preserve stage-specific line numbers and errors; per-attempt source hashes were not captured for those iterative stages. The base, final exact source manifest, final test fixtures and retained observed changes are distinguished from an unsupported claim that final sources reproduce an earlier RED without reverting the fix.

Final source identity (all tracked/new nonignored Go files plus go.mod/go.sum, sorted path-NUL-SHA256-newline manifest) is `8aa1265af03319dd40064f405d600017cc88a031bd4b0435d6950895d383fa07`; individual hashes are in `source-manifest.json`. Changed source is gofmt'd; `git diff --check` is clean. Original failures and all earlier phase evidence remain intact.

## Remaining Task2c work and self-review limits

Task2c owns complete forward/reverse cross-ledger validation, dynamic origin reconciliation against all rows, prefix replay of reply/admission/freeze/publication, immutable permission/charge conservation, and both directions of observation provenance. Existing ordinary message replay cannot accept candidate-only public reply admissions or terminal-source reply authorizations without the scoped version branch. Private reopen deliberately refuses these unfinished protocols before attempting to claim validation or advance activation epochs.

Task2c also owns root external authorization reservation/debits, persisted root/global physical slots, runner work selection using the same bindings, full accepted-descendant and external-obligation joins, root publication/inspection/CLI, and final public schema4 activation/recovery tests. The existing Store.Run has not been made invocation-aware in this slice. No logical parent gets a physical slot here. The legacy foundation and scalar/collection-only matrix must remain separate when2c enables the complete protocol.

Self-review checked closure/admission ordering, all-member transaction rollback, alias/copy boundaries, source/reply authorization separation, original result deduplication, exact freezes versus delivery order, explicit absence versus enum values, and version/profile isolation. Missing full replay/resource/runner enforcement is an explicit gate, not a proof of those properties. No controller-owned old plan or historical evidence was edited.

## Independent review fix round1 — I1 retirement after local success

Fix base: `a648ea739d21f04e011be3214f7819130a5f3f2a`. The original Important finding was:

> Retiring an already-succeeded member fails to close its still-active scope.

The unchanged independent overlay reproduces the actual ordering: a required static helper remains locally `succeeded`, a root peer has committed/authorized Invoke, and accepted RetireSubject newly retires the helper. Before this fix the scope stayed open and admitted two children. Original reviewer source, overlay, full overlaid test bytes, input hashes, output and numeric command result are copied without alteration under `review-fix1/original-*`. Original review SHA256 remains `9eae098875dfe542c251fded40eddf5ce648f5410d423bb6259ac5202395b677`; `original-review.md` preserves its exact bytes. The review itself is not rewritten.

### Policy and implementation

This fix implements controller-authored normative R7. Newly setting a subject's retired flag closes its still-unpublished scope even if its local projection already succeeded. The local projection and previous source authorization remain untouched. Accepted `already_terminal` CancelRun/AbandonRun has no new target transition and creates no closure. The first non-success scope cause wins; published outcomes stay immutable.

`recordCompositionLifecycle` now passes the actual accepted lifecycle result and the existing lifecycle fact's order into `recordCompositionLifecycleClosure`. Only an actual retirement transition or open-to-terminal transition can add scope evidence. The ordinary schema3 lifecycle decision is unchanged. There is no scheduler, delegation capability, callback policy or rewritten local terminal.

A separate private-protocol `composition_closures` table retains the first applicable retirement closure: `scope_id` (invocation ID or `root:<root.ID>`), `root_id`, unique `decision_order` FK to the existing `composition_facts` lifecycle row, and canonical `compositionClosure{RootID, LifecycleOperationID, Reason, Freeze}`. Its Freeze retains the original target Run/Sequence, lifecycle fact ID as Trigger, the same lifecycle decision order, status `cancelled`, and exact committed reply/Invoke/external IDs. Reason is `subject_retired`. The row is inserted inside the same owner transaction as the retirement, lifecycle result and fact; there is no artificial second ordering transaction. No old successful freeze is mutated.

`compositionScopeFor` derives the immutable member set; `captureCompositionIntents` shares the existing exact committed-prefix scan between ordinary freezes and retirement closures. Existing non-success freeze must precede the retirement order and remains authoritative. If retirement closes first, later draining peer success/failure cannot insert a competing freeze or change its reply prefix. Preparation selects the earliest retained non-success evidence, with a successful freeze retained only as historical evidence when retirement intervenes. Already published results preclude a new scope closure record; the original lifecycle retirement fact still records the subject transition, and the published result itself is a permanent scope fence.

`compositionScopeClosed` now recognizes retained retirement closures and published invocation results at the resource root and every invocation ancestor. Previously allowed source authorization stays allowed; subsequent admission returns `scope_closed`, creates no children and spends no charges. Root closure evidence is retained even before any Invoke. **Root public publication remains Task2c work**, and must consult this evidence rather than local terminal status. Task2c external attempt reservation must use this same final-boundary helper; no external reservation/debit implementation is claimed here.

Outcome preparation and settlement cannot publish success after applicable unpublished retirement. They resolve the frozen reply obligations and retain `cancelled/subject_retired`, including a sole valid public response if present. R7's additional controller ruling is covered explicitly: when two frozen Send intents conflict, keep `cancelled/subject_retired`, set ResponsePresent=false and retain both original intent/candidate IDs. Cancellation does not certify a valid reply protocol; it prioritizes the retained retirement cause over an unpublished conflict-derived result. Without retirement, `failed/conflicting_replies` behavior remains unchanged. Earlier non-success freezes and already-published outcomes preserve their existing rules.

Private protocol validation/open now also gates nonempty `composition_closures`. A self-review probe showed that a retirement-only history (no Invoke/admission/new fact kind) otherwise bypassed the existing unfinished-protocol gate and reopened. It now returns ErrVersion before claiming closure replay. The table exists only in newly constructed private `alan/composition/v1` stores. Public schema4 remains gated, foundation/schema3 layouts are unchanged, and no migration or compatibility claim for earlier unfinished private protocol stores is introduced.

### Evidence and coverage

Every fix-round Go command records exact argv/environment, numeric subprocess exit and source identity in `review-fix1/commands.json`. The capture wrapper is retained as `capture-command.py.txt`; source manifests are adjacent to each output. All use `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`.

- `01-original-probe-red.log`: unchanged original overlay, exit1; accepted retirement still admitted two members. `05-original-probe-green.log` passes after the first implementation; `10-original-probe-final-green.log` reruns the unchanged overlay on final source, exit0, scope_closed=true, admission accepted=false/reason=scope_closed, zero admitted members.
- `02-retirement-red.log`: dynamic local-success retirement still admitted nested work; retirement after success freeze still published succeeded. First-cause/rollback fixtures also encountered the not-yet-created closure table; these are disclosed setup/API failures, not behavioral proof. Already-published success, no-op Cancel/Abandon and prior-failure diagnostic controls already passed.
- `03-supported-ancestor-red.log`: fully provisioned static helper retirement failed to fence a previously admitted dynamic descendant's authorized nested Invoke, exit1. It now passes in `04-retirement-observed.log` and `08-focused-green.log` without changing local success or the admitted peer's open status.
- `04-retirement-observed.log`: initial focused R7 tests pass, including exact original authorization preservation, no children/charges after closure, retirement before publication with unchanged success-freeze bytes, immutable published success, no-op controls, both first-cause orderings, duplicate operation/new repeated retirement, and injected rollback of retirement/result/fact/closure together.
- `06-closure-only-gate-red.log`: supported retirement-only store reopened without protocol validation, exit1. The private gate fix passes in08/09.
- `07-retirement-conflict-red.log`: existing conflict logic overwrote the first retirement cause with failed/conflicting_replies, exit1. `08-focused-green.log` passes the confirmed R7 precedence and retains both candidate IDs without selecting a response. Normal non-retirement conflict/failure/reply tests also pass.
- `09-executor-suite.log`: `go test ./internal/executor -count=1`, exit0. A single complete executor package run was justified by the shared lifecycle/Commit/freeze/admission integration. No redundant repository-wide, race or integration-tagged suite was run. No production/test source changed afterward; final original overlay rerun10 also passes.

The supported controls comprise the unchanged original static-member probe, `TestRetirementStaticAncestorClosesAdmittedDescendants` using only provisioned exact authority, and the retirement-only reopen probe. Direct dynamic-member tests use a **test-only exact permission seed after admission**, because supported immutable provisioning cannot know namespace-derived subjects before Create. This setup is retained and labeled in `retirementFixtureReplies`; it is outside supported immutable public deployment/provisioning and is only an internal lifecycle/closure-unit probe. It adds no production grant/delegation API and does not claim public dynamic-lifecycle authoring or replay qualification. This boundary follows the controller's fix-round refinement; seeded probes do not replace the fully provisioned controls.

`TestRetirementDynamicSucceededMemberClosesScope` additionally checks the closure's original lifecycle operation, target run/sequence, exact source fact ID/order and frozen nested Invoke ID. Other tests retain original success/failure/closure descriptor bytes across later transitions and exercise actual known-outcome settlement, not only admission fencing. `red-tests.go.txt`, `red-tests-complete.go.txt`, `06-tests.go.txt`, `07-tests.go.txt`, `final-tests.go.txt`, `final-closure.go.txt` and `tracked-fix.patch` retain source inputs alongside the original review probe.

Final exact Go/go.mod/go.sum source identity: `d525dec36ec4542882a8ec40608976b70c323030b4a9e56e3736eab2f5eecbe1`, recorded in `09-executor-suite.source.json` and the final probe manifest. The original source/report evidence remains unchanged; this section is appended. Source is gofmt'd and the Go-source whitespace check is clean. The complete staged whitespace check reports standard diff context prefixes (space before tab) in retained tracked-fix.patch; that patch evidence is preserved byte-for-byte. Controller-owned contract and qualification edits are deliberately unstaged by this commit.

### Precise Task2c closure validation handoff

The live closure reader checks canonical descriptor bytes and relational scope/root/order/status/reason fields. Complete bidirectional cross-ledger/prefix validation remains behind the existing Task2c gate. Before public activation, validate all of the following:

1. Every closure belongs to the target's retained immutable scope/root/member set and to one actual accepted RetireSubject transition with Before.Retired=false and After.Retired=true. `LifecycleOperationID` must identify that original request/result; no-op Cancel/Abandon, duplicate decisions and unrelated source membership cannot justify a closure.
2. The FK order must identify the actual lifecycle fact, with exact Kind/OperationID/ID/root/target/sequence linkage. Freeze.Trigger must equal that fact ID; Freeze.DecisionOrder must equal its order; Freeze.Run/Sequence must match the actual Before target. Canonical descriptors and relational columns must agree in both directions.
3. Reconstruct the exact reply/Invoke/external prefix from ordered member execution facts strictly preceding that lifecycle decision. Neither later delivery nor later member computation can expand it. Check all missing/invented candidates, facts and reverse links; do not trust the retained final projection alone.
4. Enforce first-non-success order against existing terminal freezes/closures and publication. A prior non-success wins; prior success freeze remains unchanged beside applicable retirement; retirement first prevents later replacement; already-published outcomes remain authoritative and require no new scope closure. Every qualifying first retirement transition needs its closure row, and every row needs that transition.
5. Apply the same evidence to root outcome/publication and last external reservation authority. Do not infer root public success from local success, refund prior authorization/reservations, or cancel draining peers. Replay must validate evidence without creating rows or dispatching work.

No prior Task2a M1 coverage claim changes. No providers, credentials, subprocess crash qualification, new dependency, schema migration or subagent was used. Scoped independent rereview is still required.
