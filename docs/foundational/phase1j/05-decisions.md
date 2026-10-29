# Phase 1J decisions

- The attached assignment authorizes execution, independent review and a draft PR. Routine design decisions and review corrections need no additional approval round.
- Baseline: `d68ec667d276a4bd9e1be11613905c628cfea704`, tree `25c2e22f21181fe70d8f2b2f8bc7e44c512ef948`, fetched main including Phase 1I and the user’s TeX formatting fixes (PR19).
- Isolated managed worktree, branch `codex/phase1j-retained-joins`; the original `latex` checkout and untracked GitGuardian cache are preserved.
- Preserve the original K1 contract, C0 contexts and negative evidence. J1 is a new strict-join contract, not a retrospective correction to K1.
- Use independent agents for specification review, model/oracle and final review. The reference model freezes before correspondence inspection of the new context artifacts.
- Use existing public authoring operations and test-only retained-state observation/process barriers. No runtime/compiler/schema/accounting/dependency changes, paid calls or effectful experiments.
- Preserve the user’s TeX formatting; run `tectonic alan-dissertation.tex` once after manuscript work and substantive review are complete. Compiler warnings are acceptable.
- GitGuardian is optional. No scan has been run in this phase. The change contains deterministic pure fixtures, not new credentials or provider integration.

## Preflight and progress

Baseline original model and runtime negative checks passed (meaning the historical failure remains observed); see `evidence/baseline/baseline.json`. Protocol and implementation plan are being prepared for independent review before implementation.

## Implemented result and review corrections

- Independent protocol review approved exact J1 before product code. It exposed
  ordinary Send/first-authorization separation and shared-scope effects of static
  peer Failure. The chosen peer always succeeds locally after reporting the true
  S outcome; root decides semantic failure only after strict joining.
- The finite model preserves root lifetime/resource demand but finds report
  source closure pending states in every original configuration. Keep the simple
  protocol and counterexample; do not add handshakes, alter authorization policy,
  or silently impose delivery-before-terminal fairness.
- Fixed Store.Run samples complete and four process cuts recover. These are
  separately scoped positive observations, not evidence against the modeled
  failure. The new question is required communication's source-authority lifetime.
- Existing operational bounds200 steps/200 dispatches were sufficient. Semantic
  I/R limits and physical capacity1 were not increased for primary cases.
- Raw public Provision program catalog order is nondeterministic. Frozen raw
  bytes retain provenance; reproduction compares programs keyed by ID and exact
  program/template bytes, preserving ordered instruction arrays.
- Scoped context review REJECTED a test that used whole-Deployment equality
  without accounting for this catalog order (9 failures in50 repeats). Correct
  the test comparison, preserve the RED and exact-byte checks, then rereview.
- The independent oracle is tightened for distinct extra reports and report
  consumption before candidate consumption under the already frozen J1 contract.
  Preserve its prior bytes and new REDs; rerun dependent qualification with source
  identities rather than attributing old runs to the changed checker.

## User-requested upstream rebase

The user reported a separate manuscript math correction during qualification.
Fetched main and rebased onto `5764ec0b5679c72c80e56c44fd9e4198c986b91c` (tree
`729e93d5d83422307023196e8a7c5677fb8e71f1`), including PR20. A temporary stash
preserved all524 phase-file hashes; both manuscripts matched upstream exactly
outside the new J1 additions after application. No conflicts or code changes.
The original experimental baseline remains d68ec66, distinct from this integration
base. The new text no longer labels the separately repaired structured-loop
argument open. The original frozen J1/plan and historical evidence stay unchanged.

## Final disposition

Independent whole-change review returned APPROVE with no unresolved actionable
findings. Its runtime-path metadata finding was corrected and checked against
the logs; no scientific result changed. The single end-of-phase Tectonic build
passed with accepted warnings, and the new text/continuation were visually
checked. Final source and evidence hashes are recorded for draft-PR publication.
The mathematical correction remains inherited unchanged from the integration
base. The unresolved research boundary is ordinary report source authorization
lifetime, not the already repaired structured-loop argument.
