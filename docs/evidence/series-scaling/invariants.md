# Frozen invariant analysis: immutable protocol-mode projection

Frozen after the complete baseline matrices and before any production change,
2026-11-04. Select one private protocol-mode projection; leave root-report
membership reuse and every other candidate unimplemented.

At 128, settlement sweeps occupy 31.350/50.704 seconds (61.83%) of ordinary
Execute. At 256 they occupy 235.700/337.680 seconds (69.80%); root reporting is
54.145 seconds (16.03%). These are separate coarse-timer diagnostics, excluded
from headline comparisons. The plain sampled allocation profile at 128
attributes 75.29% of bytes to invocationEnabled's full profile decoding.
The instrumented 256 execution calls invocationEnabled 398,333 times, PrepareRoot
392,960 times and settleRoots 1,535 times. Its 399,868 total descriptor decodes
process 54.111 GB of immutable-per-Store input. Full ledger/protocol replay
validator calls during Execute are zero; fresh typed snapshots still occur at
every approval. These overlapping scope/count/allocation measurements support
the selected repetition mechanism; they do not predict removable elapsed time.

The plain diagnostic Execute was 359.390 seconds at 256; the separate timer run
was 337.680 seconds. That variation is preserved. Neither is a before/after
optimization result. The predeclared paired plain-source campaign decides that.

## Necessary work, unchanged-input work and temporal boundaries

`Store.invocationEnabled` in composition_store.go asks whether this Store's
private retained composition profile has Version == compositionProtocolVersion.
It presently decodes the entire raw descriptor to answer that single question.
`settleRoots` calls it, then calls PrepareRoot for every root; each PrepareRoot
calls it again. Other invocation, message, lifecycle, continuation, dispatch,
source/fact and Commit paths use the same discriminator. The source audit lists
actual call sites. Decoder work grows with the provision even though its answer
cannot change during the private profile's lifetime.

Class A: retaining/canonicalizing and validating the profile, independent
ledger/replay/causal validation, current source authority, root readiness,
charges, results, settlement and dispatch gates remain necessary.
Class B: decoding the same privately owned immutable raw bytes just to recover
this previously validated protocol version repeats unchanged evidence.
Class C: PrepareRoot→SettleRoot, readiness→Commit, snapshot→ApproveContinuation,
authorization→reservation→dispatch and fresh-owner activation are distinct
serialized checks; none may be skipped or replaced by this flag.

The alternative per-report membership memo would remove real repeated SQL,
but at 128 root reporting occupies only 14.84% of Execute. It is deferred, and
neither it nor sorting/grouping/index work belongs in this change.

## Exact change and binding

Add one private boolean to compositionProfile. Derive it at the existing sole
constructor from the same successfully validated, defensively copied canonical
compositionProvision whose bytes are retained as raw. A valid invocation
protocol profile is true; the private foundation profile and ordinary nil
profile are false. Return this immutable projection in invocationEnabled.
Keep descriptor() unchanged for callers needing an independent descriptor.
No shared descriptor/map, Store-global history cache or public API is introduced.

The binding is the exact privately owned profile object, not a final epoch,
program name, equal payload or a caller's mutable provision. Profile retention
already clones caller containers and validates versions/catalogue/membership/
slots. The object is published only after all retention succeeds. The flag
must never be constructed from unvalidated configuration or written separately
by a runtime mutation. Search confirms one production profile constructor and
no mutation of its raw bytes.

## Invalidators and lifetime

There is no supported mutation of the retained composition profile within a
Store lifetime. New invocations, observations, approvals, grants/authority,
results, resource charges, closures, activation epochs and receipt-conflict
format promotion do not change this protocol version. Those facts retain all
of their current checks. A version flag conveys no authority to spend or execute.

Create constructs its own validated profile. Open reads the persisted schema,
version markers and canonical bytes and constructs a new validated profile
before existing activation renewal. A foreign/malformed/truncated/mismatched
stored descriptor is rejected by those unchanged checks, rather than receiving
a previous owner's flag. Replay and inspection remain independent validators;
neither reads or borrows a live owner's in-memory summary. Distinct Stores have
distinct objects even when their subject/generation names coincide.

If caller aliases or decoded descriptors could mutate retained bytes, or a
second constructor could forge the flag, this premise would be false. If a
future protocol made mode mutable, that change would need to replace this
projection contract explicitly. Unsafe memory corruption or unsupported direct
SQLite writers while an owner is active do not become a new supported failure
model: the previous implementation also read the captured private raw string,
not live database bytes on each invocationEnabled call.

## Falsifiers and verification

- Nil ordinary profile, private foundation profile and supported invocation,
  reconciliation, dispatch and protected-continuation formats retain their mode.
- Mutating the original provision or an independently decoded descriptor after
  retention does not change mode, original bytes, limits or grants.
- Different Stores, generations and successive owners do not share mode state;
  fresh owner revalidates and reconstructs it. Schema/version/raw corruption
  rejects Open/Replay/inspection without partial activation renewal.
- An unapproved protected target remains blocked; true mode is not an approval.
  Stale approval, source revocation and final dispatch gates remain effective.
- Original SIGKILL witnesses, immutable originals, unknown receipts, conflicts,
  exact charges, live owner exclusion and competing snapshot readers still pass.
- Same-seed baseline/candidate full public cuts, all History pages, Replay,
  exact request/result bytes, approvals, charges, contacts and durable typed
  table cells compare without identity/epoch normalization.

The performance RED is the retained baseline profile/count matrix, not a claim
that existing semantic behavior was wrong. Regression checks may therefore pass
both baseline and candidate. A focused allocation check may demonstrate zero
allocation per mode query after construction, while the complete paired ordinary
workload remains the success criterion. Do not replace an existing negative test
expectation to make the change pass.

## Bounded claim and stopping

The projection adds constant per-profile storage and removes full JSON decoding
from this one predicate. Root sweep counts/order, readiness, validation, snapshots,
SQL and resource/dispatch behavior are otherwise unchanged. Other full profile
copies, root-report queries and repeated scans remain. A flag is never durable
authority or a validated ledger prefix. No new dependency, table, migration,
format, public signature or execution abstraction is justified.

Stop if any validation/authority/corruption counterexample cannot be resolved
within this projection, if the attribution does not support it, or if frozen
paired ordinary measurements show only noise. Do not add root-membership caching
or another optimization to rescue it. Do not infer a universal scaling law or a
specific latency saving from the allocation share alone.
