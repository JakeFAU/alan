# Retained local dispatch admission

An ordinary adapter reservation consumes its resource allowance before the executor enters its final dispatch gate. Earlier formats retain both authorization and reservation, but a missing result leaves zero or one possible local calls. In particular, the database looks the same immediately before and after admission to that gate. A read-only HTTP GET can yield different data on a second request.

A deployment can set `CompositionProvision.RecordDispatchAdmissions` when creating a finite composition without the controlled receipt-read profile. Its scoped ordinary adapters retain one additional fact. After the executor checks the open generation, revision, scope, owner and context, it saves the operation ID, its original prepared digest and the current composition journal prefix before releasing the opportunity to call the adapter. A failed evidence write prevents the call. This operation neither reserves another unit nor grants a retry. The journal is append-only, with one row per operation. Both a result and its eventual selection remain independent facts.

| Last established fact | What a fresh owner can establish |
|---|---|
| Intent or source authorization only | No opportunity has been spent. The original request may progress under its existing authority. |
| Paid reservation, no gate admission | No physical call was admitted. The spent reservation survives even if the process died here. |
| Gate admission, no result | The owner admitted one local opportunity. Transmission, provider acceptance and the response remain unknown. |
| Result retained | Settlement and selection can continue from the original bytes without another call. A local transport failure does not establish remote nonapplication. |

Formats 9 and 10 carry this option on the composition protocol. Format 10 also retains conflicting receipt evidence. Existing formats 3–8 and unscoped adapters keep their original interpretation. The opt-in profile is independent of receipt reconciliation. Inspection exposes whether the format supports this fact and the recorded gate for each applicable operation. Replay and fresh-owner validation must reject a gate with no eligible original reservation, the wrong prepared digest, a closed or ineligible source at that prefix, or a retained dispatched result without its gate.

This evidence assumes a single host, the existing owner lock and SQLite durability. It establishes local admission, never receipt at a remote server. Only an independent endpoint acknowledgment can establish acceptance at the tested external boundary. A user-approved fresh weather request gets a separate identity and charge.

The gate must lie between complete committed transactions. Its reservation and any original result belong to the same owner. Replay and inspection check this against shared reopen counts, including evidence from other runs. They compare provably overlapping calls with each root's in-flight ceiling. An operation with no retained result has no recorded end time. A later owner can settle and select an existing result.
