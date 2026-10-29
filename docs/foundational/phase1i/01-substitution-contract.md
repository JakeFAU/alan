# Phase 1I substitution contract K1 — revision 1 for scoped rereview

Status: specification corrections R1–R3 are incorporated; independent rereview
is pending. Candidate/model implementation remains gated. The rejected original
is [contract-v0](evidence/reviews/contract-v0.md). Freeze this document's digest
in the reviewed author/model briefs before either candidate author begins.

## Question, guarantees and evidence categories

Schema compatibility, asymmetric behavioral refinement, and symmetric contextual
equivalence are different claims. K1 permits local and one-helper realizations
with different charges. It therefore does not imply Definition 4.6 equivalence.

Gs: every admitted candidate reply preserves the request's application `id` and
returns its Boolean complement; each request has at most one reply, and candidate
subtree actions/charges obey K1. Gs applies to all prefixes, including refusal,
pending and closure cases. It is stronger than runtime provenance validation.
Gc: in `answer` mode, under the initial and prefix entitlement below and eventual
service, the caller consumes the correct present answer and completes; sequential
and parallel S also complete successfully. Wrong application id, extra candidate
work or silent nonresponse can violate these guarantees despite valid schemas.

The investigation checks four separate obligations: (1) local A <= K1 and B <= K1;
(2) Gs/Gc on the nondeterministic reference products C[K1]; (3) Gs/Gc on C[A] and
C[B]; (4) correspondence of sampled retained runtime traces to those products.
A counterexample to any obligation is retained. Product enumeration is bounded
model checking of the named configurations, not a proof over arbitrary contexts.
The local relation below never quantifies over context executions. No universal
substitution or exhaustive concrete-runtime conformance claim follows.

## Public protocol and mode/outcome distinction

One fresh invocation, generation 1, request
`{id: integer 1..2, value: integer 0..1, mode: string}` with mode exactly
`answer`, `fail`, `absent`, or `pending`. Present response:
`{id: integer 1..2, value: integer 0..1}`. Public schemas validate structure;
finite-domain membership and behavioral relation are contract/environment-profile
obligations. `schema.Field` has no integer Min/Max: the id/value sets are
semantic input restrictions, not claims of scalar-schema enforcement. The mode field is String with MaxBytes 8 and Enum exactly the four modes,
enforced by existing schema fields. Out-of-domain
integer inputs are outside the census; optional ordinary branch rejection does
not extend the API or silently add cases.

After request consumption K1 chooses local work or one single-member helper.
This choice is permitted in each mode. The helper receives the same id/value
with `mode=answer`, computes the complement, replies once and succeeds. The
candidate must consume its original helper outcome before its mode action. A
helper refusal causes candidate failure without reply, regardless of mode.
With no helper, or after its successful present result, mode action is:

| Mode/boundary | Candidate local projection | Child invocation envelope (status/reason) | Response | Already-admitted wrapper's own envelope |
| --- | --- | --- | --- | --- |
| `answer` | succeeded, after one correct Send | succeeded/member_succeeded | present, same id and `1-value` | succeeded/member_succeeded, its own identical reply |
| `fail` | failed, no Send | failed/member_failed | absent | failed/member_failed, no reply |
| `absent` | succeeded, no Send | failed/missing_reply | absent | failed/member_failed, no reply; original child reason retained separately |
| `pending` | open at unmatched ordinary receive | no publication | absent | open awaiting original child outcome |
| helper refused | failed after consuming refusal | failed/member_failed | absent | failed/member_failed, no reply |
| entry refused | no candidate run | rejected/exact admission reason | absent | failed/member_failed, if wrapper entry itself was admitted |

Pending may use a second Receive on the public request port after the original
request was consumed. This environment supplies no second request/release.

The table assumes no logical closure or unresolved external descendant. A denied
wrapper entry has its own rejected envelope and no wrapper run. Default-bound
response fields never establish presence. Local Success is not invocation
success: `internal/executor/invocation_outcome.go:324–439` waits for reply
admissions, joins known descendants/external obligations, and maps missing reply
to failure. The pure K1 helper terminates normally or is refused; it cannot
produce an unknown external obligation.

## Frozen finite states and transition rules

K1 is a small nondeterministic specification, not an implementation interpreter.
State contains immutable request/root/generation and operation links; root
limits and spent `(I,R,E)`; depth; a root-closed bit; and the following finite
per-operation fields for entry/helper/wrapper/S where present:

- intent `uncommitted|committed`; admission `undecided|accepted|rejected(reason)`;
- request `absent|admitted|consumed`; reply `absent|committed|accepted|refused`;
- local control `uncreated|receive|choose|helper_wait|mode|pending|terminal`;
- terminal/freeze `none|succeeded|failed|cancelled`; publication and outcome inbox
  each `absent|present`; outcome consumption `false|true`;
- helper choice `undecided|local|helper`, finite payload, exact operation handle;
  no repeated request, reply, helper, publication or consumption.

A terminal commit and singleton member freeze are one model event: they share a
retained transaction. Publication, outcome delivery and caller consumption are
separate events. Root/caller local terminal is not a child invocation envelope.
The rules below apply to each singleton invocation; helper/S have fixed local
answer behavior, and wrapper/caller control is fixed in the next section.

| Event | Guard | State change / visible consequence |
| --- | --- | --- |
| `invoke_commit(t)` | open source at that Invoke; ticket t uncommitted | commit one immutable operation/handle and payload; no charge or child yet |
| `admit(t)` | committed, undecided | if source closed, reject `source_closed`; else if root closed reject `scope_closed`; else if depth exceeds cap reject `depth_limit`; else if I spent is at cap reject `invocation_limit`; else if R spent is at cap reject `created_run_limit`; else accept, atomically charge `(1,1,0)`, create singleton and admit its request |
| `request_consume(t)` | accepted, open member at its entry receive, request admitted | consume exact request once; enter `choose` for candidate, fixed answer for helper/S, child-invoke control for wrapper |
| `choose(local)` | candidate at choose | enter mode action; exclude unused helper ticket |
| `choose(helper)` | candidate at choose | enable exactly one helper commit, then wait on its original handle |
| `helper_consume` | helper outcome delivered, candidate open at exact AwaitInvocation | consume it once; succeeded+present+correct enters mode action; rejection enters local failure; any malformed helper outcome is a contract violation |
| `reply_commit` | open actor at answer Send after required input/helper consumption | commit one response with preserved id/complement (or S's fixed response); no charge |
| `reply_admit` | committed reply not yet decided | record authenticated reply admission; valid K1 reply is accepted; terminal source alone does not erase an earlier committed reply; closure/source-denial records remain distinct |
| `terminal_freeze(s)` | open actor at its mode end; answer has committed reply | set local succeeded/failed and capture committed reply/child obligations; pending mode instead enters unmatched receive with no terminal |
| `publish` | rejected admission, or frozen member with every captured reply admission known | rejection publishes rejected/reason; failed freeze publishes failed/member_failed; succeeded freeze additionally waits for all accepted descendant publications and known external results, then publishes succeeded/member_succeeded if reply accepted, otherwise failed/missing_reply |
| `outcome_deliver` | publication exists, not yet delivered | retain original operation's outcome observation in caller inbox; delivery alone is not continuation readiness |
| `outcome_consume` | caller open at exact matching AwaitInvocation | consume original outcome once and take caller table transition |
| `close_root` | root not already closed; external granted cancellation | mark root closed and root caller cancelled; forbid its future local commits/consumption; retain all facts/charges; new subtree admissions/reservations reject; already-admitted descendants may drain locally |

For the closed root caller, an outcome may remain retained/delivered without
being consumed. Root completion uses its cancellation envelope; it is not the
child's failure/success envelope. A previously published child outcome is
immutable. This phase models only root cancellation, not arbitrary child
retirement/ownership races. Runtime closure reasons and source-authorization
precedence are retained exactly in the abstraction, with `source_denied:*`
distinguished from the simpler source/scope guards above. These internal
source-authorization variants are a refinement of model `closed_refusal`, not
an assertion that every runtime closure refusal has one literal reason. No
closure success/progress claim is made.

## Frozen callers, S and named context products

The five shapes are named finite templates, not arbitrary context syntax trees.
All static callers begin open at their first Invoke with known request constants;
there is no extra external initialization event. They retain consumed child
status/reason/presence/value and exact handles in data. No caller validates away
a faulty candidate response: it forwards/uses present data unchanged so the
independent checker can detect wrong id/complement.

S is a singleton: request `{id:1..2,value:0..1}`, one matching Receive, one Send
`{id:request.id,value:0}`, then Success. S has no helper/effect. Its accepted
invocation costs `(1,1,0)` at depth 1. Denial publishes rejected/reason with no run
or charge. A static peer invokes S with the original id/value. The sequential
caller invokes S with the actual candidate id/value after its answer. S's
constant zero result is never mistaken for the candidate's complement.

| Shape | Initial work and fixed provisioning | On candidate/wrapper succeeded + present | On terminal non-success or succeeded + absent | Pending behavior |
| --- | --- | --- | --- | --- |
| bare | one static root caller; invoke candidate | retain exact reply, local Success | retain envelope, local Failure, no further invocation | await original envelope |
| hidden | same as bare | same; only internal communication labels hidden | same | same |
| sequential | one static root caller; invoke candidate | invoke S once with exact reply; await it; Success iff S succeeded+present, otherwise Failure | Failure without committing S | wait candidate or S; never skip either |
| parallel | static root caller plus static peer in same root | caller retains answer and Success; peer independently invokes/awaits S and succeeds iff S succeeded+present | caller Failure; peer still executes its one S path | each side waits only on its own original envelope |
| wrapped | one static root caller invokes singleton wrapper, wrapper invokes candidate | wrapper Send exact reply then Success; outer caller retains wrapper result and Success | wrapper Failure without reply; outer caller Failure | wrapper awaits candidate, caller awaits wrapper |

Each static caller's local Success needs no Send because the context result is
its retained data/status, not a further invocation return. The wrapper does
need Send because it is invoked. Parallel observation records both branch
outcomes: context success means both local branches succeeded; a terminal
failure in one branch does not cancel the other. A pending branch plus completed
peer is a partial/pending context, never full completion. Root publication, if
observed, is retained separately from this branch-result tuple.

## Tickets, entitlement and resource interference

Logical tickets represent worst-case **future demand**, not runtime reservations.
Entry e and optional helper h each cost `(1,1,0)`; sequential/parallel S adds s;
wrapped adds w. All are singleton admissions. No effect ticket exists in K1.
The initial ticket sets are `{e,h}`, `{e,h,s}`, `{e,h,s}`, `{e,h}`, `{w,e,h}`
for bare/sequential/parallel/hidden/wrapped respectively. The optional h remains
in the maximum demand until the contract's local branch makes it impossible.
Every ticket has state `possible|committed|spent|excluded`. Commit changes
possible to committed without debit. Acceptance transfers committed to spent
and increments the ledger once. Refusal excludes that unspent ticket; entry
refusal also excludes its unrealizable subtree. Choosing local excludes h.
Non-answer termination excludes sequential s. No rule refunds a spent ticket.

Let L be original limits, p actual spending before the modeled boundary, and U
unspent possible/committed tickets. Initial residual b is L-p. Entitlement is
fixed **from K1 plus the declared context**, before inspecting A/B termination:
`answer`, open root, `p + sum(initial tickets) <= L` coordinatewise, and maximum
path depth within cap. Thus bare/hidden need `(2,2,0)` and depth 2;
sequential/parallel `(3,3,0)` and depth 2; wrapped `(3,3,0)` and depth 3.
At every permitted prefix the rely invariant is `spent + sum(U) <= L` (spent
includes p). Acceptance moves, rather than frees, that amount. Branch exclusion
removes unrealizable future demand but never erases a charge. Demand is updated
only by K1 transitions. A faulty candidate extra commit/charge, premature stop,
or missing response is a conformance/guarantee failure; it does not remove the
initial entitlement or retroactively exclude a configuration.

The positive census starts p=0 at the fixed-provisioned boundary, with limits
I/R in 0..3. Runtime residual-budget probes use actual preceding admissions or
wrapper costs; no authoritative database edits. Static fixed cost is 1 run,
except parallel fixed 2; any test operator is reported separately. Created-run
charges do not include these fixed runs. Max depth is a bound, not a consumed
resource. All pure fixtures use positive physical capacity despite effects 0.

All 3,840 configurations are safety-compatible; insufficient capacity/depth is
not discarded. Completion-entitled configurations are the answer subset above,
unchanged across K1/A/B. Compute the count directly from the formula:
bare/hidden each `4 payloads * 2 I * 2 R * 2 depths = 32`; sequential/parallel
each `4 * 1 * 1 * 2 = 8`; wrapped `4 * 1 * 1 * 1 = 4`, hence **84** total.
The expanded arithmetic, not any candidate outcome, defines this partition.

Gc also assumes eventual service of continuously enabled local/protocol actions,
adequate finite step/dispatch/byte limits and no logical closure. After finitely
many physical interruptions the same retained work resumes. Infinite crashes,
watchdog exhaustion and absent declared environmental inputs confer no progress
guarantee. The finite model has no scheduler-idle self-loops; maximal finite
paths must end in their specified terminal/pending state. A cutoff is incomplete
exploration, not termination evidence.

## Separately checked local relation and reference products

The local open environment is frozen independently of the five contexts. Its
initial states range over the 16 requests, candidate entry depth 1 or 2, depth cap
1..3, residual I/R 0..3, one root/generation, effects 0, and an optional external
peer ticket q in `{absent,possible}`. Entry intent is initially uncommitted;
all member/protocol fields are empty and root is open. A wrapper, if already
admitted, contributes prior spending and entry depth 2, not an unbounded local
adversary. The local environment can service each declared admission, reply,
publication and delivery event once; take one `peer_debit(q)` of `(1,1,0)` if q
exists and sufficient residual I/R remain; and take `close_root` at most once.
It supplies no pending-mode release, fabricated reply, extra request, arbitrary
budget increment or other peer debit. Peer debit spends q exactly once and is
forbidden after closure. Local answer entitlement is entry+helper+q demand and
sufficient depth before execution; closure marks an explicit progress-assumption
violation, while safety remains checked. Scarce cases still explore peer-first
and helper-first choices without priority.

For candidate component graph M, use an asymmetric relation R between M states
and K1 states with equal public request, ledger, root closure, obligation links,
protocol stage and observed outcome; private control and unused private names
may differ. Initial pairs must exist for every local initial state. Then:

1. Each reachable M component event must match a finite K1 path with the same
   labeled commit/admission/payload/charge/terminal facts and end in R. Pure
   local instructions may stutter only when they change none of those facts.
2. Every enabled declared environment event on either side must be accepted on
   the other with identical event/ledger change and related successors. At
   implementation-private instruction boundaries the related K1 state may
   remain unchanged; environment service cannot be postponed by inventing a
   component action. This explicitly checks stability under the one peer debit
   and root closure, rather than assuming it from equal totals.
3. A maximal open/no-closure path in answer mode with initial entitlement must
   consume input and, after any chosen helper, commit the correct reply and
   reach succeeded publication. Fail/absent must reach their table envelope
   under the same service/resource assumptions; pending must reach the declared
   unmatched receive. Scarce-resource refusal paths must reach the corresponding
   failure/rejection when their finite required events are serviced. No other
   stuck state or extra action is accepted.
4. Show graph acyclicity/rank or analyze strongly connected components after
   removing terminal/pending states: an internal stuttering cycle is rejected.
   Unfair withheld environmental service is distinguished from component
   silence; “all finite output traces allowed” alone never proves progress.

R may choose K1's local/helper branch to match M; it need not match every K1
component choice (refinement is asymmetric). K1's own reference graph contains
both choices independently of A/B. Construct C[K1] with the frozen callers and
ticket semantics, enumerate all transitions and check Gs/Gc **before** treating
A/B products as preservation evidence. Record separate local-relation,
reference-product and A/B-product results and counterexamples. This does not
claim the local checks alone prove arbitrary contextual closure.

## Census and artifact-to-model correspondence

Base census per K1, A and B: five shapes *2 ids *2 values *4 modes *4 invocation
limits *4 created-run limits *3 depth caps = **3,840** named configurations.
Effects 0, generation 1; one optional helper/peer/wrapper maximum, with wrapped
and parallel separate. All reachable states/transitions, terminal/pending/stuck
outcomes, revisits and any truncation are reported separately per graph. Local
relation initial states are the independent product above, not this context
count. No recursion, retries, ownership transfer or multiple replacements.

Authors receive the same frozen contract/schema/API brief and implement
`New() (author.Template, error)` in separate directories. Freeze first source
and retained-artifact hashes before either author sees the other's logic.
The orchestrator assigns A the local and B the one-helper architecture solely
to ensure structural diversity; branch/value/response logic is authored
independently. Independence means separate implementation authors with no
cross-reading before freeze, not blind independent selection of architecture.
The independent model writer shares only the contract and public API brief
until the reference model is fixed, then inspects frozen artifacts for mapping.
The model author first implements K1/reference callers from this document, then
independently transcribes each frozen candidate artifact's acyclic control,
branch expressions, ports/handles, replies and terminal steps into its component
graph. Retain a reviewed mapping from each artifact instruction/branch to graph
edge or pure stutter, covering both branches, initialization sets, every emitted
intent and exact receive guard. No production compiler/reducer import, no
expected graph/output obtained by running A/B. Shared JSON decoding is permitted
and declared trusted; interpretation and expected values are independent.

Conformance initially concerns those transcribed finite graphs. An independent
review must inspect correspondence coverage; runtime samples must check retained
projection/event membership against it. Missing mapping or unmodeled opcode is
an actionable gap, not an ignored diagnostic. Inspected correspondence plus
samples is not a compiler proof or exhaustive runtime-semantics validation.
Operational step/dispatch/byte bounds are calculated from the frozen finite
artifacts before runtime qualification; changing them cannot hide a cutoff.

## Identity, hiding and observed records

Use one consistent renaming per trace/correspondence, not a bijection between
all private A and B names: only B may have a helper. Preserve public anchors,
equality/distinctness, causal parentage, generation, provenance, application id,
operation-to-result links and root affiliation. Own-state restart preserves the
same retained operations. Never erase identity fields independently. Additional
private helper names/charges are allowed by refinement, not hidden into alleged
pairwise equality. Artifact hash, private PC/registers, timestamps and pure step
count are outside Gs/Gc, but retained for diagnostics/scheduler analysis.

Abstraction uses original admissions/reasons, reply evidence, member freezes,
publications, consumed observations, final projections, charges/reservations and
endpoint contacts/evidence. Hiding changes internal communication labels only;
shared debit, refusal, blocking obligation, child/enclosing outcome distinction
and external action survive. Model membership, runtime sampling and replay
consistency are separate checks.

## Separately indexed probes, closure and effect/recovery companion

These are outside the 3,840 census and must have per-scenario source/result IDs:

- `weak/{1,2,3}`: sequential local W_A versus one-helper W_B, identical integer
  input 0/output 0, I=R in 1..3, depth 2. At 2, W_A can admit S but W_B spends both
  admissions before S; also run `weak/run-limit` with I=3,R=2. Expected outcomes
  are hypotheses until executed. Unequal transfer debits cannot refute Definition 4.6.
- `equal/{1,2,3}`: both invoke the same helper, Q adds exactly one pure Set before
  helper Invoke; fixed peer competes for S, I=R in 1..3, depth 2, input 0. Check
  the proposed local weak relation first, compare abstract trace sets, then
  record actual Store.Run schedule/name order. Different concrete winners alone
  do not refute abstract equivalence; no scheduler/identity manipulation.
- `closure/{entry-commit,entry-admitted,helper-commit,reply-commit,published}`:
  bare answer id 1/value 0, I=R=3, depth 3, root cancellation once at each reachable
  named boundary before its next protocol action. Cross both local/helper
  reference paths and each concrete artifact; report absent helper boundary as
  inapplicable by construction, never as a passed schedule. Inspect original
  outcome/closure reasons; do not invent child cancellation.
- `pure-reopen/{entry-commit,outcome-retained}`: each candidate, bare answer
  id 1/value 0, I=R=3, depth 3; ACK then SIGKILL and fresh-process Open/Run before
  entry admission or caller outcome consumption respectively. No artifact swap.
- `effect/{known,unknown-reopen,closure-before-reservation,reserved-then-closure,late-live}`:
  separate K1E artifact, bare id 1/value 0, I=R=2, depth 2, effects 1, one controlled
  non-idempotent endpoint request; no helper. Known authentic result supplies
  value 1 and permits answer; unknown after attempted request remains pending
  with original operation and spent reservation. Unknown-reopen uses ACK,
  SIGKILL and fresh Open without receipt/retransmission. Late-live retains a
  surviving transport, closes root after contact, then releases authentic
  original receipt. These are distinct schedules, not a new reconciliation API.

K1E replaces the optional helper ticket with one effect ticket `(0,0,1)` and
requires a matching known response before replying. Source authorization and
final root reservation are separate: only an already allowed **final reservation**
can survive closure and permit its original launch; earlier source authorization
alone cannot bypass the later scope check. Unknown is neither failure nor
success. Late evidence never authorizes replacement work. Runtime successful
publication also joins unresolved external obligations. Pure A/B conformance
therefore does not establish effectful substitution.

## Decisive negative controls

Execute ordinary valid-schema variants: (a) answer with the other allowed id,
while preserving complement; (b) commit/admit a second candidate helper, giving
three subtree admissions including entry, under a larger witnessing envelope
I=R=3/depth 2. The latter is deliberately outside K1's subtree demand, not outside
runtime capacity. The unchanged checker must reject actual retained wrong-id
reply or extra subtree commit/accepted charge for that reason. Retain successful
protocol/replay behavior where observed. Construction errors or unrelated guard
rejections are not semantic negative detections.

Unknown-to-success is not a required negative: an attempted premature Send/Success
may be blocked by runtime publication joining. If exercised, report that guard
result, not a vacuous candidate-contract detection. Preserve every original
counterexample and reviewed revision; a later restriction needs semantic
justification and independent rereview.
