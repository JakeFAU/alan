# Phase 1L Task 2a independent specification and quality review

Specification compliance: **REJECT**. Code/research quality: **REJECT**.
The frozen package has two reproduced component-oracle false negatives at the
selected K1 helper boundary. Correct the checker and add focused regressions,
then freeze and rereview the changed bytes before qualification. The written
helper rules need no weakening. This is model-package review, not a reference
premise, positive qualification, or substitution verdict.

## Reviewed identities and scope

Verified aggregate
`5d624369114baa82c4e6a13a03ef9b470b41af51032233c9a1909147ce256de1`
using the declared sorted path + NUL + SHA256 + newline encoding. All five
contract/code digests match task-2a-freeze.json. Every added-file body in
`task-2a-review.diff` matches its current file. All eight archived file texts in
`docs/foundational/phase1l/evidence/model/reference-r1.json.gz` match their current
files; archive SHA256 is
`a1c7a10f8ebc7a59dba5abc80af2580bbc497ef45e301daf92204981fd17972d`.
All 21 inherited-source digests listed by the freeze also match.

Reviewed the brief, report, freeze, complete contract/four-file diff and the
relevant inherited primitives against the prior independent checklist. The
reference author had stopped. No ExploreJoin/product census, new parent/artifact
mapping, runtime execution, routine suite, production/source edit, commit or TeX
run occurred. Historical and author-reported passing runs were not rerun or
claimed as reviewer evidence.

## Findings

### R1 — P2: Candidate reply after helper refusal is accepted as K1-conforming

Location: `research/substitution/invocation_join_oracle.go:333–350`.

At reply_commit the checker rejects an unconsumed helper, but does not reject a
consumed unsuccessful helper. At terminal_freeze it correctly expects failed
for that helper result, so a candidate that sends the correct complement and
then fails receives no component violation. This contradicts the explicit
failure-without-reply rule in the new contract lines63–65 and original K1
lines44–49/57.

Reproduced with answer/id1/value0, I=R2/depth2: e and s are accepted, h commits
and receives invocation_limit refusal with zero debit, e consumes that original
refusal, then e commits/authorizes/admits a correct reply and fails. Its failed
published envelope truthfully retains the accepted response; both original e/s
outcomes are consumed and the parent honestly fails. `CheckJoinPrefix` accepts
this truthful protocol history and `CheckJoinComponent` incorrectly returns no
violations. The unchanged runtime publication rule permits a response on a failed
member envelope, so erasing presence or rejecting the prefix is not the remedy.

Required correction: enforce the helper-result precondition at the candidate
reply boundary. Preserve truthful prefix acceptance and separate G failure; add
an independent regression for reply-after-consumed-refusal. A conforming refused
helper/candidate-failure-without-reply control must remain accepted.

### R2 — P2: An admitted helper may fail without its required reply undetected

Location: `research/substitution/invocation_join_oracle.go:327–331` and the
candidate-only terminal checks at342–352.

All helper terminal events are skipped by the added discipline checks. The
inherited CheckTrace does not enforce the helper's fixed answer/Success behavior.
Thus an admitted helper can consume its request, fail without replying, publish
failed/member_failed, and be consumed by a correctly failing candidate while the
component monitor reports no violation. Original K1 lines44–49 fixes the helper
to a correct reply followed by Success; refusal is the separate no-run exception.

Reproduced in the positive input/allowance profile answer/id1/value0/I=R3/depth2:
e/s/h admissions all succeed, h fails without a reply, e consumes the failed
helper outcome and fails without replying, and the parent consumes e/s and fails.
`CheckJoinPrefix` accepts the honest history; `CheckJoinComponent` returns no
violations. This is component nonconformance, not a conforming replacement's
substitution counterexample. The local-relation caveat correctly limits proof
claims, but does not supply this omitted finite helper rule to the trace verdict.

Required correction: check the admitted helper's fixed reply/terminal behavior
without rejecting honest failure history at the protocol layer. Add a regression
for this failure and preserve valid helper success and admission-refusal cases.

## Independent probe evidence

The probe starts from the frozen hand-written protocol fixture, independently
inserts helper events and mutates the candidate behavior. It calls only the
prefix/component/G checkers, not a graph explorer or runtime. Retained source:
`task-2a-review-probe.go.txt`; exact captured output:
`task-2a-review-probe.log`.

Command actually executed against a temporary Go overlay:

```text
env -u OPENAI_API_KEY -u ALAN_MODEL_EVIDENCE GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build go test -overlay /tmp/alan-phase1l-task2a-review/overlay.json -run '^TestReview' -count=1 -v ./research/substitution
```

The environment was set equivalently by Python subprocess; Go exited1:

- TestReviewRefusedHelperNoReplyControl: PASS (truthful prefix and K1 checks pass;
  failed scarce execution does not establish G).
- TestReviewRefusedHelperReply: FAIL (expected component rejection was absent).
- TestReviewAcceptedHelperFailure: FAIL (expected component rejection was absent).

One earlier single-case overlay probe independently reproduced R1, also with a
Go FAIL result; the retained three-case run supersedes it as the compact witness.
To recreate the overlay, concatenate the frozen invocation_join_test.go and
`task-2a-review-probe.go.txt` into a temporary .go file, then map the original test
file's absolute path to that temporary file using Go's overlay Replace object.
No repository Go source needs modification for reproduction.

## Other reviewed obligations

Source inspection found the reference retains both K1 branches and the original
S request, replaces old caller decisions, and exposes actual scarce admission
refusal. e/s consumption is ordered and required before decision; member End,
reply authorization/admission, publication, delivery, consumption, parent End
and root publication remain separate. The stable binding exception and its raw
validation obligations are stated without extending it to invocation admission.
These are reviewed source properties, not enumerated reachability results.

The operation stage maximum27, private-PC maxima15+6, initial rank100 and declared
cap10621 are supported by the finite fields and forward-only inherited targets.
Obsolete root/peer fields are excluded from rank and fixed by state checks. The
Cartesian model/prefix bounds are conservative finite bounds, not census counts.
The explorer checks every generated edge before deduplication, records failures
without pruning, includes prefix state in its key, and leaves cycle status
unresolved when rank fails. For valid prefix histories, the keyed monitor retains
the future-relevant sets and helper outcome used by the current component check;
past prefix failures are retained before merges. This resolves the previous
representative-trace concern, subject to rerunning affected validation after
checker changes.

The source explicitly separates truthful history, component checking and G;
wrong-id/pending/incomplete fixtures exercise that separation. The two omitted
helper rules above are the actionable gaps. No additional blocking finding was
identified in this source inspection. Actual finite-edge rank checks, all maxima,
artifact correspondence, concrete limits and raw evidence remain later gates.
