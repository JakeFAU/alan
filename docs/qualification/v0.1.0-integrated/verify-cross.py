"""Cross-check both SQLite engines against public pre-release, frozen, and scoped histories.

Usage: python3 -B verify-cross.py BASE RC INTEGRATED RESEARCH CROSS OUT
CROSS is the completed sqlite-1.14.53/compare.py output. OUT must be fresh.
Set GOMODCACHE/GOCACHE to verified local caches and GOWORK=off.
"""

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile

PRE_RELEASE = "64a692b6f82ae10353f50c8741377166bc6cf854"
RC = "bea0acc86d9a7302ccc3511327455922503c4a57"
FROZEN = "003dc9c45ac9b0c1f48565632f5be80f0f05beed"
MOD52 = "v1.14.52"
MOD53 = "v1.14.53"


def load(module, path):
    spec = importlib.util.spec_from_file_location(module, path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def cmd(args, cwd=None, env=None, log=None):
    args = list(map(str, args))
    if log is None:
        return subprocess.check_output(args, cwd=cwd, env=env)
    with log.open("wb") as f:
        subprocess.run(args, cwd=cwd, env=env, stdout=f, check=True)


def head(cwd):
    return cmd(["git", "rev-parse", "HEAD"], cwd).decode().strip()


def sha(file):
    return hashlib.sha256(file.read_bytes()).hexdigest()


def build(source, package, target):
    cmd(
        ["go", "build", "-mod=readonly", "-buildvcs=false", "-o", target, package],
        source,
    )
    return target


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ("base", "rc", "integrated", "research", "cross", "out"):
        parser.add_argument(key, type=Path)
    args = parser.parse_args()
    base, rc, integrated, research, cross, out = [
        getattr(args, k).resolve()
        for k in ("base", "rc", "integrated", "research", "cross", "out")
    ]
    out.mkdir()
    assert head(rc) == RC and head(base) == PRE_RELEASE
    info = [
        ("base", PRE_RELEASE),
        ("initial_rc", RC),
        ("integrated", head(integrated)),
        ("research_extraction", FROZEN),
    ]
    c = load(
        "binding_comparison",
        integrated / "docs/qualification/sqlite-1.14.53/compare.py",
    )
    runtime = c.compare_stores
    frozen = out / "frozen"
    frozen.mkdir()
    archive = out / "research.tar"
    cmd(
        [
            "git",
            "archive",
            "--format=tar",
            "--output",
            archive,
            FROZEN,
            "go.mod",
            "go.sum",
            "cmd/alan",
            "adapters",
            "author",
            "compose",
            "effects",
            "runtime",
            "schema",
            "internal/executor",
            "internal/boundary",
            "internal/fixture",
        ],
        research,
    )
    with tarfile.open(archive) as tar:
        tar.extractall(frozen, filter="data")
    labels = {
        "frozen": build(frozen, "./cmd/alan", out / "frozen-alan"),
        "rc": build(rc, "./cmd/alan", out / "rc-alan"),
        "integrated": cross / "alan-v1.14.53",
        "base": cross / "alan-v1.14.52",
    }
    versions = {
        b: (MOD53 if lab == "integrated" else MOD52) for lab, b in labels.items()
    }
    for _label, b in labels.items():
        info_text = cmd(["go", "version", "-m", b]).decode()
        assert (
            "\tdep\t" + c.MODULE + "\t" + versions[b] + "\t" + c.ENGINES[versions[b]][0]
            in info_text
            and "\t=>\t" not in info_text
        )
    selected = json.loads((cross / "sources.json").read_text())
    for label, wanted_revision in (
        ("base", PRE_RELEASE),
        ("integrated", head(integrated)),
    ):
        binary = labels[label]
        identified = selected[versions[binary]]
        if identified["commit"] != wanted_revision or identified["cli_sha256"] != sha(
            binary
        ):
            raise ValueError(
                f"cross-binding input binary does not identify {label} at {wanted_revision}"
            )

    def read(b, cmd, db):
        return c.checked_output(runtime.cli(b, cmd, db), cmd, versions[b])

    cuts = []
    for other in ("frozen", "rc"):
        for writer, reader in ((other, "integrated"), ("integrated", other)):
            for boundary, phase in (
                ("execution.staged", "pre-commit"),
                ("execution.committed", "post-commit"),
            ):
                p = out / f"{writer}-{reader}-{phase}"
                p.mkdir()
                history, intents = runtime.compare(
                    (labels[writer], labels[reader]), p, boundary, phase, read=read
                )
                assert (history, intents) == (26, 4)
                cuts.append(
                    dict(
                        writer=writer,
                        reader=reader,
                        boundary=boundary,
                        history=history,
                        intents=intents,
                    )
                )
    capture = out / "capture.py"
    capture.write_text("""#!/usr/bin/env python3
import os,sys,subprocess,shutil
from pathlib import Path
verb,flag,db=sys.argv[1:]
assert flag=="--db" and verb in ("inspect","replay")
p=subprocess.run([os.environ["FROZEN_PRE_SCOPE"],*sys.argv[1:]],capture_output=True)
if p.returncode==1 and p.stderr.strip()==b"unsupported version" and verb=="inspect":
    src=Path(db); dst=Path(os.environ["CAPTURE_DIR"])/src.name
    assert not dst.exists()
    shutil.copyfile(src,dst)
    if Path(db+"-wal").exists(): shutil.copyfile(db+"-wal",str(dst)+"-wal")
sys.stdout.buffer.write(p.stdout);sys.stderr.buffer.write(p.stderr);sys.exit(p.returncode)
""")
    capture.chmod(0o700)
    scoped = []
    for lab, source in (("rc", rc), ("integrated", integrated)):
        cap = out / ("scoped-" + lab)
        cap.mkdir()
        log = cap / "checks.jsonl"
        env = dict(
            os.environ,
            FROZEN_PRE_SCOPE=str(labels["base"]),
            CAPTURE_DIR=str(cap),
            ALAN_V010_OLD_READER=str(capture),
        )
        tests = "^Test(ScopedAssetHistoryFormats|ScopedContinuationConflictPromotion)$"
        cmd(
            [
                "go",
                "test",
                "-count=1",
                "-json",
                "-tags=integration",
                "-run",
                tests,
                "./internal/executor",
            ],
            source,
            env,
            log,
        )
        for v in range(17, 31):
            db = cap / f"format-{v}.sqlite"
            wanted = sha(db)
            wal = Path(str(db) + "-wal")
            saved_wal = sha(wal) if wal.is_file() else None
            known, replay = runtime.inspect_and_replay(
                (labels["rc"], labels["integrated"]), db, read
            )
            r = json.loads(replay)
            assert json.loads(known)["Tables"]["metadata"][0]["schema_version"] == v
            scoped.append(
                dict(
                    writer=lab,
                    format=v,
                    database_sha256=wanted,
                    wal_sha256=saved_wal,
                    replay_commits=r["HistoryCount"],
                    replay_intents=r["IntentCount"],
                )
            )
    quickstarts = {}
    for lab, source in (("base", base), ("rc", rc), ("integrated", integrated)):
        quickstarts[lab] = build(
            source, "./examples/quickstart", out / f"quickstart-{lab}"
        )
    quick = []
    for writer, reader, v in (
        ("base", "integrated", 3),
        ("rc", "integrated", 17),
        ("integrated", "rc", 17),
    ):
        qp = out / f"quickstart-{writer}-{reader}"
        qp.mkdir()
        db = qp / "source.sqlite"
        control = qp / "control.sqlite"

        def invoke(lab, verb, store):
            return cmd([quickstarts[lab], verb, store]).decode().strip()

        assert invoke(writer, "start", db) == "quiescent=true open_runs=1"
        # A completed start closed the owner's WAL.
        runtime.clone_stopped_store(db, control)
        inspection = json.loads(runtime.cli(labels[reader], "inspect", db))
        assert inspection["Tables"]["metadata"][0]["schema_version"] == v
        if v == 17 and writer == "integrated":
            for verb in ("inspect", "replay"):
                deny = subprocess.run(
                    [labels["base"], verb, "--db", str(db)], capture_output=True
                )
                assert (
                    deny.returncode == 1
                    and deny.stderr.strip() == b"unsupported version"
                )
        assert invoke(reader, "inspect", db) == invoke(writer, "inspect", control)
        assert (
            invoke(reader, "resume", db)
            == invoke(writer, "resume", control)
            == "quiescent=true open_runs=0"
        )
        assert invoke(reader, "inspect", db) == invoke(writer, "inspect", control)
        assert "programs=1 attempts=1 results=1" in invoke(reader, "inspect", db)
        answer = invoke(reader, "replay", db)
        assert (
            answer == invoke(writer, "replay", control)
            and "verified commits=6 intents=1" in answer
        )
        quick.append(
            dict(
                writer=writer,
                reader=reader,
                format=v,
                attempts=1,
                results=1,
                replay=answer.splitlines(),
            )
        )
    result = dict(
        disposition="PASS",
        sources=dict(info),
        binary_sha256={k: sha(v) for k, v in labels.items()},
        expected_sqlite_drivers={lab: versions[b] for lab, b in labels.items()},
        excerpted_source_archive_sha256=sha(archive),
        crash_cases=cuts,
        physical_scope_formats=scoped,
        cross_binding_asset_quickstart=quick,
        scoped_read_comparison="Inspect and Replay cover both engines after checking their exact two live engine identifiers; preserved DB and WAL bytes. A pre-scope reader must reject inspect and replay on all 28 writer/format cases.",
        crash_case_contract="Cross-reader and same-writer resumed stores match; durable database/WAL unchanged by reads; both replayers reject deliberate corruption.",
    )
    (out / "verification.json").write_text(json.dumps(result, indent=2) + "\n")
    print(
        f"{len(cuts)} additional crash cuts, {len(scoped)} scoped cross-engine fixtures, {len(quick)} asset quickstarts PASS"
    )


if __name__ == "__main__":
    main()
