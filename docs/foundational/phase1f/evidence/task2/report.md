> Publication note: original handoff paths below identify the author's scratch records. The source manifest is retained here as `source-manifest.json`, and the complete default test log as `default.go-test.jsonl.gz`; `review.md` and `review-source.tar.gz` retain the independent review package.

# Task 2 implementation report

Status: **DONE** (required slice complete; independent review and process qualification remain parent-owned).

Code ownership is frozen/released. No further implementation or test edits are planned. Any subsequent test failure will be reported before editing. No subagents, commits, branch changes, credentials, paid inference, heldout reads, prior-evidence modifications, dependencies, runtime edits or public authoring API extensions were used.

## Files and source identity

Added `examples/investigation/{investigation.go,store.go}` and external-package tests in `config_test.go`, `external_test.go`, `http_test.go`, `investigation_test.go`. Added `cmd/alan-investigate/{main.go,main_test.go}`. Added `docs/foundational/phase1f/authoring-obstacles.md`.

Baseline HEAD: `e2157a8fb2837ea3d7d0232683072182a0ee832b`.

Exact source/test manifest: `.superpowers/sdd/00-plan/task-2-source-manifest.json`. Aggregate SHA-256 of its canonical `files` array: `4cb9fb09dacf4ca303a7d1e8d2dffe518fc114ea3d375b1727e607134346826a`.

| File | SHA-256 |
| --- | --- |
| `cmd/alan-investigate/main.go` | `76b6466d171d2832a908c8dab8bae266e2d9000b6844d8b6f2bd510cd8f76303` |
| `cmd/alan-investigate/main_test.go` | `f8bc35b2b944e4979922d09ff83cef548f27adcda1ede126e2f46fe4a4839f7a` |
| `docs/foundational/phase1f/authoring-obstacles.md` | `e4879dfc27f27e968290aef1af6256e5a5e584bd6678fc60fd8484a372686714` |
| `examples/investigation/config_test.go` | `2a196d38e9aa6fbb3d03539d7bc1d0e69286dc4e8a064148ba87578e5acc356c` |
| `examples/investigation/external_test.go` | `b779e7bc93d0d68f78a0c8cb483f1ab1a9585d50ef9d54a197f333a23a6d7b4c` |
| `examples/investigation/http_test.go` | `bb43d3a772a5d35de8c565c240fa6018cc1436654ccb77787e557d2bc71d13f9` |
| `examples/investigation/investigation.go` | `1f7823a5ba483a47ddc7afca4718c8a53df08fa66023d766ebf7698960625063` |
| `examples/investigation/investigation_test.go` | `842f91dcc64a81134d0e1c2bebde3e68895c2288fe8286e818ffaf52ff482a06` |
| `examples/investigation/store.go` | `8db346ebe8aa22e846d703de935ec3c7eb429821d41ddefccbdc01c6513fd32c` |

The report and raw test log are handoff evidence, not included in the implementation manifest. The final default-suite JSON log is `.superpowers/sdd/00-plan/task-2-default.go-test.jsonl`, SHA-256 `491ae92f208e0437d7d2aff97e597c38d5964eb087d15ca55fa43e2abc00b099`.

## Public application API

- `New(name, caller, obligation string, b Budgets) (Component, error)` returns ordinary `Instances author.Application` and a `Port author.Port`. `Budgets{MaxRevisions: 0|1}` selects whether the optional round is authorized.
- `Report(name, caller, obligation, child string, b Budgets) (Component, error)` uses exactly the same `New` child and adds an ordinary report participant. It computes publication routing from the public child disposition, without child state access or additional model work.
- `RequestSchema()` is the existing one-question evidence request record (`question`, maximum 4096 UTF-8 bytes). Empty questions fail the ordinary component; provisioning also rejects whitespace-only input.
- `ResultSchema()` has 37 scalar fields. Nine application fields are `disposition`, `answer`, `initial_candidate`, `final_candidate`, `review_decision`, `review_reason`, `initial_review_decision`, `initial_review_reason`, `revision_count`. The other 28 are `e1_` through `e4_` prefixes applied to the existing helper fields `disposition`, `answer`, `document`, `version`, `snippet`, `start`, `end`.
- e1/e2 are initial helpers, e3/e4 are revision helpers. Unused slots have disposition `unused` and typed empty values. Exact citations are copied from validated helper public results; the coordinator model cannot supply or overwrite citation fields. A revised candidate sees all four slots. Initial-candidate grading uses e1/e2; final-candidate grading may use all populated slots. These are evidence references, not proof that every answer claim follows from them.
- Public dispositions are `answer`, `clarification`, `insufficient_evidence`, `no_match`, `invalid_result`, `budget_exhausted`, `stopped`, `revision_limit`. The no-match detail is normally present in helper slots and can lead to synthesis abstention. Only `answer` is accepted. `answer` is empty for rejected/invalid/stopped candidates; clarification/insufficient-evidence dispositions may carry explanatory text there. An empty review decision means no review was reached.
- `ReportResultSchema()` adds `report_outcome` for 38 total fields: `publish` for an accepted answer; `needs_attention` otherwise. Every investigation field is preserved unchanged. Report routing is an ordinary application judgment, not independent factual validation.
- `DefaultConfig()`, `Provision(question string, c Config, assets []effects.Asset)`, `Inspect(ctx,path)`, `Run(ctx,path,apiKey)` provide retained deployment behavior analogous to the public evidence example. `ErrNotInvestigation` supports CLI routing to the unchanged baseline inspector.

`Config` contains `Application="investigation/v1"`, explicit `Model`, `Endpoint`, `MaxOutputTokens`, `MaxResponseBytes`, `RequestTimeout`, `MaxRevisions`, and `Nested`. Defaults are Responses endpoint, 1024 output tokens, 65536 response bytes, 60s per-effect timeout, one revision, standalone. Model has no inferred default. The key is supplied only to Run; config and artifacts contain no secret. Nil/empty assets mean an empty corpus, not a hidden bundled corpus.

## Ordinary protocol and ceilings

The coordinator decomposes the question, sends both initial helper requests before either receive, synthesizes, and asks a separate ordinary reviewer. The reviewer returns schema-bounded accept/revise/stop plus a reason and (for revise) two nonempty subquestions. The coordinator permits at most one received, valid, authorized revision, sends two distinct one-shot helpers, synthesizes using earlier plus new evidence, and reviews again. A second revise or a zero revision budget stops with `revision_limit`; it creates no third round.

Fixed logical maxima: coordinator 3 model requests, reviewer 2, four helpers each 2 model/1 lookup = 13 model requests and four lookups. With `MaxRevisions=0`, the last pair remains unused and the maximum reachable opportunity is seven model requests/two lookups. Ordinary request updates persist counters atomically with intents. Model output, empty candidate/reason/subquestions, invalid helper citations, missing evidence and unresolved effects have explicit branches; unknown outcomes never trigger automatic retry or revision.

Unused helper/reviewer participants can remain waiting. `Run` success means the physical executor returned, not that every participant terminated or the application produced an answer. A blocked unknown can leave `View.Result=nil` indefinitely.

## CLI

```
alan-investigate init --db PATH --question QUESTION --model MODEL \
  --corpus-manifest docs/foundational/phase1f/evaluation/corpus/manifest.json \
  [--nested] [--max-revisions 0|1] [--max-output-tokens N] \
  [--max-response-bytes N] [--request-timeout DURATION] [--endpoint URL]
alan-investigate run --db PATH
alan-investigate recover --db PATH
alan-investigate inspect --db PATH
alan-investigate replay --db PATH
```

`run`/`recover` use runtime `OPENAI_API_KEY`. Init/inspect/replay do not request credentials or dispatch providers. Init additionally accepts `--baseline` to provision unchanged `evidence.Provision` with the identical independently retained manifest assets and explicit two-model/one-lookup ceilings. Baseline and nested are mutually exclusive. Baseline run/inspect use the existing public evidence functions. CLI selection may probe store type, but each returned View derives all its summaries from one snapshot, never mixed reads.

Manifest schema is `phase1f-corpus/v1`; document paths are relative to the directory containing `corpus/` (the evaluation root in the frozen fixture). IDs and exact file bytes are preserved. SHA-256 and declared byte length are checked. `effects.Asset.Version` stays the content SHA-256 required by the existing API; manifest editorial version `1` is not substituted. No corpus concatenation, span renumbering or whitespace transformation occurs.

`Run` reuses retained artifacts/config/assets; it neither recompiles nor reads original source/manifest paths. `replay` directly uses read-only `runtime.Replay` with no adapter registry. The only private CLI import is the already-existing observational `internal/boundary.FromEnvironment` hook, explicitly authorized by parent for process tests/spend gates; the reusable application package uses public packages only.

## Tests and red/green evidence

All Go commands used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1f-go-build GOPROXY=off GOSUMDB=off`; controlled HTTP used loopback only.

1. Before implementation, `go test ./examples/investigation` failed because the new public package had no non-test Go files. The first compiler pass then rejected an actual budget-exit definite-assignment error: `steps/8:send:coordinator: value "subquestion1" is not definitely available`. Typed initialization of all reviewer reply fields corrected the authored program; no compiler change was made. The initial 22 standalone/nested cases and public factory checks then passed (18.253s).
2. Before CLI implementation, `go test ./cmd/alan-investigate` failed on undefined `runCLI`. After implementation, manifest init, retained config/nested choice, baseline routing, source deletion, recover/inspect/replay, no credential requests on read-only commands and invalid CLI checks passed (0.760s).
3. Unknown tests passed for first sibling, first review and second review. Two sequential opens/runs preserve exact unknown identity, spent counters and attempt count; helper2 finishes while helper1 is unknown. No result is invented or revision added. Read-only replay succeeds.
4. Full real OpenAI adapter HTTP test runs the nested two-round workflow against a controlled service. It passed with 13 model HTTP contacts, four lookups, distinct initial/final candidates, retained earlier plus new evidence, accepted revised result, and server-closed replay. No paid service was contacted.
5. The useful outer report requirement was added as a failing test before implementation. Accepted/revised-accepted cases reported empty outcome instead of `publish`, and stopped case empty instead of `needs_attention`. Ordinary report routing plus public result retention made those tests pass (3.515s).
6. Final controlled table has 15 scenarios in both standalone/nested form (30): accept, revise/accept, second-revise limit, disabled revision, stop, clarify, missing, invalid plan, empty subquestion, invalid citation, invalid review reason, invalid revision question, empty synthesis, schema-invalid synthesis, and unsupported-answer-with-no-evidence guard. Counters equal committed intents and remain within participant limits; every spent participant retains a pending identity. Citation bytes/document/content hash/spans are checked against the fixture independently of the application schema. Revision acceptance in the fixture depends on receiving both retained earlier evidence and the two newly acquired facts, not merely review call count.
7. A temporary standalone Go module compiles both public composition forms. External-package tests actually execute both forms. Configuration bounds and deterministic artifact identity are covered. Manifest mismatched hash/size and path escape are rejected.
8. **Final source-matched full default suite:** `go test -count=1 -json ./...` exited 0. Fourteen tested packages passed, no failed tests; investigation 32.874s, CLI 1.308s. Raw JSON is retained at the path/hash above and includes bounded `INVESTIGATION` records. An earlier full default run also passed before the final test-fixture strengthening.
9. `go vet -tags integration ./...` exited 0 after final production code. The extra focused race command `go test -count=1 -race ./examples/investigation ./cmd/alan-investigate` exited 0: investigation 501.625s, CLI 10.573s. It started before the final test-only evidence-dependence strengthening, with the same production source; therefore this is supporting evidence, not a claim of final-source race qualification. Parent explicitly owns final integration/race/vet qualification after boundary tests; this extra run is not a required task gate.

No required check currently fails. Full integration/process-kill qualification was intentionally left to the next task, per the slice boundary and parent direction.

## Stable inspection and process-test boundary names

Provisioned subjects (generation 1) are `caller`, `investigation`, `investigation/reviewer`, `investigation/helper1` through `investigation/helper4`, and each helper's `/retrieval` child. Nested additionally has `report`. Top-level and report-child obligation is `investigation`; helper obligations are `helper1` through `helper4`; reviewer obligation is `review`; each internal retrieval obligation is `retrieval`.

- Coordinator model effects: `decompose` then up to two `compose`. Ordinary state: `model_spent`, `model_limit=3`, `revision_count`, `revision_limit`, `pending_plan`, `pending_synthesis1`, `pending_synthesis2`. Initial/final candidates and all evidence slots remain ordinary state.
- Reviewer model effect: `review` (at most twice), with `model_spent`, `model_limit=2`, `pending_review1`, `pending_review2`.
- Each existing helper: model effects `plan` and `synthesize`; state `model_spent`, `model_limit=2`, `pending_plan`, `pending_synthesis`.
- Each retrieval: lookup effect `lookup`; state `lookup_spent`, `lookup_limit=1`, `pending_lookup`.
- Capability names remain `evidence-model` and `evidence-lookup`. Intent IDs, external namespace-bearing keys, observation identities and artifact IDs are available through `View.Snapshot`, `View.Results`, and `View.Runs`; `View.Unknown` lists unresolved allowed attempts. Derive actual epoch/sequence/ordinal from retained identities instead of wildcarding boundary filters.
- Controlled unknown examples observed local IDs `["investigation/helper1",1,15,0]`, `["investigation/reviewer",1,11,0]`, `["investigation/reviewer",1,23,0]`. These illustrate current emitted identities; qualification must bind each actual store's complete identity and namespace, not assume the example values globally.
- Existing boundary names/mechanisms are unchanged. `INVESTIGATION` is reserved/emitted for new focused JSON records; next qualification can use it for process records.

Useful next cuts: after decomposition before either helper completes; partial sibling success; after initial review receipt/consumption but before revision count/sends; after second-round requests/results; before/after final report reply. Validate counters, distinct request IDs, no additional round after second revise, and read-only replay under removed credentials/sources. The CLI source-deletion smoke case takes the clarification path; next process qualification should exercise deletion across a full evidence-producing path as well.

## Decisions, limitations and review boundary

Concrete obstacles and selected public-only solutions are recorded in `authoring-obstacles.md`: finite slot expansion, existing flat-record model serialization, forward finite rounds, useful definite-assignment rejection, useful ordinary report policy, and one-snapshot decoding. No semantic extension was needed.

These tests establish bounded protocol behavior with controlled providers, not substantive model quality, general fairness, arbitrary nesting, provider idempotence or eventual completion. Citation provenance is mechanical; semantic claim entailment/abstention require independent grading. Unknown work may prevent a final reply; unused static participants may wait. The separate reviewer is an application judgment, never the independent evaluation oracle. Live evaluation, source/prompt freeze, heldout unsealing, real process-kill schedules and independent review remain parent/next-task responsibilities.
