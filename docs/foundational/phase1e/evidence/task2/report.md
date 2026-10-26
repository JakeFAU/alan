# Task 2 report

Status: DONE, frozen for independent review. No commits made. No provider or evidence-assistant application implementation in this slice.

## Author result

An external Go module imports only `author`, `runtime`, and `schema`, compiles an ordinary caller plus two separately wired instances of the same echo component, executes two finite commits, closes, resumes in a second process without compilation, and replays in a third process. The retained result is `echo through two helpers`: three successful runs, four ordinary messages. Source: `author/testdata/consumer.go`; harness: `author/external_test.go`. No raw opcodes, numeric positions, SQL, epoch/revision manipulation, or internal imports appear in the consumer.

The compiler emits the same generic finite instruction basis into the existing Store/Run/Replay path. It adds no scheduler or callback execution. Private `Artifact` storage is a string containing canonical JSON; accessors decode/copy. Runtime provisioning takes a separate byte copy. Templates, canonical schemas and their hashes, compiler/semantic versions, qualified labels, and instruction-position names are retained inside that program JSON.

## Exact public interfaces for Task 3/4

`schema` is a pure standard-library package:

```go
type Value struct { Kind string; Number int64; Text string; Flag bool }
func Int(int64) Value
func Text(string) Value
func Bool(bool) Value
func (Value) Valid() bool

type Field struct { Kind string; MaxBytes int; Enum []string }
type Record map[string]Field
type Retained struct { ID string; Record Record }
func (Record) Check() error
func (Field) Validate(Value) error
func (Record) Validate(map[string]Value) error
func (Record) Retain() (Retained, error)
func (Retained) Check() error
func ResultRecord(payload Record) (Record, error)
func ValidateResult(payload Record, values map[string]Value) error
```

Every field is required; no additional fields, nested objects, arrays, null, floats, optional fields or general schema dialect. Kinds are exactly `string`, `int`, `bool`. Strings must be valid UTF-8 and declare `MaxBytes` in `[1, 1<<20]`; the bound is bytes, not runes. Optional finite string Enum entries are unique, bounded and canonicalized in sorted order; empty/nil Enum normalize identically. Integers are signed int64; bools/ints cannot carry string constraints. MaxFields=128. Nil/empty Record canonicalize to `{}`. IDs are lowercase SHA-256 hex of JSON encoding the canonical Record. Shared validation is not an independent oracle.

```go
const CompilerVersion = "alan/author/v1"
const SemanticVersion = "alan/1e/v1"
const MaxInstructions = 10000
const MaxNesting = 64

type Definition struct {
    Name string
    Inputs, Outputs schema.Record
    Values map[string]schema.Value
    Templates map[string]string
    Effects map[string]Effect
    Ports map[string]Port
    Steps []Step
}
type Effect struct { Capability string; Input, Output schema.Record }
type Port struct {
    Peer, Obligation string
    Send, Receive schema.Record
    Local bool
}
type Expr struct { /* private */ }
type Step struct { /* private */ }
func Var(string) Expr
func String(string) Expr
func Integer(int64) Expr
func Boolean(bool) Expr
func Offset(name string, delta int64) Expr
func Set(name string, value Expr) Step
func Add(name string, a, b Expr) Step
func Label(string) Step
func Goto(string) Step
func IfEqual(a, b Expr, yes, no []Step) Step
func Repeat(count int, body ...Step) Step
func Success() Step
func Failure() Step
func Send(port string, values map[string]Expr) Step
func Receive(port string, bindings map[string]string) Step
func Request(request, effect string, values, updates map[string]Expr) Step
func Await(effect, request string, bindings map[string]string) Step
func Compile(Definition) (Artifact, error)
type Artifact struct { /* private */ }
func (Artifact) ID() string
func (Artifact) Bytes() []byte
func (Artifact) Metadata() Metadata

type Instance struct { Artifact Artifact; Inputs map[string]schema.Value }
type Application map[string]Instance
func (Application) Provision() (runtime.Provision, error)
```

Inputs are initial named values validated at provisioning, not implicit inbound messages. `Values` are emitted constant assignments and cannot shadow initial inputs. Outputs name values required at Success; Failure need not produce them. Receive/Await binding maps are `payload field -> variable`; callers may select a subset. Sequential composition is the Steps slice. IfEqual compares exact scalar values. Goto is forward-only; labels are lexically scoped to each body, with nearest enclosing-label resolution. Repeat expands statically, gives each iteration its own lexical label scope, supports zero/empty repetition, and can exit to a later enclosing label. Negative counts/backward branches are rejected. Limits also bound expansion visits to 100,000 and compiled artifact JSON to the existing 1 MiB default program limit. Reachable fallthrough without an explicit terminal is rejected. Unreachable instructions remain encoded; definite-assignment checks consider reachable predecessor paths. Zero-repeat bodies do not execute or contribute references.

`Metadata`, `EffectMetadata`, and `PortMetadata` are public aliases of executor evidence structs (consumers can name the aliases without importing internal code):

```go
type Metadata struct {
    Compiler, Name string
    Inputs, Outputs schema.Retained
    Templates map[string]string
    Positions []string
    Labels map[string]int
    Effects map[string]EffectMetadata
    Ports map[string]PortMetadata
    Receives map[int]schema.Retained
}
type EffectMetadata struct { Capability string; Input, Output schema.Retained }
type PortMetadata struct {
    Peer, Obligation string
    Send, Receive schema.Retained
    Local, SendUsed, ReceiveUsed bool
}
```

Positions and Labels are inspection metadata; callers never supply numeric execution positions. All Metadata returns are independently decoded copies. Definition and returned-map mutations cannot alter Artifact bytes or identity. Invalid UTF-8 template strings are rejected instead of silently changed by JSON encoding.

`runtime` delegates functions `DefaultLimits`, `Create(ctx,path,Provision,Limits) (*Store,error)`, `Open(ctx,path,Limits) (*Store,error)`, `Inspect(ctx,path,Limits) (Inspection,error)`, and `Replay(ctx,path,Limits) (ReplayReport,error)` to executor. It aliases Store, Limits, Artifact, Provision, Subject, InitialRun, Permission, RunID, Projection, RunState, Authority, Input, AdmissionResult, RunOptions, RunReport, Inspection, ReplayReport and Value. Existing Store methods include Run, AdmitInput, Close and Namespace. Error aliases: ErrLimit, ErrVersion, ErrCorrupt, ErrConflict, ErrStale, ErrLocked, ErrRunActive. This is deliberately a provisional facade; the Store alias also exposes existing lower-level executor methods. No new transport/registry method is claimed.

## Frozen effect/result convention

Effect declaration keys are logical effect names. Capability is a fixed deployment registry key, not a model-supplied URL or endpoint. Request emits:

- Intent.Kind=`adapter`, Target=Capability, Generation=1, Obligation=effect declaration key.
- Payload is exactly the declared Input record. Compile checks field names/types and literal constraints; Task 3 must validate dynamic payload constraints before dispatch as required by the adapter contract.
- Request's first argument names a variable receiving the committed **local intent ID**, not the namespaced external key. Updates execute atomically in that same generic emit reduction; Offset supports the budget increment without a policy opcode.
- Await requires a definitely assigned request identity from the same declared effect. Its Match is Kind=`adapter`, Provenance=`runtime-adapter`, Source=Capability, SourceGeneration=1, Obligation=effect declaration key, Operation=request variable.
- Derived adapter Permission: Kind=`adapter`, Source=component subject, Target=Capability, Operation=effect declaration key, Revision=1, Obligation=effect declaration key.

Effect.Output is **provider/tool payload only**. The two reserved names `valid` and `disposition` may not appear in it. `schema.ResultRecord` adds runtime-owned `valid: bool` and `disposition: string` bounded to 128 bytes, while retaining all payload fields. These envelope fields are available to Await bindings. Max payload fields for effects is therefore 126. The runtime/adapter must populate validation fields itself; the model cannot supply them as part of its output schema.

`schema.ValidateResult`: when valid=true, validate exact declared payload types, enums and bounds. When valid=false, require exactly typed zero defaults (`""`, `0`, `false`) for every payload field; success-only enums are deliberately not enforced on those defaults. The ordinary program must branch on valid/disposition before treating payload fields as success. Disposition classification strings are reserved for Task 3, constrained here only by type/byte limit. Reducer uses ValidateResult for authored adapter receives; ordinary receives use full Record.Validate.

Task 3 must implement actual dispatch/authorization/result admission and may refine the trusted admission integration with explicit compatibility review. Current v2 admissions CHECK does **not** yet contain the adapter domain. No adapter attempt or observation is manufactured by Task 2; current runner leaves adapter intents unresolved. Static adapter permissions are accepted for future provisioning. The namespaced external key remains `ExternalKey(Store.Namespace(), localIntentID)` from Task 1; adapters must not send the local request variable as a globally unique provider key.

## Static component/wiring convention and runtime validation

Each Application key is a static subject; generation=revision=1, owner=`owner:<name>`, open initial projection with validated Inputs. Programs deduplicate by artifact ID; all maps and derived grants emit in sorted order. Send-used ports derive ordinary message grants. A matching used receive port on the peer must name the sender and same obligation and exact canonical schema identity. Missing peers or incompatible request/reply records fail Provision. `Local=true` requires Peer=`local`, no Send schema, and derives an operator input grant only when received. Grants for declared effects are explicit even when the definition does not request them.

Ordinary Send/Receive uses generation 1, fixed named obligation, empty Operation and existing ordinary-message/local-input provenance. This is one bounded question per provisioned component, not a long-lived multiplexed service. Reusable components are author-time Go functions returning artifacts for explicit public peer/obligation wiring; the external smoke instantiates one helper twice with different peers/obligations. No child-generation inference or private-state inspection is needed.

Compile guarantees scalar kinds, definitely assigned references on all reachable predecessors, binding field names/unique destinations, request-effect correlation, matching branch kinds, bounded forward control, and literal payload constraints. It does not statically prove arbitrary referenced-string bounds or enums; runtime enforces those at initial-state validation, authored input/message admission, receive, and successful final output. Ordinary invalid ingress is retained as rejected `schema_invalid` with no observation, and replay rechecks accepted author admissions. Payload field selection does not bypass validation of unbound fields. Port receive contract ambiguities are rejected.

Schema checks are confined to authored programs. Legacy `alan/1d/v1` programs retain old scalar representation, instruction behavior, and byte identities. Program.Author is optional with `omitempty`; a fixed legacy byte fixture verifies its absence. New programs use `alan/1e/v1` with mandatory compiler metadata; unknown semantics/compiler versions fail Create/Open/Replay explicitly with ErrVersion, and legacy semantics plus author metadata are rejected. Store schema/key version 2 and metadata engine semantics are unchanged. Artifact envelope and encoded program Semantics must agree.

## Files and evidence

New production: `author/author.go`, `compile.go`, `application.go`; `schema/schema.go`; `runtime/runtime.go`; `internal/executor/author_metadata.go`.

Narrow existing production edits: executor `program.go` (shared scalar alias, optional metadata, author receive/output guards), `store.go` (supported semantic versions, initial schema guard, reserved adapter grants), `messages.go` (author schema admission/replay checks), `replay.go` (retained initial schema guard).

New tests: author compile/application/external tests plus race-build flag files and testdata consumer; schema tests; integration-tagged runtime facade tests; executor author_metadata tests. No existing tests weakened. No README/full-doc edits by this implementer.

Evidence root: `.superpowers/sdd/00-plan/evidence/task-2/`.

- `red/compiler.log`: initial absent author/schema package build failure; preserves missing public API baseline. These were compile failures, not falsely described as executed assertions.
- `red/envelope.log`: Await rejected runtime disposition when output schema was payload-only; fixed with separate runtime envelope.
- `red/application.log`: absent static Application and ValidateResult APIs.
- `red/runtime.log`: wrong type/enum/extra-field inputs were admitted and overlong final output accepted; exact input cases remain in runtime tests and copied reproduction source.
- `red/terminal-branches.log`: two-terminal IfEqual generated an invalid unreachable jump; lowering now omits unnecessary join jumps.
- `red/metadata.log`: missing receive metadata and ambiguous same-channel schemas were accepted; recovery/compile validation hardened.
- `red/canonical-schema.log`: nil/empty equivalent enum constraints produced different schema IDs; normalized.
- `red/reproduction-tests/`: retained current reproducer tests with the original defect inputs. Initial missing-package baseline is commit `270e857`; later logs describe intermediate implementation defects, not historical baseline defects. Full pre-fix source snapshots/diffs were not saved for those intermediate defects; retained scope is the failure logs and the unchanged original triggering input cases within the copied tests, plus final source. No unavailable pre-fix source is claimed.
- `focused-1.log`, `focused-2.log`, `focused-3.log`, `full-1.log`, `default-final.jsonl`, `integration-final.jsonl`, `external-race.log`: verification outputs.
- `frozen-source-sha256.json`: owned source/test file digests at handoff.

All commands used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`, source baseline Go 1.27.3, stdlib plus existing SQLite dependency only. Focused commands: `go test ./author ./schema`; `go test -tags integration ./runtime ./author ./schema`; `go test -tags integration ./author ./runtime ./schema ./internal/executor`; `go test -count=1 -tags integration -v ./author ./runtime ./schema` — PASS after documented fixes. Full first regression: `go test -count=1 -tags integration ./...` — PASS (`full-1.log`). Final `go test -count=1 -json ./...` — PASS, 405 passing test/subtest events and zero failures (runtime has no default-tag tests). Final `go test -count=1 -json -tags integration ./...` — PASS, 414 passing test/subtest events and zero failures; all nine packages with tests passed, cmd/alan-resource has no tests. `go test -count=1 -race -tags integration -run TestExternalConsumerCompileRunReopenReplay -v ./author` — PASS, logged child `go build ... -race .`, three separately race-instrumented processes. `git diff --check` — PASS. Final whole-phase race/vet/dependency qualification remains parent-owned.

## Limits and handoff

No adapter/provider/corpus implementation, transport retries, dynamic subjects, lifecycle-authoring builders, source-level SDK stability promise, or general nested data interpreter. Current adapter definitions can be compiled and provisioned but cannot execute a transport until Task 3. The external smoke uses ordinary messages precisely so its compile/run/reopen/replay result is functional now.

Flat records, static expansion and explicit schema/validation versioning are the narrow choices requested by the integration notes. Integration tests are build-tagged per the Go testing skill, so `-tags integration` is required to run the public ingress and external-module smoke; default full regression is preserved separately. Race-enabled author tests pass `-race` to their child binary build. The shared schema validators are consistency machinery, not an independent validation oracle.

## Review fix round 1 — R1 and R2

Status: DONE, code frozen for scoped re-review. The independent findings in `.superpowers/sdd/00-plan/task-2-review.md` remain intact. No commits, subagents, unrelated implementation, or repeated full-suite qualification.

The unchanged public probes were rerun before edits. `review-round1/probe-before.log` again reports `expected=succeeded actual=failed` for both scenarios. Original emitted artifacts and closed databases are retained in `probe-before/`, in addition to the reviewer's original evidence. The first reproduction command used the wrong output-directory location; its path error is retained separately as `probe-before-path-error.log`, and the corrected unchanged probe then reproduced both actual defects. The pre-fix complete Go/module archive and manifest supplied by the parent remain at `before-source.tar.gz` and `before-manifest.json`. Original regression input sources are copied under `red-tests/` before production fixes; `regressions-red.log` records executed assertion failures, separate from the earlier missing-API build failures.

**R1:** `compiler.reachesBoundary` now traverses the lowered body control flow and resolved label targets to determine whether its lexical end can be reached. A then-arm join is omitted only when that boundary is unreachable. Jumps to trailing labels, nested trailing labels, and repeated-body exits therefore pass through a jump over the else arm. An unresolved label outside the body is an explicit exit, not fallthrough. The existing both-terminal-arms case remains valid without an unreachable out-of-range join target. This replaces the faulty last-opcode test; it does not change reducer behavior or rewrite stored programs.

Executable regression `TestRuntimeConditionalLexicalContinuations` covers the exact forward-Goto/trailing-Label input, nested trailing labels, repeat exits at the arm boundary, both terminal arms, and selecting the else arm. Each case runs through public Create/Run, checks the live durable projection via Inspect, closes, and asserts offline Replay's status. The original review probe now prints `trailing-label expected=succeeded actual=succeeded`; its corrected artifact/database are in `probe-after/`.

**R2:** One shared `schema.CheckIdentifier(name string) error` policy rejects empty or invalid UTF-8 identifiers without normalization. Definition.Name remains optional; a nonempty Name uses the policy. Schema field names, metadata effect names/capabilities, port names/peers/obligations, template keys, initial value names, and Application subject keys now use that policy. `author/checkStepIdentifiers` checks every emitted step's variable destination/reference, label/Goto, effect/port reference, request/Await name, payload/update key, binding field/destination, and expression reference before lowering/JSON. This includes emitted unreachable instructions; zero-repeat bodies are omitted and emit no identifiers. Derived subject owners and grant identities are built only from validated names. Provision input keys are checked against the retained closed input schema before any descriptor encoding. No Unicode normalization or lossy replacement is performed.

`TestCompileRejectsInvalidUTF8Identifiers` includes the exact distinct-byte `0xfe`/`0xff` probe and the identifier surfaces above; `TestApplicationRejectsInvalidUTF8Identity` covers provisioned subject names. The unchanged review probe now returns the useful compile error `steps/0:set: invalid UTF-8 identifier "\xfe"`, so no artifact/store is created for it. Valid Unicode is explicitly retained: the runtime regression executes/replays Japanese subject/port/obligation/input/output/template/capability names and distinct composed `é` versus decomposed `e\u0301` variable names with separate values. Another compiler test checks Unicode Request/Await/effect/update identifiers.

Additive public interface only:

```go
// schema
func CheckIdentifier(name string) error // nonempty valid UTF-8; no normalization
```

All existing builders and runtime facade signatures are unchanged. Old retained raw programs retain their instruction semantics; compiler rejection prevents newly authored invalid identifiers from collapsing during JSON encoding.

Covering changed production: `author/compile.go`, new `author/identifiers.go`, `author/application.go`, `schema/schema.go`, `internal/executor/author_metadata.go`. Covering tests: `author/compile_test.go`, `author/application_test.go`, `runtime/runtime_test.go`. `round1-fix.diff` compares these exact files with the archived pre-fix sources, including the new helper. `after-manifest.json` and refreshed `../frozen-source-sha256.json` freeze owned sources/tests.

Verification used the same offline/local Go environment stated above:

1. Unchanged `go run . <probe-before-directory>` from the retained external review module — reproduced both defects; stdout, artifacts, databases retained.
2. `go test -count=1 -tags integration -run 'TestRuntimeConditionalLexicalContinuations|TestCompileRejectsInvalidUTF8Identifiers|TestApplicationRejectsInvalidUTF8Identity' ./author ./runtime` — expected assertion FAIL before production edits (`regressions-red.log`), then PASS (`regressions-green.log`).
3. `go test -count=1 -tags integration -v ./author ./runtime ./schema` — PASS (`covering-tests.log`), including the external-module compile/run/reopen/replay test and valid-Unicode preservation tests.
4. Unchanged `go run . <probe-after-directory>` from the same retained external review module — lexical probe now succeeds in Run/Replay and invalid identifiers are rejected (`probe-after.log`). This diagnostic prints results; the executable regression suite provides assertions.
5. `go test -count=1 -run 'TestLegacyProgramEncodingUnchanged|TestAuthorMetadataRejectsMissingReceiveContract|TestAuthorUnknownVersionsRejectedOnRecovery' -v ./internal/executor` — PASS (`metadata-regression.log`).
6. `git diff --check` and scoped `gofmt -l` — PASS/no output. Whole-phase full/race/vet qualification remains with parent; no claim of a new full-suite run in this fix round.
