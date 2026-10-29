ALAN Phase 1I Codex Brief
Behavioral contracts and contextual substitution
Work in https://github.com/JakeFAU/alan.

The approved next research question is:
Can independently authored agents that satisfy the same public behavioral contract be substituted within any admissible composition while preserving every guarantee that the composition derives from that contract?

Start this bounded research increment and carry it through implementation or a conclusive negative result, qualification, manuscript reconciliation, independent review, and a draft PR. This is an execution assignment, not a request to return only a plan.

Make routine design and implementation decisions autonomously, explain their reasons, and record them. Use independent agents for substantive work where available, especially the separate implementations, model/oracle, and reviews. Resolve contract and plan review findings internally; do not add another user approval round for ordinary choices within this scope. Respect repository permissions and required gates. Stop for a material expansion of scope or genuinely missing access/authorization, after completing useful independent work.

Do not merge or release. No paid provider calls are authorized. Use controlled local endpoints for effect experiments. Do not send messages to people or request external reviews on the user's behalf; internal agent review and draft PR publication are sufficient for this assignment.
1. Starting point and governing purpose
Read the current AGENTS.md and applicable repository instructions before changing anything. ALAN is a research project about retainable computation, semantics, composition, and truthful recovery. Optimize for a precise falsifiable result.

Phase 1H was merged in PR #17 at merge commit 82573b1eaf349f9567dc34d91f6d8ddcbb6a3177. Verify the actual current main, merged ancestry, working-tree state, and relevant intervening changes. Work on an isolated branch/worktree based on the verified current source. Preserve unrelated work. Record the baseline commit and tree; do not assume the cited merge is still the latest head.

Phase 1H qualifies ordinary retained adaptive composition with result-derived ordered frontiers, direct/nested use of the identical artifact, complete-round barriers, persistent limits, shared root accounting, and selected process-recovery schedules. It does not establish substitution between different implementations. Its source-specific reports and original failures must remain intact.

Read these existing sources, following relevant references selectively:
- AGENTS.md and README.md.
- docs/foundational/phase1h/01-adaptive-contract.md, 02-qualification.md, 03-review.md, 04-runbook.md, and 05-decisions.md.
- docs/foundational/phase1h/evidence/task3/manuscript-audit.md.
- paper/alan-dissertation.md, especially §4.5: Definitions 4.5–4.6, Lemma 4.7, and Theorem 4.8. Inspect §2.5 and dependent recovery claims in §6 only as needed to resolve this increment's dependencies.
- The corresponding sections of paper/alan-dissertation.tex and relevant entries in paper/references.bib.
- The actual public author, schema, and runtime surfaces, existing independent semantic model, invocation/accounting/closure code, and reusable process-test facilities needed for the chosen witnesses.
Verify current paths and symbols. Do not infer implementation behavior from the manuscript or treat passing replay as independent semantic validation.
2. Distinguish the claims before building
The primary target is contract-relative guarantee preservation. For a behavioral contract K, implementations A and B, a contract-admissible context C, and a selected guarantee G, investigate a rule of the form:
A refines K and B refines K and C[K] satisfies G
    implies C[A] satisfies G and C[B] satisfies G.
Give this an actual definition in the chosen model. Do not make the implication true merely by defining conformance through success in every context; the point is to investigate whether a separately checkable component relation supports compositional reasoning.

For a positive result, require at least one nontrivial guarantee whose preservation depends on the candidate honoring K, rather than solely on an invariant the runtime enforces for every program. Show reachable admissible inputs and contexts where both conforming implementations satisfy it, and an executable violating implementation or weakened contract that the independent checker distinguishes. Refusing every request, excluding every competing context, or promising no relevant observable behavior is not sufficient. Any progress guarantee remains conditional on explicit environmental and resource assumptions.

Keep three claims separate:
1. Matching input/output schemas.
2. Satisfying/refining the same behavioral contract under explicit assumptions.
3. Contextual equivalence: matching permitted observable behavior in both directions throughout the declared context class.
Two conforming implementations may make different contract-permitted choices. Compare them against K; require pairwise equivalence only where it is actually claimed. Preservation of safety does not by itself establish deadlock freedom, required responses, fairness, or eventual completion.

The existing Theorem 4.8 assumes the stronger budget-indexed failure/divergence relation of Definition 4.6, including matching resource debits and readiness. Assess that theorem on its stated premises. Do not present a pair with unequal charges as a counterexample to a theorem that assumes equal charges. Separately determine whether the chosen public contract supplies those premises, supports a weaker refinement result, or is insufficient.
3. Freeze the semantic boundary
Before writing the two implementations, specify and independently review a small contract and model containing:
- Accepted inputs, allowed result relations, interaction order, response absence, and terminal outcomes.
- Environmental assumptions, authorized effects, provenance/correlation, and closure obligations.
- Resource demands and relevant remaining allowances: invocations, created runs, effect reservations, depth, and any other limit the witness can reach. Count enclosing and fixed provisioning costs explicitly.
- Legitimate pending states, unknown results, and any required progress under stated assumptions.
- The context grammar, permitted observations, shared state, and rules for compatibility/admissibility.
- The abstraction relating model traces to retained-runtime records.
Define admissibility before choosing A and B. A well-typed and runtime-authorized context is not automatically entitled to a successful-completion guarantee. Resource rules must account for prior spending, outstanding reservations, enclosing overhead, and reachable prefixes where peers can act. Equal final resource totals do not alone establish substitutability under contention; pointwise lower cost is not automatically a theorem either.

For identity abstraction, declare which names may be consistently renamed and preserve equality, distinctness, provenance, correlation, and recovery continuity. Do not erase identity fields independently to obtain matching traces. State which diagnostic internals, artifact hashes, or timing observations are outside the selected contextual guarantee. External effects, meaningful refusals, and shared charges must not disappear merely because internal ports are hidden.

A contract restriction discovered during the experiment is allowed only with the original counterexample retained, an explicit semantic justification, and independent rereview. Do not redefine admissibility after seeing failures solely to force a positive result.
4. Keep the first model small and explicit
Start with acyclic programs, one substitution position, one root, one competing peer, at most one enclosing wrapper, and one optional helper inside a candidate implementation. Use a tiny finite payload domain, one logical generation, and residual resource values sufficient to expose boundary cases; {0,1,2,3} is a useful starting set for admission/run allowances.

Cover the bare call, a call followed by another invocation, parallel competition, hiding, and a bounded invocation wrapper. State whether context construction is enumerated from a grammar or restricted to named finite templates. A list of fixtures is not exhaustive enumeration of every context within an unstated bound.

Use one controlled endpoint when qualifying the effect/uncertainty boundary. Begin with pure witnesses for completion recovery and add the smallest effectful companion required for the selected guarantees. Label separate profiles and artifacts explicitly; pure completion evidence does not establish effectful substitution. Do not extend the model into arbitrary recursion, multiple replacement sites, distributed ownership, or a general effect language.

Freeze the exact bounds, enumerator, exclusions, and initial states before claiming exhaustive coverage. Increase a bound only to address a concrete missing distinction. Explain the consequence of each bound. A search cutoff is not evidence of termination or absence of divergence; use a valid rank or cycle analysis where such a claim is made.

The primary substitution boundary is a fresh invocation/deployment. Recover each implementation from its own retained state. Replacing code underneath an existing retained continuation, migrating that state, and changing artifact versions mid-execution are outside this increment.
5. Required distinguishing experiments
Treat the following as hypotheses/probes until actually executed. Attribute each result to its exact contract, model, source, and failure regime.
A. Shared-budget counterexample to the weak contract
A context has two invocation admissions remaining. It invokes the candidate, awaits the result, then invokes S. Weak-contract witness W_A returns an integer result locally, using one admission. W_B returns the same result through one helper, using two. With W_A, S can be admitted; with W_B, S is denied. Keep this pair distinct from the conforming A/B pair under the later frozen behavioral contract; strengthening the contract may exclude a weak-contract witness.

Construct the smallest executable/model witness. Show whether it exposes an unstated resource assumption, violates the selected contract, or falls outside the stronger equivalence relation. Then check the corrected contractual reasoning, including sufficient and insufficient remaining capacity. Do not obtain the boundary by editing authoritative database state.
B. Equal-charge implementations with different internal scheduling
Construct a candidate pair differing by an additional pure local step before a contested admission, while preserving the relevant interface and charge demand. Place a competing peer under a scarce shared allowance.

Determine whether the pair satisfies the proposed local relation before testing congruence. Compare the abstract sets of permitted traces separately from the behavior selected by concrete Store.Run. Report a distinguishing schedule if one exists; report bounded absence honestly if none is found. Do not assume fairness implies confluence or change the scheduler to make the claim pass.
C. Boundary behavior and hiding
Exercise bare and wrapped uses with the chosen success, failure, absent-response, admission/refusal, and readiness cases. Preserve the difference between a child's outcome and an enclosing wrapper's own outcome. Check that hiding internal communication cannot conceal external effects, shared charges, or obligations relevant to the public contract.
D. Closure, uncertainty, and retained recovery
Use separately identified schedules where necessary:
- Closure before and after the relevant admission/authorization boundary.
- Real process interruption and fresh-process reopening at meaningful committed boundaries in the pure completion witnesses.
- One controlled external attempt with an unavailable receipt, followed by reopening: preserve the original unknown obligation and spent reservation without inventing success, failure, or permission to retransmit.
- Where supported by the existing protocol, authentic late evidence on its original operation after interruption/closure, without reopening prohibited work. Distinguish a live late-transport schedule from fresh-process unknown recovery; do not invent a new reconciliation API to combine them.
Use retained-state acknowledgements and existing process facilities. Test machinery may observe, supply declared environmental events, and interrupt processes; it must not author candidate decisions, mutate retained projections, or rebuild missing work outside the application. Report unsupported combinations as explicit limits, not as completed cases.
E. Negative controls
Include authentic ordinary implementation variants that preserve schemas and ordinary successful outputs but violate selected guarantees. Choose a small decisive set, such as an extra uncontracted charge, incorrect correlation, or conversion of unknown work into success. Require the independent checker to reject the actual executed violation for the intended reason. Invalid program construction or a fixture setup error is not a successful semantic negative control.
6. Four reviewed execution stages
Stage 1 — Baseline, claim audit, and contract
Record current source identity and relevant merge dependencies. Map the existing theorem's hypotheses to actual semantic boundaries. Produce the frozen contract, observation relation, selected guarantees, model bounds, distinguishing examples, and a concrete implementation plan using inspected APIs and paths.

An independent reviewer must challenge admissibility, progress assumptions, resource accounting, and the distinction between refinement and equivalence before implementation authors begin. A proven defect in the broad claim may redirect the increment toward a bounded negative result; it must not be concealed by quietly weakening the goal.
Stage 2 — Independent implementations and model
Give separate authors the same frozen contract and public API references. They must not copy or coordinate their implementation logic or inspect each other's implementation before their first independent versions are frozen. Record any later integration edits. Different implementations should have meaningful internal differences permitted by the contract, without artificial external effects merely to equalize cost.

Have a separate workstream build the small model/oracle from the contract. Do not import the production reducer/compiler as the oracle or generate expected outcomes by running A or B. Shared parsers or low-level record decoding may be reused if their role and trust assumptions are explicit.

If subagents are unavailable, use separate authored implementations and clearly report the reduced independence; do not claim independent authorship that did not occur.

Report model census data: bounds, generated/admissible/rejected contexts, explored states/transitions, terminal and stuck outcomes, cycle/ranking results if applicable, exclusions, and truncations. Preserve minimal distinguishing traces and the original contract version for each result.
Stage 3 — Public-runtime witnesses
Build both implementations through exported ALAN APIs, with an external-consumer boundary comparable to existing examples. Candidate applications must not use internal executor packages, raw SQL, privileged coordinator hooks, or host-side policy callbacks.

Run the decisive direct, sequential, parallel, wrapped, budget, closure, and recovery cases. Verify the model-to-runtime abstraction against actual retained observations, intents, outcomes, charges, and correlation. Keep model enumeration, sampled runtime schedules, and proof arguments separate.

Reuse existing qualified process/accounting machinery where appropriate. Do not copy the entire Phase 1H evidence infrastructure or manufacture new tooling merely to reproduce its directory volume.
Stage 4 — Claims, qualification, and final review
Resolve the selected result and update the manuscript accordingly. Qualify the exact final contributing source, review the complete branch independently, and open a draft PR. The final reviewer should assess agreement among the contract, independently authored implementations, model, runtime witnesses, and manuscript statements.
7. Manuscript work is narrowly consequential
The Phase 1H audit identifies unresolved loop-ranking and scheduler-confluence arguments. Inspect those dependencies rather than treating them as established lemmas:
- Acyclic witnesses may avoid a general loop-ranking dependency. Do not claim to have repaired Theorem 2.6 unless an actual valid measure and bound are supplied.
- Local determinism and finite reduction do not establish that shared-budget, closure, admission, authorization, and delivery transitions commute. Assess the critical pairs needed for the selected claim.
- A sound contextual relation may preserve genuine nondeterminism without requiring a unique quiescent state. Do not redesign execution merely to rescue existing confluence prose.
- Bounded invocation has admission, depth, creation, and charge semantics. Do not erase these by calling it cost-free syntactic sugar.
- If Theorem 4.8, Lemma 4.7, or their assumptions are narrowed, reconcile directly dependent recovery claims. Do not preserve an unconditional conclusion whose supporting premise has been withdrawn.
Synchronize affected mathematical content in Markdown and LaTeX. Update BibTeX only when references change, using verified primary sources. Keep unrelated manuscript obligations explicitly open; this is not a full dissertation rewrite.

Useful primary starting points, if needed for the chosen formulation, are Liskov and Wing's A Behavioral Notion of Subtyping, de Alfaro and Henzinger's interface-refinement work, and Abadi and Lamport's open-system specification work. Verify the precise result before relying on it; none supplies an automatic proof for ALAN.
8. Qualification and change discipline
Use repository-required gates and commands discovered from current source. Choose additional qualification to resolve the actual changed boundaries. Run focused model, public-consumer, process, and applicable race checks. Production compiler/reducer/accounting changes require the corresponding broader regression gates. Do not impose another multi-hour full-suite campaign solely because Phase 1H ran one, and do not omit a required gate.

Record exact command, environment, source identity, exit status, actual selected cases, and evidence locations. Confirm that intended tests executed; an empty successful selector proves nothing. Preserve original failed commands and counterexamples. Do not relabel historical results as current-source runs. Reuse prior evidence only with an explicit unchanged-input argument and honest scope.

Prefer existing runtime and language behavior. A runtime/API repair is justified only by a retained minimal counterexample showing it is required for this research boundary, followed by a narrowly reviewed fix. Keep a disproved theorem as a valid research outcome rather than expanding infrastructure to force agreement.

No generic contract DSL, certification service, new scheduler, deployment platform, provider expansion, live spending, unrestricted synthesis, automatic repair of unknown non-idempotent work, or business exactly-once claim is part of this assignment.
9. Deliverables and stopping rule
Follow the existing phase-package convention under docs/foundational/phase1i/:
- 00-plan.md: question, source baseline, exact scoped plan, and progress.
- 01-substitution-contract.md: behavioral contract, relation, contexts, observations, assumptions, model bounds, and selected guarantees.
- 02-qualification.md: results by evidence category, exact-source commands, census, counterexamples, and limits.
- 03-review.md: independent contract/model and whole-branch review findings and their disposition.
- 04-runbook.md: minimal reproduction commands for the model and decisive runtime witnesses.
- 05-decisions.md: semantic choices, rejected alternatives, contract revisions, and remaining obligations.
- A compact machine-readable result index, model and witness sources, and retained counterexample traces sufficient to reproduce the claims. Use existing repository conventions for exact names and packaging; avoid recursive copies of historical archives.
- Synchronized affected manuscript sections and the draft PR.
Complete the selected bounded investigation, including its decisive model and runtime checks, unless an identified external blocker prevents them. State whether the selected bounded claim is supported or falsified and preserve any inconclusive result honestly. The universal theorem may remain unresolved; identify the exact missing argument separately from the completed bounded result. Proof difficulty alone does not justify skipping the bounded experiments or expanding the project indefinitely. A completed negative result must state the affected guarantee, the counterexample, why the original assumptions failed, and the corrected or withdrawn claim.

Do not announce universal substitution from two implementations or finite enumeration. A universal claim requires a valid argument over the declared context grammar and hypotheses. Name proof, bounded model checking, tested runtime behavior, and open obligations separately.

Before finishing, require no unresolved actionable implementation, evidence, or reproducibility defect within the delivered scope. An expressly retained theoretical open obligation is not such a defect. If an external blocker prevents a required gate or draft publication, complete the unaffected work and report exactly what remains blocked rather than fabricating completion.

The final handoff should lead with the research result, then state what was built, which distinguishing contexts were found or ruled out within the bounds, how the paper changed, what actually ran, and what remains open. Include the draft PR and concise reproduction instructions. Leave merge and release to the repository owner.

Begin with the baseline and claim audit, then continue through the reviewed stages.