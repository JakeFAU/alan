# Phase 1L reproduction

Run from the repository root at the reviewed Phase 1L source. The recorded host
uses Go1.27.3, darwin/arm64 and CGO1. No live provider calls or new dependencies
are needed. Keep a separate fresh store for each ordinary run; recovery alone
reopens its original store.

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/private/tmp/alan-phase1l-go-build
```

## Public preflight

```sh
python3 docs/foundational/phase1l/evidence/preflight/capture.py
```

The capture builds the separate public consumer, runs its normal/race and static
checks, then creates and executes A/B stores for id1/value0/answer. It preserves
actual argv, environment, exits, sources, artifacts and retained records in
`evidence/preflight/records.json.gz` and its compact index. Direct CLI commands
are in [the example README](../../../examples/invocationjoin/README.md).
This is a two-sample feasibility check, separate from model qualification.

## Reused boundary regressions

These focused checks cover public author/schema/collection behavior and the
invocation admission, reply, descendant join, resource and reopen boundaries
used by this experiment. Their captured outputs are in
`evidence/core/checks.json.gz`; the protected-source manifest binds the inherited
sources. They do not claim a rerun of every historical integration campaign.

```sh
go test -json -count=1 -timeout=10m ./author ./schema ./compose
go test -json -count=1 -race -timeout=15m ./author ./schema ./compose
go test -json -count=1 -tags integration -run '^(TestInvocation.*|TestCompositionRootJoinWaitsAndPublishes|TestCompositionNestedAutomaticRunAndConsumption|TestCompositionFailedChildCannotHideUnresolvedGrandchild|TestCompositionNestedCapacityOneExecutesLeafTransport|TestCompositionReopenPreservesAdmissionBudget)$' -timeout=10m ./internal/executor
go test -json -count=1 -race -tags integration -run '^(TestInvocation.*|TestCompositionRootJoinWaitsAndPublishes|TestCompositionNestedAutomaticRunAndConsumption|TestCompositionFailedChildCannotHideUnresolvedGrandchild|TestCompositionNestedCapacityOneExecutesLeafTransport|TestCompositionReopenPreservesAdmissionBudget)$' -timeout=15m ./internal/executor
go vet ./author ./schema ./compose
go mod verify
```

## Reference and concrete qualification

The [Task2b reproducer](evidence/model/task2b/README.md) gives the exact K1-first
reference/replacement commands, current-source local subset, artifact extraction
and independent correspondence checks. Its index binds command records and all
maximal-state traces. The unchanged01-contract.md describes the pre-census freeze;
the approved reference sources are in
[reference-r2.json.gz](evidence/model/reference-r2.json.gz).

The [runtime reproducer](evidence/runtime/README.md) gives capture, RED/GREEN and
audit details. It records GOCACHE=/tmp/alan-phase1f-go-build, removes ALAN_* and
OPENAI_API_KEY, and creates fresh deliberate evidence/store paths. Exact argv,
environment, selected cases, exits and source identities are in its index/bundle.
Its pending-review wording is historical; the final disposition is in
[03-review.md](03-review.md). These are the final focused selections:

```sh
go test -json -tags integration -run '^TestInvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes|Recovery)$' -count=1 -timeout=10m ./internal/substitutiontest
go test -json -race -tags integration -run '^TestInvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes|Recovery)$' -count=1 -timeout=15m ./internal/substitutiontest
go vet -tags integration ./internal/substitutiontest
python3 docs/foundational/phase1l/evidence/runtime/audit.py
```

Recorded normal/race exits are 0, with 39 leaves each (6.120416s/68.229870s);
integration vet, four-file gofmt check and independent audit pass. Running capture
or audit rewrites its evidence outputs; use a disposable checkout when reproducing
without changing the archived package. Source inspection alone is not a rerun.

The recovery test acknowledges one precise B predicate after Store methods return:
both invokes committed; root open at PC3; original e consumed once; original S
admitted at sequence0/PC0, request unconsumed, outcome unproduced; I/R3/E0/fixed1.
It verifies actual SIGKILL and a fresh-process Open of the same database, then
original-work completion without replacement handles or duplicate consumption.
The same test executable supplies process children, including race instrumentation.

## Default-suite watchdog disposition

The retained original command `go test -json -count=1 -timeout=10m ./...` exited1
after 602.125356s in the unchanged adaptive worst-case byte test. Its output/source
identity remain in [revision1 records](evidence/runtime/revision1/records.json.gz).
README documents 60m. The controller reran only the two unfinished top-level tests:

```sh
go test -json -count=1 -timeout=60m -run '^(TestAdaptiveWorstCaseBytes|TestAdaptiveWorstCaseBytesPreviousDecision)$' ./examples/adaptive
```

[Follow-up evidence](evidence/core/adaptive-watchdog-followup.json.gz) records
exit 0 after 1,002.650s: both top-level tests and 10 named subcases pass, no fail/skip,
all 105 compiled-source hashes unchanged. The earlier 11 adaptive top-level passes
plus these 2 exactly cover the live 13-test listing; other default packages passed
in the original command. This is combined coverage, not a successful monolithic
default command. No assertion, source or tolerance changed to obtain it.

## End-phase manuscript check

After independent whole-change approval, from `paper/` the controller ran exactly
one `tectonic alan-dissertation.tex` invocation. It exited 0 with the accepted
warnings and produced the tracked 18-page PDF. No source was changed by the build.
The insertion is on page 14; visual inspection covered pages 13–15. Exact output,
source/PDF hashes and scope are in [the render record](evidence/final/tectonic.json.gz)
and [inspection record](evidence/final/pdf-inspection.json). Final review and
source-bound evidence disposition are in [03-review.md](03-review.md).
No code changed after the recorded final focused gates.
