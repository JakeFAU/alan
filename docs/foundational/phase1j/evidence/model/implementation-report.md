# Independent model/oracle implementation report

Question: does J1's retained root join prevent the original scope-admission fence
and establish the frozen completion guarantee? **Bounded result:** the modeled
root-lifetime/resource obligations hold, but all-schedule completion is falsified
in every one of the eight K1, A and B configurations. Peer report authorization
can first occur after peer Success and be permanently refused. A successful
Store.Run sample would not invalidate this broader fair-interleaving witness.

## Independence and inspected scope

The author read AGENTS, the task brief, reviewed K1/model/candidate primitive
sources, old boundary evidence and existing ordinary-message authorization code.
J1's approved contract digest is
`a950450d33ab928ccb875f8c98de83a83c956d8759fa0f39a71afe1c06c52a81`.
Reference K1 products were executed before A/B products in the first test run.
`reference-freeze.json` and `reference-freeze-sources/*.go.txt` retain exact source
bytes before any new context artifact or joined runtime-output inspection.
New root/peer compiled artifacts were inspected only after that freeze. No joined
runtime output was used to derive expected responses or reference transitions.

Subsequent self-review tightened two pre-existing J1 constraints: distinct extra
peer reports and root report consumption before candidate consumption. The prior
oracle remains archived; `oracle-review-correction.json` records the change,
which occurred after compiled-context inspection and before any runtime-output inspection. It is a disclosed oracle correction, not part of the pre-artifact freeze. The normalized API is unchanged.
Later additions cover unexpected stuck classification, ledger/depth checks,
exact source-refusal witness ordering, compiled correspondence and positive
witness retention. No production or previous-phase source was changed.

## Evidence and meanings

`summary.json` contains exact totals. `current/census-*.json.gz` contains all eight
configurations per product, exact reachable-state/edge/maximal-state counts,
strict-rank/cycle/cutoff counts, exact abstract maximal-execution counts, and
shortest decisive intermediate, successful, and report-refusal witnesses.
`current/c0-comparison.json.gz` retains all eight original configurations for
K1/A/B and their root terminal fence failures. Normal tests recognize these
negative results; they do not reclassify completion entitlement.

All joined products have zero resource-prefix, ledger/depth, root-lifetime,
report-causality, rank or unexpected-stuck failures. Report delivery can precede
candidate consumption or follow it, and each configuration reaches the decisive
state where candidate is consumed, S remains unadmitted and root remains open.
Every terminal successful A/B state spends respectively (2,2,0)/(3,3,0), with two
static provisioned runs outside that debit. The inherited K1 optional helper
choice produces both debit outcomes. The report has no invocation/effect debit.

Every pending counterexample explicitly orders S consumption → report commitment
→ peer terminal → denied first authorization → refused delivery; the root has
consumed candidate, remains open, has no aggregate publication and is not
cancelled. No hypothetical scheduler cutoff is used. The finite rank proves
acyclicity of the enumerated graph, not progress of a permanently unavailable
report. Fair service permits the denied authorization and cannot repair it.

`artifact-correspondence.json` maps all 28 root and 40 peer instructions for both
candidate variants (136 entries), preserves full operands and hashes, and checks
original first-two root/first-34 peer instructions. Pure suffix instructions use
an explicit finite-stutter abstraction; path counts concern that abstract graph.
The independent JSON evaluator tests 6,400 root outcome combinations per variant,
40 peer status/presence cases per variant, and wrong-request pending. Absent
response bindings are omitted, so reading an absent value fails the evaluator.
A13/B16/helper7 mapping is reused from unchanged reviewed Phase1I and rechecked;
original source and compiled identity evidence is retained. This does not prove
the author/compiler correct or establish arbitrary-context correspondence.

`completion-hypothesis-red.txt.gz` retains the failing assertion that all positive
maximal states complete (exit1). `completion-hypothesis-red_test.go.txt` and
`reproduce-red.py` reconstruct it without changing production/model sources.
`commands.json` records executed commands, environment, expected/actual status
and output paths. Current model tests, focused old artifact correspondence,
model race and vet checks are recorded separately; a failure means evidence,
not permission to weaken the graph or oracle.

## Files, assumptions and limitations

Changes are confined to `research/join` and this model evidence directory.
Libraries import neither author/compiler/schema/runtime/executor/storage nor
candidate constructors. Tests inspect frozen JSON artifacts and source hashes.
The pure graph contains no real process-kill implementation: recovery and raw
adapter qualification belong to separate concrete workstreams. Raw provenance,
selected observation identity, source disposition persistence and original-store
recovery must be established there before normalized oracle checks are trusted.

Results assume the frozen one-request/generation1 topology, known pure K1/S
behavior, no external cancellation, valid grants, finite operational bounds and
eventual enabled service. The graph is not a model of every endpoint, arbitrary
run retirement, dynamic participant set, infinite crashes or effectful recovery.
Ordinary report authorization remains the next unresolved protocol boundary.
No handshake, privileged join, runtime repair, new budget, compiler extension,
confluence theorem or universal substitution claim was introduced.

Final status: model/oracle implementation complete for the bounded negative scope.
Current v2 normal and race suites and vet exit0; reconstructed false-completion
hypothesis exits1 as expected. `current-source-hashes.json` binds final source;
`evidence-manifest.json` binds retained evidence. Independent whole-task review
and concrete runtime/process/manuscript conclusions remain controller-owned.
