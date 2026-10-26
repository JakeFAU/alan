# Task 3 — narrow adapters and retained corpus

Status: implementation complete, frozen for review; final verification results below. No commits, subagents, live provider calls, credential discovery, application implementation or CLI changes by this implementer.

## Schema and atomic-unit design (recorded before production edits)

The design section was recorded before production changes: schema revision **3**, retaining external key version **2**, rejects previous layouts before activation; no migration. New immutable `corpus_assets`, `adapter_attempts`, and `adapter_results` tables; `adapter` joins the admissions domain. Existing settlements link adapter results to observations. Corpus commits in the provisioning transaction.

Atomic units are deliberately separate:

1. Ordinary `emit` commits the intent and any application budget updates.
2. Pure preparation validates/fixes selected non-secret configuration, exact wire request, declared input/output schemas and copied retained assets. Missing registration or preparation/configuration failure occurs before authorization/attempt reservation.
3. Existing `AuthorizeIntent` commits the first source disposition in `attempts`.
4. The adapter-attempt transaction rechecks committed request identity and static grant, then freezes prepared bytes and their SHA-256 digest. Only afterward may a worker execute.
5. Result transaction checks original intent/authorization/prepared request, candidate IDs, bounds and output schema, normalizes the runtime envelope and commits candidate/evidence/result digest. Current source epoch/status do not replace original correlation.
6. Separate settlement transaction writes admission, observation and settlement atomically.
7. Existing ordinary `receive` consumes the observation in its normal execution transaction.

An authorized attempt without a durable result is Unknown, including a crash after reservation and before any physical contact. It never repeats or reconciles automatically. A completed transport returning `unknown` leaves that attempt unresolved. A known-but-unsettled result settles without an adapter registry, credential or reinvocation. Denied attempts are deterministic local knowledge: if interrupted before their result commit, recovery constructs the same denial and settles without transport. A retained authorized result may settle after source closure or epoch renewal; settlement does not reopen the source.

`TestSchemaV2RejectedBeforeEpochRenewal` changes a store to schema/user version 2, verifies Open returns ErrVersion and directly checks the epoch is unchanged. Metadata engine semantics and existing external key derivation remain unchanged. Existing namespace-version tests needed no expectation weakening; key version remains 2. Schema-2 historical evidence requires its retained source, not this binary.

## Exact public integration interfaces

`effects` is stdlib plus the local schema package. It imports no Store/reducer/oracle. It supplies:

```go
const MaxAssets = 64
const MaxAssetBytes = 1 << 20
const MaxCorpusBytes = 4 << 20
const MaxWireBytes = 1 << 20

type Asset struct { ID, Source, Version, Content string }
func NewAsset(id, source string, content []byte) (Asset, error)
func Snapshot(id, path string) (Asset, error)
func (Asset) Check() error
func CheckAssets([]Asset) error

type Request struct {
    ID, Key, Capability, Effect string
    Input, Output schema.Retained
    Payload map[string]schema.Value
    Assets []Asset
}
type Prepared struct {
    Request Request
    Digest string
    Protocol, Endpoint, Model string
    MaxOutputTokens, MaxResponseBytes int
    Wire string
}
type Usage struct {
    InputTokens, OutputTokens, TotalTokens *int64
    Details json.RawMessage
}
type Candidate struct {
    ID, Key, Disposition string
    Payload map[string]schema.Value
    HTTPStatus int
    RequestID, ResponseID, ReturnedModel, Response string
    Usage *Usage
}
type Adapter interface {
    Prepare(Request) (Prepared, error)
    Execute(context.Context, Prepared) Candidate
}
func Copy[T any](T) T
func Freeze(Prepared) Prepared
func CheckPrepared(Prepared) error
func Result(schema.Record, Candidate) (map[string]schema.Value, string)
```

`Prepare` is trusted deployment code: its documented obligation is pure, bounded, nonblocking serialization/config validation with no I/O, goroutines, policy or Store access. The interface is **not a sandbox** and does not enforce a CPU/time bound on arbitrary implementations. It runs outside the Store mutex. Only `Execute` is a physical effect implementation, called in a bounded worker after durable authorization. Workers receive JSON-deep-copied values, not Store/oracle references. Registering arbitrary user callback implementations means trusting them to follow this contract. Runtime callback fixtures are test-only and are not provider protocol qualification.

`runtime.Provision` (existing alias) gains `Assets []effects.Asset`. Set it on the result of `author.Application.Provision()` before `runtime.Create`. `runtime.RunOptions` gains `Adapters map[string]effects.Adapter`, keyed by the exact static `author.Effect.Capability`. `runtime.RunReport` gains `UnknownAdapters []string` containing exact local intent IDs. Existing `UnresolvedIntents`, `PendingDeliveries`, and `OpenRuns` remain populated. Quiescent still means no eligible work now, not successful completion.

`runtime` adds aliases `Asset`, `AdapterAttempt`, `AdapterResult`. The latter evidence types are:

```go
type AdapterAttempt struct {
    Intent Intent
    AuthorizationID string
    Prepared effects.Prepared
    Allowed bool
    Reason string
}
type AdapterResult struct {
    Digest string
    Candidate effects.Candidate
    Values map[string]schema.Value
    Disposition string
}
```

The existing public facade signatures and all compiler APIs remain unchanged. `Kind=adapter`, capability Target, generation 1, logical effect Obligation, local request variable and exact `runtime-adapter` Await provenance follow the reviewed Task2 convention. Namespaced `Request.Key` is derived from the Store namespace; `Request.ID` is the local committed intent ID.

Output schema describes payload only. Runtime sets `valid` and `disposition` using `schema.ResultRecord`/`ValidateResult`. Invalid outputs receive typed zero defaults, including empty strings even where success enums exclude them. A provider proposing a tool cannot invoke it: the thin model adapter returns one candidate; later branches and separately committed tool requests remain ordinary application instructions.

## OpenAI Responses adapter

```go
// package adapters/openai
type Config struct {
    APIKey, Model, Endpoint string
    MaxOutputTokens, MaxResponseBytes int
}
func New(Config) (*Adapter, error)
```

Model, runtime API key and positive finite limits are required. Endpoint defaults to `https://api.openai.com/v1/responses`; only that official HTTPS endpoint or an explicitly configured literal-loopback HTTP endpoint is accepted. Userinfo, query and fragment are rejected. No model field supplies URL/header authority. `MaxOutputTokens` is 1..32768. The adapter accepts `MaxResponseBytes` 1..1MiB; Run additionally requires at most `Limits.MaxObservationBytes/8` (131072 at defaults); use 65536 or smaller for the application. Request Wire is at most 1MiB.

Input requires `instructions:string` and at least one other caller-declared flat field. The adapter checks the full retained Input schema. For exactly `{instructions, input:string}`, user content is the `input` text verbatim (the existing text convenience). For every other shape, it serializes all fields except `instructions` as one canonical JSON object of native string/int64/bool scalars, with encoding/json sorted keys and deterministic escaping. That JSON object becomes the single user message content; no schema.Value wrappers or fields are dropped. Field selection is entirely the caller's ordinary Request map, with no stage/role/application handling. Full question and lookup document/version/snippet/byte offsets can therefore reach synthesis using existing author.Var expressions, without a concatenation/formatting opcode. Instructions, all input fields, schema and exact resulting wire body are retained before dispatch. Preparation chooses the explicit model, fixed endpoint, output token bound and strict payload-only JSON Schema, then retains the exact JSON body. Request includes one user input message, `store:false`, `truncation:"disabled"`, and `text.format` with `type:"json_schema"`, stable schema-derived name, `strict:true`, all fields required and `additionalProperties:false`. There are no tools, previous-response/conversation IDs, streaming, background job or hidden loop. Local UTF-8/byte/enum/int64 validation is authoritative for the supported flat schema.

Exactly one `http.Client.Do` runs on a dedicated no-proxy, no-keepalive transport; request rewind is disabled and redirects are returned rather than followed. Context carries the owner-selected timeout. `X-Client-Request-Id` is an ASCII digest derived from the namespaced operation key (one allowed attempt per operation); it is tracing, not idempotence. Authorization is constructed only in memory.

Known dispositions: `validated`, `http_rejected`, `provider_failed`, `incomplete`, `refusal`, `protocol_invalid`, `schema_invalid`, `response_limit`, `credential_redacted`. `unknown` denotes transport uncertainty and produces no result/observation. HTTP 5xx does not prove absent remote work or known billing. Reasoning items may precede the one assistant output text; tool calls, unrecognized output kinds, ambiguous/missing text, duplicate JSON keys and invalid payloads cannot become validated success.

Candidate evidence includes bounded sanitized response string, HTTP status, x-request-id, response ID, requested/returned models separately and optional actual token counts plus usage detail JSON. Absent usage stays nil. Refusal/incomplete/error details remain in the bounded response. Metadata is bounded; oversized response ID/model becomes known protocol_invalid. Response sanitization covers direct and JSON-escaped echoes in body, metadata, nested output text and usage details. Malformed JSON containing Unicode escapes is replaced by an explicit omission marker when safe decoded redaction is unavailable. Thus Response is replay-relevant **sanitized bounded evidence**, not always byte-identical wire data. No raw request headers or transport error strings are retained. Fake credential tests are controlled loopback only.

The protocol shape follows the already retained official primary-source notes in `docs/foundational/phase1e/provider-docs.md`; no new broad research or live request was performed.

## Lookup and corpus

```go
// package adapters/lookup
const MaxSnippetBytes = 2048
func New() *Adapter
func InputSchema() schema.Record  // query string, max 512 bytes
func OutputSchema() schema.Record
// document string max 512, version string max 64, snippet string max 2048,
// start int64, end int64
```

Provision selected bytes with `effects.NewAsset` or `effects.Snapshot`. Document ID is bounded to 512 bytes, provenance Source to 4096, content to 1MiB per UTF-8 document, 64 documents and 4MiB total. Version is lowercase SHA-256 hex of exact content bytes. Source is never read by Run/Open/Replay.

Lookup is deterministic: documents sorted by ID, lines in byte order; score distinct query tokens against case-folded letter/number tokens, keeping the first highest score. Query has at most 32 tokens. Each line's candidate snippet is its first at most 2048 bytes, cut at a UTF-8 boundary. A matching result returns one exact retained document/version and byte half-open `[start,end)` span. No match gives `no_match`; empty/too-many tokens gives `query_invalid`; cancellation gives unknown under the same conservative no-repeat rule. No embeddings/network/current-file rereads. This is intentionally simple token overlap, not a semantic retrieval guarantee.

## One owner and observation boundaries

Run keeps its existing single completion channel, worker count, join/drain/accounting and fair ordinary-work opportunities. Fixture and adapter deliveries occupy the same `MaxInFlight` slots; no second scheduling loop. Fixture endpoint validation remains literal-loopback-only. Cancellation joins workers then accounts returned known candidates with the existing bounded cleanup context.

All new events carry original Subject, Generation, Epoch, committed intent Sequence, exact local Intent ID, and authorization ID in Attempt:

- `adapter.authorized` / `pre-transport`: attempt durable, before Execute.
- `adapter.response` / `pre-record`: returned candidate only, result not yet durable.
- `adapter.finished` / `transport-ended`: physical worker completion boundary.
- `adapter.result-recorded` / `pre-settlement`: result transaction committed; no settlement/ordinary consumption yet.
- `adapter.settled` / `pre-consumption`: observation/admission/settlement committed; ordinary receive has not consumed it.
- `adapter.unknown` / `unresolved`: no durable result; attempt remains uncertain.

`internal/boundary` now tracks adapter authorized/finished events in the same active-attempt map as fixture events for process barrier acknowledgments. These are observation-only boundaries, not a second policy surface. Task5 can use inherited-pipe `boundary.FromEnvironment` directly. This slice tests stops at real boundaries in-process; it does not claim the later SIGKILL qualification.

Inspect/Open/Replay validate new corpus hashes/limits, canonical attempt/result bytes, prepared/result digests, committed intent/schema/grant/original authorization binding, normalized output envelope and admission/observation/settlement links. Inspect's new checks do not run ordinary reducers. Replay accepts no registry and invokes no adapters. Digests detect independent changes; these are consistency checks, not cryptographic authentication against a party rewriting the whole database and all digests.

## Tests, evidence and actual defects

Evidence root: `.superpowers/sdd/00-plan/evidence/task-3/`.

- `red/protocol-baseline.log`, `red/runtime-baseline.log`: initial missing-package/API build failures before production implementation. These are not mislabeled as executed assertion failures.
- `red/hardening.log`: executed assertions exposed escaped credential leakage in decoded metadata/payload, accepted duplicate output keys, and Inspect accepting corrupted adapter settlement bytes. The initial original inputs remain in tests. `red/hardening-before/inspect.go` and `effects.go` preserve first checkpoints; the openai/adapters source checkpoints in that directory were overwritten by the next checkpoint, so they are **not** claimed as the complete original source for those first defects.
- `red/escaped-and-denied.log` with `red/second-hardening-before/`: escaped raw error evidence remained reversible, and a deterministic denied attempt without result was stranded. Corrected by decoded evidence redaction and deterministic denial recovery.
- `red/metadata-malformed.log` with `red/third-hardening-before/`: malformed Unicode-escaped body could retain an encoded credential; oversized returned ID was classified validated. Corrected by explicit safe omission and metadata validation/bounds.
- `red/protocol-runtime.log`, `red/asset-bounds-before.go`: asset ID had no byte bound. The real HTTP runtime/corpus tests in the same run passed; the failing asset assertion is separate.
- `red/result-digest.log`, `red/result-digest-before/`: changed raw result evidence was accepted by Inspect. Added result digest checks in recording and all offline readers.
- `red/record-input.log` and `red/record-input-before/` retain both failing assertions for the Task4 integration correction: text-only preparation could not carry a full question/evidence record. The generic native-scalar record convention above fixes this without author/runtime policy changes. `record-input-race-green.log` covers actual HTTP serialization plus lookup-to-model Run/Replay with exact question/document/version/snippet/offset checks.
- Corresponding `*-green.log` files retain passing corrections. `ownership-first.log` is the final initial ownership run; preliminary test syntax/import compilation mistakes were corrected before assertions ran and are not feature red evidence.

Tests cover committed-only dispatch; exact capability/schema/grant checks; mismatched identity and conflicting immutable results; preparation unable to reinterpret payload; worker copies; cancellation; old epoch and closed source late evidence; known result resumed without registry; unknown request never repeated; schema2 rejection before epoch renewal; changed evidence rejected by Inspect/Open/Replay; literal corpus offsets after source deletion; real HTTP errors/redirect/refusal/incomplete/malformed/extra-tool/duplicate-key/usage/model cases and credential exclusion. Callback fixtures test generic owner contracts; all provider protocol tests exercise the actual HTTP serializer/parser.

The mixed fixture+adapter capacity oracle exercises limits 1 and 2, ordinary peer progress while a fixture request is held, and faster adapter settlement before the held fixture is released. A retained negative-control source copy deliberately permits one extra worker plus one extra completion slot; the unchanged test fails `adapter bypassed shared capacity`. `negative-control.json`, `negative-capacity.diff`, `negative-runner.go.txt`, `negative-test.go.txt`, `negative-capacity.log` retain configuration/source/diff/result. An earlier mutation raised the worker bound without raising the buffer; its join blocked. That deliberately broken child was explicitly SIGQUIT-stopped; its stack/output is preserved separately in `negative-capacity-buffer-mismatch-interrupted.log`, not represented as a normal completed test. The corrected negative command has a 20-second Go test watchdog.

Some ownership cases were added after the initial broad implementation and passed on their first run; they are regression coverage, not claimed as individual red-first demonstrations. Executed defect reproductions and the deliberate shared-capacity negative control provide the retained failing evidence above.

## Verification commands

All commands use `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`. Only standard library and the already pinned SQLite dependency; no go.mod/go.sum edits.

- Focused normal: `go test -count=1 -tags integration ./effects ./adapters/... ./runtime` — PASS (`hardening-green.log`, with later covering logs).
- Ownership/schema compatibility: `go test -count=1 -tags integration -run 'TestAdapter|TestSchemaV2' ./internal/executor` — PASS (`ownership-green.log`).
- Focused race: `go test -count=1 -race -tags integration -run 'TestAdapter|TestSchemaV2|TestRealResponses|TestLookupSnapshot|TestKnownAdapter|TestUnknownAdapter|TestMissingAdapter|TestInspectRejectsAdapter|TestResponses|TestEscaped|TestDuplicate|TestUsageAnd|TestOversize|TestRetained|TestSnapshot|TestPrepared|TestAsset' ./internal/executor ./runtime ./effects ./adapters/...` — PASS (`focused-race-frozen.log`). These new tests spawn no child binaries. Existing child-process race instrumentation is unchanged.
- Full current integration regression: `go test -count=1 -tags integration ./...` — PASS (`full-integration-final.log`; another frozen run result appended below).
- `git diff --check` — PASS. Final scope formatting and source digests below.

Whole-phase final race/vet/dependency qualification, Task4 application/CLI and Task5 process-kill qualification remain parent-owned. Live gate: **NOT RUN**; no live model or credential was configured or searched for.

## Frozen handoff result

DONE. After the requested generic record-input addition:

- `go test -count=1 -race -tags integration ./adapters/openai ./runtime` — PASS (`record-input-race-green.log`), including both new record serialization/composition assertions and all covering protocol/runtime tests.
- `go test -count=1 ./...` — PASS (`full-default-frozen.log`). OpenAI/runtime integration tests correctly remain behind `-tags integration`.
- `go test -count=1 -tags integration ./...` — PASS, exit 0 (`full-integration-frozen.log`). This is the current frozen full integration regression after record mode, not the earlier pre-change run.
- Scoped `gofmt -l` — empty (`formatting.log`); `git diff --check` — PASS; `git diff -- go.mod go.sum` — empty.
- `frozen-source-sha256.json` records all 21 owned source/test file identities. `verification-environment.json` records Go/runtime environment and schema/key revisions. `red/reproduction-tests/` retains final test sources including original triggering inputs; before-checkpoint directories include `to-final.diff`.

The untouched compiler APIs remain reviewed/frozen. Task4 can use either text or generic record input as documented; no concat opcode is needed. Task5 can use the listed durable observation boundaries with exact local intent and original authorization IDs. No new API allows unknown requests to be repeated automatically, and Replay has no execution registry.

## Review fix round 1 — R1, R2, R3

Status: DONE, frozen for scoped re-review. The findings in `.superpowers/sdd/00-plan/task-3-review.md` and original probes/databases under `evidence/task-3/independent-review/` remain unchanged. No commits, subagents, new opcodes, schema changes, public-interface changes, or unrelated full-suite qualification.

The unchanged protocol probe was rerun before edits: `review-round1/probe-before.jsonl` again recovered the escaped header and eight-layer JSON credential and accepted canonical incomplete plus noncanonical completed status. The unchanged ledger probe was run with its original relative output paths inside the disposable `review-round1/ledger-before/` working directory; its newly created pristine/orphan databases and `ledger-probe-before.json` reproduce Inspect accepting the orphan while Open/Replay reject it. Original review database filenames were never reused or overwritten. The parent-provided complete pre-fix Go/module archive and manifest remain `review-round1/before-source.tar.gz` and `before-manifest.json`.

New assertion tests were written and executed before production fixes. `regressions-red.log` records actual failures for escaped headers, eight nested strings, nested object keys, usage details; status/error/output/role/content/type/text/usage aliases; and Inspect's orphan observation. The distinct payload-key preservation test is a compatibility control rather than a claimed red-first failure. `r2-red.log` separately reproduces the Unicode long-s status alias.

**R1:** Removed the previous inconsistent direct-replacement/final-serialized-candidate pass and the fail-open nested-string fallback. One `evidenceSanitizer` policy now handles returned header metadata, raw response, decoded response/model identifiers, payload keys/string values and usage detail JSON. It inspects standard JSON objects, arrays, keys, nested JSON strings and valid unquoted JSON escape fragments. The bounds are four nested string-decoding layers (unchanged, not raised), structural JSON depth 64, 65,536 inspection nodes, and `8*MaxResponseBytes + 8192` inspected bytes per Execute. Invalid UTF-8, unsupported/malformed escape decoding, duplicate JSON structure, or any exhausted inspection budget produces omitted evidence instead of the uninspected input. Secret-bearing leaves are omitted; unsafe keys are removed. Empty strings/null/absent usage detail represent omission, so no fixed redaction marker can itself echo a configured credential. Provider-response bounds are still applied after sanitization. Any altered evidence prevents a previously validated candidate from remaining validated. Known HTTP rejection/failure/incomplete classifications remain non-success and preserve safe available evidence.

This supersedes the earlier report's sanitizer/omission-marker description. It deliberately may omit benign data with unsupported escaping or excessive nesting; it does not claim arbitrary decryption or recognition of non-JSON encodings. The tests recursively decode the retained candidate data, object keys and string fragments to check recoverability, rather than merely searching for the original raw spelling. The eight-layer counterexample hits the fail-closed bound. The same policy covers the previously inconsistent header and usage paths.

The first R1 covering run (`r1-green.log`, despite its early filename) FAILED the existing `TestDuplicateOutputKeysRejected`: the conservative sanitizer changed the disposition to credential_redacted instead of schema_invalid. The test was preserved. Payload schema parsing now precedes the final altered-evidence success guard, keeping duplicate/invalid payloads schema_invalid. `r1-covering-green.log` then passes. No failure is presented as a passing run.

**R2:** Added `exactProtocolKeys`, a pre-decode guard scoped to fields consumed by the provider envelope structs: root id/model/status/error/output/usage; output type/role/content; content type/text/refusal; and usage input_tokens/output_tokens/total_tokens. It rejects noncanonical aliases in the same Unicode case-fold equivalence class before Go's case-insensitive struct decoding can overwrite canonical data. Exact duplicate checking remains separate. The payload JSON remains map-backed and is never passed through this alias guard. Regression controls validate distinct `answer` and `Answer` fields and separate composed `é` and decomposed `e\u0301` fields with their original values. Existing canonical protocol cases still pass.

**R3:** `validateAdapterEdges` now tracks fully validated forward admission/observation/settlement/result bindings and performs a reducer-free reverse pass over retained observations. Every `runtime-adapter` observation must have exactly one verified binding. This catches the review's deletion of both admission and settlement, including a consumed observation. A durable result without any observation remains valid pending evidence. The regression explicitly exercises both the orphan and the valid known-but-unsettled state. No policy reduction or adapter execution occurs in Inspect.

Unchanged probes after fixes are in `probe-after.jsonl` and `ledger-probe-after.json`. Results: canonical control `validated`, one contact; escaped header `credential_redacted`, not recoverable; nested error `http_rejected`, not recoverable; conflicting status `protocol_invalid`, one contact. The new orphan database under `ledger-after/` is rejected by Inspect/Open/Replay with `unexplained authenticated adapter observation`. The probes emit diagnostics and exit 0; the executable regressions supply assertions.

Exact round-one verification, using `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`:

1. Original `go run <absolute independent-review/probe.go>` and `go run <absolute independent-review/ledger-probe.go>` before/after — diagnostic counterexamples reproduced, then corrected; ledger working directories are the disposable `ledger-before/` and `ledger-after/` locations above.
2. `go test -count=1 -tags integration -run 'TestReviewR' ./adapters/openai ./runtime` — expected assertion FAIL before fixes (`regressions-red.log`).
3. `go test -count=1 -tags integration -run 'TestReviewR1|TestResponses|TestEscaped|TestUsageAnd|TestDuplicate|TestOversize' ./adapters/openai` — PASS after the documented intermediate classification correction (`r1-covering-green.log`).
4. `go test -count=1 -tags integration ./adapters/openai` — PASS (`r2-covering-green.log`).
5. `go test -count=1 -tags integration -run 'TestReviewR3|TestKnownAdapter|TestInspect|TestAdapter|TestUnknownAdapter|TestMissingAdapter' ./runtime ./internal/executor` — PASS (`r3-covering-green.log`).
6. `go test -count=1 -race -tags integration -run 'TestReviewR|TestResponses|TestEscaped|TestUsageAnd|TestDuplicate|TestOversize|TestRealResponses|TestLookupFields|TestKnownAdapter|TestUnknownAdapter|TestMissingAdapter|TestInspect|TestAdapter|TestSchemaV2' ./adapters/openai ./runtime ./internal/executor` — PASS (`targeted-race.log`). These selected tests introduce no child binary builds.
7. `go test -count=1 -tags integration ./adapters/openai ./runtime ./internal/executor` — PASS for all affected packages (`covering-packages.log`). No new whole-repository qualification was run in this round.
8. Scoped `gofmt -l` — empty (`formatting.log`); `git diff --check` — PASS.

`round1-fix.diff` compares all seven changed/added source and test files with the parent-provided pre-fix archive. `regression-inputs/` preserves the exact regression sources. `after-manifest.json` and refreshed `../frozen-source-sha256.json` freeze all 25 owned source/test files. Existing single-owner capacity, attempt/result/settlement boundaries, conservative Unknown profile, pure trusted Prepare contract, record input, and compiler interfaces are unchanged. Live gate remains NOT RUN. Task4/Task5 stay parent-gated pending this scoped re-review.
