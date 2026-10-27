# Phase 1 qualification and counterexamples

Date: 24 October 2026. This phase produced a test-only executable model of the [bounded operational specification](01-operational-spec.md), not a production runtime. The corrected model passed nine bounded safety searches, all three required broken-guard controls produced counterexamples, and two additional counterexamples distinguish run cancellation from retirement.

Independent review also found real defects in the first implementation: tag-only receipt matching and two identity collisions. These were reproduced, corrected, and retained as regression tests without excluding the offending inputs. The final independent-review verdict is **APPROVE for Phase 1 research-deliverable readiness**.

## Artifacts and reproduction

- [Research model and checks](../../../research/semantics/README.md) — five Go test-only files, with no importable production implementation.
- [Frozen scope and plan](00-plan.md), including explicit alternatives and exclusions.
- [Machine-readable evidence](qualification.jsonl) — 15 deterministic JSON records, including complete search alphabets, bounds, rejection reasons, witnesses, and exact counterexample/Control traces.
- [Decision updates](../05-decisions.md) — findings and unresolved architectural choices.

Run from the repository root with the installed Go 1.27.3 toolchain:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache GOPROXY=off GOSUMDB=off go test -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache GOPROXY=off GOSUMDB=off go test -race -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache GOPROXY=off GOSUMDB=off go vet ./...
gofmt -l research/semantics/*_test.go
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache GOPROXY=off GOSUMDB=off go test ./research/semantics -run TestQualification -count=1 -v
```

All commands passed in independent primary-agent runs; `gofmt -l` emitted no paths. `go.mod`, dependencies, and production packages are unchanged. The Go race check concerns the implementation of the tests; modeled concurrency is explored as event orderings, not inferred from Go's scheduler. At the original repository baseline, `go test ./...` reported no packages because no Go source existed.

To regenerate the evidence file without retaining Go's timing/log prefixes:

```sh
set -o pipefail
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1-go-cache GOPROXY=off GOSUMDB=off \
  go test ./research/semantics -run TestQualification -count=1 -v |
  sed -n 's/^.*QUALIFICATION //p' > docs/foundational/phase1/qualification.jsonl
```

## Bounds and search method

Every fixture uses at most two generations per subject, two activation epochs per run, eight committed instructions per run, two live continuations, two speculative proposals, two ordinary intents per run, two attempts per intent, and four retained inbox entries per run. There are two integer registers with values 0–8 and two effect-reference slots. Programs are immutable finite arrays; the largest program has nine instructions. Each turn executes one selected instruction. Ownership uses a distinct principal and revision 7, independently of generation and epoch.

The nine positive searches each contain one subject and a fixed event alphabet. The three-subject Control example is a separate reachability witness. Event alphabets are intentionally narrow: a retry scenario can expose generation 2 without including every possible instruction or lifecycle event for generation 2. Complete alphabets are saved in the JSON evidence and declared in `qualificationScenarios`; results apply to those alphabets, not the Cartesian product of all legal events/programs/domains.

The explorer performs breadth-first search. State identity includes the whole machine, immutable program configuration, full histories, speculative proposals, external resource state, and independent monitor facts. Every accepted successor is checked before deduplication. Rejected actions return unchanged state. No partial-order reduction or lossy merging is used.

**States** counts discovered unique nonviolating states including the initial state. **Transitions** counts accepted attempted edges, including stutters/revisits and a violating successor. **Rejected** includes unavailable actions and explicit capacity rejection; it is not a count of bugs. **Revisits** counts accepted edges merged with an already discovered state. **Boundary** counts states at the depth limit that were not expanded. A violating successor is included in the trace/depth but not in the state count.

Positive searches sum to **331 scenario states and 681 accepted transitions**, with 1,240 rejected actions, 359 revisits, and 58 unexpanded depth-boundary states. These are sums over separate graphs, not globally unique states. Twenty-three positive rejections are `capacity:epochs`; the evidence separates that bound from semantic rejections. Other listed finite bounds were not reached by these positive alphabets. A dedicated test forces the hard state cap to one and requires an error. No qualification search exhausted its hard state cap.

## Positive search results

No safety monitor reported a violation in these searches.

| Scenario | Depth limit | State cap | States | Transitions | Rejected | Revisits | Boundary |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Epoch and competing sequence | 5 | 10,000 | 56 | 86 | 136 | 31 | 19 |
| Retirement, authorization, application, receipt | 7 | 10,000 | 45 | 91 | 154 | 47 | 10 |
| Service binding, lost acknowledgment, retry | 6 | 10,000 | 30 | 85 | 77 | 56 | 3 |
| Abandonment, retry, late settlement | 8 | 20,000 | 66 | 126 | 330 | 61 | 9 |
| Run cancellation, retry disabled for cancelled outcome | 4 | 1,000 | 7 | 20 | 8 | 14 | 0 |
| Run cancellation, retry enabled for cancelled outcome | 4 | 1,000 | 11 | 31 | 9 | 21 | 1 |
| Receive choice, duplication, interruption | 7 | 10,000 | 16 | 50 | 46 | 35 | 0 |
| Idempotent endpoint with ambiguous attempts | 8 | 20,000 | 50 | 96 | 240 | 47 | 8 |
| Non-idempotent endpoint with ambiguous attempts | 8 | 20,000 | 50 | 96 | 240 | 47 | 8 |

Only the cancellation-disabled and receive-choice alphabets reached fixed points before their respective depth limits (maximum discovered depths 3 and 6). Even those fixed points concern their finite alphabets alone. The other searches are explicitly truncated at depth.

## What the checks actually establish within the model

| Contract | Evidence and limit |
| --- | --- |
| Structural validity | Monitors check current-run references, unique continuation identities, valid PCs, capacity, and intent/attempt/receipt linkage. These checks are separate from semantic progress. |
| Independent local progress | `TestReadyContinuationProgressAndFaultVersusInterruption` advances and completes a ready fork while its sibling waits on an absent message. This is an enabledness/reachability witness, not a fair-scheduling theorem. |
| Stale reducer and competing proposals | `TestStaleEpochAndCompetingSequence` and the epoch search exercise invalidated proposals and sequence conflicts; generation and ownership-revision rejection have separate focused checks. Dispatcher incarnations are excluded. |
| Interruption versus failure | Prepare creates no committed intent or consumed observation. Interrupt preserves the journal; retry of that still-active run is rejected. A committed Fault ends it and makes fresh retry eligible. |
| External success without a receipt | Separate application events change only the modeled world. The journal stays unresolved. `TestWorldIsInvisibleToOrdinaryDecisions` compares ordinary actions in equal runtime-visible states with and without unrecorded application. |
| Endpoint contract differences | Focused tests apply two attempts: the modeled idempotent endpoint mutates once and the non-idempotent endpoint twice. A new generation uses a new key and mutates again even with idempotency. Endpoint key retention/atomicity are assumptions, not ALAN guarantees. |
| Retirement and cancellation races | Search and focused checks exercise both cancellation/retry policies, retirement ordering, cancellation of an old run after retry, and late cancellation of already failed/abandoned runs. No stop-retry latch is implied. |
| Closure before/after authorization | Every terminal status blocks new ordinary attempts. Already authorized attempts may apply and settle after closure; never-authorized intents are suppressed. Both relevant orders are reachable. |
| Late receipts and notifications | The late-settlement witness requires a newly recorded receipt when the source was already closed. Frozen old-generation notifications retain their outcome, route, and generation after retry; repeated delivery does not advance the new run. |
| Fixed-version replay | Replay reconstructs version, epoch, sequence, data/reference slots, frontier, consumed observations, immutable intents, terminal status, and frozen outcome. Wrong version and independently tampered data/frontier/selection/payload/intent/notification/epoch/sequence are rejected. Live machine and external state remain unchanged. |
| Lost service acknowledgment | Admission is retained while acknowledgment is withheld; after failure and retry, retransmission returns the original binding and inserts nothing into the new run. Descriptor conflict and closed exact-run targets are separately rejected. |
| Correlation and identity separation | Regression tests retain spoofed receipt labels and colliding raw IDs, reject wrong-effect receipt consumption, preserve both messages, and prevent an unrelated admission from satisfying a Send. An additional forged-reducer control shows the consumption monitor catches a bypassed guard. |

Replay uses the same pure instruction reducer as execution, with independent history/descriptor comparisons; it is not a separately implemented reference semantics. Pending inbox/admission, attempts, mutable receipt/suppression evidence, and notification-delivery ledgers are checked separately, not reconstructed as a global replay. No storage crash recovery or consistent distributed snapshot is implemented.

## Guard-removal counterexamples

Each mutant changes the transition behavior while keeping the safety monitor unchanged. Qualification fails if the expected violation is not found. The traces below are shortest in their configured alphabets; the JSON retains every event argument.

| Removed protection | Search states / accepted edges before detection | Violated invariant | Trace |
| --- | --- | --- | --- |
| Epoch validation at commit | 14 / 19 | `commit-epoch` | Prepare at epoch 1 → Reactivate to epoch 2 → Commit old proposal |
| Closure/suppression decision at authorization | 15 / 20 | `authorization-after-closure` | Prepare Emit → Commit intent → RetireSubject → AuthorizeAttempt |
| Stable service binding on retransmission | 21 / 37 | `stable-service-binding` | Admit `m` to generation 1 → Prepare Fault → Commit Fault → Retry to generation 2 → Admit `m` again |

The closure mutant bypasses both the closure check and its suppression consequence, so a redundant guard cannot conceal the missing decision. It falsifies missing closure enforcement; it does not simulate or validate an actual distributed check/commit protocol. The binding trace never delivers the original acknowledgment.

## Policy counterexamples

These preserve all implementation guards. The intentionally false property is “any CancelRun stops future generations.”

1. With `retryCancelled=true`: `CancelRun(active g1) → Retry(g1)` creates `g2`. The explorer finds this at depth 2 (3 states, 4 accepted edges).
2. With `retryCancelled=false`: `Prepare Fault → Commit Fault → CancelRun(g1) → Retry(g1)` also creates `g2`, because late cancellation preserves the failed outcome. Detection occurs at depth 4 (7 states, 10 accepted edges).

These outcomes are not patched away. Subject retirement provides the permanent barrier in this model. A run-only stop-retry latch remains a separate unmodeled alternative with its own authority and race semantics.

## Ordinary Control composition

`controlWitness` provisions three programs using the same interpreter. Control sends two jobs; worker `c` is delivered and completes before worker `b`. Each worker receives a job, emits a resource effect, consumes its exact receipt, sends a result, consumes send acceptance, and succeeds. Control consumes its two send receipts, receives both worker results, branches on their sum, and succeeds.

The witness contains **65 events, 20 commits, 6 intents, 6 authorized/applied attempts, 4 message admissions, 6 receipts, and 3 delivered terminal notifications**. Both external mutations occur; Control's aggregate is 2. All three runs replay without changing live state. No transition dispatches on the names Control, Supervisor, or Worker; those names select ordinary program data only.

This demonstrates one complete composition with reordered work, not all possible interleavings or dynamic creation, supervision, group cancellation, or compositional equivalence. It is separate from the single-subject breadth-first searches.

## Defects found during adversarial review

| Finding | Counterexample and initial failing evidence | Correction |
| --- | --- | --- |
| R1 Receipt-tag impersonation | Emit `charge`; admit an ordinary message tagged `receipt:charge`; receive it with zero attempts/receipts. `TestReceiptReceiveRejectsSpoofedOrdinaryMessage` failed because Prepare succeeded instead of rejecting correlation. | Explicit observation kind, two finite expected-intent reference slots, Receive operands, and an independent consumption monitor. Same-tag receipts from distinct intents are also tested. |
| R2a Inbox identity collision | Admit raw ID `receipt:a/g1/n1/i0`; emit/authorize/apply the effect; record its receipt. The accepted ordinary payload was overwritten. `TestOrdinaryMessageCannotAliasReceiptObservation` failed on that loss. | Disjoint internal message/receipt keys and retained-payload validation. The original raw input remains legal and independently receivable. |
| R2b Transport identity collision | An environmental message uses the same raw ID as a generated Send. Admission selected the environmental payload, but Send receipt reported acceptance. `TestExternalRawIDCannotAliasGeneratedSendAdmission` failed because receipt succeeded before its own admission. | Origin-tagged ingress/Send identities, explicit origin selection, and complete descriptor/provenance checks in both transition and monitor. Both colliding raw identities coexist. |
| Acknowledgment coherence | A Send receipt exposed acceptance while its acknowledgment marker stayed false. `TestSendReceiptAcknowledgesAdmission` failed. | Recording Send admission receipt marks acknowledgment observed; environmental ingress acknowledgment remains an independent event. |

The initial tests-first run failed to compile because model symbols did not yet exist; that is recorded as compile-time RED, not evidence that a behavioral invariant detects a defect. The first executable failure was a checker comparison of nil history with an empty prefix; the checker was corrected without weakening the contract. The R1/R2/acknowledgment failures above and the permanent guard-removal controls provide the substantive behavioral falsification evidence.

Independent scope review also required an explicit active-run authorization condition and a fixed replay projection before qualification. Those clarifications, the terminal-disposition simplification, and the explicit exclusion of dispatcher-incarnation fencing are documented in the operational specification. No discovered counterexample was discarded by excluding its input or assuming it could not occur.

## Assumptions and excluded claims

All authority/commit/admission/authorization operations are indivisible model transitions; no distributed implementation is provided. Runtime, configuration, and authority enforcement are trusted. The endpoint's idempotent mode assumes atomic key/mutation handling and retention throughout the finite test. Complete finite history retention is not a production retention policy.

External mutation outcomes cover successful application and missing receipts, with Unknown represented as unresolved evidence. Known mutation rejection/error returns, conflicting authenticated receipts, read-only outcome queries, compensation, and endpoint fencing are not modeled. Send admission may return accepted or rejected. Static edge permissions do not establish general capability security, delegation, or revocation.

Dynamic creation, ownership transfer and cross-partition consent, cleanup custodians, group stop, dispatcher-incarnation fencing, real persistence, version migration, deduplication expiry, composite snapshots, resource exhaustion protocols, arbitrary programs, and unbounded fairness/liveness remain unresolved or excluded. Bounded safety exploration is not an unbounded proof. Notification and ready-continuation witnesses demonstrate possibility, not eventuality under every scheduler.

## Independent review status

An independent reviewer who authored neither model nor specification reproduced R1 and both R2 cases in isolated copies, rechecked their corrections, and independently passed the full tests, targeted regressions, the race check, vet, and formatting check. All 15 evidence records exactly matched the reviewer's fresh qualification run. After reviewing the complete report and decision update, the reviewer returned **APPROVE for Phase 1 research-deliverable readiness**, with no blocking findings remaining. This approves the research deliverables, not mathematical correctness, unbounded guarantees, or production readiness.
