# Task 5 independent performance review

Spec compliance: **APPROVE**. Task quality: **APPROVE**.

Scope: exact `b2b9cc9f8dc1b0af7f8f38ac684136b53709469e` through
`6934e41aed749a14d2f68c26603518b8dfd57839`. This is the Task 5 performance,
methodology, attribution and evidence gate, **not final merge review**.

## Review method and scope

Read the complete supplied eight-file change in passes: measurement/control code,
extraction/tests, reproduction/results prose, then the large numeric JSON. For
the JSON, parsed the complete added diff and compared its full object with the
checkout and retained raw extraction; independently checked every primary metric,
phase, row and diagnostic summary against raw reports rather than printing all
14,256 numeric lines. No changed implementation file was reread outside the diff.

Named risks justified these focused reads outside the change: approved guard and
proof implementation (`campaign/analyze.py`, `validation.py`, `custody.py`);
possible counter-placement/SQL-attribution error (`instrument.py`,
`selector_probe.py`, P2 generator and SQLite counter helper); fabricated control
reuse or dependent reference (`readiness_reuse_test.go:46`,
`readiness_reference_test.go:14`); source/build contamination and claimed retained
program identity (frozen archives/manifests, actual overlays and isolated SQLite
copies); profile over-attribution (retained full pprof tables and line annotations).
These checks did not reopen general production correctness review.

## Strengths and verified evidence

- **Frozen campaign and custody hold.** `results.md:12` and `reproduce.md:9`
  identify the actual approved binding. Independently hashed the binding to
  `66a8c3a4d2d596ba53c3c1e35930eb4c706f747c52cb402e7584efd4fdfd6116`;
  verified all **4,125** files in the Task5b raw index (3,243 Task5b and 882
  primary-campaign files), all current bound source/module/dependency/tool/Go/
  Python/binary identities, and source archive contents and commit comments.
  A/B archives resolve to the declared `926a899b` and `4f5fe2cc` commits. Both
  original driver byte sets agree except the explicitly bound module replacement.
  Build commands, Go1.27.3/CGO1/SQLite1.14.52 and resolved source paths agree.
  Current production Go/module bytes equal the timed candidate; all original P2
  files equal both frozen sources. The supplied change contains only eight P3
  evidence/tool files; no API/schema/dependency/production/P2 modification.

- **Every original row is accounted for.** `results.md:28` is supported by all
  98 exact argv/cwd/environment/start/outcome/meta/report records, frozen row
  order and exclusions, report/progress hashes and serial process intervals.
  Exactly 14 excluded warmups and 84 measured outcomes remain, all exit 0.
  Reapplied the approved complete direct-proof validator to all retained reports
  as read-only analysis; all pass. Independently checked configurations, phase
  values, contacts/evidence/prefix cardinalities and logical counts. No profile
  or counter report enters the primary sample set. The inherited HTTP outer
  label is qualified; actual direct transport has no server contacts.

- **Numerical claims and unchanged guards reproduce.** `results.md:39` and
  `results.md:77` agree with independent rational-median calculations from raw
  integer nanoseconds/bytes. B256 median Execute is
  **60.5581259375s → 47.1938262505s**, reduction **22.068548985140068%**.
  The six exact baseline-minus-candidate differences are
  `13.342555500, 13.409476208, 13.546315124, 13.192766792, 13.474601458,
  13.122908041` seconds. The separately reported median paired ratio also
  reproduces. Every bound's full resource/runtime metric samples, medians,
  ranges, all non-Execute phases and summed approval phases agree with raw
  records. Every original strict max(10%,1ms) phase/small-work guard and
  max(10%,8MiB) RSS guard passes. B1's +0.1181045ms is retained, not hidden.
  B256 allocation is **53,008,366,108 → 42,191,146,816 bytes** while RSS is
  **86,499,328 → 87,179,264 bytes** (+0.6484375MiB). `results.md:68` correctly
  separates cumulative allocation from peak memory and states map-retention cost.
  The six-sign 1/64 statement remains conditional on independent fair signs;
  neither the report nor this review establishes that assumption on this host.

- **Mechanism is independently supported, including unchanged work.**
  `extract_task5.py:18` reconciles weighted visit/distinct/selected histograms
  and actual ProgramID frequencies; `extract_task5.py:52` checks load/decode/
  validation, binding, reduction and observation predictions. Checked raw A/B
  counters at all seven bounds, not just the compact results. At B256,
  successful loads/decodes/validations are **178,601 → 1,537**, SQL executions
  **3,600,555 → 3,246,427**. At every bound the only SQL-family execution changes
  are the program-row SELECT and schema-version SELECT, each reduced by the
  eliminated-load count; attribution-overflow/incomplete events are zero.
  Counter placement observes visits before cache lookup, loads at their real
  success boundary, binding at each actual call, and reductions/observation
  selection at the actual sites. All `work.*` counters match between A/B:
  B256 retains 178,601 bindings, 1,023 readiness/committed reductions, 511
  observation attempts/selections, 29,358,102 observation considerations,
  360,320 considered runs, 8,421,376 root-intent checks and 65,536 roots.
  The raw complete proof retains 256 effects/direct entries/charges/results/
  selections and 255 later approvals. Actual retained program bytes rehash to
  the two claimed IDs and contain no HandoffPeer ports. Reviewer SQLite reads
  used separate copies; original main/WAL/SHM hashes remain bound.

- **Fixed controls answer the intended reuse/overhead question.**
  `control_overlay.py:77` fixes one excluded call and 20 measured calls with
  unchanged selections and no intervening Commit; `diagnostics.py:18` freezes
  six balanced pairs per case before execution. Verified all 36 retained
  samples, medians, allocations, actual four-times-eight or 32-times-one ID
  distributions, 32 bindings per call and zero/one reductions as appropriate.
  The genuine fixture uses retained encoded programs, and the independent
  baseline selector body exactly matches the baseline function, SHA256
  `e542cad01b791114ed6f53c4fa94df05a6e1387cb8395c93a54f1ccc3caca3b3`.
  The timed source is the frozen candidate plus only the timing test; counters
  use a separate binary. Distinct-control medians reproduce 1,422,090.625 →
  1,425,321.875ns and 819,602.2 → 829,687.0 allocated bytes per call. The +0.227%
  time/+1.23% allocation estimates and shared-case savings in `results.md:123`
  are appropriately limited to these controls.

- **Supplement and profiling remain separate.** `supplement_task5.py:15`
  preserves the original four-bound declaration and a separately frozen fixed
  six-row 16/32/64 A/B coverage supplement, consistent with the controller's
  ruling at `progress.md:85`. Both protocol/tool/build-input hash bridges hold;
  no primary/control sample or threshold changes. All 70 diagnostic child gates
  exit 0, have matching command/executable/output identities, and are serial
  after primary timing. The B128 profile uses the ordinary candidate binary
  with 524,288-byte sampling; duration 12.56s/sample CPU16.16s and all six quoted
  inclusive allocation entries match retained pprof output. `results.md:147`
  correctly avoids adding overlapping scopes or equating sampled CPU with
  exclusive wall time. The independently checked B256 Run/approval phase
  medians and source-line profile still expose observation/root/snapshot costs.
  No second optimization was introduced.

- **Tests and failure handling are meaningful within this task.**
  `test_task5_diagnostics.py:16` includes 98 deliberate counter/distribution
  inconsistencies; `:34`, `:41` and `:51` check preserved work, full primary rows
  and fixed controls; `:64` rejects missing/duplicate overlay anchors.
  The retained five-test output is clean: exit 0, no skips or warnings, and its
  source/stdout/stderr hashes match. `diagnostics.py:61` uses exclusive gate/log
  creation, records outcomes before post-exit metadata, and stops on failed
  child status; source/build custody is checked around diagnostic phases.
  Examined all diagnostic stderr: 7,098 expected progress lines, no unexpected
  warning/error output. No workload or test suite was rerun by this reviewer.

## Findings

**Critical:** none. **Important:** none. **Minor:** none.

The original Task5a REJECT is not superseded as historical evidence. Its original
binding still hashes to `4be88671127cc9856609ea1b1a7fc8bdc1d8dfd4dac5f0df1499c22bac111c64`;
the replacement bridge changes only the three approved tool hashes. The prior
REJECT/counterexamples and explicit `task-5a-rereview-1.md` APPROVE remain present.
This review applies to the approved replacement campaign, not the rejected tools.

## Verification record and remaining boundary

Read-only review checks are retained under
`/private/tmp/alan-p3-performance-review-6934e41/`: `check.py`, `results.json`,
`stdout.log`, empty `stderr.log`; `bridges.py`, `bridges-results.json`,
`bridges.stdout.log`, empty `bridges.stderr.log`; and `log-check.json`.
Both substantive check scripts completed successfully. An exploratory read used
an incorrect `measurement.mod` filename; it was corrected to the actual frozen
`go.mod` and is not a campaign/tool failure. No new timing, broad tests, builds,
Git/index/branch operations, production edits or nested agents occurred. Only
this requested review report was written in the checkout.

Task 5's performance conditions support retaining the single optimization for
Task 6. Full current-tree ordinary/integration/integration-race, focused
semantic/process, public-consumer, historical-reader, authority, Staticcheck,
documented build/vet/module gates, final adversarial review and draft-PR/raw
publication remain controller-owned obligations. Earlier qualifications are not
relabelled as current execution. `main-after-primary.json` records remote main
still at `926a899b` after this campaign; this review does not assert a newer
remote observation. No merge or release is approved.
