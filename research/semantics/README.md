# Bounded executable semantics research

This is a **test-only research model**, not an importable runtime, SDK, provider,
transport, or persistence implementation. It exercises the frozen
[operational specification](../../docs/foundational/phase1/01-operational-spec.md);
[qualification and limitations](../../docs/foundational/phase1/02-qualification.md)
are maintained separately.

Run with the repository's Go 1.27.3 toolchain and standard library:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache go test ./research/semantics -count=1
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache go test ./research/semantics -run TestQualification -count=1 -v
```

`model_test.go` holds the finite interpreter and serialized transitions.
`checker_test.go` holds independent transition monitors and fixed-version replay.
`explorer_test.go` performs deterministic BFS; `qualification_test.go` supplies
small scenario alphabets and the static three-subject Control witness.
`contracts_test.go` contains focused observable-contract and regression tests.
There are no agent callbacks, goroutines, clocks, network calls, or dependencies.
One selected instruction executes per prepared/committed local turn.

Fixtures bound generations/epochs to 2, committed instructions per run to 8,
live continuations to 2, proposal slots to 2, intents per run to 2, attempts per
intent to 2, retained inbox entries per run to 4, and data values to integers
0–8. There are two data registers and two effect-reference slots. Programs are
fixed per subject; the largest program has 9 instructions. Most searches use
one subject; Control uses three statically provisioned subjects. Instructions
can fork fresh continuation identities, with total identity growth also bounded
by the commit limit. All ownership fixtures use principal `owner` and revision
7, separately from generation and activation epoch.

An Emit/Send instruction writes its derived intent ID into a selected reference
slot. A receipt Receive requires receipt-kind evidence matching that exact slot,
as well as its label. An ordinary Receive requires message-kind evidence.
Transport/admission identities distinguish ingress raw IDs from generated Send
IDs, and inbox identities further distinguish messages from receipts. Arbitrary
raw ingress strings remain allowed, including strings that look like intent or
receipt IDs. A Send receipt records acknowledgment of its own exact admission;
ordinary ingress acknowledgment is an independent event. External resource
mutation, admission, and receipt recording remain separate transitions.

Every `QUALIFICATION` line is deterministic JSON containing bounds, event
alphabet, state/transition/rejection/revisit counts, shortest-depth boundary
counts, reached witnesses, and any shortest counterexample trace. Accepted
transitions include stutters and violations; states count unique admitted search
states including the initial state, excluding a violating successor. `MaxDepth`
is the largest discovered shortest depth (including a violating successor).
`Boundary` counts unique states at the configured depth limit that are not
expanded. Hard state-cap exhaustion is an error. Monitor facts, full journal
histories, proposals, and external state are all included in search identity;
monitors run before deduplication. A finite alphabet may reach a fixed point
before its depth limit; that is not a claim about the omitted events.

Three broken-guard controls must produce actual violations of the same monitors:
stale epoch commit, authorization after closure, and service-message rebinding.
Two additional policy counterexamples distinguish cancellation from retirement
and preserve a preexisting failed outcome under late cancellation. A finite
65-event Control execution is a reachability witness, not exhaustive exploration
of every three-subject interleaving or a fairness guarantee.

Replay rebuilds version, epoch, sequence, data/reference slots, frontier,
consumed observations, immutable intent descriptors, terminal status and frozen
notification from initial state and recorded selections/lifecycle events.
Pending inbox entries, admission bindings, dispatch/receipt ledgers and delivery
state are checked separately, not reconstructed. Replay must not mutate the live
machine or dispatch historical work.

Atomicity is assumed by the transition rules. External idempotence retains keys
for the entire fixture; the non-idempotent profile deliberately permits repeated
physical mutations. Neither proves exactly-once external work. The dispatcher
uses trusted current owner/revision checks; dispatcher incarnation fencing and
endpoint fencing are not modeled. Dynamic creation, ownership transfer,
capability delegation/revocation, group cancellation, cleanup custodians,
distributed serialization, durability, migration, retention expiry and unbounded
liveness are excluded. All terminal outcomes suppress never-authorized ordinary
intents; already-authorized attempts may still apply and settle.

## Phase 1B bounded composition qualification

The additive `composition_*_test.go` files define ordinary bounded envelopes,
independent public-contract checks, replayed seed traces, suffix searches, and
leaf/nested completion witnesses. All programs use the same `step`, `reduce`,
`inspect`, and `replay` paths. Phase 1 fixtures and their 15 evidence records stay
unchanged. New immutable programs use version `v1b`.

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test ./research/semantics -run '^TestComposition' -count=1
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test ./research/semantics -run '^TestCompositionQualification$' -count=1 -v
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test ./... -count=1
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test -race ./... -count=1
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go vet ./...
```

`COMPOSITION` records disclose each complete fixture configuration, SHA-256
configuration/scenario identity, exact setup events, finite suffix alphabet,
separate setup length/depth bound, search counts, rejection reasons, combined
witness suffixes, and shortest mutant suffixes. Prepend the disclosed seed to a
suffix to reproduce its full trace. Every seed starts at `newMachine(config)`
and uses transitions plus the same monitors; monitor facts enter suffix state
identity. Rejected events also update witness-only checker facts, so a witness
can require a rejected old/duplicate result and later legitimate contribution in
one trace. These facts cannot affect ordinary transitions. Every successor is
checked before deduplication; a hard-cap hit fails the test.

New bounds are four static subjects, two generations/epochs, twelve commits,
two continuations/proposals, three intents, two attempts, eight inbox entries,
and values 0–8. Ordinary state adds two upstream envelopes, two downstream
request references, and four accepted logical keys; programs are capped at
twelve instructions. Fresh exploration uses depth eight; focused suffixes use
depth ten; every scenario has a hard cap of 5,000 unique states. Narrow alphabets
and long explicit setup traces limit claims to their disclosed suffix windows.
Completed leaf/nested traces are separately labeled reachability-only.

The source-manifest record hashes `go.mod` and every
`research/semantics/*_test.go` in sorted repository-path order, with unsigned
64-bit big-endian byte lengths before each path and raw file content. It lists
per-file SHA-256 values and the aggregate SHA-256. Generated evidence is excluded
to avoid a circular identity. Configuration/scenario identities hash their Go
`encoding/json` canonical serialization (sorted string map keys). The source
identity is emitted only in Phase 1B records.

See the [public composition contract](../../docs/foundational/phase1b/01-composition-contract.md)
for permitted behaviors, provenance, challenge windows, and exclusions. This
qualification does not claim general contextual equivalence, unbounded recursive
closure, fairness, durability, or production readiness.
