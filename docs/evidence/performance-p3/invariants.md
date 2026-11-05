# P3 single-readiness-call reuse invariant (pre-implementation)

This freezes the smallest candidate rule after Task 1 APPROVE. Production is
still baseline `926a899b651fccc48be52a054612a92afcf1ddfd`; Task 1 documentation
head is `5e2045d1556083548e2926a965c43695322388c0`. Task 2 deliberately commits
an expected-RED mechanism test before candidate production. This is a bounded
ownership argument with executable controls, not proof against arbitrary writers
or a speedup result. The [contract](contract.md) still governs acceptance.

## Rule and placement

Inside one `Store.selectReady` invocation, after the existing current projection,
retirement and generation checks, load the exact retained `run.ProgramID` on its
first reached visit. Retain only successful **unbound** results, keyed by the
entire ProgramID string, and only if no `Author.Ports` value has
`Peer == HandoffPeer`. Determine that eligibility **before calling binding**.
Every visit, including a hit and an excluded program, still calls
`boundCompositionProgram` with its actual full RunID. Failed loads immediately
return their original error; there is no negative entry. An excluded program
loads again on every visit. Do not store a bound program or readiness result.

The map is a local variable, inaccessible to Store, globals, snapshots or later
calls. It ends on every selector return, including errors and quiescence, before
the caller's subsequent `Commit`. Each later call and reopened owner reloads.
A conservative port scan and the map's allocations are part of candidate cost.
No new cloning is proposed: existing member copies remain; excluded handoff
loads preserve their original decoding and binding costs.

## Identity and validation inputs

`store.go:loadProgram` checks in this order: program-row lookup, supported stored
semantics, SHA-256 of the retained encoded bytes against the exact ID, stored
byte limit, strict JSON decoding, embedded/stored semantics equality, the stored
request-authority feature context, then `validateProgram`. Strict decoding
rejects unknown and trailing fields. `program.go:validateProgram` calls metadata,
operand/instruction and collection validation. Metadata includes compiler and
semantic compatibility, positions, identifiers, retained input/output/receive/
port/effect/invocation schemas, templates and collections. Schema checking
validates bounds and retained IDs. This entire successful result is reusable;
a hash alone is not an independent validation certificate.

The same pinned connection, `s.limits`, supported-version implementation,
retained bytes/semantics and stored format context apply throughout a legitimate
call. `store.go:insertArtifact` validates first and accepts an existing ID only
when both bytes and semantics match. Production has no program UPDATE/DELETE.
Limits are assigned by create/open, and supported schema promotion is serialized
with other Store operations. `selectReady` holds `Store.mu` for its entire body.
`store.go:acquire` enforces cooperative single-host flock ownership. `Close`
cancels/joins registered owner users and Run before acquiring that mutex to close
the connection/release ownership. The focused concurrent-close control also
checks that an active direct selector finishes before connection closure.

**No selector transaction snapshot is asserted.** The mutex and cooperative lock
do not exclude a hostile SQL connection. Arbitrary unsynchronized mutation during
an active read is outside this model. Pre-call corruption must still fail when
first reached. Tests that edit SQL bytes, format context, limits or descriptors
explicitly identify those disposable impossible histories; their corruption is
not treated as a supported API. Format 14's positive fixture uses a real allowed
authority attempt and retained conflicting receipt, rather than direct schema
editing. Formats 13/14 do not acquire continuation, dispatch-admission,
reconciliation or invocation-template support from this change.

## Alias and ownership audit

| Source path | Nested ownership and obligation |
| --- | --- |
| `program.go:decode`, `loadProgram` | Fresh decode owns Code, Author, literals, maps and pointers. Struct assignment alone is shallow. |
| `invocation.go:boundCompositionProgram` | Reads current stored format and membership by subject **and generation**, then retained binding/template on every visit. No hit may skip it. |
| `composition_binding.go:resolveMemberProgram` | Encode validates the original; exact member ProgramID is checked; decode creates fresh nested storage before changing endpoint operands, receive provenance/operation, generation and Author.Ports. Existing revalidation/copy costs remain. |
| `invocation_handoff.go:bindHandoffCaller/Ports/Port` | The concrete exception: shallow Program copy writes shared Code Effect.Target, Match.Source and Author.Ports. It changes HandoffPeer itself. Exclude before binding, including paths that might ultimately deep-copy. The actual retained handoff test checks two same-ID callers and two loads. |
| `program.go:validateProgram`, `author_metadata.go`, `collections.go:validateCollections` | Read instruction/metadata maps and operands; construct local validation structures. They do not transfer ownership of source data. |
| `schema/schema.go:Retain/Check`, `schema/collections.go:cloneRecord/Field.Retain` | Retain checks source then recursively copies records and enum slices; canonicalization sorts the copy. Nested source schemas must remain unchanged. |
| `program.go:Reduce/eval` | Revalidates program and bound size; validates the actual projection; clones Data values and cursors; evaluates cloned literal/reference values; builds new intents/payload/update maps. AuthorityUse is copied before returning its pointer (its contained fields are value/scalar data). |
| `collections.go:reduceCollection/cloneCursors`; `schema/collections.go:Clone/CloneValues` | Writes the new projection and owned cursor snapshots/lists. Nested list records are recursively copied; list accessors also copy. No cached program is used as mutable cursor/projection state. |

The controls mutate returned bound Match literals, Effect payload literals,
metadata position slices, receive maps and nested port schema enum slices, then
compare exact unbound and second-bound bytes. A real collection program is
retained through Create; mutating the reducer's returned list decoder cannot
change its literal or source schema. Returned AuthorityUse mutation cannot change
the source effect. The handoff shallow-copy witness intentionally demonstrates
aliasing, while the retained handoff test exercises the production path and
checks the conservative exclusion's required load count. These are focused
probes supporting the source audit; they do not claim exhaustive pointer proof.

## Per-run semantics, ordering and interruption

Keep the existing sorted RunID traversal and rotation after `last`. Read actual
runs, projections, current generation and retirement before touching the cache.
Keep the existing ordered observation scan, full RunID equality, consumption
check, ErrIneligible handling and first eligible choice. Every original Reduce
call still receives its actual program binding, projection, observation, RunID
and sequence. It still checks projection/schema/output bounds and creates the
same authority tuple: RunID (including generation), epoch, sequence and revision.
No cached validity implies readiness, scope membership, an approval or authority.
No extra read or early binding is allowed for a skipped or unvisited run.

The frozen reference in `readiness_reference_test.go` copies the baseline
selector body with only its name changed, records source commit/function hash,
and never calls a candidate reuse helper. `p3Compare` compares the entire
`readyRun` (Authority and observation), error text and relevant `errors.Is`
identities. Both sides run without mutation between them. Explicit expectations
also assert waiting, chosen run/observation, rotation, sequence and epoch changes,
and deterministic errors, so differential equality is not the only oracle.

Pre-cancelled contexts and closed Stores must fail at their existing entry/read
boundaries. A SQL authorizer cancels on the second run-projection read for a
logical mid-call control; that query remains on hits. Concurrent Close is joined
through channels without elapsed-time assertions. Removing redundant SQL removes
some elapsed cancellation and injected-I/O opportunities. This work promises
neither the same wall-clock interruption point nor a failure for a removed read.

## Frozen control map

All names below start with `TestP3Readiness` and live in the three focused test
files under `internal/executor`.

| Required distinction | Executable control |
| --- | --- |
| Shared ID, distinct inputs/ports | `SameCallBoundPrograms`: genuine two-receive members share an ID; the first consumes its request and waits, then the second is selected with its own full authority/observation in the **same call**. Actual projection/load/membership-read counters prove both visits and bindings. `SharedInputsAndPorts` additionally probes bound copies and distinct per-run input peers. |
| Same spelling, generations | `GenerationAndReopen`: genuine lifecycle Retry creates worker generation 2; old completed generation is skipped; reopening changes epoch, preserving sequence/RunID. A later disposable stale-open projection isolates the generation check. |
| Shared/few and distinct IDs | `LoadReuse`: 32 eligible waiting runs, four genuine IDs versus 32; exact counts, separate no-visit control. |
| Independent Store/context/lifetime | `IndependentStoresAndCancellation`, `FreshValidationEveryCallAndStore`: same artifacts with independent readiness, later validation-limit and byte-corruption checks. |
| Aliasing and handoff | `SharedInputsAndPorts`, `ReduceNestedIsolation`, `HandoffShallowAliasWitness`, `HandoffExcludedBeforeBinding`. |
| Multiple observations/rotation | `OrderingAndLaterCalls`: two genuine accepted inputs for one run, sorted ID choice, later run on rotation, real Commit and changed sequence. |
| Waiting before ready/quiescence | `OrderingAndLaterCalls`, `LoadReuse`, and `SharedInputsAndPorts` ineligible-peer observation. |
| First-use invalid/missing program | `EarlyReturnAndDisposableCorruption`: missing row, wrong bytes, unsupported semantics and limit; early ready run prevents eager failure, rotation reaches exact failure. Repair follows failed load. |
| Per-run binding errors | `SameCallSecondBindingCorruption`: the same genuine waiting-first cut with only the second member binding corrupted; two actual visits/bindings precede the expected error. Original `BindingCorruption` and handoff descriptor controls remain. |
| Between-call changes | Real admission/Commit/Retry/reopen plus explicitly disposable later limits/bytes/context mutations. |
| Cancellation/closing | `IndependentStoresAndCancellation`, `CancellationAtRunBoundary`, `CloseJoinsSelector`. |
| Ineligible/no visits | Empty Store in `LoadReuse`; genuinely completed run with unreachable corrupt artifact in `ClosedRunsNeverLoaded`; `DisposableRetiredOpenRun` separately isolates the retirement check. |
| Request authority context | `RequestAuthorityContext`: separate format 13 and genuine receipt-conflict-promoted format 14; later disposable context corruption rejected. Broader authority/recovery qualification remains Task 4. |

## Mechanism instrumentation and expected RED

`p3CountLoads` installs the pinned SQLite driver's authorizer only during a
selector call, always permitting queries. It counts preparation of reads of
`programs.bytes`; these direct QueryRow calls have no prepared-statement cache.
The baseline calibration must observe exactly one such read per eligible run.
The successful positive fixtures ensure these are successful full loads, not
merely attempted reads. This is a test diagnostic, not production telemetry.

The retained RED is `LoadReuse/shared_four`: baseline and current production
both perform **32**, expected candidate **4**, with the failure text
`redundant successful readiness loads: got 32 want 4`. `distinct` observes 32
and passes; `no_visit` observes zero and passes. No elapsed-time assertion is
used. Source/binary commands and raw outputs belong under
`/private/tmp/alan-p3-j216kxoz/task2/` with passing semantics separate from RED.

`selector_probe.py` additionally creates a disposable source copy and injects
counters at real loadProgram, boundCompositionProgram and Reduce entries. Only
its generated test activates collection around either selector. It retains
input hashes, exact patches and generator identity; production workspace files
remain unchanged. For 32 waiting runs, bindings stay **32**, reductions **0**;
with a sole late eligible observation, bindings stay **32**, reductions **1**.
The same-call composition probe separately requires **2** visits, **2** bindings
and **1** reduction, with baseline loads **2**. These semantic controls permit
pre-optimization loads 2 or candidate loads 1; the original RED independently
requires the optimization. The first bound result must never become the retained
source passed into the later member binding.
The probe independently reports load entries; ordinary RED owns the required
load reduction. Probe counts are not timings. Candidate reuse must still be
qualified by exact successful load/decode/validation counts in the Task 1-style
campaign instrumentation, without subtracting instrumented timing overhead.

## File/function change map and remaining gates

Task 2 adds only this invariant, `selector_probe.py`, and
`readiness_{reference,reuse,alias}_test.go`. No production byte, dependency,
public API, schema, index, ownership or scheduler policy changes. The proposed
Task 3 production boundary is `runner.go:selectReady` plus, if useful, one small
private predicate scanning unbound author ports. It does not modify binding,
Reduce, loadProgram, commits, approvals or observation scans.

Before production: independent invariant/alias/RED review. After production:
all these semantic controls and RED must pass, disposable count comparison must
preserve remaining work, and the contract's later differential, authority,
recovery, compatibility, full suite/race/Staticcheck and performance gates remain.
No full suite is claimed on this intentionally RED commit. No unrestricted
immutability, cache lifetime, latency, allocation or peak-memory result follows.
