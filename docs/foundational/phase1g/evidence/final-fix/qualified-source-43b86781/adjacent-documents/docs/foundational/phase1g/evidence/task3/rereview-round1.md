# Task3 scoped rereview — fix round1

**R1: ADDRESSED. Spec compliance: Approved for this scoped fix. Code quality: Approved.** No new findings or observed out-of-scope findings.

Reviewed `6a5d9ea182e760c61149feaff47fe8dc0ff72f93` → `75eb9c2978d539d9a7c04035e8140d7dd7e1caa0`. This review covers the canonical JSON field guard, its regressions and retained evidence only. It resolves the sole Important finding from the original Task3 review; it does not qualify Task4 or repeat the runtime/whole-branch review.

## R1 resolution and strengths

- `cmd/alan-compose/main.go:285` permits exactly `item_id`, `number` and `question` in the existing token pass. Any case alias or unknown field now errors before the case-insensitive struct decoder can overwrite an identity/payload. The exact duplicate-key check at `cmd/alan-compose/main.go:282` remains intact. The fix is narrowly placed at the faulty boundary without runtime, artifact, schema or policy changes.
- `cmd/alan-compose/main_test.go:101` adds two canonical positive cases with exact schema-value comparisons and eight rejecting alias cases. It covers identity, number and question; all-case-only names; and identity aliases in both orders. The canonical question fixture also verifies order independence. Existing malformed/null/exact-duplicate and retained-application CLI tests remain present.
- `docs/foundational/phase1g/evidence/task3/review-fix1/04-permanent-canonical-guard-red.log:1` shows both canonical controls already passed before the fix and all eight aliases reached the intended failing rejection assertions. `docs/foundational/phase1g/evidence/task3/review-fix1/05-cli-package-green.log:1` records the entire CLI package passing after the fix, including those cases, initialization/recovery, inspection and replay (actual exit 0; 0.644s).
- The original pointer-form, row-form and case-only probes retain their inputs and command lines across RED/GREEN. `docs/foundational/phase1g/evidence/task3/review-fix1/06-original-alias-pointer-green.log:1`, `07-original-alias-rows-green.log:1` and `08-original-case-only-green.log:1` record explicit `unknown item field` errors and passing probes. The logged `records_error=expected list` is expected diagnostic output from the unchanged row-form probe inspecting the zero value returned with the rejection; it is not a test/runtime failure.

## Evidence verification

Read the exact code companion once and inspected the full package's supplemental logs, command records, original probe sources, overlays and verification record. Repeated source manifests were summarized and compared by hashes. No untouched implementation was reviewed and no test commands were rerun; the fix and existing focused evidence raised no additional concrete doubt requiring a probe.

All 174 entries in the final source manifest match the checkout. Final-source identity is `b1139d2b786a5a4068088e0bf3618be39d50666be2e0d1d8403da360d095015c`; command records 05–08 all retain actual exit 0 at that identity (`docs/foundational/phase1g/evidence/task3/review-fix1/commands.json:1`). Records 01–04 retain their original actual exit 1 and earlier source identities, so historical reproduction remains separate from current-source regression. Formatting/whitespace checks retain exit 0 with empty output (`docs/foundational/phase1g/evidence/task3/review-fix1/verification.json:1`).

Independently calculated preserved-input SHA256 values match `original-inputs.json:1`:

- Original alias pointer: `0bb3b4a877b59f72d458d0e3c36770174c13c01e14e73ba3a64c795cfd41a9cd`.
- Original alias rows: `2b13e4d4e5eb1c8ba1cb65ef4259120b1bfa113bf463693efcf684060526a7b6`.
- Original case-only: `4f1dcdba0e97e84a05ee4eab0f6c95ecf006eaca36a2919782adfb04c7ab763c`.
- Original review: `1f18f8dc7ea4700cd9935999448594b0fa4d722f2556d3fedb129e38df9eb2dd`; original counterexamples and verdict text remain unchanged.

## Scope limits

No new Critical, Important or Minor findings in the fix. No out-of-scope findings observed. Earlier broad checks remain evidence for their earlier source identity, not fresh full-tree validation. Task4 process-kill, denied-network replay, final archive and whole-tree gates remain pending under controller authority. No source/index/branch changes, subagents, live requests or provider credentials were used; the only checkout write is this report.
