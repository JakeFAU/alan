# Task 5 report: ordinary external demo and independent conformance

Status: implementation and qualification source frozen for scoped independent review. Parent owns public documentation, historical reproduction, final complete source/dependency identity, durable evidence archive, whole-branch review and publishing. No commits, subagents, dependency changes, interpreter/authority/transport/schema changes, controller repairs or saved historical evidence edits were made in this slice.

## Changed files and preservation

- `cmd/alan/main.go`: external demo flags and selection only.
- New `internal/executor/external_example.go`: ordinary `ExternalProvision(endpoint, profile)` data builder.
- New `cmd/alan/external_demo_test.go`: actual protocol/CLI tests.
- New `research/semantics/runtime_conformance_test.go`: six independently driven preserved-model/runtime qualification tests.

Comparison against `.superpowers/sdd/00-plan-phase1d/task-5-base` finds only `cmd/alan/main.go` changed among its existing 50 source/dependency files. All thirteen original research Go files match byte-for-byte; exact byte counts and SHA-256 values are in `/tmp/alan-task5-preserved-files.json`. `go.mod`, `go.sum`, runtime protocol and prior tests match the base. The new bridge intentionally changes the historical suites' dynamically generated *current* source manifests. No saved historical manifest was rewritten.

## External demo contract and reproducible interface

The ordinary fixture has static subjects `client` and `peer`, generation 1, source revision 1. Client emits one external counter mutation `delta=5`, profile `idempotent` or `non-idempotent`, obligation `counter-receipt`, resource generation 1 and exactly the supplied literal loopback HTTP URL. Its next instruction receives the authenticated receipt for its exact generated pending operation and binds `total`; then it ends successfully. Peer independently ends successfully. The only external permission is `(external, client, supplied URL, apply, revision 1)`. Validation reuses the existing external request protocol validation. No roles, new opcodes or lifecycle behavior were added.

`--profile` defaults to `idempotent`; an explicit unsupported profile, missing endpoint, or non-loopback endpoint is rejected before store creation. This external example is separate from the nested supervision example. It does not claim that the nested demo repeats business work.

With binaries built using `go build -o /tmp/alan ./cmd/alan` and `go build -o /tmp/alan-resource ./cmd/alan-resource`, run the endpoint in a separate terminal:

```sh
/tmp/alan-resource --db /tmp/demo-resource.sqlite --token demo-resource --listen 127.0.0.1:8791
```

Then, using fresh store paths:

```sh
/tmp/alan init --db /tmp/demo-external.sqlite --demo external --endpoint http://127.0.0.1:8791/apply --profile idempotent
/tmp/alan run --db /tmp/demo-external.sqlite
/tmp/alan inspect --db /tmp/demo-external.sqlite
/tmp/alan replay --db /tmp/demo-external.sqlite
```

Use `--profile non-idempotent` with another fresh executor store to select the conservative unknown-outcome behavior. Endpoint process returns its actual URL on startup; the displayed fixed port is an example that must be free. This slice's behavioral tests exercise `runCLI` directly against a real HTTP resource backed by its separate SQLite database. Task4 remains the compiled-process/SIGKILL qualification source. Reproducible applied-but-missing-receipt checks are `go test ./cmd/alan -run '^TestCLIExternalDemo$' -count=1 -v`; the test endpoint applies first and responds HTTP503, without a receipt. Both profiles remain open/unknown while peer finishes. Two runs yield one mutation/total5; idempotent makes two contacts, non-idempotent only one. Successful receipt cases finish and make one contact. Replay is checked against the complete executor inspection and independent endpoint contact count.

## Independent oracle and encoding relation

Each bridge starts with a named original `supervisionScenarios()` fixture, verifies the exact seed length, runs its original events using preserved `step` and `inspect`, and carries the returned monitor. The runtime program is hand-authored ordinary opcode data and uses only public Create/Open/Snapshot/Commit/AuthorizeIntent/LoadLifecycleRequest/DecideLifecycle/SettleLifecycle/DeliverTerminal/AdmitInput/DispatchExternal/Inspect APIs. No runtime Reduce or Replay appears in the bridge. No SQL mutation or direct storage connection is used. Read-only Inspect is the evidence surface, including exact retained BLOB descriptors.

The common view compares subject current generation and retirement; every retained inner/worker run's open/terminal status and frozen outcome status; expected child, spent, limit, obligation, pending operation, consumed outcome classification, accepted result flag and successor; all emitted lifecycle request source/target/generation/revision/operation/obligation fields; source authorization; immutable first result acceptance/rejection/reason/successor and original request; exactly one retained result per decided operation and no extra lifecycle intents; notice bound run, trusted provenance, source generation, obligation, exact operation correlation and consumed disposition.

Exact mapping:

- Model `active` maps to runtime `open`; other statuses are literal.
- `inner/g1/declared-obligation/init2` is an explicit literal in the runtime's second initialization instruction, matching the model initializer. Only this finite generation-1 supervisor encoding is claimed.
- The five ordinary initializer commits correspond to the original five model Prepare/Commit turns. Named runtime fields map to the preserved model's local slots; all numeric values match exactly.
- Model failure classification `2` maps to runtime outcome `failed` (`1` succeeded; `3` abandoned; `4` cancelled; `5` suspicion in the adapter, though the selected runtime programs cover failed/succeeded terminal inputs only).
- Exact ID maps are recorded, never arbitrary string stripping. Model `inner/g1/n10/i0` maps to runtime `["inner",1,9,0]`; raw probe `inner/g1/n6/i0` maps to `["inner",1,6,0]`. External `worker/g1/n1/i0` and `worker/g2/n1/i0` map to `["worker",1,1,0]` and `["worker",2,1,0]`. Every request descriptor retains its raw identity. The independently provisioned runtime probe uses its own new tuple key and obligation `fresh-probe`.
- Model rejection `retry policy` maps only to runtime `not_retryable`; `authority` maps only to `grant_denied`; accepted empty model reason maps only to `retried`. Original runtime reasons remain in records; unknown reasons fail the bridge.
- Runtime Emit uses the already-supported operand Offset to reserve spent and set pending atomically. No committed scratch increment is necessary with the actual Task3 API.
- Ordinary failure/budget branches have different PC/commit counts. Only named public checkpoints are compared; exact PCs and whole histories are retained as evidence rather than declared equal.
- **Model lifecycle Receive atomically consumes and clears pending. Runtime Receive consumes and binds result, while pending remains populated until a separate ordinary Set.** Every such unmatched intermediate is recorded with both projections and explicitly marked neither equivalent nor a stutter. In C, the runtime is reopened at this intermediate and its residual continuation/pending is checked before Set. Equality is only asserted at the resulting declared macro-boundary. No new atomicity claim is made.
- Physical Open is a projection/evidence-preserving boundary with a checked implementation epoch rule: every open run advances epoch by one, every terminal run remains frozen, and all other inspected rows remain identical. Model Reactivate advances only the selected run. Global epoch equality is therefore not claimed; C checks exact selected-run equality at the intended interruption boundary before the additional runtime-only reopen.

The adapters extract/read evidence; they do not interpret runtime programs or recompute lifecycle target policy. Fresh runtime program artifacts and full config/trace bytes are recorded for reproducibility.

## Fixture matrix and exact traces

| Test | Preserved fixture / seed | Trace and asserted boundary |
| --- | --- | --- |
| AcceptedLossAndSuccessor | `B_accepted_lost_ack_and_successor`, 25 events | Full seed; suffix DecideLifecycle original ID, Prepare worker/g2 C1, Commit. Compares initialization, actual worker failure, terminal admission/consumption, branches, reservation, separate source authorization, accepted result with no admission, repeated result after two physical reopen boundaries, then actual g2 Fault commit on unchanged worker artifact. 28 model events, 12 checkpoints. |
| InterruptedCommittedRequest | `C_committed_intent_interrupted`, 23 events | Full seed; exact suffix AuthorizeLifecycle, DecideLifecycle, AdmitLifecycleResult, Prepare inner/g1 selecting original result, Interrupt slot0, Reactivate inner/g1 owner/revision7. Reopen, reject old runtime authority with complete evidence unchanged; consume original result using fresh authority, record/reopen unmatched consumed-but-pending state, clear, branch/copy/jump to expect g2. 37 model events, 13 checkpoints. |
| RejectionSurvivesFailure | `B_rejected_then_failed_retransmission`, 14 events | Full seed; Prepare/Commit worker Fault; repeat original DecideLifecycle; admit/consume retained rejection and ordinary requester Fault. Reopen before failure and before retransmission. Then separately labeled runtime-only static `probe` emits a distinct operation, receives acceptance creating g2; original rejection remains identical and g2 executes. 22 model events, 12 checkpoints. This original fixture has no supervisory policy contract; it is not labeled failure-driven retry. |
| ObservationDoesNotGrantAuthority | `I_observation_without_lifecycle_authority`, 23 events | Full seed; source authorization succeeds; target decision rejects absent lifecycle grant although observation grant exists; result admission/consumption, branch, ordinary Fault. 32 model events, 11 checkpoints. |
| ExternalGenerationsAndLateReceipt | `J_unknown_effect_recovery_late_settlement`, 38 events | Full seed; suffix Prepare/Commit worker/g2 Emit, AuthorizeAttempt/ApplyExternal g2, RecordReceipt old g1. 43 model events, 30 checkpoints. Exact runtime differences below. |
| ClosureBeforeExternalAuthorization | Same unchanged J fixture, prefix events1–12 only | Worker Prepare/Commit Fault, then rejected AuthorizeAttempt (`closed`). Rejected event, unchanged model state and monitor check are recorded explicitly. Runtime closes then stores denied authorization/source-denied settlement with no HTTP contact or fabricated receipt. 15 model events, 3 checkpoints. |

All six CONFORMANCE records carry `test` names so the two J-derived fixtures remain distinguishable. Tests do not parallelize. All model event rejections are fatal except the one explicitly recorded expected closure rejection. Every model safety monitor must pass.

## External generation and receipt relation

The J runtime worker differs deliberately from the model's Emit→Fault: it emits the same explicit `business:order-7` counter mutation with delta1, then waits for a local control input, branches `fail` to ordinary End failed or `wait` to an exact receipt receive. The g1 `fail` input creates failure through ordinary committed transitions; g2 `wait` positions a real current-generation receive for correlation testing. Full control inputs/admissions/commits are recorded. They are runtime contract setup, not translated model events. No whole-trace equivalence is asserted.

The independent in-process HTTP endpoint uses real `fixture.Resource.Apply` in its own SQLite database. Initially it applies each request and sends HTTP503 so neither execution retains a receipt. Model AuthorizeAttempt+ApplyExternal is compared with one public runtime DispatchExternal call; the public API combines those boundaries, so this bridge does not compare an intermediate external-attempt-only point. The source never reads endpoint storage or receives an inferred success/failure input based on its contents. All endpoint reads are test evidence only.

Checkpoints establish old Emit without attempt, unknown application, reopen preserving uncertainty, frozen g1 failure, accepted Retry, lifecycle consumption and expected generation2, actual g2 Emit with distinct key and the same business reference/delta, two actual external mutations with old receipt still absent, then late original receipt settling only g1. Idempotent retransmission retrieves the original endpoint receipt; it does not create a new mutation. g1 stays failed and the late receipt remains unconsumed on g1. g2's full state and supervisor policy state remain identical. Attempting the old observation at g2's actual receipt wait returns ErrIneligible and changes no inspected evidence. After reopen, dispatching g2 obtains its own retained receipt, which g2 consumes and then succeeds. Final endpoint evidence: total2, two application receipts, four contacts (one original and one idempotent retrieval per key). Final new receipt consumption is separately labeled runtime-only, after the common model trace.

External common projections compare exact source generation, declared business identity/delta, immutable intent/attempt/request key, authorization presence, receipt presence/key, settlement disposition and independent world mutation count. Model late receipt on a terminal source does not create an inbox entry; runtime intentionally retains a trusted unconsumed closed-run receipt observation. This ledger representation difference is recorded and not erased. Model suppression is a bit plus rejected authorization; runtime retains denied authorization plus explicit denial settlement/observation. Neither permits transport after fresh source denial.

This shows execution identity deduplication, not business deduplication. It does not claim the nested ordinary demo has this business ambiguity. Task4 owns separate-process isolation, killed transaction/application windows and fault-oracle controls.

## Test-first evidence and verification

New observable CLI behavior was tested first. Initial scaffolding compile errors (RunID field access and contacts integer type) and sandbox loopback bind failure were corrected before claiming RED. Genuine behavioral RED `/tmp/alan-task5-cli-red.log`: all four profile/receipt cases fail `flag provided but not defined: -endpoint`. GREEN `/tmp/alan-task5-cli-green.log`: all pass after the small builder/CLI change. Invalid configuration uses the existing protocol validation rather than adding a new validator.

New bridge coverage qualifies an existing implementation, so it passed on its first behavioral runs after compilation scaffolding was fixed. No false preimplementation RED claim is made. Logs: `/tmp/alan-task5-bridge-first.log`, `bridge-core.log`, `bridge-external.log`, `bridge-green.log` (each prefixed `/tmp/alan-task5-`). No protocol defect was observed and no oracle expectation was weakened to hide one. The macro-boundary/late-inbox differences above are retained limitations.

Final commands:

```sh
GOCACHE=/tmp/alan-phase1d-go-build go test -count=1 ./...
GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor ./internal/fixture ./internal/boundary ./internal/durabletest ./cmd/alan ./cmd/alan-resource ./research/semantics
GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
GOCACHE=/tmp/alan-phase1d-go-build go test ./research/semantics -run '^TestRuntimeConformance' -count=1 -v
git diff --check
gofmt -l cmd/alan/main.go cmd/alan/external_demo_test.go internal/executor/external_example.go research/semantics/runtime_conformance_test.go
```

Full normal `/tmp/alan-task5-full.log` exits0: CLI0.271s, boundary0.131s, durable controller4.576s, executor2.813s, fixture0.208s, all semantics4.808s. Vet `/tmp/alan-task5-vet.log` has no diagnostics. Diff/format checks have no output. Final focused conformance exits0. Race `/tmp/alan-task5-race.log` exits0: executor7.419s, controller54.594s, CLI1.682s, all historical+conformance semantics81.827s; unchanged fixture/boundary packages cached PASS. No race diagnostics. Task4’s race tag instruments the actual controller child binaries as in its already-reviewed implementation. Loopback commands used the user-authorized targeted sandbox escalation after normal sandbox bind denial; no automatic-review rejection occurred.

## Raw evidence, limits and handoff

- `/tmp/alan-task5-conformance.log`: final verbose raw Go test output with six `CONFORMANCE` JSON records and full Test context.
- `/tmp/alan-task5-conformance.jsonl`: extracted records in original order; 1,874,092 bytes at capture time. Each record includes test/fixture, exact model config, full seed and chosen trace, complete runtime provision/artifact bytes, literal operation/observation maps, ordered runtime actions, all public/mismatched checkpoints, complete read-only snapshots, external endpoint evidence and limitations.
- `/tmp/alan-task5-preserved-files.json`: thirteen original research file hashes, byte counts and equality checks against base.
- `/tmp/alan-task5-race.log`: race result log.

Parent must apply the final Phase1D complete runtime+test+dependency source identity and preserve Test/order context when archiving records. The old model-only manifest is not represented as the complete runtime identity. Transient absolute paths/ports/PIDs/resource process identifiers are genuine observed inputs, not canonical invented values; the full provisions retain their exact endpoint values. The public instructions above are reproducible definitions; the evidence retains actual execution bytes.

Unsupported: whole-machine bisimulation, a general compiler, arbitrary supervisor generations/obligation derivation, suspicion/cancelled/abandoned policy mappings, dynamic creation/ownership, arbitrary concurrent programs, business dedup/compensation, byte-equal PCs/histories/epoch counts, and atomic runtime lifecycle Receive+pending clear. No new process/crash guarantee beyond prior Task4 is asserted. Frozen source is ready for scoped independent review; no known failing test or unresolved runtime defect remains at this handoff.
