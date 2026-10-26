# Task 4 scoped R1 re-review — frozen addendum

**R1: ADDRESSED.** **Specification verdict: pass for the scoped Task 4 repair. Code-quality verdict: pass; no new blocking findings in the fix diff.** Task 4 may proceed to Task 5 qualification. This does not claim Task 5 or the live gate is complete.

## Source and scope

Reviewed the five files frozen by `evidence/task-4/review-round1/after-source-sha256.json`, the exact `fix.diff`, the appended implementation report, the original independent review/counterexample and the final scoped race log. All five current-source hashes match the fix manifest. Inspected the underlying `runtime.Inspect` transaction and `runtime.Open` validation relevant to the fix. Parent's separate README/qualification/contract changes are outside this R1 re-review and remain for whole-branch review.

The original `task-4-review.md`, original probes, inconsistent View and original evidence manifest remain unchanged. This addendum records the repair independently; it does not overwrite the original finding.

## Why the fix addresses R1

`examples/evidence/store.go:103–108` now performs exactly one database read via `runtime.Inspect`. The private `inspectionRuns` function at lines 176–224 decodes run identity, program, epoch, sequence and projection from that snapshot, joining subject revisions from the same snapshot. Config and public Result derive from those projections. Logical request counts, allowed/denied reservations, results, unknown IDs and the exposed Snapshot all derive from that same returned value. There is no second replay/read transaction and no removal or weakening of the visible consistency comparison.

The row decoder rejects missing/null required columns, invalid identity/sequence metadata, incompatible JSON field types, missing subject references, malformed projection JSON and null projections. It consumes the known public inspection row shape and preserves int64 metadata without float conversion. The new tests compare the complete returned run metadata/projections, public result/config and accounting against the same raw snapshot while ordinary computation advances.

Removing the second `Replay` also removes historical reconstruction from application inspection. This is explicitly stated in the function comment, runbook and appended report. The decoder is not claimed to validate historical causality or every possible corrupted projection. This boundary is acceptable for this fix: the requirement is coherent inspection, separate CLI/runtime Replay still validates history, and `runtime.Open` continues to call full `validateLedger` before activation. No model/tool execution authority or application validation branch moved into inspection.

## Independent verification

Environment for both commands: `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`.

1. `go test -count=1 -race -tags integration -run 'TestInspectConcurrentProgressUsesOneCommittedView|TestInspectionRunsRejectMalformedRows' -v ./examples/evidence` — PASS, 4.906s. All twelve malformed-row cases passed. The controlled HTTP clarification case checked 18 progressing Views plus initial/held/final snapshots; the lookup case checked 36 progressing Views plus initial/held/final snapshots. Final dispositions and logical counts were clarification 1/0 and answer 2/1.
2. Reran an exact byte-for-byte copy of the original broad independent diagnostic `inspection-original-probe.go.txt` with `go run /tmp/alan-task4-rereview-original.go`. It completed its bound without a mismatch and intentionally ended with `panic: no mismatch found in bounded probe` (child exit 2, go-run exit 1). This is absence of the original counterexample within its bounded schedule, **not an assertion-test PASS**. The diagnostic was not changed to manufacture successful exit semantics.

Read, without needlessly rerunning, the implementer's final scoped race evidence for actual race-built CLI subprocess execution, offline replay, unknown no-repeat recovery and denied-attempt accounting. No broad test suite was repeated. No implementation, test-source changes, commits or subagents were made by this reviewer.

## Evidence and remaining limits

New evidence is under `evidence/task-4/independent-review/rereview1/`: independent focused race log, unchanged diagnostic source/log, copied fix-source manifest, and SHA-256 artifact manifest. This addendum is included in that new manifest. The original evidence manifest was not rewritten.

Concurrent tests cover observed schedules; the structural single-read fix removes the split-transaction mechanism independently of schedule luck. Full-phase process-kill/contact evidence and negative controls remain Task 5 work. Live gate remains NOT RUN; no live credentials/model were used. This is a scoped fix review, not a renewed review of unrelated phase changes or a production-readiness claim.
