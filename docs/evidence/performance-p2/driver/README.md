# P2 public workload and campaign tools

This independent Go module copies the corrected P1 public-package workload and
uses the same `author.BoundedSeries` definition, station `north`, provisioned
first approval, fresh public `ContinuationBasis`/protected approval before each
successor, one ordinary Run per position and one in-flight dispatch. Bounds are
1, 8, 16, 32, 64, 128 and 256. Report version is `alan/performance-p2/v1`;
confirmatory schedule contract is `alan/performance-p2/paired-v1`. Earlier P1
files are unchanged. This is diagnostic tooling, not a production optimization.

Build with `driver/build.sh ALAN_SOURCE OUTPUT_BINARY`. Its temporary modfile
adds the sole local replacement; checked-in dependencies remain the original
ALAN module and pinned go-sqlite3 v1.14.52. Use exact source export/build/binary
hashes and the frozen campaign environment. No source label verifies itself.
A fresh store is required and failures emit partial reports without retries.
This README describes intended measurements; no measurements are supplied here.

## Frozen HTTP transport

The default `-mode=http` starts one same-process server outside Execute. It binds
numeric loopback only; default `-endpoint=127.0.0.1:0` chooses an ephemeral port.
The bound listener is synchronously ready before Execute, without a warmup
request. An exact address such as `-endpoint=127.0.0.1:43127` permits genuine
same-seed differential comparisons with the identical retained endpoint. The
retained URL is `http://<bound address>/measure`; never normalize ports/digests
away. The explicit address is the listener address, not an arbitrary URL.
Shutdown and connection cleanup are outside Execute. The deferred failure-report
boundary drains handlers/connections before copying either final journal, including
after forced connection close, and joins shutdown errors with the original error. The first connection's
setup, handler work, HTTP goroutines, allocations, CPU and GC occur in the
measured process. Server evidence is independently populated; resource accounting
is process-wide and includes the server.

One HTTP/1.1 POST per Execute sends compact Go `encoding/json` of the entire
original `effects.Prepared`, including authentic Request ID/key, capability,
effect, input/output schemas, payload/assets, protocol, endpoint, stable action
Wire and frozen digest. The content type is `application/json`. No optional
secret settings or provider calls exist. The server requires canonical envelope
bytes, valid Prepared digest and exact incoming Host/path matching Prepared
endpoint. It derives its accepted-contact journal from that incoming body and
records the actual deterministic response bytes and complete write outcome.
It never reads the client Prepared object or the adapter journal.

The stable action Wire is compact JSON with ordered keys `request`, `protocol`,
`endpoint`. Its request is the same original Request with only ID/key blank.
The complete original ID/key remain in Prepared.Request and the transmitted
JSON envelope, covered by its digest. This envelope/action distinction is
necessary: public same-action continuation scope blanks Request ID/key but
retains Wire byte-for-byte, so embedding the operation ID in Wire would reject
every successor. Wire has no self-referential digest. Its request is freshly
copied when constructing the stable action bytes.

The response is compact JSON with ordered keys `operation_id`, `key`,
`response_id` (`local-response:<original ID>`) and `sample` (`present`), with no
trailing newline. Both journals retain exact envelope/response bytes and SHA256.
The retained Candidate has original ID/key, validated disposition, sample
`present`, HTTPStatus 200, RequestID equal to operation ID, ResponseID as above,
and Response equal to these exact bytes. The response cap is 4096 bytes; the
request cap is 65536 bytes. The server has an 8192-byte header limit. Unsupported
methods, paths, content types, bounds or provenance reject the request.

Transport configuration is fully reported: no proxy, TLS or HTTP/2; client
request timeout, dial timeout, TCP keepalive interval, response-header timeout,
server read-header/read/write timeouts are 30 seconds; idle connection timeout
and server idle timeout are 90 seconds; Expect-Continue timeout is one second.
Connections are reused, with one maximum connection/idle connection per host and
one global idle connection. Redirects are disabled, no Idempotency-Key is sent,
and the nonempty POST uses a non-rewindable body with nil GetBody. Thus Go's
transport cannot replay it following a failed reused connection. Provider
retries are zero. The whole sample deadline defaults to 30 minutes and is
separate from the finite transport timeout. No retry loop replaces a failure.

`-mode=direct` preserves the P1 Prepared protocol `measurement/v1`, endpoint
`deterministic-local-no-http`, station-text Wire, 256-byte response cap and exact
historical Candidate identities/payload/policies. It is a separately labeled
historical bridge, never a headline HTTP sample. It preserves the effect
protocol while the new original-state reader/resource probes/returned-cut check
can change observer/cache/GC state relative to archived P1; it is not a
byte-identical historical timing measurement. `-prefix=false` may be used
explicitly for that bridge and for the predeclared overhead calibration.

## Measurements and prefix boundaries

All original stages remain. `original_state_verification` separately inspects
the freshly provisioned roots/runs before any contact.
`final_series_proof` separately measures the retained identity proof.
`execute_window` encloses all Runs, successor approvals, stage snapshots/progress
logs, contact checks and prefix bookkeeping. HTTP startup/shutdown, construction,
Create, original-state reader, final readers/proof, completed Close/Open/resume
and Replay are outside this ordinary Execute window. An interval-reopen schedule
has its own `execution_window_including_reopen` name and never enters strict
ordinary timing.

The per-stage resource probes record process user/system CPU, Go TotalAlloc,
Mallocs, NumGC, NumForcedGC, PauseTotalNs and fixed available runtime metrics:
GC assist/dedicated/idle mark CPU, GC pause/total CPU, automatic/forced/total GC
cycles and goroutines-created. Supported names are discovered once from
`runtime/metrics.All`, sorted and reported. The frozen Go1.27.3 confirmatory
validator requires the full reviewed vocabulary; a missing metric fails the
configuration gate rather than silently changing available evidence. No GC is
forced. CPU/GC classes overlap, and asynchronous work can cross boundaries.
GCCPUFraction is not differenced. Live goroutine count is not goroutines-created.
SQLite/C allocations are not Go TotalAlloc.

Prefixes are on by default and mandatory in all 98 confirmatory processes.
Preallocated records describe T(B,k) with B provisioned roots and k selected
observations. For k>1, its direct interval includes that position's fresh
approval snapshot/basis and approval establishment, followed by Run and contact
count checks. An initial snapshot starts the contiguous interval; every later
snapshot ends the preceding interval and starts the next. A final snapshot is
not an extra Inspect/Replay. Prefix snapshots read process resources only.
Every iteration, with prefixes on or off, validates its already returned
RunReport against the initial inspected B-root identity registry. It checks
exact generation-one Run/root membership; prior roots remain succeeded with
unchanged original outcome/charges; the new root succeeds with one effect
debit and its original root outcome; future roots remain open/unstarted with
zero charges and pending members. OpenRuns must equal those future roots.
For this frozen program, the first Run makes three reductions, successors four,
and each makes one dispatch; quiescence, zero rejected admissions and empty
unresolved/pending-delivery/unknown-effect lists are required. These restrictions
follow the known request/await/success declaration, not generic ALAN semantics.
A succeeded original root of this program implies its sole awaited result was
selected; quiescence or contact count alone would not establish that fact.
The final independent retained observation/reduction proof still runs.
The checks use returned data only, with no new readers/SQL, and are included
inside the prefix/Execute window. Newly established original root evidence is
copied once per position and compared on later cuts. Root membership is emitted
once, and each prefix retains a membership SHA256, derived completed/selected/
future/prior counts, work/uncertainty counts and its new root's original
ID/status/outcome/charges; final evidence must match these original outcomes.
This serializes O(B) proof evidence overall instead of B full B-root reports.

Each record contains its index, declaration bound, derived retained positions, actual
adapter/server contact counts, direct deltas and cumulative deltas. There is
no per-prefix JSON serialization or new prefix progress output inside Execute.

Snapshots are not simultaneous physical boundaries: MemStats, rusage, runtime
metrics and optional diagnostic counters are read sequentially. Timestamp is
read after the resource probes. The snapshot's own overhead can straddle its
reported memory/CPU/time boundaries. Bookkeeping after a prefix is included in
the next interval; the final bookkeeping plus initial/final boundary overhead
is explicitly `prefix_residual = execute_window - final cumulative prefix`.
Arithmetic rejects reset/underflow, reordered/missing/duplicate prefixes,
negative durations and inconsistent direct/cumulative/residual/count values.
Floating GC metric arithmetic uses 1e-12 absolute/relative roundoff tolerance.
The reported fixed prefix configuration documents these boundaries. An off
sample disables only new per-prefix resource snapshots/records, preserving all
operations, original stage probes/progress logs and final identity proofs.
The contract's 16 baseline-only on/off calibration processes are excluded
rows, not confirmatory replacements or an overhead correction to subtract.

`execute_totals` (`execute_sum`) remains the sum of run/approval body probes,
not the enclosing elapsed time. Nested scopes are inclusive and cannot be
summed as disjoint costs. Prefixes are dependent observations of fixed declared
B; T(256,8) is not standalone T(8,8), and unrelated process samples do not reveal
an exact 256th-operation cost. `process_maxrss_cumulative` is always a lifetime
high-water gauge, including in prefixes/residuals; it is never subtracted to
invent a phase peak. The controller also retains wait4's complete process peak.
Darwin RSS units are bytes; Linux units are KiB. `process_wall_ns` covers main
entry through workload/profile finalization before final JSON serialization;
the controller records complete process elapsed/CPU. Stage logs remain the
fixed P1 start/done format and impose measured overhead.

## Semantic proof and controlled campaign

A successful proof requires B distinct original operation IDs, request keys,
result digests, observations and roots; B completed/selected/spent positions,
B contacts in each HTTP journal, B-1 protected approvals and 4B-1 reductions.
Both ordered journals must equal the retained Prepared and Candidate evidence
and exact expected request/response bytes/digests. Missing/duplicate/reordered
contacts fail the bijection. Every selected observation must identify the paid
original and its actual selected reduction. Approvals must name the predecessor's
exact operation/prepared/result digests and be selected before successor effect
commit. Charges, reservation, allowed admission and dispatch bounds are checked.
Completed reopen/resume must preserve all original evidence without another
contact. Replay independently checks retained namespace/runs/roots/history.
This local deterministic proof does not establish external exactly-once effects.

`make_schedule.py` freezes the 7 bounds, 14 excluded warmups, six adjacent AB/BA
pairs per bound and seed 20261104. It binds source labels, executable bytes,
Go/runtime/environment, HTTP/prefix configuration, report version and SHA256
of the campaign controllers and driver source/module/build/README files.
`run_schedule.py` writes raw stdout/progress and post-process metadata before
rejecting failed proof/configuration or post-run binary hash/read errors. It
stops on failure without replacing a row. Excluded diagnostic rows use equals-style `-mode=direct`/`-prefix=false` flags
and retain their actual reported configuration and cannot enter `analyze_pairs.py`.

The analyzer validates every row/artifact and excludes only declared warmups.
It reports every paired difference, ratio of candidate/baseline medians and
median paired ratios separately. Its primary B256 criterion is all six positive
baseline-minus-candidate Execute differences, ties not wins; under six independent
fair null signs the all-six probability is 1/64. A one-sided six-sign tail is
reported explicitly. Shared-host drift and dependent prefix measurements limit
inference. Other bounds/phases/resources/growth models are descriptive. Inherited
per-phase >max(10%,0.5ms) flags are descriptive annotations, not new rejection
thresholds. The
small-bound guard is B1/B8/B32 slowdown above both 10% and 1ms. Profiling,
trace, counter and recovery processes remain separate from timing.

Generated scratch instrumentation may assign `counterSnapshot`; ordinary
public builds leave it nil. Dynamic newly observed counter keys start at zero;
disappearing/decreasing counters reject. Counter/SQL/scoped timing semantics,
coverage/overflow and overhead belong to separately reviewed Task2b tools.
Never include instrumented/profiled rows in headline medians.

Profile interpretation: stage labels from pprof.Do use context.Background;
server/runtime/system-stack CPU need not inherit those labels. Full profiles,
caller stacks and explicit unresolved costs remain necessary; labeled stages
are not a complete exclusive CPU partition.
