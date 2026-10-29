# Phase1I public runtime/consumer/process report

## Question and bounded result

Investigated whether ordinary exported-API applications using independently frozen A/B fit the K1 products on named contexts/resource profiles, and preserve original operation, observation, charge and external-attempt facts across actual process interruption, root cancellation, unknown recovery and replay.

Result: all236 selected normal leaves and the matching236 race leaves PASS, including tests that require authentic faulty programs and the root-success correspondence counterexample to be rejected. This is sampled retained-runtime evidence under fixed bounds, not exhaustive concrete-runtime conformance, a compiler proof, or a universal substitution theorem. Final controller source freeze/whole-branch review is separate.

Decisive negative: both frozen A and B, in a separately identified parallel fixture with32 ordinary acyclic peer Set instructions before the original S Invoke, produce root succeeded, peer failed, S rejected scope_closed at original I=R3/D2, id1/value0/answer, Closed=false and initial entitlement=true. The unchanged optimistic CheckCandidateContext rejects; independently supplied CheckBoundaryContext accepts the actual terminal tuple while retaining its completion counterexamples. The original parallel constructor is unchanged. This is failure of the optimistic C[K1]Gc/runtime correspondence premise under permitted pure-stutter scheduling latitude; it does not falsify an implication whose C[K]Gc premise is unestablished. No runtime repair attempted.

## Owned changes and interfaces

- examples/substitution/consumer: separate Go module; CLI init/run/recover/inspect/replay with -db/-candidate/-context/-id/-value/-mode/-i/-r/-depth. init alone authors/provisions; run/recover only Open and automatic Run. Standalone imports exported author/runtime and public example packages. Root ./... excludes it.
- examples/substitution/probes.go: Probe(name) constructs distinct weak-a/weak-b constant-zero artifacts, p/q sharing one helper (q has precisely one additional ordinary Set immediately before Invoke), and valid wrong-id/extra-helper artifacts. DelayedParallel builds a separately identified ordinary public peer program; no scheduler/identity forcing or authoritative record editing.
- examples/substitution/effect.go: separate K1E EffectCompanion using ordinary Request/Await against one controlled capability; optional second request distinguishes source authorization from final scope reservation.
- internal/substitutiontest: integration-only matrix, normalization, independent model monitor/membership, process ACK/SIGKILL/fresh Open, declared environment operator, loopback endpoint, live closure control and replay checks. No application callback policy or candidate/context Prepare/Commit stepping.
- docs/foundational/phase1i/evidence/runtime: original records, ACK/PID/argv/selected environment/binary hashes, endpoint ledgers, normalized original-operation/parent maps, independent monitor verdicts, model memberships/rejections, fixture/source manifests, compact development failures and normal/race summaries.

No runtime/compiler/A/B/protocol/model/deployment.go/paper files were edited by this stream. No commits/subagents/paid calls/merges/releases.

## Frozen profiles, paths and source identity

Before execution: fixtures.json freezes matrix payloads/modes/shapes/envelopes/refusals, weak resource limits, eight fresh samples each P/Q, two valid negatives, four pure cuts, nine applicable closure cuts, five K1E profiles and replay scope. fixtures-supplement-root-fence.json freezes the added32-step concrete scheduling witness before it ran.

fixtures-qualification-v1.json records exact selected commands, expected leaf counts and source SHA256 before race-v1. development/source-snapshot-v1.json.gz preserves source text. One later drift was research/substitution/probes_test.go, which is a model package test file and is not compiled into this root harness dependency; graph/library sources and this stream's code were unchanged. Controller final source freeze remains authoritative for final branch qualification.

Normal original evidence: evidence/runtime/normal-v1, normal-v1.jsonl.gz, normal-v1-summary.json. Race originals: race-v1, race-v1.jsonl.gz, race-v1-summary.json. Every artifact keeps exact original Inspect rows (including programs/history/intent/observation descriptors), not repaired or mutated records. The original SQLite temp directory is not retained; original public Inspect JSON records/hashes are retained.

Normal/race source bounds: MaxSteps200, MaxDispatch200, physical MaxInFlight1. Persisted resource limits match each frozen profile; no semantic bound inflation. Operational normal watchdog10m/race30m; subprocess100s/600s. No sleep-based crash placement. Same actual -race test executable launches race children; every process start records race_instrumented=true in race-v1.

## Executed gates and exact selected counts

Environment for test commands:
`env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build`.

Root selected normal command, exit0:
`ALAN_I_ARTIFACTS=<evidence/runtime/normal-v1> go test -tags integration -run '^TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence)$' -count=1 -json -timeout=10m ./internal/substitutiontest`.

Root selected race command, exit0:
`ALAN_I_ARTIFACTS=<evidence/runtime/race-v1> go test -race -tags integration -run '^TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence)$' -count=1 -json -timeout=30m ./internal/substitutiontest`.

Both selected precisely:
- TestSubstitutionRuntime:190 leaves =160 sufficient (A/B *five shapes *four payloads *four modes) +30 explicit invocation/created-run/depth refusal leaves.
- TestSubstitutionProbes:26 leaves =weak8 +P/Q16 +valid faulty2.
- TestSubstitutionRecovery:4 leaves =A/B bare entry-commit and outcome-retained.
- TestSubstitutionClosure:9 applicable leaves, specified below.
- TestSubstitutionEffects:5 leaves =known, unknown-applied-reopen, closure-before-reservation, reserved-then-closure, late-authentic-receipt.
- TestSubstitutionRootSuccessFence:2 required counterexample leaves, A/B.

All236 exact leaf names are in the normal and race summaries. Zero selected SKIP events, zero race reports. TestSubstitutionProcessChild is a subprocess entry helper (not an extra top-level experiment); it skips only if invoked directly without ALAN_I_DB. A/helper-commit is explicitly inapplicable, not passed or counted: frozen A has no helper.

Standalone module commands in examples/substitution/consumer: go test -count=1 -v ./... PASS (TestConsumerInitRunInspectReplay/a and /b, two leaves); go test -race -count=1 -timeout=10m ./... PASS; go vet ./... exit0; go mod verify exit0/all modules verified. Focused root go vet -tags integration ./internal/substitutiontest ./examples/substitution/... exit0. Own files gofmt applied. Observed tool results retained in consumer-gates.json. Root existing regressions/build/full manifests remain controller-owned and were not duplicated.

Additional record-only command after runs, exit0 each:
`python3 docs/foundational/phase1i/evidence/runtime/check-records.py docs/foundational/phase1i/evidence/runtime/{normal-v1,race-v1}`.
This post-run original-JSON checker verifies8 weak payload/admission obligations and9 exact closure envelopes/charges; normal-v1-record-checks.json and race-v1-record-checks.json. It is not the frozen K1 independent graph oracle and does not replace it.

## Actual observations

Sufficient A/B matrix: answer completes; fail produces failed/member_failed without response; absent local Success produces failed/missing_reply child envelope; wrapped non-success maps to wrapper failed/member_failed while original child remains separately retained; pending remains open at its actual unmatched receive PC after original request consumption. Parallel pending includes the peer's completed S branch and remains partial, never full completion. CheckTrace uses original coherent operation/parent links; explicit Publications and RawStatuses retain absence/status distinction. Actual Kind=invocation outcome consumptions are included from selected history observations. CheckCandidateContext uses independent transcribed graphs, not replay as oracle.

Weak probes: both sufficient candidates return original id1/result0. At budget2 weak-a admits candidate+S and succeeds; weak-b admits candidate+helper, then S is rejected invocation_limit. At3 both succeed with different charges (weak-a2, weak-b3). I3/R2 separately exposes weak-b S created_run_limit. At1 weak-b helper is refused, candidate fails; no invented successful weak result. These are separate weak contracts, not mislabeled A/B or a counterexample to an equal-charge premise.

P/Q: eight fresh namespaces per artifact. All16 concrete samples had peer S win; candidate helper rejected, root failed, peer succeeded. No sampled distinguishing P/Q outcome. Raw namespace, sorted-name-dependent identities, facts and debit order remain in originals. Model trace sets/local relation are separate model evidence; equal totals do not establish readiness preservation or contextual equivalence.

Valid negatives: wrong-id executes actual normal successful Run/Replay with schema-valid wrong application id; unchanged CheckTrace rejects application_id. Extra-helper executes actual successful Run/Replay, accepts three invocations/created runs under I=R3 and is rejected extra_helper. No malformed-schema/guard-denial fake negative; no unknown-to-success detection claimed.

Pure recovery: each real child ACKs selected immutable retained boundary, parent captures before, sends SIGKILL, verifies WaitStatus SIGKILL, launches different fresh PID that only Open+Run, and observes successful original continuation. Original intent/observation/charge rows and operation/member links conserved; root epoch increases. Replay child has absent adapter registry/key, invalid proxies and actual macOS sandbox-exec deny network*. Before/after complete Inspect equality checked.

K1E: loopback endpoint is hosted in controller process, executor in separate child; only declared HTTP protocol crosses boundary. Endpoint independently records original operation/key/application before response. Known success records one genuine validated receipt. Unknown-applied-reopen ACK/kill loses original receipt, retains one original allowed attempt/reservation/key and spent effect1, fresh Open+Run produces no recontact, no fabricated receipt, open continuation and original unknown handle. Known/unknown replay has absent registry/key with OS deny-network and unchanged full ledger+endpoint count.

Live profiles remain separate from killed-worker unknown recovery. Reserved-then-closure pauses at real allowed reservation before physical contact, closes root through pre-granted input while main Run services operator, resumes original worker and observes one allowed physical request after closure. Late-authentic-receipt first obtains actual endpoint application, closes root, then releases original HTTP response to same live worker. Both consume the original valid result and attempt second request: second source authorization Allowed=true, final reservation Allowed=false/scope_closed, no second endpoint contact, effects1. Results contain one genuine validated original receipt plus a separate source_denied result for denied second request; they are not two external receipts. Root cancellation remains immutable.

## Nine exact closure cases

All root outcomes remain cancelled; extra static operator fixed cost1 outside root; charges below are invocations=created runs, effects0.

A entry-commit: debit0; entry rejected source_denied:source_closed, no response.
A entry-admitted: debit1; already admitted candidate drains succeeded/member_succeeded with present response.
A reply-commit: debit1; candidate succeeded/member_succeeded, present.
A published: debit1; existing succeeded/member_succeeded publication immutable, present.
B entry-commit: debit0; entry rejected source_denied:source_closed, no response.
B entry-admitted: debit1; helper rejected scope_closed; candidate failed/member_failed, absent.
B helper-commit: debit1; same helper scope_closed refusal and absent failed candidate.
B reply-commit: debit2; accepted helper and candidate both succeeded/member_succeeded, present.
B published: debit2; both original success publications remain immutable.

The operator exception was explicitly confirmed by controller: no host stepping of candidate/context, but declared environment operator may Commit its own receive/emit and DecideLifecycle to inject cancellation at retained cut, using normal grants/authorization/transactions. Program, grants and setup trace are in original inspections. Live effect operator executes entirely through child Store.Run. Source-authorization denial and final admission/reservation denial remain distinct.

## Preserved failures, corrections and limitations

Initial tests-first scaffold fails at missing runMatrix/runProbes/runRecovery/runClosure/runEffects symbols; this is compile RED, not a claim of full behavioral TDD. Consumer initially attempted to JSON-encode Template.ID method; actual test failure retained, corrected to ID(). One mistaken root go test ./... instead of nested module was interrupted after~14s; partial pass and interrupt failures in examples/adaptive/investigation retained in development/alan-phase1i-consumer-attempt1.log. No root suite pass claimed, no broad integration H suite executed.

Matrix attempt1 retained raw/normalized failures and original decoder source: independent model had helper-return scratch leaking into candidate absent/fail/pending envelope and represented open root as empty. Model author corrected graph scratch separately and Observe explicit open (no decoder payload coercion). Decoder's incorrect wrapper-entry-refusal fallback was removed: nonexistent candidate remains empty, wrapper original publication remains explicit. Initial selected observation Kind was corrected from invocation_outcome to actual retained invocation so consumption monitor is exercised.

First live process observer dropped lifecycle ACK through its cut wrapper; actual timeout retained. Forwarding fixed observer only. Initial live count assertion expected one adapter_result but actual normal denied-second protocol retains two (validated original + source_denied synthetic result). Original failed live records retained, assertion refined to exact original IDs/dispositions/source authorization/final reservation; no external effect/receipt count weakened. Development process source hash capture occurred after ACK-forwarding correction, so named post-attempt1-live-fix-source-hashes, with source-capture-note.json; actual original child binary hashes/ACKs retained. No claim of original first-live source text freeze.

Pending obligations: independent whole-branch/harness review, controller final source freeze and final branch qualification/PR. Closure safety CheckTrace is checked, but exact closure end-state model membership is not claimed for this subset; original terminal tuples retained and checked against declared operator/runtime boundaries. K1E is a separate explicitly bounded effect companion, not an extension of the pure K1 graph proof. Original temp SQLite files were not archived, only exact public Inspect records. No distributed guarantee, fairness proof, scheduler confluence, generic receipt reconciliation API, or recovery equivalence inferred.
