# Phase 1N Task 3a independent reference review r1

Reviewer: `/root/n_reference_review`, independent of the reference and public
construction authors. Review date: 2026-10-30. This is the reference-machinery and
argument-readiness gate before exhaustive reference/replacement qualification.

**Specification: APPROVE. Quality: APPROVE.** No blocking defect was found in the
reviewed freeze. Approval permits K1-first qualification of this machinery; it is
not approval of an unrun census, A/B acceptance, artifact correspondence, runtime
behavior, or the unrestricted weak-refinement implication. The restricted
conditional constructor argument closes under the explicit premises below. The
stronger claim that every weak K1 refinement supplies those premises remains open.
There are no substantive rejection findings to waive or discard.

## Reviewed identity and evidence

The exact approved input is N-contract-1, SHA-256
`001c1fe5cd7be722aa75ae00a2be84ec702b51783a5595754d38ec8e6b2d8fd9`.

| Reviewed item | SHA-256 |
| --- | --- |
| `research/substitution/callable_join.go` | `98712edefc1e9dd44f146089c4e3e8d1da2fcac1df5a57421d73ce31c6a07a08` |
| `research/substitution/callable_join_explorer.go` | `34377a3e528c12e6ff5bb803a16337eb434fa5d82120b4bf0888a0d47cdd58cd` |
| `research/substitution/callable_join_local.go` | `d918c235822262b0c630d6ce69721036d2b614a5e4020585c3e9b86ed68275fb` |
| `research/substitution/callable_join_oracle.go` | `765fafb73477584e65f94f89ef270ffe1c3f4019d9f5ffa6420dc816d406c6b1` |
| `research/substitution/callable_join_relation.go` | `61c61652dd9279147fd524a77e03476a0006d17e2c8e022f0582a29937d6997a` |
| `research/substitution/callable_join_test.go` | `51179b5747f84f7faeb97d028d78d61cec794a21f7793a2cf3eca4185600e5b8` |
| `docs/foundational/phase1n/02-lifting.md` | `53bf70cb2aad625b4d5a3a753610ba271c84e738f557904860988350cbe40e54` |
| `evidence/model/reference-r1.json` | `4603206f1eccf3bb471ca1df821aa56c613633d98dfdd9d84016786db137023e` |
| `evidence/model/reference-r1.tar.gz` | `8d2a2cfb47c37f8f4a666e95c5dfa8f386fffef5af46e5ff12a32de22fb1b0b3` |
| Sorted path/NUL/source-SHA/newline aggregate | `36583a769994342aca07144e6ae0dcd4f4a62a30274a0863c75ee9b28a36d8c2` |
| `.superpowers/sdd/00-plan-phase1n/task-3a-diff.txt` | `6de198ea0b8e78ca17c820d666002de91aa4e9fa4b7b26a3e30be2ac87778586` |
| `.superpowers/sdd/00-plan-phase1n/task-3a-report.md` | `bc0fa34d07f8de9b87ada96ef283d03d46572a6d6bb18bbf3459ee995215479a` |
| `.superpowers/sdd/00-plan-phase1n/task-3-brief.md` | `31ec5d181e2aad340581c98a100dc0b051a492b717afc2737fc38bf2446d139c` |
| `.superpowers/sdd/00-plan-phase1n/constraints.md` | `12d3c6383fd26aa3a92a8c8ae7c408e2c4e7f38f128abd124dee6918ba1bb6dd` |

All 66 manifest source/input entries match both the working files and archived
bytes. All 23 `NEW` entries in the supplied diff match their declared hashes.
All 53 manifest historical inputs match baseline
`3c5118ea9763a27cca432ca91aac667bd59e21e7` using independent `git show` byte hashes.
Ambient HEAD `907e22694149cf2ba52069c7c2e7038fdacc63fa` is identity metadata only.
These checks do not constitute behavioral review of every historical file.

The six new Go files and lifting document were inspected in full. Targeted shared
semantics inspected were K1's contract; `model.go`, `candidates.go`, `relation.go`,
`oracle.go`, `explorer.go`, `invocation_join.go`, `invocation_join_oracle.go`,
`consumer_reason.go`, `consumer_reason_oracle.go`, `consumer_reason_local.go`;
and the exact binding/authorization/reply-capture/publication sections of
`internal/executor/invocation.go`, `messages.go`, and `invocation_outcome.go`.
Preparation, capture/freeze scripts, manifest, evidence README and preserved
capture-failure record were also inspected. Shared historical implementations
remain an independence limitation.

Read-only Python/hash/archive checks recomputed the final command-record hashes,
source bindings, exit fields and leaf events. Normal and race records each contain
66 passing leaves, zero failing/skipped leaves, exit 0 and empty stderr. Vet has
exit 0 and empty stderr. Their source hashes match this freeze. The informational
version record predates formatting, as disclosed. These are inspected author-run
records, not new reviewer-run tests. No captured test was rerun; no full product,
local A/B acceptance or finite replacement lifting was executed. No new
`examples/callablejoin` source, representation/preflight/runtime outputs or public
construction report was opened. Only this review file was written.

## Checked boundaries

**REF-1 — Separate obligations and monitors: satisfactory.**
`callable_join_oracle.go:206-277,279-431` separates truthful inner-prefix policy,
K1 component checks, K_join completion, public-only policy and G_outer. K_join
projects away O's private decision/End/root publication. The public checker uses
only original input and J's public events. The false-report fixture at
`callable_join_test.go:256-284` correctly distinguishes public authenticity/policy
from report truth and from mode-conforming G_outer. The no-report fixtures retain
successful-End/missing_reply and failed-End/member_failed as distinct absent
public outcomes. No report is inferred from classification, Send, or delivery.
Required correction: none.

**REF-2 — Search identity, every-edge checks and maxima: satisfactory machinery.**
`callable_join_explorer.go:29-37,99-174` keys the full model and prefix monitor,
checks rank/state/projection/prefix/component events before deduplication, retains
actual zero-successor maxima, does not prune failures, and marks cycle status
unresolved after a rank failure. There is no configured cutoff. Component-monitor
memory is represented by the keyed operation commitments, admissions, reply
values/dispositions, publications and selected consumptions; pending evidence is
separately keyed. The relation checker is explicitly a finite single-event
checker, not a weak-path or progress proof (`callable_join_relation.go:28-84`).
Required correction: none. Census success and rank extrema remain unmeasured.

**REF-3 — Physical projection and refusal/accounting: satisfactory.**
`callable_join_local.go:5-101` uses EntryDepth2/Depth3, residual I/R3, and maps only
accepted original S admission to the sole q debit. It preserves e/helper state,
checks projected event equality against the actual physical local graph, and
rejects a refused S instead of inventing a debit. The depth/residual adapter in
`callable_join.go:73-85,125-143` has the stated guard bijection. J's prior debit is
separate from q. The zero-valued `possible` ticket and `Initial` at
`model.go:27-31,106-135` show that e is uncommitted at the post-J-admission boundary;
there is no hidden precommitted-entry mismatch. The J request is a local stutter,
then its e Invoke establishes the local commit. S admission can occur before or
after e admission once J has committed e then S. All-mode local maxima are checked
by the new depth2/3 predicate, not the old depth1/2 predicate.
Required correction: none. Positive qualification must still establish the full
prefix demand invariant and absence of refusals; the smaller representable limits
and synthetic refusal-projection fixture are not a scarce-resource census.

**REF-4 — Late first authorization and failed present publication: satisfactory.**
`callable_join.go:178-209` does not require reply authorization before J End.
The directed paths deliberately take Send, End, first authorization, admission,
publication and selection for successful and failed J. The monitor checks the
source-open snapshot against End while requiring original bound reply identity
and stable revision/generation (`callable_join_oracle.go:131-170`). The production
source supports exactly this exception: `invocation.go:123-140` binds the source
run/entry member/caller generation/obligation; `messages.go:154-165` checks source
revision and generation before the reply exemption; `invocation_outcome.go:164-184`
checks original Send schema; `:354-373,426-440,571-621` waits for captured reply
admission, retains accepted responses on Failure, and converts missing reply only
on successful End. Ordinary invocation authorization receives no such exception.
Required correction: none. `ReplyBound` is a normalized premise, not raw identity,
route, schema or retained-binding authentication; that correspondence remains open.

**REF-5 — Exact pending boundary and finite bounds: satisfactory machinery.**
`callable_join_oracle.go:345-376,414-418`, `callable_join_explorer.go:55-66,103-127`,
and `callable_join_local.go:115-141` jointly require original request consumption,
correct completed/consumed chosen helper, actual unmatched candidate receive,
actual original S selection, J at its original e await without report/End/publication,
and O at its original J await without decision/End/root publication. A prefix
verdict alone does not claim maximality. The explorer separately requires zero
successors. No release or Pending diagnostic exists.
The factor counts in `02-lifting.md:245-271` cover the new finite fields, and the
new rank additions are monotone once-only bits/stages. The cap 12221 is a bound
for this finite representation, not a runtime instruction budget or a uniform
bound on arbitrary candidates' private computations. Required correction: none.

## Argument status and the exact restricted rule

**ARG-1 — Restricted conditional constructor rule: justified source-level argument.**
The transition induction closes with no remaining logical bridge inside the
following explicit scope. It is a conditional mathematical argument checked by
source reasoning, not a machine-checked generic theorem or an empirical acceptance
result.

For the fixed J and public environment E in N-contract-1, let X range over
candidates in the declared e/helper event semantics. Candidate variation concerns
their local/private instructions; original caller-owned commitment/selection and
shared protocol/environment operations retain their declared rules. X has no
access to J/O/S private control, no extra operations or environmental inputs, and
no post-freeze component work requiring an early caller selection. Supply an R
on the physical EntryDepth2/Depth3, residual I/R3 local environment satisfying:

1. Initial pairing and equality, under one immutable identity renaming, of request,
   original links/handles, protocol stages and outcomes/presence, source terminal
   state, closure, ledger, depth and future demand. Private PCs/registers may differ.
2. Each candidate event matches the same single reference event with a related
   successor, or is a pure private stutter preserving those public fields. A weak
   alternative is sufficient only when its finite matching path has no additional
   nonmatching protocol event and its intermediate stutters preserve every guard
   listed here. This is an explicit extra condition, not a redefinition of K1.
3. Bilateral readiness and related successors for the declared environment events,
   including the sole q debit. Caller-owned e commitment and original selection
   use the same public readiness rules and cannot acquire candidate-private guards.
   Split authorization uses equal original commitment/source-open/generation/revision/
   binding predicates and equal retained first dispositions; neither candidate can
   observe or mutate an extra authorization variable to change its local behavior.
4. Initial and prefix entitlement with fixed prior J debit (1,1,0), residual
   demand at most (3,3,0), full demand at most (4,4,0), E0, and no refund. K1 keeps
   its independent local/helper alternatives in all four modes. A matched X/K1
   path has equal charges; separate A and B paths need not have equal charges.
5. Separate all-mode local progress and nondivergence under eventual eligible
   service: exact answer/fail/absent envelope or designated unmatched pending after
   original request and chosen helper consumption. A well-founded private-stutter
   argument or equivalent nondivergence obligation is required independently of R.
   E supplies stable original bindings/authority, open eligible caller, sufficient
   finite operational limits, no closure/retirement and no pending release.

Then the relation in `02-lifting.md:110-133`, extended by equal authorization
sidecars and the same S/J state, is preserved by the fixed constructor. The fixed
entry prefix initializes it at the uncommitted local initial pair. Candidate
steps transfer by R; S admission transfers by bilateral q readiness; other S
steps stutter in the projection. Matching J control enforces the same e-then-S
commitment and S-then-e original selections. A matching eventwise step cannot
need an early e selection. For the guard-stable weak alternative, monotone selected
flags rule out an intermediate selection on a path ending unselected; a path
matching selection starts with the common S gate already satisfied. Source/binding
agreement permits the same first authorization, even after End, and current
admission guards preserve the retained disposition and exact debit/refusal fact.
Equality of selected evidence gives the same diagnostic, then the same distinct
Send/End/admission/publication/delivery/selection stages. Strengthening the relation
with O control gives the fixed public-policy corollary.

Progress also transfers in this restricted scope. J's finite entry prefix and
S's independent finite answer path eventually make q and the S-selection gate
available; entitlement prevents resource starvation. Delaying e selection cannot
block prepublication component work because e is frozen first. Each inserted
authorization stage is once-only and eventually serviced under stable binding,
so it adds finite delay. These delays preserve local service assumptions rather
than introducing a fair infinite starvation suffix. The separate candidate
progress/nondivergence premise then yields its mode boundary; J and O have finite
post-join suffixes. Pending retains the stated unmatched wait rather than being
forced to terminate. This reasoning does not derive liveness from safety simulation.

Consequently the restricted rule establishes conditional lifting
`J[X] <= J[K1]`. **If** the independent reference premise `J[K1] |= K_join` is then
established under E, it yields K_join preservation; fixed O's policy gives the
corresponding G_outer result. The reference premise and finite replacement
instances have not yet been established by this task.

**ARG-2 — Arbitrary weak K1 refinement implies the stronger supplied-R premise:
unresolved, not disproved.**
The exact remaining step is a normalization/extension lemma from unchanged K1
`phase1i/01-substitution-contract.md:218-242` to the intermediate guard stability,
caller-action readiness and authorization-sidecar extension used at
`02-lifting.md:155-190`. K1 supplies endpoint agreement, event-fact matching,
bilateral declared environment readiness and separate progress. The new argument
must not simply equate those prose clauses with preservation of every constructor
guard at all intermediate states of an arbitrary weak matching path. No universal
counterexample is asserted here; nor was a proof of that implication supplied.
The finite checker at `callable_join_relation.go:30-79` checks the stronger eventwise
instance and cannot discharge this normalization lemma by successful enumeration.

Required disposition before any broader theorem claim: either prove that exact
lemma against a precise K1 weak-path definition, or retain ARG-2 and state the
restricted rule with ARG-1's additional supplied-R premise. Do not silently amend
K1, treat schema compatibility as behavioral refinement, or call a prospective
finite A/B result a proof of the broad implication. The existing freeze explicitly
leaves this question to review (`02-lifting.md:182-190,290-298`), so retaining this
open obligation does not block K1-first qualification of the approved machinery.

## Remaining gates

Run the 16 independent K1 products first; then the actual shifted A/B local
relation and separate all-mode open-progress checks; then the 16 products and
finite lifted relations per replacement. Preserve every maximum and any failed
edge/rank/refusal with its exact source identity. K_join and G_outer predicates
on one product are not two independent censuses. No state/edge/maximum count is
predetermined by this review.

Compiled representation/instruction correspondence, raw selected-evidence and
reply-binding authentication, concrete operational bounds, genuine runtime
samples, independent raw audit and manuscript reconciliation remain open.
Neither this approval nor ARG-1 establishes those obligations, arbitrary-context
congruence, symmetric exact-debit equivalence, scheduler confluence or recovery.
