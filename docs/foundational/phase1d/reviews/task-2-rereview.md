# Task 2 fix round 1 independent scoped re-review

Reviewed 2026-10-25. Scope: R1 transport-domain collision correction and fix-introduced breakage only. Read the original review, amended report, complete `task-2-fix-1.diff`, and new regression tests. All 17 current-file SHA-256 values match the frozen package. No repository source edits or subagents.

**R1: ADDRESSED.**

**Specification verdict: PASS for Task 2. Quality verdict: PASS for Task 2.** No blocking scoped findings remain, and no fix-introduced correctness defect was found. This is not a Phase 1D completion verdict.

## Identity and provenance checks

The durable admission key now includes the trusted domain and exact raw transport string. Domain selection remains internal to the input/message/terminal API; raw strings are wrapped as data and never interpreted as discriminators. Domain and raw transport have explicit columns with a uniqueness constraint. Exact descriptor conflicts remain enforced within each domain.

Both original valid inputs are preserved exactly in executable regressions: the accepted input using the generated message-delivery string and the rejected input using the generated terminal-delivery string. They remain legal and retain their original accepted/rejected results. The intended authenticated message/terminal delivery progresses, and the unrelated colliding ordinary input is not consumed as a privileged observation. Close/Open, retransmission, inspection, and Replay retain the typed bindings.

Message ACKs resolve only message-domain admissions and persist the canonical admission key. Terminal ACK lookup is terminal-domain-specific before the frozen route acknowledgment changes. Replay checks the stored key against domain/raw ID, domain against descriptor kind, and the raw ID against descriptor/result; message ACK references must resolve to the matching message intent. The same arbitrary raw string can coexist across all three domains, while changed input/message/terminal descriptors still conflict within their own domain.

## Focused independent execution

I copied the frozen source into `/tmp/alan-task2-rereview-probe` and ran the original-input regressions, all-three-domain/same-domain-conflict regression, and transport/ACK corruption regression. I added one temporary reviewer probe there, without changing repository source:

- A raw input ID equal to the complete canonical admission-looking string `["admission","message","inner"]` remains valid.
- That input alone cannot satisfy AcknowledgeMessage for the same raw string.
- Message transport `inner` and a message transport whose raw string looks like its canonical admission key coexist with that input.
- Both message transports retain the same logical observation, the typed ACK succeeds only after message admission, and Replay validates the result.

Command: `GOCACHE=/tmp/alan-phase1d-go-build go test -v ./internal/executor -run 'TestReviewNested|TestAcceptedInputCannotShadow|TestRejectedInputCannotShadow|TestMessageTerminalAndInputShare|TestReplayRejectsChangedTransportDomain'`.

Result: PASS, 0.328s. Exact output: `/tmp/alan-task2-rereview-probe.log`. Additional probe: `/tmp/alan-task2-rereview-probe/internal/executor/review_extra_test.go`. No full-suite rerun was needed for this scoped review; the amended report retains same-source full/race/vet results.

## Continuing boundary

The schema refinement is explicitly unreleased, with no migration or compatibility claim for earlier prototype stores. The real accepted-Retry service-binding check remains Task 3 cross-slice work, as originally agreed. Lifecycle reconciliation, endpoint uncertainty, full process-kill qualification, independent policy/research comparison, and final archive verification remain later acceptance requirements.
