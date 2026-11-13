# Frozen baseline and apparatus

Task 1 changes documentation and disposable measurement tooling only. Production
source remains `d59eaf2976849e450635d252c7bca62eeccbef2e`, tree
`dc082ea07e4c24d8f36e56d756771f7c1154b0ec`; the branch's frozen contract/plan
commit before this task was `85bcdeeeb70a572390ce732b0ac855b5d5e1a917`.
The initial clean-worktree/PR snapshot records P3 PR #14 open/draft and main
`926a899b651fccc48be52a054612a92afcf1ddfd`; this is a dated initial snapshot,
not a delivery-time PR check. No production/API/schema/dependency edits, merge
or publication occurred.

The clean `git archive` export is outside the worktree at
`/private/tmp/alan-scaling-ZbqVzA/baseline-source`; archive SHA-256 is
`a1c1b2facfed2aa36ab1bede3376978173174ab9a2a7807cbd61d42f802d6ae2`.
[baseline-data.json](baseline-data.json) binds the source inventory, tools,
binaries, exact row commands/exits/raw-output hashes and environment. The new
raw root is `/private/tmp/alan-scaling-ZbqVzA`. Original P2/P3 paths and the
frozen P3 raw archive were not modified.

## Host and ordinary suite

Go 1.27.3, darwin/arm64, Apple M4, macOS 27.2 (26B123), Apple clang 21.0.0.
The pinned go-sqlite3 v1.14.52 actually links bundled SQLite 3.53.4, source ID
`2026-07-24 19:02:57 bf7c7f30031888f4e796e429ab3978879485813aaca6f641c7b33e4e09459bcc`.
Native compile options and executable/build identities are retained under
`environment/`. Runtime configuration is `GOMAXPROCS=4 GOGC=100
GOMEMLIMIT=off GOWORK=off GOTOOLCHAIN=local`, CGO enabled. Direct deterministic
single-in-flight transport makes no provider or HTTP requests.

The baseline command `go test -json -count=1 -timeout=30m ./...`, with the fixed
runtime environment, exited 0: 1,304 test pass events, 18 passing packages,
three packages without tests and two actual frozen-old-reader skips
(`TestRequestAuthorityFormatsConflictPromotionAndOldReader/{13,14}/frozen_old_reader`).
The original sandbox attempt was blocked on Go build-cache access and is
retained separately; it is not counted as a runtime failure or a pass.
No Task 1 source change justified rerunning that whole production suite.

## Descriptive pilots and structural diagnostics

The original public P2 driver was built against the clean export and run once
at each bound below. Every process exited 0, including exact original identity,
contact, paid/selected/completed, protected-approval, 4B−1 reduction, full returned
cut/prefix, final inspection/proof, completed reopen/no-contact resume and Replay
checks. Creation, initial/final audits, reopen and Replay are separate stages.
Execute includes all B Run calls and B−1 full snapshot/basis/approval calls.

| B | Execute seconds | Go cumulative allocated bytes in Execute | Lifetime peak RSS bytes at Execute end |
|---:|---:|---:|---:|
| 1 | 0.004965 | 1,601,888 | 21,676,032 |
| 32 | 0.801285 | 641,794,664 | 32,473,088 |
| 128 | 11.278570 | 10,071,123,136 | 52,822,016 |
| 256 | 46.384300 | 42,187,364,600 | 74,711,040 |

These single descriptive pilots are **excluded from confirmatory samples**.
An initial overlay compilation overlapped the pilot batch; no pilot timing is
used for inference, acceptance or an old/new comparison. CPU, allocation count,
GC, stages, prefixes and process RSS remain in every raw output. No speedup or
memory bound follows from this table.

Separate final compact-counter processes passed at B1,8,32,128,256. Their
Execute totals are actual counter deltas, not timing inputs:

| B | SQL VM executions | SQLITE_ROW returns | Fullscan steps | VM steps | Observation-loop visits |
|---:|---:|---:|---:|---:|---:|
| 1 | 332 | 231 | 0 | 5,085 | 1 |
| 8 | 6,197 | 6,950 | 3,626 | 120,189 | 792 |
| 32 | 61,961 | 88,910 | 66,650 | 1,364,180 | 54,859 |
| 128 | 836,827 | 1,329,135 | 1,833,626 | 21,683,648 | 3,756,812 |
| 256 | 3,246,427 | 5,255,119 | 11,515,290 | 97,664,128 | 29,358,102 |

B256 also records 360,320 run visits, 178,601 readiness bindings, 1,023
readiness reductions and 1,023 Commit reductions; 392,960 settlement-root visits;
65,536 report-root visits; 8,421,376 report root×intent checks; and 229,760
pending-delivery intent visits. The decisive original P3 scanner/SQL counts are
reproduced. Per-bound and per-bound-squared normalizations and first/median/last
contiguous prefix increments are retained in baseline-data; full per-position
scalar deltas remain in raw JSON. SQLite rows returned are not rows examined.
Unknown transaction scope and unavailable connection-setup internal VM work
remain limitations. No new profiles were needed: existing P3 attribution plus
these exact scanner counts establish the present hypotheses.

## Apparatus qualification and boundaries

The [compact overlay](apparatus/README.md) reuses hash-pinned P2 tools with exact
input inventories and explicit work anchors. The original proof file is byte
identical. Snapshot is the default; targeted is a runtime interface assertion,
so the baseline builds without the future API. Actual invalid mode exits 2;
actual targeted B1 exits 1 with the explicit unavailable-API error and no contact.

Eight new Python tests passed; all three original generator-anchor tests passed
with the pinned SQLite source; the copied SQLite compact-summary arithmetic test
passed. Original/full versus compact counters at B1/B8 matched respectively
3,873/13,750 common stage/prefix values and exact Execute family/scope/scalar
partitions, with no attribution overflow. Both independent public proofs passed.
Fresh namespaces differ: this is not a same-seed raw-ID/durable-state equality
claim. Later differential qualification remains necessary.

Only fixed SQL/scope/work vocabulary is copied per resource snapshot; two full
registry snapshots produce one terminal Execute delta. B256 raw output is
24,615,361 bytes, without cumulative family maps at every prefix. Generation
records adapted generator source, patches and pre/post-format inventories;
rows retain stdout/stderr, argv/cwd, whitelist, binary/source before/after checks
and exits. Failed/incomplete invocations and build-only iterations are retained
in `failed-attempts.json`, including the host-Python extraction API mismatch,
blocked CPU query, corrected summary-test invocation and original no-env skip.

The [invariants](invariants.md) freeze lazy corruption/order boundaries, exact
completion identity, closure/freeze/publication precedence, transaction rechecks,
continuation-history/stale-read rules and the rejected persistent-cache scope.
B calls returning B roots and B−1 complete approval snapshots retain a quadratic
output/materialization floor. Call-local bulk reads can remove unrelated probes
and cross products while leaving material quadratic row work; later evidence
must report that honestly. No confirmatory schedule or candidate outcome is
claimed by Task 1.
