# Task3 independent scoped review

Spec compliance: **Issues found — needs fixes.** Code quality: **Needs fixes.** One Important input-boundary defect; no Critical findings. The ordinary application construction and retained runtime use otherwise satisfy the scoped Task3 requirements reviewed here.

Reviewed base `347aea866550ceaf21de5e04452b6cf0dcb6e2dd` through head `6a5d9ea182e760c61149feaff47fe8dc0ff72f93`, against the task brief, normative contract R1–R8, and frozen acceptance plan. This is the Task3 gate, not Task4 process qualification or whole-branch/runtime re-review.

## Strengths

- `compose/collection.go:83` scatters Invoke plus retained handles before the separate ordered collect loop at `compose/collection.go:86`; identity is the explicit `item_id`, and original handles feed Await. `compose/collection.go:89` guards payload use with presence and retains zero-or-one typed responses, including empty record responses. No specialized scheduler/opcode or private child state implements policy.
- `examples/worktree/worktree.go:67` builds a fixed catalogue and one intake root; `examples/worktree/worktree.go:113` wraps the same collection port ordinarily. Public identity evidence (`docs/foundational/phase1g/evidence/task3/artifact-identities.json:1`) reports one static run for each profile, equal direct/nested collection IDs and entry/collection schemas, and constant catalogues (deterministic 2/3 templates, 3/4 programs; evidence 2/3 templates, 4/5 programs).
- `examples/worktree/consumer/contract_test.go:17` freezes literal expected integers. `examples/worktree/consumer/contract_test.go:267` checks ID/value/order and distinct operations; the faulty ordinary collector at `examples/worktree/consumer/contract_test.go:289` reaches valid runtime completion and replay but fails that same oracle. This is meaningful independent application validation.
- `examples/worktree/effects_integration_test.go:176` uses causal held-transport/admission/result acknowledgments for capacities 1/2 and direct/nested profiles; `examples/worktree/effects_integration_test.go:451` joins actual collector Receive acceptance to exact input. `examples/worktree/effects_integration_test.go:337` retains an unknown original operation and charged effect across reopen while an independent sibling completes.
- `author/invocations.go:152` exposes checked public schemas from fresh decoding. The compiler refinement at `author/compile.go:435` and `author/compile.go:601` is limited to string serialization and dynamic invocation handles; `author/retained_handles_runtime_test.go:15` tests an authentic unrelated recorded outcome without consuming it. No executor/runtime semantics changed in this task.
- `cmd/alan-compose/main.go:149` recovers by Open/Run without factory reconstruction, and `cmd/alan-compose/main.go:200` uses Replay without adapters. `examples/worktree/worktree.go:52` reuses the unchanged evidence factory; the diff does not alter original prompts, corpus, evaluation, earlier evidence, runtime scheduler or provider catalogue.

## Important finding R1: case-folded JSON fields silently replace identity and payload

Location: `cmd/alan-compose/main.go:281` (exact-name duplicate checking), interacting with `cmd/alan-compose/main.go:225` and `cmd/alan-compose/main.go:231` (struct decoding). `checkItemKeys` treats `item_id` and `ITEM_ID` as distinct keys, but `encoding/json` matches both to the same tagged struct field. `DisallowUnknownFields` also accepts case-only aliases. Consequently:

- `[{"item_id":"original","ITEM_ID":"replacement","number":2}]` becomes item ID `replacement` without an error.
- `[{"item_id":"item-a","number":2,"NUMBER":100}]` becomes number `100` without an error.
- `[{"ITEM_ID":"case-only","NUMBER":2}]` is accepted and renamed into canonical declared fields.

This bypasses the new duplicate/extra-field boundary, silently changes the user-supplied work identity or payload before durable admission, and violates the closed-record rejection requirement in `docs/foundational/phase1g/01-composition-contract.md:109` (R6). Existing exact-duplicate coverage at `cmd/alan-compose/main_test.go:91` cannot detect it. Reject noncanonical field names during the token pass (allow exactly `item_id`, `number`, `question`) and preserve exact duplicate checking, or decode exact-key raw maps with explicit validation. Add case-only and canonical-plus-alias regression cases for identity and payload. Keep these failing inputs and outputs unchanged as original evidence.

## Evidence and review scope

The source companion diff was read once in bounded passes. The full exact package was inventoried and its supplemental evidence inspected; repeated historical/source manifests were summarized rather than reread as application code. Final manifest `735c85c420c5cac56729afa8df1ae45252ab8ffea422e4673ed8042870085750` has 174 files, all matching checkout hashes. Supplied final command records 25–29 and 31–32 show exit 0; command 30 retains its actual exit 1 after the 90-second race watchdog. The unchanged-source 240-second rerun completed in 95.007 seconds (`docs/foundational/phase1g/evidence/task3/31-consumer-race-expanded-watchdog.log:1`). This is documented instrumentation overhead, not an erased failure or a claimed liveness proof. No broad tests were rerun by this reviewer.

Named focused checks outside diff context:

1. **Controlled-endpoint redirect escape risk:** inspected existing `adapters/openai/openai.go:27` endpoint validation and `adapters/openai/openai.go:168` HTTP client setup. It rejects user/query/fragment endpoint changes, uses a transport without environment proxy configuration, and returns `http.ErrUseLastResponse` for redirects. No escape found; no network probe was needed.
2. **Schema-copy alias risk:** inspected existing `author/invocations.go:21` `bundle`, whose fresh `json.Unmarshal` establishes deep separation for the new accessor, including nested maps/slices. Zero raw template returns an error.
3. **CLI semantic-key mismatch:** only the focused overlay tests below were run; they call the unchanged `readItems` and write temporary input files. No source/index/branch mutation, live requests, credentials, process qualification, or subagents.

Cannot verify from this task diff: runtime-wide closure/provenance/replay invariants and final whole-tree compatibility gates; they retain their Tasks1/2 review status. Actual SIGKILL/fresh-process boundary schedules, denied-network replay and final archive/full-tree race/vet remain Task4, as explicitly stated at `examples/worktree/README.md:78` and the task report. No planned Task4 evidence is counted as completed Task3 work.

## Exact original counterexample and focused probe reproduction

Source head: `6a5d9ea182e760c61149feaff47fe8dc0ff72f93`. Unmodified `cmd/alan-compose/main.go` SHA256: `18e6fb9a49cfe79bd234af3bb6d91db0ed8457db45a83e6cd56531a515c4c49b`. All probes use cwd `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`, and environment `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build`. Go overlay maps that checkout's `cmd/alan-compose/main_test.go` to `/tmp/alan-task3-review-casefold/probe_test.go`; production source remains unchanged. Each focused command returned actual exit 1 because the expected rejection did not occur. Files under `/tmp/alan-task3-review-casefold` are supplementary; the source/inputs/outputs needed to reconstruct them are retained below.

Original probe source:

```go
package main

import (
 "os"
 "path/filepath"
 "testing"
)

func TestReviewCaseFoldedItemFields(t *testing.T) {
 cases := []string{
  `[{"item_id":"original","ITEM_ID":"replacement","number":2}]`,
  `[{"item_id":"item-a","number":2,"NUMBER":100}]`,
 }
 for _, input := range cases {
  path := filepath.Join(t.TempDir(), "input.json")
  if err := os.WriteFile(path, []byte(input), 0600); err != nil { t.Fatal(err) }
  got, err := readItems(path)
  t.Logf("input=%s parsed=%+v error=%v", input, got, err)
  if err == nil { t.Error("case-aliased duplicate semantic field accepted") }
 }
}
```

Original command:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build go test -overlay=/tmp/alan-task3-review-casefold/overlay.json ./cmd/alan-compose -run '^TestReviewCaseFoldedItemFields$' -count=1 -v
```

Original exact output (no results were overwritten):

```text
=== RUN   TestReviewCaseFoldedItemFields
    main_test.go:18: input=[{"item_id":"original","ITEM_ID":"replacement","number":2}] parsed={Kind:list Number:0 Text: Flag:false List:0x64e840f34438} error=<nil>
    main_test.go:19: case-aliased duplicate semantic field accepted
    main_test.go:18: input=[{"item_id":"item-a","number":2,"NUMBER":100}] parsed={Kind:list Number:0 Text: Flag:false List:0x64e840f34510} error=<nil>
    main_test.go:19: case-aliased duplicate semantic field accepted
--- FAIL: TestReviewCaseFoldedItemFields (0.00s)
FAIL
FAIL	github.com/JakeFAU/alan/cmd/alan-compose	0.351s
FAIL
```

A second focused invocation used the identical cases and command, replacing only the pointer-form logging statement with `rows, recordsErr := got.Records()` and `t.Logf("input=%s parsed=%+v error=%v records_error=%v", input, rows, err, recordsErr)` to expose the actual rewritten fields. Full source SHA256: `2b13e4d4e5eb1c8ba1cb65ef4259120b1bfa113bf463693efcf684060526a7b6`. Exact output:

```text
=== RUN   TestReviewCaseFoldedItemFields
    main_test.go:19: input=[{"item_id":"original","ITEM_ID":"replacement","number":2}] parsed=[map[item_id:{Kind:string Number:0 Text:replacement Flag:false List:<nil>} number:{Kind:int Number:2 Text: Flag:false List:<nil>}]] error=<nil> records_error=<nil>
    main_test.go:20: case-aliased duplicate semantic field accepted
    main_test.go:19: input=[{"item_id":"item-a","number":2,"NUMBER":100}] parsed=[map[item_id:{Kind:string Number:0 Text:item-a Flag:false List:<nil>} number:{Kind:int Number:100 Text: Flag:false List:<nil>}]] error=<nil> records_error=<nil>
    main_test.go:20: case-aliased duplicate semantic field accepted
--- FAIL: TestReviewCaseFoldedItemFields (0.00s)
FAIL
FAIL	github.com/JakeFAU/alan/cmd/alan-compose	0.144s
FAIL
```

Case-only probe source:

```go
package main

import (
 "os"
 "path/filepath"
 "testing"
)

func TestReviewCaseOnlyItemFields(t *testing.T) {
 cases := []string{
  `[{"ITEM_ID":"case-only","NUMBER":2}]`,
 }
 for _, input := range cases {
  path := filepath.Join(t.TempDir(), "input.json")
  if err := os.WriteFile(path, []byte(input), 0600); err != nil { t.Fatal(err) }
  got, err := readItems(path)
  rows, recordsErr := got.Records()
  t.Logf("input=%s parsed=%+v error=%v records_error=%v", input, rows, err, recordsErr)
  if err == nil { t.Error("noncanonical fields accepted as declared schema fields") }
 }
}
```

Case-only command and exact output:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build go test -overlay=/tmp/alan-task3-review-casefold/overlay.json ./cmd/alan-compose -run '^TestReviewCaseOnlyItemFields$' -count=1 -v
```

```text
=== RUN   TestReviewCaseOnlyItemFields
    main_test.go:18: input=[{"ITEM_ID":"case-only","NUMBER":2}] parsed=[map[item_id:{Kind:string Number:0 Text:case-only Flag:false List:<nil>} number:{Kind:int Number:2 Text: Flag:false List:<nil>}]] error=<nil> records_error=<nil>
    main_test.go:19: noncanonical fields accepted as declared schema fields
--- FAIL: TestReviewCaseOnlyItemFields (0.00s)
FAIL
FAIL	github.com/JakeFAU/alan/cmd/alan-compose	0.161s
FAIL
```

Overlay JSON:

```json
{"Replace": {"/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan/cmd/alan-compose/main_test.go": "/tmp/alan-task3-review-casefold/probe_test.go"}}
```
