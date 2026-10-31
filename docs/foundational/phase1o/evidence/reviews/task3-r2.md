# Phase 1O Task 3 — independent scoped rereview R2

Spec: **APPROVE** (implementation adherence to the unchanged frozen finite contract, within Task 3's stated scope).

Implementation/certificate quality: **APPROVE**.

All three R1 findings are **ADDRESSED**. No new actionable defect was found in the scoped fix. R1 remains a preserved rejection of the earlier source, not a waived finding or a retroactive approval.

Reviewed `.superpowers/sdd/00-plan-phase1o/task3-r2-diff.txt`, SHA-256 `657f63f62542bab2b32a3a375a0a208cad5d9bdbf9609265ee2815cb0c05f8cd`, the appended R2 section of `task-3-report.md`, and `task3-r2-index.json`, SHA-256 `0073057241d94f15c51abd635da7970c622522eadbc69901c5481107702d12f6`. The package identifies base commit `ffd00be531fbb986b86160bd6c22a87345fee14a`. Source review was limited to the three-file fix diff and the already-reviewed R1 implementation; no broad audit or whole-phase verdict is implied.

## Findings resolved

1. **R1/P1 nil-grammar bypass — ADDRESSED.** `research/substitution/weak_refinement.go:69–78,271–319,379–387`. Structural indexing no longer accepts a rule-table argument or decides rule validity. Certificate and cycle validation explicitly run `checkWeakGrammar`, which rejects nil, malformed owner/private classifications, and every unresolved referenced rule. Certificate validation returns Input errors before relation/transfer/readiness assessment. An explicitly empty table remains legal for graphs without edges, while rank validation intentionally remains structural-only. `TestWeakRefinementRequiredGrammar` at `weak_refinement_test.go:312` covers each graph side, component/environment ownership, nil/empty/missing-rule tables, pairs-only witnesses, the explicit empty-alphabet case, cycle validation and the unaffected rank path. The omitted-grammar false acceptance is closed.

2. **R1/P2 overstrong unmatched `silent_demand` — ADDRESSED.** `research/substitution/weak_refinement.go:214–220` and `weak_refinement_normalization_test.go:266`. Unmatched silent reference steps still preserve selected current guards; they no longer receive a blanket future-demand-preservation obligation. Endpoint demand equality remains intact. `TestWeakRefinementSilentSemanticChoice` at `weak_refinement_test.go:393` accepts the declared choice-then-visible path with equal endpoint demand/stable current guards and rejects its changed-guard variant with exactly `silent_guard` on the choice. The normalization assertion now requires exactly eight Strong issues, all endpoint `demand` inequalities, for R_w; R_g still has none. The appended report correctly preserves the old 12-issue result as historical output and identifies its four `silent_demand` issues as the implementation's extra requirement. No frozen01/02/03 clause was weakened to obtain this result.

3. **R1/P2 `Direct` private-preservation bypass — ADDRESSED.** `research/substitution/weak_refinement.go:174–185,200–213`. Genuine candidate private edges and traversed genuine reference private edges now check Projection plus selected Guards/Demand independently of Direct. Explicit `private_guard` and `private_demand` issues retain side and edge identity. Allowing Direct to identify unchanged private work is sound here because it cannot skip these checks. `TestWeakRefinementPrivateSelectedFields` at `weak_refinement_test.go:436` separates guard-only/demand-only changes, candidate mutations, equal-endpoint mutations on both Direct-matched sides, and reference interiors outside R that mutate and restore the field. Unchanged private Direct matches and actual nonprivate matched local choice remain positive controls. Thus correcting finding 2 did not remove the genuine-private accounting constraint.

The fixes preserve the earlier division between weak witness acceptance, optional Strong issues, malformed input, progress certificates and rejected supplied witnesses. They add no search/nonexistence procedure, alter no relation state domain, and change no original K1/J/O semantics. I found no new breakage from moving private checks before the missing-component branch or from separating grammar validation from structural rank indexing.

## Evidence checked

I parsed all 11 R2 compressed records and independently verified their compressed hashes, argv and exits against the R2 index. Every current final-source hash matches that index. Normal/race and final static/module/format/vet records identify the same final source manifest. The frozen01/02/03 hashes remain unchanged. R1 review hash remains `f578c90c7b9a304c34d41e21cc135b19e2931bf6508d5bae49ebea8acac2b78d`; the prior index hash remains `434a4b87eb0913e197d158f728fb5e7479aa678a65265e30b7cedba0ea43e5e2`. The controller separately verified preservation of all 28 original compressed records; this rereview did not rerun that earlier historical-log audit.

- `task3-r2-red.json.gz` runs the four regression/normalization test groups against the exact R1 core hash `6ae6c988afbe5c694323f12c25fe0206214394078b03962aa63f1afda2d65902`: exit 1, 30 run events, 19 failures and 11 passes. Its named failing cases expose all three findings, rather than a compile failure.
- `task3-r2-green-regressions.json.gz` uses the same focused test selection: exit 0, 30 passes.
- Normal and race records use the exact anchored alternation of 26 selected top-level test names: each records exit 0, 26 top-level passes, 96 run/pass test+subtest events, and no failures/skips. Both stdout streams record R_w=8 and R_g=0, each with 16 pairs, 12 component and 24 bilateral environment obligations.
- Recorded `go vet ./research/substitution`, `go mod verify`, and `gofmt -d` on the seven owned files succeed. Vet/format output is empty; modules reports all verified.
- Recorded unfiltered `staticcheck ./research/substitution` remains exit 1 with the same six historical diagnostics: ST1005 at `callable_join.go:290,301,307` and `callable_join_local.go:65`; U1000 at `invocation_join_explorer.go:84` and `invocation_join_oracle.go:61`. Their current source identities remain those checked against HEAD in R1. No new-file diagnostic or suppression appears. Approval does not relabel this as a green whole-package staticcheck run.

Recorded verification uses the local offline toolchain and readonly-module environment, excluding ALAN_* and OPENAI_API_KEY. This rereview executed source/hash/record inspections only: no new Go test invocation, broad suite, runtime execution, rendering, code/test modification, commit or subagent was needed.

Final changed-source identities:

- `weak_refinement.go`: `ffbeb5c07d6ab66020ac9242e25935d75514b8e5248a19de501bd850fe84f8a1`.
- `weak_refinement_normalization_test.go`: `7c88f3a574a2f0f3af453932f3189d9847958e45731e928e2ef730133669a946`.
- `weak_refinement_test.go`: `6dfc27aaa4ca3592154891a1058f623eade4f5ed30a13b1863f9fae22113ac52`.

Approval covers the corrected supplied-certificate machinery and finite clause fixtures only. It does not establish arbitrary A_op/A_pub admissibility, full Account_D, full ARG-1, coherent retained-sidecar extension for arbitrary R, all-mode K1 progress, runtime correspondence, or NONE. General P_boundary ARG-2 remains OPEN. Manuscript/04 and whole-phase acceptance remain outside this review.
