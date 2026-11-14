# Retained-query apparatus qualification

The B256 pilot missed all three conditions in the [frozen contract](contract.md).
The branch delivers an experiment and its evidence. Production runtime, tests,
dependencies and nested public consumer sources are byte-identical to the baseline.
The order swap lives in disposable exports. [Results](results.md)

## Source and custody

| Identity | Commit | Tree |
|---|---|---|
| Application baseline | `9db473bae28e5b806c3ccdc7e2e134c1d0b38f27` | `56669e93b11a35ab8a67d09918b3a5e055944015` |
| Initial apparatus and original smoke | `481de68a98f50efc0c79918a822444afb4464074` | `2d89f8f0ef14016dd629ac7f147fc2a23be77eb4` |
| Corrected apparatus for the counters, clock, profile, primary pilot and detached-checkout reproduction | `ca65abb857a75f3677ce88a74cbeb94a2251450b` | `bb51e53578fda278c7b02a08c50c6efc9b56f7d8` |
| Reviewed Task 1 data and prose | `c19785d5ca2f3a6cc2ea4efe3ca539c76a0a6b80` | `3cca2e46782f03f2bb351d11cb0c98efed1b309a` |

The [measurement bridge](data/source-bridge.json) applies through the Task 1
evidence commit. The [delivery bridge](data/delivery-bridge.json) accounts for
the following documentation and evidence commit, including the root README
navigation entry. The bridge records each changed path and content digest,
with its own path listed separately to avoid self-reference. Subsequent review
transcripts can be bound with another documentation-only bridge.

Task 3 matched all 125 compact lane rows and 327 command and custody records
against local raw files. The checks covered sample ordering, command and
binary binding, source inventories, the exact one-file overlay, independent
workload cardinalities, and per-stage counts. An independent calculation
recovered the paired medians and spread, AB/BA strata, resource medians, SQL
registry deltas and all clock on/off/empty aggregates from raw reports. The
pilot comprises 98 rows, with 14 warmups and 84 measured rows. Reproduction
rows below belong to a separate run. [Timing rows](data/timing-rows.jsonl) ·
[Timing summary](data/timing-summary.json) · [Counters](data/counters.json) ·
[Clock](data/clock.json)

The measurement roots are under the local
`.superpowers/sdd/plan-retained-query-scaling/` workspace. The original smoke
is in `smoke/`, the corrected lanes are in `frozen-counters/`,
`frozen-clock/`, `frozen-profile/` and `pilot/`, and the profile renders
are bound under `profile-render/`. The ordinary suites are in
`baseline-suite/` and `baseline-suite-loopback/`. The retained attempts
`preflight-clock-1/`, `preflight-clock-2/` and `counters/` contain partial
and preflight records. The committed files index the source, command, output
and result hashes. Raw archives, logs, generated source, profile binaries and
SQLite stores remain local. [Lane manifests](data/manifests.json) ·
[Gate ledger](data/gates.jsonl) · [Environments](data/environments.json) ·
[Profile bindings](data/profile.json)

## Detached-checkout reproduction

A shared local clone outside the development worktree checked out the
corrected apparatus commit in detached state. Its local Git exclude kept the
fresh plan outputs outside tracked source. The reproduction used Go 1.27.4
on Darwin arm64, Apple clang 21.0.0, Python 3.9.6 and cached SQLite v1.14.52,
with Go module proxy and sum database access disabled. All 829 tracked file
hashes and Git identity matched before and after each group of checks.
The generated control inputs and stock smoke inputs each matched across
their run boundaries.

| Check | Exit and result |
|---|---|
| `python3 -B -m unittest discover -v -s docs/evidence/retained-query-scaling -p test_measure.py` | 0, seven tests passed |
| `measure.py --lane clock --preflight` | 0, generated Go controls and diagnostic build |
| `go test -count=1 -json -run='^Test(P7\|Measurement\|ScalingCompact)' .` in generated SQLite | 0, 35 passing test events, one passing package, zero skips or failures |
| `measure.py --lane smoke` | 0, four fresh stock rows at bounds 1 and 8 in both arms |
| `go build ./...`, `go vet ./...`, `go mod verify` | 0 for each at the root module |
| `staticcheck -f json ./...`, `staticcheck -f json -tags=integration ./...` | Initially 1 for each because the sandbox denied writes to the default cache. Repeated with a writable reproduction-local cache, both exited 0 with zero diagnostics |
| `gofmt -l` on the generated files formatted by preflight and all 354 tracked Go files | 0, empty output for both |

Both stock arms passed retained results, charge, contact, reopen and Replay
checks in the smoke. The clock preflight runs Go controls without diagnostic
sample rows. The smoke confirms path-independent apparatus reproduction on
this host. Its elapsed values are outside the primary latency comparison.
The raw inventory indexes were hashed before any cleanup and remain under
`/private/tmp/alan-p7-task3.OFX5ygCH`. The compact record includes exact
argv, directories, environment, exits, input and output digests and smoke
row identities. [Reproduction record](data/reproduction.json) ·
[Reproduction commands](results.md#uninstrumented-paired-pilot)

## Qualification limits and review

The Task 1 ordinary baseline run `go test -count=1 -json ./...` exited 0
with 1,420 passing test events, 18 passing packages, three packages without
test files and four optional frozen-reader subtest skips. The preceding
sandboxed attempt exited 1 with local loopback bind failures. The ordinary
suite covers existing recovery tests. Task 3 reused this result at identical
runtime and test bytes. The four subtests require the historical reader
binaries named in [the results](results.md#inputs-and-ordinary-baseline).

The discarded overlay did not proceed to a production candidate. This
delivery has no new selected-write SIGKILL or weather process-cut campaign,
same-seed failure/recovery comparison, historical-reader campaign, nested
consumer campaign, full integration or race-integration suite, or focused
race suite. Staticcheck's integration-tag run analyzed Go code. P6
qualification remains attached to its own source and is outside this
evidence. The final SQLite rollback control and the successful workload's
reopen and Replay checks establish only the behaviors they exercise.

The independent Task 1 review passed the frozen negative-result specification
and approved the evidence. It reported no rejected findings or requested
implementation changes. Some early controller and packaging corrections
lack separate terminal-outcome logs, and the gate ledger covers recorded
campaign work. [Independent Task 1 review](reviews/task-1.md)

Task 3 delivery review also passed and approved the evidence. Its sole minor
cache-location wording finding is corrected above. [Review records](reviews/README.md)
