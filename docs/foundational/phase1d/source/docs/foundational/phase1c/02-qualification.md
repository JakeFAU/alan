# Phase 1C qualification and counterexamples

Status: independently reviewed bounded research package, 25 October 2026. Specification and quality passed; the final verdict is **APPROVE for Phase 1C bounded research-deliverable readiness**. This is not approval of a production specification or proof of unbounded correctness.

## Baseline and reproducibility

The starting tree was clean at `82dc5649cfe0eccb2fde3ecea1e12cc851377af5`. Its model/test source digest was `afc89e4bf24126b33da5341f1bc4c582c81b0b80f51a04af1674daacf0b3c49a`, independently recomputed using the retained Phase 1B manifest algorithm. The installed compiler reports `go version go1.27.3 darwin/arm64`.

Before model edits, these commands passed:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off go test -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off \
  go test ./research/semantics -run '^(TestQualification|TestCompositionQualification)$' -count=1 -v
```

The full baseline suite reported `ok github.com/JakeFAU/alan/research/semantics 0.908s`. Extracting `QUALIFICATION ` and `COMPOSITION ` JSON payloads reproduced the 15 Phase 1 and 20 Phase 1B saved records byte-for-byte. A separate temporary extraction of `git archive 82dc5649cfe0eccb2fde3ecea1e12cc851377af5` independently reproduced the same records.

The earlier evidence files stay unchanged. Phase 1B's manifest hashes every current test source, so adding Phase 1C changes that manifest and SourceID even if all earlier behavior stays identical. Exact historical reproduction uses the pinned baseline; current-source behavioral comparison removes only the manifest record and each record's SourceID before comparison. The new source's identity must always be reported truthfully.

An isolated historical reproduction can be prepared without changing the active branch:

```sh
baseline_dir=$(mktemp -d /tmp/alan-phase1c-baseline.XXXXXX)
git archive 82dc5649cfe0eccb2fde3ecea1e12cc851377af5 | tar -x -C "$baseline_dir"
(
  cd "$baseline_dir"
  GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off \
    go test ./research/semantics -run '^(TestQualification|TestCompositionQualification)$' -count=1 -v > evidence.log
  sed -n 's/^.*QUALIFICATION //p' evidence.log > phase1.actual.jsonl
  sed -n 's/^.*COMPOSITION //p' evidence.log > phase1b.actual.jsonl
  cmp phase1.actual.jsonl docs/foundational/phase1/qualification.jsonl
  cmp phase1b.actual.jsonl docs/foundational/phase1b/qualification.jsonl
)
```

## Scope and interpretation

The [frozen plan](00-plan.md) and [public supervision contract](01-supervision-contract.md) distinguish ordinary recovery policy from lifecycle enforcement. All executable code remains an isolated test-only Go model. This package makes no production, durability, distributed fencing, ownership transfer, arbitrary-depth equivalence, fairness, or eventual-recovery claim.

## Artifacts and exact source

- [Operational contract](01-supervision-contract.md) and [frozen plan](00-plan.md).
- [Executable model and commands](../../../research/semantics/README.md).
- [Machine-readable evidence](qualification.jsonl): complete fixtures/programs, source/configuration/scenario identities, actual setup traces, event alphabets, required witnesses, seed monitor state, environmental stimuli, statistics, and counterexamples.
- [Source manifest](source-manifest.json): `go.mod` plus all thirteen model/test files, fourteen files total. Aggregate SHA-256: `08a107be1b4eff35986ab605dd1ef77db660baa6fb8e920f2d88a8f3031b859b`.

The manifest hashes sorted repository paths, each preceded by its unsigned 64-bit big-endian UTF-8 byte length, then raw file bytes preceded by their unsigned 64-bit big-endian length. It also records each file's SHA-256 and byte count. Evidence/docs are excluded to avoid circular identity; they are not model input. Configuration/scenario IDs hash their canonical Go JSON serialization. Every new record carries the actual source identity. The interpreter version for new immutable programs is `v1c`.

The 32 records comprise one manifest, eighteen positive searches, five guard-removal searches, one faulty ordinary-program search, five finite protocol traces, and a separate continuation of that faulty program after the oracle has reported its violation, and the retained review counterexample as an oracle-regression record. No earlier artifact or regression input was removed.

## Search method and bounds

Each seed begins at `newMachine(config)` and executes actual `step` and `inspect` calls. The resulting monitor is retained and emitted. Breadth-first suffix exploration checks every attempted successor before deduplication, including rejected actions. Full machine/configuration, journal histories, proposals, authority and protocol ledgers, world state, and monitor facts enter the SHA-256 exploration identity. No lossy abstraction or partial-order reduction is used. Required witnesses are replayed again, including fixed-version reconstruction of every run, with a complete-machine immutability assertion around replay.

The finite profile permits three static subjects, three generations and epochs, 48 commits/run, two continuations/proposals, eight intents/run, two external attempts/intent, sixteen inbox entries/run, values 0–8, eight local scalar fields, two names, one pending request, and eight consumed logical keys. The largest selected program has 21 instructions, below the frozen 32-instruction cap. Authorization/result retention is bounded by committed intents; notice variants are finite declared alphabet identities. No bound increased.

All suffixes use depth 10 and cap 5,000; fresh initialization uses depth 8. Setup is separate from suffix depth and reaches 109 events. A hard-cap hit is an error, covered by a focused test. None occurred. A finite alphabet reaching a fixed point is not exhaustive over omitted events/programs.

**States** counts unique nonviolating states including the seed. **Accepted** counts accepted attempted edges, including stutters and a violating successor. **Rejected** counts unavailable/invalid/capacity-limited attempts, not bugs. **Revisits** includes nonviolating rejected attempts merged with an existing machine/monitor state, as in Phase 1B. **Boundary** counts unexpanded states at the shortest-depth limit. The violating successor is not counted as an admitted state. Witness-only facts can change after a rejected attempt, so they are included in state identity. Recorded counterexamples are shortest suffixes for their disclosed seeds, not globally smallest ALAN failures.

## Positive searches

No safety monitor reported a violation in the eighteen positive searches. They sum to **1,576 scenario states and 4,973 accepted edges**, with 3,966 rejected attempts, 7,381 revisits, and 221 unexpanded depth-boundary states. Counts are summed across separate graphs, not globally unique states. The largest graph has 501 states. The 428 `capacity:epochs` rejections are reported explicitly; other positive rejection reasons and per-case counts are in the JSON.

| Fixture | Seed | States | Accepted | Rejected | Revisits | Boundary |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| fresh_ordinary_initialization | 0 | 27 | 53 | 67 | 94 | 3 |
| A_duplicate_failure_current_generation | 34 | 19 | 44 | 70 | 96 | 0 |
| B_accepted_lost_ack_and_successor | 25 | 97 | 322 | 260 | 486 | 0 |
| B_rejected_then_failed_retransmission | 14 | 23 | 67 | 48 | 93 | 0 |
| C_committed_intent_interrupted | 23 | 180 | 611 | 429 | 861 | 50 |
| D_CancelRun_retry_cancelled_false | 13 | 46 | 111 | 119 | 185 | 0 |
| D_RetireSubject_retry_cancelled_false | 13 | 44 | 99 | 121 | 177 | 0 |
| D_CancelRun_retry_cancelled_true | 13 | 60 | 147 | 153 | 241 | 0 |
| D_RetireSubject_retry_cancelled_true | 13 | 44 | 99 | 121 | 177 | 0 |
| E_suspicion_interrupt_reactivation | 11 | 298 | 681 | 999 | 1383 | 18 |
| F_budget_one_exhaustion_capacity_three | 37 | 116 | 116 | 260 | 261 | 22 |
| G_old_lifecycle_result_eligible_current_requester | 30 | 13 | 32 | 33 | 53 | 0 |
| H_nested_boundary_lost_ack_interrupt | 68 | 501 | 2324 | 1042 | 2866 | 127 |
| I_observation_without_lifecycle_authority | 23 | 19 | 61 | 47 | 90 | 1 |
| J_unknown_effect_recovery_late_settlement | 38 | 34 | 74 | 96 | 137 | 0 |
| D_policy_failure_CancelRun | 24 | 21 | 54 | 30 | 64 | 0 |
| D_policy_failure_RetireSubject | 24 | 21 | 46 | 38 | 64 | 0 |
| H_nested_late_duplicate_current_outcome | 109 | 13 | 32 | 33 | 53 | 0 |

## Evidence-to-claim mapping

| Class | Evidence and supported claim | Limit |
| --- | --- | --- |
| Searched safety A/G/H | Distinct transports for one frozen failure; old child and lifecycle results reach eligible current receives; correct current outcomes can then be consumed | Explicit finite windows and correlation convention, not all possible observer protocols |
| Searched safety B | Accepted Retry acknowledgment can be withheld; retransmission returns the original result. An originally rejected Retry stays rejected after the worker later fails | Whole decision ledger retention is assumed; no storage crash implementation |
| Searched safety C/E | Pending intent/count survive physical reactivation; proposal Interrupt discards uncommitted work; suspicion is consumed as classification 5 and ordinary code waits | Environmental activation replacement and suspicion are injected; no perfect failure detector or autonomous platform recovery |
| Searched safety D | Real policy requests race CancelRun/RetireSubject in both orders. Separate ordinary request probes examine active/cancelled outcomes under both RetryCancelled settings | Raw protocol probes have no failure-driven policy contract; they test lifecycle eligibility, not a demonstrated supervisory policy |
| Searched safety F/I | History-derived count respects the logical obligation limit; missing target grant yields retained rejection and ordinary escalation | Static grants, no delegation/transfer; root-wide budgets across fresh inner obligations are not modeled |
| Searched safety J | Old external application remains in the separate world; late receipt settles only its original intent while successor work runs | No safe business retry, compensation, or cross-generation endpoint deduplication |
| Searched reachability | Required combined traces include stale rejection then current consumption, lost acknowledgment/retransmission plus successor Commit, interruption/reactivation with pending intent, and same-generation work after suspicion | Existence of these traces is not universal progress or liveness |
| Finite protocol reachability | Worker success; abandonment recovery; cancelled-worker escalation; budget exhaustion; outer→inner→worker recovery/failure propagation | Some witnesses end at a recovery protocol boundary with active runs; no global termination claim |
| Falsification | Five broken protections and one faulty policy are detected by unchanged independent monitors | Deliberate guard removal is a model experiment, not a distributed failure implementation |
| Assumptions | Atomic local commit, atomic target decision/result, trusted runtime provenance, static rights, immutable versions, complete finite retention, external endpoint profile | Naming these assumptions does not implement durability, consensus, fencing, or authority transfer |
| Excluded/untested | General recursive equivalence, dynamic creation, cleanup custodians, ownership transfer, capability delegation/revocation, production transport/persistence/provider work, arbitrary external failures and unbounded fairness/liveness | None is inferred from search counts, replay, or reviewer approval |

The runtime has no role-name or ancestry branches. The fixed ordinary program branches on local classification and count, commits a lifecycle Emit, consumes its result, copies the returned successor generation, and loops or terminates. Generic name substitution preserves the full machine in a focused test. Only the checker reads `Supervision.Policies`; runtime target enforcement reads the separate grant list. Observing a child is insufficient authority to retry it.

The terminal-result and lifecycle ledgers are independent of program state and external resource state. Source dispatch authorization and target rights are separate. Decision-time generation/revision checks reject stale operations even when source authorization succeeded. Accepted/rejected operation IDs retain their first committed result. Replay reconstructs the ordinary projection/intent history without changing those ledgers, child generations, or the world; it is not full distributed recovery.

Environmental direct Reactivate, Abandon, CancelRun, RetireSubject, Retry and Suspect events are identified in each search record's `EnvironmentalStimuli`. The initial work and ordinary supervisor Retry requests use Prepare/Commit/Emit. External cancellation/abandonment and physical recovery stimuli are never counted as demonstrated supervisor policy. Runtime decision handlers reusing baseline lifecycle transitions are enforcement of a program's committed request, not environment-injected policy.

## Negative controls and retained counterexamples

All five mutants leave their independent monitors unchanged. The faulty ordinary program changes one policy branch and leaves runtime guards intact. Full event arguments and complete seed programs are retained in JSON.

| Control | Seed | States | Accepted | Rejected | Revisits | Detection depth | Invariant |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| broken_budget_persistence | 26 | 3 | 2 | 2 | 1 | 2 | `supervision-budget-persistence` |
| broken_operation-dedup | 14 | 10 | 17 | 11 | 18 | 3 | `supervision-operation-immutable` |
| broken_failure-correlation | 34 | 7 | 11 | 17 | 21 | 3 | `supervision-failure-correlation` |
| broken_lifecycle-authority | 23 | 3 | 3 | 5 | 5 | 2 | `supervision-lifecycle-authority` |
| broken_old-lifecycle-result | 30 | 9 | 17 | 13 | 21 | 3 | `supervision-old-lifecycle-result` |
| faulty_ordinary_budget_branch | 37 | 94 | 95 | 200 | 201 | 10 | `supervision-retry-budget` |

- **Operation deduplication:** after a real rejected-while-active Retry seed, commit worker Fault and retransmit the same operation. Without retained-result deduplication, the old rejection changes into acceptance. Merely repeating an accepted Retry would be masked by the separate expected-generation guard; this test avoids that vacuity.
- **Failure correlation:** deliver a second transport for the old worker failure into the eligible current wait, then Prepare/Commit its consumption. Removing subject/generation/logical-key validation is detected before it can become an independent recovery decision.
- **Budget persistence:** Prepare/Commit a lifecycle result whose faulty reducer resets spent count. The oracle compares the count with committed request history for the obligation.
- **Lifecycle authority:** source authorization still succeeds, but removing the target grant check allows DecideLifecycle to change a child without a grant. The independent target-rights monitor detects it.
- **Old lifecycle result:** admit an old requester generation's result by service routing while the new requester has its own pending request; Prepare/Commit consumption. The mutant removes the full operation/obligation/generation correlation block; the unchanged monitor rejects it. This is a correlation-family control, not a claim that one generation comparison alone is the only protection.
- **Faulty ordinary budget branch:** after one valid recovery and a second worker failure, a program skips its limit check and commits another Retry intent. The monitor detects the second decision at suffix depth 10, while generation 3 is still within capacity. A separate 49-event trace continues this already-invalid program through ordinary source authorization and runtime decision to create generation 3. Every edge is still inspected and the violation is retained; the trace is explicitly unsafe. This proves the oracle is not a hidden runtime budget enforcer.

The initial behavioral RED was `TestSupervisionTerminalBecomesOrdinaryObservation`: after committed Fault and DeliverTerminal, the baseline had no ordinary observer inbox entry. The retained input failed with `frozen child failure did not become an ordinary observation` before the additive representation was implemented. No failing input was removed.

## Finite protocol witnesses and nested recovery limit

| Witness | Events | Lifecycle decisions | Final boundary |
| --- | ---: | ---: | --- |
| complete_worker_success | 19 | 0 | Worker and supervisor succeeded |
| complete_authorized_abandonment_recovery | 39 | 1 | Worker g1 abandoned by environment; ordinary supervisor accepted retry and waits; g2 active |
| complete_cancelled_worker_escalation | 26 | 0 | Worker cancelled by environment; supervisor classified and failed without retry |
| complete_budget_two_exhaustion | 71 | 2 | Three committed worker failures; supervisor failed after two recovery decisions |
| complete_nested_exhaustion_outer_recovery | 79 | 2 | Inner exhausted after worker g2 failure; outer consumed inner failure, retried inner, consumed acceptance; outer and inner g2 remain active |

Each record contains full configuration, trace, and final machine. Every run is replayed with no dispatch. These records complete the named protocol demonstration, not necessarily all computations. The nested search separately commits actual instructions in inner generation 2.

A fresh inner supervisor explicitly initializes a new local obligation. It starts with child generation 1 under this fixture's initialization program; it does not discover or read the private state of worker generation 2. In the 109-event nested-old-result seed, it consumes the historical child outcome it explicitly named, requests Retry with that stale expected generation, receives a committed rejection, and fails. The outer then distinguishes old inner/g1 failure from this new inner/g2 failure. This is honest rejection/escalation, not evidence of automatic current-child handoff or successful recursive repair. Such handoff and a hierarchy-wide business budget need an explicit ordinary protocol beyond this increment.

## External ambiguity remains visible

J's immutable Tag explicitly names `business:order-7`. The worker first applies that operation without a recorded receipt, fails, and is retried by ordinary policy. A successor instruction can emit the same explicit business reference under a new execution intent ID. The `ambiguous-repeat-business-mutation` witness contains two mutations. The missing receipt never becomes proof of non-execution, the old operation is not erased, and the business reference is not silently changed to a new generation identity.

This is a retained counterexample to the stronger claim that recovering computation makes business repetition safe. The qualified supervisor policy is a computation-recovery policy. Endpoint-specific reconciliation or explicit permission to repeat business work remains necessary; no such protocol is asserted. A world-different/runtime-equal test also confirms that ordinary reduction cannot inspect hidden external application state.

## Verification and independent review

Primary verification on the reviewed source used:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off go test ./research/semantics -run '^TestSupervision' -count=1
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off go test -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off go test -race -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off go vet ./...
gofmt -l research/semantics/*_test.go
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1c-go-cache GOPROXY=off GOSUMDB=off \
  go test ./research/semantics -run '^(TestQualification|TestCompositionQualification|TestSupervisionQualification)$' -count=1 -v
git diff --check
```

Focused supervision tests passed in 3.944s and the full suite in 4.462s; primary race passed in 81.657s; the delegated implementer independently passed race in 82.777s on the same final source; vet emitted no diagnostics; gofmt emitted no paths; diff checking passed. Combined qualification passed. An independent Python implementation verified all fourteen per-file hashes/lengths, the aggregate source hash, and every new SourceID. All fifteen Phase 1 records remain byte-identical on current source. All nineteen Phase 1B behavioral records remain identical after removing only SourceID; its twentieth record is the necessarily changed manifest. Exact all-twenty historical reproduction passed from the pinned archive, as described above.

### Independent adversarial review findings

A separate reviewer, who did not author the implementation, reviewed the complete source, contracts and evidence, independently verified the source manifest, and replayed all thirty non-manifest records, 88 saved witness suffixes, six negative-control traces, and stored final machines for source `0f507d1dc22c6faf403de916e0ccbcb4ca5cff1fb8680140ed863fe760caee1b`. It also checked ordinary non-Retry lifecycle requests and that changing test-only policy metadata cannot change runtime transitions. All earlier phase documents/evidence were independently compared against the pinned archive and remained byte-identical.

The initial verdict was **not approved pending correction** for two medium findings:

| Finding | Evidence | Disposition |
| --- | --- | --- |
| R1: nested interruption coverage missing | H's original 68-event seed/eight-event alphabet had Reactivate but no Interrupt; it searched 319 states and 1,446 accepted edges | Add actual Interrupt and require the corresponding same-execution ordered witness; no bound change |
| R2: cross-execution interruption witness | A global interrupted flag could combine inner/g2 interruption with outer/g1 reactivation and claim pending outer recovery | Retain exact false-positive input; key interruption/recovery facts to the same execution and pending operation, with temporal order |

The exact R2 suffix after the unchanged H seed is: authorize `outer/g1/n10/i0` → decide that operation → Reactivate outer/g1 → Prepare inner/g2 continuation 1 in proposal slot 0 → Interrupt slot 0. The old checker reported `committed-intent-interrupted-reactivated` for outer although its proposal had not been interrupted. This is an evidence-oracle counterexample, not a runtime safety violation. A private expanded-H probe added only Interrupt and reached 505 states/2,401 accepted edges before the witness correction; those preliminary counts are not the final qualification counts.

The exact input first failed `TestSupervisionInterruptionWitnessCorrelation` with `oracle falsely joined outer/g1 reactivation with later inner/g2 interruption`. The correction records the interrupted proposal's execution and exact pending operation; an ordered reactivation fact requires the same pending identity on that same run. Separate tests reject reactivation-before-interruption and accept interruption-before-reactivation. H retains its original configuration/68-event seed/all prior events, appends Interrupt, and requires the ordered combined witness. The regression fixture and seed hashes are pinned, and `oracle_regression_interruption_execution_and_order` retains complete config, seed, exact five-event suffix, old reviewed identity, corrected witness output, and no runtime safety violation.

Final corrected source is `08a107be1b4eff35986ab605dd1ef77db660baa6fb8e920f2d88a8f3031b859b`. Only supervision witness/search and test code changed in this correction; interpreter and safety-monitor files are byte-identical to the initially reviewed source. No input was removed, no environmental assumption strengthened, and no bound increased. Counts changed because the searched H alphabet expanded and witness-history facts became correctly qualified. H now explores 501 states/2,324 accepted edges with 127 depth-boundary states. A dedicated new-explorer hard-cap regression was also added to substantiate the cap-failure statement.

Independent scoped re-review marked **R1 ADDRESSED** and **R2 ADDRESSED**. It independently verified all fourteen final source-file identities, exact preservation of the original regression configuration/68-event seed/five-event suffix, and preservation of all earlier fixture configurations/seeds/alphabets (H only adds Interrupt). All 32 final evidence records reproduced byte-for-byte; focused regressions and qualification passed in 3.657s. The reviewer checked the final report and decisions against the evidence. Final verdict: **specification PASS, quality PASS, APPROVE for Phase 1C bounded research-deliverable readiness**, with no blocking findings remaining. The initial withheld approval, coverage gap, oracle defect, and correction remain part of this record. Approval concerns the exact bounded package, not unbounded correctness.
