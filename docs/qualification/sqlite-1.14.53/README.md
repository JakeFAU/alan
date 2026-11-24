# Public runtime SQLite qualification

On November 24, 2026, we tested the change from `github.com/mattn/go-sqlite3 v1.14.52` to `v1.14.53` against public runtime base `64a692b6f82ae10353f50c8741377166bc6cf854`. The root module selects the binding directly. The five nested Go modules under `docs/evidence` reproduce historical performance experiments and retain their recorded pins. Upstream maintains the latest patch on the 1.14 line. The update changes Go/C binding behavior and the bundled SQLite engine. [Upstream release](https://github.com/mattn/go-sqlite3/releases/tag/v1.14.53) · [Upstream support policy](https://github.com/mattn/go-sqlite3/security/policy)

| Binding | Verified upstream commit | Bundled SQLite | Go module checksum |
| --- | --- | --- | --- |
| v1.14.52 | `b0be46fa28d17ee0b65c79774ac0dad84b6db068` | 3.53.4 | `h1:wVbm2Qnf4OXkqhBTSPuCRZDRnxfbVrrmiCEroVdog8U=` |
| v1.14.53 | `93c7e0a2ec6762a38ab371755d49a2b3f4a12e67` | 3.54.1 | `h1:mnTM0owzUMrKuLo+0w51NUiB0ir2Fo7kFXlPt/l3BGs=` |

The host used Go 1.27.4 on macOS 27.3 arm64, CGO, Apple Clang 21.0.0, a local filesystem with SQLite WAL, and loopback test endpoints. Each binary reported its expected binding and checksum. Each compared Inspect and Replay record reported its engine's exact version and SQLite source ID. The version-specific check then compared the remaining output bytes. SQL field values, journal ordering, connection settings other than engine identity, and ALAN storage versions participated in those comparisons.

## What survived the binding change

| Exercise | Observed result |
| --- | --- |
| Both writer/reader directions, immediately before and after an acknowledged nested execution commit | Four SIGKILL cases. The opposite binary resumed the stopped store and the writer resumed an identical stopped copy. Both recovered 26 commits and four intents, with identical run reports and retained output. Read-only APIs preserved database/WAL bytes. Both binaries rejected a subsequent corrupted history record in each case. |
| Ten existing old-writer formats 3–12 | Independent copies for Inspect, Replay, typed inspection and owner Open yielded 80 reader/API cases. All 60 paired exports matched after checking the engine identity fields. Only the open-run epochs changed in the typed Open comparison. The original ten fixture checksums matched before and after. |
| Public K01/K02 and promoted conflicts, formats 13–16 | Each binding produced both authored parents. The existing internal conflict writer promoted copies to 14 and 16 and checked that the original selected result survived. Both CLIs inspected and replayed all eight retained states. |
| Pinned historical readers | The pre-authority CLI at `af6943a5805a9a6c0e8fe77f804264f4765d07ec` accepted its own format-3 fixture. Its Inspect and Replay each rejected genuine format 13 and format 14 with exit 1 and `unsupported version`. The pre-binding owner at `d59eaf2976849e450635d252c7bca62eeccbef2e` opened a candidate-written format-13 K01, advanced the open member's epoch and took zero steps and dispatches. The format-15/16 controls observed its version refusal on both copies. |
| Physical integrity | Python's SQLite 3.54.1 reported `ok` for `PRAGMA integrity_check` and zero rows for `foreign_key_check` on all 18 disposable fixture copies. |

The original same-engine comparison stopped at the first across-binding Inspect/Replay byte mismatch. On the legacy matrix, 20 typed export pairs matched outright. The other 40 differed only at the two engine identity fields. The strict same-engine comparison also passed its four control schedules. The [binding-specific comparator](compare.py) pins the two live identities and preserves every other byte in its comparisons.

## A failed work-count assumption

The first candidate's ordinary and integration checks exposed a difference between SQLite authorization callbacks during statement preparation and calls that actually execute a query. On the two-member readiness fixture, SQLite 3.54.1 authorizes the bound projection and member JOINs twice per execution. The experiment changed the binding and bundled engine together. Native `SQLITE_TRACE_STMT` on both bindings counted two run reads and two binding reads. The reference selector loaded the program twice and the optimized selector loaded it once. The old callback counter on 3.54.1 reported four run reads and four binding reads. The timing and CPU cost of those extra authorizations remain unmeasured. [Native trace fixture](readiness_trace_test.go.txt)

The ordinary failures were `TestP3ReadinessSameCallBoundPrograms` and `TestP3ReadinessSameCallSecondBindingCorruption`. Integration also failed `TestFinishRunReportDescriptorQueryCount` at sizes 1, 4 and 8 and `TestStaticExecutionHotWritesAndAffectedRoot` on its baseline run calibration. The old cancellation test could pass on the second authorization of the first run. The revised test observes the bound arguments and requires `r00/1` followed by `r01/1` at cancellation. The logical counts remain two visits/two bindings, eight baseline visits/one optimized visit for the static fixture, and one report read per batched family.

The test-only observer associates each authorization with its Go query and with that query's rows during a deferred first step. It keeps repeated queries with identical arguments distinct. An initial observer missed an added prepared projection query and two SELECTs in one batched call. The old binding's original report oracle failed those mutations while that observer passed. The corrected observer reports incomplete measurement for unscoped reads and SQL with an internal semicolon. Guard tests execute prepared, Exec and batched reads. The repeated-lookup control requires all three visits across two identical subjects and one distinct subject. These changes concern test instrumentation. Runtime execution, SQLite DDL and ALAN program versions retain their existing definitions. [Observer and guard tests](../../../internal/executor/sqlite_query_probe_test.go)

## Verification

The dependency-only input tree was `6ed94820972962d4a267f496c17c5332ce88310b`. The complete ordinary and integration module suites passed on test-input tree `ab0a584d15ee0c98e32a461caf91cd09287eb4f3`. All 19 packages with tests passed in each mode. The integration suite skipped its opt-in historical expected-failure test, `TestPhase3B3BaselineS2CompletionExpectedFailure`. Final Go/module input tree `5af936342d0d4820e792936baa0e963cfbc442cd` differs from that tree only in `internal/executor/sqlite_query_probe_test.go`. The final executor package passed its ordinary, integration and race-enabled integration suites, including both frozen-reader families. Those runs recorded 749, 1,600 and 1,600 passing tests/subtests respectively. The focused format, historical-owner and native-trace recipe also ran on the final tree. The native trace confirmed the same statement counts with either binding.

The first full-module race-enabled integration run on `ab0a584d` used `GOMAXPROCS=3` and shared the host with the research repository's ordinary suite. It exited 1 after the 60-minute package timer expired in `examples/outing-planner` during `TestDelegatedReplacementRetainsEvidence`, at 3,600.198 seconds. The other 18 tested packages passed. The run recorded 3,007 passing tests/subtests and the same opt-in skip. The outing-planner package completed 38 of its 54 registered top-level tests, with one interrupted and 15 still to start. Its 166 passing tests/subtests included all of `TestDelegatedProcessRecovery`. The timeout was the sole failure event.

Go module verification and tidy checks, the package build, ordinary and integration Go vet, and Staticcheck 2026.2.2 in both modes passed. A supplemental govulncheck v1.9.0 source scan of the candidate module in ordinary and integration modes reported zero reachable known vulnerabilities. The Go vulnerability database snapshot was modified November 23, 2026 at 20:05:14 UTC and had archive SHA-256 `5493d11c19e1bc9cf7e47764a90c34d530af052c869f043576bafd92ceabc736`. This scan covers Go symbols. The bundled native SQLite implementation falls outside that analysis.

The initial sandbox blocked the public module proxy, loopback binds and the macOS Staticcheck cache. The authorized reruns verified the two Go module checksums, exercised the loopback fixtures and completed both lint modes. Historical measurements retain their original source and dependency assumptions.

## Reproduction

Use a clean candidate checkout with the full public runtime Git history, Go, a C compiler, Git, tar, Python 3 and local loopback. The following commands assume the candidate is committed at HEAD. The focused reader gate uses a Git archive of HEAD. Scratch sources and database copies belong outside both checkouts.

```sh
set -eu
candidate=$(pwd -P)
work=$(mktemp -d)
base=64a692b6f82ae10353f50c8741377166bc6cf854
pre_authority=af6943a5805a9a6c0e8fe77f804264f4765d07ec
export GOWORK=off GOTOOLCHAIN=local CGO_ENABLED=1 GOFLAGS=
git clone --quiet --shared --no-checkout "$candidate" "$work/baseline"
git -C "$work/baseline" checkout --quiet --detach "$base"
go mod download -json github.com/mattn/go-sqlite3@v1.14.52 github.com/mattn/go-sqlite3@v1.14.53
python3 -B docs/qualification/sqlite-1.14.53/compare.py "$work/baseline" "$candidate" "$work/cross"
python3 -B docs/qualification/phase3b3/qualify-old-reader.py "$work/format-gate"

mkdir "$work/pre-authority"
git archive --format=tar -o "$work/pre-authority.tar" "$pre_authority"
tar -xf "$work/pre-authority.tar" -C "$work/pre-authority"
go -C "$work/pre-authority" build -mod=readonly -buildvcs=false -o "$work/pre-authority-cli" ./cmd/alan
export ALAN_PHASE3B3_OLD_READER="$work/pre-authority-cli"
export ALAN_PHASE3B4_OLD_READER="$work/cross/historical-owner"
export ALAN_PHASE3B4_PUBLIC_PARENTS="$work/cross/v1.14.52/parents"
go mod verify
go mod tidy -diff
gofmt -l adapters author cmd compose effects examples internal runtime schema
go build ./...
go vet ./...
go vet -tags=integration ./...
staticcheck ./...
staticcheck -tags=integration ./...
go test -count=1 -timeout=60m ./...
ALAN_PHASE3B4_PROMOTED_OUTPUT="$work/integration-promoted" go test -count=1 -timeout=60m -tags=integration ./...
ALAN_PHASE3B4_PROMOTED_OUTPUT="$work/race-promoted" go test -count=1 -timeout=60m -race -tags=integration ./...
```

To reproduce the original failure using the baseline tests, copy the candidate module files to the scratch directory. This command exits 1 at the four existing test names.

```sh
cp go.mod "$work/rejected.mod"
cp go.sum "$work/rejected.sum"
go -C "$work/baseline" test -count=1 -mod=readonly -modfile="$work/rejected.mod" -tags=integration \
  -run '^Test(P3ReadinessSameCallBoundPrograms|P3ReadinessSameCallSecondBindingCorruption|FinishRunReportDescriptorQueryCount|StaticExecutionHotWritesAndAffectedRoot)$' ./internal/executor
```

The focused script saves source and binary identities, build info, native statement traces, original fixture hashes, and both sets of legacy raw exports. Each SIGKILL comparison reads the stopped database before resuming it in place. Its case directory retains the recovered store and deliberately corrupted control. Repeating the recipe recreates the pre-recovery cut. The official format gate saves all four required command-level pass events. The SIGKILL evidence covers one host and the specified process boundaries. Power loss, all filesystem or engine builds, the earlier performance campaigns, and general storage equivalence remain outside the qualification.
