# Acceptance evidence map

This map tracks the user contract, not a claim that pending checks have passed. Final qualification must name exact tests and retained artifacts. Successful reachability, searched safety, crash behavior and live provider access must remain distinct.

| Gate | Required observation | Evidence status |
| --- | --- | --- |
| A: authoring | External module uses named steps, typed input/output, branches, effect waits and component contracts; deterministic artifact; missing/type-invalid references rejected; same retained artifact reopens without rebuilding | Task 2 PASS after independent R1/R2 fixes; final full-source qualification pending |
| B: branching and nesting | Real serializer/parser against controlled HTTP returns lookup, clarification/insufficient evidence, refusal/incomplete/invalid cases; ordinary helper contract hides internals | Tasks 3–4 pending |
| C: recorded model result | Acknowledged post-record/pre-consumption SIGKILL; independent provider contact ledger unchanged for that operation after reopen; pending identity and logical budget survive | Task 5 pending |
| D: recorded tool result | Acknowledged post-record/pre-synthesis SIGKILL; changed/deleted original corpus cannot affect retained evidence; no lookup reattempt | Task 5 pending |
| E: unknown model | Provider records acceptance and withholds response; controller kills executor; reopen retains Unknown and performs no repeat | Task 5 pending; repeat policy deliberately unsupported |
| F: offline replay | Successful run replays with no credential/provider/source path; zero adapter invocations and identical public result/projection | Task 5 pending |
| G: namespace/authority | Two stores, same local IDs, different keys; same-store reopen stable; wrong capability/parameters/correlation/endpoint rejected | Namespace PASS in task1 evidence; adapter authority pending |
| H: independent progress | Held model, ordinary peer and caller advance before release; shared mixed-transport capacity; cancellation/closure preserves authenticated late evidence and known-unsettled classification | Historical Phase 1D.1 baseline PASS; new adapters pending |
| I: no hidden policy | Provider response recorded, ordinarily consumed/validated, then separate lookup intent committed/authorized; unchanged monitor catches a deliberate bypass | Task 5 pending |
| Live | Explicit configured model and runtime credential, bounded bundled corpus, actual response/usage/result, same-record offline replay | NOT RUN: no credential/model configured |

The namespace regression retains original pre-fix source and two-store databases. Later genuine defects must likewise retain their original input before fixing. A monitor belongs to qualification; it must never select a runtime branch or repair state.
