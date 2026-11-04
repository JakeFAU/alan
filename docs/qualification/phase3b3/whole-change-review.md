# Phase 3B.3 independent whole-change review

2026-11-04. **Specification: APPROVE. Quality: APPROVE. Overall: APPROVE within the declared Phase 3B.3 scope.** No Critical or Important finding remains. One reproduced Minor diagnostic defect is retained below. This approves the bounded implementation, qualification, manuscript reconciliation and reproducibility package for reviewable delivery; it does not establish unrestricted model refinement, universal runtime progress, or authorize a merge or release.

The final execution and documentation gates have been inspected. This verdict was withheld until the final ordinary research command completed and the final documentation/evidence supplement arrived. Delivery commits and PR state remain parent-owned work after this review.

## Scope and identities

I read the user packet and frozen contract before assessing the implementation, then the supplied complete research/runtime diff packages, progress rulings, final results/runbook, previous rejected findings and correction reviews, raw qualification outputs, and the final completion supplement. Generated manifest bodies were checked structurally and by digest rather than treated as semantic source. Existing approvals were supporting evidence, not a substitute for this review.

| Boundary | Reviewed identity |
| --- | --- |
| Research base | `8197683c23a3ca774750fc3c2c389c5a64dd38b8` |
| Research executable | `7894bfae268e6dc0f5418bda3c6022f3a8a55540` |
| Runtime base | `af6943a5805a9a6c0e8fe77f804264f4765d07ec` |
| Runtime primitive | `4f6f5f2500a48b2ca01ee7b09a823fa63656b7ab` |
| Runtime executable | `759df6e3d2505f27e1287b2cfeb07bc4290afc0a` |
| Supplemental prospective integration | tree `bdbbae5f89f0c76d48618b3d8891dc40199ca4ed`, main `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315` plus task `c7fe6d87cdd49878558d676a301bec2d217ae3a6` |
| Final documentation | All 45 entries of `final-doc-completion-source.json`; zero hash mismatches |

The supplied review packages were `final-research-committed.diff`, `final-research-candidate.diff`, `final-runtime.diff`, and `final-completion-docs.diff`. I did not regenerate these diffs. I independently compared all 584 current research Go/module inputs and all 304 runtime inputs with their frozen manifests. An in-memory archive of the research base additionally confirmed all 563 prior Go/module inputs remain byte-identical. That named preservation check did not regenerate a review diff. Reads outside diff bodies resolved evidence provenance, final execution, legacy compatibility, manuscript rendering or the specific mutation-generator precondition.

## Strengths and substantive conclusions

The runtime change has a coherent small semantic boundary. Typed committed grant/revocation requests, explicit own/delegated selection and retained admission facts reuse the existing lifecycle, composition, dispatch and charge machinery. Complete permission tuples remain relational: the Cartesian controls exercise distinct physical actions and resources. A retained grant is bound to the exact holder/root, recipient, request, service operation and generation; data shaped like a grant or a prior successful result does not confer authority. An invalid selected delegated path does not silently fall back to available own authority.

The first permission/refusal and original attempt remain immutable. Fresh physical exposure revalidates the original request, selected authority, reservation/debit and current historical prefix under the existing locking/ownership boundary. Retained grants survive the qualified same-Store reopen while volatile physical admission does not. Exact-root revocation constrains later admissions without erasing earlier lawful opportunities or receipts. The public dispatch modes that lack the necessary authority gate remain refused for this profile. Legacy inert operation names and old reader behavior are treated separately from the new typed protocol.

The positive fixture proves the relevant tested behavior through actual custody, not a ready message or paid flag alone: both originals are committed; A services BA while its own AB result remains unpaid; B selects the authentic BA reply before servicing AB; each original result is selected by its requester. Actual receipts, replies, generations, operation IDs, endpoint wires and serving-root charges are joined. Real acknowledged process cuts preserve the original Store and complete database sidecar sets, and the unknown controls preserve unpaid originals without inventing results.

The research checker has useful independence from the runtime. It uses separate wire types, strict JSON/exact integers, a bounded five-instruction interpreter, original permission/root membership and historical fact-prefix checks. It does not invoke the runtime authorization helper or SQLite. The earlier route, immutable-source continuity and owner-epoch false acceptances were corrected with substantive tests; the original REJECT report and exact adverse inputs are preserved. The verified-byte reuse and recipient/holder naming corrections are present. A valid raw history, a supported projection, an unchanged-model verdict and an actual external contact remain separate conclusions.

The decisive counterexample remains visible: an authentic BA selection and accepted grant followed by ready selection produce runtime `authority_request_not_admitted`, while the unchanged LateGrant model permits Allowed/Delegated at the coarse admitted-request prefix. The ordinary positive instead selects ready before BA and services BA immediately. An expected-refusal test passing does not turn the former history into correspondence agreement. The change accurately reports a narrower runtime contract rather than amending the old model to hide this result.

## Findings

### Minor — validate the mutation generator's complete-parent precondition

Locations: `research/delegationcorrespondence/cmd/mutations/main.go:20–25` and `research/delegationcorrespondence/mutations.go:48` (also the analogous direct row indexing that follows).

The CLI accepts any `CheckRaw(...).Valid` input as sufficient to invoke `RawMutations`, but a valid incomplete history need not contain a lifecycle result. The generator's second mutation directly indexes `lifecycle_results[0]`. I reproduced the failure with the actual frozen cut-01 initial snapshot:

```text
/tmp/alan-phase3b3-research-frozen/correspondence-mutations \
  /tmp/alan-phase3b3-runtime-frozen/normal/cut-01-before-grant-3895259448/initial/inspect.json

Exit: 2
Stdout: empty
Stderr: panic: runtime error: index out of range [0] with length 0
        ... RawMutations .../mutations.go:48
        ... main.main .../cmd/mutations/main.go:25
```

The exact command/output record is `/tmp/phase3b3-final-review-mutations-initial.json`. This is a diagnostic/API-precondition defect, not a raw-policy false acceptance: the process exits with apparatus/input status 2, and all fifteen qualified mutations use the complete S2 positive parent. It does not invalidate those mutations, the raw checker, or the corpus verdicts. The final results/runbook now explicitly document this precondition and current panic behavior. Deferring a source correction preserves the frozen qualification and is acceptable for this delivery.

Recommendation for a later narrow change: check required grant/effect/receipt structure before generating mutations and return an explicit precondition error with status 2. Cover the raw-valid incomplete input with a focused regression test. Do not merely recover and discard arbitrary panics.

No other actionable finding was established. In particular, expected model RED, expected unqualified-correspondence exit 1, killed checker mutants, the inherited research lint diagnostics, and the retained export-provenance apparatus failure are not unexpected implementation failures.

## Verification and evidence actually inspected

I did not rerun broad Go, race or model suites. The following execution results are verified retained records from this task's frozen runs; my new execution comprised byte/digest checks, structured JSON comparisons, manuscript normalization, visual image inspection and the focused CLI probe above.

- **Research final gate:** `completion.json` reports stable source, 563 unchanged base inputs, byte-identical old lint and no unexpected status. I checked all 42 command records against expected exits and independently rehashed every command log. Fourteen expected nonzero commands are explicitly separated from passing assertions.
- **Research ordinary execution:** independently parsing `ordinary-module.log` gives 2,382 passing test/subtest events, 48 passing packages and zero failure events. The command was `go test -json -count=1 -timeout=60m ./...` with raw-corpus and exhaustive-progress environment variables enabled. The only named skips are `TestCommunicationArtifactExport`, `TestJoinArtifactExport` and `TestCandidateAExport`; fourteen other packages have no tests. The model/checker race record has 303 passing events, no named test skip and only five command packages without tests. Prior reciprocal and authority integration records pass in ordinary/race modes with 22 and 20 events respectively.
- **Research controls:** the four old model/audit exports retain byte-equal results; final source mutants fail actual assertions for duplicate keys, attempt binding, fresh revocation policy and causal predecessor checks. I inspected those failure messages rather than inferring a killed mutant from nonzero status alone. The scoped Staticcheck logs are clean; the six integration durable-test diagnostics are byte-identical to the pre-change baseline. Whole-research integration lint is not clean and is not claimed to be.
- **Raw correspondence:** each final normal/race report has 48 cases, 117 captured phases, 1,761 verified artifacts, 43 valid final histories and five hostile rejections. No raw-valid case has an external, continuity or projection apparatus error. The sole named mismatch is `immediate_selection_narrows_model_admission`. All fifteen generated raw mutations reject at substantive recorded edges; the actual recipient mutation first reaches original-source integrity, not a claimed later recipient-policy branch.
- **Finite models:** the final corpus has 22 cases/eight classes and retains the prior 33 histories/four ancestry controls. The LateGrant query has 111,065 states/840,616 arcs and 107,213 safe-completable states; the Revocable query has 220,564 states/1,740,832 arcs and 193,681 safe-completable states. Both exact predicate comparisons have zero mismatches. Maximum shortest safe suffixes are 23 and 22, with independently audited witnesses. These are finite existential reachability results.
- **Runtime freeze:** all eight command logs match their recorded hashes and exit 0. Ordinary execution records 1,265 test/subtest passes; core integration 1,597; focused authority race 105; each reciprocal ordinary/race run 53, covering 48 indexed cases. The current baseline has seven passing events and its one documented opt-in falsifier skip. Ordinary and integration runtime Staticcheck logs are empty.
- **Advanced runtime main:** I checked all original/corrected supplemental command log hashes. The initial archive-only reciprocal run failed because the exporter required a Git HEAD, and its failed outputs remain preserved. Corrected execution uses the identical prospective tree with honest scratch identity `f29e41b7532eaa0cbfd2b1f99f5171986a8d9a90`; it has 53 passing test events, zero failures/skips and 48 cases. Core integration, authority/mode race and integration Staticcheck are separately successful. No original branch was merged or rebased.
- **Archives and source custody:** I verified each listed member in the final 1,375-member research archive and 3,784-member runtime archive, along with the inspected development/rejected/review/manuscript/current-main/legacy packages. The final research archive SHA256 is `0b62b673702d3acd23997a2ae0f0bd20ae227902eee57e67c8f39c799ef3f75e`; its wrapper execution/source/summary/controller files equal the archived copies. Eight binaries are deliberately omitted with hashes and rebuild records. The source-guard preflight typo and the prior SQLite SHM custody incident are accurately classified separately from semantic failures and definitive regenerated evidence.

The archive-check output is `/tmp/phase3b3-final-review-archive-check.json`; the scoped normalization output is `/tmp/phase3b3-final-review-normalization.json`. Other targeted verification commands were read-only Python over supplied JSON, archive members or source bytes. No SQLite original or copy was opened during this review.

## Manuscript and reproduction

The changed mathematical content accurately reflects the final code and evidence: scoped one-hop authority, actual original-result custody, selective revocation, conservative unknowns, two existential finite progress predicates, the additional immediate-selection condition, checker limitations and the unsupported whole-Store chronological projection. It does not convert positive final service subsequences into a proof of chronological crash correspondence.

I ran `python3 docs/foundational/phase3b3/evidence/manuscript-check.py /tmp/phase3b3-final-review-normalization.json`; it passed. Abstract, Phase 3B.3, Conclusion and the changed Phase 3B.2 boundary paragraph normalize equally across Markdown and LaTeX. BibTeX is unchanged from the research base. I visually inspected final rendered pages 1, 2 and 35–39: the new material is readable without clipping or overlap. The final PDF is 39 pages, SHA256 `a85407e18f221dfed46a24c7389c2c83898d5513862815e56d0feaf6adfee925`. Existing TeX warnings at 215/559 remain separate; no new layout defect was found.

The runbook/evidence map distinguish executable and delivery identities, old reader construction, opt-in corpus/progress tests, explicit expected verdicts, relocated absolute paths, omitted rebuildable binaries, disposable database copies and the actual controller's historical paths. I checked local document links and the frozen contract digest. The final prose supplement and all 45 candidate hashes agree. Historical Markdown/TeX section-number differences outside the changed slice remain disclosed; this review does not certify global manuscript equivalence.

## Explicitly unproved or declined domains

These are intentional limits, not waived findings:

1. **Universal model/runtime refinement:** declined. Delayed selection is a concrete counterexample to unrestricted stuttering correspondence. Separate retained grant plus revocation, repeated admissions/contacts, applied-without-local-receipt and full shared-process crash/reopen chronology lack a proved direct mapping to the unchanged model. Unsupported is not agreement.
2. **Progress from every runtime prefix or every fair execution:** declined. The finite predicates characterize existence of safe completion paths in frozen graphs. First refusal, revocation, unknown outcomes and real ready/lifecycle/resource obligations can block runtime suffixes. Neither quiescence nor a finite runner limit proves scheduler deadlock or completion. Serial/serial dependency and receipt/reply waiting remain distinct.
3. **Physical exactly-once contact, hostile-host completeness or unique returning exposure:** declined. A gate is an admission opportunity. Some applied/unknown cuts have two contacts and one application under a trusted stable-key endpoint. Immutable adapter wires do not carry gate ordinals. Unsigned owner/endpoint context and Store bytes do not authenticate themselves or prove that all exposure was reported.
4. **Unobserved initial renewal:** declined for the two controls without pre-Open snapshots. Their `RenewalChecked=false` is correct; supplied Open/post-open/gate binding and later captured renewal are separate checks. Original request epoch and current owner/gate epoch must remain distinct.
5. **General authority algebra, programs or hostile histories:** declined. The runtime is exactly two initial logical principals with singleton complete tuples and one-hop request binding. The checker is the declared generation-one, static profile-13, five-instruction evidence domain. The two physical B action tuples map many-to-one only in the selected model abstraction; this is not a shared-counter or revocation-algebra isomorphism. Profile-14 process behavior, arbitrary programs, arbitrary generations and complete malicious-history detection are unqualified.
6. **Universal legacy equivalence:** declined. Ten genuine old-created formats 3–12, 80 primary API calls and 60 byte-equal paired exports are finite compatibility evidence. Formats 5/8 have no receipt-read execution in this fixture; 9–12 have no open runs at the capture. Old reader rejection of 13/14 is a separate tested boundary.
7. **Independent owners, multihop or N-agent closure:** declined as outside the task. These need additional ownership/fencing, transport, attenuation/revocation, cyclic-obligation and shared-resource models and counterexamples. Same-Store single-host SIGKILL evidence does not establish them.
8. **Global paper cleanup, unrelated lint or CLI hardening:** declined for this slice. The scoped paper reconciliation is complete; inherited structural differences and six old lint diagnostics are preserved. The sole mutation-CLI diagnostic improvement is documented and deferred as Minor, not represented as fixed.

The direct answers in the final results are appropriately conditional: the scoped extension demonstrates ordinary retained delegation and actual reciprocal service; the exact old model conditions do not establish unrestricted runtime safety/progress correspondence; the new protocol is a tested sufficient mechanism rather than a minimality theorem; broader ownership/chain/agent settings remain research obligations.

## Recommendation and final assessment

Proceed with narrowly scoped draft research/runtime delivery using the reviewed executable identities and final document manifest. Preserve the rejected candidates, exact mismatch, expected nonzero statuses, minor CLI probe and all limits above in the evidence and PR account. Do not describe expected-outcome success as model agreement, the raw checker as a proof, or this approval as merge/release status.

**Final specification verdict: APPROVE. Final quality verdict: APPROVE. Final overall verdict: APPROVE for the bounded Phase 3B.3 change.** No source fix or additional broad test run is required before that scoped delivery. Only this report and temporary read-only review/probe outputs were written; no source/index/HEAD/branch/archive was mutated, no subagent was spawned, and no merge or release was performed.
