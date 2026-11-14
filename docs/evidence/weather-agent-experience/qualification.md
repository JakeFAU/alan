# Weather agent qualification

The weather example and its operator and authoring guides were qualified on
November 14, 2026. The [contract](contract.md) describes the scope. The
controlled provider binds to loopback. The archived Open-Meteo run was read
offline from a private copy.

## Frozen source and toolchain

| Identity | Value |
| --- | --- |
| Baseline commit | `42122cd69742d8a231d24cd497a95f6952163e27` |
| Baseline tree | `1f8f8fa27838165b93756bbeb31811b0ea99821f` |
| Reviewed weather source | `b0d52c8fcc6805be66551180edc9d4f658fa6333` |
| Qualified commit | `9c2b0f6b34c59f2ea57517c45f6f7036ff253b16` |
| Qualified tree | `1d5a8d79e1b3814924d15cdff007718635569afe` |
| Git archive SHA-256 before and after | `768c9833741de6897e1d527e844247ff3c66c127771dc9ca07f60d5a55838e66` |
| Tracked file SHA-256 manifest before and after (863 paths) | `3d7a7fb53d42f40bde4d37fa65beb7f4e570bcd56c5322078e7e0d9a30346476` |

Go 1.27.4 on darwin/arm64, Apple clang 21.0.0
(`clang-2100.4.1.1`), CGO enabled, and Staticcheck 2026.2.2 (0.8.2).
The root module declares Go 1.27.3 and resolves
`github.com/mattn/go-sqlite3 v1.14.52` against the checked-in
[`go.sum`](../../../go.sum). Commands used `GOTOOLCHAIN=local GOPROXY=off
GOSUMDB=off GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=1`, writable
temporary Go and Staticcheck build caches, and a locally cached module.
The independent weather test build supplies its own `-mod=mod` flag
and uses the same offline module cache.

## Repository gates

Run from the repository root. The test commands cover the root module.
Five historical nested evidence modules have their own `go.mod` files.

| Command | Exit | Observed result |
| --- | --- | --- |
| `go test -count=1 -json ./...` | 0 | 1,424 passing test events, 18 passing packages, 4 test skips |
| `go test -count=1 -json -tags integration ./...` | 0 | 2,703 passing test events, 18 passing packages, 6 test skips |
| `go test -race -count=1 -json -tags integration ./...` | 0 | 2,703 passing test events, 18 passing packages, 6 test skips |
| `go build ./...` | 0 | Built all root packages |
| `go vet ./...` | 0 | No diagnostics |
| `go mod verify` | 0 | All modules verified |
| `git ls-files -z -- '*.go' \| xargs -0 gofmt -l` | 0 | No files listed |
| `git diff --check 42122cd69742d8a231d24cd497a95f6952163e27 HEAD` | 0 | No whitespace diagnostics |
| `staticcheck ./...` | 0 | No diagnostics |
| `staticcheck -tags integration ./...` | 0 | No diagnostics |
| `staticcheck ./examples/weather-agent` | 0 | No diagnostics |

Three packages have no tests in each configuration. The four test skips
common to the suites were the format 13 and 14 `frozen_old_reader` subtests
of `TestRequestAuthorityFormatsConflictPromotionAndOldReader` and the false
and true `frozen_reader` subtests of `TestRequestBindingConflictAndOldReader`.
They require `ALAN_PHASE3B3_OLD_READER` or `ALAN_PHASE3B4_OLD_READER`.
The integration configurations also skipped
`TestPhase3B4PromotedPublicParents` (requires freshly exported exact public
K01/K02 parents and an output root) and
`TestPhase3B3BaselineS2CompletionExpectedFailure` (opt-in historical
falsifier). Those historical inputs were outside this qualification.

Both `TestCompetingReadCannotSettleAdopterDebt` children passed in the
integration and race-integration runs. The previous intermittent failure
and test-order investigation remain in the
[series scaling qualification](../series-scaling/qualification.md#initial-delivery-qualification-and-later-corrections).
The test-order correction is present in this repository as `8c4fb12`.

The first successful build and offline walkthrough emitted one warning
apiece while Go tried to write local VCS module metadata to the protected
module cache. A fresh temporary module cache populated from the existing
SQLite module zip and checksum eliminated the warning. The repeated
`go build ./...`, `go mod verify` and offline walkthrough each exited 0
with empty stderr.

## Weather checks

- The [documented controlled SIGKILL command](../../../examples/weather-agent/README.md#reproduce-a-controlled-interruption)
  ran the named `accepted_and_lost_then_explicitly_reobserved` subtest
  and exited 0. The fixture held the first response after
  acknowledging its GET. Fresh readers and two resumes preserved the original
  uncertainty at one observed contact. Only explicit approval followed by
  resume produced the second contact and distinct retained reading.
- The [offline walkthrough](../../../examples/weather-agent/README.md#reproduce-the-archived-reading-offline)
  and top-level `--help` and `status --help` exited 0. The archived
  Store's SQLite SHA-256 and HTTP response digest matched the manifest.
  Status reported the original `["weather-1",1,1,0]` reading of 80.3 °F
  and 77% relative humidity at `2026-11-03T20:15:00Z`. The CLI reported
  one spent opportunity and unknown provider cost. Replay verified 15
  reductions. The archived Store and its manifest are unchanged from the
  baseline.
- For the compatibility comparison, build the weather CLI once from the
  baseline and once from the qualified source with
  `go build -buildvcs=false -mod=readonly -o PATH ./examples/weather-agent`.
  On each retained cut, compare the baseline `inspect --db FILE` bytes with
  qualified `inspect --db FILE`, `inspect --db FILE --json` and
  `status --db FILE --json`. Also read human `status --db FILE` and
  check database and WAL bytes before and after every qualified reader.
  All 12 JSON comparisons were byte-identical and all 16 reader commands
  exited 0 with identical main/WAL bytes. An absent WAL counts as empty.
  SHM is outside this check.

| Retained cut | JSON bytes per detailed read |
| --- | --- |
| Two retained readings in a three-position loopback deployment | 19,330 |
| Acknowledged loopback GET with the first response missing | 13,377 |
| Retained HTTP 503 from loopback | 5,404 |
| Existing live Store read offline | 8,529 |

## Source bridge and log custody

The qualified commit changes four Markdown files. Its Go files, module files
and weather fixtures match the reviewed weather source. The tracked
files matched the frozen manifest before and after qualification. To
reproduce the manifest on a clean checkout at the qualified commit, run
`git ls-files -z | xargs -0 shasum -a 256` from the repository root.
The evidence commit adds this record, records the supported SQLite formats
in the root README and restores the weather README's old section links with
anchors. Verify the bridge with `git diff --name-status 9c2b0f6b34c59f2ea57517c45f6f7036ff253b16 HEAD`
and the executable and fixture assertion with
`git diff --exit-code 9c2b0f6b34c59f2ea57517c45f6f7036ff253b16 HEAD -- '*.go' go.mod go.sum examples/weather-agent/testdata`.

Raw command, stdout, stderr, exit and source-manifest records are retained
locally under
`.superpowers/sdd/plan-weather-agent-experience/qualification/`. The
paired Stores, baseline metadata and byte comparisons are under that plan's
`json-compatibility/` directory.
