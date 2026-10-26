# Task2C independent review

Review date: 2026-10-26. Scope: the optional public exact-span lookup exclusion profile, shared evidence helper builder, ordinary investigation revision wiring, and their retained qualification evidence. The binding requirements were `.superpowers/sdd/00-plan/task-2c-brief.md`. This is a scoped review, not the separate whole-branch review.

## Verdicts

**Spec compliance: PASS. Code quality: PASS. No blocking findings.** No production changes are requested for this slice. The reviewed change implements a bounded public data-contract extension without a new interpreter capability. Final-source full-race qualification and separately identified live evaluation remain parent gates; this review establishes neither of those outcomes.

## Exact reviewed source

HEAD remains `e2157a8fb2837ea3d7d0232683072182a0ee832b`; Task2C is uncommitted working-tree content. Its comparison base is the preserved full-source identity `f61145ff7f8a78165901e49c73437179e854620dc7a7bb73bf1402f536c08a53`. The before-source archive SHA-256 independently matches `9041aff36d4936455ea5a655fd2ff0148df520fd8c3bbb0d90ba47c192c4dc3e`.

The supplied review diff SHA-256 is `2cd4edcd47ef62c46045fe55b1f40e7639f74c1e1418d11a6d0d361a102426e7`. All eight current code/test files match the frozen implementation manifest, whose canonical files-array SHA-256 independently recomputes to `ae62993b623060a6185097c81fa183c657e1e8bafd58d91b3878ac21a2c61f14`. This is the slice identity, not a claim that HEAD contains it or that it is a newly computed whole-tree identity.

| File | Verified SHA-256 |
| --- | --- |
| `adapters/lookup/lookup.go` | `64f2023ab7962a16fcf1290d4d82f1b7b97dba4e9f5cf38da4fef27f1fac4d7d` |
| `adapters/lookup/excluding_test.go` | `22487336a88bc61b3a5baf837752a9a202dc8be454c22533fd17a44356125789` |
| `adapters/lookup/testdata/repeated-span.json` | `96f996b61e983dbd664168ac9e1bc6e669e83a89ea00fb5c3b1e0de48aaeaa9d` |
| `examples/evidence/evidence.go` | `406230c17bcb9372492e589e8e3e38cbcd413e1abad151af4a73fcda2959fcd5` |
| `examples/evidence/excluding_test.go` | `1adb0c1c4700f32804b225d09fb8d033213c96abe18f2e392f693aa2ae022cb2` |
| `examples/evidence/testdata/excluding_consumer.go` | `f69c8857b131811f0ad3961a2800ec7b8261d1023b7cfa11a09bc8a59c4d6c9d` |
| `examples/investigation/investigation.go` | `8ac171e3a6ca1258843f60118c6b9d545cdab892426656eb4ad053e05e6a6fca` |
| `examples/investigation/excluding_test.go` | `8e78ddf6dd9699d5d8d1b58a50726d434983b40527886ceec5c9e9c5600c66e0` |

All eight support-file hashes in the implementation manifest also match, including contract documentation, original input, mutant, revision records, red-result transcription, implementation report and compressed focused log. Against the preserved manifest, 130 prior Go/module/process-fixture files were checked: only the three intended production files above changed. The existing process tests and retained counterexample fixture were unchanged. The five helper/investigation prompt constants were compared directly to the archive and are byte-identical.

## Spec findings

- **Explicit immutable profiles and compatibility — satisfied.** `adapters/lookup/lookup.go:20` preserves the query-only schema; line 25 adds exactly two typed tuple slots. Lines 54–90 check retained-schema content/ID consistency before selecting either recognized profile and check the output declaration. Lines 138–147 revalidate execution input and reject protocol/profile mismatches. The narrowly documented zero-value input exception accepts only the original closed query-only payload. It cannot activate exclusions. Invalid nonzero declarations now fail explicitly, as required by the validation contract. `examples/evidence/excluding_test.go:51` anchors legacy helper/retrieval artifact IDs to the retained development artifacts.
- **Exact filtering and safe tuple bounds — satisfied.** `adapters/lookup/lookup.go:92` accepts only all-zero unused slots or complete valid identities/ranges. Asset validation establishes content hash and UTF-8 validity; `0 <= start < end <= len(content)` is established before boundary indexing at lines 115–116. Unknown valid references are ignored, with global bounds still enforced. `isExcluded` at line 126 compares all four identity components. The only candidate-selection delta at line 188 adds ineligibility for an exact tuple; lexical scores, ordering, tie breaking, token/query/snippet bounds and corpus bytes are not changed. Existing one-citation output remains unchanged.
- **Ordinary public helper propagation — satisfied.** `examples/evidence/evidence.go:24`, line 57 and lines 66–101 provide the optional public schema/factory through the shared builder. Caller exclusions pass through ordinary Receive/Send/Receive/Request fields. The planner still receives the unchanged question/instructions record and cannot overwrite caller exclusions. Legacy factory behavior uses the same original schema and artifact identity. The standalone external consumer at `examples/evidence/testdata/excluding_consumer.go:42` uses public APIs, observes exact effect fields, obtains the third eligible span, checks two model/one lookup requests and replays.
- **Bounded authorized revision — satisfied.** `examples/investigation/investigation.go:81` retains legacy initial helpers and selects the optional factory only for helpers 3/4. Lines 254–288 construct exclusions solely from public e1/e2 answer fields; non-answer slots are zeroed, and both revision requests receive the same initial set. Existing review/revision-budget control at lines 320–335 remains unchanged. The controlled test at `examples/investigation/excluding_test.go:20` checks exact public tuples, fresh effect IDs, durable pending identities/counters, preserved initial evidence, unseen eligible evidence and replay, including an initial non-answer case. Counts remain 13 model/four lookup requests on the full path, 12/four in the partial case, with one revision.
- **Authority and compatibility boundary — satisfied in this slice.** The registry remains the existing lookup provider (`examples/investigation/store.go:279`); shared `author.Application.Provision` still derives static wiring/effect grants (`author/application.go:17`). No runtime, compiler, storage, scheduler, semantic-version or broader delegation source change was found in the source-boundary comparison. No child-private-state read or new orchestration mechanism appears in the production diff.

## Quality and independent evidence

The implementation has a small, shared profile-selection/validation path and a direct exact-tuple predicate. It does not duplicate the evidence helper program or add a history-dependent adapter policy. No correctness or maintainability issue requiring revision was found.

`adapters/lookup/excluding_test.go:25` first runs the retained legacy input, proves the old schema cannot express the new fields, and asserts an independently specified next span. The same independent `exclusionMonitor` at line 270 is called on corrected positives and a result deliberately ignoring exclusions. That mutant passes ordinary output-schema validation before failing for the exact excluded tuple; no unrelated guard masks it. The preserved mutant input/output/positive record exactly matches the corresponding raw test-log event. Both retained ordinary revision records likewise match actual raw events.

The original repeated-span fixture's source inspection hash independently matches `a41e07b13d1a188fb9b4af81773643a3894c2c8b7f3b105da79179e42161fba1`. Its legacy request, repeated candidate and both initial result objects occur unchanged in that development-v2 inspection. It is concrete retained live provenance, not a newly invented reproduction input. The lookup tests also cover zero/duplicate/two exclusions, all candidates excluded, unknown references, partial/malformed/range/UTF-8 cases, exact subspans, declared schema/protocol validation and unchanged term/truncation limits.

The existing compressed focused log was parsed independently. Its decompressed SHA-256 is `01cecbe5190e948c0d7666e87a1f24fbceb6f8e66a4c079b208abc9b62ef4d98`. It contains six package passes and zero failure events for the recorded integration-tagged command covering lookup, evidence, investigation, both CLIs and `internal/investigationtest`. The normal process controls `TestRetainedCounterexamples`, `TestReviewAndRevisionInterruption`, `TestPartialSiblingUnknownRecovery`, and `TestCompleteAndFaultyReplayDispatch` passed. The provider child entry is skipped at the top level as a subprocess helper.

## Caveats and remaining gates

The original red failures are retained as explicitly labeled transcriptions in `red-results.json`, not complete independently authenticated raw red streams. The successful focused stream and concrete live/mutant inputs are retained in full. This limitation does not undermine the directly inspected contract or positive/negative monitor evidence, but the red transcription should not be described as a raw log.

No suites were rerun: the existing passing focused log answered the qualification questions, and inspection left no specific unanswered risk requiring an additional probe. This review performed no paid calls, credential/heldout/gold access, subagent delegation, commits or product edits. It writes only this report. It does not independently certify preservation of every historical generated artifact; the relevant prior source/process fixture, archive hash and original live provenance were verified.

Filtering two known spans cannot guarantee relevance, completeness, better synthesis or distinct results across revision siblings. Unknown valid references intentionally exclude nothing. Final-source full-race qualification, whole-branch review and separately identified revision-3 evaluation remain outside this scoped verdict. There are no Task2C blockers to those next steps.
