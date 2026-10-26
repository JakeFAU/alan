# Task 3 independent review

Review date: 2026-10-26. Scope: the Task 3 subprocess qualification and its retained evidence, independently reviewed against the application contract and task brief. No production edits, paid calls, credentials, heldout inputs, subagents, commits, or redundant suite runs were used.

**Specification verdict: PASS for the stated finite qualification slice.** The selected interruptions, partial sibling progress, unknown recovery, supported earlier-message redelivery, ordinary nested completion, and offline replay are evidenced. Arbitrary late adapter-candidate admission is explicitly outside the demonstrated slice. The brief itself permits reporting that unavailable public operation and exercising the closest supported settlement path, so the parent’s scope ruling is consistent with the reviewed requirement; it is not a waiver inferred from passing tests.

**Code-quality verdict: APPROVE.** No blocking correctness, causal-oracle, or evidence-integrity finding. The monitors use genuine positive inputs before negative controls, and the executable tests exercise the real CLI and provider process. The final normal/race logs record passing results; this review independently verified those retained results and source identities, without rerunning the suite.

## Findings and evidence

1. **Partial progress is causally established.** `TestPartialSiblingUnknownRecovery` pauses helper2 before terminal commit, observes the separate provider’s fsynced alpha accepted/applied events, then releases the commit. The committed acknowledgment is helper2 generation 1, epoch 2, sequence 59; its in-flight set contains the exact alpha authorization. The test binds it to the completed public projection and consumed synthesis observation, correlates the held request’s wire/client ID/digest to its durable adapter attempt, and checks both child PIDs alive. Alpha has no responding event. This proves selected sibling progress while the other provider operation remains held, without a sleep or controller-driven computation schedule.

2. **Review and revision cuts identify actual durable operations.** Initial review is epoch 2/sequence 11; second review is epoch 3/sequence 23, both `adapter.result-recorded/pre-settlement`. Independent parsing of both final logs confirmed the event’s source, generation, epoch, sequence, intent and authorization against retained authority and committed adapter/intent rows, with no settlement. The second cut has revision count one, consumed review1 and pending review2. Recovery completes ordinary report publication with 13 model contacts, four lookups and 17 attempts. Original reviewer attempts remain exactly equal across recovery. The second cut is a recorded second-review result after revision synthesis, not an earlier revision-helper crash; the report describes that concrete cut honestly.

3. **Live and retained epochs are correctly separated.** `executor.Open` increments epochs of open owners. Active boundary validation requires the acknowledged current epoch; durable validation separately requires the original attempt authority. Post-reopen preservation checks permit epoch advancement while retaining pending IDs, prior evidence and nondecreasing budgets. The substituted historical review acknowledgment still passes the actual configured wildcard fields and durable historical correlation, but is rejected as an active cut. Epoch/sequence mutants pass wildcard matching and fail exact durable correlation; the changed source fails both the configured source match and authority correlation. These are meaningful fault controls.

4. **Earlier evidence is admitted through the existing executor protocol.** After killing at pending review2, `DeliverMessage` redelivers helper1 reply `["investigation/helper1",1,57,0]` and reviewer1 reply `["investigation/reviewer",1,20,0]` with new transport IDs. Both admissions resolve to their original consumed observations. The live monitor compares every participant’s full projection and sequence, plus the complete observation rows, and verifies pending/budget/evidence preservation. It does not compare a hand-selected partial projection against itself. The retained regression narrows its inputs to the runs relevant to its pending-identity mutant; the live positive path still checks observations. Private `recordAdapterResult`/`settleAdapter` cannot be invoked through a supported external test API. No SQL or fabricated state supplies a late result. The demonstrated claim is duplicate old-message settlement during a pending newer review, together with recorded-result recovery, not arbitrary late HTTP response admission.

5. **Unknown outcome stays uncertain.** The provider computes and fsyncs the alpha payload before withholding a response. After kill/reopen, the exact helper1 attempt `["investigation/helper1",1,15,0]`, unknown list and completed sibling projection remain; the provider ledger is unchanged and revision count stays zero. No final result is asserted. The report correctly calls completion blocked rather than successful recovery completion.

6. **Replay checks all participants and complete store equality.** Each final mode contains three offline replays, with all 12 participants present once, exact retained projections, and history counts 475, 217 and 475 for recovered, unknown and complete stores. Source inspection confirms registry-free read-only `runtime.Replay`; tests compare full `investigation.View` values before/after, not the compact log projection. Provider process termination, removed key, source deletion, parsed ledger equality and ledger SHA equality provide separate evidence. Zero tool dispatch is explicitly inferred from the registry-free implementation; no dynamic lookup counter is invented. The negative wrapper actually invokes the real OpenAI adapter with a retained prepared request. Independent ledger inspection confirmed 13 to 14 accepted contacts and matching dispatched wire bytes; the unchanged replay monitor rejects it.

7. **Race children and source removal are real.** The race log contains `go build ... -race ./cmd/alan-investigate` and `go test -c ... -race ./internal/investigationtest`, and process records carry their binary hashes. Both final logs contain no data-race warning or failed-test marker. Corpus/manifest directories are removed before the relevant recovery or complete run; retained assets support the subsequent ordinary work. Longer race watchdogs bound infrastructure acknowledgment waits; the semantic request timeout remains two minutes.

8. **Failures and counterexamples are retained faithfully.** All nine fixture cases are exact parsed counterexample objects from `normal-fixture`; the fixture’s source digest matches that raw log. Both original failed logs remain lossless. Probe1 reports epoch-equality failures for caller; its captured pre/post views exactly match the raw log. Probe2 reports the full-RunState redelivery equality failure, and the missing post-view is explicitly disclosed. The correction follows the inspected `Open` behavior rather than weakening attempt-identity checks. There are three selected process scenarios and three CLI kills per suite; nine controls are not counted as additional schedules.

## Independent checks performed

Read the brief, report, patch, all seven archived/current test inputs, application contract, authoring observations, Task 2 report, boundary matcher, executor open/message/adapter/replay paths, and CLI routing. Fresh Python checks verified all seven source byte lengths/SHA-256 values, all eight compressed and decompressed log digests/lengths/outcomes, the six source-file patch bodies, archive byte equality, exact fixture provenance, original probe1 input provenance, final-log durable authority correlation, replay participant uniqueness/projections, and the actual extra provider dispatch. All checks completed successfully. No concrete gap justified another integration/race execution. The verification-before-completion discipline was applied to these review checks; archived suite results remain labeled as archived results.

## Exact review inputs

All hashes below are SHA-256. The manifest enumerates exact commands, environment, toolchain, eight raw/compressed log hashes and lengths; its own identity is pinned here.

| Input | SHA-256 |
| --- | --- |
| `manifest.json` | `e2c38053cef718ff5130c3422108b74ac61c8048583ca635d99a8c6f8d207b8d` |
| `manifest.original.json` | `1a64bd98b3b85ffdb4dde1a00f47e7d5eb74b185ce4ea354203e1b47e85d82fb` |
| `report.md` | `582334ab0b2b17f1e2aa42072f61dda58367333adc8134f390cbe2e78a002cf0` |
| `review-source.tar.gz` | `b49112e79c113ef2c80549a1864e969863dc6b766718fc4610aae20ffa9dba74` |
| `original-epoch-probe.json` | `e3613eb1a2793c5ba26c221b7b7fcd3478b9ef29d2be36ac9e990909c23eb595` |
| `.superpowers/sdd/00-plan/task-3-brief.md` | `5bbd941b866b53c68b1d61872b147b64a5a3954b02110602adbf8ef1bb0ac914` |
| `.superpowers/sdd/00-plan/task-3-review.patch` | `a3ecba20a490374210f95c7ccc83c00ab367109c45727b664579044994fafdfa` |
| final normal raw JSONL | `4ea65b1fd8f734b1716ddff16d872a267b72d601c918085eecb2173243150be4` |
| final race raw JSONL | `7c7fd7a39be08be71007fadbf1853b7cda34a0617b393f3f47c95796831b4e84` |

Paths in the next table are relative to `internal/investigationtest/`.

| Reviewed source | SHA-256 |
| --- | --- |
| `counterexamples_test.go` | `01b507e049f9b9da8706b367b8001f52c8be63454ebb99ce610af4754ed5e77a` |
| `process_test.go` | `5a2227ef295de32bd2e37c93c555609e704418d9fd89138d68ff11ca750de9d8` |
| `provider_test.go` | `94d989c84960c3ebfdaafae99378b2f8a25522164935948d30470f16e50b4e15` |
| `qualification_test.go` | `666584c2584750dc8f554749371be9aeee15889e1ee94d52cb7e617bbec7a1da` |
| `race_off_test.go` | `5b88c6476ef0f926859d9c7709f977d1d5788b376ca439d9084a3586f4b156d0` |
| `race_on_test.go` | `e200c742186e70dd0efe07b3426dd14eb6b3bd80dcad24a16977122eb1cdcc07` |
| `testdata/retained-counterexamples.json` | `cb37937612009e28c5767349436a3a4094bf934429bafbb1d3787078c8c4523b` |

Final normal archive records exit 0 in 7.491s; final race archive records exit 0 in 87.458s. This approval supports the selected schedules and monitor sensitivity. It does not establish exhaustive safety/liveness, eventual completion of unknown work, semantic answer quality, arbitrary late-adapter admission, business idempotence or universal cancellation.
