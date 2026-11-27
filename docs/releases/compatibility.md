# v0.1 compatibility

## Public Go API

Applications define closed records with `schema`, compile `author.Definition` programs and wire instances with `author.Application.Provision`. Implement a trusted `effects.Adapter` to supply a tool. Hosts operate the local `runtime.Store` through `Create`, `Open`, `AdmitInput`, `Run` or `RunProgress`, continuation approval and `Close`. Readers use `InspectExecution` for typed queries and `Replay` to verify retained history offline. The exported APIs in `compose` and the supplied `adapters/*` packages support their documented bounded profiles. See public Go documentation for method signatures and ownership.

For v0.1.x, existing documented calls, error identities and configuration meanings are the compatibility contract. Use keyed struct literals and `errors.Is`. Patch releases may add compatible behavior or correct a violation of a documented invariant. A later v0 minor release may change the API. Its notes must identify affected callers and the required source changes. Pin a release in your application. Raw SQLite tables, executor implementation types, internal instruction layouts, human-readable error text and example command output outside the documented quickstart are implementation details.

`runtime.DefaultLimits()` returns bounds for one `Create`, `Open`, `Inspect` or `Replay` call. They are never retained, so a store may be reopened under different bounds. `RunOptions` requires positive step and dispatch budgets on every call. Zero in-flight and timeout fields select the documented defaults. The default effect timeout is chosen for a local adapter; a model provider needs its own. An empty registry can run local steps and reports an unavailable capability when a request needs a tool. Check the returned error and durable state after interruption or budget exhaustion. [Execution progress](../execution-progress.md)

Only a `runtime.Store` obtained from `Create` or `Open` owns a database. Calling any other method on a zero or nil `Store` is unsupported and returns an error matching `sql.ErrConnDone` rather than faulting. `Namespace` reports an empty namespace and `Close` succeeds.

## SQLite and history

Store formats have a separate version contract. At open, the runtime checks metadata, SQLite's version marker, required structures and ledger invariants before renewing ownership. Inspect and replay can read supported stores without an adapter. An unsupported version returns `runtime.ErrVersion`; inconsistent retained evidence can return `runtime.ErrCorrupt`. Keep operating system and SQLite errors available to the caller.

| Original format | Corresponding asset-scoped format | Protocol |
| --- | --- | --- |
| 3 / 6 | 17 / 20 | Ordinary computation |
| 4 / 7 | 18 / 21 | Finite composition |
| 5 / 8 | 19 / 22 | Controlled receipt reconciliation |
| 9 / 10 | 23 / 24 | Local dispatch admission evidence |
| 11 / 12 | 25 / 26 | Protected continuation |
| 13 / 14 | 27 / 28 | Request authority v1 |
| 15 / 16 | 29 / 30 | Exact request binding v2 |

Each pair's second version retains conflicting receipt evidence. Its first distinct, valid losing receipt promotes the format within the transaction that records it. Existing histories retain their original asset interpretation. New deployments with assets or an explicit asset scope select the matching scoped format. Normal reopen and replay preserve the physical version. Format 1, format 2 and unknown versions are unsupported. The release qualification report identifies the concrete fixture and reader coverage. [Capability-scoped asset selection](../asset-scopes.md)

There is no general migration or downgrade operation. To change an existing application's asset contract, provision a new store and retain the original for its history. A pre-scope reader supporting formats 3–16 rejects the scoped formats. Rolling back after a format promotion requires a compatible reader or restoring a complete earlier backup; a backup predating a dispatch cannot establish whether that dispatch reached its endpoint.

For an upgrade, stop the owner, allow in-flight work to finish when possible, and take a coherent backup. Preserve the database and any committed WAL frames together. Keep the binary/module version, application and adapter configuration needed to resume. Test `InspectExecution` and `Replay` on a disposable copy, then verify ownership and recovery with `Open` on that copy. Opening an interrupted store advances live epochs. [Storage and WAL behavior](../inspection.md) · [Dispatch and uncertainty](../dispatch-admissions.md)
