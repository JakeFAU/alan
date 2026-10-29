ALAN Phase 1J Codex Brief
Retained joins and scope lifetime
Work in JakeFAU/alan.

The approved research question is:
Can an ordinary retained join discipline preserve the existing completion guarantee by keeping the root open until its required parallel work has settled?

Execute this bounded research increment through implementation or a conclusive negative result, qualification, manuscript reconciliation, independent review, and a draft PR. Returning only a plan is insufficient. Make routine choices autonomously and record their reasons. Use independent agents for the model/oracle and substantive review where available; resolve ordinary review findings without another user approval round.

Do not merge or release. No paid provider calls or messages to people are authorized. Respect repository permissions and required gates. Complete useful independent work before reporting genuinely missing access or a material scope expansion.
1. Start from the actual negative result
Phase 1I was merged in PR #18, merge commit 328e4be02954a75fd1e408d4c6d567dfff0a8e3b. Verify current main, ancestry, intervening changes, and working-tree state. Read applicable AGENTS.md instructions. Work in an isolated branch/worktree and preserve unrelated work. Record the actual baseline commit and tree.

Read selectively:
- docs/foundational/phase1i/00-plan.md through 05-decisions.md, especially the frozen contract, final qualification, and resolved review findings.
- docs/foundational/phase1i/evidence/model/root-success-parallel-counterexample-K1.json and the corresponding A/B traces.
- docs/foundational/phase1i/evidence/runtime/fixtures-supplement-root-fence.json, evidence/runtime/review-fix/final-report-v2.md, and evidence/reviews/final-review.md.
- examples/substitution/deployment.go, the frozen a/, b/, and protocol/ artifacts, research/substitution/, internal/substitutiontest/, and the separate public consumer module.
- The actual exported authoring/messaging interfaces, retained invocation-outcome handling, and internal/executor/invocation.go, particularly compositionScopeClosed.
- Relevant claims in paper/alan-dissertation.md and its LaTeX counterpart.
Verify paths and symbols against the current source.

Phase 1I found a shared context-lifetime failure. In the parallel context, the candidate can answer correctly, the root can consume that answer and execute local Success, and the static peer can subsequently have its invocation of S refused with scope_closed. The aggregate result can then be failed/member_failed. No external cancellation is needed.

All eight initially completion-entitled parallel configurations have a failing modeled schedule for K1, A, and B. Concrete delayed-peer witnesses realize the failure for both implementations. This falsifies the context's completion premise; it does not distinguish A from B or refute a substitution implication whose premise requires that the reference context already guarantees completion. Nor do unequal-charge candidates refute the manuscript's stronger exact-debit relation.

Preserve K1, its original entitlement classification, the old contexts, and their evidence. Green tests that recognize this historical failure remain useful. Do not reinterpret them as evidence that completion was repaired.
2. Change the context protocol, with an explicit causal comparison
Name the original context C0 and the new context C_join. Introduce a separately identified join contract, such as J1; do not retroactively edit K1.

For the primary comparison, preserve:
- Frozen A and B source and compiled artifact identities, their input/output behavior, and optional-helper structure.
- The original S workload/artifact.
- The delayed peer's original 32 ordinary Set instructions, S invocation, and wait on S's original retained handle.
- Root and peer provisioning, resource limits, candidate invocation/consumption path, and production runtime behavior.
The permitted intervention is the root's retained join wait and final decision, the peer's outcome-reporting suffix, and the minimum public ports/routes/grants and schema fields needed to connect them. The peer's old immediate terminal decision may be replaced by reporting followed by termination. Root and peer artifacts necessarily change: give them new identities, preserve the originals, and record exact source/compiled diffs and hashes.

Inspect existing public Send/`Receive` facilities or their actual equivalents. Use ordinary retained programs. Do not introduce a Join API, automatic scope lease, dynamic participant registry, privileged coordinator, new scheduler, or database mutation. Replacing the static peer with an invoked child changes topology and accounting; it cannot substitute for this comparison.

Keep executor/compiler/schema semantics, dependencies, closure policy, and accounting unchanged. If existing public APIs cannot express the required protocol, preserve the smallest concrete obstruction and finish the analysis. Do not silently repair the runtime to make the hypothesis pass.
3. Freeze the join protocol before implementation
The peer is predeclared. Its obligation exists before the root can terminate, including while S has not yet been invoked. Counting only already-admitted children would miss the original failure.

Use a strict join for this experiment:
1. The root invokes the candidate and retains/consumes its actual outcome.
2. The root retains that result and waits for the designated peer's correlated report before normal termination. A report arriving earlier must remain available through the ordinary retained messaging mechanism.
3. The peer consumes S's actual outcome through the original invocation handle. It then reports the observed status, reason, response presence, and required response/correlation fields, with no subsequent scope-dependent admission or reservation work.
4. After consuming both branch outcomes, the root succeeds only if both meet their respective success conditions. The candidate must preserve the application id and return 1 - value; use S's actual existing contract for its result.
5. Known failure, refusal, or disallowed response absence is reported and retained. The root joins the other required branch and then fails. A pending branch remains pending, even if its sibling has failed.
6. Authorized external cancellation is a separate interruption of this discipline. Preserve its actual outcomes and charges; it is outside the positive completion premise.
A success-only report would create an accidental wait on failures and is unacceptable. Do not convert unknown or pending evidence into known failure or permission to repeat an operation.

Specify correlation using what the public API actually exposes: the designated root and peer, generation, obligation/port, and application request identity. Explain which checks come from runtime wiring and which are application checks. A stale or wrong-request report must not discharge the intended obligation. Duplicate delivery or replay must not create a second completion or new work.

An authenticated report is a peer assertion, not automatically a runtime receipt proving S completed. The root need not inspect private S state. The independent oracle must establish that the peer consumed the original S outcome before reporting and that the report matches that consumed evidence. A premature report may fool the trusting root while still being rejected as a peer-contract violation.

Sending the report may precede the peer's own terminal instruction. Define its meaning as discharge of the required S work and future protected demand; separately verify the peer's eventual terminal state and the aggregate result. Do not label message commitment as peer termination.
4. Demonstrate lifetime preservation instead of assuming it
State and check both obligations:
- Resource: at each relevant prefix, spent resources plus outstanding contractual demand do not exceed the configured limit; equivalently, outstanding demand fits the residual allowance. Use the existing charging rules for each resource.
- Lifetime: while a required participant retains future work that depends on this scope's admission/reservation authority, the root does not voluntarily execute a terminal action that fences it.
Treat the second as a property of C_join to establish. Do not make it an environmental assumption, remove normal root Success/`Failure` from the model, or define admissibility as eventual success. Ordinary root termination remains an available runtime transition; the program must prevent it at the wrong point.

Explicitly distinguish local run status, aggregate outcome, explicit lifecycle cancellation, and the actual admission-open predicate. Phase 1I's normalized Closed=false does not imply that admissions remain possible.

The decisive positive witness must contain a retained intermediate state with the candidate outcome consumed, the peer still requiring S admission, and the root open at its mandatory receive. Then show S admission, original-outcome consumption, report commitment/delivery/consumption, and the final states. A successful run in which the peer happened to finish first does not demonstrate the intended intervention.
5. Keep the independent model small
Freeze the original eight positive configurations:
- Application id in {1,2}; value in {0,1}.
- Candidate mode answer.
- Invocation and dynamic-created-run limits I=R=3.
- Depth cap in {2,3}; effect allowance zero.
- The same static root/peer provisioning and declared service assumptions.
First check C_join[K1], retaining the reference contract's permitted local/helper choices. Then check C_join[A] and C_join[B] through independently justified artifact-to-model mappings. Reuse reviewed primitives where appropriate, but do not derive the reference oracle from the implementations' outputs.

Enumerate all modeled schedules within these explicitly named products. Include commitment, admission/refusal, publication, consumption, report delivery, normal termination, and freeze boundaries at the distinctions needed by the runtime. Preserve actual consumed payloads and identities. Model the report arriving before or after the candidate result.

State eventual-service assumptions and absence of external cancellation. A finite search cutoff cannot establish progress: provide a valid rank or suitable cycle/fairness analysis for the claim actually made. Legitimate pending controls are outside the positive completion profile.

Report exact initial-state, reachable-state, transition, and maximal-execution/cycle counts as appropriate. Distinguish all-schedule model results from concrete runtime samples. These eight configurations are a finite scope, not arbitrary admissible compositions or a statistical failure rate.
6. Run a focused distinguishing matrix
Use the original delayed-peer fixture and unchanged Store.Run for the decisive concrete comparison, with physical capacity one. Preserve its original operational limits where sufficient; if reporting requires more execution steps, document that overhead without changing semantic resource allowances. Do not force order through sleeps, chosen identities, authoritative-state edits, or scheduler changes.

Use one payload for concrete controls unless another exposes a distinct issue. Do not multiply every control across the entire model domain.
Case	Required evidence
Original C0, A/B, I=R=3	Reproduce the retained root-success/peer-`scope_closed` counterexample; distinguish local success from aggregate failure.
Original C0, larger I=R=4	Show that additional resource capacity does not remove the same lifetime failure.
C_join, A/B, I=R=3	Test completion, including the decisive intermediate state, actual report consumption, and final peer/aggregate outcomes.
C_join, insufficient capacity	Separately expose invocation and run limits, preferably I2/R3 and I3/R2. B cannot guarantee all three required admissions/runs; retain actual refusal and report known failure. Joining must not fabricate capacity.
Premature-report peer mutant	Use a valid ordinary program that reports before consuming S's required outcome. The unchanged guarantee checker must reject the causal/reporting violation; do not add root-side privileged inspection.
Correlation control	A schema-valid wrong-request report must not count as the expected join. Distinguish root rejection, runtime refusal, and oracle rejection.
Known failure/absence	Exercise a known candidate failure and a known S failure; cover response absence where the existing profile supports it. Confirm reports propagate actual outcomes and strict joining does not wait for success-only messages.
Pending and explicit cancellation	A pure pending branch yields no false completion. Separately exercise a supported authorized cancellation and preserve the actual interrupted outcome.


Failure/pending S variants are separately identified control artifacts, not the unchanged S used in the primary pair. A bounded pending observation is not a proof of eventual nontermination.

Expected successful dynamic accounting is A: candidate + S = 2 invocations and 2 created runs; B: candidate + helper + S = 3 and 3. Fixed provisioning is root + static peer = 2 runs, reported separately under existing accounting. Effects remain zero. The report adds ordinary communication, not another invocation or external effect. Verify these expectations; record actual, possibly smaller charges for refusals/failures.

Retain one causal explanation for each control. If a concrete schedule differs from the prediction, investigate and preserve it instead of tuning the fixture until it passes.
7. Qualify retained joining across real interruption
Use the existing process facilities for four focused pure cases: A/B crossed with two boundaries.
- Candidate outcome consumed, peer report not committed, preferably before S admission.
- Peer report committed or delivered, root has not yet consumed it.
Establish the cut from retained evidence, acknowledge the checkpoint through the test facility, actually terminate the process, and open the original store in a fresh process. Preserve original run/generation/artifact identities, invocation handles, outstanding join, report identity if already committed, and spent charges. Verify root lifetime and eventual outcomes from actual resumed records.

Do not recreate the root, resubmit work, manufacture a replacement report, or let the harness author protocol decisions. Inspection may observe retained state; it must not mutate authoritative state. If an exact cut is unsupported, identify the demonstrated boundary and limitation rather than claiming the intended cut occurred.

This phase is pure. Preserve Phase 1I's effect/unknown evidence, but do not start effectful substitution, external retransmission, reconciliation, or unknown-repair work.
8. Treat the evidence adapter as part of the apparatus
Extend the reviewed raw-record consistency checks only where joining introduces new observations. The oracle must connect:
- Original admission/operation identity to the exact outcome consumed by each waiter.
- The peer's consumed S outcome to the report payload and its actual ordering.
- The exact report consumed by the root to the final decision.
- Root/peer status, freeze/closure reasons, correlation, and resource charges to retained records.
Do not substitute published/final data for what a program actually consumed. Do not erase identities independently or collapse local terminal status into aggregate status.

Add focused adversarial checks for the changed decoder/report mapping, including early reporting and wrong correlation. Preserve rejection reasons and distinguish an invalid peer contract from a conforming-context runtime failure. Reusing evidence requires unchanged source/artifact identity and an explicit explanation; unchanged A/B need no new independent-authorship campaign.
9. Execute in reviewed stages and leave a reproducible record
First freeze and independently review the protocol, bounds, oracle, and permitted intervention. Write a short implementation plan with actual inspected paths and commands. Then separate context implementation from independent model/oracle work where possible. Run the matrix and process witnesses, reconcile discrepancies, and obtain a final independent review of the resulting source and claims.

Run relevant model, public-consumer, integration/process, and focused race checks, plus repository-required gates. Use the repository's pinned toolchain and record commands, environment, exit status, baseline/final source identities, and evidence hashes. Broaden testing only for an actual remaining risk or required gate.

Create:
- docs/foundational/phase1j/00-plan.md.
- 01-join-contract.md, including outcome policy, lifetime obligation, assumptions, bounds, and raw observation mapping.
- 02-qualification.md, separating established results, falsified hypotheses, and open claims.
- 03-review.md, with findings, fixes, and final disposition.
- 04-runbook.md, with reproduction commands.
- 05-decisions.md, including unchanged identities, allowed changes, exclusions, and the next unresolved boundary.
- Compact machine-readable model/runtime traces and an evidence index identifying each case, contract, artifacts, limits, actual cut, commands, and result.
Keep source and evidence additions narrow and follow repository conventions. Update relevant manuscript passages consistently in Markdown and LaTeX, with source references. Do not restore general confluence, normal-form, contextual equivalence, or Theorem 4.8 claims on the strength of this experiment.
10. Completion and stopping rules
A supported bounded result requires the original negative to remain reproducible; C_join[K1], C_join[A], and C_join[B] to satisfy the declared guarantees across the frozen positive model scope; concrete witnesses to demonstrate the intended lifetime difference; adverse controls to expose contract violations; and the specified recovery cases to preserve the retained join.

A new reproducible counterexample is also a completed research result when its failed premise and causal boundary are identified. An API obstruction or unachieved execution boundary must be reported as such, with completed evidence preserved; it is not a successful qualification.

Do not select the answer in advance, increase budgets to conceal failure, change runtime closure semantics, or expand into a universal congruence proof. A positive result would support this join discipline as sufficient within the tested/modelled scope. It would not establish universal necessity, minimality, arbitrary-context substitution, or general effectful recovery.

Finish with a reviewed draft PR that states the hypothesis, exact intervention, result, evidence, limitations, and next unresolved question. Report its URL, baseline/final commits, substantive changes, validation, and remaining findings. Do not merge.

The purpose is to turn Phase 1I's failure into a precise causal experiment: determine what retaining the root until required peer work settles actually buys, and preserve whatever boundary the experiment reveals.