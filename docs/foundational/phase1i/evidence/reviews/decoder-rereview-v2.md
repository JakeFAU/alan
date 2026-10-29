# Phase 1I scoped independent decoder rereview

Decision: **APPROVE** the decoder correction slice for model-review R1/R2 and the runtime review's explicit-closure finding. No unresolved actionable finding remains in this rereview scope. This is not approval of the actual P/Q/W model mapping, final combined normal/race qualification, manuscript, or whole branch.

Workdir: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`. Base `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`. Read the fix handoff, both preserved v1 REJECT reports, all of `evidence_test.go`, `decoder_consistency_test.go`, `decoder_review_test.go`, `decoder_closure_test.go`, current normalization and its call sites, closure assertions and root-success fence. No edits, subagents, paid calls, commits, or external messages. All new review outputs are in `/tmp`.

## Disposition

**R1 — resolved.** Normalization first calls `validateRetained` and emits an explicit `retained_evidence` violation without `EvidenceChecked` on a mismatch. Consumption fields are then decoded directly from the selected observation payload, not copied from a publication. The validator separately links observations to original publication/admission/delivery, checks actual provenance, template source, generation, obligation, caller, operation, parent/root/member and binding records, and checks the selected history/consumption sequence. It evaluates the original receive guard using the predecessor projection and original handle register, and compares present bound output fields to the actual successor. Consequently, reducing the normalized model observation after successful validation no longer silently discards unchecked versions of the R1 identity/payload distinctions.

**R2 — resolved.** The validator now requires terminal singleton freezes and applicable root-scope freezes; checks their run, scope, root, sequence, trigger and fact links; checks captured invocation/reply/external obligations; and ties local terminal projection to terminal history and the original end instruction. Publication is checked against accepted frozen replies and local freeze status. Local success with no reply remains failed/missing_reply at the invocation boundary. It does not collapse this distinction into local failure. The cancelled-root path uses its actual lifecycle evidence. Root success alone does not imply that all independently provisioned static peers completed, and the static-root freeze handling preserves this distinction.

**Explicit logical closure — resolved.** `Closed` now derives from the actual accepted open→cancelled root lifecycle transition, with original intent and current projection linkage. Missing or mismatched cancellation evidence fails. The declared CancelRun profile does not silently reinterpret other closure records: unsupported composition-closure profiles explicitly reject. Normal root success remains `Closed=false`, so the root-success admission-fence counterexample is preserved.

The `requireEvidence` gate is used in primary sample membership, closure, root-fence, weak/PQ and intentional wrong-id/extra-helper paths. Intentional application-contract violations therefore cannot waive raw-evidence validation. The unchanged independent application oracle remains responsible for its separate K1 behavior checks. No production reducer/compiler is invoked by the new evidence validator.

## Independent executions and evidence

Ran from the workdir:

```sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build ALAN_I_ARTIFACTS=/tmp/alan-phase1i-decoder-rereview-controls go test -tags=integration -run '^(TestSubstitution(Decoder.*|RootSuccessFence)|TestReviewConsumedPayloadIdentityPreserved)$' -count=1 -json -timeout=2m ./internal/substitutiontest > /tmp/alan-phase1i-decoder-rereview-controls.jsonl 2>&1
```

Exit 0, package elapsed 0.691s, **61 passed leaves**, no failures. This consists of the five original isolated sensitivity tests, 33 added evidence-consistency mutations, three cancellation-evidence controls, 18 archived actual normal/race cancellation cases, and two actual A/B root-success-fence witnesses. No duplicate full 236-case runtime run was performed. In particular the original five attacks now reject explicitly with `retained_evidence`; they are not made green by final model outcome membership rejecting a zero-valued observation. The 18 actual historical cancellations now decode `Closed=true`, while both root-success witnesses retain `Closed=false` and their original completion counterexamples.

Fresh original/mutated records and positive root-fence records are at `/tmp/alan-phase1i-decoder-rereview-controls/`. Parsed exact leaf list, package result and independently recomputed source hashes are `/tmp/alan-phase1i-decoder-rereview-evidence.json`. All 13 internal/substitutiontest source hashes matched `evidence/runtime/review-fix/decoder-current-source-manifest.json` at review. Key hashes:

- evidence_test.go: `9d0d01b6ec168e347aa0bbaefdc4a1abfaa1bdf9b1f8bd8bbdb20b7be98b806c`
- fixtures_test.go: `a004f4413fde4f5075406af7e949d5dc7247344db4850b7241ddacf5ed326c9a`
- decoder_consistency_test.go: `0683436821e7dc7be64bc9447e57614a5cdfde75a0f867b310827837f2a2432f`
- decoder_review_test.go: `89a3616ff8c68c4f272c429a977ca89198fd786a4720941a939efb58e7172d16`
- decoder_closure_test.go: `5947cb07f2c62e4195fb7aeb23df04a9984d3b08b7b81c7b920eabeecb545336`

The preserved model-review-v1 report is byte-identical to my original `/tmp` REJECT report. Contract SHA256 remains `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`; A/B source hashes remain their frozen first-version values; oracle SHA256 remains `9f7f5f59414c439392ab813759c6ed1c99f4be21aaa44f03a24cf953eb3899ad`. `git diff --stat` against base for internal/executor, runtime, author, schema and effects is empty.

## Scope and remaining gates

This approves correction of the identified decoder omissions under the declared finite witness profiles. It does not establish that this consistency validator detects every possible fabricated retained store, prove compiler/reducer correctness, or establish general trace refinement. The negative mutation tests modify in-memory Inspection values only and make no claim that production execution creates those inconsistent records.

The model author's actual P/Q/W correspondence work is active and was deliberately not approved here. Its integration may change contributing source, so the final exact-source normal/race run and final whole-branch review remain required. The reported 295-leaf development run is attributable to the implementer's handoff; my independent execution is the 61-leaf focused run above. Preserved REJECT evidence must remain alongside this scoped resolution.
