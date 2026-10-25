# Task 1 report: durable execution commit and reopen

Status: implementation complete for Task 1, ready for independent review. No commits made. No old research code or evidence changed. This is the first runtime slice, not completion of Phase 1D protocols or qualification.

## Files and dependency

- `internal/executor/program.go`: closed finite pure interpreter and immutable program encoding.
- `internal/executor/store.go`: schema, connection verification, OS exclusivity, provisioning, recovery, execution transaction and trusted private observation insertion.
- `internal/executor/replay.go`: read-only consistency reconstruction and validation.
- `internal/executor/{program,store,crash}_test.go`: 18 top-level tests plus subtests, real SQLite, OS subprocesses and SIGKILL.
- `go.mod`, `go.sum`: one direct dependency, `github.com/mattn/go-sqlite3 v1.14.52`; no other infrastructure libraries.

Read binding Phase 1D and Phase 1C contracts, task brief, database/error/testing/security Go skills, and TDD/verification references. User-authorized schema and standard-library-plus-selected-driver choices override generic database-skill advice. No subagents were spawned.

## Exact public interfaces

```go
func EncodeProgram(p Program) ([]byte, string, error)
func Reduce(p Program, s Projection, o *Observation, id RunID, seq int64, l Limits) (Reduction, error)
func DefaultLimits() Limits
func Int(n int64) Value
func Text(s string) Value
func Lit(v Value) Operand
func Ref(s string) Operand

func Create(ctx context.Context, path string, p Provision, limits Limits) (*Store, error)
func Open(ctx context.Context, path string, limits Limits) (*Store, error)
func (s *Store) Close() error
func (s *Store) Snapshot(ctx context.Context, id RunID) (RunState, error)
func (s *Store) Commit(ctx context.Context, a Authority, selected string) (RunState, error)
func (r RunState) Authority() Authority
func Replay(ctx context.Context, path string, limits Limits) (ReplayReport, error)
```

`Store.recordObservation(ctx, Observation)` is deliberately **unexported**. It is a trusted internal insertion primitive, not transport authentication. Later protocol APIs must authenticate provenance and original run binding before insertion, and use transaction-local insertion for atomic transport admission. The CLI cannot call this primitive. Snapshot currently exposes only a run; later inspection needs full protocol records for independent evidence.

Important types:

- `RunID{Subject string, Generation int64}`.
- `Authority{Run RunID, Epoch, Sequence, Revision int64}`; checked against current subject generation, retirement and run state before reduction/writes.
- `RunState{ID RunID, ProgramID string, Epoch, Sequence, Revision int64, Projection Projection}`.
- `Projection{PC int, Data map[string]Value, Status string}`.
- `Value{Kind string, Number int64, Text string, Flag bool}`. Tags are `int`, `string`, `bool`; inactive fields must be zero.
- `Operand{Ref string, Literal *Value, Offset *int64}`. Exactly one reference/literal; optional checked integer offset.
- `Program{Semantics string, Code []Instruction}`. `SemanticVersion = "alan/1d/v1"`.
- `Instruction{Op, Dst string, A, B Operand, Target, Otherwise int, Match *Match, Bind map[string]string, Effect *Effect, Updates map[string]Operand, Outcome string}`.
- `Match{Kind, Provenance string, Source, Generation, Obligation, Operation Operand}`. All matching fields are exact, including original receiving run.
- `Effect{Kind string, Target, Generation, Obligation Operand, Payload map[string]Operand}`.
- `Observation{ID string, Run RunID, Kind, Provenance, Source string, SourceGeneration int64, Obligation, Operation string, Payload map[string]Value}`.
- `Intent{ID string, Run RunID, Sequence int64, Ordinal int, Kind, Target string, Generation int64, Obligation string, Payload map[string]Value}`.
- `Artifact{ID, Semantics string, Bytes []byte}`; content identity is lower-case SHA-256 of exact complete program bytes, separate from semantic version.
- `Subject{ID string, Generation int64, Owner string, Revision int64, Retired, RetryCancelled bool}`.
- `InitialRun{ID RunID, ProgramID string, Projection Projection}`.
- `Permission{Kind, Source, Target, Operation string, Revision int64}`; kinds lifecycle/observation/message share one explicit permission relation.
- `Provision{Programs []Artifact, Subjects []Subject, Runs []InitialRun, Permissions []Permission}`.
- `Limits{MaxProgramBytes, MaxDataBytes, MaxObservationBytes int}`; explicit default 1 MiB for each. Nonpositive applicable limit or overflow produces `ErrLimit`; there is no research scalar/generation bound.
- `ReplayReport{SQLiteVersion, SQLiteSourceID string, HistoryCount, IntentCount int, Runs []RunState}`.

Sentinels: `ErrLimit`, `ErrIneligible`, `ErrClosed`, `ErrVersion`, `ErrCorrupt`, `ErrConflict`, `ErrStale`, `ErrLocked`. Operational database/validation errors may also be returned with context; not every malformed-state diagnostic currently wraps ErrCorrupt.

## Finite computation

One scheduled reduction executes exactly one instruction. `set` evaluates one operand, `add` checked integer addition, `branch` equality with explicit true/false PCs, `jump` one explicit PC transition, `receive` consumes the selected exact-correlated observation and binds declared payload fields, `emit` creates one generic immutable effect descriptor plus declared ordinary local updates, and `end` freezes ordinary succeeded/failed outcome. Loops always yield through repeated commits. No callbacks, supervisor opcode, program-name branch, ancestry special case, generation-dependent success or historical constants are present.

Emit evaluates every payload and update operand against the pre-step data. An offset such as `spent + 1` and saving the pending intent key occur in the same execution commit. The intent ID is the unambiguous JSON tuple `[subject,generation,sequence,ordinal]`; ordinal is 0 in this one-effect-per-instruction language. Receive rejects two payload fields binding the same local destination, avoiding map-order nondeterminism. Ordinary programs cannot claim `cancelled` or `abandoned`; these classifications require future lifecycle authority evidence.

JSON objects are generated with deterministic standard-library map-key ordering. Strict decoding rejects unknown fields/trailing JSON. Programs retain exact artifact bytes, hash and semantic version; projection/history/observation/intent comparisons use canonical generated bytes where relevant.

## Exact schema and transaction boundaries

`SchemaVersion = 1`; both `metadata.schema_version` and SQLite `user_version` must match. DDL is the `schema` constant in store.go.

| Relation | Columns / keys / binding |
| --- | --- |
| metadata | singleton PK constrained to 1, schema_version, semantics |
| programs | id PK, semantics, bytes BLOB |
| subjects | id PK, positive generation, owner, positive revision, boolean retired/retry_cancelled |
| permissions | kind/source/target/operation/revision composite PK; source FK subjects |
| runs | subject/generation PK, program_id FK, immutable initial BLOB, positive epoch, nonnegative sequence, projection BLOB; subject FK |
| history | subject/generation/sequence PK, positive epoch/source_revision, nullable selected_observation FK, successor BLOB, intents BLOB; run FK |
| observations | id PK, subject/generation run FK, immutable descriptor BLOB, nullable consumed_sequence; deferred composite FK to exact history selection commit |
| intents | id PK, subject/generation/sequence/ordinal UNIQUE, source_revision, descriptor BLOB; exact history FK |
| terminal_obligations | subject/generation PK, sequence/history FK, frozen outcome BLOB, sorted destinations BLOB, delivery_status BLOB |
| admissions | id PK, descriptor, result, observation_id FK; reserved for Task 2 |
| attempts | id PK, intent_id FK, descriptor, result; reserved for Tasks 3/4 |
| lifecycle_results | id PK, descriptor, result; reserved for Task 3 |
| settlements | id PK, intent_id FK, descriptor, observation_id FK; reserved for Tasks 3/4 |

No migration, garbage collection or retention horizon exists. Reserved protocol tables are not implemented protocols or proof of authority. Records are immutable through the provided API except explicit epoch/projection/sequence, observation consumption and future delivery state. External SQL writers are outside the supported model; startup/replay detects tested retained-state corruptions.

1. Create canonicalizes path/acquires exclusivity before creating/opening SQLite, requires a newly created file, then one transaction creates schema and provisions artifacts, subjects, runs and permissions. Full validation precedes COMMIT. It does not reopen/re-provision an existing store. Failed creation can leave an invalid/empty database file that must not be mistaken for initialized state.
2. Open canonicalizes/acquires exclusivity, verifies settings, validates retained artifact/history/observations/intents/terminals/current state inside a consistent transaction, then advances only open-run epochs and COMMITs. Generation, projection, sequence, budget and pending keys are unchanged.
3. Commit validates current authority/openness, selects retained eligible observation, performs pure reduction, marks consumption, inserts reconstructible history and immutable intent rows, advances current projection/sequence, and inserts frozen terminal obligations/destinations when closing, in one transaction. No dispatcher exists.
4. Replay uses a separate `mode=ro` plus `query_only` connection and one read transaction. It does not call Open/Create, advance epochs, dispatch, admit or create a missing database.

## SQLite, filesystem and process evidence

Single pinned `database/sql.Conn`, max one pooled connection, no connection lifetime recycling, mutex-serialized Store operations. Every opened writer/inspector connection reads back WAL, FULL (=2), foreign_keys=1, busy_timeout=5000ms, expected query_only. Engine version/source ID are recorded; engines below 3.51.3 are rejected in this supported profile, ensuring the bundled WAL-reset fix baseline.

Observed driver engine:

```text
SQLite 3.53.4
source 2026-07-24 19:02:57 bf7c7f30031888f4e796e429ab3978879485813aaca6f641c7b33e4e09459bcc
Go go1.27.3 darwin/arm64, CGO-enabled driver
```

Absolute/symlink-canonical paths converge to a persistent `.lock` sidecar. OS `flock(LOCK_EX|LOCK_NB)` is held until Close/process exit; never unlink the sidecar. Nonregular/multiply-linked database files and unsafe sidecars are rejected before SQLite open. Unsupported Windows portability is not claimed; implementation uses Unix syscall flock/O_NOFOLLOW. Concurrent malicious replacement/deletion, network filesystem behavior, external writers and hardware failure remain outside the profile.

`crash_test.go` includes a reusable test-only barrier envelope: scenario, PID, store, phase, exact run, epoch, sequence, intent ID. The pre-COMMIT test registers a SQLite function/temporary trigger solely from test code and pauses the **real Commit** after history+intent writes are staged and before its projection UPDATE/COMMIT. No production failpoint callback was introduced. The post-COMMIT test acknowledges only after Commit returns successfully. Controller validates every envelope field, sends SIGKILL to the acknowledged PID, and verifies signal termination.

Observed before-commit subprocess: PID58420, run worker/42, epoch1, sequence1, intent `["worker",42,1,0]`; after SIGKILL replay found history0/intents0/sequence0. Observed after-commit subprocess: PID58421, same logical fixture identities, distinct scenario/store/process; after SIGKILL replay found history1/intents1/sequence1. Both reopened at epoch2, proving lock release and same-run recovery. Exact unique temporary store/scenario paths and engine output are in `/tmp/alan-task1-expanded-green.log`. These are process-death tests, not power-loss tests.

## Tests and RED evidence

Tests name real behavior and use literal expected values/SQL independent inspection rather than a production reducer oracle for commit effects. Replay itself is explicitly only a consistency check.

- Reducer finite instruction, input immutability, nonresearch values (1000 and generations99/123), explicit limits, loop yield.
- Exact receive kind/provenance/source-generation/obligation/operation/run matching, ordinary terminal closure.
- Atomic arithmetic budget reservation with intent/pending and overflow rejection.
- Ordinary cancellation/abandonment rejected; ambiguous receive bindings rejected.
- Commit/reopen pending/count/sequence retention; stale epoch rejected; terminal obligation retention.
- Selected observation immutable logical identity; history insertion failure rolls back consumption and projection.
- Intent insertion failure rolls back preceding history and all execution writes.
- Stale revision/epoch/sequence/generation rejected without history.
- Non-receive Emit with a selected observation returns ErrIneligible and leaves consumption, history, intents and projection unchanged.
- Unsupported metadata/schema and artifact semantic versions, changed artifact bytes/projection/history/intent and deleted intent rejected by Open/Replay.
- Actual second OS process exclusion, symlink alias exclusion, hardlink rejection; no epoch change from excluded opener.
- Real read-only mutation denial, settings readback, missing replay path never provisioned.
- Selected observation changed with latest projection untouched is rejected.
- Exact artifact bytes retained, same artifact deduplicated, changed bytes under ID rejected.
- Deleted/corrupted terminal outcome/destinations/delivery state rejected.
- Corrupt startup does not advance epoch.
- Acknowledged pre/post-COMMIT SIGKILL atomicity and lock release.

Retained red/green logs:

| Test cycle | RED evidence | GREEN evidence |
| --- | --- | --- |
| Pure reducer initial stubs | `/tmp/alan-task1-reducer-red.log`: finite work fails `not implemented`, exact receive fails `wrong operation: not implemented` | `/tmp/alan-task1-reducer-green.log` |
| Real store/replay initial stubs | `/tmp/alan-task1-store-red.log`: all store behavior tests fail `not implemented` | `/tmp/alan-task1-store-green.log` |
| Atomic arithmetic reservation | `/tmp/alan-task1-reservation-red.log`: spent remains900 instead of901 | `/tmp/alan-task1-reservation-green.log` |
| Ordinary authority outcome | `/tmp/alan-task1-authority-outcome-red.log`: ordinary program manufactured cancelled/abandoned | `/tmp/alan-task1-validation-green.log` |
| Authority outcome + ambiguous binding | `/tmp/alan-task1-validation-red.log`: both privileged outcomes accepted; duplicate binding accepted | `/tmp/alan-task1-validation-green.log` |

Additional acceptance/corruption/SIGKILL tests were added against the implemented core and passed on their first run; they are disclosed as additional coverage, not retroactively claimed as new RED-first cycles. No mutation test result is represented as a feature RED.

## Commands and final results

Dependency download first failed under sandbox DNS; the authorized elevated retry succeeded:

```text
go get github.com/mattn/go-sqlite3@v1.14.52
```

Initial default-cache full/race/vet attempts could not open a Go build-cache file (`operation not permitted`) before test execution. The commands were rerun with a writable explicit cache; this was an environment issue, not a hidden test failure.

```text
GOCACHE=/tmp/alan-phase1d-go-build go test ./...
ok github.com/JakeFAU/alan/internal/executor 0.174s
ok github.com/JakeFAU/alan/research/semantics (cached; preceding full run 5.517s)

GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor
ok github.com/JakeFAU/alan/internal/executor 2.435s

GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
(no diagnostics)

git diff --check
(no diagnostics)
```

Logs: `/tmp/alan-task1-full-green.log`, `/tmp/alan-task1-race-green.log`, `/tmp/alan-task1-vet.log`. Earlier expanded verbose core/SIGKILL output: `/tmp/alan-task1-expanded-green.log`. Controller had already reproduced historical15/20/32 and whole historical race baseline before implementation; this task ran full normal suite and focused new-package race as requested.

## Required follow-up / limitations

1. **Lifecycle reconciliation is intentionally not implemented.** Current replay validates only ordinary execution closure. Current terminal history FK assumes a closing execution commit. Task3 must explicitly extend terminal linkage to retained lifecycle operation/result (including sequence0 cancellation), retain successor creation provenance, and reconcile authority-caused closure/generation separately from reducer history. Do not weaken projection checking to allow unexplained mismatches. Parent was informed before handoff.
2. Reserved admissions/attempts/lifecycle/settlement relations require concrete protocol descriptor types and possibly schema refinement before final schema freeze. They have no handlers, source authorization, target grants, dedupe/rejection replay or receipt semantics yet.
3. The private observation insertion is trusted storage only. Authentication, transport identity, ordinary versus privileged observation separation and exact original binding belong in the later protocol boundary. No public input API exists in Task1.
4. Future inspection must expose all retained protocol records; current Snapshot/Replay report is intentionally narrower.
5. There is no runner, dispatcher, endpoint, scheduling fairness, external exactly-once claim or business retry guarantee yet.
6. Full replay uses the same reducer and therefore is not independent semantic conformance. Independent policy/ledger checkers and historical trace comparisons remain later task work.
7. Resource limits bound artifact/projection/observation bytes and arithmetic; retained history volume is not bounded/garbage-collected. Open/replay scans full history.
8. Startup settings may establish SQLite WAL/FULL before discovering invalid retained metadata, but no ALAN epoch/history/intent changes occur before successful validation. Exclusivity is held first.
9. Existing stores are never silently migrated. Failed initialization leaves a path that future Create refuses and Open rejects; operator cleanup/retry policy is not yet a public workflow.

Ready for task-scoped independent review; implementation paused here.

## Review fix round 1 — R1 generated intent output bound

Independent review `task-1-review.md` returned specification FAIL / quality CHANGES REQUESTED for one P2 finding: generated intent JSON was not covered by any configured limit. This round addresses only R1 and the review's nonblocking receive-test evidence note. No Task 2 protocol work, commits, new dependencies, schema changes or historical research edits were made. Source is frozen again for independent review.

### Reproduction and original input preservation

Read the exact independent probe and output before implementing. A legitimate 1,163-byte Emit/End program has eight payload keys (`a` through `h`), each `Ref("text")`; the initial open projection contains one 500,000-character string of `x` under `text`, encoded as 500,092 bytes. It emits a generic `mutation` to `endpoint`, generation 1, obligation `ob`, under run worker/1. Before the fix, Reduce produced 4,000,627 bytes of intent-array JSON and Commit retained a 4,000,625-byte individual descriptor. No corruption, special runtime role or invalid identity was required.

The exact original reviewer source and output are now additionally preserved beside this report as `task-1-r1-original-probe.go.txt` and `task-1-r1-original-probe.log`; the original `/tmp/alan-task1-review-probe/...` and `/tmp/alan-task1-review-probe.log` remain intact. The original ordinary input shape is retained directly as `amplifiedOutputFixture(500000)` in `program_test.go`.

New RED run, before the enforcement change:

```text
GOCACHE=/tmp/alan-phase1d-go-build go test -v ./internal/executor \
  -run 'TestGeneratedOutput|TestReplayUsesConfiguredGeneratedOutputLimit'

TestGeneratedOutputRejectsRepeatedReferenceAmplification:
  review R1 original input: program=1163 projection=500092,
  eight references to 500000-character text
  generated output limit not enforced: bytes=4000627 err=<nil>
TestGeneratedOutputHonorsExactConfiguredJSONBoundary:
  one byte over accepted: <nil>
TestGeneratedOutputLimitPrecedesExecutionWrites/500000:
  amplified intent committed: <nil>
TestReplayUsesConfiguredGeneratedOutputLimit:
  replay ignored output limit
FAIL
```

Exact RED log: `/tmp/alan-task1-r1-red.log`. The below-bound 100,000-character counterpart already passed in this RED run; it protects successful behavior while the amplification cases establish the missing guard. Initial MaxOutputBytes type/default scaffolding was present for compilation but was not enforced during RED.

### Correction and current interface amendment

The Limits declaration in the original Task 1 report above is superseded by:

```go
type Limits struct {
    MaxProgramBytes, MaxDataBytes, MaxObservationBytes, MaxOutputBytes int
}
```

`DefaultLimits()` explicitly sets all four named fields to 1 MiB. `MaxOutputBytes` bounds the **complete generated intent-array JSON** of one reduction, including descriptor metadata, payload keys/values, array/object delimiters and commas. An instruction with no intents still has the two-byte empty array, so a configured limit less than 2 rejects it. Nonpositive output limits return ErrLimit. The bound is an explicit operational configuration, not a new research scalar bound or an implicit business policy.

`Reduce` calls `checkOutputSize` before returning any proposed reduction and before the Store execution transaction makes its first write. Store Commit and read-only replay both already call Reduce; they therefore share the enforcement. No Store/replay equality check was removed or relaxed, and no alternate unchecked dispatch/persistence path was added. There is no public dispatcher in this slice.

The size check accounts exact standard-library JSON encoding incrementally. It reserves the outer array delimiters, accounts each intent's metadata with an empty payload map, then accounts individual payload keys, colons, values and commas. It never serializes the amplified entire payload map or intent array during rejection. Raw string byte checks precede each temporary encoding; each temporary is limited by the remaining output budget in raw bytes, with at most sixfold JSON escaping plus fixed scalar/struct overhead. Bounded per-entry temporary allocation is intentional; this is not a claim that peak process memory equals MaxOutputBytes. Repeated references are copied as string descriptors in the intermediate map, not concatenated into one amplified byte buffer. Checked remaining-budget subtraction avoids size-accounting overflow.

Existing successful outputs retain identical canonical JSON bytes, IDs and semantics. The bound is checked against JSON including HTML escaping, control characters, line separators and invalid UTF-8 replacement, not just source string length. Reopening/replaying a store created with a larger intentional MaxOutputBytes must supply a sufficiently large bound; a smaller bound fails closed. Limits remain caller-supplied operational configuration, consistent with the existing input limits.

Production source changed only in `program.go`. `store.go`, `replay.go`, schema and driver files are unchanged in this fix round.

### Regression and precise receive coverage

There are now 23 top-level tests (five added in this round), with these new checks:

- The exact original eight-reference/500,000-character input returns `ErrLimit` and does not mutate the initial projection.
- The same shape with 100,000-character text succeeds and persists the independently inspected 800,625-byte descriptor.
- A configured limit equal to the actual standard-library output length succeeds; one byte less and zero fail. This includes escaped metadata and payload characters.
- An oversized Emit leaves run sequence/projection, history, intents, terminal obligations and a waiting observation unchanged. It cannot select/consume an observation by design; the waiting observation check makes that retention explicit.
- A receive with the deliberately restrictive one-byte output limit returns ErrLimit before consuming its selected observation or writing history/projection/intents. This exercises the shared guard on the consumption path.
- The original large output is allowed with an explicitly configured 5 MiB output bound; Replay succeeds with that bound and rejects the same retained history under the 1 MiB default. Exact regenerated-descriptor equality remains active.

The nonblocking review note was also addressed: `TestReceiveExactCorrelationAndTerminal` now independently changes kind, provenance, source, source generation, obligation, receiving subject and receiving generation, in addition to its existing wrong-operation case. Every mismatch requires ErrIneligible. These one-field negatives and the receive/no-consumption coverage were added after the output fix and passed on first run; they are additional coverage, not claimed as new RED cycles.

### Verification after correction

Focused RED→GREEN command output: `/tmp/alan-task1-r1-green.log` (PASS 0.282s).
Expanded output/receive tests: `/tmp/alan-task1-r1-expanded-green.log` (PASS 0.293s).

Final commands/results, with shell test exit status preserved:

```text
GOCACHE=/tmp/alan-phase1d-go-build go test ./...
ok github.com/JakeFAU/alan/internal/executor 0.285s
ok github.com/JakeFAU/alan/research/semantics (cached)

GOCACHE=/tmp/alan-phase1d-go-build go test -race ./internal/executor
ok github.com/JakeFAU/alan/internal/executor 3.419s

GOCACHE=/tmp/alan-phase1d-go-build go vet ./...
(no diagnostics, exit 0)

git diff --check
(no diagnostics)
```

Logs: `/tmp/alan-task1-r1-full.log`, `/tmp/alan-task1-r1-race.log`, `/tmp/alan-task1-r1-vet.log`. Whole-tree normal tests include unchanged historical research; the cache result is disclosed rather than presented as fresh historical execution. The source has not changed after these runs. Report/evidence preservation was completed afterward. Ready for independent review of fix round 1.
