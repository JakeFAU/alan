# P2 disposable diagnostic overlays

Tooling only; no diagnostic has been accepted by the presence of these sources.
The frozen experiment is [../contract.md](../contract.md). These files derive
from the immutable series-scaling tooling and implement the reviewed exact
semantic-context addendum. They do not select a production optimization.

`generate.py` copies ALAN, the pinned SQLite v1.14.52 module and the **whole
multi-file P2 driver** before adding counters. The only `.go` additions are in
those copies. Input trees and module caches are never edited. Existing output
paths, outputs inside inputs, missing/duplicate text or signature anchors fail
closed. An anchor failure leaves a diagnostic scratch directory and no accepted
binary. Root must review its generated patch before interpreting measurements.

## Exact scope definition

At each function entry below the copied function replaces its local `ctx` with
`measurecounter.Scope(ctx,id)` before any closure capture. Every downstream SQL
invocation that receives it has exactly that scalar semantic operation. The
nearest tagged owner wins. Shared `prepareRoot`, `readRootCompletions`,
`compositionMembershipRoot` and validators stay untagged: their SQL inherits the
owner, preserving report versus settlement/create/open/replay distinctions.
This is exact declared semantic ownership, not arbitrary Go caller tracing.

| ID | Semantic scope | Source owning entry in `internal/executor/` |
| --- | --- | --- |
| 0 | unknown | Missing context value; legacy/contextless calls and transaction Commit/Rollback |
| 1 | invalid | ID outside this allowlist |
| 2 | createStore | `store.go:createStore` |
| 3 | openStore | `store.go:openStore` |
| 4 | Replay | `replay.go:Replay` |
| 5 | Inspect | `inspect.go:Inspect` |
| 6 | InspectExecution | `inspection_execution.go:InspectExecution` |
| 7 | Run | `runner.go:Store.Run` |
| 8 | finishRun | `runner.go:Store.finishRun` |
| 9 | selectReady | `runner.go:Store.selectReady` |
| 10 | settleRoots | `composition_root.go:Store.settleRoots` |
| 11 | Commit | `store.go:Store.Commit` |
| 12 | AuthorizeIntent | `messages.go:Store.AuthorizeIntent` |
| 13 | ApproveContinuation | `continuation.go:Store.ApproveContinuation` |
| 14 | PrepareRoot | `composition_root.go:Store.PrepareRoot` |
| 15 | SettleRoot | `composition_root.go:Store.SettleRoot` |
| 16 | prepareAdapterWork | `adapters.go:Store.prepareAdapterWork` |
| 17 | accountAdapter | `adapters.go:Store.accountAdapter` |
| 18 | recordAdapterResult | `adapters.go:Store.recordAdapterResult` |
| 19 | settleAdapterLocked | `adapters.go:Store.settleAdapterLocked` |
| 20 | recordDispatchAdmission | `dispatch_admission.go:Store.recordDispatchAdmission` |
| 21 | AdmitInput | `messages.go:Store.AdmitInput` |
| 22 | AdmitInvocation | `invocation.go:Store.AdmitInvocation` |
| 23 | prepareExternal | `external.go:Store.prepareExternal` |
| 24 | recordExternalReceipt | `external.go:Store.recordExternalReceipt` |
| 25 | settleExternalLocked | `external.go:Store.settleExternalLocked` |

There are 24 named owners plus unknown/invalid. Context WithCancel, WithTimeout
and WithoutCancel preserve the scalar. A new Background/TODO loses it. There is
no mutable global current operation, scope stack, statement-cached context,
connection-last-use guess, stack sample, raw ID, payload or per-event log.
SQLiteRows already retained its cancellation context before this overlay; the
only new Rows field is the immutable uint8 scalar. Stmt retains only its bounded
normalized family. Statement query/exec extracts the current invocation scalar
before binding, eager stepping or an Exec worker. Rows keeps it through deferred
Next, cancellation, eager pending-row consumption and close. No-args exec passes
it explicitly through each combined prepare/step/finalize call.

## Counter definitions and bounds

Snapshots are cumulative map[string]uint64 values; no map key disappears and no
value decreases. The hook contains no active-connection gauge or mutable
completeness flag. Engine status counters reset only immediately after a real C
step, including failures. Snapshot reads never reset engine/counter state.

`sql.template.<normalized-family>.<metric>` and `sql.total.<metric>` preserve
family totals and the previous metric meanings:

- `completed_prepares`: explicit successful prepares of real statements (cache
  reuse is not another prepare; automatic reprepare is unavailable).
- `invocation_attempts`: statement query/exec entry before bind, or a real
  prepared statement in the combined no-args path. Failed prepares and empty
  submissions have separate driver counts and no invented invocation/run.
- `step_calls`: actual C steps, including eager steps/errors, excluding later
  consumption of a buffered pending row.
- `executions`: SQLite SQLITE_STMTSTATUS_RUN deltas. These are VM starts,
  **including executions that fail**, not successful API outcomes.
- `sqlite_row_returns`: SQLITE_ROW returns, not exact rows examined.
- `fullscan_steps`, `vm_steps`: SQLite status deltas after actual steps.
- `step_errors`, `bind_errors`: unsuccessful step/bind counts. Bind errors
  contribute attempted invocation but no C step or VM execution.

The scoped partition is
`sql.scope.<name>.class.<class>.post_autocommit.<state>.template.<family>.<metric>`.
`class` is fixed configuration (`read_write_capable`, `read_only_configured`,
unknown), derived from the driver's parsed file-URI `mode=ro` or `_query_only=1` settings.
It does not establish ownership, write activity or transaction read-only
enforcement. `post_autocommit` is `off`/`on` **after each step**, before
reset/finalize; prepare/attempt/bind counters use `unknown` because no step state
was observed. Successful BEGIN normally observes off; successful COMMIT/ROLLBACK
normally observes on. Failed COMMIT may observe off before its real cleanup
ROLLBACK. No before/after transitions, transaction IDs, snapshot IDs or durability
claims are inferred. API Commit/Rollback use Background in the original driver;
their SQL remains scope unknown, including cleanup after failed Commit.

The family dictionary admits 1024 families, each at most 4096 bytes, plus one
reserved overflow family. Statements also retain the bounded registered family.
There are at most 16384 sparse `(family,scope,class,post-state)` cells, plus one
separate overflow metric vector. Scope/class/post-state enums are fixed. Metrics
use fixed-size structs; driver keys come only from a finite set of source literal
keys. No connection map, origin map or unbounded event stream exists. Family and
scope counts are different dimensions: **never sum them into executions**.
For each metric, all scoped cells plus `sql.scope_overflow.<metric>` must exactly
equal `sql.total.<metric>`, independently of any family/cell cap truncation.

`sql.attribution.family_overflow_events`, `cell_overflow_events`,
`invalid_scope_events`, `incomplete_events` and family/cell counts are cumulative.
Overflow event counts are probe admissions/updates, not distinct omitted SQL
families or invocations. Any family/cell/invalid event defeats a complete top-ten
attribution gate. `attribution.py --snapshot CUMULATIVE_SNAPSHOT.json` derives
`attribution_complete=false` outside the monotone hook if any incomplete event
exists. Even when true, `known_semantic_scope_attribution_complete` is false if
unknown step calls exist. Unknown control/setup attribution remains a disclosed
limitation; completeness means no truncation, not that every SQL caller is known.
No bound is silently enlarged after a run.

`sql.driver.*` counts the following API boundaries and their actual outcomes:
`open`, `close`, `begin` (the common begin path), `begin_tx`, `commit`, `rollback`
have calls/success/errors. Legacy Begin has `begin.legacy_calls`; BeginTx also
records requested read_only/read_write. This driver discards TxOptions; a
read-only request is not SQLite enforcement. Cleanup ROLLBACK after failed
Commit is an actual SQL step, not a call to SQLiteTx.Rollback. Repeated successful
Close increments API success; only an actual successful sqlite3_close_v2 on a
live connection increments `physical_close.success`. `connection_constructed`
includes constructed connections that subsequently fail initialization.
Open attempts that fail before construction have no fabricated connection use.
`connection_setup.calls/success/errors` counts direct setup sqlite3_exec API
calls, potentially containing several statements. Their prepare/run/VM/row work
is unavailable and excluded from these metric families. No per-handle raw-open
or per-connection distribution is claimed. `conn_exec`, `conn_query`,
`stmt_exec`, `stmt_query` expose API calls and cancellable contexts as separate
levels; do not add the two levels together. Prepare errors/no-statement events
are separate (`prepare.*`, `noargs.*`). Error and unknown-state metrics never
imply successful SQL execution.

Function counters remain independent: `fn.*.calls` counts entry including errors;
existing descriptor input/initial-marshal bytes remain `fn.*` byte metrics;
`work.*` contains reducer call categories, inspection materialized rows/bytes,
readiness loops, and root × intent loops. `work.finishRun.root_intent_pairs`
counts only those source loop iterations under finishRun's inherited scalar;
`work.readRootCompletions.root_intent_pairs` counts all calls. Root must compare
the finishRun-scoped membership-family actual execution count to the former and
to successful fixed-B B×k before claiming reconciliation. Counts alone do not
establish SQL CPU or latency cost.

## Separate coarse timing and profiles

`timing.py` creates a distinct disposable ALAN/driver copy using the original
SQLite module, with no SQL overlay. Its fixed boundaries cover Run/report,
root settlement/readiness, root completion reads, ledger/composition validation,
InspectExecution, authority/adapter preparation/transport/accounting/retention/
settlement, plus Commit, ApproveContinuation, open/create and dispatch retention.
Timers run once per coarse boundary including error returns, never per SQL or
row. Elapsed wall spans are inclusive and overlap; they are not additive or CPU
samples. CPU/alloc profiles and traces use the ordinary uninstrumented binary.
Neither overlay contributes to headline medians or receives overhead subtraction.

## Root-only execution commands

These are proposed commands; running them is the root agent's serialized work.
Use frozen exported ALAN source, the original pinned SQLite module, corrected P2
multi-file driver and new disposable output paths. None of the old evidence is
rewritten. Run generator-anchor tests before generation; preserve failures.

```sh
P2_SQLITE_SOURCE="$SQLITE" python3 -m unittest discover \
  -s docs/evidence/performance-p2/instrument -p 'test_*.py' -v
python3 docs/evidence/performance-p2/instrument/generate.py \
  --alan "$ALAN" --sqlite "$SQLITE" --driver "$DRIVER" --out "$COUNTERS"
# Review COUNTERS/measurement.patch before accepting diagnostics.
python3 docs/evidence/performance-p2/instrument/verify.py \
  --sqlite "$COUNTERS/sqlite" --out "$PROOFS"
# verify.py serializes focused ordinary/race proofs and the normalizer mutant.
# For the final build use a separate new output; build.sh generates then builds.
docs/evidence/performance-p2/instrument/build.sh \
  "$ALAN" "$SQLITE" "$DRIVER" "$BUILT_COUNTERS"
python3 docs/evidence/performance-p2/instrument/timing.py \
  --alan "$ALAN" --sqlite "$SQLITE" --driver "$DRIVER" --out "$COARSE" --build
```

Proof payload includes the prior lexical normalizer and exact step accounting;
prepared/cached reuse under distinct scopes; deferred/eager/pending/partial/early-close/
empty/error rows; cancellable worker Exec and cached/deferred pre-step query cancellation;
bind/prepare failures; multi-statement no-args; nested/sibling/WithoutCancel;
legacy unknown; two concurrent connections; read-only class and refused write;
success/failed deferred-constraint Commit and real cleanup rollback; repeated
Close/Open errors; tiny family/cell/length caps; exact partition reconciliation;
and non-resetting snapshots. Tests are measurement soundness proofs, not generic
tracing infrastructure. Scope counts still need the real public-driver B8 smoke
and same-state/schema/root×intent reconciliation after source review.

Original, generated pre-gofmt and post-gofmt manifests bind every input/copy,
including all P2 driver files. Generation records source/tool hashes, scopes and
exact transformation patches. Build records binary/modfile SHA-256, toolchain,
platform and embedded Go build metadata. Coarse generation records separate
identities. Root must retain proof/workload output, exact commands/environment
and ordinary/instrumented state comparison; none is asserted here as passed.

Unavailable: exact indexed rows examined, trigger-internal statement identities,
automatic reprepare counts, setup direct-exec VM work, arbitrary Go caller
identity, per-connection distribution, per-category allocations/CPU time,
transaction identity/read-only enforcement and authoritative workload results
before root execution. Go allocation profiles also exclude C allocations.
