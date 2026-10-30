# Phase 1O finite certificate validation contract

Status: frozen before implementation after Task2 conditional-argument approval. This specifies expected clause-level certificate outcomes, not executed results or candidate acceptance.

## Sources and exact question

The sole Phase 1O semantic source read was `docs/foundational/phase1o/01-weak-contract.md`, SHA-256 `ac1c0ecbf62bd82c718a7a3ad3dcf000552d14e093868321b0f7dcc4b370b000` (verified with `shasum -a 256`). This specification derives independently from its §§3–7, original K1 §§Frozen finite states, Tickets, Separately checked local relation, and Identity/hiding. I did not read `02-arg2.md` or another agent's argument/review report.

Source inspection was limited to `research/substitution/model.go`, `relation.go`, `callable_join_local.go`, `callable_join_relation.go`, `invocation_join.go`, `invocation_join_oracle.go`, relevant `callable_join.go` authorization/Send/End edges, and production `messages.go:103–177`, `invocation.go:123–140,218–224,296–312`, `invocation_outcome.go:164–184,354–373,426–440,571–621`. The old checkers are historical single-event, canonical-erasure witnesses; none supplies the arbitrary weak relation proposed below.

Question: does a supplied original-state relation, with exact finite matching paths, satisfy the declared clauses? Existence of another relation is outside scope.

## 1. Minimal interface: finite supplied-certificate validation

Record-level interface only; no generic framework, solver, runtime, composition adapter, or public API is requested.

A problem record contains:

- Two finite, explicitly enumerated directed graphs X and K. Nodes and edges have stable IDs; edges retain source, target, exact normalized event facts, and owner/environment classification. The selected finite alphabet and any silent classification are declared before validation.
- Legal initial node IDs, immutable configuration/request/identity-renaming data, and the exact finite scope. Abstract machinery fixtures below have an explicitly declared toy grammar. K1 clause fragments instead name original transitions and give a legal-prefix anchor. A fragment's omitted actions are named ports, not suppressed enabled actions in a claimed complete graph.
- A required endpoint projection table P on nodes. For K1, the selected F/P reading and all mandatory original facts must be explicit. A private label never licenses erasure of a charge, commit, consumption, terminal fact, outcome, or repeated visible occurrence.
- Optional strong-clause tables, each separately named: node demand derived from declared tickets; caller/current authorization guard predicates; retained first dispositions. Each table needs fixed state and edge accounting abstractions from the represented fields (source/target tickets, exact debit, and update), checked against original transitions. It cannot be a favorable annotation attached to a path.
- Optional completion-boundary predicate and natural-number rank on the complete finite graph used for a progress claim. No progress claim is made for an open-port fragment.

A supplied witness contains:

- A finite relation R, explicitly a set of original X-node/K-node pairs, under one fixed identity renaming. R is not required to be functional.
- For each related reachable pair and **every outgoing X component edge**, one exact finite list of K-edge IDs starting at that pair's K node. Its endpoint is named and paired with the X successor in R. An empty list is a real submitted stutter, not permission to search for a better path.
- For each related reachable pair and **every outgoing declared environment edge on either side**, one immediate counterpart edge ID on the other side, with identical event/ledger facts and related successors. Multiple edges carrying the same label still have separate obligations. The counterpart is existential for that edge, but is not a multi-step path.
- Where legal reachability is claimed beyond an original K1 initial, an exact predecessor path from a legal initial or the stated original finite prefix to the fragment's anchor. Authorization records additionally name the immutable intent descriptor and original binding/source snapshots needed to check each claimed first disposition and repeated lookup.

Check reachability from legal initials, independently of relation existence. Well-formed disconnected nodes are harmless; claimed legal reachability must be justified.

For component transfer, concatenate the mandatory visible occurrences of the submitted reference path, in order, and compare them with the candidate edge's exact occurrence word. For the fixtures this is either the empty word or a singleton. A visible candidate edge must match exactly its visible occurrence, with only genuinely silent reference edges before/after it. A genuine candidate private edge must preserve P and every required candidate public fact; its reference path has the empty occurrence word. Ownership and event facts come from the declared grammar/original rule, not a mutable `Environment` Boolean. A component event cannot be substituted for declared environment service.

Initial coverage requires an R pair with an aligned legal K initial for every declared X initial/configuration. Check P at every related reachable pair and path endpoint. Interiors need not belong to R. R, renaming, and accounting abstraction stay fixed across branches; different outgoing events may use different paths.

For an optional strong-clause certificate, check equality of the separately named demand/guard/first-disposition tables at related pairs, and preservation of each relevant current guard dependency by every matching-path **silent** reference step. The matched visible event may change its own stage. Only implemented strong clauses may receive PASS; these partial checks cannot be called full `ARG1Compatible_N` without the remaining requirements from frozen §7.

### 1.1 Do not change the original relation domain

R relates original X/K1 states, not histories or history-unrolled X. Accounting must be a declared function of represented X state, or be reported unavailable. History-dependent annotations do not establish that function.

Separate sidecar/provenance validation checks authorization legality. It neither proves arbitrary raw X determines demand nor repairs R by splitting X states. Coherent relation extension with actual retained sidecars remains a separate obligation.

### 1.2 Results stay separate

The report has separate fields, not one conflated verdict:

| Field | Meaning |
| --- | --- |
| Input / legal-certificate status | Well-formed graphs/certificates and legal claimed original transitions/initials/provenance, or an exact error. |
| Candidate-domain admissibility | Stated A_op/A_pub scope and independently checked ownership/correspondence, or not assessed. Toy fixtures are not ALAN candidates. |
| Accounting | Exact original spending/tickets/demand abstraction and its checks, or not assessed. It is not inferred from a successful output trace. |
| Endpoint + component-transfer witness | P equality and all outgoing X component edges checked from each fixed related pair against the submitted paths. |
| Bilateral environment readiness | Immediate matching service in both directions, at every related reachable pair. |
| Weak supplied witness | PASS only when the preceding weak relation clauses all pass for the stated finite scope/reading. |
| Progress | Independently established, falsified by a concrete eligible divergence/stuck behavior, or not established. |
| Optional strong-clause supplied witness | Exact named demand/guard/disposition obligations, separately from weak and progress. |
| Existence of another relation | NOT DECIDED. A rejected supplied relation/path is never NONE. |

A missing obligation, a legal but wrong matching word, a wrong endpoint pair, missing readiness, or strong-fact inequality rejects the submitted witness. A dangling ID, noncontiguous alleged path, false legal-initial claim, fabricated binding, or alleged authorization transition inconsistent with the original rules is an input/legal-certificate error. Neither is an accepted semantic counterexample.

Rejected paths leave alternatives open. Interrupted validation is incomplete. Bounds derive from submitted graph/relation/list sizes; no arbitrary cap, NONE verdict, or existence-search API is needed.

### 1.3 Minimal positive progress certificate

For a complete finite open/no-closure fixture, a rank into the natural numbers that strictly decreases on every edge outside the designated completion/pending boundary, together with checks of every reachable zero-outdegree node and every declared boundary, is a sufficient positive certificate. Boundary nodes must have their specified completed/pending facts; a pending boundary has no supplied release. This stronger sufficient method avoids claiming completeness of rank search and needs no SCC machinery.

Rank rejection alone is inconclusive; an exhibited eligible private cycle falsifies nondivergence. Actual K1 claims need explicit entitlement/service premises. Repeated authorization lookup checks query equality, not an invented semantic idle cycle.

## 2. Five mandatory machinery fixtures

F1–F4 use toy labels, not ALAN operations: a/b are distinct mandatory visible occurrences, d declared environment service, tau fact-preserving private work. Other edges are component events. Their grammar is explicit; A_op/A_pub and D_N/full conformance are NOT ASSESSED. F5 checks original authorization legality.

Listed edges are exhaustive. Nodes are reachable except F5's disconnected control. Initials, pairs, and paths below are part of the proposed freeze.

### F1 — Nonfunctional relation and alternative finite paths

X initial `x0`, edges `A: x0 -a-> x1`; boundary `x1`.

K initial `k0`, edges:

- `L: k0 -tau-> kl`; `R: k0 -tau-> kr`;
- `AL: kl -a-> k1`; `AR: kr -a-> k1`; boundary `k1`.

Projection: P(x0)=P(k0)=P(kl)=P(kr)=0; P(x1)=P(k1)=1. All optional strong demand/guard values are the same constant 0; they do not introduce a hidden distinction between kl and kr.

Submit `R1={(x0,k0),(x0,kl),(x0,kr),(x1,k1)}`. Exact component certificates:

| Pair / X edge | Submitted K path |
| --- | --- |
| (x0,k0) / A | [L,AL] |
| (x0,kl) / A | [AL] |
| (x0,kr) / A | [AR] |

Valid alternatives: first path `[R,AR]`; or smaller relation `R1small={(x0,k0),(x1,k1)}` using either first path. No canonical erasure or functionality is required.

Expected: input/legal grammar PASS; endpoint/component witness PASS; readiness PASS vacuously; weak witness PASS; constant strong-clause witness PASS. Positive ranks: X `(x0:1,x1:0)`, K `(k0:2,kl:1,kr:1,k1:0)`. Both complete toy graphs progress to their boundary. These are not original K1 progress/conformance verdicts.

Bad witness: replace the `(x0,kl)` path by `[AR]`. AR starts at kr, so this is a noncontiguous path/input-certificate error. A structurally valid alternative bad witness is the empty path at `(x0,k0)/A`: it is rejected because `[] != [a]` and its endpoint is not the required successor pair. Neither failure invalidates the valid alternatives above.

Hand check: each pair's sole event has an a-only path from its fixed K node to `(x1,k1)`; silent steps preserve P/guards; no environment obligations.

### F2 — Same outputs, missing immediate environment readiness

X initial `x00`, with nodes `x10,x11,xd0,xd1` and edges:

- `T: x00 -tau-> x10`;
- `A00: x00 -a-> xd0`; `A10: x10 -a-> xd0`; `A11: x11 -a-> xd1`;
- `D10: x10 -d(env)-> x11`; `DD: xd0 -d(env)-> xd1`.

K initial `k0`, nodes `k1,kd0,kd1`, edges:

- `D0: k0 -d(env)-> k1`; `DK: kd0 -d(env)-> kd1`;
- `AK0: k0 -a-> kd0`; `AK1: k1 -a-> kd1`.

P is `(aOccurred,dOccurred)`: x00/x10/k0=(0,0); x11/k1=(0,1); xd0/kd0=(1,0); xd1/kd1=(1,1). Boundary is `(1,1)`.

Submit `R2={(x00,k0),(x10,k0),(x11,k1),(xd0,kd0),(xd1,kd1)}`. Component paths: T maps to `[]`; A00 and A10 map to `[AK0]`; A11 maps to `[AK1]`. From `(x10,k0)` match D10/D0 immediately in both directions; from `(xd0,kd0)` match DD/DK immediately in both directions. At `(x00,k0)` the K edge D0 has **no** X counterpart. The attempted path `[T,D10]` is forbidden as an immediate readiness counterpart.

Expected: input/legal toy grammar PASS; endpoint/component-transfer witness PASS; all candidate-origin environment checks PASS; bilateral readiness FAIL exactly at `(x00,k0), D0`; weak witness REJECTED; strong readiness witness REJECTED. The two graphs have the same completed words `{ad,da}` after erasing T, which is insufficient for readiness.

Toy progress PASS by ranks X `x00:3,x10:2,x11:1,xd0:1,xd1:0`, K `k0:2,k1:1,kd0:1,kd1:0`. Optional accounting is 0. This delayed-kernel-service toy has no A_op ownership justification.

Hand check: component words/endpoints agree, but D0 at the initial pair cannot require T first.

### F3 — Safety permits private divergence; progress rejects it

X initial `x0`, sole edge `T: x0 -tau-> x0`, no completion boundary reachable. K initial `k0`, edge `A: k0 -a-> k1`, boundary k1. P(x0)=P(k0)=0; P(k1)=1. Submit `R3={(x0,k0)}` and matching path `[]` for T. All optional strong values at x0/k0 are constant 0. There are no environment edges.

Expected: legal toy grammar PASS; endpoint/component witness PASS; readiness PASS vacuously; weak safety witness PASS; constant strong safety clauses PASS. Progress FAIL with the exact infinite execution `x0 -T-> x0 -T-> ...`. A positive rank would require rank(x0)>rank(x0), impossible. There is no candidate output action being unfairly starved: T is its only enabled action, and no declared environment service is withheld. The reference's optional component a is not a bilateral requirement of asymmetric safety.

Separate progress therefore rejects combined conformance; no ALAN candidate/contextual counterexample is claimed.

### F4 — An extra visible occurrence cannot be a private stutter

X initial `x0`, edges `A: x0 -a-> x1`, `B: x1 -b-> x2`; boundary x2. K initial k0, edge `AK: k0 -a-> k1`; boundary k1. P(x0)=P(k0)=0, P(x1)=P(x2)=P(k1)=1. The fact that b leaves this toy endpoint projection unchanged does not erase its declared visible occurrence.

Submit `R4={(x0,k0),(x1,k1),(x2,k1)}`. A maps to `[AK]`. The purported B certificate is `[]` at `(x1,k1)`.

Expected: legal toy grammar PASS; endpoint equalities PASS; B component transfer REJECTED because `[b] != []`; readiness PASS vacuously; weak witness REJECTED. A classifier that calls B private merely because P(x1)=P(x2) is wrong. Strong event transfer rejects for the same reason. Toy ranks X `(2,1,0)` and K `(1,0)` show complete toy progress PASS, independently.

Hand check: k1 has no outgoing path; [] cannot emit b. This occurrence control makes no ALAN duplicate-action admissibility claim.

### F5 — False reachable authorization history is an invalid certificate

Use original bound reply r for admitted e, request `{id:1,value:0,mode:answer}`, correct reply `{id:1,value:1}`, stable original revision/generation, unretired source, and the original binding predicate (protocol/message/entry member/caller subject/caller generation/obligation all match). The legal-prefix anchor comes from:

`invoke_commit(e); admit(e,accepted,+1,+1,0); request_consume(e); choose_local; reply_commit(e,{1,1})`.

No q or closure transition occurs in this prefix. Their exclusion from this narrow schedule is not a claim that D_N disables them. The reply is committed, not yet authorized/admitted; e is still open. This is a reached boundary fragment, not a replacement D_N initial.

The certificate claims:

`terminal_freeze(e,succeeded); firstAuthorize(r,Allowed=false,Reason=source_closed)`.

Expected: INVALID LEGAL CERTIFICATE at the firstAuthorize edge. In the original first-authorization switch, stable revision and generation are checked before the original bound-reply exception. With those equal and r bound, closed source alone does not deny r; the only legal result here has `Allowed=true,Reason=authorized`. A complete path/provenance checker must reject the claimed denied successor before treating it as a relation counterexample. Weak/strong/progress verdicts on the bad alleged graph are NOT ASSESSED, not semantic FAIL/NONE.

Positive control: replace the last record by the actual first result `A_late=(ID=authorization(r),IntentID=r,Source=Authority(source at lookup),ClosureOperation=the actual queried value,Allowed=true,Reason=authorized)` and a repeat lookup returning **that exact same record**. The prefix is legal for the stated authorization clause; repeat lookup may not recompute a different result after End.

Disconnected-node control: a finite graph may additionally contain a node holding a denied sidecar with no incoming legal path and no claim that it is reachable. Its mere existence does not reject the valid reachable certificate. Adding the false reachability claim/edge above does reject it. Structural well-formedness of disconnected data is still required.

This checks the original clause, not general realizability; false first/repeat records are provenance errors, not ARG-2 refutations.

## 3. Minimal original K1 accounting/branch clauses

These clauses have actual K1 tickets and operation fields, not toy demand annotations. They are finite boundary fragments and have no W_N/W_full or Progress_N verdict.

Fix original local configuration C: request `{1,0,answer}`, entry depth2, cap3, residual I=R3, E0, q possible, root open, with no prior local spending. The original D_N initial has all e/h protocol fields empty. Reach anchor B through exactly:

`invoke_commit(e); admit(e,accepted,+1,+1,0); request_consume(e)`.

At B: spent=(1,1,0), e ticket spent/accepted/input consumed, e control choose, e freeze none, h ticket possible/uncommitted/disabled, q possible. Identities and all untouched protocol fields are fixed as in original Initial and the prefix. Original tickets, not observed candidate termination, supply demand.

### 3.1 Exact finite fragment generator

The fragment's node fields are `(branch,qState,closed,hStage)` plus the fixed prefix fields above. `branch` ranges undecided/local/helper; qState possible/spent; closed false/true. Initially `(undecided,possible,false,uncommitted)` is B. hStage remains uncommitted outside helper branch; in helper branch it ranges uncommitted/committed/accepted/rejected_scope. Only states reached by the following rules are included:

1. If branch=undecided, choose_local sets branch=local, e control=mode, h ticket=excluded/disabled; choose_helper sets branch=helper, e control=invoke, h ticket=possible/enabled. Both are original component events. They add no debit.
2. At any open node with qState=possible and remaining I/R, peer_debit(q) changes it to spent and increments actual I/R once. No q debit exists after closure or after q spent. It changes no e/h ticket or reply/terminal fact.
3. At any not-closed node, close_root sets closed, root caller cancelled/terminal, retains every charge/ticket/protocol fact, and never reopens the root. Already admitted e remains locally open here. This rule does not authorize future caller commit/selection.
4. In helper branch with hStage=uncommitted, invoke_commit(h) sets hStage=committed, h ticket=committed, and e control=await, with original same-id/value answer request; no debit. The source is e, whose freeze remains none. Root closure does not itself freeze e; new h admission still obeys rule 5.
5. For hStage=committed, admit(h) is an immediate environment action. If closed it sets hStage=rejected_scope, excludes the unspent h ticket, gives exact scope_closed and no debit. If open it sets hStage=accepted, h ticket spent, creates/adopts its admitted request, and increments I/R once. Capacity is sufficient even if q spent: e+h+q costs at most (3,3,0), depth3 is allowed. No h admission repeats.

Omitted actions are explicit ports: e mode actions; helper local work; later publications/selections. They remain original enabled actions, so omitting them proves no W/readiness claim. q/closure stay available after h acceptance/rejection under their stated guards. Preserve each event's original facts, obtaining terminal status from Freeze when the coarse label is empty.

Listed finite fields/monotone stages define this fragment exactly, without caps or added private PCs.

### 3.2 Fixed accounting values and possible/local normalization

Write L for B after choose_local, H for B after choose_helper, and append q to denote the one q debit. U is exactly the set of original possible/committed tickets. The demand column is future unspent demand only; actual spending remains separate.

| Boundary | Actual spent I=R | U | Future demand I=R | spent + demand |
| --- | ---: | --- | ---: | ---: |
| B | 1 | {h,q} | 2 | 3 |
| L | 1 | {q} | 1 | 2 |
| H | 1 | {h,q} | 2 | 3 |
| Bq | 2 | {h} | 1 | 3 |
| Lq | 2 | {} | 0 | 2 |
| Hq | 2 | {h} | 1 | 3 |
| H after h commit | 1 | {h,q} | 2 | 3 |
| H after accepted h | 2 | {q} | 1 | 3 |
| Hq after accepted h | 3 | {} | 0 | 3 |

Closure retains these ledger/ticket values until a listed refusal legitimately excludes the unspent h ticket. No closure-progress assertion follows. A spent ticket never becomes excluded to refund a charge. In particular local choice removes future h demand, not the already spent e debit. Commit transfers possible to committed without freeing demand; acceptance transfers committed to spent without freeing the root's spent+U amount.

The possible/local branch window can be used only as a conditional **pair test**. Let an independently represented local-branch state xL have an explicitly defined abstraction to L's original fields, including h impossible/excluded. Then `(xL,L)` has equal original ledger/tickets/demand. `(xL,B)` has equal ledger but different future demand (1 versus 2). A demand-equality strong clause rejects that latter pair. Whether it can occur in a weak witness depends on the unresolved selected F/P interpretation and the rest of the graph; this fixture does not settle it.

Under a declared F_event/P_stage reading, choose_local is visible and B/L disagree in the required projection. Under a declared F_fact/P_boundary reading, a proposed silent choose_local still needs explicit review of projection and guard dependencies. The concrete change h possible→excluded above remains true; declaring the edge silent does not prove those dependencies stable. A reference path crossing B→L cannot receive a strong guard-stability PASS merely because its endpoints later agree. Alternative related states/paths such as L remain available in a separately submitted relation; rejection of `(xL,B)` proves no nonexistence claim.

Demand reads xL's declared state fields. Missing state information leaves accounting unresolved; never split xL by history or vary its demand by visit.

### 3.3 q, closure, fork, and a merged illegal h edge

Freeze these exact local observations:

- q can occur immediately from B, L, H (and their helper-stage descendants) while open and unspent, independent of a private instruction address. At Bq/Lq/Hq it cannot repeat. Endpoint matching that waits for an X component step before q fails readiness.
- close_root can occur immediately at every open fragment node, including B/L/H and nodes after q. At each closed counterpart q is absent; prior spending and intent facts persist. Refusal of a subsequently serviced committed h is scope_closed because e remains open in this fragment. This is not an invented source_closed schedule.
- The fork B→L and B→H is existential for asymmetric **component** transfer. A local-only candidate is not required to implement H merely because the reference can choose helper. Environment service from each actually included related pair remains bilateral.
- `B; choose_local; invoke_commit(h)` is not a legal original history. At L, h is disabled/excluded; the original Invoke guard has no edge. A quotient node M that merges L and H and imports H's h-commit successor may not claim that imported edge is legally reachable after the local branch. Checking a favorable representative from H while using L's demand is inconsistent provenance.

Expected for the last submitted history: INVALID LEGAL CERTIFICATE at invoke_commit(h), before a weak/strong/candidate verdict. The legal alternative `B; choose_helper; invoke_commit(h); admit(h)` is accepted by these fragment rules, with the open/closed admission result exactly as rule 5 specifies. These are original finite rule instances, not an A_op negative, not full conformance, and not permission to alter original X state identity to repair a quotient.

### 3.4 History annotations do not necessarily descend to a raw-state map

This additional diagnostic tests an accounting-map certificate, not W. Use the same lawful original prefix to B and its local extension to L. The original histories `pi_B=prefix` and `pi_L=prefix;choose_local` are both lawful; their demand vectors differ as recorded above. Consider this fully specified raw representation fragment, anchored at raw u after the common entry prefix:

- `t_keep: u -> m`, with no mandatory visible occurrence;
- `t_local: u -> m`, with no mandatory visible occurrence;
- `a: m -> z`, the one correct reply_commit(e,{1,1}) occurrence.

The fragment has only local reply work after m, no h-invocation edge, and no cycle. Declared alignment of t_keep is the empty reference path from B to B; alignment of t_local is `[choose_local]` from B to L. From m, the proposed reference paths for a are `[choose_local,reply_commit]` at B and `[reply_commit]` at L, both ending at the same original replied-local boundary S. Thus the submitted original-state relation is exactly `{(u,B),(m,B),(m,L),(z,S)}`. These alignments are conditional on an explicitly approved F_fact/P_boundary reading that admits their silence/projection; F_event/P_stage rejects earlier. q/closure and later outcome actions are unassessed ports, so this diagnostic asserts no weak conformance or progress premise.

The accounting-map certificate additionally claims that f(m) equals future h+q demand from the reference history associated with the visit. The t_keep history would require f(m)=(2,2,0); the t_local history would require f(m)=(1,1,0). Since m is one original raw state, no such function f exists. An all-successor history lift retaining both annotations still has two values over m; it does not make f a raw-state function. In the t_keep alignment, h exclusion occurs later inside the submitted matching path for a; the mere fact that m's future raw behavior is local does not make the earlier history annotation equal to the local-excluded annotation.

Expected: REJECTED ACCOUNTING-MAP CERTIFICATE / REPRESENTATION DIAGNOSTIC, with m and those two required values as the finite reason. If a legal original-transition claim and its state fields are well formed, this demand inconsistency is distinguished from the illegal imported h edge in §3.3. It does not show any raw X is weak-conforming, prove that these alignment hypotheses follow from K1, refute existence of another accounting/strong relation, or establish NONE. A different relation using normalized L instead of B may be worth checking after the interpretation gate; rejecting this supplied map says nothing about that relation's existence. Replacing m by history-indexed states would change the original relation domain and is not an accepted repair of this certificate.

## 4. Minimal first-authorization clause matrix

Use one immutable already committed original bound reply r. Source generation/revision and binding remain stable, caller remains eligible, and no root closure/retirement occurs. Check two source End statuses `t ∈ {succeeded,failed}` with the exact original binding test. A succeeded e-after-answer example witnesses t=succeeded. For t=failed the source is the already invoked J that commits its diagnostic before its original End; do **not** assert that K1 e in fail mode sends a reply. The callable-join original rules separately permit J's diagnostic Send and failed End.

The exact finite schedules are these four schemata, with one repeat query apiece:

| Schedule | Required first result | Required repeat |
| --- | --- | --- |
| commit(r); firstAuthorize(r) while open; End(succeeded); lookup(r) | allowed/authorized | exact saved first record |
| commit(r); End(succeeded); firstAuthorize(r); lookup(r) | allowed/authorized | exact saved first record |
| commit(r); firstAuthorize(r) while open; End(failed); lookup(r) | allowed/authorized | exact saved first record |
| commit(r); End(failed); firstAuthorize(r); lookup(r) | allowed/authorized | exact saved first record |

These are clause instances under legal boundary premises, not complete runtime executions. First records retain the actual Authority/closure fields read then. Early/late snapshots may differ; each repeat returns its own exact saved record. Reply admission, presence, publication, and consumption remain separate.

Bad certificates to reject before semantic classification:

1. In any late bound-reply row, first deny with source_closed alone despite matched revision/generation/binding.
2. In any row, change a saved first allowed result to denied at repeat lookup, or change its recorded source/closure fields to a fresh snapshot.
3. Authorize r before r was committed, or replace its immutable descriptor at repeated lookup. The actual old-descriptor mismatch is a conflict/error, not a fresh first disposition.

Retain two separating negative **legal-result** controls: if source revision mismatches at first authorization, source_revision_changed wins even for a bound reply; with matching revision but mismatched generation, source_generation_changed wins. These are legal denials under their changed premises, outside the positive stable profile. Conversely, the bound-reply exception must not be generalized to a nonreply h invocation: with no earlier authorization and a source already closed, its first disposition is source_closed; with an earlier allowed first result, lookup retains that result but the later invocation-admission source/scope checks still apply. These controls check the order of the existing original switch; they do not add retirement/cancellation experiments to E_N.

Sidecars remain candidate-inaccessible. Valid clauses do not establish raw-state demand or arbitrary R's coherent extension across service orders.

## 5. Controller rulings needed before implementation

1. Confirm that Task 3 actually needs supplied-certificate machinery after the Task 2 semantic review. Preparation here is independent; no unresolved proof step is declared fixed.
2. Choose and name the exact weak F/P interpretation used by any future K1 certificate. The branch-window clause exposes its bookkeeping/demand issue but intentionally does not prejudge the interpretation.
3. Confirm whether optional demand/current-guard checks are useful. If implemented, name the exact strong clauses and independently define their node-state abstraction; do not label that subset full ARG-1 without its other clauses.
4. Retain the error taxonomy and original state-pair domain. In particular a legal-transition/provenance error is not a candidate counterexample, and accounting cannot be supplied by silently history-splitting arbitrary X.

No necessity for an existence-search/NONE API, SCC engine, arbitrary validation caps, candidate campaign, public witness, new composition adapter, production change, manuscript edit, rendering, test run, commit, or subagent is established by this preparation.

## 6. Append-only normalization fixture (controller-requested)

The preceding body is preserved byte-for-byte at SHA-256 `731ac0d3c8e1b93eec45ab5f793be0ceb59603e27e67693d23fbd1f35e6d0684`. This addition specifies a nonconstant, explicitly state-accounted normalization example. It is conditional on the declared F_fact/P_boundary projection below, not a full W_N/A_op/A_pub or progress verdict.

### 6.1 Complete finite scope, states, and accounting

Use C from §3 and the lawful prefix `invoke_commit(e);admit(e,accepted,+1,+1,0)`. The fragment begins **before request consumption**, with the original e admitted/open, request admitted/unconsumed, h uncommitted/disabled/possible, q possible, root open, actual spent=(1,1,0). This anchor supplies initial equality of protocol, spending, and demand. The original caller is awaiting e; closure cancels that caller but does not freeze the already admitted e.

Write every state as `(control,q,c)`, q∈{0,1} for q possible/spent, c∈{0,1} for open/closed root. Initial q=c=0. All four parameter combinations are reachable. Actual spent is `(1+q,1+q,0)` at every listed boundary. Fixed identities, source/member links, admitted e, reply absent, e freeze none, no outcome, and helper intent/admission absent are preserved. Source e remains open even when c=1. No charge is hidden in a component step.

X has its own four control states, without A/B PCs or registers:

| X control | Represented fields / fixed accounting map |
| --- | --- |
| R | request unconsumed; h potential=true, ticket possible; private result unset |
| P0 | request consumed; h potential=true, ticket possible; private result unset |
| P1 | request consumed; h potential=true, ticket possible; private result=1−input.value |
| L | request consumed; actual local choice recorded; h potential=false, ticket excluded; computed result retained |

K controls are R (original entry receive), B (original choose after input consumption), L (original local mode). h ticket is possible at R/B and excluded at L. Helper is disabled at all these states. K's unused private result need not equal X's.

Exact component edges, for each q,c, are:

- X `recv(q,c): R->P0`, the original request_consume(e) occurrence; input consumed once.
- X `calc(q,c): P0->P1`, one finite private calculation, empty occurrence word; it preserves all protocol/ticket/source fields.
- X `local(q,c): P1->L`, the actual lawful choose_local bookkeeping action; it excludes only unspent h, preserves actual spending, and is not a generic private calculation.
- K `receive(q,c): R->B`, original request_consume(e).
- K `choose(q,c): B->L`, original choose_local.

Both graphs have an immediate q edge `(v,0,0)->(v,1,0)` at **every** listed control v, with exact peer_debit(q), +(1,1,0), possible→spent. Both have immediate closure edges `(v,q,0)->(v,q,1)` at every v,q, with the original caller-cancelled/root-closed facts. No q edge exists at c=1 or q=1. These are all in-scope environment events; they cannot wait for recv, calc, or local. Local member steps remain enabled after root closure, as the original admitted-member rules permit.

Fixed future demand is the sum of represented unspent tickets: `(2−q,2−q,0)` at X R/P0/P1 and K R/B; `(1−q,1−q,0)` at X/K L. Each edge's accounting map is fixed above: recv/calc preserve tickets, local/choose exclude h without a refund, q transfers its own ticket to spent with one charge, closure preserves tickets/charges. No history annotation supplies these values.

Projection P includes original request/identity, ledger, root closure/caller cancellation, e admission/request stage, original obligation links, reply/terminal/outcome facts, and h intent/admission facts. For this **declared conditional P_boundary instance**, branch/ticket/private calculation control is outside P. Thus consumed X P0/P1/L and K B/L have equal P for the same q,c. F_event/P_stage does not accept that projection/silent choice; no conclusion under those readings is asserted.

Both L states expose the same explicit local-mode port: in this fixed answer instance, one correct reply_commit(e,{id:input.id,value:computed complement}), followed by the original answer End/reply-service/publication protocol. These port transitions are not executed or suppressed. The fragment stops before them, so no complete W_N or progress claim follows.

### 6.2 Two exact supplied relations and all paths

For **every** q,c include these pairs, under the same fixed identity map:

| R_w (anticipates choice) | R_g (defers choice) |
| --- | --- |
| (X R,K R) | (X R,K R) |
| (X P0,K L) | (X P0,K B) |
| (X P1,K L) | (X P1,K B) |
| (X L,K L) | (X L,K L) |

Every row uses identical q,c on its two nodes. The exact component path certificates are:

| X edge | R_w path from its fixed K node | R_g path from its fixed K node |
| --- | --- | --- |
| recv(q,c) | [receive(q,c),choose(q,c)] | [receive(q,c)] |
| calc(q,c) | [] | [] |
| local(q,c) | [] | [choose(q,c)] |

For **each pair in either relation**, match every eligible environment q/closure edge immediately by the identical q/closure edge on the other side. The resulting pair is exactly the same table row at the updated parameters, so readiness holds in both directions. This specifies every in-scope outgoing edge and counterpart, including closed and q-spent boundaries.

Expected under the stated conditional F_fact/P_boundary instance: both supplied relations satisfy initial coverage, endpoint/component transfer, and immediate in-scope q/closure readiness. R_w fails the demand-equality strong clause at every P0/P1 pair: its X demand exceeds K demand by (1,1,0). Its receive match also performs an unmatched local exclusion after the matched input event. R_g preserves equal tickets/demand at every pair and matches exclusion exactly when X actually performs it; no reference-only intermediate exclusion occurs.

Clarification of §1's wording for this direct-event case: a directly matched original semantic event may change its own stage/ticket even when F_fact gives that event an empty **visible word**. Here X local and K choose are the same matched local-exclusion event. They are not guard-preserving private stutters. The strong alternative checks guard stability for **unmatched surrounding** reference stutters; treating the directly matched exclusion itself as such a stutter would incorrectly reject a single-event lockstep match. The supplied certificate must identify this direct matching edge. R_w's extra choose after receive has no corresponding actual X exclusion at that step and receives no such exemption.

R_g passes precisely this fragment's optional endpoint-demand and current q/closure/source-guard checks: q availability depends on the equal open/q/spent facts; closure availability on the equal root-closed bit; source e stays unfrozen; the caller is equally open/awaiting or cancelled. It is not a full ARG-1 authorization/caller-selection/progress certificate. Rejecting R_w as a strong certificate does not show no R_g exists; R_g above is the explicit alternative.

### 6.3 Optional minimal helper fork

To attack the premature pair, add at X P1 a second actual choice `helper: P1->H`, retaining possible h and enabling its original commit; then `commit_h: H->C`, which commits h with original payload/handle, changes its ticket possible→committed, and adds no charge. Add the original K fork `choose_helper(q,c): B->H` and `invoke_commit_h(q,c): H->C` (the original invoke_commit(h) event). H has e at helper Invoke; C has e awaiting the original h handle. Add the same immediate q/closure parameter edges to every new H/C boundary. h admission/service and local-mode actions remain explicit ports, outside this fragment's q/closure-only readiness scope; e still has freeze none.

Extend R_g by `(X H,K H)` and `(X C,K C)` for every q,c. Exact new paths are `[choose_helper(q,c)]` for X helper and `[invoke_commit_h(q,c)]` for X commit_h. The original local path remains `[choose(q,c)]`. Tickets/demand/source guards agree along both branches.

R_w's included `(X P1,K L)` is now unusable for this expanded component-transfer fragment: the only potential empty-word match of X helper stays at K L; pairing X H with K L would then leave its actual invoke_commit(h) without any K path. A local-excluded K L cannot re-enable h or erase that commit. This rejects the premature pair/path strategy, while the normalized supplied R_g supports both choices. It is not a NONE or contextual-failure conclusion.

## 7. Append-only authorization/admission timing matrix

This refines §4's first-order coverage to include admission timing; the original body remains preserved. Use **original J's bound diagnostic reply** r for both End statuses s∈{succeeded,failed}. Anchor at J's original post-child-selection/decision boundary, open before its diagnostic Send, with a valid original report, stable revision/generation/binding, eligible open caller, and no closure/retirement. Original J commits its diagnostic before either End, so failed End with that committed reply is lawful. Captured child obligations are those already resolved for that decision; no new work is introduced.

Let c=that once-only reply_commit(r), a=its first AuthorizeIntent result, d=its original reply admission, and t_s=J's original End/freeze(s), capturing r. Enumerate exactly these six schedules, with one repeated authorization lookup after the listed prefix:

| Schedule | Source at first a | First a | d |
| --- | --- | --- | --- |
| c;a;d;t_succeeded | open | allowed/authorized | accepted before End |
| c;a;t_succeeded;d | open | allowed/authorized | accepted after End |
| c;t_succeeded;a;d | succeeded | allowed/authorized | accepted after End |
| c;a;d;t_failed | open | allowed/authorized | accepted before End |
| c;a;t_failed;d | open | allowed/authorized | accepted after End |
| c;t_failed;a;d | failed | allowed/authorized | accepted after End |

At d the raw retained reply record has Accepted=true, Reason=recorded_reply with the original payload/binding and the saved a; the normalized protocol occurrence is accepted reply_admit. Valid schema is a stated premise. Neither t_s nor closed source alone replaces this original bound-reply admission with an invocation source-closed guard. No publication/presence is inferred merely from c or a.

For each s, also freeze the unresolved prefix c;t_s: the committed original r is captured by the freeze, first authorization remains unset, and reply admission remains undecided. Its continuation may legally be a;d. While admission is undecided, preparation remains pending on reply_admission; it must not infer missing_reply, denied authorization, or an absent final diagnostic. The same unresolved-admission distinction applies to c;a;t_s before d. Once d and t_s both exist and the stated original captured obligations are resolved, original J publication has status s, reason member_s, and the present original diagnostic, including s=failed. Publication/delivery/selection are separate ports, not extra scheduled actions here.

At each schedule's repeat lookup, require byte-for-byte/field-for-field equality with that schedule's **first saved a**, including authorization ID/intent descriptor, Source Authority snapshot, ClosureOperation, Allowed, and Reason. In early-a rows this remains the open-source snapshot after End; in late-a rows it is the actual post-End snapshot. Different schedules need not share snapshot values. Repeated lookup creates no new first disposition and is not a semantic progress self-loop.

False late source_closed denial of a valid bound reply, changed repeat records, uncommitted authorization, and changed immutable descriptors remain invalid-provenance/certificate controls. Revision/generation mismatch denials and nonreply invocation source/admission checks from §4 remain separately labeled outside-profile controls; none is an expected failure of these six stable J schedules. This matrix qualifies only the original authorization/admission clause and does not establish arbitrary R's sidecar extension, full W_N, runtime execution, or progress.

## 8. Controller scope freeze

Frozen after Task2 R2 approval at source commit `ffd00be531fbb986b86160bd6c22a87345fee14a`. The independent proposal input (including append-only refinements) is SHA-256 `396e86af553fb677890eedc9be1eaa06d1793e0939f6deadaca6bba635b8cc3e`. Only its title/status are normalized here; this section records the controlling implementation decisions.

- Implement explicit finite supplied-certificate validation and the named small clause fixtures; no existence solver, NONE result, candidate campaign, composition adapter or public runtime execution. Rejected witnesses leave other relations open. No arbitrary path/state cap is required: work is bounded by the submitted finite graph/relation/path sizes.
- Implement F1–F5, the original branch/accounting clauses and raw-state demand-map diagnostic, the §6 normalization example and helper-fork control, and all six §7 authorization/admission schedules with the stated invalid-certificate and outside-profile controls. Initial/provenance anchors must be checked against the named original rules. Well-formed unreachable nodes alone do not fail a reachable obligation.
- Toy grammars qualify the machinery only. K1 normalization uses the explicitly declared F_fact/P_boundary fragment and a fixed state/edge account; it is not W_N/W_full or a compiler/runtime witness. The optional strong checks are named endpoint demand and current-guard clauses only, never full ARG1Compatible_N. Authorization legality is a separate supplied-history check.
- Preserve ordered occurrences, multiplicity, fixed state pairs, every original outgoing edge and bilateral immediate in-scope environment service. No history splitting repairs an accounting map. Do not weaken or rewrite original K1/A/B/J/O or production code. Sharing inspected original protocol rules establishes a transcription check, not independent proof of those inherited rules.
- Use the explicit positive rank certificates and negative private cycle to qualify progress handling. Open-port fragments receive no complete progress/conformance verdict. Actual all-mode progress remains a separate premise; no uniform runtime bound is asserted. Historical baseline qualification is pinned separately, not rerun as a larger campaign.
- Freeze expected outcomes before implementation. Record RED and subsequent focused normal/race/static/module/format verification with exact sources and environment. A run interruption is incomplete evidence; witness rejection is not relation nonexistence. No search-completeness or new full-profile census will be claimed.

The general P_boundary representation implication remains OPEN. These checks validate nontrivial finite instances of the reviewed sufficient-class construction and its failure classifications; they cannot discharge that universal obligation.
