# Phase 1O Task 2 scoped rereview — R2

R1-1: **ADDRESSED** by an explicit additional representation condition and a
narrowed conclusion, not by proving the unrestricted implication.

Spec compliance: **APPROVE**.
Semantic/task quality: **APPROVE**.
New blocking findings in the correction: **none**.

This approval covers the R1-1 correction and its conditional source argument at
the Task 2 gate. It does not approve a candidate, finite implementation, whole
phase, universal ARG-2 theorem or manuscript reconciliation. The original P1
finding remains preserved in the immutable R1 review.

## Scope and identities

Reviewed the appended Fix round 1 report and the dispatched correction diff.
The combined diff output truncated its central section, so numbered excerpts of
the corrected argument supplied the missing certificate/proof text and exact
locations. Read the frozen independent quotient follow-up as a mathematical
input, not as an approval vote. The unchanged K1/N/frozen-O interpretation and
conditional B/C findings from R1 remain the review baseline.

- HEAD: `bec75fa3db8cdafb6295f3d0e8ebb619400c1f36`.
- Correction diff SHA-256:
  `ef16d6437aef0b1ea227f3f33ddbad3e2c99741f5d70f255718def65eef247af`.
- Corrected `docs/foundational/phase1o/02-arg2.md` SHA-256:
  `ca3def80b384335694afcd97c68202e6d93821aad414ac305dc701495599e84c`.
- Frozen `01-weak-contract.md` SHA-256:
  `ac1c0ecbf62bd82c718a7a3ad3dcf000552d14e093868321b0f7dcc4b370b000`.
- Preserved `evidence/reviews/task2-r1.md` SHA-256:
  `2e641ee83da8c9d1ec0ccc0d47f5d4fdd7c283bdc168bf1a4af61c83e9afb45c`.
- Independent `task-2-accounting-quotient.md` SHA-256:
  `4c2c474e4a624a789ffb0c7680146bcb20eacb67bee82d2e99d53111e3aa7040`.

## Why R1-1 is addressed

1. **The representation assumption is now explicit.**
   `02-arg2.md:33-52` leaves unrestricted P_boundary open, including F_event,
   and states Account as an additional sufficient hypothesis. It does not claim
   unchanged ticket laws prove the existence of those maps for arbitrary X.
   `:478-494` repeats this qualification in the conclusion and distinguishes an
   unresolved proof obligation from a specification-gap verdict or counterexample.

2. **The certificate is independent of the target.**
   `:165-220` supplies alpha on original states and lambda on original edges.
   The finite checks are initial ticket/branch values, agreement with existing
   accounting and public protocol/ledger facts, legal ordered ticket updates,
   branch/mode order, and coverage of every original edge with unchanged atomic
   groups. Their validation does not quantify over Rg, require reference demand
   equality, assume current-guard stability, assume bilateral readiness, or
   derive progress. W and Progress retain those separate duties. These are
   finite checks for a supplied finite reachable graph and supplied maps;
   neither certificate discovery nor public-program correspondence is proved.
   The required fidelity to original bookkeeping cannot be replaced by choosing
   convenient new ticket meanings or hiding a graph edge.

3. **Auxiliary triples now yield original-state pairs.**
   `:243-286` uses `(x,k,g)` as proof witnesses and explicitly projects them to
   `Rg subset States(X) x States(K1)`. For any included `(x,g)`, a generating k
   witnesses a triple whose closure handles every actual outgoing X edge and
   required environment move. The existential choice of generating k therefore
   does not weaken the universal outgoing-edge obligation. At merges, fixed
   alpha(x) makes demand independent of the witness/history; each resulting
   successor projects back to Rg. No history coordinate becomes a candidate
   state or input. The existing fixed-reference guard analysis supplies transfer
   of choice stutters; Account is not used to assume that desired conclusion.

4. **The failed quotient is retained accurately.**
   `:124-161` separates operational conservativity from observation descent.
   Demand descends through a projection precisely when it is constant on the
   reachable fibers being preserved. Constancy of the complete history or a
   unique private-PC interpretation is not required. The stronger full-record
   Account condition is correctly described as sufficient, not necessary.
   `:222-227` preserves the possible/excluded local-only merge as a diagnostic
   without calling its lawful continuations resurrection or claiming that no
   alternative faithful certificate could exist.

5. **P_stage keeps its actual source scope.**
   `:66,231-241,478-486` relies on the frozen P_stage reading's original-state
   choice/ticket/demand equality. It retains Rw under that stronger projection;
   it does not derive that projection from the A/B erasure checker or silently
   impose it on P_boundary. The F_event/P_boundary proof now also requires
   Account, correctly distinguishing visible-history alignment from a
   state-respecting demand observation.

The correction preserves the original atomic groups, current-guard argument,
caller ownership, delayed selection, conditional legal authorization-history
extension and separate progress requirements. R1's corresponding source checks
remain conditional findings; none is promoted into a repair of unrestricted
accounting representation. No new breakage was found in the changed argument.

## Evidence and remaining obligations

Executed focused source/report reads, SHA-256 checks and `git rev-parse HEAD`.
A read-only Markdown check of corrected 02 found zero missing local link targets
and zero trailing-whitespace lines. No Go tests, semantic probes, solver,
census, candidate/public execution, render, commit or other Git mutation was
performed. Only this R2 review file was written.

Unrestricted P_boundary ARG-2 remains **OPEN** for both F readings. A universal
state-accounting/quotient/selection theorem or a different suitable original-
state relation is still required. Account necessity, candidate acceptance-set
differences, a conforming negative, public/compiler correspondence and uniform
operational bounds remain unestablished. Any subsequent finite validation or
implementation needs its own scoped gate and cannot turn this conditional
source approval into universal qualification.
