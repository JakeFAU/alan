# A recoverable document investigator

The investigator assesses a proposition or yes/no question from two document searches chosen by the operator. An ordinary ALAN parent invokes two instances of the same worker template before it waits for either. Each worker has its own run, search query, lookup effect and, when it finds evidence, model request. The parent selects each original invocation outcome and retains its findings and unresolved questions in SQLite.

## Run the offline example

From the repository root, start the fixture in one terminal. It serves two fixed Responses proposals on loopback and prints each HTTP contact. It has no access to the store.

```sh
python3 examples/investigator/testdata/responses.py
```

In a second terminal, build the example with Go 1.27.3 and CGO. Each command starts a separate process.

```sh
work=$(mktemp -d)
go build -o "$work/investigator" ./examples/investigator
db="$work/reading-room.sqlite"
"$work/investigator" init "$db" examples/investigator/testdata/case.json
"$work/investigator" answer "$db"       # pending, no provider contacts
"$work/investigator" run "$db"          # run until quiescent
"$work/investigator" inspect "$db"      # projections, original attempts, results and charges
"$work/investigator" resume "$db"       # same stored configuration, fresh owner
"$work/investigator" replay "$db"       # verify the history without adapters
"$work/investigator" answer "$db"       # independently verify and render the parent's answer
```

The supplied case asks “Is the reading room ready to reopen?” The fixture returns conflicting interpretations. The retained answer has status `disagreement` and includes both findings.

| Worker | Model verdict | Verified source | Exact quotation |
| --- | --- | --- | --- |
| Primary | supports | `inspection`, bytes [0, 44) | The ventilation inspection passed on Monday. |
| Counter | opposes | `access`, bytes [0, 47) | The wheelchair ramp remains closed for repairs. |

The answer includes each document's full SHA-256, three unresolved questions, the two original worker handles, and a provider response and usage record for each worker. For this fixture, inspection and replay expose 76 history entries, eight intents, two worker runs and four charged effect reservations. The server received two HTTP requests across the run, resume, and read-only commands. Its token counts are synthetic.

## Supply documents and a model

Edit a copy of the case file. Supply a UTF-8 question of up to 4,096 bytes, two lookup queries of one to 32 tokens each (up to 512 bytes), and one or more named UTF-8 documents. Relative document paths resolve from the case file's directory. Initialization snapshots their contents and versions. Later commands use those retained bytes even if the original files move. The corpus limits are 64 documents, 1 MiB per document and 4 MiB in total. [Example specification](testdata/case.json)

Each lookup chooses its highest-scoring line, up to 2,048 UTF-8 bytes, using the existing deterministic token matcher. Ties use document ID and line order. The two search queries are explicit in the work specification. Their coverage determines which evidence the workers can see. Both queries may return the same span, which counts as one source. Each configuration key must appear once, using its documented lower-case spelling.

To use the official provider, select a model available to your account that supports strict structured output. Set `provider.endpoint` to `https://api.openai.com/v1/responses` or omit it. Set positive per-call token and response byte limits, initialize a **new** store and supply `OPENAI_API_KEY` through your environment. The official endpoint requires `--live` each time you run or resume. Loopback uses a fixed test credential. [Responses adapter](../../adapters/openai/openai.go)

```sh
"$work/investigator" init "$work/live.sqlite" /path/to/your-case.json
"$work/investigator" run "$work/live.sqlite" --live
"$work/investigator" answer "$work/live.sqlite"
```

The model receives the question, worker role, query and one candidate with its document version, byte range and text. It returns a scalar record containing a verdict, proposed quotation, interpretation and optional unresolved question. The Responses adapter owns the HTTPS request, secrets, response parsing and usage evidence. The local application checks proposal text shape when accepting a response and when verifying a previous owner’s retained result. It re-parses retained Responses evidence to detect contradictions between a usable HTTP body, the reported usage and IDs, and the proposed finding. A successful provider response can still yield an unsupported finding.

## Read the answer and its costs

| Status | Meaning |
| --- | --- |
| `pending` | The root has no published decision. Inspect each worker's operation, provider state and outstanding obligations. |
| `completed` | Both original workers contributed exact-source findings with matching verdicts. |
| `disagreement` | Both contributed verified evidence and their verdicts differ. The parent preserves both. |
| `partial` | Exactly one contributed verified evidence. The other worker's reason and unresolved question travel with the answer. |
| `incomplete` | Neither worker contributed a verified finding, or the root published an unsuccessful outcome. |

A quote must equal the worker's entire retrieved snippet. The parent compares that snippet with the proposed quote before admitting a finding. At read time the application checks the originally selected worker, lookup and provider identities, recomputes the document version and verifies the exact UTF-8 byte slice. It then checks the parent's completion, findings, and ordered unresolved questions against those same selected outcomes. A contradictory answer stops rendering. `model_interpretation` is the model's reading of the source. `verified_source` identifies what the document actually says.

A previous owner’s structured provider metadata must agree with the retained HTTP evidence wherever its fields can be checked. Read the raw inspection if verification of a prior result fails.

The fixed topology permits at most two invocations, two created worker runs, and one model call per worker. The retained root enforces a lifetime `max_effects` allowance (1–4, shared by lookup and model). Its `max_in_flight` setting (1–2) bounds local adapter work for the active owner. After process loss the provider can still be processing an unknown request while a fresh owner dispatches the other worker, even with a local limit of one. A refused reservation carries its reason. Previously charged reservations survive reopening. The runner also bounds each service pass and gives each external call 60 seconds.

Each model request has a `max_output_tokens` allowance and a response bound of up to 64 KiB. That leaves room in the 1 MiB retained result for both the escaped provider evidence and its parsed usage details. The application rejects a `max_total_output_tokens` budget that cannot cover both workers' per-call allowances before it creates a store. The report separates effect requests, reservation decisions, charged work, model adapter executions and possible provider contacts. Identified contacts have a matching HTTP response. A lost response can leave the contact count as a range. The adapter retains usage fields it parses from a well-formed successful HTTP envelope, including a model refusal or incomplete response. A malformed envelope or HTTP error can leave only bounded diagnostic text. When a failed response was scrubbed or clipped and its body cannot substantiate retained fields, `response_body_evidence` is `incomplete`. The answer withholds body-derived provider IDs, returned model and usage in that case. Unavailable counters and dollar estimates are `null`. The output allowance does not cap the input-token bill. [OpenAI token accounting](https://developers.openai.com/api/docs/guides/token-counting)

## Interrupt and recover

After interrupting the owner, use `resume` on the same database. An eligible model intent without a reservation can still run with its original identity. Once the runtime has reserved an ordinary model attempt, a crash can leave its external result unknown. A fresh owner preserves that obligation and its charge. An already recorded response can be settled and consumed locally. Use `answer` to distinguish a refused dispatch, a provider refusal, a malformed response, a local proposal rejection and a missing result. For an allowed model attempt, `workers[].model.client_request_id` is the outgoing correlation ID that an operator can take to the provider. It is available even if the response was lost. The separate `provider_request_id` comes from a returned HTTP response. This example has no protocol for adopting an external receipt for an ordinary Responses call.

`inspect`, `replay` and `answer` use read-only SQLite transactions. Replay reconstructs the ledger without preparing or executing adapter work. Answer verification can reconstruct a request’s expected wire bytes locally. None of these commands contacts the provider or reserves another effect. SQLite can create an empty WAL and a shared-memory index on a closed store. Read-only replay preserves database bytes and existing WAL frames. Execution assumes one active owner, local filesystem locks and functional SQLite durability.

## Build an application like this

1. Declare the scalar worker input and reply schemas. Compile its receive, request, await and reply steps into an `author.Definition`, then bind the worker's entry port with `author.NewTemplate`. [Authored programs](program.go)
2. Compile a parent with two `Invoke` steps and two `AwaitInvocation` steps. Bind each original operation, status, reason, response-presence flag and response fields to its role. The parent applies the aggregation policy and retains bounded findings and gaps.
3. Provision the parent and attach the template, retained assets, capability allowlist and resource root. The invocation deployment derives worker grants from the declared capabilities. Register the concrete lookup and model adapters when serving the store. [Deployment](main.go)
4. Read one typed execution snapshot. Look up a worker invocation by its original operation handle. Its published outcome and selecting parent reduction are explicit. Read each worker’s frozen request and retained response, then check the investigator’s quotation against the retained document bytes. [Answer verification](report.go)

Opening an investigation checks the original parent program, template, budget and any earlier model reservations. The shipped CLI serves the investigator version it can verify. Raw inspection and replay can still read other valid ALAN histories. The investigator now uses the public runtime to interpret its retained program, worker ownership, outcome delivery and actual selection. Its own code still approves the exact program and spending policy, checks application-specific role bindings and provider text, and verifies source spans. The raw `inspect` command remains available for existing tooling. The separate [execution inspector](../inspect-execution/README.md) summarizes the same store without using its storage schema. The investigator's [construction gate](construction_test.go) is a smaller, independent example of the two-worker join.

## Reproduce the qualification

Use a local filesystem with SQLite locking, Go 1.27.3 with CGO, a C compiler and loopback HTTP. No API key is needed. The process tests use a child running the normal command handler with a test-only boundary observer. They kill that process only after a matching acknowledgment, then use a separately built investigator binary to reopen the database.

```sh
go test -count=1 -v -run 'TestPublicConstructionGate|TestInvestigatorDecisions' ./examples/investigator
go test -count=1 -v -run 'TestInvestigator(CrashHistories|CompletionOrders|ProviderHistories|DurableEffectCeiling)' ./examples/investigator
go test -count=1 -tags integration -run TestInvestigatorReplayDoesNotExecuteAdapters ./examples/investigator
go test -count=1 -race -tags integration -timeout=60m ./...
```

The six crash histories distinguish an unreserved intent, a reserved attempt before transport, a POST accepted without a retained result, a retained result before settlement, one published worker outcome and both original outcomes before parent selection. The test server keeps its own contact records. The checks compare those records with ALAN's attempts, results, original identities and durable charges. Separate function-coverage processes check that replay never enters the lookup or Responses adapter. The test endpoint exercises the actual HTTP adapter. In a deployed store, provider metadata relies on the registered adapter and is not independently signed by the provider. A live provider run requires separate authorization and a budget.
