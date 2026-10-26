# Task 3 independent review

Reviewed frozen Task 3 source against BASE `47a3317` on 2026-10-25. The 21 source/test files match `evidence/task-3/frozen-source-sha256.json`; the independent verification is retained in `evidence/task-3/independent-review/source-verification.json`.

**Specification verdict: CHANGES REQUIRED.** The implementation has the intended single-runtime architecture and conservative recovery profile, but the credential-exclusion and invalid-provider-output contracts have concrete counterexamples. Inspect also misses a reverse adapter-ledger edge.

**Code-quality verdict: CHANGES REQUIRED.** Ownership, transaction separation, immutable binding, and focused tests are generally sound. The hand-written sanitizer has inconsistent coverage and fails open at its recursion limit; the typed envelope decoder accepts conflicting case aliases; adapter validation duplicates several forward checks without the complete reverse observation check. Resolve the findings below before approving this slice.

## Findings

### R1 — P1: reversible credentials survive returned metadata and bounded raw evidence

**Locations:** `adapters/openai/openai.go:127`, `adapters/openai/openai.go:167`; `adapters/openai/json.go:69`.

The HTTP adapter redacts `x-request-id` with direct replacement, and the final candidate pass only repeats direct replacement over its serialized JSON. A header containing `review-fake-\u0073ecret`, with runtime fake credential `review-fake-secret`, survives as `Candidate.RequestID`. The candidate remains `validated`; decoding the Unicode escape recovers the full credential.

Separately, `redactEvidence` returns `direct` when its nested-string depth exceeds four. A 401 error body containing the same escaped fake credential wrapped in eight JSON string layers survives in `Candidate.Response`. Eight ordinary JSON decodes recover the credential. This is within the configured 65536-byte response bound. These are returned fields that `recordAdapterResult` copies, hashes, and persists without an additional secret-aware sanitizer, so the defect reaches retained evidence/inspection when used by Run. No real credential was used or accessed.

**Evidence:** `independent-review/probe.go` and `probe-output.jsonl`, cases `escaped_header` and `nested_json_credential`: respectively `header_decodes_to_credential:true` and `nested_decodes_to_credential:true`, exactly one loopback contact each. Original OpenAI source is preserved under `independent-review/original-source/`.

**Required action:** use one bounded, conservative sanitization policy for every returned credential-bearing string and raw-detail path, including headers, keys, nested strings, and limit/failure paths. When a decoding/sanitization budget is exhausted, omit unsafe evidence instead of returning potentially reversible content. Add regressions that actually decode retained strings and verify the fake credential cannot be recovered. Do not merely check raw substring absence or raise the recursion cutoff.

### R2 — P2: case-insensitive JSON struct decoding turns canonical incomplete status into success

**Location:** `adapters/openai/openai.go:187-200` (and the case-sensitive duplicate-key check in `adapters/openai/json.go`).

A response containing `"status":"incomplete","Status":"completed"` plus one valid output text is returned as `validated`. `uniqueJSON` treats these as distinct keys, while `encoding/json` maps both to the same struct field and uses the later value. The canonical provider status therefore loses authority to a noncanonical alias. The same decoding pattern covers error, output, role, and content type. Adding JSON struct tags alone does not remove Go's case-insensitive matching.

**Evidence:** `independent-review/probe-output.jsonl`, case `case_conflicting_status`: retained raw response has canonical `status:incomplete`, but the candidate has `Disposition:validated`, payload `answer:ok`, and one HTTP contact. The control with canonical completed status also validates.

**Required action:** decode decision-bearing protocol keys with exact spelling, or explicitly reject aliases/conflicts before typed decoding. Retain this exact input and add focused coverage for status/error/output/content aliases. Canonical incomplete/refusal/error data must never become validated success because of extra fields.

### R3 — P2: Inspect accepts an orphan authenticated adapter observation

**Locations:** `internal/executor/adapters.go:473-559`; `internal/executor/inspect.go:71`.

`validateAdapterEdges` starts from adapter admissions and settlements but never checks the reverse link from every authenticated adapter observation. After a valid run, deleting its adapter admission and settlement leaves the consumed `runtime-adapter` observation and retained result. Inspect returns success and exposes that orphan observation as part of the snapshot. This contradicts the report's claim that Inspect validates the new admission/observation/settlement links.

This is an inspection consistency gap, not an execution bypass: the same database is correctly rejected by Open and Replay through `validateMessaging` with `unexplained authenticated observation`. A committed known-but-unsettled result with no observation must remain valid; the missing reverse check must distinguish that state from an unexplained observation.

**Evidence:** `independent-review/ledger-probe.go`, `ledger-probe-output.json`, pristine `ledger-original.db`, and modified `ledger-orphan.db`. Output: Inspect error `<nil>`, one observation, one adapter result, zero admissions/settlements; Open and Replay both reject. The probe uses a trusted callback solely to construct ledger state; it is not claimed as HTTP protocol qualification.

**Required action:** add a reducer-free reverse validation of every adapter observation and its exact admission/settlement/result binding to Inspect's adapter checks. Preserve known-but-unsettled support and test both valid pending evidence and orphan observations.

## Positive assessment and boundaries

- Preparation compares a copied request against the committed request, runs outside the Store lock, and precedes source authorization and adapter reservation. Workers launch only after those commits. The supplied Prepare contract expressly trusts deployment implementations to be pure, bounded, nonblocking, and free of Store access. The API is not a sandbox, and cancellation/join depends on trusted Execute respecting context. This is an acceptable stated limited trust profile; no additional runtime or callbacks inside reduction were found.
- Request identity, declared schemas, exact capability/static grant, namespace-derived key, original authorization, prepared digest, result correlation/digest, normalized result envelope, and settlement/admission identities are checked. Schema 3 retains key version 2; Open validates before epoch renewal. Result recording, settlement, and ordinary consumption remain distinct commits.
- Reserved allowed attempts without results remain Unknown and are not repeated. Known-but-unsettled results settle without registry/provider execution; denied reservations recover deterministically. Late results bind to the original attempt and do not reopen closed sources. A source authorization alone precedes the separate durable physical reservation; no worker can run in that gap.
- Fixtures and adapters use the same active count, completion channel, cancellation, join, cleanup accounting, and ordinary scheduling loop. Existing focused tests exercise capacity 1/2, independent ordinary progress, faster later completion, cancellation with a returned candidate, epoch renewal and closed-source late evidence. The retained negative capacity mutation fails the unchanged capacity test as claimed.
- Corpus bytes, versions, and bounds are retained at provisioning. Lookup uses only those bytes, deterministic token/line ordering, and exact byte spans. Source paths are provenance; recovery/replay do not reread them. Replay accepts no adapter registry.
- The HTTP adapter has a single nonstreaming request, fixed deployment endpoint, explicit model and finite limits, strict payload schema, full retained instructions/input, no tools/conversation state, redirects disabled, and no adapter retry loop. The fresh non-rewindable transport does not reuse a connection for an implicit transport retry. Known rejection/refusal/incomplete/schema-invalid paths are represented, subject to R2. Optional returned usage remains optional.
- Generic record input emits every caller-declared non-instructions scalar as native JSON. Question plus document/version/snippet/start/end reach the model through ordinary values; no stage-specific adapter behavior or authoring opcode was introduced.

## Evidence and scope

Read the Task 3 brief, report, frozen review package and owned source/tests, authoring/effects contract, retained provider notes, and requirements sections 5/6/7/8/9C-I/11. Followed relevant shared owner, authorization, replay, inspection, and SQLite edges outside the immediate new files. The retained provider notes supplied the protocol source context; no live provider call, credential discovery, or new protocol research was needed.

Accepted the supplied frozen default, integration and focused race runs as regression evidence, inspected their successful outputs, and inspected the negative control configuration/failing output. Did not rerun broad suites. Independently executed:

```text
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off go run .superpowers/sdd/00-plan/evidence/task-3/independent-review/probe.go
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off go run .superpowers/sdd/00-plan/evidence/task-3/independent-review/ledger-probe.go
```

Both probe processes exited 0 because they emit observations rather than test assertions; their output explicitly demonstrates the failures above. The ledger probe creates fresh evidence database filenames and should be replayed in a disposable copy with those filenames absent, preserving the original retained databases. Source, original inputs, and output were retained before any correction. No runtime/test source edits, commits, or subagents were made by this reviewer.

Task 4 application/CLI behavior and Task 5 real process SIGKILL qualification are outside this slice and are not claimed complete. Live gate: NOT RUN. No general production-readiness, business exactly-once, power-loss, or unlimited-liveness conclusion follows from this review.
