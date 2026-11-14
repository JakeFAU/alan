# P6 Task 1 independent review

Reviewed base `b1db0d51638566836aea787b4d06e13fff6714fb` through head
`5ed2d73a30c9e17c2626ff6e65d915e304e87673`, using the supplied exact diff,
Task 1 brief/report, progress contract, and semantic/cost audit design inputs.

**Overall: APPROVE. Spec compliance: APPROVE. Code quality: APPROVE.**

No actionable finding in this task-scoped review. This approves the common
measurement apparatus and excluded baseline, including readiness to use the
apparatus for the planned candidate smoke and subsequent primary campaign.
It does not approve a production implementation, establish performance benefit,
or replace the later semantic/recovery/methodology qualification.

## Spec and measurement assessment

- The six changed files contain only the new measurement apparatus, driver
  templates, apparatus controls and compact baseline evidence. No production,
  dependency, schema, ownership, scheduler, authority or cleanup code changes.
- Every arm uses the same targeted continuation approval loop, three-field
  progress and contact assertions, and per-operation wall clocks. Only the
  generated typed invocation helper selects Run or RunProgress
  (`docs/evidence/progress-scaling/execute.go.txt:1`,
  `docs/evidence/progress-scaling/measure.py:189`). Baseline compilation does
  not require the new API. Per-prefix report proofs and resource snapshots
  are removed equally; CPU/allocation measurements enclose Execute.
- Final full Run is separately measured. Completed full reports require zero
  work, quiescence, no added contacts, empty non-nil obligations and exact
  typed root equality (`completed.go.txt:34`). The driver transformation
  retains the initial identity proof and rich P2 final result/selection/
  approval/charge proof, adds completed-reopen checks, and compares independent
  Replay roots (`measure.py:104`). These checks are outside Execute and within
  lifecycle measurement. Progress is not treated as a historical certificate.
- The timing schedule has 147 serial child rows: 21 excluded warmups and 126
  measured rows, covering seven bounds, three arms and all six adjacent arm
  orders. Builds finish before timing children (`measure.py:57`, `:495`).
  Profiles and counters use separate builds/runs; ordinary timing has neither.
- Exact Git archives, source trees, module verification, SQLite contents,
  generated module inputs, tool identities and binaries are bound. Primary
  timing requires a clean checkout and matches executable apparatus/helpers
  to the candidate archive (`measure.py:340`, `:465`). Before/after input
  snapshots detect drift across the campaign (`measure.py:498`, `:518`).
- Failed child outputs/status are retained, then represented in durable row
  records; failed pairs do not receive timing estimates (`measure.py:303`,
  `:430`). The reused recorder writes outcome before post-run metadata and
  kills the process group on timeout. The RSS wrapper reaps its sole child
  before obtaining child resource usage. Execute-end cumulative RSS and
  child-exit RSS are explicitly distinct (`measure.py:235`, `:291`).

## Evidence checked

Read the submitted diff and inspected unchanged helpers only for the concrete
risks above: source/dependency custody, timeout/exit retention, child RSS,
resource scope, counter partitioning, and final identity/charge proof.
No Go suites, builds, or benchmark samples were rerun.

Read-only checks against this plan's retained raw directories established:

- Smoke (B1/B8), profile (B256), and counters (B8/B256) manifest, build-input,
  result, rows, source archive, binary, command, outcome, stdout and stderr
  hashes match `docs/evidence/progress-scaling/baseline.json`.
- All five raw rows pass the current report validator with their recorded
  identity and post-exit RSS, and match their compact workload/resource rows.
  The counter rows also pass the complete SQL partition checks.
- B256 counters report 397,884 Execute SQL executions, 233,344 in finishRun,
  and 65,536 roots emitted. These are diagnostic counters, not isolated
  allocation or CPU attribution.
- Both B256 profile files match their recorded SHA-256 hashes, decompress as
  gzip, and contain nonempty payloads. This does not independently validate
  profile-symbol attribution.
- Ordinary-suite raw log hashes match the compact metadata. Parsing those
  logs independently confirms 1,410 test passes, 18 package passes, four
  listed test skips and no fail events. The recorded command exit is zero;
  this review did not rerun that command.

## Limits and follow-through

The baseline is explicitly excluded and supports no comparative timing claim.
The progress arm has not yet been built or run at this reviewed head.
Candidate smoke, differential and recovery evidence, primary three-arm
timing, and separate candidate diagnostics remain required by later tasks.
Task 3 must add medians and observed ranges alongside the apparatus's per-block
contrasts (`measure.py:430`); the Task 1 report already assigns this work there.
Operator exclusion of unrelated builds/tests remains necessary: the local
measurement lock excludes only another P6 harness. Raw evidence is local,
not a public download.

The parent reported later supplemental profile rendering; that work is outside
the reviewed commit and is not part of this approval. No new CPU/allocation
attribution claim is endorsed by this review.
