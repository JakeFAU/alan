# Phase 1D independent whole-branch review

Reviewed 2026-10-25. Baseline `a0d337c7b4f7340a55dea11b619cf9b22c25e2b0`; branch `codex/phase1d-durable-executor`; implementation is uncommitted. Reviewed source digest `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`, all 53 entries independently verified. Archive `49f68316357b07ac4bbd3f979cb47ebe25e732a28e3375081fb9ccacfcd8f3ea`, all 83 members independently verified.

**Specification verdict: PASS for the declared Phase 1D implementation profile. Quality verdict: PASS with one Minor documentation correction before publication. No Critical or Important finding; no observed runtime defect. Ready for a draft PR once M1 is corrected and the pending publication/evidence checkboxes record their actual completion. No merge/release recommendation.**

This was a fresh independent review, not a concatenation of scoped approvals. The repository, index and branch were not edited; no subagents were spawned. Temporary extracted source, the reviewer probe, logs and this report are under `/tmp/alan-phase1d-final-review`.

## Complete finding list

### M1 — Minor: rejected-result crash row overstates successor execution

Location: `docs/foundational/phase1d/02-qualification.md:44`, supported by `internal/durabletest/faults_test.go:145` and the published `TestRejectedResultCrashAndLostRejectionControl/lost-rejection=false` DURABLE record containing `after_new_request`.

The row says: “A distinct authorized operation succeeds and its successor computes.” The actual retained snapshot has `worker`, generation 2, sequence **0**, status **open**. Its initial instruction waits for ordinary input. This fixture proves new-operation acceptance and successor creation, but does not execute a successor instruction. The requester completes at sequence 4; that is not successor computation. Both normal and race records retain this state.

Impact: the table assigns stronger evidence to this specific process-crash fixture than it contains. It is not a runtime safety defect or a gap in the central accepted-crash claim: the accepted nested crash case executes worker generation 2 through sequence 3, and the separately labeled conformance rejection extension also executes its successor.

Correction: state that the distinct operation creates generation 2, and attribute actual successor computation to the accepted nested crash case and separate conformance extension. No runtime or test change is necessary. The parent acknowledged the finding and plans that correction; this report records the original finding rather than claiming a correction already observed.

Exact original wording and both full evidence records are retained in `minor-evidence-precision.json` alongside this report. This is the complete actionable finding list.

## Review scope and technical assessment

Read the complete runtime production paths in `program.go`, `store.go`, `messages.go`, `runner.go`, `lifecycle.go`, `lifecycle_replay.go`, `external.go`, `replay.go`, `inspect.go`, the ordinary example builders, both CLIs, boundary observer and independent resource. Reviewed the full durable controller/oracle/fault-control implementation, the independent conformance bridge, focused persistence/protocol/correlation regressions and the source review package. Read all six Phase 1D plan/contract/qualification/review/reproduction/runbook documents, manifests, retained original findings and counterexample descriptions, machine evidence and archive reproduction record. Unchanged historical model source was checked for byte preservation; this was not a new exhaustive audit of every old model transition or the vendored SQLite implementation.

The local execution transaction consumes the selected observation, stores exact reduction/history/intents, advances projection/sequence, and freezes terminal notification obligations together. Staged and committed barriers bracket the actual transaction writes/COMMIT. No production dispatch occurs from the speculative reduction.

Source authorization is separately committed and retained. A previous allowed authorization survives later closure; a new denied authorization remains explicit. Target lifecycle permission and expected generation/revision/retirement/eligibility are checked separately. Existing results compare the complete request and return before reevaluating changed eligibility. Target mutation, result and successor/closure provenance share the decision transaction. Settlement and later ordinary consumption remain separate.

Startup reconstructs ordinary history, exact generated descriptors and consumed observations, then reconciles authority-caused closure and Retry origins. Zero-step successors and sequence-zero authority closure are represented without fabricated ordinary commits. Epoch renewal follows validation and changes only open-run epochs. Read-only replay uses mode=ro/query_only and a consistent snapshot, with no dispatcher or live activation opener. Inspection is intentionally a raw coherent evidence view, not the semantic replay validator.

Trusted admission domains separate raw input transport strings from runtime message/terminal/lifecycle/receipt identity. Existing admission results preserve original bindings. Correlation includes receiving run, kind, provenance, source generation, obligation and operation. The ordinary nested programs and scheduler have no role/name dispatch; Retry budgets and pending IDs are ordinary persisted data, and the successor interprets explicit input using the retained artifact.

External effects cross a real separate process/store boundary. The runtime uses the endpoint protocol, not endpoint SQL. Original-key idempotent reattempt returns the first receipt; non-idempotent prior attempts remain unknown without automatic repetition. Source closure does not retroactively cancel authorized uncertainty. This does not imply business-level exactly-once effects or cross-store key namespacing.

The controller validates scenario/start token/PID/store/boundary/phase/run/generation/epoch/sequence/intent/authorization and retained facts, kills the corresponding process and requires SIGKILL exit. Timeout is a test failure, not evidence of a boundary. The actual-pipe mismatch regression retains the original cross-execution bug. All eight requested windows are represented, including second recovery crash. Fault injection is test-only; the common record checker does not call the production reducer or authority decision function. Lost rejection, reset policy state, and escaped speculative dispatch have observable incorrect states and unchanged-oracle detections, not merely precondition failures.

## Independent executions and evidence audit

All commands ran in the extracted `/tmp` copy with `GOCACHE=/tmp/alan-phase1d-go-build GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.

1. Independently authored `internal/executor/final_review_probe_test.go`: `go test ./internal/executor -run '^TestFinalReviewReachableLifecycleSchedules$' -count=1 -v`. PASS, 18.974 seconds. Twenty deterministic seeds, each up to 80 schedule choices, three ordinary static participants, mixed Retry/CancelRun/AbandonRun/RetireSubject requests including self-targets, repeated decision/settlement, closure and reopen. Every completed action boundary is checked by Replay; repeated decisions must be identical; reopen must preserve logical state and advance only open-run epochs. Public APIs only, no SQL mutation. This probes reachable consistency rather than independent semantic equivalence or exhaustive scheduling.
2. Fresh `go test -count=1 ./...` in that extracted copy, including the added reviewer probe: PASS for every package. Reported package times: CLI 0.574s, boundary 0.073s, durable controller 5.694s, executor 10.495s, resource 0.227s, semantics 5.183s. Output: `authorized-tests.log`. Initial sandboxed focused/controller attempts failed because local socket binding was prohibited; `focused-tests.log` and `crash-tests.log` retain those environment failures. The authorized local-loopback rerun passed; no remote service was used.
3. Independent `audit.py`: all 53 source entries and aggregate digest match; all 83 archive entries match. Raw normal/race Go JSON logs independently reassembled into all 160 marker records per mode and compared exactly with published records, including Package/Test context and ordering. Each mode has 321 passing test/subtest events, 87 DURABLE records, 16 killed-process records matched to exact PID/config acknowledgments, and three nonempty fault-control diagnostics. Output: `evidence-audit.log`.
4. All thirteen pre-existing research Go files byte-match the baseline via `git show`; `preservation-audit.json` records paths. The existing different-execution and misordered interruption regressions remain and pass in the fresh full suite.

I reviewed rather than reran the complete published race suite, static checks and historical archive reproductions. Their exact-source logs are retained; both raw normal/race streams have no failing test events. The parent-provided offline archive reproduction used an initially empty module cache and only the bundled local proxy, passed full tests, mod verification and vet, and verified unchanged members. My separate archive/member verification agrees with its identities.

## Considered behaviors and explicit limits

- A deleted unreferenced rejection can evade self-consistency replay. That is the deliberately injected external-writer fault and is detected by the independent before/after oracle; arbitrary coherent database tampering is outside the storage profile.
- Runtime result consumption followed by pending clear has a real consumed-but-pending intermediate. The bridge retains it, reopens it, and calls it neither equivalent nor a stutter. Equality applies only at selected common macro-boundaries. Global epoch renewal, explicit external control inputs and closed-run receipt observations are also disclosed representation differences.
- A new external demo store reuses execution-derived keys; the runbook now requires a fresh resource/executor pair when switching profiles. No business namespace guarantee is claimed.
- Malformed program operands, capacity/byte bounds and unsupported effects can stop or leave unresolved invocation work; the contract does not promise arbitrary-program fairness or eventual recovery. Generation overflow is an explicit operational limit.
- Read-only SQLite WAL bookkeeping is distinguished from semantic dispatch. Hostile aliases, external writers, replacement of a live database, initialization interruption, power loss, distributed fencing and unbounded histories are not established by these process-crash tests.

No additional concrete in-scope defect was found. Passing tests, the schedule probe and this review are finite evidence, not an unbounded correctness proof.
