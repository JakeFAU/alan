# Independent reference construction evidence

`reference-r1.json` is the review-freeze manifest; `reference-r1.tar.gz` preserves
its source/input snapshot. The new construction and its outputs were unopened.
This is Task3a construction evidence, not exhaustive qualification.

Final source-bound records:

- `construction-normal-freeze.json.gz`: 66 directed/adversarial leaves pass.
- `construction-race-freeze.json.gz`: the same 66 leaves pass under `-race`.
- `vet-freeze.json.gz`: reference-package static check, exit 0.
- `toolchain.json.gz`: Go1.27.3 darwin/arm64, informational version identity.

Each compressed record includes exact argv, controlled environment, exit,
stdout/stderr, actual test leaves and permitted source hashes. `capture.py` strips
ALAN_* and OPENAI_API_KEY, pins offline/readonly Go settings and stages records
under `/private/tmp` before the worktree copy. Labels are exclusive: use new names
for reruns. `freeze.py` refuses to overwrite the r1 archive/manifest.

Earlier passing normal/race/format/static captures are source-bound intermediate
construction records. `capture-write-failure.json` retains the two initial
unrecorded attempts and the observed write-permission failure; their inner test
verdicts are unknown, not RED/GREEN evidence. No full product/local/replacement
qualification was run. See `../../02-lifting.md` and the task-3a report for the
argument and remaining review gate.
