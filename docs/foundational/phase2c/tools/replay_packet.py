"""Replay distinct captured Store images on the specified binary, without modifying the archive."""
from argparse import ArgumentParser
import gzip
from hashlib import sha256
import json
from pathlib import Path
import subprocess
import tempfile

EVIDENCE = Path(__file__).resolve().parents[1] / "evidence"


def main():
    p = ArgumentParser(description=__doc__)
    p.add_argument("cli", type=Path)
    p.add_argument("--historical", action="store_true", help="check pre-correction captures separately")
    args = p.parse_args()
    cli = args.cli.resolve()
    distinct = {}
    families = (("process-normal", "process-race", "public-checked", "refresh-normal", "refresh-race",
                 "controls-normal", "controls-race") if args.historical else
                ("qualified-normal", "qualified-race", "qualified-public", "held-ordered-normal", "held-ordered-race"))
    for family in families:
        directory = EVIDENCE / family
        assert directory.is_dir(), directory
        found = []
        for path in sorted(directory.rglob("*.sqlite.gz")):
            if path.name.endswith("-local.sqlite.gz") or path.name in ("before.sqlite.gz", "after.sqlite.gz"):
                found.append(path)
                raw = gzip.decompress(path.read_bytes())
                fingerprint = sha256(raw).hexdigest()
                distinct.setdefault(fingerprint, []).append(path)
        assert found, directory
    with tempfile.TemporaryDirectory() as work:
        db = Path(work) / "capture.sqlite"
        for digest, paths in distinct.items():
            db.write_bytes(gzip.decompress(paths[0].read_bytes()))
            cmd = subprocess.run([str(cli), "replay", "--db", str(db)], capture_output=True)
            if cmd.returncode or sha256(db.read_bytes()).hexdigest() != digest:
                raise RuntimeError((paths, cmd.returncode, cmd.stderr))
    print(json.dumps(dict(binary_sha256=sha256(cli.read_bytes()).hexdigest(),
                          images=sum(map(len, distinct.values())), distinct=len(distinct),
                          equivalence_classes={k: [str(p.relative_to(EVIDENCE)) for p in v]
                                               for k, v in distinct.items()}), sort_keys=True))


if __name__ == "__main__":
    main()
