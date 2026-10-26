# Task 4 independent adversarial review — frozen original review

Reviewed base: `18a751c58560360945afb69b4e49fb469afacc4f`, branch `codex/phase1e-authoring-agent`, plus the Task 4 uncommitted source identified by `evidence/task-4/frozen-source-sha256.json`. All 17 source/report hashes matched before review. No implementation or test source was changed, and no commit was made.

## Verdicts

**Specification verdict: changes required for coherent operator reporting.** The ordinary nested application, public composition boundary, explicit budgets, model proposal authority, exact citation checks, retained configuration/corpus and offline CLI replay satisfy the inspected Task 4 design. The application inspection API can expose mutually inconsistent committed states in one View (R1).

**Code-quality verdict: changes required (one medium-severity correctness defect).** The defect is a cross-transaction assembly error in a public operator API, not a runtime reduction or transport race. No additional blocking quality findings were established. Long compact builder expressions are harder to scan but are not reported as a separate blocker.

## R1 — Medium: assemble inspection summaries from one coherent snapshot

Location: `examples/evidence/store.go:102–111` (two reads); `:113–124` (newer public result); `:133–169` (older counts/results/unknowns).

`Inspect` first obtains a coherent `runtime.Inspect` snapshot, then separately calls `runtime.Replay`, whose read transaction can observe later commits. `View.Snapshot`, logical counts, Results, Attempts and Unknown come from the first transaction, while `View.Runs` and public Result come from the second. A concurrently progressing runner can therefore return one View that never existed as a committed application state. This is a logical race even when the Go race detector passes.

The preserved independent probe ran the real controlled Responses serializer/parser with an ordinary clarification workflow, called `evidence.Run` concurrently with `evidence.Inspect`, and compared returned Runs against the supplied Snapshot. The material retained counterexample contains:

- Caller Snapshot: generation 1, sequence 1, PC 1, status open, no disposition or answer (before Receive).
- Caller View.Runs: generation 1, sequence 2, PC 2, status open, disposition clarification, answer `Which guarantee?` (after Receive).
- View.Result already publishes that clarification while View.Snapshot still contains the earlier caller projection.
- Assistant also differs: Snapshot sequence 28 / PC 104, View.Runs sequence 29 / PC 106.
- Independently observed provider contacts: 1. Logical model requests: 1; no lookup. This particular retained case demonstrates public-result/state inconsistency, not a request-count mismatch. The same split-read structure exposes counts to skew, but that separate manifestation is not claimed as observed.

Original broad comparison also detected assistant PC 0 versus PC 1 during startup. Both original probes and logs are retained; the refined counterexample is the material one above. The full JSON includes the exact request ID, namespace, retained artifacts, sequences, projections, attempt and provider result.

Expected behavior: public Result, ordinary Runs, intent counts, attempts, results, unknown IDs and Snapshot describe the same committed read view. Derive the summary from one coherent snapshot, or introduce a public combined read that validates/replays and inspects within the same transaction. Do not merely hide Snapshot or suppress the failing comparison. Add a concurrent-progress regression asserting matching identities, sequences/projections, public result and logical accounting.

Reproduction (from repository root, controlled loopback only):

```sh
cp .superpowers/sdd/00-plan/evidence/task-4/independent-review/inspection-probe.go.txt /tmp/alan-task4-review-probe.go
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off \
  go run /tmp/alan-task4-review-probe.go
```

The bounded stress probe observes normal concurrent progress and contains no sleep, runtime mutation, provider bypass or production failpoint. Scheduling can affect which interleaving is seen; the preserved JSON/log proves the original failure independently of a future rerun. The probe exits normally after printing `MISMATCH` to preserve evidence; absence of a mismatch in its bound panics. Its temporary output path is `/tmp/alan-task4-inconsistent-view.json`.

## Independent checks and positive findings

Read the exact requirements, plan, authoring/effects contract, Task 4 brief, integration notes, implementation report, complete reviewed application/CLI/tests/docs, review diff, and frozen source manifest. Checked the underlying public runtime read transactions and lookup/provider boundary where needed.

- `evidence.New` constructs ordinary planner/synthesizer and retrieval worker artifacts with static capability names and grants. The caller consumes the returned Port. The external module builds two helpers without internal imports, numeric PCs, SQL, or child-state access; its observed path is clarification, accurately documented.
- The model's tool name is compared to literal lookup before ordinary Send; capability and endpoint are deployment/static values. The worker emits a separately authorized lookup intent. No hidden adapter tool loop was found.
- Recorded model validity gates branch selection. Synthesis receives the original question and retained document/version/snippet/start/end. Final ordinary comparisons check all five citation fields plus a nonempty answer before publishing answer.
- Per-question counters are initialized once and incremented atomically with Request; pending IDs are ordinary state. Budget 0 and budget 1 branches are finite and complete the worker skip protocol where appropriate. Run reopens artifacts rather than recompiling.
- Provision retains corpus bytes; CLI init snapshots the explicitly selected file once. The actual CLI subprocess test removes that file before run and later stops the provider/removes credentials for replay. Replay constructs no adapter registry.
- The report correctly distinguishes logical requests, allowed reservations, denied decisions, independently observed HTTP contacts and optional actual usage. It does not claim reservations prove remote contact or missing usage is zero. Unknown no-repeat recovery is covered by retained focused tests and source.
- Independently added five boundary probes through the real HTTP provider adapter: synthesis refusal, synthesis incomplete, synthesis malformed typed output, empty lookup query, and explicitly empty corpus. All passed under `-race`; all programs terminated, and offline replay passed after stopping the provider. Synthesis failures produced invalid_result with 2 model/1 lookup/3 reservations/2 contacts; empty query produced invalid_result with 1 model/0 lookup/1 reservation/1 contact; empty corpus produced no_match with 1 model/1 lookup/2 reservations/1 contact.

Independent probe command:

```sh
cp .superpowers/sdd/00-plan/evidence/task-4/independent-review/boundaries_test.go.txt /tmp/alan-task4-boundaries_test.go
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1e-go-build GOPROXY=off GOSUMDB=off \
  go test -count=1 -race -v /tmp/alan-task4-boundaries_test.go
```

Result: PASS, five subtests, 9.752s command test duration. Original source and output are preserved. No broad suite was needlessly repeated; the provided final suite/race evidence was read as implementer evidence rather than represented as an independent rerun.

## Preserved evidence and review limits

`evidence/task-4/independent-review/` contains the original and refined inspection probes, original logs, complete inconsistent View, independently written boundary probe and race log, reviewed-source archive, and original reviewed-source SHA-256 manifest. `artifact-sha256.json` fingerprints these review artifacts. The archive contains the reviewed Task 4 source/report and governing documents; its committed dependencies are identified by the base above. Parent will preserve the complete phase source before fixes.

No live model ID or credential was configured. Live gate remains NOT RUN. Controlled responses establish protocol/application behavior, not live inference quality or substantive truth of every answer sentence. Task 5 acknowledged SIGKILL/provider-contact qualification and negative controls remain pending; their absence is not reported as a Task 4 completion claim or as a new Task 4 defect. This review does not requalify all compiler/provider/runtime internals or establish unbounded fairness, production readiness, or power-loss durability.

This report is frozen before fixes. Follow-up verification should be appended in a separate review addendum so the original finding, source identity and inputs remain intact.
