# Phase 1O — Result classification and public realizability

Status: **source classification with reviewed finite Task3 outcomes; scoped
Task4 completion-update review pending**. This records the conditional source
result approved by
[Task2 R2](evidence/reviews/task2-r2.md), preserving the rejected universal claim
and [R1 finding](evidence/reviews/task2-r1.md). The finite evidence below is
approved only within [Task3 R2](evidence/reviews/task3-r2.md)'s stated scope; it
does not establish candidate acceptance or the open universal implication.

Source baseline: `ffd00be531fbb986b86160bd6c22a87345fee14a`. R2 identifies the
corrected [argument](02-arg2.md) by SHA-256
`ca3def80b384335694afcd97c68202e6d93821aad414ac305dc701495599e84c` and frozen
[interpretation](01-weak-contract.md) by
`ac1c0ecbf62bd82c718a7a3ad3dcf000552d14e093868321b0f7dcc4b370b000`.
The controller pins [03-finite-contract](03-finite-contract.md), including §8,
at SHA-256 `a41c2216caa80d979365e61470dce6fc03c2616ad6f2faeab2e2431b0418a083`.
These are inherited source identities, not newly rerun checks or runtime evidence.

## 1. Exact claim and scope

The unchanged unrestricted target is:

```text
forall X in A_op:
  (exists Rw. W_N^sigma(X,Rw)) and Progress_N(X)
    => exists Rg subset States(X) x States(K1).
         ARG1Compatible_N(X,Rg), under E_N.
```

For **P_boundary under either F_event or F_fact, ARG-2 remains OPEN**.
Visible-history alignment does not by itself make demand a function of the
original X state. No universal accounting/quotient/selection theorem is supplied.

The approved source result adds `Account_N(X; alpha, lambda)` on the original
states and edges. With W and separate Progress_N it yields a suitable
original-state Rg under E_N. Account checks initial tickets, faithful state
accounting, ordered original edge updates, branch/mode order and all-edge/atomic
group coverage. Its checks do not assume reference-demand equality, ARG-1,
readiness or progress. It is an **additional sufficient representation
hypothesis**; its universal existence and necessity are unestablished.

Under frozen **P_stage**, under either F reading, W already compares
original-state choice/ticket/demand stages. The source argument retains Rw and
derives the remaining bridge duties
under the named assumptions. This closes the source implication for that
stronger reading, not a derivation of P_stage from P_boundary or the historical
A/B single-event erasure checker.

D_N is D_full restricted to depth2/cap3/residual I=R3/q. Full W, Progress and
Account, when supplied, restrict configuration-wise to D_N, giving a full-premise
corollary for the sufficient class. A D_N certificate says nothing about
uncertified full configurations. An open-port fragment is narrower still. Local
closure/scarcity safety is distinct from the no-closure composition environment;
no new whole-profile or full-contract qualification is established here.

## 2. Premise provenance and lifting boundary

| Obligation | Derived source portion | Independent or additional premise |
| --- | --- | --- |
| A: guards and demand | Fixed extra choices preserve current constructor/protocol guards. Account supplies original-state normalization; P_stage supplies endpoint stage/demand equality directly. | Account existence for arbitrary P_boundary X remains open. Current-guard preservation is not demand equality. |
| B: caller and delayed selection | W plus fixed original caller rules gives immediate caller matches; original finite S service opens J's selection gate. | Owner framing belongs to A_op. Eventual eligible service and Progress are separate; the Environment Boolean supplies neither ownership nor readiness. |
| C: authorization | Given coarse transfer under E_N, compatible legal first-authorization histories have the stated retained-record extension; each run retains its own raw provenance. | Stable authority/binding, inaccessible sidecars and the coarse relation are premises. Permitted authorization histories do not repair missing X demand representation. |
| Composition | ARG-1-compatible relations support conditional lifting. | Reference/product premises, including the required J[K1] premise, remain necessary for K_join/G_outer preservation. Schema compatibility does not establish them. |
| Progress and bounds | Completion, designated pending and divergence remain distinct. | Candidate all-mode Progress_N and adequate concrete operational limits remain separate. Finite authorization counts are not a private-program rank or uniform runtime bound. |

E_N retains J's prior debit1, full I=R4, original S as sole q, stable
identity/authority/bindings, the open eligible caller, no closure/retirement/
release, eventual eligible service and adequate finite limits. These composition
premises do not follow from weak safety. A_op is narrower than arbitrary A_graph;
inspected public owner separation does not prove A_op=A_pub or general
compiler/runtime correspondence.

## 3. Failure classifications

| Object | Permitted conclusion |
| --- | --- |
| Accounted local example with premature Rw | Bad supplied strong witness: demand differs at private boundaries. A normalized Rg is supplied for the declared fragment. Rw failure is neither failure of every relation nor failure of J[X]. |
| Unresolved local/helper fork | The premature local pair cannot handle the actual helper commitment; normalized Rg preserves both actual choices. Pair/path strategy failure is not NONE. |
| Lawful possible/excluded merge | Representation diagnostic: two lawful history accounts over one raw m give different proposed demands. It refutes that supplied map/history-descent argument, not every faithful map or relation. No resurrection occurs. |
| Exclusion followed by imported helper commitment | Invalid original-history/accounting certificate: excluded/disabled h cannot commit. Reject it before weak-conformance or contextual classification; it is distinct from the lawful merge. |
| G_q | Analytic A_graph-only control: deleting caller selection while q is enabled can break local ARG-1 readiness, but violates A_op owner framing. In fixed J, S has spent q before e selection. No fixed-J failure or public realization follows. |
| Private divergence or postponed environment service | Separate progress/readiness failures. Matching completed outputs is insufficient; toy controls establish no A_op negative. |
| False late bound-reply denial or changed first record | Invalid legal/provenance certificate under stable binding. Legal revision/generation denials and nonreply checks keep their changed outside-profile premises. |

No conforming A_op negative, certificate-implication counterexample,
no-eligible-witness result, K_join violation, candidate acceptance-set difference
or exact specification gap is established. The missing proof is not classified
as intrinsic impossibility. Failure of a sufficient proof method warrants
neither strengthening unchanged K1 nor repairing the runtime.

## 4. Diagnostic minimization and finite boundary

The retained one-merge diagnostic is [03 §3.4](03-finite-contract.md): two raw
edges u→m align respectively to staying at B and lawful exclusion at L, followed
by one shared local reply edge m→z. The submitted history map requires both
`(2,2,0)` and `(1,1,0)` at m. Retain the common original entry-commit/admission/
input prefix, fixed spending, h possible/excluded distinction, unchanged m
identity and shared continuation. q/closure/later outcomes remain named
unassessed ports. Local future behavior does not retrospectively exclude h.

This is a small causal diagnostic, **not proved globally minimum**. Minimization
must preserve original premises, the two lawful accounts over one original
state, atomic boundaries and the continuation showing no resurrection. Splitting
m by history, deleting an incoming account, changing initial entitlement or
importing a helper edge changes the question. Before calling any future object
a counterexample, establish original admissibility, weak conformance, progress,
environment and, for contextual failure, J[K1] premises. Preserve the required
branching certificate rather than demanding that every negative be a linear run.

Task3's corrected finite implementation received independent **APPROVE** for
specification and implementation/certificate quality in
[Task3 R2](evidence/reviews/task3-r2.md), SHA-256
`ca8927d6351cacafab5c5e55ea987c8f5920ea936fcb996d7fc1d519d7a5848a`.
Its [final index](evidence/core/task3-r2-index.json), SHA-256
`0073057241d94f15c51abd635da7970c622522eadbc69901c5481107702d12f6`, pins
source identities, compressed records, exact argv and exits. These are recorded
Task3 results, inspected here without rerunning tests:

| Fixture family | Actual scoped result and limit |
| --- | --- |
| F1–F4 | Alternative/nonfunctional supplied relations pass; noncontiguous paths are invalid certificates. F2 rejects missing immediate bilateral readiness; F3 accepts weak safety while its private cycle falsifies progress; F4 rejects the extra visible occurrence. These qualify toy machinery, not ALAN admissibility or whole-profile conformance. |
| F5 and invalid-history controls | False late bound-reply denial is rejected as invalid provenance; a well-formed unreachable control does not fail a reachable obligation. No semantic NONE verdict. |
| Original branch/accounting and raw merge | The branch fragment has 22 states/31 edges. The raw m diagnostic retains demands (2,2,0) versus (1,1,0) and rejects that supplied map; the imported excluded-helper continuation is a separate illegal-prefix control. These are finite rule instances, not W_N/W_full or complete progress. |
| §6 normalization/helper fork | In the declared F_fact/P_boundary fragment, both supplied relations pass weak checks over 16 pairs, 12 component and 24 bilateral environment obligations each. Premature Rw has exactly eight Strong issues, all endpoint demand inequalities; normalized Rg has zero. The helper fork rejects the premature pair strategy while the normalized alternative handles both choices. Named endpoint-demand/current-guard checks are not full ARG1Compatible_N or full Account_D. |
| §7 six J reply schedules | All six original diagnostic-reply schedules check both End statuses, legal authorization/admission timing and exact saved-record lookup. Unresolved-admission, invalid-certificate and changed outside-profile controls retain their distinct outcomes. These are clause checks, not runtime execution or arbitrary-relation sidecar extension. |
| Positive ranks/negative private cycle | Complete toy progress and its rejection controls are checked separately from safety. Open-port fragments retain no complete progress/conformance verdict. |

The final normal and race records each pass 26 selected top-level tests,
96 test/subtest events and 85 leaf cases, with no failures/skips. Recorded
`go vet ./research/substitution`, `go mod verify` and `gofmt -d` on the seven
owned files succeed. Unfiltered `staticcheck ./research/substitution` remains
exit 1 with six historical diagnostics in unchanged files; it is not a green
whole-package staticcheck result. Verification used Go 1.27.3 darwin/arm64,
offline/readonly modules and the recorded environment, excluding ALAN_* and
OPENAI_API_KEY. Exact selections/commands and hashes are in the final index.

The [preserved Task3 R1 rejection](evidence/reviews/task3-r1.md) remains evidence
against the earlier validator. R2 corrects omitted-grammar acceptance, genuine
private selected-field preservation and an overstrong unmatched-silent demand
check without changing frozen01/02/03. The old Rw count of 12 includes four
`silent_demand` issues from that extra implementation requirement; it is
historical output, not the final result or a target-contract obligation. The
retained R2 RED record has 19 failures/11 passes in 30 events against the earlier
core; the corrected focused record passes all 30. R1 findings were corrected
and rereviewed, not waived. These finite repairs do not prove the universal
representation theorem or universal Account existence.

Sharing inspected original rules establishes transcription checks, not
independent proof of those rules. Supplied paths are not an existence search;
a rejected path, relation, rank or map is never NONE. No search-completeness or
new full-profile census claim is intended.

## 5. Public-realizability decision

**No new public execution campaign is justified by this result.** The open
question quantifies over every permitted P_boundary candidate representation;
one public execution cannot establish that universal original-state theorem.
The approved conditional proof introduces no execution distinction requiring
realization. No qualified model contextual failure needs authentication and no
new public-candidate/compiler correspondence claim is made. An unrelated
accepted run would not supply the clause fixtures' missing premises.

This does not assert that the raw merge or any arbitrary graph is publicly
realizable or unrealizable. A separately authored candidate becomes informative
only for a concrete question: checked noncanonical public correspondence,
authentic realization of a qualified counterexample, or the exact interface
limitation of that witness. First fix candidate/consumer and map emitted handles,
selected observations, authority/binding decisions, charges and publications
under supported operations and unchanged scheduling or a labeled supported
driver. Retain J/O source/policy and document any new bindings/catalogue
identities. None of these new claims is needed here; no authoritative state
change or invented outcome is proposed.

## 6. Remaining obligations

1. Preserve Task3's accepted scope, final evidence and historical rejection; do
   not promote supplied finite checks into universal or whole-profile results.
2. Independently review this Task4 finite-outcome completion update. The source
   classification and no-campaign decision remain unchanged; Task3 approval does
   not itself approve this update or the whole phase.
3. Resolve unrestricted P_boundary ARG-2 with a faithful original-state account,
   a demand-preserving conservative quotient/selection, or another suitable
   original-domain relation; alternatively supply a fully qualified negative or
   demonstrated specification gap. Current evidence selects none of these.
4. Keep Account necessity, candidate acceptance-set differences, A_pub/compiler
   correspondence and uniform operational bounds unestablished unless separately
   proved/qualified. A finite fragment or single trace supplies none of them.
5. Reconcile final accepted claims with both manuscript representations in Task5.
   No code–paper reconciliation or manuscript change is claimed here.
