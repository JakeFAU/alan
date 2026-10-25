#!/usr/bin/env python3
"""Bundle frozen sources plus an already verified local dependency proxy."""
from pathlib import Path
import gzip, hashlib, io, json, struct, sys, tarfile
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/'docs/foundational/phase1d1'
proxy=Path(sys.argv[1]); manifest=json.loads((DOC/'source-manifest.json').read_text())
h=hashlib.sha256()
for f in manifest['files']:
    raw=(ROOT/f['path']).read_bytes(); name=f['path'].encode()
    assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256']
    h.update(struct.pack('>Q',len(name))+name+struct.pack('>Q',len(raw))+raw)
assert h.hexdigest()==manifest['source_sha256']
manifest['qualification_tools']=[dict(path=str(p.relative_to(ROOT)),bytes=p.stat().st_size,sha256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in sorted((DOC/'tools').glob('*.py'))]
manifest['review_probe_sources']=[dict(path=str(p.relative_to(ROOT)),bytes=p.stat().st_size,sha256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in sorted([p for folder in ('review','final-review') for p in (DOC/'evidence'/folder).glob('*.go.txt')])]
(DOC/'source-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
files={f['path']:(ROOT/f['path']).read_bytes() for f in manifest['files']}
for name in ['README.md','LICENSE','docs/foundational/phase1d1/source-manifest.json']:
    files[name]=(ROOT/name).read_bytes()
for p in (ROOT/'docs/foundational').glob('*.md'):files[str(p.relative_to(ROOT))]=p.read_bytes()
for phase in ('phase1','phase1b','phase1c','phase1d'):
    for p in (ROOT/'docs/foundational'/phase).glob('*.md'):files[str(p.relative_to(ROOT))]=p.read_bytes()
for name in ('00-plan.md','01-dispatch-contract.md','04-runbook.md','requirements.md'):
    files[str((DOC/name).relative_to(ROOT))]=(DOC/name).read_bytes()
for p in (DOC/'tools').glob('*.py'):files[str(p.relative_to(ROOT))]=p.read_bytes()
# The checks tool compares against this immutable baseline payload.
p=DOC/'evidence/baseline/normal.records.jsonl.gz';files[str(p.relative_to(ROOT))]=p.read_bytes()
for p in proxy.rglob('*'):
    if p.is_file():files[str(p.relative_to(proxy))]=p.read_bytes()
expected=manifest['dependency']['module_files'] if 'module_files' in manifest['dependency'] else None
# The driver archive's bytes are pinned independently of the module-cache location.
zipfile=files['dependencies/github.com/mattn/go-sqlite3/@v/v1.14.52.zip']
assert hashlib.sha256(zipfile).hexdigest()=='ebd1fef8e9434fbdc2c94e35b9991df62d472c00170f57fc4f75daa69f658a1b'
with (DOC/'source.tar.gz').open('wb') as out:
    with gzip.GzipFile(fileobj=out,mode='wb',filename='',mtime=0) as z:
        with tarfile.open(fileobj=z,mode='w') as tar:
            for name,raw in sorted(files.items()):
                info=tarfile.TarInfo(name);info.size=len(raw);info.mode=0o644;info.mtime=0
                tar.addfile(info,io.BytesIO(raw))
archive=(DOC/'source.tar.gz').read_bytes()
result=dict(SourceID=manifest['source_sha256'],Archive='source.tar.gz',Bytes=len(archive),SHA256=hashlib.sha256(archive).hexdigest(),
    Scope='Complete frozen Go/module sources, contracts, README, tools, baseline comparison input and pinned dependency proxy. Generated final evidence/reviews/publication remain alongside this archive to avoid self-reference. Installed Go/C toolchain is not bundled.',
    Files=[dict(Path=name,Bytes=len(raw),SHA256=hashlib.sha256(raw).hexdigest()) for name,raw in sorted(files.items())])
(DOC/'archive-manifest.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({k:v for k,v in result.items() if k!='Files'},indent=2));print('members',len(files))
