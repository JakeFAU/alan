#!/usr/bin/env python3
"""Frozen-source auxiliary checks; retain actual command exits and complete output."""
from pathlib import Path
import gzip,hashlib,json,os,platform,subprocess,time
ROOT=Path(__file__).resolve().parents[4];DOC=ROOT/'docs/foundational/phase1g';OUT=Path(os.environ.get('ALAN_QUALIFICATION_OUT',str(DOC/'evidence/task4/final')));OUT.mkdir(parents=True,exist_ok=True)
m=json.loads((DOC/'source-manifest.json').read_text())
def verify():
 for f in m['files']:
  raw=(ROOT/f['path']).read_bytes();assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256'],f['path']
verify();assert not (OUT/'checks.json').exists()
env=dict(os.environ);env.pop('OPENAI_API_KEY',None);env.update(GOTOOLCHAIN='local',GOFLAGS='-mod=readonly',PYTHONDONTWRITEBYTECODE='1');env.setdefault('GOPROXY','off');env.setdefault('GOSUMDB','off');env.setdefault('GOCACHE','/tmp/alan-phase1f-go-build')
cmds=[['go','vet','./...'],['go','vet','-tags=integration','./...'],['go','-C','examples/worktree/consumer','vet','./...'],['go','mod','verify'],['go','-C','examples/worktree/consumer','mod','verify'],['gofmt','-l']+[f['path'] for f in m['files'] if f['path'].endswith('.go')],['python3','-m','unittest','discover','-s','docs/foundational/phase1e/tools','-p','test_*.py','-v'],['python3','-m','unittest','discover','-s','docs/foundational/phase1f/tools','-p','test_*.py','-v']]
records=[]
for i,cmd in enumerate(cmds):
 start=time.monotonic();p=subprocess.run(cmd,cwd=ROOT,env=env,capture_output=True)
 stem=f'check-{i+1:02d}';(OUT/(stem+'.stdout.gz')).write_bytes(gzip.compress(p.stdout,mtime=0));(OUT/(stem+'.stderr.gz')).write_bytes(gzip.compress(p.stderr,mtime=0))
 verify();records.append(dict(Command=cmd,ExitCode=p.returncode,ElapsedSeconds=round(time.monotonic()-start,3),Stdout=stem+'.stdout.gz',Stderr=stem+'.stderr.gz',SourceUnchanged=True))
 (OUT/'checks.json').write_text(json.dumps(dict(SourceID=m['source_sha256'],Directory=str(ROOT),Platform=platform.platform(),Environment={k:env[k] for k in ('GOTOOLCHAIN','GOFLAGS','GOPROXY','GOSUMDB','GOCACHE')},Commands=records),indent=2)+'\n')
 print(stem,cmd[:5],p.returncode,flush=True);assert p.returncode==0
 if cmd[0]=='gofmt':assert not p.stdout
