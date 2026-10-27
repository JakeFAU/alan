#!/usr/bin/env python3
"""Reuse verified archive extraction and existing collectors with an empty module cache."""
from pathlib import Path
import gzip,hashlib,json,os,subprocess,tarfile,tempfile,time
ROOT=Path(__file__).resolve().parents[4];DOC=ROOT/'docs/foundational/phase1g'
m=json.loads((DOC/'archive-manifest.json').read_text());archive=DOC/m['Archive']
assert hashlib.sha256(archive.read_bytes()).hexdigest()==m['SHA256']
OUT=DOC/'evidence/task4/extracted';OUT.mkdir(parents=True,exist_ok=True)
assert not (OUT/'reproduction.json').exists()
commands=[];started=time.monotonic()
with tempfile.TemporaryDirectory(prefix='alan-phase1g-offline-') as tmp:
    folder=Path(tmp)
    with tarfile.open(archive) as tar:
        assert len(tar.getmembers())==len(m['Files'])
        for f in m['Files']:
            member=tar.getmember(f['Path']);raw=tar.extractfile(member).read()
            assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
            assert len(raw)==f['Bytes'] and hashlib.sha256(raw).hexdigest()==f['SHA256']
        tar.extractall(folder,filter='data')
    env=dict(os.environ,GOTOOLCHAIN='local',GOMODCACHE=str(folder/'module-cache'),GOCACHE='/tmp/alan-phase1f-go-build',GOPROXY='file://'+str(folder/'dependencies'),GOSUMDB='off',GOFLAGS='-mod=readonly',PYTHONDONTWRITEBYTECODE='1',ALAN_QUALIFICATION_OUT=str(OUT))
    env.pop('OPENAI_API_KEY',None)
    assert not Path(env['GOMODCACHE']).exists()
    planned=[]
    for mode in ('default','normal','race','consumer','consumer-race'):
        planned.append(['python3','docs/foundational/phase1g/tools/qualify.py',mode])
    for mode in ('default','normal','race'):
        planned.append(['python3','docs/foundational/phase1g/tools/compare.py',mode])
    planned.append(['python3','docs/foundational/phase1g/tools/checks.py'])
    for i,cmd in enumerate(planned):
        start=time.monotonic();p=subprocess.run(cmd,cwd=folder,env=env,capture_output=True,timeout=2400)
        stem=f'{i+1:02d}'
        (OUT/(stem+'.stdout.gz')).write_bytes(gzip.compress(p.stdout,mtime=0));(OUT/(stem+'.stderr.gz')).write_bytes(gzip.compress(p.stderr,mtime=0))
        commands.append(dict(Command=cmd,Directory=str(folder),ExitCode=p.returncode,ElapsedSeconds=round(time.monotonic()-start,3),Stdout=stem+'.stdout.gz',Stderr=stem+'.stderr.gz'))
        result=dict(SourceID=m['SourceID'],ArchiveSHA256=m['SHA256'],VerifiedMembers=len(m['Files']),InitialModuleCacheEmpty=True,CredentialAbsent=True,Environment={k:env[k] for k in ('GOTOOLCHAIN','GOMODCACHE','GOCACHE','GOPROXY','GOSUMDB','GOFLAGS')},Commands=commands,ElapsedSeconds=round(time.monotonic()-started,3))
        (OUT/'reproduction.json').write_text(json.dumps(result,indent=2)+'\n')
        print(stem,cmd,p.returncode,flush=True)
        assert p.returncode==0,commands[-1]
    manifest=json.loads((folder/'docs/foundational/phase1g/source-manifest.json').read_text())
    for f in manifest['files']:
        raw=(folder/f['path']).read_bytes();assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256']
    result.update(Result='PASS',SourceUnchanged=True)
    (OUT/'reproduction.json').write_text(json.dumps(result,indent=2)+'\n')
