# Retained-query scaling implementation plan

> Use superpowers:subagent-driven-development, with independent task review.

**Goal:** Determine whether the remaining history-insertion scan is significant
and avoid its repeated work only if a narrow, measured correction is justified.

**Spec:** [contract.md](contract.md).

## Global Constraints

- Baseline `9db473bae28e5b806c3ccdc7e2e134c1d0b38f27`; preserve all Run,
  RunProgress, Open, Replay and InspectExecution contracts.
- Preserve authorization, approval, eligibility, reservation, admission,
  accounting, settlement, recovery and cleanup checks and existing test expectations.
- No public API, dependency, schema, index, migration, cache, scheduler,
  ownership, retry or BoundedSeries change.
- One conditional production change: adjacent history/consumption write order
  in Store.Commit. Do not expand to the two other scans to rescue a weak result.
- Preserve evidence and negative findings; draft PR only, no merge/release.

### Task 1: Measurement apparatus and baseline diagnosis

Create `docs/evidence/retained-query-scaling/measure.py`, its focused tests and
minimal new templates if needed. Reuse tracked P6 and older helpers unchanged;
never write another plan's ignored workspace. Read this plan's semantic and
measurement audit reports. Capture current environment, dependency/toolchain,
exact baseline/tree and fresh ordinary suite before production edits.

Provide frozen-source smoke, counters/profile and paired timing lanes. Both
arms use progress, including the same explicit final full audits. Bounds
1,8,16,32,64,128,256; one warmup and six balanced AB/BA samples per arm. Validate
source/output binding, schedule, failed-row retention and unique transforms.
Use the pinned SQLite counter machinery to establish the baseline, and a
separately labelled disposable source-order overlay if useful to discriminate
the causal hypothesis. It must never masquerade as the production revision.
Attribute the three suspect queries' time with disclosed aggregate diagnostic
instrumentation or profiles. Keep primary timing uninstrumented and serial.

Record the diagnostic evidence and a candidate/no-go decision. Commit apparatus
and baseline records, independently review specification and methodology. If
the candidate is unjustified, skip Task 2, document the negative result and
complete the applicable Task 3 delivery path without fabricating qualification.

### Task 2: Atomic write-order correction, conditional on Task 1

Only production file: `internal/executor/store.go`. Add focused tests in new
test files. Insert history before setting consumed_sequence, preserving all
existing validation, other writes, callbacks, ownership and transaction edges.
Explain the local foreign-key reason in a short code comment.

Add discriminating rollback tests for each adjacent write and later failure,
failed deferred-FK COMMIT cleanup, no selection and legitimate selection,
duplicate/foreign/stale evidence. Use independent reader and acknowledged real
subprocess SIGKILL at both staged-write cuts and after durable commit. Validate
reopen, inspection, Replay, original IDs, no partial consumption and repeated
reopen. Keep test hooks outside production and original tests unchanged.
Use existing helpers rather than a new fault-injection subsystem.

Run targeted ordinary/race and affected integration/compatibility tests. Commit
after self-review, then independent semantic/recovery/compatibility review.
Record actual failing-before controls where the new test specifies the measured
work reduction; semantic safety tests should also pass the baseline.

### Task 3: Frozen qualification, evidence and delivery

Freeze executable runtime/tests/harness. Run the paired matrix, deterministic
counter comparison, profiles as useful and same-seed semantic/recovery checks.
If latency gain is within variation, report a negative latency result and stop
without expanding scope. Retain baseline/candidate contact and charge proofs.

Run full ordinary/integration/race-integration, focused race, required historical
readers, nested consumers, build/vet/module/format/Staticcheck and actual weather
recovery cuts on the exact source. Reuse tracked custody and corrected P6 reader
comparison; never relabel earlier outcomes. Verify source before/after hashes.
Document skips/failures/corrections, allocation and memory tradeoffs, remaining
quadratic work and whether this is worth further optimization.

Commit compact data, reports and reproduction commands, with a source bridge
for evidence-only changes. Obtain task and whole-branch review, resolve
substantive findings, push the new branch and open a draft PR against verified
main. No merge/release. If no production change survives the stopping rule,
qualify only the delivered apparatus/docs and clearly bound those checks.
