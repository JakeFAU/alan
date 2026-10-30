# Phase 1M C_se reference — review freeze

Status: construction checks completed; independent reference review is pending.
`evidence/model/reference-r1.json.gz` freezes sources, contract, lineage, exact
commands and outputs. No new product census, replacement acceptance, artifact
mapping, or runtime correspondence is claimed. The independent source-only
derivation remains in `.superpowers/sdd/00-plan-phase1m/task-2-derivation.md`.

## Claim and fixed domain

Keep separate: (1) X refines K1 in the declared local environment; (2) C_se[K1]
satisfies G_answer; (3) C_se[X] satisfies G_answer, for X=A/B. A failed reference
premise is not a substitution counterexample. Truthful prefixes do not establish
component conformance or progress.

Positive configurations are exactly id{1,2} × value{0,1}, mode answer, I=R3,
depth2, E0, one fixed parent: four per K1/A/B product. Original e/S are depth1
singletons; optional h is depth2. K1 independently retains local/helper choice
in every supported mode. A/B/helper/S remain original. Fail/absent/pending and
scarcity are focused controls, not additional positive configurations.

Assume stable generations/revisions/grants, valid original template/reply/schema
bindings, no retirement/cancellation, eventual eligible local/protocol service,
and sufficient finite operational bounds. Concrete capacity1 is distinct from
cumulative charges. Derive new concrete instruction/dispatch/byte limits from
artifacts after freeze; neither rank nor inherited measured limits qualifies them.

G_answer requires both original publications, deliveries and unique actual
selected consumptions by their owning parent; succeeded/present e=(original id,
1-original value), S=(original id,0); parent and root/aggregate success; original
identities/obligations and resource compliance. Predicted dynamic spending is
A+S2 versus B+helper+S3, fixed1/E0, under one worst-case allowance. This is not
exact-debit equivalence or a new observed result.

## Parent, protocol, and independent oracle

| Context | Commit order | Consumption order | Decision |
| --- | --- | --- | --- |
| Historical C_es (`Join*`) | e then s | e then s | guarded strict join |
| C_se (`ConsumerPolicy*`) | e then s | s then e | same guarded strict join |

One fixed parent retains both original handles. Both intents commit before the
first await; admissions/work may interleave. e receives original {id,value,mode};
S receives original {id,value}, without mode or candidate-result substitution.
C_se stages are0 commit e,1 commit s,2 await s,3 await e,4 decide,5 End,6 terminal;
root publication is separate. Every decision, including Failure, requires both
actual consumptions. After S consumption, unproduced/published/retained-but-
unconsumed e remains required and the parent stays open. Candidate pending has
no publication or release input. S-pending is a separate control outside the
S-answer product: completed e cannot discharge the first await.

The strict predicate guards status/presence before payload reads, then checks
correlation/complement/S-zero. C_se adds no reason-based classification. Preserve
exact reasons: absent reply after local Success gives failed/missing_reply;
local Failure gives failed/member_failed. Parent/root outcomes remain distinct
from child envelopes, and the root has no outgoing reply obligation.

Retain separate commitment, first authorization, admission, request consumption,
K1 choice/private steps, reply commitment/authorization/admission, member End,
publication, delivery, selected consumption, parent decision/End and root
publication. The original binding-qualified reply exception requires matching
protocol/message kind/entry member/caller subject+generation/entry obligation;
it permits first authorization after member End but excuses no revision or
generation mismatch. Invocation authorization has no reply exception. Final
admission still checks current source/root scope. Captured replies and applicable
descendant joins remain publication obligations.

The oracle retains per-operation protocol stages and original envelopes, ledger,
parent decision/End and root publication. Independently of the model's stage
mapping, it requires e commitment before S commitment, both commitments before
first S consumption, S consumption before e consumption, and both consumptions
before any decision. It compares actual retained envelopes without rewriting
facts/order/consumed flags. Its guarded predicate is separate from the model's
Operation predicate. `CheckJoinComponent` remains the unchanged candidate/helper
checker; truthful wrong-id failure can pass prefix checking while failing
component conformance and G. Incomplete evidence cannot establish G.

Raw authentication remains external: link original commitments, actual handles,
admissions/requests/bindings, authorization snapshots, publications, delivered
observations, selected history and consumed sequence before one consistent e/h/s
renaming. Never infer consumption from delivery or invent handles from expected
sequence tuples. The new consumer/artifacts/records were not inspected at freeze.

## Narrow reuse and preservation obligation

`consumer_policy.go` adds fixed C_se entry points for configurations, initial
state, successors, state/prefix/G checks and exploration. It reuses reviewed
JoinConfig/Event/State/Census storage; the new census wraps those fields with a
C_se label. A closed internal two-value consumption selector factors receive
and state-order guards in `invocation_join.go`, independently stated monitor
order guards in `invocation_join_oracle.go`, and routing/pending classification
in `invocation_join_explorer.go`. Historical public wrappers always select C_es;
their signatures and result fields remain unchanged. No generic context DSL or
transformed trace is introduced.

This controller-authorized factoring supersedes the preparation note's purely
additive preference: it avoids copying protocol/monitor/BFS machinery at the
cost of shared-source regression risk. Exact old/new sources and hashes are in
the freeze. Specializing the internal selector to candidate-first recovers the
old receive gates, state predicates, prefix guards and pending classification;
all other transition/check bodies are unchanged. Existing TestJoin construction
checks protect that claim now. After review, rerun the focused existing
InvocationJoin reference/replacement/local-subset qualification, since shared
internals changed; historical results alone do not qualify these new sources.

Phase1I State/Initial/Successors/CandidateGraph/CheckTrace/local relation remain
unchanged. The temporary RootPC=await view only exposes original receives;
parent gates filter them and obsolete caller fields are restored before retaining
the successor. No old root decision, peer/wrapper/closure/effect or sequential-S
input transform survives. A13/B16/helper7 private control is unchanged. Preserved
helper-refusal/no-reply and payload-leak counterexamples still motivate direct
independent envelope checks rather than shared-graph agreement alone.

Initial demand is {e,h,s}; commitment is free, accepted singleton admission
spends once (1,1,0), refusal excludes unspent demand, local choice excludes h,
and no spending is refunded. Spent+remaining demand stays within I/R3 under the
positive profile. For each answer request/A/B, reuse Local=true, Shape=local,
EntryDepth1, Depth2, I=R3, Peer=true: S admission projects to arbitrary-timing q
debit, its other actions to local stutters. Explicit authorization projects to
stutters relative to the older coarse relation and is checked separately here.
Bilateral environment readiness and separate Closure=false progress are required.
This projection covers debit interference, not S payload/lifetime, parent joins,
root publication or refined authorization. Current answer local requalification
remains gated; it does not certify future all-mode C_reason progress.

## Finite bound and exploration

For a fixed input, fields are unchanged: I/R0..3, E0, parent stage0..6, decision/
status in {empty,succeeded,failed}, root bit, six authorization values0..2; e/h/s
with ticket0..3, admission/request-stage0..2, PC0..8, choice0..2, reply0..3,
freeze0..2, seven Booleans, three id/value pairs in {0,1,2}×{0,1}, four statuses,
ten outcome reasons, seven admission reasons, empty reply reason. Private PCs
are0..15/0..6; sixteen registers range0..2. Unused state is fixed. Conservative
Cartesian model and prefix-monitor bounds are respectively:

    16*7*9*2*3^6*16*7*3^16 * (4*3*3*9*3*4*3*2^7*6^3*4*10*7)^3
    16*9*2 * (2^14*7*6*3*481)^3

Multiply them for the model×monitor key. These are loose bounds, not reachable
counts. The constant selector is not mutable state. All future-relevant component
monitor memory is contained in the per-operation prefix records; a representative
trace alone never replaces monitor identity.

Rank is100*stage+candidatePrivatePC+helperPrivatePC. Stage sums ledger, parent
stage, root bit, authorization values and per-operation ticket/admission/request-
stage/PC/choice/reply/freeze plus Enabled/FailureReady/Published/Delivered/Consumed.
Each operation contributes <=27; cap=100*(3*27+6+6+6*2+1)+15+6=10621, initial100,
hence <=10521 progressing edges. Semantic stages are monotone; private branch/
Jump PCs advance; factor100 exceeds private range21. Reversing receive labels
preserves this source argument, checked on every generated edge including
revisits. Failed rank leaves cycle status unresolved. This is not a concrete
instruction budget or an executed census.

Full BFS has no failure pruning or cutoff, checks successors before deduplication,
retains every maximum and shortest trace per failure code, and reports model and
model×monitor states separately. C_se pending requires e unmatched/no publication,
S consumed, parent stage3/open; delivered-but-unconsumed S is insufficient. G_answer
excludes pending maxima. Fairness excludes infinite idle/retry/restart-only
suffixes. An abstract publication/delivery split is not a concrete checkpoint.

## Evidence and next gate

The preserved RED checks expose old C_es ordering. Normal/race each pass50 selected
construction leaves:26 new and24 existing TestJoin leaves, with no fail/skip.
Directed K1 local/helper paths and synthetic oracle controls are not exhaustive
product results. Vet, formatting, discovery and source/diff checks are recorded.

Stop for independent reference review. After approval: C_se[K1] first, then A/B;
current local answer checks and affected historical qualification; every compiled
instruction mapped, including pure setters/branches; concrete operational bounds;
then raw runtime qualification. C_reason, arbitrary contexts, general congruence,
exact-debit equivalence, universal compiler correctness and effectful/distributed
recovery remain outside this freeze.
