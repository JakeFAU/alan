# P7 whole-branch independent final review

Reviewed 2026-11-14. Base `9db473bae28e5b806c3ccdc7e2e134c1d0b38f27`;
head `cb13497276fb6bc5256d833484ca4e7d59c33fce`, tree
`cca84dc60abec7e353a3186fa5e152d9e00dd882`.

## Verdict

**Specification: PASS for the frozen negative-result path. Quality: APPROVE
for draft delivery.** No blocking, high, medium, or minor finding remains from
this review. No production candidate is delivered. The failed advance screen
justifies skipping Task 2 and the candidate-specific qualification campaigns.
This verdict does not approve a future write-order change, merge or release.

The review examined actual apparatus and decisive raw evidence independently;
prior task approvals were supporting context, not substitutes for those checks.
The final review copy and its subsequent documentation bridge require the
planned narrow delta review before publication.

## Strengths and evidence

1. **Scope and executable identity are intact.** The complete base-to-head
   Git delta contains only README navigation and the new
   `docs/evidence/retained-query-scaling/` directory. Runtime, public API,
   original tests, dependencies, nested consumers, schema, indexes, ownership,
   retries and BoundedSeries retain baseline bytes. The five executable
   apparatus inputs remain identical to corrected freeze `ca65abb857a75f3677ce88a74cbeb94a2251450b`.
   Independently verified the exact changed-path sets and every declared blob
   digest for `data/source-bridge.json` at `c19785d`,
   `data/delivery-bridge.json` at `c1c475c`, and `data/review-bridge.json`
   at `cb13497`. Historical bridge digests are correctly interpreted at their
   named commits, including the later qualification wording change.
   `git diff --check 9db473b cb13497` passed.

2. **The scan mechanism has a discriminating source and measurement chain.**
   Read `internal/executor/store.go:821` through the owned commit boundary,
   including the original adjacent writes at lines 906–913. The disposable
   transform in `measure.py:72` swaps unique adjacent blocks; `freeze` at
   line 160 binds the exact baseline, checks identical exports, and permits
   only that one changed file. The pinned SQLite `FkCounter`/`FkIfZero`
   execution and `fkScanChildren` generator agree with the retained EXPLAIN
   and mixed-subject/generation controls in `fk_control_test.go.txt`.
   The controls check FK settings, both orders' retained rows, NULL selection
   and a deliberately failed update followed by rollback. They are properly
   described as SQL controls, without crash-safety qualification.

   Independently inspected the four raw B8/B256 counter reports. The full
   registry differences contain only history scan/VM reductions, consumption
   update VM reductions, and their corresponding scope/total aggregates.
   B256 Execute full-scan work falls from 227,970 to 97,665; all 130,305
   removed scan steps belong to history insertion. SQL executions and row
   returns do not change. Other lifecycle stages' scan counts are identical.
   The remaining 65,280 result-lookup and 32,385 charge-count scans are neither
   hidden nor optimized. References: `measure.py:247` through
   `compare_counter_pair`; `results.md`, mechanism and counter table;
   `data/counters.json`; local `frozen-counters/gates/*/stdout.log`.

3. **The stopping decision follows the frozen rule and raw stock timing.**
   `measure.py:62–69` schedules one warmup per arm and six adjacent balanced
   AB/BA pairs per bound. Lines 107–132 bind rows to their source/binary and
   implement the predeclared 1% / five-of-six / savings-above-MAD screen.
   Both stock binaries are built before serial sampling. Failed samples are
   retained by the reused P6 capture path and rejected before a successful
   campaign result. The two generated stock drivers have identical bytes.

   Matched the committed timing rows to local pilot rows, checked all 84
   primary raw stdout digests and successful workload cardinalities, and
   independently recalculated every bound's paired Execute statistics from
   raw stage values. B256 savings are +35.268250, +1.711749, +23.848917,
   −14.597666, −10.144458 and −28.664668 ms. The paired median is −0.326281254%,
   three pairs are faster, median saving is −4.2163545 ms and MAD is
   17.4148125 ms. All three advance conditions fail. B128 is +0.004245040%.
   The published results agree. There is no substituted threshold, enlarged
   bound, second optimization or lifecycle-only rationale rescuing the result.
   References: `contract.md`, Measurement and decision;
   `results.md`, Uninstrumented paired pilot; `data/timing-rows.jsonl`;
   local `pilot/gates/timing-*/stdout.log`.

4. **Work attribution stays separate from application latency.** Read the
   clock template, its tests, and generated SQLite wrappers at all five
   actual C step sites, including the two eager-first-step branches. The
   Execute start/stop hooks exclude the registry snapshots; buffered returns
   do not introduce extra measured steps. Exact prepared SQL classification,
   clock-off and empty controls match the stated boundary. Independently
   checked all 18 raw clock output hashes and aggregates; B256 history
   on-mode median is 5.854341 ms, empty median 0.024032 ms, and off totals
   are zero. The report explicitly includes cgo/preemption, excludes prepare,
   marshaling and commit, and does not equate these aggregates with removable
   elapsed time. Profile scope, allocation volume, cumulative RSS and
   separate lifecycle/reopen/Replay costs are qualified appropriately.
   References: `step_clock.go.txt:10–81`, `step_clock_test.go.txt`,
   `measure.py:83–104`, `results.md`, clock and resource tables.

5. **Outcome, authority and accounting checks are more than matching totals.**
   Read the generated stock driver and unchanged P6 transform/helpers. Each
   position verifies progress and one new direct adapter contact. The final
   proof checks distinct roots, effects, request keys, observations and
   result digests; source run and selection provenance; admission/settlement;
   one effect charge on every root; and each protected approval's predecessor,
   prepared/result digests, target and selected sequence. Full Run, typed
   inspection, reopen and Replay remain explicit, separate stages. The
   original evidence is compared through reopen and root evidence through
   Replay. B256 raw rows have 256 selected/completed/charged positions and
   contacts, 255 protected approvals and 1,023 replay history entries.
   References: unchanged `progress-scaling/execute.go.txt`,
   `progress-scaling/completed.go.txt`, and generated
   `pilot/baseline/driver/main.go:160–218,340–451` in the local workspace.
   This is successful-workload evidence within each fresh namespace, not a
   same-seed failure/recovery equivalence experiment.

6. **Reproduction and qualification are honestly bounded.** Read the actual
   detached-clone reproduction record and independently matched selected
   raw command/output hashes for Python controls, clock preflight, stock
   smoke, build, vet, module verification, both failed Staticcheck attempts
   and both successful cache-local retries. The Python log confirms seven
   passing tests; initial lint output contains cache-write errors and the
   successful retries have empty diagnostics. Recounted original ordinary
   raw events: the sandboxed run has 27 failed tests across nine failed
   packages; the loopback-enabled run has 1,420 passing test events,
   18 passing packages, three packages without tests and four skipped
   optional historical-reader subtests. The source bridge justifies reuse
   at unchanged runtime/test bytes. `qualification.md:79–96` clearly lists
   the omitted campaigns and distinguishes integration-tag Staticcheck from
   integration execution. The corrected cache wording accurately identifies
   the reproduction-local location.

   Public evidence is compact synthetic-workload data, source/build digests,
   command metadata and profile text. Inspected environment fields contain
   tool/cache/local filesystem paths rather than credential variables. No
   raw SQLite stores, profile binaries or archives are added by the branch;
   their local-only status is explicit. Early controller/packaging terminal
   logs that are unavailable are disclosed, rather than represented as
   complete custody. References: `qualification.md:35–45,48–105`,
   `results.md`, Attempt history, and `data/reproduction.json`.

## Behaviors declined for judgment

- **Discarded-overlay recovery and compatibility:** no claim about staged-write
  SIGKILL cuts, failed deferred-FK COMMIT cleanup, concurrent reader cuts,
  stale/corrupt historical stores, historical reader binaries, repeated crash
  recovery, or custom trigger semantics. Those candidate campaigns did not
  run. Successful completion/reopen and SQL rollback controls cannot replace
  them; the runtime change was not shipped.
- **General performance or absence of a small gain:** six pairs on one host
  support this frozen engineering stop decision, not a population confidence
  statement, cross-platform result or proof that another workload cannot
  benefit. Diagnostic work reduction is established only on the pinned
  workload/access paths. The declared qualifications preserve this boundary.
- **Unobserved host activity and missing early terminal records:** the campaign
  ledger cannot establish absence of unrelated background work, and narrative
  early corrections do not restore unavailable terminal logs. Neither is
  used to turn the negative pilot into a speedup claim.
- **General system or remote exactly-once guarantees:** the preserved existing
  runtime and this synthetic direct-adapter experiment do not establish new
  authority, arbitrary-context correctness, distributed recovery or remote
  effect guarantees.
- **Publication status and future changes:** this review approves the stated
  local head for draft delivery. It does not assert that a push/PR exists,
  that public raw archives were published, or that a later source delta is
  covered by this verdict.

No suites, builds or benchmark campaigns were rerun during this review. I did
not repeat the earlier exhaustive 327-record custody audit. This report is
its only filesystem write.
