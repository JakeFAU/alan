# Phase 1I independent whole-working-tree review

**Decision: APPROVE.** No unresolved actionable finding remains in the reviewed Phase 1I working-tree scope. This approves the bounded research result and its evidence package, including the preserved negative result; it does not prove general substitution or recovery equivalence.

## Reviewed identity and scope

Base commit `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`, base tree `158d2a24ece0b14f99018921797a7f6f6a45a465`, branch `codex/phase1i-contextual-contracts`. Reviewed current source identity: **`a97e9893e2218a2245953e720de4939c7aca53503b51ab2e9e76a1164ee12c38`**. Independently recomputed all 284 source/input hashes with zero mismatches and reproduced the documented path-sorted aggregate. The earlier `97decf...` identity used the same bytes in a different ordering; the final manifest now specifies canonical ordering explicitly.

This is approval of the complete current working tree relative to that base, before its narrow commits. A subsequent commit package must preserve these reviewed bytes; its exact commit/tree identities require the promised postcommit verification. Review-status metadata may accurately record this decision without changing the qualified result.

Read AGENTS, the assignment and final-review brief, plan, frozen K1, qualification/review/runbook/decision documents, all new application/model/harness source (including the nested public-consumer module), frozen author artifacts and correspondence, prior Phase 1H qualification, original and corrected scoped reviews, relevant original/final execution artifacts, complete affected Markdown/LaTeX changes and their old premises, and unchanged bibliography. Inspected final decoder and actual-probe corrections after their authors finished. Final package audit covers the 44-record qualification index, final source manifest, normal/race records, and final build/static-check records. Every indexed path/hash checked matches.

The change comprises `examples/substitution`, `research/substitution`, `internal/substitutiontest`, Phase 1I research/evidence documents, and scoped manuscript corrections. Production author/compiler/schema/runtime/executor/effect sources, root modules, prior-phase files and bibliography are unchanged. Frozen K1 and first A/B implementation/test bytes still match. No source edits, commits, subagents or paid calls were performed by this reviewer.

## Research result and claim boundaries

The central negative is supported. In the separately named runtime-boundary interpretation, all three graphs retain the original 3,840 configurations and 84 completion entitlements: all eight parallel entitled configurations have a failing schedule, while the other 76 entitled configurations complete. Actual A/B delayed-peer fixtures reproduce ordinary root local success followed by peer `scope_closed` admission refusal, without injected cancellation. Their eventual aggregate root outcome is `failed/member_failed`; local root success is not composite public success. Initial entitlement and the optimistic model's rejection remain recorded.

This falsifies the frozen optimistic `C[K1]` completion/correspondence premise. It neither refutes an implication that assumes that premise nor supplies an unequal-debit counterexample to Definition 4.6. Weak resource probes and equal-charge P/Q probes remain separate. P/Q local relation and exact finite language equality apply to the original declared protocol at the stated capacities; concrete runtime-boundary maximal-outcome membership and 16 concrete samples are separately labeled. There is no runtime-boundary language-equivalence or confluence claim.

The manuscript preserves the exact-debit relation and existing labels, withdraws the unsupported unique-normal-form/confluence argument, and identifies general congruence and dependent future-recovery simulations as open obligations. Affected Markdown and LaTeX mathematical content agrees at the declared scope. Existing loop-ranking and broader dissertation obligations remain open. Inspected compiler evidence supports changed-fragment compilation and records the pre-existing full-TeX undefined `llbracket`; I did not personally compile the manuscript and approve no full-PDF claim.

## Findings and correction review

1. **Actual probe correspondence — resolved.** My preliminary P2 finding is preserved in `/tmp/alan-phase1i-final-probe-finding.md`. The old P/Q graph reused frozen B private control and the old weak helper returned complement1 instead of actual0. The correction independently transcribes six actual programs and all 41 instructions, binds source/template/program hashes, preserves both helper arms and Q's extra Set, and adds RED regressions. Original surrogate sources/results remain. All actual weak/PQ runtime samples now require successful raw evidence validation and profile-specific actual-graph membership. Original-protocol language equality is explicitly separated from runtime-boundary membership. The correction passes my independent full model and runtime-probe reruns.

2. **Prior decoder findings — resolved.** Actual selected observation payload/provenance/generation is validated before normalization; original handles, bindings, parent/root/member, history and delivery links are checked. Local terminal/end/freeze/publication facts are checked independently, preserving local-success/missing-reply distinctions. Explicit root cancellation derives from its accepted lifecycle transition; success's admission fence does not set `Closed`. Original rejected evidence remains, and focused independent reexecution exercises all 59 decoder controls. The scoped rereview's approval is corroborating review evidence, not a substitute for these checks. This does not claim detection of every fabricated store.

3. **Archive package isolation — resolved.** The parent found two preserved snapshots ending in `.go`; their unintended documentation package failed an integration-tag compile. Both were renamed `.go.txt` without byte changes. I verified before/after hashes and independently reran package discovery: exactly 31 packages, no documentation package. The original failure remains recorded. No qualified executable source changed.

No K1/oracle weakening, entitlement reclassification, production repair, deleted negative witness, or new universal claim was used to resolve these findings.

## Independent executions

All Go commands below used `env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build` with Go 1.27.3, darwin/arm64. They exited 0 with no failed or skipped tests:

| Command from repository root unless specified | Actual passed leaves |
|---|---:|
| `ALAN_MODEL_EVIDENCE=/tmp/alan-phase1i-final-model-check go test -count=1 -json -timeout=10m ./research/substitution` | 47 (59 pass events) |
| `go test -tags integration -count=1 -json -run '^(TestReviewConsumedPayloadIdentityPreserved\|TestSubstitutionDecoderClosureFlags)$' ./internal/substitutiontest` | 23 |
| `go test -tags integration -count=1 -json -run '^(TestSubstitutionDecoderConsistency\|TestSubstitutionDecoderCancellationEvidence)$' ./internal/substitutiontest` | 36 |
| `go test -tags integration -count=1 -json -run '^TestSubstitutionRootSuccessFence$' ./internal/substitutiontest` | 2 |
| `go test -tags integration -count=1 -json -run '^TestSubstitutionProbes$' ./internal/substitutiontest` | 26 |
| In `examples/substitution/consumer`: `go test -race -count=1 -json ./...` | 2 |

The displayed backslashes before alternation bars are Markdown table escaping; the executed regexes contain ordinary `|`. Consumer race was also executed earlier without JSON and passed. `go list -tags integration ./...` and `git diff --check` independently passed.

Raw paths, SHA256 values, exact selected names and package results are indexed in `/tmp/alan-phase1i-final-independent-checks.json`. All **33** fresh model result JSON objects exactly match current retained plain/compressed payloads, including full censuses and corrected probes; comparison is `/tmp/alan-phase1i-final-model-output-comparison.json`.

## Audited recorded gates

I independently parsed the final normal-v2 and race-v2 compressed JSON logs: each has **295 passed leaves / 305 pass events**, no missing/unexpected selected case, failure or skip, elapsed 17.369s and 223.397s respectively. All 85 source/module hashes in their conservative input manifest match current bytes. These full normal/race executions were performed by the qualification workstream, not repeated in full by me.

I separately inspected original final raw records for 24 actual weak/PQ probes, nine cancellations and two root-fence cases in **each** profile. Probe evidence/membership checks pass; all fresh cancellations have `Closed=true`; both root fences have `Closed=false`, local succeeded and aggregate failed/member_failed. Original v1 misclassifications remain preserved. Process ACK/SIGKILL/fresh-process identity, pure recovery and external uncertainty records were inspected with their source: pure recovery, unknown non-idempotent reopening without recontact, and surviving late live transport remain distinct experiments.

The final index's build, standalone consumer build/vet, focused model/application/harness vet, formatting and diff checks all record exit0 at the reviewed source identity. Earlier selected semantic/executor regressions and preserved Phase1H inputs are accurately attributed; no fresh whole-Phase1H campaign is claimed.

Independent audit files also include `/tmp/alan-phase1i-final-source-check-v2.json`, `/tmp/alan-phase1i-final-runtime-gate-audit.json`, `/tmp/alan-phase1i-final-runtime-record-audit.json`, `/tmp/alan-phase1i-final-original-evidence-audit.json`, and `/tmp/alan-phase1i-final-package-list.txt`.

## Remaining obligations

General interference-stable contextual refinement/congruence, arbitrary grammar contexts, compiler correctness, full future recovery/world-response coupling, general loop ranking, retained-code migration and distributed recovery remain unproved/out of this qualification. Replay shares implementation with execution and establishes historical consistency rather than an independent semantic oracle. Finite graph/model checks, sampled single-host process behavior, source inspection and independent review have distinct evidentiary strength. Within those explicit limits, the reviewed bounded negative/positive package is ready for narrow commits and exact postcommit identity verification.
