# Phase 1I Stage 1 public API and process feasibility audit

Date: 2026-10-28. Read-only repository audit; this `/tmp` report is the only authored file. No candidates, model, production changes, provider calls, or repository files were authored. Using the Go navigation/testing skill guidance for local API inspection; substantive tests below are explicitly distinguished from source inspection.

## Question and result

Can the required weak-budget, scheduling, boundary, closure, and recovery probes be constructed at the existing exported author/schema/runtime boundary, with environmental process instrumentation kept separate?

Result: yes, within the existing single-host schema4 protocol, with an explicit qualification: lifecycle cancellation is not an `author` DSL primitive, and authentic adapter late-result recording belongs to a surviving `Store.Run` worker. Reuse the declared test-only granted operator for closure, and keep live late transport separate from fresh-process unknown recovery. No new runtime API or scheduler is presently justified.

## Inspected identity and scope

Workdir `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`; branch `codex/phase1i-contextual-contracts`; initial inspected HEAD `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`; tree `158d2a24ece0b14f99018921797a7f6f6a45a465`; `git status --short` emitted nothing. This audit does not independently establish current remote-main identity or ancestry; orchestration must supply that check.

Read the actual AGENTS.md and Phase1I assignment attachment, README, Phase1H contract/runbook, relevant qualification/review navigation, the Phase1H manuscript audit, public packages, invocation/accounting/closure implementation, and process harnesses. Platform observed: `go version go1.27.3 darwin/arm64`; root `go.mod` pins Go1.27.3 and `github.com/mattn/go-sqlite3 v1.14.52`.

Paths below are relative to that inspected workdir. They are source-inspection evidence unless explicitly listed as an executed command.

## Existing exported construction and observation boundary

`author/author.go`:

- `Definition` has `Invocations map[string]Template`, `Ports map[string]Port`, typed `Inputs/Outputs`, `Values`, `Effects`, and `Steps`.
- Public request/reply schemas belong to the public `Port`'s `Receive/Send`; `Definition.Inputs` is deployment input, not invocation input. A template entry port must be nonlocal and have an explicit caller peer and obligation.
- `Receive`, `Send`, `Set`, `Add`, `IfEqual`, `Success`, `Failure` supply enough acyclic behavior. `Request` and `Await` supply the effect companion. There is no lifecycle authoring step.
- One `Set` compiles into one ordinary pure instruction. `Label` compiles to no instruction; do not use it as the scheduling difference. Each entry in `Definition.Values` becomes a prelude `set` (`author/compile.go:105`), so account for that prelude when promising exactly one extra step. A local `Set("scratch", Integer(0))` can express a step without a `Values` declaration.

`author/invocations.go`:

- `NewTemplate(Application, entryMember, entryPort)` freezes member artifacts, immutable defaults, public schemas, nested dependencies and capabilities.
- `Template.ID`, `Bytes`, and `Schemas()` are available; `Schemas` returns fresh retained request/result schemas.
- `Invoke(pending, declaration, values, updates)` retains the operation handle in `pending` and commits updates atomically with the intent.
- `AwaitInvocation(declaration, pending, InvocationOutcomeBindings{Status, Reason, ResponsePresent, Response})` binds the original operation's authenticated outcome. Response absence binds `present=false` and does not fabricate response fields.

`author/application.go`: `Application.Provision()` derives static runs, grants and closed template/program catalogues. It supplies `Composition.Templates` when invocations exist but does not choose resource roots. Follow `author/invocations_test.go:80` and `examples/adaptive/deployment.go:23` to set `p.Composition.Roots` using exported `runtime.ResourceRoot`/`ResourceLimits`. Include every allowed dependency template ID from the retained catalogue. A nested helper must also be in its parent template's allowed dependency set; the author compiler builds that set from the declared invocation.

`runtime/runtime.go` exposes aliases for `Store`, `Provision`, `ResourceRoot`, `ResourceLimits`, invocation descriptors/outcomes/preparation/freezes, `RootCompletion`, and `Inspection`, plus `Create`, `Open`, `Inspect`, `Replay`, and `DefaultLimits`. Exported `Store` methods include `Run`, `Snapshot`, `Commit`, `AdmitInput`, `AdmitInvocation`, `PrepareInvocation`, `SettleInvocation`, `PrepareRoot`, `SettleRoot`, `Namespace`, `Close`; lifecycle methods exist on the aliased store, but the author DSL cannot author the required lifecycle intent.

Use `Store.Run` for candidate decisions and normal dispatch. The test harness may inspect public `runtime.Inspect` records and add permitted environmental inputs. Do not turn low-level exported methods into a host-side candidate policy engine. Public consumer code can import only `author`, `schema`, `runtime`, and `effects` as required; process tests can live separately under an internal test package with observation-only boundary instrumentation.

`RootCompletion` (`internal/executor/composition_inspect.go:11`) provides exact retained `Limits`, aggregate `Charges`, logical `Outcome`, local run status, pending invocation handles, unresolved effects, and original unknown effect handles. The inspect tables include the original admissions, bindings, requests, replies, freezes, reservations, publications, charges, facts, histories, observations, adapter attempts/results, and settlements. Decode records independently; do not use replay as the semantic oracle.

## Exact accounting and closure semantics

`internal/executor/invocation.go:220` (`Store.AdmitInvocation`):

1. Authorizes the immutable source intent separately.
2. In one admission transaction checks source state/revision, ancestor/root closure, declaration/schema/template permissions, capabilities, depth, cumulative root invocations, and cumulative dynamically created runs.
3. Acceptance adds exactly one invocation and `len(template.Members)` created runs, atomically with child creation, binding and request observation. A nested helper invocation adds another debit; merely retaining its template/program in the catalogue adds no debit.
4. Rejection retains a reason/outcome but adds no invocation/run charge. Relevant reasons: `invocation_limit`, `created_run_limit`, `depth_limit`, `scope_closed`, `source_closed`, and prefixed source-denial reasons. Check the actual reason rather than forcing every closure refusal to one literal.
5. Accepted charges are never refunded on settlement, failure, closure, or reopening.

Static provisioned runs are fixed costs outside `Charges.CreatedRuns`. Report total physical runs as fixed runs plus dynamically created runs. A root-only fixture has fixed1; adding one static competing peer makes fixed2; an operator adds another fixed run, ordinarily outside the candidate resource root. Do not secretly count a statically provisioned helper as a dynamic invocation.

Root is depth0; direct candidate depth1, its helper depth2; one invocation wrapper raises candidate/helper depths to2/3. One-member wrapper admission costs another `(I=1,R=1,E=0)`. A multi-member candidate admission costs `(1, memberCount, 0)` and shares its depth across all its members.

`composition_store.go:106`: nonnegative invocation/run/depth/effect ceilings permit exact zero-residual fixtures. `MaxInFlight` must remain positive even for pure effects0 fixtures. Physical `RunOptions.MaxInFlight` and persisted per-root transport capacity are distinct. `RunOptions.MaxSteps/MaxDispatch` are per-Run operational watchdogs, not persistent root charge components; set them above all selected acyclic paths, and report a watchdog exhaustion as incomplete qualification.

`composition_reservation.go:28,60`: an allowed final external/adapter reservation adds exactly one effect debit. Unknown outcome retains this debit. Denied reservations add none. Final scope closure is checked in this transaction; an earlier allowed source authorization does not force later reservation acceptance.

**Closure wording must be precise.** `compositionScopeClosed` fences invocation admission and final effect reservations after relevant root/ancestor closure. It does not retroactively revoke a committed reservation. `TestCompositionReservedAdapterMayLaunchAfterClosure` explicitly preserves an already reserved launch after closure. `TestAdaptiveClosureLateTransportCompanion` explicitly observes a second source authorization with `Allowed=true`, followed by final reservation `Allowed=false, Reason=scope_closed` and no second contact. Thus “no new reservation/admission after closure” is appropriate; “no later source authorization/physical attempt after closure” is false for this implementation boundary.

`invocation_outcome.go:291,324`: successful invocation publication requires a member freeze, admitted reply evidence, and all recorded descendant/external obligations known. Missing reply gives `failed/missing_reply`, conflicting replies `failed/conflicting_replies`; failure/cancellation can preserve a present diagnostic reply. A child's exact status/reason is separate from the enclosing wrapper's own outcome. A wrapper that maps any non-success to `Failure` produces its own `failed/member_failed`, not an impersonated child rejection/cancellation envelope.

Unknown is not an available negative receipt. `prepareAdapterWork` and `retryEligibleWork` refuse re-execution of an already allowed adapter attempt with no recorded result. Public `RunReport.UnknownAdapters` and root `UnknownExternal` retain original handles. A premature ordinary `Send` then `Success` cannot bypass the runtime's pending external join to publish success; this is not a useful candidate-owned unknown-to-success negative control without first reproducing a real gap.

## Smallest public-consumer witnesses

### W_A/W_B weak-budget counterexample

Use one static root; one-member successor S; no effects. W_A's one-member entry receives integer n, sends the required result, and succeeds. W_B's one-member entry receives n, invokes a one-member helper, awaits the original helper handle, sends the same result, and succeeds. W_B's helper is a dynamic nested dependency, not a statically provisioned member.

The root invokes W, awaits its result, then invokes S. Set root limits `MaxInvocations=2, MaxCreatedRuns=2, MaxDepth=2, MaxEffects=0, MaxInFlight=1`. Fixed cost is root1. Expected charge/admission distinction to execute: W_A+S yields `(2,2,0)` and S accepted; W_B+helper yields `(2,2,0)`, same W result, then S rejected with `invocation_limit` (invocation check precedes created-run check). Both totals can be equal at the limit; the order and identity of accepted work distinguish them.

At capacity3 both paths can admit S, with totals W_A `(2,2,0)` versus W_B `(3,3,0)`. At capacity1 W_B cannot fulfill its ordinary successful helper path; retain the actual child result/failure handling. To expose created-run refusal separately, hold invocation ceiling comfortably high while bounding runs2. A wrapper adds1 to both ceilings and one to maximum depth for the analogous two-admissions-remaining point. Do not edit authoritative DB state to obtain a residual budget; use root limits and actual enclosing/prior admissions.

These witnesses share schemas/success outputs under sufficient resources, not the equal-debit relation of Definition4.6. They are not a counterexample to Theorem4.8 on its stated equal-debit premise.

### Equal-charge extra-local-step competition

Use candidate entries that both invoke the same one-member helper and relay the same result; one adds precisely one pure `Set` immediately before `Invoke`. Place one static peer in the same root; peer competes for one admission by invoking S. Candidate admission is already spent, leaving residual1 for helper versus S. Fixed cost is root+peer2; accepted total at saturation is candidate1 plus winner1. Set invocations/runs2 (or add wrapper1 explicitly), depth2/3, effects0. Do not artificially equalize costs by adding effect requests.

Freeze both schedule sets and the local candidate relation before interpreting the result. Matching final charges alone does not establish readiness preservation or a matching weak transfer under contention. The model must determine whether the exact local relation really relates these entries; a failure of that premise is distinct from failure of congruence.

Concrete `Run` (`runner.go:51,295,364`) gives one opportunity per category per turn, selects the first sorted pending ordinary delivery, and one round-robin ready run from the lexicographic run order (`replay.go:282`). Delivery keys and invocation member subjects contain generated identities; `Create` chooses a random durable namespace. Therefore fresh-store schedule comparison can change due to name ordering as well as the inserted Set. Record actual names, pending-delivery keys and retained fact ordering; compare abstract permitted traces separately. Alpha-renaming equality does not itself make this concrete scheduler invariant under renaming. Do not change the scheduler, force IDs in the DB, or silently select a favorable random fixture.

### Boundary, absence and hiding

For bare and one-wrapper fixtures, run success+reply, explicit failure+diagnostic reply, explicit failure without reply, success without reply (`missing_reply`), and rejected admission. Bind status/reason/present separately and inspect the original child's outcome plus wrapper outcome. A permanently absent external result is a pending open run, not the failed/missing_reply case.

Hide only declared internal names/communication under one consistent renaming. Retain public result correlation, equality/distinctness, root affiliation, dynamic debit order, final reservation events and endpoint contacts. Program/artifact hashes and pure instruction counts can be diagnostic internals outside the guarantee, but must remain in source/evidence provenance.

### Pure SIGKILL/reopen

Use the actual A/B artifacts, acyclic and unpadded if a committed boundary is available. The existing `invocation.admitted/post-commit`, `execution.committed/post-commit` (or `terminal`), and `invocation.result-recorded/pre-consumption` boundaries suffice. Select a cut by immutable original handle/member and validated retained records, ACK it over inherited pipes, capture inspect output, SIGKILL, verify real signal exit, launch a different fresh process that only `Open`s and `Run`s the retained store, then check original operation/member continuity, charges, epoch renewal and terminal result. No recompilation, resubmission or replacement root on reopen.

The process observer is test-only, outside candidate artifacts. `Store.Commit` calls its observer while holding `Store.mu`; calling `s.Snapshot` from that observer deadlocks. The Phase1H harness uses an explicitly read-only SQL probe for cut selection and public `Inspect` for full validation. For smaller candidates, boundary payloads and public `Inspect` may avoid copying that SQL probe at all. Candidate code must never see SQL/internal APIs.

### One controlled unknown attempt and reopen

Use one public `Request/Await` pair against one loopback endpoint. Endpoint ACK must identify the exact original operation/key and record that one attempt entered/applied. Hold or close its response so no known receipt is recorded. At an acknowledged retained attempt/boundary, kill the process, wait for the original connection to end, and fresh-process Open/Run with the same endpoint. Required checks: one original allowed attempt/reservation, effects1 retained, same original unknown operation/key, zero additional endpoint contacts, no fabricated result/settlement or successful public completion. If pure invocation surrounds this leaf, count outer candidate/leaf admissions separately. A `(Disposition=unknown)` adapter result is deliberately not inserted as a result row; absence plus allowed attempt is the retained unknown state.

`adapter.authorized/pre-transport` alone establishes reservation but not a remote attempt; also require endpoint contact ACK to claim an external attempt happened. `runner.waiting/in-flight`, or `adapter.response/pre-record` after a controlled response-loss classification, provides useful independently inspectable cuts. Do not infer remote mutation from ALAN's attempt record alone.

### Separate live late-transport closure

Reuse the ordinary scoped static peer companion: first effect is genuinely in flight; endpoint holds a response; trigger the pre-granted operator input; wait for actual lifecycle decision and immutable root cancellation; release the authentic original response into the same live Run; let peer consume it and try a second request; require the second final reservation to be `scope_closed` with no second endpoint contact. Preserve effects1, original operation/key, late result and root cancellation.

No exported fresh-process adapter receipt reconciliation API exists: `recordAdapterResult`/`settleAdapter` are internal owner-accounting paths. Do not claim the held late transport survives SIGKILL, combine it with fresh-process unknown recovery, or synthesize result observations manually. These are distinct profiles/schedules by design.

### Authentic negative controls

The minimal decisive schema/success-preserving violation is an extra uncontracted helper invocation (as above), executed through normal author Invoke and Run and rejected by an independent debit/guarantee checker. A payload-correlation violation can use an ordinary author variant that returns the wrong request identity in a declared application payload; distinguish that from forging runtime observation provenance, which ordinary author APIs do not allow. Do not present malformed artifact rejection, bad wiring, synthetic inspect-record mutation, or a fixture setup error as an executed semantic negative.

## Reusable tests and machinery

- `author/invocations_test.go:80` / `TestInvocationPublicRuntimeEndToEnd`: smallest direct exported invocation/root setup; actually passed in this audit.
- `author/external_test.go` / `TestExternalConsumerCompileRunReopenReplay`, `author/testdata/consumer.go`: genuine external Go module, public-import guard, built child modes, race instrumentation; uses bounded graceful Run/reopen, not SIGKILL.
- `examples/adaptive/consumer/go.mod`, `contract_test.go:29`: separate-module public boundary and public Inspect/replay checks. Root `./...` excludes this module.
- `internal/boundary/boundary.go`: observation-only events; inherited fd3 ACK/fd4 release, filters by exact operation/subject/epoch/sequence, unique scenario/token/store, EOF is an error. Can be reused directly by an internal integration harness without copying historical archive infrastructure.
- `internal/compositiontest/process_test.go`: `validAck`, `durableAck`, process `startEndpoint`, `wait`, origins/correlation checks, pure direct/nested cuts (`TestProcessRecoveryCuts`). Helpers are unexported test-package code, not a reusable public support library; reuse the pattern or a narrowly factored helper, not an import that cannot compile.
- `internal/compositiontest/closure_test.go`: `addOperator`, `cancelAsOperator`, `TestOperatorClosureAtCompositionCuts`, with pre-admission/accepted/recorded/partial cuts. Operator is a declared environmental fixture and uses low-level executor instructions; do not copy it into candidates or claim the complete fixture is an external-only consumer.
- `internal/adaptivetest/process_test.go`, `cuts_test.go`: real SIGKILL, fresh PID, retained artifacts and invariants. `TestProcessAdaptiveCuts` covers pure recovery; `TestAdaptiveUnknownSibling` covers original pending unknown and no repeat; `TestAdaptiveRootDenial` covers retained spending/refusal. Their adaptive frontier-specific probes and128-step padding are unnecessary for tiny Phase1I programs.
- `internal/adaptivetest/closure_test.go`: `TestAdaptiveClosureAfterDecision` (kill/reopen/operator/denied admissions) versus `TestAdaptiveClosureLateTransportCompanion` (live authentic receipt after closure). Keep that distinction.
- `internal/adaptivetest/fixtures_test.go`: controlled HTTP adapter/provider, contact ledger, explicit held-contact/release handshakes, watchdog scaling. Copy only tiny required pieces; no paid credentials/provider needed.
- `internal/executor/composition_limits_integration_test.go`: `TestCompositionReservationClosureAfterSourceAuthorization`, `TestCompositionReservedAdapterMayLaunchAfterClosure`, `TestCompositionUnknownReopenKeepsOriginalAttemptAndBudget`, `TestCompositionEffectCeilingSharedAcrossAdapterAndLegacy`, `TestCompositionNestedCapacityOneExecutesLeafTransport`; relevant baseline/accounting regression selectors, not public candidate implementations.

## Commands actually executed by this audit

Two initial commands failed setup because the sandbox denied the default Go build cache under `/Users/jacob/Library/Caches/go-build`; no selected test ran. The failed commands were the focused author test and integration `-list` command below without `GOCACHE`. This was avoided by using a fresh permitted `/tmp` cache, without repository changes or escalation.

```
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1i-audit-gocache go test -count=1 -run '^TestInvocationPublicRuntimeEndToEnd$' -v ./author
```

Exit0; emitted `=== RUN TestInvocationPublicRuntimeEndToEnd` and PASS (0.02s test; package0.144s). This is an actual current-baseline direct invocation/root success/replay test, not any Phase1I candidate qualification.

```
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1i-audit-gocache go test -tags integration -list '^(TestProcessAdaptiveCuts|TestAdaptiveRootDenial|TestAdaptiveUnknownSibling|TestAdaptiveClosureAfterDecision|TestAdaptiveClosureLateTransportCompanion)$' ./internal/adaptivetest
```

Exit0; listed all five exact intended tests. `-list` compiles and lists; it does not run their process schedules. No claims that those schedules passed in this audit.

## Suggested qualification, to be run and recorded separately

Use the same offline toolchain/module environment, an approved writable build cache, unset `OPENAI_API_KEY`, and fresh artifact directories. First use `go test -list` to confirm selectors. For baseline semantics relevant to this scope:

```
go test -count=1 -tags integration -run '^TestComposition(ReservationClosureAfterSourceAuthorization|ReservedAdapterMayLaunchAfterClosure|UnknownReopenKeepsOriginalAttemptAndBudget|EffectCeilingSharedAcrossAdapterAndLegacy|NestedCapacityOneExecutesLeafTransport)$' -v ./internal/executor
go test -count=1 -tags integration -run '^TestExternalConsumerCompileRunReopenReplay$' -v ./author
```

New Phase1I modules/tests should have exact focused model, external-consumer and process selectors, plus matching `-race` builds where transport/process code contributes. Do not run whole historical Phase1H archives merely to fill evidence volume. If no reducer/compiler/accounting is changed, a targeted baseline set plus new witnesses is appropriate; if production code changes, broaden to its actual affected required gates.

Repository README's full gates are root default/integration/race tests, root vet default/integration, mod verify, gofmt; separate `examples/worktree/consumer` and `examples/adaptive/consumer` test/race/vet/mod-verify gates. Record which apply to final scope rather than claiming root `./...` covers nested consumers. README allows60m default/integration,600m root race; worktree consumer240s; adaptive consumer15m/60m race. Process tests require loopback sockets, child processes, SIGKILL and actual child race instrumentation. A sandbox capability failure must be reported as blocked evidence, not treated as a test result.

For every final run record full source/tree and contributing-file identity, exact command/env, exit code, actual selected case count, retained record/event paths, endpoint ledger and fresh-process PIDs. Existing reports are historical evidence with original identities, not current-source reruns.

## Remaining obligations for Stage1 freeze

The contract/model reviewer must decide the exact separately checkable local relation, admissibility at reachable shared-budget prefixes, and selected candidate-dependent guarantee. This API audit does not establish a substitution theorem, scheduler confluence, independent model validity, independent candidate authorship, or completion of any required Phase1I process case. It establishes concrete feasibility and exposes boundary assumptions that must be frozen before candidate authors begin.

In particular, retain the original weak-budget counterexample and do not reclassify unequal debits as a failure of Theorem4.8's equal-debit premise; check the extra-Set local relation and permitted trace sets independently; treat full schedule confluence and the manuscript's existing general loop-ranking claims as separate open obligations. No external citations were consulted or verified, and no manuscript changes are proposed by this API-only workstream.
