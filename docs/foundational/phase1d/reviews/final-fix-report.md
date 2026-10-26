# Final fix report — documentation-only M1

Status: M1 correction implemented and ready for the parent's scoped independent re-review. No scoped recheck is claimed. No commits or subagents. No Go/runtime/model/test/source-manifest/archive/existing machine-evidence changes.

## Finding and correction

The whole-branch review's complete actionable list contains only M1 (Minor documentation evidence precision), with specification PASS and quality PASS subject to this correction. No Critical/Important or source finding was reported. I verified the original normal and race records independently: `after_new_request` retains worker/generation2, sequence0, status open. The corrected rejected-result crash row says that generation2 is created and waits for ordinary input. Actual successor computation is explicitly attributed to the accepted nested crash fixture and the separate conformance rejection extension.

The full original report, original inaccurate wording and both complete records remain unchanged in the copied reviewer package. The temporary reviewer Go probe is retained as `.go.txt`, outside compiled source. The 03-review addition records the actual whole-branch verdict, M1 implementation disposition with scoped re-review pending, successful finite probe/full-suite/audit results, original sandbox failures and the declared limitations. It does not broaden the runtime profile or promise release/merge readiness.

## Changed files

Existing documents modified:

- `docs/foundational/phase1d/02-qualification.md`
- `docs/foundational/phase1d/03-review.md`

New verbatim reviewer artifacts (all copied from `/tmp/alan-phase1d-final-review`; exact byte equality checked):

- `docs/foundational/phase1d/reviews/final-audit.py` — 2770 bytes, SHA-256 `7c5f4f296f6f62afb4c2b2c7f35cfcfd470324794556922880a7dd65d91b410c`.
- `docs/foundational/phase1d/reviews/final-authorized-tests.log` — 382 bytes, SHA-256 `61b052946a03aa7221e990fd7206f45d8caf89deedcf740bbcd691988ee8a0b7`.
- `docs/foundational/phase1d/reviews/final-crash-tests.log` — 1107 bytes, SHA-256 `12afcc9b2a4ed662608bf34c83ec967e2080830ffe802d61336c2f2170b962e9`.
- `docs/foundational/phase1d/reviews/final-evidence-audit.log` — 549 bytes, SHA-256 `1760e88885567e4e71995cf8d8358ea2a8a823a19fdcc97d0e27918e94303165`.
- `docs/foundational/phase1d/reviews/final-focused-tests.log` — 1484 bytes, SHA-256 `54938047a1ab6167c22cb16afcb59249f568c7d14168872e5f4926f61ee30762`.
- `docs/foundational/phase1d/reviews/final-minor-evidence-precision.json` — 81203 bytes, SHA-256 `342c0b0f402397397f485517a9fad4b9fa045667c8c93ef4c6a3359fc2e87f78`.
- `docs/foundational/phase1d/reviews/final-preservation-audit.json` — 776 bytes, SHA-256 `753ef0ba307d570e68cd3a02973192bda8e74aaa4b602763d00e67e70c22785c`.
- `docs/foundational/phase1d/reviews/final-review-probe.go.txt` — 3903 bytes, SHA-256 `0a49f745104800abdd95c43b01b4680b6ba10bb920055d0087f68b1a673149fa`.
- `docs/foundational/phase1d/reviews/final-review.md` — 10886 bytes, SHA-256 `efed698c0cc9e94ed21fa9d024671709b02cf778b2dfc8d643a9455b56c2df3e`.
- `docs/foundational/phase1d/reviews/final-schedule-probe.log` — 2635 bytes, SHA-256 `46a8451538adb19339b8aad3017bd1df393f3e26ccac34a8b9d83fba45cf6f7f`.

This scratch report is the only other file written in the repository for this fix wave. No publication checkbox was changed.

## Verification

`python3 docs/foundational/phase1d/reviews/final-audit.py` exited0. Exact output is preserved at `/tmp/alan-phase1d-final-doc-fix/verification.log`:

```text
all source entries match: 53 digest: 40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f
archive manifest keys ['source_sha256', 'archive', 'sha256', 'bytes', 'archive_scope', 'members']
archive sha256 49f68316357b07ac4bbd3f979cb47ebe25e732a28e3375081fb9ccacfcd8f3ea
archive entries verified 83
normal passing events 321
normal exact extracted records 160 durable 87 matched killed PID/config 16 fault detections 3
race passing events 321
race exact extracted records 160 durable 87 matched killed PID/config 16 fault detections 3
PASS
```

The complete 53-file source digest remains `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`; all per-file raw-content hashes and byte counts match. Archive SHA-256 remains `49f68316357b07ac4bbd3f979cb47ebe25e732a28e3375081fb9ccacfcd8f3ea`; all 83 member hashes/lengths match. Both published raw streams reconstruct exactly to their existing 160 marker records, including 87 DURABLE records and 16 matching killed-PID/config identities per mode.

A before/after SHA-256 inventory of every existing file under `docs/foundational/phase1d` verifies that only 02-qualification and 03-review changed, with exactly ten new review artifacts. Thus all existing manifests, archive, qualification records, raw logs, old review records and historical evidence are unchanged. Inventory and result: `/tmp/alan-phase1d-final-doc-fix/before-files.json` and `documentation-verification.json`. All local links in the two edited documents resolve. `git diff --check` exited0 with no output. No Go test rerun was needed for this documentation-only correction; no fresh runtime test result is claimed.

## Exact document diff

The following is the exact unified diff against the documents captured before this fix wave; also saved at `/tmp/alan-phase1d-final-doc-fix/exact-document.diff`. New review artifacts are byte-identical copies as listed above.

```diff
--- before/docs/foundational/phase1d/02-qualification.md
+++ after/docs/foundational/phase1d/02-qualification.md
@@ -41,7 +41,7 @@
 | `TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash/authorized`, plus `TestAutomaticExternalAuthorizationCrash` | Authorization retained separately; no known result yet. Original committed intent remains the authority for recovery. |
 | `TestAutomaticApplicationMissingReceiptProfiles` | Separate resource acknowledges its own committed mutation; executor pauses before receipt insertion and dies. Resource survives. Idempotent recovery returns the original outcome with one mutation; non-idempotent recovery leaves unknown evidence and does not blindly resend. |
 | `TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash/accepted` | Target Retry result and worker generation 2 committed; supervisor result unconsumed. New process retains inner generation 1, obligation, pending ID and spent=1; original result is consumed and actual successor computation completes. |
-| `TestRejectedResultCrashAndLostRejectionControl/lost-rejection=false` | Original operation rejected while worker is active under valid matching authority. Worker later commits failure; reopen/retransmission returns the identical rejection. A distinct authorized operation succeeds and its successor computes. |
+| `TestRejectedResultCrashAndLostRejectionControl/lost-rejection=false` | Original operation rejected while worker is active under valid matching authority. Worker later commits failure; reopen/retransmission returns the identical rejection. A distinct authorized operation creates worker generation 2, which remains open at sequence 0 waiting for ordinary input. Actual successor computation is established by the accepted nested crash case and the separate conformance rejection extension below. |
 | `TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash/terminal` | Terminal outcome and notification obligation committed, delivery absent; reopening delivers that frozen result once logically. |
 | Same test, `second-recovery-crash` | Second process is killed after renewed activation with the same pending operation; third process continues same obligation with a later epoch and no extra budget charge. |

--- before/docs/foundational/phase1d/03-review.md
+++ after/docs/foundational/phase1d/03-review.md
@@ -135,3 +135,22 @@
 ## Final whole-branch gate

 All five scoped implementation reviews passed. The complete frozen source, final verification records and source archive are supplied for a fresh independent whole-branch review; its actual verdict and any findings are recorded below when returned. Scoped approvals are not a proof and do not substitute for that gate.
+
+
+## Final independent whole-branch review and M1 correction
+
+The fresh [whole-branch review](reviews/final-review.md) returned **specification PASS, quality PASS with one Minor documentation correction before publication**. It found no Critical or Important issue and no observed runtime defect. Its recommendation is readiness for a draft PR after M1 and publication/evidence bookkeeping are completed, not a merge or release recommendation. This review independently examined the complete implementation and qualification package; it was not a concatenation of the five scoped approvals.
+
+The reviewer verified all 53 source entries and digest `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`, all 83 archive members, all 160 extracted marker records per normal/race mode with exact Package/Test/order context, and the 16 SIGKILL identities in each mode. The [audit script](reviews/final-audit.py), [audit output](reviews/final-evidence-audit.log), and [historical preservation audit](reviews/final-preservation-audit.json) are retained verbatim. A separately authored [schedule probe](reviews/final-review-probe.go.txt) passed twenty deterministic seeds of up to eighty lifecycle schedule choices using public APIs; [probe output](reviews/final-schedule-probe.log) and the [fresh full-suite output](reviews/final-authorized-tests.log) are preserved. The probe used Replay as a consistency check, not an independent semantic oracle, and is stored as noncompiled `.go.txt`. The original [focused](reviews/final-focused-tests.log) and [controller](reviews/final-crash-tests.log) attempts preserve their sandbox socket-binding failures; the authorized local-loopback rerun passed. The reviewer inspected the published race/static/historical-reproduction evidence rather than rerunning each of those checks.
+
+**M1 — Minor, documentation correction applied; scoped re-review pending.** The rejected-result crash row in [02-qualification.md](02-qualification.md) previously said that the distinct new operation's successor computes. The normal and race `after_new_request` snapshots actually retain worker generation 2 open at sequence 0, waiting for ordinary input. The row now states successor creation and waiting, and attributes actual successor computation to the accepted nested crash case and the separately labeled conformance rejection extension. The [original wording and both complete records](reviews/final-minor-evidence-precision.json) are preserved unchanged. The original reviewer report records the finding as observed before correction. No runtime, Go test, model, manifest, archive or existing machine-evidence file changed for this correction. This is the complete actionable whole-branch finding list; the correction has not yet received its scoped independent recheck.
+
+The whole-branch verdict retains the declared limits:
+
+- Evidence covers selected finite programs, schedules and process-crash boundaries on one active host/process under local SQLite/filesystem assumptions. The schedule probe is reachable-consistency evidence, not exhaustive scheduling, arbitrary-program safety, whole-machine bisimulation or an unbounded correctness proof. Power loss, disk failure, initialization interruption, replacement of a live database, hostile aliases/external writers, arbitrary coherent database tampering and distributed fencing are not established. Read-only WAL bookkeeping is distinct from semantic dispatch.
+- Model/runtime equality applies only at named common macro-boundaries. Runtime Receive and pending-clear Set have a real unmatched intermediate; global epoch renewal, explicit external control inputs and closed-run receipt observations also differ. The external J endpoint is in-process with separate durable state; the controller suite supplies process isolation. Replay checks runtime consistency and is not the independent model oracle.
+- Execution-key deduplication is not business exactly-once behavior, business deduplication, compensation or inference of an unknown outcome from failure. Fresh demo stores reuse execution-derived keys; switching profiles requires a fresh executor/resource pair. No cross-store business-key namespace is provided, and non-idempotent uncertainty can persist indefinitely.
+- The interpreter has one PC per run, checked int64 data and explicit finite byte/work bounds. Malformed operands, unsupported effects and capacity limits can stop or leave invocation work unresolved. Generation overflow returns an operational limit without a committed target result. Successful progress witnesses do not establish arbitrary-program fairness or eventual recovery.
+- Histories are unbounded in storage growth, with no garbage collection; startup/replay scan retained history. Schema/program migration, multi-host operation, ownership transfer, dynamic subjects, fresh-supervisor child reassociation, cleanup custodians, providers and deployment are excluded. Grants are static and local transport provenance is trusted.
+
+These limits agree with the durability contract, qualification and runbook; neither the review nor M1's correction broadens the implementation profile.
```

The two whitespace-only context lines in the displayed diff are normalized for repository whitespace checks. The [exact original fix report](final-fix-report.md.gz) retains every original byte.
