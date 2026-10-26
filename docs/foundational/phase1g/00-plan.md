# Phase 1G: implementation plan at a glance

Start from merged main `80dead19846d9b034b4457a9436e43acb81da3df`. Existing default/integration tests and historical comparisons pass; [baseline evidence](evidence/baseline/baseline.md) preserves exact inputs and commands. Work is on `codex/phase1g-data-composition`; delivery is a draft PR.

Implement bounded invocation of retained ordinary component templates. A fixed reusable worker pool would need reentrant state and routing across both members of the existing one-shot evidence helper, plus suspended-parent handling. Retained child executions fit the existing runner more directly. [Independent comparison](evidence/feasibility.md) and [reviewed contract](01-composition-contract.md) record the tradeoff and counterexamples.

Add only native bounded typed record lists, durable snapshot iteration/append, and authenticated invocation request/result operations. Scatter/collect is an ordinary public program built from those primitives. The same component artifact accepts different work after initialization and can be invoked directly or behind an ordinary wrapper. No helper per possible item, runtime role type, second scheduler or provider-owned workflow.

| Bound | Qualification profile |
| --- | --- |
| Work-set size | 8 maximum; test 0, 1, 3, 7, 8 and rejected 9 |
| Physical external capacity | 1 and 2; waiting parents hold no slots |
| Invocation depth | 3 below a depth-zero root |
| Shared root allowance | 32 accepted invocations, 64 created runs, 96 external attempt authorizations |
| Provider use | Controlled fixture only; no new live spending |

Work-set acceptance, invocation admission, public reply recording and caller consumption remain separate durable commits. One root envelope covers nested work. Known failure, cancellation and unknown external outcomes remain distinct. Public success waits for all required work; late evidence never reopens a closed computation. Read-only replay validates retained facts without creating or dispatching work.

Implementation/review sequence:

1. Typed collections and finite iteration, preserving old scalar identities — implemented and independently reviewed.
2. Retained templates, invocation/binding/outcome ledgers, shared limits, closure and replay — implemented and independently reviewed.
3. Derived collection component, separate-module deterministic/evidence consumers and CLI — implemented and independently reviewed.
4. Causal capacity/crash/late-result qualification, negative controls, full checks, source archive, independent final review and draft PR.

The [task plan](phase1g-plan.md) carries interfaces and checklists. [Qualification](02-qualification.md) distinguishes completed evidence from pending claims. No distributed ownership, general migration, unrestricted recursion, SDK stability, quality retuning or UI is included.

Task3 is implemented and independently reviewed, including the retained JSON case-alias defect and fix. Task4 now covers actual process kills, final-source checks, reproducible archive and final review. See [Task3 report](evidence/task3/report.md) and [application runbook](../../../examples/worktree/README.md).
