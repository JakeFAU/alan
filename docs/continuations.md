# Authorized continuations

## Decision and frozen contract

Baseline `7b9ecccea2beaff0c6bf6825a5af98de4e88971f` has the same tree as weather delivery `478bb16bffe7422c1b61582fef1f8239caab1597`. The application makes the spending decision. A trusted local operator or controller can approve one separately metered observation with the same prepared action as a particular paid ordinary adapter request. Freshness, who operates the owner, acceptable reasons, and which operation to approve are application policy. Registering a truthful, bounded adapter remains a deployment obligation.

| Approach | Enforceable boundary and cost |
|---|---|
| Existing composition | A peer hands off an operation handle and the successor selects one local input. An ordinary owner can copy the input between namespaces, which repeat local handles. Two genuine stores each reported their first result. The second store accepted the first store's approval, dispatched a second GET and passed inspection, Replay and reopen. M4's CLI prevents that case with application-specific joins. No additional storage or reader cost. |
| Authoring helper | A macro can emit the same ports, checks and request. The reducer has no authenticated prepared-request digest or result history. A host wrapper can check an input but cannot protect raw Store execution. Two opposite orders of ordinary input admission and result retention even produced byte-identical pre-M5 databases. A helper alone would leave both trust obligations with the application. |
| Opt-in continuation slot | Reserve one local port on one original generation in a finite composition, admit its actual approval and an ordered composition fact in one transaction, and enforce its selected evidence before effect commitment and expenditure. Expose the relationship to all readers. One extra fact per approved slot, no extra table, no background work, and no migration of existing stores. |

### Causal boundaries

1. Provision a distinct scoped successor generation with at most one ordinary adapter operation, one finite resource allowance, a named protected local obligation and an explicit predecessor generation. The slot restricts the successor to the declared effect. The predecessor is identified by original operation identity and namespace. Equal payloads never establish that edge.
2. The operator supplies the expected predecessor evidence from a typed snapshot or the targeted owner reader and a bounded reason. An approval transaction rechecks the namespace, original operation and paid prepared reservation, local dispatch gate, retained result digest or its absence, and current target authority. It only admits the canonical input for that slot. The same transaction orders that input in the existing composition journal. The reason is an operator assertion. An earlier retained failure can coexist with uncertainty about remote application.
3. Commit order determines the facts at approval. The original reservation must precede it. A gate with an earlier composition prefix was admitted before it. A result belongs to that cut only if its own fact precedes the approval. A later result, selection or closure cannot change the approval's historical basis. A fact cannot split another atomic transaction. No wall clock is authoritative.
4. The named successor must select that exact observation before committing its own adapter operation. Its source, revision, effect, capability and typed parameters must match the approved single-use scope. At reservation its frozen provider configuration, wire representation and limits must also match. Only operation ID and namespace-bound request key are refreshed. Identity-bearing wire formats that change these other bytes are outside this first contract. Ordinary source grants, scope closure, accounting and final dispatch admission still apply. An approval alone spends nothing.
5. A repeated identical approval returns the original decision even after the successor advances. A conflict, raw local input on the reserved route, wrong namespace, subject, generation, predecessor, digest, action or stale authority fails closed. One selected approval can lead to at most one original adapter operation. A later owner neither invents approval nor replenishes an opportunity.
6. Inspection and Replay validate retained evidence with no adapter, provider call or policy callback. A fresh owner validates before renewing authority. A new result never settles, replaces or refunds its predecessor. Unspent original work, local settlement, reuse of a selected result, deterministic Replay and controlled receipt reconciliation retain their existing meanings. Remote application remains bounded by available external evidence.

### Qualification and stopping conditions

Exercise successful, refused, unspent, reserved without a gate, admitted without a result, and retained-but-unselected originals. Compare duplicate approval with a copied or reordered approval carrying matching final counters. Mutate namespace, subject and generation, logical correlation, prepared and result digests, parameter scope, missing original, fake completion, duplicate dispatch, authority and expenditure on disposable copies. A negative must fail before additional contact or debit. A forged retained relationship must fail inspection, Replay and reopen. A legitimate late result must remain valid. Exercise SIGKILL around approval, selection, successor commitment, reservation, gate, endpoint acceptance, result, settlement and terminal output, including repeated owners. Compare independent endpoint logs with the original identities and charges.

Existing formats 3–10 keep their old contracts. The live M4 fixture remains an offline regression. Stop at the smallest protected same-action, one-successor slot that meets these checks. Additional providers, general retries, reconciliation adoption, authentication services and arbitrary policy execution are outside this contract.

## Using a continuation

Declare one ordinary effect in each program. In the weather example, the first program starts at the request. A later program adds one local input before that same request. The application can bind values from the input if it wants to display them.

```go
ports["approval"] = author.ContinuationPort("approve-refresh", "operator_requested")
steps := []author.Step{
    author.Receive("approval", nil), // successor only
    author.Request("op", "forecast", arguments, nil),
    author.Await("forecast", "op", resultBindings),
    // Check the disposition and measurements before author.Success().
}
```

Provision an ordinary finite composition with dispatch admissions enabled. Give the successor its own one-member root, one effect opportunity, a grant for the same declared adapter, and a distinct local approval identity. Both source and successor must be static generation-one members. A deployment may declare several slots. Each target has one predecessor, and the declarations must be acyclic.

```go
profile.Continuations = []runtime.ContinuationSlot{{
    Predecessor: first, Target: next, Effect: "forecast",
    Obligation: "approve-refresh", ApprovalID: "weather-refresh-2",
}}
```

Open the original database and run normally after a process loss. If the original result exists, its settlement and selection need no new request. Inspecting the database before deciding to spend again needs neither ownership nor a registered adapter.

```go
cut, err := runtime.InspectExecution(ctx, path, limits)
if err != nil { return err }
basis, err := cut.ContinuationBasis(next)
if err != nil { return err }
original, _ := cut.Effect(basis.PredecessorID)
if !operatorWantsNewObservation(original) { return nil } // application policy

owner, err := runtime.Open(ctx, path, limits)
if err != nil { return err }
defer owner.Close()
if _, err := owner.ApproveContinuation(ctx, basis, "operator_requested"); err != nil {
    return err
}
_, err = owner.Run(ctx, options) // contains the configured adapter
return err
```

An application with an open owner can use `owner.ContinuationBasis(ctx, next)` to read the exact basis at a coherent owner cut. The read uses the declared predecessor generation and original operation keys. It needs no adapter registration, and it makes no admission, reservation or dispatch. A committed approval returns its original basis, including when a later gate or result has since been retained. Without an approval the read requires exactly one allowed paid operation for the declared effect. A retained refusal has a result digest. An empty digest means there was no retained first result at the represented cut, which is the owner read for an unapproved slot or the original decision for a committed approval.

The operator makes the spending decision separately. `ApproveContinuation` checks the submitted basis in its write transaction. If a gate or first result appeared between the read and the first approval, the earlier basis yields `ErrStale`. Repeating an identical committed approval still returns the original decision. For an independent audit of the full retained history and its causal relationships, use `runtime.InspectExecution` or `runtime.Replay`.

For deployment code, pass the original effect to the application's policy or display its fields. To resume existing work without approving anything, just open the same database and call `Run`. After execution, inspect again and follow `cut.Continuation(next)` to its approval and actual successor. Each `cut.Effect(id)` has its own reservation, gate and result. A nil selection or empty successor ID means that step has not happened. Repeating the same explicit approval returns the first local decision. A later observation can report a different provider time.

Only the holder of the local owner API may submit an approval. The application authenticates the operator and chooses which reasons to accept. The reason is retained as submitted. ALAN verifies the independently recorded predecessor evidence, selected approval and separately charged action. Machine identity, an authenticated human signature or remote deduplication is outside this single-host contract. The legacy weather CLI remains the trusted ingress for older multi-read stores.

Approval writes one existing local admission and observation, plus one ordered fact in format 11 or 12. Each public inspection reads a single database snapshot. Validation indexes protected slots and admissions, walks the existing composition order and checks a successor's original prepared bytes. Storage and additional validation work scale with retained input and action size and the number of slots, admissions and facts. An inspection lookup by target uses a binary search over approvals. The snapshot’s `ContinuationBasis` also examines the declared predecessor's effects when there is no prior approval. If several matching operations were paid, the caller must choose the original explicitly and supply its exact basis.

[Run, kill and reopen the example](../examples/weather-agent/README.md#kill-and-resume) · [Inspect the unchanged M4 live response offline](../examples/weather-agent/README.md#live-run-and-offline-reproduction)

The controlled comparison covers 18 predecessor and successor histories with 17 acknowledged SIGKILLs, 33 accepted localhost GETs and 36 paid opportunities. Both revisions made the same number of contacts and charges in each case. Each also used a separate two-read reference to locate its own instruction boundaries. The new weather program needs 31 committed reductions for two successful reads and 127 for eight. The M4 program needed 35 and 155 respectively. Its duplicate handle deliveries and comparisons are confined to the historical program loader. Neither version references storage tables or implements recovery in application code. [Per-case decisions, contacts and interruption records](../examples/weather-agent/testdata/continuation-process.jsonl)

To reproduce the timing and allocation measurements, run `go test -run '^$' -bench '^BenchmarkWeatherContinuation$' -benchmem -cpu=1 -count=7 -benchtime=12x ./examples/weather-agent` here and with the same benchmark file in the verified M4 checkout. Both sides use a local fixture and the archived live Store. First approval, duplicate approval, inspection and deterministic Replay are timed separately.
