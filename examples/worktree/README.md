# Retained public collections

`compose.New` derives one bounded scatter/collect template from any retained leaf
with a public request/result port. It uses ordinary `Invoke`, `Append`, `ForEach`
and `AwaitInvocation` steps. `worktree.New` installs one static intake/root run;
`--nested` adds an ordinary wrapper to the retained catalogue. No item programs
or subjects are provisioned before work arrives.

From the repository root, use the pinned local Go toolchain and cached modules:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build
go run ./cmd/alan-compose init --db /tmp/collection.sqlite --capacity 2 --nested
# Initialization is complete before this work input is created.
cat > /tmp/work.json <<'JSON'
[{"item_id":"item-a","number":2},{"item_id":"item-b","number":-3},{"item_id":"item-c","number":2}]
JSON
go run ./cmd/alan-compose submit --db /tmp/collection.sqlite --id one-work --input /tmp/work.json
go run ./cmd/alan-compose run --db /tmp/collection.sqlite
go run ./cmd/alan-compose inspect --db /tmp/collection.sqlite
go run ./cmd/alan-compose replay --db /tmp/collection.sqlite
# On a later process activation, recover uses the existing store and artifacts.
go run ./cmd/alan-compose recover --db /tmp/collection.sqlite
```

Use a new database path for each new root obligation. `submit` (`admit` is an
alias) only admits external input. The later collection **Receive commit** is
its application acceptance boundary. Input is an ordered list of at most eight
records with distinct `item_id` values. Repeating a transport ID retransmits that
same input; it does not create another obligation. `[]` returns an empty successful
collection. A ninth record or duplicate ID is rejected before any child work.
The example's results are `item-a=4`, `item-b=-6`, `item-c=4`, in that order.

The public collection response contains `all_succeeded` and `results`. Each
result retains `item_id`, opaque `operation`, `status`, `reason` and `responses`.
`responses` is a typed list of zero or one leaf response: absence has no fabricated
scalar defaults, including for enum-constrained fields. All item handles are
retained before collection begins; results are consumed in input order. A known
item failure/rejection produces a failed aggregate with ordered partial results.
An unknown item remains pending. Arithmetic overflow is an explicit Run error.

`inspect` exposes the root's own projection as `Result` and the coherent public
runtime snapshot as `Snapshot`. `Snapshot.Roots` and Run/Replay `Roots` distinguish
local terminal status from structured public completion, cumulative charges and
unresolved/unknown operations. An exit0 from Run means the runner returned
normally; use `Roots[].Outcome` to determine public completion. Root outcomes
have no caller reply port; the collection response is in the root projection.
Inspection does not open/activate, rebuild, plan or consult provider state.

For the existing evidence helper, initialize with `--leaf evidence`. Submit
`[{"item_id":"question-a","question":"WAL?"}]`. Its unchanged two-program policy
uses the existing Responses and lookup adapters, with a retained nonsensitive
one-line controlled corpus. Run/recover accepts
`--controlled-endpoint http://127.0.0.1:PORT` for a controlled Responses server;
only literal loopback HTTP endpoints are accepted. No credentials, environment
provider configuration or live provider endpoint is used. Both model stages and
lookup retain their usual identities and shared root accounting.

The public Go deployment API is `worktree.Deterministic` or
`worktree.Evidence(evidence.Budgets{MaxModelRequests:2, MaxLookupRequests:1})`,
then `worktree.New(leaf, nested, worktree.DefaultLimits(capacity))`. Create the
store from `Deployment.Provision`, submit with `worktree.Submit(ctx, store, id,
schema.List(...))`, and execute with `store.Run`. Recovery uses `runtime.Open`
and `store.Run` directly. `author.Template.Schemas()` returns copied retained
public schemas; consumers never decode template bytes into executor structures.

The separate module deliberately imports only public packages. Root `go test
./...` does not traverse it. Explicit commands are:

```sh
go test ./... -count=1
go test -tags=integration ./examples/worktree -count=1 -timeout=100s
go -C examples/worktree/consumer test ./... -count=1
go -C examples/worktree/consumer vet ./...
go -C examples/worktree/consumer test -race ./... -count=1 -timeout=240s
```

See [Task3 evidence](../../docs/foundational/phase1g/evidence/task3/report.md) for
actual exits, source identities, causal barriers, counts and limits. Process
SIGKILL schedules and final full-tree qualification belong to Task4.
