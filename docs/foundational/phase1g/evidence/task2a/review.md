# Task2a independent scoped review

## Spec Compliance

**❌ Issues found — ordered-prefix integrity is incomplete.** The gated foundation otherwise matches the agreed Task2a slice, but accepts scoped message consumption before its source commit in retained global order (`internal/executor/composition_facts.go:289`). This is an existing execution/history dependency, not a request to implement Task2b/2c early.

**Task quality: Needs fixes.** One Important finding; no Critical findings. Exact review scope: `bd27abe05d6014694cc1102d345ca8b1762e1b13..23d6ee4fa1fa03fa05bcdae16f998ea73788e91d`. No Task1 re-review or complete invocation qualification is implied.

## Strengths and verified requirements

- **Existing transaction paths.** `store.go:273` and `store.go:497` share construction/reopen implementations; `store.go:770` and `lifecycle.go:328` put hooks inside the original execution/lifecycle transactions. `composition_facts.go:103` accepts the caller's transaction without owning a second commit engine. Rollback/dedup tests use actual stores and injected transaction failures (`composition_store_test.go:120`, `composition_facts_test.go:11`). No scheduling loop was added.
- **Version isolation.** Public Create selects schema3 with unchanged DDL. Open validates before epoch advancement; Inspect and Replay require schema3 (`store.go:273`, `store.go:530`, `inspect.go:57`, `replay.go:79`). The real-schema4 gate/reopen test checks activation behavior (`composition_store_test.go:144`). Private `alan/composition-foundation/v1` is explicit (`composition_types.go:13`); future public schema4 must use the controller's distinct complete profile version rather than promote foundation/v1.
- **Immutable data and binding.** Encoded profile storage prevents caller mutation; root IDs include namespace and declared root run; static members must be provisioned generation1 origins and include the root (`composition_store.go:46`, `composition_store.go:88`, `composition_types.go:114`). Encoding checks precede JSON copying (`composition_types.go:136`). Binding maps are exact; resolution copies original encoded artifacts and changes only declared endpoint operands, preserving payload literals (`composition_binding.go:186`, `composition_binding.go:247`, `composition_binding.go:293`; reduction/byte-identity test at `composition_binding_test.go:55`). Authentication of an actual admitted invocation remains explicitly deferred.
- **Bidirectional reconciliation.** Root/member/template descriptors reconcile against all table rows; scoped histories/lifecycle decisions require facts and every fact requires its source row (`composition_store.go:256`, `composition_facts.go:224`, `composition_facts.go:272`). Lifecycle sequence, source-intent precedence, target prefix and closure ordering are checked (`composition_facts.go:294`). Scoped Retry is transactionally gated pending its later correlated denial policy (`composition_facts.go:176`, `composition_facts_test.go:168`).
- **Honest test evidence.** Final suite log 16 reports 15 tested packages passing and two with no tests; log 17 reports the later test-only witness assertions passing. RED records remain visible, including log 03's FK-blocked injection setup failure. Final successful logs have no unexplained warnings. Missing/invented facts, static membership mismatch and reversed same-run order reach their intended validators in log 11.

## Important finding

### I1 — Execution-prefix validation permits consuming a future scoped message

**Location:** `internal/executor/composition_facts.go:289`; existing ordinary reconstruction at `internal/executor/replay.go:173`.

The execution branch checks only the run's next sequence and lifecycle closure. Its digest includes the selected observation, but validation does not resolve that observation's source and require the corresponding source fact in the already-validated global prefix. Ordinary ledger replay iterates runs independently against complete observation/intent maps, so it cannot provide that missing order constraint.

**Reproduction:** A scoped sender commits one message, normal `DeliverMessage` admits it, and a scoped receiver consumes it through normal `Commit`. Baseline foundation validation passes with facts `[sender/1, receiver/1]`. The probe swaps only `decision_order` and each descriptor's `Order`, leaving source digests, fact IDs, history, admissions, observations and projections intact. Foundation validation still passes with `[receiver/1, sender/1]`. The focused test fails at the expected corruption rejection after successful baseline validation. Source, overlay, exact command and output are retained below.

The accepted prefix is impossible under actual transaction causality. Later freeze reconstruction could trust this impossible ordering despite correct individual source-row links and final histories. Same-run and lifecycle checks therefore do not yet establish the requested foundation's ordered-prefix integrity.

**Focused fix:** Resolve selected observations through their immutable admission/source evidence and enforce predecessors that this foundation already represents:

- `ordinary-message`: the scoped source intent's execution commit must precede consumption.
- `runtime-terminal`: the source's terminal execution commit, or the lifecycle closure transaction recorded by the terminal obligation, must precede consumption. Do not assume terminal sequence alone identifies an execution fact when lifecycle closure produced it.
- `runtime-lifecycle`: the retained lifecycle-result decision must precede consumption when represented by a composition lifecycle fact; source request execution alone is insufficient.

These relationships are available from the existing admission descriptors and source ledgers (`messages.go:205`, `messages.go:240`, `messages.go:358`, `lifecycle.go:353`). The execution branch presently performs none of these selected-observation checks. The reproduced failure is specifically ordinary-message causality; terminal/lifecycle variants have not been independently executed in this review. Add narrow nonvacuous corruption tests using valid baseline records, preserving all links/digests and changing only order. No new scheduler, protocol transaction, or general admission ledger is needed for this fix. Adapter/receipt attempt/result predecessors and future invocation/reply-result facts remain deferred to Task2b/2c; local inputs and unscoped source transactions have no source fact in this foundation.

## Minor finding

### M1 — Reserved-table test reaches FK rejection, not the reserved-profile guard

**Location:** `internal/executor/composition_facts_test.go:148`; actual error in `docs/foundational/phase1g/evidence/task2a/11-prefix-corruption-readonly.log:13`.

The `future binding table` case references a missing invocation, so ordinary FK validation rejects it before the explicit nonempty-table guard (`composition_store.go:342`). The report accurately discloses this; it is not a false GREEN or missing production check. A fixture retaining valid relational references would directly test the intended `ErrVersion` guard. This coverage improvement is not an additional blocker.

## Independent X/Y witness assessment

The original standalone argument in `evidence/task2/report.md` is valid for its stated abstraction: per-member histories omit cross-member commit order. Task2a executes X and Y using real `Store.Commit`; the test compares history grouped by run, immutable Intent values, Runs and terminal obligations (`composition_store_test.go:38`). Grouping removes incidental cross-run row enumeration while retaining every row field. This proves equality of these ordinary logical structures, not entire database files or store metadata.

I independently parsed `executed-counterexample.json`, compared all four structures exactly, and derived candidate IDs from each fact's matching history row and actual message intents until the failed successor. X gives `[A/1/1/0]`; Y gives `[A/1/1/0, A/1/2/0]`. Replacing RootIDs with one symbolic root still leaves the ordered facts different. Fact IDs/source digests are equal as sets, so different random namespaces do not manufacture the distinction.

`Frozen` is monitor output, not a retained runtime freeze/result. The witness does not establish reply authentication, invocation outcomes, structured completion, resource conservation, runner progress or arbitrary interleaving safety. X's later A.Send2 is ordinary execution after another static member fails, not evidence of an admissible late invocation reply. The report and `counterexample-input.json:44` state this limit accurately. I1 concerns another ordering constraint and does not invalidate this witness.

## Review boundaries and checks

- Read the brief, implementation report, exact supplied diff, normative R1/R2/R4/R6 contract and original Task2 witness. Reviewed the change in focused passes; no whole changed source file was separately reread.
- Named integration risk: whether original metadata validation already imposed stronger endpoint constraints. Checked `author_metadata.go:42` through `author_metadata.go:153` and relevant `program.go:153` cases; no extra finding is asserted from this inspection.
- Named integration risk: scoped cross-run observation causality. Checked the ordinary replay loop cut off in the diff (`replay.go:145`), existing fixture/API and message delivery/permission/observation mapping (`messages_test.go:12`, `messages.go:170`, `messages.go:240`, `messages.go:358`, `messages.go:467`, `lifecycle.go:353`). This confirmed the probe uses unchanged production message paths and identified the bounded source classes above.
- Ran one focused Go overlay test and one independent retained-JSON witness calculation. No broad suite/race rerun, checkout source modification, provider calls, credentials, git operations, or child agents. Only this review report is added; existing/historical evidence is unchanged.
- **⚠️ Outside this gate:** dynamic origins, public schema4 replay, invocation admission/results, scope closure, effect conservation and physical-capacity semantics remain Task2b/2c work (`composition_store.go:342`, `composition_types.go:72`). Their absence is explicitly gated and is not a finding here. Later complete-profile compatibility and qualification remain required.

## Focused probe evidence

Command (exit 1 at expected rejection assertion):

```sh
GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build GOTOOLCHAIN=local go test -overlay=/tmp/alan-task2a-review-probe/overlay.json ./internal/executor -run '^TestReviewCompositionMessageCausalPrefix$' -count=1 -v
```

The overlay adds one test from `/tmp` without changing checkout source. Original order was sender/1 then receiver/1; the mutated order was receiver/1 then sender/1. Expected validation error; observed nil error.

### Overlay

```json
{"Replace": {"/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan/internal/executor/review_causal_test.go": "/tmp/alan-task2a-review-probe/causal_test.go"}}
```

### Probe source

```go
package executor

import (
 "encoding/json"
 "testing"
)

func TestReviewCompositionMessageCausalPrefix(t *testing.T) {
 p := Provision{}
 addProgram(t, &p, "sender", []Instruction{{Op:"emit",Effect:&Effect{Kind:"message",Target:Lit(Text("receiver")),Generation:Lit(Int(1)),Obligation:Lit(Text("work")),Payload:map[string]Operand{"number":Lit(Int(17))}}},{Op:"end",Outcome:"succeeded"}}, map[string]Value{})
 addProgram(t, &p, "receiver", []Instruction{{Op:"receive",Match:&Match{Kind:"message",Provenance:"ordinary-message",Source:Lit(Text("sender")),Generation:Lit(Int(1)),Obligation:Lit(Text("work")),Operation:Lit(Text(""))},Bind:map[string]string{"number":"number"}},{Op:"end",Outcome:"succeeded"}}, map[string]Value{})
 p.Permissions=[]Permission{{Kind:"message",Source:"sender",Target:"receiver",Operation:"message",Revision:1}}
 c:=compositionProvision{Roots:[]compositionRoot{{Run:RunID{"sender",1},Members:[]RunID{{"sender",1},{"receiver",1}},Limits:compositionLimits{MaxInFlight:1}}}}
 s,err:=createCompositionStore(t.Context(),t.TempDir()+"/causal.sqlite",p,c,DefaultLimits());if err!=nil{t.Fatal(err)};defer s.Close()
 if _,err=s.Commit(t.Context(),current(t,s,"sender").Authority(),"");err!=nil{t.Fatal(err)}
 admitted,err:=s.DeliverMessage(t.Context(),identity("sender",int64(1),int64(1),0),"review-delivery");if err!=nil||!admitted.Accepted{t.Fatal(admitted,err)}
 if _,err=s.Commit(t.Context(),current(t,s,"receiver").Authority(),admitted.ObservationID);err!=nil{t.Fatal(err)}
 if _,err=validateCompositionStore(t.Context(),s.conn,DefaultLimits());err!=nil{t.Fatalf("baseline invalid: %v",err)}
 facts,err:=readCompositionFacts(t.Context(),s.conn);if err!=nil{t.Fatal(err)};if len(facts)!=2{t.Fatal(facts)}
 if _,err=s.conn.ExecContext(t.Context(),"DELETE FROM composition_facts");err!=nil{t.Fatal(err)}
 for i:=range facts { f:=facts[1-i];f.Order=int64(i+1);raw,err:=json.Marshal(f);if err!=nil{t.Fatal(err)};if _,err=s.conn.ExecContext(t.Context(),"INSERT INTO composition_facts VALUES(?,?,?,?)",f.Order,f.ID,f.Kind,raw);err!=nil{t.Fatal(err)} }
 if _,err=validateCompositionStore(t.Context(),s.conn,DefaultLimits());err==nil{t.Fatal("validator accepted receiver consumption ordered before its source message commit")}
}

```

### Probe output

```text
=== RUN   TestReviewCompositionMessageCausalPrefix
    review_causal_test.go:22: validator accepted receiver consumption ordered before its source message commit
--- FAIL: TestReviewCompositionMessageCausalPrefix (0.01s)
FAIL
FAIL	github.com/JakeFAU/alan/internal/executor	0.303s
FAIL

```

### Independent witness source

```python
import json
from pathlib import Path
p=Path('docs/foundational/phase1g/evidence/task2a/executed-counterexample.json')
a,b=json.loads(p.read_text())
for k in ('History','Intents','Runs','Terminals'):
    assert a[k]==b[k], k
print('Exact retained ordinary equality: History, Intents, Runs, Terminals')
for trace in (a,b):
    frozen=[]
    for f in trace['Facts']:
        rows=trace['History'][f['Run']['Subject']]
        row=next(r for r in rows if r['sequence']==f['Sequence'])
        if json.loads(row['successor'])['Status']=='failed':
            break
        frozen.extend(i['ID'] for i in json.loads(row['intents']) if i['Kind']=='message' and i['Target']=='caller' and i['Obligation']=='reply')
    assert frozen==trace['Frozen']
    print('Order:',[(f['Run']['Subject'],f['Sequence']) for f in trace['Facts']], 'derived committed reply prefix:',frozen)
def normalized(t):
    return [{**f,'RootIDs':['root']} for f in t['Facts']]
assert normalized(a)!=normalized(b)
assert sorted((f['ID'],f['RecordDigest']) for f in a['Facts'])==sorted((f['ID'],f['RecordDigest']) for f in b['Facts'])
print('Namespace-independent difference confirmed; fact identities and source digests equal as sets.')

```

### Independent witness output

```text
Exact retained ordinary equality: History, Intents, Runs, Terminals
Order: [('A', 1), ('B', 1), ('A', 2)] derived committed reply prefix: ['["A",1,1,0]']
Order: [('A', 1), ('A', 2), ('B', 1)] derived committed reply prefix: ['["A",1,1,0]', '["A",1,2,0]']
Namespace-independent difference confirmed; fact identities and source digests equal as sets.

```
