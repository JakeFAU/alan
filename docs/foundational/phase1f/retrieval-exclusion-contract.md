# Optional exact-span exclusions for evidence revision

This extension addresses a public data-contract obstacle, not a missing interpreter operation. The coordinator already receives exact e1/e2 source identities, but the legacy helper accepts only `question` and the legacy lookup accepts only `query`. Neither closed record can express “do not return these already-received spans.” The retained development-v2 request in [original-repeated-span.json](evidence/retrieval-extension/original-repeated-span.json) reacquired the initial header despite revised wording. The optional contracts below convey that explicit selection through ordinary authored messages and effect inputs.

## Public lookup profile

`lookup.InputSchema()` remains the legacy one-field query contract and retains `lookup/tokens/v1`. `lookup.ExcludingInputSchema()` adds eight required scalar fields to query:

| Fields, for N = 1 and 2 | Type and bound |
| --- | --- |
| `excludeN_document` | UTF-8 string, at most 512 bytes |
| `excludeN_version` | string, at most 64 bytes; nonempty values must be a lowercase 64-hex content SHA-256 |
| `excludeN_start` | integer, inclusive UTF-8 byte offset |
| `excludeN_end` | integer, exclusive UTF-8 byte offset |

The new profile is `lookup/tokens-excluding/v1`. Both profiles retain the same `lookup.OutputSchema()` and exact one-citation payload (`document`, `version`, `snippet`, `start`, `end`). The input schema IDs are:

- Legacy: `089537156ea44a0730d74a0900346c082f753006f682d15c1da26a28aa6cd246`.
- Excluding: `c0a9a080b5a2147db3d99e159b20236bdd48ea0ec988705a70ea5fb66a9791e9`.

Two slots are this profile's explicit operational bound, not an architecture-wide limit. A slot with both strings empty and both integers zero is unused. Otherwise it needs a nonempty document, well-formed version, and `0 <= start < end <= effects.MaxAssetBytes`. Partial tuples, malformed versions, negative/reversed/empty/oversize ranges are rejected. When the document and content version match a retained asset, the span must also lie within its bytes and both boundaries must be UTF-8 boundaries. Known-document tuples with a different well-formed version, and well-formed unknown document references, are ignored because they identify no retained asset. Their global range bounds still apply; boundaries cannot be checked against unavailable content. A valid subspan that is not exactly a candidate span excludes nothing.

`lookup.New()` serves both profiles. Every nonzero input declaration must pass immutable retained-schema validation and match exactly one recognized schema ID. The output declaration is likewise checked. The new fields are never accepted under the legacy or an absent input declaration. `Execute` rejects a prepared protocol inconsistent with the declared profile and handles invalid request data without indexing invalid spans.

Compatibility exception: the old public adapter allowed direct query-only requests with a zero-value `Request.Input`, and the existing direct-call test relies on that. That exact legacy convenience remains supported. It selects only the legacy query schema; it is not a way to opt into exclusions. Normal authored runtime effects always retain explicit schemas. Malformed nonzero declarations are rejected rather than treated as a legacy fallback.

## Selection semantics and authority

The operation scans the same retained bytes, tokenizes and scores exactly as before, and retains the same document-ID sorting, first-span tie break, 32-term/512-byte query limits and 2048-byte UTF-8-safe snippet cap. It merely makes a candidate ineligible when **all four** of document, content version, start and end equal a supplied nonzero tuple. It does not exclude an entire document, compare snippets approximately, change scores, or introduce a novelty bonus. Duplicate slots are harmless. Zero slots reproduce legacy results. If no positive-scoring eligible span remains, the result is `no_match`.

The adapter is a deterministic declared operation over input and retained assets. It does not know participant names, ancestry, round count or previous requests, and performs no store lookup, orchestration, retry, provider call or dynamic selection of authority. The registry still binds the existing `evidence-lookup` capability to the existing lookup adapter. Static grants and immutable input-schema identity still derive from `author.Application`.

## Optional public evidence helper

`evidence.ExcludingRequestSchema()` is the existing `question` field plus the same eight exclusion fields (nine fields total). `evidence.NewExcluding(name, caller, obligation, budgets)` returns the usual component (`Instances` plus `Port`) with that explicit request contract and the unchanged `evidence.ResultSchema()`.

`New` and `NewExcluding` call the same ordinary builder with different declared data profiles. The optional helper receives exclusions through its public caller port, forwards them through its ordinary retrieval-worker message, and includes them in the declared lookup effect. The model only selects the existing plan/query fields; it cannot rewrite the caller's exclusions. The planning and synthesis prompts from Task2B are byte-identical. The helper still makes at most two model requests and one lookup on its ordinary full path. Invalid exclusions that reach a lookup fail validation and flow through the existing invalid-result handling; no repair attempt is added.

Legacy `evidence.New`/`RequestSchema` remain unchanged for callers. Tests anchor the initial helper and retrieval artifact IDs to the actual pre-extension development-v2 store:

- `investigation/helper1`: `9c8ec36dacdccde46c960b0d4eef835d2a36cafa4b766c20635a537650a1ff66`.
- `investigation/helper1/retrieval`: `cfa4f77a2253555ed9eee8469744812ff42d4d65b5b3ae6ce21c1a6bc9cb4df2`.

An independent temporary Go module compiles and runs the optional helper through only public APIs, verifies that ordinary lookup inputs retain both exact tuples, and checks the independently expected next span and unchanged helper effect counts.

## Ordinary investigation revision

Initial helpers 1/2 still use legacy `evidence.New`. Statically provisioned revision helpers 3/4 use `NewExcluding`. Only after a valid review requests revision and the ordinary budget permits it, the coordinator builds two tuples from the e1/e2 public results. An `answer` slot supplies its document/version/start/end; any other disposition supplies an all-zero slot. Both revision requests receive that same initial set. Both sends still precede either receive.

These are fresh, explicitly authorized revision effects, not retransmission of an unknown effect. Initial evidence remains available to synthesis. No public investigation result fields, topology, review policy or ceiling changed: at most one revision, 13 model requests and four lookups (or seven/two when revision is disabled). No exclusion is inferred from child private state. Revision siblings may return the same new span, because they exclude only the initial set; this profile does not coordinate siblings' results or guarantee four distinct citations.

## Compatibility and qualification

Existing stores keep their compiled artifacts, schemas and prompt bytes. Recovery does not recompile, and the adapter continues to support valid legacy requests and protocol. Replay needs no adapter registry. There is no storage migration, compiler/runtime edit, opcode addition or semantic-version change. New optional helper/investigation artifacts receive new IDs as ordinary program bytes change. The original Phase 1E archive and both prior development comparisons remain untouched. The before-source identity is retained separately under [retrieval-extension](evidence/retrieval-extension/before-source-manifest.json).

The new tests establish the contract using retained and synthetic controlled inputs. The legacy query returns the original known span, and the new declared query plus initial exclusions returns the independently specified next eligible span. [ignored-exclusions-mutant.json](evidence/retrieval-extension/ignored-exclusions-mutant.json) retains a faulty result that ignores exclusions: it passes ordinary output-schema validation but fails the **same** independent input/output exclusion monitor used for corrected results. The rejection is an exact excluded-span match, not an unrelated guard. [ordinary-revision-records.json](evidence/retrieval-extension/ordinary-revision-records.json) retains successful same-query revisions and a partial-initial case, including effect inputs, pending identities, counts and public results.

Tests also cover zero/duplicate/two exclusions, exhausted candidates, unknown references, partial/malformed/range/UTF-8 tuples, retained schema and protocol mismatch, unchanged lexical term limit and truncated UTF-8 spans. Existing controlled example/CLI tests and frozen normal process controls run against the extension without changing their fixtures or monitors.

## Alternatives and limits

Prompt-only rewording was attempted in revision 2 and its outcomes remain reported. Increasing calls, retrying uncertain work, broadening lexical ranking, exposing private child state, or adding a runtime policy instruction would change the intervention. The selected optional flat data contract is narrower: callers explicitly identify at most two spans and the existing deterministic adapter filters exactly those candidates.

Filtering known spans does not guarantee relevance, completeness, diversity among revision siblings, semantic support or better model judgment. It may expose a less useful span or produce `no_match`; malformed references remain explicit failures. This is a development-informed extension before heldout unsealing. Parent's separately identified revision-3 measurement and independent grading determine usefulness; no live quality improvement is claimed from these controlled checks, and no heldout tuning or paid call was performed by the implementer.
