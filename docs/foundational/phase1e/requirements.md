# ALAN Phase 1E: Minimal Authoring API and the First Useful Agent

You are continuing implementation of ALAN
(Agentic Lambda Abstraction Network).

The project now has a single-host durable executor, ordinary supervision,
replay, process-crash qualification, and bounded asynchronous external
dispatch.

This phase must make that foundation usable by an application author and
demonstrate one real model-and-tool workflow running through it.

This prompt authorizes implementation, tests, a runnable example, and
independent review. Write a concise implementation plan, then execute it.
Do not return another research-only proposal.

## 1. Objective

A developer should be able to define a small agentic application in Go
without writing raw instruction arrays, numeric program counters, SQL, or
runtime lifecycle bookkeeping.

That application must:

- Make a real model request through the durable effect boundary.
- Branch on the recorded, validated response.
- Invoke an authorized tool through the same execution discipline.
- Use the tool result in subsequent model computation.
- Include a reusable composite behind an ordinary interaction contract.
- Resume after process death without repeating already-recorded interactions.
- Replay offline without contacting a model or rerunning tools.

The authoring layer must compile to the existing execution model.
It must not become a second runtime.

Success is a runnable application, not just builder types and mock tests.

## 2. Establish and preserve the baseline

Inspect the actual repository, project instructions, current branch, and
working tree. Verify that the qualified asynchronous-dispatch correction
is present before building on it.

Read the Phase 1D and 1D.1 implementation contracts, tests, runbooks, and
retained defects.

Reproduce the relevant baseline tests and preserve:
- Independent progress while external work remains in flight.
- Bounded dispatch and runner ownership.
- Atomic execution commits and retained operation outcomes.
- Accepted and rejected lifecycle recovery.
- Late-receipt correlation.
- Replay without dispatch.
- Known-but-unsettled versus unknown external outcomes.

Preserve historical qualification artifacts under their original source
identities. Report current-source regression separately.

Do not assume an old branch still exists or automatically merge unrelated
work.

## 3. First application: evidence-backed answers from a local corpus

Build a small evidence assistant over a bundled, nonsensitive technical
corpus, with an option to supply an explicitly selected local corpus.

The successful path is:

question
-> model proposes a lookup
-> ordinary program validates and selects the lookup branch
-> read-only lookup tool returns evidence
-> model produces an answer using that evidence
-> ordinary program validates the result and publishes it

Support a bounded alternative such as requesting clarification or reporting
insufficient evidence. Do not force every valid model response down one
hard-coded success path.

Use a small structured decision/result schema. A model-generated tool
request is a proposal, not permission to execute arbitrary code.

The lookup tool should return document identities, retained content versions,
and exact snippets or offsets. The final answer should identify the evidence
it used.

Check that cited document identities and quoted spans exist in the retained
tool evidence. Do not claim that this mechanical check proves every sentence
of the answer is substantively correct.

Keep retrieval simple. Do not add embeddings, a vector database, a crawler,
a search service, or an ingestion product.

Snapshot or retain the selected corpus bytes before execution. A path and
digest alone are not enough if the source later changes or disappears.

Replay must use recorded observations, not reread the current working copy.

### Demonstrate composition

Place at least one helper behind a reusable request/result interface and
implement that helper as an ordinary composition.

For example, a caller can use an evidence assistant whose internals include
planning, a retrieval worker, and synthesis.

Static expansion/provisioning is sufficient. Dynamic agent creation is not
required.

Use the same authoring and runtime interfaces for leaves and composites.
The caller must not inspect child state or rely on internal subject names.

Do not route the whole model-tool-model conversation through one opaque
adapter callback. Its decisions, requests, observations, and intermediate
states must remain visible in ordinary ALAN execution.

## 4. Minimal Go authoring surface

Create a small provisional importable authoring package.

Support the operations needed by the application:
- Named inputs, values, and outputs.
- Sequential composition.
- Conditional selection using bounded deterministic expressions.
- Named effect requests and correlated results.
- Ordinary request/reply interaction with another component.
- Explicit success, failure, and bounded repetition where needed.

Use readable labels and validation errors rather than exposing numeric
program counters to application authors.

Go builder functions may construct an artifact at authoring/provisioning
time. Persisted execution must not depend on captured Go closures,
goroutine stacks, ambient globals, or rebuilding the program from whatever
source happens to be installed after a restart.

Compile the definition into an immutable, versioned artifact with:
- Stable program identity.
- Explicit execution/control representation.
- Input/output and effect schema identities.
- Prompt/template content or retained immutable references.
- A mapping from execution positions back to author-facing names.

Identical definitions must produce identical artifacts under the same
compiler version. Map iteration or accidental allocation order must not
change program identity.

Persist the actual artifact required for recovery. Do not store only a hash
whose content is unavailable.

If the current value representation lacks structured data or field access,
add the smallest bounded generic representation necessary. Do not create
opcodes specific to this evidence assistant.

Validate missing references, invalid branches, duplicate names, incompatible
bindings, and unsupported effects before execution wherever possible.

Add an external-consumer smoke example that imports the authoring package
without importing internal packages or using test-only helpers.

Do not promise a stable general-purpose SDK yet.

## 5. Model and tool adapters

Introduce only the narrow effect-adapter interface needed for:
1. One model provider.
2. One read-only local lookup tool.

An adapter consumes a committed immutable request and returns a correlated
completion candidate. The runtime validates and persists the result before
ordinary computation can consume it.

Adapters do not:
- Select the next agent or workflow branch.
- Own a hidden model/tool loop.
- Commit agent state directly.
- Make supervision decisions.
- Inspect checker/oracle state.

Application callbacks are permitted as effect implementations, not as
unbounded hidden execution inside the deterministic reducer.

Integrate adapters with the existing bounded asynchronous dispatch owner.
Do not add unbounded goroutines or bypass the in-flight limit.

An outstanding model request must not stop unrelated runnable agents.
A completed tool response must not wait for an unrelated slow model response
merely because it was dispatched later.

Keep network authority in deployment configuration and runtime grants.
Do not replace the existing fixture's loopback restriction with unrestricted
model-selected URLs, headers, or endpoints.

## 6. One real model provider

Implement direct OpenAI Responses API support.

Use current official documentation to verify request shapes, supported
structured output or function-calling features, and error handling:

https://developers.openai.com/api/docs/guides/function-calling
https://developers.openai.com/api/docs/guides/structured-outputs
https://developers.openai.com/api/reference/go

Use a thin net/http implementation or the official Go SDK, whichever gives
the smaller auditable implementation. Pin any new dependency.

Do not use an Agents SDK or another orchestration framework.

Require an explicitly configured model ID. Record the requested model and
the returned model identifier where available. Do not invent a model name
or imply that a mutable alias pins underlying model weights.

Non-streaming requests are sufficient. Streaming, background responses,
hosted tools, and remote conversation management are not required.

Retain the bounded replay-relevant request and response, including:
- Instructions and prompt inputs.
- Model and generation parameters.
- Tool/schema definitions when used.
- Provider response/request identifiers where returned.
- Returned content and tool-call correlation.
- Usage information where available.
- Validation outcome and error classification.

Credentials and authorization headers must not enter program artifacts,
journals, inspection output, test evidence, or source archives. Resolve
credentials from runtime configuration.

Do not rely on remote conversation state as the only record of what the
model saw. ALAN must retain the input/output content needed for local replay.

### Retry and uncertainty

Default SDK/HTTP retries must be disabled or made explicit, bounded, and
observable through ALAN's attempt accounting.

A model call may be accepted remotely before its response is recorded locally.
Do not assume that a client operation key or a low temperature makes the call
idempotent.

Distinguish:
- A durably recorded response, which recovery reuses.
- A received response not yet persisted.
- A known provider rejection.
- An incomplete or schema-invalid model response.
- A dispatched request with an unknown outcome.

For unknown model outcomes, retain uncertainty and expose an explicit recovery
disposition. Do not silently issue another potentially billable request and
call it replay.

If repeat execution is supported, it must be explicitly authorized and
counted as another attempt, with possible duplicate provider work disclosed.
A general provider reconciliation system is outside scope.

Schema-invalid, refused, or incomplete outputs must not become success.
Any repair request is another ordinary bounded decision and model effect,
not a hidden retry loop inside the adapter.

## 7. Cross-store external operation identities

Correct the disclosed cross-store operation-key collision before connecting
independent runtime stores to shared external services.

Assign each newly initialized store a durable collision-resistant namespace
before it can emit external work.

Derive external operation identities from that namespace plus the existing
logical execution/intent identity using an unambiguous canonical encoding.

Preserve these properties:
- Reopening or renaming one store preserves its namespace and operation keys.
- Two independently initialized stores cannot share keys merely because
  their subjects, generations, sequences, and ordinals match.
- Retransmission of one operation preserves its identity and parameters.
- Physical worker changes do not produce a new logical operation.
- A copied database is not silently treated as an independent new store.

Document copying/forking semantics. Concurrent independent use of copies
may remain unsupported; do not pretend a copied namespace is newly unique.

Make the schema/key-version change explicit. Either support retained stores
without changing their committed keys or reject unsupported stores for live
execution before dispatch. Do not silently re-key historical work.

No general migration framework is required.

Test two stores with identical logical programs against one shared durable
fixture endpoint, and test reopen of each original store separately.

This namespace separates runtime operations. It does not provide domain-level
deduplication of the same business task across fresh executions.

## 8. Budgets and validation

Give the example explicit persisted logical budgets.

Use conservative defaults, such as three model requests and two lookup
requests per question, with configurable bounded output sizes. Record the
actual selected limits.

Physical process restart must not reset the logical budget.

Distinguish logical requests from transport attempts. A retransmission,
an intentional repeat model call, and a new application decision must not
be conflated in usage reporting.

Validate tool names and arguments against the program's declared allowed
tools and schemas before dispatch. Model output cannot grant itself broader
rights or introduce another provider endpoint.

Retain invalid model/tool outcomes as inspectable evidence with an explicit
program disposition.

Report actual returned usage. Unknown usage or cost remains unknown; do not
manufacture billing precision or assume the configured limits account for
unobserved remote work.

## 9. Required acceptance tests

### A. Authoring and compilation

An external Go consumer defines the application without raw instruction arrays,
SQL, or internal imports.

Compilation is deterministic, errors are useful, and the retained artifact
is sufficient for recovery without recompiling newer application source.

### B. Ordinary branching and nesting

A controlled provider returns different valid decisions.

Demonstrate lookup, clarification/insufficient-evidence, and invalid-output
paths through ordinary program branches.

A nested helper is used through its declared interface without runtime
special cases or caller access to its private state.

### C. Recorded model response survives process death

Persist the first model response, then SIGKILL the executor before its
ordinary consumption or subsequent tool intent commits.

Reopen the same store. Continue from the retained response without another
provider request for that completed operation.

Verify provider contacts with the controlled endpoint's own records, not
only the runtime's report.

### D. Recorded tool result survives process death

Persist the lookup result, kill the executor before synthesis proceeds, and
reopen.

Use the recorded result. Do not rerun the lookup or read changed source files
to reconstruct history.

### E. Unknown provider outcome

Have the controlled provider accept a request and withhold its result.
Kill the executor before any response is durably recorded.

Recovery must retain an unknown outcome rather than invent completion or
silently treat repeated inference as the original historical result.

Separately test the explicitly selected repeat policy, if implemented.

### F. Offline replay

Finish a successful model/tool run, then remove credentials, make the provider
unavailable, and make the original corpus path unavailable.

Replay must reconstruct the committed computation and final result with zero
model/tool invocations.

A new live run need not produce identical model output. Replaying one recorded
run must preserve its recorded observations and decisions.

### G. Namespace and authority

Two stores with identical logical IDs produce different external keys.
Reopening each store preserves its original key.

Wrong tool names, conflicting parameters under one identity, mismatched
results, and model-supplied endpoint changes are rejected.

### H. Independent progress and cancellation

Hold a model request open while unrelated ordinary work advances through the
automatic runner. Preserve the Phase 1D.1 causal ordering test.

Cancellation and closure retain valid late evidence without reviving closed
programs. Saturation must not block local progress.

### I. No hidden provider policy

Demonstrate that the provider adapter cannot execute a tool merely because
the model requested it.

The model result must first be recorded, consumed by ordinary computation,
validated, and translated into a separately committed authorized tool intent.

Keep at least one negative control proving that the tests detect bypass of
this boundary or replay-triggered provider contact.

Use explicit process/failpoint acknowledgments for crash tests. Watchdogs
fail stuck tests; sleeps do not establish causal ordering.

## 10. Live and deterministic qualification

Default automated tests use controlled HTTP provider responses and the real
adapter serialization/parsing path. Do not replace the provider adapter
itself with a mock and call that protocol qualification.

Provide an opt-in live command using the configured OpenAI model and
credentials, with the bundled nonsensitive corpus and explicit small call
and output budgets.

When authorized credentials are available, run that bounded live example
and retain:
- Exact source/program/provider configuration identity.
- Request/response identifiers where available.
- Actual tool evidence.
- Final answer.
- Usage and request counts.
- Offline replay of the same recorded execution.

Do not make default tests require credentials or spend money.

If live credentials or access are unavailable, complete implementation and
controlled qualification, provide the runnable live command, and label the
live gate NOT RUN. Do not fabricate a live result or block unrelated work.

Do not gate live success on exact model wording. Check protocol behavior,
valid evidence references, declared bounds, and explicit final disposition.

## 11. Keep the increment focused

Retain:
- Go.
- Single host and one active executor per store.
- SQLite durability.
- Bounded asynchronous dispatch.
- Static subjects and grants.
- Ordinary control and supervision.

Do not add:
- Multiple model-provider implementations.
- A generic agent SDK ecosystem.
- Dynamic agent creation.
- Multi-host scheduling or consensus.
- Ownership transfer or cleanup custodians.
- Hierarchy-wide business budgets.
- A vector database or full retrieval platform.
- A UI or cloud deployment.
- Streaming unless necessary for a demonstrated blocking requirement.

Do not reopen the foundational literature investigation.

If a generic capability is missing, add only what the application needs,
document its semantics, and test it. Do not add application-specific runtime
instructions to make the example appear compositional.

## 12. Delivery and review

Implement in independently testable slices:
1. Versioned store namespace and external-key isolation.
2. Authoring/compiler surface and external-consumer example.
3. Narrow adapter boundary, OpenAI adapter, and read-only lookup.
4. Ordinary nested application and CLI execution.
5. Crash/replay tests, live probe, and independent review.

Use a delegated implementer where available and an independent adversarial
reviewer. Give the reviewer complete source, contracts, fixtures, and evidence.

Preserve defects and their original inputs. Do not weaken a contract merely
because the first implementation cannot satisfy it.

Run the complete test suite, focused tests, race checks including subprocesses,
vet, formatting, dependency verification, and previous research/runtime
regressions.

Deliver:
- Working importable authoring package.
- Runnable Go application and documented CLI commands.
- One real model adapter and one read-only tool.
- Durable compiled artifacts and replayable results.
- Cross-store identity correction.
- Controlled crash/recovery qualification.
- Clearly separated live-provider results or NOT RUN status.
- Concise authoring/effect/recovery contract.
- Machine-readable evidence and source/dependency manifest.
- Complete reproducible source package without credentials.

Start the final report with the application author's experience and the actual
application result, then show recovery/replay evidence and remaining limits.
Do not lead with a large test count as a substitute for a useful application.

Publish on a verified appropriate branch and open/update a draft PR.
Do not merge or release automatically.

## Success criterion

A developer can author an ordinary composed ALAN application that uses a real
model and a tool, obtains an evidence-backed answer, survives process death at
recorded boundaries, and replays offline without repeating historical work.

The provider executes model requests.
The tool executes authorized lookup requests.
The ordinary program decides what happens next.
ALAN preserves and enforces that execution.