# Live-controller accounting correction

Completed 2026-10-26 in the assigned worktree. Source is frozen for independent re-review. This correction changes only `docs/foundational/phase1f/tools/live_control.py` and `test_live_control.py`, with scoped evidence below and this report. No product/application/runtime source, credentials, `.env`, paid calls, network/provider calls, subagents, commits, or original retained snapshots were touched. Existing `.before.txt` files and original review patch remain intact. An initial sandbox write was denied because this worktree is outside the writable roots; approved scoped tool escalations then allowed the authorized changes.

## Correction

Usage can reduce a reservation only after InputTokens, OutputTokens and TotalTokens are exact Python integers (bool excluded), nonnegative, consistent (`total = input + output`), and nonzero for a successful result. Cache read/write counts are exact nonnegative integers and their sum cannot exceed input. Non-object accounting/detail shapes are malformed. Absent/incomplete counts retain the full reserve with `unknown-cost-reservation-retained`; malformed accounting retains it with `malformed-usage-reservation-retained`. Both report `cost_reason` and preserve original request/response IDs when a result exists. Invalid accounting clears a prior discount/estimate so it cannot silently fund another request.

Validated usage preserves the existing cost calculation. A direct valid-cache case verifies $0.000373 estimated token cost and $0.000875 conservative charge for 100 input (30 cache-read, 10 cache-write), 20 output. Cache writes remain charged at the supplied maximum $5/M rate. The original actual-usage fixture still reconciles to $0.001686. Provider-reported usage is explicitly an assumption rather than invoice proof.

The summary uses the tested `unresolved_reservations()` helper and includes released/unknown/malformed unresolved statuses. Reservation write/release ordering is unchanged. A temporary-file regression verifies a model-validation exception leaves the durable full reservation and in-memory full charge intact. Other errors still propagate before the final reconciliation persistence, leaving the previously durable reserve intact.

## Red/green verification

Before editing controller source, added regression tests and ran the exact command from the brief, from the repository root:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s docs/foundational/phase1f/tools -p test_live_control.py -v
```

`evidence/live-controller/accounting-red.txt` retains complete output: 20 tests, FAILED (27 failing subcases, 23 errors). The original nine tests passed. Failures show the actual erroneous budget releases; errors show the old implementation's missing reason/helper contract or exceptions on malformed accounting types. There were no import/syntax/setup errors. After correction, `accounting-green.txt` retains complete fresh output: 20 tests, OK, exit 0. Existing nine tests are unchanged. Eleven added tests cover the two original review cases, bool/fraction/string/negative tokens and cache counts, excessive combined cache counts, zero billable totals, malformed shapes, missing usage, stale discount removal and subsequent budget rejection, valid cache calculation/IDs, summary totals, and durable error behavior. No gofmt applies to Python stdlib code.

The review's exact original usage-probe inputs/candidate IDs are retained in `evidence/live-controller/original-usage-probes.py`; its accounting portion was run before and after correction:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 docs/foundational/phase1f/evidence/live-controller/original-usage-probes.py
```

`original-probes-red.txt`: 0/0/-1 frees the $0.20 reserve to $0; 1/1/666 frees it to $0.00001875. `original-probes-green.txt`: both retain $0.20 with explicit malformed reasons and `probe` request/response IDs. The fd probe was not repeated, per brief. All four exact output logs and the probe script are retained.

## Concerns and scope limits

No remaining known issue in the reviewed accounting finding. Internally consistent provider metadata cannot establish invoice truth; the existing pricing/byte-bound assumptions remain local assumptions. This change does not alter budget authorization, model/tier policy, boundary validation, transport behavior, replay configuration, or credential flow. The retained actual-boundary negative controls and their product code were left unchanged; broader application verification belongs to the parent/application observer. The controller tests use immutable retained inspection data plus temporary local files and do not exercise a complete controller/provider run or crash-injection sequence. Parent must obtain the requested independent re-review before paid calls.

## SHA-256

Original source hashes match the review and the `.before.txt` copies. Current source and retained evidence:

```text
5e1cb22163408904e25d19d8615ec45a9b36fb5efea236075f3d2d37200a8ade  docs/foundational/phase1f/tools/live_control.py
d9043ffc8c2334fecc53e2eb64a5828a58d31217b5b74fdaafce840822a2dfff  docs/foundational/phase1f/tools/test_live_control.py
1f7d8796fa9d37a015e78fa68310fc6fce1c51e94b3bf9332268efe42e87e682  docs/foundational/phase1f/evidence/live-controller/live_control.py.before.txt
e230162d4cd44c10080567f019376be62212bdfafaa60e73f4ca00c962e15a5b  docs/foundational/phase1f/evidence/live-controller/test_live_control.py.before.txt
d48c9861a10d628ec08a8bcad0711e2312c35b8d71bb97a72c4be1898ecb420e  docs/foundational/phase1f/evidence/live-controller/accounting-red.txt
868cb61f561726730fe848dc2454a555b3c18aff091c2cb8556cff40daea4a94  docs/foundational/phase1f/evidence/live-controller/original-probes-red.txt
c412ee20aa52533d3e85f7140d1310f6c68faa612986aecd9f35475783a366a7  docs/foundational/phase1f/evidence/live-controller/accounting-green.txt
4e1737ac081083b2f8411e1d6720166974b52818b9076203f7d1fb9376fec4e2  docs/foundational/phase1f/evidence/live-controller/original-probes-green.txt
7e9df770f452a3dc332976a63890da68aa1939e01422c9d1637aefe620cd580a  docs/foundational/phase1f/evidence/live-controller/original-usage-probes.py
```
