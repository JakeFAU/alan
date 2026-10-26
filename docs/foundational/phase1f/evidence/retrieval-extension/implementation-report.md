# Task 2C public exact-span exclusion report

Status: **DONE**. Code/test ownership is frozen and released. All requested focused normal checks completed successfully; no further source edits are planned. Parent owns fresh independent review against the retained before-source and separately measured revision 3. No full race, paid call, heldout read, credential read, subagent, commit, branch change or runtime edit occurred.

## Before and frozen source identity

Before-source aggregate: `f61145ff7f8a78165901e49c73437179e854620dc7a7bb73bf1402f536c08a53` (the retained f61145 snapshot). Its archive is `docs/foundational/phase1f/evidence/retrieval-extension/before-source.tar.gz`, SHA-256 `9041aff36d4936455ea5a655fd2ff0148df520fd8c3bbb0d90ba47c192c4dc3e`. No prior source archive, development version, result or fixture was altered.

Task code/test manifest: `.superpowers/sdd/00-plan/task-2c-source-manifest.json`; canonical `files` array SHA-256: `ae62993b623060a6185097c81fa183c657e1e8bafd58d91b3878ac21a2c61f14`. The manifest separately lists documentation/evidence support files. The code/test manifest was frozen before waiting on the combined focused checks; it was verified unchanged while writing this report.

| Code/test file | SHA-256 |
| --- | --- |
| `adapters/lookup/lookup.go` | `64f2023ab7962a16fcf1290d4d82f1b7b97dba4e9f5cf38da4fef27f1fac4d7d` |
| `adapters/lookup/excluding_test.go` | `22487336a88bc61b3a5baf837752a9a202dc8be454c22533fd17a44356125789` |
| `adapters/lookup/testdata/repeated-span.json` | `96f996b61e983dbd664168ac9e1bc6e669e83a89ea00fb5c3b1e0de48aaeaa9d` |
| `examples/evidence/evidence.go` | `406230c17bcb9372492e589e8e3e38cbcd413e1abad151af4a73fcda2959fcd5` |
| `examples/evidence/excluding_test.go` | `1adb0c1c4700f32804b225d09fb8d033213c96abe18f2e392f693aa2ae022cb2` |
| `examples/evidence/testdata/excluding_consumer.go` | `f69c8857b131811f0ad3961a2800ec7b8261d1023b7cfa11a09bc8a59c4d6c9d` |
| `examples/investigation/investigation.go` | `8ac171e3a6ca1258843f60118c6b9d545cdab892426656eb4ad053e05e6a6fca` |
| `examples/investigation/excluding_test.go` | `8e78ddf6dd9699d5d8d1b58a50726d434983b40527886ceec5c9e9c5600c66e0` |

| Documentation/evidence support | SHA-256 |
| --- | --- |
| `docs/foundational/phase1f/retrieval-exclusion-contract.md` | `f59eb1afe1ffeaa1086bd5f9138ce02648b2db1613a12f69d515c00fb1ebbecf` |
| `docs/foundational/phase1f/authoring-obstacles.md` | `47474513c9fbb456635eb89fe66f44862f1923229dba4985a4a4c65dc2d3f33e` |
| `docs/foundational/phase1f/evidence/retrieval-extension/original-repeated-span.json` | `96f996b61e983dbd664168ac9e1bc6e669e83a89ea00fb5c3b1e0de48aaeaa9d` |
| `docs/foundational/phase1f/evidence/retrieval-extension/ignored-exclusions-mutant.json` | `50c457c390e9a5a3b5c14117c066ebb08eb3778d2ec577167b7f45bfde3c9574` |
| `docs/foundational/phase1f/evidence/retrieval-extension/ordinary-revision-records.json` | `91d5fe2d199a353fcb76e19cacc46ce30fd2b3243937f2392e7d3672ce86bf8c` |

The only production source changes are the lookup adapter, existing public evidence builder, and investigation builder. New tests/fixtures are additive. Existing evidence/investigation/CLI/process test sources and their retained counterexamples were not changed. Parent's pre-existing README and other untracked Phase1F work were not overwritten. All five Task2B prompt hashes were rechecked unchanged.

## Public API and operational semantics

- `lookup.InputSchema()` remains query-only; legacy protocol is `lookup/tokens/v1`.
- New `lookup.ExcludingInputSchema()` declares query plus `exclude1_document/version/start/end` and `exclude2_document/version/start/end`. It has a distinct immutable schema ID and `lookup/tokens-excluding/v1` protocol. The output remains `lookup.OutputSchema()`.
- Each all-zero slot means no exclusion. Nonzero slots require a nonempty document, lowercase 64-hex content-hash version, and `0 <= start < end <= effects.MaxAssetBytes`. For matching retained document/version pairs, ranges must fit the content and boundaries must preserve UTF-8. Partial, malformed, negative/reversed/empty/oversize and split-UTF-8 tuples fail validation without unsafe indexing. Unknown well-formed document/version references are ignored; known noncandidate subspans match nothing. Duplicate slots are harmless.
- Every nonzero declared input schema is checked for immutable content/ID consistency and must match one of the two explicit profiles. Output declarations are checked as well. Exclusions cannot be supplied under the legacy schema or with no declaration. Prepared protocol mismatches fail explicitly. The pre-existing zero-value `Request.Input` convenience is preserved **only** for the exact legacy query-only payload, as explicitly approved by parent. It is not a fallback for malformed nonzero declarations or the new profile.
- Selection is unchanged lexical scoring over the same bytes, same document sorting and tie break, same term/query/snippet limits. An exact document+version+start+end match merely makes that candidate ineligible. No implicit history, ancestry, role, corpus-specific policy, provider catalogue, extra dispatch or retry is consulted.
- New `evidence.ExcludingRequestSchema()` is question plus the eight tuple fields. New `evidence.NewExcluding(name, caller, obligation, Budgets)` returns the existing component shape. Both helper factories share one ordinary builder; optional inputs travel through ordinary caller Receive, retrieval Send/Receive and lookup Request. Model output cannot replace caller exclusions. Planner/synthesis prompts and all existing result/decision guards remain unchanged.
- Investigation helpers1/2 stay legacy; helpers3/4 statically use the optional factory. On an authorized revision the coordinator copies exact public e1/e2 answer identities into slots, otherwise uses zeros, then sends the same initial exclusion set to both revision siblings. Initial evidence remains retained. No private child state, raw opcodes or SQL are used. Public result schemas/topology, one-revision budget and max13 model/four lookup opportunity remain unchanged.
- The existing `evidence-lookup` registry capability serves both explicitly declared operations. No interpreter/compiler/storage/schema-limit/semantic-version change, migration or authority broadening was needed. New program bytes get new IDs; old stores retain/replay their artifacts and valid legacy requests keep their old behavior.

The full validation/compatibility contract and limits are in `docs/foundational/phase1f/retrieval-exclusion-contract.md`. The opening of `authoring-obstacles.md` now explicitly distinguishes the initial adequate authoring basis from this later optional public lookup/helper data-contract addition.

## Concrete obstacle, retained evidence and independent monitor

The original dev-v2 helper3 request is retained both as `adapters/lookup/testdata/repeated-span.json` and `evidence/retrieval-extension/original-repeated-span.json`, with its source inspection hash, original request/candidate and initial e1/e2 provenance. The actual legacy query returns the initial header span. The old closed input schema cannot express exclusions. With exactly those initial tuples under the new declared schema, the unchanged scoring returns the next eligible existing span (independently asserted document/offset values).

The same test-only `exclusionMonitor` is used for corrected positive results and a deliberate mutant that ignores exclusions. The mutant first passes ordinary output-schema validation; only then the input/output monitor rejects its exact excluded-span match. No unrelated guard masks the omission. Its actual input, schema identity, schema-valid output, rejection and corrected positive output are retained in `ignored-exclusions-mutant.json`. Two controlled ordinary revision records (both answers initially, or one non-answer initially) retain all lookup requests, IDs, counts and final public results in `ordinary-revision-records.json`.

An initial reporting script assumed each test2json output event held the full JSON marker and hit an assertion when events were split at 1024 bytes. It made no source/result edits. The corrected evidence extraction concatenated output events per test before decoding markers; the complete raw test log remains unchanged and is the source for the retained records.

## Red/green and checks

All Go commands used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1f-go-build GOPROXY=off GOSUMDB=off`. Providers were deterministic local adapters or existing controlled loopback services.

1. `go test ./adapters/lookup` first exited 1 on missing `lookup.ExcludingInputSchema`. After the profile implementation, `go test ./adapters/lookup` exited 0 (0.367s). The tests cover retained input/provenance, old-schema inability, zero/one/duplicate/two slots, all eligible spans excluded, unknown document/version, partial/malformed/range/Unicode cases, exact subspan matching, declared schemas/protocols, lexical term limit, and the unchanged UTF-8 truncation boundary.
2. `go test ./examples/evidence` first exited 1 on missing `evidence.ExcludingRequestSchema`. After the optional shared builder, `go test ./examples/evidence` exited 0 (1.129s). A standalone temporary module compiles and runs `NewExcluding`, checks ordinary effect inputs and expected third span, counts two model/one lookup effects, and replays through public APIs.
3. `go test ./examples/investigation -run TestRevisionExcludesExactlyPublicInitialSpans` exited 1 against the still-legacy revision wiring and failed both cases: repeated initial spans persisted, final disposition was `revision_limit` instead of answer (13 model/four lookup effects normally, 12/four with one initial no-match). After ordinary exclusion construction and optional-helper wiring, the same command exited 0 (3.501s). It asserts fresh distinct effect IDs for identical lexical queries, exact e1/e2 public tuples in both revision requests, zero slots for non-answers, new evidence, preserved initial evidence, counters/pending identities and replay.
4. Frozen legacy helper compatibility identities remained exact, from the actual original dev-v2 store:
   - `investigation/helper1`: `9c8ec36dacdccde46c960b0d4eef835d2a36cafa4b766c20635a537650a1ff66`.
   - `investigation/helper1/retrieval`: `cfa4f77a2253555ed9eee8469744812ff42d4d65b5b3ae6ce21c1a6bc9cb4df2`.
5. Final focused command:

```
go test -count=1 -tags integration -json ./adapters/lookup ./examples/evidence ./examples/investigation ./cmd/alan-evidence ./cmd/alan-investigate ./internal/investigationtest
```

Exit 0; all six packages passed; zero failed tests. This includes the existing frozen normal process interruption/unknown/replay controls, with no monitor or fixture adjustment to force a pass.

| Package | Elapsed | Result |
| --- | --- | --- |
| `adapters/lookup` | 0.284s | PASS |
| `cmd/alan-investigate` | 1.136s | PASS |
| `cmd/alan-evidence` | 1.96s | PASS |
| `examples/evidence` | 5.032s | PASS |
| `internal/investigationtest` | 8.701s | PASS |
| `examples/investigation` | 38.165s | PASS |

Raw log: `.superpowers/sdd/00-plan/task-2c-focused.go-test.jsonl`, SHA-256 `01cecbe5190e948c0d7666e87a1f24fbceb6f8e66a4c079b208abc9b62ef4d98`. New mutant and ordinary revision records use the existing `INVESTIGATION` marker. Source-only boundary audit found no edits to author/runtime/effects/schema/OpenAI/internal tracked source. No redundant full-race run was performed; that is the parent final-source gate.

## Limits and next boundary

The two-span bound is explicit operational scope. Filtering known spans does not ensure relevance/completeness, agreement across reports, better synthesis or good stopping. Revision siblings exclude the same initial set and may retrieve the same new span. Malformed references fail; unknown valid references exclude nothing; excluding every useful eligible span can yield no match. No additional model/lookup call repairs those outcomes automatically.

Both earlier development versions remain in their own result sets. This is a development-informed public data-model extension before heldout unsealing, not a demonstrated quality improvement. Parent's independent source review and separately identified revision-3 measurement/grading are required before making usefulness claims. No implementation/check is blocked; code ownership remains released.


## Public handoff evidence

This report is also retained at `docs/foundational/phase1f/evidence/retrieval-extension/implementation-report.md`, with `implementation-manifest.json` beside it. `red-results.json` records all three original red commands, exit codes, exact diagnostic excerpts and observed public repeated-result fields, plus each corresponding green command/exit. These are explicitly labeled transcriptions of observed tool responses, not fabricated full raw logs or reruns. Complete actual original live repeated input/provenance and complete actual mutant/positive records are retained separately as described above.

The final raw focused test stream is also published as `focused.go-test.jsonl.gz` (deterministic gzip); its decompressed SHA-256 equals the raw log hash above. No test source or production code changed during this evidence publication.
