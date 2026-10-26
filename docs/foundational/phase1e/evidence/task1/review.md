# Phase 1E Task 1 independent review

Verdict: **PASS within Task 1 scope.** No correctness, specification-compliance, or code-quality finding requiring a change was identified. This is a review of the namespace slice, not a production-readiness or whole-Phase-1E claim.

## Reviewed material and integrity

Reviewed the task brief/report/review-package diff; Phase 1E plan Global Constraints and Review Focus; the authoring/effects contract; requirements section 7; all four changed production files; the new namespace test file; and all six mechanically changed existing test files. Inspected the surrounding activation, authorization, transport, completion, ledger-validation, barrier acknowledgment, and causal-oracle code. All 11 files listed in `evidence/task-1/frozen-source-sha256.json` matched their frozen SHA-256 hashes. No runtime, test, contract, or retained evidence input was modified, and no commit was made.

Applied the Go safety, testing (review mode), and error-handling (review mode) skills. Reviewed sequentially as requested.

## Specification compliance

- **Durable namespace and versioning:** `internal/executor/store.go:298–315` obtains 32 bytes from `crypto/rand`, encodes canonical lowercase hex, and inserts the namespace, schema version 2, key version 2, and PRAGMA version inside the same provisioning transaction as the store. No usable Store is returned before provisioning commits. `ExternalKey` encodes the version, namespace, and existing local intent ID as a JSON array; the literal expected encoding test and two-store endpoint test constrain both its representation and isolation behavior.
- **Old-store rejection before activation:** `readNamespace` reads version-1-compatible metadata fields before selecting the new columns. `Open` calls full ledger validation before any epoch UPDATE (`store.go:479–491`). The version-1 test reconstructs the old metadata shape without namespace/key columns and verifies `ErrVersion` from Open/Replay/Inspect plus an unchanged epoch. There is no migration or automatic historical rekey.
- **Retained requests and physical recovery:** prepare, completion recording, and replay validation reconstruct the external key with the provisioned namespace (`external.go:92`, `:279`, `:440`). Replay compares the complete retained attempt with the reconstructed immutable request, including original authorization epoch and ID. Reopen returns previously retained attempts unchanged; new physical epochs do not change logical operation keys. Local journal/message/lifecycle/settlement IDs remain outside the new encoding.
- **Authentic late evidence:** completion recording still binds the candidate to its retained attempt, committed intent, original authorization, and complete request. It does not substitute current epoch authority. The existing old-epoch/forgery test passes with the new namespace binding.
- **Rename/copy and independent stores:** tests exercise separately initialized identical programs against a single durable endpoint, then reopen each and observe two mutations across four contacts with preserved retransmission keys. Closed-file rename and copy retain the original namespace and request. The contract and accessor documentation correctly say that a copy is not an independent identity and concurrent independent use of copies is unsupported.
- **Existing protections:** literal-loopback HTTP validation and immutable fixture parameter checks are unchanged. Source authorization and intent commit still precede transport. The change does not add provider behavior, retries, dispatch authority to replay, or business deduplication claims.

## Code quality and oracle review

The production change is small and localized. `readNamespace` centralizes version/encoding validation; the Store namespace is assigned only during creation or validated opening, so the accessor introduces no mutable shared state. `ExternalKey` is an encoder rather than a validator; its callers use the validated Store/ledger namespace as described in the task report. No introduced swallowed operational error, unsafe resource lifetime, or changed concurrency ownership was found.

The causal/crash updates translate local intent IDs to namespaced external keys only at the runtime/endpoint comparison boundary. All paused endpoint callers set the exact expected key after provisioning and before starting the executor. Although the child endpoint startup filter has an empty key, the parent acknowledgment validation uses that exact key, original authorization ID, process identity, source/generation/epoch/sequence, and retained endpoint facts. The causal event order, held in-flight checks, peer consumption checks, SIGKILL acknowledgment gate, original request preservation, and independent endpoint application checks remain in place. The negative causal-oracle test remains capable of rejecting altered order/identity; no assertion was removed to accommodate the change. The research change is confined to the current executor bridge's key comparisons.

## Independent verification and limits

Ran this focused command on the frozen source, with `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off`:

```sh
go test -count=1 -run '^(TestStoreNamespacesSeparateSharedEndpointAndSurviveReopen|TestStoreNamespaceSurvivesRenameAndCopy|TestStoreRejectsVersionOneBeforeActivation|TestStoreRejectsMalformedNamespaceAndKeyVersion|TestStoreNamespaceBindsRetainedExternalAttempt|TestExternalKeyCanonicalEncoding|TestAsyncLateReceiptValidatesRetainedEpochAndRejectsForgery|TestAsyncOracleRejectsFalseOrderAndIdentity)$' -v ./internal/executor ./internal/durabletest
```

The first sandboxed invocation passed the late-receipt and oracle checks but failed while starting `httptest.NewServer`: `listen tcp6 [::1]:0: bind: operation not permitted`. This was a sandbox fixture-listener restriction, not a failed runtime assertion; remaining executor tests did not execute in that invocation. The same command with authorized loopback access passed all selected tests and five malformed-metadata subtests (executor 0.848s; durabletest 0.126s).

The reported baseline 357/current 368 pass events and focused race run were supplied by the implementation handoff; this review did not rerun the full suite, races, vet, or the complete subprocess qualification matrix. No new negative-control mutation was injected. The review checked changed crash-oracle code and ran its existing identity/order negative test, but does not independently certify every historical crash case. Unsupported concurrent copy use, future adapters/authoring APIs, and later Phase 1E slices remain outside this decision. No newly discovered defect/counterexample required preservation; original red/green inputs and archives remain untouched.
