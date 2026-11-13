# Qualification and reproduction

## Source and input identity

| Role | Commit | Tree |
|---|---|---|
| Verified main at campaign preparation | `c72c396df09c55b0be1a91b2557dcec2ef6457a8` | `52f4d7c9a2f573e78112be0c1f07b26834918603` |
| P3 prerequisite, [PR #14](https://github.com/JakeFAU/alan/pull/14) open and draft at verification | `d59eaf2976849e450635d252c7bca62eeccbef2e` | `dc082ea07e4c24d8f36e56d756771f7c1154b0ec` |
| Integrated measurement baseline, normal merge of main into the P3 continuation | `ef38ca47179d86e7544885471ae43e86e1d6351e` | `4b8f76b5fd84df99e13c4a8381dde65d1b70746b` |
| Reviewed executable at Task 4 | `102c62062c61985bcbe5ea8c8c9d12fcf385891a` | `c69553b9754a91b38709395f13f0084077d14217` |
| Actual frozen candidate | `c620b660e0778e8b0d3cd5205242c0201b2b4019` | `c983e52ec48fa0cb2570631c4115acd21073a830` |

The Task 4 to frozen-candidate diff contains only the Task 5 apparatus and evidence documentation. The complete Git tree comparison covers executable Go/native inputs, every test and fixture (including embedded data), and all root/nested module manifests. The later commit introducing this page adds only documentation and compact derived evidence. All confirmatory commands and samples below retain their actual frozen candidate or baseline identity. The comparison commands are in [Reproduction](#reproduction).

| Frozen file | SHA-256 |
|---|---|
| `source-freeze.json` | `e1b0b264f20f05776bde07b3a724dd19868b337789e83438ad85fdd2d9850f33` |
| `qualification-freeze.json` | `d51773938675aff3a0bd9ae5496191faa2b9ee098fb00baa264b84adb7b58cea` |
| `diagnostics-freeze.json` | `1a76a0dfecaee699896d21337a29b6b5855c457f72b36f55275d3370e83603ce` |
| `timing-freeze.json` | `700ae53dd84f2723c593a6167c1d241b877c54d07f14b64cb98a6ce3d033f233` |

The timing binary SHA-256 is `c9bce78d6a265f5bd5cdb283471fc1c77576bf2e7d9d4702e2a80f27b0dad962` for baseline and `73210c512d791a32f95633880f2c29ac7f336ddb78b80dfb80696c465c99cfff` for the candidate binary shared by its modes. The qualification and diagnostic binary hashes, qualification tool hashes, source archives, command/outcome digests and before/after inventory bindings are indexed in the compact data and full frozen inputs. The recorded host is Apple M4, Mac16,10, 16 GB, darwin/arm64 with Go 1.27.4, Clang 21, SDK 27.2, Python 3.9.6, Staticcheck 2026.2.2 and go-sqlite3 v1.14.52. Runs use `GOMAXPROCS=4`, `GOGC=100`, `GOMEMLIMIT=off`, `GOWORK=off`, `GOTOOLCHAIN=local`, CGO and disabled VCS stamping. [Source, tool and gate hashes](campaign-data.json) · [Timing hashes](timing-rows.jsonl) · [Diagnostic hashes](diagnostic-data.jsonl)

## Executable gates

Frozen gate paths below are relative to `resume/final-campaign-01`. The parent command chain and exits are in sibling `campaign-launch-01`. The final qualification failure list is empty. `gates/<name>/command.json`, `before.json`, `after.json`, `outcome.json`, `stdout.log` and `stderr.log` record the actual command, working directory, environment, bound input identities, exit and output. The indexed hashes verify the original records.

| Gate | Exit | Go test pass events |
|---|---:|---:|
| `baseline-full-ordinary` | 0 | 1,392 |
| `candidate-full-ordinary` | 0 | 1,412 |
| `candidate-full-integration` | 0 | 2,631 |
| `candidate-full-integration-race` | 0 | 2,631 |
| `focused-ordinary-race` | 0 | 214 |
| `focused-integration-race` | 0 | 343 |

The full tests use `go test -buildvcs=false -mod=readonly -json -count=1 -timeout=30m ./...` with `-tags=integration` and `-race` as applicable. The focused commands retain the actual filters for static execution, root reports, readiness, continuation basis, causal recovery, request authority/binding, competing owner/reader and corruption cases. Pass events include subtests. Three no-test packages are `examples/inspect-execution`, `examples/quickstart` and `internal/storetest`. Both candidate integration modes have one opt-in skip, `TestPhase3B3BaselineS2CompletionExpectedFailure` (requires `ALAN_PHASE3B3_EXPECT_S2_COMPLETION=1`). The frozen Go inventories have no fuzz targets or `testing/quick` imports. Corruption/mutation tests are deterministic.

Both sources passed root `go build -mod=readonly ./...`, ordinary/integration `go vet -mod=readonly`, `go mod verify`, `go mod tidy -diff` and all four Staticcheck modes (ordinary, integration, race and integration-race, with race in `GOFLAGS`). Format and whitespace gates were clean. Five nested modules passed build, tidy-diff and verify. Their test pass counts are 17, 71, 0, 10 and 0, with the two differential modules declaring no test files and the independent series module's base-reader command declaring no test files. Baseline and candidate lint are clean in this frozen campaign. Earlier infrastructure blocks, intentional negative tests and implementation/harness failures retain their own outcomes. The final recorded nonzero child gates were the five expected exit-1 contact-identity rejections for capability, effect, endpoint, operation ID and protocol. [Commands and event digest index](campaign-data.json) · [Prior failures and corrections](delivery-decisions.md)

## Semantic and recovery evidence

- **Actual older readers.** The pre-authority CLI at `af6943a5805a9a6c0e8fe77f804264f4765d07ec` initialized its own supported store and successfully ran Inspect and Replay. The pre-binding owner at `d59eaf2976849e450635d252c7bca62eeccbef2e` ran the real `internal/durabletest.TestPhase3B3Owner` child on a genuine supported format-13 K01 parent. Its quiescent cut has zero steps and dispatches and retains original `b/1` open. Every full candidate suite records `TestRequestAuthorityFormatsConflictPromotionAndOldReader/{13,14}/frozen_old_reader` and `TestRequestBindingConflictAndOldReader/{false,true}/frozen_reader` for formats 15/16. Each pair rejects the new storage formats with the actual older binary. Genuine K01/K02 parents promoted 13→14 and 15→16 with the original consumed results retained. [Qualification routes](task5/qualification.py)
- **Independent public consumer.** Ordinary and race variants each recorded 16 pass events. The external module exercised eight distinct operations, contacts, charges, results and selections using the public API and independent readers, and compared canonical lowering at B1/B8/B32/B256. It opened read-only copies of supported historical archives. The named `TestIndependentPromoteSeries` ran on a real format-11 store, producing format 12 with the retained-result evidence preserved. The baseline reader decoded both, and candidate retained-format checks passed. [External consumer and promotion routes](task5/qualification.py)
- **Exact differentials.** Eight direct same-seed pairs cover success and refusal at operation 3, with and without interval reopen, for structural and targeted comparisons. Two controlled HTTP same-seed pairs cover structural success with and without reopen. Comparators retain every field of the RunReports, actual contacts/identities, typed durable values, inspection/History and Replay, on complete separate copies. Representative raw pairs contain 18 compared cuts with eight full reports, or 25 cuts with 15 reports for the direct basis refusal/reopen case. Matching rejects each wrong contact field on either side. [Differential apparatus](task5/qualification.py)
- **Local recovery.** Both builds ran weather series, continuation, recovery, public and predecessor suites. The 128 retained database archives include 86 acknowledged raw stopped captures with verified main/WAL/SHM custody, actual SIGKILL, PID and reap. Independent readers used complete copies and verified matching public and typed durable facts without changing the sealed originals. Covered cuts include approval, commit, reservation/admission, lost and retained responses, selection, terminal and repeated reopen. The lost-response cases preserve unknown original effects and separately charge separately approved successors. Retained responses permit local completion without another original contact. Four additional ordinary-writer cross-build cases cover each writer on both acknowledged `execution.staged` and `execution.committed` boundaries, matching actual reap exit −9, complete sidecars and opposite-build resume with equal RunReports and durable facts. [Recovery apparatus](task5/recovery.py)
- **Independent reproduction.** The outside-checkout route exported the frozen candidate, rebuilt using the outside source replacement and validated a B8 targeted public proof. Its binary SHA-256 is `449b1d7b0ef2ca5f9bb359dff44708962f0f4dda1a970e45040983a58a3e17db`. The persistent mirror and inventories are in `outside-reproduction/` and `outside-reproduction.json`. This uses the same workstation. [Source and output digest index](campaign-data.json)

## Reproduction

Use the [campaign commands](task5/README.md#freeze-and-run) with the frozen candidate and integrated baseline above in a fresh persistent root. The recorded order was source freeze, build qualification, qualification, differential, recovery, outside reproduction, build/run/analyze diagnostics, then build/run/analyze timing. Compare the frozen digests before interpreting the archived campaign. A fresh run binds its own binaries, output paths, report hashes and times.

To verify the complete source bridge in Git, inspect both full diff file inventories. The two changesets should contain only the listed Task 5 apparatus/evidence and subsequent reporting files. The second command identifies the commit that added this file, even after later documentation updates.

```sh
git diff --name-status 102c62062c61985bcbe5ea8c8c9d12fcf385891a c620b660e0778e8b0d3cd5205242c0201b2b4019
delivery=$(git log --diff-filter=A -1 --format=%H -- docs/evidence/execution-scaling/qualification.md)
git diff --name-status c620b660e0778e8b0d3cd5205242c0201b2b4019 "$delivery"
```

The compact results use original reports under `gates/sample-*/stdout.log`, not parsed prose or a repeated workload. Against the archived working evidence, run the read-only extraction (the output directory must already exist):

```sh
evidence=/absolute/path/to/extracted/resume
out=/absolute/path/to/empty/output
python3 -B "$evidence/task5-results/derive.py" "$evidence/final-campaign-01" "$out"
```

Compare `timing-rows.jsonl`, `timing-summary.jsonl`, `diagnostic-data.jsonl` and `campaign-data.json` with the tracked files. Timing JSONL starts with the positional row and stage column schemas. Each subsequent line is one original row in frozen order. Aggregate repeated stages within each report, using the maximum for cumulative RSS. Add user and system CPU per row. Take median/min/max over each arm's six values and subtract B from A within the same numbered pair. The extraction verifies every phase and resource aggregate against the archived strict timing analysis. The latest sampled RSS takes the maximum across all recorded phase gauges per row, then computes the arm median. The compact data indexes the raw reports, phase values, charges, profiles and qualification outcome hashes. The extraction script SHA-256 is recorded in `campaign-data.json` as `derivation_sha256`. Full profiles, SQLite captures and raw investigation output are held in the campaign evidence. [Compact data](campaign-data.json) · [Results](results.md)
