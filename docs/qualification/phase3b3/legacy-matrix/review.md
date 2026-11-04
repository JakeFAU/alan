# Task 2 finite legacy 3–12 qualification review

**Verdict: APPROVE** for the finite compatibility qualification described in the report addendum and `/tmp/alan-phase3b3-legacy-matrix`. No substantive finding. This review covers the ten old-created retained histories and their specified readers/comparisons; it does not establish universal legacy equivalence or extend the authority primitive's previously approved scope.

## Source identity

I read the qualification addendum, complete overlay harness and comparator, command/build records, reader matrix, fixture summaries and hash manifest. I independently checked:

- All **422** evidence-manifest members against `sha256.json`: zero mismatches, including harness, binaries, exports, logs and original/reader database bytes. Hashing bytes did not open a SQLite database.
- All **294** entries of the approved `task-2-fix1-source.json` against both current runtime files and the qualification's current-source manifest: zero mismatches. The newer manifest has two additional nested performance-module go.mod/go.sum entries; it does not replace or alter any approved source entry.
- All **280** original Go/module files recorded by `old-source-verification.json` against an independently obtained in-memory `git archive af6943a5805a9a6c0e8fe77f804264f4765d07ec` and the old source extraction: zero mismatches. The archive digest also matches the recorded digest.
- Both overlay mappings point to the same harness bytes, SHA-256 `5d6e72a782b145cc3c6cb2104fa25042fa3d2e1785a80b497e41810ffb525550`. Only `old.test` ran fixture creation. The recorded old/current binary hashes match their files. Recorded toolchain is Go 1.27.3 darwin/arm64.

The harness compiles as an added integration test and uses existing private test helpers. This is suitable for finite retained-state qualification and is not presented as a new public execution API. No reviewed production/test source was changed for this qualification.

## Genuine feature histories

The harness asserts `storedSchemaNumber`, creates each feature through original provision/writer APIs, and performs no schema-number SQL mutation. I independently inspected exported raw-table metadata and typed feature fields for every format; versions 3 through 12 agree. Exported counts and continuation evidence match `fixture-summary.json`.

| Formats | Retained evidence actually exercised |
| --- | --- |
| 3–8 | One original external attempt and one target-missing lifecycle decision each; composition variants 4/5/7/8 also retain a genuine later invocation. |
| 5/8 | Reconciliation profile enabled; **zero receipt-read ordinals**. These fixtures do not qualify reconciliation execution. |
| 6/7/8 | One synthetic first external receipt and one conflicting receipt through the real promotion writer. These are not physical-contact witnesses. |
| 9/10 | One adapter attempt/result and one dispatch admission, produced through ordinary Run and the in-memory callback. Format 10 also has one conflicting receipt. |
| 11/12 | Two adapter attempts/results and dispatch admissions, plus retained continuation approval, actual approval selection and successor operation. Format 12 additionally has a conflicting predecessor result. |

The initial setup failure is accurately distinguished from semantic RED. Comparing the preserved failed and corrected harness shows exactly one change: `schemaNumber` to `storedSchemaNumber` in the version assertion. The former reports underlying protocol versions, so it was the wrong assertion for formats 6–12. No expected storage version or production behavior was weakened to obtain success.

## Reader isolation, outcomes and comparison

Each fixture is closed before copying into originals. `matrixCopy` copies every present main/WAL/SHM member and uses exclusive destination creation. Every `(reader, version, API)` has its own directory and database copy; neither Open nor another reader mutates the database used by a different primary call. Open's follow-up exports intentionally inspect the same post-Open copy. No archived original is passed to a SQLite reader by this harness.

I independently verified the complete current original member set against both before/after manifests: **ten main files, no sidecars, identical hashes**. I did not open any database, original or copy, through SQLite during this review.

The comparator labels outcomes `pass` from completed export generation rather than parsing test results, so I independently checked the execution logs: old and current each contain exactly **40 distinct passing primary API subtests**, no failures, and the corresponding command records have actual exit 0. Together these support **80 primary calls** across Replay, Inspect, InspectExecution and Open. Open's follow-up Replay/raw/typed exports are additional calls, not extra primary cases.

I independently compared all **60 pairs** of old/current exports as raw bytes and checked their recorded hashes. Every pair is identical; no field removal or value normalization was used. I also recomputed each pre-/post-Open typed comparison from the JSON exports: the only changes are +1 epoch for each still-open run. Formats 3/6 renew two runs each; 4/5/7/8 renew five each; 9–12 have no open runs and their typed snapshots are unchanged. Typed history exports are complete because the harness fails if `History.More` is true. The zero-open cases should not be described as fresh-activation exercises.

## Commands and claims

`commands.json` distinguishes failed setup from final execution. Both corrected `go test -tags=integration -overlay ... -c ... ./internal/executor` builds, old fixture creation, both reader runs, and the JSON comparator record exit 0. I verified the corresponding retained logs/hashes rather than rerunning those tests.

The separately recorded full ordinary command is `go test -count=1 ./...`, with the documented local toolchain/cache/offline-module environment and frozen-old-reader variable. Its retained log reports 18 passing packages and three packages with no test files; the command record states exit 0 with authorized loopback access. This is full **ordinary** module verification. Integration and focused race evidence remain the separately scoped round-1 runs; no full-module integration or full-module race claim follows here.

The report and README correctly limit the result to these ten finite histories. They do not claim exhaustive reconciliation/continuation behavior, arbitrary old programs, universal schema compatibility, physical-effect correspondence, or a distributed recovery theorem. The previously established inert-name regression and metadata-rejection controls remain separate evidence. Independent model/process-cut correspondence, full requester result custody and manuscript reconciliation remain outstanding phase obligations.

No source edits, subagents, commits, archived SQLite opens, or redundant suite runs were performed in this review. Only this review document was written. The qualification can be packaged and bound to the primitive source freeze within the stated limits.
