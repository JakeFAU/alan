# Final delivery evidence verification

Delivery head: `2bafdf374b34b99eba29fa3d73552559994f8d95`, tree `8e521a14d26d9e60ab245702d3d9aae32903e876`. Qualified executable source: `cd18daac429786210a0df6df8fbd238e756c528c`, tree `96b31c4ac1329123e4408abe275218b17a07471d`.

This closes only the evidence-verification boundary left pending in `final-rereview.md`. The earlier whole-branch and scoped source reviews remain unchanged, including their then-current qualifications. No new broad source review, tests, provider calls or subagents were performed. Only this report was written; source, index, HEAD and branch state were not changed.

## Completed gates

- Independently parsed the final raw `ordinary.stdout`, `integration.stdout` and `race-integration.stdout` JSON streams under `final-qualification/`. Results are 1,424 / 2,703 / 2,703 passing test events, 18 passing packages each, three packages without tests each, and 4 / 6 / 6 skipped tests, with no failed events. All three `.exit` records are zero. The skip names match the published optional historical cases: four frozen-reader children in all lanes, plus `TestPhase3B4PromotedPublicParents` and `TestPhase3B3BaselineS2CompletionExpectedFailure` in integration lanes. The final documentation correctly retains the root-module/historical-input bounds.
- Checked successful final build, vet, module verification, formatting, whitespace and all three Staticcheck exit records. Their stderr files are empty. The scoped rereview already checked the focused regression's final passing output and the changed behavior; the finalized fix report still discloses the earlier sandbox bind failure, regression red run and mistaken transient-stage assertion. Those historical failures have not been relabeled as passing qualification.
- Read the completed `controlled-sigkill.stdout` and zero exit. It records an independently acknowledged first GET, withheld response, SIGKILL, then a distinct second operation after explicit approval and resume. Final evidence retains the original as `reserved_unknown`, the successor as validated/settled, two contacts and two charges. The successful parent reference request remains a separate fixture/store from the named interruption child, as established in the source review. The published description matches this bounded result.

## Source custody and bridge

- The before and after tracked manifests are byte-identical and contain 864 paths. Each manifest hashes to `803c1b8cf3fa274ec06f36c5c1482a7aa23e7dda03d9f7ad3153c0439e86f962`.
- The before and after Git-archive digest records are byte-identical. Independently hashing `git archive cd18daa` yields `7b3944c0b0e867c32984fc399d8fa430334b4c3fa652b3e7de1b8048079c0123`, matching the record and published qualification.
- Independently hashing `final-qualification/weather-agent` yields `223daf1399ef932db1aee825a5cdf83834ebb2db69208e714e1f56c3e47a515b`, matching the finalized fix report. The prior scoped review verified that the new compatibility command records use this executable and identify `cd18daa`, rather than the older copied comparison binary.
- `freeze-after.stdout` records the matching source/manifest hashes. `delivery-bridge.stdout` identifies `2bafdf3` and its tree, with only `docs/evidence/weather-agent-experience/qualification.md` changed. Both gates exit zero with empty stderr. Independent Git comparison confirms the sole 40-line Markdown addition and no Go/module/weather-testdata differences between qualified source and delivery. Tracked status is clean.

## Qualification claims

Read the complete `cd18daa..2bafdf3` documentation delta and finalized `final-fix-report.md`. The new section accurately separates the final campaign from the original `9c2b0f6` campaign, uses the correct frozen source and hashes, and records the completed suites and controlled interruption. The original campaign's commands and source bridge remain historical rather than evidence that the new code was tested at the older revision.

The previously checked final retained comparison has 12 byte-identical detailed JSON reads and 16 successful main/WAL-preserving reader commands over the four named cuts. `compatibility-inputs.stdout` confirms original producer main/WAL and baseline JSON digests on both input copies. The final offline output confirms archive SQLite/response hashes, the original reading, replay and help commands; the final source review had checked the corresponding successful exit records. The documented counts, unchanged archived evidence, four-cut scope, absent/empty WAL convention and SHM exclusion remain accurate. The separate zero-allowance test supports the new local-refusal guidance without extending the four-cut compatibility claim.

## Findings

Critical: none. Important: none. Minor: none in the final evidence delta.

## Declined to judge

- A future metadata-only review-publication commit: not yet present in this reviewed range. The controller will verify verbatim report-copy hashes and unchanged executable inputs before push.
- Remote publication, PR state, merge or release: not performed or implied by this verification.

## Assessment

**Final delivery approved at `2bafdf3`.** The source corrections were already approved at `cd18daa`; the then-pending race/control gates and source-custody checks now pass, and the sole evidence-document delta accurately describes their scope. No unresolved review findings remain. Approval supports the requested draft-PR delivery, subject to the controller's stated checks on the subsequent metadata-only report-copy commit; it does not authorize merge or release.
