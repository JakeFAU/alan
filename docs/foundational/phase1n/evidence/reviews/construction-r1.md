# Phase 1N construction review R1

Reviewer: `n_construction_review`, independently dispatched from the construction author. Scope: Task 2 feasibility/preflight only. Base: `f54c66a22ab4ba16201a12b2ad8f7ef344e38d1f`.

**Specification: APPROVE. Quality: APPROVE.** No blocking findings or required construction corrections. This does not approve K_join/G_outer qualification, all-schedule progress, compiler correctness, correspondence, or lifting.

## Exact reviewed identities

The 35-file additive diff `.superpowers/sdd/00-plan-phase1n/task-2-diff.txt` has SHA256 `d51913bd20d63233f2cdb0634d029f444e28b197151453e4804468ff0874a1a3`. Every NEW entry's size and hash was recomputed; all 35 matched current files. The preflight `index.json` has SHA256 `af3087dcf6f399482d1d04e6effb8a503dbcb0b66633f963f3d7829fed0e43ca`; all 25 indexed evidence files matched their recorded size/hash. Gzip evidence was decoded through this index, not omitted because it lacks textual diff hunks.

Contract `docs/foundational/phase1n/01-contract.md`: SHA256 `001c1fe5cd7be722aa75ae00a2be84ec702b51783a5595754d38ec8e6b2d8fd9`.
Representation `docs/foundational/phase1n/01-representation.md`: SHA256 `d0fa2892efb279d87d12111a27450425e5efe17d562b71d94d5058b5d4dffe23`.
Implementer report `.superpowers/sdd/00-plan-phase1n/task-2-report.md`: SHA256 `4f82c667418ebc326d214ac366c9e5cdb757b271685dab7502788d78e96e9978`.
The brief, constraints and repository AGENTS.md were read.

| Reviewed public source | SHA256 |
| --- | --- |
| `examples/callablejoin/deployment.go` | `3dcd0dcb2a1b018ad7a7c09d9a48653dc7340a32607f615f5644d78e33b160b4` |
| `examples/callablejoin/join.go` | `241895cdfcf9eaa29d1e25dda86fea690b4803ef8c07fa66fac4b905c4c45542` |
| `examples/callablejoin/outer.go` | `1c8d455a17888a96e677810a6dded0218d205c302a87e055f0159e77300ac3b4` |
| `examples/callablejoin/deployment_test.go` | `71949bd9c00449d0d7c40e471a3a7ef87bf94815a7e3213e6d99f3b7dd9315b0` |
| `examples/callablejoin/consumer/main.go` | `d45481fdd6a6cea91fdfbeed2c29e76a9820d0ac3c41f8ea796a6944bd9b7da4` |
| `examples/callablejoin/consumer/preflight_test.go` | `aaefdacb58901e7235f67359eac1cc196fadabdb5c4f15216755e3ed163c9ed8` |
| `examples/callablejoin/consumer/go.mod` | `1f51244a0fe6c12cb23419b508fbe0a1febc3b0f251b24781ef76be153d451b0` |
| `examples/callablejoin/consumer/go.sum` | `9eb975c181ebee792e06f732fc13ef81acad0779d94731f75133ad2bba52c5ca` |

## Specification assessment

- `examples/callablejoin/deployment.go:20` obtains original S and discards the old provision. `join.go:67` directly receives the public request, commits original candidate then S, selects S then candidate through original handles, classifies, Sends once, and Ends. No forwarding wrapper, sibling, extra reporting channel or inner-handle serialization is introduced.
- `join.go:33` and `join.go:50` classify actual selected status/reason/presence/payload, requiring valid S first. Presence guards dominate payload reads. Failed/missing-reply candidates require absent responses and empty answers. Only a validated answer appends. Mode is forwarded without driving classification.
- `outer.go:38` guards public presence and id before interpretation. Its bounded loop checks existing records; answer requires one valid record, failure requires zero. Status/reason/tag checks preserve the two diagnoses and distinguish invalid public evidence. `outer.go:57` retains and selects the original J handle; a committed decision write determines End. O does not inspect private children. Structural guard tests do not replace later adverse runtime controls.
- Original A/B/helper/S identities are pinned in tests and bundles. Protected A/B/protocol/S-builder and relevant public/executor source hashes match the base. Independently normalizing raw J bytes by only candidate ID, and O bytes by only J ID, gives equal A/B policies. No policy difference is concealed by normalization.

## Evidence and bounds assessment

I inspected captured results without rerunning tests. Accepted root commands are `go test -json -count=1 ./examples/callablejoin` and its `-race` variant; accepted separate-consumer commands are `go test -json -count=1 ./...` and its `-race` variant. Each captured exit is zero with four passing leaves. Every accepted record's construction-source hashes match this review. The consumer is a separate module using public imports and existing pinned dependencies.

The final three records capture Go 1.27.3 darwin/arm64, sqlite module v1.14.52, effective Go environment, readonly modules and disabled proxy/sumdb. The earlier construction-normal record retains argv/environment/source hashes and raw output but lacks embedded toolchain/effective-environment fields; its leaf outcomes were decoded from output. No stronger provenance is inferred for that record.

All eight accepted normal/race raw samples were decoded. O's original J operation has exactly one publication and one selected root invocation observation linked to history. Answers publish succeeded/member_succeeded with a present one-record diagnostic. A fail and B absent publish failed/member_failed with present empty-answer diagnostics retaining candidate_failed and candidate_missing_reply. Root outcomes and retained decisions agree. The test verifies a committed O set instruction, not merely a host verdict. Counts agree across normal/race: 78/9, 85/12, 62/8, 74/11 commits/dispatches. Replay is shared-runtime consistency evidence.

Raw retained bytes independently reproduce J 58 instructions/25,387 bytes; O 64/22,926; A 13/6,762; B 16/9,533; helper 7/3,951; S 3/2,632. Hashes match program IDs. The 138/148 static totals and 139/149 conservative commit ceilings follow from the fixed topology and O's MaxItems1 loop. Historical collection reduction was inspected for the named risk that loop might re-execute foreach: it jumps to target+1, so only next repeats once. Ordinary dispatch accounting and the fixed invocation/reply topology support 9/12 dispatches under the declared stable/no-retry/no-closure assumptions. Consumer limits 512/64 exceed them. Charges are (3,3,0)/(4,4,0), with depths 2/3 under I=R4/depth3/E0/fixed1.

Independent re-encoding gives 114-byte finite-profile answer arrays, 152 bytes for two MinInt64 values, and a 386-byte maximum report with longest tag and full-int64 fields. The list fits MaxBytes192. The register/cursor argument supports the loose 32KiB projection allowance for this finite stable topology; 1,548 bytes is correctly only an observed peak. Observation/output sample maxima are not promoted to universal envelope bounds. Historical defaults confirm unchanged 1MiB ceilings.

Historical source inspected for those named bounds risks: `internal/executor/collections.go` SHA256 `a3ead48f26379383a54734b2288225c3a9a64ed0a826d89de8af966c99b861c7`; `internal/executor/program.go` SHA256 `4a6813c69d509dff545c75a05410919e182e73c9f06e5fd359112e64df568374`; `internal/executor/runner.go` SHA256 `ad9fe52c4f17adb4c528b9397fb2e90338f9f9e63c5015160946daa593c1b2f4`. Other protected historical sources were hash-compared, not semantically re-reviewed.

Initial RED, nonexistent list accessor and incorrect initializer-PC assertion failures remain inspectable and agree with their recorded dispositions. The broad interrupted root run is **excluded**. Its raw output independently yields exit 1, 1,098 completed passing leaves, no completed failing leaves, and the recorded first/last timestamps. The unfinished adaptive test and termination disposition remain retained. This is neither a green root suite nor construction acceptance evidence. Process/signal details are attributed to the interruption record, not independently reproduced.

Review actions: source/diff inspection, indexed gzip/JSON decoding, SHA256/size checks, raw program normalization and arithmetic checks with Python, and targeted historical reads with rg/sed. No tests were rerun, no rendering occurred, and no production/example/model files were changed. The sole written output is this review.

## Findings and remaining gates

No blocking findings; no required Task 2 resolution. These later gates remain unverified:

1. Independent reference review, constructor premise, aligned residual environment and both S admission orders, full finite correspondence, and generic conditional lifting.
2. Negative/public-lie controls and independent raw truth audit, including absent/inconsistent public outcomes and exact cardinality defenses under adverse evidence.
3. The requested runtime evidence: eight automatic answer samples, representative A/B fail/absent/pending samples, one supported late-reply schedule, and controls, with authorization, admission, End, publication, delivery, selected consumption and root publication distinguished where applicable. The full 48-configuration census and exhaustive pending-maximality obligation belong to the finite model; this review does not require an additional runtime cross-product or exhaustive runtime schedule campaign.
4. Synchronized manuscript reconciliation and final integration verification. Existing theorem qualifications remain unchanged by this review.

Concurrent new research/model files were not inspected or modified. Their independence and results are outside this verdict.
