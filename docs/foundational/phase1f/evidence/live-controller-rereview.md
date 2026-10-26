# Live controller accounting correction: independent rereview

Reviewed 2026-10-26 against the initial `live-control-brief.md`, `live-controller-review.md`, frozen `live-control-fix.patch`, and correction report. Scope is the accounting correction in `tools/live_control.py` and `tools/test_live_control.py`, including its retained red/green evidence. No new application review, paid calls, provider/network calls, credentials, `.env` reads, subagents, source edits, or commits were performed. Only this rereview report was added.

**Specification verdict: PASS for the scoped correction.** The original P2 finding is resolved.

**Quality verdict: PASS for the scoped correction.** No new actionable finding was identified in this fix. This supersedes the initial changes-required verdict for the accounting defect; the initial review's broader scope limits and local pricing assumptions still apply.

## Resolution checked in source

`usage_counts()` accepts only exact nonnegative integer input/output/total counts, excluding booleans, and requires `total = input + output`. It conservatively rejects a zero total. Cache read/write counts must be exact nonnegative integers with their combined count no greater than input; malformed accounting/detail objects are rejected. Missing optional cache details still mean zero cache counts, as before.

Before reconciliation can discount any paid operation, it removes any previous budget charge, estimate, and validated usage and marks the operation unresolved. Missing result/usage/counts retain the full reservation with an explanatory reason. Malformed accounting is caught as a `ValueError`, retains the full reservation, and receives `malformed-usage-reservation-retained`. Existing request/response IDs are copied when a result exists. Neither path supplies `budget_charge_usd`, so `charge()` charges `reserved_usd` and another operation cannot reuse the rejected discount.

Both paths continue through `run()`'s existing ledger and run-record persistence. `summary.json` now uses `unresolved_reservations()`, which includes every operation except reconciled `usage-known` and local operations. Consequently malformed, missing, and still-released reservations are represented in the unresolved summary total. The helper test checks the combined $0.90 from released $0.20, unknown $0.30, and malformed $0.40 reservations.

Valid usage follows the existing estimate and conservative-charge arithmetic unchanged. The retained fixture still produces $0.001686 estimated cost. The cache regression verifies 100 input tokens (30 cache-read, 10 cache-write), 20 output, total 120: $0.000373 estimated cost and $0.000875 conservative charge, with the same request/response IDs and no unresolved reservation.

The diff changes only usage validation/reconciliation and the unresolved summary helper. Reservation persistence-before-release, boundary correlation, descriptor handoff, credentials, transport, and replay paths are unchanged. Subsequent model/tier/response/reservation-overrun errors still propagate; the prior durable reservation remains intact. The regression for a model-validation exception verifies both durable and in-memory full charges. This is source inspection plus a local file regression, not a claim of full crash-injection coverage.

## Evidence reviewed

The correction's exact command was:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s docs/foundational/phase1f/tools -p test_live_control.py -v
```

`live-controller/accounting-red.txt` records 20 tests with 27 failing subcases and 23 errors against the original controller; the original nine tests pass. The failures concern discount releases, malformed accounting, and the newly required reason/helper behavior. `accounting-green.txt` records the same 20 tests passing. The actual current source hashes match the correction report. The added tests cover the original two failures, token/cache field types and signs, total consistency, excessive cache totals, malformed shapes, missing usage, stale discount removal and subsequent budget rejection, valid cache pricing/IDs, summary inclusion, and durable reservation behavior.

The retained original probe script has the same original usage objects and candidate ID `probe`:

| Input/output/total | Before fix, $0.20 reserved | After fix |
| --- | --- | --- |
| `0/0/-1` | `usage-known`, charge $0 | malformed, charge $0.20; reason: nonnegative integer required |
| `1/1/666` | `usage-known`, charge $0.00001875 | malformed, charge $0.20; reason: total must equal input plus output |

The before/after probe outputs preserve `probe` request/response IDs. Original source copies retain exactly the source hashes from the initial independent review. The actual source diff against those copies matches the scoped correction. No concrete coverage gap required a rerun of the same suite or original probes; none was performed during this rereview.

The recorded test result is evidence from the implementer's retained execution, independently checked against source and the frozen hash inventory. This rereview does not claim a fresh independent test execution, full controller/provider run, invoice verification, or a newly executed replay sandbox control.

## SHA-256 checked

```text
5e1cb22163408904e25d19d8615ec45a9b36fb5efea236075f3d2d37200a8ade  docs/foundational/phase1f/tools/live_control.py
d9043ffc8c2334fecc53e2eb64a5828a58d31217b5b74fdaafce840822a2dfff  docs/foundational/phase1f/tools/test_live_control.py
1f7d8796fa9d37a015e78fa68310fc6fce1c51e94b3bf9332268efe42e87e682  docs/foundational/phase1f/evidence/live-controller/live_control.py.before.txt
e230162d4cd44c10080567f019376be62212bdfafaa60e73f4ca00c962e15a5b  docs/foundational/phase1f/evidence/live-controller/test_live_control.py.before.txt
d48c9861a10d628ec08a8bcad0711e2312c35b8d71bb97a72c4be1898ecb420e  docs/foundational/phase1f/evidence/live-controller/accounting-red.txt
c412ee20aa52533d3e85f7140d1310f6c68faa612986aecd9f35475783a366a7  docs/foundational/phase1f/evidence/live-controller/accounting-green.txt
868cb61f561726730fe848dc2454a555b3c18aff091c2cb8556cff40daea4a94  docs/foundational/phase1f/evidence/live-controller/original-probes-red.txt
4e1737ac081083b2f8411e1d6720166974b52818b9076203f7d1fb9376fec4e2  docs/foundational/phase1f/evidence/live-controller/original-probes-green.txt
7e9df770f452a3dc332976a63890da68aa1939e01422c9d1637aefe620cd580a  docs/foundational/phase1f/evidence/live-controller/original-usage-probes.py
0e6a876981c90020c5a54f51ac448c8410aae23b15539a8a87db99fcfc4bbd3f  .superpowers/sdd/00-plan/live-control-fix-report.md
c5329ee04fbdca07c8a775452f2c06e1076a1e00a4c81931cbd96e7ddaa196d1  .superpowers/sdd/00-plan/live-control-fix.patch
```
