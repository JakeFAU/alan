# Bounded observation series

## Frozen authoring contract

This contract precedes implementation. The verified base is
`1d0ef3c91589c3df69b5a004cd5362438e8eed46`, the merged continuation delivery
from PR #6. Its tree is `c7266108c1e8913534781a066defe93f4057b8dc`.

An author supplies one finite observation definition, its typed inputs, a stable
series name, a positive observation count and acceptable operator reasons. The
declaration explicitly authorizes the first observation at provisioning. Each
later observation requires a new protected continuation approval. Count is
capacity, and is never approval of the remaining observations.

The first version has one ordering policy. Position i becomes eligible for an
explicit approval after position i−1 has spent its original opportunity. This
allows an operator to request a distinct observation after an unresolved read.
It does not assert that the predecessor succeeded, terminated or was not applied
remotely. Application policy may require selection, success, freshness or other
conditions before asking for approval. The weather application continues to
finish local work when a response is already retained. No policy callback runs
during inspection, Replay or reopening.

### Lowering and identity

`author.BoundedSeries` will compile a self-contained, single-adapter observation
into ordinary original generation-one runs. The first uses the unchanged
observation program. Later positions share one program with an ordinary
protected approval receive before the observation. The helper derives subjects,
one-member resource roots and predecessor slots. `Application.Provision` continues
to derive the adapter and local-input grants. Each root has one effect opportunity
and one in-flight slot. The aggregate envelope is N paid opportunities.

The declaration must be finite and contain exactly one ordinary adapter request
and its corresponding await. Multiple operations, invocation templates, peer
wiring, and extra local inputs are outside this initial construct. Reject invalid
definitions before provisioning. No separate scheduler, durable counter, new
runtime series type or table is introduced.

Stable identities derive from the declared series name and canonical one-based
position. The Store namespace identifies the actual deployment. Original effect
identities remain those created by the normal committed request instruction.
Equal payloads, timestamps, allocation order and process identities never create
a causal edge. A returned series declaration owns its data. Caller mutations of
options, schemas, inputs or a returned provision must not alter it.

The containing series is an authoring and inspection handle for the complete
finite deployment. It is not a single invocation template or one root outcome.
The existing continuation contract requires separately closable one-member
roots. A public series view must validate the exact original deployment and
return positions in declared order using existing typed run, effect, approval,
observation and resource facts. Selected results come from actual consumption.
It must never turn an available unselected result into a selected one or mark an
aggregate successful while any position is unresolved.

### Positive controls and falsifiers

* One position executes the original program with one independently recorded
  opportunity. Eight positions have eight original identities and no ninth slot.
* Later approval withheld means no later contact or debit. The first grant is
  explicit in the declaration; provisioning itself makes no contact.
* A duplicate approval returns the same decision and cannot spend twice. An
  approval for a future position without a paid predecessor fails closed.
* A lost response at position 3 retains that exact unknown operation and its
  charge. Positions 4–8 remain unapproved through repeated reopening. Explicit
  approval may allow position 4 without changing position 3.
* A retained response survives process death and can be selected without another
  provider call. Failed/refused observations remain attributable to their own
  operations. Existing prepared-action matching, authority and scope checks apply.
* Index, generation, predecessor, approval, source grant or resource corruption
  on disposable copies must fail the owning validation boundary. A declaration
  mismatch must not be reported as a valid series. A coherent different deployment
  is not the requested declaration; no cryptographic protection against rewriting
  an entire database is claimed.
* Inspection and Replay make no external contact or durable change. An owner
  validates before renewing activation. Repeated reopening never expands capacity
  or introduces approval. Local commitment/accounting/admission guarantees are
  distinct from remote application.
* Zero is rejected by the new declaration. Historical weather allowance-zero
  stores retain their existing no-spend behavior. Bounds 1, 8, 32 and 256 are
  qualified against existing encoded-program/profile and schema limits, rather
  than a new fixed eight-item limit.

The series view distinguishes not-yet-eligible, awaiting approval, approved or
committed but unspent, paid before dispatch, admitted with missing result,
retained but unselected result, completed, and failed/refused positions. Bound
exhaustion is reported separately from successful completion. Unknown provider
usage and cost remain unknown. The application retains ownership of weather
formatting, provider timestamps and native evidence verification.

### Alternatives and compatibility

| Approach | Decision |
|---|---|
| Keep the weather provisioning loop | Already correct, but leaves the author responsible for positions, roots and continuation wiring. Grants are already derived by `Application.Provision`. |
| Pure authoring expansion | Selected. Existing protected continuations enforce the necessary identity, authority and independent expenditure. A series view removes repeated joins without changing execution. |
| One parent invoking all positions | The existing collection starts its children before collecting. Dynamic children cannot use the current static continuation slots. Extending that protocol is unnecessary for the requested sequential observations. |
| One root or repeated effects in one run | Incompatible with independently closable, single-use continuation scopes. |
| New executor series protocol | No missing invariant justifies it. |

No migration or dependency change is intended. Existing formats keep their
contracts. A single observation uses the existing dispatch-admission format.
Protected multi-observation series use existing formats 11/12. Existing weather
programs and the archived live store remain recoverable and replayable.

Qualification will include independently acknowledged SIGKILL cuts, loopback
provider contact logs, corrupted copies, external-module compilation, ordinary
and race-enabled full suites, measured scaling and independent source reviews.
Stop if the lowering requires new authority, refunds an opportunity, fabricates
selection, hides unknown work, or cannot faithfully represent the chosen policy.
