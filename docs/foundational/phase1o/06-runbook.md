# Phase 1O — Executed runbook and immutable evidence

This records completed focused commands and how to inspect/reproduce their exact argv, followed by the controller's single manuscript build and inspection. Documentation of these records adds no execution. Run from the repository root unless a record specifies otherwise; archived repository cwd is `/Users/jacob/.codex/worktrees/phase1j-retained-joins/alan`.

## 1. Recorded environment and selection

Final [R2 toolchain](evidence/core/task3-r2-toolchain.json.gz): `/opt/homebrew/bin/go version` exits 0 with `go version go1.27.3 darwin/arm64`. [Staticcheck version](evidence/core/task3-r2-staticcheck-version.json.gz) is `2026.2.2 (0.8.2)`. Child commands used:

```text
GOTOOLCHAIN=local
CGO_ENABLED=1
GOPROXY=off
GOSUMDB=off
GOFLAGS=-mod=readonly
GOCACHE=/private/tmp/alan-phase1n-model-go-build
```

Every ALAN_* variable and OPENAI_API_KEY was removed without recording secret values. Each JSON retains the environment allowlist (including PATH), UTC, cwd, argv, exit, stdout/stderr, selected Go events and source SHA-256 manifest. The allowlist is not a capture of every inherited environment variable.

Discovery was `/opt/homebrew/bin/go test -list '^TestWeakRefinement' ./research/substitution`, exit 0. Normal/race use the exact anchored alternation of all 26 discovered top-level names, not an inferred broad prefix. Literal selections and every argv element are in the immutable records and can be printed by the decoder below. No new existence search or whole-profile census was run; there are no search bounds or NONE verdicts to report. Submitted graph/relation/path sizes bound the certificate checks; rank/cycle certificates are separate.

## 2. Completed corrected-source commands

The final normal/race argv use this exact recorded selector (the shell variable only avoids repeating its literal value):

```sh
phase1o_selector='^(TestWeakRefinementOriginalBranchAccounting|TestWeakRefinementIllegalMergedHelperPrefix|TestWeakRefinementRawDemandMap|TestWeakRefinementF5InvalidAuthorization|TestWeakRefinementSixAuthorizationAdmissionSchedules|TestWeakRefinementAuthorizationInvalidControls|TestWeakRefinementAuthorizationOutsideProfile|TestWeakRefinementAuthorizationHistoryAndPending|TestWeakRefinementNormalizationRelations|TestWeakRefinementNormalizationHelperFork|TestWeakRefinementNormalizationReadinessMutation|TestWeakRefinementF1Noncanonical|TestWeakRefinementF1RejectedPathsRemainOpen|TestWeakRefinementF2ImmediateReadiness|TestWeakRefinementF3DivergenceSeparateFromSafety|TestWeakRefinementF4VisibleOccurrence|TestWeakRefinementEveryPairAndMove|TestWeakRefinementOrderedFactsAndGrammar|TestWeakRefinementUnreachableAndMalformed|TestWeakRefinementStrongGuardsAndDirectIntegrity|TestWeakRefinementProgressBoundaries|TestWeakRefinementGrammarCannotEraseFacts|TestWeakRefinementSilentInteriorsPreservePublicFacts|TestWeakRefinementRequiredGrammar|TestWeakRefinementSilentSemanticChoice|TestWeakRefinementPrivateSelectedFields)$'
/opt/homebrew/bin/go test -json -count=1 -run "$phase1o_selector" ./research/substitution
/opt/homebrew/bin/go test -json -race -count=1 -run "$phase1o_selector" ./research/substitution
```

The regression command's literal argv is:

```sh
/opt/homebrew/bin/go test -json -count=1 -run '^TestWeakRefinement(RequiredGrammar|SilentSemanticChoice|PrivateSelectedFields|NormalizationRelations)$' ./research/substitution
```

| Immutable record | Exact command form / recorded result |
| --- | --- |
| [R2 regression RED](evidence/core/task3-r2-red.json.gz) | Regression command above; exit 1, 30 run events, 19 failed/11 passed, against the exact rejected R1 core. |
| [R2 regression GREEN](evidence/core/task3-r2-green-regressions.json.gz) | Same literal argv, corrected source; exit 0, 30 passes. |
| [R2 normal](evidence/core/task3-r2-normal.json.gz) | Normal command above, with the literal 26-name anchored alternation in that record. Exit 0, 26 top-level/96 test events/85 leaves, zero failures/skips. |
| [R2 race](evidence/core/task3-r2-race.json.gz) | Race command above, with the same selection and source manifest; same counts and exit 0. |
| [R2 vet](evidence/core/task3-r2-vet.json.gz) | `/opt/homebrew/bin/go vet ./research/substitution`; exit 0, empty output. |
| [R2 modules](evidence/core/task3-r2-modules.json.gz) | `/opt/homebrew/bin/go mod verify`; exit 0, `all modules verified`. |
| [R2 format](evidence/core/task3-r2-format.json.gz) | `/opt/homebrew/bin/gofmt -d` with the seven exact new-file arguments in the record; exit 0, empty output. |
| [R2 staticcheck](evidence/core/task3-r2-staticcheck.json.gz) | `/opt/homebrew/bin/staticcheck ./research/substitution`; exit 1, the six inherited diagnostics below. |

The RED and GREEN selected commands differ in source identity, not expected semantics. RED used core SHA `6ae6c988afbe5c694323f12c25fe0206214394078b03962aa63f1afda2d65902`; final core is `ffbeb5c07d6ab66020ac9242e25935d75514b8e5248a19de501bd850fe84f8a1`. Corrected normal/race/static/module/format/vet manifests agree. A corrected source does not inherit a prior source's passing run.

Staticcheck diagnostics are ST1005 at `callable_join.go:290,301,307` and `callable_join_local.go:65`, and U1000 at `invocation_join_explorer.go:84` and `invocation_join_oracle.go:61`. The four original files remain byte-identical to the baseline/recorded originals; review verified their identity. No suppression or new-file diagnostic appears. This exit is retained as failure, not converted into a green staticcheck claim.

## 3. Exact-argv and record audit

The immutable [R2 index](evidence/core/task3-r2-index.json), SHA `0073057241d94f15c51abd635da7970c622522eadbc69901c5481107702d12f6`, pins all 11 compressed records, argv, exits and final source identities. Its historical pending-review wording is unchanged; [Task3 R2](evidence/reviews/task3-r2.md) supplies the later approval. This read-only audit prints actual argv without reconstructing the long selector:

```python
from pathlib import Path
import gzip, hashlib, json, shlex

core = Path("docs/foundational/phase1o/evidence/core")
raw = (core / "task3-r2-index.json").read_bytes()
assert hashlib.sha256(raw).hexdigest() == "0073057241d94f15c51abd635da7970c622522eadbc69901c5481107702d12f6"
index = json.loads(raw)
for entry in index["records"]:
    packed = (core / entry["file"]).read_bytes()
    assert hashlib.sha256(packed).hexdigest() == entry["sha256"]
    record = json.loads(gzip.decompress(packed))
    assert record["argv"] == entry["argv"]
    assert record["exit"] == entry["exit"]
    print(entry["file"], "exit", record["exit"])
    print(shlex.join(record["argv"]))
```

Execute the displayed normal/race/vet/module/format/staticcheck argv only against a checkout matching that record's `source_sha256`; use the recorded local/offline environment and retain a new result separately. The staticcheck command is expected to retain the listed six failures if its sources and tool version match. The audit itself was executed during documentation preparation and found no R2 hash/argv/exit mismatch; it was not a rerun of those Go commands. The live reconciled manuscript is not an input to the historical command assertions.

## 4. Preserved failures and earlier evidence

[Original Task3 index](evidence/core/task3-index.json), SHA `434a4b87eb0913e197d158f728fb5e7479aa678a65265e30b7cedba0ea43e5e2`, preserves all 28 earlier compressed runs with their own manifests. They include compile-stub RED, the initial `weakGraph` name collision, corrected legal wrong-word fixture, defensive/private-interior RED, initial new unused assignment caught by staticcheck, and later pre-review normal/race output. No earlier artifact was overwritten. The controller verified the old compressed hashes; the final R2 index records that preservation.

[Task3 R1](evidence/reviews/task3-r1.md) preserves the rejected nil-grammar, overstrong silent-demand and Direct/private-field behaviors. Its independent probe source and failed execution are archived in [task3-review-probe.tar.gz](evidence/core/task3-review-probe.tar.gz), SHA `383cb1b04956afd00a6f6934a20b943d25f1d6d1729cf49b3b3925fd02bc0309`, with `go.mod`, `types.go`, original core, exact probe test and `probe.json`. In a disposable extraction, its recorded command is `/opt/homebrew/bin/go test -v -count=1 -run '^TestTask3ReviewBoundaries$' .`, exit 1 with all three intended-contract assertions failing. Do not restore old code into the active checkout. This original probe is distinct from the 30-event R2 RED and supplies a complete small reproduction without requiring the live manuscript to resemble an earlier checkout.

The historical 12 Rw strong issues include four invalid `silent_demand` implementation requirements. Keep that output as historical. Corrected output is eight endpoint-demand issues for Rw and zero named strong issues for Rg, each with 16 pairs/12 component/24 bilateral environment obligations. Original-rule anchors and the six J schedules remain finite transcription checks, not runtime or arbitrary-relation proofs.

[Baseline checks](evidence/core/baseline-checks.json.gz) pin inherited focused listing/execution separately; [baseline.json](evidence/core/baseline.json) pins original source identities. No new broad historical gate was justified by an inherited-source change, and none is claimed.

## 5. Manuscript and delivery gate

The [pre-build source audit](evidence/core/source-audit.json), SHA `94e196afe1817d581c968db447aecdb80bf12aae7df4163a41ca6bf9859619eb`, records exact Markdown/TeX alignment and inherited-source preservation. The [whole-phase source review](evidence/reviews/final-source-r1.md) independently approves the qualified source package with no actionable finding. Its source-approval disposition preceded the controller build; the report was finalized afterward and does not claim to review the build or this closure update. No citation changed.

The immutable [paper-build.json.gz](evidence/core/paper-build.json.gz), SHA `003b72da2b8cec994c129d3bc38a0c0268b6ee49802c65014a276d9b66820702`, records two controller steps, both exit 0: one artifact-operation marker and **one** `/opt/homebrew/bin/tectonic alan-dissertation.tex` invocation with cwd `/Users/jacob/.codex/worktrees/phase1j-retained-joins/alan/paper`. Internal TeX/BibTeX passes belong to that single command. Recorded underfull/overfull hbox warnings at TeX lines 93, 189 and 423 are accepted per the user; no source change or repeat build followed.

The resulting 21-page PDF has SHA `2f1737e338641b9c47a203930b28e184bbb58d58160f714a8cd52d6718829937`. [paper-inspection.json](evidence/core/paper-inspection.json), SHA `a0621f9d84068b5cfef6d341240ec2c515f25a22f38a9fe0c7222ea1f829f3d6`, retains exact pdfinfo/pdftotext/pdftoppm argv, PNG hashes and the controller's visual PASS. The controller viewed all four Poppler page images 15–18; changed content is on pages 16–17. It observed readable text/math, no clipping, overlap, missing glyphs or damaged surrounding layout, and ordinary paragraph continuation across the changed pages. This worker inspected those records, not new rendered images, and performed no build/marker/render. Manuscript/BibTeX source identities remained unchanged through the build.

## 6. Planned archival and publication

The controller plans [sdd-workspace.tar.gz](evidence/core/sdd-workspace.tar.gz) and its external [sdd-workspace-manifest.json](evidence/core/sdd-workspace-manifest.json). **Creation and member/hash verification are pending** at this closure update. These are planned durable links, not assertions that the files already exist. The manifest will supply the archive digest and every member's identity without putting a self-hash inside the archive.

Members are relative to the plan directory: `task2-r1-rejected-argument.md` preserves the withdrawn universal source claim; `task3-r1-snapshot/` preserves the rejected seven-Go-source package; independent bridge/quotient reports, review packages, `task5-pre-closure/` and `progress.md` retain their individual identities in the manifest. Existing compressed command records and the independent probe archive remain separately tracked. Only the controller archives and removes this plan's scratch directory after verification. Final closure review and draft PR publication remain pending in [07](07-review.md).
