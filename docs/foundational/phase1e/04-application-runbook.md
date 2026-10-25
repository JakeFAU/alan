# Evidence assistant application runbook

## Scope and controlled result

`examples/evidence` is an importable ordinary composite, compiled by the same public `author` API and executed by the existing `runtime`. Its planner requests an ordinary retrieval worker, whose separately committed lookup returns retained evidence. A second model request includes the original question and document/version/snippet/start/end. Ordinary exact comparisons of document, version, snippet and both byte offsets validate one citation before replying. No runtime application opcode or adapter policy loop was added.

The controlled HTTP case asks **How does WAL work?** and returns **WAL records changes before updating the main database.** Its citation is the exact first line of bundled `corpus.txt`: **SQLite WAL records changes in a write-ahead log before updating the main database.** The document ID is `durability`; version is the SHA-256 of all bundled corpus bytes. Tests retain the actual version and public result in their output. This is a controlled fixture response, **not live model inference**. Exact citation checking establishes mechanical provenance, not substantive correctness of every sentence.

## Build and init

Requires Go 1.27.3, CGO and the existing SQLite dependency. No new module dependency is introduced.

```sh
export GOTOOLCHAIN=local
go build -buildvcs=false -o /tmp/alan-evidence ./cmd/alan-evidence
# Explicitly select an available model; there is deliberately no default model ID.
/tmp/alan-evidence init --db /tmp/evidence.sqlite \
  --question 'How does WAL work?' --model "$MODEL" \
  --max-model-requests 3 --max-lookup-requests 2 \
  --max-output-tokens 1024 --max-response-bytes 65536 --request-timeout 60s
```

`init` refuses an existing store and makes no provider contact. It compiles once, snapshots assets once, and stores all program, schema, prompt, corpus and selected non-secret configuration bytes. `--corpus /explicit/file.txt` selects one UTF-8 file instead of the bundled corpus. The library accepts several explicitly snapshotted assets. Source paths are provenance only; the source can subsequently disappear.

The default endpoint is `https://api.openai.com/v1/responses`. A literal loopback HTTP URL can be selected using `--endpoint` for controlled tests; arbitrary remote endpoints are rejected. Model text cannot change it. A selected alias is recorded, not treated as a model-weights pin.

## Execute, recover, inspect and replay

```sh
# Supply OPENAI_API_KEY securely through your process environment, then opt in:
/tmp/alan-evidence run --db /tmp/evidence.sqlite
/tmp/alan-evidence recover --db /tmp/evidence.sqlite
/tmp/alan-evidence inspect --db /tmp/evidence.sqlite
# Works without OPENAI_API_KEY, provider connectivity, or original source files:
/tmp/alan-evidence replay --db /tmp/evidence.sqlite
```

`recover` is an alias for `run`. Both open existing artifacts and configuration; neither compiles new application code nor rereads corpus paths. The key is runtime-only and is never a config field. The CLI accepts no model, corpus, endpoint or budget overrides during recovery. SIGINT/SIGTERM cancels the physical runner through a signal context; it does not logically cancel the programs. Command errors exit 1; invalid usage exits 2; successful command execution exits 0 even if the public result is a non-answer or the runner is quiescent with uncertainty.

The caller's public result has `disposition`, `answer`, `document`, `version`, `snippet`, `start`, `end`. Dispositions are `answer`, `clarification`, `insufficient_evidence`, `no_match`, `invalid_result`, `budget_exhausted`. Alternative messages use `answer` and empty citation strings and zero offsets. Invalid results send an explicit failure result and terminate the helper/caller with Failure. Other dispositions terminate successfully as protocol outcomes. The retrieval worker receives a skip message when no lookup is needed, so alternatives do not leave it waiting indefinitely.

Inspection derives every summary field, including run identities, subject revisions and projections, from the same read-only runtime snapshot transaction. It does not replay history; use the replay command for historical validation. Inspection includes the public caller result, selected Config, logical model/lookup requests, attempt reservations, unresolved unknown IDs, returned results and optional usage, ordinary run state, author metadata and the complete read-only runtime snapshot. Logical counts come from committed intents; `model_spent`/`lookup_spent`, selected limits and pending IDs also remain in ordinary projections. Denied attempts are counted separately and have no transport execution. Allowed attempt reservations are runtime evidence of authorized attempted work, not independent proof a provider received a request. Tests separately count HTTP contacts. Missing usage and unknown cost remain unknown; configured ceilings are not billing observations.

The one-question profile allows ceilings in [0,10], defaults 3/2; zero explicitly disables that effect and produces budget exhaustion when needed. The normal path uses two model effects and one lookup. Spare capacity does not start a repair loop. Model budget 1 stops before synthesis after retaining retrieval evidence. Token ceiling is [1,32768], response bytes [1,65536], transport timeout positive and at most 10m. Question is at most 4096 bytes, query at most 512, answer at most 8192, snippet at most 2048. Strings are UTF-8. Corpus bounds are the public effects limits: 64 assets, 1 MiB each, 4 MiB total. The runner uses 10,000 commit/dispatch limits and max two in-flight effects.

Recorded effects are reused. A reserved request without a result can have executed remotely; reopen reports Unknown and does not retry. There is no repeat/reconciliation command. Replay creates no live registry or adapter and performs no tool/model invocation. Store layout is schema 3/key 2; old schemas are rejected, and copying a store copies its namespace, not a new independent identity.

## Reusable ordinary boundary

```go
component, err := evidence.New("my-helper", "my-caller", "question-contract",
    evidence.Budgets{MaxModelRequests: 3, MaxLookupRequests: 2})
// Handle err. Add component.Instances to an author.Application.
// Give the ordinary caller a port named "helper" with value component.Port.
// Send {"question": author.String("How does WAL work?")} through that port,
// then Receive the fields from evidence.ResultSchema().
```

`RequestSchema()` and `ResultSchema()` return fresh public schemas. `New` returns static ordinary Instances and a caller-facing Port; the caller needs no private subject name or projection. Each instance handles one question. Model/lookup registry keys are `evidence.ModelCapability` and `evidence.LookupCapability`. The external-module consumer at `examples/evidence/testdata/consumer.go` instantiates two helpers, uses only their ports, and performs create/resume/offline replay in separate processes. It uses no internal imports, raw opcodes, numeric positions or SQL.

For the complete CLI application, `DefaultConfig`, `Provision(question, config, assets)`, `Run(ctx, path, apiKey)` and `Inspect(ctx, path)` provide convenience functions. `Config` is retained ordinary input on the example caller. It has no credential field. Supplying nil assets selects bundled content; an empty non-nil slice deliberately selects no corpus. Reusable custom callers may use the lower-level public runtime directly; the convenience Inspect/Run functions identify the standard CLI caller and config format.

## Qualification and live gate

```sh
GOTOOLCHAIN=local go test -count=1 -tags integration -v ./examples/evidence ./cmd/alan-evidence
GOTOOLCHAIN=local go test -count=1 -race -tags integration -v ./examples/evidence ./cmd/alan-evidence
GOTOOLCHAIN=local go test -count=1 -tags integration ./...
```

Tests use controlled loopback HTTP with the actual OpenAI serializer/parser, exercise alternate/invalid/citation/budget branches, preserve unknown pending identities across close/open, and check physical CLI and independent-module consumers. Under `-race`, their actual child executables are also built with `-race`. CLI tests delete the corpus source and stop the provider before offline replay. Provider contact counts are separate from runtime accounting. Process-kill boundary qualification is a separate phase task; these application tests do not claim SIGKILL coverage.

**Live gate: NOT RUN.** No model ID or authorized credential is configured in the qualification environment. The explicit commands above are the opt-in live path; init alone never spends money. A live run should retain its inspect output (non-secret), final disposition and citation, actual available usage, provider IDs, and offline replay of that same store. Wording need not match the controlled fixture. No fabricated model identifier or live answer is supplied.
