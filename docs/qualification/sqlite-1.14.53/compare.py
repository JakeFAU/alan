"""Compare the two pinned SQLite builds on disposable copies of retained stores."""

import argparse
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import sys
import tarfile

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO / "tools"))
import compare_stores

MODULE = "github.com/mattn/go-sqlite3"
PRE_BINDING = "d59eaf2976849e450635d252c7bca62eeccbef2e"
# From the respective verified module's sqlite3-binding.h.
ENGINES = {
    "v1.14.52": ("h1:wVbm2Qnf4OXkqhBTSPuCRZDRnxfbVrrmiCEroVdog8U=", "3.53.4", "2026-07-24 19:02:57 bf7c7f30031888f4e796e429ab3978879485813aaca6f641c7b33e4e09459bcc"),
    "v1.14.53": ("h1:mnTM0owzUMrKuLo+0w51NUiB0ir2Fo7kFXlPt/l3BGs=", "3.54.1", "2026-10-16 14:45:20 555e07ef33f69a0957d24d8e555b702464a131d6f1607f4c3c1e4551e8e6ca76"),
}


def digest(data):
    return hashlib.sha256(data).hexdigest()


def check_binary(binary, binding):
    info = subprocess.check_output(["go", "version", "-m", binary])
    (binary.parent / (binary.name + ".build-info")).write_bytes(info)
    expected = f"\tdep\t{MODULE}\t{binding}\t{ENGINES[binding][0]}".encode()
    if expected not in info.splitlines() or b"\t=>\t" in info:
        raise ValueError(f"unexpected or replaced module in {binary}")


def build_historical_owner(repository, output):
    archive, source = output / "pre-binding.tar", output / "pre-binding"
    tree = subprocess.check_output(["git", "rev-parse", PRE_BINDING + "^{tree}"], cwd=repository).decode().strip()
    subprocess.run(["git", "archive", "--format=tar", "-o", archive, PRE_BINDING], cwd=repository, check=True)
    source.mkdir()
    subprocess.run(["tar", "-xf", archive, "-C", source], check=True)
    owner = output / "historical-owner"
    build = ["go", "test", "-c", "-buildvcs=false", "-mod=readonly", "-tags=integration",
             "-o", str(owner), "./internal/durabletest"]
    subprocess.run(build, cwd=source, check=True)
    check_binary(owner, "v1.14.52")
    record = {"commit": PRE_BINDING, "tree": tree, "archive_sha256": digest(archive.read_bytes()),
              "binary_sha256": digest(owner.read_bytes()), "build": build}
    (output / "pre-binding.json").write_text(json.dumps(record, indent=2) + "\n")
    return owner


def checked_output(data, verb, binding):
    result = json.loads(data)
    settings, indent = (result["Settings"], 4) if verb == "inspect" else (result, 2)
    for field, expected in zip(("SQLiteVersion", "SQLiteSourceID"), ENGINES[binding][1:]):
        if settings[field] != expected:
            raise ValueError(f"{binding} {verb} reports {field}={settings[field]!r}")
        prefix = b'\n' + b' ' * indent + json.dumps(field).encode() + b': '
        field_bytes = prefix + json.dumps(expected).encode()
        if data.count(field_bytes) != 1:
            raise ValueError(f"cannot uniquely locate verified {verb} {field}")
        data = data.replace(field_bytes, prefix + b'"<verified>"')
    return data


def inspect_integrity(db, destination):
    destination.mkdir()
    probe = destination / "copy.sqlite"
    for suffix in ("", "-wal", "-shm"):
        part = Path(str(db) + suffix)
        if part.exists():
            shutil.copyfile(part, str(probe) + suffix)
    with closing(sqlite3.connect(probe.as_uri() + "?mode=ro", uri=True)) as conn:
        if conn.execute("PRAGMA integrity_check").fetchall() != [("ok",)]:
            raise ValueError(f"integrity_check failed on {db}")
        if conn.execute("PRAGMA foreign_key_check").fetchall():
            raise ValueError(f"foreign_key_check failed on {db}")
    return probe


def run_test(source, log, options, env, tags="integration"):
    with log.open("xb") as out:
        subprocess.run(["go", "test", "-buildvcs=false", "-mod=readonly", "-count=1", "-json",
                        "-tags=" + tags, *options], cwd=source, stdout=out, env=env, check=True)


def legacy_matrix(sources, output):
    source = REPO / "docs/qualification/phase3b3/legacy-matrix"
    archive = source / "evidence.tar.gz"
    if digest(archive.read_bytes()) != json.loads((source / "manifest.json").read_text())["ArchiveSHA256"]:
        raise ValueError("historical matrix archive checksum mismatch")
    output.mkdir()
    with tarfile.open(archive) as packet:
        def member(name):
            return packet.extractfile("legacy-matrix/" + name).read()

        originals = json.loads(member("originals-before.json"))
        if set(originals) != {f"v{i:02d}/store.sqlite" for i in range(3, 13)}:
            raise ValueError("unexpected original fixture inventory")
        overlay_source = output / "legacy_matrix_test.go.txt"
        overlay_source.write_bytes(member("legacy_matrix_test.go.txt"))
        for name, expected in originals.items():
            blob = member("originals/" + name)
            if digest(blob) != expected:
                raise ValueError(f"historical checksum mismatch: {name}")
            target = output / "originals" / name
            target.parent.mkdir(parents=True)
            target.write_bytes(blob)
            inspect_integrity(target, output / ("integrity-" + target.parent.name))

    for side, source in zip(("old", "current"), sources):
        overlay = output / (side + ".json")
        target = source / "internal/executor/phase3b3_legacy_matrix_test.go"
        overlay.write_text(json.dumps({"Replace": {str(target): str(overlay_source)}}))
        env = dict(os.environ, ALAN_LEGACY_MATRIX_ROOT=str(output), ALAN_LEGACY_MATRIX_MODE="read",
                   ALAN_LEGACY_MATRIX_READER=side)
        run_test(source, output / (side + ".jsonl"),
                 ["-overlay=" + str(overlay), "-run=^TestPhase3B3LegacyMatrix$", "./internal/executor"], env)
    original_root = output / "originals"
    retained = {p.relative_to(original_root).as_posix(): digest(p.read_bytes())
                for p in original_root.rglob("*") if p.is_file()}
    if retained != originals:
        raise ValueError("archived input inventory changed")
    for name in originals:
        version = Path(name).parent
        for api_file, verb in (("Inspect/export.json", "inspect"), ("Replay/export.json", "replay"),
                               ("InspectExecution/export.json", None), ("Open/raw.json", "inspect"),
                               ("Open/replay.json", "replay"), ("Open/typed.json", None)):
            path = version / api_file
            pair = []
            for side, binding in zip(("old", "current"), ENGINES):
                blob = (output / "readers" / side / path).read_bytes()
                pair.append(checked_output(blob, verb, binding) if verb else blob)
            if pair[0] != pair[1]:
                raise ValueError(f"retained export differs: {path}")
        entry = output / "readers/old" / version
        before = json.loads((entry / "InspectExecution/export.json").read_text())
        for run in before["Snapshot"]["Runs"]:
            if run["Projection"]["Status"] == "open":
                run["Epoch"] += 1
        if before != json.loads((entry / "Open/typed.json").read_text()):
            raise ValueError(f"unexpected Open delta: {version}")
    print("Formats 3–12: 80 reader/API cases, 60 matching export pairs, 10 typed Open deltas limited to open-run epochs")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline", type=Path, help="checkout with the original binding")
    parser.add_argument("candidate", type=Path, help="checkout with the candidate binding")
    parser.add_argument("output", type=Path, help="new directory outside both checkouts")
    args = parser.parse_args()
    sources = [p.resolve(strict=True) for p in (args.baseline, args.candidate)]
    output = args.output.resolve()
    if any(output == source or source in output.parents for source in sources):
        parser.error("output must be outside the checkouts")
    output.mkdir()
    os.environ.update(GOFLAGS="", GOWORK="off", GOTOOLCHAIN="local", CGO_ENABLED="1")
    owner = build_historical_owner(sources[0], output)
    binaries, revisions = {}, {}
    for source, binding in zip(sources, ENGINES):
        binary = output / ("alan-" + binding)
        subprocess.run(["go", "build", "-buildvcs=false", "-mod=readonly", "-o", binary, "./cmd/alan"], cwd=source, check=True)
        check_binary(binary, binding)
        commit, tree = subprocess.check_output(["git", "rev-parse", "HEAD", "HEAD^{tree}"], cwd=source).decode().splitlines()
        changes = subprocess.check_output(["git", "status", "--porcelain", "--", "*.go", "go.mod", "go.sum"], cwd=source)
        if changes:
            raise ValueError(f"uncommitted Go/module source at {source}")
        revisions[binding] = {"commit": commit, "tree": tree, "cli_sha256": digest(binary.read_bytes())}
        binaries[binary] = binding
        trace = Path(__file__).resolve().with_name("readiness_trace_test.go.txt")
        overlay = output / ("trace-" + binding + ".json")
        overlay.write_text(json.dumps({"Replace": {str(source / "internal/executor/sqlite_qualification_trace_test.go"): str(trace)}}))
        trace_log = output / ("trace-" + binding + ".jsonl")
        trace_test = "TestSQLiteReadinessStatementCounts"
        run_test(source, trace_log, ["-overlay=" + str(overlay), "-run=^" + trace_test + "$", "./internal/executor"],
                 os.environ, tags="sqlite_trace")
        with trace_log.open() as events:
            if not any(e.get("Test") == trace_test and e["Action"] == "pass" for e in map(json.loads, events)):
                raise ValueError(f"missing native trace pass for {binding}")

    (output / "sources.json").write_text(json.dumps(revisions, indent=2) + "\n")

    def checked_read(binary, verb, db):
        return checked_output(compare_stores.cli(binary, verb, db), verb, binaries[binary])

    for pair in (tuple(binaries), tuple(reversed(binaries))):
        for boundary, phase in (("execution.staged", "pre-commit"), ("execution.committed", "post-commit")):
            case = output / (pair[0].name + "-" + phase)
            case.mkdir()
            history, intents = compare_stores.compare(pair, case, boundary, phase, read=checked_read)
            print(f"{binaries[pair[0]]} -> {binaries[pair[1]]} {phase}: {history} commits, {intents} intents; corrupt replay rejected")

    legacy_matrix(sources, output / "legacy")
    for source, binding in zip(sources, ENGINES):
        out = output / binding
        out.mkdir()
        parents, promoted = out / "parents", out / "promoted"
        run_test(source, out / "parents.jsonl",
                 ["-run=^TestPhase3B4CampaignCompatibility$/(K01|K02)$", "./internal/durabletest"],
                 dict(os.environ, ALAN_PHASE3B4_EVIDENCE=str(parents)))
        run_test(source, out / "promoted.jsonl",
                 ["-run=^TestPhase3B4PromotedPublicParents$", "./internal/executor"],
                 dict(os.environ, ALAN_PHASE3B4_PUBLIC_PARENTS=str(parents), ALAN_PHASE3B4_PROMOTED_OUTPUT=str(promoted)))
        for version, root, pattern in ((13, parents, "K01-*/final"), (14, promoted, "K01-promoted-*"),
                                       (15, parents, "K02-*/final"), (16, promoted, "K02-promoted-*")):
            original, = root.glob(pattern + "/runtime.sqlite")
            db = inspect_integrity(original, out / f"format-{version}")
            raw, _ = compare_stores.inspect_and_replay(tuple(binaries), db, read=checked_read)
            known = json.loads(raw)
            if known["Tables"]["metadata"][0]["schema_version"] != version:
                raise ValueError(f"wrong format at {original}")
            if version == 13 and binding == "v1.14.53":
                # Reuse the frozen K01 observer for the supported historical-owner control.
                sys.path.insert(0, str(REPO / "docs/evidence/execution-scaling/task5"))
                from qualification import verify_k01_owner
                owner_out = out / "historical-owner"
                owner_out.mkdir()
                env = dict(os.environ, ALAN_PHASE3B3_CHILD="1", ALAN_PHASE2_DB=str(db), ALAN_PHASE3B3_OUTPUT=str(owner_out))
                with (owner_out / "test.log").open("wb") as log:
                    subprocess.run([owner, "-test.run=^TestPhase3B3Owner$", "-test.v", "-test.timeout=2m"],
                                   env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
                verify_k01_owner(known, *(json.loads((owner_out / name).read_text()) for name in ("open.json", "run.json")))
        print(f"Formats 13–16: four {binding} parents/promotions read and replayed by both binaries")
    print("Pre-binding owner preserved the v1.14.53 format-13 K01 cut")
    print(f"18 fixture copies passed integrity_check and foreign_key_check with Python SQLite {sqlite3.sqlite_version}")


if __name__ == "__main__":
    main()
