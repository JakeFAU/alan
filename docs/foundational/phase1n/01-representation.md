# Phase 1N public construction representation freeze

Status: N-representation-1, successful public construction preflight; independent
construction review pending. This freezes representation, not correspondence,
all-schedule progress, a lifting theorem, or complete manuscript consistency.
The governing behavior remains [N-contract-1](01-contract.md).

## Public representation and committed policy

`examples/callablejoin` is an ordinary public API consumer. `New` selects unchanged
S through the historical public constructor, discards that constructor's parent
and provision, builds one new J, and provisions the sole fixed O root. `NewJoin`
is the lower-level authoring builder accepting candidate and S template values;
the qualified `New` path always supplies original S. `NewOuter` accepts only J's
public template. No second wrapper, peer, sibling or reporting channel exists.

J's public port is `public`, peer `caller`, obligation `work`. Its request uses
the original `{id,value,mode}` schema, identity
`4afdfc3de16677b6790fd35845ab8be94031aa75aba05ffad97c67738141648c`.
The scalar integer fields remain unrestricted int64 kinds. `New` rejects inputs
outside id{1,2}/value{0,1}; mode retains MaxBytes8 and the original four-value Enum.
Those concrete input checks are distinct from scalar-schema validation.

The closed report schema has identity
`352f069279345ee2019a6582604fd376162433aa6d8cd63a885b7e23335fe726`:

| Field | Retained schema |
| --- | --- |
| `id` | Original integer kind |
| `classification` | String, MaxBytes32, Enum `answer`, `candidate_failed`, `candidate_missing_reply`, `invalid_outcome` |
| `answers` | List, MaxItems1, MaxBytes192, exact original `{id:int,value:int}` element record |

Canonical tagged record arrays use 2 bytes for `[]`, 114 bytes for a finite-profile
answer, and at most 152 bytes for any pair of int64 values (two MinInt64 values).
The 192-byte list limit therefore accommodates the original integer schema,
without narrowing its field kinds. The entire schema-valid report is at most
386 canonical bytes, even allowing its longest tag with one full-int64 answer;
that combination may be semantically invalid. Cardinality is a policy obligation:
`answer` with an empty list is schema-valid and must be rejected by O.

The compiler initializes J's declared `answers` collection at pc0. J receives its
request at pc1, commits original e then S at pcs2/3, and selects S then e through
original handles at pcs4/5. Classification begins only after both consumptions.
The explicit S guards dominate all candidate classification. Payload reads are
dominated by response-presence checks. Only a verified candidate answer appends
one record. All other branches retain the empty list. A single common Send
commits the actual diagnostic before Success/Failure. Neither classifier branches
on requested mode; mode is only forwarded to the original candidate.

O commits its original J handle at pc0 and consumes it at pc1. Presence at pc2
and report-id correlation at pc3 dominate report interpretation. Its retained
bounded ForEach at pc10, next at11, and loop at22 bind only existing answer rows.
It starts `answer_seen=false` and `answers_valid=true`, detects repeated rows,
checks original id/complement inside the loop, requires `answer_seen=true` for
an answer and `answer_seen=false` for a diagnosed failure, and checks the exact
public status/reason/tag combinations. The loop has a durable cursor snapshot
only while active; it does not serialize an inner operation handle.

The actual final tag is committed into O's `decision` register. Invalid public
evidence commits `invalid_join_outcome`; an honest defensive diagnostic commits
`invalid_outcome`. The final branch selects O's End from that retained decision.
O cannot inspect J's private child evidence. Classification, Send commitment,
reply admission, J End/freeze, invocation publication/delivery, O's selected
consumption, decision write and O End/root publication remain separate facts.

## Exact artifact identities and operational bounds

Full source hashes, original identities, byte sizes, code inventories and sample
metrics are in [artifacts-and-samples.json](evidence/preflight/artifacts-and-samples.json).
A/B template and helper identities match historical Phase 1I records; S matches
the original constructor byte-for-byte. Normalizing only candidate template IDs
makes J[A]/J[B] program bytes equal; normalizing only J template IDs makes the O
program bytes equal. Catalogue differences preserve original transitive programs.

| Artifact | A binding | B binding |
| --- | --- | --- |
| J template | `2e7f0654c6c3e62287d76dff5144674784029e3d038714eabfe17f511ca34a37` | `8f537a4abec7a501f6b6875db112da8bd2d0bcc8ecbd7d16d2c97d7e8424a942` |
| J program | `fd9d3e82946594e8114a84a156f56594d4f9dac6448656ff86e090aadb1b32eb` | `e87d0f268da24f572e9f8fd14a194a8dd96551838708c186dadd9ac5a3b68773` |
| O program | `272c68dfe7c2b422c9819a8caf80b0277f35fb91859c9ddf80915cf46fdc4725` | `5e8fee04fd8855c501c1c20b87fdc4c2a1f96d6f4152baba3a7b5a48acb45621` |

Both J programs contain 58 instructions/25,387 bytes; both O programs contain
64 instructions/22,926 bytes. Original A has 13 instructions/6,762 bytes, B has 16/
9,533, helper 7/3,951, and S 3/2,632. Each retained program is below the unchanged
1MiB program ceiling.

The following are conservative bounds derived from these fixed artifacts under
stable accepted bindings, no closure, no restart/retry, no second pending request,
and the declared finite input profile. They are not observations extrapolated
from model rank, and do not bound arbitrary replacement implementations.

| Deployment | Fixed runs | Dynamic charges (I,R,E) | Greatest depth | Static instructions | Conservative local commits | Ordinary dispatches |
| --- | ---: | --- | ---: | ---: | ---: | ---: |
| O/J[A]/S | 1 | (3,3,0) | 2 | 138 | 139 | at most 9 |
| O/J[B]/helper/S | 1 | (4,4,0) | 3 | 148 | 149 | at most 12 |

Each program is instantiated once. All control flow is forward except O's one
MaxItems1 loop; its `next` instruction executes at most twice, adding one to the
sum of code lengths. Untaken branches make these ceilings intentionally loose.
Blocked receives add no local commits. There is one admission dispatch per
invocation, at most one reply dispatch and one outcome-delivery dispatch, hence
at most three dispatches per original invocation. Failure/absence can remove a
candidate reply; pumping publication itself is not an extra ordinary dispatch.
The public consumer's MaxSteps512/MaxDispatch64 exceed these derived bounds.

The artifact register unions contain at most 18 registers (J), 16 (O), 12 (B), 4
(helper), 3 (A), 1 (S). In this finite stable profile, scalar values, exact operation
handles and the single tagged answer list each fit within 512 bytes; keys fit 64.
Allowing 32 register slots at 576 bytes each plus 1KiB for the sole cursor and
projection envelope gives a conservative 32KiB retained-projection bound.
This accounts for the cursor snapshot, not only the final register map. It is
below the unchanged 1MiB data limit. Request/report payloads fit 512 bytes.
Observation/output limits remain the unchanged 1MiB defaults; the retained raw
sample envelopes and successful replay witness their use here, rather than
claiming a universal protocol-envelope bound from observed sizes. The largest
observed projection was 1,548 bytes, separately recorded as a sample maximum.

The common entitlement remains I=R4/depth3/E0/fixed1; physical MaxInFlight1.
J admission costs (1,1,0) before its children, leaving allowance 3. e and S enter at
depth 2; B's unchanged helper enters at 3. Commitments cost zero; actual accepted
singleton admissions charge once. These counts do not prove the arbitrary S
admission-order projection or all-mode progress obligations, which remain later
reference/correspondence work.

## Public preflight evidence and limits

Four separate-module samples ran under both normal and race instrumentation:

| Sample (id1/value0) | Local commits | Dispatches | J/O result | O decision |
| --- | ---: | ---: | --- | --- |
| A answer | 78 | 9 | succeeded/member_succeeded | answer |
| B answer | 85 | 12 | succeeded/member_succeeded | answer |
| A fail | 62 | 8 | failed/member_failed | candidate_failed |
| B absent | 74 | 11 | failed/member_failed | candidate_missing_reply |

Every sample retains init/artifacts/run/replay/inspect outputs. The failure
samples verify a present diagnostic on the publication for O's original J handle,
a selected invocation observation for that handle, one committed O decision
instruction, and the matching root outcome. These are genuine deliberate failed
invocations, not successful invocation envelopes labeled as failures by a host.
The test also checks charges, report id and cardinality. Replay is shared-runtime
consistency evidence, not an independent semantic oracle.

[The evidence index](evidence/preflight/index.json) records source identities,
argv, sanitized environment, exact leaf statuses and exits. Initial failures are
retained: missing callable artifact (intentional RED), a test using a nonexistent
public list accessor, and a construction assertion omitting the compiler's
empty-list initializer. The latter two were test mistakes, not unsupported APIs
or semantics changes. No production/interface obstruction remains in this scope.

A mistakenly broad root regression was stopped at the controller's instruction;
its exit 1 and unfinished adaptive leaf are preserved and excluded from acceptance.
It is a verification-scope mistake, not a semantic failure or a green suite.
No new reference output was used to define this construction. The independent
reference author must not use these outputs to define its oracle. Shared compiler,
schema, runtime/replay and historical public authoring patterns remain explicit
assumptions. Full raw audit, adversarial controls, pending maximality, environment
alignment, model/artifact correspondence, lifting argument and synchronized
manuscript reconciliation remain later gates.
