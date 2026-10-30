# Phase 1M Task 5b independent qualification/correspondence review

Spec verdict: **APPROVE** for this qualification/correspondence slice.
Code-quality verdict: **APPROVE**. No actionable findings.

This permits proceeding to Task 6. It does not certify raw runtime observations,
classification/decision authentication, or final manuscript claims. The result is
bounded and conditional on the declared stable, pure environment and eventual
eligible service. Review agreement is not proof.

Reviewed on 2026-10-29 by m2_reference_review, without subagents, commits, code
edits, census reruns, repeated passing suites, or fresh runtime execution. Only
this review file was written. The approved Task 5a reference review remains the
construction assessment; this review checks its unchanged sources against the new
qualification, artifacts, mapping and bounds.

## Evidence identity and execution records

The reviewed [records.json.gz](../model/reason-qualification/records.json.gz) is
341273 bytes, SHA-256
`80ddc382994ef20c9b16d49dd64cc22951411a6981630fdf171a1456d924c95a`.
Its qualification-source aggregate is
`e784bc7be25355c3ec58f9df1a30bd249cb49c11fbecf6811e08efd731c0c504`.

Read-only Python checks independently verified the archive/index, all six new
source snapshots and working hashes, aggregate calculation, complete task-5b diff
manifest, extraction source and all 24 correspondence-source hashes. Every one
of the 375 recorded Go/module source hashes in each of the 15 command records
matches the working source. All 44 approved reference inputs still match their
archived texts and hashes; the approved freeze remains SHA-256
`a18845104f33dc1950795ae0d48c1d880456f3c7a2ecd24ac78442653224aeaa`.

The retained extraction and final correspondence output files were also read
directly, matched to their archived parsed contents, and rehashed:

- Artifact input: `2bfa8f118f29d993477fc611cfdf62ba26510ad4b5e226ceccc78e3ce079d80b`.
- Correspondence output: `49fa38948539b16da5a2397108a8bc1457649c2c015a2c2d9fa4536e301a4f7e`.

Independent parsing of captured Go JSON confirms 80 passed qualification leaves
per profile, from three separate commands: 16 reference, 32 replacement and 32
local. There are no failed/skipped test events. The commands use
`/opt/homebrew/bin/go test -json [-race] -count=1 -timeout=5m -run SELECTOR
./research/substitution` with the three anchored
`^TestConsumerReasonReferenceQualification$`,
`^TestConsumerReasonReplacementQualification$` and
`^TestConsumerReasonLocalQualification$` selectors. These are combined results,
not one 80-leaf invocation. Discovery lists exactly these three entry points.

Captured timestamps place reference completion before replacement command start
by approximately 0.081s normally and 0.072s with race instrumentation. All four
normal/race census pairs are structurally identical, including every maximum and
trace. Recorded toolchain is Go1.27.3 darwin/arm64, CGO1, local/offline/readonly
dependencies, explicit evidence directories and the isolated model cache.
Recorded vet, formatting, module verification and diff-check commands exit0;
format output is empty. No broad-suite result is inferred.

One of the 15 commands is intentionally retained as exit1: the initial
correspondence checker raised `KeyError: 'Members'` at line189. Its saved source
hash matches the failing command. Comparing that source to the final checker
shows exactly one change, `template['Members']['entry']` to
`template['Descriptor']['Members']['entry']`. The final correspondence command
exits0; no reference, guarantee, artifact or census change accompanied the repair.
The failure is a checker construction error, not a semantic counterexample.

## Separate bounded results

The exact domain is id{1,2} × value{0,1} × answer/fail/absent/pending, original
answer S, common I=R3, depth2, E0, fixed1. The 48 product configurations retain
both permitted K1 choices in every mode. Counts below were recomputed from the
saved rows and agree with the manifest.

| Product | Configs | Model/product states | Edges | Terminal/pending maxima | Largest rank |
| --- | ---: | ---: | ---: | ---: | ---: |
| C_reason[K1] | 16 | 6072 | 12880 | 24/8 | 7900 |
| C_reason[A] | 16 | 2680 | 5414 | 12/4 | 5806 |
| C_reason[B] | 16 | 6460 | 13636 | 12/4 | 7913 |

All recorded state/rank/prefix/component/G failures, rejections, stuck maxima,
cycles, cutoffs and counterexamples are zero. Initial rank is100, cap10721.
Model and model×monitor state counts happen to agree here; both remain reported.
The unchanged explorer checks every edge before deduplication and has neither
failure pruning nor a cutoff (consumer_reason_explorer.go:113–181).

Independent read-only assertions checked all 64 saved product maxima: original
commit e before s; unique S consumption after both commitments; candidate
consumption before classification/decision/End/root publication on terminal paths;
correct original S envelope; chosen helper's correct consumed outcome; exact
mode-specific candidate status/reason/presence/payload; and charge2/3 for K1,
charge2 for A, charge3 for B. Pending maxima retain the actual pending_receive,
S consumed, no candidate outcome/consumption/classification/decision/terminal, and
parent stage3/open. Child missing_reply remains separate from root member_failed.

G_outcome passes all 48 configurations. G_answer is separately checked on the
four answer inputs per product: consumer_reason_qualification_test.go:45–57
removes only parent_classify, preserving the original envelopes, order and
decision/End/root events. The 12 answer configurations have zero G_answer failures.
Failure and absence are classified failure results, not answer-only counterexamples.

| Local replacement | Configs | Relation pairs | Component/environment checks | Stutters | Open states/edges | Terminal/pending maxima | Largest rank |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| A | 16 | 884 | 468/1688 | 180 | 376/544 | 12/4 | 4906 |
| B | 16 | 2320 | 1328/4168 | 496 | 824/1244 | 12/4 | 6813 |

All local relation, progress and maximum failures are zero; initial rank1400 and
cap8321. Every configuration uses the exact closure-enabled local relation and
separate Closure=false exploration. Independent assertions checked all 32 saved
maxima for exact answer/fail/absent envelopes or designated pending PC, initial
request consumption, successful original helper consumption for B, q spent, open
service assumptions and exact charge2/3. This establishes the stated bounded
all-mode progress obligations under eventual eligible service, beyond finite-prefix
safety acceptance. Original S admission projects to the single arbitrary-timing q
debit; S payload/lifetime and parent joins remain separate product obligations.

The test retains failures rather than filtering them, requires nonempty maxima,
and verifies both K1 charge branches (consumer_reason_qualification_test.go:25–81).
The local wrapper checks every actual zero-successor state, exact mode progress,
relation failures, rank/cycle/cutoff/stuck/safety/trace failures, and nonempty maxima
(consumer_reason_local.go:99–142). The inherited answer-only completion predicate
does not disable the new non-answer assertions.

## Artifact correspondence and concrete bounds

Inspected all 143 mapped instruction occurrences across six distinct programs:
A13, B16, helper7, S3, parent[A]52 and parent[B]52. Program IDs independently match
their actual base64-decoded bytes. Original A/B/helper programs match the earlier
artifact files byte-for-byte; the A/B/S template bytes match the frozen originals.
The exporter checks all 32 input/binding constructions, including byte-invariant
artifacts and unchanged non-input provision (extract.go.txt:50–74).

The parents differ at exactly the candidate author declaration, invoke target
and candidate receive source; all other instructions and mapping entries agree.
Catalogue differences are separately recorded. PC0/1 commits original e/s handles;
PC2/3 consumes original s/e with operation, template, provenance, generation and
obligation guards. No post-join instruction reads requested mode. S status/reason/
presence guards dominate its payload reads; candidate success/reason/presence
guards dominate its payload reads. The separate failed-absent branch tests
member_failed versus missing_reply directly.

The checker independently states the 52-instruction form and classification table
(check_correspondence.py:23–92), and its captured finite evaluation checks 28224
post-join envelope pairs per binding. The evaluator omits absent payload keys, so
an unguarded payload access would fail; all guard/setter/End PCs are reached.
This finite artifact check includes malformed defensive cases but is neither a
universal compiler theorem nor execution evidence for a particular runtime branch.

The two concrete-to-abstract expansions are sound for this scope:

- A classification Set at PC18/31/34 or a defensive setter writes the tag and
  advances directly to its adjacent End in one reducer commit. The mapping emits
  ordered parent_classify then parent_decide for that same history step. Executing
  End is a later commit; root publication is separate settlement. program.go:312–333
  and :470–487 confirm the distinction. The intermediate abstract state supplies
  no separate durable runtime checkpoint. Task6 must link the actual committed
  Set and post-PC; a retained tag alone is insufficient.
- A branch9→Receive11 or B branch10→Receive12 reaches the actual second unmatched
  receive. The mapping expands arrival into private_step then pending_receive,
  not a receive commit or second consumption. The actual guard equals the original
  public request guard before binding. composition_binding.go:287–337 resolves
  its caller, generation, invocation-request provenance and original operation.
  Task6 must authenticate the actual executed path/open post-PC, consumed first
  request, B helper awaitPC2 and absent matching release. Mode/open status alone
  cannot supply this event.

Independent calculation from every forward edge gives maximum instruction paths
parent19/A6/B10/helper6/S3, hence A28/B38 total commits within200. Including paths
beyond unmatched receives only makes these upper bounds conservative; it assumes
no pending release. runner.go:152–158 counts delivery calls; :268–272 admits then
attempts settlement; :385–397 excludes admitted but unready invocations. Admission
atomically inserts the initial request (invocation.go:480–490). Thus each admitted
singleton needs at most admission, original reply delivery and ready outcome
settlement/delivery: A6/B9 dispatches within200. Root settlement is uncharged.

The byte formulas were independently recomputed against the actual programs and
register sets. The largest parent has 52 instructions, 21703 retained bytes and
17 scalar registers. The conservative maxima are bound program74951,
projection28042 and protocol envelope53056 bytes, each below128KiB and the four
configured1MiB caps. Source inspection of binding expansion, handles, finite
protocol strings and nested envelope structs supports the conservative allowances
in this fixed pure profile. Reducer absent-response handling deletes payload
bindings (program.go:412–421), consistent with the evaluator. These bounds do not
cover arbitrary user strings, adapters, errors, retries or external interruption;
they are distinct from model rank and measured runtime samples.

## Remaining obligations

Task6 must authenticate original raw commitments/bindings/publications/retained
observations and actual selected consumption before normalization, then link the
classification Set, post-PC decision, executed End and pending arrival exactly as
mapped above. Representative answer/fail/absent/pending runtime evidence and the
reason-confusion/observation-over-mode controls still require independent audit.
Subsequent manuscript reconciliation must preserve the bounded conditional scope.
No arbitrary-context substitution, general congruence, exact-debit equivalence,
universal compiler correctness, or new recovery result follows.

No production, shared source, reference, test, archived evidence or manuscript was
changed by the reviewer. Review execution consisted of read-only source/artifact
inspection and independent integrity, saved-result and arithmetic checks; passing
suites and the product/local census were not rerun.
