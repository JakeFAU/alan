# Author a retained outing plan

The outing planner uses two ordinary authored programs. Outing knows the
park catalogue, opening hours and travel constraints. Weather has the
capability to request hourly forecasts. The host provisions the pair,
provides its transport, and drives it through the runtime. [Runnable example](../examples/outing-planner/README.md)

## Normalize the question at provisioning

The host resolves the target date in the supplied timezone against one
reference instant. It converts each slot into a pair of consecutive UTC
hours and determines whether park hours cover the one-hour walk and
availability covers the walk plus both travel legs. It ranks all park/slot
pairs by one-way travel time, UTC start and park ID, retaining the original
configuration and the normalized records, including exclusions. Go handles
calendar arithmetic and ordering at this boundary. The retained Outing
program selects the eligible records and performs the subsequent protocol
through authored reductions. A restart keeps the normalized date and hours
even when the local calendar has advanced. [Configuration and normalization](../examples/outing-planner/configuration.go)

The deployment compiles both definitions and wires the three typed ports
with `author.Application.Provision`. It gives generation 1 of `weather` an
hourly forecast capability and an effect allowance equal to the number of
eligible candidates. Each run has a separate single-member resource scope.
`weather` can have one effect in flight. Provisioning enables retained
dispatch admissions. The host creates the Store with both initial
programs and inputs. [Provisioning](../examples/outing-planner/deployment.go)

## Exchange the proposal, assessment and decision

Outing sends the proposal with its retained proposal ID and eligible
candidates. Only the bound Weather generation can supply the assessment.
Weather selects the proposal, checks its identity and processes the bounded
candidate list in rank order. For each candidate, `author.Request` commits the
location, exact UTC hours and units and records the original operation
handle. `author.Await` selects the forecast result for that handle. Weather
requires a valid response and an allowed code at each requested hour
before marking the option suitable. The batch carries the original
forecast operation IDs alongside the assessed candidates. [Program definitions](../examples/outing-planner/program.go) · [Typed port schemas](../examples/outing-planner/protocol.go)

Outing checks the returned proposal ID, candidate order and coverage, and
the presence and uniqueness of the original forecast handles. It chooses
the first suitable option. An empty eligible list or a fully unsuitable
batch yields `no_match`. If some evidence is unavailable and no suitable
option was established, the result is `unavailable`. Weather selects the
final decision and validates its identity, outcome and selection. The
decision send and its later selection are separate retained facts.
[Evidence checks](../examples/outing-planner/evidence.go)

## Supply transport and follow the retained record

On `run` or `resume`, the host acquires ownership with `runtime.Open` and
checks that the retained deployment still agrees with its inspection. It
uses the saved endpoint to instantiate the hourly adapter. Transport runs
through the bounded public runner. The application supplies one in-flight
dispatch slot, an external timeout, and per-call step and dispatch bounds.
The resource scope's original effect allowance persists across owners.
[CLI owner path](../examples/outing-planner/main.go) · [Hourly adapter](../adapters/openmeteo/hourly.go)

The adapter prepares a single GET from each typed intent. Inspection checks
its prepared request against the retained intent and validates recorded
native responses offline. A dispatch admission means the original passed
the local transport gate. Contact at the remote endpoint is a separate
event. When no response is recorded, the operation remains outstanding.
When a response is already retained, a fresh owner can finish local
settlement and selection from the saved result. [Dispatch admissions](dispatch-admissions.md)

Use `inspect` to find an original operation and its attempt, admission,
result and settlement as those records become available. The settlement's
observation ID leads to the observation's `SelectedAt`, which names the
selecting reduction in the run history. For a peer message, look for a
message observation with that original `SourceOperationID`. The report's
`decision_selected_at` points to Weather's consumption. `replay` checks
the committed history and projections without transport. [Planner report](../examples/outing-planner/report.go) · [Inspection contract](inspection.md)

## Change the order of Weather's receive

The two Weather variants use the same types and grants. Assess-first
sends the assessment after selecting the proposal and evaluating its
forecast candidates. Confirm-first first requires the final decision.
Outing can send that decision only after it receives the assessment,
so the supported confirm-first pair has no first sender of either missing
message. The checker uses fresh Stores and fixed loopback evidence. It
reports the reciprocal wait after the bounded runner returns quiescent
and the retained cut satisfies the closed-pair checks. Ordinary readers
expose those waits as pending. [Checker](../examples/outing-planner/replacement.go)

This is empirical evidence about a fixed pair in one Store. The manuscript
at `3dc2754a95d08cf2a688d6a6a970dfed2b2408b5` §6.9 contains the
reciprocal control-order analysis and a separate same-Store fixture.
§§6.18–6.20 address finite continuation monitors over an unchanged plant.
The planner experiment leaves distributed ownership and general substitution
open. [Pinned manuscript](https://github.com/JakeFAU/alan-research/blob/3dc2754a95d08cf2a688d6a6a970dfed2b2408b5/paper/alan-dissertation.md#69-reciprocal-obligations-the-phase-3b1-service-boundary) · [Process evidence](../examples/outing-planner/QUALIFICATION.md)

## Delegated Weather

`--weather delegated` replaces Weather's local assessment loop with one ordinary
retained helper invocation. Outing's artifact and its public exchange remain
unchanged. The helper receives the exact frozen candidate batch, performs the
sequential original forecast requests, replies, and ends. Weather selects the
published complete result before sending its public assessment; Outing still
chooses and returns the decision. Delegation is bounded here to two eligible
candidates, one helper and one transport identity per original message.

Run `examples/outing-planner/testdata/delegated-walkthrough.sh` from the checkout.
It uses the existing Cedar/Hemlock fixtures, saves real CLI replacement exports,
checks complete raw Stores with the independent Python model, and shows both
accepted and allowance-zero helper scenarios. Its printed directory contains
configuration, fixtures, manifests, Stores, reports and checker output.

Completed direct and delegated plans can agree on candidate and outcome while
costs differ: `(invocations, created runs, effects)` are `(0,0,n)` and `(1,1,n)`.
A known invalid native result may complete diagnostically as unavailable;
refusal leaves the parent's outstanding wait; an acknowledged GET with no
retained response stays unknown across fresh resumes. None is a successful
replacement merely because an answer field matches.

The independent finite protocol separates broad service orderings from the
`preclose-authorization` domain. Broad ordering admits a public decision first
authorized after Outing End, which permanently records `source_closed` and
leaves Weather waiting. The constrained positive domain assumes an allowed
first authorization before sender End. The process campaign captures the
automatic decision-commit cut and checks its stronger local premise: open
sender and receiver, the exact matching wait and the decision as the sole
pending internal delivery. Finite model paths do not establish universal
runtime refinement or eventual progress under arbitrary interruptions.

Research §4.5's budget-indexed relation requires identical debit footprints;
the added invocation/run/depth does not satisfy that requirement. §6.20 concerns
finite synchronized continuation monitors around an unchanged plant, and does
not prove extraction into a separately executing helper. This example adds
bounded empirical comparison and recovery evidence; it changes no research
source or theorem. See the example's [qualification record](../examples/outing-planner/QUALIFICATION.md)
and [independent checker contract](../examples/outing-planner/model/README.md).
