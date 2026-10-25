# Task 4 independent source and crash-harness review

Specification verdict: **PASS for Task 4**.

Quality verdict: **PASS for Task 4**. No new actionable defect found in the frozen slice. This is not whole-Phase1D approval: the selected independent model/runtime bridge and final qualification/archive remain Task 5 work.

## Reviewed source and scope

Read the task brief, acceptance matrix, amended implementation report, binding durability contract, and scoped review package. Independently verified all **37** frozen file byte lengths and SHA256 identities against current source; no mismatch. Reviewed the separate resource process/store, external authorization/transport/receipt/settlement implementation, replay extensions, automatic runner changes, production pause/report boundaries, complete controller, independent record checker, and all three fault controls. Historical research implementation was outside this delta.

Source authorization is retained before external attempt preparation; an explicit external grant remains a separate requirement. Attempt and original request identity persist before HTTP, with original authorization/epoch/key reused on idempotent recovery. Non-idempotent retained uncertainty does not trigger automatic retransmission. Late settlement binds the original execution and does not reopen a closed run. Receipt provenance is derived internally and replay checks its attempt/receipt/settlement/admission links. The external validator is actually called during reconstruction; it is not an unused helper. Replay follows read-only snapshot/reconstruction paths and has no HTTP dispatch call.

The resource owns its own SQLite connection/store and mutation transaction. Identical idempotent requests return the retained original receipt; changed full descriptors conflict. The endpoint's audit identity is not a fencing or lifecycle-policy mechanism. The report correctly limits its claims concerning non-idempotent effects, coherent tampering, malformed traffic contact accounting, and process-crash rather than hardware/power-loss durability.

## Harness and retained evidence

Independently inspected the two evidence directories named in the report:

- Normal: `/var/folders/qn/d3_l9fv5585g36qycsc804hh0000gn/T/alan-task4-evidence-3798343989`.
- Race: `/var/folders/qn/d3_l9fv5585g36qycsc804hh0000gn/T/alan-task4-evidence-1330739001`.

Each contains **87 JSON records: 61 process records and 26 scenario/input records**. In each set, all 16 retained killed-process records match their exact recorded acknowledgment (PID, start token, store, boundary, phase, subject, generation, epoch, sequence, intent, and authorization identity). All process binary SHA256 records match the retained binaries. `go version -m` independently confirms `-race=true` on the race executor, resource executable, and speculative-fault test executable, with the pinned SQLite dependency. Normal binaries do not claim race instrumentation.

The controller waits on a pipe, validates exact expected identity and committed database facts, then sends SIGKILL to that PID and requires an actual signal exit. A timeout cannot substitute for a reached boundary. The real staged boundary follows the transaction's writes and precedes COMMIT; post-commit/authorization/decision acknowledgments follow successful COMMIT. Runtime failpoints only report/pause. The speculative dispatcher is confined to a test binary, with an explicitly separate test-only trigger.

The second recovery crash retains inner generation 1, sequence 7, original operation and authorization while advancing epoch 2 to 3, with a different PID/start token. Endpoint and executor application/response acknowledgments have different stores/PIDs but matching operation, authorization, generation, epoch and sequence. The automatic nested checks require actual successor instructions and ordinary original-result consumption, not merely the existence of a successor row. The terminal case pauses at terminal commit before notification dispatch.

The original controller counterexample is retained under `docs/foundational/phase1d/counterexamples/task4-barrier-*`, including its pre-fix archive. Reviewed the original failure log: wrong generation 2/epoch 99/op-B/auth-B had been accepted against generation 1/epoch 2/op-A/auth-A. The current controller fixes that defect. Independently reran the focused actual-pipe mismatch regression: all generation, epoch, operation, attempt, old/new sequence and wrong-phase cases passed.

The unchanged record checker independently detects all three disclosed controls in both evidence sets:

1. Lost rejection: `retained lifecycle_results row changed or lost`. The original retained result is `not_retryable`; actual input/program failure makes the target eligible with the relevant generation/grant unchanged, and the faulty retransmission visibly accepts and creates generation 2. The correct control retains the rejection and a distinct new operation accepts. This is not masked by stale generation or missing rights.
2. Same-run policy reset: `projection disagrees with committed ordinary history`. The reset happens after validated Open; actual Run occurs, but the corrupted pending value prevents consumption. The report accurately does not claim a repeated target decision.
3. Speculative dispatch: `endpoint application lacks committed authorized intent`. A real endpoint application survives the staged executor rollback with no committed executor intent/authorization. The correct control has zero applications at that cut and one after recomputation.

The checker does not call Reduce/Replay or encode the lifecycle decision policy. It is a deliberately narrow retention/projection/reference oracle, not a replacement for the pending independent model bridge. Read-only replay checks compare retained executor semantic records and resource requests/applications/contact counts, including the unresolved post-application state.

Evidence audit output: `/tmp/alan-task4-review-evidence-audit.log`. Reviewed reported full-suite/race/vet logs rather than repeating the unchanged full suite.

## Additional independent adversarial probe

Probe source: `/tmp/alan-task4-review-probe/internal/executor/review_probe_test.go` (copied frozen packages; production tree unchanged).

Command: `GOCACHE=/tmp/alan-phase1d-go-build go test -count=1 -v ./internal/executor -run '^TestReviewExternalDeniedGrantAndReceiptTransportNamespace$'` in `/tmp/alan-task4-review-probe`.

Result: **PASS**, log `/tmp/alan-task4-review-probe.log`.

Exact input: ordinary client external effect to `http://127.0.0.1:1/apply`, profile `idempotent`, delta 5, original intent `["client",1,1,0]`; only an input permission is provisioned, with no external grant. Before dispatch, admit ordinary input with raw transport ID `identity("receipt-delivery", intentID)`, logical ID and operation equal to that intent, target client generation 1, obligation receipt, and payload `total=999, accepted=true`.

Verified source authorization is allowed while the separate external disposition is `grant_denied`, the ordinary input remains legal and retained without shadowing the receipt domain or forging its provenance, the ordinary program consumes the actual correlated denial receipt (total 0), replay leaves the entire inspection unchanged, and reopen plus repeated input/settlement retains the original results. The initial default-cache invocation failed due to sandbox cache access; the reported passing run used the authorized temporary build cache and no network transport.

No source edits, additional subagents, or full-suite reruns were used for this review.
