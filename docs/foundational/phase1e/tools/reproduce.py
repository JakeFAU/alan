#!/usr/bin/env python3
"""Verify and run the complete snapshot using an initially empty module cache."""
from pathlib import Path
import gzip,hashlib,json,os,subprocess,tarfile,tempfile,time
ROOT=Path(__file__).resolve().parents[4]; DOC=ROOT/'docs/foundational/phase1e'
m=json.loads((DOC/'archive-manifest.json').read_text()); archive=DOC/m['Archive']
assert hashlib.sha256(archive.read_bytes()).hexdigest()==m['SHA256']
commands=[];started=time.monotonic()
with tempfile.TemporaryDirectory(prefix='alan-phase1e-offline-') as tmp:
    folder=Path(tmp)
    with tarfile.open(archive) as tar:
        assert len(tar.getmembers())==len(m['Files'])
        for f in m['Files']:
            member=tar.getmember(f['Path']); raw=tar.extractfile(member).read()
            assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
            assert len(raw)==f['Bytes'] and hashlib.sha256(raw).hexdigest()==f['SHA256']
        tar.extractall(folder,filter='data')
    env=dict(os.environ,GOTOOLCHAIN='local',GOMODCACHE=str(folder/'module-cache'),GOCACHE='/tmp/alan-phase1e-offline-build',GOPROXY='file://'+str(folder/'dependencies'),GOSUMDB='off',GOFLAGS='-mod=readonly')
    env.pop('OPENAI_API_KEY',None)
    env['PYTHONDONTWRITEBYTECODE']='1'
    assert not Path(env['GOMODCACHE']).exists()
    for cmd in [['python3','-m','unittest','discover','-s','docs/foundational/phase1e/tools','-p','test_comparison.py','-v'],['go','test','-count=1','-tags','integration','./...'],['go','mod','verify'],['go','vet','-tags','integration','./...']]:
        start=time.monotonic(); p=subprocess.run(cmd,cwd=folder,env=env,text=True,capture_output=True,timeout=900)
        commands.append(dict(Command=cmd,ExitCode=p.returncode,ElapsedSeconds=round(time.monotonic()-start,3),Stdout=p.stdout,Stderr=p.stderr))
        assert p.returncode==0,commands[-1]
    result=dict(SourceID=m['SourceID'],ArchiveSHA256=m['SHA256'],VerifiedMembers=len(m['Files']),InitialModuleCacheEmpty=True,
        Environment={k:env[k] for k in ('GOTOOLCHAIN','GOMODCACHE','GOCACHE','GOPROXY','GOSUMDB','GOFLAGS')},Commands=commands,ElapsedSeconds=round(time.monotonic()-started,3),Result='PASS')
(DOC/'evidence/archive-reproduction.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
