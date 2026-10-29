# Invocation-owned join preflight

`New(candidate, config, options)` compiles one fixed parent. It commits the
candidate invocation and the original S invocation, then awaits the original
candidate handle and original S handle. Only after both awaits does ordinary
program control check successful, present, correlated results. The candidate
must return the request id and complement; S must preserve the id and return
zero. Status and presence checks guard response-field reads. A known negative
still joins S; pending work keeps the corresponding await pending.

The original S comes from `substitution.New(... Shape: "sequential").Peer`.
Only its template is reused. The old provision is discarded. Existing S outcome
controls come from `join.New(...).Peer`. No old parent or peer executes in this
new context. `config.Request` and `config.Limits` configure the new provision;
`config.Shape` is unused because the context always has this one parent.

The separate `consumer` module imports only public ALAN packages. Its `inspect`
command returns authentic retained records, including parent handle bindings,
original invocation intents/admissions/publications, outcome observations,
selected history, charges and root publication. `artifacts` exports public
original templates, the parent program, and provision for correspondence checks.
The consumer drives the unchanged automatic runner and never supplies a result
or makes the parent's decision.

From the repository root, use the pinned offline environment:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1l-go-build
go test -count=1 ./examples/invocationjoin
go -C examples/invocationjoin/consumer test -count=1 -tags integration ./...
go -C examples/invocationjoin/consumer test -count=1 -race -tags integration ./...
go -C examples/invocationjoin/consumer build -o /private/tmp/alan-invocationjoin .
/private/tmp/alan-invocationjoin artifacts -candidate a
/private/tmp/alan-invocationjoin init -db /private/tmp/invocationjoin-a.db -candidate a
/private/tmp/alan-invocationjoin run -db /private/tmp/invocationjoin-a.db
/private/tmp/alan-invocationjoin inspect -db /private/tmp/invocationjoin-a.db
/private/tmp/alan-invocationjoin replay -db /private/tmp/invocationjoin-a.db
```

Use fresh database paths and repeat with candidate `b`. Defaults are
id1/value0/answer, I=R3, depth2, effects0 and physical capacity1, with one fixed
parent. Expected dynamic I/R charges are 2 for A+S and 3 for B+helper+S. The
preflight samples test construction feasibility; they are neither an independent
oracle nor an all-schedule substitution qualification. See the Phase1L contract
and compact evidence under `docs/foundational/phase1l/`.
