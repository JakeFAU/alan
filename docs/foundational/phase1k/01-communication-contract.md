# Phase 1K communication contract — proposed revision 2

Status: inspected design for independent review before new experiments. This
freezes questions, profiles, protocol and apparatus requirements; predicted
outcomes below are not executions. The controller records approved digests and
any rejected revisions in `03-review.md`. The binding brief is [assignment.md](assignment.md).

## Question, baseline and unchanged scope

When may a participant terminate without stranding communication still required
by its composition, and which relevant facts can retained public programs observe?
The narrow boundary is ordinary message commitment → first source authorization
→ destination admission → selected consumption. Source termination may fall
between these distinct transitions.

The controller verified merged main `94e97d5ba8b7ed2b3e22d0e5ab86fa91b7a2adc5`,
tree `a20478fa31b333c6ff0be43fc9b43826b54bbb3a`, as the Phase1K baseline.
PR21's reviewed head is `1d13bc238aada500c3b9608d400435e235fedc0b`; its tree
matches the baseline. The historical J1 experimental baseline `d68ec667d276a4bd9e1be11613905c628cfea704`,
integration base `5764ec0b5679c72c80e56c44fd9e4198c986b91c`, and reviewed source
aggregate `55260ec410e9b63393b5c24a211bdb70866228a8edb75d2ec4bc28345af8da3e`
remain distinct. See `evidence/baseline/baseline.json` for controller evidence.
PR19 formatting and PR20's mathematical correction are inherited work.

Preserve production executor, compiler, schema, dependency, accounting and
scheduler bytes. Preserve C0/K1/J1 artifacts, eight-configuration results and
historical evidence. J1 established bounded resource/root-lifetime properties,
but modeled report-refusal pending states in every positive configuration.
Concrete delayed-peer and four crash samples completed. The bad model ordering
was not an observed failure of unchanged `Store.Run`. Historical complete race
coverage used oracle v1; latest-oracle runtime race coverage selected Runtime
and Decoder only. This phase does not relabel either scope.

Primary concrete configuration: frozen B, id1/value0, answer, parallel, I=R3,
depth2, E0, physical capacity1, static root+peer2. A is added only for a distinct
uncertainty or the small artifact/resource correspondence check. No identity
search, sleeps as causal evidence, redeployment during recovery, SQL mutation,
paid calls or production repair is authorized by this contract.

## Three execution profiles

| Profile | Enabled transitions and claim domain | Limits and evidence required |
|---|---|---|
| Independent protocol model | Every enabled local/protocol transition in the finite retained-control product; report and ACK commitment, first disposition, admission and consumption are separate; source terminal and aggregate publication are explicit | Finite product and rank below; eventual service of enabled progressing work, truthful participants, no external closure or routing changes. A modeled counterexample is model reachability until mapped to public APIs. |
| Supported public runtime driver | An external consumer chooses genuine `Commit` steps, original invocation admission/settlement, and original message authorization/admission through exported APIs; current authority and actual matching retained observations only | Unchanged programs, no concurrent `Store.Run`, no branch/register manipulation or synthetic observation. Each call carries pre-state, instruction, identities and post-state. Concrete witness establishes this API transition system, not automatic scheduling. |
| Unchanged automatic `Store.Run` | Actual pending-delivery ordering, ready-run scan and one-delivery/one-local-commit loop, with root settlement; deployment/restart state and options fixed | Record pending queue and readiness at decisive cut. A sample selects an ordering; a source exclusion is conditional on the inspected state predicate. Do not generalize to other queues or arbitrary restart states. |

Common positive assumptions: static subject/generation1 and revision1, unchanged
valid grants and schemas, truthful original S/candidate paths, one request per
generation, no cancellation/retirement, no failures in the host apparatus or
storage. Fair service means enabled work is eventually attempted; it does not
turn a refusal into delivery. A finite quiescent pending observation is not proof
of infinite waiting. Unavailable public control or a missed cut is an apparatus
limitation, not semantic impossibility.

### Automatic ordering inspected before experiments

`internal/executor/runner.go:51-266` obtains pending deliveries, selects the first
internal item, performs it, then selects and commits at most one ready run.
`deliverOrdinary` delivers then acknowledges a message; it does not drain the
queue. `pendingDeliveries` (`runner.go:378-472`) sorts keys lexicographically:
ordinary messages remain pending until a `message_acknowledgments` row exists;
invocations remain pending before admission or when ready without an outcome
delivery; terminal routes remain pending until their delivery bit is acknowledged.
No effects/lifecycle work exists in the primary fixture. `selectReady`
(`runner.go:297-368`) rotates after the prior run, skips terminal/retired/stale
runs, tests current instruction eligibility, and scans unconsumed observation
ids in sorted order. On reopen the volatile last-run pointer is absent.

A useful **conditional source result** is: if the only pending internal item is
the report, the peer is open just after Send, and root is open, the next ordinary
runner loop authorizes/admits/acknowledges that report before any local commit.
No property of ready-run rotation can reverse that order. It remains conditional
on valid permissions/schema and on no independent concurrent driver.

The historical B cut
`../phase1j/evidence/runtime/final-normal/babe8c256071/before.json.gz` was read
in Task1. Its report is `["peer",1,37,0]`; peer is open at PC37/sequence37,
root open at PC2/sequence2; there is no report authorization, admission or
acknowledgment. All three invocation operations have outcome-delivery rows,
all three invocation replies have message acknowledgments, and the three
terminal obligations have empty routes. Applying the source queue rules gives
one pending internal item: that report. This is an inspection of historical
bytes, not a fresh qualification or relabeling of the original recovery run.

Fresh automatic B/J1 and exact preauthorization-restart samples must verify that
same predicate, retaining original table rows and reconstructed queue reasons.
If it holds, LATE is excluded at that exact runner boundary; do not call this a
general delivery-before-termination rule. J2's ACK boundary needs its own queue
inspection. No predicted J2 queue fact is frozen as an observation.

## Durable handoff and visibility map

| Boundary | Implementation and durable evidence | Retained-program visibility |
|---|---|---|
| Ordinary Send commits | `author/compile.go` emits message intent; `Store.Commit` (`store.go:715-852`) atomically retains history, successor and `intents` | Program advances; current `author.Send` binds no message-result handle or permission receipt |
| First source disposition | `AuthorizeIntent` (`messages.go:92-178`), `attempts` id `authorization(intent)` with immutable descriptor/result and source authority snapshot | Returned to host; no ordinary authored observation is created |
| Admission accepted/refused | `DeliverMessage` (`messages.go:180-225`) reuses that authorization, then `admit` (`messages.go:385-492`) retains `admissions` | Host receives `AdmissionResult`; refusal does not create a receiver observation |
| Destination observation retained | Accepted `admit` inserts `observations` in the same transaction; link is `AdmissionResult.ObservationID` | Eligible Receive may later select it; admission itself does not advance receiver |
| Receiver selects/consumes | `Commit` checks actual selected observation, executes current instruction and atomically writes `history.selected`, `observations.consumed_sequence`, successor | Bound payload becomes available only on that actual successful receive step |
| Local terminal | Same Commit records terminal projection, `terminal_obligations`, and composition execution/freeze facts | Source cannot execute further local instructions; authentic terminal notifications are a separate route |
| Aggregate completion | `PrepareRoot`/`SettleRoot` (`composition_root.go:12-85`) and `composition_root_publications` | Host inspection exposes completion; ordinary Receive does not automatically observe it |

Authorization checks source revision, generation, retirement and open status on
the first attempt. Ordinary messages have no terminal-source exception. A valid
composition reply has a separately identified exception through
`compositionReplyBinding`; its outcome settlement remains different from ordinary
report delivery. Later authorization calls return the exact first disposition,
even after source closure. Destination admission still checks current target
existence, generation, retirement, open status, route and schema. This is not
unconditional end-to-end delivery.

`AcknowledgeMessage(ctx,intentID,transportID)` links an original intent to a
retained admission (`messages.go:231-261`). It permits an admission refusal;
it is dispatch bookkeeping, not accepted delivery, consumption, or a retained
ordinary protocol ACK. The latter is a new message with its own authorization.

**Hypothesis H-auth:** for the same ordinary intent, with a retained allowed first
authorization, an admission-open receiver, stable routing/generations and valid
payload, later source termination alone does not invalidate that authorization
or prevent that message's admission. Eventual dispatch/eligible receive service
is needed for consumption. This hypothesis concerns source lifetime, not receiver
lifetime, successful external effects or general composition completion.

## Public driver legality and J1 pair

Inspected public entry points are aliases in `runtime/runtime.go`; the aliased
Store exposes exported methods even when a returned type lacks a facade alias:

```go
runtime.Create(ctx context.Context, path string, p runtime.Provision, l runtime.Limits) (*runtime.Store, error)
runtime.Open(ctx context.Context, path string, l runtime.Limits) (*runtime.Store, error)
runtime.Inspect(ctx context.Context, path string, l runtime.Limits) (runtime.Inspection, error)
store.Snapshot(ctx context.Context, id runtime.RunID) (runtime.RunState, error)
state.Authority() runtime.Authority
store.Commit(ctx context.Context, authority runtime.Authority, selected string) (runtime.RunState, error)
store.AuthorizeIntent(ctx context.Context, id string) (executor.Authorization, error)
store.DeliverMessage(ctx context.Context, intentID, transportID string) (runtime.AdmissionResult, error)
store.AcknowledgeMessage(ctx context.Context, intentID, transportID string) error
store.AdmitInvocation(ctx context.Context, operation string) (executor.compositionAdmission, error)
store.PrepareInvocation(ctx context.Context, operation string) (runtime.InvocationPreparation, error)
store.SettleInvocation(ctx context.Context, operation string) (runtime.AdmissionResult, error)
store.PrepareRoot(ctx context.Context, rootID string) (runtime.InvocationPreparation, error)
store.SettleRoot(ctx context.Context, rootID string) (runtime.InvocationPreparation, error)
```

The two `executor.*` spellings above describe actual declared return types, not
an import permitted in the external consumer. Use inferred local variables and
exported fields; test this external accessibility. `Inspection.Tables` exposes
retained JSON descriptors, not a public method to edit them. Root ids come from
inspection; operation/message ids come from actual history/intents. Use the
runner's original transport identity encoding for comparable dispatch evidence,
and preserve it on reattempt. A locally chosen different transport does not
replace the original obligation.

Every driver local step records: original program id/hash; run/generation;
Snapshot authority, PC and projection; instruction bytes from inspected program;
selected observation id/descriptor if any; returned state and committed history
row. Pure steps select `""`; receives select only matching real unconsumed
observations. Resolve receive operands and composition bindings from inspected original records;
do not call private Reduce or boundCompositionProgram. If selection probes actual
unconsumed observations through Commit, retain failed no-op attempts separately
from successful transitions. Drive invocation children and replies through their
genuine programs and `AdmitInvocation`/`DeliverMessage`/`SettleInvocation` paths. Never
replace an authored Invoke, Send, Receive, branch or End with a host operation
having an apparently equivalent effect. Failed public API calls remain evidence.

Use two fresh databases provisioned from exact frozen J1 B bytes:

| Pair | Required retained prefix and intervention | Prediction to test |
|---|---|---|
| LATE | Candidate and S outcomes genuinely consumed; peer truthful report committed; verify no first authorization; execute genuine remaining peer jump/End instructions; first authorize same report; attempt same original report admission and acknowledge actual result | First disposition `source_closed`; admission `source_denied:source_closed`; no report observation/consumption; root and aggregate remain pending |
| EARLY | Corresponding report commitment; first authorize while peer open; verify allowed; execute peer's genuine terminal continuation before any report admission; deliver same intent to open root, then actual root receive/decision | Same authorization reused; accepted admission, exact payload consumed; actual successful local/aggregate outcomes |

Retain intermediate snapshots, not just final status. If admission is not
attempted, explicitly report its absence rather than inventing a refusal. Run
unchanged automatic B separately with `MaxSteps:200, MaxDispatch:200,
MaxInFlight:1`; document any operational-bound increase without semantic-budget
changes. A public LATE trace is meaningful even if automatic selects EARLY.

## J2: one ordinary acknowledgment candidate

J2 is separately named, built only with existing author operations in
`examples/communication`. It retains A/B/S, static `root`/`peer` generation1,
resource membership/limits, one request and the original 32 Set prefix and
S Invoke/Await discipline. New root/peer artifacts use names
`phase1k-ack-root` and `phase1k-ack-static-peer`. Ports `report` and `ack` use new
obligations `phase1k_s_report` and `phase1k_s_ack`. Report schema and truthful
field meanings are identical to J1, including exact reason/presence and literal
zero response fillers on absence. Primary S is obtained from the frozen public
J1 builder, never reconstructed.

ACK schema: `request_id` integer and `s_operation` string MaxBytes256. Root
echoes the request id and S handle from the actual correlated report; peer checks
both against its retained original request and `peer_pending`. Runtime matching
also checks designated source, generation1 and obligation. There is exactly one
report and at most one ACK per generation. ACK does not contain a bound Send
receipt: current author.Send supplies none. The one-report context and independent
raw links establish which original report it acknowledges.

Full authored branch policy:

1. Root executes original candidate Invoke/Await. Pending candidate keeps it at
   Await even if a report is already admitted. All known candidate outcomes,
   including negative or absent response, proceed to mandatory report Receive.
2. Peer consumes its actual S outcome before sending one truthful report. Known
   failed/cancelled/abandoned/rejected or absent outcomes are reportable. Pending
   S sends nothing. A presence branch selects actual bound response fields or
   explicit zero fillers, respecting AwaitInvocation's deletion of absent bindings.
3. Root checks report request id before ACK. A wrong id enters an additional
   report Receive with no report supplied in this finite profile, then Failure
   only if that extra receive ever returns; it sends no ACK in the declared
   one-report execution. This parks invalid correlation, as J1 does.
4. After a correlated report root sends exactly one ACK, then applies J1's final
   checks: candidate succeeded/present/original id/complement and S
   succeeded/present/original id/zero produce Success; any known negative or
   mismatching response produces Failure. Exact reasons remain retained. ACK
   asserts report consumption, not successful S or successful composition.
5. Peer waits for ACK after Send. On matching request id **and exact S handle** it
   executes local Success, regardless of S outcome. A wrong id or wrong handle
   enters a further ACK Receive with no next ACK supplied, and cannot voluntarily
   terminate in this declared profile. No ACK-of-ACK, retransmission or retry.
6. Root Failure can freeze the shared static peer before ACK consumption. Desired
   authored peer Success is not a guarantee that the runtime reaches it on these
   negative branches; retain actual cancellation/freeze and outcome. The primary
   successful-answer census avoids conflating this separate failure behavior with
   the ACK source-lifetime race. No full negative recovery campaign is required.

The proposed completion claim is deliberately falsifiable: under positive
assumptions and eventual service, the successful-answer J2 product completes.
Attack trace: root consumes report → commits ACK → genuine root Success → first
ACK authorization refused `source_closed` → peer still open at ACK Receive and
aggregate completion pending. Realize it through legal public steps if possible,
with paired allowed-ACK-authorization → root Success → original ACK admission and
peer consumption control. Root local success is not aggregate completion.
Successful A/B still spend two/three dynamic invocations and created runs, plus
two fixed runs, zero effects; report and ACK must hide no extra admission.

Stop J2 after a decisive trace or concrete public obstruction is qualified. One
failed handshake does not prove impossibility for all finite ordinary protocols.

## Independent finite product and oracle

The primary J2 product is one configuration: id1/value0/answer, I=R3/depth2/E0,
reference K1 with its two permitted local/helper choices. Independently map frozen
A/B where used. Preserve J1's original eight configurations and results unchanged.
Reuse reviewed pure `research/substitution` primitives and `research/join`
report facts only; the reference worker imports no author/runtime/executor and
freezes its model/oracle before inspecting newly built J2 artifacts.

A narrow representation may extend the existing J1 state with `AckStage` in
0..4 (absent, committed, first disposition, admission disposition, consumed),
`AckAllowed` and `AckDelivered`. Filter peer terminal until ACK consumption and
root terminal until ACK commitment; report consumption still requires candidate
consumption. Root ACK commitment and peer ACK consumption increment their own
actor sequence, never a fabricated global order. Report/ACK authorization and
admission are distinct edges; accepted admission and observation retention are
one atomic boundary. Bookkeeping acknowledgment and exact idempotent reattempts
are non-progressing host stutters, not new successful deliveries.

Freeze finite domains before enumeration: four base operation slots e/h/w/s
(w excluded), optional h, root/peer local status open/succeeded in the primary
profile; context RootPC/PeerPC0..7, operation PC0..8, ticket0..3, admission0..2,
request-stage0..2, choice0..2, reply0..3, freeze0..2; booleans retain the existing
base meaning. I/R0..3, E0,
depth at most2; private PCs0..15, private Boolean/answer registers0..1 for this
id1/value0 case; delay0..32; report/ACK stage0..4; aggregate absent/succeeded.
Model root actor sequence0..4 and peer sequence0..36 (32 Sets, S consume,
report commit, ACK consume, terminal) are bounded diagnostic coordinates.
All operation/report/ACK identities are fixed symbols, payloads come from the
finite request and authentic corresponding outcomes, never unbounded strings.

Candidate intrinsic rank:
`100*substitution.Rank(base) + sum(privatePC) + privateRegister[e][7] + delay + reportStage + ackStage + aggregatePublished`.
The inspected conservative bound is 0..23202. With context PCs at most7 and
operation PCs at most8, the base-rank bound is
`6 + 7 + 7 + 3 + 100 + 4*(3+2+2+8+2+3+2+5) = 231`, including a conservative100
closure allowance. Private PCs contribute at most60, register1, delay32, stages8
and publication1: `100*231 + 60 + 1 + 32 + 8 + 1 = 23202`. Every progressing
edge, including edges to a seen state, must strictly increase rank. If the model
worker needs another control coordinate, revise this bound and record review
before interpreting counts. History/path length is not a rank substitute.

Enumerate without success pruning or cutoffs; inspect every reachable maximal
state and retain state/edge/maximal-category counts, rank failures, cycles,
cutoffs and decisive traces. Counts are not failure probabilities or concrete
instruction-schedule counts. Expected categories include complete and
ACK-refusal/open-peer pending; unexpected states are findings, not pruned.
Finite pure branch/jump suffixes need an inspected stutter mapping that preserves
terminal eligibility and permits service before/after their steps. A negative
in the single primary product suffices; no new eight-configuration campaign.

The pure oracle reuses truthful-report/candidate-before-report checks with the
actual obligation mapping explicitly supplied. Additional evidence joins the
original report/ACK intent, first authorization id and source snapshot, actual
admission/observation, selected consumption, local terminals and aggregate.
Reorder unrelated actor records without changing verdict; do not infer cross-run
order from JSON arrays. Negative decoder checks must reject denial acknowledged
as delivery, admission treated as consumption, wrong message/auth/selected
observation substituting for the original, wrong ACK request/handle, and changed
source-authorization sequence. Reuse unchanged J1 truth/correlation tests under
their source identities rather than rewriting them.

## Three exact process cuts

Use only B/J1 with original DB, artifacts, generation, handles, message and
authorization identity, charges and normal fresh-Open activation renewal. A new
Phase1K process child uses existing `internal/boundary` acknowledged pipe and
`process.wait` SIGKILL/fresh-process facilities. No synthetic stop/resume.

| Cut name | Predicate before parent ACK/SIGKILL | Reopen profile and required observation |
|---|---|---|
| `report-no-authorization` | Peer open exactly post-Send, original truthful report committed, no matching attempt/admission/ack; root open; complete queue facts retained | Unchanged automatic Run, verify exact queue predicate and actual authorization/termination order; if unavailable, explicitly use public-driver profile and label it |
| `report-authorized-peer-terminal` | Same report has allowed first attempt; peer genuine terminal; no report admission/observation/ack; root open | Public driver delivers same report; verify original allowed attempt/snapshot preserved despite renewed epoch and source closure, then actual consumption/outcome |
| `report-refused-peer-terminal` | Peer terminal; same report first authorization refused source_closed; actual admission refused source_denied:source_closed under the original runner transport; dispatcher acknowledgment already retained; root open and aggregate pending | Public driver reattempts the same authorization and delivery identities; compare exact retained authorization descriptor/result/source snapshot and admission descriptor/result, preserve refusal/bookkeeping with no replacement records, and observe actual root/aggregate pending state |

For the refused cut, complete `AuthorizeIntent`, `DeliverMessage` and
`AcknowledgeMessage` before the checkpoint, using the original report intent and
runner transport `identity("message-delivery", intentID)` (the existing JSON
encoding). After SIGKILL/fresh Open, call `AuthorizeIntent` and `DeliverMessage`
again with those exact identities. Require byte-identical old authorization
descriptor/result (including its original source authority snapshot) and old
admission descriptor/result, with the same dispatcher-acknowledgment linkage.
No replacement attempt/admission, new report observation or consumption, charge
or permission may appear. Activation ownership renews normally; this must not
rewrite the retained authorization snapshot. Inspect root local open/waiting and
absent aggregate completion separately. An authorization-only refused prefix is
not this retained-delivery retry cut.

Do not call Snapshot or another mutex-taking Store method within boundary hooks:
`execution.committed` and `intent.authorized` callbacks can hold `s.mu`.
For public cuts, finish each API call, inspect at that safe driver boundary,
validate predicate, then use the existing pipe signaling observer to pause.
For automatic cut, filter the genuine post-Send execution event without store
reentry; parent reads the committed DB via independent `runtime.Inspect` while
child is held. Verify actual predicate before kill. A checkpoint's name alone
is never proof. Retain ACK token/PID/sequence, process binary hash, killed exit,
new PID/epoch and exact before/after immutable-row prefix. If unsupported, record
the achieved boundary and obstruction. Replay is read-only and reissues nothing.

## Expressibility and terminal probe

Audit separately (a) current `author` constructors, (b) lower-level public runtime
artifacts, (c) host dispatch/inspection. Source inspection finds `author.Receive`
lowers to ordinary message/local input matching; Await/AwaitInvocation use their
own authenticated effect/invocation paths; Send exposes no message-result
binding. These facts do not establish an API-wide runtime absence.

The public runtime accepts a raw `runtime.Artifact{ID,Semantics,Bytes}` with
`alan/1d/v1` and no author metadata. Proposed external consumer probe:
provision an authored Success peer and raw receiver with a receive matching
`Kind:terminal`, `Provenance:runtime-terminal`, source peer/generation1,
obligation `done`, empty operation, binding payload `outcome`; then End Success.
Provision `runtime.Permission{Kind:"observation",Source:"peer",Target:"receiver",
Operation:"terminal",Revision:1,Obligation:"done"}`. Commit peer terminal,
inspect its frozen terminal route, deliver it, commit receiver with the actual
returned observation id, and inspect consumption/history. The terminal descriptor
uses target generation0 and binds at first admission; this probe fixes the receiver
at generation1 throughout. No internal imports.
`DeliverTerminal`/`AcknowledgeTerminal` expose an unaliased `TerminalDestination`;
try the assignable anonymous `struct{ Target, Obligation string }` argument.
Compilation, provisioning and execution remain untested predictions until the
external probe runs. Preserve exact obstruction if any; add no facade alias.

An authentic terminal observation proves the designated source reached its
retained terminal boundary; it does not retroactively prove ordinary-message
consumption. A later protocol might combine report/ACK evidence with authentic
peer-terminal observation, if this observation path is usable. This phase probes
accessibility only and does not implement a third join protocol or alter J1.
Invoking the peer instead would use invocation reply/settlement semantics, change
topology and accounting, and is a separately documented alternative, not this
static-peer solution. Any universal non-expressibility theorem needs explicit
language/scheduler assumptions and a separate proof.

## Evidence classification and stopping

Each final finding is labeled proved, bounded model-checked, concretely tested,
source-inspected, conjectured or unresolved. Retain original raw records, source
and artifact hashes, command/environment/exits, selected nonzero tests and the
causal links used by the oracle. Archive compact decisive traces and failures;
never overwrite J1 evidence or represent prior runs as current qualification.

Completion requires the three-profile relationship, J1 pair or exact obstruction,
single J2 bounded result, three cuts or named unachieved boundaries, surface
accessibility findings, synchronized scoped manuscript claims, independent
review and draft PR. No new coordination feature is required. The smallest next
semantic question must follow the evidence: execution ordering, program-visible
authorization evidence, or this particular handshake's remaining obligation.
