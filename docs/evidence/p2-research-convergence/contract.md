# Research/runtime convergence contract

Frozen before integration implementation, 2026-11-04.

The user requires the research department's runtime update and P2 optimization
to converge, and authorizes a separate PR. Deliver a separate draft; do not merge
into the default branch, release, or rewrite either reviewed history.

## Inputs and chosen approach

- Research/runtime main: `1fdd4c9e1720bf60b344914bcbc3d0938ba8d678`.
- P2 frozen delivery: `49101122c31b0d790d364831f044bb996d2649fa`.
- Common ancestor: `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315`.
- P2 production implementation: `2fe47cb75f57bf3f7f79779dc1841c9b9b8f28d0`.

Preserve PR11 and its original evidence. Start a separate branch from the verified
research main and join the P2 history with a distinct local integration commit.
Both inputs remain ancestors. Their changed paths are disjoint; any unexpected
conflict requires source inspection, not automatic resolution. Rewriting P2 or
silently applying its old measurements to the combined tree would lose evidence
identity; leaving the histories unqualified would not satisfy convergence.

This is integration and qualification, not another execution feature. The P2
map remains local to one stable readRootCompletions call, keyed by full RunID.
Mutable authority, resource, receipt, result and admission checks remain intact.
The research request-bound authority protocol and historical format rules remain
intact. Formats13/14 intentionally exclude continuations/reconciliation/templates;
no combined delegated-authority × bounded-series capability is implied.

## Permitted correction

The unchanged competing-read debt test has recorded failures on the baseline and
P2 delivery. A narrowly scoped test-only change may establish the exact intended
adopter at the existing runner.received/pre-account barrier before installing the
same settlement-abort trigger. Keep the duplicate transport blocked, use exact
intent/subject/generation identity, acknowledge real boundaries, and release on
cleanup. Preserve every debt, receipt, duplicate, publication, recovery, contact,
resource and error assertion. Do not add sleeps, rerun-until-green, runtime retries,
refunds or softened expectations. Preserve old failures and any new failures.

This schedule correction does not prove the cause of all historical failures.
A production correctness finding requires an independently reviewable correction
within the governing runtime contracts; do not silently extend this test fix.

## Qualification and falsifiers

Freeze the combined source and bind all new command logs/results to its exact
commit, tree, toolchain and before/after source hashes. Run focused P2 and authority
ordinary/race suites; complete ordinary, integration and integration-race suites;
real SIGKILL/reopen histories; exact advanced-main/combined HTTP and authority
comparisons; independent external and historical readers; reciprocal-delegation
research corpus/checker; build, vet, module/tidy/format and root Staticcheck.
Record every nonzero exit and skip without relabeling an earlier source run.

A causal/authority/resource difference, failed invariant, unexpected format
acceptance, duplicate physical contact or invalid historical judgment falsifies
convergence. Stop the affected route and preserve a discriminating history.
Independent reviewers inspect source and decisive histories. New source changes
invalidate qualification of that changed tree and require appropriate reruns.

Old P2 medians, allocations and total SQL counts remain measurements of their
original frozen sources. Advanced main adds authority reads even on feature-off
paths. This integration does not claim those old values for the combined tree.
A later performance claim requires a new controlled paired campaign. Query-count
regressions may establish the retained local optimization without such a claim.

Publish the P2 index/verifier and failed-publication records via a verified Git
blob if the attachment URL is unavailable. All twenty published raw volumes must
still reconstruct exactly the sealed14,032 original files. Publication metadata
must not alter the original sealed inputs, tested source, or failed evidence.

## Done

A draft integration PR contains both input histories, only justified/reviewed
additional changes, a frozen qualification report, exact source identities,
independent review dispositions, verified P2 evidence access, and explicit open
limitations. Any remaining gate failure is an open obligation, not an all-green
claim. No merge or release occurs.
