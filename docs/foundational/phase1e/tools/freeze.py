#!/usr/bin/env python3
"""Identify every executable source, test input, embedded asset and qualification tool."""
from pathlib import Path
import hashlib, json, platform, struct, subprocess
ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / 'docs/foundational/phase1e'
paths = [ROOT / name for name in ('go.mod', 'go.sum')]
for folder in ('author', 'schema', 'runtime', 'effects', 'adapters', 'cmd', 'internal', 'research', 'examples'):
    paths.extend(p for p in (ROOT / folder).rglob('*') if p.is_file() and '__pycache__' not in p.parts)
paths.extend((DOC / 'tools').glob('*.py'))
# Frozen comparison inputs are source fixtures, distinct from newly generated results.
paths.append(DOC/'evidence/baseline/final-normal.records.jsonl.gz')
paths.append(DOC/'evidence/tooling-correction/before-source-manifest.json')
for mode in ('normal','race'):
    for suffix in ('records.jsonl.gz','go-test.jsonl.gz'):
        paths.append(DOC/f'evidence/tooling-correction/final-{mode}.{suffix}')
files, digest = [], hashlib.sha256()
for path in sorted(set(paths)):
    name = str(path.relative_to(ROOT)); raw = path.read_bytes(); encoded = name.encode()
    digest.update(struct.pack('>Q',len(encoded))+encoded+struct.pack('>Q',len(raw))+raw)
    files.append(dict(path=name,bytes=len(raw),sha256=hashlib.sha256(raw).hexdigest()))
def command(args):
    return subprocess.check_output(args,cwd=ROOT,text=True).strip()
manifest = dict(source_sha256=digest.hexdigest(), files=files,
    scope='All package sources, tests, testdata, embedded corpus, module metadata, Phase 1E qualification Python tools and their frozen comparison fixtures. Documents and generated evidence are separately archive-manifested.',
    baseline_commit='d708493646aeae3c1e70a03cf2bb2d7db974c127',
    head_at_freeze=command(['git','rev-parse','HEAD']),
    branch_at_freeze=command(['git','branch','--show-current']),
    identity='Working-tree file bytes, not a claim that HEAD already contains these files. Aggregate is sorted uint64-big-endian path length/path/content length/content.',
    platform=platform.platform(), go_version=command(['go','version']),
    go_environment=json.loads(command(['go','env','-json','GOOS','GOARCH','CGO_ENABLED','CC','GOVERSION'])),
    dependency=dict(module='github.com/mattn/go-sqlite3',version='v1.14.52',zip_sha256='ebd1fef8e9434fbdc2c94e35b9991df62d472c00170f57fc4f75daa69f658a1b'))
(DOC/'source-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps({k:v for k,v in manifest.items() if k!='files'},indent=2));print('source files',len(files))
