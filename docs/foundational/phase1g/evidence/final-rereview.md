**M1 — Reproduction command metadata inherits the freeze host — ADDRESSED.** `tools/qualify.py:54` captures `platform.platform()` and `:55` obtains `go version` using the same selected environment and root working directory as the actual command at `:60`. `GOTOOLCHAIN=local`, readonly module selection, caches/proxy and credential removal are established before both subprocesses. A version-probe failure raises rather than substituting freeze metadata. `:50` and `:51` retain the original platform, toolchain, directory and credential-absence fields during `--parse-existing`; the existing source/raw-hash checks remain at `:44` and `:46`. The final metadata uses those execution fields at `:100` and `:106`.

**M2 — Older README sections contradict the new profile and omit its consumer command — ADDRESSED.** `README.md:62` distinguishes legacy scalar schema3 from complete schema4 typed lists, snapshot iteration and retained-template invocation. `:160` through `:163` supply separate-module consumer normal/race tests with the qualified 240s watchdog, vet and module verification. `:170` explicitly says root `./...` excludes that module and links the runbook. The location table at `:174` and `:178` and current limits at `:190` consistently describe bounded creation from a retained catalogue, shared root limits and legacy support without migration.

**New breakage in the fix diff — None.** No new Critical, Important or Minor issue identified in the changed collector, focused regression, verification/audit helpers, documentation or evidence association.

**Out-of-scope observations — None.** The existing whole-branch review remains the runtime assessment; this rereview does not repeat or extend it.

**Scope checked —** Base `e81ec0b0b0b124e74beac834f5f6c5db1c47cc1e`, head `f257e931e3aa7eaf1495470c69bc47758d89e07a`. Read the supplied final-fix brief, both implementer reports and scoped rereview instructions. Reviewed the focused source/docs/audit diff in bounded passes and inventoried the full 9,240,591-byte exact package: 548 sections, including 369 qualified-source snapshot additions, 169 new final-fix evidence additions and 10 other sections. The package also includes the preceding commit retaining `evidence/final-review.md`; that report was not treated as a new runtime change. Original reports and full referenced manifests, archive members and preservation records were checked directly. No Git commands, Go suites, race suites, live requests, credential reads, source/index/HEAD edits or subagents were used in this rereview. The sole write is this report.

**Regression evidence checked —** Read `tools/test_qualify.py:20` in full and inspected retained `regression-commands.json`, synthetic input manifests, original/current collector copies, generated metadata and decompressed outer outputs. The final original-tool run exits1 with both failures specifically on freeze-host attribution; the final current-tool run exits0 with both tests passing. The fresh test uses deliberately different freeze/executing host and toolchain values and checks version/test argv, selected environment and working directory. The parse test seeds original execution fields, changes the parser host, forbids Go invocation and checks original fields/raw bytes. Synthetic command evidence remains explicitly separate from Go runtime qualification. Earlier setup/assertion failures are retained and correctly distinguished in the report. No rerun was needed to answer a concrete doubt.

**Actual extracted checks inspected —** `final-fix/extracted/verification.json` lists only metadata regression, actual consumer qualification, consumer vet and consumer module verification, all numeric exits0. Decompressed `01.stderr.gz` confirms two Python tests pass; `02.stdout.gz` agrees with `final-consumer.command.json`; vet output is empty and module output is `all modules verified`. The real consumer log independently contains20 test-pass events, no failures and a passing package elapsed5.478s; collector elapsed18.137s. Its metadata records `macOS-27.2-arm64-arm-64bit-Mach-O`, `go version go1.27.3 darwin/arm64`, local toolchain, bundled file proxy and credential absence. The raw log's decompressed SHA256 is `79dd02f98ca217591356bb5e19b3aa66a396604c15aa5925a5d033141f5658e4`. The extracted verifier establishes a fresh initially nonexistent module cache, validates all members and checks frozen source before/after execution. This is existing single-host evidence inspected during review, not a new execution or a cross-platform claim.

**Independent read-only integrity check — PASS.** A Python hash/content inspection, with no subprocesses or writes, recomputed the current length-delimited source identity; verified every manifest member and frozen source in both archives; compared each unchanged original frozen input directly between archived original and current bytes; checked the pinned dependency ZIP; rebound each preserved snapshot to its original `before-inventory.json` entry; checked original/current raw Go log compressed/decompressed hashes, source IDs, pass counts, failures and exit fields; and checked archived checkpoint/current adjacent document hashes. Exact aggregate output:

```json
{
  "Result": "PASS",
  "CurrentInputs": 258,
  "UnchangedOriginalInputs": 256,
  "ChangedSourceInputs": ["docs/foundational/phase1g/tools/qualify.py"],
  "AddedSourceInputs": ["docs/foundational/phase1g/tools/test_qualify.py"],
  "VerifiedArchives": [
    {
      "SourceID": "43b86781f819c6bf7a957956aceaf717a5808873114e8213bf1147620bdf2ec3",
      "ArchiveSHA256": "423c1bff2aaf4e8a76a8813df6a1076977c2d8844db13e2b34919ea382732fb4",
      "Members": 499
    },
    {
      "SourceID": "ce4bbf6f9e6a0c7e94ccdfec09d97f9355b5e9bb1a6f49fcca2b17fe7256482a",
      "ArchiveSHA256": "bf2c473570a08976e5a9308cbdd895eac41afec921dcb775168b09b409abba9d",
      "Members": 613
    }
  ],
  "PreservedFilesBoundToBeforeInventory": 369,
  "ChangedOriginalFiles": [
    "README.md",
    "docs/foundational/05-decisions.md",
    "docs/foundational/phase1g/02-qualification.md",
    "docs/foundational/phase1g/04-runbook.md",
    "docs/foundational/phase1g/archive-manifest.json",
    "docs/foundational/phase1g/source-manifest.json",
    "docs/foundational/phase1g/source.tar.gz",
    "docs/foundational/phase1g/tools/qualify.py"
  ],
  "UnchangedOriginalArtifacts": 737,
  "PriorArtifactsUnchanged": 1120,
  "OldAndNewRawGoLogsHashAndCountChecked": true,
  "ArchiveDocumentSnapshotChecked": true
}
```

**Preservation and association checked —** All Go, Go test, module and fixture inputs are byte-identical to the original qualified archive; only collector metadata code changes and a focused Python regression is added to frozen inputs. The original archive/manifests, all Task4 outputs and original adjacent document bytes match their before-inventory entries under `final-fix/qualified-source-43b86781/`; live original Task4 artifacts and `evidence/final-review.md` also remain unchanged. All1,120 earlier foundational artifacts match baseline hashes. Broad default764/integration1006/race1006/consumer20/consumer-race20 passing-event logs remain source43b86781/archive423c1bff evidence. Current sourcece4bbf6f/archivebf2c4735 has focused metadata and actual consumer evidence only. The archived current report is an explicit pre-package checkpoint; the completed adjacent report and its hashes are distinguished. No old tests are relabeled, and no new broad/race qualification is inferred.

**Fix round — All findings addressed, no new Critical/Important breakage.** M1 and M2 are closed within the authorized metadata/documentation scope. The original single-host, finite-schedule runtime limits remain; final publication/status updates belong to the controller. This verdict does not authorize merge or release.
