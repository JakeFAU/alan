# Task 5 independent scoped review

Specification verdict: **PASS for Task 5**.

Quality verdict: **PASS for Task 5**. No actionable source defect found. One documentation handoff clarification is recorded below. Complete-source/global verification, historical archive reproduction, final evidence packaging, and whole-branch approval remain parent gates.

## Scope and preservation

Reviewed the four-file delta, task brief/report, independent conformance plan, complete 976-line bridge, external example builder and CLI/tests, relevant preserved model scenario construction and transition code, all six raw CONFORMANCE records, and reported normal/race/vet outputs.

Independently verified all **53** frozen source/dependency identities (byte count and SHA256): zero mismatches. All **13** pre-existing research Go files match the task base byte-for-byte. Existing runtime interpreter, authority, dispatch, persistence, replay, crash controller and dependency sources remain unchanged. Git's tracked docs/research delta contains only the parent's decisions document, not saved historical evidence. The added bridge intentionally changes dynamically generated current source manifests; it does not rewrite an earlier oracle or saved evidence.

The production change is ordinary fixture data plus CLI selection. ExternalProvision emits a declared external mutation, waits on exact authenticated receipt correlation, and supplies an independent peer. It adds no role/name dispatch, privileged recovery policy, opcode, grant shortcut, storage behavior or transport behavior. Configuration validation happens before Create. Tests exercise actual HTTP/resource persistence through runCLI, and label this separately from Task4 compiled-process crash evidence.

## Independent model relation

The expected side is driven by the preserved `step` and `inspect` with its carried monitor. Each model event must be accepted and each monitor check must pass, except the explicitly expected final source-closure rejection. There is no runtime Reduce/Replay call in the oracle. Runtime setup/advancement uses actual public APIs and ordinary programs; there is no direct SQL mutation, manufactured projection or shared policy decision implementation.

The independent adapters compare intermediate current generations/retirement, retained run statuses and terminal presence, supervisor expected generation/budget/pending/obligation/classification, exact request fields, separate source authorization, original result/reason/successor, and trusted observation binding/consumption. They check absence of extra lifecycle intents/results, not just matching a chosen row. The runtime operation map is explicit and stable across retransmissions/reopens; original raw identifiers and artifact bytes are retained.

Reviewed original scenario constructors against the selected prefixes and suffixes. The raw JSONL has six complete records, identical to the six verbose-log records in original order. Independently checked seed-prefix equality, runtime program artifact hashes, every recorded expected/actual equality, and the following trace lengths:

| Case | Original seed used | Total model events | Checkpoints |
| --- | ---: | ---: | ---: |
| Accepted result loss/successor | 25 | 28 | 12 |
| Interrupted committed request | 23 | 37 | 13 |
| Retained rejection after failure | 14 | 22 | 12 |
| Observation without target rights | 23 | 32 | 11 |
| External generations/late receipt | 38 | 43 | 30 |
| Closure before external authorization | 12-event prefix of J | 15 | 3 |

Audit output: `/tmp/alan-task5-review-evidence-audit.log`.

The accepted case repeats the original decision after reopen and commits actual generation-2 worker work. The rejection case first rejects an active worker under valid matching authority, then actually fails the worker, reopens, and repeats the same operation with unchanged rejection. A separately labeled ordinary static requester then creates a distinct accepted operation and the actual successor executes. This avoids stale-generation/missing-grant masking. The observation-permission case separately demonstrates successful source authorization and denied target lifecycle rights.

C uses the required exact ordered suffix: authorization, decision, result admission, Prepare selecting the same original result in inner/g1, Interrupt slot0, then Reactivate inner/g1. The stale runtime authority is rejected without changing the full inspection. The selected run's model/runtime epoch and retained sequence are checked at that boundary. The bridge does not claim an in-instruction runtime kill or equate all epochs.

## Representation differences remain explicit

The model lifecycle Receive consumes and clears pending atomically. Runtime Receive and Set are two ordinary commits. The bridge records the consumed-but-pending runtime checkpoint as **not equivalent and not a stutter**, with both views and the full snapshot. Independently verified the C raw record contains empty model pending and nonempty original runtime pending at that point. C also reopens there and validates the residual continuation before clearing. Equality is asserted only at the declared subsequent boundary.

Reopen validates the epoch increment for every open run and frozen epoch for terminal runs, then compares all remaining semantic tables unchanged. This is deliberately different from a model Reactivate selecting one run. It is disclosed rather than normalized into an unqualified whole-trace claim.

For J, the runtime worker adds labeled ordinary control input/branch transitions. `control-1`, generation1, mode `fail` reaches ordinary failure; `control-2`, generation2, mode `wait` reaches actual receipt PC3. These are runtime contract setup, not invented model events. The common relation checks selected public status/effect/lifecycle boundaries and does not claim compiler equivalence of the two worker programs.

The external bridge endpoint is in-process HTTP with its own durable resource database, explicitly distinct from Task4 process-isolation evidence. Authorize+Apply is compared after the combined public DispatchExternal call; no unavailable intermediate boundary is claimed. Endpoint reads supply test evidence only.

J actually emits distinct original keys `["worker",1,1,0]` and `["worker",2,1,0]` with the same business reference and delta. The retained independent endpoint evidence has two mutations, total2 and four contacts. At old receipt settlement, generation2 is genuinely waiting at PC3 for its own effect; old receipt consumption returns ErrIneligible without changing any inspected evidence. Its own receipt is then accepted/consumed and the successor finishes. Late old settlement leaves the old run failed and current worker/supervisor projections unchanged. The model lacks a new inbox entry on terminal receipt settlement, while runtime retains an unconsumed authenticated closed-run observation; the report explicitly names this ledger difference and the adapter does not assert inbox equality for it.

The separate source-closure case records the rejected model authorization event/reason, checks unchanged model state and monitor, verifies runtime source_closed/source_denied retention, zero HTTP contacts and no fabricated receipt, and preserves the frozen run across reopen.

## Verification and handoff note

Inspected actual full normal, race and vet logs; no additional full-suite rerun was needed for unchanged source. Race includes the new semantics bridge and CLI example tests. Task4's separately reviewed real-process evidence remains the basis for crash/process-isolation claims. No production source edits or subagents were used in this review.

**Nonblocking documentation clarification:** the task report suggests trying the other external profile with another fresh executor store. The public runbook should explicitly require a fresh **resource** store as well, or explain intentional key reuse. Both fresh built-in demo stores emit `["client",1,1,0]`; running the idempotent demo against an endpoint and then running a fresh non-idempotent demo against the same retained endpoint sends the same key with a changed profile. The resource correctly returns a descriptor conflict; it is not a fresh non-idempotent application. Source-derived reproducer (not separately executed): run the report's idempotent example to completion, keep that resource process/database, initialize a second executor database with the same URL and `--profile non-idempotent`, then run it. The second run remains unknown due to the HTTP409 conflict, and the endpoint retains only the first mutation. This is a runbook prerequisite/namespace limitation, not a request to change the stable-key protocol or add business deduplication. Parent was notified for the documentation gate.
