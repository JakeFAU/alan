# Phase 1E final-review correction — FR1

FR1 is corrected in one consolidated fix wave. An explicitly incomplete or in-progress output item cannot publish a validated payload, even when the response envelope says completed. This is an implementer report; independent re-review and the parent's final whole-source qualification remain separate gates.

## Scope and contract

The output-item shape now retains `status` as `json.RawMessage`, distinguishing absence from explicit null. Before interpreting item content, any present status must decode to the exact string `completed`; otherwise the adapter returns `protocol_invalid` and no payload. This covers message and reasoning items. The exact protocol-key guard now consumes status at item scope and rejects ASCII and Unicode case-fold aliases before Go's case-insensitive struct decoding can override a canonical value.

Omitted item status remains the documented existing compatibility subset, with completed top-level status and every other guard still required. Completed message/reasoning combinations remain positive controls. No controlled fixture migration, runtime guard, opcode, application policy, retry, repair request, provider tool, or dependency was introduced. The two application regression cases exercise the existing ordinary invalid-result branch.

Original bounded nonsensitive response bytes remain retained for the FR1 inputs and malformed status values. Existing sanitizer behavior remains authoritative: ambiguous duplicate-key evidence is omitted, and altered or uninspectable evidence cannot become success. Existing credential and response-limit regressions ran with this change. Already recorded outcomes remain historical replay inputs; the fix does not reparse, reinterpret, migrate or remotely reexecute them.

## Tests first and verification

Before production edits, `red.log` reproduces the exact original incomplete and in_progress HTTP bodies as wrongly validated with candidate payloads. Completed/absent controls pass. Both new ordinary application cases initially produce two model contacts instead of the required one, proving the incomplete planning decision reached lookup/synthesis. Full red test sources are in `red-source/`; original reviewed source, probes and logs remain unchanged in `../final-review/`.

The first post-fix attempt (`green-attempt1.log`, exit 1) fixed the protocol and application outcomes but exposed a test-oracle error: the new duplicate-key cases expected raw evidence retention even though the pre-existing sanitizer deliberately omits ambiguous duplicate-key JSON. The assertion was corrected only for those duplicate cases to require empty evidence; their `protocol_invalid`, nil payload and exactly-one-contact assertions remain. No production or sanitizer policy changed in response. This failed attempt is retained, not replaced.

The final focused normal and race runs each passed 96 test/subtest events with zero failures. These are bounded focused runs, not a claim to rerun the whole suite. They include 41 HTTP status cases: both exact original bodies, completed/absent message controls, message and reasoning statuses (incomplete, in_progress, null, empty, unknown, number, boolean, array, object, case-mismatched values), case/Unicode aliases and conflicts, duplicate keys, and allowed reasoning/message combinations. Each asserts exactly one contact, correct disposition, payload presence/absence, and retained evidence under the existing sanitizer contract.

All 20 `TestControlledApplication` scenarios pass, including both new incomplete-plan cases: `invalid_result`, one model request, zero lookup requests, one allowed transport attempt, and successful offline replay after endpoint closure with unchanged contact counts. Existing positive answer, clarification and insufficient-evidence branches remain covered.

Focused vet exited 0 with no output; gofmt listed no files. The unchanged original reviewer `probe.go` exited 0: its 32 arithmetic/control checks still pass, completed-message control validates, and both FR1 bodies return `protocol_invalid`, nil payload and byte-identical response evidence with exactly one contact each. The probe is an observation tool whose zero exit alone does not assert FR1; the retained last three JSON records were inspected explicitly. No live provider or real credential was used.

Actual argv, environment, exit codes and log hashes are in `evidence/final-fix/checks.json`; logs and runner source are alongside it. Go environment: `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`; Python evidence tools use `PYTHONDONTWRITEBYTECODE=1`. Authorized managed-worktree writes and loopback tests ran with escalation. Parent owns full normal/race qualification, archive refresh and publication; no unnecessary broad suite repeats were run here.

## Frozen source and preserved evidence

- Before: `fc1e319d8a81ff5269afabc0b98e1876b9a9a561a2b7595ce986296b78e225e1`, 129 files.
- After: `026321f39dce9332d24b4891a2e9164f0d14aaacdbecec69058989341e8196a8`, 130 files.
- Changed existing executable files: `adapters/openai/openai.go`, `adapters/openai/protocol.go`, `examples/evidence/evidence_test.go`.
- Added executable test: `adapters/openai/final_review_regression_test.go`.
- Updated contract paragraph: `docs/foundational/phase1e/01-authoring-effects-contract.md`.

The after manifest uses the original complete executable-source set plus the new test and the same sorted-path, uint64-big-endian length aggregation. Every other original executable input was verified unchanged. `against-frozen.patch` compares these owned files directly with the preserved source archive (including the contract); original file hashes were verified before diffing. `after-changed-source.tar.gz`, `after-changed-hashes.json` and `after-source-manifest.json` retain the exact resulting bytes and identities. The parent may package updated status documents without changing executable identity.

No source changes follow this freeze. No subagents, commits, branch/index mutations, full-suite reexecution, live-provider request or historical evidence rewrite occurred in this fix wave.
