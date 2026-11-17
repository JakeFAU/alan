# Rejected invocation inspection

This runtime ports the pending-invocation predicate and regression from ALAN
Research commit `3d1d63fa6a69f590f145768745793d80f9839268`, based on runtime
commit `40df2793f20f14bff72aa3f7b2ba3ccf4efbd605`. The port draws from
`internal/executor/composition_inspect.go` and
`internal/executor/composition_pending_regression_test.go`.

Composition-v1 settlement awaits a recorded admission decision and checks
descendant outcomes for accepted invocations. The indexed root report now
loads each invocation's admission by operation ID. A missing decision or an
accepted admission still requires the publication check. A rejected admission
contributes no pending invocation to the root. Other admission load errors
propagate. The detached snapshot applies the same rule using its retained
admission view. Its separate caller-side pending entries still show waits for
delivery and selection, together with the original intent, refusal, and
available outcome evidence.

The change is limited to these two pending calculations. Root grouping,
ordering, resource accounting, dispatch and settlement continue to use the
existing version-1 rules.

## Qualification

The regression uses the receiving runtime's two-run template and a
one-created-run allowance to produce a genuine `created_run_limit` refusal.
On the base, the live and detached reports listed both operations at the
mixed cut. The first had a recorded refusal and only the second awaited
admission. After publishing root success with both children refused and
unpublished, replay, raw inspection, and the reopened store still listed
both. The base also listed a refused call as root debt while its caller
awaited delivery, and the live report returned a result from a disposable
store with a malformed admission descriptor. The accepted-unpublished
control already passed.

All four focused tests pass on the port. The caller test checks the
unpublished refusal, its delivered but unconsumed observation, and the
recorded selection. At each cut the refusal remains visible by its original
operation ID. The malformed descriptor now returns `ErrCorrupt` through
the root report.

```sh
go test -count=1 -v ./internal/executor -run '^TestRootPendingInvocations'
go test -race -count=1 -v ./internal/executor -run '^TestRootPendingInvocations'
go test -count=1 -timeout=60m ./internal/executor ./runtime
go test -tags=integration -count=1 -timeout=60m ./internal/executor \
  -run '^(TestRootCompletions|TestFinishRunReport|TestReportReaderCached)'
staticcheck ./internal/executor ./runtime
go build ./...
```

To reproduce the failures with the final tests, restore only
`internal/executor/composition_inspect.go` and
`internal/executor/inspection_pending.go` to the receiving base in a
disposable checkout or Go overlay, then run the first command. These finite
single-host checks qualify the diagnostic on the tested composition-v1 cuts.
