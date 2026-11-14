# ALAN

ALAN (Agentic Lambda Abstraction Network) is a Go library for computations that can outlive the process running them. It retains the program, its next instruction, typed data, observations and effect requests in SQLite. An application can inspect the state, resume execution in a new process, or replay the recorded decisions.

## Hello World: A Durable Weather Agent

The [weather agent](examples/weather-agent/README.md) requests current temperature and humidity for a location you supply. The authored program waits for its own weather response and retains the provider's time and reported grid coordinates. Open-Meteo supplies modeled current conditions. [Provider documentation](https://open-meteo.com/en/docs)

```sh
work=$(mktemp -d)
go build -o "$work/weather-agent" ./examples/weather-agent
"$work/weather-agent" start --db "$work/weather.sqlite" --lat 26.93 --lon -80.08 \
  --units fahrenheit --endpoint https://api.open-meteo.com/v1/forecast --allowance 2
"$work/weather-agent" run --db "$work/weather.sqlite"
"$work/weather-agent" inspect --db "$work/weather.sqlite"
"$work/weather-agent" replay --db "$work/weather.sqlite"
```

If you kill the running process, use `resume --db "$work/weather.sqlite"`. A retained response can finish the report locally. A spent opportunity with a missing response remains attached to the original request. To authorize one more lookup, use `refresh --db "$work/weather.sqlite" --read 2`, then `resume`. Inspect and replay need no provider connection. The successor selects one protected local approval. ALAN binds that decision to the original request and charges the new operation separately. [Authorize and inspect a continuation](docs/continuations.md#using-a-continuation)

Declare a finite sequence with `author.BoundedSeries(observation, author.SeriesOptions{ID: "weather", Count: 8, Inputs: inputs, First: author.ApproveAtProvision})`. ALAN derives the individual runs, grants, resource scopes and continuation relationships. Only the first observation is approved at provisioning; each later one still needs its own explicit approval. `series.Inspect(cut)` presents the ordered original results, selections, spending and unresolved work. [Bounded observation series](docs/bounded-series.md)

[Series qualification](docs/bounded-series-qualification.md) records the eight-observation run, process recovery, independent review and measured scaling.

[Execution scaling](docs/series-scaling.md) records a later optimization of repeated profile decoding: the controlled 256-operation median falls from 347.99s to 105.52s, with unchanged authorization and recovery checks. The report includes raw-evidence provenance, allocation and memory results, and the limits of the measurement.

[Marginal execution cost](docs/evidence/performance-p2/results.md) measures a separate paired campaign: reusing membership lookups within each root report reduces the 256-operation median from 104.63s to 57.75s and SQL statements from 11.77 million to 3.38 million. Cumulative allocation falls while peak process memory rises slightly. The report preserves the exact evidence, recovery checks and limits of this single-host result.

[Readiness program reuse](docs/evidence/performance-p3/decision.md) records a separate integrated-baseline comparison: the 256-operation median falls from 60.56s to 47.19s by reusing validated unbound programs within one readiness call. Per-run binding, authorization and observation checks remain. The report distinguishes allocation savings, slightly higher peak memory, exact source qualification and scope limits.

[Bounded execution scaling](docs/evidence/execution-scaling/results.md) measures guarded owner discovery, grouped full reports and a separately tested targeted continuation-basis read. The results include recovery qualification and the remaining full-report costs.

[Full report scaling](docs/evidence/report-scaling/results.md) measures report-local batching. On the frozen 256-operation series, median Execute elapsed fell from 7.031 s to 4.613 s with complete reports. The results include SQL work, allocation, peak RSS, and compatibility and recovery qualification.

## Run a computation

Use Go 1.27.3, CGO, a C compiler and a local Unix filesystem. In your application module, run `go get github.com/JakeFAU/alan/runtime`. Import `github.com/JakeFAU/alan/author` to define a program and `github.com/JakeFAU/alan/runtime` to run it. From a checkout, run the [quickstart](examples/quickstart/main.go).

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

In the quickstart, the definition declares an effect named `search`, its capability `corpus`, and its input and output schemas. Provisioning derives an adapter grant in `p.Permissions` for subject `reader`, revision 1, capability `corpus` and operation/obligation `search`. Review the grants when creating the store. To withhold this authority, omit that adapter grant before calling `runtime.Create`.

At execution time the host sets `Adapters: map[string]effects.Adapter{"corpus": lookup.New()}` in `runtime.RunOptions`. For an ordinary adapter, trusted `Prepare` code validates and serializes the request without I/O. The runner checks source authority and the retained grant, reserves an attempt and applies its local dispatch gate before calling `Execute`. That method performs one context-aware operation and returns the supplied request ID and key. `author.Request` saves the original operation ID and `author.Await` selects its result. A missing registry entry leaves the committed request available for the next run without spending an attempt.

## Guarantees and limits

- One active executor owns a store on a single host. Recovery assumes working filesystem locks, SQLite WAL and durable synchronization. For a retained program and selected observation, local reduction is deterministic. Replay checks the committed history with that reducer. It never contacts an endpoint.
- A lifecycle lookup records what existed at its decision boundary. A later invocation can legally create a worker that a controller previously found missing. Inspection, Replay and fresh-owner recovery preserve both events. [Worked example and retained witnesses](docs/lifecycle-recovery.md#worked-example)
- Revocation closes admission for covered dispatches. An already admitted call can still reach the endpoint. `Store.AwaitDispatchQuiescence` waits for covered local work and its accounting after revocation. An earlier remote mutation can still be unresolved.
- A missing ordinary adapter result stays unknown across recovery. The built-in external protocol can retry its idempotent profile with the original key when the endpoint honors it. An unknown non-idempotent request stays unresolved. The controlled reconciliation profile separates permission to read an original receipt from permission to adopt it. A published historical outcome stays fixed.
- Finite composition invokes retained templates under shared root bounds for depth, invocations, created runs and effects. The runner limits its own in-flight calls. A quiescent runner can have open runs waiting for input, a receipt or service. Distributed ownership, general dynamic spawning, PostgreSQL, history compaction and a managed provider or tool catalog remain unimplemented.
- Supported SQLite formats are 3 (ordinary), 4 (complete composition) and 5 (reconciliation), plus 6, 7 and 8 after the corresponding store retains conflicting receipt evidence. A composition created with local dispatch-admission evidence uses 9, or 10 after a conflicting receipt. Opening or replaying preserves the format. The original result and version-2 external key survive promotion. Program semantics remain `alan/1d/v1`, `alan/1e/v1` and `alan/1g/v1`. Authored scalar and collection/invocation programs retain compiler versions `alan/author/v1` and `alan/author/v2`, respectively. [Local dispatch admission](docs/dispatch-admissions.md)

## Development

This runtime was extracted from [research commit `003dc9c4`](https://github.com/JakeFAU/alan-research/commit/003dc9c45ac9b0c1f48565632f5be80f0f05beed). The research repository keeps the monograph, historical campaigns and their source-bound qualifications. See the [extraction inventory and verification commands](docs/extraction.md).

For a fix that applies to both repositories, record its originating commit and regression case. Port it manually in a separate pull request against the other repository and rerun that repository's checks. A changed research claim needs qualification against its new source.

Licensed under [Apache 2.0](LICENSE).
