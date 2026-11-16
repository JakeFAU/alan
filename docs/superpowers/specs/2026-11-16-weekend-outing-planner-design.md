# Weekend outing planner

Status: approved for implementation by the user's “Make it so” on November 16,
2026. The concrete engineering choices below implement the reviewed proposal.

## Question and intended result

Build a small application answering: “Find a good place and time for a walk
tomorrow.” Two ordinary retained ALAN programs cooperate through
proposal → assessment → confirmation. Outing owns the bounded park catalogue,
availability, opening hours and travel constraints. Weather obtains forecasts
for the proposed places and times.

The application should demonstrate three things:

1. A working pair recommends a park and a time, with evidence explaining the
   choice, or establishes that none of the retained options meets the criteria.
2. Process interruption preserves both programs, proposals, forecast evidence
   and outstanding decisions. A fresh owner resumes from the retained cut.
3. Replacing only Weather with a confirmation-first program exposes a circular
   dependency even though the typed ports still match: “The planner needs a
   forecast; the forecast service requires the planner’s final choice.”

The research claim is a bounded application witness: changing a participant’s
ordering requirements can remove completion from this fixed protocol. It is
not a general whole-agent substitution theorem.

## Sources inspected

- Runtime main: `f30e31426c1b00fe49b5afce1f9c45d5739c49f9`.
- Research main: `3dc2754a95d08cf2a688d6a6a970dfed2b2408b5`.
- Research `AGENTS.md`: explicit state, falsifiable claims, preserved adverse
  evidence, and honest scope for recovery and composition.
- [Manuscript, pinned source](https://github.com/JakeFAU/alan-research/blob/3dc2754a95d08cf2a688d6a6a970dfed2b2408b5/paper/alan-dissertation.md):
  §§3.4/4.5 distinguish interface matching, admission, readiness and progress;
  §6.9 contains the reciprocal control-order trap and same-Store witness;
  §§6.18–6.20 qualify finite continuation-monitor results over an unchanged
  plant and leave full-agent substitution/runtime refinement open.
- Research `examples/communication/mutual` supplies an ordinary two-root
  message-order precedent. Its sender-close boundary must not be hidden.
- Runtime `author/application.go`, `author/author.go`, runtime inspection,
  `examples/weather-agent`, and `adapters/openmeteo` provide existing authoring,
  ownership, evidence and transport boundaries.
- [Open-Meteo forecast documentation](https://open-meteo.com/en/docs), inspected
  November 16, 2026: hourly weather codes and temperature, explicit hourly
  intervals, units, UTC timestamps and returned grid coordinates.

Source inspection is not a new qualification run. No tests, crash campaigns or
live provider calls have been executed for this design.

## Approaches

**Recommended: command-line example, two retained runs in one Store.** Place the
application beside `examples/weather-agent`. Each participant has its own
artifact, subject, generation, program counter, typed projection and history.
One Store supplies supported single-host ownership and durable message delivery.
This directly tests ordering and common-owner recovery through public APIs.

**Separate Stores/processes.** Each participant could have a separately owned
Store, but the application would also need a durable inter-Store transport,
correlation/admission protocol and failure qualification. That substantially
expands the research boundary and is deferred from this first application.

A browser interface can later render the same retained report. It is not needed
to answer the present composition question.

## Proposed scope and choices

These defaults are proposed by the implementer and remain reviewable:

- A Go CLI using existing standard-library flag conventions and public `author`,
  `runtime`, `schema` and `effects` packages. No new dependencies.
- One planning session per Store, two static subjects named `outing` and
  `weather`, and one final decision. No rearming, live artifact replacement,
  calendar access, geocoding, maps, bookings or dynamic discovery.
- Strict JSON configuration with at most four parks and four requested one-hour
  start slots: at most sixteen park/slot candidates. Reject oversized input
  rather than silently truncating it.
- Retain the date, IANA timezone, input reference instant, availability,
  per-park opening intervals, one-way travel minutes, maximum travel minutes
  and requested slots. Resolve “tomorrow” once at provisioning to the next
  calendar date in that timezone. Reopening never resolves it again. Late
  resume continues that date-anchored question; past slots are labelled as a
  historical outing, never presented as a new actionable recommendation.
- Slots align to UTC hourly forecast instants; walks last one hour. The entire
  walk must lie within opening hours. Availability must contain outbound travel,
  walking and return travel. Validate calendar/offset conversions; ambiguous
  local input must carry an explicit offset, and nonexistent times are rejected.
- The initial weather requirement is a configurable finite set of allowed WMO
  weather codes, defaulting to clear/mainly clear/partly cloudy/overcast
  (`0`, `1`, `2`, `3`). Both endpoint-hour samples must qualify. Temperature is
  reported as context, not an undisclosed comfort threshold. This is a declared
  hourly-sample policy, not a continuous-weather prediction.
- Rank eligible candidates by shorter travel, earlier start, then park ID.
  Select the first candidate with a validated qualifying assessment. A later
  candidate may win when an earlier one has a retained unavailable assessment;
  disclose that gap instead of claiming the earlier candidate was unsuitable.
  This is preference ordering over usable evidence, not global optimality.
- Ship labelled fictional parks and fixed-date forecasts for offline fixtures.
  Live use takes a user-supplied catalogue: opening hours and travel times are
  retained inputs, not claimed live facts. Do not infer the user’s location
  from the existing weather example.

## Retained computation boundary

Compile both participants with `author.Compile` and wire them through
`author.Application.Provision`. Supply finite resource scopes and dispatch
admission recording following the weather example; only Weather receives the
forecast capability. Each candidate has at most one original forecast opportunity.

A deterministic Go normalizer resolves dates, validates intervals, enumerates
the bounded pairs and records local eligibility with reasons. Retain original
configuration and normalized rows together; recompute their agreement when
validating the deployment. This is provisioning work, not a host continuation.
The authored Outing program selects eligible rows, sends the proposal, awaits
assessments and commits the decision. The authoring language has equality
branches but no general ordered comparison or calendar arithmetic; document
this boundary instead of introducing an unrelated language extension.

After provisioning, the host supplies adapters and drives the runtime. It must
not advance the conversation through an in-memory loop, reconstruct decisions
from current configuration files, edit SQLite directly, or inject synthetic
forecasts/confirmations to unblock the pair.

### Protocol

Every message carries a retained planning/proposal identity. Candidate IDs bind
assessments and decisions to the proposed coordinates and UTC interval. Use typed
peer ports with exact peer/generation/obligation correlation; validate payload
identities before using them.

1. Outing constructs the bounded eligible list, sends one proposal and waits for
   the matching batch assessment.
2. Healthy Weather selects the proposal, obtains one bounded hourly forecast
   per candidate, validates and retains each observation, and applies the
   weather-code policy in authored branches. It sends one assessment batch and
   then waits for the decision.
3. Outing validates the assessment batch. It retains the selected candidate and
   explanation, an exhaustive `no_match` decision, or an explicit `unavailable`
   outcome when evidence cannot settle the question. It sends that decision.
4. Weather consumes and validates the decision. Recommendation and no-match
   paths finish with both participants terminal-successful and the decision
   consumed. An assessed unavailable outcome finishes as a failed planning
   attempt; a missing external result can instead remain outstanding.

An empty eligible list still takes empty proposal → empty assessment → no-match,
so Weather is not left waiting indefinitely. Invalid retained results produce
per-candidate unavailable assessments, allowing later candidates to be checked;
a missing original response leaves its request unresolved.

The final send uses the normal runner, whose pending-delivery phase precedes the
next local reduction. At this fixed cut, all earlier assessments have been
consumed and no other messages remain. Verify decision delivery before Outing’s
terminal reduction, including reopen after decision commitment. Do not add an
unsupplied local hold merely to keep a successful sender alive. Arbitrary manual
reduction/dispatch schedules are outside this example’s qualification.

### Forecast adapter

The existing `open-meteo/current/v1` cannot answer a future-hour question. Add a
distinct versioned hourly adapter in `adapters/openmeteo`, reusing endpoint
restrictions and guarded single-GET transport where possible. Preserve current
request bytes and behavior with regression checks.

Freeze location, units and exact start/end UTC hours before dispatch. Retain the
raw native response, requested location, returned grid, sample times, units,
weather codes and temperatures. Require both requested samples exactly once,
valid bounded values and consistent units. Missing/null samples, wrong hours,
malformed JSON, HTTP refusal and invalid native evidence cannot become suitable
weather. Forecast and configured allowed codes must belong to the documented
Open-Meteo WMO subset; an unknown code is unavailable, not unsuitable. Inherit
the existing nearest-grid distance bound of 50 km and preserve both coordinates;
this is an application sanity check, not a provider promise. Extra fields may
be ignored only under the bounded-response policy.

Use loopback fixtures for normal tests and the reproducible demo. A live endpoint
must be explicit in the retained configuration; `start`, readers, replacement
checking and replay do not contact it. Preserve provider attribution. Acceptance
requires the deterministic end-to-end application and live transport support,
not a live capture; any later live run must be identified separately.

## Outcomes and interruption

Human output shows a place, local start/end with date/timezone, travel time,
forecast evidence and selection reason. JSON additionally exposes both runs,
original operations, observations, committed selections, waits and remaining
work. Distinguish:

- `recommended`: qualifying local/weather evidence and completed final decision.
- `no_match`: no locally eligible candidate, or every eligible candidate has a
  validated unsuitable assessment. Scope the conclusion to these options.
- `unavailable`: a retained error prevents the conclusion and no suitable
  candidate was established. This is not no-match.
- `pending`: local processing, delivery, assessment or a decision remains.
- `unknown_forecast`: dispatch was admitted but no response is retained. Keep
  the original operation; reopening neither resends it nor substitutes a result.
- `circular_dependency`: the supported closed pair has the evidenced cycle
  below. Never infer this solely from elapsed time or quiescence.

Proposed CLI: `start --db ... --config ...`, `run`, `resume`, `status`,
`inspect --json`, `replay`, and
`check-replacement --config ... --fixture ... --weather ... [--output-dir ...]`.
Run/resume share an owner path. Readers validate deployment and native evidence
without execution ownership. Validate again after opening before supplying
transport, following the weather example.

## Replacement check and witness

Initially support two declared variants: `assess-first` and `confirm-first`.
The latter consumes the same proposal and waits for the same decision before
issuing any forecast. Preserve typed port schemas and grants, the Outing
artifact, configuration and fixture. Unknown arbitrary programs are unsupported,
not pronounced compatible.

Run fresh candidate deployments against deterministic fixtures in isolated
Stores; never modify an existing session. A required fixture file binds a fixed
reference instant and exact latitude/longitude/UTC-range tuples to native HTTP
status, headers and response bytes. The checker hosts one literal-loopback
server for both candidate deployments and their reopen checks. Before creating
the Stores, substitute that endpoint into a derived configuration and retain
the original/derived configuration and fixture digests with the report. Missing
fixture tuples are errors; a configured live endpoint is never used as fallback.

An output directory retains candidate Stores and reports for offline inspection
and replay. A fresh check uses new Stores. The manual walkthrough supplies a
fixture-server command that outlives the planner subprocesses; the endpoint is
frozen at provisioning and never silently rebound on resume. This keeps fixture
transport available for unfinished requests during deliberate crash exercises.

Report artifact IDs, scenario, bounds and result. For the bad variant require
actual retained evidence that:

- Outing waits for Weather’s assessment and Weather waits for Outing’s decision.
- Neither receive has a matching admitted observation.
- Neither required message is pending for delivery or dispatch.
- No unresolved forecast, authorized outside producer or other declared local
  input can discharge a wait; execution did not merely exhaust its step bound.
- The artifacts match the supported programs whose next send of either missing
  message is causally after receiving the other.

Then whichever missing message were sent first would require the other to have
been received earlier. In this closed pair there is no first action that breaks
the cycle. Reopening preserves, rather than repairs, this obstruction.

Present the application explanation with program counters, wait obligations and
history references in the detailed report. Success of the healthy fixture is a
completion witness for that case, not proof that all replacements/live forecasts
are nonblocking. Resource exhaustion, unknown responses, unfinished local work
and genuine circular waiting must produce different checker results.

## Qualification and acceptance

Encode adverse expectations before implementation: confirm-first must wire
successfully but cannot emit its first missing message. Wrong-hour or invalid
data must never support that candidate or a false no-match. A different valid
candidate may still qualify. A missing original response leaves the single-batch
protocol pending, rather than supporting a completed recommendation.

1. Suitable data yields the expected candidate/evidence, both agents finish,
   and Weather consumes the exact decision. All-unsuitable and empty-eligible
   fixtures produce exhaustive no-match and finish both runs.
2. Test calendar/offset validation, park opening boundaries, travel-inclusive
   availability, stable ranking, bounds and duplicate IDs. A mixed invalid/valid
   batch must not silently erase the later valid candidate.
3. Wrong proposal/candidate identity, duplicate/missing assessments, native
   evidence mismatch, missing values and wrong hours cannot support that candidate.
   Lack of usable evidence must remain distinct from no-match.
4. Both weather definitions pass typed wiring. The checker explains the bad
   pair through retained waits and preserves the witness after reopen. Pending
   delivery, step exhaustion and missing responses control false cycle reports.
5. Kill fresh processes at acknowledged cuts after proposal commitment, after
   forecast retention before selection, and after decision commitment before
   consumption. Compare namespace, generations, artifacts, original operation
   IDs, evidence and history prefixes. Open epochs advance and the local suffix
   completes without another GET for already retained forecasts.
6. A provider-accepted/lost-response cut remains unknown through fresh-process
   resumes, preserving its original operation and contact count. No retry or
   invented assessment.
7. Offline inspection/replay preserve observations and contact counts. Build as
   an external consumer to enforce the public API boundary; run CLI commands
   in fresh processes, not only as package functions.

Use existing test-only causal barriers rather than sleeps to choose cuts. Check
candidate ranking and native fixtures independently of the compiler/reducer;
replay alone is not that oracle.

Run applicable build/module verification, targeted tests, integration process
tests, race checks and Staticcheck, plus repository suites required by the final
scope. Record commands/results only after execution. Fresh independent review
must challenge the cycle diagnosis, final-send completion, no-match epistemics
and crash evidence before delivery.

## Change boundaries and builder discretion

Expected production changes are the hourly adapter and `examples/outing-planner`;
supporting changes are tests, fixtures, a walkthrough and README link. No runtime
instruction, storage format or authority primitive is proposed. Preserve the
existing weather example and archives.

This application realizes existing bounded semantics; manuscript composition
obligations remain open. Explain that relation in example documentation. If
implementation exposes a semantic discrepancy, preserve its counterexample and
separately determine whether synchronized manuscript changes are required. Do
not silently edit research or describe this as a new Phase 3C result.

The builder may choose file boundaries, bounded field sizes, fixture layout,
formatting and test helpers. Changes to the single-Store scope, weather policy,
provisioning/computation boundary or meaning of compatibility require a visible
design update.

Deliver implementation on a narrowly scoped branch and pull request with actual
verification and limitations. Do not merge automatically.

## Design review

A fresh builder check found unresolved fixture sourcing, partial-evidence
ranking, live acceptance and forecast-admissibility choices. The clarifications
above resolve them as implementer proposals. The reviewer returned “nothing
important left” after those choices were supplied. This checks design readiness;
it is neither user approval nor implementation qualification.
