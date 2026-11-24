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

## Delegated Weather: final bounded campaign

The final executable/test/fixture source snapshot contains 409 selected files at
checkout head `4af37867c030cb353df3b85686ff8bdc20fe6ec6`, plus the Task5 worktree
changes. Its sorted name/hash-map SHA-256 is
`d10d834d86837c1ea2b31c6e3f2e8d5e51e04eac9a041bca1ee99aa5f4d2aa9a`.
This identifies frozen source bytes, independently of the qualification archive
commit. The [controlled archive](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/qualification/delegated-weather/README.md)
contains the source map, raw inputs, complete captures, original command/exits,
negative controls, review history and a portable independent-check reproducer.
The [decision record](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/qualification/delegated-weather/DECISIONS.md)
preserves the controller's scope/rework rulings and is not an evidence verdict.

```sh
ALAN_RAW_CAPTURE_DIR=/tmp/delegated-normal go test -tags integration \
  -count=1 -timeout=60m -run '^TestDelegated(Process|Qualification|Raw)' -v ./examples/outing-planner
examples/outing-planner/testdata/delegated-walkthrough.sh /tmp/delegated-walkthrough
ALAN_RAW_CAPTURE_DIR=/tmp/delegated-race go test -v -count=1 -timeout=60m \
  -race -tags integration ./...
```

The single verbose full integration/race run supplies race coverage for the named
focused process/CLI tests. No separate final filtered race campaign is claimed.
All required repository commands above were rerun on the final source after the
last helper edit; the archive records their exact actual argv/exits. Earlier
completed and interrupted campaigns remain separately labeled as superseded.

Final verification passed with no race diagnostic. Normal and race campaigns each retained:

| Campaign | Accepted cuts | Covered row entries summed per cut | Process SIGKILL witnesses | Actual CLI pair comparisons |
| --- | ---: | ---: | ---: | ---: |
| normal | 110 | 26,421 | 13 | 2 |
| race | 110 | 26,421 | 13 | 2 |

The row sum counts repeated retained prefixes per cut. Each cut covers all 42 tables.
The two real CLI comparisons per campaign separately check four complete Stores;
the walkthrough adds its own accepted/refused pairs. The fixture-backed Python
campaign ran all 14 methods without skips, including the two real exports and
11 adverse export variants. Source-only execution ran five methods and explicitly
skipped nine capture-dependent methods. All 24 model methods passed.

| Required full gate (actual argv) | Exit | Elapsed seconds |
| --- | ---: | ---: |
| `go mod verify` | 0 | 0.078 |
| `go mod tidy -diff` | 0 | 0.053 |
| `go build ./...` | 0 | 1.543 |
| `staticcheck ./...` | 0 | 0.283 |
| `staticcheck -tags integration ./...` | 0 | 0.340 |
| `go test -count=1 -timeout=60m ./...` | 0 | 53.787 |
| `go test -count=1 -timeout=60m -tags integration ./...` | 0 | 227.165 |
| `go test -v -count=1 -timeout=60m -race -tags integration ./...` | 0 | 2982.967 |

Elapsed values are diagnostic run observations, not application benchmarks.

The twelve genuine SIGKILL/reopen cuts are:

| Cut | Actual acknowledged coordinates | Endpoint GETs at cut → completion |
| --- | --- | --- |
| Proposal committed before delivery | Outing/gen 1/epoch 2/seq 18 |0→2|
| Invocation committed before admission | Weather/gen 1/epoch 2/seq 8 |0→2|
| Accepted helper/request before selection | Weather caller/gen 1/epoch 0/original seq 8 |0→2|
| Native helper result before settlement | Helper/gen 1/epoch 1/original seq 5 |1→2|
| Settled helper result before selection | Helper/gen 1/epoch 1/original seq 5 |1→2|
| Second original forecast before authorization/dispatch | Helper/gen 1/epoch 1/seq 43 |1→2|
| Helper reply/End/freeze before publication | Helper/gen 1/epoch 1/seq 82 |2→2|
| Published complete outcome before parent consumption | Weather caller/gen 1/epoch 0/original seq 8 |2→2|
| Parent assessment committed before delivery | Weather/gen 1/epoch 2/seq 38 |2→2|
| Public decision committed before first authorization | Outing/gen 1/epoch 2/seq 56 |2→2|
| Allowed first decision authorization before admission | Outing/gen 1/epoch 2/original seq 56 |2→2|
| Decision admitted, matching Weather receive staged/unselected | Weather/gen 1/epoch 2/next seq 39 |2→2|

“Helper” labels the actual namespace-derived admitted member; full unmodified
subjects, invocation/outcome/effect IDs and events remain in every witness/Store.
Invocation service hooks use caller/original-call coordinates and epoch 0;
adapter hooks retain the original authorization epoch and omit Root. Actual
Intent/Attempt/Root fields are verified against the same interrupted Store.
Each owner is killed immediately after the acknowledged barrier, before later
inspection, and its waited signal must be SIGKILL. A fresh independent public CLI
owner resumes the same original history. Published helper response/delivery
already form one atomic retained boundary; no publication-without-delivery cut
is invented. Parent consumption and final decision selection each occur once.

At every decision-commit capture, the independent raw checker verifies open
sender/receiver, the exact matching Weather wait, and the original decision as
the sole pending internal delivery. It inventories all ordinary/invocation/
terminal routes and raw original acknowledgments; public Pending alone suppresses
recorded-reply acknowledgment debt and cannot establish this premise. A later
completed Store cannot retroactively establish that earlier cut.

The independently acknowledged held GET is bound to its original helper effect,
prepared key/client-request header and dispatch. After owner SIGKILL, two fresh
resumes and all status/inspect/replay readers preserve one total endpoint contact,
one original invocation/member/effect, missing result/settlement/observation,
unknown debt and both outstanding ancestors. No later candidate starts. This
observes missing retained evidence under the declared no-later-result/no-retry
continuation; it does not prove permanent unavailability of a genuine result.

Raw row comparisons preserve namespace, canonical program inventory, root
profiles/limits, original generations/inputs, histories, original operations,
retained admissions/results and paid charge/fact prefixes. Legitimate subsequent
work can add its first charge. Only already-open activation epochs renew; closed
helper epochs stay fixed. Offline readers agree with complete retained snapshots
and remain contact-free. Known invalid native results are diagnostic evidence,
separate from absent-result uncertainty.

The fresh raw campaign also exercises Send→End→Prepare with a frozen bound reply
still unadmitted: `reply_admission` remains pending and no empty success publishes.
It then authorizes/records that original bound reply after End, acknowledges the
same transport, publishes its exact complete response and selects it once.
The separate public-decision late-first-authorization control permanently records
`source_closed`; the allowed-before-End/admitted-after-End pair can complete.
Those are deliberate public-API schedules, not default-runner failures.

Two real integration findings remain preserved. The original refused replacement
export described an older pre-reopen epoch and failed strict raw report coverage;
production fix `3e873c0` now exports the final delegated cut while preserving
legacy output. The later refused export omitted nonexistent original-ID keys as
declared, while the reader expected empty strings; reader fix `4af3786` now
constructs only existing original keys and keeps exact ID/order/count/cost/extra
field checks. Both independent approvals and original failures are archived.
Task4's earlier REJECT, altered-final-wait acceptance, four semantic RED controls
and corrected continuation/debt rejection also remain. Old 83 captures were
rechecked after the fixes and are not relabeled newly captured executions.

A full race attempt on the superseded `33bb3b06...` source exhausted the unchanged
60m package limit after all 110 cuts, before the actual CLI table tail. It is a
failed gate, not a race pass; no data race or semantic assertion preceded its
alarm. The raw pause harness had inspected the complete Store at every boundary.
A controlled, sole-job, same-fixture/seed/race-flag diagnostic pair measured 369
full inspections in 6m10.366209750s versus 59 in 1m13.363730833s after a categorical
prefilter. Total hook counts were 369/365; configuration/fixture/manifest and
complete cut/resumed row digests matched exactly. The final helper skips only
event shapes incapable of satisfying any existing capture predicate. All 14 raw
cut cases, public controls, snapshots/assertions, strict comparisons, replays and
60m limit remain unchanged. Instrumentation/fixed-address/seed setup is external
and archived as diagnostic code, with no general application speedup claim.

The finite model remains 346 configurations, 31,102 states and 79,866 edges. Broad
interleavings retain source-closure nonblocking counterexamples. Positive
nonblocking is conditional on the named `preclose-authorization` domain,
adequate admission and known fixture outcomes permitting successful planning;
no fairness or inevitable arbitrary-scheduler completion is inferred. Direct and
delegated successful costs `(invocations,created runs,effects)` are `(0,0,n)` and
`(1,1,n)`, with an extra helper depth/reply. Unavailability, refusal and unknown
are not successful replacement or the unchanged confirm-first cycle.

These results cover physical schema 9, fixed canonical artifacts and authority,
frozen normalized inputs, 0–2 candidates, one helper, capacity one, one transport
per original and single-host SQLite ownership. Complete raw row coverage and a
compatible causal placement do not supply an unrecorded total chronology or an
independent compiler/full-bytecode proof. Capture/source integrity, normalization,
OS, interpreter/stdlib and SQLite remain trusted. Research §4.5's equal-debit
relation and §6.20's unchanged-plant synchronized monitors do not establish this
helper extraction. No research edit, live-provider, performance, distributed-
ownership, unrestricted-refinement or general congruence claim is made.
