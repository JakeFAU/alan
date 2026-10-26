# Task 5 process qualification implementer report

The actual `alan-evidence` application answers **“WAL retains a durable log.”** through the real OpenAI Responses serializer/parser, an ordinary separately committed lookup, and ordinary synthesis/citation validation. Both recorded-boundary crash cases recover that answer after deleting the selected source file. The retained citation is `selected-corpus`, SHA-256 `bcdf04a60c21be85488f48d87ad918f0d06423b517de5660ea05db7b104c17be`, byte span `[0,26)`. This is controlled protocol/application evidence, not live inference. **Live: NOT RUN**; no live credentials/model were configured or sought.

Implementation complete and frozen for scoped independent review. No commits, subagents, dependency changes, old crash-oracle changes, parent qualification-tool edits, or live requests by this implementer. Parent owns broad final suites, research regression comparison, archive/offline reproduction, documentation integration and publication.

## Production and test architecture

The only production change is six lines in `cmd/alan-evidence/main.go`: attach the existing observation-only `boundary.FromEnvironment` context for `run` and `recover`, matching `cmd/alan`. No scheduling, role dispatch, callback policy, effect semantics, authoring API, or replay behavior changed.

New `internal/evidencetest` is an integration-tagged test package, separate from `internal/durabletest`:

- `process_test.go`: actual CLI build/launch, inherited-pipe acknowledgments, exact process/barrier/durable-authorization matching, SIGKILL/SIGTERM, inspections, restart and offline replay.
- `provider_test.go`: a separate test-child process serving real HTTP. It knows no runtime store path. Accepted request identity, exact bounded nonsecret wire bytes/digest and selected response are independently appended to JSONL and fsynced before acknowledgment. Authorization headers are never retained. Literal synthetic credentials are fixture-only. `responding` means the response was selected and durably recorded before the HTTP write; runtime candidate evidence separately proves receipt.
- `progress_test.go`: a separate test-only generic owner child for static mixed-transport programs, using real adapters and ordinary `Store.Run`. This host is for supplemental capacity/late-evidence tests; application crash/recovery tests use the actual CLI. Its static fixture transport program uses existing opcodes and grants. The provider process also owns an independent existing SQLite fixture resource for its `/fixture` endpoint.
- `control_test.go`: unchanged read-only replay monitor versus an actual faulty replay-dispatch wrapper; exact acknowledgment substitutions and duplicate acknowledgment rejection.
- `race_on_test.go`, `race_off_test.go`: select `-race` for **both actual child binaries**, including the provider/test-owner host.

Provider JSONL is a serialized fsync'd process-test contact log, not a new runtime store or claimed power-loss engine. It does not read or derive its contacts from runtime state. The existing fixture's SQLite implementation remains unchanged. Controller reads independent records and runtime snapshots after/between acknowledged boundaries. Ordering uses explicit pipes and committed/provider acknowledgments; timeouts only fail stalls. No sleeps establish causality.

## Executed process cases

### C: model result durable, before ordinary consumption/tool authorization

`TestRecordedModelAndLookupCrashRecovery/C-model` pauses the actual CLI at `adapter.result-recorded/pre-settlement` for `assistant`. It checks an exact durable allowed authorization, local intent/attempt/source generation+epoch, result existence, absent settlement, exactly one model logical request, and no lookup intent. It sends SIGKILL only after acknowledgment and checks the matching signal exit. After source deletion, another actual CLI child resumes. An acknowledged lookup `adapter.finished` proves the real lookup Execute returned. The original model attempt remains byte-equivalent; ordinary consumption, pending ID, store namespace, final answer and two unique real model contacts are checked.

### D: lookup result durable, before synthesis

The D test first captures `adapter.finished/transport-ended` for `assistant/retrieval`, independently observing actual lookup execution. It then pauses at `adapter.result-recorded/pre-settlement` for the same subject: one model request, one lookup request, no synthesis request, result present and settlement absent. After SIGKILL and deletion of the original corpus, recovery arms a barrier for authorization of the exact old lookup intent. Recovery exits normally and the acknowledgment pipe is EOF: **zero old lookup dispatch re-entry**, without inferring that fact solely from final row count. The original lookup attempt and pending ID survive; the final result uses the retained span and there are exactly two unique model contacts.

The authorization/finished boundaries surround the runtime's actual adapter Execute call. They are transport observations, not counters synthesized from persisted intent counts. Final results and exact contact/request correlation also guard against duplicated/replaced model operations.

### E: accepted/withheld provider request with no runtime result

`TestAcceptedUnknownCrashDoesNotRepeat` releases the runtime's authorized boundary, then waits for the independent provider's fsync'd accepted acknowledgment. It correlates exact provider PID, original `X-Client-Request-Id`, wire bytes/hash, namespaced external key, and durable allowed reservation before SIGKILL. The provider withholds its response. Reopen has one model request/attempt, zero results/lookups, the same Unknown intent, same budget/pending projections and identical original attempt. It makes no repeat contact. No reconciliation or deliberate repeat policy is claimed.

### Offline replay

Each C/D/E case runs the actual CLI `replay` after killing the provider process, deleting the source file and removing the key from the child environment. Replay takes no adapter registry. It preserves all inspection rows (including every lifecycle/external/adapter ledger), every application projection/public result, namespace and provider contact record.

Machine evidence uses `adapter_invocations_inferred: 0` with an explicit basis: registry-free replay API/code path plus unchanged full store and independent contact monitor. This is **not mislabeled as a dynamic lookup invocation counter**. D's independently observed lookup execution and absence of old dispatch re-entry are separate evidence.

### Held model, independent peer/caller and shared capacity

`TestHeldModelOrdinaryProgressAndSharedCapacity/{1,2}` runs a real held model HTTP request with an ordinary peer/caller pair and independent lookup + old fixture transport. The controller first pauses the peer before commit, waits for exact durable provider acceptance, then allows peer commit and caller consumption. It records original process/operation/barrier identities and verifies the held model's exact authorization remains active.

At capacity 1, no competing lookup reservation or fixture attempt exists while the caller has already consumed the peer result. At capacity 2, lookup and fixture settle before model release and while its exact authorization is active. Both fast completions are durable before release. All five ordinary subjects later succeed. Thus model/tool/fixture share capacity and local ordinary work advances during saturation.

The initial controller incorrectly assumed the fast lookup would not settle before the caller's receive acknowledgment. A real run disproved that assumption. The corrected controller accepts independent fast-completion ordering, while retaining the required provider-accepted → peer-committed → caller-consumed → model-release order. This was a **test-harness ordering defect**, not a runtime defect or a newly promised lookup/caller ordering guarantee. Original log and complete then-current process/progress/provider sources were preserved before correction in `red/fast-order*`.

### Cancellation, closure and late original evidence

`TestCancellationRetainsRealReceivedModelCandidate` pauses actual CLI at `adapter.response/pre-record`, checks no durable result yet, and sends external-controller SIGTERM. The CLI exits 1 with cancellation evidence. Drain records and settles the received candidate with its original intent/authorization, leaving no Unknown for that candidate. Recovery completes the answer without repeating the first model operation. This is an explicit external controller cancellation injection, **not demonstrated supervisory policy**.

`TestRealModelLateEvidenceAfterSourceClosure` holds real HTTP while the model source reaches ordinary terminal success. Only after the committed closure acknowledgment does the provider respond. The late result is recorded under its original epoch/sequence/intent/attempt, shown in the known-but-unsettled snapshot, then settled. The closed source stays succeeded; replay validates the result. Prior Task3 unit coverage and old Phase1D/1D.1 cancellation/epoch/oracle cases remain unchanged for parent full verification.

## Nonvacuous negative control and identity controls

`TestFaultyReplayDispatchIsDetected` first runs correct replay with the provider still available through `monitorReplay`, checking the complete runtime snapshot and independent provider log. Then a test-only faulty wrapper calls the same read-only replay and directly executes the original retained Prepared request through the **real** OpenAI adapter. It deliberately bypasses runtime source-closure guards, so the fault cannot be vacuously rejected before HTTP. The provider records one additional real accepted contact with the original request/client identity and the adapter returns validated. The exact same monitor rejects the changed independent log. Original attempt, candidate, before/after snapshots and complete contacts are emitted.

`TestAcknowledgmentRejectsIdentitySubstitution` rejects changed PID, store, epoch, generation, sequence, intent, attempt, phase, subject, token, scenario, and duplicate acknowledgment through the same matcher used by actual child tests. Existing earlier false-order/identity oracles are untouched.

## Evidence, configuration and source preservation

All evidence is below `.superpowers/sdd/00-plan/evidence/task-5/`. Single-line `EVIDENCE {...}` records contain exact build commands and binary hashes, real child PIDs/exits/signals, configured and acknowledged failpoints, original authorization/intent IDs, all retained program/schema/template/corpus bytes and config through snapshots, before/stopped/after/replay projections and every ledger, exact independent contacts, controlled requested/returned model and optional usage. Fixture model is `controlled-requested-model`; returned model is deliberately `controlled-returned-model`. The successful run has two model logical requests and one lookup, three actual allowed attempt reservations. Fixture usage is supplied as 13 input/7 output/20 total per model response and retained as controlled reported values, not live provider billing.

The first executed missing-seam failure is `red/cli-boundary-executed.log`: the actual CLI completed instead of acknowledging the requested boundary. Its exact before-production source is `red/main-before.go.txt`. `red/cli-boundary.log` is a preceding test compilation mistake (`Intent.Epoch` does not exist), not an executed behavioral failure. `red/fast-order.log` and `red/fast-order-before/` retain the independent-completion harness defect and source inputs. Intermediate passing logs are retained, not overwritten to erase earlier evidence.

`source/` freezes every file changed/added by this implementer; `source-manifest.json` contains SHA-256/size per scoped source. Parent will make the complete phase source/dependency archive, including all new packages/embedded corpus/testdata/qualification tools. Runtime fixture files use test temp directories and are recreated by offline tests; complete retained runtime artifacts and independent contact contents are embedded in the JSON evidence. No secret headers or real credential values are retained.

## Scoped verification

Every command uses `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`.

- `go test -count=1 -tags integration -v ./internal/evidencetest` — PASS (`focused-normal.log`).
- `go test -count=1 -race -tags integration -v ./internal/evidencetest` — PASS (`focused-race.log`); actual CLI and provider/owner child builds explicitly include `-race` in the emitted build records.
- `go test -count=1 -tags integration ./cmd/alan-evidence ./internal/boundary` — PASS (`cli-boundary-regression.log`).
- `go vet -tags integration ./internal/evidencetest ./cmd/alan-evidence` — PASS (`vet.log`, empty).
- `gofmt -l cmd/alan-evidence/main.go internal/evidencetest` — no output (`format.log`).
- `git diff --check` — PASS (`diff-check.log`, empty).

No new dependency or runtime migration. Parent runs tagged/default whole-tree suites, full race/vet/module/research checks, independent review and final offline archive reproduction after this source freeze.

## Independent-review correction T5-R1 — fix round 1 of 5

The independent reviewer found a medium-severity test-oracle gap: real C/D/E filters intentionally wildcard the sequence, and the durable-correlation helper did not compare the acknowledged sequence against the matching retained operation. The fully specified synthetic sequence control therefore did not cover the actual crash path. The reviewer changed only the original actual C-model acknowledgment's Sequence 15 to 16; both old checks accepted it. Independently audited original normal/race runtime traces had correct identities. **This is a qualification-oracle defect, not a runtime flaw.** The earlier statement that sequence substitution was rejected must be read with this correction.

The scoped fix changes only `internal/evidencetest/process_test.go`, `control_test.go` and adds `testdata/actual-c-model-boundary.json`. `validateDurableBoundary` now returns an error so actual crash cases and the control call exactly the same implementation. After matching durable authorization/subject/generation/epoch, it binds the event to the matching adapter attempt's original Intent and checks its Sequence. It independently finds the committed intent descriptor, checks that Sequence, and verifies full equality with the attempt's Intent. The expected sequence is read from retained source evidence; no fixture sequence constant is pinned in the matcher. Missing or conflicting retained operation evidence is rejected.

`TestWildcardAcknowledgmentRejectsRetainedSequenceSubstitution` uses the exact real wildcard filter, original acknowledgment, and Sequence-only substituted acknowledgment preserved by the reviewer. The new testdata is a byte-preserving JSON-row subset of the reviewer's actual stopped snapshot (`attempts`, `adapter_attempts`, `intents`), with the original full evidence file's SHA-256 recorded as provenance. Both original and substituted acknowledgments pass the wildcard matching stage; the original passes durable correlation and the substitution is rejected by that shared durable helper. Earlier fully specified identity/duplicate controls remain unchanged.

Before the correction, the new test executed and failed with `actual wildcard path accepted sequence substitution against retained operation`; `fixround1/red.log` retains it. `fixround1/before-source/` preserves both original changed Go files; `red-source/` preserves the complete red regression/refactor source and actual-input fixture before the sequence-binding fix. The reviewer's complete pre-fix evidence/source and parent's complete pre-review source remain untouched. After the fix, the original portable reviewer overlay was rerun successfully: it still reproduces the **old** oracle flaw using its preserved source (`original-reproducer.log`). Its PASS is explicitly not a claim that the flaw remains in current source.

Frozen corrected scope is `evidence/task-5/fixround1/source/` with `source-manifest.json`. `changed-file-manifest.json` identifies the two changed Go files and one new testdata file with before/after hashes. `fix.diff` is the exact diff from the original seven-file scoped source, including the added fixture. The initial `source/` and `source-manifest.json` remain unchanged as historical evidence; the fixround1 manifest supersedes them for current review. Parent must include the new testdata file in the complete phase archive.

Verification commands and exact exit codes are retained in `fixround1/commands.json`. All Go commands use the established offline environment. Results:

- Red exact wildcard regression — expected FAIL, exit 1.
- Focused normal `TestWildcardAcknowledgment|TestAcknowledgment|TestRecorded|TestAccepted|TestFaulty` — PASS, exit 0. This covers the new actual-input control, unchanged synthetic controls, actual C/D/E crashes/recovery/replay and real replay-dispatch negative control.
- Same focused suite with `-race` — PASS, exit 0; emitted build records again show `-race` for actual CLI and provider child binaries.
- Scoped integration vet — PASS; gofmt and `git diff --check` produce no output.
- Original preserved reviewer reproducer — PASS in its explicitly defect-reproducing sense.

No production/runtime/source-semantics, dependency, parent tool/document, or old crash-oracle changes. No commits or delegation. Parent retains ownership of broad final verification and independent re-review. Live remains NOT RUN.
