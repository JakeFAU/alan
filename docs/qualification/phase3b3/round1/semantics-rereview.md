# Task 2 authority semantics re-review, round 1

**Specification verdict: APPROVE. Code-quality verdict: APPROVE.** These verdicts cover this reviewer's Task-2 authority-semantics/minimality scope and the correction delta. Both P2 findings in `task-2-semantics-review.md` are **ADDRESSED**. That original REJECT report remains unchanged as historical evidence. This is not approval of the whole research phase or a substitute for the other reviewer's live-request-binding decision.

## Reviewed identity and scope

Runtime base remains `6eca5f81faf41165e7fa77819537e49fafcd0249` plus uncommitted correction source. I inspected `task-2-fix1-review.diff` in full, the report's Round 1 section, relevant current source and regression tests, and the recorded command/log evidence under `/tmp/alan-phase3b3-round1`. I independently verified all **294** source entries in `task-2-fix1-source.json` against the live runtime and all **33** entries in the round-1 evidence hash manifest: zero mismatches.

- Source manifest SHA-256: `1a073a07f114216a0f6937f6bba5b24f5eba89f682e96a49a511fbbb438e81c1`.
- Governing contract: v3, research `360cae66`; current contract SHA-256 independently checked as `f19d9c13f84c17496fabef91cb7575cf52972ff228c0b69e063cdc1c99100e12`.

No production/test code was edited, no tests were rerun, no archived SQLite original was opened, no subagent was spawned, and no commit was made by this re-review.

## Finding 1 — Legacy inert lifecycle-name compatibility: ADDRESSED

`validateRequestAuthorityProgram` no longer rejects a legacy literal merely because its text spells DelegateAuthority/RevokeAuthority. `validateRequestAuthorityIntent` retains its typed selector rejection, while authority-specific claim parsing now requires `enabled`. Thus a legacy Ref-derived name follows the same ordinary retention behavior as a literal. `loadLifecycleRequest` still refuses those names outside 13/14, now with the original `unknown lifecycle operation` behavior. Its opt-in branch still validates the claim, and DecideLifecycle still reloads the exact committed request before any decision.

This realizes the approved v3 distinction: old strings stay inert; they are not retroactively interpreted as grants or revocations. Removing only the name-based checks also preserves historically retained bare-intent prefixes in replay/inspection/open instead of introducing the previously proposed overbroad bulk rejection. Typed selectors, new result witnesses and admission facts do not become legal in legacy profiles.

The original four-case test is retained without weakened expectations. It exercises both DelegateAuthority and RevokeAuthority, literal and Ref forms, ordinary legacy Create/Commit, unknown-operation loader refusal, Replay, InspectExecution and Open. `red.log` records the four original failures; `legacy-green.log` and each final suite record all four passing. The original old-source comparison remains in the previous preserved evidence. This is concrete schema-3 regression coverage plus the inspected feature-independent branch correction, not a newly established exhaustive equivalence theorem for formats 3–12.

## Finding 2 — Forbidden nested authority witness in legacy inspection: ADDRESSED

`inspection_effects.go` now extracts the nested EffectAuthority from either ExternalAttempt or AdapterAttempt before branching on the feature. Its legacy branch rejects `i.Authority != nil || w != nil`. The correction therefore checks the field the original independent falsifier injected, covers both attempt kinds, and does not impose opt-in witness requirements on clean legacy attempts.

`TestRequestAuthorityLegacyExternalWitness` reproduces the original mutation and includes a clean control. The integration-tagged `TestRequestAuthorityLegacyAdapterWitness` does the same for the other attempt kind. The common helper separately asserts outcomes from Replay, InspectExecution, and Open after closing the fixture Store. The pre-fix RED log specifically shows clean readers passing, Replay/Open already rejecting malformed data, and inspection wrongly accepting both nested witnesses. The post-fix log shows all six clean reader checks and all six malformed reader checks passing; the final integration and focused race JSON streams include every one of those cases. These are rejection tests for forged retained evidence; no transport or unauthorized physical application is claimed.

## Inspection of the remaining fix delta

The other production change extracts the existing external offline reconstruction predicate into `validateExternalAttempt`, then calls it at the live admission transaction; adapters reuse `validateAdapterAttempt`. The extracted comparison still binds the complete ExternalAttempt to the retained intent, original authorization identity/epoch, namespace-derived key, original authority witness and retained reservation. The remaining offline receipt check uses `a.Allowed` only after that exact comparison succeeded. The added gate check precedes primary fact insertion and leaves existing selected-path, historical-prefix, original-debit and fresh-current-authority checks in place.

I found no new breakage or scope expansion in this delta. It introduces no scheduler, table, charge/refund path, authority source, or generic retry. The reopen regression explicitly keeps the original wire request unchanged while requiring a different fresh activation epoch. Reusing read-only validation inside the existing owned transaction does not mint an additional permission. Arbitrary adapter Wire/Endpoint reconstruction is still outside the trusted adapter ABI; the report correctly does not claim otherwise. The other independent reviewer owns the final finding-level assessment of live original-request binding.

## Retained verification inspected

`/tmp/alan-phase3b3-round1/commands.json` records exact commands, cwd, base HEAD, environment and actual process exits. Test environment includes `GOTOOLCHAIN=local`, `GOCACHE=/tmp/alan-phase3a-go-build`, `GOPROXY=off`, `GOSUMDB=off`, and `ALAN_PHASE3B3_OLD_READER=/tmp/alan-phase3b3-old-reader/alan` for final suites. Staticcheck uses `STATICCHECK_CACHE=/tmp/alan-phase3b3-staticcheck`.

I independently parsed all final JSONL records:

| Recorded command | Exit | Independently counted test/subtest passes |
| --- | --- | --- |
| `go test -json -count=1 ./internal/executor ./runtime ./author` | 0 | 759 |
| `go test -json -tags=integration -count=1 ./internal/executor ./runtime ./author` | 0 | 1,586 |
| `go test -json -race -tags=integration -run '^TestRequestAuthority' -count=1 ./internal/executor ./runtime ./author` | 0 | 105 |

Each stream has three package passes and zero failure or skip events. Counts include subtests; they are not independent experiment counts. Normal/integration scope is the three named packages, not `./...`; race is focused, and author has no matching tests. Both recorded Staticcheck commands for the same packages, with and without integration tags, exited 0 with empty logs. The earlier misleadingly named `authority-green.log` is explicitly retained as exit 1 for a fixture assumption; it was not used as green evidence. The corrected non-idempotent fixture preserves the old no-retry behavior and probes the gate separately.

Existing positive authority, refusal/revocation, historical validation, accounting, ordinary two-agent Run, reopen, format and old-reader tests are included in the final scoped verification. No fresh probe was needed because the fixes are direct, the targeted negative/clean controls cover both findings, and their source and retained RED/GREEN evidence matched the manifest.

## Remaining limits

This approval clears this reviewer's two implementation findings for the identified candidate; it does not convert shared runtime validation into an independent authority oracle. Full requester result custody, model/runtime differential correspondence, real process-cut and independent contact qualification, precise old-format snapshot qualification, manuscript reconciliation, and the existing two-agent/single-Store/one-hop limits remain as previously recorded. The positive ordinary Run retains its empty-ack limitation. Direct composition DispatchExternal remains unsupported. Task-level or phase-level completion still requires the separate remaining evidence and reviews.
