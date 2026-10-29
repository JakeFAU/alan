# Phase 1J independent context implementation review v1

**Overall: REJECT. Specification: APPROVE. Code quality: REJECT.**

This review covers the ordinary context builder, its focused tests, separate public consumer, and frozen context evidence. It does not approve the independent model, raw runtime adapter, process-crash qualification, manuscript reconciliation, or an all-schedule completion claim. One reproduced test defect blocks approval. The context implementation itself conforms to J1; the fix belongs in its test/evidence, not in runtime or authoring semantics.

## Question and reviewed identity

Does this implementation express frozen J1 through the permitted ordinary-program intervention, preserve the C0 invocation paths and static accounting, and avoid inventing outcomes when a branch is failed or pending?

Worktree: `/Users/jacob/.codex/worktrees/phase1j-retained-joins/alan`. Independently read HEAD: `d68ec667d276a4bd9e1be11613905c628cfea704`. Tracked `git diff --name-only` was empty; Phase1J additions were untracked, including independent workstreams outside this review. I reconstructed all eight added files from the supplied review diff and compared their complete text to current files: all equal.

| Reviewed input | SHA256 |
|---|---|
| `assignment.md` | `70db30c2377b45cc08f110ac3836f09cae9ba3604e593e2b51198a292883648c` |
| `00-plan.md` | `f773775b5ac65a21d63180eec7862c703590f6ea6c279d6669c9d58be9f506dd` |
| `01-join-contract.md` | `a950450d33ab928ccb875f8c98de83a83c956d8759fa0f39a71afe1c06c52a81` |
| `.superpowers/sdd/00-plan/context-review.diff` | `0617305cd409caaddfdebd36be6d314a9fda5bc48922ec1b9dc0f64f2b5cb8a2` |
| `examples/join/deployment.go` | `0bcbff9113b9e11b148f076e839b5279b28c833e77fbe0069fab14902ce3249e` |
| `evidence/context/implementation-report.md` | `75da7ed0bdf8343fc9ddd4f2e157f3712c93bf630d508b9eb1d1a8b12c3da3d7` |
| `evidence/context/source-sha256.json` | `dfe1ad81037ceaebdd975e66052f8db42dab0b66b5cb4e0fc85c7c9581f63e17` |
| `evidence/context/evidence-sha256.json` | `71a777cb26effb601605abf243d5cf0e7e726cb6b1a4a50388458b19c30841db` |
| `evidence/context/artifacts/sha256.json` | `6cc46a9329d59a5b800704fd58f5c6f0d95847885502c80335bbc744a7d179ad` |

Document paths above are relative to `docs/foundational/phase1j/` unless otherwise specified. The three manifests independently validated all 13 source, 64 context-evidence and 45 artifact entries, respectively; zero mismatches against retained files. They pin the remaining exact reviewed source and artifact identities without duplicating the complete manifest here.

Read applicable AGENTS.md, assignment, frozen contract, context plan/interface, protocol review, all eight context/consumer files and full supplied diff, implementation report and selected recorded logs, exact source excerpts and patches, original substitution deployment/probes, public provision/compiler correspondence, ordinary-message source authorization, and the relevant Phase1I manuscript qualification. No source changes, commits, TeX builds or subagents were performed.

## Blocking finding

**[P2] Compare provision programs by identity, not catalogue iteration order — `examples/join/deployment_test.go:110–111`.** `TestJoinConfigurationControls` builds the empty-shape/explicit-answer and normal primary deployments independently, then calls `reflect.DeepEqual(d, primary)`. Their public `Provision.Programs` catalogue can have a different slice order because existing `author.Application.Provision` gathers catalogues while ranging over an application map. Identical artifacts therefore spuriously fail the new test. After observing the order-only re-export difference below, I ran the single configuration test 50 times: it failed 9 times with `empty shape / explicit answer changed artifacts` and the command exited 1. This is reproducible validation instability, not a semantic change in the context.

Required correction: compare the programs keyed by ID with exact Bytes/Semantics equality, or sort copied Programs slices by ID before comparing. Preserve the test's checks of every other deployment field and exact artifact identities; do not remove the assertion or normalize instruction order. Do not repair existing runtime/author code for this issue. Repeat this focused test enough times to exercise catalogue ordering, refresh affected source/evidence hashes and review diff, and request re-review. The raw provision snapshots can remain historical exact bytes with the reproduction caveat below.

## Specification and code findings

- **Permitted intervention preserved.** `New` obtains original S from `substitution.New`, keeps candidate/entry identities and two static generation-1 members, and adds only the root/peer programs and ordinary join grant. Public `author.Application.Provision` and `runtime.ResourceRoot` declarations supply wiring and accounting. No internal import or runtime/compiler/schema/dependency modification occurs in this workstream. The separate consumer has its own module with the existing pinned SQLite dependency and local module replacement.
- **Causal comparison checked independently for A and B.** Re-exported candidate/S bytes equal their C0 counterparts; primary root instructions 0–1 and peer instructions 0–33 equal C0 exactly. Both resource-root declarations are equal, including limits, members and allowed template IDs. All four pretty-printed/sorted-key compiled patches reproduce exactly. Source excerpts match the actual functions, and their source patch reproduces exactly. New root IDs are `d7fa07a0913dc035c5a1e6b47a1d663064184f4818bb1e38e91d9d3ca157e216` and `4c5ea75d4e08467042f3f25491ac341a5e0347652207d9970ffc6be3b4220292`; shared primary peer ID is `6a77bf8322f8bf227e7c78597d6e753d8d9390f3c981fec7f2bdc59814ea71ef`.
- **Strict join is implemented.** Root consumes the original candidate outcome before mandatory Receive and makes no normal terminal decision before that receive. Its nested checks evaluate status and presence before response bindings, then candidate id/complement and S id/zero. Peer consumes the original S handle before a single report, copies status/reason/presence/handle, uses literal zero fillers on absence, and locally succeeds after reporting even on known S failure. There is no subsequent scope-dependent admission in the conforming suffix. Pending branches have no fabricated report or terminal transition.
- **Correlation claims are appropriately limited.** Ordinary source/generation/obligation matching is compiled by existing public APIs; root checks application request id. Wrong request parks at an additional Receive under the one-report profile. Root retains but does not authenticate the S operation string; premature report is an intentionally separate invalid-peer control. The additional receive is not a general extra-message/retry protocol and is not claimed as one.
- **Tests are meaningful within their stated scope.** Structural comparisons protect actual compiled prefixes and resource roots. Runtime tests preserve C0 failure, check successful A/B charges, failure/absence/pending behavior, wrong correlation, and the intentionally trusting premature mutant. Separate consumer exercises public init/open/run/inspect/replay and original-store recovery paths. Their successful replay does not independently prove compiler semantics, and completed-store reopen is not a process-crash witness. Those limitations are explicitly assigned to later workstreams.
- **Known source-authorization negative remains.** `Send` is followed by peer Success without acknowledgement. Existing `messages.go` can reject first authorization after peer terminal with `source_closed`; this review neither repairs nor excludes that schedule. Passing sampled executions therefore do not establish all-schedule J1 completion.

## Re-export caveat: raw provision catalogue ordering

The exporter at `examples/join/artifacts_test.go:48` serializes the public provision directly. Existing `author/application.go` ranges over the application map while adding invocation catalogue programs. My independent re-export changed only `Programs` order in three provision files. All 36 separately exported program/template files were byte-identical. All nine provisions were identical after sorting **only** the `Programs` array by `ID`; no instruction arrays, run arrays, resource declarations, permissions, bytes or other fields were normalized.

| File | Frozen SHA256 | Fresh SHA256 |
|---|---|---|
| `c0-a-provision.json` | `36aa4d5b8cb86d7be3af0f84ee0b1ebff455a52a4642a5fa9a33e25fa737d238` | `ac046b5514eeed541d5dc17b9807c523968f67cb275afd33da8f2e03a31d5f61` |
| `premature-provision.json` | `9624a94357a6a68ece6b81e033dc691e322a013078372bd8b156da429e8abf72` | `7e8d0a89c9e578ad61bc508adfaf13405be21caccb7771ba3a11a8929ee7e05f` |
| `wrong-request-provision.json` | `d2587630c73468adc0ad05f81b7af5c089df2cca2e02e134311f820ae048e92d` | `e19e0763ea5cb859d9414ccc19b2b0c5be11c43824ec55dfe589adeeb56f49ae` |
| `sha256.json` | `6cc46a9329d59a5b800704fd58f5c6f0d95847885502c80335bbc744a7d179ad` | `0b81c6773bd85bb6dc301f1c88522433b6723d239e6b6cf9d3ef4c31db68edae` |

The initial exact-byte comparison deliberately failed on these four files; investigation then established the restricted order-only equivalence above. The frozen manifest remains accurate for the archived bytes. The implementation report does not promise deterministic provision serialization. Action for final runbook: re-export into a fresh directory; compare program/template bytes and IDs exactly and provision fields with Programs keyed by ID; do not require fresh raw provision hashes to match or overwrite the historical snapshots. This is a reproducibility qualification, not a semantic defect or reason to change existing runtime/author code.

## Commands and new evidence actually executed

Go commands used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1j-go-build` from the worktree root:

```sh
ALAN_JOIN_EXPORT=/tmp/alan-phase1j-context-review-exports go test -count=1 -run '^TestJoinArtifactExport$' ./examples/join
go run /tmp/alan-phase1j-context-review-probe.go
```

Both commands above exited zero. The first explicitly selected and executed the exporter. The second was a reviewer-authored public-API probe retained verbatim below (SHA256 `805ee89d5575ca655ee1a29e40e172f22a72c5ad73eb3a177200a2bec8c9e566`). It filled a focused test gap without modifying implementation files:

```text
wrong-id: root=failed peer=succeeded S=succeeded report=succeeded replay=passed
weak-a: root=failed peer=succeeded S=succeeded report=succeeded replay=passed
pending-with-s-failure: root=open peer=succeeded S=failed report= replay=passed
```

Both malformed successful candidate responses fail after joining S. A pending candidate remains pending even though S failure has been consumed by the locally successful peer. These are concrete samples, not exhaustive schedules. Temporary stores were retained at `/var/folders/qn/d3_l9fv5585g36qycsc804hh0000gn/T/alan-context-review-probe-2648087999` during this review.

The new ordering concern then justified this targeted repetition rather than rerunning the whole reported gate suite:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1j-go-build go test -count=50 -run '^TestJoinConfigurationControls$' ./examples/join
```

Exit status: **1**. Output contained nine repetitions of:

```text
--- FAIL: TestJoinConfigurationControls (0.00s)
    deployment_test.go:111: empty shape / explicit answer changed artifacts
```

This result supersedes the provisional approval considered before the targeted repetition; the review's final verdict is REJECT pending correction and re-review.

Additional executed commands: read-only `cat`, `sed`, `rg`, `nl`, `git status --short`, `git diff --name-only`, `git rev-parse HEAD`, and inline `python3` checks using hashlib/json/difflib. Python checked all manifest digests, reconstructed all eight new files from the diff, compared exact source excerpts/patches, compared root/peer prefixes, re-created all four compiled patches, checked unchanged A/B/S bytes and root declarations, and diagnosed the re-export ordering difference. The restricted provision comparison was:

```python
for p in sorted((base / 'artifacts').glob('*-provision.json')):
    a, b = [json.loads(d.read_text()) for d in [p, fresh / p.name]]
    a['Programs'].sort(key=lambda x: x['ID'])
    b['Programs'].sort(key=lambda x: x['ID'])
    assert a == b, p.name
```

I inspected implementer normal/race/vet and historical-negative evidence through the report/manifests and read selected normal logs; I did not rerun those entire gates or claim them as reviewer-executed checks. No claim of model completeness, runtime causal adapter validity, SIGKILL recovery, resource refusal matrix, stale/duplicate transport handling, manuscript consistency, or universal receipt authenticity is made here. Those remain explicit final-review obligations.

## Exact reviewer probe source

Save the following bytes as `/tmp/alan-phase1j-context-review-probe.go` to repeat the targeted command above. No repository source is edited.

```go
package main
import (
 "context"
 "fmt"
 "os"
 "path/filepath"
 "github.com/JakeFAU/alan/author"
 join "github.com/JakeFAU/alan/examples/join"
 "github.com/JakeFAU/alan/examples/substitution"
 "github.com/JakeFAU/alan/examples/substitution/a"
 "github.com/JakeFAU/alan/runtime"
)
func must(err error) { if err != nil { panic(err) } }
func main() {
 ctx:=context.Background()
 dir,err:=os.MkdirTemp("", "alan-context-review-probe-"); must(err)
 for _,name:=range []string{"wrong-id","weak-a","pending-with-s-failure"} {
  var c author.Template
  mode,want:="answer","failed"
  opt:=join.Options{}
  if name=="pending-with-s-failure" { c,err=a.New(); mode="pending"; want="open"; opt.SMode="fail" } else { c,err=substitution.Probe(name) }; must(err)
  d,err:=join.New(c,substitution.Config{Shape:"parallel",Request:substitution.Request{ID:1,Value:0,Mode:mode},Limits:runtime.ResourceLimits{MaxInvocations:3,MaxCreatedRuns:3,MaxDepth:3,MaxInFlight:1}},opt);must(err)
  path:=filepath.Join(dir,name+".db")
  s,err:=runtime.Create(ctx,path,d.Provision,runtime.DefaultLimits());must(err)
  _,err=s.Run(ctx,runtime.RunOptions{MaxSteps:200,MaxDispatch:200,MaxInFlight:1});must(err)
  root,err:=s.Snapshot(ctx,runtime.RunID{Subject:"root",Generation:1});must(err)
  peer,err:=s.Snapshot(ctx,runtime.RunID{Subject:"peer",Generation:1});must(err)
  if root.Projection.Status!=want || peer.Projection.Status!="succeeded" { panic(fmt.Sprintf("%s: root=%s peer=%s",name,root.Projection.Status,peer.Projection.Status)) }
  if want=="failed" && root.Projection.Data["join_status"].Text!="succeeded" { panic("failure without joined S") }
  if want=="open" && peer.Projection.Data["peer_status"].Text!="failed" { panic("S failure not consumed") }
  must(s.Close())
  _,err=runtime.Replay(ctx,path,runtime.DefaultLimits());must(err)
  fmt.Printf("%s: root=%s peer=%s S=%s report=%s replay=passed\n",name,root.Projection.Status,peer.Projection.Status,peer.Projection.Data["peer_status"].Text,root.Projection.Data["join_status"].Text)
 }
 fmt.Println("retained probe stores:",dir)
}
```
