# Weather agent

This example reads modeled current air temperature at 2 metres and relative humidity for explicit WGS84 coordinates. The provider supplies the time when its model values apply and the grid cell used. Requested coordinates and grid coordinates can differ. The free Open-Meteo endpoint is for noncommercial use and needs no key. Weather data by [Open-Meteo](https://open-meteo.com/) under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). The example formats the provider time in UTC. [Forecast API](https://open-meteo.com/en/docs) · [Usage terms](https://open-meteo.com/en/terms)

## Run it

Run these commands from the repository root with Go 1.27.3, CGO and a local Unix filesystem.

```sh
work=$(mktemp -d)
go build -o "$work/weather-agent" ./examples/weather-agent
agent="$work/weather-agent"
db="$work/weather.sqlite"
"$agent" start --db "$db" --lat 26.93 --lon -80.08 --units fahrenheit \
  --endpoint https://api.open-meteo.com/v1/forecast --allowance 2
"$agent" run --db "$db"
"$agent" inspect --db "$db"
"$agent" replay --db "$db"
```

`start` provisions the original authorized read and its finite allowance in one Store creation. It makes no GET. Repeating it with the same parameters recognizes the existing deployment. A conflicting configuration is refused. Allowances range from 0 to 8. Units are `celsius` or `fahrenheit`. For controlled tests, the only alternative endpoint is a literal HTTP loopback address and port ending in `/v1/forecast`.

`run` and `resume` execute the authored program in the existing database. Add `--json` for the structured report. `inspect` returns the original operation and frozen URL, authorization, paid reservation, local dispatch admission, native response, selected observation, pending work and history. `replay` checks the retained reductions offline. The provider supplies no usage or price counter for this request, so the report leaves cost unknown.

## Kill and resume

To interrupt a real run, launch it in the background, then signal its PID while it is running. A fast provider may already have answered. The retained cut determines what happens next.

```sh
"$agent" run --db "$db" &
pid=$!
kill -KILL "$pid"
wait "$pid" || true
"$agent" inspect --db "$db"
"$agent" resume --db "$db"
```

Before a paid reservation, the original operation may still progress. After a paid reservation with no gate, no transmission was admitted and the opportunity is spent. After the gate, a missing response remains unresolved even if the endpoint accepted the GET. A retained result can be settled and selected without a network call. Reopening a terminal report returns its original timestamp and measurements.

For another physical read, approve the **numbered next observation**. Repeating the same approval is idempotent. The command records approval only. A subsequent run performs the separately charged lookup. An exhausted allowance refuses a request for another slot.

```sh
"$agent" refresh --db "$db" --read 2
"$agent" resume --db "$db"
```

The [authored program](program.go) contains the request, correlated wait, checks and terminal decision. Each read has one finite effect scope. Later reads require both their predecessor's original operation handle, sent by the authored program, and a selected local approval matching that handle. The weather CLI also verifies the named predecessor, the store-specific prepared request digest, the original result digest when one exists, and the recovery reason before registering the adapter. This approval check belongs to the weather CLI. A separately written owner that calls Store.Run directly can bypass that application policy and supply a falsely described approval. Such an owner must implement the same ingress checks or be trusted. The runtime still enforces its per-operation gate and finite resource grants. The next observation can succeed while the original stays unresolved. All slots, approvals and reported results remain in the same Store. Recovery assumes one executor at a time on a host with working filesystem locks and SQLite WAL durability. [Dispatch evidence contract](../../docs/dispatch-admissions.md)

## Controlled example

A synthetic loopback provider accepted read 1 and withheld its response. The test killed the ordinary weather executable after the server acknowledged that GET. Two fresh owners resumed the store and made no further contact. After explicit approval, read 2 returned **76.1 °F and 77% relative humidity** at the fixture time **2026-11-03 10:00 UTC**. The first operation remained open. Two distinct requests had used both permitted opportunities. [Recorded contact IDs, response hashes and causal cuts](testdata/controlled-process.jsonl)

The following command reproduces the campaign using only the local provider.

```sh
go test -count=1 -tags integration -run '^TestWeatherProcessRecovery$' -v ./examples/weather-agent
```

The campaign builds the application as a separate public Go module. A test-only owner attaches inherited-pipe acknowledgments to otherwise invisible runtime boundaries. The provider-acceptance cut and ordinary recoveries use the uninstrumented public executable.

## Live run and offline reproduction

The production executable made one authorized GET for **26.93, −80.08** on November 3, 2026. The following lines are from replaying that saved execution. Weather data by [Open-Meteo](https://open-meteo.com/) under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). The measurements are as supplied and the valid time is formatted in UTC.

```text
Read 1. Previously retained. At 2026-11-03T20:15:00Z, Open-Meteo reports 80.3 °F with 77% relative humidity for (26.93, -80.08), using grid (26.93391, -80.07091).
Location 26.93, -80.08. Units fahrenheit. Opportunities spent 1 of 1. Provider cost unknown.
Replay verified 15 committed reductions.
```

To see the original URL, HTTP body and selection without making a new request, unpack a private copy of the [live Store](testdata/live.sqlite.gz). The [evidence manifest](testdata/live.json) identifies its producer revision and the SHA-256 digests of the database, prepared request and response.

```sh
gzip -dc examples/weather-agent/testdata/live.sqlite.gz > "$work/saved.sqlite"
"$agent" inspect --db "$work/saved.sqlite"
"$agent" replay --db "$work/saved.sqlite"
```
