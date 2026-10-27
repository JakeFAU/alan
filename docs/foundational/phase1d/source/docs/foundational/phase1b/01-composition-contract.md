# Phase 1B public composition contract

Status: bounded research contract, defined before implementation. It is an allowed-behavior contract for the selected fixtures, not an equivalence theorem for arbitrary compositions. Phase 1 atomicity and retention assumptions remain unchanged.

## Boundary to be tested

A caller sends a request to a service with a stable request identity containing the caller's run generation. A leaf provider performs one declared unit resource operation and replies. The alternative provider receives the same request, issues an ordinary downstream request, waits for its worker's result, and replies through the same boundary. An outer coordinator uses that provider as one branch alongside an independent peer. The nested topology is outer → provider → worker; all participants execute the same finite instruction interpreter.

The public observations are admission/rejection and its retained run binding; correlated result contributions; terminal outcomes and delivery; and external resource applications, operation keys, arguments, and unresolved receipt state. Internal endpoint names may be hidden when comparing the public contract, but resource mutations, duplicate contributions, failure, and closure cannot be hidden. The two implementations need not have identical internal traces or timing.

## Allowed behavior

1. **Admission:** acknowledgment establishes inbox acceptance or rejection only. A service message binds at its first admission; retransmission retains that binding across closure/retry. Acknowledgment loss cannot transfer the old message to a new generation.
2. **Correlation:** a consumed result must be an ordinary message from the expected authorized peer, carry the expected request identity including generation, and identify the logical branch result it satisfies. Receipt observations and environmental ingress cannot impersonate generated peer results. Tags alone convey none of this authority.
3. **Aggregation:** one logical result key contributes at most once, including when two different Send intents carry that same key. Another legitimate branch remains able to contribute. The accepted-key state is ordinary bounded program state, committed and replayed with the reducer; it is not a privileged coordinator service.
4. **Success:** a provider may publish success only after its declared resource result or its downstream provider's valid result has been observed. An outer success is supported by both distinct branch contributions and the declared aggregate. The leaf/composite comparison checks this obligation, provenance, and resource footprint, not scalar equality alone.
5. **Failure and closure:** failed, abandoned, cancelled, and succeeded outcomes are immutable. A result admitted before closure may remain permanently unconsumed. Preparing its receive before closure does not permit commitment afterward. No late result reopens that run or satisfies another run's request.
6. **Retry:** a fresh generation has fresh request/effect identities and initial ordinary state. Old results can arrive while it is active; only a result for its current obligation may contribute. This increment must actually execute new-generation instructions. Lifecycle authority events are injected by the test environment; they do not demonstrate ordinary supervisor policy.
7. **Resource visibility:** application and receipt recording remain separate. A unit mutation is attributable to an authorized immutable effect descriptor and the selected request path; a successful result requires its declared evidence. No receipt means unresolved knowledge. Closure cannot undo an applied operation or prevent a previously authorized attempt from applying. Keys remain scoped to execution; no business-level deduplication across generations is promised.
8. **Progress:** waiting inner work must not disable an independently enabled peer commit. Complete nested/leaf runs are possible witnesses. No eventual-response, fair scheduling, group-stop, or cleanup promise is added. Partial effects and unresolved outcomes are allowed and reported.

For the comparison fixture, both provider variants perform the same declared unit operation (label and argument) through one leaf execution on their request path; the independently progressing branch has its own unit operation. Concrete effect IDs/provenance differ and remain visible in evidence. A mapping of each concrete leaf to its public branch must be reported explicitly. A future contract that exposes internal topology or timing could distinguish the implementations.

The one-unit limit is per complete public/root request identity, including its caller generation. Two different nested request envelopes do not create two independent public resource budgets for the same upstream request. Distinct fresh root generations remain distinct operations; this rule does not introduce business-level deduplication across retries.

## Ordinary operational extensions permitted

Retain finite sender/run, request correlation, logical result key, and reply-routing data through committed descriptors, transport, admission, and observations. Add explicit instruction operands to capture an upstream envelope, forward a downstream request, send a computed register value, and match/record a correlated result. A nested provider must retain upstream and downstream correlation simultaneously. Bound those slots and key sets as in the plan.

The selected representation adds optional `Mode`, `Slot`, `Key`, and `FromReg` instruction operands. The expected peer is derived from the request's target, not a new authority source. A finite request envelope records caller, caller generation, request commit index, logical result key, and reply address. Sender identity/generation and generated-send versus environmental origin are retained separately from these application fields; an ingress payload cannot appoint itself a trusted sender. Projection state holds two saved upstream envelopes, two pending downstream request references with expected peer, and at most four accepted logical keys.

| Ordinary instruction mode | Local rule |
| --- | --- |
| Send/request | Derive a fresh request from caller run and commit; save its envelope and expected peer in the selected pending slot; emit the immutable message intent. |
| Receive/capture | Consume an admitted request and retain its envelope in the selected saved slot for an ordinary reply. |
| Receive/result | Require generated-message provenance, expected peer, exact pending correlation including caller generation, and a not-yet-accepted logical key; add its value and retain the accepted key in ordinary projection state. |
| Send/reply | Copy a saved upstream envelope and a selected bounded register value into an ordinary Send intent. Its later dispatch, admission, acknowledgment, and receipt use the existing transitions. |
| Emit/effect | Copy the saved upstream envelope onto the unit resource intent so evidence can relate its operation key and arguments to the public request path. Application and receipt remain separate events. |

These modes remain one bounded instruction per turn and are committed/replayed with the same rules as other ordinary computation. They expose protocol data and checks; they are not types for coordinators or supervisors. Legacy instructions with no mode retain Phase 1 behavior. Reply routing is explicit: the late-generation challenge uses a service address so an old, not-yet-admitted reply can reach a new run's inbox and be rejected by correlation. An already admitted transport retains its original run binding. Exact-run routing would supply an additional earlier rejection and is not substituted for this challenge.

The reducer may read its program, projection, selected admitted observation, and finite captured metadata. It cannot inspect the external world, another subject's mutable state, or checker facts. Runtime transitions cannot branch on Control/Supervisor/Worker names. Request/reply/aggregation rules must apply to any ordinary program using those operands.

A control removing generation correlation must admit wrong old-request consumption while a current receive is otherwise eligible. A logical-duplicate control must allow a second contribution from a distinct transport with the same logical result key. An origin control must preserve tag/correlation/value while bypassing only provenance protection. The corresponding unmodified monitor must detect each bad event; an unrelated capacity or closure rejection does not qualify it.

## Challenge windows and expected outcomes

| Window | Expected behavior | Evidence required |
| --- | --- | --- |
| Inner waits; peer is ready | Peer commits useful work while inner remains waiting | Search witness with both conditions in the same state/edge |
| Lost acknowledgment; receiver closes/retries; retransmission | Original admission remains bound to old run; new run executes fresh work | Combined suffix search with duplicate event and actual generation-2 commits |
| Worker result admitted before coordinator closure | It can be consumed before closure, but afterward stays pending; stale proposal cannot commit | Both relevant orders or a trace plus rejected stale commit; no claim of eventual handling |
| Old result during eligible new-generation receive | Old result cannot contribute; a correctly correlated current result can | Reachable overlap and both attempted old consumption and accepted current consumption |
| Applied effect without receipt, then closure/retry | World retains application; old receipt settles only old intent; new run executes independently | Separate application/closure/retry/receipt events, not an atomic helper |
| Two transports for one branch result | At most one contribution; another branch remains usable | Actual duplicate logical key with different transport IDs and healthy alternate branch |

Searches may replay disclosed setup prefixes before enumerating suffix orderings. Each setup must start from immutable fixture configuration and execute ordinary transitions plus the same monitors. The resulting monitor state is part of the BFS starting node. No direct state fabrication can serve as reachability evidence. Report setup length separately from suffix depth, and state which challenges are covered in combination by each alphabet.

## Claim categories and stopping point

Searched safety concerns only the listed finite fixtures, alphabets, reachable seeds, and depth bounds. Successful reachability shows one permitted execution. Atomic authority, trusted descriptor provenance, complete finite retention, and resource key retention remain assumptions. General recursive closure, contextual equivalence, unbounded safety/liveness, real durability, dispatcher fencing, transfer, dynamic creation, and cleanup custodians remain untested or excluded.

Any discovered counterexample is retained with its trace and source identity. Correct the model or narrow the claim explicitly; do not silently exclude the offending input. This phase stops at the independently reviewed qualification package.
