# Independent Task 2 transition probes

These are review-only inputs, not application source changes. `fixture_test.go.txt` extracts the controlled adapter from the frozen Task 2 test source. `probe_test.go.txt` independently overrides its replies at three previously untested transitions and checks exact disposition, request counts, retained initial state, public outer routing, and read-only replay. No provider or network is used.

Executed 2026-10-26 against the source identified in `../review-source-manifest.json`; exit 0, three new scenarios passed. `output.txt` retains the console output captured by the review tool.

To rerun, copy the four `.txt` inputs into a temporary directory with the `.txt` suffix removed, adjust only the local `replace` directory in `go.mod` if the reviewed checkout has moved, then run:

```sh
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1f-go-build GOPROXY=off GOSUMDB=off go test -mod=mod -count=1 -v -run TestNewTransitionCounterexamples .
```

Original cases: initial synthesis asks `Which deployment?` after receiving two valid citations; revision synthesis returns `insufficient_evidence` after all four valid citations; second review returns an out-of-enum `execute` decision after an initial valid revise. These try to expose accidental review/acceptance and stale initial-round review state. All were rejected/stopped as intended.
