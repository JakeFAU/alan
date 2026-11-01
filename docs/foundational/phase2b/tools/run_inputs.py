"""Verify the source identities used by the final Go campaigns and their children."""
import gzip
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
EVIDENCE = ROOT / "docs/foundational/phase2b/evidence"
BRIDGE = "research/semantics/runtime_conformance_test.go"
BASE = "72e406b73b05a3ed4a6e8b37801ab6e0f0a73209"
BUILD = (".go", ".mod", ".sum", ".c", ".h")


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    original = sha(subprocess.check_output(["git", "show", BASE + ":" + BRIDGE], cwd=ROOT))
    paths = subprocess.check_output(
        ["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=ROOT
    ).decode().split("\0")
    final = {p: sha((ROOT / p).read_bytes()) for p in paths if p.endswith(BUILD)}
    assert final[BRIDGE] != original

    def check(inputs, earlier, label):
        actual = {p: h for p, h in inputs.items() if p.endswith(BUILD)}
        assert BRIDGE in actual and set(actual) <= set(final), label
        mismatches = {p for p, h in actual.items() if final[p] != h}
        assert mismatches == ({BRIDGE} if earlier else set()), (label, mismatches)
        if earlier:
            assert actual[BRIDGE] == original, label
        return len(actual)

    groups = ("world", "composition", "direct", "core", "public", "endpoint", "consumer", "bridge")
    for label in [g + "-final-" + m for g in groups for m in ("normal", "race")] + ["root-suite-final"]:
        with gzip.open(EVIDENCE / "commands" / (label + ".json.gz")) as f:
            record = json.load(f)
        assert record["exit"] == 0, label
        earlier = not label.startswith(("bridge-", "root-suite-"))
        assert check(record["inputs"], earlier, label) == len(final), label

    for mode in ("normal", "race"):
        files = [p for group in ("process", "composition-process")
                 for p in (EVIDENCE / (group + "-" + mode)).glob("*-sources-*.json")]
        assert files, mode
        for p in files:
            count = check(json.loads(p.read_bytes()), True, p)
            assert count == 408, (p, count)
        print(f"{mode}: {len(files)} child source inventories verified, 408 paths each")

    print(f"{len(final)} Go/native/module files in each of 17 successful command records")
    print("Earlier focused runs differ only in the uncompiled conformance fixture", BRIDGE, original)
    print("The bridge and full-suite runs use its delivered identity", final[BRIDGE])


if __name__ == "__main__":
    main()
