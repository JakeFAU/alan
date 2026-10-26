# Evidence comparison correction

The copied Phase 1D.1 `checks.py` incorrectly demanded historical/current raw equality for composition and supervision, including their source manifests. The retained invocation failed at `('normal', 'COMPOSITION')`. Original tool bytes, complete pre-correction source, inputs, and failure are preserved under `docs/foundational/phase1e/evidence/tooling-correction/`; none was relabeled or modified.

Only `docs/foundational/phase1e/tools/checks.py` changed, with new reusable `comparison.py` and focused `test_comparison.py`. No Go source or parent qualification/report tools changed. `evidence/evidence-tools/correction.patch` preserves the exact tool diff; `retained-comparison.json` and `audit.log` record byte lengths and SHA-256 hashes of all three tool files.

## Verified distinction

For retained normal and race suites, all 15 QUALIFICATION payloads equal historical baseline payloads exactly and in order. All 20 COMPOSITION and 32 SUPERVISION records differ only in the source-manifest header and subsequent top-level SourceID. These fields are explicitly validated, recorded, and excluded only from the named behavioral comparison. Every remaining field, including bounds, alphabets, counts, traces, and control findings, remains exactly equal in order. Raw equality is explicitly false.

Both embedded manifests contain the same ordered 15 paths. `go.mod` and all model/search files are unchanged. The only changed path is `research/semantics/runtime_conformance_test.go`: historical SHA-256 `599001e7c6e911d3aa27aa243e5e295f1cce3bbbb6083507333f4ee907b1d7ba` (42278 bytes) becomes `f2f3588736b0e42ad013a3640a3a328c18ca9043454862bdf68c5b08ba418cc5` (42350 bytes). The two exact edits replace `rid` with `ex.ExternalKey(snapshot.Namespace, rid)` in request-key and receipt-key assertions. The production comparator verifies current file bytes, reconstructs those two historical assertions in memory, verifies their historical bytes, and recomputes both length-prefixed aggregates. Every record SourceID must match its corresponding manifest. Wrapper SourceID must match the actual suite source identity supplied by the caller.

All six CONFORMANCE tests retain identical test/fixture identities and ordered model streams, configurations, mappings, and limits. Across 81 checkpoints, all 63 declared public actual/expected projections match historical values, as do assertions, labels, and model-event counts wherever present. The other 18 checkpoint labels and field shapes match; their raw snapshot/resource/outcome contents are outside that projection comparison. The retained Go JSON contains passing events for all six tests; `passed=true` alone does not suffice. All six complete raw payloads differ in both modes, and no whole-trace or raw-runtime equivalence is claimed.

An additional independent decoded raw audit enumerates every difference, including nested JSON descriptors and base64 program bytes, for all 12 mode/case pairs. It found only namespace/key version additions, schema version 1→3, empty adapter/corpus tables, `Provision.Assets:null`, explicitly namespaced external keys retaining the same local intent, ephemeral loopback URLs, corresponding verified program content hashes, and process IDs. No changed behavioral value was found outside those disclosed differences. The full old/new values and paths are retained in `raw-conformance-differences.json`. This audit preserves differences rather than deleting fields and calling the result raw equality. Its classifications apply to the retained inputs; future production checks claim only the stated public projection domain, with Go assertions covering runtime contracts.

## Commands and results

Run from the repository root:

```
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s docs/foundational/phase1e/tools -p test_comparison.py -v
PYTHONDONTWRITEBYTECODE=1 python3 docs/foundational/phase1e/evidence/evidence-tools/audit_retained.py
```

Both exited 0. Seven tests pass, including subcases that reject changed state/transition counts, alphabet, counterexample trace, control finding, record removal/order, public projection (even when actual and expected are both changed), model stream, projection field presence, wrapper/record/manifest/file identities, and missing Go test pass evidence. The same comparator functions are used by `checks.py` and these controls. Positive controls accept the retained normal/race evidence with explicitly documented provenance changes.

The audit also successfully reran the production comparator against a temporary copy containing only current `go.mod` and semantic source files, with evidence records supplied in memory. No Git directory, historical checkout, or retained source archive was required by the comparator. It is suitable for the complete extracted archive. The tests themselves use bundled `tooling-correction` evidence. `checks.py` now executes that exact unittest discovery gate with the running Python interpreter and `PYTHONDONTWRITEBYTECODE=1`, recording its command, exit code and output alongside the static checks. The parent owns inclusion of its frozen fixtures in source/archive manifests. No Python cache files were generated.

Retrospective inputs remain source `c67d46284af60388c8ecd3b96ce0f1cb1613d3c3ad29ed6975ae00fde3318912`. These are not fresh Go tests of corrected tools. Full `checks.py` execution is deliberately left to the parent after freeze and fresh default/normal/race suites under the new source identity. No broad Go suite was rerun here. No unresolved behavior difference was identified within the audited retained evidence; the public-projection comparison is intentionally narrower than raw runtime equivalence.
