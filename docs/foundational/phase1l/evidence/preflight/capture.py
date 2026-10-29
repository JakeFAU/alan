#!/usr/bin/env python3
"""Capture the small construction feasibility checks; this is not an oracle."""
import base64
import datetime
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[5]
DEST = Path(__file__).resolve().parent
PINNED = dict(GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off",
              GOFLAGS="-mod=readonly", GOCACHE="/private/tmp/alan-phase1l-go-build")
ENV = dict(os.environ, **PINNED)
COMMANDS = []


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def run(argv):
    print("RUN", " ".join(argv), flush=True)
    start = time.monotonic()
    result = subprocess.run(argv, cwd=ROOT, env=ENV, capture_output=True, text=True)
    record = dict(argv=argv, cwd=str(ROOT), environment=PINNED,
                  exit=result.returncode, seconds=round(time.monotonic()-start, 3),
                  stdout=result.stdout, stderr=result.stderr)
    COMMANDS.append(record)
    print("EXIT", result.returncode, record["seconds"], flush=True)
    if result.returncode:
        raise RuntimeError(record)
    return result.stdout


def selected(snapshot):
    """Bind original handles to raw rows without reconstructing operation IDs."""
    tables = snapshot["Tables"]
    parent = next(row for row in tables["runs"] if row["subject"] == "root")
    projection = json.loads(parent["projection"])
    joins = {}
    for name, field in (("candidate", "candidate_operation"), ("s", "s_operation")):
        operation = projection["Data"][field]["Text"]
        intent = next(row for row in tables["intents"] if row["id"] == operation)
        admission = next(row for row in tables["composition_invocations"]
                         if row["operation_id"] == operation)
        publication = next(row for row in tables["composition_publications"]
                           if row["operation_id"] == operation)
        delivery = next(row for row in tables["composition_outcome_deliveries"]
                        if row["operation_id"] == operation)
        observation = next(row for row in tables["observations"]
                           if row["id"] == delivery["observation_id"])
        history = next(row for row in tables["history"]
                       if row["subject"] == "root"
                       and row["selected_observation"] == observation["id"])
        joins[name] = dict(handle_from_parent=operation, intent=intent,
                           admission=admission, publication=publication,
                           observation=observation, history=history)
    return dict(parent=parent, joins=joins, roots=snapshot["Roots"])


with tempfile.TemporaryDirectory(prefix="alan-phase1l-preflight-", dir="/private/tmp") as tmp:
    binary = str(Path(tmp)/"consumer")
    bundle = dict(started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  purpose="construction feasibility only", commands=COMMANDS, samples={})
    try:
        run(["go", "version"])
        run(["go", "test", "-json", "-count=1", "./examples/invocationjoin"])
        run(["go", "test", "-json", "-count=1", "-race", "./examples/invocationjoin"])
        prefix = ["go", "-C", "examples/invocationjoin/consumer"]
        run(prefix+["test", "-json", "-count=1", "-tags", "integration", "./..."])
        run(prefix+["test", "-json", "-count=1", "-race", "-tags", "integration", "./..."])
        run(["go", "vet", "./examples/invocationjoin"])
        run(prefix+["vet", "-tags", "integration", "./..."])
        run(prefix+["mod", "verify"])
        run(prefix+["build", "-o", binary, "."])
        run(prefix+["list", "-f", "{{join .Imports \"\\n\"}}", "."])
        bundle["binary_sha256"] = sha(Path(binary).read_bytes())
        for candidate in ("a", "b"):
            path = str(Path(tmp)/(candidate+".db"))
            sample = {}
            sample["artifacts"] = json.loads(run([binary, "artifacts", "-candidate", candidate]))
            sample["init"] = json.loads(run([binary, "init", "-db", path, "-candidate", candidate]))
            sample["run"] = json.loads(run([binary, "run", "-db", path]))
            sample["inspection"] = json.loads(run([binary, "inspect", "-db", path]))
            sample["replay"] = json.loads(run([binary, "replay", "-db", path]))
            sample["selected_raw_rows"] = selected(sample["inspection"])
            bundle["samples"][candidate] = sample
    finally:
        sources = sorted((ROOT/"examples/invocationjoin").rglob("*.go"))
        sources += [ROOT/"examples/invocationjoin/consumer/go.mod",
                    ROOT/"examples/invocationjoin/consumer/go.sum",
                    ROOT/"examples/invocationjoin/README.md", Path(__file__).resolve()]
        originals = ["examples/substitution/a/candidate.go", "examples/substitution/b/b.go",
                     "examples/substitution/deployment.go", "examples/substitution/protocol/protocol.go",
                     "examples/join/deployment.go", "author/invocations.go", "author/compile.go",
                     "compose/collection.go", "go.mod", "go.sum"]
        sources += [ROOT/p for p in originals]
        source_hashes = {str(p.relative_to(ROOT)): sha(p.read_bytes()) for p in sources}
        bundle["source_hashes"] = source_hashes
        bundle["base_commit"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
        bundle["original_source_unchanged_from_base"] = {
            p: sha(subprocess.check_output(["git", "show", "HEAD:"+p], cwd=ROOT)) == source_hashes[p]
            for p in originals}
        raw = json.dumps(bundle, separators=(",", ":")).encode()+b"\n"
        archive = gzip.compress(raw, mtime=0)
        (DEST/"records.json.gz").write_bytes(archive)
        index = dict(purpose=bundle["purpose"], base_commit=bundle["base_commit"],
                     source_hashes=source_hashes,
                     source_manifest_sha256=sha(json.dumps(source_hashes, sort_keys=True,
                                                          separators=(",", ":")).encode()),
                     original_source_unchanged_from_base=bundle["original_source_unchanged_from_base"],
                     records=dict(file="records.json.gz", sha256=sha(archive)),
                     commands=[{k: v for k, v in c.items() if k not in ("stdout", "stderr")}
                               for c in COMMANDS], samples={})
        for candidate, sample in bundle["samples"].items():
            joins = sample["selected_raw_rows"]["joins"]
            index["samples"][candidate] = dict(identities=sample["init"],
                root=sample["inspection"]["Roots"][0],
                original_joins={k: dict(handle=v["handle_from_parent"],
                    commit_sequence=v["intent"]["sequence"],
                    consumed_sequence=v["observation"]["consumed_sequence"],
                    selected_observation=v["history"]["selected_observation"])
                    for k, v in joins.items()},
                program_sha256={p["ID"]: sha(base64.b64decode(p["Bytes"]))
                    for p in sample["artifacts"]["Provision"]["Programs"]})
        (DEST/"evidence-index.json").write_text(json.dumps(index, indent=2)+"\n")
        print("Evidence saved:", DEST/"evidence-index.json", flush=True)
