# Investigation authoring observations

The initial bounded workload fit the existing public authoring basis. Subsequent Task2C development evidence exposed a narrower public data-contract limitation: legacy lookup/helper requests could not carry already-received spans to exclude. Task2C adds optional public lookup/helper input schemas and an optional helper factory, described below. It does not change the runtime, scheduler, reducer, store, schema field limit, operation versions or public authoring primitives. `examples/investigation` imports only public packages and the existing public evidence example. The CLI alone uses the pre-existing `internal/boundary.FromEnvironment` observational qualification hook, exactly as `alan-evidence` does; it adds no program policy or new boundary mechanism.

## Flat evidence and retained candidates

Smallest workload: synthesize from two helper records, retain those records, then synthesize from four helper records after one revision. `schema.Record` has scalar fields, not variable collections. The selected solution is finite author-time expansion into `e1_` through `e4_` fields. Each slot has the existing evidence helper's seven fields. The result has 37 fields, and the outer report has 38, below the unchanged 128-field limit. Non-answer slots retain their explicit helper disposition; never-used slots are `unused` with typed empty fields. No JSON-in-string collection or new schema type is needed.

Candidate citation provenance is not delegated to a model: the coordinator returns the validated helper slots themselves. Semantic support and relevance remain independent grading questions. An initial candidate uses e1/e2; a revised candidate may use all four slots without discarding earlier facts.

## Model input formatting

Smallest workload: transmit a question plus two evidence records and later a prior candidate/review reason to a model. There is no authored string interpolation operation. The existing OpenAI adapter already serializes ordinary scalar records into canonical JSON model input. The application supplies declared fields plus a retained `instructions` string. No formatting opcode or adapter policy was added.

## Finite revision and separate participants

Smallest workload: receive `revise`, check a persisted limit, send two new requests, then reject another `revise`. The coordinator uses forward branches over two author-time-expanded rounds. Four separately provisioned one-shot evidence helpers provide fresh effect identities. The ordinary reviewer has two finite receive/request/reply sections, with distinct pending identity variables. Both helper messages are authored before either receive. A public test leaves the first helper unknown while the second finishes, showing independent progress.

The fixed participant ceilings are coordinator 3 model requests, reviewer 2, and each of four helpers 2 model/1 lookup. `MaxRevisions=0` prevents the second pair from receiving work. Ordinary request updates persist spent counters atomically with effect intent; no hierarchy-wide runtime budget was introduced.

## Definite assignment at a budget exit

Smallest input: an authored reviewer branches to its reply label on exhausted budget before awaiting a model response, while the reply contains model-selected subquestions. The first compiling test failed with `steps/8:send:coordinator: value "subquestion1" is not definitely available`. This is a useful authoring check, not a missing runtime capability. The selected solution initializes all public reply fields to typed defaults, then assigns the explicit budget/invalid decision and reason on those paths. No compiler relaxation was required.

## Public composition with useful outer policy

The outer report receives the unchanged investigation result through its port. Ordinary `Set`/`IfEqual` steps derive `report_outcome=publish` only for an accepted answer and `needs_attention` otherwise. It forwards every investigation field unchanged plus that routing field. It accesses no child projection, SQL or instructions and adds no effect/model cost. A test first failed on the forwarding-only wrapper, then passed with distinct accepted/stopped outcomes.

## Inspection and source retention

The example needs configuration, counters, pending identities, results and a consistent operator snapshot. Existing `runtime.Inspect` supplies one public snapshot; the application decodes its public rows and derives every summary from that snapshot, following the Phase 1E evidence example's validated pattern. This duplicates a small public-row decoding helper rather than adding a convenience API to the runtime.

The CLI snapshots each manifest document independently, verifies its SHA-256 and byte length, and retains the original bytes/ID. Manifest paths are relative to the directory containing `corpus/`. `effects.Asset.Version` remains the content hash required by the existing contract; editorial manifest version `1` is not substituted for it. Reopen/replay do not read the manifest or source files. The baseline option provisions the unchanged public `evidence.Provision` with exactly the same assets.


## Development-informed query and evidence-scope correction

Subsequent development exposed a concrete presentation mismatch within the existing basis: one model emitted a 336-byte, 50-lexical-term query. It passed the public 512-byte field schema but lookup correctly rejected it against its unchanged 32-term execution contract. Other broad queries reacquired headers, and a reviewer accepted an incomplete candidate after finding the requested facts absent from supplied snippets. These are authoring-policy and retrieval-quality obstacles, not missing interpreter capabilities.

The initial decision in [development-findings.md](development-findings.md) was to hold source. After independent blinded grading/adjudication found only one of eight delivered outputs fully satisfactory, parent authorized one bounded prompt revision before heldout unsealing. That earlier decision and all original results remain retained. The selected correction changes only the shared helper planner and investigation synthesis/review prompts: disclose the actual tool limits, ask for discriminative lexical queries, distinguish snippet gaps from corpus-wide absence, and use the existing bounded review/revision route for missing substantive aspects. No decomposition, helper synthesis, branch, API, runtime, adapter, ranking, corpus or rubric change was made.

Both baseline and investigation use the same revised helper. New stores compile new ordinary artifact IDs; old stores retain old prompts and replay unchanged. [development-prompt-revision.md](development-prompt-revision.md) records exact before/after prompt hashes and raw bytes, the original query counterexample, alternatives, development-informed comparison scope, and the independent revision-2 evaluation boundary. Controlled tests cannot establish that the prompt revision improves model quality.


## Optional exact-span exclusion contract (Task2C)

The smallest retained workload is an ordinary coordinator holding two public helper source identities while requesting another bounded lookup. Legacy `{question}` and `{query}` records reject extra fields, so that caller cannot express “exclude these exact spans” through the public interface. Development-v2 still reacquired initial headers after prompt revision; [original-repeated-span.json](evidence/retrieval-extension/original-repeated-span.json) retains the actual request, initial provenance and repeated result. This is a reusable public data-contract/adapter-profile obstacle, not a missing interpreter semantic operation.

The selected extension adds `lookup.ExcludingInputSchema()` with two optional all-zero-or-exact document/version/start/end tuples and protocol `lookup/tokens-excluding/v1`, plus `evidence.ExcludingRequestSchema()` and `NewExcluding`. The existing builder and interpreter carry these fields through ordinary messages and effect inputs. Initial investigation helpers remain legacy; revision helpers receive exact e1/e2 public answer spans, or zero slots for non-answers. Scoring, ranking, tie breaking, bounds, prompts, output schemas and the one-revision/13-model/four-lookup ceiling stay unchanged. No extra lookup, retry, provider catalogue or runtime policy was added.

The original zero-Input query-only direct adapter convenience is retained as an explicit compatibility case; every nonzero declared schema is validated and exclusions require the new declaration. Legacy helper/retrieval artifact identities are anchored to the original development-v2 store in tests. [retrieval-exclusion-contract.md](retrieval-exclusion-contract.md) specifies tuple validation, unknown-reference behavior, exact public APIs, operational bounds, compatibility, alternatives and limitations. A schema-valid mutant that ignores exclusions fails the same independent input/output monitor as corrected positive cases. Controlled same-query ordinary revisions acquire previously unseen spans; that does not yet establish improved live usefulness. Both prior development versions remain retained, and parent owns a separately identified revision-3 evaluation before heldout unsealing.
