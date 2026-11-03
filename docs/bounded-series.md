# Bounded observation series

## Declare once

An observation is an ordinary `author.Definition` with one adapter request, one
await, and the application's result checks. It declares the effect and input and
output schemas as usual. Wrap that definition once to obtain eight separately
accounted observations:

```go
series, err := author.BoundedSeries(observation, author.SeriesOptions{
    ID: "weather", Count: 8, Inputs: inputs,
    First: author.ApproveAtProvision,
    ApprovalReasons: []string{"operator_requested"},
})
if err != nil { return err }
provision, err := series.Provision()
if err != nil { return err }
owner, err := runtime.Create(ctx, path, provision, runtime.DefaultLimits())
if err != nil { return err }
defer owner.Close()
_, err = owner.Run(ctx, options) // normal adapter registry and runner limits
```

`Count: 1` uses the unchanged single-observation program. `Count: 8` derives eight
original subjects, eight independent one-opportunity roots and seven protected
continuation slots. The compiler derives the required grants. The application
does not construct these records. Zero is rejected. The count and one-opportunity
limit per position form the finite resource envelope; the adapter's existing
prepared request retains provider configuration and response limits.

`First: author.ApproveAtProvision` is required and authorizes only the first
observation. Creation itself makes no contact. There is no implicit approval of
the other seven. An empty `ApprovalReasons` list accepts any reason allowed by the
existing bounded approval schema; a nonempty list restricts it to those strings.
The owner authenticates the operator and the application decides whether to spend.

After reviewing the original evidence, explicitly authorize a particular next
position. Use the same ordinary Store APIs as a separately authored continuation:

```go
cut, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
if err != nil { return err }
next, err := series.Run(2)
if err != nil { return err }
basis, err := cut.ContinuationBasis(next)
if err != nil { return err } // the predecessor must have a paid opportunity
_, err = owner.ApproveContinuation(ctx, basis, "operator_requested")
if err != nil { return err } // stale evidence or wrong authority fails closed
_, err = owner.Run(ctx, options)
```

This remains a distinct physical operation with its own original identity and
expenditure. Approval alone spends nothing. Redelivery of the same decision
cannot spend twice. To recover, use `runtime.Open` on the original database and
`owner.Run`. No reconstruction of a volatile series counter is needed. A retained
response can finish locally; a paid operation with a missing response keeps its
original uncertainty. A new owner never approves the next position.

## Inspect the whole declaration

```go
view, err := series.Inspect(cut)
if err != nil { return err }
for _, position := range view.Positions {
    fmt.Println(position.Index, position.State, position.Root.Charges.Effects)
    if position.Observation != nil {
        // This is the actual selected result, including its original reduction.
        fmt.Println(position.Observation.ID, position.Observation.Payload)
    }
}
```

The handle checks the original deployment against its declaration. It orders
existing typed evidence by position, so applications need no joins on raw rows or
payload strings. `SeriesPosition` references the original run, effect, protected
approval, actual selected observation and resource root. `SeriesView` carries
the Store namespace, series ID, declared bound, positions and aggregate counts.
Rebuild the declaration from the application's original definition and inputs
after a process restart. This rebuild performs authoring only; it neither creates
a new Store nor grants additional authority. The runtime's general inspection
API remains available for a Store whose original authoring source is unavailable.

| Position state | Meaning |
|---|---|
| `not_eligible` | The preceding position has no paid original opportunity. |
| `awaiting_request` | The explicitly provisioned first observation has no effect yet. |
| `awaiting_approval` | A paid predecessor exists; this position has no approval. |
| `approved` | The protected approval exists, but no new effect is committed. Its selection is independently visible. |
| `committed`, `authorized` | The original effect exists, with source authorization separately established in the latter case. |
| `spent_before_dispatch` | Its opportunity was charged without a local dispatch admission. |
| `outcome_unknown` | Dispatch was admitted and no original result is retained. |
| `result_unselected` | An open run has a retained result that has not been consumed. Settlement is separately visible. |
| `selected` | The original result was consumed; local work remains. |
| `completed` | The selected result is validated, the run succeeded and local obligations are settled. |
| `failed`, `refused` | Local failure or runtime refusal is retained. The original effect and its uncertainty remain visible. |

`Selected` counts consumption, independently of success. `Completed`, `Failed`
and `Pending` classify positions. `Unresolved` separately follows retained
external uncertainty, including a local transport failure or conflicting receipt.
`Spent` counts paid opportunities. `BoundExhausted` means all N opportunities
were spent; it does not establish success. Position N+1 is never available, even
when fewer opportunities were spent. Provider usage remains on the typed effect,
with nil for unknown quantities. No token-to-price conversion is introduced.

This first construct supports standalone finite deployments of the same
self-contained adapter action. It does not create a parent invocation outcome.
Definitions with peer/local ports, invocation templates or repeated requests are
rejected. Application policy can stop before approving the next position, and a
failure before expenditure cannot make that next position eligible. An unresolved
paid original can be followed only by an explicit new approval. There is no global
physical single-flight guarantee across independently approved roots.

Construction expands O(N) static records and shares at most two program artifacts.
Input-copy cost also depends on the retained input size. The additional view uses
indexed runtime lookups and O(N log N) work, including approval searches; it does
not re-read the database. Ordinary `InspectExecution` and `Replay` retain their
existing snapshot contracts and costs. Default encoded-program/profile limits
apply, rather than an eight-position limit in the public authoring API. Historical
storage contracts remain unchanged: a single dispatched observation uses format
9, protected series use 11, and existing conflict retention can promote them to
12. Older readers that support these formats can read the ordinary lowered runs.

For multiple positions, the generated predecessor operation identity must fit
the existing 4,096-byte continuation field, including JSON escaping. Construction
checks this before any Store or expenditure exists. A collection loop before the
request reserves the full sequence-counter width conservatively. A single
observation has no continuation-field restriction.

[Weather application](../examples/weather-agent/README.md) ·
[Executable public example](../author/series_test.go) ·
[Protected approval protocol](continuations.md)

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
