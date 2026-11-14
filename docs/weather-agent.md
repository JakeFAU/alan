# Author a durable weather reading

The [runnable weather agent](../examples/weather-agent/README.md) uses the
public `author`, `runtime`, `effects` and `adapters/openmeteo` packages. The
excerpts below use variables and helpers from that application.

## Request and select the response

The definition declares a `forecast` effect with the `open-meteo` capability
and the adapter's input and output schemas. The program passes latitude,
longitude and units. `Request` writes the original operation handle to
`operation_id`. `Await` uses that handle to select its correlated response.
The bindings include the validity flag, disposition and each typed output
field. [Authored program](../examples/weather-agent/program.go)

```go
steps = append(steps, author.Request("operation_id", effectName, map[string]author.Expr{
    openmeteo.Latitude: author.Var(openmeteo.Latitude), openmeteo.Longitude: author.Var(openmeteo.Longitude),
    openmeteo.Units: author.Var(openmeteo.Units),
}, nil))
bindings := map[string]string{schema.ResultValid: "valid", schema.ResultDisposition: "disposition"}
for field := range output {
    bindings[field] = field
}
steps = append(steps, author.Await(effectName, "operation_id", bindings),
    author.IfEqual(author.Var("valid"), author.Boolean(true), nil, []author.Step{author.Failure()}),
    author.IfEqual(author.Var("disposition"), author.String("validated"), nil, []author.Step{author.Failure()}),
)
```

Before succeeding, the definition also checks that the source, provider time,
grid coordinates, measurements and their units are present. A retained HTTP
response can carry a refusal or an invalid result. The program publishes
measurements only after selecting a validated observation.

## Provision one authorized opportunity

Run `start --allowance 1` for one observation. `configuration.series` passes
`Count: c.Allowance` (1 here) and `First: author.ApproveAtProvision` to
`author.BoundedSeries`. Its typed inputs retain the location, units, endpoint
and allowance. `series.Provision()` derives the original run, capability
grant and one-opportunity resource scope from the declaration. The host calls
`runtime.Create(ctx, *path, p, runtime.DefaultLimits())` with that provision
and closes the Store. [Provisioning source](../examples/weather-agent/program.go) · [Store creation](../examples/weather-agent/main.go)

An explicit larger count provisions one separately bounded position per read.
The first starts approved. A later position must consume its own protected
approval before issuing its request. The operator chooses `refresh --read N`
and the host records the decision through `Store.ApproveContinuation` on current
series. ALAN binds that decision to the exact predecessor evidence. The weather
policy also checks whether a retained predecessor result needs local processing.
`status` can show an eligible approval. The approval itself comes from
`refresh`. [Bounded series guide](bounded-series.md) · [Weather approval policy](../examples/weather-agent/approval.go)

## Reopen and drive the Store

The host validates the retained application before it supplies a transport. It
checks the exact deployment and original location, units, grants, scopes and
endpoint. `runtime.Open` acquires the Store for execution. The host checks the
snapshot again against the namespace and configuration it inspected before
opening. `openmeteo.New(c.Endpoint)` accepts the official HTTPS URL or the
fixture's literal HTTP loopback endpoint. The authored request has no endpoint
field. [Deployment checks](../examples/weather-agent/deployment.go) · [Owner path](../examples/weather-agent/main.go)

The runtime depends on the adapter being trusted host code. The adapter's
`Prepare` validates and freezes the GET without network I/O. The runner checks
authority and expenditure and records local admission before calling
`Execute`. The weather adapter binds execution to the prepared request.
[Open-Meteo adapter](../adapters/openmeteo/openmeteo.go)

```go
_, err = owner.RunProgress(ctx, runtime.RunOptions{
    Adapters: map[string]effects.Adapter{capability: adapter},
    MaxSteps: 512, MaxDispatch: 128, MaxInFlight: 1, ExternalTimeout: 30 * time.Second,
})
```

`MaxSteps`, `MaxDispatch` and `MaxInFlight` bound local steps, dispatch work
and concurrency for this `RunProgress` invocation. `ExternalTimeout` applies
to each external dispatch. The opportunity is a lifetime limit in the Store.
Opening again or calling `RunProgress` again leaves prior charges in place.
The application closes the owner and can reopen from a fresh process on the
same host, using working filesystem locks and SQLite WAL durability.
[Execution progress and reporting](execution-progress.md)

## Read the typed evidence

`runtime.InspectExecution(ctx, path, runtime.DefaultLimits())` returns a
validated snapshot without taking ownership. The weather host additionally
checks that the retained request matches its configured adapter and verifies
the native response evidence. `s.Run(readID(1))` gives the original read and
its projection. `s.Effect(run.EffectIDs[0])` follows the operation once the
read has issued its sole effect. The effect records `AdapterAttempt`,
`DispatchAdmission`, `AdapterResult` and `Settlement` as they become
available. When present, the settlement links to
`s.Observation(e.Settlement.ObservationID)`. Its `SelectedAt` identifies the
local reduction that consumed the observation. For a series,
`series.Inspect(s)` gives the ordered per-position evidence.
[Weather report source](../examples/weather-agent/report.go) · [Inspection contract](inspection.md)

```go
if e.Settlement != nil {
    r.ObservationID = e.Settlement.ObservationID
    o, _ := s.Observation(r.ObservationID)
    if o.SelectedAt == nil {
        r.State = "awaiting_selection"
    } else {
        r.SelectedAt, r.SelectedObservationID = o.SelectedAt, o.ID
        r.State = "observation_selected"
    }
}
```

A retained response can supply the original reading after a fresh owner finishes
local work. A paid reservation without dispatch admission consumed the
opportunity before transmission. If local dispatch was admitted but the
response is absent, the Store preserves uncertainty about the provider and
the operation keeps its original identity. A separately approved successor
obtains its own result and leaves that uncertainty in place.
[Dispatch evidence](dispatch-admissions.md)
