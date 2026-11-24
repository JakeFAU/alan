#!/usr/bin/env python3
"""Prepare disposable v0.1 reader fixtures and frozen executables from Git."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

BASE = "64a692b6f82ae10353f50c8741377166bc6cf854"
PRE_AUTHORITY = "af6943a5805a9a6c0e8fe77f804264f4765d07ec"
PRE_BINDING = "3f8307664b6543331a41c943e91fb5bac8b376ef"


def run(args, cwd, **extra):
    subprocess.run(args, cwd=cwd, check=True, env={**os.environ, "GOWORK": "off", "GOTOOLCHAIN": "local", **extra})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path, help="new directory for the immutable inputs")
    output = parser.parse_args().output.resolve()
    output.mkdir()
    repo = Path(subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip())
    matrix = output / "matrix"
    matrix.mkdir()
    with tempfile.TemporaryDirectory() as tmp:
        source = Path(tmp) / "source"
        run(["git", "clone", "--quiet", "--shared", "--no-checkout", str(repo), str(source)], repo)
        for revision, command in [
            (PRE_AUTHORITY, ["go", "build", "-o", str(output / "old13"), "./cmd/alan"]),
            (PRE_BINDING, ["go", "test", "-tags", "integration", "-c", "-o", str(output / "old15"), "./internal/durabletest"]),
            (BASE, ["go", "build", "-o", str(output / "pre-release-reader"), "./cmd/alan"]),
        ]:
            run(["git", "checkout", "--quiet", "--detach", revision], source)
            run(command, source)

        legacy = source / "docs/qualification/phase3b3/legacy-matrix"
        manifest = json.loads((legacy / "manifest.json").read_text())
        checksums = {row["Path"]: row["SHA256"] for row in manifest["Files"]}
        archive = legacy / "evidence.tar.gz"
        if hashlib.sha256(archive.read_bytes()).hexdigest() != manifest["ArchiveSHA256"]:
            raise ValueError("historical archive checksum mismatch")
        with tarfile.open(archive) as tar:
            for v in range(3, 13):
                name = f"originals/v{v:02}/store.sqlite"
                data = tar.extractfile("legacy-matrix/" + name).read()
                if hashlib.sha256(data).hexdigest() != checksums[name]:
                    raise ValueError("historical database checksum mismatch: " + name)
                (matrix / f"v{v:02}.sqlite").write_bytes(data)

        parents = output / "legacy-authority"
        promoted = output / "legacy-promoted"
        run(["go", "test", "-count=1", "-tags", "integration", "-run",
             "^TestPhase3B4CampaignCompatibility$/^K0[12]$", "./internal/durabletest"],
            source, ALAN_PHASE3B4_EVIDENCE=str(parents))
        run(["go", "test", "-count=1", "-tags", "integration", "-run",
             "^TestPhase3B4PromotedPublicParents$", "./internal/executor"],
            source, ALAN_PHASE3B4_PUBLIC_PARENTS=str(parents), ALAN_PHASE3B4_PROMOTED_OUTPUT=str(promoted))
        for version, directory, pattern in [(13, parents, "K01-*/final"), (14, promoted, "K01-*"),
                                            (15, parents, "K02-*/final"), (16, promoted, "K02-*")]:
            db, = directory.glob(pattern + "/runtime.sqlite")
            for suffix in ["", "-wal"]:
                part = Path(str(db) + suffix)
                if part.exists():
                    shutil.copyfile(part, matrix / f"v{version:02}.sqlite{suffix}")
    hashes = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(matrix.iterdir())}
    (output / "inputs.json").write_text(json.dumps(hashes, indent=2) + "\n")


if __name__ == "__main__":
    main()
