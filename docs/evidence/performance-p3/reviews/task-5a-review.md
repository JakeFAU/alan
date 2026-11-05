# Task 5a independent tooling checkpoint review

Spec compliance: **REJECT**. Task quality: **REJECT — needs fixes**.

Reviewed exact base `4f5fe2cc83316900a2f387180dd9f856db5303a0` through head
`59bc43a0e37d3b6b10b24a2090d64cc664a46970`. Read the complete 1,012-line review
package in sequential passes. The package contains eight new P3 campaign files;
no production, public API, schema, dependency pin, or P2 changes. No changed
source was reread outside the package. No Git operation, primary row, build,
broad suite, or nested agent was run during this review.

## Strengths

- `campaign/validation.py:186-250` checks the genuine unchanged DIRECT Prepared,
  scalar adapter journal and Candidate, including exact protocol/endpoint,
  request schemas/payload, operation/key/observation identity, Prepared/result/
  continuation-scope digests, zero HTTP fields and empty server journal.
  `driver/transport.go:204-206,258-264` independently establishes those actual
  direct wire shapes; this is not an HTTP proof relabeled as direct.
- `campaign/validation.py:109-179` retains the returned-cut/root/debit,
  membership, prefix arithmetic and residual proof. The inherited outer HTTP
  resource label is preserved and explained rather than normalized away.
- `campaign/run.py:33-40,43-92` exclusively creates row artifacts, preserves
  start/outcome records, checks pre/post custody and stops on failure.
  `run.py:75` saves the child outcome before post-exit metadata reads;
  `run.py:65` passes only the bound whitelist to the child.
- `campaign/custody.py:52-96` and `analyze.py:43-61` bind exact source/module/
  binary/dependency/tool identities, all 98 ordered commands and configuration,
  all 14 retained excluded warmups and 84 measured rows, and serial chronology.
- `campaign/analyze.py:21-31,87-95` implements the pre-data ruling from
  `progress.md:73`: improvement is 1 minus candidate median Execute divided by
  baseline median Execute, with all six B256 differences positive and paired
  ratios reported separately. The strict 5% equality case fails as required.
  No contrary interpretation was found.

## Important findings

### [P2] Exact frozen guard equality can be incorrectly rejected

`docs/evidence/performance-p3/campaign/analyze.py:17-18,77-81` converts integer
nanoseconds to floating-point seconds before subtracting and comparing the
max(10%,1ms) boundary. A baseline median of 1,000,000,000ns and candidate median
of 1,100,000,000ns produces `guard(1.0,1.1,.001) == True`, although the slowdown
is exactly 10% and must not exceed the strict guard. Likewise 9,000,000ns to
10,000,000ns is exactly 1ms and is incorrectly rejected. This can falsely trip
`guard_violation` and demand withdrawal/resolution for a result that meets the
frozen contract. `test_campaign.py:132-136` tests near 10% and one 1ms equality
but misses these representative exact equalities.

Keep the decision arithmetic in original integer nanoseconds (or exact doubled
medians/rationals) and compare the relative and absolute strict thresholds
without floating subtraction. Apply that exact guard consistently to all
small Execute and non-Execute/summed-approval phases; preserve the frozen
thresholds. Add exact relative-boundary and absolute-boundary tests, including
half-integer medians and values immediately on either side. RSS should use the
same exact comparison in bytes if sharing the helper.

Evidence: `/private/tmp/alan-p3-tooling-review-59bc43a/results.json`,
`focused_boundary`: both counterexamples have actual violation true and exact
expected violation false. This experiment invokes only the pure guard helper.

### [P2] Strict causal proof ignores retained approval decision order

`docs/evidence/performance-p3/campaign/validation.py:241-250` verifies the
continuation's payload, scope and selection but never validates its retained
`DecisionOrder`. Starting from the actual B8 candidate fixture, changing only
`evidence[1].approval.DecisionOrder` to `-1`, or to `19` after that successor's
root completion at order `18`, is accepted by `validate_report`. Both are
impossible for the declared serial public-approval workload. Thus the claimed
full causal approval proof can accept malformed or chronologically detached
retained approval evidence. `test_campaign.py:70-104` has no such negative.

Require an integer positive approval decision order and its proper relationship
to the predecessor and successor completed-root decision orders for this frozen
workload, rejecting missing/invalid/backwards/future orders. The unchanged
stopwatch calls the next public approval only after the previous Run and its
returned-cut proof (`driver/main.go:323-370`); the continuation history assigns
and consumes the original decision order (`internal/executor/continuation_history.go:117-140`).
Add discriminating negative controls without changing the driver or workload.

Evidence: `/private/tmp/alan-p3-tooling-review-59bc43a/results.json`,
`focused_proof_mutations`, plus complete retained mutated reports
`negative_approval_decision_order.json` and
`approval_after_root_completion.json` in that directory. Both report ACCEPTED.

## Checks and named risks

- **Frozen source/custody substitution:** independently compared actual source,
  driver module, binary, dependency, Go, Python and tool hashes to the final
  binding. All match. Binding SHA256 is
  `4be88671127cc9856609ea1b1a7fc8bdc1d8dfd4dac5f0df1499c22bac111c64`.
  Both exported tar contents exactly match their source manifests (607 A files,
  629 B files); archive hashes match. A is baseline `926a899b...`, B is candidate
  `4f5fe2cc...`. Retained build records and Go build-info bind each binary to its
  corresponding local source replacement, Go1.27.3/CGO1 and SQLite1.14.52.
- **Driver/P2 drift:** all 34 original P2 manifest entries match current bytes;
  both built driver copies differ from the original only in the declared
  `go.mod` replacement. The final tool-byte bridge changes only validation.py
  and test_campaign.py; every non-tool identity remains equal.
- **Missing raw evidence:** verified all 5,350 indexed artifacts against
  `raw-index-checkpoint.json`, with zero missing/changed entries. Index SHA256
  is `2176bd8dc66ff946f2fb963559bd930757ac9550cccf9ae5d4b09276295a2884`.
- **Genuine direct wire/stopwatch:** checked unchanged driver direct Prepare /
  Execute paths and `main.go:319-417` against the validator. Each Execute window
  encloses the N Run calls, N-1 fresh public InspectExecution/ContinuationBasis/
  ApproveContinuation cycles, progress and returned-cut/prefix work. No
  stopwatch or approval shortcut was introduced. Existing B1/B8 A/B raw reports
  all validate; no new driver process was needed.
- **False completeness/test claim:** read the final strict-tool log (9 passes),
  existing P2/P3 logs (22/5 passes), and both driver JSON logs (71 test/subtest
  passes each, no failures/skips, empty stderr). Existing missing/extra-row,
  metadata, custody, direct mutation and failed-child tests are substantive.
  The two findings above are uncovered cases; no reported suite was rerun.
- **Failure loss/credential capture:** inspected retained exit-23 outcome and
  the whitelist. Outcome persistence precedes post-exit custody reading;
  original sandbox loopback/syntax/setup failures remain in the hash-verified
  archive. No inherited credential values are passed or captured by this
  runner's explicit environment dictionary.
- **Control-input drift:** source manifests bind genuine candidate waiting-run
  fixtures and selector_probe.py; its three source-frozen cases are 32/4
  waiting, 32/32 waiting and 32/4 last-ready. These are ready inputs, not executed
  control evidence. Diagnostic materialization/execution and actual visited/
  distinct distributions remain Task5b obligations.
- **Guard/causal doubt:** ran only the focused pure-Python reproductions above,
  with scripts/results/stdout/stderr retained externally. The first reviewer
  script had an unmatched-parenthesis syntax error before execution; the failed
  script and stderr remain, and corrected `check-v2.py` exited 0. The 5% exact
  equality returned false and a one-nanosecond-better case returned true.

## Checkpoint disposition

**Do not authorize any confirmatory warmup at head59bc43a.** Fix the two findings,
freeze the corrected tools and replacement binding without changing the sources,
binaries, workload, schedule or guard meaning, and obtain independent re-review.
The actual campaign output directory does not exist. No performance conclusion
is supported or claimed here. Task5b's 98 rows, diagnostic controls/mechanism
counts/profiles and later full-suite/delivery obligations remain outstanding.

Review raw evidence: `/private/tmp/alan-p3-tooling-review-59bc43a/`.
