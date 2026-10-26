# Task 4 — ordinary evidence application and CLI

Status: DONE, source and report frozen for independent review. No commits made. No core runtime, author compiler, provider or lookup code changed. Base reviewed source: `18a751c`, branch `codex/phase1e-authoring-agent`.

## Public application boundary

Package `github.com/JakeFAU/alan/examples/evidence` imports public packages only. `New(name, caller, obligation string, b Budgets) (Component, error)` returns `Component{Instances author.Application, Port author.Port}`. `Budgets` has `MaxModelRequests, MaxLookupRequests int`; valid range is 0–10, with zero an explicit disabled-effect budget. `RequestSchema()` and `ResultSchema()` return fresh records. Public capability constants are `ModelCapability="evidence-model"`, `LookupCapability="evidence-lookup"`.

The request is `{question:string<=4096 bytes}`. The result is a closed record:

- `disposition:string` enum `answer`, `clarification`, `insufficient_evidence`, `no_match`, `invalid_result`, `budget_exhausted`.
- `answer:string<=8192`, `document:string<=512`, `version:string<=64`, `snippet:string<=2048`.
- `start:int`, `end:int`, exact byte offsets retained from lookup.

Alternative messages use answer plus empty citation strings/zero offsets. A successful answer requires nonempty model answer and ordinary exact equality for all five citation fields against retained lookup evidence. Invalid results are explicitly sent to the caller before helper and standard caller terminate Failure. Other outcomes are successful completion of the public protocol. The internal retrieval worker receives an ordinary skip request for non-lookup branches so all three programs can terminate; the caller never accesses private subjects or state.

`New` constructs an ordinary planner/synthesizer and a separate ordinary retrieval worker using only author `Receive`, `Request`, `Await`, `IfEqual`, `Send`, `Set`, labels, forward branches and terminal outcomes. Go closures construct finite steps only at authoring time. No app opcode, hidden provider policy, callback scheduler or provider tool registry was introduced. The model's `tool` string must equal literal `lookup`; it never supplies the capability or endpoint. Request schemas and grants are static. The retrieval worker emits its own separately authorized lookup. Synthesis uses the generic record mode with question/document/version/snippet/start/end, not a formatting opcode.

The external consumer at `examples/evidence/testdata/consumer.go` is compiled as module `external.example/evidenceconsumer`. It assembles an ordinary customer plus two helper instances through returned public Ports, with no internal imports, SQL or raw instructions. It executes create, resume and offline replay in separate processes. Under `-race`, the actual consumer binary is race-instrumented.

## Convenience application, retained configuration and operator API

`DefaultConfig() Config` sets endpoint `https://api.openai.com/v1/responses`, model unset, logical ceilings 3 model/2 lookup, max output tokens 1024, max response bytes 65536, request timeout 60 seconds. `Config` fields are `Model, Endpoint string`, `MaxModelRequests, MaxLookupRequests, MaxOutputTokens, MaxResponseBytes int`, `RequestTimeout time.Duration`. No secret field exists. Output-token range 1–32768, response range 1–65536 and no greater than default observation capacity/8, positive timeout <=10m. Runtime limits remain defaults; Run uses 10000 commits/dispatches and 2 in flight.

`Provision(question string, c Config, assets []effects.Asset) (runtime.Provision, error)` validates configuration, compiles once and includes immutable corpus assets. Nil assets select bundled `corpus.txt`; non-nil empty assets select no corpus. Selected custom files are snapshotted by the CLI before calling Provision. `effects.CheckAssets` supplies existing 64-document/1 MiB-document/4 MiB-corpus limits. The standard caller retains Config JSON as ordinary input along with question; all programs/templates/schemas are retained runtime artifacts. Normal execution spends exactly 2 model and 1 lookup. Surplus capacity is not a repair loop.

`Run(ctx context.Context, path, apiKey string) (runtime.RunReport, error)` reads retained config, opens existing artifacts and supplies the key only to OpenAI runtime configuration. It does not compile or read corpus sources. `Inspect(ctx context.Context, path string) (View, error)` returns `Config`, public `Result map[string]schema.Value`, `ModelRequests`, `LookupRequests`, `Attempts`, `DeniedAttempts`, `Unknown []string`, `Results []runtime.AdapterResult`, `Runs []runtime.RunState`, `Snapshot runtime.Inspection`. The helper's model_spent/model_limit/pending_plan/pending_synthesis and the worker's lookup_spent/lookup_limit/pending_lookup are ordinary persisted data, incremented atomically with corresponding Request. Operators can see those names; caller policy does not use them.

**Accounting boundary:** View.Attempts counts allowed durable attempt reservations. A reservation is not independent evidence of provider contact: process death can occur before launch. DeniedAttempts counts denied decisions separately. Logical counts derive from committed intents, independent of reservations. Controlled tests use an independent HTTP handler contact counter; Task5 will supply durable external contact records. Results retain optional actual returned usage; missing usage remains nil and unknown cost is not inferred. Unknown lists allowed requests without a retained result. The no-repeat profile persists them across physical close/open.

Convenience Run/Inspect use the standard caller/config profile; custom reusable compositions can use the lower-level public runtime. Inspect combines the runtime's coherent raw snapshot and its independently validated replay report; it has no mutation or transport authority. Replay CLI directly calls runtime.Replay with no registry or credential access.

## CLI

`cmd/alan-evidence` uses stdlib flag to preserve the repository's only-existing-SQLite dependency constraint. Commands: `init`, `run`, `recover` (run alias), `inspect`, `replay`; all require `--db`. `init` additionally requires `--question` and explicit `--model`; optional flags `--corpus`, `--endpoint`, `--max-model-requests`, `--max-lookup-requests`, `--max-output-tokens`, `--max-response-bytes`, `--request-timeout`. Init refuses overwrite and makes no HTTP contact. Run/recover obtain runtime-only `OPENAI_API_KEY` from environment and accept no changed model/corpus/limits. Inspect/replay need no key. Output is JSON; errors stderr; usage failures exit2, runtime failures exit1; success exit0 means command execution, not necessarily an answer. Signal context handles SIGINT/SIGTERM physical cancellation.

`README.md` preserves prior nested/external demos and historical links, adds the application. Full instructions are `docs/foundational/phase1e/04-application-runbook.md`. Schema3/key2, Go1.27.3 and current one-active-owner profile remain unchanged. No inferred official model ID is provided. **Live gate NOT RUN**: no configured model/credential; runbook gives explicit opt-in init/run/inspect/replay commands. Controlled model identifiers are test fixtures, not real model names.

## Actual controlled outcome

Question: `How does WAL work?`

Answer: `WAL records changes before updating the main database.`

Citation: document `durability`; version `c2d44001abf76e7220188c766c6ca6b4080dd56afbff64ac2bcc48510af84437`; snippet `SQLite WAL records changes in a write-ahead log before updating the main database.`; byte range `[0,82)`.

This is a **controlled fixture response using the real Responses serializer/parser**, not live inference. Requested model `controlled-model`, returned `controlled-returned`, response ID `controlled-response`. Normal outcome: 2 logical model, 1 logical lookup, 3 allowed attempt reservations, 2 independently observed HTTP contacts. Each model fixture returns actual usage 20 input/10 output/30 total tokens, which is retained unchanged; lookup has no provider token usage. Mechanical citation provenance does not prove every answer sentence.

The actual CLI subprocess fixture separately returned `WAL retains a durable log.` using explicitly selected corpus content, then succeeded on recovery and offline replay after deleting the original file and stopping the provider. This fixture omits usage; tests assert it stays nil.

## Tests, defects and retained evidence

Evidence directory: `.superpowers/sdd/00-plan/evidence/task-4/`.

- Test-first application and CLI baselines are `red/missing-package.log` and `red/cli.log`: missing implementation build failures, not executed assertion failures.
- Controlled application cases: answer, clarification, insufficient_evidence, wrong tool, endpoint-shaped tool, no_match, wrong document/version/snippet/start/end, empty answer, invalid typed output, refusal, incomplete, model budget1, model budget0, lookup budget0. Checks assert public outcome, logical counts/reservations and independent HTTP contacts. Negative wrong-tool/endpoint cases prove zero lookup effects; citation mismatches cannot publish answer.
- Unknown HTTP disconnect case verifies one contact, one spent model budget and stable pending identity across physical close/open; no second inference, no fabricated result, replay succeeds.
- CLI subprocess test: init/no contact, remove selected corpus, run, inspect, recover/no repeated effect, stop provider, remove key, replay. Child builds carry `-race` when parent does.
- External-module two-helper test verifies reusable public boundary and complete terminal outcomes, offline replay, two independent controlled contacts, race-built consumer children.
- Configuration tests reject implicit model, invalid ceilings, zero/oversize bounds, zero/oversize timeout and arbitrary endpoint. CLI usage/help tests check caller-facing validation.
- Denied capability test removes only the static model grant and proves 1 logical request, 0 allowed reservations, 1 denied decision and 0 HTTP contacts with explicit invalid_result.

Observed intermediate defect 1: API accepted response bound65537 despite CLI's declared65536 ceiling, because the existing runtime observation capacity/8 is larger. Original failing input/test and pre-fix source are in `response-bound-before/`, failure in `focused-1.log`; `response-bound.diff` applies the explicit application cap. Original boundary remains unchanged in the test.

Offset refinement: first test requiring fields failed against old result schema (`red/offset-contract.log`). With fields added but ordinary comparisons absent, exact snippet plus start=-1 or end=-1 incorrectly published answer (`red/offset-checks.log`). Pre-schema source and pre-comparison source are in `offsets-before/`; `offset-checks.diff` adds the generic comparisons. The initial positive test also mistakenly expected end79; independent byte counting established82. That test mistake and its original source/log are retained (`offsets-before/test-with-incorrect-length.go.txt`, `red/incorrect-test-length.log`), not described as a runtime defect.

Observed intermediate defect 2: summary counted a denied adapter ledger row as a transport attempt. Exact failed test is `red/denied-count.log`, pre-fix source/original input `denied-count-before/`, fix `denied-count.diff`, passing focused result `denied-count-green.log`. Runtime ledger semantics were correct; only application inspection counting changed.

No failed inputs were dropped to make tests green. No runtime/provider defect fix is hidden in this slice. Dedicated process-kill controller, durable provider fixture and full-phase negative controls remain Task5; these focused tests do not claim process-kill qualification.

## Future process harness interfaces

Task5 can call `Provision` once then `runtime.Create`, use `runtime.Open` on recovery, obtain Config through Inspect, and pass direct OpenAI/lookup adapters under the public capability constants to the existing Run owner. Standard operator subject names are caller, assistant, assistant/retrieval; they are not caller contract dependencies. Logical effect names are plan, synthesize, lookup. Existing runtime boundary hooks/ledgers are unchanged. Pending IDs and per-question counters are persisted ordinary data, and all selected corpus bytes are Provision.Assets. Controlled provider synthesis responses must include answer/document/version/snippet/start/end. Replay accepts only store/limits and can run without provider, credentials or source corpus.

## Final verification

All commands used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`; `go version` reports `go1.27.3 darwin/arm64`.

- `go test -count=1 -race -tags integration -v ./examples/evidence ./cmd/alan-evidence` — PASS, `focused-race-final.log`; both real child builds log `-race`. Application package 53.322s, CLI package14.251s during concurrent regression runs. No failing cases.
- `go test -count=1 -json -tags integration ./...` — PASS, 560 passing test/subtest events,14 passing packages, zero failures (`integration-final.jsonl`).
- `go test -count=1 -json ./...` — PASS,449 passing test/subtest events,12 passing packages, zero failures (`default-final.jsonl`).
- `go vet ./examples/evidence ./cmd/alan-evidence` — PASS (`vet.log`, empty).
- `git diff --check` — PASS; `gofmt -l examples/evidence cmd/alan-evidence` — empty.

Earlier full passes are preserved as `integration-before-offsets.jsonl` (557 events), `integration-before-accounting.jsonl` (559 events), and `default-before-accounting.jsonl` (449 events). They are intermediate evidence, not substituted for final verification. Final whole-phase race/vet/dependency qualification remains parent-owned. `frozen-source-sha256.json` records every owned production/test/doc file and this report. No live test was run; no commit was made.

## Review fix round 1 — R1 coherent inspection

Status: fixed and frozen for scoped re-review. This addendum supersedes the original report's description of Inspect combining a runtime snapshot and a separate replay report. The original reviewed source, original review and its counterexample remain preserved; no commits were made.

**Root cause and reproduction.** The original `evidence.Inspect` called `runtime.Inspect` and `runtime.Replay` in separate read transactions. Run state/public result could advance after the raw snapshot and its intent/attempt/result ledgers had been captured. The unchanged independent probe reproduced this before modification: `MISMATCH run=caller snapshot_status=open replay_status=open snapshot_pc=1 replay_pc=2 model_requests=1 model_spent=0 contacts=1`. Exact output and returned View are `review-round1/original-probe-before.log` and `inconsistent-view-before.json`; the original reviewer evidence is unchanged under `independent-review/`. Parent preserved the full pre-fix source in `review-round1/before-source.tar.gz`.

**Fix.** `examples/evidence/store.go` now calls `runtime.Inspect` once. The new private `inspectionRuns` decoder derives every RunState from that returned snapshot's run rows and joins subject revisions from its subject rows. Config and public Result are derived from those same decoded projections; logical counts, allowed/denied reservations, provider results, unknown IDs and the exposed Snapshot still use those same table records. No second read transaction is used. No Snapshot fields were hidden and no comparison was suppressed. Public API shapes, ordinary programs, effects, budgets and CLI transport behavior are unchanged. No core/runtime API change was needed.

**Validation boundary.** This is coherent inspection, not historical reconstruction. It relies on the existing `runtime.Inspect` snapshot and its program/adapter validation, and explicitly decodes the returned public row format. The application decoder rejects missing/null required fields, wrong JSON types, invalid run/subject identity metadata, missing subject revisions, invalid projection JSON and null projections. These checks do not prove the projections follow from the full history. Removing the separate Replay removes that extra historical integrity check from Inspect; `runtime.Replay` / CLI `replay` remains the explicit historical verification operation. `runtime.Open` retains its existing validation during execution. The application runbook now states this distinction, rather than claiming a table decode is replay.

**Regression and red/green.** New integration-tagged `examples/evidence/store_integration_test.go` exercises controlled real HTTP clarification and lookup paths while the ordinary runner progresses concurrently. An explicit HTTP entered/release handshake establishes a held model request; no sleep or production failpoint establishes progress. Every sampled View compares all run identities, programs, epochs, sequences, subject revisions and projections against its own raw Snapshot; then public Config/Result, committed logical request counts, ordinary spent counters, allowed/denied reservations, all result descriptors and unknown identities. Initial, held and final snapshots are also checked. Before the fix both subtests failed on their second post-release sample: assistant snapshot sequence18 versus summary sequence19. The unchanged first test input is retained as `regression-original.go.txt`, with `regression-red.log`. After the coherence fix the test passed (`regression-green.log`). Final race verification checked19 clarification and37 lookup progress snapshots plus initial/held/final views, and reached the expected clarification and answer outcomes with1/0 and2/1 model/lookup counts.

`examples/evidence/store_decode_test.go` specifies twelve malformed-row cases for the new decoder. The first decoder accepted six missing/null/invalid-metadata cases; `row-validation-red.log`, `row-validation-original.go.txt` and `decoder-before-row-validation.go.txt` preserve that intermediate input/source. Required-field and identity validation now rejects all twelve (`row-validation-green.log`). This is scoped decoder hardening, not a newly alleged defect in the runtime's generated snapshot rows.

**Commands and results.** Every command used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off` and the existing Go1.27.3:

1. `go run /tmp/alan-task4-review-probe.go` using an exact copy of `independent-review/inspection-probe.go.txt` — before fix, exits0 after printing MISMATCH; retained counterexample above. After final fix the unchanged probe exhausts its bound without mismatch and intentionally panics `no mismatch found in bounded probe` (child exit2; `go run` exit1). `original-probe-after.log` records this. It was **not** edited to convert its diagnostic exit semantics into a passing test, and that panic is not reported as an assertion-test PASS.
2. `go test -count=1 -tags integration -run TestInspectConcurrentProgressUsesOneCommittedView -v ./examples/evidence` — red before, green after (`regression-red.log`, `regression-green.log`).
3. `go test -count=1 -run TestInspectionRunsRejectMalformedRows -v ./examples/evidence` — red before explicit decoder validation, then all12 cases pass (`row-validation-red.log`, `row-validation-green.log`).
4. `go test -count=1 -race -tags integration -v ./examples/evidence ./cmd/alan-evidence` — scoped package regression PASS after coherence fix (`scoped-race.log`), including race-built real children. This preceded the final defensive decoder checks.
5. Final amended-code verification: `go test -count=1 -race -tags integration -run 'TestInspectConcurrentProgressUsesOneCommittedView|TestInspectionRunsRejectMalformedRows|TestCLIProcessesRetainCorpusAndReplayOffline|TestUnknownRetainsPendingIdentityAndBudget|TestDeniedCapabilityIsNotCountedAsTransportAttempt' -v ./examples/evidence ./cmd/alan-evidence` — PASS (`final-scoped-race.log`), application6.529s, CLI10.636s. Actual CLI child build includes `-race`. Unknown recovery and denied-attempt accounting remain correct.
6. `go vet ./examples/evidence ./cmd/alan-evidence`, `git diff --check`, and `gofmt -l` on the three changed/new Go files — PASS/empty (`final-vet.log`). No broad whole-phase suite was repeated; parent owns final qualification after Task5.

Changed source: `examples/evidence/store.go`; new tests `store_integration_test.go`, `store_decode_test.go`; runbook `docs/foundational/phase1e/04-application-runbook.md`; this appended report. `review-round1/fix.diff` compares these files to the preserved full before-source archive. `review-round1/after-source-sha256.json` freezes these five files and leaves the original reviewed manifest intact. Original probes/counterexamples remain untouched. Task5's separate CLI boundary-hook work is not part of this fix. Live gate remains NOT RUN.
