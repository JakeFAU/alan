# Phase 1I scoped independent candidate/context review

Decision: **APPROVE** this candidate/context implementation slice.

Date: 2026-10-28 local / 2026-10-29 UTC. Scope: frozen first A/B sources and artifacts, their author qualification evidence, schema-only shared protocol, and the five ordinary context builders/wiring tests. No actionable implementation, evidence-integrity, or reproducibility defect was found within this slice. This is not approval of the independent model, artifact-to-model transcription, all-context runtime qualification, manuscript, process/effect companions, or the whole branch.

## Inputs and source identity

Read the frozen K1 contract, full candidate brief, approved contract rereview, complete seven-file diff `/tmp/alan-phase1i-candidates.diff`, all seven current source files, A/B reports, context report, first source/template/program artifacts and manifests, all 44 retained author samples, qualification logs/indices, and context RED/failed-attempt/GREEN evidence.

Independently recomputed K1 SHA256: `cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`, matching the author brief and specification approval. The frozen document's older pending-status header does not supersede the separate exact-digest APPROVE record. Final navigation/status reconciliation remains the controller's later documentation work.

Read-only Python checks established:

- The full supplied review diff reconstructs exactly all seven current source files.
- Every A/B first-version manifest hash matches its retained file; current A/B source/test bytes equal their first snapshots.
- Exact decoded program bytes in each template bundle hash to the recorded artifact IDs. A contains 13 instructions; B contains helper 7 and candidate 16. No loop/foreach/next instruction appears.
- All16 A sample hashes match its index. All44 A/B retained samples were decoded and their envelope/presence/payload/accounting behavior rechecked independently of the Go assertions.
- Current context source/test bytes equal the retained GREEN source snapshots. Every context RED/attempt1/GREEN manifest hash matches; JSON Go pass events equal the result summaries:0,4,11 respectively, where GREEN is10 subcases plus parent.

No Go test was rerun: the instruction explicitly allowed unchanged-source evidence review. No model execution, provider call, repository edit, commit, or subagent was used. The only authored file is this `/tmp` report. The record checks above validate retained evidence consistency and source correspondence; they do not turn shared-runtime replay into an independent semantics proof.

## A — local candidate

Source: `examples/substitution/a/candidate.go`; first source SHA256 `c98a5c6d16738d98af9140e6edce6d4df0ddcb51c2330ba974e0e51a82005911`; test SHA256 `033373ad1e384b32b6b9597faa2ed23a3f74b8f8015371d17b4c2df9d3503710`.

Template ID `a03135203f8b728a6aa7b732d06952cc87b70b3aced2c0833293e333760eece5`; exact bundle digest `db5453b6e08abe5ecbf9614d472626c228675ba197abdaa36cad4aa3c5b04409`; singleton program `74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7`.

A consumes the original public request once. Its answer branch directly sends id unchanged and chooses literal 1 for input 0, literal 0 for input 1, then succeeds. Fail terminates failed without reply; absent terminates locally succeeded without reply; pending waits at a second ordinary receive that the frozen environment never supplies. The out-of-profile path after that second receive is not counted as reachable K1 behavior. All control edges advance. There is no helper, extra charge, effect, callback, internal import or hidden host policy.

This matches A's assigned K1 local choice. The ordinary public boundary and exact request/response schemas come from the schema-only protocol package. Integer bounds remain explicit environment premises, correctly not attributed to scalar-schema validation.

The retained author test log contains all 16 id/value/mode cases. Each uses real Create/Run/Inspect/Replay, asserts quiescence below fixed caps, exactly one accepted subtree admission/run, total fixed+dynamic runs 2, exact `(I=1,R=1,E=0)` charges, and local-versus-invocation outcome distinctions. All four pending cases remain open/unpublished at the second receive with caller still waiting on the original handle. Fail and absent are correctly distinguished as `member_failed` versus `missing_reply`. All four answer cases preserve id and complement. Operational caps 128/128 exceed the finite path bound; no cutoff is relabeled as pending.

The retained RED is an explicit unimplemented-function error, transparently described as test-first evidence rather than a semantic negative. No helper denial test is required for a component with no helper; entry-budget/closure/context tests remain later controller work.

## B — one-helper candidate

Source: `examples/substitution/b/b.go`; first source SHA256 `ff813a85585175522e2bf5cc4550caeb6c169c7e507c7ce51ee8bba43ec933e8`; test SHA256 `8e845090cb8d05fbd3d352f3c9572e546de2a1d3c3527608e2a8e849a657c391`.

Template ID `2e10975004e0351d07b237e0360a2b96b50d5884b651cfa5bfc8e88af62da266`; exact bundle digest `8693f675a4ec4be6d14a859157fbbf9d21b3608e4485d32b900421c0d7d7b5aa`; candidate program `ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67`; helper program `698256075f67b360cacb0b20e16c5aeafc561c76255aeb3fd1ca7481ef9e9a6b`.

B consumes the original request, invokes one singleton complement helper in every mode, preserves id/value and sets helper mode to answer, then awaits the exact original helper operation. It tests succeeded status and response presence before its mode action. Refusal yields failure without a candidate reply regardless of requested mode. The helper independently branches on input 1, assigns complement 0/1 to a private register, sends original id/complement once and succeeds. B's answer forwards these actual returned fields unchanged; non-answer/pending branches match K1. No default response register is treated as proof of presence.

This is meaningful structural and algorithmic variation from A: dynamic nested computation, original handle consumption and result forwarding versus local direct reply branches, with distinct branch direction and private value register. Assigned A-local/B-helper architectures are disclosed; the independent logic claim is not inflated into independent architecture discovery. Source contains only schema sharing and ordinary APIs, with no other candidate import. The no-cross-reading statement is the authors' and controller's attestation; hashes prove retained byte continuity, not authors' mental independence.

The28 retained samples cover all 16 requests plus invocation/run/depth helper refusals in each of the four modes. Sufficient-bound cases have exactly `(2,2,0)` charges and 3 total runs; refused helper cases have `(1,1,0)` and 2 total runs, no helper member/run, exact rejected helper reason and failed candidate envelope without reply. The denial subset transparently fixes id 1 / value 0; it is not misreported as the full finite payload product.

Independent JSON checks confirmed the original helper outcome is consumed in all 28 samples, with runtime-invocation provenance, source template and original handle; every candidate Send follows that consumption. Each helper input preserves the request id/value and answer mode. All present candidate responses preserve id and complement. No extra subtree admissions, external/adapter attempts, or effect charges appear. Pending leaves the helper finished but candidate/caller open. Actual maximum 19 steps / 6 dispatches is below fixed 200/100 caps.

The build-tagged B tests require `-tags integration`, unlike A's default tests. This is clear in the sources/reports and must remain explicit in later reproduction commands; default root tests alone do not qualify B. The author disclosed the unused-import setup failure and missing-feature RED separately from semantic results. First source freeze occurred after initial qualification but before cross-reading, with post-freeze qualification evidence matching that source; this meets the stated independence boundary.

## Ordinary contexts and shared protocol

`examples/substitution/protocol/protocol.go` contains only fresh request/response schemas and fixed port/peer/obligation constants. It has no response oracle or mode policy. Mode enumeration/bound is correct; integers are only typed, consistently with K1.

`examples/substitution/deployment.go` realizes the frozen five shapes without importing either candidate or changing runtime behavior:

- Bare/hidden: one static root, candidate invoked once, original outcome retained; success requires succeeded+present, otherwise root failure; pending waits at the original await.
- Sequential: commits S only after candidate succeeded+present, using the actual returned id/value without validating or repairing them; awaits S and succeeds only on its succeeded+present outcome.
- Parallel: static root and peer share the one resource root. Peer independently invokes S with original request id/value, awaits its own original operation, and uses its own outcome. There is no explicit cancellation policy tied to candidate non-success. Runtime root-scope closure rules remain relevant under alternate schedules; this builder does not prove non-answer peer completion for every schedule, and K1 gives no Gc entitlement to those modes.
- Wrapped: singleton wrapper receives the original request, invokes candidate, awaits its original outcome, sends actual response fields only for succeeded+present, and produces its own success/failure. It does not overwrite child's `missing_reply`/rejection envelope with a claim that the wrapper has the same reason.
- S: one Receive, one id-preserving response with constant value0, one Success. It does not recompute or impersonate the candidate complement.

The constructors retain all catalogue templates and set one explicit root with caller/peer membership. Fixed run accounting is 1 except parallel 2; wrapper/S/candidate/helper costs remain ordinary admission-created-run debits. No artificial effect or preprovisioned dynamic helper is used. Hidden shares the same execution as bare; actual hiding is an observation projection for the later checker, not a claimed new runtime primitive.

The context wiring tests exercise 10 shape/presence cases using a clearly labeled local reply/no-reply fixture. They check exact present response forwarding, root success/failure, independent concrete peer completion, exact invocation/run/effect charges, fixed run counts and replay. The absent fixture locally succeeds without sending, so it exercises the missing-reply boundary rather than only an explicit failure branch. These are appropriate wiring tests; they do not purport to cover pending/failure/refusal/closure across A/B or complete the 3,840-configuration assignment. Preserved first-attempt missing-Initial.Data failures were fixed by explicit empty input maps, with unchanged test bytes; the correction is narrow and no production behavior changed.

## Approval limits and next evidence boundary

No source changes or additional gate are requested for this slice. Proceed to the already assigned independent model/transcription and public-runtime witnesses. Retain first authored versions and record any subsequent integration edits rather than replacing their evidence.

This approval supports specification compliance by code inspection and the inspected44 author / 10 context runtime samples. It does not establish the separately checked local relation, nondeterministic reference-product guarantees, complete context census, artifact-to-model correspondence, scheduler confluence, contextual equivalence, recovery substitution, or a universal theorem. Those remain explicit later-stage obligations. The first-version REDs are authoring evidence; authentic wrong-id/extra-helper semantic negatives still belong to the planned independent-checker qualification.
