#!/usr/bin/env python3
"""Serial cross-source qualification; fail on the first exact mismatch.

Does not build binaries and never opens original archived SQLite evidence.
"""
import argparse
import hashlib
import json
import os
import pathlib
import shlex
import subprocess
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--baseline-binary", required=True)
parser.add_argument("--candidate-binary", required=True)
parser.add_argument("--baseline-source", required=True, help="verified source label recorded verbatim")
parser.add_argument("--candidate-source", required=True, help="verified source label recorded verbatim")
parser.add_argument("--tools", default=str(pathlib.Path(__file__).resolve().parent))
parser.add_argument("--archives", required=True, help="immutable real main+WAL archive directory")
parser.add_argument("--expected-archive-count", type=int, default=21)
parser.add_argument("--output", required=True, help="new output directory; existing directory is refused")
args = parser.parse_args()
base = pathlib.Path(args.baseline_binary).resolve(strict=True)
candidate = pathlib.Path(args.candidate_binary).resolve(strict=True)
tools = pathlib.Path(args.tools).resolve(strict=True)
archives = pathlib.Path(args.archives).resolve(strict=True)
output = pathlib.Path(args.output).absolute()
output.mkdir(parents=True, exist_ok=False)
env = os.environ.copy()
env["GOMAXPROCS"] = "4"
commands = []
comparisons = []


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def dump(name, value):
    (output / name).write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def archive_hashes():
    return {path.name: sha(path) for path in sorted(archives.iterdir()) if path.is_file()}


def call(name, command):
    command = [str(value) for value in command]
    stdout, stderr = output / (name + ".json"), output / (name + ".stderr")
    started = time.time()
    with stdout.open("wb") as out, stderr.open("wb") as err:
        process = subprocess.run(command, env=env, stdout=out, stderr=err)
    record = {"name": name, "argv": command, "exit": process.returncode,
              "started_unix": started, "elapsed_seconds": time.time() - started,
              "stdout": stdout.name, "stderr": stderr.name}
    commands.append(record)
    dump("commands.json", commands)
    with (output / "commands.sh").open("a") as shell:
        shell.write("GOMAXPROCS=4 " + shlex.join(command) + " > " + shlex.quote(str(stdout)) + " 2> " + shlex.quote(str(stderr)) + "\n")
    print(name + ": exit=" + str(process.returncode), flush=True)
    if process.returncode != 0:
        raise RuntimeError("exact qualification stopped at " + name + "; inspect retained stdout/stderr")
    return stdout


def compare(name, left_report, right_report, left_db, right_db):
    path = call(name, ["python3", tools / "compare.py", "same-seed", left_report, right_report,
                       "--baseline-db", left_db, "--candidate-db", right_db])
    result = json.loads(path.read_text())
    if not result.get("equal_for_stated_coverage"):
        raise RuntimeError("comparator did not affirm its stated coverage")
    comparisons.append({"name": name, "report": path.name,
                        "namespace": result["namespace"], "tables": result["tables"],
                        "public_cut_count": result["public_cut_count"]})
    dump("comparisons.json", comparisons)


before = archive_hashes()
dump("archives-before.sha256.json", before)
manifest = {
    "version": "alan/series-differential-campaign/v1",
    "baseline_source": args.baseline_source, "candidate_source": args.candidate_source,
    "binary_sha256": {"baseline": sha(base), "candidate": sha(candidate)},
    "tool_sha256": {path.name: sha(path) for path in sorted(tools.iterdir()) if path.is_file()},
    "campaign_sha256": sha(pathlib.Path(__file__).resolve()),
    "gomaxprocs": 4, "archive_directory": str(archives), "output": str(output),
    "source_labels_are_caller_supplied": True,
}
dump("inputs.json", manifest)
try:
    archived_stores = sorted(archives.glob("*.sqlite"))
    if len(archived_stores) != args.expected_archive_count:
        raise RuntimeError("archive count differs from declared count")
    for archived in archived_stores:
        if not pathlib.Path(str(archived) + "-wal").exists():
            raise RuntimeError("expected retained main+WAL pair is missing: " + archived.name)
    seed_dir = output / "seed"
    seed_dir.mkdir()
    seed = seed_dir / "store.sqlite"
    call("seed", [base, "-mode", "create-seed", "-bound", "8", "-db", seed, "-source-label", args.baseline_source])
    for scenario, interval in [("success", 0), ("success", 1), ("refusal3", 0), ("refusal3", 1)]:
        name = scenario + "-reopen" + str(interval)
        left = output / name / "baseline" / "store.sqlite"
        right = output / name / "candidate" / "store.sqlite"
        call(name + "-clone", ["python3", tools / "clone_seed.py", seed, left, right])
        reports = []
        for side, binary, database, source in [("baseline", base, left, args.baseline_source),
                                               ("candidate", candidate, right, args.candidate_source)]:
            reports.append(call(name + "-" + side, [binary, "-mode", "execute-existing", "-bound", "8",
                           "-db", database, "-source-label", source, "-scenario", scenario,
                           "-reopen-every", str(interval)]))
        compare(name + "-comparison", reports[0], reports[1], left, right)
    for archived in archived_stores:
        name = "archive-" + archived.stem
        left = output / name / "baseline" / "store.sqlite"
        right = output / name / "candidate" / "store.sqlite"
        call(name + "-clone", ["python3", tools / "clone_cut.py", archived, left, right])
        reports = []
        for side, binary, database, source in [("baseline", base, left, args.baseline_source),
                                               ("candidate", candidate, right, args.candidate_source)]:
            reports.append(call(name + "-" + side, [binary, "-mode", "generic-reader", "-db", database,
                                                   "-source-label", source]))
        compare(name + "-comparison", reports[0], reports[1], left, right)
    after = archive_hashes()
    dump("archives-after.sha256.json", after)
    if before != after:
        raise RuntimeError("original archive inventory/bytes changed")
    dump("summary.json", {"ok": True, "same_seed_execution_pairs": 4,
                          "generic_archive_pairs": len(archived_stores), "comparisons": len(comparisons),
                          "commands": len(commands), "original_archive_files_unchanged": len(before),
                          "normalization": ["source_label", "store", "physical SQL row ordering"],
                          "no_epoch_or_identity_normalization": True,
                          "scope": "controlled 8-position execution and same-history generic validation/renewal; generic mode never Run/resume"})
except Exception as error:
    dump("archives-after.sha256.json", archive_hashes())
    dump("summary.json", {"ok": False, "error": str(error), "commands": len(commands),
                          "completed_comparisons": len(comparisons)})
    raise
finally:
    dump("output-sha256.json", {str(path.relative_to(output)): sha(path) for path in sorted(output.rglob("*"))
                               if path.is_file() and path.name != "output-sha256.json"})
print("all exact cross-source differential comparisons passed", flush=True)
