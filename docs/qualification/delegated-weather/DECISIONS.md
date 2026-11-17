# Delegated Weather implementation decisions

These decisions were made within the approved implementation scope. They record tradeoffs and their rework consequences; they do not establish correctness. The [qualification record](../../../examples/outing-planner/QUALIFICATION.md) distinguishes tests, bounded model results, historical failures and remaining limits.

The entries preserve implementation-ledger order. Research release `v0.1.0` resolves to `19b6d2f575ded6f24a5f846089a94a850cf86e42`; its verified eight-file delta from reviewed `4a4a7026aead71470111f51e64650ab988ef174e` contains CI/release scripts and documentation, with no manuscript or runtime-source change.

1. Port rejected-invocation inspection separately from the feature. A refused call must not appear as unfinished descendant work. Exclude the research commit's composition-v2, draining and format changes; deliver four separately reviewed stacked PRs.
   Cost if wrong: Revise the narrow port and its review; broader composition changes remain separate work.

2. Default delegated helper admission to one, preserve explicit zero, and compare that option on repeated start. Resume derives the retained option and cannot override it.
   Cost if wrong: Correct CLI compatibility, repeated-start tests and documentation.

3. Use gpt-6.1-sol/high for ordinary integration and scoped review, and gpt-6-astra/high for the independent model/raw-reader architecture and final integrated review. Keep production implementation sequential.
   Cost if wrong: Reassign or re-review work; model selection itself supplies no correctness evidence.

4. Require the independent checker to inventory every Store table and verify complete observable coverage. State trusted input/capture integrity and the bounded correspondence claim explicitly; this is not an independent compiler or general bytecode proof.
   Cost if wrong: Revise the supported domain or reader and recapture affected evidence.

5. Keep application validation within the public snapshot API, which does not expose an inert program inventory. Require the independent raw checker to reject extra retained artifacts against the frozen manifest.
   Cost if wrong: Add a separately reviewed public API or narrow the claim; do not imply public enumeration that does not exist.

6. Place the pre-execution manifest in deployment_manifest.go. Bind version, configuration hash, complete candidates, canonical artifact bytes, each deployment's initial inputs/options and truthful source provenance.
   Cost if wrong: Correct the manifest and its consumers together and re-review their binding.

7. Separate root/pending validation in delegated_pending.go from causal operation/selection validation in delegated_evidence.go, without changing the evidence contract.
   Cost if wrong: Refactor the boundary and rerun focused evidence regressions.

8. Restrict this example to one transport identity per original message. Acknowledge manually delivered messages using that same identity before automatic running, or reuse the runner's deterministic identity. Multiple transports remain legitimate broader-runtime behavior outside this checker's domain.
   Cost if wrong: Extend the model and qualify additional transport behavior; do not misclassify it as runtime corruption.

9. Enumerate all declared known forecast vectors and no-result masks for zero through two candidates. Use the existing two-candidate fixture for the confirm-first control and preserve the separate empty-proposal late-authorization counterexample. Missing durable results remain uncertainty, and later originals are not started after an unknown.
   Cost if wrong: Revise the finite domain, fixtures and enumeration; absence alone cannot prove permanent unavailability.

10. Split the protocol model and raw reader into check.py and evidence.py. The launcher captures both source buffers before executing either, hashes and compiles those exact bytes, and uses fresh namespaces with explicit wiring and no hidden local imports.
   Cost if wrong: Rework the loader/source binding and repeat independent review and affected checks.

11. Keep the Go capture/control harness with the raw-reader implementer because its adverse schedules can require coupled reader/model corrections. Do not introduce simultaneous production writers.
   Cost if wrong: Sequential work may take longer; any unsafe interface assumption still requires correction and review.

12. Capture a complete SQLite cut with one read-only source transaction and backup. Make only the new destination self-contained, verify complete schema/typed-row equality and preserve versions, commands and errors. Never repair original evidence.
   Cost if wrong: Recapture from intact sources if the capture contract fails.

13. Relate the final concrete continuation to the model with a bounded pure-opcode walk on a fresh projection copy. Stop before observable actions, match exact tuple/payload/correlation and debt, and reject unsupported, unresolved, cyclic or over-bound states. Validate historical receive bindings from their actual predecessors.
   Cost if wrong: Correct the supported continuation relation and recheck affected positive and hostile cuts; this does not establish a universal VM proof.

14. For closed CLI Stores whose WAL header needs read-side metadata, use public read-only inspect before the unchanged capture helper. Preserve main-file hashes, logical reports and epochs, and record creation of empty WAL/read-side SHM separately.
   Cost if wrong: Recapture CLI evidence if preparation changes retained state; do not reopen the original as a runtime owner to make capture pass.

15. After validating the command's own recovery reopen, export the complete final delegated report and roots while retaining first-run progress and the recovery witness. Preserve legacy bytes and the strict checker; do not accept a stale report as the final cut.
   Cost if wrong: Redo the narrow application correction and source-bound campaign rather than relax report checks.

16. Keep this task pinned to reviewed research commit `4a4a7026`. Tagged releases are available for future ports. The release was subsequently verified as described above.
   Cost if wrong: Compare sources and requalify if a required semantic release change is later identified.

17. Preserve the CLI omission contract for absent forecast IDs. Correct the independent checker's expected optional mapping shape while keeping literal equality for present originals, candidate association, counts/order, costs and extra fields.
   Cost if wrong: Correct the comparison and recheck hostile exports on captured sources; never normalize away a real identity mismatch.

18. Use the single required full verbose integration/race suite to cover the named focused process and accepted/refused CLI race tests, retaining their capture outputs. Run heavy suites sequentially and omit only the duplicate focused race invocation; all named tests and full gates remain required.
   Cost if wrong: Run a focused check if the full log does not establish that the required tests executed.

19. Begin the final reviewer's read-only code examination during the long full race run, using committed stack `4af3786` and a hash-bound snapshot of frozen Task 5 code. The same reviewer must finish against the final committed package, full evidence, older-reader result and Task 5 gate before any APPROVE/publication. This is preparatory review, not a substitute for final review.
   Cost if wrong: Changed source requires targeted re-examination; no preliminary approval can be carried forward as final.

20. Address the observed full-race package timeout with a narrow test-harness event prefilter before `rawModelPause` performs full retained inspection. Existing match clauses determine the necessary event categories; keep all 14 cut cases, public-decision/frozen-reply controls, every predicate/assertion/capture, all tests and the 60-minute limit unchanged. Record controlled representative before/after inspection counts and diagnostic elapsed times on identical fixtures/toolchain/flags, then refreeze and requalify. No runtime/model/checker semantics or performance claim is added.
   Cost if wrong: Missing or altered cut capture must fail existing checks and requires correction/review; old full-race timeout remains a failed run, not a pass.
