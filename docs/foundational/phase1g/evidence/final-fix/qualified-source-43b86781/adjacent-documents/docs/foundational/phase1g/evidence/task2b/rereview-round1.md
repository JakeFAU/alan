# Independent Task2b scoped rereview — fix round 1

**Finding I1: ADDRESSED. Spec verdict: Approved for this bounded fix. Quality verdict: Approved.** No new Critical, Important or Minor issue was found in the fix diff. This is a rereview of retirement closure and R7, not a fresh full branch review, public activation approval, or Phase1G qualification.

Reviewed range: `a648ea739d21f04e011be3214f7819130a5f3f2a` to `4b350ec815ec057bbf5de93465285e7843629d13`, including code fix `16ffae2` and controller contract/review documentation. Read the original Task2b brief, bounded fix brief, exact source/contract diff companion, all new retirement test source, fix report and relevant retained command/evidence files. Used the existing review and Go skill guidance. No source/index/HEAD mutation, git operation, subagent, credential access or live call occurred. Only this rereview document was added.

## I1 disposition

The original Important finding was: **Retiring an already-succeeded member fails to close its still-active scope.** The unchanged original probe demonstrated accepted retirement, an unchanged local succeeded terminal, an open peer and subsequent admission of two child members.

The fix addresses the actual missing lifecycle-to-scope transition:

- `internal/executor/composition_facts.go:208` passes the accepted lifecycle result and its ordered fact into the closure handler. `internal/executor/invocation_closure.go:66` distinguishes a newly retired subject from an accepted already-terminal cancellation/abandonment no-op. It preserves ordinary local terminal state and existing source authorization.
- `internal/executor/invocation_closure.go:82` retains the retirement cause separately, in the same owner transaction as retirement, result and lifecycle fact. Its descriptor identifies the root, original lifecycle operation, target run/sequence, exact fact ID/order and frozen committed intent sets. The table's decision-order FK refers to that existing fact; no second ordering transaction or scheduler is introduced.
- `internal/executor/invocation.go:174` and `:195` fence admission at the static root and every invocation ancestor using retained closure evidence. Already-published invocation results also remain permanent fences. The supported ancestry test confirms that a static required-member retirement blocks an already-admitted descendant's pending Invoke without cancelling the descendant's open run or rewriting earlier allowed authorization.
- `internal/executor/invocation_outcome.go:204` prevents later draining terminals from replacing retirement closure. Preparation at `:297` selects applicable retirement evidence over an unpublished success freeze, preserves first non-success precedence and reports `cancelled/subject_retired`. Previous successful freeze bytes are retained unchanged. Existing publication remains immutable.
- `internal/executor/invocation_outcome.go:363` preserves R7 retirement precedence for conflicting frozen replies, returns no selected public response and keeps the original reply identities/candidates. The normal non-retirement conflict path remains failed/conflicting_replies.
- `internal/executor/composition_store.go:279` closes the discovered retirement-only reopen gap. A nonempty closure table is rejected before unfinished protocol replay can be claimed.

The controller's R7 resolves the earlier ambiguity explicitly: actual retirement before publication yields cancelled/subject_retired; local success is not rewritten; earlier non-success or publication wins; successful freeze alone is not publication; already_terminal CancelRun/AbandonRun makes no new closure. The implementation and covering tests match that ruling. No further policy ambiguity blocks this finding's disposition.

## Test and evidence assessment

The retained tests exercise actual owner transactions, observations, admission decisions and settlement. The strongest supported paths are the unchanged original static-member probe and `TestRetirementStaticAncestorClosesAdmittedDescendants` (`invocation_retirement_test.go:300`). The latter uses provisioned exact static authority, checks no new members/charges, preserves original source authorization and leaves admitted peers open.

Direct dynamic-target tests use a clearly documented test-only exact permission seed after admission. Their coverage is conditional internal lifecycle/closure coverage, not proof that the current public immutable provisioning API can authorize lifecycle requests against future namespace-derived subjects. They do not replace the supported static/ancestry cases. No production delegation API was added.

Covering tests also check:

- retirement after success freeze but before actual outcome publication, including late authenticated reply admission and unchanged historical freeze bytes (`:143`);
- retirement after published success, with stable result identity and outcome (`:176`);
- CancelRun and AbandonRun already-terminal no-op controls (`:206`);
- retirement-first versus failure-first ordering, including original diagnostic response retention and exclusion of later replies from the closure prefix (`:226`, `:248`);
- duplicate lifecycle delivery and repeated retirement, exact cause/order/pending-intent linkage (`:84`);
- injected closure insertion failure rolling back retirement, lifecycle result and ordered fact together (`:268`);
- a retirement-only private store failing closed on reopen (`:347`);
- conflicting reply candidate retention without replacing cancelled/subject_retired (`:377`).

Read the numeric subprocess results in `review-fix1/commands.json` and corresponding original output. The original probe and supported ancestor probe failed before repair (commands 01 and 03, exit 1). Command 02 includes both behavioral failures and missing-table setup failures; those setup failures are not treated as behavioral proof. Commands 06 and 07 retain the retirement-only reopen and retirement/conflict precedence failures. Command 08 passes the retirement tests plus existing reply/conflict/freeze/closure controls, exit 0. Command 09 runs the executor package once and passes, exit 0. Command 10 reruns the unchanged original probe on final source, exit 0:

```text
retirement accepted=true reason=retired retired=true member_status=succeeded peer_status=open
scope_closed=true admission_accepted=false admission_reason=scope_closed admitted_members=0
```

No suites or probes were rerun during this rereview: source review and retained exact evidence resolved the scoped question without a new unanswered doubt. Historical RED reproduction and current-source GREEN regression remain separate claims.

## Independent preservation and identity checks

A read-only Python byte/hash check compared all seven copied original evidence files against the original reviewer files under `/tmp/alan-task2b-review-retire`: probe, complete overlaid test source, overlay mapping, input manifest, command, output and original review. Every comparison was equal. The original saved review also still equals `evidence/task2b/review.md`.

Key original identities remain:

| Evidence | SHA256 |
| --- | --- |
| Supplemental original probe | `8205b0a9fbeb3b11bc20ba50a5c7c250f588222af8580d06f9954c2d9f947190` |
| Complete original overlay test | `f41fa6001f4228e54ba8ab20aa6ee4e55981c10d54b9a28907d160e2bbd0b461` |
| Original failure output | `7b780df60cd1cc22770e6b4799f8eaf86501b76618ad017eb90a121e3dfb87f4` |
| Original review | `9eae098875dfe542c251fded40eddf5ce648f5410d423bb6259ac5202395b677` |

Checked current bytes of all six changed Go files plus unchanged `invocation_test.go` against the final executor-suite manifest: all seven match. The retained final retirement-test and closure-source snapshots also match current files byte-for-byte. Final reported source identity is `d525dec36ec4542882a8ec40608976b70c323030b4a9e56e3736eab2f5eecbe1`; this rereview verified the named source subset, not every unrelated file in that manifest. The capture wrapper retains exact argv, constrained environment and numeric Go subprocess status. The original evidence was preserved rather than regenerated or relabeled.

## New fix-diff issues

Critical: none. Important: none. Minor: none.

The separate closure record is justified by the requirement to preserve a prior local success and successful freeze while recording later scope closure. Shared prefix capture avoids two independently evolving reply/Invoke/external scans. Errors propagate through the existing transaction boundary. The private protocol layout change and lack of compatibility/migration for unfinished private stores are disclosed; public schema4 activation remains gated.

## Boundaries and out-of-scope observations

- Full forward/reverse closure validation, replay of ordered prefix facts, and root publication remain Task2c. The live reader's limited canonical/relational checks are not presented as those completed validators. The fix report lists the exact missing checks, and unfinished histories remain gated.
- External attempt reservation/debit enforcement, root/global physical capacity and runner integration remain Task2c. This rereview verifies the shared scope helper's invocation admission use, not future external enforcement.
- Direct dynamic lifecycle authority remains unavailable through supported immutable public provisioning. Seeded-grant tests establish only the conditional internal behavior described above.
- Earlier Task2a M1 remains a recorded coverage gap; this fix does not resolve it.
- No broad compatibility audit, full manifest audit, subprocess crash test, physical saturation test or final branch review was performed. No source outside the supplied fix source/contract diff was newly inspected for behavior; prior review context was reused, and evidence/snapshot reads were limited to verifying this repair.

**Assessment:** I1 is addressed with retained regression evidence and the R7 policy implemented. The bounded fix is approved; later Task2c and full branch review remain required.
