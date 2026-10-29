# Phase 1L Task 2b independent qualification review

Specification compliance: **APPROVE**. Code/research quality: **APPROVE**.
No actionable blocking finding in the completed Task2 scope.

The inspected evidence establishes the stated bounded finite-model premise and
replacement results for the four frozen positive inputs, plus the reported
finite artifact correspondence. This approval does not establish concrete raw
history validity, actual runtime schedule coverage, recovery, general compiler
correctness or arbitrary-context substitution. Task2a approval remains a
source-package approval; this review adds the separately checked Task2b evidence.

## Scope and identities

Read task-2b-brief.md, task-2-report.md, the complete task-2b-review.diff, the new
qualification test, extractors/capture/bundling/correspondence scripts, command
records, compressed qualification data and relevant original/runtime source.
No Go tests, census/ExploreJoin, builder extraction, runtime execution or routine
suite was rerun. Reviewer work used source inspection and independent Python
checks of retained hashes, arithmetic, records, instruction bytes and DAG bounds.
No model/production source, TeX, commit or nested agent was changed/created.

Verified the approved five-file aggregate is still
`794c6cc516b04a736a12d757212221e2bd9309f6c33c15282e2819561d7b676b`.
All 128 source-manifest files match current bytes; manifest SHA256 is
`10317fccdf0ee4bb90c632d2099df800f93d14cfd5b9584ba5091ab983363ab1`.
All indexed reproducer and nine command-record hashes match. Every command's
25 model-source hashes matches current source; no test result is SKIP or FAIL.
The executed temporary extractor/checker sources also match the archived copies.

Verified qualification.json.gz SHA256
`faf47c274b4dfe5eda51fea80d4c91df31ca22fd41435b7a12460ecfaa94cca5`,
27,463 compressed / 416,318 uncompressed bytes. Artifact extraction text matches
its captured temporary output and indexed SHA256
`cb2a7119d9c0933f063265743df6272418935ae82e9c9c470fdb5ccdadb269e9`.
All four normal/race evidence files are byte-identical, match their indexed
hashes, and decode to the bundled records. This verifies retained source-bound
run records; it does not assert newly executed reviewer tests.

## Reference premise and replacement evidence

The normal K1 command completed before the replacement command began (completion
UTC and recorded duration independently compared). It selected all four
id1/2 × value0/1 answer configurations; A and B subsequently selected the same
four inputs with identical I=R3, depth2, E0 and fixed1. The reference retains both
local/helper branches. Reference source was frozen before artifact extraction.

Independently recomputed the aggregate table from every bundled configuration:

| Product | Configurations | States / model states | Edges | Revisits | Terminal maxima |
| --- | ---: | ---: | ---: | ---: | ---: |
| K1 | 4 | 1,952 / 1,952 | 4,276 | 2,328 | 8 |
| A | 4 | 926 / 926 | 2,006 | 1,084 | 4 |
| B | 4 | 1,706 / 1,706 | 3,678 | 1,976 | 4 |

For every record, edges+1=states+revisits. All reported refusal, rank, state,
prefix, component, G, pending, stuck, cycle and cutoff counts are zero and all
counterexample arrays empty. Every one of the16 maxima independently checked
from bundled observations has both originals consumed, correct present e/S
results, parent/root Success, E0 and fixed1. K1 maxima spend I=R2 or3; A spends2;
B spends3. Rank ranges are100..7,800 /100..5,706 /100..7,813 under cap10,621.
These are maximal states with representative traces, not a count of all traces.

The new qualification wrapper retains every completed exploration, uses t.Error
rather than fail-fast pruning, visits every frozen configuration, and checks
all maximal results/accounting. The unchanged reviewed explorer checks every
edge before deduplication and keys independent prefix state. No sample-only
subset, cutoff success, discarded failure or revised model was found.

## Local subset and current-source verification

The exact eight local cases are four per A/B, Local=true, Shape=local,
EntryDepth1, Depth2, I=R3, Peer=true and Closure=true; separate open progress uses
Closure=false. Checked sums: A300 pairs /156 component /600 environment checks /
52 private stutters and120 open states /184 edges; B632 /344 /1,172 /104 and
220 /338. Failures/counterexamples are zero. This is current-source evidence for
this subset, not a relabeled historical full campaign.

The single accepted S debit correctly projects to arbitrary-timing q=(1,1,0),
with other S activity stuttering for the local candidate relation. The report
properly excludes S reply/lifetime, parent joins and root publication from that
local claim. It also preserves the distinction between disabled q under scarcity
and an actual rejected S invocation. No I=R2 qualification is inferred here.

Inspected captured normal selections4+8+11 leaves, race23 leaves, and vet exit0.
The same source hashes bind normal/race output. Prior Task2a unit/probe results
are explicitly prior evidence; the report does not claim those were rerun.

## Artifact mapping and operational limits

Verified actual program bytes hash to their program IDs, original A/B/helper
instruction arrays match all36 inherited mapping records, and A/B/S template
objects match the frozen Phase1K material. The independent checker reconstructs
all25 instructions of each parent and all3 S instructions, including operands
and unused fields. Together this covers89 distinct artifact-instruction
occurrences:36 inherited +3 shared S +50 across the two parent artifacts.

The all-input extractor builds each of four inputs for each binding, verifies
unchanged parent/candidate/S bytes and non-input provision, and records the
original three request fields. Source inspection confirms the typed fields are
created with schema.Int/schema.Text. Parent artifacts differ at exactly the
three candidate-binding leaves; the catalogue difference is A versus B/helper,
with S shared. Limits, fixed parent/root, guard logic and result policy agree.
No candidate-specific behavior/allowance is introduced.

PC0/1 commit e/s before PC2/3 consume their original handles. PC4–7 compute the
Boolean complement; PC8–15 check candidate then S status/presence before payload
fields. Each false branch reaches Failure and the all-true branch reaches
Success. S receives the original id and replies(id,0). The correspondence treats
parent_decide as a ghost event at the actually reached End after the pure suffix,
not a fabricated retained decision register. Task3 must authenticate that path;
its normalizer must not evaluate G on behalf of the parent.

Independent backward DAG calculation confirms parent16, A6, B10, helper6 and S3
maximum local commits, giving totals25/35. All control targets advance within
bounds. Runtime runner source confirms at most two successful invocation dispatch
calls plus one bound-reply call per admitted invocation: initial admission,
ready publication/delivery, and reply. Pending invocations are not requeued until
ready; outcome delivery and message acknowledgments suppress repeats. This
supports6/9 dispatch bounds for the stated stable successful automatic runner,
not unbounded retransmission/restart. MaxSteps200/MaxDispatch200 exceed these
bounds. The program-byte observations are correctly limited to the finite
artifacts and are not presented as a universal database/payload bound.

## Runtime boundary qualifications preserved

Inspected composition binding, authorization and settlement source. The reply
exception requires the declared original entry/caller/protocol/obligation and
still checks source revision/generation. First authorization may follow member
End. Stored program identities remain unchanged by ephemeral endpoint binding.
Authorization.Source.Sequence and the committed intent's sequence are distinct.

SettleInvocation inserts the publication, outcome observation and delivery row
in one transaction. Splitting this atomic evidence into ordered abstract
publish/deliver events is an explicitly stated overapproximation; model
intermediate states do not establish real checkpoint reachability. Actual
publication-only evidence must not fabricate delivery. Root publication has no
new ordinary-message obligation. These mapping caveats are accurate source
observations, with authentic dynamic binding, raw selected observations, actual
cuts, controls and SIGKILL recovery still requiring Task3 evidence.

The bounded model result is accepted within its explicit stable-authority,
finite-resource and eventual-service assumptions. No broader proof or runtime
qualification is supplied by this review.
