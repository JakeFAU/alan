# Task 2B bounded prompt correction report

Status: **DONE**. Code ownership was frozen/released before waiting on the post-change suite. No further implementation or test edits are planned. Parent owns independent source review, labeled revision-2 development runs/grading, and the final-source full-suite/race gate.

## Exact intervention

Only three ordinary prompt constants changed:

- `examples/evidence/evidence.go: planningPrompt`: presents the unchanged 512 UTF-8 byte and 32 lexical-term limits, punctuation/repetition counting, exact token matching without stemming, and concise discriminative entity/fact/action queries instead of broad natural-language restatement. Both baseline and investigation use this same public helper.
- `examples/investigation/investigation.go: synthesisPrompt`: distinguishes retrieved snippets from the complete corpus, avoids inferring report agreement/conflict without findings, and keeps a useful qualified partial candidate on the existing review path when targeted retrieval could address a missing substantive aspect.
- `examples/investigation/investigation.go: reviewPrompt`: checks completeness against requested aspects as well as support for existing claims, rejects absence-from-snippets as proof of corpus absence, and uses complementary missing/contrary evidence questions for the existing bounded revision.

Investigation decomposition and evidence synthesis prompts are unchanged byte-for-byte. A one-off source audit replaced prompt literals with a common sentinel and verified both entire source files otherwise byte-identical to their retained before snapshots. No tests, finite branches, guards, schemas, APIs, versions, runtime, adapters, lexical ranking, budgets, corpus, questions, or grading rubric changed. No specific corpus entity, gold answer, scenario threshold or heldout value was inserted. No subagents, commits, branch changes, credentials, paid calls or heldout reads occurred.

## Preservation and comparison scope

`docs/foundational/phase1f/evidence/prompt-revision-v2/` contains raw before/after bytes for all five prompts, full before-source snapshots, `prompt-manifest.json` with hashes/byte counts, and `original-query-counterexample.json`. Raw prompt files have no added newline; hashes cover UTF-8 prompt bytes, not their JSON encoding. The extracted dev-01 baseline counterexample preserves its exact 336-byte/50-term query, original inspection hash, planning/lookup results and public invalid result. The original inspection, store and all original results remain untouched.

`development-prompt-revision.md` contains before/after prompt and source hashes, exact rationale, unchanged limits, alternatives, original evidence links and the development-informed evaluation boundary. `authoring-obstacles.md` appends the concrete query/scope obstacle and selected public-only solution. The original `development-findings.md` hold-source decision was not erased: the new document explicitly states parent revisited it after independent blinded grading/adjudication found only one of eight delivered outputs fully satisfactory.

Baseline remains simple public `evidence.Provision`, but its shared planner is now revised and must not be called byte-identical to the initial baseline. Old stores keep old artifacts/prompts; only new provisioning compiles the changed bytes into new IDs. No migration or semantic-version change is needed. The initial eight outputs remain the initial-version denominator. Parent's new development pairs are separately labeled revision 2; no further tuning is authorized by default. Increased prompt bytes can increase input usage and must be reflected in actual cost/usage reporting.

## Source hashes

Manifest: `.superpowers/sdd/00-plan/task-2b-source-manifest.json`.
Aggregate SHA-256 of its canonical files array: `cd45431301ed2fcae374adf53f5aa44cad2148d536213bd09dcba0210849433d`.

| Program source | Before SHA-256 | After SHA-256 |
| --- | --- | --- |
| `examples/evidence/evidence.go` | `327b45abcfe76cd72cc15b5cf812bb9b5b5da995a3905960b2c040638eb42ec1` | `8a0ff3cbdd3f75a3e5dee91263102f5e1accd54f4b5d3daf0c892a0ebb79a3ec` |
| `examples/investigation/investigation.go` | `1f7823a5ba483a47ddc7afca4718c8a53df08fa66023d766ebf7698960625063` | `9a78528763da73618a5ee8358494fe962abbfdf9330432f25345a661b7a230ea` |

All changed/added task files, including preservation evidence:

| File | SHA-256 |
| --- | --- |
| `examples/evidence/evidence.go` | `8a0ff3cbdd3f75a3e5dee91263102f5e1accd54f4b5d3daf0c892a0ebb79a3ec` |
| `examples/investigation/investigation.go` | `9a78528763da73618a5ee8358494fe962abbfdf9330432f25345a661b7a230ea` |
| `docs/foundational/phase1f/authoring-obstacles.md` | `d354c6ef406413b55bc9b59e32bdf0d75b5ad3ab1791f2f2ddd2bebc80703733` |
| `docs/foundational/phase1f/development-prompt-revision.md` | `96aa292685de372fde90bdbbd9a2b9fc784b5a4d8db8ef4a4e2d57c09d379e95` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/evidence-planningPrompt.after.txt` | `7d2c6d5af2b634fa3fb7d5d356d8afd60c8627faafe15852935599fd81c64820` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/evidence-planningPrompt.before.txt` | `e84eab31864890faf3c3ac7215d550bce296f125f3245a734ef98ff9bdc2d702` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/evidence-source.before.go.txt` | `327b45abcfe76cd72cc15b5cf812bb9b5b5da995a3905960b2c040638eb42ec1` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/evidence-synthesisPrompt.after.txt` | `a08f66eedf50b640e7eecf5c32d1bf1887ea8eb5c686232db5d2467f901f5ac5` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/evidence-synthesisPrompt.before.txt` | `a08f66eedf50b640e7eecf5c32d1bf1887ea8eb5c686232db5d2467f901f5ac5` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-planningPrompt.after.txt` | `3d70ade8828038594b6df4bfc244e149ef92c46fece6698291b29566e6ec6f4a` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-planningPrompt.before.txt` | `3d70ade8828038594b6df4bfc244e149ef92c46fece6698291b29566e6ec6f4a` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-reviewPrompt.after.txt` | `ea88a691da4daf1e3b153c114e5a6b7b38d29dee2191449bd9c92cf0dfaac77f` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-reviewPrompt.before.txt` | `e351655adcd94c5679d7ef274812bb2e45dd3dfb6ec7a2ae91388d776cd7086f` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-source.before.go.txt` | `1f7823a5ba483a47ddc7afca4718c8a53df08fa66023d766ebf7698960625063` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-synthesisPrompt.after.txt` | `e99fd3cfb1dd6baf0f0f6635585d51caf9c722ed93bac76df22ea4b7f53302b9` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/investigation-synthesisPrompt.before.txt` | `722b763207b522f79202f775252d1120139ec37a861f9f9a4ff6a1379bbc5318` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/original-query-counterexample.json` | `ec069e9c9fad7ad3b7a4e82a16cdf91217737436dbca20c241be2b05b4366e5f` |
| `docs/foundational/phase1f/evidence/prompt-revision-v2/prompt-manifest.json` | `115260f7b7eabf78792ef011da28e7d191f4f8368b049000d6df842518fb5369` |

## Verification

Every Go command used `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1f-go-build GOPROXY=off GOSUMDB=off`. No live provider was contacted; provider-facing tests used controlled loopback endpoints.

Before edits:

```
go test -count=1 -tags integration ./examples/evidence ./examples/investigation ./cmd/alan-evidence ./cmd/alan-investigate -run 'TestControlledApplication|TestHTTPReviewRevision|TestCLIManifestInitRecoverInspectReplay|TestCLIUsage|TestCLIHelp'
```

Exit 0. Evidence 2.440s, investigation 2.166s, evidence CLI 0.450s, investigation CLI 1.168s. Raw result: `.superpowers/sdd/00-plan/task-2b-before-tests.log`.

After edits:

```
go test -count=1 -tags integration -json ./examples/evidence ./examples/investigation ./cmd/alan-evidence ./cmd/alan-investigate
```

Exit 0, all four packages passed, zero failed tests. Evidence 3.817s, investigation 30.407s, evidence CLI 1.665s, investigation CLI 0.978s. This includes existing public accept/revise/stop/invalid-output and budget/identity cases, full real-adapter controlled HTTP revision, standalone consumer compilation, retained corpus/recovery/replay, unknown outcomes and CLI checks. Raw JSON: `.superpowers/sdd/00-plan/task-2b-focused.go-test.jsonl`, SHA-256 `813114bdcc058c63c3db6f6f310ee2a9c2c0f84629529e18271a44f37494b1ca`.

A separate preservation/scope audit passed: exactly three changed prompts; all bytes outside prompt constants unchanged; all retained before/after prompt hashes match; every prompt remains below the existing 4096-byte instruction field limit; original counterexample inspection hash still matches and its recorded lookup disposition remains `query_invalid`.

No brittle prompt-string regression test was added. Existing behavioral tests establish that the protocol and boundaries survive changed prompt bytes; they do not simulate or prove model obedience or quality improvement. No new red/green semantic claim is made for stochastic prompt behavior. A redundant full-default or 501-second focused-race rerun was not needed for three constants and was not run, per the task brief; parent owns final-source broad qualification.

## Remaining limits

A model may still exceed the tool contract, choose irrelevant terms, retrieve headers, miss conflicting findings, request an ineffective revision or stop incompletely. Prompt guidance cannot enforce those semantic outcomes. The tool guard and ordinary explicit failure/stop branches remain unchanged. Byte-valid citations and application review are still not independent proof of completeness or entailment.

No revision-2 paid result, quality gain, cost, heldout outcome, or independent approval is claimed here. The useful comparison remains parent-run and independently graded. No implementation work is blocked.
