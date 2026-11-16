# Weekend outing planner qualification

## Fresh-process recovery

```sh
go test -count=1 -timeout=60m -tags integration ./examples/outing-planner \
  -run 'TestOutingProcessRecovery|TestPlanner(CalendarBounds|CelsiusOfflineEvidence|CLIStartRecognizesRetainedDeployment)' -v
go test -count=1 -timeout=60m -race -tags integration ./examples/outing-planner \
  -run 'TestOutingProcessRecovery|TestPlanner(CalendarBounds|CelsiusOfflineEvidence|CLIStartRecognizesRetainedDeployment)' -v
examples/outing-planner/testdata/walkthrough.sh
```

The recovery test builds all example production Go files in an independent
module with `GOPROXY=off`, a replace directive to the checkout, and Go's
ordinary import restrictions. The subprocesses for `start`, `resume`,
`status`, `inspect`, and `replay` use that binary. To stop an executing
owner at a retained transaction boundary, a test subprocess runs the same CLI
with the existing inherited-pipe observer. The controller checks the
acknowledged Store, process, event, phase, run, epoch and sequence against
the cut, then verifies that the owner exited by SIGKILL. For the lost
response, the independent HTTP handler acknowledges the original GET
with its response withheld before the controller kills the ordinary owner.

The following cuts were observed in the targeted integration run. Logical
operation coordinates in the table are scoped to each scenario's Store.
Contact counts come from the separate loopback endpoint. It checks the
request tuple and records the decoded operation ID and client request
header. The test binds both to the retained dispatch attempt and requires
distinct IDs for different contacts.

| Acknowledged cut | Retained evidence at interruption | Contacts at cut → after resume | Result |
| --- | --- | --- | --- |
| Outing proposal commit, epoch 2, sequence 18 | Original proposal exists before admission or selection | 0 → 2 | Exhaustive `no_match` under the empty allowed-code policy. Weather selects Outing's decision at sequence 38. |
| Weather result record before settlement, epoch 2, sequence 9 | Cedar's first result is durable before its observation enters the inbox | 1 → 2 | `recommended`. Recovery settles and selects the same result, then makes the new Hemlock GET. |
| Outing decision commit, epoch 2, sequence 56 | Exact decision retained before admission or selection | 2 → 2 | On resume, Weather's accepted admission and inbox observation are visible while Outing is still open at sequence 57, with its terminal reduction staged at epoch 3, sequence 58. After release, both runs succeed and Weather selects that same observation at sequence 86. |
| Weather next request commit, epoch 2, sequence 47 | Cedar's forecast is already retained and selected. Hemlock's original intent has no attempt or admission yet. | 1 → 2 | `recommended`. Only the pending Hemlock intent dispatches on resume. |
| Endpoint accepts and withholds Cedar response for Weather generation 1, sequence 9, ordinal 0 | Dispatch admission and original attempt exist without result, settlement or observation | 1 → 1 → 1 | `unknown_forecast` through two fresh resumes. |
| Confirmation-first Weather reaches its decision wait, epoch 2, sequence 6 | Weather selected the sole proposal. Neither receive has a matching observation. | 0 → 0 | Both runs remain open with the same reciprocal waits. Ordinary readers report `pending`. |

Fresh owners preserve the namespace, original generations and artifact IDs,
provisioning, operation records and reduction history prefix. They renew
open epochs. The test verifies both terminal-successful runs and Weather's
specific decision selection against the selecting history step on completion.
Fresh readers agree on the retained cut. Replay returns the same projections
and does not change the snapshot or endpoint contact count. The separate
walkthrough observed four GETs across its two ordinary sessions, then checked
the assess-first control (two GETs per deployment) and the confirm-first
candidate (zero GETs and the same closed wait after reopen).

## Preserved adverse findings

Independent review exposed the following failures before the process
qualification. Each entry records the observed failure and the retained
regression.

| Reviewed revision and observed failure | Correction and regression |
| --- | --- |
| `f6e40ee`: omitted or null one-way travel or maximum travel decoded as known zero. Focused RED accepted all four malformed inputs. | `83e8fe0` requires both values to be present and non-null. `TestPlannerRequiresExplicitTravelMinutes` rejects each malformed input and accepts explicit zero. |
| `f6e40ee`: Weather consumed a correctly correlated confirmation with invalid proposal or candidate identity and made two GETs before rejecting it. | `93c63fa` moves those identity checks ahead of forecast dispatch. `TestPlannerConfirmationFirstRejectsMismatchesBeforeForecast` requires each altered confirmation to be selected and rejected with zero forecast intents and zero endpoint contacts. |
| `23b779f`: the report exposed a committed forecast's effect identity before adapter preparation but omitted the actual intent payload. Focused RED found the missing Weather operation at sequence 9. | `ffabff0` exposes all original intents and compares them on checker reopen. `TestPlannerReportsCommittedForecastRequest` checks exact ID, run, sequence, ordinal, capability, obligation and Celsius request payload, including both UTC hours, before any attempt exists. |

The additional review coverage checks actual Celsius request units, both
temperatures and both weather codes against the fixture and validates native
evidence after the endpoint closes. Calendar coverage includes exact opening
and closing boundaries, an early closing, and a return trip falling outside
availability when departure fits. A counting loopback fixture checks that
initial, repeated and conflicting starts make zero contacts.

Independent Task 4 review at `0eeb0ec` returned REJECT on spec compliance
and task quality. Eventual consumption and terminal success could also follow authorization
before Outing closed and admission after closure, because authorization and
admission commit separately. The process regression now pauses the resumed
owner at the staged terminal reduction, independently reads the exact
accepted admission and Weather inbox observation while Outing is still open,
then releases the owner and checks consumption of that same observation.
The review also found that reader-to-reader JSON comparisons could hide a
shared omission. Each offline reader's complete intents, effects and
observations now agree with the public snapshot.

The process witnesses preserve observed behavior for these local cuts, native
fixtures and supported Weather programs. They leave arbitrary schedules,
distributed ownership, live provider behavior and general substitution
outside the experiment. The manuscript's reciprocal control-order work is in
§6.9. Its §§6.18–6.20 concern finite continuation monitors over an unchanged
plant. [Pinned manuscript at `3dc2754a95d08cf2a688d6a6a970dfed2b2408b5`](https://github.com/JakeFAU/alan-research/blob/3dc2754a95d08cf2a688d6a6a970dfed2b2408b5/paper/alan-dissertation.md#69-reciprocal-obligations-the-phase-3b1-service-boundary)

## Final provider-code review

Independent whole-branch review at `3f78b61118f2c8910a82f8b436044f1f6c3af71a`
found that the hourly adapter and implementation plan omitted WMO 97 (heavy
thunderstorm). The omission classified valid hourly responses as unavailable and
rejected explicit policies containing 97. The correction adds 97 to the supported
set. The default walking policy allows only codes 0–3. [Open-Meteo forecast API documentation](https://open-meteo.com/en/docs)

```sh
go test -count=1 -v ./adapters/openmeteo ./examples/outing-planner \
  -run 'TestHourly(RequestAndEvidence|WeatherCodeSet)$|TestPlanner(HeavyThunderstorm|WeatherPolicySchemaBound)$'
```

Before the correction this command exited 1. The adapter marked the native-shaped
HTTP 200 fixture unavailable, the default authored planner returned `unavailable`
instead of `no_match`, explicit `[2,97]` was rejected, and the code count was 28.
After the correction the same command exited 0. Under the default policy both
validated assessments are unsuitable and the planner returns exhaustive
`no_match`. The explicit policy accepts the same samples and selects Cedar.
Both cases reopen and replay after the loopback endpoint closes and retain two
original GETs each. The 29-code policy passes the retained input schema's
32-item and 2,048-byte limits.

The adapter, current-weather example and planner package tests passed. The
focused race run passed in 1.286 seconds for the adapter and 121.542 seconds
for the planner. Scoped Staticcheck and the repository build also passed.

The hourly protocol was unpublished at review. If a pre-correction development
Store retained an otherwise valid 97 response as unavailable, this revision's
offline verifier rejects that record when the native body now validates.

The independent reviewer also ran a maximum-input probe at the reviewed commit
with sixteen eligible candidates and the original 28 codes. Both assess-first
deployments returned `recommended` with 16 GETs, 2,872 reductions and 19
dispatches apiece. Both reopened with zero additional reductions or dispatches.
The probe took about 273.596 seconds. Each Store occupied 111,665,152 bytes and
the report occupied 4,453,845 bytes. These are uncontrolled observations at the
reviewed revision. The revised 29-code policy was checked against the input
schema without rerunning the maximum-input probe.

## Repository verification

Run from the repository root with the Go cache and loopback fixtures available.

```sh
go mod verify
go mod tidy -diff
gofmt -l examples/outing-planner/cli_test.go examples/outing-planner/planner_test.go \
  examples/outing-planner/barrier_process_test.go examples/outing-planner/process_test.go \
  examples/outing-planner/walkthrough_fixture_test.go
go build ./...
staticcheck ./...
staticcheck -tags integration ./...
go test -count=1 -timeout=60m ./...
go test -count=1 -timeout=60m -tags integration ./...
go test -count=1 -timeout=60m -race -tags integration ./...
```

For the initial qualification at `0eeb0ec`, all commands passed on Darwin
arm64 with Go 1.27.4 and Staticcheck 2026.2.2.
Formatting listed no files. The ordinary, tagged integration, and tagged
integration race suites took 54.51, 76.32, and 817.52 seconds respectively.
The focused tagged run took 15.237 seconds and its race run took 246.393
seconds. The documented inspect and replay commands produced byte-for-byte
matching JSON after the walkthrough fixture shut down.

The test-only correction for decision admission before termination passed the
focused six-case integration run in 15.427 seconds and its race run in
253.877 seconds. Both Staticcheck modes, formatting and the example build
also passed. No production source changed in that correction.
