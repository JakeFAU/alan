# Phase 3B.3 legacy format matrix

Ten fixtures were created by frozen af6943a through actual writer APIs. Old and candidate readers each received independent complete copies for Replay, Inspect, InspectExecution and Open. The 80 primary calls passed; 60 paired JSON exports match byte for byte, and post-Open typed deltas are only the expected renewed epochs. Formats 5/8 exercise the reconciliation profile but not a receipt-read operation. This is finite compatibility evidence, not universal equivalence.

`evidence.tar.gz` retains all source overlays, raw fixtures and reader copies, JSON exports, commands and logs, including the initial storage/protocol version assertion setup error. Rebuildable `.test` binaries remain at the original temporary path; their hashes, sizes and build instructions are retained, not the binary bytes. The source overlay is identical for both versions. `manifest.json` binds every archived member, omitted binary, original manifest check and approved source snapshot. The full ordinary module suite also passed on this source; integration and focused race results are retained separately in `../round1`.

Never open an archived database with SQLite, even read-only. Copy the complete database/WAL/SHM set to a disposable directory first. All archive verification here read only file bytes; no archived database was opened.
