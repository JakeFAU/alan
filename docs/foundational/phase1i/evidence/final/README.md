# Controller verification records

Each command JSON records the exact argv, selected environment, source inventory,
exit, elapsed time and hashes of raw gzip stdout/stderr. Test records list actual
pass/skip names. `capture.py.txt` preserves the capture script. Its first temporary
version failed Python parsing at the format-command list (mismatched closing
parenthesis); no Go command ran until that script-only typo was corrected.

The first gate-source inventory uses the captured JSON insertion order (Path
component sort); its `ordering_clarification` makes that explicit. The final
phase `source-manifest.json` uses canonical repository-path UTF-8 ordering.
Different inventory algorithms are not interchangeable source IDs.

`publish/` contains final builds (including the standalone consumer executable),
focused vet, gofmt and diff checks against the final phase source ID. Earlier
semantic/executor regression commands here have identical relevant inputs after
the test-only probe/decoder fixes; new source-specific model/runtime gates are
recorded in their respective evidence directories. No broad whole-Phase1H rerun
is implied.

`archive-source-isolation.json` preserves the discovered unintended Go package
from two archived `.go` snapshots, its compile failure, byte-preserving `.go.txt`
renames and corrected package list. Those snapshots are evidence, not executable
packages. `package.py.txt` is the exact source/index curation method; source files
and original evidence are in this checkout. It is not a semantic oracle.
