"""Compare executed normal/race Go test leaves and distinguish helper entry points."""
import gzip
import json
from pathlib import Path
import re

E = Path(__file__).resolve().parents[1] / "evidence"


def tests(label):
    if label.startswith("model/"):
        output = (E / (label + ".txt")).read_text()
    else:
        with gzip.open(E / "commands" / (label + ".json.gz"), "rt") as handle:
            command = json.load(handle)
        assert command["exit"] == 0, label
        output = command["output"]
    statuses = re.findall(r"^\s*--- (PASS|SKIP|FAIL): (\S+)", output, re.M)
    if output.startswith("{"):
        events = (json.loads(line) for line in output.splitlines())
        statuses = [(e["Action"].upper(), e["Test"]) for e in events
                    if "Test" in e and e["Action"] in ("pass", "skip", "fail")]
    # Process diagnostics are JSON-encoded strings on a single log line.
    # Only the top-level Go harness produces an actual line-level PASS here.
    assert statuses and all(status == "PASS" for status, _ in statuses), label
    names = sorted(name for _, name in statuses)
    return [name for name in names if not any(other.startswith(name + "/") for other in names)]


groups = {
    "model": ("model/normal", "model/race"),
    "authority": ("authority-normal-reviewfix", "authority-race-reviewfix"),
    "closed_withholding": ("withholding-normal", "withholding-race"),
    "process": ("process-normal-reviewfix", "process-race-reviewfix"),
    "parent": ("parent-normal", "parent-race-final"),
    "consumer": ("consumer-normal-final", "consumer-race-final"),
    "recorder_and_lifecycle": ("inherit-normal", "inherit-race"),
    "inherited_endpoint": ("endpoint-inherited-normal-reviewfix", "endpoint-inherited-race-reviewfix"),
    "runtime_and_resource": ("runtime-resource-normal", "runtime-resource-race"),
}
if __name__ == "__main__":
    result = {}
    for group, (normal, race) in groups.items():
        a, b = tests(normal), tests(race)
        assert a == b, (group, a, b)
        result[group] = dict(normal=normal, race=race, count=len(a), leaves=a, skips=[])
    assert "TestPhase2ExecutorProcess" in result["process"]["leaves"]
    print(json.dumps(result, indent=2))
