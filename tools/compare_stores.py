"""Compare two ALAN CLI binaries on the same retained computation across SIGKILL."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import secrets
import select
import shutil
import signal
import sqlite3
import subprocess
import tempfile
import time


def cli(binary, command, db, *args):
    return subprocess.check_output(
        [binary, command, "--db", db, *args], timeout=60,
    )


def clone_stopped_store(source, destination):
    # A killed process can leave committed frames in its WAL.
    shutil.copyfile(source, destination)
    wal = Path(str(source) + "-wal")
    if wal.exists():
        shutil.copyfile(wal, str(destination) + "-wal")


def read_acknowledgement(stream, timeout=30):
    deadline = time.monotonic() + timeout
    line = bytearray()
    while len(line) < 65536:
        remaining = deadline - time.monotonic()
        if remaining <= 0 or not select.select([stream], [], [], remaining)[0]:
            raise TimeoutError("executor did not acknowledge the requested crash point")
        data = os.read(stream.fileno(), 4096)
        if not data:
            raise ValueError("executor closed its pipe before a complete acknowledgement")
        line.extend(data)
        if b"\n" in line:
            return json.loads(line.split(b"\n", 1)[0])
    raise ValueError("oversized acknowledgement")


def crash(binary, db, boundary, phase):
    snapshot = json.loads(cli(binary, "inspect", db))["Tables"]
    subject = next(row for row in snapshot["subjects"] if row["id"] == "inner")
    run = next(row for row in snapshot["runs"]
               if row["subject"] == "inner" and row["generation"] == subject["generation"])
    event = {
        "Scenario": "correspondence", "Token": secrets.token_hex(16),
        "Boundary": boundary, "Phase": phase, "Store": str(db), "Subject": "inner",
        "Generation": run["generation"], "Epoch": run["epoch"] + 1, "Sequence": 7,
        "Intent": json.dumps(["inner", run["generation"], 7, 0], separators=(",", ":")),
    }
    # Forward the child's stdout and stdin pipes to the executor's barrier protocol.
    command = ["sh", "-c", 'exec 3>&1 4<&0; exec "$@"', "barrier", binary, "run", "--db", db]
    env = dict(os.environ, ALAN_BARRIER=json.dumps(event))
    with subprocess.Popen(command, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE) as child:
        try:
            acknowledgement = read_acknowledgement(child.stdout)
            for key, value in dict(event, PID=child.pid).items():
                if acknowledgement.get(key) != value:
                    raise ValueError(f"unexpected barrier {key}: {acknowledgement!r}")
        finally:
            child.kill()
        if child.wait(timeout=30) != -signal.SIGKILL:
            raise RuntimeError("executor did not die from SIGKILL")


def durable_files(db):
    wal = Path(str(db) + "-wal")
    return {path.name: hashlib.sha256(path.read_bytes()).digest()
            for path in (db, wal) if path.exists()}


def inspect_and_replay(binaries, db):
    # A read-only SQLite connection can create an empty WAL and a shared-memory
    # index. Supply the empty sidecar when needed, then verify every durable byte.
    wal = Path(str(db) + "-wal")
    if not wal.exists():
        wal.touch(exist_ok=False)
    before = durable_files(db)
    first = (cli(binaries[0], "inspect", db), cli(binaries[0], "replay", db))
    second = (cli(binaries[1], "inspect", db), cli(binaries[1], "replay", db))
    if first != second or first[0] != cli(binaries[0], "inspect", db):
        raise ValueError("cross-binary inspection or replay differs")
    if before != durable_files(db):
        raise ValueError("read-only operations changed the database or its WAL")
    return first


def compare(binaries, directory, boundary, phase):
    db = directory / "crashed.sqlite"
    cli(binaries[0], "init", db, "--demo", "nested")
    crash(binaries[0], db, boundary, phase)
    inspection, _ = inspect_and_replay(binaries, db)
    intents = json.loads(inspection)["Tables"]["intents"] or []
    lifecycle = [row for row in intents if json.loads(row["descriptor"])["Kind"] == "lifecycle"]
    if len(lifecycle) != int(phase == "post-commit"):
        raise ValueError("partial or missing lifecycle intent at crash point")

    # Resume the actual killed database with the opposite binary, and its exact
    # stopped copy with the writer. Both preserve the original namespace.
    stable = directory / "control.sqlite"
    clone_stopped_store(db, stable)
    recovered = cli(binaries[1], "run", db)
    control = cli(binaries[0], "run", stable)
    if recovered != control or inspect_and_replay(binaries, db) != inspect_and_replay(binaries, stable):
        raise ValueError("recovery changed retained history or the outcome")
    report = json.loads(recovered)
    replay = json.loads(cli(binaries[1], "replay", db))
    if not report["Quiescent"] or report["OpenRuns"] or len(replay["Runs"]) != 4:
        raise ValueError("nested computation failed to complete")

    # Deliberately corrupt one committed successor in the disposable control.
    with sqlite3.connect(stable) as conn:
        changed = conn.execute("UPDATE history SET successor=X'7B7D' WHERE rowid=(SELECT min(rowid) FROM history)")
        if changed.rowcount != 1:
            raise ValueError("failed to inject exactly one corrupted history row")
    for binary in binaries:
        result = subprocess.run([binary, "replay", "--db", stable], capture_output=True, timeout=60)
        if result.returncode != 1 or b"corrupt" not in result.stderr.lower():
            raise ValueError(f"expected replay corruption rejection, got {result}")
    return replay["HistoryCount"], replay["IntentCount"]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("extracted", type=Path)
    args = parser.parse_args()
    binaries = (args.source.resolve(strict=True), args.extracted.resolve(strict=True))
    for writer, reader in (binaries, binaries[::-1]):
        for boundary, phase in (("execution.staged", "pre-commit"), ("execution.committed", "post-commit")):
            with tempfile.TemporaryDirectory() as path:
                history, intents = compare((writer, reader), Path(path).resolve(), boundary, phase)
            print(f"{writer.name} -> {reader.name} {boundary}: {history} commits, {intents} intents, corrupt replay rejected")


if __name__ == "__main__":
    main()
