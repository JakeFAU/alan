# Runner fault isolation

The runtime ports the scheduler correction from ALAN Research
`6913566b01679090546721498b4609355d8ffc79`, based on runtime main
`5fdfa72c34235ea2b09f969b06dfcdd7c281aa7d`. The runtime design commit
`71e37b346b27a32aec04d927fc1a65b346c5ee77` is the verification baseline.
The research manuscript discusses the originating experiment in §2.4.

## Runtime behavior

The ordered ready scan now continues past a fault in a run's next
instruction. The faulty run keeps its projection, sequence and observations.
If the scan exhausts eligible work and the fault persists, `Run` and
`RunProgress` return the fault with run identity and program counter. The
report shows the work already committed and has `Quiescent: false`.

A matching observation can fail its receive schema, omit a required binding or
exceed the successor projection byte limit. The selector retains that
observation unconsumed and tries later candidates in the same order. If none
can advance, it includes the first malformed candidate's identity and cause
in the fault. A normal guard mismatch remains an ordinary wait. Explicit
`Reduce` and `Commit` calls identify malformed candidates with
`ErrIneligible` and preserve their underlying errors, including `ErrLimit`.

Program loading, binding, retained-input validation and observation reads
remain fail-fast. An integrity failure inside the reducer, such as a missing
adapter declaration, remains fatal. Commit, dispatch, settlement and budget
errors also retain their immediate behavior. The port keeps the existing
transactional observation consumption, history and replay path.

## Runtime fork

The ordinary and guarded indexed selectors share a reduction probe. The
ordinary selector keeps its per-scan cache for unbound programs and loads
handoff-bound programs separately. The indexed selector keeps its candidate
frontier, cached programs, primary-key observation loads and negative
certificates for ordinary waits. It leaves faulty ranks in the frontier so
subsequent scans and runner calls can still report them. The runtime keeps
its private selector signatures and distinguishes the aggregate of local
faults from fatal errors at the shared runner boundary. Dispatch, authority,
resource limits, storage format and program semantic versions are unchanged.

## Qualification

| Probe | Runtime baseline | Port |
| --- | --- | --- |
| Output-schema fault before a healthy peer | Both selectors return with zero commits and the healthy peer open. | The first call commits both healthy reductions and returns the fault. The second call returns the same run fault with zero commits. The faulty snapshot is unchanged and replay checks two commits. |
| Oversized candidate before a valid one | Both selectors return `resource limit` before consuming either observation. | Both consume only the valid candidate at sequence 1 and reach success at sequence 2. Replay checks that selected prefix. |
| All candidates invalid | Both report a bare `resource limit`. | Two calls each report the first candidate identity and `ErrLimit` with zero commits. Both observations remain unconsumed. |
| Missing binding and schema mismatch | Each malformed candidate blocks the later valid input. | Each malformed candidate remains unconsumed and the valid input commits. |
| Corrupt retained artifact | An artifact hash error aborts before the healthy peer. | Same result, including when the indexed guard detects the out-of-band mutation and falls back to the baseline loader. |
| Silent child's oversized successful End | Five commits. The child is open at sequence 2 and the parent is at PC 3 with a legal receive. | Seven commits. The child is still open at sequence 2 while the parent consumes its second input and reaches PC 5. The child still reports `ErrLimit`. The in-budget failed End still finishes after eight commits. |

For the indexed receive regression, reopening after ordinary admissions
rebuilds the index from the validated ledger. Its fixture uses a 512-byte
data limit and a 1024-byte payload to leave room for existing composition
freeze records while exercising a receive that exceeds the projection cap.

The regression command below failed against baseline production code before
the fix and passed on the port. To reproduce with the final tests, create a Go
overlay mapping the absolute paths of `internal/executor/{program,runner,static_execution}.go`
to copies of those files from `71e37b3`, then add `-overlay <path-to-overlay.json>` to
the focused command. The final baseline overlay produced the failures shown
above, with both silent-child scheduler-cut tests and the artifact-corruption
control passing.

```sh
focus='^(TestRunner(IsolatesReductionFault|SkipsObservationWhoseBindingExceedsProjectionLimit|ReportsFirstInvalidObservation|SkipsMalformedReceiveCandidates|FailsFastOnArtifactCorruption)|TestSilentChild(EndProjectionLimit|ParentSchedulerCuts))$'
go test -count=1 -v ./internal/executor -run "$focus"
go test -race -count=1 -v ./internal/executor -run "$focus"
go test -count=1 -timeout=60m ./internal/executor ./runtime
go test -tags=integration -count=1 -timeout=60m ./internal/executor -run '^TestStaticExecution'
staticcheck ./internal/executor ./runtime
go test -count=1 -timeout=60m ./...
go build ./...
```

These are finite single-host schedules under the existing Store ownership
model. They establish progress in the tested cuts. Budget exhaustion,
unresolved external work and storage failures can still stop advancement.
