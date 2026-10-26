# Task 5 independent process qualification review

Reviewed 2026-10-25 against BASE `54d67bc1aa004f32e0e051bcee9d3fc2271a1b03` and the seven-file Task 5 source manifest. All current scoped files matched both recorded SHA-256/size and frozen source copies. Review covered requirements, the authoring/effects contract, task brief and qualification handoff, implementer report, complete scoped source/diff, original failures, and normal/race evidence. No implementation files, runtime semantics, parent qualification tools, dependencies, or commits were changed by this reviewer.

**Specification verdict: changes requested for one false-identity qualification gap.** The retained actual C/D/E and H traces support their scoped behavior; no runtime contract violation was found. The claimed rejection of sequence substitution does not hold for the filters actually used by the crash controller.

**Code-quality verdict: changes requested for the same medium-severity harness defect.** The production change is only the existing observation-context hookup in CLI run/recover. No separate production-quality blocker was found in that delta. This is a scoped gate, not the final whole-branch review or archive/offline-reproduction gate.

## Finding T5-R1 — Medium / P2: actual crash acknowledgment path accepts a false sequence

Locations: `internal/evidencetest/process_test.go:376`, `:598-629`; `internal/evidencetest/control_test.go:69-83`.

The actual C/D/E barrier filters leave `Sequence` unspecified. `validateAcknowledgment` therefore only requires a positive sequence. `validateDurableBoundary` binds authorization, intent, subject, generation, and epoch, but never compares the acknowledgment sequence with the matching retained intent. The negative control uses a fully specified synthetic filter, so its passing sequence test does not exercise this gap.

Exact counterexample: take the actual retained C-model acknowledgment and stopped snapshot from `focused-normal.log`, with intent `["assistant",1,15,0]`, epoch 2, and sequence 15. Preserve the actual PID, token, scenario, store, subject, phase, intent and attempt; change only `Sequence` to 16. Both existing controller checks accept it. The independent overlay test prints `CONFIRMED: actual C-model matcher and durable check accept Sequence=16 while retained intent and original ack Sequence=15`. This test intentionally passes when the defect is reproduced; it is not a successful rejection test.

Expected: reject any mismatch between the event sequence and the durable matching intent/adapter attempt sequence. Add a control using the same wildcard filter shape as the real process case, followed by the same durable correlation check. Keep the earlier fully specified controls. Bind the value from retained source evidence; do not merely pin a fixture constant or relax the exact-identity claim.

Impact: a misreported sequence or acknowledgment from an incorrectly instrumented boundary can be accepted as exact operation evidence. This does **not** demonstrate a false boundary in the original successful traces: the independent audit compared the actual retained C/D/E events to their durable intent sequences and found them correct.

Preserved evidence is under `evidence/task-5/independent-review/`: complete `before-source/` and its manifest, exact `actual-c-model-evidence.json`, explicit original/filter/substituted input JSON, overlay source, original command/log, and `reproduce_sequence.py`. From the checkout root, run:

```sh
python3 .superpowers/sdd/00-plan/evidence/task-5/independent-review/reproduce_sequence.py
```

The reproducer overlays the preserved original files and adds the probe without modifying checkout source, so the counterexample remains reproducible after remediation.

## Verified evidence and independent probes

- C/D use actual built `alan-evidence` children. Boundaries originate around real adapter Execute and result persistence. Acknowledged result rows are present and unsettled before the matching child exits by SIGKILL. Source deletion precedes recovery. Original attempt descriptors, pending IDs, namespace, final answer/citation, and provider request identities are retained. D has a real lookup-finished acknowledgment and a zero old-lookup authorization acknowledgment on recovery, rather than inferring invocation absence from final row count alone.
- A separate provider process fsyncs accepted contacts before acknowledging them, retains wire digest/client identity, and does not receive a runtime store path. `responding` is correctly scoped to selected/fsync'd response before HTTP write; runtime receipt is separate evidence. E correlates acceptance to the original allowed reservation, leaves no result, and retains Unknown, one request/attempt and unchanged ordinary projections after reopen.
- Independent JSON audit of both retained normal/race logs checked the actual operation sequence against the durable intent, matching epoch/authorization, and cross-subject ordering through message admissions/observations: assistant model consumption at sequence 16 → retrieval send 26; retrieval message consumption 12 → lookup intent 15 → lookup consumption 16 → reply 17; assistant reply consumption 27 → synthesis intent 34. These are local sequences linked by actual message identities, not numeric comparisons between unrelated subjects. Final ordinary counters are model 2 and lookup 1; E projections and pending data remain unchanged.
- Offline replay invokes the real CLI with no credential, dead provider and deleted original source. The API/code path has no registry/dispatcher. Full snapshots remain equal and replay returns exactly the expected run/projection set. The report correctly labels zero invocations as a structural inference plus contact/store evidence, not a dynamic tool counter.
- A fresh independent run of `TestFaultyReplayDispatchIsDetected` passed: the real OpenAI adapter returned a validated candidate and the provider accepted one additional contact, increasing 2 to 3. The unchanged `monitorReplay` rejected it. The original sandbox loopback denial and authorized successful rerun are both retained.
- A fresh five-repeat `TestHeldModelOrdinaryProgressAndSharedCapacity` probe passed at both capacities. Provider acceptance precedes peer commit and caller consumption before release. At capacity 1 no competing lookup/fixture reservation exists at caller progress; at capacity 2 both fast completions are durable while the exact held model authorization remains active. The generic test owner is correctly distinguished from actual CLI application cases. The earlier false fast-completion ordering assumption and original source/log are preserved.
- Cancellation is explicitly external SIGTERM, not a claimed supervisory decision. Retained before/after evidence shows an actual received candidate drained under its original authorization, with no Unknown after recording/settlement. Late-closure evidence preserves original correlation, shows the result recorded with settlement absent, and later settlement without reviving the succeeded source. This was independently checked in both retained runs.
- Both retained race build records explicitly compile the actual CLI and provider/owner binaries with `-race`. Original missing-seam and harness-ordering failures remain preserved. Live remains NOT RUN; no live model, credential search, or external service was used.

Probe commands, complete raw logs, evidence-audit script/result, exact defect inputs and reproducer self-check are retained alongside this review. Broad default/tagged/race/vet suites and final source/dependency archive reproduction are parent-owned and were not needlessly repeated here. Passing finite local process tests do not establish power-loss durability, arbitrary fairness, or live model behavior.
