# Task 2 independent review

## Verdicts

**Specification compliance: CHANGES REQUIRED.** The implementation covers the declared Task 2 boundary, but accepted author definitions can compile to different control/value semantics. Findings R1 and R2 must be fixed before the authoring slice is accepted.

**Code quality: CHANGES REQUIRED.** The decomposition into author/compiler, pure schemas, immutable artifact bytes, and the thin existing-runtime facade is appropriate. However, last-op inspection is insufficient control-flow analysis for join placement, and persisted identifiers are not validated consistently with persisted values. These are reproduced correctness defects, not stylistic preferences.

## Findings

### R1 — P1: trailing labels let the selected then branch execute the else branch

**Location:** `author/compile.go:180-188` (`compiler.expand`, conditional join omission).

**Scenario:** `IfEqual(true, true, [Goto("join"), Label("join")], [Failure()]); Success()` compiles without error. The jump targets the position of the trailing label. Since the last emitted opcode is `jump`, lowering omits the jump over the else branch; the label therefore resolves to the else branch's `Failure`. Public Create/Run and offline Replay both produce `failed`, although the true branch should continue to the following `Success`. The accepted artifact has branch target 1, jump target 2, and `failed` at 2. This also affects reachable trailing labels after more complex terminal/jump paths; last emitted opcode does not establish whether the lexical branch continuation is reachable.

**Action:** Model branch fallthrough/continuations explicitly or otherwise prove that no reachable control edge targets the branch-end boundary before eliding the join. Preserve the existing two-terminal-branch case without creating invalid unreachable targets. Add executable regressions for then-arm trailing labels reached by forward Goto, nested trailing labels, and repeat exits within conditional arms; assert selected behavior and replay, not just successful compilation.

**Evidence:** `evidence/task-2-review/main.go`, `trailing-label.artifact.json`, `output.log`, and the exact reviewed source archive. The original stdout is `trailing-label compile_error=<nil>` followed by `expected=succeeded actual=failed`.

### R2 — P2: invalid UTF-8 variable names silently alias during artifact encoding

**Location:** `author/compile.go:320-323` (`write` validates only a nonempty name), with serialization at `Compile`'s `executor.EncodeProgram` call.

**Scenario:** Set the variable named byte `0xfe` to 1, set the distinct variable named byte `0xff` to 2, then compare the first variable to 1. Compilation accepts both distinct Go string names and the reference. JSON encoding replaces each invalid sequence with U+FFFD, making both persisted destinations and the reference identical. Public Run and Replay consequently take the false branch and report `failed` instead of `succeeded`. Both the wrong persisted artifact and output are retained. This is analogous to the template UTF-8 issue already guarded by the implementation, but affects program semantics rather than only text fidelity.

**Action:** Reject invalid UTF-8 identifiers before encoding, using one consistent policy across variable destinations/references, labels, effect/port names, capabilities, obligations, and other persisted identity strings. Do not normalize distinct identifiers silently. Add a regression requiring a useful compile error for these exact inputs and audit metadata/provisioning identity strings for the same serialization loss.

**Evidence:** `evidence/task-2-review/main.go`, `utf8-identifiers.artifact.json`, `output.log`, and the reviewed source archive. The resulting projection contains one replacement-character variable with value 2.

## Reviewed coverage and positive checks

- Read Task 2 brief/report, review-package diff, authoring-surface notes, full authoring/effects contract, and requirements sections 1, 3, 4, 8, and 9A. Read the production author, schema, facade, executor metadata implementation, relevant existing-runtime changes, and compiler/application/runtime/recovery/external-consumer tests. Applied the Go safety, structs/interfaces, and code-style review guidance. No source or existing test edits, commits, or further delegation were performed.
- All 19 owned source/test hashes match the supplied frozen manifest, verified again when archiving. Retained exact reviewed sources and diff alongside each probe's input, artifact, and stdout.
- Authoring emits the existing generic finite instruction basis. The facade delegates to the existing Store/Run/Replay and adds neither callbacks in deterministic reduction nor another scheduler. Repeat uses finite expansion, instruction/visit/nesting/artifact-byte bounds, and forward-only resolved jumps. Definite assignment intersects reachable predecessor states; typed binds reject missing fields and duplicate destinations; request-effect identity is tracked. R1 is a lowering defect before that analysis, so downstream validation cannot recover the intended lexical control flow.
- Artifact storage is private immutable string data; byte/metadata accessors and Provision copy state. Canonical schema bytes and IDs, versions, templates, positions, and labels are embedded. Supported semantics are checked at encode/create/load; unknown compiler or semantic versions have explicit recovery tests. Legacy Value representation and legacy scalar validation remain unchanged; optional metadata is omitted from legacy encoding, with an exact byte fixture.
- Closed flat schemas enforce required fields, scalar kinds, string byte bounds/enums, and no extra fields. Initial inputs, ordinary ingress, receive payloads including unbound fields, successful output, and accepted admissions during replay have validation paths. Effect payload schemas exclude runtime-owned `valid`/`disposition`; invalid result envelopes require typed defaults. Dynamic effect payload dispatch validation remains explicitly assigned to Task 3.
- Static Application provisioning creates fixed subjects/runs/grants, verifies used send boundaries against compatible peer receives, deduplicates artifacts, and sorts generated output. The public consumer defines one caller and two instances of the same helper with distinct wiring. It uses no internal import, raw instruction arrays, PCs, or SQL.
- The external harness creates a new `external.example/consumer` module in `t.TempDir`, builds an executable, then invokes separate create/resume/replay processes. Resume and replay do not compile definitions. The retained race log explicitly shows the child `go build ... -race .` and all three modes passing. This verifies ordinary-message composition, not pending adapter transport execution.

## Validation and scope limits

Inspected supplied JSONL evidence: default run has 405 passing test/subtest events and zero failures; integration-tagged run has 414 passing events and zero failures. Inspected the external race log. Did not repeat the full suite or race qualification. Ran only the two focused public-consumer diagnostic scenarios above, plus nonmutating `gofmt -l` and `git diff --check` (both clean). The diagnostic command exits zero after printing the mismatches; it is not a passing assertion test.

Task 3 dispatch, provider functionality, corpus retention, application budgets/citation policy, and later phase qualification remain outside this review. Task 2's `Request` intentionally emits an unresolved adapter intent; its missing transport is not a Task 2 finding. This review does not claim provider/corpus completion, unbounded fairness, business exactly-once behavior, production readiness, or any broader SDK stability. Historical research/runtime/negative-control evidence was not modified. Review evidence is a separate current-source comparison.
