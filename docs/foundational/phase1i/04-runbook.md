# Phase 1I runbook

Use the complete source checkout, Go 1.27.3 and pinned
`github.com/mattn/go-sqlite3 v1.14.52` with CGO. Recorded platform: darwin/arm64,
macOS 27.2. Local fixtures require no provider credential or paid call. Process
tests use loopback and SIGKILL; the macOS replay check uses `sandbox-exec`.
With dependencies already cached, run from repository root:

```sh
unset OPENAI_API_KEY
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE="${TMPDIR:-/tmp}/alan-phase1i-go-build"
ALAN_MODEL_EVIDENCE="$(mktemp -d /tmp/alan-i-model.XXXXXX)" \
  go test -v -count=1 -timeout=10m ./research/substitution
go test -tags integration -count=1 -timeout=10m ./examples/substitution/...
```

A new machine must first obtain the declared toolchain and pinned module.
Offline mode fails rather than substituting another version. Model tests expect
the preserved parallel counterexamples: green means the failures were detected,
not that the disproved completion guarantee became true. They run original and
boundary-extension products, local relations, probes and separate K1E automata.

## Public-consumer CLI

```sh
cd examples/substitution/consumer
go test -count=1 ./...
go build -o /tmp/alan-substitute .
demo_dir="$(mktemp -d /tmp/alan-i-demo.XXXXXX)"
/tmp/alan-substitute init -db "$demo_dir/run.db" -candidate b -context wrapped -id 1 -value 0 -mode answer -i 3 -r 3 -depth 3
/tmp/alan-substitute run -db "$demo_dir/run.db"
/tmp/alan-substitute inspect -db "$demo_dir/run.db"
/tmp/alan-substitute replay -db "$demo_dir/run.db"
/tmp/alan-substitute recover -db "$demo_dir/run.db"
```

Use a fresh database per deployment. Candidate `a` is local, `b` uses one helper.
Contexts: `bare`, `sequential`, `parallel`, `hidden`, `wrapped`. Modes: `answer`,
`fail`, `absent`, `pending`. Public request domain: ID1..2/value0..1; unknown modes
or out-of-profile requests are rejected. `recover` opens the retained artifact
and runs eligible work; `replay` verifies history without dispatch. A completed
store stays completed. The following subprocess cases supply actual crash evidence.

## Decisive runtime and process witnesses

Return to repository root:

```sh
ALAN_I_ARTIFACTS="$(mktemp -d /tmp/alan-i-runtime.XXXXXX)" \
  go test -tags integration -json -count=1 -timeout=10m \
  -run '^(TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence|Decoder.*)|TestReviewConsumedPayloadIdentityPreserved)$' \
  ./internal/substitutiontest
ALAN_I_ARTIFACTS="$(mktemp -d /tmp/alan-i-race.XXXXXX)" \
  go test -race -tags integration -json -count=1 -timeout=30m \
  -run '^(TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence|Decoder.*)|TestReviewConsumedPayloadIdentityPreserved)$' \
  ./internal/substitutiontest
```

The final selector executes 295 leaves: matrix190, probes26, pure recovery4,
closure9, effects5, root-success counterexamples2 and decoder controls59.
The archived normal-v1/race-v1 closure files are intentional test inputs retained
in this checkout. Use fresh evidence directories:
exclusive record creation prevents overwriting earlier results. The Go JSON log
maps test names to gzip inspections, acknowledgments, process exits, original and
normalized identities and checker results. For the smallest concrete negative:

```sh
go test -tags integration -v -count=1 \
  -run '^TestSubstitutionRootSuccessFence$' ./internal/substitutiontest
```

The ordinary delayed-peer fixture expects root local success and peer `scope_closed`
failure at I=R3/depth2. It requires original-model rejection and boundary-model
membership with the failed completion check still retained.
Its eventual aggregate root outcome is failed/member_failed, not composite success.

Run the full focused harness package to include decoder regressions added during
review; the internal process-child entrypoint skips outside a controlled child:

```sh
go test -tags integration -count=1 -timeout=10m ./internal/substitutiontest
go test -race -tags integration -count=1 -timeout=30m ./internal/substitutiontest
go vet -tags integration ./research/substitution ./examples/substitution/... ./internal/substitutiontest
go mod verify
```

[Qualification](02-qualification.md) and [result index](qualification.jsonl)
attribute each actual command to source identity and selected cases. Prior phase
inputs/evidence remain historical. Runtime/replay share a reducer; agreement is
historical consistency, not independent semantic validation.

The unknown-effect case kills only after the endpoint acknowledges application
and before receipt recording. Reopening keeps the original unknown obligation
and spent reservation without retransmission. The live-late-transport case is
separate. These tests cover process crashes, not power loss, storage corruption,
distributed partitions, arbitrary contexts or universal liveness.
