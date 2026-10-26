# Phase 1F public investigation contract

This is an application protocol over the existing authoring API and executor. The implementation must remain an external public consumer. Static application builders may expand finite control and wire participants; reducers, scheduler, store and adapters gain no role-specific policy.

## Boundary and bounded decisions

An investigation accepts one nonempty question through a statically wired request/reply port with an explicit obligation. Its public factory returns ordinary instances and this port. The same factory is used by a direct caller and by an additional ordinary report component. Neither caller inspects child state, SQL or internal instructions. All message and effect correlation retains the existing source/run/obligation guards.

The coordinator asks the model for two evidence subquestions, sends both before awaiting either, and consumes public results from two existing evidence helpers. It synthesizes a candidate from their evidence. A separate ordinary reviewer receives question, candidate and evidence, makes one schema-bounded model request, and returns accept, revise or stop with reason and proposed revision subquestions. Ordinary coordinator branches decide what follows. Model content is untrusted data; only finite enumerated choices with validated fields can advance the protocol.

There are at most two rounds: initial plus one explicitly authorized revision. The revision count and pending requests persist in ordinary state. A revision requires a received valid review requesting revision and remaining budget. A second revise request stops at the revision limit and never creates another round. Clarification, insufficient evidence, invalid result, exhausted budget and unresolved provider work are distinct from a supported final answer. Accepting an answer is the application review judgment, not independent proof of support.

The revision uses separately provisioned one-shot helpers and new effect identities. It may retain earlier evidence together with new evidence so that acquiring a missing fact does not erase supported facts. Final public output must include disposition, answer, initial candidate, review decision/reason, revision count and exact source citations or evidence references sufficient for independent grading. Citation provenance must be mechanically derived/checked against helper evidence; semantic claim entailment is independently evaluated. The exact exported schema and factory signature are documented with implementation.

## Budgets, uncertainty and stopping

Maximum opportunity is one planning request plus two rounds each containing four helper model requests, one synthesis and one review: 13 model requests, four lookups, one revision. Per-participant logical counters and immutable effect IDs persist across physical restart. These are explicit application ceilings, not universal runtime limits or hierarchy-wide business budgets. A lower configured revision limit must work; invalid limits fail at authoring/provisioning.

An adapter attempt with unknown outcome remains the same unresolved effect. Startup does not dispatch it again or turn it into a revision. This can prevent a final public reply; inspection must expose the pending/unknown identity. No eventual-completion claim is made. Unused future-round helpers may remain waiting; no cleanup, dynamic creation or fresh-generation reassociation is introduced.

## Qualification distinctions

Tests must demonstrate ordinary evidence-dependent decisions, partial sibling progress, successful standalone and nested public interaction, review/revision interruption and durable budget/operation correlation. Read-only replay has no registry or dispatch authority. Independent support/provenance/abstention grades and provider usage/latency are reported separately from durability. Controlled providers establish protocol behavior only; live results are labeled separately. Historical Phase 1E evidence remains immutable.

No new runtime semantic capability is assumed. Before any authoring extension, retain the smallest public workload obstacle and classify it. If the existing basis suffices, record that finding without adding convenience APIs merely to hide this application.

## Development amendment: explicit revision exclusions

Task2C preserves the interpreter and this public result contract. Initial helpers use the legacy question-only port; statically provisioned revision helpers use the optional public `evidence.NewExcluding` port. The ordinary coordinator copies e1/e2 answer citation identities into at most two exclusion tuples, uses all-zero tuples for non-answer slots, and sends the same set to both revision helpers only after an authorized bounded revision. No helper reads private state or infers exclusions. Original helper prompts and legacy compiled artifact IDs are retained. The new lookup profile filters exact supplied identities with unchanged lexical ranking and output schema; [retrieval-exclusion-contract.md](retrieval-exclusion-contract.md) defines validation, versioning and compatibility. Both revision siblings may return the same new span. No effect-count, revision-budget, unknown-outcome or replay rule changes.
