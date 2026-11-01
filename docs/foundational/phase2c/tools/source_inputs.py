"""Check the complete Go/native/module inventory for the release gates. Report root-run deltas separately."""
from hashlib import sha256
import gzip
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
E = ROOT / "docs/foundational/phase2c/evidence"
SUFFIX = (".go", ".mod", ".sum", ".c", ".h")
SELECTED = ("executor", "consumer", "prior-world", "histories", "surrounding", "public")


def record(label):
    with gzip.open(E / "commands" / (label + ".json.gz")) as f:
        r = json.load(f)
    assert r["exit"] == 0, label
    return r


def inputs(r):
    return {p: h for p, h in r["inputs"].items() if p.endswith(SUFFIX)}


def main():
    paths = subprocess.check_output(["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=ROOT).decode().split("\0")
    current = {p: sha256((ROOT/p).read_bytes()).hexdigest() for p in paths if p.endswith(SUFFIX)}
    labels = [f"qualified-{name}-{mode}" for name in SELECTED for mode in ("normal", "race")]
    for label in labels + ["qualified-replay", "qualified-vet"]:
        captured = inputs(record(label))
        assert captured == current, (label, sorted(p for p in captured.keys() | current.keys()
                                                  if captured.get(p) != current.get(p)))
    for mode in ("normal", "race"):
        r = record("qualified-public-" + mode)
        driver = "examples/reconciliation/consumer/qualify.py"
        assert r["inputs"][driver] == sha256((ROOT/driver).read_bytes()).hexdigest()
    old = inputs(record("root-suite"))
    delta = {p: {"ordinary_root": old.get(p), "focused_final": current.get(p)}
             for p in sorted(old.keys() | current.keys()) if old.get(p) != current.get(p)}
    print(json.dumps({"qualified_inputs": len(current), "matched_exactly": labels + ["qualified-replay", "qualified-vet"],
                      "root_run_input_count": len(old), "root_run_differences": delta}, indent=2))


if __name__ == "__main__":
    main()
