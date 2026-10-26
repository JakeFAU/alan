# Phase 1E pre-change baseline reproduction

## Source and isolation

This is a **new Phase 1E baseline run against historical Phase 1D1 source**, not a change to Phase 1D1 qualification or a claim about future Phase 1E behavior. The source was extracted with `git archive d708493646aeae3c1e70a03cf2bb2d7db974c127` into `/tmp/alan-phase1e-baseline-source`; no test ran against the mutable worktree. `d708493` and the published `338c89f` resolve to the same Git tree, `4cafbf51208b6e958a5bdba801ec97fee6fab7ac`. The checkout was clean before and after the run.

All 58 files in the Phase 1D1 source manifest matched their recorded lengths and SHA-256 values. The manifest's length-prefixed, path-sorted aggregate recalculated as `9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63`, both before and after tests. This is the retained Phase 1D1 source ID. Go was `go1.27.3 darwin/arm64`. Tests used `GOTOOLCHAIN=local`, `GOFLAGS=-mod=readonly`, `GOCACHE=/tmp/alan-phase1e-go-build`, `GOPROXY=off`, and `GOSUMDB=off`. Loopback and subprocess access were enabled for the fixture tests. No credentials or live provider calls were used.

## Commands and outcomes

| Command in archived source | Result | Raw/metadata evidence |
| --- | --- | --- |
| `go test -json -count=1 ./...` | Exit 0, 357 test/subtest pass events, 0 failures, 29.046s wall time | `final-normal.go-test.jsonl.gz`, `final-normal.command.json` |
| `go test -json -count=1 -race ./...` | Exit 0, 357 test/subtest pass events, 0 failures, 106.432s wall time | `final-race.go-test.jsonl.gz`, `final-race.command.json` |
| `go vet ./...` | Exit 0, empty output | `static-checks.json` |
| `gofmt -l cmd internal research` | Exit 0, empty output | `static-checks.json` |
| `go mod verify` | Exit 0, `all modules verified` | `static-checks.json` |

The package `cmd/alan-resource` has no package tests and is used as a built child executable by integration tests. In each full mode, output contains 15 QUALIFICATION, 20 COMPOSITION, 32 SUPERVISION, six CONFORMANCE, 119 DURABLE, and 22 directly emitted ASYNC markers. Six DURABLE records additionally carry `type=ASYNC`. Raw JSON and extracted records retain original event order; no trace was reordered to produce a passing sequence.

## Historical comparison

For **each** normal and race run, the ordered payloads of all 15 QUALIFICATION, 20 COMPOSITION, and 32 SUPERVISION records equal the corresponding retained Phase 1D1 **final** records byte for byte after JSON decoding, without normalization. This compares behavior under the same source identity; it does not assert that wrapper metadata, process IDs, timestamps, or temporary paths are equal. All six CONFORMANCE records have `passed=true` and the retained test/fixture order. Four of their six complete payloads match exactly. Two include ephemeral endpoint/program details, so no full-payload equality claim is made for them. The existing six independent checkpoint assertions passed in the new test runs. `comparison.json` records counts, equality results, fixture names, and source identities.

## Preserved causal and recovery behavior

The full suites passed the named held-progress, saturation, out-of-order receipt, cancellation, limits, late evidence, ordinary supervisor progress, closure, known receipt settlement failure, and replay cases. For the six CLI held/crash scenarios in **both** modes, the retained causal order begins `endpoint-held → peer-committed → peer-delivered → held-live`: the unrelated peer and caller committed while the exact external request remained held. The two settle cases then record response release and settlement. Each scenario had one endpoint application and total 5. Idempotent held-kill and received-kill recovery contacted the endpoint twice under the same key and retained one application; non-idempotent cuts contacted once, with no invented receipt or settlement. See `causal-cases.json` and the complete DURABLE records.

Each mode includes 85 subprocess records and 20 acknowledged `SIGKILL` exits. All 85 process records in the race mode say `race_children=true`; the normal mode says false. The six conformance cases include accepted loss/successor, interrupted committed request, rejection surviving failure, observation without lifecycle authority, late external settlement, and closure before authorization. Existing crash and replay tests passed alongside the new async tests. The known-but-unsettled regression test passed in both modes; its intended distinction from unknown is asserted by the retained test, not inferred from a test name alone.

## Scope and limits

This reproduces one full normal and one full race execution from the immutable source with controlled local fixtures. It establishes no Phase 1E authoring/provider behavior and no live OpenAI outcome. The 357 count is test/subtest pass events, not independent properties. The normal and race runs have their own raw traces and command metadata. Historical Phase 1D1 evidence remains under its original source identity and was not rewritten. No current-source regression was found in this pre-change baseline.

Evidence directory: `/tmp/alan-phase1e-baseline-evidence/`. It contains raw compressed Go JSON, extracted compressed marker records, command metadata, deterministic comparison, causal case summary, static checks, and the collector/audit scripts. `evidence-manifest.json` gives SHA-256 and byte lengths for evidence files.
