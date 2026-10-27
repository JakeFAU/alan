# Phase 1B recursive composition qualification plan

> Execution uses a delegated implementer and an independent adversarial reviewer. This is a bounded research increment only.

**Goal:** try to falsify the claim that an ordinary nested composition can satisfy the same declared interaction contract as a leaf computation.

**Architecture:** extend the existing finite interpreter with explicit message-envelope operands and bounded ordinary correlation/aggregation state. Use small multi-subject searches, including suffix searches starting from fully replayed setup traces. All subjects use the same interpreter. Compare leaf and composite against an allowed-behavior contract, without claiming contextual equivalence.

**Toolchain:** installed Go 1.27.3, standard library, test-only source files. No network, provider, database, clock, infrastructure, or arbitrary agent callbacks.

**Specification:** [01-composition-contract.md](01-composition-contract.md), together with the unchanged [Phase 1 rules](../phase1/01-operational-spec.md).

## Baseline and approach

The clean source baseline is `f81dd200d05f7c9de62d25fa9baf6c9f6dd08bf9`. Before any model changes, `go test -count=1 ./...` passed and a fresh `TestQualification` run exactly matched all 15 records in `phase1/qualification.jsonl`. That evidence file, existing fixtures, negative controls, policy counterexamples, and regression inputs will be retained.

Three approaches were considered: exploring every event from an empty multi-subject system, replaying selected setup traces followed by bounded exhaustive suffix exploration, and adding more successful scripted compositions. The second is selected to reach the dangerous overlap windows without making state count the objective; a small fresh-start sender/receiver search and complete recursive witnesses complement it. Setup is part of the evidence, not an assumed reachable state. The third alone would repeat Phase 1's coverage gap. The first remains a broader future experiment.

The current Receive checks ordinary message kind and tag but does not retain semantic request correlation or sender-generation information. Extending those explicit data contracts is necessary to test the hypothesis. This is not an orchestration type or a durability claim. Lifecycle events remain environment-injected authority actions.

## Frozen bounds and exclusions

- At most four static subjects: outer coordinator, replaceable provider, nested worker, independent peer. A leaf provider uses the same public boundary without the nested worker.
- Phase 1 fixtures retain their exact bounds. New fixtures allow at most two generations and epochs per subject, twelve commits per run, two continuations, two proposal slots, three intents per run, two attempts per intent, eight inbox entries per run, and values 0–8.
- At most two saved upstream envelopes, two pending downstream request references, and four accepted logical result keys per run. Programs have at most twelve instructions. Generic envelope fields range over finite fixture identities and retain their provenance through replay.
- Initial suffix depth limit: 10; hard state cap: 5,000 per scenario. Fresh-start sender/receiver depth: 8, cap: 5,000. Complete reachability traces are separately reported and do not borrow the safety-search claim.
- Any increased bound, changed alphabet, or semantic amendment must be recorded with its reason and coverage cost before publishing results. A hard-cap hit is an error, never a positive qualification result.
- No dynamic creation, ownership transfer, cleanup custodian, dispatcher-incarnation fencing, distributed atomicity, persistence, provider integration, fairness theorem, or general equivalence proof.

## Task 1: Freeze contract and source baseline — primary agent

- [x] Inspect code, existing operational rules, composition discussion, decisions, and qualification results.
- [x] Reproduce all Phase 1 evidence and tests before changes.
- [x] Define the public contract, failure cases, bounds, and evidence categories before implementing.
- [x] Obtain read-only scope advice on correlation, seeded monitor history, and nonvacuity.
- [x] Publish this plan before model changes.

## Task 2: Implement composition qualification — delegated implementer

**Files:** minimally extend `research/semantics/model_test.go` and `checker_test.go`; keep existing baseline fixture/test definitions intact. Add `composition_model_test.go` for ordinary envelope helpers, `composition_checks_test.go` for independent contract monitors, `composition_search_test.go` for explicit seeded exploration and fixtures, and `composition_contracts_test.go` for qualification/regressions/witnesses. Extend `research/semantics/README.md` with Phase 1B commands. Split a new file further by responsibility only after recording the change.

**Interfaces:** reuse `step(machine,event)`, `reduce(config,*run,selection)`, `inspect(machine,machine,event,monitor)`, and `replay(machine,id,version)`. New qualification entry point: `TestCompositionQualification`, emitting deterministic `COMPOSITION ` JSON records. New seeded exploration must execute every seed event through both transitions and monitors, carry the resulting monitor facts, check every successor before deduplication, and include every future-relevant field in identity. The implementer may choose concrete field names for the finite envelope operands, but must document them for the operational appendix before qualification.

- [x] Write contract tests first; record behavioral RED where the baseline admits stale, duplicate, or wrong-origin logical results. Preserve the offending inputs through the correction.
- [x] Add generic bounded capture/forward/correlation/value operands and ordinary accepted-result state; no role-name conditionals in runtime transitions or reducer.
- [x] Add a fresh sender/receiver search and focused seeded searches for the six challenge windows in the contract. Wherever retry is in a scenario, include actual successor Prepare and Commit events and require reached successor computation.
- [x] Include a combined sender/receiver suffix in which receiver closure/retry, retransmission, interruption, fresh work, and late acknowledgment can interleave. Run result-closure, old-result/generation-2, and logical-duplicate searches against the nested topology as well as the leaf topology.
- [x] Require combined witnesses, not disconnected flags: old result presented while a new-generation receive is eligible, correct current result accepted, duplicate distinct-transport result blocked while another branch contributes, and external application without receipt followed by closure/retry.
- [x] Build the leaf and nested implementations against the same public monitor; include outer → nested → worker depth and independently progressing peer work. Label completed traces as reachability evidence.
- [x] Add generation-correlation, logical-duplicate, and observation-origin negative controls, using unchanged monitors and corresponding positive fixtures. Require actual wrong consumption/contribution to be detected.
- [x] Emit complete fixture/config identities, seed traces, alphabets, bounds, counts, rejection reasons, witnesses, and shortest mutant traces. Add exact SHA-256 identity of all model/test sources and `go.mod`, with an explicit algorithm and file list; exclude generated evidence from its own digest.
- [x] Verify all old qualification records remain identical; run focused tests, then all tests/race/vet/gofmt. Return exact commands/results and semantic amendments. Do not commit; primary agent packages only reviewed changes.

## Task 3: Qualify and independently challenge — primary agent and reviewer

**Files:** create `02-qualification.md`, `qualification.jsonl`, and a source manifest in this directory; append a Phase 1B entry to `../05-decisions.md`. Preserve Phase 1 documents/evidence unchanged.

- [x] Independently reproduce the final evidence using the recorded source identity.
- [x] Write an evidence-to-claim table separating searched safety, reachability, assumptions, and excluded claims; include unresolved outcomes and all review counterexamples.
- [x] Supply the reviewer the complete model/test sources, full diff, plan/contract, fixture evidence, and exact source identity.
- [x] Obtain independent adversarial APPROVE or retain unresolved rejection findings without claiming readiness; fix substantiated failures and re-review.
- [x] Commit the research increment, open a separate draft PR against the Phase 1 branch under the standing finishing preference, and stop. No runtime implementation or automatic merge.

## Preflight and review focus

| Shared boundary | Conflict to prevent | Resolution |
| --- | --- | --- |
| Contract / interpreter | Hidden correlation supplied by test helper | Preserve typed envelope data in ordinary committed state and replay |
| Interpreter / monitor | Guard reused as oracle | Derive consumed-result obligations from retained descriptors/history independently |
| Setup / search | Fresh monitor loses seed facts | Run the complete seed through the same monitor and carry it into BFS |
| Retry / receive | New generation exists but cannot run | Include and require successor Prepare/Commit and eligible current-result continuation |
| Dedup / aggregation | Same transport duplicate mistakes itself for logical dedup | Two different transport identities with the same logical result key |
| Leaf / composite | Equal scalar hides effect or failure differences | Same declared contract projection, including resource effects and terminal outcomes |
| Model / report | Truncated graph called proof | Publish alphabets, seed, cutoffs, cap errors, and explicit claim categories |
| Task 1 / Task 2 | Implementation chooses undocumented semantics | Amend contract before qualification; retain alternatives and costs |
| Task 2 / Task 3 | Evidence points to later edited source | Hash all model/test source bytes; regenerate and re-review after changes |

One implementer owns the coupled interpreter/monitor/explorer changes. Documentation and independent review have separate owners. Work uses a new research branch in the existing clean checkout; prior Phase 1 commits and evidence remain available unchanged.
