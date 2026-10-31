# Reproduce the finite-tree qualification

Use Go 1.27.3 on the local CGO toolchain and the existing sqlite3 module cache. No provider calls or downloads are needed. The evidence directory contains separate compressed command records with actual argv, executable, working directory, selected environment, inputs, outputs and exit status. Public fixture capture uses a fresh ALAN_I_ARTIFACTS directory; reusing one causes the recorder to refuse an overwrite.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1q-go-cache
go test -count=1 -timeout=3m -v ./examples/owningtree
go test -race -count=1 -timeout=3m -v ./examples/owningtree
go -C examples/owningtree/consumer test -count=1 -timeout=8m -v ./...
go -C examples/owningtree/consumer test -race -count=1 -timeout=15m -v ./...
go test -run '^Test(FiniteTreeCensus|TreeCounterexamples|TreeProgressEnumeration)$' -count=1 -timeout=5m -v ./research/owningtree
go test -race -run '^Test(FiniteTreeCensus|TreeCounterexamples|TreeProgressEnumeration)$' -count=1 -timeout=5m -v ./research/owningtree
auto='^TestOwningTree(Pairs|Nested)$'
schedules='^TestOwningTree(TightLimits|DelayedFrontier|LateReply|Invalid|FaultIsNotPending)$'
ALAN_I_ARTIFACTS="$(mktemp -d)" go test -tags=integration -count=1 -timeout=60m -parallel=4 -run "$auto" -v ./internal/substitutiontest
go test -race -tags=integration -count=1 -timeout=60m -parallel=4 -run "$auto" -v ./internal/substitutiontest
ALAN_I_ARTIFACTS="$(mktemp -d)" go test -tags=integration -count=1 -timeout=40m -parallel=3 -run "$schedules" -v ./internal/substitutiontest
go test -race -tags=integration -count=1 -timeout=40m -parallel=3 -run "$schedules" -v ./internal/substitutiontest
go test -run '^(TestRetainedB|TestOriginalBindingsAndCommonPolicy|TestNominalCandidateCertificates)' -count=1 -timeout=4m -v ./research/substitution ./examples/callablejoin ./research/nominal
go test -race -run '^(TestRetainedB|TestOriginalBindingsAndCommonPolicy|TestNominalCandidateCertificates)' -count=1 -timeout=6m -v ./research/substitution ./examples/callablejoin ./research/nominal
go vet -tags=integration ./research/owningtree ./examples/owningtree ./internal/substitutiontest
go -C examples/owningtree/consumer vet ./...
go mod verify
go -C examples/owningtree/consumer mod verify
python3 docs/foundational/phase1t/evidence/audit_model.py
python3 docs/foundational/phase1t/evidence/audit_runtime.py
python3 docs/foundational/phase1t/evidence/review_constructor.py
python3 docs/foundational/phase1t/evidence/catalogue.py --check
python3 docs/foundational/phase1t/evidence/verify_delivery.py
```

The [A base](evidence/leaf-actual-public.json.gz) and [independent instruction audit](evidence/leaf-actual-pc-audit.json.gz) have separate records. For the standalone consumer use the [building and execution records](evidence/construction.md); the consumer-execution record contains its real inspection and replay. [Model](evidence/model.md) gives the graph representation and source of every computed count. The model covers a larger input domain than the public campaign. The [80 automatic leaves](evidence/automatic-leaves.txt), [14 manual/adverse leaves](evidence/manual-leaves.txt) and [run index](evidence/run-index.json) identify each normal capture. Each index entry connects the uncompressed Go value hash with the recorded invocation that produced it.

**Revisions during qualification.** The first positive automatic campaign exercised the original A/B inputs before an exact schema-identity check was added to reject a narrowed candidate input. The frozen A/B constructor produces identical retained bytes after the fix. The final automatic normal/race campaign exercises the corrected code. A separate evidence-only change records the rejected observation, error, and pre/post snapshots on the delayed-right schedule. The affected leaf was rerun normally and with race detection; its normal replayable evidence supersedes that one earlier capture. Concurrent changes listed in a recorder include unrelated source files watched by the recorder. Use the reviewed source hashes of the actual gated path and the explicitly named reruns. The signed reviews and identity index identify final inputs. These are focused phase checks.
