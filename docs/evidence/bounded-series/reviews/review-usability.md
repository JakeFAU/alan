# Independent public authoring ergonomics review — REJECT

Reviewer: `series_review_usability`. Scope is the Task 1 public bounded-series authoring/inspection surface, not weather integration or whole-milestone qualification. No production edits, commits, subagents, internal-package imports, SQL, network provider contacts or existing-suite reruns were used.

## Frozen sources

- Initial source: `/private/tmp/alan-bounded-series.m9VryKAu/task1-review-source`, tree/revision `4bd4e8e228d8b67ecc144ca0eb1946a95bc39317`, manifest SHA-256 `3235d6935bc3220526b94463964cbc750992f8e0befd5d6b442e56abedcc3448`.
- Corrected classification candidate: `/private/tmp/alan-bounded-series.m9VryKAu/task1-fixed-source`, tree `4e1bdb0781ea7dc0ad329ea3cfd8501397c3ef8d`, manifest SHA-256 `89d766439419741771bead16e59da8014652322c74c60e173288452094f0a6b7`.
- Independently hashed all 307 files in each supplied manifest: zero mismatches. Inspected `author/series.go`, `author/series_inspect.go`, `author/continuation.go`, the existing public author/template/runtime/schema APIs, `docs/bounded-series.md`, Task 1 implementation report, original diff and classification correction.
- Read the detailed user milestone specification from the attached Pasted text. This verdict is intentionally not approval of the entire milestone.

## Blocking finding U1 — reject impossible derived continuation identities before spending

**Priority P1.** `author/series.go:59-61` accepts any nonempty UTF-8 series ID and derives subjects from it at lines 120-129. The successor receives an ordinary `ContinuationPort` at line 111, whose hidden `previous_operation_id` limit is 4096 bytes (`author/continuation.go:10`). The committed operation identity is JSON-derived and can exceed this bound even for a modest raw ID containing escaped characters.

Discriminating public-only experiment, independently reproduced on both frozen sources:

1. Declare a two-position freezer-reading series with either `strings.Repeat("x", 4100)` or `strings.Repeat("\x00", 700)` as ID. Both are valid according to the public identifier contract.
2. `BoundedSeries`, `Provision`, `runtime.Create` and the first `owner.Run` all succeed.
3. Genuine typed inspection reports first observation `Completed=1`, `Spent=1`; the adapter recorded one executed read.
4. Obtain the real second position using `series.Run(2)` and its real `cut.ContinuationBasis`.
5. `owner.ApproveContinuation(..., "continue")` fails with `continuation approval refused (schema_invalid): identity conflict`.

A developer has already paid for an external read before discovering that ALAN's generated lifecycle cannot express the promised continuation. There is no API documentation telling them to reason about the encoded internal identity size; a raw ID-length check alone would miss the 700-NUL case. Construction must either reject such declarations before provisioning or derive an identity supported by the continuation schema. Check the encoded predecessor identity at every relevant position, including decimal index growth and operation sequence representation; do not reject count-one declarations merely for an unused continuation limit. Preserve ordinary long-but-representable identifiers.

Probe: `usability-external/long_id_test.go`. Logs: `usability-long-id-original.log`, `usability-fixed.log`, `usability-fixed-retained.log`. Corrected-source retained ordinary-writer stores:

- `usability-external/stores/TestLongLegalSeriesID_long_ASCII-2581638154/freezer.db`
- `usability-external/stores/TestLongLegalSeriesID_escaped_controls-2290099227/freezer.db`

No database mutation was used to manufacture this failure. These stores include the executed first operation and real refused second approval.

## Public-only positive controls

Created unrelated module `example.net/independent-series-author` under `usability-external`, importing only standard library and public `author`, `effects`, `runtime`, `schema` packages. It defines a warehouse freezer sensor read with sensor input, integer milli-Celsius output and provider-style recorded timestamp. A deterministic in-process adapter implements the actual public Prepare/Execute protocol; all effects/results, authorizations, dispatch admissions, settlements, selections and debits are genuine runtime writes.

- **Eight useful observations:** one declaration with explicit `First: author.ApproveAtProvision`; seven ordinary successive approval calls; eight adapter contacts, eight distinct original operation IDs, eight selected and completed observations, eight independently charged opportunities, no ninth run. Selected typed payloads were read directly from `SeriesPosition.Observation.Payload` and summed to `-143964`; eight distinct timestamps matched their actual recorded values. No application-owned subjects, resource roots, grants, SQL or joins were constructed. Bound exhaustion was true; another Run caused no contact.
- **Withheld next approval:** after each paid position, next state was awaiting approval, later positions remained ineligible, and contacts equaled the explicitly approved count.
- **Transport failure at position 3:** the agent branches on the runtime `valid` field and terminates locally failed. Inspection shows two completed observations, one failed, one externally unresolved, three selected, three spent; usage is nil, failed payload is invalid, and later positions have no contact. Two close/open/run cycles preserved those facts and original third identity. Explicit fourth approval produced one distinct fourth reading while third remained unresolved: completed3, failed1, unresolved1, spent4.
- **Nil reasons:** intentionally permits an arbitrary application reason within the existing 64-byte schema. A 65-byte reason rejects before the next read. This is documented in `SeriesOptions` and was not treated as omitted authorization.
- **Uncommon normal identifiers:** Unicode, slash, newline and quotes round-trip through declaration, execution, continuations, inspection and Replay. The failing encoded-size boundary is therefore distinguished from blanket unusual-ID rejection.
- **Ownership and invalid inputs:** mutations of original input map, original definition and returned provision do not change the series. Zero/negative bounds, omitted first approval, malformed UTF-8 ID, zero Series and nil snapshot reject deterministically. Existing nested schema/list isolation tests were inspected, not rerun.
- **Read-only inspection and Replay:** before/after public inspection objects compare equal; repeated series inspection yields the same view and no adapter contact. This is an observational read-only check, not a physical byte-for-byte SQLite assertion.

Positive control retained stores from the corrected candidate:

- `usability-external/stores/TestEightRealReadingsPublicOnly-846417830/freezer.db`
- `usability-external/stores/TestTransportFailureRemainsExternallyUnresolved-928383499/freezer.db`
- `usability-external/stores/TestNilReasonsAndOwnedInputs-3362589082/freezer.db`

These cover genuine close/open behavior, not process SIGKILL. Process cuts and live/weather evidence remain outside this review.

## Ergonomics and specification assessment

For a standalone deployed finite sequence, this is a substantial reduction in developer work: the author provides one Definition and one SeriesOptions value, obtains one Provision and creates one ordinary Store. Approval remains the existing public snapshot → Run → ContinuationBasis → ApproveContinuation flow. Selected results are available directly, and aggregate counts never need to turn a retained unselected result into selected evidence. The author need not know root IDs, private receive wiring, grant construction or recovery implementation. Ordinary Run/Open/Inspect calls suffice.

The frozen contract is candid that this is **an aggregate inspection/deployment handle, not an invocable parent computation**. The type cannot be used as `author.Template` or an `author.Application` component; `Inspect` also deliberately requires the exact entire deployment, so a caller cannot combine it with unrelated static work and retain this series view. The existing template API requires a public entry boundary, while BoundedSeries rejects ports. These are real bounded-scope limitations, not a demonstrated runtime defect. The chosen standalone contract can satisfy the requested useful eight-observation application, with an application aggregate derived from actual selected readings. It does not establish general first-class parent/child composability, a selected aggregate root result, arbitrary continuation predicates or user-configurable per-position resource profiles. Delivery must state this scope rather than claim that an arbitrary ALAN caller can invoke the entire series as one retained parent operation.

The explicit policy is predecessor opportunity spent plus new operator approval, including after unknown results; successful/freshness continuation is application policy. That matches the frozen contract and the demonstrated distinct-read recovery behavior, but final guide must show that policy and avoid calling spent-predecessor eligibility predecessor success. Fixed one-effect/one-in-flight roots make Count the explicit aggregate operation budget; runtime storage limits are still supplied separately. No separate workflow engine was observed.

`Pending` and `Unresolved` are independent: a terminal local transport failure is failed and externally unresolved, with no claim that failed means the provider did nothing. The field comments explain this correctly. BoundExhausted is explicitly all opportunities spent, while position N+1 is always invalid; it is not a success flag. No additional blocker is raised on those names.

## Exact commands and outcomes

All Go commands used:

```sh
CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache go test -mod=mod -v ./...
CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache go test -mod=mod -v -run TestLongLegalSeriesID ./...
```

Working directory: `/private/tmp/alan-bounded-series.m9VryKAu/usability-external`. `go.mod` uses local replacement to the exact frozen export; initially task1-review-source, then task1-fixed-source. Go module and sum are retained.

1. Original4bd positive suite before boundary probe: PASS, `usability-external.log`.
2. Original4bd ID probe: FAIL in both named cases for U1, `usability-long-id-original.log`.
3. Corrected4e1 complete external suite: positive tests PASS, both U1 cases FAIL, `usability-fixed.log`.
4. Corrected4e1 rerun with persistent stores: same outcomes, `usability-fixed-retained.log`.

Independent fixture SHA-256 after persistent-store instrumentation:

- `series_test.go`: `75fd6a96fdf5bf2d77bfe4946b90fbf107bec43b68782f70c607cfd3f590eb15`
- `long_id_test.go`: `d7d95451be32d29a6a569c6181f402ec5c47620c932d7af9e64f66845a1d6c90`

## Verdict

**REJECT** Task 1 candidate `4e1bdb0781ea7dc0ad329ea3cfd8501397c3ef8d` until U1 is corrected and re-reviewed. The ordinary developer flow otherwise passed this independent external-module assessment within the explicit standalone-deployment scope. No final whole-source or whole-milestone approval is implied.

# Scoped U1 correction re-review — ADDRESSED / APPROVE

This addendum preserves the original REJECT and failing evidence above. It applies only to the U1 correction and its affected public contract, not final weather/docs integration or final whole-source qualification.

Reviewed immutable source `/private/tmp/alan-bounded-series.m9VryKAu/task1-fix2-source`, tree `0c6f05ba75765e81a42bbb7ab700b93ebcef336e`, manifest SHA-256 `0192634424818e2d35ee5f647e4bfbb362521344e756cd247548e41472270134`. Independently verified all 307 manifest files with zero mismatches. Reviewed `task-1-fix2.patch`, Round 2 implementation report and added regression tests. Owning reducer source confirms exactly `json.Marshal([]any{subject, generation, sequence, 0})`; compiler source confirms Repeat expansion, forward jumps and explicit ForEach loops.

**U1 ADDRESSED.** Constructor validates the encoded predecessor identity before producing a deployment, using the existing ContinuationPort field bound. It checks first predecessor and the largest later predecessor, accounts for the prepended receive's sequence width, and excludes the final non-predecessor position. It uses Go's actual JSON escaping rather than raw string length. Count-one series bypass this unused restriction. For a collection prefix the conservative full positive int64 sequence width is safe and documented; static prefixes use request PC+1 as a safe upper bound. No authority, runtime, resource, schema or identity-algorithm change was introduced.

Independent external-module correction probes are retained under `usability-fix2` with local replace pointing to exactly this source. The original `usability-external` fixtures and prior output/stores remain unchanged. Ran only original counterexamples and scoped new boundary probes, with the same offline/cache environment as the initial review:

```sh
CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache go test -mod=mod -v -run 'TestLongLegalSeriesID|TestU1' ./...
```

Working directory: `/private/tmp/alan-bounded-series.m9VryKAu/usability-fix2`. Result **PASS**, exit 0; exact output `usability-fix2.log`.

- Both unchanged original counterexamples now reject at declaration: encoded sizes 4112 and 4212, existing limit4096.
- Independent strict checks require wrapped `runtime.ErrLimit` for original ASCII/NUL IDs, exact one-byte oversize, HTML-sensitive and U+2028 escaping.
- All six identical IDs with Count1 were actually provisioned, executed and inspected successfully: one contact, one completed, one spent. Genuine retained stores are named in the log.
- Exactly4096-byte predecessor identities in ASCII, NUL, HTML-sensitive and U+2028 cases all execute first read → ordinary accepted approval → second read, with exactly two contacts, completed2, spent2. Genuine retained stores are named in the log.
- Count10 allows an unused final index10 digit, while Count11 rejects the predecessor index10 overflow.
- A prefix producing request sequence9 is allowed for Count2; Count3 rejects the added protected receive's sequence10 overflow. Removing one ID byte permits actual three-position execution through ordinary approvals, with the second predecessor exactly4096 bytes and contacts3/completed3/spent3.

**Spec/quality verdict: APPROVE for Task 1's scoped public authoring surface after U1 correction on this exact tree**, within the standalone finite-deployment/aggregate-view limitations already stated above. No new substantive finding in the correction diff. Original source remains rejected; this verdict does not retroactively approve it, and does not constitute final whole-source/milestone approval. Await final weather/docs frozen source for the planned final public-consumer check.

# Integrated source/public consumer review — scoped documentation REJECT

Reviewed immutable `/private/tmp/alan-bounded-series.m9VryKAu/producer-source`, producer commit `8a5f2ef564d5d750afa506675431d414bf92fb65`, tree `8caba38e10658672dc7f597751d322332f7d94bc`, manifest SHA-256 `ca95102a18437722cf61cce7dcf03ac7a98c2283266e193816ac4351c08ac048`. Independently verified all311 manifest files with zero mismatches. `author/series.go` and `author/series_inspect.go` are byte-identical to approved fix2. Inspected public guide, main README, weather README, ordinary definition/provision path, original/legacy loaders, report integration, approval policy and relevant tests. No production changes, subagents, commits, broad suite reruns or provider contacts.

**Public consumer/implementation assessment: APPROVE within the declared standalone scope. Documentation assessment: REJECT until U2/U3 below are corrected and reviewed.** Original U1 remains ADDRESSED. No additional production-code blocker found.

## U2 — attribute operator authentication to the application

**P2 documentation finding.** `docs/bounded-series.md:36` currently says verbatim:

> The owner authenticates the operator and the application decides whether to spend.

Here `owner` is the `*runtime.Store` returned in the example. Its approval API receives a typed basis and reason, not authenticated operator identity. The public contract in `docs/continuations.md` correctly states that only the local owner API holder submits approval and **the application authenticates the operator**. The new sentence can mislead an ordinary author into believing Store supplies an identity-authentication layer.

Required correction: state that the application authenticates the operator/controls access to the local owner API and decides whether to spend; ALAN validates the protected predecessor evidence and scoped approval. No new authentication mechanism is requested.

This is supported by the public API itself and the external consumer, which completes ordinary approvals using owner+basis+reason with no operator identity credential. The retained reason is an application assertion, not an authentication proof.

## U3 — refresh the snapshot before the post-execution inspection example

**P2 documentation finding.** The approval example captures `cut` before `ApproveContinuation` and `owner.Run`. The following “Inspect the whole declaration” snippet calls `series.Inspect(cut)` without obtaining another snapshot. Copied sequentially, it therefore displays the predecessor cut rather than the newly completed second observation.

Discriminating independent experiment on this exact source: `TestGuideSnapshotRequiresRefresh` in `usability-integrated/retained_weather_test.go`. Execute first reading, retain `cut`, approve and execute second reading, inspect that same `cut`: `Completed=1`. Take a fresh `runtime.InspectExecution` and inspect it: `Completed=2`. Exact result is in `usability-integrated.log`; genuine writer store is `usability-external/stores/TestGuideSnapshotRequiresRefresh-2633485999/freezer.db`.

Required correction: obtain a fresh `runtime.InspectExecution(ctx, path, limits)` immediately before `series.Inspect(cut)` in the post-run inspection example, with error handling. The detached snapshot contract is correct; the guide should demonstrate it accurately.

## Independent integrated consumer evidence

Created unrelated module `usability-integrated` using local replacement to exactly producer-source. Only public `author`, `runtime`, `effects`, `schema`, `adapters/openmeteo` and standard library imports. Preserved all prior fixtures/sources/logs separately.

Exact command, cwd `/private/tmp/alan-bounded-series.m9VryKAu/usability-integrated`:

```sh
CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local GOCACHE=/private/tmp/alan-runtime-health.18e1DyxD/cache GOMODCACHE=/private/tmp/alan-runtime-health.18e1DyxD/modcache go test -mod=mod -v ./...
```

**PASS**, exit0, external module only, `usability-integrated.log`. This does not duplicate repository-wide verification.

- Repeated exact public consumer eight-reading, transport-failure/reopen, mutation, nil-reason, zero/invalid and U1 boundary controls. All PASS. Eight real freezer writes retain eight distinct operation IDs, eight completed/selected/spent, direct selected aggregate `-143964`; third transport failure retains uncertainty through reopen and explicit fourth approval.
- Read private copies of five producer fixture database+WAL pairs, using public `runtime.InspectExecution`, reconstructed public BoundedSeries from exact ordinary weather definition+initial inputs, `Series.Inspect`, Open-Meteo native response verification and `runtime.Replay`. No raw SQL, internal imports, adapters or owner execution used for these reads. Before/after typed snapshots match.
- `ordinary-eight`: selected8, completed8, spent8, unresolved0, eight distinct selected provider timestamps, Replay127 reductions.
- `third_accepted_response_lost-stopped`: selected2, completed2, spent3, unresolved1, Replay33 reductions.
- `third_accepted_response_lost-final`: selected3, completed3, spent4, unresolved1, Replay49 reductions; prior uncertainty remains distinct from successful fourth result.
- `third_result_retained-stopped`: selected2, completed2, spent3, unresolved0, Replay33 reductions; retained third result is not promoted to selected.
- `after_fourth_approval-stopped`: selected3, completed3, spent3, unresolved0, Replay47 reductions; fourth approval has not spent a fourth opportunity.

## Final-user-question assessment

**Yes, for this documented standalone finite observation sequence:** an ordinary author can supply one request/await/result-check definition, a bound, inputs and explicit first authorization; obtain one provision and use normal Create/Run/Open APIs. Seven later spend decisions use the existing public ContinuationBasis/ApproveContinuation APIs. They need not construct subjects, resource scopes, grants, private receives, checkpoints, SQL or a recovery scheduler. Typed actual selected payloads support useful aggregate values without application-owned storage joins. The existing external freezer fixture is an executable non-weather demonstration; the integrated weather path uses the same public declaration.

Necessary application responsibilities remain visible: install a truthful bounded adapter, retain original definition/configuration to rebuild the authoring handle, authenticate/control operator access and choose whether another spend is useful. These are deployment/application policy obligations, not knowledge of ALAN's underlying durable construction. If original authoring source is unavailable, runtime general inspection still works.

The guide candidly says this is a standalone deployment aggregate, not an invocable parent outcome, and that there is no global physical single-flight guarantee across independently approved roots. It defines selection, success, expenditure and external uncertainty separately; the weather README preserves provider attribution, per-reading timestamps, unknown cost, historical-program compatibility and the CLI's0–8 limit distinct from the public API. The example does not support arbitrary parent composition, per-result author-defined runtime policy execution or changing action/input per index, and delivery should continue to state that boundary. No substantive residual abstraction leak was found beyond those intentional limits and the two documentation corrections above.

**Source-specific verdict: REJECT integrated guide for tree8caba38 pending U2/U3; implementation/public consumer remains approved within scope.** Await the small documentation correction and final evidence-only tree confirmation. Do not relabel producer measurements as generated by a later source.

# Scoped U2/U3 guide correction — ADDRESSED / APPROVE

Reviewed exact corrected `docs/bounded-series.md`, SHA-256 `a43e9f6eccb45488f531966684ff7a589334488418c1e8fc713dc8b5736fb0e7`, against the immutable producer `8a5f2ef564d5d750afa506675431d414bf92fb65` version. Read `doc-fix.patch`, `doc-fix-report.md`, the complete two-hunk diff and surrounding examples. The original documentation REJECT above remains preserved.

- **U2 ADDRESSED.** The guide now states: “The application authenticates the operator and decides whether to spend. The owning Store enforces the scoped retained authorization.” This correctly separates operator authentication/application policy from ALAN's retained authorization checks and matches the existing continuation trust contract.
- **U3 ADDRESSED.** Immediately before `series.Inspect(cut)`, the guide now obtains a fresh `runtime.InspectExecution` snapshot and handles its error. It explicitly says detached snapshots do not change after subsequent commits. This fixes the exact staleCompleted1/freshCompleted2 behavior established by the integrated external-module experiment.

**Guide approval: APPROVE for the exact corrected file SHA above.** No new substantive issue in either correction. No Go semantics changed and no broad suite was rerun; prior public consumer evidence remains attributed to its original immutable producer source. Await final whole-source/evidence-only delivery tree confirmation before describing this as a final delivered-source approval.
