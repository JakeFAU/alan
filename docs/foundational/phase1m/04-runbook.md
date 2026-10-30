# Phase 1M reproduction and evidence

Run from repository root at the relevant reviewed checkpoint: milestone1
`1f71b56148f2fd6ef41261b7caf2996e8eeb2533`, milestone2
`985e08cf54f2dbcb0014269a49f91babe166f8e7`, based on main
`f5b52853ce3cf2b93563eed5eb8268234386e691`. Recorded platform is Go1.27.3,
darwin/arm64, CGO1; captures use local/offline/readonly dependencies, remove ALAN_*
and OPENAI_API_KEY, then supply their explicit fresh artifact/evidence directory.
No live provider calls or new dependencies are required.

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=1
export GOCACHE=/private/tmp/alan-phase1l-go-build
```

## Evidence entry points

| Layer | Exact commands, sources and records |
| --- | --- |
| Public constructions | [C_se index](evidence/preflight/evidence-index.json), [reason index](evidence/reason-preflight/evidence-index.json), [public CLI](../../../examples/consumingpolicy/README.md) |
| Approved independent freezes | [C_se](evidence/model/reference-r1.json.gz), [reason](evidence/model/reason-reference-r1.json.gz) |
| K1-first census/local/mapping | [C_se reproducer](evidence/model/se-qualification/README.md), [reason reproducer](evidence/model/reason-qualification/README.md) and their index/records bundles |
| Actual runs and independent audits | [C_se runtime](evidence/runtime/se/README.md), [reason runtime](evidence/runtime/reason/README.md) and their command/file indices |
| Reused boundary/static gates | [baseline checks](evidence/core/checks.json.gz), [discovery](evidence/core/discovery.json.gz), [milestone1 static](evidence/core/milestone1-static.json.gz), [final code-static](evidence/core/final-static.json.gz) |

Capture scripts use exclusive labels or write fixed output archives; audit scripts
rewrite their own audit/index outputs. Use a disposable checkout and fresh labels/
directories to preserve originals. Source inspection of saved records is not a
rerun. Construction evidence does not define the independently frozen oracle.

Reason model qualification uses a fresh ALAN_MODEL_EVIDENCE directory per command,
the isolated model cache `/private/tmp/alan-phase1m-model-go-build`, and these
anchored entry points in order. Repeat with -race; recorded timeout is 5m for both
profiles. The [reason index](evidence/model/reason-qualification/index.json) records
the exact capture argv/environment and exporter/checker commands; its bundle
retains every maximum and failed initial checker source.

```sh
go test -json -count=1 -timeout=5m -run '^TestConsumerReasonReferenceQualification$' ./research/substitution
go test -json -count=1 -timeout=5m -run '^TestConsumerReasonReplacementQualification$' ./research/substitution
go test -json -count=1 -timeout=5m -run '^TestConsumerReasonLocalQualification$' ./research/substitution
```

The model capture sets the evidence override; choose separate fresh directories
when invoking these commands directly. Normal/race each pass 16 reference,
32 replacement and 32 local leaves across three commands. C_se separately records
85 leaves/profile across four commands; its reproducer preserves the historical
regression and answer-only local subset. These are combined selected coverages,
not monolithic suite commands.

## Current affected runtime selections

The final captured selections are below. Normal/race results are 25 reason,
57 affected C_es/C_se and 40 focused model/oracle leaves, all exit 0 without failed/
skipped final leaves. Race commands add -race; reason timeout becomes 5m, others
stay unchanged. Exact invocation, cwd, environment, source snapshots, raw records
and selected leaves are in the [reason runtime index](evidence/runtime/reason/index.json).

```sh
go test -json -tags integration -run '^TestConsumerReason(Runtime|Modes|Controls|Decoder|ClassificationCut)$' -count=1 -timeout=3m ./internal/substitutiontest
go test -json -tags integration -run '^Test(ConsumerPolicy(Runtime|Controls|Orders|Decoder|HistoricalWrapper)|InvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes))$' -count=1 -timeout=5m ./internal/substitutiontest
go test -json -run '^TestConsumerReason(Domain|DirectedReference|ClassificationTable|OracleAdversaries|ObservationsOverrideRequestedMode|PendingEvidence|StateGuards|InvalidSControl)$' -count=1 -timeout=3m ./research/substitution
python3 docs/foundational/phase1m/evidence/runtime/reason/audit.py
```

The audit checks saved raw rows/instructions independently of Go monitors and
rewrites this folder's audit/index only. It verifies selected envelopes, actual
classification Set/End, original pending path, exact reasons, negative copies,
sources and historical trace equality. It does not rerun a store or census.
Command records' `records` map retains actual result/cut/replay/journal entries;
each result keeps raw inspection and distinct raw/prefix/component/G verdicts.

Earlier M1 runtime commands pass 19 new and 39 historical leaves per profile.
Their source manifests are historical snapshots; later reason CLI/sidecar additions
are not relabeled as present during those runs. Final affected checks preserve
the reused semantic boundaries and exact 48-inspection historical equality. Only
M1 reran L's existing recovery test. Its exact candidate-first predicate, SIGKILL/
fresh-process behavior and source identity remain in the
[L runbook](../phase1l/04-runbook.md); no C_se/reason recovery campaign is claimed.

## Static records and task archive

Final code-static captures root `git diff --check`, repository-wide `gofmt -l`,
root `go vet ./...` with and without integration tags, root `go mod verify`,
and consumingpolicy consumer-module integration vet/module verification. All
seven commands exit 0; their actual cwd/environment and 378 Go/module hashes are
in final-static.json.gz. The 380-path final runtime manifest additionally includes
capture/audit scripts. No new broad default/adaptive campaign is claimed.

The controller archived [task-records.json.gz](evidence/task-records.json.gz) before scratch cleanup.
It maps exact scratch-relative filenames to `{sha256,text}` under `files`, with
format/base/checkpoint metadata. For example, `files["task-5b-report.md"].text`,
`files["task-6-report.md"].text`, `files["task-4-fix-r1.md"].text` and
`files["progress.md"].text` recover the unique reports, correction and ledger.
Verify each text's SHA256 before use. This avoids links to removed scratch paths;
independent reviews are already permanent under evidence/reviews. The archive
hash is not embedded in a self-referential manifest.

The [final source review](evidence/reviews/final-r1.md) approves Task 7 and the
whole branch. After that review, the controller ran exactly one
`tectonic alan-dissertation.tex` from `paper/`, then `pdfinfo`, `pdftotext -layout`
and `pdftoppm` for pages 13–16. The [render record](evidence/manuscript/render.json)
retains exact argv/cwd/exits/stdout/stderr, source/PDF hashes and visual findings.
The 19-page PDF contains Phase 1M on 14–15; accepted TeX warnings remain recorded.
These are current executed checks, independent of historical L rendering.
