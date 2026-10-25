# Phase 1D.1 immutable baseline reproduction

## Source and execution boundary

Before any implementation changes, I extracted Git commit `6effdc02259af5335633bfc41e9786386430fa7c` (tree `7fea16fd6a4f4903e9ea3966cedd104f746723e7`) with `git archive 6effdc02259af5335633bfc41e9786386430fa7c | tar -x -C /tmp/alan-phase1d1-baseline-source`. All commands below ran from this separate snapshot, except the expressly identified historical or published archive checks. The worktree itself was not used as a test source. The snapshot's 53 files in `docs/foundational/phase1d/source-manifest.json` each matched their byte length and SHA-256 before and after testing. Recomputing the manifest's length-prefixed aggregate SHA-256 gave `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`, exactly the published Phase 1D source identity. The manifest includes `go.mod` SHA-256 `1d8b5c8939d1422813ed2b004c6560e3ec3f262709d57e9b957f6994ffd9bdeb` and `go.sum` SHA-256 `9eb975c181ebee792e06f732fc13ef81acad0779d94731f75133ad2bba52c5ca`.

Host: Go `go1.27.3`, Darwin 27.2.0, arm64, `CGO_ENABLED=1`, `CC=cc`. The only external module is `github.com/mattn/go-sqlite3 v1.14.52`. The baseline Go commands used `GOTOOLCHAIN=local`, `GOCACHE=/tmp/alan-phase1d1-go-build`, `GOPROXY=off`, `GOSUMDB=off`, and `GOFLAGS=-mod=readonly`. Local loopback and subprocess crash fixtures were run with authorized execution privileges. No source files were changed.

## Current base results

From `/tmp/alan-phase1d1-baseline-source`:

| Command | Result | Test/subtest pass events | Markers | Raw Go JSON SHA-256 |
| --- | --- | ---: | --- | --- |
| `go test -json -count=1 ./...` | PASS, 26.958 s, empty stderr | 321 | 15 QUALIFICATION, 20 COMPOSITION, 32 SUPERVISION, 6 CONFORMANCE, 87 DURABLE | `6a0839382939101e7a3bf2fa11f05a38e3c1f2421b7f31f41928f4de7fe6e572` |
| `go test -json -count=1 -race ./...` | PASS, 110.001 s, empty stderr | 321 | same counts | `97a193b97e14c348e08d8d7c088380e2c0ecaad940a2385f2a06e615041bff32` |
| `go vet ./...` | PASS, no diagnostics | | | |
| `gofmt -l cmd internal research` | PASS, no files listed | | | |

Both Go JSON streams and the reconstructed machine records are retained as `normal.*.gz` and `race.*.gz` in this directory; `summary.json` has package results, hashes, commands, and environment. The 87 DURABLE markers in each mode include 61 actual process records and 16 `exit_signal=killed` records. All 61 process records have `race_children=false` in normal mode and `true` in race mode. They retain actual process IDs, ports, paths, and barrier facts. These physical values legitimately differ by run; no byte-identity claim is made for DURABLE records or complete Go JSON logs. Six CONFORMANCE records pass and three DURABLE records retain detected fault controls in each mode.

The current base's Phase 1 marker payloads are byte-identical to saved `docs/foundational/phase1/qualification.jsonl`. Current Phase 1B and 1C markers are behavior-identical after omitting only each `source_manifest` record and top-level `SourceID`; their complete bytes differ because the later source identity and dependency manifest differ. This is a current-source comparison, separate from the historical reproduction below.

## Historical source reproduction

I verified the three archive byte lengths and SHA-256 values against `docs/foundational/phase1d/historical-source/manifest.json`, extracted each independently, and ran its own qualification test with the baseline offline Go environment. The exact marker payload bytes, including their original source identity when present, match each extracted archive's saved `qualification.jsonl`:

| Archive / original commit | Command | Result | Exact records / SHA-256 |
| --- | --- | --- | --- |
| Phase 1, `f81dd200d05f7c9de62d25fa9baf6c9f6dd08bf9`, archive `2b2a1074b9b328314f7e7de7b2a649aaac514eb096ef4a9939a1321ca31a87ca` | `go test ./research/semantics -run '^TestQualification$' -count=1 -v` | PASS, 0.966 s | 15/15, `ebed311b655456f88e7519aa4e50eb55de5dca76cb3ec45935ab7b4d479a3834` |
| Phase 1B, `82dc5649cfe0eccb2fde3ecea1e12cc851377af5`, archive `b7237ff9332387b3a1b5be411efcab0761a951a652fa58571d41bba73c6cf447` | `go test ./research/semantics -run '^TestCompositionQualification$' -count=1 -v` | PASS, 1.061 s | 20/20, `1ef0b6a664753183126a54ff91d09128c1d3a3898fa04209270ea51494fd0c77` |
| Phase 1C, `3e003c91d35af7055f5be99aa9c79c74f3ea44db`, archive `33517e6b79b7d850c9bdae2ccba9f8f70237cf284b32f81fd2d31be417282013` | `go test ./research/semantics -run '^TestSupervisionQualification$' -count=1 -v` | PASS, 4.734 s | 32/32, `c0e7dba31fc8de2e979bd6a3dd829364b1e4b2a233a22e6ccf8634ac90f7c0da` |

Phase 1B's original source manifest is `afc89e4bf24126b33da5341f1bc4c582c81b0b80f51a04af1674daacf0b3c49a`; Phase 1C's is `08a107be1b4eff35986ab605dd1ef77db660baa6fb8e920f2d88a8f3031b859b`. Phase 1's saved records predate a source-manifest record. Compressed verbatim test logs and extracted marker payloads are retained here.

## Published Phase 1D archive reproduction

The original `docs/foundational/phase1d/source.tar.gz` has SHA-256 `49f68316357b07ac4bbd3f979cb47ebe25e732a28e3375081fb9ccacfcd8f3ea` (3,215,619 bytes). After extraction to `/tmp/alan-phase1d1-original-phase1d`, all 83 archive-manifest members matched their byte lengths and hashes; its 53 source entries recomputed the same `40a5b1...` digest. With a fresh `/tmp/alan-phase1d1-original-phase1d-modcache` and bundled `GOPROXY=file:///tmp/alan-phase1d1-original-phase1d/dependencies`, `go test -count=1 ./...` passed in 23.305 s. `go mod verify` reported `all modules verified`. Its raw package-output log is retained as `original-phase1d.test.log.gz`. This validates the published archive and bundled pinned dependency independently of the Git-base snapshot.

## Confirmed scheduling defect in the base

`internal/executor/runner.go:34-104` selects a pending external delivery, calls `s.DispatchExternal(ctx, delivery.intent)` synchronously at line 63, and only after that call returns reaches `s.selectReady(ctx, last)` at line 92. `internal/executor/external.go:150-151` performs the HTTP request with a two-second client timeout and waits for the response inside that call. Therefore, a held external response prevents the single automatic runner from selecting a peer's ready ordinary work during the hold. The passing historical timeout test demonstrates eventual progress after the external attempt ends; it does not establish progress while that exact request remains in flight. This is a source-level causal finding, not a claim that the new held-request acceptance test has already been run against this snapshot.
