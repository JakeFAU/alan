#!/usr/bin/env python3
"""Clone a closed, untouched seed without changing its namespace or evidence."""
import argparse
import fcntl
import hashlib
import json
import os
import pathlib
import shutil
import sqlite3

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("seed")
parser.add_argument("destinations", nargs="+")
args = parser.parse_args()
seed = pathlib.Path(args.seed).resolve(strict=True)
# ALAN's owner uses this same advisory lock. Acquire the existing regular lock
# file without following symlinks; never clone a concurrently owned seed.
fd = os.open(str(seed) + ".lock", os.O_RDWR | os.O_NOFOLLOW)
try:
    fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    wal = pathlib.Path(str(seed) + "-wal")
    if wal.exists() and wal.stat().st_size:
        raise RuntimeError("seed WAL is nonempty; close/checkpoint the owner first")
    connection = sqlite3.connect(seed.as_uri() + "?mode=ro", uri=True)
    try:
        if connection.execute("PRAGMA quick_check").fetchall() != [("ok",)]:
            raise RuntimeError("SQLite quick_check rejected seed")
        namespace = connection.execute("SELECT namespace FROM metadata WHERE singleton=1").fetchone()[0]
        if connection.execute("SELECT count(*) FROM history").fetchone()[0] or connection.execute("SELECT count(*) FROM adapter_attempts").fetchone()[0]:
            raise RuntimeError("seed already contains computation or effect attempts")
    finally:
        connection.close()
    digest = hashlib.sha256(seed.read_bytes()).hexdigest()
    outputs = []
    for value in args.destinations:
        destination = pathlib.Path(value).absolute()
        if destination.resolve(strict=False) == seed:
            raise RuntimeError("destination aliases seed")
        for suffix in ("", "-wal", "-shm", ".lock"):
            if pathlib.Path(str(destination) + suffix).exists():
                raise RuntimeError("destination/sidecar already exists: " + str(destination))
        destination.parent.mkdir(parents=True, exist_ok=True)
        with destination.open("xb") as target, seed.open("rb") as source:
            shutil.copyfileobj(source, target)
        copied = hashlib.sha256(destination.read_bytes()).hexdigest()
        if copied != digest:
            raise RuntimeError("seed clone bytes differ")
        outputs.append({"path": str(destination), "sha256": copied})
    if hashlib.sha256(seed.read_bytes()).hexdigest() != digest or (wal.exists() and wal.stat().st_size):
        raise RuntimeError("seed changed while cloning")
    print(json.dumps({"namespace": namespace, "seed_sha256": digest, "clones": outputs}, indent=2))
finally:
    os.close(fd)
