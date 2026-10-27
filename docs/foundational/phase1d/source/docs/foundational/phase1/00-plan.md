# Phase 1 executable semantics plan

> Execution uses delegated implementation and independent adversarial review. This plan authorizes a research model only.

**Goal:** falsify the candidate execution and lifecycle contracts under bounded adversarial interleavings.

**Architecture:** a deterministic transition system in a Go test-only package, with an explicit finite instruction language. Worker proposals, committed journals, transport evidence, and external resource state are separate. Scenario-specific searches enumerate event orderings and check independent invariants after each transition.

**Toolchain:** repository Go 1.27.3 and standard library only. No dependency installation, network, database, model provider, clock, goroutine scheduler, or infrastructure.

**Baseline:** the five documents in the parent directory are provisional. At inspection the repository was on `main` at `c41961040f4392b2d615306f7a24cb211ecb2ec7`, with those five documents untracked, no Go source or tests, and the requested Go version installed locally. Preserve that baseline and its previous review history.

## Frozen scope

The operational specification in [01-operational-spec.md](01-operational-spec.md) defines the modeled choices. In particular, permanent cancellation is named **subject retirement**. Run-only cancellation is separate and is exercised with both explicit automatic-retry policies. One choice is not silently substituted for the other.

Adopt D1, D2, D4–D9, D12, and D13 within finite domains. D3 is represented by a closed instruction language, not arbitrary callbacks. D10 is the subject-retirement profile plus an explicitly separate run-cancellation alternative. D11 is limited to ordinary static Control composition; group cancellation and surviving custodians are excluded. Dynamic creation, ownership transfer, distributed atomicity, capability delegation/revocation, schema migration, deduplication expiry, and unbounded fairness are excluded. Ownership revision is still checked independently of activation epoch.

All atomicity is assumed by the model's transition rules. None of the tests establishes distributed serialization, storage durability, exactly-once external execution, or production readiness. External application and receipt recording must never be one atomic helper.

## Tasks and ownership

1. **Freeze semantics and assertions — primary agent.**
   - [x] Inspect the repository, all five documents, and installed toolchain.
   - [x] Publish this plan before implementation.
   - [x] Specify state, bounded instructions, transition atomicity, cancellation alternatives, invariants, and excluded behavior.
2. **Implement one tightly coupled model/checker task — delegated implementer.**
   - [x] Write observable-contract tests first and retain the first failing result.
   - [x] Create only `research/semantics/*_test.go` and a short research README. Split model, interpreter, explorer, and qualification fixtures by responsibility; no importable runtime package.
   - [x] Implement prepare/interrupt/commit, messaging and stable binding, separate attempt/application/receipt events, lifecycle rules, and fixed-version replay.
   - [x] Add bounded exhaustive scenario exploration, explicit progress witnesses, ordinary static Control composition, and at least three guard-removal negative controls.
   - [x] Return reproducible commands, exact search statistics, counterexample traces, and any semantic deviations. Do not alter the specification silently.
3. **Qualify, challenge, and finish — primary agent plus independent reviewer.**
   - [x] Inspect implementation and independently rerun tests, race checks, formatting, and `go vet`.
   - [x] Write `02-qualification.md` with exact bounds, counts, reachable witnesses, counterexamples, and limitations.
   - [x] Append Phase 1 evidence to `../05-decisions.md`, preserving open alternatives and earlier disagreements.
   - [x] Obtain independent adversarial review of code, assertions, exploration, and report; preserve rejection findings and correction/re-review history.
   - [x] Stop with the reviewed research deliverables. No promotion into production packages or implementation of the runtime.

## Preflight and review focus

| Boundary | Potential conflict | Resolution |
| --- | --- | --- |
| Specification and model | Callback power could conceal unbounded computation | Exactly one closed-language instruction per local turn |
| Model and oracle | State merging or guards could make invariant checks vacuous | Full future-relevant state identity, separate monitors, negative controls, reachable witnesses |
| Retirement and cancellation profiles | A run cancellation could accidentally stop every future run | Separate operations and explicit retry-after-cancel policy variants |
| Journal and endpoint | Atomic effect helpers could erase ambiguous completion | Separate authorization, application, and receipt transitions |
| Model and qualification report | A depth or state cap could be reported as exhaustive proof | Report depth cutoffs, event alphabets, states, transitions and state-cap failures |
| Task 1 and task 2 | Both describe the same semantics | Implementer consumes this spec; deviations require a recorded scope amendment |
| Task 2 and task 3 | Report depends on actual model output | Use freshly executed results, not projected counts |

The model and its invariant checks share tightly coupled state definitions, so one implementer owns them. The primary agent owns documentation and verification; the final reviewer authors neither. Work remains in this checkout on a research branch so the existing untracked foundational baseline is preserved, with model isolation enforced by the test-only directory.
