# Phase1G Task2c — complete composition protocol integration

Implementation base: `a66760fb23c93e333d2244508eab4eeb3f1f7c82`, branch `codex/phase1g-data-composition`. Scope: Task2c integration and its self-review evidence. This completes the public bounded protocol surface needed by Task3; it is not the ordinary collection application, process-kill qualification, independent scoped approval, or full Phase1G acceptance. No earlier evidence/review is rewritten.

Normative inputs: composition contract R1–R7 and controller-authored R8, Task2a/Task2b reports and retirement fix handoff. R8 was resolved during this task using the concrete two-call counterexample below; the controller owns and commits the contract change separately. Source/test/evidence changes in this slice do not stage that contract file.

## Result and implementation

The existing `Store.Run` now schedules committed Invoke delivery, actual bound member reductions, correlated outcome publication/consumption and root publication. Pending nonready invocation joins are excluded from the deliverable queue so they cannot starve eligible work. Selection and ordinary Replay use the same immutable `boundCompositionProgram` resolution as Commit. There is one owner loop, one SQLite transaction path and the existing adapter/external workers; no second scheduler or executor.

The owner tracks physical active counts globally and by retained resource root. A launch needs both limits; it scans past a saturated root. Waiting logical parents own no physical slot. A returned unknown adapter candidate releases its local worker slot, creates no known result, remains charged, is not automatically retried, and does not prevent independent eligible work. Larger recovery `MaxInFlight` cannot raise the persisted root ceiling. These are local owned-worker counts, not a claim that unknown remote activity stopped.

Adapter and legacy external attempts now share `compositionReservationDecision` and `recordCompositionReservation`, called in the same owner transaction that inserts the attempt. Separate source authorization is preserved. Grant/schema checks precede reservation; ancestor/root closure and cumulative effect allowance are checked before an allowed reservation/debit. Every scoped allowed or denied attempt has a reservation fact and digest of the exact retained attempt. Each allowed attempt charges one effect; denied attempts charge zero. Existing attempt retransmission reuses the retained decision. Normal `scope_closed`/`effect_limit` denials become existing correlated negative adapter/receipt outcomes and do not fail the whole runner. For adapters the ordinary candidate disposition remains `source_denied` (or existing `grant_denied`); the precise reservation reason remains inspectable in its original attempt/reservation. Unknown candidates still create no `adapter_results` row.

A reservation granted before closure may physically launch afterward; a separate test executes the controlled adapter after cancellation and root publication and retains the late result. Authorization alone does not confer this permission. Previously charged unknown work is neither refunded nor rebound. Existing scoped Retry remains the durable `invocation_retry_unsupported` decision, with full replay validation; dynamic origins never masquerade as static `run_origins`.

Success joins resolve every frozen committed Invoke admission, frozen public reply admission, accepted descendant outcome and known external obligation. Missing and conflicting replies become protocol failure only after their frozen admissions finish. First non-success/retirement causes retain their original prefix and may publish early without waiting for every draining member. Public root completion uses the same rules, except a provisioned root has no template public reply requirement. Both root and invocation outcomes retain immutable original identities; subsequent preparation returns a previously published outcome rather than recomputing it from later facts.

### Primary transitive-join counterexample

`16-transitive-join-red.log` (Go exit 1) shows an admitted outer scope whose helper fails after its entry has admitted a leaf. The outer failure publishes, the root finishes locally, and a second root Invoke is rejected. Before the fix the root incorrectly prepared `succeeded` while the admitted leaf had no outcome. The exact unchanged test function and schedule are retained in `transitive-join-counterexample-input.go.txt` and `transitive-join-counterexample.md`; original source identity is `5d08c76f467f30c2ea75eb2a9aa4973f744c9f2590e8b28f369da31c65f1c877`.

The corrected success join traverses accepted descendants and their committed invocation/external obligations even below an already-known failed scope. Rejected admissions are retained known negatives and create no subtree. An accepted descendant with no publication or an external intent with no known result prevents ancestor success. This does **not** require every open draining peer to terminate, introduce group cancellation, or change the child's early non-success publication. Prefix replay applies the same transitive obligation rule using only facts already preceding publication. The unchanged regression passes in17 and the final suites.

### R8: physical transport has one public Store entrypoint

The existing exported synchronous `Store.DispatchExternal` bypassed the owner accounting. `35-dispatch-bypass-red.log` is a real two-call causal loopback execution, not a hypothetical trace: the scoped case records contacts=2, physical peak=2, source authorizations=2, attempts=2 and reservations=2 under persisted root capacity1. The unscoped-in-composition case also records contacts=2/peak=2 and no root reservations. Both handlers were held until the second contact, without sleeps. The exact executable input is `dispatch-counterexample-input-fixed.go.txt`;34 preserves its preceding fixture compile error separately.

Controller R8 requires every physical Store dispatch in complete `alan/composition/v1` stores to use `Store.Run`, including their unscoped executions because those share the global ceiling. `DispatchExternal` now returns exported `ErrDispatchMode` (`composition transport requires Store.Run`) before source authorization, attempt reservation, contact or fabricated effect outcome. The original committed intent remains eligible for Run.36 passes the unchanged two-call assertions, checks zero refusal-side mutation, then runs the same intents successfully through Run at physical peak1. Existing schema3 synchronous positive tests also pass. All exported Store methods were inspected: Run and DispatchExternal are the only physical transport entrypoints; the latter is refused for this profile. Public standalone adapter/HTTP APIs outside Store are outside this executor guarantee.

## Public API handoff for Task3/4

The Task2b author API is unchanged:

- `author.NewTemplate(application, entryMember, entryPort)` retains the reusable template.
- `Definition.Invocations map[string]Template` declares finite permitted targets.
- `Invoke(pending, template, values, updates)` commits original operation correlation and updates atomically.
- `AwaitInvocation(template, pending, InvocationOutcomeBindings{Status,Reason,ResponsePresent,Response})` consumes the authenticated immutable envelope. Branch on response presence before payload use.
- `Application.Provision()` gathers the closed catalogue. Trusted deployment must fill `Composition.Roots` with exact generation1 static members, allowed template/capability IDs and `ResourceLimits{MaxInvocations,MaxCreatedRuns,MaxDepth,MaxEffects,MaxInFlight}`. Static Invoke artifacts without an assigned root now reject at provisioning.

New/activated runtime behavior:

- `runtime.Create(ctx,path,provision,limits)` accepts nonnil Composition whose Version is omitted or `alan/composition/v1`; it persists the explicit protocol version. `Open`, `Inspect`, `Replay` support complete schema4 after full validation. Public author/runtime end-to-end coverage uses the omitted author-supplied Version, with no private constructor or seeded grant.
- `Store.Run(ctx, RunOptions)` executes the entire graph. Existing `AdmitInvocation`, `PrepareInvocation`, and `SettleInvocation` remain available for explicit durable protocol steps; they do not provide a second transport dispatcher.
- `Store.PrepareRoot(ctx, rootID) (InvocationPreparation,error)` exposes pending structured completion or the original published outcome. `Store.SettleRoot(ctx,rootID)` publishes an eligible immutable root outcome; automatic Run calls the same path.
- `RunReport.Roots`, `ReplayReport.Roots`, and `Inspection.Roots` are `[]RootCompletion`. Each has `RootID`, original `Run`, `LocalStatus`, persisted `Limits`, cumulative `Charges` (Invocations/CreatedRuns/Effects), optional `Outcome`, `Pending`, `PendingInvocations`, `UnresolvedExternal`, and `UnknownExternal`. `Outcome` is present only after publication. A cancelled/failed scope can retain unresolved external IDs. Original inspection tables remain exposed alongside all complete-profile tables.
- `runtime.ErrDispatchMode` aliases the explicit new-profile unsupported synchronous transport-mode error. Use Run for physical dispatch.

Root scope identity is `root:<resource root ID>`. Invocation scope identity remains its canonical invocation ID. Root outcomes use `alan/root-outcome/v1`; invocation result and observation identities retain the existing Task2b domains. Preparation pending reasons are `members`, `reply_admission`, `invocation_admission`, `descendant_outcome`, and `external_outcome`. Local quiescence still does not imply completion.

### Version matrix

| Provision/store | Result |
| --- | --- |
| No Composition, legacy scalar or collection-only artifacts | Unchanged schema3 public Create/Open/Inspect/Replay and synchronous DispatchExternal. Invocation-bearing artifacts cannot enter schema3. |
| Public Composition with omitted Version or `alan/composition/v1` | New schema4 complete profile, external key version2, full validation before activation; no conversion. |
| Existing private `alan/composition-foundation/v1` | Retains its private foundation path and guard; public Open/Inspect/Replay and explicit public Create reject it. It is never promoted. |
| Earlier unfinished private `alan/composition/v1` layouts | No migration/compatibility claim. Missing ledgers fail with explicit query/validation errors, before epoch advancement. Open/Replay create no missing tables or rows. |
| Unknown schema/profile/binding/program version | Fail closed; rejected Open does not advance activation authority. |

Program versions are unchanged: invocation/collection artifacts `alan/1g/v1` with compiler `alan/author/v2`; legacy scalar `alan/1e/v1` with `alan/author/v1`. Bound legacy member artifacts keep their original bytes/IDs. No dependency or migration was added.

## Persistence and ordered evidence handoff

Schema3 DDL remains unchanged. Complete schema4 retains Task2a/2b tables and adds:

| Table | Exact retained relationship |
| --- | --- |
| `composition_deployment` | Singleton canonical original static Subjects/Runs, allowing reverse validation of every static root origin. |
| `composition_permissions` | Singleton canonical deployed static permissions; all current permissions must equal these plus exact template-derived dynamic grants. |
| `composition_reservations` | `operation_id` FK to original intent, root FK, canonical operation/root/kind/run/sequence/allowed/reason/attempt digest. One per scoped adapter/legacy attempt, including denial. |
| `composition_root_publications` | `scope_id`, root FK, unique ordered fact FK, canonical immutable outcome. The fact FK is deferred within the one publication transaction. |

Existing live protocol tables remain `composition_admission_decisions`, `composition_invocations`, `composition_bindings`, `composition_members`, `composition_requests`, `composition_replies`, `composition_freezes`, `composition_closures`, `composition_publications`, `composition_results`, `composition_outcome_deliveries`, `composition_charges`, plus the original profile/root/static-member/template/fact tables. Invocation charges remain one admission plus all member runs; static fixed cost is excluded. Effect charges use the same charge ledger with Kind=`effect`, Effects=1 and exact reservation decision order.

Ordered `composition_facts` kinds now include `authorization` and `external_result` in addition to execution/lifecycle/admission/reservation/reply/freeze/outcome. Scoped source authorizations and known adapter results/legacy receipts append their facts inside their original owner transactions. Unknown candidates append no result fact. Every fact has canonical ID, roots, run/sequence, operation and exact source-row digest. Root publications use the outcome kind with the root scope as operation. Denied legacy attempts are already known at their reservation fact; adapter negative results follow their actual result fact. Freeze remains adjacent to its terminal trigger; retirement closure uses the actual lifecycle decision's existing order.

Reconstruction validates both relational columns and canonical descriptors. It checks static and dynamic origins in both directions; original artifacts/initial states; exact member/binding/request inputs; all derived grants; original admission decisions and permanent denial; conserved invocation/run/effect charges; all candidate/result/publication/delivery/observation links; and every scoped fact source/reverse edge. An invented static origin, a forged dynamic grant, a missing whole ledger, or a reply candidate surviving deletion of both transport edges fails. No dispatch registry, credentials, repairs, admission, new runs or database writes are available to read-only Replay.

The prefix validator reconstructs active membership, per-run sequences/projections, retirement, authorized operations, admitted children, scope closures, frozen sets, known results, published outcomes and cumulative charges. It enforces source authorization before admission/reservation/reply, creation before member execution/request consumption, reservation before result, result/publication before consumption, first applicable failure/retirement, exact frozen intent prefixes, joins before success and immutable output identity. Source authority and lifecycle target authority remain separate; existing lifecycle decision policy is reused, with the scoped Retry restriction applied explicitly. Read-only replay does not fabricate new generations or advance epochs.

Task2a order-only message, terminal-execution, terminal-lifecycle (zero and nonzero source sequence), and lifecycle-result tests remain unchanged and pass. New protocol and external result ordering probes extend their coverage. Deferred Task2a M1 is now explicitly covered by `TestCompositionFoundationReservedTableGuardValidFK`: its inserted reserved invocation has valid intent/template/root foreign keys, PRAGMA foreign_key_check is clean, and validation reaches the precise foundation ErrVersion nonempty-protocol guard. The original weaker test and evidence remain untouched.

## Task4 observational hooks

- `invocation.admitted`, phase `post-commit`: emitted after the real admission/children/charges transaction commits. `Intent` is the original Invoke operation; `Attempt` is invocation ID; `Root`, Subject, Generation, Sequence identify the original source/root.
- `invocation.result-recorded`, phase `pre-consumption`: emitted after immutable publication plus authenticated observation/delivery commit. `Attempt` is outcome ID; `Intent` and `Root` retain original correlation. Parent consumption is still a later Commit.
- `boundary.Event.Root` is an optional exact filter alongside existing fields. Existing `execution.committed` with selected observation remains the work-acceptance boundary.

`TestCompositionPostCommitBarriers` queries the actual committed rows at these hooks and checks the result observation is unconsumed. No hook manufactures state. Actual subprocess SIGKILL at these boundaries remains Task4; this slice makes no process-kill qualification claim.

## Verification and retained original failures

All commands use `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`. `capture-command.py.txt` captures each real Go subprocess's numeric exit, complete output and an exact SHA256 manifest of tracked/untracked Go/go.mod/go.sum sources. Wrapper shell exit0 is not used as a substitute for the stored Go exit. `commands.json` is the command authority; `.source.json` and `.log` files are adjacent. No source snapshots or original failures were replaced by later green output.

Behavioral RED/GREEN pairs:

- 01/02: automatic Run ignored Invoke (zero admissions/publications/computed members) → two computed invocations/four completed member runs.
- 06/07: effect ceiling and ancestor closure allowed a new adapter reservation → durable correlated denials, no second charge, original source authorization preserved.
- 08/09: same root exceeded persisted physical capacity while independent root existed → skip saturated root and keep the independent root progressing.
- 10/12: complete dynamic history was rejected by the temporary private replay gate → accepted by full reconstruction, unchanged facts.
- 16/17: primary transitive unresolved-grandchild counterexample → ancestor success stays pending.
- 19/20: published preparation lost original identity, and an invented static origin passed → original publication identity plus immutable deployment-origin reconstruction.
- 23/25: public complete profile was still gated → public create/run/replay/inspect/reopen supported with preserved state and correct activation epochs.
- 27/28: deleting both public reply transport edges left an accepted candidate/publication → explicit reverse-edge rejection.26 only reached the preexisting FK rejection with acknowledgments retained and is not claimed as this monitor's behavioral RED.
- 29/30: an unscoped static Invoke program entered deployment → rejected before activation.
- 35/36: causal two-call synchronous capacity bypass → explicit no-mutation API-mode refusal and bounded normal Run.

03 is an expected missing-API compile RED,04 is the initial root publication FK staging error,11 is an implementation compile error,24 is an obsolete test import compile error, and34 is a counterexample fixture-path compile error. Their output is retained and none is mislabeled behavioral proof.13's only suite failures were the three intentionally temporary Task2b private-gate expectations; their original full input bytes are retained in `temporary-gate-inputs.go.txt`. Current tests preserve the input schedules and now expect successful complete validation/reopen. Genuine foundation/unknown-version/corruption rejections remain.

Additional positive/adversarial coverage includes missing/delayed/conflicting replies, failed entry and failed internal-helper paths, normal and retirement closure orders, supported static ancestor retirement, scoped Retry denial, request/outcome consumption, adapter unknown with independent work, unknown reopen without replaying transport, shared adapter/legacy allowance, legacy late receipts, earlier reservation launching after closure, two-root capacity and nested actual leaf transport at capacity1. Nested saturation uses causal provider/runner barriers and checks independent pure computation before release; no sleep establishes progress. Public recovery preserves cumulative admission and effect budgets and cannot silently raise a persisted root limit.

Final verification on exact Go/go.mod/go.sum source identity `32a2aa026b7be64c082d9672331daf338384ee118294338afdc12a03ec709a8b`:

| Evidence | Exact Go command | Numeric Go exit |
| --- | --- | --- |
| `39-default-final.log` | `go test ./... -count=1` | 0 |
| `40-integration-final.log` | `go test -tags=integration ./... -count=1` | 0 |
| `41-scoped-race.log` | `go test -race -tags=integration ./internal/executor -run '^TestComposition(PhysicalCapacity\|NestedCapacityOne\|UnknownIndependent\|ReservedAdapter\|SynchronousDispatch)' -count=1 -timeout=90s` | 0 |

Default passed all 15 tested packages (2 packages had no tests). Integration passed all 18 tested packages (alan-resource had no tests). The scoped race run passed changed worker-ownership, held-transport, unknown-accounting, and synchronous-dispatch refusal tests. No Go source changed afterward. `final-format-checks.json` records numeric exit0 and empty output for `gofmt -l` on all changed/new Go source and `git diff --check`. The subsequent complete staged whitespace check returns exit2 only for a preserved blank line at EOF in `transitive-join-counterexample-input.go.txt`; that retained original function excerpt is not rewritten to silence an evidence-format warning. Earlier full default passes32/37 are retained, but39/40/41 are the final-source claims. Standalone whole-tree vet, whole-tree race and process/archive qualification remain Task4.


## Self-review findings and limits

Self-review found and fixed the transitive negative-outcome join hole, missing static-origin reverse authority, reply-candidate reverse admission gap, unscoped static Invoke provisioning gap, immutable published preparation identity, and the separate synchronous physical-dispatch bypass resolved by controller R8. Earlier Task2b retirement-only/foundation gates are explicitly distinguished from complete protocol validation. Dynamic lifecycle tests that manually seed permissions remain labeled internal probes in the earlier report; they are not used as public deployment/replay claims here. Supported static lifecycle paths are replayed without seeded permissions.

Public transport capacity is a single-host owned-Run guarantee. Standalone adapter/HTTP calls outside Store are excluded. Pure local reductions remain serialized and finite; no universal liveness or unbounded proof is claimed. Successful finite traces and corruption probes are reachability/checking evidence. Scope failure can be known while required external evidence remains unresolved; both remain visible. The runtime does not choose retries, assess evidence quality, aggregate collection item policy, clean up/cancel descendants, delegate lifecycle capability, migrate old private layouts, or reopen a logical scope.

Task3 must build the ordinary reusable collection application using the unchanged public author API. Task4 owns complete application/process SIGKILL qualification, faulty monitor variants, final whole-source race/vet/archive and final acceptance. Independent Task2c scoped review remains required before proceeding past that gate. No live providers, credentials, new dependencies, general migration, parallel source implementer or reviewer subagent was used in this slice.


## Task2c review fix round1 — I1 (2026-10-26)

Fix base: `168f4d74b51a970def5996d23f01e9d01865ae07`. Independent review `review.md` found **I1: Replay accepts ordinary-message consumption before source authorization**. The retained public Create/DeliverMessage/Commit probe only swaps the source authorization and receiver consumption facts, consistently rewriting relational order and descriptor Order. Ordinary records, admission/observation bindings, exact source descriptors/digests and foreign keys remain intact. This was a read-only reconstruction defect; no evidence showed a live writer producing the impossible order.

The complete-profile consumption validator now resolves the selected ordinary-message or runtime-lifecycle observation through its already validated admission to the exact source intent. It obtains that intent's scoped authorization evidence and requires that exact fact ID in the validated prefix, in addition to the existing execution/target-decision predecessor. Genuine unscoped sources produce no represented authorization fact and remain valid. Foundation validation and its resolver, live authorization/admission writers, public API and schema are unchanged. Source-denied lifecycle observations never require an invented target-decision fact.

New permanent public Replay tests are in `internal/executor/composition_consumption_test.go`. The message test covers the scoped order-only rejection and valid unrepresented-source control. The mutation helper verifies every ordinary inspection table remains equal, every fact identity/source digest remains equal except order, and foreign-key cleanliness. The unchanged Task2a tests retain message, terminal execution, terminal lifecycle sequence0/after-step, lifecycle-result causal rejections and genuine unrepresented/local-input positives.

The lifecycle audit uses real public source Commit/end, denied AuthorizeIntent and SettleLifecycle calls. Baseline Replay accepts the authenticated late source-denied observation and there are zero lifecycle target-decision rows. Moving the denied authorization before the source closing execution is rejected by the preexisting `authorization before source prefix` guard. This control **passed before the fix** and is not a new behavioral RED. The closed source cannot consume the late observation; the audit does not claim a new reachable source-denied consumption schedule. The new explicit authorization check also handles runtime-lifecycle consumption if encountered, without changing lifecycle policy.

All executions used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`. Exact command arrays, numeric subprocess exits, outputs and per-file source manifests are retained under `review-fix1/`; `capture-command.py.txt` is the capture script. Wrapper shell exits are not substituted for Go exits.

| Evidence under `review-fix1/` | Exact Go command | Numeric Go exit | Source identity |
| --- | --- | --- | --- |
| `01-original-overlay-red.log` | `go test -overlay /tmp/alan-task2c-independent-review/overlay.json ./internal/executor -run '^TestIndependentCompositionMessageAuthorizationPrefix$' -count=1 -v -timeout=30s` | 1 | `32a2aa026b7be64c082d9672331daf338384ee118294338afdc12a03ec709a8b` |
| `02-permanent-regressions-before.log` | `go test ./internal/executor -run '^TestComposition(MessageAuthorizationPrefix\|SourceDeniedLifecycleAuthorizationPrefix)$' -count=1 -v -timeout=30s` | 1 | `2f2d00e6cf890f9fbd707fb159aff97f0773624cf86c53b171cca778b182045d` |
| `03-focused-green.log` | `go test ./internal/executor -run '^TestComposition(MessageAuthorizationPrefix\|SourceDeniedLifecycleAuthorizationPrefix\|SelectedObservationCausalOrder\|SelectedObservationWithoutScopedPredecessor)$' -count=1 -v -timeout=30s` | 0 | `fc2b0465eea920d6b04cd3c3896c6be909ba5a6b292ed9ebf9c58ff6a1ebf282` |
| `04-original-overlay-green.log` | `go test -overlay /tmp/alan-task2c-independent-review/overlay.json ./internal/executor -run '^TestIndependentCompositionMessageAuthorizationPrefix$' -count=1 -v -timeout=30s` | 0 | `fc2b0465eea920d6b04cd3c3896c6be909ba5a6b292ed9ebf9c58ff6a1ebf282` |
| `05-executor-package.log` | `go test ./internal/executor -count=1 -timeout=120s` | 0 | `fc2b0465eea920d6b04cd3c3896c6be909ba5a6b292ed9ebf9c58ff6a1ebf282` |

Original overlay RED output: `Replay error = <nil>` followed by `Replay accepted ordinary-message consumption before its source authorization`. Unchanged-overlay GREEN output: `Replay error = corrupt durable state: observation consumed before source authorization`. Permanent pre-fix output independently shows the same nil-error failure for scoped message consumption, with unrepresented and source-denied controls already passing. Package GREEN output: `ok github.com/JakeFAU/alan/internal/executor 5.001s`. Exact full output remains in each named log.

The original review input remains byte-for-byte unchanged, SHA256 `577a4fc817fcedad612d6092d2f1f144248db2e2000bc04befaed81413e4997c`; both executions use the identical original overlay path and input. `verification.json` records equality against the retained input/map, original review artifact hashes and exit0/empty-output formatting checks. Reproduction on another host requires only remapping the overlay paths. Source manifests hash checkout Go/go.mod/go.sum bytes; the overlay's additional virtual test is identified separately by its input SHA256.

Self-review checked exact intent/admission linkage, exact seen fact identity, unrepresented-source behavior and foundation isolation. The executor package was run because the shared read-only validator is used by public Replay/Open across ordinary message, lifecycle and composition fixtures; it passed. No full repository, integration-tagged whole suite or race rerun was performed for this narrow validator change. The earlier full-suite claims above remain evidence for their earlier source identity, not the new one. No production Go source changed after the three GREEN executions. Controller-owned `02-qualification.md` was not edited or staged. No live writer policy, contract ruling, API, dependency or migration was added. Independent scoped rereview is still required; Task3/4 application/process acceptance remains outside this fix.
