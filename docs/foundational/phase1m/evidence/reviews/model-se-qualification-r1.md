# Phase 1M Task 2b qualification/correspondence review

Spec verdict: **APPROVE** for this qualification/correspondence slice.
Code-quality verdict: **APPROVE**, with one non-blocking documentation finding
below. No model, qualification, mapping or operational-bound defect found.

This is an independent evidence review of the unchanged approved reference and
the Task 2b delta. It permits proceeding to Task 3. It does not approve the still
gated runtime causal evidence, C_reason, all-mode conformance, or final manuscript
claims. Review agreement is not proof.

## Finding

**P3 — distinguish spent units from common limits.**
.superpowers/sdd/00-plan-phase1m/task-2b-report.md:15-16 and
evidence/model/se-qualification/README.md:19-20 describe K1's “I=R2 and3”
terminal branches immediately after using I/R for configured limits. The archived
Config always has I=R=3; the two branches have observed dynamic spending
(I_spent,R_spent)=(2,2) or (3,3). A spends (2,2), B spends (3,3), all under the
same allowance. State that distinction explicitly in both sentences and retain it
in the final qualification account. This requires wording only, no new execution
or archive replacement. It is separate from the already retained dispatch
correction.

## Scope, identity and evidence checks

Reviewed task-2b-report.md, task-2b-diff.txt, the existing task brief, the evidence
README/index/archive, the new qualification test and capture/extractor/checker/
bundler, the narrowly reused Phase 1L correspondence checker, actual C_se consumer
sources/artifacts, and protocol source relevant to finite operational bounds.
No subagents, commits, whole-branch re-review, runtime causal records, or
behavioral test reruns were used.

The approved reference archive remains SHA-256
3663adea727e91f714e3257dae60331d43c194b074b7bb7bfe1c7ccc5dde329f with aggregate
cdfe2a191641129dc591b92aabd680d238a80a1e027cfb77a82edc84fd465cd1.
All 36 frozen working inputs still match.

Task 2b [records.json.gz](../model/se-qualification/records.json.gz) has SHA-256
d8600027dd6bed0d0b768d2c24cc40b37cf85fb2e1d68241f3669ec7d9ae07a8;
its source aggregate is
83fb6fd153b21e56699561561dc8114b1fcb74e8ddd9211f37aef2d2263a335e.
Read-only Python inspection independently recomputed the archive and source
hashes, compared all archived new source snapshots to working files, and checked
the relevant model/test/consumer hashes in every command record. The temporary
extractor's captured hash matches extract.go.txt. All correspondence source
hashes match. The index agrees with these results.

The 15 complete command records have exit0. Parsing their saved Go JSON streams
again gives 85 passed leaves per profile: 4 reference, 8 replacement, 23
historical/local/instruction, 50 construction/oracle. There are no failed or
skipped events. These are combined results from four commands per profile, not
one 85-leaf command. Recorded toolchain is Go1.27.3 darwin/arm64, CGO1,
local/offline/readonly. The initial exporter/capture filename collision remains
explicitly disclosed as an incomplete command record; it is not presented as
another captured green command.

## Separate bounded results

The archive preserves exactly id{1,2} × value{0,1}, answer, common I=R3,
depth2, E0 and one fixed parent for each product.

| Product | Configs | Model/product states | Edges | Terminal maxima | Dynamic spending per maximum |
| --- | ---: | ---: | ---: | ---: | --- |
| C_se[K1] | 4 | 1996 | 4376 | 8 | (2,2,0) or (3,3,0) |
| C_se[A] | 4 | 944 | 2046 | 4 | (2,2,0) |
| C_se[B] | 4 | 1784 | 3842 | 4 | (3,3,0) |

All recorded state/rank/prefix/component/G failures, rejections, pending/stuck
maxima, cycles, cutoffs and counterexamples are zero. Independently inspecting
every archived C_se maximum confirms the two original correct present outcomes,
actual s-then-e consumption in its retained trace, parent/root success, fixed1,
and the stated spending. Normal/race census structures match exactly. The rank
maximum is 7800/5706/7813, below10621; initial rank is100.

Saved timestamps put K1 completion before the first A/B command starts in each
profile (approximately12.0s normal and60.3s race gaps). The new test deliberately
has distinct reference/replacement entry points
(consumer_policy_qualification_test.go:60-68). Its checks do not prescribe graph
counts: :25-51 reject failures/incompletion and require the specified branch
spending set; :54-56 emit all configurations, including failures.

The current answer-only local relation separately covers A/B300/632 pairs,
156/344 component checks,600/1172 environment checks and52/104 stutters, with
zero failures. The separate Closure=false runs have120/220 states,
184/338 edges and4/4 terminal outcomes. Every retained open outcome is the correct
present answer, with no pending, stuck, rank/cycle/cutoff or counterexample.
invocation_join_qualification_test.go:77-84 explicitly separates relation from
progress; explorer.go:156-157 checks entitled open completion. This accepts the
stated answer subset; it does not certify fail/absent/pending progress for a
future C_reason context.

Current C_es regression records also match the stated K1/A/B
1952/926/1706 states,4276/2006/3678 edges and8/4/4 maxima. They supplement the
approved shared-wrapper source preservation argument.

## Correspondence and bounds

The adapted checker changes exactly five asserted-once pieces of the historical
checker (check_correspondence.py:9-20). The changed receive expectation is
independent of the compiled bytes being checked. It also shifts the candidate
source-difference location from Code2 to Code3 and swaps the per-PC receive map.

The 89 covered instructions belong to six distinct artifacts:
A13, B16, helper7, S3, parent[A]25 and parent[B]25. Program hashes are checked
against their actual base64 bytes. Original candidate/S template bytes and the
inherited A/B/helper instruction records are tied to their historical identities.
The exporter checks all four input constructions per binding, including invariant
artifact bytes and all non-input provision fields (extract.go.txt:50-74).

For each parent, PC0/1 commits the original e/s handles, PC2/3 selects original
s/e outcomes with operation/template/provenance/generation/obligation predicates,
PC4-7 computes the complement, PC8-15 executes the guarded policy, and PC16-24
contains the actual Success/Failure End. Status and presence dominate their
respective payload guards. The pure setters/branches/jump are finite stutters;
decision is assigned at the resolved-End boundary, not inferred from a retained
tag. Both original receives precede every guard and End. This is finite artifact
correspondence, not a compiler theorem or evidence that a runtime branch ran.

Independent recomputation from the archived forward control-flow edges gives
longest paths parent16/A6/B10/helper6/S3. Summing admitted programs yields
A25/B35 local commits, within200. The 64KiB byte bound is conservative for this
fixed scalar, pure, stable profile: at most16 registers, five outcome payload
fields, bounded original handles and fixed protocol strings, one intent per
reduction, and finite binding expansion. The checked program/projection/envelope
bounds remain below the four1MiB caps. Source inspection includes the binding
operand/port changes at composition_binding.go:287-337 and the named envelope
structs; these bounds do not cover arbitrary payloads, adapter errors or repeated
external interruptions.

The dispatch correction is substantively correct. runner.go:152-158 counts a
delivery call, :268-272 first admits then attempts settlement, and :385-397
excludes an admitted unready invocation until it is ready. invocation.go:480-490
creates the request observation inside admission. Thus each successful singleton
needs at most initial admission, original reply-message delivery, and ready
outcome settlement/delivery: A6/B9. Root settlement at runner.go:241 is separate
from Dispatches. This source argument does not depend on the earlier mislabeled
request-dispatch explanation or on Task 3 samples.

## Remaining boundary

Task 3 must still authenticate actual original retained records and selected
consumption, show both original publication orders and the S-consumed/e-required
open state, and retain the dual pending controls. Shared model protocol and
finite artifact checking do not establish those concrete observations.
Eventual eligible service and the declared stable pure profile remain assumptions.
All-mode local progress, C_reason, general congruence, compiler correctness and
effectful/distributed recovery remain outside this approval.

Only this review artifact was added by the reviewer. No code, tests, archived
qualification records, reference freeze or manuscript was changed.

## P3 correction disposition

**RESOLVED.** Scoped follow-up verified the exact two-file delta in
.superpowers/sdd/00-plan-phase1m/task-2b-wording-fix.md and the current passages.
Both now explicitly distinguish dynamic spending (2,2)/(3,3) from the unchanged
common limits I=R=3. The report and README hashes respectively match
85572617033808e140d46b09456582e0f6506a82bb662d2e5a1f3ae7f1ab85b2 and
2ddd0eb48cd622243705946b54e58e7500a969d98ae80e1b037e728314968837.
The qualification archive still hashes to
d8600027dd6bed0d0b768d2c24cc40b37cf85fb2e1d68241f3669ec7d9ae07a8.
No tests, census, broader review or code changes were performed in this follow-up.
The original finding remains preserved above; both APPROVE verdicts stand.
