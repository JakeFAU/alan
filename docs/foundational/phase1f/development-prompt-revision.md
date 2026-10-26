# Development-informed prompt revision 2

This is one bounded authoring-policy revision before heldout unsealing. It changes three prompt constants, with no branch, guard, schema, API, compiler, runtime, adapter, retriever-ranking, corpus, question or grading-rubric change. It has not been paid-tested by its implementer, and no quality improvement is claimed from the controlled tests.

## Why the earlier hold-source decision was revisited

The original [development findings](development-findings.md#scope-decision) explicitly chose to hold the reviewed source after observing retrieval failures. That historical decision remains intact. It was revisited by the parent after [independent blinded grading](evidence/live-development/grading-report.md) and [adjudication](evidence/live-development/adjudication.md) found only one of eight delivered development outputs fully satisfactory. Correctly preserved operations and byte-valid citations did not make the remaining answers complete or well supported. This change is therefore development-informed, not a retrospective improvement to the initial comparison.

The exact initial outcomes, stores, original grades and adjudicated grades remain unchanged. All eight initial outputs, including the empty invalid result, stay in the initial-version denominator. The next authorized comparison is one new development pair per case, labeled revision 2 and independently graded; it is not a replacement, retry or correction of the first run. No further tuning is authorized by default. The implementer read only the permitted development evidence and public materials, did not read heldout files, and did not unseal anything. Parent owns source review, subsequent evaluation and heldout release.

## Concrete evidence and selected change

1. The [dev-01 baseline inspection](evidence/live-development/dev-01-baseline/inspect.json) retains a 336-byte, 50-term query. It satisfied the 512-byte schema but exceeded lookup's existing 32-term limit, producing `query_invalid` and the helper's explicit `invalid_result`. The [extracted original counterexample](evidence/prompt-revision-v2/original-query-counterexample.json) preserves the exact query, original inspection hash, planning/lookup results and public result. The source inspection/store were not modified.
2. The [dev-01 investigation inspection](evidence/live-development/dev-01-investigation/inspect.json) shows full-question context repeatedly retrieving compatibility/header lines instead of the requested action rule. Its initial reviewer requested revision, but both new queries again matched the header; final review accepted an incomplete qualified answer. The [dev-04 investigation inspection](evidence/live-development/dev-04-investigation/inspect.json) similarly retains queries that reacquired report metadata without the reports' actual findings.
3. The shared evidence planner now discloses both existing limits: 512 UTF-8 bytes and 32 lexical terms, including punctuation splitting and repeated-term counting. It describes exact lexical matching without stemming and asks for a few discriminative entity/fact/action terms instead of a verbose question restatement. Relevant action/status vocabulary and alternate word forms may distinguish an answer-bearing line from common header context. It does not add corpus entities, answers, target values, query expansion code or a different retrieval algorithm.
4. Investigation synthesis now distinguishes retrieved excerpts from the complete corpus. Missing excerpt content is a retrieval gap unless an explicit source establishes absence. One report's finding or metadata about multiple reports does not establish their agreement/disagreement. A useful partial candidate with a retrievable missing aspect can use the existing provisional `answer` path to reach review, rather than prematurely treating a retrieval gap as established corpus insufficiency. Existing clarification and insufficient-evidence choices remain available; no specific answer is forced.
5. The reviewer now checks all requested substantive aspects rather than only checking the claims the candidate elected to make. A narrowly qualified omission is not automatically a complete answer. When a focused query could supply a missing aspect, the reviewer can request the existing bounded revision with complementary subquestions seeking distinct missing/contrary/independent facts rather than restating broad context or duplicating established facts. Claims about plural reports require their actual findings. Ordinary coordinator branches still decide whether a revision remains authorized.

Decomposition was left unchanged: it already asks for distinct substantive aspects and focused subquestions. The retained failures were downstream query construction and the treatment of missing excerpts. The evidence helper's synthesis prompt also remains unchanged. Only the three specified constants changed; a source audit verified all other bytes of both builders against the retained pre-change snapshots.

## Shared baseline and unchanged bounds

Both systems still call the same public `evidence.New` implementation. The baseline remains the simple public `evidence.Provision` application; its shared planner prompt is now revised, so it must not be described as byte-identical to the original Phase 1E or first development baseline. Investigation's four ordinary helpers receive that same revision. No one-system-only helper advantage was introduced.

The tool still returns one best-matching line, has the same lexical ranking/tie behavior and requires at most 32 terms/512 query bytes. Existing schema validation, citation-provenance checks, unknown handling and all stopping branches are unchanged. The investigation ceiling remains one revision, 13 model requests and four lookups; zero revisions still limits reachable work to seven model requests/two lookups. The configured one-question baseline remains two model requests/one lookup. More instruction bytes can change input-token usage and cost; the revision-2 comparison must retain actual usage rather than treating compute as unchanged.

Changed ordinary prompt/template bytes produce new compiled artifact IDs for newly provisioned stores. Existing stores retain and replay their original artifacts and prompts; no recompilation on recovery, migration, opcode change, or semantic-version bump is involved. The original Phase 1E archives and first development evidence remain untouched.

## Prompt and source identities

Exact raw UTF-8 prompt bytes (no added newline) are retained in [prompt-revision-v2](evidence/prompt-revision-v2/prompt-manifest.json), with before/after files for all five relevant prompts and full pre-change source snapshots. Hashes are SHA-256 of those raw prompt bytes, not the JSON encoding.

| Source / constant | Change | Before SHA-256 | After SHA-256 |
| --- | --- | --- | --- |
| `examples/evidence/evidence.go` / `planningPrompt` | revised | `e84eab31864890faf3c3ac7215d550bce296f125f3245a734ef98ff9bdc2d702` | `7d2c6d5af2b634fa3fb7d5d356d8afd60c8627faafe15852935599fd81c64820` |
| `examples/evidence/evidence.go` / `synthesisPrompt` | unchanged | `a08f66eedf50b640e7eecf5c32d1bf1887ea8eb5c686232db5d2467f901f5ac5` | `a08f66eedf50b640e7eecf5c32d1bf1887ea8eb5c686232db5d2467f901f5ac5` |
| `examples/investigation/investigation.go` / `planningPrompt` | unchanged | `3d70ade8828038594b6df4bfc244e149ef92c46fece6698291b29566e6ec6f4a` | `3d70ade8828038594b6df4bfc244e149ef92c46fece6698291b29566e6ec6f4a` |
| `examples/investigation/investigation.go` / `synthesisPrompt` | revised | `722b763207b522f79202f775252d1120139ec37a861f9f9a4ff6a1379bbc5318` | `e99fd3cfb1dd6baf0f0f6635585d51caf9c722ed93bac76df22ea4b7f53302b9` |
| `examples/investigation/investigation.go` / `reviewPrompt` | revised | `e351655adcd94c5679d7ef274812bb2e45dd3dfb6ec7a2ae91388d776cd7086f` | `ea88a691da4daf1e3b153c114e5a6b7b38d29dee2191449bd9c92cf0dfaac77f` |

| Source | Before SHA-256 | After SHA-256 |
| --- | --- | --- |
| `examples/evidence/evidence.go` | `327b45abcfe76cd72cc15b5cf812bb9b5b5da995a3905960b2c040638eb42ec1` | `8a0ff3cbdd3f75a3e5dee91263102f5e1accd54f4b5d3daf0c892a0ebb79a3ec` |
| `examples/investigation/investigation.go` | `1f7823a5ba483a47ddc7afca4718c8a53df08fa66023d766ebf7698960625063` | `9a78528763da73618a5ee8358494fe962abbfdf9330432f25345a661b7a230ea` |

## Alternatives and limits

Keeping the original prompts was the documented first choice; independent development grading made one bounded correction worthwhile before unsealing. Relaxing lookup's term limit, adding stemming or alternate ranking, fetching more lines, increasing the revision budget, adding scenario-specific query templates, or changing the rubric would alter the intervention and are outside this revision. Exact prompt-string assertions would merely mirror prose and cannot establish improved model behavior, so no brittle snapshot test was added.

Existing controlled public-example/CLI tests check contracts, ordinary accept/revise/stop branches, counts, IDs, citations, retained assets and replay. They cannot establish that an uncontrolled model will follow the new advice or retrieve the right facts. The model may still exceed the query contract, miss needed lines, fail to find contrary evidence, request an unhelpful revision, or stop incompletely. The unchanged guards must continue to represent those outcomes honestly. Independent revision-2 grading, actual usage and the original-version comparison remain separate evidence.
