# Phase 1K final scoped rereview, round 2

N1: **ADDRESSED**.
Specification verdict: **APPROVE**.
Research and code-quality verdict: **APPROVE**.
New actionable findings: **none** after the qualification-label correction
identified and completed during this rereview.

This review covers the N1 fixture-isolation correction and new breakage or claim
changes introduced by that diff. It carries forward the unchanged substantive
assessment in final-r1.md; it does not repeat the whole research review, qualify
a complete J2 runtime protocol, claim a final PDF build, or authorize merging.

## Reviewed scope and identities

Read the exact final-fix review package, current decoder test, preserved original
test, implementation report, source-before/source-after manifests and archives,
final-fix gate records and outputs, single-leaf retained rejection, new recorder,
and changed narrative metadata. The manuscript and four other runtime/test
sources plus oracle remain unchanged from the approved substantive review.

| Reviewed item | SHA256 |
| --- | --- |
| Final-fix review package | 97bb192ca70bc3f53691d54954175d9567bc4c8421cbcb2d7ebf13e9a2451afe |
| Final-fix report | 65e1017a930863395ab5f3867a87385c4f07b006576539e951d03ba94a887dae |
| Current decoder test | 81513baaf1c9c8ee9359c3924436e004af0868cc54968f6f3ef458634699e1ba |
| Current five runtime/test-source aggregate | e174c4b7ca2e256259045ecdc117378111ecbb3d3de14b0ca5f7d88affb5eb04 |
| Current whole-source aggregate | 5c2931245ac033365563325e0d38e521d0dd2b5c04e0ee2fd0c80547ae326e0d |
| Final-fix source identity | 8a50e775fa49023e20451489ac4f53b6ff9f2c8b49dd1ca6640f4423afe67c01 |
| Final-fix evidence index | a9e5deb4d484908cd102ab3c6092962857d7cf47f3f6e684384f38d2e7ae7176 |
| Qualification after one-line correction | 956e35043e1135dbc6f9f2a07e7b6a337aeabc51b475adf0e3f28fa6d3407fff |
| Whole evidence index before this report | 23ff6c11da280df6f0bd5fabd6a9862c90eaf1a9a5705f59cd788ae057a86b96 |
| Source manifest before this report | b8340fe67e201a0439bd14a8d5d4bd7e76e77eb0613fcd1322d3a384fe9d3aac |

Original final-r1.md remains byte-identical at
a7e07dda44f7eaf40a371c71c13245df0050fb17d4417a610327dc7e09a5c2f9.
Its P3 diagnostic is historical evidence, not erased by this approval.

## Strengths and N1 resolution

At communication_decoder_test.go:21-45, the local fixture function looks up a
cached snapshot or obtains the required late, early or admitted state by the
same genuine public-driver paths. It is called from every dependent selected
attack and from the unrelated-order and J2-unavailable cases. Therefore no case
requires a sibling source subtest to execute first. The source controls continue
to request those same genuine states, and the admitted-unconsumed control retains
its explicit positive assertions.

Each attack still marshals its source into a separate inspection, performs the
same mutation, requires that bytes changed, and requires decoder rejection.
Source hashes now refer to the fixture actually requested by that case. There is
no missing-fixture skip, invented empty source, swallowed panic, weakened
correlation, replacement transition, or bypass of the decoder.

The cache is used by sequential subtests and introduces no concurrent map access.
Only this test file changed among the five runtime/test sources. Decoder
implementation, driver, process code, automatic test and oracle are unchanged.
The change does not affect the previously reviewed protocol semantics.

## Diagnostic and recorded qualification

I reran the exact original single-leaf diagnostic once on the corrected source,
from the worktree root:

~~~sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=1 GOCACHE=/private/tmp/alan-phase1k-runtime-cache go test -tags integration -count=1 -timeout=2m -run '^TestCommunicationDecoder$/^wrong-dispatch-link$' ./internal/substitutiontest
~~~

Observed exit **0**, package elapsed **4.485s**. This is a new reviewer diagnostic,
not a rerun of the broader qualification campaign.

The source-bound worker records independently preserve:

- Exact original-command RED: exit1, 0.219s, original decoder-test hash
  59e3751bacfd411ec6857ea18a3d4251a0e4865a20365aaca5194840d64ed8b2.
- Exact corrected-command GREEN: exit0, 4.054s.
- Verbose single-leaf proof: exit0, 4.779s; exactly
  TestCommunicationDecoder/wrong-dispatch-link is selected, without source-*
  siblings. Its retained rejection says "report acknowledgment original admission
  mismatch", with distinct original and mutated hashes. It is the intended attack,
  not a zero-selection pass or unrelated setup failure.
- Both decoder top-level tests under normal and race: exit0 with 21 passing
  leaves each, 18.190s and 253.591s respectively.
- Focused integration-tagged vet: exit0.

All corrected gates bind the same five source hashes. Their recorded command,
environment and output hashes were checked. Local/offline/readonly Go settings,
CGO, absent provider credential and separate evidence destinations are retained.
No broad or race test suite was rerun during this rereview.

## Qualification-label correction

The initial narrative diff correctly separated final focused decoder coverage
from the complete R2 27-case normal/race campaign, but one sentence in
02-qualification.md still called R2's semantic RED "current-source".
I requested the precise replacement "The R2-source Send-is-enough RED exits1."
The controller made only that one-line clarification and refreshed document
metadata; I inspected the corrected text and hashes.

With that correction, the narratives state the source scopes accurately:
e40b4b6 retains the full 27-case normal/race campaign and its semantic RED;
e174c4b7 has the latest 21-case decoder normal/race, single-leaf and focused-vet
qualification. No latest-source full driver/process campaign or RED execution
is invented. The unchanged driver/process results retain their prior scope.

## Preservation and verification

Read-only standard-library checks verified all 621 prior runtime/review artifacts,
all 129 entries of the final-fix artifact index, both five-member source archives,
current five-source hashes and aggregate, and source/output bindings for every
final-fix command. The before/after manifests show exactly one changed source:
communication_decoder_test.go.

The refreshed whole index has 935 files and the source manifest 445 entries;
all listed file bytes and hashes match, and the whole-source aggregate recomputes
to 5c2931245ac033365563325e0d38e521d0dd2b5c04e0ee2fd0c80547ae326e0d.
All separately hashed narrative/manuscript documents and all 131 protected
baseline files match. git diff --check exits0.

Manuscript hashes remain exactly the approved ones:
Markdown 3ab2149e857069dfcd1060bb80706ff6a6a5f506815fc7147f4ef6ba04407843;
LaTeX 1951b38cd515e8753c13fdde60d34df261c202095ab323acd6d4ee010e980701.
The oracle remains
7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4.

Only this review report was written by me. No source fixes, index/HEAD/branch
mutations, commits, TeX invocation or nested agent was used.

## Issues and recommendations

Critical: none. Important: none. Minor: N1 is closed; the stale RED-source label
was corrected during review. No unresolved actionable item remains in this
scoped correction.

Proceed with the one already-planned end-phase Tectonic invocation, its
renderability record, final metadata/diff checks and draft PR. Preserve the
source-scope distinctions above in final reporting.

## Declined to judge

- Unchanged substantive model/runtime/manuscript conclusions: covered by final-r1.md and outside this N1-only rereview.
- Complete J2 correspondence, universal scheduler/protocol claims and distributed recovery: the original explicit limits remain unchanged.
- Full driver/process qualification on e174c4b7: not run or claimed; this correction changes decoder test setup only.
- Final PDF, final commit identity and draft-PR metadata: still pending finishing steps at this review point.
- Merge authorization: explicitly absent.

## Assessment

**Ready to merge? No:** finishing artifacts remain pending and merging is not
authorized. **Ready for the final renderability and draft-PR gates? Yes.**

N1's concrete filtered-test failure is fixed without changing the investigated
semantics or weakening its adversarial checks. The latest evidence and narrative
accurately state both the new focused qualification and the preserved broader
R2 campaign.
