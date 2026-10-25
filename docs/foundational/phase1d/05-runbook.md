# Phase 1D runnable examples

Use the repository's Go 1.27.3 toolchain, CGO and a C compiler on the qualified Unix host. The API and on-disk schema are provisional; no migration is supplied.

## Ordinary nested completion

From the repository root:

```sh
task_dir=$(mktemp -d)
GOTOOLCHAIN=local go build -buildvcs=false -o "$task_dir/alan" ./cmd/alan
"$task_dir/alan" init --db "$task_dir/normal.sqlite" --demo nested
"$task_dir/alan" run --db "$task_dir/normal.sqlite"
"$task_dir/alan" inspect --db "$task_dir/normal.sqlite" > "$task_dir/before.json"
"$task_dir/alan" replay --db "$task_dir/normal.sqlite"
"$task_dir/alan" inspect --db "$task_dir/normal.sqlite" > "$task_dir/after.json"
cmp "$task_dir/before.json" "$task_dir/after.json"
```

The default ordinary program sends failure input to worker generation 1, consumes its committed failure, spends one retry, consumes the retained accepted result, then sends success input to generation 2. The same worker artifact interprets both inputs. Outer consumes inner's public terminal outcome. The runtime scheduler has no special cases for those names.

The current example completes 26 execution commits and seven deliveries. Inspection exposes four retained runs, one lifecycle result and settlement, inner `spent=1`, inner `expected=2`, and three actual computation commits in worker generation 2. It retains the failed first worker run alongside the successful successor. These counts describe this fixture, not runtime limits.

`run` opens the existing store, validates retained artifacts/history/protocol records, renews active epochs, and automatically advances eligible work. It never initializes a new store or resets an obligation. `init` refuses to overwrite an existing store. Read-only `inspect` and `replay` do not renew epochs. A quiescent runner may still have waiting runs or unknown effects; inspect the report and retained outcomes before treating quiescence as completion.

## Ordinary exhaustion and escalation

```sh
"$task_dir/alan" init --db "$task_dir/exhausted.sqlite" --demo nested \
  --retry-budget 1 --successor-input fail
"$task_dir/alan" run --db "$task_dir/exhausted.sqlite"
"$task_dir/alan" inspect --db "$task_dir/exhausted.sqlite"
```

The ordinary inner program requests one successor, observes another failure, then reports failure because its budget is exhausted. Outer propagates that public failure with its declared zero-retry policy. A successful CLI exit means the runner completed its invocation; program failure is an inspectable outcome, not a command transport error.

## Inspection and operational bounds

Inspection includes subject/generation/epoch/revision, immutable program bytes and identity, residual program state, selected/consumed observations, intents and authorization, lifecycle requests/results, run creation/closure links, admissions and settlements, and pending terminal obligations. Data stored as JSON BLOBs is returned as exact UTF-8 JSON strings within table rows.

The runner defaults to 10,000 commits and 10,000 deliveries per invocation, configurable with `--max-steps` and `--max-dispatch`. Byte limits default separately to 1 MiB for program, projection, observation/admission and generated intent-array encoding. CLI flags are `--max-program-bytes`, `--max-data-bytes`, `--max-observation-bytes`, `--max-output-bytes`. Reopen/replay require limits sufficient for retained records. Limit errors do not become committed application failure or erase already retained work.

## Separate durable resource

Build both executables. Start the resource in a second terminal and leave it running while the executor stops and reopens:

```sh
GOTOOLCHAIN=local go build -buildvcs=false -o "$task_dir/alan-resource" ./cmd/alan-resource
"$task_dir/alan-resource" --db "$task_dir/resource.sqlite" --token demo-resource \
  --listen 127.0.0.1:8791
```

With the same `task_dir` in the first terminal:

```sh
"$task_dir/alan" init --db "$task_dir/external.sqlite" --demo external \
  --endpoint http://127.0.0.1:8791/apply --profile idempotent
"$task_dir/alan" run --db "$task_dir/external.sqlite"
"$task_dir/alan" inspect --db "$task_dir/external.sqlite"
"$task_dir/alan" replay --db "$task_dir/external.sqlite"
"$task_dir/alan" run --db "$task_dir/external.sqlite"
```

The client commits a counter mutation of five, waits for its correlated receipt, and completes. A separate ordinary peer progresses independently. Reopening uses the existing executor/resource pair and the original operation key. The standalone external example and nested example are separate fixtures; the model/runtime bridge additionally exercises external work across worker retry.

To try `--profile non-idempotent`, use **fresh resource and executor databases together**. Both built-in profiles use a canonical key derived from `client`, generation 1, sequence 1, ordinal 0. A new database filename does not create a new external-key namespace. Changing the profile against a previously used resource key conflicts with its immutable descriptor. Cross-store key namespaces are outside this profile.

## Real process-kill demonstrations

These tests build and launch the actual CLI and a separate resource executable. The independent controller waits for a pipe acknowledgment identifying the committed boundary, run, epoch, sequence and operation, then sends SIGKILL and verifies its exit signal. No timing sleep selects the crash boundary. `DURABLE` records include commands, binary hashes, process exits, barriers, complete provisioning and retained snapshots. The test prints its retained temporary artifact directory; the published evidence embeds those records so it does not depend on those temporary files remaining available.

```sh
GOTOOLCHAIN=local go test -count=1 -v ./internal/durabletest \
  -run '^TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash$'
GOTOOLCHAIN=local go test -count=1 -v ./internal/durabletest \
  -run '^TestRejectedResultCrashAndLostRejectionControl$'
GOTOOLCHAIN=local go test -count=1 -v ./internal/durabletest \
  -run '^TestAutomaticApplicationMissingReceiptProfiles$'
```

The first command covers accepted Retry before result consumption, authorization, terminal notification, and a second crash during recovery. The accepted case keeps inner generation 1, its obligation and spent budget, then actually runs worker generation 2. The second command retains a rejection after target eligibility changes and detects the deliberately faulty variant. The target change uses normal worker input and computation through the public store API while requester delivery is held; it is controller setup, not demonstrated supervisor policy. The third kills after external application but before receipt recording. Idempotent recovery returns the retained outcome; the non-idempotent profile remains unknown and is not automatically resent. Its separately labeled deliberate repetition experiment shows two mutations.

The controller also runs replay between inspections and checks that neither the live lifecycle ledger nor resource applications change. Replay alone cannot continue an obligation; `run` performs live recovery.

## Full qualification and offline reproduction

```sh
GOTOOLCHAIN=local go test -count=1 ./...
GOTOOLCHAIN=local go test -count=1 -race ./...
GOTOOLCHAIN=local go vet ./...
gofmt -l cmd internal research
GOTOOLCHAIN=local go test -count=1 -v ./research/semantics \
  -run '^(TestQualification|TestCompositionQualification|TestSupervisionQualification|TestRuntimeConformance.*)$'
```

Tests require local loopback sockets, subprocess execution and SIGKILL permission. Go's race detector requires a supported host and C compiler. Set a writable `GOCACHE` if the default cache is restricted. The qualified platform and exact commands are in [qualification](02-qualification.md).

The complete [source archive](source.tar.gz) includes source, tests, contracts, original research evidence, and the pinned SQLite module as a local Go proxy. It does not bundle the Go toolchain or C compiler. After extracting it into a new directory:

```sh
export GOTOOLCHAIN=local
export GOPROXY="file://$PWD/dependencies"
export GOSUMDB=off
export GOMODCACHE="$(mktemp -d)"
export GOCACHE="$(mktemp -d)"
go mod verify
go test -count=1 ./...
go test -count=1 -race ./...
go vet ./...
```

The first test command fetches the pinned module from that local proxy into the empty cache; `go.sum` checks it. Run `go mod verify` again afterward to verify the extracted dependency. `REPRODUCE.md` and the separate archive inventory describe the exact files. Original Phase 1/1B/1C archives under `historical-source/` reproduce their historical identities; running those tests against this implementation tree compares current behavior with a different source manifest.
