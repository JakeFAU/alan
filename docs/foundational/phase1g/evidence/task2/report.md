# Phase 1G Task 2 checkpoint — design mapping

Status: implementation not started; no source edits or tests run. Base 65fd931, branch codex/phase1g-data-composition. Contract R1–R6 and integration notes read. Task1 APIs/version report read. No provider/credentials/dependencies used.

## Required implementation spine

Existing histories order only within a run. Schema4 needs an ordered composition decision/fact ledger written in actual existing transactions. It must link every relevant scoped execution commit, lifecycle closure, invocation admission, reply admission, attempt reservation, and outcome publication. Failure freeze facts cannot be reconstructed from final projections alone. Schema3 SQL and IDs must remain unchanged.

Immutable binding inputs must retain original program IDs and map only compiler-declared port endpoint operands. Live and replay must apply one pure binding interpretation; no patched bytes/hash alias. Dynamic generation1 origins need both-direction validation and cannot pass as provisioned roots or Retry successors.

Public API correction from controller: AwaitInvocation needs ordinary reason binding in addition to status and response-presence. Planned choice is explicit InvocationOutcomeBindings{Status, Reason, ResponsePresent string}; response payload bindings remain map[string]string. No hidden reason payload/state.

## Existing integration points

- author/compile.go: invocation declarations, typed Invoke handle and AwaitInvocation lowering/assignment; retain absent metadata omission and scalar hashes.
- author/application.go: retained closed template validation and explicit provisioning additions.
- internal/executor/store.go: schema selection, creation validation, immutable bindings, execution commit facts.
- internal/executor/messages.go: source authorization reply exception limited to designated committed public route; separate candidate admission.
- internal/executor/lifecycle.go and lifecycle_replay.go: invocation origins and scoped Retry denial; lifecycle closure facts in target transaction.
- internal/executor/adapters.go and external.go: root reservation/debit after source authorization in durable attempt transaction; preserve prior allowed reservations and known-negative denial.
- internal/executor/runner.go: invocation delivery on existing owner; per-root physical counts and skip saturated root while scanning eligible external work.
- internal/executor/replay.go, inspect.go: ordered prefix verification, origin/permission/charge reconstruction and keyless read-only report.

## Concrete replay counterexample

Required members A and B have histories A: Send(response1), Send(response2); B: Failure. Schedule X is A.Send1, B.Failure, A.Send2; schedule Y is A.Send1, A.Send2, B.Failure. The final per-member histories/projections/intents are identical. Contract R2 freezes only response1 in X, both responses in Y. Candidate delivery can occur after either schedule. Therefore checking the final rows or recording the failure only when a delivery callback notices it cannot verify the required freeze. The failure fact and prefix freeze must be recorded in B.Failure's own commit transaction, with ordered facts for the A commits. This is a storage/replay requirement, not a need for a second scheduler or workflow-wide transaction.

## Validation pending

All Task2 regression obligations remain pending. No RED/GREEN evidence yet. No Task2 public API or table name is finalized or available to Task3.

## Handoff boundary

This worker is returning the expressly permitted coupling boundary rather than shipping a partial semantic implementation. Task2 is NOT complete. No public APIs/tables/source changes have been made and no Go tests have run. There is no Task2 implementation commit.

A coherent next implementation milestone is a feature-gated schema4 authority spine: immutable deployment catalogue/resource roots, binding and dynamic origin structures, version selection, pure binding interpretation, and ordered scoped facts attached to their actual existing transaction boundaries. Its first RED tests should establish missing/forged dynamic origin refusal, unchanged scalar artifact identities, schema3 refusal to acquire composition state, and the two-prefix freeze witness above. Do not expose successful template admission while required-member completion, late reply authentication, scoped closure and replay validation remain missing.

Next sequential milestones within Task2 are (1) committed Invoke admission/rejection identity and the complete R2 state machine, including the committed-but-unadmitted Invoke join; (2) root attempt reservations, scoped lifecycle and per-root physical scheduling integrated in existing transaction/owner paths; (3) both-direction ledger reconstruction, corruption/reopen and full regression qualification. These are execution boundaries within the approved design, not a proposed architecture change or permission request. Task3 must continue waiting for complete Task2 APIs and tests.

Exact inspected HEAD: 65fd931b483b7990ca3c0154e8e0e5fd02ff72fc. At handoff git status also showed controller-owned modifications to docs/foundational/phase1g/phase1g-plan.md and untracked docs/foundational/phase1g/00-plan.md; this worker did not edit them. This checkpoint and its tracked evidence copy are the worker's only writes.
