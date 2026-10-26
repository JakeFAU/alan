# Phase 1G Task 1 — bounded typed collections and durable iteration

Task scope is implemented: native lists, closed element schemas, snapshot iteration, append, version-selected authoring, and compatibility. No invocation, template admission, schema-4 storage, root budgets, application builder, or runner feature was added. Existing Application.Provision was corrected to use the artifact's actual semantic version and deep-copy its input values.

## Public APIs

- `schema.Value` adds optional `List *ListData`; `ListData` owns private records. `schema.List(records ...map[string]Value) Value` constructs and copies a list. Invalid/oversized inputs produce an invalid list value, which `Valid`/schema validation reject; callers validate against their expected Field. `Value.Records() ([]map[string]Value,error)`, `Value.At(int) (map[string]Value,error)`, `Value.Len() (int,error)`, and `Value.Clone() Value` return independent values. `Value.Equal(Value) bool` compares ordered lists structurally; Go pointer identity is not collection equality. Direct Go Value assignment retains ordinary shallow assignment semantics; constructors, accessors and artifact/projection/result boundaries copy.
- `schema.Field{Kind:"list", Element: schema.Record, MaxItems: n, MaxBytes: b, UniqueKey: "id"}` declares an ordered list of closed records. MaxItems is positive even when empty input is accepted. MaxBytes counts exact canonical typed JSON array bytes, including nested records, tags, keys, escaping and delimiters. UniqueKey is optional and must name a string element field. `Field.Append(Value,map[string]Value) (Value,error)` validates the combined result before copying and never changes its input on failure. `Field.Retain() (Field,error)`, `schema.Default(Field) Value`, and `schema.CloneValues(map[string]Value)` support schema/default/copy boundaries.
- Operational ceilings are explicit: 8 schema/list nesting levels, 128 fields per record, 4096 total retained schema fields, 10000 items per list, and 1 MiB total canonical array bytes. Exact shared byte preflight bounds work before cloning or marshaling small aliased graphs that would otherwise amplify. No arithmetic multiplication is used for count/byte allowance calculation.
- `author.Definition.Collections map[string]schema.Field` declares bounded list locals, initialized to canonical empty lists unless a matching `Values` entry is supplied. Inputs cannot be redeclared as locals. `author.Append(name string, values map[string]Expr) Step` appends one closed record. `author.ForEach(collection Expr, bindings map[string]string, body ...Step) Step` binds element fields to variables. `author.Value(schema.Value) Expr` freezes literals. List literals require a schema supplied by a declared/previously typed destination, a typed payload/append boundary, or a typed comparison operand; an untyped standalone literal cannot invent an element schema. `CollectionSemanticVersion` and `CollectionCompilerVersion` expose the new version names; the old constants retain their values.

Example:

```go
items := schema.Field{Kind: "list", MaxItems: 8, MaxBytes: 8192,
    UniqueKey: "id", Element: schema.Record{
        "id": {Kind: "string", MaxBytes: 20},
        "payload": {Kind: "string", MaxBytes: 20},
    }}
// Declare Inputs["items"] and Collections["results"] with this Field.
author.ForEach(author.Var("items"), map[string]string{"id": "id", "payload": "payload"},
    author.Append("results", map[string]author.Expr{
        "id": author.Var("id"), "payload": author.Var("payload"),
    }))
```

## Residual semantics

ForEach lowers one body, independently of MaxItems, plus snapshot (`foreach`), binding/advance (`next`), and structured return (`loop`) instructions. Snapshot creation commits separately from binding. Binding increments the next index and retains both snapshot and current body PC, so reopening resumes that same body's work. Each instruction reduces once and yields. Nested loops own different cursor entries keyed by their snapshot PCs. Forward exits clean abandoned residuals; lifecycle-closed mid-loop snapshots remain valid retained evidence. Success/failure steps clean active residuals. Forged body PCs lacking their required cursor, invalid cursor origins/indexes, crossing loop ranges, or branches entering a body from outside fail explicitly. Public Goto remains forward only; only validated structured return instructions can go backward in new programs.

Definite assignment includes the zero-iteration exit, so body-only assignments cannot establish post-loop variables. List types include their complete retained field schema, not only Kind. Ordinary Request and its updates remain atomic; iteration does not erase pending request identities, and Await resumes the same body after its correlated result. No aggregate or invocation policy is hidden in a new opcode.

## Version and compatibility matrix

| Program semantics | Compiler | Collections | Storage in this slice |
| --- | --- | --- | --- |
| `alan/1d/v1` | none | rejected in code, projection and admission | existing schema 3 |
| `alan/1e/v1` | `alan/author/v1` | rejected in schemas, code, projection and admission | existing schema 3 |
| `alan/1g/v1` | `alan/author/v2` | native bounded lists and structured iteration | unchanged schema-3 SQL layout, with versioned program/projection JSON |

Compilation selects the new pair only when collection semantics are used; all new absent JSON fields omit. Scalar Value and Field JSON ordering/bytes remain unchanged. The unchanged evidence helper identity regression passed in the full default suite: `investigation/helper1` remains `9c8ec36dacdccde46c960b0d4eef835d2a36cafa4b766c20635a537650a1ff66`; its retrieval program remains `cfa4f77a2253555ed9eee8469744812ff42d4d65b5b3ae6ce21c1a6bc9cb4df2`. Historical source/evidence files were not edited.

An existing untyped `alan/1d/v1` reducer test deliberately uses invalid UTF-8 scalar text to exercise exact encoding/json output accounting. The first focused sweep exposed an accidental tightening. The final executor preserves that original scalar acceptance/equality; schemas and native lists remain strict. Native value JSON decoding rejects invalid UTF-8 bytes, unknown fields and duplicate object keys. Existing canonical historical JSON retains its meaning.

The general effect result normalizer now returns canonical empty-list defaults and copies successful list payloads. The OpenAI Responses serializer/parser remains scalar-only: Prepare explicitly rejects collection input or output schemas before any execution. Lookup's existing exact schema identities already reject unsupported schemas. No provider capability was widened.

## Verification and retained failures

`commands.json` records the exact Go invocations, environment, expected outcome, and original output file for every retained attempt. Logs are never overwritten. Important evidence:

- `01`/`02`: behavioral schema RED/GREEN for accepted counts 0/1/3/7/8 and rejected 9, using distinct IDs with identical payloads.
- `03`/`06`: missing author APIs through per-instruction reopen/replay GREEN, plus one compiled loop body and zero-iteration assignment rejection.
- `07`/`08`: malformed list JSON, closed fields, wrong kinds, invalid UTF-8, duplicate IDs, exact total byte boundaries, deep copies, and canonical failed-result defaults.
- `09`/`11`: nested source-list replacement with independent cursor recovery, Request/Await correlation, explicit flat-provider refusal.
- `10`/`11`: small aliased nested graph rejected before aggregate copying; additional schema/value tests cover nesting, aliased schema total fields, empty element schema round trips, and null-record rejection.
- `12`/`13`: legacy collection admission refusal.
- `14`: failed focused attempt, despite its original filename ending in green; named failure was `TestGeneratedOutputHonorsExactConfiguredJSONBoundary`, repaired without changing the historical test. `17` and subsequent suites passed it.
- `15`: invalid missing-cursor probe that accidentally passed because a reused JSON destination retained old schema map entries. It is explicitly not RED evidence. Fresh decode in `16` exposed the actual missing-cursor failure; `17` passed after the fix.
- `19`/`20`: nested literals checked against the expected boundary schema.
- `21`: first full default suite passed. Self-review then found two production defects, retained as RED in `22`: lifecycle-closed cursor projections and a cyclic author graph overflowing the feature-detection prepass stack. `23` passed after their fixes. The cyclic-graph subprocess uses a bounded Go stack and no provider; its failed child stack text is capped at 1500 bytes by the test.
- `24`: final `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/alan-phase1f-go-build GOTOOLCHAIN=local go test -count=1 ./...` passed, covering all 15 tested packages (runtime and alan-resource have no tests). No production changes followed.
- `25`: supplementary test-only reducer check passed after the full-suite binary had compiled: an over-limit append leaves the original projection byte-identical. No reason to repeat the already clean full suite.

All changed Go files were gofmt'd; the Go-source whitespace check was clean. The complete staged whitespace check reports original indentation/trailing spaces in the retained raw subprocess stack output (`22-final-review-red.log`); that evidence is preserved byte-for-byte. No live provider calls or credentials were used. No integration-tagged or race suite was claimed or run for this slice.

## Changed source files and remaining work

Schema: `schema/schema.go`, `schema/collections.go`, `schema/collections_test.go`.
Author: `author/author.go`, `author/compile.go`, `author/identifiers.go`, `author/application.go`, `author/collections_test.go`.
Executor: `internal/executor/program.go`, `internal/executor/author_metadata.go`, `internal/executor/collections.go`, `internal/executor/program_test.go`.
Adjacent boundaries: `effects/effects.go`, `effects/collections_test.go`, `adapters/openai/openai.go`, `adapters/openai/collections_test.go`.

Evidence lives only in this new `evidence/task1` directory. Parent-owned contract snapshots/fixture plans remain unstaged by this task. Task 2 and later phases still need invocation semantics, schema 4, root-envelope accounting, reusable composition and acceptance fixtures; none are claimed complete here. Independent scoped review remains a parent responsibility.
