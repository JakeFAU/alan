# Rejected variants and excluded controls

These observations precede the final paired campaign. Their source labels,
exit statuses and evidence directories remain separate from the frozen
qualification. Task 4 reviewed the apparatus and its excluded smoke before
the confirmatory schedule began.

## Report-reader design

- **Scope-only presence marker.** It returned `members` for unpublished roots
  whose original read was denied by SQLite column authorization, including
  an empty matching set. Four unpublished controls reproduced false success.
  The published cases retained the original success result. Their preliminary
  control also exposed missing authorization-callback observations, so the
  corrected test covers reached denial and published short-circuit behavior.
  Raw overlay and RED are `task2-checks/rejected-presence-only-overlay.json` and
  `task2-checks/red-presence-only.log`.
- **Function-based presence marker.** SQLite authorizer IGNORE on `typeof`
  returned NULL and hid a retained freeze with a corrupt descriptor.
  The original report returned `unexpected EOF`. Raw source and RED are
  `task2-checks/rejected-typeof-marker.go` and
  `task2-checks/red-marker-function.log`. The accepted change uses
  function-free presence sums and retains point reads after denied
  speculative access. `task2-checks/green-presence.log` covers both controls.
- **Structural RED.** The original root/run statement counts were
  [2,2], [8,8], [16,16] at 1, 4 and 8 roots. The accepted version reads
  each batch once and compares the complete result with the frozen oracle.
  `task2-checks/{red,green}-structural.log` preserves both sides.

## Compact report bypass

A private mutation of an earlier retained admission to `{` exposed an
integrity check in the ordinary full report after successful unrelated
execution. The rejected overlay omitted only `finishRun`. The positive
case passed and the mutated case returned nil error. The corrected witness
asserts the original `io.ErrUnexpectedEOF` sentinel. The initial assertion
had expected `*json.SyntaxError` and exited 1 for that assertion alone.
The final bypass RED exited 1 for the intended lost diagnostic. This
supports the documented deferral of a compact result under the existing
contract. [Compact-result decision and reproducer](compact-result-decision.md)

## Qualification apparatus

- Initial excluded smoke source `f36c913b19febb7966e945de3748584b91f2b7d7`.
  Source freeze and four builds succeeded. The shell stopped with exit 130
  during final build input binding before any row was launched. Its parser
  would have rejected real driver progress on stderr.
- Corrected excluded smoke source `cd84a37446c34b42eed6857b23aa9335af6ea2ed`.
  Six sample exits were 0. The repeated weather test exited 0 and sealed
  three SIGKILL triplets. The wrapper failed to reconstruct a structured
  test2json event split across multiple output chunks. The final reader
  reassembles by package and test and rejects truncated evidence.
- Reviewed excluded smoke source `7679755c7298f229fa468b14da54bcfe1420c4f8`.
  Six sample exits were 0. Historical PASS events, a same-seed report
  comparison, three sealed repeated stops and independent readers passed.
  The deliberate `OpenRuns` null mutation exited 1 at
  `$.cuts[2].run_report.OpenRuns`. Nine Python controls passed. Six
  disposable-data mutations rejected wrong source, missing prefix,
  contamination, wrong contact identity and a false child exit.
  The native binding hashes selected executables and SDKSettings.json
  and records SDK identity. The remaining SDK content is a same-host assumption.

The local control roots are under
`/Users/jacob/.codex/worktrees/recoverable-investigator/alan/.superpowers/sdd/plan-report-scaling/`.
The compact file index records selected control/rejection artifacts, including
their byte hashes and original relative paths.
