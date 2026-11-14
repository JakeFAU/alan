# Weather agent

This example reads modeled current air temperature at 2 metres and relative
humidity for WGS84 coordinates. It retains the provider's valid time, formatted
in UTC, and the grid cell used for the reading. The grid coordinates can differ
from the requested coordinates. Weather data by Open-Meteo under CC BY 4.0.
[Forecast API](https://open-meteo.com/en/docs) · [Usage terms](https://open-meteo.com/en/terms) · [Data license](https://creativecommons.org/licenses/by/4.0/)

## Run one reading

Use Go 1.27.3 or later, CGO, a C compiler and a local Unix filesystem. Run these
commands from the repository root. The `run` command below can contact
Open-Meteo. To use a saved reading, go straight to
[offline reproduction](#reproduce-the-archived-reading-offline).

```sh
work=$(mktemp -d)
go build -o "$work/weather-agent" ./examples/weather-agent
agent="$work/weather-agent"
db="$work/weather.sqlite"
"$agent" start --db "$db" --lat 26.93 --lon -80.08 --units fahrenheit \
  --endpoint https://api.open-meteo.com/v1/forecast --allowance 1
"$agent" run --db "$db"
"$agent" status --db "$db"
```

`start` provisions the store without a GET. Repeating it with the same parameters
recognizes the existing deployment. Conflicting parameters are refused. Use
`celsius` or `fahrenheit`. The only alternative to the official endpoint is HTTP
on a literal loopback address and explicit port with the path `/v1/forecast`.

## Read status and continue after a restart

`status` shows the original operation ID once committed, whether a response
was retained and selected, how many opportunities were spent, and available
next commands at that read-only snapshot. If `weather-agent` is outside your
`PATH`, replace that name in the suggested command with `"$agent"` or the quoted
path to your executable. Run `"$agent" --help` for the command list.

| What the store shows | What can happen next |
| --- | --- |
| Original work or a separate read already approved | `resume` can advance it and may contact the provider. |
| Retained response with settlement, selection or publication pending | `resume` can finish that work locally using the retained response. Any other already approved work can still contact the provider. |
| Spent opportunity before local dispatch admission | The dispatch gate admitted no transmission for that operation. The opportunity is spent. |
| Local dispatch admitted and response missing | The provider response remains unknown. Reopening preserves the original operation without sending it again. |
| Completed reading | `status` shows the original time and measurements. |
| Eligible for separate approval | Optionally authorize the indicated numbered read, then `resume` to perform its own lookup. |

A successful command exit means the CLI completed the requested action. A
reading may still be pending, unavailable or unresolved. Look for the
reported measurements in `status`, or check the read's `state` and
`local_status` in JSON. `inspect`, `inspect --json` and `status --json` return the
same detailed report for the same retained state. It includes the frozen
request, authorization, reservation, dispatch evidence, retained response and
selection, pending work and history. Local contact bounds describe what the
retained cut supports. This request supplies no provider cost counter.
`replay` verifies committed reductions locally. Both reader commands and
replay work without a provider connection.

### Authorize a separate observation

Start a fresh store with `--allowance 2` or another explicit bound up to 8 if
you want capacity for later readings. The first is approved at provisioning.
The zero-allowance form keeps the refusal path. When `status` shows that read 2
is eligible, you can approve it explicitly.

```sh
"$agent" refresh --db "$db" --read 2
"$agent" resume --db "$db"
"$agent" status --db "$db"
```

`refresh` records approval without making a GET. Repeating the same approval
is idempotent. Each later read has a separate approval, operation identity and
charge. The next read must qualify under the weather policy, including finishing
pending local work on a predecessor with a retained result. A successful
successor leaves any earlier unresolved operation attached to its own request.
In detailed JSON, a protected read's `continuation` links its approval,
predecessor evidence, selected input and successor operation. For series,
`bound_exhausted` reports expenditure. Historical deployments retain their
original programs and report fields. [Authoring walkthrough](../../docs/weather-agent.md) · [Continuation contract](../../docs/continuations.md#using-a-continuation)

## Reproduce a controlled interruption

An arbitrary process kill can land before dispatch, while the provider is
responding or after local retention. Check the retained state to see what work
remains. To reproduce an exact interruption, run the loopback test. Its fixture
acknowledges the first GET and withholds the response before the test sends
SIGKILL to the ordinary weather executable.

```sh
go test -count=1 -tags integration -run '^TestWeatherProcessRecovery$/^accepted_and_lost_then_explicitly_reobserved$' -v ./examples/weather-agent
```

Fresh readers and two fresh owner resumes preserve the unknown first response
without another contact. After explicit approval, read 2 makes a distinct
request and reports the fixture reading of **76.1 °F and 77% relative humidity**
at **2026-11-03 10:00 UTC**. The full
[`TestWeatherProcessRecovery`](process_test.go) also exercises local cuts using
test-only acknowledgments. [Retained contact IDs and causal cuts](testdata/controlled-process.jsonl)

## Reproduce the archived reading offline

Run from the repository root. Unpack a private copy of the [live Store](testdata/live.sqlite.gz)
and use the read-only commands.

```sh
work=$(mktemp -d)
go build -o "$work/weather-agent" ./examples/weather-agent
agent="$work/weather-agent"
gzip -dc examples/weather-agent/testdata/live.sqlite.gz > "$work/saved.sqlite"
"$agent" status --db "$work/saved.sqlite"
"$agent" inspect --json --db "$work/saved.sqlite"
"$agent" replay --db "$work/saved.sqlite"
```

The archive comes from one authorized production GET for **26.93, −80.08** on
November 3, 2026. It records **80.3 °F and 77% relative humidity** at
**2026-11-03 20:15 UTC**, using grid **26.93391, −80.07091**. Replay verifies
15 committed reductions. Weather data by Open-Meteo under CC BY 4.0.
[Producer revision and evidence digests](testdata/live.json) · [Data license](https://creativecommons.org/licenses/by/4.0/)
