# Task 2 independent specification and quality review

Verdict: **APPROVE**.

- Specification compliance: **APPROVE** for Task 2.
- Quality, semantics, recovery and public API: **APPROVE** for Task 2.
- No blocking findings. This is not approval of Task 3 qualification or a merge.

Reviewed base `5ed2d73a30c9e17c2626ff6e65d915e304e87673` to head
`65a998b81e3b2c429672edcc202938eb3cdcffc9`. HEAD matched the requested
commit and tracked status was clean at review entry. Read the supplied diff,
Task 2 brief/report and binding `docs/evidence/progress-scaling/contract.md`.
Inspected unchanged owning source for the specific risks below. The authorized
bounded-series documentation cross-reference is included in scope.

## Specification and semantic findings

1. The implementation does omit only the final report traversal. `Run` forwards
   to `run(..., true)`; `RunProgress` calls the same engine with false and projects
   its three counters (`internal/executor/runner.go:64-78`). An independent
   source comparison against the base confirmed that the former Run body is
   byte-identical after substituting the one final-report choice, and all source
   from `deliverOrdinary` onward is byte-identical. The public options/report
   types are unchanged. Nine changed files comprise the runner/API, weather
   call, documentation and new tests; no existing tests were modified.
2. The full-report path preserves its quiescence and partial-error behavior.
   The new assignment at `runner.go:132` calculates the same predicate previously
   passed directly to `finishRun`. The unchanged helper sets that flag before
   initializing report slices and collecting report data (`runner.go:560-580`),
   and joins report errors without resetting it. No report fields, collection
   ordering or nil/empty initialization changed.
3. Execution still performs eligibility, preparation, final dispatch admission,
   commit and root settlement before choosing quiescence (`runner.go:193-303`).
   In particular, the removed traversal was after those transactions and after
   result accounting. It cannot replace a reached execution gate. The unchanged
   attempt/result loaders reject malformed encodings (`adapters.go:105-126`),
   and `retryEligibleWork` calls them before charging another dispatch
   (`runner.go:192-207,683-711`). The new malformed-evidence test retains an
   actual result first, corrupts precisely that reached evidence, and requires
   ErrCorrupt, zero new counters, no extra callback and no settlement
   (`runner_progress_integration_test.go:162-203`).
4. The narrower diagnostic scope is explicit and tested. The historical
   admission witness advances independent work, then requires full Run's EOF
   diagnostic with quiescence still true, plus InspectExecution/Replay/fresh
   Open rejection (`runner_progress_test.go:105-143`). The unchanged full report
   still decodes all admissions (`runner.go:662-680`). There is no claim that
   progress validates an entire history or reproduces all corrupt-store errors.

## Recovery, concurrency and evidence findings

1. Both methods register under the same owner mutex and use the same done,
   cancel and wake state (`runner.go:88-103`). Cleanup cancels workers, joins
   them, drains every active completion, accounts under the existing detached
   five-second context, and only then releases the runner (`runner.go:120-142`).
   `Close` waits for that done signal before closing database and ownership
   handles (`store.go:707-742`). No extra concurrency mechanism was introduced.
2. Received-result handling retains its detached bounded context and joins
   observer, accounting and caller-cancellation errors
   (`runner.go:714-727`). `accountExternal` preserves the distinction between
   transport error and sticky handling failure (`external.go:239-252`), while
   adapter result recording precedes separate settlement
   (`adapters.go:326-359`). The new tests exercise cancellation after dequeue,
   retention before settlement, joined errors and recovery with no further
   callback (`runner_progress_integration_test.go:99-160`). Cross-method
   exclusion and Close use explicit waiting/callback barriers and verify
   retained settlement after release (`runner_progress_integration_test.go:207-310`).
3. Positive parity uses genuinely created stores, closed seed copies and fresh
   Open on both branches, not fabricated result rows. Complete typed snapshots
   and independent Replay reports are compared
   (`runner_progress_test.go:18-66`). Static adapter success, refusal and unknown
   outcomes preserve authority/reservation/admission/charges and selection;
   the dynamic fixture exercises fallback discovery and two published
   invocations (`runner_progress_integration_test.go:24-97`;
   `runner_progress_test.go:69-103`). Unknown effects remain unresolved after
   reopening without another contact.
4. The corrected unknown-result assertion is justified, not a weakened
   invariant. An admitted opportunity does not prove that the callback ran:
   `transportAdapter` still has pre-call exits (`adapters.go:301-309`). Unknown
   candidates do not create result evidence (`adapters.go:326-331`), and the
   inspection minimum becomes one upon retained qualifying result evidence
   (`inspection_effect_results.go:27-33`). The test separately requires durable
   dispatch bounds 0..1, the retained admission, and exactly one observed callback
   per arm (`runner_progress_integration_test.go:36-69`).

## Public API and scope

The runtime alias exposes the requested signature and the external-package
example includes a compile-time signature check (`runtime/runtime.go:32-33`;
`runtime/progress_example_test.go:69`). The executable example demonstrates a
quiescent but open input wait and explicit inspection. The guide documents
per-call counters, partial progress on error, local work counted as dispatch,
the unrelated-history diagnostic difference, and no completion or remote-finality
guarantee (`docs/execution-progress.md:3-36`). Weather changes only its
discarded-report invocation (`examples/weather-agent/main.go:170-184`). No
approval-policy, BoundedSeries API, schema, dependency, retry, scheduler,
ownership or persistent cache change appears in the reviewed scope.

## Verification and limits

This review independently checked the source equivalence described above and
the absence of changed pre-existing tests. It did not rerun suites: no concrete
remaining doubt justified duplicating the implementation's reported focused
ordinary/integration/race, report-oracle, build and scoped Staticcheck runs.
Those execution results are attributed to `task-2-report.md`, not presented as
new independent test runs. The report also records the initial missing-API RED
and the diagnostic-boundary mutation failure; this reviewer did not recreate
those historical runs.

Task 3 still must supply the frozen full integration/race campaign, archived
real-process crash and historical-reader evidence, paired measurements and
independent methodology/recovery qualification. In particular, this review
does not treat those promised results as passed, establish a speedup, prove
arbitrary corrupt-store equivalence, or claim exactly-once remote effects.
