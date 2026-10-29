# Phase1I runtime final integration and capture — v2

Result: final exact-source normal **295/295 PASS (17.369s)** and race **295/295 PASS (223.397s)**, exit0 each, no skips, failures or race reports. Actual child processes in race-v2 execute the same real race-instrumented binary. Focused vet and owned-file gofmt checks have empty output. No production runtime/compiler/author/schema/effects/boundary, frozen K1, A/B or oracle edits; no commits/subagents/paid calls.

## Exact source and commands

`docs/foundational/phase1i/evidence/runtime/fixtures-qualification-v2.json` was frozen before the final pair. It contains actual normal/race `go list -deps -test` project build-source selections, 85 project-source hashes (plus conservative explicit model-test/nested-consumer sources), exact expected295 names, commands, Go version/platform and profile parameters. SHA256: `2202e48959cf40708628b1d27525f6f2e9533c5051a4f5f7cca4a4b89b877fc0`. Complete source text snapshot: `runtime/review-fix/final-source-snapshot-v2.json.gz`. Hashes were checked after both final runs: zero drift.

Final model source agrees with the independent model author's stable `evidence/model/probe-artifacts/final-source-manifest.json`; actual probe graph API was confirmed stable before capture. Actual public P/Q/W transcription is separate from frozen B control. The original protocol language/local-relation results are separate from the inspected runtime-boundary extension used for concrete terminal membership; no boundary-language equality claim follows.

Both final commands use:
`env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build`.

Normal:
`ALAN_I_ARTIFACTS=<evidence/runtime/normal-v2> go test -tags integration -run '^(TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence|Decoder.*)|TestReviewConsumedPayloadIdentityPreserved)$' -count=1 -json -timeout=10m ./internal/substitutiontest`.

Race:
`ALAN_I_ARTIFACTS=<evidence/runtime/race-v2> go test -race -tags integration -run '^(TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence|Decoder.*)|TestReviewConsumedPayloadIdentityPreserved)$' -count=1 -json -timeout=30m ./internal/substitutiontest`.

Focused static: `go vet -tags integration ./internal/substitutiontest ./examples/substitution/...`; owned formatting: `gofmt -l internal/substitutiontest/*.go examples/substitution/probes.go examples/substitution/effect.go examples/substitution/consumer/*.go`. Outputs empty. No duplicated old regressions or unchanged consumer test reruns were added; standalone consumer's earlier same-source normal/race/vet/mod gates remain in consumer-gates.json.

Both profiles select old236 leaves plus59 decoder leaves: matrix190, probes26, recovery4, closure9, effects5, root-fence2, original isolated sensitivity5, archived actual cancellation18, additional consistency33 and cancellation-evidence3. ProcessChild is an explicit subprocess entry, not another claimed experiment. A/helper-commit is inapplicable and uncounted. Every exact selected name and package timing is in `normal-v2-summary.json` / `race-v2-summary.json`; compressed event logs are `normal-v2.jsonl.gz` / `race-v2.jsonl.gz`.

## Corrections and independent rereview

Both original REJECT reports and original failing source/logs/inputs remain at `evidence/reviews/{model-review-v1,runtime-review-v1}.md`, `reviews/decoder-v1`, and `runtime/review-fix/original` / `red`. Original236 v1 records and logs were not replaced.

Independent scoped decoder rereview returned **APPROVE** for R1/R2 and explicit logical closure. Retained report: `evidence/reviews/decoder-rereview-v2.md`; independent61-leaf log, selected names and source hashes: `reviews/decoder-v1/decoder-rereview-v2{.jsonl,-evidence.json}`. This approval covers the decoder slice; it is not a whole-branch/model-mapping/manuscript approval. The final integration only adds actual probe membership at the already approved raw-evidence gate.

`normalize` first validates raw retained links, requiring `EvidenceChecked=true` for every actual matrix, weak/PQ, semantic negative, closure and root-fence witness. Missing/mismatched observation payload/provenance/template/generation/operation/obligation/caller/parent/root/binding/delivery/history/receive-handle evidence explicitly rejects. Consumption fields are read from the actual selected observation Payload, not copied from publication. Terminal local projections/end instructions, freeze triggers/run/scope/root/sequence/captured obligations, accepted replies and publication envelopes are checked separately. Success without reply remains local success plus failed/missing_reply invocation envelope.

Logical root cancellation is derived from the actual accepted open→cancelled lifecycle transition and matching original intent/current projection/freeze/publication, not `composition_closures` table nonemptiness. All9 fresh and18 archived real cancellation cases decode Closed=true in each final profile; both successful-root-fence cases remain Closed=false. Undeclared retirement profiles reject explicitly rather than silently masquerading as this phase's root CancelRun profile.

## Actual probe model integration

All **24 actual weak/PQ executed leaves** now call:
`CheckProbeContext(config, profile, observed, boundary=true)`.
Their retained `probe_model` result has no counterexamples, and the normalized record states exact actual-probe terminal membership. Independent raw evidence is required even when a separate weak/K1 complement disagreement is intentional. P/Q additionally requires the unchanged K1 trace monitor's safety verdict. The valid wrong-id and extra-helper controls retain EvidenceChecked=true and still reject for intended application_id and extra_helper reasons after actual successful Run/Replay; accepted extra-helper debit3 remains required.

Weak8 and P/Q16 profiles/bounds are unchanged. Weak-b helper/candidate return0 matches the actual artifact rather than the old complement-helper surrogate. P/Q actual distinct controls and Q's extra pure Set are independently transcribed. All8 P and8 Q concrete samples in normal-v2 retained peer success/candidate helper refusal/root failure; no sampled distinguishing P/Q outcome. Model language/local relation evidence remains separate from these concrete samples.

## Process and closure evidence

Original real ACK/SIGKILL/fresh Open+Run mechanics, exact original records, endpoint keys, spent reservations and OS-network-denied replay are retained unchanged in the final source. Four pure recovery cases per final profile retain distinct killed/fresh/replay PIDs, one actual SIGKILL, unchanged binary digest within profile and root epoch **2→3**, followed by successful original continuation. Exact PIDs/argv/env/binary hashes/exits and epoch pairs are in each profile summary and per-case records. Cancellation and live effect observations are independently recorded by actual retained facts/HTTP endpoint ledger; unknown recovery still has one spent original attempt and zero recontact, while live late authentic receipt remains separate.

Nine final closure cases keep the same raw envelopes and charges previously qualified, now with corrected normalized Closed=true:
- A entry-commit: debit0, rejected/source_denied:source_closed.
- A admitted/reply-commit/published: debit1, succeeded/member_succeeded, present.
- B entry-commit: debit0, rejected/source_denied:source_closed.
- B admitted/helper-commit: debit1, helper rejected/scope_closed; candidate failed/member_failed, absent.
- B reply-commit/published: debit2, helper and candidate succeeded/member_succeeded, present.
All root outcomes remain cancelled; operator fixed cost1 is outside root. Root success fencing later peer work remains a distinct negative witness, not logical cancellation. Both A/B frozen optimistic memberships reject those root-success tuples, while boundary membership retains completion counterexamples.

## Evidence index and remaining scope

Final gate/source/count/epoch overview: `runtime/review-fix/final-gates-v2.json`. Original row artifacts: `runtime/normal-v2` and `runtime/race-v2`. Supplementary post-run checks: normal-v2-record-checks.json and race-v2-record-checks.json, each verifies8 weak payload/admission and9 exact closure obligations without importing ALAN. Reproducible capture: `runtime/review-fix/capture-v2.py`. New normal/race summaries explicitly retain24 probe memberships,9 corrected fresh closure observations,4 recovery PID/epoch groups and2 root-fence model rejections.

The 295-case development run and intermediate compile/guard corrections remain historical under review-fix; only v2 final pair is claimed as this exact-source final runtime gate. These tests and raw-consistency checks do not prove compiler correctness, general trace refinement, detection of every fabricated store, scheduler confluence, a distributed recovery theorem or universal substitution. Controls mutate only in-memory Inspection values; they do not claim production runtime corruption. Original SQLite temp directories are not retained, but complete original public Inspect row bytes/program/history/ledger records and hashes are retained.

Runtime stream is complete and ready for parent final source index, summary/manuscript reconciliation, whole-branch review and PR. No further source edits planned.
