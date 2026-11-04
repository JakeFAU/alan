# Task 1 independent specification and code-quality review

**Specification verdict: APPROVE for the current-runtime baseline task.**
**Code-quality verdict: APPROVE.** No blocking production or fixture finding.
**Archive condition: regenerate the affected exports on frozen fixture source
before final archival qualification.** The reviewer-induced evidence-handling
incident below is separate from these source verdicts.

Reviewed the Task 1 brief, report and complete `task-1-review.diff`; the Phase
3B.3 packet (`00-brief.md`), runtime gap (`01-runtime-gap.md`) and plan; all three
actual changed files; relevant public-runtime permission/inspection source; and
the selected raw retained/typed/history/endpoint evidence. Runtime HEAD is
`af6943a5805a9a6c0e8fe77f804264f4765d07ec`. The three new files exactly match
the complete supplied diff and archived source SHA256 values. Tracked production
diff is empty; only `examples/delegation/` and the new integration fixture are
untracked. No checkout source edits, commits or subagents were used by this review.

## Findings and remaining gates

- **P1/P2 source findings: none.** The fixture does not broaden A's permissions
  to manufacture S2, introduce production behavior, or claim new delegation.
- **P2 evidence handling, reviewer-induced:** opening archived endpoint SQLite
  databases with Python `sqlite3.connect('file:'+path+'?mode=ro', uri=True)`
  changed 24 endpoint `-shm` sidecars across the eight selected run directories.
  Database, WAL, exported raw/typed/history JSON and endpoint contents did not
  change. Every original selected manifest entry (183 total) matched before
  these reads. The subsequent digest check detected the SHM changes. The
  independent SQL checks still matched all 24 endpoint databases to their
  exported contacts/applications/key records.

  The changed bytes were preserved under
  `/tmp/alan-phase3b3-baseline-review/review-observed-shm/` with expected and
  observed SHA256 values in `review-shm-side-effects.json`. Before the parent's
  instruction against reconstruction arrived, the reviewer reconstructed the
  changed WAL-index header fields/checksums, accepted candidates only when their
  complete-file SHA256 matched the initial manifest, and wrote those exact-match
  candidates back. Recovered candidates remain in `exact-recovered-shm/`.
  Thus all 183 entries currently match again, but this is **not untouched
  archival custody**. Preserve the incident and original expected-failure
  provenance; regenerate affected exports on the frozen fixture revision as
  directed by the parent. Do not relabel reconstructed sidecars as originals.
  This is not a defect attributed to the submitted fixture.
- **Required delivery gate:** fixture source remains uncommitted. The source
  hashes bound this local review; they do not discharge the plan's immutable
  source qualification requirement. Freeze before final qualification or relying
  on the baseline as an immutable implementation prerequisite.

## Specification checks

1. **Two ordinary roots and fixed authority:**
   `examples/delegation/baseline.go:45-90` constructs A/B through the public
   authoring API and removes automatic A adapter grants before provisioning.
   Exact remaining tuples are A/counter-A/red/increment and
   B/counter-B/blue/increment; the unit test checks this for all five scenarios.
   `internal/executor/adapters.go:128-131` requires a complete exact tuple rooted
   in the actual serving subject. B's permission is not an alternative A path.

2. **Authentic custody and logical generations:** the integration helper at
   `phase3b3_baseline_test.go:228-279` checks original request producer, selected
   request, service, selected adapter result, reply producer and selected reply.
   I independently rederived 10 custody chains and 13 adapter permission decisions
   from raw `inspect.json` tables rather than treating typed inspection or the
   helper's output as its own oracle. Each request/reply admission, payload,
   original operation, source/target generation, history selection and consumed
   sequence joined correctly. Endpoint keys and complete returned receipts joined
   to exactly one allowed original attempt. This is a fixture-specific check,
   not the later independent delegation-authority checker.

3. **S1 apparatus and refused S2 attempt remain distinct:** canonical `local`
   has A/g1 operation `["a",1,6,0]` and B/g1 `["b",1,12,0]`, both validated.
   B selects A's reply at B/sequence 5 before B's service at sequence 12. Endpoint
   contacts A/red, B/blue, A/blue are `(1,1,0)`. Missing, claim and Cartesian
   attempts retain source authorization allowed and adapter `grant_denied`, select
   that refusal, and have `(0,0,0)` contacts. Their baseline does not realize S2.

4. **Cartesian interpretation is bounded accurately:** the main Cartesian case
   tries the absent A/blue scope tuple. The additional test at lines 417-481
   exercises the operation coordinate with
   `P={(counter-A/red,op1),(counter-B/blue,op2)}` and actually refuses
   `(counter-A/red,op2)`, while both legitimate controls execute. These test exact
   static tuple lookup. They do not establish attenuation of a future delegated
   grant or the future primitive's Cartesian rejection.

5. **Result laundering is an additional adverse schedule:** source and report
   explicitly reverse the positive policy. B/g1 `["b",1,6,0]` produces an authentic
   result that A selects at A/sequence 5 before its refused `["a",1,12,0]` attempt.
   The actual attempt carries the prior mutation. Contacts are `(0,1,0)`.
   This supports a prior-result negative control, not completion under the
   A-servicing/B-serial grammar. The four main cases preserve that grammar.

6. **RED assertion is meaningful:** lines 383-415 first execute and inspect S1,
   then require the missing-authority run's specific allowed-source/
   `grant_denied` cause and zero contacts, then assert both authentic results
   were consumed. It is not an unconditional failing test. The preserved command
   exits 1 at the desired-completion assertion, with original operation
   `["a",1,6,0]` identified. Holds keep both roots open; “completion” means both
   service results consumed (`paid=true`), not terminal-root completion or
   liveness. The report states this limitation.

## Additional independent control executed

The complete baseline integration/race suites were not rerun. I created disposable
backups of canonical S1 and changed exactly one `history.selected_observation`
at A/g1/sequence 7 from A's original adapter result to B's adapter result. Both
results have equal payloads (`validated`, mutation 1, total 1), so payload equality
does not explain the rejection.

Executed from `/private/tmp/alan-phase3b3-runtime`:

```text
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off go run /tmp/alan-phase3b3-baseline-review/main.go /tmp/alan-phase3b3-baseline-review/valid-copy.sqlite /tmp/alan-phase3b3-baseline-review/equal-payload-cross-result.sqlite
```

Exit 0: the valid backup passes public `InspectExecution` and `Replay`; the mutant
returns `ErrCorrupt` from both (original observation selection / substituted
observation selection). This rejects the malformed selected-history edge; it
does not independently prove runtime authority safety. Mutation details, driver,
command/environment record, raw stdout/stderr, independent raw-table checking
script and SQL checking script are under
`/tmp/alan-phase3b3-baseline-review/`. The final scripts use disposable physical
copies for all further SQLite opens. The original problematic reads performed
`sqlite_master`, `history` and endpoint `contacts`, `applications`, `keys` SELECTs;
their exact Python access pattern and SQL are preserved in this review and
`endpoint-and-run.py` (now corrected to copy first).

## Verification evidence inspected

Archived command JSON/log pairs show the focused unit/integration suites passing,
integration race passing, the RED test normally skipped, its opt-in invocation
failing at the claimed authority assertion, Staticcheck passing without
diagnostics, and Go 1.27.3 darwin/arm64. These are the implementer's retained runs,
not newly executed suites by this reviewer. Initial source/diff/digest comparisons,
raw custody/tuple/contact checks and the malformed-selection public-API control
were independently executed in this review. Earlier development/environment
failures are separately reported and are not semantic evidence.

Task 1 can support proceeding to source freeze and the separately reviewed
authority contract/implementation. It provides no new grant representation,
fresh-admission delegation check, revocation, recovery qualification, correspondence
result or manuscript theorem. Those remain later tasks.
