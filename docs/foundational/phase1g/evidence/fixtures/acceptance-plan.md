# Phase 1G independent fixture expectations

This test-design record precedes the implementation. It is not execution evidence. The qualification may choose exact API spellings after each reviewed slice, but must retain these inputs/expected relations or document a change and its cost. All lifecycle cancellation below is explicitly operator/controller injected; it is not demonstrated supervision policy.

## Frozen deterministic input families

One retained direct collection artifact has a typed request list with maximum eight items. The item component doubles an integer through ordinary finite computation. IDs are stable application keys, not payload hashes or worker names. Inputs are prefixes of:

| Input position | Item ID | Input integer | Expected output integer |
| --- | --- | --- | --- |
| 0 | item-a | 2 | 4 |
| 1 | item-b | -3 | -6 |
| 2 | item-c | 2 | 4 |
| 3 | item-d | 0 | 0 |
| 4 | item-e | 11 | 22 |
| 5 | item-f | -1 | -2 |
| 6 | item-g | 9 | 18 |
| 7 | item-h | 4 | 8 |
| 8 | item-i | 100 | rejected as ninth item |

Run counts 0, 1, 3, 7, 8 and rejected 9 after store initialization. Same compiled bytes and initial topology for every count within a selected direct/nested profile. Expected output list preserves accepted input order. Prefix counts use the table directly; the independent monitor must not call Reduce, Replay or an author compiler to calculate values. The two input values of 2 create two distinct admitted item invocations/results, even if their output values agree. A separate two-record request repeats item-a with different payloads; reject the whole input before any children.

The nested profile's ordinary wrapper calls the **same collection template** and forwards its public result, producing root → wrapper → collection → item execution depth three. It does not inspect private child state. Record direct and nested public request/result schema hashes, common collection template ID, and wrapper-only differences. This is bounded depth qualification, not arbitrary recursive equivalence.

## Required event families and monitors

| Family | Required causal events | Independent observations |
| --- | --- | --- |
| Runtime work | init completed before input submission; acceptance Receive commit; ordinary Invoke commits; admission decisions; item results; root consumption | one canonical accepted list; unique operation per input ID; no ninth/duplicate-ID children; exact ordered ID/value relation |
| Physical capacity | more effectful items than slots; configured root capacity 1 or 2; transport enter/leave acknowledgments | max locally active transports respects both root and Run limit; queued logical work is reported separately |
| Reverse results | explicit release of later item's provider response before earlier item's response | later result durable while earlier pending; final output still input order |
| Nested saturation | held transport acknowledged; wrapper and collection child admission/pure progress acknowledged before release; required leaf transport completion after release | parents do not own physical slots; actual descendant completion, not mere creation |
| Held capacity two | first transport held; independent second transport applies/replies; first still acknowledged held | sibling result/commit before first release; no timing-based causal claim |
| Unknown outcome | provider applies and drops response; owner records unknown; eligible sibling completes | original operation remains unknown/spent across reopen; zero repeated contacts for it; aggregate cannot succeed |
| Accepted work crash | boundary acknowledgment after accepted-list commit, SIGKILL, new PID opens store | exact accepted list/artifacts/remaining allowance preserved; no input resubmission/planner |
| Partial completion crash | at least one result consumed/committed and another pending, acknowledged cut, SIGKILL/reopen | completed IDs/results remain; pending operation identity and root counters do not reset |
| Recorded-before-consumed crash | original child outcome retained, parent hasn't consumed it, acknowledged cut, SIGKILL/reopen | same result identity consumed once; no second invocation/budget debit |
| Late/duplicate | original result retransmitted after consumption while another item/invocation pending | same original admission/result; no new item's consumption or rebinding |
| Closure | cancel before/after invocation admission; cancel before/after result recording; source authorization before cancel but reservation afterward | precise rejected/admitted children and charges; existing uncertainty remains; late evidence never reopens root |
| Replay | close process; remove credentials/adapter registry; deny network; replay retained store | unchanged durable ledgers/provider contact counts; no creation, replanning or dispatch |

Use existing boundary acknowledgment pipes and existing controlled provider/accounting patterns. Deadline/watchdog failures detect a hung test; elapsed sleeps never establish which durable boundary was interrupted. Report every actual killed PID, exit, token, scope/item/operation/attempt identity and inspection. Adapter return/slot release does not prove remote work stopped.

## Negative controls

Test-only faults must use the corrected positive monitors unchanged and reach the affected boundary. Retain exact original fixture and smallest observed failing trace. Candidate controls: payload-based coalescing of item-a/item-c; later item's response assigned to earlier handle; recovered root charge reset; missing child origin with otherwise valid local projections; creation before source intent commitment; published success with a required unknown child; replay that invokes creation/dispatch; a waiting parent incorrectly retaining the sole physical slot. A negative control that is stopped by an unrelated schema/closure check does not qualify its intended monitor.

At least one incorrect ordinary collector should produce a wrong item mapping while runtime identity guards stay intact; this distinguishes a public contract oracle from hidden runtime policy. Ledger corruption tests separately establish explicit rejection rather than automatic repair.

## Effectful evidence leaf

Retain the existing evidence helper's two-program policy and use it as the leaf template. Controlled Responses protocol and immutable lookup corpus supply reproducible outcomes, not a live quality claim. Record both model-stage identities, retrieval identity, public citation fields, invocation scope and root charges. Initial/new invocation allowance is not a reset of the root's shared allowance. Preserve the diagnostic `invalid_result` response when the helper then commits Failure. Do not edit Phase 1F prompts or graded answers.

## Measurements and report language

Record source/app artifact identities, author-facing LOC, template and initial run counts, list count, created invocation/run/intent/result records, maximum observed local active transports, selected wall-clock latency and restart outcome. Distinguish constant catalogue/provisioning cost from work-proportional retained records. Pass events, sample traces and bounded schedules are not independent proof counts. No new provider spending or reuse of the previous study's ledger is authorized by this plan.
