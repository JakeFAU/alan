# Independent final whole-branch Phase 1F review

Review date: 2026-10-26. Reviewer: independent final-review agent. Reviewed against current base `4bd58ce73e1036c490b0b622609b90d92fa58030`, the supplied whole-branch brief/diff, the public contracts, and retained executable and evaluation evidence. This is a fresh whole-branch review, not acceptance by inheritance of task approvals.

## Verdict

**Specification compliance: PASS for draft-PR readiness, within the stated bounded scope.** The application is authored through ordinary public components and supports standalone/nested use, evidence-dependent review and one explicit revision. The prior live gate is closed with retained actual outcomes. Evaluation, interruption/unknown/replay qualification, and a usable runbook are delivered. The measurements demonstrate some useful adaptive outcomes and substantial remaining failures; they do not establish general usefulness or superiority.

**Technical quality: PASS WITH ONE MINOR DOCUMENTATION FINDING for draft-PR readiness.** No Critical or Important finding, product correction, or additional test execution is required by this review. M1 below should be corrected in the publication documentation update. It does not alter the numerical quality result, source freeze, or readiness verdict. Original application code and all heldout outputs must remain frozen.

**Publication boundary:** ready to open a draft PR after the parent completes the planned review-index/publication updates. This verdict is not permission to merge or release and does not replace the user's PR/CODEOWNER signoff. The still-pending publication links/checklist at review time are explicitly parent-owned work, not a hidden implementation pass.

## Findings

### Critical

None found.

### Important

None found.

### Minor M1 — distinguish improved stopping from improved factual support

Locations: `docs/foundational/phase1f/evaluation-results.md:24` and `docs/foundational/phase1f/02-qualification.md:54`.

The first sentence says “Investigation delivered support improved from initial candidate to final on held-01 and held-03”; the second describes improved support in both the supported and missing-evidence cases. On the retained input **held-01 / investigation / response-06**, the adjudicated required-fact support is **4/4 initially and 4/4 finally**. The change is the stopping/applicability judgment, **false to true**, after resolving the unnecessary P12 qualification. See `live-heldout/grades-adjudicated.json:1425`, final support at line 1548, initial support at line 1686, and stopping at lines 1511/1644. Held-03 does increase supported required groups from 1/3 to 3/3 while still failing strict stopping.

Impact: the prose blurs two independently evaluated dimensions and can be read as claiming additional factual support for held-01. The displayed scores and two-versus-one strict success count remain correct. Suggested wording: “Held-01 improved stopping/applicability with required-fact support unchanged at 4/4; held-03 improved required-fact support from 1/3 to 3/3 but still failed strict stopping.” Change only the summary prose, retaining original grades and outputs.

Review-time document hashes: qualification `3cc535ab1181247625405c20cb1d047280079d1c079836094c7eed429b1dc3d1`; evaluation results `f98ee7c83f3613b492de864ee6e3f222482bc2ea4542de316151c49323e0b03f`. These identify the original finding even after publication wording changes.

## Source and evidence identities actually checked

I recomputed the manifest's length-prefixed path/content digest from every live contributing file, checked individual byte lengths and SHA-256 values, and checked every archive member against its archive manifest. I also checked all 33 source/diff inventory entries against live files. No use of `git diff HEAD` alone was made.

| Object | Verified SHA-256 / identity |
| --- | --- |
| Complete contributing source, 189 files | `2dacb402e50ee938d6184c14b40ba469fdcb8d91d9c30eac178b0dcb2a0fd3f8` |
| `source-manifest.json` bytes | `11aefe963bfb533c3b160584fd1d06c4b693c161eaf65b0c822cb0d908b163bb` |
| Pre-unseal contributing source, all 185 files still unchanged | `35f21d7f4624358717176b805195a0ebfab03fd53400e92a93a4fc33d617c3f1` |
| `evidence/retrieval-extension/after-source-manifest.json` | `3bc0497edb37053ae6f97244e08f5bc03f4e114a9d0e9cd959298251ba1dcae4` |
| `source.tar.gz`, all 355 members checked | `95407d737dc6002e3a43d4317e7390280a3f5b7dc983ab8f4c247e0fa139dcc6` |
| `archive-manifest.json` bytes | `cdbfe7810de535eea1dc876074d3592a2754c134953c324b0b50adb012c9df97` |
| `pre-heldout-freeze.json` | `16aff76356dbb4049bb5151d8edb574f8cafe333ac6262667e7a3029274c6d4e` |
| Evaluation freeze manifest | `afee3bc1b40c8f51bebf16ede379183f736136e43656bf90334d02690e6ce024` |
| Published heldout manifest / original corrected commitment | `600b2804b5e14ee8c7d5bf255408a612a68c71519237e0567ad30337a853fa33` |
| Published heldout inputs | `57641425198d3a3b02c3d6e685701e51ea2b581fc78541bceccc752a69148dcb` |
| Published heldout gold | `5ddce62c4bc796c000f93548f50ac46e93198127c498605ad6e7a82fafe7a437` |
| Final development and heldout binary recorded in every run | `62e830b4ba79a6e4542d03ccf461a4a24afe2f18ad0b5ccc61f6af8d9c262e50` |
| `evidence/live-budget.json` | `e185c70eefe69657414e50542de936f72dd85549d6a0e471c2585f275daeddb7` |
| Joined `evidence/evaluation-results.json` | `2a4c7dcc5868b551a0a06d7af15e37695f0eca686519180cf45dfc19096d35e9` |
| Heldout adjudicated grades | `ccfdecab67fde662409c3146b5faaaf99c9b8d71074a189f869146d5601339ad` |
| Supplied whole-branch diff | `ab453e5a8fbd18cbbbbe15602d0f027a42e35f5cb1e02a2b5aac52bb03bc6435` |

The pre-unseal review and run-plan hashes also match the freeze. The heldout manifest equals its earlier commitment and its two published inputs match that manifest. Every final-development and heldout run record agrees with the frozen binary identity. This demonstrates byte custody consistency; it is not independent observation of who accessed sealed content or a reproducible-binary attestation. The 185-to-189 extension adds the published heldout artifacts, not product tuning.

## Inspected implementation and reasoning

All changed production files were read: `adapters/lookup/lookup.go`, `examples/evidence/evidence.go`, `examples/investigation/{investigation.go,store.go}`, and `cmd/alan-investigate/main.go`. I read their new tests, external consumers, the new process/provider/counterexample package and race selectors, all new Python tools/tests, README/decision changes, and the requested plan, contracts, qualification/review/runbook, authoring observations, exclusion contract, run plan, freeze and evaluation reports. I also traced the unchanged runtime facade and adapter recovery/result/replay entry points that establish the reused authority boundary.

The coordinator sends both helper messages before either receive. It holds distinct initial and revision evidence, preserves pending identities and spent counters, and checks the received review before incrementing a bounded revision counter. A second revise terminates at the limit. The reviewer and helpers are ordinary, statically provisioned participants. The outer report derives publication routing solely from the public result; it adds no effect policy or scheduler. Invalid model envelopes/required fields lead to explicit failure/stopping paths. Model judgments remain untrusted semantic data even after structural validation.

The optional lookup profile validates immutable input/output declarations, exact protocol/profile agreement and zero-or-complete tuples. Matching retained references get range and UTF-8 checks before indexing. Selection excludes only exact document/content-version/start/end identities; scoring and first eligible tie behavior are unchanged. Unknown well-formed references and noncandidate subspans intentionally exclude nothing. Exclusions come from public initial helper answer slots, cannot be rewritten by model output, and are passed through ordinary messages. Legacy helper artifact anchors and legacy undeclared query-only compatibility have direct tests. The ignored-exclusion mutant is a valid output citation rejected by the same independently expressed input/output monitor.

Inspection derives summaries from a single public snapshot. Provision snapshots and verifies source bytes; recovery uses retained configuration/artifacts/assets. Unknown allowed attempts are skipped on reopen by the existing executor; a registry cannot turn them into new attempts. Replay enters the existing read-only ledger-validation path with no adapter registry. These are reused guarantees, not a new independent reducer proof.

The spending controller reserves and fsyncs before release, checks exact run/epoch/sequence/authorization correlation, refuses repeated operation identities and caps cumulative allowances. Reconciliation validates nonnegative integer token counts, consistent totals and cache counts, preserves allowances when usage is missing/malformed, and rejects model/tier deviations. A reconciliation exception leaves the prior durable reservation. The preserved original malformed-usage cases and their controls were inspected. The qualification controller does not choose application transitions.

## Evidence checked independently of summary prose

**Complete qualification:** verified decompressed raw-log hashes, absence of failure events and recomputed pass-event counts: default 550/37.742s; integration 756/39.440s; race 756/609.307s. Normal and race each contain 102 investigation records, including the three selected process scenarios and nine retained negative controls. Both actual child builds include `-race` in the race run. The raw archive-reproduction outputs match their recorded hashes and show 550 default pass events with an initially empty separate module cache and bundled file proxy. Auxiliary evidence records both vets, formatting, module verification, diff check, seven old comparator controls and 23 new Python controls passing.

Relevant command-record hashes: default `278eb9a20a4686d420230899dc201bf90d5ff47dc674affd38f1c1a1a801153f`; normal `3ddf0d680b1e72049fd668adc31ae3f7038f0e6383553acbe7635e725301f111`; race `10b43aff0f5203bb2799110437893078eb762a0427ed8dd42edbf3fbeb3a9b48`; archive reproduction `70f37e2a883c1f0be81931d12c5d5328edd46cee92ceb09d71d0bd2306bb2a64`; auxiliary checks `a1a3f18a618dcc4a989971352f6dbe9be5b3fdc589ad1490cefe2420e50274a1`.

**Causal process evidence:** inspected the monitors and full record structures, then independently checked selected normal/race records: review1 epoch 2/sequence 11 and review2 epoch 3/sequence 23 match retained authorization sources and remain unsettled at their selected cuts. The partial-sibling commit carries the held helper1 authorization; its exact wire matches the fsynced accepted/applied provider record and has no responding record. Offline before/after views and provider-ledger hashes agree. Real CLI SIGKILLs, exact pending identities, retained-message redelivery, genuine extra HTTP dispatch control, source deletion and unchanged unknown outcomes are retained. The task3 manifest is `e2c38053cef718ff5130c3422108b74ac61c8048583ca635d99a8c6f8d207b8d`.

**Earlier behavior:** independently compared all 15 qualification, 20 composition and 32 supervision payloads exactly in all three modes. Read the comparator and retained six-stream conformance results: their comparison is of declared public projections/model streams, not raw ephemeral ledger equality. Git's complete changed-file listing shows no old-phase artifact or runtime implementation edits. Historical archive reproduction remains separately identified.

**All live outcomes:** for all 32 paired runs I checked store hashes, exact run-record equality with the shared ledger, successful controller exits, provider IDs/returned model and usage, logical model/lookup totals, replay control records, actual delivered answer versus normalization/grade join, and source-input hashes. Recomputed per-study medians and strict success joins. The old live gate has its separate actual two-model/one-lookup result and unchanged pre/post-replay inspection. Totals recompute to 207 model requests, 71 lookups, 89,185 input and 35,044 output tokens, $0.591738 estimated charges, $0.9930895 conservative accounting, no live unknown reservations, and 33 retained successful offline replay checks. These are accounting estimates, not billing invoices.

**Heldout quality:** read the actual eight delivered outputs against the four questions, required fact groups, stopping rules and adjudication. The strict count is correctly two of four investigation versus one of four baseline. The missing signed-decision next action and the empty contradiction delivery remain failures. All 30 initial/final citation instances independently match document IDs, content hashes, bounds and exact source bytes. The irrelevant Birch citation in a Maple answer and duplicate spans add no coverage. An empty answer is never replaced by its diagnostic candidate. Strict/permissive stopping sensitivities and unavailable precision denominators remain disclosed.

## Limits and considered behaviors deliberately not judged as established

- **General usefulness or stable quality:** declined. Four related synthetic heldout cases and development-informed iterations support only descriptive results. More calls/retrieval opportunities confound an algorithm-only or equal-compute superiority claim. The failures are useful evidence, not grounds to tune heldout code.
- **Claim entailment from citations or application approval:** declined. Byte provenance, review acceptance and ordinary `publish` are separate from independent semantic support. The retained incomplete and stopped outputs falsify the stronger implication.
- **Arbitrary nesting/equivalence and universal author naming:** declined. The same public factory has controlled direct/nested reachability and an external-module boundary; this is not arbitrary-depth equivalence or a stable SDK. Unique static names/wiring remain the author's provisioning responsibility.
- **Unknown completion or retry safety:** declined. The held provider outcome remains unresolved and blocks final delivery. Reopen preserving that uncertainty is the intended result. Unused future helpers and the second reviewer receive may remain waiting; no cleanup/liveness guarantee is claimed.
- **Arbitrary late external candidates:** declined. The tested owner API can redeliver retained messages and settle recorded results; arbitrary adapter-result injection would require private operations or fabricated SQL. The scope limitation is explicit. No workaround was added.
- **Exhaustive concurrency, power loss, fairness and distributed ownership:** declined. Three causally acknowledged process scenarios, two review cuts, one sibling cut and race instrumentation are concrete finite evidence. They do not prove every schedule, storage power-failure model or multiple-executor deployment.
- **Independent semantic conformance from shared runtime/replay code:** declined. Same-reducer reconstruction is consistency evidence. The prior finite model comparison remains bounded to its declared projections and streams.
- **Provider-side dollar enforcement or future pricing/model identity:** declined. The local reservation formula and recorded alias/model/tier are disclosed assumptions. No credentials, live provider requests, price lookup, invoice audit or new authorization was used during this review.
- **Malicious local corpus manifest confinement:** declined as outside this trusted local authoring workflow. The CLI rejects lexical traversal and verifies requested content identity; it is not a sandbox for hostile filesystem symlinks. No corpus reread occurs during recovery/replay.
- **Cryptographic proof of blinding/actor access:** declined. Commitments and unchanged source/input bytes were verified, while access/timing attestations and practical blinding are retained process evidence. Answer structure can suggest the system; the grading disclosures acknowledge that.
- **Independent reproduction of every platform or exact binary bytes:** declined. The archive contains all declared sources/fixtures and the pinned dependency, with actual local archive-build evidence; Go/C toolchains and operating system are not bundled. Source/manifests and recorded binary identity are not a cross-platform binary reproducibility proof.
- **Pending publication metadata as a defect:** declined. The brief explicitly assigns final checklist, review links and draft-PR publication to the parent after this report. They must not be described as already completed by this review.

No narrow executable probe was needed after the source/evidence checks resolved the concrete risks. I did not rerun full suites, use credentials/network, invoke inference, alter product/index/branch state, or spawn agents. Review-time work consisted of source reading, read-only hashing/parsing and this report. Two initial exploratory read commands encountered reviewer-script assumptions (package-level pass events lack `Test`; the old gate uses `result.json`, not `summary.json`); corrected reads completed without changing inputs. They were not application/test failures. Original fixture-version, controller-usage, overstrict-epoch, overlong-query, repeated-span and ignored-exclusion counterexamples remain untouched.
