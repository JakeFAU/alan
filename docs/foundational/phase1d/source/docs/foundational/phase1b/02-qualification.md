# Phase 1B recursive composition qualification

Status: bounded research evidence; independent review returned **APPROVE for Phase 1B research-deliverable readiness**. The corrected model passes eleven positive multi-subject searches. Five guard-removal controls and one faulty-program control produce counterexamples. Leaf and nested implementations also complete separate reachability traces against the same declared public contract. This supports those finite examples; it does not establish general recursive closure or contextual equivalence.

## Source and retained baseline

Baseline commit: `f81dd200d05f7c9de62d25fa9baf6c9f6dd08bf9`. Before changes, the full Phase 1 suite passed and all fifteen saved qualification records reproduced exactly. They still reproduce byte-for-byte after this increment. Existing negative controls, policy counterexamples, regression inputs, fixture definitions, and Phase 1 documents/evidence remain unchanged.

The source producing the final evidence has SHA-256 identity:

```text
afc89e4bf24126b33da5341f1bc4c582c81b0b80f51a04af1674daacf0b3c49a
```

The [source manifest](source-manifest.json) lists all ten inputs: `go.mod` and every `research/semantics/*_test.go`, including the existing model and tests. For each sorted repository path, hash its uint64 big-endian byte length, UTF-8 path bytes, uint64 big-endian content length, and raw file bytes, in that order. The manifest also gives each file's byte length and individual SHA-256. Documentation and generated evidence are excluded from the source digest to avoid circularity. The primary agent independently recomputed this digest from the files, and the reviewer received an isolated copy of the complete source at this identity.

The [twenty deterministic evidence records](qualification.jsonl) comprise the source manifest, sixteen original composition searches, the retained faulty-program search, and two complete traces. Each search contains its full configuration/programs, configuration and scenario hashes, seed, event alphabet, limits, rejection reasons, witnesses, and counterexample if any. Source identity accompanies every result. Config/scenario hashes use the exact canonical Go JSON representation emitted by the tests; the full definitions are present, so reconstruction does not depend on a hash alone.

Read the [plan](00-plan.md), [contract and operational additions](01-composition-contract.md), and [research source guide](../../../research/semantics/README.md) with this report. This package adds only test-model code and documentation.

## Reproduction

From the repository root, using the installed Go 1.27.3 toolchain:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test -race -count=1 ./...
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go vet ./...
gofmt -l research/semantics/*_test.go
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off go test ./research/semantics -run '^(TestQualification|TestCompositionQualification)$' -count=1 -v
```

All passed on the final source in the primary agent's independent run; gofmt emitted no paths. Fresh output matched all fifteen old records and all twenty new records exactly. The race check concerns the Go test implementation; modeled concurrency comes from explicit event orderings.

To regenerate composition evidence, preserving pipeline failure:

```sh
set -o pipefail
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1b-go-cache GOPROXY=off GOSUMDB=off \
  go test ./research/semantics -run '^TestCompositionQualification$' -count=1 -v |
  sed -n 's/^.*COMPOSITION //p' > docs/foundational/phase1b/qualification.jsonl
```

The first JSON record contains the manifest. Running the same extraction with `QUALIFICATION ` and `TestQualification` reproduces the unchanged Phase 1 file.

## Experimental boundary

New configurations permit four static subjects, two generations and epochs per subject, twelve commits per run, two continuations and proposal slots, three intents per run, two attempts per intent, eight inbox entries per run, and integers 0–8. Ordinary state has two saved upstream envelopes, two pending request references, and four accepted result keys. Programs are limited to twelve instructions; the actual programs are shorter. These bounds were fixed before implementation and were not increased. Phase 1 retains its original, smaller fixture bounds.

The ordinary request envelope names caller, caller generation, request commit index, logical branch key, and reply address. Trusted transport origin and source generation are separate retained metadata. A result receive matches a pending request and expected peer and records the logical contribution in ordinary projection state. A nested provider simultaneously retains its upstream request and downstream obligation. The same interpreter executes all roles, including the replaceable provider; runtime behavior does not dispatch on role names. Contract configuration identifies public branches and their concrete resource leaves only for the oracle.

There is one fresh-start search. The other searches enumerate suffixes after exact setup traces replayed from initial configuration through both `step` and `inspect`. The longest setup is **108 events**; its suffix bound is ten. These are deliberately focused overlap windows, not exhaustive whole-system interleavings from startup. Seed actions and all subsequent lifecycle actions are environment-injected authorized events. No ordinary supervisor program is demonstrated.

Every attempted successor is checked before deduplication. Identity includes the full machine/configuration, histories, proposals, external world, and monitor facts, hashed with SHA-256; collision resistance is a practical assumption. Rejected actions leave the machine unchanged but may retain path facts such as “an old result was rejected.” These facts allow a later accepted current result to establish a combined witness on the same path and are included in identity. Monitors never supply state to the reducer.

**States** counts unique nonviolating instrumented states, including the seed endpoint. **Accepted** counts accepted attempted transitions, including stutters and a violating successor. **Rejected** counts refused attempts. **Revisits** counts all nonviolating attempted edges that reach an existing instrumented state, including rejected attempts; this differs from Phase 1's accepted-edge-only revisit count. **Boundary** counts unexpanded depth-limit states. **Max** is the largest discovered shortest suffix depth. Seeds are excluded from all these suffix counts. A hard cap of 5,000 states fails the test; a dedicated regression verifies that behavior.

## Positive searches

No monitor violation occurred in these eleven graphs: **192 scenario states, 473 accepted transitions, 665 rejections, 957 revisits, and two unexpanded boundary states**. These totals sum separate graphs; they are not globally unique states. No hard cap was exhausted. Only the fresh-start search reaches its depth cutoff; all other listed finite alphabets reach fixed points before ten. Neither fact extends coverage to omitted events, other programs, or arbitrary recursion depths.

| Scenario | Seed | Limit | States | Accepted | Rejected | Revisits | Boundary | Max |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| fresh_sender_receiver | 0 | 8 | 19 | 29 | 73 | 84 | 2 | 8 |
| inner_wait_peer_ready | 14 | 10 | 5 | 6 | 19 | 21 | 0 | 4 |
| lost_ack_retry_fresh_work | 16 | 10 | 51 | 237 | 222 | 409 | 0 | 7 |
| admitted_result_closure | 38 | 10 | 6 | 5 | 13 | 13 | 0 | 3 |
| old_result_eligible_generation2 | 78 | 10 | 12 | 30 | 30 | 49 | 0 | 5 |
| applied_unresolved_close_retry | 22 | 10 | 37 | 93 | 129 | 186 | 0 | 6 |
| distinct_transport_duplicate | 43 | 10 | 21 | 18 | 66 | 64 | 0 | 5 |
| environment_origin | 38 | 10 | 2 | 2 | 4 | 5 | 0 | 1 |
| nested_admitted_result_closure | 52 | 10 | 6 | 5 | 13 | 13 | 0 | 3 |
| nested_distinct_transport_duplicate | 57 | 10 | 21 | 18 | 66 | 64 | 0 | 5 |
| nested_old_result_eligible_generation2 | 108 | 10 | 12 | 30 | 30 | 49 | 0 | 5 |

The lost-acknowledgment suffix includes receiver abandonment/retry, old retransmission, new admission, actual generation-2 Prepare/Commit, interruption, and old Send receipt/acknowledgment. One retained witness combines original binding, successful new-run computation, and late settlement of the old Send. The old-result cases place old and current replies in the same new-generation inbox; the old reply fails correlation and a current reply actually contributes. Closed inboxes or absent generation-2 instructions cannot satisfy these checks.

## Evidence-to-claim table

| Category | Claim and evidence | Practical limit |
| --- | --- | --- |
| Searched safety | Stable admission binding and closure authorization remain enforced while receiver retry/fresh work/late acknowledgment interleave in `lost_ack_retry_fresh_work`. | Selected alphabet and seed; sender recovery is partly setup, and all recovery is environment-driven. |
| Searched safety | Admitted results can be consumed before closure; after closure a prepared receive cannot commit and its result remains pending. Leaf and nested closure searches cover both orders. | Pending is allowed, not successful handling or eventual delivery. |
| Searched safety | Leaf and nested old replies cannot satisfy an eligible generation-2 request; exact current results can. The generation mutants prove the monitor detects wrong consumption. | Generation-2 request/worker setup is fixed; suffixes search admission/selection/commit order. |
| Searched safety | Two distinct reply Sends with one logical result key cannot contribute twice, while another branch still contributes. Leaf and nested mutants are detected by the same contribution monitor. | A two-branch finite protocol, not arbitrary joins or quorum semantics. |
| Searched safety | Matching environmental payload, envelope, claimed sender, and tag do not establish generated-message provenance. The origin mutant commits the spoof and is detected. | Trusted admission/descriptor enforcement is assumed; no cryptography or general capability system. |
| Searched safety | External application remains visible without a receipt; closure/retry and successor computation can precede late settlement of only the old intent. | Selected successful-mutation outcome profile; no compensation or authoritative external query. |
| Searched safety / falsification | Resource applications must resolve to the declared public branch/leaf and one full root request budget. The retained faulty nested program is rejected on its second unit application. | This is a contract oracle rejecting an invalid ordinary composition, not automatic correctness of arbitrary programs. |
| Successful reachability | `inner_wait_peer_ready` finds a peer Emit commit while the inner provider remains waiting for a result. | Enabledness witness; no fairness or eventual scheduling theorem. |
| Successful reachability | Complete leaf and nested runs satisfy the same correlation, distinct-contribution, causal-result, terminal, and resource monitors; each run replays without dispatch. | Two complete witnesses plus the searched windows above; no trace equality or contextual equivalence proof. |
| Assumptions | Atomic commit/admission/authority/authorization, trusted runtime provenance, retained histories/keys, immutable `v1b` programs, and the configured public mapping. | Modeling these does not implement distributed atomicity, durability, or endpoint fencing. |
| Untested / excluded | Arbitrary depth/programs, general compositional closure, dynamic creation, ownership transfer, cleanup custodians, failure-policy supervision, dispatcher fencing, business-key deduplication across retries, provider/persistence implementation, and unbounded liveness. | No positive count or successful trace resolves these questions. |

## Counterexamples that keep the checks non-vacuous

The five guard mutants use the corresponding positive program, seed, alphabet, limits, and unmodified monitors; only the selected guard changes. Their traces are shortest **suffix** counterexamples for those reachable seeds. Prepend the full JSON seed to reproduce from initial configuration.

| Negative control | Seed | States | Accepted | Rejected | Revisits | Boundary | Detection depth |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| broken_generation-correlation | 78 | 8 | 15 | 10 | 17 | 0 | 3 |
| broken_logical-duplicate | 43 | 13 | 13 | 19 | 19 | 0 | 4 |
| broken_origin | 38 | 3 | 5 | 4 | 6 | 0 | 3 |
| broken_nested_distinct_transport_duplicate | 57 | 13 | 13 | 19 | 19 | 0 | 4 |
| broken_nested_old_result_eligible_generation2 | 108 | 8 | 15 | 10 | 17 | 0 | 3 |
| faulty_program_nested_resource_fanout | 36 | 1 | 1 | 1 | 1 | 0 | 1 |

All use depth limit ten and cap 5,000. Counterexample suffixes:

- **Generation:** admit the provider's generation-1 reply into outer generation 2 → prepare its receive → commit. Normalizing away caller generation admits a stale logical result; monitor reports `composition-generation-correlation`. The nested version has the same suffix after its longer recursive setup.
- **Logical duplicate:** prepare/commit provider reply at commit 4 → prepare/commit its distinct reply at commit 5, carrying the same envelope. Removing accepted-key membership permits two contributions; monitor reports `composition-logical-duplicate`. The nested version repeats this through the nested provider.
- **Origin:** admit environmental `spoof` with the exact legitimate envelope, tag/value and claimed sender → prepare → commit. Bypassing origin/peer checks causes `composition-origin`.
- **Faulty nested program:** after a 36-event valid prefix, apply the second authorized worker effect. Two distinct downstream request IDs share the same full public/root request. The unchanged corrected monitor reports `composition-resource-cardinality`. This changes ordinary program data, not a guard; it is reported separately as a program-fault negative control.

The initial origin regression was behavioral RED against the Phase 1 tag-only matcher: `Prepare rejection = "", want "origin"`. The retained test still supplies that environmental spoof, with exact matching envelope metadata added, rather than excluding it.

## Leaf and recursive comparison

Both use public branches `left` and `right`. In the leaf fixture, concrete resource leaves are `{left: provider, right: peer}`. In the nested fixture they are `{left: worker, right: peer}`, with outer → provider → worker request/reply depth. The mapping affects the checker only. Each provider captures its upstream envelope, observes its own exact unit receipt or a supported downstream result, and sends the computed result through ordinary Send. The outer consumes two distinct branch contributions.

| Complete witness | Events | Commits | Physical applications, including Send | Admissions | Delivered terminal outcomes | Unit mutations |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Leaf provider | 57 | 18 | 6 | 4 | 3 | 2 |
| Nested provider | 76 | 24 | 8 | 6 | 4 | 2 |

Evidence retains full admissions/bindings, contributions, intent descriptors and settlement flags, world operations/keys, and terminal outcomes. Internal message/effect identities and work counts differ visibly. Send acknowledgments need not all be recorded for these complete computations; the record does not claim global settlement. Duplicate reply intents left unauthorized at closure are suppressed. The comparison is conformance to the specified allowed behavior under the exercised schedules, not equivalence inferred from two aggregate values of 2.

## Independent review and retained correction

The initial reviewed source was `58f5205a63cc9bd2e2df2d61451350b9ed1751be25edc966b59ffa6d6fc41240`. Independent review returned **REJECT pending correction and re-review** for R1/P1: the resource-cardinality monitor compared immediate envelopes and therefore missed duplicate resource work introduced by a nested provider.

The reviewer changed only ordinary provider/worker programs within the frozen bounds. Provider captured `outer/g1/n1`, issued downstream requests at its commits 2 and 3, and worker captured/applied one unit for each. All 37 transitions and monitor calls passed, with two mutations for the single public left request and none for right. The complete-run scalar/count assertion did not detect this gap in the search oracle.

`TestCompositionNestedRootResourceCardinalityRegression` reproduced the exact input with behavioral RED: `monitor="", want composition-resource-cardinality`. The correction resolves full root envelope identity through retained immutable request history, preserves generation, rejects ambiguous ancestry as a contract failure, and compares distinct applied effects at that root. The original configuration and all 37 events are retained exactly in `compositionFanoutCounterexample` and the emitted program-fault evidence. No bound increased and no offending input was removed. A separate regression permits distinct root generations to have distinct resource operations.

The independent reviewer re-ran the original probe and confirmed rejection exactly at event 37, the second ApplyExternal. All twenty final evidence records reproduce exactly; the previous eighteen non-manifest records are unchanged except their source identity. The original Phase 1 evidence also remains unchanged. After reviewing the corrected source, this report, and the decision update, the independent reviewer returned **APPROVE for Phase 1B research-deliverable readiness**, with no blocking findings remaining. The reviewer independently passed focused/full tests, race and vet, and verified all evidence records and source identity. The initial REJECT and its correction remain part of this record. Approval applies to the bounded research package, not production readiness or general recursive equivalence.

Before qualification, primary coverage review also expanded the initial search set: receiver lifecycle changes moved into a combined suffix, and closure/old-result/duplicate windows gained nested variants. Those changes add named coverage under the same frozen bounds. They are not a claim that every possible combination has been searched.
