# Phase 1D baseline reproduction

Before runtime behavior changed, an independent read-only agent reproduced all saved research evidence from isolated source archives. No research source or saved record was rewritten. Initial checkout was clean at `3e003c91d35af7055f5be99aa9c79c74f3ea44db`; the verified implementation base `a0d337c7b4f7340a55dea11b619cf9b22c25e2b0` has the same tree.

Actual machine records are retained for [platform](evidence/baseline/platform.jsonl), [source manifests](evidence/baseline/manifests.jsonl), [comparisons](evidence/baseline/comparisons.jsonl), and [scenario counts](evidence/baseline/counts.jsonl). Temporary paths in those records identify the original executions; the records do not assert that those temporary directories remain available.

## Historical source reproduction

| Phase | Source commit | Original records | Result |
| --- | --- | ---: | --- |
| 1 | `f81dd200d05f7c9de62d25fa9baf6c9f6dd08bf9` | 15 | byte-for-byte identical |
| 1B | `82dc5649cfe0eccb2fde3ecea1e12cc851377af5` | 20 | byte-for-byte identical |
| 1C | `3e003c91d35af7055f5be99aa9c79c74f3ea44db` | 32 | byte-for-byte identical |

Independent SHA-256 verification of all fourteen Phase 1C manifest entries (168,568 bytes) and the aggregate matched `08a107be1b4eff35986ab605dd1ef77db660baa6fb8e920f2d88a8f3031b859b`. The algorithm is the one retained in the Phase 1C manifest: sorted paths with uint64-BE path and raw-file lengths.

## Current-source comparison before implementation

Phase 1 still produced fifteen byte-identical records. Phase 1B produced nineteen identical behavioral records after omitting only its source-manifest record and top-level `SourceID`; its current manifest truthfully includes the Phase 1C files. Phase 1C produced thirty-two byte-identical records. Neither fixture definitions nor traces, bounds, counters, witnesses, rejections, or final states were normalized.

Adding SQLite changes `go.mod`, so current research source manifests will change even when research behavior does not. Final implementation qualification reports that current-source comparison separately; it cannot claim the old source hash for new source.

## Platform and actual checks

Host: Darwin 27.2.0, arm64. Compiler: `go version go1.27.3 darwin/arm64`.

All baseline Go commands used:

```sh
export GOTOOLCHAIN=local
export GOCACHE=/tmp/alan-phase1d-baseline-go-cache
export GOPROXY=off
export GOSUMDB=off
```

| Command | Actual result |
| --- | --- |
| `go test -count=1 ./...` | PASS, research package 4.355s |
| `go test -race -count=1 ./...` | PASS, research package 78.925s |
| `go vet ./...` | exit 0, no diagnostics |
| `gofmt -l research/semantics/*_test.go` | exit 0, no paths |
| `git diff --check` | exit 0, no diagnostics |

Complete original trees are also retained as [Phase 1](historical-source/phase1.tar.gz), [Phase 1B](historical-source/phase1b.tar.gz), and [Phase 1C](historical-source/phase1c.tar.gz) archives, with [archive identities](historical-source/manifest.json). These permit historical reproduction without access to the original Git repository. Extract an archive into a separate temporary directory, or use `git archive <commit> | tar -x -C <directory>`. In that directory run:

```sh
# Phase 1
 go test ./research/semantics -run '^(TestQualification)$' -count=1 -v
# Phase 1B
 go test ./research/semantics -run '^(TestQualification|TestCompositionQualification)$' -count=1 -v
# Phase 1C
 go test ./research/semantics -run '^(TestQualification|TestCompositionQualification|TestSupervisionQualification)$' -count=1 -v
```

Extract each log line's payload following `QUALIFICATION `, `COMPOSITION `, or `SUPERVISION `, preserving order and newlines, then compare to that phase's saved `qualification.jsonl`. Current-source behavior comparison removes only `Name=source_manifest` records and top-level `SourceID` before structural comparison. Historical byte comparisons perform no normalization.

The published archives were independently extracted and rerun after packaging. All 15/20/32 records again matched byte-for-byte; [actual archive-verification records](evidence/baseline/published-archives.jsonl) retain commands, exit codes, elapsed times and output hashes. These checks ran offline with the pinned local Go toolchain.

## Preserved oracle regression

Phase 1C's `TestSupervisionInterruptionWitnessCorrelation`, `TestSupervisionInterruptionWitnessOrdering`, and `oracle_regression_interruption_execution_and_order` remain required regressions. The retained false-positive input authorizes and decides an outer lifecycle request, reactivates that outer execution, then prepares and interrupts a different inner execution. That suffix must not prove an interruption-before-reactivation witness for the outer execution. A genuine witness must name the same run and pending operation in the correct order.

The historical 1/1B/1C positive counts remain 331/192/1,576 scenario states and 681/473/4,973 accepted edges, respectively. The new implementation does not relabel these finite research searches as durability evidence.
