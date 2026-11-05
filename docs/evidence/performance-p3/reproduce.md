# Reproducing the P3 evidence

The canonical campaign already ran once. Do not overwrite it, append samples,
replace rows or rerun until a preferred result appears. The immutable contract
and design are [contract.md](contract.md) and [schedule.json](schedule.json).
The mandatory strict direct tooling gate was independently approved at
`b2b9cc9f8dc1b0af7f8f38ac684136b53709469e`, after the retained REJECT at 59bc43a.

Canonical primary binding:
`/private/tmp/alan-p3-j216kxoz/task5/fix1/checkpoint-binding.json`, SHA256
`66a8c3a4d2d596ba53c3c1e35930eb4c706f747c52cb402e7584efd4fdfd6116`.
It binds every source/module/dependency file, both actual binaries, original
driver bytes, exact controller hashes, all 98 argv/cwd rows and whitelist. Source
commits are A 926a899b651fccc48be52a054612a92afcf1ddfd and
B 4f5fe2cc83316900a2f387180dd9f856db5303a0. The actual primary invocation is
recorded in task5b/primary-command.json; its launcher and outcome are retained.

```sh
PYTHONDONTWRITEBYTECODE=1 python3 docs/evidence/performance-p3/campaign/run.py \
  /private/tmp/alan-p3-j216kxoz/task5/fix1/checkpoint-binding.json \
  66a8c3a4d2d596ba53c3c1e35930eb4c706f747c52cb402e7584efd4fdfd6116
```

That command now deliberately refuses its existing output. Independent
reproduction requires a separately declared new output/input binding, preserving
the original campaign and its interpretation. `campaign/freeze.py NEW_ROOT`
exports the exact two commits and builds the unchanged driver with retained
module replacements; it does not run a workload. A new binding is not a waiver
of the independent tooling/input review requirement.

Canonical analysis command (use a new output path; do not overwrite evidence):

```sh
PYTHONDONTWRITEBYTECODE=1 python3 docs/evidence/performance-p3/campaign/analyze.py \
  /private/tmp/alan-p3-j216kxoz/task5/fix1/checkpoint-binding.json NEW_ANALYSIS_JSON
```

It rechecks every warmup/measured outcome and raw hash, full direct proof,
source/binary/tool/environment custody, exact schedule/order and fixed guards.
Actual analysis command/outcome/hashes are in task5b/analysis-*.json.

Separate diagnostics are defined by
`task5b/diagnostics/protocol.json`, SHA256
`c900752dd6305157550f643bc95514b17544a7f11f52de48e3e072565d524341`.
The fixed 16/32/64 coverage supplement is
`task5b/diagnostics/coverage-supplement.json`, SHA256
`73103db330860efb3f5c593d79fa8318b9bf19b0a3842402eed53a3e31fe45f3`.
The original and supplement are separate declarations, each frozen before its
rows ran; the supplement completes the seven-bound mechanism matrix and does
not change primary/control timing samples. Tool bytes are copied into
`diagnostics/frozen-tools/`. Generated overlays and exact test/binary/source
inputs are bound by `diagnostics/built-inputs.json` before control execution.

Executed diagnostic routes, with raw root
`/private/tmp/alan-p3-j216kxoz/task5b/diagnostics`:

```sh
python3 docs/evidence/performance-p3/diagnostics.py freeze RAW_ROOT --binding PRIMARY_BINDING
python3 docs/evidence/performance-p3/diagnostics.py build RAW_ROOT
python3 docs/evidence/performance-p3/diagnostics.py controls RAW_ROOT
python3 docs/evidence/performance-p3/diagnostics.py counters RAW_ROOT
python3 docs/evidence/performance-p3/supplement_task5.py freeze RAW_ROOT
python3 docs/evidence/performance-p3/supplement_task5.py run RAW_ROOT
python3 docs/evidence/performance-p3/diagnostics.py profile RAW_ROOT
python3 docs/evidence/performance-p3/extract_task5.py PRIMARY_ANALYSIS RAW_ROOT ORIGINAL_DIAGNOSTIC_RESULT
python3 docs/evidence/performance-p3/supplement_task5.py extract RAW_ROOT --input ORIGINAL_DIAGNOSTIC_RESULT --output COMPLETE_RESULT
```

The supplement was frozen while original counter rows ran, then executed only
after they finished. Counters, controls and profile did not overlap. New
independent executions require new raw directories. The 32-run controls preserve
source-frozen genuine fixtures and an independent baseline selector; six fixed
AB/BA pairs, one excluded call and 20 measured calls per process are diagnostic
only. Timing and instrumented binaries are separate. No public approval shortcut
or primary driver alteration is used.

Every actual child argv/cwd, whitelisted settings, executable pre/post hash,
exit, stdout/stderr hash and host load is in `diagnostics/gates/NAME/`. The
recorder saves child outcomes before post-exit metadata reads. Full native
counter reports, SQLite stores, patches, profiles and source manifests remain
external, with their index and digests in [results-data.json](results-data.json).
Supplementary pprof full cumulative tables and selectReady allocation line
annotations are read-only analysis of the same retained profiles, not new runs.

No production Go, public API, schema, scheduler, authority, durability, pinned
module or original P2 source changed during Task 5. Full current-tree repository
and adversarial qualification are later controller-owned gates, not implied by
these benchmark/diagnostic executions.

## Final source and repository qualification

The [decision and review index](decision.md) answers the eight governing
questions and preserves rejected findings. Final gates use a new external root
`/private/tmp/alan-p3-j216kxoz/task6/fix2` for the complete source run at
`4184c5129b9d7b9e62f155c41b86763f7a103417`; no primary, counter or control row
is rerun. The later external-input correction prompted by review is qualified separately
as described below.
The complete 4184c51 run used the following routes from its clean committed
source. Existing directories now refuse overwrite; independent reproductions
must choose a fresh root and identify their checked-out commit:

```sh
PYTHONDONTWRITEBYTECODE=1 /opt/homebrew/bin/python3.14 -B docs/evidence/performance-p3/final/qualify.py freeze /private/tmp/alan-p3-j216kxoz/task6/fix2
PYTHONDONTWRITEBYTECODE=1 /opt/homebrew/bin/python3.14 -B /private/tmp/alan-p3-j216kxoz/task6/fix2/candidate/docs/evidence/performance-p3/final/qualify.py execute /private/tmp/alan-p3-j216kxoz/task6/fix2
```

A new reproduction uses a new root; existing gate directories refuse overwrite.
`freeze.json` binds the clean final Git export, historical exports, tool hashes,
SQLite source and the unchanged timed Go/module byte bridge. `execution-plan.json`
records the whitelist and module inventory. The executor runs from the export.
Each gate retains exact argv/cwd, whitelist, executable and source/tool hashes
before/after, stdout/stderr, actual exit, timeout and parsed terminal events.
The actual outcome is fsynced before post-exit metadata; failed children,
unreadable executables and timeouts remain failures. Meaningful recorder tests
exercise real exit23/deleted executable, actual timeout and synthetic credential
exclusion. No inherited environment capture is used.

The default external deadline is 2400s, following the previous full-qualification
budget; full Go suites use `-timeout=30m`, focused suites `10m`. A timeout kills
and reaps the child's process group and is never accepted. Go1.27.3,
SQLite1.14.52 and the installed Staticcheck2026.2.2 stay fixed. Staticcheck runs
ordinary/integration and actual compiler race/integration-race with
`GOFLAGS=-race`; previous tags-only race checks are not relabeled as these gates.

Routes include focused P3 ordinary/race, full ordinary/integration/integration-
race, authority/protocol and both actual frozen-old-reader subtests in each
appropriate full gate, CLI/root build, ordinary/integration vet, module verify/
tidy-diff, gofmt and committed-diff whitespace. The fresh old CLI is built from
`af6943a5805a9a6c0e8fe77f804264f4765d07ec`. The unchanged qualified public route
runs ordinary/race consumers, genuine conflict promotion11→12, baseline generic
readers and retained-format checks on complete main/WAL/SHM copies. All five
nested Go modules are copied and explicitly bound to the tested export for test,
build, verify and tidy-diff; this is six modules including the root. No-test
packages are listed separately and are not execution campaigns. The one opt-in
historical S2 falsifier remains separately identified if skipped; old-reader
opt-in skips are closed by supplying the actual binary.

The complete 4184c51 run records are `task6/fix2/attestation.json`,
`task6/fix2/final-source-tool-hashes.json` and `task6/fix2/raw-index.json`.
The external-input correction records are
`task6/review-fix1/external-attestation.json` and its raw index.
`task6/delivery-attestation.json` links those actual source-specific outcomes
and the checked Go/module byte bridge; it does not relabel the 4184c51 run.
The associated new draft PR links the verified evidence bootstrap/index and checksummed archive
attachments, plus final independent review and exact tested/delivery SHA. Raw
large logs, stores, binaries and failed attempts stay external. Do not infer a
published URL or final outcome from this prequalification protocol; consult the
actual attestation and delivery PR. Earlier Task4/5 evidence keeps its original
source identities and retained limitations.

The first final-source attempt at `8d2bdd5276e97814a76e5e014da5b6b331c4357f`
passed ordinary tests, then integration failed while unchanged reciprocal tests
recorded their source HEAD (`phase3b3_runtime_test.go:462`). A plain Git archive
has no metadata; the direct `git rev-parse HEAD` reproduction exits128. The
original integration exit1, 48 failing provenance events, 2,298 passing events
and one historical S2 skip remain in `task6/`. This is an apparatus failure,
not a passing integration run or a diagnosed runtime regression.

The corrected freeze retains a standalone shallow local Git metadata copy outside
the source export, verifies its HEAD equals the frozen source, and binds its full
inventory. Only candidate test gates receive explicit `GIT_DIR`/`GIT_WORK_TREE`
pointing at those identities. No production/test assertion changes. Corrected
qualification initially used `task6/fix1`, preserving the original attempt and
its executable-input byte bridge. That attempt later stopped at the build-output
incident below; the complete run is `task6/fix2`.

A second preserved apparatus stop occurs in `task6/fix1/gates/module-1-build`:
`go build ./...` exits0 but places the standalone driver's binary inside its
copied source module. The source-inventory guard correctly rejects that added
file; no source file changed. The corrected controller explicitly directs nested
build outputs into `module-builds/INDEX/`, outside every source inventory. The
original binary, successful child outcome and failed custody check remain; they
are not a runtime or test failure. Each gate keeps its actual source identity.


## Reviewed actual external-module custody correction

The preserved [Task 6 REJECT](reviews/task-6-review.md) identifies that the
external candidate and baseline-reader module copies were absent from the
recorder's source mapping. Their original successful test exits remain real,
but those records lack the required before/after actual-build-input custody.
The unchanged Task 4 route is now called through `final/external_inputs.py`.
It verifies each copy against the frozen original before intentional binding,
checks that exactly the declared go.mod replacement is added, persists both
inventories and the byte bridge, and registers the actual bound module before
compilation. Every subsequent gate now hashes those actual module inputs.

The focused tests reject changed non-module bytes, a wrong replace target and
an actual child changing an input after registration. The later source is
frozen separately at `task6/review-fix1`. Its six recorder/input tests and the
complete affected external route run afresh: ordinary/race public consumers,
genuine 11→12 promotion, baseline readers and retained formats, with complete
main/WAL/SHM copy custody. All 330 Go/module input files are checked byte-equal
to the full 4184c51 run. Root Go tests, build/vet/module/format and Staticcheck
remain qualified at 4184c51; they are not relabeled as runs at the correction
commit. The delivery attestation names both source trees and gate sets.

To reproduce the affected route from the correction's immutable export, run
`python3.14 final/qualify.py external NEW_FROZEN_ROOT` (the full script path is
`docs/evidence/performance-p3/final/qualify.py` inside that export). First run
its `freeze` phase against the clean committed source as above. The exact
retained launcher, argv/cwd/whitelist, source inventories and outcomes are in
`task6/review-fix1/`; fresh reproductions must choose a new root. Neither an
existing failed gate nor a source-specific successful result is overwritten.
