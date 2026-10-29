# Phase1I independent model handoff

The central result is a bounded negative: runtime root Success closes admission
for a still-open static peer. With the frozen parallel caller, I=R=3 and depth2,
the peer can therefore fail S admission despite initial K1 completion entitlement.
No explicit cancellation is needed. The frozen contract and 84-entitled partition
were not changed. This falsifies runtime correspondence / the C[K] completion
premise, not an implication that assumes a true C[K] premise.

## Executed census

| Interpretation | Configurations | Entitled | States | Edges | Failing entitled | Unaffected entitled complete |
|---|---:|---:|---:|---:|---:|---:|
| boundary/K1 | 3840 | 84 | 142688 | 225260 | 8 | 76 |
| boundary/local | 3840 | 84 | 95840 | 147358 | 8 | 76 |
| boundary/helper | 3840 | 84 | 128968 | 202184 | 8 | 76 |

All eight failed entitled configurations are parallel, for each graph. Extension
safety checks, stuck states, rank failures and cutoffs are zero. All generated
successors are checked before state deduplication; full-state identity includes
private control/registers. Positive-graph acyclicity is checked by intrinsic rank.
The shortest K1 completion failure is 15 model events; transcribed A/B shortest
traces retain their pure instructions. The JSON counterexamples retain every event.

Corrected original-v3 contract-only products remain separately positive:
K1 141760 states/224564 edges, A95264/146926, B128616/201920, each3840/84.
These do not establish runtime correspondence. Separate local relations each
cover3072 initial environments, A81260 pairs and B118408, with bilateral peer/
closure service checks at every private instruction boundary. No-closure progress
is checked separately (A33736/B44100 states); closure-enabled maximality is not
used to make progress vacuous.

## Distinguishing probes and independent negatives

Weak input0/output0 sequential probes: budget1 both fail,2 local succeeds/helper
fails,3 both succeed; I3/R2 repeats the budget2 distinction. Unequal debits do not
refute Definition4.6. Equal-charge P/Q add one private Set to Q before the same
helper. Local weak relation checks pass first (58/123/160 paired states); exact
finite epsilon/subset trace-language equality then passes parallel budgets1..3
(124/223/231 paired subsets). Concrete Store.Run schedules remain separate.

The unchanged pre-inspection CheckTrace rejects the wrong-id model (shortest
accepted-reply counterexample8 events) and extra-helper commit (16 events), and
the latter reaches an accepted third admission under I=R3. Its deliberately
forbidden ticket creation also violates the reference rank; that diagnostic is
retained and does not substitute for extra_helper detection. No acyclicity claim
is made from a failed rank for that malformed graph. Frozen oracle SHA256 is
9f7f5f59414c439392ab813759c6ed1c99f4be21aaa44f03a24cf953eb3899ad.

## Preserved corrections and trust boundaries

- Reference/oracle frozen before candidate inspection: reference-freeze.json,
  reference-v1-sources/, reference-v1-results/. Both author artifacts were read
  only after this freeze.
- V1 collapsed failed child consumption and terminal freeze. V2 separated them;
  original source/results retained.
- V2 leaked consumed-helper scratch into candidate public response presence.
  Runtime evidence exposed it after shared protocol/state checks missed it.
  Direct frozen-contract fail/absent/pending regression failed RED, then v3 split
  Returned* fields from publication fields. pre-payload-fix-v2/ and
  payload-leak-red.log retain superseded results and failure. This demonstrates
  the earlier shared-protocol validation limitation; it is not runtime repair.
- The boundary extension's direct peer cancellation test first caught
  source_closed versus correct scope_closed. peer-boundary-red.log retained.
  Extension leaves static peer open; cancellation fences scoped admission.
- artifact-correspondence.json covers every A13/B16/helper7 retained instruction,
  full operands and receive guards, both branch arms, original helper handle,
  and unreachable pending-tail instructions. JSON decoding is trusted;
  interpretation is independently hand-written, with no production imports.
  Inspected correspondence is not a compiler proof. Independent review is pending.
- Five K1E scenario automata check reservation/launch and known/unknown retention
  only; they are not an effectful artifact transcription or full reply protocol.
  Mathematical reopen is not SIGKILL evidence. Named pure closure graphs are
  separately counted and absent A helper boundaries are inapplicable.

## Interfaces and files

Only research/substitution and docs/foundational/phase1i/evidence/model changed.
README.md explains every model product and remaining limits. APIs:
CheckTrace(Request,[]Observation); CheckCandidateContext(Config,Variant,
ContextObservation); CheckBoundaryContext(...) (Census,error). Local=A,
Helper=B. Boundary membership returns retained guarantee counterexamples so
matching a runtime failure cannot erase a failed Gc claim. Open root/peer are
explicit open; absent unpublished candidate status/reason remain empty.

Evidence is under docs/foundational/phase1i/evidence/model, with compressed full
per-configuration censuses, shortest traces, summaries, source manifest, REDs,
and final-tests.log/final-vet.log. No paid calls, production changes, subagents,
commits, or full-suite runs. Exact commands are in final-source-manifest.json:
focused package tests PASS6.131s, vet exit0, gofmt no output.

## Remaining obligations

Independent model/correspondence review; sampled runtime trace/projection
membership and process recovery are other workstreams. The root admission
counterexample needs retained-runtime confirmation (controller/runtime workstream
is executing it). No arbitrary-context theorem, compiler correctness, full
concrete-runtime exhaustiveness or effectful substitution is claimed. The
contract remains unchanged; any future repair requires its own justification and
review. K1E automata are narrower than full effect protocol qualification.
