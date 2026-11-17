# Independent finite Outing protocol

This source-only model is authored from the [delegated Weather protocol](../../../docs/superpowers/specs/2026-11-17-delegated-weather-design.md), independently of the Go programs, compiler, reducer and evidence verifier. It establishes bounded graph results. The companion evidence reader reconstructs the declared protocol from a complete schema9 Store and checks a compatible model path. Process-kill recovery and independent endpoint-contact qualification remain separate work.

Run with the Python standard library:

```sh
python3 -I -B examples/outing-planner/model/test_model.py
python3 -I -B examples/outing-planner/model/run.py enumerate --output /tmp/outing-model.json
```

`run.py` captures `check.py` and `evidence.py` before executing either. It hashes and compiles those exact immutable byte buffers into fresh namespaces, injects the model API explicitly, and performs no filesystem import or local module-cache lookup. `source_sha256` keeps its original meaning (`check.py`); `source_files` identifies both named buffers. `source_set_sha256` hashes UTF-8 canonical sorted name-to-hash JSON prefixed by `outing-source-set/v1` and one NUL byte. Missing or unexpected executable inputs reject. Unexecuted sibling test files are not source-set members.

Output also records the captured launcher hash, interpreter/SQLite versions and reproduction command. Launcher integrity, interpreter/standard library, frozen manifests and capture/source integrity are trusted. Tests replace both source paths after capture and verify the captured bytes execute; they also exercise missing/extra executable inputs. Direct execution of `check.py`, nonisolated launcher execution and unknown commands reject.

## Domain and interpretation

The campaign enumerates both `broad` and `preclose-authorization` domains; candidate counts 0, 1, 2; all vectors of `suitable`, `unsuitable`, `unavailable`; every subset of originals chosen to have no later retained result; direct/delegated variants; and helper allowances 0/1. Direct allowance settings are repeated input cases with identical behavior, rather than distinct runtime resource configurations. The separate fixed confirm-first control has two candidates (`suitable`, `unsuitable`), matching the preserved legacy cycle fixture.

Candidate indices refer to already normalized frozen order. This model does not prove calendar normalization or native provider parsing. All candidates are assessed in order, with one original per candidate, sequential selection and capacity one. A suitable candidate wins in frozen order even when another candidate is unavailable. With no suitable candidate, any unavailable result gives diagnostic `unavailable`; otherwise the answer is `no_match`, including n=0. Delegation invokes one helper even at n=0. Allowance zero is a real retained invocation refusal, with no member, forecast contact, debit or successful mark.

A **known** fixture provides exactly its declared original response. A selected **no-result continuation** has no later retained result and no retry; later candidates remain unstarted when an earlier one is blocked. No-result masks on unreachable later candidates remain inputs, not invented operations. A dispatched original without a durable result is also a reachable prefix under the known-response configuration: raw absence alone is an epistemic gap, not proof that an authentic response could never arrive. A raw correspondence consumer must preserve this ambiguity and may not choose the permanent-no-result continuation solely from an absent row. Symbolic dispatch/contact counts in this model are not evidence of actual endpoint contact; concrete contact claims need independent process/endpoint evidence.

`broad` permits End before first authorization of Outing's decision. The first refusal is immutable, its refused admission creates no observation, and its acknowledgment does not consume the required decision. Initial completion may still be possible while nonblocking fails. `preclose-authorization` restricts ordinary sender End until its committed public messages have allowed first authorization. It does not collapse authorization, admission, acknowledgment or selection. Bound helper replies can first authorize after helper End only for the exact original admitted binding. Frozen committed replies block publication until admission. The post-End exception does not apply to public messages.

Invocation publication, accepted result copy (accepted calls only), outcome observation and delivery link form **one transition**. Refused calls can publish/deliver a correlated outcome without a member or accepted result copy. Parent selection is later. There is no durable publication-without-delivery cut. Member End/freeze and root End/freeze are atomic model boundaries; root publication remains later. Root settlement debt excludes a known rejected invocation, while caller publication/selection and Outing receive debt remain separately visible. Bound reply recording and transport acknowledgment remain distinct even where public Pending diagnostics may treat the recorded reply as acknowledged.

The model assumes fixed generations/revisions, permissions, schema-valid messages, one helper/binding, one transport identity per original, no retirement, reconciliation, external operator input or additional invocations. The wider runtime can legitimately use multiple transport identities; exclusion here is a finite-domain boundary, not a corruption theorem. Repeated same-identity message service is idempotent. No fairness is assumed for existence of finite completion suffixes; no claim of inevitable completion under an arbitrary scheduler is made.

## Results and costs

Every reachable state is checked for original identity, causal prefix, exact binding, immutable authorization disposition, result provenance, complete candidate coverage, local control, frozen obligations, atomic retained boundaries, ledger balance and capacity. Breadth-first exploration retains a shortest path to each state. Reverse reachability from successful marks computes `CanComplete`; nonblocking means **every reachable state** has some finite successful suffix. A mark requires recommended/no-match, all required observations consumed, all transport acknowledgments retained, all accepted work discharged, both local successes and both root publications without debt. Quiescence and a copied final answer are insufficient. Fully completed unavailable computations are counted separately as diagnostic terminal states. Refusal and unknown branches are neither successful planning nor the confirm-first cycle.

Each JSON case reports actual state/edge/mark counts, initial possible completion, nonblocking, semantic answer sets, successful root charges and counterexample traces/debts. The shortest nonblocking witness can end at Outing End before the denial is recorded. Separate shortest `source_refusal` and `source_refusal_acknowledged` witnesses show the later retained reason and refused service. The explicitly ordered `allow_before_end_admit_after_end` paired control is labeled as a control, **not** a shortest witness. Empty nonblocking traces are legitimate when even the initial state has no successful suffix.

Semantic answer sets are compared only under adequate admission and known fixture outcomes, separately by scheduling domain. Empty equal sets for unavailable outcomes do not count as successful replacement. Fully successful direct/delegated root debits are `(0,0,n)` and `(1,1,n)` (invocations, created runs, effects); delegation also adds one depth level and a bound reply. Ordinary messages incur no invocation/effect debit; their original commitments and service debts remain explicit. This is not exact-debit bisimulation, general congruence, an application of the research continuation-monitor theorem, universal runtime refinement, or proof of live provider correctness.

## Stable API for raw correspondence

`check.py` owns the finite protocol API. `evidence.py` receives that API from the captured-source launcher; it does not import the model from a filesystem/module cache.

- `Config(variant, outcomes, allowance=1, domain="broad", no_result=())`: immutable validated configuration. Outcomes and no-result indices are tuples; indices are sorted unique. Unsupported variants/classes/domains/counts/allowances/masks reject.
- `initial(config) -> State`; frozen `State`, `Message`, `Forecast`, `Invocation` dataclasses expose original identities, local control, inbox selection, source dispositions/snapshots, admissions, acknowledgments, forecast stages/results, invocation binding/member/request/freeze/publication/selection, assessment rows, decision, charges, contacts and root freezes/publications. Forecast stages are 0 absent, 1 commit, 2 authorize, 3 reserve, 4 dispatch, 5 result, 6 settle, 7 select. Absent originals are declared symbolic slots, not retained operations.
- `Event(stage, operation, actor, target="", candidate=-1, binding="", disposition="")`; `successors(config, state) -> iterable[(Event, State)]`. Message stages are `message.commit`, `.authorize`, `.admit`, `.acknowledge`, `.select`; forecast stages use the above names; invocation stages are `invocation.commit`, `.authorize`, `.admit`, `.request.select`, `.publish_deliver`, `.outcome.select`; termination/publication stages are `helper.end_freeze`, `root.end_freeze`, `root.publish`. Admission disposition carries accepted/refusal reason; source disposition is allow/deny. The helper binding is `binding:invocation:weather:assess`; original forecast IDs encode owner and candidate. Map concrete original IDs injectively to these symbols.
- `advance(config, state, event) -> State` accepts exactly one enabled event. `validate_transition(config, before, event, after)` rejects mutations, dropped costs and altered obligations. `repeat_service` recognizes exactly retained same-identity message authorization/admission/ack events without recomputing first dispositions.
- `validate_state`, `validate_mark`, `debts`, `root_debts`, `state_view` expose independent checks and separately inspectable caller/transport versus root-descendant debt. `state_view` adds original charge records, candidate cursor, remaining budgets and depth. `policy` returns `(outcome, candidate_index)` with `-1` for no selection. `successful_mark` and `diagnostic_terminal` distinguish successful planning from diagnostic completion.
- `explore(config) -> Graph`: states in BFS discovery order, edges as lists of `(Event, target_index)`, shortest predecessor links, successful marks, diagnostic terminals and reverse-reachable `completing` set. `Graph.trace(index)` returns the shortest event path; `graph_summary` returns serializable counts/verdicts/witnesses. `configurations` declares the complete finite campaign; `enumerate_models` runs it.

Events represent causal boundaries, **not fabricated observed timestamps**. A raw reader must account for complete recorded events and search for a legal model path consistent with retained global fact order plus causal links. Ordinary admissions/acknowledgments and adapter settlements do not supply their own retained total chronology. An End/freeze or invocation publication/delivery can account for multiple linked rows/facts at one atomic model boundary. The evidence reader uses this API for concrete correspondence within the declared fixture domain.


## Complete raw Store checks

```sh
python3 -I -B examples/outing-planner/model/run.py check \
  --store cut.db --configuration derived-configuration.json \
  --fixture fixture.json --manifest deployment-manifest.json \
  --report planner-report.json --output checked.json
python3 -I -B examples/outing-planner/model/run.py compare \
  --baseline baseline.db --candidate candidate.db \
  --configuration derived-configuration.json --fixture fixture.json \
  --manifest deployment-manifest.json --replacement replacement.json \
  --output compared.json
```

`--report` and `--replacement` are optional coverage cross-checks. Raw Store records determine the verdict. A failure writes `accepted: false` and its invariant witness, then exits nonzero. The compare command independently checks both complete Stores before comparing semantic answers, candidate-aligned original forecast IDs and separate actual costs. Equal answers with pending obligations never count as successful replacement.

The reader opens the source with `mode=ro`, enables query-only, and enumerates the schema and every row in **one read transaction**, including committed WAL frames. It pins physical schema9, all 42 source tables, exact schema objects/DDL/columns/primary keys/foreign keys/indexes, and rejects extra objects. Unsupported lifecycle, external-receipt, asset and closure categories must be empty. Every retained row receives an explicit structural or semantic rule; a nonempty unclassified table fails. Complete `programs` inventory is matched against the selected deployment's roots and helper prototype, including inert rows that public snapshots cannot enumerate.

Inventory encodes each SQL cell as a one-byte type (`N`, `I`, `F`, `T`, `B`), an unsigned eight-byte big-endian byte length, and its payload. Integer payloads are ASCII decimal, real payloads are IEEE-754 big-endian doubles, text is UTF-8 and BLOB payloads are unchanged. Row hashes prefix this concatenation with `outing-raw-row/v1` plus NUL. This distinguishes SQL NULL, empty text, empty bytes and numeric/string values. `raw_cut_sha256` hashes the sorted table-to-ordered-row-hash map with `outing-raw-cut/v1` plus NUL. Schema objects and metadata are also included in output; these inventory hashes are separate from source protocol hashes.

Reconstruction covers contiguous complete histories, original-operation bijections, observation selection, first source authorizations, all message transports and raw acknowledgments, accepted/refused invocation groups, helper bindings/requests/replies, terminal freezes, atomic outcome publication/delivery, original adapter reservations/dispatch gates/native fixture results/settlements and ordered charges. Observable operations, receives and End cannot be hidden as pure steps. Native fixture response fields and suitability are independently derived. The trusted canonical program/input manifest and frozen normalization remain explicit inputs; this is not an independent compiler, complete bytecode, calendar or arbitrary-provider proof.

Global composition facts retain order. Ordinary admission, acknowledgment and adapter settlement have only causal constraints. The output supplies one **compatible model placement**, never an invented observed total chronology. First authorization's source sequence is checked at its actual source projection/global prefix, separately from the original intent sequence. Broad-domain late first decision authorization is valid incomplete evidence. Allowed-before-End/admitted-after-End remains distinct and can complete. A committed frozen helper reply blocks publication until its original admission.

Every historical selection must match the exact receive tuple resolved from its actual predecessor, including provenance, source/generation, obligation and original correlation under the helper binding. After matching the observable path, the reader relates each final concrete continuation to the model phase. It walks a fresh projection copy through only the fixed artifacts' pure `set`, `branch`, `jump`, `foreach`, `next`, `loop` and `append` instructions, stopping before the next receive, emit or End. Unresolved operands, unsupported instructions and paths exceeding 4,096 pure steps reject. The next boundary's exact tuple, original correlation or emit payload must match the model phase, and a separately raw-derived debt ledger must equal the model debts. Output retains this `final_continuations` relation, each pure PC path and `concrete_debts`. These checks always run, including without an optional report. This interprets the supported pure continuation slice; it does not replay all historical pure updates or prove general VM/compiler correctness.

Outputs separate successful completion, diagnostic unavailable completion, refusal, unknown forecast, confirm-first cycle and pending cuts. They expose complete symbolic debts, original mappings and `raw_runner_message_debt`. Public Pending diagnostics treat recorded replies as acknowledged; raw runner debt still requires `message_acknowledgments`. A known invalid fixture response can retain the runtime's `unresolved` consequence classification despite a durable diagnostic result; this is distinct from missing-result uncertainty.

`decision_commit_cut.verified` checks the complete captured decision-commit cut, including both open endpoints, exact matching Weather receive, no first decision authorization and the original decision as the sole remaining internal message delivery. A later final Store cannot establish that historical premise; attribution requires separately preserved capture/source integrity and prefix binding. Raw dispatch admissions prove gate passage only. No endpoint-contact or owner-kill claim is inferred here.

## Capturing and testing

Capture while the source cut and WAL are available:

```sh
python3 -I -B examples/outing-planner/model/test_evidence.py capture SOURCE.db FRESH-CUT.db
```

The helper opens a read-only source transaction, uses SQLite backup, switches **only the fresh destination** to DELETE journal mode, and compares every source/destination typed row and complete schema metadata. `FRESH-CUT.db.capture.json` records equality, the source inventory and destination byte hash. Existing destinations reject. The checker never repairs a source or uses `immutable` against active WAL. A lone main-file copy without its required WAL is not a complete capture.

```sh
python3 -I -B examples/outing-planner/model/test_model.py
python3 -I -B examples/outing-planner/model/test_evidence.py
ALAN_RAW_CAPTURE_DIR=/tmp/outing-raw-captures go test -tags integration \
  -count=1 -run '^TestDelegatedRaw' -v ./examples/outing-planner
OUTING_RAW_FIXTURE=COMPLETE-PAIR-DIR OUTING_MUTATION_DIR=/tmp/outing-mutations \
  python3 -I -B examples/outing-planner/model/test_evidence.py CompleteEvidenceControls
OUTING_RAW_PENDING_FIXTURE=UNKNOWN-PAIR-DIR \
  python3 -I -B examples/outing-planner/model/test_evidence.py PendingReportControls
OUTING_RAW_CUTS=RETAINED-CAMPAIGN-DIR OUTING_MUTATION_DIR=/tmp/outing-mutations \
  python3 -I -B examples/outing-planner/model/test_evidence.py ContinuationControls
```

Without capture variables, source-only inventory/loader/model tests run and capture-dependent tests explicitly skip. The Go harness freezes configuration, fixtures and artifact manifest **before** Create, exercises real public-runtime service orders, captures complete cuts and invokes Python. It supplies no correspondence verdict. `ALAN_RAW_CAPTURE_DIR` retains uniquely named scenario directories with configuration/fixture/manifest, original Stores, complete captures, optional reports and checker JSON. Mutation controls operate only on fresh copies and retain mutation logs under `OUTING_MUTATION_DIR`. The process-kill/reopen campaign, endpoint acknowledgments and final archival qualification remain separate from these in-process fixture controls and finite graph enumeration.

## Genuine process and public CLI qualification

```sh
ALAN_RAW_CAPTURE_DIR=/tmp/delegated-process go test -tags integration \
  -count=1 -timeout=60m -run '^TestDelegated(Process|Qualification)' -v ./examples/outing-planner
ALAN_RAW_CAPTURE_DIR=/tmp/delegated-process-race go test -v \
  -count=1 -timeout=60m -race -tags integration ./...
examples/outing-planner/testdata/delegated-walkthrough.sh
```

The final campaign consolidates focused process/CLI race coverage into the
verbose full integration/race run above; its log records those named tests.

The process test acknowledges an existing runtime barrier in a child owner,
kills that exact PID with SIGKILL, checks the retained cut independently, and
resumes with a fresh public CLI process. Helper identities come from the same
Store, never a baseline namespace. The withheld-response test separately binds
an endpoint-acknowledged original GET to its dispatch attempt and keeps the
endpoint alive through two fresh resumes and all read-only readers. No extra
contact, replacement helper, later candidate, invented receipt or hidden debt
is accepted. The CLI qualification compares actual replacement exports with
independently checked baseline/candidate Stores. Raw immutable-row and original
charge/fact prefixes are compared across process recovery; legitimate later
work may add its first charge.

The compact source-bound archive and reproduction instructions live under
`docs/qualification/delegated-weather/`. Process/contact evidence, freshly
captured raw controls, older rechecked cuts and finite enumeration are distinct
result categories. Capture integrity, canonical artifacts, normalization, the
single-host runtime/SQLite, OS, interpreter and standard library remain trusted.

Actual accepted/refused CLI exports are exercised separately from synthetic
export mutations. After a captured process/CLI campaign, run both real pairs
and all eleven adverse export variants with:

```sh
OUTING_REPLACEMENT_FIXTURES=/tmp/delegated-process \
OUTING_MUTATION_DIR=/tmp/delegated-export-controls \
  python3 -I -B examples/outing-planner/model/test_evidence.py OptionalReplacementControls
```

The root must contain exactly one `cli-replacement-accepted-*/pair` and one
`cli-replacement-refused-*/pair`. Optional original-ID keys appear only when a
real original exists; explicit empty keys, hidden/substituted originals, extra
rows/fields, changed order and changed costs reject. Full fixture execution sets
this variable alongside the raw complete/unknown/cut variables above. Without
those captures, fourteen methods are discovered, five source-only methods run
and nine capture-dependent methods explicitly skip.
