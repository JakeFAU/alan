# Inspecting retained executions

`runtime.InspectExecution(ctx, path, limits)` returns a detached `*runtime.ExecutionSnapshot` or an error. It reads an existing store at a single committed SQLite snapshot. Call it again to see later work. Keep the first snapshot to compare cuts. The raw `runtime.Inspect` and deterministic `runtime.Replay` interfaces remain available.

## Follow an original outcome

An investigator calls its two workers and saves the operation handles in its retained state. This example starts with the logical investigator and follows its actual selections.

```go
func showSelections(ctx context.Context, db string, w io.Writer) error {
    cut, err := runtime.InspectExecution(ctx, db, runtime.DefaultLimits())
    if err != nil {
        return err
    }
    parent, ok := cut.Run(runtime.RunID{Subject: "investigator", Generation: 1})
    if !ok {
        return errors.New("investigator generation is absent")
    }
    for _, handle := range parent.InvocationIDs {
        call, _ := cut.Invocation(handle)
        if call.Outcome == nil {
            fmt.Fprintln(w, handle, call.State, call.Dependencies)
            continue
        }
        fmt.Fprintln(w, handle, call.Outcome.Status, call.Outcome.ID)
        if call.SelectedAt != nil {
            step, _ := cut.Reduction(*call.SelectedAt)
            fmt.Fprintln(w, "parent consumed", step.SelectedObservationID, "at", step.Sequence)
        }
    }
    return nil
}
```

Import `context`, `errors`, `fmt`, `io` and `github.com/JakeFAU/alan/runtime`. The [standalone reader](../examples/inspect-execution/main.go) prints worker roles, lookup quotations, original model operations, usage, outstanding dependencies and failures. Its qualification compiles it under an unrelated Go module path. [Run the worked examples](../examples/inspect-execution/README.md)

## Types and identity

| Public type or accessor | What it represents |
| --- | --- |
| `ExecutionSnapshot` | Store namespace and physical storage version, optional protocol features, logical runs, original operations, invocations, effects, observations, admissions, assets, composition roots and pending obligations. |
| `Run(RunID)` → `ExecutionRun` | Subject plus generation, original program ID and semantics, initial and current projections, latest activation epoch, committed sequence, authority, current instruction, and optional receive selector. `Origin` identifies a provisioned root, lifecycle retry, or invocation member. A retry names its predecessor and lifecycle operation. An invoked member names its owning invocation and operation. |
| `Invocation(operationID)`, `InvocationByID(invocationID)` → `InvocationView` | The immutable source operation is available before admission. Admission adds a namespace-bound invocation ID, its decision and its exact template members. A refusal can publish a rejected outcome without creating a child. |
| `InvocationView.Replies`, `ResponseOperationID`, `Outcome` | Retained entry-member reply attempts, the original reply supplying a selected response if present, and the structured child publication. A failed publication may still carry an authenticated diagnostic reply. `ParentSelection` reports `unpublished`, `unconsumed`, `consumer_closed` or `selected`. A terminal generation cannot newly select a delivered outcome. |
| `LifecycleDecision(operationID)` → `LifecycleResult` | Original authority decision, including its optional before/after target witnesses and retry successor. `LifecycleDecisions` is ordered by decision sequence. |
| `Observation(id)` → `ObservationView` | Admitted envelope, consumer generation, producer principal, correlation, causal operation and proven producing run or reduction where available. `Origin` distinguishes input, message, terminal, lifecycle, adapter, receipt and invocation transports. `Admissions` links original transport attempts. |
| `Admission(id)` → `AdmissionView` | One attempted delivery and its admission or refusal. Multiple transport attempts can refer to the same logical input. An accepted invocation reply goes into the publication protocol. |
| `Reduction(ReductionRef)` → `ExecutionStep` | Exact consumer generation and sequence, source authority, instruction, selected observation and emitted operation IDs. `SelectedAt` is set by a recorded reduction. A matching receive or successful delivery alone cannot set it. |
| `Effect(operationID)` → `ExecutionEffect` | Original source, capability and name, historical authorization, reservation, dispatch bounds, first result, settlement, receipt evidence and controlled reconciliation links. |
| `Root(id)`, `Asset(id)`, `Operation(id)` | Original structured completion and envelope, retained source asset, or emitted operation. Each lookup returns a value and a presence flag. |
| `Pending`, `InvocationView.Dependencies` | Outstanding obligations with a run and the relevant original operation, observation or member. `DependsOnRun` identifies a logical member. `DependsOn` identifies the prerequisite operation, observation, root or terminal route for the given kind. |
| `History(offset, limit)` → `HistoryPage` | Up to 1,000 committed reductions in retained append order. Start at zero, advance by `Next` while `More` is true. `Total` counts the full snapshot. Invalid windows return `ErrLimit`. Global reduction order is distinct from the composition protocol's decision order. Ordinary admission has no timestamp that would order it against an unrelated later closure. |

The four lifecycle classifications have Go types `InvocationStage`, `EffectStage`, `RunLocalState` and `ExternalConsequence`. Use their public constants in a switch, for example `runtime.InvocationPublished`, `runtime.EffectReservedUnknown` and `runtime.ConsequenceUnresolved`. `Permissions` lists retained grants in insertion order. `Deployment` is the original static work specification in a composed store. `Reconciliation` is its optional read/adoption grant profile.

Logical identities belong to the snapshot's namespace. Operation IDs retain source subject, generation, sequence and ordinal. Invocation membership, publication and transport use their protocol-specific IDs. Payload equality, names, timestamps and token counts never establish an edge. Lookups and history windows share the same caller-owned snapshot. Treat the returned values as immutable if sharing them across goroutines.

## Lifecycle, dependencies and incomplete information

A run's `Projection.Status` is `open`, `succeeded`, `failed`, `abandoned` or `cancelled`. `ExecutionRun.LocalState` separates an open run, a `terminal_with_obligations` run, a `terminal_with_untracked_operations` run with an opaque emitted operation, and `terminal_settled` for the local protocols with retained evidence. `Subject` describes current subject-wide authority, which can have advanced beyond a historical generation. A terminal run can still list `UnresolvedConsequences`.

* **Input and delivery.** `Wait` names the receive kind and its provenance, producer, generation, obligation and operation selector. A dynamic selector with unreadable operands has `Resolved=false`. An open run at a non-receive instruction has no receive wait. `MatchingObservationIDs` lists matching, unconsumed envelopes. Scheduling, authority and actual selection are separate. Ordinary input can have an operator principal and no ALAN producer run. A legacy envelope without protocol provenance is marked `unrepresented`.
* **Invocation.** `committed` precedes the source decision. `authorized` and `source_refused` describe that decision before invocation admission. `admitted` and `refused` record the member-creation decision. `frozen` fixes the terminal member prefix. `closed` records a separate retirement decision. `published` has a retained original outcome and an observation for the caller. `consumed` has the specific parent reduction. The pointers to earlier facts remain available as later states appear.
* **Outstanding work.** An open receive may be waiting on ordinary input, a worker outcome or an effect result. `published_unconsumed` identifies an outcome delivered to a still-open parent without selection. A closed parent retains an unconsumed publication as historical evidence. Message and terminal deliveries awaiting a transport acknowledgment list their original `AdmissionIDs`, even when the recipient has already selected its observation. Unpublished invocations expose their own outstanding member, reply, descendant, result or publication. Root-local success can precede a publication blocked on an original admission, descendant publication or external result. `RootCompletion.Pending` follows Replay’s first blocker. An empty blocker with an absent outcome means eligible for local publication. The separate pending publication entry remains until it happens.
* **Known absence and applicability.** An accessor returning false means the supplied identity is absent in the cut. A nil `SelectedAt` on a retained observation means no consumption is recorded. A nil result on an authorized reservation leaves its outcome unresolved. Adapter-specific fields are inapplicable to a built-in external operation. Nil usage means unknown. Check the stage, kind and `Features` before interpreting optional fields. An impossible identity, broken causal reference or contradictory decision fails the entire inspection. No partial graph is returned.

## Effects, evidence and resources

| Effect state | Retained fact |
| --- | --- |
| `committed` | The computation emitted the original operation. No source decision exists. |
| `authorized`, `authorization_refused` | Original source authority has been decided. Inspect its run, epoch, sequence, revision and reason. |
| `reservation_refused` | Permission, scope or resource policy denied dispatch. The reservation carries the reason. |
| `reserved_unknown` | An opportunity was spent. The original result is absent. A crash can leave this state even if a provider received the request. |
| `result_unsettled` | The first result or receipt is retained. Delivery to the source remains outstanding. |
| `settled` | The original local disposition was settled to its observation. Follow that observation for consumption. A refusal of a built-in external reservation already determines its local outcome. |

`ResultDisposition` comes from the original protocol, including validated output, refusal, HTTP or transport error, or a built-in receipt. The annotated original result preserves adapter protocol, provider identity where supplied, endpoint, bounded response evidence and nullable provider counters. Interpreting a model's business proposition and accepting a citation remain application decisions.

`Consequence` separately records `no_dispatch`, `unresolved`, `receipt_retained`, `conflicting_receipts` or `read_only` for the controlled read protocol. A receipt is an assertion by the original adapter or ledger. An HTTP failure, cancellation, lost response or local dispatch drain cannot establish remote nonapplication. A positive read can establish receipt evidence while the local original remains missing. `ReadsOfOriginal` resolves to query effect IDs. Each `EffectRead` binds its ordinal, resolver authority, original authorization, optional snapshot and adoption decision. Adoption uses the retained grant and original key. Exact redeliveries coalesce. Distinct valid losing offers appear in `AdditionalReceipts`. Only the first result supplies the selectable observation.

The root reports separate limits and debits for invocations, created runs and effect reservations. Capacity and depth describe an envelope. `Dispatch` bounds **local executions** using retained reservation and result evidence. A single ordinary allowed attempt with a lost response is 0..1. The host may separately know that a provider accepted the POST. A nil maximum for an idempotent built-in operation denotes uncounted physical retries with its existing key. Receipt reads retain individual opportunities. These numbers do not count remote mutations. Provider usage is per original result, with independent optional input, output and total counters. Applications can preserve explicit priced estimates in typed data. ALAN has no versioned price ledger or universal monetary field. The investigator retains no price and reports an unknown cost.

## Storage, access and performance

| Storage | Representation |
| --- | --- |
| 3 and 6 | Ordinary provisioned runs, retry origins, operations, admissions, effects, settlements and consumption. No structured composition deployment, root expenditure or receipt-read protocol. |
| 4 and 7 | Complete public finite composition, root limits/debits, original admissions, bindings, committed decisions and publications. |
| 5 and 8 | Composition plus the controlled receipt-read grant, ordinal and adoption protocol. |
| 9 and 10 | Composition with opt-in local dispatch admission for scoped ordinary adapters. `Features.DispatchAdmissions` identifies the protocol. `ExecutionEffect.DispatchAdmission` records the gate after the named composition prefix. |
| 6, 7, 8 and 10 | Also retain distinct valid receipts rejected for settlement in an append-only evidence journal. Earlier formats have no independent journal for those offers. |

`StorageVersion` reports the physical number. `Features` reports which optional protocols the store can describe. Formats 1–2, the private foundation-only profile and unknown versions are unsupported. Inspection never promotes a store. An empty composition or read history is distinguishable from a format without that protocol. The conflict journal appears atomically with its first distinct rejected offer. An empty journal in a promoted store fails validation.

The reader opens an existing path with SQLite `mode=ro` and query-only mode. All ledger reads share one transaction and one WAL snapshot. It releases that transaction before building the semantic indexes in memory. It can run while another process owns the executor. Writer ownership, epochs and grants remain under the existing exclusive owner lock. SQLite locking, filesystem permissions or cancellation can make inspection fail. There is no fallback to a writable or stale immutable-file connection.

For a scoped ordinary adapter in format 9 or 10, a spent reservation with an absent gate has a physical execution bound of 0. Once the local gate admits the operation the bound becomes 0..1. Transmitting the request and receiving it at the provider are separate events. A retained adapter result can establish local execution. [Dispatch evidence and recovery](dispatch-admissions.md)

The database and durable WAL frames are unchanged by inspection. SQLite may create an empty `-wal` and initialize or update the transient `-shm` WAL index and reader lock marks. In tests, opening a bare checkpointed database created a zero-byte WAL and a 32,768-byte shared-memory file. Reading such a database from an unwritable directory without usable sidecars can fail. When archiving an interrupted store, capture a coherent database **and its WAL**. Concurrent executor commits can change those bytes independently of the reader.

There is one ordered bulk scan per supported table and a fixed set of catalogue, version and foreign-key checks. No edge issues SQL. Memory is proportional to the retained encoded records and materialized program/selection data. Building indexes decodes retained bytes and visits runs, observations, effects and history. Identity lists are sorted for deterministic traversal. Updating publication prerequisites walks at most the retained ownership depth per affected operation. Template and capability checks also examine their declarations. History returns an O(1) view into the detached slice, and enumerating that page costs O(returned steps). A small history page does not bound snapshot construction or memory. Large payloads and wide/deep templates increase cost. The diagnostic page covers reductions, while authorization and reconciliation use their own causal records.

A driver trace on the public composition format counted 41 ordered table reads and 14 connection, transaction, version and integrity statements on each of 201, 801 and 1,801 retained reductions. The largest case had 360 calls, 720 observations and 3,963 composition facts. Validating a single wide binding once lets all its member programs share the checked identities during construction. The allocation and latency benchmarks measure one process on one machine. Larger encoded ledgers can require substantial heap space.

When a cut contains a lifecycle decision, a lifecycle authorization or a missing transport target that a later invocation created, inspection checks the retained claims against the order established by the journals. It uses the retained append orders, known atomic records, target bindings, the original observation’s earliest delivery and its actual consumption. A successful owner reopen advances every then-open generation together. Epochs count activations within each generation, so different runs can reach epoch 2 on different reopens. Retained epoch witnesses constrain shared intervals where their lifetimes permit the comparison. The claims share one ordering check. Replay and Open construct the same constraints once per qualifying cut, alongside their independent reduction and protocol checks. For V relevant transaction records and E links, construction and the ordering pass take O(V + E) work. The reader groups records whose observed epochs fix their relative reopen counts and checks the remaining ordering constraints. With C groups and Q distinct ordered pairs, the solver uses at most O(C × Q) additional integer operations and O(V + E) auxiliary entries overall. Integer precision accommodates the sum of retained epoch distances. The reader never enumerates individual reopens or assigns a reported time to independent events.

The typed reader authenticates recorded causal joins and compares correlated operands, bound receive fields, original authority and dispositions. Replay additionally executes the deterministic reducer to verify the full policy transition history. Provider honesty and the authenticity of an entirely replaced database require separate evidence.

A lifecycle controller can address an invocation member before admission creates it. In that case `LifecycleDecision` returns the original `target_missing` with no target witness. The same snapshot can contain the later member. Inspection, Replay and owner reopen check those facts against the retained order of the lookup and creation. For example, an earlier admission already bound to the member can contradict a claimed absence. Neither elapsed wall time nor the final activation counters establish the order on their own. See the [worked recovery example and its limits](lifecycle-recovery.md#worked-example).

## Qualification

| History or negative control | Reproducible check |
| --- | --- |
| Completed two-worker case, a failed worker with both outcomes selected, one outstanding provider operation and a published sibling | `TestExternalExecutionInspector` builds a separate Go module, checks the output against provider contacts, reads while an owner is paused, kills it and reopens. |
| An unsuccessful original followed by a validated offer, duplicates, conflicting evidence, lookup without adoption | `TestInspectEffectRetainsFirstOutcomeAndLaterOffers`, `TestReceiptReadGrantCannotAdoptOriginal`, `TestIncompatiblePositiveReadKeepsAdoptedOriginal`. |
| Revoked authority, closed local work, outstanding reservations and lost read opportunities | `TestPhase2COriginalBindingControls`, `TestInspectReceiptReadRejectsReboundAuthority`, and the interrupted investigator above. |
| Real SIGKILL before/after receipt evidence commits across 3→6, 4→7 and 5→8 | `TestReceiptConflictCrash` checks the original operation and selectable outcome through reopening. |
| Coherent concurrent cuts, pagination and five historical SQLite fixtures | `TestExecutionInspectionDuringOwnerWrites`, `TestExecutionInspectionHistoricalSnapshots`. The latter compares logical generations, projections and counts to independent Replay and fingerprints the database and WAL. |
| Swapped sibling outcomes, wrong generations, missing envelopes, wrong consumer reductions, forged source authority, permissions, receipts and settlement | The `TestInspect` suites use independently damaged copies or individual disposable stores. They include changes rehashed to match the altered bytes. |
| Original app-specific interpretation | Investigator tests check selected findings against their original effects, registry/deployment approval, citation bytes, worker roles, gap reasons and the final verdict. |

```sh
go test -count=1 -race -tags integration -run 'Test(Inspect|ExecutionInspection|ReceiptConflictCrash)' ./internal/executor
go test -count=1 -run '^TestExternalExecutionInspector$' ./examples/investigator
go test -run '^$' -bench '^BenchmarkExecutionInspection$' -benchmem -benchtime=3x -count=3 ./internal/executor
```
