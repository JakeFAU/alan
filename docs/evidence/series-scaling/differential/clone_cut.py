#!/usr/bin/env python3
"""Copy immutable archived main+WAL evidence into disposable reader directories.

The caller must supply an archived cut from a reaped owner, not an active store.
No SQLite connection is opened against the original evidence archive.
"""
import argparse
import hashlib
import json
import pathlib
import shutil

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("archive")
parser.add_argument("destinations", nargs="+")
args = parser.parse_args()
source = pathlib.Path(args.archive).resolve(strict=True)
files = {suffix: pathlib.Path(str(source) + suffix) for suffix in ("", "-wal")}
files = {suffix: path for suffix, path in files.items() if path.exists()}

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

before = {suffix: digest(path) for suffix, path in files.items()}
clones = []
for value in args.destinations:
    target = pathlib.Path(value).absolute()
    if target.resolve(strict=False) == source:
        raise RuntimeError("destination aliases archived evidence")
    for suffix in ("", "-wal", "-shm", ".lock"):
        if pathlib.Path(str(target) + suffix).exists():
            raise RuntimeError("target or sidecar exists: " + str(target))
    target.parent.mkdir(parents=True, exist_ok=True)
    copied = {}
    for suffix, path in files.items():
        output = pathlib.Path(str(target) + suffix)
        with output.open("xb") as destination, path.open("rb") as original:
            shutil.copyfileobj(original, destination)
        copied[suffix] = digest(output)
    if copied != before:
        raise RuntimeError("archive clone differs")
    clones.append({"path": str(target), "sha256": copied})
if before != {suffix: digest(path) for suffix, path in files.items()}:
    raise RuntimeError("archive changed during clone")
print(json.dumps({"archive": str(source), "sha256": before, "clones": clones}, indent=2))
