# Independent Phase 1F fixture review

Reviewed at 2026-10-26 13:50:14 UTC, before this reviewer inspected any new application code or application output. This report is outside the frozen hash set. It contains no heldout questions, answers, source-to-case mappings, or retrieval witnesses.

## Verdicts

**Specification verdict: BLOCKED by a public fixture/runtime identity conflict (finding 1).** Both splits contain four questions with two substantive aspects per question and the promised category coverage. Exact corpus identities, byte spans, separate inference inputs and gold, semantic stopping modes, one-revision scope, independent grading, and the unseal sequence are specified. Those design requirements are otherwise coherent, but exact asset loading and citation grading cannot both honor the existing public runtime contract. The published scope is deliberately narrow; this is not a production-readiness or broad-generalization evaluation.

**Fixture/oracle semantic quality verdict: PASS; executable fixture validity is blocked by finding 1.** I manually checked every required fact and stopping rule in both splits against the complete corpus. I found no unsupported gold answer, incorrect numeric condition, wrong rollback target, invalid reference, unjustified category. No frozen file was edited.

## Review and verification evidence

Read all eight corpus documents and their manifest, development inputs and gold, protocol, rubric, verifier, freeze manifest, freeze report, and all three sealed files. At the parent's request, also inspected only the existing public asset contract at `effects/effects.go:23-36`; no new application implementation was read. The review did not use credentials, network access, paid inference, application output, or new application implementation. No subagents were used.

Executed from the repository root:

```text
python3 docs/foundational/phase1f/evaluation/verify.py --sealed
{"checked_cases": 8, "documents": 8, "freeze_identity": "eb5546bf2495869dabfc70242af523d2a5fc1005ccb674cc729c12f6df741de9", "lexically_reachable_lines": 25, "lines": 25, "ok": true, "sealed_checked": true}
```

Exit status was zero. I also independently enumerated the fixture files and computed SHA-256 values; the declared file counts, payload byte sizes and digests matched. The verifier establishes exact references and canonical lexical reachability, not likely model queries or runtime adapter behavior.

| Reviewed identity | SHA-256 |
| --- | --- |
| Public freeze manifest | `eb5546bf2495869dabfc70242af523d2a5fc1005ccb674cc729c12f6df741de9` |
| Corpus manifest | `0ac8086f6f8ced3c98cc83dc89cd823da9c53de145fffa1bfbfc0e32ef5ffb98` |
| Development inputs | `0f149a4c10f2b1ee64a58bfdf1039bb1e4445c93d80510e0b503b0b9f8b810c3` |
| Development gold | `a45988cc4b9f02162c89d810347f124f166b0639627de7e8088a28e50168ac53` |
| Protocol | `6cf9403cd7db450c9690108492aceec33781095a0695a38129224720367814fd` |
| Rubric | `945c0c3994ee6c5b23669be07d371c6da75e190c062282c694fabd242c166da9` |
| Verifier | `564a1077d603ccd34d0dfdbb5414f839667ea836f7a978c2872e4aad03a47fba` |
| Sealed manifest | `11476bb1cce2cdd2c56873779df2a8e16f93602702aa10d84218c41bb161a4d8` |
| Sealed inputs | `57641425198d3a3b02c3d6e685701e51ea2b581fc78541bceccc752a69148dcb` |
| Sealed gold | `93b411749e37b1c7848a9cc3802875f96b0cd9b11555a0632d62e413c9d4793b` |

The corpus manifest and public freeze manifest transitively commit to every reviewed source document; their complete per-document hashes are retained in those manifests.

## Semantic findings

- Supported cases ask for facts directly present in their applicable source scope. Numeric operators remain strict, durations remain consecutive, and the gold preserves units and restoration targets.
- Ambiguity cases omit a selector required by the sources, with no source default. Allowing either a targeted clarification or a complete conditional answer is justified. The protocol explicitly permits clarification without the otherwise known second aspect; graders must apply the mode-dependent fact set.
- Missing-evidence cases are expressly limited to what the packet establishes. The gold correctly distinguishes an unrecorded decision from approval, rejection, or a claim that no decision exists anywhere. Known facts still require an answer.
- Contradiction cases contain incompatible reports for the same scope and window with explicit equal authority, no supersession, and no recorded resolution. The gold correctly requires attribution and withholds a unified verdict while retaining agreed facts. Canonical conflict references include both reports and their authority context.
- Gold references entail the listed factual content. They are sufficient references rather than an exclusive citation list, so valid shorter spans or genuinely entailing combinations remain acceptable. The request itself supplies evidence of an omitted selector; source spans supply the selector requirement.

## Comparison fairness and isolation

Subject to resolving finding 1, the protocol is adequate for a comparison of these two configured systems under their recorded budgets. Identical questions, sources, model configuration and evidence policy, predetermined order/budgets, no outcome-dependent retries, operational failures retained in the denominator, and separate actual compute reporting prevent an implicit equal-compute claim. It permits correct first answers and does not manufacture an investigation win or reward revision. Raw output retention, initial/final grades and a blinded grader when practical support inspection of any apparent gain.

The heldout workflow is adequate for the stated trust boundary if executed as written: freeze source/prompt/configuration and baseline identity before unsealing; an independent custodian checks the artifact and commitment; publish original bytes with timestamps/hashes; supply inputs to the runner while excluding gold from inference; treat later changes as exploratory. No implementation/configuration freeze artifact or actual unseal/run is attested by this review. The parent and custodian must still perform those gates. This reviewer will not disclose sealed content to implementers before that gate.

## Retained findings and limits

1. **Blocking specification/interface defect: frozen version identity cannot satisfy the public asset contract.** Every corpus-manifest document uses `version: "1"`; the gold references use the same version. Protocol requires loading the exact bytes with manifest ID/version, and the rubric requires emitted citation versions to match that manifest. However, existing `effects.NewAsset` sets `Version = hex(SHA256(Content))`, while `effects.Asset.Check` rejects any other version (`effects/effects.go:23-36`). Literal loading with version `"1"` is invalid; normal loading yields content-hash versions and then violates the frozen rubric. This is not a heldout-content issue, and `verify.py --sealed` cannot detect it because it checks the fixture against itself. Parent notification requests that the original custodian reconcile the fixture identity/citation contract before an evaluated run. Do not weaken runtime content identity or silently normalize citations to conceal the mismatch. Any correction must preserve the original freeze and record changed bytes, new commitments, the reason, and contamination impact. Exact existing-contract file SHA-256: `1f8a8cbe17efdbbf662dcc7a5436878b2a699d61efecd4d4fe70623a07ae22c7`.

2. **Low: fact groups are not uniformly atomic.** For example, public `dev-04` combines equal authority, no supersession, no recorded resolution and the resulting uncertainty in one required fact. A rollback trigger also bundles metric, operator, magnitude and duration. These groups are fully evidenced and do not justify a false full pass, but partial-fact recall is coarse and case denominators differ. Keep exact claim-level reasons alongside the frozen fact-group score and report case-level results. Do not present pooled fact recall as an equal-weight measure of every semantic obligation. No silent gold change is requested.
3. **Low: one public stopping-rule phrase needs consistent interpretation.** Public `dev-03` says that obtaining the missing signed decision is the stopping condition. Read with its allowed partial-answer mode and protocol, the valid result is to state the packet limit, supply known facts and identify the missing decision; it cannot require actually obtaining an external decision during this closed-corpus run. Record that interpretation before outputs are graded. This does not change any frozen answer or require a corpus correction.
4. **Limits already acknowledged by the protocol remain material.** Four heldout cases provide no statistical or domain-representativeness basis; related templates mainly test rule transfer. Readable corpus facts can suggest likely tasks, and workflow sealing is not enforced process or operating-system isolation. Explicit absence statements avoid testing open-corpus search completeness. Single-line lexical retrieval and heterogeneous line layout affect opportunity, so this is neither a controlled retrieval ablation nor an equal-compute comparison.
5. **Execution and grading remain unverified.** This review validates fixture coherence and the written procedure. It does not establish adapter behavior, model performance, pricing, latency, durability, recovery, replay, actual blindness, or future compliance. An independent grader must still atomize real outputs, check all extra claims, verify citation entailment, and seek a second independent reviewer for disputed judgments.

Custodian correction is requested through the parent for finding 1 before unsealing or running the evaluation. No frozen bytes were changed by this reviewer. Any correction must preserve old/new identities and document contamination impact under the existing protocol; this semantic review is not permission to silently replace the original freeze.
