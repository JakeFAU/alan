ALAN Phase 1H: Bounded Adaptive Composition
Assignment
Implement the next bounded ALAN increment in the existing repository and deliver a reviewed draft PR.

Phase 1G established execution of a variable work set supplied after initialization. Phase 1H must establish that the results of one round can determine the identities, inputs, and amount of work in the next round, using the same retained application artifact.

This is an ordinary reusable application built through ALAN's public Go surface. It is a test of the framework's expressive power and authoring usability. Use existing primitives first. A small generic authoring repair is permitted only after a concrete public-API limitation has been reproduced and independently reviewed.

Proceed autonomously through implementation and qualification. Resolve routine choices from the repository and this brief. Stop at a draft PR with evidence and a concise completion report; do not merge or release.
1. Establish the actual baseline
Read repository instructions, current main, the reviewed Phase 1G branch/PR, the public authoring APIs, the existing collection application, and the qualification tools. Record the exact starting commit and identify existing capabilities before adding anything.

If Phase 1G is still unmerged, create a separate branch based on its reviewed implementation and make that dependency explicit in the draft PR. Do not merge an earlier PR to unblock this assignment. Preserve unrelated work and historical evidence.

Fix the two small documentation issues identified in the Phase 1G review: label historical pending statements as historical, and add the exact supported direct CLI example beside the nested example. Preserve frozen historical source/evidence identities; put current guidance in the appropriate current documents.

Write a short Phase 1H contract and task plan in the repository's established documentation layout. Have an independent reviewer check the contract before implementation. Record actual files and interfaces after inspecting the code; do not assume new APIs are necessary.
2. Freeze the adaptive contract
One reusable component receives seed work after initialization and performs the following ordinary computation:
1. Accept an ordered typed frontier with explicit work IDs.
2. Invoke the retained leaf component for the current frontier.
3. Consume the required results through ordinary public operations.
4. Derive an ordered next frontier from those actual results.
5. Continue within a finite round and total-work limit, or return an explicit stopping result.
Use a round barrier for this increment: decide the next frontier after all required outcomes for the current round are known. Independent work within a round still progresses normally. Do not add speculative next-round admission or a streaming policy.

The same adaptive artifact and request/result contract must work directly and behind an ordinary wrapper. Round logic stays inside the adaptive component so additional rounds do not add invocation depth. Reuse ordinary authoring helpers where useful; do not introduce a separate invoked round coordinator that silently exceeds the depth profile.

Define before implementation:
- Typed request, leaf result, and application result schemas.
- Stable work IDs, invocation operation identities, round membership, and result ordering.
- The exact frontier derivation and stopping policy.
- How accepted round evidence, the frontier decision, and pending invocation intents survive recovery.
- How known failures, root-budget denial, application bounds, cancellation, and unknown work appear to the caller.
Existing durable application state may retain the decision, or retained inputs may support deterministic reconstruction. Do not invent a new runtime decision ledger when the current machinery already suffices. Recovery must preserve committed decisions and pending operations; it must not repeat completed external work to reconstruct them.

For this application, reject duplicate work IDs within or across frontiers before admitting that frontier's item work. Distinct IDs with identical payloads remain distinct work. This is application identity validation, not business-level deduplication.
3. Keep a narrow qualification profile
Use:
Bound	Profile
Rounds	Initial round plus at most two follow-ups; maximum three
Total distinct leaf work items	Eight across the entire obligation
Frontier/schema bounds	Within the existing Phase 1G finite value and byte limits
Physical external capacity	One and two
Maximum invocation depth	Three below a depth-zero root
Shared root envelope	32 accepted invocations, 64 created runs, 96 external attempt reservations
Provider use	Controlled fixtures; no new live spending


Round and total-work limits belong to the application request/profile and are persisted. Wrappers, leaf member runs, and external reservations remain charged according to Phase 1G. Calculate the actual deepest path and worst-case resource use before implementation. The two-program evidence helper contributes two member runs at one invocation depth.

Do not raise root limits during recovery or allocate a fresh root allowance for a new round. Distinguish protocol completion from the application's stopping reason. A round or work limit must preserve the unperformed frontier and report incomplete application work; it must not be presented as successful exhaustion of the task.

An unknown required outcome remains pending and spent. It cannot be interpreted as zero, no findings, a known failure, or permission for replacement work. No public success may conceal unresolved transitive obligations. Keep the existing closure and late-evidence rules.
4. Prove actual dependence on returned results
Before implementation, retain an independently authored expected-work ledger for a compact controlled fixture. A suitable default uses this ordinary policy:
- Observed value 0 produces no follow-up.
- Observed value 1 produces one follow-up with a stable derived work ID.
- Observed value 2 produces two ordered follow-ups with stable derived IDs.
The fixture can use the following actual returned observations:
Work item	Observation	Next work
alpha	0	Empty
beta	1	beta/check
gamma	2	gamma/left, gamma/right
beta/check	0	Empty
gamma/left	1	gamma/left/check
gamma/right	0	Empty
gamma/left/check	0	Empty


This yields frontiers of 3, 3, and 1 items, with seven leaf invocations.

Retain a paired case using the same artifacts and initial work but changing gamma's controlled observation from 2 to 0. Its frontiers become 3 and 1 items; gamma's descendants must not be admitted. Also cover empty seeds and a case whose first-round results produce no follow-up.

The expected ledger must be independent of the application's frontier-building function. Verify work identities, values, order, causal predecessors, round membership, stop reason, and charges. The policy must depend on typed outcomes, not fixture names or a hard-coded schedule.

Use ordinary bounded iteration and conditionals. Do not unroll a separate application graph per possible item or round in Go. A finite control list traversed by the existing durable iteration mechanism is acceptable.
5. Include one controlled effectful application case
Reuse the existing evidence-helper template as an ordinary composed leaf in at least one controlled-provider case. Its public result or known failure must affect the next-frontier decision. Read outcomes through the public request/result contract.

Keep this case small. It demonstrates that the adaptive application can incorporate an existing effectful composition; it does not introduce an answer-quality study or retune Phase 1F prompts, corpus, grades, or policy.

The CLI or harness may provision fixtures, submit data, run, inspect, and recover. Frontier selection and stopping policy must execute inside ordinary ALAN programs. Host Go code, test inspectors, and provider fixtures must not perform that orchestration on the application's behalf.

Demonstrate the adaptive component from a separate Go module through the public API, both directly and nested. Report any awkward authoring steps and the application code required.
6. Qualify the new boundaries
Reuse the existing qualification tools and causal controls. Add targeted evidence for:
- A round's required results being known and its decision retained or reconstructible, before any next-round item admission.
- Some next-frontier invocation intents or admissions being committed while others remain pending.
- Partial next-round result consumption while other required work remains unfinished.
Use actual subprocess termination and fresh-process recovery at acknowledged boundaries. Verify original work IDs, operation identities, decision inputs, charges, partial results, and eventual application result against the independent oracle.

Also cover:
- Out-of-order completion producing the same ordered frontier and result.
- Logical work exceeding physical capacity in direct and nested use.
- Round-limit and total-work-limit stopping with an explicit remaining frontier.
- A lower shared root budget causing a retained denial after work acceptance.
- A known failed leaf and a required unknown external outcome.
- Closure between decision and admission, with late evidence unable to reopen the closed computation.
- Read-only replay without provider dispatch or ledger mutation, using the existing supported isolation mechanism.
Add one authentic faulty application variant that ignores a returned observation or substitutes a predetermined frontier. It must reach the relevant runtime boundary and be rejected by the unchanged application oracle. Distinguish application semantic rejection from a runtime/replay protocol rejection.

Use retained events and barriers to establish ordering. Watchdogs only detect hangs. Preserve original failures and source identities.
7. Implementation and review sequence
Use sequential implementation slices with independent review at each meaningful boundary:
1. Baseline, documentation cleanup, reviewed contract, and independent fixture expectations.
2. Ordinary adaptive component, public direct/nested consumer, and controlled effectful case.
3. Recovery and negative-control qualification, final checks, runbook, and whole-branch review.
If existing public machinery expresses the application, keep the executor and persistence model unchanged. If it cannot, first produce the smallest executable authoring counterexample and compare a local authoring solution with the smallest generic extension. Review the chosen change before implementing it. Preserve scalar compatibility and existing artifact interpretation.

Changes requiring a new scheduler, lifecycle model, persistence migration, distributed ownership, unrestricted recursion, or a new privileged agent type exceed this slice. Complete useful independent work and retain a precise blocked checkpoint if such a dependency is unavoidable; do not silently broaden the assignment.

Run the repository's required default/integration/race and external-consumer checks appropriate to changed inputs. Reuse existing historical comparisons and reproduction machinery. Associate every result with its actual tested source. Do not relabel earlier suite results or repeatedly run unchanged broad suites without a concrete gate or risk to resolve.
8. Deliverables and stopping point
Deliver:
- The ordinary reusable adaptive component and public Go consumer.
- A runbook showing direct/nested use, the paired observation cases, inspection, recovery, and application stopping results.
- The reviewed contract, independent expected ledger, qualification results, source identity, and preserved counterexamples.
- An independent final review and a draft PR with its exact head and available CI status.
The final report should answer:
1. What new computation can an author express?
2. Which existing primitives were sufficient, and what generic change, if any, was required?
3. Did the paired cases demonstrate that returned results changed admitted work?
4. Did direct/nested use, recovery, closure, shared budgets, and replay preserve the contract?
5. What remains incomplete or unresolved?
Stop there. No merge, release, UI, distributed runtime, long-lived service model, new live-provider spending, or additional milestone.
Source basis
Use the repository's current Phase 1G plan, composition contract, qualification, review, and runbook as the baseline. This Phase 1H scope is the next assignment, not a claim that these capabilities have already been implemented.