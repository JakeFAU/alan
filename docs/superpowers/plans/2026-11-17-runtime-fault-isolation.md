# Runtime Fault Isolation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the research scheduler correction so unrelated eligible work survives a run-local fault.

**Architecture:** Adapt the research change to the existing executor's ready scan and receive-candidate error classification. Preserve the runtime fork's authority, caching and dispatch paths.

**Tech Stack:** Existing Go module, SQLite and test-only loopback fixtures; Staticcheck.

**Spec:** `docs/superpowers/specs/2026-11-17-runtime-fault-isolation-design.md`.

## Global Constraints

- Work only in the current isolated worktree, on `codex/runtime-fault-isolation`.
- Originating research commit: `6913566b01679090546721498b4609355d8ffc79`.
- No new dependency, storage format, program semantic version or authority primitive.
- Retain faulty state and real errors; only local reduction/candidate errors permit scanning onward.
- Integrity, loading and Store errors remain fail-fast.
- No rewriting retained records or replaying external effects.
- Read research AGENTS.md and memory/adversarial/known_weaknesses.md; never edit research or read .envrc.
- Use Staticcheck. Preserve RED evidence and independent review findings. Commit only this logical port.
- Controller handles branch transitions and pull request publication; never merge automatically.

## Review Focus

- Fault precedes healthy peer in scan order, across two calls: healthy work progresses while error persists.
- Invalid receive before valid receive: only the actual valid observation becomes consumed.
- All matching candidates invalid: blocked error remains visible with candidate identity.
- Integrity/load error: no accidental broad error suppression.
- Oversized child end: parent may advance, but the child does not gain an illegal terminal commit.

### Task 1: Port and qualify scheduler fault isolation

**Files:**
- Modify: `internal/executor/program.go`, `internal/executor/runner.go`.
- Create/adapt: `internal/executor/runner_fault_isolation_test.go`.
- Modify: `internal/executor/runner_silent_child_test.go` only for the changed progress expectation.
- Create: `docs/research-ports/runner-fault-isolation.md`.

**Interfaces:** Existing `Store.Run`, `Reduce`, ready scan and error identities retain their signatures. New private error classification follows the originating patch; no public API change is required.

- [ ] Read the originating commit diff and matching runtime files. Record any necessary fork-specific adaptation before changing production code.
- [ ] Port the regressions first. Reproduce healthy-peer starvation and first-invalid-observation starvation; record exact RED commands/output under the plan workspace or a named outside-checkout log directory.
- [ ] Adapt the narrow production correction. Preserve cached/program-bound paths and original run ordering; add the concrete integrity and all-invalid controls if existing tests do not cover them.
- [ ] Run focused RED-to-GREEN regression, the silent-child control and corresponding race test. Test both repeated runner calls and replay of the selected prefix.
- [ ] Run `go test -count=1 -timeout=60m ./internal/executor ./runtime` and `staticcheck ./internal/executor ./runtime`. Run `go test -count=1 -timeout=60m ./...` once after final source edits and `go build ./...`.
- [ ] Record original research SHA, exact observable changes, RED/GREEN results, commands, limitations and any fork-specific changes in the port note. Inspect complete status/stat/diff and formatting, then commit this port only.
- [ ] Write the detailed report to the provided task report path; return commit SHA, verification summary and unresolved concerns. Independent review is controller-owned.
