# Phase 1M qualification

**Result: bounded, conditional preservation for both consuming policies.**
C_se preserves G_answer on its declared answer domain. C_reason preserves
G_outcome on answer/fail/absent/pending, and separately preserves G_answer on
its answer subset. Independent reference premises, local replacement checks,
finite products and concrete samples have separate evidence below. No bounded
substitution counterexample or runtime/model correspondence failure was found.
[Review dispositions](03-review.md), [reproduction](04-runbook.md) and the
[machine index](evidence/index.json) give the source-bound scope.

The unchanged [S-first contract](01-contract.md) and
[reason contract](01-reason-contract.md) are historical pre-census freezes.
Their pending-review wording and the captured READMEs/reports describe those
checkpoints; the later approvals/results are recorded here and in 03-review.md.
The approved freezes are [reference-r1.json.gz](evidence/model/reference-r1.json.gz)
and [reason-reference-r1.json.gz](evidence/model/reason-reference-r1.json.gz).

## Contexts and guarantees

| Context | Original commitments | Actual consumptions | Policy |
| --- | --- | --- | --- |
| Historical C_es (Phase 1L) | e then S | e then S | guarded strict join |
| C_se | e then S | S then e | same guarded strict join |
| C_reason | e then S | S then e | classify retained outcomes after both joins |

These are alternative executions with one fixed owning parent, both original
handles retained, no extra reporting peer/channel, wrapper, retry, release input
or effect. S receives the original request id/value and returns original id/zero.
Each context has one builder parameterized by A/B: guards, policy and allowance
agree, while parent bytes/IDs differ at exactly the candidate author declaration,
invoke target and receive source; B's catalogue also contains its original helper.
Original A/B/helper/S artifacts remain unchanged. Consumer and reference authors
were distinct; shared compiler/runtime, pure primitives and reviewed infrastructure
limit independence. Postfreeze artifact checks supply a separate obligation.

G_answer requires both original outcomes published, delivered and selected/consumed
once by their owner; correct succeeded/present candidate id/complement and S id/zero;
parent/root/aggregate success; original identities, obligations and resource compliance.
C_se covers id{1,2} × value{0,1}, answer: four configurations per product, twelve total.
C_reason adds candidate modes fail/absent/pending while S remains answer: sixteen
per product, forty-eight total. Both use common I=R3/depth2/E0/fixed1. Stable
generations/revisions/grants, valid bindings/schemas, no retirement/cancellation,
sufficient finite operational bounds and eventual eligible service are assumptions.
Physical capacity 1 applies to concrete samples, independently of cumulative charges.

For G_outcome, S must first validate as succeeded/member_succeeded, present,
original id/zero after both consumptions. Actual candidate evidence then determines:

| Consumed candidate, with valid S | Classification | Parent terminal |
| --- | --- | --- |
| succeeded/member_succeeded, present, correct id/complement | answer | Success |
| failed/member_failed, absent | candidate_failed | Failure |
| failed/missing_reply, absent | candidate_missing_reply | Failure |
| other completed evidence, or invalid S | invalid_outcome | Failure |

Possibly absent payloads are guarded. Original reasons remain retained: a genuine
absent candidate has missing_reply, while this parent's deliberate Failure gives
parent/root member_failed. Requested mode predicts component behavior in the oracle;
the parent interprets actual envelopes. invalid_outcome is defensive, outside the
conforming profile. Pending requires the actual unmatched original receive, S
consumed and e required at open parent PC3, with no candidate publication, completed
classification, decision, terminal result or root outcome. Failure/absence are
correct classified failures, not failures of Phase 1L's answer-only claim.

## Independent finite results

The [C_se bundle](evidence/model/se-qualification/index.json) and
[C_reason bundle](evidence/model/reason-qualification/index.json) record K1-first
enumeration before replacements in each normal/race profile. Both retain K1's
independent local/helper choices in every declared mode, check every incoming edge
before deduplication without failure pruning/cutoff, and preserve every maximum
with a representative trace. Maxima count states, not all interleaving traces.

| Product | Configurations | States | Edges | Terminal maxima | Pending maxima |
| --- | ---: | ---: | ---: | ---: | ---: |
| C_se[K1] | 4 | 1996 | 4376 | 8 | 0 |
| C_se[A] | 4 | 944 | 2046 | 4 | 0 |
| C_se[B] | 4 | 1784 | 3842 | 4 | 0 |
| C_reason[K1] | 16 | 6072 | 12880 | 24 | 8 |
| C_reason[A] | 16 | 2680 | 5414 | 12 | 4 |
| C_reason[B] | 16 | 6460 | 13636 | 12 | 4 |

Every maximum satisfies its selected guarantee. State/rank/prefix/component/G
failures, refusals, stuck maxima, cycles and cutoffs are zero; normal/race structures
agree. Reason's twelve answer configurations separately satisfy G_answer after
forgetting only the added classification event. Product rank starts 100 with caps
10621/10721 for C_se/C_reason; every generated edge increases rank. Eventual service
excludes infinite idle/retry/restart-only stuttering; rank is not a runtime budget.

Local conformance and progress are checked separately from the products. C_se's
current answer subset covers four requests per A/B; C_reason requalifies all sixteen
per replacement with the unchanged closure-enabled relation and separate
Closure=false progress. Its all-mode results are:

| Replacement | Relation pairs | Component/environment checks | Stutters | Open states/edges | Terminal/pending maxima |
| --- | ---: | ---: | ---: | ---: | ---: |
| A | 884 | 468/1688 | 180 | 376/544 | 12/4 |
| B | 2320 | 1328/4168 | 496 | 824/1244 | 12/4 |

All relation/progress/maximum failures are zero. Actual maxima check exact envelopes,
initial request, original consumed helper where chosen, q debit and designated
pending receive. Open-local rank starts 1400, cap 8321. Original S admission projects
to one arbitrary-timing q debit, its other actions to stutters; authorization is a
stutter relative to the older coarse relation and checked separately in the product.
This projection does not establish S payload/lifetime or parent joins. No full
historical Phase 1I census was rerun.

## Artifact correspondence and resource entitlement

The reviewed mappings cover 89 C_se and 143 C_reason instruction occurrences, including
both bound parents and unchanged A/B/helper/S. Reason checks all 32 input/binding
builds and 28224 post-join envelope pairs per binding, omitting absent payload keys
and reaching every guard/setter/End. Pure guards/setters/jumps have explicit stutter
mappings. These are finite artifact checks, not universal compiler correctness.

A reason classification Set writes the tag and reaches its adjacent End in the
same committed history step. Its ordered abstract expansion is parent_classify
then parent_decide; no separate runtime checkpoint exists between them. Executing
End is later, followed by root settlement. Pending arrival is authentic A PC9→Receive PC11
or B PC10→Receive PC12, with original request/path/binding and B's consumed helper await PC2;
it expands to private_step then pending_receive without another receive consumption.

| Artifact-derived upper bound | C_se A/B | C_reason A/B |
| --- | ---: | ---: |
| Total local commits | 25/35 | 28/38 |
| Counted dispatch calls | 6/9 | 6/9 |
| Common conservative bytes | 64 KiB | 128 KiB |

Configured limits are 200 steps, 200 dispatches and four 1 MiB byte caps. Dispatches
count initial admission with atomic request insertion, original reply delivery and
ready original outcome settlement/delivery; root settlement is uncharged. Failure,
absence or pending can omit later calls. Reason's largest retained parent is 21703
bytes/17 registers; conservative bound-program/projection/envelope maxima are
74951/28042/53056 bytes. Bounds derive from forward paths and finite pure-profile
representations, independently of sample measurements and model rank. A/local K1
spend dynamic I/R=2; B/helper K1 spend 3 under the same worst-case limits 3, E0/fixed1.
Exact-debit equivalence is not claimed.

## Concrete evidence and source scope

Both [S-first runtime](evidence/runtime/se/index.json) and
[reason runtime](evidence/runtime/reason/index.json) authenticate original emitting
commitments→handles→admissions/requests/bindings→publication/delivery→selected
history/consumed sequence before normalization. Independent Python audits establish
raw links without Go decoder/model/reducer calls; they reuse hash-pinned inert
historical functions rather than executing old capture/audit top levels.

| Separate command family | Normal leaves | Race leaves |
| --- | ---: | ---: |
| C_se new runtime/orders/controls/decoder/wrapper | 19 | 19 |
| M1 affected C_es, including historical recovery | 39 | 39 |
| C_reason new runtime/modes/controls/decoder/cut | 25 | 25 |
| M2 affected C_es/C_se, excluding recovery | 57 | 57 |
| M2 focused reason model/oracle | 40 | 40 |

Each context has eight automatic A/B answer samples. Reason additionally has six
representative A/B fail/absent/pending executions, five executable controls, five
detached corruption controls and one classification-cut/completion test per profile.
The full four-mode domain belongs to model enumeration, not these runtime samples.

Public B schedules witness both original publication orders. After consuming S at
sequence 3 the parent stays open at PC3 requiring e: e is unproduced in one cut and
retained-unconsumed in the other; original work completes with e consumed at sequence 4.
Helpers are never substituted for original candidate completion. Dual pending
controls preserve the first-await versus second-await distinction. Runtime original
publication/delivery is atomic; an abstract split is not a real intermediate cut.
Reason's genuine absent cut has classification/decision at root sequence 18/order 51,
PC34→35 and open End, no root outcome; End executes at sequence 19/order 52. Completion
preserves the earlier raw prefix. No cut between classification and decision is claimed.

Wrong-id and requested-absent→DeliberateFailure are truthful raw/prefix executions
with correct defensive/observation-based policy but component/profile rejection.
Genuine missing_reply remains distinct. Invalid S waits for both joins. Budget2
retains authentic helper refusal, zero refused debit and spending 2 outside entitlement.
Detached rehashed tag/branch/reason/pending/final-tag corruptions are validation
controls, not executable components or substitution counterexamples.

Milestone 1 is checkpoint `1f71b56148f2fd6ef41261b7caf2996e8eeb2533`; milestone 2 is
`985e08cf54f2dbcb0014269a49f91babe166f8e7`. Earlier captures retain their complete
historical manifests: later reason CLI/shared-sidecar additions mean not every M1
manifest equals the final tree. Reuse rests on unchanged relevant semantic sources
plus affected later regressions, including exact trace/handle/verdict equality over 48
historical inspections. Final reason commands bind 380 sources at aggregate
`bfbba3d721fefdc63da9b76dce92b2af8226a6cbe1d837b1601ed8a903520590`.
The [final code-static record](evidence/core/final-static.json.gz) records seven
passing commands, 378 Go/module hashes and 90 protected inputs unchanged; the runtime
manifest additionally includes capture/audit scripts. Detailed identities are indexed.

## Limits and remaining obligations

No production/compiler/runtime repair or new process/crash campaign was needed.
Historical [Phase 1L recovery](../phase1l/02-qualification.md) keeps its candidate-first
consumed-e/unfinished-S predicate and original source; M1 reran that regression,
M2 did not. Preserved construction/checker/capture/reporting failures and the Task 4
REJECT/rereview are classified in 03-review.md. They do not establish semantic
substitution failures or erase historical I/J/K findings.

[Whole-branch source review](evidence/reviews/final-r1.md) approved the result with
no actionable findings. The [single end-phase Tectonic build and PDF inspection](evidence/manuscript/render.json)
passed with accepted warnings; the new discussion is on pages 14–15 of 19, with
neighboring pages 13–16 inspected. General congruence, arbitrary context
grammars, universal scheduling/compiler correctness, exact-debit equivalence,
effectful/distributed recovery, deeper nesting and invocation-result adapters remain
open. The next research obligation is a separately specified larger context family
with its own observation and resource assumptions, not an extrapolation from these
two finite consuming policies.
