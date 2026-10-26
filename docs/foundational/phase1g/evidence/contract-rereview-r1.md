# Phase 1G scoped contract rereview — revision 1

**Verdict: NEEDS_REVISION for R2 only. R1 and R3–R6 are addressed at contract level.**

This is the requested scoped rereview of the original six findings and conflicts introduced by their refinements. The original `contract-review.md` verdict and counterexamples remain unchanged. Reviewed revision-1 contract SHA-256: `e1f6685d33585a136616cb4b48240d051e27fa482c5c12d5136bcc8c7530b0af`. Original review remains SHA-256 `10e5cc8985b3ebe5b5344d958d3488b64b0fca17dc690decf8c2759987570699`; retained original draft is `d9fe4a23b8fe40ccedc877f0382994db55766dc42e8b9beee348aeb603a4e8da`.

Read the amended contract and execution outline. No implementation, tests, agents, live requests, credentials or commits were used. These are specification schedules, not observed runtime failures. This review is not proof and does not supersede later implementation/qualification review.

## Finding disposition

| Finding | Disposition | Basis |
| --- | --- | --- |
| R1 — root accounting/lifecycle | Addressed | Deployment-retained root identity; one obligation per root; descendants inherit; explicit static versus created-run accounting; atomic all-member admission/debit; effect debit at allowed attempt reservation; persistent denials/unknown charges; scoped/root Retry rejected. |
| R2 — invocation result state machine | Partially addressed; two remaining blockers below | Required members, status/response-presence envelope, committed reply notification, failure-trigger freeze and ordinary collector aggregation are defined. Failure-before-reply and internal-member failure are now coherent. Successful publication still omits two unresolved committed obligations. |
| R3 — closure/authorization race | Addressed | Scoped versus sibling closure, attempt-reservation cutoff, ancestor checks in transaction, historical source authorization preservation, post-closure launch of an already reserved attempt, late evidence and physical cancellation distinctions are explicit. |
| R4 — permissions/binding | Addressed | Closed deployment catalogue/allowlist, declaration plus root permission, capability checks, canonical invocation/member IDs and separately versioned binding environment eliminate the former undefined authority/hash transformation. |
| R5 — physical capacity | Addressed | Per-root plus global launch ceiling, no waiting-parent slots, capacity-1 local progress versus capacity-2 independent external completion, remote-unknown limitation and parent-slot negative control are explicit. |
| R6 — collections/versioning | Addressed | Snapshot/cursor/body commits, immutable copying, bounds/equality, zero-iteration initialization, versioned artifacts/bindings/schema paths and bidirectional replay checks are specified. Concrete public obstacle tests remain implementation-plan deliverables. |

R1 implementation must attach every participating initially provisioned helper to its declared resource root when it can spend that root's work; reporting it as fixed creation cost does not exempt its effects or lifecycle from root rules. This follows the amended shared-root contract and is an implementation obligation, not permission to create helper-local allowances.

## Remaining R2 blockers

### R2.1 — successful terminal must wait for committed reply notification admission

Revision 1 explicitly freezes committed public reply IDs on a non-success terminal, drains those notifications, and only then publishes the result. That resolves the evidence helper's Send(`invalid_result`) then Failure case. The success rule instead says exactly one valid response is required and a terminal-success scope without a response fails. It does not distinguish absence of any committed reply from a committed reply whose delivery/admission has not finished.

**Counterexample:** endpoint commits reply intent M; all required members subsequently commit success; M's delivery is paused. Scope publication sees no recorded public response and publishes protocol failure. In the alternative schedule, M is recorded before the last success commit and publication succeeds. Member computation and reply intent are identical; only notification delivery order changed the logical outcome.

**Required refinement:** at the all-members-success boundary, retain or derive the immutable committed public reply-intent set. No committed reply means missing-response failure. One committed reply with unresolved admission keeps publication pending; resolve it through the normal authenticated notification path before publishing success or a specific invalid/rejected-reply failure. Multiple distinct committed reply intents are protocol failure independently of delivery order. A notification's retried delivery must reuse its retained disposition. This is the success-side equivalent of the already specified failure freeze.

### R2.2 — join must include committed Invoke operations awaiting admission

The success guard requires known results for admitted descendants and committed external obligations. A committed Invoke intent that has not reached its admission transaction belongs to neither category. Source intent, source authorization and admission are deliberately distinct durable boundaries, so this interval is real and must be included.

**Counterexample:** P commits Invoke I and its own public reply, then reaches local success while I's admission is paused. It has no admitted descendant yet, all members have succeeded, and the reply is valid. Current success predicates allow P to publish success. I subsequently reaches admission and is rejected because P/scope closed, or (under another ordering) is admitted before P's terminal and creates required child work. Publication must not leap over the committed operation merely because child origins do not yet exist.

**Required refinement:** successful publication requires every committed Invoke operation from the scope to have a retained admission outcome. An accepted operation joins its descendant scope; a rejected operation is known correlated evidence that ordinary policy may handle. An unresolved admission keeps publication pending. Preserve source-closure rules: this does not require admitting children from a closed source, only recording the correct rejection before declaring the join resolved. Apply the rule transitively so a known failed child outcome cannot conceal unresolved descendants/effects when a higher ancestor attempts success.

These are two additional predicates within the chosen structured join; no application-specific oracle, scheduler or new role is needed.

## Local terminal versus public scope status

The parent's clarification is coherent with the amendment: an execution's scalar `succeeded` projection is a local terminal fact, while scope publication remains pending until the generic join succeeds. Existing descendants can continue under the declared closure policy. Inspection and public outcomes must expose the distinction; neither CLI nor collector may present a root's scalar status alone as successful composition. The contract already says root publication observes the join rule. No inherent conflict was found in keeping these two states separate.

Non-success publication may happen before descendants finish, as explicitly chosen. Such descendants retain their records/uncertainty and cannot authorize new scope work after closure. Successful ancestors still require the transitive absence of unresolved required obligations. A response payload accompanying a failed scope remains diagnostic/application evidence, not runtime success.

## Limits and next gate

No new blocker was identified in the other refinements. The version matrix is now a decision, not an implementation choice; source-matched compatibility tests must substantiate it. Capacity-one saturation claims are now feasible, including actual leaf completion after release and the faulty parent-slot control.

Resolve R2.1/R2.2 in the normative success rule, retaining this report and original schedules. Then a narrow rereview can determine contract readiness. Implementation correctness, acceptance-case evidence, performance, public authoring ergonomics and independent replay validation remain unestablished and belong to the planned later gates.
