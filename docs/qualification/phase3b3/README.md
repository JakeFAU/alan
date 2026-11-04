# Phase 3B.3 runtime qualification

The current-runtime baseline is frozen at `8319a9067c16aaf5e3ecccb3d0c62f5c536e6140`, with production behavior unchanged from `af6943a5805a9a6c0e8fe77f804264f4765d07ec`. `baseline-frozen.tar.gz` contains fresh normal/race executions and the expected failing S2-completion assertion. Its source manifest binds all 283 Go/module inputs; commands retain exact exits and logs. This establishes the pre-primitive gap, not successful delegation.

S1 consumes two authentic original results; missing/claim/Cartesian authority attempts retain `grant_denied` and no unauthorized contacts. The additional prior-result laundering control uses a separately identified reversed service schedule. Baseline Cartesian tests exercise static exact tuple lookup; the two operation labels invoke the same physical increment. They do not establish the later delegated physical-operation control.

`baseline-development.tar.gz` and `baseline-review.tar.gz` preserve development/RED history and the independent review, including a reviewer-induced archival custody incident: read-only SQLite opens changed 24 SHM sidecars, and matching-byte reconstructions were written before the parent instruction against reconstruction arrived. Those sidecars are not untouched originals. Database/WAL and exported JSON did not change according to before/after hashes. The final `baseline-frozen` archive was freshly regenerated after source commit and all 320 referenced artifact digests were verified before packaging. The incident, observed/recovered bytes and mutation driver remain explicit. Never open archived SQLite directly; make a complete disposable copy including WAL/SHM before database inspection.

`baseline-manifest.json` binds every member of all three archives. `baseline-freeze-command.py` preserves the exact generating script and its original clean-source assertion; its original absolute output paths are provenance, not a portable runner. To reproduce, check out the pinned source in a clean worktree and run the commands in `baseline-frozen/commands.json`, choosing a new output directory for `ALAN_PHASE3B3_EVIDENCE`. The expected-S2 command must exit 1 at the identified missing-authority assertion; ordinary normal/race runs must exit 0.

`rejected-literal-root` preserves the first implementation probe's original failing test and log. It falsifies the precompiled hashed-root construction in contract v1; contract v2 uses exact logical root locators. No general delegation impossibility or positive runtime realization follows from that probe.

## Primitive evidence index

The reviewed primitive is frozen at `4f6f5f2500a48b2ca01ee7b09a823fa63656b7ab`.
`primitive-freeze.json` binds its contract, source manifest and scoped approvals.
Full reciprocal-result and process-cut qualification is a separate experiment,
now frozen at `759df6e3d2505f27e1287b2cfeb07bc4290afc0a` and indexed in
[`task4-frozen/README.md`](task4-frozen/README.md). Both ordinary and race runs
retain 48 cases; the eight final commands passed. Independent raw-history
correspondence and its explicit model-domain limits are recorded separately in
the research repository. Passing runtime controls do not erase the retained
immediate-selection model/runtime mismatch.

| Location | Evidence and interpretation |
| --- | --- |
| `rejected-candidate-v2` | Exact rejected source, independent REJECT reports and retained malformed-history probes. Restore its source-only snapshot over `6eca5f81faf41165e7fa77819537e49fafcd0249` to supply unchanged embedded fixtures. |
| `task2-completion` | Initial scoped verification, four-case legacy compatibility RED/old-source GREEN, closed legacy Store, and original debit RED/green logs. The debit source reconstruction is explicitly not the original failed source or an executed reconstruction. |
| `round1` | Corrected source evidence, all sixteen original-request mutation REDs, final scoped ordinary/integration/race checks and independent APPROVE re-reviews. The misleadingly named `authority-green.log` is a preserved failed fixture assumption. |
| `legacy-matrix` | Ten old-created formats 3–12 fixtures; 80 primary reader calls and 60 byte-identical paired exports; complete ordinary module suite. Four rebuildable test binaries are omitted, with hashes/build commands retained. |
| `task4-development` | Complete failed-controller and candidate qualification history, including the original immediate-selection narrowing witness. |
| `task4-reviews` | Independent binding/custody and recovery/admission APPROVE reports and raw-evidence checks. |
| `task4-frozen` | Immutable-source final runtime qualification: eight commands, ordinary/race reciprocal corpora, ten cut labels with two distinct unknown subcases. |
| `current-main` | Clean prospective integration with main `76031c34`, including the initial Git-provenance export failure and corrected same-tree qualification; neither branch was merged. |
| [whole-change-review.md](whole-change-review.md) | Independent specification, quality and overall APPROVE for the bounded Phase 3B.3 slice; explicit unsupported domains and deferred research mutation-CLI diagnostic. |

## Extracting and rerunning historical checks

Work in a new scratch directory. Extract archives there, preserving their
top-level directories (`baseline-frozen`, `candidate-source`, `binding-review`,
`semantics-review`, `round1`, or `legacy-matrix`). Verify archive members against
the corresponding wrapper manifest before running anything. For example, from
this runtime repository:

```sh
phase3b3_scratch=$(mktemp -d /tmp/alan-phase3b3-reproduce.XXXXXX)
tar -xzf docs/qualification/phase3b3/legacy-matrix/evidence.tar.gz -C "$phase3b3_scratch"
python3 "$phase3b3_scratch/legacy-matrix/compare.py"
```

That comparator reads the extracted JSON exports and hashes file bytes; it does
not open SQLite. It rewrites the derived comparison JSON deterministically; the
recorded extraction check confirms that all 419 extracted member hashes remain
unchanged. It checks the retained comparisons, not a new execution of the
old/current readers. Do not run a fixture generator against the extracted
historical originals. New executions need a separate fresh evidence directory.

Use a separate checkout or `git archive` extraction for each pinned source.
The old source is `af6943a5805a9a6c0e8fe77f804264f4765d07ec`; the corrected
source is the primitive freeze above. For the rejected candidate, first extract
its base commit, then overlay `candidate-source/source.tar.gz`. Its
`source.json` binds the source-only overlay. Do not substitute the current
production files when reproducing the fully rejected candidate.

Historical command records and overlay JSONs deliberately retain their original
absolute paths. Keep them unchanged. Make working copies, remap each overlay's
`Replace` keys to the corresponding new checkout paths and its values to the
extracted test or source files, and change command cwd/output paths to fresh
directories. For the legacy matrix, both overlays must use exactly the same
`legacy_matrix_test.go.txt` bytes. Rebuild the omitted binaries with the recorded
Go version and flags; run only the old binary in create mode, then both readers
on new independent copies. `ALAN_LEGACY_MATRIX_ROOT` must name that new output
root. Place a working copy of `compare.py` beside the newly generated exports
for their comparison. The original binary hashes identify the original
execution; path-remapped rebuilds are not claimed to be byte-identical binaries.

For round1's expanded RED, overlay only the preserved pre-fix admission source
onto the corrected source, using the corrected expanded test files. This is
the documented partial-source falsifier, not the fully rejected candidate.
Preserve expected exit 1 for semantic RED separately from compilation, sandbox
or fixture setup failures. Use cached dependencies and the recorded environment;
loopback/subprocess tests require the corresponding local permissions.

Before any SQLite reader or owner Open, copy the complete database/WAL/SHM set
to a fresh disposable directory. A read-only SQLite open can change SHM bytes.
Keep the extracted originals and historical manifests unchanged.
