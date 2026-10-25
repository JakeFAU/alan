# Task 1 independent specification and quality review

Reviewed 2026-10-25 against the frozen package in `task-1-review-package.diff`. All eight source/dependency SHA-256 values matched the current files. Read the complete new runtime and test sources, task brief/report, binding durability contract, and relevant retained Phase 1C findings. No source files were changed, no agents spawned, and no already-reported full-suite commands repeated.

**Specification verdict: FAIL pending R1 (narrow resource-bound requirement).**

**Quality verdict: CHANGES REQUESTED for R1.** The atomicity/recovery/replay and core crash-controller design otherwise meets this task's scope. These verdicts do not assess unfinished Phase 1D runner/lifecycle/endpoint work.

## R1 — Bound generated intent descriptors before persistence (P2)

Locations: `internal/executor/program.go:94`, `program.go:374`, `internal/executor/store.go:584` and `store.go:601`.

The explicit limits cover the program, projection, and incoming observation, but no limit covers the generated effect/intent descriptor. An ordinary program can refer to one bounded local string from many payload fields. Each reference is small in the immutable program; the resulting serialized intent repeats the full string. `Reduce` accepts it, and `Commit` serializes and persists it twice, once in history and once in the intent ledger. This defeats the plan's explicit serialized-payload/per-step operational bounds and permits large memory/disk amplification within one otherwise valid step. It is separate from the disclosed, intentionally unbounded number of retained history records.

I reproduced this using only legitimate public types/APIs, with an isolated copy of the frozen source under `/tmp/alan-task1-review-probe`:

- Default limits: 1,048,576 bytes each for program/data/observation.
- Valid program: 1,163 bytes, consisting of Emit and End.
- Valid initial projection: 500,092 bytes; `text` is a 500,000-character string.
- Emit payload has eight keys, each `Ref("text")`; no special roles, invalid identity, external SQL, or corruption is involved.
- `Reduce` succeeds with an intent-array encoding of 4,000,627 bytes.
- `Create` and `Commit` both succeed; independent SQL inspection finds a persisted intent descriptor of 4,000,625 bytes.

More payload references amplify further while the program remains below its own limit. The finding is **absence of an applicable generated-payload bound**, not a claim that `MaxDataBytes` was documented to include every other relation.

Exact executable probe: `/tmp/alan-task1-review-probe/internal/executor/review_probe_test.go`. Exact output: `/tmp/alan-task1-review-probe.log`. Command: `GOCACHE=/tmp/alan-phase1d-go-build go test -run TestReviewEmittedPayloadBound -v ./internal/executor` in that isolated copy. The probe intentionally fails after reporting accepted/persisted sizes; the ordinary tests were not rerun.

Requested correction: define and document an intent/output byte limit, or explicitly apply a chosen existing limit to generated descriptors. Enforce it before any execution writes. Prefer bounded size accounting or bounded serialization so the rejection itself does not allocate an arbitrarily amplified JSON buffer. Ensure replay applies the same semantic limit without weakening descriptor equality. Add a focused regression with legitimate repeated references that requires `ErrLimit`, unchanged projection/sequence/history/intents, and no consumed observation where applicable; include a below-bound successful counterpart. Preserve the original input and finding in qualification.

## Nonblocking evidence precision

The report says tests cover exact kind/provenance/source-generation/obligation/operation/run receive matching. The source guard checks all these fields, but `TestReceiveExactCorrelationAndTerminal` independently perturbs only operation. Either narrow the statement to distinguish reviewed code from exercised tests, or add one-field-at-a-time negative cases when this matching code is next changed. This is not a second correctness finding.

## Positive findings and scope boundaries

- The instruction language is closed and finite, with one generic instruction per commit; Emit creates immutable descriptors only. I found no callback-based program semantics, policy role/name branch, runtime retry-budget handler, or generation-based manufactured success.
- Execution authority checks precede reduction; consumption, history, current projection/sequence, immutable intents, and ordinary terminal obligation are in one SQLite transaction. Deferred consumption linkage and rollback tests cover genuine intermediate-write failures.
- Recovery holds exclusivity, validates the ledger in one transaction, and only then advances open-run epochs. It does not reset ordinary data or create a fresh logical generation.
- Replay uses a separate read-only/query-only connection and consistent transaction; it cannot call live Open, create a missing store, dispatch, or advance epochs. It is correctly described as consistency checking with the same reducer, not independent conformance.
- Filesystem handling canonicalizes symlinks, rejects multiply linked database files, holds a persistent sidecar flock, and checks settings/engine identity on each pinned connection. Live replacement and hostile aliases are explicitly excluded rather than claimed safe.
- The crash harness pauses inside the actual Commit transaction via a test-only trigger for the pre-commit case, and acknowledges after Commit returns for the post-commit case. The parent validates scenario/PID/store/phase/run/epoch/sequence/intent before SIGKILL and checks signal exit. Recovery uses the retained database; the controller does not reconstruct ordinary state.
- Private observation insertion is correctly disclosed as trusted storage, not transport authentication. Public admissions/provenance enforcement remain Task 2/3 work.
- Reserved lifecycle relations and the required later replay/terminal-linkage extension are explicitly documented. Their incompleteness is not counted against Task 1. Final Phase 1D approval still requires lifecycle-derived closure, zero-step successors, durable denial/rejection, endpoint uncertainty, same-obligation second-crash evidence, independent policy checks, and source preservation verification.
