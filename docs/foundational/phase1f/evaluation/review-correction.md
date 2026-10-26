# Independent Phase 1F correction rereview

Reviewed at **2026-10-26 13:57:49 UTC**. This is a scoped review of freeze revision 2 against the preserved revision 1. The original `review.md` remains unchanged. This report is outside the frozen hash set and discloses no heldout content, answers, source-to-case mappings or retrieval witnesses.

## Verdicts

**Specification verdict: PASS for revision 2 within the originally stated bounded exercise.** The blocking version-identity finding in the original review is resolved. Corpus manifest versions, all gold evidence-reference versions, protocol and rubric now agree with the existing public `effects.NewAsset` / `effects.Asset.Check` requirement that runtime version equals lowercase hexadecimal SHA-256 of exact content bytes. Editorial revision `1` is separately retained as metadata. No runtime weakening or citation rewrite is required.

**Fixture/oracle semantic quality verdict: PASS, unchanged from the original review.** Corpus and question bytes are unchanged; the gold has no semantic changes. The original nonblocking observations about coarse fact grouping and consistent interpretation of the public missing-decision stopping rule remain applicable. The original limitations on sample size, template transfer, retrieval layout, unequal compute and workflow-only sealing also remain applicable.

No blocking correction finding remains. This approval concerns the corrected fixtures and written procedure, not any application implementation, run result or actual unseal compliance.

## Independently checked correction scope

Read the correction report, the three public history artifacts, corrected public fixture files, corrected sealed materials, and the complete retained public/sealed original fixture. Compared directly against the retained original bytes rather than relying solely on the corrected verifier.

- Verified the original and corrected public freeze commitments, every listed public file hash and byte size, and the original and corrected sealed manifest/payload hashes and byte sizes.
- Verified `history/v1-public.tar.gz` contains exactly the 14 originally committed public files plus the original freeze manifest, with byte-for-byte equality to the retained originals. `history/v1-freeze-manifest.json` also exactly matches the original freeze manifest.
- All eight corpus files are byte-for-byte identical. Each corpus-manifest document differs only by replacing its runtime version with its existing content hash and adding `editorial_revision: "1"`; IDs, paths, hashes, byte sizes, line numbers and byte spans are unchanged. The manifest adds an explicit version-contract description.
- Development and heldout input payloads are each byte-for-byte identical to their originals.
- Both gold JSON structures are exactly identical after making only the expected `evidence[].version` substitution from `"1"` to the corresponding document content hash. Checked all **42 references** across both splits. Claims, evidence snippets, offsets, categories, acceptable modes, forbidden claims, aspect labels and stopping rules are unchanged.
- Reviewed protocol and rubric diffs: they clarify runtime versus editorial identity and document correction history; they introduce no answer, stopping or grading-semantic change beyond repairing citation version validity. The verifier adds runtime-version checks and preservation/semantic-equality checks. It still does not prove runtime adapter behavior or model query quality.
- The existing public asset-contract file retains the same SHA-256 observed during the original review. The original review file also exactly matches its retained copy.

The correction is justified by a public interface incompatibility, and its observable changes are confined to that defect and preservation metadata. No result-dependent or question-dependent tuning is present in the compared artifacts. The parent/custodian attest to pre-implementation/output timing and continued nondisclosure; this rereview did not inspect application output, new application code, credentials or network services and made no paid calls. No frozen file was edited.

## Verification result

Executed:

```text
python3 docs/foundational/phase1f/evaluation/verify.py --sealed
{"checked_cases": 8, "documents": 8, "freeze_identity": "afee3bc1b40c8f51bebf16ede379183f736136e43656bf90334d02690e6ce024", "lexically_reachable_lines": 25, "lines": 25, "ok": true, "sealed_checked": true}
```

Exit status: zero. Independent direct comparisons described above also passed. The verification checks eight cases, eight documents, 25 lines, exact source references, content-hash versions and unchanged semantic gold; it does not attest that an implementation/configuration freeze or unseal has occurred.

## Retained source identities

| Identity | SHA-256 |
| --- | --- |
| Original public freeze | `eb5546bf2495869dabfc70242af523d2a5fc1005ccb674cc729c12f6df741de9` |
| Corrected public freeze | `afee3bc1b40c8f51bebf16ede379183f736136e43656bf90334d02690e6ce024` |
| Original sealed aggregate | `11476bb1cce2cdd2c56873779df2a8e16f93602702aa10d84218c41bb161a4d8` |
| Corrected sealed aggregate | `600b2804b5e14ee8c7d5bf255408a612a68c71519237e0567ad30337a853fa33` |
| Retained original public archive | `804ea80dee24f4889dd1243b4a235a1c528c0cef0147e1d23adb9aa6f6a8a2f7` |
| Correction history JSON | `2e8b947f4e6bd7148e7885b907e3e99768a98abf74799ff235240aa10fa187e4` |
| Corrected corpus manifest | `0c1d4763cd254f29c131be1bdcf9666c6bba4aaa8ab3d1978f66e84b6275bac8` |
| Corrected development gold | `092d1569170da2fa91e23850ddde856bd3587f1881959e59c9d0467b1da6fa21` |
| Corrected sealed gold | `5ddce62c4bc796c000f93548f50ac46e93198127c498605ad6e7a82fafe7a437` |
| Existing asset contract | `1f8a8cbe17efdbbf662dcc7a5436878b2a699d61efecd4d4fe70623a07ae22c7` |
| Unchanged original review | `f4f421a80e95f554db7d5efdd307fee0a6ad41222cff0045b50750b82dfb8f82` |

Before unsealing, the parent must still freeze exact source, prompts, baseline definition, model/configuration and run limits; the independent custodian must check that freeze and publish the original revision-2 heldout bytes with retained identities and timestamps under the existing protocol. This review does not authorize bypassing those gates.
