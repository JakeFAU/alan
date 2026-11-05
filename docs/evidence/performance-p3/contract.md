# P3 experiment contract

Production baseline is `926a899b651fccc48be52a054612a92afcf1ddfd`, tree
`a58fe3b0ef0b9a7525c46c4f0a662774e6d4221d`. This contract is frozen before
candidate production changes or candidate results. Task 1 collects baseline
attribution only. Independent attribution/ownership challenge must approve the
premises below before Task 2/3 proceeds. The controller-owned [plan](plan.md)
provides the complete executable task sequence; this file owns measurement rules.

## Proposed mechanism and source premises

Within one `Store.selectReady`, lazily reuse a successfully validated **unbound**
program by its exact retained ProgramID, only when binding cannot mutate the
retained object. ProgramID is the SHA-256 of retained encoded bytes. `loadProgram`
also checks supported stored semantics, size under Store limits, strict JSON,
embedded semantics, request-authority format context and `validateProgram`
(author/compiler metadata, schemas, instruction and collection rules). All these
inputs must remain fixed during the call; hash equality alone is insufficient.

The selector holds `Store.mu` on its pinned connection. Supported mutations use
that mutex and cooperative single-host owner exclusion. `Close` joins Run and
owner users before taking that mutex and releasing its flock. Programs are
inserted only after validation; an existing identity with different bytes or
semantics conflicts. There is no supported program UPDATE/DELETE. Format context
and limits cannot change within the call. No transaction snapshot is asserted
for this selector, and a Go mutex does not exclude an arbitrary SQL writer.
Hostile unsynchronized writes during the call remain outside this ownership
model. Corruption present before first use must still fail at that first use.

**Concrete alias hazard:** `bindHandoffPorts` passes a shallow Program copy to
`bindHandoffPort`, which mutates nested Code Effect/Match and Author.Ports.
Unconditional struct-by-value map reuse is unsafe. Preferred smallest rule for
independent review: exclude any program with an Author port whose Peer equals
`HandoffPeer` from retention, keeping its original load/binding every visit.
This conservative exclusion also covers paths that would ultimately deep-copy;
it costs a scan of author ports on first successful load and leaves handoff
work unchanged. Alternatively, isolate only that binding path with a complete
copy, retaining its cost, but this requires additional alias tests and source
review. No exclusion may bypass binding or suppress binding errors.

`resolveMemberProgram` already Encode/decode-copies before modifying endpoints,
receive selectors and port metadata. `Reduce` clones projection values/cursors,
evaluated values and output payloads, copies AuthorityUse, and mutates the new
projection; validations and schema checks read program data. Task 2 must test
nested aliasing and inspect all relevant paths rather than relying on this
preliminary read. Every reducer validation, per-run binding, full RunID,
generation/retirement, projection, sequence/epoch, current scope and observation
check remains. The map ends on every return before the subsequent Commit.

The prediction is counts, not an asymptotic runtime promise: for reusable
programs, successful readiness loads/decodes/unbound validations become the sum
of distinct IDs actually reached per call, replacing the sum of eligible visits.
Excluded handoff programs retain one load per visit. Binding/reducer and selected
outcome counts must stay equal. Laziness preserves rotation, sorted observation
order, first-use errors and early return. Failed loads are not successful entries.
Eliminated SQL reads eliminate some elapsed cancellation/injected-I/O chances;
logical boundary comparisons are required, not identical wall-clock faults.

The fresh baseline counter prediction for the successful bounded series is:

| Bound | Actual eligible loads | Sum of distinct IDs per call | Predicted eliminated loads | Predicted eliminated SQL executions |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 4 | 4 | 0 | 0 |
| 8 | 172 | 49 | 123 | 246 |
| 128 | 44,297 | 769 | 43,528 | 87,056 |
| 256 | 178,601 | 1,537 | 177,064 | 354,128 |

Each skipped baseline load removes its program-row query and stored schema
request-authority query. Thus B256 total Execute SQL is predicted to change
3,600,555 → 3,246,427, conditional on no other instrumentation/semantic change.
These two genuine series ProgramIDs contain no HandoffPeer ports; exclusions
must be independently checked. Per-run bindings remain 178,601, readiness
reductions 1,023 and observation attempts/selections 511 (includes 255 local
approvals plus 256 effect-result observations). Logical committed reductions
remain 1,023, contacts/effects/charges/results 256 and later approvals 255.
The latter workload counts do not define generic ALAN correctness.

## Workload and stopwatch

Reuse the unchanged `docs/evidence/performance-p2/driver` directory by source
hash. Primary measurement explicitly uses `-mode=direct -prefix=true`, the
existing deterministic in-process adapter, with fresh process and fresh store
for each row. Its HTTP default is deliberately overridden under this P3
contract. Direct adapter contacts are not HTTP contacts or remote applications.
Each successful bound N includes N Run calls and N-1 fresh public
InspectExecution/ContinuationBasis/ApproveContinuation cycles inside
`execute_window`. Prefix resource probes, existing progress reports and returned
report validation remain inside. Provisioning, Create, original-state check,
final report/proof, completed reopen/Run and Replay retain separate stages.
No later observation is preapproved; no cheaper private approval path is used.

The Task 1 schedule is one uninstrumented pilot at each 1,8,128,256, separate
counter rows at those bounds, one B256 coarse row and one B128 CPU/allocation
profile. These single observations choose the experiment; they are not paired
results, medians or evidence of a candidate speedup. All failures remain.

## Exact confirmatory design

[schedule.json](schedule.json) freezes 98 ordered rows. For each bound in
1,8,16,32,64,128,256, run excluded warmup A then B, followed by six adjacent fresh
process pairs in exact order AB, BA, AB, BA, AB, BA. A is the baseline and B the
candidate. This is 14 retained excluded warmups and 84 measured rows. No pilot
row enters this campaign. Bind both source commits, complete executable input
manifests, actual binary hashes, controller hashes and exact absolute argv/cwd
before the first warmup. Binding immutable identities must not change row order,
workload, exclusions or acceptance rules. The old P2 strict paired validator
expects HTTP; P3 must qualify a direct-mode validator adaptation before using
this schedule. Diagnostic mode acceptance alone is insufficient for final paired
validation. Negative controls must reject wrong direct protocol/endpoint and
contact capability/effect/payload identity, dropped/duplicate/reordered prefixes,
wrong logical counts, missing phases, changed binary/source/tool/environment,
missing/failed/replaced rows and profile/counter contamination. Preserve P2
HTTP tools unchanged; independently review the P3 adapter before any confirmatory
row. No smaller campaign, adaptive sample expansion, dropped slow row,
until-green retry or arithmetic subtraction of instrumentation overhead.

Use the same Go1.27.3/CGO/pinned SQLite toolchain on both sources, GOMAXPROCS=4,
GOGC=100, GOMEMLIMIT=off, GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off,
GOSUMDB=off and unchanged durability/concurrency/payload/reporting/approval policy.
The exact existing P2 ENVIRONMENT dictionary is retained in the schedule binding.
Record whitelisted environment only, full command/process exit, CPU, allocation
bytes/counts, GC metrics, whole-process peak RSS and before/after binary hashes.
Preserve child outcomes if post-exit metadata reads fail. Record shared-host load;
thermal state, caches and CPU placement remain uncontrolled. Run serially with
no task-launched heavy qualification/build/profile/compression during timing.

## Controls, guards and stopping

Separate diagnostic controls must include 32 eligible runs sharing four genuine
programs and 32 eligible runs with 32 distinct genuine programs. Freeze their
source/inputs before executing candidate controls. They quantify realized
reuse and no-sharing overhead, not invented provisioned-run reuse. Include B1,
early-return, waiting-before-ready, multiple eligible observations, errors on
first reached use, per-run binding corruption, generations, separate Stores,
reopen, changes between calls, cancellation/Close and handoff alias controls.
Genuine programs are required for positives; impossible mutation histories are
explicitly test-only corruption probes. Authority format13/14 is a separate
fixture, with its restrictions and immediate-selection mismatch unchanged.

Report every paired value, median baseline/candidate and median paired ratio.
Useful performance requires all six B256 baseline-minus-candidate Execute
differences positive and a median reduction exceeding 5%, with lower cumulative
Go allocation and the predicted mechanism counts. The six-sign 1/64 tail applies
only under independent fair null signs; shared-host conditions limit inference.
A result indistinguishable from noise is a no-change result, not permission to
expand sampling. No universal latency or peak-memory improvement is predicted.

Small workloads B1/B8/B16/B32 must not have candidate median Execute slowdown
exceeding **both 10% and 1ms** (equivalently max(10% baseline,1ms)). Apply the same
max(10%,1ms) guard at every bound to each separately measured non-Execute phase
and summed approval snapshot/basis and approval establishment phase. No pilot
noise justifies weakening this initial guard. Report all phase data even if below
it. Whole-process RSS growth above max(10% baseline,8MiB) requires explicit
resolution before acceptance; allocation traffic is not peak memory.

Any semantic difference, unsupported ownership premise, unexplained mechanism
count, material guard violation or failed qualification stops acceptance until
resolved with original failures retained. Withdraw the production change if this
single optimization cannot meet the invariant/guards; do not add a second
optimization. Full ordinary/integration/integration-race, focused semantic/process,
public consumer, historical-reader, authority, Staticcheck and documented
build/vet/module gates are controller-owned later tasks. This contract approves
neither production code nor delivery. No merge or release.
