# Independent Phase 1H Task 2 application review

Review target: `5721df236155f42f075fd33e5df937a2caf6db83..52392fa42628b6094c7e15c898cf6e7493a70d93` in `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.

**Spec Compliant: Issues. Task Quality: Needs fixes.** This is a Task 2 application/consumer review, not whole-branch or final process qualification approval.

## Required fix

### P2 — Retain the whole accepted frontier count before emitting its item intents

Location: `examples/adaptive/adaptive.go:90` (per-item increment), with admission loop at `:119` and preflight total at `:86`.

The normative contract at `docs/foundational/phase1h/01-adaptive-contract.md:29` defines `work_count` as items in application-admitted frontiers, separately from runtime admission charges. The reviewed requirement is whole-frontier application acceptance before partial item intent emission. Currently `increment("work_count")` occurs inside the loop immediately before each Invoke. A persisted state after the first Invoke therefore reports only a prefix even though the entire frontier passed validation/capacity checks and was accepted.

This is confirmed by actual existing evidence, without regenerating a run or guessing from source. In `docs/foundational/phase1h/evidence/task2/cli-direct-inspect.log.gz`, adaptive subject `919b9161a9bff8657b663435e25e4bf854f1c5a8f07b1bb329495bd8d6bd1c59`, generation 1, epoch 1, history sequence 502, successor PC 170:

- current frontier is `[7,10,11]` after the three round-one items;
- retained preflight `trial` is 6;
- `work_count` is 4;
- that transition emits the round-two first Invoke, operation `["919b9161a9bff8657b663435e25e4bf854f1c5a8f07b1bb329495bd8d6bd1c59",1,502,0]`;
- admitted-ID history is only `[1,2,3,7]` at this point.

The nested inspection has the same mismatch at sequence 502/PC 170 under subject `f07172639df332061b464d69a97a2021f9869d6a196d5754b392e1cc46e002f0`. Both compressed inspections' uncompressed hashes match their command records.

Final replies eventually count the full frontier correctly, including runtime rejection; this finding concerns the retained partial-admission state that Task 3 must inspect after interruption/closure. Persist the accepted total before the first item Invoke (the successful preflight already computes that total), remove per-item count increments, and add a focused assertion at a real partial-intent boundary. Keep runtime charges tied to actual admitted operations. Do not redefine the frozen contract to make prefix counts pass. If the history list deliberately records emitted items rather than accepted-frontier IDs, document that distinction or bring it into alignment as well.

## Open evidence gate

The final root default run at source manifest `11c3dabbdebc10baa189f84527d9cc9ac06d2482f0386c18efe69e0eec63ec19` is still pending: `root-default-final.command.json` did not exist at this review's last evidence check. Consequently the corrected full maximum-size matrix has no final passing command record yet. This is an unresolved acceptance condition, not an assertion that the pending run failed. It remains open even apart from the counter fix.

The first root default run failed at source `9b180f59...`; the failures and exact pre-delta test source are retained. The corrections address fixture Inputs and the missing `child_id` inventory entry, add a previous-decision witness, and improve transition labels; no production source or byte ceiling was changed. I verified current source hashes against the final manifest (zero mismatches). Earlier consumer/default/integration manifests differ from current manifested source only in `examples/adaptive/bytes_test.go`; their passing results must retain those older identities. Any production fix from this review needs fresh relevant evidence and regenerated artifact identities.

## Reviewed behavior and quality

- The production slice is confined to the ordinary application and public consumer. The reviewed range contains no runtime, author, schema, or internal package changes. Source constructs a single durable three-token ForEach round body; there is no host-side frontier construction or invoked round template.
- Input schemas retain eight seeds, finite membership validation for IDs 1..8 and limits 1..3/1..8, candidate/remaining capacity 16, and exact scalar leaf schema identity. Evidence narrows only the adaptive question to 512 bytes and accepts the unchanged helper's exact public schema IDs.
- Preflight scans the complete seed/candidate identity set and prior history before affected leaf invocation. Capacity uses equality before increment. Invokes precede ordered Await collection, and derivation follows the full round barrier. Failure/invalid-response/duplicate/exhaustion/work-limit/round-limit precedence is correct in source. Successful siblings still derive their children after another known failure. Unknown outcomes block the ordinary Await; Task 3 owns the executed failed-plus-unknown/recovery case.
- Derivation consumes actual retained response fields. Scalar 0/1/2 yields the required children and values; evidence disposition mapping and copied child questions match the contract. Child identity arithmetic is bounded by validated seeds and three rounds. Decisions clear the prior one-row list then Append one complete candidate/stop/completion record before continuation.
- Response presence is explicit and missing responses are not fabricated as zero. Evidence binds only disposition while retaining original full helper outcomes. The summary tests compare each summary operation to the original runtime outcome and check 8192-NUL answers. Complete-state inventory rejects additional full-response bindings/list copies.
- Default deployment limits retain 32 invocations/64 created runs/depth 3/96 effects; CLI capacity is restricted to 1 or 2. Public `New` accepts explicit ResourceLimits, enabling authentic denial tests. Direct/nested share the adaptive artifact and schemas. Actual scalar CLI snapshots contain ordered work `[1,2,3,7,10,11,31]`, root succeeded, and each result's operation/status/observation matches the original outcome.
- The separate consumer module reads the pre-implementation fixture ledger rather than importing application derivation. Fixtures were already present by `ef06eb2`, before this implementation range. Consumer checks exercise both nesting choices, capacities 1/2, and primary/paired observations. Parent/edge/round checks additionally exist in the application tests; the consumer's fixture assertions are narrower but collectively adequate for this slice.
- Root/wrapper forwarding preserves present payloads and branches succeeded versus generic nonsuccess through ordinary Success/Failure. The scoped plan clarification explicitly limits executed cases to succeeded/failed/rejected/absence/diagnostic, leaves individual child cancelled/abandoned untested, and assigns root operator closure to Task 3. That is a reasonable and honestly stated adjustment given the absent public lifecycle API. I do not require fabricated outcomes or a new API merely to claim those cases executed.
- CLI init/submit/run/inspect/replay/recover uses public APIs. Recover only opens/runs existing state. Request parsing requires limits/seeds, rejects duplicate/case-alias/unknown keys and malformed seed values, and delegates mode-specific schema validation to persisted intake. Loopback endpoint validation rejects nonliteral/nonloopback/non-HTTP endpoints and userinfo. Both transports refuse redirects; the scalar transport only submits/decodes one leaf capability request and does not orchestrate adaptive work. Credentials are not read from environment. Stdlib flag handling, returned errors, JSON output, and signal context are suitable for this bounded example.
- Authoring usability is accurately described: finite membership emits many equality checks, IDs require explicit pair scans, and nested constructor expressions are dense. README exposes limits, data shape, outcome inspection and executable CLI use. No claimed convenience API hides a host orchestrator.

## Byte and evidence assessment

`examples/adaptive/bytes_test.go:315-404` traverses every committed history successor as a complete Projection, including all cursors, checks each native Value and recursively each list array, checks each observation/intents serialization, reads actual compiled artifacts, and checks runtime operation/reason producer bounds. This addresses the Task 1 P2 objection to final-state-only or one-cursor probes. Inventory allowlists are explicit and complete for the source inspected.

The additional two-round witness requires a retained previous decision, current/next lists, round and cumulative evidence, handles, and two data-bearing cursor snapshots to coexist. Its command passes at the final source identity; reported adaptive peak 175587 bytes, root 71336, wrapper 71396, helper 102890, artifact 117091 are below unchanged envelopes. The original maximum-helper-fields witness reports helper 151321 bytes. These are actual reachable cases; they do not establish that all conservative independent inventory caps can be simultaneously saturated. The final corrected full matrix remains pending as noted above.

I checked 21 retained command records against log hashes (including decompressed inspection logs) and manifest digests: no mismatches. I also independently extracted direct/nested result-operation correspondence from actual CLI inspections and confirmed scalar maximum compiled artifact length 113908 bytes. The summaries, code assertions, and retained command outcomes are consistent within their stated tested scope. No broad suite or competing heavy check was rerun.

The tests-first chronology is a real process deviation: only constructor-red and selected later behavioral regressions are retained as pre-fix failures; the complete matrix did not predate implementation as the brief requested. The report acknowledges this precisely. It cannot be repaired by relabeling later tests. Preserve that limitation in final qualification rather than claim full original TDD compliance.

## Review disposition

Fix and qualify the retained whole-frontier counter, then close the pending source-bound root/byte gate. The remaining reviewed application behavior is consistent with the Task 2 contract. Process kill/recovery, causal faulty variants, root closure and full race/final qualification remain Task 3 work and are not approved by this review.
