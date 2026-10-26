#!/usr/bin/env python3
"""Freeze exact current sources and reproducible fixture inputs; no generated result self-hashes."""
from pathlib import Path
import hashlib, json, platform, struct, subprocess
ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / "docs/foundational/phase1g"
base = json.loads((ROOT / "docs/foundational/phase1f/source-manifest.json").read_text())
paths = {ROOT / f["path"] for f in base["files"]}
for folder in ("compose", "author", "schema", "runtime", "effects", "adapters", "cmd", "internal", "research", "examples"):
    paths.update(p for p in (ROOT/folder).rglob("*") if p.is_file() and "__pycache__" not in p.parts)
paths.update((DOC/"tools").glob("*.py"))
# Inherit all earlier evaluation/reproduction inputs unchanged.
paths.add(ROOT/"go.mod"); paths.add(ROOT/"go.sum")
paths.update((DOC/"evidence/fixtures").glob("*"))
paths.update((DOC/"evidence/task3").glob("input-*.json"))
paths.add(ROOT/"docs/foundational/phase1f/source-manifest.json")
for mode in ("default", "normal", "race"):
    paths.add(ROOT/f"docs/foundational/phase1f/evidence/final-{mode}.records.jsonl.gz")
files, digest = [], hashlib.sha256()
for p in sorted(paths):
    name = str(p.relative_to(ROOT)); raw=p.read_bytes(); n=name.encode()
    digest.update(struct.pack(">Q",len(n))+n+struct.pack(">Q",len(raw))+raw)
    files.append(dict(path=name,bytes=len(raw),sha256=hashlib.sha256(raw).hexdigest()))
def command(args): return subprocess.check_output(args,cwd=ROOT,text=True).strip()
manifest=dict(source_sha256=digest.hexdigest(),files=files,
    scope="All current Go/package/test/testdata/assets and module bytes, inherited Phase1F source/qualification inputs, Phase1G composition and nested consumer modules, Python tools and frozen acceptance inputs. Documentation and generated evidence are separately archive-manifested.",
    baseline_commit="49ae5f896d88f080a2e10cbfef1b537d27f55df5",baseline_source=base["source_sha256"],
    head_at_freeze=command(["git","rev-parse","HEAD"]),branch_at_freeze=command(["git","branch","--show-current"]),
    identity="Sorted uint64-big-endian path length/path/content length/content. Working-tree identity, not a claim HEAD contains these changes.",
    platform=platform.platform(),go_version=command(["go","version"]),
    go_environment=json.loads(command(["go","env","-json","GOOS","GOARCH","CGO_ENABLED","CC","GOVERSION"])),dependency=base["dependency"])
(DOC/"source-manifest.json").write_text(json.dumps(manifest,indent=2)+"\n")
print(json.dumps({k:v for k,v in manifest.items() if k!="files"},indent=2));print("source files",len(files))
