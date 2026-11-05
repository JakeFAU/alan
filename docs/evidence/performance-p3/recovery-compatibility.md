# P3 bounded recovery and compatibility qualification

Task 4 passed its declared bounded routes against production baseline
`926a899b651fccc48be52a054612a92afcf1ddfd` (tree
`a58fe3b0ef0b9a7525c46c4f0a662774e6d4221d`) and candidate
`49df59f1f767dd63fec5e238bf565d023261a75b` (tree
`b86590205169e661a1bf14ad5d72701b4e0a684a`). No production code, dependency,
format, scheduler, approval rule or hardened competing-reader test changed in
this task. Performance acceptance and whole-repository gates remain later work.

Fresh raw evidence is rooted at
`/private/tmp/alan-p3-j216kxoz/task4/qualification`; the enclosing Task 4
`raw-index.json` also includes initial setup failures. These local artifacts,
not prior P2/convergence campaigns, support the results below. The frozen
`route-manifest.json`, complete Git-export source inventories, exact binary
hashes, command argv/cwd/whitelisted environment/exits, and stdout/stderr remain
in that root. Go was `go1.27.3 darwin/arm64`, CGO enabled, GOMAXPROCS=4,
GOGC=100, GOMEMLIMIT=off, GOWORK/GOPROXY/GOSUMDB=off and GOTOOLCHAIN=local.
Setup used explicit Python 3.14; legacy subprocess tools used their recorded
system Python executable. No environment-wide capture, live provider or timing
campaign was used.

## Observed routes

| Route | Observed result and retained evidence |
| --- | --- |
| Direct same-seed success/refusal3 × ordinary/reopen | Four final complete-copy pairs, exact 42-table typed schema/cell equality plus full inspection, History, Replay and SeriesView. Ordinary reports have 18 public cuts; reopen reports have 25. `direct-complete/`, `gates/direct-complete-*`. |
| Controlled HTTP same-seed success8 × ordinary/reopen | Two final complete-copy pairs, exact 42-table evidence, public cuts, returned RunReports and both contact journals. 18/26 cuts, one fixed endpoint within the two-source campaign, exact original namespace/IDs/epochs/receipts. `http-complete/`. |
| Comparator challenges | 38 original tool/guard checks, 53 matching-report checks, and five explicit matching wrong operation/capability/effect/protocol/endpoint mutants all produced their intended outcomes. There were 85 structured semantic rejections, three comparator positives and eight clone/interface refusals. Import/setup failures do not count as negative proof. |
| Genuine weather processes | Each source: all ten series cuts, seven selected continuation cuts, four selected recovery cuts and public eight-read/original-lowering/corruption checks; 37 PASS events and zero skips. Internal cuts validate ACK identity; accepted/lost cuts use independent endpoint acceptance. Both validate SIGKILL and reap before fresh-owner execution. |
| Retained process corpus | 31 new archives per source: ten copies immediately after reap and before readers, plus the existing reference/stopped/final captures. All 62 contain main, WAL and SHM, each with source-before/copy/source-after SHA-256 agreement. Independent public consumer binaries/modules are retained. |
| Exact archive readers | All 62 archives passed baseline/candidate complete-copy generic reads: full inspection, every History page, Replay, one Open/Close renewal and exact typed SQL equality. These readers have no adapter registry and do not Run, approve or dispatch. |
| Actual opposite-build resume | Four final nested-CLI cases: both writer directions × staged/pre-commit and committed/post-commit. Original ACK, SIGKILL/-9 reap, sealed stopped DB, complete independent copies, exact returned RunReport and generic-reader/typed durable equality. `cross-build-final/`; this local nested fixture is separate from weather and authority. |
| Request authority | Both sources, ordinary and race: 30 top-level tests / 104 PASS events each, zero skips. Own/delegated use, original request/generation mismatch, revocation before a fresh gate, retained grant across reopen, and immediate-selection refusal remain checked separately. Public committed-control tests also pass on both sources. |
| Competing reader | Byte-identical `TestCompetingReadCannotSettleAdopterDebt`: both sources ordinary/race, three PASS events per invocation, no failures or assertion changes. File SHA-256 `7b193fe538e273e52b3941738868c3580d400bb340f6dae460f77b16fb6dec07`. |
| External public consumer | Four tests / 16 PASS events ordinary and race, zero skips: eight real operations, canonical lowering and fresh format9/11 stores (bounds1/8/32/256). Create-only larger bounds are compatibility checks, not timings. |
| Historical formats/readers | Genuine archived format4 lifecycle histories and format9 weather remain byte-preserving under readers. Genuine retained conflict promotes11→12; baseline generic readers accept complete isolated11/12 copies. Candidate retains eight selections/completions/spent for both, with format12 preserving one unresolved conflict. |
| Pre-authority reader | Fresh source `af6943a5805a9a6c0e8fe77f804264f4765d07ec`, tree `5d1f64029c7fd764ddd1941bc77b831beb3bdcb3`, built CLI SHA-256 `7be9c64746b9da789f2f1bbe81972f6480720f6f615b6813236b7c0495873dbe`. Both format13/14 old-reader inspect/replay rejection subtests execute in every authority invocation; no opt-in skip remains. |

The series raw stops distinguish paid reservation before admission from
admitted work with missing results; only the latter is reported unresolved.
Accepted/lost responses retain expenditure and original uncertainty. Retained
results finish without reissue. Later observations require separate public
approval; absent fourth approval is not supplied by recovery. Earlier original
effects remain equal. The exact retained-request boundary is additionally
exercised by `TestWeatherProcessRecovery/after_request_commit`; that fixture's
temporary database is not claimed as one of the 62 series archives.

The archive overlay changes only evidence capture: complete-sidecar copying,
before-reader stopped copies and independent consumer retention. Original and
modified test hashes and full patches are retained under `weather-overlay-*`.
Runtime paths, crash assertions and all semantic checks remain unchanged.

## Preserved apparatus findings and limits

- Initial setup retained a Python3.9 unsupported `tarfile` keyword error and a
  macOS Python cache permission denial. Explicit installed Python3.14 and a
  scratch cache resolved setup; no dependency was installed.
- Initial baseline weather failed before cases at a sandbox loopback bind
  denial (exit2). That gate and its consumer binary remain. The declared cases
  then ran with authorized loopback access, distinct gate names and fresh output.
- Original P1/P2 seed and mutant helpers copied only main DB bytes despite
  an extant empty WAL and SHM sidecar. Initial `direct/` execution and `http/` results remain
  labeled with that custody limitation. Fresh `direct-complete/` and
  `http-complete/` runs use emitted copy-only patches; original tools and exact
  comparator semantics remain unchanged. The 62 archive comparisons already
  used complete copies and did not need replacement.
- The first cross-build wrapper imported the original Python controller and
  created one `.pyc` inside the candidate export. All original source bytes
  stayed identical, but the full inventory was temporarily different. The
  failed audit and exact cache/path/hash are retained, and the cache was moved
  to `apparatus-import-cache/`. A focused real import test qualifies bytecode
  suppression; the final wrapper's four cross-build cases ran afresh under
  `cross-build-final/`. `cross-build-final-bridge.json` records the old/new
  wrapper hashes and patch. Initial cases remain separate evidence.
- Two later audit-script errors used the wrong log stream for the retained
  bind denial and the wrong filename suffix for matching-proof output. Both
  failures, diagnoses and audit revisions remain. Corrected `audit-v4.py`
  passes without altering any runtime output or comparator result.

The final audit verifies 607 baseline, 623 candidate and 389 old-reader source
files; all original export inventories match again. It verifies 17 root binary
identities, 219 complete-copy proofs, sealed originals and all 348 recorded gate
outcomes (including the explicitly retained bind denial). Selected tests have
zero skips. Final wrapper tests cover all sidecars, an actual source mutation
during copying, failed-child retention/overwrite refusal, and a real import
without bytecode writes.

These results do not establish authority-plus-series support, arbitrary hostile
writers, distributed recovery, or broader model agreement. The existing
immediate-selection refusal and format13/14 restrictions remain. Independent
weather executions have different namespaces/endpoints; their causal assertions
are not represented as one cross-build authority/weather history. Only declared
same-seed/same-history pairs receive exact equality claims. Generic-reader
comparisons are not physical execution. No B256 timing or speedup claim follows.

## Reproduction

Use a fresh absolute `OUT`, an authorized local-loopback execution environment,
and Python3.12+ (the actual setup used `/opt/homebrew/bin/python3.14`). The fixed
source commits are in `qualification/run.py`; `--repo` supplies the local Git
repository containing them. Run serially, away from timing campaigns:

```sh
python3.14 qualification/run.py freeze --root "$OUT" --repo "$REPO"
python3.14 qualification/run.py build --root "$OUT"
python3.14 qualification/run.py weather --root "$OUT"
python3.14 qualification/run.py archives --root "$OUT"
python3.14 qualification/complete_campaign.py --root "$OUT"
python3.14 qualification/run.py authority --root "$OUT"
python3.14 qualification/cross_build.py --root "$OUT"
python3.14 qualification/run.py external --root "$OUT"
python3.14 -m unittest discover -s qualification -v
python3.14 qualification/run.py seal --root "$OUT"
```

The final reusable wrapper omits the superseded main-only direct/HTTP routes.
Exact executed original and revised controller sources remain in the raw root;
final wrapper changes are explicitly bridged rather than relabeling old runs.
An existing case/gate destination is refused. Raw evidence is local, not a
portable committed dataset; source paths, identities and limitations above are
part of the qualification boundary. No push, merge or release occurs here.
