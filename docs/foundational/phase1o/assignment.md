ALAN Phase 1O — Weak Refinement and the Callable-Join Rule Implementation Plan
For agentic workers: Execute this research assignment task by task. Use subagent-driven development or plan execution skills where available. Parallelize independent lemma attempts and adversarial review after their common definitions are frozen; keep dependent acceptance gates sequential.

Goal: Determine whether unchanged K1 weak refinement supplies the stronger local relation needed by Phase 1N's callable-join rule, or identify the exact obstruction.

Architecture: Hold K1, J, O, original components, semantic allowances and production semantics fixed. First formalize and audit the missing implication; then use only the finite relation machinery, candidate experiments and public executions needed to resolve its uncertain steps.

Tech stack: Existing Go research infrastructure, finite transition systems, explicit relation/progress certificates, and the current public authoring/runtime API when a concrete witness is justified. Use pinned repository dependencies.

Spec: This approved brief; unchanged Phase 1I K1; and Phase 1N's final ARG-1/ARG-2 disposition. Historical freezes and later qualification remain separately identified.

Work in JakeFAU/alan. Carry the work through independent review, scoped manuscript reconciliation and a reviewed draft PR. This is an execution assignment; do not return only a plan. Make routine research and implementation decisions autonomously.

The question is:
Does the existing K1 refinement contract already provide the information required by the callable-join rule, or is an additional locally checkable condition needed?

The objective is a justified answer. A proof, a precisely classified negative, or a demonstrated specification gap can each be useful. A failed guessed proof, rejected candidate or exhausted heuristic is not by itself a completed answer.
1. Baseline and global constraints
PR25 is merged at 42c837f545ae60676935c3f46e38d17d98f009df; the reviewed delivery head is 765226b88edd65536586cb06c4df3f53c85c4390. Verify current main, ancestry, applicable AGENTS.md instructions and unrelated work. Use an isolated branch/worktree or a verified clean existing worktree.

Read narrowly:
- docs/foundational/phase1i/01-substitution-contract.md, especially its separately checked local relation, event semantics, entitlement and progress clauses.
- docs/foundational/phase1n/01-contract.md, 02-lifting.md, 03-qualification.md, 05-decisions.md, and 06-review.md.
- docs/foundational/phase1n/evidence/reviews/reference-r1.md: its final ARG-1 statement and ARG-2 disposition are more precise than the earlier construction-freeze wording.
- research/substitution/model.go, candidates.go, relation.go, and the relevant callable_join*.go projection, relation and progress code.
- The exact production ownership/binding/authorization rules needed by an attempted lemma. Read public construction/raw mapping only when correspondence requires it.
Phase 1N established bounded nested preservation and a reviewed conditional constructor argument. It did not establish ARG-2. Its A/B checkers use a particular private-state-erasure relation with single-event matching; they do not search all weak relations.

Preserve:
- Original K1, A/B/helper/S, J/O construction and policies, and historical evidence.
- Production compiler, executor, schema semantics, scheduler, accounting and dependencies.
- I=R4, depth3, E0, fixed1 for the full J/O profile; J's prior debit1 and original S's sole independently timed debit.
- Stable identity/authority/bindings, the declared open eligible caller, no retirement/cancellation in the positive composition profile, no pending release, and eventual eligible service.
- PR19 formatting, PR20 mathematics, and the distinction between asymmetric refinement and symmetric exact-debit equivalence.
No new sibling, deeper tree, reporting protocol, effect, recovery matrix, production repair or generic context generator is part of this phase. Correct ordinary new proof/checker/example mistakes without changing the claim being tested. An exact historical defect must be preserved and reported, not silently repaired into the premise.

Do not merge, release, make paid provider calls or send messages to people.
File responsibilities
Location	Purpose
docs/foundational/phase1o/01-weak-contract.md	Frozen interpretation, candidate/domain boundaries, ownership and event visibility
docs/foundational/phase1o/02-arg2.md	Quantified claim, premise provenance, three bridge arguments and exact remaining gaps
Additive research/substitution/weak_refinement*.go and focused tests	Only necessary weak-relation, certificate, progress or finite-search machinery
Phase 1O qualification/runbook/review and compact evidence index	Actual results, certificates, searches, source identities and independent review
Optional separate public example and additive raw mapping	A justified independently authored witness, only after the realizability gate
Existing manuscript Markdown/TeX	Reconcile the resulting claim and limits at phase end


Follow existing naming conventions where appropriate. Do not modify the old eventwise checker to make it appear to have always checked arbitrary weak refinement. A documentation/proof-led result does not require a new solver or runtime package merely to fill this table.
Review focus
Keep five risks visible: silently strengthening K1; confusing one failed witness with nonexistence; moving between local-profile and full-contract quantifiers; inventing impossible authorization histories or candidate powers; and inferring progress from safety or a bounded search.
2. Task 1 — Freeze the exact implication and candidate domain
- [ ] Record baseline/protected identities and the precise historical clauses being interpreted.
- [ ] Define the weak relation, admissible candidate class, environment domain and progress predicate independently of the desired conclusion.
- [ ] Create the premise-provenance table below and resolve or explicitly identify material ambiguities.
- [ ] Obtain independent semantic review before using these definitions to accept or reject candidate experiments.
Separate the two local domains
Define W_full(X, R_w) as the original K1 relational, safety and readiness clauses over its full original local initial-state domain. Define Progress_full(X) as its separately required mode-progress and nondivergence clauses. Complete original local conformance requires an eligible relation and the applicable progress obligations. Do not confuse this local domain with the five historical context-product censuses.

Use W_N and Progress_N for those same unchanged obligations restricted to the aligned Phase 1N local domain below. This notation separates existing requirements; it does not weaken or strengthen K1:
- id{1,2} × value{0,1} × mode{answer,fail,absent,pending};
- candidate entry depth2, cap3, residual I=R3, E0, one q debit;
- closure transitions retained for local safety/readiness, as in CallableJoinLocalConfig;
- a separate open/no-closure progress check, as in QualifyCallableJoinLocal.
The composed positive environment E_N has full I=R4 after including J and excludes cancellation/retirement. Keeping closure in the local safety check does not introduce cancellation into that composition profile.

The immediate target is:
For X in the declared admissible candidate class, under E_N:

    (there exists R_w satisfying W_N(X, R_w))
    and Progress_N(X)
        implies
    (there exists R_g satisfying ARG1Compatible_N(X, R_g)).
ARG1Compatible_N must include ARG-1's permitted guard-stable weak alternative, not just its sufficient single-event special case.

A proof of this profile implication can support the full-premise implication by justified restriction of both W_full and Progress_full. A negative candidate satisfying only W_N and Progress_N does not automatically refute that full-premise implication. Either establish the original full local conformance for that witness or label the result profile-scoped. Full local qualification is distinct from rerunning all historical context products.

A universally quantified input-family claim requires the candidate premise over that family. One bad composition input can refute such a claim only after its corresponding agent-conformance premise has been established.
Preserve existential and strategy quantifiers
The target asks whether some suitable strong relation exists. It does not require every weak witness to extend.

Allow nonfunctional relations, alternative related reference states and alternative finite matching paths where the original definition permits them. A single relation, or coherent configuration-indexed family, must cover all relevant reachable component and environment choices under one fixed identity renaming. Separate favorable certificates for completed traces do not establish simulation.

Do not require a reference choice before observing the candidate event it matches unless the original semantics requires that order. Conversely, do not use hindsight to choose incompatible matches for the same related state and future obligations.

Keep three propositions distinct:
1. This supplied weak relation cannot be extended.
2. This weak-conforming candidate admits no ARG-1-compatible relation.
3. This weak-conforming candidate breaks K_join when placed in J.
Only the third is a contextual behavioral counterexample, with its reference and progress premises established.
Make visibility and ownership explicit
Specify the state projection and the ordered visible events, including multiplicity, payloads, charges, refusal reasons, terminal facts, original identities and atomicity. Specify which transitions may be hidden and exactly which facts a private stutter must preserve.

Map every choice to the written K1 contract, its fixed transition semantics or a separately identified environment assumption. Distinguish the abstract candidate class from the publicly authorable subset.

Audit candidate-owned, caller-owned and environment-owned actions separately. Original invoke_commit and outcome_consume events can be caller-owned while carrying Environment=false; that implementation flag alone is not a proof that an arbitrary candidate may control them. Equally, the prose clause about bilateral declared environment readiness must not be silently expanded to all actions without justification.

A candidate must not gain powers unavailable in the admitted semantics: changing caller/S state, reading retained authorization variables, adding operations, supplying undeclared inputs or releasing pending work. If such a restriction comes from fixed operational ownership rather than the weak relation, say so.

Use a compact table:
ARG-1 requirement	Original clause or operational rule	Status	Evidence
Each required identity, protocol, outcome, closure, ledger, depth, demand, readiness and progress condition	Exact source	Derived / independent E_N assumption / additional / ambiguous	Lemma, certificate or precise gap


Do not define admissibility by the very intermediate guard stability that ARG-2 is meant to establish. If K1 is materially ambiguous, preserve the plausible interpretations and explain their different consequences. Do not silently choose a stronger reading for a proof or a weaker reading for a counterexample. A proposed future clarification must remain separate from unchanged K1.
3. Task 2 — Attempt the three bridge arguments first
- [ ] Write a direct derivation or explicit failed obligation for each bridge.
- [ ] Challenge the derivations independently, including interleavings, ownership and progress.
- [ ] Identify the smallest finite experiment needed for any unresolved step.
- [ ] If the arguments close, validate their nontrivial finite instances and proceed to final qualification without inventing a larger campaign.
Bridge A: weak-path normalization
Determine whether matching paths can be selected so that J's relevant guards remain valid at intermediate states. Preserve ordered observable facts and the original atomic groups; do not match an unordered set of endpoints or split an atomic transition to manufacture an interleaving.

Make clear which obligations concern related endpoints and which concern intermediate states. Address additional protocol events, monotone commitment/consumption/terminal facts, optional-helper future demand, and the compatibility of path choices across future moves.

The delayed e-selection gate matters: J permits caller selection only after S has been consumed. Explain why an allowed matching path can survive that restriction without relying on an earlier selection or changing the environment's service obligations.
Bridge B: caller ownership and readiness
Derive, or identify the absence of, the framing rule that keeps original commitment and selection under their declared owner. Establish that candidate-private state cannot impose an additional guard on those caller actions.

Preserve bilateral environment readiness at private instruction boundaries. A candidate must not postpone a required environment action by inventing a private computation step.

If the fixed public API enforces a condition that the abstract K1 wording does not determine, report the two scopes separately. Do not claim a theorem for arbitrary graphs from an operational restriction established only for authored programs.
Bridge C: retained authorization extension
Construct the relation over legally reachable authorization histories, preserving each original intent's commitment, source identity/revision/generation, binding and immutable first decision.

The same coarse local state may have several legitimate retained-history extensions. Allow the required history-distinguished augmented states; do not require an unjustified single canonical sidecar assignment.

Cover every legal history admitted by the fixed environment. Do not select only a favorable authorization order, and do not introduce a Cartesian product of impossible dispositions. In particular, retain both authorization before J End and first bound-reply authorization after End. Invocation authorization has no reply exception.

Show which matching facts preserve each guard and how the extension remains closed under later service. Equal final totals or recomputation after End cannot replace original first decisions.
Progress is a separate obligation
Preserve all-mode progress/nondivergence and the actual intended pending boundary. A safety relation alone cannot discharge these conditions.

Use an appropriate rank, SCC/fairness argument or explicit equivalent certificate. Distinguish a private divergent cycle, unfairly withheld service and the designated unmatched pending receive. Do not infer progress from successful completed traces.

Keep semantic allowances fixed. Adequate finite operational bounds remain a premise: the old rank or 512-step execution setting is not a uniform bound on arbitrary private implementations. For a new concrete witness, derive an adequate common operational cap before qualification and record it separately from the fixed I/R/depth entitlement.
4. Task 3 — Build only the finite machinery needed by the unresolved argument
- [ ] Freeze any new checker semantics and independently justified tiny fixtures before accepting search results.
- [ ] Implement separate witness validation and, only where needed, a complete finite existence procedure.
- [ ] Validate counterexample/certificate reporting and incomplete-search behavior.
- [ ] Declare the finite candidate/search domain before running it; preserve all attempted classifications.
The existing CheckLocalRelation and CheckCallableJoinLifting remain historical eventwise witnesses. A new weak checker must implement the frozen weak clauses, not call the old checker under a new name.

Keep two interfaces conceptually distinct:
- Witness validation: accept or reject a supplied relation and its matching paths, giving the exact failed obligation.
- Existence checking: establish EXISTS with a checked witness, NONE with a checked exclusion argument, or INCOMPLETE when the declared relation class was not exhaustively decided.
Malformed inputs or submitted authorization histories falsely claimed to be legally reachable are input/certificate errors. They are not semantic counterexamples. A finite graph may contain harmless unreachable nodes; obligations begin from its legal initial states.

For a claimed NONE result, justify completeness for the declared finite graphs and relation class. A suitable greatest-fixed-point or simulation-game procedure may be enough; a new generic verification framework or external solver dependency is not required. A negative certificate may be branching rather than a single linear trace.

Permit all admissible alternative paths and pairs. Derive any sufficient weak-path bound from the finite semantics. A timeout, arbitrary path/state cap, greedy mismatch or failed canonical erasure is INCOMPLETE or a rejected witness, never proof of nonexistence.

Certificate validation should be independent of the search algorithm. A second copy of the same algorithm is weak evidence. Freeze expected outcomes for a small independently understood fixture set, including:
1. A valid noncanonical relation with alternative matching paths.
2. Missing bilateral environment readiness despite agreeing outputs.
3. A private divergent cycle with a safety witness but failed progress.
4. An extra visible event incorrectly treated as a private stutter.
5. A submitted authorization history claimed reachable but inconsistent with legal transition provenance, rejected as an invalid certificate.
These qualify the machinery; they need not be publicly authorable ALAN candidates. Do not predetermine a disputed guard case's answer before resolving its semantic interpretation.
Candidate experiments
Use a small declared family of independent private-state encodings, instruction splitting/grouping or private calculations near the unresolved boundary. Keep the original e/helper operation vocabulary, requests, effects and allowances.

Sharing a fixed protocol kernel is legitimate when established as a rule of the original candidate domain. Document what the kernel guarantees. A generator that inherits ARG-1 readiness or guard stability by construction cannot establish that those properties follow from a weaker premise.

In particular, CandidateGraph currently filters shared Successors transitions. Merely extending A/B with more private PCs may be a useful positive control but provides little evidence about the missing implication. Prefer a nontrivial relation over independently represented private state when an experiment is needed.

For every purported witness, record separately:
- admissibility and exact environment/input scope;
- original weak conformance and its explicit witness;
- separate readiness/progress results;
- strong-certificate existence, nonexistence or incompleteness;
- any subsequent frozen-J behavior.
Use a new candidate-parametric composition adapter only if needed. It must retain frozen J/O semantics and justify its mapping. Check its K1/A/B instances against the existing products before trusting new-candidate failures. An adapter defect is not a substitution counterexample.
5. Task 4 — Classify and minimize the result; execute only informative witnesses
- [ ] Establish the original premise before presenting a candidate as a counterexample.
- [ ] Minimize substantive witnesses without changing their assumptions or erasing causal events.
- [ ] Decide whether a public execution resolves a concrete realizability/correspondence question.
- [ ] Independently review the strongest supported conclusion and its limits.
Use the following categories:
Result	Permitted conclusion
Reviewed normalization/extension argument for every candidate in an explicit class	ARG-2 closed for that class and named environment
One witness fails, another eligible witness exists	Certificate/normalization-choice failure
Weak premise and progress hold; no eligible ARG-1 witness exists	The certificate implication fails in its stated scope; J may still behave correctly
Weak premise/progress and J[K1] premise hold; J[X] violates K_join	Genuine bounded substitution counterexample in the qualified domain
Old admissibility/readiness/progress/conformance fails	Invalid attempted witness or useful negative control
Only a bounded candidate family passes	Bounded support; generic ARG-2 remains open
The original clauses do not determine a necessary ownership/visibility rule	Exact specification gap, with consequences stated per interpretation
A checker/search cannot decide within its justified resources	Incomplete result with the unresolved obligation retained


Failure of a sufficient proof method does not by itself justify strengthening K1 or repairing the runtime. Keep any proposed weaker lifting rule or future contract clarification separate from the unchanged target.

A no-witness certificate may require a finite branching game. A genuine contextual failure may have a concrete linear execution. Preserve the appropriate object; do not demand that every logical negative be a runtime trace.
Public realizability gate
Add a separately authored public candidate and focused consumer only when it establishes something specific:
- a noncanonical yet accepted implementation with a checked relation;
- an authentic realization of a meaningful model counterexample; or
- the exact public-interface limitation of a proposed semantic witness.
J/O source and policy remain frozen; new template bindings/catalogue identities must be documented. Before any runtime conclusion, map the new candidate artifact and original emitted handles, selected observations, authority/binding decisions, charges and publications. Use supported public operations and unchanged automatic scheduling or a clearly labeled supported driver.

Do not alter authoritative state, invent outcomes or change production semantics to realize a witness. A model-only obstruction stays model-only if the public interface cannot express it. If the proof settles the question without an execution distinction, no additional runtime campaign is required.

No new recovery, effect, scarcity or topology cross-product is required.
6. Task 5 — Qualification, independent review, manuscript and delivery
- [ ] Verify the new argument/checkers and any justified concrete witness with focused checks.
- [ ] Obtain independent semantic and implementation/certificate review, then final whole-phase review.
- [ ] Resolve actionable findings and rerun affected checks with exact source identities.
- [ ] Reconcile Markdown/TeX, build the PDF once at phase end, inspect changed pages, and deliver a reviewed draft PR.
For new Go machinery, run the relevant normal/race tests, static checks, formatting and module verification using the actual pinned local toolchain, CGO and offline/readonly dependencies. Check test discovery and derive selectors from implemented names. Re-run affected historical gates only for an identified reuse/change risk or repository requirement.

Do not predeclare counts or passing commands. Record actual argv, environment, exits, search bounds, selected leaves, source hashes and certificate verdicts. Keep finite search, mathematical argument, certificate validation, source inspection and runtime execution as separate evidence categories. Source-review agreement does not make an argument machine-checked.

If a broad command times out, preserve it and identify unfinished work before targeted follow-up. Combined focused coverage is not a green monolithic suite. Corrected sources do not inherit old execution results without an explicit unaffected-source justification.

Use independent agents where useful for the three lemmas, adversarial examples and certificate validation. Keep one controlling definition/version for each gate. Freeze the weak interpretation before candidate acceptance, and prevent search output from redefining expected conformance.

Keep evidence compact: clause/provenance table, exact quantified argument, decisive certificates or witnesses, declared search domain, qualification/limits, source map, executed runbook and review disposition. Reuse historical evidence by pinned reference. New audit commands should read their immutable archive/commit inputs directly where practical, instead of requiring the live reconciled manuscripts to resemble a historical checkout. Any restoration must occur only in a disposable copy.

TeX remains the manuscript truth. Synchronize corresponding Markdown; preserve inherited formatting, mathematics and historical results. Update ARG-2 only to the scope actually established. A profile/class-specific result does not close arbitrary-context congruence or universal compiler correctness.

After source/claim review, run tectonic alan-dissertation.tex once from paper/ under the accepted warning policy and inspect changed PDF pages. Do not regenerate per implementation commit.

Finish the authorized scope when it yields a reviewed argument, minimized qualified witness, demonstrated specification gap, or honestly bounded unresolved result. Correct ordinary mistakes and test alternative witnesses before classifying a substantive failure. Do not expand the topology or silently adopt a stronger contract to manufacture closure.

The final report and draft PR must answer:
1. What precise version of unchanged weak K1 was formalized, and for which candidate/environment domain?
2. Which ARG-1 premises were derived, supplied by E_N, additional, or ambiguous?
3. What happened to each of the three bridge arguments?
4. Is the result about one certificate, existence of any suitable certificate, or actual J behavior?
5. Does it concern W_N, W_full, a finite candidate family, or a broader proved class?
6. What was proved, mechanically checked, searched, executed, source-reviewed, or left open?
7. What exact source/evidence identities and independent dispositions support that conclusion?
Draft only. The intended outcome is a sharper substitution criterion: establish what the current local contract guarantees, and expose any additional promise composition actually needs.
Pinned primary sources
- PR25
- Original K1 contract
- Exact ARG-1/ARG-2 review disposition
- Phase 1N qualification
- Original eventwise local checker
- Original transition semantics
- Candidate transcription and shared transitions
- Aligned local relation/progress checks
- Finite callable lifting checker