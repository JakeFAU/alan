# Phase 1C ordinary supervision and recovery policy plan

> Execution uses one delegated implementer for coupled model changes and an independent adversarial reviewer. This is bounded research, not a runtime specification approved for production.

**Goal:** try to falsify the claim that ordinary programs can observe committed child outcomes, make bounded recovery decisions, request authorized lifecycle changes, consume their results, and propagate failure through nested supervision.

**Architecture:** retain the existing finite interpreter, speculative evaluation, atomic commit, external world, and lifecycle rules. Represent lifecycle requests as ordinary immutable intents and lifecycle notifications/results as authenticated observations. Keep policy branches, retry counts, and obligation state in the ordinary program projection. Extend the existing seeded breadth-first methodology.

**Toolchain:** installed Go 1.27.3; standard library; all executable source remains in `research/semantics/*_test.go`.

**Specification:** [01-supervision-contract.md](01-supervision-contract.md), the retained [Phase 1 rules](../phase1/01-operational-spec.md), and [Phase 1B contract](../phase1b/01-composition-contract.md). The user's Phase 1C requirements govern conflicts.

## Baseline and preservation

The clean source baseline is commit `82dc5649cfe0eccb2fde3ecea1e12cc851377af5`, with model/test SHA-256 `afc89e4bf24126b33da5341f1bc4c582c81b0b80f51a04af1674daacf0b3c49a`. Before model changes, the full Go suite passed, and a fresh combined qualification run reproduced all 15 Phase 1 and all 20 Phase 1B records byte-for-byte. The source digest was independently recomputed from all ten manifest files. Commands and output are retained in the qualification report.

All earlier regression inputs, policy counterexamples, guard controls, documents, and evidence remain. Phase 1B dynamically hashes every test source file, so its emitted source manifest and SourceID necessarily change when Phase 1C is added. Exact historical reproduction uses the pinned baseline tree. Current-source regression reproduction compares every old behavioral record, removing only SourceID and the source-manifest record; it never substitutes the old identity for new code. Both checks must be reported distinctly.

Work uses `codex/phase1c-ordinary-supervision` in the existing clean checkout. The draft PR targets the existing `codex/phase1b-recursive-composition` research branch, verified at the baseline commit. Phase 1B's earlier PR was closed when its own base branch was deleted; this increment does not merge or silently reopen that PR.

## Selected approach and alternatives

1. **Selected:** add narrow ordinary effect/observation representations around existing lifecycle operations, plus finite local data access needed to express policy. Programs choose actions using Receive, Set/finite arithmetic, Branch, Emit, and terminal instructions. The runtime validates authority and operation identity; it never chooses a recovery action.
2. **Retained alternative:** an ordinary lifecycle-service subject reached via Send/Receive. That still needs an authority boundary and retained operation outcomes. Adding a service execution here would complicate the smallest experiment without removing that obligation.
3. **Rejected for this phase:** one RetryPolicy/Supervise instruction, privileged supervisor types, direct program mutation of another run, or a runtime-selected retry decision. These would assume the hypothesis being tested.

The baseline cannot route its frozen terminal outcome into an ordinary inbox, or express a lifecycle request as an intent. Those are missing protocol representations, not evidence that a special supervisor computation type is necessary. Any further semantic addition must be classified in the contract before qualification.

## Frozen initial bounds

- At most three statically provisioned subjects: outer, inner, worker. Single-supervisor fixtures use two. Names are fixture data, never interpreter branches.
- New fixtures: at most three generations and three activation epochs per subject; 48 commits per run; two continuations; two proposal slots; eight intents per run; two attempts per intent; sixteen inbox entries per run; integer data 0–8.
- At most 32 instructions per immutable program; two ordinary obligation/correlation slots and eight consumed logical-notification keys per run; retry limits 0–2. Identity domains are finite fixture strings plus bounded subject/generation/commit/slot tuples. Retained operation outcomes are bounded by committed intents plus at most four declared environmental requests. Transport variants are declared explicitly in each finite alphabet.
- The additive ordinary local representation has eight bounded integer fields, two names, one pending operation reference, and last-observation metadata. Legacy data/reference slots are unchanged. This concretizes the initial ordinary-state allowance; it is not a bound increase.
- Focused suffix search depth 10, hard cap 5,000 states per scenario. A small fresh-start search uses depth 8 and the same cap. Complete witness traces are finite, separately reported, and do not establish exhaustive safety.
- Setup prefixes execute real transitions and all monitors. Every attempted successor is checked before deduplication. Machine, configuration, histories, ledgers, external world, proposals, and future-relevant monitor facts enter state identity.
- No bound increase is implicit. Record amendments, reason, and coverage cost before generating final evidence. Cap exhaustion is an error. Report all finite-capacity rejections and depth cutoffs.

## Global constraints

- No production runtime or SDK, network, provider, persistence, wall clock, arbitrary callback, external workflow system, or new dependency.
- No dynamic creation, ownership transfer, general capability delegation, distributed consensus, group cancellation, cleanup custodians, arbitrary-depth equivalence, or unbounded fairness/liveness guarantee.
- Logical subject, execution generation, activation epoch, ownership revision, supervisory obligation, and lifecycle operation identity remain distinct.
- Intent commitment, lifecycle authorization, transition decision, outcome admission, and ordinary outcome consumption are distinct events.
- Application failure, authorized abandonment, run cancellation, retirement, success, and physical interruption/suspicion remain distinguishable.
- Communication permission never grants lifecycle authority. Authority is static fixture data, not inferred from topology or names.
- Atomic authority/transition decisions remain assumptions. No new durability, distributed fencing, or transfer claim.
- Monitors may reject a contract violation but cannot select program actions or repair state.

## Task 1: Freeze baseline and contract — primary agent

- [x] Inspect the foundation, model, reports, and review findings.
- [x] Reproduce both earlier qualification packages and the full baseline suite.
- [x] Identify missing ordinary representations and publish this plan and contract before model edits.
- [x] Obtain read-only design advice and record any amendments before implementation. The advice requires retained semantic rejections, decision-time target validation, service-routed stale outcomes, and a budget-1 fixture with capacity for generation 3. A deduplication control must retransmit an originally rejected retry after its original rejection condition changes; an accepted-retry repeat alone can be masked by the expected-generation check.

## Task 2: Implement ordinary supervision qualification — delegated implementer

**Files:** minimally extend `research/semantics/model_test.go` and `checker_test.go`; add `supervision_model_test.go`, `supervision_checks_test.go`, `supervision_search_test.go`, and `supervision_contracts_test.go`. Further responsibility-based splits require a recorded reason. Do not alter old fixture definitions, expected counts, counterexample inputs, or saved evidence. Primary agent owns documentation.

**Interfaces:** retain `step(machine,event)`, `reduce(config,*run,selection)`, `inspect(machine,machine,event,monitor)`, `replay(machine,id,version)` and the composition explorer pattern. Add `TestSupervisionQualification`, emitting deterministic `SUPERVISION ` JSON records including the current complete source manifest. Concrete optional field names may be selected by the implementer; report them for the operational appendix before final qualification.

- [x] Write observable contract tests first and record meaningful behavioral RED; retain original failing inputs.
- [x] Add authenticated terminal/suspicion and lifecycle-result observations without allowing a payload to appoint its own provenance.
- [x] Add ordinary immutable lifecycle intents, static authority grants, separate authorize/decision/result-admission transitions, and retained results for both accepted and rejected operation IDs. Use existing lifecycle rules; do not make program policy choices in runtime.
- [x] Express child identity/generation, obligation, classification, count/limit, pending request, result, and terminal choice in bounded ordinary state/instructions. Document every added capability's minimality classification. Demonstrate that branches choose recovery and budget handling.
- [x] Build ordinary worker success/failure/abandonment variants, a single-supervisor bounded retry policy, and an outer→inner→worker witness where inner recovery is exhausted and outer applies its own policy using only public observations.
- [x] Add searched windows for scenarios A–J in the contract, including both serialization orders, actual successor Prepare/Commit, late old outcomes while current work is eligible, and ambiguous external work across recovery.
- [x] Include accepted and rejected lifecycle request retransmission, supervisor interruption after intent commit, and replay without lifecycle or external dispatch. Environmental lifecycle stimuli must be identified, never counted as ordinary supervisor policy.
- [x] Add five targeted mutants (operation deduplication, failure correlation, retry-budget persistence, lifecycle authority, old-generation lifecycle-result isolation) detected by unchanged independent monitors. Add a faulty ordinary policy program under normal runtime guards.
- [x] Emit complete fixtures/programs, exact seed events, alphabets, bounds, states/accepted/rejected/revisits/cutoffs, rejection reasons, required witnesses, source/scenario identities, and shortest suffix counterexamples. Preserve every legitimate counterexample.
- [x] Run focused/full tests, race, vet, gofmt, and old behavioral reproduction. Return exact commands, results, source identity, semantic amendments, and concerns. Do not commit or spawn helpers; primary agent packages reviewed changes.

## Task 3: Evidence, adversarial review, and draft PR — primary agent/reviewer

- [x] Generate `qualification.jsonl` and `source-manifest.json`; independently verify file hashes and reproducibility.
- [x] Write `02-qualification.md` with an evidence-to-claim table separating searched safety, reachability, assumptions, excluded/untested claims, and unresolved outcomes.
- [x] Append findings to `../05-decisions.md` and update the research README.
- [x] Review complete model/test sources, full diff, contracts, exact source identity, and all fixture evidence independently; retain findings and correction traces.
- [x] Correct substantiated failures or narrow claims explicitly, then re-review. Approval is research readiness only.
- [x] Run final focused/full/race/vet/gofmt and exact archived Phase 1/1B reproduction; compare old behavioral results on new source.
- Delivery step after this reviewed package: commit the increment, push its own branch, open/attach a draft PR against the existing research baseline, and stop. No merge or runtime implementation.

## Preflight review focus

| Boundary | Failure to challenge | Required check |
| --- | --- | --- |
| Ordinary data / runtime | Policy hidden in an opcode or authority handler | Faulty program must remain expressible; oracle detects it |
| Intent / lifecycle ledger | Duplicate request executes again or changes prior rejection | Same identity retains immutable original outcome |
| Authorization / transition | Authority or generation checked only before a racing cancellation | Revalidate/serialize at the transition decision |
| Observation / obligation | Duplicates or stale generations create new decisions | Correlation and independent decision-budget monitors |
| Retry / budget | Worker generation or supervisor epoch resets count | History-derived budget over one logical obligation |
| Program / world | Missing receipt interpreted as no mutation | Separate resource model and late-settlement checks |
| Seed / BFS | Setup bypasses monitors or state identity loses history | Replayed prefixes with carried monitor state |
| Evidence / baseline | Changed source mislabeled as historical source | Exact pinned-tree reproduction plus explicit current-source comparison |

## Recorded review amendment

Independent review of source `0f507d1dc22c6faf403de916e0ccbcb4ca5cff1fb8680140ed863fe760caee1b` found that H's nested boundary alphabet used Reactivate without Interrupt, and a global interruption witness flag could combine different executions. The correction adds the existing Interrupt event to H and qualifies witness history by the same run and pending operation, in interruption-before-reactivation order. The original fixture/seed and exact five-event false-positive suffix remain as a regression input. This adds coverage and corrects evidence interpretation; it neither strengthens an environmental assumption nor increases any bound. Final statistics must be regenerated after the oracle change.
