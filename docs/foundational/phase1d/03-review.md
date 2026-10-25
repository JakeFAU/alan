# Phase 1D review history

This is a chronological record. Earlier pending/rejection statements describe their review stage; later dispositions retain and resolve them. The final whole-branch verdict appears at the end.

## Independent preimplementation contract review

Reviewed 2026-10-25. Read-only review of `docs/foundational/phase1d/{00-plan,01-durability-contract}.md`, Phase 1C supervision contract and retained review findings, and foundational decisions. No runtime source review, branch changes, or test execution was performed.

**Specification verdict: PASS for proceeding with implementation, with the narrow acceptance criteria below made explicit before qualification.** The plan preserves the important Phase 1C boundaries and does not require architectural redesign. **Implementation quality verdict: NOT ASSESSED; final source/controller review remains required.** These are prospective ambiguity risks, not claims of defects in code that has not yet been reviewed.

## Clarifications to carry into implementation and tests

### C1 — Startup validation must account for authority transitions outside reducer history (high priority)

Contract lines 40–46 and 68 require reconstruction of history/projection and also allow CancelRun/RetireSubject/Retry to change lifecycle state separately. Pure reducer replay alone cannot explain a run cancelled by a lifecycle transaction, the creation of a successor with no execution commits yet, or a retained notification created by external closure.

Clarify that startup reconstructs ordinary execution state from program/history and separately reconciles lifecycle-authority changes against retained operation results and terminal obligations. It must validate history continuity, selected observation identity/content/consumption, exact intent descriptors, current sequence/projection, terminal status/outcome, and successor linkage. Do not treat lifecycle-caused status differences as either ignorable mismatches or automatic history corruption. A lifecycle decision that closes a run must atomically retain its terminal outcome/notification obligation, even though it is not an ordinary execution commit.

Acceptance tests: reopen immediately after accepted CancelRun, RetireSubject, and Retry; include a successor with zero instructions committed; corrupt a lifecycle successor link and a selected observation while keeping the latest projection unchanged; verify startup rejects before epoch advancement or dispatch. This closes a particularly likely gap between Task 1 replay and Task 3 protocol work.

### C2 — Separate source denial from target rejection, while retaining both outcomes (high priority)

Contract lines 43–44 clearly distinguish source authorization from target rights, but do not explicitly name the durable result of source authorization denial. Phase 1C requires an unauthorized lifecycle request to obtain a retained rejection outcome. Returning only a transient Go error or leaving the operation pending indefinitely would lose that behavior.

Clarify the durable source disposition and how its denial is exposed to a still-open requester. Target-grant denial must remain a lifecycle result after valid source authorization. Retransmission first compares the complete immutable descriptor and then returns the retained accepted/rejected result; changed bytes under the same identity are an identity conflict, even if a prior result exists. A retained authorization survives subsequent source closure and is usable for completion/settlement; closure must not cause a new source-validity check to erase it.

Acceptance tests: observation allowed/target grant denied, source authorization denied, authorization then source closure then target decision, and rejected-while-target-active Retry followed by actual target failure and identical retransmission. The last case must preserve the original rejection while a genuinely new operation can be accepted; expected-generation mismatch must not be the reason the old operation stays rejected.

### C3 — Define physical retransmission under retained authorization (medium priority)

Lines 43 and 62 allow recovery of an idempotent original intent but forbid new authorization after closure. State whether a physical retry uses the same durable authorization identity or a new attempt identity derived from the retained authorization. The distinction must prevent the dispatcher from quietly creating a new permission after closure, without making an already authorized uncertain operation unrecoverable.

For the non-idempotent profile, unknown must be visible and automatic work selection must stop short of blindly reissuing it. If the demonstration repeats it, label that repetition as an explicit authorized experimental action and independently count both endpoint mutations.

### C4 — Explicitly reject hard-linked database aliases before SQLite open/write (medium priority)

Line 11 permits rejecting unsupported aliases, which is sufficient if concretely implemented. Canonical absolute paths plus symlink resolution do not unify hard links. Clarify supported alias behavior: relative/symlink aliases converge to one lock identity; multiply linked database files are rejected (or another equally strong inode-aware design is documented). A per-path sidecar lock alone is insufficient for hard links. Acquire exclusivity before initialization, journal changes, epochs, or any writes. Do not unlink the sidecar on close. File replacement while live remains expressly excluded.

Acceptance tests must use a second OS process, including an alias path, and show unchanged epoch/ledger after exclusion; then show a fresh process succeeds after SIGKILL releases ownership. SQLite explicitly warns that multiple filesystem names can separate associated journals ([SQLite corruption guidance, §2.6](https://www.sqlite.org/howtocorrupt.html)).

### C5 — Replay isolation is logical and capability based, with an actual read-only open (medium priority)

Lines 7, 11, and 68 should mean read-only SQLite open **plus** query-only defense, not query-only alone. SQLite documents that `query_only` still permits checkpoint operations ([PRAGMA query_only](https://sqlite.org/pragma.html#pragma_query_only)). Use a consistent read transaction for all inspected relations. Replay must not call the live opener, increment epochs, provision missing state, process a lifecycle result, or contact the endpoint. A missing store is an error, not initialization. Do not use `immutable=1` against an actively changing WAL database.

Measure semantic ledgers/epochs and endpoint mutation state before/after under a quiescent comparison window. A strict raw-file-byte equality rule may confuse SQLite sidecar mechanics with ALAN mutation; record what the evidence comparison covers. Separately retain an independent policy/record checker: rerunning the production reducer only proves self-consistency.

### C6 — Crash barriers must identify the killed execution and actual committed boundary (high priority for controller review)

The plan already requires this, but carry the Phase 1C cross-execution witness defect directly into controller assertions. Every acknowledgment should include a unique scenario/process-start identity, PID, store identity, exact failpoint, relevant run/generation/epoch/sequence, operation/intent, and commitment phase. The controller must wait for that acknowledgment, kill that PID with SIGKILL, and assert signal termination. Never substitute a timeout, normal exit, stale acknowledgment file, or another execution's barrier.

For uncommitted transaction tests, acknowledge after the relevant writes inside the still-open transaction; for committed tests acknowledge only after successful COMMIT. Endpoint application acknowledgment must come from its independent process after its own transaction. Recovery receives only the existing store path plus ordinary explicitly declared input; it must not rebuild policy state from controller memory. The second crash must be in recovery of the same logical execution/obligation/operation, with a later activation epoch. Retain the lost-rejection, reset-policy-state, and uncommitted-dispatch fault controls against unchanged checks.

## Already clear and worth preserving

Stable transport admission retains both rejection and first run binding across generation changes. Different transport IDs still require logical observation deduplication and exact kind/provenance/correlation. External input may not spoof lifecycle/terminal/receipt provenance. No runtime branch may select behavior by worker/supervisor name, ancestry, retry count meaning, or generation number; a renamed fixture should produce the same public behavior. New generations execute the same retained instruction semantics and explicit input protocol. No new obligation or budget reset follows from physical activation replacement.

## Primary-document checks

The pinned [go-sqlite3 v1.14.52 release](https://github.com/mattn/go-sqlite3/releases/tag/v1.14.52) exists and was labeled latest at review time. Before qualification, record its actual bundled SQLite version/source ID from the built binary, not the operating system SQLite executable. Current [SQLite WAL documentation](https://sqlite.org/wal.html) describes a WAL-reset race fixed in 3.51.3 (with 3.44.6/3.50.7 backports); verify the bundled engine includes that fix. One execution writer reduces exposure but is not a reason to omit engine identity.

The documents correctly describe WAL as same-host and FULL synchronization as an additional commit sync; process SIGKILL evidence does not prove power-loss survival. Confirm all writer/read-only connections read back the intended settings, and preserve exact CGO/build/platform dependencies in the final reproducible source archive. SQLite's documentation also explains read-only WAL prerequisites; test inspect/replay against the real post-SIGKILL WAL state rather than assuming a cleanly closed database.

## Task 1 implementation review and retained output-bound defect

The independent task reviewer checked the frozen eight-file core/dependency package and its complete source/tests. Initial verdict: **specification FAIL, quality CHANGES REQUESTED**, solely for R1/P2, generated intent output amplification. Other core atomicity, same-run recovery, replay, exclusivity and subprocess barrier requirements passed that scoped inspection; later protocols were not yet assessed.

The exact valid input is retained by `amplifiedOutputFixture(500000)` in `internal/executor/program_test.go`: a 1,163-byte Emit/End program copies a 500,000-character local string into payload keys a–h. The projection is 500,092 bytes. Before correction, reduction generated a 4,000,627-byte intent array and Commit persisted a 4,000,625-byte descriptor. Program, state and observation limits did not constrain the generated output. The probe used public APIs and legitimate data; it did not modify database state or bypass a guard.

The [original probe](counterexamples/task1-output-amplification.go.txt), [actual failing output](counterexamples/task1-output-amplification.log), and [complete pre-fix core source with probe](counterexamples/task1-output-before-fix.tar.gz) are retained. The pre-fix archive SHA-256 is `b821bbb5ac0eccc2b4094b646d89bbf2054fee1a6d924b1c0b26741f6e9e05b8`. Extract it into a separate directory and run `go test ./internal/executor -run '^TestReviewEmittedPayloadBound$' -count=1 -v` with the pinned driver available: failure demonstrating the accepted oversized intent is expected. This historical probe intentionally differs from the current regression's rejection assertion.

The correction introduces explicit configurable `MaxOutputBytes`, default 1 MiB, covering the complete intent-array encoding. Incremental accounting checks generated descriptors before execution writes without first materializing the amplified whole JSON buffer. It preserves exact descriptor comparison during replay. The cost is explicit: an output exceeding this operational limit is refused even when its program/state inputs fit; a caller may supply a larger declared limit. The same original input passes with a declared 5 MiB limit and fails with the default. The original counterexample is preserved rather than removed.

Tests retain the original rejecting input, an eight-reference 100,000-character successful counterpart, exact escaped-JSON boundary behavior, unchanged state/history/intents/observations on rejection, and replay with adequate versus inadequate limits. A separate evidence-precision note prompted one-field receive-correlation tests for kind, provenance, source, source generation, obligation, receiver subject and receiver generation. The source had already checked those fields, but the earlier test independently perturbed only operation identity.

Two implementation findings were also retained before review: ordinary End initially accepted authority-only cancelled/abandoned outcomes, and duplicate receive bindings could depend on Go map iteration order. `TestOrdinaryEndCannotClaimAuthorityOutcome` and the ambiguous-binding regression preserve those inputs. Ordinary End now reports success/failure; authorized lifecycle closure owns cancellation/abandonment.

Task 1 scoped re-review returned **R1 ADDRESSED, receive coverage ADDRESSED, specification PASS, quality PASS**. The reviewer verified the frozen file hashes and found no fix-introduced defect. This approves the core slice only; whole-phase source and crash-controller review remains required.

## Task 2 implementation review: transport namespace collision

The independent review of the frozen sixteen-file source/dependency package returned **specification FAIL, quality CHANGES REQUESTED** for R1/P2. An ordinary input's raw transport ID shared the admission primary-key space with automatic message and terminal delivery IDs. Both accepted and rejected input admissions could reserve a future delivery ID. The automatic runner then stopped with `ErrConflict`, and recovery retained the conflict. This repeats the identity-isolation issue retained in the original Phase 1 research; preserving that model does not make a separately written runtime immune to it.

Two original valid inputs are retained in the [probe](counterexamples/task2-transport-shadow.go.txt) and [actual failure output](counterexamples/task2-transport-shadow.log):

- An accepted input to receiver uses raw transport ID `["message-delivery","[\"sender\",1,1,0]"]`, logical ID `ordinary-input`, payload number 7. The ordinary sender later commits its message with number 42. Run stops after one local commit and zero deliveries, and fails identically after reopen.
- A normally rejected input to a missing target uses raw transport ID `["terminal-delivery","child",1,1,"observer","public"]`, logical ID `rejected-input`, and an empty payload. The child's later committed failure cannot deliver its frozen terminal notification.

Neither probe forges runtime provenance, modifies SQL, or manufactures a generation. Both original strings remain legal regression inputs. The requested correction gives each trusted transport domain its own durable identity, updates acknowledgment references/replay consistently, and preserves same-domain descriptor conflict checks. An additional regression must permit message and terminal deliveries to use the same raw transport string without collision.

The [complete pre-fix source with probes](counterexamples/task2-transport-before-fix.tar.gz) is retained, SHA-256 `6a4658348cba1bc9ad36e63fd0f6e2726d1ba359702c170f8a99200400f7f81a`. Extract separately and run `go test ./internal/executor -run 'TestReview.*Shadow' -count=1 -v` with the pinned driver available; these historical probes are expected to fail at their progress assertions.

The correction qualifies durable admission identities by the trusted input/message/terminal domain, while preserving the caller's raw ID and same-domain immutable comparison. ACK references and replay validation use the qualified key. Four regression groups retain the original accepted and rejected inputs, simultaneous reuse of one arbitrary raw string across all three domains, retransmission/reopen binding, and corrupted key/domain/ACK rejection. No raw prefix was banned.

Scoped independent re-review returned **R1 ADDRESSED, specification PASS, quality PASS**. The reviewer also exercised an admission-looking raw string and found no fix-introduced defect. This is approval of Task 2 only; lifecycle, external-resource and whole-phase crash qualification remain separate reviews.

## Task 3 implementation review: retained lifecycle decisions

The independent [complete review](reviews/task-3-review.md) returned **specification PASS, quality PASS** for the lifecycle slice. It checked distinct intent/authorization/decision/settlement/consumption boundaries, immutable original results, source-denial versus target-rejection semantics, static grants, original-generation correlation, run creation and sequence-zero closure provenance, and validation before epoch renewal. No scoped defect was found.

The review also examined the nested ordinary programs. Worker behavior follows explicit input in an unchanged artifact. Inner reserves its own budget with its Retry intent, consumes the retained result and continues; outer observes only the public inner terminal outcome. No runtime role dispatch or supervisory opcode was introduced. Representation exhaustion at `MaxInt64` is explicitly an operational error with retained prior work and no partial decision.

Three independently authored [probe definitions](reviews/task-3-probes.go.txt) and their [actual output](reviews/task-3-probes.log) are retained. Repeated bounded runner invocations with Close/Open exercised three retries through worker generation 4 while inner stayed generation 1 and preserved spent=3. Five fixed seeds produced 500 attempted lifecycle actions, with reconstruction checked after each attempt. A message-specific probe confirmed original service binding after actual Retry and namespace isolation from a colliding raw input ID. These are successful finite probes; the 500 actions are neither an exhaustive safety search nor an independent semantic oracle.

Review approval remains limited to this slice. Full crash-controller, endpoint, negative-control and model/runtime comparison review is still required.

## Task 4 construction finding: crash-acknowledgment identity

Coordinator inspection during controller construction found that `child.wait` compared scenario, process token, PID, store, boundary, subject and sequence, but only required generation/epoch to be positive. It did not compare operation/attempt identities to the expected values. This weakened evidence about exactly which execution had reached the crash boundary.

The [original probe](counterexamples/task4-barrier-identity.go.txt) calls the actual original `child.wait` through a pipe. With expected inner/generation 1/epoch 2/sequence 7/operation A/authorization A, it supplies the same process context but generation 2/epoch 99/operation B/authorization B. The [actual failure output](counterexamples/task4-barrier-identity.log) shows that the controller accepted it. The probe deliberately reports failure after that acceptance and never kills a process. This is an oracle defect, not evidence that the executor itself produced the wrong acknowledgment or violated a runtime invariant.

The [complete pre-fix source and probe](counterexamples/task4-barrier-before-fix.tar.gz) preserve the original input. Run `go test ./internal/durabletest -run '^TestReviewBarrierAcceptsWrongLogicalIdentity$' -count=1 -v` in an extracted copy with the pinned dependency available; the acceptance failure is expected. Correction and independent controller review remain pending.

The implementation correction now validates all declared identity fields, checks corresponding retained executor/endpoint facts before SIGKILL, and uses an explicitly predicted identity for uncommitted writes. Pipe-based regressions retain one-field changes to generation, epoch, operation and attempt, plus wrong phase and older/newer sequence. Failpoint filters now honor generation/epoch/attempt selection. These checks have passed in development; independent source-and-controller review is still pending.

## Task 4 independent source and crash-controller review

The [complete independent review](reviews/task-4-review.md) returned **specification PASS, quality PASS**, with no new actionable defect. It verified all 37 frozen source identities, both sets of 87 actual evidence records, all recorded binary hashes, exact correlation for all 16 SIGKILLs in each set, and race instrumentation in the executor, resource and fault-helper children. The [evidence audit](reviews/task-4-evidence-audit.log) is retained. The controller identity correction above is **addressed**; wrong-generation, epoch, operation, authorization, phase and sequence inputs now fail the actual pipe-validation path.

The reviewer verified the three unchanged-oracle detections against their actual persisted evidence. Lost rejection changed a previously rejected operation to accepted after genuine target failure, with grant/generation unchanged. Reset policy state disagreed with retained ordinary history after a valid reopen. Speculative dispatch left a real resource mutation after the executor transaction rolled back, with no committed intent/authorization. Correct counterparts passed the same record checker.

An additional independently authored [external-grant/provenance probe](reviews/task-4-probes.go.txt) and [passing output](reviews/task-4-probes.log) separate allowed source authorization from denied external permission. A legal input using the future raw receipt-transport ID remains isolated from the actual authenticated denial receipt through consumption, replay and reopen. The resource fixture has no runtime policy oracle, and the executor uses only its declared HTTP protocol.

This approval covers the implemented crash slice. Selected model/runtime comparison, final source archive and whole-phase review remain separate gates.


## Task 5 — independent model comparison and external CLI example

A separate implementer added six public-projection comparisons and the ordinary external CLI fixture without modifying the thirteen existing model source files or the runtime interpreter. Independent scoped review returned specification PASS and quality PASS; see [complete Task 5 review](reviews/task-5-review.md) and [evidence audit](reviews/task-5-evidence-audit.log).

The review checked all 53 source identities, exact model seeds, retained model counterexamples, named comparison boundaries, and generation-2 receipt eligibility. Runtime Receive followed by an ordinary pending-clear Set is two commits, while the model combines them; the unmatched intermediate is retained, reopened, and explicitly excluded from equality claims. The external J comparison uses separate runtime input steps and an in-process endpoint, so its claim is narrower than whole-trace equivalence or process isolation.

A documentation clarification was retained: new built-in demo stores reuse the same execution-derived external key. Switching endpoint profiles requires a fresh executor/resource pair; a fresh database filename does not create a business-key namespace. This is a source-derived limitation, not an executed runtime defect.

## Final whole-branch gate

All five scoped implementation reviews passed. The complete frozen source, final verification records and source archive are supplied for a fresh independent whole-branch review; its actual verdict and any findings are recorded below when returned. Scoped approvals are not a proof and do not substitute for that gate.


## Final independent whole-branch review and M1 correction

The fresh [whole-branch review](reviews/final-review.md) returned **specification PASS, quality PASS with one Minor documentation correction before publication**. It found no Critical or Important issue and no observed runtime defect. Its recommendation is readiness for a draft PR after M1 and publication/evidence bookkeeping are completed, not a merge or release recommendation. This review independently examined the complete implementation and qualification package; it was not a concatenation of the five scoped approvals.

The reviewer verified all 53 source entries and digest `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`, all 83 archive members, all 160 extracted marker records per normal/race mode with exact Package/Test/order context, and the 16 SIGKILL identities in each mode. The [audit script](reviews/final-audit.py), [audit output](reviews/final-evidence-audit.log), and [historical preservation audit](reviews/final-preservation-audit.json) are retained verbatim. A separately authored [schedule probe](reviews/final-review-probe.go.txt) passed twenty deterministic seeds of up to eighty lifecycle schedule choices using public APIs; [probe output](reviews/final-schedule-probe.log) and the [fresh full-suite output](reviews/final-authorized-tests.log) are preserved. The probe used Replay as a consistency check, not an independent semantic oracle, and is stored as noncompiled `.go.txt`. The original [focused](reviews/final-focused-tests.log) and [controller](reviews/final-crash-tests.log) attempts preserve their sandbox socket-binding failures; the authorized local-loopback rerun passed. The reviewer inspected the published race/static/historical-reproduction evidence rather than rerunning each of those checks.

**M1 — Minor, addressed; scoped independent re-review PASS.** The rejected-result crash row in [02-qualification.md](02-qualification.md) previously said that the distinct new operation's successor computes. The normal and race `after_new_request` snapshots actually retain worker generation 2 open at sequence 0, waiting for ordinary input. The row now states successor creation and waiting, and attributes actual successor computation to the accepted nested crash case and the separately labeled conformance rejection extension. The [original wording and both complete records](reviews/final-minor-evidence-precision.json) are preserved unchanged. The original reviewer report records the finding as observed before correction. No runtime, Go test, model, manifest, archive or existing machine-evidence file changed for this correction. This is the complete actionable whole-branch finding list. The [scoped independent re-review](reviews/final-rereview.md) verified the corrected attribution and original records, confirmed unchanged source/archive/evidence, and returned specification PASS, quality PASS, with no remaining actionable findings. Ready for the authorized draft PR; no merge or release is recommended.

The whole-branch verdict retains the declared limits:

- Evidence covers selected finite programs, schedules and process-crash boundaries on one active host/process under local SQLite/filesystem assumptions. The schedule probe is reachable-consistency evidence, not exhaustive scheduling, arbitrary-program safety, whole-machine bisimulation or an unbounded correctness proof. Power loss, disk failure, initialization interruption, replacement of a live database, hostile aliases/external writers, arbitrary coherent database tampering and distributed fencing are not established. Read-only WAL bookkeeping is distinct from semantic dispatch.
- Model/runtime equality applies only at named common macro-boundaries. Runtime Receive and pending-clear Set have a real unmatched intermediate; global epoch renewal, explicit external control inputs and closed-run receipt observations also differ. The external J endpoint is in-process with separate durable state; the controller suite supplies process isolation. Replay checks runtime consistency and is not the independent model oracle.
- Execution-key deduplication is not business exactly-once behavior, business deduplication, compensation or inference of an unknown outcome from failure. Fresh demo stores reuse execution-derived keys; switching profiles requires a fresh executor/resource pair. No cross-store business-key namespace is provided, and non-idempotent uncertainty can persist indefinitely.
- The interpreter has one PC per run, checked int64 data and explicit finite byte/work bounds. Malformed operands, unsupported effects and capacity limits can stop or leave invocation work unresolved. Generation overflow returns an operational limit without a committed target result. Successful progress witnesses do not establish arbitrary-program fairness or eventual recovery.
- Histories are unbounded in storage growth, with no garbage collection; startup/replay scan retained history. Schema/program migration, multi-host operation, ownership transfer, dynamic subjects, fresh-supervisor child reassociation, cleanup custodians, providers and deployment are excluded. Grants are static and local transport provenance is trusted.

These limits agree with the durability contract, qualification and runbook; neither the review nor M1's correction broadens the implementation profile.

The [complete documentation fix report](reviews/final-fix-report.md) retains its exact diff and verification. Final review approval applies to the qualified source and the corrected evidence claims. Publication is a separate Git/PR action; it does not broaden this profile.
