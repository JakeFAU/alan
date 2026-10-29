# Phase 1I substitution contract K1 — Stage 1 review candidate

Status: proposed before candidate authors begin. Changes require a retained prior
version, reason and independent review. This document supplies the contract to
both authors and to the independent model; it does not infer a contract from
their implementations.

## Question and three relations

Matching schemas gives structural compatibility only. K1 refinement means that a
component's own open transition graph satisfies the local response, demand,
correlation and waiting rules below under all declared local inputs and
environmental actions. It is checked without invoking surrounding contexts.
Contextual guarantee preservation is then a separate check of composed graphs.
Pairwise equality of trace sets is a third, stronger question; K1 allows different
costs and internal structures, so it does not imply Definition 4.6 equivalence.

The selected safety guarantee Gs is: every public success response is the unique
correlated answer to its request, with the required Boolean complement, and no
candidate-caused effect or resource demand outside K1 occurs. The conditional
completion guarantee Gc is: in the success mode, an open resource-compatible
context obtains that answer and (for sequential/parallel contexts) its ordinary
peer invocation also succeeds. This depends on application behavior: returning
the wrong request ID, ignoring the input, or spending extra resources violates
it even when runtime guards and replay accept the program.

The proposed bounded rule is: separately established A <= K1 and B <= K1, plus
the explicit prefix-compatible context assumptions, suffice for Gs and Gc in the
enumerated context class. The checker must report a counterexample if this is
false. No definition of refinement quantifies over all contexts; no universal
ALAN theorem follows from the finite experiment.

## Pure public protocol

One fresh invocation, one generation, one request. The public request record is
`{id: integer 1..2, value: integer 0..1, mode: string}` with mode in
`answer`, `fail`, `absent`, `pending`. The response record, when present, is
`{id: integer 1..2, value: integer 0..1}`.

* `answer`: after consuming the invocation request, send exactly one response
  with the same id and value `1 - request.value`, then terminate successfully.
* `fail`: commit failure without a response. This is application failure, not
  physical interruption or admission refusal.
* `absent`: terminate successfully without a response. Success status alone
  therefore does not entail the answer guarantee.
* `pending`: remain open at an unfulfilled ordinary receive after consuming the
  request; no response and no terminal outcome are required. No invented release
  observation is supplied in this profile.

The component may implement its answer locally or use one ordinary single-member
helper, whose request/result it must correlate. The whole candidate subtree,
including entry admission, uses at most 2 invocations and 2 created runs, zero
effect reservations and depth at most entry depth + 1. All four modes obey that
upper bound; unused allowance is not a charge. At most one helper operation may
be committed. No retries, re-entries, loops, extra messages, lifecycle requests,
or undeclared external effects are permitted in this pure profile.

Helper denial is permitted to yield candidate failure without a response in a
capacity/depth-incompatible environment; it must not manufacture the successful
answer. Entry denial creates no candidate run and returns the runtime's retained
refusal. These are safety cases, not cases satisfying Gc's resource hypothesis.

The runtime authenticates invocation provenance and matches operation identities;
the program must still preserve application correlation (`id`) and consume the
matching helper result. Terminal envelopes, response presence and response
contents are distinct observables. Ordinary wrapper status is its own status:
it forwards a present response and maps child success to its own success and
other terminal outcomes to its own failure. It cannot impersonate child
cancellation or abandonment.

## Named contexts and admissibility, fixed before A/B

This is finite template enumeration, not enumeration of arbitrary syntax trees.
The five shapes are `bare`, `sequential`, `parallel`, `hidden`, `wrapped`:

1. `bare`: one fixed caller invokes the substitution position and awaits it.
2. `sequential`: that caller obtains the candidate's answer, then invokes a
   one-member ordinary constant peer S and awaits S. S costs (1 invocation,
   1 created run, 0 effects) at depth 1.
3. `parallel`: the fixed caller and one fixed competing peer share a root.
   The caller invokes the candidate; the peer independently invokes S. Both
   members progress through the same runner. Fixed provisioning costs 2 runs;
   the other contexts have 1 fixed run. Static runs are outside dynamic charges.
4. `hidden`: `bare` with internal candidate/helper message labels hidden from
   the boundary projection. Charges, refusals, public request/result correlation,
   terminal outcomes and any external action remain visible.
5. `wrapped`: one ordinary single-member wrapper is invoked before it invokes
   the candidate. Wrapper admission costs (1,1,0) and adds one depth. It has its
   own correlated result and terminal outcome, as specified above.

Every context may be well-typed and authorized while resource-incompatible for
Gc. Report those categories separately. All are admissible for safety probing.
For completion, charge remaining *maximum obligations*, not final actual totals:
bare/hidden need (2,2,0), sequential/parallel need (3,3,0), wrapped needs (3,3,0),
with maximum depth 2/2/2/2/3 respectively. At every reachable prefix, prior
spending plus remaining worst-case candidate, peer and enclosing demand must
fit the envelope. Outstanding committed requests retain their full allowance;
an admission or reservation debit transfers demand from outstanding to spent,
never frees it. No competing actor beyond the declared one may spend the root.
This is a rely condition checked from actor residuals, not a runtime reservation
mechanism. Peers remain present and may act at every enabled prefix; no priority
or isolation is assumed. Insufficient capacity is retained as a distinguishing
case, never discarded as an invalid fixture.

Gc additionally assumes answer mode, no logical closure, no missing external
input (the pure answer protocol needs none), eventual execution of enabled local
and delivery transitions, and adequate declared operational step/byte limits.
Finite maximal executions can check completion; a depth cutoff cannot. Physical
interruption only pauses execution; after a finite number of interruptions the
same retained execution resumes. Arbitrary infinite crashes have no progress
guarantee. Gs remains applicable to prefixes and closure cases.

## Model, bounds and environmental transitions

Use a separate finite labeled transition system with explicit per-actor residual
control, request/response queues, pending invocations, terminal outcomes, root
spent/outstanding demand, depth, closure, and a consistent identity map. No
production reducer/compiler import or expected output obtained by running A/B.
Separate actors may step, commit an invocation, admit/reject it, publish a result,
settle it, consume it or close the root. Request commitment and admission are
distinct. The model authors must spell out these finite rules before census.

Frozen positive census inputs: five named shapes; id in {1,2}; value in {0,1};
four modes; initial dynamic invocation and run allowances independently in
{0,1,2,3}; depth cap in {1,2,3}; effects 0; generation 1. This is 3,840 context
configurations per candidate (5*2*2*4*4*4*3). One optional helper, one peer, one
wrapper maximum, with wrapper and competing-peer shapes separate. No arbitrary
recursion, repeated replacements, lifecycle retry or ownership transfer.
Exhaust all reachable states for each finite configuration; report generated,
resource-compatible, rejected and stuck outcomes plus transitions/revisits.
Use an explicit acyclic rank or graph cycle check; report any truncation as
incomplete exploration. Prefix closure cases and the effectful companion are
separately indexed searches/witnesses, not silently included in this census.

Distinguishing probes are separate artifacts: W_A/W_B use identical weak integer
input/output behavior but local versus helper cost; equal-charge P/Q differ by
one additional pure local step before a contested helper admission. Establish
their proposed local relation first, compare permitted trace sets, then record
actual Store.Run-selected schedules independently. Neither unequal charges nor
different concrete schedules alone falsifies Definition 4.6's conditional claim.

## Identity and observation abstraction

One bijective map renames fresh store namespaces, subjects, invocation/intent
operations and transport observation identities consistently across a trace.
It preserves equality, distinctness, causal parentage, source/target generation,
provenance, application id, operation-to-result linkage and restart continuity.
Artifact hashes and private instruction names/PCs may differ and are diagnostic,
not part of Gs/Gc. Concrete timestamps, lexical ordering of fresh hashes and
number of pure steps are outside Gs/Gc but are retained when analyzing Store.Run.
Never erase each identity field independently to force equality.

Runtime abstraction uses retained invocation admissions/outcomes, history of
consumed observations and emitted intents, final projections, resource charges
and reservations, and endpoint contact/outcome records. Admission != readiness;
an accepted inbox entry may wait for its receive. A hidden helper's charge and
its blocking obligation stay observable. A wrapper's result cannot overwrite
the child envelope in the abstraction. Model equality, runtime sample membership
and replay consistency are three different checks.

## Effect/closure/recovery companion K1E

Separate companion artifact/profile: one authorized non-idempotent controlled
endpoint request with immutable key/parameters and at most one effect reservation.
The endpoint may apply and lose the response. An unknown result is not success
or failure: the candidate waits with its original operation, and the charge
persists on fresh-process reopening. No automatic retransmission, replacement
intent or new business identity is permitted. This profile adds one effect to
the pure demand envelope; it is not evidence that pure A/B are effect-equivalent.

Logical closure before admission/reservation rejects new scoped work. Closure
after reservation preserves uncertainty; source authorization and root reservation
are distinct boundaries, and prior authorization/reservation may still permit
launch. Already admitted children may drain locally. A separately live transport
may settle authentic late evidence on the original operation; this does not
reopen closed work. There is no new fresh-process receipt reconciliation API.
Closure/recovery environment actions are labeled test-environment actions.

Pure recovery cuts: after candidate intent commitment before admission, and
after child outcome retention before caller consumption. SIGKILL follows a
retained-state ACK; recovery loads the same artifacts, generation and operation
and changes only activation authority as the existing runtime specifies. Replay
has no adapter/dispatch authority and must preserve live ledgers/endpoint contacts.

## Falsification

Execute ordinary faulty variants with valid schemas/programs: incorrect echoed
id and extra candidate subtree charge, plus an effectful unknown-to-success
variant if the controlled boundary makes the fault observable. Unchanged external
contract monitors must reject actual retained violations for those exact reasons.
A construction error or an earlier unrelated runtime guard is not detection.
Keep original inputs and shortest model counterexamples. A newly discovered
restriction requires an explicit versioned revision and independent rereview.
