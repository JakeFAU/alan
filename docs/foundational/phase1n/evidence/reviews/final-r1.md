# Phase 1N final source and Task 5 review — R1

Reviewer: `/root/n_final_review`, independent of the implementation and
reconciliation authors. Date: 2026-10-30. This reviews the whole Phase 1N branch
and Task 5 source reconciliation before the intentionally deferred PDF build.
One Important finding requires correction; no additional semantic, mapping,
census or manuscript-content defect was found.

## Reviewed identities and method

Base: `3c5118ea9763a27cca432ca91aac667bd59e21e7`.
Head: `bf54943395350d57d27ded35efbbe048013e0028`, plus submitted uncommitted
Task 5 documents, core/marker records and manuscript additions.
The complete supplied `final-r1-diff.txt` is 1,028,961 bytes / 21,584 lines,
SHA256 `42bb22dffa8a2599c69d6626e3808d46fe8c934c80defcfbe573ec39746a55e5`.

Reviewed the package in passes: requirements and freezes; authored Go/model/
decoder/Python sources and tests; current prose/manuscript; prior reviews;
generated indexes and compressed evidence. Generated JSON/archives were parsed
and their relevant source maps, command events, counts and actual records
examined. All 58 textual new-file hunks were reconstructed and compared with
current files, respecting no-final-newline markers. This is not a claim of
semantic rereview of every historical source archived with the evidence.

Read `00-plan.md`, constraints, Task 5 brief/report and the code-reviewer
instructions; applied Go testing Review-mode guidance. Author reports and prior
reviews were treated as inspectable claims. No tests, race tests, model searches,
correspondence/semantic-audit reruns, render, commit, push or subagents occurred.
Read-only Python hash/archive/event/metric/normalization checks were the reviewer
executions. This report is the sole authored file.

| Task 5 input | SHA256 |
| --- | --- |
| `03-qualification.md` | `556a86dc00b936571785cb1379da0e3ba5d599f1662f7f317d7962e3d8f203a2` |
| `04-runbook.md` | `9213f3d095aa4b5348c31ce2381e150124985dfe1ab30371af90a6c726cdf00a` |
| `05-decisions.md` | `3d6db3900232e6db10e49abe62812e9d9cc80ce23360810695777c9fd1696aac` |
| `06-review.md` | `1ca45d395d3f68bd1e3a48d0a87996a795157a01f98e8bdfeba295ee2045f6e1` |
| `paper/alan-dissertation.tex` | `86364b3fae3550cd1802e982804b022b6226d273d83824334d103548fc3690df` |
| `paper/alan-dissertation.md` | `c7c7e0579b85acf38514e051ea76d2f683265da61ed4f832913550104011b458` |
| Task 5 brief | `a5a76f84a1f23709d3d6f101bdd6224c164f37b115f6bb899f638a9121952adb` |
| Task 5 author report | `49ebe8c321aca85e8b33e7149ddc613d6ce45e24503c9b4d5728f5680ee3d9f5` |

Numbered documents are relative to `docs/foundational/phase1n/`; brief/report
are the submitted `.superpowers/sdd/00-plan-phase1n/` inputs.

## Strengths and verified boundaries

- Public J/O code realizes original e-then-S commitment, S-then-e consumption,
  guarded actual-evidence classification, one bounded diagnostic before End,
  present reports on diagnosed failures, and O's original-handle public policy.
  The three absence cases remain distinct. The genuine controls test report
  truth separately from public authenticity and requested-mode conformance.
- ARG-1 closes under its supplied eventwise or guard-stable R, bilateral
  environment/caller readiness, authorization/resource assumptions and separate
  all-mode progress. The prior J debit and sole S/q debit have explicit physical
  depth/resource guard correspondence. Delayed e selection and late bound-reply
  authorization are justified under those premises. ARG-2 remains open in the
  final docs and manuscript; unchanged weak K1 is not silently strengthened.
  See `02-lifting.md:97`, `03-qualification.md:57` and
  `evidence/reviews/reference-r1.md:137`.
- Model/monitor keys, every-edge-before-deduplication checks, separate local
  progress and actual zero-successor pending maxima support the stated finite
  result. New static/raw mappings cover instructions and cursor states and use
  actual emitted handles and selected histories. Explicit atomic groups and
  immutable-tag ghost decisions do not invent durable checkpoints.
- Source-level independence is appropriately limited: the Python policy/audit
  does not call the Go reducer/classifier, but original artifacts, protocol
  formats and historical machinery are shared. Replay remains shared-runtime
  consistency evidence. Targeted unchanged-source reads confirmed the original
  selection/freeze links and reply-binding exception, including
  `internal/executor/invocation.go:123`, `messages.go:154`,
  `invocation_outcome.go:164,354,426,571` and the relevant
  `internal/substitutiontest/evidence_test.go` checks.
- The only existing files modified from baseline are the two manuscript files.
  Protected source identities, production behavior/dependencies, original K1/
  A/B/helper/S and historical evidence are preserved. Removing just the new
  six-paragraph block restores each baseline manuscript byte-for-byte. New
  Markdown/TeX content and math match after markup-only normalization; TeX's
  no-newline EOF and unchanged BibTeX are preserved. Rendered layout is unjudged.

## Evidence inspection, without rerunning qualification

All 117 entries in preflight/model-qualification/runtime indexes matched their
size/hash records. All 32 command-source archives matched their hashes, exact
member sets and 12,830 member hashes. The 66-entry reference archive also matches
its manifest. Final runtime/historical maps match all 405 current inputs; final
static checks' 400 Go/module inputs match current bytes.

Exact indexes: preflight
`af3087dcf6f399482d1d04e6effb8a503dbcb0b66633f963f3d7829fed0e43ca`;
model qualification
`0487869a81edac47283086492da876ad921567660819453290672f70ceca7f7b`;
runtime `a932031e049e78f908f248d1ad6af9454f5f9fe43f599ea3afaf59ba8011089e`.
Final runtime source aggregate:
`8b0c3e9b628dfbda57822df8e0005369635831ff99ed76ae9c3d075630d92a7d`.
Controller static capture:
`a645a99a179067c9a06a8944b2d19a07f98237aa0ee8653b200d73db9d2dba58`.

Reparsed saved command events confirm, per normal/race profile: four leaves per
accepted construction/consumer command, 66 frozen-reference leaves, 114
reference-first qualification leaves, 29 new runtime leaves and 82 inherited
leaves. Baseline captures separately contain 8/40 leaves and three original
reply-boundary leaves per profile. Eight controller static commands exit zero.
These are author/controller-run results, not reviewer-run tests.

Recomputed saved K1/A/B totals are states 6,376/2,864/6,644; edges
13,232/5,622/13,844; terminal maxima 24/12/12 and pending maxima 8/4/4.
All 64 maxima retain traces with empty contract/outer violations. Recorded
failures/refusals/stuck/cutoff/cycle counts are zero. Local pairs are 884/2,320;
progress states/edges 376/544 and 824/1,244; lifted pairs 2,864/6,644 with 16
initials each. Current correspondence has 283 mapped instructions and
73,984 J / 6,784 O cases per binding. Full normal/race finite exports agree.
K_join/G_outer predicates and repeated profiles are not additional censuses.

All 19 principal raw records per runtime profile agree with reported
commits/dispatches, static mappings and bounds. In the actual B/absent late
record `360b706714c9/result.json.gz`, Send is fact65, End/freeze67/68, first
authorization69, admission70, publication/delivery71 and original O selection72.
The post-End cut has no prior report authorization/admission and preparation
waits for `reply_admission`. The allowed authorization names source sequence31,
revision1/generation1, whose retained End is failed. Both profiles retain the
present candidate_missing_reply on failed/member_failed and O's matching decision.
Genuine control verdicts, seven detached attacks and two incomplete-prefix
claims remain separate; supported-driver RunReport is honestly null.

Earlier failures, superseded checker/configuration records and the interrupted
broad root run remain inspectable and excluded where stated. That interrupted
record has 1,098 passing and three skipped leaves plus the unfinished adaptive
test; exit1 is not a green full suite. Prior review identities and ten verbatim
rulings match the current review/decision documents. No behavioral rerun was
needed to establish the remaining issue.

## Finding — Important / P2 / F1

**Incomplete restoration instructions for the historical identity audit.**

Location: `docs/foundational/phase1n/04-runbook.md:49` through line59;
cross-reference `evidence/model/qualification/audit.py:57` and
`evidence/model/reference-r1.json:171`.

The runbook names restoration of the three ignored scratch inputs in a
disposable checkout. The historical audit hashes **every** file in the frozen
reference manifest, including both manuscript files. Restoring only the named
scratch inputs still fails at the delivered branch because reconciliation
intentionally changed the manuscripts.

A focused read-only comparison of all 66 manifest entries found exactly those
two current mismatches. Frozen Markdown hash is
`a94a65c6a3d1489f826306f32d8546d8461163830a140c5b0ca791dcab815e29`;
frozen TeX hash is
`2d19244524b404712c68d04fb8847809d5a1904847ef1e21c3e9bb57c12d4f1b`.
Current hashes are in the source table above. Exact frozen manuscript and
scratch bytes are present in `reference-r1.tar.gz` and match the manifest.

This is a reproducibility defect, not falsification of prior qualification or a
reason to undo valid manuscript reconciliation. Correct the runbook to restore
the full frozen manifest scope needed by that audit in a disposable checkout,
explicitly including pre-Phase1N Markdown/TeX and scratch inputs. Explain the
intentional current-manuscript difference; never overwrite the current working
manuscript or alter frozen manifests. Keep the separate task-records archive/
restoration requirement, including pinned `task-3b-report.md`. Verify proposed
restoration hashes and obtain focused independent rereview; no census, behavioral
test or PDF rerun is needed for this correction.

The controller acknowledged F1 and independently observed the same mismatches.
F1 remains unresolved in this exact reviewed package. No other Critical,
Important or Minor finding requiring change was identified.

## Declined to judge

- Arbitrary weak-K1 normalization/arbitrary-context congruence: ARG-2 and the
  broad theorem are explicitly open; this phase claims supplied-R ARG-1.
- Arbitrary candidates, recursive closure, an outer sibling/second debit or new
  scarcity/cancellation/capacity cross-products: excluded from this fixed
  research scope, with no broader preservation claim.
- Universal compiler/scheduler correctness, exact-debit A/B equivalence and
  effectful/distributed recovery: outside the named evidence and not asserted.
- Rendered PDF quality: the single Phase1N build/visual inspection is explicitly
  after corrected source approval; the edit marker is not a build result.
- Final task archive/index and PR bookkeeping: honestly pending actual outcomes.
  This review cannot certify future contents. F1 is not deferred because the
  present restoration instructions are already incomplete.

## Verdict and remaining gates

**Task 5 specification: REJECT.**
**Task 5 quality: REJECT.**
**Whole-phase source verdict: REJECT.**

The rejection is limited to F1's reproducibility requirement. The callable
boundary, restricted ARG-1, finite/runtime evidence and synchronized manuscript
claims otherwise satisfy the reviewed scope. Preserve this rejection and its
correction disposition, then complete focused rereview, the one Tectonic build
and changed-page inspection, exact task-record archival before cleanup, and
final evidence/review/PR bookkeeping. Pending fields are currently honest.
Authorization remains a reviewed draft PR only; no merge. ARG-2 and the other
open research claims remain open after those completion steps.
