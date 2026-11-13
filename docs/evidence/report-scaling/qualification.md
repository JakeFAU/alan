# Full report qualification and reproduction

The frozen comparison uses the following source identities.

| Source | Commit | Tree |
|---|---|---|
| Integrated baseline | `5f5d382c8d1e9c7369e7624e3a962746c9bf2229` | `cb6dc03624010d13f8af8d045af44e490110bfaa` |
| Tested candidate | `7679755c7298f229fa468b14da54bcfe1420c4f8` | `7aeb84be8acf414800a649152746aacd3e03b5e6` |
| Historical CLI | `af6943a5805a9a6c0e8fe77f804264f4765d07ec` | `5d1f64029c7fd764ddd1941bc77b831beb3bdcb3` |
| Historical owner | `d59eaf2976849e450635d252c7bca62eeccbef2e` | `dc082ea07e4c24d8f36e56d756771f7c1154b0ec` |

The candidate was checked out clean before the freeze. The report-local
production change is `6fadae8575bb1567f4d11f69edbb8258d5f3dce6`. Later
evidence commits retain the tested source label. Source archives, full
export inventories, actual binaries, commands, environments and
before/after input and output hashes are bound by the manifests.
[Source and build manifests](qualification-data/freeze-manifests.json) ·
[Exact recorded commands and exits](qualification-data/gates.jsonl) ·
[Command environments by digest](qualification-data/gate-environments.json)

The host was an Apple M4 (Mac16,10, 10 logical cores, 16 GB), with Go
1.27.4 darwin/arm64, system Python 3.9.6, Apple Clang 21.0.0, macOS SDK
27.2, macOS 27.2 (26B123), Staticcheck 2026.2.2 and cached go-sqlite3
v1.14.52. The recorded environment used `GOMAXPROCS=4`, `GOGC=100`, `GOMEMLIMIT=off`, CGO,
disabled VCS stamping and offline Go module resolution. It created fresh Go
and Staticcheck caches for this campaign. Native binding hashes selected
tool executables and SDKSettings.json and records the SDK identity. It
assumes the remaining SDK header and library bytes are stable on the host.

## Reproduction

Check out the exact tested candidate in a separate clean worktree with all
four commits available locally. The freeze checks that the worktree HEAD
equals the requested candidate. Run on a matching host with the
offline Go module cache, adequate space for the retained raw files and
loopback access for the test and recovery phases. Choose a fresh root and
fresh outside path. Run serially and leave the host idle during timing.

```sh
candidate=7679755c7298f229fa468b14da54bcfe1420c4f8
checkout=/private/tmp/alan-p5-reproduction-source
git worktree add --detach "$checkout" "$candidate"
cd "$checkout"

app=docs/evidence/report-scaling/campaign.py
root=/private/tmp/alan-p5-reproduction
outside=/private/tmp/alan-p5-reproduction-outside
digest() { shasum -a 256 "$1" | cut -d ' ' -f1; }

/usr/bin/python3 -B "$app" freeze --root "$root" --candidate "$candidate"
source_digest=$(digest "$root/source-freeze.json")
phase() {
  phase_name=$1
  shift
  /usr/bin/python3 -B "$app" "$phase_name" --root "$root" --source-digest "$source_digest" "$@"
}

phase build-timing
timing=$(digest "$root/timing-freeze.json")
phase run-timing --build-digest "$timing"
phase analyze-timing --build-digest "$timing"

phase build-counters
counters=$(digest "$root/counters-freeze.json")
phase run-counters --build-digest "$counters"
phase analyze-counters --build-digest "$counters"

phase build-qualification
qualification=$(digest "$root/qualification-freeze.json")
phase gates --build-digest "$qualification"
phase differential --build-digest "$qualification"
phase recovery --build-digest "$qualification"
phase outside-rebuild --outside "$outside"
```

Inspect each actual phase exit before proceeding. Each new run has its own
digests, paths, per-Store identities, clocks and measurements. The recorder
refuses to overwrite previous attempts. Keep any failed attempt with its
actual status. Use the separate [baseline route](baseline.md) only for its
excluded baseline controls.

## Go verification

Counts include subtest pass events. The two complete integration variants
each skip the opt-in historical
`TestPhase3B3BaselineS2CompletionExpectedFailure`. All required old-reader
and promotion tests ran. The baseline and candidate ordinary suites each
include three packages with no tests.

| Gate | Exit | Test pass events | Package passes | Test skips |
|---|---:|---:|---:|---:|
| Baseline ordinary | 0 | 1,412 | 18 | 0 |
| Candidate ordinary | 0 | 1,414 | 18 | 0 |
| Candidate integration | 0 | 2,683 | 18 | 1 |
| Candidate integration race | 0 | 2,683 | 18 | 1 |
| Candidate focused ordinary race | 0 | 17 | 2 | 0 |
| Candidate focused integration race | 0 | 68 | 2 | 0 |
| Independent consumer ordinary | 0 | 16 | | 0 |
| Independent consumer race | 0 | 16 | | 0 |
| Nested independent series module | 0 | 17 | 1 | 0 |
| Nested P2 measurement driver | 0 | 71 | 1 | 0 |
| Nested P2 differential | 0 | 0 | | 0 |
| Nested series-scaling driver | 0 | 10 | 1 | 0 |
| Nested series-scaling differential | 0 | 0 | | 0 |

Both roots build and pass ordinary/integration `go vet`,
`go mod verify` and `go mod tidy -diff`. The candidate matches the
frozen oracle and `gofmt -l` reports no files. The exact baseline to
candidate diff passes the whitespace check. All five nested modules
were tested, built and verified, and their tidy diffs were empty. The
two differential modules have no test files. The separate same-seed
qualification below uses the series-scaling differential route.

Staticcheck exits 0 with zero diagnostics in all 13 scopes. These are
ordinary, race, integration and integration-race for each root plus
the five candidate nested modules. The independent series module uses
the `series` tag. [Lint records](qualification-data/staticcheck.json) ·
[Qualification outcome](qualification-data/qualification-outcome.json) ·
[All commands and exits](qualification-data/gates.jsonl)

## Semantic and recovery evidence

Four B8 pairs use complete copies of one seed and the targeted approval
basis in both builds. Success and retained third refusal are each run with
zero or seven interval reopens. The exact comparator includes the
complete public cuts, typed evidence, History, Replay, original IDs,
contacts, approvals, charges and durable relational rows. It normalizes
source labels, Store paths and physical SQLite row order. The cases
contain 18 or 25 exported public cuts per arm, including eight ordinary
full RunReports and seven additional zero-progress RunReports in the reopen
cases. The intentional
`OpenRuns` array-to-null mutation returns exit 1 and identifies the exact
field. The refusal cases retain the paid, selected third refusal, its
unresolved consequence and eight original contacts. The final views
show eight selected and eight spent, with eight completed in the success
cases and seven completed, one failed and one unresolved in the refusal
cases. The approved fourth uses a different operation ID and references
the original paid third
evidence. [Comparison outcomes and cut hashes](qualification-data/differential.jsonl)

The weather qualification uses an independently rebuilt public consumer
and a controlled loopback provider. For each build it retains ten B8
acknowledged or provider-witnessed process kills and three stops in one
repeated interruption history. The test reaps each actual SIGKILL and
immediately seals all three SQLite members before a reader can open the
Store. Each build reads a separate complete copy of each stopped Store.
The checks include the original approval pre/post-commit, selection,
authorization, reservation, gate, accepted response lost, result retained,
settled and terminal cuts. The same stopped input yields equal typed
evidence, History and Replay in both readers. The repeated history
retains one original contact and one charge through three interruptions.

The endpoint journal proves which requests reached the controlled
provider. It records acceptance of the original third request with the
response held. ALAN retains that original as paid and unresolved, with
no selected observation and no repeated third request. In the fourth
reservation and gate cuts, four local charges coexist with three provider
contacts. The reservation cut has zero unresolved outcomes and the gate
cut has one. After the separately approved fourth operation in the
lost-response history, there are four contacts and charges, three
completions and selections, and one unresolved original. Both builds
preserve the independently approved successor and leave future positions
unapproved. These counts distinguish local
charge/attempt state from endpoint receipt and response evidence.

Four additional nested-demo histories exercise opposite-build resumes in
both writer directions, at `execution.staged` pre-commit and
`execution.committed` post-commit. Each writer acknowledges the exact
cut and is reaped with status −9. Independent copies resume the same
stopped input under both builds. The quiescent RunReports are byte equal,
and the exact durable comparisons pass. The 26 weather and four nested
captures retain every main/WAL/SHM member and its copy digest. These are
controlled process stops on one local filesystem.
[Kill, reap, reader and comparator records](qualification-data/recovery.jsonl)

## Historical and independent consumers

The historical CLI initializes, inspects and replays its supported Store.
The historical owner opens a genuine format-13 K01 parent, renews only
its open `b/1` run and returns the original quiescent cut with zero
steps and dispatches. Genuine K01 and K02 compatibility and promotion
tests pass. The promoted formats 14 and 16 retain the consumed original
result and conflict evidence. Those controls record zero additional
endpoint applications. The four old-reader rejection subtests actually
pass with the supplied old CLI and old-owner binaries in both ordinary
gates and both complete integration gates.
[Historical positive](qualification-data/historical-positive.json) ·
[Test events](qualification-data/test-events.json)

The independent eight-operation consumer runs in separately compiled
ordinary and race builds. Its public Create, Run, Inspect, Open and
Replay checks retain eight original contacts, charges, selections and
results. It reads the archived formats 4 and 9 with unchanged main
bytes and empty WAL. A separately discovered Go test performs genuine
format 11 to 12 promotion on a copy. The baseline-linked reader opens
and replays each original retained series with zero contacts. The
candidate's format-12 retained view exposes the independent conflict.
[Consumer and reader logs](qualification-data/toolchain-and-public-logs.json)

## Outside rebuild

The candidate Git archive was independently exported under
`/private/tmp/alan-p5-final-outside-7679755-91686728/source/` and the
ordinary public driver built there against that export. The completed
B8 proof validates eight original contacts, charges, selections and
results with complete reports, followed by independent inspection,
completed reopen and Replay. The outside directory and retained copy
have matching file inventories.
[Outside binary, report and file hashes](qualification-data/outside-reproduction.json)

## Delivery source check

Before committing this package, the staged diff contained only the two
README updates and added report-scaling Markdown and JSON/JSONL evidence.
A byte comparison checked the full 756-file candidate export against its
frozen inventory and verified the 754 files other than the two readmes
against the delivery worktree. This includes tested source, tests, fixtures,
module files and the consumed apparatus. The same check rehashed 22
selected source/tool inputs, eight Go toolchain files, 11,965 Go standard
library source files and 106 cached SQLite files. The source and build
labels continue to name the exact frozen candidate.

## Retained artifacts

The compact package ships all timing and diagnostic row resources and
actual exits, strict analyses, semantic comparison results, process
witnesses and complete-copy records, named test events, lint results and
source/build provenance. The large raw captures remain local at
`/Users/jacob/.codex/worktrees/recoverable-investigator/alan/.superpowers/sdd/plan-report-scaling/task5-final/`.
Within that root, `archives/` keeps the four complete Git tar archives,
`sources/` keeps their exports, `builds/` contains generated modules,
overlays and binaries, and `samples/` and `gates/` contain the raw stores,
unabridged reports, command logs and full before/after inventories.
`historical-controls/`, `differential/`, `weather-recovery/`,
`cross-build-recovery/` and `external-route/` hold the captured evidence
and independent working copies. The original outside proof is at
`/private/tmp/alan-p5-final-outside-7679755-91686728/`; its verified retained
copy is `outside-reproduction/` under the campaign root.

The complete local `task5-final-raw-index.jsonl` sits beside the raw root
and gives the relative path, SHA-256 and byte count of 18,419 retained files.
The tracked custody record gives its absolute location and SHA-256, per-group
index hashes and counts, and individual hashes for the source archives and
phase manifests. The archive hashes and bound inventories cover the source
exports. The index excludes compiler and lint caches and transient working
files. The raw files and full index remain local. The tracked compact records
and commands above provide the inspectable and reproducible package.
[Raw custody manifest](qualification-data/raw-custody.json) ·
[Excluded and rejected controls](rejected-variants.md)
