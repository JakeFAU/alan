# Task 1 report

Status: DONE. Implementation frozen for independent review; no commits made.

## Result and interfaces

New stores atomically provision schema version 2, key version 2, and 32 crypto/rand bytes encoded as lowercase hex in metadata. Open validates metadata and all retained requests before advancing epochs. External request keys are canonical JSON `[2,namespace,localIntentID]`; local intent, message, lifecycle, authorization and settlement identities are unchanged. Receipt completion and replay reconstruct the key from the stored namespace. Loopback-only fixture validation, immutable descriptors, and authentic late-evidence handling remain intact.

Exact exported interfaces:

```go
const SchemaVersion = 2
const ExternalKeyVersion = 2
func (s *Store) Namespace() string
func ExternalKey(namespace, intentID string) string
```

`Inspection` adds `Namespace string` and `KeyVersion int`; its metadata table also exposes both durable fields. `ReplayReport` adds `Namespace string`. `ExternalKey` is a pure encoder; use the validated namespace from Store/Inspection, not untrusted input. Internal `readNamespace(context.Context, queryer) (string, error)` checks schema, PRAGMA user_version, semantics, key version, and canonical 256-bit namespace encoding.

Reopen and rename retain identity. A closed database copy retains exactly the same namespace and request keys; concurrent independent use of copies is unsupported. Namespace separation does not provide business deduplication. Version-1 stores are rejected with ErrVersion before activation/dispatch; current Inspect/Replay also reject them. No migration or historical rekey; retained old source remains the historical reproduction path.

## Files

Production only: internal/executor/store.go, external.go, inspect.go, replay.go.

New tests: internal/executor/store_namespace_test.go. Covers shared durable endpoint with two identical programs and separate reopen/retransmission, local identity preservation, rename/copy request preservation, old schema rejection without epoch advance, malformed/unsupported metadata, retained-attempt namespace tampering, and canonical key encoding.

Mechanical key-sensitive test updates: internal/executor/async_test.go, fault_process_test.go; internal/durabletest/async_test.go, oracle_test.go, resource_test.go; research/semantics/runtime_conformance_test.go (current executor bridge only; research models untouched).

The endpoint pause filter starts before provisioning supplies its namespace, so its startup Intent filter is unspecified. Before executor launch, the parent sets the exact expected external key from the provisioned namespace. Acknowledgment validation still requires exact key, authorization, process, source/generation/epoch/sequence, and independent endpoint records. Runtime boundary events retain local intent IDs; comparisons explicitly derive their external key. All prior causal/order checks remain.

## Retained counterexample and verification

Evidence root: `.superpowers/sdd/00-plan/evidence/task-1/` (ignored session evidence; preserve during final packaging).

`red/before-source.tar.gz` retains the complete pre-fix working source including the initial failing test (excluding .git, session evidence, and prior Phase1E evidence). `red/input/` retains original provision, both closed runtime stores, shared endpoint store, and endpoint inspection. `red/test.log`: expected FAIL, identical keys, 4 contacts, **1 mutation / total 5**, proving collision. `red/metadata-tests.go` retains additional pre-fix tests; `red/metadata-tests.log` proves missing namespace and old-version activation/epoch advance. Existing baseline reproduction was supplied by parent in docs/foundational/phase1e/evidence/baseline; it was not overwritten.

All test commands used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`.

1. `ALAN_NAMESPACE_EVIDENCE="$PWD/.superpowers/sdd/00-plan/evidence/task-1/red/input" go test -count=1 -run '^TestStoreNamespacesSeparateSharedEndpointAndSurviveReopen$' -v ./internal/executor` — expected FAIL above.
2. `go test -count=1 -run 'TestStoreNamespace|TestStoreRejectsVersionOne' -v ./internal/executor` — expected pre-fix FAIL; logs retained.
3. `go test -count=1 -run 'TestStoreNamespace|TestStoreRejects' -v ./internal/executor` — first intermediate invocation failed to build because external_example still called the old helper signature (`focused.log`). Fixed without expanding production scope: externalRequest keeps its parameter-validation signature, and durable callers bind the namespace. Repeated command PASS (`focused-2.log`, 0.759s).
4. `go test -count=1 ./...` — all packages PASS (`full-1.log`), including durabletest 10.516s and executor 8.905s.
5. `go test -count=1 -race -run 'TestStoreNamespace|TestStoreRejects|TestExternalKey|TestExternalAuthorization|Test.*Async|Test.*Completion|Test.*Receipt' ./internal/executor ./internal/durabletest` — PASS (`focused-race.log`): executor 4.383s; durabletest 35.858s (race subprocess binaries selected by existing harness).
6. `ALAN_NAMESPACE_EVIDENCE="$PWD/.superpowers/sdd/00-plan/evidence/task-1/green/input" go test -count=1 -run '^TestStoreNamespacesSeparateSharedEndpointAndSurviveReopen$' -v ./internal/executor` — PASS; closed stores and endpoint inputs/inspection retained separately in green/input. 4 contacts, **2 mutations / total 10**.
7. `go test -count=1 -json ./...` — final current-source PASS, **368 test/subtest pass events, zero failures/skips**, retained full output `full-final.jsonl`. Package elapsed seconds: boundary .080, fixture .288, cmd/alan 1.082, research/semantics 5.521, executor 10.200, durabletest 11.479; cmd/alan-resource has no test files.
8. `git diff --check` — PASS (no output).

`frozen-source-sha256.json` records every modified/new source file at handoff. Full final race/vet/process qualification gates remain with parent. No provider, adapter, compiler, authoring facade, or application work performed here.
