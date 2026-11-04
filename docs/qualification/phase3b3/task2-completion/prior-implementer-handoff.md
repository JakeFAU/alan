# Phase 3B.3 Task 2 implementation handoff

Read-only handoff written after parent transferred source ownership to
`/root/phase3b3_primitive_completion`. This agent will make no further source edits,
commits, or test runs. The parent-requested task-2 report is still the obsolete
v1-obstruction report; replacement must update it.

## Source and authority

Runtime worktree: `/private/tmp/alan-phase3b3-runtime`, branch
`codex/phase3b3-request-delegation`.
Current HEAD during final commands: `6eca5f81faf41165e7fa77819537e49fafcd0249`.
That commit is the parent's evidence packaging; production base remains
`af6943a5805a9a6c0e8fe77f804264f4765d07ec`, baseline fixture commit
`8319a9067c16aaf5e3ecccb3d0c62f5c536e6140`.
Implementation is **uncommitted**. No commits, subagents, new dependencies, paid
services, or external endpoints were used. Three existing baseline files were
not edited. No archived SQLite originals were opened.

Frozen contract is research version 2, commit `f47e5caa`, contract SHA-256
`c96241fa365807c4322756d0513c9fda05edff6650c4bc09f32f8cde1bbcb863`.
Version 1's literal-root construction was genuinely falsified and parent archived
its original source/log/contract before correcting RootID to RootRun locators.

## Implemented candidate

New `internal/executor/request_authority.go` (572 lines at handoff): public
protocol types, canonical claims/selectors, logical-root candidate derivation,
original `composition_permissions` tuple membership, retained lifecycle grant and
revocation decisions, current-source issuance checks, immediate selected request
binding, selected-path effect witnesses, and historical shared-fact-prefix
reconstruction. It does not use broad lifecycleGrant for the live control path.

New `internal/executor/request_authority_admission.go` (333 lines): typed primary
`authority_admission` facts in existing composition_facts, per-original ordinals,
exact immutable attempt digests, actual owner epoch/source sequence, original
reservation and debit checks, source/scope and same selected root recheck,
canonical insertion/readback before opportunity release, historical gate
recomputation, and earlier allowed gate requirement for retained actual results.
No additional table, accounting ledger, scheduler, transitive grants, or generic
adapter retry was added.

Existing files changed: `internal/boundary/boundary.go`; executor
`adapters.go`, `composition_facts.go`, `composition_replay.go`,
`composition_store.go`, `composition_types.go`, `dispatch.go`, `external.go`,
`inspection_causality_authority.go`, `inspection_causality_chronology.go`,
`inspection_causality_sources.go`, `inspection_deployment.go`,
`inspection_effect_order.go`, `inspection_effect_results.go`,
`inspection_effects.go`, `inspection_execution.go`, `inspection_publications.go`,
`lifecycle.go`, `lifecycle_replay.go`, `messages.go`, `program.go`,
`receipt_conflict.go`, `runner.go`, `store.go`; `runtime/runtime.go`.

New tests: executor `request_authority_test.go`,
`request_authority_negative_test.go`, `request_authority_formats_test.go`,
`request_authority_run_test.go`, integration-tagged
`request_authority_adapter_test.go`; public consumer
`runtime/request_authority_test.go`.

Exact frozen public aliases are in runtime/runtime.go. Existing runtime aliases
expose `InspectionFeatures.RequestAuthority`, `LifecycleResult.Authority`,
`ExternalAttempt.Authority`, `AdapterAttempt.Authority`, and
`ExecutionEffect.AuthorityAdmissions`. Literal selectors are outside Payload;
emit copies the pointed-to value. Existing empty-field JSON uses omitempty.
Formats 13/14 route to composition protocol 4; 13→14 conflict promotion is tested.
13/14 do not imply old dispatch-journal or continuation features.

Typed inspection recomputes authority history while still in its coherent read
transaction, before releasing its read mark. It then checks other ordinary
in-memory evidence with a private `authorityValidated` marker. Its marker is not
read from the Store. This reuses the runtime historical policy helper; it is not
an independent research authority oracle. The parent's independent checker must
remain independent. Every primary gate has a distinct chronology node and
(operation, ordinal) key rather than collapsing to operation alone.

## Boundary and digest interface for Task 4

- `boundary.Event.AuthorityAdmissionOrdinal int64`, JSON omitempty.
- `authority.admission`, phase `allowed` or `refused`, occurs after durable gate
  commit and lock release, before worker release. It has exact intent, ordinal,
  actual gate epoch and actual sequence. Boundary observer errors release no
  worker; an allowed opportunity may remain retained but unexposed.
- `dispatch.consider/pre-admission` remains before the gate.
- `dispatch.admitted/pre-call`, external/adapter worker events and completion
  events carry the same allowed ordinal and actual epoch/sequence.
- `dispatch.refused` carries refused ordinal when a new authority gate exists.
- Existing `external.authorized` is inside the worker after gate admission.
- Stable external Request.Epoch remains original authorization epoch. The new
  event/admission Source identifies physical activation; do not conflate them.
- Attempt digest: `compositionIdentity("alan/attempt-row/v1", fullAttempt)`.
- Primary RecordDigest:
  `compositionIdentity("alan/authority-admission/v1", AuthorityAdmission)`.
- Primary ID:
  `compositionIdentity("alan/fact/authority-admission/v1", []any{namespace, operationID, ordinal})`.
- Existing complete-composition direct DispatchExternal remains ErrDispatchMode.
  The supported physical path is ordinary Run. Parent explicitly instructed not
  to broaden direct APIs just to satisfy overbroad wording in contract section 4.

Authority refusal strings appear directly in request_authority.go (search
`authority_`). Ordinary attempt Allowed/Reason still combines source, authority,
and resource decision; nested Authority keeps permission disposition distinct.
Adapter/external settlement disposition maps authority denial to existing
grant_denied while nested witness preserves its precise reason.

## Actually executed verification

All commands below used cwd `/private/tmp/alan-phase3b3-runtime` and environment:

```
GOTOOLCHAIN=local
GOCACHE=/tmp/alan-phase3a-go-build
GOPROXY=off
GOSUMDB=off
```

Final ordinary/integration/race commands additionally used
`ALAN_PHASE3B3_OLD_READER=/tmp/alan-phase3b3-old-reader/alan`.
Staticcheck commands additionally used
`STATICCHECK_CACHE=/tmp/alan-phase3b3-staticcheck`.

Final **actual completed exit statuses were collected** from existing exec
sessions after the handoff instruction (polling already-finished commands, no
new test execution). All were 0:

1. `go test -json -count=1 ./internal/executor ./runtime ./author`
   → `/tmp/alan-phase3b3-final-normal.jsonl`.
   294 top-level tests passed, 735 pass events including subtests, 0 failures,
   0 skips; 58 authority pass events.
2. `go test -json -tags=integration -count=1 ./internal/executor ./runtime ./author`
   → `/tmp/alan-phase3b3-final-integration.jsonl`.
   381 top-level tests passed, 1,544 pass events including subtests, 0 failures,
   0 skips; 63 authority pass events.
3. `go test -json -race -tags=integration -run '^TestRequestAuthority' -count=1 ./internal/executor ./runtime ./author`
   → `/tmp/alan-phase3b3-final-race.jsonl`.
   25 top-level tests, 63 pass events including subtests, 0 failures, 0 skips.
   Author has no matching tests and package completion is recorded separately.
4. `staticcheck ./internal/executor ./runtime ./author`
   → `/tmp/alan-phase3b3-final-staticcheck.log`, empty, exit 0.
5. `staticcheck -tags=integration ./internal/executor ./runtime ./author`
   → `/tmp/alan-phase3b3-final-staticcheck-integration.log`, empty, exit 0.
6. `git diff --check`, exit 0 (run before final suites and subsequently checked).

Loopback test launches initially failed in sandbox (`bind: operation not
permitted`); authorized tool escalation was used for the scoped suites and
ordinary-Run loopback witness. This environmental failure is not semantic RED.

Earlier full scoped integration run:
`go test -tags=integration ./internal/executor ./runtime ./author`
→ `/tmp/alan-phase3b3-scope-second.log`, exit 0. Earlier ordinary scope run
`/tmp/alan-phase3b3-scope-first.log` failed one pre-existing adversarial test
because the new DDL feature check queried exact lowercase table name rather than
lower(name). Corrected; final suites include that test and pass.

## RED and development evidence

1. Original design RED:
   `go test ./internal/executor -run '^TestRequestAuthorityLiteralRootLocator$' -count=1 -v`
   → `/tmp/alan-phase3b3-literal-root-red.log`, exit 1, first Store succeeds and
   second ordinary Create with prior exact hashed ResourceRoot.ID fails
   `root identity mismatch`. Original source/log archived by parent at runtime
   `docs/qualification/phase3b3/rejected-literal-root`.
   Active test now correctly asserts historical rejection; logical-root
   two-Store construction is a separate positive test.
2. Interfaces-first RED:
   `go test ./internal/executor -run '^TestRequestAuthority' -count=1`
   → `/tmp/alan-phase3b3-format-red.log` and
   `/tmp/alan-phase3b3-policy-red.log`, exit 1 (missing types/fields, compilation).
   These are compile RED, **not executed policy failures**.
3. Gate-interface RED:
   `go test ./internal/executor -run '^TestRequestAuthorityAdmissionAndReopen$' -count=1`
   → `/tmp/alan-phase3b3-gate-red.log`, exit 1 (missing field compilation).
4. **Substantive implementation RED**:
   `go test ./internal/executor -run '^TestRequestAuthorityGateRequiresOriginalDebit$' -count=1 -v`
   → `/tmp/alan-phase3b3-debit-red.log`, exit 1, exact failure
   `gate accepted reservation without original debit` after deleting the
   operation's original composition_charges row from a disposable live Store.
   The gate originally checked allowed attempt plus presence of reservation fact
   but omitted exact original debit reload. Fix reloads exact reservation,
   charge descriptor/root/order, and original source allowance.
   Same command → `/tmp/alan-phase3b3-debit-green.log`, exit 0.
   **Exact original failing production source bytes were NOT saved.** Do not
   claim a retained original rejected source snapshot. The failing test source
   remains at the end of request_authority_negative_test.go and the RED log is
   original. A reverse patch removing the reservationRoot/reservationRaw through
   original-source-allowance block before retainedEffectAuthority, and removing
   authorityAttempt's new exact-intent/authID comparison, would be a
   reconstruction of the omitted checks; label any such artifact reconstruction,
   not original preserved bytes. I did not write a reconstructed artifact.

Other development logs (all are original logs, not frozen source snapshots):
- `/tmp/alan-phase3b3-policy-first.log`: focused policy green after fixing fixture
  message Operation correlation (ordinary-message observations have empty
  Operation; authentic request is followed through admission.IntentID).
- `/tmp/alan-phase3b3-gate-first.log`: failed unfinished gate routing plus test
  cleanup nil panic; `/tmp/alan-phase3b3-gate-second.log` final green overwrite.
- `/tmp/alan-phase3b3-run-first.log`: sandbox listener failure;
  `/tmp/alan-phase3b3-run-second.log`: ordinary two-agent Run green.
- `/tmp/alan-phase3b3-negative-first.log`,
  `/tmp/alan-phase3b3-negative-second.log`,
  `/tmp/alan-phase3b3-causal-first.log`,
  `/tmp/alan-phase3b3-policy-attacks.log`: focused negative tests green.
- `/tmp/alan-phase3b3-adapter-first.log`: fixture compile error (Request.Name
  should be Request.Effect); `/tmp/alan-phase3b3-adapter-second.log`: green.
- `/tmp/alan-phase3b3-formats-first.log`: format promotion, feature masking,
  old-reader rejection, and distinct holder/shared-root tests green.
- `/tmp/alan-phase3b3-public-first.log`: public committed-control consumer green.
  Public test was simplified before final runs to construct correct logical
  recipient root directly; final runs verify that final source.

## Qualified scope and pending work

The raw ordinary Run smoke uses B's committed grant, matched lifecycle result,
accepted branch, then ready handshake; A selects ready then genuine BA and emits
service immediately. B waits for BA-paid before selecting AB and doing its own
service. Two external receipts are selected, effects have settlements, both
agents succeed, and the loopback ledger records two contacts/applications.
BA-paid and AB-paid are **empty application acknowledgments**, not result-bearing
opaque replies. Receipt receives do not branch on accepted; the test separately
checks retained actual receipts/effect records. Do not infer original requester
opaque result custody solely from terminal statuses. Task 4 must carry the full
result-bearing reply chain. Mapping is S2/LateGrant-before-first-decision, not
S2/Explicit inline atomic Commit.

The adapter fixture fixes each capability to a distinct in-memory mutable
resource through trusted Prepare/Execute callbacks; increment and double perform
different real state changes. Cross resource/operation pairs are refused and
never invoke Execute. It is a controlled host-local adapter test, not a durable
remote ledger or distributed proof. Generic allowed/no-result adapter remains
unknown after reopen and prepareAdapterWork does not retry it.

Controlled direct owned-transition tests cover selected grant/refusal, own path,
root and recipient generation/revision/scope errors, missing original permission,
result-as-grant, no fallback even when own permission exists, no redelegation,
reuse, immediate-selection failure, sticky refusal after late grant/reopen,
source closure before issuance, normal delegator completion preserving consent,
revocation after an allowed gate denying later gates while leaving another tuple
allowed, repeat gate ordinals/new epochs, forged caller LifecycleRequest,
corrupted primary descriptors/order, missing gate with a retained result,
original debit removal, malformed canonical claims, unsupported profile
combinations, 13→14 conflict promotion, masked 13→4 feature, and frozen old-reader
inspect/replay rejection for 13 and 14 on disposable copies.

Not done by this agent: final frozen-source packaging, full-repository suite,
exhaustive prior-model reruns, independent authority checker, full differential
model/runtime corpus, SIGKILL qualification, hostile-store activation proofs
beyond existing epoch bounds, manuscript/PDF work, independent reviews, commits,
or PR. Parent/replacement own these. Existing broad normal/integration scope
passing is evidence within that scope, not universal 3–12 byte compatibility.

## Last read-only concern: unserviced dynamic lifecycle operations in legacy stores

**Suspected, not demonstrated. No reproducer was executed and no source edit was
made for this concern.** Parent asked source remain stable; then transferred work.

Current live `Commit` and `loadIntent` invoke validateRequestAuthorityIntent,
which rejects DelegateAuthority/RevokeAuthority when format is not 13/14.
`loadProgram`/insertArtifact reject literal authority operation and selector
fields outside the feature. However, `validateRequestAuthorityProgram` cannot
know a dynamic `Effect.Payload["operation"]` operand's eventual value. A program
using Ref("operation") with initial data containing "DelegateAuthority" may
have been allowed to commit an unserviced unknown lifecycle intent by the frozen
old runtime (before its lifecycle request loader refuses the unknown operation).

`readIntents` in replay.go bulk-decodes descriptors without invoking
validateRequestAuthorityIntent. Ordinary replay reconstructs reducer outputs but
does not apply that new Store-level per-intent feature gate to every output.
validateMessaging calls loadIntent for authorization/admission records, but an
unserviced intent might have neither. Typed inspection similarly checks
lifecycle results and effects but may not reject a bare committed new control
operation whose operation name was dynamically selected.

Suggested demonstration for replacement: use the already-built frozen old CLI
`/tmp/alan-phase3b3-old-reader/alan` to create a disposable legacy Store from a raw
program whose lifecycle operation operand is Ref("op") with initial op set to
DelegateAuthority, then run until old delivery rejects unknown operation after
commit. Inspect whether a new reader Replay/Open/InspectExecution admits that
bare authority-control intent despite absent feature. Alternatively a temporary
Go overlay could construct a valid retained reduction without altering reviewed
candidate source. Do not assert the bypass without an executed retained witness.
If reproduced, add the new per-intent feature validator at bulk replay and typed
history boundaries, preserving legacy ordinary behavior and rerun relevant
checks. Notify parent before source edits as requested.
