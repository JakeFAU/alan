# P7: residual retained-query work

Frozen before production changes at verified main
`9db473bae28e5b806c3ccdc7e2e134c1d0b38f27`, tree
`56669e93b11a35ab8a67d09918b3a5e055944015`. PR #17 is merged and its
delivery tree is identical to this baseline. The user authorized continuing the
bounded scaling investigation. Deliver a draft PR; do not merge or release.

## Question and first experiment

P6 removed the implicit full report from each progress call. Its B256 execution
still recorded 227,970 SQLite full-scan steps. Three templates account for all
of them: history insertion (130,305), authorization-result lookup (65,280), and
root effect-charge counting (32,385). The B8/B256 counts are consistent with
quadratic growth. They are not measurements of how much wall time those scans
consume. Establish that distinction before choosing a production change.

Investigate history insertion first. `Store.Commit` currently updates the
selected observation's `consumed_sequence` before inserting the referenced
history row in the same transaction. The pinned SQLite foreign-key program
can skip a parent-insert child scan when its deferred violation count is zero.
The present order appears to create and then discharge a temporary violation.
Test that explanation against real Store execution and the pinned SQLite,
including a disposable source-order counterfactual if needed. Do not call the
explanation established until the counters discriminate between the orders.

The conditional candidate is only to insert the history row before updating
consumption. All validation remains before these writes; all other writes and
the existing commit boundary remain unchanged. No persistent summary or cache
is introduced. Do not proceed to the other two scans to rescue a weak result.

## Necessary work, repetition and boundaries

Source authority, eligibility, reduction, protected continuation checks,
foreign-key enforcement and atomic commitment are necessary work. Repeated
searches for a history parent that is about to be inserted may be avoidable
inside the transaction. Neither historical validity nor authority is reused.

The proposed reuse condition is therefore not a cache predicate: both rows
already pass the existing validation and are written in the same owned SQLite
transaction. A failure, cancellation, failed COMMIT or process death before
commit must retain neither write. A successful commit must retain both, with
the same identities, sequence, projection, intents, resources and original
observations. Concurrent supported readers must see a committed cut. Fresh
Open, Replay and inspection independently validate the unchanged retained facts.

SQLite triggers/hooks can observe statement order inside a transaction. Custom
schema mutations and arbitrary SQL writers are not a newly supported API.
Test-only hooks may expose the intermediate state for fault injection; they
must not be described as production callbacks or durable evidence. The
application boundary callbacks stay in their original positions.

Other candidates have different risks: looking up authorization by its primary
key must still check the retained intent relationship; root-charge counting
enforces an expenditure boundary and cannot be replaced by a process-local
allowance assumption. Neither is included in this first experiment.

## Invariants and falsifiers

- No public API, dependency, schema, table, index, migration, ownership,
  scheduling, retry, provider or BoundedSeries change.
- Preserve every original ID, generation, approval/grant, selected observation,
  independent charge, contact admission, retained result/conflict and unknown
  external consequence. No refund or new authority after a crash.
- Preserve validation and error boundaries reached by ordinary execution,
  including stale authority, duplicate/foreign consumption and malformed
  retained evidence. Keep existing tests and expectations unchanged.
- Roll back both adjacent writes on either statement's failure and on later
  failure; failed foreign-key COMMIT must not leak the staged transaction.
- Actual SIGKILL after either staged write must recover the previous committed
  cut. After commit it must recover the whole reduction. Repeated reopening
  must not duplicate consumption, expenditure or physical dispatch.
- Preserve supported historical formats and existing reader version rules.
  Do not assert every possible private schema has identical statement effects.
- Stop if the measured cause differs, a required semantic change is discovered,
  correctness requires relaxing a check, or the experiment needs broader work.

## Measurement and decision

Use the same public deterministic P6 workload and its equal final full report,
typed inspection, reopen and Replay checks. Both arms use RunProgress. Freeze
the actual source identities and harness; never relabel P6 timing as P7 timing.
Record baseline environment, dependencies, ordinary tests and new diagnostic
measurements before production edits. Separate instrumented counts/timing and
profiles from uninstrumented primary timing. Low-overhead attribution may
aggregate time for only the three suspect SQL templates; disclose its boundary
and overhead rather than treating it as a primary latency result.

For a viable candidate, compare bounds 1,8,16,32,64,128,256 in separate serial
processes, one excluded warmup and six measured samples per arm/bound, balanced
AB/BA blocks. No concurrent builds/tests during primary timing. Retain every
row and failed attempt. Report wall/CPU, allocation bytes/count, peak RSS,
statements/rows/VM/full-scan counts and actual results/contacts/charges. Include
the complete lifecycle and separate reopen/Replay costs. A larger diagnostic
bound requires a disclosed separate purpose; it cannot replace this matrix.

Distinguish a reproducible reduction of structural work from a latency benefit.
For the disposable candidate pilot, the advance criterion is a median paired
Execute reduction of at least 1%, with at least five of six B256 pairs faster
and median paired savings greater than their median absolute deviation. This
is an engineering screening rule, not a confidence interval or a promised gain.
Also check B128 and smaller bounds for material regression. Freeze this rule
before the pilot; do not select a friendlier rule after seeing its results.
If timing improvement is within variation, report that negative result, stop
this optimization sequence and do not broaden the patch. A mechanically smaller
scan count is not sufficient evidence of a meaningful application speedup.

## Qualification and delivery

Before delivery, qualify the exact clean executable revision: focused ordinary
and race, full ordinary/integration/race-integration, real process cuts,
historical readers, build, vet, formatting, module verification and applicable
Staticcheck. Include nested public consumers. Compare same-seed baseline and
candidate retained semantics, including failure/uncertainty and independently
approved later work. Test rollback, committed-cut readers, stale evidence and
copied/corrupt stores at the affected boundary. No exactly-once remote claim.

Independent review covers semantics, recovery, methodology, compatibility and
scope. Preserve rejected findings, raw local artifacts, exact commands, exits,
skips and source hashes; commit compact reproducible evidence and an explicit
source bridge for evidence-only commits. Do not imply local raw archives have
been published. An honest profiling/negative-result PR satisfies the stopping
rule; it must state which work was not justified or performed.
