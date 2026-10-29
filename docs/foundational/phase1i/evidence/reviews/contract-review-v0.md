# Phase 1I Stage 1 independent contract and execution-plan review

Decision: **REJECT** the current specification/plan freeze. The bounded research direction is appropriate; the findings below require specification corrections, not scope expansion or a runtime change. Candidate implementation authors should remain gated until a scoped rereview accepts the revised contract and author/model briefs.

Date: 2026-10-28. Reviewed the Stage 1 candidate versions of `assignment.md`, `00-plan.md`, `01-substitution-contract.md`, `05-decisions.md`, claim audit and API audit in `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`. Also read repository AGENTS.md, inspected MD §4.5 and corresponding TeX definitions/lemma/theorem, public author invocation construction/tests, and `internal/executor/invocation_outcome.go`. This review performed no implementation, model execution, baseline rerun, external-source verification, paid calls or repository edits. This report is the sole authored file.

## Question and assessment

Can a separately checked component relation support a nontrivial response and conditional-completion guarantee in the declared finite contexts, without importing the existing unproved general congruence argument?

Yes, this is a suitable bounded investigation. However, the present document does not yet freeze enough of that local relation or of the context transitions to support the claimed Stage 1 gate. The current plan permits decisive semantic choices to occur after candidate authors start. The following are specification defects, not executed counterexamples to preservation.

## Required corrections

### R1 — Freeze the component relation, environment alphabet and reference product before candidate authors start

Priority: P1. Locations: contract lines 10–14, 27–31, 93–107 and 115–121; plan Task 1/Task 2.

The contract says the component's open graph satisfies response, demand and waiting rules under all environmental actions, but does not define the graph relation, admissible environmental actions, or how an implementation graph is obtained. The model rules are explicitly deferred until “before census,” which permits them to be selected after A/B exist. There is also no explicit reference `C[K1]` behavior/check corresponding to the assignment's premise. Directly checking two context products is useful bounded evidence, but without this separation it can collapse into contextual testing dressed as independently established refinement.

The shared-budget rely condition is promising, but “checked from actor residuals” needs an exact contract-based meaning. Entitlement must not be recomputed from a faulty candidate's demands or completed executions so that an over-spending or stuck implementation turns its own failed case into an ineligible case. Likewise, unspent declared allowance can disappear when an optional branch becomes impossible, whereas an accepted debit cannot be refunded. These are different operations.

Concrete correction:

1. Before dispatching A/B, add a small textual transition table for K1 and its open environment. Specify request admission versus consumption, response commitment versus admission, local terminal/freeze versus invocation publication, helper commit/admit/refuse/outcome/consume, and closure. State the exact rule guards, finite state fields, and whether closure disables which future actions. A short table is sufficient; a generic DSL or fully implemented checker is unnecessary at this stage.
2. Define the asymmetric local check precisely: which component events must match a K1 transition or permitted stutter, which inputs/environment transitions must be accepted, which terminal/pending states are allowed, and what additional rank/maximal-execution condition rules out silent nonresponse in entitled answer mode. Trace inclusion alone is not that progress condition.
3. Freeze the local environment alphabet independently of context enumeration. If peer debits are represented locally, define their bounded multiplicity, guard and charge, include the shared remaining ledger and closure bit, and state their rely constraints. Alternatively state a simpler explicit contract-level resource invariant plus the separate finite context-product check; do not imply a general compositional proof.
4. Define a small nondeterministic specification K1 that permits the local-answer and one-helper realizations, and check the selected guarantee for its named products separately from A/B conformance and A/B products. If the project deliberately chooses only bounded products rather than a reference-product proof, state the weaker evidence category plainly and retain the local relation as a distinct result.
5. Define demand accounting by logical tickets or an equivalent exact rule. Entry `(1,1,0)`, optional helper `(1,1,0)`, peer `(1,1,0)` and wrapper `(1,1,0)` are counted once. Commit creates no runtime debit; accepted admission transfers its ticket to spent; denial adds no debit; terminal branch exclusion may remove unrealizable *future demand* but never refund spent. At every prefix check `spent + remaining contract demand <= limits`. The initial entitlement and every subsequent permitted rely transition must be defined from K1 and the declared context, not retrofitted from candidate outcomes. A candidate exceeding K1 is a conformance failure, not grounds for removing its context from the completion class.
6. State how actual candidate code connects to independently authored component graphs. For example, independent graph transcription from each frozen artifact plus an explicit reviewed instruction/event correspondence, with runtime samples checking projection membership. Do not claim exhaustive concrete-runtime conformance merely because independently invented A/B-shaped automata pass. Keep shared record-decoder trust explicit.

Acceptance evidence for rereview: revised relation/transition/entitlement definitions, corresponding Task 1 gate and Task 2 handoff requirements. Model execution is not required to repair this pre-implementation finding.

### R2 — Disambiguate local terminal status, public invocation envelope and wrapper behavior

Priority: P1. Locations: contract lines 40–48 and 63–69; `internal/executor/invocation_outcome.go:324–439`.

The `absent` mode currently says “terminate successfully without a response,” followed by “Success status alone therefore does not entail the answer guarantee.” That is true of the local projection but false if read as successful invocation publication. Inspection confirms that a succeeded freeze with no accepted reply produces `failed/missing_reply` (lines 426–429). This distinction affects both the specification graph and the wrapper. The API audit already identifies it; the parent independently flagged it during this review.

Concrete correction: freeze a mode/outcome table with separate columns for local projection status, invocation status/reason, response presence, and enclosing wrapper status/reason. In an open pure case:

- answer: local succeeded; child succeeded/member_succeeded; response present and complement/correlation valid; wrapper succeeds with its own correlated reply.
- fail: local failed; child failed/member_failed; response absent; wrapper failed/member_failed.
- absent: local succeeded; child failed/missing_reply; response absent; wrapper failed/member_failed. Preserve the original child reason separately.
- pending: local open at an unmatched ordinary receive; no child terminal publication; wrapper remains open awaiting the child outcome.
- entry rejection: no candidate run; retained rejected/reason envelope and no response; an already-admitted wrapper maps it to its own failure.

Qualify these as the no-closure/no-unresolved-descendant cases; preserve actual closure reasons/precedence separately. Explicitly state that default-bound response fields are not evidence of presence. The existing `answer` label should be used consistently instead of the currently unnamed “success mode.”

Acceptance evidence: revised table and matching candidate-author requirements; no new runtime behavior is needed.

### R3 — Complete the finite context specification for all enumerated modes

Priority: P2, gate-blocking because it affects the frozen state census. Locations: contract lines 76–89 and 123–133.

The sequential context only describes what happens once it obtains the answer, yet the declared exhaustive census includes fail, absent, pending and refused admissions for every shape. It is presently unspecified whether the caller terminates, remains blocked, still invokes S, or produces some other public outcome on a non-answer terminal envelope. The bare/parallel caller's own completion behavior and S's reply/terminal protocol are also not fixed. Those choices change the explored graph, spent/remaining demand, stuck counts and boundary observations, so they cannot be quietly selected by a fixture author after the freeze.

Concrete correction: give a short residual-control table for each caller shape and a fixed one-shot protocol for S. A narrow reasonable choice is: await the original candidate envelope; on succeeded+present, consume/forward the exact response and (sequential only) invoke/await S; on any terminal non-success, terminate failure without starting S; on pending, remain awaiting. State how parallel records the candidate and independent peer outcomes and what outer completion it promises. Specify S's request, constant response, success and refusal behavior, and the wrapper table from R2. If another behavior is chosen, freeze it explicitly; no particular choice above is mandatory.

Keep the five named templates and 3,840 base configurations; that multiplication is correct. Record local graph initial states and the mapping of each configuration to the fixed callers. Closure-prefix/effect/probe families may remain separate from this count, but freeze their indexed scenarios before execution and never add them to the 3,840 figure by implication.

## Clarifications to carry into the revised briefs

These are not independent demands to enlarge scope:

- The identity policy is directionally correct. Explain that consistent alpha-renaming is per trace/correspondence, not a bijection between all private names of A and B: one may contain a helper and the other may not. Preserve public anchors, equality/distinctness, parentage, operation links, root affiliation and own-state restart continuity. Additional private names and visible charges are permitted by refinement; do not erase them to assert pairwise equality.
- Replace “prior authorization/reservation may still permit launch” with the precise condition: a previously allowed final reservation can survive closure; prior source authorization alone does not bypass the later root reservation check. The preceding closure rule otherwise states the correct boundary.
- The optional unknown-to-success negative must not be counted if the runtime join suppresses successful publication. Keep the actual attempted fault and guard result if exercised; use wrong correlation and an actually accepted extra subtree charge as the decisive candidate-owned negatives.
- Give candidate authors the same frozen protocol/schema and API references and record first-version hashes. Require meaningful permitted structural difference, but do not coordinate source logic. The current public `New() (author.Template, error)` interface and exported construction are feasible by inspection.
- Step/dispatch/byte bounds may be justified from the frozen finite artifacts before runtime qualification; watchdog exhaustion must remain an incomplete run, never a legitimate semantic terminal state. A no-op scheduling loop must not turn a finite acyclic semantic graph into a false cycle/rank argument.

## Findings that support the chosen scope

Gs depends on application payload/correlation and declared demand, rather than only runtime provenance. Gc requires actual answer delivery and peer completion under explicit resource/progress assumptions. Sufficient budgets 2 for bare/hidden and 3 for sequential/parallel/wrapped admit nonvacuous cases; insufficient budgets remain safety probes. The all-prefix maximum-demand approach can preserve real competition without assuming priority or isolation once R1 is made precise.

The weak-budget pair is correctly separated from Definition 4.6: its unequal matched costs cannot refute a theorem requiring equal costs. The equal-charge extra-step probe also correctly requires checking its local relation and abstract trace sets before interpreting a concrete scheduler winner. The existing manuscript's peer-interference and invocation proof obligations remain open; no general theorem is established or falsified by this review.

Hiding retains charges/refusals/obligations and does not suppress endpoint effects. Pure own-state recovery, fresh-process unknown recovery, and live late transport are correctly distinct profiles. The selected acyclic shapes avoid reliance on the unresolved general loop-ranking argument. No production repair or new effect reconciliation API is justified at this gate.

## Rereview and stopping boundary

Preserve this REJECT and the reviewed draft, resolve R1–R3 in the contract/plan/briefs, then request scoped rereview. The expected correction is a precise small model specification, not a general congruence proof. Following approval, independent A/B/model work may begin; final evidence must still distinguish bounded model results, inspected correspondence, sampled runtime behavior and historical baseline evidence.
