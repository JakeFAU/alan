# Task 3 scoped re-review — fix round 1

**Specification verdict: APPROVED for R1/R2/R3 and the seven-file correction scope.**

**Code-quality verdict: APPROVED for this correction scope.** No additional actionable defect was found in the fix diff. This closes the three findings in `task-3-review.md`; it does not claim completion of application/CLI, process-kill, live-provider, or whole-phase qualification.

## R1 — ADDRESSED

`adapters/openai/sanitize.go` replaces the inconsistent direct substitutions and fail-open nested-string fallback with a shared bounded policy. `Execute` applies it to raw response evidence, HTTP request ID, decoded response/model metadata, payload keys/string values, and usage detail JSON. Sanitized payload evidence cannot remain a validated candidate: alteration clears the payload and changes an otherwise validated disposition to `credential_redacted`.

The string-layer, structural-depth, node and byte limits fail closed. At an inspection limit or unsafe decoding path, the implementation emits empty/null/omitted data instead of returning the uninspected string. JSON object keys are checked as well as values, and unsafe keys are removed. The raw response path cannot bypass the same policy through the response-limit branch. The final candidate handling checks metadata before bounding it and retains only sanitized usage details. The previous fixed marker is gone, avoiding a marker accidentally echoing a configured credential.

The unchanged original probe now reports `escaped_header: credential_redacted` and `nested_json_credential: http_rejected`, with both recoverability flags false. The canonical control still validates; every case has one contact. The regression oracle recursively decodes retained strings, JSON keys and escape fragments, covering the header, eight-layer error, nested key, decoded metadata and usage-detail paths. The eight-layer input now exercises omission at the existing four-layer limit rather than relying on a larger cutoff.

**Limits accepted:** this conservative policy can discard benign unsupported escapes or deeply nested evidence, including detail that would otherwise aid diagnostics. Empty strings/null/absent details represent omission; they do not establish that the provider omitted the original field. It recognizes JSON and JSON escape fragments, not arbitrary encodings, encryption, reconstruction from separate fragments, or general secret inference. The byte/node budgets bound sanitizer inspection; underlying parsing/serialization also remains bounded by the response-size limit. This is consistent with the documented narrow credential-echo protection and trusted deployment profile. It is not a general data-loss-prevention claim.

## R2 — ADDRESSED

`adapters/openai/protocol.go` guards the exact names of every field consumed by the provider-envelope structs before typed decoding. Noncanonical aliases in Go's Unicode case-fold class are rejected, including the retained `Status` counterexample and the Unicode long-s alias. The check covers root, output item, content and usage fields; duplicate JSON-key checking remains separate.

The original canonical `status:incomplete` plus `Status:completed` input now produces `protocol_invalid`. The added regressions cover error/output/role/content/type/text/usage alias overrides. Application payload parsing remains map-backed and outside the protocol-key guard. The compatibility control preserves distinct `answer`/`Answer` and composed/decomposed Unicode field names and values. The existing canonical protocol cases pass in the covering evidence.

## R3 — ADDRESSED

`internal/executor/adapters.go` records the observation IDs from fully verified forward adapter admission/result/settlement bindings and then checks every retained `runtime-adapter` observation has exactly one such binding. The pass reads/validates observation descriptors without reduction or adapter execution.

The unchanged ledger probe now causes Inspect, Open and Replay to reject the orphan with `unexplained authenticated adapter observation`. The regression's other branch stops after durable result recording and confirms Inspect accepts one adapter result with no observation; Replay also accepts that pending state. Thus the reverse check does not confuse known-but-unsettled evidence with an orphan observation or Unknown.

## Evidence and source identity

Reviewed the appended round-one report and all seven changed/added files in `evidence/task-3/review-round1/round1-fix.diff`:

- `adapters/openai/openai.go`, `json.go`, `sanitize.go`, `protocol.go`, and `review_regression_test.go`.
- `internal/executor/adapters.go`.
- `runtime/review_adapters_test.go`.

Independently compared the current owned files with `before-source.tar.gz`: exactly those seven files differ. All 25 current owned source/test files match `after-manifest.json` and the refreshed `frozen-source-sha256.json`, which agree. The review's three original source checkpoints match the archived pre-fix files. Original reviewer probes and databases were not modified by this re-review.

Examined the retained before/after probe outputs, assertion failures in `regressions-red.log` and `r2-red.log`, the documented intermediate duplicate-payload classification failure, and its covering correction. The original diagnostic probes exit zero by design; they are not assertion tests. The after outputs show the corrected classifications and ledger rejection described above.

Accepted and inspected these supplied executions, using `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`:

```text
go test -count=1 -tags integration ./adapters/openai ./runtime ./internal/executor
go test -count=1 -race -tags integration -run 'TestReviewR|TestResponses|TestEscaped|TestUsageAnd|TestDuplicate|TestOversize|TestRealResponses|TestLookupFields|TestKnownAdapter|TestUnknownAdapter|TestMissingAdapter|TestInspect|TestAdapter|TestSchemaV2' ./adapters/openai ./runtime ./internal/executor
```

Both pass in `covering-packages.log` and `targeted-race.log`. The focused R1/R2/R3 covering logs also pass. Original before/after HTTP and SQLite diagnostics are retained in `probe-before.jsonl`, `probe-after.jsonl`, `ledger-probe-before.json`, `ledger-probe-after.json`, and separate `ledger-before/` / `ledger-after/` directories.

No new concrete doubt required another probe. This re-review ran read-only source/hash/archive comparisons and inspected the retained evidence; it did not repeat tests, create new fixtures, edit runtime/test code, commit, or spawn subagents. The original single-owner and transaction architecture is unchanged by these fixes. Live remains NOT RUN, and Tasks 4/5 plus final whole-phase checks remain outside this verdict.
