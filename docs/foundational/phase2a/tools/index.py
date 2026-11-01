"""Write or verify the complete Phase 2A source and evidence inventory."""
import argparse
import hashlib
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
PHASE = "docs/foundational/phase2a/"
INDEX = PHASE + "evidence/index.sha256"
SIGNOFF = PHASE + "evidence/whole-review.md"


def inventory():
    paths = set(subprocess.check_output(
        ["git", "ls-files", "-co", "--exclude-standard", "-z"], cwd=ROOT
    ).decode().split("\0"))
    baseline = {
        line.split("  ", 1)[1]
        for line in (ROOT / PHASE / "evidence/base-inputs.sha256").read_text().splitlines()
    }
    selected = {
        path for path in paths
        if path in baseline
        or path == "AGENTS.md"
        or path.startswith((PHASE, "examples/effectboundary/", "research/effectprotocol/"))
        or path.endswith((".go", ".mod", ".sum", ".c", ".h"))
        or path in {
            "paper/alan-dissertation.md", "paper/alan-dissertation.tex",
            "paper/alan-dissertation.pdf", "paper/references.bib",
        }
    }
    selected.difference_update({INDEX, SIGNOFF})
    return "".join(
        hashlib.sha256((ROOT / path).read_bytes()).hexdigest() + "  " + path + "\n"
        for path in sorted(selected)
    )


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--write", action="store_true")
    args = parser.parse_args()
    actual = inventory()
    if args.write:
        (ROOT / INDEX).write_text(actual)
    elif (ROOT / INDEX).read_text() != actual:
        raise SystemExit("source/evidence inventory differs")
    print(f"PASS {len(actual.splitlines())} source/evidence files")
    print(hashlib.sha256(actual.encode()).hexdigest(), INDEX)
    print("The index and whole-phase signoff are bound by the containing Git commit.")


if __name__ == "__main__":
    main()
