# Phase 1L independent review

Review agreement is a source-bound quality gate, not a proof of substitution.

## Public construction preflight

[Independent review](evidence/reviews/preflight.md): **APPROVE** for specification
compliance and code/evidence quality, with no actionable findings. It covers the
one-parent construction, original A/B/S templates, exact three compiled binding
differences, public consumer, original selected outcomes and A2/B3 resource
accounting. Source manifest:
`e1f1137e4763bec9b2a49dd573be2084bbb094e11c925ab2e400c26baf83ab83`.

This review covers the two-sample feasibility milestone only; subsequent gates
have their separate dispositions below.

## Independent reference and oracle

[Initial reference review](evidence/reviews/model-r1.md): **REJECT** for both
verdicts. Two synthetic but truthful protocol traces escaped K1 component checks:
candidate reply after consumed helper refusal, and admitted helper Failure
without its required reply. Prefix truth and component conformance must remain
separate. The original reference freeze and failing overlay probes are preserved;
qualification was gated on correction and independent rereview.

[Scoped reference rereview](evidence/reviews/model-r2.md): **APPROVE** for both
verdicts; R1/R2 addressed. Revised aggregate
`794c6cc516b04a736a12d757212221e2bd9309f6c33c15282e2819561d7b676b`.
The original probes first reproduced RED and then passed GREEN; corrected normal
and race runs cover24 selected leaves. Source, exact commands and outputs are in
`evidence/model/reference-r2.json.gz`. This approves the reference/checker source
and finite argument for subsequent qualification; it is not a census result.

[Qualification and correspondence review](evidence/reviews/qualification.md):
**APPROVE** for specification and code/research quality, no actionable findings.
It verifies K1-first enumeration of all12 product configurations and16 maxima,
the precise local-refinement subset, normal/race evidence and89 mapped instruction
occurrences across the unchanged components and two bound parent programs.
Concrete raw history, schedules, controls and recovery have the separate review
below.

## Concrete evidence and corrected validator

A retained [extra-charge counterexample](evidence/runtime/revision1/charge-row-red.json.gz)
first exposed an adapter accepting an unlinked charge plus matching inflated root
totals. Admission/charge cardinality and root-total checks correct that defect.

[Task3 initial review](evidence/reviews/task-3-r1.md): **REJECT** for specification
and quality. R1 accepted coherently forged source revisions without binding them
to the original subject/history/admission; R2 omitted an orphan intent lacking
an emitting commitment. The [probes](evidence/reviews/task-3-r1-probes.json.gz) and
[rejected source/records](evidence/runtime/rejected-r1/index.json) remain intact.
These findings concern evidence validation; authentic captured runtime histories
were not substitution counterexamples.

[Scoped rereview](evidence/reviews/task-3-r2.md): **APPROVE** for specification and
quality; **R1/R2 ADDRESSED**, no new blocking finding. The corrected adapter binds
source revisions to original subjects/commitments and requires exact inverse
intent/history/execution coverage. Three fixed probes are RED against the archived
adapter and GREEN against the correction. All 19 corruption checks pass per
normal/race profile; 39 selected leaves, scoped vet/formatting and independent
record audit pass on source aggregate
`533faa388a04c63b18648bf1585e12ca54b416552d5bf9af01616d10fefc1e50`.
The rereview checks retained evidence and source; it does not claim new Go runs.

The frozen runtime README/index/report retain their capture-time pending-review
status. The final disposition is the rereview linked above and this document;
their historical wording does not reopen or replace that source-bound gate.
Likewise, 01-contract.md remains the pre-census freeze; its permanent corrected
reference archive is [reference-r2.json.gz](evidence/model/reference-r2.json.gz).

## Enclosing phase gates

The controller's adaptive follow-up completed the two previously unfinished tests
and 10 subcases under the documented 60m watchdog, with 105 sources unchanged.
The original broad 10m timeout remains failed. This is separate from Task3's
corrected evidence review; [04-runbook.md](04-runbook.md) records combined coverage.
[Fresh whole-change review](evidence/reviews/final.md): **APPROVE** for
specification and code/research quality, with zero actionable findings. The exact
review snapshot and pre-final bookkeeping inputs remain in
[review-package.json.gz](evidence/final/review-package.json.gz). The review
independently checks the reference premise, mapping, source identities, raw
records, recovery and manuscript scope; it does not substitute for proof.

The controller then ran one `tectonic alan-dissertation.tex` invocation: exit 0,
accepted warnings, unchanged TeX during the build, and an 18-page tracked PDF.
Pages 13–15, including the new Phase1L discussion on page 14, were visually
inspected without observed clipping or overlap. The [render record](evidence/final/tectonic.json.gz)
and [PDF inspection](evidence/final/pdf-inspection.json) retain the exact scope.
The [machine index](evidence/index.json) binds final records; the unique task
reports and controller rulings are retained in [task-records.json.gz](evidence/task-records.json.gz).
