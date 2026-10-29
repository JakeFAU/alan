# Phase 1J independent protocol review v1

**Verdict: APPROVE the frozen experiment design and implementation plan.** No unresolved actionable design finding remains in these reviewed bytes. This is not approval of a positive completion claim, an implemented artifact, a model census, or runtime qualification.

## Exact reviewed identity

Baseline supplied by the controller: commit `d68ec667d276a4bd9e1be11613905c628cfea704`, tree `25c2e22f21181fe70d8f2b2f8bc7e44c512ef948`. This reviewer did not independently repeat ancestry qualification.

| File | SHA256 independently checked |
|---|---|
| `docs/foundational/phase1j/00-plan.md` | `f773775b5ac65a21d63180eec7862c703590f6ea6c279d6669c9d58be9f506dd` |
| `docs/foundational/phase1j/01-join-contract.md` | `a950450d33ab928ccb875f8c98de83a83c956d8759fa0f39a71afe1c06c52a81` |

Read AGENTS.md and the Phase1J assignment, both complete proposed documents, selected Phase1I frozen contract/final review/runtime qualification, actual delayed-peer fixture, public authoring/compiler/messaging interfaces, invocation/aggregate boundaries, reviewed decoder apparatus, and relevant manuscript qualification passages. Approval covers the narrow intervention: unchanged A/B/S and candidate path; unchanged static provisioning and resource allowance; unchanged 32-Set/S-Invoke/original-Await prefix; new root join decision and peer report suffix; additive ports/schema/grant. No runtime, compiler, schema semantics, dependency or accounting repair is authorized by this review.

## Prefreeze findings and their disposition

1. **Ordinary report authorization race — preserved as an open negative boundary.** An ordinary Send commits an intent, not authorization. `internal/executor/messages.go:154-172` rejects a first authorization after sender termination; `messages.go:180-228` translates the denial into message admission refusal. The static peer report is not an invocation reply. The reviewed J1 explicitly separates commitment, first authorization, admission/delivery, consumption and terminal events, and retains Send → peer Success → first authorization as a permitted fair schedule. It forbids silently assuming delivery before termination and distinguishes a Store.Run sample from all modeled schedules. This corrects the initial incomplete scheduling specification. The source-level adverse schedule is a reason to investigate, not an executed model/runtime result in this review.

2. **Peer failure could close shared authority too early — corrected.** A static member belongs to the root scope (`internal/executor/invocation_closure.go:23-42`); a non-success member terminal freezes it without waiting for all members (`internal/executor/invocation_outcome.go:222-271`). The revised peer ends locally succeeded after reporting any known S outcome. That status means reporting-program termination, not S success or guaranteed report delivery. The root retains the actual S failure and decides only after both required outcomes; a pending sibling remains pending. This avoids introducing an earlier shared failure fence in the strict-join controls.

3. **Absent responses are deleted, not default-filled — corrected.** `internal/executor/program.go:412-419` deletes response bindings when presence is false. The revised contract requires a post-await presence branch and explicit zero literals for absent report fields. The oracle must distinguish those fillers from response evidence and verify actual selected payloads.

4. **First authorization needs raw evidence — corrected.** The final raw mapping now explicitly joins attempts/authorization to the original report intent, first disposition/source snapshot and terminal order, then preserves `source_closed` / `source_denied:source_closed`. Message commitment, peer termination and delivery remain different facts. Earlier drafts are preserved by the author as `contract-draft-v1.md` and `plan-draft-v1.md` in this directory; these were prefreeze corrections, not previously approved revisions.

## Feasibility and claim checks

- Public compilation fixes ordinary message source/target generation 1, peer, obligation and provenance (`author/compile.go:280-305`); the exact invocation wait additionally matches the original operation handle (`author/compile.go:307-325`). `author/application.go:58-84` supplies static generation and compatible grant wiring. Application request identity therefore needs the explicit root check; wrong request parks without discharging the expected join. The public interface does not make a peer assertion a runtime receipt. The specified oracle checks the original S handle, selected outcome, payload equality and peer consumption-before-report ordering.
- Frozen delayed-prefix construction is visible in `examples/substitution/probes.go:80-96`, and original S/peer behavior in `examples/substitution/deployment.go`. The plan obtains original S through the existing public deployment rather than silently changing its workload. Exact preservation and compiled-diff checks remain required implementation evidence.
- Root local terminal status closes admission independently of aggregate completion (`internal/executor/invocation.go:163-201`). J1 retains normal root terminal transitions and checks lifetime as a program property; it does not redefine admission-open from Phase1I Closed=false. The report can discharge future S demand before peer local termination; peer and aggregate completion still require separate evidence.
- The reference product keeps K1 local/helper choices separate from artifact mappings. The eight initial configurations, spent-plus-outstanding resource envelope, individual I/R controls, original accounting and zero effects are explicit. All-schedule claims require complete finite exploration and rank/cycle evidence; successful samples cannot substitute for that result.
- The plan preserves C0 at capacities 3 and 4, the decisive root-wait/peer-pre-admission intermediate state, adverse reporting/correlation/failure/absence/pending/cancellation controls, and four original-store process interruptions. Retained-state predicates and actual fresh-process continuation are required; unsupported cuts must remain limitations.

## Review evidence and remaining obligations

This was a source/document review. Executed read-only `rg`, `sed`, `cat`, numbered-source inspection, `shasum -a 256`, and `git diff --check` (exit 0). The latter only checks tracked diffs and is not claimed to validate untracked document formatting. Rechecked both SHA256 values immediately before writing this report. No Go tests, model search, runtime experiment, process kill, TeX compilation, implementation edit, commit or subagent were performed by this reviewer.

The implementation must still establish artifact preservation, exact causal evidence, the independent census/rank, concrete matrices, process cuts and manuscript reconciliation. In particular, the ordinary-report late-first-authorization schedule may falsify all-schedule J1 completion even when lifetime preservation succeeds. Preserve that result if reproduced; do not turn this design approval into approval of completion, an impossibility theorem for all public protocols, or universal substitution/recovery.
