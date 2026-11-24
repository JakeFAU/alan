# ALAN

ALAN (Agentic Lambda Abstraction Network) is a Go library for computations that can outlive the process running them. It retains the program, its next instruction, typed data, observations and effect requests in SQLite. It is intended for bounded, single-host agents such as a ticket router that waits for a tool or a document investigator that must retain its evidence. An application can inspect the state, resume execution in a new process, or replay the recorded decisions.

## Build your own durable agent

The ticket router receives a priority, asks an application-owned adapter to choose a queue, and retains the answer in SQLite. Its [complete Go source](examples/ticket-routing/main.go) uses only `author`, `schema`, `effects` and `runtime`. The application defines the record fields, routing rule and success branch. ALAN retains the request identity and the result that branch consumes.

The minimum Go version is 1.27.3. Go 1.27.4 is recommended. Enable CGO and use a C compiler and a local Unix filesystem with working locks, SQLite WAL and durable synchronization. The example installs the v0.1.0 release. To test a candidate before publication, follow the [packaged-module qualification](docs/releases/consumer.md).

<!-- consumer-commands -->
```sh
mkdir ticket-routing
cd ticket-routing
export GOWORK=off
go mod init example.org/ticket-routing
go get github.com/JakeFAU/alan/runtime@v0.1.0
alan_dir=$(go list -m -f '{{.Dir}}' github.com/JakeFAU/alan)
cp "$alan_dir/examples/ticket-routing/main.go" main.go
go build -mod=readonly -o triage .
./triage start tickets.sqlite
./triage resume tickets.sqlite
./triage inspect tickets.sqlite
./triage replay tickets.sqlite
```

Each invocation starts a new process. `start` admits one ticket, checks the missing-capability error and exits with the request retained and unspent. `resume` opens the store and registers the routing desk. The adapter returns the original ID and key with the selected queue. Inspection and replay run offline. Expected application output follows.

<!-- consumer-output -->
```text
ticket accepted; routing request retained; attempts=0
routing calls=1; open runs=0
triager succeeded; queue=expedited
replay verified 5 commits, 1 intent; queue=expedited
```

Edit the local `main.go` to change the agent. `program` defines its checked input, effect and output contracts. `routingDesk.Prepare` validates the request, and `routingDesk.Execute` implements the tool. The authored branch checks the runtime's `valid` field before publishing a queue. [API and store compatibility](docs/releases/compatibility.md)

## A durable weather agent

The [weather agent](examples/weather-agent/README.md) asks Open-Meteo for modeled
current temperature and humidity. Start with one authorized reading. The
program waits for its own response, validates it and retains the provider's
time and grid coordinates in SQLite. [Provider documentation](https://open-meteo.com/en/docs)

```sh
work=$(mktemp -d)
go build -o "$work/weather-agent" ./examples/weather-agent
"$work/weather-agent" start --db "$work/weather.sqlite" --lat 26.93 --lon -80.08 \
  --units fahrenheit --endpoint https://api.open-meteo.com/v1/forecast --allowance 1
"$work/weather-agent" run --db "$work/weather.sqlite"
"$work/weather-agent" status --db "$work/weather.sqlite"
```

After a restart, use `status` to see the retained result or unresolved
operation and the available next commands. `inspect --json` shows the
detailed evidence and `replay` checks the committed reductions offline.
Use the [archived reading](examples/weather-agent/README.md#reproduce-the-archived-reading-offline)
to try these commands without a provider connection. The
[authoring walkthrough](docs/weather-agent.md) follows the same Go program
from its request to the typed evidence.

The [weekend outing planner](examples/outing-planner/README.md) uses two retained programs to choose a walk from hourly forecast evidence and reproduce a circular wait offline.

## Run a computation

From an application module, install the version shown in the ticket-routing example. Import `github.com/JakeFAU/alan/author` to define a program and `github.com/JakeFAU/alan/runtime` to run it. From a checkout, run the [quickstart](examples/quickstart/main.go).

```sh
work=$(mktemp -d)
go build -o "$work/quickstart" ./examples/quickstart
"$work/quickstart" start   "$work/reader.sqlite"
"$work/quickstart" inspect "$work/reader.sqlite"
"$work/quickstart" resume  "$work/reader.sqlite"
"$work/quickstart" replay  "$work/reader.sqlite"
```

1. `start` calls `author.Compile` on an ordinary program, wires an instance named `reader` with `author.Application.Provision`, and retains the program and a two-line document with `runtime.Create`. Its first `Store.Run` saves the query “SQLite” and waits at a local input port. The runner reports `quiescent=true open_runs=1`.
2. `inspect` uses `runtime.Inspect` to read the retained projection at sequence 1. The `programs`, `runs` and `history` tables expose the program bytes, current state and individual commits. At this point there are zero adapter attempts.
3. `resume` uses `runtime.Open` to validate the ledger and renew the open run's activation epoch. It supplies a `continue` event with `Store.AdmitInput` and calls `Store.Run` with a lookup adapter. The program consumes the correlated result and takes its success branch.
4. `replay` uses `runtime.Replay` to reconstruct the six commits and one intent in a read-only transaction. It prints `reader succeeded line="SQLite retains its state."`. Both read APIs work without a registered adapter. Run `inspect` again to see the settled projection and the single attempt and result.

Use `Store.RunProgress` when the application needs per-call step and dispatch
counts and a quiescence flag. Call `Store.Run` for the full report, or
`runtime.InspectExecution` for a typed view of retained state and outstanding
work. [Execution progress and reporting](docs/execution-progress.md)

## Run a document investigation

The [investigator example](examples/investigator/README.md) starts two retained workers, verifies their model-proposed quotations against local documents, and preserves agreement, disagreement or an unresolved call across process restarts. Run the bundled loopback provider in one terminal with `python3 examples/investigator/testdata/responses.py`, then use a second terminal.

```sh
work=$(mktemp -d)
go build -o "$work/investigator" ./examples/investigator
"$work/investigator" init "$work/case.sqlite" examples/investigator/testdata/case.json
"$work/investigator" run "$work/case.sqlite"
"$work/investigator" answer "$work/case.sqlite"
```

## Inspect an execution

An application can inspect a live, interrupted or completed store through `runtime.InspectExecution`. The returned snapshot connects each caller to its invoked members, the original effect and result, and the exact parent reduction that consumed a published outcome. A missing provider response remains attached to the original operation.

```sh
go run ./examples/inspect-execution "$work/case.sqlite"
```

The reader takes no ownership and needs no adapter registry. See the [typed inspection contract](docs/inspection.md) and the [completed, failed and interrupted investigator examples](examples/inspect-execution/README.md).

## Authorize and register an adapter

In the quickstart, the definition declares an effect named `search`, its capability `corpus`, and its input and output schemas. Provisioning derives an adapter grant in `p.Permissions` for subject `reader`, revision 1, capability `corpus` and operation/obligation `search`. Review the grants when creating the store. To withhold this authority, omit that adapter grant before calling `runtime.Create`. Bind the documents this capability needs in `p.AssetScopes`. [Capability-scoped inputs](docs/asset-scopes.md)

At execution time the host sets `Adapters: map[string]effects.Adapter{"corpus": lookup.New()}` in `runtime.RunOptions`. For an ordinary adapter, trusted `Prepare` code validates and serializes the request without I/O. The runner checks source authority and the retained grant, reserves an attempt and applies its local dispatch gate before calling `Execute`. That method performs one context-aware operation and returns the supplied request ID and key. `author.Request` saves the original operation ID and `author.Await` selects its result. A missing registry entry leaves the committed request available for the next run without spending an attempt.

## Guarantees and limits

- One active executor owns a store on a single host. With working filesystem locks, SQLite WAL and durable synchronization, a replacement process can recover committed decisions after the owner exits or is killed. Qualification exercised process interruption at specific commit boundaries. Physical power loss, storage-device failure and network filesystems are outside that fault model. For a retained program and selected observation, local reduction is deterministic. Replay checks committed decisions offline.
- A lifecycle lookup records what existed at its decision boundary. A later invocation can legally create a worker that a controller previously found missing. Inspection, Replay and fresh-owner recovery preserve both events. [Worked example and retained witnesses](docs/lifecycle-recovery.md#worked-example)
- Revocation closes admission for covered dispatches. An already admitted call can still reach the endpoint. `Store.AwaitDispatchQuiescence` waits for covered local work and its accounting after revocation. An earlier remote mutation can still be unresolved.
- An attempt reservation records a local charge. The endpoint may already have acted even if its response was lost. A missing ordinary adapter result stays unknown across recovery. The built-in external protocol can retry its idempotent profile with the original key when the endpoint honors it. An unknown non-idempotent request stays unresolved. The controlled reconciliation profile separates permission to read an original receipt from permission to adopt it. A published historical outcome stays fixed.
- Finite composition invokes retained templates under shared root bounds for depth, invocations, created runs and effects. The runner limits its own in-flight calls. A quiescent runner can have open runs waiting for input, a receipt or service. Distributed ownership, general dynamic spawning, PostgreSQL, history compaction and a managed provider or tool catalog remain unimplemented.
- The v0.1.x API contract covers documented public calls, error identities and configuration meanings. A later v0 minor release can require source changes. Before upgrading, stop the owner and back up the database and its committed WAL together. [API and store upgrade policy](docs/releases/compatibility.md)
- Supported ALAN store formats are 3 (ordinary), 4 (complete composition) and 5 (reconciliation), plus 6, 7 and 8 after the corresponding store retains conflicting receipt evidence. Compositions with local dispatch evidence use 9/10. Protected continuations use 11/12. Request authority v1 uses 13/14 and exact request binding v2 uses 15/16. In each pair, the second format retains conflicting receipt evidence. Opening or replaying preserves the format. The original result and version-2 external key survive promotion. Program semantics remain `alan/1d/v1`, `alan/1e/v1` and `alan/1g/v1`. Authored scalar and collection/invocation programs retain compiler versions `alan/author/v1` and `alan/author/v2`, respectively. [Local dispatch admission](docs/dispatch-admissions.md) · [Protected continuations](docs/continuations.md) · [Request authority and exact request binding](examples/delegation/README.md)

- New deployments can select assets per capability. Scoped formats 17–30 preserve the matching ordinary, composition, reconciliation or authority protocol and record the selected asset identities and versions. [Bind assets to capabilities](docs/asset-scopes.md) · [Store compatibility and upgrade policy](docs/releases/compatibility.md)

## Development

Performance work deferred beyond v0.1.0 includes measuring the elapsed cost of fallback observation scans and guarded candidate-list copies, and measuring creation latency and peak memory for larger provisions. The existing scheduler evidence counts work on fixed workloads. [Performance evidence and measured limits](docs/performance.md)

This runtime was extracted from [research commit `003dc9c4`](https://github.com/JakeFAU/alan-research/commit/003dc9c45ac9b0c1f48565632f5be80f0f05beed). The research repository keeps the monograph, historical campaigns and their source-bound qualifications. See the [extraction inventory and verification commands](docs/extraction.md).

For a fix that applies to both repositories, record its originating commit and regression case. Port it manually in a separate pull request against the other repository and rerun that repository's checks. A changed research claim needs qualification against its new source.

Licensed under [Apache 2.0](LICENSE).
