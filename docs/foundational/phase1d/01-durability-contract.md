# Phase 1D durability contract

Status: implementation profile, frozen before runtime code. The [plan](00-plan.md) records baseline and exclusions. Concrete schema/API details may be refined without weakening these boundaries; material changes are recorded in qualification.

## Storage and failure model

One active executor owns a local SQLite store on one host. Use `github.com/mattn/go-sqlite3 v1.14.52` with its bundled SQLite, `database/sql`, a single pinned connection, WAL, `synchronous=FULL`, foreign keys, and bounded busy timeout. Read back effective settings on every opened connection, including inspection/replay connections. Fail closed on unsupported schema, semantics, artifact hash, or settings. No silent default fallback. Record the SQLite engine version in inspection/evidence.

The driver is selected from its [current release](https://github.com/mattn/go-sqlite3/releases/tag/v1.14.52) and [connection documentation](https://github.com/mattn/go-sqlite3#connection-string): it provides a maintained pinned database/sql implementation without other Go infrastructure dependencies. It requires CGO and a C compiler. A pure-Go driver is a viable alternative but adds dependencies not needed on this supported host. SQLite documents [WAL's same-host requirement](https://sqlite.org/wal.html) and [FULL synchronization](https://sqlite.org/pragma.html#pragma_synchronous). Configuration verification is required because unknown PRAGMAs can be silently ignored.

An OS advisory file lock is held for the executor lifetime and released by process exit. Canonicalize paths and handle aliases; reject unsupported unsafe file aliases. Cooperating second executors fail before epoch changes or writes. This is single-host exclusivity, not distributed fencing. External database writers, deleting/replacing a live database, hostile filesystem aliases, network filesystems, disk corruption, and hardware lying about synchronization are outside the profile. Inspection/replay uses read-only/query-only connections and consistent snapshots.

Tests kill an acknowledged subprocess with no graceful shutdown. An uncommitted transaction rolls back; a successful COMMIT acknowledged by a failpoint survives process death under the storage assumptions. A crash exactly inside an unacknowledged COMMIT can leave either whole outcome; sequence/history/intent rows distinguish them. No partial transition is allowed. These are process-crash tests, not simulated power-loss or disk-failure tests.

## Durable schema and identity

Schema version 1 has these logical relations (exact DDL lives with the implementation):

| Relation | Key and retained content |
| --- | --- |
| metadata | schema and supported semantic version |
| programs | content identity, semantic version, immutable complete program bytes |
| subjects | logical ID, current generation, owner/revision, retirement and explicit retry-cancelled policy |
| grants/routes | explicit source/target operation permission and observation/message permissions |
| runs | subject/generation; immutable program ID and initial projection; epoch, sequence, status, current serializable projection |
| run origins | one static root per subject, or exact accepted Retry operation and predecessor; original artifact/initial projection |
| history | run/sequence; epoch, selected continuation/observation, successor projection, exact generated intent descriptors, terminal outcome |
| observations | immutable logical identity, provenance, exact run/obligation correlation, payload and consumed sequence |
| admissions | trusted transport domain plus raw transport ID, immutable descriptor, original run binding, accepted/rejected outcome, associated inbox insertion |
| intents | execution/sequence/ordinal ID, immutable descriptor, source revision, obligation and target binding |
| attempts | authorization identity, intent ID, source authority and protocol evidence; unresolved outcome remains explicit |
| lifecycle results | operation ID, complete request descriptor, accepted/rejected result/reason and successor generation |
| settlements | authenticated outcome identity, original operation/intent and execution/obligation binding |
| external attempts / receipts | immutable permitted request before transport; independently recorded verified receipt, if known |
| terminal obligations | frozen run outcome and retained notification destinations/delivery status |

Records are not garbage collected in this increment. No migration or retention horizon is promised. SQLite uniqueness/foreign-key constraints and explicit immutable-descriptor comparisons enforce identity; a reused ID with changed bytes is an error, never an update. JSON serialization is versioned and deterministic where compared. Distinguish logical subject, generation, epoch, ownership revision, supervisory obligation, intent/lifecycle operation, transport ID, logical observation, and external/business key.

The concrete admission key is `["admission", trustedDomain, rawTransportID]`. Input, ordinary-message and terminal APIs establish their own domain; callers cannot forge it through input payloads. All otherwise valid raw IDs remain legal, including strings resembling runtime-generated IDs; existing nonempty-identity and byte-limit checks still apply. The same raw ID can occur in different domains, while a changed descriptor within one domain conflicts. Acknowledgments reference the qualified durable admission ID. Inspection exposes the domain, raw ID and qualified ID. This corrects the retained transport-collision counterexample without excluding its valid input.

## Atomic units

1. **Provisioning:** retain static programs, subjects, first runs, routes and grants in one initialization transaction; reopening never provisions again.
2. **Activation recovery:** while holding exclusivity, validate retained artifacts/history/projections, then advance active-run epochs in a transaction. It does not change generations, projections, budgets, pending IDs, or consumed identities. A second crash may advance epochs again with the same logical state.
3. **Execution commit:** validate current generation, epoch, expected sequence and openness; atomically consume one selected eligible observation (if any), retain its selection, advance residual control/data, append reconstructible history, retain immutable intents, and retain terminal outcome/notification obligation when closing. A speculative evaluation cannot dispatch.
4. **Message admission:** compare/retain transport identity and descriptor; choose its first run binding; atomically retain acceptance or rejection and accepted inbox insertion. Retransmission returns that result even after a generation change. No old service request is rebound to current work.
5. **Attempt authorization:** separately validate the committed source intent and closure/revision rules, and persist authorization. Source closure denies new authorization, but does not erase prior authorization or imply that an external mutation failed.
6. **Lifecycle decision:** separately validate retained request identity and source authorization, then target grant, expected generation, revision, retirement and eligibility. Atomically retain target transition plus accepted result, or an immutable rejection. Check for an existing result before reevaluating circumstances. Observation permission does not confer lifecycle authority.
7. **Observation settlement/admission:** separately retain authenticated exact correlated result or receipt, then expose it to its original execution. Late evidence may settle an old obligation; closed execution stays closed. Ordinary consumption happens only in a later execution commit.
8. **Terminal delivery:** separately admit retained frozen terminal notifications under explicit routes and record delivery acknowledgment. Lost acknowledgment can repeat transport admission but not its binding or logical consumption.

No transaction spans a whole workflow, a nested composition, or runtime plus endpoint. Intent commitment, dispatch authorization, external application/target decision, outcome recording/admission and ordinary consumption remain distinct.

## Computation and ordinary policy

Programs retain immutable bytes and semantic version `alan/1d/v1` (or the exact constant documented in source). The closed instruction language contains generic local data operations, explicit finite branch/jump, typed receive/correlation, immutable effects/messages and terminal outcomes. Each scheduled reduction executes finite work; loops yield through repeated commits. Residual program counters and data are serialized. Operational resource limits are explicit errors, not hidden historical research bounds.

The first concrete subset uses one serializable program counter per run and one generic instruction per commit; independent runs supply concurrent progress. Integers are checked `int64` values. Default operational byte limits are separately 1 MiB for program artifacts, local projection, incoming observations, and the complete generated intent-array JSON. These are configurable admission/evaluation limits, not architectural scalar/generation bounds. An empty intent array occupies two bytes. Exceeding a limit rejects the local step before writes; it does not manufacture a committed program failure. Reopen/replay must use limits sufficient for retained history. There is no history garbage collection, and startup/replay scans retained history.

The runner chooses eligible continuations and records selection, executes bounded pure reduction, and commits before dispatch. Waiting work does not prevent other eligible runs from progressing. Scheduling order is an implementation choice with no unbounded fairness promise. External transport timeout is physical uncertainty, not deterministic program failure. No branch uses program names, roles, or ancestry.

The ordinary inner policy initializes a logical obligation, expected child generation and retry budget; consumes an authenticated committed child failure; branches on remaining budget; commits a Retry intent and budget reservation/pending ID; consumes the exact retained result; updates expected generation; and continues or escalates. Duplicated notifications do not create another decision. A physical restart preserves this ordinary projection. The outer observes only public inner outcomes and applies its own ordinary computation. The worker follows versioned instructions and explicit inputs, including an input that changes its behavior after a retry; no runtime generation special case manufactures success.

Retry creates a fresh generation only after an explicit accepted lifecycle operation. CancelRun addresses one run, with configured retry-cancelled policy; RetireSubject permanently denies future generations. Physical activation loss preserves the active logical run; it is not application failure. Fresh-supervisor child reassociation and cross-generation supervisory budgets are excluded; same-execution recovery is required.

The implemented lifecycle descriptor uses the committed intent ID as its operation ID; this remains distinct from obligation, target generation and activation epoch. It retains the source revision separately from expected target revision. `LoadLifecycleRequest`, `AuthorizeIntent`, `DecideLifecycle` and `SettleLifecycle` expose the read/authorization/decision/settlement boundaries; ordinary `Commit` consumes the result later. Repeating an identical accepted or rejected decision returns its first result before current target eligibility is reconsidered. Source denial is retained separately from target rejection.

Accepted Retry starts from the original static root's artifact and initial projection. Explicit roots need not start at generation 1. Decision records retain ordered before/after target state and exact creation/closure links, including closure at sequence 0. CancelRun and AbandonRun preserve an already terminal outcome; RetireSubject also sets permanent retirement. Lifecycle outcomes correlate to the request's original expected target generation, even when rejecting a stale-generation request. Late settlement may retain evidence for a closed requester, which remains closed.

Generation identifiers use checked `int64` representation. Eligible Retry at `MaxInt64` returns operational `ErrLimit`, with no target result or successor; its previously committed intent, budget reservation and authorization remain retained. This capacity error is not a committed lifecycle rejection or an application failure. No migration or wider representation is supplied in this profile.

## External-resource protocol

A separate executable owns a separate durable store and listens only on local loopback. Its mutation transaction does not access the executor store. In the idempotent profile, the operation key is the stable original intent key; exact key/parameter repeats return the retained original outcome and conflicting parameters fail. The fixture can commit then withhold/drop response. Executor recovery may query/reattempt only through the declared protocol, never inspect the endpoint database.

A separate non-idempotent profile demonstrates ambiguity. If a mutation may have applied without a receipt and the endpoint cannot resolve the key, retain unknown evidence. Repeating that mutation can apply twice. A new generation's new execution-scoped intent does not automatically inherit/redefine a business operation key, and successful computation recovery does not imply safe business retry.

The concrete fixture exposes strict JSON `POST /apply`, with a retained whole request descriptor under its key. Idempotent recovery repeats that protocol with the original key, parameters, authorization and request epoch; no separate query method is implemented. A static external grant names source subject, literal loopback URL, `apply`, and source revision. The resource's fixed protocol generation 1 does not bound ALAN subject generations. The executor shares wire data types with the fixture but calls only HTTP, never the fixture's storage methods.

An external attempt record commits before transport. A verified receipt commits before its separate observation settlement. Each runner invocation sends at most once per external intent; an existing non-idempotent attempt is never automatically sent again, including after a crash before its first physical send. The two-second HTTP timeout, response loss and unsuccessful/undecodable responses leave explicit `UnknownExternal` evidence. They do not manufacture program failure. Ready work may continue after an unknown result. Receipt provenance is established by the trusted configured loopback protocol and runtime admission, not a remote cryptographic identity protocol.

The endpoint retains contact and application records in its own WAL/FULL database, verifies effective settings on its writer and read-only inspection connections, and reports its actual SQLite source identity. The counter fixture deliberately has no business-level deduplication or compensation. Controlled process death establishes only the stated local process-crash behavior.

## Replay and independent evidence

Read-only replay reconstructs from initial projection, retained artifact and recorded selections/observations; compares every successor and regenerated intent descriptor to history and current projection. Its object/command has no dispatcher or lifecycle-write capability. Verify live ledger and endpoint state are unchanged before/after replay. Live startup recovery can later dispatch unresolved authorized work; that is separately labeled live recovery.

Independent tests inspect committed records and public projections rather than relying on reducer assertions. Compare selected allowed traces with the preserved research model without sharing the reducer or policy oracle. Replay calling the same reducer is a consistency check, not independent conformance. Fault controls must expose lost rejection after changed eligibility, reset budget/pending state in the same execution, and dispatch before commit, with the normal contract check detecting each.

The central acceptance case retains one supervisor execution/obligation, one committed recovery intent and budget charge, one target successor, and the original result across SIGKILL after lifecycle decision. The successor must actually execute ordinary instructions after recovery. Qualification records concrete identities, barriers, process exits, inspections, and limits.

## Preimplementation review clarifications

Independent contract review returned specification PASS to proceed, with implementation quality deferred until source/controller review. These clarifications make existing requirements testable; they do not increase the authorized scope.

- **C1, lifecycle reconstruction:** validate ordinary history separately from authority-induced closure and successor creation. Retained lifecycle results explain those changes, including zero-step successors. Lifecycle closure atomically retains its terminal outcome/notification obligation. Corrupt successor links and selected-observation records must fail startup before epoch changes or dispatch.
- **C2, source denial:** a valid committed intent denied source authorization retains its denial/suppression disposition. It does not become a target-authority rejection or disappear as a transient error. A still-open requester can observe its retained denial under the appropriate correlated protocol; a closed requester stays closed. An uncommitted/forged identity is an error, not a new durable operation. Existing authorization survives later source closure.
- **C3, physical retransmission:** the idempotent profile may retransmit under the same retained authorization and original operation key; that is not permission for a fresh intent or business operation. Record physical transmission attempts separately when needed for diagnosis. Non-idempotent uncertainty is not selected for blind automatic retransmission. Any deliberate duplicate-mutation experiment is explicitly identified as such.
- **C4, aliases:** canonical relative/symlink paths must converge to one ownership lock; multiply linked database files are rejected before SQLite writes. Do not unlink a sidecar lock on close. File replacement while active remains excluded.
- **C5, replay:** use SQLite `mode=ro` plus `query_only`, without `immutable=1` on a live WAL store. Inspection/replay read one consistent snapshot and never call the live recovery opener. Compare semantic ledgers and external effects; SQLite sidecar bookkeeping is not itself an ALAN dispatch.
- **C6, barriers:** acknowledgments identify the scenario/process start, PID, store, boundary, and relevant run/generation/epoch/sequence/intent. The controller kills that exact process and verifies signal exit. A post-commit acknowledgment follows successful COMMIT; an in-transaction acknowledgment follows writes but precedes COMMIT. The second recovery crash correlates the same obligation/operation and later epoch. Timeouts bound failed tests only.

Record the actual bundled SQLite engine version and source ID. The selected engine must include the WAL-reset fix described in [SQLite's current WAL documentation](https://sqlite.org/wal.html#the_wal_reset_bug). The complete original review and eventual dispositions will be retained in the qualification package.
