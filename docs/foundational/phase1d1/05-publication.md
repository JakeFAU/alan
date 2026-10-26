# Phase 1D.1 publication

Completed the two tasks in the frozen plan: coupled asynchronous dispatch qualification, then README/evidence/review/publication. Plan checkboxes inside the source snapshot reflect its capture stage; this record supplies final completion status without rewriting that archived plan.

Draft PR: [#5 — Phase 1D.1: bound external dispatch and preserve independent progress](https://github.com/JakeFAU/alan/pull/5). It was created open and draft on 2026-10-25 against verified `codex/phase1b-recursive-composition` at `6effdc02259af5335633bfc41e9786386430fa7c`. No merge or release was performed.

Implementation commit: `3a869e1` (full reviewed source and evidence). Packaging correction: `cc05e1ea0d2ab624a69efa876d8f42a9666cdedd` (same source; retained unified patch compressed byte-for-byte so whole-branch whitespace checks pass). That was the initial published PR head. This subsequent publication-only record does not change the qualified Go/module source.

Qualified source: `9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63`, 58 files. Source archive: `cb0763f6a9e23a1c9a35f647b4658d61adba16a8acaddc40767e70a390b5d535`, 95 members. Both have complete per-file manifests.

Final normal/race suites each pass 357 test/subtest events, including actual race-built child executables. Static checks, prior research comparisons, 18 README smoke commands and empty-module-cache archive reproduction pass. Independent whole-branch review identified one reporting defect; the correction, unchanged failing/passing probe, scoped approval and final artifact audit are retained. No review finding remains open.

All earlier phase evidence and the 83-file user extraction in the original checkout remain unchanged. The registered worktree and implementation branch remain available for review. Only this plan's ignored temporary review workspace is removed after its reports and inputs are retained in this package.
