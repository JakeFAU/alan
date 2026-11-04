#!/usr/bin/env python3
"""Small semantic/proof smoke campaign; run only after measurement lock release."""
import argparse
import copy
import fcntl
import hashlib
import json
import os
import pathlib
import shutil
import sqlite3
import subprocess
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("binary", help="differential runner built against a selected frozen source")
parser.add_argument("output", help="new output directory")
parser.add_argument("archive", help="closed/immutable real store for generic reader smoke")
args = parser.parse_args()
binary = pathlib.Path(args.binary).resolve(strict=True)
output = pathlib.Path(args.output).absolute()
output.mkdir(parents=True, exist_ok=False)
tools = pathlib.Path(__file__).resolve().parent
commands = []
env = os.environ.copy()
env["GOMAXPROCS"] = "4"


def call(name, command, expected=0):
    started = time.time()
    result = subprocess.run([str(arg) for arg in command], env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    (output / (name + ".json")).write_bytes(result.stdout)
    (output / (name + ".stderr")).write_bytes(result.stderr)
    record = {"name": name, "argv": [str(arg) for arg in command], "exit": result.returncode, "expected_exit": expected, "elapsed_seconds": time.time()-started}
    commands.append(record)
    (output / "commands.json").write_text(json.dumps(commands, indent=2))
    if result.returncode != expected:
        raise RuntimeError("unexpected exit for " + name + ": " + result.stderr.decode(errors="replace"))
    print(name + ": exit=" + str(result.returncode), flush=True)
    return json.loads(result.stdout) if expected == 0 else None


def compare(name, left_report, right_report, left_db, right_db, expected=0):
    return call(name, ["python3", tools/"compare.py", "same-seed", left_report, right_report, "--baseline-db", left_db, "--candidate-db", right_db], expected)


seed = output / "seed.sqlite"
call("seed", [binary, "-mode", "create-seed", "-db", seed])
pairs = {}
for scenario, interval in [("success", 0), ("success", 1), ("refusal3", 0), ("refusal3", 1)]:
    name = scenario + "-reopen" + str(interval)
    left, right = output/(name+"-left.sqlite"), output/(name+"-right.sqlite")
    call(name+"-clone", ["python3", tools/"clone_seed.py", seed, left, right])
    for side, db in [("left", left), ("right", right)]:
        call(name+"-"+side, [binary, "-mode", "execute-existing", "-db", db, "-source-label", side, "-scenario", scenario, "-reopen-every", str(interval)])
    lrep, rrep = output/(name+"-left.json"), output/(name+"-right.json")
    compare(name+"-comparison", lrep, rrep, left, right)
    pairs[name] = (lrep, rrep, left, right)

left, right = output/"generic-left.sqlite", output/"generic-right.sqlite"
call("generic-clone", ["python3", tools/"clone_cut.py", args.archive, left, right])
for side, db in [("left", left), ("right", right)]:
    call("generic-"+side, [binary, "-mode", "generic-reader", "-db", db, "-source-label", side])
compare("generic-comparison", output/"generic-left.json", output/"generic-right.json", left, right)

lrep, rrep, ldb, rdb = pairs["success-reopen0"]
original = json.loads(rrep.read_text())
# Mutations affect actual identity/evidence, not the comparison implementation.
report_mutations = {
    "wrong-id": lambda r: r["contacts"][0]["prepared"]["Request"].update(ID="foreign-operation"),
    "wrong-contact-key": lambda r: r["contacts"][0]["prepared"]["Request"].update(Key="foreign-request-key"),
    "wrong-approval-predecessor": lambda r: next(c for c in r["cuts"] if c["snapshot"].get("Continuations"))["snapshot"]["Continuations"][0]["Approval"].update(PredecessorID="foreign-predecessor"),
    "wrong-digest": lambda r: r["contacts"][0]["prepared"].update(Digest="0"*64),
    "wrong-selection": lambda r: r["cuts"][-1]["history"][1].update(SelectedObservationID="foreign-observation"),
    "wrong-uncertainty": lambda r: r["cuts"][-1]["snapshot"]["Effects"][0].update(Consequence="unresolved"),
    "foreign-namespace": lambda r: r["cuts"][0]["snapshot"].update(Namespace="0"*64),
}
for name, mutation in report_mutations.items():
    changed = copy.deepcopy(original)
    mutation(changed)
    path = output/("mutated-"+name+".json")
    path.write_text(json.dumps(changed))
    compare("reject-"+name, lrep, path, ldb, rdb, 1)

sql_mutations = {
    "grant": "UPDATE permissions SET revision=revision+1 WHERE source='series-1'",
    "charge": "UPDATE composition_charges SET descriptor=CAST(json_set(descriptor,'$.Effects',0) AS BLOB) WHERE operation_id='[\"series-1\",1,1,0]'",
    "epoch": "UPDATE runs SET epoch=epoch+1 WHERE subject='series-1'",
    "missing-result": "DELETE FROM adapter_results WHERE intent_id='[\"series-1\",1,1,0]'",
}
for name, sql in sql_mutations.items():
    path = output/("mutated-"+name+".sqlite")
    shutil.copyfile(rdb, path)
    connection = sqlite3.connect(path)
    try:
        cursor = connection.execute(sql)
        if cursor.rowcount < 1:
            raise RuntimeError("mutation touched no original record: " + name)
        connection.commit()
    finally:
        connection.close()
    compare("reject-"+name, lrep, rrep, ldb, path, 1)

call("clone-reject-existing", ["python3", tools/"clone_seed.py", seed, ldb], 1)
call("clone-reject-executed", ["python3", tools/"clone_seed.py", ldb, output/"must-not-clone-executed.sqlite"], 1)
with open(str(seed)+".lock", "rb+") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    call("clone-reject-live-owner", ["python3", tools/"clone_seed.py", seed, output/"must-not-clone-live.sqlite"], 1)
nonempty = output/"wal-seed.sqlite"
shutil.copyfile(seed, nonempty)
pathlib.Path(str(nonempty)+".lock").touch()
pathlib.Path(str(nonempty)+"-wal").write_bytes(b"nonempty WAL must be refused before SQLite read")
call("clone-reject-wal", ["python3", tools/"clone_seed.py", nonempty, output/"must-not-clone-wal.sqlite"], 1)

manifest = {}
for path in sorted(output.iterdir()):
    if path.is_file() and path.suffix in (".json", ".stderr"):
        manifest[path.name] = hashlib.sha256(path.read_bytes()).hexdigest()
(output/"output-sha256.json").write_text(json.dumps(manifest, indent=2))
(output/"summary.json").write_text(json.dumps({"ok": True, "commands": len(commands), "same_seed_pairs": 4, "generic_reader_pairs": 1, "negative_report_cases": len(report_mutations), "negative_sql_cases": len(sql_mutations), "clone_guard_cases": 4}, indent=2))
print("all differential tool checks passed", flush=True)
