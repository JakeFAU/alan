# Phase 1I independent candidate B — first version frozen

Question: can one ordinary singleton helper, used in all four modes, realize the frozen K1 public reply/outcome and charge contract? Result: the authored artifact passes bounded public-runtime qualification for all 16 declared requests and 12 helper refusals. This is tested implementation behavior and replay consistency, not independent semantic conformance or a substitution theorem.

Worktree: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.
Contract SHA256 independently recomputed before implementation: `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`. Read the approved Stage 1 rereview in `docs/foundational/phase1i/evidence/reviews/contract-rereview-v1.md`. No candidate A source, artifact, report, or logic was read. No subagents, production edits, dependencies, paid calls, commits, push, or PR.

## Sources and freeze

Only authored package: `examples/substitution/b/b.go` and `b_test.go` (integration build tag on tests). Evidence: `docs/foundational/phase1i/evidence/authorship/b/`. Frozen at `2026-10-29T00:15:58.651161+00:00`; first exact source snapshots use `.go.txt` suffix. No source edits after this freeze.

- `b.go` SHA256: `ff813a85585175522e2bf5cc4550caeb6c169c7e507c7ce51ee8bba43ec933e8` (3,841 bytes).
- `b_test.go` SHA256: `8e845090cb8d05fbd3d352f3c9572e546de2a1d3c3527608e2a8e849a657c391` (13,046 bytes).
- Template ID: `2e10975004e0351d07b237e0360a2b96b50d5884b651cfa5bfc8e88af62da266`.
- Exact template bytes: `template.json`, 19,465 bytes, SHA256 `8693f675a4ec4be6d14a859157fbbf9d21b3608e4485d32b900421c0d7d7b5aa`. Template identity and bundle-byte digest are distinct.
- Candidate program: `ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67`, 9,533 bytes, 16 instructions.
- Helper program: `698256075f67b360cacb0b20e16c5aeafc561c76255aeb3fd1ca7481ef9e9a6b`, 3,951 bytes, 7 instructions.
- Nested helper template ID: `1806cf3a9e1e67ec0ed25a665b7a8a9be2dd7cf37585c5b1cd3e274b12595e3d`.

`first-version-sha256.json` lists exact source/template/program hashes and sizes. Per-program files preserve exact `runtime.Artifact.Bytes`, not pretty-printed reconstructions. `template.json` retains the candidate singleton descriptor, nested helper descriptor, both programs and public schemas. The freeze utility ran successfully; `freeze.log` is retained.

## Public APIs inspected and used

Read `author/author.go`, `author/invocations.go`, `author/application.go`, `author/invocations_test.go`, `author/compile.go`, `runtime/runtime.go`, `schema/schema.go`, and schema-only `examples/substitution/protocol/protocol.go`. Used `author.Compile`, `Definition`, ordinary `Receive`, `Invoke`, `AwaitInvocation`, `IfEqual`, `Set`, `Send`, `Success`/`Failure`, `Application`, `NewTemplate`, `Template.ID/Bytes/Schemas`; public `runtime.Create`, `Store.Run`, `Inspect`, `Replay` and public resource/inspection/outcome aliases; scalar `schema` constructors.

To understand fields of public runtime aliases, consulted their defining declarations in `internal/executor/{program.go,store.go,runner.go,replay.go,inspect.go,composition_types.go,composition_inspect.go,invocation.go}` and located inspection table definitions. None of the authored Go files imports an internal package; tests do not call a reducer/compiler as an independent oracle, raw SQL, host callbacks or other candidate logic. Runtime replay is explicitly treated as shared-implementation consistency.

## Artifact branch/event structure

Both templates contain exactly one ordinary `entry` member. No loops, Repeat, ForEach, effect, callback, padding Set, or backward edge. Public protocol schemas and Port/Peer/Obligation constants are used unchanged. Integer range membership is a K1 environment premise; schema kinds do not enforce the finite domains.

Candidate PCs (zero-based):

- 0 consumes original public request into `requested_id/requested_value/requested_mode`.
- 1 commits exactly one `complement` helper invocation, saving its exact handle in `helper_operation`; payload preserves id/value and sets mode to `answer`.
- 2 awaits exactly that original handle with runtime-invocation provenance, helper template ID, generation 1, and `complement` obligation.
- 3 branches helper status succeeded: yes 4, no 15 (Failure).
- 4 branches helper response present: yes 5, no 14 (Failure). Default-bound values are not treated as a present response.
- 5 branches mode answer: yes 6 Send forwarded helper id/value then 7 Success; no 8.
- 8 branches mode fail: yes 9 Failure, no 10.
- 10 branches mode absent: yes 11 Success without Send, no 12.
- 12 is a second ordinary public Receive, unmatched under K1's environment. It leaves pending open. PC 13 Failure could follow only an extra request outside that environment.
- 14/15 are Failure branches for absent/unsuccessful helper outcomes; ordinary refusal takes 15.

Helper PCs: 0 receives id/value/mode; 1 branches input_value==1 (yes 2 Set complement=0, 3 forward jump to 5; no 4 Set complement=1); 5 sends original id and complement; 6 Success. This has 6 or 7 executed instructions depending on the value. Private identifier names are not claimed equivalent to A's names.

In each of the 28 retained samples, the original helper outcome is consumed once by the candidate before any candidate Send. `verify-evidence.py.txt` independently checks these retained operation links/order and the frozen hashes using JSON inspection only; `qualification-summary.json` retains its results. This is an author-side evidence check, not the separately assigned independent K1 model/oracle.

## Tests and commands actually executed

All Go commands used prefix:

```sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build
```

No full suite was run, as instructed. Local formatting: `gofmt -w examples/substitution/b/b.go examples/substitution/b/b_test.go` (and the test file alone after adding evidence metadata/freeze utility).

Test-first sequence: wrote the public API tests and a compiling `New()` stub returning an empty retained template before the implementation. First attempt failed to compile due to an unused protocol import in the test; corrected the test to inspect the retained public schemas. Then ran:

```sh
go test -tags=integration ./examples/substitution/b -run '^TestNewContractRequests/id1-value0-answer$' -count=1 -timeout=10m -v
```

Observed RED: `New returned empty retained template; candidate B is not implemented` (exit 1, retained `red.log`). This is a missing-feature baseline, not a semantic negative control. Implemented the helper and candidate, then the same command passed (18 steps, 6 dispatches, 3 replayed runs; `initial-green.log`).

Ran all package tests before freeze:

```sh
go test -tags=integration ./examples/substitution/b -count=1 -timeout=10m -v
```

Exit 0, all 16 request cases and 12 refusal cases passed (`initial-qualification.log`).

With `ALAN_PHASE1I_B_FREEZE` set to the absolute B evidence directory, ran:

```sh
go test -tags=integration ./examples/substitution/b -run '^TestFreezeFirstVersion$' -count=1 -timeout=10m -v
```

Exit 0; retained exact template/program bytes (`freeze.log`), then generated `first-version-sha256.json` from them and the source snapshots using Python SHA256.

After freeze, with `ALAN_PHASE1I_B_EVIDENCE` set to the absolute `evidence/authorship/b/samples` directory, ran:

```sh
go test -tags=integration ./examples/substitution/b -run '^TestNew' -count=1 -timeout=10m -v
```

Exit 0, 28/28 scenario subtests passed; `qualification.log` and 28 full RunReport/Inspection/ReplayReport JSON samples retained (2,730,911 sample bytes). Each case creates a fresh real SQLite store, runs its fixed caller, inspects it read-only and replays it. Assertions distinguish candidate local status, child invocation envelope, response presence/value, helper request payload, parent/depth link, helper admission/outcome, exact charges/run counts, and reply counts. No fabricated admissions/receipts or database writes.

Finally ran `python3 docs/foundational/phase1i/evidence/authorship/b/verify-evidence.py.txt`: exit 0; source/artifact hashes match, all 28 original helper outcomes consumed, every candidate Send follows that consumption, maximum 19 steps/6 dispatches.

Resource/results table:

| Cases | Limits I/R/depth | Candidate local | Invocation envelope | Charge I/R/E |
| --- | --- | --- | --- | --- |
| 4 answer | 2/2/2 | succeeded | succeeded/member_succeeded, present same id and 1-value | 2/2/0 |
| 4 fail | 2/2/2 | failed | failed/member_failed, absent | 2/2/0 |
| 4 absent | 2/2/2 | succeeded | failed/missing_reply, absent | 2/2/0 |
| 4 pending | 2/2/2 | open | unpublished, caller awaiting original child | 2/2/0 |
| 4 helper I refusal | 1/2/2 | failed | failed/member_failed, absent; helper rejected/invocation_limit | 1/1/0 |
| 4 helper R refusal | 2/1/2 | failed | failed/member_failed, absent; helper rejected/created_run_limit | 1/1/0 |
| 4 helper depth refusal | 2/2/1 | failed | failed/member_failed, absent; helper rejected/depth_limit | 1/1/0 |

Each refusal group crosses all four modes at id1/value0. Helper acceptance cases cover all four id/value pairs in every mode. No helper run or charge appears on refusal. The fixed root caller contributes one static run; admitted entry/helper each adds one created run. All effects limits are 0 and physical MaxInFlight is 1.

Operational bound derivation from the frozen finite artifacts: candidate 16 + helper 7 + qualification caller 7 = 30 total static instructions, with all executable edges forward except an unmatched receive that contributes no endless local commits. At most two invocation intents, two reply intents and two outcome deliveries appear, with no external work. Caps MaxSteps=200 and MaxDispatch=100 exceed these finite paths without manufacturing work; every observed run was quiescent below both caps (9–19 steps, 3–6 dispatches). All public byte limits remain DefaultLimits=1 MiB each; candidate/helper/caller artifacts are respectively 9,533/3,951/4,396 bytes, no collections, request strings at most 7 bytes and a finite set of scalar/handle registers. No observed cutoff is counted as a passing pending/terminal case.

Other unsuccessful commands: initial unprivileged `mkdir -p` of the two assigned directories was blocked by the managed-worktree sandbox, then scoped `require_escalated` succeeded. Read-only location probes attempted nonexistent `.superpowers/sdd/00-plan-phase1i/evidence/reviews/contract-rereview-v1.md`, `internal/executor/composition_completion.go`, and `internal/executor/composition_provision.go`; actual review/type paths were found with `rg`. The initial unused-import build failure and expected empty-template RED are disclosed above. No runtime qualification command failed after implementation.

## Remaining obligations and manuscript scope

The independent K1/local-relation/context-product graph construction, instruction-to-model correspondence review, 3,840-configuration products, counterexample controls, closure/reopen/effect companions and adversarial review remain with the controller/independent model author. The 28 author cases do not establish arbitrary-context preservation, exhaustive concrete-runtime conformance, compiler correctness, distributed behavior, scheduler confluence, or effectful recovery congruence.

No manuscript file changed in this narrowly assigned author slice. B uses only existing ordinary retained public constructs; it proposes no runtime semantic change. Any broader conformance/substitution claim or manuscript qualification must await the separate independent checks. The initial contract's conditional and bounded language remains essential. Candidate source is frozen; further controller-requested edits must be separately recorded rather than replacing this first-version evidence.
