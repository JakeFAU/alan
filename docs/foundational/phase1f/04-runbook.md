# Investigation application runbook

This application reuses the current single-host executor, immutable programs, static grants and direct Responses/lookup adapters. Requirements remain Go1.27.3, CGO and a C compiler. There is no second scheduler or provider-owned workflow. One active executor owns a store.

## Start, inspect and reopen

Choose an available model and authorize its spend before running. Init, inspection and replay do not contact a model. The example below uses the model explicitly selected for this qualification; it is a mutable provider alias, not a pinned provider implementation.

```sh
go build -buildvcs=false -o /tmp/alan-investigate ./cmd/alan-investigate
alan_investigation_dir=$(mktemp -d)
/tmp/alan-investigate init --db "$alan_investigation_dir/run.sqlite"   --question 'For Cedar agent 3.2 production Linux deployments in window C17, which of kernels 6.1 and 5.15 is supported, and what exact trigger and target version govern rollback?'   --corpus-manifest docs/foundational/phase1f/evaluation/corpus/manifest.json   --model gpt-6.1-sol --nested --max-revisions 1   --max-output-tokens 4096 --max-response-bytes 65536 --request-timeout 120s
# OPENAI_API_KEY must be in the environment for these paid commands.
/tmp/alan-investigate run --db "$alan_investigation_dir/run.sqlite"
/tmp/alan-investigate inspect --db "$alan_investigation_dir/run.sqlite"
/tmp/alan-investigate recover --db "$alan_investigation_dir/run.sqlite"
env -u OPENAI_API_KEY /tmp/alan-investigate replay --db "$alan_investigation_dir/run.sqlite"
```

Init refuses an existing store. Omit `--nested` for a direct ordinary caller; nested adds a report participant that derives `publish` or `needs_attention` from the investigation's public disposition. `--max-revisions 0` disables the optional round. `--baseline` instead provisions the simple evidence helper with the shared Phase1F query-prompt correction with the identical corpus assets and a two-model/one-lookup ceiling; baseline and nested are mutually exclusive. The manifest snapshots each exact document separately and checks its SHA256 and byte length. Its paths are relative to the evaluation root containing `corpus/`. Runtime citation version is the content hash. Reopen never reads the original corpus paths or recompiles application source.

The investigation can make up to13 model requests and four lookups; the no-revision path at most seven and two. These are ordinary application ceilings. CLI token/response/time settings bound individual transports; they are not a dollar billing cap. The qualification controller below supplies the additional local spending gate used for measured runs. A provider request is billable even if its reply is lost.

## Spending-controlled qualification

`tools/live_control.py` wraps an already initialized store and the CLI through existing pre-transport observation barriers. Use one ledger across all authorized runs. A new user experiment can create a ledger with its explicitly authorized cap; the shipped controller currently requires the $10 profile used here. Do not reset the ledger or repeat an unknown run to evade its limit. It accepts only the selected official model/endpoint and at most4,096 output tokens, and refuses a store with prior adapter attempts.

```sh
printf '%s\n' '{"maximum_total_usd":10,"reservations":[]}' > "$alan_investigation_dir/budget.json"
python3 docs/foundational/phase1f/tools/live_control.py   --binary /tmp/alan-investigate --db "$alan_investigation_dir/run.sqlite"   --ledger "$alan_investigation_dir/budget.json" --run-id my-authorized-run   --output "$alan_investigation_dir/evidence"
```

Run this in place of the direct `run` command, before any attempt. The script reserves a conservative allowance durably before releasing each request, checks correlated source/run/epoch/sequence/attempt records, retains provider usage and unknown-cost reservations, then verifies replay without a credential or network access. It uses macOS sandbox-exec for the measured network-denial control; other platforms need an equivalent controller. The [run procedure](evaluation-run-plan.md) states the pricing/tokenization assumptions, instrumentation overhead and limits. It is a qualification tool, not a billing service. Responses with inconsistent usage cannot release an allowance.

## Reading outcomes

`View.Result` is the public caller result, or null if the obligation remains pending. `answer` is the delivered text; `initial_candidate` and `final_candidate` are diagnostic candidates. A stopped/rejected candidate is not an accepted answer. `disposition`, `review_decision`, `review_reason`, initial review fields and `revision_count` expose ordinary policy. Evidence slots e1/e2 come from the initial round; e3/e4 from the optional revision. Only helper slots marked answer contain mechanically checked source provenance. The application's reviewer is not a factual oracle; independent grading evaluates support and citation coverage separately.

`ModelRequests` and `LookupRequests` count committed intents. `Attempts` counts authorized reservations; neither is a billing count. `Results` preserves available usage and provider identifiers. `Unknown` identifies attempts without recorded outcomes; these are not automatically resent and can prevent completion. `Runs` and the full `Snapshot` expose subject/generation/epoch, program identity, retained counters, pending request IDs, histories and ledgers from one read snapshot. A successful process exit can be a quiescent wait, an explicit stop or a failed logical outcome. Inspect the records.

Physical restart renews activation authority while preserving the same logical executions, counters, pending identities and consumed observations. It creates no revision by itself. A new revision requires the ordinary coordinator to consume a valid review requesting it with budget remaining. Prior evidence remains available to the second synthesis. Revision helpers use the optional public exclusion request to avoid exact initial citation spans; both revision siblings receive the same initial set and may still select the same new span. The legacy lookup profile remains supported. Unused static helpers or the reviewer's second receive may remain waiting; no child cleanup or eventual-completion guarantee is supplied.

## Reuse from a separate Go module

`examples/investigation.New(name, caller, obligation, Budgets{MaxRevisions:1})` returns `Instances author.Application` and `Port author.Port`. Merge its instances with an ordinary caller definition that sends `RequestSchema()` and receives `ResultSchema()`. `Report` wraps the identical component and exposes `ReportResultSchema()` with one additional routing field. Callers need no internal participant name, raw instruction, SQL or child-state access. The external-consumer test is a complete compiling example; the package tests run both topologies.

```sh
GOTOOLCHAIN=local go test -count=1 ./examples/investigation ./cmd/alan-investigate
GOTOOLCHAIN=local go test -count=1 -tags integration ./internal/investigationtest
```

These commands use controlled providers, not paid inference. Process qualifications use a separate provider process, acknowledged persistence barriers and actual SIGKILL; consult the qualification report for supported cuts and limitations. Focused race instrumentation can be substantially slower than ordinary tests. Historical Phase1E reproduction is separate from current-source regression and from live answer quality.

## Optional evidence-helper exclusions

`evidence.NewExcluding` has the same component shape and result port as the legacy helper. Its `ExcludingRequestSchema` adds two explicit document/content-version/start/end tuples. All-zero tuples mean no exclusion. The existing lookup adapter recognizes `lookup.ExcludingInputSchema()` as protocol `lookup/tokens-excluding/v1`; legacy query-only behavior remains `lookup/tokens/v1`. A nonzero tuple must be well formed, and matching retained references must have safe UTF-8 boundaries. Unknown well-formed references exclude nothing. See [the exact contract](retrieval-exclusion-contract.md) before authoring requests. This optional deterministic filter does not guarantee relevance or completeness and adds no model calls.

## Reproduce the qualification without provider access

The measured source is2dacb402e50ee938d6184c14b40ba469fdcb8d91d9c30eac178b0dcb2a0fd3f8. `source-manifest.json` identifies189 contributing files; `source.tar.gz` and `archive-manifest.json` provide the complete source/test/data inputs and pinned SQLite module proxy. Reports generated after packaging remain alongside the source snapshot. Go1.27.3 and a C compiler are required; the toolchains are not bundled.

```sh
export GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1f-go-build
export GOSUMDB=off PYTHONDONTWRITEBYTECODE=1
unset OPENAI_API_KEY
python3 docs/foundational/phase1f/evidence/archive-reproduction.py
alan_qual_dir=$(python3 -c 'import json; print(json.load(open("docs/foundational/phase1f/evidence/archive-reproduction.json"))["SourceDirectory"])')
(
  cd "$alan_qual_dir"
  export GOMODCACHE="$alan_qual_dir/module-cache"
  export GOPROXY="file://$alan_qual_dir/dependencies"
  python3 docs/foundational/phase1f/tools/qualify.py default
  python3 docs/foundational/phase1f/tools/qualify.py normal
  python3 docs/foundational/phase1f/tools/qualify.py race
  python3 docs/foundational/phase1f/tools/compare.py default
  python3 docs/foundational/phase1f/tools/compare.py normal
  python3 docs/foundational/phase1f/tools/compare.py race
  go vet ./...
  go vet -tags integration ./...
  go mod verify
  gofmt -l author schema runtime effects adapters cmd internal research examples
  python3 -m unittest discover -s docs/foundational/phase1f/tools -p 'test_*.py' -v
)
```

The remaining commands run in that isolated tree using its file-only dependency proxy. The collector uses fresh test runs and retains raw ordering, actual exits and source checks. The archive command verifies every archive member, extracts into a new temporary directory, uses an initially empty module cache and the bundled file-only dependency proxy, then runs the complete default suite. It leaves that directory for inspection. These controlled tests require loopback sockets and process/SIGKILL permission; no model credential or paid inference is used. For exact earlier-source reproduction use the separately retained [baseline archive/report](evidence/baseline/report.md). Do not relabel a current-source run as historical reproduction.

For all eight frozen evaluation cases, [unseal-report.md](evaluation/unseal-report.md) gives the tested portable reconstruction command using only published inputs and original sealed archive. It runs the unchanged fixture verifier without requiring this task's ignored workspace. [Measured usefulness and cost](evaluation-results.md) links each blinded grade, adjudication, raw result and shared spending ledger. `tools/normalize.py` rebuilds normalization and opaque packets from an existing study directory; do not replace historical grades after reordering or editing outputs.

A retained live store can be inspected and replayed without live access:

```sh
go build -buildvcs=false -o /tmp/alan-investigate ./cmd/alan-investigate
/tmp/alan-investigate inspect --db docs/foundational/phase1f/evidence/live-heldout/held-01-investigation/store.sqlite
env -u OPENAI_API_KEY /tmp/alan-investigate replay --db docs/foundational/phase1f/evidence/live-heldout/held-01-investigation/store.sqlite
```

Replaying is historical verification. Running/recovering a store is live executor work and may dispatch eligible unresolved work under its protocol; do not substitute it for replay. Unknown authorized provider work is not automatically repeated. The stored provider responses and citation bytes are evidence, not instructions or a reason to contact the provider again.
