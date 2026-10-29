# Phase 1J retained join contract J1 — revision 1

Status: proposed freeze for independent review; no implementation or qualification
is asserted by this document. The reviewed digest and any correction history belong
in `03-review.md`. K1 and its historical completion entitlement remain unchanged.

## Question and causal intervention

Can an ordinary retained join discipline keep the root open until the required
static peer has consumed S's actual outcome, thereby prevent the Phase 1I
root-success admission fence within the declared finite scope?

C0 is the preserved Phase 1I delayed parallel context: 32 ordinary peer `Set`s,
then its original S `Invoke` and `AwaitInvocation`, concurrent with the root's
candidate invocation. C_join replaces only root decision control with retained
joining, the peer's post-await decision with report/termination, and the minimum
static ports/grants/report schema needed to connect them. The root candidate
Invoke, handle, AwaitInvocation and actual response bindings stay intact.

A/B source, compiled programs/templates, helper structure and S bytes are fixed.
S consumes `{id,value}`, replies `{id:request.id,value:0}`, then succeeds. Its
constant zero is distinct from the candidate complement. Root/peer remain static
members of one resource root; neither becomes an invoked child. The original
C0 constructor/artifacts and negative evidence stay available. New root/peer
identities, exact source/compiled diffs and hashes must be recorded. No production
executor/compiler/schema, closure policy, dependency or accounting change is
permitted. Capacity increases are controls, never repairs of the primary pair.

## Public protocol and correlation

Use existing `author.Invoke`, `AwaitInvocation`, `Send`, `Receive`, `IfEqual`,
`Set`, `Success` and `Failure`. Static public ports are both named `join`, with
obligation `phase1j_s_join`: root receives from `peer`, peer sends to `root`.
`author.Application.Provision` supplies the matching ordinary message grant.

The report record is:

| Field | Schema and meaning |
|---|---|
| `request_id` | integer; original application request id |
| `s_operation` | string, MaxBytes 256; original retained S invocation handle |
| `status` | string, MaxBytes 20, enum succeeded/failed/cancelled/abandoned/rejected |
| `reason` | string, MaxBytes 4096; exact consumed reason |
| `response_present` | boolean; exact consumed presence flag |
| `response_id`, `response_value` | integer; actual bound fields when present; explicit zero fillers otherwise, carrying no response claim |

The schema's status/reason bounds mirror the existing invocation envelope. The
operation bound is specific to the static `peer` generation-1 handle in this
finite program; verify actual serialized length during artifact qualification.
The compiler permits serializing a retained operation into a string payload;
this loses its static declaration proof, not its string identity.

Runtime receive matching checks ordinary-message provenance, source `peer`,
source generation 1, obligation, destination root generation 1 and the static
permission. Application code checks `request_id` against the root's retained
input; integer ranges are environmental restrictions, not schema Min/Max.
There is exactly one request per deployment/generation. Generation changes,
multiple outstanding requests, changing peers and reused obligations are outside
J1. A stale-generation report cannot match this receive. A schema-valid
wrong-request report is consumed as invalid correlation but does not discharge
S: the root enters a further ordinary receive with no supplied next report and
cannot terminate in the declared one-report profile. This is application
rejection, distinct from runtime admission refusal and oracle rejection.

The root trusts the designated peer's report. It cannot independently authenticate
`s_operation` as S's consumed receipt through this ordinary messaging interface.
The oracle must connect that handle to the original S operation and selected
outcome. Premature reporting can fool the root and must still fail the unchanged
J1 guarantee checker as a peer-contract violation. Authenticated communication
is not a runtime receipt proving the assertion true.

## Strict retained outcome discipline

1. Before any root termination, the predeclared peer obligation exists, even
   before S's Invoke has committed. Root invokes the candidate once and consumes
   its actual outcome using its original handle.
2. Root retains candidate status/reason/presence/response and executes mandatory
   join Receive even if the candidate failed, was refused or lacked a response.
   Early report delivery remains in the ordinary retained inbox until this point.
3. Peer executes the unchanged 32 Sets, original S Invoke and original matching
   AwaitInvocation. After consumption it sends exactly one report of those bound
   facts for every known terminal/refused outcome. A pending S produces no report.
   AwaitInvocation deletes response bindings on absence, so branch on presence
   after awaiting: send actual bound fields when present, explicit zero literals
   otherwise. Initializing the same bindings before await is insufficient.
4. After sending, the peer performs only pure decision/terminal instructions,
   with no future scope-dependent admission or reservation. Peer executes local
   Success regardless of S outcome: this means the reporting program finished,
   not that S succeeded or delivery is guaranteed. Peer Failure would freeze the
   shared scope while candidate work may remain and is excluded here. Sending
   discharges required S work and future protected demand, not peer termination.
   Its eventual local terminal
   state and the aggregate root outcome must be checked separately.
5. After consuming the correlated report, root succeeds iff candidate status is
   succeeded, presence is true, id is original, value is `1-request_value`, and
   reported S status is succeeded, presence is true, id is original, value zero.
   Otherwise root fails after joining. Exact reasons are retained regardless of
   the decision. Failure/absence is never interpreted as pending or success.
6. A pending candidate holds root at its original AwaitInvocation even if S has
   reported. A pending S holds root at join even if the candidate failed. Neither
   unknown nor pending grants permission to retry. Authorized external cancellation
   separately interrupts the discipline and is excluded from positive completion.

No second completion or new work follows duplicate transport delivery or replay:
original logical message identity and single receive consumption remain linked;
root has no repeat decision/invoke path. Check duplicate delivery/idempotent
retention and replay as apparatus controls; do not claim arbitrary malicious
extra-report handling or ignore a genuinely distinct extra peer Send.

## Finite profiles, obligations and scheduling

The primary census is exactly eight configurations: application id in {1,2},
value in {0,1}, mode answer, I=R=3, depth in {2,3}, E=0. Fixed provisioning is
root+static peer=2 runs outside dynamic created-run spending. Check these same
eight products separately for K1 (local or one helper), frozen A (local), and
frozen B (one helper). No general composition grammar is quantified.

Positive assumptions: truthful conforming peer; unchanged K1 candidate contract;
original S contract; provisioned grants remain valid; no external cancellation;
message/outcome dispatch attempts, enabled local execution and admission
processing eventually receive service. Fair service permits a refused message;
it does not guarantee authorization or delivery. Physical runtime capacity is one. Fair
service is not an assumption that root remains open or that completion occurs.

Resource obligation at every relevant prefix is spent plus outstanding maximum
contractual demand <= each configured limit. Initially demand includes candidate,
optional K1/B helper and S, whether committed or not. Acceptance transfers one
(I,R) ticket into spent; actual branch exclusion removes only unspent optional
helper demand. No report debit is an invocation or external effect. Expected
successful spending is A=(2,2,0), B=(3,3,0); refusals preserve their actual smaller
charges. Depth is checked at each prospective admission (candidate/S depth1,
helper depth2), not treated as an additive ticket. Capacity controls I2/R3 and
I3/R2 do not satisfy the positive B envelope and must reveal actual refusals.

Lifetime obligation: while required participants retain future work needing
this root's admission/reservation authority, root must not voluntarily execute
Success or Failure that fences that work. This is a property to check, never an
admissibility assumption. Root normal terminal transitions remain in the model,
guarded by actual program control. Local root status, aggregate publication,
explicit cancellation and admission-open are separate fields. In particular,
Phase 1I `Closed=false` does not imply admission-open: runtime
`compositionScopeClosed` closes admission whenever root local status is non-open.

The independent finite model retains operation identities, actual consumed
payloads and finite per-operation stages: commit, admission/refusal, request
consumption, optional helper choice/work, response commit/admission, terminal
freeze, publication, delivery and consumption. Add separate report commitment,
source authorization (allowed/refused), message admission/delivery, root consumption,
and peer/root terminal/aggregate boundaries.
Represent all orders, including report before candidate outcome; do not merge
report Send with peer termination or outcome publication with consumption.
Abstract the 32 pure Sets only with an explicit finite stutter correspondence;
retain root terminal eligibility at the same semantic points.

Enumerate all reachable modeled states and edges from each initial state without
success-based pruning. Report exact states, transitions, maximal-state categories,
cycles, cutoffs and rank failures; distinguish maximal states from complete path
counts. Establish a strictly decreasing finite remaining-event rank (or explicit
cycle/fairness analysis) including optional-branch resolution and every separate
report/terminal boundary. Zero cutoffs plus acyclicity and all maximal positive
states complete support the finite all-schedule claim; a timeout alone does not.
Pending, bad correlation and cancellation controls have separate hypotheses.


### Ordinary-message authorization is an open failure boundary

The static peer report is an ordinary message, not an invocation reply.
`internal/executor/messages.go` rejects first source authorization after the
source run is terminal (`source_closed`); the composition-reply exemption does
not cover this port. Consequently Send commitment then peer Success then first
report authorization is a permitted fair interleaving to test. It may leave root
open forever without report while required S work is already settled. Model the
exact source refusal and resulting pending root; do not assume Send commits a
durable authorization, delete this schedule, or strengthen eventual service into
successful delivery. A concrete Store.Run sample may authorize before terminal;
that scheduler-specific observation cannot establish the all-schedule claim.
This is a potential conclusive negative for simple J1, not permission to add an
acknowledgment handshake or change runtime policy in this phase.

## Independent oracle and raw evidence mapping

`research/join` must not import author/runtime/executor or derive expected answers
from running A/B. Reuse reviewed pure K1 primitives where sound, explicitly retain
the runtime root-terminal fence, and justify actual A/B/root/peer/S instruction
mappings against frozen artifacts. The reference oracle originates from J1/K1.
Do not substitute a B-derived surrogate for an independently mapped program.

The test-only raw adapter may reuse `validateRetained` and row types from
`internal/substitutiontest/evidence_test.go`, extending checks additively for
ordinary join messages. It must fail closed before lossy normalization:

| Evidence | Required linkage |
|---|---|
| programs, runs, roots, static members, bindings | exact artifacts, original run/generation, scope and fixed topology |
| intents, admission decisions, publications/outcome deliveries | original candidate/helper/S operations, reasons, members, charges |
| history.Selected + observations + pre-receive program/data | exact observation selected by AwaitInvocation/Receive, matching original handle/source/generation/obligation; actual consumed payload, not latest publication |
| peer await history then peer Send intent/history | S consumption precedes report commitment in peer sequence; copied payload/handle matches that consumed evidence |
| message admission/observation/ack records | original peer Send identity, source/target/generation, actual report payload and delivery; no invented success from mere intent commitment |
| root receive history and later terminal/freeze | exact report selected, request check, retained candidate data, J1 decision and no premature root terminal |
| freezes, closures, lifecycle results, root publications, resource facts | local vs aggregate status; explicit authorized interruption; admission-open predicate and exact spending |

Use causal per-run sequences and retained links, not an invented total ordering of
independent histories. Preserve operation/message identities under one consistent
mapping. For absent responses require explicit zero report fillers and deleted response
bindings after AwaitInvocation, without promoting fillers into present data. Attack changed mapping with
early-report, wrong-request, wrong selected payload/identity, stale generation,
wrong S handle and report/consumption-order mutations. Classify invalid apparatus,
invalid peer contract, and conforming-context failure separately.

## Decisive concrete and interruption evidence

Reproduce C0 A/B at I=R3 and I=R4. C_join A/B at I=R3 must exhibit an actual
retained intermediate state: candidate outcome consumed, root open at mandatory
join Receive, peer still before S admission. Then retain S admission, original
S outcome consumption, report commitment/delivery/root consumption, final peer
and aggregate status. An execution where peer finished first is insufficient.
Use unchanged Store.Run, MaxInFlight1, existing MaxSteps/MaxDispatch200 initially;
record any needed operational-step increase without changing semantic allowances.
No sleeps, selected identities, authoritative-state mutation or scheduler changes.

Focused controls use id1/value0 unless a distinct counterexample requires more:
separate I2/R3 and I3/R2, ordinary premature-report mutant, schema-valid wrong
request, known candidate failure/absence, separately identified S failure/absence,
pending branch and granted external cancellation. Preserve actual outcomes,
charges and causal rejection reasons. The primary S stays byte-identical.

Four recovery cases are A/B crossed with (a) consumed candidate/report uncommitted,
preferably S unadmitted, and (b) report committed or delivered/root unconsumed.
Use existing integration-only boundary observers to establish retained predicates,
ACK the cut, SIGKILL, and open the original DB in a fresh process. Preserve all
run/generation/artifact/operation identities, report identity when committed,
outstanding join and charges. Never redeploy/resubmit/replace a report. An
unsupported exact boundary is a named limitation, not an achieved cut.

A positive result is conditional bounded model evidence plus concrete sampled
single-host crash evidence. It establishes neither universal necessity/minimality
nor arbitrary-context substitution, confluence, compiler correctness, distributed
recovery, effectful recovery, or the manuscript's open Theorem 4.8.
