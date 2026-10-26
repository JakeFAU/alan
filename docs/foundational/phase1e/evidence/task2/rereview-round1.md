# Task 2 scoped re-review — fix round 1

## Verdicts

**Specification compliance: PASS for the declared Task 2 boundary after these fixes.** R1 and R2 are **ADDRESSED**. No additional specification defect was identified in the eight-file fix diff.

**Code quality: PASS for this fix round.** The fixes address the causes with bounded control-flow analysis and one shared identifier policy. No blocking new breakage was identified. These verdicts close the previous Task 2 review findings; they do not qualify Task 3 or the full phase.

## R1 — ADDRESSED

`author/compile.go` replaces last-opcode join omission with `compiler.reachesBoundary(start, end)`. It traverses the lowered body's actual successors, resolves recorded named jumps, treats terminals as stopping, and inserts the join precisely when a reachable path reaches the lexical end. A jump to a trailing label now reaches that boundary even when the last emitted opcode is a jump. Jumps to unresolved enclosing labels are exits from the current body, rather than lexical fallthrough; an enclosing body's later traversal can resolve its own labels after expansion. Nested branch targets are already lowered before traversal. The visited set and existing instruction/nesting bounds keep this finite, and final forward-target validation remains in place.

The corrected original probe artifact has the named jump targeting the added join at PC 2, that join targeting Success at PC 4, and the false branch starting at PC 3. `review-round1/probe-after.log` reports `expected=succeeded actual=succeeded`; `probe-before.log` retains the prior failure for the same input. The executable regression verifies both the live durable projection and offline replay for the original trailing-label case, nested trailing labels, repeat exits at the arm boundary, terminal arms, and false-branch selection. Tests of the terminal-arm case guard against reintroducing the earlier invalid unreachable join.

## R2 — ADDRESSED

`schema.CheckIdentifier` requires a nonempty, valid UTF-8 string and does no normalization. `author/identifiers.go` applies it before lowering to emitted step destinations, references, labels, effect/port references, request identities, payload/update keys, and binding fields/destinations. Initial value keys and Application subject keys are checked separately. Metadata validation covers definition names when supplied, template keys, effect names/capabilities, and port names/peers/obligations; schema field names use the same helper. This prevents JSON's replacement of invalid bytes from aliasing authored names. Emitted unreachable steps are checked; zero-repeat bodies emit no names and retain their documented omitted-body behavior.

The original `0xfe`/`0xff` input now fails compilation with `steps/0:set: invalid UTF-8 identifier "\xfe"`, as retained in `review-round1/probe-after.log`. The new compiler/application assertions cover this exact scenario and the adjacent identifier surfaces. The runtime Unicode regression preserves Japanese public identities and separately stored composed `é` and decomposed `e\u0301` variable names through execution/replay. The Unicode Request/Await compilation regression also passes. Thus the fix rejects invalid encodings without merging valid Unicode names.

## Compatibility and evidence

The fix changes authoring/metadata validation and compiler lowering, not the reducer or stored instruction interpretation. The legacy-semantic early return in `validateMetadata` remains before these new identifier checks. Legacy Value representation, generic opcode semantics, and omitted author metadata encoding are unchanged. The retained `metadata-regression.log` shows PASS for the exact legacy byte fixture, missing receive-contract rejection, and unknown compiler/semantic version recovery rejection.

Read the appended Task 2 fix report and the complete `evidence/task-2/review-round1/round1-fix.diff`. Independently regenerated that diff against `evidence/task-2-review/reviewed-source.tar.gz` and current source: exact match for all eight changed files. Independently verified all 20 current owned source/test file hashes against both `review-round1/after-manifest.json` and refreshed `evidence/task-2/frozen-source-sha256.json`; the manifests agree and every hash matches.

Inspected the following supplied evidence under `.superpowers/sdd/00-plan/evidence/task-2/review-round1/`:

- `probe-before.log`, `probe-after.log`, and `probe-after/trailing-label.artifact.json`: original reproduced wrong results, then corrected branch behavior and UTF-8 rejection.
- `regressions-red.log`: executed assertion failures before fixes, distinct from build failures.
- `regressions-green.log`: focused author/runtime regression commands pass after fixes.
- `covering-tests.log`: 72 passing test/subtest events, no FAIL lines, across author/runtime/schema; includes the external-module separate-process smoke, conditional continuation cases, and valid-Unicode preservation.
- `metadata-regression.log`: focused legacy and version-recovery tests pass.

The exact original public probe input source remains in `evidence/task-2-review/main.go`, alongside the original artifact/output evidence and frozen source archive. The prior review and its evidence remain unchanged.

## Limits

This was a scoped fix-diff re-review, not a restarted broad review. No production/test edits, commits, subagents, full-suite/race reruns, or extra behavior probes were performed. Behavioral verification above comes from inspected retained executable-test/probe outputs; independent work here verified source identity and analyzed the fix. Whole-phase full/race/vet qualification remains parent-owned. Adapter dispatch, providers, corpus/application policy, and future Task 3 functionality are outside this verdict.
