ALAN Phase 1N — Callable Join Contract Implementation Plan
For agentic workers: Execute this brief task by task, using subagent-driven development or plan execution skills when available. Keep construction and reference authorship distinct, and obtain independent review at the semantic gates below.

Goal: Determine whether the qualified owning join can itself satisfy a public component contract, and establish the smallest justified rule that lifts K1 substitution through that fixed constructor.

Architecture: A fixed outer caller O invokes one callable join J[X]. J invokes the unchanged candidate X and original S, consumes S then X through their original outcomes, and returns a typed diagnostic before its own Success or Failure. O makes its decision from J's original public invocation outcome.

Tech stack: Existing ALAN Go authoring/runtime APIs, finite research models, public consumer modules, and retained-record qualification tools. Use the repository's actual pinned toolchain and dependencies.

Spec: The approved behavioral contract and scope in this brief, together with the pinned Phase 1M sources below. This is a research execution assignment, not a request to return another plan.

Work in JakeFAU/alan. Complete implementation, qualification, independent review, scoped manuscript reconciliation, and a draft PR. Make routine decisions autonomously. Do not merge, release, make paid provider calls, or send messages to people.

A positive result must explain why the constructor preserves the contract. A negative result must identify the failed premise or exact transition. Another passing census alone does not establish the requested lifting rule.
1. Baseline, scope, and file ownership
PR24 is merged at 3c5118ea9763a27cca432ca91aac667bd59e21e7; its reviewed head is d59595d7081139a149d404d51ba97b46925f6d15. Verify current main and ancestry rather than assuming this remains the latest commit. Read applicable AGENTS.md instructions, inspect unrelated work, and select an isolated branch/worktree; a verified clean existing worktree is acceptable.

Read narrowly:
- docs/foundational/phase1m/01-reason-contract.md, 02-qualification.md, 03-review.md, 04-runbook.md, and 05-decisions.md.
- The original K1 contract and relevant local relation/progress evidence from Phases 1I, 1L, and 1M.
- examples/consumingpolicy/reason.go, the unchanged A/B/helper/S public templates, and existing external consumers.
- author/invocations.go, compose/collection.go, and the reply/freezing/publication paths needed for this construction.
- Relevant research/substitution/consumer_reason*.go, local relation/progress machinery, and Phase 1M raw-record authentication and audit facilities.
Phase 1M established bounded C_reason preservation over 16 configurations per K1/A/B product, including specified pending behavior. Its local qualification applies to its recorded environment. It did not establish a public callable join contract or the new constructor rule.
Global constraints
- Preserve original A/B/helper/S artifacts and the K1 contract. Keep prior contexts and historical evidence identifiable and unchanged.
- Preserve production compiler, executor, schema semantics, scheduler, accounting, and dependencies. Use existing public authoring/runtime behavior.
- Correct ordinary example, model, adapter, and test mistakes within this scope. If a genuine production/interface obstruction remains, preserve its smallest witness, complete unaffected work, and report the blocked obligation; do not redefine semantics to force a positive result.
- Preserve PR19 formatting, PR20 mathematics, and existing theorem qualifications.
- No outer sibling, extra forwarding wrapper, static peer, ordinary report/ACK protocol, inner-handle serialization, pending-release request, retry/fallback, new effect, or generic context generator.
- Current authorization is for a reviewed draft PR only.
Use an additive layout, adjusting ordinary file names to repository conventions:
Location	Responsibility
examples/callablejoin/	Public report schema, callable J constructor, fixed O/deployment, construction checks, separately named controls
examples/callablejoin/consumer/	Separate module using public imports to build and execute the complete deployment
research/substitution/callable_join*.go	Independent reference/monitor, environment alignment, lifting relation, finite exploration and checks
internal/substitutiontest/callable_join*.go	Genuine runtime schedules, explicit new raw mapping, decoder/control qualification
docs/foundational/phase1n/	Contract, argument, qualification/limits, runbook, decisions, reviews and compact evidence indexes
Existing manuscript Markdown and TeX	Scoped reconciliation after results settle


Reuse reviewed infrastructure without exporting or restructuring historical model internals merely for this task. Shared changes require focused compatibility checks.
Review focus
The five principal risks are: an absent diagnostic accepted as a valid candidate failure; a truthful source mistaken for a truthful report; an unproved budget/depth/environment shift; a classification or Send prefix mistaken for published completion; and a finite pending prefix mistaken for the required maximal pending state. The tasks below must exercise each.
2. Task 1 — Freeze the callable contract and its environment
- [ ] Record the exact baseline, protected source identities, new file responsibilities, and actual scope of independent authorship.
- [ ] Freeze the following topology, report semantics, resource profile, and guarantees before qualification.
- [ ] Independently review the contract and environment assumptions. Correct defects explicitly and retain version identities.
Topology and ownership
O is the sole fixed root at depth 0. It invokes one J[X] at depth 1 and retains that original invocation handle.

J receives the original request through a public port. It commits candidate e and original S in that order before its first await. Both receive the same original inputs used in Phase 1M; S receives id/value, not the candidate's complement. J consumes S then e through their original handles. Every normal classification, report, and terminal decision follows both consumptions, including error paths. B's unchanged helper is at depth 3.

J is a new callable artifact. The old C_reason root has inputs but no public entry/reply interface; do not relabel it as invocable. Do not insert a separate forwarding wrapper around it.

One J builder is parameterized by candidate binding. One O builder is parameterized by J's public template. A/B receive identical policy and allowances; account for the intended binding/catalogue/compiled-identity differences. O must not special-case A or B.
Request and diagnostic
The request remains {id, value, mode} with the original schema.

Use a bounded diagnostic with:
- id: original application request identity.
- classification: answer, candidate_failed, candidate_missing_reply, or invalid_outcome.
- answers: a bounded zero-or-one list of the original answer record {id, value}.
An answer requires exactly one correctly correlated complement record. All other tags require an empty list. A maximum list length of one does not establish that an answer exists. Use ordinary public schema/authoring primitives and guard all potentially absent reads. Freeze concrete bounds and representation after the public construction preflight, before model/artifact correspondence; representation adjustments must preserve these semantics.

No inner operation handle is needed in this report. O correlates the request through the application id and consumes its own original typed J handle. Do not reintroduce handle/string comparison.

J classifies actual selected inner outcomes using the Phase 1M policy. First validate S as succeeded/member_succeeded, response present, original id/zero. With valid S:
Actual consumed candidate evidence	J diagnostic	J local End and published envelope
succeeded/member_succeeded, present, correct id/complement	answer, one answer record	Success; succeeded/member_succeeded, diagnostic present
failed/member_failed, response absent	candidate_failed, empty answers	Failure; failed/member_failed, diagnostic present
failed/missing_reply, response absent	candidate_missing_reply, empty answers	Failure; failed/member_failed, diagnostic present
Other completed candidate evidence, or invalid S	invalid_outcome, empty answers	Failure; failed/member_failed, diagnostic present
Designated pending candidate	No diagnostic	No J End or invocation publication


J sends the diagnostic on its invocation-bound public reply before executing its terminal instruction. Error reporting does not turn failure into success.

The invalid_outcome tag defines defensive behavior. It is excluded from conforming K_join maxima under the declared K1/S profile. Its existence must not weaken the positive contract.

Distinguish three levels of absence:
1. Candidate absence: J consumes candidate failed/missing_reply and reports candidate_missing_reply; J itself publishes failed/member_failed with a present diagnostic.
2. J reaches Success without its required report: its invocation publishes failed/missing_reply with an absent response.
3. J reaches Failure without a report: its invocation remains failed/member_failed with an absent response.
The last two violate the completed-diagnostic requirement. Neither is a valid candidate_failed report.
K_join and the outer guarantee
Specify K_join independently of A/B execution outputs. Its obligations include truthful reporting of actual original inner consumptions, required joins, identity/payload/resource safety, the table above, and mode-specific progress under the declared environment.

Requested mode determines the conforming behavior predicted by the independent specification. It must not replace actual observation-based classification inside J or O.

J may forward request.mode to the candidate, but J's classifier and O's decision must not branch on it. Original id/value remain available for correlation and complement checks.

O awaits its original J outcome, guards response presence, and checks correlation, status/reason/tag consistency, and exact answer cardinality/content. It then records its actual decision in program state:
- answer with a valid succeeded envelope leads to Success.
- candidate_failed or candidate_missing_reply with a valid failed/member_failed envelope leads to Failure while retaining the distinct tag.
- A well-formed invalid_outcome report leads to the corresponding defensive Failure.
- Missing, inconsistent, or invalid J evidence leads to a distinct invalid_join_outcome Failure; it must not synthesize a candidate diagnosis.
O has no access to J's private child rows, inner registers, or inner handles. The harness may inspect those to qualify J's report truth. O's policy uses the public outcome and original input only, and its observed decision must come from committed instructions rather than a host reconstruction.

Define G_outer separately from K_join: O consumes the original published J outcome once and takes the matching guarded policy; under conforming K_join it preserves answer, the two distinct diagnosed failures, or the specified pending state, with correct root/aggregate publication and safety. Error modes are correct classified failures, not successful completion.

A schema-valid lie may cause O to execute its public policy faithfully. Detecting that lie belongs to J's semantic conformance obligation; authentication alone does not prove it truthful.
Resource and service profile
The common entitlement is I=R4, depth cap 3, E0, fixed1. Concrete samples use physical capacity1.
Construction	Total dynamic invocation/run charges	Greatest depth
O → J[A] with original S	3	2
O → J[B] with B's original helper and original S	4	3


These are derived expectations to verify, not reported new measurements. The one J admission leaves residual allowance3 for e/helper/S. The candidate's entry depth shifts from 1 to 2; its helper shifts from 2 to 3.

Assume stable generations/revisions/grants, valid template/request/reply/schema bindings, no cancellation or retirement, sufficient finite operational limits, and eventual eligible local/protocol service. O remains open through J consumption; J remains open through both child consumptions.

Only original S supplies the existing arbitrary-timing local-environment debit. J's admission is the prior fixed offset, not another concurrent debit. Prove this projection, including S admission before or after e admission. Permit K1's local/helper alternatives and unequal actual A/B costs under the common allowance. Exact-debit equivalence is not claimed.
3. Task 2 — Build the public construction and preflight the interface
- [ ] Implement the public schema, callable J, and fixed O using existing APIs. Keep J's classifier and O's public validator explicit and separately inspectable.
- [ ] Compile a separate consumer module using public imports. Execute A and B answer preflights and at least one genuine failed invocation carrying its present diagnostic.
- [ ] Verify original A/B/helper/S identities and derive the new artifact's instruction, dispatch, byte, depth, and charge bounds.
- [ ] Review the public construction before accepting its evidence as correspondence.
The source precedent is compose/collection.go: public Receive, original child invokes/awaits, public Send, then Success or Failure. prepareCompositionScope retains captured reply-admission obligations and can attach a single admitted response to a failed invocation. This is source support for feasibility, not evidence that this new program is correct.

The independent model author may work from the frozen behavioral specification and reviewed semantics while preflight proceeds. Do not give that author preflight outputs as the definition of expected behavior. Document reused source and shared compiler/runtime assumptions; two files alone do not establish independent authorship.

Keep reply commitment, first source authorization, admission, J End/freeze, J invocation publication/delivery, O selected consumption, and O End/root publication distinct. A successful Send commit or retained classification alone is not completion.
4. Task 3 — Establish the restricted lifting argument
- [ ] Derive and independently review J[K1] and the K_join monitor before replacement acceptance.
- [ ] Establish the resource/depth/environment alignment and requalify all-mode local A/B progress where the old result does not transfer directly.
- [ ] Write the explicit constructor relation and transition-case argument; mechanically check its finite instantiations.
- [ ] Explore the declared reference product first, then the A/B replacements, and report precisely which proof obligations were established.
Keep the logical obligations separate:
X refines K1 under the aligned candidate environment.
J[K1] satisfies K_join.
The fixed constructor lifts that relation:
    J[X] refines J[K1] under the declared join environment.
The fixed outer policy uses K_join to preserve G_outer.
State K_join and the constructor rule without referring to O's private PCs or registers. O is the verified realization of the required public caller behavior, not part of the definition of K_join.

The target rule ranges only over candidates in the declared semantics satisfying the required event refinement, bilateral environment readiness, resource conditions, and separate progress/nondivergence obligations. Local safety simulation alone is insufficient. Do not silently strengthen K1 or treat schema compatibility as this premise.

Give an explicit lifted relation covering:
- related candidate/reference states and the same original S state;
- matching J control and the declared public caller-environment and ownership obligations; matching O control is added only for the outer-policy corollary;
- original handles, retained outcomes, selected consumptions and correlation under consistent identity renaming;
- the prior J ledger offset, residual budget, depth shift, and the single S environment debit;
- matching diagnostics, reply obligations, terminal decisions and publications.
Check the cases for candidate steps, S/environment steps, J's public entry and inner joins, classification, reply commitment/authorization/admission, J termination/publication, and O consumption/termination. Identify pure stutters explicitly. Preserve refusal/accounting distinctions; do not erase them to obtain a relation.

In particular, the invocation-bound reply exception must cover the exact reply source, generation/revision, caller identity, route/schema, and binding assumptions. J can terminate after its Send and before first authorization; the rule must justify that schedule rather than exclude it. O's ownership supplies the required open destination under this profile.

Use an explicit finite transition argument and an independently checked invariant/relation. A new proof assistant or generic framework is not required. Distinguish a general constructor argument within these declared semantics from its A/B finite instances and from concrete artifact/runtime correspondence.

If only the finite instances can be established, report bounded nested preservation and the precise missing lifting step. Do not call enumeration itself a constructor theorem. If the argument fails substantively, preserve the smallest exact gap or counterexample instead of extending the context family.
Domain, maxima, and progress
Use id{1,2} × value{0,1} × mode{answer,fail,absent,pending}, with unchanged answer S: 16 configurations per K1/A/B product, 48 total. Retain K1's independent local/helper choice in every mode. Do not prescribe state, edge, maximum, or test-leaf counts.

A single full product may support both inner-contract and outer-policy predicates through justified projections; do not duplicate its counts as independent evidence. Any standalone abstract K_join checking must retain its declared resource/progress obligations.

For pending, establish all of the following under eventual service:
- The candidate consumed its original request, completed/consumed any chosen helper, and reached the designated unmatched receive.
- Original S completed and was actually consumed by J.
- J waits at the original e await, without a completed diagnosis, reply, End, or invocation publication.
- O waits at the original J await, without a completed decision, End, or root publication.
Do not send the forbidden release input or manufacture a Pending reply. Check genuine maximality and progress/rank/cycle obligations separately from acceptance of finite prefixes. Include monitor memory in state equivalence where needed, inspect every transition before deduplication, and report cutoffs rather than treating them as completed searches. Model rank is not a runtime instruction budget.
5. Task 4 — Qualify actual records and attack the new boundary
- [ ] Add an explicit new artifact/raw-record mapping; historical parent-PC mappings cannot certify J/O by renaming the root.
- [ ] Run the focused automatic and supported-driver samples below.
- [ ] Implement the genuine negative controls with separate identities and layer-specific expected verdicts.
- [ ] Independently audit original identities, actual consumptions, reports, decisions, bounds, and selected evidence.
Authenticate O's actual Invoke and emitted J handle; J admission and public request consumption; original e/S commitments, bindings and selected histories; J's actual classification and Send payload; reply authorization/admission and J End/publication; and O's original selected J observation, decision, End and root publication.

Do not construct handles from expected tuples, replace selected evidence with a later publication, or infer committed decisions from requested mode. An instruction mapping must account for every new branch/setter/Send/End, with justified stutters and any atomic groups recorded honestly.
Positive and pending samples
Run all eight A/B answer combinations across the four id/value inputs through the unchanged automatic runner. Add representative genuine A/B fail, absent, and pending executions. Verify the two layers of reasons, public diagnostic content, observed outer decision, charges and intended waits. The finite model supplies the full declared input census; runtime samples are not all-schedule coverage.

Add one supported late-reply schedule, preferably B/absent:
1. J consumes both original child outcomes.
2. J commits its candidate_missing_reply diagnostic Send.
3. J commits Failure.
4. First J reply authorization and admission occur after that End.
5. J's original invocation outcome publishes failed/member_failed with the diagnostic present.
6. O consumes that original outcome and takes the matching diagnosed-failure branch.
Retain causal authorization/admission evidence and the exact binding exception. If the supported API cannot express this cut, preserve that exact limitation instead of editing authoritative state or claiming the automatic run exercised it. No sleeps or host-computed answers may stand in for causality.
Genuine negative controls
Use separately named control programs. Raw authentication must recognize their actual identities and effects rather than reject them merely for differing from the positive artifact.
Control	Required distinction
Valid children, omit J report, then Success	Genuine J failed/missing_reply, response absent; O rejects as invalid_join_outcome; K_join fails
Valid children, omit J report, then Failure	Genuine J failed/member_failed, response absent; O still rejects; status alone cannot establish candidate_failed
Consume genuine candidate missing_reply and valid S, then falsely report candidate_failed	Report provenance/binding and O's actual policy execution pass; report-truth/K_join and the conforming profile fail
Reuse the Phase 1M deliberate-failure component under an absent request	J truthfully reports candidate_failed from actual evidence; policy validity passes while K1 mode conformance/profile fails


The false-report control is a semantic lie in a genuine execution, not a detached payload edit. It tests why a public report needs a constructor invariant. The mode-disagreement control prevents requested mode from masquerading as observation.

Add a small decoder attack set for the new original J handle/selected-observation boundary, report/branch evidence, and a Send/classification prefix falsely labeled as End/publication. Keep executable controls, detached corruptions, model negatives and construction errors separately labeled. Reuse applicable historical attacks instead of duplicating the archive.

No new S-mode, cancellation, scarcity, capacity, or crash cross-product is required. Add recovery work only for a concrete new persistence risk or an applicable repository gate. Historical recovery evidence retains its original predicate and source identity. If using existing hooks, respect held Store locks and preserve genuine acknowledged-process semantics.
6. Task 5 — Review, qualify, reconcile, and deliver
- [ ] Run new construction/public-consumer, model/relation/oracle, and focused runtime/decoder checks normally and with race instrumentation.
- [ ] Run affected unchanged baseline gates, formatting, static checks, and module verification. Inspect selected test discovery and exact source identities.
- [ ] Obtain independent reference/lifting, construction/runtime-mapping, and final whole-phase reviews; resolve actionable findings and rerun affected checks.
- [ ] Reconcile Markdown/TeX, build the PDF once at phase end, inspect changed pages, and prepare the reviewed draft PR.
Use the actual pinned local Go toolchain, offline/readonly dependency configuration and required CGO settings. Derive exact selectors from implemented tests; record complete argv, environment, selected leaves, exits and source hashes. Do not predeclare passing commands or counts.

Broaden testing only for a concrete risk or required gate. Preserve failed commands and identify unfinished tests before targeted follow-up. Combined coverage is not a single green full-suite run. Later source corrections do not inherit earlier runtime qualification without an explicit unchanged-source/affected-scope justification.

Keep evidence compact and reviewable: frozen contract, environment/lifting argument, source/artifact maps, model summaries, decisive real traces, controls, exact runbook, qualification/limits, and review disposition. Reference prior evidence instead of copying it. Preserve substantive rejected claims and counterexamples; ordinary construction mistakes need an honest work log, not an inflated research claim.

Classify negative results precisely:
- Reference J[K1] fails: contract/constructor premise failure.
- Local A/B premise fails after alignment: component/environment conformance failure.
- Reference and local premises pass but replacement fails: bounded substitution counterexample.
- Authentic mapped execution disagrees with the model: correspondence failure.
- Finite instances pass but the generic lifting step remains unsupported: bounded preservation with an open rule.
- A public construction remains genuinely blocked: exact interface/authoring obstruction.
Do not redefine K_join/G_outer or discard a permitted schedule after seeing its result. Minimize within this declared family; do not claim a globally smallest counterexample.

TeX remains the manuscript truth. Synchronize the corresponding Markdown discussion, preserve inherited mathematics and formatting, and keep theorem status aligned with evidence. Run tectonic alan-dissertation.tex once from paper/ after semantic/source review, accepting the established warning policy. Inspect changed PDF pages and avoid per-commit regeneration.

The final report and draft PR must state:
1. Whether the callable public J contract was realized, including present diagnostics on failed outcomes.
2. Whether the restricted lifting rule was established, and its exact candidate/environment/progress scope.
3. The bounded K1/A/B and outer-consumer results, actual census counts, concrete samples and controls.
4. What was proved in the finite semantics, mapped to artifacts, executed, source-inspected, or left open.
5. Exact source/evidence identities, tests actually executed, review disposition, and any unresolved blocker.
Arbitrary-context congruence, arbitrary authored candidates without the relation premises, recursive tree closure, universal compiler/scheduler correctness, exact-debit equivalence, and effectful/distributed recovery remain open.

Do not add an outer sibling S in this phase. That later extension would create a second independently timed interfering debit and would need a new environment argument. Finish this boundary first: a correct consuming composition exposing a public contract that its caller can use as one component.
Pinned primary sources
- PR24 and merge status
- Phase 1M qualification
- C_reason reference contract
- Existing C_reason authoring
- Public collection reply-then-Failure precedent
- Invocation outcome preparation and reply retention