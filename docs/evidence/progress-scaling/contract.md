# P6: execution progress without implicit full reports

Frozen before production changes against merged PR #16, main
`e43e4f6be4500d177b50caf29181608d09b99c62`. The user authorized the next bounded
experiment: separate execution progress from repeated whole-history reporting,
investigate safe reuse, and preserve existing `Run`. API details below are the
implementation decision. Delivery is a draft PR; no merge or release.

## Question and decision

Can an application advance an owned computation without requesting a full
historical report after every advancement, retaining all execution authority,
accounting and recovery checks?

Add opt-in `Store.RunProgress(context.Context, RunOptions) (RunProgress, error)`.
`runtime.RunProgress` exposes only `Steps`, `Dispatches` and `Quiescent`. Both
methods use one private engine and identical mandatory cleanup. Existing Run
calls unchanged `finishRun`; progress omits only that final report/diagnostic
traversal. No validation result is cached or claimed to have been reused.

Create and fresh Open continue validating the retained ledger. Subsequent
execution transactions check the authority and evidence they use. Progress
does **not** certify the entire retained history. Callers request independent
InspectExecution or Replay for that purpose. Full Run retains its existing
report diagnostics, but is itself not a complete Replay.

### Alternatives examined

1. Streaming validators while dropping output arrays could save allocations,
   but retains repeated global work and needs substantial traversal refactoring.
   Its benefit is unmeasured; it is not ruled out.
2. Cross-call validation reuse needs a complete mutation/invalidation proof.
   Existing staticExecution is discovery state, not a validation certificate.
   A corrupt private write followed by a legitimate write falsifies a simple
   post-commit watermark; authority, results and root readiness invalidate
   different facts. Do not introduce this protocol in this slice.
3. Explicit progress has the smallest implementation and an honest narrower
   diagnostic contract. It separates requested output from execution without
   inventing a reusable validity certificate.

P5's corrupt historical admission witness pins the distinction: full Run keeps
its EOF diagnostic; progress need not revisit an unrelated admission. That
witness does not demonstrate unauthorized dispatch or justify dropping a check
reached by execution. Preserve the original test unchanged.

## Invariants and exact boundary

- Preserve defaults/errors, cross-method single-runner exclusion and Close
  coordination, scheduling, limits and all execution transactions.
- Preserve source/generation/revision checks, protected approval selection,
  independent reservation/debits, final dispatch admission, original IDs,
  results/conflicts, settlement, publication and reconciliation.
- Cancel/join workers, drain completions and account results under the existing
  detached bounded cleanup context before return. Preserve joined errors and
  sticky accounting failures.
- Set progress quiescence after accounting using the existing engine predicate.
  Quiescent means no supported eligible work at the observed boundary; it does
  not mean completed, remotely final, forever idle or a consistent audit cut.
- Steps/Dispatches keep their existing per-call meaning. Dispatches is a runner
  count, not paid opportunities or proven provider contacts.
- Preserve every full Run field/order, nil/empty distinction, partial result
  and error, including quiescence staying true after a report-only error.
- No Store-lifetime summary/cache, schema/table/migration/dependency, scheduler,
  ownership, automatic retry or BoundedSeries API change.
- Open, Replay, inspection and historical format acceptance stay unchanged.
  No progress interpretation grants authority or changes activation epochs.

The Store lock excludes cooperating owners on one host, not arbitrary SQL
writers. Unsupported private SQL, hostile concurrent writers and physical
corruption are not newly protected by progress. Tests must expose the narrower
diagnostic scope. Authority checks reached by execution still reject malformed
or mismatched evidence; no equivalence claim for all corrupt-store errors.

## Application and qualification

Weather already obtains complete typed snapshots before/after execution and
discards RunReport. Change only that execution call; keep ingress/approval and
snapshot-derived uncertainty. Benchmark progress per observation, with a full
report, inspection, reopen and Replay separately timed afterward. Include total
lifecycle cost as well as execution cost so shifted work stays visible.

Compare deterministic same-seed full/progress histories: reached errors,
counters, IDs, grants/approvals, selected observations, charges, contacts,
results, uncertainty, root outcomes and independent Replay. Include success,
refusal, unresolved response followed by separately approved work, duplicates,
limits/cancellation, active-run exclusion, dynamic and static/fallback
discovery, and representative historical formats.

Keep all old report/corruption tests unchanged. Add the explicit unrelated
corrupt admission distinction plus subsequent full Run/audit/Open rejection.
Add reached malformed result/attempt, cancellation-at-accounting, error-join,
Close and cross-method concurrency tests. A changed diagnostic contract must
not conceal a removed execution gate.

Use acknowledged real weather SIGKILL/reap at approval, reservation/admission,
lost response, retained result/settlement, selection and completion cuts.
Fresh Open/repeated recovery preserve original evidence and prevent additional
unauthorized contact. Existing owner/readers, receipt, admission and causal
checks remain required. No exactly-once external-effect claim.

## Measurement and stopping rules

Record baseline/tree, clean state, dependencies/toolchain/environment, ordinary
suite and fresh baseline profile before production edits. Use one public
deterministic harness for baseline full, candidate full control and candidate
progress. Bounds 1,8,16,32,64,128,256; one excluded warmup and six measured
samples per arm/bound, balanced adjacent ordering in separate serial processes.
No concurrent builds/tests during primary timing. Remove old per-prefix report
proof equally in all arms, retain equal cheap counts/contact checks, and verify
every result/charge/approval outside execution. Do not compare changed harnesses
as a speedup. Separately retain CPU/allocation profiles and SQL/report counts.
Measure construction/Create, targeted approval, execution, final report/audit,
reopen/Replay, elapsed/CPU, bytes/allocations and actual process peak RSS.

Freeze executable source before final ordinary/integration/race-integration,
focused race, real-process, historical-reader, build, vet, formatting, module
and Staticcheck checks. Record commands/exits/skips/hashes; timeouts are not
passes. Independent review covers semantics/API, recovery and methodology.
Preserve rejected findings. Evidence-only delivery needs a source-content
bridge. Preserve raw artifacts locally and commit compact reproducible records;
do not imply local archives are public downloads.

Stop or correct narrowly if execution changes, a reached gate is lost, full
Run differs, recovery invents authority/contact, or benefit is within noise.
Do not add validation caches to rescue a negative result. Report remaining
global work without universal asymptotic or latency claims.
