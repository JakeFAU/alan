# Reproduce and distinguish the executions

Base `a461f25fb42d1663735f68296460eceda4270d47` has the same tree as reviewed Phase 2B head `41fc50b3b821dcc4b3700920de3109fb3204315d`. The working branch started in a clean isolated checkout. Source ancestry and both historical inventories were verified on their documented historical trees. The opt-in profile introduces Store schema 5 and a separately initialized endpoint ledger format 2. Existing schema 3/4 behavior, old ledger format 1, the original mutation no-repeat rule and the historical evidence archives remain distinct. [Frozen requirements](01-contract.md) · [Historical A check](evidence/commands/phase2a-index.json.gz) · [Historical B check](evidence/commands/phase2b-index.json.gz)

| Source | Commit |
|---|---|
| Independently accepted prospective contract and minimal construction | `77c253e0c72ab77217b1e87c266a941a91fe2354` |
| Controlled endpoint and protocol | `7f06149` |
| Runtime candidate whose refusal checker was subsequently rejected | `e3a4375` |
| Rejected model candidate and first complete graph | `9f108d2b1eaf37cb585982f6c26d9830abff6fb7` |
| First process campaign, initial ingress/R ordering gap and working public consumer | `f5728074e7dfd94221672ef99944ce73e5095c5e` |
| Corrected model, independent full-edge checker, actual final-duplicate guard | `6a1c6486e4d38055d3718aa321f174e4b9f08eff` |
| Independently approved runtime repair | `407e055edd88bdadc812fef62bf9fea3101dc58c` |
| Final Go/native/module tree, including both source and endpoint barriers | `3eb02f15528a6a702b95492bc0cf30844f13d883` |

## Environment and commands

The recorded environment used Go 1.27.3, darwin/arm64, CGO, SQLite through the already cached mattn/go-sqlite3 v1.14.52, Python 3, real process creation and signals, and controlled loopback TCP. No provider was contacted. The recorder stores executable SHA-256, argv, actual cwd and environment, source manifest, stdout/stderr and exit. The final external-consumer driver is also source-hashed. Child process manifests and journals identify executable build mode, process, activation, physical ordinal and application ledger independently. [Command records](evidence/commands) · [Toolchain](evidence/commands/toolchain.json.gz)

From the repository root with the pinned modules available locally, use fresh evidence directories per run. Supplying a different writable build cache affects timing.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOFLAGS=-mod=readonly GOPROXY=off GOSUMDB=off PYTHONDONTWRITEBYTECODE=1
work=$(mktemp -d)
export GOCACHE="$work/cache"
phase=docs/foundational/phase2c
for mode in normal race; do
  raceflag=
  if [ "$mode" = race ]; then raceflag=-race; fi
  ALAN_PHASE2_EVIDENCE="$work/$mode" go test $raceflag -tags=integration -count=1 -v -timeout=12m \
    -run '^TestPhase2C(Process|ResolverAuthority|CompetingResolvers|OriginalBindingControls|RefreshAfterAdoption)' ./internal/durabletest
  go test $raceflag -tags=integration -count=1 -v -timeout=20m ./internal/executor
  go test $raceflag -tags=integration -count=1 -v -timeout=5m ./runtime ./author ./internal/fixture ./adapters/receiptquery ./cmd/alan-resource ./examples/reconciliation ./research/reconciliation/...
  (cd examples/reconciliation/consumer && go test $raceflag -count=1 -v ./...)
  python3 -B examples/reconciliation/consumer/qualify.py "$mode" "$work/public/$mode"
  go test $raceflag -tags=integration -count=1 -timeout=12m -v \
    -run '^TestPhase2(B(RevokedCrashPair|RecordedRecovery|DelayedApplication|LiveOrders|SupersededOpportunity|OpenSuccess|NestedAuthority|PrivateFailureOrders|SlowEffectAndPeers|SuccessfulSourceCapturesMissingEffect|ExactPriorStores|NestedDrainCoversDescendant)|PairedCrashCuts|KnownEvidenceCrashCuts|AdapterRealErrorAndSelectedUnsuccessful|ParentAdapterCrash|AAuthority|AClosedWithholding)$' ./internal/durabletest
done
go test -p=2 -count=1 -timeout=60m ./...
go vet -tags=integration ./internal/durabletest ./internal/executor ./internal/fixture ./runtime ./author ./effects ./adapters/receiptquery ./cmd/alan-resource ./examples/reconciliation ./research/reconciliation/...
go mod verify
(cd examples/reconciliation/consumer && go mod verify && go vet ./... && go build -buildvcs=false -o "$work/consumer" .)
```

The separate consumer uses only exported packages. It builds separate normal or race executables for both its own CLI and the controlled resource. The Go tagged crash harness runs a real Store owner process and an independent endpoint process. Local audit database readers arrange or verify schedules. They do not fill the program's target or application observation. The two-resolver control supplies a local ID to its second resolver. The primary handoff is ordinary. The process manifest and public driver document their boundaries. [Causal matrix](01-process-matrix.md) · [Live construction](04-live-and-public.md)

## Independent reruns

```sh
go build -buildvcs=false -o "$work/alan" ./cmd/alan
go run ./research/reconciliation/cmd/export > "$work/graph.jsonl"
gzip -dc "$phase/evidence/model/graphs.jsonl.gz" | cmp - "$work/graph.jsonl"
python3 -B "$phase/tools/audit_model.py" --mutants --challenge-prior --histories
python3 -B "$phase/tools/audit_raw.py" \
  --process-normal "$phase/evidence/qualified-normal" --process-race "$phase/evidence/qualified-race" \
  --authority-normal "$phase/evidence/qualified-normal" --authority-race "$phase/evidence/qualified-race" \
  --public-normal "$phase/evidence/qualified-public/normal" --public-race "$phase/evidence/qualified-public/race"
python3 -B "$phase/tools/attack_replay.py" "$phase/evidence/qualified-normal/atomic-import/final-local.sqlite.gz" "$work/alan" \
  --spent "$phase/evidence/qualified-normal/exhausted/ordinal-1-local.sqlite.gz"
python3 -B "$phase/tools/replay_packet.py" "$work/alan"
python3 -B "$phase/tools/replay_packet.py" "$work/alan" --historical
python3 -B "$phase/tools/census.py" > "$work/census.json"
cmp "$work/census.json" "$phase/evidence/leaf-census.json"
python3 -B "$phase/tools/source_inputs.py"
python3 -B "$phase/tools/scope.py"
python3 -B "$phase/tools/index.py"
git diff --check a461f25fb42d1663735f68296460eceda4270d47
```

The initial, rejected and intermediate runs have their original command records. The complete first-repair captures that preceded final hardening are packed under their original three relative directory names. To inspect these intermediate images separately use `tar -xzf "$phase/evidence/first-repair-captures.tar.gz" -C "$work"`. All 1,160 file digests were compared to the source directories before packing. SHA-256 of the archive is `fd26abccf568adfccac813e647df4809d17e5fc774f0da0f9153466560a4a48f`. The accompanying [historical core patch](evidence/first-repair-core.patch.gz) converts the final two production files to the exact first-repair bytes from the recorded manifest. Apply only in a disposable checkout of the final Go tree. The accepted final runs are directly under `qualified-normal`, `qualified-race` and `qualified-public`. [Review and failed commands](08-review.md)

The delivery index covers source inputs, corpus evidence, manuscript and predecessor digests. It excludes its own bytes and the final whole-phase review which cites that index. The containing Git commit binds all three. Run the historical verification in the appropriate historical checkout rather than comparing older manifests to newer code.

## Completed repository gates

The full ordinary root command `go test -p=2 -count=1 -timeout=60m ./...` ran once to exit 0. All 36 packages with tests passed and nine packages had no test files. The adaptive example took 1,534.530 seconds. The 60-minute per-package watchdog accommodated that example. The recorder captured 499 Go, native and module inputs at launch on HEAD `77c253e0c72ab77217b1e87c266a941a91fe2354`. Work continued while the suite ran. Twelve of those inputs changed and five more were added before the final focused runs. Each of the twelve final normal/race command manifests and the qualified vet and replay manifests matches all 504 final inputs. The comparison names each difference and gives its before/after digest. [Root suite command and output](evidence/commands/root-suite.json.gz) · [Per-file source comparison](evidence/commands/source-inputs.json.gz)

The format check covered 47 Go files and listed none requiring formatting. Focused integration vet, the separate public consumer's vet and build, and module verification in both modules exited 0. [Format](evidence/commands/format.json.gz) · [Integration vet](evidence/commands/qualified-vet.json.gz) · [Consumer vet](evidence/commands/public-consumer-vet.json.gz) · [Consumer build](evidence/commands/delivery-consumer-build.json.gz) · [Root modules](evidence/commands/root-modules.json.gz) · [Consumer modules](evidence/commands/public-consumer-modules.json.gz)
