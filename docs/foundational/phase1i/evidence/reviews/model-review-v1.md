# Phase 1I independent model/correspondence review

Decision: **REJECT** the combined model/runtime-correspondence handoff pending R1–R2 below. The finite model calculations and inspected A/B instruction transcription have no blocking finding from this review; the rejection concerns required runtime correspondence evidence, not the retained negative research result. Reviewer agreement is not proof.

Workdir: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`. HEAD: `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`; contributing files remain uncommitted. Read the assignment, frozen contract/rereview, all current `research/substitution/*.go` files, model README/report, exact correspondence manifest and A/B retained artifacts/source, and runtime normalization (moved from `runtime_test.go` into `fixtures_test.go`). No repository source or documentation edits, commits, external messages, paid calls, or subagents. Only `/tmp` review artifacts and disposable test databases were created.

## R1 — P1: consumption evidence is replaced by the publication it should independently corroborate

Location: `internal/substitutiontest/fixtures_test.go:120–123,228–230,248–253,309–323`.

`normalize` reads the selected observation but emits `outcome_consume` using `ps[o.Operation]`. Thus its consumed status, reason, response presence and payload come from the publication, not the actual consumed observation. The small observation decoder also discards provenance, source, source generation and obligation. `CheckTrace` consequently compares publication data against a second copy of itself; final context membership cannot recover the erased information. This violates the frozen contract's explicit preservation of generation/provenance/correlation and its requirement to abstract consumed observations, with one consistent identity correspondence.

Reproduced against an actual B/bare/id1/value0/answer execution at I=R=depth=3. Independently altered only the in-memory Inspection supplied to the decoder (never the retained database):

- `consumed_payload`: changed the two invocation observations' existing `response.id` from 1 to 2.
- `consumed_provenance`: changed their existing provenance to `ordinary-message`.
- `consumed_generation`: changed their existing `SourceGeneration` to 99.

For each isolated mutation, the complete normalized trace and Context remain identical, `Violations=[]`, and `CheckCandidateContext(..., Helper, ...)` returns nil. These are decoder sensitivity counterexamples, not a claim that the production runtime currently creates corrupt observations. They establish that the advertised independent correspondence check cannot reject the relevant discrepancy.

Required correction: decode consumption from the selected observation, independently validate it against publication and the actual selected receive/handle, and preserve/check provenance, generation, operation, parent/root and caller relationships. Missing/mismatching evidence must produce an explicit rejection instead of silently disappearing. Add these isolated decoder negative controls; retain the original failing inputs/logs. A separate consistency validator before lossy normalization is acceptable if it checks these facts independently and its rejection is part of the qualification path.

## R2 — P2: member terminal/freeze facts are outside the membership check

Location: `internal/substitutiontest/fixtures_test.go:151–309` and `research/substitution/explorer.go:8–28,215–229`.

The runtime decoder records raw run statuses only diagnostically, ignores `composition_freezes`, and checks final candidate publication plus root/peer status and charges. Therefore it does not check the frozen contract's distinction between candidate local projection/terminal freeze and invocation envelope. This is a material omitted correspondence boundary, particularly because the earlier v1/v2 defects were exactly at terminal and response-presence boundaries.

Two further isolated mutations of the same actual execution demonstrate the gap:

- `candidate_local_status`: changed the candidate member's actual retained projection from succeeded to failed (one record).
- `missing_member_freezes`: removed all three freeze records from the decoder input.

Again each leaves the normalized trace and Context identical, violations empty, and membership accepted. Existing raw diagnostic retention is useful but is not validation. Required correction: validate member terminal/freezes and their linkage to original publication and local projection, with explicit missing-evidence rejection. It need not claim exhaustive compiler validation or turn these tests into production changes.

## Preserved and reproduced positive/negative results

- The fixed K1 initial entitlement remains 84/3840, including the eight parallel configurations. It is computed without A/B outcomes; future-ticket/spent checks run on successors before deduplication.
- Local relation checks cover 3072 independent environments per candidate; they match component events asymmetrically and environment readiness bilaterally at private instruction boundaries. Separate no-closure progress avoids vacuity from optional cancellation.
- Full state identity includes private PCs/registers; no history length is used. Strict rank is checked on revisited edges. Current positive graphs have zero rank failures/cutoffs/stuck states. This only establishes acyclicity for these finite enumerated graphs.
- K1 is built independently of candidate selection. Candidate instructions share declared protocol transitions but transcribe branch operands/private registers separately. I inspected all A13/B16/helper7 mappings, both branch arms, helper invocation payload/handle/receive guard, and unreachable second-receive tails. The package imports no production compiler/interpreter. Shared protocol assumptions remain an explicit limitation, not an independent proof of that protocol.
- Re-ran the whole focused model package: PASS, 5.555s. Every emitted result JSON matched the retained model evidence structurally; every model-source hash matched `final-source-manifest.json`. K1/A/B original state/edge counts and extension counts match the report. Each boundary census retains eight completion failures and 76 unaffected entitled successes, without setting external Closed or deleting initial entitlement.
- Wrong-id and extra-helper controls reject for their intended codes under the unchanged oracle; the extra-helper mutation also has a retained rank failure and an actual third charge. This does not support an acyclicity claim for that malformed graph.
- V1 failed-consumption/freeze and V2 helper-payload leak snapshots/results and RED logs are retained; the separate current fail/absent/pending oracle passes.
- Root cancellation extension leaves the static peer alive and rejects its admission as scope_closed. Frozen original cancellation assumptions are not silently promoted to runtime truth.
- Re-ran `TestSubstitutionRootSuccessFence`: PASS for A and B. The separately named 32-Set peer is explicit; original frozen caller remains unchanged. Actual root success, peer failure and scope_closed denial are checked directly; original K1 rejects the final observation and the boundary extension retains its completion counterexample.
- Weak W unequal charges remain outside Definition 4.6. Equal-charge P/Q local relation and abstract trace-language equality remain separate from deterministic Store.Run. The five K1E automata are explicitly auxiliary schedules, not effectful artifact correspondence or universal recovery proof.

## Exact reproduction and artifacts

All commands executed from the workdir above:

```sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build ALAN_MODEL_EVIDENCE=/tmp/alan-phase1i-review-model go test -v -count=1 -timeout=10m ./research/substitution > /tmp/alan-phase1i-review-model-tests.log 2>&1

env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build ALAN_I_ARTIFACTS=/tmp/alan-phase1i-review-adversarial go test -overlay=/tmp/alan-phase1i-review-overlay.json -tags=integration -count=1 -timeout=2m -v ./internal/substitutiontest -run '^TestReviewConsumedPayloadIdentityPreserved$' > /tmp/alan-phase1i-review-normalization-expanded.log 2>&1

env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build ALAN_I_ARTIFACTS=/tmp/alan-phase1i-review-root-fence go test -tags=integration -count=1 -timeout=2m -v ./internal/substitutiontest -run '^TestSubstitutionRootSuccessFence$' > /tmp/alan-phase1i-review-root-fence.log 2>&1
```

Exit codes: 0, 1 (all five isolated negative decoder tests fail as described), 0. The review test is `/tmp/alan_phase1i_review_normalization_test.go`; overlay mapping is `/tmp/alan-phase1i-review-overlay.json`. A rerun should use a fresh artifact directory because `retain` intentionally uses O_EXCL. Each mutation's original/mutated Inspection and normalized results is retained under `/tmp/alan-phase1i-review-adversarial/{da5cc19c4328,f205b7742a43,d25f4a4a6ddc,6833f76b39d0,90fd8aa06621}/input.json.gz` in listed order. Initial exploratory combined-mutation test/log is separately preserved; it originally used a nonexistent `Generation` key, then was corrected to actual `SourceGeneration`, and the final isolated tests assert field existence before mutation. Conclusions rely on the corrected isolated executions.

Full independently recomputed source hashes: `/tmp/alan-phase1i-review-source-manifest.json`.

- Contract: `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`.
- Runtime fixtures: `2c5bec796eaba5e7ddcfcf8d46314e8f41fc085c5dc9a39d3a51983013898607`.
- Correspondence manifest: `c2ac40dbb2477786e5b8ebeb79cf7636993bf35a4553899ccf700fdbedaa44a3`.
- Frozen oracle: `9f7f5f59414c439392ab813759c6ed1c99f4be21aaa44f03a24cf953eb3899ad`.

No claim of production corruption or universal model failure follows from these decoder attacks. R1/R2 are actionable evidence defects in the required sampled runtime correspondence. Correct and re-review them while preserving the supported bounded negative result. Whole-branch manuscript review, general contextual closure, compiler correctness, and distributed recovery remain outside this scoped approval decision.
