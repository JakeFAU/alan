# C_reason qualification and compiled correspondence

Reference APPROVE preceded this work. All44 frozen inputs remain unchanged against
`../reason-reference-r1.json.gz` (archive a18845104f33dc1950795ae0d48c1d880456f3c7a2ecd24ac78442653224aeaa).
The census is the declared id{1,2}/value{0,1}/four-mode, original answer S,
I=R3/depth2/E0/fixed1 domain. K1 completed before A/B in normal and race profiles.

| Product | Configs | Model/product states | Edges | Terminal/pending maxima | Largest rank |
| --- | ---: | ---: | ---: | ---: | ---: |
| K1 |16|6072|12880|24/8|7900|
| A |16|2680|5414|12/4|5806|
| B |16|6460|13636|12/4|7913|

All rank/state/prefix/component/G failures, rejections, stuck maxima, cycles,
cutoffs and counterexamples are zero. Normal/race census structures are identical.
K1 retains both choices in every mode (spending2/3); A spends2 and B3. Rank starts100,
cap10721. Every maximum satisfies G_outcome. Separately, the four answer inputs per
product satisfy G_answer after forgetting only the additional classification event.
Fail/absent are correct classified failures, not counterexamples to answer-only G.

Local A/B each cover16 exact configurations with closure-enabled relation and
separate Closure=false progress. A:884 pairs/468 component/1688 environment checks,
180 stutters; open376 states/544 edges,12 terminal+4 pending maxima, rank<=4906.
B:2320 pairs/1328 component/4168 environment checks,496 stutters; open824 states/
1244 edges,12 terminal+4 pending maxima, rank<=6813. Both start1400, cap8321.
All relation/progress/maximum failures are zero. Exact reasons, helper/request
consumption, q spending and unmatched pending receive are checked at every maximum.
S projects only to the arbitrary-timing q debit; its payload/lifetime and parent
joins remain separate product obligations. Eventual eligible service is assumed.

Mapping checks all143 instruction occurrences: original A13/B16/helper7/S3 and
two52-instruction parents. All32 input/binding constructions have the same code per
binding; A/B differ only in candidate invocation target/receive source/author ID
and required catalogue closure. Original template/program bytes match the frozen
artifacts. An independent finite post-join evaluator checks28224 envelope pairs
per binding against the policy table, omitting absent payload fields; every guard,
setter and End PC is exercised. This is artifact checking, not model/runtime census.

**Classification mapping:** Set at answer18, failure31, missing-reply34 or the
12 defensive setters reaches its adjacent End in the SAME committed history step.
The ordered abstract expansion is parent_classify then parent_decide; there is no
separate runtime commit/checkpoint between them. Executing End is a later commit;
root publication is later protocol settlement. Pure guards/expected_value writes/
jumps stutter. The model's intermediate state overapproximates runtime granularity.

**Pending mapping:** authenticated A branch9→second Receive11 or B branch10→second
Receive12 expands to private_step then pending_receive. The second receive does
not consume anything. Verify the original bound program/first request, actual
executed branch/open post-PC, B's original helper awaitPC2 and absence of a matching
release. Resolve its caller/provenance/operation match through the original binding.
Requested mode and generic open status alone do not establish this boundary.

Artifact-derived local-commit bounds are A28/B38 (parent19/A6/B10/helper6/S3),
including conservative paths beyond unmatched receives without assuming release.
Dispatch bounds6/9 count admission with atomic request insertion, original reply
message delivery and ready outcome settlement/delivery; root settlement is uncharged.
Max retained program21703 bytes; conservative bound-program74951, projection28042,
protocol envelope53056. Common128KiB bound fits all1MiB caps; steps/dispatches200,
capacity1. The parent now has17 scalar registers. These finite pure-profile bounds
are separate from model rank and from measured samples or arbitrary adapters.

`records.json.gz` preserves exact commands/env/exits/leaves/source hashes, all64
product maxima with traces per profile,32 local maxima, compiled artifacts/mapping,
and checker sources. Normal/race each pass80 qualification leaves (16K1+32A/B+32local).
Vet/format/discovery/toolchain/module/diff checks pass. One artifact-check command
failed with a template JSON descriptor lookup KeyError; its exact source/command is
preserved. Correcting that lookup alone produced the passing final mapping.

Reproduce from repository root into a fresh temporary directory. `capture.py` pins
Go1.27.3/local/offline/readonly/CGO1, removes ALAN_* and API key, and records the
explicit evidence directory. Use the exact argv/environment in `index.json`:
run `TestConsumerReasonReferenceQualification` before
`TestConsumerReasonReplacementQualification`, then `TestConsumerReasonLocalQualification`,
each once normal and once race; all selectors are anchored. Copy extract.go.txt to
that directory as extract.go, run the captured exporter and checker commands.
`bundle.py TEMP_DIRECTORY` packages records, verifies source/freeze identity and
normal/race equality; preserve existing archive/index before reproduction.

This is a bounded conditional result, awaiting independent qualification review.
No Task6 raw authentication/runtime qualification, general congruence, compiler
correctness theorem, arbitrary-context equivalence or new recovery claim is made.
