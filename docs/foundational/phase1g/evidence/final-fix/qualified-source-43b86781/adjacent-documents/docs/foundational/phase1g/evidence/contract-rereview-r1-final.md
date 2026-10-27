# Phase 1G contract rereview — final R2 addendum

**Verdict: PASS for the scoped R1–R6 contract review. No unresolved contract blocker remains from those findings.**

This addendum follows, and does not overwrite, `contract-rereview-r1.md`, whose NEEDS_REVISION verdict and schedules describe the earlier revision-1 text. Original `contract-review.md` likewise remains unchanged. After that scoped report, the author amended only the successful-publication paragraph to address R2.1/R2.2. This addendum reviews that amendment, not implementation.

Reviewed contract SHA-256: `a97e4c87fd2381bbfe22b6dcac272075d99b21a34fc0ed71f72ffa1920c990f8`. The preceding reviewed contract was `e1f6685d33585a136616cb4b48240d051e27fa482c5c12d5136bcc8c7530b0af`; initial scoped rereview SHA-256 is `b4ea8af1391b9a9cc14035a9c973067095f19616948437389d95c3b4aa96e739`.

## R2.1 — addressed

The all-members-success boundary now freezes committed public reply intents and requires normal reply admission to finish before evaluating response presence. A delayed committed reply keeps publication pending; it cannot cause early missing-response failure. Distinct duplicate reply intents remain a protocol violation under the existing R2 rule, while transport retransmission reuses its admission. The original schedule now has one logical result independent of reply delivery timing.

## R2.2 — addressed

The success freeze includes committed Invoke intents. Every such intent requires an immutable admission outcome before publication. Accepted admissions join their descendants; rejected admission is retained known negative evidence; a pending admission remains required work. The change explicitly preserves source/ancestor closure rules and does not grant late work merely to make the join complete. The unchanged requirement that any unresolved descendant evidence prevents success applies transitively, including descendants of a failed scope.

The paragraph also explicitly separates local terminal state from the public composition outcome and requires inspection/CLI to expose both. That resolves the potential reporting ambiguity without modifying the ordinary scalar meaning of a local terminal projection.

## R1 and remaining refinements

There is no remaining R1 contract blocker. Root accounting, scope membership, attempt-reservation charges and unsupported scoped/root Retry are sufficiently defined for implementation. Initially provisioned helpers participating in the root's work must be assigned that root by trusted provisioning; their fixed creation-cost exclusion is not an exclusion from effect budgets or closure. Implementation review should verify this conservation path alongside dynamically created members.

R3–R6 retain the addressed disposition in the preceding report. No new inherent conflict was found in this limited success-rule amendment. The resulting contract is ready for the planned semantic implementation and qualification work.

## Limits

No source implementation, test execution, live call, credential, subagent, commit or merge was performed. PASS is a specification-readiness judgment for the six findings, not proof of safety/liveness, implemented correctness, acceptance-case success, historical compatibility or production readiness. The original counterexample schedules remain regression obligations. Task and final reviews must check the actual atomic boundaries, public status/reporting, transitive joins, root/grant conservation, replay validation and causal capacity controls.
