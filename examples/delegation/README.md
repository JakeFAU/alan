# Finite reciprocal request binding

`Reciprocal` preserves its v1 immediate-predecessor policy. `RequestBinding` opts a newly provisioned Store into `runtime.RequestAuthorityBindingVersion` (`alan/request-authority/v2`): an accepted grant identifies one original request and one exact service operation. The original request must have been genuinely selected earlier in the same service generation. Ready messages and permitted scratch reductions can intervene.

```go
provision, manifest, err := delegation.RequestBinding(delegation.RequestBindingOptions{
    EndpointA: "http://127.0.0.1:1234/apply",
    EndpointB: "http://127.0.0.1:1235/apply",
    LocalSteps: 3,
    TerminalHandshake: true,
})
```

The caller supplies two distinct literal loopback HTTP endpoints implementing the existing resource protocol. Pass `provision` to `runtime.Create`; `manifest.ProgramIDs` and the exact original/service/reply locators describe that provision. They are planned coordinates, not evidence that any effect occurred. Inspect actual effects, retained results and their consumption after running.

By default A selects the original request, then ready. `ReadyFirst` reverses these two receives. `LocalSteps` accepts 0 through 32 and alternates literal writes/copies in a scratch register. A non-nil `ForwardBranch` requires `LocalSteps=0`; either finite branch executes three reductions over five code positions. The service and planned reply are A(4+n) and A(7+n), where n is the number of executed inserted reductions, not textual PC displacement. B remains at B9/B12. The constructor updates every claim, affected branch target, authority version and program hash before provisioning. Principal request IDs remain AB/A1 and BA/B1.

The v2 physical formats are 15/16, with a non-unique observation admission index. V1 formats 13/14 retain the original policy; there is no in-place migration. Receipt-conflict promotion preserves the selected authority policy. The fixed Store metadata semantics do not restrict supported authored artifacts to raw programs; the existing raw, authored and collection validation rules still apply. This example itself uses raw `alan/1d/v1` programs.

Historical selection is provenance, not fresh permission. Current gates still require the original allowed operation, reservation/debit, current source/root policy, owner epoch and unrevoked selected grant. A future selection cannot repair an earlier refusal. Admission alone, a different equal-payload request, a result, or an application register cannot substitute for the exact original selection. Retained known results preserve their meaning after revocation; they do not authorize another contact.

Focused checks:

```sh
go test -count=1 ./examples/delegation -run TestRequestBinding
go test -count=1 ./internal/executor -run TestRequestBinding
go test -count=1 -tags integration ./internal/executor ./internal/durabletest -run 'TestRequestBinding|TestPhase3B4AuthoredAdapterCompatibility|TestPhase3B3CartesianOperations'
```

The automatic example tests exercise adjacent/nonadjacent receive order, 1/3 local steps and both branch outcomes with two actual endpoint applications and retained-result consumption. These bounded tests and shared runtime Replay checks are not an independent policy proof, a general dynamic-locator construction, or a whole-Store crash qualification. The independent campaign, adverse cut corpus and cost measurements belong to the research qualification.
