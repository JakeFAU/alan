#!/usr/bin/env python3
"""Deterministic current H sources and pinned offline dependency, no nested archives."""

from pathlib import Path
import gzip, io, json, subprocess, tarfile
from common import ROOT, DOC, inventory, source_id, environment, sha

manifest = json.loads((DOC / "source-manifest.json").read_text())
files = inventory()
assert files == manifest["Files"] and source_id(files) == manifest["SourceID"]
members = {p: (ROOT / p).read_bytes() for p in files}
for p in DOC.rglob("*.md"):
    if "/development/" not in str(p):
        members[str(p.relative_to(ROOT))] = p.read_bytes()
members["docs/foundational/phase1h/source-manifest.json"] = (
    DOC / "source-manifest.json"
).read_bytes()
cache = Path(
    subprocess.check_output(
        ["go", "env", "GOMODCACHE"], cwd=ROOT, env=environment(), text=True
    ).strip()
)
proxy = "github.com/mattn/go-sqlite3/@v/"
for suffix in ("info", "mod", "zip"):
    members["dependencies/" + proxy + "v1.14.52." + suffix] = (
        cache / "cache/download" / proxy / ("v1.14.52." + suffix)
    ).read_bytes()
members["dependencies/" + proxy + "list"] = b"v1.14.52\n"
assert (
    sha(members["dependencies/" + proxy + "v1.14.52.zip"])
    == manifest["Dependency"]["zip_sha256"]
)
archive = DOC / "source.tar.gz"
assert not archive.exists(), "refuse to replace archive"
with archive.open("wb") as f:
    with gzip.GzipFile(fileobj=f, mode="wb", filename="", mtime=0) as z:
        with tarfile.open(fileobj=z, mode="w") as t:
            for name, raw in sorted(members.items()):
                info = tarfile.TarInfo(name)
                info.size = len(raw)
                info.mode = 0o644
                info.mtime = 0
                t.addfile(info, io.BytesIO(raw))
result = dict(
    SourceID=manifest["SourceID"],
    SHA256=sha(archive.read_bytes()),
    Bytes=archive.stat().st_size,
    Files=[
        dict(Path=n, Bytes=len(r), SHA256=sha(r)) for n, r in sorted(members.items())
    ],
    Scope="Current complete contributing source/tools/fixture/comparison inputs, H prose and pinned sqlite proxy. Historical archives and generated run evidence remain separately retained alongside; toolchain excluded.",
)
(DOC / "archive-manifest.json").write_text(json.dumps(result, indent=2) + "\n")
print(json.dumps({k: v for k, v in result.items() if k != "Files"}, indent=2))
