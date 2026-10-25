# Phase 1D.1 runbook

Start with the root [README](../../../README.md), which covers builds, static nested/external examples, reopen, inspection and replay. Run from the repository root with Go 1.27.3, CGO and a C compiler. `--max-in-flight 2` and `--external-timeout 2s` are the conservative defaults. Capacity one still permits independent ordinary work. Use fresh executor/resource database pairs; database filenames do not establish business namespaces.

## Reproduce current qualification

```sh
export GOTOOLCHAIN=local
export GOCACHE=/tmp/alan-phase1d1-go-build
# A populated module cache is sufficient; the archive instructions below supply it offline.
export GOPROXY=off
export GOSUMDB=off
python3 docs/foundational/phase1d1/tools/qualify.py normal
python3 docs/foundational/phase1d1/tools/qualify.py race
python3 docs/foundational/phase1d1/tools/checks.py
python3 docs/foundational/phase1d1/tools/smoke.py
```

The collector verifies the exact source manifest before and after running. It saves raw `go test -json` output and extracts complete records without rearranging them. Running it in the checkout intentionally replaces only the new phase's final evidence; use the archive in a new directory to preserve published evidence. PIDs, ephemeral ports, absolute paths and process tokens vary. Compare causal assertions and public projections, not clocks or byte-identical live transcripts. `--parse-existing` re-extracts the saved verified raw log without rerunning tests.

Focused cases:

```sh
go test -race -count=1 ./internal/executor -run '^TestAsync' -v
go test -race -count=1 ./internal/durabletest \
  -run '^TestAutomaticAsyncHeldProgressAndConcurrentCrash$' -v
go test -count=1 ./internal/durabletest -run '^TestAsyncOracleRejectsFalseOrderAndIdentity$' -v
```

The subprocess test launches the actual CLI and separate endpoint. It owns exact pipe handshakes and kills at acknowledged boundaries. Tests do not ask a human to race `kill` against a sleep. Runtime `ALAN_BARRIER` instrumentation is a test observer, not an operational API or ordinary program behavior.

## Complete offline source snapshot

`source.tar.gz` contains the module sources/tests, contracts/README, qualification tools and the exact SQLite module as a file-proxy zip/mod/info. `archive-manifest.json` identifies the archive and every member; generated review/result files remain alongside the archive to avoid self-reference. Verify the SHA-256 and member inventory before extraction. The installed Go/C toolchain is external and identified by the source manifest.

```sh
alan_repro_dir=$(mktemp -d)
tar -xzf docs/foundational/phase1d1/source.tar.gz -C "$alan_repro_dir"
cd "$alan_repro_dir"
export GOTOOLCHAIN=local
export GOPROXY="file://$alan_repro_dir/dependencies"
export GOSUMDB=off
export GOMODCACHE="$alan_repro_dir/module-cache"
export GOCACHE="$alan_repro_dir/build-cache"
go mod verify
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

[archive-reproduction.json](evidence/archive-reproduction.json) records the actual empty-module-cache extraction/test run. For original historical evidence reproduction, use each retained earlier phase's source archive and commands in [the baseline report](evidence/baseline/report.md); do not stamp those results with the current runtime digest.

## Reproduce the synchronous negative control

Extract `evidence/implementation/sync-negative-control.tar.gz` into another empty directory. Use its manifest to verify the files, and use the final archive's dependency proxy if the module cache is empty. From that control directory:

```sh
GOTOOLCHAIN=local go test ./internal/executor \
  -run '^TestAsyncHeldRequestPeerProgress$' -count=1 -v
```

Expected: FAIL, `held request ended before peer commit`. The byte-identical traced test passes on corrected source. `sync-negative-control.json` retains the original pre-telemetry fixture digest; `implementation-evidence.json` and the per-file archive manifest identify the final traced fixture. The other complete pre-fix archives reproduce cancellation, post-commit accounting and observer-pipe failures with commands in their adjacent JSON metadata. Saved source excerpts use `.go.txt` so documentation is not compiled as a package.

## Reading a stopped run

Inspect the report and database records together. `Quiescent` means no supported eligible work in that invocation; an unresolved non-idempotent attempt may remain Unknown with an open run. `MaxSteps` counts committed ordinary transitions; `MaxDispatch` reserves supported delivery/launch work; completion handling spends no second unit. Additional eligible work at an exhausted budget returns an error. Cancellation and Close cancel/join transport, account queued candidates when possible, and report unresolved evidence or persistence failure. They do not synthesize CancelRun or RetireSubject.

To reopen, run the same CLI against the same store. It renews physical epochs, keeps logical generations/obligations and original keys, and permits only the declared profile's recovery. `replay` is read-only historical verification; subsequent `run` is live recovery and may dispatch eligible unresolved work.
