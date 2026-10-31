"""Verify Phase 1T scope, input freeze, evidence index and source reconciliation."""

from hashlib import sha256
import gzip
import json
from pathlib import Path
import re
import subprocess
import sys

phase = Path("docs/foundational/phase1t")
evidence = phase / "evidence"
base = "153e981879cee32a67fd4784e9c162542e2dc421"


def git(*args):
    return subprocess.check_output(["git", *args])


def digest(data):
    return sha256(data).hexdigest()


def verify_changes():
    git("diff", "--check", base)
    info = json.loads((evidence / "base.json").read_text())
    assert (
        info["base"] == base
        and info["tree"] == git("rev-parse", base + "^{tree}").decode().strip()
    )
    for path, h in info["inputs"].items():
        assert digest(Path(path).read_bytes()) == h, path
    assert digest((phase / "assignment.md").read_bytes()) == info["packet_sha256"]
    paper = {"paper/alan-dissertation." + ext for ext in ("md", "tex", "pdf")}
    files = dict(
        (p, s)
        for s, p in (
            line.split("\t")
            for line in git("diff", "--name-status", "--no-renames", base)
            .decode()
            .splitlines()
        )
    )
    files.update(
        {
            p: "A"
            for p in git("ls-files", "--others", "--exclude-standard")
            .decode()
            .splitlines()
        }
    )
    for p, s in files.items():
        if p in paper:
            assert s == "M", p
        else:
            assert s == "A" and (
                p.startswith(
                    (str(phase) + "/", "examples/owningtree/", "research/owningtree/")
                )
                or re.fullmatch(
                    r"internal/substitutiontest/owning_tree_[a-z_]+_test\.go", p
                )
            ), p
    for p, h in info["protected"].items():
        assert git("rev-parse", base + ":" + p).decode().strip() == h, p
        assert git("diff", base, "--", p) == b"", p
    assert Path("paper/references.bib").read_bytes() == git(
        "show", base + ":paper/references.bib"
    )
    for doc in phase.rglob("*.md"):
        prose = re.sub(r"```.*?```", "", doc.read_text(), flags=re.S)
        for link in re.findall(r"\]\(([^)]+)\)", prose):
            path = link.split("#")[0]
            if path and "://" not in path:
                assert (doc.parent / path).exists(), (doc, link)
    return len(files)


def verify_paper():
    # Both documents receive two insertions. Remove them and recover the exact base.
    sections = []
    for ext, opening, closing, after_first, after_second in (
        (
            "md",
            "**Finite owning trees (Phase 1T).** ",
            "**Series 1 conclusion.** ",
            "## 5. Pillar IV",
            "### 8.2",
        ),
        (
            "tex",
            r"\paragraph{Finite owning trees (Phase 1T).} ",
            r"\paragraph{Series 1 conclusion.} ",
            r"\section{Pillar IV",
            r"\bibliographystyle",
        ),
    ):
        path = "paper/alan-dissertation." + ext
        body = Path(path).read_text()
        added = []
        for start, end in ((opening, after_first), (closing, after_second)):
            assert body.count(start) == 1 and body.count(end) == 1, path
            left, tail = body.split(start)
            part, right = tail.split(end)
            added.append(part)
            body = left + end + right
        assert body.encode() == git("show", base + ":" + path), path
        sections.append(added)
    assert sections[0] == sections[1], "The two manuscript additions differ"

    inspection = json.loads((evidence / "paper-inspection.json").read_text())
    record_path = evidence / "paper-build.json.gz"
    record = json.load(gzip.open(record_path))
    assert record["exit"] == 0 and record["argv"] == [
        "tectonic",
        "alan-dissertation.tex",
    ]
    assert digest(record_path.read_bytes()) == inspection["log_sha256"]
    assert (
        digest(Path("paper/alan-dissertation.pdf").read_bytes())
        == inspection["pdf_sha256"]
    )
    for key, name in (
        ("md", "alan-dissertation.md"),
        ("tex", "alan-dissertation.tex"),
        ("references.bib", "references.bib"),
    ):
        assert (
            digest((Path("paper") / name).read_bytes())
            == inspection["source_sha256"][key]
        )


def verify_commands():
    for label, prefix, count in (
        ("automatic", "runtime", 80),
        ("manual", "manual", 14),
    ):
        expected = set((evidence / (label + "-leaves.txt")).read_text().splitlines())
        assert len(expected) == count
        for mode in ("normal", "race"):
            name = prefix + "-" + mode + "-final"
            record = json.load(gzip.open(evidence / (name + ".json.gz")))
            assert (
                record["exit"] == 0
                and digest(record["output"].encode()) == record["output_sha256"]
            ), name
            passed = set(re.findall(r"--- PASS: (Test\S+) \(", record["output"]))
            leaves = {
                p for p in passed if not any(q.startswith(p + "/") for q in passed)
            }
            assert leaves == expected, (name, leaves ^ expected)
            files = [
                "examples/owningtree/tree.go",
                "examples/owningtree/validate.go",
                "internal/substitutiontest/owning_tree_runtime_test.go",
            ]
            if label == "manual":
                files.append("internal/substitutiontest/owning_tree_fault_test.go")
                repeat = json.load(
                    gzip.open(evidence / ("manual-" + mode + "-rejected.json.gz"))
                )
                schedule = "internal/substitutiontest/owning_tree_schedule_test.go"
                assert repeat["source_sha256"][schedule] == digest(
                    Path(schedule).read_bytes()
                )
                assert (
                    repeat["exit"] == 0
                    and "--- PASS: TestOwningTreeDelayedFrontier/right ("
                    in repeat["output"]
                )
            for f in files:
                assert record["source_sha256"][f] == digest(Path(f).read_bytes()), (
                    name,
                    f,
                )


def verify_runs():
    index = json.loads((evidence / "run-index.json").read_text())
    files = set()
    for item in index:
        p = evidence / item["file"]
        assert p not in files, p
        files.add(p)
        data = gzip.decompress(p.read_bytes())
        h = digest(data.removesuffix(b"\n"))
        assert h == item["value_sha256"]
        run = json.load(gzip.open(evidence / item["command"]))
        assert run["exit"] == 0 and h in run["output"], p
    assert files == set((evidence / "runs").rglob("*.json.gz")) and len(files) == 174
    return len(files)


def verify_index(write):
    index = evidence / "identity-index.json"
    # The index excludes itself. No artifact includes the digest of itself.
    inventory = (
        git(
            "ls-files",
            "--cached",
            "--others",
            "--exclude-standard",
            "--",
            str(phase),
            "examples/owningtree",
            "research/owningtree",
            "internal/substitutiontest/owning_tree_*_test.go",
            "paper/alan-dissertation.md",
            "paper/alan-dissertation.tex",
            "paper/alan-dissertation.pdf",
        )
        .decode()
        .splitlines()
    )
    expected = {p: digest(Path(p).read_bytes()) for p in inventory if Path(p) != index}
    if write:
        index.write_text(json.dumps(expected, indent=2, sort_keys=True) + "\n")
    else:
        assert json.loads(index.read_text()) == expected, "phase identity index differs"


if __name__ == "__main__":
    write = sys.argv[1:] == ["--write-index"]
    assert write or len(sys.argv) == 1
    print(
        "Verified",
        verify_changes(),
        "change paths and",
        verify_runs(),
        "raw/replay records.",
    )
    verify_paper()
    verify_commands()
    verify_index(write)
