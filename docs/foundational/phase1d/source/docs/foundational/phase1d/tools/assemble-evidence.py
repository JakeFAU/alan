#!/usr/bin/env python3
"""Reconstruct Phase 1D evidence from retained Go JSON logs; no tests or mutations."""
from pathlib import Path
import collections
import gzip
import hashlib
import json
import re

ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / "docs/foundational/phase1d"
SOURCE = json.loads((DOC / "source-manifest.json").read_text())["source_sha256"]
PATTERN = re.compile(r"^\s*(?:[^\s:]+:\d+:\s*)?(QUALIFICATION|COMPOSITION|SUPERVISION|CONFORMANCE|DURABLE) (.*)$")


def records(mode):
    buffers, result = {}, []
    raw = gzip.decompress((DOC / f"evidence/final-{mode}.go-test.jsonl.gz").read_bytes())
    meta = json.loads((DOC / f"evidence/final-{mode}.command.json").read_text())
    assert hashlib.sha256(raw).hexdigest() == meta["RawUncompressedSHA256"]
    assert meta["ExitCode"] == 0 and meta["QualificationSourceID"] == SOURCE
    for line in raw.decode().splitlines():
        event = json.loads(line)
        if event.get("Action") != "output":
            continue
        key = (event.get("Package", ""), event.get("Test", ""))
        value = buffers.get(key, "") + event.get("Output", "")
        while "\n" in value:
            one, _, value = value.partition("\n")
            match = PATTERN.match(one)
            if match:
                result.append({"QualificationSourceID": SOURCE, "Mode": mode,
                               "Package": key[0], "Test": key[1], "RecordIndex": len(result) + 1,
                               "Marker": match[1], "Evidence": json.loads(match[2]),
                               "RawPayload": match[2]})
        buffers[key] = value
    assert not any(v.strip() for v in buffers.values())
    assert dict(collections.Counter(r["Marker"] for r in result)) == meta["Markers"]
    return meta, result


output, comparisons, audits = [], [], []
for mode in ("normal", "race"):
    meta, rows = records(mode)
    output.append(meta)
    for phase, marker in (("phase1", "QUALIFICATION"), ("phase1b", "COMPOSITION"), ("phase1c", "SUPERVISION")):
        selected = [r for r in rows if r["Marker"] == marker]
        saved_bytes = (ROOT / f"docs/foundational/{phase}/qualification.jsonl").read_bytes()
        current_bytes = ("\n".join(r["RawPayload"] for r in selected) + "\n").encode()
        saved = [json.loads(line) for line in saved_bytes.splitlines()]

        def behavior(values):
            return [{k: v for k, v in r.items() if k != "SourceID"}
                    for r in values if r.get("Name") != "source_manifest"]

        current = [r["Evidence"] for r in selected]
        assert behavior(saved) == behavior(current), (phase, mode)
        if phase == "phase1":
            assert saved_bytes == current_bytes
        comparison = {"Kind": "current_research_comparison", "QualificationSourceID": SOURCE,
                      "Mode": mode, "Phase": phase, "Records": len(current),
                      "BehaviorRecords": len(behavior(current)), "BehaviorIdentical": True,
                      "ByteIdentical": saved_bytes == current_bytes,
                      "Normalization": "omit source_manifest record and top-level SourceID only",
                      "SavedSHA256": hashlib.sha256(saved_bytes).hexdigest(),
                      "CurrentSHA256": hashlib.sha256(current_bytes).hexdigest()}
        comparisons.append(comparison)
        output.append(comparison)
    durable = [r for r in rows if r["Marker"] == "DURABLE"]
    processes = [r for r in durable if r["Evidence"].get("type") == "process"]
    kills = [r for r in durable if r["Evidence"].get("exit_signal")]
    assert len(durable) == 87 and len(processes) == 61 and len(kills) == 16
    assert all(r["Evidence"]["exit_signal"] == "killed" for r in kills)
    assert all(r["Evidence"]["race_children"] == (mode == "race") for r in processes)
    faults = [r for r in durable if r["Evidence"].get("unchanged_oracle_failures")]
    assert len(faults) == 3
    conformance = [r for r in rows if r["Marker"] == "CONFORMANCE"]
    assert len(conformance) == 6 and all(r["Evidence"]["passed"] for r in conformance)
    audit = {"Kind": "evidence_audit", "QualificationSourceID": SOURCE, "Mode": mode,
             "DurableRecords": len(durable), "ProcessRecords": len(processes),
             "SIGKILLExits": len(kills), "ScenarioAndInputRecords": len(durable) - len(processes),
             "ChildRaceFlagsMatchMode": True, "DetectedFaultControls": len(faults),
             "ConformanceFixtures": len(conformance),
             "ConformanceCheckpoints": [len(r["Evidence"]["checkpoints"]) for r in conformance],
             "Scope": "Record inventory; independent barrier/ledger checks are implemented by tests and reviewed separately."}
    audits.append(audit)
    output.append(audit)
    output.extend({k: v for k, v in r.items() if k != "RawPayload"} for r in rows)

output.extend(json.loads(x) for x in (DOC / "evidence/final-static.jsonl").read_text().splitlines())
output.append({"Kind": "baseline_file_preservation", "QualificationSourceID": SOURCE,
               "Evidence": json.loads((DOC / "evidence/final-preservation.json").read_text())})
for name, values in (("qualification.jsonl", output), ("evidence/final-research-comparisons.jsonl", comparisons),
                     ("evidence/final-record-audit.jsonl", audits)):
    (DOC / name).write_text("".join(json.dumps(r, separators=(",", ":")) + "\n" for r in values))
print(json.dumps({"records": len(output), "source": SOURCE, "audits": audits}, indent=2))
