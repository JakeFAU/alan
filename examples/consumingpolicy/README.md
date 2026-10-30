# S-first consuming parent

`NewSFirst(candidate, config, Options{})` builds C_se: commit the original
candidate invocation then original S, await S then candidate, and only then apply
strict guarded success/presence/id/complement/S-zero checks. The retained fields
are `candidate_operation`, `s_operation`, and candidate/s status/reason/present/id/
value. Failure or absence never skips the other join. Pending receives no release.

Original A and B are imported through their public constructors. `substitution.New`
provides original S as `Peer`; `join.New` provides separately named S controls.
Only their returned templates are reused; those historical provisions never run.
The new context has one fixed owning parent, common I=R3/depth2/E0, capacity1.
Original A spends2 and original B+helper spends3 with S included.

This parent is a new derivation from the Phase 1M behavioral contract, authored
separately from the independent finite model. Explicit nested response guards
follow both joins. Public API calls, retained register names, resource-root setup,
and the reviewed public template access pattern follow Phase 1L. CLI dispatch,
store lifecycle, positive/raw-linkage test helpers and capture mechanics reuse
`examples/invocationjoin/consumer` and the Phase 1L preflight script; new S-first,
both-side failure, wrong-id and dual-pending assertions target this parent.
This is implementation-author independence from the new model author, not
independence of reused compiler/runtime, templates or infrastructure.

The A/B bound parent identities differ at candidate invoke target, candidate
receive source and author candidate template metadata; catalogue closures differ
because B retains its helper. The same policy, guards, original input and limits
apply to both bindings. Original component template bytes remain unchanged.

From the repository root, with Go1.27.3 installed:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1l-go-build
go -C examples/consumingpolicy/consumer build -o /private/tmp/alan-consuming-policy .
/private/tmp/alan-consuming-policy init -db /private/tmp/alan-s-first.db -policy s-first -candidate b -id 1 -value 0 -mode answer
/private/tmp/alan-consuming-policy run -db /private/tmp/alan-s-first.db
/private/tmp/alan-consuming-policy inspect -db /private/tmp/alan-s-first.db
/private/tmp/alan-consuming-policy replay -db /private/tmp/alan-s-first.db
```

Use a fresh database per ordinary execution. `recover` reopens original work.
`artifacts -policy s-first -candidate a` requires no database.
Controls select `-mode fail|absent|pending`, `-s-mode fail|absent|pending`, or
`-candidate wrong-id`. Capacity and dispatch/step limits match the unchanged
Phase 1L automatic runner: capacity1, MaxSteps200, MaxDispatch200. These two
construction samples and focused controls do not establish the independent
reference premise, all-input G qualification or arbitrary-context substitution.
