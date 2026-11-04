# Bounded-series compatibility and performance

The exact producer successfully constructed and executed bounds 1, 8, 32 and 256 through public APIs. Its complete lowered Provision is byte-equal to an independently authored canonical manual expansion. The old generic runtime reads, replays and reopens genuine new-helper stores in formats 11 and 12. No provider HTTP request occurred in these measurements.

The existing runtime scales poorly for large independently approved deployments: the final 256-position execution took **364.275 seconds and 406.697 GB of cumulative allocation traffic**. This is not peak resident memory. The author helper adds no execution protocol layer, and this milestone does not optimize the inherited runtime scans. Do not describe accepting bound 256 as a low-latency qualification.

## Exact source and environment

Final producer commit `8a5f2ef564d5d750afa506675431d414bf92fb65`, tree `8caba38e10658672dc7f597751d322332f7d94bc`, immutable export of 311 files. Its source-manifest SHA256 is `ca95102a18437722cf61cce7dcf03ac7a98c2283266e193816ac4351c08ac048`; archive SHA256 is `2bdb98a8c2630ae01e3a3d09338fe623aed979d2c2292a09e5b1c4ca11d5cf8a`. The manifest was independently hashed, and the producer commit's tree was independently checked. `manifests/producer-code.sha256` records all 276 Go production/test and module files for final-deliverable comparison.

Verified base commit `1d0ef3c91589c3df69b5a004cd5362438e8eed46`, tree `c7266108c1e8913534781a066defe93f4057b8dc`. All 179 source/test files across `internal/executor`, `runtime`, `effects`, `schema` and `adapters` are byte-identical between base and producer, including the symmetric file set. New constructor SHA256 is `d7a4110ef87b045d4abc694ee90938b1933708802755e89bdba29d498af619ae`. The final constructor includes fail-fast checking that JSON-escaped predecessor identities fit the existing 4096-byte continuation field; earlier candidate samples are not substituted for this run.

Go 1.27.3, darwin/arm64, Apple M4, Darwin 27.2.0; CGO SQLite v1.14.52. Every benchmark uses `-cpu=1`. Commands used `CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local` and the offline caches recorded in `manifests/commands.json`. No custom GOGC/GOMEMLIMIT/GOEXPERIMENT/GOMAXPROCS variables were inherited.

Benchmarks ran serially in this reviewer. The host also carried other agents' functional tests and reviews, particularly ordinary/race work near the start. Full execution has one expensive sample per bound; construction and view have five 100 ms samples and report medians. These are empirical measurements, not statistically established speed improvements or regressions. No benchstat significance claim is made.

## Construction and own-view costs

All alternatives below ran against the final producer in the same command. They obtain a complete Provision on the controlled one-input, one-adapter observation. `Manual cached` compiles original/successor artifacts once and explicitly expands the Application, roots and continuation slots. `Manual loop` mirrors the historical weather compile-per-position pattern. The old weather application itself accepts at most eight observations; synthetic 32/256 controls are not claims that its old CLI accepted those bounds. At bound 1 the cached-artifact comparator unnecessarily compiles a successor, so the loop comparator is also shown.

| Bound | Manual cached µs | Manual loop µs | Series + Provision µs | Series bytes/op | Series allocs/op | Provision copy µs |
|---:|---:|---:|---:|---:|---:|---:|
| 1 | 129.74 | 75.62 | 130.24 | 155374 | 1331 | 12.43 |
| 8 | 410.54 | 731.94 | 541.90 | 625646 | 5277 | 74.13 |
| 32 | 1286.78 | 2914.68 | 1651.43 | 1840812 | 15415 | 254.22 |
| 256 | 9237.77 | 23394.14 | 11838.61 | 12672158 | 109929 | 1897.47 |

The helper additionally validates grammar, inputs, encoded resource limits and representable continuation identities, and owns its declaration and returned provision. The table exposes that work rather than promising a construction speedup. Larger-bound allocation growth is approximately linear for this fixed definition/input size.

The own view consumes an already validated detached snapshot; it does not re-read the store. Five-sample final-source medians:

| Bound and cut | View µs | Bytes/op | Allocs/op |
|---|---:|---:|---:|
| 1 provisioned | 1.65 | 1304 | 19 |
| 8 provisioned | 20.45 | 26834 | 193 |
| 32 provisioned | 93.89 | 138401 | 785 |
| 256 provisioned | 732.45 | 1101467 | 6317 |
| 8 completed | 26.46 | 30288 | 265 |

`Series.Inspect` compares sets with maps once and joins indexed runs/effects/observations/roots/reductions, with binary-search approval lookup. Expected own work is O(total declaration plus N log N), rather than a full-store scan or compilation per position. Ordinary snapshot validation is measured separately below.

## Complete execution and final reads

The driver stops at quiescence after each observation. For each successor it obtains a new validated snapshot, derives the existing ContinuationBasis, explicitly approves once, and resumes the ordinary runner. Execute excludes Create and includes those approval-related public operations. Its adapter executes entirely in process, returns one typed retained sample, and asserts exactly N entries. No HTTP/socket implementation exists in the driver.

| Bound | Create ms | Execute ms | Execute bytes/op | Execute allocs/op | Final Inspect ms | Replay ms | Completed-view µs |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 5.31 | 3.98 | 1729648 | 21219 | 1.38 | 2.60 | 26.17 |
| 8 | 10.25 | 129.96 | 83909240 | 801109 | 5.14 | 18.88 | 41.79 |
| 32 | 35.26 | 2151.85 | 1843636768 | 12476307 | 17.42 | 76.14 | 107.13 |
| 256 | 254.94 | 364274.95 | 406696928992 | 1477297998 | 135.05 | 867.07 | 813.96 |

The last column is a single sample after each completed execution; the repeated view table above is the stronger own-view timing evidence. Provider usage and monetary costs are not inferred from these local measurements; the controlled adapter supplies no usage counters.

The original base/manual samples remain in `raw/baseline-*`: execution at 1/8/32/256 was 4.15/112.62/1880.98/368984.42 ms, allocating 1.70/75.80/1504.33/404097.47 MB cumulatively. Their synthetic approval names were shorter than the final helper names, so raw allocation/timing deltas do not isolate helper overhead. Independent canonical manual expansion using the **same** generated names yields complete Provision JSON byte equality at all four bounds. The generic executor sources are unchanged.

Existing repeated work is visible in the owning source:

- `internal/executor/runner.go` re-reads pending intents/observations and scans run IDs/open runs each turn, decoding the same programs for waiting runs.
- Its `settleRoots` call runs every turn. `composition_root.go` decodes the composition descriptor and calls PrepareRoot/SQL for every root, including published roots.
- A new unapproved ContinuationBasis scans declared slots, and the public approval loop first validates a fresh full-store cut.

These observations establish inherited repeated full-store/root work and measured superlinear complete execution. They do not prove an exact asymptotic exponent or attribute every allocation without profiling. Executor optimization should be a separate measured change.

## Compatibility controls

All final controls passed on the exact producer; raw logs are `raw/final-*`.

- Public external-module compilation and complete canonical lowering equality at 1/8/32/256 passed. Fresh generic stores at those bounds select existing formats 9/11/11/11 and InspectExecution/Replay/Open/Close successfully without an adapter registry.
- Genuine historical format-4 provisioned/missing/present lifecycle archives and archived weather format 9 remain readable/replayable. Main database bytes are unchanged and no nonempty WAL is written by InspectExecution/Replay. The application retains its separate historical program recognizer; generic old-reader compatibility is not a promise that an old weather CLI accepts new programs.
- A new helper-produced eight-position store has eight original effects, eight selected/completed results, eight paid opportunities and eight controlled adapter entries. Its namespace is `a3aecd65be3ffe50f7e2461839879ec1357df8da42fbd68630ce8433b60a723c`.
- A generic reader binary built only against the immutable base successfully InspectExecution/Replay/Open/Closes disposable copies of that actual new-helper format-11 store and a genuine format-12 promotion. Reopening is tested on disposable copies because owner renewal writes authority facts.
- Format 12 is produced by opening a new-helper store copy and offering a valid alternate result through the existing production `recordAdapterResult` writer. The real writer returns ErrConflict, retains one losing receipt and promotes 11→12. No SQL header changes or fabricated history create this positive control. Original digest `9c37771827b9027631a44a6391874c7f15870eece0da041b40d6ddcffa7f21c9` remains the selected result.
- New Series.Inspect accepts both genuine stores. Format 11 reports selected/completed/spent 8, pending 0, unresolved 0. Format 12 preserves selected/completed/spent 8 and pending 0 while independently reporting unresolved 1 for conflicting receipt evidence. No aggregate success outcome or remote-finality claim is invented.

Formats 3/4/5 and dispatch 9/10 cannot express the protected multi-position continuation protocol. New multi-position creation uses existing format 11; losing-receipt retention can promote it to 12. There is no migration in this authoring change. A coherent historical deployment is not thereby the requested Series declaration.

## Reproduction and source alignment

`README.md` and `reproduce.sh` describe the portable external module. Its go.mod deliberately contains no local replacement; reproduction copies it and runs `go mod edit -replace` against the selected immutable export. The final reproduction script itself passed, including construction/control, full all-bound runtime, read-only archives, canonical equality, repeated views, genuine 11→12 promotion and reverse base readers. Exact argument arrays and environment are in `manifests/commands.json`; raw source-labeled logs and derived values are retained. No production files were edited by this reviewer.

Earlier candidate tree `4bd4e8e` and corrected-task1 tree `4e1bdb07` logs remain separately labeled. They preserve prior evidence and are not relabeled as final producer measurements. Earlier independent failure-classification and long/escaped-identity REJECT findings remain part of the controller's review history; this report does not waive them.

Final-deliverable source confirmation and compatibility/performance APPROVE remain pending the controller's final immutable pointer. The producer measurement is complete; final confirmation must show all 276 owning production/test/module files unchanged, with added code confined to the disclosed evidence drivers.

## Evidence packaging correction

A final packaging check reproduced a root-module build failure: the overlay writer
payload named `overlay/promote_series_test.go` was discoverable outside the driver
submodule, where its executor-internal symbols are unavailable. The RED output is
retained in `raw/packaging-overlay-red.log`. The payload is now
`overlay/promote_series_test.go.txt`, and reproduction maps its unchanged bytes to
the same virtual executor `_test.go` path. Its source-content SHA256 remains
`e3d59a5470806bb5f06c231b66ff9dfeb444d0a10f9da3bff5fa7e88e886c749`.

A fresh disposable format-11 copy was genuinely promoted to12 through the renamed
.txt payload and the same production writer; InspectExecution/Replay and original
receipt preservation passed (`raw/packaging-txt-promotion.log`). Root `go list ./...`
with only the obsolete payload hidden by a Go file overlay discovered20normal
packages and no evidence package (`raw/packaging-root-discovery.log`). The renamed
.txt payload is ignored by ordinary Go package discovery; the public driver has
its own nested go.mod. The controller must remove the obsolete .go file when copying
the corrected package. No producer source or measurement function changed, and no
full256benchmark was repeated for this packaging-only correction. The portable
go.mod also had one redundant trailing empty line removed after measurement; its
module/dependency declarations are unchanged. Final raw samples retain their actual
producer/driver identity rather than being relabeled.
