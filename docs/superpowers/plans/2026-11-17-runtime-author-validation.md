# Runtime Author Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port sounder schema containment and invocation-handle provenance checks into the runtime author compiler.

**Architecture:** Adapt the originating research compiler validation and helpers while retaining public builders and artifact semantics. Historical malformed programs remain fixtures for runtime boundary tests.

**Tech Stack:** Existing Go module, author/schema/runtime packages, SQLite tests and Staticcheck.

**Spec:** `docs/superpowers/specs/2026-11-17-runtime-author-validation-design.md`.

## Global Constraints

- Work only in the isolated worktree on `codex/runtime-author-validation`, after the scheduler port review.
- Originating research source: `28cf2e28236026e3ea00fc43e0ee3367de7e7ae2`.
- No dependency, storage-format, runtime-primitive, compiler-tag or semantic-version changes.
- Reject unsound newly authored definitions without rewriting retained artifacts or weakening runtime checks.
- Preserve valid direct outing artifacts and pre-port Store inspection/replay compatibility.
- Read research AGENTS.md and memory/adversarial/known_weaknesses.md. Never modify research or read .envrc.
- Staticcheck is the lint gate; retain RED findings, exact source provenance and independent review results.
- Root owns branch/PR transitions. Commit only this compiler port and its receiving-repository regressions/docs.

## Review Focus

- Partial enum overlap and wider byte bounds must be rejected rather than treated as existence of a valid value.
- Different invalid values at branch joins must not erase constraints; valid guarded alternatives remain usable.
- Invalid adapter results carry defaults and must not pass a narrow sink without a valid retained guard.
- Copied/list-carried handles must retain their declaration; strings, overwrite and cross-iteration laundering remain refused.
- Historical malformed artifacts still exercise runtime guards; valid pre-port planner Stores must remain inspectable.

### Task 1: Port and qualify schema/handle validation

**Files:**
- Modify: `author/compile.go`, `author/compile_test.go`, existing retained handle tests.
- Create/adapt: `author/string_bounds.go`, `author/handle_provenance.go`, `author/compiler_soundness_test.go`, `author/testdata/prechange-wrong-invocation.json`.
- Modify as needed: `runtime/runtime_test.go` and narrowly scoped `runtime/testdata` historical fixtures from the originating commit; existing example definitions identified by actual compilation failures only.
- Create: `docs/research-ports/author-validation.md`.

**Interfaces:** Public author builders and artifact metadata/semantic tags remain unchanged. The new helpers are private validation; no delegated-app API is introduced in this port.

- [ ] Inspect originating source/diff and receiving compiler. Capture valid current outing artifact identities and a complete fixture Store before production edits, for post-port inspection/replay.
- [ ] Bring meaningful compiler regressions first and record RED: partially overlapping enums, excessive bytes, invalid branch joins, unchecked reply defaults and forged/wrong-declaration invocation handles. Separate historical artifact tests from newly compiled input tests.
- [ ] Adapt the source helpers and compiler flow. Preserve runtime-only features absent from research, including retained continuation/handoff support. Reconcile only genuine affected authored definitions, with the exact reason and effect on artifact bytes recorded.
- [ ] Keep raw source-bound historical fixtures for defenses that can no longer be reached through accepted author definitions; do not delete or invert their runtime expectations.
- [ ] Run the focused new tests normal/race, valid collection/handle controls, historical fixture tests and pre-port planner inspection/replay. Every expected refusal must be tied to the intended static or runtime boundary.
- [ ] Run `go test -count=1 -timeout=60m ./author ./runtime ./compose ./examples/outing-planner`, `staticcheck ./author ./runtime ./compose ./examples/outing-planner`, `go build ./...` and one final `go test -count=1 -timeout=60m ./...` after edits.
- [ ] Record RED/GREEN, provenance, byte/Store compatibility results, commands and limits in the port note and provided report. Inspect status/stat/full diff and formatting; commit only this logical compiler port. Independent review follows.
