# Phase 1O — unchanged weak K1 contract and candidate domain

Status: Task 1 definition/provenance freeze, submitted for independent semantic
review. This document establishes neither ARG-2 nor candidate acceptance. The
review gate applies before any candidate search or conformance verdict uses these
definitions. Material interpretive alternatives below remain visible; choosing a
reading is not evidence that it follows uniquely from K1.

The question is whether unchanged K1 local weak conformance, with its separately
required progress, guarantees **some** relation suitable for the fixed callable
join rule. The boundary under investigation is the e/helper local component,
its fixed caller/protocol actions, and the sole q debit; J/O/S and production
semantics are held fixed.

## 1. Baseline, sources and historical status

The task baseline is merged PR25 commit
`42c837f545ae60676935c3f46e38d17d98f009df`, tree
`5cf3fc7094df1bff097b639f9141c63e04bac04e`, on
`codex/phase1o-weak-refinement`. Reviewed delivery head
`765226b88edd65536586cb06c4df3f53c85c4390` is an ancestor with the identical tree.
These identities were checked locally; this task did not fetch a moving remote.
The controller's [baseline manifest](evidence/core/baseline.json) pins the
protected tree and individual inputs. No inherited source is changed here.

Source abbreviations below denote repository paths and one-based baseline lines:

| Key | Source and decisive clauses | SHA-256 |
| --- | --- | --- |
| K | `docs/foundational/phase1i/01-substitution-contract.md`: 31–69 protocol; 71–117 states/events; 151–198 entitlement/service; 200–250 local relation/progress; 288–304 identity/hiding | `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865` |
| N | `docs/foundational/phase1n/01-contract.md`: 38–47 ownership; 177–222 positive environment; 224–253 obligations | `001c1fe5cd7be722aa75ae00a2be84ec702b51783a5595754d38ec8e6b2d8fd9` |
| L | `docs/foundational/phase1n/02-lifting.md`: 49–90 projection; 99–190 relation/bridges; 192–239 authorization/progress | `53bf70cb2aad625b4d5a3a753610ba271c84e738f557904860988350cbe40e54` |
| R1 | `docs/foundational/phase1n/evidence/reviews/reference-r1.md`: 140–203 final ARG-1; 205–224 ARG-2 | `c6f5adf78ffa283a88a31ab417700a9b72f25ea07a6cb5ee54238ae716ad3246` |

The final R1 formulation controls what this phase calls ARG-1; L:182–190 is an
earlier construction freeze, not a proof that K1 already entails intermediate
guard stability. Phase 1N's later `03-qualification.md:21–35,60–78`,
`05-decisions.md:29–34`, and `06-review.md:56–60` record bounded preservation,
conditional ARG-1, and unresolved (not disproved) ARG-2. Their results are prior
inspectable evidence, not new experiments in Task 1.

## 2. Two local domains and one composition environment

Let D_full be K:202–216's **local initial-state** product:

- id `{1,2}` × value `{0,1}` × mode `{answer,fail,absent,pending}`;
- entry depth `{1,2}`, cap `{1,2,3}`, residual I and R independently `{0,1,2,3}`,
  E0, one fixed root/generation, and q initially absent or possible;
- entry uncommitted, member/protocol fields empty, root open;
- declared service plus at most one q debit and at most one root closure.

`relation.go:21–40` spells out this product with `Closure:true`. Its size is
16×2×3×4×4×2 = 3,072 initial configurations, a direct arithmetic statement,
not a new enumeration result. This is distinct from the five historical context
products and their 3,840-configuration census (K:254–260).

D_N is the restriction with the same 16 requests, candidate entry depth2,
cap3, residual I=R3, E0, and one q debit. Closure transitions remain available
for local safety/readiness. Progress is checked separately on the open/no-closure
graph. This is exactly the profile in `CallableJoinLocalConfig`
(`callable_join_local.go:5–8`) and the safety/progress split in
`QualifyCallableJoinLocal` (`:178–219`). The latter implements the historical A/B
checks, not a solver for all weak relations.

E_N is the fixed positive **composition** environment from N:177–222 and
R1:140–171: O depth0/fixed1; J depth1; original e and S depth2; optional original
helper depth3; full I=R4, cap3, E0. J's prior debit is `(1,1,0)`; the sole
independently timed debit is original S's accepted admission, projected to q.
S may be admitted before or after e once its commitment exists. J commits e
then S and consumes S then e through the original handles. E_N supplies stable
original identity, generation, revision, grants and request/reply bindings;
open eligible callers through their required consumptions; sufficient finite
operational limits; eventual eligible local/protocol service; no cancellation,
retirement, extra inputs or pending release. Closure in D_N safety does **not**
add cancellation to E_N. Smaller representable constructor limits are outside
this positive profile.

Define W_full(X,R_w) to be K1's original relational, safety and readiness
obligations on D_full, and Progress_full(X) its separately required mode-progress
and nondivergence obligations there. Define W_N and Progress_N by restricting
those same obligations to D_N. These names separate existing requirements;
they do not strengthen or weaken K1. Complete local conformance is
`(exists R_w. W_D(X,R_w)) and Progress_D(X)` for the stated D.

The immediate target, for X in the declared candidate class and under E_N, is:

```text
(exists R_w. W_N(X, R_w)) and Progress_N(X)
    implies
(exists R_g. ARG1Compatible_N(X, R_g)).
```

A proof on D_N supports the implication from the **full** premise only after
showing restriction of both W_full and Progress_full. A negative with only
W_N/Progress_N is profile-scoped; it does not refute the full-premise implication
without original full local conformance for that X. Full local qualification
is not the five context censuses. Likewise, an input-family assertion requires
conformance over that family: one bad composed input is a contextual refutation
only when the corresponding component premise and reference/progress premises
are established.

## 3. Candidate class independent of conformance

Use A_op for finite component transition systems in the declared e/helper event
interface, with arbitrary finite local/private instruction state. They need not
be the two historical `Variant`s, have the A/B register layout, or share their
private-state-erasure relation. K:218's graph M and K:262–284's transcribed finite
graphs motivate this abstract class; it is not every Go function of type `Graph`.

A_op fixes operational ownership: candidate variation supplies e's own local
instructions and, if used, the declared single helper's local implementation.
The original caller, protocol service and q/closure rules are the fixed local
kernel. Candidate instructions change their own retained control/data and issue
ordinary interface actions; they do not directly edit caller/S/J/O state,
protocol bookkeeping, authoritative charges or original identity links. The
interface has one entry and at most the original single helper, no extra
operation, effect, sibling, message source, second request or pending release.
This is the phase's stated semantic domain (K:33–69,73–106,202–216;
R1:140–145), not a conclusion of W. Branch errors, wrong payload computations,
premature terminal choices, stuck control and divergent private control are not
excluded merely by naming this class: the conformance/progress premises must
reject them when they violate K1.

Fixed ownership is an external operational restriction, not bilateral readiness
silently added to W. An unconstrained graph that removes the caller's enabled
selection using candidate-private state, mutates S, or writes a retained
authorization result is outside A_op. If “candidate graph M” is instead read as
any labeled graph unconstrained by that ownership interface, call the domain
A_graph and keep that different claim separate. K's local relation clauses alone
do not constitute an ownership type system. A negative in A_graph without an
A_op realization is not a replacement counterexample for this task.

A_pub is the subset with an independently checked correspondence to a program
constructed through the unchanged public authoring/runtime API under this
interface. No theorem here equates A_op with A_pub. Public `Invoke` and
`AwaitInvocation` build the source's own emit/receive, with the original pending
handle and typed envelope (`author/invocations.go:94–107`,
`author/compile.go:307–329`). The runtime receive checks target run, provenance,
source, generation, obligation and operation (`internal/executor/program.go:362–397`).
The published envelope exposes status/reason/presence/response, not retained
first-authorization variables (`internal/executor/invocation_payload.go:9–25`).
The pure reducer receives its own projection/selected observation, operates on
its data, and rejects a closed source (`program.go:290–317,323–387,470–477`).
This supports the public boundary; it is not an exhaustive API-realizability proof
for arbitrary abstract graphs. A proposed public witness still needs its own
instruction mapping and operational-limit evidence.

No post-freeze component instructions are admitted: this follows from the fixed
terminal transition semantics (K:81–87,102; `program.go:302–304`), rather than
being assumed because delaying e selection ought to work. In particular, a
candidate cannot resume after its frozen outcome to request an early caller
selection. No candidate can read or mutate an added authorization variable:
that variable belongs to retained protocol service and is absent from its
admitted inputs, not a private register available to the candidate.

## 4. Projection, visible facts and atomic boundaries

At related boundaries K:218–225 explicitly requires equality of public request,
ledger, root closure, obligation links, protocol stage and observed outcome.
Identity uses one immutable renaming preserving root affiliation, generation,
causal parentage, operation/result/handle links and equality/distinctness
(K:288–304). Application id/value are literal data. Different helper names may
be renamed consistently; helper work and its charges cannot be erased.

Call P_K the required semantic projection: immutable configuration/request and
original links; spent `(I,R,E)` and closure; per-operation intent/admission and
exact rejection, request admitted/consumed, committed reply payload and
admission/refusal, local terminal/freeze, publication/delivery/selection, and
original outcome status/reason/presence/payload. These facts are named in
K:71–106,218–225,299–303. Source-open is determined by the source's terminal or
root-closure fact, not an independent private Boolean (`model.go:138–143`).
Helper possibility/choice and ticket exclusion determine future demand under
K:153–176. They must be recorded when assessing demand; whether all such
bookkeeping and coarse control fields must be *equal at every related endpoint*
is the projection interpretation P below. No arbitrary-candidate proof may
silently equate P_K with the A/B checker's stronger projection.

The finite model's concrete `publicState` erases **only** `PrivatePC` and
`PrivateRegisters` (`relation.go:15–19`): it also compares every `Operation`
field, coarse `PC`, `Choice`, `Enabled`, returned-helper values, `FailureReady`,
root/peer scratch control and `PeerTicket` (`model.go:49–81`). That is a particular
witness, not proof that the prose requires equality of every such field in every
weak relation. The actual unmatched receive and terminal facts remain semantic
facts even though a candidate's private instruction address can differ.

Visible observations are ordered occurrences, not sets or final totals. Keep
operation/parent/role/request, response id/value, explicit presence, status and
exact reason, debit deltas, and associated terminal/freeze facts. The model
record is `Observation` (`oracle.go:12–22`); terminal kind alone is insufficient:
coarse `terminal_freeze` leaves `Observation.Status` empty while successor
`Freeze` carries success/failure (`model.go:290–329`). The constructor adds a
status label then explicitly removes it in the coarse projection
(`invocation_join.go:207–211`, `callable_join_local.go:78–80`). A matcher must
retain the terminal fact, not interpret the empty coarse string as its absence.

| Event/boundary | Owner and flag in local model | Visible semantics and ordering |
| --- | --- | --- |
| `invoke_commit(e)` | Original caller; `Environment=false` | Original immutable request/handle, zero charge; precedes admission. |
| `invoke_commit(h)` | Candidate e; `false` | The one helper's original commitment/payload, zero charge; precedes its admission. |
| `admit(e/h)` | Protocol service; `true` | Exact acceptance or refusal; acceptance atomically charges `(1,1,0)`, creates member and admits request. Refusal costs zero. |
| `request_consume(e/h)` | Respective member; `false` | Exact input consumed once; cannot be hidden as no-input work. |
| `choose_local/helper` | Candidate/specification bookkeeping; `false` | Choice and h exclusion/enabling are recorded by the reference; see visibility reading F below. They are not extra runtime callbacks. |
| `outcome_consume(h)` | Candidate e, as helper's caller; `false` | Actual original selected helper envelope; precedes K1's mode action. |
| `reply_commit(e/h)` | Respective member; `false` | Once-only Send payload; commitment is not admission or response presence. |
| `reply_admit(e/h)` | Protocol service; `true` | Accepted/refused original committed reply and exact reason. May occur after End. |
| `terminal_freeze(e/h)` | Respective member; `false` | Local End and member freeze are one atomic model event, not publication. |
| `publish(e/h)` | Protocol service; `true` | Original immutable envelope, only after captured reply dispositions; successful freeze additionally waits for accepted descendants. |
| `outcome_deliver(e/h)` | Protocol service; `true` | Original observation retained in caller inbox; not selected consumption. |
| `outcome_consume(e)` | Original caller; `false` | Original exact-handle selection under caller-open/await guard; no candidate-private gate. |
| `pending_receive` | Candidate; `false` | Actual designated unmatched receive; neither terminal nor publication. |
| `peer_debit(q)` / `close_root` | Declared environment; `true` | Sole once-only debit with open/resource guard / once-only root closure retaining prior facts. |
| `private_step` | Candidate/helper; `false` | May erase only when it changes no required public fact; no hidden commit, debit, consume, terminal or outcome. |

Sources: K:92–106; `model.go:181–287,290–423`; owner extraction in
`candidates.go:17–43,56–69`; private-step implementation `:130–213`.
The `Environment` Boolean is a checker classification, **not** an ownership
certificate. In particular, `candidates.go` preserves historical e commitment
and caller selection, but this inspection establishes those A/B transcriptions,
not every abstract graph or public program. The general ownership premise is
the independently stated A_op interface and the fixed caller transition rules.

Order and multiplicity are preserved even when internal communication labels
are hidden (K:299–303). No swap of reply commitment and freeze, no duplicate
publication/consumption, no admission without commitment and no aggregate
“net zero” erasure is allowed. Different operations may interleave when their
guards allow it. K1 models publication and delivery as separate ordered events
(K:86–88,103–105); runtime `SettleInvocation` may retain both in one transaction
(`invocation_outcome.go:499–568`). That modeling expansion supplies no durable
checkpoint between them. Likewise, accepted admission plus child/request/debit
is atomic (`invocation.go:218–219`), and freeze captures the committed prefix
inside the execution transaction (`invocation_outcome.go:220–288,571–621`).
Weak matching does not create an interleaving inside a fixed atomic transition.

A candidate private stutter must preserve P_K and the selected reading's
additional public bookkeeping, as well as all its visible facts. A label saying
`private_step` is insufficient. Private PC/register/name changes, timestamps and
pure step count may be hidden (K:224–225,290–297); retained protocol facts may
not. The question whether **reference matching intermediates** have to preserve
every ARG-1 guard is kept separate from this requirement on a candidate's own
pure stutter.

## 5. Weak relation and progress: quantifiers before experiments

For a stated interpretation sigma from section 6 and D in `{D_N,D_full}`, write
W_D^sigma(X,R_w). R_w is one relation, or a coherent configuration-indexed family,
between reachable X and K1 local states under one fixed identity renaming.
It is not required to be functional. It obeys:

1. Initial coverage for every configuration in D and equality of the required
   public projection at every related pair (K:218–221).
2. For **every related reachable pair** `(x,k)` and every enabled component event
   `x --a--> x'`, there is a finite K1 path from that k to some k' with the same
   required labeled facts and `(x',k')` in R_w (K:223–225). Pure candidate-local
   instructions may match stutter only if their required facts are unchanged.
   The reference's local/helper choice is existential, not bilateral (K:244–245).
   Fixed caller events are included in the local graph's forward checks without
   thereby granting their ownership to X.
3. Every enabled **declared environment** event on either side has an immediate
   counterpart with identical event/ledger change and related successor
   (K:226–231). This includes admission, reply admission, publication, delivery,
   q debit and root closure (K:208–216). No candidate step may be required first
   to postpone service at a private instruction boundary. The clause is not
   silently extended to every `Environment=false` action.
4. K1 safety applies at all prefixes, including scarce refusals, pending and
   closure (K:14–17,71–117,151–176); wrong payload, extra action or retroactive
   demand deletion cannot repair entitlement. Equality and event matching must
   use the same original evidence throughout the relation.

For each already-related `(x,k)`, the matching path from k and related successor
k' may be chosen after seeing the candidate event. Nonfunctional R remains
allowed, but every included pair must meet all its obligations. The quantifier
is `forall related pair, forall event, exists path from k`, not
`exists preselected path, forall events`. All successors used remain in the
same relation and satisfy future obligations, including adverse environment
choices. Separate favorable certificates for completed traces do not establish
simulation. K:290's per-trace correspondence renaming is not permission to rename
one retained operation differently after each branch.

Progress_D(X) is separately K:232–242 under K:192–198's service assumptions.
For every open/no-closure maximal path with the applicable initial entitlement
and eventual service, input and any chosen original helper are consumed, then:
answer has its correct present succeeded envelope; fail has
`failed/member_failed/absent`; absent has `failed/missing_reply/absent` after
local success; pending reaches the designated unmatched receive after required
consumptions, with no reply/freeze/publication. A refused helper yields the
specified candidate failure; refused entry yields its exact rejected envelope.
Scarce paths must reach corresponding refusal/failure when finite required
service occurs. No other stuck state or additional action is accepted.
There is no closure progress claim. Nondivergence requires a well-founded rank
or a sound SCC/other argument excluding internal stuttering cycles outside the
terminal/designated-pending boundary. Unfairly withheld environment service is
not component silence. Finite cutoffs or merely allowed finite traces do not
prove Progress_D.

In D_N all mode paths have sufficient e/helper/q capacity and depth; eventual
eligible service includes the original local caller and sole q action. The
historical open qualifier's complete maxima retain q spent, helper selected
when used, and exact e delivery/selection or pending boundary
(`callable_join_local.go:104–165`). This is the reference for the profile's
observable completion, not a demand that an arbitrary X share A/B PCs or the
finite rank cap. Caller completion after e publication follows the fixed
caller/service rule; it must not be smuggled in as a new candidate capability.

Keep three propositions distinct:

1. A supplied R_w does not extend to an ARG-1 witness.
2. X satisfies the weak premise but admits **no** ARG-1-compatible relation.
3. Such X breaks K_join inside the fixed J under E_N.

Only (3), with independently established J[K1] and progress premises, is a
contextual behavioral counterexample. Failing the private-state-erasure checker
or one guessed matching strategy establishes neither (2) nor (3).

## 6. Preserved interpretations of unchanged wording

The common obligations above do not settle every weak-path/projection choice.
Use the following explicit names rather than quietly changing meanings during
an argument. When a result depends on a reading, display the superscript in
W_D^sigma; unqualified W_D means unchanged K1 with these ambiguities outstanding.
No candidate may be accepted/rejected under an unresolved choice without the
independent semantic review recording its scope.

| Issue | Plausible reading and source | Different consequence / unresolved obligation |
| --- | --- | --- |
| F — which reference steps are silent? | **F_event:** the full model semantic event word is visible except genuine public-fact-preserving private instructions. **F_fact:** K:223–225 names commit/admission/payload/charge/terminal facts and allows private control differences; a contract-choice/control step can be unlabelled when it emits none of those named facts, while related endpoints still agree in protocol/outcomes. K:71–106 and 299–303 constrain either reading. | F_fact can admit intermediate bookkeeping not permitted by F_event. In particular, `choose_local` changes possible helper demand and `choose_helper` enables it. Calling either a silent model choice is not a proof that it preserves the constructor's demand/guard dependencies. Both readings preserve ordered occurrences and multiplicity of mandatory protocol facts, selected outcomes and debits; neither licenses bags of facts, net totals or hidden extra protocol operations. |
| P — private control versus semantic stage | **P_stage:** all coarse protocol/control, choice/ticket/demand and caller-readiness stages represented by `publicState` are public; actual implementation PC/registers alone are private. **P_boundary:** K:218–220 mandates the named protocol/outcome stages, but permits local control differences not yet changing them; it does not explicitly equate every `Enabled`, `FailureReady`, scratch root PC or branch-bookkeeping field. | Historical A/B use P_stage. An argument for arbitrary X must identify which extra equality follows from operational histories, which is required by “protocol stage,” and which is additional. Endpoint demand/guard equality and intermediate stability cannot be asserted solely because `publicState` compares them. Source terminal, selected outcomes and exact charge facts remain mandatory under both. |
| O — event ownership/readiness | **O_op:** M is a component in fixed operational ownership as in A_op. **O_graph:** the phrase “candidate component graph M” is treated as an unconstrained labeled graph, including extra guards on caller-owned `Environment=false` events. The implementation flag only marks the historical checker partition. | O_op is the declared Phase 1O domain; O_graph is a separate broader interpretation to record if a purported negative needs it. Bilateral readiness for caller actions is not literally K:226's environment clause. Under O_op it must be derived from fixed caller guards plus related public facts; under O_graph it may fail because the ownership interface was removed. |

F_event/P_stage are evidence-backed sufficient readings for the historical
checkers, not a silent replacement for the weaker prose readings. Conversely,
F_fact/P_boundary cannot erase mandatory protocol stage/outcome facts merely
to manufacture a counterexample. The finite K1 transition rules, identity,
once-only discipline and atomic boundaries stay fixed in every reading.
The task has not resolved these alternatives or proved they yield different
candidate acceptance sets; their consequences are precise proof obligations,
not experimental results. A future clarification making one reading normative
would be a separately labeled contract change, not an ARG-2 proof for unchanged K1.

## 7. ARG1Compatible_N and premise provenance

ARG1Compatible_N(X,R_g) means all of R1:140–171's relation/interface/environment
requirements on the physical depth2/cap3 residual I=R3 domain under E_N:
initial pairing; equality under fixed renaming of request, original links,
protocol/outcomes/presence, source terminal/closure, ledger, depth and future
demand; event transfer and bilateral declared environment readiness; fixed
caller commitment/selection readiness; equal split-authorization predicates and
retained first dispositions; initial/prefix entitlement. It permits **either**
single matching reference events/public-field-preserving private stutters **or**
the reviewed finite weak-path alternative with no additional nonmatching protocol
event and intermediate stutters preserving every listed guard. It is not defined
as the single-event checker's success. Its separate progress requirement is
Progress_N with the same service/nondivergence assumptions; including it in
ARG1Compatible_N repeats a premise, not a derivation of liveness from R_g.

For weak intermediates, “preserve guards” means that each stutter preserves the
facts on which the relevant current constructor/protocol guard depends; it does
not prohibit the actual matched event from changing its own stage. Equal
source/binding conditions and equal retained authorization sidecars must extend
the relation coherently over all reachable service orders, not one favorable
historical authorization schedule. ARG-2 asks for **some** suitable R_g; R_g may
use alternative related reference states/paths and need not contain a supplied R_w.

In the table, **derived** refers to the cited unchanged clause or fixed
operational rule, not to an already completed ARG-2 lemma. **Additional** means
ARG-1 explicitly requires it and its derivation from weak K1 is still open.

| ARG-1 requirement | Original clause or operational rule | Status | Evidence or precise gap |
| --- | --- | --- | --- |
| Initial pair over every aligned input | K:202–221; `CallableJoinLocalConfig:5–8` | Derived by domain restriction | D_N is a subset of D_full; restrict one coherent family, not just selected successful traces. |
| Original identity, request and handles | K:33–39,74,83–84,218–221,290–294 | Derived | Immutable renaming preserves provenance/root/generation/operation-result links; application id/value are unchanged data. Raw authentication remains a separate correspondence obligation. |
| Protocol stage, payload, outcome presence and exact reason | K:78–106,218–225,299–303 | Derived; P ambiguity for extra control fields | Terminal/publish/deliver/consume remain distinct; empty raw terminal status is completed from Freeze. No general all-field projection lemma supplied. |
| Source terminal and closure | K:81–87,94–106; `model.go:138–143` | Derived for named state facts | Source-open is a function of original source/Freeze/Closed. General weak-intermediate stability remains part of the bridge. |
| Ordered events, multiplicity and atomicity | K:84–106,299–303; `model.go:181–423` | Derived for mandatory facts; F ambiguity for bookkeeping | No reordering, extra protocol event or net-debit erasure; classify choice/control stutters explicitly. |
| Same single-event match **or guard-stable weak alternative** | K:223–225; R1:151–155 | Additional / unresolved | Weak matching and pure candidate stutter clauses do not alone prove every constructor dependency stable at every reference intermediate. |
| Bilateral declared environment readiness | K:208–216,226–231 | Derived | Immediate counterpart and related successor, including q/closure and every private instruction boundary. |
| Caller-owned e commitment and selection readiness | K:94,105,122–124; `model.go:230–242,375–423`; R1:157–158 | Fixed ownership derived; relation transfer unresolved | `Environment=false` does not license candidate control. Equality of the relevant caller guard must still be shown for the selected weak projection. |
| No post-freeze e work requiring early selection | K:81–87,102–105; `program.go:302–304`; R1:145 | Derived from fixed operational termination | This disallows an impossible resumption, not a guard-stability premise inserted into W. |
| Actual ledger equality, exact debit and no refund | K:151–176,218–227 | Derived | Commit zero; admission transfers demand to one `(1,1,0)` debit; refusal zero; matched X/K1 charges equal. Unequal A/B charges are allowed. |
| Physical depths and prior J offset | N:179–205; `callable_join_local.go:5–39` | Independent E_N assumption plus derived mapping | J contributes `(1,1,0)` beforehand; only accepted original S maps to q. Refused S cannot be erased as a debit. |
| Future demand / prefix entitlement | K:153–176,213–215 | Entitlement derived; equality across arbitrary weak matches ambiguous | Initial residual demand ≤ `(3,3,0)`, full ≤ `(4,4,0)` under E_N. Choice/exclusion bookkeeping must not be equated with ARG-1 intermediate preservation without the P/F bridge. |
| Stable generation/revision/grants and bindings; eligible open callers | N:195–199; R1:159–171 | Independent E_N assumption | These are not new weak-relation guarantees. Caller remains open through original selections; no retirement/cancellation in composition. |
| Equal first authorization predicates/dispositions | `messages.go:103–177`; `invocation.go:123–140,218–224,296–312`; R1:159–161 | Operational rule derived; extension to arbitrary R_w additional | First outcome is retained and reused. Need a relation extension covering source-open/identity/binding and all service orders; coarse K1 lacks authorization sidecars. |
| Original bound reply may first authorize after End | `messages.go:154–165`; `invocation.go:123–140`; `invocation_outcome.go:164–184,354–373,426–440,571–621` | Derived operational rule with E_N binding premises | Revision/generation checked before reply exception. Do not require early authorization or transfer this exception to invocations. Failed J may retain a present diagnostic. |
| No candidate observation/mutation of authorization state | Public invocation outcome schema and own-projection reducer, section 3 | Fixed interface restriction; public realization separately checked | Excludes invented authorization side channels; not inferred from safety simulation or the Boolean Environment flag. |
| All-mode progress and nondivergence | K:192–198,232–242; `callable_join_local.go:104–219` | Separate existing obligation | Neither relation safety nor a bounded successful search proves progress for arbitrary private control. |
| S finite service, no release and sufficient limits | K:60–61,128–133,192–198; N:195–222 | Independent E_N assumption / fixed S behavior | No q priority or extra debit; pending stays unmatched. Constructor progress transfer must preserve service assumptions. |

Production paths in the last rows are under `internal/executor/`; named local
model functions are under `research/substitution/`. References assert inspected
source rules, not compiler proofs or universal runtime qualification.

## 8. Gate and remaining obligations

Independent semantic review must approve the exact domain, ownership split,
projection/visibility readings, existential quantifiers and progress separation
before they are used to judge a candidate. Review may approve a deliberately
qualified statement while leaving K1's interpretive alternatives unresolved;
that does not make either stronger reading the unchanged contract by fiat.

Task 2 must discharge or precisely classify the event/intermediate-state,
caller-readiness and authorization-extension bridges for the stated reading(s).
It must distinguish an unusable supplied witness from nonexistence of any
suitable witness and from a contextual K_join failure. A finite search must
state what it quantifies over; a publicly executed negative requires separate
A_pub realization evidence. No candidate search, acceptance run, proof of ARG-2,
or public witness execution occurred here.

The manuscript currently states conditional ARG-1 and open ARG-2 in Markdown
line 562 and LaTeX line 381. Task 1 changes neither disposition, so these source
checks find no immediate status contradiction. Phase-end reconciliation of any
new conclusion, hypotheses and interpretation scope remains the controller's
obligation; no manuscript file or reference entry is edited in this task.
