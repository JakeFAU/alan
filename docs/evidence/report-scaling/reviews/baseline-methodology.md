# Task 1 independent review

Spec: **APPROVE**. Quality: **APPROVE**.

Reviewed base `ce731c59d99e3dc77af296ab021f652f02d234fb` through head
`a8f32107aae32e5aa5a5785be1c6b91a139fbbeb` against `task-1-brief.md` and
`docs/evidence/report-scaling/contract.md`. No actionable findings.

## Scope and evidence

- The complete diff adds six report-scaling evidence/apparatus files. It changes
  no production source, dependencies, schema, or prior evidence. Diff whitespace
  validation passed. The measured production commit remains `5f5d382c8d1e9c7369e7624e3a962746c9bf2229`,
  tree `cb6dc03624010d13f8af8d045af44e490110bfaa`; the checkout's three untracked
  apparatus files at measurement time are disclosed, rather than called a clean
  whole checkout.
- Independently regenerated the frozen oracle in memory and matched the fixture
  byte-for-byte. Independently archived the pinned Git source and matched its
  recorded digest. The oracle freezes the report traversal, root grouping and
  charges, root/invocation preparation and composition scope preparation. Shared
  authoritative leaf readers are identified in `README.md`; independence is
  appropriately bounded to changes outside those shared implementations.
- Read the generated `task1-baseline/ordinary/driver/prefix_proof.go` against the
  actual `RunReport`, `RootCompletion`, limits, charge and outcome types. For this
  workload, assertions cover every report/root field, nil/empty distinctions,
  deterministic root/open-run ordering, unchanged prior outcomes, original
  membership, and exact work/uncertainty counts. Final typed inspection ties each
  first-returned outcome back to retained evidence. Completed reopen verifies the
  returned zero-progress report, and Replay checks the complete root slice.
  These assertions do not claim coverage of arbitrary histories.
- The ordinary route uses the exported production source and unmodified SQLite;
  its driver has no instrumentation hook. The separate counter route changes
  only report scope/root counting in the executor plus the reused SQLite probes.
  Existing driver, recorder, validation and probe modules are reused and bound
  by inventories; there is no new dependency or copied independent framework.
- Verified all 13 bound source inventories and 23 tool/artifact hashes against
  current raw inputs, plus the apparatus, binaries, source archive, results,
  all 37 baseline command/stdout/stderr records and actual zero exits, all eight
  baseline row records, and the three smoke output hashes.
- Revalidated the saved seven ordinary rows and B256 counter row using the
  apparatus validators, recomputed proof digests/resources/attribution, and
  matched `baseline.json`. The report emitted 65,536 roots and recorded 625,792
  finishRun executions; its five named target families total 392,960. Complete
  registry partitions pass. Timing rows are uninstrumented; excluded single
  observations and the two-query accounting model are not presented as measured
  performance gains. The 48-execution historical difference is explicitly
  unresolved outside report scope.
- Recounted the raw ordinary test JSON: 1,408 test passes, 18 package passes,
  four test skips and three packages with no tests, with no failures. The four
  historical-reader skips and required fixture variables are disclosed. The
  oracle compile gate records successful integration-tagged compilation with
  no tests selected. This is compilation evidence, not differential execution.

Raw evidence checked: `.superpowers/sdd/plan-report-scaling/task1-baseline/`
(`source-freeze.json`, `input-inventory.json`, `results.json`, `gates/`,
`samples/`, generated drivers and counter transformation records) and
`task1-smoke/`. Committed interpretation: `docs/evidence/report-scaling/README.md`,
`baseline.md`, `baseline.json`, `measure.py`, `oracle.py`, and
`report_frozen_test.go.txt`.

## Continuing obligations

Approval is for Task 1 apparatus and excluded baseline only. Task 2 must install
and exercise the frozen oracle on same-seed histories, including errors and
partial reports, and keep shared validators unchanged or separately freeze/test
their original behavior. Historical-reader qualification, corruption/recovery
coverage, repeated paired measurements and final gates remain later tasks.
No Go suite, execution sample or timing campaign was rerun during this review;
checks above only read/revalidated retained evidence and regenerated immutable
source/fixture bytes in memory. Review writes are limited to this report.
