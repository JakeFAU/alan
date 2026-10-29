# Phase 1I independent review

Independent whole-working-tree review returned **APPROVE** at source identity
`a97e9893e2218a2245953e720de4939c7aca53503b51ab2e9e76a1164ee12c38`.
The [final report](evidence/reviews/final-review.md) covers the complete source,
contracts, corrected evidence and affected manuscript claims. Exact postcommit
identity verification and draft publication follow; approval is not a proof.

The independent [claim audit](evidence/reviews/claim-audit.md) distinguished the
exact-debit relation in Definition 4.6 from a weak schema/result contract, and
identified missing interference, invocation and scheduler arguments with direct
recovery dependencies. The separate [API audit](evidence/reviews/api-audit.md)
verified public construction, accounting and reusable process boundaries.

The fresh [contract review](evidence/reviews/contract-review-v0.md) returned
REJECT before candidate implementation. Its three actionable findings were:

- R1: define the separate local relation, environment events, C[K1] reference
  product and contract-derived resource entitlement before authors start.
- R2: distinguish local Success without a Send from the published
  `failed/missing_reply` envelope and wrapper failure.
- R3: specify caller and S behavior for every census mode and refusal.

The [original draft](evidence/reviews/contract-v0.md) is retained. A separate
contract implementer supplied [revision 1](evidence/reviews/contract-v1.md) and
its [fix report](evidence/reviews/contract-fix-v1.md). The same independent
reviewer [approved all three corrections](evidence/reviews/contract-rereview-v1.md)
at contract digest `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`.
The frozen document's pending-status header records its review-time state; this
exact-digest approval is the subsequent status. This is specification inspection,
not an executed counterexample, conformance result or proof.

The independent [candidate/context review](evidence/reviews/candidate-context-review.md)
returned APPROVE after checking both frozen first versions, all 44 author
samples, the schema-only shared package and ten context wiring cases. It verified
source/artifact hashes and inspected the full seven-file source diff. It did not
rerun those tests or approve the separate model, census or process evidence.

The independent [model/correspondence review](evidence/reviews/model-review-v1.md)
returned REJECT on two evidence defects. It reproduced five isolated corrupt
inspection inputs that the decoder incorrectly accepted: consumed payload,
provenance and generation, candidate local terminal status, and missing member
freezes. Publication data was being copied into the consumption event, and
local-terminal/freeze facts were diagnostic only. These falsify the decoder's
sensitivity claim, not the runtime's record integrity. The reviewer independently
reran the model suite, matched emitted evidence, and reproduced both real
root-success counterexamples without finding a blocking model-calculation defect.

The separate [runtime/process review](evidence/reviews/runtime-review-v1.md)
returned REJECT because all nine cancellation cases in both normal/race records
were normalized as `Closed=false`. It independently reran 236 runtime leaves and
two consumer race cases and inspected the archived process/endpoint records.
Those passes did not repair the omitted logical-cancellation check.

All original reports, source, failed inputs and logs are retained. One coupled
test-only correction addresses actual selected observation data/identity,
member terminal/freezes and explicit root cancellation. The independent
[decoder rereview](evidence/reviews/decoder-rereview-v2.md) returned APPROVE after
61 focused cases: five original attacks, 33 additional evidence controls, three
cancellation-evidence controls, 18 original closure records and two actual root
fence witnesses. It checked that raw validation is required for weak/PQ and
intentional application faults too. This is a finite evidence validator, not a
general corruption detector or a second runtime reducer.

The fresh whole-branch reviewer identified a further probe-correspondence gap:
the first P/Q model used B-derived control, and the weak helper's internal value
differed from its actual artifact. Those surrogate sources/results remain under
model evidence. The correction maps all 41 actual probe instructions across six
retained programs and checks local relation before original-protocol trace
languages. Runtime-boundary membership remains a separate check. The final
reviewer independently reran the complete model (47 leaves), all 59 decoder
controls, two real root-fence cases, 26 probes and two consumer race cases.
All 33 fresh model result objects match retained payloads. It audited both final
295-case normal/race logs and raw probe/closure/fence records; the full suites
were run by the qualification workstream rather than repeated by the reviewer.
Frozen K1, A/B, runtime guards and original counterexamples remain unchanged.
Review agreement never substitutes for bounded checks or proves universal behavior.

The final packaging audit found two archival `.go` snapshots that created an
unintended failing Go package. Byte-preserving `.go.txt` renames and a fresh
31-package inventory resolved it. The reviewer checked hashes and package
discovery. No qualified model/runtime source changed. All original actionable
findings above are resolved; general theoretical obligations remain explicit.
