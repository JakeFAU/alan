# Phase 1I Stage 1 contract corrections

Date: 2026-10-28. Documentation-only scoped implementation of review R1/R2/R3.
Working tree: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.
Independent scoped rereview remains required; this report does not approve its own changes.

## Changed files and disposition

- `docs/foundational/phase1i/01-substitution-contract.md`: revision1 for rereview.
  - R1: sections at lines71,151,200,252 freeze finite state/event rules; asymmetric local simulation plus explicit environmental matching and maximal-path/cycle obligations; separate nondeterministic reference `C[K1]` checks; K-derived initial/prefix ticket demand that cannot disappear because a faulty candidate stops or overspends; and reviewed artifact-instruction/event correspondence without compiler/reducer imports or a compiler-proof claim.
  - R2: lines31–69 distinguish local terminal status from invocation status/reason/presence and the wrapper's own envelope. `absent` yields local succeeded but failed/missing_reply publication, then wrapper failed/member_failed. Pending and entry/helper rejection are explicit. Default-bound values do not count as a present response.
  - R3: lines119–149 freeze every caller shape, S's fixed protocol, non-success handling, exact forwarding and pending behavior. Parallel records both branches without cancelling its peer. Wrapping has its own Send/terminal behavior.
  - Base census remains3840 per reference/candidate graph; fixed completion-entitled answer subset is84: bare32, hidden32, sequential8, parallel8, wrapped4. Formula uses initial K/context resource obligations, not candidate outcomes. Separate indexed probe/effect/closure/reopen families are fixed at lines306–344.
  - Actual schema constraint recorded: integer domains are semantic/environment restrictions because scalar Field has no Min/Max. Mode is String/MaxBytes8/Enum of four modes. Assigned local A/helper B architecture is distinguished from independently authored implementation logic; no cross-reading before first hashes.
  - Required negative controls are wrong application id and actually admitted extra subtree charge under sufficient runtime capacity. An unknown-to-success attempt blocked by the runtime join is not a claimed semantic negative detection.
- `docs/foundational/phase1i/00-plan.md`: Task1 keeps authors gated until scoped rereview; Task2 separately requires reference model, local relation, independent artifact transcription and products; Task3 negatives and correspondence limitations align with the contract.
- `docs/foundational/phase1i/05-decisions.md`: records R1–R3, retained original REJECT/draft, revised semantics,84-case entitlement partition, independence scope and evidence limits.

## Digests

SHA256 of revised contract:
`cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`

Other edited files:
- plan: `09edc6a88c7c7fc5b002938854b4b960cdc87d399a9a9dafcb377fa72988e7cb`
- decisions: `920abdf3736d393afdc233d554515d38ac5d29a2fb3ff2c790d4f52ca32db595`

Original contract-v0 preserved, SHA256:
`9876974ef1a4285a989117196702db54426f22f7b391b6b9124a6ca6bf562698`

## Verification and exact limits

Read the full independent contract review and API audit; inspected author public operations, invocation/outcome code including `prepareCompositionScope`, reply admission and successful-publication joins. The relevant outcome guards support `missing_reply` and pending external-join wording. Inspected existing contract/plan/decisions and original-draft presence.

Executed read-only source/document inspection, SHA256 calculations, arithmetic (`5*2*2*4*4*4*3=3840`, `32+32+8+8+4=84`), scans for obsolete draft language, and per-file whitespace checks (zero trailing-whitespace lines). `git diff --check` exited0 but these phase files are untracked, so it is not claimed to validate their contents; the explicit per-file checks do. No candidate/model implementation, tests, compiler proof, runtime experiment, paid calls, commits or production changes were performed.

Default file copy was blocked by filesystem permissions. The scoped elevated write of these three authorized documentation files succeeded. No permission blocker remains.

Open gate: independent scoped rereview of the revised rule table, local relation, entitlement and complete context tables; only after acceptance should shared author/model briefs freeze this digest and authors begin. Broader theorem/interference/recovery claims remain explicitly open.
