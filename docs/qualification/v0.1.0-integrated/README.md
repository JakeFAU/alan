# v0.1.0 integrated release gate

The proposed release source is [`57c230f8d1ac1a0bac4cc3b36f7b8e6b46775530`](https://github.com/JakeFAU/alan/commit/57c230f8d1ac1a0bac4cc3b36f7b8e6b46775530), tree `0f0d2ba44db71461701afbc819743bcc15bbeb72`. It contains all four reviewed PR heads and the final documentation and distribution checks. The proposed tag is `v0.1.0`. The intended consumer is an independent Go developer building a bounded, single-host agent. [Proposed release notes](../../releases/v0.1.0.md) · [Four-process quickstart](../../../README.md#build-your-own-durable-agent)

## Integration and independent review

| Input | Head reviewed | Decision and effect on qualification |
| --- | --- | --- |
| [#29, Go setup](https://github.com/JakeFAU/alan/pull/29) | `6ca7a1879f962134c4c55b653d1f25f649eb459d` | APPROVE. The module minimum remains Go 1.27.3. Go 1.27.4 is recommended. Six setup instructions changed, requiring a new distribution checksum. |
| [#30, scheduler evidence](https://github.com/JakeFAU/alan/pull/30) | `e1f4dfc209befc8bd13d20ab03d9450d2a483f32` | APPROVE. It records finite work counts without changing scheduling. All eleven workloads reproduced with the integrated engine. |
| [#31, SQLite binding](https://github.com/JakeFAU/alan/pull/31) | `6e7b765b92070ddc48462dbae003a7ff7caceac3` | APPROVE. go-sqlite3 v1.14.53 selects SQLite 3.54.1. This changes both Go sums and the storage engine, so scoped and historical readers, both crash directions, dispatch recovery and the race suite were rerun. |
| [#32, RC-001](https://github.com/JakeFAU/alan/pull/32) | `bea0acc86d9a7302ccc3511327455922503c4a57` (source), `41400512d993d2e1470c5779867a1787a75dd83e` (evidence) | APPROVE. The independent review re-exercised external installation, crash-safe creation, adapter output and scoped assets. Its original qualification used SQLite 3.53.4. |

GitHub merged #29, #30 and #31 in that order. Their merge commits are `7028aaf`, `5d8c528` and `992c10e`. RC-001 then incorporated that main revision at `2943fc3`. [#33](https://github.com/JakeFAU/alan/pull/33) supplies the final corrections and qualification on top of #32. The scheduler measurements and setup guidance can be reviewed independently. The final operational qualification depends on combining #31 with #32.

Independent review identified two medium documentation findings on the combined tree. One merge duplicated six performance rows and left seven links pointing to files excluded from the Go module. The extraction guide also prescribed exact byte comparison between executables reporting different engine identities. The integrated changes restored unique rows, pinned the archived links, and distinguished same-engine comparison from the checked cross-engine procedure. Review verified all 225 relative references in the 41 shipped Markdown files and the nine pinned blob references it examined. **Outstanding findings by severity: critical 0, high 0, medium 0, low 1.** The open item is the measured missing-target inspection cost below. The independent dispositions are recorded here and in the [API, scheduler and documentation review](independent-review.json) and [storage review](storage-results.json). GitHub branch-protection approvals are a separate repository operation.

## Qualification at the proposed source

The complete Go documentation for nine public packages and an additional importable example matched RC-001 byte for byte. The production Go source diff against RC-001 is empty. The binding, its tests and the shipped module differ, which is why the previous candidate manifest cannot authorize publication of this revision.

| Integrated verification | Result |
| --- | --- |
| Ordinary | 1,764 named tests and subtests passed, 0 skipped, 136.45 s. |
| Integration | 3,147 named tests and subtests passed, 1 skipped, 315.15 s. |
| Race-enabled integration | 3,147 named tests and subtests passed, 1 skipped, 3039.07 s. |
| Build and static analysis | Module checksums, tidy diff, gofmt, build, ordinary and integration vet and Staticcheck passed on Darwin/arm64, Go 1.27.4, CGO, Apple clang 21, using a local SQLite WAL filesystem. |
| Independent application | From empty module/build caches on Darwin/arm64 Go 1.27.4 and hosted Linux/amd64 Go 1.27.3, installed the tagged candidate from an isolated file proxy. Verified 16 external Go documentation targets. A ticket suspended with zero paid attempts, a second process registered its adapter and dispatched once, a third inspected the expedited queue, and a fourth replayed five commits and one intent offline. |
| Store formats | Exact archived 3–16 inputs verified; both supported engine builds inspected/replayed 28 independently written scoped fixtures, two per format 17–30. The pre-scope reader rejected each scoped fixture. The historical comparison covered 60 matching exports for 3–12, both writers for 13–16 and 18 SQLite integrity and foreign-key checks. |
| Paired crash recovery | All 12 acknowledged SIGKILL schedules passed. Three older binaries (public base, RC-001 and frozen research) each wrote and read against the integrated binary on both sides of a commit. Recovery matched 26 committed transitions and four intents, preserved database/WAL bytes on read, and refused injected corruption. Three cross-version corpus quickstarts each resumed with one dispatch and replayed six commits and one intent. |
| Scheduler | The 11 stock and instrumented workloads reproduced the archived count records after checking both SQLite engine identities. At the 64-run/64-observation example the fallback made 4,096 observation checks. No new latency or workload-frequency claim was qualified. |

Every test invocation used `-count=1 -timeout=60m`, the pinned historical readers and fresh promotion outputs. The race run used `GOFLAGS=-p=2 GOMAXPROCS=4 GOGC=200` to control parallelism and `GORACE=atexit_sleep_ms=0` to omit the detector’s post-exit wait in the many child processes. Each suite covered 23 packages, including four without test files. The opt-in historical expected-failure falsifier was the sole named skip in each tagged run. The compressed event files record the suite's named verdicts. [Full commands, counts, checksums and controls](gates.json) · [Cross-version reproduction](storage-reproduction.md) · [Scheduler counts](scheduler-rows.jsonl)

The runtime root now selects go-sqlite3 v1.14.53. All 16 research module files declaring that dependency at research main `19b6d2f` retained v1.14.52, including the root. The historical extraction binary was built with its original files. The 490 original runtime evidence blobs, previous qualification packet, archived scheduler rows and original binding qualification remained unchanged. [Research pins](research-pins.json)

## Integrated performance

The original RC used the previous SQLite binding. We repeated its controlled, 256-operation finite series against the original public baseline on one Darwin/arm64 host. All 20 runs (four excluded warm-ups and sixteen measured samples) checked 256 paid attempts, 255 approvals, 1,023 reductions, reopen and replay. Four balanced blocks gave full Run medians of 4.700 → 4.810 s and RunProgress medians of 1.327 → 1.392 s. Median within-block changes were +2.12% and +4.82%, respectively. The comparison includes all release changes and each revision’s pinned engine. [Measurement summary](performance/performance-summary.json) · [Reproduction](performance/performance-reproduction.md)

New-store validation and durable initialization cost time. The one-step Create/Close median went from 2.063 to 7.266 ms without assets and from 3.333 to 7.357 ms with a 128 KiB asset. Across the nine causal recovery benchmarks, the largest movement was missing-target inspection, from 2.363 to 2.986 ms (+26.35% comparing medians). All three paired blocks rose. The cause, production frequency and behavior on larger inputs remain open. This is a low-severity performance follow-up. No correctness check failed and the release contract specifies no latency threshold. The eight other recovery medians ranged from −4.83% to +5.66%. [Measurements for all 66 rows](performance/performance-creation-recovery.json)

## Distributable and publication boundary

The integrated ZIP contains **488 files, 2,694,714 bytes**. The two hosts independently produced identical ZIP bytes, canonical Go sums, content inventory and application results. The verifier checked the license, public sources, example, release notes, two historical fixture digests and absence of archived campaign directories. [Darwin candidate manifest](candidate-consumer.json) · [Linux manifest](linux-consumer.json) · [Hosted run](https://github.com/JakeFAU/alan/actions/runs/45456077057)

| Identity | Value |
| --- | --- |
| ZIP SHA-256 | `91950f98ec4b28a6e549fc12e86998ff9ba79ae75aae7e7a72694cf6caa76dca` |
| Inventory SHA-256 | `1cdbc7b014139f747d12264cbf8fcd7050217cc05d3fb264a695d62308dfdb4b` |
| Go module | `h1:u7XheIMCn3qfRnCF9zelOR9zJDKC9LEC9KvrukIKan8=` |
| go.mod | `h1:bcCHLYgenqOFuEiadL70fQjYGYCi7XaIwZXW7y69Iy0=` |

The supported model is one owning process per store, on a single host with working local locks and sync. Ordinary external effects can remain unknown after interruption. Physical power loss, device failure, network filesystems and distributed execution are outside the qualified fault model. Linux coverage is the packaged application. The earlier time measurements still describe their original binaries. Wall time for fallback scans and guarded candidate copying, plus creation latency and peak memory for larger provisions, are deferred. [Compatibility policy](../../releases/compatibility.md)

**Ready for the v0.1.0 publication decision.** This report records pre-publication checks. Publication requires explicit authorization for a named commit. If maintainer integration produces a different Git commit, establish tree equivalence and retain a passing consumer manifest for the exact new SHA before tagging. The command below applies if the authorized tag points at the source recorded at the top of this report. For a separately verified main commit, use its exact Git-commit manifest. After publication, run the verifier against the manifest for the authorized tag target. It checks the Git tag first, then installs from the public Go proxy with the public checksum database and compares the actual contents. Preserve the published tag. Any repair uses a newly qualified patch release.

```sh
python3 tools/qualify_consumer.py --published docs/qualification/v0.1.0-integrated/candidate-consumer.json --output published-v0.1.0.json
```
