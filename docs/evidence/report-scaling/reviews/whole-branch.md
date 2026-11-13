# Independent final whole-branch review

Reviewed 2026-11-13 against `docs/evidence/report-scaling/contract.md` and `plan.md`.

- Base: `5f5d382c8d1e9c7369e7624e3a962746c9bf2229`.
- Delivery reviewed: `fd3a3fde1f1ddc4c397beaa84abedc63b03e1833`, tree `a63527dc244baa46fddc81ba06a7afcb8be95345`.
- Actual frozen qualification source: `7679755c7298f229fa468b14da54bcfe1420c4f8`, tree `7aeb84be8acf414800a649152746aacd3e03b5e6`.

## Strengths

1. **Small production scope with coherent ownership.** `internal/executor/runner.go:529` acquires the existing Store mutex before creating the reader at line 542. The reader is consumed entirely before unlock. `composition_inspect.go:28` creates a fresh reader for each independent traversal; inspection and Replay use their existing read transactions (`inspect.go:38`, `replay.go:33`). The reader never becomes Store state or a mutation/authority input. The complete production diff comprises `report_read.go` and the two small call-site changes. No schema, module, public API, dispatch, admission, settlement, resource-accounting, original operation identity, or receipt-validation change appears.

2. **Raw data collection preserves logical validation order.** `report_read.go:67` and `:105` accumulate descriptors/projections without decoding them, reject duplicate keys and unsupported storage classes, close Rows, and discard the affected batch on query/scan/iteration failure. `:146` and `:195` decode only at the original logical use sites and check cancellation on cached reads. Missing subjects still produce `sql.ErrNoRows`; RunID keys retain generation. `runner.go:550` retains the original ordered run-ID traversal and partial-report assignments. `composition_inspect.go:50` preserves sorted roots, first-root charges/membership work, nil root results on failure, and the original error precedence. Source comparison with `store.go:744` confirms equivalent RunState fields and decoder errors for the accepted batch domain. Fallback preserves SQLite Scan conversions and errors outside it.

3. **Root preparation preserves published and pending boundaries.** `report_read.go:171` reproduces initial metadata, original publication decoding and the no-freeze/no-closure `members` result. A present unpublished freeze or closure delegates to unchanged `prepareRoot` and `prepareCompositionScope` (`composition_root.go:23`, `invocation_outcome.go:324`). The function-free presence sums reference all originally reached selected columns. Published roots still bypass unused corrupt/denied preparation evidence. The ordinary leaf path continues closure canonical/identity/order validation (`invocation_closure.go:44`) and authenticated intent/result checks (`composition_reservation.go:124`); batching does not replace them with a global intent projection.

4. **Cleanup and unknown-outcome semantics remain intact.** `runner.go:94` still cancels, joins all workers, drains each outstanding completion, and accounts for it before reporting. Reporting remains under a cleanup context independent of caller cancellation. Unresolved/unknown classification still distinguishes a retained receipt from a later local settlement and uses the unchanged authoritative loaders (`runner.go:587`). The diff does not alter those paths. Tests cover receipt arrival, settlement, retained conflicting receipts and earlier correlation errors winning over later malformed root/projection bytes (`report_read_test.go:316`).

5. **The oracle and tests discriminate meaningful failures.** `oracle.py:11` freezes the original traversal and preparation functions from the exact baseline, leaving only unchanged authoritative leaf readers shared. Regeneration with `python3 docs/evidence/report-scaling/oracle.py --check` exited 0, and the installed fixture equals the tracked frozen text. The 13 integration tests cover structural query savings, published/unused evidence, missing subjects/runs, SQL conversion fallback, late partial results, duplicate fallback, lifetime isolation, generations and cancellation. Existing fault-hook assertions remain; the additions in `composition_inspect_test.go:595` preserve traversal/no-write checks. Actual final integration and integration-race logs contain all 13 PASS events.

6. **Rejected designs remain rejected and inspectable.** Actual `task2-checks/red-presence-only.log` shows false success versus denied original columns in four unpublished cases. Actual `red-marker-function.log` shows a `members` success versus the original malformed-freeze `unexpected EOF`; `green-presence.log` records the corrected controls passing. Those results explain the current presence expressions rather than being waved away. The compact witness (`runner_compact_assessment_test.go:42`) independently checks legitimate r01 completion before asserting the old r00 admission diagnostic. The actual overlay differs by exactly the one deferred `finishRun` invocation; its positive passes and its negative exits 1 with the intended nil-error diagnostic loss. `compact-result-decision.md:14` correctly leaves validation-preserving streaming output possible and unmeasured; it claims neither unauthorized execution nor impossibility of compact APIs.

7. **Measurements and qualification claims are bounded and reproducible.** `campaign.py:31` specifies 14 excluded warmups and six adjacent balanced AB/BA pairs at each of seven bounds. Its input/row validation, process identity checks and independent post-exit high-water recording separate source, timing and diagnostic evidence. `campaign_qualification.py:177` uses same-seed complete reports, and `:245` seals stopped stores before independent reader copies. `results.md:88` reports fewer statements alongside the much smaller VM reduction and increased full scans. `results.md:106` retains quadratic output/work, and `:130` states that transient memory scales with retained descriptor/projection bytes. `qualification.md:27` correctly limits native binding to selected executable bytes and SDK identity/settings.

## Independent evidence checks in this review

Read the prepared whole-branch package and changed production/tests, apparatus and prose, recovering portions omitted by output truncation. Parsed all 18 compact JSON/JSONL files as structured data rather than dumping their payloads. Consulted Task 2–5 reviews, the actual Task 5 report, progress ledger and delivery plan; their prior approvals did not substitute for this source inspection. Inspected unchanged callers/loaders and custody helpers only for the named ownership, identity, validation and stopped-store risks.

- Recomputed the seven Execute median pairs from the 98 compact rows, verified the twelve measured rows and balanced order at every bound, and checked actual stdout digests and Execute values for all twelve B256 measured reports. B256 is 7.0309272295→4.613166646 seconds with six positive differences. B1 has four positive and two negative differences. B256 post-exit RSS medians are 85,721,088→87,162,880 bytes.
- Independently read all four actual same-seed public-report pairs. After removing only source labels and Store paths, the complete JSON objects are equal, including arrays/null, original identities and all 18/25 public cuts. The retained comparator additionally checks durable rows; its exhaustive audit is recorded in Task 5 review.
- Reassembled actual fragmented weather output for both builds and both series/repeated-recovery routes and matched all compact controlled facts. The reservation/gate cases retain four local charges and three contacts with zero/one unresolved result. The accepted-response-lost history retains four contacts/charges, three completed/selected and one unresolved original. Each repeated history retains one original contact and charge through three kills.
- Independently hashed six selected sealed main/WAL/SHM triplets: both repeated histories' first stops, both accepted-response-lost stops, baseline post-commit nested stop and candidate pre-commit nested stop. All matched their complete seal/source-before/source-after records. Read all 26 weather kill witnesses and four nested ACK/reap records; these retain actual signal-9/−9 and PID identities. No database was opened. The earlier independent Task 5 review provides the exhaustive 30-triplet/52-reader/26-comparison audit, which this review did not redundantly rerun.
- Checked actual final ordinary/integration/integration-race, public parent/promotion and supported historical-owner log hashes against compact records. The complete integration variants contain the thirteen new report tests and their one explicitly named opt-in historical-falsifier skip. The shipped 393 command outcomes contain 392 exits 0 and the expected null-mutant exit 1. The Task 5 review independently checked all command/file hashes, all thirteen Staticcheck scopes, external format controls and outside proof; no broad suites or builds were rerun here.
- Verified current HEAD/tree and clean tracked/untracked status against the review dispatch, the production reader's exact frozen bytes, and the recorded delivery bridge. The prior exhaustive bridge check binds all 756 exported files and 754 unchanged live files other than the two README updates, including tests, fixtures, modules and consumed apparatus. Its evidence-only delivery label is distinct from the tested source. This review does not relabel tests as having run on a later commit.

The only review write is this file. No checkout/index/HEAD mutation, database open, application run, benchmark, broad suite, build, or subagent was used. Prior ignored P4 archives were not accessed.

## Issues

- Critical: none found.
- Important: none found.
- Minor: no actionable finding identified.

No retained rejection is being waived. The rejected marker designs were corrected and their discriminating tests remain. The compact bypass remains rejected; no incomplete public API is delivered. The scoped freeze/cleanup/identity, error, compatibility and evidence contracts are satisfied by the reviewed implementation and bounded qualification.

## Declined to judge

Each line is an explicit boundary for the controller's ruling, not an unreported exception:

- Universal latency or memory improvement for arbitrary retained histories, especially large projections: only the stated seven-bound small-projection series was measured; call-local memory scales with retained bytes and peak RSS increased here.
- Identical wall-clock cancellation timing or identical SQL-authorizer callback counts: the optimization intentionally changes SQL timing/counts. Reviewed cancellation causes and reached column-denial behavior remain covered; arbitrary stateful private callback behavior is not a public Store contract.
- Coherence in the presence of unsupported concurrent external SQL writers that bypass owner exclusion: the reader relies on the existing supported owner-lock/read-transaction contract; this branch neither introduces nor qualifies hostile-writer protection.
- Power-loss, storage-device failure, remote exactly-once effects or unrestricted distributed recovery: retained evidence exercises acknowledged local process SIGKILL and same-filesystem recovery, not those stronger failure models.
- Benefit or feasibility of a future validation-preserving compact/streaming API or reusable validation certificate: no such API is implemented or measured. The witness supports the narrower loss of existing diagnostics when bypassing finishRun.
- Byte stability of every Apple SDK header/library and performance on other toolchains/hosts: only the stated selected native tools/settings are bound; broader environment qualification is absent and explicitly disclaimed.
- Public availability of the bulky raw captures/full index: they remain explicitly local, with compact records, hashes and reproduction tools committed. Publication was outside this delivery scope.

## Ledger rulings and recommendations

The five existing `Ruling:` entries were considered: proceed under the authorized two-step slice; retain this plan's evidence workspace; permit Task 3's concrete deferral; defer acknowledgment/settlement batching; and split apparatus review from frozen qualification. They are consistent with the supplied contract and do not excuse any failing implementation. Their costs are respectively scope/authorization mismatch if the controller misread the user, retained disk usage, postponed compact-result benefit, remaining per-intent SQL work, and extra review latency. The controller should preserve them and explicitly resolve the declined-to-judge lines in the delivery record. No additional implementation work is recommended for this scoped branch.

Keep the tested-source/delivery bridge and all rejected/control evidence when appending the final review records. Complete the planned draft-PR delivery after that evidence-only check; this approval is independent scrutiny, not merge or release authorization.

## Assessment

**Specification: APPROVE. Code quality: APPROVE. Final scoped branch: APPROVE.**

The implementation reduces repeated report reads without moving authority or validation into reusable state, and the decisive retained histories support the preserved report/error/recovery behavior. The measured benefit and remaining quadratic, memory, environment and custody limits are stated accurately. No corrective fix is required before the planned review-record/source-bridge commit and draft PR.
