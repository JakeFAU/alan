# C_inv contract and independent finite reference — review freeze

Status: independently derived after the construction-feasibility gate, before
product qualification or inspection of the new parent artifacts/runtime outputs.
This document and `research/substitution/invocation_join*.go` define the review
candidate. No C_inv census, reference-premise result, replacement result or new
artifact correspondence is claimed yet. Exact review identities are retained in
`.superpowers/sdd/00-plan-phase1l/task-2a-freeze.json`.

## Claim, domain and observation

Investigate the bounded implication

`A <= K1 and B <= K1 and C_inv[K1] satisfies G`
`implies C_inv[A] satisfies G and C_inv[B] satisfies G`.

The reference premise is a separate obligation, checked first. Reference failure
is a context-premise failure, not a substitution counterexample. Local refinement
is independently checked component behavior, never defined as passing C_inv.

The positive domain is exactly four requests: id in {1,2}, value in {0,1}, mode
answer; common I=R3, depth cap2, E0; one fixed parent; original singleton A/B/S and
B's optional singleton helper. Candidate and S are direct depth1 children and the
helper is depth2. Physical capacity1 in concrete checks is distinct from these
cumulative dynamic admission limits. There is no static reporting peer, 32-Set
prefix constraint, wrapper, report/ACK exchange, effect or additional observer run.

G requires both original invocations to publish and their original outcomes to be
delivered and actually consumed; both statuses succeeded and responses present;
candidate response id=request.id/value=1-request.value; S response
id=request.id/value0; parent success; matching root/aggregate success publication;
and resource compliance. Parent result is retained state/status plus root
publication. It has no ordinary outgoing result-message obligation.

Stable logical generations, grants, source revisions, valid original template and
reply bindings, valid finite-domain schemas, no retirement/cancellation, eventual
eligible local/protocol service, and adequate operational step/dispatch/byte limits
are explicit assumptions. Finite physical interruptions may resume original work;
infinite crashes, infinite scheduler/retry stutter and watchdog exhaustion do not
supply progress. Exact concrete operational limits must be justified against the
compiled artifact mapping before concrete qualification; this abstract rank is
not a measured runtime instruction budget.

## Parent and component behavior

One parent definition is parameterized only by candidate template. It commits the
candidate invocation e, then S invocation s, before its first await or decision.
Their admissions/execution may interleave with those commitments. The e envelope
is exactly the original {id,value,mode}; s receives exactly original {id,value},
with no mode field and no substitution of a candidate response into S's request.

The parent awaits e and then s using their distinct original retained handles.
Known refusal, failure or absence in e still leads to s's required await before
the normal final decision. A pending e keeps the parent at e's await even if s
has completed; a finished sibling cannot discharge an outstanding handle.
Once both original envelopes are selected, status and presence guard all payload
reads. The parent succeeds exactly on G's two result predicates, otherwise fails.
Decision, local terminal End and root publication remain distinct boundaries.
A root publication requires no reply; the root is not an invoked component.

K1 independently chooses local work or one helper after consuming its request,
in each mode. Local choice excludes the unused helper ticket. Helper choice
commits h with the same id/value and mode answer, then consumes its original
outcome before the candidate mode action. A refused/failed helper causes candidate
failure without reply. Answer commits the correct complement reply and succeeds;
fail fails without reply; absent succeeds locally without reply and therefore
publishes failed/missing_reply; pending remains at its unmatched receive without
publication. S consumes its request, replies with original id/value0 and succeeds.
The original A13/B16/helper7 private instruction transcriptions remain unchanged.

The abstract parent's guarded decision collapses only its finite pure local
checks after both original consumptions. Later mapping must cover every compiled
parent instruction and prove these extra private steps do not change a relevant
protocol fact or create an internal cycle; failure to do so blocks correspondence.

## Protocol and lifetime transitions

Every progressing event has an explicit transition:

| Event | Required predecessor / retained effect |
| --- | --- |
| invoke_commit | Open source at its invoke; retain original request/handle, zero debit |
| invoke_authorize | Original commitment; retain immutable first source disposition |
| admit | Original authorization; current source/scope, declaration/template/schema/depth and I/R guards; accept once with (1,1,0), or retain exact refusal and zero debit |
| request_consume | Accepted original member/request; consume once |
| choose_local/helper | K1 ghost branch/demand choice, not a runtime instruction |
| reply_commit | Original open requested member's Send; retain actual payload, no debit |
| reply_authorize | Original reply commitment; first disposition separate from admission |
| reply_admit | Original allowed disposition and valid reply schema/binding; preserve committed payload |
| terminal_freeze | Member's actual local succeeded/failed End; committed replies remain obligations |
| publish | Rejection, or terminal member with captured replies decided; successful member also joins admitted descendants; retain original outcome |
| outcome_deliver | Publication exists; retain its original invocation observation |
| outcome_consume | Open caller at that original handle; select the delivered envelope once |
| parent_decide | Both e/s consumed; strict guarded result predicate |
| parent_terminal | Actual parent End agrees with its retained decision |
| root_publish | Parent terminal and required joins resolved; agree with parent outcome |

`compositionReplyBinding` requires retained protocol match, message kind,
template entry member, bound caller subject/generation, and entry-port obligation.
Only that binding-qualified reply bypasses retired/non-open-source denial at first
source authorization. Revision/generation mismatch still denies authorization.
Thus a correct original reply may first authorize and admit after its member End.
The special reply admission path records authorization and validates the entry
reply schema; it is not ordinary-message receiver-open/route admission. Stable
valid bindings/schemas make its acceptance deterministic, not an assumed successful
schedule. Both before-End and after-End service orders remain enabled.

Invocation authorization does not gain the reply exception. Final invocation
admission still checks current source openness and shared-root/ancestor scope.
Root local termination fences new admissions. Here the parent cannot terminate
before both original outcomes are consumed, and the candidate cannot terminate
before its chosen helper is resolved. Those obligations are context/component
properties to check, not a global deletion of lifetime guards.

## Demand and local evidence reuse

Initial demand is {e,h,s}. Commit transfers no resource. Accepted singleton
admission spends exactly one I and R; refusal spends neither; no refund exists.
The optional h disappears only through local choice or an unrealizable rejected
entry subtree. Initial entitlement is answer plus enough resources/depth for all
three possible admissions. At entitled prefixes, spent plus possible/committed
future demand must remain within I/R3. Expected successful dynamic spending is
A+S2 versus B+helper+S3; both have fixed1/effects0. Different permitted debits stay
observable; exact-debit equivalence is not asserted.

Reuse the original Phase1I local subset for each of four answer requests:
Local=true, Shape=local, EntryDepth1, Depth2, I=R3, Peer=true. S's sole accepted
admission projects to the old arbitrary-timing external q debit (1,1,0), and its
other actions stutter for this local component relation. The old relation checks
bilateral environment readiness at every private instruction boundary; its
separate Closure=false progress check is the relevant positive branch. This
projection covers debit interference, not S's payload/lifetime, parent joins or
root publication. The new product models actual S admission/refusal, not a q
action that disappears when resources are exhausted.

Original source identities and A/B/helper mapping are recorded in the preparation
note and Phase1I final-source manifest. Historical local relation counts remain
historical until targeted current-source checks bind reuse. Phase1I's failed
parallel-context premise and all I/J/K artifacts/evidence remain intact.

## Independent checking interfaces

`JoinConfig` contains Request,I,R,Depth. `JoinEvent` carries the original
normalized `Observation`, plus SourceOpen/SourceRetired/RevisionMatches/
GenerationMatches/ReplyBound for first authorization facts. ReplyBound asserts
all binding conditions above, independently authenticated from raw records.
A single consistent identity renaming maps original candidate/helper/S operations
to e/h/s and their parent to root/e. Every operation fact repeats the same role,
parent and request envelope. Accepted admission/reply reason is normalized to
empty; refused reasons and outcome reasons are retained exactly. Original rows
and unnormalized reasons remain available in the raw evidence.

Three results remain separate:

1. Raw history validation links original intents, actual handles, admissions,
   bindings, authorization snapshots, publications, delivery observations,
   predecessor receive guards, selected history and consumed sequence before
   normalization. Reuse `validateRetained` with C_inv-specific supplemental checks;
   this happens outside the pure model. Never fabricate an operation identity
   from an expected sequence tuple. Validate incomplete-return paths too.
2. `CheckJoinComponent` applies unchanged `CheckTrace` to candidate/helper facts
   and separately checks mode/helper-before-reply/End discipline. Passing prefixes
   do not establish the independently checked local relation or progress.
3. `CheckJoinPrefix` validates protocol/identity/context ordering and honest
   decisions without requiring correct component application payloads.
   `CheckJoinGuarantee` additionally requires complete positive G evidence.
   Missing publication/selection/completion never becomes success.

A genuine wrong-id reply can have truthful raw records and an honestly failed
parent: prefix acceptance, component rejection and G failure must all be visible.
An incomplete or pending honest prefix may pass prefix checking while G fails.
Out-of-domain/unstable-binding histories are outside this finite profile, not
silently certified. Admitted reply payload id/value remain available unchanged.

## Finite representation, rank and enumeration

`JoinState` wraps the old comparable component state, replacing all old parent
control/decision fields. The unused wrapper and static peer remain fixed; no old
root termination or sequential S input survives. Added fields are parent stage
0..6, decision/status in {empty,succeeded,failed}, root-publication bit, and two
authorization dispositions per possible operation in {0,1,2}.

Each of e/h/s has fixed role/source/depth/request, ticket0..3, admission0..2,
request stage0..2, protocol PC0..8, choice0..2, reply0..3, freeze0..2 and seven
Boolean fields. Its three retained id/value pairs range over 0..2/0..1; publication
status has four values; outcome reason ten; admission reason seven; reply reason
is empty in this stable valid-reply profile. Private candidate/helper PCs are
0..15/0..6 and their 16 registers range over 0..2. Ledger I/R is 0..3, E is zero.
These domains yield the conservative model-state cardinality upper bound

`16*7*9*2*3^6*16*7*3^16 * (4*3*3*9*3*4*3*2^7*6^3*4*10*7)^3`.

It is a loose Cartesian bound, not a measured reachable-state count. The prefix
monitor adds a finite product of per-operation Boolean stages, original bounded
outcome records, bounded ledger and parent status. Its state is explicitly part
of the explorer key; `States` reports model×monitor states and `ModelStates`
reports distinct model states. A conservative prefix-state bound is
`16*9*2 * (2^14*7*6*3*481)^3`: fourteen Boolean flags per operation,
seven rejection reasons, six reply id/value pairs, three terminal statuses,
and at most481 empty-or-bounded published envelopes. Multiplying the two bounds
bounds the exploration key space. This avoids relying on one representative trace
for path-dependent prefix checks. The component monitor's future-relevant sets
(commits/admissions/replies/publications/consumptions), helper result and reply
presence are all contained in that same prefix state; violations are checked
before every merge.

Intrinsic rank is `100*stage + candidatePrivatePC + helperPrivatePC`. Stage sums
I+R+E, parent stage, root-publication bit, all six authorization dispositions and,
for each operation, ticket+admission+request-stage+protocol-PC+choice+reply+freeze
plus Enabled/FailureReady/Published/Delivered/Consumed bits. Each operation's
stage sum is at most27. Therefore the exact declared conservative cap is

`100*(3*27 + 6 + 6 + 6*2 + 1) + 15 + 6 = 10621`.

Initial rank is100. Every protocol/parent/authorization edge strictly increases
at least one stage and never decreases another; original private branch/Jump
edges strictly advance their own private PC, with no register-only edge whose
PC stays fixed. Payloads, trace length and obsolete root PCs contribute nothing.
The factor100 exceeds the entire private-PC range21; no private decrease is
required by the inspected A/B/helper transcription. Rank therefore bounds a
progressing path by10521 edges under these transition definitions. This is a
source argument to be challenged and checked on every generated edge; it is not
an already-executed census result.

`ExploreJoin` performs full BFS without a depth/time cutoff or failure pruning,
checks every successor before deduplication, retains every maximal model×monitor
state and a shortest trace per failure code, and records states/edges/revisits,
refusals, terminal/pending/stuck maxima, rank/domain/prefix/component/G failures.
A failed rank makes cycle status unresolved (-1), never acyclic by declaration.
Retries, idle scheduling and same-state physical restart are stutters excluded
from the progressing graph; eventual service excludes an infinite all-stutter
suffix. No count is a probability or an exhaustive concrete-runtime result.

## Controls and remaining gates

I=R2 is insufficient entitlement: any real competing admission may be refused,
with authentic zero debit and retained reason. Fail/absent/pending are policy
controls outside positive G. Their outcomes must not invent payloads or skip the
other required join. A one-consumed checkpoint must retain an open parent and
classify the other original outcome as unproduced, published, or retained but
unconsumed. Recovery resumes original handles/artifacts/generations and spending.

After independent review freezes this package: qualify C_inv[K1] first, then the
unchanged A/B bindings; independently map the actual compiled parent and original
A/B/helper/S; justify operational bounds; run selected concrete schedules and
recovery with separate labels from automatic samples and model enumeration.
General congruence, arbitrary contexts, compiler correctness, effectful recovery,
distributed semantics and universal equivalence remain open.
