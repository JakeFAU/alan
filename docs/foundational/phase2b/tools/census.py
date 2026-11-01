"""Compare actual final Go harness leaves, skips and outcomes in both regimes."""
import gzip
import json
from pathlib import Path
import re

EVIDENCE = Path(__file__).resolve().parents[1] / "evidence"
GROUPS = ("model", "world", "composition", "consumer", "core", "endpoint", "public", "bridge", "direct")

def leaves(group, mode):
    if group == "model":
        output = (EVIDENCE / "model" / (mode + ".txt")).read_text()
    else:
        with gzip.open(EVIDENCE / "commands" / (group + "-final-" + mode + ".json.gz")) as f:
            record = json.load(f)
        assert record["exit"] == 0, (group, mode)
        output = record["output"]
    items = re.findall(r"^\s*--- (PASS|SKIP|FAIL): (\S+)", output, re.M)
    assert items and all(status != "FAIL" for status, _ in items), group
    names = {name for _, name in items}
    assert len(names) == len(items), (group, "duplicate harness names")
    return {name: status for status, name in sorted(items, key=lambda item: item[1])
            if not any(other.startswith(name + "/") for other in names)}

def census():
    result = {}
    all_leaves = set()
    for group in GROUPS:
        normal, race = (leaves(group, mode) for mode in ("normal", "race"))
        assert normal == race, (group, "different outcomes", normal.keys() ^ race.keys())
        if group != "direct":
            assert not all_leaves.intersection(normal), (group, "inter-group overlap")
            all_leaves.update(normal)
        else:
            assert set(normal).issubset(result["core"]["leaves"]), "direct controls repeated within core"
        result[group] = {"leaves": list(normal), "count": len(normal),
                         "skips": [name for name in normal if normal[name] == "SKIP"]}
    return {"groups": result, "unique_leaves_per_regime": len(all_leaves),
            "skips_per_regime": sum(len(result[g]["skips"]) for g in GROUPS if g != "direct")}

if __name__ == "__main__":
    print(json.dumps(census(), indent=2))
