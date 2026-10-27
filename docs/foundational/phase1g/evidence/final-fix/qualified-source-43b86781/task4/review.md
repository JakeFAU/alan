# Task 4 independent review

## Spec Compliance

- ✅ Spec compliant for the task-scoped qualification gate. Reviewed range `49ae5f896d88f080a2e10cbfef1b537d27f55df5..6c78f45f62fd12c1ccff6314b3e73d6a80fb8d0a` against the Task 4 brief and normative R1–R8 contract. Changes provide process schedules, independent checks, controlled-provider evidence and reproducible source packaging. Existing Go edits are the four named RunID fields (`author/invocations_test.go:43`) and disclosed watchdog literals (`examples/worktree/effects_integration_test.go:180`, `examples/worktree/effects_integration_test.go:220`); no production Go change.
- ⚠️ Whole-branch adversarial review, verified-main publication and attached draft PR remain controller follow-up gates (`docs/foundational/phase1g/evidence/task4/report.md:3`). No merge or release approval is implied.
- ⚠️ Earlier-task runtime semantics and controls are not independently re-reviewed here. Final-source execution is retained, including separate consumer faulty ordinary collector and historical comparisons, but this review assesses Task 4 harness/evidence, not all prior implementations (`docs/foundational/phase1g/tools/qualify.py:113`, `docs/foundational/phase1g/tools/compare.py:16`).

## Strengths

- Acceptance joins the exact request, observation, consumption, history and successor items. ACK validation binds PID/token/store/source/generation/sequence/epoch/intent; invocation-result ACKs separately join result, invocation and original source authorization without inventing an emitted epoch (`internal/compositiontest/process_test.go:89`, `internal/compositiontest/process_test.go:112`, `internal/compositiontest/process_test.go:122`, `internal/compositiontest/process_test.go:590`). Final normal records 446/505 reach collector sequence 5; 486/545 reach sequence 27 with three known child outcomes and two uncollected results.
- SIGKILL checks actual process status; a new process runs Open/Run without rebuild or input resubmission. Original admissions/results/charges persist, and accepted cases interrupt recovery again at recorded-result boundaries (`internal/compositiontest/process_test.go:209`, `internal/compositiontest/process_test.go:360`, `internal/compositiontest/process_test.go:386`). Final normal evidence contains 35 starts/exits, 13 SIGKILL exits, six cuts and two recovery interruptions (`docs/foundational/phase1g/evidence/task4/process-summary-qualified-normal.json:1`).
- Held-effect evidence is separate from deterministic partial consumption. It causally observes one active request and a collected sibling, waits for connection closure, and reopens with the original unknown work; five admissions, eight created runs and three effects remain. Provider correlation joins exact prepared keys, intents and questions; equal payloads have distinct operations/contacts (`internal/compositiontest/effects_test.go:88`, `internal/compositiontest/effects_test.go:219`, `internal/compositiontest/ownership_test.go:99`). Final normal records 429/436 retain exactly three contacts and one held contact.
- Operator closure obeys single-writer sequencing: kill/wait owner, open/cancel/close operator, then fresh Run. Retransmission and foreign-operation probes similarly wait for owner death and require unchanged inspections. The report correctly avoids claiming that the partial append cut reached the next item's Await (`internal/compositiontest/closure_test.go:67`, `internal/compositiontest/process_test.go:637`, `docs/foundational/phase1g/evidence/task4/report.md:43`).
- Literal item values and algebraic counts are independent expectations. The actual waiting-parent overlay reaches occupancy one with zero adapter attempts and fails the unchanged ownership monitor specifically for a non-external owner. Reports distinguish that runtime fault, the authentic faulty ordinary collector, persisted-corruption tests and monitor-input mutations for budgets/origins/premature success/replay (`internal/compositiontest/process_test.go:498`, `internal/compositiontest/process_test.go:670`, `internal/compositiontest/ownership_test.go:26`, `docs/foundational/phase1g/tools/slot_control.py:15`, `docs/foundational/phase1g/evidence/task4/report.md:49`). Original unrelated overlay setup failure is retained, not counted as reaching the intended fault.
- Independently recomputed all 257 source entries and aggregate `43b86781f819c6bf7a957956aceaf717a5808873114e8213bf1147620bdf2ec3`, archive SHA256 `423c1bff2aaf4e8a76a8813df6a1076977c2d8844db13e2b34919ea382732fb4`, and all 499 member hashes. Nested replacement paths and pinned proxy are included; reproduction verifies/extracts before commands with initially absent module cache and credential removed (`docs/foundational/phase1g/tools/package.py:6`, `docs/foundational/phase1g/tools/reproduce.py:12`, `docs/foundational/phase1g/tools/reproduce.py:20`).
- All five final Go streams match their metadata raw hashes, have actual exit zero and no failure events. Pass events are 764/1006/1006/20/20 for default/integration/race/consumer/consumer-race. Nine outer commands and eight auxiliaries exit zero; three comparisons preserve declared 15/20/32 payload and six conformance-projection scopes (`docs/foundational/phase1g/evidence/task4/extracted/reproduction.json:1`, `docs/foundational/phase1g/evidence/task4/extracted/checks.json:1`, `docs/foundational/phase1g/tools/compare.py:16`). Default stderr is only the expected SQLite download notice from the bundled file proxy; no warning or race diagnostic was found.
- Historical failure identities remain distinct: interrupted first race has unknown Go exit, e413/f006 races have actual exit one, and the outside-root metadata exception is retained. Recomputed all 1,120 baseline artifact hashes with no differences (`docs/foundational/phase1g/evidence/task4/first-source-stop.json:1`, `docs/foundational/phase1g/evidence/task4/race-failure-classification.json:1`, `docs/foundational/phase1g/evidence/task4/race-attempt4-failure-classification.json:1`, `docs/foundational/phase1g/evidence/task4/prior-artifacts-final-verification.json:1`).

## Issues

### Critical (Must Fix)

None found.

### Important (Should Fix)

None found.

### Minor (Nice to Have)

- M1 — `docs/foundational/phase1g/tools/qualify.py:100`: command metadata copies Platform and GoVersion from the frozen source manifest. A future fresh-workspace reproduction on another host would label its Go commands with the original freeze host/version. Capture the executing platform and `go version` for each fresh command; retain manifest values under explicitly named freeze fields if useful. This does not invalidate the retained same-host qualification. Auxiliary checks already capture the executing platform independently (`docs/foundational/phase1g/tools/checks.py:18`).

## Assessment

**Task quality: Approved**, with the non-blocking metadata portability issue above.

Causal bounded recovery evidence and runnable archive reproduction support the scoped claims. Reports keep known outcomes, unresolved work, evidence mutations and executed faults distinct.

**Checks performed:** Read supplied source/tool/docs diff in bounded passes, consulted complete package for evidence inventory and referenced records, verified source/archive/member/raw-log hashes and historical inventory, and inspected causal records plus positive/negative slot-overlay and empty-header output. No Go test, broad suite, live request, source edit, commit or git-state mutation was performed. The only written file is this authorized review report. No broader source crawl was needed; unchanged source bytes were read only for identity verification.
