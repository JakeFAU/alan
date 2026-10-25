# Phase 1D.1 final reporting-fix re-review

**Original P2: ADDRESSED. Specification verdict: PASS. Quality verdict: PASS. No new actionable findings in the fix diff.**

The correction distinguishes a valid durable receipt awaiting local settlement from an unknown external physical result. Together with the original whole-branch review, this removes the only identified implementation blocker for the declared draft-PR scope. Refreshed final qualification, artifact packaging and draft publication remain parent-owned; this scoped verdict does not imply merge, release, production readiness, or unbounded correctness.

## Scope and exact identity

Reviewed `/tmp/phase1d1-final-fix.diff`, implementer report, final source, permanent regressions, retained RED/GREEN logs and archive manifests. Compared every original source member against the retained whole-branch-review archive. Only `internal/executor/runner.go` changed among existing source files; `internal/executor/async_reporting_test.go` is the sole added Go source. The fix does not change transport, scheduling, transactions, operation identity, dependencies or schema.

Independently verified all 58 source file lengths/hashes and recomputed the sorted length-framed aggregate: **`9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63`**. Previous reviewed source remains `ee144b08e6abac10c704c990e83a80314326a1eac3956bc5ecf9489f8f0afd5a`. Audit result: `/tmp/phase1d1-final-re-review-audit.json`.

No checkout edits, commits or subagents. No complete suite or redundant probe rerun; the exact original independent regression was already executed unchanged on the correction, and its output and bytes were inspected.

## Finding disposition

**P2 at `internal/executor/runner.go:422–432`: ADDRESSED.**

For an allowed unsettled attempt, `finishRun` now calls the existing package-level `externalOutcome(ctx, s.conn, id)` validator while already holding the store mutex. This is the correct non-locking helper; calling the Store method instead would deadlock, and the patch does not do that.

- A genuinely absent receipt (`sql.ErrNoRows`) still adds the operation to `UnknownExternal`.
- A valid durable receipt makes the physical outcome known, so the report does not add it to `UnknownExternal`.
- Invalid syntax, invalid binding or other read/validation failures are joined with the original run error. Corruption is not silently converted into a known or unknown outcome.
- The valid-but-unsettled operation still reaches unchanged `UnresolvedIntents` and `PendingDeliveries` construction. The settlement failure and `Quiescent=false` remain visible.
- No live dispatch, settlement or application mutation is performed by reporting. Non-idempotent recovery still settles/consumes the retained receipt without a second transport request.

The permanent regression verifies all of these observable known-receipt reporting properties and reopen/replay behavior. Its two corrupt-receipt cases verify syntax and binding errors remain exposed alongside the original settlement error. The new early return on corruption follows the existing report-error convention; a report accompanying a corruption error is not asserted to be complete.

## Evidence checked

1. Original permanent-regression RED: known receipt was marked Unknown, and both malformed-receipt cases suppressed the outcome validation error. All three expected failures occurred before the runtime edit.
2. Focused normal and race GREEN: five repetitions, 35 passing test/subtest events in each saved log, no failure. Group includes known receipt, malformed receipt, authentic old epoch/forged candidate, late source closure, and receipt-persistence-failure uncertainty.
3. Original independent expected-report overlay: verified its current bytes equal the retained original test. Saved `-race -count=5` output has five passes, one total endpoint contact across failure, reopen and replay, and no unresolved/unknown result after successful recovery. No assertion was weakened.
4. Verified complete pre-fix-plus-new-regression archive: 58 members, every length/hash matches, archive SHA256 `5707a3159e35cbc73abe58a6944463eb2817846bfc30459e8255e8ca2c6e49d4`. The old runtime and permanent failing input remain reproducible.
5. Verified corrected aggregate and exact two-file change boundary against the preserved original archive as described above.

Logs and manifests are retained under `docs/foundational/phase1d1/evidence/final-fix/`; the original whole-branch source archive and gate logs remain separately under `evidence/pre-final-review/`.

## Readiness and limits

The original finding is closed; no issue is deferred or parked. No new breakage was found in the correction. The original review's strengths, evidence assessment and explicit declined judgments continue to apply to unchanged source: finite controlled progress/crash evidence, one automatic Run owner, single-host/static profile, context-budget rather than hard wall-clock cleanup, and no generalized exactly-once or production claim.

At this report's initial writing, parent was refreshing the complete normal/race qualification, final archive and documents. Final packaging must carry the corrected source identity and preserve the earlier review/RED under its original identity. This report approves the code correction and tests; it does not independently assert completion of gates not yet presented or publication actions not yet performed. Final artifact verification may be appended once parent supplies the completed metadata.

## Final artifact identity closeout

**Artifact verification: PASS.** The parent has now supplied the refreshed qualification and offline-reproduction artifacts. The earlier packaging caveat is satisfied; commit/push/draft-PR publication remains the next parent action.

Identity-only audit independently verified:

- All 58 current source lengths/hashes and the length-framed aggregate still equal `9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63`; qualification-tool and retained probe hashes match too.
- Final archive SHA256 is `cb0763f6a9e23a1c9a35f647b4658d61adba16a8acaddc40767e70a390b5d535`. Its exact 95-member inventory, every member length/hash, archived source manifest, and archived bytes corresponding to current checkout files match.
- All 431 qualification-index rows identify the corrected source. Every evidence pointer matches the exact saved record hash and identity fields; each mode contains 214 records.
- Raw compressed/decompressed normal and race log hashes match their command metadata. Each log contains 357 passing test/subtest events and no failure event. Recorded durations are 19.801s normal and 102.571s race.
- Offline-reproduction metadata identifies this same archive/source, verifies 95 members and an initially empty module cache, and records successful full normal tests (39.266s), module verification and vet. I inspected and identity-checked this execution record; I did not repeat that build.
- Preserved pre-review archive remains SHA256 `4ea1db7953fdc754941ea45b35fa2c5d806a36309cadd1a149af7185af5c38fd` under original `ee144...` source identity. Its separate normal/race logs retain their original identities, hashes and 353 passing events. They were not relabeled as corrected-source evidence.

Exact audit script/output: `/tmp/phase1d1-final-artifact-audit.py` and `/tmp/phase1d1-final-artifact-audit.json`. No tests, source/document edits, or new broad review were performed for this closeout. **No remaining review blocker for the authorized draft PR was identified.**
