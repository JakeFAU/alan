"""Count paired terminal Go harness leaves and the separate real-CLI experiments."""
import gzip
import json
from pathlib import Path
import re

COMMANDS = Path(__file__).resolve().parents[1] / "evidence/commands"
GROUPS = {
    "model, endpoint, authoring and public facade": "qualified-surrounding-{}",
    "process, runtime controls and final guard": "qualified-histories-{}",
    "separate public module": "qualified-consumer-{}",
    "executor, replay and historical Q": "qualified-executor-{}",
    "prior process and composition": "qualified-prior-world-{}",
}


def command(name):
    with gzip.open(COMMANDS / (name + ".json.gz")) as stream:
        record = json.load(stream)
    assert record["exit"] == 0, name
    return record["output"]


def leaves(name):
    output = command(name)
    results = re.findall(r"^\s*--- (PASS|FAIL|SKIP): (\S+)", output, re.M)
    assert results and all(status == "PASS" for status, _ in results), name
    names = {test for _, test in results}
    assert len(names) == len(results), name
    return {test for test in names if not any(other.startswith(test + "/") for other in names)}


def census():
    groups, seen = {}, set()
    helpers = {"TestLedgerCLIHelper", "TestTask4FaultProcess"}
    for group, pattern in GROUPS.items():
        a, b = pattern if isinstance(pattern, tuple) else [pattern.format(mode) for mode in ("normal", "race")]
        tests = leaves(a)
        assert tests == leaves(b), (group, tests ^ leaves(b))
        omitted = tests & helpers
        tests -= helpers
        assert not (tests & seen), (group, sorted(tests & seen))
        seen |= tests
        groups[group] = {"leaves": sorted(tests), "count": len(tests), "helper_only": sorted(omitted)}
    cli = ["positive", "immutable", "read-only"]
    for mode in ("normal", "race"):
        output = command("qualified-public-" + mode)
        assert all(name + ": verified" in output for name in cli)
    return {"go_leaves_per_regime": len(seen), "go_skips": 0,
            "independent_cli_histories_per_regime": cli, "groups": groups}


if __name__ == "__main__":
    print(json.dumps(census(), indent=2))
