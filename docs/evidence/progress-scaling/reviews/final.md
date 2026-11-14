# Independent whole-branch review — P6 execution progress

## Reviewed identity and method

Base: `e43e4f6be4500d177b50caf29181608d09b99c62`.
Head: `7314f3d63e75fbd2a367bfcb5ae35e4f566c1ec9`.
Tree: `932d9b1490e49fbd7983830420d85b8039124a37`.

The worktree was clean at review start and after the read-only checks. The supplied review package matches `git diff -U10` for this exact range. I read the binding contract and plan, production and test changes, public documentation, measurement and qualification apparatus, and evidence prose. I used the requesting-code-review/code-reviewer.md standard and Go performance/benchmark guidance. Prior task approvals were context, not substitutes for inspecting the code and artifacts.

Outside-diff reads were limited to named risks: common runner cleanup and report completion, Close coordination, weather approval/snapshot handling and Inspect/Replay projections, the inherited final workload verifier, historical-reader names, and the acknowledged SIGKILL/archive hooks. No tests, builds, providers, or reader binaries were rerun. Verification below consists of read-only Git/source comparisons and recomputation/hash checks of retained artifacts. No tracked file, Git index, branch, or commit was modified.

## Strengths

### Semantics and compatibility

The implementation is a small explicit contract split. `Run` and `RunProgress` use the same private engine; the latter projects only the three promised fields. Entry validation, single-runner exclusion, execution transactions, scheduling, limits, worker cancellation/join, detached bounded result accounting, error joining, and owner release remain common. The only mode-dependent operation is the final `finishRun` call. That function and the other existing helpers are unchanged. Setting quiescence before the full call preserves its prior value even when the report traversal adds an error.

The new tests cover genuine static and dynamic execution parity, unknown effects, per-call limits, entry failures, cross-method exclusion, Close waiting for an adapter, cancellation with retained results and joined errors, and reached malformed attempt/result evidence. The unrelated corrupt-admission test explicitly demonstrates the narrower diagnostic contract while retaining full Run's EOF diagnostic and subsequent inspection, Replay, and Open rejection. Existing tests were not edited.

Weather changes only the discarded-report invocation. Its fresh Open, before/after typed inspection, explicit approval policy, and evidence-derived uncertainty remain intact. The public example and documentation explain per-call counters, possible committed work on error, and quiescence without completion or remote-finality guarantees. No cache, schema, dependency, ownership, scheduler, retry, or BoundedSeries change is present.

### Measurement and source attribution

I mechanically checked both source bridges. Measured source `f8ecf081867b29012bbf1688eddfa067abbc1e43` to reader correction `fe5baf86ca285fc2496e2ed679c00d35bf119924` changes exactly the two declared Python checker files. Correction to reviewed head changes exactly the 14 listed delivery files plus the bridge itself. Every listed SHA-256 matches the current content. Production, Go tests, measurement source, and inherited helpers are unchanged across those transitions.

The three arms use the same transformed driver, identical progress/contact checks, and the same final identity/approval/charge/result verifier. Explicit final full reports, typed inspection, completed reopen, and Replay remain in the lifecycle. The complete original per-position proof is removed equally from all arms. Timing namespaces are not mistaken for same-seed identity evidence.

I recomputed every one of the 147 row resource/cardinality summaries from actual raw output and reaped RSS, validated the complete schedule and build identities, and recomputed the complete timing summary with exact equality to the tracked summary. There are 21 excluded warmups and 126 measured rows. Recorded primary child intervals are serial; no other retained gate overlaps their window. The B256 Execute/lifecycle medians reproduce as baseline full 4.543353/6.405379 s, candidate full 4.585003/6.424068 s, and candidate progress 1.298190/3.156178 s.

All six counter summaries recompute from the raw registries, including the complete report partition. Profile producer binaries, input profiles, rendered text, and quoted allocation attributions match retained originals. Documentation distinguishes cumulative allocation activity from peak RSS, discloses full-control overlap and small-bound limits, retains final audit cost, and does not turn this direct-adapter measurement into a weather CLI latency or linear-executor claim. Remaining scans and the unisolated history-insert attribution are qualified appropriately.

### Qualification, recovery, and retained failures

All 536 compact gate records match their raw commands, outcomes, bound-file hashes, and 53 referenced environments. The exact aggregate is **535 exit-0 records and one exit-1 pre-freeze smoke record** (`task3-parity-smoke/weather-parity`), with no timeout/spawn failure. The dispatch brief's assertion that all 536 exited 0 was incorrect; the executor acknowledged the correction. The tracked qualification prose already preserves that smoke failure at `docs/evidence/progress-scaling/qualification.md:83`. It is not an unresolved frozen qualification failure.

I recomputed the root/focused suite pass counts from raw JSON and confirmed all four required historical-reader PASS events in all five suites. Ordinary/integration/race-integration report 1,424/2,706/2,706 test passes; focused race selections report 173/294. The only integration skip is the disclosed opt-in historical S2 falsifier. All nine qualification root/nested Staticcheck configurations have empty stdout/stderr and successful recorded exits. Build, vet, module, formatting, oracle, independent-consumer, nested-module and outside-rebuild records are bound to their recorded raw artifacts; these are retained results, not fresh executions in this review.

I independently checked actual same-seed JSON: all 16 cuts match for each success, refusal, and unknown-third/approved-fourth scenario, with four contacts. The opposite-method retained-result resumes match, with 14 reductions, one local dispatch count, and one total original HTTP contact. All 14 SIGKILL/reap records match actual retained process records, and their archived main/WAL/SHM bytes match recorded hashes. The inherited hook archives after actual reap and before readers.

The original qualification controller remains exit 1 for the two copied-reader comparison failures. Source inspection supports the correction: Inspect emits detail fields and response bodies while Replay emits verified history count and the shared projection. The corrected comparator retains strict comparison of all shared fields and the commit count; its negative controls cover material identity/evidence mutations. I recomputed all 13 fresh copied-reader comparisons from their 26 raw outputs. Recorded main/WAL hashes remain unchanged; SHM changes after all 26 reader observations are preserved and disclosed. The original failed controller, pre-freeze smoke, profile setup failure, and scoped successful corrections are not relabeled as passes.

## Issues

Critical: none.

Important: none.

Minor: none in the reviewed branch requiring correction. The handoff aggregate-count error above is recorded explicitly and was acknowledged; the tracked evidence and qualification prose retain the correct failures.

## Recommendations

Preserve the exact reviewed source identity, original failed attempts, and this correction in delivery records. Any later report-copy/documentation commit needs its own exact content bridge; this verdict does not attest to that future commit or its contents. No executable change or additional suite run is requested by this review.

## Declined to judge

- Protection against hostile concurrent SQL writers, unsupported private mutations, or arbitrary physical corruption: outside the existing owner model; progress explicitly promises no new whole-history validity certificate.
- General distributed ownership, remote finality, exactly-once effects, or guaranteed completion/liveness: outside this single-host execution/report contract and not established by these crash experiments.
- Universal asymptotic or latency behavior, other workloads/hosts/toolchains, and weather CLI end-to-end speedup: the retained measurements qualify only the bounded controlled fixture and host.
- An experimentally isolated cause for the residual history-insert scan or extra execution-stamp reads: the report explicitly leaves those causal attributions unresolved; this change does not alter those paths.
- A hermetic freeze of every native SDK header/library or a claim of public raw-archive availability: only the documented selected tools/source/SDK settings and local artifact bindings are supplied.
- Future delivery commits, remote branch/PR state, merge, or release readiness: not present in the reviewed tree and outside the authorized draft-delivery verdict.

## Assessment

**APPROVE for draft delivery.**

Ready for the authorized draft PR: **Yes**. The implementation preserves the shared execution and accounting boundary while making the omitted report diagnostics explicit, and the controlled benefit is supported by retained semantic, recovery, compatibility, and resource evidence. This is approval of the exact reviewed branch for draft delivery, not authorization to merge or release.
