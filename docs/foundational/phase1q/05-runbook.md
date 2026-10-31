# Phase 1Q — Executed checks and reproduction

The [baseline](evidence/baseline.json) records the fetched merge, complete tracked inventory and actual Go environment. Run from the repository root using Go 1.27.3 on darwin/arm64 with CGO enabled. The commands below use the same local toolchain, offline modules and readonly dependency policy as the recorded checks.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1q-go-cache
probe='^TestDiagnosticRelayHandleObservation$'
boundary='^Test(InvocationHandleRoundTripThroughList|InvocationAwaitRejectsKnownWrongDeclarationAndNonString|AdapterAwaitStillRejectsPlainString|SerializedInvocationHandleCannotRebindOutcome)$'
go test -tags=integration -list "$probe" ./internal/substitutiontest
go test -list "$boundary" ./author
go test -tags=integration -run "$probe" -count=1 -timeout=2m -v ./internal/substitutiontest
go test -race -tags=integration -run "$probe" -count=1 -timeout=2m -v ./internal/substitutiontest
go test -run "$boundary" -count=1 -timeout=2m -v ./author
go test -race -run "$boundary" -count=1 -timeout=2m -v ./author
gofmt -d internal/substitutiontest/diagnostic_relay_identity_test.go
go vet -tags=integration ./internal/substitutiontest
go mod verify
```

Exact executed argv, including flag order, are in the [14-record index](evidence/probe/probe-records.json). Each compressed record contains its cwd, UTC, environment, source identities, output and exit. Final normal and race runs each passed the two probe leaves and all four separately selected inherited tests. The two-minute Go watchdog bounds test elapsed time. The probe itself derives a 16-commit upper bound from its acyclic code and one-element loop, and requires exactly three protocol dispatches. It ran at physical activation capacity one. Vet and the formatting diff had empty output, and module verification passed.

The earlier successful behavioral invocations used Go’s default timeout. Following review, all four were repeated with the explicit watchdog at identical behavioral source bytes. Discovery also compiled an earlier source revision that referenced the wrong inspection column. It listed the test without exercising that assertion. The [discovery patch](evidence/probe/probe-discovery-source.diff) reconstructs that earlier file from the submitted source and the [revision record](evidence/probe/probe-source-revisions.json) identifies both hashes. The corrected probe reads the stored program bytes. All fourteen command records are preserved.

To inspect the source of the two branch decisions after temporary stores are removed, read the verbose output in the final normal and race records. Each line containing `handle_observation_evidence=` carries a genuine public Store inspection. Match the root's original sequence-two intent to the admitted child, publication, outcome delivery and observation selected in the parent history. Inspect the next PC and the subsequently committed branch marker. The [independent raw review](evidence/probe-r1.md) records those joins and the exact limits.

```sh
python3 docs/foundational/phase1p/evidence/check_reference.py
```

This read the archived N reference product and policy correspondence directly from their immutable commit. Its [Phase 1Q output](evidence/inherited-reference.json.gz) is evidence inspection. The depth-zero census remains Phase 1N's execution. Nested relay qualification has no new runs on the obstruction branch.

The final source and build checks are recorded in [review and delivery](06-review.md). To reproduce manuscript alignment and whole-change-set verification from a later checkout, pass the exact Phase 1Q delivery commit to `python3 docs/foundational/phase1q/evidence/verify_sources.py`.
