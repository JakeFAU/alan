# Phase 1L Task 2a scoped rereview — round 1

Specification compliance: **APPROVE**. Code/research quality: **APPROVE**.
R1: **ADDRESSED**. R2: **ADDRESSED**. No new blocking finding in the fix diff.

This approval covers the frozen independent reference/oracle package at aggregate
`794c6cc516b04a736a12d757212221e2bd9309f6c33c15282e2819561d7b676b`.
It clears the source-review gate for subsequent qualification; it does not
establish C_inv[K1]'s premise, candidate substitution, runtime correspondence,
finite census results or a recovery claim. The original REJECT and witnesses
remain preserved in task-2a-review.md and reference-r1.json.gz.

## Identity and scope checks

Recomputed the five-file canonical aggregate and all individual digests: they
match the revised freeze. The scoped diff exactly equals the changes from the
immutable r1 sources to current sources. Only invocation_join_oracle.go and
invocation_join_test.go changed. Contract, transition generator, explorer, rank,
bounds and all 21 inherited-source identities are unchanged.

All 16 archived texts in reference-r2.json.gz match current files. Its SHA256 is
`adebc8c4dccce81f8656ef4746680825b354f4bf4115354989b5742de2e45937`.
The r1 archive retains its reviewed SHA256
`a1c7a10f8ebc7a59dba5abc80af2580bbc497ef45e301daf92204981fd17972d`.
The original review document and independent probe source are unchanged.

Read the original findings, fix report, complete two-file fix diff, checks index,
and each captured command record. Verified every record hash and exact selected
leaf list/count. No routine suites, new probe, census/ExploreJoin, artifact
mapping, runtime execution, source edits, commits, TeX or delegation were
performed in this rereview. All passing commands below are inspected author-run
records, not newly executed reviewer tests.

## Finding disposition

**R1 ADDRESSED:** CheckJoinComponent now emits helper_result at candidate
reply_commit when the original helper was consumed unsuccessfully (oracle
lines350–360). It preserves the existing helper_join check before consumption.
The correction applies to both refusal and failed helper outcomes. It leaves
CheckJoinPrefix unchanged, so truthful failed envelopes retain their admitted
response rather than being rejected or sanitized at the history layer.

**R2 ADDRESSED:** The helper branch now tracks accepted admission, rejects an
incorrect committed answer, and requires a correct committed reply followed by
Success for an admitted helper (oracle lines328–345). Checking commitment rather
than reply admission at End preserves the existing legal schedule where first
reply authorization/admission occurs after End. Admission refusal remains the
separate no-run case. Regressions cover helper Failure/no reply, Success/no reply,
correct reply followed by Failure, and wrong reply at commitment; valid helper
success and refused/no-reply controls remain accepted.

## Exploration identity and new-breakage assessment

For any valid prefix, helperAdmitted equals Prefix.Ops[helper].Admitted.
helperReplyCorrect equals ReplyCommitted AND ReplyID=request.id AND
ReplyValue=1-request.value. The fixed config is constant throughout one explorer
run. Existing helperCommitted/helperConsumed/helperOK are likewise derived from
keyed Committed/Consumed/Outcome status and presence. Therefore the correction
adds no future-relevant history absent from the model×prefix key and needs no
new state bound. Invalid or duplicate protocol histories are already rejected
and their violations retained before deduplication; they are not evidence for a
passing model claim.

The changes add checker predicates and specification-authored fixtures only.
They do not remove transitions, erase failure evidence, weaken G, alter
normalization or change the root/child authorization semantics. No additional
blocking issue was identified within this scoped diff. The prior source-review
assessment of unchanged reference structure/rank remains applicable, with actual
edge checks and every maximal state still to be qualified after this gate.

## Inspected RED/GREEN evidence

- Original review probes: review-red exit1 against the original source hashes;
  refusal/no-reply control PASS and both R1/R2 attacks FAIL. review-green exit0
  against current source hashes; all three unchanged probes PASS. The recorded
  overlay replacement and original probe text were independently hash-checked.
- New helper-discipline regressions: regression-red exit1 with two valid controls
  passing and six attack assertions failing. regression-green exit0 on current
  source hashes with all eight leaves passing. The intermediate r1-green record
  is correctly tied to its intermediate oracle, not the final freeze.
- Final normal and race records both exit0, each with the same 24 selected leaves
  on the final frozen source hashes. The final vet record has exit0/empty output.
  Their exact argv, environment unsets/overrides, output and source identities are
  retained in task-2a-fix1-checks-index.json and its hashed records.

The repaired component checker now rejects both retained counterexamples while
allowing their truthful protocol prefixes and keeping G as a separate result.
No positive qualification conclusion is inferred from this approval.
