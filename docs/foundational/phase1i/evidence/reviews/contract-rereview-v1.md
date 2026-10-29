# Phase 1I Stage 1 scoped contract/plan rereview

Decision: **APPROVE** the revised specification and execution plan for Stage 2 dispatch.

Date: 2026-10-28. Scope: original R1/R2/R3 and inconsistencies introduced by their fixes. Reviewed `/tmp/alan-phase1i-contract-fix.md`, `/tmp/alan-phase1i-contract-v1.diff`, and current phase contract, plan and decisions. Rechecked relevant invocation admission, reply and outcome source while assessing the new rule table. No repository edits, tests, model executions, subagents, paid calls or new proof/gate requirements.

Approved contract SHA256, independently recomputed:
`cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`

This approves specification/execution-plan quality. It does not establish conformance, substitution, correspondence, runtime qualification, or a theorem.

## Per-finding disposition

### R1 — Resolved

The revised contract freezes finite protocol state, event guards, effects and an independent local environment. It defines asymmetric matching, matching environmental interference, explicit progress obligations and rejection of internal stuttering cycles. It separately requires local A/B refinement, nondeterministic reference C[K1] checks, A/B context checks, and runtime trace correspondence. The plan now gates authors on this reviewed contract and has the model author construct K1/reference callers before transcribing frozen artifacts.

Ticket accounting now distinguishes future demand from irreversible spent charges. K1/context determines initial entitlement, and candidate over-spending, premature termination or nonresponse cannot retroactively remove it. Acceptance transfers rather than refunds demand; legitimate branch exclusions remove only unspent future demand. The stated 84 initially completion-entitled configurations follow from the displayed arithmetic: 32 bare + 32 hidden + 8 sequential + 8 parallel + 4 wrapped, within the unchanged 3,840 configurations. Insufficient-budget cases remain safety inputs.

The artifact correspondence boundary is explicit: independent graph transcription and reviewed instruction/branch/handle coverage, with sampled runtime membership, rather than an exhaustive concrete-runtime or compiler-proof claim. This answers the original finding without requiring a general compositional proof at this gate.

### R2 — Resolved

The outcome table distinguishes local projection status, child invocation envelope, response presence, and wrapper envelope. In particular, absent is local succeeded but child failed/missing_reply, and the wrapper produces its own failed/member_failed outcome. Pending has no child terminal publication and leaves the wrapper awaiting the original handle. Admission/helper refusal and no-closure assumptions are stated separately. Default-bound fields cannot establish response presence. These corrections agree with the inspected pure invocation publication rules.

### R3 — Resolved

S now has a fixed request/reply/terminal protocol. All five callers specify successful, terminal non-success/absent and pending behavior. Sequential work starts S only after succeeded+present; parallel retains both branch outcomes and does not cancel its peer merely because the caller fails; wrapped forwarding uses its own reply and terminal status. The outer retained-data observation is distinguished from invocation results and root publication. These definitions supply concrete behavior for all enumerated modes without silently changing the census.

## New-breakage assessment and retained limits

No new gate-blocking inconsistency was found in the scoped corrections. Assigned local-A/helper-B architecture is transparently distinguished from independent implementation logic; first-version no-cross-reading/hash requirements remain intact. Per-trace identity correspondence permits different private-name cardinalities while preserving public anchors and causal links. Final effect reservation is correctly distinguished from earlier source authorization, and the optional runtime-suppressed unknown-to-success attempt cannot be counted as successful semantic negative detection.

The named closure/effect/reopen families are separately indexed rather than counted as part of the pure 3,840 configurations. Closure has safety scope only; exact runtime reasons remain retained through the declared abstraction. Operational limits must still be justified from frozen artifacts, and incomplete searches/runs cannot become passing evidence. These are existing execution obligations, not additional requirements from this rereview.

The original REJECT and draft should remain preserved as planned. Freeze this approved digest in the candidate/model briefs and proceed with the existing staged implementation and qualification plan. General contextual equivalence, scheduler confluence, unrestricted recovery congruence and compiler correctness remain outside this approval.
