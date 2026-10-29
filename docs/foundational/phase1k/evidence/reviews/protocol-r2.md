# Phase 1K protocol/profile rereview r2 — APPROVE

Spec compliance: **APPROVE**. Design quality: **APPROVE**.

Scope: R1/N1 corrections and new breakage introduced by their diff. This approves
the revised experiment design only. It does not qualify implementation, model
counts, public-driver executions, terminal accessibility or crash recovery.

## Exact reviewed identities

- Plan `00-plan.md` SHA256:
  `eca229f7d3de8f462ab138cf895e851d69d0ce8f7da20cb9630945f3116e1adb`.
- Contract `01-communication-contract.md` SHA256:
  `075debb7c6596d70fb8438367f30399feefde326a6005e2be63603583178d544`.
- Fix review package SHA256:
  `e00dc1cbb8619b6ff17a1f04f2f52f006a2555a75db388dbcff86cde654176a5`.
- Appended worker report SHA256:
  `398a4c48fe799ab431d899348f7c24deb268c6c8e5c33088e26d633143044161`.
- Preserved r1 report SHA256:
  `cbb35f0822174c75ccd1882d3de98d5f927de9a95109260076401bfde11a0981`.

These were independently computed from the current files. Preserved r1 plan and
contract hashes remain `9e89954ad0d327b9c2be7441b9a9f27c4924a7943351710192ef9f502da78346`
and `0b9013128dd84790da1d3fe9466fcb9fb4b4ecbfb1cf850576d50064076ca112`.
All 19 full-file source hashes in the r1 source appendix still match. The original
REJECT remains intact; this rereview resolves its findings without rewriting it.

## R1 — addressed

`01-communication-contract.md:315–328` now requires a refused first authorization,
a real refused admission under the original runner transport identity, and a
dispatcher acknowledgment before checkpoint/SIGKILL. `00-plan.md:125–134` repeats
the same required implementation obligation.

After fresh Open, the driver repeats both authorization and delivery using those
exact identities. The required comparison includes byte-identical authorization
descriptor/result/source snapshot and admission descriptor/result, unchanged
acknowledgment linkage, no replacements or invented observations/consumption/
charges/permissions, and separate root-local and aggregate pending observations.
The text explicitly rejects an authorization-only prefix as a substitute.

This corrects the substantive mismatch: the planned post-reopen delivery now
exercises the existing-admission path at `internal/executor/messages.go:408–415`,
in addition to authorization reuse at lines 112–123. Pre-kill acknowledgment is
legal even for refusal (lines 231–260). It removes automatic queue eligibility,
but the cut expressly resumes through the public driver; `DeliverMessage` at
lines 180–225 does not prohibit an explicit retry merely because an acknowledgment
exists. Therefore the correction does not introduce a scheduler/control conflict.
Actual retained-row and process evidence remains a later qualification gate.

## N1 — addressed

`01-communication-contract.md:261–279` now distinguishes context RootPC/PeerPC
0..7 from operation PC0..8 and spells out both conservative arithmetic steps:
base rank 231 and extended cap 23202. Those restrictions agree with the source
assignments inspected in r1. The requirement to check every progressing edge,
including revisits, and to revise/review any new control coordinate is preserved.
This is a corrected finite-domain derivation, not an executed graph census.

## Diff scope and remaining gates

The complete current-versus-preserved diffs contain only the revision header,
refused-cut policy/table/plan synchronization, and PC-domain/rank clarification.
No new actionable issue was found in those corrections. The other pre-experiment
boundary assessments in r1 continue to apply to unchanged design content.

Executed read-only document inspection, SHA256/source-manifest checks, and two
`git diff --no-index --unified=3` comparisons. Each diff exited 1 because the
intended changes exist. No model/context/runtime experiment, Go test/build,
process execution, TeX build, source edit, commit or nested agent was used.
Only this rereview report was written.

The controller may freeze these approved hashes and proceed with the planned
experiments. Public-call legality/compilation, model-artifact correspondence,
raw causal mapping, exact process predicates and final claims remain subject to
the later independent implementation and qualification reviews.
