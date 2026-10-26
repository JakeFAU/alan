#!/usr/bin/env python3
"""Verify and rerun the packaged source in an isolated tree with a local dependency proxy."""
import datetime
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import time

ROOT=Path(__file__).resolve().parents[4]
DOC=ROOT/'docs/foundational/phase1f'
OUT=DOC/'evidence'
manifest=json.loads((DOC/'archive-manifest.json').read_text())
archive=DOC/'source.tar.gz'
assert hashlib.sha256(archive.read_bytes()).hexdigest()==manifest['SHA256']
work=Path(tempfile.mkdtemp(prefix='alan-phase1f-archive-reproduction-'))
with tarfile.open(archive) as tar:
    expected={f['Path']:f for f in manifest['Files']}
    assert len(tar.getmembers())==len(expected)
    for member in tar.getmembers():
        assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
        raw=tar.extractfile(member).read();entry=expected[member.name]
        assert len(raw)==entry['Bytes'] and hashlib.sha256(raw).hexdigest()==entry['SHA256']
        destination=work/member.name;destination.parent.mkdir(parents=True,exist_ok=True);destination.write_bytes(raw)
source=json.loads((work/'docs/foundational/phase1f/source-manifest.json').read_text())
assert source['source_sha256']==manifest['SourceID']
for entry in source['files']:
    raw=(work/entry['path']).read_bytes()
    assert len(raw)==entry['bytes'] and hashlib.sha256(raw).hexdigest()==entry['sha256']
env={k:v for k,v in os.environ.items() if k not in ('OPENAI_API_KEY','ALAN_BARRIER')}
env.update(GOTOOLCHAIN='local',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build',
           GOMODCACHE=str(work/'module-cache'),GOPROXY=(work/'dependencies').as_uri(),GOSUMDB='off')
commands=[]
for name,cmd in [('modules',['go','mod','download']),('default',['go','test','-json','-count=1','-timeout','20m','./...'])]:
    start=time.monotonic();started=datetime.datetime.now(datetime.timezone.utc).isoformat()
    proc=subprocess.run(cmd,cwd=work,env=env,capture_output=True)
    raw=proc.stdout;path=OUT/f'archive-{name}.stdout.gz';path.write_bytes(gzip.compress(raw,mtime=0))
    row={'Command':cmd,'Directory':str(work),'StartedUTC':started,'ElapsedSeconds':round(time.monotonic()-start,3),
         'ExitCode':proc.returncode,'Stderr':proc.stderr.decode(),'Stdout':str(path.relative_to(ROOT)),
         'RawSHA256':hashlib.sha256(raw).hexdigest()}
    if name=='default':
        events=[json.loads(line) for line in raw.splitlines()]
        row['PassingEvents']=sum(e.get('Action')=='pass' and bool(e.get('Test')) for e in events)
        row['Failures']=[e for e in events if e.get('Action')=='fail']
    commands.append(row)
    if proc.returncode:break
result={'Kind':'archive_reproduction','SourceID':source['source_sha256'],'ArchiveSHA256':manifest['SHA256'],
        'VerifiedMembers':len(expected),'VerifiedSourceFiles':len(source['files']),
        'ScriptSHA256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        'Environment':{k:env[k] for k in ('GOTOOLCHAIN','GOFLAGS','GOCACHE','GOMODCACHE','GOPROXY','GOSUMDB')},
        'CredentialPresent':False,'Commands':commands,'SourceDirectory':str(work),
        'Scope':'Isolated archive source with initially empty module cache and file-only pinned dependency proxy. Controlled tests may use loopback; no provider credential.'}
(OUT/'archive-reproduction.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2),flush=True)
assert len(commands)==2 and all(c['ExitCode']==0 for c in commands)
