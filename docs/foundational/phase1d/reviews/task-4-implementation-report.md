# Task 4 report: independent resource, acknowledged crashes, and fault controls

Status: implementation is frozen for source-and-harness review. Final verification results are recorded below. No commits, subagents, dependency changes, historical research changes, or Task 5 bridge work. Parent owns public documentation and final packaging. The Go/TDD and verification guidance was applied; standard library and the already pinned mattn SQLite driver remain the only dependencies.

## Scope and files

New production code: `internal/fixture/resource.go`, `cmd/alan-resource/main.go`, `internal/executor/external.go`, `internal/boundary/boundary.go`.

Supporting changes: executor `store.go`, `messages.go`, `lifecycle.go`, `lifecycle_replay.go`, `runner.go`, `inspect.go`, and CLI `cmd/alan/main.go`. No interpreter/Reduce change; ordinary programs continue to contain no supervisory instruction or resource implementation logic.

Tests: `internal/fixture/resource_test.go`, `internal/executor/external_test.go`, `internal/executor/fault_process_test.go`, `internal/boundary/boundary_test.go`; `internal/durabletest/{crash,resource,oracle,faults,barrier_validation,race_on,race_off}_test.go`. The previous unsupported-effect runner test now uses Kind `future-operation`, since `external` became a supported protocol. Its original fairness/limit/unresolved assertions are unchanged. The initial full-suite failure caused by that old placeholder is retained in `/tmp/alan-task4-full-initial.log`.

## Independent resource protocol and store

`alan-resource --db PATH --token PROCESS_START_TOKEN --listen 127.0.0.1:0` owns a separate SQLite database, OS lock, pinned connection and loopback HTTP listener. Startup emits actual URL, token, store and PID as JSON. Only literal loopback listen addresses are accepted. The executor receives an ordinary endpoint URL; it never reads this database. The test controller may inspect it independently.

Public HTTP protocol: POST `/apply`, strict one-object JSON, bounded request body. Request and receipt types:

```go
type Request struct {
    Key, Profile string
    Subject, Authorization string
    Generation, Epoch, Sequence int64
    Delta int64
}
type Receipt struct {
    Key, Profile string
    Delta, Mutation, Total int64
    Process string
    PID int
}
```

The fixture mutates an integer by Delta. The operation key is the exact original durable intent ID. Profile is `idempotent` or `non-idempotent`. Subject/generation/epoch/sequence/authorization are retained audit identity, not a resource-side policy oracle or fencing claim. The endpoint pins the **whole original request descriptor**, including those fields, under the key. Idempotent identical requests return the original retained receipt, including its original endpoint process identity, without a new mutation. Changed parameters/descriptor return HTTP409. Non-idempotent identical requests apply again; this profile does not provide a query/deduplication guarantee. Integer overflow fails explicitly before mutation.

One resource transaction records key/descriptor if new, protocol contact, and application/receipt if applied. A duplicate idempotent request records another protocol contact and returns the original application receipt. The resource application barrier is reached only after that COMMIT, and can withhold the HTTP response. There is no transaction across the resource and executor stores.

Resource schema (separate database):

- `metadata(version INTEGER NOT NULL)`, version1.
- `keys(key TEXT PRIMARY KEY, profile TEXT NOT NULL, delta INTEGER NOT NULL, descriptor BLOB NOT NULL)`.
- `applications(number INTEGER PRIMARY KEY, receipt BLOB NOT NULL)`.
- `contacts(number INTEGER PRIMARY KEY, request BLOB NOT NULL)`.

`fixture.Open(ctx,path,process)` creates a fresh fixture store or reopens an existing version1 store. It verifies WAL, synchronous FULL, foreign_keys1, busy_timeout5000, query_only0, and an engine at least3.51.3 (the WAL-reset fix). Paths canonicalize the parent directory; file symlinks and hardlinks are rejected. `fixture.Inspect(ctx,path)` is mode=ro/query_only1, verifies the same effective settings and engine, metadata version, and quick_check, and returns settings/engine-source ID, complete requests/applications, total and contact count. Missing/unsupported/non-WAL inspection fails explicitly. No migrations or coherent-tamper detection are claimed.

## Generic external intent execution

New public API:

```go
func (*Store) DispatchExternal(context.Context, string) (bool, error)
type ExternalAttempt struct {
    Intent Intent
    AuthorizationID string
    Request fixture.Request
    Allowed bool
    Reason string
}
type ExternalOutcome struct {
    Attempt ExternalAttempt
    Disposition string
    Receipt *fixture.Receipt
}
```

The boolean reports settled versus still physically unknown. Internal helpers prepare retained attempts, record verified receipts, and settle authenticated evidence. Only protocol request/receipt types are used by executor code; it does not call resource Open/Apply/Inspect.

An ordinary Emit with Kind `external` declares a literal loopback HTTP URL Target, Generation1 (resource identity for this fixture), nonempty obligation, and payload `profile` string plus `delta` integer. The usual generated-output bound still applies before commit. Generation1 here identifies the explicitly selected resource protocol; it does not restrict subject generations or retry counts. Static permission `(external, source subject, target URL, apply, committed source revision)` is required.

Dispatch first uses retained source authorization, then commits an immutable ExternalAttempt containing exact intent/request and grant disposition, **before HTTP transport**. Source denial and missing external grant settle as distinct `source_denied` / `grant_denied` outcomes without contact. Previously allowed source authority survives source closure; new authority after closure is denied. A late outcome is retained for the original execution and does not resume a closed run.

Allowed idempotent recovery reuses the original key, request parameters, authorization identity and request epoch. Each Run invocation selects at most one HTTP attempt per external intent. Allowed non-idempotent recovery never sends again once an attempt exists, even if a crash occurred between durable authorization and the first send. That conservative gap is physical uncertainty, not evidence of no application.

Each HTTP request has a two-second operational timeout, context propagation, no redirects, bounded strict response decode, and exact receipt key/profile/delta matching with positive mutation/process identity. Timeout, connection error, unsuccessful HTTP response or undecodable response leaves the operation unknown; no application failure observation is synthesized. Wrong decoded receipt identity is an explicit error. Ordinary ready peers still run after an unknown result. Wall-clock values affect only operational timeout/controller deadlines, never deterministic reduction.

A verified receipt is committed separately from receipt admission/settlement. The latter atomically inserts a trusted `receipt` observation and settlement for the original source run. Observation Provenance is `resource-receipt`; source is the target URL/resource generation1; operation is the original intent ID; obligation is the original obligation. Payload contains accepted/disposition/total/mutation. Admission acceptance means evidence was settled, not that an application succeeded. Late evidence can be admitted to a closed requester while remaining unconsumed.

RunReport adds `UnknownExternal []string`, sorted original intent IDs with retained allowed attempts and no settled outcome. Existing unresolved intent descriptors and pending delivery keys also remain visible. Quiescence with unknown work is not success. No business retry key, cross-generation deduplication, compensation, owner transfer, or stronger exactly-once effect claim is added.

## Executor schema and replay refinements

Unreleased SchemaVersion1 remains; there is no upgrade path for older slice databases.

- `external_attempts(intent_id TEXT PRIMARY KEY REFERENCES intents(id), descriptor BLOB NOT NULL)` retains full ExternalAttempt.
- `external_receipts(intent_id TEXT PRIMARY KEY REFERENCES external_attempts(intent_id), descriptor BLOB NOT NULL)` retains the original Receipt.
- Trusted admission domain CHECK adds `receipt`; canonical domain-qualified keys remain unchanged. Automatic raw transport is `["receipt-delivery", intentID]` in that domain. Raw ingress strings cannot impersonate this domain.
- Existing settlements add typed ID `["external-outcome", intentID]` and full ExternalOutcome descriptor, linked to observation `["observation","receipt",intentID]`.
- Inspect includes both new tables without hiding descriptors or original authorization/receipt linkage.

Replay explicitly validates external request derivation, static grant/source authorization binding, receipt identity and canonical bytes, outcome type, settlement/admission/observation linkage. The prior lifecycle settlement validator now dispatches by the retained intent kind; it does not skip validation of unfamiliar settlement data. No receipt is permitted for denied external authority. Replay uses no HTTP, live Open, dispatch or lifecycle execution path. Controller tests compare complete executor inspection (including epochs) and independent endpoint snapshots before/after replay.

## Real boundary instrumentation

`boundary.WithObserver`, `Reached`, and `FromEnvironment` provide observation only. The optional `ALAN_BARRIER` JSON config supplies scenario/process-start token, exact store and selected event filters. The normal CLI calls FromEnvironment for `run`; the resource process does likewise. It uses inherited fd3 for one JSON acknowledgment and fd4 for one-byte `c` release. EOF/invalid release fails; absence of configuration does nothing. This is a pause mechanism, not an unsafe policy/dispatch flag.

Event fields: Scenario, Token, Store, Boundary, Phase, Subject, PID, Generation, Epoch, Sequence, Intent, Attempt. Specified generation/epoch/sequence/string filters match exactly; zero numeric/empty string fields are documented wildcards in the runtime filter. **The acceptance controller declares and compares exact expected identities**, including intentional empty terminal intent/attempt fields. It does not treat positive generation/epoch as sufficient.

Instrumented boundaries:

- `execution.staged`: real Commit transaction after history, selected consumption, intents, projection and terminal obligation writes, before COMMIT. Its prospective sequence/intent are explicitly expected; visible retained sequence must still be previous.
- `execution.committed`: only after successful COMMIT, phase post-commit or terminal.
- `intent.authorized`: after authorization transaction COMMIT, including identical retained authorization. Event epoch/sequence describe current execution; retained authorization itself remains unchanged.
- `lifecycle.decided`: after target/result transaction COMMIT, phase accepted/rejected, including retained identical decisions during recovery.
- `external.authorized`: after the external attempt is committed, before transport; exact source/intent/authorization identity.
- `resource.applied`: independent endpoint COMMIT complete, before response.
- `external.response`: matching HTTP receipt received, before executor receipt recording.

Lifecycle intent/source authorization/target decision/settlement/ordinary consumption remain separate transactions. The reducer has no observer or context policy input. All deliberate faults are separate `_test.go` code, never observer callbacks or production CLI options.

## Controller evidence and eight crash windows

The controller builds actual `cmd/alan` and `cmd/alan-resource`, initializes using the real CLI, and runs the automatic generic runner. It waits for a pipe acknowledgment, compares full expected scenario/token/PID/store/boundary/phase/run/gen/epoch/sequence/operation/attempt, verifies retained store facts, then sends SIGKILL to that PID. It requires a signaled SIGKILL exit; timeout or normal exit never qualifies. Deadlines only fail stuck tests. Expected execution identity comes from prelaunch inspection plus the known prospective instruction boundary. A second recovery process must renew epoch while preserving logical execution and original operation.

| Window | Actual evidence and post-restart assertion |
|---|---|
| Uncommitted local transaction | Nested inner sequence7 and external client sequence1 pause after staged writes; SIGKILL leaves prior complete history/projection/intent set. Correct external path produces no endpoint application; recomputation later applies once. |
| Intent commit/pre-dispatch | Inner sequence7 committed Retry/reservation/pending, no target decision; restart uses same intent and spends once. |
| Authorization/unknown | Nested source authorization cut and external pre-transport attempt cut; durable authorization exists before contact, then original idempotent work resolves. |
| External application/missing receipt | Endpoint application COMMIT acknowledged independently, endpoint inspected, response released, executor pre-record barrier acknowledged, executor SIGKILL. Idempotent recovery contacts same key twice total with one mutation; non-idempotent remains unknown with one contact/application. |
| Accepted lifecycle/pre-consume | Actual nested CLI reaches accepted Retry result/successor before settlement/consumption; same inner run/pending/budget resumes and worker g2 receives and executes real instructions on unchanged artifact. |
| Rejected lifecycle/pre-consume | Valid grant and matching generation/revision reject Retry because worker is open. After SIGKILL, actual input and worker receive/End commits make worker failed. Identical operation retains rejection; ordinary requester consumes it; a new authorized ordinary operation subsequently accepts. |
| Terminal/pre-notification | Worker terminal outcome and frozen obligation committed before dispatch. Restart delivers and nested ordinary consumers finish. Existing Task2 duplicate transport/ACK/binding tests remain in the full suite. |
| Second recovery crash | Kill at accepted result, reopen and acknowledge the same retained result with renewed epoch, kill again; third actual CLI process consumes original operation with one charge and executes worker g2. |

The rejected-result setup deliberately drives only the worker's actual receive/End through existing Store APIs after CLI input, while holding the requester's delivery back, so changed eligibility occurs **before** retransmission. It does not manufacture a failed projection. Probe emission/decision, result retransmission/consumption and new accepting probe use the automatic CLI. This isolates the required authority window without role-specific runtime scheduling.

Additional compiled CLI checks cover normal and renamed nested programs, second-executor/symlink/hardlink exclusion before any epoch mutation, exact replay immutability and independent endpoint behavior. Task1 stale/corrupt/version and Task2 forged-origin/binding tests, plus historical Phase1C ordered interruption tests, remain unchanged and run in the full suite.

## Independent checks and test-only controls

`internal/durabletest/oracle_test.go` does not call Reduce, Replay or lifecycle policy helpers. It compares retained immutable rows, links current projections to committed ordinary successor history (with explicit authority-closure distinction), checks endpoint applications against committed external intents/allowed attempts, checks one idempotent mutation per key, and checks nested original operation/budget/generation/consumption/worker-artifact and actual successor work. The same checkRecords function is used for correct and faulty runs.

1. **Lost rejection:** test-only SQL deletes exactly the original otherwise-unreferenced rejected result after SIGKILL, before worker becomes eligible and retransmission. Same operation actually changes to accepted/successor2; unchanged oracle reports `retained lifecycle_results row changed or lost`. Correct variant retains the original rejected descriptor/result and a new operation accepts. Grant and generation/revision are unchanged, so no unrelated guard masks the defect.
2. **Recovered policy reset:** after successful live Open/epoch renewal, separate test-only injection resets inner spent and pending in the same run; actual Run then settles evidence but cannot consume the mismatched pending operation. Independent history/projection comparison reports `projection disagrees with committed ordinary history`. It does not rely on runtime replay rejecting corruption, or claim a repeated decision occurred.
3. **Speculative dispatch:** a test-only executor test binary installs a temporary SQLite trigger on the real Commit, before the projection UPDATE, after intent/history writes are staged. The fault sends that staged external intent through the actual declared HTTP protocol without committed authorization. The unchanged production boundary observer later pauses after all staged writes; SIGKILL rolls back the transaction. Endpoint application persists with no committed intent/attempt, and oracle reports `endpoint application lacks committed authorized intent`. Correct counterpart has zero endpoint applications before rollback and one after live recomputation.

The non-idempotent duplicate experiment is separately labeled `explicit unsafe non-idempotent repetition`: posting the original request intentionally a second time changes total5 to10 and creates two application receipts. This is not an automatic recovery strategy or a business-level guarantee.

## Retained controller defect and regression

Parent review found a real controller-oracle defect before freeze: child.wait compared generation/epoch only with `>0` and omitted expected Intent/Attempt. Original input expected inner/gen1/epoch2/seq7/opA/authA but a matching token/PID/store/subject/seq ACK with gen2/epoch99/opB/authB could pass. This is an oracle defect, not an observed runtime safety failure.

Original parent snapshot/probe: `.superpowers/sdd/00-plan-phase1d/task-4-before-barrier-fix`; original actual-pipe child.wait probe log `/tmp/alan-task4-barrier-original.log`. Local regression RED `/tmp/alan-task4-barrier-red.log` independently showed one-field generation, epoch, operation and attempt substitutions accepted by the original predicate; older/newer sequence and wrong phase already rejected.

The correction compares all fields exactly and verifies retained facts before allowing the claimed kill. The regression now sends each one-field mismatch on an actual pipe through the same readAcknowledgment path used by child.wait. Runtime filtering also checks specified generation, epoch and attempt. GREEN `/tmp/alan-task4-barrier-green.log`. This preserves the Phase1C same-execution/ordered-interruption lesson without inserting a runtime policy oracle.

## Test-first evidence and verification

Initial feature RED logs and observed failures:

- `/tmp/alan-task4-resource-red.log`: resource durable key/original receipt and unsafe repetition tests fail `not implemented`; GREEN `/tmp/alan-task4-resource-green.log`.
- `/tmp/alan-task4-external-red.log`: external automatic receipt/unknown tests fail `invalid permission kind`; GREEN `/tmp/alan-task4-external-green.log`. Initial import-name/build and sandbox-loopback setup errors were corrected before treating the behavior failure as RED.
- `/tmp/alan-task4-controller-red.log`: actual CLI completes without acknowledgment (`EOF`) for staged and committed cuts; GREEN `/tmp/alan-task4-controller-green.log`.
- `/tmp/alan-task4-resource-http-red.log`: no resource command exists; GREEN `/tmp/alan-task4-resource-http-green.log` exercises real separate process/HTTP.
- `/tmp/alan-task4-unknown-red.log`: RunReport lacks explicit UnknownExternal; final GREEN in `/tmp/alan-task4-timeout.log`, which also confirms a real two-second timeout after endpoint application leaves unknown while a ready peer commits.
- `/tmp/alan-task4-resource-version-red.log`: read-only resource inspection incorrectly accepted unsupported metadata version; GREEN `/tmp/alan-task4-resource-version-green.log`.
- `/tmp/alan-task4-resource-settings-red.log`: read-only resource inspection incorrectly accepted non-WAL store; GREEN `/tmp/alan-task4-resource-settings-green.log` after readback validation of all settings/engine.
- Controller identity RED/GREEN described above; the original input is retained.

Expanded scenario/coverage logs (passed when added): `/tmp/alan-task4-nested-crashes.log`, `/tmp/alan-task4-external-crashes.log`, `/tmp/alan-task4-controls.log`. These later coverage tests are not misrepresented as independent preimplementation REDs. Fault controls intentionally produce observable semantic violations and assert that the unchanged oracle rejects them.

Final commands and actual results are appended below. All loopback subprocess commands required targeted sandbox escalation because local bind was denied in the normal sandbox; that authorized escalation succeeded. No automatic-review rejection blocked work.

## Evidence locations and limitations

Every controller invocation writes JSON evidence files and logs `DURABLE` records. Records retain complete provision/program bytes or initial snapshots, pre/post semantic tables/epochs, actual process command/PID/exit/barrier, binary SHA256, selected environment/race mode, endpoint requests/applications/receipts, inputs and explicit assertions/counterexample failures. Actual ephemeral paths/PIDs/ports are retained rather than invented deterministic identities. Build metadata confirms Go1.27.3, darwin/arm64, CGO1, mattn SQLite1.14.52 and race instrumentation where applicable. Parent will add final source/dependency package identity after freeze.

Endpoint contact counts describe valid protocol requests accepted into its request ledger; they are not a packet sniffer for malformed/unrelated HTTP traffic. Replay's implementation has no dispatch path, and tests compare all retained executor and endpoint semantic records. This task supplies the required crash/control matrix; broader independent model-conformance/qualification bridge remains Task5. No cross-host ownership, network filesystem durability, business compensation, retention/GC, schema migration, arbitrary coherent-tamper detection or external effect exactly-once guarantee is claimed.

## Final verification result and frozen evidence

Final commands, after all source and test edits:

```sh
GOCACHE=/tmp/alan-phase1d-go-build go test -count=1 ./...
GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor ./internal/fixture ./internal/boundary ./internal/durabletest ./cmd/alan ./cmd/alan-resource
GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
git diff --check
```

All exit0. Full uncached normal log `/tmp/alan-task4-full.log`: CLI0.215s, boundary0.132s, controller4.827s, executor2.876s, resource0.158s, historical semantics4.822s. The resource command is tested through its real subprocess rather than standalone main-package unit tests.

Focused race log `/tmp/alan-task4-race.log`: executor7.448s, resource1.223s, controller55.756s; unchanged boundary/CLI packages cached PASS. The controller's race-tag constant adds `-race` to **both** real child executable builds and the test-only speculative dispatcher test binary. `/tmp/alan-task4-child-build-info.log` confirms `-race=true` on all three final child binaries. This is not controller-only race coverage. No race diagnostics occurred. Vet and diff check have no diagnostics.

Final **normal** controller evidence: `/var/folders/qn/d3_l9fv5585g36qycsc804hh0000gn/T/alan-task4-evidence-3798343989` (87 JSON records plus actual databases, provision/input files, child logs and compiled binaries).

Final **race** controller evidence: `/var/folders/qn/d3_l9fv5585g36qycsc804hh0000gn/T/alan-task4-evidence-1330739001` (87 JSON records plus databases, inputs/logs and race binaries).

Five control-comparison records contain two correct variants with no oracle failures and the three explicit faults with exactly the expected observable failure categories described above. All final source is frozen; only this report was written after the final checks. No known open implementation/test failure remains for Task4 review.
