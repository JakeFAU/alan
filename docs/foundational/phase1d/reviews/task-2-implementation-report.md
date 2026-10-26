# Task 2 report: automatic runner, ordinary messaging, inspection and CLI

Status: Task 2 implementation complete and frozen for independent review. No commits or subagents. No historical research files/evidence changed. Parent owns public Phase 1D documents. Task 1 reviewed output-limit guards and canonical history/intent comparison remain intact.

## Files and scope

New implementation: `internal/executor/{runner,messages,inspect}.go`, `cmd/alan/main.go`.
New tests: `internal/executor/{runner,messages,inspect}_test.go`, `cmd/alan/main_test.go`.
Supporting changes: `store.go` permission/route/ack schema and transaction-local observation helper; `replay.go` frozen route representation and new protocol-evidence validation. Task 1 `program.go` is unchanged in this task, including MaxOutputBytes enforcement.

No dependency changes. The CLI uses standard-library flag/JSON/signal handling, consistent with the user's selected dependency policy rather than the generic CLI skill's additional-library defaults. Applied existing Go database/testing/error/context guidance and TDD workflow.

## Public interfaces

```go
func (s *Store) Run(ctx context.Context, options RunOptions) (RunReport, error)
func (s *Store) AdmitInput(ctx context.Context, input Input) (AdmissionResult, error)
func (s *Store) AuthorizeIntent(ctx context.Context, id string) (Authorization, error)
func (s *Store) DeliverMessage(ctx context.Context, intentID, transportID string) (AdmissionResult, error)
func (s *Store) AcknowledgeMessage(ctx context.Context, intentID, transportID string) error
func (s *Store) DeliverTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) (AdmissionResult, error)
func (s *Store) AcknowledgeTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) error
func Inspect(ctx context.Context, path string, limits Limits) (Inspection, error)
```

Key representations:

```go
type RunOptions struct { MaxSteps, MaxDispatch int }
type RunReport struct {
    Steps, Dispatches int
    Quiescent bool
    OpenRuns []RunID
    UnresolvedIntents []Intent
    PendingDeliveries []string
    RejectedAdmissions int
}
type Input struct {
    TransportID, LogicalID, Target string
    Generation int64
    Obligation, Operation string
    Payload map[string]Value
}
type AdmissionResult struct {
    TransportID string
    Accepted bool
    Reason string
    Bound *RunID
    ObservationID string
}
type TerminalDestination struct { Target, Obligation string }
type Authorization struct {
    ID, IntentID string
    Allowed bool
    Reason string
    Source Authority
}
type AdmissionDescriptor struct {
    TransportID, Kind, LogicalID string
    Source RunID
    Target string
    Generation int64
    Obligation, Operation string
    Payload map[string]Value
    IntentID string
    TerminalSequence int64
}
type ConnectionSettings struct {
    JournalMode, SQLiteVersion, SQLiteSourceID string
    Synchronous, ForeignKeys, BusyTimeoutMS, QueryOnly int
}
type Inspection struct {
    Settings ConnectionSettings
    Tables map[string][]map[string]any
}
```

AdmissionDescriptor is inspectable retained evidence, **not** a public arbitrary-origin admission API. The trusted `recordObservation` and transaction-local `insertObservation` functions remain private. Permission now additionally contains `Obligation string`.

## Scheduling and unresolved work

The runner performs deterministic round-robin selection over sorted subject/generation IDs, skips blocked receives, and selects the lexicographically first unconsumed observation that the same pure reducer accepts under exact correlation. A non-receive evaluates with no observation. The actual selected run/sequence/epoch and observation identity are retained by the existing execution history transaction. It invokes real Store.Commit; applications require no harness Prepare/Commit calls.

After each committed instruction and on initial/recovery entry, pending supported deliveries are handled before another instruction. Message effects are recognized solely by the generic `Intent.Kind == "message"` protocol. Terminal delivery comes from frozen terminal obligations, independently of source-run openness. No program-role/name, ancestry or generation-success branch exists. Renamed terminal propagation fixtures produce the same public result.

A dispatch comprises a bounded number of separate transactions: source authorization, admission, then acknowledgment for an ordinary message; admission then acknowledgment for a terminal notice. The runner does not combine these protocol boundaries. Manual public delivery/ack operations allow lost acknowledgments to be exercised. Stable automatic transport IDs are JSON tuples derived from the original intent or frozen terminal/source/destination binding.

`MaxSteps` and `MaxDispatch` are positive operational budgets. Exhaustion returns ErrLimit with a report containing remaining open runs, unresolved intent descriptors and pending delivery keys. Unsupported intent kinds remain visible and are not dispatched. Quiescent means there is no supported eligible work now; it does not mean all work succeeded or that waiting/unresolved work will eventually progress. Open runs and unsupported intents can remain in a quiescent report. RejectedAdmissions counts retained rejection dispositions, including older admissions.

The current runner exits at quiescence and is not a daemon. A later explicit input plus run/reopen continues the same logical execution. The round-robin cursor is ephemeral; restarting may change the next eligible scheduling choice, while retained selections/history remain exact. No unbounded fairness or global total-order journal is claimed; execution history identifies each committed per-run selection.

## Ordinary input and authenticated origins

The local input capability is fixed to Source=`local`, SourceGeneration=1 (the fixed local ingress identity), Kind=`input`, Provenance=`local-input`. Input has no Source/Kind/Provenance fields. Payload keys with those names remain ordinary data. The CLI uses strict JSON unknown-field rejection, so an input document cannot claim terminal/lifecycle/receipt provenance. This is a local operator capability, not remote user authentication.

An explicit permission `(kind=input, source=local, target, operation=input, revision=0)` is required. Ordinary message permission is `(kind=message, source=<committed source subject>, target, operation=message, revision=<committed source revision>)`.

Message origin is reconstructed from the immutable committed intent, with Kind=`message`, Provenance=`ordinary-message`, the exact source execution and intent logical key. Its receive Operation is empty; applications can carry request/reply references as ordinary payload. Message delivery cannot use an uncommitted identity. Payload claims cannot change runtime metadata.

Terminal origin is reconstructed from the retained terminal outcome and frozen destination, with Kind=`terminal`, Provenance=`runtime-terminal`, the exact source subject/generation and sequence-bound logical identity. Its payload exposes only `outcome` (the public classification), not source private data. Its obligation is the explicitly provisioned route obligation, not something read from receiver policy state. Frozen terminal delivery does not require a new ordinary source authorization after closure.

## Admission, authorization and acknowledgments

Admission compares the complete immutable descriptor for an existing transport ID before reevaluating target circumstances. Exact retransmission returns its original accepted/rejected result and Bound run. Changed fields under the same transport ID are ErrConflict.

Generation 0 requests select the target's current run once; an explicit generation must match. Missing/retired/closed targets, target-generation mismatch, missing route and source denial produce retained rejection results. Malformed identities/descriptors, conflicting logical identities and configured byte-limit violations are errors rather than normal semantic admission dispositions.

Accepted admission and inbox insertion share one transaction. Different transport IDs carrying the same logical observation compare exact observation descriptors and reuse the same observation row and consumed_sequence. Changing payload or binding under a logical observation ID fails with ErrConflict; an old logical observation is not silently rebound. Global observation identities are namespaced JSON tuples by origin kind and logical key.

Generic `AuthorizeIntent` is a separate retained source disposition in the attempts relation. It checks the committed intent's source generation/revision, retirement and openness. A new authorization after closure is denied and retained with its reason. Existing authorization survives source closure/reopen. An unknown/uncommitted ID errors without inventing an attempt. Authorization does not confer target route or lifecycle authority. In Task 2 a message source denial is retained in attempts and the admission rejection; future lifecycle result observation of source denial remains Task 3.

Ordinary-message admission and acknowledgment are separate: message_acknowledgments references the retained admission. A lost acknowledgment can repeat delivery under the original authorization/transport without another inbox row. Terminal acknowledgment separately updates the frozen obligation's route-keyed delivery_status. A true status means the admission disposition was acknowledged, including an explicit rejection; it does not turn rejection into successful inbox delivery. All admission details remain inspectable.

## Exact schema refinements for parent documentation

SchemaVersion/user_version remain 1 during this unreleased qualification work; no migration is introduced. New stores use the refined DDL. Task 1 prototype databases are not promised compatibility; a missing new relation/column fails startup rather than being silently migrated.

1. `permissions` now contains `(kind, source, target, operation, revision, obligation)`, all NOT NULL, with all six fields in its primary key. The source FK is removed to permit the fixed non-subject local-input principal. Create explicitly validates non-input source subjects and positive revision; input requires exactly source local, operation input, revision 0. Existing kinds remain lifecycle/observation/message; input is added.
2. Observation routes use kind observation, operation terminal, explicit source revision, target and Obligation. Only matching revision routes are frozen on ordinary closure. The query returns sorted distinct `{Target,Obligation}` destinations.
3. `terminal_obligations.destinations` stores JSON `[]TerminalDestination` instead of `[]string`. `delivery_status` remains a JSON map of booleans, now keyed by canonical tuple `["route",target,obligation]`. Replay uses the closing history record's source revision when reconstructing the static frozen route set.
4. Added `message_acknowledgments(intent_id TEXT PRIMARY KEY REFERENCES intents(id), transport_id TEXT NOT NULL REFERENCES admissions(id))`.
5. Existing `admissions` is now populated with complete AdmissionDescriptor/result JSON and nullable observation_id FK.
6. Existing `attempts` is now populated for generic source authorization: stable ID `["authorization",intentID]`, exact `{Intent,Revision}` descriptor, and Authorization result including the source Authority checkpoint.
7. Lifecycle-results and settlements relations remain reserved/empty; no lifecycle or endpoint handler was added.

The static terminal route obligation is an explicit configuration cost. It supports same-execution observation/recovery; it does not implement fresh-supervisor reassociation or a cross-generation supervisor budget.

## Startup/replay and read-only inspection

Existing artifact/history/projection/intent reconstruction remains unchanged. New `validateMessaging` additionally checks:

- Exact authorization descriptor/intent identity, source execution boundary and retained allowed/denied classification.
- Complete admission descriptor/result identity, original binding, observation linkage and static route evidence for accepted admissions.
- Message origin and retained source disposition; terminal origin against the frozen outcome/routes; fixed local-input origin.
- Exact expected observation content for accepted admissions; every observation claiming one of the three runtime ingress provenance values must have supporting admission evidence.
- Message ACK references a matching message admission; terminal ACK has matching retained terminal admission evidence.

It does not reevaluate a retained admission against current target openness/current generation; that would erase prior results. Lifecycle-derived authority/projection transitions still require Task 3 reconciliation as already recorded in Task 1. The private trusted-fixture provenance used by older storage tests is not an externally accessible ingress.

Inspect opens mode=ro + query_only and one read transaction, verifies connection/schema/semantic/artifact identity, and returns every relation: metadata, programs, subjects, permissions, runs, history, observations (including consumed_sequence), intents, attempts, admissions, lifecycle_results, settlements, terminal_obligations and message_acknowledgments. SQLite BLOB JSON is exposed as its exact UTF-8 string, retaining original artifact bytes for hashing/comparison. Rows are ordered by retained rowid and output map keys by normal JSON ordering. Inspection performs no policy reduction; an ordinary program waiting on missing local data can still be inspected. Replay remains the separate stronger reconstruction operation. Tests compare complete coherent inspection snapshots before/after replay.

## CLI

```text
alan init    --db PATH --provision JSON
alan run     --db PATH [--max-steps N] [--max-dispatch N]
alan input   --db PATH --request JSON
alan inspect --db PATH
alan replay  --db PATH
```

All commands accept positive `--max-program-bytes`, `--max-data-bytes`, `--max-observation-bytes`, `--max-output-bytes` (default 1 MiB each). Run defaults to 10,000 steps and 10,000 dispatches; they are operational budgets. Init reads a JSON Provision with Artifact.Bytes represented by normal Go JSON base64. It provisions only a new store. Run/input use Open (physical activation epochs may advance); inspect/replay never do. Input JSON is decoded before Open, so unknown forged origin fields cannot even advance an epoch. Input admits only; it does not execute policy. Run performs the progress.

Stdout is machine-readable JSON. Diagnostics go to stderr. Usage errors exit 2; runtime/limit errors exit 1. Run still prints the unfinished report on an operational limit. Signal cancellation propagates through the command context; no CLI dependency was added.

## Test-first evidence and verification

There are now 40 top-level tests across executor and CLI (Task 1 baseline 23, Task 2 additions 17), plus subtests.

Initial runner/message RED: `/tmp/alan-task2-core-red.log`. New input permission tests failed `invalid permission kind`; other automatic delivery/runner tests failed `not implemented`. These were feature-absence failures, not compilation errors. Core GREEN: `/tmp/alan-task2-core-green.log`, including all previous Task 1 tests and real pre/post-COMMIT SIGKILL tests.

Inspection/CLI RED: `/tmp/alan-task2-inspect-cli-red.log`, both inspection cases and both CLI flows failed `not implemented`. GREEN: `/tmp/alan-task2-inspect-cli-green.log`.

Protocol-integrity RED: `/tmp/alan-task2-auth-validation-red.log`. Changing accepted=true to false, changing admitted input provenance to runtime-terminal, or deleting the supporting admission all remained incorrectly accepted by Replay. After `validateMessaging`, GREEN is `/tmp/alan-task2-auth-validation-green.log`; it also covers rollback, source denial, missing route and lost ACK paths.

Additional coverage, explicitly not claimed as independent RED cycles:

- Admission-table insertion failure rolls back accepted inbox insertion.
- Missing message route and new source authorization after closure retain distinct stable rejection evidence.
- Terminal acknowledgment without admission is rejected, and forged retained ACK is rejected by replay.
- Automatic public terminal propagation is unchanged after renaming both subjects.
- Dispatch budget exhaustion leaves pending/unresolved work visible.
- Full semantic inspection is unchanged by inspect/replay; inspection does not reduce ordinary policy.
- CLI init → waiting run → explicit input → completed ordinary run → inspect/replay; strict origin-field and usage rejection.

Focused actual run: `/tmp/alan-task2-focused-green.log`:

```text
GOCACHE=/tmp/alan-phase1d-go-build go test ./internal/executor ./cmd/alan
ok github.com/JakeFAU/alan/internal/executor 0.524s
ok github.com/JakeFAU/alan/cmd/alan 0.265s
```

Final whole-tree and focused-race results:

```text
GOCACHE=/tmp/alan-phase1d-go-build go test ./...
ok github.com/JakeFAU/alan/cmd/alan (cached)
ok github.com/JakeFAU/alan/internal/executor (cached)
ok github.com/JakeFAU/alan/research/semantics (cached)

GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor ./cmd/alan
ok github.com/JakeFAU/alan/internal/executor 3.617s
ok github.com/JakeFAU/alan/cmd/alan 1.320s

GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
(no diagnostics, exit 0)

git diff --check
(no diagnostics)
```

Logs: `/tmp/alan-task2-full.log`, `/tmp/alan-task2-race.log`, `/tmp/alan-task2-vet.log`. Cached historical results are disclosed; no historical code changed.

Built-binary smoke evidence:

```text
GOCACHE=/tmp/alan-phase1d-go-build go build -buildvcs=false -o /tmp/alan-task2-cli ./cmd/alan
```

An initial build emitted a nonfatal sandbox VCS/module-stat-cache warning; the explicit `-buildvcs=false` build completed without diagnostics, exit 0. This flag affects build metadata, not runtime semantics. The actual CLI binary was exercised in `/tmp/alan-task2-cli-smoke-l731795b` with retained provision/input/init/waiting/admission/execution/replay and before/after inspection JSON files. The initial run reported Quiescent=true, Steps=0, one open ordinary/1 run. After input value 4321, run reported Steps=2 and no open runs. Replay reconstructed the same ordinary/1 execution at epoch 4, sequence 2, succeeded, value 4321, using bundled SQLite 3.53.4/source `2026-07-24 19:02:57 bf7c7f30031888f4e796e429ab3978879485813aaca6f641c7b33e4e09459bcc`. `cmp` found before/after inspection JSON identical.

## Pending cross-slice checks and limitations

- **Real generation-change admission binding is intentionally pending Task 3.** Task 2 establishes first binding and retransmission across target closure and physical reopening. It does not use test SQL to manufacture a generation transition. Task 3 must perform a real accepted Retry, then retransmit the identical original transport and verify retained original binding/result.
- Lifecycle handlers, authority-induced closure/successor replay, source-denial result observations for lifecycle requests, nested policies and the central same-obligation crash scenario remain Task 3+.
- External resource protocol, physical attempt receipts, unknown mutation outcomes and endpoint idempotency remain Task 4+.
- Dispatch prioritizes supported pending deliveries and is bounded; no unbounded fairness or asynchronous transport liveness is claimed.
- The local input interface assumes authority to operate the local store; it is not a remote identity provider. Its immutable origin cannot claim runtime privileged metadata.
- Semantic rejection is retained; malformed/limit/identity-conflict errors are not fabricated protocol successes or deterministic program failures.
- Plain-message transport acknowledgment is ledger evidence, not an automatically delivered application-level receipt. Applications can use an explicit ordinary reply message.
- Inspection exposes all records but does not itself perform semantic replay; call Replay for stronger reconstruction.
- Complete retention and full relation scans remain deliberate; no history bound, migration, garbage collection or production throughput claim is added.

Ready for task-scoped independent review. Implementation paused here.

## Review fix round 1 — R1 trusted transport-origin namespaces

Independent Task 2 review returned specification FAIL / quality CHANGES REQUESTED for one P2 correctness finding: ordinary raw input IDs occupied the same durable admission-key namespace as generated message and terminal delivery IDs. This scoped round fixes that identity boundary only. No Task 3 handlers, additional dependencies, commits, subagents or historical research changes were made. Source is frozen for re-review.

### Exact reproduction and retained RED evidence

Read the complete review and its two isolated probes before changing production code. Both original legal inputs are retained as literal strings in `internal/executor/transport_domains_test.go`:

```text
Accepted ordinary input:
TransportID = ["message-delivery","[\"sender\",1,1,0]"]
LogicalID   = ordinary-input
Target      = receiver
Generation  = 0
Payload     = {number: Int(7)}

Rejected ordinary input:
TransportID = ["terminal-delivery","child",1,1,"observer","public"]
LogicalID   = rejected-input
Target      = missing
Generation  = 0
Payload     = {}
```

The accepted input preceded an ordinary sender/receiver message with payload number 42. The rejected input preceded a child failed outcome and its explicit public terminal route. Neither input supplies runtime provenance or performs private SQL. Before the fix, each Run made one execution commit and then stopped with ErrConflict at its generated delivery. The original reviewer evidence also demonstrates the message conflict persisting across real Close/Open.

Original reviewer sources/output are preserved unmodified as `task-2-r1-original-probe.go.txt` and `task-2-r1-original-probe.log` beside this report; originals remain under `/tmp/alan-task2-review-probe` and `/tmp/alan-task2-review-probe.log`.

New tests were written before the fix. RED command:

```text
GOCACHE=/tmp/alan-phase1d-go-build go test -v ./internal/executor \
  -run 'TestAcceptedInputCannotShadow|TestRejectedInputCannotShadow|TestMessageTerminalAndInputShare'
```

Exact log `/tmp/alan-task2-r1-red.log` records:

- Accepted original input: Steps 1, Dispatches 0, both runs open, original message unresolved, ErrConflict.
- Rejected original input: Steps 1, Dispatches 0, observer open, frozen terminal delivery pending, ErrConflict.
- Input/message/terminal using the same arbitrary raw string: message admission fails with ErrConflict after the input has occupied that raw ID.

No prefix/string was banned, rewritten or removed from the accepted input domain.

### Durable identity correction

The trusted APIs still choose the domain: AdmitInput constructs input, DeliverMessage reconstructs message from a committed intent, and DeliverTerminal reconstructs terminal from a frozen obligation. The caller cannot supply this discriminator through Input. The private admission helper accepts only those three trusted domains; it does not parse a domain out of raw transport text.

The durable key is now the canonical JSON tuple:

```text
["admission", trustedDomain, rawTransportID]
```

Every raw string remains legal, including strings that resemble generated delivery IDs or complete admission keys. Identical raw IDs can exist independently in input, message and terminal domains. Within one domain, the exact immutable descriptor comparison is unchanged: changed descriptors under the same raw ID return ErrConflict. Public Input/Delivery/Acknowledgment method signatures remain unchanged; AdmissionDescriptor.TransportID and AdmissionResult.TransportID continue to retain and return the exact raw caller ID.

Schema amendment (supersedes the original Task 2 admission/ACK descriptions):

```sql
CREATE TABLE admissions (
  id TEXT PRIMARY KEY,
  domain TEXT NOT NULL CHECK(domain IN ('input','message','terminal')),
  transport_id TEXT NOT NULL,
  descriptor BLOB NOT NULL,
  result BLOB NOT NULL,
  observation_id TEXT REFERENCES observations(id),
  UNIQUE(domain,transport_id)
);
CREATE TABLE message_acknowledgments (
  intent_id TEXT PRIMARY KEY REFERENCES intents(id),
  admission_id TEXT NOT NULL REFERENCES admissions(id)
);
```

Message acknowledgment looks up the **message-domain** admission and persists its canonical admission ID. Terminal acknowledgment looks up the **terminal-domain** admission before updating the frozen route status. Ordinary input rows therefore cannot satisfy or shadow either ACK lookup. The message ACK FK is renamed from transport_id to admission_id to state what it references; raw transport identity remains directly inspectable on the admission row and within its descriptor/result.

Replay/startup now validates the stored id against domain + raw transport_id, domain against the reconstructed descriptor kind, and raw transport_id against both descriptor and result. It reconstructs message/terminal evidence using the raw ID while indexing admissions and ACKs by canonical typed ID. Existing exact observation, binding, authorization, outcome and descriptor equality checks remain active. Inspection automatically exposes id/domain/transport_id and the typed acknowledgment FK in its coherent snapshot; no policy execution was added to inspection.

SchemaVersion remains 1 in this unreleased prequalification profile, with no migration. Earlier Task 2 prototype stores lacking domain/transport_id/admission_id fail closed instead of being silently converted. This is a disclosed schema refinement, not a backwards-compatibility claim.

### Regression coverage and unchanged contracts

Four top-level regression tests were added (44 total across executor/CLI):

1. **Original accepted input versus automatic message:** exact raw ID is admitted and retained; automatic Run completes four ordinary commits and one dispatch; receiver consumes number 42 from authenticated message while the colliding input observation remains unconsumed. Reopen retains the original accepted input, message admission, ACK and run binding.
2. **Original rejected input versus automatic terminal:** exact raw ID retains target_missing; automatic Run completes child failure plus observer receive/end (three commits, one dispatch). Reopen preserves the input rejection, terminal admission and ACK, with one terminal observation.
3. **All three domains share an arbitrary raw string:** input rejection, ordinary-message acceptance and terminal acceptance coexist under `same caller-chosen transport`. Their inspectable durable IDs are distinct. Changed descriptors within input, message and terminal each still produce ErrConflict. Message/terminal retransmission after Open returns the original result/binding, and ACKs refer to the message-domain durable admission ID.
4. **Typed-key/ACK corruption:** Replay rejects changed domain, changed raw-ID column, a noncanonical untyped id, and an ACK redirected to a valid input-domain row. This additional corruption coverage passed on first run after the scoped fix; it is not claimed as an independent RED cycle.

The existing source-authorization-after-closure, message lost-ACK, terminal lost-ACK, immutable original admission, logical observation deduplication and read-only inspection/replay tests were not weakened or redefined. They continue to pass. The genuine Retry generation-binding test remains pending Task 3 as before.

Production changes relative to `task-2-before-fix`: only `internal/executor/messages.go` and `internal/executor/store.go`. New tests are in `internal/executor/transport_domains_test.go`. Runner, interpreter/output limits, CLI, inspection implementation, dependencies and previous tests are unchanged.

### Verification and evidence

Focused original regressions GREEN: `/tmp/alan-task2-r1-green.log` (PASS 0.205s).
Expanded originals, post-closure/lost-ACK tests and corruption coverage: `/tmp/alan-task2-r1-expanded-green.log` (PASS 0.308s).

Final commands preserve actual test exit status:

```text
GOCACHE=/tmp/alan-phase1d-go-build go test ./...
ok github.com/JakeFAU/alan/cmd/alan 0.147s
ok github.com/JakeFAU/alan/internal/executor 0.417s
ok github.com/JakeFAU/alan/research/semantics (cached)

GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor ./cmd/alan
ok github.com/JakeFAU/alan/internal/executor 3.672s
ok github.com/JakeFAU/alan/cmd/alan 1.231s

GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
(no diagnostics, exit 0)

git diff --check
(no diagnostics)
```

Logs: `/tmp/alan-task2-r1-full.log`, `/tmp/alan-task2-r1-race.log`, `/tmp/alan-task2-r1-vet.log`. Cached historical test output is disclosed; historical source is unchanged. No production/test source changes followed these runs. Report and original-probe preservation completed afterward. Ready for task-scoped re-review.
