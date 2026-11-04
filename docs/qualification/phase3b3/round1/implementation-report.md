# Task 2 implementation report — candidate under verification

Updated 2026-11-04. This replaces the obsolete pre-production literal-root obstruction report, preserved separately under `docs/foundational/phase3b3/evidence/rejected-contract-v1`.

## Question and current result

Can committed ordinary lifecycle decisions provide exact, non-redelegable, immediate-next-reduction effect authority, with fresh physical admission evidence, using existing composition ordering and resource accounting? The uncommitted candidate in `/private/tmp/alan-phase3b3-runtime`, branch `codex/phase3b3-request-delegation`, base HEAD `6eca5f81faf41165e7fa77819537e49fafcd0249`, implements the frozen contract v2. This is a candidate for independent review, not an approved or fully qualified result. No commits were made by this completion agent.

## Candidate scope

New `request_authority.go` implements typed namespace-free logical root claims/selectors, immutable lifecycle decisions and historical exact tuple/request selection checks. New `request_authority_admission.go` implements primary composition facts and per-original ordinals. Existing lifecycle, program, external/adapter, dispatch, replay, inspection, composition/version and runner boundary files integrate that protocol. Formats 13/14 opt in; 13→14 retains receipt conflicts. Public aliases are in `runtime/runtime.go`; public consumer evidence is `runtime/request_authority_test.go`. No new authority table, scheduler, accounting ledger, compiler convenience API, or generic adapter retry is introduced.

## Inspected retained test evidence (prior implementer)

Parsed complete JSON lines in `/tmp/alan-phase3b3-final-normal.jsonl`: three package passes (`internal/executor`, `runtime`, `author`), 735 test/subtest pass events, no failure or skip events. `/tmp/alan-phase3b3-final-integration.jsonl`: the same three package passes, 1544 pass events, no failure or skip. `/tmp/alan-phase3b3-final-race.jsonl`: the same three package passes, 63 pass events, no failure or skip. These are event counts including subtests, not independent experiment counts. The JSON logs do not independently establish the original command line or process exit status; fresh bounded checks and exact commands will be recorded below. Empty final Staticcheck logs alone do not establish successful exit status.

Inspected source includes positive ordinary two-agent Run, public committed lifecycle control, exact and crossed adapter resource/operation cases, generic unknown/no-retry, revocation timing, sticky refusal, wrong root/holder/request/generation/sequence, no fallback/redelegation, primary gate corruption, missing original debit, result-without-gate, feature masking and frozen old-reader rejection tests. Full coverage against every contract attack remains subject to independent review.

`/tmp/alan-phase3b3-debit-red.log` records `TestRequestAuthorityGateRequiresOriginalDebit` failing with “gate accepted reservation without original debit”; corresponding green log records that test passing. The candidate gate now checks reservation and scoped debit. Original failing production source provenance and permanent evidence retention remain to be completed; a reconstruction must be labeled explicitly.

## Pending checks and unresolved concern

Fresh full scoped verification completed before adding the deliberately failing compatibility regression; exact commands and results follow. Frozen reader invocation explicitly set `ALAN_PHASE3B3_OLD_READER=/tmp/alan-phase3b3-old-reader/alan` so compatibility tests cannot silently skip. Creation/open/replay/inspection behavior for old formats 3–12 needs its exact command scope stated, not inferred merely from schema DDL.

The candidate rejects new authority operation names in legacy literal programs and at Commit/loadIntent. Legacy programs can also compute lifecycle operation names through Ref. `readIntents` in replay.go decodes those rows directly. A disposable Store created by frozen old public APIs established this is an inert historical prefix, not an authority bypass: both old and candidate Replay/InspectExecution/Open succeed; old LoadLifecycleRequest refuses unknown lifecycle operation and candidate refuses unsupported version. Adding a bulk validator would reject historically valid inert data. Parent requested a compatibility correction, pending independent contract review. No production edit has been made.

## Boundaries and remaining obligations

This runtime implements the authority-safe staged grant-ready subset corresponding to S2/LateGrant before first permission decision. It does not realize S2/Explicit atomic request-plus-grant Commit. Endpoint deduplication and eventual response assumptions remain necessary for progress. Generic unknown adapters retain unknown/unpaid state. Gate retention cannot prove a physical call happened or uniquely identify a returning exposure; independent contact and acknowledged boundary traces remain necessary. Real SIGKILL correspondence and independent research checker/manuscript reconciliation are later qualification work, not established by these primitive tests. Independent authority/minimality and attenuation/request-binding review is pending.


## Fresh completion-agent verification

All commands below ran from `/private/tmp/alan-phase3b3-runtime` with `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off`. Tests additionally set `ALAN_PHASE3B3_OLD_READER=/tmp/alan-phase3b3-old-reader/alan`; Staticcheck set `STATICCHECK_CACHE=/tmp/alan-phase3b3-staticcheck-cache`.

- `go test -json ./internal/executor ./runtime ./author -count=1`: exit 0, 294 top-level tests /735 pass events, no failures/skips.
- `go test -tags integration -json ./internal/executor ./runtime ./author -count=1`: exit 0, 381 top-level tests /1544 pass events, no failures/skips.
- `go test -race -tags integration -json ./internal/executor ./runtime ./author -run '^TestRequestAuthority' -count=1`: exit 0, 25 top-level tests /63 pass events, no failures/skips. The author package has no tests matching the filter.
- `staticcheck ./internal/executor ./runtime ./author` and `staticcheck -tags integration ./internal/executor ./runtime ./author`: each exit 0, empty output.
- `git diff --check`: exit 0 after adding the compatibility regression.

These are the three named packages, not `./...`; race coverage is focused, not the whole suite. Logs are retained in runtime `docs/qualification/phase3b3/task2-completion/{normal-loopback,integration-loopback,race-loopback}.jsonl` and corresponding Staticcheck logs. Initial sandbox runs failed at local httptest listener creation; authorized loopback runs above completed. Initial Staticcheck attempts failed writing its default macOS cache; explicit STATICCHECK_CACHE fixed the environment without source changes. Environmental failures are not semantic RED.

Existing integration `TestLifecycleRecoveryAcrossFormats` constructs and checks versions3–8 with real writer/promotion, Replay/inspection/Open. Existing dispatch-admission tests cover9/10, and continuation tests11/12. Their full integration-suite execution passed; this is regression coverage across those profiles, not a new exhaustive snapshot-equivalence proof of every old-format behavior. New13/14 tests explicitly exercised conflict promotion, old inspect/replay rejection, feature masking, and holder/root distinction without skips.

## Confirmed compatibility RED and pending contract correction

The four new semantic cases are `TestRequestAuthorityLegacyUnknownOperationCompatibility/{DelegateAuthority,RevokeAuthority}/{literal,ref}` in `internal/executor/request_authority_legacy_test.go`. The test uses ordinary legacy Create/Commit, expects the prior unknown-operation loader refusal and verifies Replay/InspectExecution/Open. With identical test SHA256 `38fd4713a706a89bdbd051e48b1a36f54e2046ad8b7665ca2bf33a073a2adc47`:

- Frozen af6943a source: all four subtests pass, command exit0.
- Candidate contract-v2 source: all four fail, command exit1. Literal cases fail Create; Ref cases fail Commit with unsupported version.

Exact command: `go test ./internal/executor -run '^TestRequestAuthorityLegacyUnknownOperationCompatibility$' -count=1 -v` with the environment above. Reproduction runner, exact exit records, logs, test and public probe source, closed old-created Store and SHA256 manifest are in runtime `docs/qualification/phase3b3/task2-completion/legacy-compatibility/`. The exact pre-regression candidate Go/module snapshot is `pre-compatibility-source.tar.gz` (SHA256 `aeefc0d9df7e0ed277550048dc1079aae1cf2235b7120d1f43e1097cb49bcc3f`). Parent separately froze candidate-v2 for independent review. The snapshot predates the new RED test.

Proposed narrow correction, awaiting independent review: in legacy formats preserve formerly unknown literal/dynamic lifecycle-name retention and the old unknown-operation loader refusal; do not recognize Delegate/Revoke semantics there. Typed authority fields/results/admission facts remain opt-in. Do not add a bulk-read validator that rejects legitimate old inert prefixes. This is a correction to v2's overbroad prohibition of the operation strings, not permission to mint authority in old formats. The current worktree deliberately contains the failing regression, so it must not be described as currently all green.

## Additional scope clarification

Complete-composition direct `DispatchExternal` continues to return `ErrDispatchMode`; the physical path qualified here is ordinary Run. No broadening of direct APIs was made merely to satisfy overbroad contract language. The `authority.admission` boundary is after durable commit/lock release and before worker release, with exact ordinal/actual activation; original external Request.Epoch remains unchanged.

Original debit RED/green logs are permanently retained in runtime `docs/qualification/phase3b3/task2-completion`. The original failed production source was not preserved. `debit-reconstructed-pre-fix.go.txt` is explicitly a reconstruction of the captured candidate with its reservation/debit block removed; it has not been executed and is not presented as original rejected bytes. The exact candidate test remains available in the source snapshot.

Completion-agent changes are limited to the compatibility regression, this report, and qualification evidence. No production edits, commits, baseline edits or manuscript edits were made. Contract correction, subsequent green rerun, independent authority/attenuation review, real crash/contact qualification and manuscript reconciliation remain pending.

## Round 1 corrections after independent REJECT — 2026-11-04

This append supersedes the pending correction/verification statements above; it does not replace or erase either review. The independently rejected candidate-v2 and both review reports remain preserved. Corrected source is uncommitted on `codex/phase3b3-request-delegation` in `/private/tmp/alan-phase3b3-runtime`, still based on HEAD `6eca5f81faf41165e7fa77819537e49fafcd0249`. Governing contract is frozen v3 at research `360cae66`, `02-contract.md` SHA-256 `f19d9c13f84c17496fabef91cb7575cf52972ff228c0b69e063cdc1c99100e12`. No new wire types or feature scope were introduced. This is a tested correction candidate requiring both independent reviewers to re-review; their REJECT verdicts are not self-cleared by these results.

### Exact implementation delta

Five production files changed relative to candidate-v2:

- `internal/executor/request_authority.go`: program/intent validation continues to forbid typed authority selectors outside the feature, but no longer rejects formerly inert lifecycle payload names there. Authority claim parsing is feature-gated. Neither literal nor Ref payload text gains authority semantics in legacy formats.
- `internal/executor/lifecycle.go`: loading DelegateAuthority/RevokeAuthority in a legacy Store returns the original `unknown lifecycle operation` refusal. Opt-in recognition and committed request comparison remain intact.
- `internal/executor/inspection_effects.go`: the legacy branch rejects either nested ExternalAttempt.Authority or AdapterAttempt.Authority, as well as Intent.Authority. Clean legacy attempts continue to validate.
- `internal/executor/external.go`: extracted the existing offline reconstruction predicate into `validateExternalAttempt`, with the same full attempt comparison. It rebuilds external Request from the actual retained intent, original namespace-derived key, original authorization ID and original authorization epoch. Offline receipt validation still runs afterward.
- `internal/executor/request_authority_admission.go`: inside the existing owned admission transaction, before recording or releasing a gate, reloads and validates the original external attempt through that predicate, or the adapter attempt through existing `validateAdapterAttempt`. The latter reconstructs every derivable Request field from the retained program/schema/inputs/assets, compares the complete request, and verifies prepared digest/shape/bounds. No independent reconstruction of arbitrary opaque adapter Wire, Endpoint, Model or deployment configuration is claimed; those remain trusted adapter ABI output. Existing selected-witness, historical-prefix, original reservation/debit, current source/root and ordinal checks remain after original-request validation. No extra charge, refund, retry or scheduler is introduced.

Four test files changed/added in this round: `request_authority_legacy_test.go`, new `request_authority_binding_test.go`, new integration-tagged `request_authority_binding_adapter_test.go`, and `request_authority_test.go`. The first retains the original four literal/Ref compatibility cases and adds clean/malformed external controls. Adapter controls independently check the other nested witness. Replay, InspectExecution and Open are each exercised for clean and malformed legacy fixtures. The binding tests adapt the independent reviewer reproductions, coherently rewrite the attempt and reservation/fact digests, and require gate refusal. They cover all eight external Request fields and all eight adapter Request fields, with the prepared digest coherently recomputed for adapters. The independent equal-payload/different-original-request test is also retained. Reopen coverage now explicitly checks immutable wire Request identity and original Request.Epoch against a different fresh activation epoch.

These malformed owned-Store probes perform no external contact or adapter Execute. They test erroneous gate release, not an observed unauthorized application. The non-idempotent profile mutation also checks the unchanged suppression of retry preparation before explicitly probing the gate. The ordinary positive Run test remains the earlier local-loopback witness, with the previously documented empty-ack/result-custody limitations.

### Preserved RED and development checks

All new evidence is under `/tmp/alan-phase3b3-round1/`; the parent owns permanent archive packaging. `commands.json` records exact command lines, environment, cwd, HEAD and observed process exits; `sha256.json` binds logs, overlay, test snapshots and final changed-source copies. No archived original SQLite file was opened, and no runtime archive directory was modified in this round.

Before any correction, `go test -tags=integration -count=1 -v ./internal/executor -run '^TestRequestAuthority(Legacy|Substituted)'` exited **1** (`red.log`). It reproduced all three findings: four inert-name failures; both nested witness inspection holes with Replay/Open already rejecting; external delta 1→999 and adapter increment→double each releasing a gate while Replay/inspection rejected. Exact adapted tests and pre-fix authority/lifecycle/inspection source are in `red-source/`; the complete original candidate-v2 archive remains the authoritative full pre-fix snapshot, including external.go.

After the legacy fixes, `go test -tags=integration -count=1 -v ./internal/executor -run '^TestRequestAuthorityLegacy'` exited **0** (`legacy-green.log`). After original-request validation, the first command above exited **0** (`focused-green.log`).

Expanded mutations were run with `go test -overlay /tmp/alan-phase3b3-round1/binding-red-overlay.json -tags=integration -count=1 -v ./internal/executor -run '^TestRequestAuthoritySubstituted'`. The overlay supplies exact saved pre-fix admission source, with other source left at the corrected candidate. Initial `binding-expanded-red.log` exited **1** with 15 substantive gate failures and one fixture-assumption failure: changing profile to non-idempotent correctly returns no prepared retry worker. The ensuing focused run, misleadingly named `authority-green.log`, also exited **1** solely on that fixture assumption; it is not reported as green. Both original test versions/logs are retained. Correcting only that test to expect nil work for non-idempotent and then probe the gate yielded `binding-expanded-red-corrected-fixture.log`, exit **1**, with all **16** field mutations substantively releasing a gate under the pre-fix overlay. `expanded-red-tests/` and `expanded-red-corrected-tests/` distinguish those exact source versions. No production correction was needed for this fixture issue.

### Final executed verification

Commands ran from `/private/tmp/alan-phase3b3-runtime` with `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off`. Tests set `ALAN_PHASE3B3_OLD_READER=/tmp/alan-phase3b3-old-reader/alan`. Local-loopback suites used authorized sandbox escalation. Staticcheck set `STATICCHECK_CACHE=/tmp/alan-phase3b3-staticcheck`.

| Command | Actual exit | Evidence and scope |
| --- | --- | --- |
| `go test -json -count=1 ./internal/executor ./runtime ./author` | 0 | `normal.jsonl`: 298 top-level tests, 759 test/subtest pass events, 3 package passes |
| `go test -json -tags=integration -count=1 ./internal/executor ./runtime ./author` | 0 | `integration.jsonl`: 387 top-level tests, 1,586 test/subtest pass events, 3 package passes |
| `go test -json -race -tags=integration -run '^TestRequestAuthority' -count=1 ./internal/executor ./runtime ./author` | 0 | `race.jsonl`: 31 top-level tests, 105 test/subtest pass events, 3 package passes; author has no matching tests |
| `staticcheck ./internal/executor ./runtime ./author` | 0 | `staticcheck.log`, empty output |
| `staticcheck -tags=integration ./internal/executor ./runtime ./author` | 0 | `staticcheck-integration.log`, empty output |
| `git diff --check` | 0 | Empty output |
| `gofmt -l` over the nine round-1 source/test paths above | 0 | Empty output |

All three final JSONL streams contain no failure or skip events. Counts include subtests and are not independent experiments. Normal/integration cover the three named packages, not the entire module; race is focused. The final suite includes all corrected legacy and binding cases, existing revocation/history/debit checks, same-Store reopen, real ordinary Run, format masking/conflict promotion and frozen-old-reader rejection. These bounded tests and shared runtime validation are not an independent authority model or a proof of unrestricted compatibility.

No commits, subagents, baseline-file edits, dependencies, external paid services or manuscript edits were made in this round. Source is stable for re-review. Remaining obligations are both reviewers' independent re-review, Task 3/4 correspondence and process-cut qualification, full original requester result custody, precise legacy snapshot qualification and manuscript reconciliation. Direct composition DispatchExternal still refuses with ErrDispatchMode; no unsupported physical route was added. Historical valid permissions/results remain checked at their original prefixes, not against today's revoked set.

## Qualification addendum — legacy formats 3–12 and full ordinary module

After both independent round-1 re-reviews returned APPROVE, performed only the requested missing qualification. No production or test source in either runtime checkout was changed. The approved nine round-1 source/test paths still byte-match `/tmp/alan-phase3b3-round1/final-source`; runtime HEAD remains `6eca5f81faf41165e7fa77819537e49fafcd0249` with the approved uncommitted candidate. No commits or subagents.

Evidence is `/tmp/alan-phase3b3-legacy-matrix/`. One auditable overlay source, `legacy_matrix_test.go.txt`, SHA-256 `5d6e72a782b145cc3c6cb2104fa25042fa3d2e1785a80b497e41810ffb525550`, was compiled unchanged against frozen af6943a and current source with Go 1.27.3 and cached dependencies. `old-source-verification.json` verifies all 280 original Go/module files against `git archive af6943a5805a9a6c0e8fe77f804264f4765d07ec`; the old extraction's separate nonproduction probe did not alter those files. Binary hashes are old `3051718d1fd4c4a57f53a71102805cef7bfb09a83cb1f67eee1971eb1e61350c` and current `f2efd3d648c41181dfa5b6d396fe92d80d5d5cecb9c5ec91560aff582613e956`. `build.json`, `current-source.json`, `commands.json` and `sha256.json` retain identities and all commands/exits.

Only old code created the ten fixtures. Formats 3–8 adapt the valid original-writer path in `lifecycle_recovery_formats_test.go`: real committed external intent/authorization/reservation, target-missing lifecycle decision, and a later invocation for composition formats. Formats 5/8 enable reconciliation but perform no receipt-read operation. Formats 6/7/8 retain a synthetic first external receipt plus a rejected conflicting receipt through the real promotion writer. Formats 9/10 reuse dispatch-admission `gateStore` and ordinary Run with an in-memory callback, with a real conflicting-result submission for 10. Formats 11/12 adapt the existing continuation promotion fixture, retaining actual approval, selection and successor operation; 12 retains a conflicting predecessor result. No metadata-number SQL rewriting or physical endpoint claim is involved.

The initial harness called `schemaNumber` to assert storage format; that helper projects 6–12 onto their underlying protocol versions. The setup therefore exited 1 with seven format assertion failures. Exact failed source, binaries, logs and the first three closed fixtures remain under `setup-protocol-version-error/`. The correction is solely `schemaNumber`→`storedSchemaNumber` in the overlay assertion. Both corrected builds and old fixture creation then exited 0. This is a fixture setup error, not semantic RED or a production correction.

Every original was closed before archival copying. Old/current Replay, Inspect, InspectExecution and Open each received a separate complete copy, copying every present database/WAL/SHM member. The final ten closed originals had only the main database files; their complete member/hash sets before and after are identical. No archive original was opened by SQLite, even read-only. Typed exports include full retained history; Open copies export post-Open Replay, raw inspection and typed inspection.

All **80 primary reader calls** passed (ten formats × four APIs × two versions). All **60 paired JSON exports** are byte-identical, with no field removal or value normalization. `reader-matrix.json` records each outcome/export hash; `differences.json` is empty. `compare.py` additionally verifies that each post-Open typed export differs from its original only by the expected +1 epoch for each still-open logical run. `fixture-summary.json` records actual storage features, history/table counts, continuation evidence and renewed runs. This establishes finite compatibility on these ten retained histories, not universal old semantics or exhaustive behavior of every feature in formats 3–12.

Exact build commands, executed in old/current source respectively, both exit 0:

```text
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off go test -tags=integration -overlay /tmp/alan-phase3b3-legacy-matrix/old-overlay.json -c -o /tmp/alan-phase3b3-legacy-matrix/old.test ./internal/executor
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off go test -tags=integration -overlay /tmp/alan-phase3b3-legacy-matrix/current-overlay.json -c -o /tmp/alan-phase3b3-legacy-matrix/current.test ./internal/executor
```

The old binary ran `-test.run '^TestPhase3B3LegacyMatrix$' -test.count=1 -test.v` with `ALAN_LEGACY_MATRIX_ROOT=/tmp/alan-phase3b3-legacy-matrix ALAN_LEGACY_MATRIX_MODE=create`, exit 0. Each binary then used the same flags with `ALAN_LEGACY_MATRIX_MODE=read` and `ALAN_LEGACY_MATRIX_READER=old` or `current`, each exit 0. Logs are `create-old.log`, `read-old.log` and `read-current.log`. `python3 /tmp/alan-phase3b3-legacy-matrix/compare.py` exited 0. No production amendment was needed for these comparisons.

Finally, from the stable current checkout, `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off ALAN_PHASE3B3_OLD_READER=/tmp/alan-phase3b3-old-reader/alan go test -count=1 ./...` exited **0** using authorized local-loopback escalation. Output is `full-ordinary-module.log` in the matrix evidence directory. This is the entire ordinary module suite; integration and focused race results remain the separately recorded round-1 runs. Parent owns archive packaging and review of this qualification addendum. Independent model/process-cut correspondence, full requester result custody and manuscript reconciliation remain later obligations.
