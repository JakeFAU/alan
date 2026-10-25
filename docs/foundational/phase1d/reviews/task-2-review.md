# Task 2 independent specification and quality review

Reviewed 2026-10-25 against `task-2-review-package.diff`, the task brief/report, binding durability contract, and retained Phase 1 namespace findings. All 16 source/dependency SHA-256 values match the frozen package. Read the complete new runtime/CLI/test files and supporting store/replay delta. No repository source changes, subagents, or duplicate full-suite runs. Targeted adversarial probes were run against an isolated copy of the frozen source.

**Specification verdict: FAIL pending R1.**

**Quality verdict: CHANGES REQUESTED for R1.** One scoped correctness finding remains. No other blocking finding was identified in this review. The explicitly deferred real-Retry generation-binding test is not counted as a Task 2 defect.

## R1 — Separate ordinary-input transport identity from generated protocol transports (P2)

Locations: `internal/executor/messages.go:359` (raw transport lookup), the admission insert later in `admit`, `loadAdmission`, and `internal/executor/runner.go:177` / `:202` (automatic message/terminal transport keys).

The observation namespace is tagged, but the transport/admission namespace is not: `Input.TransportID` goes directly into the same `admissions.id` key space used by automatic message and terminal delivery tuples. A legitimate ordinary input can therefore occupy a generated delivery key before that delivery exists. Its immutable descriptor necessarily differs from the eventual message/terminal descriptor, so automatic delivery returns ErrConflict and stops the runner. The retained row survives recovery, so restarting repeats the failure. Even a rejected ordinary input can cause this.

This repeats the substantive Phase 1 retained finding where environmental ingress could shadow a generated Send admission. It is not an unauthorized-provenance escalation, but it breaks automatic progress and typed identity isolation with valid input and static grants. There is no contract reserving these ordinary transport strings; banning the observed string/prefix would discard a retained valid-input class rather than fix the identity boundary.

### Exact legitimate input A: accepted input shadows a message

Provision ordinary sender/1 and receiver/1. Sender emits one generic message to receiver/1, obligation `work`, payload `number=42`, then ends. Receiver awaits that exact authenticated ordinary message. Include the matching message grant and a normal local-input grant to receiver.

Before running either program, admit:

```text
TransportID = ["message-delivery","[\"sender\",1,1,0]"]
LogicalID   = ordinary-input
Target      = receiver
Generation  = 0
Payload     = {"number": Int(7)}
```

This is accepted as an ordinary local-input observation, with no forged runtime metadata. The runner commits sender's Emit, then fails after Steps=1 / Dispatches=0 with `identity conflict`; the intended message remains pending and both runs remain open. Read-only Replay succeeds. After Close/Open, the next Run fails with the same conflict.

### Exact legitimate input B: rejected input shadows a terminal notification

Provision child/1 with ordinary End(failed), observer/1 awaiting its public runtime-terminal outcome, and the explicit observation route `(child, observer, terminal, revision=1, obligation=public)`. Before running, admit:

```text
TransportID = ["terminal-delivery","child",1,1,"observer","public"]
LogicalID   = rejected-input
Target      = missing
Generation  = 0
Payload     = {}
```

The input obtains the normal retained `target_missing` rejection. Run commits child's terminal outcome and then fails after Steps=1 / Dispatches=0 with `identity conflict`; the frozen terminal notification remains pending and observer stays open. No input permission is required for this negative disposition, and no malformed descriptor, external SQL, manufactured generation, or private insertion primitive is involved.

### Reproduction evidence

- Isolated frozen source/probes: `/tmp/alan-task2-review-probe/internal/executor/review_probe_test.go`.
- Output: `/tmp/alan-task2-review-probe.log`.
- Command in the isolated copy: `GOCACHE=/tmp/alan-phase1d-go-build go test -run 'TestReview.*Shadow' -v ./internal/executor`.
- Both probes intentionally fail at the expected-progress assertion after logging exact raw IDs, admission results, and unfinished runner reports. Input A additionally proves persistence through actual Open; no test SQL modifies execution state.

### Requested correction

Make the durable admission key explicitly include the trusted transport-origin domain (at least input versus message versus terminal), independently of the caller's raw transport string. Preserve the raw ID in descriptors/results where useful, with exact descriptor conflict checks within the same domain. Update ACK references, inspection, and replay validation consistently so the discriminator cannot be supplied by ordinary input or lost during recovery. Keep the exact original strings legal.

Add regressions retaining both inputs above: ordinary accepted/rejected results remain immutable; generated delivery succeeds; observers consume only the intended authenticated observation; ACK/retransmission and replay retain their original typed binding. Include an ordinary message and terminal delivery sharing an identical raw caller-chosen transport string, to ensure this is a complete protocol-domain fix rather than an input-prefix special case. Preserve both original probes and their failure output in qualification.

## Positive findings

- Runner selection is ordinary and deterministic within an invocation, skips blocked receives, and commits through the reviewed Store transaction. The same pure reducer determines receive eligibility; history retains the selected observation. The scheduling cursor is correctly documented as ephemeral, with no unbounded fairness claim.
- Step/dispatch exhaustion is explicit, with unfinished supported deliveries and unsupported intents visible. Quiescence is not mislabeled completion.
- Input cannot choose Source/Kind/Provenance; CLI unknown origin fields are rejected before live Open. Message origin derives from the committed intent and terminal origin from frozen outcome/routes. Terminal payload exposes the public outcome only.
- Source authorization is separately retained and survives closure. New authorization after closure is retained as denial. Target route permission remains separate from source authorization. Ordinary dispatch does not use a role/name/generation shortcut.
- Accepted message admission and observation insertion share a transaction. Retransmission compares immutable descriptors and returns the original accepted/rejected disposition. Tagged logical observation IDs prevent the same logical input or message from being consumed again under another transport ID.
- Message/terminal admission and acknowledgment are separate transactions; ACKs require retained evidence. Frozen terminal obligations remain dispatchable after source closure.
- Inspect opens read-only/query-only and reads one coherent snapshot without policy reduction; Replay remains the separate stronger reconstruction path. The expanded messaging validation relates accepted observations to their admission/origin evidence.
- Task 1 output-bound enforcement and core SQLite/exclusivity/crash boundaries remain intact in this task's delta.

## Continuing acceptance boundaries

Task 3 must still demonstrate retained original service binding after real accepted Retry, without manufactured SQL generations; implement lifecycle-derived closure/successor reconstruction, retained lifecycle rejection, and source-denial result consumption. Those omissions are disclosed cross-slice work rather than additional Task 2 findings. Endpoint uncertainty, full process-kill qualification, independent policy/research conformance, and final source archive review remain later tasks.
