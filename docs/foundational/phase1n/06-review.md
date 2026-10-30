# Phase 1N review disposition

Status: Tasks 1, 2, 3a, 3b, 4 and whole-phase source review have explicit independent
APPROVE dispositions. Final R1's sole reproducibility finding F1 and REJECT remain
preserved; the corrected source received independent R2 APPROVE. The controller
then completed the single final PDF build, changed-page inspection, exact task
record archive and archive-specific restoration check. These execution records
are separate from the reviewers' source verdicts.

## Authorship and input boundaries

Contract author `n_contract`, construction author `n_construction`, reference/model
author `n_model`, runtime author `n_runtime`, and reconciliation author `n_docs`
have separate scopes. The controller owns review packages, integration, final
source checks, rendering and the draft PR. Workers did not commit, push or render.

The reference author derived the model/monitor from approved N-contract-1 and
named historical sources without new construction/preflight/runtime outputs.
Only after reference approval at
`c3c5b81938d177d73bca28532c3f8886ad02211b` did the
[firewall release](evidence/model/qualification/firewall-release.json) permit new
artifact inspection for correspondence. Shared historical K1/M/protocol machinery,
original component mapping, schema/runtime formats and compiler/replay limit
independence. The Python finite policy evaluator and runtime raw audit do not
call the Go classifier/reducer; they still share the declared record formats.

## Completed independent gates

| Gate / reviewer | Exact review SHA256 | Disposition and scope |
| --- | --- | --- |
| [Contract](evidence/reviews/task1-r1.md), n_contract_review | `d54e42694d6eb47864e639db18c672e0000dc955d24d9daad52cbb59b145e4cb` | Specification and research quality APPROVE; behavioral/environment freeze only |
| [Construction](evidence/reviews/construction-r1.md), n_construction_review | `613901061787b9115c99223adf452559b021160a4a1140701fd83139ee662e07` | Specification and quality APPROVE; public feasibility/preflight |
| [Reference](evidence/reviews/reference-r1.md), n_reference_review | `c6f5adf78ffa283a88a31ab417700a9b72f25ea07a6cb5ee54238ae716ad3246` | Specification and quality APPROVE; reference machinery and restricted ARG-1 |
| [Qualification](evidence/reviews/qualification-r1.md), n_qualification_review | `2e7e13d8662021e161e0bd66d538d0d02c06e975984a23163d0f9183bf542713` | Specification and quality APPROVE; finite/local/lifting, artifact mapping and bounds |
| [Runtime](evidence/reviews/runtime-r1.md), n_runtime_review | `249c987882044ca7475f2d7c64de706d1cb7d3ea39c23efdd8c7447990fcecdd` | Specification and code/evidence quality APPROVE; genuine execution, raw mapping, controls and audit |

Each review contains the exact input/source/archive identities and inspected
scope. Contract SHA256 is
`001c1fe5cd7be722aa75ae00a2be84ec702b51783a5595754d38ec8e6b2d8fd9`.
Reference source aggregate is
`36583a769994342aca07144e6ae0dcd4f4a62a30274a0863c75ee9b28a36d8c2`.
Task 3b qualification index SHA256 is
`0487869a81edac47283086492da876ad921567660819453290672f70ceca7f7b`.
Approved Task 3b commit is
`cb678321324437d9d5c7247c8a139fc5dc401201`; this does not include later Task 4
or reconciliation files. The [runbook](04-runbook.md) links exact captures.

No blocking specification/quality finding or substantive REJECT was returned in
these five reviews. Construction review clarified later sample/census scope;
the verdict did not change. Qualification review independently checked preserved
PC-coverage and operational-metadata corrections, the exact current r2 records,
all 67 indexed files, eight authored inputs and 66 unchanged reference inputs.
Reviewer hash/archive/source checks are distinct from author-run behavioral
tests: reference/qualification reviewers did not rerun the model census or tests.

ARG-1 is a checked source-level conditional argument, not a universal or
machine-checked theorem. ARG-2 remains explicitly unresolved: arbitrary weak
unchanged K1 refinement has no proved normalization/extension to the supplied
guard-stable/caller-ready/authorization-sidecar relation. The reviews do not
waive this gap or derive its closure from finite acceptance.

## Runtime approval and remaining completion boundary

Task 4's final [runtime index](evidence/runtime/index.json), SHA256
`a932031e049e78f908f248d1ad6af9454f5f9fe43f599ea3afaf59ba8011089e`, records 29
new and 82 inherited leaves per normal/race profile and a passing independent raw
audit. Its source aggregate is
`8b0c3e9b628dfbda57822df8e0005369635831ff99ed76ae9c3d075630d92a7d`.
Independent runtime review returned specification and code/evidence quality
APPROVE with no findings. The reviewer inspected the complete scoped diff and
verified every indexed hash, archived source and selected leaf, the actual late
authorization rows and all principal measurements without rerunning tests or
audits. The exact Task 4 commit is
`bf54943395350d57d27ded35efbbe048013e0028`. Its 400 Go/module entries match the
controller's static source map. That approval does not include manuscript or
whole-phase review. A reviewer lookup initially expected a list instead of the
saved historical record's dictionary; subsequent read-only inspection verified
the actual 48-entry record. No evidence was changed.

## Final R1 rejection and F1 correction

[Final source review R1](evidence/reviews/final-r1.md), SHA256
`0b66025e0069b67857450ebc4be43729a3a43fc3bf365cbdf721c7cc10a8f9ef`, returned
**Task 5 specification REJECT, Task 5 quality REJECT and whole-phase source REJECT**.
The sole Important/P2 finding F1 was incomplete historical-audit restoration
instructions: the audit hashes all 66 frozen manifest entries, including both
pre-Phase1N manuscripts, while the runbook named only three scratch inputs.
The review found no additional semantic, mapping, census or manuscript-content
defect. The original REJECT and finding remain unchanged and inspectable.

The corrected [runbook](04-runbook.md) restores the full frozen scope only in a
disposable copy, with exact archive/manifest/member identity checks and explicit
old-paper restoration. It separately preserves the pinned task-3b report's final
task-records archive requirement. [Focused evidence](evidence/core/final-review-fix.json.gz),
SHA256 `3eedc5038eca33ee073155049545d68863c19603f5d30b1b76a39276250a6010`,
reproduces exactly two manuscript mismatches and saved-audit exit 1 after
scratch-only restoration; all 66 hashes match and the same audit exits 0 after
full restoration. Original source files are unchanged. This reruns only the
saved-evidence audit in a temporary copy, not behavioral qualification or a render.
The [final archive restoration record](evidence/core/final-task-record-restoration.json.gz)
subsequently passed using the actual 39-file [task-record archive](evidence/task-records.json.gz),
including the qualification index's exact pinned report. Current sources remained
unchanged and the disposable copy was removed.

[Focused independent R2](evidence/reviews/final-r2.md), SHA256
`f61a9aef21f8ecbe65ceb14e9189375c2de0df9e81d40da642c02ed6dcb98655`,
returns **F1 ADDRESSED**, no new breakage, **Task 5 specification APPROVE,
Task 5 quality APPROVE and whole-phase source APPROVE**, under the original full
review plus this scoped correction. R2 does not retroactively change R1 or claim
a new behavioral campaign.

After that approval, the controller ran exactly one final Tectonic build from
`paper/`, exit 0, yielding 20 pages. Phase 1N pages 15–16 and neighboring pages
14–18 passed visual inspection; the established box warnings were accepted.
The [render record](evidence/manuscript/render.json) pins the actual source, PDF,
commands and findings. The successful [edit marker](evidence/manuscript/marker.json)
precedes manuscript edits and remains distinct from the build result.

The [delivery index](evidence/index.json) binds final source/diff checks, reviews,
archive and render. Post-review documentation changes record only these completed
controller gates. Draft PR delivery remains controller-owned; no automatic merge
is authorized. No claim
here closes arbitrary-context congruence, recursive closure, universal compiler/
scheduler correctness, exact-debit equivalence or effectful/distributed recovery.
