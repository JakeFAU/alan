# Independent cancellation ownership review

Runtime/spec verdict: PASS for the scoped ownership correction. Qualification/code-quality verdict: CHANGES REQUESTED for one P2 test-oracle gap below. No runtime defect found in this bounded review; the passing runtime traces remain valid observations. This review does not reopen the separately closed FR1 provider review or qualify the whole branch.

Reviewed frozen source `6e4ca5841098ed674ce3ff3f98bbc9c2be389602f6d491d54f1389db69ec317e` (131 files). All source-manifest hashes matched before and after review. All six retained correction-verification log hashes matched. The complete `adapters_test.go`, including `TestAdapterCancellationAccountsReceivedCandidate`, is byte-identical to the before-source archive. Runtime/tests/index were not edited; probes used Go overlays and an extracted before-source under `/tmp`.

## Finding CR1 — P2: bind the new dequeue acknowledgment to durable epoch and sequence

Location: `internal/evidencetest/process_test.go:603-606` in `TestCancellationAfterDequeueRetainsRealModelCandidate`.

The selected filter specifies boundary, phase, and subject; epoch and sequence are intentionally wildcards. `ackBoundary`/`validateAcknowledgment` therefore validate only positive epoch/sequence, not their equality with the durable operation. The later result-ID and authorization-ID assertions also do not check either field. Unlike the existing recorded-boundary crash tests, this new test never calls `validateDurableBoundary`.

Confirmed with the actual CLI, original wildcard filter, real store, and controlled provider: an overlay changes only the `runner.received` event to `Epoch += 7; Sequence += 9`. The entire new cancellation test still passes, including settlement, no-new-work assertions and recovery. This is an actual monitor counterexample, not evidence that cancellation runtime accounting fails.

The preserved unmutated stopped input has PID 60461, epoch 2, sequence 15, intent `["assistant",1,15,0]` and its retained authorization. Substitution against that exact input independently confirms that epoch 9 / sequence 15 and epoch 2 / sequence 24 both pass the current wildcard monitor but fail the existing durable checker. The original input passes both. The actual mutant CLI run acknowledged epoch 9 / sequence 24 while its durable intent remained sequence 15.

Action: immediately after `before := inspect(t, db)`, require `validateDurableBoundary(before.Snapshot, ack)` to succeed. Add/retain a regression covering the actual wildcard filter and the legitimate stopped input, plus separate epoch and sequence substitutions; do not make the filter fully specified just to cause synthetic rejection. Preserve original passing traces and this counterexample. A temporary overlay adding this one assertion accepts the unchanged runtime and rejects the actual CLI mutant with `boundary not bound to durable source authority`.

Evidence in `evidence/cancellation-review/` adjacent to this report:
- `strengthened-original-cli-stopped-input.json`: exact valid acknowledgment, actual wildcard filters, full before/after snapshots, provider contacts.
- `mutant-cli-stopped-input.json` and `mutant-cli.log`: wrong epoch/sequence acknowledgment accepted by the existing complete CLI test.
- `actual-input-substitution.log` and `process-substitution.go.txt`: independent substitutions of the original actual input; original accepted, both mismatches detected by the durable checker.
- `strengthened-original-cli.log` / `strengthened-mutant-cli.log`: suggested assertion accepts the original (exit 0) and rejects the mutant (exit 1).
- `runner.original.go.txt`, `process.original.go.txt`, `regression.original.go.txt`: unchanged review inputs; overlay files retain precise probe deltas.

## Runtime and contract checks

Both live receives retire exactly one active slot and immediately call the same helper. The helper retains the original completion identity, reports the observation-only handoff with the original context, then uses `WithTimeout(WithoutCancel(ctx), 5s)` solely for accounting. It preserves observer, accounting and caller errors with `errors.Join`. Returning cancellation after accounting prevents further owner-loop scheduling; the detached context never enters preparation, authorization, worker transport or ordinary commits. Deferred shutdown still joins workers and drains the remaining active slots exactly once. `Close` waits for Run to relinquish ownership before closing SQLite.

Receipt recording and settlement remain separate commits. Candidate correlation validation still occurs inside accounting. A failure before recording can remain Unknown; a recorded candidate whose settlement fails stays known/pending. No helper retry/requeue or implicit inference repeat was introduced. Physical cancellation does not alter logical run status, and existing settlement semantics retain late evidence without reopening closed runs.

Independent race probes, beyond repeating the implementer's suite, all passed:
1. A non-cancellation observer error at the dequeue boundary still records and settles the owned candidate, returns the observer error, and advances no ordinary steps.
2. A real five-second accounting deadline is allowed to expire after result recording while the caller is canceled. Run preserves both cancellation and deadline errors; the result is known/pending, not Unknown. Offline replay succeeds; recovery settles/consumes without another adapter execution. Observed deadline elapsed 5.006655583s.
3. A mismatched candidate key combined with a dequeue observer error and cancellation preserves all three errors, records no result/settlement, reports Unknown, and does not automatically resend on recovery.

These are finite checks, not a hard wall-time shutdown guarantee. Workers and observers remain trusted/cooperative; existing mutex/SQLite commit and join behavior is not newly made deadline-interruptible. The helper's separate accounting timeout plus deferred cleanup budget are accurately disclosed rather than claimed as one global five-second shutdown bound.

## Red controls and evidence integrity

Independently reconstructed the red source from the retained `before-source.tar.gz`, applied `red-observer-and-cli.patch`, and installed the preserved `red-cancellation-regression-test.go.txt`. The red command failed as intended: both adapter/fixture dequeue cases lost evidence, the between-result-and-settlement cancellation failed, and the real CLI owner-cancellation case lost its received candidate. The observational seam is shared with the fix; it does not manufacture results or relax original assertions. This supports the ownership diagnosis.

`commands.json` contains exact commands, exit codes and log digests. `identity.json` and `post-review-identity.json` record unchanged source and original-regression verification. Probe files are retained here and under `/tmp/alan-cancellation-review`; overlays contain their original absolute paths. Reproduction of the red archive also remains at `/tmp/alan-cancellation-review/red-reproduction`. No credentials or live-provider calls were used.

Only CR1 requires correction. Re-review can be limited to the actual wildcard acknowledgment/durable binding and its preserved positive/substitution controls, followed by source-matched final qualification managed by the parent.
