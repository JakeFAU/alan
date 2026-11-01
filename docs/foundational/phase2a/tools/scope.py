"""Verify base inventory and the exact permitted path boundary of Phase 2A."""
from pathlib import Path
import hashlib
import subprocess

ROOT = Path(__file__).resolve().parents[4]
BASE = "18d35a2b312cf7180926d4b4cc5c7c2094cd68e2"
PHASE = "docs/foundational/phase2a/"
EXACT = {
    "cmd/alan-resource/main.go", "internal/executor/external.go", "internal/executor/phase2a_conflict_test.go",
    "internal/durabletest/phase2_process_test.go", "internal/durabletest/phase2_crash_test.go",
    "internal/durabletest/phase2a_authority_test.go", "internal/durabletest/phase2a_withholding_test.go",
    "paper/alan-dissertation.md", "paper/alan-dissertation.tex", "paper/alan-dissertation.pdf",
}


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT)


def verify():
    manifest = (ROOT / PHASE / "evidence/base-inputs.sha256").read_text().splitlines()
    for line in manifest:
        expected, path = line.split("  ", 1)
        assert hashlib.sha256(git("show", BASE + ":" + path)).hexdigest() == expected, path
    changes = set(git("diff", "--name-only", "-z", BASE).decode().split("\0"))
    changes.update(git("ls-files", "--others", "--exclude-standard", "-z").decode().split("\0"))
    changes.discard("")
    for path in sorted(changes):
        assert path in EXACT or path.startswith((PHASE, "research/effectprotocol/", "examples/effectboundary/")), path
    assert git("show", BASE + ":paper/references.bib") == (ROOT / "paper/references.bib").read_bytes()
    frozen = (ROOT / PHASE / "02-protocol.md").read_bytes()
    assert hashlib.sha256(frozen).hexdigest() == "a9d159bfb487d1fed9b12792de9efede253b1fc3753e827281be01179056c553"
    for fmt in ("tex", "md"):
        excerpt = (ROOT / PHASE / ("evidence/inherited-dichotomy." + fmt)).read_bytes()
        assert excerpt in git("show", BASE + ":paper/alan-dissertation." + fmt)
    print("PASS base input inventory, frozen protocol, historical theorem excerpts, untouched bibliography, Series 1 and path scope")


if __name__ == "__main__":
    verify()
