"""Capture one explicitly selected output-blind reference construction command."""
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

root = Path(__file__).resolve().parents[5]
label, *argv = sys.argv[1:]
destination = Path(__file__).parent / (label + ".json.gz")
if destination.exists():
    raise SystemExit("refusing to overwrite " + str(destination))
env = {k: v for k, v in os.environ.items() if not k.startswith("ALAN_") and k != "OPENAI_API_KEY"}
settings = dict(GOTOOLCHAIN="local", CGO_ENABLED="1", GOPROXY="off", GOSUMDB="off",
                GOFLAGS="-mod=readonly", GOCACHE="/private/tmp/alan-phase1n-model-go-build")
env.update(settings)
paths = sorted((root / "research/substitution").glob("*.go"))
paths += [root / p for p in ["go.mod", "go.sum", "docs/foundational/phase1n/01-contract.md"]]
sources = {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}
started = time.time()
run = subprocess.run(argv, cwd=root, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
events = []
for line in run.stdout.splitlines():
    try:
        event = json.loads(line)
    except json.JSONDecodeError:
        continue
    if event.get("Test") and event.get("Action") in ("pass", "fail", "skip"):
        events.append({k: event[k] for k in ("Action", "Package", "Test") if k in event})
names = [e["Test"] for e in events]
leaves = [e for e in events if not any(n.startswith(e["Test"] + "/") for n in names)]
record = dict(argv=argv, cwd=str(root), environment=settings,
              stripped_environment=["ALAN_*", "OPENAI_API_KEY"],
              sources=sources, elapsed_seconds=time.time() - started,
              exit=run.returncode, stdout=run.stdout, stderr=run.stderr,
              test_events=events, leaves=leaves,
              input_boundary="approved contract plus named historical semantics; new construction and outputs unopened")
spool = Path("/private/tmp/alan-phase1n-model-capture")
spool.mkdir(exist_ok=True)
staged = spool / destination.name
if staged.exists():
    raise SystemExit("refusing to overwrite staged " + str(staged))
with gzip.GzipFile(staged, "wb", mtime=0) as output:
    output.write(json.dumps(record, sort_keys=True, indent=2).encode() + b"\n")
destination.write_bytes(staged.read_bytes())
print(json.dumps(dict(label=label, exit=run.returncode, leaves=len(leaves),
                     record=str(destination.relative_to(root)))))
sys.exit(run.returncode)
