# Reproduce the Phase 2A campaign

Work at the source identity in the PR. The integration base is `18d35a2b312cf7180926d4b4cc5c7c2094cd68e2` and the protocol commit is `4c8a735`. Run from the repository root with the checked-in separate consumer replace directive. The apparatus requires SQLite CGO, Go 1.27.3, a writable build cache, local process creation and loopback TCP. All endpoints are controlled local processes. The test name "ExecutorProcess" selects helper code in an actual new process. Its top-level invocation without the child environment returns immediately and contributes no scientific case.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOFLAGS=-mod=readonly
export GOPROXY=off GOSUMDB=off
out="$(mktemp -d)"
export GOCACHE="$out/build-cache"
phase='docs/foundational/phase2a'
process='^TestPhase2(PairedCrashCuts|KnownEvidenceCrashCuts|EndpointRestartBindingAndHistoricalReceipt|AdapterRealErrorAndSelectedUnsuccessful|ParentAdapterCrash|ExecutorProcess|EndpointLegacyPaddedJSON|EndpointLegacyMethodRejectBeforeBody)$'
for mode in normal race; do
  flags=()
  if [ "$mode" = race ]; then flags=(-race); fi
  ALAN_PHASE2_EVIDENCE="$out/process-$mode" go test "${flags[@]}" -tags=integration -count=1 -timeout=8m -v -run "$process" ./internal/durabletest
  ALAN_PHASE2A_AUTHORITY="$out/authority-$mode" go test "${flags[@]}" -tags=integration -count=1 -timeout=8m -v -run '^TestPhase2AAuthority$' ./internal/durabletest
  ALAN_PHASE2A_AUTHORITY="$out/withholding-$mode" go test "${flags[@]}" -tags=integration -count=1 -v -run '^TestPhase2AClosedWithholding$' ./internal/durabletest
  ALAN_PHASE2A_PARENT_EVIDENCE="$out/parent-$mode" go test "${flags[@]}" -tags=integration -count=1 -v ./examples/effectboundary
  go test "${flags[@]}" -count=1 -v ./research/effectprotocol/...
  (cd examples/effectboundary/consumer && go test "${flags[@]}" -tags=integration -count=1 -v ./...)
done
go mod verify
(cd examples/effectboundary/consumer && go mod verify && go vet -tags=integration ./... && go build -o "$out/consumer" .)
go vet -tags=integration ./research/effectprotocol/... ./examples/effectboundary ./internal/executor ./internal/durabletest ./internal/fixture ./cmd/alan-resource ./runtime
go run ./research/effectprotocol/cmd/export "$out/graphs.jsonl.gz"
python3 "$phase/audit-process.py" "$out/process-normal" "$out/process-race"
git diff --check
```

Verify the checked-in packet from the repository root before regenerating captures.

```sh
python3 -B docs/foundational/phase2a/tools/index.py
python3 -B docs/foundational/phase2a/tools/scope.py
python3 -B docs/foundational/phase2a/tools/audit_model.py
python3 -B docs/foundational/phase2a/tools/audit_observers.py
python3 -B docs/foundational/phase2a/tools/audit_withholding.py
```

The observer and withholding checks use the archived normal and race directories. The index hashes the current source/evidence set and checks that every eligible file is listed. Its two explicit exclusions are the index itself and the closing reviewer record, both bound by Git. `tools/index.py --write` regenerates the index for an intentionally changed packet.

Use bash or zsh for the array of flags. Writing capture directories separately avoids mixing PIDs, binaries or endpoint states across runs. The endpoint survives its executor's SIGKILL. A clean new process renews activation for open runs. Endpoint restart is a distinct dedicated experiment that reopens the same resource store. For actual argv, cwd, selected environment, source and executable hashes, outputs and exit status, read the compressed [command records](evidence/commands), [model tool record](evidence/model/tool-record.json) and per-child provenance records in the process directories. Paths in those records identify the original checkout and temporary execution context. A raw-store reference can be resolved in a portable checkout using its basename in the same capture directory. SQLite images are gzip-compressed online backups. Decompress to a file and open `file:/path/to/backup.sqlite?mode=ro&immutable=1`. The local-store/endpoint DBs have unrelated schemas. Reading the endpoint journal or DB is an oracle operation.

The model qualification driver in `evidence/model/qualify.py` also records formatting, vet, normal and race test binaries and the exact exported graphs. It regenerates canonical files in that directory; use a disposable checkout to reproduce those. The independent checkers, negative controls and exact qualification census are linked from [qualification](06-qualification.md). The final PDF is built from the paper directory using `tectonic alan-dissertation.tex`. See [review](08-review.md) for its source review, build warnings and visual inspection.
