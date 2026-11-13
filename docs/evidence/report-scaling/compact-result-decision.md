# Compact execution result: defer this milestone

Assessed against Task 2 commit `6fadae8575bb1567f4d11f69edbb8258d5f3dce6`,
under the frozen [report materialization contract](contract.md).

## Decision and useful boundary

Keep the existing `Run` result and inspection workflow. Add no public type,
method, option, or validation cache in this assessment. Full-report batching
remains the production change. A compact API is possible, but there is no
demonstrated material benefit here from omitting output lists alone, and
skipping their associated validation changes the frozen integrity coverage.

An output-only compact result could retain validation while omitting roots,
open runs, unresolved/unknown lists and pending delivery keys. It could save
output allocation, copying and sorting. These savings are not measured by
this assessment and are not proved negligible. Such a result still requires
the current global reads and decodes unless those validators are separately
refactored or their previous results can soundly be reused. Simply calling
`finishRun` and then discarding fields saves none of its construction work.

Task 2's excluded B8 diagnostic reduced finishRun statements from 708 to 348;
332 other report statements remained in each arm. Its excluded B256 candidate
still allocated 3277.285 MiB across execution and emitted the required 65,536
root records. Those figures concern full reports, include work beyond returned
arrays, and are not a compact-result experiment or paired timing conclusion.
They do not identify how much allocation a streaming validator would save.

The least powerful justified boundary remains the existing full result.
Introducing a second public contract just for unmeasured thinner arrays is
not justified in this slice. Avoiding repeated global validation would require
a new proof of validation reuse or an explicit reduction of retained-history
coverage; neither is part of this optimization. This is a bounded deferral,
not a claim that every useful compact implementation needs a certificate.

## Validation and output are distinct

| Work | Current owner | Compact assessment |
| --- | --- | --- |
| Worker cancellation/join, draining completions, durable result accounting | `Run` cleanup defer and `accountExternal` | Must remain before any result is returned. |
| Authority, reservations, admission, original identities, receipt correlation, settlement and publication | Existing transaction/dispatch/result paths | Must remain authoritative; a report summary cannot replace them. |
| Root preparation, reached freeze/closure/descendant checks and member effect results | `readRootCompletionsWithReader`, `prepareRoot` and authoritative leaf loaders | Validation is mixed with root construction; published and pending short circuits must remain exact. |
| Every retained run projection, global intent identity, unresolved attempt/result checks | `finishRun` and existing loaders | Current retained-data error coverage remains even when an item is unrelated to this call's progress. |
| Pending invocation readiness, terminal JSON, historical admission-result JSON, charge aggregation | Existing report traversal | Some are diagnostic reads, not authorization. Dropping them still changes the frozen integrity/error contract. |
| Returned lists, root records, classifications, ordering, rejected count | Report construction | An explicit smaller output contract can omit them, but cannot silently omit the preceding validation. |
| Fresh Open, Replay, typed inspection | Independent evidence validation | Stay independent. A caller's later inspection is not a replacement for the current Run's report-time checks. |

`staticExecution` only supplies disposable discovery state. Its guard is not
a report-validation certificate. A reuse design would need an explicit initial
validation boundary, a complete map of trusted writes and invalidations, and
fallback before an intervening unsupported mutation could be acknowledged by
a later legitimate write. This assessment neither adds nor proves that design.

## Executable counterexample and genuine positive control

`TestRunCompactAssessmentGenuineHistory` creates two waiting composition roots,
admits each input through `AdmitInput`, and completes each through ordinary
`Run`. Each consumes three steps, zero external dispatches, and the expected
input value. The test checks retained admission/observation counts and then
independent Inspect, Open and Replay. No manually inserted evidence supplies
this positive history.

`TestRunCompactAssessmentRetainedAdmission` first completes root r00 in the
same way. It then deliberately corrupts exactly one retained historical
admission result with an **unsupported private SQL UPDATE in a disposable
test store**. A legitimate input for r01 is admitted afterward. Root r01
completes in three steps with zero dispatches and the correct value, while
full Run returns the decoder's `io.ErrUnexpectedEOF` for r00's admission.
The existing `Quiescent=true` assignment survives the later reporting error.

The reproducible Go overlay changes only the deferred `finishRun` call to
the same quiescence assignment. Worker join, result accounting, scheduling,
and transaction validation remain intact. Under this deliberately rejected
hypothesis, the genuine positive control still passes, but the corruption
witness fails because Run returns nil error. The RED therefore discriminates
lost reporting coverage from a broken fixture or failed unrelated execution.
It tests a bypass hypothesis, not an implemented compact API.

This witness does **not** demonstrate unauthorized dispatch, a safety failure
in a supported public history, a hostile-external-writer guarantee, or
insufficient validation for every possible smaller output contract. It
demonstrates that current execution checks alone do not imply the report's
global retained-data diagnostic coverage. Existing malformed retained receipt
tests separately pin result validation and joined settlement errors; the
admission witness must not be substituted for those safety checks.

## Reproduction and remaining question

From the repository root, with a fresh output directory:

```sh
python3 -B docs/evidence/report-scaling/compact-assessment.py \
  --root /private/tmp/alan-compact-assessment \
  --gocache /private/tmp/alan-task3-go-cache
```

The script leaves production sources untouched. It records source hashes,
toolchain, exact commands, raw logs and actual exits. Ordinary, integration,
integration-race and overlay-positive checks must exit 0; the overlay RED
must exit 1 with the specific lost-diagnostic message and nil error. These
are correctness checks, not performance measurements. The tests also run
directly with `go test -run '^TestRunCompactAssessment' ./internal/executor`.

The remaining question is whether a validation-preserving streaming traversal
has a measured allocation/time benefit sufficient to justify another result
contract. If the desired saving instead depends on avoiding global reads,
first specify and independently validate a reuse protocol or explicitly
authorize a different integrity contract. Until then callers continue using
ordinary full Run and the existing independent inspection/reopen/Replay APIs.
