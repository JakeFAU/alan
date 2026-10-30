# Phase 1M C_reason reference — construction review freeze

Status: independently derived reference implemented; independent review is pending.
No C_reason product census, all-mode local qualification, replacement acceptance,
consumer-artifact inspection or runtime correspondence is claimed by this freeze.
The historical `01-contract.md`, C_es/C_se APIs and primitive sources are unchanged.

## Claim, domain and assumptions

Keep separate: X refines K1 under the applicable local environment; C_reason[K1]
satisfies G_outcome; C_reason[X] satisfies G_outcome for X=A/B. The declared census
is id{1,2} × value{0,1} × candidate mode{answer,fail,absent,pending}, original answer
S, I=R3/depth2/E0/fixed1:16 configurations per product,48 total, K1 first. The
reference independently permits local/helper choice in every mode. A property
must hold on every permitted maximum; a favorable branch is insufficient.

Assume stable generations/revisions/grants, valid original bindings and schemas,
no retirement/cancellation, eventual eligible local/protocol service and sufficient
finite operational bounds. Physical capacity1 is a concrete-sample condition;
model rank is not an instruction/dispatch/byte allowance. No extra S-mode,
capacity, depth or cancellation crossproduct, pending-release input, effect,
retry, fallback, reporting channel or wrapper is included.

`ConsumerReasonEntitled` includes all four modes with the same worst-case e/h/s
allowance. Historical answer-only `JoinEntitled`/`Entitled` are unchanged and
cannot suppress G_outcome or local mode-progress checks. Predicted spending is
2 for A/local K1 and3 for B/helper K1 in every mode: helper precedes mode action,
and original S always admits. This is not exact-debit equivalence.

## Fixed parent and independent classification

Both original intents commit e then s before the first await; their admission/work
may interleave. S receives original id/value, never the candidate complement.
The parent retains both handles, consumes s then e, then classifies their actual
retained envelopes. Even invalid S must not bypass the original candidate join.

Keep historical parent stages0 commit e,1 commit s,2 await s,3 await e,4 classify
then decide,5 End,6 terminal. Classification is a distinct retained event within
stage4; root publication remains a separate event after End. Empty tag is neutral,
not a completed classification. The policy does not inspect requested mode.

First require consumed S succeeded/member_succeeded, present, original id/zero.
Then apply the following table to consumed candidate evidence; invalid S always
selects the defensive branch. Guard possibly absent payloads before reading them.

| Actual e with valid S | Classification | Parent decision |
| --- | --- | --- |
| succeeded/member_succeeded, present, original id/complement | answer | Success |
| failed/member_failed, absent | candidate_failed | Failure |
| failed/missing_reply, absent | candidate_missing_reply | Failure |
| any other completed evidence | invalid_outcome | Failure |

Events distinguish `parent_classify` (tag only), `parent_decide` (selected terminal
status), `parent_terminal` (committed End), and `root_publish` (terminal status and
root reason). A classification write alone does not prove decision/End occurred.
A neutral setter and intermediate pure branches may map to justified stutters;
actual compiled mappings remain unopened until reference review approval.

Original e/S envelopes are never rewritten. Parent deliberate Failure publishes
failed/member_failed, including when e is failed/missing_reply. No propagation of
the child's missing_reply into the root reason is required or permitted here.
An absent request to an actually failing component classifies candidate_failed:
truthful prefix/policy execution, but a departure from K1 mode conformance and
G_outcome for that request. invalid_outcome is defensive, not a conforming label.

## Guarantee, monitors and intended pending maxima

Every prefix preserves original identity, provenance, one-charge admissions,
authorization/admission separation, reply/publication/delivery/selected-consumption
boundaries, once-only original joins and payload safety. Reply authorization retains
the reviewed binding-qualified exception; invocation authorization has no exception.
Actual raw handle/binding/selected-history authentication remains external.

The monitor stores original per-operation envelopes and consumed flags. It derives
classification independently of the model's Operation predicate, requires a unique
classification event after both joins, then separately checks decision, End and
root publication. Prefix validity, K1 component conformance and G_outcome are
separate results. Requested mode determines oracle expectations, not observed tags.

- Answer: both correct succeeded/present outcomes consumed, answer classification,
  matching Success/End/root success; G_answer holds.
- Fail: both consumed, e failed/member_failed absent, candidate_failed,
  matching Failure/End/root failed/member_failed; S valid.
- Absent: both consumed, e failed/missing_reply absent, candidate_missing_reply,
  matching Failure/End/root failed/member_failed; S valid.
- Pending: e consumed its initial request and any chosen successful helper, reached
  its designated unmatched receive, and has no reply/freeze/publication/delivery/
  consumption. S is published/delivered/consumed; parent stage3/open, tag neutral,
  no decision/End/root publication. Delivery of S alone is insufficient.

New monitor memory records the actual candidate `pending_receive` event; its
absence cannot be replaced by a safe finite prefix. No subsequent candidate action
or new helper commitment is accepted after that unmatched receive. The explorer
must establish maximality independently; a standalone trace verdict cannot prove
no other action is enabled. Component correctness remains separately checked.

## Narrow reuse and local progress

`consumer_reason*.go` is additive. The model delegates non-root successors to
reviewed C_se, discards its parent actions and emits the four explicit parent
events above. The independent monitor delegates original non-root protocol checks
and unchanged End/root checks, while independently checking classification and
decision. No old event, flag, order or envelope is transformed. C_es/C_se public
functions, JSON types, state guards and contracts remain unchanged.

The BFS follows the reviewed exploration pattern with new model×monitor keys;
`CheckJoinComponent`, K1, `CandidateGraph`, `CheckLocalRelation`, `ExploreGraph`,
`CheckTrace` and their sources are reused exactly. Shared protocol machinery and
candidate graphs are assumptions, not independent compiler implementations. This
model author has not inspected the new consumer or its artifacts/runtime output.

Local requalification after review uses A/B ×16 requests with Shape=local,
Local=true, EntryDepth1, Depth2, I=R3, Peer=true, Closure=true for the relation;
Closure=false separately for progress. S admission projects to the single q debit
at arbitrary timing, all other S actions to stutters; explicit authorization is a
stutter relative to the older coarse relation and checked separately here. This
projection covers debit interference, not S payload/lifetime or parent joins.

The qualifier retains every actual local maximum around unchanged ExploreGraph,
checking exact mode envelopes, first request, original successful consumed helper
for B, q spent, no closure/refusal and exact charge2/3. Answer requires correct
present succeeded/member_succeeded; fail requires absent failed/member_failed;
absent requires absent failed/missing_reply; pending requires the unmatched receive,
no publication and an open caller. Require nonempty maxima and zero relation,
rank/cycle/cutoff/stuck/safety/trace/maximum failures. Historical answer-only
completion and acceptance of finite safety traces alone do not certify this.

## Finite bounds and fairness

Let O=`4*3*3*9*3*4*3*2^7*6^3*4*10*7`, the reviewed per-operation Cartesian factor.
For fixed input the unchanged C_se model and prefix bounds were:

    M = 16*7*9*2*3^6*16*7*3^16 * O^3
    P = 16*9*2 * (2^14*7*6*3*481)^3

New finite bounds are `5*M` for model (neutral plus four tags) and `10*P` for
prefix monitor (five tags and one pending Boolean); model×monitor is their product.
These are conservative Cartesian bounds, not reached-state counts. No extra
mutable parent stage was added. Future-relevant component-monitor memory is in
keyed operation records; classification and pending memory are keyed too.

Intrinsic rank is `JoinRank + 100*[tag nonempty]`: the classification event strictly
increases the new milestone; all other edges preserve it and inherit old rank.
The cap is10721, initial100, at most10621 progressing edges. Tag values, requested
mode and history length are not rank. Every edge, including revisits, must pass.

For open local progress only e/h are live; w/s have fixed excluded tickets. A loose
state bound is `16*8*3*6*2*4*16*7*3^16 * O^2`: ledger, root PC/status/payload/
presence, peer ticket, private PCs/registers and two operations. Peer PC is fixed7,
closure false, and immutable bindings/unused fields are fixed. Original graphRank
has cap`100*(2*27+6+6+7+7+3)+15+6=8321`: live operation sums, unused tickets,
ledger, root/peer PCs, peer ticket and private PCs; register e[7] remains zero in
unchanged A/B. The qualifier reports actual rank extrema without running here.
The reused closure-enabled relation keeps its original finite domain and checks.

Full BFS has no failure pruning or cutoff, checks edges before deduplication,
retains every maximum and shortest failure witness, and distinguishes model from
model×monitor counts. Strict rank excludes directed cycles in the explored graph;
rank failure leaves cycle status unresolved. Eventual eligible service excludes
infinite idle/retry/restart-only suffixes. Pending is an intended maximum, not
successful termination or evidence that a missing release eventually arrives.

## Review gate and remaining obligations

Construction evidence contains a preserved four-leaf RED probe of the old strict
policy plus final source-bound normal/race checks, exact leaves, exits, environment
and source hashes. Directed K1 paths and artificial adverse histories/maximum
fixtures are construction tests, not exhaustive product or replacement results.

Stop for independent reference review. After approval: K1-first16, A/B16 each;
all-mode local relation/progress; then complete compiled instruction correspondence
and concrete operational bounds, raw selected-observation/classification linkage,
representative runtime controls and independent review. No universal substitution,
general congruence, arbitrary contexts, universal compiler correctness or new
recovery theorem follows. Manuscript reconciliation is deferred until results.
