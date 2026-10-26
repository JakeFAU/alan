# ALAN Phase 1D.1: Bounded Asynchronous Dispatch and Independent Progress

Implement a focused correction to ALAN’s durable executor.

Phase 1D established a runnable single-host executor with persisted state,
ordinary supervision, lifecycle-operation identity, replay, and actual
process-crash qualification.

A subsequent source review identified a scheduling defect: the automatic
runner synchronously waits for external HTTP work before selecting further
ordinary computation. The existing timeout test establishes eventual peer
progress after the request ends, not progress while the request is in flight.

This phase must repair that behavior while preserving Phase 1D’s durability
and recovery contracts.

This prompt authorizes implementation, testing, and independent review.
Write a concise plan, then execute it. Do not return only another design
document.

## 1. Objective and decisive acceptance test

An external operation must not prevent unrelated eligible agents from
committing ordinary work.

The decisive execution is:

1. Agent B commits an external intent.
2. The endpoint receives the authorized request and deliberately withholds
   its response.
3. While that exact request remains in flight, an independently eligible
   agent C commits useful work.
4. C's message or terminal outcome becomes available to its ordinary caller.
5. Only afterward does the controller release B's response.
6. B's result is durably recorded and its computation continues correctly.

The passing path must not depend on B's HTTP request timing out, being
cancelled, or completing first.

“C eventually succeeds after B times out” is not acceptance.

Use the actual automatic runner. The controller must not call Commit on C's
behalf or start another executor to manufacture independent progress.

The relevant concurrency is between separate logical executions. Implementing
multiple continuations within one run is not required for this repair.

## 2. Establish the actual baseline and reproduce the defect

Inspect the current repository, working tree, branch, and project instructions.
Preserve unrelated changes. Verify the current Phase 1D implementation rather
than assuming its branch or PR state.

The reviewed implementation was commit:
121b504591509e1879b9cdd7eea0eb1b6280d613

The reported paths were:
- internal/executor/runner.go
- internal/executor/external.go
- internal/executor/external_test.go

The reported call chain was:
Store.Run -> DispatchExternal -> HTTP request/response -> selectReady

Trace the current code and confirm or correct that diagnosis.

Before changing the implementation, add a regression that demonstrates
whether a peer can commit while the external request remains held open.
Retain behavioral RED against the affected baseline and GREEN against
the correction.

Use explicit endpoint and commit acknowledgments. A watchdog may fail a stuck
test, but elapsed time must not be the evidence that the desired ordering
occurred.

If the current branch already contains a correction, verify it against the
same contract and extend missing tests instead of duplicating implementation.

## 3. Implementation direction

Preserve the single-host, single-active-executor, SQLite-backed profile.

Prefer a small scheduling/commit loop with a bounded set of external transport
workers. Workers perform network work and return correlated completion
candidates. Durable validation and state advancement continue through the
existing authoritative paths.

A different implementation is acceptable if it satisfies the same contracts
with less complexity. Do not introduce another orchestration framework,
broker, daemon, or storage backend.

### Bounded dispatch

Provide a positive configurable maximum for in-flight external work, with a
documented conservative default.

The limit must bound actual active work, not just a channel whose producers
can create unlimited blocked goroutines. Bound any additional queue or leave
undispatched work in the existing durable intent ledger.

Reserve capacity without blocking the scheduler. If all external slots are
occupied, continue eligible local computation, internal delivery, lifecycle
processing, and completion handling.

Within the active runner, the same intent must not accidentally occupy
multiple transport slots. Distinguish intended recovery retries from duplicate
dispatch caused by repeated scans.

Keep one scheduler owner per store. Reject overlapping Run invocations if
necessary rather than letting independent in-flight maps compete.

### Isolation from network waiting

Do not hold a store-wide mutex, SQLite transaction, or scheduler control path
while waiting for:

- An external response or response body.
- A worker slot.
- An external worker to finish.
- Space in a completion queue that prevents unrelated scheduling.

Short serialized database operations are acceptable. Serializing all agent
progress behind network latency is not.

### Preserve the durable protocol

Retain these distinct boundaries:

intent commit
-> attempt authorization
-> external application
-> receipt persistence
-> ordinary receipt consumption

No worker may dispatch an uncommitted or unauthorized intent.

A goroutine starting is not durable authorization. A completed HTTP call is
not a committed observation. A volatile queue is not the recovery ledger.

Workers must carry immutable operation/attempt correlation. Completion
processing must validate the retained descriptor and authorization before
recording evidence or making an observation consumable.

Keep supervisor policy ordinary. Do not add Control, Supervisor, or Worker
runtime types or role-name branches.

## 4. Scheduling, waiting, and lifecycle semantics

Define the runner's behavior explicitly before qualifying it.

When runnable local work exists, perform it without waiting for external
transport completion. Use a simple scheduling discipline that gives local
commits, internal deliveries, and completed receipts opportunities to advance.
Do not replace external head-of-line blocking with perpetual delivery priority.

When no immediate work is eligible but requests are in flight, wait for a
completion, cancellation, or another supported wakeup. Do not busy-spin and
do not report Quiescent=true while locally owned requests remain in flight.

Preserve the distinction between:
- No currently eligible local work.
- In-flight transport.
- Retained unresolved external outcomes.
- Application completion.
- Runner budget exhaustion or cancellation.

A request timeout or transport cancellation is not proof that the endpoint
did not mutate. Preserve the existing idempotent and non-idempotent recovery
profiles. Do not add blind retries for uncertain non-idempotent operations.

An authorized request may finish after its source run closes. Retain valid
late evidence without reopening the run or advancing another generation.
Do not reject an authentic old receipt merely because its originating
activation epoch is obsolete; stale computation commits and late external
evidence have different contracts.

Actual observation consumption order must remain recorded or reconstructible
for replay. Concurrent completion does not require deterministic live network
ordering.

Define how MaxSteps, MaxDispatch, and the in-flight limit interact. Reserve
budgets before launching work so concurrent workers cannot overspend them.
Scanning, polling, or receiving a completion must not accidentally spend a
second dispatch budget unit for the same launch.

Define shutdown ownership:
- Stop launching new work.
- Cancel or drain local transport according to an explicit bounded policy.
- Account for received completions or retain their unresolved obligations.
- Join owned workers before closing the store or releasing exclusivity.
- Do not leave background goroutines accessing a closed store after Run returns.

Keep Run-context cancellation distinct from logical CancelRun/RetireSubject.
Stopping the executor must not silently cancel every application execution.

## 5. Required tests

### A. Held request with independent progress

Implement the decisive regression from section 1.

The endpoint acknowledges receipt of one exact request and holds it open.
A peer must then commit and produce its declared observable result before
the endpoint response is released.

Record that no response, timeout, or transport cancellation completed the
held request before the peer commit. Confirm the final receipt is correlated
and consumed after release.

Run this through the automatic CLI in at least one subprocess integration
test. An in-process test may supplement it.

### B. Saturated external capacity

Fill all configured external slots with deliberately held requests and leave
additional external work pending.

Verify:
- Active transport never exceeds the configured limit.
- Extra requests do not reach the endpoint prematurely.
- Ordinary peer computation and internal delivery still advance.
- Releasing one slot admits eligible pending work.
- Scheduler rescans do not duplicate an already in-flight intent.

Include capacity one. A single external slot must not imply a single blocked
scheduler.

### C. Reordered completions

Launch two distinct authorized external operations and release their responses
in the opposite order.

Verify exact intent/attempt/result correlation, independent receipt
persistence, and correct ordinary consumption. The faster response must not
wait for the slower operation merely because it was dispatched second.

Replay the recorded executions without endpoint contact.

### D. Lifecycle progress while external work waits

While an external request remains held open, let an ordinary supervisor
process a separate worker failure and advance its existing lifecycle protocol.

Use a real ordinary program, not controller-injected policy decisions.
Distinguish any environmental setup or cancellation stimulus.

Also exercise source closure around authorization:
- Closure before authorization prevents dispatch.
- A previously authorized operation may finish after closure.
- Its late receipt settles the old obligation without reviving the run.

### E. Shutdown and runner limits

Exercise context cancellation and configured budget exhaustion with active
requests and queued completions.

Verify worker ownership and cleanup, absence of post-close store access,
accurate unresolved reporting, and recovery from retained records on reopen.

Do not silently discard a received outcome and report successful settlement.
If persistence cannot complete, retain uncertainty and expose the error.

### F. Timeout and unavailable endpoint

Retain the existing timeout/Unknown behavior, but verify that unrelated work
progresses before the failed request finishes when it is otherwise eligible.

An ordinary timeout must not become a process-wide application failure.
Fatal storage or corruption errors must remain explicit rather than being
swallowed to keep the scheduler running.

## 6. Crash recovery under concurrent dispatch

Keep the existing Phase 1D subprocess crash suite intact and add focused
coverage for the changed architecture.

At minimum:

1. Hold an external response after endpoint application.
2. Observe unrelated peer commits while that request is in flight.
3. SIGKILL the executor before its receipt is durably recorded.
4. Keep the external resource process and database alive.
5. Start a new executor against the same runtime store.
6. Verify the retained intent, authorized uncertainty, peer history, and
   correct endpoint-profile recovery.

For the idempotent profile, use the original operation key and verify the
retained logical mutation outcome. For the non-idempotent profile, preserve
Unknown rather than automatically repeating the mutation.

Also cover a response received by a worker but not yet durably recorded.
An in-memory completion queue must not be mistaken for committed evidence.

The accepted-Retry supervisor crash case, retained-rejection case, second
crash during recovery, terminal-notification recovery, and replay-no-dispatch
checks must continue to pass.

Do not combine all components into one transaction or fake endpoint state
inside the runtime database to make these tests easier.

## 7. Validate the new test oracle

Retain the original synchronous-dispatch behavior as a reproducible baseline
or test-only negative control. The unchanged held-request test must reject it.

A false positive includes:
- The peer committed before the endpoint began holding the request.
- The request timed out before peer progress.
- The controller directly advanced the peer.
- A different request supplied the hold/release acknowledgment.
- The test observed only final state after Run returned.

Correlate every decisive event with scenario, process/start token, subject,
generation, intent, and relevant durable sequence.

Establish ordering through explicit handshakes, not comparisons of clocks
from different processes. Watchdogs fail tests; they do not establish progress.

Retain any discovered runtime or oracle defect with its original input and
pre-fix failure. Do not weaken the acceptance condition until the existing
implementation passes.

## 8. Scope

This is a focused Phase 1D repair.

Keep:
- Go and the existing SQLite dependency/storage profile.
- One host and one active executor per store.
- Static subjects and existing grants.
- The current single-PC-per-run representation.
- Existing operation identities, replay, and lifecycle policies.

Do not add providers, a public SDK, a UI, multi-host execution, dynamic
creation, ownership transfer, history compaction, or general schema migration.

Do not bundle the separately disclosed cross-store operation-key namespace
limitation into this repair.

No broad literature pass is required. The attached research discussion
motivates independent progress; it does not impose an unproved equivalence
theorem or require a new calculus.

Avoid a large abstraction framework for a small dispatch problem. Do not
solve bounded concurrency by spawning one goroutine for every retained intent.

## 9. Execution, evidence, and delivery

Use a delegated implementer and an independent reviewer when available.
Have one owner coordinate the coupled runner, dispatcher, completion, and
shutdown changes.

Produce:
- The corrected runtime and CLI options.
- The original failing held-request regression and passing correction.
- Saturation, reordered-completion, shutdown, and crash tests.
- A concise dispatch/progress contract and updated runbook.
- Source-linked machine-readable traces and qualification results.
- An exact source/dependency manifest and reproducible source archive.

Run focused tests, the complete suite, race detection including actual child
executables, vet, formatting, diff checks, and prior research regressions.

Concurrent runtime records need not be byte-identical across executions.
Verify causal ordering, durable invariants, and the declared final
projections; preserve raw evidence rather than sorting away races.

Report historical baseline reproduction separately from current-source
behavioral comparisons. Do not rewrite old qualifications with a new source
identity.

The independent reviewer must examine:
- Network waits and lock/transaction ownership.
- Saturation and completion-queue backpressure.
- Duplicate in-flight claims.
- Shutdown/store-close races.
- Late receipts and source closure.
- Whether the decisive test can pass without actual independent progress.

Publish the repair on an appropriate branch or verified continuation of the
existing Phase 1D work, and open/update a draft PR. Do not merge or release
automatically.

Start the final report with the decisive observed sequence:

endpoint holding request
-> independent peer committed
-> peer result delivered
-> endpoint response released
-> original result settled

Then state the exact source identity, tests run, retained defects, and any
remaining limits.

Success is independent progress during external waiting, with the existing
durable execution contracts preserved.