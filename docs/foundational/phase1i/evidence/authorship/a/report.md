# Phase 1I candidate A first-version report

Question: can the assigned local, singleton, ordinary-program realization implement the finite K1 mode table with one entry admission and no helper/effect? Result: all 16 concrete request qualifications passed. This is tested implementation behavior and shared-runtime replay consistency; independent local refinement, reference products, product census, correspondence review, and any theorem remain separate controller obligations.

## Authority and independence

Read the frozen candidate brief first, repository AGENTS.md, full K1 contract and Stage1 rereview. Verified K1 SHA256 `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`; rereview decision APPROVE authorizes specification quality only. Architecture A=local was assigned, not independently selected. Authored mode/value/reply logic independently; did not inspect candidate B source, artifacts or reports, and did not read model-author logic. No subagents, shared/production edits, dependencies, full-suite tests, provider calls, commits, push or PR. Only assigned `examples/substitution/a`, `evidence/authorship/a`, and /tmp outputs written. Scoped escalated writes were used because the managed worktree is outside the current writable roots; the primary checkout was untouched.

## Inspected public API and authoring

Read `author/author.go`, `author/invocations.go`, `author/application.go`, `author/invocations_test.go`, `runtime/runtime.go`, `schema/schema.go`, and schema-only `examples/substitution/protocol/protocol.go`. Public calls used: `author.Compile`, `Definition`, `Receive`, `IfEqual`, `Send`, `Success`, `Failure`, `Var`, `String`, `Integer`, `Boolean`, `NewTemplate`, `Template.ID/Bytes`, `Application.Provision`, `Invoke`, `AwaitInvocation`, `InvocationOutcomeBindings`; `runtime.Create`, `Store.Run`, `Store.Snapshot`, `Store.Close`, `Inspect`, `Replay`, `DefaultLimits`; and schema constructors/values. Inspected author compiler lowering and the underlying type definitions for public runtime aliases (Inspection, RunReport, RunState, ResourceRoot/Limits, InvocationOutcome, RootCompletion, ReplayReport), including composition inspection table shapes. Candidate/tests import no internal package, raw SQL, or host callback. The testing assertion that input is consumed reads the candidate's retained request registers through public Snapshot; expected response/envelope/charge assertions use public observations.

## Frozen first version

Live implementation: `examples/substitution/a/candidate.go`; tests: `examples/substitution/a/candidate_test.go`.

Exact first source snapshots are `candidate.go.txt` and `candidate_test.go.txt` in this evidence directory. `first-version-sha256.json` records source, exact template/program bytes, RED log, ID and prequalification bound note hashes. Source and retained bytes were frozen before the first runtime qualification; no source edits followed. A later export was byte-identical.

- Candidate source SHA256: `c98a5c6d16738d98af9140e6edce6d4df0ddcb51c2330ba974e0e51a82005911`.
- Test source SHA256: `033373ad1e384b32b6b9597faa2ed23a3f74b8f8015371d17b4c2df9d3503710`.
- Template ID: `a03135203f8b728a6aa7b732d06952cc87b70b3aced2c0833293e333760eece5`.
- Exact `template.json` SHA256: `db5453b6e08abe5ecbf9614d472626c228675ba197abdaa36cad4aa3c5b04409` (9979 bytes; template ID is the descriptor identity, not a claim that it hashes the whole bundle).
- Singleton retained artifact: `74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7.program.json`, SHA256 identical to artifact ID; 6762 bytes, semantics `alan/1e/v1`, compiler `alan/author/v1`, 13 instructions, zero initial data fields, zero initialization Sets, zero invocation declarations, zero effects.

## Exact retained structure for independent mapping

The template has only member `entry` and public port `public`, peer placeholder `caller`, obligation `substitution`, generation 1. Public request/response schemas come unchanged from `protocol.Request/Response`. Integer domain bounds are semantic restrictions; out-of-domain inputs were not claimed schema-rejected.

Zero-based instruction PCs:

| PC | Retained action / edge |
|---|---|
|0|Receive ordinary message: peer caller, generation 1, obligation substitution, empty operation guard; bind id/request_id, value/request_value, mode/request_mode.|
|1|Branch request_mode == answer: true 2, false 7.|
|2|Branch request_value == 0: true 3, false 5.|
|3|Emit one public reply {id:request_id,value:1}; fallthrough 4.|
|4|Forward jump 6 (compiler branch join).|
|5|Emit one public reply {id:request_id,value:0}; fallthrough 6.|
|6|End succeeded.|
|7|Branch request_mode == fail: true 8, false 9.|
|8|End failed.|
|9|Branch request_mode == absent: true 10, false 11 (enum/domain leaves pending).|
|10|End succeeded with no reply.|
|11|Second Receive, same public request guard as PC0, no bindings; blocks because the environment supplies no second request.|
|12|End succeeded, reachable only with an extra environment input outside this experiment.|

Both value branches and all mode branches are retained. Every control edge advances; there is no loop, recursion, padding, helper or external effect. The static guard uses the public caller placeholder; normal invocation binding supplies the actual caller identity. No candidate operation handle is introduced. Qualification caller commits exactly one `candidate` invocation into `handle`, awaits that exact handle, binds status/reason/presence and id/value response, succeeds iff succeeded+present, otherwise fails. It never corrects response data. Pending leaves the caller at that original AwaitInvocation. The caller has 7 retained instructions (per inspected qualification program); all variants are archived inside per-request Inspect tables.

## Commands actually executed and outcomes

All Go test commands explicitly used:

```
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build ...
```

Toolchain checked: `go version go1.27.3 darwin/arm64`.

1. Wrote all16 observable public API tests first, with a minimal compileable New stub returning `candidate A is not implemented`. Ran `go test -timeout=10m -count=1 -v ./examples/substitution/a`: actual RED, all16 subtests failed at that explicit error, export skipped without environment output. A first shell logging wrapper used zsh read-only variable `status`, so its post-test wrapper failed; inspected its retained RED log and reran the exact test directly, confirming exit1. This is TDD authoring RED, not a semantic negative control and not a compiler rejection.
2. Implemented candidate narrowly and ran `gofmt -w examples/substitution/a/candidate.go examples/substitution/a/candidate_test.go`.
3. Export-only before runtime qualification: `ALAN_PHASE1I_A_OUTPUT=/tmp/alan-phase1i-a-first go test -timeout=10m -count=1 -v -run '^TestCandidateAExport$' ./examples/substitution/a`: PASS. Inspected exact emitted Code; fixed operational bounds before runtime qualification; froze first source/artifacts/hash manifest.
4. Qualification: `ALAN_PHASE1I_A_OUTPUT=/tmp/alan-phase1i-a-qualification go test -timeout=10m -count=1 -v ./examples/substitution/a`: PASS, all16 cases and export. `green.log` preserves complete output, `qualification/*.json` preserves Run report, caller/child snapshots, full Inspect tables and Replay report for every case; `qualification-index.json` records exact hashes and counts.
5. Python verification checked first manifest hashes, exact current-source equality to frozen snapshots, and repeat template byte equality: PASS. `git status --short -- examples/substitution/a docs/foundational/phase1i/evidence/authorship/a` showed only the assigned new directories. `git diff --check -- ...` produced no error; these new files are untracked, so formatting is separately checked with gofmt.

Two exploratory reads named absent paths `runtime/composition.go` and `internal/executor/composition_schema.go`; cat/sed reported not found. Located actual alias/type/schema files with rg and read them. No behavior conclusion depended on those failed reads.

## Qualification and bounds

All runs: I=R=3, maximum depth3, effects0, positive physical capacity MaxInFlight1, generation1. Actual charges exactly (1,1,0), one admission, one dynamic singleton, two total runs including the fixed root. Helper denial is inapplicable by construction; A has no helper boundary.

- Four answer cases: local succeeded, child succeeded/member_succeeded with present id-preserving complement, caller/root succeeded. Steps11 for value0,10 for value1; dispatches3.
- Four fail cases: local failed, child failed/member_failed, no reply, caller/root failed. Steps8, dispatches2.
- Four absent cases: local succeeded, child failed/missing_reply, no reply, caller/root failed. Steps9, dispatches2.
- Four pending cases: local open at PC11, caller open at original await, no reply/publication/root completion and no consumed presence binding. Steps5, dispatches1. Quiescence is not a completion claim.

Every run reached quiescence with no cutoff and replay succeeded. Operational MaxSteps=128/MaxDispatch=128 fixed before runtime qualification. Candidate13 + caller7 instruction count gives a conservative20 ordinary-commit upper bound for this acyclic pair; longest feasible combined path11 is observed. Single entry/request/reply/publication/outcome chain bounds dispatch far below128 (actual max3); no helpers/external effects can amplify it. Each default byte cap is1MiB; inspected programs6762/4410 bytes and template9979 bytes are below it. Prequalification reasoning is retained in `operational-bounds-first.txt`; no limit was raised after observing runtime behavior.

## Limits and remaining obligations

No semantic negative-control variant was executed by this author. Scarce admission budgets, root closure, restart/crash, five contextual products, reference graph/local refinement, additional weak/equal/effect probes, census, correspondence review, and manuscript integration belong to the controller's separate stages. The 16 deterministic runs and shared reducer replay are not independent semantic conformance and establish no general substitution, contextual equivalence, fairness theorem, compiler correctness, or effectful result. No production semantics were changed; the manuscript's distinction between charge-preserving symmetric equivalence and K1 refinement remains material. No manuscript files were changed within this author-only scope.
