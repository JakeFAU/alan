"""Phase1H source and ordered-evidence primitives; never validates H using G's list."""

from pathlib import Path
import gzip, hashlib, json, os, re, struct, subprocess

ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / "docs/foundational/phase1h"


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def inventory(root=ROOT):
    paths = set()
    for folder in root.iterdir():
        if (
            folder.is_dir()
            and folder.name not in ("docs", "dependencies")
            and not folder.name.startswith(".")
        ):
            paths.update(
                p
                for p in folder.rglob("*")
                if p.is_file()
                and "__pycache__" not in p.parts
                and ".git" not in p.parts
            )
    for folder in (root / ".github",):
        if folder.exists():
            paths.update(p for p in folder.rglob("*") if p.is_file())
    for name in (
        "AGENTS.md",
        "go.mod",
        "go.sum",
        "README.md",
        "LICENSE",
        ".gitignore",
        ".envrc",
    ):
        if (root / name).exists():
            paths.add(root / name)
    for phase in (
        "phase1a",
        "phase1b",
        "phase1c",
        "phase1d",
        "phase1e",
        "phase1f",
        "phase1g",
        "phase1h",
    ):
        doc = root / "docs/foundational" / phase
        paths.update((doc / "tools").glob("*.py"))
        if phase == "phase1h":
            # Contributing specifications only; generated reports may embed SourceID.
            for name in (
                "assignment.md",
                "01-adaptive-contract.md",
                "phase1h-plan.md",
                "04-runbook.md",
                "05-decisions.md",
            ):
                if (doc / name).exists():
                    paths.add(doc / name)
        paths.update((doc / "evidence/fixtures").glob("*"))
    # Exact old comparison inputs and implementation, not historical nested archives.
    for name in (
        "docs/foundational/05-decisions.md",
        "docs/foundational/phase1e/source-manifest.json",
        "docs/foundational/phase1e/tools/comparison.py",
        "docs/foundational/phase1h/evidence/task2/collect-default-inputs.py",
        "docs/foundational/phase1h/evidence/task3/harness-review-evidence.py",
        "docs/foundational/phase1h/evidence/task3/comparison-inputs/runtime_conformance-before.go.txt",
    ):
        if (root / name).exists():
            paths.add(root / name)
    for mode in ("default", "normal", "race"):
        p = root / f"docs/foundational/phase1e/evidence/final-{mode}.records.jsonl.gz"
        if p.exists():
            paths.add(p)
    return {
        str(p.relative_to(root)): sha(p.read_bytes())
        for p in sorted(paths)
        if p.is_file()
    }


def source_id(files):
    return sha(json.dumps(files, sort_keys=True, separators=(",", ":")).encode())


def environment():
    env = dict(os.environ)
    env.pop("OPENAI_API_KEY", None)
    env.update(GOTOOLCHAIN="local", GOFLAGS="-mod=readonly")
    env.setdefault("GOPROXY", "off")
    env.setdefault("GOSUMDB", "off")
    env.setdefault("GOCACHE", "/tmp/alan-phase1f-go-build")
    env["PYTHONDONTWRITEBYTECODE"] = "1"
    return env


MARKER = re.compile(
    r"^\s*(?:[^\s:]+:\d+:\s*)?(QUALIFICATION|COMPOSITION|SUPERVISION|CONFORMANCE|DURABLE|ASYNC|EVIDENCE|INVESTIGATION|WORKTREE|ADAPTIVE|ADAPTIVE_BYTES|ADAPTIVE_ACCEPTANCE_BOUNDARY) (.*)$"
)


def parse(raw, source, mode):
    buffers = {}
    records = []
    passes = []
    failures = []
    packages = []
    for line in raw.splitlines():
        e = json.loads(line)
        key = (e.get("Package", ""), e.get("Test", ""))
        if e.get("Action") == "output":
            value = buffers.get(key, "") + e.get("Output", "")
            while "\n" in value:
                one, _, value = value.partition("\n")
                matched = MARKER.match(one)
                if matched:
                    payload = matched[2]
                    records.append(
                        dict(
                            SourceID=source,
                            Mode=mode,
                            Package=key[0],
                            Test=key[1],
                            RecordIndex=len(records) + 1,
                            Marker=matched[1],
                            Evidence=json.loads(payload)
                            if payload.startswith(("{", "["))
                            else dict(Kind="text_trace", Text=payload),
                        )
                    )
                elif (
                    plain := re.sub(r"^\s*[^\s:]+:\d+:\s*", "", one).strip()
                ).startswith("{"):
                    try:
                        v = json.loads(plain)
                    except json.JSONDecodeError:
                        continue
                    if isinstance(v, dict) and v.get("type") == "ASYNC":
                        records.append(
                            dict(
                                SourceID=source,
                                Mode=mode,
                                Package=key[0],
                                Test=key[1],
                                RecordIndex=len(records) + 1,
                                Marker="ASYNC",
                                Evidence=v,
                            )
                        )
            buffers[key] = value
        elif e.get("Action") == "pass" and key[1]:
            passes.append("/".join(key))
        elif e.get("Action") == "fail":
            failures.append(e)
        if e.get("Action") in ("pass", "fail", "skip") and not key[1]:
            packages.append(e)
    if any(v.strip() for v in buffers.values()):
        raise ValueError("unterminated Go output")
    return dict(records=records, passes=passes, failures=failures, packages=packages)
