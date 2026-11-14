# P7 Task 1 independent review

Reviewed 2026-11-14. Scope: task-1-brief.md, task-1-report.md, the frozen contract and audit proposals, and the supplied `7a4a46c..c19785d` diff. Reviewed final files and selectively read retained raw artifacts under this plan's named roots. No production edits, Git mutations, agents, builds, benchmarks, or repeated test suites were performed. This report is the only review write. Whole-branch and Task 3 delivery review remain separate.

## Verdicts

- **Specification: PASS for the frozen negative-result path.** The B256 advance screen failed all three conditions; Task 2 was correctly skipped. No production change was delivered and neither remaining scan was optimized.
- **Quality: APPROVE Task 1.** No substantive correctness, measurement, source-binding, or scope defect found. No blocking or medium-severity findings; no requested implementation changes.

These verdicts do not qualify the disposable candidate for production or substitute for the applicable Task 3 apparatus/docs checks.

## Source and scope evidence

Verified Git identities directly: baseline `9db473bae28e5b806c3ccdc7e2e134c1d0b38f27`, tree `56669e93b11a35ab8a67d09918b3a5e055944015`; corrected harness `ca65abb857a75f3677ce88a74cbeb94a2251450b`, tree `bb51e53578fda278c7b02a08c50c6efc9b56f7d8`; reviewed head `c19785d5ca2f3a6cc2ea4efe3ca539c76a0a6b80`, tree `3cca2e46782f03f2bb351d11cb0c98efed1b309a`.

The three Task 1 commits are the initial harness, build-metadata parser correction, and evidence/result commit. Every baseline-to-head changed path is under `docs/evidence/retained-query-scaling/`. No runtime, public API, old test, dependency, schema, index, migration, scheduler, ownership, retry, or BoundedSeries path changed. Thus the delivered application retains the baseline implementation of the named behavioral boundaries.

Independently checked `data/source-bridge.json`: both changed-path inventories agree with Git, and every post-measurement file digest matches. All five lane manifests bind their five P7 apparatus inputs to the actual recorded harness commit. Smoke uses `481de68`; counters, clock, profile and primary pilot use corrected `ca65abb`. The intervening correction only fixes stock/replaced SQLite build-info parsing and adds its Python control; no measured runtime changed.

Rehashed the actual pilot baseline/overlay source inventories (822 files each) and the actual clock-generated ALAN/SQLite inventories (823/113 files). The pilot baseline matches its archived baseline; all lane archives have SHA-256 `62336d8ca4f2f5328ec22ea51ebe97402c0ce2a48aec507fc005212ccac0e5eb`. The pilot overlay differs only in `internal/executor/store.go`. Its unique adjacent-statement transform preserves surrounding checks and parameters; patch digest `b343f561fe77970a062f79dc3428d719227308f392fcbefcd35525b5dd7cd388` agrees with manifests and the retained patch. The eight reused helper-directory inventories agree with the baseline archive. No prior-plan raw data was substituted for these new rows.

## Apparatus and semantic checks

Inspected all new apparatus/template/test code. Outside the diff, reads were limited to named risks: the reused P6 source/build/row validators and driver, the generated clock's actual SQLite step sites, and retained source/output custody. `measure.py` checks fresh output roots, clean tracked harness, exact baseline tree, unique anchors, row/design/source/binary binding, full schedule completion, preserved failed rows, and before/after bound-input equality.

The actual stock driver uses RunProgress in Execute and the same final full Run, typed inspection/series proof, close/Open, full Run after Open, resumed evidence comparison and Replay in both arms. Its proof checks original root/run membership, unique operation/key/result/observation identities, selected links, predecessor approval binding, one root effect charge, admitted/settled effects, result bytes and digests, and one cumulative new contact per position. No independent-namespace equality is asserted. All raw lane rows agree with their compact workload counts and retained evidence/contact/root cardinalities.

The diagnostic classifier binds the three exact prepared SQL strings. Inspected generated `sqlite3.go`: five actual call sites are wrapped—row execution, noncancellable Next, both eager-first-step branches, and cancellable Next. Buffered `pendingStep` return occurs before the Next wrappers. Start follows the initial Execute registry snapshot; stop precedes the final snapshot. Preparation, result marshaling and commit are outside these brackets. Clock-off performs no clock reads or aggregate increments; empty mode brackets the clocks before C. The disclosed inclusive-cgo/preemption and diagnostic-build limitations are appropriate; these values are not represented as saved latency.

Retained `frozen-clock/gates/step-clock-controls/stdout.log` hashes to `data/controls.json`, with 35 passing test events, one passing package, no failures or skips. Read the new eager/buffered/next/done, lookalike exclusion, disabled/window controls and pinned counter proof coverage. The FK control uses the baseline DDL, mixed observation subjects/generations at 1/8/64/256, actual step counters, guarded EXPLAIN paths, fresh-connection row/FK checks and failed-update rollback. Its mechanism claim is properly distinguished from crash/application recovery qualification.

## Raw results and arithmetic

Independently verified all **125 retained lane rows**: 4 smoke, 4 counters, 18 clock, 1 profile, and 98 pilot rows. Their compact representations equal the raw row records; raw stdout/stderr/command/outcome hashes match; source labels, binary hashes, lane flags, stage resource vectors, and cardinalities bind correctly. All five lane input snapshots agree before/after. Verified **327 gate records**, including raw custody hashes, full command/cwd/environment reconstruction and exits.

Complete raw Execute registry differencing at B8/B256 confirms only these template-level differences:

| Difference, baseline minus overlay | B8 | B256 |
| --- | ---: | ---: |
| History full-scan steps | 105 | 130,305 |
| History VM steps | 493 | 397,045 |
| Consumption-update VM steps | 15 | 511 |

No other template metric changes. Attribution-incomplete counts are zero. A zero-valued scoped history-scan entry is absent in the overlay; treating absent sparse counters as zero yields the same difference. All lifecycle stage counter vectors are unchanged except Execute and its inclusive lifecycle totals. SQL executions and row returns stay equal; result-lookup and charge-count scans persist. This supports the bounded mechanism claim without establishing a general scaling theorem.

Independently recomputed every published paired metric, all raw vectors, AB/BA strata, separate arm medians, paired savings and MAD. The 98-row schedule contains 14 warmups and 84 primary observations, with six adjacent alternating AB/BA pairs at every required bound. All sample intervals are serial, and no other **recorded** gate overlaps the primary window; both stock builds precede it. Build-info logs bind the separate ALAN exports and identical stock SQLite v1.14.52 checksum. Primary driver sources contain no step-clock derivative.

B256 raw Execute savings in block order are **+35.268250, +1.711749, +23.848917, −14.597666, −10.144458, −28.664668 ms**. Independently obtained:

- Median paired change **−0.326281%**, below +1%.
- **3/6** faster pairs, below 5/6.
- Median saving **−4.2163545 ms**, below MAD **17.4148125 ms**.
- AB **+1.827720%**, BA **−1.131850%**; B128 **+0.004245%**.

The smaller-bound results, lifecycle data, per-sample CPU sums, allocation volumes and RSS statements agree with the evidence. The report appropriately avoids significance, zero-gain or general regression claims. Clock on/off/empty aggregates and wall ranges independently match the table, including B256 history median 5.854341 ms and zero clock-off totals. The CPU profile's whole-lifecycle scope and inability to isolate individual SQL templates are correctly disclosed.

## Baseline, failures, and limits

The fresh successful baseline ordinary log contains 1,420 passing test events, 18 passing packages, three no-test-file packages and four explicit historical-reader skips. Both ordinary-suite exports match all 822 archived baseline files; their source inventories remain unchanged. The earlier failed ordinary log retains exit 1, 411 passing test events, 27 failed test events and nine failed packages, with loopback-bind failures. The report correctly distinguishes successful ordinary coverage from missing historical-reader qualification.

The initial failed counter root contains built binaries and build-info gates but no sample rows; the parser correction and fresh final counter lane support the disclosed correction history. The retained preflight roots and final control results support their staging. **Verification limit:** separate top-level terminal-outcome logs for the early duplicate-gate exception, initial Python red controls, and packaging correction were not present in the named roots I inspected. Those early narratives were not used to validate final measurements or the decision. The recorded inner gates/final data and final source were independently checked instead.

The gate timeline cannot establish absence of arbitrary unrelated host background work. This remains a single-host bounded pilot; opposite order strata and the stated spread are retained rather than suppressed. Raw local archives remain unpublished, as disclosed. This review did not rerun existing tests or perform candidate selected-write SIGKILL, same-seed recovery, historical-reader, full integration/race, or production qualification. Their absence is consistent with the frozen no-go path and must remain explicit in delivery.

Task 1's failed screen is supported. Proceed with the applicable Task 3 evidence/apparatus delivery path; preserve the no-production-change conclusion and draft-only/no-merge boundary.
