# Phase 1F live investigation and deep-composition implementation plan

> Use subagent-driven development: sequential implementation owners, scoped independent reviews, and final adversarial review. The user authorizes this application-led increment; existing global instructions choose delegation and draft-PR publication. This plan proceeds within that scope rather than adding a design-approval pause.

**Goal:** A reusable investigation component answers bounded technical questions from frozen evidence, adapts through ordinary review/revision, runs independently or nested, and retains tested recovery/replay behavior.

**Architecture:** Reuse the public `author`, `runtime`, `schema`, `effects`, Responses/lookup adapters and one-question `examples/evidence` helper. A finite authored coordinator plans two subquestions, sends both to ordinary helpers, synthesizes their public results, receives a separate ordinary review component's decision, and either publishes, abstains, or explicitly authorizes one revision using statically provisioned next-round helpers. An ordinary outer report component demonstrates the identical investigation port at another depth. Existing one-PC scheduling and immutable effects remain unchanged.

**Baseline:** Fresh `origin/main` is `e2157a8fb2837ea3d7d0232683072182a0ee832b`, merging PRs 6 and 7. Its tree equals Phase 1E publication `aa925516f9d7fd39668c7d507813a5dae06b9d8a`; all 133 executable/test/tool/input files match source SHA-256 `8f95bda980504caec8c29cbe196297f61abafe7941d11ddc51effc19c9321caa`. Reuse the clean attached worktree on `codex/phase1f-investigation`. The original checkout and its `.env` are not modified. Prior phase evidence remains unchanged.

## Frozen initial scope and alternatives

- Technical change/readiness investigations over a small synthetic, nonsensitive document corpus. Two subquestions per round, initial round plus at most one revision; all limits are application limits, not new architecture constants.
- Four development and four held-out questions, each split covering supported answers, ambiguity, missing evidence and contradictions. An independent evaluator freezes corpus/questions/rubrics before application tuning. Held-out content stays sealed from the implementer until code/prompts freeze; no tuning against its answers. Small-sample results are descriptive, not general benchmark claims.
- Reuse one-shot evidence helpers by static expansion per round. Unused future-round participants may remain waiting; no implicit child cleanup, cancellation or reassociation is claimed. Bounded waiting and Unknown may stop useful progress explicitly.
- Public flat records and named operations are the first attempt. Record concrete authoring obstacles before any extension. Prefer reusable record/wiring helpers if necessary; no new interpreter opcode without a smallest executable counterexample of missing semantics.
- A single opaque model/tool callback would hide ordinary decisions, and a dynamic supervisor/runtime redesign is outside scope. Fixed static expansion preserves the current boundary with a small, auditable cost.
- Live budget: user authorized approximately $10; treat $10 as the maximum total for the Phase 1E gate and Phase 1F experiments. Select and record an officially documented, account-available model. User prefers Sol 6.1 for moderate work and permits a cheaper model for easy work. No secret enters logs/artifacts; model and budget must be configured before paid inference. Unknown outcomes reserve possible spend and are never automatically repeated.

## Tasks

### 1. Baseline, evaluation freeze and existing live gate

- [x] Reproduce merged Phase 1E source and complete normal/race qualification plus existing comparisons without relabeling old evidence.
- [x] Freeze the independent corpus, development questions, sealed held-out questions and support/provenance/abstention/revision rubric. Record hashes and unsealing policy.
- [x] Run the existing evidence CLI and direct adapter with explicit model/call/token/monetary limits, retain actual outcomes/usage/provider IDs/citations and offline replay. If unavailable, mark live BLOCKED and continue independent work.

### 2. Public-consumer investigation and nested application

- [x] Write public-contract tests first; author two subquestion branches, synthesis, separate review and one bounded revision or explicit stopping using only public APIs.
- [x] Retain logical budgets, original pending IDs, round and review decisions in ordinary state. Unknown work is not a new revision. Validate public evidence/citation fields mechanically.
- [x] Demonstrate the same `New` component independently and inside a larger authored component. External consumer imports no internals, raw instructions, SQL or private child state.
- [x] Record obstacle inputs and narrowly justified authoring improvements, if any. Provide init/run/recover/inspect/replay CLI. Review the slice before proceeding.

### 3. New-boundary qualification

- [x] Reuse actual subprocess/barrier and independently logged controlled-provider machinery. Exercise partial sibling completion, interruptions during review and revision, retained budgets, late results and Unknown provider outcomes.
- [x] Observe exact source/intent/authorization/epoch/sequence correlation; invoke durable binding checks on actual wildcard filters. Check offline replay has no dispatch authority.
- [x] Add meaningful negative controls for new contract/monitor claims. Retain every failure, original input and source; independently review runtime assumptions and harness causality.

### 4. Usefulness evaluation and delivery

- [x] Freeze final implementation/prompts before unsealing held-out inputs. Run the simple baseline and investigation with the same corpus/model/configuration and bounded live spending. Preserve all valid and unsuccessful outcomes.
- [x] Independently grade support, citation provenance, appropriate abstention and revision changes; report latency and actual available model usage/cost estimates separately from durability and the application's own review verdict.
- [x] Run focused/full/race/vet/format/dependency and earlier-phase regressions; retain exact source identities, complete reproducible source and machine-readable evidence.
- [x] Obtain final independent review, update README/decisions/runbook, commit the increment and open a draft PR against verified main. No merge, release or further runtime development.

## Review focus

1. A sibling completes while another waits: its retained result and public correlation must survive interruption without requiring the waiting sibling to finish (Task 3).
2. Review requests revision after evidence is unknown: no fresh operation may accidentally reinterpret or repeat the uncertain one; budgets cannot reset on restart (Tasks 2/3).
3. Citation-valid but unsupported answers or reviewer-approved contradictions: independent grading must separate support from provenance and application review (Tasks 1/4).
4. Same component used at different depths: no caller accesses private child state or runtime branch names; static wiring and port contracts are checked (Task 2).
5. A process acknowledgment has the right token but wrong epoch/sequence: real-filter monitors must reject against retained records, preserving Phase 1C/1E oracle regressions (Task 3).

## Exclusions and stop

One host/active executor, static subjects/grants and existing SQLite authority only. No distributed ownership, dynamic spawning, new provider catalogue, general migration, stable SDK, UI, hidden orchestration or unbounded liveness claim. No business-level exactly-once inference claim. Stop after the reviewed runnable application, measured results and draft PR.

## Recorded development amendments

The original scope above is retained. Task 2B changed three ordinary prompt constants after the first development study: concise lookup queries and explicit retrieved-subset/completeness instructions. Task 2C adds optional exact-span exclusions after a second study exposed repeated retrieval through a query-only public request. The original prompts, lookup inputs, application sources and both studies are retained with their identities. Neither amendment changes the runtime, scheduler, corpus, evaluation rubric, model settings, maximum model/lookup calls or revision count. The shared helper prompt change affects both systems; the optional exclusion profile affects only investigation revision helpers. The final study therefore compares declared application configurations, not an isolated algorithm or equal compute. See `evaluation-run-plan.md`, `development-prompt-revision.md` and `retrieval-exclusion-contract.md` for the recorded scope and limitations. Heldout content remains sealed until final implementation review and source freeze.
