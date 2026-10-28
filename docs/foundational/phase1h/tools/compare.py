#!/usr/bin/env python3
"""Exact old semantic comparisons against actual H-source collected records."""

import gzip, importlib.util, json, os, sys
from pathlib import Path
from common import ROOT, DOC, inventory, source_id
from semantic_compare import compare_deterministic

mode = sys.argv[1]
assert mode in ("default", "normal", "race")
out = Path(os.environ.get("ALAN_H_OUT", str(DOC / "evidence/task3/final"))) / (
    "final-" + mode
)
spec = importlib.util.spec_from_file_location(
    "comparison", ROOT / "docs/foundational/phase1e/tools/comparison.py"
)
c = importlib.util.module_from_spec(spec)
spec.loader.exec_module(c)
meta = json.loads((out / "command.json").read_text())
assert meta["ExitCode"] == 0 and meta["SourceUnchanged"]
current = meta["SourceID"]
base = json.loads(
    (ROOT / "docs/foundational/phase1e/source-manifest.json").read_text()
)["source_sha256"]
old = c.rows(ROOT / f"docs/foundational/phase1e/evidence/final-{mode}.records.jsonl.gz")
new = c.rows(out / "records.jsonl.gz")
go = [
    json.loads(x)
    for x in gzip.decompress((out / "stdout.gz").read_bytes()).splitlines()
]
passed = {
    e["Test"]
    for e in go
    if e.get("Action") == "pass"
    and e.get("Package") == "github.com/JakeFAU/alan/research/semantics"
    and "Test" in e
}
results = []
for marker, count in (
    ("QUALIFICATION", 15),
    ("COMPOSITION", 20),
    ("SUPERVISION", 32),
    ("CONFORMANCE", 6),
):
    a = [r for r in old if r["Marker"] == marker]
    b = [r for r in new if r["Marker"] == marker]
    c.require(len(a) == len(b) == count, "marker count")
    c.require(
        all(r["SourceID"] == base and r["Mode"] == mode for r in a),
        "historical association",
    )
    c.require(
        all(r["SourceID"] == current and r["Mode"] == mode for r in b),
        "actual H source association",
    )
    c.require(
        [(r["Package"], r["Test"]) for r in a]
        == [(r["Package"], r["Test"]) for r in b],
        "ordered test origins",
    )
    x = [r["Evidence"] for r in a]
    y = [r["Evidence"] for r in b]
    if marker == "CONFORMANCE":
        result = c.compare_conformance(x, y, passed)
    elif marker == "QUALIFICATION":
        c.require(x == y, "exact ordered payloads")
        result = dict(Marker=marker, Records=count, ExactOrderedPayloads=True)
    else:
        result = compare_deterministic(marker, x, y, ROOT)
    results.append(result)
report = dict(
    SourceID=current,
    ComparatorSourceID=source_id(inventory()),
    BaselineSource=base,
    Mode=mode,
    Comparisons=results,
    Scope="Actual H records: exact15 qualification payloads, ordered composition/supervision behavior after independently validating both provenance manifests and the sole pinned upstream bridge edit, and existing six public conformance projections/model streams. Not historical raw-metadata equality or historical-source execution.",
)
path = out / "comparison.json"
assert not path.exists()
path.write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps(report, indent=2))
