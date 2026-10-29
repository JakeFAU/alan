# Phase1I independent finite model

The runtime-boundary extension falsifies completion for the frozen parallel
context despite its initial K1 entitlement. In each K1/A/B census, exactly eight
of 84 entitled configurations have a failing schedule; the other 76 entitled
configurations complete. No external `close_root` occurs in this witness.
The root caller succeeds before its static peer admits S; the actual final
admission guard sees a non-open root projection and rejects S as `scope_closed`.
This falsifies the runtime correspondence / C[K] completion premise, not an
implication that assumes that premise. The contract, five callers, and initial
84-configuration entitlement are unchanged.

## Products and claims

- `model.go`: K1's frozen finite protocol and callers. Each committed operation,
  admission, request consume, reply commit/admission, terminal freeze,
  publication, delivery and consumption has a separate transition.
- `candidates.go`: independent hand transcription of frozen A13 / B16 / B-helper7
  retained instructions. Public protocol assumptions are shared with K1;
  instruction branch operands, reply values, and private registers are separate.
- `relation.go`: asymmetric local relation over 3,072 independent initial
  environments, with bilateral peer-debit/closure/service readiness at every
  candidate instruction boundary. This does not quantify over context runs.
- `explorer.go`: full-state BFS; every successor checked before deduplication;
  shortest counterexamples and intrinsic strictly increasing rank for the
  positive graphs. No cutoff and no scheduler-idle self-loop. Local progress
  is checked again with closure disabled, because optional closure must not
  make the open-path progress check vacuous.
- `runtime_boundary.go`: separately named inspected boundary extension, including
  root terminal admission fencing and static peer survival of root cancellation.
  It retains failing completion checks, rather than converting root Success into
  an external cancellation that would remove the guarantee.
- `oracle.go`: pre-candidate-inspection normalized prefix safety checker, unchanged
  through the corrections below. `envelope_test.go` adds a separately motivated
  direct contract oracle: fail/absent/pending cannot publish a candidate response.

The original corrected-v3 contract products support the declared bounded
model-only checks. They do **not** establish runtime correspondence. The
boundary extension is an inspected finite abstraction of specified guards;
concrete Store.Run and process recovery witnesses are separate evidence.
Neither interpretation proves compiler correctness, arbitrary contextual
closure, distributed semantics, or the manuscript's universal theorem.

## API

`Request`, `Observation`, `Violation`, `CheckTrace(Request, []Observation)` expose
an independent normalized safety monitor. `Operation`/`Parent` identity strings
must be consistently mapped, and debit fields are event deltas. Prefix checking
does not infer successful completion from an incomplete retained trace.

`Config`, `ContextObservation`, `CheckCandidateContext` check exact maximal
candidate product outcome membership. `Local` means A; `Helper` means B.
`CheckContext` selects the ideal K1 local/helper branch, while
`CheckCandidateContext` uses actual transcribed private control. An unpublished
candidate has empty status/reason and absent response. Open root/peer projections
are represented as `open`; nonparallel peer status remains empty.

`CheckBoundaryContext` returns `(Census, error)`: membership in the boundary
extension does not erase its completion counterexamples. Runtime negative
witnesses must retain rejection by the original model and the extension's
failed Gc result. Full trace/projection membership beyond these normalized
interfaces remains the runtime workstream's evidence obligation.

## Preserved defects and limits

1. Reference v1 collapsed helper/wrapper failed-outcome consumption with local
   terminal freeze. V2 split those boundaries. Original hashes, sources and
   census remain in `reference-freeze.json`, `reference-v1-sources/` and
   `reference-v1-results/`.
2. V2 reused public outcome fields as consumed-helper scratch, leaking present
   helper data into candidate fail/absent/pending. This passed earlier local and
   product checks because those graphs shared the protocol state representation.
   Runtime sampling exposed it. `TestModeResponseAbsenceIndependent` failed on
   the frozen-contract expectation before v3 separated returned-child data from
   public publication. `pre-payload-fix-v2/` and `payload-leak-red.log` preserve
   the superseded results. This is a model bug, not a runtime repair.
3. The extension initially classified an open peer's post-cancellation admission
   as `source_closed`; a direct test exposed the mistake. It now preserves the
   peer and classifies the shared-root fence as `scope_closed`. The original
   contract graph's cancellation convention is not promoted to runtime truth.
   `peer-boundary-red.log` is retained.
4. Wrong-id and extra-helper model mutations are rejected by the unchanged
   checker. The extra-helper graph deliberately creates a previously excluded
   ticket, so the ordinary rank also fails; this diagnostic is retained and is
   not the semantic detection. The `extra_helper` rejection and a reachable
   accepted third `(1,1,0)` debit are required independently. No acyclicity claim
   is made for that malformed graph from the failed rank.
5. Weak profiles have input0/output0, unlike K1's complement requirement. Their
   output monitor is explicitly separate. At budget2 the local implementation
   admits S and the helper implementation cannot; I3/R2 behaves likewise.
   Unequal debits do not refute Definition4.6. Equal-charge P/Q first pass the
   proposed local weak relation, then exact epsilon/subset trace-language
   equality for the three declared parallel capacities. Concrete scheduling
   outcomes are not inferred from this abstract equality.
6. Named closure checks and the five K1E schedules are auxiliary. The K1E graph
   checks reservation/launch/known-versus-unknown epistemic obligations; it is
   not a transcription of an effectful artifact or a complete public reply
   protocol/recovery implementation. `reopen_same_operation` is a mathematical
   retention event; only the separate runtime workstream can qualify SIGKILL.

Reproduce from repository root:

```sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build \
  ALAN_MODEL_EVIDENCE=/tmp/alan-phase1i-model \
  go test -v -count=1 -timeout=10m ./research/substitution
```

The evidence directory records exact per-configuration JSON (gzip compressed),
summary counts, shortest traces, frozen instruction correspondence, source
hashes, RED logs and current verification. Rank proves acyclicity only of the
finite enumerated positive graphs. This is bounded model evidence, not an
unbounded theorem or exhaustive runtime qualification.

## Final-review probe correspondence correction

The original weak/equal-charge probe model used B-derived surrogates. Independent
final review found that actual P/Q have no mode/presence dispatch, and actual
W_B's helper returns zero rather than the surrogate helper's complement. That
source and its results remain in `pre-probe-review-fix/`; those results alone
cannot establish correspondence to the public probe artifacts.

`probe_graph.go` now transcribes the actual six retained programs (41 instructions):
W_A5, W_B8, P8, Q9, zero helper4, complement helper7. The dedicated
`probe-artifacts/correspondence.json` binds source and template/program bytes,
complete operands/branches, exact public and original-handle receive guards,
and Q's inserted scratch Set. Q's flag7 represents that scratch=0 is defined;
its private canonical control locations align with P after the insertion. The
local weak relation hides that private definedness only, while environment
service remains directly enabled on both sides. Both helper branch arms are
mapped; the value1 arm has a structural instruction test outside the fixed
input0 product census. No extra product configurations are claimed.

The direct RED regressions and their correction preserve the evidence boundary.
All three local weak relations are checked before any contextual language check.
`CheckProbeContext` provides exact maximal-outcome membership for the four actual
profiles, optionally using the already separately declared runtime-boundary
extension. Its weak response checker requires zero for candidate **and helper**;
it is separate from the unchanged K1 `CheckTrace` complement oracle. Artifact
byte extraction constructs templates but never executes them to derive expected
outputs; it is not a compiler correctness proof.

The P/Q local relation and exact contextual trace-language equality reported for
this correction use the **original declared protocol** only. Runtime samples
can separately check maximal-outcome membership with `boundary=true`; that does
not establish trace-language equality under the runtime-boundary extension.
