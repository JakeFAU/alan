# Whole-branch weather experience review

Reviewed `42122cd69742d8a231d24cd497a95f6952163e27..b0e510efe55b0eebe97214d65f4db53dae886bd5` against `docs/evidence/weather-agent-experience/contract.md` and the implementation plan. Review date: 2026-11-14. Applied the requesting-code-review/code-reviewer method. Read the initial Task 1 review, its rereview, Task 2 review and current ledger. Reviewed the complete changed source and documentation plus relevant owning source, retained outputs and raw qualification logs. No subagents, new provider calls, suite reruns, source/index/HEAD/branch changes, publishing or merge.

## Strengths

- **The change stays in the example and presentation boundary.** The range changes 18 paths, all weather example code/tests or Markdown. Direct `git diff --exit-code` checks found no changes under runtime, internal, author, adapters, effects or schema, in module files, or in weather testdata. The authored programs, deployment recognizer and original archive remain unchanged. The only approval implementation refactor factors the existing basis/reason calculation into `weatherApprovalBasis`; actual authority still comes from the existing Store method (`examples/weather-agent/approval.go:11`, `main.go:199`). Wrapped approval failures preserve the underlying error and add predecessor/operation context.

- **Status has the correct read-only ownership boundary.** `main.go:83` enters the same validated `inspect` path; the execution switch has only replay/run/resume/refresh branches. Status neither opens an owner nor calls RunProgress or ApproveContinuation. `inspect` verifies the deployment and native weather evidence before summarization (`main.go:135`, `deployment.go:14`, `deployment.go:60`). Owning inspection uses a detached read-only transaction (`internal/executor/inspection_execution.go:226`). `status --json` selects the existing detailed summary/encoder, while bare inspect remains detailed JSON (`main.go:121`). Status's only additional work is classifying and formatting the validated cut.

- **Guidance respects causality and application policy.** Protected approval uses the same `ContinuationBasis` and `weatherReason` as refresh. The basis requires a paid original operation; weather policy waits for terminal local result processing when a result already exists (`internal/executor/continuation_history.go:30`, `examples/weather-agent/approval.go:22`). Unknown original operations are never offered a retry; a separate successor keeps distinct identity, charge and uncertainty. Provider-contact guidance aggregates all actionable reads, so a retained result does not imply all work in a resume is local (`status.go:29`, `:45`, `:130`). Invalid accepted legacy approval suppresses advancement to match owner preflight (`status.go:15`, `main.go:162`).

- **The Important legacy finding is resolved.** `status.go:108` checks the legacy input-domain transport identity, including a refused admission or one on another route, before recommending refresh. Valid approvals are recognized earlier. This corresponds to the existing domain-wide key and conflict semantics in `internal/executor/messages.go:365`, `:435`. `status_test.go:108` uses actual admitted/refused M4 histories, covers both routes, checks wrapped ErrConflict and preserved originals, and retains a valid-approval control. There is no remaining open Important finding from the task reviews.

- **Tests inspect retained behavior and independent process evidence.** The changed tests exercise actual Store histories, JSON equality, main/WAL preservation, corrupt-evidence rejection, legacy programs, explicit/duplicate/early/skipped approvals and shell argv preservation. The shell test includes apostrophes and substitution-looking path text (`status_test.go:164`). The ordinary-process recovery code kills after independently acknowledged acceptance, compares original effects after two fresh resumes, then checks a distinct explicitly approved successor (`process_test.go:359`). Status checks at the kill cut are repeated and compare durable bytes. Terminal local completion and retained-result cuts receive separate guidance checks (`process_test.go:247`, `continuation_predecessors_test.go:66`).

- **Documentation teaches the actual public application.** `docs/weather-agent.md` follows the real request, correlated await, validity checks, finite provisioning, Create/Open/RunProgress and typed inspection; it explicitly identifies excerpts rather than presenting a second runnable application. The example guide distinguishes command success from observation success and snapshot advice from approval. Performance history remains linked, including the P7 failed threshold, and the root introduction now leads with one reading. The two previously used section anchors are retained. No unrestricted recovery or performance claim was added.

## Qualification evidence checked

All paths below are relative to `.superpowers/sdd/plan-weather-agent-experience/` unless a repository path is explicit.

- Parsed the raw `qualification/{ordinary,integration,race-integration}.stdout` JSON streams and checked command/exit records: respectively 1,424 / 2,703 / 2,703 passing test events; 18 passing packages each; 4 / 6 / 6 test skips and three packages without tests each; no failed events. Skips match the four optional frozen-reader children plus the two documented integration-only historical checks. These were retained results, not newly executed suites.
- Checked exit records for build, vet, module verification, formatting and all three Staticcheck scopes. The first build's 177-byte stderr warning is preserved. Temporary-cache build/module/offline rechecks exit zero with empty stderr; the qualification does not conceal the earlier warning.
- Inspected `qualification/source.commit`, `source.tree`, `env.sh`, freeze and bridge outputs. Both tracked manifests hash to `3d7a7fb53d42f40bde4d37fa65beb7f4e570bcd56c5322078e7e0d9a30346476`; before/after archive-hash files and tracked manifests are equal. The qualified revision is `9c2b0f6b34c59f2ea57517c45f6f7036ff253b16`; delivery differs only in the root README, weather README and qualification record. A direct Git comparison confirms no Go changes after reviewed source `b0d52c8`.
- Independently byte-compared all 12 retained JSON outputs against their four baseline outputs: success 19,330 bytes, interrupted 13,377, unavailable 5,404, saved 8,529. Read `json-compatibility/compare.py` and `comparison.json`: each of the 16 candidate reader commands exited zero and preserved main/WAL bytes, with absent WAL treated as empty. The interrupted cut includes the unknown original and completed successor; this is a retained-cut comparison, not an exhaustive historical-state proof. SHM is explicitly excluded.
- Inspected `qualification/controlled-sigkill.stdout` and the corresponding process source. The named child records the first accepted GET with withheld response, SIGKILL, then a second distinct operation only after approval, ending with two contacts and two charges while the first remains `reserved_unknown`. The parent's separate successful reference call has its own fixture/store and is not silently counted as part of that child's two-contact history.
- Inspected retained offline status and replay output: original operation `["weather-1",1,1,0]`, 80.3 °F, 77%, provider time `2026-11-03T20:15:00Z`, one spent opportunity, unknown provider cost and 15 replayed reductions. Direct Git checks confirm the archive and manifest are unchanged from baseline.

## Issues

### Critical

None.

### Important

None.

### Minor

1. **Fresh zero-allowance status omits useful local-refusal guidance.** `examples/weather-agent/status.go:89–95` falls through to no action for the first open read with no effect and allowance zero; `application_test.go:182–183` deliberately asserts that resume is absent. The displayed state is `awaiting request` followed by `No remaining read slot.` Existing resume can nevertheless finish local refusal processing, establishing the original operation and selected `source_denied` result without spending an opportunity. The same classifier already recommends local resume after an effect is committed at allowance zero (`status.go:96–100`). This is a small inconsistency in the promised concrete next-action/local-work presentation. It does not permit a provider call, hide a possible successful reading, or break execution.

   A bounded experiment used the retained qualified binary `qualification/weather-agent` and a fresh temporary Store, endpoint `http://127.0.0.1:1/v1/forecast`, latitude 26.93, longitude -80.08, fahrenheit, allowance 0. Start, status, resume, status and inspect all exited zero. Initial status was `Read 1. awaiting request Response absent. Observation unselected. Opportunity unspent.` with no resume command. After resume, status was `Weather unavailable (source_denied)` with the original operation, selected observation, zero of zero spent and no remaining slot. Detailed evidence has `reservation_reason=effect_limit`, no dispatch admission, adapter-execution/contact bounds [0,0], and local status failed. Outputs and Store are retained in `/var/folders/qn/d3_l9fv5585g36qycsc804hh0000gn/T/weather-final-review-zero-8usjwr_p` (`0.stdout` through `4.stdout`). No live provider was used.

   Exact reproduction from the repository root (the experiment used a Python-created temporary directory and the same argv):

   ```sh
   agent="$PWD/.superpowers/sdd/plan-weather-agent-experience/qualification/weather-agent"
   work=$(mktemp -d)
   "$agent" start --db "$work/zero.sqlite" --lat 26.93 --lon -80.08 --units fahrenheit --endpoint http://127.0.0.1:1/v1/forecast --allowance 0
   "$agent" status --db "$work/zero.sqlite"
   "$agent" resume --db "$work/zero.sqlite"
   "$agent" status --db "$work/zero.sqlite"
   "$agent" inspect --db "$work/zero.sqlite"
   ```

   Recommendation: classify this first, open, no-effect zero-allowance case as local work, keeping the zero allowance and no-slot text and suppressing provider/refresh guidance. Update the existing focused assertion to distinguish initial local completion from the completed refusal. This is presentation of already supported behavior, not a request to change runtime/provisioning or grant more authority. It is optional polish, not a merge blocker.

2. **Clarify ExternalTimeout's per-dispatch scope.** `docs/weather-agent.md:78–79` groups all options as limiting this call's “waiting,” which can be read as a deadline on the whole RunProgress call. The runner passes the timeout to each external work item (`internal/executor/runner.go:203–228`), and `transportAdapter` constructs the timeout context for that dispatch (`internal/executor/adapters.go:295–315`). Distinguish per-invocation step/dispatch/concurrency limits from the timeout on each external call. This confirms the deferred Task 2 Minor finding; source behavior is correct.

3. **State the offline build prerequisite.** `examples/weather-agent/README.md:109–119` labels the walkthrough offline but starts with `go build`, which can need a download on a cold module/toolchain cache. The qualification explicitly used local dependencies (`docs/evidence/weather-agent-experience/qualification.md:24–28`). State that offline readers need a prebuilt executable or the required Go toolchain and dependencies already cached. The read commands themselves are offline; this confirms the other deferred Task 2 Minor finding.

## Recommendations

Address the three small presentation findings together if taking a final correction wave. The two Markdown fixes need source/whitespace checks; a zero-allowance classifier change needs a focused behavioral check that it offers only local completion, preserves zero spending and reader bytes, and removes resume after terminal completion. There is no evidence here requiring another broad suite campaign or protocol changes. If code changes, record a new source bridge rather than describing the prior full-suite results as having run at the new commit.

## Declined to judge

- Live Open-Meteo availability, production terms, rate limits or current weather: no live calls were authorized for qualification; this change does not alter the adapter or provider contract, and the archive is explicitly historical.
- General distributed ownership, arbitrary adapter behavior, cross-host durability, or universal recovery correctness: existing runtime/trust assumptions, outside this presentation slice; this review only checks that the new guidance does not expand those claims.
- Nested historical evidence modules and skipped optional historical campaigns: the recorded full-suite commands cover the root module and disclose these limits; no new requirement to rerun the separate historical producers is established by this change.
- Eventual applicability of copied commands after another writer changes the Store: status explicitly describes one snapshot, and actual commands revalidate under ownership. Suggestions are not grants or promises about a later cut.

## Assessment

**Ready to merge? Yes, with optional Minor polish.** No Critical or Important findings remain at `b0e510e`; the original Important legacy-ID issue is fixed. The implementation preserves authority, original uncertainty, detailed JSON and historical evidence while improving the ordinary operator and developer path. Qualification supports the stated source-bound, single-host claims; it is not an exhaustive guarantee. This is approval of the reviewed branch for the requested draft-PR delivery, not authorization to merge or release.
