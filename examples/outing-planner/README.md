# Weekend outing planner

Two retained ALAN programs plan a one-hour walk. Outing proposes the eligible
park and time pairs. Weather assesses their hourly forecasts and sends back a
batch. Outing sends one decision, which Weather consumes before completing.
Both programs run as static subjects in a single local Store.

## Run the offline walkthrough

Run the script from the repository root with Go 1.27.4 or later, CGO, a C
compiler, and Python 3.

```sh
examples/outing-planner/testdata/walkthrough.sh
```

The walkthrough builds the planner and starts a separate IPv4 loopback fixture
server. The server stays up across `start`, `run`, `resume`, `status`,
`inspect`, and `replay` subprocesses. It uses the same versioned fixture
parser as the replacement checker. The fixture's reference instant is
October 10, 2026 at 16:30 UTC, so `tomorrow` resolves to October 11 in
America/New_York. Both parks and their travel details are fictional. The
native-shaped Open-Meteo responses are fixed test data.

- **Recommendation.** Cedar Park (fictional), 11:00–12:00 local time,
  eight minutes away. The two endpoint-hour samples have WMO codes 1 and 2 and
  temperatures of 17.2 and 18.3 °C. Hemlock Park (fictional) is also assessed.
- **No match.** The same two eligible walks receive valid forecasts under an
  explicitly empty allowed-code list. Both assessments are unsuitable, the
  final decision is `no_match`, and both runs succeed.

The script prints the directory containing the executable, exact configurations,
Stores, and JSON reports. Its separate fixture server shuts down after those two
scenarios. Set `work` to the directory printed by the script to read its
saved Stores offline.

```sh
"$work/outing-planner" status --db "$work/recommended.db"
"$work/outing-planner" inspect --db "$work/recommended.db"
"$work/outing-planner" replay --db "$work/recommended.db" --json
```

For a manual fixture session, create a temporary directory and build the planner
and fixture executable. In one terminal, keep the following fixture command
running while using the planner in another. Pass the absolute paths for the
fixture and readiness file.

```sh
work=$(mktemp -d)
go build -o "$work/outing-planner" ./examples/outing-planner
go test -tags integration -c -o "$work/fixture-server" ./examples/outing-planner
ALAN_OUTING_WALKTHROUGH_FIXTURE="$PWD/examples/outing-planner/testdata/fixture.json" \
  ALAN_OUTING_WALKTHROUGH_READY="$work/endpoint" \
  "$work/fixture-server" -test.run '^TestOutingWalkthroughFixtureServer$' -test.v -test.timeout=10m
```

Use the endpoint printed to `$work/endpoint` as the configuration's `endpoint`
and copy the fixture's `reference_time` into the configuration before
provisioning. In the second terminal, set `work` to the same directory. Use a
fresh Store and the resulting configuration file.

```sh
"$work/outing-planner" start --db "$work/session.db" --config "$work/configuration.json"
"$work/outing-planner" run --db "$work/session.db"
"$work/outing-planner" resume --db "$work/session.db"
```

The endpoint is retained at `start`. Keep that fixture listener running for
work that can still dispatch. Completed history and recorded results remain
available through readers after shutdown.

## Configuration and outcomes

The version 1 [example configuration](testdata/configuration.json) gives the
shape of the inputs. Supply up to four parks and four requested one-hour
start slots, an IANA timezone, availability, park opening hours, coordinates,
one-way travel minutes, and a maximum one-way travel duration. Explicit numeric
zero is a valid travel duration. Slots must start on a whole UTC hour. A bare
local clock time requires an unambiguous local instant. Use an explicit
RFC3339 offset consistent with the named zone for an ambiguous clock time.
The `date` is an ISO calendar date or `tomorrow`, based on the retained
reference instant. When omitted, `reference_time` is captured at provisioning.
A late resume uses the same date and request hours. A recommendation whose
start has passed carries the `historical` label.
The host retains the original configuration and the normalized candidates,
including local exclusions and their reasons. The normalized rank is
one-way travel time, UTC start, then park ID. A walk must fit the opening
window, and the entire round trip must fit the availability window.

The default policy allows WMO codes 0, 1, 2, and 3 at **both** the start and end
hours. Supply `allowed_weather_codes` to choose a different subset of the
adapter's supported codes. An empty list excludes every valid forecast. The
forecast records preserve the requested location, returned grid, sample
timestamps, temperatures, codes and native response. The 50 km
nearest-grid sanity bound checks the returned location against the request.
The weather policy applies to the two retained samples and makes no
claim about conditions between those samples. [Open-Meteo forecast documentation](https://open-meteo.com/en/docs)

- `recommended` records the first eligible option in rank order with
  qualifying data and a consumed final decision.
- `no_match` records either an empty locally eligible set or exhaustive,
  validated unsuitable assessments. The empty-set path completes the
  same proposal, assessment, and decision protocol.
- `unavailable` records an incomplete evidentiary basis when no
  qualifying option was established. A malformed or refused native
  response can produce an unavailable assessment.
- `unknown_forecast` records a dispatch admission with no retained
  result. The original operation stays outstanding across resumes.
  There is no automatic redispatch and no synthetic forecast.
- `pending` covers unfinished steps, deliveries, assessments, or
  decisions. The detailed report retains exact waits and original
  operation IDs even when neither participant can advance.

`status` presents the current result. `inspect` exposes the configuration,
normalized candidates, generations, activation epochs, original operations,
effect attempts and results, observation selections, and committed
history. `replay` checks the retained reductions and projections. These
readers use the saved Store offline. `run` and `resume` can contact the
provisioned endpoint for new work.

A live configuration is optional. To make a live request, supply current
dates, a real park catalogue and schedule, and the explicit endpoint
`https://api.open-meteo.com/v1/forecast` in a new Store. Invoke `run` or
`resume` to execute forecast transport under the provisioned authority.
The example's qualification uses loopback responses.

## Check the two Weather variants

The walkthrough runs both checks. To run them separately, build the planner
as above and pass an unused directory for each check.

```sh
for variant in assess-first confirm-first; do
  "$work/outing-planner" check-replacement \
    --config examples/outing-planner/testdata/configuration.json \
    --fixture examples/outing-planner/testdata/fixture.json \
    --weather "$variant" --output-dir "$work/replacement-$variant"
done
```

Each check creates two fresh Stores, `baseline.db` and `candidate.db`,
with identical frozen inputs and a loopback endpoint. The directory also
contains the original and derived configurations, fixture, and
`replacement.json`. The assess-first control completes in both Stores.
The confirm-first candidate passes typed wiring and selects the proposal,
then waits for Outing's decision. Outing needs the assessment before it can
send that decision. The checker reports `circular_dependency` after the
bounded runner finishes with that evidenced closed pair. It verifies the
same obstruction after reopen. Ordinary inspection and replay of the
candidate show `pending` with the exact correlated waits.

This is a fixed-pair, same-Store application witness for a completion
obstruction caused by control order. The research manuscript at
`3dc2754a95d08cf2a688d6a6a970dfed2b2408b5` §6.9 studies reciprocal
service obligations and includes a separate bounded same-Store witness.
§6.10 adds an authority-attenuation experiment. §§6.18–6.20 study finite
continuation monitors with an unchanged plant. The checks here cover the two
declared Weather programs, frozen inputs, bounded runner and tested
single-host cuts. General whole-agent replacement, distributed recovery
and universal progress remain open. [Pinned manuscript](https://github.com/JakeFAU/alan-research/blob/3dc2754a95d08cf2a688d6a6a970dfed2b2408b5/paper/alan-dissertation.md#69-reciprocal-obligations-the-phase-3b1-service-boundary)

Follow the [authoring walkthrough](../../docs/outing-planner.md) from provisioning
to the final decision. The [qualification notes](QUALIFICATION.md) record the
reproducible cuts and preserved adverse findings.

### Let Weather use a helper

The `delegated` Weather program keeps the same Outing program and public
proposal → assessment → decision exchange. Weather asks one retained helper to
assess Cedar and Hemlock in order, waits for its complete original response,
then sends the public assessment. The helper owns the original forecast GETs.
The example supports zero, one or two eligible candidates and one helper.

```sh
examples/outing-planner/testdata/delegated-walkthrough.sh
```

The walkthrough uses the existing two-park configuration and local native
forecast fixtures. It prints its evidence directory, compares the direct and
delegated plans, and independently checks both complete raw Stores alongside
the CLI's actual replacement export. A second pair refuses helper creation
with allowance zero: that records `helper_refused`, makes no candidate GETs,
and leaves Outing's assessment wait visible. Refusal is not `no_match`.

For your own supported configuration, create and continue a retained session:

```sh
go run ./examples/outing-planner start --db /tmp/delegated-outing.db \
  --config examples/outing-planner/testdata/configuration.json --weather delegated
go run ./examples/outing-planner run --db /tmp/delegated-outing.db
go run ./examples/outing-planner resume --db /tmp/delegated-outing.db
go run ./examples/outing-planner status --db /tmp/delegated-outing.db
```

Those commands use the configuration's endpoint; the walkthrough supplies the
local fixture automatically. `status`, `inspect` and `replay` are contact-free
readers. If a contacted forecast has no retained result after interruption,
resume preserves `unknown_forecast` and the original debt; it does not retry
that GET or start the later candidate. The completed delegated plan pays for
one invocation, one created run, one additional depth level, and the same
number of forecast effects as direct Weather. See [QUALIFICATION.md](QUALIFICATION.md)
for the exact finite model, actual process cuts, adverse controls and limits.
