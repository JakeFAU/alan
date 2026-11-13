# Spec compliance: APPROVE — Task 4 apparatus and excluded smoke only

- Reviewed base `2b30d62fc8ac219d082b77f8af38eb81c9a38547` through head `7679755c7298f229fa468b14da54bcfe1420c4f8`. Read the prepared diff once; recovered only the middle omitted by tool-output truncation. No Git diff re-derivation, production edits, builds, or test reruns.
- The four added apparatus/control files match this task's scope. Full campaign execution, external/cross-build qualification and final gates remain Task 5 obligations, as explicitly scoped by `qualification-ruling.md` and `task-4-report.md:141`.

# Strengths and verified requirements

- `docs/evidence/report-scaling/campaign.py:31`: exact seven-bound schedule, 14 excluded warmups and 84 measurements, six adjacent balanced AB/BA pairs per bound. Diagnostic counters use a separate component. `test_campaign.py:16` checks schedule cardinality, order and ordinary full-report mode.
- `campaign.py:50`, `campaign.py:124`, `campaign.py:197`: exact clean candidate, baseline/historical pins, whole source exports/archives, selected tools, related build evidence and parent-source/tool subset identities are bound. A coherent foreign build manifest cannot replace the original source binding. Narrow live P5 bindings allow later Markdown evidence without permitting consumed code drift; the archived source remains wholly bound.
- `campaign.py:244`, `campaign.py:276`, `campaign.py:298`: process identity, actual exit, timeout/spawn errors, command/environment, input snapshots, output digests, complete reports, order and nonoverlapping intervals are checked. Existing directories prevent replacement of failed attempts; retained failure rows prevent acceptance.
- `sample_usage.py:13`, `campaign.py:231`, `campaign.py:328`: one wrapper child, post-exit getrusage and a unique final usage marker separate whole-process high-water from sampled Execute-stage cumulative RSS. Driver progress on stderr is accepted; duplicate/missing markers and inconsistent RSS/exit evidence fail.
- `campaign_qualification.py:177`: both independently built direct consumers receive the same seed and targeted basis. All required full RunReport cuts are validated. The non-nil OpenRuns-to-null mutation must produce comparator exit 1.
- `campaign_qualification.py:223`, `campaign_qualification.py:245`: fragmented test2json lines are reassembled per package/test; missing structured tails and duplicates fail. Real acknowledged process reaps, complete pre-reader main/WAL/SHM copies, two independent reader copies and retained public-consumer build provenance use the existing custody route.
- `campaign_qualification.py:64`, `campaign_qualification.py:70`: real historical binaries, environment variables, archived Git metadata and mandatory PASS events are supplied. Full gates, nested modules, differential/recovery and outside-rebuild entry points are present; their confirmatory outcomes are not claimed.

# Actual excluded evidence checked

- Final implementer report SHA-256: `f12a2a47fd703a43666a21b1290743e9db2f9cf7ddedaf60686b603515b6f8ce`. Raw root: `.superpowers/sdd/plan-report-scaling/task4-smoke-r3`.
- Independently hashed source freeze `2cf75366707f03394fe63dc0398f7f8d84857be7c4ddab110c65c64af16ee419`; it names candidate `7679755c7298f229fa468b14da54bcfe1420c4f8`, tree `7aeb84be8acf414800a649152746aacd3e03b5e6`. Qualification freeze `bf05b7c9537c7ac944dba0faec5f28503b84d6a51fa3cfb0a59794955ceb6bbe` and smoke freeze `fbf393ec1f334f0909acccb5bb751962e706321848df34388b45052cb4796433` both link to that source digest.
- `task4-smoke-r3/smoke-run/complete.json:1`: six rows, no failures. Read all six raw sample outcomes: exit 0, no timeout or spawn error. Independently rehashed each stdout/stderr against after.json; checked each final child-usage record has exit 0 and high-water at least every reported stage sample. Both arms' source labels and binary paths agree with their designated role. `smoke-analysis.json:1` binds the same source/build digests and marks the analysis excluded.
- `task4-controls/final-focused-controls.log:21`: nine apparatus controls passed. Expected exit 7, SIGTERM and timeout records are intentional controls. `task4-controls/smoke-r3-mutants.log:1`: six disposable-data mutations were rejected: source, missing prefix, counter/profile contamination, contact identity and false child exit. `task4-controls/results-addition-control.log:1` records the live Markdown addition check.
- `task4-smoke-r3/gates/direct-refusal3-reopen1-exact/outcome.json:1`: exact comparator exit 0. The two raw direct reports contain 25 public cuts and 15 complete RunReports each. `direct-refusal3-reopen1-reject-null/outcome.json:1` retains expected exit 1.
- Read raw PASS events from `gates/public-parents/stdout.log`, `gates/public-promotion/stdout.log`, `gates/old-owner-open-supported/stdout.log`, and `gates/excluded-four-historical-rejections/stdout.log`: K01/K02 compatibility and promotion, historical owner positive, and all four named old-reader rejection cases actually passed.
- `task4-smoke-r3/gates/candidate-weather-recovery/outcome.json:1`: test exit 0. Read three distinct kill records for PIDs 58726, 58729 and 58732: SIGKILL, reaped true and nonempty acknowledged witnesses. Independently hashed all three archived main/WAL/SHM sets and matched source-before/source-after/copy metadata. Six independent reader copy records preserve all three members; six reader exits and three exact comparator exits are 0.
- Earlier interrupted/parser-failed attempts remain disclosed in `task-4-report.md:61` and retained under `task4-smoke-initial`, `task4-smoke-r2`, and `task4-controls`. Their results were not substituted for the final r3 run.

# Named-risk helper checks

- Input drift/provenance: inspected unchanged `measure.py` build_driver/strengthen_driver, `execution-scaling/task5/custody.py:55`, `controller.py:123`, `overlays.py:45`, and `performance-p3/final/recorder.py:84`. Verified controlled generated modules, binary binding, strict snapshots and actual outcomes preserved before postmetadata failures.
- Incomplete/unequal full reports: inspected `execution-scaling/task5/measurement.py:171` and `qualification.py:365`, `qualification.py:462`. Verified shared full-report validation and exact-cut requirements; no compact-result cohort is introduced.
- False historical success or damaged crash custody: inspected `qualification.py:81`, `qualification.py:125`, `custody.py:46`, `custody.py:111`, and `recovery.py:21`, `recovery.py:111`. Verified supplied old readers, mandatory PASS events, immediate post-reap archive overlay and complete-copy checks.
- Nested/public qualification routing: inspected `qualification.py:314`, `qualification.py:335`; frozen replacement paths and inherited public-consumer route are used. Actual full runs remain Task 5.

# Issues and limits

- Critical: none found.
- Important: none found within the approved Task 4 scope.
- Native-input boundary: `execution-scaling/task5/controller.py:123` binds selected native executable bytes and SDK identity/settings, not every Apple SDK header/library byte. No observed SDK drift was waived. Final evidence must preserve this limitation and must not claim an independently hashed complete native SDK inventory.
- Confirmatory timing, full ordinary/integration/race/static/module gates, all differential pairs, both-arm series/cross-build recovery, independent eight-operation consumers and outside rebuild remain unverified execution outcomes. Task 4 approval establishes apparatus readiness and the stated excluded smoke, not Task 5 completion, performance benefit, unrestricted semantic equivalence or merge approval.

# Code quality: APPROVE — Task 4 only

- Reuses the tracked measurement, recorder, comparison and custody routes; separates measurement and qualification orchestration; preserves actionable failure evidence. Focused controls exercise real child failures and real retained-data mutations. No blocking quality finding remains at the reviewed head.
