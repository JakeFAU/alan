# Phase 1I Behavioral Contracts Implementation Plan

> **For agentic workers:** use superpowers:subagent-driven-development for the
> bounded tasks below. The assignment's explicit independent parallel authorship
> takes precedence over a skill's sequential implementation default.

**Goal:** test whether a separately checkable behavioral contract preserves
nontrivial application guarantees in the declared finite composition contexts.

**Architecture:** independently authored ordinary public-API candidates, a separate
finite model/oracle, and sampled retained-runtime/process witnesses. Preserve
existing runtime/compiler/accounting semantics; a repair needs a retained minimal
counterexample and narrow review. All new documentation stays in this phase.

**Tech stack:** repository Go 1.27.3, standard library, existing pinned SQLite
driver v1.14.52; no new dependency or paid call.

**Spec:** [assignment](assignment.md) and [K1](01-substitution-contract.md).

## Baseline and research boundary

Verified fetched main/PR17 merge: `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`.
Tree `158d2a24ece0b14f99018921797a7f6f6a45a465`. No intervening main changes;
clean initial tree. Reuse the existing isolated worktree on new branch
`codex/phase1i-contextual-contracts`. [Baseline identity](evidence/baseline/identity.json)
defines the exact tracked Go/module/AGENTS input inventory, source digest
`ada0c277824f94c7724a154965419132bc35d08a41809dc52cff14327026c037`.
This is its declared inventory, not the historical Phase1H inventory.

The initial `go test -json -count=1 -timeout=10m ./author ./schema ./research/semantics`
passed 280 test/subtest events in 6.290s across 3 packages. Raw output, exact
environment and actual selected names are retained under evidence/baseline.
Phase1H full-suite evidence remains historical; no fresh full-suite claim.

The weakest schema/result-only claim may fail under shared budgets. The existing
Definition4.6 explicitly requires matching debits, so such a witness does not
refute its premise. Investigate interference stability and concrete scheduling
separately, and reconcile directly dependent recovery prose. Acyclic witnesses
avoid reliance on the open general loop-ranking proof.

## Global constraints

- Fresh invocation/deployment substitution only; recover each artifact's own
  retained state. No migration or replacement of retained continuations.
- One root, one replacement position, one peer, one optional helper and at most
  one wrapper; finite payloads and resource bounds specified by K1.
- Pure A/B, weak-budget W_A/W_B, equal-charge P/Q and effectful companion profiles
  have distinct identities and claims. No effect-equivalence inference from pure
  completion tests.
- No privileged roles, SQL in candidate applications, arbitrary callbacks,
  second scheduler, new generic DSL, new infrastructure or live provider calls.
- No copy/coordination of A/B implementation logic before each first version is
  frozen. Authors get the same contract and public API references. Record their
  independent first hashes and later integration edits.
- Model/oracle must not import compiler/reducer or derive expected answers from
  running A/B. Record decoding may be shared and is a declared trust boundary.
- Every failed input, contract revision and review finding stays attributable to
  its original source. No passing empty selectors, fabricated gates or proof by
  agreement. No merge/release or external review requests.

## Approaches and decisions

1. Attempt a general congruence proof first: rejected as the execution path;
   unresolved scheduler/interference premises would obscure the bounded question.
2. One public contract with explicit demand envelopes, separate conformance and
   context checks, plus minimal runtime witnesses: selected. It can support a
   precise bounded result or expose a small counterexample.
3. Enforce equal cost in all candidate implementations: unnecessary for the
   selected refinement guarantee; retain equal-charge equivalence as a separate
   probe rather than disguising extra cost with artificial effects.

## Review focus

Prefix resource compatibility with peer spending and outstanding work; coherent
identity renaming and application correlation; response absence versus terminal
success; closure before/after reservation and unknown recovery; independently
checked behavior rather than replay/compiler self-agreement. Separate local
component simulation, reference C[K1] products, implementation products, and
artifact/runtime correspondence; none substitutes for another.

## Task 1 — Freeze and review the boundary

Files: this plan, contract, evidence/baseline, evidence/reviews, 05-decisions.md.
Independent claim/API audits inspect actual manuscript and APIs. A fresh reviewer
challenges K1 and the task plan before candidate authors begin.

- [x] Verify merge/source and reproduce focused baseline.
- [x] Read prior phase contracts/results/review/runbook and inspect dependencies.
- [x] Draft K1, assumptions, finite census and profile separation.
- [x] Preserve the rejected original at evidence/reviews/contract-v0.md.
- [x] Incorporate R1–R3: finite transition/environment rules, asymmetric local
  relation, separate C[K1] checks, K-derived ticket entitlement, and complete
  local/envelope/caller/S/wrapper tables. See contract revision 1.
- [x] Obtain independent scoped rereview of those corrections; retain REJECT
  and its disposition. APPROVE released candidate/model authors.
- [x] Freeze revised contract digest and identical protocol/API author/model
  briefs. Briefs must include finite integer domains as semantic/environment
  restrictions (schema.Field has no integer Min/Max), absent=>missing_reply,
  exact original-handle consumption, pending without release, and local versus
  wrapper envelopes. Assign local A/one-helper B architecture explicitly;
  decision logic stays separately authored with no cross-reading before freeze.

## Task 2 — Independent candidates and finite model

Files: `examples/substitution/protocol`, `examples/substitution/a`,
`examples/substitution/b`, `research/substitution`. Shared protocol contains only
schema declarations; both candidate APIs are `func New() (author.Template, error)`.
Each author works only in its assigned candidate directory and writes a report
with first-version hashes; no reading the other implementation before freeze.
The model consumes K1 directly and owns independent monitors usable on normalized
runtime observations. Implement the frozen K1 and callers before transcribing
A/B. First check the reference C[K1] products separately from local A/B relation
and C[A]/C[B] products. The 3,840 configurations and 84 initially completion-entitled
configurations are fixed by contract/context rules, never by a candidate
finishing or abandoning work. Local graph initial states/environment alphabet
are specified separately and are not counted as these context configurations.

After A/B first versions freeze, independently transcribe their actual retained
artifacts and archive an instruction/branch/handle-to-graph correspondence.
Review both branch arms, prelude sets, output values and every intent/receive.
Do not import production compiler/reducer or generate expectations by running
them. Shared JSON decoding is an explicit trust boundary. Graph conformance,
inspected correspondence and sampled concrete traces are separate evidence;
none claims a compiler proof or exhaustive concrete-runtime conformance.

- [x] Freeze local conformance tests before implementation where feasible; retain
  actual RED commands and do not claim broader tests-first chronology.
- [x] Author/freeze A and B independently using the same public contract.
- [x] Implement frozen K1 transition graph/environment, asymmetric local checker
  and separate reference context products; preserve stuck/progress distinctions
  and reject internal stuttering cycles rather than accepting trace inclusion.
- [x] Independently transcribe/review frozen candidate artifacts, then check
  A/B local relations and named products. Maintain initial entitlement and
  K-ticket prefix invariant even when a candidate misbehaves.
- [x] Report each graph census, transitions/outcomes, acyclic rank/cycle analysis
  and shortest counterexamples. No scheduler-idle self-loops or hidden cutoffs.
- [x] Execute weak-budget and equal-charge probes, negative controls and census.
- [x] Review candidate/model agreement independently; retain fixes as new versions.

## Task 3 — Retained-runtime and process witnesses

Files: `examples/substitution/consumer` (separate Go module),
`internal/substitutiontest` (integration-only controller), narrow phase tools.
Public candidate applications import only exported author/schema/runtime/effects;
test-only environment lifecycle operator and observation barriers are explicit.
Reuse boundary/process facilities without copying Phase1H's evidence machinery.

- [x] Test bare/sequential/parallel/hidden/wrapped actual execution, demand,
  correlation, absence/failure/pending and insufficient budgets.
- [x] Record concrete P/Q schedules separately from abstract trace sets.
- [x] Execute real ACK/SIGKILL/fresh-open cuts on pure candidates; verify stable
  operations/observations/charges and changed activation authority.
- [x] Qualify closure boundaries, one unknown non-idempotent request across
  reopen and separate live late evidence; replay with no dispatch authority.
- [x] Execute valid faulty programs and require unchanged monitors to reject
  actual wrong application id and accepted extra subtree charge. Unknown-to-success
  is not a required negative; a runtime publication join blocking a premature
  success is recorded as a guard result, not a semantic negative detection.
- [x] Review code, observed records and crash harness, then run focused race checks.

## Task 4 — Qualification, manuscript and final review

Files: 02-qualification.md, 03-review.md, 04-runbook.md, 05-decisions.md,
compact qualification.jsonl/source manifest and traces; affected paper MD/TeX.
BibTeX changes only if new verified primary references are used.

- [x] Qualify final contributing source with exact command/source/pass names;
  model, public consumer, process and applicable race checks; vet/gofmt/build.
- [x] Run existing focused invocation/accounting/replay/semantic regressions.
  No production change is planned: do not repeat Phase1H's multi-hour whole suite
  merely by precedent. Broaden gates if a production boundary actually changes.
- [x] Reconcile Lemma4.7/Theorem4.8 premises and dependent §6 claims with evidence;
  keep unrelated theorem/representation obligations explicitly open.
- [x] Retain concise reproduction/source identity; no recursive archive copying.
- [ ] Independent whole-branch review, resolve actionable findings, commit/push
  and open a draft PR against verified main. Leave merge to owner.

## Progress and exclusions

Stage 1 and the candidate/context slice were independently approved. The finite
model and actual runtime expose an entitled parallel-completion counterexample:
root success fences its still-open peer. All original entitlement/configurations
remain. Independent model/runtime reviews found decoder defects after the first
236 normal/race witnesses; the correction passed scoped rereview and 295 final
normal/race cases. Actual probe transcription replaced the reviewed surrogates.
Whole-working-tree review returned APPROVE; exact postcommit verification and
draft publication remain the final steps.
The original baseline and superseded evidence retain their source-specific labels.
Excluded: arbitrary recursion/context grammar, multiple replacement sites,
distributed ownership, eventual recovery under infinite interruption, unknown
non-idempotent repair, new reconciliation APIs, business exactly-once and stable
SDK claims. A finite negative result is an acceptable research completion.
