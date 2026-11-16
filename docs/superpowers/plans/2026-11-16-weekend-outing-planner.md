# Weekend Outing Planner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a retained two-agent outing planner with truthful forecasts,
recoverable decisions and an evidenced circular-dependency replacement check.

**Architecture:** Two ordinary authored root programs communicate within one
SQLite Store. A versioned hourly Open-Meteo adapter supplies observations;
retained projections own all protocol decisions. The CLI only provisions,
drives, inspects and compares bounded deployments.

**Tech Stack:** Go 1.27.3+, existing SQLite driver, standard library, public ALAN
packages, Python standard library for the standalone fixture server if useful.

**Spec:** `docs/superpowers/specs/2026-11-16-weekend-outing-planner-design.md`

## Global Constraints

- Work only in this isolated `codex/weekend-outing-planner` worktree.
- One planning session per Store, two static subjects named `outing` and `weather`, and one final decision.
- At most four parks and four requested one-hour start slots: at most sixteen park/slot candidates.
- No new dependencies, runtime instructions, storage formats or authority primitives.
- Use public ALAN packages in example production code; no internal executor import or direct SQLite access.
- Keep original configuration, normalized candidates, proposal, assessment, decision and effect identities retained.
- Missing external results remain unknown across reopen; never automatically resend or invent evidence.
- `no_match` requires no locally eligible candidate or exhaustive validated unsuitable assessments.
- Preserve the current weather adapter's request bytes, behavior and historical example.
- Bound claims to this single-host application and supported weather variants; no general substitution or distributed-recovery claim.
- Use Staticcheck as the preferred lint check, and deterministic local fixtures for qualification.
- Inspect full diffs before narrowly scoped commits; preserve negative evidence and independent reviewer REJECT findings.
- Push a branch and create a pull request when ready; do not merge automatically.

## Review Focus

- DST folds/gaps and fractional-offset zones must not silently move requested slots; test in Task 2.
- A valid later candidate may qualify after an unavailable earlier row; missing responses still prevent the batch from completing; test in Tasks 2/4.
- Missing fixture tuples, duplicate JSON keys and live-endpoint fallback must be rejected; test in Tasks 1/2/3.
- Completion requires final decision consumption, not only a terminal sender or an empty dispatcher; test in Tasks 2/4.
- Reader/owner validation must reject changed artifacts, grants and native evidence before supplying transport; test in Tasks 2/3.

## File structure

- `adapters/openmeteo/hourly.go`, `hourly_observation.go`, `hourly_test.go`:
  the new protocol. A small shared transport helper is permitted to avoid
  duplicating current GET logic; preserve its observable behavior.
- `examples/outing-planner/configuration.go`, `configuration_test.go`:
  strict configuration and deterministic finite normalization.
- `examples/outing-planner/program.go`, `protocol.go`, `deployment.go`,
  `program_test.go`: retained programs, schemas, scope and evidence validation.
- `examples/outing-planner/main.go`, `report.go`, `replacement.go`, `fixture.go`,
  `main_test.go`, `replacement_test.go`: CLI, typed report, fixed-pair check and
  explicit loopback fixtures. Focused extra files are permitted if necessary.
- `examples/outing-planner/process_test.go`, test-only barrier helpers,
  `testdata/`, `README.md`, `docs/outing-planner.md`, root `README.md`:
  fresh-process qualification, reproducible inputs and user walkthrough.

### Task 1: Versioned hourly forecast adapter

**Files:** Create hourly adapter/observation/tests above; modify current adapter
only for shared transport extraction if needed.

**Interfaces:**
- Produce `NewHourly(endpoint string) (*HourlyAdapter, error)` implementing
  `effects.Adapter`, `HourlyInputSchema() schema.Record`,
  `HourlyOutputSchema() schema.Record`, and
  `VerifyHourlyResponseEvidence(p effects.Prepared, c effects.Candidate) error`.
- Export `HourlyProtocol = "open-meteo/hourly/v1"`, `StartHour = "start_hour"`,
  `EndHour = "end_hour"`, `StartWeatherCode = "start_weather_code"`,
  `EndWeatherCode = "end_weather_code"`, `StartTemperature = "start_temperature"`,
  `EndTemperature = "end_temperature"`, and `ValidWeatherCode(code int64) bool`.
- Input: existing latitude/longitude/units strings and StartHour/EndHour canonical
  UTC RFC3339 strings, whole hours exactly one hour apart, nonzero year.
- Output: existing Source/GridLatitude/GridLongitude/TemperatureUnit strings,
  StartHour/EndHour strings, exact decimal StartTemperature/EndTemperature
  strings and integer StartWeatherCode/EndWeatherCode.

- [ ] Write `TestHourlyRequestAndEvidence` before implementation: freeze exact
  `hourly=temperature_2m,weather_code`, `start_hour`/`end_hour` ISO-minute query,
  `timezone=GMT`, units, `cell_selection=nearest`; verify two exact samples,
  retained request identity and evidence roundtrip.
- [ ] Run `go test ./adapters/openmeteo -run TestHourly -count=1` and record RED.
- [ ] Implement adapter reusing endpoint validation, single-GET protections,
  bounded response handling and native evidence validation. Preserve existing
  `Protocol` and current schema/GET bytes. Use duplicate-key-rejecting native
  JSON decoding and the existing 50 km nearest-grid sanity bound.
- [ ] Add table negatives: malformed/duplicate keys, wrong/missing/duplicate
  sample time, mismatched array sizes, null values, unsupported WMO codes,
  wrong units/zone/grid, forged successful payload or prepared request,
  redirects, truncated/oversized body and invalid time ranges. Documented WMO
  codes are `0,1,2,3,45,48,51,53,55,56,57,61,63,65,66,67,71,73,75,77,80,81,82,85,86,95,96,99`.
- [ ] Run `go test -count=1 ./adapters/openmeteo ./examples/weather-agent`,
  `go test -race -count=1 ./adapters/openmeteo`, and
  `staticcheck ./adapters/openmeteo`; record GREEN, self-review and commit.

### Task 2: Configuration and ordinary retained planning programs

**Files:** Create configuration/protocol/program/deployment and focused tests.
Do not create a placeholder executable; Task 3 adds the CLI.

**Interfaces:**
- Consume Task 1 hourly schemas, fields and evidence verifier.
- Produce `readConfiguration(path string, now time.Time) (configuration, error)`,
  `(c configuration) normalize() (normalizedPlan, error)`,
  `(c configuration) provision(variant string) (runtime.Provision, error)`,
  `retainedConfiguration(s *runtime.ExecutionSnapshot) (configuration, string, error)`
  and `verifyPlannerEvidence(s *runtime.ExecutionSnapshot, c configuration, variant string) error`.
- `configuration` retains exported JSON fields `Version`, `ReferenceTime`,
  `Date`, `Timezone`, `Units`, `Endpoint`, `MaxTravelMinutes`,
  `AllowedWeatherCodes []int64`, `Availability []timeWindow`, `Slots []string`,
  `Parks []park`. Use snake_case JSON names, version 1; omitted reference time
  freezes supplied `now`, date accepts an explicit YYYY-MM-DD or `tomorrow`.
- `timeWindow` has Start/End strings. `park` has ID/Name/Latitude/Longitude
  strings, TravelMinutes int, Opening []timeWindow. Clock inputs may be HH:MM
  on the target local date or explicit-offset RFC3339; reject ambiguous clock
  input unless disambiguated with an offset. Normalize to UTC, never round a
  requested slot. Empty availability/parks/slots can express no feasible option.
- `normalizedPlan` exposes Date string and Candidates []candidate; candidate
  fields are ID, ParkID, Name, Latitude, Longitude, Start, End strings,
  TravelMinutes int, Eligible bool, Reason string. Retain all bounded rows and
  their eligibility; sort by travel/start/park ID. Exported field types may be
  extended without changing these interfaces when useful to the report.
- Variants are `assess-first` and `confirm-first`; only Weather changes.
  Subject IDs `outing`, `weather`; capability `open-meteo-hourly`; effect
  `forecast`; obligation names `outing-proposal`, `outing-assessment`,
  `outing-decision`. Each message carries proposal ID and exact candidate data.

- [ ] Write and run failing normalization and runtime tests, including
  `TestPlannerRecommendationAndNone`, `TestPlannerConfirmationFirstWaits`,
  `TestPlannerPartialEvidence`, `TestPlannerCalendarBounds`; record RED.
- [ ] Implement strict bounded JSON loading (including duplicate keys), freeze
  reference date once, validate IDs/coordinates/intervals/WMO policy and compute
  travel-inclusive local eligibility. Test 4×4 maximum, over-bound rejection,
  malformed offsets, DST fold/gap and a fractional-offset zone. Never silently
  truncate or approximate a slot to a different forecast hour.
- [ ] Implement two ordinary authored programs. Outing filters retained rows,
  sends a batch, awaits it and decides; Weather requests/awaits each original,
  applies allowed codes using authored equality/collection steps, sends the
  batch and consumes the decision. Empty candidates complete the same protocol.
  Retained invalid results become per-row unavailable; an original missing
  response remains waiting. Valid later candidates may qualify; no-match
  cannot follow incomplete evidence. Keep state transitions out of Go loops.
- [ ] Validate full batch identity, coverage and correspondence before deciding.
  Candidate list equality or equivalent authored checks must exclude omission,
  duplicates, changed coordinates/times and mismatched proposal IDs. Persist the
  selected forecast operation/observation link needed by reporting.
- [ ] Provision two finite scopes, Weather-only forecast authority, at most one
  forecast per candidate and MaxInFlight 1. Reconstruct exact initial programs,
  inputs, grants and scopes from retained configuration on inspection. Verify
  every forecast request and successful native result against those inputs.
- [ ] Exercise real runtime execution/reopen/replay with loopback fixtures:
  recommendation, all-unsuitable, zero eligible, earlier invalid/later valid,
  invalid-only unavailable, mismatched messages, exact final decision selection,
  both variants wiring successfully and different Weather artifact IDs. Test
  unsupported deployment/grant/artifact changes are refused.
- [ ] Run `go test -count=1 ./examples/outing-planner`,
  `go test -race -count=1 ./examples/outing-planner`, and
  `staticcheck ./examples/outing-planner`; record GREEN, review and commit.

### Task 3: CLI, retained reports and replacement witness

**Files:** Create main/report/replacement/fixture files and their tests; add
initial bounded sample configuration and fixture JSON under testdata.

**Interfaces:**
- Consume Task 2 configuration, provision and validation functions.
- Produce `runCLI(ctx context.Context, args []string, out io.Writer) error`,
  `buildReport(s *runtime.ExecutionSnapshot, c configuration, variant string) (plannerReport, error)`
  and executable commands `start`, `run`, `resume`, `status`, `inspect`, `replay`,
  `check-replacement`; common `--db`, readers `--json`; start `--config`,
  `--weather` (default assess-first); checker `--config`, `--fixture`, `--weather`,
  optional `--output-dir`. Help, bad flag/argument handling and errors follow the
  existing weather example. A command succeeding is distinct from plan success.
- Fixture version 1: reference_time and entries mapping exact latitude,
  longitude, start_hour/end_hour strings to status, header map and native body
  string. Freeze bytes/digests; no fixture entry may issue an external request.
  Checker creates an isolated literal-loopback HTTP server and fresh Stores;
  retain source/derived config and fixture hashes in its JSON report.

- [ ] Write and run CLI/replacement tests before implementation; record RED for
  recommendation text/JSON, no-match/unavailable/pending separation, no network
  on readers/replay, and the real circular wait message.
- [ ] Implement start without GET, idempotent same-deployment recognition and
  conflicting-start refusal. Implement owner/read paths with exact deployment
  and native-evidence checks, rechecking after Open before enabling transport.
  Preserve original namespace/configuration/endpoint and use bounded runner
  calls. Run/resume must not silently classify ErrLimit as quiescence.
- [ ] Report both agents, proposal/candidate/assessment/decision data, original
  effects, native evidence/selection links and outstanding waits. Human copy
  always includes calendar date/timezone, explains evidence gaps and labels
  elapsed slots historical. Recommended requires both successful and decision
  consumed; unknown transport and failed attempt stay visibly distinct.
- [ ] Implement checker using actual runs, not a name-based failure message.
  Require recognized exact program artifacts, both actual waits, no matching
  observation/pending relevant message/effect/external input and no exhausted
  bound before diagnosing the supported cycle. Show PC and obligation evidence
  and the fixed-pair first-missing-action explanation. Baseline and candidate
  use the same Outing artifact/config/fixtures, except Weather program bytes.
- [ ] Require fixture input, reject missing/duplicate tuple definitions and
  unexpected requests, never fall back to a live endpoint, and preserve hashes
  and Stores when output-dir is supplied. Fresh-owner reopen retains the bad
  witness. Unsupported variants are unsupported, not compatible.
- [ ] Negative controls: pending message, small step bound and unknown provider
  response do not report circular_dependency. Tampered native result and
  broadened grants cannot reach transport. CLI handles empty/bad/extra args.
- [ ] Run `go test -count=1 ./examples/outing-planner`,
  `go test -race -count=1 ./examples/outing-planner`,
  `go build ./examples/outing-planner`, `staticcheck ./examples/outing-planner`;
  remove any generated repository-root executable, record GREEN and commit.

### Task 4: Process recovery, offline walkthrough and final qualification

**Files:** Add process tests/barrier helpers, fixture-server invocation and
complete testdata/README/docs; update root README with one application link.
Production fixes exposed by tests belong in an independently reviewable commit.

**Interfaces:** Consume the ordinary CLI and public runtime snapshots. Build
the example as an external consumer without internal imports. Test-only helper
code may use the existing causal barrier machinery under an integration tag;
do not add production instrumentation or time-based kill guesses.

- [ ] Write fresh-process tests for proposal committed, forecast result recorded
  before selection, decision committed before consumption and provider-accepted
  response lost. Record the exposed assertion failures before any required fix;
  existing-correct behavior can be qualified directly without fabricating RED.
- [ ] Acknowledge each exact retained cut before SIGKILL. Fresh readers and
  owners must preserve namespace, generations, artifacts, operation IDs and
  history prefix; open epochs renew. Known-response cuts complete with no
  repeated GET. The lost original remains unknown through two fresh resumes.
  Verify both terminal outcomes and actual decision consumption in completed
  cases; interrupted bad replacement stays blocked after reopen.
- [ ] Make fixture contact counts and original IDs independent endpoint evidence.
  Exercise multiple candidate contacts and a pending-next-candidate cut; retained
  earlier forecasts must not be replayed as new calls. Assert offline replay
  and inspection neither mutate projections nor contact the provider.
- [ ] Supply executable offline walkthrough with labelled fictional parks and
  fixed dates, a fixture server that outlives planner commands, recommended and
  no-match paths, checker commands and explicit optional live configuration.
  Document allowed-code policy, 50 km grid sanity bound, history/unknown
  outcomes, provisioning normalization and bounded research interpretation.
- [ ] Run targeted integration/race tests and the documented walkthrough. Save
  compact reproducible evidence/commands, not invented counts or arbitrary large
  database dumps. Include research base SHA and precise manuscript sections.
- [ ] Run full final checks: `go mod verify`, `go mod tidy -diff`, scoped gofmt,
  `go build ./...`, `staticcheck ./...`, `staticcheck -tags integration ./...`,
  `go test -count=1 -timeout=60m ./...`,
  `go test -count=1 -timeout=60m -tags integration ./...`,
  `go test -count=1 -timeout=60m -race -tags integration ./...`.
  Investigate actual failures; do not weaken tests. Commit after self-review.
- [ ] Controller obtains independent task and whole-branch review, fixes and
  re-reviews substantiated findings, then pushes and opens a narrow PR with
  invariant, scope, base, RED/qualification commands, limitations and no
  unrelated changes. Do not merge.
