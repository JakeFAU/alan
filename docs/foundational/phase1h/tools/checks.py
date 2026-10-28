#!/usr/bin/env python3
"""Require named process boundaries and causal negative controls in actual output."""

import gzip, json, sys
from pathlib import Path

required = {
    "TestProcessAdaptiveCuts",
    "TestAdaptiveCapacityAndOutOfOrder",
    "TestAdaptiveRootDenial",
    "TestAdaptiveUnknownSibling",
    "TestAdaptiveClosureAfterDecision",
    "TestAdaptiveClosureLateTransportCompanion",
    "TestAdaptiveReadOnlyReplay",
    "TestFaultyAdaptiveApplicationOracle",
}
folder = Path(sys.argv[1])
meta = json.loads((folder / "command.json").read_text())
assert meta["ExitCode"] == 0 and meta["SourceUnchanged"]
events = [
    json.loads(x)
    for x in gzip.decompress((folder / "stdout.gz").read_bytes()).splitlines()
]
passed = {e.get("Test") for e in events if e.get("Action") == "pass"}
assert required <= passed, required - passed
records = [
    json.loads(x)
    for x in gzip.decompress((folder / "records.jsonl.gz").read_bytes()).splitlines()
]
assert all(r["SourceID"] == meta["SourceID"] for r in records)
a = [r["Evidence"] for r in records if r["Marker"] == "ADAPTIVE"]
kinds = {r.get("kind") for r in a}
expected = {
    "crash_cut",
    "recovered",
    "capacity_held",
    "out_of_order_completed",
    "root_denial",
    "unknown_failed_sibling",
    "closure_after_decision",
    "closure_late_transport_companion",
    "read_only_replay",
    "faulty_application_oracle",
}
assert expected <= kinds, expected - kinds
fault = [r for r in a if r.get("kind") == "faulty_application_oracle"]
assert (
    len(fault) == 1
    and fault[0]["forbidden_admitted"] == [10, 11, 31]
    and "rejected" in fault[0]["application_verdict"]
)
assert len([r for r in a if r.get("kind") == "recovered"]) == 6
print(
    json.dumps(
        dict(
            SourceID=meta["SourceID"],
            RequiredTests=sorted(required),
            Kinds=sorted(kinds),
            Result="PASS",
        ),
        indent=2,
    )
)
