# Independent Go and lifetime review

## First candidate — REJECT

The candidate loses a recovered settlement error on each of three public service paths. A later same-owner Q succeeds despite the earlier error. This violates the frozen rule that the first accounting failure for an originating generation remains visible to all covering waits for the life of that owner. The defect requires genuine available evidence, a service attempt and a persistence error. A missing world response, a first-permission denial and a waiter's cancelled context have different meanings.

I reviewed the following file SHA-256 identities on contract commit `656f84b58424b3e2a62ce9a9eeb6bd72f84a5784`.

| File | SHA-256 |
|---|---|
| internal/executor/adapters.go | `795600fafe193a1d17e7635eb3344ed6291bec73c943b53d2b1cf74709bf8183` |
| internal/executor/dispatch.go | `3c753045644829439d0696d303cddcc83f6d067c95ac32407820de58ae64e873` |
| internal/executor/external.go | `70db9eefac7bfc25f6451d466db70cd5d3dd882e47173b055f129c1ae8436d09` |
| internal/executor/runner.go | `28936954d382d29e78d4c1af7f44a673ad5fc87f38e55ff067986e8be659e9b9` |
| internal/executor/store.go | `2e6239c2a3997528f0661ae2d7a6cca9114ed24739f42101a9d20adba6355103` |
| runtime/runtime.go | `d024d2e7c5c3077db6d602472b05b15447964c83506e3ef80ebc57545c8b4603` |

The public gate is present in both normal Run transports and synchronous direct raw dispatch. Its current-source and scope reads serialize with closure commits under Store.mu. Its final owner check and registration also hold ownerMu in the existing order. Each registered object has its own channel; the scheduler preserves it until normal/cleanup accounting, independently of physical-slot release. Direct work and Q have owner pins; Close prevents later pins, signals waiters and joins Run and pins before releasing the connection and flock. Scope matching walks the immutable caller bindings and includes a matching resource root, independent of the alias member's own terminal status. The defect is in accounting service outside these volatile physical-worker registrations.

### Witness

I took the first-owner setup of `TestPhase2BStickyAccountingAndRecoveredSettlement` unchanged. A real HTTP receipt is recorded, and a TEMP trigger rejects its initial settlement. Natural End revokes the source. The failure correctly reaches Q on that first owner. Close and fresh Open preserve the receipt with zero settlements. On the new connection I installed this independent fault:

```sql
CREATE TEMP TRIGGER reject_recovery BEFORE INSERT ON settlements
BEGIN SELECT RAISE(ABORT,'phase2b recovery settlement failure'); END
```

I attempted service separately by Q, DispatchExternal and Run in three fresh fixture instances. The first service call on the new owner returned the SQLite failure every time. I dropped only the TEMP trigger and called Q again on that same owner for the exact original source. Each second Q returned nil, created one settlement from that original receipt, and left the POST count at one. The prior same-owner failure had disappeared. These runs make no claim about an unobserved application.

The root cause is that `finishDispatch` stores errors only when a new opportunity token is being released. Existing raw or adapter results can be settled without another A. `settleRecordedDispatch` and the already-recorded branches in preparation return errors directly. The new owner never stores that genuine failure for the source. Correcting only the Q caller would leave both other routes vulnerable. A fix must distinguish a Q waiter's timeout from an actual accounting failure and preserve unknown outcomes without creating evidence.

I executed the following selection using Go 1.27.3, CGO, readonly cached modules and the loopback fixture. The first sandbox attempt failed to bind a loopback socket before the scientific test. The rerun with a loopback-capable environment reached all three falsifiers and exited 1.

```sh
go test -overlay /private/tmp/alan-phase2b-sticky-review/overlay.json ./internal/executor -run '^TestReviewerRecoveredService' -count=1 -v -timeout 90s
```

The baseline fixture `quiescence_test.go` has SHA-256 `0f469b0d0f42ace5787526054f758058d5db85ab5d8c0d37cdaa09b86b7af3e6`. The executed overlay has SHA-256 `e30dc9e6b32d2d99d46ce4ef74b9c7003183f17e420fcf28963b6cf09c4de792`. Its compressed unified diff `repro.patch.gz` has SHA-256 `b2a5a319aeee4bec7d3e4645779a89063cdd4d03befc5ce85f270041e253452f`. In each of `TestReviewerRecoveredServiceQ`, `TestReviewerRecoveredServiceDirect` and `TestReviewerRecoveredServiceRun` the first line reported `phase2b recovery settlement failure receipt 1 settlements 0`. The second reported `second Q <nil> receipt 1 settlements 1 POSTs 1`.

## Corrected candidate — APPROVE

The corrected files have SHA-256 `35e86498bb3daa8bfc4854c731c66e9e00c8a4375435f23c1db3ea82ccc194cd` for adapters.go, `f122162ac61fed0ddc9b7e7f4f43622f424e9a27437fcc05eaebafe8b3d3d9e4` for dispatch.go and `ee8d2d5a511f6ecfbdd50ef9ab5eb97a7c40ecc2b8033e8317d67d145c7ac8cd` for external.go. I verified runner.go, store.go and runtime.go still have the hashes in the first review. This approval covers the core Go implementation. The rejected case above remains in the record.

On these sources I ran the same three overlay tests under the race detector using the same pinned local environment. All three passed. Each fresh owner's first service encountered `phase2b recovery settlement failure`. After removal of the trigger the second Q on that same owner returned the same failure, with one genuine retained receipt, zero settlements, and one POST. I also ran the integration-tagged recovered accounting and source-mismatch selection. Its five service combinations and two mismatch cases passed. No process-level conclusions rely on these tests.

```sh
go test -race -overlay /private/tmp/alan-phase2b-sticky-review/overlay.json ./internal/executor -run '^(TestReviewerRecoveredService|TestPhase2BRecoveredAccounting|TestPhase2BRecoveredSourceMismatch)' -count=1 -v -timeout 90s
go test -race -tags integration ./internal/executor -run '^(TestPhase2BRecoveredAccounting|TestPhase2BRecoveredSourceMismatch)$' -count=1 -v -timeout 90s
```

The new helper rebinds the retained result to its original source, attempts settlement, and stores the first genuine failure in the same Store.mu critical section that guards Q's inspection. A concurrent Q cannot observe a gap between the failed write and publication of its error. Concurrent servicing of a previously found record can still succeed later, while the same owner's waits keep returning the historical error. Raw direct dispatch, automatic raw/adapter handling and the host wait all use the shared path for surviving evidence. The transaction lifetime detaches from caller cancellation in `beginOwnedTx`; checked operations and commit still observe the service context. Cancelled callers receive a context error and can leave the durable result for another service attempt. A raw attempt with no response and an allowed adapter attempt with no candidate are withheld under the existing route rules. A synthesized original raw denial has no external receipt. Genuine pending adapter results are serviced with their original binding and schema checks.

This correction closes the three demonstrated failures. The source argument about overlapping observers uses owner exclusion, retained immutable bindings and cooperative, successful local service as specified in the frozen contract.
