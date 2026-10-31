"""Audit a Phase 1P source tree against the integrated Phase 1O tree."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

BASE = "d0def139632c73936a3e821d93271a62267fa46d"
PHASE = "docs/foundational/phase1p/"
TITLE = "Direct preservation by the callable join (Phase 1P)."
PAPER = {f"paper/alan-dissertation.{ext}" for ext in ("md", "tex", "pdf")}


def git(*args):
    return subprocess.check_output(["git", *args])


def source(revision, path):
    return Path(path).read_bytes() if revision == "WORKTREE" else git("show", f"{revision}:{path}")


def section(revision, ext, start, following):
    path = f"paper/alan-dissertation.{ext}"
    text = source(revision, path).decode()
    if text.count(start) != 1:
        raise ValueError(f"{path}: expected one Phase 1P heading")
    left, tail = text.split(start)
    body, right = tail.split(following, 1)
    if (left + following + right).encode() != source(BASE, path):
        raise ValueError(f"{path}: inherited bytes changed")
    return body


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("revision", help="delivery commit or WORKTREE before the final commit")
    args = parser.parse_args()
    rev = args.revision
    target = [] if rev == "WORKTREE" else [rev]
    changed = git("diff", "--no-renames", "--name-only", "-z", BASE, *target, "--").decode().split("\0")
    if rev == "WORKTREE":
        changed += git("ls-files", "--others", "--exclude-standard", "-z").decode().split("\0")
    outside = [p for p in changed if p and p not in PAPER and not p.startswith(PHASE)]
    if outside:
        raise ValueError(f"change outside the approved scope: {outside}")
    md = section(rev, "md", "**" + TITLE + "**", "## 5. Pillar IV:")
    tex = section(rev, "tex", r"\paragraph{" + TITLE + "}", r"\section{Pillar IV:")
    if re.sub(r"\\path\{([^}]+)\}", r"`\1`", tex) != md:
        raise ValueError("the Phase 1P manuscript texts disagree")
    sources = [p for p in sorted(PAPER) if not p.endswith(".pdf")] + ["paper/references.bib"]
    print(json.dumps({
        "base": BASE, "revision": rev, "protected_scope": "pass",
        "inherited_manuscript_bytes": "identical", "new_text_alignment": "exact after markup",
        "source_sha256": {p: hashlib.sha256(source(rev, p)).hexdigest() for p in sources},
    }, indent=2))


if __name__ == "__main__":
    main()
