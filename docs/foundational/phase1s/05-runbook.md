# Phase 1S qualification commands

Run in the repository root with Go 1.27.3, CGO and cached readonly dependencies. The compressed command records retain argv, cwd, UTC, relevant environment, source identities at invocation, exit, stdout/stderr and its SHA-256. The [input audit](evidence/audit_inputs.py) checks inherited bytes against the pinned integration commit; N and R histories are read as prior evidence. The [local leaf list](evidence/local-leaves.txt) gives the 51 expected leaf names for each definitive local run. The three selected public tests have no subtests. The run output identifies each executed name.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1q-go-cache
python3 docs/foundational/phase1s/evidence/audit_inputs.py
go test -list '^TestRetainedB' ./research/substitution
go test -tags=integration -list '^(TestRetainedB|TestNominalWrongAnswerCertifies)' ./internal/substitutiontest
go test -list '^Test(CallableJoinAlignedLocal|WeakRefinementF3DivergenceSeparateFromSafety|NominalCandidateCertificates)' ./research/substitution ./research/nominal
go test -run '^(TestCallableJoinAlignedLocalQualification|TestWeakRefinementF3DivergenceSeparateFromSafety|TestNominalCandidateCertificates)$' -count=1 -timeout=3m -v ./research/substitution ./research/nominal
go test -run '^TestRetainedB' -count=1 -timeout=3m -v ./research/substitution
go test -race -run '^TestRetainedB' -count=1 -timeout=3m -v ./research/substitution
public='^Test(RetainedBClosedHelperReply|RetainedBOptionalResponseFaultControl|NominalWrongAnswerCertifies)$'
ALAN_I_ARTIFACTS="$(mktemp -d)" go test -tags=integration -run "$public" -count=1 -timeout=3m -v ./internal/substitutiontest
go test -race -tags=integration -run "$public" -count=1 -timeout=3m -v ./internal/substitutiontest
go vet -tags=integration ./research/substitution ./internal/substitutiontest
gofmt -l research/substitution/retained_b_correspondence_test.go research/substitution/retained_b_witness_test.go research/substitution/retained_b_audit_test.go internal/substitutiontest/retained_b_closure_test.go internal/substitutiontest/retained_b_fault_test.go
go mod verify
```

Go's public recorder writes to a fresh directory per invocation. The two [preserved gzip records](evidence/public) are verbatim copies from the normal run. To inspect closure, follow the helper's original committed message through the root cancellation, the helper's own freeze, and the first authorization and disposition of that same message. For the fault control, follow the original h invocation to the missing-reply envelope and selected observation, and compare the candidate's pre/post Await projection with the next retained instruction. An open projection alone cannot distinguish an ordinary unmatched wait from a failed reduction.

The phase-specific tests are opt-in fixed-instance checkers in test files. Qualification used the focused normal/race selectors above. No dependencies, external services or extra relay depths are needed.
