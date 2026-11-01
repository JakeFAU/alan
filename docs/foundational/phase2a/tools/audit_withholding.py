#!/usr/bin/env python3
"""Supplemental independent A5 audit of the two additional closed-source schedules."""
import hashlib
import json
from pathlib import Path

from audit_observers import EVIDENCE, effect, one, projection, read, run, selected


def verify(x, mode):
    adapter = x["route"] == "adapter"
    route = "adapter" if adapter else "external"
    conditions = x["endpoint_argv"]
    assert x["race"] == (mode == "race")
    assert conditions[conditions.index("--response-mode") + 1] == ("normal" if adapter else "drop-after-commit")
    stages = [x[k]["Tables"] for k in ("before", "closed", "after")]
    before, closed, after = stages
    op = one(before["intents"], subject="client")["id"]
    assert projection(before, "client")["Status"] == "open"
    assert projection(closed, "client") == projection(after, "client")
    assert projection(after, "client")["Status"] == "cancelled"
    assert not before["lifecycle_results"]
    (decision_row,) = closed["lifecycle_results"]
    assert closed["lifecycle_results"] == after["lifecycle_results"]
    decision = json.loads(decision_row["result"])
    request = decision["Request"]
    assert decision["Accepted"] and decision["Reason"] == "closed"
    assert request["Intent"]["Run"] == {"Subject": "operator", "Generation": 1}
    assert request["Intent"]["Target"] == "client" and request["Operation"] == "CancelRun"
    assert request["ExpectedRevision"] == request["SourceRevision"] == 1
    one(closed["permissions"], source="operator", target="client", kind="lifecycle", operation="CancelRun", revision=1)
    assert decision["Before"]["Projection"] == projection(before, "client")
    assert decision["After"]["Projection"] == projection(closed, "client")
    assert decision["Before"]["Run"] == decision["After"]["Run"] == {"Subject": "client", "Generation": 1}
    a, au, wire = effect(before, route, op)
    assert a["Allowed"] and au["Allowed"]
    if adapter:
        assert a["Prepared"]["Model"] == "unknown"
    for t in stages:
        assert len(t[route + "_attempts"]) == 1 and effect(t, route, op) == (a, au, wire)
        assert run(t, "client")["sequence"] == 1
        assert not t["external_receipts"] and not t["adapter_results"]
        assert not any(row["intent_id"] == op for row in t["settlements"]) and not selected(t, op)
    assert x["journal_after"] == x["journal_before"] and x["world_before"] == x["world_after"]
    post, commit = [json.loads(line) for line in x["journal_after"].splitlines()]
    assert post["Event"] == "ingress" and (post["Method"], post["Path"]) == ("POST", "/apply")
    assert post["Body"] == wire and json.loads(wire)["Key"] == x["key"]
    assert post["Contact"] == commit["Contact"] == 1 and post["PID"] == commit["PID"] == x["endpoint_pid"]
    assert commit["Event"] == "committed" and x["world_after"]["Contacts"] == 1
    (receipt,) = x["world_after"]["Applications"]
    (request,) = x["world_after"]["Requests"]
    assert commit["Receipt"] == receipt and receipt["Key"] == request["Key"] == x["key"]
    assert receipt["Profile"] == request["Profile"] == ("idempotent" if adapter else "non-idempotent")
    assert receipt["Mutation"] == 1 and receipt["Total"] == request["Delta"] == (1 if adapter else 5)
    missing = "UnknownAdapters" if adapter else "UnknownExternal"
    assert x["first_report"][missing] == x["second_report"][missing] == [op]
    assert x["second_report"]["Quiescent"]


if __name__ == "__main__":
    if not __debug__:
        raise SystemExit("Python assertions must be enabled")
    repo = Path(__file__).resolve().parents[4]
    for mode in ("normal", "race"):
        cmd = read(EVIDENCE / "commands" / f"withholding-{mode}.json.gz")
        assert cmd["exit"] == 0 and ("-race" in cmd["argv"]) == (mode == "race")
        for path in ("internal/durabletest/phase2a_withholding_test.go", "internal/executor/adapters.go",
                     "internal/executor/external.go", "cmd/alan-resource/main.go", "examples/effectboundary/adapter.go"):
            assert cmd["inputs"][path] == hashlib.sha256((repo / path).read_bytes()).hexdigest()
        cases = [read(f) for f in (EVIDENCE / f"withholding-{mode}").glob("*.json.gz")]
        assert sorted(x["route"] for x in cases) == ["adapter", "non-idempotent"]
        for x in cases:
            verify(x, mode)
        print(f"{mode}: 2 actual closures, 2 first POSTs/applications, 0 new ingresses/results/selections")
