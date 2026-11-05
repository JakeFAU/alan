# P2 implementation plan

> Use subagent-driven development with independent review gates. The user has
> authorized diagnosis through draft delivery; routine choices need no new approval.

**Goal:** Explain marginal retained-execution cost and qualify at most one optimization.
**Architecture:** One frozen public workload, separate diagnostic source copies,
one production change only after attribution and invariant review.
**Tech stack:** Existing Go/SQLite/Python tools; no new runtime dependency.
**Spec:** [Frozen experiment contract](contract.md), implementing the supplied P2 assignment.

## Global constraints

- Preserve original durable facts, authority, paid opportunities and uncertainty.
- No public API, SQLite format, ownership, scheduler or general caching changes.
- Earlier evidence and source exports are immutable. Every new run has its own identity.
- Measure serially; separate instrumented/profiled diagnostics from headline timing.
- Stop after one justified production optimization or deliver a negative result.
- No merge or release; push a draft PR after complete qualification.

## Review focus

- Fixed declaration B versus retained prefix k: never conflate their marginal costs.
- Mutable authority/accounting and later receipts cannot inherit stale cached reports.
- Corrupt or causally impossible membership and generation references must still
  reject; preserve legitimate absent/inapplicable membership at its original boundary.
- Evidence comparators must not normalize IDs, endpoints, grants, debits or selection.
- Final compiler/tool hashes and published evidence must match tested frozen source.

## Tasks and gates


### Task 1

Verify baseline/ancestry/environment and freeze contract. Read-only reviewers
inspect source and protocol; commit the contract before diagnostic implementation.

### Task 2

Create `docs/evidence/performance-p2/driver/` from the corrected P1 public
workload, adding controlled endpoint/journal, adjacent prefixes and GC accounting.
Add provenance/duplicate/missing-prefix mutations; preserve old tools unchanged.
Create independent counter/timing copies under `instrument/`; test SQL family,
scope and prefix arithmetic. Review all tools before collecting authoritative data.

### Task 3

Reproduce baseline bridge and collect size/counter/profile matrices serially.
Retain raw artifacts outside source. Write attribution, top SQL families, T(B,k)
and semantic cost-model reports. Identify missing evidence explicitly.

### Task 4

Freeze selection/invariant with falsifiable prediction and independent
challenge. Only then write targeted semantic/regression tests and the smallest
production change. Preserve RED controls and review every substantive correction.

### Task 5

Compare exact same-seed histories, negative mutations, real crash cuts,
historical readers and external consumer. Freeze implementation. Collect the
predeclared 98-process paired matrix and post-change attribution; stop optimizing.

### Task 6

Final independent reviews, compact documentation and exact frozen-source
ordinary/integration/race/build/vet/Staticcheck/tool qualification. Preserve failures.
Commit, push draft PR, seal/upload/hash-check evidence; answer all ten questions.

## Preflight conflict scan

| Tasks | Shared interface or rule | Ruling |
| --- | --- | --- |
| 1 → 2 | Workload/prefix definitions | Tool must implement T(B,k), not standalone T(k,k). |
| 2 → 3 | Counters/profiles/timing | Diagnostic transformations never enter normal binaries. |
| 3 → 4 | Cost versus optimization choice | No production mechanism selected until direct measurements. |
| 4 → 5 | Invariant and exact comparator | Compare every original retained field; no semantic normalization. |
| 5 → 6 | Frozen source and later documents | Final export gets separate verification; earlier runs retain identities. |
| Each task | Scope and tests | Contract governs; failures are evidence, not permission to weaken semantics. |

The production file and targeted regression names are intentionally selected only
after Task 3 evidence, as the assignment requires. That selection is a second
committed gate, not an invitation to implement several candidates.
