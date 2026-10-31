# Reproduce Phase 1R

Use the pinned Go 1.27.3 toolchain, CGO, the existing module cache, and a fresh output directory for each invocation that retains evidence. The SQLite fixtures use temporary independent local stores and no live adapters. The compressed command records under `evidence/` contain the argv, relevant environment, source hashes, exit, and captured output. Go's artifact recorder names each evidence directory by test-name digest and refuses to overwrite it.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1q-go-cache
go version
go test -tags=integration -list '^TestNominal' ./research/nominal ./internal/substitutiontest
go test -tags=integration -list '^TestDiagnosticRelayHandleObservation$' ./internal/substitutiontest
go test -list '^Test(InvocationHandleRoundTripThroughList|InvocationAwaitRejectsKnownWrongDeclarationAndNonString|AdapterAwaitStillRejectsPlainString|SerializedInvocationHandleCannotRebindOutcome)$' ./author
controls='^TestNominal(QAndLiteralTwin|CandidateCertificates|DataFlowAttacks|TransitiveIdentity|IterationBackedge|DependencyLeak|AliasFreshExecution|WrongAnswerCertifies)$'
old='^Test(DiagnosticRelayHandleObservation|InvocationHandleRoundTripThroughList|InvocationAwaitRejectsKnownWrongDeclarationAndNonString|AdapterAwaitStillRejectsPlainString|SerializedInvocationHandleCannotRebindOutcome|OriginalBindingsAndCommonPolicy)$'
ALAN_I_ARTIFACTS="$(mktemp -d)" go test -tags=integration -count=1 -timeout=3m -v -run "$controls" ./research/nominal ./internal/substitutiontest
go test -race -tags=integration -count=1 -timeout=3m -v -run "$controls" ./research/nominal ./internal/substitutiontest
go test -tags=integration -count=1 -timeout=3m -v -run "$old" ./author ./internal/substitutiontest ./examples/callablejoin
go test -race -tags=integration -count=1 -timeout=3m -v -run "$old" ./author ./internal/substitutiontest ./examples/callablejoin
ALAN_I_ARTIFACTS="$(mktemp -d)" go test -tags=integration -count=1 -timeout=3m -v -run '^TestNominalCertifiedRelocation$' ./internal/substitutiontest
go test -race -tags=integration -count=1 -timeout=3m -v -run '^TestNominalCertifiedRelocation$' ./internal/substitutiontest
go vet -tags=integration ./...
go mod verify
```

For each raw relocation inspection, start at its sole original root, follow each original commitment to the corresponding admission and member, then link the committed/frozen bound reply to its publication and the delivery to the caller's actual selected observation. The original history and predecessor PC identify the selected Await and operation register. Inspect both the leaf's local End or open receive and the outward diagnostic, especially for absent and pending. In the two distinct-name alias inspections, compare the two actual branch edges and the operation selected by each receive. The prior same-depth Q probe emits its complete inspections in the verbose output of the inherited command records.

## Broader untagged tests

`go test -count=1 -timeout=60m ./...` passed 23 packages in the sandbox, with five further packages reporting no test files. Six packages failed at local HTTP listeners or their dependent resource processes. The complete six packages passed on a fresh run with loopback access and the same pinned offline Go environment. The combined results cover all 29 packages with untagged tests. [Sandbox invocation](evidence/unit-all.json.gz) · [Loopback rerun](evidence/unit-loopback-retry.json.gz)

```sh
go test -count=1 -timeout=60m ./cmd/alan ./cmd/alan-investigate ./examples/investigation ./internal/durabletest ./internal/executor ./research/semantics
```
