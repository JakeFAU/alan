# Task2c independent scoped rereview — round1

Reviewed 2026-10-26. Base `168f4d74b51a970def5996d23f01e9d01865ae07`; head `9e73d0fbdbfb9b58d6627f467b2ddd3c5f9ef421`. Exact current Go/go.mod/go.sum identity: `fc2b0465eea920d6b04cd3c3896c6be909ba5a6b292ed9ebf9c58ff6a1ebf282`.

**Spec verdict: Approved for this scoped Task2c fix and prior I1 closure. Quality verdict: Approved; qualityApproved=true.** I1 is addressed. No new Critical, Important or Minor issue was found in the fix diff. This closes the sole blocker from the original Task2c scoped review; it is not whole-branch or Phase1G application/process acceptance.

## Scope

Reviewed the exact source companion `.superpowers/sdd/phase1g-plan/review-task2c-fix1.source.diff`, appended Task2c report and its identical evidence copy, and fix-round command/log/manifest/verification evidence. The production diff is confined to the complete-profile read-only consumption validator; the only other Go change is the new focused test file. Requirements remain the original Task2c brief and contract R1–R8 already reviewed. No live writer, schema, public API, lifecycle policy, dependency or contract change is included.

No source/index/HEAD/branch mutation, new test run, broad suite rerun, live call, credential access or subagent. There was no concrete unanswered doubt requiring an additional probe. Read-only source/hash checks independently validated the reported verification evidence.

## Per-finding disposition

### I1 — ADDRESSED

Original finding: public Replay accepted a scoped ordinary-message consumption fact before its source authorization fact, despite source execution preceding both.

The fix at `internal/executor/composition_prefix.go:305` runs after the existing selected-observation predecessor validation. For ordinary-message and runtime-lifecycle observations it resolves the selected observation through its admitted descriptor to `d.IntentID`, derives that original intent's exact scoped authorization fact using `compositionSourceEvidence`, and requires its ID in the already validated `seen` prefix. It checks causal occurrence, not merely final authorization-row existence or another operation's authorization. The existing predecessor resolver validates the observation/admission/intent binding; the protocol fact validator separately checks each fact's identity/source digest. Errors propagate rather than being interpreted as absent authority.

The original counterexample now reaches the intended `corrupt durable state: observation consumed before source authorization` rejection. Original source input SHA256 remains `577a4fc817fcedad612d6092d2f1f144248db2e2000bc04befaed81413e4997c`. The input, original overlay mapping and original review artifacts are unchanged. The retained red output still shows nil Replay error; it has not been replaced by the green result.

The permanent test at `composition_consumption_test.go:10` reproduces real public Create/Commit/DeliverMessage/consumption, first accepts the valid history, then exchanges only the authorization and receiver-consumption fact order. Its helper verifies all ordinary inspection tables remain equal, exact fact identities/source digests remain equal except Order, and foreign keys are clean. The scoped assertion requires ErrCorrupt and the specific new authorization diagnostic, so an unrelated guard cannot substitute for this causal check.

### Preservation checks

- **Unrepresented sources:** `compositionSourceEvidence` returns an empty fact for an intent whose source has no resource-root membership. The new branch consequently imposes no invented authorization fact on that source. The complete-profile unrepresented ordinary-message positive control passes before and after the fix.
- **Foundation behavior:** the new code is in `compositionPrefix.validateConsumption`, used by complete protocol reconstruction. Foundation validation and `compositionObservationPredecessor` are unchanged. The unchanged Task2a causal tests pass all five ordering cases and all four unrepresented/local-input controls.
- **Lifecycle audit:** the real source Commit/end, denied AuthorizeIntent and SettleLifecycle schedule retains a late authenticated observation with zero lifecycle target-decision rows. Baseline Replay passes. Moving authorization before the source closing execution fails at the preexisting `authorization before source prefix` guard. Log02 confirms this control already passed before the fix. The source is closed and cannot consume the late observation; neither this report nor the fix report treats that control as a newly reachable denied-consumption regression. The new branch does not invent a target decision or change live policy.
- **Other observations:** terminal evidence keeps its existing source/target-decision predecessor path; invocation requests/results and adapter/receipt consumption remain in their existing specialized branches.

## Evidence verified

All recorded commands use `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`.

| Retained evidence under `review-fix1/` | Numeric Go exit | Reviewed meaning |
| --- | ---: | --- |
| `01-original-overlay-red.log` | 1 | Unchanged independent probe reproduces I1 on original source `32a2aa…`. |
| `02-permanent-regressions-before.log` | 1 | New scoped message test fails with nil Replay error; unrepresented and source-denied lifecycle controls already pass. |
| `03-focused-green.log` | 0 | Permanent tests plus unchanged Task2a causal rejection/positive controls pass on final source. |
| `04-original-overlay-green.log` | 0 | Unchanged independent probe reaches the exact new authorization rejection. |
| `05-executor-package.log` | 0 | Complete executor package passes on final source. |

Exact command arrays and numeric subprocess statuses were read from `commands.json`; complete outputs were inspected. Independently recomputed the current source manifest and compared every path/hash entry, as well as the identity, with manifests03/04/05: all exact matches to `fc2b0465eea920d6b04cd3c3896c6be909ba5a6b292ed9ebf9c58ff6a1ebf282`. Verified HEAD, the two changed Go paths, all original review-artifact hashes, unchanged original overlay input/mapping bytes, unchanged Task2a control file, and equality of report copies. Formatting records retain exit0/empty output.

These are verified retained execution records, not test executions by this rereviewer. Earlier default/integration/scoped-race suites remain evidence for their earlier `32a2aa…` source identity; they are not relabeled as having run on this fix.

## Quality assessment and new issues

The narrow implementation reuses the exact source-evidence derivation and validated admission linkage without adding policy or a competing representation of authority. The tests separate the actual new behavioral regression from controls that already worked and explicitly protect order-only mutation validity. This is an appropriate fix for I1.

Critical: none. Important: none. Minor: none. No new out-of-scope defect discovered during this fix review.

The only outside-diff source inspections were `compositionSourceEvidence` and the ordinary-message/runtime-lifecycle branches of `compositionObservationPredecessor`, to verify exact intent linkage, unrepresented membership behavior and the absent-target-decision case. No whole-file or repository-wide audit was repeated.

## Boundaries retained

Task3 ordinary collection/application acceptance and Task4 actual subprocess SIGKILL, faulty-monitor variants, whole-source race/vet/archive and final branch review remain future work. Public dynamic lifecycle grant authoring, live-provider quality, multi-host coordination, migration, and unbounded safety/liveness remain outside this review. The original Task2c report's strengths and exclusions stand; only its I1 blocker is closed here.
