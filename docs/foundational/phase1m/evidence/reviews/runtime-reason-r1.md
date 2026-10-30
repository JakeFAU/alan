# Phase 1M Task 6 independent runtime evidence review

Specification verdict: **APPROVE** for the bounded Task 6 runtime qualification.
Code-quality verdict: **APPROVE**. Actionable findings: **none**.

Question: do retained C_reason executions classify both originally selected
outcomes, including their actual reasons, and authenticate the separate terminal
step or designated pending boundary? The inspected evidence supports this tested,
bounded claim. No substitution counterexample or runtime/model correspondence
failure was found. Review agreement is not proof.

Reviewed on 2026-10-29 by m2_runtime_review against base
`1f71b56148f2fd6ef41261b7caf2996e8eeb2533` and the supplied uncommitted Task 6
slice. Read the task brief/report/diff, assignment sections 4–8, applicable working
agreements and Go review/testing guidance, approved Task 4 and Task 5b reviews,
the three new Go test files, shared decoder delta, relevant existing validator,
driver, consumer/reference code, capture/audit scripts and saved records. Task 4/5
consumer/reference changes are context, not silently included in this approval.

## Original records and classification

`consumer_reason_evidence_test.go:59–65` authenticates the raw inspection before
building reason events. The shared validator still checks original history and
selected observations, commitments, source authority, admission, binding/request,
publication/delivery, consumed sequence, resource charges and actual parent
instruction transitions. Its complete base source was independently compared
after removing only the new wrapper and optional order-metadata statements; the
remaining source is byte-identical. The metadata covers both appendEvent and the
separate authorization-event append. No old trace field or validation predicate
was removed or changed.

At `consumer_reason_evidence_test.go:155–199`, actual selected observation IDs
provide the full envelopes and receive bindings. Roles are resolved through the
previously authenticated emitted handles, not constructed from expected sequence
tuples. Classification requires both original consumptions in S/e order, checks
that their bound values remain unchanged, and refuses invented absent payload
registers. The tag is the committed literal Set's actual successor value; the
instruction must reach its adjacent End. The root's branch transitions are checked
by the existing pure-step verifier, so the final tag alone cannot certify a path.

The independent expected-policy calculation uses selected status, reason,
presence and guarded payload, not requested mode. The parent program likewise
interprets these observations only after both joins. The separately retained
component/profile verdicts can reject truthful defensive execution.

Classification and decision share one actual sequence/order
(`consumer_reason_evidence_test.go:189–208,336–343`), with classification ordered
first in the abstract expansion. End is a later commit. The public-driver test
at `consumer_reason_decoder_test.go:111–160` uses Store commits and ordinary
service; it does not compute successors or edit authoritative state. Both
profiles retain the same precise cut: root sequence18, order51, PC34→35, open at
End, candidate_missing_reply/failed, no root outcome. Completion executes End at
sequence19/order52. Independent comparison confirmed all 35 prior history rows,
original intents and observation descriptors/consumed sequences remain intact in
the final 36-history-row snapshot. No checkpoint between classification and
decision is claimed.

## Pending and controls

Pending recognition is restricted to original byte-identified A/B programs
(`consumer_reason_evidence_test.go:216–334`). It checks the original admission,
binding, first selected request, whole executed candidate path, actual branch
arrival, open current PC, and the binding-resolved receive match. It excludes an
unconsumed matching release and any candidate publication/consumption. B must
also have consumed its original successful helper outcome at awaitPC2 before
arrival. The declared no-second-request assumption remains visible.

Independent saved-row checks found A sequence4 at PC9→11 and B sequence8 at
PC10→12. ParentPC3 retains S consumed and e required, with no classification,
terminal result or root outcome. These are actual unmatched-receive arrivals,
not outcomes inferred from a requested pending mode.

The archived executable controls preserve the distinctions required by the
assignment (`consumer_reason_runtime_test.go:151–212`):

- Wrong-id retains its real succeeded/member_succeeded/present id2/value1 reply,
  then commits invalid_outcome/Failure after both consumptions. Raw/prefix pass;
  component and profile guarantee reject.
- An absent request to DeliberateFailure retains failed/member_failed/absent and
  commits candidate_failed. Requested-mode conformance/profile reject; observed
  policy is correct.
- Genuine A/B absence retains failed/missing_reply and candidate_missing_reply.
  Root failure remains member_failed, not fabricated propagation of missing_reply.
- Invalid S fail/absent still consumes both outcomes before invalid_outcome.
- Budget2 retains the genuine helper invocation_limit refusal, zero refused debit
  and total dynamic spending2.

The five detached decoder controls retain original and altered inspections and
rehash changed history. Their labels accurately distinguish them from executable
component controls. Captured rejections are classification path mismatch at PC34,
branch mismatch at PC30, selected/bound payload mismatch, pending path mismatch at
PC8, and final projection/history mismatch. The pending attack keeps the apparent
receive PC but falsifies predecessor data. Independent review checks rejected all
five altered copies in each profile; no live mutation result is inferred.

## Independent checks and captured qualification

This reviewer ran read-only `git status`, scoped `git diff`, `git show`, `rg`,
`sed`, `nl`, and Python heredoc inspections. No Go test, model census, public
runtime execution or saved audit script was rerun. The saved audit writes output
files, so its source and result were inspected rather than invoked. No concrete
unanswered doubt required a fresh behavioral probe.

The review's Python checks independently reconstructed original emitted intents
and retained handles, admission inputs/caller/template, request/binding/member,
publication payload, delivered observation, owning selected history and consumed
sequence, receive-match operands and publication-before-consumption order. They
walked actual parent instructions and pending candidate paths, derived the
classification table directly from consumed payloads, checked adjacent End and
separate terminal execution, and compared the resulting facts to the sidecar.
All 21 non-mutation records per profile passed; all five original mutation-source
records passed and all five altered copies failed. These checks did not import
the Go decoder/model/reducer or the saved Python audit functions.

The supplied audit (`runtime/reason/audit.py:11–19,27–153`) separately imports
only hash-pinned inert historical functions through AST extraction, then derives
root instructions, selected envelopes, policy, End and pending evidence from raw
rows. Its main function checks the same five detached copies and historical
equality. No historical top-level capture/audit execution is imported.

Independent parsing of saved Go JSON stdout confirmed the following separate
normal/race command results, with matching selected leaves, passing terminal test
events, exit0 and no failed/skipped final events:

| Selection | Normal leaves | Race leaves |
| --- | ---: | ---: |
| New reason runtime/modes/controls/decoder/cut | 25 | 25 |
| Affected C_es/C_se runtime/order/scarcity/decoder/prefix | 57 | 57 |
| Focused approved reason model/oracle | 40 | 40 |

The new selection has eight answer inputs, six A/B mode samples, five executable
controls, five detached negatives and one cut/completion test. G_answer is checked
separately on answer samples. Whole historical trace/handles/verdict equality
records are exactly equal to the original C_se wrapper-before record on all48
archived inspections in both profiles; this is not merely equality of pass flags.

Inspected exact commands use `/opt/homebrew/bin/go test -json -count=1` with the
anchored selectors in README, integration tags for runtime/baseline and `-race`
for race profiles. Scoped integration vet, four-file `gofmt -l` (empty output),
module verification, discovery, toolchain and diff-check commands all exit0.
Recorded environment is Go1.27.3 darwin/arm64 CGO1, local toolchain,
offline/readonly dependencies, the declared cache and fresh artifact directories,
with ALAN_* and OPENAI_API_KEY removed before the artifact variable is supplied.

All380 source hashes in each final behavioral/static command match current files,
and the aggregate recomputes to
`bfbba3d721fefdc63da9b76dce92b2af8226a6cbe1d837b1601ed8a903520590`.
All23 indexed file hashes/sizes match. Saved source snapshots also match their
recorded source hashes. Verified principal record digests:

- Normal: `ed9f9064187be6dda599118ed0d7156db46f44032e6b56b0b449a18fe65d878f`.
- Race: `3e5cafebe525ce228c20bd80b4d4fc024bba48f43768a5b2acc05771ea8a7e24`.
- Audit source: `421c6b409c883862ac2a989f457b9fa7f39ea816d8684922b9b499f056fa5370`.
- Audit output: `1a33ca654d02bbeae96e9c6b1992d10fe2afa1322fab419d5004e7237b9d1690`.

Four historical exit1 commands remain intact and are not relabeled final greens:
required_classification RED; construction-green's missing authorization-order
entry/index panic; construction-fixed's pending binding lookup error; and
audit-attempt1's missing not-yet-captured baseline-race record. The intermediate
pending-fixed command has three leaves. The report accurately distinguishes these
construction/capture failures from semantic counterexamples.

## Limits and remaining obligations

The fixed profile is I=R3, depth2, E0, fixed1, capacity1, with stable authority,
pure bounded inputs, no cancellation and eventual eligible service where progress
is claimed. The budget2 control is explicitly outside that positive allowance.
Runtime200/200 guards use the approved Task5b artifact justification: A28/B38
commits, dispatch6/9 and conservative128KiB within configured1MiB caps. Samples do
not establish those bounds or universal scheduling properties anew.

No production/runtime/compiler, consumer, reference, original template, dependency
or manuscript change belongs to Task6. The additive sidecar intentionally supports
this family; it is not a general malformed-inspection validator or arbitrary
program interpreter. Historical recovery retains its prior source and scope;
there is no new crash or recovery result. Universal substitution, arbitrary-context
congruence, compiler correctness and distributed/effectful recovery remain open.

Controller-owned Markdown/TeX reconciliation and final delivery remain outstanding.
This approval clears the runtime evidence gate, not those later obligations. Only
this review artifact was written; no subagents, code edits or commits were used.
