# Independent Task2b review — templates, admission, replies, freezes and known settlement

Reviewed base `56d25e0ab4b614afdec791f8ff090852d65627bc` to head `a648ea739d21f04e011be3214f7819130a5f3f2a`, using the supplied exact diff. Read the Task2b brief first, the normative composition contract in full, and the implementation report as claims to verify. This is a scoped slice review, not final branch approval or Phase1G qualification.

## Spec verdict

**Needs fixes.** The intended gated Task2b architecture is present, and the principal admission/reply/freeze behaviors have targeted tests and source support. One reproduced lifecycle ordering violates inherited scope closure required by R1/R3. This belongs to Task2b's explicit admission/ancestor-closure boundary, not deferred Task2c work.

## Strengths

- Public authoring retains typed templates and closed dependency catalogues. Invoke remains an ordinary emit with pending identity and state updates; AwaitInvocation uses correlated ordinary receive with explicit status, reason and response presence. There is no scatter policy opcode, special worker role, or second executor.
- Accepted admission, all member origins/runs, binding, initial public request, ordered fact and invocation/run charges share one owner transaction. Original accepted/rejected decisions are reused. Tests cover missing uncommitted operations, denial, second-member insertion rollback, fresh identity, nested shared accounting and admission/closure orderings.
- Reply notification eligibility is narrowly selected by immutable caller binding. It does not grant adapter or internal-message authority after terminal state. Candidate deduplication is by original Send; separate Sends remain separate obligations. Failure freezes capture committed reply IDs before delivery callbacks, preserving Send(invalid_result) then Failure and preventing later replies from replacing the selected diagnostic response.
- Explicit absent responses avoid fabricated enum values and remove stale payload bindings. Known settlement is separately retained and correlated; pending admission, accepted descendant joins and external joins cannot silently become successful outcomes.
- Public schema4 construction/activation remains gated. The new protocol profile is explicitly separate from the older foundation profile, and the report accurately identifies runner, full joins, replay and resource enforcement as unfinished.

## Critical findings

None found in the reviewed scope.

## Important findings

### I1 — Retiring an already-succeeded member fails to close its still-active scope

Locations: `internal/executor/composition_facts.go:202` / `:208`, `internal/executor/invocation_outcome.go:228` / `:232`, and `internal/executor/invocation.go:172`.

`recordCompositionLifecycle` passes only the target run and fact ID into `freezeCompositionScope`. That function derives closure solely from the target's existing projection status. When RetireSubject succeeds against an already-succeeded member, the durable subject becomes retired but its prior local terminal remains succeeded. If a peer remains open, the success path returns without recording a scope freeze. `compositionScopeClosed` subsequently sees an open root and no non-success freeze, so an already committed/authorized peer Invoke is admitted after retirement.

The focused probe used an explicitly provisioned static helper in the resource root's membership. It committed helper success, committed/authorized a root Invoke, retired the helper with an authorized lifecycle operation, then admitted the root Invoke. Actual evidence: retirement accepted=true, reason=retired, retired=true, member_status=succeeded, peer_status=open; scope_closed=false; admission accepted=true and materialized two member runs. R1 explicitly includes statically provisioned members in scoped lifecycle rules, and R1/R3 require retirement to close the affected scope and prevent later descendant admission. The same status-only lifecycle handoff is used for dynamic members; this probe establishes the static-member defect without claiming a separately executed dynamic case.

Fix the scope lifecycle transition independently of rewriting an already-retained local run terminal. Carry the accepted lifecycle operation/result into the closure decision, retain its ordered closure evidence atomically, and enforce it at admission. Add this ordering as a regression. Preserve the earlier source authorization as historical evidence.

Controller ruling needed: the precise eventual public outcome status/reason when a locally succeeded member is later retired is not fully specified by the existing first-non-success-terminal wording. That ambiguity does not permit fresh descendant admission after actual subject retirement. This finding does not assert that accepted already_terminal CancelRun/AbandonRun no-ops must rewrite status or behave identically; those cases were not probed.

## Minor findings

None identified. The existing Task2a M1 reserved-table guard coverage gap remains deferred; the invocation reopen gate does not resolve it.

## Boundaries considered and declined to judge

- Runner selection, root/global physical capacity, external reservation/debit enforcement: explicitly Task2c, with public activation still gated.
- Complete accepted-descendant/external joins and root publication/inspection: explicitly Task2c; Task2b exposes pending reasons rather than claiming completion.
- Schema4 recovery and full forward/reverse cross-ledger replay: explicitly Task2c; unfinished protocol stores fail closed.
- Subprocess SIGKILL, saturation/progress monitors, provider behavior and Task3 collection policy: outside this slice. No qualification inferred from manually driven protocol tests.
- Already-terminal CancelRun/AbandonRun semantics and eventual outcome status after retirement: no unobserved behavior asserted; retirement closure defect is independently reproduced above.

## Verification and outside-diff inspection

Read source changes in focused passes: public author API/compiler; catalogue/version/binding; atomic admission; reply/freeze/preparation/settlement; integration hooks; corresponding targeted tests. Did not rerun reported suites. Inspected retained `19-command.json` and `19-default-suite.log`: reported full default suite exit 0, with passing package output. This review does not claim an independent rerun or independent reproduction of historical RED stages.

Outside changed hunks, inspected only named integration risks:

- `internal/executor/lifecycle.go:145-211` and `:307-345`: whether accepted retirement changes an already-succeeded projection and whether lifecycle records/hook share the transaction. It preserves local success and records retirement, explaining I1.
- `internal/executor/composition_binding.go:60-130`: verify the inherited retention path actually enforces required metadata, initial state, sole public boundary and compatible closed internal ports.
- `author/compile.go:590-620`: rule out duplicate output destinations causing nondeterministic response/envelope binding; existing validation rejects duplicate destinations.

No source/index/HEAD changes, git operations, credential access, live provider calls or subagents. One newly authored focused test was appended through a Go overlay stored entirely under `/tmp`. The exact supplemental input and original observed failure follow. Existing source tests and historical evidence were not overwritten.

## Quality assessment

**Needs fixes.** The implementation is appropriately gated and uses the existing owner transaction and ordinary execution machinery. Fix I1 before accepting Task2b's closure boundary; final full branch review remains separate.

## Exact focused probe evidence

The overlay replaces only the test-file view at build time. `invocation_test.go` is the unchanged base test bytes followed by the exact probe below. The command runs only the new test. The Python capture wrapper returned successfully; the actual Go subprocess exit is **1**, retained in command.json.

### Command

```json
{
  "cwd": "/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan",
  "argv": [
    "go",
    "test",
    "-overlay=/tmp/alan-task2b-review-retire/overlay.json",
    "./internal/executor",
    "-run",
    "^TestReviewRetiredSucceededMemberClosesScope$",
    "-count=1",
    "-v"
  ],
  "environment": {
    "GOTOOLCHAIN": "local",
    "GOPROXY": "off",
    "GOSUMDB": "off",
    "GOCACHE": "/tmp/alan-phase1f-go-build"
  },
  "exit_code": 1
}
```

### Input identities

```json
{
  "base_test": "/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan/internal/executor/invocation_test.go",
  "base_test_sha256": "870d26436ad92b5a9975dbf6092a9e014bea9480997e8c5351a60897b964edf0",
  "probe_sha256": "8205b0a9fbeb3b11bc20ba50a5c7c250f588222af8580d06f9954c2d9f947190",
  "overlay_sha256": "f41fa6001f4228e54ba8ab20aa6ee4e55981c10d54b9a28907d160e2bbd0b461"
}
```

### Exact supplemental test input

```go

// Review probe: RetireSubject succeeds after a static member's local success,
// while its root-scope peer still owns a committed, authorized Invoke.
func TestReviewRetiredSucceededMemberClosesScope(t *testing.T) {
 p, c := invocationFixture(t, "succeeded", 1)
 addProgram(t, &p, "static-helper", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
 c.Roots[0].Members = append(c.Roots[0].Members, RunID{"static-helper", 1})
 addInvocationController(t, &p, "RetireSubject")
 for i := range p.Runs {
  if p.Runs[i].ID.Subject != "controller" { continue }
  for j := range p.Programs {
   if p.Programs[j].ID != p.Runs[i].ProgramID { continue }
   var program Program
   if err := json.Unmarshal(p.Programs[j].Bytes, &program); err != nil { t.Fatal(err) }
   program.Code[0].Effect.Target = Lit(Text("static-helper"))
   raw, id, err := EncodeProgram(program)
   if err != nil { t.Fatal(err) }
   p.Programs[j] = Artifact{ID: id, Semantics: program.Semantics, Bytes: raw}
   p.Runs[i].ProgramID = id
   break
  }
 }
 for i := range p.Permissions {
  if p.Permissions[i].Source == "controller" { p.Permissions[i].Target = "static-helper" }
 }
 s, err := createInvocationStore(t.Context(), t.TempDir()+"/retire.sqlite", p, c, DefaultLimits())
 if err != nil { t.Fatal(err) }
 defer s.Close()
 if _, err := s.Commit(t.Context(), current(t, s, "static-helper").Authority(), ""); err != nil { t.Fatal(err) }
 root, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
 if err != nil { t.Fatal(err) }
 operation := root.Projection.Data["pending"].Text
 auth, err := s.AuthorizeIntent(t.Context(), operation)
 if err != nil || !auth.Allowed { t.Fatal(auth, err) }
 retirement := authorizedDecision(t, s, committedRequest(t, s, "controller"))
 if !retirement.Accepted || !retirement.After.Retired { t.Fatal(retirement) }
 t.Logf("retirement accepted=%v reason=%s retired=%v member_status=%s peer_status=%s", retirement.Accepted, retirement.Reason, retirement.After.Retired, retirement.After.Projection.Status, current(t,s,"root").Projection.Status)
 closed, err := compositionScopeClosed(t.Context(), s.conn, RunID{"root", 1})
 if err != nil { t.Fatal(err) }
 admission, err := s.AdmitInvocation(t.Context(), operation)
 if err != nil { t.Fatal(err) }
 t.Logf("scope_closed=%v admission_accepted=%v admission_reason=%s admitted_members=%d", closed, admission.Accepted, admission.Reason, len(admission.Members))
 if admission.Accepted { t.Fatal("retired member left its scope open to new child admission") }
}
```

### Original output

```text
=== RUN   TestReviewRetiredSucceededMemberClosesScope
    invocation_test.go:1026: retirement accepted=true reason=retired retired=true member_status=succeeded peer_status=open
    invocation_test.go:1031: scope_closed=false admission_accepted=true admission_reason=accepted admitted_members=2
    invocation_test.go:1032: retired member left its scope open to new child admission
--- FAIL: TestReviewRetiredSucceededMemberClosesScope (0.04s)
FAIL
FAIL	github.com/JakeFAU/alan/internal/executor	0.201s
FAIL
```
