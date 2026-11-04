# Bounded observation series qualification

## Source and scope

Verified base: `1d0ef3c91589c3df69b5a004cd5362438e8eed46`, tree
`c7266108c1e8913534781a066defe93f4057b8dc`. PR #6 was merged; its
rebased main tree matched the prior delivery. The contract was committed as
`32058f6` before implementation. Production changes are confined to public
`author` expansion/inspection and the weather example. Runtime, executor,
adapters, effects, schema and dependencies remain byte-identical to base.

The ordinary-writer evidence producer is
`8a5f2ef564d5d750afa506675431d414bf92fb65`, tree
`8caba38e10658672dc7f597751d322332f7d94bc`, 311-file content/name manifest
`ca95102a18437722cf61cce7dcf03ac7a98c2283266e193816ac4351c08ac048`.
It passed `go test -json -p=3 -count=1 -timeout=60m ./...`: exit0,
1,172 tests/subtests, no skips, 45.617 seconds. Source hashes matched before
and after. Delivery adds preserved evidence and documentation corrections;
final immutable delivery verification and exact head are recorded in the PR.
Earlier candidate tests and benchmarks retain their own source identities.

Environment: Go 1.27.3, darwin/arm64, Apple M4, CGO enabled, Apple clang 21;
SQLite dependency `github.com/mattn/go-sqlite3 v1.14.52`;
Staticcheck 2026.2.2 (0.8.2). Go tests/builds resolved modules offline with
`GOWORK=off GOPROXY=off GOTOOLCHAIN=local`. Exact commands, working directories,
cache paths, exits and raw output are retained with the evidence.

## Developer work

The previous application already used a loop, rather than eight copied blocks.
The comparison counts that actual implementation. Lines include braces; the
second number excludes blank and full-comment lines.

| Application responsibility at bound 8 | Before | After |
|---|---:|---:|
| Current provisioning, validation and error handling |39/37 lines|26/26 lines|
| Expansion block / series declaration |24/22 lines|5/5 lines|
| Subject/run identities constructed during provision |8|0|
| Per-position resource scopes constructed |8|0|
| Predecessor slots wired |7|0|
| Explicit Permission literals |0|0|
| Application-owned SQL queries |0|0|
| Public declaration/provision calls |Application.Provision plus manual expansion|BoundedSeries + Series.Provision|
| Controlled completed-series contacts / original operations |8/8|8/8|

Both outputs retain 8 runs, 8 roots, 7 protected slots, 15 compiler-derived grants
(8 adapter + 7 input) and at most 2 program artifacts. The new current path removes
manual position/root/slot wiring and the successor approval-receive prefix.
Exact old M4/M5 producers remain in a 100-line/89-code-line compatibility file;
this is not a claim that total application LOC decreased. No custom recovery
path, storage table, dependency or executor protocol was added.

The public surface is `BoundedSeries`, `SeriesOptions`, the explicit
`ApproveAtProvision` policy, `Series` with ID/Count/Run/Provision/Inspect, and
SeriesView/SeriesPosition/SeriesState. The existing Store continuation methods
continue to carry operator decisions. The application still owns result checks,
operator authentication, whether another read is desirable, and weather
formatting/native evidence verification. A series is a standalone finite
whole-deployment handle, not an invocable parent outcome or a parallel scheduler.

## Controlled histories

The [trace](../examples/weather-agent/testdata/series-process.jsonl),
[manifest](../examples/weather-agent/testdata/series.json) and
[original DB/WAL archive](../examples/weather-agent/testdata/series-stores.tar.gz)
retain 11 isolated histories: one full 8-observation control and 10 actual process
kills. There are 46 accepted loopback GETs and 48 paid opportunities across those
histories, not in one Store. No live Open-Meteo requests were made. The original
live fixture and both earlier process traces are byte-identical to base.

| Acknowledged cut | Result after fresh-owner recovery |
|---|---|
| Before fourth approval commit | Three completed; no fourth contact until explicit approval. |
| After fourth approval commit | Same original approval selected; one fourth operation. |
| After fourth approval selection, before request | Original selection retained; one fourth operation. |
| After fourth source authorization | Existing authorized original progresses once. |
| After fourth paid reservation, before gate | Four charges, three contacts; no refund; fourth remains spent before dispatch. |
| After fourth gate, before adapter call | Four charges, three actual contacts in this controlled schedule; retained runtime evidence remains uncertain. |
| Third accepted at endpoint, response withheld | Third stays unknown; repeated reopen makes no extra contact; explicit fourth approval permits a distinct result. |
| Third response retained, before settlement | Original response settles/selects with no additional third GET. |
| Third settled, before selection | Original observation is selected without a provider call. |
| Third terminal reduction, before root publication | Selected local result becomes completed after remaining local settlement. |

Each instrumented boundary uses an inherited-pipe ACK bound to the Store,
process, subject, generation, sequence and relevant operation. The provider
acceptance cut uses its independent accepted-GET signal and an uninstrumented
public executable. Controllers require actual SIGKILL/reap. Sleeps do not
establish cuts. Ordinary recoveries are fresh processes from an unrelated module
using public packages. All fifth-through-eighth work remains unapproved until
its own decision. Retained original responses/approvals and prior effect records
are compared across recovery; duplicate approval never adds expenditure.

Nine disposable-copy attacks alter position identity, generation, predecessor,
approval target, source grant, root allowance, selected approval, selected result
sequence or observation operation. Inspect, InspectExecution, Replay and Open
reject them. Failed Open and both readers preserve tested DB/WAL bytes. A
separately valid foreign deployment is rejected by declaration matching and by
the weather legacy fallback. The immutable positive writer is never SQL-edited.

Read-only checks compare main DB and WAL bytes and provider contact logs.
The campaign treats an absent WAL and an empty WAL as the same durable content.
SHM/lock auxiliary files are not durable evidence and are not given a universal
byte-invariance guarantee. Extract archived DB and matching WAL together to a
private directory and use `inspect`/`replay`; their ephemeral loopback endpoint
is no longer a running service. No remote exactly-once or cryptographic
protection against a coherently rewritten entire Store is claimed.

## Compatibility and measured limits

No format migration: count 1 uses existing format 9; protected series use 11;
conflicting receipt retention may promote 11 to 12. Generic readers from verified
base successfully inspect/replay/reopen new genuine 11/12 stores. Historical real
format 4 lifecycle archives and the live format 9 weather Store remain readable
and unchanged. Older formats cannot acquire a protected continuation protocol
that they never retained; their original contracts remain supported.

The independent public-package driver uses a deterministic in-process adapter,
with no HTTP. Final producer measurements below are empirical samples on the
shared development host. Construction is the median of five samples; execution
is one sample including a fresh snapshot/basis/approval for every successor.

| Bound | Series + Provision ms | Full execute ms | Final typed Inspect ms | Replay ms |
|---:|---:|---:|---:|---:|
|1|0.130|3.98|1.38|2.60|
|8|0.542|129.96|5.14|18.88|
|32|1.651|2,151.85|17.42|76.14|
|256|11.839|364,274.95|135.05|867.07|

The 256-position run allocated **406.697 GB cumulatively**, not resident memory.
The baseline manual expansion took 368,984.42 ms and allocated 404.097 GB. These
samples do not establish a speedup or regression. An independent canonical
manual expansion with identical names produces byte-identical entire Provision
JSON at all four bounds; the generic executor is unchanged. The inherited runner
repeatedly scans runs, observations and roots, and approval obtains a fresh
validated full-store snapshot. This cost merits a separate runtime performance
slice; accepting 256 is not a low-latency claim.

The additional Series.Inspect view performs no SQL. Provisioned-view medians at
1/8/32/256 are 1.65/20.45/93.89/732.45 microseconds, allocating
1,304/26,834/138,401/1,101,467 bytes. The completed-eight view median is
26.46 microseconds and 30,288 bytes. See the [full report, allocation tables,
controls and raw logs](evidence/bounded-series/performance/REPORT.md).


## Independent review and corrections

Separate reviewers inspected public usability, causal composition, authority and
accounting, process recovery, and inspection/Replay/compatibility/performance.
Their [verbatim reports](evidence/bounded-series/reviews/) retain source identities
and dispositions; the performance review is [with its experiments](evidence/bounded-series/performance/REVIEW.md).
Final whole-delivery review and the final exact tree are recorded in the PR;
producer/task approvals are not relabeled as that final review.

| Substantive finding | Correction and discriminating evidence |
|---|---|
| Failed after request, before selection, appeared pending. | Preserve terminal local failure independently of an unselected result or unknown external consequence. Genuine known/unknown results before and after reopen pass. Initial REJECT then APPROVE retained. |
| Long or escaped series name allowed a paid first read but made the next approval unrepresentable. | Reject before provisioning using the actual JSON-encoded predecessor and existing 4,096-byte field. Exact-boundary ASCII/control/HTML/U+2028 histories execute both reads; overflow, sequence and index-width cases reject. Independent external consumer re-review APPROVE. |
| Guide attributed operator authentication to the Store. | Assign authentication and spending policy to the application; the Store enforces retained scoped authority. Independent guide re-review APPROVE. |
| Guide reused a pre-run snapshot for post-run inspection. | Capture a new snapshot. Independent experiment distinguishes stale `Completed: 1` from fresh `Completed: 2`; guide re-review APPROVE. |

[RED outputs and correction patches](evidence/bounded-series/red/) preserve the
rejected rules. Two new process-test expectations were also corrected against
actual cuts: a reservation without a gate is spent but not externally unknown;
a terminal reduction before root publication is selected but not yet locally
completed. No production semantics or prior milestone assertion was weakened.


The existing ordinary Staticcheck diagnostics are ST1005 at
`adapters/openai/openai.go:49,59` and U1000 at `author/race_off_test.go:5`.
The integration configuration instead includes the same ST1005 and SA4006 at
`runtime/runtime_test.go:1117,1126`. These were reproduced on unchanged base;
new author/weather code has no introduced finding. The prior intermittent
`TestCompetingReadCannotSettleAdopterDebt/adopter_query` evidence remains
preserved. This integration-tagged case is absent from the producer ordinary
suite; its final integration outcome is recorded in the PR. Any failure must be
checked against base rather than dismissed as known flakiness.

For reproduction, use the public guide and the test commands in the PR. The
new process campaign alone is:

```sh
go test -count=1 -tags integration -run '^TestWeatherSeriesProcess$' -v ./examples/weather-agent
```

Set `ALAN_SERIES_EVIDENCE_DIR` to a new private directory to retain its ordinary
writer DB/WAL pairs. Benchmark reproduction uses the independent public-package
driver in [the performance evidence](evidence/bounded-series/performance/README.md).
