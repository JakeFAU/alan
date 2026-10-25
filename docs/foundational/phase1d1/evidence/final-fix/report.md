# Final review P2 fix — DONE / frozen

The report now distinguishes an authorized unknown physical outcome from a valid durable receipt awaiting separate local settlement. A valid receipt removes only the UnknownExternal classification. The failed settlement still returns its explicit error, Quiescent remains false, and the intent/delivery remain unresolved/pending. Existing durable outcome validation is used; malformed or incorrectly bound receipt evidence surfaces its validation error rather than being called known or Unknown.

## Exact scope

Only two source paths changed relative to reviewed source ee144:

- `internal/executor/runner.go`: inside `finishRun`, an allowed unsettled external attempt calls the existing `externalOutcome(ctx, s.conn, id)` validator. Only sql.ErrNoRows appends UnknownExternal. Other validation failures are joined with the original run error. All remaining report construction and runtime behavior are unchanged.
- New `internal/executor/async_reporting_test.go`: permanent known-receipt/failed-settlement regression derived from the independent probe, strengthened to require unresolved intent and pending delivery retention; verifies non-idempotent reopen settles/consumes without another endpoint contact and replay never dispatches. Two malformed-receipt subcases require the existing syntax/binding validation error to be exposed by finishRun.

No docs, dependency, schema, transaction, scheduling, transport or other cleanup changes. No commits or subagents. Comparison against the pre-fix per-file manifest confirms runner.go is the only existing source whose bytes changed.

## Source identities and retained RED

Length-framed aggregate algorithm matches the parent manifest: SHA256 over sorted paths, uint64BE path byte length, path bytes, uint64BE content byte length, content bytes; go.mod/go.sum plus all cmd/internal/research Go files.

- Reviewed source before: `ee144b08e6abac10c704c990e83a80314326a1eac3956bc5ecf9489f8f0afd5a` (57 files), independently recomputed against preserved original files.
- Corrected frozen source: `9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63` (58 files).
- Patch against ee144 bytes: `/tmp/phase1d1-final-fix.diff`, SHA256 `fd1ec8e209cd117047b624152847745481947f429ca3cb5e9101efbe59306bb7`.
- Per-file before manifest: `/tmp/phase1d1-final-fix-before-manifest.json`.
- Final per-file/log/diff/source manifest: `/tmp/phase1d1-final-fix-manifest.json`.
- Complete compact pre-fix source plus new regression archive: `/tmp/phase1d1-final-fix-before.tar.gz`, SHA256 `5707a3159e35cbc73abe58a6944463eb2817846bfc30459e8255e8ca2c6e49d4`; per-member hashes `/tmp/phase1d1-final-fix-red-manifest.json`. This archive was written after the new regression failed and before the runtime edit. It is compilable on the same pinned dependency environment; parent owns the existing bundled offline proxy.
- Original independent RED and exact input retained unchanged: `/tmp/phase1d1-final-fix-independent-original-red.log`, `/tmp/phase1d1-final-fix-independent-original-test.go`.
- New permanent regression behavioral RED: `/tmp/phase1d1-final-fix-red.log`. Known applied receipt was incorrectly Unknown; both malformed receipt cases suppressed validation. All failed against unchanged ee144 runtime before editing runner.go.

## Exact validation

Environment for Go commands: `GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d1-go-build GOPROXY=off GOSUMDB=off`. Local HTTP tests used authorized sandbox escalation.

1. Pre-fix RED:

   `go test ./internal/executor -run '^TestAsync(KnownReceipt|RunReportRejects)' -count=1 -v`

   Go test exit1 expected, preserved `/tmp/phase1d1-final-fix-red.log`.

2. Focused normal GREEN:

   `go test ./internal/executor -run '^TestAsync(KnownReceipt|RunReportRejects|LateReceipt|AuthorizedTransport|ReceiptPersistenceFailure)' -count=5 -timeout=60s -v`

   Exit0, `/tmp/phase1d1-final-fix-focused-normal.log` (0.364s). Covers the amended report path, missing-receipt uncertainty/persistence failure, late authentic evidence, forged/old-epoch correlation, late source closure, malformed receipt evidence and non-idempotent settlement recovery.

3. Same focused group under race detection:

   `go test -race ./internal/executor -run '^TestAsync(KnownReceipt|RunReportRejects|LateReceipt|AuthorizedTransport|ReceiptPersistenceFailure)' -count=5 -timeout=60s -v`

   Exit0, `/tmp/phase1d1-final-fix-focused-race.log` (2.146s).

4. Unchanged independent expected-report probe GREEN:

   `go test -race -overlay=/tmp/phase1d1-final-review/red-overlay.json -count=5 -timeout=60s -run '^TestFinalReviewKnownReceipt' -v ./internal/executor`

   Exit0, `/tmp/phase1d1-final-fix-independent-green.log` (1.453s). The overlay input bytes equal the retained original independent test; no assertion changed. Each run reports one endpoint contact across failure, reopen consumption and replay.

5. `git diff --check -- internal/executor/runner.go` exit0; `gofmt -l internal/executor/runner.go internal/executor/async_reporting_test.go` empty.

Full/race qualification, refreshed docs/manifest/archive and publication are intentionally parent-owned for the new source. No unresolved item remains in this fix wave. Source is frozen pending parent validation and targeted independent re-review.
