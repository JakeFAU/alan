"""Measure creation and causal recovery on two local runtime checkouts."""
import argparse
import json
import os
from pathlib import Path
import re
import statistics
import subprocess
import tempfile

cli = argparse.ArgumentParser(description=__doc__)
cli.add_argument("--baseline", type=Path, required=True)
cli.add_argument("--candidate", type=Path, required=True)
cli.add_argument("--fixture", type=Path, required=True)
cli.add_argument("--output", type=Path, required=True)
args = cli.parse_args()
versions = {"base": args.baseline.resolve(), "candidate": args.candidate.resolve()}
rows = []
with tempfile.TemporaryDirectory() as temp:
    for phase, order in enumerate((("base", "candidate"), ("candidate", "base"), ("base", "candidate"))):
        for label in order:
            root = versions[label]
            overlay = Path(temp) / (label + ".json")
            source = root / "runtime/release_create_bench_test.go"
            overlay.write_text(json.dumps({"Replace": {str(source): str(args.fixture.resolve())}}))
            command = ["go", "test", "-overlay", str(overlay), "-run", "^$", "-bench",
                       "^(BenchmarkReleaseCreation|BenchmarkCausalRecovery)$", "-benchmem",
                       "-benchtime=5x", "-count=1", "-cpu=1", "./runtime"]
            result = subprocess.run(command, cwd=root, env=dict(os.environ, GOWORK="off", GOTOOLCHAIN="local"),
                                    text=True, capture_output=True, check=True)
            found = 0
            for line in result.stdout.splitlines():
                match = re.fullmatch(r"(Benchmark\S+)\s+5\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op", line.strip())
                if match:
                    name, ns, size, allocations = match.groups()
                    rows.append(dict(arm=label, block=phase+1, name=name, ns=float(ns),
                                     bytes=int(size), allocations=int(allocations)))
                    found += 1
            if found != 11:
                raise SystemExit("expected 2 creation and 9 recovery rows:\n" + result.stdout)
summary={}
for name in sorted({r["name"] for r in rows}):
    summary[name]={}
    for label in versions:
        data = [r for r in rows if r["name"] == name and r["arm"] == label]
        summary[name][label]={k:statistics.median(r[k] for r in data) for k in ("ns","bytes","allocations")}
args.output.write_text(json.dumps(dict(rows=rows,median=summary),indent=2)+"\n")
