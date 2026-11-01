# Process instrumentation audit

The apparatus uses existing process barriers, acknowledged inherited pipes, OS SIGKILL exit checks, read-only inspection and binary hashes from `internal/durabletest`. Original fixture sources/tests are preserved. New phase tests use their same primitives and independent persistent resource process. This note is source inspection by the implementer, not independent approval.

| Observer | Exact placement | Authority/state effect |
|---|---|---|
| `external.authorized/pre-transport` (existing) | Worker, after durable attempt reservation, before request encoding/Client.Do | Holds worker only; does not represent physical launch |
| `adapter.authorized/pre-transport` (existing) | Worker after durable attempt, before actual Execute | Holds worker only |
| `resource.ingress/pre-apply` (new) | HTTP body read/strict valid decode, optional ingress journal flushed, before Resource.Apply | Holds request before external mutation; no executor Store operation |
| `resource.applied/post-commit` (existing) | Resource.Apply returned after SQLite commit, before response encoding | Independent world commit, no invented receipt |
| `external.response/pre-record` (existing) | Successful HTTP response fully decoded, request binding validated, volatile receipt held | No result row yet |
| `adapter.response/pre-record` (existing) | Actual Execute returned a candidate | Candidate can be unsuccessful/unknown; not proof of nonapplication |
| `external.result-recorded/pre-settlement` (new) | recordExternalReceipt returned after commit/unlock, immediately before settleExternal | No-op without observer; preserves real separate transaction cut |
| `adapter.result-recorded/pre-settlement` (existing) | recordAdapterResult returned, immediately before settleAdapter | No-op without observer |
| `*.settled` (existing) | Actual settlement returned after commit, accounting loop blocked before scheduler selects | Known observation exists; not yet selected |

No new observer calls Store while Store.mu is held. The external seam calls only boundary.Reached after the recording function has returned. New endpoint journal observations do not decide authorization, repair evidence, change budgets, add receipts, or expose oracle state to ordinary programs. When enabled they append/Sync the received raw body and actual ledger result; unavailable journaling aborts the fixture request/experiment and is not a runtime policy. The journal's Contact identifier is scoped by Process token/PID; it restarts at one when the endpoint process restarts. Application truth comes from the raw resource database, not the passive journal.

The optional endpoint `--response-mode drop-after-commit` is an explicitly selected fault injection: it commits an ordinary application, reaches the existing applied boundary, then closes a real hijacked transport without encoding a response. Default normal response behavior and the underlying resource protocol remain intact. Ingress observation precedes Apply and the applied observation follows it. An endpoint arrival alone never establishes whether a worker started before/after local closure; cancellation experiments need the controller's pre-call hold/release and committed-closure order as well.

Safely closed baseline database copies establish paired identity; no authoritative row is edited. Each branch has its own independent world established before the executor starts. Raw backups use Python SQLite's backup API against read-only source connections and do not reset/checkpoint a live world. Oracle inspections/journals only flow to test assertions and evidence files, not to the executor registry or ordinary parent. Public Prepare serializes the original request; Execute receives no Store or world-database access.

Fresh subprocesses inherit the runner registry explicitly. Their helper imports public runtime for Open/Run, and uses the same test executable in normal/race mode. Resource binaries use the existing raceChildren build-tag selector. Each selected genuine executor crash is acknowledged before kill, its WaitStatus is verified as SIGKILL, and recovery PID differs. Endpoint restart is explicitly recorded as a separate endpoint kill, not mislabeled as executor recovery.

Independent review found the first version had changed the unjournaled fixture protocol by reading before route rejection and imposing a complete-body limit. The reviewed correction explicitly branches on journal presence: no-journal uses the original method/path guard and streaming decoder without a contact mutex; only the selected instrumented profile buffers/checks the complete body. Both behaviors have actual-HTTP regression evidence in the reviewfix normal/race command records. The rejected source is retained by exact hash and content.
