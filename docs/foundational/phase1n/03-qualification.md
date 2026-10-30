# Phase 1N callable join qualification

Status: public construction, reference, finite/static qualification, runtime
and whole-phase source review approved. Final R1's historical-audit reproduction
finding F1 is preserved, corrected and independently approved in R2. The single
final PDF build and changed-page inspection passed; final task records are
archived and their historical restoration check passed.
[Review disposition](06-review.md) records the actual gates; historical
freeze documents retain their original status rather than being rewritten.

## Question and result

The fixed owning join J is an ordinary public callable component. It consumes
original candidate and S outcomes, replies with a bounded truthful diagnostic,
and deliberately fails for diagnosed errors. Fixed root O consumes its original
J invocation outcome and commits a guarded decision using only that public
evidence and original input. This boundary was constructed and executed without
changing the production compiler, executor, schemas, scheduler, accounting or
dependencies. Original K1, A/B/helper/S and historical evidence remain unchanged.

The result is bounded nested preservation plus **ARG-1, a restricted conditional
constructor argument**. It is not arbitrary-context congruence. **ARG-2 remains
open:** unchanged arbitrary weak K1 refinement has no established implication to
the stronger relation premises below. No counterexample disproving that general
implication is asserted. [Contract](01-contract.md), [representation](01-representation.md)
and [lifting argument](02-lifting.md) retain their exact freezes.

| Evidence category | Established scope | Limit |
| --- | --- | --- |
| Source-level mathematical argument | Reviewed ARG-1 transition/progress argument under supplied R and environment E | Conditional, not a machine-checked generic theorem |
| Exhaustive finite exploration | K1-first reference and A/B products, separate local relation/progress, finite lifted relation | Declared finite event semantics/input domain |
| Artifact correspondence | Exact fixed instruction/guard mapping and independent finite policy evaluator | No universal compiler correctness |
| Concrete execution | Automatic answer/mode samples, supported late reply and genuine controls | Single-host samples, not all permitted schedules |
| Independent raw audit | Actual original identities, selected histories, instructions, reports and public decisions | Shared record/schema formats remain assumptions |
| Pure replay | Retained-history consistency without changed inspection | Shared-runtime consistency, not an independent semantic oracle |

## Public contract, environment and argument

O is the sole fixed root at depth 0; J enters at depth 1, original candidate/S at 2,
and B's unchanged helper at 3. J commits e then S and consumes S then e before any
normal classification. It reports `{id,classification,answers}` with at most one
original answer record. Only `answer` has one correlated complement record;
diagnosed errors have none. Both J and O guard presence before payload reads and
classify actual evidence, never requested mode.

Candidate `failed/missing_reply` becomes a present `candidate_missing_reply`
diagnostic on J's deliberately failed `failed/member_failed` outcome. Candidate
`failed/member_failed` yields `candidate_failed` on that same failed public
envelope. Missing J reports are different: successful End converts to
`failed/missing_reply/absent`; failed End remains `failed/member_failed/absent`.
O rejects either as `invalid_join_outcome`, without inventing a candidate diagnosis.

Common entitlement is I=R4/depth 3/E0/fixed 1. A spends (3,3,0), B (4,4,0).
J's prior fixed (1,1,0) admission leaves residual I=R3; only accepted original S
admission maps to the arbitrary-time local q debit. The physical depth/resource
guard correspondence is explicit. There is no outer sibling or second interfering
debit. Different A/B charges do not establish Definition 4.6's symmetric
exact-debit relation.

[Reference review ARG-1](evidence/reviews/reference-r1.md) supplies the precise
rule. X ranges over the declared e/helper event semantics, with no extra
operations/inputs or access to J/O/S private control. A supplied R preserves
original identities under one fixed renaming, public protocol/outcomes/presence,
source terminal/closure state, ledger/depth and future demand. Candidate events
match the same single event or a private stutter preserving those public fields.
A weak-path alternative additionally excludes nonmatching protocol events and
requires stable guards at intermediate stutters. Bilateral environment and
caller-owned commitment/selection readiness, equal authorization sidecars and
initial/prefix entitlement are separate premises.

Separate all-mode local progress/nondivergence and eventual eligible service are
required, with stable bindings/authority, open eligible caller, sufficient finite
limits, no cancellation/retirement and no pending release. Under these premises
J preserves the relation; the independently qualified J[K1] premise yields
K_join, and adding fixed O control gives G_outer. ARG-2 is the missing
normalization/extension lemma for intermediate guards, caller readiness and
authorization sidecars from arbitrary unchanged weak K1 refinement. A/B finite
eventwise witnesses instantiate ARG-1; enumeration does not discharge ARG-2.

## Finite and static results

The [qualification records](evidence/model/qualification/README.md), independently
[approved](evidence/reviews/qualification-r1.md), cover id {1,2}, value {0,1}, mode
{answer,fail,absent,pending}: 16 configurations per product, 48 total. K1 retains
its independent local/helper alternative in every mode.

| Product with O | States | Edges | Terminal / pending maxima |
| --- | ---: | ---: | ---: |
| J[K1] | 6,376 | 13,232 | 24 / 8 |
| J[A] | 2,864 | 5,622 | 12 / 4 |
| J[B] | 6,644 | 13,844 | 12 / 4 |

All recorded invariant/property/rank failures, refusals, stuck states, cutoffs and
cycles are zero. Every edge is checked before deduplication with the full
model/monitor key. K_join and G_outer project the same products; normal/race
repeats are not additional configurations. The rank cap 12,221 is an abstract
finite bound, not a runtime budget or a bound for arbitrary private computation.

A/B aligned local relations contain 884/2,320 pairs; separate open-progress graphs
have 376/824 states and 544/1,244 transitions, each with 12 terminal/four pending
maxima. Lifted relations have 2,864/6,644 pairs and 16 initial pairs each, with zero
failures. Pending maxima have consumed the original request/chosen helper and
original S, e at its designated unmatched receive, J open at original e await,
and O open at original J await. No diagnostic, J publication, O decision or root
publication is invented. Zero successors establish finite maximality; safety of
an earlier prefix alone does not.

Current r2 correspondence maps 283 instructions and six exact A/B binding
differences, checking 73,984 J and 6,784 O policy cases per binding. Schema rejection
is distinct from defensive policy execution; absent fields are not defaulted.
All static instructions remain mapped, including schema-unreachable setters.
J classification expands to classification plus an immutable-tag ghost decision
at one Set commit; the actual later branch/End is checked. Invoke/handle/intent,
Receive/binding/consumption, End/freeze and publication/delivery have explicit
atomic groups. No ghost intermediate state is claimed as a durable checkpoint.

Approved conservative bounds are A 139/B 149 local commits and 9/12 dispatches;
the public consumer uses 512 steps/64 dispatches/capacity 1. Detailed byte bounds
in the current correspondence record fit unchanged 1 MiB limits. They depend on
the fixed finite stable no-retry/no-closure profile and original identity sizes;
they are not arbitrary-value or recovery bounds. The earlier representation
freeze's estimates retain their historical scope.

## Actual execution and adverse evidence

Final [runtime records](evidence/runtime/README.md) contain 19 principal cases per
normal/race profile: eight automatic A/B answers across four id/value inputs,
six representative A/B fail/absent/pending cases, one supported B/absent late
reply, and four genuine controls. Seven detached attacks, two incomplete-prefix
checks and one control-identity construction leaf bring the total to 29 passing
leaves per profile. The unchanged focused M/L selection passes 82 per profile.
These numbers count test leaves, not distinct model configurations or schedules.
The independent Python audit passes both final profiles; [Task 4 review](evidence/reviews/runtime-r1.md)
returns specification and quality APPROVE with no findings.

Automatic answers use 75–78 commits for A and 84–85 for B, with 9/12 dispatches.
Fail/absent use 62/65 for A and 71/74 for B; pending uses 13/22 and retains the actual
unmatched receives, J at e-await PC 5 and O at J-await PC 1. Pending runtime
quiescence is distinct from exhaustive model maximality. All automatic samples
use the stated 512/64/capacity 1 settings and original charges/depths.

The supported B/absent schedule authenticates J's Send at fact 65, Failure
execution/freeze at 67/68, **first** reply authorization 69, admission 70, original
J publication/delivery 71 and O selected consumption 72. At the post-End cut no
authorization/admission exists and preparation waits for `reply_admission`.
The first authorization records the failed source authority and is allowed by
the original entry-member reply binding to the actual caller/generation/work
obligation, with stable revision/generation and matching schema/template. O
consumes a present `candidate_missing_reply` on `failed/member_failed` and commits
the matching failure. Invocation authorization has no such exception. The driver
uses sequential public Store methods, with no hooks, sleeps, fabricated handles
or authoritative-state writes; it has no fabricated automatic RunReport.

| Genuine control | Raw authentication / O policy | Truth and conformance |
| --- | --- | --- |
| Valid children, no report, J Success | Pass; O selects failed/missing_reply/absent and commits invalid_join_outcome | Completed-diagnostic/K_join premise fails |
| Valid children, no report, J Failure | Pass; O selects failed/member_failed/absent and commits invalid_join_outcome | Completed-diagnostic/K_join premise fails |
| Real candidate missing_reply, false candidate_failed report | Pass; O faithfully executes candidate_failed policy | Report truth and K_join fail, although candidate conforms |
| Historical DeliberateFailure under absent request | Pass; truthful candidate_failed and matching O policy | K1 mode conformance fails; mode-specific K_join premise unavailable |

The controls are genuine separately identified executions; none is a substitution
counterexample under conforming premises. Authenticity alone cannot prove report
truth. Seven rehashed detached corruptions of handle, selected observation,
classification, report successor, J/O branch or cursor are rejected by both
decoder and independent audit. Authentic classification-only and Send-only
prefixes pass prefix checks but reject claims of End/publication/completion.

The new mapping follows actual emitted handles, bindings, selected observations,
committed instructions/cursors, report transport and terminal/publication links.
The Python audit does not call the Go decoder, oracle or reducer. Shared original
artifacts, schema/protocol formats and historical structural helpers limit
independence. Exact evidence/source identities and executed argv are linked in
[the runbook](04-runbook.md); source identity alone is not semantic truth.

## Remaining obligations

The [final evidence index](evidence/index.json) binds the delivered sources,
reviews, archive and render records. The single final build produced a 20-page
PDF; changed pages 15–16 and neighboring pages 14–18 passed visual inspection,
with the established box warnings accepted. No production or public-interface
obstruction was found in the recorded scope. The broad
congruence theorem, arbitrary candidates without supplied R, recursive tree
closure, universal compiler/scheduler correctness, exact-debit A/B equivalence
and effectful/distributed recovery remain open. No new recovery campaign ran;
historical recovery keeps its original predicates and source identities.
