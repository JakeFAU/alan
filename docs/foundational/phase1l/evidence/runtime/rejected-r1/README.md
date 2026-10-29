# C_inv concrete runtime evidence

The final corrected-source capture has 36 focused leaf cases in each normal and
race run: eight positive payloads, eight controls, two B completion orders,
sixteen detached-record corruption checks, one incomplete-prefix check and one
recovery predicate. The process child is the parent's os.Executable, including
race instrumentation. Programs use common MaxSteps200/MaxDispatch200/capacity1.
No production/runtime, original A/B/helper/S or frozen model source changed.

From the repository root with the pinned local Go1.27.3 toolchain/dependencies:

```sh
python3 docs/foundational/phase1l/evidence/runtime/capture.py --focused
python3 docs/foundational/phase1l/evidence/runtime/audit.py
```

capture.py removes ALAN_* and OPENAI_API_KEY, sets GOTOOLCHAIN=local,
GOPROXY=off, GOSUMDB=off, GOFLAGS=-mod=readonly and
GOCACHE=/tmp/alan-phase1f-go-build; ALAN_I_ARTIFACTS is a fresh temporary directory.
It records exact commands, environment overrides, timestamps, exits, named cases,
output, source aggregate, journals, raw decisive inspections and separate
raw/prefix/component/G results in records.json.gz. sources.json binds298 files,
including all protected inherited sources. audit.py independently checks source
hashes, raw original-handle/commit/selection links, results/debits and process
identity/consumed-row preservation, then updates index.json with all file hashes.
This audit is bounded evidence checking, not a second runtime or general proof.

revision1/ preserves the earlier source-bound 35-leaf normal/race capture,
public-consumer normal/race5-leaf checks and scoped vet, all exit0. Its default
`go test -json -count=1 -timeout=10m ./...` exited1 after602.125s because
examples/adaptive/TestAdaptiveWorstCaseBytes/nested=false/sixteen reached that
watchdog. README specifies60m for the repository default suite. No source or
assertion was changed to hide this; the controller owns any unaffected-package
retry. That earlier capture is not relabeled under the corrected adapter.

A preserved substantive RED in revision1/charge-row-red.json.gz shows the former
adapter accepted a detached extra unlinked I/R charge plus matching inflated root
totals. It includes original source, temporary overlay test, command/environment,
mutated raw witness and wrongly accepted verdicts. Only the adapter was corrected:
accepted/refused admission-to-charge cardinality and root totals are now checked.
The final extra-charge regression rejects it. source-snapshot.json.gz retains the
three changed pre-correction Go files; other sources are bound by the old manifest.
The historical overlay RED is reproducible at that retained source, not claimed
to remain RED against the corrected adapter. No live database was modified.

One exact B recovery predicate is instantiated once per normal/race profile:
both root invokes committed; root open at PC3; original candidate outcome consumed
once; S admitted but member sequence0/PC0 with its original request unconsumed and
no S publication/delivery. All three original admissions/charges already exist.
ACK binds PID/store/token/scenario/boundary/phase/root/epoch/sequence/S intent.
The child receives actual SIGKILL; a new PID only Opens the same DB and Runs the
remaining original work. Epoch2 becomes3; handles, artifacts, logical identities,
old consumed_sequence and charges persist; each original outcome is selected once.
These are selected public/process executions, distinct from automatic positive
samples, finite-model enumeration, distributed recovery or arbitrary contexts.
