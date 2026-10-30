The approved question is:
Does bounded K1 substitution remain valid when the consuming program changes the order and interpretation of original outcomes, while preserving ownership, required joins, and worst-case resource entitlement?

Complete two ordered milestones: an S-first parent preserving Phase 1L's positive guarantee, then a parent that distinguishes component failure from a missing required reply. Carry the work through independent review, scoped manuscript reconciliation, and a draft PR. Make routine implementation and verification decisions autonomously. This is an execution assignment; do not return only a plan. Do not merge, release, make paid provider calls, or send messages to people.

A negative result is valuable when it identifies the failed premise or transition. There is no instruction to manufacture a failure, and no existing evidence that these extensions break substitution.
1. Establish the baseline and reuse the evidence
Read current AGENTS.md and applicable repository instructions. Verify current main, ancestry, and unrelated work before selecting an isolated branch/worktree. A verified clean existing worktree is acceptable.

PR23 is merged at f5b52853ce3cf2b93563eed5eb8268234386e691; its reviewed source is 8310bb270eed536ba717ea0e9f2a35f23ae6e68e. Verify the actual current source; do not assume that merge remains the latest main. Preserve historical I/J/K/L artifacts, evidence, and source labels, including PR19 TeX formatting and PR20 mathematics.

Read narrowly:
- docs/foundational/phase1l/01-contract.md, 02-qualification.md, 03-review.md, 04-runbook.md, and 05-decisions.md.
- Phase 1I's K1 contract and applicable local A/B refinement evidence.
- examples/invocationjoin/deployment.go and its external consumer.
- Original A/B/helper/S templates and public accessors in examples/substitution/.
- research/substitution/invocation_join*.go and the relevant reusable component/monitor primitives.
- The Phase 1L raw-record adapter, focused tests, and capture/audit facilities in internal/substitutiontest and its evidence directory.
Phase 1L qualified four answer inputs with I=R3, depth2, E0, fixed1, capacity1. All 16 maxima across its K1/A/B products satisfied G. Its concrete child-completion-order samples still consumed candidate then S. Fail, absent, and pending were controls outside its positive census. Those findings are the baseline, not results for the new contexts.

Primary sources: PR23, reviewed qualification, K1 contract.
2. Keep the extension small and explicit
Use these names for the behavioral specifications; repository API names may follow existing conventions:
Context	Commit order	Consumption order	Decision policy
C_es: historical Phase 1L	Candidate e, then original S	e, then s	Existing guarded strict join
C_se: milestone 1	Same	s, then e	Same guarded strict join
C_reason: milestone 2	Same	s, then e	Explicit outcome classification after both joins


These are alternative parents in separate executions. There is one owning caller per invocation, one fixed parent, and no additional concurrently running parent.

The new consuming definition should have a distinct author from the independent model/oracle. Author it from the behavioral contract; record what was reused and the actual scope of independence. Do not claim independent authorship merely from renaming the existing parent. Reuse public APIs, unchanged templates, and reviewed infrastructure.

For each new context, one builder is parameterized by the candidate binding. A/B receive the same policy, input, guards, limits, and operational assumptions. Binding/catalogue metadata and compiled identities may differ; document the semantic and compiled differences rather than claiming byte-identical parent artifacts.

Suggested implementation locations are an additive examples/consumingpolicy/ package with a separate public consumer, additive research/substitution/consumer_policy*.go files, focused internal/substitutiontest tests, and compact docs/foundational/phase1m/ evidence. Adjust names after inspecting repository conventions. Preserve the historical builders and named results; do not refactor unrelated infrastructure.

Both original invocation intents must commit before the first await. Admissions and execution may interleave. S receives the original request id/value and returns its original id/zero result; it does not receive the candidate's complement. Retain both original handles.

Every normal decision, including an error decision, follows both actual outcome consumptions. In particular, S-first does not authorize early termination after an S failure. Required pending work keeps the parent open. External cancellation remains outside the positive and classified-outcome profiles.

Keep the original A/B/helper/S artifacts unchanged. No static reporting peer, ordinary report/ACK channel, enclosing wrapper, pending-release input, retry, fallback invocation, effect, new runtime primitive, or generic context generator is part of this task.
3. Milestone 1: reverse consumption order and preserve G
Construct C_se with the same strict result policy as Phase 1L. The only intended behavioral change is awaiting S before the candidate. Preserve status/presence guards before payload reads and the exact correlation/complement/S-zero requirements.

First compile and run a small external consumer for both A and B on id1/value0/answer using public imports and the unchanged automatic runner. Record expected outcomes and charges from the specification before interpreting these preflight executions. Do not define the reference from observed outputs.

Qualify the original positive guarantee, named G_answer here:
- Both original invocations publish; their original outcomes are delivered, selected, and actually consumed exactly once by their owning parent.
- Both outcomes are succeeded and present, with the correct candidate id/complement and original S id/zero.
- Parent and root/aggregate publication succeed.
- Original identities, obligations, and resource limits are respected.
The domain and assumptions remain:
- id in {1,2}, value in {0,1}, mode answer.
- Common I=R3, depth cap2, E0; one fixed parent.
- Stable generations/revisions/grants, valid template/reply/schema bindings, no retirement or cancellation, and eventual eligible local/protocol service.
- Physical capacity1 for concrete samples; justify finite instruction/dispatch/byte limits against the new compiled artifacts rather than inheriting measured limits as already qualified.
Expected successful dynamic spending remains A+S2 and B+helper+S3. Both receive the same worst-case allowance. Exact debit equality is not claimed.

There are four configurations per product, hence 12 new positive product configurations across K1/A/B. Check C_se[K1] first, including both K1 local/helper branches, then C_se[A] and C_se[B]. Do not prescribe state, edge, maximal-state, or test-leaf counts.

Expose authentic concrete executions with both relative orders of the original candidate and S invocation outcome publications. Record member-local termination, delivery, and parent consumption separately; helper completion is not candidate completion. In at least one execution, S's original outcome has been consumed while the candidate's original outcome remains required and unconsumed, and the parent stays open. Record whether the candidate outcome is unproduced, published, or retained but unconsumed. Finish the original work normally. Use supported public transitions or existing observation facilities; no authoritative-state edits, host decisions, or sleeps standing in for causality.

Retain focused dual pending controls: candidate pending lets C_se consume completed S before waiting for e; S pending leaves the parent at its first await even if the candidate has completed. Do not supply the forbidden second request that would release candidate pending.
4. Milestone 2: interpret failure and absence explicitly
C_reason keeps C_se's commit/consumption order and both required joins. After both consumptions, its authored program classifies the actual retained outcomes. The requested mode may predict expected behavior in the independent oracle; it must not substitute for inspecting the observations inside the parent.

Validate S's successful, present, correctly correlated original id/zero outcome before accepting any candidate classification. For this milestone, require the authentic successful reason member_succeeded as well. Validation happens after both joins.

Use the following policy and retained classification tags:
Consumed candidate evidence, with valid S	Classification	Parent terminal
succeeded/member_succeeded, present, original id and correct complement	answer	Success
failed/member_failed, response absent	candidate_failed	Failure
failed/missing_reply, response absent	candidate_missing_reply	Failure
Any other completed candidate evidence, or invalid S evidence	invalid_outcome	Failure


Guard all possibly absent payload reads. A missing reply is a protocol failure, not a successful empty answer. Absence and ordinary failure both have status failed; response presence alone cannot distinguish them. Preserve the exact original reason alongside the classification.

Keep child and root reasons distinct. In the absent case, the candidate's original reason is missing_reply, while this parent deliberately executes Failure and its own root/aggregate failure reason is member_failed. The distinction survives in the original candidate envelope and committed classification; do not require or invent propagation of missing_reply into the root reason.

A neutral initialization is allowed if the authoring API needs one. It is not a completed classification. No answer/error classification or terminal decision occurs before both required observations are consumed. Pending has no published outcome and no completed classification.

Observe the classification through the same public retained-state/root-inspection boundary as Phase 1L. It must be set by committed program instructions, not computed by a host decoder. Do not introduce a final ordinary Send at root termination.

Define and independently check G_outcome:
- For answer, the parent takes the answer branch and satisfies G_answer.
- For fail, both original outcomes are consumed, the candidate's failed/member_failed envelope is retained, classification is candidate_failed, and parent/root/aggregate finish with the corresponding failure.
- For absent, both are consumed, failed/missing_reply remains distinct, classification is candidate_missing_reply, and parent/root/aggregate finish with the corresponding failure.
- For pending candidate, under eventual service S completes and is consumed; the parent remains open at the original candidate await, with no invented candidate publication, completed classification, or terminal result.
- Identity, provenance, consumption, resource, and payload-safety obligations hold throughout.
This is correct policy execution across outcomes, not successful completion in every mode. Failure/absence configurations are not counterexamples to Phase 1L's answer-only G.

For the finite milestone-2 census, keep S's original answer template and use id{1,2} × value{0,1} × candidate mode{answer,fail,absent,pending}, at the same I=R3/depth2/E0/fixed1. This is 16 configurations per product, 48 across K1/A/B. Declare the domain before results; do not add a full S-mode, capacity, depth, or cancellation cross-product.

The expected conforming labels are answer, candidate_failed, and candidate_missing_reply, plus the specified pending state. invalid_outcome is a defensive branch/control, not another permitted K1 answer. A prefix with a faulty component may be truthful without satisfying component conformance or the profile guarantee.
5. Independent reference, conformance, and correspondence
Keep these obligations separate for each context and guarantee:
X refines K1 under the applicable environment
C[K1] satisfies the selected guarantee
C[X] satisfies that same guarantee, for X = A and B
The reference retains K1's independent local/helper choice in every declared mode. A property must hold for all permitted reference executions; one favorable reference path is insufficient.

Reuse the reviewed local relation and pure primitives with exact source identities. Requalify the relevant local A/B subset for all modes introduced in milestone 2; Phase 1L's current answer-only subset does not by itself certify those additional configurations. Explain why the original S debit/environment projection still applies. No full historical census rerun is required solely to restate earlier evidence.

That local requalification includes progress under the declared entitlement/service assumptions: answer reaches its correct present outcome, fail and absent reach their distinct specified envelopes, and pending reaches its designated unmatched receive. Acceptance of finite safety traces alone does not establish these obligations.

Freeze the behavioral contract, bounds, and independently derived reference before qualification. Construction preflight may precede the reference implementation, but keep its execution outputs out of the model author's initial derivation. The oracle follows K1 and this specification. After freezing the reference, inspect the actual new artifacts for correspondence. Review the reference before using it to accept replacements.

Extend the finite representation and monitors only as needed for reversed awaits and classification. Preserve actual invocation commitment, authorization/admission, reply, publication/delivery, selected consumption, classification, terminal, and root publication distinctions. Use binding-qualified reply exceptions only where their actual conditions hold.

Map all new compiled instructions, including classification setters and branches, to model events or justified pure stutters. Do not treat a retained tag as proof that the associated branch executed. Prove/check the finite rank or appropriate cycle/fairness obligations, including intended pending maxima. Explore without failure pruning or a search cutoff presented as completion.

Report G_answer and G_outcome separately. Label model enumeration, automatic samples, supported public-driver schedules, and any recovery evidence distinctly.
6. Focused evidence and controls
Reuse the Phase 1L adapter and its independent audit. Before normalization, link each observed handle to its original commitment, admission, request/binding, outcome publication, retained observation, and actual selected history/consumed sequence. Never construct handles from expected sequence tuples or replace selected evidence with a later publication.

For C_reason, also link the actual classification write and terminal decision to those consumed envelopes. The independent expected-policy calculation may use the requested mode; actual classification evidence must come from the program.

Include:
- All eight automatic positive A/B executions per new context, across the four answer inputs.
- Representative genuine A/B fail, absent, and pending executions for C_reason, with exact reasons and branch evidence; the model supplies the declared full finite domain.
- The milestone-1 relative completion-order and intermediate-state witnesses, plus the dual pending controls.
- The existing executable wrong-id control where reusable: retain its honest consumed reply, require the new parent's correct defensive behavior, and distinguish raw/prefix validity from component/G rejection.
- A focused negative that detects confusion of member_failed and missing_reply or a wrong classification. Prefer, if existing fixtures make it straightforward, an absent request to a separately named control component that deliberately fails: its authentic member_failed must produce candidate_failed, while component/profile checks reject the departure from K1's requested behavior. This can replace a weaker classification negative and directly checks observation-based rather than request-mode-based classification. Label executable mutations, detached decoder corruptions, and pure-oracle negatives accurately; do not build a new fixture framework for this control.
- Relevant existing scarcity controls if they protect reused code. Do not spend the phase reproducing a known unequal-budget distinction.
Use the existing process-test apparatus only if it directly answers a new persistence risk or is required by repository gates. A new broad crash matrix is not required. If adding the opposite one-consumed checkpoint, acknowledge the exact predicate, use genuine SIGKILL/fresh-process Open of the same database, and preserve original identities/handles/charges and once-only consumption. Do not call back into Store under its held boundary lock. Historical recovery evidence keeps its original scope and source identity.

Do not duplicate all earlier decoder mutations or evidence archives. Add checks for new ordering/classification risks; rerun affected shared checks where reuse or edits create a concrete risk.
7. Keep repairs and conclusions disciplined
No production/compiler/runtime change is expected. Correct ordinary construction mistakes rather than declaring another research boundary prematurely.

If a demonstrated author/compiler defect blocks the approved construction, a minimal correction consistent with existing semantics is allowed after independent scrutiny, with a regression and affected requalification. Preserve the exact original obstruction and any artifact/version impact. Do not weaken identity, authorization, resource, observation, or guarantee conditions. A genuinely new semantic requirement is a separately reported scope boundary.

Classify any negative precisely:
- C[K1] fails: reference/context-premise failure.
- A/B do not refine K1 under the stated profile: component-conformance failure.
- Reference passes and a conforming replacement fails: bounded substitution counterexample.
- Authentic runtime and mapped model disagree: correspondence failure.
- The consumer separates behaviors K1 permits: a stronger observation/equivalence claim is unsupported.
- A construction remains blocked after reasonable corrections: the smallest exact interface/authoring obstruction.
Minimize a substantive failing context/trace within this declared family without erasing causal events or changing entitlement. Do not claim a globally smallest counterexample over arbitrary contexts. Do not silently redefine G or exclude a failing schedule after seeing it.

If milestone 1 exposes an unresolved substantive problem, prioritize its diagnosis and reduced witness before expanding milestone 2. A classified result with the reason milestone 2 is blocked is acceptable; an avoidable implementation error is not a stopping rule.
8. Verification, review, manuscript, and delivery
Run new public-consumer builds/executions, model/oracle and focused runtime/decoder checks normally and with race instrumentation, relevant unchanged baseline gates, and repository-required formatting/static/module checks. Use the actual pinned toolchain and offline/readonly dependency configuration. Inspect test discovery and record exact argv, environment, selected leaves, exits, and source hashes. Derive selectors from implemented names; do not invent passing commands.

Broaden testing only for an identified risk or required gate. If a broad watchdog expires, retain that failed command and enumerate unfinished tests before targeted follow-up. Combined coverage must not be described as a single green command. Results remain bound to the sources actually executed.

Obtain independent review of the reference premise, new consumer behavior, all-mode conformance reuse, artifact mapping, selected observations/classification, resource assumptions, and final claims. Resolve actionable findings and rerun affected checks. Review agreement is not a proof.

Keep phase documentation compact: approved contract, qualification/limits, exact runbook and evidence index, decisive traces, and review disposition. Reuse historical evidence by reference. Maintain a clear delta from Phase 1L; do not copy its archive or create redundant narratives.

Reconcile the relevant Markdown and LaTeX discussion after results settle. TeX remains the manuscript truth. Preserve inherited formatting, mathematics, and theorem status. Run Tectonic once at phase end with the accepted warning policy; inspect changed PDF pages. Do not regenerate the PDF per implementation commit.

Finish with a reviewed draft PR and a concise report covering:
- The result for C_se under G_answer.
- The result for C_reason under G_outcome, or its exact unresolved blocker.
- Actual bounded scope, configurations, counts, representative runtime evidence, and source identities.
- Any preserved counterexample and its classification.
- What changed, validation actually executed, and the next unresolved question.
General congruence, arbitrary contexts, exact-debit equivalence, universal compiler correctness, effectful/distributed recovery, deeper nesting, and invocation-result adapters remain open unless separately authorized later.

The intended outcome is a small, executable extension from one consuming parent to an explicit family of policies, with each claim tied to the guarantee and evidence that actually supports it.