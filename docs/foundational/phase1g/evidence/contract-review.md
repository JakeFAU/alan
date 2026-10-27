# Phase 1G independent contract review

**Verdict: NEEDS_REVISION before new semantic implementation.**

Reviewed 2026-10-26 at `80dead19846d9b034b4457a9436e43acb81da3df`. This reviews the initial “review draft” of `01-composition-contract.md`, before product changes. The architectural choice is feasible and plausibly the smallest extension compatible with the existing one-shot, multi-program evidence component. The draft does not yet specify several required outcomes precisely enough for independent qualification. The findings request bounded decisions, not a new architecture investigation.

## Scope and basis

Read the original Phase 1G brief, draft contract, Phase 1D.1 dispatch contract, Phase 1E/1F contracts and selected independent review records. Traced public `author` definitions/provisioning, scalar schemas, reducer instructions, `Store.Run`, source/message authorization, adapter reservation, lifecycle retry/closure, replay origins, and the evidence helper's reply path. Applied Go concurrency/context review guidance to the scheduling/closure boundary, without broad audit or delegation.

Only this report was written. No product changes, live requests, credentials, subagents, commits, or tests. Counterexamples below are reasoning schedules, not observed new-runtime failures. This review is not proof or independent verification of the historical source digest quoted by the draft. An initial attempt to write the report into the managed worktree was denied by the filesystem sandbox; no report existed after that attempt.

## Architecture judgment

Template invocation is defensible over a reusable participant pool. Current public ports wire fixed peers/generation 1, `Repeat` unrolls at authoring time, and records are flat scalars. A retained artifact cannot accept a typed runtime-sized list and instantiate the existing two-program helper per item. A pool would need invocation-local state, correlation, reuse semantics, and a solution to parents waiting on the same pool. The proposal preserves ordinary orchestration and the existing scheduler.

This is a practical minimum for the present one-shot helper, not proof of global minimality. Runtime support should implement identities, immutable binding, bounded state, admission and correlated outcomes. Selection, scatter/collect, ordering and application failure policy remain ordinary programs.

## Required refinements

### R1 — root creation, membership, debit and lifecycle

“Each admitted top-level work obligation” needs an operational identity. Name the retained event creating the envelope, its trusted supplier, and how every program in an invocation and its descendants inherit the same root. A nested component must not create a fresh root through its direct-use request path. State whether initial root runs and the root invocation count toward the declared ceilings.

Specify atomic check-and-debit boundaries: reserve all template runs together or reject without partial origins/charges. Source authorization is distinct from external authorization: `AuthorizeIntent` records source eligibility, while `prepareAdapterWork` later checks a capability grant and retains the attempt. Name the event spending allowance, whether denials spend it, and retain rejection outcomes permanently. Resource exhaustion that is an item outcome must be correlated back to the collector; a store-wide error alone does not settle that item or allow ordinary collection of the others.

Explicitly define `Retry`, `AbandonRun`, and `RetireSubject` for invocation-owned runs. Existing Retry creates a fresh generation from initial projection. Rejecting scoped Retry is a narrow safe first profile; supporting it requires root inheritance, new run/effect charges and explicit new caller correlation. Physical reopen is not Retry.

**Counterexample R1:** root R has effect ceiling 1; child C spends it and fails; existing lifecycle Retry starts C generation 2 from initial state; generation 2 gets a fresh envelope or uncharged effect. R performs two provider operations. Also test ordinary wrapper W forwarding to collection M: M's identical public request contract must not accidentally establish another allowance.

### R2 — one invocation outcome state machine

A template has several members and one public reply endpoint. “Public response or a defined terminal failure/rejection” does not define which member failures settle it, success without a reply, or reply/failure arbitration. Existing evidence code sends an `invalid_result` response and then executes `Failure`; this is a concrete arbitration case.

Specify required-member/descendant completion, the success gate, reply cardinality, and an immutable outcome envelope. Distinguish a valid public application-failure response from runtime success. Retain the settlement decision; callback order must not silently choose it. Cover missing/duplicate/conflicting reply, member failure/cancellation, admission rejection and runtime/schema errors. Corruption may remain a surfaced store error but must not become item completion.

The ordinary collector must define aggregation: ordered success/failure item outcomes, a declared disposition for failures/rejections, and pending aggregate when required work is unknown. Empty input remains successful empty output. Define partial results if the root allowance is exhausted after some children were admitted. Authentication of a reply does not establish all required work completed.

**Counterexample R2a:** member A replies success while B has a required unknown provider operation; caller consumes A and succeeds. Authentication passes but no-premature-success fails.

**Counterexample R2b:** A sends `invalid_result`, then fails. Delay reply admission until a terminal-failure result is recorded; reverse that schedule. Without arbitration, the same child history exposes different item results.

**Counterexample R2c:** A awaits B; B fails without replying. No provider is pending, but A and the collection wait indefinitely unless invocation failure has a route.

### R3 — closure scope and authorization/launch race

The draft's no-automatic-descendant-cancellation choice is viable. Complete the rule for non-root/member cancellation: does it stop that invocation's future descendants or only its addressed run? Specify normal failure, abandonment, retirement and success. Still-open descendants whose effects are denied by root closure need defined durable denial outcomes, not unexplained waiting.

Name the precise external authorization event whose survival permits work after closure. Current source authorization and adapter attempt reservation are separate transactions. Serialize the root closure check with the external debit/attempt permission and preserve historical source authorization. State whether an attempt reserved before closure but not launched may launch afterward; “may settle” alone does not answer this. Late evidence remains bound to the old operation even after newer work exists.

**Counterexample R3:** descendant commits E; `AuthorizeIntent(E)` allows it; root cancellation commits; adapter attempt is then reserved and launched. If source authorization is the cutoff this is allowed; if attempt reservation is the cutoff it must be denied atomically. A closure check outside the reservation transaction is insufficient.

### R4 — capability and binding authority

Name the deployment authority permitting a template's transitive grants. State how retained allowed template IDs/capabilities constrain child declarations and whether dependencies are closed at provisioning. Data cannot choose arbitrary artifact bytes, absolute peers, adapters, grants or root IDs. Any child selector must choose from an immutable finite authorized set.

Define canonical identity inputs and the relation between binding and program hashes. Binding either produces explicitly new canonical program bytes/IDs or is a separately versioned immutable environment; patching instructions under the original artifact ID is invalid. Reject missing members/incompatible schemas before admission charges. Cross-invocation access is limited to the designated immutable caller route.

**Counterexample R4:** caller can invoke T; T declares U with an undeployed capability. Deriving child grants from declarations alone escalates authority. Alternatively, two T invocations sharing a literal internal peer name cross-wire replies unless scope binding validates the exact invocation.

### R5 — physical capacity and saturation evidence

Capacity can coherently mean owned active adapter transports/candidates; local reductions are serialized, logical invocations bounded by total admissions/runs. State that in options/measurements. Never hold an invocation slot while a parent waits. Define how persisted root concurrency combines with global `RunOptions.MaxInFlight` and other roots; larger recovery options cannot increase the root ceiling.

Capacity 1 with a held external operation permits independent pure computation, admissions and message delivery, but cannot launch a second external operation. Once a transport returns unknown its physical slot may be released while its logical obligation remains unresolved. Remote activity is unknown, not proven stopped. Capacity 2 can demonstrate another external leaf completing while one is held. Qualification must distinguish these assertions.

**Required positive schedule:** P invokes Q; Q invokes L, at capacity 1; P/Q hold no transport slots. While an earlier transport is held, causally acknowledge Q/L admission or pure independent completion. After release, L completes. This rules out parent-slot deadlock without requiring impossible simultaneous external progress at capacity 1.

**Required negative control:** charge the sole slot to waiting P before Q's leaf becomes eligible. The unchanged progress monitor must reject the deadlock. Creating Q is not evidence of transport-requiring descendant completion.

### R6 — collections, cursor commits and compatibility

Freeze snapshot capture, cursor advancement, append overflow, invalid indexing, nested depth and aggregate encoded-byte bounds. Copy/freeze public and retained list boundaries so caller mutation cannot change accepted work. A body appending elsewhere cannot mutate its iteration snapshot. Cursor/operation updates must preserve correlation across failure. Define structural equality and typed defaults without changing old scalar bytes.

Choose the artifact/binding/store version matrix before implementing semantic changes; the current “version 3 supported unchanged, or new functionality rejected” leaves this undecided. Replay must validate creation origins, root membership, conserved charges, result selection and binding authority against retained facts, not just replay child projections around unchecked new ledgers.

Retain one tiny old-model obstacle per semantic extension: a typed post-initialization variable list; finite iteration crossing an asynchronous boundary without authored numeric PCs; and another invocation of a one-shot template without provisioning a named graph. No second framework is necessary.

**Counterexample R6:** snapshot `[a,b]`; persist advancement past a; die before committing a's invocation intent; recovery loses a. Reverse the writes without atomic correlation and recovery duplicates a. The existing commit protocol supplies the mechanism, but the new lowering must preserve it explicitly.

## Parent's proposed refinements received during review

After preliminary findings, the parent proposed: provision an immutable root envelope for a declared root run before its single accepted work set; descendants inherit it; generation-1-only scoped members with Retry rejected; keep reply candidates separate and use a generic structured-join gate before publishing success; failed/cancelled members produce explicit failed/cancelled scope results; missing reply/terminal mismatch fails; preserve late evidence; use adapter slots only and distinguish capacity-1 local progress from capacity-2 external progress.

These address the central R1/R2/R5 concerns in direction. They were proposals in a message, not changes to the reviewed contract, so this initial verdict remains NEEDS_REVISION. The revision must still specify root accounting inclusion, authority/debit transaction, global/root capacity combination, all lifecycle closure cases, capability derivation/binding identity, application-failure reply preservation, and exact join/result arbitration. A runtime structured join must enforce generic completion facts without deciding an application's evidence or aggregation policy.

## Acceptance and limitations

The draft intends coverage of all A–J. Independent decisions require the refinements above. Distinguish transport admission from accepted-work consumption in crash evidence. Duplicate IDs and excessive item count reject before child work; shared-budget exhaustion during iteration needs declared partial-result behavior. Test admission/closure and authorization/attempt cuts in addition to accepted work, partial completion and recorded-but-unconsumed results.

Replay must use no registry/credentials, leave evidence unchanged, and validate new ledgers. Retain faulty origin, root-budget and correlation controls caught by independent monitors. Monitors should compute expected ordered tree results and budget totals from inputs/retained records rather than compare the reducer only with itself.

No implementation correctness, exhaustive race coverage, fairness, provider-side cancellation, remote concurrency bound, arbitrary recursion/nesting equivalence, malicious-adapter safety or performance is established. Ordinary programs still need valid termination and aggregation policy. R1–R6 are sufficient bounded refinements to proceed toward the requested cases without a privileged role, second scheduler, ownership transfer, or business exactly-once claim.
