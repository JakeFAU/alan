# Phase 1O Task 2 independent argument review — R1

Spec compliance: **REJECT**.
Semantic/task quality: **REJECT**.
Highest severity: **P1 — unclosed representation premise in the claimed universal implication**.

This is the Task 2 argument gate, not a whole-phase verdict. Rejection means that
the asserted closure for every frozen reading is not established by this artifact;
it does not mean ARG-2 is false. The review found no executed candidate,
contextual counterexample, solver result or machine proof, and treats none as
implied by the source argument.

## Reviewed identities and method

- Base and current HEAD: `bec75fa3db8cdafb6295f3d0e8ebb619400c1f36`.
- Reviewed patch: `.superpowers/sdd/00-plan-phase1o/task2-r1-diff.txt`, SHA-256
  `29e1e92b16af333c54a910ddaa10319bb9778ea55d0f370c9c0ea3560c2afab9`.
- Frozen definition: `docs/foundational/phase1o/01-weak-contract.md`, SHA-256
  `ac1c0ecbf62bd82c718a7a3ad3dcf000552d14e093868321b0f7dcc4b370b000`.
- Reviewed argument: `docs/foundational/phase1o/02-arg2.md`, SHA-256
  `7d37091194cc5e283ae9007959fc56b5ad64fa1f78d04f9752b2a04554b13d21`.

Read the dispatched brief, constraints and author report first; read the supplied
patch once. The initial combined output truncated, so numbered source excerpts
were used to recover missing argument text and precise locations. Checked the
unchanged K1 contract and final N R1 formulation directly, then inspected the
reference choice/guard rules, the historical candidate relation, the local J
projection, the constructor authorization/selection rules, and production
first-authorization/binding/admission/replay code. Agreement between the authors
was not used as evidence.

## Blocking finding

### R1-1 — P1: establish the accounting representation on the original X before claiming all-reading closure

Primary locations: `docs/foundational/phase1o/02-arg2.md:140-145`,
`:149-166`, `:195-205`, and the conclusion at `:399-404`.

The ticket **laws themselves** have direct original support. K1
`docs/foundational/phase1i/01-substitution-contract.md:97-98` gives the choice
updates; `:153-163` gives original ticket states, initial demand, commit,
admission, refusal and exclusion; `:171-176` prohibits retrospective demand
repair and updates demand only through K1 transitions. This review does not
replace those laws with a scalar budget inequality or demand defined by the
language of future completed executions.

The missing implication concerns the representation to which those laws apply.
The construction begins with X's already fixed actual accounted history, allows
that history to supply an accounting projection that is not determined by X's
state, and then says that Rg is a relation on existing history-distinguished
semantic states. The original quantified object has not been shown to contain
those states. K1 `:218-225` specifies a relation between M and K1 states; the
frozen O definition `01-weak-contract.md:244-247` likewise specifies reachable X
and K1 local states. Its A_op definition at `:104-122` permits arbitrary finite
private instruction representations, and its demand discussion at `:172-175`
requires recording the accounting facts without supplying a general
history-to-X-state representation theorem. The historical concrete graphs do
carry ticket/choice fields (`research/substitution/model.go:49-81`), but their
state representation is not established for every X by the arbitrary-graph
premise.

The two relevant scopes must therefore be separated:

1. If X already includes fixed semantic ticket/choice state, the laws constrain
   its original edges. The normalization argument has an identified accounting
   boundary and can compare its demand to g without changing X. The argument's
   all-successor check then addresses the proposed exclusion/commit conflict.
2. If X is only its raw finite transition system and the account is supplied by
   a history annotation, the artifact must define that annotation conservatively
   and show that the resulting witness is an ARG1Compatible relation in the
   original target's domain. Asserting that these are already existing semantic
   states is the unsupported step.

A small separating schema explains why the merge paragraph does not discharge
this by itself. One prefix can perform lawful local exclusion; another can
retain possible h. Both can reach the same raw instruction state whose only
future is a no-helper mode action, with exclusion included in that action on
the second history. There is no helper-commit edge to prune and no resurrection
on either history. An account-indexed proof nevertheless compares that raw
state to two reference boundaries with different U. If ticket state is part of
X, these are two different X semantic states; if it is not, equality of X's
original future demand at a state pair has not been defined. This is a
representation/quantifier diagnostic, **not** a qualified candidate or a
counterexample to existence of a suitable relation.

Requiring every original outgoing edge on both histories is necessary and the
text correctly requires it. It prevents favorable-continuation filtering. It
does not, alone, prove preservation of the target relation domain and its demand
projection. Nor does the explicit authorization-history extension in Bridge C
supply this missing accounting result: Task 2 expressly permits retained first-
decision augmentation, and C starts from a supplied coarse relation.

Required correction: either derive a conservative accounting abstraction from
the unchanged premise, including fixed atomic grouping, initial coverage,
all-history/all-successor behavior and a demonstration that the resulting Rg is
an admissible witness for the original X; or state the existing-accounted-state
condition as an explicit additional interpretation/domain hypothesis and retain
the unrestricted representation case as an open specification/normalization
obligation. A unique canonical private-PC map or a unique authorization sidecar
is not required. A history proof is permissible, but its connection to the
quantified candidate must be proved rather than asserted. Until then, remove or
qualify the claim that no additional locally checked condition is needed for all
frozen readings. This finding is an unclosed proof obligation, not a demonstrated
unsoundness of the conditional accounted-state construction.

## Checked arguments that do not independently block this gate

- **Current guards and ordered weak paths.** The only additional reference
  choices are at `model.go:274-287`; they do not change source terminal/closure,
  actual ledger/depth, committed intents, reply dispositions or selected
  envelopes. In that region the environment guards at `:211-227` depend on
  closure/ledger, not optional uncommitted demand. J authorization/selection
  guards at `invocation_join.go:124-145,174-225` also do not inspect that demand.
  Thus the current-guard reading can support N R1 `:151-155` without demanding
  raw U be constant during an actual matched exclusion. Equality of demand at
  related endpoints remains a distinct duty and is exactly where R1-1 applies.
  Ordered mandatory facts and monotonic consumed flags exclude a hidden helper
  round trip, extra publication or early e selection. No contrary atomicity case
  was found in the inspected fixed event interface.
- **Caller framing and delayed selection.** Fixed owner rules are explicitly an
  A_op restriction, not a result of the Environment Boolean. The original
  caller's commit and consume predicates follow commitment/closure/delivery
  facts (`model.go:230-242,375-423`). The local scratch Await projection at
  `callable_join_local.go:32-38` is correctly distinguished from J's S-first gate.
  The immediate caller-match argument and finite delay reasoning do not need
  candidate-private readiness. G_q is correctly classified outside A_op and
  does not establish a fixed-J contextual counterexample.
- **Authorization extension, conditional on coarse transfer.** Production
  `internal/executor/messages.go:103-177` loads the immutable committed intent,
  returns an existing first row before current-source checks, and otherwise
  tests revision/generation before its closed/retired-source exception.
  `internal/executor/invocation.go:123-140` is the bound-message predicate;
  invocations have no reply exception, and `:296-312` retains current admission
  guards. The argument covers first authorization before End and first bound-
  reply authorization after either End status, plus later lookup of the old
  record. Pairing all reachable legal histories, rather than arbitrary row
  assignments on coarse states, is the appropriate formulation. Its normalized
  equality preserves each run's own raw provenance; replay's source-prefix
  requirement at `composition_replay.go:494-526` is not claimed to equate raw
  sequences between implementations. No separate C counterexample was found
  under stable E_N and no sidecar observation by candidates.
- **Progress and scope.** The artifact retains separate all-mode progress and
  nondivergence; it does not use finite safety traces as a rank proof. Its
  once-only authorization count is not presented as a bound on arbitrary private
  programs. S's eventual finite service opens the gate, and pending retains the
  actual unmatched receive with the required consumptions. D_full-to-D_N
  restriction, the no-closure composition environment, full versus residual
  resources, A_graph/A_op/A_pub and source reasoning versus machine proof are
  kept separate. These statements remain conditional on the coarse relation and
  reference/product premises, not evidence repairing R1-1.

## Evidence and disposition

Executed only focused read-only source extraction, `shasum -a 256` for the three
review identities, `git rev-parse HEAD`, and narrow path/symbol searches. One
search named nonexistent `research/substitution/monitor.go`; it returned a
missing-file diagnostic and supplied no evidence. No Go tests, finite census,
solver, candidate execution, public run, manuscript render or Git mutation was
performed. Only this review file was written.

The brief permits an explicit failed obligation, so this remains useful Task 2
research. The present conclusion instead claims the representation-dependent
step closed for every frozen reading. That overstates what was derived and
blocks accepting the artifact in its current form. Retain R1-1 through correction
and independent rereview; finite transcription fixtures cannot by themselves
prove the missing universal representation claim. No whole-phase completion or
permission to begin candidate acceptance is supplied by this review.
