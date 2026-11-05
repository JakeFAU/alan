# Baseline and source map

Inspected and exported base: `926a899b651fccc48be52a054612a92afcf1ddfd`, tree
`a58fe3b0ef0b9a7525c46c4f0a662774e6d4221d`, on
`codex/p3-readiness-program-reuse`. Initial controller GitHub records establish
#9 merged, #11 closed unmerged, #12 merged, no open PR or existing P3 branch.
They are retained in `../initial-state.json` and `../pr-state-initial.json`
relative to the Task 1 raw root. No production source changes are part of Task 1.

The original #9/#11/#12 delivery SHAs are **not** ancestors of this main.
Actual #9 merge `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315` is an ancestor and
has exactly the #9 delivery tree `eda6f49da75bcb81d70eaaa244536d7258a9a312`.
Integrated request-authority base `1fdd4c9e1720bf60b344914bcbc3d0938ba8d678` is
also an ancestor. The complete current tree is byte-identical to #12 delivery
`4c8b4fdbca79b573ddd29fdc12fe920113af3cd1`; `git diff --stat` is empty.
The P2 `composition_inspect.go` is byte-identical to original P2 delivery
`49101122c31b0d790d364831f044bb996d2649fa` (SHA-256
`45f7a30be72c6212df23f0b493852f0b926e476a8de46a1a0602c3e2f49378c9`).
P1-containing files also contain integrated authority changes and are not falsely
reported byte-identical to the old delivery. The retained invocation-mode field
and predicate remain in current source. Raw ancestry exits and bridges are in
`ancestry.json` and `source-bridges.json`.

| Owning source | Actual behavior relevant to reuse |
| --- | --- |
| `internal/executor/runner.go:320` selectReady | Mutex held through all returns; reads ordered RunIDs and sorted observations; rotates after last run; checks actual projection, subject generation and retirement before load; loads/binds lazily; early returns on first ready run/observation or error. |
| `internal/executor/store.go:561` loadProgram | Fetches stored semantics/bytes; SHA-256 ProgramID; supported semantics; MaxProgramBytes; strict JSON; embedded semantics; request-authority format gate; full validateProgram. |
| `internal/executor/store.go:519` insertArtifact | Validates before insert; existing identity must retain equal semantics/bytes; no supported program update/delete site. |
| `internal/executor/request_authority.go:60` requestAuthorityEnabled | Stored schema number determines format13/14 feature context; this SQL validation input is part of successful load, not a ProgramID-only fact independent of a call. |
| `internal/executor/invocation.go:98` boundCompositionProgram | Reads current schema, full-RunID membership/binding and template, validates binding, then resolves member; absent member routes to handoff caller binding. |
| `internal/executor/composition_binding.go:289` resolveMemberProgram | Revalidates/encodes unbound program, checks member ProgramID, decodes independent nested data before writing bound endpoints/ports. This per-run work remains. |
| `internal/executor/invocation_handoff.go:154` bindHandoffCaller/Ports | Handoff program may mutate nested Code and Author.Ports through shallow copy; exclusive ownership is explicitly required. Exclude from reuse or minimally isolate before binding, with adversarial alias tests. |
| `internal/executor/program.go:120,132,299` decode/validateProgram/Reduce | Strict decode rejects unknown/trailing fields. Validation checks author/compiler metadata, schemas/instructions/collections. Reduce performs fresh validation, size/projection checks, clones data and cursors, evaluates cloned operands and constructs independent outputs. |
| `internal/executor/author_metadata.go:47`, `collections.go:26,168` | Metadata/collection validators read source data; collection reducer writes cloned next projection. Retained-schema validation creates copies rather than mutating source schemas. |
| `schema/schema.go:94`, `schema/collections.go:330` | Retain checks then clones before marshaling; Field.Retain uses a fresh record. This is allocation work, not ownership transfer from the input. |
| `internal/executor/store.go:161,693` acquire/Close | Cooperative flock excludes supported other owners; Close cancels/joins Run and owner users before mutex-held connection closure and lock release. Arbitrary unsynchronized SQL writers are not excluded by this mutex. |

Program values contain slices, maps and pointers. Passing them by value never
proves nested isolation. Preliminary reads found the concrete handoff hazard;
Task 2 owns focused alias probes and independent review of the complete read-only
claim. No unrestricted immutability, adversarial-store or concurrent arbitrary
SQL guarantee is claimed.

Existing coverage includes ordinary author/schema/runtime/executor suites,
real-process weather recovery and receipt conflicts, request-authority own and
delegated use/refusal/revocation/reopen, format restrictions and historical reader
checks, complete typed inspection and Replay, public consumers and hardened
competing-reader accounting. Baseline ordinary terminal events retain exact
passes/skips. Full integration/race and historical external-binary gates remain
later qualification obligations; historical reports do not qualify new source.

## Tools and measurement identity

Read README, extraction/inspection contracts, series-scaling attribution, P2
results/attribution/invariants/reproduction, convergence results/qualification,
and the actual driver, counter, coarse-timer and recorder implementations.
[source-identities.json](source-identities.json) records reused tool and relevant
production bytes. The complete tracked baseline manifest is raw
`baseline-manifest.json`; the Git export must match it exactly.

Unchanged tools: `performance-p2/driver/build.sh`, the complete nested driver
module, `performance-p2/run_schedule.py`, `campaign_validation.py`, P2
instrument/generate.py and its SQLite overlay, and instrument/timing.py.
New `instrument.py` invokes that unchanged P2 generator, then adds only P3
call-local diagnostic counters to disposable exports. Anchors fail closed;
P2/P3 patches and before/after manifests remain separate. It counts eligible
visits, actual encountered IDs, summed per-call distinct IDs, successful/failed
loads, decode calls/bytes, unbound validator calls, bindings, observation
attempts/selections and all-return visit/distinct histograms. It does not alter
tracked runtime source. SQL instrumentation is expensive; its timings are not
primary latency evidence. Coarse/profile binaries are separately identified.

The unchanged campaign recorder calls wait4 and records exit/usage before a
post-exit binary hash read; its OSError path writes failed metadata. Existing
22 campaign tests include removed and mode-000 binaries. Added Task1 tooling
control combines exit7 with self-deletion and inherited synthetic credential/
identity sentinels, requiring retained failure/usage and no sentinel values in
metadata. No recorder correction is needed. The convergence run_gate is reused
for declared build/test outer command logs; only relevant environment values
and omitted variable names are recorded. Output logs are not environment dumps.

Host identities are recorded with exact argv/cwd/exits in `identities.json`:
Go1.27.3 darwin/arm64, CGO=1, pinned go-sqlite3 v1.14.52 with module sums,
Apple clang, OS and CPU/memory identity. No dependency install or upgrade.
Ordinary timing uses fresh binaries/processes/stores and the fixed P2
measurement environment. Host load is recorded; thermal/cache/placement state
is uncontrolled. Raw root: `/private/tmp/alan-p3-j216kxoz/task1`.
