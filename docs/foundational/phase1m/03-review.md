# Phase 1M independent review

All Tasks 1–6 have independent specification and quality **APPROVE** within the
separate scopes below. Review agreement is a source-bound quality gate, not a proof.
The frozen contracts, captured READMEs and task reports retain capture-time pending
wording; this disposition records later gates without rewriting those artifacts.

| Gate | Review | Approved scope |
| --- | --- | --- |
| Task1 | [construction](evidence/reviews/task1-r1.md) | C_se public construction/two-sample preflight |
| Task2a | [reference](evidence/reviews/model-se-r1.md) | independent C_se reference/freeze before census |
| Task 2b | [qualification](evidence/reviews/model-se-qualification-r1.md) | K1-first answer census, local subset, mapping/bounds |
| Task3 | [runtime](evidence/reviews/runtime-se-r1.md) | authenticated C_se samples, publication orders, pending cuts and regressions |
| Task 4 | [construction/review and rereview](evidence/reviews/task4-r1.md) | C_reason construction/preflight after report-only R1 correction |
| Task 5a | [reference](evidence/reviews/model-reason-r1.md) | independent reason reference/freeze before census |
| Task 5b | [qualification](evidence/reviews/model-reason-qualification-r1.md) | all-mode K1-first products, local progress, artifact mapping/bounds |
| Task 6 | [runtime](evidence/reviews/runtime-reason-r1.md) | raw selected envelopes, actual classification/End/pending paths, controls and audit |

Task 2b's nonblocking P3 distinguishes actual spending 2/3 from unchanged common
limits 3; the report/README correction and hash check are appended to that review.
The dispatch derivation correction is separate: request insertion is atomic with
admission, not another counted dispatch. The three possible calls are admission,
original reply delivery and ready outcome settlement/delivery. Numeric 6/9 bounds
remain valid; root settlement is uncharged.

Task 4's original specification APPROVE but quality/overall **REJECT** is preserved
verbatim. R1 found the repeated incorrect invocation/request/reply explanation in
its report. Only that paragraph changed. The same review appends independent
**R1 RESOLVED; specification/quality/overall APPROVE**, verifies original/corrected
report hashes and unchanged preflight sources/records, and claims no suite rerun
or recapture. Original rejection remains evidence even though this gate is closed.

## Preserved negatives and failures

| Evidence | Classification and disposition |
| --- | --- |
| [C_se scaffold RED](evidence/preflight/construction-red.txt) | Expected absent construction; Go FAIL retained, inner Go exit not separately captured. Wrapper exit 0 is not a passing Go claim. |
| [preflight metric failure](evidence/preflight/failed-metrics/failure.txt), [completion record](evidence/preflight/capture-completion.txt) | Capture-only projection/successor lookup corrected; final capture succeeds. Preserve original partial records and enclosing-wrapper limitation. |
| [C_se reference freeze](evidence/model/reference-r1.json.gz), [runtime order RED](evidence/runtime/se/red-order.json.gz) | Old candidate-first monitor rejects authentic S-first order. Construction/policy-extension RED, not substitution failure. |
| [C_se qualification bundle](evidence/model/se-qualification/records.json.gz) | Exporter/capture filename collision left the initial command record incomplete; repeated extraction produced identical bytes and a complete record. |
| [reason construction RED](evidence/reason-preflight/construction-red.json), [reference construction RED](evidence/model/reason-construction/red.json.gz) | Missing reason CLI and old answer-only guarantee/monitor extension gaps, corrected before qualification. Directed construction tests are not a product census. |
| [reason correspondence bundle](evidence/model/reason-qualification/records.json.gz) | Initial KeyError Members versus Descriptor.Members; exact failed checker retained. One lookup correction, final mapping exit 0, no changed census/artifact/guarantee. |
| [required-classification RED](evidence/runtime/reason/red-classification.json.gz) | Old stream lacks new classification evidence; expected adapter-construction RED. |
| [construction-green](evidence/runtime/reason/construction-green.json.gz), [construction-fixed](evidence/runtime/reason/construction-fixed.json.gz) | Both exit 1 despite labels: omitted authorization order metadata, then wrong-level binding operation-ID lookup. Narrowly corrected; final normal/race commands pass. |
| [audit attempt1](evidence/runtime/reason/audit-attempt1.json.gz), [final audit](evidence/runtime/reason/audit-final.json.gz) | Initial audit ran before baseline-race existed; exact failed script retained. Later audit passes with completed inputs. Intermediate pending-fixed is only a three-leaf check. |

Executable wrong-id and requested-absent→DeliberateFailure controls preserve actual
consumed evidence and correct policy while rejecting component/profile conformance.
Detached decoder corruptions are a separate category. Neither these controls nor
the construction/checker/capture/report failures constitute a conforming-replacement
substitution counterexample. The [qualification](02-qualification.md) and linked
runtime indices retain exact observations, selected histories and verdicts.
Historical I/J/K/L negatives and L's default-watchdog disposition remain in their
original archives and [review](../phase1l/03-review.md), with no current broad-suite
or expanded recovery claim.

## Controller completion gates

Final current-code static verification is recorded in
[final-static.json.gz](evidence/core/final-static.json.gz): seven commands exit 0,
378 Go/module hashes and 90 protected unchanged inputs. Source-bound behavioral
coverage and current affected regressions are distinguished in 02-qualification.md.

The [fresh final review](evidence/reviews/final-r1.md) returns Task 7 specification
APPROVE, Task 7 quality APPROVE and whole-branch source/evidence APPROVE, with no
actionable or parked findings. All seven declined-scope entries are accepted for
their recorded reasons: the assignment excludes the broader research domains and
repeated campaigns; fabricated checkpoints/tag-only evidence are already rejected;
historical freezes and failed records remain immutable. No finding was waived.

The subsequent [render record](evidence/manuscript/render.json) records exactly one
`tectonic alan-dissertation.tex` invocation, exit 0, accepted warnings and 19 PDF
pages. Pages13–16 were visually inspected, including Phase 1M on 14–15; no clipping,
overlap or missing glyphs were observed in the changed discussion or neighbors.
This is controller artifact validation after source review, not evidence the
reviewer claimed to have executed.

[Task records](evidence/task-records.json.gz) retain the unique reports, exact
corrections, ledger and review packages as described in [04-runbook.md](04-runbook.md).
Scratch paths are archive keys, not links to files that must survive cleanup.
