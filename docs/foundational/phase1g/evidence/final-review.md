# Phase 1G independent whole-branch review

Reviewed 2026-10-26. Exact range: `80dead19846d9b034b4457a9436e43acb81da3df..e81ec0b0b0b124e74beac834f5f6c5db1c47cc1e`. The user Phase1G specification, normative `01-composition-contract.md` including R1–R8, and `phase1g-plan.md` govern this review.

**Bounded Phase1G readiness: Approved for the requested draft PR, with two non-blocking Minor findings. No Critical or Important finding remains from this review.** This is technical review of the provisional single-host increment and its finite qualification evidence. It is neither a production-readiness proof nor authorization to merge or release. Publication, final documentation disposition and user review remain controller responsibilities.

## Scope and method

Reviewed the supplied source/contracts companion and consulted the full exact-range package (813 changed-file entries, including 726 evidence entries), original review findings, current regression implementations, qualification/runbook, and referenced retained records. Passes covered typed values/compiler residuals; retained templates and public handles; transaction authority and charges; closure/publication; Run ownership; reconstruction/version boundaries; ordinary consumer/CLI; and causal process/reproduction evidence. Prior approvals were treated as evidence, not as substitutes for checking those integrations.

No subagents, live requests, credential reads, Go test reruns, dependency changes, source edits, index/HEAD mutations, commits or publication were performed. No new concrete runtime counterexample required a focused executable probe. Independent read-only Python checks recomputed source/archive/member/raw-stream hashes, command exits/counts and historical artifact preservation. The sole review write is this report.

## Strengths

- **The requested capability is ordinary composition.** `compose/collection.go:25` derives scatter and ordered collection from Receive, ForEach, Invoke, Append and AwaitInvocation. Its two retained loop bodies do not depend on item count. `examples/worktree/worktree.go:67` provisions one static driver and a closed catalogue; direct and nested deployments retain the same collection template and public schemas. There is no privileged collector/agent role, item-specific generated helper graph or second scheduler.
- **Finite typed data survives interruption.** `schema/collections.go:54`, `:203`, `:244` and `:297` enforce copied native record lists, closed schemas, nesting/count/byte/key bounds and atomic append. `internal/executor/collections.go:168` retains snapshot and cursor separately from body commits. `author/compile.go:480` checks typed loop bindings; `:632` includes the empty-loop path in assignment analysis. `internal/executor/program.go:450` commits pending operation identity with updates and emitted intent. Original scalar Append panic coverage is retained at `schema/collections_test.go:201`.
- **Handle serialization does not grant result authority.** `author/compile.go:307` emits a declaration/template-specific receive; `:601` permits a saved invocation handle string while retaining rejection for known wrong request types. `internal/executor/program.go:383` matches exact run, provenance, source/template, generation, declaration and operation. `store.go:755` separately prevents reuse of a consumed observation. Missing responses remove payload bindings rather than fabricating values (`program.go:413`; `invocation_payload.go:27`). `author/retained_handles_runtime_test.go:15` reaches the wrong-declaration Await with two authentic results and verifies neither can be rebound.
- **Authority comes from retained metadata and deployment.** `composition_binding.go:135` rejects undeclared authority-bearing endpoints; `:200` validates exact binding identities/member maps; `:246` resolves only declared endpoint operands into an ephemeral interpretation of unchanged artifact bytes. `invocation_catalogue.go:20` verifies deployed templates and public schemas. Payload strings do not select peers, grants, roots or code.
- **Admission and debit are atomic and immutable.** `invocation.go:220` separates source authorization from the final admission transaction. The transaction rechecks current source and ancestor closure, declarations, allowlists and depth/work limits, then retains all origins/runs/bindings/request observations and charges together (`:283`, `:367`, `:494`). Repeated operations return their original accepted/rejected decision. `composition_reservation.go:28` and `:60` perform the analogous final closure/effect-ceiling check and one-time reservation charge for adapter and legacy effects. The shared-category test at `composition_limits_integration_test.go:415` prevents an allowance per adapter category.
- **Public completion is distinct from local termination.** `invocation_outcome.go:222` freezes the committed prefix at the first applicable terminal boundary. `:324` waits for frozen reply admissions and, on success, admitted descendant/external joins. `:627` traverses descendants below a known failed child so that its early failure publication cannot hide unresolved grandchildren. `composition_root.go:11` provides the corresponding separate root publication. `prepareInvocation` returns the original published result (`invocation_outcome.go:291`), and late callbacks cannot rewrite it.
- **Lifecycle and physical ownership boundaries remain explicit.** `invocation_closure.go:65` distinguishes actual transitions, retirement after local success, no-op cancellation, earlier failure and prior publication. `invocation.go:165` enforces ancestor scope closure; reservation before closure can still launch under its original authority. `runner.go:53` keeps one Run owner and both global and persisted per-root active counts; waiting parents never acquire transport slots. `external.go:245` rejects synchronous dispatch for the complete profile, including unscoped work, before mutation/contact. Unknown adapter candidates create no known result (`adapters.go:268`) and retained allowed attempts are not automatically retried (`runner.go:578`).
- **Recovery reconstructs rather than repairs.** `composition_replay.go:17` derives exact dynamic origins, requests, permissions, charges, results and delivery links, and `:373` validates their ordered prefixes. `composition_prefix.go:255` requires the exact represented consumption predecessors, including the separately retained source authorization. `replay.go:22` has a read-only connection/transaction and no adapter registry. `store.go:540` obtains the existing owner lock and validates the entire store before advancing activation epochs. Schema3 and complete schema4 take explicit branches; the unfinished foundation profile is not silently promoted (`replay.go:79`, `composition_store.go:263`).
- **The consumer uses independent expectations.** `examples/worktree/consumer/contract_test.go:18` freezes literal IDs, inputs and expected outputs. Its count/direct/nested checks use one retained deployment per profile; empty work, duplicate payloads, rejected duplicate IDs/ninth item and lower budgets are separate cases. The faulty ordinary collector actually executes and passes runtime reconstruction while the same literal mapping oracle rejects its wrong values (`:289`). `cmd/alan-compose/main.go:190` and `:257` bound input and reject null, duplicate and case-alias fields before lossy struct decoding.
- **Process claims use actual causality.** `internal/compositiontest/process_test.go:89` joins acceptance to the collector's consumed request and exact history successor, not transport intake. `:122` joins ACKs to durable transactions and does not invent invocation-hook epoch data. `:360` checks actual SIGKILL status; recovery runs the test executable's public Open/Run path without reprovisioning. The effectful partial case retains a genuinely unknown held operation, distinct from deterministic partial collection. Operator closure kills/waits for the old owner before its separate transition. `ownership_test.go:26` supplies an unchanged physical-slot monitor for the executed waiting-parent fault; the report labels evidence mutations separately from runtime faults.

## Issues

### Critical — Must Fix

None found.

### Important — Should Fix

None found.

### Minor — Non-blocking

**M1 — Reproduction command metadata inherits the freeze host.**

- Location: `docs/foundational/phase1g/tools/qualify.py:100`.
- `Platform` and `GoVersion` are copied from the source manifest even for a fresh execution. A later run on another compatible host/toolchain therefore attributes that run to the original freeze environment. This is a reproducibility-metadata defect, not a runtime defect or reason to invalidate the retained same-host qualification; `checks.py:18` independently captures its executing platform.
- Capture the executing platform and `go version` for new runs. Keep freeze metadata under distinct names if useful, and preserve original execution metadata when parsing existing logs. A correction to this frozen tool must receive a new source identity and explicitly scoped verification; do not relabel existing logs or archives.
- Disposition: confirms deferred Task4 M1 as **Minor**, suitable for a small follow-up rather than restarting the completed broad qualification solely for this issue.

**M2 — Older README sections contradict the new profile and omit its consumer command.**

- Locations: `README.md:185`, `:156`, `:166`, `:170` (compare the accurate new section at `:66`). The final “Current limits” paragraph still says dynamic creation is absent and this increment uses only schema3. The location table still calls the schemas scalar, and the tests block omits the separate worktree consumer module while the following paragraph broadly says external-consumer gates are included.
- A reader can conclude that bounded template creation/schema4 is unsupported, or execute the root commands and miss the Phase1G separate-module consumer. Root integration still covers the older external-consumer test, so that paragraph is ambiguous in its new context rather than evidence of a missing retained qualification run.
- Distinguish legacy schema3 from the complete schema4 profile, describe bounded retained-template creation and typed lists, and add explicit `go -C examples/worktree/consumer test` normal/race commands (with the qualified consumer watchdog) plus its vet/module checks or a clear link. The Phase1G runbook already has the correct operational guidance.
- Disposition: **Minor** documentation reconciliation. This concern was supplied by the controller during review and independently checked against the current README and consumer module.

## Prior findings and integration disposition

- Task1 scalar Append panic: guarded list-field/value validation remains at `schema/collections.go:297`, with the original malformed-kind regression family retained. No reopening of the issue found.
- Task2a source-execution-before-consumption finding and Task2c source-authorization-before-consumption finding: both predecessor relationships exist in `composition_prefix.go:295` and `:302`. The order-only regression in `composition_consumption_test.go:10` retains ordinary rows/digests and clean FKs while exchanging the fact order. Genuine unrepresented sources remain supported; the source-denied lifecycle control at `:73` is correctly described as an existing passing guard, not invented RED evidence.
- Task2a weak-FK Minor: **resolved for its stated reserved-profile guard** by `composition_integration_test.go:426`, which inserts valid relational references, checks FK cleanliness, and requires the exact unfinished-profile ErrVersion diagnostic. This does not claim individual corruption coverage for every reserved table.
- Task2b retirement finding: `invocation_closure.go:82` records separate ordered scope closure without rewriting local success. R7 prioritization and immutable prior publication are represented in both live preparation and replay. The seeded dynamic lifecycle-grant case remains conditional internal evidence; supported static/inherited cases are separately retained.
- Task2c failed-child/unfinished-grandchild, direct-dispatch bypass and missing origin/grant/reply-link findings: transitive join traversal, profile-wide DispatchExternal refusal and exact-row/reverse-link reconstruction remain present. Prior red inputs are retained and their current regressions appear in the qualified source.
- Task3 CLI null/exact duplicate/case-alias findings: the token pass at `cmd/alan-compose/main.go:257` now requires canonical field names before struct decoding. The retained handle test and faulty collector support different claims and are not substituted for one another.
- Task4 fixture/header/watchdog attempts: the report preserves their original input/source/status. Client header joins use X-Client-Request-Id as correlation only. Earlier interrupted/failed race and extraction attempts remain separate from the final-source pass. Deferred M1 is listed above rather than silently dropped.

## Evidence checked independently

Read-only recomputation confirmed all 257 contributing source files and aggregate identity `43b86781f819c6bf7a957956aceaf717a5808873114e8213bf1147620bdf2ec3`. Archive bytes are 26,496,535, SHA256 `423c1bff2aaf4e8a76a8813df6a1076977c2d8844db13e2b34919ea382732fb4`; all 499 member lengths/hashes match the archive manifest. The archive contains nested-module paths and the pinned dependency proxy; archived reports are disclosed checkpoints, with final results/reviews adjacent to avoid circular identities.

For each retained extracted stream, recomputed compressed/raw hashes, parsed Go events and matched actual metadata exit and pass counts:

| Mode | Go exit | Passing test events | Go elapsed seconds |
| --- | ---: | ---: | ---: |
| default | 0 | 764 | 74.594 |
| integration (`normal`) | 0 | 1006 | 72.673 |
| integration race | 0 | 1006 | 1087.721 |
| separate consumer | 0 | 20 | 7.105 |
| separate consumer race | 0 | 20 | 103.435 |

No failure event appears in those five streams. `evidence/task4/extracted/reproduction.json` records all nine outer commands at exit0 and PASS; `extracted/checks.json` records eight auxiliary exit0 results. These are inspected retained executions, not new test runs by this reviewer. The three comparison commands cover the declared 15 QUALIFICATION/20 COMPOSITION/32 SUPERVISION payloads and six CONFORMANCE projections. The historical research controls remain inputs, not replacements for new composition checks. Independently hashed the baseline's 1,120 prior artifacts: no differences.

Parsed the final normal WORKTREE records: 35 starts/exits, 13 actual killed exits with numeric -1, six crash cuts, two recovery interruptions, four operator closures, one held-effect partial/unknown recovery pair, and two duplicate-result schedules. The SIGKILL witnesses include records3/10/20/73/430/455/458/476/495/514/517/535/554. Counts are from this one run and are not summed across race/normal. Source inspection verifies the relevant monitors use accepted IDs/values, exact durable boundaries, causal provider acknowledgments and unchanged before/after snapshots. The scoped Task4 review additionally traces the detailed normal record joins; no claim of exhaustive schedule exploration follows.

## Named checks of existing code

Unchanged portions of changed files were consulted only for concrete integration risks: `store.go:540` and `:730` for ownership/activation and stale or duplicate consumption; `messages.go:110` for retained authorization versus the narrow public-reply exception; `program.go:360` for exact receive matching after serialized handles; `runner.go:53` and `:578` for existing global slots and unknown retry policy; `adapters.go:268` and `:291` for unknown/late-result behavior; `replay.go:22` and `inspect.go:28` for read-only access. No unrelated application/provider implementation was reviewed for correctness. Historical unchanged files were hashed to establish preservation, which is not a new correctness review of their content.

## Behaviors considered and declined to judge

Every item below is explicitly outside this review's affirmative conclusion; none is silently treated as passing:

- Unbounded safety, fairness, eventual completion or all interleavings: finite tests and source inspection do not prove them; no such claim is required.
- Arbitrary recursion or dynamic code/deployment/grant creation: only the closed retained catalogue and explicit depth/resource envelope are implemented and qualified.
- Reusable long-lived worker pools: the reviewed design chose bounded durable invocation; static-pool reuse is an alternative, not missing scope.
- Multi-host execution, distributed fencing, ownership transfer or concurrently used store copies: expressly excluded by the single-host/one-owner contract.
- Automatic group cancellation, compensation, cleanup custodians or child reassociation: R1/R3/R7 specify scope closure and retained late evidence instead.
- Public dynamic lifecycle-grant delegation: the seeded internal retirement probe does not establish that unsupported authoring surface; static/inherited closure is separately supported.
- Remote operation termination or business exactly-once behavior: an unknown local result retains uncertainty, and the client-request header proves correlation only.
- Direct adapter/HTTP calls made outside the executor: they are outside Run's physical ownership guarantee; all supported complete-profile Store transport paths were assessed.
- Live provider authorization, cost or model answer quality: no live work is authorized in this increment; controlled transport and unchanged prior evidence cannot establish new quality claims.
- Native collection support in the Responses adapter: `adapters/openai/openai.go:48` explicitly rejects collection schemas; the reused evidence leaf has its existing scalar contract. This bounded extension does not promise every adapter handles lists.
- Schema migration, general corrupt-store repair or arbitrary malicious rewriting of every mutually consistent source row: the profile rejects unsupported/inconsistent histories and does not supply migration or cryptographic external provenance.
- Power loss, disk failure, remote-filesystem lock semantics or all operating systems: actual SIGKILL and the qualified macOS host do not establish those guarantees.
- OS-level network denial on hosts without sandbox-exec: the harness reports its absence; only the observed qualified host supplies that additional guarantee.
- Process-kill qualification of the alan-compose executable: the killed binary is the test executable using public Open/Run, while the CLI has distinct functional tests; the report states this accurately.
- Deterministic partial cut as proof of unfinished deterministic leaf computation: that cut has one collected and two known/unconsumed outcomes; the held-effect fixture separately proves remaining unknown work.
- Duplicate injection as proof that the next Await PC was already reached: the append-cut probe makes no such claim; the saved-handle test independently reaches its Await boundary.
- Monitor-input mutations as executed runtime faults for every invariant: the report classifies them honestly; only the actual faulty collector, waiting-parent runtime overlay and retained real corruption cases support their respective stronger claims.
- Exhaustive individual corruption tests for every reserved table: Task2a M1's targeted guard is covered, but that is not an exhaustive matrix.
- Performance/scalability guarantees or optimized startup/replay complexity: histories and retained lists are bounded for this qualification, timings are measurements, and full-history validation remains disclosed.
- Compatibility with malformed legacy values that were never in supported historical evidence: no expansion of that compatibility promise is inferred; retained supported scalar identities and histories were checked.
- Stable SDK, production hardening, UI, provider catalogue, deployment/release or automatic merge: all remain outside the provisional delivery requested here.

## Recommendations and assessment

Reconcile M2 in current documentation and retain M1 as an explicit small tooling follow-up or fix it under a fresh source identity. Preserve all original evidence and do not rerun broad suites without a source change or new concrete concern. Any subsequent runtime/contract change needs targeted rereview and correctly attributed verification.

**Approved within the bounded Phase1G scope; ready to publish the requested draft PR for human review.** The integrated authoring/runtime/consumer paths implement data-dependent retained work with root-shared limits and immutable outcomes, and the inspected evidence supports the stated finite recovery and no-dispatch replay claims. The two Minor findings do not undermine the qualified runtime result. This approval does not authorize merge/release and does not assert unbounded correctness or production readiness.
