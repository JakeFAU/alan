# Task 1 fix round 1 independent scoped re-review

Reviewed 2026-10-25. Scope: R1 generated-output bound and the nonblocking receive-coverage note from `task-1-review.md`; inspect for breakage introduced by this correction. Read the complete fix delta in `task-1-fix-1.diff`, amended report, original task brief/review, and retained RED/GREEN/full/race/vet logs. All eight current-file SHA-256 values match the frozen fix package. No source edits, subagents, or redundant test reruns.

**R1: ADDRESSED.**

**Receive-coverage note: ADDRESSED.**

**Specification verdict: PASS for Task 1. Quality verdict: PASS for Task 1.** No blocking scoped findings remain; no fix-introduced correctness defect was found. Approval remains for the task-scoped durable interpreter/store/replay slice, not completion of Phase 1D.

## R1 disposition

`Limits.MaxOutputBytes` explicitly bounds the complete JSON intent array and defaults to 1 MiB. `Reduce` enforces it before returning a result, so the existing Store path rejects before consumption/history/intent/projection writes. Replay passes through the same limit while retaining exact descriptor comparisons.

I checked the size-accounting logic against the actual generated representation: the array brackets and inter-intent commas, descriptor with empty payload-object braces, and each key/colon/value/comma are all counted exactly. Generated intents always use a nonnil payload map. Map iteration order does not affect total encoded length or the limit decision. The raw-string prechecks are lower bounds on standard-library JSON encoding size; individual temporary encodings are bounded without assembling the amplified array. The report correctly distinguishes this bound from a claim that total peak memory is exactly MaxOutputBytes.

The original eight-reference input is preserved without reducing its size or excluding it as invalid. Retained RED evidence reproduces the original acceptance and failed persistence guard. GREEN evidence covers rejection of that original input, successful below-limit output with independently inspected descriptor size, exact/one-byte-below limits with escaping, unchanged execution ledgers on rejection, selected receive retention on the shared failure path, and larger explicitly configured output accepted/replayed only under an adequate bound. The original reviewer probe/log are retained beside the report.

## Receive coverage disposition

The added one-field changes independently exercise kind, provenance, source, source generation, obligation, receiving subject, and receiving generation; the prior wrong-operation case remains. Each requires ErrIneligible, and retained expanded GREEN output shows all cases passing. The report now accurately distinguishes these post-fix coverage additions from the output-bound RED cycle.

## Verification and continuing boundary

The frozen logs report focused tests passing, `go test ./...` passing (historical package cache disclosed), new-package race passing, and vet with no diagnostics. Source hashes matched the reviewed package; production changes are limited to `program.go`, with transaction, replay, schema, exclusivity, and crash harness code unchanged.

The prior review's positive findings continue to apply. Explicitly deferred lifecycle reconciliation, protocol authentication/admission, runner, endpoint, independent policy conformance, and complete Phase 1D crash qualification remain later work and are not silently approved here.
