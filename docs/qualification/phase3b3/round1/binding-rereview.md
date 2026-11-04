# Task 2 attenuation/request-binding re-review, round 1

2026-11-04. **Original P2 finding: ADDRESSED. Specification: APPROVE. Code quality: APPROVE, within this Task 2 binding review scope.** No new blocking finding was found in the correction delta. This does not approve later correspondence/recovery qualification or waive the separate reviewer's legacy findings. The original `task-2-binding-review.md` REJECT report and its executed witnesses remain unchanged.

## Source and review scope

Read the Round 1 section of `task-2-report.md`, the complete 706-line `task-2-fix1-review.diff`, `task-2-fix1-source.json`, and the actual owning implementation/test paths in `/private/tmp/alan-phase3b3-runtime`. The candidate remains uncommitted atop `6eca5f81faf41165e7fa77819537e49fafcd0249`; governing contract is now frozen v3 as recorded in the report. Independently verified all **294** source/module manifest entries against the current runtime files: no mismatches. The delta is five production paths and four test paths, with no additional production expansion.

This re-review focuses on the original P2: reconstruct the original request before releasing a fresh gate. Legacy operation-name compatibility and forbidden legacy nested witnesses belong to the other reviewer. Their deltas were read for interactions but are not independently cleared by this report.

## Why the P2 is addressed

`recordAuthorityAdmission` now calls the original-attempt validator **inside its existing owned transaction**, before computing/inserting the primary gate and before the caller registers/releases an opportunity. The same `mu`/`ownerMu` exclusion and transaction cover validation, fresh policy evaluation and gate commitment. The added validators use the transaction query interface; they neither reenter Store locks nor perform transport, adapter preparation, charges, refunds or new authority decisions.

For external effects, extracted `validateExternalAttempt` rebuilds the entire wire `Request` from the real retained intent and original authorization. Namespace-derived key, authorization ID and **original authorization epoch** are reconstructed; the fresh physical activation is not substituted into the original wire request. Full `ExternalAttempt` comparison also preserves the exact intent, combined disposition and historical selected authority witness. The extraction preserves the old offline validator's comparison; its later receipt check uses the now-validated `a.Allowed`, equivalent to the old locally computed `allowed`.

For adapters, the gate calls the existing `validateAdapterAttempt`. That helper reconstructs `Prepared.Request` through `adapterRequest`, using the original retained program's effect declaration, input/output schemas, intent payload, capability, operation, original ID/key and retained assets. It compares the complete request and verifies prepared digest/shape/bounds. This closes the observed `increment`→`double` substitution without asserting independent reconstruction of opaque adapter `Wire`, `Endpoint`, `Model` or trusted deployment configuration.

After those checks, the original selected witness, exact reservation/debit, fresh source/root validity, historical prefix interpretation and per-original ordinal checks remain in `expectedAuthorityAdmission`. Historical first permission decisions continue to use the reservation prefix; current revocation is considered at the new gate. The fix does not make a sticky first allow/refusal mutable or regrant on reopen.

## Retained evidence independently inspected

Read `/tmp/alan-phase3b3-round1/commands.json`, overlay and logs; verified every one of the **33** entries in `sha256.json` against its actual evidence file. The saved overlay admission source is byte-for-byte identical to the original frozen candidate-v2 admission source. Both corrected expanded-RED test files exactly match the final runtime test files; `final-source/external.go` and `final-source/request_authority_admission.go` also exactly match current source.

The original adapted two-case RED and focused GREEN are preserved. More strongly, `binding-expanded-red-corrected-fixture.log` records all **16** mutations releasing an erroneous opportunity under the exact pre-fix admission-source overlay: external delta/profile/key/epoch/authorization/subject/generation/sequence and adapter effect/ID/key/capability/payload/input/output/assets. Each retains a genuine original effect/authority/debit and coherently rewrites the attempt/reservation/fact digests; adapter prepared digests are recomputed too. Offline validators reject the malformed histories. This is a controlled partial-source overlay experiment, not a claim that every source file was reverted to the original candidate.

The first expanded RED and misleadingly named `authority-green.log` genuinely contain the non-idempotent-profile fixture-assumption failure: preparation correctly returns nil for unknown non-idempotent work. The corrected test preserves that expectation and separately probes the gate. The final 16-case RED consists of actual gate-release failures, not compilation or that fixture failure. No expected authorization behavior was relaxed.

Parsed complete final JSONL streams and checked pass events for every mutation, the equal-payload/different-original-request regression, and admission/reopen regression:

| Retained command scope | Recorded exit | Test/subtest passes | Package passes | Fail/skip events |
| --- | --- | --- | --- | --- |
| Ordinary `internal/executor`, `runtime`, `author` | 0 | 759 | 3 | 0/0 |
| Integration, same three packages | 0 | 1,586 | 3 | 0/0 |
| Race + integration, `^TestRequestAuthority`, same packages | 0 | 105 | 3 | 0/0 |

The command/environment/cwd/HEAD records supply process exits; JSON pass events alone are not treated as command provenance. Both Staticcheck command records report exit 0 and their retained logs are empty. Exact commands and environments remain in `commands.json`; evidence hashes remain in `sha256.json`. Counts include subtests and are not independent experiments.

The expanded tests require both an error and no released opportunity. External/profile explicitly tests existing non-idempotent retry suppression, then directly probes the gate. Reopen coverage compares the entire immutable original Request across reopening, requires its original epoch to differ from the fresh activation epoch, and checks the new gate's actual epoch and next ordinal. The equal-payload request test continues to reject selection of B/1/2/0 under the grant bound to B/1/1/0. Existing positive ordinary Run, revocation/history and debit cases are included in the retained final streams.

No test suite was rerun during this re-review, as requested; these are independently inspected retained executions, not executions claimed by this reviewer. No source edits, SQLite archive opens, subagents or commits occurred. Only this new review report was written.

## Remaining boundary

The P2 concerns malformed owned-Store gate release, not an ordinary-public-API exploit; the regressions perform no physical contact. Approval establishes the reviewed correction and bounded supporting evidence. It does not establish arbitrary opaque adapter-wire provenance, an independent authority oracle, real SIGKILL/contact correspondence, full original-requester opaque-result custody, unrestricted compatibility, or a global completion theorem. Those obligations and the other reviewer's findings retain their existing ownership and status.

**Verdict: P2 ADDRESSED; specification APPROVE / code quality APPROVE for the round-1 attenuation and request-binding correction.**
