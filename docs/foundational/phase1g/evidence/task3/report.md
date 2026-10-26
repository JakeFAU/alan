# Phase1G Task3 — public ordinary collection application

Implemented against reviewed runtime base `347aea866550ceaf21de5e04452b6cf0dcb6e2dd` on `codex/phase1g-data-composition`. Task3 implementation and scoped verification are complete. Controller review and Task4 process/full-tree qualification remain separate. No runtime scheduler, executor opcode, SQL, storage version, dependency, provider catalogue, prompt, historical fixture or earlier evidence was changed.

## Implementation and public API

`compose.New(leaf author.Template, options compose.Options) (author.Template,error)` compiles one ordinary collection program. Defaults are eight items, 262144 canonical bytes per retained list, and 128 UTF-8 bytes per item ID. Its request is `{items:[{item_id,...leaf input}]}`. `item_id` is reserved: a conflicting leaf input schema rejects explicitly. Both accepted input and handle/result lists retain distinct item IDs; equal payloads are separate work.

The program first Receives its work list, scatters Invoke+Append into a typed ordered handle list, and only then traverses that list to Await each original operation. Invoke commits pending correlation before the later Append; an interruption between those steps resumes that Append using the existing pending identity. No item-dependent Go factory, topology expansion, private child inspection, special executor instruction or callback implements collection policy. Responses retain `{item_id,operation,status,reason,responses}`; `responses` is a bounded zero-or-one list of the leaf's exact response record. Presence is tested before reading optional payload bindings. Empty response records, absent enum-constrained payloads and stale prior payload bindings are handled without fabricated defaults.

Empty input emits no item Invoke/effect and returns `{all_succeeded:true,results:[]}`. Ordinary enclosing calls still charge normally. Every known non-success produces an ordered item outcome and a failed aggregate after its diagnostic public reply. Unknown required work waits on its original handle. Integer overflow produces an explicit Run error, preserves the accepted work/operations, and invents no successful result.

`author.Template.Schemas() (input, output schema.Retained, err error)` is the narrow read-only authoring improvement. It validates and returns newly decoded copied public contracts; the zero template errors. Generic consumers do not decode private template bundles. Copy/conflict/zero tests exercise the consumer-visible contract.

A second concrete authoring obstacle was discovered by the actual two-loop program: Invoke's static `request:invocation:item` type could neither enter a string handle field nor be recovered from that list for AwaitInvocation. The controller authorized a compiler-only refinement. Request identities are accepted at ordinary schema string payload/list boundaries; AwaitInvocation accepts a plain string variable. Known `request:other` mismatches and non-string handles remain rejected, and legacy adapter Await still requires its known declaration. Serializing a handle loses static declaration proof; runtime provenance, template, generation, declaration and exact original operation matching remain authoritative. No artifact interpretation, emitted opcode, semantic/compiler version or old artifact bytes changed. The exact compiler RED input and original compiler are retained.

`examples/worktree` supplies `Deterministic()`, `Evidence(evidence.Budgets)`, `DefaultLimits(capacity)`, `New(leaf,nested,limits)` and `Submit(ctx,store,identity,items)`. New chooses a closed retained catalogue, one generation1 static root/member, explicit template/capability allowlists and the shared immutable resource envelope. Direct depth is root→collection→leaf; nested inserts an ordinary wrapper and reaches depth3. The collector template and both public schemas are identical between profiles. Evidence uses `evidence.New`'s public Instances, two unchanged programs and existing policy; only their empty initial input maps are made explicit for template retention. Legacy factories/artifact identity assertions pass in the default suite.

`cmd/alan-compose` offers init, submit/admit, run/recover, inspect and replay. Init finishes with no work list. Submit reads bounded UTF-8 plain JSON records after initialization, rejects null/duplicate fields/unknown fields and uses local-input admission. Run/recover opens retained artifacts and calls the existing automatic Run; it does not call an application factory, recompile or reconstruct work. Inspect reads the coherent runtime snapshot and its own root projection, exposing both local status and structured root completion. `boundary.FromEnvironment` preserves existing inherited-pipe observational hooks for Task4. The evidence CLI accepts only explicit literal-loopback HTTP controlled Responses endpoints and retains a nonsensitive one-line test asset; it reads no credentials or live provider configuration. Tests use the existing Responses/lookup adapters and controlled HTTP handlers.

## Public runbook

The complete command/Go API runbook is [examples/worktree/README.md](../../../../../examples/worktree/README.md). From the repository root:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build
go run ./cmd/alan-compose init --db /tmp/collection.sqlite --capacity 2 --nested
# Create this input only after init has returned.
cat > /tmp/work.json <<'JSON'
[{"item_id":"item-a","number":2},{"item_id":"item-b","number":-3},{"item_id":"item-c","number":2}]
JSON
go run ./cmd/alan-compose submit --db /tmp/collection.sqlite --id one-work --input /tmp/work.json
go run ./cmd/alan-compose run --db /tmp/collection.sqlite
go run ./cmd/alan-compose inspect --db /tmp/collection.sqlite
go run ./cmd/alan-compose replay --db /tmp/collection.sqlite
go run ./cmd/alan-compose recover --db /tmp/collection.sqlite
```

Use a new store for a new root obligation. Retrying the same input transport retains its original binding; an existing root's allowances do not reset. `run` exit0 means the runner returned normally; `Roots[].Outcome` determines public completion. Root outcomes have no caller reply port, so the collection's result is in the root's own projection (`inspect.Result`), while `Snapshot.Roots` exposes completion, charges and unresolved/unknown evidence. Local quiescence and a local terminal are not substituted for public completion. Reopen/replay do not need the application builder. For evidence, init with `--leaf evidence`, submit items containing `question` instead of `number`, and supply `run --controlled-endpoint http://127.0.0.1:PORT` with a controlled Responses server.

## Identities and constant versus proportional costs

[artifact-identities.json](artifact-identities.json) retains every public template/program ID, program byte length/semantics, initial run and direct/nested entry/collection schema ID. [32-public-artifact-identities.log](32-public-artifact-identities.log) is actual public-only measurement output, not a calculated expected-output oracle. Its supplementary executable input and SHA256 are retained in `commands.json`.

| Leaf | Initial static runs | Catalogue templates direct/nested | Program artifacts direct/nested | Common collector bundle bytes |
| --- | ---: | ---: | ---: | ---: |
| Deterministic integer | 1 | 2 / 3 | 3 / 4 | 24097 |
| Existing two-program evidence helper | 1 | 2 / 3 | 4 / 5 | 97748 |

The deterministic collector is `e14ef83c379f0dd1f1c0684cce4f56b3166eb427ab236f08cbaa50121f902b3e`, with request schema `839a0ed11c1763207ea608ca48238e296bab0cf18ef636ecbe63a31a6ad66bcf`, response schema `e9bb2d71fe60cfc122fc2ed7948341208cc84b0e47a51d84bbc1938b40f47f52`, and collector program `1d0d9772899fba53f71cdc20b579f9f8c57b5b25b77ba9983ba75e777b250a30` (13918 bytes). Its nested wrapper template is `aa440b4ae73d9c0920423f911b64e0e70c4f57506fca862b18243297c4996ba7`.

The evidence collector is `064f0a9994bf5e85858608903bbe0caab1b6ff28d8c785c943183a2bc2a937b8`, with request schema `aab20ff18a22f489197bd1f3f3ff4a4fb883c4df7b56a07e22ad9af6c3efa5fd`, response schema `e50726c3addea168ead44a6c842bc383c677c395390a650acb66e885161947c1`, and leaf template `69dfc68ebf5abf335be6f1f959c519227849c35edcd7120d2056b2ee31c18443`. The public-only identity command explicitly checks that each wrapper's own schemas equal its collection schemas. Wrapper/root program identities may differ by profile; collector bytes/ID do not.

The frozen integer prefixes are exactly `2,-3,2,0,11,-1,9,4` with expected outputs `4,-6,4,0,22,-2,18,8`, IDs item-a through item-h. item-i=100 is the rejected ninth item. [input-0.json](input-0.json), input-1/3/7/8/9.json and input-duplicate.json retain these transport fixtures. Their literal expected mapping was written before production implementation; no compiler/reducer/replay calculates the oracle.

Actual retained-record measurements from [26-consumer-final.log](26-consumer-final.log):

| n | Direct admissions / created runs / results | Nested admissions / created runs / results | Direct intents / history | Nested intents / history |
| ---: | --- | --- | --- | --- |
| 0 | 1 / 1 / 1 | 2 / 2 / 2 | 2 / 23 | 4 / 31 |
| 1 | 2 / 2 / 2 | 3 / 3 / 3 | 4 / 41 | 6 / 49 |
| 3 | 4 / 4 / 4 | 5 / 5 / 5 | 8 / 77 | 10 / 85 |
| 7 | 8 / 8 / 8 | 9 / 9 / 9 | 16 / 149 | 18 / 157 |
| 8 | 9 / 9 / 9 | 10 / 10 / 10 | 18 / 167 | 20 / 175 |

These result counts include enclosing collection/wrapper results; the public result list has exactly n entries. All deterministic effects are zero. Initial static root cost is one and is excluded from the created-run/admission charge. The collection emits n item calls; the driver adds one enclosing call and the nested wrapper adds another. Evidence leaves create two member runs per item: created runs are `2n+1` direct or `2n+2` nested. Clarification uses one model effect per item; the actual answer uses two model effects plus one lookup. No allowance is waived for an empty enclosing call.

`authoring-metrics.json` records physical lines including comments/blanks, byte counts and hashes: compose builder113 lines/5466 bytes; worktree application140/6961; CLI297/9017. The public builder/application total253 physical lines; no helper-per-item factory expansion is counted as collection support. Normal measured direct/nested n8 Run latencies were about1.18s/1.22s in command26, and the complete consumer test took8.412s. These are local observations, not throughput guarantees.

## Causal evidence and independent monitors

[28-evidence-final.log](28-evidence-final.log) records capacities1 and2 for both profiles with three logical leaves. A handler acknowledges the first held transport. The admission observer then acknowledges complete scatter (4 direct or5 nested invocations) while that transport remains held. The independent acceptance join verifies the collector's exact accepted items before release. At capacity2 a later leaf's immutable result is acknowledged and inspected while the first request remains held and its outcome is absent. Final results retain original order and every required leaf completes after release. Observed controlled-handler peaks are exactly1/2 respectively, under Run capacity2 and persisted root limits1/2. Timeouts only detect hangs; no sleep establishes order.

The unknown provider applies/drops its response, producing an original unknown operation while the sibling returns `clarify:independent`. Reopening and Run retain the same unknown ID and two spent effects, with contacts exactly2 before and after; aggregate outcome remains absent. This is recovery of the same obligation, not automatic business retry/deduplication. The log retains original unknown ID, sibling outcome/operation and contacts.

The answer/invalid/effect-limit fixtures retain planning, synthesis and lookup operation keys/source runs plus public citation fields and root charges. The answer cites document durability, immutable version `38d8f72fbdc8db91555da78282c78d1058664782bc97978c38eefa1490c088f8`, span0..54, and the exact controlled sentence. A mismatched synthesis citation preserves `invalid_result` with failed status; an effect ceiling1 allows planning, denies lookup, retains its negative result, and preserves failed diagnostic policy. No prompt/corpus quality study or provider spend is claimed.

Lower invocation/run limits2 permit collector acceptance of three items, then retain one success and two ordered rejections. Depth1 permits collector acceptance but rejects every leaf. These are accepted work followed by item rejection; an enclosing invocation rejected before its Receive is a different boundary. Optional payload/empty-record tests preserve presence and absence separately. Overflow is an explicit error with no fabricated public result.

The compiler handle negative reaches two authentic recorded successful outcomes with exact original operation IDs, then attempts to Await first using the serialized second declaration's handle. It reaches the actual receive PC, consumes neither unrelated outcome and remains open; Replay accepts this valid pending history. Static wrong-known-declaration, non-string and legacy adapter plain-string negatives also pass.

`TestFaultyOrdinaryCollectorReachesIndependentMappingMonitor` in the separate module deliberately invokes each authentic leaf with0 while preserving accepted item IDs, distinct handles, schema validity and runtime joins. Runtime root completion succeeds and Replay accepts its identity history. The exact same literal mapping oracle used by the positive fixture rejects `item-a=0` versus required4. Task4 should reuse this test/oracle/control rather than create another version. This demonstrates a public application mapping obligation that runtime identity guards do not secretly repair.

## Verification, original REDs and source identity

Every actual command uses `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`. [commands.json](commands.json) records exact argv, numeric subprocess exits, output and the exact Go/go.mod/go.sum source identity. Adjacent `.source.json` files retain per-file SHA256 manifests; the command capture wrapper's shell exit is never substituted for the actual Go exit. [original-input-verification.json](original-input-verification.json) verifies10/10 retained original test/compiler/pre-fix CLI source files against their captured hashes. Original logs/inputs are never overwritten by GREEN output.

Behavioral RED→GREEN pairs: handle serialization06→07 (`append field type mismatch`→successful two-loop authoring); null work18→19 (null incorrectly became an empty list→explicit error); duplicate JSON identity fields22→23 (silent replacement→explicit error). The original two-loop source and original compiler are retained byte-for-byte. Missing-package02 and missing-CLI-API05 are setup/compile REDs, not runtime behavioral proofs.

Other original failures remain labeled honestly:01 matched no root packages because the consumer is a nested module;03 exposed missing explicit initial data maps;04 repeated the handle type obstacle;08 used a missing local-port Peer;09 used an incorrect input Operation;13's controlled fixture credential equaled its retained model string and correctly hit existing credential-retention refusal;15 used a nonexistent exported constant in a test. These are not claimed as successful runs or independent runtime counterexamples. The same fixture shapes were corrected without tuning frozen item/resource bounds. Command30 hit a90s race watchdog while ordinary validation was progressing; its full original stack/log remains. Command31 changes only that watchdog to240s, not item/model/depth/resource bounds, and passed in95.007s with no source changes. This exposes substantial race-instrumented validation overhead; it is neither a liveness argument nor a performance target failure.

Final-source checks use `735c85c420c5cac56729afa8df1ae45252ab8ffea422e4673ed8042870085750`:

| Evidence | Exact command | Actual exit |
| --- | --- | ---: |
| 25-author-focused-final | `go test ./author ./compose ./cmd/alan-compose -count=1` | 0 |
| 26-consumer-final | `go -C examples/worktree/consumer test . -count=1 -v` | 0 |
| 27-default-final | `go test ./... -count=1` | 0 |
| 28-evidence-final | `go test -tags=integration ./examples/worktree -count=1 -v -timeout=100s` | 0 |
| 29-consumer-vet | `go -C examples/worktree/consumer vet ./...` | 0 |
| 30-consumer-race | `go -C examples/worktree/consumer test -race ./... -count=1 -timeout=90s` | 1 (watchdog) |
| 31-consumer-race-expanded-watchdog | `go -C examples/worktree/consumer test -race ./... -count=1 -timeout=240s` | 0 |
| 32-public-artifact-identities | `go run /tmp/alan-task3-identities.go` (retained input) | 0 |

Default passed17 tested packages; three packages had no default tests. Root `go test ./...` does not enter the nested module, hence the explicit consumer commands. No Go source changed after these final-source checks. Formatting/source whitespace checks are retained in final-format-checks.json. Full-tree integration/race/vet and final source archive remain Task4.

## Task4 acceptance/process handoff and limits

The existing CLI inherited-fd boundary environment is supported. Work has four distinct boundaries: external transport admission; root intake Receive (currently root sequence3); outer collector/wrapper Invoke admission; and the **collector's own public Receive at generation1 sequence5**, after its four ordinary initializers. Wrapper Receive is earlier and is not collector acceptance. Actual acceptance is not established by `submit` returning, `invocation.admitted`, or the outer root projection alone.

The `collectionAcceptance` test monitor joins exact relationships from public coherent Inspection: find the invocation with the common collection template ID; join `composition_requests.invocation_id` to its original observation ID; require that observation's `consumed_sequence`; join the same subject/generation/history sequence and selected observation; decode that history successor as public `runtime.Projection`; require its `Data["items"]` to equal the exact correlated ordered input. It returns/logs invocation, original operation, collector member subject/generation, selected observation, Receive sequence and accepted items. [28-evidence-final.log](28-evidence-final.log) retains all four actual acceptance identities before releasing physical work. Dynamic member subjects vary by namespace/invocation: resolve the actual admitted collector and filter its exact subject/sequence, rather than treating every sequence5 event as acceptance. The monitor reads children only as test evidence; application behavior never does so.

Task4 owns actual SIGKILL PIDs/exits and fresh-process reopen at collector acceptance, partial collection, and recorded-result-before-consumption; late/duplicate/stale result attempts, closure races, additional ledger corruption/negative variants, denied-network read-only replay, final whole-source archive and full-tree integration/race/vet/publication. These are not claimed by Task3. Existing runtime hooks separate admission and immutable result recording from parent consumption. Preserve the current oracle/faulty ordinary collector and all earlier Phase1F/1G artifact/evidence identities.

Self-review checked the two-loop separation, exact pending-handle retention, optional payload guards, duplicate/over-limit pre-child refusal, resource charge accounting, copied schemas, generic leaf-field conflict handling, absence of executor/SQL/private-child behavior, local-only controlled transport, recovery without authoring, real collector acceptance, and unchanged legacy identity tests. There is no remaining known functional blocker in this slice. Scope remains bounded single-host examples with finite traces, not universal safety/liveness, arbitrary recursion, external business deduplication, compensation, custody/cleanup, SDK stability or answer-quality qualification. There are no live provider calls, real credentials, new dependencies, child/reviewer subagents, commits outside this slice, push or PR in this task.
