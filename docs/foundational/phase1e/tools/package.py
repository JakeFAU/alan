#!/usr/bin/env python3
"""Bundle frozen source and the pinned local module proxy, with deterministic members."""
from pathlib import Path
import gzip, hashlib, io, json, struct, subprocess, tarfile
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/'docs/foundational/phase1e'
manifest=json.loads((DOC/'source-manifest.json').read_text()); files={}; digest=hashlib.sha256()
for f in manifest['files']:
    raw=(ROOT/f['path']).read_bytes(); name=f['path'].encode()
    assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256'],f['path']
    digest.update(struct.pack('>Q',len(name))+name+struct.pack('>Q',len(raw))+raw)
    files[f['path']]=raw
assert digest.hexdigest()==manifest['source_sha256']
for name in ('README.md','LICENSE','docs/foundational/phase1e/source-manifest.json'):
    files[name]=(ROOT/name).read_bytes()
for p in (ROOT/'docs/foundational').rglob('*.md'):
    # Reports are useful alongside source; do not import user extracts or old nested snapshots.
    if '/source/' not in str(p.relative_to(ROOT)):
        files[str(p.relative_to(ROOT))]=p.read_bytes()
p=DOC/'evidence/baseline/final-normal.records.jsonl.gz'; files[str(p.relative_to(ROOT))]=p.read_bytes()
cache=Path(subprocess.check_output(['go','env','GOMODCACHE'],cwd=ROOT,text=True).strip())
module='github.com/mattn/go-sqlite3/@v/'
for suffix in ('info','mod','zip'):
    name='v1.14.52.'+suffix
    files['dependencies/'+module+name]=(cache/'cache/download'/module/name).read_bytes()
files['dependencies/'+module+'list']=b'v1.14.52\n'
assert hashlib.sha256(files['dependencies/'+module+'v1.14.52.zip']).hexdigest()==manifest['dependency']['zip_sha256']
with (DOC/'source.tar.gz').open('wb') as out:
    with gzip.GzipFile(fileobj=out,mode='wb',filename='',mtime=0) as z:
        with tarfile.open(fileobj=z,mode='w') as tar:
            for name,raw in sorted(files.items()):
                info=tarfile.TarInfo(name);info.size=len(raw);info.mode=0o644;info.mtime=0
                tar.addfile(info,io.BytesIO(raw))
archive=(DOC/'source.tar.gz').read_bytes()
result=dict(SourceID=manifest['source_sha256'],Archive='source.tar.gz',Bytes=len(archive),SHA256=hashlib.sha256(archive).hexdigest(),
    Scope='Complete current Go/module sources, tests/testdata/assets, qualification tools, contracts/reports, baseline comparison input and pinned dependency proxy. Later generated evidence/reviews/publication remain alongside archive to avoid recursive identity. Go/C toolchains are not bundled.',
    Files=[dict(Path=name,Bytes=len(raw),SHA256=hashlib.sha256(raw).hexdigest()) for name,raw in sorted(files.items())])
(DOC/'archive-manifest.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({k:v for k,v in result.items() if k!='Files'},indent=2)); print('members',len(files))
