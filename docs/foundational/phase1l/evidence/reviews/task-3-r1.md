# Phase 1L Task 3 independent runtime review

Specification compliance: **REJECT**. Code/research quality: **REJECT**.
Two reproduced raw-evidence validation defects block the claimed authentic
history qualification. The saved executions themselves support the reported
positive/control/order/recovery observations; these findings concern the
adapter's ability to reject contradictory or uncommitted evidence before
normalization. Neither finding is a runtime/reference semantic counterexample.

## Blocking findings

### R1 [P1] Bind first-authorization revision to the actual retained source

Location: `internal/substitutiontest/invocation_join_evidence_test.go:287–293`
and `:563`.

The decoder checks that the intent row's source_revision, attempt descriptor
Revision and Authorization.Source.Revision agree with each other. It checks all
subject revisions are1 separately, but never links these values to the actual
source subject, commitment history revision or admission Invocation.SourceRevision.
It then supplies RevisionMatches=true to the independent reference unconditionally.

Reproduced on the frozen saved complete B inspection: change only e's intent
source_revision from1 to2, its matching attempt descriptor Revision to2 and
Authorization.Source.Revision to2, then recompute that authorization fact's
record digest. All subject, commitment history and admission source revisions
remain1. The adapter returns no raw error, RawValid=true, and empty Prefix,
Component and Guarantee violations. Thus mutually agreeing altered rows are
accepted as the original stable authority snapshot despite contradictory retained
source evidence. The digest is not an independent authority check.

Required correction: authenticate the revision against the actual original source
subject and commitment/admission/history links before asserting RevisionMatches.
Apply it on incomplete and complete paths, preserving legitimate epoch renewal
on Open. Add the detached coherent mutation as a regression; simple digest-only
corruption does not exercise this defect. Do not weaken the frozen reference.

### R2 [P1] Require every retained intent to have its exact original commitment

Location: `internal/substitutiontest/invocation_join_evidence_test.go:207–216`
and the coverage check at `:709`.

The shared helper checks history intents against the intent table in one
direction. The new adapter's final coverage check accounts for histories,
authorizations, admissions, replies and publications, but not the inverse
intent-to-committed-history relation. Terminal freeze checking also ignores an
intent if its run/sequence has no execution fact.

Reproduced on the same saved B inspection: append a copy of e's invocation intent
with a fresh ID and Sequence999, leaving all history, execution, admission and
charge rows unchanged. There is no commitment history or execution fact for that
intent. Nevertheless the adapter again returns RawValid=true and empty Prefix,
Component and Guarantee violations. Normalization silently omits the extra work.
This is not a legitimate committed-but-unadmitted prefix: the commitment itself
is absent. An initial probe keeping the existing commitment sequence was
correctly rejected by freeze coverage; the missing-sequence variant exposes the
gap and both outcomes are retained.

Required correction: check a bijective original linkage between every intent
row and its exact emitting history/execution, including intent identity and
source metadata, before normalization. Reject orphan intents without requiring
valid committed intents to have been admitted, authorized or completed yet.
Retain this absent-commitment mutation as a regression and keep the current
incomplete-prefix positives.

## Evidence and source binding

Reviewed the brief, preparation checklist, complete seven-file added diff, four
new Go files, capture/audit/reproduction scripts, task report, freeze manifests,
index and record bundles. Independently verified all298 current source hashes,
the aggregate, indexed artifact hashes/sizes, protected inherited files, the
complete added diff against current file contents, and the immutable
review-freeze source/report snapshot. Rechecked source identity after probes:

`dd76f693f21070a43fe318667705ccead7a1ec47e6763f6c31267123cec68d6a`.

Final records.json.gz remains SHA256
`0beffb2debb917147c30371081ee19299b04c714dd285e8e6d8534836689f3cc`.
Its normal/race commands each selected36 leaves, exited0 and contain no failed or
skipped selected qualification case. These are retained implementer runs, not
new reviewer suite executions.

Reviewer executed two narrow Go overlay probes against an embedded retained
inspection only; neither calls Create, Open, Run, Commit or a runtime/census
campaign. Both Go commands exited1 because a corrupted snapshot was accepted:

- `go test -json -overlay=<temporary overlay> -tags integration -run
  '^TestTask3ReviewerRawCompleteness$' -count=1 -timeout=2m
  ./internal/substitutiontest` — coherent revision mutation falsely accepted;
  existing-sequence orphan and corrected extra-charge mutation correctly rejected.
- Same command with `-run
  '^TestTask3ReviewerRawCompleteness$/orphan-intent$'` — absent-sequence orphan
  falsely accepted.

Exact argv, pinned offline environment, output, overlay source/hash, baseline
inspection, all detached witnesses and the independent Python record audit are
retained in `task-3-review-probes.json.gz`,125,736 bytes, SHA256
`f9b4e363c0ae4bb37a76165e297fdae005e7a68844d1ac792b6144cbb2018437`.
The original baseline decodes with all four passing verdicts before mutation.
No worktree Go source, production/reference source, test expectation, TeX, commit
or nested agent was changed/created during review.

The prior extra-charge RED is inspectable and honestly retained with its
pre-correction adapter, probe and wrongly accepted raw witness. The current
correction checks accepted/refused admission-to-charge cardinality, per-admission
charges and root totals. The reviewer probe independently confirms the original
orphan fourth charge plus inflated root total now fails with
`admission/charge cardinality or root completion total mismatch`.

## Findings verified without a new campaign

A separate read-only Python audit derived requests from retained parent initial
state, checked every recorded intent's actual commitment/source revision,
selected-row/consumed-sequence links and publication envelopes, and independently
walked every parent instruction path in the decisive normal/race snapshots. The
actual source records in these snapshots do not contain R1 or R2 corruption.

- Eight positives in each profile cover exactly A/B × id1/2 × value0/1, answer.
  Candidate/S and parent IDs match the approved mapping; both original results
  are correctly present and selected once, parent/root succeed, fixed1/E0 and
  A2/B3 dynamic charges hold under the common allowance.
- Eight controls in each profile preserve separate raw, prefix, component and
  G outcomes. Wrong-id is actually consumed as succeeded/present id2,value1;
  raw/prefix accept, component/G reject, and parent/root fail after both joins.
  Scarcity actually refuses helper with invocation_limit and totalI/R2; code does
  not prescribe the loser. Fail/absence preserve both joins; missing responses
  stay absent. Pending candidate/root waits at PC2 and pending S/root at PC3;
  no terminal root outcome is invented. S controls remain outside positive S.
- Ordered raw publications show candidate-first e29/s37 versus S-first s13/e36
  in both profiles. The actual parent selection sequences remain e3 then s4.
  The S-first intermediate is retained-unconsumed S with unfinished candidate.
- Recovery before-state is root open at PC3, original e selected once, original
  s required/unconsumed, admitted S sequence0/PC0/request unconsumed, no S
  publication, all three admissions/charges retained. ACK checks relevant event
  fields plus PID/store/token/scenario/boundary and the independent predicate.
  No Store reentry occurs under a boundary lock.
- SIGKILL exits and fresh Open/Run completion are retained for normal
  PIDs73179/73183 and race PIDs73229/73253. Original handles, programs, generations,
  consumed rows/sequences and charges persist; no new invocation/run is added,
  only S's original reply intent. Each original outcome is selected exactly once;
  epoch2 becomes3. Both starts use the same os.Executable and binary SHA256; the
  race flag follows the same race-built executable, rather than a plain child.
  Replay preservation is asserted by the source-bound test, separately from the
  independent raw checks.

The actual parent guard path check, explicit source-sequence ordering, original
request-message binding checks, verified atomic publication/delivery expansion,
control classifications and process construction are useful and remain within
the narrow intended scope. No additional blocking finding in those mechanisms
was established by this review. Fixing R1/R2 still requires affected focused
checks and independent re-review of the new frozen package.

## Remaining gates and qualification limits

The inherited default nonintegration attempt's10m watchdog in
examples/adaptive/TestAdaptiveWorstCaseBytes/nested=false/sixteen is retained,
not relabeled as success. Per controller handoff, only its two unfinished tests
are separately running at README's documented60m bound. At this review the
adaptive-watchdog-followup.json.gz disposition is not yet present. This pending
controller gate is distinct from both the successful captured Task3 commands
and the two blocking decoder findings.

No routine suite, model census, public-consumer execution or recovery campaign
was rerun by the reviewer. Manuscript reconciliation, default-suite disposition,
final static checks and draft PR remain controller work. General congruence,
arbitrary contexts, distributed/effectful recovery and universal compiler
correctness remain unestablished. This rejection does not erase the passing
bounded model or authentic concrete observations; it withholds approval of the
current evidence-validation package until the reproducible gaps are corrected.
