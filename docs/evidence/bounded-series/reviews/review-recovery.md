# Independent Task 2 recovery and weather review

Verdict: **APPROVE Task 2 frozen weather integration and recovery implementation, for the reviewed source and bounded controlled evidence only.** No blocking Task 2 spec or quality finding found. This is not approval of the whole milestone or of a later integrated source. The separately reported author long/escaped-ID usability REJECT remains outstanding until its own correction and independent re-review; this review does not waive it.

## Source and scope

Reviewed the supplied user specification, task-2-brief.md, task-2-report.md, task-2-diff.patch, and actual source under `task2-frozen/`. Independently recomputed all 311 SHA256 entries in `task2-frozen-manifest.json`: zero mismatches. Package base is `220312ea17bb4014763e23a0a22237683bcdc910`, with weather overlay. Primary files: program.go, legacy_program.go, deployment.go, approval.go, report.go, new series application/process/corruption tests, and existing process/provider/history/barrier helpers. Reviewed author/series_inspect.go to assess application use of its evidence semantics; this is not a substitute for the separate author API review.

Production weather changes correctly move positive-bound declaration/provisioning to BoundedSeries without an application-owned position/root/slot loop or successor approval prefix. Existing observation validation remains. Zero bound remains the historical no-spend route. The count-one artifact and 15 reductions are asserted. No application SQL or second durable scheduler was added.

Independently compared the legacy compiler/provision bodies against git base: exactly equal after function renaming. All four historical weather testdata files are byte-identical to base; no preexisting Test function was removed. Current test-only slot lookups compare against the freshly compiled exact source declaration while retaining the admission/selection/scope proofs.

## Evidence independently checked

The completed ordinary run is `task2-weather-final.log` with immutable `weather-fixtures-final/`, matching its recorded source copy/manifest in the developer report. Parsed all 11 SERIES_CONTROLLED rows: one ordinary eight-position reference plus ten interruption cases, 46 accepted localhost GETs and 48 charges in aggregate. These totals include the reference and are controlled-fixture evidence, not live Open-Meteo evidence.

Copied all 21 retained SQLite DB/WAL pairs to `/private/tmp/review-series-db-_38siqf0/` before review. Read the actual SQLite records from the copies. Ordinary eight has eight runs, eight original adapter reservations, eight gates, eight results, eight settlements, seven selected local approvals, and eight selected adapter observations. Stopped/final counts match each declared cut.

Ran an independent targeted public-reader check from the exact frozen source, using scratch harness `review-retained.go`:

```
CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local \
GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache \
GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache \
go run /private/tmp/alan-bounded-series.m9VryKAu/review-retained.go
```

Exit 0: all 21 retained DB/WAL pairs pass runtime.InspectExecution and runtime.Replay, with namespace/operation-count agreement and byte-identical database and WAL hashes before/after each pair. No adapter was configured, and no provider network or broad test suite was run by this reviewer. Ordinary eight replayed 127 committed reductions; representative final lost-third history replayed 49, retained-third final 63.

## Recovery and authority findings

- `barrier_process_test.go:136` checks ACK PID, token, scenario, exact store, target run/generation, boundary/phase, requested sequence, operation, and approval identity. `:154` issues actual SIGKILL, waits for reaping, and requires a signaled SIGKILL WaitStatus. Causal cut proof uses inherited pipes, not sleep timing.
- `process_test.go:24` builds production weather files in an unrelated public Go module. Ordinary runs, reads, replay and fresh recovery owners use that uninstrumented production consumer; only invisible internal-cut owners use the test binary. The acceptance/lost-response cut uses the production binary and independent endpoint gate.
- `series_process_test.go:88` exercises ten cuts around positions three/four, retains earlier original effects, performs two fresh resumes, duplicates explicit fourth approval, probes out-of-order fifth, and checks positions five through eight remain without approvals/effects/observations. Final fifth is awaiting approval and sixth through eighth are not eligible.
- Actual lost-third final DB retains four original reservations/gates but only three results/settlements: positions one, two and separately approved four succeed; third remains original, charged and unknown. No fabricated third selected observation or refund appears. Its raw provider record has third acceptance with response held, then separate fourth contact.
- Actual fourth-reservation cut retains four reservations and only three gates/results. Recovery retains its charge without contact or external uncertainty. Fourth-gate cut retains four gates, three results and three contacts; its unknown outcome is a local conservative bound, not an assertion that the provider accepted it.
- Actual retained-third pre-settlement cut has three results but two settlements; settled cut has three settlements but only two selected adapter observations; terminal cut has three selected observations while local root publication remains pending. The test correctly expects Selected at terminal reduction and Completed only after recovery settles local obligations.
- `history_process_test.go:180` checks typed/CLI inspect/replay agreement, history and pending views, exact DB/WAL hashes and independent provider-contact log stability. The reviewer reader check additionally validated all retained archives offline.

## Application reporting, matching and adversarial checks

`deployment.go:16` accepts exact Series.Inspect or exact genuine historical provisioning. It does not relax native request/response evidence verification. The runtime-valid foreign approval declaration test separately proves exact source matching rejects a changed declaration and cannot escape through historical fallback. Historical zero/one stores are intentionally compatible where their declarations coincide.

`report.go:129` uses ordered Series positions and aggregate selected/completed/failed/pending/unresolved/exhaustion fields, with added JSON series_state preserving historical Read.State vocabulary. Successor predecessor IDs derive from the actual protected continuation. Successful weather reports still require the original selected validated response, checked payload fields, source and provider observation time; prior results remain described as retained. No automatic next-observation approval or generic weather policy has moved into the author package. `weatherReason` still waits for terminal predecessor work when a result is retained.

`series_corruption_test.go:14` mutates nine isolated copies of a real eight-observation writer history and requires Inspect, InspectExecution, Replay and Open rejection plus failed-open/read-only DB/WAL immutability. It checks affected row counts and preserves the positive source. The coherent alternative-declaration case is independent of structural-history corruption. These checks establish the tested corruption boundary, not tamper resistance against arbitrary coherent rewrites.

## Limits and completion gates

At review completion, frozen full race output was still progressing in `task2-weather-race-frozen.log`; `weather-fixtures-frozen/` had not yet been emitted by its series campaign. I have not represented that race run as passing or reviewed nonexistent frozen campaign archives. Developer ordinary full-weather PASS and Staticcheck PASS are recorded in task-2-report.md; the independent reader verification above is complete.

Controller must retain the eventual frozen-race result accurately and run the required final integrated qualification on the exact approved author correction plus weather source. New final code or changed qualification claims require appropriate re-review. There is no approval here to merge/release, and no unrestricted recovery, remote finality, live-provider, or whole-milestone claim.

## Integrated producer confirmation

Verdict: **APPROVE the recovery/weather scope at producer commit `8a5f2ef564d5d750afa506675431d414bf92fb65`, tree `8caba38e10658672dc7f597751d322332f7d94bc`.** No new blocking finding. This confirmation supersedes the earlier source-specific integration reservation for this producer; final integrated race and delivery checks remain separate.

Independently verified the commit resolves to that tree and recomputed Git blob identities for all 311 files in `producer-source/`: zero mismatches. Recomputed its SHA256 source manifest, `ca95102a18437722cf61cce7dcf03ac7a98c2283266e193816ac4351c08ac048`, and checked all 311 file hashes. Compared source with the original reviewed frozen tree: weather Go code differs only in legacy import grouping; the README gains series instructions. Runtime and internal code are identical. The author changes are the precise escaped predecessor-identity size prevalidation and its tests in task-1-fix2.patch. The controller reports independent UX APPROVE at `ce1ac0c`; this review relies on that separate author review for closure of U1, while independently confirming the correction did not alter weather/runtime recovery behavior.

Inspected the actual packaged evidence and original producer artifacts:

- `examples/weather-agent/testdata/series.json` correctly binds the producer, manifest, raw log, trace and archive. The packaged gzip raw log decompresses exactly to `producer-results/ordinary.log` (SHA256 `f8afb1606cd781c7ad348784aab3cf8c94a1cbb74d05b07fbda9c20c75a3be9a`). The recorded command is the full ordinary `go test -json -p=3 -count=1 -timeout=60m ./...`, exit 0, 45.617 seconds. Independently counted 1,172 passed tests, zero skipped tests and zero failures from the raw events. Packages without tests have normal package-level skip events; these are not skipped test cases.
- Reassembled output chunks per `(Package, Test)` before parsing SERIES_CONTROLLED rows. All 11 parsed trace records exactly equal packaged `series-process.jsonl` records (SHA256 `b3c1105c9b68bedaab6ca9f7233bd76f9e0d1d33f0c5e928c0806e640e4d97bb`). They retain ten SIGKILL witnesses, 46 provider contacts and 48 paid opportunities across the eleven isolated histories. This does not mean 48 charges in one bound-eight store.
- Verified archive SHA256 `1724189442bc940cf953203214998c076d6ca5c5280e75c9519a38d0bc45784d`. All 42 main/WAL members of `series-stores.tar.gz` exactly match both the sizes/hashes in series.json and the original files in `producer-fixtures/`. Nonempty stopped WALs are preserved; no legacy testdata was replaced.

Extracted those verified archive bytes into disposable `/private/tmp/review-producer-db-conrzc1z/`. Re-ran only the targeted public-reader harness against the integrated producer source, with the same offline environment as above:

```
go run /private/tmp/alan-bounded-series.m9VryKAu/review-producer-retained.go
```

Exit 0. All 21 archived DB/WAL pairs passed runtime.InspectExecution and runtime.Replay, matched namespace/original-operation counts, and retained identical main/WAL byte hashes. No adapter was configured. Full output is `review-producer-retained.log`; packaging-check summary is `review-producer-check.json`. No broad suite or provider network was run by this reviewer.

Read actual records from the new archive copies as a discriminator: ordinary-eight has eight independently charged reservations/gates/results/settlements and eight selected adapter observations; lost-third final has four charges/gates but selected results only for weather-1, weather-2 and weather-4. The third original remains charged with no result. Fourth pre-dispatch reservation retains four charges but three gates; fourth post-gate retains four charges/gates but three results. Retained-third pre-settlement has three results/two settlements, settled-third has three settlements with third selection absent, and terminal-third has third selection present. These match the new trace and preserve the previously reviewed local/remote uncertainty distinction and explicit successor authority.

The older frozen full-weather race PASS (443.928 seconds) remains evidence for that older frozen source. It is not represented as an integrated-producer full-race result. Controller's integrated full race is still upcoming at this confirmation. This approval covers the specified producer source and its packaged recovery evidence, not a later changed delivery tree, merge/release, live-provider behavior, or unrestricted recovery theorem.
