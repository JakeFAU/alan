# Phase 1I scoped manuscript reconciliation

Status: **DONE_WITH_CONCERNS**. The consequential claim/status edits are complete;
whole-branch independent review and final runtime qualification remain separate.
The bounded model negative is reported as model evidence. No general theorem,
compiler proof, runtime confirmation or final review is invented.

Workdir: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.
Historical manuscript baseline: `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`.
Approved K1 contract SHA256:
`cadd87220e6b9a06ccb597dc808b31ddae072d486d13feb703e0fb480294d865`.
Before/after manuscript hashes and inspected model-result hashes are recorded in
`source-identity.json`. Original affected passages, including the superseded
proof text and summaries, are preserved with baseline line locations in
`original-excerpts.md`. No old phase evidence was altered.

## Result and changed claims

The original general congruence argument does not establish stability under peer
spending/closure, invocation admission/depth/member cost, or the connection
between abstract interleavings and the concrete scheduler. It remains an open
claim under its stated premises. The current weak-contract unequal-cost witness
does not refute Definition 4.6, which requires equal debits on every matched
transfer. The manuscript preserves that exact-debit definition and distinguishes
upper-bound contract refinement from symmetric equivalence.

The independently executed runtime-boundary model extension supplies a narrower
negative: ordinary root success may fence a delayed parallel peer's S admission
with `scope_closed`, at I=R=3/D=2, without an external cancellation and while the
original completion entitlement remains true. Inspected model reports include
this counterexample for K1, local A and helper B. Their full boundary-extension
summaries each retain 3,840 configurations and 84 entitled cases: all 8 entitled
parallel configurations fail Gc; the other 76 complete in those finite graphs.
This falsifies the frozen parallel reference completion premise/runtime
correspondence, not an implication with a valid C[K1] premise and not the
stronger theorem's exact-debit premise. These are attributed workstream results,
not experiments run by this manuscript editor. Runtime confirmation/final
qualification and independent review are still separate at this report's cut.

The general recovery claims directly relying on the withdrawn arguments are
narrowed: no-await dependency alone is not resource/world/closure independence;
a late authenticated receipt does not alone rejoin future bisimulation; hiding
private retry state does not erase shared debit or external effects. Replay's
historical argument and conservative no-repeat/no-fabrication facts remain
distinct from the proposed future simulations.

## Markdown–LaTeX mapping

| Claim | Markdown anchor | LaTeX anchor | Synchronized consequence |
| --- | --- | --- | --- |
| Admissible grammar | Definition 4.5, §4.5.1 | Admissible Contexts definition | Structural compatibility is not completion entitlement; remaining demand/reservations/depth explicit |
| Exact-debit relation | Definition 4.6 and premise audit | Same existing definition plus premise audit | Existing transfer/readiness/divergence clauses retained; unequal debit excluded from theorem counterexamples |
| Scheduler | Lemma 4.7 | `lem:scheduler` | Nonblocking structure qualified by continued service; O(1)/numeric global bound and unique-normal-form inference withdrawn |
| General substitution | Theorem 4.8 | `thm:congruence` | Historical numbered claim explicitly open; interference/invocation/hiding/scheduler obligations replace invalid proof assertion |
| Bounded investigation | §4.5.2 Phase1I paragraphs | Phase1I paragraphs after congruence | 3,840/84 frozen scope, parallel Gc negative with 8/76 partition, separate evidence categories, preserved model corrections |
| Recovery definitions | Definitions 6.3–6.5 and qualification notes | Two-Regime subsection prose | Future simulation requires interference/schedule/coupled-world premises; no automatic late-evidence or retry-hiding equivalence |
| Recovery claim | Theorem 6.6 and Part II/III limits | `thm:fundamental` and recovery limits | Replay argument kept; general future simulations explicitly open; safety not promoted to equivalence |
| Cross-generation consequence | §6.6 opening | Paragraph before `thm:crossgen` | Within-generation idempotent safety attributed to endpoint contract, not an unproved future-simulation theorem; no retry permission after closure |
| Summaries | Abstract items 3/4, §1.4, §8.1/8.2 | Abstract, introduction items, conclusion qualification | No stale assertion of completed general congruence/recovery proof |

Existing labels and theorem/lemma/definition/proposition environment counts are
preserved. The TeX remains a condensed manuscript representation; this edit
aligns the affected claims/status, not every dissertation paragraph. No new
numbered theorem or reference was introduced.

## Checks actually performed

- Read the scoped brief, frozen contract, claim/API audits, candidate/context
  review status and current model outputs; inspected the scope guard at
  `internal/executor/invocation.go:165–177` and the separate `RootTerminalFence`
  graph extension. The manuscript editor did not rerun Go model/runtime tests.
- Computed SHA256 source identities and retained original excerpt line numbers.
  Inspected the three root-success counterexample JSONs and three boundary
  census JSONs; the report records their exact hashes, not merely an agent's
  assertion.
- Compared changed content and searched for residual complete-congruence,
  unconditional recovery/retry-hiding claims; corrected two stale TeX introduction
  summaries and the MD cross-generation introductory dependency.
- Checked unchanged MD §2.5/Phase1H case-study block and TeX loop-theorem block.
  Compared all TeX label sequences and citation-key sequences: unchanged.
  Environment counts before/after: theorem 9, lemma 2, definition 2, proposition 1.
  Checked environment nesting; balanced. Bibliography SHA256 is unchanged.
- `git diff --check` passed on the manuscript edits. This is a textual check,
  not mathematical proof or rendering validation.
- No terminal `pdflatex`, `tectonic` or `latexmk` was found. The desktop compiler
  attempted the complete edited document and returned compile-failed at the
  pre-existing unchanged line 215 `\llbracket` undefined control sequence. No
  complete PDF was produced. `full-compile-summary.json` records the exact source
  hash and a summary of the returned diagnostic; it is not a full tool log.
- The final affected §4.5/recovery sections were compiled in a standalone
  diagnostic fragment using the original preamble. The desktop compiler returned
  **success**, twice (the second after the model-result wording update).
  `changed-sections.tex` and `fragment-compile.json` retain the final fragment
  and successful tool response. This checks the changed sections' TeX syntax;
  it does not qualify the complete dissertation, bibliography, global numbering
  or visual layout.

## Remaining obligations and deliberate exclusions

- Full-document compilation remains blocked by the pre-existing `\llbracket`
  issue; no TeX installation or unrelated preamble repair was attempted.
- Final public-runtime/process qualification, delayed-peer runtime witness and
  independent whole-branch review must determine the final evidentiary scope.
  Manuscript text currently attributes the boundary-extension result explicitly
  to the finite model and refers readers to source-specific Phase1I records.
- General Theorem 4.8 implication, hidden divergence/ready-set closure and
  interference-stable resource families remain open. No stronger conclusion is
  inferred from the bounded negative or from agreement between finite graphs.
- General loop ranking/Theorem 2.6, distributed authority, arbitrary context
  grammars, multiple replacement sites, migration beneath retained state,
  compiler correctness, and whole-dissertation prose/math parity are untouched.
- The original model's collapsed helper/wrapper failure-consumption/freeze and
  later premature-payload-visibility corrections remain preserved as failures;
  optimistic original reference results do not establish runtime correspondence.
- No production code, candidate/model source, contract, bibliography or unrelated
  phase document was modified by this workstream. No subagents, paid calls,
  external-source verification, commits or publication were performed.
