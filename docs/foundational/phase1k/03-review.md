# Phase 1K independent review record

The assignment requires independent challenges to the profile map, public driver,
model/artifact correspondence and final claims. Review approval is a quality gate,
not a mathematical proof. Source hashes bind each verdict to its actual scope.

## Protocol gate

Initial plan SHA256: `9e89954ad0d327b9c2be7441b9a9f27c4924a7943351710192ef9f502da78346`.
Initial contract SHA256: `0b9013128dd84790da1d3fe9466fcb9fb4b4ecbfb1cf850576d50064076ca112`.
[Initial independent review](evidence/reviews/protocol-r1.md): **REJECT** for both
spec compliance and design quality. R1 requires a retained refused delivery before
kill so reopening tests the same delivery identity, not its first admission. N1
clarifies context versus operation PC bounds. The initial document bytes are
preserved in `evidence/reviews/protocol-r1-documents.json.gz`. Revision was required before experiments.
No new Phase 1K experiment precedes the gate;
unchanged J1 baseline tests are separately recorded under `evidence/baseline/`.

[Scoped rereview](evidence/reviews/protocol-r2.md): **APPROVE** for spec and
design quality. R1 and N1 addressed, no new issues. Approved plan SHA256
`eca229f7d3de8f462ab138cf895e851d69d0ce8f7da20cb9630945f3116e1adb`; contract SHA256
`075debb7c6596d70fb8438367f30399feefde326a6005e2be63603583178d544`. This approves design only;
new experiment results and final claims remain separate review obligations.

## Independent reference model/oracle

[Initial review](evidence/reviews/model-r1.md): **REJECT** for spec compliance
and research/code quality. The graph census and source freeze checked out, but
the oracle accepted two impossible peer sequences: ACK consumption before report
Send, and first report authorization after the ACK's consumption. The review
retains the exact malformed traces and expected-failing overlay diagnostic. These
are oracle counterexamples, not runtime-generated histories.

The initial reference archive and evidence remain immutable under `evidence/model/`.
Correction and a new source freeze are required before the context implementation.

[Scoped model rereview](evidence/reviews/model-r2.md): **APPROVE** for both
verdicts. R1 addressed; no new issue in the correction. Corrected reference freeze
`7ef775362562faa6b7ac1c4a364638545160be57b75daed81577b5417e0e7868`;
current oracle `7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4`.
Regression RED and corrected normal/race/vet/semantic-RED evidence are separately
retained under `evidence/model/revision2/`. Actual J2 artifact correspondence
remains a later gate.

## J2 construction and external terminal probe

[Independent context review](evidence/reviews/context-r1.md): **APPROVE** for spec
and code/research quality, with no actionable findings. The review verifies the
final source-bound gates, exact frozen peer compile failure and empty returned
Provision, available artifact identities, and authentic external terminal
observation admission and selected consumption. Approval covers this bounded
obstruction/probe result; it does not imply a compiled or executed complete J2.

The separate [independent correspondence audit](evidence/model/correspondence.md)
verifies available root/A/B/S fragments and preserves the full-peer gap. A typed
collection alternative would change the frozen prefix/control contract and was
not implemented. No universal authoring or runtime impossibility is concluded.

## Public driver, raw decoder and process cuts

[Initial runtime review](evidence/reviews/runtime-r1.md): **REJECT** for spec and
code/research quality. R1 is a decoder early-return bypass: a retained report with
wrong target or obligation could miss strict lookup and be accepted before the
truth checker. Both read-only mutations and their failing diagnostic are retained.
The review independently checked the original public pair, automatic boundary
and all six normal/race process records; those executions were not falsified.

The initial source aggregate `07ee2145f258069aaaa169eeded371f2e1ff1cf48b344340e4681b9b5072a2d6`,
archive/index and complete normal/race results remain historical evidence. The
correction received new source-bound full focused normal/race qualification:
27 passing leaves on identical corrected source aggregate
`e40b4b699b1682cce99718ca95d7d12292a6637025635843cf9ffba72e7b64c1`.
Initial results are not relabeled as corrected-source runs.

[Scoped runtime rereview](evidence/reviews/runtime-r2.md): **APPROVE** for
specification and code/research quality. R1 closed, no new scoped findings. The
reviewer's original lookup-bypass witness now passes; source-bound normal/race
qualification includes all 27 leaves. All 328 original artifacts remain unchanged.
The verdict covers corrected runtime source aggregate `e40b4b699b1682cce99718ca95d7d12292a6637025635843cf9ffba72e7b64c1`.

## Final whole-phase review

[Final whole-phase review](evidence/reviews/final-r1.md): **APPROVE** for
specification and research/code quality of the bounded result, with no critical
or important finding. The reviewer verified all 803 then-indexed evidence files,
444 source entries, all 131 protected baseline sources, source freezes, selected
gates, raw recovery records, correspondence limitations and normalized MD/TEX
content equality. This is independent source/evidence review, not a new proof.

N1 (P3) identifies a focused-test reproducibility defect: selecting an individual
decoder attack skips sibling fixture setup and panics before the intended check.
The original diagnostic is preserved in the review. The single final fix wave
now constructs genuine fixtures lazily for each selected decoder case. Exact
single-leaf RED/GREEN, 21 focused decoder leaves in normal/race and focused vet
are retained in [the fix report](evidence/reviews/final-fix-report.md). Latest
five-source aggregate `e174c4b7ca2e256259045ecdc117378111ecbb3d3de14b0ca5f7d88affb5eb04`
has focused decoder coverage; R2's full 27-case campaigns remain bound to their
original source. All 621 prior runtime/review artifacts are unchanged.

The review's declined-to-judge items match the explicit scope: full J2 runtime
correspondence/negative branches/actual charges, universal protocol or scheduler
claims, distributed/effectful recovery, general compiler correctness, unrelated
existing theorem audits and alternative handle constructions remain unresolved
or outside this phase. Final renderability and commit/PR metadata are finishing
obligations, not waived work. No review authorizes merging.

[Scoped final rereview](evidence/reviews/final-r2.md): **N1 ADDRESSED; APPROVE**
for specification and research/code quality, with no open actionable finding.
The reviewer independently reran the exact original single-leaf command (exit0,
4.485 seconds), verified all 621 prior artifacts and 129 final-fix indexed files,
935 then-indexed whole-phase evidence files, 445 source entries, and 131 protected
files. Latest focused qualification and R2-source RED/full-run labels were checked.
The manuscript retains the approved R1 hashes.

After approval, the controller ran the one phase-end Tectonic command: exit0,
18-page PDF, warnings retained under the authorized policy. New prose on PDF
pages6/13/14 was visually inspected using Poppler. The build and PDF inspection
records are in `evidence/gates/`. No mathematical or execution claim is inferred
from successful typesetting. Commit and draft-PR identities are supplied by Git
and the PR metadata; no merge is authorized.
