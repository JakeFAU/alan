# Reproducing the qualification

The base was `72e406b73b05a3ed4a6e8b37801ab6e0f0a73209`. The separate contract commit was `656f84b58424b3e2a62ce9a9eeb6bd72f84a5784`. Run the final selections from the candidate checkout. For the historical check, use the exact previous checkout at `24485a2b4f5bb3af45027995c6e6d8f34e0acb2d`. The two base trees match. The prior 1,318-file index was verified there. [Base and matrix](01-source-and-matrix.md) · [Historical verification](evidence/commands/historical-index.json.gz)

The accepted runtime is commit `a57b754e8639182131ca07d8490541689987f6fc`. The model and correspondence are `93f3d0557d6ff3a5ebe2ef8e06593e5976cdbff3`. All Go and fixture inputs for the first focused campaigns can be reconstructed at `b3b7ebd722b8b2d2256c52596b535acaf221a430`. The corrected conformance fixture and all Go sources of the passing root run are at `08cf3533b381177c4d9515b786467bff82a06a92`. The reviewed manuscript and PDF are `08cfb4fd84470057941f8b0c9bc8cd14196032c1`. Per-command inputs below identify which tree supplied each execution.

## Environment and captures

The recorded host was darwin/arm64, Go 1.27.3, CGO enabled and github.com/mattn/go-sqlite3 v1.14.52 from the existing module cache. Use the matching Go and C toolchains, the existing pinned modules, a writable build cache, loopback TCP, OS process creation/kill and SQLite file locking. The independent decoder requires Python 3 with the standard sqlite3 and gzip modules. All counter endpoints run locally. From the repository root, these settings reproduce the Go build policy.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOFLAGS=-mod=readonly GOPROXY=off GOSUMDB=off
export PYTHONDONTWRITEBYTECODE=1 LC_ALL=C
campaign=$(mktemp -d)
export GOCACHE="$campaign/go-cache"
phase=docs/foundational/phase2b
```

Each compressed command record contains the resolved executable and SHA-256, complete argv including selectors, working directory, selected environment, source inventory, combined output and exit status. Most Go captures share the recorder. The model qualifier additionally hashes its built test/export executables and stores their Go build information. Child schedulers, endpoint executables, epoch changes, fixture acknowledgments and WaitStatus are captured separately inside the process campaign. The normal build uses the ordinary test binary. The race build executes the race-built test binary in every executor child and builds the endpoint with race instrumentation. An observation from a regular child under a race parent would fail the provenance check. Paths and PIDs in the archive identify the actual execution. Use fresh directories to reproduce it. [Command records](evidence/commands) · [Model build and execution](evidence/model/tool-record.json) · [Recorder](tools/record.py)

## Executed selections

The following normal commands were repeated with `-race` immediately after `go test`. For world and composition, use disjoint evidence directories for each command and regime. Set `ALAN_PHASE2_EVIDENCE` and `ALAN_PHASE2A_AUTHORITY` to those absolute paths to retain new captures. Unset the overrides for unrelated checks. The consumer command runs in `examples/revocation/consumer`. Tagged variants here use real repository selectors.

```sh
go test -tags=integration -count=1 -timeout=8m -v -run '^TestPhase2(B(RevokedCrashPair|RecordedRecovery|DelayedApplication|LiveOrders|SupersededOpportunity|OpenSuccess)|PairedCrashCuts|KnownEvidenceCrashCuts|AdapterRealErrorAndSelectedUnsuccessful|ParentAdapterCrash|AAuthority|AClosedWithholding)$' ./internal/durabletest
go test -tags=integration -count=1 -timeout=2m -v -run '^TestPhase2B(NestedAuthority|PrivateFailureOrders|SlowEffectAndPeers|SuccessfulSourceCapturesMissingEffect|ExactPriorStores|NestedDrainCoversDescendant)$' ./internal/durabletest
go test -tags=integration -count=1 -timeout=12m -v ./internal/executor
go test -tags=integration -count=1 -timeout=2m -v -run '^TestPhase2B' ./internal/executor
go test -tags=integration -count=1 -timeout=5m -v -run '^Test(Phase2EndpointLegacyPaddedJSON|Phase2EndpointLegacyMethodRejectBeforeBody|Phase2EndpointRestartBindingAndHistoricalReceipt|SeparateResourceHTTPRetainsReceiptAcrossDuplicate|AutomaticApplicationMissingReceiptProfiles|AutomaticExternalAuthorizationCrash)$' ./internal/durabletest
go test -tags=integration -count=1 -timeout=5m -v ./runtime ./internal/fixture
go test -count=1 -timeout=3m -v ./research/semantics
(cd examples/revocation/consumer && go test -count=1 -timeout=2m -v ./...)
```

The model checks, test binaries, deterministic complete outgoing export and bounded per-profile counts can be recreated in an empty directory. The audits below read the delivered evidence. The original-history checker reconstructs the rejected and corrected executors in scratch directories. The history command returns success when it obtains the specified verdict on the observed behavior. The independent graph checker applied directly to either historical rejected graph exits 1. [Independent model run commands](evidence/model-review.md)

```sh
python3 -B research/dispatchauthority/qualify.py "$campaign/model"
python3 -B "$phase/tools/audit_model.py" --mutants
python3 -B "$phase/tools/audit_model.py" --implementation-experiment rejected
python3 -B "$phase/tools/audit_model.py" --implementation-experiment corrected
python3 -B "$phase/tools/audit_orders.py"
python3 -B "$phase/tools/census.py" > "$campaign/leaf-census.json"
cmp "$campaign/leaf-census.json" "$phase/evidence/leaf-census.json"
python3 -B "$phase/tools/scope.py"
python3 -B "$phase/tools/run_inputs.py"
python3 -B "$phase/tools/index.py"
git diff --check 72e406b73b05a3ed4a6e8b37801ab6e0f0a73209
```

## Ordinary suite, tooling and manuscript

```sh
go test -p=2 -count=1 -timeout=60m ./...
go vet -tags=integration ./internal/durabletest ./internal/executor ./runtime ./effects ./cmd/alan-resource ./research/dispatchauthority/...
go vet ./research/semantics
go mod verify
(cd examples/revocation/consumer && go mod verify && go vet ./... && go build -buildvcs=false -o "$campaign/quiescence" .)
```

The [complete ordinary root run](evidence/commands/root-suite-final.json.gz) exited zero on the delivered Go sources. All 33 packages with tests passed, and eight packages reported no test files. The adaptive package completed in 1,447.676 seconds. The [initial 12-minute run](evidence/commands/root-suite.json.gz) timed out in the adaptive package and exposed the displaced G1 conformance test. A [pre-correction 60-minute run](evidence/commands/root-suite-60m.json.gz) reproduced the bridge failure. Its adaptive child was explicitly terminated to make room for a complete corrected run. The two earlier runs are retained as failures.

The original [format selection](evidence/commands/format.json.gz) and the [additional conformance fixture selection](evidence/commands/bridge-format.json.gz) had empty output. Focused vet, consumer vet/build and both module verifications exited zero. The compiler, schema, dependency pins and previous research archive have the same bytes as at the integration base. Per-command source manifests record what existed as each experiment ran. The final process, composition, direct, core, consumer and endpoint binaries were built before the ordinary suite exposed a displaced test in research/semantics. That test file is outside all those binaries. Each 408-file child source manifest includes the original test at SHA-256 `ac85028e1360e2b0ee9c8a1b7ddd925454ce4b900dd3a0843a0b04cbd1be4dfc`. That file is the only Go/native/module difference from the delivery tree. The original bytes are retained at the integration commit. The delivered bridge fixture SHA-256 is `d65c6eeffa5dfaf7b7e0a064e4bf9e6d59675d1eecee1b4f5577c6ac5c9beb89`. The final bridge and ordinary suite were qualified separately after the fixture repair. Independently authored Python audit tools were finalized at their own recorded identities.

The manuscript was built once by invoking `tectonic alan-dissertation.tex` in `paper/` after both manuscript sources received approval. Tectonic exited zero and emitted a 26-page A4 PDF, 266,535 bytes, SHA-256 `3c0003f705f30998a542b6eeb742d695984780ee9491401f6a29eed46e2e2f75`. There were underfull boxes at lines 189 and 463 and a 5.2184-point overfull box at 463. All pages were rendered. Pages 3–5 and 20–26, including the revised statements, continuations and bibliography, were visually inspected with no clipped text observed. The independent reviewer also read pages 21 and 24. [Build output](evidence/commands/manuscript-pdf.json.gz) · [Reviewed source identities](evidence/manuscript-review.md)

## Reading and preserving failures

`baseline-loopback-{normal,race}` is the working pinned baseline. Earlier `baseline-{normal,race}` records show the local listener restriction. The new-behavior checks on that runtime failed for real admitted/fresh exposure orders. Separate compile failures establish only that the observer API was absent. The rejected recovery-accounting implementation, intermediate fixture failures and rejected graph exports remain archived alongside final records. The final leaf census uses the named final runs, never smoke runs or compiler failures. [Core REDs and corrections](evidence/core) · [Review dispositions](08-review.md)

To capture a new experiment, invoke `python3 -B "$phase/tools/record.py" LABEL --out "$campaign/commands" -- COMMAND ARGS`. It refuses to overwrite an existing record. Use `--cwd` and `--env NAME=VALUE` before the separator as needed. The final index can be checked in place from a fresh checkout. It excludes its own bytes and the closing review. The containing commit binds both. [Index](evidence/index.sha256) · [Whole-phase review](evidence/whole-review.md)
