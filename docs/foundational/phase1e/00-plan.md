# Phase 1E minimal authoring and evidence assistant implementation plan

> Use subagent-driven development: one implementer per sequential slice, task review at each boundary, then independent whole-branch review. The user explicitly authorizes implementation after this concise plan; no additional design approval is needed.

**Goal:** A Go author defines a composed evidence assistant using named operations; model proposal, lookup, synthesis and publication remain visible durable transitions and replay offline.

**Architecture:** Compile a small labeled Go definition into the existing finite interpreter. Keep the authoritative Store and bounded Run owner. Add one generic adapter boundary for a thin OpenAI Responses adapter and a deterministic read-only corpus lookup; existing fixture transport remains restricted to loopback. Statically expand the helper behind a request/result contract.

**Tech Stack:** Go 1.27.3, existing go-sqlite3 v1.14.52, standard-library HTTP/crypto/JSON. No additional infrastructure or orchestration dependency.

**Spec:** [complete user request](requirements.md). [Contract](01-authoring-effects-contract.md). [Official provider research](provider-docs.md).

## Baseline and working state

PR #5 is confirmed merged. Fresh fetch gives base `d708493646aeae3c1e70a03cf2bb2d7db974c127`, identical tracked tree to published `338c89f`. Reuse the existing clean managed worktree on new branch `codex/phase1e-authoring-agent`; do not alter the original checkout or its user-extracted source files. Baseline Go/module source is `9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63` (58 files). Baseline reproduction uses an immutable Git snapshot. No credential/model is configured in the task environment; live is NOT RUN unless explicitly configured during the task.

## Global Constraints

- The authoring layer must compile to the existing execution model. It must not become a second runtime.
- One host and one active executor per store; SQLite WAL/FULL; bounded asynchronous dispatch; static subjects/grants; ordinary control and supervision.
- No worker dispatches before intent commit and source authorization. Result recording, settlement and ordinary consumption remain distinct.
- Model output is a proposal, not authority. Network endpoints/credentials come from deployment configuration, never model output.
- No callbacks inside deterministic reduction; persisted artifacts, templates, schemas, control and corpus bytes must suffice after restart.
- Preserve prior research/runtime inputs, negative controls and historical evidence under original identities. Current-source behavioral comparisons are separate.
- No automatic retry of an unknown provider outcome, no provider reconciliation or deliberate repeat policy in this increment. Known invalid/refused/incomplete results take explicit ordinary non-success branches.
- No dynamic creation, multi-host execution, ownership transfer, embeddings/vector service, UI, streaming, public SDK stability or business exactly-once claim.

## Review Focus

1. A valid provider result proposes another tool/endpoint or extra fields: schema and ordinary branch guards must prevent tool authority manufacture (Tasks 2–4).
2. Receipt persistence succeeds and settlement fails, or cancellation races with either: retain known evidence, accurate pending/unknown state and existing Close ownership (Tasks 3/5).
3. Canonical compilation and schema validation under shuffled maps, invalid references and bounded loop exits; no missing artifact after reopen (Task 2).
4. Corpus path disappears or bytes change after provisioning: lookup/replay use retained snapshot, exact version/span checks reject invented citations (Tasks 3–5).
5. A copied/renamed database and two independent stores: stable original keys, distinct fresh namespaces, no silent rekey or invented fork safety (Tasks 1/5).

## Task 1: Versioned namespace and external keys

**Files:** `internal/executor/{store,external,inspect,replay}.go`, a focused namespace test file; mechanically update affected runtime/process contract expectations only where the now-authorized key change requires it.
**Produces:** durable 256-bit random namespace; canonical external key derived from key-version, namespace and local intent ID. Existing local journal/message/lifecycle IDs stay unchanged. Expose namespace through inspection and a narrow accessor/key derivation helper for later adapters.

- [x] Reproduce the merged baseline before implementation; add failing two-store/shared-endpoint, reopen/rename/copy and old-version checks.
- [x] Initialize schema/key version 2 and namespace atomically before external work. Reject unsupported version-1 stores before live activation/dispatch; no migration or historical rekey. Retained old archives remain the historical reproduction path.
- [x] Reconstruct/validate every retained fixture request using the stored namespace. Keep loopback enforcement, immutable parameters, late authority and old causal oracle protections.
- [x] Demonstrate two identical logical programs against one durable endpoint have separate keys/mutations, and reopen each preserves its original key. Copy retains identity; concurrent independent use of copies is unsupported.
- [x] Run focused and complete current-source regressions, retain any failure, report exact interfaces/change list for review.

## Task 2: Authoring compiler and public facade

**Files:** new `author/` (compiler and tests), small public `runtime/` facade delegating to `internal/executor`, narrow program metadata/validation changes, external-consumer smoke under `examples/` or a temporary external module test.
**Consumes:** existing reducer, durable artifacts/provisioning and version-2 store. **Produces:** `author.Compile(Definition) (Artifact, error)` and static application/component definitions; public Create/Open/Run/Inspect/Replay access without external imports of internal packages.

- [x] Pin observable tests first: author uses named values, labels, sequential/conditional steps, named effects with correlated waits, ordinary request/reply, success/failure and bounded repetition without raw opcodes/PCs/SQL.
- [x] Compile readable author steps to existing generic instructions. Use the current bounded flat typed record representation unless a demonstrated application need requires more; the first response/citation schemas can be flat records. No assistant-specific opcode.
- [x] Persist compiler version, semantic version, input/output/effect schema identities, prompt/template bytes and execution-position/name map within the immutable artifact. Preserve legacy program encoding/semantics where supported; reject unknown versions explicitly.
- [x] Reject duplicate/missing labels, references not definitely available, wrong types/bindings, invalid branches, unsupported effects and unbounded author repetition. Identical definitions under shuffled maps encode identically.
- [x] Run external-consumer compile/run/reopen/replay without internal imports. Freeze exact public interfaces in the task report for the next slice; review before proceeding.

## Task 3: Narrow adapters and retained corpus

**Files:** `internal/executor` adapter request/attempt/result/settlement support and Run integration; public adapter contract as needed; `adapters/openai` and `adapters/lookup`; corpus snapshot support; focused tests using real HTTP serialization/parsing.
**Compatibility:** adapter/corpus ledgers advance schema 2 to schema 3, keep key version 2, and reject previous layouts before activation; no migration or key rewriting. The namespace slice evidence remains under its original version.

**Consumes:** retained author schemas/artifacts, Store namespace, existing bounded owner. **Produces:** one-request completion candidates, durable attempt/result records, runtime-configured adapter registry, retained immutable corpus assets.

- [x] Tests first for committed/authorized dispatch, correlation/immutable conflicts, mixed fixture/adapter capacity, late evidence, known-unsettled results, invalid tool names/parameters, and credential exclusion.
- [x] One bounded worker pool/owner across supported transports. Adapter receives immutable request/assets and cannot access Store, choose policy, execute another adapter or mutate journal. Retain request/result/schema identity and validation disposition before ordinary consumption.
- [x] Thin nonstreaming OpenAI Responses HTTP: strict `text.format` schema, full retained inputs/instructions, explicit model ID, `max_output_tokens`, `store:false`, no hidden retries/redirects/remote conversation. Classify HTTP rejection, unknown transport, refusal, incomplete, invalid schema and validated output separately; retain bounded request/response, response/request IDs, requested/returned model and actual optional usage. Credentials are runtime-only and excluded from retained bytes/errors.
- [x] Snapshot explicitly selected corpus files (or bundled nonsensitive texts) at provisioning. Lookup is read-only over retained bytes, bounded token matching, returning one document/version and exact snippet with byte offsets. No live filesystem reread for recovery/replay.
- [x] An authorized attempt lacking a durable model result remains Unknown on reopen with explicit disposition and no retry. The same conservative no-repeat rule may be used for the local lookup; known results always settle without reinvocation. Existing idempotent/non-idempotent fixture profiles remain separate.
- [x] Focused protocol/race tests and current runtime regressions; exact adapter/CLI integration interfaces and controlled fixture inventory to review.

## Task 4: Ordinary nested application and CLI

**Files:** new reusable example package and `cmd/alan-evidence` (or equivalent small command), public external-consumer example, bundled corpus, README/runbook additions and application tests.
**Consumes:** author API, public runtime, OpenAI and lookup adapters. **Produces:** runnable `init/run/inspect/replay` application with configured provider/model and static composition.

- [ ] A caller uses only the evidence helper's declared request/result endpoint. Inside, an ordinary planning/synthesis program requests an ordinary retrieval component which emits lookup and returns evidence. No caller access to child state/names, no runtime role branch.
- [ ] Ordinary program selects lookup, clarification or insufficient evidence from recorded validated model output; invalid/refused/incomplete output yields explicit failure. The adapter never directly executes a proposed tool.
- [ ] Successful path: question -> model proposal -> ordinary validation/branch -> separately committed authorized lookup -> retained evidence -> second model request -> ordinary citation checks -> published answer. Require exact retained document/version/snippet equality for the single citation, and report this as mechanical provenance checking rather than proof of every answer sentence.
- [ ] Persist per-question model/lookup spent counters; defaults max 3 model requests/max 2 lookup requests, configurable positive bounded token/output sizes. Stop at explicit budget disposition without resetting on physical reopen. Expose logical requests, actual attempt counts, optional usage and unknowns separately.
- [ ] Controlled responses exercise all branches, wrong tool/endpoint/citation, nested contract and author-readable inspection. Opt-in live mode requires model+credential; default tests never spend money.

## Task 5: Process qualification, review and publication

**Files:** subprocess controller/tests, controlled durable HTTP provider fixture, negative controls, `docs/foundational/phase1e/*`, appended decisions, final README.

- [ ] Retain controlled endpoint contacts independently. Kill only after acknowledged model-result/tool-result/unknown-provider boundaries; reopen same store with budgets and IDs intact. Recorded tool result survives source-corpus change/removal. Replay with no credentials/provider/corpus path causes zero adapter invocations.
- [ ] Requalify held model plus independent peer/caller progress, saturation and reordered mixed completions, cancellation/closure/late evidence and old Phase 1D/1D.1 cases.
- [ ] Unchanged independent checks reject at least one test-only bypass of record-consume-validate-tool-commit ordering or replay-triggered provider contact. Preserve exact counterexamples and complete source inputs.
- [ ] If credentials and explicit model become available, run only the bounded bundled-corpus live probe and offline replay. Otherwise record live NOT RUN and usable command; controlled evidence is not described as live inference.
- [ ] Full/focused/race including children, vet, gofmt, module verification, previous research/runtime comparisons; raw causal evidence, source/dependency manifests and complete credential-free archive/offline reproduction.
- [ ] Independent adversarial whole-branch review of source/contracts/fixtures/evidence; correct or explicitly scope legitimate findings, retain originals. Commit/push dedicated branch and open draft PR against verified base. No merge/release.

## Alternatives and stopping point

A free-form callback application would create unreplayable hidden control, so builders run only at artifact construction. A general structured-data calculus is unnecessary for the first flat decision/result records; add generic capabilities only when a concrete test proves necessity. Structured response proposals keep the tool authorization boundary smaller than native provider tool loops. Unknown provider inference is conservatively blocked rather than automatically repeated; no reconciliation/repeat UI is claimed. Stop with the usable application, controlled evidence, live status, reviewed source and draft PR.
