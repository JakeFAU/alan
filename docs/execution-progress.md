# Execution progress

`Store.RunProgress(ctx, options)` advances an owned computation with the same
`runtime.RunOptions` as `Store.Run`. Both methods use the same scheduler,
execution checks, and accounting. They share one active runner per Store.
When a call stops with workers in flight, the runner joins them and accounts
their results before returning. Keep the returned counts when handling an
error, since the call may have committed work before it stopped.

```go
progress, err := store.RunProgress(ctx, runtime.RunOptions{
    MaxSteps: 100, MaxDispatch: 20, Adapters: adapters,
})
```

- `Steps` counts reductions committed during this call.
- `Dispatches` counts dispatches charged against this call's dispatch limit.
  The count can include local delivery and settlement work. Use retained effect
  evidence to examine an individual reservation or dispatch admission.
- `Quiescent` means the runner found no supported eligible work at the
  observed boundary and finished accounting without an execution error. A
  quiescent store can have open runs waiting for input or unresolved effects.
  Quiescence carries no guarantee about completion, future work, or remote
  finality.

`RunProgress` skips the final report traversal. The runner still validates
the retained evidence it uses while executing. An unrelated malformed
historical admission may surface in full `Run` even after progress returned
successfully. `Create` and fresh `Open` retain their existing ledger checks.

Use `Store.Run` when the call should also return the full execution report,
including open runs, unresolved operations, and root completions. Use
`runtime.InspectExecution` to read a coherent, typed cut of retained state
and `runtime.Replay` to validate the retained ledger independently of
advancement. See the [typed inspection contract](inspection.md) for what the
snapshot records about dispatch and result evidence.
