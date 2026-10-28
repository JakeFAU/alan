# Phase 1H Task3 independent harness checkpoint review

Checkpoint spec compliance: **APPROVE**.
Task quality: **APPROVE**.

This approves proceeding from the focused harness checkpoint to its declared final qualification gates. It is not final Phase1H approval, merge approval, a proof of the general semantics, or a claim that the corrected source has passed all suites.

Reviewed range: `19065f30d4135c8710faed3db5c7311a01c2530e` → `4463e9aeaad0f76cce4df14af4e47419632a86d3`, in `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`. Controller's later evidence-only commit `edd6653` is outside this range. No checkout, index, HEAD, or repository files were changed by this review. Only review outputs under `/tmp` were written. No subagents or Go test/build/race runs were used.

## Question and result

Does this narrowly scoped harness provide independently inspectable evidence for real process cuts, truthful unknowns, root closure, physical capacity and an application-policy negative control without moving application policy into Go?

**Bounded, source-specific support established.** I found no actionable correctness or specification deviation within this checkpoint. The Go harness constructs fixtures and observes/interrupts actual execution; ordinary retained programs continue to own frontier selection. No production runtime, schema, authoring or adaptive application changes are in this range. Frozen expected-work fixtures are unchanged.

## Evidence checked directly

I read the checkpoint brief and report, contract, all six new Go harness files, all eleven tools, runbook, scoped manuscript additions and manuscript audit. I inspected the exact range and the two rebased source-archive differences. I independently decoded the retained raw logs, records and relevant snapshots, rather than accepting emitted PASS labels.

Source attribution:

- `development/rebased-focused`: source `5d8758491348bd793f90b7f818d66013c17e379f2ec0a54906ad217eb0a9400a`, exit 1, 399.154 seconds. The raw events show both capacity2/held0 subtests failed. The six recovery cuts, denial/unknown variants, primary closures, live companion, isolated replay and faulty application passed in that failed overall run.
- `development/rebased-capacity-fix1`: source `9f15cc582430e99a5bd7e192ef8f6d3f8a4b36fd2d9bbffa65728eff37241816`, exit 0, 89.939 seconds. Raw events show all six capacity cases passed.
- `development/rebased-public-consumer`: first rebased source above, exit 0, 16.984 seconds; two actual nested/direct paired capacity1 consumer cases passed.
- All ten development source archives match their original manifests member by member; all ten raw stdout/stderr hashes and archive hashes match capture metadata. All 166 retained artifact compressed hashes and encoded-value hashes verify. The frozen expected-work SHA256 is `4fc58131e849fc15fd45dee5426c087fb7db4f99cb9539667da7b075f9cbff97`.
- Only `internal/adaptivetest/cuts_test.go` and `docs/foundational/phase1h/tools/freeze.py` differ between rebased captures. The Go difference is within the capacity test. All 271 corrected inventory files match the current checkout bytes. There is no basis to relabel the earlier failed combined command as a corrected-source all-pass command, and the report does not do so.

Evidence directories below are under `docs/foundational/phase1h/evidence/task3/development/`.

1. **ACK and recovery.** `process_test.go:185` validates candidates through public Inspect before forwarding the real inherited barrier; `monitors_test.go:336` and `:346` bind process/filter and retained source identities. Direct snapshot checks of each `rebased-focused/artifacts/TestProcessAdaptiveCuts_*/before.json.gz` found the ACK's exact subject/generation/epoch/sequence history row, matching successor projection and original intent. Decision, first successor intent, and partial-result cuts occurred at adaptive sequences 486, 501, and 526 respectively. Work count is already six at the first successor intent. Both partial snapshots contain results `[1,2,3,7]`; item10 has no outcome while item11 already has one. Original histories, intents, root descriptors, invocation rows and charges survive into each after snapshot. Final ordered result IDs are `[1,2,3,7,10,11,31]`. The logs retain real SIGKILL exits and distinct fresh child starts. The separate pure leaf's 128 bounded iterations are disclosed and do not interrupt external attempts.

2. **Capacity and causality.** `cuts_test.go:195` now takes the provider ledger before Inspect, retains both before correlation assertions, and sends the fd5 trigger afterward. `process_test.go:207` admits the input through the one live Store; `fixtures_test.go:225` defines ordinary Receive/Set/Success. For all six corrected cases, the pre-trigger snapshot has marker0 and no progress observation, while the held snapshot has marker1/succeeded with consumed input. The held contact has entered but has no response. Capacity1 retains one contact; capacity2 retains three contacts with the two unheld siblings completed. Both held-value schedules finish against the same frozen ordered oracle. Fixed root+witness cost remains separate. The original two failures and exact source remain preserved; their missing pre-trigger snapshot pairs cannot now independently establish the precise failed interleaving. The disclosed race diagnosis is consistent with the old inspection-before-ledger ordering; the correction removes that causal mismatch without weakening the contact monitor.

3. **Closure and source versus scope authorization.** `closure_test.go:64` tests decision/SIGKILL/actual open-to-cancelled transition and denied successor admissions, and explicitly identifies repeated prior settlement as duplicate known evidence. The distinct live companion starts at `:139`. Its retained closure snapshot has no adapter result and an open peer; the later snapshot has a genuine first result. I independently verified the second operation `["peer",1,4,0]` has allowed source authorization and a denied `scope_closed` composition reservation, unchanged closure rows, no second contact, and charges0/0/1. This supports the two disclosed schedules; it does not establish a complete adaptive barrier containing an unknown required result.

4. **Unknown and denial.** `cuts_test.go:99` prevents a decision/public result while gamma is unknown despite beta's known failure. Each after snapshot retains the original unknown adapter attempt from before, no root outcome, no additional contacts, and unchanged charges including three effects. Root-denial snapshots/results preserve whole application count6, four accepted leaves, ordered denied10/11 with `invocation_limit`, and the lowered invocation ceiling across recovery. They are not ordinary exhaustion results.

5. **Authentic faulty application.** `controls_test.go:63` constructs an ordinary source overlay, executes it, and requires actual forbidden admissions. The retained faulty snapshot independently shows gamma0 consumed at adaptive sequence268 and accepted invocations for10/11/31. The unchanged paired oracle rejects `[1,2,3,7,10,11,31]` against `[1,2,3,7]`; Run/Replay acceptance is separately recorded. This is an application-policy counterexample, not a runtime replay counterexample. Preserve it.

6. **Read-only replay and tooling.** `process_test.go:267` uses actual sandbox-exec network denial, with no replay adapter registry or credential; retained raw records report equal ledger digests and seven contacts before and after. The capture/inventory machinery preserves failures, exact source bytes, drift checks and structured marker association. I inspected freeze/package/reproduction code, including pinned dependency handling and extracted-source verification. Those operations have not run for the final archive and are not operationally qualified by this review. The final extracted run is intentionally narrower than full-suite reproduction.

7. **Manuscript scope.** New case-study paragraphs in `paper/alan-dissertation.md:233` and `paper/alan-dissertation.tex:192` align this increment, distinguish bounded implementation evidence from proof, and label pre-rebase qualification historical. No reference change is needed for this repository case study. This does not resolve the separate ranking/confluence/general manuscript obligations recorded in the audit.

## Findings

No critical, important, or minor actionable finding was confirmed for this checkpoint. The evidence has genuine limitations, but the report names them and does not claim the pending gates have passed.

## Declined to judge / remaining gates

- Final default/integration/race and old/new consumer normal/race suites, applicable static/build/module checks and semantic comparisons: explicitly subsequent plan gates, unexecuted here.
- Final freeze/package/extracted offline reproduction operation and final reviewed archive identity: code inspected, actual operation pending.
- Whole-branch review, draft PR and actual CI status: later delivery gates; no merge approval.
- Distributed ownership, universal recovery/liveness/fairness, power-loss recovery, live endpoint behavior and compiler correctness: outside this controlled single-host checkpoint; finite fixtures do not prove them.
- General loop-ranking proof, fair-schedule confluence, wider Markdown–LaTeX alignment and proof rhetoric: pre-existing open manuscript obligations, not repaired or established here.
- Additional child cancellation/abandonment permutations and automatic eventual repair of unknown non-idempotent work: not executed or claimed.
- Controller's separate upstream baseline checks: not reviewed; a default command that skips integration-tagged runtime tests provides no runtime behavioral qualification.

Review verification consisted of read-only git/source inspection, Python standard-library archive/hash/raw-event checks, and `python3 /tmp/alan-phase1h-review-evidence.py` against saved artifacts. That inspection script completed successfully. No new behavior test, broad suite, or repeat loop was run.
