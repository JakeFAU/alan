# Bounded adaptive composition

`NewScalar(leaf)` and `NewEvidence(leaf)` compile one ordinary durable adaptive
member. It admits a whole frontier, invokes all items before collecting original
handles, retains ordered public outcomes, derives children from those actual
outcomes, and commits one decision before the next admission. A three-token
`ForEach` reuses the same round body. No host callback builds a frontier.

Requests contain `seeds`, `max_rounds` (1..3), and `max_work` (1..8). Scalar seeds
are `{work_id,value}`; evidence seeds are `{work_id,question}`. Initial IDs are
1..8. Child IDs are `3*parent+1` (check/left) and `3*parent+2` (right). Scalar
observations 0/1/2 yield zero/check/left+right children with values 0/(1,0).
Evidence answer/no_match yield none, clarification yields check, and
insufficient_evidence yields left/right. Child questions copy the original.

The evidence profile accepts at most 512 raw UTF-8 bytes per question and binds
only the unchanged helper's disposition. Adaptive response rows contain the exact
scalar observation or evidence disposition in a zero-or-one `responses` list.
Full evidence answers and citations remain in the original authenticated outcome
and are inspectable by each row's operation ID. Missing responses stay absent.

Replies include `complete`, `stop_reason`, `rounds`, `work_count`, ordered
`results`, and the entire unperformed `remaining` frontier (up to sixteen rows).
The accepted frontier total is retained before its first item intent. Private
`history` is progress through the item-emission loop and can be a prefix during
emission; the full `current` frontier and `work_count` retain acceptance. History
is complete before the next candidate collision check. Runtime charges count
only actual admitted operations and reservations.
Known non-success precedes invalid observations, duplicate identities, exhaustion,
work bound, then round bound. Exhaustion succeeds; bounded stops succeed with
`complete:false`; invalid requests/results, duplicates, and leaf failures send a
diagnostic reply then fail. Each wrapper retains its own reason and maps child
success to Success and other statuses to Failure. It does not copy an arbitrary
child envelope status into its own envelope.

`New(component,nested,limits)` provisions one static root. `Submit` accepts the
complete request only after initialization. The fixed example envelope is
32 invocations / 64 created runs / depth 3 / 96 effects and physical capacity 1
or 2. Direct depth is root0 → adaptive1 → leaf2. Nested inserts a wrapper at1,
so adaptive is2 and both unchanged evidence-helper members are3. Rounds do not
add depth. Reopen existing stores to recover; constructors are for initialization.

The runnable consumer is a separate module using only public packages:

```sh
# Run from repository root. Use a fresh path for each init.
go -C examples/adaptive/consumer run . init --db /tmp/adaptive-direct.sqlite --capacity 1
cat >/tmp/adaptive-request.json <<'JSON'
{"seeds":[{"work_id":1,"value":0},{"work_id":2,"value":1},{"work_id":3,"value":2}],"max_rounds":3,"max_work":8}
JSON
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-direct.sqlite --id work --input /tmp/adaptive-request.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-direct.sqlite
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-direct.sqlite
go -C examples/adaptive/consumer run . replay --db /tmp/adaptive-direct.sqlite
go -C examples/adaptive/consumer run . recover --db /tmp/adaptive-direct.sqlite

# The same component, with one ordinary wrapper and capacity 2:
go -C examples/adaptive/consumer run . init --db /tmp/adaptive-nested.sqlite --nested --capacity 2
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-nested.sqlite --id work --input /tmp/adaptive-request.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-nested.sqlite
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-nested.sqlite
go -C examples/adaptive/consumer run . replay --db /tmp/adaptive-nested.sqlite
go -C examples/adaptive/consumer run . recover --db /tmp/adaptive-nested.sqlite
```

Default `--leaf scalar` is pure and needs no credential or network. `--leaf
controlled` uses a single bounded scalar HTTP request `{value:int}` and response
`{observation:int}` at an explicit `--controlled-endpoint`. `--leaf evidence`
uses the unchanged two-member evidence helper with its existing Responses and
lookup adapters; its controlled endpoint speaks the existing Responses fixture
protocol. Endpoints must be literal loopback HTTP, and redirects are refused.
No ambient OpenAI credential is read. No live provider mode is supplied.

Tests: `go test ./examples/adaptive -count=1 -timeout=30m`, focused transport
integration with `-tags=integration -run TestControlledEvidenceAdaptiveTransport`,
and `go -C examples/adaptive/consumer test -count=1 ./...`. The root module does
not discover tests in the consumer module. Every-transition byte measurements,
artifact/schema identities, source manifests, logs, and authoring friction are
retained under `docs/foundational/phase1h/evidence/task2`.
