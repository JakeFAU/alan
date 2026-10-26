# Task 5 T5-R1 scoped independent re-review

**Finding T5-R1: ADDRESSED. Specification gate: PASS for this correction. Code-quality gate: PASS for this correction.** No new fix-diff breakage or unresolved finding was identified. This closes the single medium-severity issue in `task-5-review.md`; that original review and its counterexample remain historical evidence.

Reviewed the appended implementer report, exact `fixround1/fix.diff`, changed-file and complete scoped source manifests, frozen source, commands, red regression and focused normal/race evidence. Only two Go test files and the new retained-input fixture changed. All eight current scoped files match the corrected manifest and frozen copies. There is no production change in this correction.

The actual C/D/E process cases and the new regression now call the same error-returning `validateDurableBoundary`. It preserves authorization/epoch/subject/generation checks, matches the adapter attempt to the event, compares the event sequence with the retained adapter Intent.Sequence, and separately requires the committed intent sequence and full descriptor to agree. The expected sequence comes from retained evidence rather than a fixture constant. The existing fully specified controls remain intact.

I independently compared the new testdata filter, original acknowledgment, substituted acknowledgment, provenance digest and every retained snapshot row against the original independent-review evidence. They match. The new regression deliberately leaves Sequence wildcard in the filter, so both acknowledgments reach the durable check; only the original passes that check.

A fresh overlay probe against **current corrected source** used the original **full** stopped snapshot, independently of the newly added subset fixture. Six cases passed:

- Original actual acknowledgment/full snapshot accepted.
- Original Sequence 15→16 counterexample rejected with `boundary sequence 16 differs from retained adapter intent sequence 15`.
- Mutating both acknowledgment and adapter-attempt sequence rejected against the unchanged committed intent.
- Altering the committed intent target while retaining its sequence rejected by full descriptor equality.
- Missing adapter attempt rejected.
- Missing committed intent rejected.

The new repository wildcard regression also passed in that same targeted invocation. Exact overlay source, mapping and complete output are retained under `evidence/task-5/independent-review/rereview1/`. Command:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off go test -count=1 -tags integration -overlay /tmp/alan-task5-rereview1/overlay.json -run '^(TestIndependentR1CurrentHelper|TestWildcardAcknowledgmentRejectsRetainedSequenceSubstitution)$' -v ./internal/evidencetest
```

I also verified the retained corrected normal/race logs both contain original acceptance and substituted rejection, and the race log records `-race` in both actual child build commands. Implementer red evidence shows the exact new wildcard regression failed before the fix. The older portable reviewer overlay intentionally uses preserved pre-fix sources; its PASS reproduces the historical defect and is **not** evidence of current corrected behavior.

This review ran no broad suite, contacted no provider, changed no implementation source, and made no commit. Parent owns final whole-tree verification and archive reproduction under the corrected source identity. The final whole-branch review remains a separate gate.
