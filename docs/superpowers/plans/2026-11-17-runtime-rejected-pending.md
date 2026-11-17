# Rejected Invocation Inspection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make root pending-invocation inspection match existing v1 settlement after a rejected admission.

**Architecture:** Adapt only the admission predicate and regression from research commit `3d1d63fa6a69f590f145768745793d80f9839268`, preserving the runtime's indexed inspection.

**Tech Stack:** Existing Go, SQLite, executor/runtime tests and Staticcheck.

**Spec:** `docs/superpowers/specs/2026-11-17-runtime-rejected-pending-design.md`.

## Global Constraints

- Root owns branch transitions and publication. Work on `codex/runtime-rejected-pending` after the prior ports are reviewed.
- No composition-v2 semantics, format changes, dependencies or public APIs.
- Refusals remain retained and inspectable; only false descendant pending debt is removed.
- Read research AGENTS.md and its read-only adversarial reference; do not edit research or read `.envrc`.
- Preserve fork-specific indexes, ordering and error propagation.

## Review Focus

- Unadmitted and accepted unpublished operations must remain pending.
- Rejected operations must retain admission/outcome evidence and caller obligations.
- Open/closed roots and replay must agree without a storage rewrite.
- A malformed admission record must remain an error.
- The source commit's broader lifecycle/draining changes must remain absent.

### Task 1: Correct and qualify root pending inspection

**Files:**
- Modify: `internal/executor/composition_inspect.go`, `internal/executor/inspection_pending.go`.
- Create/adapt: `internal/executor/composition_pending_regression_test.go`.
- Create: `docs/research-ports/rejected-invocation-inspection.md`.

**Interfaces:** Existing `RootCompletion.PendingInvocations`, snapshot and replay interfaces remain unchanged.

- [ ] Compare the originating predicate/regression and current settlement rule; record the exact receiving base.
- [ ] Add the regression first and demonstrate the rejected/unadmitted mismatch before production edits. Retain exact RED output.
- [ ] Port only the narrow accepted-admission predicate in both owning pending calculations, preserving load errors and indexed/detached inspection.
- [ ] Check rejected, unadmitted and accepted-unpublished controls through root reports and detached public snapshots; retain original refusal evidence and caller wait before outcome delivery and after delivery/selection as applicable. Check reopen and replay and cover malformed admission errors through an existing or focused test.
- [ ] Run focused normal/race regressions, `go test -count=1 -timeout=60m ./internal/executor ./runtime`, `staticcheck ./internal/executor ./runtime` and `go build ./...`. Final integrated full suites are controller-owned.
- [ ] Record scope, provenance and results in the port note; inspect complete status/stat/diff, format and commit only this port. Write the detailed report to the supplied report path. Independent review follows.
