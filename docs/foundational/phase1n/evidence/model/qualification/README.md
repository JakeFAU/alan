# Phase 1N Task3b qualification

This is postfreeze qualification of N-contract-1 and reference-r1, followed by
artifact correspondence. It does not alter the approved reference, the public
construction, historical semantics, or the contract. Independent Task3b review
is still required. The results below are bounded checks and a separately reviewed
conditional argument, not a theorem inferred from enumeration.

## Argument status

**ARG-1 is closed in its restricted conditional scope**, as approved in
`../../reviews/reference-r1.md`. Supply a relation R at physical EntryDepth2/3
with residual I/R3 and prior J debit (1,1,0): original identities, protocol stages,
outcomes/presence, source terminal state, ownership, closure, ledger/depth and
future demand agree under one fixed renaming. Candidate transitions match one
original event or a public-field-preserving private stutter. A weak-path variant
additionally requires no intervening nonmatching protocol event and guard-stable
intermediate stutters. Shared environment/caller operations have bilateral
readiness, including sole S admission as q, caller-owned commitment and delayed
original selection, and equal authorization sidecars. Separate all-mode local
progress/nondivergence, eventual eligible service, stable original bindings and
authority, open caller, sufficient finite limits, no closure and no pending
release are premises. Under these premises the fixed J preserves R and K_join;
adding the fixed O control yields the outer-policy corollary.

**ARG-2 remains open:** arbitrary weak refinement of unchanged K1 does not yet
have a proved implication to that stronger supplied-R premise. The missing
normalization/extension lemma must preserve intermediate guards, caller-owned
commitment/selection readiness, and split authorization sidecars. No result here
proves or disproves that implication. K1 was not strengthened. Matching O private
PCs/registers is required only by the outer-policy corollary, not K_join or the
constructor relation.

The finite A/B witnesses below supply instances of the stronger eventwise
relation and separate progress. They instantiate ARG-1; they do not establish
ARG-2 for arbitrary candidates. Raw runtime correlation/binding authentication
and retained-history correspondence remain Task4 obligations. No manuscript file
was changed in this delegated slice; final Phase1N reconciliation must retain
this exact distinction.

## Authorship and reproducibility

`firewall-release.json` records release after reference approval at commit
`c3c5b81938d177d73bca28532c3f8886ad02211b`. Only then were approved
`examples/callablejoin`, representation, preflight and construction review made
available for correspondence. The Task4 static inventory was used as a map,
not an oracle. The frozen reference was written without those inputs. Shared
historical K1/M/protocol machinery and Phase1I component correspondence remain
explicit independence limitations. The new Python policy evaluator does not
call the author combinators, executor reducer, or frozen Go classifiers.

Each command directory retains `command.json.gz` (actual argv, pinned overrides,
stripped-variable policy, stdout/stderr, exit, all test events/leaves, output
hashes, source hashes and during-command change check) and `sources.tar.gz`
(exact source bytes). The archive contains all Go/module sources, the new Python
scripts present at command start, and approved N contract/representation/lifting/
review. Full result files retain every maximum, trace and failure field.
`audit-r2/audit.json.gz` derives a compact 64-maximum index, census totals, command
identities, normal/race equality and freeze-integrity checks. `index.json` binds
all final evidence bytes without replacing earlier captures.

All Go commands used `/opt/homebrew/bin/go` (verified Go1.27.3 darwin/arm64),
GOTOOLCHAIN=local, CGO_ENABLED=1, GOPROXY=off, GOSUMDB=off,
GOFLAGS=-mod=readonly and GOCACHE=/private/tmp/alan-phase1n-model-go-build.
Unrelated ALAN_* variables and OPENAI_API_KEY were removed; only the capture's
ALAN_MODEL_EVIDENCE was added. No network/model call, broad root test suite,
commit, push or render was performed.

The exact anchored Go entrypoints are:

- `^TestCallableJoinReferenceQualification$`
- `^TestCallableJoinAlignedLocalQualification$`
- `^TestCallableJoinReplacementQualification$`
- `^TestCallableJoinFiniteLiftingQualification$`
- `^TestCallableJoinArtifactCorrespondenceInputs$`

The first four use `go test [-race] -json -count=1 -timeout=5m -run SELECTOR
./research/substitution`, in that order within each profile. The fifth follows
model qualification. Initial and final `-list` discovery are retained. Actual
passing leaves per profile are 16, 32, 32, 32 and 2 respectively (114 total;
228 across both profiles). There were no Go failures/skips or source changes
during a command. Focused `go vet ./research/substitution` exited 0. Python
checker/audit runs are separately recorded and are not Go test leaves.

The first model captures precede gofmt and the additive artifact extraction test.
Their exact sources remain archived. The audit runs gofmt on each archived
qualifier and verifies exact equality with the final qualifier bytes; it does
not pretend the later source snapshot executed those earlier tests. All 66
reference-r1 source/input hashes are still identical. Normal/race full JSON
exports are byte-identical for every model/local/lifting/artifact result.

## Reference-first product census

The domain is id {1,2} × value {0,1} × mode {answer,fail,absent,pending};
16 configurations per K1/A/B product, 48 total. K1 retains its independent
local/helper choice in every mode, hence two maxima per configuration.

| Product | States / model states | Edges | Revisits | Terminal / pending maxima | Rank min..max / cap |
| --- | ---: | ---: | ---: | ---: | --- |
| J[K1] with O | 6,376 | 13,232 | 6,872 | 24 / 8 | 100..9,300 / 12,221 |
| J[A] with O | 2,864 | 5,622 | 2,774 | 12 / 4 | 100..7,206 / 12,221 |
| J[B] with O | 6,644 | 13,844 | 7,216 | 12 / 4 | 100..9,313 / 12,221 |

Every state/prefix/component/projection/K_join/G_outer/rank failure count is zero.
Every refusal, stuck, cutoff and cycle count is zero. The full model-plus-monitor
key and every-edge-before-deduplication checks are frozen machinery. Equal model
and product state counts are observed here, not an excuse to remove monitor
memory. K_join and G_outer project the same full products; their counts are not
independent censuses. Race repeats are repeat checks, not more configurations.

Full charges at every A maximum are (3,3,0), B (4,4,0), K1 independently both.
The prior J charge is not q; accepted original S is the sole q debit. Candidate
and S depths are 2 and helper depth 3; no refunds or refusals are erased. Every
pending maximum has consumed the original candidate request and any chosen
helper, is at its designated unmatched receive, has actually selected original
S, leaves J at original e await with no diagnostic/reply/End/publication, and
leaves O at original J await with no decision/End/root publication. No release
input or Pending diagnostic is introduced. Maximality, all-mode progress,
well-founded rank and prefix acceptance are distinct checks.

## Aligned local and finite lifting checks

| Check | A | B |
| --- | ---: | ---: |
| Aligned local relation pairs | 884 | 2,320 |
| Component / environment checks | 468 / 1,688 | 1,328 / 4,168 |
| Private stutters | 180 | 496 |
| Separate open-progress states / transitions | 376 / 544 | 824 / 1,244 |
| Open-progress terminal / pending maxima | 12 / 4 | 12 / 4 |
| Open-progress rank min..max / cap | 1,400..4,906 / 8,321 | 1,400..6,813 / 8,321 |
| Finite lifted relation pairs | 2,864 | 6,644 |
| Lifted component / environment checks | 2,954 / 5,336 | 7,280 / 13,128 |
| Lifted private stutters | 644 | 1,512 |

Each relation has 16 initial pairings and zero failures. Local relation includes
the declared environmental closure cases; the separate progress runs are open.
Every local progress refusal/stuck/cycle/rank/cutoff count and maximum error is
zero. The historical progress summary's `Entitled=4` counts its answer-only
predicate; the new explicit all-mode maximum predicate qualifies all 16 records
and does not reinterpret that legacy field. Local and lifted full records retain
all actual accounting and counterexample fields.

## Artifact correspondence and finite guards

`normal-correspondence-r2` and `race-correspondence-r2` are the current checker
records. There are 283 mapped instructions: J58 and O64 for both bindings plus
original A13, B16, helper7 and S3. Every instruction, operand, guard target, setter,
Send, End, collection declaration and cursor transition is represented. Original
component bytes match historical frozen artifacts. Six exact A/B difference paths
are retained: candidate metadata/Invoke/Await in J and corresponding J metadata/
Invoke/Await in O. All 16 input variations per binding preserve program identity
and topology; only initial request data change.

J's classification Set expands to `join_classify` and a ghost `join_decide` fixed
by the immutable classification. Checked remaining code contains only jumps,
one original Send and the exact PC55 tag-to-End branch. This does not invent a
durable intermediate decision checkpoint. O's decision Set similarly fixes PC61.
Receive selection/binding/consumption and Invoke/handle/intent are atomic groups;
End/freeze is separate from later publication. The abstract split of publication
and delivery need not denote two durable runtime commits. J Send/End may precede
first authorization; the original bound-reply exception still requires exact
source, generation/revision, caller, route/schema, binding and open destination.
Static correspondence does not discharge Task4's raw authentication obligation.

The Go extractor uses the actual original Receive to establish schema eligibility
and absent-field omission, not to decide policy. Each binding supplies 140 inner
envelopes (136 accepted, four succeeded-without-response rejected) and 1,703 outer
envelopes (1,696 accepted, those four plus unknown tag/two rows/missing answers
rejected). The finite domain is five statuses × four reasons × absent or present
records: child id {0,1,2}, value {0,1}; report id {0,1,2}, four declared tags,
empty or one of those six answer rows. Requests use id {1,2}, value {0,1}.

The independent evaluator checks 73,984 J cases and 6,784 O cases per binding.
J classifications are answer 4, candidate_failed 4, candidate_missing_reply 4,
invalid_outcome 73,972. O decisions are answer 4, candidate_failed 4,
candidate_missing_reply 4, invalid_outcome 4, invalid_join_outcome 6,768.
All exact expected statuses and reports agree. Complete representative inputs
and traces are retained for every distinct J path (22) and O path (54), including
cursor snapshots/index/exhaustion. Absent fields stay absent and an accidental
operand read raises an error. These are schema-valid adverse combinations, not
claims that every combination is produced by this closed deployment.

J setters PC27/49 are schema-unreachable because successful absent responses are
rejected; their shared successor jumps PC28/50 remain reachable. O PC14 needs a
second row excluded by MaxItems1; PC50 needs a tag outside the four-member enum.
Every other post-selection PC is witnessed. Static correspondence still covers
all unreachable instructions. Invalid reports are recorded as schema rejection,
not credited as guarded policy behavior.

The one-element O cursor executes `next` at most twice. Summing every static PC
(including mutually exclusive and unexecuted pending-release paths), plus that
one repeated `next`, gives conservative local commit bounds A139/B149. At most
three dispatches per admitted singleton gives A9/B12. Source-checked public
consumer settings are **512 steps / 64 dispatches / capacity1**. These are not
model ranks. Largest retained J/O program is 25,387 bytes; the conservative
bound-program allowance is at most 416,142 bytes. Projection/observation/single
intent-array allowance is 55,616 bytes; checked finite schema payloads reach 571
bytes; original handle allowance is 76 bytes. All fit default 1,048,576-byte caps.
Bounds assume the declared finite stable, no-error/no-retry/no-closure profile,
fixed original identity sizes, one list element, and one intent per Emit. They
do not cover arbitrary external strings, adapters, retries or recovery.

## Preserved corrections

`normal-correspondence-1` exited 1 on a new checker coverage assertion. It wrongly
listed shared jumps PC28/50 as unreachable along with setters27/49. Exact edges
PC26→28 and PC48→50 refute that assumption. Only the coverage expectation was
corrected; `normal-correspondence-2` passes. No frozen model, policy table or
public artifact changed.

The first successful correspondence captures also mislabeled historical M's
200/200 operational settings as the N consumer configuration. The controller
identified this metadata defect by source inspection. Current r2 captures retain
the derived 139/149 and 9/12 bounds, assert the actual 512/64 source settings and
check they fit. Earlier successful captures and `audit-final` are preserved but
superseded on that configuration metadata; `audit-r2` is current. A read-only
inventory command also raised `KeyError: 'Collections'` when accessing absent O
optional metadata; it changed no file or verdict and was replaced by `.get`.
There is no discovered frozen-reference or public-construction counterexample.
