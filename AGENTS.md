# AGENTS.md — ALAN Research and Engineering Instructions

## 1. Mission

ALAN (Agentic Lambda Abstraction Network) is a **research project in programming-language semantics, interactive computation, and crash-consistent distributed systems**. It is not primarily an LLM orchestration product, workflow builder, or production SaaS platform.

The central research question is whether agentic computation can be a **first-class, retainable mathematical object**: finite-step residual control, typed state, correlation obligations, and effect intents that can be inspected, composed, persisted, replayed, and recovered without relying on a surviving host-language call stack.

**Optimize for learning what is true.** Prefer a smaller falsifiable semantic result over a broad demonstration; a preserved counterexample over a misleading green test; a precise limitation over a heroic workaround. Production hardening is warranted only when it enables trustworthy experiments or protects an invariant under study.

## 2. Authority: Evidence, Not Artifacts

No existing artifact is automatically authoritative, including this file, the manuscript, a theorem statement, an implementation, a test, a previous agent's conclusion, or an AI-generated review. Resolve disagreements with explicit assumptions, formal arguments, independent checks, executable witnesses, and reproducible evidence.

- **Theory proposes** definitions, invariants, and behavioral contracts.
- **Implementation realizes** a bounded, versioned semantics and exposes gaps in the theory.
- **Tests and adversarial experiments probe** claims within their stated scope; passing tests do not prove universal properties.
- **Counterexamples falsify** overbroad claims and must be retained, not explained away.
- **Independent review challenges** the result; reviewer agreement alone is not proof.

Distinguish at all times: proved result; proof sketch; model-checked bounded result; tested implementation property; conjecture; intended behavior; open obligation. Do not promote one category to another without evidence.

## 3. Research Priorities

Prioritize, in order:

1. **Semantic coherence:** explicit state, identity, authority, effect boundaries, and observable behavior.
2. **Falsifiability:** minimal counterexamples, negative tests, crash/race schedules, and independent oracles.
3. **Replay and recovery truthfulness:** no invented receipts, no silent effect duplication, no papering over unknown external outcomes.
4. **Compositionality:** derived coordination and supervision rather than privileged agent classes, subject to clearly stated limits.
5. **Reproducibility:** deterministic fixtures, pinned assumptions, archived witnesses, and commands sufficient to reproduce findings.
6. **Engineering simplicity:** the least machinery needed to answer the research question.

Do not prioritize deployment, scale, observability platforms, polished UI, marketing, GitHub stars, or generalized infrastructure without an explicit research requirement. Avoid introducing a service mesh, Kubernetes, workflow engine, distributed control plane, or elaborate abstraction solely for future-proofing.

## 4. Current Architectural Orientation (Verify Against Code)

The manuscript currently describes:

- A version-bound, finite-step deterministic local reducer over retained programs and serializable projections.
- Explicitly separated probabilistic model observations, deterministic program reductions, and authoritative world transitions.
- Typed provenance/correlation for messages, terminal outcomes, lifecycle operations, and external receipts.
- Distinct computation, subject, logical generation, activation epoch, and authority-revision identities.
- Orthogonal provenance, communication, and lifecycle-authority relations.
- Durable intent commitment separated from authorization, application, receipt recording, settlement, and consumption.
- Honest epistemic treatment of suppressed, unknown, known, and conflicting effect outcomes.
- Bounded composition, structured collection iteration, and resource accounting under declared envelopes.
- Replay that checks historical internal consistency without reissuing external effects.

**These are research hypotheses/contracts to verify, not a license to assume the code already realizes every manuscript claim.** The present single-host SQLite implementation and its bounded semantics must not be represented as a proven distributed implementation. Do not import architecture or dependencies from unrelated projects.

## 5. Paper–Implementation Consistency

The research manuscript has three representations:

- `paper/alan-dissertation.md`
- `paper/alan-dissertation.tex`
- `paper/references.bib`

Treat the manuscript as a proposed account of the computational model, **not an unconditional implementation specification**. Treat the code as an executable realization, **not automatically the correct semantics**.

For every change that affects semantics, invariants, language constructs, identity/authority, composition, durability, resource bounds, replay, formal claims, or qualification:

1. **Paper → implementation:** Identify manuscript requirements not met by the code. When a requirement is sound, feasible, and in scope, implement it with qualification evidence. If it is infeasible or unsound, revise the manuscript rather than forcing a fictional guarantee into code.
2. **Implementation → paper:** When code, tests, adversarial review, or proof work falsifies or narrows a manuscript claim, revise the claim, assumptions, proof, and conclusion. Distinguish an intrinsic impossibility from a limitation of the current implementation.
3. **Unresolved disagreement:** Record exactly what diverges, the evidence, the consequence, and the next proof or experiment. Never silently mark it resolved.
4. **Three-file synchronization:** Keep the Markdown and LaTeX mathematical content aligned. Update BibTeX when references change. Preserve notation, theorem numbering/cross-references, hypotheses, counterexamples, and actual qualification status. Do not invent citations or assert that source links were checked when they were not.
5. **Scope clarity:** A mathematical model may be more general than its prototype. Explicitly identify the implemented subset, model assumptions, and unsupported generalizations rather than treating this as a defect that demands production-scale implementation.

A spelling fix or unrelated refactor does **not** require a paper edit. A semantic change is not complete until its manuscript implications have been checked. If synchronized edits are too large for the current task, state the remaining work explicitly; do not claim full consistency.

## 6. Research Workflow for Agents

### Before changing anything

- Read the relevant code, tests, research notes, and manuscript sections. Verify paths and symbols actually exist.
- State the research question or failed invariant in one or two sentences.
- Identify the smallest affected semantic boundary and the assumptions needed.
- Search for prior counterexamples and qualification evidence; avoid rediscovering or contradicting established results without explanation.

### For substantive changes

1. **Specify:** Write the intended transition, observable behavior, invariant, or theorem hypothesis precisely enough to be falsified.
2. **Attack:** Construct a minimal adverse schedule, malformed observation, permission race, crash window, retry ambiguity, or resource-bound counterexample. Try to make the claim fail before implementing it.
3. **Implement narrowly:** Prefer simple, explicit state transitions and stable representations. Avoid special-case runtime roles when ordinary programs suffice.
4. **Qualify:** Add targeted positive and negative tests, boundary conditions, and reproducible witnesses. Use an independent model/oracle when code and replay share the same reducer or compiler.
5. **Reconcile:** Update the manuscript and supporting research artifacts for semantic changes; preserve failed attempts as evidence.
6. **Report:** Summarize what was proved, what was tested, what failed, what changed, and what remains unknown.

Do not claim an experiment ran, a test passed, or a proof was checked unless it actually occurred in this task or is explicitly attributed to prior, inspectable evidence.

## 7. Specific Semantic Hazards to Test

Give special attention to:

- Crash before commit, after commit/before authorization, after authorization/before receipt, and after receipt/before consumption.
- Cancellation or retirement racing authorization or delivery; no new mutation authorized after closure.
- Non-idempotent effects with unknown outcomes: never infer success/failure or blindly retransmit.
- Idempotency scope: execution-generation keys are not automatically business-level deduplication keys across retries.
- Stale activation epochs, generations, ownership revisions, and replayed or spoofed correlation identifiers.
- Confusing message admission with continuation readiness; confusing communication permission with lifecycle authority.
- Resource amplification, bounded collection nesting, root-budget conservation, and transport starvation.
- Hidden internal divergence and external mutations that must remain observable in compositional equivalence.
- Shared-implementation validation: replay agreeing with run does not validate compiler semantics independently.

If a theoretical assertion depends on fairness, termination, endpoint behavior, atomicity, trusted dispatch, finite bounds, or a particular scheduler, state that assumption visibly.

## 8. Proof and Evidence Standards

- State theorem domains and quantifiers, environmental assumptions, observation relations, and failure models explicitly.
- Separate mathematical proof from code inspection and bounded empirical evidence. A code pointer is not a formal proof.
- When presenting a bisimulation/congruence argument, examine both directions, the precise context grammar, shared state, resource accounting, refusals, and divergence.
- When presenting recovery results, distinguish safety, liveness, observational equivalence, and conservative epistemic preservation. Do not claim symmetric equivalence in a case where a missing non-idempotent receipt can change continuation behavior.
- When a proof fails, prefer weakening the theorem accurately over changing definitions merely to make the proof go through.
- Retain minimal failing traces, seeds, environment assumptions, and expected observations. Label historical findings separately from current passing results.
- Cite primary literature accurately; distinguish an established theorem, a new theorem under stated assumptions, and an engineering interpretation.

## 9. Change Discipline

- Keep modifications cohesive and reviewable. Do not rewrite unrelated packages or redesign architecture during a narrow research slice.
- Preserve existing observable contracts unless the task explicitly changes them, with corresponding negative evidence and documentation.
- Prefer deterministic local fixtures before live model calls or paid endpoints. Record costs and nondeterminism when live evaluations are necessary.
- Never delete or weaken a test simply to obtain a green build; update it only when the contract legitimately changes, and explain why.
- Do not fabricate benchmark results, qualification counts, source locations, or reference entries.
- Use the repository's actual build, test, lint, and formatting commands once discovered. **Do not assume** commands, CI providers, directory structure, or tooling beyond what is present.
- Keep secrets and credentials out of commits, fixtures, logs, and research evidence.
- For irreversible external effects or destructive repository operations, require explicit authorization rather than inferring it from general research intent.

## 10. Output and Review Contract

For a substantive research task, provide a concise final report covering:

- **Question / claim:** what was under investigation.
- **Result:** established, falsified, bounded/conditional, or unresolved.
- **Evidence:** concrete tests, traces, proof steps, and commands actually executed.
- **Changes:** code, tests, research notes, and manuscript files touched.
- **Remaining obligations:** precise gaps, including any code–paper discrepancies.

For small mechanical changes, a short summary suffices. Avoid status theater, inflated certainty, ceremonial approvals, and promises of future work. If a task is blocked, complete independently useful research and state the exact blocker.

## 11. Working Principle

**ALAN succeeds as research when its claims become more precise and more difficult to falsify, not when every demo passes or every feature ships.** A well-supported impossibility result, an exposed assumption, or a preserved counterexample is progress. Keep the paper, executable semantics, and evidence in honest conversation with one another.
