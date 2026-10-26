# Task 1 independent scoped review

## Spec Compliance

- ❌ Issues found: the collection API's malformed-input safety is incomplete. `schema.Field.Append` panics for a valid scalar field/value instead of rejecting a non-list append (`schema/collections.go:295`, `schema/collections.go:299`). This is the Important finding below.
- The reviewed scope otherwise implements the Task 1 interfaces and R6 behaviors: typed lists and copying (`schema/collections.go:54`, `schema/collections.go:85`); single-body structured loops (`author/compile.go:211`); retained snapshots/cursors (`internal/executor/collections.go:168`); version selection (`author/compile.go:67`) and legacy refusal (`internal/executor/program_test.go:189`). No invocation, application builder, second scheduler, or privileged role runtime was introduced in this diff. The existing application provisioning and provider changes are necessary adjacent boundaries.
- ⚠️ Exact historical helper artifact IDs cannot be independently verified from this diff, because the unchanged golden tests are not included. The retained final default-suite output is clean (`docs/foundational/phase1g/evidence/task1/24-final-default-suite.log:1`); the controller should retain the existing identity regression in the eventual integration gate. This review did not re-run that suite.

## Strengths

- Schema/list operational ceilings are checked before recursive cloning; shared byte and field budgets bound aliased graph expansion (`schema/collections.go:420`, `schema/collections.go:507`). Exact canonical-byte tests, duplicate identity checks, empty encoding, nesting, null records and defensive copies exercise meaningful behavior (`schema/collections_test.go:36`, `schema/collections_test.go:131`, `schema/collections_test.go:149`).
- Snapshot capture and element binding are separate reductions, and binding advances only the cursor while body work resumes at the retained PC (`internal/executor/collections.go:168`, `internal/executor/collections.go:182`). Cursor validation rejects missing residuals and invalid origins/indexes; program validation rejects branches entering a body and crossing loop ranges (`internal/executor/collections.go:66`, `internal/executor/collections.go:127`).
- Tests reopen at every instruction boundary across the required counts, cover nested source-list replacement, and preserve Request/Await correlation (`author/collections_test.go:30`, `author/collections_test.go:152`, `author/collections_test.go:207`). Zero iterations do not establish body-only values (`author/collections_test.go:125`).
- New optional fields use omission to preserve scalar encoding, and new program/compiler versions are explicit (`schema/schema.go:13`, `internal/executor/author_metadata.go:10`, `internal/executor/program.go:46`). The scalar-only provider rejects list schemas during Prepare (`adapters/openai/openai.go:49`).
- Original failed attempts and the invalid probe remain explicitly labeled and preserved, followed by focused repairs and a clean final suite (`docs/foundational/phase1g/evidence/task1/commands.json:87`, `docs/foundational/phase1g/evidence/task1/commands.json:123`, `docs/foundational/phase1g/evidence/task1/24-final-default-suite.log:1`). No current success claim is inferred from the misleading historical filename `14-focused-green.log`.

## Issues

### Critical (Must Fix)

- None found in the reviewed scope.

### Important (Should Fix)

1. **Reject scalar receivers before dereferencing list storage.** `schema/collections.go:295` first calls `f.Validate(v)`, which correctly accepts `Field{Kind: "int"}` paired with `Int(1)`. Line 299 then evaluates `v.List.records` even though `v.List` is nil. The public append method consequently crashes its caller for a representable wrong-kind operation, instead of returning its declared error result. This contradicts the task's malformed-schema/value safety requirement and is inconsistent with the other list accessors, which reject wrong kinds explicitly. Add a `f.Kind == "list"` / list-value guard before dereferencing storage; cover int, bool and string receivers, malformed list values, and valid append in a focused regression. The existing reducer only calls this through list declarations, so the demonstrated exposure is the public schema API, not a demonstrated durable-store corruption path.

   Isolated probe against the reviewed checkout (no repository source edits):

   ```go
   package main

   import (
       "fmt"
       "github.com/JakeFAU/alan/schema"
   )

   func main() {
       defer func() {
           if v := recover(); v != nil {
               fmt.Printf("Field.Append scalar input PANIC: %v\n", v)
           }
       }()
       _, err := (schema.Field{Kind: "int"}).Append(schema.Int(1), map[string]schema.Value{})
       fmt.Printf("Field.Append scalar input returned: %v\n", err)
   }
   ```

   Exact command:

   ```sh
   GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build GOTOOLCHAIN=local go run /tmp/alan-phase1g-task1-review-probe/main.go
   ```

   Original output (process exited 0 because the probe deliberately recovered the panic):

   ```text
   Field.Append scalar input PANIC: runtime error: invalid memory address or nil pointer dereference
   ```

   The original standalone source and output remain at `/tmp/alan-phase1g-task1-review-probe/main.go` and `/tmp/alan-phase1g-task1-review-probe/output.log`; their contents are preserved here as durable review evidence.

### Minor (Nice to Have)

- None raised. Existing RED log stack text and earlier “no tests to run” output are explicitly retained historical evidence, not warnings from the final successful verification.

## Assessment

**Task quality: Needs fixes.** The durable collection/iteration design is coherent and has substantial behavioral coverage. Fix the reproducible public append panic before accepting this slice.

**Review limits and checks:** Reviewed brief, implementer report, normative contract R6, and exact `29627d5..a1477e30fa172a44f7ce932458a2847ff88d83d8` review package. Read the diff in sequential sections; re-read only an output-truncated evidence section. No changed source file was read separately and no outside-diff integration inspection was needed. Ran only the specific scalar-append probe above, not a suite, race run, or live-provider call. No agents, credentials, commits, source modifications, or historical evidence edits. This is a Task 1 gate, not approval of the branch or later invocation/store/application work.
