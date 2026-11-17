# Runtime fault isolation port

The user approved the delegated Weather extension and its relevant runtime
prerequisites. This port asks whether one faulty retained reduction or malformed
receive candidate can prevent unrelated eligible work from progressing.

Runtime base: `5fdfa72c34235ea2b09f969b06dfcdd7c281aa7d`.
Originating research change: `6913566b01679090546721498b4609355d8ffc79`.
Research main: `4a4a7026aead71470111f51e64650ab988ef174e`.
The research manuscript at that main documents the correction around §2.4.

## Contract

- A run-local reduction fault leaves its projection, sequence and observations
  unchanged. Continue the existing ordered ready scan so healthy peers can run.
- If eligible work drains while the faulty run remains blocked, return its
  actual error; do not turn failure into successful completion or silent skip.
- A receive candidate whose binding, receive schema or successor byte bound is
  invalid stays retained and unconsumed. A later valid candidate may be selected.
  If none can advance, retain the first candidate error in the eventual report.
- Artifact loading, integrity and Store errors remain fail-fast. This is not a
  general policy to ignore errors or to retry external work.
- Preserve the runtime fork's caching, dispatch, authority, resource and
  persistence mechanisms. No new storage or program semantic version.
- Historical records are not rewritten. Replay must still check the actual
  selected observation and original committed prefix.

## Implementation and qualification

Adapt the originating `program.go`, `runner.go` and their regression tests to
the separately maintained runtime. Its separately optimized selector in
`static_execution.go` must obey the same fault-isolation rule, retaining errored
ranks for subsequent calls without discarding the guarded fast path.
Copying the research executor wholesale is
out of scope. Keep the port's tests and provenance together in
`docs/research-ports/runner-fault-isolation.md`.

Reproduce the first-run fault ahead of a healthy peer and poisoned receive ahead
of valid input before the production correction; preserve RED output. Verify two
runner calls, retained faulty state, the selected valid input, and replay. Retain
the oversized child boundary's actual behavior without weakening its size rule.
Check integrity errors still stop execution. Run focused normal/race tests,
applicable package tests, Staticcheck and the full ordinary repository suite.

## Delivery and limits

This is its own narrowly scoped pull request from current runtime main.
No automatic merge. The compiler and delegated Weather are separate changes.
The port transfers tested single-host scheduler behavior, not unbounded fairness,
eventual recovery or a general progress theorem. Read the research adversarial
reference before implementation/review; do not modify it or research sources.
