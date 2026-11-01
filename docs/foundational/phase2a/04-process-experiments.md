# Process evidence: retained knowledge and independent world

Question: can the exact retained local state survive a real executor loss while the independent world's history differs, and which actual continuation follows on reopening? These are bounded tested results at one host, one original operation, one physical worker slot, one SIGKILL and one fresh executor per branch. They are not an arbitrary-context equivalence or a distributed recovery proof.

## Source and reproducibility

Final command records are `evidence/commands/process-normal-reviewfix.json.gz`, `process-race-reviewfix.json.gz`, and `process-sql-audit-reviewfix.json.gz`. Each gzip file is a JSON object with exact executable/hash, argv, cwd, explicit environment, source-file input hashes, output and exit code. All three commands exited zero. Normal duration reported by Go was 8.597s; race duration 29.217s. Both use Go 1.27.3, CGO, readonly cached dependencies, no network dependencies, and an approved loopback sandbox escalation.

The command selector is:

```sh
go test [-race] -tags integration ./internal/durabletest \
  -run '^TestPhase2(PairedCrashCuts|KnownEvidenceCrashCuts|EndpointRestartBindingAndHistoricalReceipt|AdapterRealErrorAndSelectedUnsuccessful|ExecutorProcess|ParentAdapterCrash|EndpointLegacyPaddedJSON|EndpointLegacyMethodRejectBeforeBody)$' \
  -count=1 -v
```

`record.py` supplied `GOTOOLCHAIN=local`, `CGO_ENABLED=1`, `GOPROXY=off`, `GOSUMDB=off`, `GOFLAGS=-mod=readonly`, and `/private/tmp/alan-phase2a-gocache`. Exact values and the complete source manifest are in the records. At the post-run comparison, all captured Go/C/dependency inputs still matched. The workspace-wide command inventory also included `tools/audit_observers.py`, which changed separately during qualification; that independent audit source is not an input to either process binary and must be qualified by its own subsequent command record. Individual start records additionally bind actual process PIDs, argv, inherited barrier configuration, binary hashes, Go build info, and source manifests. The executor is the same test executable freshly exec'd with a single helper selector; its helper calls public `runtime.Open` and `Store.Run` with the ordinary adapter registry. In the race campaign this executor binary and the separately built `alan-resource` binary both contain race instrumentation.

Final evidence is in `evidence/process-normal/` and `evidence/process-race/`. Raw databases are coherent SQLite backups compressed as `.sqlite.gz`, not edited database rows. The record fields retain original absolute execution paths as provenance. To open artifacts after checkout relocation resolve every referenced archive by its basename within the record's containing directory. Read decompressed WAL-header backup images through SQLite `mode=ro&immutable=1`; `audit-process.py` demonstrates this without rewriting any header or authoritative store.

## Paired cuts

For each route, a single provision was safely closed and copied before execution. Both branches retained its exact namespace, program bytes, original operation, prepared request, endpoint target URL, generation and epoch. There was no peer in this comparison. U0 started its own empty world, crashed and recovered while that same endpoint process remained alive, then the endpoint stopped. U1 started a distinct initially empty endpoint store at the identical URL before its executor started; it remained alive through U1's crash/recovery. Recovery of U0 could not preapply U1's world. Neither world was reset during recovery.

U0 acknowledged the route's `*.authorized/pre-transport` worker boundary, and SIGKILL occurred before Execute/HTTP transport. U1 acknowledged `*.response/pre-record` after real endpoint commit and authentic response acquisition, then SIGKILL destroyed the volatile result. The independent endpoint distinguishes application zero/one. The complete local `Inspection` objects, and independently decoded raw SQL tables, are exactly equal between cuts for all three pairs: no epoch, field, identity, request or row was erased or renamed. This equality concerns the recovery cut; later endpoint evidence may legitimately distinguish histories.

| Route | Applications at U0/U1 cut | New physical valid requests on recovery | Applications after recovery U0/U1 | Actual consumer |
|---|---|---|---|---|
| Raw idempotent | 0 / 1 | 1 / 1 | 1 / 1 | Both succeed with retained receipt |
| Raw non-idempotent | 0 / 1 | 0 / 0 | 0 / 1 | Both remain at original Await |
| Ordinary adapter over idempotent endpoint | 0 / 1 | 0 / 0 | 0 / 1 | Both remain at original Await |

Each crash has a validated acknowledged PID/event, SIGKILL exit attestation, distinct fresh PID, epoch renewal, binary fingerprints, before/after raw local and world stores, ingress journal, result/settlement tables and actual consumer projection. The raw idempotent U1 resubmission preserves the original exact receipt, including original endpoint process and PID. The conservative routes preserve uncertainty and withhold eligible physical repetition by policy; their safety sample does not establish useful completion.

Member effect expenditure is not present in these unscoped pairs. Original attempt/authorization rows remain one per operation, with no reservation duplication. Physical resubmission and fresh per-Run step/dispatch expenditure are separately visible in subprocess output; equal durable cuts do not assert equal future schedules or expenditure. Parent member cases retain one effect reservation/charge in their raw stores.

## Known result cuts and actual selection

Raw and adapter routes each have two additional real SIGKILL cases: result row committed with zero settlements, and result plus settlement/observation committed before selection. Raw pre-settlement evidence uses the new passive `external.result-recorded/pre-settlement` seam. Adapter uses its pre-existing `adapter.result-recorded/pre-settlement` seam. These callbacks occur after result transaction commit and Store mutex release. The settlement is a distinct transaction. The post-settlement worker accounting seam blocks the scheduler before it can select that observation.

Fresh execution uses the retained result without another endpoint contact, completes settlement when needed, selects the settlement observation exactly once and succeeds. Raw SQL decoding checks the actual selected observation identity; a result-row count alone is insufficient. There are four such cases per normal/race campaign.

An ordinary owning parent adds two process cases. `TestPhase2ParentAdapterCrash/unknown` kills the invoked child's reserved worker before Execute. Fresh recovery leaves root open with `knowledge=may_have_applied`, no selected child outcome, one adapter attempt, zero result and zero endpoint application. `.../recorded` kills after the child's genuine result commit and before settlement. Fresh recovery settles and consumes it, the child replies through its ordinary invocation contract, and root succeeds with `knowledge=historical_application_receipt`, without another contact. These cases use no independently enabled peer; the separate public parent probe covers peer progress at capacity one.

## Result knowledge and world knowledge

Four adapter result-mode samples each make a real durable remote application. Success produces a validated historical application receipt. `selected-error` deliberately returns an unsuccessful candidate after receiving authentic success. `unknown` returns no persistable result and leaves the consumer open. `drop-after-commit` uses an actual hijacked HTTP connection closed by the endpoint after its application commit; Execute observes a real transport failure and retains `transport_error`, then the guarded consumer fails. Neither the selected unsuccessful candidate nor the transport error establishes remote nonexecution. This transport experiment is separate from losing a volatile response to SIGKILL.

## Identity, endpoint trust and historical receipts

The endpoint restart control sends an idempotent operation, SIGKILLs only the endpoint process, restarts the same store/address, and repeats the exact request. The original exact receipt survives. An altered delta under the same key receives actual HTTP 409. A new business-attempt key applies again. A later repeat of the old key still returns total 5 while the current world total is 10: historical receipt knowledge is not present resource-state knowledge. The journal records five HTTP ingress requests; only four valid contacts commit in the resource ledger, with two applications. This distinguishes physical contact, valid contact and application. No read-only reconciliation capability is provided or claimed.

For adapter wire traffic, public prepared identity is its original Request.Key. Private raw-executor authorization/epoch fields are zero, not fabricated. The fixture binds the complete decoded request, including those zeros. The adapter receives only the endpoint's declared HTTP interface; it does not read its ledger, journal, process barriers or oracle snapshots. Endpoint-only duplicate/restart controls establish endpoint behavior, not an automatic runtime adapter retry.

## Replay and second decoding

Each final normal/race campaign contains 28 replay checks. Each archives raw local and world SQLite backups both before and after replay, and exact ingress journal bytes. The second decoder compares every SQL table before/after plus ingress bytes; no valid, invalid or conflicting endpoint contact is hidden by the valid-contact count. Replay invokes the runtime's shared reducer and is not independent semantic validation.

`audit-process.py` uses only Python standard-library SQLite, gzip and JSON. It independently decodes raw table bytes, compares complete U0/U1 tables, reconstructs world applications/totals and original request/key bindings, checks actual result/settlement selection, verifies restart/interference, and checks the coverage count. It reports per campaign: 6 paired branches, 3 pairs, 4 known crashes, 2 parent crashes, 28 replays, 4 result modes, 1 endpoint control. Its author is the apparatus implementer; this is a second mechanism, not independent-author approval. Independent source/boundary review remains a separate qualification obligation.

## Preserved narrower and rejected evidence

`process-normal-v1-narrow.log.gz` and `process-race-v1-narrow.log.gz` preserve earlier passing runs before ingress-byte replay comparison and raw replay snapshots were added. They do not establish the stronger final replay result. Their original raw archives remain outside the PR at `/private/tmp/alan-phase2a-narrow-v1/`; final authoritative raw copies are in the checked-in phase evidence. The first sandbox run failed at loopback bind before any scientific schedule; `process-rejected-sandbox.log.gz` preserves that failure, including accidentally broad selection of authority tests. It is not a counterexample. `process-audit-rejected-deserialize.log.gz` preserves a checker defect: in-memory SQLite deserialize could not read WAL-header backups. The decoder now reads immutable temporary files without modifying the archive, and both full final directories pass.

Remaining limits: trusted endpoint and dispatch, literal local HTTP, finite fixture deduplication retention, one crash per selected branch, and no hidden in-flight packet at the paired cut. The tests establish bounded behavior, truthful local ambiguity and conditional raw-idempotent progress. They do not establish fairness under arbitrary schedules, stronger non-idempotent reconciliation, unlimited deduplication, general parent knowledge, or effectful compositional equivalence.

## Independent review correction and final archive identity

Independent review rejected the initial endpoint instrumentation because default unjournaled operation read bodies before method/path rejection and imposed a new complete-body bound. The rejected endpoint source is preserved exactly in `evidence/rejected-endpoint-source-v2.go.txt`, SHA256 `ba0660a42d28a75adf643223e84c71aaadba04ac2fb76869bce014b2941c8e11`, matched against its earlier command input manifest. The default branch now uses the original streaming decoder, immediate route rejection and no contact mutex. The explicitly selected journal profile retains a complete bounded body; it is a stated experimental profile.

Both new actual-HTTP regression leaves pass in normal/race qualification: `TestPhase2EndpointLegacyPaddedJSON` accepts the original valid JSON plus padding beyond the historical 1MiB decode window; `TestPhase2EndpointLegacyMethodRejectBeforeBody` obtains 404 despite a declared body that was never sent. The full final selectors contain 16 substantive leaf cases plus the inert helper selector. There are no skipped cases. The full normal/race campaigns were rerun after this source correction. Earlier `process-*-final.json.gz` commands remain preserved as rejected-source captures, not qualification of the corrected endpoint; their raw archives are outside the PR under `/private/tmp/alan-phase2a-narrow-v2/`. Only the corrected raw stores occupy `process-normal/` and `process-race/`.

Final compressed command SHA256 values: normal `f8de606ed59d39a0880498541edd864d075f05f6e30890f94ddf00d7ba80dd66`; race `4ad1e4ec6ed3d023621172e7cfdd06a905e80848ef2906457f58b76d52d8e6c0`. The corrected reducer-free audit passes both final raw directories. Independent review status is reported in the phase-wide review record, separately from this implementer note.
