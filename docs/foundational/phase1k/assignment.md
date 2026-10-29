ALAN Phase 1K Codex Brief
Required communication and safe termination
Work in JakeFAU/alan.

The next research question is:
When may an agent terminate without stranding communication that its composition still requires, and can the existing public authoring vocabulary establish that condition?

Execute this bounded investigation through concrete experiments, qualification, manuscript reconciliation, independent review, and a draft PR. Produce an execution-profile result and a precise account of the available observations; a new coordination feature is not the required outcome. A counterexample or demonstrated authoring limitation is a valid result.

Make routine decisions autonomously. Use independent work for the reference model/oracle and substantive review, resolve ordinary findings internally, and preserve failed hypotheses. No merge, release, paid provider calls, or messages to people are authorized.
1. Establish the actual baseline
Read current repository instructions and relevant Phase 1I/1J evidence before making changes. Preserve unrelated work and use an isolated branch/worktree.

Phase 1J is PR #21, reviewed head 1d13bc238aada500c3b9608d400435e235fedc0b, reviewed source aggregate 55260ec410e9b63393b5c24a211bdb70866228a8edb75d2ec4bc28345af8da3e. It was still an open draft when this assignment was prepared. Verify its current head, status, ancestry, and intervening changes.

If merged, use verified current main containing its result. If still open, branch from the verified Phase 1J head and create a stacked draft targeting that branch, clearly recording the dependency. Do not merge PR21, amend its historical evidence, or fold all its changes into an apparently independent PR. Record the actual baseline and final source identities.

Keep these historical identities distinct:
- Phase 1J experimental baseline: d68ec667d276a4bd9e1be11613905c628cfea704.
- Its integration base: 5764ec0b5679c72c80e56c44fd9e4198c986b91c.
- PR19's TeX formatting and PR20's mathematical correction are inherited work. Preserve them; the structured-loop correction is not an unresolved Phase 1J obligation.
Read selectively:
- docs/foundational/phase1j/00-plan.md through 05-decisions.md, the model counterexample, final review, and relevant raw/process records.
- Frozen examples/join/, its separate consumer module, research/join/, and additive join tests in internal/substitutiontest/.
- author/author.go, author/compile.go, the public runtime facade, and relevant provision/observation interfaces.
- internal/executor/messages.go, runner.go, store.go, and composition reply/outcome handling.
- The scoped manuscript discussion and applicable AGENTS.md instructions.
Phase 1J established a bounded distinction: the checked root-lifetime/resource obligations hold, concrete delayed-peer executions and four crash samples complete, but every modeled positive configuration also admits report-refusal pending states. The bad order is report commitment, peer termination, then first report authorization refused as source_closed. It is not yet an observed failure of unchanged automatic Store.Run.

Preserve that exact claim. Historical complete race coverage used the earlier oracle; latest-oracle race coverage was focused. Do not relabel it as a complete latest-source run.
2. Separate three execution profiles before proposing a repair
Define the domain of each claim:
Profile	What must be established
Independent protocol model	Which commitment, authorization, delivery, consumption, and terminal transitions it allows; its finite abstraction and fairness assumptions.
Supported public runtime driver	Which enabled transitions an external driver can legitimately select through exported APIs while executing the unchanged retained programs.
Unchanged automatic Store.Run	Which orderings the actual runner selects or excludes for the declared deployment and restart states.


A runtime sample admitted by the model does not prove every modeled counterexample is concretely realizable. A manually driven public execution is useful concrete evidence, but is not an automatic-runner failure.

Inspect actual ordering before adding experiments. At the reviewed source, the runner offers one internal delivery and then one local commit per loop; this is not a general drain-all-deliveries barrier. Determine the relevant pending queue and ready-run conditions for these fixtures. Justify any narrower ordering invariant from source and retained records.

Do not change the production scheduler, search arbitrary identities until an ordering appears, or use sleeps as causal evidence. If the bad automatic ordering is excluded for the frozen fixture, state that bounded exclusion. If it remains unresolved, say so. An unavailable observation/control point is an apparatus limitation, not an impossibility proof.

Freeze this profile map and independently review it before interpreting new results.
3. Identify the actual handoff and observation boundaries
Build a compact table tying each boundary to its implementation, durable record, and program visibility:
1. Ordinary message intent committed by Send.
2. First source authorization retained, allowed or refused.
3. Admission to the destination accepted or refused.
4. The destination observation retained.
5. The receiver actually selects and consumes that observation.
6. The sender and receiver reach local terminal states.
7. Aggregate composition completion is published.
Verify the current rules rather than copying conclusions uncritically:
- AuthorizeIntent records the first source disposition and returns that disposition on later attempts. A first attempt on a non-open ordinary-message source is refused; invocation replies have a separately justified exception.
- Prior allowed authorization can survive source termination. Destination status/generation, permission, and schema checks still matter at admission.
- AcknowledgeMessage records dispatch/admission linkage. It can acknowledge a refused admission; it is neither proof of accepted delivery nor proof of receiver consumption.
- An ordinary protocol ACK is another message subject to its own source authorization. Keep it distinct from dispatcher bookkeeping.
- author.Send currently exposes no retained message-result binding; ordinary Receive is not automatically a runtime receipt or terminal-observation wait.
Inspect the complete relevant public surfaces for exceptions. A result returned to the Go host is not automatically an observation available to a retained authored program.

Formulate the narrow hypothesis to test: with the receiver remaining admission-open, stable routing/generations, valid payload, and the stated eventual-service assumptions, source termination after retained allowed authorization should not itself invalidate this same ordinary message. This is a source-lifetime claim, not unconditional delivery or end-to-end completion.
4. Concretize J1's counterexample and its paired control
Keep J1's actual root/peer artifacts unchanged for this experiment, including A/B/S, the original peer prefix, topology, limits, and payload. Use candidate B, id1/value0, answer mode, I=R3, depth2, E0 as the primary concrete witness. Add A only if it resolves a distinct uncertainty.

Use a separate test driver through supported public runtime operations, such as inspection/snapshot, current authority, instruction commitment, authorization, and delivery. Discover exact signatures. It may choose legitimate enabled transitions and select actual matching observations; it may not fabricate observations, choose program branches, change registers, mutate authoritative database state, or reconstruct missing work. Do not run this driver concurrently with Store.Run.

For each manual local commit, bind the unchanged program identity, preceding instruction/PC and state, selected observation if any, and resulting retained history. A host operation with an equivalent apparent effect is not a substitute for executing that instruction.

Obtain two fresh executions of the same provisioned program:

LATE:
- Candidate and S results are obtained through their actual invocation paths.
- Peer consumes S and commits its truthful report.
- No first report authorization exists.
- Peer executes its genuine remaining instructions and terminates.
- The first authorization of the same report is attempted and refused.
- Retain any actual downstream admission/refusal record; if admission was not attempted, explicitly record its absence. Establish absent report observation/consumption and the actual root/aggregate state.
EARLY:
- Reach the corresponding truthful committed report.
- Retain its first allowed authorization while the peer is open.
- Let the peer terminate before report admission.
- Deliver that same report to the still-open root.
- Verify the retained authorization is reused, admission is accepted, and the root consumes the original payload and reaches its actual outcome.
These are predictions until executed. Record any different result as evidence. Preserve real intermediate states and causally linked records, not merely final statuses or harness annotations.

Run the unchanged automatic profile separately on the frozen fixture and, where feasible, the precise preauthorization restart state. Explain the correspondence or difference. A legal manual LATE witness establishes transition-system reachability even if the automatic sample selects EARLY.

If a purported public transition cannot be executed, retain the exact API/precondition obstruction. Do not substitute private methods while labeling the result public.
5. Freeze one acknowledgement protocol and attack it
Introduce a separately named J2 candidate using existing public authoring operations:
- Peer consumes S and sends the same truthful report, then waits for a correlated ordinary ACK before local Success.
- Root consumes the candidate outcome and report, commits that ACK, then performs its ordinary final decision.
- Root and peer retain the same membership, request identity, generation, resource limits, and A/B/S artifacts. New context artifacts, ports, and report/ACK correlation receive new identities.
Specify the full branch policy before implementation. Retain J1's truthful outcomes and pending discipline; known negative outcomes are reportable, and the peer's reporting-program Success must not be confused with S success. The initial distinguishing profile is successful answer mode, so a failure there suffices to falsify its proposed completion claim.

The independent model must represent the expected moved race:
Root consumes report → root commits ACK → root terminates → first ACK authorization is refused → peer remains waiting.

The peer's wait may protect the original report while the new message remains exposed. Distinguish root local success from aggregate completion with an open peer.

Start with id1/value0, depth2, I=R3, E0 and a reference K1 local/helper choice; map actual A/B behavior independently where used. State exact state/rank bounds and inspect every reachable maximal state in that declared product. Preserve the original J1 eight-configuration result unchanged. A universal completion claim is falsified by one valid trace; do not require another full payload-by-recovery campaign after that trace is established.

Realize the decisive J2 trace through legal public transitions if possible, with a paired ACK-authorized-before-root-termination control. Record actual resource spending: successful A/B workloads remain two/three dynamic invocations and runs, with two fixed provisioned runs and zero effects. Ordinary ACK communication must not hide additional admissions.

Stop this candidate once its decisive result is established. Do not add ACK-of-ACK rounds or an unbounded retry mechanism to chase success. One failed handshake does not establish an impossibility theorem for all finite ordinary protocols.
6. Test authorization persistence at exact process cuts
Use the existing acknowledged checkpoint/SIGKILL/fresh-process facilities. Keep the original database, logical identities, artifacts, invocation handles, message identity, authorization identity, and charges. Renew activation ownership normally.

Boundary hooks may execute while a store mutex is held. Use the existing signaling facilities without reentrant store calls; inspect at a safe boundary outside that lock. Verify the committed predicate rather than inferring it from a checkpoint name.

Use three focused B/J1 cases:
Exact retained cut	Required observation after reopening
Report committed; no first authorization; peer still open at its post-Send continuation	Run the declared automatic or public-driver profile, explicitly recording which. Establish the actual authorization/termination order and outcome.
Allowed authorization committed; peer terminal; report not yet admitted; root open	Deliver the original report and verify whether its prior authorization survives reopening and source closure.
First authorization refused after peer termination	Reattempt the same authorization/delivery identity and verify the retained refusal is not converted into fresh permission. Preserve the pending state truthfully.


The prior J1 cut “report committed, root unconsumed” does not establish that authorization was absent. Do not reuse it under the stronger label without inspecting its records.

No redeployment, replacement report, refund, fabricated receipt, authoritative SQL edit, or synthetic stop/resume in place of a real process kill. If a cut is unsupported, report the actual achieved boundary and why. A bounded quiescent observation is not a proof of infinite waiting.

Do not multiply these cuts across all payloads/candidates. Existing unchanged recovery evidence retains its original source labels; qualify the newly examined authority boundaries.
7. Determine where a usable completion fact is expressible
Audit three surfaces separately:
- Ordinary retained programs built with the existing author constructors.
- Lower-level retained artifacts constructible through the public runtime interface.
- Host-side dispatch/inspection operations.
The runtime already has distinct terminal-notification and invocation-outcome mechanisms. Inspect whether and how an external public consumer can provision and consume them. A raw public artifact may express an observation that current author constructors cannot; distinguish that from a missing runtime capability. Do not claim an API-wide absence from inspecting author.Send alone.

Attempt one minimal external-consumer probe, with no internal imports, that provisions a terminal-producing peer and a lower-level retained receiver for its authentic runtime-terminal observation. Verify source termination, the retained terminal route, delivery, and actual receiver consumption through the existing public runtime interface. Retain successful execution or the exact compile/provision/execution obstruction. This establishes accessibility at that interface level only; do not extend it into a third complete join protocol, rewrite compiler output, or alter the J1 root.

Explain which closure-stable fact could discharge the outstanding communication obligation, and what an ordinary program can actually observe today. For example, a report/ACK protocol followed by an authentic runtime peer-terminal observation is a candidate for later study only if its observation path is available. It must not be silently presented as another ordinary ACK.

Document the invoked-peer alternative separately: invocation replies have different settlement semantics, and invoking the peer changes topology and accounting. It is not the unchanged static-peer solution and is not a new campaign in this phase.

If the authoring surface lacks the needed observation, give a source-backed, precisely scoped limitation and the smallest next semantic question. Do not add a constructor, message receipt, lease, exception, scheduler rule, or new lifetime policy in this increment. Any general non-expressibility argument needs its own explicit language/scheduling assumptions and proof; examples alone are insufficient.
8. Keep the evidence and implementation narrow
Keep production executor, compiler, schema, dependencies, accounting, and scheduler unchanged. Preserve C0, K1, J1, their artifacts, model results, and negative controls. Extend research/context/test code additively under repository conventions.

The raw oracle must link original intent, first authorization disposition and snapshot, actual admission result/observation, selected consumption, source termination, and aggregate status. Use actual causal records; do not invent a global order across unrelated per-run histories. Distinguish scope_closed, source_closed, destination refusal, resource denial, and a valid pending state.

Add focused decoder/adversarial checks for the changed mapping: a dispatch acknowledgement of denial must not become successful delivery; admission must not become consumption; a different message/authorization must not discharge the original obligation. Reuse unchanged truthful-report/correlation checks with explicit source identity.

Have independent reviewers challenge the profile map, public-driver legality, model/artifact correspondence, and final claims. Fix findings and rerun affected checks. Run focused normal/race/public-consumer/process checks and repository-required gates with the actual pinned toolchain and offline dependency settings. Record exact commands, environment, selected cases, exits, source/artifact hashes, and evidence identities. Passing zero selected tests is not a gate.

Reuse earlier evidence only within its exact scope. Archive compact decisive traces, source mappings, counterexamples, and reproduction commands; avoid copying previous evidence collections or running unrelated campaigns solely because an earlier phase did so.
9. Deliverables and stopping rules
Write a short inspected implementation plan, then create:
- docs/foundational/phase1k/00-plan.md.
- 01-communication-contract.md: profiles, hypotheses, J2, observable facts, bounds, and authority/lifetime assumptions.
- 02-qualification.md: paired results, exact execution level, recovery cuts, and expressibility findings.
- 03-review.md, 04-runbook.md, and 05-decisions.md.
- A compact machine-readable evidence index and the decisive raw/model traces.
Reconcile the relevant Markdown and LaTeX manuscript statements. Preserve the inherited math correction and formatting. The .tex source is the manuscript truth; the PDF is a renderability artifact. Run Tectonic once at phase end, after substantive manuscript review, using the established build command. Accept the previously authorized warning policy. Do not regenerate or commit a new PDF on every implementation commit.

Phase 1K is complete when it establishes:
1. The relationship, including any unresolved correspondence, between the abstract bad schedule, supported public transition control, and the automatic runner.
2. The paired first-authorization-before/after-termination result, or a concrete reachability obstruction.
3. A bounded result for the single frozen J2 acknowledgement protocol.
4. Exact retained-authorization recovery evidence or named unachieved boundaries.
5. Which completion facts the current public authoring/runtime surfaces expose.
Classify each result as proved, bounded model-checked, concretely tested, source-inspected, conjectured, or unresolved. A model-only failure must retain that label. A public-driver witness must not be promoted to an automatic-scheduler failure. A source-inspected authoring gap must not become a universal impossibility claim.

Finish with a reviewed draft PR and report its URL/dependency, source identities, changes, actual validation, findings, and next unresolved question. Do not merge.

The intended learning is precise: determine whether the next obstacle is execution ordering, missing program-visible evidence of authorization, or the chosen acknowledgement protocol—and identify the exact boundary before changing the runtime.