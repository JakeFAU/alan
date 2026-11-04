# Frozen reciprocal-delegation qualification

Executable source: `759df6e3d2505f27e1287b2cfeb07bc4290afc0a`.
The unchanged primitive is `4f6f5f2500a48b2ca01ee7b09a823fa63656b7ab`;
the source freeze adds eight fixture/test files and changes no production file.
The final authority contract is research `360cae66f5ffdd6e7a95a568552df5522122da21`,
`docs/foundational/phase3b3/02-contract.md`.

`evidence.tar.gz` contains 3,784 files under `runtime-frozen/`, including all
commands, logs, raw process and endpoint witnesses, closed database sets and
case manifests. `manifest.json` binds every archive member. The archive hash is
`0c8c48f2c9c2c064ed03e210ddec7d0ec8179162d5203692326e99e9254a58d4`.
All member hashes and all 1,761 case-artifact digests in each corpus were checked;
archive verification opened no SQLite database.

The 304 Go/module inputs in `source.json` were unchanged throughout the eight
commands in `commands.json`. Go was `go1.27.3 darwin/arm64`; Staticcheck was
`2026.2.2 (0.8.2)`. All eight commands exited zero:

| Command group | Observed result |
| --- | --- |
| Full ordinary module | 1,265 test/subtest pass events, 18 passing packages; three additional packages contain no tests |
| Core integration | 1,597 pass events across four packages, no skips |
| Reciprocal ordinary / race | 48 indexed cases and 53 pass events in each run, no skips |
| Focused authority race | 105 pass events across three packages, no skips |
| Existing baseline on current source | Seven pass events and one deliberate opt-in expected-failure test skipped |
| Full-module Staticcheck, ordinary / integration | Both clean |

Pass-event counts include subtests and do not count independent experiments.
The historical baseline's actual S2 expected failure is retained in
`../baseline-frozen.tar.gz`; the current opt-in skip is not new RED evidence.
`verification-summary.json` preserves Go's package-level `skip` events for the
three packages without tests, separately identifiable by their null test name.

Each corpus includes ordinary S1/S2/S3, fifteen binding controls, four genuine
Cartesian-operation controls, revocation timing, late-grant refusal, actual
result laundering, serial/serial waiting, generic and revoked unknown outcomes,
and five malformed-Store copies. The ten requested cut labels produce eleven
acknowledged SIGKILL/reopen witnesses because the admitted/no-contact and
applied/no-local-receipt subcases are distinct. Labels 04 and 05 intentionally
observe the same runtime boundary. Cut 10 has both original results consumed;
B still awaits the terminal handshake. It is not an unpaid-original witness.

Successful recovery of an applied but unrecorded idempotent effect can contact
the endpoint twice but applies the stable operation once. An allowed gate
proves an opportunity, not a contact. Generic adapter unknowns block without
retransmission; revocation before a fresh admission can leave an already applied
effect unknown. These are separate outcomes, not universal exactly-once or
completion claims. Ordinary reciprocal fixtures use format 13; format 14 is
covered by the primitive tests, not by these process cuts.

The positive fixture selects ready, then the authentic original request, then
service. The wrong-prior-selection control retains a valid exact grant and an
earlier original selection, but an intervening ready selection causes refusal.
The old abstract model permits authorization at that prefix. This is a retained
model/runtime discrepancy under unrestricted stuttering, not missing consent.
Separate grant plus revocation and duplicate-contact/unknown-result histories
also require explicit model-domain qualifications in the research checker.

Extract to a fresh scratch directory and verify members against the wrapper
manifest. Preserve historical absolute paths and digest-bound files. A checker
must explicitly map `/tmp/alan-phase3b3-runtime-frozen/normal` (or `/race`) to the
corresponding extracted directory. To rerun, check out the source freeze and
use fresh output roots for the exact commands in `commands.json`. Build the old
reader from `af6943a5805a9a6c0e8fe77f804264f4765d07ec`, as described in the
parent README, and set `ALAN_PHASE3B3_OLD_READER` to that new binary.

`freeze.py` is the original generating script, with historical absolute paths
retained. Copy it before remapping its repository, output, cache and old-reader
paths; pass the frozen source commit as its sole argument. Never rerun it into
the archived output directory. Do not open archived SQLite originals, including
read-only opens: copy each complete database/WAL/SHM set to disposable storage.
Post-Open snapshots use online backup before Run.

Independent binding/custody and recovery/admission reviews, their scripts and
derived checks are retained in `../task4-reviews`. Both scoped reviews returned
APPROVE. They do not establish whole-phase model correspondence or a general
composition theorem. Development failures remain in `../task4-development`.
