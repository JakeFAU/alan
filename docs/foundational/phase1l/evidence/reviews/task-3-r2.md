# Phase 1L Task 3 fix round 1 independent rereview

Specification compliance: **APPROVE**. Code/research quality: **APPROVE**.
R1: **ADDRESSED**. R2: **ADDRESSED**. No new blocking finding in the scoped fix.
This approves the corrected Task3 package within its declared bounded profile;
it does not approve the enclosing phase's remaining controller gates.

The original REJECT and counterexamples remain intact in
`task-3-r1.md`, `task-3-r1-probes.json.gz`, and runtime/rejected-r1/.
This review records correction of those findings rather than waiving them.

## R1 — original source revision authentication

The adapter now indexes retained subjects and verifies every history row's
source revision/generation against its actual subject. Full committed intents
inherit that authenticated history revision. Every intent table row must agree
with it, and admission Invocation.SourceRevision must agree with the original
intent. First-authorization descriptor/result revisions are checked against the
same commitment and subject. RevisionMatches, GenerationMatches and
SourceRetired now derive from those checked records.

Consequently the prior coherent mutation cannot certify mutually agreeing
altered intent/attempt rows: subject/history/admission remain1 while those rows
claim2, and raw validation fails before normalization. The same rule runs before
admission, so committed-but-unadmitted revision corruption also fails. No raw
error is recast as component nonconformance or G failure.

Epoch renewal remains independent of source revision. The code does not require
an old commitment history epoch to equal a later first-authorization epoch. The
expanded incomplete-prefix positive closes/reopens the Store, first authorizes
original S at unchanged sequence2, and asserts its epoch exceeds the commitment
epoch while revision remains1. The source-bound normal/race runs pass this
assertion and retain the accepted fifth prefix stage. Existing physical recovery
still records epoch2→3 with original consumed rows and spending preserved.

## R2 — inverse original intent/commitment/execution coverage

The adapter constructs a unique map of full executor.Intent values from original
history, including ordinal and source fields. Each table intent must match one
of those values and its row subject/generation/sequence/ordinal/source_revision.
Duplicate commitments and unmatched rows fail; commitment/table cardinality must
agree. Every intent must also appear exactly once in the ordered execution-fact
walk before raw validity is asserted.

The absent-sequence999 orphan is therefore rejected even though it has no freeze,
authorization, admission or charge that could otherwise expose it. The fix does
not require committed work to be admitted, authorized or completed. Initial,
e-committed-unadmitted, e-admitted-request-unconsumed, both-committed-s-unadmitted,
and S-admitted-after-epoch-renewal prefixes remain raw/prefix/component-valid
while correctly failing completed G. Operation IDs still come from actual
retained commitments; no expected tuple is constructed.

## Source and covering evidence checked

Read the appended fix report, exact scoped diff, changed adapter/regression code,
probe script, capture/audit changes, old/new snapshots and retained outputs.
Independently reconstructed the exact diff from the two review snapshots.
Verified all298 source hashes and confirmed that only the adapter and decoder
Go files changed from the rejected source manifest; runtime/process test sources,
inherited helpers, production/reference code and dependencies are unchanged.
All indexed file hashes/sizes, current snapshot contents and the original review/
probe/archive hashes match.

Corrected source aggregate:
`533faa388a04c63b18648bf1585e12ca54b416552d5bf9af01616d10fefc1e50`.
Records SHA256:
`213966027f857c5696d92d972eb9efa1debb7ed14253b53c34a1c4c173321677`.

The recorded RED and GREEN commands use the same regression-source SHA256
`27e53aecddb707491841fabf30cb3fca0d4b65e0927d37e750d77f6234112f4e`.
RED overlays exactly the archived rejected adapter
`002482103b77ccec8ae9e78ed089bf181d47ccc405b2384e42698fdda8181039`;
GREEN uses corrected adapter
`e8882db6ba7053a88d33a8bc06355ee63ce55444ba879767a7341bbf2d3999e0`.

- RED selects coherent-revision, orphan-intent and unadmitted-revision. All three
  fail because raw corruption is accepted; exit1 and all three detached accepted
  witnesses are retained. The incomplete witness fails completed G naturally,
  which does not hide its wrongful raw acceptance.
- GREEN selects the same three cases and exits0. All reject with
  `intent lacks exact original commitment/source metadata`.
- Corrected normal/race records each select39 leaves and exit0, with no failed
  or skipped qualification leaves. Scoped integration vet and the four-file
  gofmt check exit0 with empty output. The earlier extra-charge regression
  remains among the19 successful corruption checks per profile.

A separate reviewer read-only Python check recomputed intent/commitment row
bijections and subject/history/intent/attempt/admission revision agreement for
all24 saved decisive inspections per profile. It also checked the five retained
incomplete stages, unique S authorization in the renewed-epoch stage, all19 raw
rejection records, and recovery epochs2→3. These are checks of retained evidence,
not newly executed Go tests, runtime histories or a model census.

No passing suite, public consumer, model or runtime campaign was repeated by the
reviewer. No Go/production/reference source, TeX, commit or nested agent was
changed/created. The two requested review copies are byte-identical.

## Scope limits and separate controller gate

No new breakage attributable to this fix was found. The original passing Task3
observations remain bounded automatic samples, selected public schedules and one
physical recovery predicate per normal/race profile, alongside the independently
approved finite-model result. This approval does not establish arbitrary-context
congruence, distributed/effectful recovery or universal compiler correctness.

The adaptive retry was pending at handoff. The controller subsequently reports
exit0 for both tests and10 named subcases, with no fail/skip and unchanged compiled
sources; the original broad10m timeout remains recorded without a monolithic
suite-success claim. That is controller-attributed evidence, not a new adaptive
review or execution here. Its final disposition, manuscript/final checks and
draft PR remain separate from this corrected R1/R2 approval.
