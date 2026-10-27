# Phase 1G starting baseline

Starting main commit: `80dead19846d9b034b4457a9436e43acb81da3df` (tree `77395bdf55a98349c66f39629c594befc5fa2985`), captured from a clean worktree on branch `codex/phase1g-data-composition` at 2026-10-26 17:49:38 UTC. No production or test source changed during capture.

[Identity](identity.json) verifies all 189 contributing current-main files against published Phase1F source `2dacb402e50ee938d6184c14b40ba469fdcb8d91d9c30eac178b0dcb2a0fd3f8`. All 355 members of the retained source archive were checked against its manifest before extraction. Archive SHA-256 is `95407d737dc6002e3a43d4317e7390280a3f5b7dc983ab8f4c247e0fa139dcc6`. [Published source](published-source-manifest.json) and [archive](published-archive-manifest.json) manifests retain exact inputs.

Suites execute an isolated frozen copy. This is historical source reproduction plus separately verified equivalence of the starting main's contributing inputs. It is **not** final current-source Phase1G regression qualification. Later source edits cannot change these runs. Race qualification is deferred to the final complete-source gate.

## Executed checks

Platform: macOS 27.2 arm64; Go 1.27.3 darwin/arm64; CGO enabled; Apple clang 21.0.0. Environment: `GOTOOLCHAIN=local GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build GOPROXY=off GOSUMDB=off`. `OPENAI_API_KEY` and `ALAN_BARRIER` were removed before execution. Tests use controlled loopback providers; no credentials or live provider calls were used.

| Exact command in frozen source | Result |
| --- | --- |
| `go test -json -count=1 -timeout 20m ./...` | PASS; 550 test/subtest pass events; 38.428s |
| `go test -json -count=1 -timeout 20m -tags integration ./...` | PASS; 756 pass events; 41.432s |
| `go vet ./...` and `go vet -tags integration ./...` | PASS |
| `gofmt -l author schema runtime effects adapters cmd internal research examples` | PASS; empty output |
| `go mod verify` | PASS; all modules verified |
| `python3 -m unittest discover -s docs/foundational/phase1e/tools -p test_comparison.py -v` | PASS; seven controls |
| `python3 -m unittest discover -s docs/foundational/phase1f/tools -p 'test_*.py' -v` | PASS; 23 controls |

[Summary](summary.json), `final-{default,normal}.command.json`, compressed raw `.go-test.jsonl.gz` and `.records.jsonl.gz`, [auxiliary output](auxiliary.json) and [collector invocations](collector-invocations.json) retain exact directories, commands, statuses, timings and hashes. Pass-event totals count tests/subtests, not independent properties. Default/integration retain respectively 37/102 INVESTIGATION records, both retain 119 DURABLE records, and integration retains 120 EVIDENCE records. Actual ASYNC counts are 23/25; no deterministic ASYNC-count assertion is added.

The existing Phase1F `qualify.py` ran unchanged except an in-memory output-directory redirect and rendering its raw log path absolute. Existing Phase1E comparison functions check published Phase1E and Phase1F records separately in both modes: 15 QUALIFICATION, 20 COMPOSITION and 32 SUPERVISION payloads match exactly in declared order; six CONFORMANCE public projections and model streams match. Raw ephemeral conformance ledgers are not claimed identical. [All 16 comparison results](historical-comparisons.json) are retained. [Before](prior-artifacts-before.json)/[after](prior-artifacts-after.json) SHA-256 maps verify all 1,120 preexisting foundational artifacts remain byte-identical.

There were no Go-suite failures. The original wrapper then looked for published generated Phase1F logs inside the source-only archive, where they intentionally do not exist. The [unchanged wrapper](run-baseline.py), [reproduced traceback](postprocess-original-error.log.gz), [correction record](postprocess-correction.json) and [corrected postprocessor](postprocess-baseline.py) remain. Comparison reads the immutable repository's published logs instead; all passing Go outputs were reused without rerun. The traceback is explicitly a reproduction of the original post-collection failure, not an invented original process transcript. [Capture script](capture-baseline.py) retains the snapshot method.

## Existing capability and missing contract

Reusable public foundations are immutable versioned artifacts; typed flat request/result ports with static wiring validation; atomic effect request plus budget-state updates; static capabilities/grants; retained assets; one bounded Run owner; independent intent, receipt, settlement and consumption commits; unknown-outcome preservation without automatic repetition; and registry-free read-only Replay. Ordinary evidence and investigation components already run directly and nested. Process fixtures establish causally acknowledged sibling progress, SIGKILL recovery at selected recorded-result cuts, original-message redelivery and dispatch-free replay with independent provider ledgers.

The public schema accepts only scalar int/string/bool fields. `Repeat` expands a compile-time integer; `Goto` only targets a later label. Ports bind a static peer at generation 1; `Application.Provision` creates initial runs and static grants. Phase1F provisions separate one-shot helpers for each possible revision. There is no public typed runtime work-set iteration, reusable item dispatch/collection, accepted-work ledger, shared hierarchy budget, or logical descendant ownership/cancellation contract. These gaps cannot be met by hiding a larger fixed graph behind a builder.

Relevant retained limits: [Phase1E contract](../../../phase1e/01-authoring-effects-contract.md), [Phase1E review](../../../phase1e/03-review.md), [Phase1F contract](../../../phase1f/01-application-contract.md), [Phase1F review](../../../phase1f/03-review.md), and [process review](../../../phase1f/evidence/task3/review.md). Current persistence is schema 3, external key version 2; public author semantics `alan/1e/v1`, compiler `alan/author/v1`. Unsupported older stores are rejected, not silently migrated. Existing logical effect budgets are per participant; they are not hierarchy-wide budgets. Physical cancellation accounts a dequeued completion but does not logically cancel an application. Unknown work can block completion. Arbitrary late external adapter-candidate injection is not a public tested operation; supported duplicate old-message settlement is narrower. Static dormant helpers may remain waiting. Review/finite tests establish neither arbitrary recursion nor dynamic spawning, fairness, distributed ownership, power-loss durability or business deduplication.
