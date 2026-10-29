# Phase 1J qualification

The proposed **all-schedule completion guarantee is falsified for simple J1**.
The retained join preserves the checked root-lifetime and resource obligations,
but introduces a separately exposed communication boundary: a peer may consume
S, commit its report, terminate, and then have the report's first source
authorization refused as `source_closed`. The waiting root receives no report.
This is a finite model counterexample tied to the existing authorization rule;
it is not an observed failure of the unchanged automatic `Store.Run` scheduler,
and it is not an impossibility result for every public join protocol.

Concrete unchanged-runner samples establish a narrower positive result: retaining
the root removes the original delayed-peer admission failure in those executions.
They include the decisive intermediate state and four real process interruptions.
The original C0 failure remains reproduced at both the original and larger
resource allowances. Source authorization, required scope lifetime and resource
quantity are distinct obligations.

## Baseline, intervention and independence

Fetched main baseline `d68ec667d276a4bd9e1be11613905c628cfea704`, tree
`25c2e22f21181fe70d8f2b2f8bc7e44c512ef948`, contains merged Phase1I PR18 and the
user's TeX formatting fixes from PR19. [Baseline records](evidence/baseline/baseline.json)
identify source hashes and actual initial negative-test commands. Before final review, the
work was rebased at the user’s request onto `5764ec0b5679c72c80e56c44fd9e4198c986b91c`
(tree `729e93d5d83422307023196e8a7c5677fb8e71f1`), incorporating PR20’s separate
structured-loop theorem correction and GitGuardian cache ignore rule. All524
Phase1J files retained their exact hashes; production/test inputs were unchanged.
The original
checkout and untracked GitGuardian cache were preserved.

J1 was independently reviewed and frozen before implementation at SHA256
`a950450d33ab928ccb875f8c98de83a83c956d8759fa0f39a71afe1c06c52a81`.
The context, model/oracle and runtime adapter were written by separate agents.
[Reference freeze](evidence/model/reference-freeze.json) retains exact model/oracle
bytes from before the new context artifacts were inspected. The independent
mapping subsequently checks actual artifact control; this is not a compiler proof.
The reference and concrete products reuse reviewed Phase1I pure primitives, so
independence is qualified at that shared representation boundary.

Only new ordinary application/model/test files and Phase1J research evidence are
added, alongside scoped manuscript additions. A/B/S template/program bytes,
original K1, original contexts, production compiler/executor/schema/accounting,
dependencies and scheduler are unchanged. New root/peer program identities,
source excerpts and exact compiled diffs are in [context evidence](evidence/context/implementation-report.md).
The root's first two instructions and peer's first 34 are identical to C0.
The peer now reports every known S outcome then ends locally succeeded; the root
joins both actual outcomes before choosing success or failure. The report is an
authenticated peer assertion, not a transferred runtime receipt.

## Finite model result

Eight original input configurations per graph: id1/2, value0/1, answer mode,
I=R3, depth2/3, E0, static root+peer. The declared eventual-service assumption does
not require authorization to succeed. Normal terminal actions and failed first
authorization remain represented. All eight configurations in every graph have
both successful and report-refusal pending maximal states.

| Graph | Initial states | Reachable states | Edges | Maximal states | Successful | Pending |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| K1 reference | 8 | 12,320 | 25,648 | 32 | 16 | 16 |
| Frozen A / local | 8 | 6,160 | 12,620 | 16 | 8 | 8 |
| Frozen B / helper | 8 | 11,260 | 23,124 | 16 | 8 | 8 |

Resource-prefix, root-lifetime and truthful-report failures are zero in these
conforming products; cycles, rank failures and cutoffs are zero. Each edge is
checked, including revisits. Per-configuration records, exact abstract path
counts and witnesses are in [model evidence](evidence/model/current/).
Maximal-state counts are not path counts or concrete runtime experiment counts.
The 32 prefix Sets are modeled individually; finite pure suffix computations are
mapped as stuttering. Counts concern this declared abstraction, not every raw
instruction interleaving. Report-before/after-candidate orders are both reachable.

The new negative does not restore Phase1I's missing C[K1] completion premise,
prove general congruence, or equate candidates with unequal permitted charges.
It isolates another failed premise while retaining the root-lifetime result.

## Concrete causal matrix

All primary execution uses unchanged Store.Run, physical capacity1, original
MaxSteps200/MaxDispatch200, id1/value0, and E0. The normal harness has **36 passed
leaves**: 13 matrix cases, four C0 controls, four recovery cases, cancellation,
13 decoder/oracle mutations and duplicate delivery. Exact case names and raw
retained records are in [current normal log](evidence/runtime/oracle-v2-normal.log).

| Case | Observed result / qualification |
| --- | --- |
| C0 A/B, I=R3 and4 | Root locally succeeded; S admission refused scope_closed; peer failed; aggregate failed/member_failed. More quantity does not preserve lifetime. |
| C_join A/B, I=R3 | Candidate consumed while peer S unadmitted and root open at mandatory receive; then actual S/report consumption and local/aggregate success. Dynamic spending A=(2,2,0), B=(3,3,0), fixed runs2. |
| B I2/R3 and I3/R2 | S refused invocation_limit or created_run_limit respectively; actual refusal reported; root joins then fails. No invented capacity. |
| Premature-report peer | Schema-valid ordinary assertion can produce root success; independent oracle rejects premature_report because original S consumption did not precede report. |
| Wrong-request peer | Runtime admits ordinary report; root consumes it and parks; oracle identifies request_correlation. No successful join discharge. |
| Candidate fail/absent and S fail/absent | Actual member_failed or missing_reply evidence retained and reported; root joins then fails, peer reporting program succeeds. |
| Candidate pending; pending candidate with S failed; S pending with candidate failed | Root stays open at required wait. A completed or failed sibling does not discharge pending required work. Bounded observation, not a proof of infinite execution. |
| Explicit cancellation | Granted ordinary lifecycle cancellation retains actual interrupted status/charges; outside positive completion premise. |
| Duplicate delivery/replay | Original logical report identity retained, no second consumption/new work; replay leaves records unchanged. |

The independent adapter checks the original selected observation and handle,
source/generation/obligation, report payload and causal ordering, authorization
snapshot, delivery/ack linkage, local/aggregate outcomes and spending. Mutations
are apparatus probes, not claims of production corruption. Neither publication
nor final register values substitute for actual consumed evidence.

## Four process cases

For each A/B: (1) candidate consumed, report uncommitted and S still unadmitted;
(2) original report committed, root has not consumed it. Existing test boundaries
acknowledge the retained predicate; the controller sends SIGKILL and starts a
fresh process on the same database. Original runs/generations/artifacts,
invocation operations, retained facts and already-committed report identities
survive; activation epochs renew, and resumed execution completes. Replay is
separately checked without mutation. These are single-host process-crash samples,
not power-loss, distributed or infinite-interruption results.

## Verification and preserved corrections

Context public API tests, context race, standalone consumer normal/race/build,
new model checks, focused raw/process checks, and unchanged author/schema/semantic,
invocation/closure/replay and old substitution-model regressions were executed.
The earlier complete race run used the preserved v1 oracle; after two sequential
oracle checks were added, the entire normal suite and affected runtime/decoder
race selector were rerun. Unchanged process race coverage retains its original
source label; no full latest-oracle race claim is made. Their logs and exact commands live under `evidence/context`, `evidence/model`,
`evidence/runtime`, and [unchanged gates](evidence/gates/unchanged-gates.json).
The complete final index/source inventory and final review disposition accompany
phase completion; see [review](03-review.md) and [runbook](04-runbook.md).

Preserved development failures include the context's initial unimplemented RED,
the model's false completion assertion, and two raw adapter errors (incorrect
initial-projection field name and a peer-source filter that also selected
invocation request observations). Fixes narrow the observation adapter; they do
not repair production semantics or erase the new negative. Original sources/logs
remain in each workstream's development evidence.

Raw public Provision catalog arrays can vary in order. Archived hashes identify
the actual retained bytes; reproduction compares program/template bytes and
keyed provision semantics (Programs sorted by ID), not whole raw Provision JSON
byte equality. Model/runtime correspondence, manuscript synchronization and
independent review are evidence with distinct strength, not mathematical proof.

## Remaining obligations

Simple J1 has no all-schedule completion qualification. Required ordinary reports
need a justified source-authorization/delivery lifetime condition or a separately
specified protocol; no general repair or impossibility theorem is established.
Arbitrary contexts, weakest lifetime conditions, compiler correctness,
exact-debit congruence, distributed recovery and effectful/unknown
outcome recovery remain open. The structured-loop theorem correction is inherited from PR20,
not a Phase1J result or an obligation reopened by this experiment. Bibliography is unchanged because no new external
source or citation is introduced. Markdown/LaTeX additions preserve existing
open theorem status; the single end-of-phase Tectonic invocation exited 0 and produced a 17-page PDF
with accepted warnings. Pages 12–13 containing the addition and its continuation
were visually inspected; see [build record](evidence/gates/manuscript-build.json).
