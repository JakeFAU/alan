# Task 6 independent review

Spec compliance: **REJECT — one Important actual-build-input custody gap.**
Task quality: **REJECT — needs apparatus correction.**

Scope: exact `6934e41aed749a14d2f68c26603518b8dfd57839` through
`4184c5129b9d7b9e62f155c41b86763f7a103417`, reviewed from the supplied
`review-6934e41..4184c51.diff` in bounded passes. This is the Task 6
qualification-tooling/documentation gate, not final whole-branch review or
merge/release approval. The final `fix2` pipeline was still running when this
finding was reported; no unfinished outcome is treated as passing.

## Strengths and verified checks

- `final/recorder.py:73–122` exclusively creates gate directories, passes an
  explicit environment, starts a child process group, records actual status and
  timeout, fsyncs `outcome.json` before post-exit hashes, and rejects failed
  status/custody. `final/test_recorder.py:14–47` uses real exit23/self-deletion,
  real timeout and synthetic-credential exclusion. Frozen `fix2` retained outputs
  show all three tests passing, plus ten P3 tooling/diagnostic and four complete
  sidecar/custody tests, without warning output. No reviewer test rerun occurred.
- `final/qualify.py:34–114` separately exports and binds candidate, baseline and
  old-reader inputs, inventories fixed tools/SQLite dependency and standalone
  Git metadata, and verifies the exact HEAD. The scoped `GIT_DIR`/`GIT_WORK_TREE`
  repairs provenance in unchanged reciprocal fixtures without weakening their
  assertions or relabeling the earlier failed execution. Nested outputs at
  `final/qualify.py:188–193` are explicitly outside copied source modules.
- `final/qualify.py:146–211` declares focused ordinary/race, full ordinary,
  integration/integration-race, actual authority/old-reader checks, all five
  nested modules plus root, build/vet/module/format and four Staticcheck modes.
  Race Staticcheck uses actual `GOFLAGS=-race`. No-test packages are classified
  separately. All full-suite old-reader pass events inspected correspond to
  actual format13 and format14 `frozen_old_reader` subtests, not skip counts.
- Independent read-only checks at
  `/private/tmp/alan-p3-final-review-4184c51/{check.py,checks.json,stdout.log,stderr.log}`
  completed exit0. They verify all eight preserved review copies against the
  original reports and committed SHA256 index, all **330** Go/module bridge
  entries against actual fix1/fix2/workspace bytes, frozen candidate/baseline/old
  inventories, metadata and tools, and every then-completed gate's stdout/stderr
  digest and source/tool snapshots. Original REJECT reports are byte-identical.
- `decision.md:1–87` answers the eight required questions, distinguishes source,
  implementation and timing identities, qualifies supported owner/aliasing/
  cancellation/distinct-program boundaries and allocation versus peak RSS, and
  leaves later publication/integration obligations explicit. No second
  optimization, API/schema/dependency or original P2 edit appears in this scope.
  Task 5's independent approval is preserved; no timing was repeated by review.
- Earlier attempts remain separately attributable: initial ordinary has1,306
  pass events; initial integration exit1 has48 provenance-related fail events,
  2,298 passes and the one historical S2 opt-in skip. Fix1 ordinary/integration/
  integration-race have1,306/2,351/2,351 pass events; authority105, with two actual
  old-reader subtests in each. Fix1 `module-1-build` child exited0 and its guard
  rejected only the added output binary. Both diagnosed apparatus incidents,
  their original logs and the exact-source bridges remain. These are not
  asserted to be final4184 execution or runtime regressions.

## Important finding

### [P2] Public consumer and baseline-reader actual module inputs lack per-gate custody

`final/qualify.py:169` calls the unchanged `qualification/run.py:215` external
route. That route creates `external/candidate-module` and
`external/baseline-reader-module`, intentionally changes their go.mod replacement,
then compiles the public ordinary/race consumer and baseline reader from those
copies. Neither copy is added to the final `Recorder.sources` mapping.
`final/recorder.py:69–71` can inventory only that mapping. In contrast, the later
nested-module path explicitly registers its actual copy at
`final/qualify.py:175`.

This is observable in
`/private/tmp/alan-p3-j216kxoz/task6/fix1/gates/{external-ordinary-build,external-build-reader}/{command,before,after}.json`:
`cwd` selects the external copied module, but both snapshots list only
`baseline`, `candidate`, `git-provenance`, `old` and `sqlite-dependency`.
The actual external module files used for compilation have no before/after
source inventory. `checks.json` above preserves these exact source-key lists.
The same source path is present at4184; unfinished fix2 external gates are not
claimed as evidence either way.

The copy/mod-edit source and an eventual final raw index can establish the
intended input and its end state, but do not provide the explicitly required
before/after hashes of actual build inputs or detect their mutation during
compilation. No source mutation or runtime semantic failure is alleged.

Register each actual external module after its intentional replacement binding
and before compilation; bind that initial inventory to the frozen original with
only the declared replacement delta. Preserve the first records. Requalify the
affected public-consumer/reader route with recorded actual input inventories;
unchanged root semantic suites need not be repeated solely for this recording
correction. Review the exact correction and complete final attestation before
any Task6 APPROVE.

## Minor navigation follow-up

`reproduce.md:133–135` identifies canonical final files at `task6/attestation.json`
and `reproduce.md:154` names corrected qualification at `task6/fix1`, whereas the
then-current complete-pipeline candidate is under `task6/fix2`. Ensure the
reviewed delivery bootstrap/index and PR point explicitly to the real final
attestation and preserved attempts; update evidence-only navigation with its
executable-input bridge as appropriate. Do not imply the absent top-level
attestation exists or substitute an earlier attempt.

## Review boundary

Critical findings: none. Important: one. Minor: one navigation follow-up.
The remaining completed final gates, full attestation, final whole-change
review, publication custody/actual URL checks, current-main integration decision,
push and new draft PR remain controller-owned. This REJECT concerns missing
qualification apparatus evidence and does not reject the independently approved
performance result or diagnose a production correctness failure.

Named supporting reads outside the diff were limited to the unchanged external
route and shared inventory helper for the actual-copy custody risk, the supplied
reviewer contract, and retained source/gate/bridge records for provenance,
coverage and outcome preservation. No broad/repeated tests, benchmarks, source
repair, Git/index/HEAD/branch mutation, original evidence edits or nested agents
occurred. Only this requested report was written in the checkout.
