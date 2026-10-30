# Phase 1M Task 2a independent C_se reference review

Spec verdict: **APPROVE** for the construction/freeze gate.
Code-quality verdict: **APPROVE**. No actionable findings in the reviewed delta.

This approves proceeding to the separately gated qualification work. It does not
establish C_se[K1] or C_se[A/B] satisfies G_answer, certify runtime correspondence,
or approve C_reason. Review agreement is not proof.

Reviewed on 2026-10-29 by the independent m1_reference_review agent, without
subagents, commits, census execution, replacement acceptance, new consumer or
compiled-artifact inspection, or Phase 1M preflight/runtime evidence inspection.

## Question and source identity

Does the reference express the same guarded original-outcome join with only the
parent's selection order reversed, while preserving independently checkable
protocol, component, resource, progress, and finite-exploration obligations?

The frozen input is [reference-r1.json.gz](../model/reference-r1.json.gz), source
aggregate cdfe2a191641129dc591b92aabd680d238a80a1e027cfb77a82edc84fd465cd1,
archive SHA-256
3663adea727e91f714e3257dae60331d43c194b074b7bb7bfe1c7ccc5dde329f.
Read-only Python checks recomputed the archive hash and the sorted path:hash JSON
aggregate, checked every archived text hash, and compared all 36 frozen inputs
with the working files: no mismatches. Git show checks independently matched all
three archived historical shared files to reviewed source
8310bb270eed536ba717ea0e9f2a35f23ae6e68e. The complete working shared-file diff was
inspected.

Inputs read include the task brief/report/diff/derivation, assignment (especially
sections 3, 5, 7), current contract, C_se source/tests, shared Join source/tests,
and the unchanged model, candidate graph, component oracle and local relation.
Historical K1 contract and local-qualification source were read only to check
the named S-debit/progress and monitor-memory risks.

## Challenges and disposition

| Boundary | Evidence and conclusion |
| --- | --- |
| Fixed policy and historical API preservation | consumer_policy.go:24-49 always selects consumeSFirst. invocation_join.go:112-122, invocation_join_oracle.go:61-67/301-305/399-406 and invocation_join_explorer.go:84-106 retain candidate-first historical wrappers and signatures. Substituting that constant into every changed branch recovers the old receive, state, prefix and pending predicates; the remaining protocol/check bodies are unchanged. This is source inspection, with current historical qualification still required. |
| Both original joins, including failure | invocation_join.go:174-186 commits e then original s; :212-244 gates s then e and permits either decision only at stage4. The temporary old caller view is cleared at :165-170. Independently, invocation_join_oracle.go:118-135 requires each original commitment and its order, :273-294 requires original delivered unconsumed envelopes and s before e, and :82-102 requires both joins before decision/End/publication. An S failure has no early-exit exception. |
| Payload guards and truthful prefixes | Model predicate invocation_join.go:103-106 and separate retained-envelope predicate invocation_join_oracle.go:53-58 short-circuit on success/presence before reading each payload. Prefix validity, CheckJoinComponent (:330-396), and G_answer (:405-419) remain distinct. Wrong-id evidence can remain truthful while component conformance and G fail. Fail/member_failed and absent/missing_reply remain distinct; zero defaults cannot create a present answer. |
| Protocol authority and obligations | invocation_join.go:124-144 separates first invocation/reply authorization; :188-206 gates admission. Oracle :138-172 checks stable revision/generation, retirement, retained open/terminal snapshots, and the original binding-qualified reply exception. Only reply authorization admits a terminal member; invocation authorization retains its source check. Oracle :173-207 rechecks admission source/root/budget, :233-267 preserves committed-reply settlement and applicable descendant publication obligations. Raw binding authentication remains an explicit external obligation, not established by the ReplyBound Boolean. |
| Independent K1 branches | invocation_join.go:151-154 explicitly uses the Reference graph, whose model.go:274-288 emits both local/helper choices independently of mode. No selector branch removes helper work. consumer_policy_test.go:307-359 checks eight named mode/choice paths and both choices at the decision point; these are construction paths, not all interleavings. |
| Finite representation and rank | The contract's Cartesian factors cover the actual bounded fields. The prefix record has 14 Booleans, bounded rejection/reply/terminal fields and an empty-or-fixed-metadata envelope bound of 481. The constant order adds no state. The per-operation stage maximum is 27; ledger, parent, authorization and root contributions give rank cap10621 and initial100. The changed receives each increase ParentStage; original semantic stages and private instruction targets remain monotone. This is a source argument awaiting every-edge census checks, not an executed exhaustive result or concrete instruction budget. |
| Search identity, failures and fairness | invocation_join_explorer.go:58-61 keys model plus prefix monitor. For prefix-valid histories, CheckTrace's commitment/admission/reply/publication/consumption memory and the component checker's helper/reply flags are determined by that keyed prefix. Every incoming edge is checked before deduplication at :156-199; an already reported violation is not turned into acceptance. There is no failure pruning or cutoff. Rank failure leaves cycles unresolved (:203-205). S-first pending requires S consumed and stage3 (:88-97); it cannot mistake delivery for consumption. Eventual eligible service and exclusion of infinite idle/retry/restart-only suffixes remain assumptions. |
| Entitlement and future conformance | Initial e/h/s demand and one-charge admissions are unchanged. S's admission projects to one arbitrary-timing q debit; its payload/lifetime and the parent's joins do not follow from that projection. relation.go:67-108 checks component matching and bilateral environment readiness; invocation_join_qualification_test.go:77-84 separately requests Closure=false progress. The new contract correctly leaves current answer requalification and future all-mode progress open. |

No broader correctness claim follows for malformed arbitrary JoinState values or
prefix-invalid traces: the reviewed acceptance path starts from the fixed
initial state and requires zero state/prefix/component/rank/guarantee failures.
The monitor is not a raw-record authentication layer.

## Evidence inspected, not rerun

The frozen records retain the four expected RED order failures, then the final
normal/race construction commands selecting 50 leaves each (26 new, 24 existing),
with exit0 and no failures/skips; the recorded vet, formatting, discovery,
diff-check and unchanged-primitives commands also exit0. Source hashes in the
freeze bind those records to this review.

The recorded commands include:

- /opt/homebrew/bin/go test -json -count=1 -timeout=2m -run '^(TestConsumerPolicy|TestJoin)' ./research/substitution
- /opt/homebrew/bin/go test -json -race -count=1 -timeout=3m -run '^(TestConsumerPolicy|TestJoin)' ./research/substitution
- /opt/homebrew/bin/go vet ./research/substitution

I did not rerun these suites or run an additional behavioral probe: source
inspection resolved the specific doubts about the delta. New execution in this
review was limited to read-only file/diff/source-integrity inspection. No
behavioral counterexample was produced or suppressed.

## Remaining obligations

1. Run all four C_se[K1] positive configurations first, preserving both branches
   and every modeled interleaving; then qualify A/B. Report separate reference,
   replacement, local-conformance and G_answer verdicts.
2. Requalify current answer local relation plus open progress, and the affected
   historical InvocationJoin reference/replacement/local subset, normally/race.
   Shared-source preservation inspection is not fresh qualification.
3. After gate release, inspect every new compiled instruction and unchanged
   component identity; justify pure stutters, guards, handles, End and concrete
   instruction/dispatch/byte bounds. Rank does not supply those limits.
4. Authenticate original commitments/bindings/publications/delivery/selected
   history and consumed sequence before normalization. Obtain both original
   completion orders, the S-consumed/e-required open state, and dual pending
   controls. S-pending is correctly outside the S-answer product.
5. Before C_reason acceptance, qualify all added modes and their explicit
   progress/envelope obligations; current answer checks and directed K1 paths
   cannot certify them.

Only this review artifact was added. No production, reference, test, contract,
manuscript, or historical evidence file was changed by the reviewer.
