# Phase 1J concrete retained-join witnesses

Question: does keeping the root at an ordinary retained receive preserve the
scope lifetime needed by its static peer? The concrete samples answer yes for
this fixed Store.Run schedule. They do not establish the all-schedule completion
claim: ordinary report source authorization can fail after peer termination,
as investigated independently by `research/join`.

## Scope and method

Only four additive `internal/substitutiontest/join_*_test.go` files implement
this harness. The unchanged reviewed `validateRetained`, fixture provisioning,
run inspection, stable-prefix, process ACK/SIGKILL/wait, replay, and environment
operator helpers are reused. No production runtime, compiler, scheduler,
dependency, old fixture, or authoritative database row was changed. The new
process entrypoint selects cuts by reading actual retained state. It uses the
unchanged runner with MaxInFlight=1, MaxSteps=200, MaxDispatch=200. No additional
operational allowance, sleeps, identity selection, paid calls, or protocol-driving
harness input was needed. The separate authorized cancellation uses the reviewed
ordinary lifecycle operator.

The test-only adapter validates original report intent/history, source
authorization and its actual source sequence snapshot, admission, selected
observation, receive guard and actual bound payload, and acknowledgment links.
It preserves selected candidate/S outcomes rather than replacing them with latest
publications. Absent responses must delete response bindings; explicit report
fillers carry no present-response assertion. Actual charges are reconciled with
accepted original admissions. Behavioral decisions and report truth are checked
by the separately authored `research/join.CheckTrace`; event ordering uses real
per-run sequences, not a fabricated cross-run total order. Authorization and
message admission are distinct events. The adapter's source-generation/revision
profile is intentionally static J1; other source-denial reasons fail closed as
unsupported by this profile.

The final oracle-v2 normal suite contains 36 passing leaf cases (13 matrix,
13 decoder/oracle mutations, four original-context cases, four process recovery
cases, one duplicate-delivery and one explicit-cancellation case).

## Concrete results

- C0: A and B at I=R3 and I=R4 each reproduce root local success, peer S
  `scope_closed`, and aggregate failure. More capacity does not repair lifetime.
- C_join: A and B succeed locally and in aggregate, including peer local success.
  Each includes candidate consumed, S unadmitted, root open at the mandatory
  message receive. Actual dynamic spending is A=(2,2,0), B=(3,3,0); root and peer
  are two separately reported fixed runs.
- B I2/R3 rejects S with `invocation_limit`; B I3/R2 rejects S with
  `created_run_limit`. Both report the actual refused outcome and fail after
  joining, with actual spending (2,2,0).
- Candidate fail and S fail yield `failed/member_failed`. Candidate absence and S
  absence yield `failed/missing_reply`. The report and root decision preserve
  these known outcomes.
- The valid premature-report program produces root success in this runner
  sample. The unchanged independent checker rejects `premature_report`, because
  the peer's original S consumption follows report commitment. This is a peer
  contract violation, not a decoder rejection or runtime receipt.
- The schema-valid wrong-request report is admitted and consumed as ordinary
  communication. Root remains open at its extra receive; the oracle rejects
  `request_correlation`. It does not discharge the designated join.
- Candidate pending with successful or failed S, and S pending with failed
  candidate, remain open. The pending branch has consumed its original request
  and is retained at an open receive. These are bounded observations, not proofs
  of eventual nontermination.
- Authorized root cancellation at candidate-consumed preserves its actual
  cancelled aggregate outcome and spent charges. The operator is outside the
  root's two static participants and is reported separately.
- Thirteen targeted decoder/oracle mutations are rejected: changed report payload,
  stale source generation, selected identity, bound payload, source authorization
  snapshot, delivery logical identity, acknowledgment identity, duplicate
  observation, causal order, wrong S handle, wrong request, distinct extra report, and report
  consumption before candidate consumption. Two actual
  duplicate deliveries of the original transport return the original admission
  and leave the ledger unchanged. Replay likewise leaves records unchanged.

## Four actual process interruptions

A and B are each crossed with `candidate-consumed` and `report-committed`.
The child ACKs the retained predicate, the parent sends SIGKILL and verifies the
signal, and another PID opens the original database without redeployment.
Candidate-consumed cuts occur at root sequence2, before S admission, with charges
A=(1,1,0), B=(2,2,0). Report cuts occur at peer sequence37, with the report intent
committed and root still unconsumed; neither has an ordinary report observation
at that cut. Charges are already A=(2,2,0), B=(3,3,0).

Original run/generation/artifact, invocation, report-if-present, history,
observation, admission, and charge records survive as an immutable prefix.
Root activation epochs renew 2→3. Each resumed root, peer and aggregate succeeds.
A fresh replay process (with macOS network denial when available) preserves the
entire final inspected ledger. ACK, executable hash, argv, PID/signal/exit output,
before/after records and actual epochs are retained separately for every case.

## Reproduction and evidence

From the repository root, choose a new output directory on each invocation
because the reviewed retention helper refuses to overwrite artifacts:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
GOCACHE=/tmp/alan-phase1j-go-build ALAN_I_ARTIFACTS=/tmp/alan-join-normal-unique \
go test -tags integration -count=1 -timeout=10m -v -run '^TestJoin' ./internal/substitutiontest
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
GOCACHE=/tmp/alan-phase1j-go-build ALAN_I_ARTIFACTS=/tmp/alan-join-race-unique \
go test -race -tags integration -count=1 -timeout=30m -v -run '^TestJoin' ./internal/substitutiontest
```

`final-normal.log`/`final-normal/` and `race.log`/`race/` preserve the complete
oracle-v1 normal (exit0,9.151s) and race (exit0,135.803s) qualification, including
all process cases. Independent oracle self-review then strengthened two existing
J1 checks: extra reports and root report consumption before candidate consumption.
The preserved v1 source and correction rationale are owned by the model evidence.
`oracle-v2-normal.log`/`oracle-v2-normal/` requalify the entire suite against v2
(exit0,7.933s). `oracle-v2-race.log`/`oracle-v2-race/` recheck the changed matrix
and thirteen decoder/oracle mutations under race (exit0,70.933s); unchanged process machinery
was already fully race-qualified. Completed statuses appear in `execution.json`.
`source-manifest.json` records final source hashes, while
`source-manifest-oracle-v1.json` preserves the earlier source census. `runtime-summary.json` retains compact case results and
normalized events; `index.json` maps test names and every evidence file to hashes.
`normal.log`/`normal/` preserve the prior passing run before stronger assertions
for exact refusal causes and pending evidence were added.

Development REDs are retained separately. `red-build.log` records the expected
unavailable new independent package during initial test authoring. The first
runtime RED used the wrong raw key `initial_projection`; the inspected schema
actually names it `initial`. The second filtered all peer-origin message
observations as reports, accidentally including the runtime-invocation-request
observation delivered to S. Restricting the report mapping to the designated root
fixed that apparatus bug. Exact pre-correction decoder sources and logs are
preserved. These are harness development failures, not semantic counterexamples.

The guarantee remains bounded: these samples show lifetime preservation and
retained recovery under the concrete runner, not all fair schedules, general
substitution, distributed recovery, arbitrary generations/registries, or effectful
recovery. No concrete `source_closed` report schedule is claimed by this suite.
