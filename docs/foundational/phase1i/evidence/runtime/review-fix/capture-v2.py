#!/usr/bin/env python3
"""Capture final focused logs and inspectable witness metadata; no runtime imports."""
import collections
import gzip
import hashlib
import json
import pathlib
import sys

root = pathlib.Path.cwd()
base = root / "docs/foundational/phase1i/evidence/runtime"
manifest = json.loads((base / "fixtures-qualification-v2.json").read_text())
assert not [p for p, h in manifest["source_sha256"].items() if hashlib.sha256((root / p).read_bytes()).hexdigest() != h]

def read(path):
    with gzip.open(path) as stream:
        return json.load(stream)

def folder(profile, test):
    return base / profile / hashlib.sha256(json.dumps(test, separators=(",", ":")).encode()).hexdigest()[:12]

for profile in sys.argv[1:]:
    log = pathlib.Path("/tmp/alan-phase1i-" + profile + ".jsonl").read_bytes()
    events = [json.loads(line) for line in log.splitlines()]
    assert events[-1]["Action"] == "pass", events[-1]
    names = [e["Test"] for e in events if e.get("Action") == "pass" and "/" in e.get("Test", "")]
    assert sorted(names) == sorted(manifest["expected_leaf_names"]) and len(names) == 295
    summary = {
        "profile": profile, "exit": 0, "elapsed_seconds": events[-1]["Elapsed"],
        "selected_pass_names": names, "leaf_count": len(names),
        "leaf_counts": dict(collections.Counter(n.split("/")[0] for n in names)),
        "failures": [e.get("Test") for e in events if e.get("Action") == "fail"],
        "skips": [e.get("Test") for e in events if e.get("Action") == "skip"],
        "race_reports": [e.get("Output") for e in events if "DATA RACE" in e.get("Output", "")],
        "source_manifest": "fixtures-qualification-v2.json", "source_drift": [],
        "probe_memberships": [], "closures": [], "recoveries": [], "root_success_fences": [],
    }
    for test in names:
        path = folder(profile, test)
        if test.startswith("TestSubstitutionProbes/") and not test.endswith(("wrong-id", "extra-helper")):
            result = read(path / "result.json.gz")
            normalized = result.get("normalized", result.get("decoded"))
            assert normalized["EvidenceChecked"] and "actual probe graph" in normalized["ModelMembership"]
            assert not result["probe_model"]["Counterexamples"]
            summary["probe_memberships"].append({"test": test, "membership": normalized["ModelMembership"], "observed": normalized["Context"]})
        elif test.startswith("TestSubstitutionClosure/"):
            result = read(path / "after.json.gz")
            assert result["normalized"]["EvidenceChecked"] and result["normalized"]["Context"]["Closed"]
            summary["closures"].append({"test": test, "observed": result["normalized"]["Context"], "charges": result["inspection"]["Roots"][0]["Charges"]})
        elif test.startswith("TestSubstitutionRecovery/"):
            result = read(path / "after.json.gz")
            starts = [read(p) for p in path.glob("start-*.json.gz")]
            exits = [read(p) for p in path.glob("exit-*.json.gz")]
            assert len(starts) == 3 and len(exits) == 3
            assert len({s["pid"] for s in starts}) == 3
            assert all(s["race_instrumented"] == profile.startswith("race") for s in starts)
            assert result["after_root_epoch"] > result["before_root_epoch"]
            assert sum(e["signal"] == "killed" for e in exits) == 1
            summary["recoveries"].append({"test": test, "before_epoch": result["before_root_epoch"], "after_epoch": result["after_root_epoch"], "starts": starts, "exits": [{k: v for k, v in e.items() if k != "output"} for e in exits]})
        elif test.startswith("TestSubstitutionRootSuccessFence/"):
            result = read(path / "counterexample.json.gz")
            observed = result["normalized"]["Context"]
            assert result["normalized"]["EvidenceChecked"] and not observed["Closed"]
            assert result["boundary_census"]["Counterexamples"]
            summary["root_success_fences"].append({"test": test, "observed": observed, "optimistic_membership": result["normalized"]["ModelMembership"], "boundary_counterexample_codes": [c["Code"] for c in result["boundary_census"]["Counterexamples"]]})
    assert len(summary["probe_memberships"]) == 24 and len(summary["closures"]) == 9
    assert len(summary["recoveries"]) == 4 and len(summary["root_success_fences"]) == 2
    with gzip.open(base / (profile + ".jsonl.gz"), "wb") as stream:
        stream.write(log)
    (base / (profile + "-summary.json")).write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps({"profile": profile, "leaf_count": len(names), "elapsed_seconds": summary["elapsed_seconds"], "probe_memberships": 24, "cancelled_roots_closed_true": 9, "source_drift": []}))
