# Execution record

The research contract was committed at `5067bcf` before production changes.
Baseline source is `af6943a5805a9a6c0e8fe77f804264f4765d07ec`.

1. **Complete:** verified source, immutable export, environment and contract;
   ordinary (1,172) and integration (2,116) tests and both Staticcheck profiles.
2. **Complete:** public driver, separate aggregate probes, complete baseline
   timing/counter/coarse-timer matrices and CPU/allocation/trace profiles;
   independent methodology, instrumentation and source-boundary review.
3. **Complete:** invariant frozen at `ddcac16`; one private protocol-mode
   projection and regression tests committed at `e32d0e2`. Targeted ordinary,
   integration/race and Staticcheck pass; initial independent semantic approval.
4. **Complete:** candidate diagnostics, 25 exact differential comparisons,
   actual SIGKILL/recovery archives, adversarial reuse-premise checks, existing
   external consumer/historical fixtures, and independent semantic/recovery/scope
   reviews. Full ordinary/integration/integration-race and four Staticcheck
   configurations pass on immutable implementation `e32d0e2`.
5. **Complete:** all 98 predeclared paired processes and four separate interval
   diagnostics pass. Source manifests and frozen binary/tool hashes match.
   Independent raw-data recomputation approves the bounded performance result.
6. **Delivery boundary:** commit the reproducible tools and reports, then verify
   the exact immutable delivery head and retain its raw logs separately so the
   qualified source is not edited to record its own result. The draft PR's
   delivery record identifies that head, the post-commit qualification and the
   evidence archives. Do not merge or release.

Headline timing is serialized. Agents may inspect source or write reports during
measurement, but must not run builds, tests or other benchmarks concurrently.
Diagnostic instrumentation and profiling are excluded from headline timing.
