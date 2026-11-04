# Bounded-series qualification records

These are retained experiments and review records, not another runtime subsystem.
The evidence producer is commit `8a5f2ef564d5d750afa506675431d414bf92fb65`.
`producer-source.json` and `producer-source.sha256` bind its immutable export;
`producer-ordinary.json` records the complete ordinary command and exit status,
and `producer-ordinary.log.gz` preserves the raw `go test -json` stream.
Long Output fields can span multiple events: concatenate Output by Package/Test
before parsing `SERIES_CONTROLLED` lines.

The original database and WAL cuts, trace and hashes are in
[weather testdata](../../../examples/weather-agent/testdata/series.json).
The archive contains 21 ordinary-writer stores and their corresponding WALs.
No disposable corrupted copy replaces a positive control. Extract into a new
private directory and inspect/replay with no adapter registration.

`reviews/` preserves independent findings and scoped re-review dispositions;
`red/` preserves negative outputs and exact correction patches. Local scratch
paths inside verbatim reports identify the original qualification workspace.
The first rejected candidate tree was not delivered; decompress `red/task-1-fix1.patch.gz` and reverse that patch
from commit `220312ea17bb4014763e23a0a22237683bcdc910` to reconstruct that source.
Patch archives preserve their exact context whitespace. The long-identity counterexample uses that committed source directly. Later
fixes are separate commits; no reviewed commit was rewritten.

The [portable performance consumer](performance/README.md) retains baseline,
candidate and exact producer measurements separately. Its only internal import
is a disclosed test overlay for the existing conflicting-receipt writer.

[Qualification interpretation](../../bounded-series-qualification.md) explains
counts, compatibility, read-only limits and inherited performance costs. Final
immutable delivery verification, exact delivery commit/tree and final independent
approval are recorded in the draft PR after this evidence is frozen. This avoids
putting a self-referential final source hash inside that source itself.
