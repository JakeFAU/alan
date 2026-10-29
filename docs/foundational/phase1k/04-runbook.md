# Phase 1K reproduction runbook

Use the repository's pinned Go1.27.3 and sqlite3 v1.14.52 with the existing local
dependency cache. Recorded host: darwin/arm64, CGO enabled. Commands below are
reproduction instructions; exact commands actually executed, environments, source
identities, outputs and exits live beside their evidence. Never write a reproduction
into an existing retained evidence directory.

From the repository root:

```sh
unset OPENAI_API_KEY
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/tmp/alan-phase1k-go-build
go test -v -count=1 -timeout=2m ./research/communication
go test -race -v -count=1 -timeout=2m ./research/communication
go vet ./research/communication
```

The current model/oracle source freeze is in
`evidence/model/revision2/reference-freeze.json`; its normal/race/vet commands and
logs are in that directory. The initial rejected oracle archive and its execution
records remain separately under `evidence/model/`. The finite census retains all
four K1 maxima and the expected ACK-refusal trace. Reproduce the deliberately
false all-complete claim with:

```sh
python3 docs/foundational/phase1k/evidence/model/reproduce-red.py
```

Expected exit is **1**. This semantic RED uses a temporary Go overlay; it does not
change the committed tests or treat the preserved pending maxima as success.

For the exact J2 construction boundary and available artifacts:

```sh
go test -count=1 -v ./examples/communication
go test -race -count=1 -v ./examples/communication
```

These tests verify the required compiler obstruction and available original/root
artifacts. They do not run a complete J2 deployment. The desired runtime-policy
checks in `evidence/context/attempted/` are preserved source, not passing test
evidence. `evidence/context/source-identity.json`, `final-source-check.json` and
`checks/` bind the current source and applicable gates.

From `examples/communication/consumer`, with the same environment:

```sh
go test -tags=integration -count=1 -v .
go test -race -tags=integration -count=1 -v .
go build -o /tmp/alan-phase1k-terminal-consumer .
phase1k_probe_dir=$(mktemp -d)
/tmp/alan-phase1k-terminal-consumer -db "$phase1k_probe_dir/terminal.db" -evidence "$phase1k_probe_dir/evidence"
```

This executable only probes authentic terminal access: source End, frozen route,
admission, original observation selection and receiver End. It creates a fresh
local database. It does not implement J2 or a third join. Final task evidence is
under `evidence/surfaces/terminal-probe-reviewed/`.

Unchanged-core gates were executed separately and passed. Their actual commands
and hashes are in `evidence/gates/core-commands.json`: author/schema/semantics,
four named executor closure/reply/replay tests, unchanged `research/join`, and
`go mod verify`. `evidence/gates/environment.json` records the actual toolchain.

From the repository root, reproduce the complete focused public-driver, decoder,
automatic and process cases with distinct evidence directories:

```sh
phase1k_normal_evidence=$(mktemp -d)
ALAN_I_ARTIFACTS="$phase1k_normal_evidence" go test -tags integration -v -count=1 -timeout=10m -run '^TestCommunication' ./internal/substitutiontest
phase1k_race_evidence=$(mktemp -d)
ALAN_I_ARTIFACTS="$phase1k_race_evidence" go test -race -tags integration -v -count=1 -timeout=30m -run '^TestCommunication' ./internal/substitutiontest
```

The ProcessChild top-level skip is an entrypoint guard, never a qualification
case. Parent Recovery cases require actual pipe acknowledgment, SIGKILL, distinct
fresh PID and Open of the original database. Recreating a Provision is a different
experiment. The refused cut already has original authorization, refused admission
and dispatcher ACK before kill; the reopened explicit retry must preserve those
same identities and results. First authorization after reopen may have a newer
activation epoch than the unchanged latest history sequence.

The false Send-is-enough claim remains an opt-in expected exit1:

```sh
phase1k_red_evidence=$(mktemp -d)
ALAN_I_ARTIFACTS="$phase1k_red_evidence" ALAN_K_RED=send-is-enough go test -tags integration -v -count=1 -timeout=10m -run '^TestCommunicationJ1PublicPair/late$' ./internal/substitutiontest
```

Separate external public-method visibility and independent raw-record audits:

```sh
go test -C docs/foundational/phase1k/evidence/runtime/public-consumer -v -count=1 ./...
go test -C docs/foundational/phase1k/evidence/runtime/public-consumer -race -v -count=1 ./...
python3 docs/foundational/phase1k/evidence/runtime/audit-journals.py "$phase1k_normal_evidence" "$phase1k_race_evidence"
python3 docs/foundational/phase1k/evidence/runtime/audit-process.py "$phase1k_normal_evidence" "$phase1k_race_evidence"
go vet -tags integration ./examples/communication/... ./research/communication ./internal/substitutiontest
```

The initial runtime archive/index and source identity remain immutable. The
corrected decoder freeze is under `evidence/runtime/revision2/`; its full normal
and race records are separate from earlier source versions. The exact latest
results and reviewed scope are recorded in qualification and review documents.
J2 construction tests report their specific compiler obstruction; no J2 public
pair or automatic sample is represented by these J1 cases.

The final fixture-isolation correction is separately qualified under
`evidence/runtime/final-fix/`. The exact single-leaf reproduction and affected
normal/race checks are:

```sh
go test -tags integration -count=1 -timeout=2m -run '^TestCommunicationDecoder$/^wrong-dispatch-link$' ./internal/substitutiontest
go test -tags integration -v -count=1 -timeout=10m -run '^TestCommunicationDecoder' ./internal/substitutiontest
go test -race -tags integration -v -count=1 -timeout=30m -run '^TestCommunicationDecoder' ./internal/substitutiontest
go vet -tags integration ./internal/substitutiontest
```

Use fresh `ALAN_I_ARTIFACTS` directories to retain these reproductions. The final
source received 21-case focused decoder normal/race coverage; the complete
27-case normal/race campaign retains its R2 source identity. The original panic,
corrected selected-leaf proof, exact commands and source/output hashes are archived.

Exactly one phase-end invocation of `tectonic alan-dissertation.tex` ran from
`paper/` after substantive manuscript and scoped final review. It exited0 and
produced the 18-page PDF with accepted warnings. Build/source/PDF/log hashes are
in `evidence/gates/tectonic.json`; pages6/13/14 were rasterized and visually checked
as recorded in `evidence/gates/pdf-inspection.json`. The TeX source is the
manuscript; the PDF is its renderability artifact.
