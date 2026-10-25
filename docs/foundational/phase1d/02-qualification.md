# Phase 1D qualification

The single-host executor passes the declared process-crash contracts, including the central accepted-Retry recovery case. This qualifies the selected programs, schedules and local storage profile. It does not establish general production readiness, arbitrary program equivalence, power-loss tolerance or distributed fault tolerance. All five scoped reviews and the final independent whole-branch review passed. One minor evidence-attribution finding was corrected and independently rechecked; the original finding and final verdict are retained in [03-review.md](03-review.md).

## Source and reproducibility

The implementation base is `a0d337c7b4f7340a55dea11b619cf9b22c25e2b0`. The exact qualified Go source/dependency-file digest is `40a5b1f2ef55a901362e7bb5aca42b4e3acb5d73745b5cfda82d3c57361aef7f`: 53 files, individually hashed in [source-manifest.json](source-manifest.json), including all runtime, CLI, controller, oracle and research source. Both final runs checked these files before and after testing. The digest describes the uncommitted frozen source used for qualification; the subsequent publication commit also contains generated documentation and evidence. No historical record was stamped with this new identity.

The pinned driver is `github.com/mattn/go-sqlite3 v1.14.52`, origin `b0be46fa28d17ee0b65c79774ac0dad84b6db068`, module sum `h1:wVbm2Qnf4OXkqhBTSPuCRZDRnxfbVrrmiCEroVdog8U=`. Effective inspected settings are WAL, synchronous=2 (FULL), foreign_keys=1, busy_timeout=5000; read-only connections additionally report query_only=1. The bundled engine is SQLite 3.53.4, source `2026-07-24 19:02:57 bf7c7f30031888f4e796e429ab3978879485813aaca6f641c7b33e4e09459bcc`. Driver source, including all 106 module files, is available in the complete [source archive](source.tar.gz), via its included local Go proxy. [archive-manifest.json](archive-manifest.json) identifies the complete archive and every member. Toolchain/compiler binaries are prerequisites, not bundled artifacts.

[qualification.jsonl](qualification.jsonl) retains 334 records from the final normal/race runs: complete fixtures, provisions, model seeds/events, selected snapshots, process commands/binary hashes, barrier identities, exits, fault-control diagnostics, research comparisons and verification commands. Each extracted test record retains Mode, Package, Test and RecordIndex. [Raw normal](evidence/final-normal.go-test.jsonl.gz) and [raw race](evidence/final-race.go-test.jsonl.gz) Go JSON streams preserve ordering and original payloads. `python3 docs/foundational/phase1d/tools/assemble-evidence.py` reconstructs the JSONL from those streams. The actual collector used for the runs is retained under tools; its fixed local paths are provenance, not portable requirements. Collection-tool identities are recorded separately in the archive inventory; they are outside the Go source digest.

PIDs, ports, paths, timings and binary hashes are physical evidence and can differ on reproduction. The research records are deterministic under their stated identities. Full fixture definitions and unchanged checks, rather than byte equality of process metadata, define repeatability.

## Actual commands and results

Qualified on Go 1.27.3, darwin/arm64, CGO enabled, Darwin 27.2.0 and the recorded Apple clang toolchain. Exact platform/compiler/Go binary identities are in the source manifest and baseline platform record. Final test environment: `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d-go-build GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly`. Local-loopback execution required sandbox escalation; it did not use a remote service.

| Command | Actual result |
| --- | --- |
| `go test -json -count=1 ./...` | PASS, 5.994 s; 321 passing test/subtest events; no failures |
| `go test -json -count=1 -race ./...` | PASS, 82.578 s; same 321 events; actual controller children built with `-race` |
| `go vet ./...` | PASS, exit 0, 0.299 s |
| `gofmt -l` over all 51 Go files | PASS, no paths, 0.019 s |
| `git diff --check` | PASS, no output, 0.011 s |
| Focused slice tests and review probes | PASS; actual commands and RED/GREEN history in [implementation reports](reviews/task-4-implementation-report.md) and the other task reports |
| Original source qualification, before runtime changes | Phase 1: 15/15; Phase 1B: 20/20; Phase 1C: 32/32 exact historical records |
| Published historical archives, extracted and rerun | Same 15/20/32 exact records; [actual commands/results](evidence/baseline/published-archives.jsonl) |
| Final current-source research comparison, normal and race | Phase 1: 15 byte-identical; Phase 1B: 19 behavioral records equal; Phase 1C: 31 behavioral records equal |

The latter comparisons omit only each source_manifest record and the top-level SourceID when applicable. Changed manifests include the new dependency and comparison source. All thirteen pre-existing research Go files remain byte-identical to the base. Every other base file is unchanged except declared append-only additions to go.mod and the decisions document. The Phase 1C different-execution / wrongly ordered interruption-reactivation oracle regression remains present and passes. See [preservation](evidence/final-preservation.json), [current comparisons](evidence/final-research-comparisons.jsonl), and [historical reproduction](04-baseline-reproduction.md). The complete archive was extracted into a fresh directory and passed `go test -count=1 ./...` in 17.556 s using an initially empty module cache and only its bundled `file://` proxy. `go mod verify` and `go vet ./...` then passed; all 83 archive members remained unchanged. This independently checks package completeness, not another crash-model proof. [Archive reproduction](evidence/archive-reproduction.json) retains the actual environment and output.

## Process-crash evidence

Each final run emits 87 DURABLE records: 61 process records (including 16 SIGKILL exits) and 26 scenario/input records. The 16 include correct executions and deliberate faulty variants; they are not 16 independent safety proofs. A controller uses an inherited pipe to wait for an exact scenario/start-token/PID/store/boundary/phase/subject/generation/epoch/sequence/intent/authorization acknowledgment, checks retained facts, kills that process and verifies its signal exit. Timeouts fail stuck tests; no sleep determines a claimed boundary. Failpoints only expose and pause normal persistence paths.

| Crash window / executable fixture | Surviving evidence and allowed result |
| --- | --- |
| `TestAutomaticCLINestedCommitCrashBoundaries/execution.staged` | Writes staged inside the execution transaction, no COMMIT; kill rolls back projection, history and intent together. No effect escaped. |
| Same test, `execution.committed` | COMMIT acknowledged before dispatch; whole transition and pending Retry/budget survive. Recovery dispatches committed work. |
| `TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash/authorized`, plus `TestAutomaticExternalAuthorizationCrash` | Authorization retained separately; no known result yet. Original committed intent remains the authority for recovery. |
| `TestAutomaticApplicationMissingReceiptProfiles` | Separate resource acknowledges its own committed mutation; executor pauses before receipt insertion and dies. Resource survives. Idempotent recovery returns the original outcome with one mutation; non-idempotent recovery leaves unknown evidence and does not blindly resend. |
| `TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash/accepted` | Target Retry result and worker generation 2 committed; supervisor result unconsumed. New process retains inner generation 1, obligation, pending ID and spent=1; original result is consumed and actual successor computation completes. |
| `TestRejectedResultCrashAndLostRejectionControl/lost-rejection=false` | Original operation rejected while worker is active under valid matching authority. Worker later commits failure; reopen/retransmission returns the identical rejection. A distinct authorized operation creates worker generation 2, which remains open at sequence 0 waiting for ordinary input. Actual successor computation is established by the accepted nested crash case and the separate conformance rejection extension below. |
| `TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash/terminal` | Terminal outcome and notification obligation committed, delivery absent; reopening delivers that frozen result once logically. |
| Same test, `second-recovery-crash` | Second process is killed after renewed activation with the same pending operation; third process continues same obligation with a later epoch and no extra budget charge. |

The normal accepted-case snapshot records process 94345 stopped at `lifecycle.decided/accepted`, inner generation 1 / epoch 2 / sequence 7, operation `["inner",1,7,0]`, obligation `worker-obligation`, spent=1 and that exact pending ID. The reopened process finishes inner generation 1 / epoch 3 / sequence 16 with spent=1 and pending cleared. Worker generation 2 advances from sequence 0 to sequence 3 and succeeds using the same artifact as failed generation 1. The original single lifecycle-results row is unchanged. These are actual fixture identities, not universal operation names.

A crash precisely inside an unacknowledged SQLite COMMIT is not selected by this harness. Under the contract either whole outcome is permitted, distinguished by committed sequence/history/intents. No partial transition is permitted. These acknowledged before/after tests qualify process death under local SQLite/filesystem synchronization assumptions, not power loss or disk failure.

The ordinary nested fixture also runs with renamed subjects, reaches 26 execution commits/seven deliveries, and actually executes three worker-generation-2 instructions. Its worker artifact is identical across generations; explicit ordinary messages select fail versus success. The outer reads only public inner outcomes. Unit contracts additionally cover bounded exhaustion, stale activation, retry/cancel/retire ordering, source closure before/after authorization, late old results, stable service binding across real Retry, and corruption rejection before epoch renewal. No scheduler branch recognizes those roles.

The rejected-case target change is disclosed controller setup through ordinary input/worker computation using public store operations, while requester delivery is held. It does not claim an ordinary supervisor decided that test-environment action. Most end-to-end cases run the actual automatic CLI scheduler without issuing local steps from the controller.

## Fault controls and retained counterexamples

All three test-only faults are rejected by the unchanged `checkRecords` outside runtime policy:

| Fault | Non-vacuous exposure | Actual monitor diagnostic |
| --- | --- | --- |
| Lose retained rejection after restart | Original request has valid grant/revision/generation; target now eligible; deletion lets that same operation accept | `retained lifecycle_results row changed or lost` |
| Reset recovered budget/pending state | Mutate same run after validated Open, then call actual Run; recovered projection conflicts with retained history | `projection disagrees with committed ordinary history` |
| Dispatch before intent commit | Test-only SQL trigger sends real HTTP inside an uncommitted local transaction; endpoint commits, controller kills and local writes roll back | `endpoint application lacks committed authorized intent` |

The budget-control result does not claim that a duplicate generation actually occurred: the ordinary wait cannot consume after its pending ID is reset, and the history mismatch exposes the fault. Fault plumbing lives in test files, not the production failpoint observer. The speculative variant deliberately violates the dispatch boundary; the corresponding corrected scenario makes zero endpoint applications.

Three legitimate development counterexamples are retained with complete pre-fix sources, original probes and observed failing output under [counterexamples](counterexamples): output amplification past input/state limits; input shadowing automatic transport IDs; and a crash-controller acknowledgment that accepted the wrong generation/epoch/operation. The first two were found by independent scoped review. The third was found by the parent and then independently checked after correction. It was an oracle defect, not an observed runtime safety violation. The corrected model retains the original inputs and adds an explicit configurable output limit, trusted admission domains, and exact barrier identity checks. No valid input was removed to improve the evidence.

## Independent reference comparison

Six fixtures use preserved research `step`/`inspect` and monitors on one side, new runtime public APIs on the other. They share no reducer or recovery-policy oracle. Full original seeds, applied setup/actions and expected/actual public projections are retained in CONFORMANCE records.

| Fixture | Model events / compared checkpoints | Claim |
| --- | --- | --- |
| Accepted loss and successor | 28 / 12 | Retained accepted result, one decision, real successor work |
| Interrupted committed request | 37 / 13 | Same-execution ordered interruption/reactivation, stale authority denied, pending/budget preserved |
| Rejection survives failure | 22 / 12 | Original rejection immutable after target change; separately labeled new runtime request may succeed |
| Observation without lifecycle rights | 32 / 11 | Source authorization distinct from target authority; observation alone grants none |
| External generations and late receipt | 43 / 30 | Two execution keys, two mutations; old receipt cannot satisfy generation 2's eligible receipt wait |
| Closure before external authorization | 15 / 3 | Closed-source denial and no endpoint contact |

These are selected transition/projection comparisons and successful reachability witnesses, not exhaustive runtime schedule exploration. The original finite model searches retain their own bounds, counts and exclusions; no larger bound is inferred from these examples.

Representation differences are explicit: model Receive atomically clears pending, while runtime Receive plus Set is two commits. The consumed-but-pending runtime intermediate is retained as **not equivalent and not a stutter**, and reopening there resumes its residual instruction. Open renews all active epochs while model Reactivate selects one run. External J adds explicit runtime fail/wait inputs; its endpoint is in-process with separate durable state, so process isolation comes from the controller suite. Its late closed-run receipt leaves a retained unconsumed runtime observation where the model only settles evidence. Only named common boundaries are compared. Replay uses the runtime reducer itself and is a consistency check, not this independent comparison.

## Evidence-to-claim boundary

| Claim class | Evidence | Limit |
| --- | --- | --- |
| Durable ordinary recovery | Actual automatic CLI, acknowledged SIGKILL, reopen from same store, retained snapshots and successor computation | Selected finite executions; one active host/process |
| Atomic local boundaries | Transaction rollback/commit, corruption tests, independent ledger checks | Depends on SQLite and declared filesystem assumptions |
| Historical replay cannot dispatch | Read-only/query-only opener, no dispatcher in replay, ledger/resource before-after equality | Live recovery is separately allowed to dispatch unresolved work |
| Source versus target authorization | Separate ledgers, closure tests and independent comparison | Static grants; no delegation or ownership transfer |
| Exactly-once logical decision identity | Immutable result retransmission, duplicate admission, negative controls | Not exactly-once arbitrary business mutation |
| Resource ambiguity | Independent process/database, missing receipt, non-idempotent repetition experiment, two-generation J comparison | No business deduplication, compensation or outcome inference from failure |
| Runtime safety checks | Concrete regression inputs, subprocess scenarios and three detected faulty variants | Not exhaustive safety for arbitrary programs or schedules |
| Progress | Independent peer and successful nested recovery witnesses | No fairness or eventual-recovery guarantee |
| Review | Five scoped reviews plus complete-source independent final review | Review is evidence, not proof |

Remaining scope limits: one PC per run; bounded finite steps and checked int64 data; explicit configurable byte/work limits; no history garbage collection; startup/replay scans all history; no schema/program migration; trusted local transport provenance; no cross-store business-key namespace; no multi-host operation, ownership transfer, dynamic subjects, fresh-supervisor child reassociation, cleanup custodians, providers or deployment. At generation capacity, Retry returns operational ErrLimit without a committed lifecycle result; retained intent/budget/authorization remain visible. Non-idempotent uncertainty can remain indefinitely. None of these limitations prevents the qualified same-execution nested recovery case.
