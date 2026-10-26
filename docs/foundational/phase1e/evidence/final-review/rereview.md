# Phase 1E final scoped re-review — FR1 correction

**FR1: ADDRESSED.**

**Specification verdict: APPROVED for FR1 and the consolidated correction scope.**

**Code-quality verdict: APPROVED for this correction scope.** No new actionable breakage was found in the fix diff. Together with the original whole-branch review, this closes the only final-review finding. The parent still owns fresh full-source qualification, archive/status refresh and draft-PR publication. This review authorizes neither merge nor release.

## Why the correction resolves FR1

The output-item struct in `adapters/openai/openai.go` retains status as `json.RawMessage`, preserving the distinction between absent status and an explicit JSON null. Before interpreting any item, a present value must decode as the exact string `completed`. Incomplete, in-progress, unknown, empty, null and nonstring values return `protocol_invalid` before any application payload is populated. This covers both accepted message and reasoning item types.

`adapters/openai/protocol.go` includes status in the exact output-item key set. Case-fold aliases, including Unicode long-s, are rejected before Go's struct decoder can override the canonical field. Existing duplicate-key and sanitizer checks remain intact. There is still one HTTP exchange and no retry, tool execution, repair request, runtime special case or dependency change.

Omitted item status remains the explicit compatibility subset recommended in the original review: completed top-level status and all existing payload/content guards remain required. Positive controls cover absent/completed message and reasoning items. The contract paragraph documents this distinction and the rejection of present invalid values. It also correctly states that already retained dispositions remain historical observations rather than being silently reparsed, migrated or remotely repeated by this correction.

## Independent verification

Reviewed the complete `against-frozen.patch`, the full new test, existing application assertions around both new cases, the report, changed-file hashes and the raw focused logs. Independently recomputed all 130 source/input hashes and aggregate `026321f39dce9332d24b4891a2e9164f0d14aaacdbecec69058989341e8196a8`. Against original `fc1e319d8a81ff5269afabc0b98e1876b9a9a561a2b7595ce986296b78e225e1`, exactly these executable inputs differ:

- `adapters/openai/openai.go`.
- `adapters/openai/protocol.go`.
- `examples/evidence/evidence_test.go`.
- Added `adapters/openai/final_review_regression_test.go`.

The only additional owned file is the documented contract paragraph. All five changed-file hashes match the frozen correction manifest; every other original executable input is unchanged.

Independently reran the unchanged original review `probe.go` using fresh disposable SQLite stores and a synthetic loopback HTTP provider. All 32 arithmetic/control cases still match their oracle. The completed-message control remains validated. Both exact original FR1 bodies now produce `protocol_invalid`, null payload, byte-identical retained response evidence and exactly one contact. These outcomes were separately asserted from the JSON output; the diagnostic process's zero exit alone was not treated as proof.

The retained focused normal and race logs each contain 96 passing test/subtest events and no failures; their hashes match checks.json. The 41 status cases include original inputs, absent/completed controls, message/reasoning invalid statuses, ASCII/Unicode aliases and conflicts, duplicates, and allowed mixed-item combinations. Each checks disposition, payload, evidence and one contact. The 20 application cases include both new incomplete-plan cases, which reach the ordinary `invalid_result` branch with one model request, zero lookup requests and one attempt, followed by successful offline replay after endpoint closure. Existing successful answer and alternative branches remain passing. Focused vet and gofmt logs are empty and successful.

The retained red run reproduces the original incorrect acceptance and the application's extra downstream contact. The first green attempt failed only the duplicate-key evidence-retention expectation; the production parser fix was already behaving correctly. The test correction requires omission for duplicate-key cases, as the existing sanitizer already did, while preserving their nonsuccess/no-payload/one-contact assertions. This is a justified test-oracle correction, not a weakening of FR1 or a change to sanitization. The original failed attempt and red test sources remain preserved.

Evidence: `.superpowers/sdd/00-plan/evidence/final-review/rereview/{rereview-audit.json,rereview-probe.jsonl,commands.md}`. Original review inputs/source/logs were not overwritten. No source/index/HEAD changes, commits, subagents, live requests or broad suite repeats were made during this re-review.

## Declined to judge / limits

- Fresh final whole-suite and final archive results: those follow the source correction and are the parent's separate remaining qualification gates; this report does not claim they already passed.
- Revalidation or migration of historical provider classifications: intentionally excluded by the retained-observation/replay contract; the correction applies to new receipts and documents that boundary.
- Reopening unchanged whole-branch concerns or expanding feature/production scope: this is the requested one scoped re-review. The explicit limits in `final-review.md` still apply, including live NOT RUN and draft-PR-only delivery.

No unresolved review finding remains. Subject to the parent's source-matched final checks and archive refresh, the reviewed branch is ready for the requested **draft PR**.
