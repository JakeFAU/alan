# Phase 1J reproduction

Use Go1.27.3, CGO and pinned go-sqlite3 v1.14.52 on a local Unix filesystem.
Recorded platform: darwin/arm64. No provider credential, paid endpoint or new
module dependency is required. From repository root, with dependencies cached:

```sh
unset OPENAI_API_KEY
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE="${TMPDIR:-/tmp}/alan-phase1j-go-build"
j_evidence=$(mktemp -d /tmp/alan-phase1j.XXXXXX)
ALAN_JOIN_MODEL_EVIDENCE="$j_evidence/model" \
  go test -v -count=1 -timeout=10m ./research/join
go test -tags integration -count=1 ./examples/join
ALAN_I_ARTIFACTS="$j_evidence/runtime" \
  go test -tags integration -v -count=1 -timeout=10m \
  -run '^TestJoin(Runtime|OriginalContext|Decoder|DuplicateDelivery|Recovery|Cancellation)$' \
  ./internal/substitutiontest
ALAN_I_ARTIFACTS="$j_evidence/race" \
  go test -race -tags integration -v -count=1 -timeout=30m \
  -run '^TestJoin(Runtime|OriginalContext|Decoder|DuplicateDelivery|Recovery|Cancellation)$' \
  ./internal/substitutiontest
```

Model evidence paths must be absolute: Go runs package tests in their package
working directories. Use new output directories. The runtime harness reuses the
existing `ALAN_I_ARTIFACTS` evidence writer; these are new Phase1J cases, not old
Phase1I results. `TestJoinProcessChild` is a controlled subprocess entrypoint,
not a separately counted experiment. `TestJoinArtifactExport` intentionally skips
without an explicit output destination.

A passing model gate requires the new source-authorization counterexamples to
remain visible in all eight configurations of each graph. Green does not mean
all-schedule completion. The graph retains normal terminal actions, first source
authorization and refusal; its finite stutter abstraction is documented separately
from concrete `Store.Run` schedules. Census JSON records states, edges, maximal
states/executions, rank and witnesses. The model's preserved RED completion assertion can be reproduced with
`python3 docs/foundational/phase1j/evidence/model/reproduce-red.py`; it reconstructs
a temporary Go overlay and requires the false completion assertion to fail.

The runtime selector covers the original C0 failure at capacities3/4, joined
samples and adverse controls, four actual ACK/SIGKILL/fresh-process recoveries,
explicit cancellation, raw adapter mutations and duplicate delivery. Inspect the
retained intermediate states, original consumed evidence, charges and aggregate
outcomes; command success alone does not establish those facts. Process tests use
actual SIGKILL and existing test-only boundary observers; watchdogs are limits,
not termination evidence.

## Public consumer

```sh
cd examples/join/consumer
go test -count=1 ./...
go test -race -count=1 ./...
go build -o /tmp/alan-join .
join_demo=$(mktemp -d /tmp/alan-join-demo.XXXXXX)
/tmp/alan-join init -db "$join_demo/run.db" -candidate b -id 1 -value 0 -mode answer -i 3 -r 3 -depth 2
/tmp/alan-join run -db "$join_demo/run.db"
/tmp/alan-join inspect -db "$join_demo/run.db"
/tmp/alan-join recover -db "$join_demo/run.db"
/tmp/alan-join replay -db "$join_demo/run.db"
```

Use a fresh DB per deployment. Controls: `-s-mode fail|absent|pending`,
`-wrong-request`, `-premature-report`, and candidate `-mode fail|absent|pending`.
A premature peer assertion can fool the root; rejection belongs to the independent
causal checker, not a privileged root inspection API. Wrong-request parks in the
single-report profile. Recovery opens retained artifacts without rebuilding work.

## Identities and focused gates

`evidence/context/artifacts/sha256.json` binds archived raw exports. Program and
template bytes are stable; public Provision program catalogue order can vary.
For re-export comparison, compare programs by ID (or sort only the Programs list
by ID), retaining instruction-array order. Do not require raw Provision JSON
byte equality or overwrite historical exports. The independent context review
records this distinction.

From repository root:

```sh
go test -count=1 -timeout=10m ./research/substitution
go test -count=1 -timeout=10m ./author ./schema ./research/semantics
go test -count=1 -timeout=10m \
  -run '^(TestInvocationRootClosureSerializesWithAdmission|TestInvocationReplyExceptionDoesNotAuthorizeOtherEffects|TestInvocationFailureFreezeIsAtomicAndIgnoresLateReply|TestInvocationSuccessWaitsForCommittedUnadmittedChild|TestReplayRejectsChangedSelectedObservation)$' \
  ./internal/executor
go vet -tags integration ./examples/join/... ./research/join ./internal/substitutiontest
go mod verify
git diff --check
```

The phase ends with one `tectonic alan-dissertation.tex` invocation from `paper/`,
after manuscript reconciliation and review. Reproducing pure tests does not require
regenerating the PDF. Warnings are accepted. No distributed, storage-corruption,
infinite-interruption or effectful-recovery claim follows from these commands.
