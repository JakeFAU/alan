# Phase 1N callable join behavioral contract

Status: behavioral review freeze, version N-contract-1. This specifies the new
callable boundary and its environment before qualification. It does not establish
a callable construction, a new test result, or a lifting theorem. Concrete report
bounds and representation remain to be frozen after the public construction
preflight and before model/artifact correspondence. Independent contract review
and its exact reviewed identity are separate controller gates.

The research question is whether the qualified owning join can itself satisfy a
public component contract, and what conditional rule lifts K1 substitution through
this fixed constructor. The smallest new boundary is one invocation-bound reply
from J to O, including J's own publication and O's original selected consumption.

## Baseline and protected history

The checked-out baseline and locally fetched `origin/main` are
`3c5118ea9763a27cca432ca91aac667bd59e21e7`. The reviewed Phase 1M head
`d59595d7081139a149d404d51ba97b46925f6d15` is an ancestor; both trees are
`726502d2da6d1d800c474d673581940c6ce827dd`. The controller performed the fetch;
this task checked local refs, ancestry and tree equality. These are observations
at freeze time, not a claim that main cannot move.

[Protected baseline identities](evidence/core/protected-baseline.json) pin the
relevant source blobs and historical trees. Original K1, A/B/helper/S artifacts,
prior contexts and all historical evidence remain identifiable and unchanged.
Production compiler, executor, schema, scheduler, accounting and dependencies are
preserved. The manuscript retains PR19 formatting, PR20 mathematics and existing
theorem qualifications; scoped reconciliation follows the new results.

Phase 1M's [qualification](../phase1m/02-qualification.md) records bounded
C_reason preservation over 16 configurations per K1/A/B product in its own
I=R3/depth2 environment. Its [reason freeze](../phase1m/01-reason-contract.md),
[reviews](../phase1m/03-review.md), [runbook](../phase1m/04-runbook.md) and
[decisions](../phase1m/05-decisions.md) retain their original scope. They do not
establish a public callable join or the depth-shifted constructor rule here.

## Fixed topology and ownership

O is the sole fixed root at depth 0. It invokes one J[X] at depth 1 and retains
that original invocation handle. J receives the original request through a public
port. It commits candidate e and original S in that order before its first await.
Both receive the same original inputs used in Phase 1M: e receives `{id,value,mode}`
and S receives `{id,value}`, never the candidate's complement. J consumes S then
e through their original handles. Every normal classification, report and terminal
decision follows both consumptions, including error paths. B's unchanged helper
is at depth 3; e and S are at depth 2.

J is a new callable artifact. The old C_reason root has inputs but no public
entry/reply interface; it must not be relabeled invocable or placed behind a
separate forwarding wrapper. One J builder is parameterized by candidate binding.
One O builder is parameterized by J's public template. A/B receive identical
policy and allowances; intended binding, catalogue and compiled-identity
differences must be accounted for. O must not special-case A or B.

## Public request and diagnostic

The request remains `{id, value, mode}` with the original schema. The finite
profile is id in `{1,2}`, value in `{0,1}`, and mode in
`{answer,fail,absent,pending}`. Original scalar integer schemas enforce kinds,
not these finite integer sets; the original mode schema has MaxBytes 8 and the
four-value Enum. The original answer record is `{id,value}`.

The bounded diagnostic has exactly these semantic fields:

| Field | Meaning |
| --- | --- |
| `id` | Original application request identity |
| `classification` | `answer`, `candidate_failed`, `candidate_missing_reply`, or `invalid_outcome` |
| `answers` | Bounded zero-or-one list of original answer records `{id,value}` |

An `answer` requires exactly one correctly correlated complement record. All
other tags require an empty list. A maximum list length of one does not establish
that an answer exists. Use ordinary public schema/authoring primitives and guard
all potentially absent reads. Concrete byte bounds, retained collection layout
and compiled representation are deliberately not selected by this behavioral
freeze. Freeze them after the public construction preflight, before model/artifact
correspondence; any representation adjustment must preserve these semantics.

No inner operation handle is needed in this report. O correlates the request
through the application id and consumes its own original typed J handle. Do not
reintroduce handle/string comparison or serialize inner handles.

## K_join classification and publication

K_join is specified independently of A/B execution outputs. It requires truthful
reporting of the actual original inner consumptions, both required joins,
identity/payload/resource safety, the table below, and mode-specific progress
under the declared environment. Original requests, joins and the diagnostic are
once-only: no duplicate report, replacement child or substituted outcome is allowed.
Requested mode predicts conforming behavior in
the independent specification; it must never replace actual observation-based
classification inside J or O. J may forward request.mode to e, but neither J's
classifier nor O's decision branches on mode. Original id/value remain available
for correlation and complement checks.

After both consumptions, first validate S as `succeeded/member_succeeded`, response
present, original id/zero. With valid S, classify actual selected candidate evidence:

| Actual consumed candidate evidence | J diagnostic | J local End and published envelope |
| --- | --- | --- |
| `succeeded/member_succeeded`, present, correct id/complement | `answer`, one answer record | Success; `succeeded/member_succeeded`, diagnostic present |
| `failed/member_failed`, response absent | `candidate_failed`, empty answers | Failure; `failed/member_failed`, diagnostic present |
| `failed/missing_reply`, response absent | `candidate_missing_reply`, empty answers | Failure; `failed/member_failed`, diagnostic present |
| Other completed candidate evidence, or invalid S | `invalid_outcome`, empty answers | Failure; `failed/member_failed`, diagnostic present |
| Designated pending candidate | No diagnostic | No J End or invocation publication |

J sends the diagnostic on its invocation-bound public reply before executing its
terminal instruction. Error reporting does not turn failure into success.
`invalid_outcome` defines defensive behavior and is excluded from conforming
K_join maxima under the declared K1/S profile. It cannot weaken the positive
contract or excuse a malformed conforming outcome.

Three levels of absence remain distinct:

1. Candidate absence: J consumes candidate `failed/missing_reply` and reports
   `candidate_missing_reply`; J publishes `failed/member_failed` with a present
   diagnostic.
2. J reaches Success without its required report: its invocation publishes
   `failed/missing_reply` with an absent response.
3. J reaches Failure without a report: its invocation remains
   `failed/member_failed` with an absent response.

The last two violate the completed-diagnostic requirement. Neither is a valid
`candidate_failed` report. A classification write, diagnostic Send commitment,
reply admission, local End, invocation publication, delivery and selected
consumption are distinct semantic facts. A prefix ending at any earlier fact
cannot be reported as completed publication or consumption.

Existing `prepareCompositionScope` collects admitted replies for failed members
as well as successful members; missing-reply conversion applies to successful
members requiring a reply. Existing binding-qualified reply authorization permits
service of a committed original reply after its member End, subject to unchanged
identity/revision/binding checks. Invocation authorization receives no such
exception. These source observations support a feasibility premise; public
construction and concrete correspondence still need their own evidence.

## O policy and G_outer

O awaits its original J outcome once. It guards response presence before reading
diagnostic data and checks original application-id correlation, status/reason/tag
consistency, and exact answer cardinality/content. It records its actual decision
in program state by committed instructions:

| Actual public J evidence | O decision |
| --- | --- |
| Valid `answer`, exactly one original id/complement record, `succeeded/member_succeeded`, report present | Success |
| Valid `candidate_failed`, empty answers, `failed/member_failed`, report present | Failure retaining `candidate_failed` |
| Valid `candidate_missing_reply`, empty answers, `failed/member_failed`, report present | Failure retaining `candidate_missing_reply` |
| Well-formed `invalid_outcome`, empty answers, `failed/member_failed`, report present | Corresponding defensive Failure |
| Missing, inconsistent or invalid J evidence | Distinct `invalid_join_outcome` Failure |

Every valid report also carries the original application id. The last row must
not synthesize a candidate diagnosis. No J outcome means O remains at its original
await; an absent response on a published outcome is a different observation.
O has no access to J's private child rows, inner registers or inner handles. Its
policy uses only the public outcome and original input. The harness may inspect
private evidence to qualify report truth; a host reconstruction cannot stand in
for O's committed observed decision.

G_outer is separate from K_join: O consumes the original published J outcome
once and takes the matching guarded policy. Under conforming K_join it preserves
answer, the two distinct diagnosed failures, or the specified pending state,
with correct root/aggregate publication and safety. Answer gives O Success and
root/aggregate success; the two error modes give O Failure and root/aggregate
`failed/member_failed`, with their distinct retained decision tags. These are
correct classified failures, not successful completion. O is a fixed root and
has no further ordinary reply obligation.

Schema-valid authenticated lies can make O execute its public policy faithfully.
Detecting such a lie belongs to J's semantic conformance obligation. Authenticity
proves where a report came from; it alone cannot prove that the report describes
the actual selected child outcomes truthfully. Raw validity, truthful prefix
policy, K1 conformance, K_join conformance and G_outer therefore remain separate
verdicts.

## Environment, resources and progress

The common entitlement is I=R4, depth cap 3, E0, fixed1. Concrete samples use
physical capacity1. The positive finite request profile has 16 configurations per
K1/A/B product, with original S fixed to its answer behavior.

| Construction | Total dynamic invocation/run charges | Greatest depth |
| --- | ---: | ---: |
| O → J[A] with original S | 3 | 2 |
| O → J[B] with B's original helper and original S | 4 | 3 |

These are derived expectations to verify, not new measurements. One J admission
leaves residual allowance3 for e/helper/S. The candidate's entry depth shifts
from 1 to 2; its helper shifts from 2 to 3. Commitments have no debit; accepted
singleton admissions charge once, refusals charge zero, and no refund is assumed.
Permit K1's local/helper alternatives in every mode and unequal actual A/B costs
under the common allowance. Exact-debit equivalence is not claimed.

Assume stable generations/revisions/grants, valid template/request/reply/schema
bindings, no cancellation or retirement, sufficient finite operational limits,
and eventual eligible local/protocol service. O remains open through J consumption;
J remains open through both child consumptions. Concrete step, dispatch and byte
limits need artifact-derived justification; abstract rank is not a runtime budget.

Only original S supplies the existing arbitrary-timing local-environment debit.
J's admission is a prior fixed offset, not another concurrent debit. The required
projection starts after J admission, subtracts its `(1,1,0)` charge from the
common allowance, places candidate entry/helper at depths 2/3, maps S's sole
accepted admission to the existing q debit, and maps other S actions to stutters.
It must preserve the local component's allowed environment/readiness, including
S admission both before and after e admission. Do not merely substitute old
depth1/depth2 counts or rerun an unchanged hardcoded qualifier and call this
established. Debit projection alone proves neither S payload/lifetime safety nor
J/O joins or public outcome behavior.

For answer/fail/absent, eventual service must lead through both J consumptions,
the corresponding diagnostic, J End/publication/delivery, O consumption and
matching O End/root publication. Pending instead requires the candidate's actual
designated unmatched receive after its initial request and any chosen successful
original helper consumption; S has published, delivered and been consumed by J.
J remains open awaiting original e, with no classification, diagnostic, End or
publication; O remains open awaiting original J, with no decision, End or root
publication. No second request or pending-release input is supplied. A safe finite
prefix alone cannot establish this maximal pending state: exploration must check
that no progressing transition remains enabled. Eventual service excludes infinite
idle/retry/restart-only suffixes, and is not a claim that pending work terminates.

## Conditional lifting obligations and evidence separation

Keep four obligations separate: X refines K1 under the aligned residual local
environment; J[K1] satisfies K_join; J[X] refines J[K1] under a stated
constructor-specific relation; and O preserves G_outer for a conforming K_join
component. The first requires mode-specific progress as well as local safety.
The second is an independent reference-premise obligation. The third must lift
the first through the fixed constructor's protocol, joins, reporting, resource
and progress boundaries, so that J[X] also satisfies K_join. The fourth is the
outer-caller corollary, using only J's public outcome. These obligations are not
established by this freeze or defined as “passes the A/B products.”

The generic argument for this fixed constructor must justify the environment and
depth projection, original-handle ownership/once-only joins, report truth,
publication boundaries, safety at every prefix, allowed refusals and the absence
of new internal divergence. It must preserve the designated pending maximum and
the service assumptions through both new boundaries. K1 retains its independent
local/helper choice; replacement agreement must not define the reference.

Bounded K1-first products, separate A/B local relation and open-progress checks,
replacement products, finite artifact correspondence, runtime samples and raw
audits can discharge named finite obligations with source identities. They cannot
alone prove the generic lifting implication. A failure of the reference premise
is a constructor-premise failure, not a conforming-replacement counterexample.
Retain the smallest failed premise/transition if the argument cannot be completed.

Definition 4.6's symmetric exact-debit relation and Theorem 4.8's open general
congruence claim are unchanged. This work does not claim arbitrary contexts,
universal compiler/scheduler correctness, deeper recursive nesting, new process
recovery, or effectful/distributed recovery.

## File ownership and independent authorship

| Additive location | Responsibility |
| --- | --- |
| `examples/callablejoin/` | Public report schema, callable J constructor, fixed O/deployment, construction checks, separately named controls |
| `examples/callablejoin/consumer/` | Separate public-import module building and executing the complete deployment |
| `research/substitution/callable_join*.go` | Independently authored reference/monitor, environment alignment, lifting relation, finite exploration/checks |
| `internal/substitutiontest/callable_join*.go` | Genuine runtime schedules, explicit new raw mapping, decoder/control qualification |
| `docs/foundational/phase1n/` | Contract, argument, qualification/limits, runbook, decisions, reviews, compact evidence indexes |
| Existing manuscript Markdown and TeX | Scoped reconciliation after results settle; BibTeX only if references change |

At this freeze, this contract author has inspected original construction,
reference, runtime and manuscript sources listed in the task report. No new J/O
artifact, reference or execution output was used to define the contract. This is
a separate contract author, not a claim that all future layers are already
independently authored or reviewed. The controller must record distinct actual
construction and reference authors, their input scope and freeze identities.
The reference author must derive from this contract and permitted historical
machinery before inspecting new construction artifacts/output; correspondence
opens those artifacts only after reference review. Shared historical protocol
machinery and compiler/runtime dependencies limit independence and remain named
assumptions. A raw audit should independently compare original selected evidence,
report Send and O decision instructions, not call the same classifier as its oracle.

The narrow handoff source map is:

| Boundary | Existing source entry points |
| --- | --- |
| K1 and unchanged components | `docs/foundational/phase1i/01-substitution-contract.md`; `examples/substitution/protocol/protocol.go`, `a/candidate.go`, `b/b.go`, `deployment.go` (original S) |
| Original consuming policy and public consumer | `examples/consumingpolicy/reason.go`, `deployment.go`, `consumer/main.go` |
| Public callable/report authoring precedents | `author/invocations.go`; `compose/collection.go` (bounded response list and Send before Failure) |
| Reply binding and publication | `internal/executor/invocation.go`, `messages.go`, `invocation_outcome.go` |
| Historical reason reference and independent monitor | `research/substitution/consumer_reason.go`, `consumer_reason_oracle.go`, `consumer_reason_explorer.go` |
| Local relation and all-mode progress | `research/substitution/relation.go`, `candidates.go`, `explorer.go`, `consumer_reason_local.go` |
| Raw selected evidence and independent audit | `internal/substitutiontest/consumer_reason_evidence_test.go`, `consumer_reason_decoder_test.go`; `docs/foundational/phase1m/evidence/runtime/reason/audit.py` |

This map locates reuse boundaries; it does not claim every mapped file was fully
reviewed here. The report records actual inspection scope and exact identities.

Reuse reviewed infrastructure narrowly without exporting or restructuring old
model internals solely for this task. Any necessary shared compatibility change
requires focused checks and explicit source scope; historical evidence is never
rewritten. No outer sibling, extra forwarding wrapper, static peer, ordinary
report/ACK protocol, inner-handle serialization, pending-release request,
retry/fallback, new effect or generic context generator is included. A genuine
production/interface obstruction requires a preserved witness and blocked
obligation, not a changed production semantics or weakened positive contract.
