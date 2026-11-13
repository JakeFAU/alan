# Report scaling implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox syntax for tracking.

**Goal:** Reduce redundant full-report work and assess optional smaller execution results.

**Architecture:** Call-local typed batch reads under existing owner/transaction boundaries; original validation remains authoritative. The compact-result decision follows the full-report measurement and a validation audit.

**Tech Stack:** Existing Go and go-sqlite3; standard-library qualification tooling.

**Spec:** `docs/evidence/report-scaling/contract.md`.

## Global Constraints

- Preserve every existing report field, ordering, nil/empty distinction and retained-data validation/error behavior.
- No scheduling, authority, dispatch, resource, original identity, recovery, storage-format or dependency change.
- Reuse is call-local under the existing owner lock or read transaction; no durable or process-lifetime authority cache.
- A compact result cannot suppress necessary integrity checks; defer it with a concrete counterexample if a new validation protocol would be needed.
- Preserve prior evidence, all rejected findings and actual command exits. Draft PR only; no merge or release.

## Review Focus

- Used versus unused corrupt rows: batching must preserve when validation applies.
- Unsuccessful or missing local results must not imply remote nonapplication.
- Failed/canceled reporting must still join workers and retain received results.
- Historical formats may lack optional tables and must retain their accepted path.
- Call-local summaries must not survive writes, owner changes or transaction cuts.

### Task 1: Freeze baseline and reusable measurement

**Files:** Add compact apparatus and baseline records under `docs/evidence/report-scaling/`; no production code changes.

**Interfaces:** Public BoundedSeries, Store.Run, Store.ContinuationBasis, InspectExecution and Replay; preserve existing qualification drivers where reusable.

- [ ] Record source/tree, clean status, toolchain, dependencies and sanitized environment.
- [ ] Provide a small reproducible measurement route using existing public driver or a focused new driver. Keep sources and counters explicitly bound, profiles separate, and full report output verified.
- [ ] Run baseline ordinary tests and excluded baseline samples at all seven bounds; capture B256 report attribution using existing or focused counters/profiles.
- [ ] Record actual report validation obligations and safe batch boundaries. Establish red structural tests or frozen differential oracle before production edits.
- [ ] Review apparatus/spec compliance independently and commit baseline/apparatus evidence.

### Task 2: Batch full reports without changing their contract

**Files:** `internal/executor/runner.go`, `composition_inspect.go`, a focused report-reader file if useful, and corresponding integration tests. Existing decoder helpers may be shared only without changing their checks.

**Interfaces:** Existing finishRun/readRootCompletions inputs and outputs unchanged; no public additions.

- [ ] Add an independent baseline finishRun/report oracle and targeted regression cases for known, unknown, pending, terminal, canceled and corrupted evidence. Reuse the existing root report oracle.
- [ ] Implement the smallest batching change supported by the audit, under the existing coherent read cut. Preserve original point paths for unsupported/error-domain cases when needed.
- [ ] Demonstrate fewer SQL statements for increasing roots/operations while comparing all report fields and errors. Test malformed used/unused evidence and receipt/correlation attacks.
- [ ] Run focused ordinary/integration/race checks and excluded measurements. Preserve any failed candidates and fix with reviewed evidence.
- [ ] Commit and obtain independent spec/quality review before proceeding.

### Task 3: Decide compact execution-result contract

**Files:** Decision record under `docs/evidence/report-scaling/`; public runtime/runner changes only if independently justified by this contract.

**Interfaces:** Existing Run remains complete. Any opt-in result must have explicit completeness semantics and retain worker cleanup, accounting and mandatory validation.

- [ ] Use the audit and Task 2 data to distinguish output construction from required global integrity checks.
- [ ] Retain an executable counterexample if omitting report materialization changes retained-history validity. If a new validation protocol/certificate is required, defer this API and document why.
- [ ] Otherwise freeze a narrowly specified opt-in interface and tests before implementation, compare execution facts under both modes, and review independently. Do not claim full-report and smaller-output measurements are equivalent contracts.
- [ ] Commit the decision, any qualifying code and tests; obtain explicit review disposition.

### Task 4: Review reproducible qualification apparatus

**Files:** A small P5 campaign entry point and controls under `docs/evidence/report-scaling/`, reusing the tracked P4 measurement/custody/qualification helpers without changing them.

**Interfaces:** Exact baseline/candidate source exports, TaskRecorder, existing full-report driver, historical readers, same-seed comparator and acknowledged process barriers.

- [ ] Record the Task 3 decision and split this qualification work from campaign execution so the apparatus is reviewed before confirmatory measurements.
- [ ] Implement source/tool freeze, build, excluded smoke, frozen gate/recovery/differential routes, paired timing and strict analysis. Bind actual source/archive/tool/binary identities and preserve every failed attempt.
- [ ] Freeze a single full-report comparison with seven bounds, 14 excluded warmups and 84 measured rows in six adjacent balanced AB/BA pairs per bound. Counter/profile captures stay separate from primary timing.
- [ ] Supply real old-reader fixtures, exact required test events, full main/WAL/SHM custody, and independently rebuilt public consumers using the existing reviewed helpers.
- [ ] Add focused apparatus controls for altered/missing rows, wrong source/binary, incomplete reports and unexpected exits; run excluded smoke, commit and obtain independent apparatus review.

### Task 5: Frozen qualification and draft delivery

**Files:** Focused documentation/README and compact raw evidence, reproduction scripts and review reports under `docs/evidence/report-scaling/`.

**Interfaces:** Reviewed source commits and unchanged ordinary public consumers/recovery tests.

- [ ] Freeze clean source; export independent baseline/candidate trees and record hashes before/after.
- [ ] Run six balanced pairs per bound plus excluded warmups, isolated from test/build workloads. Compare full reports, contacts, authority, charges, original IDs and outcomes on matched histories.
- [ ] Run acknowledged real-process SIGKILL/reopen and read-only/compatibility suites affected by reporting, plus the independent eight-operation consumer.
- [ ] Run complete ordinary, integration and integration-race suites, build, vet, module verification, formatting/whitespace and whole-repository applicable Staticcheck. Record skips and failures honestly.
- [ ] Obtain independent semantic/durability, methodology and whole-branch reviews. Resolve substantive findings and requalify changed inputs.
- [ ] Write bounded results, remaining quadratic costs and compact-result decision. Verify delivery source bridge; push and create/attach a draft PR with exact base/head identities. Do not merge or release.
