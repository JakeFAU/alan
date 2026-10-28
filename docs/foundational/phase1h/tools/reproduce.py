#!/usr/bin/env python3
"""Verified offline extraction; focused reproduction, explicitly not a broad rerun."""

from pathlib import Path
import gzip, json, os, subprocess, sys, tarfile, tempfile, time
from common import ROOT, DOC, sha

manifest = json.loads((DOC / "archive-manifest.json").read_text())
archive = DOC / "source.tar.gz"
assert sha(archive.read_bytes()) == manifest["SHA256"]
out = DOC / "evidence/task3/extracted"
out.mkdir(parents=True, exist_ok=False)
commands = []
with tempfile.TemporaryDirectory(prefix="alan-phase1h-offline-") as tmp:
    root = Path(tmp) / "source"
    root.mkdir()
    with tarfile.open(archive) as t:
        assert len(t.getmembers()) == len(manifest["Files"])
        for f in manifest["Files"]:
            m = t.getmember(f["Path"])
            assert (
                m.isfile()
                and not Path(m.name).is_absolute()
                and ".." not in Path(m.name).parts
            )
            raw = t.extractfile(m).read()
            assert len(raw) == f["Bytes"] and sha(raw) == f["SHA256"]
        t.extractall(root, filter="data")
    env = dict(
        os.environ,
        GOMODCACHE=str(Path(tmp) / "empty-module-cache"),
        GOTOOLCHAIN="local",
        GOPROXY="file://" + str(root / "dependencies"),
        GOSUMDB="off",
        GOFLAGS="-mod=readonly",
        GOCACHE="/tmp/alan-phase1f-go-build",
        PYTHONDONTWRITEBYTECODE="1",
        ALAN_H_OUT=str(out / "commands"),
    )
    env.pop("OPENAI_API_KEY", None)
    assert not Path(env["GOMODCACHE"]).exists()
    planned = [
        ["go", "test", "-run", "^$", "./..."],
        ["go", "-C", "examples/worktree/consumer", "test", "-run", "^$", "./..."],
        ["go", "-C", "examples/adaptive/consumer", "test", "-run", "^$", "./..."],
        [
            "go",
            "test",
            "-json",
            "-tags",
            "integration",
            "./internal/adaptivetest",
            "-run",
            "TestProcessAdaptiveCuts/nested=false/(decision|partial)$|TestAdaptiveUnknownSibling/false$|TestAdaptiveClosureLateTransportCompanion$|TestAdaptiveReadOnlyReplay$|TestFaultyAdaptiveApplicationOracle$",
            "-count=1",
            "-timeout=20m",
        ],
        [
            "go",
            "-C",
            "examples/adaptive/consumer",
            "test",
            "-json",
            "-run",
            "TestAdaptivePublicConsumerDirectNested",
            "-count=1",
            "-timeout=15m",
            "./...",
        ],
        [
            "go",
            "test",
            "-json",
            "-tags",
            "integration",
            "./research/semantics",
            "-count=1",
            "-timeout=5m",
        ],
    ]
    for index, cmd in enumerate(planned):
        start = time.monotonic()
        p = subprocess.run(cmd, cwd=root, env=env, capture_output=True)
        stem = f"{index + 1:02d}"
        (out / (stem + ".stdout.gz")).write_bytes(gzip.compress(p.stdout, mtime=0))
        (out / (stem + ".stderr.gz")).write_bytes(gzip.compress(p.stderr, mtime=0))
        commands.append(
            dict(
                Command=cmd,
                Directory=str(root),
                ExitCode=p.returncode,
                ElapsedSeconds=time.monotonic() - start,
                RawSHA256=sha(p.stdout),
                StderrSHA256=sha(p.stderr),
            )
        )
        result = dict(
            SourceID=manifest["SourceID"],
            ArchiveSHA256=manifest["SHA256"],
            VerifiedMembers=len(manifest["Files"]),
            InitialModuleCacheEmpty=True,
            CredentialAbsent=True,
            Commands=commands,
            Scope="Offline empty-module-cache compile of root/both consumers, focused new boundaries/controls/public consumer and old semantic package. No extracted full default/integration/race suite claim.",
            Result="running" if p.returncode == 0 else "FAIL",
        )
        (out / "reproduction.json").write_text(json.dumps(result, indent=2) + "\n")
        print(stem, cmd, p.returncode, flush=True)
        assert p.returncode == 0
    # Compare current extracted semantic payloads with unchanged original records.
    sys.path.insert(0, str(root / "docs/foundational/phase1h/tools"))
    import common

    parsed = common.parse(
        gzip.decompress((out / "06.stdout.gz").read_bytes()),
        manifest["SourceID"],
        "normal",
    )
    baseline = [
        json.loads(x)
        for x in gzip.decompress(
            (
                root
                / "docs/foundational/phase1e/evidence/final-normal.records.jsonl.gz"
            ).read_bytes()
        ).splitlines()
    ]
    import importlib.util

    spec = importlib.util.spec_from_file_location(
        "old", root / "docs/foundational/phase1e/tools/comparison.py"
    )
    c = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(c)
    for marker, count in [
        ("QUALIFICATION", 15),
        ("COMPOSITION", 20),
        ("SUPERVISION", 32),
        ("CONFORMANCE", 6),
    ]:
        a = [r["Evidence"] for r in baseline if r["Marker"] == marker]
        b = [r["Evidence"] for r in parsed["records"] if r["Marker"] == marker]
        assert len(a) == len(b) == count
        if marker == "CONFORMANCE":
            c.compare_conformance(
                a, b, {p.rsplit("/", 1)[-1] for p in parsed["passes"]}
            )
        else:
            assert a == b
    assert (
        common.inventory(root)
        == json.loads(
            (root / "docs/foundational/phase1h/source-manifest.json").read_text()
        )["Files"]
    )
    result.update(
        Result="PASS", SourceUnchanged=True, OriginalSemanticComparisons="PASS"
    )
    (out / "reproduction.json").write_text(json.dumps(result, indent=2) + "\n")
