# Task 4 independent binding, custody and physical-control review

2026-11-04. **Scoped specification APPROVE / code quality APPROVE. No blocking finding in the reviewed candidate.** This approves the new fixture/control implementation and the supplied development-source evidence within the stated two-principal contract. It is not whole-phase approval, independent model/runtime correspondence approval, or approval of a final committed/re-exported corpus. The recovery reviewer separately owns SIGKILL and physical-admission timing review.

## Inputs and method

Read `task-4-brief.md`, `task-4-report.md`, frozen contract v3, the selection-narrowing report, the complete 2,253-line `task-4-review.diff`, and all eight new source files in `/private/tmp/alan-phase3b3-runtime`. Every diff body is exactly its inspected new source file. Independently verified all **304** source/module entries in `task-4-source.json`: no size/hash mismatch. Source is `347094a7b18d351c6ad18df526ef6133777bca44` plus these eight uncommitted files; core primitive remains the separately reviewed `4f6f5f2` implementation. HEAD alone does not identify the fixture.

Read actual raw Inspect exports, complete history rows, admissions, intents, authority/attempt/result descriptors, serving-root charges, endpoint ingress/commit journals and endpoint snapshot JSON from the ordinary and race corpus roots. Case names were used to locate intended controls, not as proof of their verdicts. In particular, the independent completed-custody check selected its inputs from raw two-run terminal statuses, without using `Classification`, and then derived the original request/service/reply links from retained rows.

Executed only two new focused, read-only Python assertion scripts over exported JSON; no runtime helper or model oracle is imported:

- `/tmp/alan-phase3b3-task4-binding-check.py` → `...-binding-check.json`: **32 completed raw histories / 64 original service custody chains**, no failures, exit 0.
- `/tmp/alan-phase3b3-task4-controls-check.py` → `...-controls-check.json`: **28 recorded focused control checks** across ordinary/race, plus missing-selector and no-fallback assertions, no failures, exit 0.

These are targeted review assertions for supplied evidence, not a hostile-input-complete independent authority checker. Script/output, source-freeze/diff and case-index hashes are retained in `/tmp/alan-phase3b3-task4-binding-review-evidence.json`. No SQLite database was opened; the reviewer read only JSON, source and journal bytes. No suite was rerun, no source/index/branch was edited, and no subagents or commits were used. Only this review report and temporary review scripts/results were written.

## Full actual-result custody and service policy

`examples/delegation/reciprocal.go:141–159` binds values from the actual original receipt, branches on accepted, emits a result-bearing reply carrying original request/service identities and mutation/total, and selects the peer reply. The ready message only controls sequencing. S1 replaces the unused lifecycle emit/receive with ordinary set reductions (`:161–164`); its final raw histories contain no lifecycle grant result. S2 has no original external A permission.

`internal/durabletest/phase3b3_runtime_test.go:135–210` checks the full server/requester RunID, original service, actual retained receipt, result selection, reply production and requester selection against independent endpoint receipt bytes. `reciprocalSelectedHistory` at `:379` joins **subject and generation and sequence** to the exact selected observation. Equal sequence numbers belonging to a and b are not interchangeable.

The independent raw check confirmed all 64 chains: emitted message intent → accepted generation-bound admission → exact server receive in its own history → immediate next service intent → actual retained receipt → exact server result receive → actual result-bearing reply → accepted requester admission and exact requester receive. Receipt mutation/total equal the carried values; endpoint ingress requests equal the immutable retained request, and committed endpoint receipts equal the retained receipt. Original source key, authorization and epoch remain bound. Every allowed original has one effect debit in its actual serving root.

Both original requests are committed before A's first service. A's own result is still unpaid then. B commits its service only after selecting A's authentic BA result-bearing reply. Thus these completed histories realize A-servicing/B-serial with two outstanding originals, rather than an empty-ack or paid-flag substitute. Terminal-observation completion remains separate from the two original result selections.

Representative ordinary raw witness: `/tmp/alan-phase3b3-runtime-development/qualified-ordinary/ordinary-s2-handshake-2594529318/` (`final/inspect.json`, `final/history-000.json`, `endpoint-{a,b}.jsonl`, `endpoints.json`). Matching race witness: `qualified-race/ordinary-s2-handshake-160131873/`. All other successful two-run final snapshots, including process recovery suffixes, were checked by the same raw joins; this does not independently approve their process-cut timing.

## Attenuation, original identity and negative meaning

The raw complete-tuple relation remains `(Kind, Source, Target, Operation, Revision, Obligation)`. Accepted delegated effects point to B's actual accepted lifecycle result, exact root/recipient/service/original request and admitted selected observation; serving-root accounting is independently checked. The ordinary binding-control exports show the expected refusal at the actual boundary, rather than concluding from zero contacts alone: removed original tuple/root/issuer/recipient errors refuse issuance; service sequence/resource/selected-grant errors refuse the original effect; no gate or debit is created for those refused attempts.

The controls are substantively distinct:

- `ordinary-s3-missing`: a real accepted grant exists, but the actual selector is nil; the retained reason is `authority_missing`. It is correctly described as missing-selector evidence.
- `first-refused-then-grant`: `before-grant/inspect.json` contains **zero lifecycle results**, a real delegated selector and first `authority_grant_missing` refusal. `after-grant` contains one later accepted program grant, while the original attempt is byte-equivalent as JSON, still refused, with no gate, charge or contact. Reopen did not repair that original.
- `control-invalid-selected-no-own-fallback`: raw original permissions include A's exact own tuple, but the explicitly selected invalid delegated path is refused; it never falls back to A's permission.
- `actual-prior-service-result-as-grant`: B7 is an actual allowed external original with an actual endpoint receipt. B selects that result, emits B10's genuine result-bearing reply, and A selects it at A3 before its A5 laundering attempt selects GrantID=B7. A5 refuses with no gate/contact at endpoint B. This is a real prior-result control, explicitly under reversed service ordering, not a guessed nonexistent result or the positive service policy. Ordinary path: `qualified-ordinary/actual-result-laundering-3711814529/`; race path: `qualified-race/actual-result-laundering-2100418879/`.

For delayed selection, raw A2 selects genuine BA, A3 selects ready, and A4 is refused with `authority_request_not_admitted`. The original is admitted and retained; interpreting that reason as a missing original would be wrong. This is the documented **model-valid delayed-selection/runtime-contract-refusal narrowing**. The current report and case Interpretation preserve it. No model/runtime agreement is inferred from the test expecting refusal.

## Real Cartesian physical controls

`internal/durabletest/phase3b3_adapter_test.go:33–47` maps `add-one` to wire Delta=1 and `add-ten` to Delta=10; untrusted payload delta=999 cannot choose the operation or endpoint. Capability maps to immutable deployment endpoint, and Execute checks the receipt's exact key/profile/delta before producing its normalized actual result (`:49–82`).

Inspected both corpora's four cases, reconstructing the verdict from each claim's full original row membership in `composition_permissions`, not from its case name. Both legitimate rows survive issuance → selected exact effect witness → allowed gate → actual endpoint request/application → retained adapter result → exact A result selection. The actual physical totals are **1** at R1 and **10** at R2, with corresponding wire and endpoint receipt equality; the unused endpoint has zero contacts. Both crossed rows are absent from the relation and refuse issuance with `authority_permission_missing`, no effect/gate/debit/contact. This attacks tuple Cartesian amplification using real distinct actions, not obligation relabeling over an unconditional increment.

Representative ordinary paths: `qualified-ordinary/cartesian-r1-add-one-3239573361/`, `cartesian-r2-add-ten-339000223/`, `cartesian-r1-add-ten-1891980157/`, `cartesian-r2-add-one-3859714346/`. The corresponding race cases are independently checked in the controls script output. These are service controls, not claims that both reciprocal obligations complete in the adapter fixture.

## Unknown prefixes and hostile mutants

Generic adapter unknown retains the exact original attempt, one gate and debit, one independent endpoint contact/application, no local adapter result/settlement/selection, and the same original in fresh-owner `UnknownAdapters`, with no new dispatch. Applied-unknown external followed by exact-root revocation likewise retains its old application and debit, no invented local receipt or reply, and a later refused gate with the same original in `UnknownExternal`. The raw control checks confirmed these facts in both corpora. They are explicit blocked/unknown prefixes, not successful completion or atomic model Apply events. The recovery reviewer retains ownership of fresh-epoch and cut mechanics.

All five hostile histories contain an actual one-row SQL mutation and remain independently raw-exportable. Their four reader outcomes are real rejections: missing grant → missing row; forged grant origin → corruption; result-as-selected-grant → original-prefix witness mismatch; altered gate attempt digest → historical gate-policy mismatch; substituted receipt key → original-result binding corruption. The gate mutant recomputes its record digest, so a stale hash alone is not the stated oracle. These are meaningful internal-consistency rejection controls, **not** the Task 3 independent authority checker or a proof against arbitrary coherently fabricated Store histories. Raw mutation inputs, exact before/after bytes and SQL, base provenance and each reader outcome are retained in both indexed corpora.

## Verification and limits of approval

Inspected the retained qualified ordinary/race logs: each records 53 passing test/subtest markers, no failure/skip markers, and package PASS (21.063s ordinary; 129.224s race). The indexes each contain 48 cases. Inspected the final empty Staticcheck log, ordinary full-module output and worker verification summary; the latter's 1,761 digest checks per corpus are attributed to that recorded verification, not claimed as rerun here. The new raw assertions above and 304 source hashes were actually checked during this review.

The two reciprocal action tuples/resources differ, so mapping them to the unchanged model's same-action vocabulary requires an explicit whole-action abstraction. Separate grant/ready retention maps only to the declared LateGrant subset; delayed-selection refusal, revocation, repeated exposures and applied-with-no-local-result prefixes must retain their stated boundaries. Endpoint stable-key atomic deduplication and trusted adapter/deployment behavior remain assumptions. Store-only gates do not prove physical contact or uniquely identify the returning exposure. No distributed-owner, arbitrary-schedule, N-principal or universal completion claim is established.

Parent still owns source commit/freeze and final re-export, independent Task 3 checker/model classification and manuscript reconciliation. **Final verdict remains scoped specification APPROVE / quality APPROVE for Task 4 attenuation, actual-result custody and physical-control evidence; no whole-phase approval is given.**
