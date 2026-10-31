"""Check the complete Phase 1S delta and recover both inherited manuscript sources."""
import re
import subprocess
from pathlib import Path

BASE = "61f44fc2c62772065ff3d483f52c0235670f586c"
DOCS = "docs/foundational/phase1s/"
ADDITIONS = {
    "research/substitution/retained_b_correspondence_test.go",
    "research/substitution/retained_b_witness_test.go",
    "research/substitution/retained_b_audit_test.go",
    "internal/substitutiontest/retained_b_fault_test.go",
    "internal/substitutiontest/retained_b_closure_test.go",
}
PAPER = {f"paper/alan-dissertation.{ext}" for ext in ("md", "tex", "pdf")}


def git(*args):
    return subprocess.check_output(["git", *args])


def recover(name, begin, end):
    content = Path(name).read_bytes()
    assert content.count(begin) == content.count(end) == 1, name
    left = content.index(begin)
    right = content.index(end, left)
    assert content[:left] + content[right:] == git("show", BASE + ":" + name), name


git("diff", "--check", BASE, "--")
delta = {}
for row in git("diff", "--name-status", "--no-renames", BASE, "--").decode().splitlines():
    status, path = row.split("\t")
    delta[path] = status
for path in git("ls-files", "--others", "--exclude-standard").decode().splitlines():
    delta[path] = "A"
for path, status in delta.items():
    assert status == ("M" if path in PAPER else "A"), (path, status)
    assert path.startswith(DOCS) or path in ADDITIONS or path in PAPER, path
assert (PAPER | ADDITIONS).issubset(delta)
recover("paper/alan-dissertation.md", b"**Original conformance of retained B (Phase 1S).**", b"## 5. Pillar IV")
recover("paper/alan-dissertation.tex", rb"\paragraph{Original conformance of retained B (Phase 1S).}", rb"\section{Pillar IV")
assert Path("paper/references.bib").read_bytes() == git("show", BASE + ":paper/references.bib")
for p in Path(DOCS).rglob("*.md"):
    content = re.sub(r"```.*?```", "", p.read_text(), flags=re.S)
    for target in re.findall(r"\]\(([^)]*)\)", content):
        dest = target.split("#")[0]
        if dest and "://" not in dest:
            assert (p.parent / dest).exists(), (p, dest)
print("Scope, whitespace, local links, restored manuscript sources and protected bibliography verified.")
