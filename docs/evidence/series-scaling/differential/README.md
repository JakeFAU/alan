# Differential qualification plan and tools

These tools are separate from the frozen timing driver. They use the public
ALAN author/runtime/effects/schema packages. Direct SQLite access is confined
to external Python proof/clone utilities, including explicitly corrupted copies
for negative tests. The application constructs its positive histories through
runtime calls and never edits SQL or creates synthetic authority or ambiguity.

The tools were first built against the immutable baseline and their small
proof campaign passed: four exact baseline-to-baseline eight-position pairs
(success/refusal3, ordinary/reopen-every1), a generic-reader pair from an actual
accepted-response-lost SIGKILL archive, nine rejecting comparator mutations,
and four rejecting clone guards. Staticcheck and Python syntax checks passed.
The frozen candidate then passed all four exact baseline/candidate execution
pairs and generic-reader comparisons on all 21 actual baseline crash archives.
An independent reviewer reproduced all 25 comparisons without this comparator,
and separately read all 21 candidate archives with the baseline binary. Two
supplemental mutations of request key and approval predecessor also reject.
See [qualification](../qualification.md). Do not run these tools concurrently
with timing or profiling campaigns.

## Exact same-namespace comparison

`runtime.Create` generates a random namespace. Independently created stores
therefore have different request keys, prepared/result hashes, resource-root
IDs, and downstream composition fact digests. Blanket omission of those hashes
would conceal semantic differences. The exact qualification instead starts
from one untouched provisioned store shared as immutable input to both source
variants:

1. Build this independent module against each selected frozen ALAN source using
   `build.sh ALAN_SOURCE OUTPUT_BINARY`. It uses a temporary local-replace
   modfile; no absolute replacement is committed. Record source commit/tree,
   changed-source hashes, runner hashes, Go environment, and binary hashes.
2. Use the baseline runner's `create-seed` mode to construct the same
   BoundedSeries definition as the timing driver, call public Create, then
   Close. Public InspectExecution/Series.Inspect/History/Replay must show zero
   effects, charges, approvals, and reductions. No adapter is contacted.
3. Run `clone_seed.py SEED BASELINE_DB CANDIDATE_DB` only on the closed seed.
   It acquires ALAN's existing advisory lock, rejects a nonempty WAL, validates
   SQLite quick_check and no executed history/attempts, rejects existing output
   databases/sidecars, and retains byte hashes. Namespace and retained grants
   are unchanged. Each independent process receives its own copy.
4. Execute the two runner binaries serially against their copies with identical
   schedules/scenarios. Each uses public Open and the ordinary
   InspectExecution.ContinuationBasis -> ApproveContinuation -> Run paths.
   Every cut includes the complete exported snapshot, all History pages,
   Replay report, and SeriesView. Each actual in-process adapter entry retains
   its complete Prepared request and Candidate, including original ID/key.
5. Compare using `compare.py same-seed`. Public arrays, selected observation
   references, authority epochs/revisions, decision order, prepared/result
   digests, scoped grants, root charges, pending/uncertainty evidence, and
   contact order/provenance are exact. The SQLite proof independently compares
   every user schema object and every table's exact typed cells, including
   complete descriptor BLOB bytes. Physical row ordering is sorted; retained
   sequence/order columns are not changed. File/page/WAL-layout equality is
   not required. No epoch adjustment is permitted between identical schedules.

Only `source_label` and the store invocation path are removed from the public
report comparison. No permission, program, operation, request, approval,
observation, result, hash, outcome, or uncertainty is normalized in this mode.
A mismatch stops qualification and reports its first path; do not expand the
normalization allowlist to obtain a pass.

The common seed means this experiment tests Open/execution/read/recovery
behavior against one original provision, while fresh Create behavior remains
covered by the separate fresh-process timing/proof campaign and fresh-storage
compatibility tests. The additional public cut inspections affect performance;
this runner is semantic qualification only.

Illustrative serial commands (choose fresh absolute output paths):

```sh
export GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=4
export GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache
export GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache
./build.sh "$BASE_SOURCE" "$OUT/base-differential"
./build.sh "$CANDIDATE_SOURCE" "$OUT/candidate-differential"
mkdir -p "$OUT/seed"
"$OUT/base-differential" -mode create-seed -bound 8 \
  -db "$OUT/seed/store.sqlite" -source-label "$BASE_ID" > "$OUT/seed.json"
python3 clone_seed.py "$OUT/seed/store.sqlite" \
  "$OUT/ordinary/base.sqlite" "$OUT/ordinary/candidate.sqlite" > "$OUT/ordinary-clones.json"
"$OUT/base-differential" -mode execute-existing -bound 8 \
  -db "$OUT/ordinary/base.sqlite" -source-label "$BASE_ID" > "$OUT/ordinary-base.json"
"$OUT/candidate-differential" -mode execute-existing -bound 8 \
  -db "$OUT/ordinary/candidate.sqlite" -source-label "$CANDIDATE_ID" > "$OUT/ordinary-candidate.json"
python3 compare.py same-seed "$OUT/ordinary-base.json" "$OUT/ordinary-candidate.json" \
  --baseline-db "$OUT/ordinary/base.sqlite" \
  --candidate-db "$OUT/ordinary/candidate.sqlite" > "$OUT/ordinary-comparison.json"
```

Repeat cloning from the original untouched seed into new output paths for:

| Schedule | Runner flags | Required comparison |
| --- | --- | --- |
| ordinary8 | `-scenario success -reopen-every 0` | 8 original contacts/charges/results/selections/completions; 7 approvals |
| recovery8 | `-scenario success -reopen-every 1` | identical intermediate recovery schedule; no extra contacts |
| refusal3, approve4 onward | `-scenario refusal3 -reopen-every 0` | original third refusal retained and selected; separately approved fourth through eighth; 8 spent/selected, 7 completed, and the original refused operation's retained unresolved consequence |
| refusal3 with recovery | `-scenario refusal3 -reopen-every 1` | same retained refusal and no extra contacts across reopening |

The observation program deliberately matches the timing driver's Request/Await/
Success sequence. A refused adapter payload yields a failed SeriesPosition even
though that local program subsequently reaches Success; those are distinct
facts, both retained. The baseline runtime classifies this dispatched nonvalidated refusal as
`Consequence=unresolved` with `ReceiptEvidenceState=none`, even though its local
refusal result is retained and selected. The runner preserves and checks that
original uncertainty (Unresolved=1); it does not turn the refusal into a missing
local result. Actual accepted-request/lost-response ambiguity requires
an externally witnessed SIGKILL case below, not a fabricated Candidate.

Run the same-seed pairs at bound 1 and selected larger bounds if needed for a
changed candidate, scheduling all work after headline measurements. Full eight
position cuts are the primary semantic fixture. This protocol is limited to
one-host controlled adapter execution; it does not establish remote finality.

## Generic crash-archive reader compatibility

`-mode generic-reader -db COPY` uses public InspectExecution, every History
page, and Replay, then Open+Close and the same complete readers again. It never
calls Run, registers an adapter, approves a continuation, or reconstructs weather
configuration. It therefore compares validation and owner renewal against one
actual retained causal history, not execution/resume after the cut.

For every immutable actual SIGKILL archive produced by the existing series
suite, preserve main+WAL and its hash manifest. `clone_cut.py ARCHIVE BASE_COPY
CANDIDATE_COPY` copies both main and WAL without opening the original evidence.
Unlike clone_seed, it supports retained history and nonempty WAL. It requires
that the archive already be immutable and its writer reaped; pre/post hashes
check copying consistency, not live-owner safety. Never point it at an active
store. Each generic reader operates only on its disposable copy.

```sh
python3 clone_cut.py "$ARCHIVE/cut.sqlite" \
  "$OUT/read-base/store.sqlite" "$OUT/read-candidate/store.sqlite" > "$OUT/read-clones.json"
"$OUT/base-differential" -mode generic-reader -db "$OUT/read-base/store.sqlite" \
  -source-label "$BASE_ID" > "$OUT/read-base.json"
"$OUT/candidate-differential" -mode generic-reader -db "$OUT/read-candidate/store.sqlite" \
  -source-label "$CANDIDATE_ID" > "$OUT/read-candidate.json"
python3 compare.py same-seed "$OUT/read-base.json" "$OUT/read-candidate.json" \
  --baseline-db "$OUT/read-base/store.sqlite" \
  --candidate-db "$OUT/read-candidate/store.sqlite" > "$OUT/read-comparison.json"
```

The exact comparator supports both execute-existing and generic-reader reports.
Both copies have the same original namespace, selected observations, grants,
charges, original authorization epochs, and uncertainty. They undergo the same
one Open+Close, so any renewed current activation must also compare exactly.
All final typed SQL cells are exact; no epoch offset is normalized. Contacts in
a generic report are empty because this mode has no execution capability. Any
claim about recovery running new/replayed work remains supported by the separate
real baseline/candidate process suites, not this generic reader.

## Existing random-namespace timing records

`compare.py driver-relations BASELINE.json CANDIDATE.json` compares only the
fields retained by the timing driver. It maps resource-root IDs, request keys,
and prepared/result digests by **original series position and operation ID**,
never by equal sample payload. It keeps original operation IDs, selected
observation IDs/reduction references, approval targets/predecessors,
ScopeDigest/TargetRevision/DecisionOrder, response identities, and aggregate
counts exact. Both original identity maps are emitted for audit.

This deliberately limited mode cannot compare full grants/charges, epochs,
prepared/result preimages, or complete uncertainty objects because those are
absent from timing JSON. Mapping a hash does not verify its omitted preimage.
The producer's successful invariants and Replay remain separate evidence. Do
not use driver-relations as a substitute for the same-seed comparison or crash
qualification. Diagnostic profiles/counters are not semantic-equivalence proof.

## Real SIGKILL cuts and named existing tests

Run the existing ordinary-writer tests on **both frozen sources**, serially,
with `-v -count=1`. They execute the same application command handler. Internal
boundary pauses use acknowledged inherited pipes: PID/token/scenario/store,
subject/generation, phase, and optional sequence/intent/attempt must match.
`Process.Signal(SIGKILL)` is followed by `Wait` and verified Unix wait status.
No sleep-based guess establishes a crash cut. Provider acceptance/lost response
uses the independently observed literal-loopback fixture's accepted original
operation and withheld response. Public run/resume/read/replay commands use an
independently compiled uninstrumented consumer.

| Required boundary | Named existing test/subtest | Actual acknowledged writer cut |
| --- | --- | --- |
| before protected approval commit | `TestWeatherContinuationProcess/before_approval_commit` | continuation.approval / pre-commit |
| after protected approval commit | `TestWeatherContinuationProcess/after_approval_commit` | continuation.approval / post-commit |
| before approval selection | `TestWeatherContinuationProcess/before_approval_selection` | execution.staged / pre-commit, measured approval sequence |
| after approval selection | `TestWeatherContinuationProcess/after_approval_selection` | execution.committed / post-commit, measured approval sequence |
| before successor request commit | `TestWeatherContinuationProcess/before_successor_commit` | execution.staged / pre-commit, measured request sequence |
| after successor request commit | `TestWeatherContinuationProcess/after_successor_commit` | execution.committed / post-commit, measured request sequence |
| after intent authorization | `TestWeatherProcessRecovery/after_authorization` and series `after_fourth_authorization` | intent.authorized / authorized |
| paid reservation before final admission | continuation `after_paid_reservation`; series `after_fourth_reservation` | dispatch.consider / pre-admission |
| final gate before call | continuation `after_final_admission`; series `after_fourth_gate` | dispatch.admitted / pre-call |
| provider accepted, response lost | continuation `accepted_lost_response`; series `third_accepted_response_lost` | independent endpoint accepted original request; response withheld, owner killed |
| response returned before retention | `TestWeatherProcessRecovery/HTTP_response_before_retention` | adapter.response / pre-record |
| result retained before settlement | continuation `retained_before_settlement`; series `third_result_retained` | adapter.result-recorded / pre-settlement |
| settled before consumption | continuation `settled_before_consumption`; series `third_settled` | adapter.settled / pre-consumption |
| before result selection | `TestWeatherProcessRecovery/before_selection` | execution.staged / pre-commit, measured selection sequence |
| after result selection | `TestWeatherProcessRecovery/after_selection` | execution.committed / post-commit, measured selection sequence |
| before terminal commit | `TestWeatherProcessRecovery/before_terminal` | execution.staged / terminal |
| after terminal commit | continuation `after_terminal`; series `third_terminal` | execution.committed / terminal |
| repeated interrupted owners | `TestWeatherProcessRecovery/two_interrupted_recoveries` | retained -> selected -> terminal staged; strictly advancing owner activation |
| request transaction atomicity | `TestSIGKILLExecutionCommitBoundaries` in internal/executor | acknowledged transaction boundary and actual SIGKILL |

`TestWeatherSeriesProcess` adds eight-position context and checks prior original
effects remain equal, rejects an unpaid out-of-order fifth approval, preserves
lost third/fourth uncertainty without refund or duplicate contact, and requires
separate later approval. `after_fourth_reservation` distinguishes known no-final-
admission from gate-admitted unresolved work; do not flatten these into one
unknown state. It exports snapshots using `ALAN_SERIES_EVIDENCE_DIR`, preserving
both main DB and retained WAL where present. Logs contain `SERIES_CONTROLLED`,
`WEATHER_CONTROLLED`, and continuation records; retain complete raw test output,
barrier witnesses, provider contacts, and stopped/final SQLite archives.

Example commands, executed from each selected source root after lock release:

```sh
ALAN_SERIES_EVIDENCE_DIR="$OUT/series-artifacts" \
  go test -run '^TestWeatherSeriesProcess$' -v -count=1 ./examples/weather-agent
# The following two suites also include repeated recovery/accepted-lost cases.
go test -run '^TestWeatherContinuationProcess$' -v -count=1 ./examples/weather-agent
go test -run '^TestWeatherProcessRecovery$' -v -count=1 ./examples/weather-agent
go test -run '^TestSIGKILLExecutionCommitBoundaries$' -v -count=1 ./internal/executor
```

Independent crash runs have distinct namespaces and loopback ports. Compare
original logical position/operation, grant scope, paid charge, approval and
selection relationships, retained/unknown disposition, and exact contact counts.
Retain each run's raw hashes and epoch values; never discard them from evidence.
Epochs may differ between *different* owner schedules, but must match between
the same declared schedule; within one history, recovery must preserve original
historical epochs and advance only current eligible activations. The exact
same-seed comparator intentionally refuses random-namespace crash archives;
existing per-case causal assertions remain the primary crash oracle. If full
cross-variant crash equality is required, extend those controllers to use one
shared seed and a fixed provider endpoint rather than masking hashes or epochs.

## Additional compatibility and adversarial qualification

Run focused existing tests under both sources, preserving exit status/raw logs:

- `go test -run '^TestWeatherSeries(Eight|OriginalAndLegacyPrograms|Corruption)$' -v -count=1 ./examples/weather-agent`
  exercises the independent eight-read CLI consumer, retained prior responses,
  historical non-series lowering, and corruption of position/generation/slot/
  approval/scoped grant/root allowance/selected observation relations.
- `go test -tags=integration -run '^TestContinuation' -v -count=1 ./runtime`
  covers authentic foreign-store basis, wrong predecessor/target/generation,
  foreign prepared/result digests, protected-ingress forgery, substituted action,
  stale basis before approval, retired authority, and one approval/one operation.
- `go test -run '^Test(ExclusiveOwnerAndAliases|StaleAuthorityRejected|ReadOnlyReplaySettingsAndNoProvisioning|CorruptionRejectedBeforeEpochAdvance|Continuation|DispatchAdmissionRequiresOriginalOwner)' -v -count=1 ./internal/executor`
  checks exclusive owner/aliases, stale authority, read-only Replay, atomic
  rejecting Open, preserved approved cuts, and dispatch ownership.
- `go test -run '^TestHistoricalReplayAndArtifactEncodingCompatibility$' -v -count=1 ./runtime`
  plus `TestExecutionInspectionHistoricalSnapshots` in internal/executor retain
  historical semantics/encoding/read-only behavior.
- The prior independent module at
  `docs/evidence/bounded-series/performance/module` has
  `TestHistoricalArchivesRemainReadOnly`, `TestFreshStorageFormats`,
  `TestRetainedSeriesFormats`, and `cmd/base-reader`. Follow its reproduce script
  for fixture/ALAN_SOURCE/series tag setup rather than invoking an empty fixture
  test. The base-reader opens only **disposable copies** because successful Open
  may renew authority. Verify old generic runtime reads new format11/12 stores
  and consumes no provider registry; historical archive readers preserve bytes.

No `func Fuzz...` target exists in author/runtime/internal/examples in the source
inspected for this plan. Existing mutation tables are adversarial regressions,
not a completed fuzz campaign. If fuzzing is required, add a measurement-only
external/overlay fuzz target with fixed valid-seed copies and bounded mutations
of basis identity/scope/selection fields, require public reader rejection and
byte-preserving failed Open plus zero new contacts, retain the exact seed and
crash corpus, then run its explicitly named target serially. Do not claim a
fuzz result from `go test -fuzz` without an actual discovered target.

## Validation still required before use

After measurement lock release: build both binaries; run seed/clone plus the
four schedules; run comparison against both correct pairs and deliberate
negative copies (changed contact key, selected observation, scoped grant, paid
charge, approval predecessor, epoch, and unresolved marker). The comparator must
reject each changed field. Verify clone helper refuses live locks, nonempty WAL,
existing targets, and executed seeds. Run Python syntax checks and Go
Staticcheck with temporary selected-source modfiles. Preserve exact commands,
inputs, outputs, hashes, and test statuses. No qualification claim follows from
source review alone.
