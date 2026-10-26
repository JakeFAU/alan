# Task 1 scoped rereview — round 1

## Spec Compliance

- ✅ **ADDRESSED:** the sole Important finding from `review.md` is resolved. `schema/collections.go:296` explicitly rejects a non-list field, non-list value, or nil list storage before validation and before the dereference at line 302. Matching scalar pairs now return an error, and malformed list schemas/values still pass through the existing validation path.
- ✅ **Spec compliant for the previously reviewed Task 1 scope, with this repair.** No new requirement gap or breakage found in `a1477e30fa172a44f7ce932458a2847ff88d83d8..4180ba540d9b35ff92b4e1273a6365906cb18d9a`.

## Strengths

- The three-line guard directly establishes the missing precondition without changing valid-list append, bounds, uniqueness, copying, encoding, or reducer semantics (`schema/collections.go:295`).
- The regression matrix verifies matched int/bool/string receivers, schema/value mismatches, missing storage, mixed payloads, wrong tags, and malformed list schemas return errors without panic (`schema/collections_test.go:201`). The valid append test checks the returned row, caller-data copying, and unchanged source (`schema/collections_test.go:234`).
- Retained RED evidence shows all three scalar receiver panics before repair (`26-append-kind-red.log:1`). The focused tests and complete schema package passed cleanly (`27-append-kind-green.log:1`, `28-schema-fix-round1.log:1`). The unchanged original probe now reports an explicit error (`29-review-probe-green.log:1`); its original source and panic output are retained separately (`review-probe/main.go.txt:1`, `review-probe/original-output.log:1`). Evidence paths in this paragraph are relative to this report's directory.

## Issues

- **Critical:** none.
- **Important:** none remaining from the original review; no new issue found in this repair diff.
- **Minor:** none raised.

## Assessment

**Task quality: Approved.** The repair is narrow, correct, and backed by the specific failing regression plus successful valid-input coverage. The original review report remains unchanged as historical evidence.

**Limits and checks:** Reviewed only the exact repair diff and appended implementer report against the original finding and task brief already read. No additional tests were needed: inspection raised no new doubt unanswered by the retained focused runs. No broad rereview, suite reruns, outside-diff source reads, source modifications, agents, commits, credentials, or live calls. This is Task 1 approval, not whole-branch or later invocation/store/application approval.

**Prior verification limitation:** The controller reports resolving the old helper-ID evidence question through the unchanged, unskipped golden test at `examples/evidence/excluding_test.go:51` and the passing package in the retained default-suite output, with named-test evidence to be retained during final integration. That is controller-supplied verification, not an additional source inspection or test run by this rereviewer.
