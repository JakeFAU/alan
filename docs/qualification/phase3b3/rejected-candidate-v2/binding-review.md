# Task 2 independent attenuation and request-binding review

2026-11-04. **Specification: REJECT. Code quality: REJECT.** One substantive P2 finding remains in the frozen candidate: the live physical gate checks the authority selector and reservation digest but does not reconstruct the derivable external/adapter request from the original retained effect. Two independently added falsifiers demonstrate the gap. The historical validators correctly reject both malformed histories. Correction and re-review are required; this rejection must remain preserved after correction.

## Reviewed claim and source

Question: does one-hop delegation preserve the exact original complete permission tuple, original request and service operation, logical generations and root locators at issuance, first effect decision and every fresh physical gate, including historical replay/inspection and same-Store reopen?

Read the user packet `docs/foundational/phase3b3/00-brief.md`, frozen contract v2 `02-contract.md`, `task-2-brief.md`, `task-2-report.md`, the full `task-2-review.diff` and corresponding actual source. Review covered the two new authority implementation files; lifecycle request/decision/replay; program emission and intent loading; external/adapter preparation, reservation and result binding; dispatch/runner; composition prefix/facts/version/profile; inspection authority, chronology, deployment, effects, results and publication paths; and all new authority tests/public aliases. Adjacent original request builders and validators were traced rather than treating digests as authorization.

The reviewed frozen source is `/tmp/alan-phase3b3-candidate-v2/source.tar.gz`, SHA256 `8ef231aa4ae78e10389528b546458a930961cd47c787d8ecd3e699c72f23e754`, uncommitted candidate atop `6eca5f81faf41165e7fa77819537e49fafcd0249`. The review diff includes the deliberately RED legacy-compatibility regression. The live implementation files were inspected in `/private/tmp/alan-phase3b3-runtime`; experiments used a separate extraction at `/tmp/alan-phase3b3-binding-probe`. All 292 source/module entries in `source.json` were verified against that extraction with no mismatches after the experiments. Only new review test files were added there. The source-only archive omitted embedded SQLite gzip fixtures required for compilation; complete compressed fixtures were copied from the runtime checkout without opening them, with hashes retained in `evidence/copied-fixtures.json`.

No reviewed source, archived original SQLite database, baseline fixture, branch or commit was modified by this reviewer. No subagents were used.

## P2: reconstruct the original request before releasing a fresh gate

Location: runtime `internal/executor/request_authority_admission.go:24–37` (`authorityAttempt`) and `:51–89` (`expectedAuthorityAdmission`).

`authorityAttempt` reloads the attempt and compares only its `Intent` and `AuthorizationID` with the real intent. `expectedAuthorityAdmission` then compares the reservation against a digest of that same loaded attempt and reconstructs the permission witness. It never independently derives the external wire request or adapter `Prepared.Request` from the immutable intent/program. Coherently rewriting the attempt and reservation digest therefore preserves those checks while changing the action handed to transport. Recomputing a digest establishes agreement between the modified records, not agreement with the original effect.

The external attack preserves the genuine intent, accepted grant, full permission tuple, selected original request, source authorization, reservation order and original serving-root debit. It changes only `ExternalAttempt.Request.Delta` from 1 to 999, updates the reservation `AttemptDigest` and regenerates its mirrored composition fact. `prepareExternalWork` then returns the substituted delta and `registerDispatch` durably records an allowed gate and releases an opportunity. Replay rejects `external attempt binding`; typed inspection rejects `external wire request differs from original effect`.

The adapter attack preserves the same original chain and changes `Prepared.Request.Effect` from `increment` to `double`, recomputes `Prepared.Digest`, the reservation attempt digest and its mirrored fact. Opaque `Wire`, `Endpoint` and deployment configuration remain untouched. The live gate releases an opportunity. Replay rejects `adapter attempt binding`; typed inspection rejects `adapter request differs from original program, inputs or assets`.

This is an **owned live Store malformed-data** witness, requiring direct database mutation after an ordinary allowed reservation; it is not an ordinary-public-API exploit. No physical call or adapter Execute was performed, so the experiment establishes erroneous gate release, not an observed unauthorized application. The scope is nevertheless required here: contract §2 binds external delta/profile bytes to the exact service operation; §4 binds the exact prepared/raw request; §8 requires malformed/substituted-attempt rejection; the existing debit regression already tests live malformed-store gate refusal. Offline rejection does not undo a released physical opportunity.

Required correction: before live gate release, reconstruct and compare every checkable original-request field, using the original authorization namespace/key/epoch and immutable effect/program/schema/assets. Existing `externalRequest`/historical `validateExternal` logic and `adapterRequest`/`validateAdapterAttempt` already identify these checkable predicates. Preserve the same selected authority witness, exact original reservation/debit, no fallback, and historical prefix logic. Do not claim arbitrary opaque adapter wire can be recomputed independently of its trusted adapter ABI. Add the two live regressions and re-review the correction.

Exact witnesses:

- External original executed test: `/tmp/alan-phase3b3-binding-probe/evidence/substituted-wire-original-test.go.txt`, SHA256 `30ab4d4ff77be206963ee6c431f7a88bf78cee1abd2a81fdae768e04be17a35e`.
- Current combined external/equal-payload test: `/tmp/alan-phase3b3-binding-probe/internal/executor/request_authority_binding_review_test.go`.
- Adapter test: `/tmp/alan-phase3b3-binding-probe/internal/executor/request_authority_binding_adapter_review_test.go`.
- Logs: `evidence/substituted-wire.log` and `evidence/substituted-adapter-request.log`; exact commands/exit statuses: `evidence/command.json` and `evidence/substituted-adapter-command.json`.
- Fresh closed disposable databases: `evidence/substituted-wire.sqlite` and `evidence/substituted-adapter-request.sqlite`. Copy complete files before opening either for further analysis.
- Complete evidence hashes: `/tmp/alan-phase3b3-binding-probe/evidence/sha256.json`.

## Checks that survived this review

`rootReason` tests the complete six-field `Permission` with `slices.Contains`, separately checks holder provenance/revision/generation and exact `RootRun` membership, and rejects exact-root revocation. It does not construct independent resource/operation dimension sets. The adapter action predicate requires exact Kind/Target/Operation/Obligation; the external predicate requires exact endpoint/`apply`/obligation. Issuance reloads the committed lifecycle request, source authorization and original committed message; normal delegator completion does not implicitly revoke retained consent, as required by v2.

The delegated path names an actual accepted issuer lifecycle decision, requires exact Root, recipient RunID/revision/root and actual ServiceOperationID, and follows the authentic original message admission to the immediately preceding selected generation-bound observation. There is no own/delegated fallback or delegated-parent resolution. The gate reuses the original witness and re-evaluates that same selector at its fresh prefix. Root-specific revocation, per-original ordinals and original serving-root debit survive the existing focused tests; reopening neither regrants nor refunds. Scope is exactly two static principals and the declared feature exclusions.

A novel ordinary-transition test independently inserted two authentic requests with the same source, generation, recipient, obligation and payload. The accepted grant named request B/1/1/0, while A selected B/1/2/0. The immediate next effect was refused with `authority_request_not_admitted`; Replay and InspectExecution accepted the truthful refused history. Command `go test ./internal/executor -run '^TestBindingReviewEqualPayloadRequestSubstitution$' -count=1 -v` exited 0. Evidence: `evidence/equal-payload.log` and `evidence/equal-payload-command.json`. Its first attempt accidentally reused a transport ID and failed at the unrelated ordinary admission identity check; the original fixture-error source/log are retained and are not presented as authority RED.

Historical authority is reconstructed at the lifecycle/reservation/gate composition prefix, rather than from today's revoked set. `retainedEffectAuthority` compares the full expected original witness. Gate validation binds descriptor, namespace, source activation bounds, ordinal, original debit and attempt digest. Allowed actual results require an earlier valid allowed gate for their exact original. The observed gap is specifically missing independent original-request reconstruction in the live gate, despite its presence in offline validation.

## Evidence inspected and commands actually executed

Inspected `/tmp/alan-phase3b3-primitive-handoff.md`, baseline manifest, logical-root design review, preserved rejected-v1 obstruction and original literal-root RED archive. Inspected original debit RED/green logs; original failing production bytes were not retained, so the subsequently recorded removed-check source is explicitly a reconstruction, not an original rejected snapshot.

Parsed all JSON lines in retained `final-normal`, `final-integration`, and `final-race` logs: respectively 735, 1544 and 63 **test/subtest** pass events, each with three package passes and no failure/skip events. Empty retained Staticcheck logs were inspected; process exits are attributed to the handoff/completion report, not inferred from emptiness. These suites predate the known legacy-compatibility RED and do not prove the current whole candidate green. Identical passing suites were not rerun.

New commands used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase3a-go-build GOPROXY=off GOSUMDB=off` in the disposable extraction:

1. `go test ./internal/executor -run '^TestBindingReviewSubstitutedWireRequest$' -count=1 -v`: exit 1, substantive fresh-gate binding RED. The initial compilation-only failure caused by absent embedded fixtures is preserved separately as `evidence/setup-*` and is not semantic evidence.
2. `go test ./internal/executor -run '^TestBindingReviewEqualPayloadRequestSubstitution$' -count=1 -v`: exit 0 after correcting the independent transport-ID fixture error.
3. `go test -tags=integration ./internal/executor -run '^TestBindingReviewSubstitutedAdapterRequest$' -count=1 -v`: exit 1, substantive fresh-gate binding RED.

No physical contacts, full-suite reruns, Staticcheck reruns, SIGKILLs, model correspondence runs or manuscript builds were performed by this reviewer.

## Scope and remaining obligations

The separate semantics/minimality reviewer owns the confirmed legacy unknown-operation compatibility conflict. This report does not waive or reclassify it; candidate-wide approval remains withheld independently of this binding finding. Full requester opaque-result custody, real SIGKILL/contact correspondence, independent authority checking and manuscript reconciliation belong to later Tasks 3/4. Existing ordinary-Run smoke and direct primitive checks do not establish a global correspondence or completion theorem. Store-only gates cannot prove a physical contact occurred or uniquely attribute returning bytes to a particular exposure, and historical empty-reopen timing retains the documented epoch-bound limitation.

**Final verdict remains specification REJECT / code quality REJECT for frozen candidate v2, pending a narrow live original-request binding correction and independent re-review.**
