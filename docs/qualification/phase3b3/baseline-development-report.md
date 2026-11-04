# Task 1 report — current public runtime authority baseline

Status: IMPLEMENTED AND LOCALLY QUALIFIED; independent parent review pending.
No production runtime files, prior research fixture files, or existing tests were
changed. No commit was created. Runtime HEAD remains
`af6943a5805a9a6c0e8fe77f804264f4765d07ec` in
`/private/tmp/alan-phase3b3-runtime`. Source snapshots and SHA256 digests are
retained under `/tmp/alan-phase3b3-baseline/fixture-source/` and
`source-digests.json`; an immutable fixture commit remains the parent's freeze
step. This is a historical baseline, not an S2 realization.

## Question and result

Can the unchanged public API realize B-to-existing-A request-bound delegation,
when A has only its own complete counter-A/red increment permission and B has
counter-B/blue increment? The ordinary retained attempt is refused at the exact
adapter permission decision. Source authorization is allowed; the adapter attempt
is retained as `grant_denied`; the refusal result is actually selected; independent
endpoint ledgers show zero unauthorized physical contacts. Grant-shaped message
data, an authentic prior B result, and resource/scope or operation Cartesian
mixing do not confer authority.

The S1 apparatus control consumes two authentic reciprocal results through the
ordinary authored A-servicing/B-serial policy. The opt-in desired S2-completion
assertion then fails for the authority reason, after a separate S1 control passes.
This establishes a tested current-API obstruction for this fixture, not a proof
that an integrated new primitive is impossible, and not a universal safety theorem.

## Changed files

- `examples/delegation/baseline.go`: exports `BaselineScenario`, the five exact
  requested constants and `Baseline(scenario) (runtime.Provision, error)`; returns
  two ordinary authored roots and fixed principal-local complete permissions.
  Endpoint adapters remain separately supplied by the caller.
- `examples/delegation/baseline_test.go`: deterministic construction, exactly two
  static roots, exact A/red and B/blue grants, and unknown-scenario rejection.
- `internal/durabletest/phase3b3_baseline_test.go`: separately configured trusted
  loopback adapters for A/red, B/blue, A/blue; existing separate-process counter
  fixtures; actual public runtime Create/Run/Inspect/InspectExecution/Replay;
  exact original-custody assertions, independent contacts, export and RED mode.

The main local/missing/claim/cartesian fixtures retain A-servicing/B-serial.
`result_launder` deliberately reverses that order (B services first, A serial)
solely to obtain a genuine prior B result before A's adverse attempt. It is an
extra negative control outside the main positive schedule grammar.

All fixtures retain an ordinary local hold receive after service consumption.
“Completion” here means both original authentic results consumed (`paid=true`
for both logical roots); it does not claim terminal-root completion or liveness.
The holds preserve the authenticated result custody for inspection.

## Actual witnesses and identity binding

Canonical non-race artifacts are indexed in `selected-witnesses.json` and summarized
in `selected-summary.json` under `/tmp/alan-phase3b3-baseline`.

| Case | Retained actual adapter operations | Independent contacts A/red, B/blue, A/blue |
| --- | --- | --- |
| local | A `["a",1,6,0]` validated; B `["b",1,12,0]` validated | 1, 1, 0 |
| missing | A `["a",1,6,0]` B/blue increment `grant_denied` | 0, 0, 0 |
| claim | Same refused original, actual grant-shaped request claim | 0, 0, 0 |
| cartesian | A `["a",1,6,0]` A/blue increment `grant_denied` | 0, 0, 0 |
| result_launder | B `["b",1,6,0]` validated; A `["a",1,12,0]` B/blue refused after consuming B result | 0, 1, 0 |
| operation Cartesian | A `["a",1,1,0]` A/red op1 validated; B `["b",1,1,0]` B/blue op2 validated; A `["a",1,3,0]` A/red op2 refused | 1, 1, 0 |

The extra operation-coordinate control fixes complete permission set
`P={(counter-A/red,op1),(counter-B/blue,op2)}`. It actually attempts the absent
`(counter-A/red,op2)`, preserving the distinct scope-coordinate Cartesian probe.
This control also uses exactly two ordinary static roots, but does not purport
to be reciprocal service.

For main fixtures, `original-bindings.json` records actual request-producing
operation, requester and server RunIDs/generation, admitted request observation,
exact request-selection reduction, service operation and sequence, original
adapter-result observation/selection, and authentic reply operation/observation/
selection. It is checked against typed causal producers and original raw operation
payloads, never merely the authored `request_id` string or final paid flags.
For example B's original request `["b",1,4,0]` is selected by A/generation 1 at
sequence 5; A emits `["a",1,6,0]`, selects its result at sequence 7 and, only for
validated S1, emits reply `["a",1,9,0]` that B selects at sequence 5. B's own
service is later at sequence 12, after that reply selection, preserving the policy.
The prior-B-result control checks A's authentic B-reply selection at sequence 5
before A's refused operation at sequence 12, with actual prior mutation included
as ordinary request data.

## Commands and outcomes

All commands below used Go `go1.27.3 darwin/arm64`,
`GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off`
and `ALAN_PHASE3B3_EVIDENCE=/tmp/alan-phase3b3-baseline`.
Full raw stdout/stderr and JSON command/environment/status records are retained
as named `.log` / `.json` pairs in that directory.

- `go test ./examples/delegation -count=1 -v`: PASS (`unit-final`).
- `go test -tags=integration ./internal/durabletest -run '^TestPhase3B3Baseline' -count=1 -v`:
  PASS all five scenarios plus operation Cartesian; RED-only test skipped normally
  (`integration-final`).
- `go test -race -tags=integration ./internal/durabletest -run '^TestPhase3B3Baseline' -count=1 -v`:
  PASS, including race-enabled endpoint children (`integration-race`).
- `ALAN_PHASE3B3_EXPECT_S2_COMPLETION=1 go test -tags=integration ./internal/durabletest -run '^TestPhase3B3BaselineS2CompletionExpectedFailure$' -count=1 -v`:
  EXPECTED EXIT 1 after passing S1 apparatus control (`expected-s2-failure`).
  Exact diagnostic: `EXPECTED FAIL: desired S2 completion unavailable on unchanged public API: original ["a",1,6,0] run={Subject:a Generation:1} authorization=allowed adapter=grant_denied; both authentic results required`.
- `env STATICCHECK_CACHE=/tmp/alan-phase3b3-staticcheck staticcheck -tags=integration ./examples/delegation ./internal/durabletest`:
  PASS, no diagnostics (`staticcheck-final`). No Go vet substituted.
- `gofmt -w` on the three new files: PASS (`format`, `format-final`).
- `go test ./examples/delegation -count=1`: PASS after final import grouping
  (`unit-reviewed`); no semantic edits after integration/race qualification.
- `git diff --check`: PASS (`diff-check`); these are new untracked files, so
  this check by itself is not a substantive review of their contents.

Preserved development/environment failures: first unit invocation before the
fixture existed failed with “no non-test Go files”; first fixture construction
caught an accidentally renamed `Local` field, corrected before passing unit
qualification. Integration compilation before adding the operation-coordinate
control reported an unused author import (`integration-first`). A sandboxed
loopback startup attempt failed existing helper readiness (`integration-first-run`);
the identical suite passed with loopback escalation (`integration-loopback`).
Staticcheck's default cache was outside the writable sandbox; those failures are
preserved (`staticcheck`, `staticcheck-local-cache`). `XDG_CACHE_HOME` does not
redirect its macOS cache; `STATICCHECK_CACHE` does, and the final run passed.
None of these earlier failures is represented as a semantic counterexample.
The initial two unit development failures preceded the command logger; their
outputs remain in the agent tool transcript, while all subsequent qualification
commands have complete file logs.

## Exported evidence and concerns

Each completed run exports raw provision, actual runtime SQLite Store, full raw
Inspect BLOB strings, typed inspection, explicit history (the typed snapshot's
private history is not JSON-exported automatically), Run report, Replay report,
original custody bindings where applicable, independent endpoint snapshots and
exact closed endpoint database bytes/WAL sidecars. Per-run `digests.json` and
aggregate `all-evidence-digests.json` contain SHA256 digests. Endpoint processes
are stopped before Replay, so Replay success cannot depend on endpoint access.
The independent endpoint contact/key/application/receipt assertions are kept
separate from Replay's internal-consistency result. Positive Stores are never
tampered with; no negative DB copies were necessary.

No crash/reopen, selective root revocation, authority transfer, fresh physical
admission under new delegation, model correspondence, broader Go suite or paper
reconciliation is claimed by this task. Production semantics are unchanged, so
no new manuscript theorem/semantic change is made. Parent should review the
fixture and preserved negative evidence, freeze its source, and only then start
the production worker under the independently checked frozen contract. The
historical refusal must remain labeled as the unchanged-runtime obstruction.

## Source SHA256

- `examples/delegation/baseline.go`: `bfd27250eb305c77b94343ba8d6299132125c48b6b2638ce8ad239a44611436e`
- `examples/delegation/baseline_test.go`: `193bb6094fbd202f0551e774a079b077a62589022794acf7622492d0db4c7c29`
- `internal/durabletest/phase3b3_baseline_test.go`: `201f5cb0f1837c531888297a08a920f267b582c9d0fdeeec162cf314d7d15e7a`
