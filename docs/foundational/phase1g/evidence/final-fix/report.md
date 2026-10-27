# Phase1G final review fix wave — M1 and M2

M1 and M2 are implemented within the authorized tooling/documentation scope. No Go, runtime, application, module, fixture, schema or contract input changed. The focused checks below passed; scoped independent rereview and publication remain the controller's responsibility. Original Task4 and whole-branch review reports are unchanged.

## Source and evidence association

| Evidence | Source SHA256 | Archive SHA256 |
| --- | --- | --- |
| Retained full qualification | `43b86781f819c6bf7a957956aceaf717a5808873114e8213bf1147620bdf2ec3` | `423c1bff2aaf4e8a76a8813df6a1076977c2d8844db13e2b34919ea382732fb4` |
| Current metadata tool and focused checks | `ce4bbf6f9e6a0c7e94ccdfec09d97f9355b5e9bb1a6f49fcca2b17fe7256482a` | `bf2c473570a08976e5a9308cbdd895eac41afec921dcb775168b09b409abba9d` |

The current source has 258 inputs. The current archive has 613 verified members and 26,752,699 bytes. Existing `tools/freeze.py` and `tools/package.py` produced both identities without modification. The archive includes complete current sources, the nested consumer module, original control sources and the pinned SQLite file proxy. The SQLite ZIP remains `ebd1fef8e9434fbdc2c94e35b9991df62d472c00170f57fc4f75daa69f658a1b`. Toolchains are external prerequisites.

[Source delta](source-delta.json) inventories all 256 unchanged original inputs; the only changed frozen input is `docs/foundational/phase1g/tools/qualify.py`, and the only added frozen input is `tools/test_qualify.py`. [Final audit](final-audit.json) recomputes the current aggregate source identity, verifies every member/source input in both archives, checks the dependency ZIP and independently compares all 257 original source inputs with Git base `10f6d8c9766c41d2b9b5129c3eeaf5d5614c55ba`. This proves the current Go/tests/modules/fixture inputs are the same bytes as the fully qualified source. It does not assign the old broad-suite logs the new tooling identity.

The original archive/manifests and **all** Task4 files are preserved byte-for-byte under [qualified-source-43b86781](qualified-source-43b86781/source-manifest.json), with the original report at `qualified-source-43b86781/task4/report.md` and full-suite outputs at `qualified-source-43b86781/task4/extracted/`. Existing `evidence/task4/` files also remain unchanged. [Preservation inventory](qualified-source-preservation.json) verifies all 369 snapshot files. `qualified-source-43b86781/adjacent-documents/` preserves original adjacent Phase1G Markdown plus root README/current decisions and original archive timing bytes. These are concrete preserved document bytes, rather than claims that their old hashes still describe edited live files. The original archive separately retains its original archived document editions and exact per-member hashes.

[Before inventory](before-inventory.json) and final audit show exactly eight changed original files: README, current decisions, Phase1G qualification/runbook, top-level source/archive manifests, source archive and qualify.py. All other 737 inventoried original artifacts are unchanged. All 1,120 earlier foundational artifacts independently match the baseline inventory again. New evidence and the Python regression are additional files.

## M1: executing metadata and parse preservation

Fresh `qualify.py` runs now obtain `platform.platform()` and `go version` before the Go test command. The version probe uses the same selected environment and root working directory as the test command, including `GOTOOLCHAIN=local`, `GOFLAGS=-mod=readonly`, selected caches/proxy and removal of the provider credential. A failed version probe cannot silently fall back to freeze metadata. The manifest's platform/version continue to describe the freeze environment.

`--parse-existing` now preserves recorded `Platform`, `GoVersion`, `Directory` and `CredentialAbsent`, alongside the already-preserved command, environment, start time, duration, exit and stderr. It does not execute Go to discover the parsing host's toolchain. It still validates source identity and the original raw-log hash. Parsing cannot recover correct metadata that an old record never captured; old records remain untouched by this wave.

`tools/test_qualify.py` runs the real collector with a deliberately different synthetic freeze host/toolchain and a small fake Go executable. The fake checks the selected environment, working directory and version/test argv; its output is explicitly synthetic metadata evidence. A second test seeds a recorded command, parses it on a different synthetic host with Go invocation forbidden, and checks preservation of the original execution fields and raw bytes. Neither test is a Go runtime qualification.

The exact final RED/GREEN command records are in [regression-commands.json](regression-commands.json). The command is:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover \
  -s docs/foundational/phase1g/tools -p test_qualify.py -v
```

With `ALAN_METADATA_TEST_TOOL` selecting retained [original tool bytes](qualify.original.py.txt), both tests fail specifically on freeze-host attribution (actual exit1). With the current frozen tool, both pass (actual exit0). `recorded-original/` and `recorded-fixed/` retain complete synthetic input manifests, tool bytes, command output, raw logs and generated metadata; outer stdout/stderr are retained as gzip files. `ALAN_METADATA_TEST_EVIDENCE` selects a fresh retention directory and refuses to replace existing inputs.

Earlier development attempts also remain: the first fresh synthetic command failed because macOS `/var` and `/private/var` aliases differed in the temporary-root assertion; canonicalizing the test root corrected that setup. The corrected RED then failed both intended host assertions. The first GREEN reached a test-only argv assertion that included the executable name on one side but excluded it on the other; matching argument slices corrected that assertion. Final recorded RED/GREEN used the same final regression code. These setup/assertion failures are not runtime defects or intended fault detections.

## M2: README reconciliation

README now distinguishes legacy schema3 scalar records from complete schema4 bounded typed lists, snapshot iteration and retained-template invocation. Current limits describe bounded data-dependent creation from a static catalogue and shared root envelopes. The location table includes typed-list schemas, composition and the separate consumer. Explicit consumer normal/race commands use `-timeout=240s`, followed by consumer vet and module verification. The root integration paragraph clearly states that root `./...` excludes that separate module.

The qualification page, runbook and current decisions identify the tooling-only correction and link this evidence. Original full-suite reproduction commands now select the retained original archive. The current focused verifier is separate. These changes do not claim the controller's final rereview or publication is complete.

## Actual fresh extracted checks

[verify_focused.py](verify_focused.py) follows the existing `reproduce.py` extraction, member validation and environment pattern, while running only these four commands. [Extraction record](extracted/verification.json) retains exact directory, environment, argv, numeric exits, durations and stdout/stderr files. It verified all 613 members, used a fresh temporary extraction and initially nonexistent module cache, the pinned bundled file proxy, required shared build cache, local toolchain and no provider credential. Sources were checked before/after each command and again after execution.

| Extracted command | Actual exit | Evidence |
| --- | ---: | --- |
| Python metadata regression above | 0 | `extracted/01.stdout.gz`, `01.stderr.gz`; 2 tests passed |
| `python3 docs/foundational/phase1g/tools/qualify.py consumer` | 0 | `extracted/02.stdout.gz`, `02.stderr.gz`; actual inner Go command below |
| `go -C examples/worktree/consumer vet ./...` | 0 | `extracted/03.stdout.gz`, `03.stderr.gz` |
| `go -C examples/worktree/consumer mod verify` | 0 | `extracted/04.stdout.gz`, `04.stderr.gz` |

Actual inner command:

```sh
go -C examples/worktree/consumer test -json -count=1 -timeout=240s ./...
```

[Consumer command metadata](extracted/final-consumer.command.json) records actual exit0, 20 passing events, no failures, package time5.478s, collector duration18.137s, `macOS-27.2-arm64-arm-64bit-Mach-O` and `go version go1.27.3 darwin/arm64`. The complete raw JSON is retained as `final-consumer.go-test.jsonl.gz`, with decompressed SHA256 `79dd02f98ca217591356bb5e19b3aa66a396604c15aa5925a5d033141f5658e4`. The fresh module-cache download used the bundled file proxy. This is actual Go execution, distinct from the synthetic metadata regression.

No new broad default/integration/race suite ran. The original source43b full qualification remains: default764, integration1006, race1006, consumer20, consumer-race20 passing events, all exits0; three comparisons and eight auxiliary checks passed. Those results remain under their original source/archive identity. The README race command is reconciled guidance, not a claim it was newly executed in this wave.

## Archive timing, final verification and self-review

`freeze.log`, `package.log` and `extracted.log` retain package/command output. The new archive was packaged after the metadata RED/GREEN checks and documentation reconciliation, before extracted checks. Its `evidence/final-fix/report.md` is explicitly a pre-package checkpoint. Completed raw extracted results, final audit/report and controller rereview remain adjacent to avoid recursive identities. The archive and archive manifest retain exact document hashes; the final adjacent report has its own bytes. [Archive document snapshot](archive-document-snapshot.json) explicitly distinguishes archived checkpoint hashes from current adjacent document hashes.

Final verification uses `python3 docs/foundational/phase1g/evidence/final-fix/audit.py`, the focused Python regression, and scoped `git diff --check`. The audit verifies all258 current source inputs again after the real commands, both archives and all retained snapshots, the unchanged-Go association and all1120 prior artifacts. The self-review checked selected environment/toolchain propagation, refusal to overwrite evidence, parsing without Go, original raw-byte preservation, README/profile consistency, command/raw-log association and separation of retained broad results from new focused results. No additional finding remains from that pass.

This is the authorized single small fix wave. There are no new runtime/contract claims, live calls, credential-file reads, paid work, source/module upgrades or broad/race reruns. Finite-schedule and platform limitations from the original report remain. Scoped independent rereview, final review-status documents, plan publication checkboxes, push and PR remain controller responsibilities.
