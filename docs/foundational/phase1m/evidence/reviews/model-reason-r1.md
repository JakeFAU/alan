# Phase 1M Task 5a independent C_reason reference review

Spec verdict: **APPROVE** for the construction/freeze gate.
Code-quality verdict: **APPROVE**. No actionable findings in this slice.

This permits proceeding to the separately gated qualification work. It does not
establish C_reason[K1] or C_reason[A/B] satisfies G_outcome, establish all-mode
local progress, or certify the new consumer or runtime correspondence. Review
agreement is not proof.

Reviewed on 2026-10-29 by the independent m2_reference_review agent. No subagents,
commits, product/local census, replacement acceptance, new consumer/source/artifact
inspection, or new preflight/runtime-output inspection occurred. Only this review
file was written. Go review/testing/safety guidance and evidence-before-claims
guidance were applied within the assigned construction-only scope.

## Question and frozen identity

Does the reference preserve both original commitments and S-first joins, classify
the actual consumed envelopes with distinct child/root reasons, and provide sound
finite search and all-mode maximum checks for the subsequent qualification?

The reviewed freeze is [reason-reference-r1.json.gz](../model/reason-reference-r1.json.gz):

- Archive SHA-256: `a18845104f33dc1950795ae0d48c1d880456f3c7a2ecd24ac78442653224aeaa`.
- Source aggregate: `b6745667b6c5c14aebf9100f9b0b50ea729ea5d8ed4c34c779676efdd1dbca9b`.
- Base commit: `1f71b56148f2fd6ef41261b7caf2996e8eeb2533`.

Read-only Python checks recomputed the archive hash and compact sorted path/hash
JSON aggregate, checked all archived source-text hashes, and matched all 44 frozen
inputs to working files. Every hash in the supplied complete task-5a-diff.txt
also matches its current file. Independent `git show BASE:path` comparisons found
all 29 reused inputs unchanged: 28 existing research Go files and the historical
01-contract.md. The four captured-evidence hashes match the freeze.

Read: applicable AGENTS instructions, task-5 brief/preparation/report/diff,
assignment sections 4–5/7 and their ordering/protocol context, new contract and six
consumer_reason Go files, approved C_se reference reviews, and the exact reused
Join/model/candidate/oracle/explorer/relation sections needed for the risks below.
K1's mode, helper, entitlement and environment contracts were checked directly.
Existing qualification/correspondence reviews were background evidence only.

## Challenges and disposition

| Boundary | Source evidence and conclusion |
| --- | --- |
| Domain and entitlement | consumer_reason.go:34–51 derives exactly id{1,2} × value{0,1} × answer/fail/absent/pending, 16 configurations per product, with I=R3/depth2. The new entitlement predicate includes all four modes. Original S remains fixed through JoinInitial and joinExpectedMeta. No added S/capacity/depth/cancellation crossproduct is present. |
| Both commitments and both actual joins | consumer_reason.go:89–99 retains the unchanged non-root C_se transitions. invocation_join.go:174–186 commits e then s; :212–224 permits s consumption only at stage2 and e only at stage3. The new parent first acts at stage4 and additionally requires both consumed flags (:59–62, :105–109). No failure-specific early exit exists. The independent prefix monitor retains original commitment, delivery, once-only consumption and S-first checks from invocation_join_oracle.go:118–135/273–294. |
| Observation-based classification and payload guards | consumer_reason.go:59–83 reads actual consumed status/reason/presence and only then payload; it never reads requested mode. S must be succeeded/member_succeeded, present, original id/zero. The independent predicate consumer_reason_oracle.go:13–45 restates the table on retained publication envelopes. Failed/member_failed and failed/missing_reply select distinct tags; malformed completed evidence selects invalid_outcome. Tests include deliberately failed evidence for an absent request: truthful prefix acceptance remains separate from component/G rejection. |
| Classification, decision, End and root | consumer_reason.go:105–128 emits separate events; classification alone cannot establish completion. consumer_reason_oracle.go:82–121 requires unique classification after both consumptions, then the matching decision and inherited End/root checks. The inherited root check requires member_<parent status>, so absent child missing_reply remains unchanged while deliberate parent Failure publishes member_failed. invalid_outcome is excluded from the requested-mode guarantee at :201–221. |
| Protocol boundaries and narrow reuse | The new wrapper drops only events whose operation is root. It preserves original non-root events, state and authorization data; original envelopes are not rewritten. invocation_join.go:124–144/188–206 still separates invocation/reply authorization from admission. invocation_join_oracle.go:138–207 retains stable snapshot checks and the binding-qualified terminal-reply exception, while invocation authorization has no such exception. Reply settlement/publication/delivery/selected consumption remain distinct. Raw identity/binding authentication is explicitly external. |
| K1 paths and pending | model.go:274–288 retains independent local/helper choices in every mode; helper consumption precedes the candidate mode action. consumer_reason_oracle.go:62–79 records the actual pending_receive, rejects later candidate activity/new helper commitment, and requires any committed helper already consumed. G_outcome additionally requires original request/admissions, correct consumed helper if chosen, exact charge, valid consumed S, no candidate outcome, and an unclassified open parent (:170–199). reasonMaximalKind delegates the S-first stage3/S-consumed boundary, and exploration establishes maximality from zero successors. A finite pre-pending prefix cannot certify progress. |
| Model and monitor identity | reasonSearchKey includes the complete comparable model and prefix state (consumer_reason_explorer.go:42–45); the added classification and CandidatePending memory are keyed. For prefix-valid histories, CheckJoinComponent/CheckTrace memory is determined by keyed operation commitment/admission/reply/publication/consumption records. Their extra helper/reply predicates add no hidden path history. Every incoming edge is checked before deduplication (:138–175), so a merged state cannot hide a newly arriving violation. Prefix-invalid paths already prevent acceptance; no broader guarantee for arbitrary malformed states/traces is claimed. |
| Finite domain, rank and fairness | The contract's 5×old-model and 10×old-monitor factors cover the added five-valued tag and monitor Boolean. Original parent stages remain 0..6. Classification adds exactly one monotone 100-point milestone; other transitions preserve it and inherit JoinRank. Recomputing the conservative caps gives 10721 product and 8321 open-local. The local state formula covers root outcome fields, peer ticket, private PCs/registers and two live operations; unused slots and e[7] stay fixed in the reused graph. Every product edge, including revisits, checks increasing rank/cap; failure leaves cycles unresolved. These are source-derived bounds, not census counts or concrete operational budgets. Eventual eligible service still excludes infinite idle/retry/restart-only suffixes. |
| Local relation and all-mode progress | consumer_reason_local.go:99–132 reuses the exact closure-enabled relation, then independently explores Closure=false while retaining every actual maximum. CheckConsumerReasonLocalMaximum (:17–72) checks exact requested answer/fail/absent envelopes or designated pending PC, first request, original successful consumed helper for B, q spent, no closure/refusal, and exact spending2/3. :133–142 rejects relation/rank/cycle/cutoff/stuck/safety/trace/maximum failure and empty maxima. The old answer-only completion predicate cannot suppress these added checks. ExploreGraph checks every incoming edge before deduplication; its CheckTrace memory is determined by the retained protocol fields for these fixed graphs. |
| S debit projection and scope | model.go:222–227 provides one arbitrary-timing q debit. The C_reason contract maps original S admission to q and other S actions to stutters; authorization stutters relative to the older coarse local relation while being checked separately in the product. This covers resource interference, not S payload/lifetime or parent joins. The unchanged relation checks component simulation and bilateral environment readiness, not liveness by itself. |

No source-level doubt remained that warranted a new behavioral probe. This is a
construction review supported by source arguments and captured focused checks,
not enumeration of every permitted reference execution.

## Captured evidence inspected, not rerun

[construction.json.gz](../model/reason-construction/construction.json.gz) hashes to
`88f9e8919f67124048d083f6b6d002e966df93b1a08df686ea61ce4c86489af2`.
All 40 execution-source hashes match the final freeze. Independent parsing of
the saved Go JSON confirms 99 passed leaves per normal/race command: 49 new
C_reason and 50 unchanged C_se/Join leaves, no failed/skipped events, and exit0.
The captured selected-leaf lists match the independently derived leaves.

The selector is the literal argv element:

```text
^(TestConsumerReason.*|TestJoin.*|TestConsumerPolicy(SFirstOracle|RejectsCandidateFirst|FirstSelectionTransition|PendingPrefix|OracleRequiresOriginalJoins|HonestFailureIsNotConformance|AbsentPayloadAndFailedS|DirectedReferenceBranches|DomainAndStateGuards))$
```

The recorded commands are `/opt/homebrew/bin/go test -json -count=1 -timeout=2m
-run SELECTOR ./research/substitution` and the corresponding `-race -timeout=3m`
command. The recorded discovery, Go version, vet, gofmt and diff-check commands
also exit0; gofmt output is empty. Recorded toolchain is Go1.27.3 darwin/arm64,
with local/offline/readonly dependencies, CGO1, and the stated isolated cache.

The preserved RED record has four failed leaves and exit1: the old answer-only
guarantee rejects fail/absent/pending, and the old monitor rejects classification.
Its embedded probe source hash verifies. This is an extension-gap witness, not a
runtime counterexample. The preliminary green record retains 81 passed leaves
and its own verified source snapshot; final claims are tied to the later 99-leaf
records. No suite or census was rerun during this review. New execution was
limited to read-only source/diff/archive inspection and hash/record checks.

## Remaining obligations

1. Execute K1-first16, then A16 and B16, preserving every modeled branch and
   interleaving. Require all state/prefix/component/rank/G checks and every
   maximum to pass; report G_answer and G_outcome separately. Directed examples
   cannot substitute for these all-execution results.
2. Execute the exact all-mode A/B local relation and separate open-progress
   qualification. Report exact answer, fail, absent and intended pending maxima;
   zero finite-prefix safety failures alone are insufficient.
3. After gate release, map every new compiled instruction, classification setter,
   branch, actual decision and End to events or justified stutters. Derive finite
   operational instruction/dispatch/byte bounds from those artifacts.
4. Authenticate original commitments, bindings, publications, retained outcomes,
   selected consumption and actual classification/decision instructions before
   normalization; retain reason-confusion and observation-over-mode controls.
5. Complete representative runtime evidence, independent qualification review and
   synchronized manuscript reconciliation after results settle. No general
   congruence, arbitrary-context substitution, exact-debit equivalence, universal
   compiler correctness or new recovery theorem follows from this approval.

No production, reference, tests, contract, archived evidence or manuscript was
changed by this reviewer.
