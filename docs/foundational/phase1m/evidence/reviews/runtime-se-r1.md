# Phase 1M Task 3 runtime and causal-evidence review

Spec verdict: **APPROVE** for Task 3 / the milestone-1 runtime gate.
Code-quality verdict: **APPROVE**.
Actionable findings: **none**. No unresolved causal-evidence defect found.

Question: do the bounded C_se executions preserve G_answer and required joins,
with original records authenticated before normalization? The recorded evidence
supports that tested claim for the eight declared automatic samples and two
public schedules. This is not a universal scheduler, compiler, substitution or
recovery proof. C_reason remains outside this approval.

## Review scope and independent checks

Reviewed task-3-brief.md, task-3-report.md and task-3-diff.txt; assignment sections
3 and 5–7; the approved Task 2b qualification/bounds review; the three affected Go
test files; the reused raw validation and public driver; and the new capture,
audit, README, index and archived command/raw records. No subagents or commits
were used. No passing behavioral suite or audit script was rerun; the audit
script writes evidence outputs, so it was inspected rather than invoked.

Executed read-only `git show`, `git status --short`, `rg`, `sed`, `nl`, and Python
heredoc inspections of the archives. These independently recomputed source/file
hashes, parsed saved Go JSON event streams, checked source snapshots and wrapper
equivalence, and reconstructed original e/S raw links without calling the Go
decoder, model, or saved audit functions. The raw inspection checked original
emitting-history intent equality, retained handle, admission caller/operation/
inputs/template, accepted binding/request/observation, publication/delivery,
owning selected history and consumed sequence, and the predecessor's actual
receive operation register. All checked assertions passed. No concrete unresolved
doubt warranted another behavioral probe.

## Preservation and raw validation

`invocation_join_evidence_test.go:96–112` extracts the raw pass and keeps the
historical three checker calls in their wrapper. Independently compared the raw
body from `out.Handles` through `out.RawValid = true` against merge base
f5b52853ce3cf2b93563eed5eb8268234386e691: it is text-identical. The existing
`validateRetained` checks remain before normalization, including inverse
commitment/source-authority links, selected receive predicates, consumed-row
linkage, binding/publication payload checks and charge accounting. No old
safeguard is parameterized away.

The new wrapper (`consumer_policy_runtime_test.go:22–30`) changes only the
context prefix/guarantee verdicts; component conformance remains separate.
The historical-wrapper test at :345–394 opens the original Phase 1L archive
read-only and compares full decoded traces, handles and verdicts. The before
record and both final profile records contain exactly the same 48 comparison
entries/hashes. Historical sources and archives were not copied or modified by
this delta. The retained RED is an authentic S-first execution rejected by the
old `consume_order` policy, explicitly not a substitution counterexample.

The independent Python audit at `runtime/se/audit.py:58–172` establishes causal
facts from raw records and compiled receive operands; it does not obtain these
links from normalized Go events. Its only historical imports are three
hash-pinned inert function definitions extracted by AST (:16–27); historical
top-level execution is excluded. The saved audit command exits 0 and its source
snapshot/hash matches the current script. It separately checks verdict labels
after establishing raw evidence. This remains a bounded record audit, not a
second runtime implementation.

## Execution, schedules and controls

The automatic cases (:151–176) use the unchanged runner with A/B and exactly
id{1,2} × value{0,1}, answer. Independent archive inspection confirmed all eight
distinct configurations per profile. Both original correct present outcomes are
consumed once by the owning parent, S at sequence 3 and e at sequence 4; parent
and root succeed. Dynamic spending is A2/B3 under common I=R3, depth2, E0,
fixed1 and capacity1. Operational 200/200 and byte limits retain Task 2b's
separate artifact-bound justification rather than being inferred from samples.

Public schedules (:262–339) retain original handle identities and complete the
same Store normally after their checked cuts. Both normal and race records show:

| Schedule | Original publication orders | S-consumed/e-required cut | Final original consumptions |
| --- | --- | --- | --- |
| S first | S13, e37 | parent open PC3; e unproduced, member/request sequence0; spending2 | S3, e4 |
| Candidate first | e29, S36 | parent open PC3; e retained-unconsumed; spending3 | S3, e4 |

The cut acknowledgments are present in the retained journals (12 and 31 journal
events respectively). Causal summaries keep helper/member terminal freeze,
original publication, delivery observation and actual consumption distinct.
Original B completes through its helper; helper completion is not substituted
for e publication. Old consumed observations remain identical after completion.
Publication/delivery are atomic here; no published-undelivered cut is asserted.

The reused driver commits via public Store methods and lets the runtime evaluate
the actual instruction. It chooses service/run/observation scheduling, records
rejected observation probes without state changes, and does not compute a parent
decision or successor projection. There are no SQL mutations, sleep-based
causality or pending-release inputs in these executions.

Controls (:179–257) preserve candidate-pending at PC3 after S consumption and
S-pending at PC2 despite completed e being retained. Wrong-id remains raw/prefix
valid, retains and consumes its actual id2/value1 response, then fails after both
joins; component/G reject it. Budget2 authentically refuses the helper with
`invocation_limit`, zero refused debit and actual total spending2.

The decoder tests distinguish authentic opposite-policy histories from detached
corruptions. Both authentic histories remain raw-valid and fail the opposite
consumption policy. Saved detached failures are precisely
`selected receive guard/original handle mismatch` for coherently swapped
selections and `consumed observation lacks selected history` for delivery
promoted to consumption. Tests preserve original and altered inspections.

## Source-bound qualification and limitations

Reparsed archived stdout independently: normal and race each have 19 new leaves
and 39 affected historical leaves, exactly matching the saved selections, with
no failed/skipped final events and exit0. The historical selection includes all
19 decoder negatives, incomplete-prefix checks and existing recovery. These are
four separate passing commands, not a new combined or broad crash campaign.

All 367 current source paths match each final normal/race command's recorded
hashes and aggregate
ab2db041175840a7dea8c6175aeb7bc24cedcb44555d7aefad0e125f64a6ecbf.
All indexed file hashes match. Saved discovery, scoped integration vet,
three-file formatting (empty output), readonly module verification and toolchain
records exit0. The retained Go1.27.3 darwin/arm64 CGO1 commands specify local
toolchain, offline/readonly dependencies and fresh artifact/store directories.
Before/RED source differences are expected and are not relabeled final evidence.

No new C_se process-recovery claim is made. The historical crash predicate
retains its C_es scope. Stable identities/authority, valid bindings, pure bounded
inputs, no cancellation and eventual eligible service remain assumptions.
C_reason, all-mode progress/conformance and scoped manuscript reconciliation
remain later gates. This review permits proceeding to milestone 2; it does not
claim those obligations are complete.

Only this review artifact was written by the reviewer.
