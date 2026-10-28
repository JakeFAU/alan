# Phase1H whole-branch review — intermediate evidence addendum

This continues the same review recorded in `/tmp/alan-phase1h-final-review.md`; that stage1 report is preserved unchanged. **No new load-bearing finding. Final APPROVE/REJECT remains withheld pending root race, covering adaptive-consumer race, and final metadata/diff/publication inspection.** The newly completed normal/integration and archive/extracted evidence gates are supported by the actual records inspected below.

Frozen head remains `680be378e0104d5fcf7d045fced8123b0909f189`, base `56892cb20c4f4add4bcf786d146b0aed82fbc5d1`, SourceID `770df465af2bfc3bfa78547b6ad821427fbe69a01cd5ad284ff9030e82269912`. All 277 contributing current file hashes still match. No repository/index/HEAD edits, subagents, application tests, builds, suite reruns or provider dispatch were performed by this reviewer. This was evidence inspection, not another code pass.

## Full normal/integration evidence

Independently verified `evidence/task3/final/final-normal/command.json`, source inventory and source archive member bytes, decompressed stdout/stderr hashes, and exact source/head association. Actual command was `go test -json -count=1 -timeout=60m -tags integration ./...`, exit0, 1899.128 seconds. It reports source unchanged at770df465.

Reparsed raw Go events independently and compared the resulting record sequence against saved `records.jsonl.gz`, including package/test origin, marker, payload, wrapper source/mode and record index. Actual results are1146 test/subtest pass events,24 passing packages, no fail event,224 ADAPTIVE records,12 ADAPTIVE_BYTES and2 ADAPTIVE_ACCEPTANCE_BOUNDARY records. Thus the claimed counts do not depend only on command summaries or PASS strings.

All66 adaptive retained artifacts' compressed hashes verified and their JSON decoded. Actual record kinds include six cut/recovery pairs, two root denials, two unknown-sibling schedules, six capacity/out-of-order pairs, two primary closures, one late-transport companion, one isolated replay, and one authentic faulty application. Thirty-two process starts/exits and60 ACK negative controls are retained. Recovery records have distinct old/new PIDs and no resubmission; the recorded killed subprocesses are actual SIGKILL exits.

Independently inspected the current-source snapshots for all six process cuts, both unknown cases, all six capacity cases, the authentic faulty execution and the live closure companion. They retain the stage1 contract: first successor intent with full count6/trial6; original acknowledged source transition; immutable prior records; partial result prefix with an unfinished sibling; original failed beta/unknown gamma and spent effects; progress after held contact; original gamma0 plus forbidden10/11/31 admissions rejected by the unchanged oracle; and allowed source authorization followed by denied `scope_closed` reservation. Both direct/nested root-denial snapshots show accepted6 but only four runtime-accepted leaves, ordered rejected10/11 and failed diagnostic outcome. Both primary closures show actual open-to-cancelled lifecycle decisions, unchanged original charges and attempted late admission rejection. Isolated replay retains identical ledger digests and seven contacts before/after, no credential or adapter registry and the declared macOS deny-network subprocess policy.

Normal semantic comparison was independently recomputed against the original normal records:15 exact QUALIFICATION payloads,20 COMPOSITION/32 SUPERVISION records with their complete old/current source manifests and inner IDs validated separately, then19/31 exactly equal ordered behavior records. The six CONFORMANCE cases match the existing declared public checkpoint/model-stream domain and actual Go pass events. This is not raw whole-runtime-trace equivalence; labels/field shapes and the established projection domain retain their stated limits.

## Archive integrity and meaning

Verified `docs/foundational/phase1h/source.tar.gz` SHA256 `2e9efb6136f3de728cff2a09c3a7a3d0686375e6054577e4bb384414102d1d36`,5,258,134 bytes,309 unique safe regular members. Every member's exact size/hash matches `archive-manifest.json`; every source member matches the frozen277-input manifest. The bundled source-manifest object equals the adjacent manifest.

The32 additional members are exactly27 declared prose snapshots, the source manifest, and four pinned sqlite3 proxy members (list/info/mod/zip forv1.14.52). The zip hash matches the frozen dependency declaration. No unmanifested extra file, unsafe path, duplicate member, link or missing source input was found.

`evidence/task3/archive-context.json` accurately distinguishes immutable package-time prose snapshots from later adjacent reports. The archive was packaged while normal/consumer race were still active and final review was pending. Its older pending prose therefore does not pretend to be final qualification. The archive need not be rewritten when final generated reports complete; those reports must remain explicitly adjacent evidence with their own actual identities/status. The package includes sources/tools/fixtures/manuscript and dependency proxy, not the Go/C toolchain or all historical evidence archives.

## Actual extracted reproduction

Verified the outer `final-offline-reproduction` capture's source archive, source/head identity, raw-output hashes, exit0/259.806 seconds and source-unchanged assertion. Independently checked all six inner command stdout/stderr hashes and exits against `extracted/reproduction.json` and the outer command transcript.

The executed frozen `reproduce.py` sets a fresh nonexistent GOMODCACHE before its first command, uses the bundled `file://` proxy with checksum lookup off and credential removed, and checks source inventory after execution. The actual first compile stderr records sqlite3v1.14.52 download through that configured proxy. This is an initially empty **module cache**; the script deliberately reuses the named Go build cache and relies on the installed local toolchain. No stronger empty-build-cache, hermetic-toolchain or OS-wide network-denial claim is made for the whole reproduction. The replay subcase separately applies its explicit network-denial policy.

Actual executed commands cover root compilation, both separate consumer compilations, selected adaptive integration controls, all eight public adaptive-consumer direct/nested/capacity/paired paths, and the semantic package. The raw events contain all seven exact required focused pass paths: direct decision and partial recovery, capacity1 causal progress, failed-plus-unknown, late-transport companion, isolated replay and authentic faulty-program rejection. All eight exact public-consumer paths passed. There are no failure events in these three JSON streams. The18 retained adaptive artifact hashes from the extracted execution also verify.

The extracted semantic record counts and behavior comparisons were independently checked at15/20/32/6 with the same strict provenance boundary and declared conformance projection. Reproduction is therefore supported as the **declared focused extracted-source reproduction**, not an extracted full default/integration/race suite. It does not exercise every direct/nested recovery variant or substitute for pending whole working-tree race qualification.

## Remaining final gates

- Complete root race at the frozen source, including its actual process/fault instrumentation, all required cases, raw hashes and semantic comparisons.
- Complete covering adaptive-consumer race; preserve the original15-minute failed command and distinguish its replacement's actual coverage.
- Inspect final generated qualification/index/report/review files and all changes after680be37. Contributing source must remain frozen or receive justified fresh qualification.
- Inspect final draft PR/base/head and actual available CI status; no merge or release.

The previously stated research limits and original tests-first chronology limitation remain unchanged. No new code fix or qualification rerun is requested by this addendum. Reviewer inspection scripts/results: `/tmp/alan-phase1h-final-stage2-audit.py`, `/tmp/alan-phase1h-final-stage2-audit.json`, `/tmp/alan-phase1h-final-stage2-snapshots.py`, `/tmp/alan-phase1h-final-stage2-snapshots.json`.
