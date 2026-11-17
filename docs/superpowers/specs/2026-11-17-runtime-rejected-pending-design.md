# Truthful pending invocation inspection

The approved delegated Weather example includes a refused helper admission. A
source comparison found that this runtime's root inspection lists every
unpublished invocation as pending, although composition-v1 settlement excludes
invocations whose admission was rejected. Research commit
`3d1d63fa6a69f590f145768745793d80f9839268` contains a narrow correction to this
predicate alongside its larger composition-v2 work. Port only this correction
and its regression, in a separate receiving-runtime change.

## Contract and bounded scope

- An invocation with no admission decision remains in `PendingInvocations`.
- An accepted but unpublished invocation remains pending.
- A rejected invocation does not create descendant settlement debt and is not
  listed in root `PendingInvocations`. Its original intent, admission refusal,
  outcome delivery and any unconsumed caller wait remain visible through their
  existing evidence paths. Excluding descendant debt never erases these records.
- Inspection, replay and reopen agree on this distinction. Storage/load errors
  still propagate. There is no new storage format, composition version,
  settlement behavior, dispatch behavior, public API or dependency.

Reproduce the mismatch before changing production code. Adapt the originating
regression to this runtime's actual resource accounting and add the accepted
pending control if it is not covered. Keep the receiving fork's indexed root
inspection and ordering. Do not import v2's early-return/drain behavior or any
other hunk from the originating commit.

This corrects diagnostic correspondence with the existing v1 settlement rule.
It is not a new liveness or substitution result. Record exact source provenance,
RED/GREEN commands and limits in a narrow port note. Deliver a separately
reviewed stacked PR, without merging automatically.
