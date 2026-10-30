# Phase 1N executed runbook and evidence map

This is an index of actual captured commands, not a claim that they ran again
during documentation. All paths below are repository-relative unless linked.
Complete argv, cwd, sanitized/effective environment, stdout/stderr, exits, selected
leaves, source hashes and archives live in the linked records. New reproduction
captures require fresh labels or a disposable checkout; never overwrite evidence.

## Toolchain, sources and capture boundaries

The pinned local toolchain is Go 1.27.3 darwin/arm64, `/opt/homebrew/bin/go`.
All Go profiles use `GOTOOLCHAIN=local`, `CGO_ENABLED=1`, `GOPROXY=off`,
`GOSUMDB=off`, `GOFLAGS=-mod=readonly`. Initial construction records retain the
actual executable token `go`; later records use the absolute path. Caches are
`/private/tmp/alan-phase1l-go-build` for baseline/preflight/controller static,
`/private/tmp/alan-phase1n-model-go-build` for model, and
`/private/tmp/alan-phase1n-runtime-go-build` for runtime. Capture policies strip
unrelated `ALAN_*` variables and `OPENAI_API_KEY`; model/runtime capture installs
its own fresh `ALAN_MODEL_EVIDENCE`/`ALAN_I_ARTIFACTS` destination. Exact destinations
and stripped-variable lists are in each record; no provider call ran.

Base main is `3c5118ea9763a27cca432ca91aac667bd59e21e7`. The controller's
[pre-paper fetch](evidence/core/pre-manuscript-main.json) exited 0 and confirmed
unchanged origin/main and merge base, with HEAD
`cb678321324437d9d5c7247c8a139fc5dc401201` after Task 3b. It also pins pre-edit
TeX/Markdown/BibTeX hashes. [Protected baseline](evidence/core/protected-baseline.json)
records historical identities; original components and production semantics were
not changed. Source identity is evidence provenance, not semantic correctness.

| Boundary | Exact evidence map | Recorded result |
| --- | --- | --- |
| Initial baseline | [baseline](evidence/core/baseline.json.gz) | Consuming policy 8 and reason reference 40 leaves; exit 0 |
| Original reply behavior | [boundary baseline](evidence/core/boundary-baseline.json.gz) | Three unchanged reply tests, normal/race; exit 0 |
| Public construction/consumer | [preflight index](evidence/preflight/index.json) | Four accepted commands, four leaves each; exit 0 |
| Reference freeze | [manifest](evidence/model/reference-r1.json), [archive](evidence/model/reference-r1.tar.gz) | 66 directed/adversarial leaves per profile and focused vet; exit 0 |
| Finite qualification/correspondence | [index](evidence/model/qualification/index.json), [README](evidence/model/qualification/README.md) | 114 Go leaves per profile; current r2 Python checks/audit pass |
| Actual runtime and inherited gates | [index](evidence/runtime/index.json), [README](evidence/runtime/README.md) | 29 new and 82 historical leaves per profile; final raw audit passes |
| Final code static checks | [capture](evidence/core/final-static.json.gz) | Eight commands exit 0; 400 Go/module hashes unchanged |

Each final runtime/historical source map contains 405 inputs, including 400
Go/module files, with aggregate SHA256
`8b0c3e9b628dfbda57822df8e0005369635831ff99ed76ae9c3d075630d92a7d`.
Their source archives share SHA256
`138b4d0ee6a45e3e76d94619926d1d341d7ce4ffc9c37a79c432b479d5e445d4`.
All report no source changes during execution. Earlier captures keep their own
source bytes. The model audit checks that gofmt on the eight earlier qualifier
snapshots gives current bytes; it does not claim the later extractor ran earlier.

## Historical audit reproduction after reconciliation

The saved-evidence audit checks **all 66 files** in the frozen reference manifest
against their working-copy paths. Restore that entire manifest scope from the
[reference source archive](evidence/model/reference-r1.tar.gz) **only in a
disposable copy** before running it. This includes the pre-Phase1N Markdown/TeX
manuscripts and the three ignored scratch files `constraints.md`, `task-3-brief.md`
and `task-3-preparation.md`. The delivered manuscripts intentionally differ because
they contain the valid Phase1N reconciliation; restoring scratch inputs alone
does not suffice. Never overwrite the current working manuscript or change frozen
manifests to accommodate this intentional difference.

The [restoration verifier](evidence/core/final-review-fix.py) pins the manifest
and archive identities below, verifies the exact 66 regular archive members and
every member hash, creates its own disposable copy under `/private/tmp`, restores
all 66 paths there, checks all restored hashes, runs the existing saved-evidence
audit there, and removes the copy. It first reproduces the insufficient
scratch-only restoration as a negative control. Source files are read only;
the specified new output file is the sole persistent write. Use a fresh output
name; the script refuses to replace an existing record.

| Input | Required SHA256 |
| --- | --- |
| reference-r1.json | `4603206f1eccf3bb471ca1df821aa56c613633d98dfdd9d84016786db137023e` |
| reference-r1.tar.gz | `8d2a2cfb47c37f8f4a666e95c5dfa8f386fffef5af46e5ff12a32de22fb1b0b3` |
| Frozen paper/alan-dissertation.md | `a94a65c6a3d1489f826306f32d8546d8461163830a140c5b0ca791dcab815e29` |
| Frozen paper/alan-dissertation.tex | `2d19244524b404712c68d04fb8847809d5a1904847ef1e21c3e9bb57c12d4f1b` |

From the delivered repository root, with the recorded Python/Go toolchain present:

```sh
python3 docs/foundational/phase1n/evidence/core/final-review-fix.py --source . --output /private/tmp/alan-phase1n-restoration-new.json.gz
```

This runs no model search, Go tests, runtime campaign or PDF build. The existing
audit reads saved records, invokes read-only gofmt on archived bytes and checks
the pinned Go version. The [recorded correction check](evidence/core/final-review-fix.json.gz)
retains exact commands/environment/output and the original two-mismatch witness:
scratch-only audit exit 1; all 66 restored hashes match; restored audit exit 0;
current source hashes unchanged. [Independent R2](evidence/reviews/final-r2.md)
returns F1 ADDRESSED and whole-phase source APPROVE; this correction neither
falsifies nor repeats prior behavioral qualification.

Separately, the qualification index pins `task-3b-report.md` at SHA256
`6d89366d36b61fe45a69a734a3872995a6cde01e0c8e232013f35c80475b1abf`.
The controller preserved that exact report in [the 39-file task-record archive](evidence/task-records.json.gz),
SHA256 `c3c7ad26fa1cfdab7d230d7dc5ae126231b09d65fe504aad1f32a6964ce43889`.
Its schema is `phase1n-task-records-v1`, with the report under
`files["task-3b-report.md"]` as `{sha256,text}`. The following also verifies both
the declared hash and the SHA256 of the exact UTF-8 text, and restores it only
at its expected scratch path in the disposable copy:

```sh
python3 docs/foundational/phase1n/evidence/core/final-review-fix.py --source . --task-records docs/foundational/phase1n/evidence/task-records.json.gz --output /private/tmp/alan-phase1n-restoration-final-records-new.json.gz
```

The [actual final-archive check](evidence/core/final-task-record-restoration.json.gz),
SHA256 `8d96f2a4f2e33186a0322d66c6769057f10a5a837654537f72e9bef8ea106d91`,
passed with the pinned report verified/restored, all 66 manifest inputs restored,
saved-audit exit 0 and current sources unchanged. The original correction record
retains its earlier pending status. The frozen-reference audit itself does not
read this later report, but the qualification index's authored-input identity
requires it. These are historical
audit/record restoration requirements, not runtime dependencies. Do not substitute
current task text for a pinned historical input or alter the frozen indexes.

## Construction and frozen reference

The four accepted construction commands were, from the indicated cwd:

```sh
# Repository root
go test -json -count=1 ./examples/callablejoin
go test -race -json -count=1 ./examples/callablejoin
# examples/callablejoin/consumer, a separate public-import module
go test -json -count=1 ./...
go test -race -json -count=1 ./...
```

The consumer leaves are A answer, B answer, A fail and B absent at id 1/value 0,
with init/artifact/run/replay/inspect evidence. They precede the final Task 4
samples; counts are not combined into a configuration census. Exact command
order/options remain in each indexed preflight record.

Frozen reference construction ran these selectors before any product census or
new artifact inspection; its archive pins the then-current source and test set:

```sh
/opt/homebrew/bin/go test -json -count=1 -run '^TestCallableJoin' ./research/substitution
/opt/homebrew/bin/go test -race -json -count=1 -run '^TestCallableJoin' ./research/substitution
/opt/homebrew/bin/go vet ./research/substitution
```

The broad prefix here describes only the archived reference freeze, not a present
selector for repeating its 66-leaf scope after later tests were added.

## Reference-first finite qualification

For each profile, commands execute the following anchored selectors in order;
normal uses the first exact argv pattern and race the second:

```sh
/opt/homebrew/bin/go test -json -count=1 -timeout=5m -run SELECTOR ./research/substitution
/opt/homebrew/bin/go test -race -json -count=1 -timeout=5m -run SELECTOR ./research/substitution
```

| SELECTOR (one exact argument) | Passing leaves per profile |
| --- | ---: |
| `^TestCallableJoinReferenceQualification$` | 16 |
| `^TestCallableJoinAlignedLocalQualification$` | 32 |
| `^TestCallableJoinReplacementQualification$` | 32 |
| `^TestCallableJoinFiniteLiftingQualification$` | 32 |
| `^TestCallableJoinArtifactCorrespondenceInputs$` | 2 |

Initial/final `-list` discovery and focused vet are retained. Current independent
correspondence commands consume the normal/race artifact captures respectively:

```sh
python3 docs/foundational/phase1n/evidence/model/qualification/check_correspondence.py docs/foundational/phase1n/evidence/model/qualification/normal-artifacts/callable-join-artifacts.json.gz
python3 docs/foundational/phase1n/evidence/model/qualification/check_correspondence.py docs/foundational/phase1n/evidence/model/qualification/race-artifacts/callable-join-artifacts.json.gz
python3 docs/foundational/phase1n/evidence/model/qualification/audit.py
/opt/homebrew/bin/go vet ./research/substitution
```

Current checker labels are `normal-correspondence-r2`, `race-correspondence-r2`
and `audit-r2`; their exact sources and full results are indexed. Earlier labels
with the same argv retain different checker bytes and their own disposition.
Normal/race complete model/local/lifting/artifact results are byte-identical.
All 228 Go leaves passed, with no failing/skipped leaves. Python checks are not Go
test leaves. Independent review verified all 67 indexed evidence files, eight
authored inputs and 66 unchanged reference inputs without rerunning these tests.

## Actual runtime, attacks and focused inherited gates

The final exact commands, all from the repository root, were:

```sh
/opt/homebrew/bin/go test -json -tags integration -run '^TestCallableJoin(Runtime|Modes|Controls|LateReply|Decoder|Prefixes|ControlIdentities)$' -count=1 -timeout=5m ./internal/substitutiontest ./examples/callablejoin
/opt/homebrew/bin/go test -json -race -tags integration -run '^TestCallableJoin(Runtime|Modes|Controls|LateReply|Decoder|Prefixes|ControlIdentities)$' -count=1 -timeout=10m ./internal/substitutiontest ./examples/callablejoin
/opt/homebrew/bin/go test -json -tags integration -run '^Test(ConsumerReason(Runtime|Modes|Controls|Decoder|ClassificationCut)|ConsumerPolicy(Runtime|Controls|Orders|Decoder|HistoricalWrapper)|InvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes))$' -count=1 -timeout=5m ./internal/substitutiontest
/opt/homebrew/bin/go test -json -race -tags integration -run '^Test(ConsumerReason(Runtime|Modes|Controls|Decoder|ClassificationCut)|ConsumerPolicy(Runtime|Controls|Orders|Decoder|HistoricalWrapper)|InvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes))$' -count=1 -timeout=8m ./internal/substitutiontest
/opt/homebrew/bin/go test -tags integration -list '^TestCallableJoin(Runtime|Modes|Controls|LateReply|Decoder|Prefixes|ControlIdentities)$' ./internal/substitutiontest ./examples/callablejoin
python3 docs/foundational/phase1n/evidence/runtime/audit.py normal race
```

All six commands exited 0; final profiles have no failing/skipped leaves. Exact
selected leaves are retained in each `command.json.gz`; 19 principal executions,
seven decoder attacks, two prefix checks and one construction identity leaf make
29 per profile. The historical selection has 82 per profile and compares 48
archived inspections, whose full-value hash agrees across profiles. No historical
archive was rewritten. The final audit covers both profiles and rejects all
seven changed snapshots while accepting their originals.

## Final code static checks and completed paper gate

The controller's capture (SHA256
`a645a99a179067c9a06a8944b2d19a07f98237aa0ee8653b200d73db9d2dba58`)
records these eight commands, all exit 0 with empty stderr; gofmt output is empty:

```sh
/opt/homebrew/bin/go version
git diff --check
/opt/homebrew/bin/gofmt -l author schema runtime effects adapters cmd internal research examples
/opt/homebrew/bin/go vet ./...
/opt/homebrew/bin/go vet -tags integration ./...
/opt/homebrew/bin/go mod verify
/opt/homebrew/bin/go -C examples/callablejoin/consumer vet -tags integration ./...
/opt/homebrew/bin/go -C examples/callablejoin/consumer mod verify
```

This code-static capture precedes manuscript reconciliation. Final source identity
and diff checks are recorded in the [delivery index](evidence/index.json). The
controller's [PDF edit marker](evidence/manuscript/marker.json) has exit 0. After
source review, exactly one `tectonic alan-dissertation.tex` from `paper/` exited 0,
producing the 20-page [PDF](../../../paper/alan-dissertation.pdf), SHA256
`dbe54bd450683fb4987905d47e197b3b1557ac0375c2f336d537be0a2a6ab281`.
The [render record](evidence/manuscript/render.json) retains commands, output,
source/PDF/image identities and visual findings. Phase 1N occupies pages 15–16;
the controller inspected pages 14–18 and found legible text/math and coherent
page transitions without clipping, overlap or missing glyphs. The existing
underfull/overfull box warnings were accepted under the user's policy. No source
edit or second build followed. BibTeX references are unchanged.

## Retained failures and coverage limits

Preflight preserves missing-artifact RED, a test's nonexistent `ListData.Items`
accessor and a PC expectation omitting the compiler's empty-list initializer.
Its mistakenly broad root `go test ./...` was stopped, exited 1, and left an
unfinished adaptive parent/sixteen leaf; it is excluded from qualification. The
archive retains exact output rather than describing a partially run suite as green.

Reference capture permission failure and unrecorded inner verdicts remain
unclaimed. Model `normal-correspondence-1` exited 1 on a wrong unreachable-jump
expectation; current r2 also corrects historical 200/200 metadata to actual 512/64.
Task 4 `red-controls` exited 1 before test execution for the initially absent API.
Its 20-leaf implementation-1 and 29-leaf implementation-2 precede final source
changes and are not relabeled final. Their exact records remain indexed.

These are combined focused gates, not one green root full-suite run. No new
scarcity/cancellation/capacity/crash cross-product or recovery campaign ran.
Source changes after qualification require affected reruns or a specific
unchanged-source justification; later files do not inherit earlier evidence.
