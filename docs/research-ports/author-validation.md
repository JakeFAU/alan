# Runtime author validation

The runtime ports the author admission correction from ALAN Research
`28cf2e28236026e3ea00fc43e0ee3367de7e7ae2`. The receiving baseline is
`fdac2f8453b8df08cc7ca0da92b0de907149b76f`, which includes the reviewed
runner at `d28d78bd`. Before this port, the receiving compiler matched the
research patch's compiler preimage after substituting the module import path.
The research manuscript describes the corrected admission checks in §2.4.

`Compile` checks every tracked ordinary string alternative against the
destination byte bound and enum. It joins branch alternatives and retains
correlations for supported unchanged guards. The default from an invalid
adapter reply contributes to the result's possible values until a validity
guard excludes it. Iteration accounts for the initial state, completed bodies
and loop-carried values, and discards predicates tied to prior rows.

The provenance pass checks the declaration on each `AwaitInvocation`. It
tracks original invocation identities through copies, collection fields and
iteration snapshots. It follows loop backedges and joins possible writes.
The existing reducer and Store enforce concrete schemas, request identities
and resource limits. The compiler emits the existing program semantics,
compiler metadata and instruction encoding.

## Receiving compatibility correction

The research flow pass omits an entire `ForEach` body when it can prove the
snapshot empty. A typed list copy and nested `ForEach` in that body still
need collection annotations in the encoded program. The witness compiled
at the receiving baseline to
`a1ca6f5c97121366fb2ebb1fdce7f7093af7e4d6b3ede6df2367a01d13a1b7d2`.
Both the initial port and research main at `4a4a7026` refused the same
definition with `invalid structured loop`.

The receiving compiler visits the lexical body for type checking and
collection annotations. Edges from a proven-empty body carry no facts into
executable joins. The invocation pass uses the same proven-empty snapshot
fact when deciding whether to enter the body. With the original provenance
pass, an unreachable append of an ordinary string caused a later
`AwaitInvocation` to refuse an otherwise authentic retained handle.
`TestInvocationHandleProvenanceIgnoresKnownEmptyLoopBody` accepts that definition
and requires the matched refusal when the list is an input that may have rows.
`TestCompileKnownEmptyLoopPreservesHistoricalArtifact` checks the full baseline
artifact ID.

## Qualification

| Probe | Receiving baseline | Port |
| --- | --- | --- |
| Partially overlapping enums, unconstrained strings flowing into enums, and excessive byte bounds including UTF-8 | Accepted the unsafe definitions. | Refuses at the constrained output or payload. Accepts contained schemas. |
| Invalid or mixed branch joins | Accepted invalid alternatives erased at joins. The finite enum test reached a reducer output-schema error for source `["a"]`, destination `["bb"]`. | Preserves joined alternatives. The finite enum control checks accepted schema pairs through actual reductions. |
| Adapter defaults and collection lifetime | Accepted an unchecked default at an enum output, stale row guards and unsafe last-row values. | Refuses at the destination. Accepts the guarded adapter result and valid correlated row and loop cases. |
| Forged, overwritten, mixed or wrong-declaration handles | Accepted ordinary strings at the invocation wait, including a later iteration after overwrite. | Refuses these at the same-declaration check. A copied handle in a retained list selects its original successful outcome and replays. |

The historical request, send, invocation, output and tampered send-metadata
tests use the prechange provisions from the research port. The nine schema
provisions were captured against research commit
`3d1d63fa6a69f590f145768745793d80f9839268` per the originating fixture
loader. The wrong-declaration provision is imported byte for byte from
`28cf2e28` (SHA-256
`b9cb5109d813f1d9818465208060c050194b87e09389bf1e71e1bb49cd14b116`).
It still reaches its original invocation wait after both child outcomes
are durable. The wait consumes neither unrelated result nor fabricated
response. The schema tests check the targeted runtime diagnostic,
no invalid committed intent, and replay where the fixture admits a Store.

The seeded differential test retains the historical adapter request from
the receiving baseline (program ID
`33612aa9cf01b1093343c008d2e5d0bf55609cd5f6a97c6f3b50abc071464078`).
With seed `20261027`, the compiler accepts 67 definitions and refuses 33.
The same retained request executes and replays successfully for eight
valid sampled inputs. Eleven out-of-enum sampled inputs reach the runtime
schema guard, commit no intent and replay as open.

The prechange outing walkthrough verified two completed fixture Stores and
the confirmation-first reciprocal wait. The port rebuilt deployments from
each original configuration and matched program IDs, program bytes,
semantic versions and all initial projection bytes. Inspection and replay
reports for copies of those old Stores matched byte for byte. The old
database files also matched their original bytes after both readers ran.

| Outing program | Bytes | SHA-256 and program ID |
| --- | ---: | --- |
| Outing | 29859 | `5cffe9d8d1e2235f03b6c8c23945196f6579dbd632479bac4077106483d8d516` |
| Weather, assess-first | 35074 | `d9c6d7c7ea715fcf0fe8e86b3ae0b40ece4e0a730f06236ffa5daef3f794bfbc` |
| Weather, confirm-first | 35074 | `82dda63983a442146a666b06d6c061bb6616a2382b6b5626d405f1c5802df009` |

All three artifacts use `alan/1g/v1`. Continued admission of the existing
example definitions required no source edits. The focused controls also
include the bounded series and historical and current paired handoff paths.

```sh
focus='^Test(Compile(|Immutable)(KnownEmptyLoop|String|Loop|Adapter|Collection|Correlated)|CompiledStringBounds|Invocation(Handle|Await|PublicRuntime)|AdapterAwait|Collection|NestedCollection|SerializedInvocationHandle|Phase2D|BoundedSeries|RuntimeValidatesInitialAndFinalSchemas|RuntimeEnforces(AuthoredSchemas|UnambiguousSendContracts)|RuntimeExecutesFalseRejectionPrograms|PR15Adversarial|HistoricalReplay)'
go test -tags integration -count=1 -timeout=60m -v -run "$focus" ./author ./runtime ./compose
go test -race -count=1 -timeout=60m -tags integration -v -run "$focus" ./author ./runtime ./compose
go test -count=1 -timeout=60m ./author ./runtime ./compose ./examples/outing-planner
staticcheck ./author ./runtime ./compose ./examples/outing-planner
staticcheck -tags integration ./author ./runtime ./compose ./examples/outing-planner
go build ./...
go test -count=1 -timeout=60m ./...
```

These results qualify tracked author flows, historical guards and the
specific retained fixtures. Conservative analysis can refuse definitions
when its available facts cannot establish containment or provenance.
Concrete schemas, actual identities, resource limits and eventual progress
remain runtime properties.
